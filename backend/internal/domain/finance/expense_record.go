package finance

import (
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ExpenseCategory represents the category of an expense
type ExpenseCategory string

const (
	ExpenseCategoryRent        ExpenseCategory = "RENT"        // 房租
	ExpenseCategoryUtilities   ExpenseCategory = "UTILITIES"   // 水电费
	ExpenseCategorySalary      ExpenseCategory = "SALARY"      // 工资
	ExpenseCategoryOffice      ExpenseCategory = "OFFICE"      // 办公费
	ExpenseCategoryTravel      ExpenseCategory = "TRAVEL"      // 差旅费
	ExpenseCategoryMarketing   ExpenseCategory = "MARKETING"   // 市场营销
	ExpenseCategoryEquipment   ExpenseCategory = "EQUIPMENT"   // 设备费
	ExpenseCategoryMaintenance ExpenseCategory = "MAINTENANCE" // 维修费
	ExpenseCategoryInsurance   ExpenseCategory = "INSURANCE"   // 保险费
	ExpenseCategoryTax         ExpenseCategory = "TAX"         // 税费
	ExpenseCategoryOther       ExpenseCategory = "OTHER"       // 其他费用
)

// IsValid checks if the category is a valid ExpenseCategory
func (c ExpenseCategory) IsValid() bool {
	switch c {
	case ExpenseCategoryRent, ExpenseCategoryUtilities, ExpenseCategorySalary,
		ExpenseCategoryOffice, ExpenseCategoryTravel, ExpenseCategoryMarketing,
		ExpenseCategoryEquipment, ExpenseCategoryMaintenance, ExpenseCategoryInsurance,
		ExpenseCategoryTax, ExpenseCategoryOther:
		return true
	}
	return false
}

// String returns the string representation of ExpenseCategory
func (c ExpenseCategory) String() string {
	return string(c)
}

// DisplayName returns a human-readable name for the category
func (c ExpenseCategory) DisplayName() string {
	switch c {
	case ExpenseCategoryRent:
		return "房租"
	case ExpenseCategoryUtilities:
		return "水电费"
	case ExpenseCategorySalary:
		return "工资"
	case ExpenseCategoryOffice:
		return "办公费"
	case ExpenseCategoryTravel:
		return "差旅费"
	case ExpenseCategoryMarketing:
		return "市场营销"
	case ExpenseCategoryEquipment:
		return "设备费"
	case ExpenseCategoryMaintenance:
		return "维修费"
	case ExpenseCategoryInsurance:
		return "保险费"
	case ExpenseCategoryTax:
		return "税费"
	case ExpenseCategoryOther:
		return "其他费用"
	default:
		return string(c)
	}
}

// ExpenseStatus represents the status of an expense record
type ExpenseStatus string

const (
	ExpenseStatusDraft     ExpenseStatus = "DRAFT"     // Draft, not yet submitted
	ExpenseStatusPending   ExpenseStatus = "PENDING"   // Submitted, pending approval
	ExpenseStatusApproved  ExpenseStatus = "APPROVED"  // Approved
	ExpenseStatusRejected  ExpenseStatus = "REJECTED"  // Rejected
	ExpenseStatusCancelled ExpenseStatus = "CANCELLED" // Cancelled
)

// IsValid checks if the status is a valid ExpenseStatus
func (s ExpenseStatus) IsValid() bool {
	switch s {
	case ExpenseStatusDraft, ExpenseStatusPending, ExpenseStatusApproved,
		ExpenseStatusRejected, ExpenseStatusCancelled:
		return true
	}
	return false
}

// String returns the string representation of ExpenseStatus
func (s ExpenseStatus) String() string {
	return string(s)
}

// IsTerminal returns true if the expense is in a terminal state
func (s ExpenseStatus) IsTerminal() bool {
	return s == ExpenseStatusApproved || s == ExpenseStatusRejected || s == ExpenseStatusCancelled
}

// CanSubmit returns true if the expense can be submitted for approval
func (s ExpenseStatus) CanSubmit() bool {
	return s == ExpenseStatusDraft
}

// CanApprove returns true if the expense can be approved/rejected
func (s ExpenseStatus) CanApprove() bool {
	return s == ExpenseStatusPending
}

// CanCancel returns true if the expense can be cancelled
func (s ExpenseStatus) CanCancel() bool {
	return s == ExpenseStatusDraft || s == ExpenseStatusPending
}

// PaymentStatus represents whether the expense has been paid
type PaymentStatus string

const (
	PaymentStatusUnpaid PaymentStatus = "UNPAID"
	PaymentStatusPaid   PaymentStatus = "PAID"
)

// ExpenseRecord represents an expense record aggregate root
// It tracks non-trade expenses like rent, utilities, salary, etc.
type ExpenseRecord struct {
	shared.TenantAggregateRoot
	ExpenseNumber   string          `json:"expense_number"`
	Category        ExpenseCategory `json:"category"`
	Amount          decimal.Decimal `json:"amount"`
	Description     string          `json:"description"`
	IncurredAt      time.Time       `json:"incurred_at"` // When the expense was incurred
	Status          ExpenseStatus   `json:"status"`
	PaymentStatus   PaymentStatus   `json:"payment_status"`
	PaymentMethod   *PaymentMethod  `json:"payment_method"`
	PaidAt          *time.Time      `json:"paid_at"` // When the expense was paid
	Remark          string          `json:"remark"`
	AttachmentURLs  string          `json:"attachment_urls"` // JSON array of attachment URLs
	SubmittedAt     *time.Time      `json:"submitted_at"`    // When submitted for approval
	SubmittedBy     *uuid.UUID      `json:"submitted_by"`    // User who submitted
	ApprovedAt      *time.Time      `json:"approved_at"`     // When approved
	ApprovedBy      *uuid.UUID      `json:"approved_by"`     // User who approved
	ApprovalRemark  string          `json:"approval_remark"`
	RejectedAt      *time.Time      `json:"rejected_at"` // When rejected
	RejectedBy      *uuid.UUID      `json:"rejected_by"` // User who rejected
	RejectionReason string          `json:"rejection_reason"`
	CancelledAt     *time.Time      `json:"cancelled_at"` // When cancelled
	CancelledBy     *uuid.UUID      `json:"cancelled_by"` // User who cancelled
	CancelReason    string          `json:"cancel_reason"`
}

// NewExpenseRecord creates a new expense record
func NewExpenseRecord(
	tenantID uuid.UUID,
	expenseNumber string,
	category ExpenseCategory,
	amount valueobject.Money,
	description string,
	incurredAt time.Time,
) (*ExpenseRecord, error) {
	// Validate inputs
	if expenseNumber == "" {
		return nil, shared.NewDomainError("INVALID_EXPENSE_NUMBER", "Expense number cannot be empty")
	}
	if len(expenseNumber) > 50 {
		return nil, shared.NewDomainError("INVALID_EXPENSE_NUMBER", "Expense number cannot exceed 50 characters")
	}
	if !category.IsValid() {
		return nil, shared.NewDomainError("INVALID_CATEGORY", "Expense category is not valid")
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

	expense := &ExpenseRecord{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		ExpenseNumber:       expenseNumber,
		Category:            category,
		Amount:              amount.Amount(),
		Description:         description,
		IncurredAt:          incurredAt,
		Status:              ExpenseStatusDraft,
		PaymentStatus:       PaymentStatusUnpaid,
	}

	expense.AddDomainEvent(NewExpenseRecordCreatedEvent(expense))

	return expense, nil
}

// Submit submits the expense for approval
func (e *ExpenseRecord) Submit(submittedBy uuid.UUID) error {
	if !e.Status.CanSubmit() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot submit expense in %s status", e.Status))
	}
	if submittedBy == uuid.Nil {
		return shared.NewDomainError("INVALID_USER", "Submitter user ID cannot be empty")
	}

	now := time.Now()
	e.Status = ExpenseStatusPending
	e.SubmittedAt = &now
	e.SubmittedBy = &submittedBy
	e.UpdatedAt = now

	e.AddDomainEvent(NewExpenseRecordSubmittedEvent(e))

	return nil
}

// Approve approves the expense
func (e *ExpenseRecord) Approve(approvedBy uuid.UUID, remark string) error {
	if !e.Status.CanApprove() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot approve expense in %s status", e.Status))
	}
	if approvedBy == uuid.Nil {
		return shared.NewDomainError("INVALID_USER", "Approver user ID cannot be empty")
	}

	now := time.Now()
	e.Status = ExpenseStatusApproved
	e.ApprovedAt = &now
	e.ApprovedBy = &approvedBy
	e.ApprovalRemark = remark
	e.UpdatedAt = now

	e.AddDomainEvent(NewExpenseRecordApprovedEvent(e))

	return nil
}

// Reject rejects the expense
func (e *ExpenseRecord) Reject(rejectedBy uuid.UUID, reason string) error {
	if !e.Status.CanApprove() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot reject expense in %s status", e.Status))
	}
	if rejectedBy == uuid.Nil {
		return shared.NewDomainError("INVALID_USER", "Rejector user ID cannot be empty")
	}
	if reason == "" {
		return shared.NewDomainError("INVALID_REASON", "Rejection reason is required")
	}

	now := time.Now()
	e.Status = ExpenseStatusRejected
	e.RejectedAt = &now
	e.RejectedBy = &rejectedBy
	e.RejectionReason = reason
	e.UpdatedAt = now

	e.AddDomainEvent(NewExpenseRecordRejectedEvent(e))

	return nil
}

// Cancel cancels the expense
func (e *ExpenseRecord) Cancel(cancelledBy uuid.UUID, reason string) error {
	if !e.Status.CanCancel() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot cancel expense in %s status", e.Status))
	}
	if cancelledBy == uuid.Nil {
		return shared.NewDomainError("INVALID_USER", "Canceller user ID cannot be empty")
	}
	if reason == "" {
		return shared.NewDomainError("INVALID_REASON", "Cancel reason is required")
	}

	now := time.Now()
	e.Status = ExpenseStatusCancelled
	e.CancelledAt = &now
	e.CancelledBy = &cancelledBy
	e.CancelReason = reason
	e.UpdatedAt = now

	e.AddDomainEvent(NewExpenseRecordCancelledEvent(e))

	return nil
}

// MarkAsPaid marks the expense as paid
func (e *ExpenseRecord) MarkAsPaid(paymentMethod PaymentMethod) error {
	if e.Status != ExpenseStatusApproved {
		return shared.NewDomainError("INVALID_STATE", "Only approved expenses can be marked as paid")
	}
	if e.PaymentStatus == PaymentStatusPaid {
		return shared.NewDomainError("ALREADY_PAID", "Expense is already paid")
	}
	if !paymentMethod.IsValid() {
		return shared.NewDomainError("INVALID_PAYMENT_METHOD", "Payment method is not valid")
	}

	now := time.Now()
	e.PaymentStatus = PaymentStatusPaid
	e.PaymentMethod = &paymentMethod
	e.PaidAt = &now
	e.UpdatedAt = now

	e.AddDomainEvent(NewExpenseRecordPaidEvent(e))

	return nil
}

// Update updates the expense details (only allowed in draft status)
func (e *ExpenseRecord) Update(
	category ExpenseCategory,
	amount valueobject.Money,
	description string,
	incurredAt time.Time,
) error {
	if e.Status != ExpenseStatusDraft {
		return shared.NewDomainError("INVALID_STATE", "Can only update expense in draft status")
	}
	if !category.IsValid() {
		return shared.NewDomainError("INVALID_CATEGORY", "Expense category is not valid")
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

	e.Category = category
	e.Amount = amount.Amount()
	e.Description = description
	e.IncurredAt = incurredAt
	e.UpdatedAt = time.Now()

	return nil
}

// SetRemark sets the remark
func (e *ExpenseRecord) SetRemark(remark string) {
	e.Remark = remark
	e.UpdatedAt = time.Now()
}

// SetAttachmentURLs sets the attachment URLs (JSON array)
func (e *ExpenseRecord) SetAttachmentURLs(urls string) {
	e.AttachmentURLs = urls
	e.UpdatedAt = time.Now()
}

// Helper methods

// GetAmountMoney returns amount as Money
func (e *ExpenseRecord) GetAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(e.Amount)
}

// IsDraft returns true if expense is in draft status
func (e *ExpenseRecord) IsDraft() bool {
	return e.Status == ExpenseStatusDraft
}

// IsPending returns true if expense is pending approval
func (e *ExpenseRecord) IsPending() bool {
	return e.Status == ExpenseStatusPending
}

// IsApproved returns true if expense is approved
func (e *ExpenseRecord) IsApproved() bool {
	return e.Status == ExpenseStatusApproved
}

// IsRejected returns true if expense is rejected
func (e *ExpenseRecord) IsRejected() bool {
	return e.Status == ExpenseStatusRejected
}

// IsCancelled returns true if expense is cancelled
func (e *ExpenseRecord) IsCancelled() bool {
	return e.Status == ExpenseStatusCancelled
}

// IsPaid returns true if expense is paid
func (e *ExpenseRecord) IsPaid() bool {
	return e.PaymentStatus == PaymentStatusPaid
}
