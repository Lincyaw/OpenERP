package finance

import (
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ReceivableStatus represents the status of an account receivable
type ReceivableStatus string

const (
	ReceivableStatusPending   ReceivableStatus = "PENDING"   // Unpaid, outstanding balance > 0
	ReceivableStatusPartial   ReceivableStatus = "PARTIAL"   // Partially paid, 0 < outstanding < total
	ReceivableStatusPaid      ReceivableStatus = "PAID"      // Fully paid, outstanding = 0
	ReceivableStatusReversed  ReceivableStatus = "REVERSED"  // Reversed/voided (e.g., return)
	ReceivableStatusCancelled ReceivableStatus = "CANCELLED" // Cancelled before any payment
)

// IsValid checks if the status is a valid ReceivableStatus
func (s ReceivableStatus) IsValid() bool {
	switch s {
	case ReceivableStatusPending, ReceivableStatusPartial, ReceivableStatusPaid,
		ReceivableStatusReversed, ReceivableStatusCancelled:
		return true
	}
	return false
}

// String returns the string representation of ReceivableStatus
func (s ReceivableStatus) String() string {
	return string(s)
}

// IsTerminal returns true if the receivable is in a terminal state
func (s ReceivableStatus) IsTerminal() bool {
	return s == ReceivableStatusPaid || s == ReceivableStatusReversed || s == ReceivableStatusCancelled
}

// CanApplyPayment returns true if payments can be applied in this status
func (s ReceivableStatus) CanApplyPayment() bool {
	return s == ReceivableStatusPending || s == ReceivableStatusPartial
}

// SourceType represents the type of source document that created the receivable
type SourceType string

const (
	SourceTypeSalesOrder  SourceType = "SALES_ORDER"
	SourceTypeSalesReturn SourceType = "SALES_RETURN" // Negative receivable (credit)
	SourceTypeManual      SourceType = "MANUAL"       // Manually created receivable
)

// IsValid checks if the source type is valid
func (s SourceType) IsValid() bool {
	switch s {
	case SourceTypeSalesOrder, SourceTypeSalesReturn, SourceTypeManual:
		return true
	}
	return false
}

// PaymentRecord represents a payment applied to the receivable
type PaymentRecord struct {
	ID               uuid.UUID       `gorm:"type:uuid;primary_key"`
	ReceivableID     uuid.UUID       `gorm:"type:uuid;not null;index"`
	ReceiptVoucherID uuid.UUID       `gorm:"type:uuid;not null;index"` // Reference to the receipt voucher
	Amount           decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	AppliedAt        time.Time       `gorm:"not null"`
	Remark           string          `gorm:"type:varchar(500)"`
}

// TableName returns the table name for GORM
func (PaymentRecord) TableName() string {
	return "receivable_payment_records"
}

// NewPaymentRecord creates a new payment record
func NewPaymentRecord(receivableID, voucherID uuid.UUID, amount valueobject.Money, remark string) *PaymentRecord {
	return &PaymentRecord{
		ID:               uuid.New(),
		ReceivableID:     receivableID,
		ReceiptVoucherID: voucherID,
		Amount:           amount.Amount(),
		AppliedAt:        time.Now(),
		Remark:           remark,
	}
}

// GetAmountMoney returns the amount as Money value object
func (p *PaymentRecord) GetAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(p.Amount)
}

// AccountReceivable represents an account receivable aggregate root
// It tracks money owed by a customer for goods/services provided
type AccountReceivable struct {
	shared.TenantAggregateRoot
	ReceivableNumber  string           `gorm:"type:varchar(50);not null;uniqueIndex:idx_receivable_tenant_number,priority:2"`
	CustomerID        uuid.UUID        `gorm:"type:uuid;not null;index"`
	CustomerName      string           `gorm:"type:varchar(200);not null"`
	SourceType        SourceType       `gorm:"type:varchar(30);not null;index"`
	SourceID          uuid.UUID        `gorm:"type:uuid;not null;index"`          // ID of the source document (e.g., SalesOrder)
	SourceNumber      string           `gorm:"type:varchar(50);not null"`         // Number of the source document
	TotalAmount       decimal.Decimal  `gorm:"type:decimal(18,4);not null"`       // Original amount due
	PaidAmount        decimal.Decimal  `gorm:"type:decimal(18,4);not null"`       // Amount already paid
	OutstandingAmount decimal.Decimal  `gorm:"type:decimal(18,4);not null;index"` // Remaining amount due
	Status            ReceivableStatus `gorm:"type:varchar(20);not null;default:'PENDING';index"`
	DueDate           *time.Time       `gorm:"index"` // When payment is expected
	PaymentRecords    []PaymentRecord  `gorm:"foreignKey:ReceivableID;references:ID"`
	Remark            string           `gorm:"type:text"`
	PaidAt            *time.Time       // When fully paid
	ReversedAt        *time.Time       // When reversed
	ReversalReason    string           `gorm:"type:varchar(500)"` // Reason for reversal
	CancelledAt       *time.Time       // When cancelled
	CancelReason      string           `gorm:"type:varchar(500)"` // Reason for cancellation
}

// TableName returns the table name for GORM
func (AccountReceivable) TableName() string {
	return "account_receivables"
}

// NewAccountReceivable creates a new account receivable
func NewAccountReceivable(
	tenantID uuid.UUID,
	receivableNumber string,
	customerID uuid.UUID,
	customerName string,
	sourceType SourceType,
	sourceID uuid.UUID,
	sourceNumber string,
	totalAmount valueobject.Money,
	dueDate *time.Time,
) (*AccountReceivable, error) {
	// Validate inputs
	if receivableNumber == "" {
		return nil, shared.NewDomainError("INVALID_RECEIVABLE_NUMBER", "Receivable number cannot be empty")
	}
	if len(receivableNumber) > 50 {
		return nil, shared.NewDomainError("INVALID_RECEIVABLE_NUMBER", "Receivable number cannot exceed 50 characters")
	}
	if customerID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_CUSTOMER", "Customer ID cannot be empty")
	}
	if customerName == "" {
		return nil, shared.NewDomainError("INVALID_CUSTOMER_NAME", "Customer name cannot be empty")
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

	ar := &AccountReceivable{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		ReceivableNumber:    receivableNumber,
		CustomerID:          customerID,
		CustomerName:        customerName,
		SourceType:          sourceType,
		SourceID:            sourceID,
		SourceNumber:        sourceNumber,
		TotalAmount:         totalAmount.Amount(),
		PaidAmount:          decimal.Zero,
		OutstandingAmount:   totalAmount.Amount(),
		Status:              ReceivableStatusPending,
		DueDate:             dueDate,
		PaymentRecords:      make([]PaymentRecord, 0),
	}

	ar.AddDomainEvent(NewAccountReceivableCreatedEvent(ar))

	return ar, nil
}

// ApplyPayment applies a payment to the receivable
// Returns error if payment exceeds outstanding amount or receivable is in terminal state
func (ar *AccountReceivable) ApplyPayment(amount valueobject.Money, voucherID uuid.UUID, remark string) error {
	if !ar.Status.CanApplyPayment() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot apply payment to receivable in %s status", ar.Status))
	}
	if amount.Amount().LessThanOrEqual(decimal.Zero) {
		return shared.NewDomainError("INVALID_AMOUNT", "Payment amount must be positive")
	}
	if amount.Amount().GreaterThan(ar.OutstandingAmount) {
		return shared.NewDomainError("EXCEEDS_OUTSTANDING", fmt.Sprintf("Payment amount %.2f exceeds outstanding amount %.2f", amount.Amount().InexactFloat64(), ar.OutstandingAmount.InexactFloat64()))
	}
	if voucherID == uuid.Nil {
		return shared.NewDomainError("INVALID_VOUCHER", "Receipt voucher ID cannot be empty")
	}

	// Create payment record
	record := NewPaymentRecord(ar.ID, voucherID, amount, remark)
	ar.PaymentRecords = append(ar.PaymentRecords, *record)

	// Update amounts
	ar.PaidAmount = ar.PaidAmount.Add(amount.Amount())
	ar.OutstandingAmount = ar.TotalAmount.Sub(ar.PaidAmount)

	// Update status based on outstanding amount
	if ar.OutstandingAmount.IsZero() {
		now := time.Now()
		ar.Status = ReceivableStatusPaid
		ar.PaidAt = &now
		ar.AddDomainEvent(NewAccountReceivablePaidEvent(ar))
	} else {
		ar.Status = ReceivableStatusPartial
		ar.AddDomainEvent(NewAccountReceivablePartiallyPaidEvent(ar, amount))
	}

	ar.UpdatedAt = time.Now()
	ar.IncrementVersion()

	return nil
}

// Reverse reverses the receivable (e.g., due to sales return)
// This creates a credit/negative adjustment
func (ar *AccountReceivable) Reverse(reason string) error {
	if ar.Status.IsTerminal() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot reverse receivable in %s status", ar.Status))
	}
	if reason == "" {
		return shared.NewDomainError("INVALID_REASON", "Reversal reason is required")
	}

	now := time.Now()
	previousStatus := ar.Status
	ar.Status = ReceivableStatusReversed
	ar.ReversedAt = &now
	ar.ReversalReason = reason
	ar.UpdatedAt = now
	ar.IncrementVersion()

	ar.AddDomainEvent(NewAccountReceivableReversedEvent(ar, previousStatus))

	return nil
}

// Cancel cancels the receivable (only if no payments have been applied)
func (ar *AccountReceivable) Cancel(reason string) error {
	if ar.Status.IsTerminal() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot cancel receivable in %s status", ar.Status))
	}
	if ar.Status == ReceivableStatusPartial || ar.PaidAmount.GreaterThan(decimal.Zero) {
		return shared.NewDomainError("HAS_PAYMENTS", "Cannot cancel receivable with existing payments")
	}
	if reason == "" {
		return shared.NewDomainError("INVALID_REASON", "Cancel reason is required")
	}

	now := time.Now()
	ar.Status = ReceivableStatusCancelled
	ar.CancelledAt = &now
	ar.CancelReason = reason
	ar.OutstandingAmount = decimal.Zero // No longer outstanding
	ar.UpdatedAt = now
	ar.IncrementVersion()

	ar.AddDomainEvent(NewAccountReceivableCancelledEvent(ar))

	return nil
}

// SetDueDate updates the due date
func (ar *AccountReceivable) SetDueDate(dueDate *time.Time) error {
	if ar.Status.IsTerminal() {
		return shared.NewDomainError("INVALID_STATE", "Cannot modify due date for receivable in terminal state")
	}

	ar.DueDate = dueDate
	ar.UpdatedAt = time.Now()
	ar.IncrementVersion()

	return nil
}

// SetRemark sets the remark
func (ar *AccountReceivable) SetRemark(remark string) {
	ar.Remark = remark
	ar.UpdatedAt = time.Now()
	ar.IncrementVersion()
}

// Helper methods

// GetTotalAmountMoney returns total amount as Money
func (ar *AccountReceivable) GetTotalAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(ar.TotalAmount)
}

// GetPaidAmountMoney returns paid amount as Money
func (ar *AccountReceivable) GetPaidAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(ar.PaidAmount)
}

// GetOutstandingAmountMoney returns outstanding amount as Money
func (ar *AccountReceivable) GetOutstandingAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(ar.OutstandingAmount)
}

// IsPending returns true if receivable is pending
func (ar *AccountReceivable) IsPending() bool {
	return ar.Status == ReceivableStatusPending
}

// IsPartial returns true if receivable is partially paid
func (ar *AccountReceivable) IsPartial() bool {
	return ar.Status == ReceivableStatusPartial
}

// IsPaid returns true if receivable is fully paid
func (ar *AccountReceivable) IsPaid() bool {
	return ar.Status == ReceivableStatusPaid
}

// IsReversed returns true if receivable is reversed
func (ar *AccountReceivable) IsReversed() bool {
	return ar.Status == ReceivableStatusReversed
}

// IsCancelled returns true if receivable is cancelled
func (ar *AccountReceivable) IsCancelled() bool {
	return ar.Status == ReceivableStatusCancelled
}

// IsOverdue returns true if the receivable is past due date and not paid
func (ar *AccountReceivable) IsOverdue() bool {
	if ar.Status.IsTerminal() {
		return false
	}
	if ar.DueDate == nil {
		return false
	}
	return time.Now().After(*ar.DueDate)
}

// DaysOverdue returns the number of days past due (0 if not overdue)
func (ar *AccountReceivable) DaysOverdue() int {
	if !ar.IsOverdue() {
		return 0
	}
	return int(time.Since(*ar.DueDate).Hours() / 24)
}

// PaymentCount returns the number of payments applied
func (ar *AccountReceivable) PaymentCount() int {
	return len(ar.PaymentRecords)
}

// PaidPercentage returns the percentage of total that has been paid (0-100)
func (ar *AccountReceivable) PaidPercentage() decimal.Decimal {
	if ar.TotalAmount.IsZero() {
		return decimal.NewFromInt(100)
	}
	return ar.PaidAmount.Div(ar.TotalAmount).Mul(decimal.NewFromInt(100)).Round(2)
}
