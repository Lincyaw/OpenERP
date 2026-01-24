package strategy

import (
	"context"
	"sort"

	"github.com/shopspring/decimal"
)

// PricingMethodType represents the type of pricing method
type PricingMethodType string

const (
	PricingMethodStandard         PricingMethodType = "standard"
	PricingMethodTiered           PricingMethodType = "tiered"
	PricingMethodCustomerSpecific PricingMethodType = "customer_specific"
)

// String returns the string representation of the pricing method
func (m PricingMethodType) String() string {
	return string(m)
}

// PriceTier represents a price tier for tiered pricing
type PriceTier struct {
	MinQuantity decimal.Decimal // Minimum quantity for this tier (inclusive)
	MaxQuantity decimal.Decimal // Maximum quantity for this tier (exclusive), zero means no limit
	UnitPrice   decimal.Decimal // Unit price for this tier
}

// CustomerPrice represents a customer-specific price
type CustomerPrice struct {
	CustomerID    string
	CustomerType  string          // Customer type (e.g., "vip", "wholesale", "retail")
	ProductID     string
	UnitPrice     decimal.Decimal
	MinQuantity   decimal.Decimal // Minimum quantity required for this price
	DiscountRate  decimal.Decimal // Discount rate as a percentage (0-100)
	PriorityOrder int             // Lower number = higher priority
}

// StandardPricingStrategy uses the base product price directly
type StandardPricingStrategy struct {
	BaseStrategy
}

// NewStandardPricingStrategy creates a new standard pricing strategy
func NewStandardPricingStrategy() *StandardPricingStrategy {
	return &StandardPricingStrategy{
		BaseStrategy: NewBaseStrategy(
			"standard",
			StrategyTypePricing,
			"Standard pricing uses the product's base selling price",
		),
	}
}

// CalculatePrice calculates the final price using standard pricing
func (s *StandardPricingStrategy) CalculatePrice(ctx context.Context, pricingCtx PricingContext) (PricingResult, error) {
	totalPrice := pricingCtx.BasePrice.Mul(pricingCtx.Quantity)

	return PricingResult{
		UnitPrice:       pricingCtx.BasePrice,
		TotalPrice:      totalPrice,
		DiscountAmount:  decimal.Zero,
		DiscountPercent: decimal.Zero,
		Currency:        pricingCtx.Currency,
		AppliedRules:    []string{"standard_pricing"},
	}, nil
}

// SupportsPromotion returns false for standard pricing
func (s *StandardPricingStrategy) SupportsPromotion() bool {
	return false
}

// SupportsTieredPricing returns false for standard pricing
func (s *StandardPricingStrategy) SupportsTieredPricing() bool {
	return false
}

// TieredPricingStrategy applies pricing based on quantity tiers
type TieredPricingStrategy struct {
	BaseStrategy
	Tiers []PriceTier // Price tiers sorted by MinQuantity ascending
}

// NewTieredPricingStrategy creates a new tiered pricing strategy
func NewTieredPricingStrategy(tiers []PriceTier) *TieredPricingStrategy {
	sortedTiers := make([]PriceTier, len(tiers))
	copy(sortedTiers, tiers)
	sort.Slice(sortedTiers, func(i, j int) bool {
		return sortedTiers[i].MinQuantity.LessThan(sortedTiers[j].MinQuantity)
	})

	return &TieredPricingStrategy{
		BaseStrategy: NewBaseStrategy(
			"tiered",
			StrategyTypePricing,
			"Tiered pricing applies different prices based on quantity ranges",
		),
		Tiers: sortedTiers,
	}
}

// CalculatePrice calculates the final price using tiered pricing
func (s *TieredPricingStrategy) CalculatePrice(ctx context.Context, pricingCtx PricingContext) (PricingResult, error) {
	unitPrice := s.findApplicableTierPrice(pricingCtx.Quantity, pricingCtx.BasePrice)
	totalPrice := unitPrice.Mul(pricingCtx.Quantity)

	originalTotal := pricingCtx.BasePrice.Mul(pricingCtx.Quantity)
	discountAmount := originalTotal.Sub(totalPrice)
	var discountPercent decimal.Decimal
	if !originalTotal.IsZero() {
		discountPercent = discountAmount.Div(originalTotal).Mul(decimal.NewFromInt(100))
	}

	appliedRules := []string{"tiered_pricing"}
	if discountAmount.IsPositive() {
		appliedRules = append(appliedRules, "quantity_discount")
	}

	return PricingResult{
		UnitPrice:       unitPrice,
		TotalPrice:      totalPrice,
		DiscountAmount:  discountAmount,
		DiscountPercent: discountPercent.Round(2),
		Currency:        pricingCtx.Currency,
		AppliedRules:    appliedRules,
	}, nil
}

// findApplicableTierPrice finds the applicable tier price for the given quantity
func (s *TieredPricingStrategy) findApplicableTierPrice(quantity, basePrice decimal.Decimal) decimal.Decimal {
	if len(s.Tiers) == 0 {
		return basePrice
	}

	// Find the applicable tier (highest tier that quantity qualifies for)
	var applicablePrice decimal.Decimal = basePrice
	for _, tier := range s.Tiers {
		if quantity.GreaterThanOrEqual(tier.MinQuantity) {
			// Check if there's a max quantity limit
			if tier.MaxQuantity.IsZero() || quantity.LessThan(tier.MaxQuantity) {
				applicablePrice = tier.UnitPrice
			}
		}
	}

	return applicablePrice
}

// SupportsPromotion returns false for tiered pricing
func (s *TieredPricingStrategy) SupportsPromotion() bool {
	return false
}

// SupportsTieredPricing returns true for tiered pricing
func (s *TieredPricingStrategy) SupportsTieredPricing() bool {
	return true
}

// CustomerSpecificPricingStrategy applies customer-specific prices
type CustomerSpecificPricingStrategy struct {
	BaseStrategy
	CustomerPrices   []CustomerPrice        // Customer-specific prices
	FallbackStrategy PricingStrategy        // Fallback strategy if no customer price found
}

// NewCustomerSpecificPricingStrategy creates a new customer-specific pricing strategy
func NewCustomerSpecificPricingStrategy(customerPrices []CustomerPrice, fallback PricingStrategy) *CustomerSpecificPricingStrategy {
	// Sort by priority (lower priority number = higher priority)
	sortedPrices := make([]CustomerPrice, len(customerPrices))
	copy(sortedPrices, customerPrices)
	sort.Slice(sortedPrices, func(i, j int) bool {
		return sortedPrices[i].PriorityOrder < sortedPrices[j].PriorityOrder
	})

	if fallback == nil {
		fallback = NewStandardPricingStrategy()
	}

	return &CustomerSpecificPricingStrategy{
		BaseStrategy: NewBaseStrategy(
			"customer_specific",
			StrategyTypePricing,
			"Customer-specific pricing applies special prices based on customer agreements",
		),
		CustomerPrices:   sortedPrices,
		FallbackStrategy: fallback,
	}
}

// CalculatePrice calculates the final price using customer-specific pricing
func (s *CustomerSpecificPricingStrategy) CalculatePrice(ctx context.Context, pricingCtx PricingContext) (PricingResult, error) {
	// Try to find a customer-specific price
	customerPrice := s.findCustomerPrice(pricingCtx)
	if customerPrice == nil {
		// Fall back to the fallback strategy
		return s.FallbackStrategy.CalculatePrice(ctx, pricingCtx)
	}

	var unitPrice decimal.Decimal
	appliedRules := []string{"customer_specific_pricing"}

	// Apply either fixed price or discount rate
	if !customerPrice.UnitPrice.IsZero() {
		unitPrice = customerPrice.UnitPrice
		appliedRules = append(appliedRules, "fixed_customer_price")
	} else if !customerPrice.DiscountRate.IsZero() {
		// Apply discount rate to base price
		discountMultiplier := decimal.NewFromInt(100).Sub(customerPrice.DiscountRate).Div(decimal.NewFromInt(100))
		unitPrice = pricingCtx.BasePrice.Mul(discountMultiplier)
		appliedRules = append(appliedRules, "customer_discount_rate")
	} else {
		unitPrice = pricingCtx.BasePrice
	}

	totalPrice := unitPrice.Mul(pricingCtx.Quantity)
	originalTotal := pricingCtx.BasePrice.Mul(pricingCtx.Quantity)
	discountAmount := originalTotal.Sub(totalPrice)
	var discountPercent decimal.Decimal
	if !originalTotal.IsZero() {
		discountPercent = discountAmount.Div(originalTotal).Mul(decimal.NewFromInt(100))
	}

	return PricingResult{
		UnitPrice:       unitPrice,
		TotalPrice:      totalPrice,
		DiscountAmount:  discountAmount,
		DiscountPercent: discountPercent.Round(2),
		Currency:        pricingCtx.Currency,
		AppliedRules:    appliedRules,
	}, nil
}

// findCustomerPrice finds the applicable customer-specific price
func (s *CustomerSpecificPricingStrategy) findCustomerPrice(pricingCtx PricingContext) *CustomerPrice {
	for _, cp := range s.CustomerPrices {
		// Check if customer matches
		customerMatches := cp.CustomerID == pricingCtx.CustomerID ||
			(cp.CustomerID == "" && cp.CustomerType == pricingCtx.CustomerType)

		if !customerMatches {
			continue
		}

		// Check if product matches (empty ProductID means applies to all products)
		productMatches := cp.ProductID == "" || cp.ProductID == pricingCtx.ProductID

		if !productMatches {
			continue
		}

		// Check minimum quantity requirement
		if !cp.MinQuantity.IsZero() && pricingCtx.Quantity.LessThan(cp.MinQuantity) {
			continue
		}

		return &cp
	}

	return nil
}

// SupportsPromotion returns false for customer-specific pricing
func (s *CustomerSpecificPricingStrategy) SupportsPromotion() bool {
	return false
}

// SupportsTieredPricing returns false for customer-specific pricing
func (s *CustomerSpecificPricingStrategy) SupportsTieredPricing() bool {
	return false
}

// CombinedPricingStrategy combines multiple pricing strategies and applies the best price
type CombinedPricingStrategy struct {
	BaseStrategy
	Strategies   []PricingStrategy
	SelectBest   bool // If true, select the best (lowest) price; if false, apply all sequentially
}

// NewCombinedPricingStrategy creates a new combined pricing strategy
func NewCombinedPricingStrategy(strategies []PricingStrategy, selectBest bool) *CombinedPricingStrategy {
	return &CombinedPricingStrategy{
		BaseStrategy: NewBaseStrategy(
			"combined",
			StrategyTypePricing,
			"Combined pricing evaluates multiple strategies and applies the best price",
		),
		Strategies: strategies,
		SelectBest: selectBest,
	}
}

// CalculatePrice calculates the final price using combined strategies
func (s *CombinedPricingStrategy) CalculatePrice(ctx context.Context, pricingCtx PricingContext) (PricingResult, error) {
	if len(s.Strategies) == 0 {
		return NewStandardPricingStrategy().CalculatePrice(ctx, pricingCtx)
	}

	if s.SelectBest {
		return s.selectBestPrice(ctx, pricingCtx)
	}

	return s.applySequentially(ctx, pricingCtx)
}

// selectBestPrice evaluates all strategies and returns the lowest price
func (s *CombinedPricingStrategy) selectBestPrice(ctx context.Context, pricingCtx PricingContext) (PricingResult, error) {
	var bestResult PricingResult
	bestInitialized := false

	for _, strategy := range s.Strategies {
		result, err := strategy.CalculatePrice(ctx, pricingCtx)
		if err != nil {
			continue
		}

		if !bestInitialized || result.TotalPrice.LessThan(bestResult.TotalPrice) {
			bestResult = result
			bestInitialized = true
		}
	}

	if !bestInitialized {
		return NewStandardPricingStrategy().CalculatePrice(ctx, pricingCtx)
	}

	bestResult.AppliedRules = append(bestResult.AppliedRules, "best_price_selected")
	return bestResult, nil
}

// applySequentially applies strategies in order (last one wins if applicable)
func (s *CombinedPricingStrategy) applySequentially(ctx context.Context, pricingCtx PricingContext) (PricingResult, error) {
	currentCtx := pricingCtx
	var lastResult PricingResult
	var allRules []string

	for _, strategy := range s.Strategies {
		result, err := strategy.CalculatePrice(ctx, currentCtx)
		if err != nil {
			continue
		}

		lastResult = result
		allRules = append(allRules, result.AppliedRules...)

		// Update the base price for the next strategy
		currentCtx.BasePrice = result.UnitPrice
	}

	lastResult.AppliedRules = allRules
	return lastResult, nil
}

// SupportsPromotion returns true if any strategy supports promotions
func (s *CombinedPricingStrategy) SupportsPromotion() bool {
	for _, strategy := range s.Strategies {
		if strategy.SupportsPromotion() {
			return true
		}
	}
	return false
}

// SupportsTieredPricing returns true if any strategy supports tiered pricing
func (s *CombinedPricingStrategy) SupportsTieredPricing() bool {
	for _, strategy := range s.Strategies {
		if strategy.SupportsTieredPricing() {
			return true
		}
	}
	return false
}
