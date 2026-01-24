package finance

import (
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Aggregate type constant for DebitMemo
const AggregateTypeDebitMemo = "DebitMemo"

// Event type constants for DebitMemo
const (
	EventTypeDebitMemoCreated          = "DebitMemoCreated"
	EventTypeDebitMemoApplied          = "DebitMemoApplied"
	EventTypeDebitMemoPartiallyApplied = "DebitMemoPartiallyApplied"
	EventTypeDebitMemoVoided           = "DebitMemoVoided"
	EventTypeDebitMemoRefundReceived   = "DebitMemoRefundReceived"
)

// DebitMemoItemInfo represents item information for events
type DebitMemoItemInfo struct {
	ItemID         uuid.UUID       `json:"item_id"`
	ProductID      uuid.UUID       `json:"product_id"`
	ProductName    string          `json:"product_name"`
	ProductCode    string          `json:"product_code"`
	ReturnQuantity decimal.Decimal `json:"return_quantity"`
	UnitCost       decimal.Decimal `json:"unit_cost"`
	DebitAmount    decimal.Decimal `json:"debit_amount"`
	Unit           string          `json:"unit"`
}

// DebitMemoCreatedEvent is raised when a new debit memo is created
type DebitMemoCreatedEvent struct {
	shared.BaseDomainEvent
	MemoID               uuid.UUID           `json:"memo_id"`
	MemoNumber           string              `json:"memo_number"`
	PurchaseReturnID     uuid.UUID           `json:"purchase_return_id"`
	PurchaseReturnNumber string              `json:"purchase_return_number"`
	PurchaseOrderID      uuid.UUID           `json:"purchase_order_id"`
	PurchaseOrderNumber  string              `json:"purchase_order_number"`
	SupplierID           uuid.UUID           `json:"supplier_id"`
	SupplierName         string              `json:"supplier_name"`
	Items                []DebitMemoItemInfo `json:"items"`
	TotalDebit           decimal.Decimal     `json:"total_debit"`
	Reason               string              `json:"reason"`
}

// NewDebitMemoCreatedEvent creates a new DebitMemoCreatedEvent
func NewDebitMemoCreatedEvent(dm *DebitMemo) *DebitMemoCreatedEvent {
	items := make([]DebitMemoItemInfo, len(dm.Items))
	for i, item := range dm.Items {
		items[i] = DebitMemoItemInfo{
			ItemID:         item.ID,
			ProductID:      item.ProductID,
			ProductName:    item.ProductName,
			ProductCode:    item.ProductCode,
			ReturnQuantity: item.ReturnQuantity,
			UnitCost:       item.UnitCost,
			DebitAmount:    item.DebitAmount,
			Unit:           item.Unit,
		}
	}

	return &DebitMemoCreatedEvent{
		BaseDomainEvent:      shared.NewBaseDomainEvent(EventTypeDebitMemoCreated, AggregateTypeDebitMemo, dm.ID, dm.TenantID),
		MemoID:               dm.ID,
		MemoNumber:           dm.MemoNumber,
		PurchaseReturnID:     dm.PurchaseReturnID,
		PurchaseReturnNumber: dm.PurchaseReturnNumber,
		PurchaseOrderID:      dm.PurchaseOrderID,
		PurchaseOrderNumber:  dm.PurchaseOrderNumber,
		SupplierID:           dm.SupplierID,
		SupplierName:         dm.SupplierName,
		Items:                items,
		TotalDebit:           dm.TotalDebit,
		Reason:               dm.Reason,
	}
}

// EventType returns the event type name
func (e *DebitMemoCreatedEvent) EventType() string {
	return EventTypeDebitMemoCreated
}

// DebitMemoAppliedEvent is raised when a debit memo is fully applied
type DebitMemoAppliedEvent struct {
	shared.BaseDomainEvent
	MemoID        uuid.UUID       `json:"memo_id"`
	MemoNumber    string          `json:"memo_number"`
	SupplierID    uuid.UUID       `json:"supplier_id"`
	SupplierName  string          `json:"supplier_name"`
	TotalDebit    decimal.Decimal `json:"total_debit"`
	AppliedAmount decimal.Decimal `json:"applied_amount"`
}

// NewDebitMemoAppliedEvent creates a new DebitMemoAppliedEvent
func NewDebitMemoAppliedEvent(dm *DebitMemo) *DebitMemoAppliedEvent {
	return &DebitMemoAppliedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeDebitMemoApplied, AggregateTypeDebitMemo, dm.ID, dm.TenantID),
		MemoID:          dm.ID,
		MemoNumber:      dm.MemoNumber,
		SupplierID:      dm.SupplierID,
		SupplierName:    dm.SupplierName,
		TotalDebit:      dm.TotalDebit,
		AppliedAmount:   dm.AppliedAmount,
	}
}

// EventType returns the event type name
func (e *DebitMemoAppliedEvent) EventType() string {
	return EventTypeDebitMemoApplied
}

// DebitMemoPartiallyAppliedEvent is raised when debit is partially applied
type DebitMemoPartiallyAppliedEvent struct {
	shared.BaseDomainEvent
	MemoID          uuid.UUID       `json:"memo_id"`
	MemoNumber      string          `json:"memo_number"`
	SupplierID      uuid.UUID       `json:"supplier_id"`
	AppliedAmount   decimal.Decimal `json:"applied_amount"`
	TotalApplied    decimal.Decimal `json:"total_applied"`
	RemainingAmount decimal.Decimal `json:"remaining_amount"`
}

// NewDebitMemoPartiallyAppliedEvent creates a new DebitMemoPartiallyAppliedEvent
func NewDebitMemoPartiallyAppliedEvent(dm *DebitMemo, appliedAmount valueobject.Money) *DebitMemoPartiallyAppliedEvent {
	return &DebitMemoPartiallyAppliedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeDebitMemoPartiallyApplied, AggregateTypeDebitMemo, dm.ID, dm.TenantID),
		MemoID:          dm.ID,
		MemoNumber:      dm.MemoNumber,
		SupplierID:      dm.SupplierID,
		AppliedAmount:   appliedAmount.Amount(),
		TotalApplied:    dm.AppliedAmount,
		RemainingAmount: dm.RemainingAmount,
	}
}

// EventType returns the event type name
func (e *DebitMemoPartiallyAppliedEvent) EventType() string {
	return EventTypeDebitMemoPartiallyApplied
}

// DebitMemoVoidedEvent is raised when a debit memo is voided
type DebitMemoVoidedEvent struct {
	shared.BaseDomainEvent
	MemoID     uuid.UUID       `json:"memo_id"`
	MemoNumber string          `json:"memo_number"`
	SupplierID uuid.UUID       `json:"supplier_id"`
	TotalDebit decimal.Decimal `json:"total_debit"`
	VoidReason string          `json:"void_reason"`
}

// NewDebitMemoVoidedEvent creates a new DebitMemoVoidedEvent
func NewDebitMemoVoidedEvent(dm *DebitMemo) *DebitMemoVoidedEvent {
	return &DebitMemoVoidedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeDebitMemoVoided, AggregateTypeDebitMemo, dm.ID, dm.TenantID),
		MemoID:          dm.ID,
		MemoNumber:      dm.MemoNumber,
		SupplierID:      dm.SupplierID,
		TotalDebit:      dm.TotalDebit,
		VoidReason:      dm.VoidReason,
	}
}

// EventType returns the event type name
func (e *DebitMemoVoidedEvent) EventType() string {
	return EventTypeDebitMemoVoided
}

// DebitMemoRefundReceivedEvent is raised when a refund is received from supplier
type DebitMemoRefundReceivedEvent struct {
	shared.BaseDomainEvent
	MemoID       uuid.UUID       `json:"memo_id"`
	MemoNumber   string          `json:"memo_number"`
	SupplierID   uuid.UUID       `json:"supplier_id"`
	SupplierName string          `json:"supplier_name"`
	RefundAmount decimal.Decimal `json:"refund_amount"`
	RefundMethod string          `json:"refund_method"`
}

// NewDebitMemoRefundReceivedEvent creates a new DebitMemoRefundReceivedEvent
func NewDebitMemoRefundReceivedEvent(dm *DebitMemo, refundAmount decimal.Decimal) *DebitMemoRefundReceivedEvent {
	return &DebitMemoRefundReceivedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeDebitMemoRefundReceived, AggregateTypeDebitMemo, dm.ID, dm.TenantID),
		MemoID:          dm.ID,
		MemoNumber:      dm.MemoNumber,
		SupplierID:      dm.SupplierID,
		SupplierName:    dm.SupplierName,
		RefundAmount:    refundAmount,
		RefundMethod:    dm.RefundMethod,
	}
}

// EventType returns the event type name
func (e *DebitMemoRefundReceivedEvent) EventType() string {
	return EventTypeDebitMemoRefundReceived
}
