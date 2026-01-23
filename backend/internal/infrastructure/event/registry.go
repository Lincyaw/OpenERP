package event

import (
	"sync"

	"github.com/erp/backend/internal/domain/shared"
)

// HandlerRegistry manages event handler registrations
type HandlerRegistry struct {
	mu       sync.RWMutex
	handlers map[string][]shared.EventHandler // eventType -> handlers
	wildcard []shared.EventHandler            // handlers for all events
}

// NewHandlerRegistry creates a new handler registry
func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make(map[string][]shared.EventHandler),
		wildcard: make([]shared.EventHandler, 0),
	}
}

// Register adds a handler for specific event types
// If no event types are provided, the handler receives all events
func (r *HandlerRegistry) Register(handler shared.EventHandler, eventTypes ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(eventTypes) == 0 {
		// Wildcard handler - receives all events
		r.wildcard = append(r.wildcard, handler)
		return
	}

	for _, eventType := range eventTypes {
		r.handlers[eventType] = append(r.handlers[eventType], handler)
	}
}

// Unregister removes a handler from all event types
func (r *HandlerRegistry) Unregister(handler shared.EventHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Remove from wildcard handlers
	r.wildcard = removeHandler(r.wildcard, handler)

	// Remove from specific event type handlers
	for eventType, handlers := range r.handlers {
		r.handlers[eventType] = removeHandler(handlers, handler)
		if len(r.handlers[eventType]) == 0 {
			delete(r.handlers, eventType)
		}
	}
}

// GetHandlers returns all handlers for a specific event type
// This includes both type-specific handlers and wildcard handlers
func (r *HandlerRegistry) GetHandlers(eventType string) []shared.EventHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Combine type-specific handlers with wildcard handlers
	typeHandlers := r.handlers[eventType]
	result := make([]shared.EventHandler, 0, len(typeHandlers)+len(r.wildcard))
	result = append(result, typeHandlers...)
	result = append(result, r.wildcard...)

	return result
}

// GetAllHandlers returns all registered handlers
func (r *HandlerRegistry) GetAllHandlers() []shared.EventHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := make(map[shared.EventHandler]bool)
	result := make([]shared.EventHandler, 0)

	for _, handler := range r.wildcard {
		if !seen[handler] {
			seen[handler] = true
			result = append(result, handler)
		}
	}

	for _, handlers := range r.handlers {
		for _, handler := range handlers {
			if !seen[handler] {
				seen[handler] = true
				result = append(result, handler)
			}
		}
	}

	return result
}

// removeHandler removes a handler from a slice of handlers
func removeHandler(handlers []shared.EventHandler, target shared.EventHandler) []shared.EventHandler {
	result := make([]shared.EventHandler, 0, len(handlers))
	for _, h := range handlers {
		if h != target {
			result = append(result, h)
		}
	}
	return result
}
