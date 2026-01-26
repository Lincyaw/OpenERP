package inventory

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestStockLock_NewStockLock(t *testing.T) {
	inventoryItemID := uuid.New()
	quantity := decimal.NewFromInt(100)
	expireAt := time.Now().Add(time.Hour)

	lock := NewStockLock(inventoryItemID, quantity, "sales_order", "SO-001", expireAt)

	assert.NotEqual(t, uuid.Nil, lock.ID)
	assert.Equal(t, inventoryItemID, lock.InventoryItemID)
	assert.Equal(t, quantity, lock.Quantity)
	assert.Equal(t, "sales_order", lock.SourceType)
	assert.Equal(t, "SO-001", lock.SourceID)
	assert.Equal(t, expireAt, lock.ExpireAt)
	assert.False(t, lock.Released)
	assert.False(t, lock.Consumed)
	assert.Nil(t, lock.ReleasedAt)
}

func TestStockLock_IsActive(t *testing.T) {
	t.Run("returns true when not released and not consumed", func(t *testing.T) {
		lock := createTestStockLock()
		assert.True(t, lock.IsActive())
	})

	t.Run("returns false when released", func(t *testing.T) {
		lock := createTestStockLock()
		lock.Release()
		assert.False(t, lock.IsActive())
	})

	t.Run("returns false when consumed", func(t *testing.T) {
		lock := createTestStockLock()
		lock.Consume()
		assert.False(t, lock.IsActive())
	})
}

// BUG-013: Tests for atomic expiration checking
func TestStockLock_IsExpired(t *testing.T) {
	t.Run("returns false for future expiration", func(t *testing.T) {
		lock := NewStockLock(
			uuid.New(),
			decimal.NewFromInt(100),
			"sales_order",
			"SO-001",
			time.Now().Add(time.Hour), // Expires in 1 hour
		)

		assert.False(t, lock.IsExpired())
	})

	t.Run("returns true for past expiration", func(t *testing.T) {
		lock := NewStockLock(
			uuid.New(),
			decimal.NewFromInt(100),
			"sales_order",
			"SO-001",
			time.Now().Add(-time.Hour), // Expired 1 hour ago
		)

		assert.True(t, lock.IsExpired())
	})
}

// BUG-013: Tests for IsExpiredAt - the atomic expiration check method
func TestStockLock_IsExpiredAt(t *testing.T) {
	t.Run("returns false when reference time is before expiration", func(t *testing.T) {
		expireAt := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
		lock := NewStockLock(
			uuid.New(),
			decimal.NewFromInt(100),
			"sales_order",
			"SO-001",
			expireAt,
		)

		// Reference time is 1 hour before expiration
		referenceTime := time.Date(2024, 6, 15, 11, 0, 0, 0, time.UTC)
		assert.False(t, lock.IsExpiredAt(referenceTime))
	})

	t.Run("returns true when reference time is after expiration", func(t *testing.T) {
		expireAt := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
		lock := NewStockLock(
			uuid.New(),
			decimal.NewFromInt(100),
			"sales_order",
			"SO-001",
			expireAt,
		)

		// Reference time is 1 hour after expiration
		referenceTime := time.Date(2024, 6, 15, 13, 0, 0, 0, time.UTC)
		assert.True(t, lock.IsExpiredAt(referenceTime))
	})

	t.Run("returns false when reference time equals expiration", func(t *testing.T) {
		expireAt := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
		lock := NewStockLock(
			uuid.New(),
			decimal.NewFromInt(100),
			"sales_order",
			"SO-001",
			expireAt,
		)

		// Reference time equals expiration time exactly
		referenceTime := expireAt
		assert.False(t, lock.IsExpiredAt(referenceTime))
	})

	t.Run("consistent results with same reference time across multiple locks", func(t *testing.T) {
		// BUG-013: This test demonstrates the atomic checking pattern
		// Using a single reference time ensures all locks are evaluated
		// with the same point in time, preventing race conditions

		referenceTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

		// Lock 1: expires before reference time (expired)
		lock1 := NewStockLock(
			uuid.New(),
			decimal.NewFromInt(50),
			"sales_order",
			"SO-001",
			time.Date(2024, 6, 15, 11, 0, 0, 0, time.UTC), // 1 hour before reference
		)

		// Lock 2: expires after reference time (not expired)
		lock2 := NewStockLock(
			uuid.New(),
			decimal.NewFromInt(50),
			"sales_order",
			"SO-002",
			time.Date(2024, 6, 15, 13, 0, 0, 0, time.UTC), // 1 hour after reference
		)

		// Lock 3: expires at reference time (not expired - boundary)
		lock3 := NewStockLock(
			uuid.New(),
			decimal.NewFromInt(50),
			"sales_order",
			"SO-003",
			referenceTime, // Exactly at reference
		)

		// All checks use the same reference time - atomic behavior
		assert.True(t, lock1.IsExpiredAt(referenceTime), "lock1 should be expired")
		assert.False(t, lock2.IsExpiredAt(referenceTime), "lock2 should not be expired")
		assert.False(t, lock3.IsExpiredAt(referenceTime), "lock3 should not be expired (boundary)")
	})
}

func TestStockLock_Release(t *testing.T) {
	lock := createTestStockLock()

	lock.Release()

	assert.True(t, lock.Released)
	assert.NotNil(t, lock.ReleasedAt)
	assert.False(t, lock.IsActive())
}

func TestStockLock_Consume(t *testing.T) {
	lock := createTestStockLock()

	lock.Consume()

	assert.True(t, lock.Consumed)
	assert.NotNil(t, lock.ReleasedAt)
	assert.False(t, lock.IsActive())
}

func TestStockLock_TimeUntilExpiry(t *testing.T) {
	t.Run("returns positive duration for future expiry", func(t *testing.T) {
		lock := NewStockLock(
			uuid.New(),
			decimal.NewFromInt(100),
			"sales_order",
			"SO-001",
			time.Now().Add(time.Hour),
		)

		duration := lock.TimeUntilExpiry()
		assert.True(t, duration > 0)
		assert.True(t, duration <= time.Hour)
	})

	t.Run("returns negative duration for past expiry", func(t *testing.T) {
		lock := NewStockLock(
			uuid.New(),
			decimal.NewFromInt(100),
			"sales_order",
			"SO-001",
			time.Now().Add(-time.Hour),
		)

		duration := lock.TimeUntilExpiry()
		assert.True(t, duration < 0)
	})
}

func TestStockLock_MinutesUntilExpiry(t *testing.T) {
	lock := NewStockLock(
		uuid.New(),
		decimal.NewFromInt(100),
		"sales_order",
		"SO-001",
		time.Now().Add(30*time.Minute),
	)

	minutes := lock.MinutesUntilExpiry()
	assert.True(t, minutes >= 29 && minutes <= 30)
}

// Helper function
func createTestStockLock() *StockLock {
	return NewStockLock(
		uuid.New(),
		decimal.NewFromInt(100),
		"sales_order",
		"SO-001",
		time.Now().Add(time.Hour),
	)
}
