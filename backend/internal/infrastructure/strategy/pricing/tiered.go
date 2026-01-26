package pricing

import (
	"context"
	"sort"

	"github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/shopspring/decimal"
)

// PriceTier represents a single pricing tier with minimum quantity and unit price
type PriceTier struct {
	MinQuantity decimal.Decimal `json:"min_quantity"`
	UnitPrice   decimal.Decimal `json:"unit_price"`
}

// TieredPricingStrategy implements quantity-based tiered pricing
// Prices decrease as quantity increases (volume discounts)
type TieredPricingStrategy struct {
	strategy.BaseStrategy
	tiers []PriceTier
}

// NewTieredPricingStrategy creates a new tiered pricing strategy with the given tiers
// Tiers should be provided in any order - they will be sorted by min quantity ascending
func NewTieredPricingStrategy(tiers []PriceTier) *TieredPricingStrategy {
	// Sort tiers by min quantity ascending for efficient lookup
	sortedTiers := make([]PriceTier, len(tiers))
	copy(sortedTiers, tiers)
	sort.Slice(sortedTiers, func(i, j int) bool {
		return sortedTiers[i].MinQuantity.LessThan(sortedTiers[j].MinQuantity)
	})

	return &TieredPricingStrategy{
		BaseStrategy: strategy.NewBaseStrategy(
			"tiered",
			strategy.StrategyTypePricing,
			"Tiered pricing based on quantity thresholds",
		),
		tiers: sortedTiers,
	}
}

// GetTiers returns a copy of the pricing tiers
func (s *TieredPricingStrategy) GetTiers() []PriceTier {
	result := make([]PriceTier, len(s.tiers))
	copy(result, s.tiers)
	return result
}

// CalculatePrice calculates the price using quantity-based tiers
// It finds the highest tier whose MinQuantity is <= the order quantity
func (s *TieredPricingStrategy) CalculatePrice(
	ctx context.Context,
	pricingCtx strategy.PricingContext,
) (strategy.PricingResult, error) {
	// Default to base price if no tiers match
	unitPrice := pricingCtx.BasePrice
	appliedRules := []string{}

	// Find the applicable tier (highest tier where quantity >= min_quantity)
	// Since tiers are sorted ascending, we iterate from the end
	for i := len(s.tiers) - 1; i >= 0; i-- {
		if pricingCtx.Quantity.GreaterThanOrEqual(s.tiers[i].MinQuantity) {
			unitPrice = s.tiers[i].UnitPrice
			appliedRules = append(appliedRules, "tiered_pricing")
			break
		}
	}

	totalPrice := unitPrice.Mul(pricingCtx.Quantity)

	// Calculate discount from base price
	baseTotalPrice := pricingCtx.BasePrice.Mul(pricingCtx.Quantity)
	discountAmount := baseTotalPrice.Sub(totalPrice)
	discountPercent := decimal.Zero
	if baseTotalPrice.GreaterThan(decimal.Zero) {
		discountPercent = discountAmount.Div(baseTotalPrice).Mul(decimal.NewFromInt(100)).Round(2)
	}

	return strategy.PricingResult{
		UnitPrice:       unitPrice,
		TotalPrice:      totalPrice,
		DiscountAmount:  discountAmount,
		DiscountPercent: discountPercent,
		Currency:        pricingCtx.Currency,
		AppliedRules:    appliedRules,
	}, nil
}

// SupportsPromotion returns false as tiered pricing doesn't support promotions
func (s *TieredPricingStrategy) SupportsPromotion() bool {
	return false
}

// SupportsTieredPricing returns true as this is a tiered pricing strategy
func (s *TieredPricingStrategy) SupportsTieredPricing() bool {
	return true
}

// DefaultTieredPricingStrategy creates a tiered strategy with NO default tiers (pass-through).
//
// IMPORTANT: This is a pass-through strategy that returns the base price unchanged.
// It is registered as an available strategy type, but actual tier configuration
// must be provided per-tenant or per-product based on business requirements.
//
// To use tiered pricing with actual discounts, use NewTieredPricingStrategy() directly:
//
//	tiers := []PriceTier{
//		{MinQuantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100)},
//		{MinQuantity: decimal.NewFromInt(10), UnitPrice: decimal.NewFromInt(95)},   // 5% off
//		{MinQuantity: decimal.NewFromInt(50), UnitPrice: decimal.NewFromInt(90)},   // 10% off
//		{MinQuantity: decimal.NewFromInt(100), UnitPrice: decimal.NewFromInt(85)},  // 15% off
//	}
//	strategy := NewTieredPricingStrategy(tiers)
//
// Why no default tiers? Tiered pricing requires absolute unit prices which vary by product.
// A "default" tier with fixed prices (e.g., 100 CNY) would be meaningless for products
// priced at different levels (e.g., 10 CNY or 10,000 CNY items).
func DefaultTieredPricingStrategy() *TieredPricingStrategy {
	return &TieredPricingStrategy{
		BaseStrategy: strategy.NewBaseStrategy(
			"tiered",
			strategy.StrategyTypePricing,
			"Tiered pricing placeholder - configure tiers per tenant/product",
		),
		tiers: []PriceTier{},
	}
}
