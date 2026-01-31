package billing

import (
	"fmt"

	"github.com/stripe/stripe-go/v81"
)

// StripeConfig holds configuration for Stripe integration
type StripeConfig struct {
	// SecretKey is the Stripe secret API key (sk_test_xxx or sk_live_xxx)
	SecretKey string `json:"secret_key" mapstructure:"secret_key"`

	// PublishableKey is the Stripe publishable key for frontend (pk_test_xxx or pk_live_xxx)
	PublishableKey string `json:"publishable_key" mapstructure:"publishable_key"`

	// WebhookSecret is the secret for verifying webhook signatures
	WebhookSecret string `json:"webhook_secret" mapstructure:"webhook_secret"`

	// IsTestMode indicates if using Stripe test mode
	IsTestMode bool `json:"is_test_mode" mapstructure:"is_test_mode"`

	// DefaultCurrency is the default currency for subscriptions (e.g., "cny", "usd")
	DefaultCurrency string `json:"default_currency" mapstructure:"default_currency"`

	// PriceIDs maps plan names to Stripe Price IDs
	PriceIDs map[string]string `json:"price_ids" mapstructure:"price_ids"`

	// SuccessURL is the URL to redirect after successful checkout
	SuccessURL string `json:"success_url" mapstructure:"success_url"`

	// CancelURL is the URL to redirect after cancelled checkout
	CancelURL string `json:"cancel_url" mapstructure:"cancel_url"`

	// BillingPortalReturnURL is the return URL from Stripe billing portal
	BillingPortalReturnURL string `json:"billing_portal_return_url" mapstructure:"billing_portal_return_url"`
}

// DefaultStripeConfig returns a default configuration for development/testing
func DefaultStripeConfig() *StripeConfig {
	return &StripeConfig{
		IsTestMode:      true,
		DefaultCurrency: "cny",
		PriceIDs: map[string]string{
			"free":       "",                    // Free plan has no Stripe price
			"basic":      "price_basic_monthly", // Replace with actual Stripe Price ID
			"pro":        "price_pro_monthly",   // Replace with actual Stripe Price ID
			"enterprise": "price_ent_monthly",   // Replace with actual Stripe Price ID
		},
	}
}

// Validate validates the Stripe configuration
func (c *StripeConfig) Validate() error {
	if c.SecretKey == "" {
		return fmt.Errorf("stripe: secret key is required")
	}

	// Validate key format
	if c.IsTestMode {
		if len(c.SecretKey) > 7 && c.SecretKey[:7] != "sk_test" {
			return fmt.Errorf("stripe: test mode enabled but secret key is not a test key")
		}
	} else {
		if len(c.SecretKey) > 7 && c.SecretKey[:7] != "sk_live" {
			return fmt.Errorf("stripe: live mode enabled but secret key is not a live key")
		}
	}

	if c.DefaultCurrency == "" {
		return fmt.Errorf("stripe: default currency is required")
	}

	return nil
}

// GetPriceID returns the Stripe Price ID for a given plan
func (c *StripeConfig) GetPriceID(plan string) (string, error) {
	priceID, exists := c.PriceIDs[plan]
	if !exists {
		return "", fmt.Errorf("stripe: no price ID configured for plan: %s", plan)
	}
	if priceID == "" && plan != "free" {
		return "", fmt.Errorf("stripe: price ID not set for plan: %s", plan)
	}
	return priceID, nil
}

// InitStripeClient initializes the Stripe client with the configured API key
func (c *StripeConfig) InitStripeClient() {
	stripe.Key = c.SecretKey
}
