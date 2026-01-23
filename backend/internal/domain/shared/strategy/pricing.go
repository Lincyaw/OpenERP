package strategy

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// PricingContext provides context for pricing calculation
type PricingContext struct {
	TenantID     string
	ProductID    string
	CustomerID   string
	CustomerType string
	Quantity     decimal.Decimal
	BasePrice    decimal.Decimal
	Currency     string
	OrderDate    time.Time
	PromotionIDs []string
}

// PricingResult contains the result of pricing calculation
type PricingResult struct {
	UnitPrice       decimal.Decimal
	TotalPrice      decimal.Decimal
	DiscountAmount  decimal.Decimal
	DiscountPercent decimal.Decimal
	Currency        string
	AppliedRules    []string
}

// PricingStrategy defines the interface for pricing calculation
type PricingStrategy interface {
	Strategy
	// CalculatePrice calculates the final price for a given pricing context
	CalculatePrice(ctx context.Context, pricingCtx PricingContext) (PricingResult, error)
	// SupportsPromotion returns true if the strategy supports promotional pricing
	SupportsPromotion() bool
	// SupportsTieredPricing returns true if the strategy supports quantity-based tiered pricing
	SupportsTieredPricing() bool
}
