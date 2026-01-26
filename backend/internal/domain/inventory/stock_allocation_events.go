package inventory

import (
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Allocation event type constants
const (
	// EventTypeStockAllocationCompleted is raised when all items in an allocation request
	// are successfully locked.
	EventTypeStockAllocationCompleted = "StockAllocationCompleted"

	// EventTypeStockAllocationPartial is raised when some items fail to allocate,
	// but compensation has not yet been applied.
	EventTypeStockAllocationPartial = "StockAllocationPartial"

	// EventTypeStockAllocationFailed is raised when allocation fails completely
	// (no items were successfully locked).
	EventTypeStockAllocationFailed = "StockAllocationFailed"

	// EventTypeStockAllocationCompensated is raised when compensation (rollback)
	// has been applied to successful locks after a partial failure.
	EventTypeStockAllocationCompensated = "StockAllocationCompensated"

	// EventTypeStockAllocationReleased is raised when allocated stock is released
	// (e.g., order cancelled).
	EventTypeStockAllocationReleased = "StockAllocationReleased"
)

// AllocationItemInfo contains minimal information about an allocation item for events
type AllocationItemInfo struct {
	InventoryItemID uuid.UUID       `json:"inventory_item_id"`
	WarehouseID     uuid.UUID       `json:"warehouse_id"`
	ProductID       uuid.UUID       `json:"product_id"`
	Quantity        decimal.Decimal `json:"quantity"`
	LockID          uuid.UUID       `json:"lock_id,omitempty"`
	Success         bool            `json:"success"`
	ErrorMessage    string          `json:"error_message,omitempty"`
}

// CompensationInfo contains minimal information about a compensation operation
type CompensationInfo struct {
	InventoryItemID uuid.UUID `json:"inventory_item_id"`
	LockID          uuid.UUID `json:"lock_id"`
	Success         bool      `json:"success"`
	ErrorMessage    string    `json:"error_message,omitempty"`
}

// ReleaseInfo contains minimal information about a release operation
type ReleaseInfo struct {
	InventoryItemID uuid.UUID       `json:"inventory_item_id"`
	LockID          uuid.UUID       `json:"lock_id"`
	Quantity        decimal.Decimal `json:"quantity"`
	Success         bool            `json:"success"`
	ErrorMessage    string          `json:"error_message,omitempty"`
}

// StockAllocationCompletedEvent is raised when all items are successfully allocated
type StockAllocationCompletedEvent struct {
	shared.BaseDomainEvent
	CorrelationID uuid.UUID            `json:"correlation_id"`
	SourceType    string               `json:"source_type"`
	SourceID      string               `json:"source_id"`
	Items         []AllocationItemInfo `json:"items"`
	TotalQuantity decimal.Decimal      `json:"total_quantity"`
	ItemCount     int                  `json:"item_count"`
}

// NewStockAllocationCompletedEvent creates a new StockAllocationCompletedEvent
func NewStockAllocationCompletedEvent(
	correlationID uuid.UUID,
	sourceType, sourceID string,
	items []AllocationItemResult,
) *StockAllocationCompletedEvent {
	var tenantID uuid.UUID
	itemInfos := make([]AllocationItemInfo, len(items))
	totalQty := decimal.Zero

	for i, item := range items {
		itemInfos[i] = AllocationItemInfo{
			InventoryItemID: item.InventoryItemID,
			WarehouseID:     item.WarehouseID,
			ProductID:       item.ProductID,
			Quantity:        item.Quantity,
			LockID:          item.LockID,
			Success:         item.Success,
			ErrorMessage:    item.ErrorMessage,
		}
		totalQty = totalQty.Add(item.Quantity)
	}

	return &StockAllocationCompletedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypeStockAllocationCompleted,
			AggregateTypeInventoryItem,
			correlationID, // Use correlation ID as aggregate ID for this cross-aggregate event
			tenantID,
		),
		CorrelationID: correlationID,
		SourceType:    sourceType,
		SourceID:      sourceID,
		Items:         itemInfos,
		TotalQuantity: totalQty,
		ItemCount:     len(items),
	}
}

// EventType returns the event type name
func (e *StockAllocationCompletedEvent) EventType() string {
	return EventTypeStockAllocationCompleted
}

// StockAllocationPartialEvent is raised when some items fail to allocate
type StockAllocationPartialEvent struct {
	shared.BaseDomainEvent
	CorrelationID   uuid.UUID            `json:"correlation_id"`
	SourceType      string               `json:"source_type"`
	SourceID        string               `json:"source_id"`
	Items           []AllocationItemInfo `json:"items"`
	FailedIndices   []int                `json:"failed_indices"`
	SuccessCount    int                  `json:"success_count"`
	FailureCount    int                  `json:"failure_count"`
	TotalQuantity   decimal.Decimal      `json:"total_quantity"`
	LockedQuantity  decimal.Decimal      `json:"locked_quantity"`
	PendingQuantity decimal.Decimal      `json:"pending_quantity"` // Quantity that couldn't be locked
}

// NewStockAllocationPartialEvent creates a new StockAllocationPartialEvent
func NewStockAllocationPartialEvent(
	correlationID uuid.UUID,
	sourceType, sourceID string,
	items []AllocationItemResult,
	failedIndices []int,
) *StockAllocationPartialEvent {
	var tenantID uuid.UUID
	itemInfos := make([]AllocationItemInfo, len(items))
	totalQty := decimal.Zero
	lockedQty := decimal.Zero
	pendingQty := decimal.Zero
	successCount := 0

	for i, item := range items {
		itemInfos[i] = AllocationItemInfo{
			InventoryItemID: item.InventoryItemID,
			WarehouseID:     item.WarehouseID,
			ProductID:       item.ProductID,
			Quantity:        item.Quantity,
			LockID:          item.LockID,
			Success:         item.Success,
			ErrorMessage:    item.ErrorMessage,
		}
		totalQty = totalQty.Add(item.Quantity)
		if item.Success {
			lockedQty = lockedQty.Add(item.Quantity)
			successCount++
		} else {
			pendingQty = pendingQty.Add(item.Quantity)
		}
	}

	return &StockAllocationPartialEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypeStockAllocationPartial,
			AggregateTypeInventoryItem,
			correlationID,
			tenantID,
		),
		CorrelationID:   correlationID,
		SourceType:      sourceType,
		SourceID:        sourceID,
		Items:           itemInfos,
		FailedIndices:   failedIndices,
		SuccessCount:    successCount,
		FailureCount:    len(failedIndices),
		TotalQuantity:   totalQty,
		LockedQuantity:  lockedQty,
		PendingQuantity: pendingQty,
	}
}

// EventType returns the event type name
func (e *StockAllocationPartialEvent) EventType() string {
	return EventTypeStockAllocationPartial
}

// StockAllocationFailedEvent is raised when allocation completely fails
type StockAllocationFailedEvent struct {
	shared.BaseDomainEvent
	CorrelationID uuid.UUID            `json:"correlation_id"`
	SourceType    string               `json:"source_type"`
	SourceID      string               `json:"source_id"`
	Items         []AllocationItemInfo `json:"items"`
	FailedIndices []int                `json:"failed_indices"`
	TotalQuantity decimal.Decimal      `json:"total_quantity"`
	FailureCount  int                  `json:"failure_count"`
}

// NewStockAllocationFailedEvent creates a new StockAllocationFailedEvent
func NewStockAllocationFailedEvent(
	correlationID uuid.UUID,
	sourceType, sourceID string,
	items []AllocationItemResult,
	failedIndices []int,
) *StockAllocationFailedEvent {
	var tenantID uuid.UUID
	itemInfos := make([]AllocationItemInfo, len(items))
	totalQty := decimal.Zero

	for i, item := range items {
		itemInfos[i] = AllocationItemInfo{
			InventoryItemID: item.InventoryItemID,
			WarehouseID:     item.WarehouseID,
			ProductID:       item.ProductID,
			Quantity:        item.Quantity,
			LockID:          item.LockID,
			Success:         item.Success,
			ErrorMessage:    item.ErrorMessage,
		}
		totalQty = totalQty.Add(item.Quantity)
	}

	return &StockAllocationFailedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypeStockAllocationFailed,
			AggregateTypeInventoryItem,
			correlationID,
			tenantID,
		),
		CorrelationID: correlationID,
		SourceType:    sourceType,
		SourceID:      sourceID,
		Items:         itemInfos,
		FailedIndices: failedIndices,
		TotalQuantity: totalQty,
		FailureCount:  len(failedIndices),
	}
}

// EventType returns the event type name
func (e *StockAllocationFailedEvent) EventType() string {
	return EventTypeStockAllocationFailed
}

// StockAllocationCompensatedEvent is raised when compensation is applied after partial failure
type StockAllocationCompensatedEvent struct {
	shared.BaseDomainEvent
	CorrelationID    uuid.UUID          `json:"correlation_id"`
	SourceType       string             `json:"source_type"`
	SourceID         string             `json:"source_id"`
	Compensations    []CompensationInfo `json:"compensations"`
	TotalCompensated int                `json:"total_compensated"`
	SuccessCount     int                `json:"success_count"`
	FailureCount     int                `json:"failure_count"`
}

// NewStockAllocationCompensatedEvent creates a new StockAllocationCompensatedEvent
func NewStockAllocationCompensatedEvent(
	correlationID uuid.UUID,
	sourceType, sourceID string,
	results []CompensationResult,
) *StockAllocationCompensatedEvent {
	var tenantID uuid.UUID
	compensations := make([]CompensationInfo, len(results))
	successCount := 0
	failureCount := 0

	for i, r := range results {
		compensations[i] = CompensationInfo{
			InventoryItemID: r.InventoryItemID,
			LockID:          r.LockID,
			Success:         r.Success,
			ErrorMessage:    r.ErrorMessage,
		}
		if r.Success {
			successCount++
		} else {
			failureCount++
		}
	}

	return &StockAllocationCompensatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypeStockAllocationCompensated,
			AggregateTypeInventoryItem,
			correlationID,
			tenantID,
		),
		CorrelationID:    correlationID,
		SourceType:       sourceType,
		SourceID:         sourceID,
		Compensations:    compensations,
		TotalCompensated: len(results),
		SuccessCount:     successCount,
		FailureCount:     failureCount,
	}
}

// EventType returns the event type name
func (e *StockAllocationCompensatedEvent) EventType() string {
	return EventTypeStockAllocationCompensated
}

// StockAllocationReleasedEvent is raised when allocated stock is released
type StockAllocationReleasedEvent struct {
	shared.BaseDomainEvent
	CorrelationID uuid.UUID       `json:"correlation_id"`
	SourceType    string          `json:"source_type"`
	SourceID      string          `json:"source_id"`
	Releases      []ReleaseInfo   `json:"releases"`
	TotalReleased decimal.Decimal `json:"total_released"`
	ReleaseCount  int             `json:"release_count"`
	SuccessCount  int             `json:"success_count"`
	FailureCount  int             `json:"failure_count"`
}

// NewStockAllocationReleasedEvent creates a new StockAllocationReleasedEvent
func NewStockAllocationReleasedEvent(
	correlationID uuid.UUID,
	sourceType, sourceID string,
	results []ReleaseItemResult,
) *StockAllocationReleasedEvent {
	var tenantID uuid.UUID
	releases := make([]ReleaseInfo, len(results))
	totalReleased := decimal.Zero
	successCount := 0
	failureCount := 0

	for i, r := range results {
		releases[i] = ReleaseInfo{
			InventoryItemID: r.InventoryItemID,
			LockID:          r.LockID,
			Quantity:        r.Quantity,
			Success:         r.Success,
			ErrorMessage:    r.ErrorMessage,
		}
		if r.Success {
			successCount++
			totalReleased = totalReleased.Add(r.Quantity)
		} else {
			failureCount++
		}
	}

	return &StockAllocationReleasedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypeStockAllocationReleased,
			AggregateTypeInventoryItem,
			correlationID,
			tenantID,
		),
		CorrelationID: correlationID,
		SourceType:    sourceType,
		SourceID:      sourceID,
		Releases:      releases,
		TotalReleased: totalReleased,
		ReleaseCount:  len(results),
		SuccessCount:  successCount,
		FailureCount:  failureCount,
	}
}

// EventType returns the event type name
func (e *StockAllocationReleasedEvent) EventType() string {
	return EventTypeStockAllocationReleased
}
