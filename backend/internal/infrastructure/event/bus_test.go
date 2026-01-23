package event

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// testEvent implements DomainEvent for testing
type testEvent struct {
	shared.BaseDomainEvent
	Data string `json:"data"`
}

func newTestEvent(eventType string, tenantID uuid.UUID) *testEvent {
	return &testEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(eventType, "TestAggregate", uuid.New(), tenantID),
		Data:            "test data",
	}
}

// testHandler implements EventHandler for testing
type testHandler struct {
	eventTypes []string
	handled    []shared.DomainEvent
	err        error
	mu         sync.Mutex
}

func newTestHandler(eventTypes ...string) *testHandler {
	return &testHandler{
		eventTypes: eventTypes,
		handled:    make([]shared.DomainEvent, 0),
	}
}

func (h *testHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handled = append(h.handled, event)
	return h.err
}

func (h *testHandler) EventTypes() []string {
	return h.eventTypes
}

func (h *testHandler) setError(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.err = err
}

func (h *testHandler) getHandled() []shared.DomainEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]shared.DomainEvent(nil), h.handled...)
}

func TestInMemoryEventBus_Publish(t *testing.T) {
	logger := zap.NewNop()
	bus := NewInMemoryEventBus(logger)

	handler := newTestHandler("TestEvent")
	bus.Subscribe(handler, "TestEvent")

	event := newTestEvent("TestEvent", uuid.New())
	err := bus.Publish(context.Background(), event)

	require.NoError(t, err)
	assert.Len(t, handler.getHandled(), 1)
	assert.Equal(t, event, handler.getHandled()[0])
}

func TestInMemoryEventBus_Publish_MultipleEvents(t *testing.T) {
	logger := zap.NewNop()
	bus := NewInMemoryEventBus(logger)

	handler := newTestHandler("TestEvent")
	bus.Subscribe(handler, "TestEvent")

	event1 := newTestEvent("TestEvent", uuid.New())
	event2 := newTestEvent("TestEvent", uuid.New())
	err := bus.Publish(context.Background(), event1, event2)

	require.NoError(t, err)
	assert.Len(t, handler.getHandled(), 2)
}

func TestInMemoryEventBus_Publish_MultipleHandlers(t *testing.T) {
	logger := zap.NewNop()
	bus := NewInMemoryEventBus(logger)

	handler1 := newTestHandler("TestEvent")
	handler2 := newTestHandler("TestEvent")
	bus.Subscribe(handler1, "TestEvent")
	bus.Subscribe(handler2, "TestEvent")

	event := newTestEvent("TestEvent", uuid.New())
	err := bus.Publish(context.Background(), event)

	require.NoError(t, err)
	assert.Len(t, handler1.getHandled(), 1)
	assert.Len(t, handler2.getHandled(), 1)
}

func TestInMemoryEventBus_Publish_WildcardHandler(t *testing.T) {
	logger := zap.NewNop()
	bus := NewInMemoryEventBus(logger)

	wildcardHandler := newTestHandler() // No event types = wildcard
	bus.Subscribe(wildcardHandler)

	event := newTestEvent("AnyEventType", uuid.New())
	err := bus.Publish(context.Background(), event)

	require.NoError(t, err)
	assert.Len(t, wildcardHandler.getHandled(), 1)
}

func TestInMemoryEventBus_Publish_HandlerError(t *testing.T) {
	logger := zap.NewNop()
	bus := NewInMemoryEventBus(logger)

	handler1 := newTestHandler("TestEvent")
	handler1.setError(errors.New("handler error"))
	handler2 := newTestHandler("TestEvent")
	bus.Subscribe(handler1, "TestEvent")
	bus.Subscribe(handler2, "TestEvent")

	event := newTestEvent("TestEvent", uuid.New())
	err := bus.Publish(context.Background(), event)

	// Should not return error, but continue with other handlers
	require.NoError(t, err)
	assert.Len(t, handler1.getHandled(), 1)
	assert.Len(t, handler2.getHandled(), 1)
}

func TestInMemoryEventBus_Publish_NoMatchingHandlers(t *testing.T) {
	logger := zap.NewNop()
	bus := NewInMemoryEventBus(logger)

	handler := newTestHandler("OtherEvent")
	bus.Subscribe(handler, "OtherEvent")

	event := newTestEvent("TestEvent", uuid.New())
	err := bus.Publish(context.Background(), event)

	require.NoError(t, err)
	assert.Len(t, handler.getHandled(), 0)
}

func TestInMemoryEventBus_Unsubscribe(t *testing.T) {
	logger := zap.NewNop()
	bus := NewInMemoryEventBus(logger)

	handler := newTestHandler("TestEvent")
	bus.Subscribe(handler, "TestEvent")

	event1 := newTestEvent("TestEvent", uuid.New())
	_ = bus.Publish(context.Background(), event1)
	assert.Len(t, handler.getHandled(), 1)

	bus.Unsubscribe(handler)

	event2 := newTestEvent("TestEvent", uuid.New())
	_ = bus.Publish(context.Background(), event2)
	assert.Len(t, handler.getHandled(), 1) // Still 1, not 2
}

func TestInMemoryEventBus_StartStop(t *testing.T) {
	logger := zap.NewNop()
	bus := NewInMemoryEventBus(logger)

	ctx := context.Background()
	err := bus.Start(ctx)
	require.NoError(t, err)

	// Can still publish after start
	handler := newTestHandler("TestEvent")
	bus.Subscribe(handler, "TestEvent")
	event := newTestEvent("TestEvent", uuid.New())
	err = bus.Publish(ctx, event)
	require.NoError(t, err)
	assert.Len(t, handler.getHandled(), 1)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err = bus.Stop(ctx)
	require.NoError(t, err)
}
