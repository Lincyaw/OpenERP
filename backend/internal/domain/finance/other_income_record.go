package finance

import (
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// IncomeCategory represents the category of other income
type IncomeCategory string

const (
	IncomeCategoryInvestment    IncomeCategory = "INVESTMENT"     // 投资收益
	IncomeCategorySubsidy       IncomeCategory = "SUBSIDY"        // 政府补贴
	IncomeCategoryInterest      IncomeCategory = "INTEREST"       // 利息收入
	IncomeCategoryRental        IncomeCategory = "RENTAL"         // 租金收入
	IncomeCategoryRefund        IncomeCategory = "REFUND"         // 退款收入
	IncomeCategoryCompensation  IncomeCategory = "COMPENSATION"   // 赔偿收入
	IncomeCategoryAssetDisposal IncomeCategory = "ASSET_DISPOSAL" // 资产处置收入
	IncomeCategoryOther         IncomeCategory = "OTHER"          // 其他收入
)

// IsValid checks if the category is a valid IncomeCategory
func (c IncomeCategory) IsValid() bool {
	switch c {
	case IncomeCategoryInvestment, IncomeCategorySubsidy, IncomeCategoryInterest,
		IncomeCategoryRental, IncomeCategoryRefund, IncomeCategoryCompensation,
		IncomeCategoryAssetDisposal, IncomeCategoryOther:
		return true
	}
	return false
}

// String returns the string representation of IncomeCategory
func (c IncomeCategory) String() string {
	return string(c)
}

// DisplayName returns a human-readable name for the category
func (c IncomeCategory) DisplayName() string {
	switch c {
	case IncomeCategoryInvestment:
		return "投资收益"
	case IncomeCategorySubsidy:
		return "政府补贴"
	case IncomeCategoryInterest:
		return "利息收入"
	case IncomeCategoryRental:
		return "租金收入"
	case IncomeCategoryRefund:
		return "退款收入"
	case IncomeCategoryCompensation:
		return "赔偿收入"
	case IncomeCategoryAssetDisposal:
		return "资产处置收入"
	case IncomeCategoryOther:
		return "其他收入"
	default:
		return string(c)
	}
}

// IncomeStatus represents the status of an income record
type IncomeStatus string

const (
	IncomeStatusDraft     IncomeStatus = "DRAFT"     // Draft, not yet confirmed
	IncomeStatusConfirmed IncomeStatus = "CONFIRMED" // Confirmed
	IncomeStatusCancelled IncomeStatus = "CANCELLED" // Cancelled
)

// IsValid checks if the status is a valid IncomeStatus
func (s IncomeStatus) IsValid() bool {
	switch s {
	case IncomeStatusDraft, IncomeStatusConfirmed, IncomeStatusCancelled:
		return true
	}
	return false
}

// String returns the string representation of IncomeStatus
func (s IncomeStatus) String() string {
	return string(s)
}

// IsTerminal returns true if the income is in a terminal state
func (s IncomeStatus) IsTerminal() bool {
	return s == IncomeStatusConfirmed || s == IncomeStatusCancelled
}

// CanConfirm returns true if the income can be confirmed
func (s IncomeStatus) CanConfirm() bool {
	return s == IncomeStatusDraft
}

// CanCancel returns true if the income can be cancelled
func (s IncomeStatus) CanCancel() bool {
	return s == IncomeStatusDraft
}

// ReceiptStatus represents whether the income has been received
type ReceiptStatus string

const (
	ReceiptStatusPending  ReceiptStatus = "PENDING"
	ReceiptStatusReceived ReceiptStatus = "RECEIVED"
)

// OtherIncomeRecord represents an other income record aggregate root
// It tracks non-trade income like investment returns, subsidies, interest, etc.
type OtherIncomeRecord struct {
	shared.TenantAggregateRoot
	IncomeNumber   string          `json:"income_number"`
	Category       IncomeCategory  `json:"category"`
	Amount         decimal.Decimal `json:"amount"`
	Description    string          `json:"description"`
	ReceivedAt     time.Time       `json:"received_at"` // When the income was received
	Status         IncomeStatus    `json:"status"`
	ReceiptStatus  ReceiptStatus   `json:"receipt_status"`
	PaymentMethod  *PaymentMethod  `json:"payment_method"`  // How the income was received
	ActualReceived *time.Time      `json:"actual_received"` // When the income was actually received
	Remark         string          `json:"remark"`
	AttachmentURLs string          `json:"attachment_urls"` // JSON array of attachment URLs
	ConfirmedAt    *time.Time      `json:"confirmed_at"`    // When confirmed
	ConfirmedBy    *uuid.UUID      `json:"confirmed_by"`    // User who confirmed
	CancelledAt    *time.Time      `json:"cancelled_at"`    // When cancelled
	CancelledBy    *uuid.UUID      `json:"cancelled_by"`    // User who cancelled
	CancelReason   string          `json:"cancel_reason"`
}

// NewOtherIncomeRecord creates a new other income record
func NewOtherIncomeRecord(
	tenantID uuid.UUID,
	incomeNumber string,
	category IncomeCategory,
	amount valueobject.Money,
	description string,
	receivedAt time.Time,
) (*OtherIncomeRecord, error) {
	// Validate inputs
	if incomeNumber == "" {
		return nil, shared.NewDomainError("INVALID_INCOME_NUMBER", "Income number cannot be empty")
	}
	if len(incomeNumber) > 50 {
		return nil, shared.NewDomainError("INVALID_INCOME_NUMBER", "Income number cannot exceed 50 characters")
	}
	if !category.IsValid() {
		return nil, shared.NewDomainError("INVALID_CATEGORY", "Income category is not valid")
	}
	if amount.Amount().LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("INVALID_AMOUNT", "Amount must be positive")
	}
	if description == "" {
		return nil, shared.NewDomainError("INVALID_DESCRIPTION", "Description cannot be empty")
	}
	if len(description) > 500 {
		return nil, shared.NewDomainError("INVALID_DESCRIPTION", "Description cannot exceed 500 characters")
	}

	income := &OtherIncomeRecord{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		IncomeNumber:        incomeNumber,
		Category:            category,
		Amount:              amount.Amount(),
		Description:         description,
		ReceivedAt:          receivedAt,
		Status:              IncomeStatusDraft,
		ReceiptStatus:       ReceiptStatusPending,
	}

	income.AddDomainEvent(NewOtherIncomeRecordCreatedEvent(income))

	return income, nil
}

// Confirm confirms the income record
func (i *OtherIncomeRecord) Confirm(confirmedBy uuid.UUID) error {
	if !i.Status.CanConfirm() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot confirm income in %s status", i.Status))
	}
	if confirmedBy == uuid.Nil {
		return shared.NewDomainError("INVALID_USER", "Confirmer user ID cannot be empty")
	}

	now := time.Now()
	i.Status = IncomeStatusConfirmed
	i.ConfirmedAt = &now
	i.ConfirmedBy = &confirmedBy
	i.UpdatedAt = now
	i.IncrementVersion()

	i.AddDomainEvent(NewOtherIncomeRecordConfirmedEvent(i))

	return nil
}

// Cancel cancels the income record
func (i *OtherIncomeRecord) Cancel(cancelledBy uuid.UUID, reason string) error {
	if !i.Status.CanCancel() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot cancel income in %s status", i.Status))
	}
	if cancelledBy == uuid.Nil {
		return shared.NewDomainError("INVALID_USER", "Canceller user ID cannot be empty")
	}
	if reason == "" {
		return shared.NewDomainError("INVALID_REASON", "Cancel reason is required")
	}

	now := time.Now()
	i.Status = IncomeStatusCancelled
	i.CancelledAt = &now
	i.CancelledBy = &cancelledBy
	i.CancelReason = reason
	i.UpdatedAt = now
	i.IncrementVersion()

	i.AddDomainEvent(NewOtherIncomeRecordCancelledEvent(i))

	return nil
}

// MarkAsReceived marks the income as actually received
func (i *OtherIncomeRecord) MarkAsReceived(paymentMethod PaymentMethod) error {
	if i.Status != IncomeStatusConfirmed {
		return shared.NewDomainError("INVALID_STATE", "Only confirmed income can be marked as received")
	}
	if i.ReceiptStatus == ReceiptStatusReceived {
		return shared.NewDomainError("ALREADY_RECEIVED", "Income is already received")
	}
	if !paymentMethod.IsValid() {
		return shared.NewDomainError("INVALID_PAYMENT_METHOD", "Payment method is not valid")
	}

	now := time.Now()
	i.ReceiptStatus = ReceiptStatusReceived
	i.PaymentMethod = &paymentMethod
	i.ActualReceived = &now
	i.UpdatedAt = now
	i.IncrementVersion()

	i.AddDomainEvent(NewOtherIncomeRecordReceivedEvent(i))

	return nil
}

// Update updates the income details (only allowed in draft status)
func (i *OtherIncomeRecord) Update(
	category IncomeCategory,
	amount valueobject.Money,
	description string,
	receivedAt time.Time,
) error {
	if i.Status != IncomeStatusDraft {
		return shared.NewDomainError("INVALID_STATE", "Can only update income in draft status")
	}
	if !category.IsValid() {
		return shared.NewDomainError("INVALID_CATEGORY", "Income category is not valid")
	}
	if amount.Amount().LessThanOrEqual(decimal.Zero) {
		return shared.NewDomainError("INVALID_AMOUNT", "Amount must be positive")
	}
	if description == "" {
		return shared.NewDomainError("INVALID_DESCRIPTION", "Description cannot be empty")
	}
	if len(description) > 500 {
		return shared.NewDomainError("INVALID_DESCRIPTION", "Description cannot exceed 500 characters")
	}

	i.Category = category
	i.Amount = amount.Amount()
	i.Description = description
	i.ReceivedAt = receivedAt
	i.UpdatedAt = time.Now()
	i.IncrementVersion()

	return nil
}

// SetRemark sets the remark
func (i *OtherIncomeRecord) SetRemark(remark string) {
	i.Remark = remark
	i.UpdatedAt = time.Now()
	i.IncrementVersion()
}

// SetAttachmentURLs sets the attachment URLs (JSON array)
func (i *OtherIncomeRecord) SetAttachmentURLs(urls string) {
	i.AttachmentURLs = urls
	i.UpdatedAt = time.Now()
	i.IncrementVersion()
}

// Helper methods

// GetAmountMoney returns amount as Money
func (i *OtherIncomeRecord) GetAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(i.Amount)
}

// IsDraft returns true if income is in draft status
func (i *OtherIncomeRecord) IsDraft() bool {
	return i.Status == IncomeStatusDraft
}

// IsConfirmed returns true if income is confirmed
func (i *OtherIncomeRecord) IsConfirmed() bool {
	return i.Status == IncomeStatusConfirmed
}

// IsCancelled returns true if income is cancelled
func (i *OtherIncomeRecord) IsCancelled() bool {
	return i.Status == IncomeStatusCancelled
}

// IsReceived returns true if income is received
func (i *OtherIncomeRecord) IsReceived() bool {
	return i.ReceiptStatus == ReceiptStatusReceived
}
