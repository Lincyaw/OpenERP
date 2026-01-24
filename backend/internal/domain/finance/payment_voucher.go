package finance

import (
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PayableAllocation represents the allocation of a payment voucher to a payable
type PayableAllocation struct {
	ID               uuid.UUID       `gorm:"type:uuid;primary_key"`
	PaymentVoucherID uuid.UUID       `gorm:"type:uuid;not null;index"`
	PayableID        uuid.UUID       `gorm:"type:uuid;not null;index"`
	PayableNumber    string          `gorm:"type:varchar(50);not null"` // Denormalized for display
	Amount           decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	AllocatedAt      time.Time       `gorm:"not null"`
	Remark           string          `gorm:"type:varchar(500)"`
}

// TableName returns the table name for GORM
func (PayableAllocation) TableName() string {
	return "payable_allocations"
}

// NewPayableAllocation creates a new payable allocation
func NewPayableAllocation(voucherID, payableID uuid.UUID, payableNumber string, amount valueobject.Money, remark string) *PayableAllocation {
	return &PayableAllocation{
		ID:               uuid.New(),
		PaymentVoucherID: voucherID,
		PayableID:        payableID,
		PayableNumber:    payableNumber,
		Amount:           amount.Amount(),
		AllocatedAt:      time.Now(),
		Remark:           remark,
	}
}

// GetAmountMoney returns the amount as Money value object
func (a *PayableAllocation) GetAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(a.Amount)
}

// PaymentVoucher represents a payment voucher aggregate root
// It records a payment made to a supplier
type PaymentVoucher struct {
	shared.TenantAggregateRoot
	VoucherNumber     string              `gorm:"type:varchar(50);not null;uniqueIndex:idx_payment_tenant_number,priority:2"`
	SupplierID        uuid.UUID           `gorm:"type:uuid;not null;index"`
	SupplierName      string              `gorm:"type:varchar(200);not null"`
	Amount            decimal.Decimal     `gorm:"type:decimal(18,4);not null"`   // Total payment amount
	AllocatedAmount   decimal.Decimal     `gorm:"type:decimal(18,4);not null"`   // Amount allocated to payables
	UnallocatedAmount decimal.Decimal     `gorm:"type:decimal(18,4);not null"`   // Remaining unallocated amount
	PaymentMethod     PaymentMethod       `gorm:"type:varchar(30);not null"`     // Payment method
	PaymentReference  string              `gorm:"type:varchar(100)"`             // Reference (bank txn, check #)
	Status            VoucherStatus       `gorm:"type:varchar(20);not null;default:'DRAFT';index"`
	PaymentDate       time.Time           `gorm:"not null"`                      // When payment was made
	Allocations       []PayableAllocation `gorm:"foreignKey:PaymentVoucherID;references:ID"`
	Remark            string              `gorm:"type:text"`
	ConfirmedAt       *time.Time          // When confirmed
	ConfirmedBy       *uuid.UUID          `gorm:"type:uuid"` // User who confirmed
	CancelledAt       *time.Time          // When cancelled
	CancelledBy       *uuid.UUID          `gorm:"type:uuid"` // User who cancelled
	CancelReason      string              `gorm:"type:varchar(500)"`
}

// TableName returns the table name for GORM
func (PaymentVoucher) TableName() string {
	return "payment_vouchers"
}

// NewPaymentVoucher creates a new payment voucher
func NewPaymentVoucher(
	tenantID uuid.UUID,
	voucherNumber string,
	supplierID uuid.UUID,
	supplierName string,
	amount valueobject.Money,
	paymentMethod PaymentMethod,
	paymentDate time.Time,
) (*PaymentVoucher, error) {
	// Validate inputs
	if voucherNumber == "" {
		return nil, shared.NewDomainError("INVALID_VOUCHER_NUMBER", "Voucher number cannot be empty")
	}
	if len(voucherNumber) > 50 {
		return nil, shared.NewDomainError("INVALID_VOUCHER_NUMBER", "Voucher number cannot exceed 50 characters")
	}
	if supplierID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_SUPPLIER", "Supplier ID cannot be empty")
	}
	if supplierName == "" {
		return nil, shared.NewDomainError("INVALID_SUPPLIER_NAME", "Supplier name cannot be empty")
	}
	if amount.Amount().LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("INVALID_AMOUNT", "Amount must be positive")
	}
	if !paymentMethod.IsValid() {
		return nil, shared.NewDomainError("INVALID_PAYMENT_METHOD", "Payment method is not valid")
	}
	if paymentDate.IsZero() {
		return nil, shared.NewDomainError("INVALID_PAYMENT_DATE", "Payment date is required")
	}

	pv := &PaymentVoucher{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		VoucherNumber:       voucherNumber,
		SupplierID:          supplierID,
		SupplierName:        supplierName,
		Amount:              amount.Amount(),
		AllocatedAmount:     decimal.Zero,
		UnallocatedAmount:   amount.Amount(),
		PaymentMethod:       paymentMethod,
		Status:              VoucherStatusDraft,
		PaymentDate:         paymentDate,
		Allocations:         make([]PayableAllocation, 0),
	}

	pv.AddDomainEvent(NewPaymentVoucherCreatedEvent(pv))

	return pv, nil
}

// Confirm confirms the payment voucher, allowing allocations
func (pv *PaymentVoucher) Confirm(confirmedBy uuid.UUID) error {
	if !pv.Status.CanConfirm() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot confirm voucher in %s status", pv.Status))
	}
	if confirmedBy == uuid.Nil {
		return shared.NewDomainError("INVALID_USER", "Confirming user ID is required")
	}

	now := time.Now()
	pv.Status = VoucherStatusConfirmed
	pv.ConfirmedAt = &now
	pv.ConfirmedBy = &confirmedBy
	pv.UpdatedAt = now
	pv.IncrementVersion()

	pv.AddDomainEvent(NewPaymentVoucherConfirmedEvent(pv))

	return nil
}

// AllocateToPayable allocates part or all of the payment to a payable
// Returns the allocation record created
func (pv *PaymentVoucher) AllocateToPayable(
	payableID uuid.UUID,
	payableNumber string,
	amount valueobject.Money,
	remark string,
) (*PayableAllocation, error) {
	if !pv.Status.CanAllocate() {
		return nil, shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot allocate voucher in %s status", pv.Status))
	}
	if payableID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_PAYABLE", "Payable ID cannot be empty")
	}
	if payableNumber == "" {
		return nil, shared.NewDomainError("INVALID_PAYABLE_NUMBER", "Payable number is required")
	}
	if amount.Amount().LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("INVALID_AMOUNT", "Allocation amount must be positive")
	}
	if amount.Amount().GreaterThan(pv.UnallocatedAmount) {
		return nil, shared.NewDomainError("EXCEEDS_UNALLOCATED", fmt.Sprintf("Allocation amount %.2f exceeds unallocated amount %.2f", amount.Amount().InexactFloat64(), pv.UnallocatedAmount.InexactFloat64()))
	}

	// Check if already allocated to this payable
	for _, alloc := range pv.Allocations {
		if alloc.PayableID == payableID {
			return nil, shared.NewDomainError("ALREADY_ALLOCATED", fmt.Sprintf("Already allocated to payable %s", payableNumber))
		}
	}

	// Create allocation
	allocation := NewPayableAllocation(pv.ID, payableID, payableNumber, amount, remark)
	pv.Allocations = append(pv.Allocations, *allocation)

	// Update amounts
	pv.AllocatedAmount = pv.AllocatedAmount.Add(amount.Amount())
	pv.UnallocatedAmount = pv.Amount.Sub(pv.AllocatedAmount)

	// Update status if fully allocated
	if pv.UnallocatedAmount.IsZero() {
		pv.Status = VoucherStatusAllocated
	}

	pv.UpdatedAt = time.Now()
	pv.IncrementVersion()

	pv.AddDomainEvent(NewPaymentVoucherAllocatedEvent(pv, allocation))

	return allocation, nil
}

// Cancel cancels the payment voucher
// Only drafts and confirmed vouchers without allocations can be cancelled
func (pv *PaymentVoucher) Cancel(cancelledBy uuid.UUID, reason string) error {
	if !pv.Status.CanCancel() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot cancel voucher in %s status", pv.Status))
	}
	if pv.AllocatedAmount.GreaterThan(decimal.Zero) {
		return shared.NewDomainError("HAS_ALLOCATIONS", "Cannot cancel voucher with existing allocations")
	}
	if cancelledBy == uuid.Nil {
		return shared.NewDomainError("INVALID_USER", "Cancelling user ID is required")
	}
	if reason == "" {
		return shared.NewDomainError("INVALID_REASON", "Cancel reason is required")
	}

	now := time.Now()
	previousStatus := pv.Status
	pv.Status = VoucherStatusCancelled
	pv.CancelledAt = &now
	pv.CancelledBy = &cancelledBy
	pv.CancelReason = reason
	pv.UpdatedAt = now
	pv.IncrementVersion()

	pv.AddDomainEvent(NewPaymentVoucherCancelledEvent(pv, previousStatus))

	return nil
}

// SetPaymentReference sets the payment reference
func (pv *PaymentVoucher) SetPaymentReference(reference string) error {
	if pv.Status.IsTerminal() {
		return shared.NewDomainError("INVALID_STATE", "Cannot modify voucher in terminal state")
	}
	if len(reference) > 100 {
		return shared.NewDomainError("INVALID_REFERENCE", "Payment reference cannot exceed 100 characters")
	}

	pv.PaymentReference = reference
	pv.UpdatedAt = time.Now()
	pv.IncrementVersion()

	return nil
}

// SetRemark sets the remark
func (pv *PaymentVoucher) SetRemark(remark string) error {
	if pv.Status.IsTerminal() {
		return shared.NewDomainError("INVALID_STATE", "Cannot modify voucher in terminal state")
	}

	pv.Remark = remark
	pv.UpdatedAt = time.Now()
	pv.IncrementVersion()

	return nil
}

// Helper methods

// GetAmountMoney returns total amount as Money
func (pv *PaymentVoucher) GetAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(pv.Amount)
}

// GetAllocatedAmountMoney returns allocated amount as Money
func (pv *PaymentVoucher) GetAllocatedAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(pv.AllocatedAmount)
}

// GetUnallocatedAmountMoney returns unallocated amount as Money
func (pv *PaymentVoucher) GetUnallocatedAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(pv.UnallocatedAmount)
}

// IsDraft returns true if voucher is in draft status
func (pv *PaymentVoucher) IsDraft() bool {
	return pv.Status == VoucherStatusDraft
}

// IsConfirmed returns true if voucher is confirmed
func (pv *PaymentVoucher) IsConfirmed() bool {
	return pv.Status == VoucherStatusConfirmed
}

// IsAllocated returns true if voucher is fully allocated
func (pv *PaymentVoucher) IsAllocated() bool {
	return pv.Status == VoucherStatusAllocated
}

// IsCancelled returns true if voucher is cancelled
func (pv *PaymentVoucher) IsCancelled() bool {
	return pv.Status == VoucherStatusCancelled
}

// IsFullyAllocated returns true if all amount has been allocated
func (pv *PaymentVoucher) IsFullyAllocated() bool {
	return pv.UnallocatedAmount.IsZero()
}

// AllocationCount returns the number of allocations
func (pv *PaymentVoucher) AllocationCount() int {
	return len(pv.Allocations)
}

// AllocatedPercentage returns the percentage of total that has been allocated (0-100)
func (pv *PaymentVoucher) AllocatedPercentage() decimal.Decimal {
	if pv.Amount.IsZero() {
		return decimal.NewFromInt(100)
	}
	return pv.AllocatedAmount.Div(pv.Amount).Mul(decimal.NewFromInt(100)).Round(2)
}

// GetAllocationByPayableID returns the allocation for a specific payable
func (pv *PaymentVoucher) GetAllocationByPayableID(payableID uuid.UUID) *PayableAllocation {
	for i := range pv.Allocations {
		if pv.Allocations[i].PayableID == payableID {
			return &pv.Allocations[i]
		}
	}
	return nil
}
