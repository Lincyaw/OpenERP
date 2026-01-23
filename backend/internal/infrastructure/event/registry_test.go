package event

import (
	"context"
	"testing"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/stretchr/testify/assert"
)

// mockHandler implements EventHandler for testing
type mockHandler struct {
	eventTypes []string
	handled    []shared.DomainEvent
}

func newMockHandler(eventTypes ...string) *mockHandler {
	return &mockHandler{
		eventTypes: eventTypes,
		handled:    make([]shared.DomainEvent, 0),
	}
}

func (h *mockHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	h.handled = append(h.handled, event)
	return nil
}

func (h *mockHandler) EventTypes() []string {
	return h.eventTypes
}

func TestHandlerRegistry_Register_SpecificTypes(t *testing.T) {
	registry := NewHandlerRegistry()
	handler := newMockHandler("OrderCreated", "OrderUpdated")

	registry.Register(handler, "OrderCreated", "OrderUpdated")

	handlers := registry.GetHandlers("OrderCreated")
	assert.Len(t, handlers, 1)
	assert.Equal(t, handler, handlers[0])

	handlers = registry.GetHandlers("OrderUpdated")
	assert.Len(t, handlers, 1)
	assert.Equal(t, handler, handlers[0])

	handlers = registry.GetHandlers("OrderDeleted")
	assert.Len(t, handlers, 0)
}

func TestHandlerRegistry_Register_Wildcard(t *testing.T) {
	registry := NewHandlerRegistry()
	handler := newMockHandler() // No event types = wildcard

	registry.Register(handler)

	handlers := registry.GetHandlers("OrderCreated")
	assert.Len(t, handlers, 1)
	assert.Equal(t, handler, handlers[0])

	handlers = registry.GetHandlers("AnyEventType")
	assert.Len(t, handlers, 1)
	assert.Equal(t, handler, handlers[0])
}

func TestHandlerRegistry_Register_MixedTypes(t *testing.T) {
	registry := NewHandlerRegistry()
	specificHandler := newMockHandler("OrderCreated")
	wildcardHandler := newMockHandler()

	registry.Register(specificHandler, "OrderCreated")
	registry.Register(wildcardHandler)

	handlers := registry.GetHandlers("OrderCreated")
	assert.Len(t, handlers, 2)

	handlers = registry.GetHandlers("OtherEvent")
	assert.Len(t, handlers, 1)
	assert.Equal(t, wildcardHandler, handlers[0])
}

func TestHandlerRegistry_Unregister_SpecificHandler(t *testing.T) {
	registry := NewHandlerRegistry()
	handler1 := newMockHandler("OrderCreated")
	handler2 := newMockHandler("OrderCreated")

	registry.Register(handler1, "OrderCreated")
	registry.Register(handler2, "OrderCreated")

	handlers := registry.GetHandlers("OrderCreated")
	assert.Len(t, handlers, 2)

	registry.Unregister(handler1)

	handlers = registry.GetHandlers("OrderCreated")
	assert.Len(t, handlers, 1)
	assert.Equal(t, handler2, handlers[0])
}

func TestHandlerRegistry_Unregister_WildcardHandler(t *testing.T) {
	registry := NewHandlerRegistry()
	wildcardHandler := newMockHandler()

	registry.Register(wildcardHandler)

	handlers := registry.GetHandlers("AnyEvent")
	assert.Len(t, handlers, 1)

	registry.Unregister(wildcardHandler)

	handlers = registry.GetHandlers("AnyEvent")
	assert.Len(t, handlers, 0)
}

func TestHandlerRegistry_GetAllHandlers(t *testing.T) {
	registry := NewHandlerRegistry()
	handler1 := newMockHandler("OrderCreated")
	handler2 := newMockHandler("UserCreated")
	wildcardHandler := newMockHandler()

	registry.Register(handler1, "OrderCreated")
	registry.Register(handler2, "UserCreated")
	registry.Register(wildcardHandler)

	allHandlers := registry.GetAllHandlers()
	assert.Len(t, allHandlers, 3)
}

func TestHandlerRegistry_GetAllHandlers_NoDuplicates(t *testing.T) {
	registry := NewHandlerRegistry()
	handler := newMockHandler("OrderCreated", "OrderUpdated")

	// Register same handler for multiple event types
	registry.Register(handler, "OrderCreated", "OrderUpdated")

	allHandlers := registry.GetAllHandlers()
	assert.Len(t, allHandlers, 1)
}
