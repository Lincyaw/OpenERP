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
//
// DDD Aggregate Boundary:
// InventoryItem is the aggregate root that manages:
// - Stock quantities (available, locked)
// - Unit cost calculations
// - Stock thresholds (min, max)
// - Child entities: StockBatch and StockLock
//
// All modifications to inventory state, including batches and locks, MUST go through
// this aggregate root's methods. External code should NEVER directly modify the
// Batches or Locks slices - use the provided methods instead:
// - IncreaseStock() - adds stock and optionally creates a batch
// - LockStock() - creates a lock and moves quantity from available to locked
// - UnlockStock() - releases a lock and moves quantity from locked to available
// - DeductStock() - consumes a lock and decreases locked quantity
// - DecreaseStock() - directly decreases available stock (for returns)
// - AdjustStock() - adjusts stock to match actual count
type InventoryItem struct {
	shared.TenantAggregateRoot
	WarehouseID       uuid.UUID
	ProductID         uuid.UUID
	AvailableQuantity decimal.Decimal // Available for sale/use
	LockedQuantity    decimal.Decimal // Reserved for pending orders
	UnitCost          decimal.Decimal // Moving weighted average cost
	MinQuantity       decimal.Decimal // Minimum stock threshold for alerts
	MaxQuantity       decimal.Decimal // Maximum stock threshold

	// Associations - loaded lazily
	Batches []StockBatch
	Locks   []StockLock
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
		// BUG-011: Defensive check for division by zero (should never happen due to quantity > 0 check above,
		// but provides explicit protection against potential edge cases or race conditions)
		if totalQuantity.IsZero() {
			return shared.NewDomainError("DIVISION_BY_ZERO", "Cannot calculate weighted average cost: total quantity is zero")
		}
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

// IncreaseStockWithCost increases the stock quantity with a pre-calculated unit cost.
// This method is used by InventoryDomainService which calculates the cost using
// an injected strategy (e.g., moving average, FIFO).
// Unlike IncreaseStock, this method does NOT recalculate the cost internally
// and does NOT emit InventoryCostChangedEvent (caller is responsible for that).
func (i *InventoryItem) IncreaseStockWithCost(quantity decimal.Decimal, newUnitCost decimal.Decimal, batch *BatchInfo) error {
	if quantity.LessThanOrEqual(decimal.Zero) {
		return shared.NewDomainError("INVALID_QUANTITY", "Quantity must be positive")
	}
	if newUnitCost.IsNegative() {
		return shared.NewDomainError("INVALID_COST", "Unit cost cannot be negative")
	}

	// Set the new unit cost (pre-calculated by domain service using strategy)
	i.UnitCost = newUnitCost

	// Increase available quantity
	i.AvailableQuantity = i.AvailableQuantity.Add(quantity)
	i.UpdatedAt = time.Now()
	i.IncrementVersion()

	// Create batch if batch info is provided
	if batch != nil {
		stockBatch := NewStockBatch(i.ID, batch.BatchNumber, batch.ProductionDate, batch.ExpiryDate, quantity, newUnitCost)
		i.Batches = append(i.Batches, *stockBatch)
	}

	// Emit stock increased event only (cost change event handled by caller)
	i.AddDomainEvent(NewStockIncreasedEvent(i, quantity, newUnitCost, batch))

	return nil
}

// DecreaseStock directly decreases available stock (without requiring a prior lock)
// This is used for operations like purchase returns where goods are shipped back to supplier
// Different from DeductStock which works with locked stock
func (i *InventoryItem) DecreaseStock(quantity decimal.Decimal, sourceType, sourceID, reason string) error {
	if quantity.LessThanOrEqual(decimal.Zero) {
		return shared.NewDomainError("INVALID_QUANTITY", "Quantity must be positive")
	}
	if i.AvailableQuantity.LessThan(quantity) {
		return shared.NewDomainError("INSUFFICIENT_STOCK", "Insufficient available stock to decrease")
	}
	if sourceType == "" || sourceID == "" {
		return shared.NewDomainError("INVALID_SOURCE", "Source type and ID are required")
	}

	// Decrease available quantity
	i.AvailableQuantity = i.AvailableQuantity.Sub(quantity)
	i.UpdatedAt = time.Now()
	i.IncrementVersion()

	// Emit event
	i.AddDomainEvent(NewStockDecreasedEvent(i, quantity, sourceType, sourceID, reason))

	// Check if below minimum threshold
	if i.MinQuantity.GreaterThan(decimal.Zero) && i.TotalQuantity().LessThan(i.MinQuantity) {
		i.AddDomainEvent(NewStockBelowThresholdEvent(i))
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

// GetExpiredLocks returns locks that have expired but are not yet released.
//
// Deprecated: Use GetExpiredLocksAt() with a reference timestamp for critical operations
// to avoid race conditions. This method captures time.Now() internally which may cause
// inconsistent results in concurrent scenarios.
//
// BUG-013: For atomic expiration checking, use GetExpiredLocksAt() instead.
func (i *InventoryItem) GetExpiredLocks() []StockLock {
	return i.GetExpiredLocksAt(time.Now())
}

// GetExpiredLocksAt returns locks that have expired relative to the reference time.
//
// This method should be used for critical business operations where atomicity matters.
// By passing a reference timestamp (e.g., captured at the start of a batch operation
// or received from the database query), you ensure consistent expiration checking
// throughout the operation, preventing race conditions.
//
// BUG-013: This method addresses the non-atomic lock expiration check issue.
func (i *InventoryItem) GetExpiredLocksAt(referenceTime time.Time) []StockLock {
	expiredLocks := make([]StockLock, 0)
	for _, lock := range i.Locks {
		if !lock.Released && !lock.Consumed && lock.IsExpiredAt(referenceTime) {
			expiredLocks = append(expiredLocks, lock)
		}
	}
	return expiredLocks
}

// ReleaseExpiredLocks releases all expired locks back to available stock.
//
// Deprecated: Use ReleaseExpiredLocksAt() with a reference timestamp for critical operations
// to avoid race conditions. This method captures time.Now() internally which may cause
// inconsistent results in concurrent scenarios.
//
// BUG-013: For atomic expiration checking, use ReleaseExpiredLocksAt() instead.
func (i *InventoryItem) ReleaseExpiredLocks() int {
	return i.ReleaseExpiredLocksAt(time.Now())
}

// ReleaseExpiredLocksAt releases all locks that have expired relative to the reference time.
//
// This method should be used for critical business operations where atomicity matters.
// By passing a reference timestamp (e.g., captured at the start of a batch operation
// or received from the database query), you ensure consistent expiration checking
// throughout the operation, preventing race conditions between check and action.
//
// BUG-013: This method addresses the non-atomic lock expiration check issue.
func (i *InventoryItem) ReleaseExpiredLocksAt(referenceTime time.Time) int {
	expiredLocks := i.GetExpiredLocksAt(referenceTime)
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
