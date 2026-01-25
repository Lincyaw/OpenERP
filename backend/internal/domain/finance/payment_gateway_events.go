package finance

import (
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// GatewayPaymentCompletedEvent is raised when a payment is completed via external gateway
type GatewayPaymentCompletedEvent struct {
	shared.BaseDomainEvent
	GatewayType          PaymentGatewayType   `json:"gateway_type"`
	GatewayOrderID       string               `json:"gateway_order_id"`
	GatewayTransactionID string               `json:"gateway_transaction_id"`
	OrderNumber          string               `json:"order_number"`
	Status               GatewayPaymentStatus `json:"status"`
	Amount               decimal.Decimal      `json:"amount"`
	PaidAmount           decimal.Decimal      `json:"paid_amount"`
	Currency             string               `json:"currency"`
	PayerAccount         string               `json:"payer_account"`
	PaidAt               *time.Time           `json:"paid_at,omitempty"`
}

// EventType returns the event type name
func (e *GatewayPaymentCompletedEvent) EventType() string {
	return "GatewayPaymentCompleted"
}

// NewGatewayPaymentCompletedEvent creates a new GatewayPaymentCompletedEvent from a payment callback
func NewGatewayPaymentCompletedEvent(tenantID uuid.UUID, callback *PaymentCallback) *GatewayPaymentCompletedEvent {
	return &GatewayPaymentCompletedEvent{
		BaseDomainEvent:      shared.NewBaseDomainEvent("GatewayPaymentCompleted", "PaymentGateway", uuid.New(), tenantID),
		GatewayType:          callback.GatewayType,
		GatewayOrderID:       callback.GatewayOrderID,
		GatewayTransactionID: callback.GatewayTransactionID,
		OrderNumber:          callback.OrderNumber,
		Status:               callback.Status,
		Amount:               callback.Amount,
		PaidAmount:           callback.PaidAmount,
		Currency:             callback.Currency,
		PayerAccount:         callback.PayerAccount,
		PaidAt:               callback.PaidAt,
	}
}

// GatewayRefundCompletedEvent is raised when a refund is completed via external gateway
type GatewayRefundCompletedEvent struct {
	shared.BaseDomainEvent
	GatewayType          PaymentGatewayType `json:"gateway_type"`
	GatewayRefundID      string             `json:"gateway_refund_id"`
	GatewayOrderID       string             `json:"gateway_order_id"`
	GatewayTransactionID string             `json:"gateway_transaction_id"`
	RefundNumber         string             `json:"refund_number"`
	Status               RefundStatus       `json:"status"`
	RefundAmount         decimal.Decimal    `json:"refund_amount"`
	RefundedAt           *time.Time         `json:"refunded_at,omitempty"`
}

// EventType returns the event type name
func (e *GatewayRefundCompletedEvent) EventType() string {
	return "GatewayRefundCompleted"
}

// NewGatewayRefundCompletedEvent creates a new GatewayRefundCompletedEvent from a refund callback
func NewGatewayRefundCompletedEvent(tenantID uuid.UUID, callback *RefundCallback) *GatewayRefundCompletedEvent {
	return &GatewayRefundCompletedEvent{
		BaseDomainEvent:      shared.NewBaseDomainEvent("GatewayRefundCompleted", "PaymentGateway", uuid.New(), tenantID),
		GatewayType:          callback.GatewayType,
		GatewayRefundID:      callback.GatewayRefundID,
		GatewayOrderID:       callback.GatewayOrderID,
		GatewayTransactionID: callback.GatewayTransactionID,
		RefundNumber:         callback.RefundNumber,
		Status:               callback.Status,
		RefundAmount:         callback.RefundAmount,
		RefundedAt:           callback.RefundedAt,
	}
}
