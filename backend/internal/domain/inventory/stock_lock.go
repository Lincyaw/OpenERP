package inventory

import (
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// StockLock represents a reservation of stock for a pending order or operation
type StockLock struct {
	shared.BaseEntity
	InventoryItemID uuid.UUID
	Quantity        decimal.Decimal // Locked quantity
	SourceType      string          // e.g., "sales_order", "transfer"
	SourceID        string          // ID of the source document
	ExpireAt        time.Time       // When the lock expires
	Released        bool            // Whether the lock was released (cancelled)
	Consumed        bool            // Whether the lock was consumed (fulfilled)
	ReleasedAt      *time.Time      // When the lock was released/consumed
}

// NewStockLock creates a new stock lock
func NewStockLock(
	inventoryItemID uuid.UUID,
	quantity decimal.Decimal,
	sourceType, sourceID string,
	expireAt time.Time,
) *StockLock {
	return &StockLock{
		BaseEntity:      shared.NewBaseEntity(),
		InventoryItemID: inventoryItemID,
		Quantity:        quantity,
		SourceType:      sourceType,
		SourceID:        sourceID,
		ExpireAt:        expireAt,
		Released:        false,
		Consumed:        false,
	}
}

// IsActive returns true if the lock is still active (not released or consumed)
func (l *StockLock) IsActive() bool {
	return !l.Released && !l.Consumed
}

// IsExpired returns true if the lock has expired based on current time.
//
// WARNING: This method uses time.Now() which can cause race conditions in concurrent
// environments. For critical business logic (e.g., releasing expired locks, validating
// lock status before operations), use IsExpiredAt() with a reference timestamp captured
// at the start of the operation, or rely on database-level timestamp comparisons in the
// repository layer (FindExpired, ReleaseExpired).
//
// This method is suitable for:
// - Display purposes (UI showing if a lock is expired)
// - Non-critical status checks
//
// BUG-013: See IsExpiredAt() for atomic expiration checking.
func (l *StockLock) IsExpired() bool {
	return l.IsExpiredAt(time.Now())
}

// IsExpiredAt returns true if the lock has expired relative to the given reference time.
//
// This method should be used for critical business operations where atomicity matters.
// By passing a reference timestamp captured at the start of an operation (or from the
// database query), you ensure consistent expiration checking throughout the operation,
// preventing race conditions between check and action.
//
// Example usage:
//
//	// Capture reference time once at operation start
//	referenceTime := time.Now()
//
//	// Use same reference for all expiration checks in this operation
//	for _, lock := range locks {
//	    if lock.IsExpiredAt(referenceTime) {
//	        // Process expired lock
//	    }
//	}
//
// BUG-013: This method addresses the non-atomic lock expiration check issue.
func (l *StockLock) IsExpiredAt(referenceTime time.Time) bool {
	return referenceTime.After(l.ExpireAt)
}

// Release marks the lock as released (cancellation)
func (l *StockLock) Release() {
	now := time.Now()
	l.Released = true
	l.ReleasedAt = &now
	l.UpdatedAt = now
}

// Consume marks the lock as consumed (fulfillment)
func (l *StockLock) Consume() {
	now := time.Now()
	l.Consumed = true
	l.ReleasedAt = &now
	l.UpdatedAt = now
}

// TimeUntilExpiry returns the duration until the lock expires
// Returns negative duration if already expired
func (l *StockLock) TimeUntilExpiry() time.Duration {
	return time.Until(l.ExpireAt)
}

// MinutesUntilExpiry returns the minutes until expiry, negative if expired
func (l *StockLock) MinutesUntilExpiry() int {
	return int(l.TimeUntilExpiry().Minutes())
}
