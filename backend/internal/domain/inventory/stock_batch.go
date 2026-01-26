package inventory

import (
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// StockBatch represents a batch of stock with specific attributes
// (production date, expiry date, batch number, etc.)
type StockBatch struct {
	shared.BaseEntity
	InventoryItemID uuid.UUID
	BatchNumber     string          // Batch/lot number
	ProductionDate  *time.Time      // Date of production (optional)
	ExpiryDate      *time.Time      // Expiry date (optional)
	Quantity        decimal.Decimal // Quantity in this batch
	UnitCost        decimal.Decimal // Cost per unit for this batch
	Consumed        bool            // Whether this batch is fully consumed
}

// NewStockBatch creates a new stock batch
func NewStockBatch(
	inventoryItemID uuid.UUID,
	batchNumber string,
	productionDate, expiryDate *time.Time,
	quantity decimal.Decimal,
	unitCost decimal.Decimal,
) *StockBatch {
	return &StockBatch{
		BaseEntity:      shared.NewBaseEntity(),
		InventoryItemID: inventoryItemID,
		BatchNumber:     batchNumber,
		ProductionDate:  productionDate,
		ExpiryDate:      expiryDate,
		Quantity:        quantity,
		UnitCost:        unitCost,
		Consumed:        false,
	}
}

// IsExpired returns true if the batch has expired
func (b *StockBatch) IsExpired() bool {
	if b.ExpiryDate == nil {
		return false
	}
	return b.ExpiryDate.Before(time.Now())
}

// WillExpireWithin returns true if the batch will expire within the given duration
func (b *StockBatch) WillExpireWithin(duration time.Duration) bool {
	if b.ExpiryDate == nil {
		return false
	}
	return b.ExpiryDate.Before(time.Now().Add(duration))
}

// DaysUntilExpiry returns the number of days until expiry, -1 if no expiry date
func (b *StockBatch) DaysUntilExpiry() int {
	if b.ExpiryDate == nil {
		return -1
	}
	duration := time.Until(*b.ExpiryDate)
	return int(duration.Hours() / 24)
}

// Deduct reduces the batch quantity
// Returns the actual quantity deducted (may be less than requested if batch has insufficient)
func (b *StockBatch) Deduct(quantity decimal.Decimal) decimal.Decimal {
	if quantity.GreaterThan(b.Quantity) {
		deducted := b.Quantity
		b.Quantity = decimal.Zero
		b.Consumed = true
		b.UpdatedAt = time.Now()
		return deducted
	}

	b.Quantity = b.Quantity.Sub(quantity)
	if b.Quantity.IsZero() {
		b.Consumed = true
	}
	b.UpdatedAt = time.Now()
	return quantity
}

// Add increases the batch quantity (for returns or adjustments)
func (b *StockBatch) Add(quantity decimal.Decimal) {
	b.Quantity = b.Quantity.Add(quantity)
	if b.Consumed && b.Quantity.GreaterThan(decimal.Zero) {
		b.Consumed = false
	}
	b.UpdatedAt = time.Now()
}

// GetTotalValue returns the total value of this batch
func (b *StockBatch) GetTotalValue() decimal.Decimal {
	return b.Quantity.Mul(b.UnitCost)
}

// HasStock returns true if the batch has available quantity
func (b *StockBatch) HasStock() bool {
	return b.Quantity.GreaterThan(decimal.Zero) && !b.Consumed
}

// IsAvailable returns true if the batch can be used (not consumed and not expired)
func (b *StockBatch) IsAvailable() bool {
	return b.HasStock() && !b.IsExpired()
}
