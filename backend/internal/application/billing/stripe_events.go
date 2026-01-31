package billing

import (
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// Event type constants for Stripe webhook events
const (
	EventTypeStripeSubscriptionCreated = "StripeSubscriptionCreated"
	EventTypeStripeSubscriptionUpdated = "StripeSubscriptionUpdated"
	EventTypeStripeSubscriptionDeleted = "StripeSubscriptionDeleted"
	EventTypeStripeInvoicePaid         = "StripeInvoicePaid"
	EventTypeStripePaymentFailed       = "StripePaymentFailed"
)

// Aggregate type constant
const AggregateTypeStripeBilling = "StripeBilling"

// StripeSubscriptionEvent represents a Stripe subscription-related event
type StripeSubscriptionEvent struct {
	shared.BaseDomainEvent
	SubscriptionID string `json:"subscription_id"`
	Action         string `json:"action"` // created, updated, deleted
}

// NewStripeSubscriptionEvent creates a new StripeSubscriptionEvent
func NewStripeSubscriptionEvent(tenantID uuid.UUID, action, subscriptionID string) *StripeSubscriptionEvent {
	var eventType string
	switch action {
	case "subscription_created":
		eventType = EventTypeStripeSubscriptionCreated
	case "subscription_updated":
		eventType = EventTypeStripeSubscriptionUpdated
	case "subscription_deleted":
		eventType = EventTypeStripeSubscriptionDeleted
	default:
		eventType = "StripeSubscription" + action
	}

	return &StripeSubscriptionEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(eventType, AggregateTypeStripeBilling, tenantID, tenantID),
		SubscriptionID:  subscriptionID,
		Action:          action,
	}
}

// StripePaymentEvent represents a Stripe payment-related event
type StripePaymentEvent struct {
	shared.BaseDomainEvent
	InvoiceID string `json:"invoice_id"`
	Action    string `json:"action"` // invoice_paid, payment_failed
}

// NewStripePaymentEvent creates a new StripePaymentEvent
func NewStripePaymentEvent(tenantID uuid.UUID, action, invoiceID string) *StripePaymentEvent {
	var eventType string
	switch action {
	case "invoice_paid":
		eventType = EventTypeStripeInvoicePaid
	case "payment_failed":
		eventType = EventTypeStripePaymentFailed
	default:
		eventType = "StripePayment" + action
	}

	return &StripePaymentEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(eventType, AggregateTypeStripeBilling, tenantID, tenantID),
		InvoiceID:       invoiceID,
		Action:          action,
	}
}
