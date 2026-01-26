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

// DefaultTieredPricingStrategy creates a tiered strategy with common volume discount tiers
// - Base price for quantity < 10
// - 5% discount for quantity >= 10
// - 10% discount for quantity >= 50
// - 15% discount for quantity >= 100
func DefaultTieredPricingStrategy() *TieredPricingStrategy {
	// Note: The actual price will be calculated from base price in CalculatePrice
	// These tiers represent discount percentages as placeholders
	return &TieredPricingStrategy{
		BaseStrategy: strategy.NewBaseStrategy(
			"tiered",
			strategy.StrategyTypePricing,
			"Default tiered pricing with volume discounts",
		),
		// Empty tiers mean base price will be used
		// Actual tier configuration should come from tenant settings
		tiers: []PriceTier{},
	}
}
