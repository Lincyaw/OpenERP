package finance

import (
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AccountPayableCreatedEvent is raised when a new account payable is created
type AccountPayableCreatedEvent struct {
	shared.BaseDomainEvent
	PayableID     uuid.UUID         `json:"payable_id"`
	PayableNumber string            `json:"payable_number"`
	SupplierID    uuid.UUID         `json:"supplier_id"`
	SupplierName  string            `json:"supplier_name"`
	SourceType    PayableSourceType `json:"source_type"`
	SourceID      uuid.UUID         `json:"source_id"`
	SourceNumber  string            `json:"source_number"`
	TotalAmount   decimal.Decimal   `json:"total_amount"`
	DueDate       *time.Time        `json:"due_date,omitempty"`
}

// EventType returns the event type name
func (e *AccountPayableCreatedEvent) EventType() string {
	return "AccountPayableCreated"
}

// NewAccountPayableCreatedEvent creates a new AccountPayableCreatedEvent
func NewAccountPayableCreatedEvent(ap *AccountPayable) *AccountPayableCreatedEvent {
	return &AccountPayableCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent("AccountPayableCreated", "AccountPayable", ap.ID, ap.TenantID),
		PayableID:       ap.ID,
		PayableNumber:   ap.PayableNumber,
		SupplierID:      ap.SupplierID,
		SupplierName:    ap.SupplierName,
		SourceType:      ap.SourceType,
		SourceID:        ap.SourceID,
		SourceNumber:    ap.SourceNumber,
		TotalAmount:     ap.TotalAmount,
		DueDate:         ap.DueDate,
	}
}

// AccountPayablePaidEvent is raised when a payable is fully paid
type AccountPayablePaidEvent struct {
	shared.BaseDomainEvent
	PayableID     uuid.UUID       `json:"payable_id"`
	PayableNumber string          `json:"payable_number"`
	SupplierID    uuid.UUID       `json:"supplier_id"`
	SupplierName  string          `json:"supplier_name"`
	TotalAmount   decimal.Decimal `json:"total_amount"`
	PaidAmount    decimal.Decimal `json:"paid_amount"`
	PaidAt        time.Time       `json:"paid_at"`
}

// EventType returns the event type name
func (e *AccountPayablePaidEvent) EventType() string {
	return "AccountPayablePaid"
}

// NewAccountPayablePaidEvent creates a new AccountPayablePaidEvent
func NewAccountPayablePaidEvent(ap *AccountPayable) *AccountPayablePaidEvent {
	paidAt := time.Now()
	if ap.PaidAt != nil {
		paidAt = *ap.PaidAt
	}
	return &AccountPayablePaidEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent("AccountPayablePaid", "AccountPayable", ap.ID, ap.TenantID),
		PayableID:       ap.ID,
		PayableNumber:   ap.PayableNumber,
		SupplierID:      ap.SupplierID,
		SupplierName:    ap.SupplierName,
		TotalAmount:     ap.TotalAmount,
		PaidAmount:      ap.PaidAmount,
		PaidAt:          paidAt,
	}
}

// AccountPayablePartiallyPaidEvent is raised when a partial payment is applied
type AccountPayablePartiallyPaidEvent struct {
	shared.BaseDomainEvent
	PayableID         uuid.UUID       `json:"payable_id"`
	PayableNumber     string          `json:"payable_number"`
	SupplierID        uuid.UUID       `json:"supplier_id"`
	SupplierName      string          `json:"supplier_name"`
	PaymentAmount     decimal.Decimal `json:"payment_amount"`
	TotalAmount       decimal.Decimal `json:"total_amount"`
	PaidAmount        decimal.Decimal `json:"paid_amount"`
	OutstandingAmount decimal.Decimal `json:"outstanding_amount"`
}

// EventType returns the event type name
func (e *AccountPayablePartiallyPaidEvent) EventType() string {
	return "AccountPayablePartiallyPaid"
}

// NewAccountPayablePartiallyPaidEvent creates a new AccountPayablePartiallyPaidEvent
func NewAccountPayablePartiallyPaidEvent(ap *AccountPayable, paymentAmount valueobject.Money) *AccountPayablePartiallyPaidEvent {
	return &AccountPayablePartiallyPaidEvent{
		BaseDomainEvent:   shared.NewBaseDomainEvent("AccountPayablePartiallyPaid", "AccountPayable", ap.ID, ap.TenantID),
		PayableID:         ap.ID,
		PayableNumber:     ap.PayableNumber,
		SupplierID:        ap.SupplierID,
		SupplierName:      ap.SupplierName,
		PaymentAmount:     paymentAmount.Amount(),
		TotalAmount:       ap.TotalAmount,
		PaidAmount:        ap.PaidAmount,
		OutstandingAmount: ap.OutstandingAmount,
	}
}

// AccountPayableReversedEvent is raised when a payable is reversed
type AccountPayableReversedEvent struct {
	shared.BaseDomainEvent
	PayableID         uuid.UUID       `json:"payable_id"`
	PayableNumber     string          `json:"payable_number"`
	SupplierID        uuid.UUID       `json:"supplier_id"`
	SupplierName      string          `json:"supplier_name"`
	TotalAmount       decimal.Decimal `json:"total_amount"`
	PaidAmount        decimal.Decimal `json:"paid_amount"`
	OutstandingAmount decimal.Decimal `json:"outstanding_amount"`
	PreviousStatus    PayableStatus   `json:"previous_status"`
	ReversalReason    string          `json:"reversal_reason"`
	ReversedAt        time.Time       `json:"reversed_at"`
}

// EventType returns the event type name
func (e *AccountPayableReversedEvent) EventType() string {
	return "AccountPayableReversed"
}

// NewAccountPayableReversedEvent creates a new AccountPayableReversedEvent
func NewAccountPayableReversedEvent(ap *AccountPayable, previousStatus PayableStatus) *AccountPayableReversedEvent {
	reversedAt := time.Now()
	if ap.ReversedAt != nil {
		reversedAt = *ap.ReversedAt
	}
	return &AccountPayableReversedEvent{
		BaseDomainEvent:   shared.NewBaseDomainEvent("AccountPayableReversed", "AccountPayable", ap.ID, ap.TenantID),
		PayableID:         ap.ID,
		PayableNumber:     ap.PayableNumber,
		SupplierID:        ap.SupplierID,
		SupplierName:      ap.SupplierName,
		TotalAmount:       ap.TotalAmount,
		PaidAmount:        ap.PaidAmount,
		OutstandingAmount: ap.OutstandingAmount,
		PreviousStatus:    previousStatus,
		ReversalReason:    ap.ReversalReason,
		ReversedAt:        reversedAt,
	}
}

// AccountPayableCancelledEvent is raised when a payable is cancelled
type AccountPayableCancelledEvent struct {
	shared.BaseDomainEvent
	PayableID     uuid.UUID       `json:"payable_id"`
	PayableNumber string          `json:"payable_number"`
	SupplierID    uuid.UUID       `json:"supplier_id"`
	SupplierName  string          `json:"supplier_name"`
	TotalAmount   decimal.Decimal `json:"total_amount"`
	CancelReason  string          `json:"cancel_reason"`
	CancelledAt   time.Time       `json:"cancelled_at"`
}

// EventType returns the event type name
func (e *AccountPayableCancelledEvent) EventType() string {
	return "AccountPayableCancelled"
}

// NewAccountPayableCancelledEvent creates a new AccountPayableCancelledEvent
func NewAccountPayableCancelledEvent(ap *AccountPayable) *AccountPayableCancelledEvent {
	cancelledAt := time.Now()
	if ap.CancelledAt != nil {
		cancelledAt = *ap.CancelledAt
	}
	return &AccountPayableCancelledEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent("AccountPayableCancelled", "AccountPayable", ap.ID, ap.TenantID),
		PayableID:       ap.ID,
		PayableNumber:   ap.PayableNumber,
		SupplierID:      ap.SupplierID,
		SupplierName:    ap.SupplierName,
		TotalAmount:     ap.TotalAmount,
		CancelReason:    ap.CancelReason,
		CancelledAt:     cancelledAt,
	}
}
