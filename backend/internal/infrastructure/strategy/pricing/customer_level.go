package pricing

import (
	"context"

	"github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/shopspring/decimal"
)

// CustomerLevelDiscount maps customer levels to discount percentages
type CustomerLevelDiscount struct {
	Level           string          `json:"level"`
	DiscountPercent decimal.Decimal `json:"discount_percent"`
}

// CustomerLevelPricingStrategy implements customer-level based pricing
// Different customer levels (normal, silver, gold, platinum, vip) get different discounts
type CustomerLevelPricingStrategy struct {
	strategy.BaseStrategy
	discounts map[string]decimal.Decimal
}

// NewCustomerLevelPricingStrategy creates a new customer level pricing strategy
func NewCustomerLevelPricingStrategy(discounts []CustomerLevelDiscount) *CustomerLevelPricingStrategy {
	discountMap := make(map[string]decimal.Decimal)
	for _, d := range discounts {
		discountMap[d.Level] = d.DiscountPercent
	}

	return &CustomerLevelPricingStrategy{
		BaseStrategy: strategy.NewBaseStrategy(
			"customer_level",
			strategy.StrategyTypePricing,
			"Customer level-based pricing with tier discounts",
		),
		discounts: discountMap,
	}
}

// DefaultCustomerLevelPricingStrategy creates a strategy with default level discounts
// - normal: 0% discount
// - silver: 3% discount
// - gold: 5% discount
// - platinum: 8% discount
// - vip: 10% discount
func DefaultCustomerLevelPricingStrategy() *CustomerLevelPricingStrategy {
	return NewCustomerLevelPricingStrategy([]CustomerLevelDiscount{
		{Level: "normal", DiscountPercent: decimal.Zero},
		{Level: "silver", DiscountPercent: decimal.NewFromInt(3)},
		{Level: "gold", DiscountPercent: decimal.NewFromInt(5)},
		{Level: "platinum", DiscountPercent: decimal.NewFromInt(8)},
		{Level: "vip", DiscountPercent: decimal.NewFromInt(10)},
	})
}

// GetDiscounts returns a copy of the discount configuration
func (s *CustomerLevelPricingStrategy) GetDiscounts() map[string]decimal.Decimal {
	result := make(map[string]decimal.Decimal)
	for k, v := range s.discounts {
		result[k] = v
	}
	return result
}

// GetDiscountForLevel returns the discount percentage for a given customer level
func (s *CustomerLevelPricingStrategy) GetDiscountForLevel(level string) decimal.Decimal {
	if discount, ok := s.discounts[level]; ok {
		return discount
	}
	return decimal.Zero
}

// CalculatePrice calculates the price based on customer level discount
func (s *CustomerLevelPricingStrategy) CalculatePrice(
	ctx context.Context,
	pricingCtx strategy.PricingContext,
) (strategy.PricingResult, error) {
	// Get discount percentage for customer level
	discountPercent := s.GetDiscountForLevel(pricingCtx.CustomerType)

	// Calculate discounted unit price
	// discountedPrice = basePrice * (1 - discountPercent/100)
	discountMultiplier := decimal.NewFromInt(1).Sub(discountPercent.Div(decimal.NewFromInt(100)))
	unitPrice := pricingCtx.BasePrice.Mul(discountMultiplier).Round(4)

	totalPrice := unitPrice.Mul(pricingCtx.Quantity)

	// Calculate actual discount amount
	baseTotalPrice := pricingCtx.BasePrice.Mul(pricingCtx.Quantity)
	discountAmount := baseTotalPrice.Sub(totalPrice)

	appliedRules := []string{}
	if discountPercent.GreaterThan(decimal.Zero) {
		appliedRules = append(appliedRules, "customer_level_discount")
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

// SupportsPromotion returns false as level-based pricing doesn't support promotions
func (s *CustomerLevelPricingStrategy) SupportsPromotion() bool {
	return false
}

// SupportsTieredPricing returns false as this is level-based, not quantity-based
func (s *CustomerLevelPricingStrategy) SupportsTieredPricing() bool {
	return false
}
