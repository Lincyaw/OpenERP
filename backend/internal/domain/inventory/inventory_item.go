package inventory

import (
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// InventoryItem represents inventory at a specific warehouse for a specific product.
// It is the aggregate root for inventory operations.
// The composite identifier is WarehouseID + ProductID.
type InventoryItem struct {
	shared.TenantAggregateRoot
	WarehouseID       uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex:idx_inventory_item_warehouse_product,priority:2"`
	ProductID         uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex:idx_inventory_item_warehouse_product,priority:3"`
	AvailableQuantity decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0"` // Available for sale/use
	LockedQuantity    decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0"` // Reserved for pending orders
	UnitCost          decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0"` // Moving weighted average cost
	MinQuantity       decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0"` // Minimum stock threshold for alerts
	MaxQuantity       decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0"` // Maximum stock threshold

	// Associations - loaded lazily
	Batches []StockBatch `gorm:"foreignKey:InventoryItemID;references:ID"`
	Locks   []StockLock  `gorm:"foreignKey:InventoryItemID;references:ID"`
}

// TableName returns the table name for GORM
func (InventoryItem) TableName() string {
	return "inventory_items"
}

// NewInventoryItem creates a new inventory item for a warehouse-product combination
func NewInventoryItem(tenantID, warehouseID, productID uuid.UUID) (*InventoryItem, error) {
	if warehouseID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_WAREHOUSE", "Warehouse ID cannot be empty")
	}
	if productID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_PRODUCT", "Product ID cannot be empty")
	}

	item := &InventoryItem{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		WarehouseID:         warehouseID,
		ProductID:           productID,
		AvailableQuantity:   decimal.Zero,
		LockedQuantity:      decimal.Zero,
		UnitCost:            decimal.Zero,
		MinQuantity:         decimal.Zero,
		MaxQuantity:         decimal.Zero,
		Batches:             make([]StockBatch, 0),
		Locks:               make([]StockLock, 0),
	}

	return item, nil
}

// TotalQuantity returns the total quantity (available + locked)
func (i *InventoryItem) TotalQuantity() decimal.Decimal {
	return i.AvailableQuantity.Add(i.LockedQuantity)
}

// IncreaseStock increases the stock quantity and recalculates unit cost using moving weighted average
// This is typically called during purchase receiving or stock adjustments
func (i *InventoryItem) IncreaseStock(quantity decimal.Decimal, unitCost valueobject.Money, batch *BatchInfo) error {
	if quantity.LessThanOrEqual(decimal.Zero) {
		return shared.NewDomainError("INVALID_QUANTITY", "Quantity must be positive")
	}
	if unitCost.Amount().IsNegative() {
		return shared.NewDomainError("INVALID_COST", "Unit cost cannot be negative")
	}

	oldCost := i.UnitCost
	oldQuantity := i.TotalQuantity()

	// Calculate new weighted average cost
	// New Cost = (Old Quantity * Old Cost + New Quantity * New Cost) / (Old Quantity + New Quantity)
	if oldQuantity.IsZero() {
		i.UnitCost = unitCost.Amount()
	} else {
		totalValue := oldQuantity.Mul(oldCost).Add(quantity.Mul(unitCost.Amount()))
		totalQuantity := oldQuantity.Add(quantity)
		i.UnitCost = totalValue.Div(totalQuantity).Round(4)
	}

	// Increase available quantity
	i.AvailableQuantity = i.AvailableQuantity.Add(quantity)
	i.UpdatedAt = time.Now()
	i.IncrementVersion()

	// Create batch if batch info is provided
	if batch != nil {
		stockBatch := NewStockBatch(i.ID, batch.BatchNumber, batch.ProductionDate, batch.ExpiryDate, quantity, unitCost.Amount())
		i.Batches = append(i.Batches, *stockBatch)
	}

	// Emit events
	i.AddDomainEvent(NewStockIncreasedEvent(i, quantity, unitCost.Amount(), batch))

	// Emit cost changed event if cost actually changed
	if !oldCost.Equal(i.UnitCost) {
		i.AddDomainEvent(NewInventoryCostChangedEvent(i, valueobject.NewMoneyCNY(oldCost), valueobject.NewMoneyCNY(i.UnitCost)))
	}

	return nil
}

// LockStock locks a quantity of stock for a pending order
// Returns the lock ID that must be used to unlock or deduct
func (i *InventoryItem) LockStock(quantity decimal.Decimal, sourceType, sourceID string, expireAt time.Time) (*StockLock, error) {
	if quantity.LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("INVALID_QUANTITY", "Lock quantity must be positive")
	}
	if i.AvailableQuantity.LessThan(quantity) {
		return nil, shared.NewDomainError("INSUFFICIENT_STOCK", "Insufficient available stock to lock")
	}
	if sourceType == "" || sourceID == "" {
		return nil, shared.NewDomainError("INVALID_SOURCE", "Source type and ID are required")
	}

	// Move quantity from available to locked
	i.AvailableQuantity = i.AvailableQuantity.Sub(quantity)
	i.LockedQuantity = i.LockedQuantity.Add(quantity)
	i.UpdatedAt = time.Now()
	i.IncrementVersion()

	// Create the lock record
	lock := NewStockLock(i.ID, quantity, sourceType, sourceID, expireAt)
	i.Locks = append(i.Locks, *lock)

	i.AddDomainEvent(NewStockLockedEvent(i, quantity, lock.ID, sourceType, sourceID))

	return lock, nil
}

// UnlockStock releases a previously locked quantity back to available
func (i *InventoryItem) UnlockStock(lockID uuid.UUID) error {
	// Find the lock
	var lock *StockLock
	var lockIndex int
	for idx := range i.Locks {
		if i.Locks[idx].ID == lockID && !i.Locks[idx].Released && !i.Locks[idx].Consumed {
			lock = &i.Locks[idx]
			lockIndex = idx
			break
		}
	}

	if lock == nil {
		return shared.NewDomainError("LOCK_NOT_FOUND", "Stock lock not found or already released/consumed")
	}

	// Move quantity from locked back to available
	i.LockedQuantity = i.LockedQuantity.Sub(lock.Quantity)
	i.AvailableQuantity = i.AvailableQuantity.Add(lock.Quantity)
	i.UpdatedAt = time.Now()
	i.IncrementVersion()

	// Mark lock as released
	i.Locks[lockIndex].Release()

	i.AddDomainEvent(NewStockUnlockedEvent(i, lock.Quantity, lockID, lock.SourceType, lock.SourceID))

	return nil
}

// DeductStock deducts locked stock (actual shipment/consumption)
// The lock must exist and match the quantity
func (i *InventoryItem) DeductStock(lockID uuid.UUID) error {
	// Find the lock
	var lock *StockLock
	var lockIndex int
	for idx := range i.Locks {
		if i.Locks[idx].ID == lockID && !i.Locks[idx].Released && !i.Locks[idx].Consumed {
			lock = &i.Locks[idx]
			lockIndex = idx
			break
		}
	}

	if lock == nil {
		return shared.NewDomainError("LOCK_NOT_FOUND", "Stock lock not found or already released/consumed")
	}

	// Deduct from locked quantity
	i.LockedQuantity = i.LockedQuantity.Sub(lock.Quantity)
	i.UpdatedAt = time.Now()
	i.IncrementVersion()

	// Mark lock as consumed
	i.Locks[lockIndex].Consume()

	i.AddDomainEvent(NewStockDeductedEvent(i, lock.Quantity, lockID, lock.SourceType, lock.SourceID))

	// Check if below minimum threshold
	if i.MinQuantity.GreaterThan(decimal.Zero) && i.TotalQuantity().LessThan(i.MinQuantity) {
		i.AddDomainEvent(NewStockBelowThresholdEvent(i))
	}

	return nil
}

// AdjustStock adjusts the stock to match actual quantity (used during stock taking/counting)
// The reason is recorded for audit purposes
func (i *InventoryItem) AdjustStock(actualQuantity decimal.Decimal, reason string) error {
	if actualQuantity.IsNegative() {
		return shared.NewDomainError("INVALID_QUANTITY", "Actual quantity cannot be negative")
	}
	if reason == "" {
		return shared.NewDomainError("INVALID_REASON", "Adjustment reason is required")
	}

	// Cannot adjust if there are outstanding locks
	if i.LockedQuantity.GreaterThan(decimal.Zero) {
		return shared.NewDomainError("HAS_LOCKED_STOCK", "Cannot adjust stock while there are outstanding locks")
	}

	oldQuantity := i.AvailableQuantity
	difference := actualQuantity.Sub(oldQuantity)

	i.AvailableQuantity = actualQuantity
	i.UpdatedAt = time.Now()
	i.IncrementVersion()

	i.AddDomainEvent(NewStockAdjustedEvent(i, oldQuantity, actualQuantity, difference, reason))

	// Check thresholds
	if i.MinQuantity.GreaterThan(decimal.Zero) && i.TotalQuantity().LessThan(i.MinQuantity) {
		i.AddDomainEvent(NewStockBelowThresholdEvent(i))
	}

	return nil
}

// SetMinQuantity sets the minimum stock threshold for alerts
func (i *InventoryItem) SetMinQuantity(quantity decimal.Decimal) error {
	if quantity.IsNegative() {
		return shared.NewDomainError("INVALID_QUANTITY", "Minimum quantity cannot be negative")
	}

	i.MinQuantity = quantity
	i.UpdatedAt = time.Now()
	i.IncrementVersion()

	return nil
}

// SetMaxQuantity sets the maximum stock threshold
func (i *InventoryItem) SetMaxQuantity(quantity decimal.Decimal) error {
	if quantity.IsNegative() {
		return shared.NewDomainError("INVALID_QUANTITY", "Maximum quantity cannot be negative")
	}

	i.MaxQuantity = quantity
	i.UpdatedAt = time.Now()
	i.IncrementVersion()

	return nil
}

// GetUnitCostMoney returns unit cost as Money value object
func (i *InventoryItem) GetUnitCostMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(i.UnitCost)
}

// GetTotalValue returns the total inventory value (total quantity * unit cost)
func (i *InventoryItem) GetTotalValue() valueobject.Money {
	return valueobject.NewMoneyCNY(i.TotalQuantity().Mul(i.UnitCost))
}

// IsBelowMinimum returns true if total quantity is below minimum threshold
func (i *InventoryItem) IsBelowMinimum() bool {
	return i.MinQuantity.GreaterThan(decimal.Zero) && i.TotalQuantity().LessThan(i.MinQuantity)
}

// IsAboveMaximum returns true if total quantity is above maximum threshold
func (i *InventoryItem) IsAboveMaximum() bool {
	return i.MaxQuantity.GreaterThan(decimal.Zero) && i.TotalQuantity().GreaterThan(i.MaxQuantity)
}

// HasAvailableStock returns true if there is available stock
func (i *InventoryItem) HasAvailableStock() bool {
	return i.AvailableQuantity.GreaterThan(decimal.Zero)
}

// CanFulfill returns true if the available quantity can fulfill the requested quantity
func (i *InventoryItem) CanFulfill(quantity decimal.Decimal) bool {
	return i.AvailableQuantity.GreaterThanOrEqual(quantity)
}

// GetActiveLocks returns all active (non-released, non-consumed) locks
func (i *InventoryItem) GetActiveLocks() []StockLock {
	activeLocks := make([]StockLock, 0)
	for _, lock := range i.Locks {
		if !lock.Released && !lock.Consumed {
			activeLocks = append(activeLocks, lock)
		}
	}
	return activeLocks
}

// GetExpiredLocks returns locks that have expired but are not yet released
func (i *InventoryItem) GetExpiredLocks() []StockLock {
	expiredLocks := make([]StockLock, 0)
	now := time.Now()
	for _, lock := range i.Locks {
		if !lock.Released && !lock.Consumed && lock.ExpireAt.Before(now) {
			expiredLocks = append(expiredLocks, lock)
		}
	}
	return expiredLocks
}

// ReleaseExpiredLocks releases all expired locks back to available stock
func (i *InventoryItem) ReleaseExpiredLocks() int {
	expiredLocks := i.GetExpiredLocks()
	count := 0
	for _, lock := range expiredLocks {
		if err := i.UnlockStock(lock.ID); err == nil {
			count++
		}
	}
	return count
}

// BatchInfo contains information for creating a stock batch
type BatchInfo struct {
	BatchNumber    string
	ProductionDate *time.Time
	ExpiryDate     *time.Time
}

// NewBatchInfo creates a new BatchInfo
func NewBatchInfo(batchNumber string, productionDate, expiryDate *time.Time) *BatchInfo {
	return &BatchInfo{
		BatchNumber:    batchNumber,
		ProductionDate: productionDate,
		ExpiryDate:     expiryDate,
	}
}
