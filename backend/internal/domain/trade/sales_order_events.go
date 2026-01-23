package trade

import (
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Aggregate type constant
const AggregateTypeSalesOrder = "SalesOrder"

// Event type constants
const (
	EventTypeSalesOrderCreated   = "SalesOrderCreated"
	EventTypeSalesOrderConfirmed = "SalesOrderConfirmed"
	EventTypeSalesOrderShipped   = "SalesOrderShipped"
	EventTypeSalesOrderCompleted = "SalesOrderCompleted"
	EventTypeSalesOrderCancelled = "SalesOrderCancelled"
)

// SalesOrderCreatedEvent is raised when a new sales order is created
type SalesOrderCreatedEvent struct {
	shared.BaseDomainEvent
	OrderID      uuid.UUID `json:"order_id"`
	OrderNumber  string    `json:"order_number"`
	CustomerID   uuid.UUID `json:"customer_id"`
	CustomerName string    `json:"customer_name"`
}

// NewSalesOrderCreatedEvent creates a new SalesOrderCreatedEvent
func NewSalesOrderCreatedEvent(order *SalesOrder) *SalesOrderCreatedEvent {
	return &SalesOrderCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeSalesOrderCreated, AggregateTypeSalesOrder, order.ID, order.TenantID),
		OrderID:         order.ID,
		OrderNumber:     order.OrderNumber,
		CustomerID:      order.CustomerID,
		CustomerName:    order.CustomerName,
	}
}

// EventType returns the event type name
func (e *SalesOrderCreatedEvent) EventType() string {
	return EventTypeSalesOrderCreated
}

// SalesOrderItemInfo represents item information for events
type SalesOrderItemInfo struct {
	ItemID      uuid.UUID       `json:"item_id"`
	ProductID   uuid.UUID       `json:"product_id"`
	ProductName string          `json:"product_name"`
	ProductCode string          `json:"product_code"`
	Quantity    decimal.Decimal `json:"quantity"`
	UnitPrice   decimal.Decimal `json:"unit_price"`
	Amount      decimal.Decimal `json:"amount"`
	Unit        string          `json:"unit"`
}

// SalesOrderConfirmedEvent is raised when a sales order is confirmed
// This event triggers stock locking in the inventory context
type SalesOrderConfirmedEvent struct {
	shared.BaseDomainEvent
	OrderID       uuid.UUID            `json:"order_id"`
	OrderNumber   string               `json:"order_number"`
	CustomerID    uuid.UUID            `json:"customer_id"`
	CustomerName  string               `json:"customer_name"`
	WarehouseID   *uuid.UUID           `json:"warehouse_id,omitempty"`
	Items         []SalesOrderItemInfo `json:"items"`
	TotalAmount   decimal.Decimal      `json:"total_amount"`
	PayableAmount decimal.Decimal      `json:"payable_amount"`
}

// NewSalesOrderConfirmedEvent creates a new SalesOrderConfirmedEvent
func NewSalesOrderConfirmedEvent(order *SalesOrder) *SalesOrderConfirmedEvent {
	items := make([]SalesOrderItemInfo, len(order.Items))
	for i, item := range order.Items {
		items[i] = SalesOrderItemInfo{
			ItemID:      item.ID,
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			ProductCode: item.ProductCode,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			Amount:      item.Amount,
			Unit:        item.Unit,
		}
	}

	return &SalesOrderConfirmedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeSalesOrderConfirmed, AggregateTypeSalesOrder, order.ID, order.TenantID),
		OrderID:         order.ID,
		OrderNumber:     order.OrderNumber,
		CustomerID:      order.CustomerID,
		CustomerName:    order.CustomerName,
		WarehouseID:     order.WarehouseID,
		Items:           items,
		TotalAmount:     order.TotalAmount,
		PayableAmount:   order.PayableAmount,
	}
}

// EventType returns the event type name
func (e *SalesOrderConfirmedEvent) EventType() string {
	return EventTypeSalesOrderConfirmed
}

// SalesOrderShippedEvent is raised when a sales order is shipped
// This event triggers stock deduction and accounts receivable creation
type SalesOrderShippedEvent struct {
	shared.BaseDomainEvent
	OrderID       uuid.UUID            `json:"order_id"`
	OrderNumber   string               `json:"order_number"`
	CustomerID    uuid.UUID            `json:"customer_id"`
	CustomerName  string               `json:"customer_name"`
	WarehouseID   uuid.UUID            `json:"warehouse_id"`
	Items         []SalesOrderItemInfo `json:"items"`
	TotalAmount   decimal.Decimal      `json:"total_amount"`
	PayableAmount decimal.Decimal      `json:"payable_amount"`
}

// NewSalesOrderShippedEvent creates a new SalesOrderShippedEvent
func NewSalesOrderShippedEvent(order *SalesOrder) *SalesOrderShippedEvent {
	items := make([]SalesOrderItemInfo, len(order.Items))
	for i, item := range order.Items {
		items[i] = SalesOrderItemInfo{
			ItemID:      item.ID,
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			ProductCode: item.ProductCode,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			Amount:      item.Amount,
			Unit:        item.Unit,
		}
	}

	warehouseID := uuid.Nil
	if order.WarehouseID != nil {
		warehouseID = *order.WarehouseID
	}

	return &SalesOrderShippedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeSalesOrderShipped, AggregateTypeSalesOrder, order.ID, order.TenantID),
		OrderID:         order.ID,
		OrderNumber:     order.OrderNumber,
		CustomerID:      order.CustomerID,
		CustomerName:    order.CustomerName,
		WarehouseID:     warehouseID,
		Items:           items,
		TotalAmount:     order.TotalAmount,
		PayableAmount:   order.PayableAmount,
	}
}

// EventType returns the event type name
func (e *SalesOrderShippedEvent) EventType() string {
	return EventTypeSalesOrderShipped
}

// SalesOrderCompletedEvent is raised when a sales order is completed
type SalesOrderCompletedEvent struct {
	shared.BaseDomainEvent
	OrderID       uuid.UUID       `json:"order_id"`
	OrderNumber   string          `json:"order_number"`
	CustomerID    uuid.UUID       `json:"customer_id"`
	TotalAmount   decimal.Decimal `json:"total_amount"`
	PayableAmount decimal.Decimal `json:"payable_amount"`
}

// NewSalesOrderCompletedEvent creates a new SalesOrderCompletedEvent
func NewSalesOrderCompletedEvent(order *SalesOrder) *SalesOrderCompletedEvent {
	return &SalesOrderCompletedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeSalesOrderCompleted, AggregateTypeSalesOrder, order.ID, order.TenantID),
		OrderID:         order.ID,
		OrderNumber:     order.OrderNumber,
		CustomerID:      order.CustomerID,
		TotalAmount:     order.TotalAmount,
		PayableAmount:   order.PayableAmount,
	}
}

// EventType returns the event type name
func (e *SalesOrderCompletedEvent) EventType() string {
	return EventTypeSalesOrderCompleted
}

// SalesOrderCancelledEvent is raised when a sales order is cancelled
// If WasConfirmed is true, stock locks should be released
type SalesOrderCancelledEvent struct {
	shared.BaseDomainEvent
	OrderID      uuid.UUID            `json:"order_id"`
	OrderNumber  string               `json:"order_number"`
	CustomerID   uuid.UUID            `json:"customer_id"`
	WarehouseID  *uuid.UUID           `json:"warehouse_id,omitempty"`
	Items        []SalesOrderItemInfo `json:"items"`
	CancelReason string               `json:"cancel_reason"`
	WasConfirmed bool                 `json:"was_confirmed"` // If true, stock locks need to be released
}

// NewSalesOrderCancelledEvent creates a new SalesOrderCancelledEvent
func NewSalesOrderCancelledEvent(order *SalesOrder, wasConfirmed bool) *SalesOrderCancelledEvent {
	items := make([]SalesOrderItemInfo, len(order.Items))
	for i, item := range order.Items {
		items[i] = SalesOrderItemInfo{
			ItemID:      item.ID,
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			ProductCode: item.ProductCode,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			Amount:      item.Amount,
			Unit:        item.Unit,
		}
	}

	return &SalesOrderCancelledEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeSalesOrderCancelled, AggregateTypeSalesOrder, order.ID, order.TenantID),
		OrderID:         order.ID,
		OrderNumber:     order.OrderNumber,
		CustomerID:      order.CustomerID,
		WarehouseID:     order.WarehouseID,
		Items:           items,
		CancelReason:    order.CancelReason,
		WasConfirmed:    wasConfirmed,
	}
}

// EventType returns the event type name
func (e *SalesOrderCancelledEvent) EventType() string {
	return EventTypeSalesOrderCancelled
}
