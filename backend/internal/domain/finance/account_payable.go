package finance

import (
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PayableStatus represents the status of an account payable
type PayableStatus string

const (
	PayableStatusPending   PayableStatus = "PENDING"   // Unpaid, outstanding balance > 0
	PayableStatusPartial   PayableStatus = "PARTIAL"   // Partially paid, 0 < outstanding < total
	PayableStatusPaid      PayableStatus = "PAID"      // Fully paid, outstanding = 0
	PayableStatusReversed  PayableStatus = "REVERSED"  // Reversed/voided (e.g., purchase return)
	PayableStatusCancelled PayableStatus = "CANCELLED" // Cancelled before any payment
)

// IsValid checks if the status is a valid PayableStatus
func (s PayableStatus) IsValid() bool {
	switch s {
	case PayableStatusPending, PayableStatusPartial, PayableStatusPaid,
		PayableStatusReversed, PayableStatusCancelled:
		return true
	}
	return false
}

// String returns the string representation of PayableStatus
func (s PayableStatus) String() string {
	return string(s)
}

// IsTerminal returns true if the payable is in a terminal state
func (s PayableStatus) IsTerminal() bool {
	return s == PayableStatusPaid || s == PayableStatusReversed || s == PayableStatusCancelled
}

// CanApplyPayment returns true if payments can be applied in this status
func (s PayableStatus) CanApplyPayment() bool {
	return s == PayableStatusPending || s == PayableStatusPartial
}

// PayableSourceType represents the type of source document that created the payable
type PayableSourceType string

const (
	PayableSourceTypePurchaseOrder  PayableSourceType = "PURCHASE_ORDER"
	PayableSourceTypePurchaseReturn PayableSourceType = "PURCHASE_RETURN" // Negative payable (debit)
	PayableSourceTypeManual         PayableSourceType = "MANUAL"          // Manually created payable
)

// IsValid checks if the source type is valid
func (s PayableSourceType) IsValid() bool {
	switch s {
	case PayableSourceTypePurchaseOrder, PayableSourceTypePurchaseReturn, PayableSourceTypeManual:
		return true
	}
	return false
}

// PayablePaymentRecord represents a payment made for the payable
type PayablePaymentRecord struct {
	ID               uuid.UUID       `gorm:"type:uuid;primary_key"`
	PayableID        uuid.UUID       `gorm:"type:uuid;not null;index"`
	PaymentVoucherID uuid.UUID       `gorm:"type:uuid;not null;index"` // Reference to the payment voucher
	Amount           decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	AppliedAt        time.Time       `gorm:"not null"`
	Remark           string          `gorm:"type:varchar(500)"`
}

// TableName returns the table name for GORM
func (PayablePaymentRecord) TableName() string {
	return "payable_payment_records"
}

// NewPayablePaymentRecord creates a new payment record
func NewPayablePaymentRecord(payableID, voucherID uuid.UUID, amount valueobject.Money, remark string) *PayablePaymentRecord {
	return &PayablePaymentRecord{
		ID:               uuid.New(),
		PayableID:        payableID,
		PaymentVoucherID: voucherID,
		Amount:           amount.Amount(),
		AppliedAt:        time.Now(),
		Remark:           remark,
	}
}

// GetAmountMoney returns the amount as Money value object
func (p *PayablePaymentRecord) GetAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(p.Amount)
}

// AccountPayable represents an account payable aggregate root
// It tracks money owed to a supplier for goods/services received
type AccountPayable struct {
	shared.TenantAggregateRoot
	PayableNumber     string                 `gorm:"type:varchar(50);not null;uniqueIndex:idx_payable_tenant_number,priority:2"`
	SupplierID        uuid.UUID              `gorm:"type:uuid;not null;index"`
	SupplierName      string                 `gorm:"type:varchar(200);not null"`
	SourceType        PayableSourceType      `gorm:"type:varchar(30);not null;index"`
	SourceID          uuid.UUID              `gorm:"type:uuid;not null;index"`          // ID of the source document (e.g., PurchaseOrder)
	SourceNumber      string                 `gorm:"type:varchar(50);not null"`         // Number of the source document
	TotalAmount       decimal.Decimal        `gorm:"type:decimal(18,4);not null"`       // Original amount due
	PaidAmount        decimal.Decimal        `gorm:"type:decimal(18,4);not null"`       // Amount already paid
	OutstandingAmount decimal.Decimal        `gorm:"type:decimal(18,4);not null;index"` // Remaining amount due
	Status            PayableStatus          `gorm:"type:varchar(20);not null;default:'PENDING';index"`
	DueDate           *time.Time             `gorm:"index"` // When payment is expected
	PaymentRecords    []PayablePaymentRecord `gorm:"foreignKey:PayableID;references:ID"`
	Remark            string                 `gorm:"type:text"`
	PaidAt            *time.Time             // When fully paid
	ReversedAt        *time.Time             // When reversed
	ReversalReason    string                 `gorm:"type:varchar(500)"` // Reason for reversal
	CancelledAt       *time.Time             // When cancelled
	CancelReason      string                 `gorm:"type:varchar(500)"` // Reason for cancellation
}

// TableName returns the table name for GORM
func (AccountPayable) TableName() string {
	return "account_payables"
}

// NewAccountPayable creates a new account payable
func NewAccountPayable(
	tenantID uuid.UUID,
	payableNumber string,
	supplierID uuid.UUID,
	supplierName string,
	sourceType PayableSourceType,
	sourceID uuid.UUID,
	sourceNumber string,
	totalAmount valueobject.Money,
	dueDate *time.Time,
) (*AccountPayable, error) {
	// Validate inputs
	if payableNumber == "" {
		return nil, shared.NewDomainError("INVALID_PAYABLE_NUMBER", "Payable number cannot be empty")
	}
	if len(payableNumber) > 50 {
		return nil, shared.NewDomainError("INVALID_PAYABLE_NUMBER", "Payable number cannot exceed 50 characters")
	}
	if supplierID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_SUPPLIER", "Supplier ID cannot be empty")
	}
	if supplierName == "" {
		return nil, shared.NewDomainError("INVALID_SUPPLIER_NAME", "Supplier name cannot be empty")
	}
	if !sourceType.IsValid() {
		return nil, shared.NewDomainError("INVALID_SOURCE_TYPE", "Source type is not valid")
	}
	if sourceID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_SOURCE_ID", "Source ID cannot be empty")
	}
	if sourceNumber == "" {
		return nil, shared.NewDomainError("INVALID_SOURCE_NUMBER", "Source number cannot be empty")
	}
	if totalAmount.Amount().LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("INVALID_AMOUNT", "Total amount must be positive")
	}

	ap := &AccountPayable{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		PayableNumber:       payableNumber,
		SupplierID:          supplierID,
		SupplierName:        supplierName,
		SourceType:          sourceType,
		SourceID:            sourceID,
		SourceNumber:        sourceNumber,
		TotalAmount:         totalAmount.Amount(),
		PaidAmount:          decimal.Zero,
		OutstandingAmount:   totalAmount.Amount(),
		Status:              PayableStatusPending,
		DueDate:             dueDate,
		PaymentRecords:      make([]PayablePaymentRecord, 0),
	}

	ap.AddDomainEvent(NewAccountPayableCreatedEvent(ap))

	return ap, nil
}

// ApplyPayment applies a payment to the payable
// Returns error if payment exceeds outstanding amount or payable is in terminal state
func (ap *AccountPayable) ApplyPayment(amount valueobject.Money, voucherID uuid.UUID, remark string) error {
	if !ap.Status.CanApplyPayment() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot apply payment to payable in %s status", ap.Status))
	}
	if amount.Amount().LessThanOrEqual(decimal.Zero) {
		return shared.NewDomainError("INVALID_AMOUNT", "Payment amount must be positive")
	}
	if amount.Amount().GreaterThan(ap.OutstandingAmount) {
		return shared.NewDomainError("EXCEEDS_OUTSTANDING", fmt.Sprintf("Payment amount %.2f exceeds outstanding amount %.2f", amount.Amount().InexactFloat64(), ap.OutstandingAmount.InexactFloat64()))
	}
	if voucherID == uuid.Nil {
		return shared.NewDomainError("INVALID_VOUCHER", "Payment voucher ID cannot be empty")
	}

	// Create payment record
	record := NewPayablePaymentRecord(ap.ID, voucherID, amount, remark)
	ap.PaymentRecords = append(ap.PaymentRecords, *record)

	// Update amounts
	ap.PaidAmount = ap.PaidAmount.Add(amount.Amount())
	ap.OutstandingAmount = ap.TotalAmount.Sub(ap.PaidAmount)

	// Update status based on outstanding amount
	if ap.OutstandingAmount.IsZero() {
		now := time.Now()
		ap.Status = PayableStatusPaid
		ap.PaidAt = &now
		ap.AddDomainEvent(NewAccountPayablePaidEvent(ap))
	} else {
		ap.Status = PayableStatusPartial
		ap.AddDomainEvent(NewAccountPayablePartiallyPaidEvent(ap, amount))
	}

	ap.UpdatedAt = time.Now()
	ap.IncrementVersion()

	return nil
}

// Reverse reverses the payable (e.g., due to purchase return)
// This creates a debit/negative adjustment
func (ap *AccountPayable) Reverse(reason string) error {
	if ap.Status.IsTerminal() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot reverse payable in %s status", ap.Status))
	}
	if reason == "" {
		return shared.NewDomainError("INVALID_REASON", "Reversal reason is required")
	}

	now := time.Now()
	previousStatus := ap.Status
	ap.Status = PayableStatusReversed
	ap.ReversedAt = &now
	ap.ReversalReason = reason
	ap.OutstandingAmount = decimal.Zero // No longer outstanding after reversal
	ap.UpdatedAt = now
	ap.IncrementVersion()

	ap.AddDomainEvent(NewAccountPayableReversedEvent(ap, previousStatus))

	return nil
}

// Cancel cancels the payable (only if no payments have been applied)
func (ap *AccountPayable) Cancel(reason string) error {
	if ap.Status.IsTerminal() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot cancel payable in %s status", ap.Status))
	}
	if ap.Status == PayableStatusPartial || ap.PaidAmount.GreaterThan(decimal.Zero) {
		return shared.NewDomainError("HAS_PAYMENTS", "Cannot cancel payable with existing payments")
	}
	if reason == "" {
		return shared.NewDomainError("INVALID_REASON", "Cancel reason is required")
	}

	now := time.Now()
	ap.Status = PayableStatusCancelled
	ap.CancelledAt = &now
	ap.CancelReason = reason
	ap.OutstandingAmount = decimal.Zero // No longer outstanding
	ap.UpdatedAt = now
	ap.IncrementVersion()

	ap.AddDomainEvent(NewAccountPayableCancelledEvent(ap))

	return nil
}

// SetDueDate updates the due date
func (ap *AccountPayable) SetDueDate(dueDate *time.Time) error {
	if ap.Status.IsTerminal() {
		return shared.NewDomainError("INVALID_STATE", "Cannot modify due date for payable in terminal state")
	}

	ap.DueDate = dueDate
	ap.UpdatedAt = time.Now()
	ap.IncrementVersion()

	return nil
}

// SetRemark sets the remark
func (ap *AccountPayable) SetRemark(remark string) {
	ap.Remark = remark
	ap.UpdatedAt = time.Now()
	ap.IncrementVersion()
}

// Helper methods

// GetTotalAmountMoney returns total amount as Money
func (ap *AccountPayable) GetTotalAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(ap.TotalAmount)
}

// GetPaidAmountMoney returns paid amount as Money
func (ap *AccountPayable) GetPaidAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(ap.PaidAmount)
}

// GetOutstandingAmountMoney returns outstanding amount as Money
func (ap *AccountPayable) GetOutstandingAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(ap.OutstandingAmount)
}

// IsPending returns true if payable is pending
func (ap *AccountPayable) IsPending() bool {
	return ap.Status == PayableStatusPending
}

// IsPartial returns true if payable is partially paid
func (ap *AccountPayable) IsPartial() bool {
	return ap.Status == PayableStatusPartial
}

// IsPaid returns true if payable is fully paid
func (ap *AccountPayable) IsPaid() bool {
	return ap.Status == PayableStatusPaid
}

// IsReversed returns true if payable is reversed
func (ap *AccountPayable) IsReversed() bool {
	return ap.Status == PayableStatusReversed
}

// IsCancelled returns true if payable is cancelled
func (ap *AccountPayable) IsCancelled() bool {
	return ap.Status == PayableStatusCancelled
}

// IsOverdue returns true if the payable is past due date and not paid
func (ap *AccountPayable) IsOverdue() bool {
	if ap.Status.IsTerminal() {
		return false
	}
	if ap.DueDate == nil {
		return false
	}
	return time.Now().After(*ap.DueDate)
}

// DaysOverdue returns the number of days past due (0 if not overdue)
func (ap *AccountPayable) DaysOverdue() int {
	if !ap.IsOverdue() {
		return 0
	}
	return int(time.Since(*ap.DueDate).Hours() / 24)
}

// PaymentCount returns the number of payments applied
func (ap *AccountPayable) PaymentCount() int {
	return len(ap.PaymentRecords)
}

// PaidPercentage returns the percentage of total that has been paid (0-100)
func (ap *AccountPayable) PaidPercentage() decimal.Decimal {
	if ap.TotalAmount.IsZero() {
		return decimal.NewFromInt(100)
	}
	return ap.PaidAmount.Div(ap.TotalAmount).Mul(decimal.NewFromInt(100)).Round(2)
}
