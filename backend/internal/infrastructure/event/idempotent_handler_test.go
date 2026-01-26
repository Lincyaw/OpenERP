package event

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/cache"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockEventHandler is a mock implementation of shared.EventHandler
type MockEventHandler struct {
	mock.Mock
}

func (m *MockEventHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockEventHandler) EventTypes() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

// MockIdempotencyStore is a mock implementation of shared.IdempotencyStore
type MockIdempotencyStore struct {
	mock.Mock
}

func (m *MockIdempotencyStore) MarkProcessed(ctx context.Context, eventID string, ttl time.Duration) (bool, error) {
	args := m.Called(ctx, eventID, ttl)
	return args.Bool(0), args.Error(1)
}

func (m *MockIdempotencyStore) IsProcessed(ctx context.Context, eventID string) (bool, error) {
	args := m.Called(ctx, eventID)
	return args.Bool(0), args.Error(1)
}

func (m *MockIdempotencyStore) Close() error {
	args := m.Called()
	return args.Error(0)
}

// idempotencyTestEvent is a simple test event for idempotency tests
type idempotencyTestEvent struct {
	shared.BaseDomainEvent
	Data string
}

func newIdempotencyTestEvent() *idempotencyTestEvent {
	return &idempotencyTestEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			"test.event",
			"TestAggregate",
			uuid.New(),
			uuid.New(),
		),
		Data: "test data",
	}
}

func TestIdempotentHandler_Handle_NewEvent(t *testing.T) {
	logger := zap.NewNop()
	store := cache.NewInMemoryIdempotencyStore()
	defer store.Close()

	mockHandler := new(MockEventHandler)
	event := newIdempotencyTestEvent()

	mockHandler.On("Handle", mock.Anything, event).Return(nil)

	handler := NewIdempotentHandler(mockHandler, store, logger)

	err := handler.Handle(context.Background(), event)
	require.NoError(t, err)

	mockHandler.AssertExpectations(t)
	assert.Equal(t, int64(1), handler.metrics.EventsProcessed.Load())
	assert.Equal(t, int64(0), handler.metrics.EventsDuplicate.Load())
}

func TestIdempotentHandler_Handle_DuplicateEvent(t *testing.T) {
	logger := zap.NewNop()
	store := cache.NewInMemoryIdempotencyStore()
	defer store.Close()

	mockHandler := new(MockEventHandler)
	event := newIdempotencyTestEvent()

	// Handler should only be called once
	mockHandler.On("Handle", mock.Anything, event).Return(nil).Once()

	handler := NewIdempotentHandler(mockHandler, store, logger)

	// First call
	err := handler.Handle(context.Background(), event)
	require.NoError(t, err)

	// Second call with same event (duplicate)
	err = handler.Handle(context.Background(), event)
	require.NoError(t, err)

	// Third call (still duplicate)
	err = handler.Handle(context.Background(), event)
	require.NoError(t, err)

	mockHandler.AssertExpectations(t)
	assert.Equal(t, int64(1), handler.metrics.EventsProcessed.Load())
	assert.Equal(t, int64(2), handler.metrics.EventsDuplicate.Load())
}

func TestIdempotentHandler_Handle_HandlerError(t *testing.T) {
	logger := zap.NewNop()
	store := cache.NewInMemoryIdempotencyStore()
	defer store.Close()

	mockHandler := new(MockEventHandler)
	event := newIdempotencyTestEvent()
	expectedErr := errors.New("handler error")

	mockHandler.On("Handle", mock.Anything, event).Return(expectedErr)

	handler := NewIdempotentHandler(mockHandler, store, logger)

	err := handler.Handle(context.Background(), event)
	require.Error(t, err)
	assert.Equal(t, expectedErr, err)

	assert.Equal(t, int64(0), handler.metrics.EventsProcessed.Load())
	assert.Equal(t, int64(1), handler.metrics.EventsFailed.Load())
}

func TestIdempotentHandler_Handle_StoreError(t *testing.T) {
	logger := zap.NewNop()

	mockStore := new(MockIdempotencyStore)
	mockHandler := new(MockEventHandler)
	event := newIdempotencyTestEvent()

	// Store returns error
	mockStore.On("MarkProcessed", mock.Anything, event.EventID().String(), mock.Anything).
		Return(false, errors.New("store error"))

	// Handler should still be called even if store fails
	mockHandler.On("Handle", mock.Anything, event).Return(nil)

	handler := NewIdempotentHandler(mockHandler, mockStore, logger)

	err := handler.Handle(context.Background(), event)
	require.NoError(t, err)

	mockStore.AssertExpectations(t)
	mockHandler.AssertExpectations(t)
}

func TestIdempotentHandler_Handle_Disabled(t *testing.T) {
	logger := zap.NewNop()
	store := cache.NewInMemoryIdempotencyStore()
	defer store.Close()

	mockHandler := new(MockEventHandler)
	event := newIdempotencyTestEvent()

	// Handler should be called every time when idempotency is disabled
	mockHandler.On("Handle", mock.Anything, event).Return(nil).Times(3)

	config := shared.DefaultIdempotencyConfig()
	config.Enabled = false

	handler := NewIdempotentHandler(mockHandler, store, logger,
		WithIdempotencyConfig(config),
	)

	// All calls should go through
	for i := 0; i < 3; i++ {
		err := handler.Handle(context.Background(), event)
		require.NoError(t, err)
	}

	mockHandler.AssertExpectations(t)
	// Metrics shouldn't be updated when disabled
	assert.Equal(t, int64(0), handler.metrics.EventsProcessed.Load())
	assert.Equal(t, int64(0), handler.metrics.EventsDuplicate.Load())
}

func TestIdempotentHandler_EventTypes(t *testing.T) {
	logger := zap.NewNop()
	store := cache.NewInMemoryIdempotencyStore()
	defer store.Close()

	mockHandler := new(MockEventHandler)
	expectedTypes := []string{"event.type1", "event.type2"}

	mockHandler.On("EventTypes").Return(expectedTypes)

	handler := NewIdempotentHandler(mockHandler, store, logger)

	types := handler.EventTypes()
	assert.Equal(t, expectedTypes, types)

	mockHandler.AssertExpectations(t)
}

func TestIdempotentHandler_CustomConfig(t *testing.T) {
	logger := zap.NewNop()
	store := cache.NewInMemoryIdempotencyStore()
	defer store.Close()

	mockHandler := new(MockEventHandler)
	event := newIdempotencyTestEvent()

	mockHandler.On("Handle", mock.Anything, event).Return(nil).Once()

	customConfig := shared.IdempotencyConfig{
		TTL:     1 * time.Hour,
		Enabled: true,
	}

	handler := NewIdempotentHandler(mockHandler, store, logger,
		WithIdempotencyConfig(customConfig),
	)

	err := handler.Handle(context.Background(), event)
	require.NoError(t, err)

	mockHandler.AssertExpectations(t)
}

func TestIdempotentHandler_GetWrappedHandler(t *testing.T) {
	logger := zap.NewNop()
	store := cache.NewInMemoryIdempotencyStore()
	defer store.Close()

	mockHandler := new(MockEventHandler)

	handler := NewIdempotentHandler(mockHandler, store, logger)

	wrapped := handler.GetWrappedHandler()
	assert.Equal(t, mockHandler, wrapped)
}

func TestIdempotentHandler_SharedMetrics(t *testing.T) {
	logger := zap.NewNop()
	store := cache.NewInMemoryIdempotencyStore()
	defer store.Close()

	sharedMetrics := &IdempotencyMetrics{}

	mockHandler1 := new(MockEventHandler)
	mockHandler2 := new(MockEventHandler)

	event1 := newIdempotencyTestEvent()
	event2 := newIdempotencyTestEvent()

	mockHandler1.On("Handle", mock.Anything, event1).Return(nil)
	mockHandler2.On("Handle", mock.Anything, event2).Return(nil)

	handler1 := NewIdempotentHandler(mockHandler1, store, logger,
		WithIdempotencyMetrics(sharedMetrics),
	)
	handler2 := NewIdempotentHandler(mockHandler2, store, logger,
		WithIdempotencyMetrics(sharedMetrics),
	)

	handler1.Handle(context.Background(), event1)
	handler2.Handle(context.Background(), event2)

	// Both handlers should contribute to shared metrics
	assert.Equal(t, int64(2), sharedMetrics.EventsProcessed.Load())

	mockHandler1.AssertExpectations(t)
	mockHandler2.AssertExpectations(t)
}

func TestWrapHandlersWithIdempotency(t *testing.T) {
	logger := zap.NewNop()
	store := cache.NewInMemoryIdempotencyStore()
	defer store.Close()

	mockHandler1 := new(MockEventHandler)
	mockHandler2 := new(MockEventHandler)

	handlers := []shared.EventHandler{mockHandler1, mockHandler2}

	wrapped := WrapHandlersWithIdempotency(handlers, store, logger)

	assert.Len(t, wrapped, 2)

	// Verify each is an IdempotentHandler
	for i, h := range wrapped {
		idempotentHandler, ok := h.(*IdempotentHandler)
		assert.True(t, ok, "handler %d should be IdempotentHandler", i)
		assert.NotNil(t, idempotentHandler)
	}
}

func TestIdempotencyMetrics_Stats(t *testing.T) {
	metrics := &IdempotencyMetrics{}

	metrics.EventsProcessed.Add(10)
	metrics.EventsDuplicate.Add(5)
	metrics.EventsFailed.Add(2)

	stats := metrics.Stats()

	assert.Equal(t, int64(10), stats.EventsProcessed)
	assert.Equal(t, int64(5), stats.EventsDuplicate)
	assert.Equal(t, int64(2), stats.EventsFailed)
}

func TestIdempotentHandler_ConcurrentDuplicates(t *testing.T) {
	logger := zap.NewNop()
	store := cache.NewInMemoryIdempotencyStore()
	defer store.Close()

	mockHandler := new(MockEventHandler)
	event := newIdempotencyTestEvent()

	// Handler should only be called once even with concurrent requests
	mockHandler.On("Handle", mock.Anything, event).Return(nil).Once()

	handler := NewIdempotentHandler(mockHandler, store, logger)

	const numGoroutines = 50
	errChan := make(chan error, numGoroutines)

	// Launch concurrent goroutines
	for i := 0; i < numGoroutines; i++ {
		go func() {
			errChan <- handler.Handle(context.Background(), event)
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		err := <-errChan
		assert.NoError(t, err)
	}

	mockHandler.AssertExpectations(t)
	assert.Equal(t, int64(1), handler.metrics.EventsProcessed.Load())
	assert.Equal(t, int64(numGoroutines-1), handler.metrics.EventsDuplicate.Load())
}
