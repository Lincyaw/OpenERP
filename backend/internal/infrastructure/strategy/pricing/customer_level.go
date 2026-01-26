package pricing

import (
	"context"

	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CustomerLevelProvider provides customer level lookup for pricing strategies.
// This interface allows the pricing strategy to query customer levels dynamically
// from the domain layer without hardcoding discount mappings.
type CustomerLevelProvider interface {
	// GetCustomerLevel retrieves the customer level by code for a tenant.
	// Returns the CustomerLevel value object with full details (code, name, discountRate).
	// Returns an error if the level is not found.
	GetCustomerLevel(ctx context.Context, tenantID uuid.UUID, levelCode string) (partner.CustomerLevel, error)

	// GetAllCustomerLevels retrieves all active customer levels for a tenant.
	// Returns an empty slice if no levels are configured.
	GetAllCustomerLevels(ctx context.Context, tenantID uuid.UUID) ([]partner.CustomerLevel, error)
}

// CustomerLevelPricingStrategy implements customer-level based pricing
// Different customer levels (normal, silver, gold, platinum, vip) get different discounts.
//
// The strategy can operate in two modes:
// 1. Dynamic mode (with provider): Looks up discount rates from the CustomerLevel domain objects
// 2. Static mode (without provider): Uses hardcoded fallback discount mappings
//
// When a provider is configured, the strategy will query the actual CustomerLevel
// definitions for the tenant, respecting tenant-specific discount rates.
type CustomerLevelPricingStrategy struct {
	strategy.BaseStrategy
	provider          CustomerLevelProvider
	fallbackDiscounts map[string]decimal.Decimal // Used when provider is nil or lookup fails
}

// NewCustomerLevelPricingStrategy creates a new customer level pricing strategy with a provider.
// The provider is used to look up discount rates dynamically from CustomerLevel domain objects.
// If provider is nil, the strategy will use fallback discounts.
func NewCustomerLevelPricingStrategy(provider CustomerLevelProvider) *CustomerLevelPricingStrategy {
	return &CustomerLevelPricingStrategy{
		BaseStrategy: strategy.NewBaseStrategy(
			"customer_level",
			strategy.StrategyTypePricing,
			"Customer level-based pricing with tier discounts",
		),
		provider:          provider,
		fallbackDiscounts: defaultFallbackDiscounts(),
	}
}

// NewCustomerLevelPricingStrategyWithFallback creates a strategy with custom fallback discounts.
// The fallback discounts are used when the provider is nil or when lookup fails.
func NewCustomerLevelPricingStrategyWithFallback(
	provider CustomerLevelProvider,
	fallbackDiscounts map[string]decimal.Decimal,
) *CustomerLevelPricingStrategy {
	return &CustomerLevelPricingStrategy{
		BaseStrategy: strategy.NewBaseStrategy(
			"customer_level",
			strategy.StrategyTypePricing,
			"Customer level-based pricing with tier discounts",
		),
		provider:          provider,
		fallbackDiscounts: fallbackDiscounts,
	}
}

// DefaultCustomerLevelPricingStrategy creates a strategy with default level discounts.
// This creates a strategy without a provider, using static fallback discounts:
// - normal: 0% discount
// - silver: 3% discount
// - gold: 5% discount
// - platinum: 8% discount
// - vip: 10% discount
//
// Note: For production use, prefer NewCustomerLevelPricingStrategy with a provider
// to enable dynamic, tenant-specific discount rates.
func DefaultCustomerLevelPricingStrategy() *CustomerLevelPricingStrategy {
	return NewCustomerLevelPricingStrategy(nil)
}

// defaultFallbackDiscounts returns the default discount percentages for standard levels.
// These match the standard CustomerLevel definitions in the partner domain.
// Discounts are stored as percentages (e.g., 3 for 3%, not 0.03).
func defaultFallbackDiscounts() map[string]decimal.Decimal {
	return map[string]decimal.Decimal{
		partner.CustomerLevelCodeNormal:   decimal.Zero,
		partner.CustomerLevelCodeSilver:   decimal.NewFromInt(3),
		partner.CustomerLevelCodeGold:     decimal.NewFromInt(5),
		partner.CustomerLevelCodePlatinum: decimal.NewFromInt(8),
		partner.CustomerLevelCodeVIP:      decimal.NewFromInt(10),
	}
}

// SetProvider sets the customer level provider for dynamic lookup.
// This allows configuring the provider after construction.
//
// WARNING: This method is NOT thread-safe. It should only be called during
// initialization/setup, before the strategy is used concurrently by HTTP handlers.
// For production use, prefer setting the provider at construction time via
// NewCustomerLevelPricingStrategy.
func (s *CustomerLevelPricingStrategy) SetProvider(provider CustomerLevelProvider) {
	s.provider = provider
}

// HasProvider returns true if a provider is configured.
func (s *CustomerLevelPricingStrategy) HasProvider() bool {
	return s.provider != nil
}

// GetDiscounts returns a copy of the fallback discount configuration.
// Note: These are fallback values. Actual discounts may differ based on tenant configuration.
func (s *CustomerLevelPricingStrategy) GetDiscounts() map[string]decimal.Decimal {
	result := make(map[string]decimal.Decimal)
	for k, v := range s.fallbackDiscounts {
		result[k] = v
	}
	return result
}

// GetDiscountForLevel returns the discount percentage for a given customer level code.
// This method uses the fallback discounts and does not query the provider.
// For dynamic lookup, use GetDiscountForLevelWithContext.
func (s *CustomerLevelPricingStrategy) GetDiscountForLevel(level string) decimal.Decimal {
	if discount, ok := s.fallbackDiscounts[level]; ok {
		return discount
	}
	return decimal.Zero
}

// GetDiscountForLevelWithContext returns the discount percentage for a customer level,
// using the provider if available for dynamic lookup.
// Falls back to static discounts if provider is nil or lookup fails.
func (s *CustomerLevelPricingStrategy) GetDiscountForLevelWithContext(
	ctx context.Context,
	tenantID uuid.UUID,
	levelCode string,
) decimal.Decimal {
	// If no provider, use fallback
	if s.provider == nil {
		return s.GetDiscountForLevel(levelCode)
	}

	// Try to get the level from the provider
	customerLevel, err := s.provider.GetCustomerLevel(ctx, tenantID, levelCode)
	if err != nil {
		// Fallback to static discounts on error
		return s.GetDiscountForLevel(levelCode)
	}

	// CustomerLevel stores discount rate as decimal (0.05 for 5%)
	// Convert to percentage (5 for 5%) for calculation consistency
	return customerLevel.DiscountPercent()
}

// CalculatePrice calculates the price based on customer level discount.
// If a provider is configured, it will look up the actual discount rate from
// the CustomerLevel domain object for the tenant. Otherwise, it uses fallback discounts.
func (s *CustomerLevelPricingStrategy) CalculatePrice(
	ctx context.Context,
	pricingCtx strategy.PricingContext,
) (strategy.PricingResult, error) {
	// Parse tenant ID for provider lookup
	var tenantID uuid.UUID
	if s.provider != nil && pricingCtx.TenantID != "" {
		var err error
		tenantID, err = uuid.Parse(pricingCtx.TenantID)
		if err != nil {
			// Invalid tenant ID, fall back to static discounts
			tenantID = uuid.Nil
		}
	}

	// Get discount percentage for customer level
	var discountPercent decimal.Decimal
	if s.provider != nil && tenantID != uuid.Nil {
		discountPercent = s.GetDiscountForLevelWithContext(ctx, tenantID, pricingCtx.CustomerType)
	} else {
		discountPercent = s.GetDiscountForLevel(pricingCtx.CustomerType)
	}

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

// SupportsPromotion returns false as level-based pricing doesn't support promotions.
func (s *CustomerLevelPricingStrategy) SupportsPromotion() bool {
	return false
}

// SupportsTieredPricing returns false as this is level-based, not quantity-based.
func (s *CustomerLevelPricingStrategy) SupportsTieredPricing() bool {
	return false
}

// CustomerLevelDiscount maps customer levels to discount percentages.
// This struct is kept for backwards compatibility with existing code that
// initializes the strategy with explicit discount configurations.
// Deprecated: Prefer using CustomerLevelProvider for dynamic discount lookup.
type CustomerLevelDiscount struct {
	Level           string          `json:"level"`
	DiscountPercent decimal.Decimal `json:"discount_percent"`
}

// NewCustomerLevelPricingStrategyFromDiscounts creates a new customer level pricing strategy
// from a list of CustomerLevelDiscount configurations.
// This is kept for backwards compatibility with existing code.
// Deprecated: Prefer using NewCustomerLevelPricingStrategy with a CustomerLevelProvider.
func NewCustomerLevelPricingStrategyFromDiscounts(discounts []CustomerLevelDiscount) *CustomerLevelPricingStrategy {
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
		provider:          nil,
		fallbackDiscounts: discountMap,
	}
}

// Ensure CustomerLevelPricingStrategy implements strategy.PricingStrategy
var _ strategy.PricingStrategy = (*CustomerLevelPricingStrategy)(nil)
