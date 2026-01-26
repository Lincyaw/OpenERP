package pricing

import (
	"context"
	"errors"
	"testing"

	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/google/uuid"
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
	// Create custom discounts using the legacy constructor
	customDiscounts := []CustomerLevelDiscount{
		{Level: "bronze", DiscountPercent: decimal.NewFromInt(2)},
		{Level: "silver", DiscountPercent: decimal.NewFromInt(5)},
		{Level: "gold", DiscountPercent: decimal.NewFromInt(10)},
		{Level: "diamond", DiscountPercent: decimal.NewFromInt(20)},
	}

	s := NewCustomerLevelPricingStrategyFromDiscounts(customDiscounts)
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

func TestCustomerLevelPricingStrategy_HasProvider(t *testing.T) {
	// Test without provider
	s1 := DefaultCustomerLevelPricingStrategy()
	assert.False(t, s1.HasProvider())

	// Test with provider
	mockProvider := &mockCustomerLevelProvider{}
	s2 := NewCustomerLevelPricingStrategy(mockProvider)
	assert.True(t, s2.HasProvider())

	// Test SetProvider
	s1.SetProvider(mockProvider)
	assert.True(t, s1.HasProvider())
}

// mockCustomerLevelProvider is a mock implementation of CustomerLevelProvider for testing
type mockCustomerLevelProvider struct {
	levels map[string]partner.CustomerLevel
	err    error
}

func newMockCustomerLevelProvider() *mockCustomerLevelProvider {
	return &mockCustomerLevelProvider{
		levels: make(map[string]partner.CustomerLevel),
	}
}

func (m *mockCustomerLevelProvider) addLevel(level partner.CustomerLevel) {
	m.levels[level.Code()] = level
}

func (m *mockCustomerLevelProvider) setError(err error) {
	m.err = err
}

func (m *mockCustomerLevelProvider) GetCustomerLevel(
	ctx context.Context,
	tenantID uuid.UUID,
	levelCode string,
) (partner.CustomerLevel, error) {
	if m.err != nil {
		return partner.CustomerLevel{}, m.err
	}
	if level, ok := m.levels[levelCode]; ok {
		return level, nil
	}
	return partner.CustomerLevel{}, errors.New("level not found")
}

func (m *mockCustomerLevelProvider) GetAllCustomerLevels(
	ctx context.Context,
	tenantID uuid.UUID,
) ([]partner.CustomerLevel, error) {
	if m.err != nil {
		return nil, m.err
	}
	levels := make([]partner.CustomerLevel, 0, len(m.levels))
	for _, level := range m.levels {
		levels = append(levels, level)
	}
	return levels, nil
}

func TestCustomerLevelPricingStrategy_WithProvider(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	// Create mock provider with custom CustomerLevel objects
	mockProvider := newMockCustomerLevelProvider()

	// Add custom customer levels using the domain value objects
	// CustomerLevel stores discount rate as decimal (0.15 for 15%)
	customLevel, err := partner.NewCustomerLevel("custom", "Custom Level", decimal.NewFromFloat(0.15))
	require.NoError(t, err)
	mockProvider.addLevel(customLevel)

	// Gold level with a different discount than default (12% instead of 5%)
	customGold, err := partner.NewCustomerLevel("gold", "Premium Gold", decimal.NewFromFloat(0.12))
	require.NoError(t, err)
	mockProvider.addLevel(customGold)

	s := NewCustomerLevelPricingStrategy(mockProvider)

	t.Run("uses provider discount for known level", func(t *testing.T) {
		pricingCtx := strategy.PricingContext{
			TenantID:     tenantID.String(),
			CustomerType: "custom",
			Quantity:     decimal.NewFromInt(1),
			BasePrice:    decimal.NewFromInt(100),
			Currency:     "CNY",
		}

		result, err := s.CalculatePrice(ctx, pricingCtx)
		require.NoError(t, err)

		// 15% discount on 100 = 85
		assert.True(t, decimal.NewFromInt(15).Equal(result.DiscountPercent),
			"expected 15%%, got %s%%", result.DiscountPercent)
		assert.True(t, decimal.NewFromInt(85).Equal(result.UnitPrice),
			"expected unit price 85, got %s", result.UnitPrice)
	})

	t.Run("overrides default discount with provider discount", func(t *testing.T) {
		pricingCtx := strategy.PricingContext{
			TenantID:     tenantID.String(),
			CustomerType: "gold", // Default is 5%, but provider returns 12%
			Quantity:     decimal.NewFromInt(1),
			BasePrice:    decimal.NewFromInt(100),
			Currency:     "CNY",
		}

		result, err := s.CalculatePrice(ctx, pricingCtx)
		require.NoError(t, err)

		// 12% discount on 100 = 88
		assert.True(t, decimal.NewFromInt(12).Equal(result.DiscountPercent),
			"expected 12%%, got %s%%", result.DiscountPercent)
		assert.True(t, decimal.NewFromInt(88).Equal(result.UnitPrice),
			"expected unit price 88, got %s", result.UnitPrice)
	})

	t.Run("falls back to default discount when provider returns error", func(t *testing.T) {
		// Request a level that doesn't exist in the provider
		pricingCtx := strategy.PricingContext{
			TenantID:     tenantID.String(),
			CustomerType: "silver", // Not in mock provider, will fallback to default 3%
			Quantity:     decimal.NewFromInt(1),
			BasePrice:    decimal.NewFromInt(100),
			Currency:     "CNY",
		}

		result, err := s.CalculatePrice(ctx, pricingCtx)
		require.NoError(t, err)

		// Falls back to default silver discount of 3%
		assert.True(t, decimal.NewFromInt(3).Equal(result.DiscountPercent),
			"expected 3%% (fallback), got %s%%", result.DiscountPercent)
		assert.True(t, decimal.NewFromInt(97).Equal(result.UnitPrice),
			"expected unit price 97, got %s", result.UnitPrice)
	})
}

func TestCustomerLevelPricingStrategy_WithProviderError(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	// Create mock provider that returns errors
	mockProvider := newMockCustomerLevelProvider()
	mockProvider.setError(errors.New("database connection error"))

	s := NewCustomerLevelPricingStrategy(mockProvider)

	t.Run("falls back to default discount on provider error", func(t *testing.T) {
		pricingCtx := strategy.PricingContext{
			TenantID:     tenantID.String(),
			CustomerType: "gold",
			Quantity:     decimal.NewFromInt(1),
			BasePrice:    decimal.NewFromInt(100),
			Currency:     "CNY",
		}

		result, err := s.CalculatePrice(ctx, pricingCtx)
		require.NoError(t, err)

		// Falls back to default gold discount of 5%
		assert.True(t, decimal.NewFromInt(5).Equal(result.DiscountPercent),
			"expected 5%% (fallback), got %s%%", result.DiscountPercent)
	})
}

func TestCustomerLevelPricingStrategy_InvalidTenantID(t *testing.T) {
	ctx := context.Background()

	mockProvider := newMockCustomerLevelProvider()
	s := NewCustomerLevelPricingStrategy(mockProvider)

	t.Run("falls back to default discount on invalid tenant ID", func(t *testing.T) {
		pricingCtx := strategy.PricingContext{
			TenantID:     "invalid-uuid", // Invalid UUID format
			CustomerType: "gold",
			Quantity:     decimal.NewFromInt(1),
			BasePrice:    decimal.NewFromInt(100),
			Currency:     "CNY",
		}

		result, err := s.CalculatePrice(ctx, pricingCtx)
		require.NoError(t, err)

		// Falls back to default gold discount of 5%
		assert.True(t, decimal.NewFromInt(5).Equal(result.DiscountPercent),
			"expected 5%% (fallback), got %s%%", result.DiscountPercent)
	})

	t.Run("falls back to default discount on empty tenant ID", func(t *testing.T) {
		pricingCtx := strategy.PricingContext{
			TenantID:     "", // Empty tenant ID
			CustomerType: "platinum",
			Quantity:     decimal.NewFromInt(1),
			BasePrice:    decimal.NewFromInt(100),
			Currency:     "CNY",
		}

		result, err := s.CalculatePrice(ctx, pricingCtx)
		require.NoError(t, err)

		// Falls back to default platinum discount of 8%
		assert.True(t, decimal.NewFromInt(8).Equal(result.DiscountPercent),
			"expected 8%% (fallback), got %s%%", result.DiscountPercent)
	})
}

func TestCustomerLevelPricingStrategy_GetDiscountForLevelWithContext(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	// Create mock provider with custom discount
	mockProvider := newMockCustomerLevelProvider()
	vipLevel, err := partner.NewCustomerLevel("vip", "Super VIP", decimal.NewFromFloat(0.25))
	require.NoError(t, err)
	mockProvider.addLevel(vipLevel)

	s := NewCustomerLevelPricingStrategy(mockProvider)

	t.Run("returns provider discount when available", func(t *testing.T) {
		discount := s.GetDiscountForLevelWithContext(ctx, tenantID, "vip")
		assert.True(t, decimal.NewFromInt(25).Equal(discount),
			"expected 25%%, got %s%%", discount)
	})

	t.Run("returns fallback discount when level not found", func(t *testing.T) {
		discount := s.GetDiscountForLevelWithContext(ctx, tenantID, "silver")
		assert.True(t, decimal.NewFromInt(3).Equal(discount),
			"expected 3%% (fallback), got %s%%", discount)
	})
}

func TestCustomerLevelPricingStrategy_WithFallback(t *testing.T) {
	ctx := context.Background()

	customFallbacks := map[string]decimal.Decimal{
		"basic":   decimal.Zero,
		"premium": decimal.NewFromInt(7),
		"elite":   decimal.NewFromInt(15),
	}

	s := NewCustomerLevelPricingStrategyWithFallback(nil, customFallbacks)

	t.Run("uses custom fallback discounts", func(t *testing.T) {
		pricingCtx := strategy.PricingContext{
			CustomerType: "premium",
			Quantity:     decimal.NewFromInt(1),
			BasePrice:    decimal.NewFromInt(100),
			Currency:     "CNY",
		}

		result, err := s.CalculatePrice(ctx, pricingCtx)
		require.NoError(t, err)

		assert.True(t, decimal.NewFromInt(7).Equal(result.DiscountPercent))
		assert.True(t, decimal.NewFromInt(93).Equal(result.UnitPrice))
	})

	t.Run("returns zero for unknown levels", func(t *testing.T) {
		pricingCtx := strategy.PricingContext{
			CustomerType: "gold", // Not in custom fallbacks
			Quantity:     decimal.NewFromInt(1),
			BasePrice:    decimal.NewFromInt(100),
			Currency:     "CNY",
		}

		result, err := s.CalculatePrice(ctx, pricingCtx)
		require.NoError(t, err)

		assert.True(t, decimal.Zero.Equal(result.DiscountPercent))
		assert.True(t, decimal.NewFromInt(100).Equal(result.UnitPrice))
	})
}
