package trade

import (
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Aggregate type constant for SalesReturn
const AggregateTypeSalesReturn = "SalesReturn"

// Event type constants for SalesReturn
const (
	EventTypeSalesReturnCreated   = "SalesReturnCreated"
	EventTypeSalesReturnSubmitted = "SalesReturnSubmitted"
	EventTypeSalesReturnApproved  = "SalesReturnApproved"
	EventTypeSalesReturnReceiving = "SalesReturnReceiving"
	EventTypeSalesReturnRejected  = "SalesReturnRejected"
	EventTypeSalesReturnCompleted = "SalesReturnCompleted"
	EventTypeSalesReturnCancelled = "SalesReturnCancelled"
)

// SalesReturnItemInfo represents item information for events
type SalesReturnItemInfo struct {
	ItemID           uuid.UUID       `json:"item_id"`
	SalesOrderItemID uuid.UUID       `json:"sales_order_item_id"`
	ProductID        uuid.UUID       `json:"product_id"`
	ProductName      string          `json:"product_name"`
	ProductCode      string          `json:"product_code"`
	ReturnQuantity   decimal.Decimal `json:"return_quantity"`
	UnitPrice        decimal.Decimal `json:"unit_price"`
	RefundAmount     decimal.Decimal `json:"refund_amount"`
	Unit             string          `json:"unit"`
}

// SalesReturnCreatedEvent is raised when a new sales return is created
type SalesReturnCreatedEvent struct {
	shared.BaseDomainEvent
	ReturnID         uuid.UUID `json:"return_id"`
	ReturnNumber     string    `json:"return_number"`
	SalesOrderID     uuid.UUID `json:"sales_order_id"`
	SalesOrderNumber string    `json:"sales_order_number"`
	CustomerID       uuid.UUID `json:"customer_id"`
	CustomerName     string    `json:"customer_name"`
}

// NewSalesReturnCreatedEvent creates a new SalesReturnCreatedEvent
func NewSalesReturnCreatedEvent(sr *SalesReturn) *SalesReturnCreatedEvent {
	return &SalesReturnCreatedEvent{
		BaseDomainEvent:  shared.NewBaseDomainEvent(EventTypeSalesReturnCreated, AggregateTypeSalesReturn, sr.ID, sr.TenantID),
		ReturnID:         sr.ID,
		ReturnNumber:     sr.ReturnNumber,
		SalesOrderID:     sr.SalesOrderID,
		SalesOrderNumber: sr.SalesOrderNumber,
		CustomerID:       sr.CustomerID,
		CustomerName:     sr.CustomerName,
	}
}

// EventType returns the event type name
func (e *SalesReturnCreatedEvent) EventType() string {
	return EventTypeSalesReturnCreated
}

// SalesReturnSubmittedEvent is raised when a sales return is submitted for approval
type SalesReturnSubmittedEvent struct {
	shared.BaseDomainEvent
	ReturnID         uuid.UUID             `json:"return_id"`
	ReturnNumber     string                `json:"return_number"`
	SalesOrderID     uuid.UUID             `json:"sales_order_id"`
	SalesOrderNumber string                `json:"sales_order_number"`
	CustomerID       uuid.UUID             `json:"customer_id"`
	CustomerName     string                `json:"customer_name"`
	Items            []SalesReturnItemInfo `json:"items"`
	TotalRefund      decimal.Decimal       `json:"total_refund"`
	Reason           string                `json:"reason"`
}

// NewSalesReturnSubmittedEvent creates a new SalesReturnSubmittedEvent
func NewSalesReturnSubmittedEvent(sr *SalesReturn) *SalesReturnSubmittedEvent {
	items := make([]SalesReturnItemInfo, len(sr.Items))
	for i, item := range sr.Items {
		items[i] = SalesReturnItemInfo{
			ItemID:           item.ID,
			SalesOrderItemID: item.SalesOrderItemID,
			ProductID:        item.ProductID,
			ProductName:      item.ProductName,
			ProductCode:      item.ProductCode,
			ReturnQuantity:   item.ReturnQuantity,
			UnitPrice:        item.UnitPrice,
			RefundAmount:     item.RefundAmount,
			Unit:             item.Unit,
		}
	}

	return &SalesReturnSubmittedEvent{
		BaseDomainEvent:  shared.NewBaseDomainEvent(EventTypeSalesReturnSubmitted, AggregateTypeSalesReturn, sr.ID, sr.TenantID),
		ReturnID:         sr.ID,
		ReturnNumber:     sr.ReturnNumber,
		SalesOrderID:     sr.SalesOrderID,
		SalesOrderNumber: sr.SalesOrderNumber,
		CustomerID:       sr.CustomerID,
		CustomerName:     sr.CustomerName,
		Items:            items,
		TotalRefund:      sr.TotalRefund,
		Reason:           sr.Reason,
	}
}

// EventType returns the event type name
func (e *SalesReturnSubmittedEvent) EventType() string {
	return EventTypeSalesReturnSubmitted
}

// SalesReturnApprovedEvent is raised when a sales return is approved
// This event triggers stock restoration in the inventory context
type SalesReturnApprovedEvent struct {
	shared.BaseDomainEvent
	ReturnID         uuid.UUID             `json:"return_id"`
	ReturnNumber     string                `json:"return_number"`
	SalesOrderID     uuid.UUID             `json:"sales_order_id"`
	SalesOrderNumber string                `json:"sales_order_number"`
	CustomerID       uuid.UUID             `json:"customer_id"`
	CustomerName     string                `json:"customer_name"`
	WarehouseID      *uuid.UUID            `json:"warehouse_id,omitempty"`
	Items            []SalesReturnItemInfo `json:"items"`
	TotalRefund      decimal.Decimal       `json:"total_refund"`
	ApprovedBy       uuid.UUID             `json:"approved_by"`
	ApprovalNote     string                `json:"approval_note,omitempty"`
}

// NewSalesReturnApprovedEvent creates a new SalesReturnApprovedEvent
func NewSalesReturnApprovedEvent(sr *SalesReturn) *SalesReturnApprovedEvent {
	items := make([]SalesReturnItemInfo, len(sr.Items))
	for i, item := range sr.Items {
		items[i] = SalesReturnItemInfo{
			ItemID:           item.ID,
			SalesOrderItemID: item.SalesOrderItemID,
			ProductID:        item.ProductID,
			ProductName:      item.ProductName,
			ProductCode:      item.ProductCode,
			ReturnQuantity:   item.ReturnQuantity,
			UnitPrice:        item.UnitPrice,
			RefundAmount:     item.RefundAmount,
			Unit:             item.Unit,
		}
	}

	var approvedBy uuid.UUID
	if sr.ApprovedBy != nil {
		approvedBy = *sr.ApprovedBy
	}

	return &SalesReturnApprovedEvent{
		BaseDomainEvent:  shared.NewBaseDomainEvent(EventTypeSalesReturnApproved, AggregateTypeSalesReturn, sr.ID, sr.TenantID),
		ReturnID:         sr.ID,
		ReturnNumber:     sr.ReturnNumber,
		SalesOrderID:     sr.SalesOrderID,
		SalesOrderNumber: sr.SalesOrderNumber,
		CustomerID:       sr.CustomerID,
		CustomerName:     sr.CustomerName,
		WarehouseID:      sr.WarehouseID,
		Items:            items,
		TotalRefund:      sr.TotalRefund,
		ApprovedBy:       approvedBy,
		ApprovalNote:     sr.ApprovalNote,
	}
}

// EventType returns the event type name
func (e *SalesReturnApprovedEvent) EventType() string {
	return EventTypeSalesReturnApproved
}

// SalesReturnReceivingEvent is raised when a sales return starts receiving process
// This event triggers stock restoration in the inventory context
type SalesReturnReceivingEvent struct {
	shared.BaseDomainEvent
	ReturnID         uuid.UUID             `json:"return_id"`
	ReturnNumber     string                `json:"return_number"`
	SalesOrderID     uuid.UUID             `json:"sales_order_id"`
	SalesOrderNumber string                `json:"sales_order_number"`
	CustomerID       uuid.UUID             `json:"customer_id"`
	CustomerName     string                `json:"customer_name"`
	WarehouseID      uuid.UUID             `json:"warehouse_id"`
	Items            []SalesReturnItemInfo `json:"items"`
	TotalRefund      decimal.Decimal       `json:"total_refund"`
}

// NewSalesReturnReceivingEvent creates a new SalesReturnReceivingEvent
func NewSalesReturnReceivingEvent(sr *SalesReturn) *SalesReturnReceivingEvent {
	items := make([]SalesReturnItemInfo, len(sr.Items))
	for i, item := range sr.Items {
		items[i] = SalesReturnItemInfo{
			ItemID:           item.ID,
			SalesOrderItemID: item.SalesOrderItemID,
			ProductID:        item.ProductID,
			ProductName:      item.ProductName,
			ProductCode:      item.ProductCode,
			ReturnQuantity:   item.ReturnQuantity,
			UnitPrice:        item.UnitPrice,
			RefundAmount:     item.RefundAmount,
			Unit:             item.Unit,
		}
	}

	var warehouseID uuid.UUID
	if sr.WarehouseID != nil {
		warehouseID = *sr.WarehouseID
	}

	return &SalesReturnReceivingEvent{
		BaseDomainEvent:  shared.NewBaseDomainEvent(EventTypeSalesReturnReceiving, AggregateTypeSalesReturn, sr.ID, sr.TenantID),
		ReturnID:         sr.ID,
		ReturnNumber:     sr.ReturnNumber,
		SalesOrderID:     sr.SalesOrderID,
		SalesOrderNumber: sr.SalesOrderNumber,
		CustomerID:       sr.CustomerID,
		CustomerName:     sr.CustomerName,
		WarehouseID:      warehouseID,
		Items:            items,
		TotalRefund:      sr.TotalRefund,
	}
}

// EventType returns the event type name
func (e *SalesReturnReceivingEvent) EventType() string {
	return EventTypeSalesReturnReceiving
}

// SalesReturnRejectedEvent is raised when a sales return is rejected
type SalesReturnRejectedEvent struct {
	shared.BaseDomainEvent
	ReturnID         uuid.UUID       `json:"return_id"`
	ReturnNumber     string          `json:"return_number"`
	SalesOrderID     uuid.UUID       `json:"sales_order_id"`
	SalesOrderNumber string          `json:"sales_order_number"`
	CustomerID       uuid.UUID       `json:"customer_id"`
	TotalRefund      decimal.Decimal `json:"total_refund"`
	RejectedBy       uuid.UUID       `json:"rejected_by"`
	RejectionReason  string          `json:"rejection_reason"`
}

// NewSalesReturnRejectedEvent creates a new SalesReturnRejectedEvent
func NewSalesReturnRejectedEvent(sr *SalesReturn) *SalesReturnRejectedEvent {
	var rejectedBy uuid.UUID
	if sr.RejectedBy != nil {
		rejectedBy = *sr.RejectedBy
	}

	return &SalesReturnRejectedEvent{
		BaseDomainEvent:  shared.NewBaseDomainEvent(EventTypeSalesReturnRejected, AggregateTypeSalesReturn, sr.ID, sr.TenantID),
		ReturnID:         sr.ID,
		ReturnNumber:     sr.ReturnNumber,
		SalesOrderID:     sr.SalesOrderID,
		SalesOrderNumber: sr.SalesOrderNumber,
		CustomerID:       sr.CustomerID,
		TotalRefund:      sr.TotalRefund,
		RejectedBy:       rejectedBy,
		RejectionReason:  sr.RejectionReason,
	}
}

// EventType returns the event type name
func (e *SalesReturnRejectedEvent) EventType() string {
	return EventTypeSalesReturnRejected
}

// SalesReturnCompletedEvent is raised when a sales return is completed
// This event triggers accounts receivable credit and customer balance refund
type SalesReturnCompletedEvent struct {
	shared.BaseDomainEvent
	ReturnID         uuid.UUID             `json:"return_id"`
	ReturnNumber     string                `json:"return_number"`
	SalesOrderID     uuid.UUID             `json:"sales_order_id"`
	SalesOrderNumber string                `json:"sales_order_number"`
	CustomerID       uuid.UUID             `json:"customer_id"`
	CustomerName     string                `json:"customer_name"`
	WarehouseID      uuid.UUID             `json:"warehouse_id"`
	Items            []SalesReturnItemInfo `json:"items"`
	TotalRefund      decimal.Decimal       `json:"total_refund"`
}

// NewSalesReturnCompletedEvent creates a new SalesReturnCompletedEvent
func NewSalesReturnCompletedEvent(sr *SalesReturn) *SalesReturnCompletedEvent {
	items := make([]SalesReturnItemInfo, len(sr.Items))
	for i, item := range sr.Items {
		items[i] = SalesReturnItemInfo{
			ItemID:           item.ID,
			SalesOrderItemID: item.SalesOrderItemID,
			ProductID:        item.ProductID,
			ProductName:      item.ProductName,
			ProductCode:      item.ProductCode,
			ReturnQuantity:   item.ReturnQuantity,
			UnitPrice:        item.UnitPrice,
			RefundAmount:     item.RefundAmount,
			Unit:             item.Unit,
		}
	}

	var warehouseID uuid.UUID
	if sr.WarehouseID != nil {
		warehouseID = *sr.WarehouseID
	}

	return &SalesReturnCompletedEvent{
		BaseDomainEvent:  shared.NewBaseDomainEvent(EventTypeSalesReturnCompleted, AggregateTypeSalesReturn, sr.ID, sr.TenantID),
		ReturnID:         sr.ID,
		ReturnNumber:     sr.ReturnNumber,
		SalesOrderID:     sr.SalesOrderID,
		SalesOrderNumber: sr.SalesOrderNumber,
		CustomerID:       sr.CustomerID,
		CustomerName:     sr.CustomerName,
		WarehouseID:      warehouseID,
		Items:            items,
		TotalRefund:      sr.TotalRefund,
	}
}

// EventType returns the event type name
func (e *SalesReturnCompletedEvent) EventType() string {
	return EventTypeSalesReturnCompleted
}

// SalesReturnCancelledEvent is raised when a sales return is cancelled
// If WasApproved is true, any pending inventory operations should be cancelled
type SalesReturnCancelledEvent struct {
	shared.BaseDomainEvent
	ReturnID         uuid.UUID             `json:"return_id"`
	ReturnNumber     string                `json:"return_number"`
	SalesOrderID     uuid.UUID             `json:"sales_order_id"`
	SalesOrderNumber string                `json:"sales_order_number"`
	CustomerID       uuid.UUID             `json:"customer_id"`
	WarehouseID      *uuid.UUID            `json:"warehouse_id,omitempty"`
	Items            []SalesReturnItemInfo `json:"items"`
	CancelReason     string                `json:"cancel_reason"`
	WasApproved      bool                  `json:"was_approved"` // If true, inventory operations may need to be reversed
}

// NewSalesReturnCancelledEvent creates a new SalesReturnCancelledEvent
func NewSalesReturnCancelledEvent(sr *SalesReturn, wasApproved bool) *SalesReturnCancelledEvent {
	items := make([]SalesReturnItemInfo, len(sr.Items))
	for i, item := range sr.Items {
		items[i] = SalesReturnItemInfo{
			ItemID:           item.ID,
			SalesOrderItemID: item.SalesOrderItemID,
			ProductID:        item.ProductID,
			ProductName:      item.ProductName,
			ProductCode:      item.ProductCode,
			ReturnQuantity:   item.ReturnQuantity,
			UnitPrice:        item.UnitPrice,
			RefundAmount:     item.RefundAmount,
			Unit:             item.Unit,
		}
	}

	return &SalesReturnCancelledEvent{
		BaseDomainEvent:  shared.NewBaseDomainEvent(EventTypeSalesReturnCancelled, AggregateTypeSalesReturn, sr.ID, sr.TenantID),
		ReturnID:         sr.ID,
		ReturnNumber:     sr.ReturnNumber,
		SalesOrderID:     sr.SalesOrderID,
		SalesOrderNumber: sr.SalesOrderNumber,
		CustomerID:       sr.CustomerID,
		WarehouseID:      sr.WarehouseID,
		Items:            items,
		CancelReason:     sr.CancelReason,
		WasApproved:      wasApproved,
	}
}

// EventType returns the event type name
func (e *SalesReturnCancelledEvent) EventType() string {
	return EventTypeSalesReturnCancelled
}
