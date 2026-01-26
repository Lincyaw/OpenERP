package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryIdempotencyStore_MarkProcessed(t *testing.T) {
	store := NewInMemoryIdempotencyStore()
	defer store.Close()

	ctx := context.Background()

	t.Run("marks new event as processed", func(t *testing.T) {
		eventID := "event-1"
		ttl := 1 * time.Hour

		isNew, err := store.MarkProcessed(ctx, eventID, ttl)
		require.NoError(t, err)
		assert.True(t, isNew, "new event should return true")
	})

	t.Run("returns false for already processed event", func(t *testing.T) {
		eventID := "event-2"
		ttl := 1 * time.Hour

		// First call
		isNew, err := store.MarkProcessed(ctx, eventID, ttl)
		require.NoError(t, err)
		assert.True(t, isNew)

		// Second call - should return false
		isNew, err = store.MarkProcessed(ctx, eventID, ttl)
		require.NoError(t, err)
		assert.False(t, isNew, "already processed event should return false")
	})

	t.Run("allows reprocessing after expiration", func(t *testing.T) {
		eventID := "event-3"
		ttl := 10 * time.Millisecond

		// First call
		isNew, err := store.MarkProcessed(ctx, eventID, ttl)
		require.NoError(t, err)
		assert.True(t, isNew)

		// Wait for expiration
		time.Sleep(20 * time.Millisecond)

		// Should allow reprocessing after expiration
		isNew, err = store.MarkProcessed(ctx, eventID, ttl)
		require.NoError(t, err)
		assert.True(t, isNew, "expired event should be reprocessable")
	})
}

func TestInMemoryIdempotencyStore_IsProcessed(t *testing.T) {
	store := NewInMemoryIdempotencyStore()
	defer store.Close()

	ctx := context.Background()

	t.Run("returns false for unprocessed event", func(t *testing.T) {
		processed, err := store.IsProcessed(ctx, "unknown-event")
		require.NoError(t, err)
		assert.False(t, processed)
	})

	t.Run("returns true for processed event", func(t *testing.T) {
		eventID := "processed-event"
		_, err := store.MarkProcessed(ctx, eventID, 1*time.Hour)
		require.NoError(t, err)

		processed, err := store.IsProcessed(ctx, eventID)
		require.NoError(t, err)
		assert.True(t, processed)
	})

	t.Run("returns false for expired event", func(t *testing.T) {
		eventID := "expired-event"
		_, err := store.MarkProcessed(ctx, eventID, 10*time.Millisecond)
		require.NoError(t, err)

		// Wait for expiration
		time.Sleep(20 * time.Millisecond)

		processed, err := store.IsProcessed(ctx, eventID)
		require.NoError(t, err)
		assert.False(t, processed, "expired event should return false")
	})
}

func TestInMemoryIdempotencyStore_Size(t *testing.T) {
	store := NewInMemoryIdempotencyStore()
	defer store.Close()

	ctx := context.Background()

	assert.Equal(t, 0, store.Size(), "empty store should have size 0")

	// Add some events
	store.MarkProcessed(ctx, "event-1", 1*time.Hour)
	assert.Equal(t, 1, store.Size())

	store.MarkProcessed(ctx, "event-2", 1*time.Hour)
	assert.Equal(t, 2, store.Size())

	// Adding same event shouldn't increase size
	store.MarkProcessed(ctx, "event-1", 1*time.Hour)
	assert.Equal(t, 2, store.Size())
}

func TestInMemoryIdempotencyStore_Cleanup(t *testing.T) {
	store := NewInMemoryIdempotencyStore()
	defer store.Close()

	ctx := context.Background()

	// Add events with short TTL
	store.MarkProcessed(ctx, "short-lived-1", 10*time.Millisecond)
	store.MarkProcessed(ctx, "short-lived-2", 10*time.Millisecond)
	store.MarkProcessed(ctx, "long-lived", 1*time.Hour)

	assert.Equal(t, 3, store.Size())

	// Wait for short-lived entries to expire
	time.Sleep(20 * time.Millisecond)

	// Manually trigger cleanup
	store.cleanup()

	// Only long-lived entry should remain
	assert.Equal(t, 1, store.Size())

	// Verify the long-lived entry is still there
	processed, err := store.IsProcessed(ctx, "long-lived")
	require.NoError(t, err)
	assert.True(t, processed)

	// Verify short-lived entries are gone
	processed, err = store.IsProcessed(ctx, "short-lived-1")
	require.NoError(t, err)
	assert.False(t, processed)
}

func TestInMemoryIdempotencyStore_ConcurrentAccess(t *testing.T) {
	store := NewInMemoryIdempotencyStore()
	defer store.Close()

	ctx := context.Background()
	const numGoroutines = 100
	const eventID = "concurrent-event"

	// Channel to collect results
	results := make(chan bool, numGoroutines)

	// Launch concurrent goroutines trying to mark the same event
	for i := 0; i < numGoroutines; i++ {
		go func() {
			isNew, err := store.MarkProcessed(ctx, eventID, 1*time.Hour)
			if err != nil {
				results <- false
			} else {
				results <- isNew
			}
		}()
	}

	// Collect results
	newCount := 0
	duplicateCount := 0
	for i := 0; i < numGoroutines; i++ {
		if <-results {
			newCount++
		} else {
			duplicateCount++
		}
	}

	// Exactly one goroutine should have marked it as new
	assert.Equal(t, 1, newCount, "exactly one goroutine should mark as new")
	assert.Equal(t, numGoroutines-1, duplicateCount, "all others should be duplicates")
}

func TestInMemoryIdempotencyStore_Close(t *testing.T) {
	store := NewInMemoryIdempotencyStore()

	// Close should not panic and should return nil
	err := store.Close()
	assert.NoError(t, err)

	// Multiple closes should be safe
	err = store.Close()
	assert.NoError(t, err)
}
