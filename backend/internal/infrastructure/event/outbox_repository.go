package event

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GormOutboxRepository implements OutboxRepository using GORM
type GormOutboxRepository struct {
	db *gorm.DB
}

// NewGormOutboxRepository creates a new GORM-based outbox repository
func NewGormOutboxRepository(db *gorm.DB) *GormOutboxRepository {
	return &GormOutboxRepository{db: db}
}

// WithTx returns a new repository instance with the given transaction
func (r *GormOutboxRepository) WithTx(tx *gorm.DB) *GormOutboxRepository {
	return &GormOutboxRepository{db: tx}
}

// Save persists one or more outbox entries
func (r *GormOutboxRepository) Save(ctx context.Context, entries ...*shared.OutboxEntry) error {
	if len(entries) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Create(entries).Error
}

// FindPending retrieves pending entries up to the specified limit
func (r *GormOutboxRepository) FindPending(ctx context.Context, limit int) ([]*shared.OutboxEntry, error) {
	var entries []*shared.OutboxEntry
	err := r.db.WithContext(ctx).
		Where("status = ?", shared.OutboxStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&entries).Error
	return entries, err
}

// FindRetryable retrieves failed entries that are due for retry
func (r *GormOutboxRepository) FindRetryable(ctx context.Context, before time.Time, limit int) ([]*shared.OutboxEntry, error) {
	var entries []*shared.OutboxEntry
	err := r.db.WithContext(ctx).
		Where("status = ? AND next_retry_at <= ?", shared.OutboxStatusFailed, before).
		Order("next_retry_at ASC").
		Limit(limit).
		Find(&entries).Error
	return entries, err
}

// MarkProcessing atomically marks entries as processing and returns them
func (r *GormOutboxRepository) MarkProcessing(ctx context.Context, ids []uuid.UUID) ([]*shared.OutboxEntry, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var entries []*shared.OutboxEntry

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock and fetch entries using FOR UPDATE SKIP LOCKED
		if err := tx.
			Clauses(clause.Locking{
				Strength: "UPDATE",
				Options:  "SKIP LOCKED",
			}).
			Where("id IN ? AND status IN ?", ids, []shared.OutboxStatus{
				shared.OutboxStatusPending,
				shared.OutboxStatusFailed,
			}).
			Find(&entries).Error; err != nil {
			return err
		}

		if len(entries) == 0 {
			return nil
		}

		// Update status to processing
		entryIDs := make([]uuid.UUID, len(entries))
		for i, e := range entries {
			entryIDs[i] = e.ID
		}

		now := time.Now()
		if err := tx.Model(&shared.OutboxEntry{}).
			Where("id IN ?", entryIDs).
			Updates(map[string]interface{}{
				"status":     shared.OutboxStatusProcessing,
				"updated_at": now,
			}).Error; err != nil {
			return err
		}

		// Update in-memory entries
		for _, e := range entries {
			e.Status = shared.OutboxStatusProcessing
			e.UpdatedAt = now
		}

		return nil
	})

	return entries, err
}

// Update updates an existing outbox entry
func (r *GormOutboxRepository) Update(ctx context.Context, entry *shared.OutboxEntry) error {
	entry.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(entry).Error
}

// DeleteOlderThan deletes entries older than the specified time
func (r *GormOutboxRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("status = ? AND processed_at < ?", shared.OutboxStatusSent, before).
		Delete(&shared.OutboxEntry{})
	return result.RowsAffected, result.Error
}

// Ensure GormOutboxRepository implements OutboxRepository
var _ shared.OutboxRepository = (*GormOutboxRepository)(nil)
