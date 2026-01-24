package finance

import (
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ExpenseRecordCreatedEvent is raised when a new expense record is created
type ExpenseRecordCreatedEvent struct {
	shared.BaseDomainEvent
	ExpenseID     uuid.UUID       `json:"expense_id"`
	ExpenseNumber string          `json:"expense_number"`
	Category      ExpenseCategory `json:"category"`
	Amount        decimal.Decimal `json:"amount"`
	Description   string          `json:"description"`
	IncurredAt    time.Time       `json:"incurred_at"`
}

// EventType returns the event type name
func (e *ExpenseRecordCreatedEvent) EventType() string {
	return "ExpenseRecordCreated"
}

// NewExpenseRecordCreatedEvent creates a new ExpenseRecordCreatedEvent
func NewExpenseRecordCreatedEvent(expense *ExpenseRecord) *ExpenseRecordCreatedEvent {
	return &ExpenseRecordCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent("ExpenseRecordCreated", "ExpenseRecord", expense.ID, expense.TenantID),
		ExpenseID:       expense.ID,
		ExpenseNumber:   expense.ExpenseNumber,
		Category:        expense.Category,
		Amount:          expense.Amount,
		Description:     expense.Description,
		IncurredAt:      expense.IncurredAt,
	}
}

// ExpenseRecordSubmittedEvent is raised when an expense is submitted for approval
type ExpenseRecordSubmittedEvent struct {
	shared.BaseDomainEvent
	ExpenseID     uuid.UUID       `json:"expense_id"`
	ExpenseNumber string          `json:"expense_number"`
	Category      ExpenseCategory `json:"category"`
	Amount        decimal.Decimal `json:"amount"`
	SubmittedBy   uuid.UUID       `json:"submitted_by"`
	SubmittedAt   time.Time       `json:"submitted_at"`
}

// EventType returns the event type name
func (e *ExpenseRecordSubmittedEvent) EventType() string {
	return "ExpenseRecordSubmitted"
}

// NewExpenseRecordSubmittedEvent creates a new ExpenseRecordSubmittedEvent
func NewExpenseRecordSubmittedEvent(expense *ExpenseRecord) *ExpenseRecordSubmittedEvent {
	submittedAt := time.Now()
	if expense.SubmittedAt != nil {
		submittedAt = *expense.SubmittedAt
	}
	var submittedBy uuid.UUID
	if expense.SubmittedBy != nil {
		submittedBy = *expense.SubmittedBy
	}
	return &ExpenseRecordSubmittedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent("ExpenseRecordSubmitted", "ExpenseRecord", expense.ID, expense.TenantID),
		ExpenseID:       expense.ID,
		ExpenseNumber:   expense.ExpenseNumber,
		Category:        expense.Category,
		Amount:          expense.Amount,
		SubmittedBy:     submittedBy,
		SubmittedAt:     submittedAt,
	}
}

// ExpenseRecordApprovedEvent is raised when an expense is approved
type ExpenseRecordApprovedEvent struct {
	shared.BaseDomainEvent
	ExpenseID      uuid.UUID       `json:"expense_id"`
	ExpenseNumber  string          `json:"expense_number"`
	Category       ExpenseCategory `json:"category"`
	Amount         decimal.Decimal `json:"amount"`
	ApprovedBy     uuid.UUID       `json:"approved_by"`
	ApprovedAt     time.Time       `json:"approved_at"`
	ApprovalRemark string          `json:"approval_remark,omitempty"`
}

// EventType returns the event type name
func (e *ExpenseRecordApprovedEvent) EventType() string {
	return "ExpenseRecordApproved"
}

// NewExpenseRecordApprovedEvent creates a new ExpenseRecordApprovedEvent
func NewExpenseRecordApprovedEvent(expense *ExpenseRecord) *ExpenseRecordApprovedEvent {
	approvedAt := time.Now()
	if expense.ApprovedAt != nil {
		approvedAt = *expense.ApprovedAt
	}
	var approvedBy uuid.UUID
	if expense.ApprovedBy != nil {
		approvedBy = *expense.ApprovedBy
	}
	return &ExpenseRecordApprovedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent("ExpenseRecordApproved", "ExpenseRecord", expense.ID, expense.TenantID),
		ExpenseID:       expense.ID,
		ExpenseNumber:   expense.ExpenseNumber,
		Category:        expense.Category,
		Amount:          expense.Amount,
		ApprovedBy:      approvedBy,
		ApprovedAt:      approvedAt,
		ApprovalRemark:  expense.ApprovalRemark,
	}
}

// ExpenseRecordRejectedEvent is raised when an expense is rejected
type ExpenseRecordRejectedEvent struct {
	shared.BaseDomainEvent
	ExpenseID       uuid.UUID       `json:"expense_id"`
	ExpenseNumber   string          `json:"expense_number"`
	Category        ExpenseCategory `json:"category"`
	Amount          decimal.Decimal `json:"amount"`
	RejectedBy      uuid.UUID       `json:"rejected_by"`
	RejectedAt      time.Time       `json:"rejected_at"`
	RejectionReason string          `json:"rejection_reason"`
}

// EventType returns the event type name
func (e *ExpenseRecordRejectedEvent) EventType() string {
	return "ExpenseRecordRejected"
}

// NewExpenseRecordRejectedEvent creates a new ExpenseRecordRejectedEvent
func NewExpenseRecordRejectedEvent(expense *ExpenseRecord) *ExpenseRecordRejectedEvent {
	rejectedAt := time.Now()
	if expense.RejectedAt != nil {
		rejectedAt = *expense.RejectedAt
	}
	var rejectedBy uuid.UUID
	if expense.RejectedBy != nil {
		rejectedBy = *expense.RejectedBy
	}
	return &ExpenseRecordRejectedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent("ExpenseRecordRejected", "ExpenseRecord", expense.ID, expense.TenantID),
		ExpenseID:       expense.ID,
		ExpenseNumber:   expense.ExpenseNumber,
		Category:        expense.Category,
		Amount:          expense.Amount,
		RejectedBy:      rejectedBy,
		RejectedAt:      rejectedAt,
		RejectionReason: expense.RejectionReason,
	}
}

// ExpenseRecordCancelledEvent is raised when an expense is cancelled
type ExpenseRecordCancelledEvent struct {
	shared.BaseDomainEvent
	ExpenseID     uuid.UUID       `json:"expense_id"`
	ExpenseNumber string          `json:"expense_number"`
	Category      ExpenseCategory `json:"category"`
	Amount        decimal.Decimal `json:"amount"`
	CancelledBy   uuid.UUID       `json:"cancelled_by"`
	CancelledAt   time.Time       `json:"cancelled_at"`
	CancelReason  string          `json:"cancel_reason"`
}

// EventType returns the event type name
func (e *ExpenseRecordCancelledEvent) EventType() string {
	return "ExpenseRecordCancelled"
}

// NewExpenseRecordCancelledEvent creates a new ExpenseRecordCancelledEvent
func NewExpenseRecordCancelledEvent(expense *ExpenseRecord) *ExpenseRecordCancelledEvent {
	cancelledAt := time.Now()
	if expense.CancelledAt != nil {
		cancelledAt = *expense.CancelledAt
	}
	var cancelledBy uuid.UUID
	if expense.CancelledBy != nil {
		cancelledBy = *expense.CancelledBy
	}
	return &ExpenseRecordCancelledEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent("ExpenseRecordCancelled", "ExpenseRecord", expense.ID, expense.TenantID),
		ExpenseID:       expense.ID,
		ExpenseNumber:   expense.ExpenseNumber,
		Category:        expense.Category,
		Amount:          expense.Amount,
		CancelledBy:     cancelledBy,
		CancelledAt:     cancelledAt,
		CancelReason:    expense.CancelReason,
	}
}

// ExpenseRecordPaidEvent is raised when an expense is paid
type ExpenseRecordPaidEvent struct {
	shared.BaseDomainEvent
	ExpenseID     uuid.UUID       `json:"expense_id"`
	ExpenseNumber string          `json:"expense_number"`
	Category      ExpenseCategory `json:"category"`
	Amount        decimal.Decimal `json:"amount"`
	PaymentMethod PaymentMethod   `json:"payment_method"`
	PaidAt        time.Time       `json:"paid_at"`
}

// EventType returns the event type name
func (e *ExpenseRecordPaidEvent) EventType() string {
	return "ExpenseRecordPaid"
}

// NewExpenseRecordPaidEvent creates a new ExpenseRecordPaidEvent
func NewExpenseRecordPaidEvent(expense *ExpenseRecord) *ExpenseRecordPaidEvent {
	paidAt := time.Now()
	if expense.PaidAt != nil {
		paidAt = *expense.PaidAt
	}
	var paymentMethod PaymentMethod
	if expense.PaymentMethod != nil {
		paymentMethod = *expense.PaymentMethod
	}
	return &ExpenseRecordPaidEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent("ExpenseRecordPaid", "ExpenseRecord", expense.ID, expense.TenantID),
		ExpenseID:       expense.ID,
		ExpenseNumber:   expense.ExpenseNumber,
		Category:        expense.Category,
		Amount:          expense.Amount,
		PaymentMethod:   paymentMethod,
		PaidAt:          paidAt,
	}
}
