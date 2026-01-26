package finance

import (
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AccountReceivableCreatedEvent is raised when a new account receivable is created
type AccountReceivableCreatedEvent struct {
	shared.BaseDomainEvent
	ReceivableID     uuid.UUID       `json:"receivable_id"`
	ReceivableNumber string          `json:"receivable_number"`
	CustomerID       uuid.UUID       `json:"customer_id"`
	CustomerName     string          `json:"customer_name"`
	SourceType       SourceType      `json:"source_type"`
	SourceID         uuid.UUID       `json:"source_id"`
	SourceNumber     string          `json:"source_number"`
	TotalAmount      decimal.Decimal `json:"total_amount"`
	DueDate          *time.Time      `json:"due_date,omitempty"`
}

// EventType returns the event type name
func (e *AccountReceivableCreatedEvent) EventType() string {
	return "AccountReceivableCreated"
}

// NewAccountReceivableCreatedEvent creates a new AccountReceivableCreatedEvent
func NewAccountReceivableCreatedEvent(ar *AccountReceivable) *AccountReceivableCreatedEvent {
	return &AccountReceivableCreatedEvent{
		BaseDomainEvent:  shared.NewBaseDomainEvent("AccountReceivableCreated", "AccountReceivable", ar.ID, ar.TenantID),
		ReceivableID:     ar.ID,
		ReceivableNumber: ar.ReceivableNumber,
		CustomerID:       ar.CustomerID,
		CustomerName:     ar.CustomerName,
		SourceType:       ar.SourceType,
		SourceID:         ar.SourceID,
		SourceNumber:     ar.SourceNumber,
		TotalAmount:      ar.TotalAmount,
		DueDate:          ar.DueDate,
	}
}

// AccountReceivablePaidEvent is raised when a receivable is fully paid
type AccountReceivablePaidEvent struct {
	shared.BaseDomainEvent
	ReceivableID     uuid.UUID       `json:"receivable_id"`
	ReceivableNumber string          `json:"receivable_number"`
	CustomerID       uuid.UUID       `json:"customer_id"`
	CustomerName     string          `json:"customer_name"`
	TotalAmount      decimal.Decimal `json:"total_amount"`
	PaidAmount       decimal.Decimal `json:"paid_amount"`
	PaidAt           time.Time       `json:"paid_at"`
}

// EventType returns the event type name
func (e *AccountReceivablePaidEvent) EventType() string {
	return "AccountReceivablePaid"
}

// NewAccountReceivablePaidEvent creates a new AccountReceivablePaidEvent
func NewAccountReceivablePaidEvent(ar *AccountReceivable) *AccountReceivablePaidEvent {
	paidAt := time.Now()
	if ar.PaidAt != nil {
		paidAt = *ar.PaidAt
	}
	return &AccountReceivablePaidEvent{
		BaseDomainEvent:  shared.NewBaseDomainEvent("AccountReceivablePaid", "AccountReceivable", ar.ID, ar.TenantID),
		ReceivableID:     ar.ID,
		ReceivableNumber: ar.ReceivableNumber,
		CustomerID:       ar.CustomerID,
		CustomerName:     ar.CustomerName,
		TotalAmount:      ar.TotalAmount,
		PaidAmount:       ar.PaidAmount,
		PaidAt:           paidAt,
	}
}

// AccountReceivablePartiallyPaidEvent is raised when a partial payment is applied
type AccountReceivablePartiallyPaidEvent struct {
	shared.BaseDomainEvent
	ReceivableID      uuid.UUID       `json:"receivable_id"`
	ReceivableNumber  string          `json:"receivable_number"`
	CustomerID        uuid.UUID       `json:"customer_id"`
	CustomerName      string          `json:"customer_name"`
	PaymentAmount     decimal.Decimal `json:"payment_amount"`
	TotalAmount       decimal.Decimal `json:"total_amount"`
	PaidAmount        decimal.Decimal `json:"paid_amount"`
	OutstandingAmount decimal.Decimal `json:"outstanding_amount"`
}

// EventType returns the event type name
func (e *AccountReceivablePartiallyPaidEvent) EventType() string {
	return "AccountReceivablePartiallyPaid"
}

// NewAccountReceivablePartiallyPaidEvent creates a new AccountReceivablePartiallyPaidEvent
func NewAccountReceivablePartiallyPaidEvent(ar *AccountReceivable, paymentAmount valueobject.Money) *AccountReceivablePartiallyPaidEvent {
	return &AccountReceivablePartiallyPaidEvent{
		BaseDomainEvent:   shared.NewBaseDomainEvent("AccountReceivablePartiallyPaid", "AccountReceivable", ar.ID, ar.TenantID),
		ReceivableID:      ar.ID,
		ReceivableNumber:  ar.ReceivableNumber,
		CustomerID:        ar.CustomerID,
		CustomerName:      ar.CustomerName,
		PaymentAmount:     paymentAmount.Amount(),
		TotalAmount:       ar.TotalAmount,
		PaidAmount:        ar.PaidAmount,
		OutstandingAmount: ar.OutstandingAmount,
	}
}

// ReversedPaymentInfo contains information about a payment record that was reversed
// BUG-010: Provides complete audit trail for reversed payments
type ReversedPaymentInfo struct {
	PaymentRecordID  uuid.UUID       `json:"payment_record_id"`  // Original payment record ID
	ReceiptVoucherID uuid.UUID       `json:"receipt_voucher_id"` // Receipt voucher that recorded the payment
	Amount           decimal.Decimal `json:"amount"`             // Amount of the payment
	AppliedAt        time.Time       `json:"applied_at"`         // When the payment was originally applied
	ReversedAt       time.Time       `json:"reversed_at"`        // When the payment was reversed
	CompensationID   uuid.UUID       `json:"compensation_id"`    // ID for compensation/refund record
}

// AccountReceivableReversedEvent is raised when a receivable is reversed
type AccountReceivableReversedEvent struct {
	shared.BaseDomainEvent
	ReceivableID      uuid.UUID        `json:"receivable_id"`
	ReceivableNumber  string           `json:"receivable_number"`
	CustomerID        uuid.UUID        `json:"customer_id"`
	CustomerName      string           `json:"customer_name"`
	TotalAmount       decimal.Decimal  `json:"total_amount"`
	PaidAmount        decimal.Decimal  `json:"paid_amount"`
	OutstandingAmount decimal.Decimal  `json:"outstanding_amount"`
	PreviousStatus    ReceivableStatus `json:"previous_status"`
	ReversalReason    string           `json:"reversal_reason"`
	ReversedAt        time.Time        `json:"reversed_at"`
	// Refund-related fields (BUG-009 fix)
	RefundRequired bool            `json:"refund_required"` // True if a refund needs to be processed
	RefundAmount   decimal.Decimal `json:"refund_amount"`   // Amount that was paid and needs refund
	// BUG-010: Payment record audit trail
	ReversedPaymentCount int                   `json:"reversed_payment_count"` // Number of payments reversed
	ReversedPayments     []ReversedPaymentInfo `json:"reversed_payments"`      // Detailed info for each reversed payment
}

// EventType returns the event type name
func (e *AccountReceivableReversedEvent) EventType() string {
	return "AccountReceivableReversed"
}

// NewAccountReceivableReversedEvent creates a new AccountReceivableReversedEvent
// BUG-010: Now accepts ReversalResult to include detailed payment reversal information
func NewAccountReceivableReversedEvent(ar *AccountReceivable, previousStatus ReceivableStatus, result *ReversalResult) *AccountReceivableReversedEvent {
	reversedAt := time.Now()
	if ar.ReversedAt != nil {
		reversedAt = *ar.ReversedAt
	}
	// BUG-009 fix: Include refund information in the event
	refundRequired := ar.PaidAmount.GreaterThan(decimal.Zero)

	// BUG-010: Build reversed payment details for audit trail
	reversedPayments := make([]ReversedPaymentInfo, 0, result.ReversedPaymentCount)
	for i, pr := range result.PaymentRecords {
		if pr.IsReversed() && i < len(result.CompensationRecordIDs) {
			reversedPayments = append(reversedPayments, ReversedPaymentInfo{
				PaymentRecordID:  pr.ID,
				ReceiptVoucherID: pr.ReceiptVoucherID,
				Amount:           pr.Amount,
				AppliedAt:        pr.AppliedAt,
				ReversedAt:       *pr.ReversedAt,
				CompensationID:   result.CompensationRecordIDs[i],
			})
		}
	}

	return &AccountReceivableReversedEvent{
		BaseDomainEvent:      shared.NewBaseDomainEvent("AccountReceivableReversed", "AccountReceivable", ar.ID, ar.TenantID),
		ReceivableID:         ar.ID,
		ReceivableNumber:     ar.ReceivableNumber,
		CustomerID:           ar.CustomerID,
		CustomerName:         ar.CustomerName,
		TotalAmount:          ar.TotalAmount,
		PaidAmount:           ar.PaidAmount,
		OutstandingAmount:    ar.OutstandingAmount,
		PreviousStatus:       previousStatus,
		ReversalReason:       ar.ReversalReason,
		ReversedAt:           reversedAt,
		RefundRequired:       refundRequired,
		RefundAmount:         ar.PaidAmount,
		ReversedPaymentCount: result.ReversedPaymentCount,
		ReversedPayments:     reversedPayments,
	}
}

// AccountReceivableCancelledEvent is raised when a receivable is cancelled
type AccountReceivableCancelledEvent struct {
	shared.BaseDomainEvent
	ReceivableID     uuid.UUID       `json:"receivable_id"`
	ReceivableNumber string          `json:"receivable_number"`
	CustomerID       uuid.UUID       `json:"customer_id"`
	CustomerName     string          `json:"customer_name"`
	TotalAmount      decimal.Decimal `json:"total_amount"`
	CancelReason     string          `json:"cancel_reason"`
	CancelledAt      time.Time       `json:"cancelled_at"`
}

// EventType returns the event type name
func (e *AccountReceivableCancelledEvent) EventType() string {
	return "AccountReceivableCancelled"
}

// NewAccountReceivableCancelledEvent creates a new AccountReceivableCancelledEvent
func NewAccountReceivableCancelledEvent(ar *AccountReceivable) *AccountReceivableCancelledEvent {
	cancelledAt := time.Now()
	if ar.CancelledAt != nil {
		cancelledAt = *ar.CancelledAt
	}
	return &AccountReceivableCancelledEvent{
		BaseDomainEvent:  shared.NewBaseDomainEvent("AccountReceivableCancelled", "AccountReceivable", ar.ID, ar.TenantID),
		ReceivableID:     ar.ID,
		ReceivableNumber: ar.ReceivableNumber,
		CustomerID:       ar.CustomerID,
		CustomerName:     ar.CustomerName,
		TotalAmount:      ar.TotalAmount,
		CancelReason:     ar.CancelReason,
		CancelledAt:      cancelledAt,
	}
}
