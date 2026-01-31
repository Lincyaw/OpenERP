package billing

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionStatus represents the status of a Stripe subscription
type SubscriptionStatus string

const (
	// SubscriptionStatusActive indicates an active subscription
	SubscriptionStatusActive SubscriptionStatus = "active"

	// SubscriptionStatusPastDue indicates payment is past due
	SubscriptionStatusPastDue SubscriptionStatus = "past_due"

	// SubscriptionStatusCanceled indicates the subscription is canceled
	SubscriptionStatusCanceled SubscriptionStatus = "canceled"

	// SubscriptionStatusIncomplete indicates initial payment failed
	SubscriptionStatusIncomplete SubscriptionStatus = "incomplete"

	// SubscriptionStatusIncompleteExpired indicates incomplete subscription expired
	SubscriptionStatusIncompleteExpired SubscriptionStatus = "incomplete_expired"

	// SubscriptionStatusTrialing indicates subscription is in trial period
	SubscriptionStatusTrialing SubscriptionStatus = "trialing"

	// SubscriptionStatusUnpaid indicates subscription is unpaid
	SubscriptionStatusUnpaid SubscriptionStatus = "unpaid"

	// SubscriptionStatusPaused indicates subscription is paused
	SubscriptionStatusPaused SubscriptionStatus = "paused"
)

// String returns the string representation of SubscriptionStatus
func (s SubscriptionStatus) String() string {
	return string(s)
}

// IsActive returns true if the subscription is in an active state
func (s SubscriptionStatus) IsActive() bool {
	return s == SubscriptionStatusActive || s == SubscriptionStatusTrialing
}

// CreateCustomerInput contains input for creating a Stripe customer
type CreateCustomerInput struct {
	TenantID        uuid.UUID
	Email           string
	Name            string
	Phone           string
	Description     string
	Metadata        map[string]string
	TaxExempt       bool
	PreferredLocale string
}

// CreateCustomerOutput contains the result of creating a Stripe customer
type CreateCustomerOutput struct {
	CustomerID string
	Email      string
	Name       string
	CreatedAt  time.Time
}

// CreateSubscriptionInput contains input for creating a Stripe subscription
type CreateSubscriptionInput struct {
	TenantID         uuid.UUID
	CustomerID       string // Stripe Customer ID
	PlanID           string // Internal plan ID (basic, pro, enterprise)
	PriceID          string // Stripe Price ID (optional, will be looked up from config if empty)
	TrialDays        int    // Number of trial days (0 = no trial)
	Metadata         map[string]string
	PaymentMethod    string // Payment method ID for immediate charge
	CollectionMethod string // "charge_automatically" or "send_invoice"
}

// CreateSubscriptionOutput contains the result of creating a Stripe subscription
type CreateSubscriptionOutput struct {
	SubscriptionID     string
	CustomerID         string
	Status             SubscriptionStatus
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
	TrialStart         *time.Time
	TrialEnd           *time.Time
	CancelAtPeriodEnd  bool
	ClientSecret       string // For incomplete subscriptions requiring payment
	LatestInvoiceID    string
}

// UpdateSubscriptionInput contains input for updating a Stripe subscription
type UpdateSubscriptionInput struct {
	TenantID          uuid.UUID
	SubscriptionID    string
	NewPriceID        string // New Stripe Price ID
	NewPlanID         string // New internal plan ID
	ProrationBehavior string // "create_prorations", "none", "always_invoice"
	CancelAtPeriodEnd bool   // Whether to cancel at period end
	Metadata          map[string]string
}

// UpdateSubscriptionOutput contains the result of updating a Stripe subscription
type UpdateSubscriptionOutput struct {
	SubscriptionID     string
	Status             SubscriptionStatus
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
	CancelAtPeriodEnd  bool
	PreviousPriceID    string
	NewPriceID         string
}

// CancelSubscriptionInput contains input for canceling a Stripe subscription
type CancelSubscriptionInput struct {
	TenantID          uuid.UUID
	SubscriptionID    string
	CancelAtPeriodEnd bool // If true, cancel at end of billing period; if false, cancel immediately
	Reason            string
}

// CancelSubscriptionOutput contains the result of canceling a Stripe subscription
type CancelSubscriptionOutput struct {
	SubscriptionID    string
	Status            SubscriptionStatus
	CanceledAt        *time.Time
	CancelAtPeriodEnd bool
	CurrentPeriodEnd  time.Time
}

// GetSubscriptionStatusInput contains input for getting subscription status
type GetSubscriptionStatusInput struct {
	TenantID       uuid.UUID
	SubscriptionID string
}

// GetSubscriptionStatusOutput contains the subscription status details
type GetSubscriptionStatusOutput struct {
	SubscriptionID       string
	CustomerID           string
	Status               SubscriptionStatus
	PriceID              string
	ProductID            string
	CurrentPeriodStart   time.Time
	CurrentPeriodEnd     time.Time
	TrialStart           *time.Time
	TrialEnd             *time.Time
	CancelAt             *time.Time
	CanceledAt           *time.Time
	CancelAtPeriodEnd    bool
	StartDate            time.Time
	EndedAt              *time.Time
	DaysUntilDue         int
	LatestInvoiceID      string
	DefaultPaymentMethod string
}
