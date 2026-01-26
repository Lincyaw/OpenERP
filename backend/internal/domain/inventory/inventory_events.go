package inventory

import (
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Aggregate type constant
const AggregateTypeInventoryItem = "InventoryItem"

// Event type constants
const (
	EventTypeStockIncreased       = "StockIncreased"
	EventTypeStockDecreased       = "StockDecreased"
	EventTypeStockLocked          = "StockLocked"
	EventTypeStockUnlocked        = "StockUnlocked"
	EventTypeStockLockExpired     = "StockLockExpired"
	EventTypeStockDeducted        = "StockDeducted"
	EventTypeStockAdjusted        = "StockAdjusted"
	EventTypeInventoryCostChanged = "InventoryCostChanged"
	EventTypeStockBelowThreshold  = "StockBelowThreshold"
)

// StockIncreasedEvent is raised when stock is increased (e.g., receiving inventory)
type StockIncreasedEvent struct {
	shared.BaseDomainEvent
	InventoryItemID uuid.UUID       `json:"inventory_item_id"`
	WarehouseID     uuid.UUID       `json:"warehouse_id"`
	ProductID       uuid.UUID       `json:"product_id"`
	Quantity        decimal.Decimal `json:"quantity"`
	UnitCost        decimal.Decimal `json:"unit_cost"`
	BatchNumber     string          `json:"batch_number,omitempty"`
}

// NewStockIncreasedEvent creates a new StockIncreasedEvent
func NewStockIncreasedEvent(item *InventoryItem, quantity, unitCost decimal.Decimal, batch *BatchInfo) *StockIncreasedEvent {
	batchNumber := ""
	if batch != nil {
		batchNumber = batch.BatchNumber
	}
	return &StockIncreasedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeStockIncreased, AggregateTypeInventoryItem, item.ID, item.TenantID),
		InventoryItemID: item.ID,
		WarehouseID:     item.WarehouseID,
		ProductID:       item.ProductID,
		Quantity:        quantity,
		UnitCost:        unitCost,
		BatchNumber:     batchNumber,
	}
}

// EventType returns the event type name
func (e *StockIncreasedEvent) EventType() string {
	return EventTypeStockIncreased
}

// StockDecreasedEvent is raised when stock is directly decreased (e.g., purchase return shipping)
// This is different from StockDeducted which deducts from locked stock
type StockDecreasedEvent struct {
	shared.BaseDomainEvent
	InventoryItemID uuid.UUID       `json:"inventory_item_id"`
	WarehouseID     uuid.UUID       `json:"warehouse_id"`
	ProductID       uuid.UUID       `json:"product_id"`
	Quantity        decimal.Decimal `json:"quantity"`
	UnitCost        decimal.Decimal `json:"unit_cost"`
	SourceType      string          `json:"source_type"`
	SourceID        string          `json:"source_id"`
	Reason          string          `json:"reason,omitempty"`
}

// NewStockDecreasedEvent creates a new StockDecreasedEvent
func NewStockDecreasedEvent(item *InventoryItem, quantity decimal.Decimal, sourceType, sourceID, reason string) *StockDecreasedEvent {
	return &StockDecreasedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeStockDecreased, AggregateTypeInventoryItem, item.ID, item.TenantID),
		InventoryItemID: item.ID,
		WarehouseID:     item.WarehouseID,
		ProductID:       item.ProductID,
		Quantity:        quantity,
		UnitCost:        item.UnitCost,
		SourceType:      sourceType,
		SourceID:        sourceID,
		Reason:          reason,
	}
}

// EventType returns the event type name
func (e *StockDecreasedEvent) EventType() string {
	return EventTypeStockDecreased
}

// StockLockedEvent is raised when stock is locked for a pending order
type StockLockedEvent struct {
	shared.BaseDomainEvent
	InventoryItemID uuid.UUID       `json:"inventory_item_id"`
	WarehouseID     uuid.UUID       `json:"warehouse_id"`
	ProductID       uuid.UUID       `json:"product_id"`
	LockID          uuid.UUID       `json:"lock_id"`
	Quantity        decimal.Decimal `json:"quantity"`
	SourceType      string          `json:"source_type"`
	SourceID        string          `json:"source_id"`
}

// NewStockLockedEvent creates a new StockLockedEvent
func NewStockLockedEvent(item *InventoryItem, quantity decimal.Decimal, lockID uuid.UUID, sourceType, sourceID string) *StockLockedEvent {
	return &StockLockedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeStockLocked, AggregateTypeInventoryItem, item.ID, item.TenantID),
		InventoryItemID: item.ID,
		WarehouseID:     item.WarehouseID,
		ProductID:       item.ProductID,
		LockID:          lockID,
		Quantity:        quantity,
		SourceType:      sourceType,
		SourceID:        sourceID,
	}
}

// EventType returns the event type name
func (e *StockLockedEvent) EventType() string {
	return EventTypeStockLocked
}

// StockUnlockedEvent is raised when locked stock is released
type StockUnlockedEvent struct {
	shared.BaseDomainEvent
	InventoryItemID uuid.UUID       `json:"inventory_item_id"`
	WarehouseID     uuid.UUID       `json:"warehouse_id"`
	ProductID       uuid.UUID       `json:"product_id"`
	LockID          uuid.UUID       `json:"lock_id"`
	Quantity        decimal.Decimal `json:"quantity"`
	SourceType      string          `json:"source_type"`
	SourceID        string          `json:"source_id"`
}

// NewStockUnlockedEvent creates a new StockUnlockedEvent
func NewStockUnlockedEvent(item *InventoryItem, quantity decimal.Decimal, lockID uuid.UUID, sourceType, sourceID string) *StockUnlockedEvent {
	return &StockUnlockedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeStockUnlocked, AggregateTypeInventoryItem, item.ID, item.TenantID),
		InventoryItemID: item.ID,
		WarehouseID:     item.WarehouseID,
		ProductID:       item.ProductID,
		LockID:          lockID,
		Quantity:        quantity,
		SourceType:      sourceType,
		SourceID:        sourceID,
	}
}

// EventType returns the event type name
func (e *StockUnlockedEvent) EventType() string {
	return EventTypeStockUnlocked
}

// StockDeductedEvent is raised when locked stock is deducted (shipped/consumed)
type StockDeductedEvent struct {
	shared.BaseDomainEvent
	InventoryItemID uuid.UUID       `json:"inventory_item_id"`
	WarehouseID     uuid.UUID       `json:"warehouse_id"`
	ProductID       uuid.UUID       `json:"product_id"`
	LockID          uuid.UUID       `json:"lock_id"`
	Quantity        decimal.Decimal `json:"quantity"`
	SourceType      string          `json:"source_type"`
	SourceID        string          `json:"source_id"`
}

// NewStockDeductedEvent creates a new StockDeductedEvent
func NewStockDeductedEvent(item *InventoryItem, quantity decimal.Decimal, lockID uuid.UUID, sourceType, sourceID string) *StockDeductedEvent {
	return &StockDeductedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeStockDeducted, AggregateTypeInventoryItem, item.ID, item.TenantID),
		InventoryItemID: item.ID,
		WarehouseID:     item.WarehouseID,
		ProductID:       item.ProductID,
		LockID:          lockID,
		Quantity:        quantity,
		SourceType:      sourceType,
		SourceID:        sourceID,
	}
}

// EventType returns the event type name
func (e *StockDeductedEvent) EventType() string {
	return EventTypeStockDeducted
}

// StockAdjustedEvent is raised when stock is adjusted (e.g., during stock taking)
type StockAdjustedEvent struct {
	shared.BaseDomainEvent
	InventoryItemID uuid.UUID       `json:"inventory_item_id"`
	WarehouseID     uuid.UUID       `json:"warehouse_id"`
	ProductID       uuid.UUID       `json:"product_id"`
	OldQuantity     decimal.Decimal `json:"old_quantity"`
	NewQuantity     decimal.Decimal `json:"new_quantity"`
	Difference      decimal.Decimal `json:"difference"`
	Reason          string          `json:"reason"`
}

// NewStockAdjustedEvent creates a new StockAdjustedEvent
func NewStockAdjustedEvent(item *InventoryItem, oldQty, newQty, diff decimal.Decimal, reason string) *StockAdjustedEvent {
	return &StockAdjustedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeStockAdjusted, AggregateTypeInventoryItem, item.ID, item.TenantID),
		InventoryItemID: item.ID,
		WarehouseID:     item.WarehouseID,
		ProductID:       item.ProductID,
		OldQuantity:     oldQty,
		NewQuantity:     newQty,
		Difference:      diff,
		Reason:          reason,
	}
}

// EventType returns the event type name
func (e *StockAdjustedEvent) EventType() string {
	return EventTypeStockAdjusted
}

// InventoryCostChangedEvent is raised when the unit cost changes
type InventoryCostChangedEvent struct {
	shared.BaseDomainEvent
	InventoryItemID uuid.UUID       `json:"inventory_item_id"`
	WarehouseID     uuid.UUID       `json:"warehouse_id"`
	ProductID       uuid.UUID       `json:"product_id"`
	OldCost         decimal.Decimal `json:"old_cost"`
	NewCost         decimal.Decimal `json:"new_cost"`
}

// NewInventoryCostChangedEvent creates a new InventoryCostChangedEvent
func NewInventoryCostChangedEvent(item *InventoryItem, oldCost, newCost valueobject.Money) *InventoryCostChangedEvent {
	return &InventoryCostChangedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeInventoryCostChanged, AggregateTypeInventoryItem, item.ID, item.TenantID),
		InventoryItemID: item.ID,
		WarehouseID:     item.WarehouseID,
		ProductID:       item.ProductID,
		OldCost:         oldCost.Amount(),
		NewCost:         newCost.Amount(),
	}
}

// EventType returns the event type name
func (e *InventoryCostChangedEvent) EventType() string {
	return EventTypeInventoryCostChanged
}

// StockBelowThresholdEvent is raised when stock falls below minimum threshold
type StockBelowThresholdEvent struct {
	shared.BaseDomainEvent
	InventoryItemID   uuid.UUID       `json:"inventory_item_id"`
	WarehouseID       uuid.UUID       `json:"warehouse_id"`
	ProductID         uuid.UUID       `json:"product_id"`
	CurrentQuantity   decimal.Decimal `json:"current_quantity"`
	MinimumQuantity   decimal.Decimal `json:"minimum_quantity"`
	AvailableQuantity decimal.Decimal `json:"available_quantity"`
	LockedQuantity    decimal.Decimal `json:"locked_quantity"`
}

// NewStockBelowThresholdEvent creates a new StockBelowThresholdEvent
func NewStockBelowThresholdEvent(item *InventoryItem) *StockBelowThresholdEvent {
	return &StockBelowThresholdEvent{
		BaseDomainEvent:   shared.NewBaseDomainEvent(EventTypeStockBelowThreshold, AggregateTypeInventoryItem, item.ID, item.TenantID),
		InventoryItemID:   item.ID,
		WarehouseID:       item.WarehouseID,
		ProductID:         item.ProductID,
		CurrentQuantity:   item.TotalQuantity(),
		MinimumQuantity:   item.MinQuantity,
		AvailableQuantity: item.AvailableQuantity,
		LockedQuantity:    item.LockedQuantity,
	}
}

// EventType returns the event type name
func (e *StockBelowThresholdEvent) EventType() string {
	return EventTypeStockBelowThreshold
}

// StockLockExpiredEvent is raised when a stock lock expires and is automatically released
type StockLockExpiredEvent struct {
	shared.BaseDomainEvent
	InventoryItemID uuid.UUID       `json:"inventory_item_id"`
	WarehouseID     uuid.UUID       `json:"warehouse_id"`
	ProductID       uuid.UUID       `json:"product_id"`
	LockID          uuid.UUID       `json:"lock_id"`
	Quantity        decimal.Decimal `json:"quantity"`
	SourceType      string          `json:"source_type"` // e.g., "sales_order"
	SourceID        string          `json:"source_id"`   // ID of the source document
}

// NewStockLockExpiredEvent creates a new StockLockExpiredEvent
func NewStockLockExpiredEvent(tenantID, inventoryItemID, warehouseID, productID, lockID uuid.UUID, quantity decimal.Decimal, sourceType, sourceID string) *StockLockExpiredEvent {
	return &StockLockExpiredEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeStockLockExpired, AggregateTypeInventoryItem, inventoryItemID, tenantID),
		InventoryItemID: inventoryItemID,
		WarehouseID:     warehouseID,
		ProductID:       productID,
		LockID:          lockID,
		Quantity:        quantity,
		SourceType:      sourceType,
		SourceID:        sourceID,
	}
}

// EventType returns the event type name
func (e *StockLockExpiredEvent) EventType() string {
	return EventTypeStockLockExpired
}
