package finance

import (
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PaymentVoucherCreatedEvent is raised when a new payment voucher is created
type PaymentVoucherCreatedEvent struct {
	shared.BaseDomainEvent
	VoucherID     uuid.UUID       `json:"voucher_id"`
	VoucherNumber string          `json:"voucher_number"`
	SupplierID    uuid.UUID       `json:"supplier_id"`
	SupplierName  string          `json:"supplier_name"`
	Amount        decimal.Decimal `json:"amount"`
	PaymentMethod PaymentMethod   `json:"payment_method"`
	PaymentDate   time.Time       `json:"payment_date"`
}

// EventType returns the event type name
func (e *PaymentVoucherCreatedEvent) EventType() string {
	return "PaymentVoucherCreated"
}

// NewPaymentVoucherCreatedEvent creates a new PaymentVoucherCreatedEvent
func NewPaymentVoucherCreatedEvent(pv *PaymentVoucher) *PaymentVoucherCreatedEvent {
	return &PaymentVoucherCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent("PaymentVoucherCreated", "PaymentVoucher", pv.ID, pv.TenantID),
		VoucherID:       pv.ID,
		VoucherNumber:   pv.VoucherNumber,
		SupplierID:      pv.SupplierID,
		SupplierName:    pv.SupplierName,
		Amount:          pv.Amount,
		PaymentMethod:   pv.PaymentMethod,
		PaymentDate:     pv.PaymentDate,
	}
}

// PaymentVoucherConfirmedEvent is raised when a payment voucher is confirmed
type PaymentVoucherConfirmedEvent struct {
	shared.BaseDomainEvent
	VoucherID     uuid.UUID       `json:"voucher_id"`
	VoucherNumber string          `json:"voucher_number"`
	SupplierID    uuid.UUID       `json:"supplier_id"`
	Amount        decimal.Decimal `json:"amount"`
	ConfirmedBy   uuid.UUID       `json:"confirmed_by"`
	ConfirmedAt   time.Time       `json:"confirmed_at"`
}

// EventType returns the event type name
func (e *PaymentVoucherConfirmedEvent) EventType() string {
	return "PaymentVoucherConfirmed"
}

// NewPaymentVoucherConfirmedEvent creates a new PaymentVoucherConfirmedEvent
func NewPaymentVoucherConfirmedEvent(pv *PaymentVoucher) *PaymentVoucherConfirmedEvent {
	var confirmedBy uuid.UUID
	confirmedAt := time.Now()
	if pv.ConfirmedBy != nil {
		confirmedBy = *pv.ConfirmedBy
	}
	if pv.ConfirmedAt != nil {
		confirmedAt = *pv.ConfirmedAt
	}
	return &PaymentVoucherConfirmedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent("PaymentVoucherConfirmed", "PaymentVoucher", pv.ID, pv.TenantID),
		VoucherID:       pv.ID,
		VoucherNumber:   pv.VoucherNumber,
		SupplierID:      pv.SupplierID,
		Amount:          pv.Amount,
		ConfirmedBy:     confirmedBy,
		ConfirmedAt:     confirmedAt,
	}
}

// PaymentVoucherAllocatedEvent is raised when a payment voucher is allocated to a payable
type PaymentVoucherAllocatedEvent struct {
	shared.BaseDomainEvent
	VoucherID        uuid.UUID       `json:"voucher_id"`
	VoucherNumber    string          `json:"voucher_number"`
	SupplierID       uuid.UUID       `json:"supplier_id"`
	PayableID        uuid.UUID       `json:"payable_id"`
	PayableNumber    string          `json:"payable_number"`
	AllocationAmount decimal.Decimal `json:"allocation_amount"`
	TotalAllocated   decimal.Decimal `json:"total_allocated"`
	RemainingAmount  decimal.Decimal `json:"remaining_amount"`
	IsFullyAllocated bool            `json:"is_fully_allocated"`
}

// EventType returns the event type name
func (e *PaymentVoucherAllocatedEvent) EventType() string {
	return "PaymentVoucherAllocated"
}

// NewPaymentVoucherAllocatedEvent creates a new PaymentVoucherAllocatedEvent
func NewPaymentVoucherAllocatedEvent(pv *PaymentVoucher, allocation *PayableAllocation) *PaymentVoucherAllocatedEvent {
	return &PaymentVoucherAllocatedEvent{
		BaseDomainEvent:  shared.NewBaseDomainEvent("PaymentVoucherAllocated", "PaymentVoucher", pv.ID, pv.TenantID),
		VoucherID:        pv.ID,
		VoucherNumber:    pv.VoucherNumber,
		SupplierID:       pv.SupplierID,
		PayableID:        allocation.PayableID,
		PayableNumber:    allocation.PayableNumber,
		AllocationAmount: allocation.Amount,
		TotalAllocated:   pv.AllocatedAmount,
		RemainingAmount:  pv.UnallocatedAmount,
		IsFullyAllocated: pv.IsFullyAllocated(),
	}
}

// PaymentVoucherCancelledEvent is raised when a payment voucher is cancelled
type PaymentVoucherCancelledEvent struct {
	shared.BaseDomainEvent
	VoucherID      uuid.UUID       `json:"voucher_id"`
	VoucherNumber  string          `json:"voucher_number"`
	SupplierID     uuid.UUID       `json:"supplier_id"`
	Amount         decimal.Decimal `json:"amount"`
	PreviousStatus VoucherStatus   `json:"previous_status"`
	CancelledBy    uuid.UUID       `json:"cancelled_by"`
	CancelReason   string          `json:"cancel_reason"`
	CancelledAt    time.Time       `json:"cancelled_at"`
}

// EventType returns the event type name
func (e *PaymentVoucherCancelledEvent) EventType() string {
	return "PaymentVoucherCancelled"
}

// NewPaymentVoucherCancelledEvent creates a new PaymentVoucherCancelledEvent
func NewPaymentVoucherCancelledEvent(pv *PaymentVoucher, previousStatus VoucherStatus) *PaymentVoucherCancelledEvent {
	var cancelledBy uuid.UUID
	cancelledAt := time.Now()
	if pv.CancelledBy != nil {
		cancelledBy = *pv.CancelledBy
	}
	if pv.CancelledAt != nil {
		cancelledAt = *pv.CancelledAt
	}
	return &PaymentVoucherCancelledEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent("PaymentVoucherCancelled", "PaymentVoucher", pv.ID, pv.TenantID),
		VoucherID:       pv.ID,
		VoucherNumber:   pv.VoucherNumber,
		SupplierID:      pv.SupplierID,
		Amount:          pv.Amount,
		PreviousStatus:  previousStatus,
		CancelledBy:     cancelledBy,
		CancelReason:    pv.CancelReason,
		CancelledAt:     cancelledAt,
	}
}
