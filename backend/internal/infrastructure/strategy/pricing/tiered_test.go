package pricing

import (
	"context"
	"testing"

	"github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTieredPricingStrategy_CalculatePrice(t *testing.T) {
	// Create tiered pricing with volume discounts
	// Qty 1-9: 100 per unit
	// Qty 10-49: 90 per unit (10% discount)
	// Qty 50-99: 85 per unit (15% discount)
	// Qty 100+: 80 per unit (20% discount)
	tiers := []PriceTier{
		{MinQuantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100)},
		{MinQuantity: decimal.NewFromInt(10), UnitPrice: decimal.NewFromInt(90)},
		{MinQuantity: decimal.NewFromInt(50), UnitPrice: decimal.NewFromInt(85)},
		{MinQuantity: decimal.NewFromInt(100), UnitPrice: decimal.NewFromInt(80)},
	}

	s := NewTieredPricingStrategy(tiers)
	ctx := context.Background()

	tests := []struct {
		name           string
		quantity       decimal.Decimal
		basePrice      decimal.Decimal
		expectedUnit   decimal.Decimal
		expectedTotal  decimal.Decimal
		expectedRules  []string
	}{
		{
			name:          "small quantity uses first tier",
			quantity:      decimal.NewFromInt(5),
			basePrice:     decimal.NewFromInt(100),
			expectedUnit:  decimal.NewFromInt(100),
			expectedTotal: decimal.NewFromInt(500),
			expectedRules: []string{"tiered_pricing"},
		},
		{
			name:          "quantity at tier boundary uses that tier",
			quantity:      decimal.NewFromInt(10),
			basePrice:     decimal.NewFromInt(100),
			expectedUnit:  decimal.NewFromInt(90),
			expectedTotal: decimal.NewFromInt(900),
			expectedRules: []string{"tiered_pricing"},
		},
		{
			name:          "quantity above tier uses that tier",
			quantity:      decimal.NewFromInt(25),
			basePrice:     decimal.NewFromInt(100),
			expectedUnit:  decimal.NewFromInt(90),
			expectedTotal: decimal.NewFromInt(2250),
			expectedRules: []string{"tiered_pricing"},
		},
		{
			name:          "large quantity uses highest tier",
			quantity:      decimal.NewFromInt(150),
			basePrice:     decimal.NewFromInt(100),
			expectedUnit:  decimal.NewFromInt(80),
			expectedTotal: decimal.NewFromInt(12000),
			expectedRules: []string{"tiered_pricing"},
		},
		{
			name:          "exactly at highest tier boundary",
			quantity:      decimal.NewFromInt(100),
			basePrice:     decimal.NewFromInt(100),
			expectedUnit:  decimal.NewFromInt(80),
			expectedTotal: decimal.NewFromInt(8000),
			expectedRules: []string{"tiered_pricing"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pricingCtx := strategy.PricingContext{
				TenantID:  "tenant-1",
				ProductID: "product-1",
				Quantity:  tt.quantity,
				BasePrice: tt.basePrice,
				Currency:  "CNY",
			}

			result, err := s.CalculatePrice(ctx, pricingCtx)
			require.NoError(t, err)
			assert.True(t, tt.expectedUnit.Equal(result.UnitPrice), "expected unit price %s, got %s", tt.expectedUnit, result.UnitPrice)
			assert.True(t, tt.expectedTotal.Equal(result.TotalPrice), "expected total price %s, got %s", tt.expectedTotal, result.TotalPrice)
			assert.Equal(t, tt.expectedRules, result.AppliedRules)
		})
	}
}

func TestTieredPricingStrategy_NoTiers(t *testing.T) {
	// Empty tiers should use base price
	s := NewTieredPricingStrategy([]PriceTier{})
	ctx := context.Background()

	pricingCtx := strategy.PricingContext{
		TenantID:  "tenant-1",
		ProductID: "product-1",
		Quantity:  decimal.NewFromInt(10),
		BasePrice: decimal.NewFromInt(50),
		Currency:  "CNY",
	}

	result, err := s.CalculatePrice(ctx, pricingCtx)
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(50).Equal(result.UnitPrice))
	assert.True(t, decimal.NewFromInt(500).Equal(result.TotalPrice))
	assert.Empty(t, result.AppliedRules)
}

func TestTieredPricingStrategy_UnsortedTiers(t *testing.T) {
	// Tiers provided out of order should be sorted internally
	tiers := []PriceTier{
		{MinQuantity: decimal.NewFromInt(100), UnitPrice: decimal.NewFromInt(80)},
		{MinQuantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100)},
		{MinQuantity: decimal.NewFromInt(50), UnitPrice: decimal.NewFromInt(85)},
		{MinQuantity: decimal.NewFromInt(10), UnitPrice: decimal.NewFromInt(90)},
	}

	s := NewTieredPricingStrategy(tiers)
	ctx := context.Background()

	pricingCtx := strategy.PricingContext{
		Quantity:  decimal.NewFromInt(75),
		BasePrice: decimal.NewFromInt(100),
		Currency:  "CNY",
	}

	result, err := s.CalculatePrice(ctx, pricingCtx)
	require.NoError(t, err)
	// Should hit the 50+ tier (85 per unit)
	assert.True(t, decimal.NewFromInt(85).Equal(result.UnitPrice))
}

func TestTieredPricingStrategy_DiscountCalculation(t *testing.T) {
	tiers := []PriceTier{
		{MinQuantity: decimal.NewFromInt(10), UnitPrice: decimal.NewFromInt(90)},
	}

	s := NewTieredPricingStrategy(tiers)
	ctx := context.Background()

	pricingCtx := strategy.PricingContext{
		Quantity:  decimal.NewFromInt(10),
		BasePrice: decimal.NewFromInt(100), // Base price is 100
		Currency:  "CNY",
	}

	result, err := s.CalculatePrice(ctx, pricingCtx)
	require.NoError(t, err)

	// Unit price: 90, quantity: 10, total: 900
	// Base total: 100 * 10 = 1000
	// Discount: 1000 - 900 = 100
	// Discount percent: (100/1000) * 100 = 10%
	assert.True(t, decimal.NewFromInt(100).Equal(result.DiscountAmount), "discount amount should be 100, got %s", result.DiscountAmount)
	assert.True(t, decimal.NewFromInt(10).Equal(result.DiscountPercent), "discount percent should be 10%%, got %s", result.DiscountPercent)
}

func TestTieredPricingStrategy_Properties(t *testing.T) {
	s := NewTieredPricingStrategy([]PriceTier{})

	assert.Equal(t, "tiered", s.Name())
	assert.Equal(t, strategy.StrategyTypePricing, s.Type())
	assert.False(t, s.SupportsPromotion())
	assert.True(t, s.SupportsTieredPricing())
}
