package finance

import (
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// VoucherStatus represents the status of a receipt/payment voucher
type VoucherStatus string

const (
	VoucherStatusDraft     VoucherStatus = "DRAFT"     // Not yet confirmed
	VoucherStatusConfirmed VoucherStatus = "CONFIRMED" // Confirmed and can be allocated
	VoucherStatusAllocated VoucherStatus = "ALLOCATED" // Fully allocated to receivables/payables
	VoucherStatusCancelled VoucherStatus = "CANCELLED" // Cancelled
)

// IsValid checks if the status is a valid VoucherStatus
func (s VoucherStatus) IsValid() bool {
	switch s {
	case VoucherStatusDraft, VoucherStatusConfirmed, VoucherStatusAllocated, VoucherStatusCancelled:
		return true
	}
	return false
}

// String returns the string representation of VoucherStatus
func (s VoucherStatus) String() string {
	return string(s)
}

// IsTerminal returns true if the voucher is in a terminal state
func (s VoucherStatus) IsTerminal() bool {
	return s == VoucherStatusAllocated || s == VoucherStatusCancelled
}

// CanAllocate returns true if allocations can be made in this status
func (s VoucherStatus) CanAllocate() bool {
	return s == VoucherStatusConfirmed
}

// CanConfirm returns true if the voucher can be confirmed in this status
func (s VoucherStatus) CanConfirm() bool {
	return s == VoucherStatusDraft
}

// CanCancel returns true if the voucher can be cancelled in this status
func (s VoucherStatus) CanCancel() bool {
	return s == VoucherStatusDraft || s == VoucherStatusConfirmed
}

// PaymentMethod represents the method of payment
type PaymentMethod string

const (
	PaymentMethodCash         PaymentMethod = "CASH"          // Cash payment
	PaymentMethodBankTransfer PaymentMethod = "BANK_TRANSFER" // Bank transfer
	PaymentMethodWechat       PaymentMethod = "WECHAT"        // WeChat Pay
	PaymentMethodAlipay       PaymentMethod = "ALIPAY"        // Alipay
	PaymentMethodCheck        PaymentMethod = "CHECK"         // Check/Cheque
	PaymentMethodBalance      PaymentMethod = "BALANCE"       // Customer balance (prepaid)
	PaymentMethodOther        PaymentMethod = "OTHER"         // Other methods
)

// IsValid checks if the payment method is valid
func (m PaymentMethod) IsValid() bool {
	switch m {
	case PaymentMethodCash, PaymentMethodBankTransfer, PaymentMethodWechat,
		PaymentMethodAlipay, PaymentMethodCheck, PaymentMethodBalance, PaymentMethodOther:
		return true
	}
	return false
}

// String returns the string representation of PaymentMethod
func (m PaymentMethod) String() string {
	return string(m)
}

// ReceivableAllocation represents the allocation of a receipt voucher to a receivable
type ReceivableAllocation struct {
	ID               uuid.UUID       `json:"id"`
	ReceiptVoucherID uuid.UUID       `json:"receipt_voucher_id"`
	ReceivableID     uuid.UUID       `json:"receivable_id"`
	ReceivableNumber string          `json:"receivable_number"` // Denormalized for display
	Amount           decimal.Decimal `json:"amount"`
	AllocatedAt      time.Time       `json:"allocated_at"`
	Remark           string          `json:"remark"`
}

// NewReceivableAllocation creates a new receivable allocation
func NewReceivableAllocation(voucherID, receivableID uuid.UUID, receivableNumber string, amount valueobject.Money, remark string) *ReceivableAllocation {
	return &ReceivableAllocation{
		ID:               uuid.New(),
		ReceiptVoucherID: voucherID,
		ReceivableID:     receivableID,
		ReceivableNumber: receivableNumber,
		Amount:           amount.Amount(),
		AllocatedAt:      time.Now(),
		Remark:           remark,
	}
}

// GetAmountMoney returns the amount as Money value object
func (a *ReceivableAllocation) GetAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(a.Amount)
}

// ReceiptVoucher represents a receipt voucher aggregate root
// It records a payment received from a customer
type ReceiptVoucher struct {
	shared.TenantAggregateRoot
	VoucherNumber     string                 `json:"voucher_number"`
	CustomerID        uuid.UUID              `json:"customer_id"`
	CustomerName      string                 `json:"customer_name"`
	Amount            decimal.Decimal        `json:"amount"`             // Total receipt amount
	AllocatedAmount   decimal.Decimal        `json:"allocated_amount"`   // Amount allocated to receivables
	UnallocatedAmount decimal.Decimal        `json:"unallocated_amount"` // Remaining unallocated amount
	PaymentMethod     PaymentMethod          `json:"payment_method"`     // Payment method
	PaymentReference  string                 `json:"payment_reference"`  // Reference (bank txn, check #)
	Status            VoucherStatus          `json:"status"`
	ReceiptDate       time.Time              `json:"receipt_date"` // When payment was received
	Allocations       []ReceivableAllocation `json:"allocations"`
	Remark            string                 `json:"remark"`
	ConfirmedAt       *time.Time             `json:"confirmed_at"` // When confirmed
	ConfirmedBy       *uuid.UUID             `json:"confirmed_by"` // User who confirmed
	CancelledAt       *time.Time             `json:"cancelled_at"` // When cancelled
	CancelledBy       *uuid.UUID             `json:"cancelled_by"` // User who cancelled
	CancelReason      string                 `json:"cancel_reason"`
}

// NewReceiptVoucher creates a new receipt voucher
func NewReceiptVoucher(
	tenantID uuid.UUID,
	voucherNumber string,
	customerID uuid.UUID,
	customerName string,
	amount valueobject.Money,
	paymentMethod PaymentMethod,
	receiptDate time.Time,
) (*ReceiptVoucher, error) {
	// Validate inputs
	if voucherNumber == "" {
		return nil, shared.NewDomainError("INVALID_VOUCHER_NUMBER", "Voucher number cannot be empty")
	}
	if len(voucherNumber) > 50 {
		return nil, shared.NewDomainError("INVALID_VOUCHER_NUMBER", "Voucher number cannot exceed 50 characters")
	}
	if customerID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_CUSTOMER", "Customer ID cannot be empty")
	}
	if customerName == "" {
		return nil, shared.NewDomainError("INVALID_CUSTOMER_NAME", "Customer name cannot be empty")
	}
	if amount.Amount().LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("INVALID_AMOUNT", "Amount must be positive")
	}
	if !paymentMethod.IsValid() {
		return nil, shared.NewDomainError("INVALID_PAYMENT_METHOD", "Payment method is not valid")
	}
	if receiptDate.IsZero() {
		return nil, shared.NewDomainError("INVALID_RECEIPT_DATE", "Receipt date is required")
	}

	rv := &ReceiptVoucher{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		VoucherNumber:       voucherNumber,
		CustomerID:          customerID,
		CustomerName:        customerName,
		Amount:              amount.Amount(),
		AllocatedAmount:     decimal.Zero,
		UnallocatedAmount:   amount.Amount(),
		PaymentMethod:       paymentMethod,
		Status:              VoucherStatusDraft,
		ReceiptDate:         receiptDate,
		Allocations:         make([]ReceivableAllocation, 0),
	}

	rv.AddDomainEvent(NewReceiptVoucherCreatedEvent(rv))

	return rv, nil
}

// Confirm confirms the receipt voucher, allowing allocations
func (rv *ReceiptVoucher) Confirm(confirmedBy uuid.UUID) error {
	if !rv.Status.CanConfirm() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot confirm voucher in %s status", rv.Status))
	}
	if confirmedBy == uuid.Nil {
		return shared.NewDomainError("INVALID_USER", "Confirming user ID is required")
	}

	now := time.Now()
	rv.Status = VoucherStatusConfirmed
	rv.ConfirmedAt = &now
	rv.ConfirmedBy = &confirmedBy
	rv.UpdatedAt = now
	rv.IncrementVersion()

	rv.AddDomainEvent(NewReceiptVoucherConfirmedEvent(rv))

	return nil
}

// AllocateToReceivable allocates part or all of the receipt to a receivable
// Returns the allocation record created
func (rv *ReceiptVoucher) AllocateToReceivable(
	receivableID uuid.UUID,
	receivableNumber string,
	amount valueobject.Money,
	remark string,
) (*ReceivableAllocation, error) {
	if !rv.Status.CanAllocate() {
		return nil, shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot allocate voucher in %s status", rv.Status))
	}
	if receivableID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_RECEIVABLE", "Receivable ID cannot be empty")
	}
	if receivableNumber == "" {
		return nil, shared.NewDomainError("INVALID_RECEIVABLE_NUMBER", "Receivable number is required")
	}
	if amount.Amount().LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("INVALID_AMOUNT", "Allocation amount must be positive")
	}
	if amount.Amount().GreaterThan(rv.UnallocatedAmount) {
		return nil, shared.NewDomainError("EXCEEDS_UNALLOCATED", fmt.Sprintf("Allocation amount %.2f exceeds unallocated amount %.2f", amount.Amount().InexactFloat64(), rv.UnallocatedAmount.InexactFloat64()))
	}

	// Check if already allocated to this receivable
	for _, alloc := range rv.Allocations {
		if alloc.ReceivableID == receivableID {
			return nil, shared.NewDomainError("ALREADY_ALLOCATED", fmt.Sprintf("Already allocated to receivable %s", receivableNumber))
		}
	}

	// Create allocation
	allocation := NewReceivableAllocation(rv.ID, receivableID, receivableNumber, amount, remark)
	rv.Allocations = append(rv.Allocations, *allocation)

	// Update amounts
	rv.AllocatedAmount = rv.AllocatedAmount.Add(amount.Amount())
	rv.UnallocatedAmount = rv.Amount.Sub(rv.AllocatedAmount)

	// Update status if fully allocated
	if rv.UnallocatedAmount.IsZero() {
		rv.Status = VoucherStatusAllocated
	}

	rv.UpdatedAt = time.Now()
	rv.IncrementVersion()

	rv.AddDomainEvent(NewReceiptVoucherAllocatedEvent(rv, allocation))

	return allocation, nil
}

// Cancel cancels the receipt voucher
// Only drafts and confirmed vouchers without allocations can be cancelled
func (rv *ReceiptVoucher) Cancel(cancelledBy uuid.UUID, reason string) error {
	if !rv.Status.CanCancel() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot cancel voucher in %s status", rv.Status))
	}
	if rv.AllocatedAmount.GreaterThan(decimal.Zero) {
		return shared.NewDomainError("HAS_ALLOCATIONS", "Cannot cancel voucher with existing allocations")
	}
	if cancelledBy == uuid.Nil {
		return shared.NewDomainError("INVALID_USER", "Cancelling user ID is required")
	}
	if reason == "" {
		return shared.NewDomainError("INVALID_REASON", "Cancel reason is required")
	}

	now := time.Now()
	previousStatus := rv.Status
	rv.Status = VoucherStatusCancelled
	rv.CancelledAt = &now
	rv.CancelledBy = &cancelledBy
	rv.CancelReason = reason
	rv.UpdatedAt = now
	rv.IncrementVersion()

	rv.AddDomainEvent(NewReceiptVoucherCancelledEvent(rv, previousStatus))

	return nil
}

// SetPaymentReference sets the payment reference
func (rv *ReceiptVoucher) SetPaymentReference(reference string) error {
	if rv.Status.IsTerminal() {
		return shared.NewDomainError("INVALID_STATE", "Cannot modify voucher in terminal state")
	}
	if len(reference) > 100 {
		return shared.NewDomainError("INVALID_REFERENCE", "Payment reference cannot exceed 100 characters")
	}

	rv.PaymentReference = reference
	rv.UpdatedAt = time.Now()
	rv.IncrementVersion()

	return nil
}

// SetRemark sets the remark
func (rv *ReceiptVoucher) SetRemark(remark string) error {
	if rv.Status.IsTerminal() {
		return shared.NewDomainError("INVALID_STATE", "Cannot modify voucher in terminal state")
	}

	rv.Remark = remark
	rv.UpdatedAt = time.Now()
	rv.IncrementVersion()

	return nil
}

// Helper methods

// GetAmountMoney returns total amount as Money
func (rv *ReceiptVoucher) GetAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(rv.Amount)
}

// GetAllocatedAmountMoney returns allocated amount as Money
func (rv *ReceiptVoucher) GetAllocatedAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(rv.AllocatedAmount)
}

// GetUnallocatedAmountMoney returns unallocated amount as Money
func (rv *ReceiptVoucher) GetUnallocatedAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(rv.UnallocatedAmount)
}

// IsDraft returns true if voucher is in draft status
func (rv *ReceiptVoucher) IsDraft() bool {
	return rv.Status == VoucherStatusDraft
}

// IsConfirmed returns true if voucher is confirmed
func (rv *ReceiptVoucher) IsConfirmed() bool {
	return rv.Status == VoucherStatusConfirmed
}

// IsAllocated returns true if voucher is fully allocated
func (rv *ReceiptVoucher) IsAllocated() bool {
	return rv.Status == VoucherStatusAllocated
}

// IsCancelled returns true if voucher is cancelled
func (rv *ReceiptVoucher) IsCancelled() bool {
	return rv.Status == VoucherStatusCancelled
}

// IsFullyAllocated returns true if all amount has been allocated
func (rv *ReceiptVoucher) IsFullyAllocated() bool {
	return rv.UnallocatedAmount.IsZero()
}

// AllocationCount returns the number of allocations
func (rv *ReceiptVoucher) AllocationCount() int {
	return len(rv.Allocations)
}

// AllocatedPercentage returns the percentage of total that has been allocated (0-100)
func (rv *ReceiptVoucher) AllocatedPercentage() decimal.Decimal {
	if rv.Amount.IsZero() {
		return decimal.NewFromInt(100)
	}
	return rv.AllocatedAmount.Div(rv.Amount).Mul(decimal.NewFromInt(100)).Round(2)
}

// GetAllocationByReceivableID returns the allocation for a specific receivable
func (rv *ReceiptVoucher) GetAllocationByReceivableID(receivableID uuid.UUID) *ReceivableAllocation {
	for i := range rv.Allocations {
		if rv.Allocations[i].ReceivableID == receivableID {
			return &rv.Allocations[i]
		}
	}
	return nil
}
