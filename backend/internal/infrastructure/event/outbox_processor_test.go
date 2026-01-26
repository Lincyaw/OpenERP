package event

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockOutboxRepository is a mock implementation for testing
type mockOutboxRepository struct {
	mu               sync.Mutex
	entries          map[uuid.UUID]*shared.OutboxEntry
	findPendingFn    func(ctx context.Context, limit int) ([]*shared.OutboxEntry, error)
	findRetryableFn  func(ctx context.Context, before time.Time, limit int) ([]*shared.OutboxEntry, error)
	markProcessingFn func(ctx context.Context, ids []uuid.UUID) ([]*shared.OutboxEntry, error)
	updateFn         func(ctx context.Context, entry *shared.OutboxEntry) error
	deleteFn         func(ctx context.Context, before time.Time) (int64, error)
}

func newMockOutboxRepository() *mockOutboxRepository {
	return &mockOutboxRepository{
		entries: make(map[uuid.UUID]*shared.OutboxEntry),
	}
}

func (r *mockOutboxRepository) Save(ctx context.Context, entries ...*shared.OutboxEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range entries {
		r.entries[e.ID] = e
	}
	return nil
}

func (r *mockOutboxRepository) FindPending(ctx context.Context, limit int) ([]*shared.OutboxEntry, error) {
	if r.findPendingFn != nil {
		return r.findPendingFn(ctx, limit)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
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

func (r *mockOutboxRepository) FindRetryable(ctx context.Context, before time.Time, limit int) ([]*shared.OutboxEntry, error) {
	if r.findRetryableFn != nil {
		return r.findRetryableFn(ctx, before, limit)
	}
	return nil, nil
}

func (r *mockOutboxRepository) MarkProcessing(ctx context.Context, ids []uuid.UUID) ([]*shared.OutboxEntry, error) {
	if r.markProcessingFn != nil {
		return r.markProcessingFn(ctx, ids)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*shared.OutboxEntry
	for _, id := range ids {
		if e, ok := r.entries[id]; ok {
			e.Status = shared.OutboxStatusProcessing
			result = append(result, e)
		}
	}
	return result, nil
}

func (r *mockOutboxRepository) Update(ctx context.Context, entry *shared.OutboxEntry) error {
	if r.updateFn != nil {
		return r.updateFn(ctx, entry)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[entry.ID] = entry
	return nil
}

func (r *mockOutboxRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	if r.deleteFn != nil {
		return r.deleteFn(ctx, before)
	}
	return 0, nil
}

func (r *mockOutboxRepository) FindDead(ctx context.Context, page, pageSize int) ([]*shared.OutboxEntry, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*shared.OutboxEntry
	for _, e := range r.entries {
		if e.Status == shared.OutboxStatusDead {
			result = append(result, e)
		}
	}
	return result, int64(len(result)), nil
}

func (r *mockOutboxRepository) FindByID(ctx context.Context, id uuid.UUID) (*shared.OutboxEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e, ok := r.entries[id]; ok {
		return e, nil
	}
	return nil, nil
}

func (r *mockOutboxRepository) CountByStatus(ctx context.Context) (map[shared.OutboxStatus]int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	counts := make(map[shared.OutboxStatus]int64)
	for _, e := range r.entries {
		counts[e.Status]++
	}
	return counts, nil
}

func TestOutboxProcessor_ProcessesPendingEntries(t *testing.T) {
	logger := zap.NewNop()
	serializer := NewEventSerializer()
	serializer.Register("TestEvent", &testEvent{})

	repo := newMockOutboxRepository()
	eventBus := NewInMemoryEventBus(logger)

	handler := newTestHandler("TestEvent")
	eventBus.Subscribe(handler, "TestEvent")

	// Create pending entry
	tenantID := uuid.New()
	event := newTestEvent("TestEvent", tenantID)
	payload, _ := serializer.Serialize(event)
	entry := shared.NewOutboxEntry(tenantID, event, payload)
	repo.Save(context.Background(), entry)

	config := OutboxProcessorConfig{
		BatchSize:    100,
		PollInterval: 50 * time.Millisecond,
	}
	processor := NewOutboxProcessor(repo, eventBus, serializer, config, logger)

	ctx, cancel := context.WithCancel(context.Background())
	err := processor.Start(ctx)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	cancel()
	stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Second)
	defer stopCancel()
	err = processor.Stop(stopCtx)
	require.NoError(t, err)

	// Verify event was processed
	assert.Len(t, handler.getHandled(), 1)

	// Verify entry was marked as sent
	repo.mu.Lock()
	defer repo.mu.Unlock()
	assert.Equal(t, shared.OutboxStatusSent, repo.entries[entry.ID].Status)
}

func TestOutboxProcessor_StopGracefully(t *testing.T) {
	logger := zap.NewNop()
	serializer := NewEventSerializer()
	repo := newMockOutboxRepository()
	eventBus := NewInMemoryEventBus(logger)

	config := DefaultOutboxProcessorConfig()
	processor := NewOutboxProcessor(repo, eventBus, serializer, config, logger)

	ctx := context.Background()
	err := processor.Start(ctx)
	require.NoError(t, err)

	stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err = processor.Stop(stopCtx)
	require.NoError(t, err)
}

func TestOutboxProcessor_HandleDeserializationError(t *testing.T) {
	logger := zap.NewNop()
	serializer := NewEventSerializer()
	// Note: NOT registering the event type to cause deserialization error

	repo := newMockOutboxRepository()
	eventBus := NewInMemoryEventBus(logger)

	// Create entry with unregistered event type
	tenantID := uuid.New()
	event := newTestEvent("UnregisteredEvent", tenantID)
	payload := []byte(`{"type": "UnregisteredEvent"}`)
	entry := shared.NewOutboxEntry(tenantID, event, payload)
	entry.EventType = "UnregisteredEvent"
	repo.Save(context.Background(), entry)

	config := OutboxProcessorConfig{
		BatchSize:    100,
		PollInterval: 50 * time.Millisecond,
	}
	processor := NewOutboxProcessor(repo, eventBus, serializer, config, logger)

	ctx, cancel := context.WithCancel(context.Background())
	err := processor.Start(ctx)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	cancel()
	stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Second)
	defer stopCancel()
	processor.Stop(stopCtx)

	// Verify entry was marked as failed
	repo.mu.Lock()
	defer repo.mu.Unlock()
	assert.Equal(t, shared.OutboxStatusFailed, repo.entries[entry.ID].Status)
	assert.Contains(t, repo.entries[entry.ID].LastError, "unknown event type")
}

func TestDefaultOutboxProcessorConfig(t *testing.T) {
	config := DefaultOutboxProcessorConfig()

	assert.Equal(t, 100, config.BatchSize)
	assert.Equal(t, 5*time.Second, config.PollInterval)
	assert.True(t, config.CleanupEnabled)
	assert.Equal(t, 7*24*time.Hour, config.CleanupRetention)
	assert.Equal(t, 1*time.Hour, config.CleanupInterval)
}
