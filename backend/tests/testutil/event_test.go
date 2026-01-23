package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMockEventHandler(t *testing.T) {
	handler := NewMockEventHandler("Event1", "Event2")

	assert.Equal(t, []string{"Event1", "Event2"}, handler.EventTypes())
	assert.Equal(t, 0, handler.HandledCount())
}

func TestMockEventHandler_Handle(t *testing.T) {
	handler := NewMockEventHandler("TestEvent")
	tenantID := uuid.New()
	event := NewTestEvent("TestEvent", tenantID)

	err := handler.Handle(context.Background(), event)

	require.NoError(t, err)
	assert.Equal(t, 1, handler.HandledCount())
	assert.Equal(t, event, handler.Handled()[0])
}

func TestMockEventHandler_SetError(t *testing.T) {
	handler := NewMockEventHandler("TestEvent")
	expectedErr := assert.AnError

	handler.SetError(expectedErr)

	err := handler.Handle(context.Background(), NewTestEvent("TestEvent", uuid.New()))
	assert.Equal(t, expectedErr, err)
}

func TestMockEventHandler_Reset(t *testing.T) {
	handler := NewMockEventHandler("TestEvent")
	handler.SetError(assert.AnError)

	_ = handler.Handle(context.Background(), NewTestEvent("TestEvent", uuid.New()))
	assert.Equal(t, 1, handler.HandledCount())

	handler.Reset()

	assert.Equal(t, 0, handler.HandledCount())
}

func TestNewTestEvent(t *testing.T) {
	tenantID := uuid.New()
	event := NewTestEvent("TestEvent", tenantID)

	assert.NotEqual(t, uuid.Nil, event.EventID())
	assert.Equal(t, "TestEvent", event.EventType())
	assert.Equal(t, tenantID, event.TenantID())
	assert.False(t, event.OccurredAt().IsZero())
	assert.Equal(t, "test-data", event.Data)
}

func TestNewTestEventWithID(t *testing.T) {
	eventID := uuid.New()
	tenantID := uuid.New()
	event := NewTestEventWithID(eventID, "CustomEvent", tenantID)

	assert.Equal(t, eventID, event.EventID())
	assert.Equal(t, "CustomEvent", event.EventType())
	assert.Equal(t, tenantID, event.TenantID())
}

func TestWaitForCondition(t *testing.T) {
	t.Run("condition met", func(t *testing.T) {
		counter := 0
		go func() {
			time.Sleep(20 * time.Millisecond)
			counter = 1
		}()

		result := WaitForCondition(t, func() bool {
			return counter == 1
		}, 200*time.Millisecond, 10*time.Millisecond)

		assert.True(t, result)
	})

	t.Run("condition not met within timeout", func(t *testing.T) {
		result := WaitForCondition(t, func() bool {
			return false
		}, 50*time.Millisecond, 10*time.Millisecond)

		assert.False(t, result)
	})
}

func TestWaitForEventCount(t *testing.T) {
	handler := NewMockEventHandler("TestEvent")
	tenantID := uuid.New()

	go func() {
		time.Sleep(20 * time.Millisecond)
		_ = handler.Handle(nil, NewTestEvent("TestEvent", tenantID))
		_ = handler.Handle(nil, NewTestEvent("TestEvent", tenantID))
	}()

	result := WaitForEventCount(t, handler, 2, 200*time.Millisecond)
	assert.True(t, result)
}
