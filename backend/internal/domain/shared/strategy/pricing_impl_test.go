package strategy

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStandardPricingStrategy(t *testing.T) {
	strategy := NewStandardPricingStrategy()

	t.Run("Name and Type", func(t *testing.T) {
		assert.Equal(t, "standard", strategy.Name())
		assert.Equal(t, StrategyTypePricing, strategy.Type())
		assert.False(t, strategy.SupportsPromotion())
		assert.False(t, strategy.SupportsTieredPricing())
	})

	t.Run("CalculatePrice with base price", func(t *testing.T) {
		ctx := context.Background()
		pricingCtx := PricingContext{
			TenantID:   "tenant-1",
			ProductID:  "product-1",
			CustomerID: "customer-1",
			Quantity:   decimal.NewFromInt(10),
			BasePrice:  decimal.NewFromFloat(100.00),
			Currency:   "CNY",
		}

		result, err := strategy.CalculatePrice(ctx, pricingCtx)
		require.NoError(t, err)

		assert.True(t, result.UnitPrice.Equal(decimal.NewFromFloat(100.00)))
		assert.True(t, result.TotalPrice.Equal(decimal.NewFromFloat(1000.00)))
		assert.True(t, result.DiscountAmount.IsZero())
		assert.True(t, result.DiscountPercent.IsZero())
		assert.Equal(t, "CNY", result.Currency)
		assert.Contains(t, result.AppliedRules, "standard_pricing")
	})

	t.Run("CalculatePrice with zero quantity", func(t *testing.T) {
		ctx := context.Background()
		pricingCtx := PricingContext{
			Quantity:  decimal.Zero,
			BasePrice: decimal.NewFromFloat(100.00),
			Currency:  "CNY",
		}

		result, err := strategy.CalculatePrice(ctx, pricingCtx)
		require.NoError(t, err)

		assert.True(t, result.TotalPrice.IsZero())
	})

	t.Run("CalculatePrice with decimal quantity", func(t *testing.T) {
		ctx := context.Background()
		pricingCtx := PricingContext{
			Quantity:  decimal.NewFromFloat(2.5),
			BasePrice: decimal.NewFromFloat(10.00),
			Currency:  "CNY",
		}

		result, err := strategy.CalculatePrice(ctx, pricingCtx)
		require.NoError(t, err)

		assert.True(t, result.TotalPrice.Equal(decimal.NewFromFloat(25.00)))
	})
}

func TestTieredPricingStrategy(t *testing.T) {
	tiers := []PriceTier{
		{MinQuantity: decimal.NewFromInt(1), MaxQuantity: decimal.NewFromInt(10), UnitPrice: decimal.NewFromFloat(100.00)},
		{MinQuantity: decimal.NewFromInt(10), MaxQuantity: decimal.NewFromInt(50), UnitPrice: decimal.NewFromFloat(90.00)},
		{MinQuantity: decimal.NewFromInt(50), MaxQuantity: decimal.NewFromInt(100), UnitPrice: decimal.NewFromFloat(80.00)},
		{MinQuantity: decimal.NewFromInt(100), MaxQuantity: decimal.Zero, UnitPrice: decimal.NewFromFloat(70.00)}, // No upper limit
	}

	strategy := NewTieredPricingStrategy(tiers)

	t.Run("Name and Type", func(t *testing.T) {
		assert.Equal(t, "tiered", strategy.Name())
		assert.Equal(t, StrategyTypePricing, strategy.Type())
		assert.False(t, strategy.SupportsPromotion())
		assert.True(t, strategy.SupportsTieredPricing())
	})

	t.Run("Tier 1 - 5 units", func(t *testing.T) {
		ctx := context.Background()
		pricingCtx := PricingContext{
			Quantity:  decimal.NewFromInt(5),
			BasePrice: decimal.NewFromFloat(100.00),
			Currency:  "CNY",
		}

		result, err := strategy.CalculatePrice(ctx, pricingCtx)
		require.NoError(t, err)

		assert.True(t, result.UnitPrice.Equal(decimal.NewFromFloat(100.00)))
		assert.True(t, result.TotalPrice.Equal(decimal.NewFromFloat(500.00)))
		assert.True(t, result.DiscountAmount.IsZero())
	})

	t.Run("Tier 2 - 15 units", func(t *testing.T) {
		ctx := context.Background()
		pricingCtx := PricingContext{
			Quantity:  decimal.NewFromInt(15),
			BasePrice: decimal.NewFromFloat(100.00),
			Currency:  "CNY",
		}

		result, err := strategy.CalculatePrice(ctx, pricingCtx)
		require.NoError(t, err)

		assert.True(t, result.UnitPrice.Equal(decimal.NewFromFloat(90.00)))
		assert.True(t, result.TotalPrice.Equal(decimal.NewFromFloat(1350.00)))
		// Discount: (100 * 15) - (90 * 15) = 1500 - 1350 = 150
		assert.True(t, result.DiscountAmount.Equal(decimal.NewFromFloat(150.00)))
		// Discount percent: 150 / 1500 * 100 = 10%
		assert.True(t, result.DiscountPercent.Equal(decimal.NewFromFloat(10.00)))
		assert.Contains(t, result.AppliedRules, "tiered_pricing")
		assert.Contains(t, result.AppliedRules, "quantity_discount")
	})

	t.Run("Tier 3 - 75 units", func(t *testing.T) {
		ctx := context.Background()
		pricingCtx := PricingContext{
			Quantity:  decimal.NewFromInt(75),
			BasePrice: decimal.NewFromFloat(100.00),
			Currency:  "CNY",
		}

		result, err := strategy.CalculatePrice(ctx, pricingCtx)
		require.NoError(t, err)

		assert.True(t, result.UnitPrice.Equal(decimal.NewFromFloat(80.00)))
		assert.True(t, result.TotalPrice.Equal(decimal.NewFromFloat(6000.00)))
		// Discount: 20%
		assert.True(t, result.DiscountPercent.Equal(decimal.NewFromFloat(20.00)))
	})

	t.Run("Tier 4 - 150 units (no upper limit)", func(t *testing.T) {
		ctx := context.Background()
		pricingCtx := PricingContext{
			Quantity:  decimal.NewFromInt(150),
			BasePrice: decimal.NewFromFloat(100.00),
			Currency:  "CNY",
		}

		result, err := strategy.CalculatePrice(ctx, pricingCtx)
		require.NoError(t, err)

		assert.True(t, result.UnitPrice.Equal(decimal.NewFromFloat(70.00)))
		assert.True(t, result.TotalPrice.Equal(decimal.NewFromFloat(10500.00)))
		// Discount: 30%
		assert.True(t, result.DiscountPercent.Equal(decimal.NewFromFloat(30.00)))
	})

	t.Run("Empty tiers uses base price", func(t *testing.T) {
		emptyStrategy := NewTieredPricingStrategy([]PriceTier{})
		ctx := context.Background()
		pricingCtx := PricingContext{
			Quantity:  decimal.NewFromInt(10),
			BasePrice: decimal.NewFromFloat(100.00),
			Currency:  "CNY",
		}

		result, err := emptyStrategy.CalculatePrice(ctx, pricingCtx)
		require.NoError(t, err)

		assert.True(t, result.UnitPrice.Equal(decimal.NewFromFloat(100.00)))
	})

	t.Run("Boundary - exactly at tier minimum", func(t *testing.T) {
		ctx := context.Background()
		pricingCtx := PricingContext{
			Quantity:  decimal.NewFromInt(10), // Exactly at tier 2 minimum
			BasePrice: decimal.NewFromFloat(100.00),
			Currency:  "CNY",
		}

		result, err := strategy.CalculatePrice(ctx, pricingCtx)
		require.NoError(t, err)

		assert.True(t, result.UnitPrice.Equal(decimal.NewFromFloat(90.00)))
	})
}

func TestCustomerSpecificPricingStrategy(t *testing.T) {
	customerPrices := []CustomerPrice{
		{
			CustomerID:    "vip-customer-1",
			ProductID:     "product-1",
			UnitPrice:     decimal.NewFromFloat(80.00),
			PriorityOrder: 1,
		},
		{
			CustomerID:    "wholesale-customer",
			ProductID:     "", // Applies to all products
			DiscountRate:  decimal.NewFromFloat(15.00),
			MinQuantity:   decimal.NewFromInt(10),
			PriorityOrder: 2,
		},
		{
			CustomerID:   "",
			CustomerType: "vip",
			ProductID:    "",
			DiscountRate: decimal.NewFromFloat(10.00),
			PriorityOrder: 3,
		},
	}

	strategy := NewCustomerSpecificPricingStrategy(customerPrices, nil)

	t.Run("Name and Type", func(t *testing.T) {
		assert.Equal(t, "customer_specific", strategy.Name())
		assert.Equal(t, StrategyTypePricing, strategy.Type())
		assert.False(t, strategy.SupportsPromotion())
		assert.False(t, strategy.SupportsTieredPricing())
	})

	t.Run("VIP customer fixed price", func(t *testing.T) {
		ctx := context.Background()
		pricingCtx := PricingContext{
			CustomerID: "vip-customer-1",
			ProductID:  "product-1",
			Quantity:   decimal.NewFromInt(5),
			BasePrice:  decimal.NewFromFloat(100.00),
			Currency:   "CNY",
		}

		result, err := strategy.CalculatePrice(ctx, pricingCtx)
		require.NoError(t, err)

		assert.True(t, result.UnitPrice.Equal(decimal.NewFromFloat(80.00)))
		assert.True(t, result.TotalPrice.Equal(decimal.NewFromFloat(400.00)))
		assert.True(t, result.DiscountPercent.Equal(decimal.NewFromFloat(20.00)))
		assert.Contains(t, result.AppliedRules, "customer_specific_pricing")
		assert.Contains(t, result.AppliedRules, "fixed_customer_price")
	})

	t.Run("Wholesale customer discount with min quantity", func(t *testing.T) {
		ctx := context.Background()
		pricingCtx := PricingContext{
			CustomerID: "wholesale-customer",
			ProductID:  "product-2",
			Quantity:   decimal.NewFromInt(20),
			BasePrice:  decimal.NewFromFloat(100.00),
			Currency:   "CNY",
		}

		result, err := strategy.CalculatePrice(ctx, pricingCtx)
		require.NoError(t, err)

		// 15% discount: 100 * (100-15)/100 = 85
		assert.True(t, result.UnitPrice.Equal(decimal.NewFromFloat(85.00)))
		assert.True(t, result.TotalPrice.Equal(decimal.NewFromFloat(1700.00)))
		assert.Contains(t, result.AppliedRules, "customer_discount_rate")
	})

	t.Run("Wholesale customer below min quantity - falls back", func(t *testing.T) {
		ctx := context.Background()
		pricingCtx := PricingContext{
			CustomerID: "wholesale-customer",
			ProductID:  "product-2",
			Quantity:   decimal.NewFromInt(5), // Below min quantity of 10
			BasePrice:  decimal.NewFromFloat(100.00),
			Currency:   "CNY",
		}

		result, err := strategy.CalculatePrice(ctx, pricingCtx)
		require.NoError(t, err)

		// Should fall back to standard pricing
		assert.True(t, result.UnitPrice.Equal(decimal.NewFromFloat(100.00)))
	})

	t.Run("Customer type based discount", func(t *testing.T) {
		ctx := context.Background()
		pricingCtx := PricingContext{
			CustomerID:   "another-vip",
			CustomerType: "vip",
			ProductID:    "product-3",
			Quantity:     decimal.NewFromInt(5),
			BasePrice:    decimal.NewFromFloat(100.00),
			Currency:     "CNY",
		}

		result, err := strategy.CalculatePrice(ctx, pricingCtx)
		require.NoError(t, err)

		// 10% discount for VIP type: 100 * (100-10)/100 = 90
		assert.True(t, result.UnitPrice.Equal(decimal.NewFromFloat(90.00)))
	})

	t.Run("Unknown customer falls back to standard", func(t *testing.T) {
		ctx := context.Background()
		pricingCtx := PricingContext{
			CustomerID:   "unknown-customer",
			CustomerType: "regular",
			ProductID:    "product-1",
			Quantity:     decimal.NewFromInt(5),
			BasePrice:    decimal.NewFromFloat(100.00),
			Currency:     "CNY",
		}

		result, err := strategy.CalculatePrice(ctx, pricingCtx)
		require.NoError(t, err)

		assert.True(t, result.UnitPrice.Equal(decimal.NewFromFloat(100.00)))
		assert.Contains(t, result.AppliedRules, "standard_pricing")
	})
}

func TestCombinedPricingStrategy(t *testing.T) {
	t.Run("SelectBest - chooses lowest price", func(t *testing.T) {
		// Standard strategy: price = 100
		standardStrategy := NewStandardPricingStrategy()

		// Tiered strategy: for qty 20, price = 90
		tieredStrategy := NewTieredPricingStrategy([]PriceTier{
			{MinQuantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromFloat(100.00)},
			{MinQuantity: decimal.NewFromInt(10), UnitPrice: decimal.NewFromFloat(90.00)},
		})

		// Customer strategy: price = 85
		customerStrategy := NewCustomerSpecificPricingStrategy([]CustomerPrice{
			{CustomerID: "customer-1", UnitPrice: decimal.NewFromFloat(85.00)},
		}, nil)

		combined := NewCombinedPricingStrategy(
			[]PricingStrategy{standardStrategy, tieredStrategy, customerStrategy},
			true, // Select best
		)

		ctx := context.Background()
		pricingCtx := PricingContext{
			CustomerID: "customer-1",
			Quantity:   decimal.NewFromInt(20),
			BasePrice:  decimal.NewFromFloat(100.00),
			Currency:   "CNY",
		}

		result, err := combined.CalculatePrice(ctx, pricingCtx)
		require.NoError(t, err)

		// Should select the customer-specific price of 85 (lowest)
		assert.True(t, result.UnitPrice.Equal(decimal.NewFromFloat(85.00)))
		assert.Contains(t, result.AppliedRules, "best_price_selected")
	})

	t.Run("Sequential - applies in order", func(t *testing.T) {
		// First strategy: 10% discount
		customerStrategy := NewCustomerSpecificPricingStrategy([]CustomerPrice{
			{CustomerID: "customer-1", DiscountRate: decimal.NewFromFloat(10.00)},
		}, nil)

		// Second strategy: another 5% discount on the discounted price
		secondCustomerStrategy := NewCustomerSpecificPricingStrategy([]CustomerPrice{
			{CustomerID: "customer-1", DiscountRate: decimal.NewFromFloat(5.00)},
		}, nil)

		combined := NewCombinedPricingStrategy(
			[]PricingStrategy{customerStrategy, secondCustomerStrategy},
			false, // Apply sequentially
		)

		ctx := context.Background()
		pricingCtx := PricingContext{
			CustomerID: "customer-1",
			Quantity:   decimal.NewFromInt(10),
			BasePrice:  decimal.NewFromFloat(100.00),
			Currency:   "CNY",
		}

		result, err := combined.CalculatePrice(ctx, pricingCtx)
		require.NoError(t, err)

		// First: 100 * 0.9 = 90
		// Second: 90 * 0.95 = 85.5
		assert.True(t, result.UnitPrice.Equal(decimal.NewFromFloat(85.5)))
	})

	t.Run("Empty strategies uses standard", func(t *testing.T) {
		combined := NewCombinedPricingStrategy([]PricingStrategy{}, true)

		ctx := context.Background()
		pricingCtx := PricingContext{
			Quantity:  decimal.NewFromInt(10),
			BasePrice: decimal.NewFromFloat(100.00),
			Currency:  "CNY",
		}

		result, err := combined.CalculatePrice(ctx, pricingCtx)
		require.NoError(t, err)

		assert.True(t, result.UnitPrice.Equal(decimal.NewFromFloat(100.00)))
	})

	t.Run("SupportsPromotion and SupportsTieredPricing", func(t *testing.T) {
		standardStrategy := NewStandardPricingStrategy()
		tieredStrategy := NewTieredPricingStrategy([]PriceTier{})

		combined1 := NewCombinedPricingStrategy([]PricingStrategy{standardStrategy}, true)
		assert.False(t, combined1.SupportsPromotion())
		assert.False(t, combined1.SupportsTieredPricing())

		combined2 := NewCombinedPricingStrategy([]PricingStrategy{tieredStrategy}, true)
		assert.False(t, combined2.SupportsPromotion())
		assert.True(t, combined2.SupportsTieredPricing())
	})
}

func TestPriceTierSorting(t *testing.T) {
	// Test that tiers are sorted correctly even when provided out of order
	unsortedTiers := []PriceTier{
		{MinQuantity: decimal.NewFromInt(100), UnitPrice: decimal.NewFromFloat(70.00)},
		{MinQuantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromFloat(100.00)},
		{MinQuantity: decimal.NewFromInt(50), UnitPrice: decimal.NewFromFloat(80.00)},
		{MinQuantity: decimal.NewFromInt(10), UnitPrice: decimal.NewFromFloat(90.00)},
	}

	strategy := NewTieredPricingStrategy(unsortedTiers)

	// Verify tiers are sorted
	for i := 1; i < len(strategy.Tiers); i++ {
		assert.True(t, strategy.Tiers[i-1].MinQuantity.LessThan(strategy.Tiers[i].MinQuantity),
			"Tiers should be sorted by MinQuantity ascending")
	}
}

func TestCustomerPricePriority(t *testing.T) {
	// Test that customer prices are applied by priority order
	customerPrices := []CustomerPrice{
		{CustomerID: "customer-1", UnitPrice: decimal.NewFromFloat(80.00), PriorityOrder: 3},
		{CustomerID: "customer-1", UnitPrice: decimal.NewFromFloat(90.00), PriorityOrder: 1}, // Higher priority (lower number)
		{CustomerID: "customer-1", UnitPrice: decimal.NewFromFloat(85.00), PriorityOrder: 2},
	}

	strategy := NewCustomerSpecificPricingStrategy(customerPrices, nil)

	ctx := context.Background()
	pricingCtx := PricingContext{
		CustomerID: "customer-1",
		ProductID:  "product-1",
		Quantity:   decimal.NewFromInt(5),
		BasePrice:  decimal.NewFromFloat(100.00),
		Currency:   "CNY",
	}

	result, err := strategy.CalculatePrice(ctx, pricingCtx)
	require.NoError(t, err)

	// Should use the price with priority 1 (90.00)
	assert.True(t, result.UnitPrice.Equal(decimal.NewFromFloat(90.00)))
}
