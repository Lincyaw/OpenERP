package finance

import (
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ReceiptVoucherCreatedEvent is raised when a new receipt voucher is created
type ReceiptVoucherCreatedEvent struct {
	shared.BaseDomainEvent
	VoucherID     uuid.UUID       `json:"voucher_id"`
	VoucherNumber string          `json:"voucher_number"`
	CustomerID    uuid.UUID       `json:"customer_id"`
	CustomerName  string          `json:"customer_name"`
	Amount        decimal.Decimal `json:"amount"`
	PaymentMethod PaymentMethod   `json:"payment_method"`
	ReceiptDate   time.Time       `json:"receipt_date"`
}

// EventType returns the event type name
func (e *ReceiptVoucherCreatedEvent) EventType() string {
	return "ReceiptVoucherCreated"
}

// NewReceiptVoucherCreatedEvent creates a new ReceiptVoucherCreatedEvent
func NewReceiptVoucherCreatedEvent(rv *ReceiptVoucher) *ReceiptVoucherCreatedEvent {
	return &ReceiptVoucherCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent("ReceiptVoucherCreated", "ReceiptVoucher", rv.ID, rv.TenantID),
		VoucherID:       rv.ID,
		VoucherNumber:   rv.VoucherNumber,
		CustomerID:      rv.CustomerID,
		CustomerName:    rv.CustomerName,
		Amount:          rv.Amount,
		PaymentMethod:   rv.PaymentMethod,
		ReceiptDate:     rv.ReceiptDate,
	}
}

// ReceiptVoucherConfirmedEvent is raised when a receipt voucher is confirmed
type ReceiptVoucherConfirmedEvent struct {
	shared.BaseDomainEvent
	VoucherID     uuid.UUID       `json:"voucher_id"`
	VoucherNumber string          `json:"voucher_number"`
	CustomerID    uuid.UUID       `json:"customer_id"`
	Amount        decimal.Decimal `json:"amount"`
	ConfirmedBy   uuid.UUID       `json:"confirmed_by"`
	ConfirmedAt   time.Time       `json:"confirmed_at"`
}

// EventType returns the event type name
func (e *ReceiptVoucherConfirmedEvent) EventType() string {
	return "ReceiptVoucherConfirmed"
}

// NewReceiptVoucherConfirmedEvent creates a new ReceiptVoucherConfirmedEvent
func NewReceiptVoucherConfirmedEvent(rv *ReceiptVoucher) *ReceiptVoucherConfirmedEvent {
	var confirmedBy uuid.UUID
	confirmedAt := time.Now()
	if rv.ConfirmedBy != nil {
		confirmedBy = *rv.ConfirmedBy
	}
	if rv.ConfirmedAt != nil {
		confirmedAt = *rv.ConfirmedAt
	}
	return &ReceiptVoucherConfirmedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent("ReceiptVoucherConfirmed", "ReceiptVoucher", rv.ID, rv.TenantID),
		VoucherID:       rv.ID,
		VoucherNumber:   rv.VoucherNumber,
		CustomerID:      rv.CustomerID,
		Amount:          rv.Amount,
		ConfirmedBy:     confirmedBy,
		ConfirmedAt:     confirmedAt,
	}
}

// ReceiptVoucherAllocatedEvent is raised when a receipt voucher is allocated to a receivable
type ReceiptVoucherAllocatedEvent struct {
	shared.BaseDomainEvent
	VoucherID        uuid.UUID       `json:"voucher_id"`
	VoucherNumber    string          `json:"voucher_number"`
	CustomerID       uuid.UUID       `json:"customer_id"`
	ReceivableID     uuid.UUID       `json:"receivable_id"`
	ReceivableNumber string          `json:"receivable_number"`
	AllocationAmount decimal.Decimal `json:"allocation_amount"`
	TotalAllocated   decimal.Decimal `json:"total_allocated"`
	RemainingAmount  decimal.Decimal `json:"remaining_amount"`
	IsFullyAllocated bool            `json:"is_fully_allocated"`
}

// EventType returns the event type name
func (e *ReceiptVoucherAllocatedEvent) EventType() string {
	return "ReceiptVoucherAllocated"
}

// NewReceiptVoucherAllocatedEvent creates a new ReceiptVoucherAllocatedEvent
func NewReceiptVoucherAllocatedEvent(rv *ReceiptVoucher, allocation *ReceivableAllocation) *ReceiptVoucherAllocatedEvent {
	return &ReceiptVoucherAllocatedEvent{
		BaseDomainEvent:  shared.NewBaseDomainEvent("ReceiptVoucherAllocated", "ReceiptVoucher", rv.ID, rv.TenantID),
		VoucherID:        rv.ID,
		VoucherNumber:    rv.VoucherNumber,
		CustomerID:       rv.CustomerID,
		ReceivableID:     allocation.ReceivableID,
		ReceivableNumber: allocation.ReceivableNumber,
		AllocationAmount: allocation.Amount,
		TotalAllocated:   rv.AllocatedAmount,
		RemainingAmount:  rv.UnallocatedAmount,
		IsFullyAllocated: rv.IsFullyAllocated(),
	}
}

// ReceiptVoucherCancelledEvent is raised when a receipt voucher is cancelled
type ReceiptVoucherCancelledEvent struct {
	shared.BaseDomainEvent
	VoucherID      uuid.UUID       `json:"voucher_id"`
	VoucherNumber  string          `json:"voucher_number"`
	CustomerID     uuid.UUID       `json:"customer_id"`
	Amount         decimal.Decimal `json:"amount"`
	PreviousStatus VoucherStatus   `json:"previous_status"`
	CancelledBy    uuid.UUID       `json:"cancelled_by"`
	CancelReason   string          `json:"cancel_reason"`
	CancelledAt    time.Time       `json:"cancelled_at"`
}

// EventType returns the event type name
func (e *ReceiptVoucherCancelledEvent) EventType() string {
	return "ReceiptVoucherCancelled"
}

// NewReceiptVoucherCancelledEvent creates a new ReceiptVoucherCancelledEvent
func NewReceiptVoucherCancelledEvent(rv *ReceiptVoucher, previousStatus VoucherStatus) *ReceiptVoucherCancelledEvent {
	var cancelledBy uuid.UUID
	cancelledAt := time.Now()
	if rv.CancelledBy != nil {
		cancelledBy = *rv.CancelledBy
	}
	if rv.CancelledAt != nil {
		cancelledAt = *rv.CancelledAt
	}
	return &ReceiptVoucherCancelledEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent("ReceiptVoucherCancelled", "ReceiptVoucher", rv.ID, rv.TenantID),
		VoucherID:       rv.ID,
		VoucherNumber:   rv.VoucherNumber,
		CustomerID:      rv.CustomerID,
		Amount:          rv.Amount,
		PreviousStatus:  previousStatus,
		CancelledBy:     cancelledBy,
		CancelReason:    rv.CancelReason,
		CancelledAt:     cancelledAt,
	}
}
