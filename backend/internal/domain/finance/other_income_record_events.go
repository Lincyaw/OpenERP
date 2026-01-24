package finance

import (
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// OtherIncomeRecordCreatedEvent is raised when a new other income record is created
type OtherIncomeRecordCreatedEvent struct {
	shared.BaseDomainEvent
	IncomeID     uuid.UUID       `json:"income_id"`
	IncomeNumber string          `json:"income_number"`
	Category     IncomeCategory  `json:"category"`
	Amount       decimal.Decimal `json:"amount"`
	Description  string          `json:"description"`
	ReceivedAt   time.Time       `json:"received_at"`
}

// EventType returns the event type name
func (e *OtherIncomeRecordCreatedEvent) EventType() string {
	return "OtherIncomeRecordCreated"
}

// NewOtherIncomeRecordCreatedEvent creates a new OtherIncomeRecordCreatedEvent
func NewOtherIncomeRecordCreatedEvent(income *OtherIncomeRecord) *OtherIncomeRecordCreatedEvent {
	return &OtherIncomeRecordCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent("OtherIncomeRecordCreated", "OtherIncomeRecord", income.ID, income.TenantID),
		IncomeID:        income.ID,
		IncomeNumber:    income.IncomeNumber,
		Category:        income.Category,
		Amount:          income.Amount,
		Description:     income.Description,
		ReceivedAt:      income.ReceivedAt,
	}
}

// OtherIncomeRecordConfirmedEvent is raised when an income record is confirmed
type OtherIncomeRecordConfirmedEvent struct {
	shared.BaseDomainEvent
	IncomeID     uuid.UUID       `json:"income_id"`
	IncomeNumber string          `json:"income_number"`
	Category     IncomeCategory  `json:"category"`
	Amount       decimal.Decimal `json:"amount"`
	ConfirmedBy  uuid.UUID       `json:"confirmed_by"`
	ConfirmedAt  time.Time       `json:"confirmed_at"`
}

// EventType returns the event type name
func (e *OtherIncomeRecordConfirmedEvent) EventType() string {
	return "OtherIncomeRecordConfirmed"
}

// NewOtherIncomeRecordConfirmedEvent creates a new OtherIncomeRecordConfirmedEvent
func NewOtherIncomeRecordConfirmedEvent(income *OtherIncomeRecord) *OtherIncomeRecordConfirmedEvent {
	confirmedAt := time.Now()
	if income.ConfirmedAt != nil {
		confirmedAt = *income.ConfirmedAt
	}
	var confirmedBy uuid.UUID
	if income.ConfirmedBy != nil {
		confirmedBy = *income.ConfirmedBy
	}
	return &OtherIncomeRecordConfirmedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent("OtherIncomeRecordConfirmed", "OtherIncomeRecord", income.ID, income.TenantID),
		IncomeID:        income.ID,
		IncomeNumber:    income.IncomeNumber,
		Category:        income.Category,
		Amount:          income.Amount,
		ConfirmedBy:     confirmedBy,
		ConfirmedAt:     confirmedAt,
	}
}

// OtherIncomeRecordCancelledEvent is raised when an income record is cancelled
type OtherIncomeRecordCancelledEvent struct {
	shared.BaseDomainEvent
	IncomeID     uuid.UUID       `json:"income_id"`
	IncomeNumber string          `json:"income_number"`
	Category     IncomeCategory  `json:"category"`
	Amount       decimal.Decimal `json:"amount"`
	CancelledBy  uuid.UUID       `json:"cancelled_by"`
	CancelledAt  time.Time       `json:"cancelled_at"`
	CancelReason string          `json:"cancel_reason"`
}

// EventType returns the event type name
func (e *OtherIncomeRecordCancelledEvent) EventType() string {
	return "OtherIncomeRecordCancelled"
}

// NewOtherIncomeRecordCancelledEvent creates a new OtherIncomeRecordCancelledEvent
func NewOtherIncomeRecordCancelledEvent(income *OtherIncomeRecord) *OtherIncomeRecordCancelledEvent {
	cancelledAt := time.Now()
	if income.CancelledAt != nil {
		cancelledAt = *income.CancelledAt
	}
	var cancelledBy uuid.UUID
	if income.CancelledBy != nil {
		cancelledBy = *income.CancelledBy
	}
	return &OtherIncomeRecordCancelledEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent("OtherIncomeRecordCancelled", "OtherIncomeRecord", income.ID, income.TenantID),
		IncomeID:        income.ID,
		IncomeNumber:    income.IncomeNumber,
		Category:        income.Category,
		Amount:          income.Amount,
		CancelledBy:     cancelledBy,
		CancelledAt:     cancelledAt,
		CancelReason:    income.CancelReason,
	}
}

// OtherIncomeRecordReceivedEvent is raised when income is actually received
type OtherIncomeRecordReceivedEvent struct {
	shared.BaseDomainEvent
	IncomeID       uuid.UUID       `json:"income_id"`
	IncomeNumber   string          `json:"income_number"`
	Category       IncomeCategory  `json:"category"`
	Amount         decimal.Decimal `json:"amount"`
	PaymentMethod  PaymentMethod   `json:"payment_method"`
	ActualReceived time.Time       `json:"actual_received"`
}

// EventType returns the event type name
func (e *OtherIncomeRecordReceivedEvent) EventType() string {
	return "OtherIncomeRecordReceived"
}

// NewOtherIncomeRecordReceivedEvent creates a new OtherIncomeRecordReceivedEvent
func NewOtherIncomeRecordReceivedEvent(income *OtherIncomeRecord) *OtherIncomeRecordReceivedEvent {
	actualReceived := time.Now()
	if income.ActualReceived != nil {
		actualReceived = *income.ActualReceived
	}
	var paymentMethod PaymentMethod
	if income.PaymentMethod != nil {
		paymentMethod = *income.PaymentMethod
	}
	return &OtherIncomeRecordReceivedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent("OtherIncomeRecordReceived", "OtherIncomeRecord", income.ID, income.TenantID),
		IncomeID:        income.ID,
		IncomeNumber:    income.IncomeNumber,
		Category:        income.Category,
		Amount:          income.Amount,
		PaymentMethod:   paymentMethod,
		ActualReceived:  actualReceived,
	}
}
