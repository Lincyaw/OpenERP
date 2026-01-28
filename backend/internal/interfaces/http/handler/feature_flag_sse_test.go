package handler

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/featureflag"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// MockCacheInvalidator implements featureflag.CacheInvalidator for testing
type MockCacheInvalidator struct {
	mu            sync.Mutex
	publishedMsgs []featureflag.CacheUpdateMessage
	subscribers   []func(msg featureflag.CacheUpdateMessage)
	closed        bool
}

func NewMockCacheInvalidator() *MockCacheInvalidator {
	return &MockCacheInvalidator{
		publishedMsgs: make([]featureflag.CacheUpdateMessage, 0),
		subscribers:   make([]func(msg featureflag.CacheUpdateMessage), 0),
	}
}

func (m *MockCacheInvalidator) Publish(ctx context.Context, msg featureflag.CacheUpdateMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishedMsgs = append(m.publishedMsgs, msg)
	// Notify subscribers
	for _, sub := range m.subscribers {
		go sub(msg)
	}
	return nil
}

func (m *MockCacheInvalidator) Subscribe(ctx context.Context, callback func(msg featureflag.CacheUpdateMessage)) error {
	m.mu.Lock()
	m.subscribers = append(m.subscribers, callback)
	m.mu.Unlock()

	// Block until context is cancelled
	<-ctx.Done()
	return ctx.Err()
}

func (m *MockCacheInvalidator) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *MockCacheInvalidator) GetPublishedMessages() []featureflag.CacheUpdateMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]featureflag.CacheUpdateMessage{}, m.publishedMsgs...)
}

// Tests

func TestNewFeatureFlagSSEHandler(t *testing.T) {
	invalidator := NewMockCacheInvalidator()
	handler := NewFeatureFlagSSEHandler(invalidator)

	assert.NotNil(t, handler)
	assert.Equal(t, 30*time.Second, handler.heartbeat)
}

func TestNewFeatureFlagSSEHandler_WithOptions(t *testing.T) {
	invalidator := NewMockCacheInvalidator()
	logger := zap.NewNop()

	handler := NewFeatureFlagSSEHandler(
		invalidator,
		WithSSELogger(logger),
		WithSSEHeartbeat(10*time.Second),
	)

	assert.NotNil(t, handler)
	assert.Equal(t, 10*time.Second, handler.heartbeat)
	assert.Equal(t, logger, handler.logger)
}

func TestFeatureFlagSSEHandler_Start_Stop(t *testing.T) {
	invalidator := NewMockCacheInvalidator()
	handler := NewFeatureFlagSSEHandler(invalidator, WithSSELogger(zap.NewNop()))

	// Start handler
	err := handler.Start()
	assert.NoError(t, err)

	// Starting again should fail
	err = handler.Start()
	assert.Error(t, err)

	// Stop handler
	handler.Stop()
}

func TestFeatureFlagSSEHandler_GetClientCount(t *testing.T) {
	invalidator := NewMockCacheInvalidator()
	handler := NewFeatureFlagSSEHandler(invalidator, WithSSELogger(zap.NewNop()))

	// Initially no clients
	assert.Equal(t, 0, handler.GetClientCount())

	// Add a client manually (simulating connection)
	client := &SSEClient{
		ID:   "test-client-1",
		Chan: make(chan SSEMessage, 100),
		Done: make(chan struct{}),
	}
	handler.clients.Store(client.ID, client)

	assert.Equal(t, 1, handler.GetClientCount())

	// Add another client
	client2 := &SSEClient{
		ID:   "test-client-2",
		Chan: make(chan SSEMessage, 100),
		Done: make(chan struct{}),
	}
	handler.clients.Store(client2.ID, client2)

	assert.Equal(t, 2, handler.GetClientCount())

	// Remove a client
	handler.clients.Delete(client.ID)
	assert.Equal(t, 1, handler.GetClientCount())
}

func TestFeatureFlagSSEHandler_cacheUpdateToEvent(t *testing.T) {
	invalidator := NewMockCacheInvalidator()
	handler := NewFeatureFlagSSEHandler(invalidator, WithSSELogger(zap.NewNop()))

	testCases := []struct {
		name     string
		action   featureflag.CacheUpdateAction
		flagKey  string
		expected *FlagUpdatedEvent
	}{
		{
			name:    "Updated action",
			action:  featureflag.CacheUpdateActionUpdated,
			flagKey: "test_flag",
			expected: &FlagUpdatedEvent{
				Key: "test_flag",
				Value: FlagUpdatedEventValue{
					Enabled: true,
				},
			},
		},
		{
			name:    "Deleted action",
			action:  featureflag.CacheUpdateActionDeleted,
			flagKey: "deleted_flag",
			expected: &FlagUpdatedEvent{
				Key: "deleted_flag",
				Value: FlagUpdatedEventValue{
					Enabled: false,
				},
			},
		},
		{
			name:    "Override updated action",
			action:  featureflag.CacheUpdateActionOverrideUpdated,
			flagKey: "override_flag",
			expected: &FlagUpdatedEvent{
				Key: "override_flag",
				Value: FlagUpdatedEventValue{
					Enabled: true,
				},
			},
		},
		{
			name:    "Override deleted action",
			action:  featureflag.CacheUpdateActionOverrideDeleted,
			flagKey: "override_flag",
			expected: &FlagUpdatedEvent{
				Key: "override_flag",
				Value: FlagUpdatedEventValue{
					Enabled: true,
				},
			},
		},
		{
			name:    "Invalidate all action",
			action:  featureflag.CacheUpdateActionInvalidateAll,
			flagKey: "",
			expected: &FlagUpdatedEvent{
				Key: "*",
				Value: FlagUpdatedEventValue{
					Enabled: true,
				},
			},
		},
		{
			name:     "Unknown action",
			action:   featureflag.CacheUpdateAction("unknown"),
			flagKey:  "test_flag",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := featureflag.CacheUpdateMessage{
				Action:  tc.action,
				FlagKey: tc.flagKey,
			}

			result := handler.cacheUpdateToEvent(msg)

			if tc.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tc.expected.Key, result.Key)
				assert.Equal(t, tc.expected.Value.Enabled, result.Value.Enabled)
			}
		})
	}
}

func TestFeatureFlagSSEHandler_broadcast(t *testing.T) {
	invalidator := NewMockCacheInvalidator()
	handler := NewFeatureFlagSSEHandler(invalidator, WithSSELogger(zap.NewNop()))

	// Add two clients
	client1Chan := make(chan SSEMessage, 100)
	client1 := &SSEClient{
		ID:   "client-1",
		Chan: client1Chan,
		Done: make(chan struct{}),
	}
	handler.clients.Store(client1.ID, client1)

	client2Chan := make(chan SSEMessage, 100)
	client2 := &SSEClient{
		ID:   "client-2",
		Chan: client2Chan,
		Done: make(chan struct{}),
	}
	handler.clients.Store(client2.ID, client2)

	// Broadcast a message
	msg := SSEMessage{
		Event: "test_event",
		Data:  `{"test": true}`,
		ID:    "123",
	}
	handler.broadcast(msg)

	// Give goroutines time to process
	time.Sleep(100 * time.Millisecond)

	// Both clients should receive the message
	select {
	case received := <-client1Chan:
		assert.Equal(t, msg, received)
	default:
		t.Error("Client 1 did not receive message")
	}

	select {
	case received := <-client2Chan:
		assert.Equal(t, msg, received)
	default:
		t.Error("Client 2 did not receive message")
	}
}

func TestSSEMessage_Format(t *testing.T) {
	msg := SSEMessage{
		Event: "flag_updated",
		Data:  `{"key":"test","value":{"enabled":true}}`,
		ID:    "12345",
	}

	assert.Equal(t, "flag_updated", msg.Event)
	assert.Contains(t, msg.Data, "test")
	assert.Equal(t, "12345", msg.ID)
}

func TestFlagUpdatedEvent_Structure(t *testing.T) {
	variant := "blue"
	event := FlagUpdatedEvent{
		Key: "button_color",
		Value: FlagUpdatedEventValue{
			Enabled: true,
			Variant: &variant,
		},
	}

	assert.Equal(t, "button_color", event.Key)
	assert.True(t, event.Value.Enabled)
	assert.NotNil(t, event.Value.Variant)
	assert.Equal(t, "blue", *event.Value.Variant)
}
