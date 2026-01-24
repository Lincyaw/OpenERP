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
	InventoryItemID uuid.UUID       `gorm:"type:uuid;not null;index"`
	Quantity        decimal.Decimal `gorm:"type:decimal(18,4);not null"`                   // Locked quantity
	SourceType      string          `gorm:"type:varchar(50);not null;index:idx_lock_src"`  // e.g., "sales_order", "transfer"
	SourceID        string          `gorm:"type:varchar(100);not null;index:idx_lock_src"` // ID of the source document
	ExpireAt        time.Time       `gorm:"not null;index"`                                // When the lock expires
	Released        bool            `gorm:"not null;default:false"`                        // Whether the lock was released (cancelled)
	Consumed        bool            `gorm:"not null;default:false"`                        // Whether the lock was consumed (fulfilled)
	ReleasedAt      *time.Time      `gorm:"type:timestamp"`                                // When the lock was released/consumed
}

// TableName returns the table name for GORM
func (StockLock) TableName() string {
	return "stock_locks"
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

// IsExpired returns true if the lock has expired
func (l *StockLock) IsExpired() bool {
	return time.Now().After(l.ExpireAt)
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
