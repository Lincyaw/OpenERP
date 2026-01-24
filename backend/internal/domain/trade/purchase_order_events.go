package trade

import (
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Aggregate type constant
const AggregateTypePurchaseOrder = "PurchaseOrder"

// Event type constants
const (
	EventTypePurchaseOrderCreated   = "PurchaseOrderCreated"
	EventTypePurchaseOrderConfirmed = "PurchaseOrderConfirmed"
	EventTypePurchaseOrderReceived  = "PurchaseOrderReceived"
	EventTypePurchaseOrderCompleted = "PurchaseOrderCompleted"
	EventTypePurchaseOrderCancelled = "PurchaseOrderCancelled"
)

// PurchaseOrderCreatedEvent is raised when a new purchase order is created
type PurchaseOrderCreatedEvent struct {
	shared.BaseDomainEvent
	OrderID      uuid.UUID `json:"order_id"`
	OrderNumber  string    `json:"order_number"`
	SupplierID   uuid.UUID `json:"supplier_id"`
	SupplierName string    `json:"supplier_name"`
}

// NewPurchaseOrderCreatedEvent creates a new PurchaseOrderCreatedEvent
func NewPurchaseOrderCreatedEvent(order *PurchaseOrder) *PurchaseOrderCreatedEvent {
	return &PurchaseOrderCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypePurchaseOrderCreated, AggregateTypePurchaseOrder, order.ID, order.TenantID),
		OrderID:         order.ID,
		OrderNumber:     order.OrderNumber,
		SupplierID:      order.SupplierID,
		SupplierName:    order.SupplierName,
	}
}

// EventType returns the event type name
func (e *PurchaseOrderCreatedEvent) EventType() string {
	return EventTypePurchaseOrderCreated
}

// PurchaseOrderItemInfo represents item information for events
type PurchaseOrderItemInfo struct {
	ItemID           uuid.UUID       `json:"item_id"`
	ProductID        uuid.UUID       `json:"product_id"`
	ProductName      string          `json:"product_name"`
	ProductCode      string          `json:"product_code"`
	OrderedQuantity  decimal.Decimal `json:"ordered_quantity"`
	ReceivedQuantity decimal.Decimal `json:"received_quantity"`
	UnitCost         decimal.Decimal `json:"unit_cost"`
	Amount           decimal.Decimal `json:"amount"`
	Unit             string          `json:"unit"`
}

// PurchaseOrderConfirmedEvent is raised when a purchase order is confirmed
type PurchaseOrderConfirmedEvent struct {
	shared.BaseDomainEvent
	OrderID       uuid.UUID               `json:"order_id"`
	OrderNumber   string                  `json:"order_number"`
	SupplierID    uuid.UUID               `json:"supplier_id"`
	SupplierName  string                  `json:"supplier_name"`
	WarehouseID   *uuid.UUID              `json:"warehouse_id,omitempty"`
	Items         []PurchaseOrderItemInfo `json:"items"`
	TotalAmount   decimal.Decimal         `json:"total_amount"`
	PayableAmount decimal.Decimal         `json:"payable_amount"`
}

// NewPurchaseOrderConfirmedEvent creates a new PurchaseOrderConfirmedEvent
func NewPurchaseOrderConfirmedEvent(order *PurchaseOrder) *PurchaseOrderConfirmedEvent {
	items := make([]PurchaseOrderItemInfo, len(order.Items))
	for i, item := range order.Items {
		items[i] = PurchaseOrderItemInfo{
			ItemID:           item.ID,
			ProductID:        item.ProductID,
			ProductName:      item.ProductName,
			ProductCode:      item.ProductCode,
			OrderedQuantity:  item.OrderedQuantity,
			ReceivedQuantity: item.ReceivedQuantity,
			UnitCost:         item.UnitCost,
			Amount:           item.Amount,
			Unit:             item.Unit,
		}
	}

	return &PurchaseOrderConfirmedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypePurchaseOrderConfirmed, AggregateTypePurchaseOrder, order.ID, order.TenantID),
		OrderID:         order.ID,
		OrderNumber:     order.OrderNumber,
		SupplierID:      order.SupplierID,
		SupplierName:    order.SupplierName,
		WarehouseID:     order.WarehouseID,
		Items:           items,
		TotalAmount:     order.TotalAmount,
		PayableAmount:   order.PayableAmount,
	}
}

// EventType returns the event type name
func (e *PurchaseOrderConfirmedEvent) EventType() string {
	return EventTypePurchaseOrderConfirmed
}

// ReceivedItemInfo represents information about a received item in an event
type ReceivedItemInfo struct {
	ItemID      uuid.UUID       `json:"item_id"`
	ProductID   uuid.UUID       `json:"product_id"`
	ProductName string          `json:"product_name"`
	ProductCode string          `json:"product_code"`
	Quantity    decimal.Decimal `json:"quantity"`  // Quantity received in this operation
	UnitCost    decimal.Decimal `json:"unit_cost"` // Cost per unit
	Unit        string          `json:"unit"`      // Unit of measure
	BatchNumber string          `json:"batch_number,omitempty"`
	ExpiryDate  *time.Time      `json:"expiry_date,omitempty"`
}

// PurchaseOrderReceivedEvent is raised when goods are received for a purchase order
// This event triggers stock increase in the inventory context and may create accounts payable
type PurchaseOrderReceivedEvent struct {
	shared.BaseDomainEvent
	OrderID         uuid.UUID          `json:"order_id"`
	OrderNumber     string             `json:"order_number"`
	SupplierID      uuid.UUID          `json:"supplier_id"`
	SupplierName    string             `json:"supplier_name"`
	WarehouseID     uuid.UUID          `json:"warehouse_id"`
	ReceivedItems   []ReceivedItemInfo `json:"received_items"`
	TotalAmount     decimal.Decimal    `json:"total_amount"`
	PayableAmount   decimal.Decimal    `json:"payable_amount"`
	IsFullyReceived bool               `json:"is_fully_received"` // True if this completes the order
}

// NewPurchaseOrderReceivedEvent creates a new PurchaseOrderReceivedEvent
func NewPurchaseOrderReceivedEvent(order *PurchaseOrder, receivedItems []ReceivedItemInfo) *PurchaseOrderReceivedEvent {
	warehouseID := uuid.Nil
	if order.WarehouseID != nil {
		warehouseID = *order.WarehouseID
	}

	return &PurchaseOrderReceivedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypePurchaseOrderReceived, AggregateTypePurchaseOrder, order.ID, order.TenantID),
		OrderID:         order.ID,
		OrderNumber:     order.OrderNumber,
		SupplierID:      order.SupplierID,
		SupplierName:    order.SupplierName,
		WarehouseID:     warehouseID,
		ReceivedItems:   receivedItems,
		TotalAmount:     order.TotalAmount,
		PayableAmount:   order.PayableAmount,
		IsFullyReceived: order.IsCompleted(),
	}
}

// EventType returns the event type name
func (e *PurchaseOrderReceivedEvent) EventType() string {
	return EventTypePurchaseOrderReceived
}

// PurchaseOrderCompletedEvent is raised when a purchase order is completed
// (all items have been fully received)
type PurchaseOrderCompletedEvent struct {
	shared.BaseDomainEvent
	OrderID       uuid.UUID       `json:"order_id"`
	OrderNumber   string          `json:"order_number"`
	SupplierID    uuid.UUID       `json:"supplier_id"`
	TotalAmount   decimal.Decimal `json:"total_amount"`
	PayableAmount decimal.Decimal `json:"payable_amount"`
}

// NewPurchaseOrderCompletedEvent creates a new PurchaseOrderCompletedEvent
func NewPurchaseOrderCompletedEvent(order *PurchaseOrder) *PurchaseOrderCompletedEvent {
	return &PurchaseOrderCompletedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypePurchaseOrderCompleted, AggregateTypePurchaseOrder, order.ID, order.TenantID),
		OrderID:         order.ID,
		OrderNumber:     order.OrderNumber,
		SupplierID:      order.SupplierID,
		TotalAmount:     order.TotalAmount,
		PayableAmount:   order.PayableAmount,
	}
}

// EventType returns the event type name
func (e *PurchaseOrderCompletedEvent) EventType() string {
	return EventTypePurchaseOrderCompleted
}

// PurchaseOrderCancelledEvent is raised when a purchase order is cancelled
type PurchaseOrderCancelledEvent struct {
	shared.BaseDomainEvent
	OrderID      uuid.UUID               `json:"order_id"`
	OrderNumber  string                  `json:"order_number"`
	SupplierID   uuid.UUID               `json:"supplier_id"`
	WarehouseID  *uuid.UUID              `json:"warehouse_id,omitempty"`
	Items        []PurchaseOrderItemInfo `json:"items"`
	CancelReason string                  `json:"cancel_reason"`
	WasConfirmed bool                    `json:"was_confirmed"` // If true, supplier may need to be notified
}

// NewPurchaseOrderCancelledEvent creates a new PurchaseOrderCancelledEvent
func NewPurchaseOrderCancelledEvent(order *PurchaseOrder, wasConfirmed bool) *PurchaseOrderCancelledEvent {
	items := make([]PurchaseOrderItemInfo, len(order.Items))
	for i, item := range order.Items {
		items[i] = PurchaseOrderItemInfo{
			ItemID:           item.ID,
			ProductID:        item.ProductID,
			ProductName:      item.ProductName,
			ProductCode:      item.ProductCode,
			OrderedQuantity:  item.OrderedQuantity,
			ReceivedQuantity: item.ReceivedQuantity,
			UnitCost:         item.UnitCost,
			Amount:           item.Amount,
			Unit:             item.Unit,
		}
	}

	return &PurchaseOrderCancelledEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypePurchaseOrderCancelled, AggregateTypePurchaseOrder, order.ID, order.TenantID),
		OrderID:         order.ID,
		OrderNumber:     order.OrderNumber,
		SupplierID:      order.SupplierID,
		WarehouseID:     order.WarehouseID,
		Items:           items,
		CancelReason:    order.CancelReason,
		WasConfirmed:    wasConfirmed,
	}
}

// EventType returns the event type name
func (e *PurchaseOrderCancelledEvent) EventType() string {
	return EventTypePurchaseOrderCancelled
}
