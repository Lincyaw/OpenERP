package pricing

import (
	"context"
	"testing"

	"github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomerLevelPricingStrategy_CalculatePrice(t *testing.T) {
	// Use default level discounts:
	// normal: 0%, silver: 3%, gold: 5%, platinum: 8%, vip: 10%
	s := DefaultCustomerLevelPricingStrategy()
	ctx := context.Background()

	tests := []struct {
		name                    string
		customerLevel           string
		quantity                decimal.Decimal
		basePrice               decimal.Decimal
		expectedUnitPrice       decimal.Decimal
		expectedDiscountPercent decimal.Decimal
		expectedRules           []string
	}{
		{
			name:                    "normal customer gets no discount",
			customerLevel:           "normal",
			quantity:                decimal.NewFromInt(1),
			basePrice:               decimal.NewFromInt(100),
			expectedUnitPrice:       decimal.NewFromInt(100),
			expectedDiscountPercent: decimal.Zero,
			expectedRules:           []string{},
		},
		{
			name:                    "silver customer gets 3% discount",
			customerLevel:           "silver",
			quantity:                decimal.NewFromInt(1),
			basePrice:               decimal.NewFromInt(100),
			expectedUnitPrice:       decimal.NewFromFloat(97), // 100 * 0.97 = 97
			expectedDiscountPercent: decimal.NewFromInt(3),
			expectedRules:           []string{"customer_level_discount"},
		},
		{
			name:                    "gold customer gets 5% discount",
			customerLevel:           "gold",
			quantity:                decimal.NewFromInt(1),
			basePrice:               decimal.NewFromInt(100),
			expectedUnitPrice:       decimal.NewFromInt(95), // 100 * 0.95 = 95
			expectedDiscountPercent: decimal.NewFromInt(5),
			expectedRules:           []string{"customer_level_discount"},
		},
		{
			name:                    "platinum customer gets 8% discount",
			customerLevel:           "platinum",
			quantity:                decimal.NewFromInt(1),
			basePrice:               decimal.NewFromInt(100),
			expectedUnitPrice:       decimal.NewFromInt(92), // 100 * 0.92 = 92
			expectedDiscountPercent: decimal.NewFromInt(8),
			expectedRules:           []string{"customer_level_discount"},
		},
		{
			name:                    "vip customer gets 10% discount",
			customerLevel:           "vip",
			quantity:                decimal.NewFromInt(1),
			basePrice:               decimal.NewFromInt(100),
			expectedUnitPrice:       decimal.NewFromInt(90), // 100 * 0.9 = 90
			expectedDiscountPercent: decimal.NewFromInt(10),
			expectedRules:           []string{"customer_level_discount"},
		},
		{
			name:                    "unknown level gets no discount",
			customerLevel:           "unknown",
			quantity:                decimal.NewFromInt(1),
			basePrice:               decimal.NewFromInt(100),
			expectedUnitPrice:       decimal.NewFromInt(100),
			expectedDiscountPercent: decimal.Zero,
			expectedRules:           []string{},
		},
		{
			name:                    "multiple quantity with gold discount",
			customerLevel:           "gold",
			quantity:                decimal.NewFromInt(10),
			basePrice:               decimal.NewFromInt(50),
			expectedUnitPrice:       decimal.NewFromFloat(47.5), // 50 * 0.95 = 47.5
			expectedDiscountPercent: decimal.NewFromInt(5),
			expectedRules:           []string{"customer_level_discount"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pricingCtx := strategy.PricingContext{
				TenantID:     "tenant-1",
				ProductID:    "product-1",
				CustomerType: tt.customerLevel,
				Quantity:     tt.quantity,
				BasePrice:    tt.basePrice,
				Currency:     "CNY",
			}

			result, err := s.CalculatePrice(ctx, pricingCtx)
			require.NoError(t, err)

			// Round for comparison
			resultUnitPrice := result.UnitPrice.Round(4)
			expectedUnitPrice := tt.expectedUnitPrice.Round(4)

			assert.True(t, expectedUnitPrice.Equal(resultUnitPrice),
				"expected unit price %s, got %s", expectedUnitPrice, resultUnitPrice)
			assert.True(t, tt.expectedDiscountPercent.Equal(result.DiscountPercent),
				"expected discount %s%%, got %s%%", tt.expectedDiscountPercent, result.DiscountPercent)
			assert.Equal(t, tt.expectedRules, result.AppliedRules)
		})
	}
}

func TestCustomerLevelPricingStrategy_TotalCalculation(t *testing.T) {
	s := DefaultCustomerLevelPricingStrategy()
	ctx := context.Background()

	pricingCtx := strategy.PricingContext{
		CustomerType: "gold", // 5% discount
		Quantity:     decimal.NewFromInt(20),
		BasePrice:    decimal.NewFromInt(100),
		Currency:     "CNY",
	}

	result, err := s.CalculatePrice(ctx, pricingCtx)
	require.NoError(t, err)

	// Unit price: 100 * 0.95 = 95
	// Total price: 95 * 20 = 1900
	// Base total: 100 * 20 = 2000
	// Discount amount: 2000 - 1900 = 100
	assert.True(t, decimal.NewFromInt(95).Equal(result.UnitPrice))
	assert.True(t, decimal.NewFromInt(1900).Equal(result.TotalPrice))
	assert.True(t, decimal.NewFromInt(100).Equal(result.DiscountAmount))
}

func TestCustomerLevelPricingStrategy_CustomDiscounts(t *testing.T) {
	// Create custom discounts
	customDiscounts := []CustomerLevelDiscount{
		{Level: "bronze", DiscountPercent: decimal.NewFromInt(2)},
		{Level: "silver", DiscountPercent: decimal.NewFromInt(5)},
		{Level: "gold", DiscountPercent: decimal.NewFromInt(10)},
		{Level: "diamond", DiscountPercent: decimal.NewFromInt(20)},
	}

	s := NewCustomerLevelPricingStrategy(customDiscounts)
	ctx := context.Background()

	tests := []struct {
		level           string
		expectedPercent decimal.Decimal
	}{
		{"bronze", decimal.NewFromInt(2)},
		{"silver", decimal.NewFromInt(5)},
		{"gold", decimal.NewFromInt(10)},
		{"diamond", decimal.NewFromInt(20)},
		{"unknown", decimal.Zero},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			pricingCtx := strategy.PricingContext{
				CustomerType: tt.level,
				Quantity:     decimal.NewFromInt(1),
				BasePrice:    decimal.NewFromInt(100),
				Currency:     "CNY",
			}

			result, err := s.CalculatePrice(ctx, pricingCtx)
			require.NoError(t, err)
			assert.True(t, tt.expectedPercent.Equal(result.DiscountPercent))
		})
	}
}

func TestCustomerLevelPricingStrategy_GetDiscountForLevel(t *testing.T) {
	s := DefaultCustomerLevelPricingStrategy()

	assert.True(t, decimal.Zero.Equal(s.GetDiscountForLevel("normal")))
	assert.True(t, decimal.NewFromInt(3).Equal(s.GetDiscountForLevel("silver")))
	assert.True(t, decimal.NewFromInt(5).Equal(s.GetDiscountForLevel("gold")))
	assert.True(t, decimal.NewFromInt(8).Equal(s.GetDiscountForLevel("platinum")))
	assert.True(t, decimal.NewFromInt(10).Equal(s.GetDiscountForLevel("vip")))
	assert.True(t, decimal.Zero.Equal(s.GetDiscountForLevel("unknown")))
}

func TestCustomerLevelPricingStrategy_Properties(t *testing.T) {
	s := DefaultCustomerLevelPricingStrategy()

	assert.Equal(t, "customer_level", s.Name())
	assert.Equal(t, strategy.StrategyTypePricing, s.Type())
	assert.False(t, s.SupportsPromotion())
	assert.False(t, s.SupportsTieredPricing())
}

func TestCustomerLevelPricingStrategy_GetDiscounts(t *testing.T) {
	s := DefaultCustomerLevelPricingStrategy()

	discounts := s.GetDiscounts()
	assert.Len(t, discounts, 5)
	assert.Contains(t, discounts, "normal")
	assert.Contains(t, discounts, "silver")
	assert.Contains(t, discounts, "gold")
	assert.Contains(t, discounts, "platinum")
	assert.Contains(t, discounts, "vip")
}
