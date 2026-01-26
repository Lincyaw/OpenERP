package event

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// OutboxService handles outbox event management operations
type OutboxService struct {
	repo   shared.OutboxRepository
	logger *zap.Logger
}

// NewOutboxService creates a new outbox service
func NewOutboxService(
	repo shared.OutboxRepository,
	logger *zap.Logger,
) *OutboxService {
	return &OutboxService{
		repo:   repo,
		logger: logger,
	}
}

// OutboxEntryDTO represents an outbox entry data transfer object
type OutboxEntryDTO struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	EventID       uuid.UUID  `json:"event_id"`
	EventType     string     `json:"event_type"`
	AggregateID   uuid.UUID  `json:"aggregate_id"`
	AggregateType string     `json:"aggregate_type"`
	Status        string     `json:"status"`
	RetryCount    int        `json:"retry_count"`
	MaxRetries    int        `json:"max_retries"`
	LastError     string     `json:"last_error,omitempty"`
	NextRetryAt   *time.Time `json:"next_retry_at,omitempty"`
	ProcessedAt   *time.Time `json:"processed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// OutboxFilter represents filter for querying outbox entries
type OutboxFilter struct {
	Page     int `form:"page,omitempty" binding:"omitempty,min=1"`
	PageSize int `form:"page_size,omitempty" binding:"omitempty,min=1,max=100"`
}

// OutboxListResult represents paginated outbox entry list result
type OutboxListResult struct {
	Entries    []OutboxEntryDTO `json:"entries"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

// OutboxStatsDTO represents outbox statistics
type OutboxStatsDTO struct {
	Pending    int64 `json:"pending"`
	Processing int64 `json:"processing"`
	Sent       int64 `json:"sent"`
	Failed     int64 `json:"failed"`
	Dead       int64 `json:"dead"`
	Total      int64 `json:"total"`
}

// GetDeadLetterEntries retrieves dead letter entries with pagination
func (s *OutboxService) GetDeadLetterEntries(ctx context.Context, filter OutboxFilter) (*OutboxListResult, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	entries, total, err := s.repo.FindDead(ctx, page, pageSize)
	if err != nil {
		s.logger.Error("Failed to find dead letter entries", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to retrieve dead letter entries")
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	entryDTOs := make([]OutboxEntryDTO, len(entries))
	for i, entry := range entries {
		entryDTOs[i] = toOutboxEntryDTO(entry)
	}

	return &OutboxListResult{
		Entries:    entryDTOs,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// GetEntry retrieves a single outbox entry by ID
func (s *OutboxService) GetEntry(ctx context.Context, id uuid.UUID) (*OutboxEntryDTO, error) {
	entry, err := s.repo.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to find outbox entry", zap.Error(err), zap.String("id", id.String()))
		return nil, shared.NewDomainError("ENTRY_NOT_FOUND", "Outbox entry not found")
	}
	if entry == nil {
		return nil, shared.NewDomainError("ENTRY_NOT_FOUND", "Outbox entry not found")
	}

	dto := toOutboxEntryDTO(entry)
	return &dto, nil
}

// RetryDeadEntry resets a dead letter entry for retry
func (s *OutboxService) RetryDeadEntry(ctx context.Context, id uuid.UUID) (*OutboxEntryDTO, error) {
	entry, err := s.repo.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to find outbox entry", zap.Error(err), zap.String("id", id.String()))
		return nil, shared.NewDomainError("ENTRY_NOT_FOUND", "Outbox entry not found")
	}
	if entry == nil {
		return nil, shared.NewDomainError("ENTRY_NOT_FOUND", "Outbox entry not found")
	}

	if err := entry.ResetForRetry(); err != nil {
		return nil, shared.NewDomainError("INVALID_STATUS", err.Error())
	}

	if err := s.repo.Update(ctx, entry); err != nil {
		s.logger.Error("Failed to update outbox entry", zap.Error(err), zap.String("id", id.String()))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to retry entry")
	}

	s.logger.Info("Dead letter entry reset for retry",
		zap.String("id", id.String()),
		zap.String("event_type", entry.EventType),
	)

	dto := toOutboxEntryDTO(entry)
	return &dto, nil
}

// RetryAllDeadEntries resets all dead letter entries for retry
func (s *OutboxService) RetryAllDeadEntries(ctx context.Context) (int64, error) {
	var count int64
	page := 1
	pageSize := 100

	for {
		entries, _, err := s.repo.FindDead(ctx, page, pageSize)
		if err != nil {
			s.logger.Error("Failed to find dead letter entries", zap.Error(err))
			return count, shared.NewDomainError("INTERNAL_ERROR", "Failed to retrieve dead letter entries")
		}

		if len(entries) == 0 {
			break
		}

		for _, entry := range entries {
			if err := entry.ResetForRetry(); err != nil {
				continue
			}
			if err := s.repo.Update(ctx, entry); err != nil {
				s.logger.Error("Failed to update outbox entry", zap.Error(err), zap.String("id", entry.ID.String()))
				continue
			}
			count++
		}

		if len(entries) < pageSize {
			break
		}
		page++
	}

	s.logger.Info("Retried dead letter entries", zap.Int64("count", count))

	return count, nil
}

// GetStats returns outbox statistics
func (s *OutboxService) GetStats(ctx context.Context) (*OutboxStatsDTO, error) {
	counts, err := s.repo.CountByStatus(ctx)
	if err != nil {
		s.logger.Error("Failed to get outbox stats", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to get outbox stats")
	}

	var total int64
	for _, count := range counts {
		total += count
	}

	return &OutboxStatsDTO{
		Pending:    counts[shared.OutboxStatusPending],
		Processing: counts[shared.OutboxStatusProcessing],
		Sent:       counts[shared.OutboxStatusSent],
		Failed:     counts[shared.OutboxStatusFailed],
		Dead:       counts[shared.OutboxStatusDead],
		Total:      total,
	}, nil
}

// toOutboxEntryDTO converts domain OutboxEntry to OutboxEntryDTO
func toOutboxEntryDTO(entry *shared.OutboxEntry) OutboxEntryDTO {
	return OutboxEntryDTO{
		ID:            entry.ID,
		TenantID:      entry.TenantID,
		EventID:       entry.EventID,
		EventType:     entry.EventType,
		AggregateID:   entry.AggregateID,
		AggregateType: entry.AggregateType,
		Status:        string(entry.Status),
		RetryCount:    entry.RetryCount,
		MaxRetries:    entry.MaxRetries,
		LastError:     entry.LastError,
		NextRetryAt:   entry.NextRetryAt,
		ProcessedAt:   entry.ProcessedAt,
		CreatedAt:     entry.CreatedAt,
		UpdatedAt:     entry.UpdatedAt,
	}
}
