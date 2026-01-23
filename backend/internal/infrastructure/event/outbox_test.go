package event

import (
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOutboxEntry(t *testing.T) {
	tenantID := uuid.New()
	event := newTestEvent("TestEvent", tenantID)
	payload := []byte(`{"test": true}`)

	entry := shared.NewOutboxEntry(tenantID, event, payload)

	assert.NotEqual(t, uuid.Nil, entry.ID)
	assert.Equal(t, tenantID, entry.TenantID)
	assert.Equal(t, event.EventID(), entry.EventID)
	assert.Equal(t, "TestEvent", entry.EventType)
	assert.Equal(t, event.AggregateID(), entry.AggregateID)
	assert.Equal(t, "TestAggregate", entry.AggregateType)
	assert.Equal(t, payload, entry.Payload)
	assert.Equal(t, shared.OutboxStatusPending, entry.Status)
	assert.Equal(t, 0, entry.RetryCount)
	assert.Equal(t, shared.DefaultMaxRetries, entry.MaxRetries)
}

func TestOutboxEntry_CanRetry(t *testing.T) {
	tests := []struct {
		name       string
		status     shared.OutboxStatus
		retryCount int
		maxRetries int
		expected   bool
	}{
		{
			name:       "pending cannot retry",
			status:     shared.OutboxStatusPending,
			retryCount: 0,
			maxRetries: 5,
			expected:   false,
		},
		{
			name:       "failed with retries left can retry",
			status:     shared.OutboxStatusFailed,
			retryCount: 2,
			maxRetries: 5,
			expected:   true,
		},
		{
			name:       "failed at max retries cannot retry",
			status:     shared.OutboxStatusFailed,
			retryCount: 5,
			maxRetries: 5,
			expected:   false,
		},
		{
			name:       "dead cannot retry",
			status:     shared.OutboxStatusDead,
			retryCount: 5,
			maxRetries: 5,
			expected:   false,
		},
		{
			name:       "sent cannot retry",
			status:     shared.OutboxStatusSent,
			retryCount: 0,
			maxRetries: 5,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &shared.OutboxEntry{
				Status:     tt.status,
				RetryCount: tt.retryCount,
				MaxRetries: tt.maxRetries,
			}

			assert.Equal(t, tt.expected, entry.CanRetry())
		})
	}
}

func TestOutboxEntry_MarkProcessing(t *testing.T) {
	t.Run("from pending", func(t *testing.T) {
		entry := &shared.OutboxEntry{Status: shared.OutboxStatusPending}

		err := entry.MarkProcessing()

		require.NoError(t, err)
		assert.Equal(t, shared.OutboxStatusProcessing, entry.Status)
	})

	t.Run("from failed", func(t *testing.T) {
		entry := &shared.OutboxEntry{Status: shared.OutboxStatusFailed}

		err := entry.MarkProcessing()

		require.NoError(t, err)
		assert.Equal(t, shared.OutboxStatusProcessing, entry.Status)
	})

	t.Run("from sent fails", func(t *testing.T) {
		entry := &shared.OutboxEntry{Status: shared.OutboxStatusSent}

		err := entry.MarkProcessing()

		require.Error(t, err)
	})
}

func TestOutboxEntry_MarkSent(t *testing.T) {
	entry := &shared.OutboxEntry{Status: shared.OutboxStatusProcessing}

	entry.MarkSent()

	assert.Equal(t, shared.OutboxStatusSent, entry.Status)
	assert.NotNil(t, entry.ProcessedAt)
}

func TestOutboxEntry_MarkFailed(t *testing.T) {
	t.Run("first failure", func(t *testing.T) {
		entry := &shared.OutboxEntry{
			Status:     shared.OutboxStatusProcessing,
			RetryCount: 0,
			MaxRetries: 5,
		}

		entry.MarkFailed("test error")

		assert.Equal(t, shared.OutboxStatusFailed, entry.Status)
		assert.Equal(t, 1, entry.RetryCount)
		assert.Equal(t, "test error", entry.LastError)
		assert.NotNil(t, entry.NextRetryAt)
		// First retry should be 1 second from now (approximately)
		assert.True(t, entry.NextRetryAt.After(time.Now()))
		assert.True(t, entry.NextRetryAt.Before(time.Now().Add(2*time.Second)))
	})

	t.Run("max retries exceeded becomes dead", func(t *testing.T) {
		entry := &shared.OutboxEntry{
			Status:     shared.OutboxStatusProcessing,
			RetryCount: 4,
			MaxRetries: 5,
		}

		entry.MarkFailed("final error")

		assert.Equal(t, shared.OutboxStatusDead, entry.Status)
		assert.Equal(t, 5, entry.RetryCount)
	})

	t.Run("exponential backoff", func(t *testing.T) {
		entry := &shared.OutboxEntry{
			Status:     shared.OutboxStatusProcessing,
			RetryCount: 3, // After this, retry count will be 4
			MaxRetries: 5,
		}

		before := time.Now()
		entry.MarkFailed("error")

		// 4th retry should have 8 second backoff (2^3 = 8)
		assert.True(t, entry.NextRetryAt.After(before.Add(7*time.Second)))
		assert.True(t, entry.NextRetryAt.Before(before.Add(10*time.Second)))
	})
}
