// Package testutil provides common test utilities for the ERP backend.
package testutil

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/erp/backend/internal/domain/shared"
)

// MockEventHandler is a mock implementation of shared.EventHandler for testing.
type MockEventHandler struct {
	mu         sync.Mutex
	eventTypes []string
	handled    []shared.DomainEvent
	err        error
}

// NewMockEventHandler creates a new mock event handler.
func NewMockEventHandler(eventTypes ...string) *MockEventHandler {
	return &MockEventHandler{
		eventTypes: eventTypes,
		handled:    make([]shared.DomainEvent, 0),
	}
}

// EventTypes returns the event types this handler subscribes to.
func (h *MockEventHandler) EventTypes() []string {
	return h.eventTypes
}

// Handle processes an event.
func (h *MockEventHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handled = append(h.handled, event)
	return h.err
}

// Handled returns all handled events.
func (h *MockEventHandler) Handled() []shared.DomainEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	result := make([]shared.DomainEvent, len(h.handled))
	copy(result, h.handled)
	return result
}

// HandledCount returns the number of handled events.
func (h *MockEventHandler) HandledCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.handled)
}

// SetError sets the error to return from Handle.
func (h *MockEventHandler) SetError(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.err = err
}

// Reset clears all handled events.
func (h *MockEventHandler) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handled = make([]shared.DomainEvent, 0)
	h.err = nil
}

// TestEvent is a simple domain event for testing.
type TestEvent struct {
	shared.BaseDomainEvent
	Data string
}

// NewTestEvent creates a new test event.
func NewTestEvent(eventType string, tenantID uuid.UUID) *TestEvent {
	return &TestEvent{
		BaseDomainEvent: shared.BaseDomainEvent{
			ID:            uuid.New(),
			Type:          eventType,
			TenantIDValue: tenantID,
			Timestamp:     time.Now(),
			AggID:         uuid.New(),
			AggType:       "TestAggregate",
		},
		Data: "test-data",
	}
}

// NewTestEventWithID creates a test event with a specific event ID.
func NewTestEventWithID(eventID uuid.UUID, eventType string, tenantID uuid.UUID) *TestEvent {
	return &TestEvent{
		BaseDomainEvent: shared.BaseDomainEvent{
			ID:            eventID,
			Type:          eventType,
			TenantIDValue: tenantID,
			Timestamp:     time.Now(),
			AggID:         uuid.New(),
			AggType:       "TestAggregate",
		},
		Data: "test-data",
	}
}

// WaitForCondition waits for a condition to become true.
// Returns true if the condition was met, false if timeout occurred.
func WaitForCondition(t *testing.T, condition func() bool, timeout, interval time.Duration) bool {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(interval)
	}
	return false
}

// WaitForEventCount waits until the handler has processed at least n events.
func WaitForEventCount(t *testing.T, handler *MockEventHandler, count int, timeout time.Duration) bool {
	t.Helper()

	return WaitForCondition(t, func() bool {
		return handler.HandledCount() >= count
	}, timeout, 10*time.Millisecond)
}
