package trade

import (
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Aggregate type constant for PurchaseReturn
const AggregateTypePurchaseReturn = "PurchaseReturn"

// Event type constants for PurchaseReturn
const (
	EventTypePurchaseReturnCreated   = "PurchaseReturnCreated"
	EventTypePurchaseReturnSubmitted = "PurchaseReturnSubmitted"
	EventTypePurchaseReturnApproved  = "PurchaseReturnApproved"
	EventTypePurchaseReturnRejected  = "PurchaseReturnRejected"
	EventTypePurchaseReturnShipped   = "PurchaseReturnShipped"
	EventTypePurchaseReturnCompleted = "PurchaseReturnCompleted"
	EventTypePurchaseReturnCancelled = "PurchaseReturnCancelled"
)

// PurchaseReturnItemInfo represents item information for events
type PurchaseReturnItemInfo struct {
	ItemID              uuid.UUID       `json:"item_id"`
	PurchaseOrderItemID uuid.UUID       `json:"purchase_order_item_id"`
	ProductID           uuid.UUID       `json:"product_id"`
	ProductName         string          `json:"product_name"`
	ProductCode         string          `json:"product_code"`
	ReturnQuantity      decimal.Decimal `json:"return_quantity"`
	UnitCost            decimal.Decimal `json:"unit_cost"`
	RefundAmount        decimal.Decimal `json:"refund_amount"`
	Unit                string          `json:"unit"`
	BatchNumber         string          `json:"batch_number,omitempty"`
}

// PurchaseReturnCreatedEvent is raised when a new purchase return is created
type PurchaseReturnCreatedEvent struct {
	shared.BaseDomainEvent
	ReturnID            uuid.UUID `json:"return_id"`
	ReturnNumber        string    `json:"return_number"`
	PurchaseOrderID     uuid.UUID `json:"purchase_order_id"`
	PurchaseOrderNumber string    `json:"purchase_order_number"`
	SupplierID          uuid.UUID `json:"supplier_id"`
	SupplierName        string    `json:"supplier_name"`
}

// NewPurchaseReturnCreatedEvent creates a new PurchaseReturnCreatedEvent
func NewPurchaseReturnCreatedEvent(pr *PurchaseReturn) *PurchaseReturnCreatedEvent {
	return &PurchaseReturnCreatedEvent{
		BaseDomainEvent:     shared.NewBaseDomainEvent(EventTypePurchaseReturnCreated, AggregateTypePurchaseReturn, pr.ID, pr.TenantID),
		ReturnID:            pr.ID,
		ReturnNumber:        pr.ReturnNumber,
		PurchaseOrderID:     pr.PurchaseOrderID,
		PurchaseOrderNumber: pr.PurchaseOrderNumber,
		SupplierID:          pr.SupplierID,
		SupplierName:        pr.SupplierName,
	}
}

// EventType returns the event type name
func (e *PurchaseReturnCreatedEvent) EventType() string {
	return EventTypePurchaseReturnCreated
}

// PurchaseReturnSubmittedEvent is raised when a purchase return is submitted for approval
type PurchaseReturnSubmittedEvent struct {
	shared.BaseDomainEvent
	ReturnID            uuid.UUID                `json:"return_id"`
	ReturnNumber        string                   `json:"return_number"`
	PurchaseOrderID     uuid.UUID                `json:"purchase_order_id"`
	PurchaseOrderNumber string                   `json:"purchase_order_number"`
	SupplierID          uuid.UUID                `json:"supplier_id"`
	SupplierName        string                   `json:"supplier_name"`
	Items               []PurchaseReturnItemInfo `json:"items"`
	TotalRefund         decimal.Decimal          `json:"total_refund"`
	Reason              string                   `json:"reason"`
}

// NewPurchaseReturnSubmittedEvent creates a new PurchaseReturnSubmittedEvent
func NewPurchaseReturnSubmittedEvent(pr *PurchaseReturn) *PurchaseReturnSubmittedEvent {
	items := make([]PurchaseReturnItemInfo, len(pr.Items))
	for i, item := range pr.Items {
		items[i] = PurchaseReturnItemInfo{
			ItemID:              item.ID,
			PurchaseOrderItemID: item.PurchaseOrderItemID,
			ProductID:           item.ProductID,
			ProductName:         item.ProductName,
			ProductCode:         item.ProductCode,
			ReturnQuantity:      item.ReturnQuantity,
			UnitCost:            item.UnitCost,
			RefundAmount:        item.RefundAmount,
			Unit:                item.Unit,
			BatchNumber:         item.BatchNumber,
		}
	}

	return &PurchaseReturnSubmittedEvent{
		BaseDomainEvent:     shared.NewBaseDomainEvent(EventTypePurchaseReturnSubmitted, AggregateTypePurchaseReturn, pr.ID, pr.TenantID),
		ReturnID:            pr.ID,
		ReturnNumber:        pr.ReturnNumber,
		PurchaseOrderID:     pr.PurchaseOrderID,
		PurchaseOrderNumber: pr.PurchaseOrderNumber,
		SupplierID:          pr.SupplierID,
		SupplierName:        pr.SupplierName,
		Items:               items,
		TotalRefund:         pr.TotalRefund,
		Reason:              pr.Reason,
	}
}

// EventType returns the event type name
func (e *PurchaseReturnSubmittedEvent) EventType() string {
	return EventTypePurchaseReturnSubmitted
}

// PurchaseReturnApprovedEvent is raised when a purchase return is approved
type PurchaseReturnApprovedEvent struct {
	shared.BaseDomainEvent
	ReturnID            uuid.UUID                `json:"return_id"`
	ReturnNumber        string                   `json:"return_number"`
	PurchaseOrderID     uuid.UUID                `json:"purchase_order_id"`
	PurchaseOrderNumber string                   `json:"purchase_order_number"`
	SupplierID          uuid.UUID                `json:"supplier_id"`
	SupplierName        string                   `json:"supplier_name"`
	WarehouseID         *uuid.UUID               `json:"warehouse_id,omitempty"`
	Items               []PurchaseReturnItemInfo `json:"items"`
	TotalRefund         decimal.Decimal          `json:"total_refund"`
	ApprovedBy          uuid.UUID                `json:"approved_by"`
	ApprovalNote        string                   `json:"approval_note,omitempty"`
}

// NewPurchaseReturnApprovedEvent creates a new PurchaseReturnApprovedEvent
func NewPurchaseReturnApprovedEvent(pr *PurchaseReturn) *PurchaseReturnApprovedEvent {
	items := make([]PurchaseReturnItemInfo, len(pr.Items))
	for i, item := range pr.Items {
		items[i] = PurchaseReturnItemInfo{
			ItemID:              item.ID,
			PurchaseOrderItemID: item.PurchaseOrderItemID,
			ProductID:           item.ProductID,
			ProductName:         item.ProductName,
			ProductCode:         item.ProductCode,
			ReturnQuantity:      item.ReturnQuantity,
			UnitCost:            item.UnitCost,
			RefundAmount:        item.RefundAmount,
			Unit:                item.Unit,
			BatchNumber:         item.BatchNumber,
		}
	}

	var approvedBy uuid.UUID
	if pr.ApprovedBy != nil {
		approvedBy = *pr.ApprovedBy
	}

	return &PurchaseReturnApprovedEvent{
		BaseDomainEvent:     shared.NewBaseDomainEvent(EventTypePurchaseReturnApproved, AggregateTypePurchaseReturn, pr.ID, pr.TenantID),
		ReturnID:            pr.ID,
		ReturnNumber:        pr.ReturnNumber,
		PurchaseOrderID:     pr.PurchaseOrderID,
		PurchaseOrderNumber: pr.PurchaseOrderNumber,
		SupplierID:          pr.SupplierID,
		SupplierName:        pr.SupplierName,
		WarehouseID:         pr.WarehouseID,
		Items:               items,
		TotalRefund:         pr.TotalRefund,
		ApprovedBy:          approvedBy,
		ApprovalNote:        pr.ApprovalNote,
	}
}

// EventType returns the event type name
func (e *PurchaseReturnApprovedEvent) EventType() string {
	return EventTypePurchaseReturnApproved
}

// PurchaseReturnRejectedEvent is raised when a purchase return is rejected
type PurchaseReturnRejectedEvent struct {
	shared.BaseDomainEvent
	ReturnID            uuid.UUID       `json:"return_id"`
	ReturnNumber        string          `json:"return_number"`
	PurchaseOrderID     uuid.UUID       `json:"purchase_order_id"`
	PurchaseOrderNumber string          `json:"purchase_order_number"`
	SupplierID          uuid.UUID       `json:"supplier_id"`
	TotalRefund         decimal.Decimal `json:"total_refund"`
	RejectedBy          uuid.UUID       `json:"rejected_by"`
	RejectionReason     string          `json:"rejection_reason"`
}

// NewPurchaseReturnRejectedEvent creates a new PurchaseReturnRejectedEvent
func NewPurchaseReturnRejectedEvent(pr *PurchaseReturn) *PurchaseReturnRejectedEvent {
	var rejectedBy uuid.UUID
	if pr.RejectedBy != nil {
		rejectedBy = *pr.RejectedBy
	}

	return &PurchaseReturnRejectedEvent{
		BaseDomainEvent:     shared.NewBaseDomainEvent(EventTypePurchaseReturnRejected, AggregateTypePurchaseReturn, pr.ID, pr.TenantID),
		ReturnID:            pr.ID,
		ReturnNumber:        pr.ReturnNumber,
		PurchaseOrderID:     pr.PurchaseOrderID,
		PurchaseOrderNumber: pr.PurchaseOrderNumber,
		SupplierID:          pr.SupplierID,
		TotalRefund:         pr.TotalRefund,
		RejectedBy:          rejectedBy,
		RejectionReason:     pr.RejectionReason,
	}
}

// EventType returns the event type name
func (e *PurchaseReturnRejectedEvent) EventType() string {
	return EventTypePurchaseReturnRejected
}

// PurchaseReturnShippedEvent is raised when a purchase return is shipped to supplier
// This event triggers stock deduction in the inventory context
type PurchaseReturnShippedEvent struct {
	shared.BaseDomainEvent
	ReturnID            uuid.UUID                `json:"return_id"`
	ReturnNumber        string                   `json:"return_number"`
	PurchaseOrderID     uuid.UUID                `json:"purchase_order_id"`
	PurchaseOrderNumber string                   `json:"purchase_order_number"`
	SupplierID          uuid.UUID                `json:"supplier_id"`
	SupplierName        string                   `json:"supplier_name"`
	WarehouseID         uuid.UUID                `json:"warehouse_id"`
	Items               []PurchaseReturnItemInfo `json:"items"`
	TotalRefund         decimal.Decimal          `json:"total_refund"`
	ShippedBy           *uuid.UUID               `json:"shipped_by,omitempty"`
	ShippingNote        string                   `json:"shipping_note,omitempty"`
	TrackingNumber      string                   `json:"tracking_number,omitempty"`
}

// NewPurchaseReturnShippedEvent creates a new PurchaseReturnShippedEvent
func NewPurchaseReturnShippedEvent(pr *PurchaseReturn) *PurchaseReturnShippedEvent {
	items := make([]PurchaseReturnItemInfo, len(pr.Items))
	for i, item := range pr.Items {
		items[i] = PurchaseReturnItemInfo{
			ItemID:              item.ID,
			PurchaseOrderItemID: item.PurchaseOrderItemID,
			ProductID:           item.ProductID,
			ProductName:         item.ProductName,
			ProductCode:         item.ProductCode,
			ReturnQuantity:      item.ReturnQuantity,
			UnitCost:            item.UnitCost,
			RefundAmount:        item.RefundAmount,
			Unit:                item.Unit,
			BatchNumber:         item.BatchNumber,
		}
	}

	var warehouseID uuid.UUID
	if pr.WarehouseID != nil {
		warehouseID = *pr.WarehouseID
	}

	return &PurchaseReturnShippedEvent{
		BaseDomainEvent:     shared.NewBaseDomainEvent(EventTypePurchaseReturnShipped, AggregateTypePurchaseReturn, pr.ID, pr.TenantID),
		ReturnID:            pr.ID,
		ReturnNumber:        pr.ReturnNumber,
		PurchaseOrderID:     pr.PurchaseOrderID,
		PurchaseOrderNumber: pr.PurchaseOrderNumber,
		SupplierID:          pr.SupplierID,
		SupplierName:        pr.SupplierName,
		WarehouseID:         warehouseID,
		Items:               items,
		TotalRefund:         pr.TotalRefund,
		ShippedBy:           pr.ShippedBy,
		ShippingNote:        pr.ShippingNote,
		TrackingNumber:      pr.TrackingNumber,
	}
}

// EventType returns the event type name
func (e *PurchaseReturnShippedEvent) EventType() string {
	return EventTypePurchaseReturnShipped
}

// PurchaseReturnCompletedEvent is raised when a purchase return is completed
// This event triggers accounts payable credit
type PurchaseReturnCompletedEvent struct {
	shared.BaseDomainEvent
	ReturnID            uuid.UUID                `json:"return_id"`
	ReturnNumber        string                   `json:"return_number"`
	PurchaseOrderID     uuid.UUID                `json:"purchase_order_id"`
	PurchaseOrderNumber string                   `json:"purchase_order_number"`
	SupplierID          uuid.UUID                `json:"supplier_id"`
	SupplierName        string                   `json:"supplier_name"`
	WarehouseID         uuid.UUID                `json:"warehouse_id"`
	Items               []PurchaseReturnItemInfo `json:"items"`
	TotalRefund         decimal.Decimal          `json:"total_refund"`
}

// NewPurchaseReturnCompletedEvent creates a new PurchaseReturnCompletedEvent
func NewPurchaseReturnCompletedEvent(pr *PurchaseReturn) *PurchaseReturnCompletedEvent {
	items := make([]PurchaseReturnItemInfo, len(pr.Items))
	for i, item := range pr.Items {
		items[i] = PurchaseReturnItemInfo{
			ItemID:              item.ID,
			PurchaseOrderItemID: item.PurchaseOrderItemID,
			ProductID:           item.ProductID,
			ProductName:         item.ProductName,
			ProductCode:         item.ProductCode,
			ReturnQuantity:      item.ReturnQuantity,
			UnitCost:            item.UnitCost,
			RefundAmount:        item.RefundAmount,
			Unit:                item.Unit,
			BatchNumber:         item.BatchNumber,
		}
	}

	var warehouseID uuid.UUID
	if pr.WarehouseID != nil {
		warehouseID = *pr.WarehouseID
	}

	return &PurchaseReturnCompletedEvent{
		BaseDomainEvent:     shared.NewBaseDomainEvent(EventTypePurchaseReturnCompleted, AggregateTypePurchaseReturn, pr.ID, pr.TenantID),
		ReturnID:            pr.ID,
		ReturnNumber:        pr.ReturnNumber,
		PurchaseOrderID:     pr.PurchaseOrderID,
		PurchaseOrderNumber: pr.PurchaseOrderNumber,
		SupplierID:          pr.SupplierID,
		SupplierName:        pr.SupplierName,
		WarehouseID:         warehouseID,
		Items:               items,
		TotalRefund:         pr.TotalRefund,
	}
}

// EventType returns the event type name
func (e *PurchaseReturnCompletedEvent) EventType() string {
	return EventTypePurchaseReturnCompleted
}

// PurchaseReturnCancelledEvent is raised when a purchase return is cancelled
// If WasApproved is true, any pending operations should be cancelled
type PurchaseReturnCancelledEvent struct {
	shared.BaseDomainEvent
	ReturnID            uuid.UUID                `json:"return_id"`
	ReturnNumber        string                   `json:"return_number"`
	PurchaseOrderID     uuid.UUID                `json:"purchase_order_id"`
	PurchaseOrderNumber string                   `json:"purchase_order_number"`
	SupplierID          uuid.UUID                `json:"supplier_id"`
	WarehouseID         *uuid.UUID               `json:"warehouse_id,omitempty"`
	Items               []PurchaseReturnItemInfo `json:"items"`
	CancelReason        string                   `json:"cancel_reason"`
	WasApproved         bool                     `json:"was_approved"` // If true, may need to reverse operations
}

// NewPurchaseReturnCancelledEvent creates a new PurchaseReturnCancelledEvent
func NewPurchaseReturnCancelledEvent(pr *PurchaseReturn, wasApproved bool) *PurchaseReturnCancelledEvent {
	items := make([]PurchaseReturnItemInfo, len(pr.Items))
	for i, item := range pr.Items {
		items[i] = PurchaseReturnItemInfo{
			ItemID:              item.ID,
			PurchaseOrderItemID: item.PurchaseOrderItemID,
			ProductID:           item.ProductID,
			ProductName:         item.ProductName,
			ProductCode:         item.ProductCode,
			ReturnQuantity:      item.ReturnQuantity,
			UnitCost:            item.UnitCost,
			RefundAmount:        item.RefundAmount,
			Unit:                item.Unit,
			BatchNumber:         item.BatchNumber,
		}
	}

	return &PurchaseReturnCancelledEvent{
		BaseDomainEvent:     shared.NewBaseDomainEvent(EventTypePurchaseReturnCancelled, AggregateTypePurchaseReturn, pr.ID, pr.TenantID),
		ReturnID:            pr.ID,
		ReturnNumber:        pr.ReturnNumber,
		PurchaseOrderID:     pr.PurchaseOrderID,
		PurchaseOrderNumber: pr.PurchaseOrderNumber,
		SupplierID:          pr.SupplierID,
		WarehouseID:         pr.WarehouseID,
		Items:               items,
		CancelReason:        pr.CancelReason,
		WasApproved:         wasApproved,
	}
}

// EventType returns the event type name
func (e *PurchaseReturnCancelledEvent) EventType() string {
	return EventTypePurchaseReturnCancelled
}
