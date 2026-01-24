package finance

import (
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Aggregate type constant for CreditMemo
const AggregateTypeCreditMemo = "CreditMemo"

// Event type constants for CreditMemo
const (
	EventTypeCreditMemoCreated          = "CreditMemoCreated"
	EventTypeCreditMemoApplied          = "CreditMemoApplied"
	EventTypeCreditMemoPartiallyApplied = "CreditMemoPartiallyApplied"
	EventTypeCreditMemoVoided           = "CreditMemoVoided"
	EventTypeCreditMemoRefunded         = "CreditMemoRefunded"
)

// CreditMemoItemInfo represents item information for events
type CreditMemoItemInfo struct {
	ItemID         uuid.UUID       `json:"item_id"`
	ProductID      uuid.UUID       `json:"product_id"`
	ProductName    string          `json:"product_name"`
	ProductCode    string          `json:"product_code"`
	ReturnQuantity decimal.Decimal `json:"return_quantity"`
	UnitPrice      decimal.Decimal `json:"unit_price"`
	CreditAmount   decimal.Decimal `json:"credit_amount"`
	Unit           string          `json:"unit"`
}

// CreditMemoCreatedEvent is raised when a new credit memo is created
type CreditMemoCreatedEvent struct {
	shared.BaseDomainEvent
	MemoID            uuid.UUID            `json:"memo_id"`
	MemoNumber        string               `json:"memo_number"`
	SalesReturnID     uuid.UUID            `json:"sales_return_id"`
	SalesReturnNumber string               `json:"sales_return_number"`
	SalesOrderID      uuid.UUID            `json:"sales_order_id"`
	SalesOrderNumber  string               `json:"sales_order_number"`
	CustomerID        uuid.UUID            `json:"customer_id"`
	CustomerName      string               `json:"customer_name"`
	Items             []CreditMemoItemInfo `json:"items"`
	TotalCredit       decimal.Decimal      `json:"total_credit"`
	Reason            string               `json:"reason"`
}

// NewCreditMemoCreatedEvent creates a new CreditMemoCreatedEvent
func NewCreditMemoCreatedEvent(cm *CreditMemo) *CreditMemoCreatedEvent {
	items := make([]CreditMemoItemInfo, len(cm.Items))
	for i, item := range cm.Items {
		items[i] = CreditMemoItemInfo{
			ItemID:         item.ID,
			ProductID:      item.ProductID,
			ProductName:    item.ProductName,
			ProductCode:    item.ProductCode,
			ReturnQuantity: item.ReturnQuantity,
			UnitPrice:      item.UnitPrice,
			CreditAmount:   item.CreditAmount,
			Unit:           item.Unit,
		}
	}

	return &CreditMemoCreatedEvent{
		BaseDomainEvent:   shared.NewBaseDomainEvent(EventTypeCreditMemoCreated, AggregateTypeCreditMemo, cm.ID, cm.TenantID),
		MemoID:            cm.ID,
		MemoNumber:        cm.MemoNumber,
		SalesReturnID:     cm.SalesReturnID,
		SalesReturnNumber: cm.SalesReturnNumber,
		SalesOrderID:      cm.SalesOrderID,
		SalesOrderNumber:  cm.SalesOrderNumber,
		CustomerID:        cm.CustomerID,
		CustomerName:      cm.CustomerName,
		Items:             items,
		TotalCredit:       cm.TotalCredit,
		Reason:            cm.Reason,
	}
}

// EventType returns the event type name
func (e *CreditMemoCreatedEvent) EventType() string {
	return EventTypeCreditMemoCreated
}

// CreditMemoAppliedEvent is raised when a credit memo is fully applied
type CreditMemoAppliedEvent struct {
	shared.BaseDomainEvent
	MemoID        uuid.UUID       `json:"memo_id"`
	MemoNumber    string          `json:"memo_number"`
	CustomerID    uuid.UUID       `json:"customer_id"`
	CustomerName  string          `json:"customer_name"`
	TotalCredit   decimal.Decimal `json:"total_credit"`
	AppliedAmount decimal.Decimal `json:"applied_amount"`
}

// NewCreditMemoAppliedEvent creates a new CreditMemoAppliedEvent
func NewCreditMemoAppliedEvent(cm *CreditMemo) *CreditMemoAppliedEvent {
	return &CreditMemoAppliedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeCreditMemoApplied, AggregateTypeCreditMemo, cm.ID, cm.TenantID),
		MemoID:          cm.ID,
		MemoNumber:      cm.MemoNumber,
		CustomerID:      cm.CustomerID,
		CustomerName:    cm.CustomerName,
		TotalCredit:     cm.TotalCredit,
		AppliedAmount:   cm.AppliedAmount,
	}
}

// EventType returns the event type name
func (e *CreditMemoAppliedEvent) EventType() string {
	return EventTypeCreditMemoApplied
}

// CreditMemoPartiallyAppliedEvent is raised when credit is partially applied
type CreditMemoPartiallyAppliedEvent struct {
	shared.BaseDomainEvent
	MemoID          uuid.UUID       `json:"memo_id"`
	MemoNumber      string          `json:"memo_number"`
	CustomerID      uuid.UUID       `json:"customer_id"`
	AppliedAmount   decimal.Decimal `json:"applied_amount"`
	TotalApplied    decimal.Decimal `json:"total_applied"`
	RemainingAmount decimal.Decimal `json:"remaining_amount"`
}

// NewCreditMemoPartiallyAppliedEvent creates a new CreditMemoPartiallyAppliedEvent
func NewCreditMemoPartiallyAppliedEvent(cm *CreditMemo, appliedAmount valueobject.Money) *CreditMemoPartiallyAppliedEvent {
	return &CreditMemoPartiallyAppliedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeCreditMemoPartiallyApplied, AggregateTypeCreditMemo, cm.ID, cm.TenantID),
		MemoID:          cm.ID,
		MemoNumber:      cm.MemoNumber,
		CustomerID:      cm.CustomerID,
		AppliedAmount:   appliedAmount.Amount(),
		TotalApplied:    cm.AppliedAmount,
		RemainingAmount: cm.RemainingAmount,
	}
}

// EventType returns the event type name
func (e *CreditMemoPartiallyAppliedEvent) EventType() string {
	return EventTypeCreditMemoPartiallyApplied
}

// CreditMemoVoidedEvent is raised when a credit memo is voided
type CreditMemoVoidedEvent struct {
	shared.BaseDomainEvent
	MemoID      uuid.UUID       `json:"memo_id"`
	MemoNumber  string          `json:"memo_number"`
	CustomerID  uuid.UUID       `json:"customer_id"`
	TotalCredit decimal.Decimal `json:"total_credit"`
	VoidReason  string          `json:"void_reason"`
}

// NewCreditMemoVoidedEvent creates a new CreditMemoVoidedEvent
func NewCreditMemoVoidedEvent(cm *CreditMemo) *CreditMemoVoidedEvent {
	return &CreditMemoVoidedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeCreditMemoVoided, AggregateTypeCreditMemo, cm.ID, cm.TenantID),
		MemoID:          cm.ID,
		MemoNumber:      cm.MemoNumber,
		CustomerID:      cm.CustomerID,
		TotalCredit:     cm.TotalCredit,
		VoidReason:      cm.VoidReason,
	}
}

// EventType returns the event type name
func (e *CreditMemoVoidedEvent) EventType() string {
	return EventTypeCreditMemoVoided
}

// CreditMemoRefundedEvent is raised when a credit memo is refunded
type CreditMemoRefundedEvent struct {
	shared.BaseDomainEvent
	MemoID       uuid.UUID       `json:"memo_id"`
	MemoNumber   string          `json:"memo_number"`
	CustomerID   uuid.UUID       `json:"customer_id"`
	CustomerName string          `json:"customer_name"`
	RefundAmount decimal.Decimal `json:"refund_amount"`
	RefundMethod string          `json:"refund_method"`
}

// NewCreditMemoRefundedEvent creates a new CreditMemoRefundedEvent
func NewCreditMemoRefundedEvent(cm *CreditMemo, refundAmount decimal.Decimal) *CreditMemoRefundedEvent {
	return &CreditMemoRefundedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeCreditMemoRefunded, AggregateTypeCreditMemo, cm.ID, cm.TenantID),
		MemoID:          cm.ID,
		MemoNumber:      cm.MemoNumber,
		CustomerID:      cm.CustomerID,
		CustomerName:    cm.CustomerName,
		RefundAmount:    refundAmount,
		RefundMethod:    cm.RefundMethod,
	}
}

// EventType returns the event type name
func (e *CreditMemoRefundedEvent) EventType() string {
	return EventTypeCreditMemoRefunded
}
