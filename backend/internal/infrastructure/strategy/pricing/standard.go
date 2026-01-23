package pricing

import (
	"context"

	"github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/shopspring/decimal"
)

// StandardPricingStrategy implements basic pricing without discounts
type StandardPricingStrategy struct {
	strategy.BaseStrategy
}

// NewStandardPricingStrategy creates a new standard pricing strategy
func NewStandardPricingStrategy() *StandardPricingStrategy {
	return &StandardPricingStrategy{
		BaseStrategy: strategy.NewBaseStrategy(
			"standard",
			strategy.StrategyTypePricing,
			"Standard pricing using base price without discounts",
		),
	}
}

// CalculatePrice calculates the price using base price
func (s *StandardPricingStrategy) CalculatePrice(
	ctx context.Context,
	pricingCtx strategy.PricingContext,
) (strategy.PricingResult, error) {
	totalPrice := pricingCtx.BasePrice.Mul(pricingCtx.Quantity)

	return strategy.PricingResult{
		UnitPrice:       pricingCtx.BasePrice,
		TotalPrice:      totalPrice,
		DiscountAmount:  decimal.Zero,
		DiscountPercent: decimal.Zero,
		Currency:        pricingCtx.Currency,
		AppliedRules:    []string{},
	}, nil
}

// SupportsPromotion returns false as standard pricing doesn't support promotions
func (s *StandardPricingStrategy) SupportsPromotion() bool {
	return false
}

// SupportsTieredPricing returns false as standard pricing doesn't support tiered pricing
func (s *StandardPricingStrategy) SupportsTieredPricing() bool {
	return false
}
