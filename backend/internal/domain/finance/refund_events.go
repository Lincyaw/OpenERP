package finance

import (
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// RefundRecordCreatedEvent is raised when a new refund record is created
type RefundRecordCreatedEvent struct {
	shared.BaseDomainEvent
	RefundNumber        string             `json:"refund_number"`
	OriginalOrderID     uuid.UUID          `json:"original_order_id"`
	OriginalOrderNumber string             `json:"original_order_number"`
	SourceType          RefundSourceType   `json:"source_type"`
	SourceID            uuid.UUID          `json:"source_id"`
	CustomerID          uuid.UUID          `json:"customer_id"`
	CustomerName        string             `json:"customer_name"`
	RefundAmount        decimal.Decimal    `json:"refund_amount"`
	GatewayType         PaymentGatewayType `json:"gateway_type"`
	Reason              string             `json:"reason"`
}

// EventType returns the event type name
func (e *RefundRecordCreatedEvent) EventType() string {
	return "RefundRecordCreated"
}

// NewRefundRecordCreatedEvent creates a new RefundRecordCreatedEvent
func NewRefundRecordCreatedEvent(r *RefundRecord) *RefundRecordCreatedEvent {
	return &RefundRecordCreatedEvent{
		BaseDomainEvent:     shared.NewBaseDomainEvent("RefundRecordCreated", "RefundRecord", r.ID, r.TenantID),
		RefundNumber:        r.RefundNumber,
		OriginalOrderID:     r.OriginalOrderID,
		OriginalOrderNumber: r.OriginalOrderNumber,
		SourceType:          r.SourceType,
		SourceID:            r.SourceID,
		CustomerID:          r.CustomerID,
		CustomerName:        r.CustomerName,
		RefundAmount:        r.RefundAmount,
		GatewayType:         r.GatewayType,
		Reason:              r.Reason,
	}
}

// RefundRecordCompletedEvent is raised when a refund is successfully completed
type RefundRecordCompletedEvent struct {
	shared.BaseDomainEvent
	RefundNumber         string             `json:"refund_number"`
	OriginalOrderID      uuid.UUID          `json:"original_order_id"`
	CustomerID           uuid.UUID          `json:"customer_id"`
	RefundAmount         decimal.Decimal    `json:"refund_amount"`
	ActualRefundAmount   decimal.Decimal    `json:"actual_refund_amount"`
	GatewayType          PaymentGatewayType `json:"gateway_type"`
	GatewayRefundID      string             `json:"gateway_refund_id"`
	GatewayTransactionID string             `json:"gateway_transaction_id"`
}

// EventType returns the event type name
func (e *RefundRecordCompletedEvent) EventType() string {
	return "RefundRecordCompleted"
}

// NewRefundRecordCompletedEvent creates a new RefundRecordCompletedEvent
func NewRefundRecordCompletedEvent(r *RefundRecord) *RefundRecordCompletedEvent {
	return &RefundRecordCompletedEvent{
		BaseDomainEvent:      shared.NewBaseDomainEvent("RefundRecordCompleted", "RefundRecord", r.ID, r.TenantID),
		RefundNumber:         r.RefundNumber,
		OriginalOrderID:      r.OriginalOrderID,
		CustomerID:           r.CustomerID,
		RefundAmount:         r.RefundAmount,
		ActualRefundAmount:   r.ActualRefundAmount,
		GatewayType:          r.GatewayType,
		GatewayRefundID:      r.GatewayRefundID,
		GatewayTransactionID: r.GatewayTransactionID,
	}
}

// RefundRecordFailedEvent is raised when a refund fails
type RefundRecordFailedEvent struct {
	shared.BaseDomainEvent
	RefundNumber    string             `json:"refund_number"`
	OriginalOrderID uuid.UUID          `json:"original_order_id"`
	CustomerID      uuid.UUID          `json:"customer_id"`
	RefundAmount    decimal.Decimal    `json:"refund_amount"`
	GatewayType     PaymentGatewayType `json:"gateway_type"`
	FailReason      string             `json:"fail_reason"`
}

// EventType returns the event type name
func (e *RefundRecordFailedEvent) EventType() string {
	return "RefundRecordFailed"
}

// NewRefundRecordFailedEvent creates a new RefundRecordFailedEvent
func NewRefundRecordFailedEvent(r *RefundRecord) *RefundRecordFailedEvent {
	return &RefundRecordFailedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent("RefundRecordFailed", "RefundRecord", r.ID, r.TenantID),
		RefundNumber:    r.RefundNumber,
		OriginalOrderID: r.OriginalOrderID,
		CustomerID:      r.CustomerID,
		RefundAmount:    r.RefundAmount,
		GatewayType:     r.GatewayType,
		FailReason:      r.FailReason,
	}
}
