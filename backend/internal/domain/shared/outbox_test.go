package shared

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestOutboxEntry_ResetForRetry(t *testing.T) {
	t.Run("resets dead letter entry for retry", func(t *testing.T) {
		entry := &OutboxEntry{
			ID:          uuid.New(),
			TenantID:    uuid.New(),
			EventID:     uuid.New(),
			EventType:   "TestEvent",
			AggregateID: uuid.New(),
			Status:      OutboxStatusDead,
			RetryCount:  5,
			MaxRetries:  5,
			LastError:   "some error",
			NextRetryAt: nil,
			CreatedAt:   time.Now().Add(-time.Hour),
			UpdatedAt:   time.Now().Add(-time.Minute),
		}

		err := entry.ResetForRetry()
		assert.NoError(t, err)
		assert.Equal(t, OutboxStatusPending, entry.Status)
		assert.Equal(t, 0, entry.RetryCount)
		assert.Empty(t, entry.LastError)
		assert.Nil(t, entry.NextRetryAt)
		assert.True(t, entry.UpdatedAt.After(time.Now().Add(-time.Second)))
	})

	t.Run("fails for non-dead entry", func(t *testing.T) {
		testCases := []OutboxStatus{
			OutboxStatusPending,
			OutboxStatusProcessing,
			OutboxStatusSent,
			OutboxStatusFailed,
		}

		for _, status := range testCases {
			entry := &OutboxEntry{
				ID:     uuid.New(),
				Status: status,
			}
			err := entry.ResetForRetry()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "can only retry dead letter entries")
		}
	})
}

func TestOutboxEntry_IsDead(t *testing.T) {
	t.Run("returns true for dead entries", func(t *testing.T) {
		entry := &OutboxEntry{Status: OutboxStatusDead}
		assert.True(t, entry.IsDead())
	})

	t.Run("returns false for non-dead entries", func(t *testing.T) {
		testCases := []OutboxStatus{
			OutboxStatusPending,
			OutboxStatusProcessing,
			OutboxStatusSent,
			OutboxStatusFailed,
		}

		for _, status := range testCases {
			entry := &OutboxEntry{Status: status}
			assert.False(t, entry.IsDead())
		}
	})
}

func TestOutboxEntry_MarkFailed_MovesToDeadAfterMaxRetries(t *testing.T) {
	entry := &OutboxEntry{
		ID:         uuid.New(),
		Status:     OutboxStatusProcessing,
		RetryCount: 4, // Already retried 4 times
		MaxRetries: 5,
	}

	entry.MarkFailed("final error")

	assert.Equal(t, OutboxStatusDead, entry.Status)
	assert.Equal(t, 5, entry.RetryCount)
	assert.Equal(t, "final error", entry.LastError)
	assert.True(t, entry.IsDead())
}

func TestOutboxEntry_MarkFailed_ExponentialBackoff(t *testing.T) {
	entry := &OutboxEntry{
		ID:         uuid.New(),
		Status:     OutboxStatusProcessing,
		RetryCount: 0,
		MaxRetries: 5,
	}

	// First failure: 1s backoff
	entry.MarkFailed("error 1")
	assert.Equal(t, OutboxStatusFailed, entry.Status)
	assert.Equal(t, 1, entry.RetryCount)
	assert.NotNil(t, entry.NextRetryAt)
	firstBackoff := entry.NextRetryAt.Sub(time.Now())
	assert.True(t, firstBackoff > 0 && firstBackoff <= 2*time.Second)

	// Second failure: 2s backoff
	entry.Status = OutboxStatusProcessing
	entry.MarkFailed("error 2")
	assert.Equal(t, 2, entry.RetryCount)
	secondBackoff := entry.NextRetryAt.Sub(time.Now())
	assert.True(t, secondBackoff > time.Second && secondBackoff <= 3*time.Second)

	// Third failure: 4s backoff
	entry.Status = OutboxStatusProcessing
	entry.MarkFailed("error 3")
	assert.Equal(t, 3, entry.RetryCount)
	thirdBackoff := entry.NextRetryAt.Sub(time.Now())
	assert.True(t, thirdBackoff > 3*time.Second && thirdBackoff <= 5*time.Second)
}
