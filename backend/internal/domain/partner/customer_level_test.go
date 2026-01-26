package partner

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCustomerLevel(t *testing.T) {
	t.Run("creates valid customer level", func(t *testing.T) {
		level, err := NewCustomerLevel("gold", "Gold Member", decimal.NewFromFloat(0.05))

		require.NoError(t, err)
		assert.Equal(t, "gold", level.Code())
		assert.Equal(t, "Gold Member", level.Name())
		assert.True(t, level.DiscountRate().Equal(decimal.NewFromFloat(0.05)))
	})

	t.Run("fails with empty code", func(t *testing.T) {
		_, err := NewCustomerLevel("", "Gold Member", decimal.NewFromFloat(0.05))

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "code cannot be empty")
	})

	t.Run("fails with empty name", func(t *testing.T) {
		_, err := NewCustomerLevel("gold", "", decimal.NewFromFloat(0.05))

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name cannot be empty")
	})

	t.Run("fails with negative discount rate", func(t *testing.T) {
		_, err := NewCustomerLevel("gold", "Gold Member", decimal.NewFromFloat(-0.05))

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "negative")
	})

	t.Run("fails with discount rate over 100%", func(t *testing.T) {
		_, err := NewCustomerLevel("gold", "Gold Member", decimal.NewFromFloat(1.5))

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceed")
	})
}

func TestMustNewCustomerLevel(t *testing.T) {
	t.Run("creates valid level without panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			level := MustNewCustomerLevel("gold", "Gold Member", decimal.NewFromFloat(0.05))
			assert.Equal(t, "gold", level.Code())
		})
	})

	t.Run("panics with invalid parameters", func(t *testing.T) {
		assert.Panics(t, func() {
			MustNewCustomerLevel("", "Gold Member", decimal.NewFromFloat(0.05))
		})
	})
}

func TestNewCustomerLevelFromCode(t *testing.T) {
	t.Run("creates level from code", func(t *testing.T) {
		level, err := NewCustomerLevelFromCode("gold")

		require.NoError(t, err)
		assert.Equal(t, "gold", level.Code())
		assert.Equal(t, "gold", level.Name()) // Name defaults to code
		assert.True(t, level.DiscountRate().IsZero())
	})

	t.Run("fails with empty code", func(t *testing.T) {
		_, err := NewCustomerLevelFromCode("")

		assert.Error(t, err)
	})
}

func TestPredefinedLevels(t *testing.T) {
	t.Run("NormalLevel has correct values", func(t *testing.T) {
		level := NormalLevel()

		assert.Equal(t, CustomerLevelCodeNormal, level.Code())
		assert.Equal(t, "普通会员", level.Name())
		assert.True(t, level.DiscountRate().IsZero())
		assert.False(t, level.HasDiscount())
	})

	t.Run("SilverLevel has correct values", func(t *testing.T) {
		level := SilverLevel()

		assert.Equal(t, CustomerLevelCodeSilver, level.Code())
		assert.True(t, level.DiscountRate().Equal(decimal.NewFromFloat(0.03)))
		assert.True(t, level.HasDiscount())
	})

	t.Run("GoldLevel has correct values", func(t *testing.T) {
		level := GoldLevel()

		assert.Equal(t, CustomerLevelCodeGold, level.Code())
		assert.True(t, level.DiscountRate().Equal(decimal.NewFromFloat(0.05)))
	})

	t.Run("PlatinumLevel has correct values", func(t *testing.T) {
		level := PlatinumLevel()

		assert.Equal(t, CustomerLevelCodePlatinum, level.Code())
		assert.True(t, level.DiscountRate().Equal(decimal.NewFromFloat(0.08)))
	})

	t.Run("VIPLevel has correct values", func(t *testing.T) {
		level := VIPLevel()

		assert.Equal(t, CustomerLevelCodeVIP, level.Code())
		assert.True(t, level.DiscountRate().Equal(decimal.NewFromFloat(0.10)))
	})

	t.Run("DefaultLevels returns all levels in order", func(t *testing.T) {
		levels := DefaultLevels()

		assert.Len(t, levels, 5)
		assert.Equal(t, CustomerLevelCodeNormal, levels[0].Code())
		assert.Equal(t, CustomerLevelCodeSilver, levels[1].Code())
		assert.Equal(t, CustomerLevelCodeGold, levels[2].Code())
		assert.Equal(t, CustomerLevelCodePlatinum, levels[3].Code())
		assert.Equal(t, CustomerLevelCodeVIP, levels[4].Code())
	})
}

func TestCustomerLevelDiscountPercent(t *testing.T) {
	t.Run("converts rate to percent", func(t *testing.T) {
		level := GoldLevel()

		percent := level.DiscountPercent()
		assert.True(t, percent.Equal(decimal.NewFromInt(5)))
	})
}

func TestCustomerLevelComparison(t *testing.T) {
	t.Run("Equals returns true for identical levels", func(t *testing.T) {
		level1, _ := NewCustomerLevel("gold", "Gold Member", decimal.NewFromFloat(0.05))
		level2, _ := NewCustomerLevel("gold", "Gold Member", decimal.NewFromFloat(0.05))

		assert.True(t, level1.Equals(level2))
	})

	t.Run("Equals returns false for different levels", func(t *testing.T) {
		level1 := GoldLevel()
		level2 := SilverLevel()

		assert.False(t, level1.Equals(level2))
	})

	t.Run("CodeEquals compares only codes", func(t *testing.T) {
		level1, _ := NewCustomerLevel("gold", "Gold Member", decimal.NewFromFloat(0.05))
		level2, _ := NewCustomerLevel("gold", "Different Name", decimal.NewFromFloat(0.10))

		assert.True(t, level1.CodeEquals(level2))
	})

	t.Run("IsHigherThan compares discount rates", func(t *testing.T) {
		vip := VIPLevel()
		gold := GoldLevel()
		normal := NormalLevel()

		assert.True(t, vip.IsHigherThan(gold))
		assert.True(t, gold.IsHigherThan(normal))
		assert.False(t, normal.IsHigherThan(vip))
	})

	t.Run("IsLowerThan compares discount rates", func(t *testing.T) {
		vip := VIPLevel()
		gold := GoldLevel()

		assert.True(t, gold.IsLowerThan(vip))
		assert.False(t, vip.IsLowerThan(gold))
	})
}

func TestCustomerLevelValidation(t *testing.T) {
	t.Run("IsValid returns true for valid level", func(t *testing.T) {
		level := GoldLevel()

		assert.True(t, level.IsValid())
	})

	t.Run("IsValid returns false for invalid level", func(t *testing.T) {
		level := CustomerLevel{} // Empty level

		assert.False(t, level.IsValid())
	})

	t.Run("IsEmpty returns true for empty level", func(t *testing.T) {
		level := CustomerLevel{}

		assert.True(t, level.IsEmpty())
	})

	t.Run("IsEmpty returns false for non-empty level", func(t *testing.T) {
		level := NormalLevel()

		assert.False(t, level.IsEmpty())
	})

	t.Run("IsStandardLevel identifies predefined levels", func(t *testing.T) {
		assert.True(t, NormalLevel().IsStandardLevel())
		assert.True(t, SilverLevel().IsStandardLevel())
		assert.True(t, GoldLevel().IsStandardLevel())
		assert.True(t, PlatinumLevel().IsStandardLevel())
		assert.True(t, VIPLevel().IsStandardLevel())

		custom, _ := NewCustomerLevel("custom", "Custom Level", decimal.NewFromFloat(0.15))
		assert.False(t, custom.IsStandardLevel())
	})
}

func TestCustomerLevelString(t *testing.T) {
	t.Run("formats level as string", func(t *testing.T) {
		level := GoldLevel()

		str := level.String()
		assert.Contains(t, str, "金卡会员")
		assert.Contains(t, str, "gold")
		assert.Contains(t, str, "5.0%")
	})
}

func TestCustomerLevelApplyDiscount(t *testing.T) {
	t.Run("applies discount to price", func(t *testing.T) {
		level := GoldLevel() // 5% discount
		price := decimal.NewFromFloat(100.00)

		discountedPrice := level.ApplyDiscount(price)
		expected := decimal.NewFromFloat(95.00)

		assert.True(t, discountedPrice.Equal(expected))
	})

	t.Run("no discount for normal level", func(t *testing.T) {
		level := NormalLevel()
		price := decimal.NewFromFloat(100.00)

		discountedPrice := level.ApplyDiscount(price)

		assert.True(t, discountedPrice.Equal(price))
	})
}

func TestCustomerLevelCalculateDiscountAmount(t *testing.T) {
	t.Run("calculates discount amount", func(t *testing.T) {
		level := VIPLevel() // 10% discount
		price := decimal.NewFromFloat(200.00)

		discountAmount := level.CalculateDiscountAmount(price)
		expected := decimal.NewFromFloat(20.00)

		assert.True(t, discountAmount.Equal(expected))
	})
}

func TestCustomerLevelJSONSerialization(t *testing.T) {
	t.Run("marshals to JSON", func(t *testing.T) {
		level := GoldLevel()

		data, err := json.Marshal(level)
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "gold", result["code"])
		assert.Equal(t, "金卡会员", result["name"])
		assert.Equal(t, "0.05", result["discount_rate"])
	})

	t.Run("unmarshals from JSON", func(t *testing.T) {
		jsonData := `{"code":"platinum","name":"白金会员","discount_rate":"0.08"}`

		var level CustomerLevel
		err := json.Unmarshal([]byte(jsonData), &level)
		require.NoError(t, err)

		assert.Equal(t, "platinum", level.Code())
		assert.Equal(t, "白金会员", level.Name())
		assert.True(t, level.DiscountRate().Equal(decimal.NewFromFloat(0.08)))
	})
}

func TestCustomerLevelDatabaseSerialization(t *testing.T) {
	t.Run("Value returns code string", func(t *testing.T) {
		level := GoldLevel()

		value, err := level.Value()
		require.NoError(t, err)
		assert.Equal(t, "gold", value)
	})

	t.Run("Value returns nil for empty level", func(t *testing.T) {
		level := CustomerLevel{}

		value, err := level.Value()
		require.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("Scan reads code from string", func(t *testing.T) {
		var level CustomerLevel
		err := level.Scan("silver")
		require.NoError(t, err)

		assert.Equal(t, "silver", level.Code())
	})

	t.Run("Scan reads code from bytes", func(t *testing.T) {
		var level CustomerLevel
		err := level.Scan([]byte("gold"))
		require.NoError(t, err)

		assert.Equal(t, "gold", level.Code())
	})

	t.Run("Scan handles nil value", func(t *testing.T) {
		var level CustomerLevel
		err := level.Scan(nil)
		require.NoError(t, err)

		assert.True(t, level.IsEmpty())
	})
}

func TestCustomerLevelWithDetails(t *testing.T) {
	t.Run("creates new level with updated details", func(t *testing.T) {
		// Simulate loading a level from DB (only code)
		level, _ := NewCustomerLevelFromCode("gold")

		// Enrich with full details from customer_levels table
		enriched := level.WithDetails("Premium Gold", decimal.NewFromFloat(0.07))

		assert.Equal(t, "gold", enriched.Code())
		assert.Equal(t, "Premium Gold", enriched.Name())
		assert.True(t, enriched.DiscountRate().Equal(decimal.NewFromFloat(0.07)))

		// Original should be unchanged (immutability)
		assert.Equal(t, "gold", level.Name())
	})
}

func TestCustomerLevelRecord(t *testing.T) {
	t.Run("ToCustomerLevel converts record to value object", func(t *testing.T) {
		record := &CustomerLevelRecord{
			ID:           uuid.New(),
			TenantID:     uuid.New(),
			Code:         "custom",
			Name:         "Custom Level",
			DiscountRate: decimal.NewFromFloat(0.12),
			SortOrder:    5,
			IsDefault:    false,
			IsActive:     true,
		}

		level := record.ToCustomerLevel()

		assert.Equal(t, "custom", level.Code())
		assert.Equal(t, "Custom Level", level.Name())
		assert.True(t, level.DiscountRate().Equal(decimal.NewFromFloat(0.12)))
	})

	t.Run("NewCustomerLevelRecord creates record from value object", func(t *testing.T) {
		tenantID := uuid.New()
		level := VIPLevel()

		record := NewCustomerLevelRecord(tenantID, level, 4, false)

		assert.Equal(t, tenantID, record.TenantID)
		assert.Equal(t, "vip", record.Code)
		assert.Equal(t, "VIP会员", record.Name)
		assert.True(t, record.DiscountRate.Equal(decimal.NewFromFloat(0.10)))
		assert.Equal(t, 4, record.SortOrder)
		assert.False(t, record.IsDefault)
		assert.True(t, record.IsActive)
	})

	t.Run("DefaultCustomerLevelRecords creates all standard levels", func(t *testing.T) {
		tenantID := uuid.New()

		records := DefaultCustomerLevelRecords(tenantID)

		assert.Len(t, records, 5)
		assert.True(t, records[0].IsDefault)  // Normal is default
		assert.False(t, records[1].IsDefault) // Others are not default
		assert.Equal(t, "normal", records[0].Code)
		assert.Equal(t, "vip", records[4].Code)
	})

	t.Run("TableName returns correct table name", func(t *testing.T) {
		record := CustomerLevelRecord{}
		assert.Equal(t, "customer_levels", record.TableName())
	})
}
