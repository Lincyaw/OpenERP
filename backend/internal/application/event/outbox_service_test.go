package event

import (
	"context"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockOutboxRepoForService is a mock implementation for testing OutboxService
type mockOutboxRepoForService struct {
	entries map[uuid.UUID]*shared.OutboxEntry
}

func newMockOutboxRepoForService() *mockOutboxRepoForService {
	return &mockOutboxRepoForService{
		entries: make(map[uuid.UUID]*shared.OutboxEntry),
	}
}

func (r *mockOutboxRepoForService) Save(ctx context.Context, entries ...*shared.OutboxEntry) error {
	for _, e := range entries {
		r.entries[e.ID] = e
	}
	return nil
}

func (r *mockOutboxRepoForService) FindPending(ctx context.Context, limit int) ([]*shared.OutboxEntry, error) {
	var result []*shared.OutboxEntry
	for _, e := range r.entries {
		if e.Status == shared.OutboxStatusPending {
			result = append(result, e)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (r *mockOutboxRepoForService) FindRetryable(ctx context.Context, before time.Time, limit int) ([]*shared.OutboxEntry, error) {
	return nil, nil
}

func (r *mockOutboxRepoForService) FindDead(ctx context.Context, page, pageSize int) ([]*shared.OutboxEntry, int64, error) {
	var result []*shared.OutboxEntry
	for _, e := range r.entries {
		if e.Status == shared.OutboxStatusDead {
			result = append(result, e)
		}
	}
	total := int64(len(result))

	// Apply pagination
	start := (page - 1) * pageSize
	if start >= len(result) {
		return nil, total, nil
	}
	end := start + pageSize
	if end > len(result) {
		end = len(result)
	}
	return result[start:end], total, nil
}

func (r *mockOutboxRepoForService) FindByID(ctx context.Context, id uuid.UUID) (*shared.OutboxEntry, error) {
	if e, ok := r.entries[id]; ok {
		return e, nil
	}
	return nil, nil
}

func (r *mockOutboxRepoForService) MarkProcessing(ctx context.Context, ids []uuid.UUID) ([]*shared.OutboxEntry, error) {
	return nil, nil
}

func (r *mockOutboxRepoForService) Update(ctx context.Context, entry *shared.OutboxEntry) error {
	r.entries[entry.ID] = entry
	return nil
}

func (r *mockOutboxRepoForService) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}

func (r *mockOutboxRepoForService) CountByStatus(ctx context.Context) (map[shared.OutboxStatus]int64, error) {
	counts := make(map[shared.OutboxStatus]int64)
	for _, e := range r.entries {
		counts[e.Status]++
	}
	return counts, nil
}

func TestOutboxService_GetDeadLetterEntries(t *testing.T) {
	logger := zap.NewNop()
	repo := newMockOutboxRepoForService()
	service := NewOutboxService(repo, logger)

	// Create some dead letter entries
	for i := 0; i < 5; i++ {
		entry := &shared.OutboxEntry{
			ID:            uuid.New(),
			TenantID:      uuid.New(),
			EventID:       uuid.New(),
			EventType:     "TestEvent",
			AggregateID:   uuid.New(),
			AggregateType: "TestAggregate",
			Status:        shared.OutboxStatusDead,
			RetryCount:    5,
			MaxRetries:    5,
			LastError:     "test error",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		repo.entries[entry.ID] = entry
	}

	// Create some non-dead entries
	pendingEntry := &shared.OutboxEntry{
		ID:     uuid.New(),
		Status: shared.OutboxStatusPending,
	}
	repo.entries[pendingEntry.ID] = pendingEntry

	result, err := service.GetDeadLetterEntries(context.Background(), OutboxFilter{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(5), result.Total)
	assert.Len(t, result.Entries, 5)

	for _, entry := range result.Entries {
		assert.Equal(t, "DEAD", entry.Status)
	}
}

func TestOutboxService_RetryDeadEntry(t *testing.T) {
	logger := zap.NewNop()
	repo := newMockOutboxRepoForService()
	service := NewOutboxService(repo, logger)

	// Create a dead entry
	deadEntry := &shared.OutboxEntry{
		ID:            uuid.New(),
		TenantID:      uuid.New(),
		EventID:       uuid.New(),
		EventType:     "TestEvent",
		AggregateID:   uuid.New(),
		AggregateType: "TestAggregate",
		Status:        shared.OutboxStatusDead,
		RetryCount:    5,
		MaxRetries:    5,
		LastError:     "test error",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	repo.entries[deadEntry.ID] = deadEntry

	result, err := service.RetryDeadEntry(context.Background(), deadEntry.ID)
	require.NoError(t, err)
	assert.Equal(t, "PENDING", result.Status)
	assert.Equal(t, 0, result.RetryCount)
	assert.Empty(t, result.LastError)
}

func TestOutboxService_RetryDeadEntry_NotFound(t *testing.T) {
	logger := zap.NewNop()
	repo := newMockOutboxRepoForService()
	service := NewOutboxService(repo, logger)

	_, err := service.RetryDeadEntry(context.Background(), uuid.New())
	assert.Error(t, err)
}

func TestOutboxService_RetryDeadEntry_NotDead(t *testing.T) {
	logger := zap.NewNop()
	repo := newMockOutboxRepoForService()
	service := NewOutboxService(repo, logger)

	// Create a non-dead entry
	entry := &shared.OutboxEntry{
		ID:     uuid.New(),
		Status: shared.OutboxStatusPending,
	}
	repo.entries[entry.ID] = entry

	_, err := service.RetryDeadEntry(context.Background(), entry.ID)
	assert.Error(t, err)
}

func TestOutboxService_GetStats(t *testing.T) {
	logger := zap.NewNop()
	repo := newMockOutboxRepoForService()
	service := NewOutboxService(repo, logger)

	// Create entries with various statuses
	statuses := []shared.OutboxStatus{
		shared.OutboxStatusPending,
		shared.OutboxStatusPending,
		shared.OutboxStatusProcessing,
		shared.OutboxStatusSent,
		shared.OutboxStatusSent,
		shared.OutboxStatusSent,
		shared.OutboxStatusFailed,
		shared.OutboxStatusDead,
	}

	for _, status := range statuses {
		entry := &shared.OutboxEntry{
			ID:     uuid.New(),
			Status: status,
		}
		repo.entries[entry.ID] = entry
	}

	stats, err := service.GetStats(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.Pending)
	assert.Equal(t, int64(1), stats.Processing)
	assert.Equal(t, int64(3), stats.Sent)
	assert.Equal(t, int64(1), stats.Failed)
	assert.Equal(t, int64(1), stats.Dead)
	assert.Equal(t, int64(8), stats.Total)
}

func TestOutboxService_RetryAllDeadEntries(t *testing.T) {
	logger := zap.NewNop()
	repo := newMockOutboxRepoForService()
	service := NewOutboxService(repo, logger)

	// Create multiple dead entries
	for i := 0; i < 3; i++ {
		entry := &shared.OutboxEntry{
			ID:         uuid.New(),
			Status:     shared.OutboxStatusDead,
			RetryCount: 5,
			MaxRetries: 5,
			LastError:  "test error",
		}
		repo.entries[entry.ID] = entry
	}

	// Create a non-dead entry
	pendingEntry := &shared.OutboxEntry{
		ID:     uuid.New(),
		Status: shared.OutboxStatusPending,
	}
	repo.entries[pendingEntry.ID] = pendingEntry

	count, err := service.RetryAllDeadEntries(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)

	// Verify all dead entries are now pending
	for _, entry := range repo.entries {
		if entry.ID != pendingEntry.ID {
			assert.Equal(t, shared.OutboxStatusPending, entry.Status)
			assert.Equal(t, 0, entry.RetryCount)
		}
	}
}
