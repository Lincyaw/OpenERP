package valueobject

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUnit(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		unitName       string
		conversionRate decimal.Decimal
		wantErr        bool
		errContains    string
	}{
		{
			name:           "valid base unit",
			code:           "PCS",
			unitName:       "Pieces",
			conversionRate: decimal.NewFromInt(1),
			wantErr:        false,
		},
		{
			name:           "valid non-base unit",
			code:           "BOX",
			unitName:       "Box",
			conversionRate: decimal.NewFromInt(24),
			wantErr:        false,
		},
		{
			name:           "valid decimal conversion rate",
			code:           "G",
			unitName:       "Gram",
			conversionRate: decimal.NewFromFloat(0.001),
			wantErr:        false,
		},
		{
			name:           "code normalized to uppercase",
			code:           "pcs",
			unitName:       "Pieces",
			conversionRate: decimal.NewFromInt(1),
			wantErr:        false,
		},
		{
			name:           "code with whitespace trimmed",
			code:           "  PCS  ",
			unitName:       "Pieces",
			conversionRate: decimal.NewFromInt(1),
			wantErr:        false,
		},
		{
			name:           "empty code",
			code:           "",
			unitName:       "Pieces",
			conversionRate: decimal.NewFromInt(1),
			wantErr:        true,
			errContains:    "code cannot be empty",
		},
		{
			name:           "empty code after trim",
			code:           "   ",
			unitName:       "Pieces",
			conversionRate: decimal.NewFromInt(1),
			wantErr:        true,
			errContains:    "code cannot be empty",
		},
		{
			name:           "code too long",
			code:           "ABCDEFGHIJKLMNOPQRSTU", // 21 chars
			unitName:       "Pieces",
			conversionRate: decimal.NewFromInt(1),
			wantErr:        true,
			errContains:    "code cannot exceed 20 characters",
		},
		{
			name:           "empty name",
			code:           "PCS",
			unitName:       "",
			conversionRate: decimal.NewFromInt(1),
			wantErr:        true,
			errContains:    "name cannot be empty",
		},
		{
			name:           "name too long",
			code:           "PCS",
			unitName:       "ABCDEFGHIJKLMNOPQRSTUVWXYZABCDEFGHIJKLMNOPQRSTUVWXYZ", // 52 chars
			conversionRate: decimal.NewFromInt(1),
			wantErr:        true,
			errContains:    "name cannot exceed 50 characters",
		},
		{
			name:           "zero conversion rate",
			code:           "PCS",
			unitName:       "Pieces",
			conversionRate: decimal.Zero,
			wantErr:        true,
			errContains:    "conversion rate cannot be zero",
		},
		{
			name:           "negative conversion rate",
			code:           "PCS",
			unitName:       "Pieces",
			conversionRate: decimal.NewFromInt(-1),
			wantErr:        true,
			errContains:    "conversion rate cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unit, err := NewUnit(tt.code, tt.unitName, tt.conversionRate)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}
			require.NoError(t, err)
			// Verify code is normalized (uppercase, trimmed)
			assert.Equal(t, normalizeCode(tt.code), unit.Code())
			assert.Equal(t, tt.unitName, unit.Name())
			assert.True(t, unit.ConversionRate().Equal(tt.conversionRate))
		})
	}
}

func TestNewBaseUnit(t *testing.T) {
	unit, err := NewBaseUnit("PCS", "Pieces")
	require.NoError(t, err)
	assert.Equal(t, "PCS", unit.Code())
	assert.Equal(t, "Pieces", unit.Name())
	assert.True(t, unit.ConversionRate().Equal(decimal.NewFromInt(1)))
	assert.True(t, unit.IsBaseUnit())
}

func TestNewUnitFromFloat(t *testing.T) {
	unit, err := NewUnitFromFloat("BOX", "Box", 24.0)
	require.NoError(t, err)
	assert.Equal(t, "BOX", unit.Code())
	assert.True(t, unit.ConversionRate().Equal(decimal.NewFromFloat(24.0)))
}

func TestNewUnitFromInt(t *testing.T) {
	unit, err := NewUnitFromInt("BOX", "Box", 24)
	require.NoError(t, err)
	assert.Equal(t, "BOX", unit.Code())
	assert.True(t, unit.ConversionRate().Equal(decimal.NewFromInt(24)))
}

func TestMustNewUnit(t *testing.T) {
	// Should not panic with valid inputs
	unit := MustNewUnit("PCS", "Pieces", decimal.NewFromInt(1))
	assert.Equal(t, "PCS", unit.Code())

	// Should panic with invalid inputs
	assert.Panics(t, func() {
		MustNewUnit("", "Pieces", decimal.NewFromInt(1))
	})
}

func TestMustNewBaseUnit(t *testing.T) {
	// Should not panic with valid inputs
	unit := MustNewBaseUnit("PCS", "Pieces")
	assert.Equal(t, "PCS", unit.Code())
	assert.True(t, unit.IsBaseUnit())

	// Should panic with invalid inputs
	assert.Panics(t, func() {
		MustNewBaseUnit("", "Pieces")
	})
}

func TestUnit_IsBaseUnit(t *testing.T) {
	tests := []struct {
		name     string
		rate     decimal.Decimal
		expected bool
	}{
		{
			name:     "conversion rate 1 is base unit",
			rate:     decimal.NewFromInt(1),
			expected: true,
		},
		{
			name:     "conversion rate 24 is not base unit",
			rate:     decimal.NewFromInt(24),
			expected: false,
		},
		{
			name:     "conversion rate 0.001 is not base unit",
			rate:     decimal.NewFromFloat(0.001),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unit, _ := NewUnit("TEST", "Test", tt.rate)
			assert.Equal(t, tt.expected, unit.IsBaseUnit())
		})
	}
}

func TestUnit_IsZero(t *testing.T) {
	var zeroUnit Unit
	assert.True(t, zeroUnit.IsZero())

	unit := MustNewBaseUnit("PCS", "Pieces")
	assert.False(t, unit.IsZero())
}

func TestUnit_ConvertToBase(t *testing.T) {
	tests := []struct {
		name     string
		unit     Unit
		quantity decimal.Decimal
		expected decimal.Decimal
	}{
		{
			name:     "base unit conversion (rate=1)",
			unit:     MustNewBaseUnit("PCS", "Pieces"),
			quantity: decimal.NewFromInt(10),
			expected: decimal.NewFromInt(10),
		},
		{
			name:     "box to pieces (1 box = 24 pcs)",
			unit:     MustNewUnit("BOX", "Box", decimal.NewFromInt(24)),
			quantity: decimal.NewFromInt(2),
			expected: decimal.NewFromInt(48),
		},
		{
			name:     "gram to kg (1 g = 0.001 kg)",
			unit:     MustNewUnit("G", "Gram", decimal.NewFromFloat(0.001)),
			quantity: decimal.NewFromInt(500),
			expected: decimal.NewFromFloat(0.5),
		},
		{
			name:     "decimal quantity",
			unit:     MustNewUnit("BOX", "Box", decimal.NewFromInt(24)),
			quantity: decimal.NewFromFloat(1.5),
			expected: decimal.NewFromInt(36),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.unit.ConvertToBase(tt.quantity)
			assert.True(t, result.Equal(tt.expected), "expected %s, got %s", tt.expected, result)
		})
	}
}

func TestUnit_ConvertFromBase(t *testing.T) {
	tests := []struct {
		name         string
		unit         Unit
		baseQuantity decimal.Decimal
		expected     decimal.Decimal
	}{
		{
			name:         "base unit conversion (rate=1)",
			unit:         MustNewBaseUnit("PCS", "Pieces"),
			baseQuantity: decimal.NewFromInt(10),
			expected:     decimal.NewFromInt(10),
		},
		{
			name:         "pieces to box (1 box = 24 pcs)",
			unit:         MustNewUnit("BOX", "Box", decimal.NewFromInt(24)),
			baseQuantity: decimal.NewFromInt(48),
			expected:     decimal.NewFromInt(2),
		},
		{
			name:         "kg to gram (1 g = 0.001 kg)",
			unit:         MustNewUnit("G", "Gram", decimal.NewFromFloat(0.001)),
			baseQuantity: decimal.NewFromFloat(0.5),
			expected:     decimal.NewFromInt(500),
		},
		{
			name:         "partial box",
			unit:         MustNewUnit("BOX", "Box", decimal.NewFromInt(24)),
			baseQuantity: decimal.NewFromInt(36),
			expected:     decimal.NewFromFloat(1.5),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.unit.ConvertFromBase(tt.baseQuantity)
			assert.True(t, result.Equal(tt.expected), "expected %s, got %s", tt.expected, result)
		})
	}
}

func TestUnit_ConvertTo(t *testing.T) {
	pcs := MustNewBaseUnit("PCS", "Pieces")
	box := MustNewUnit("BOX", "Box", decimal.NewFromInt(24))
	pack := MustNewUnit("PACK", "Pack", decimal.NewFromInt(6))

	tests := []struct {
		name     string
		fromUnit Unit
		toUnit   Unit
		quantity decimal.Decimal
		expected decimal.Decimal
		wantErr  bool
	}{
		{
			name:     "pcs to box",
			fromUnit: pcs,
			toUnit:   box,
			quantity: decimal.NewFromInt(48),
			expected: decimal.NewFromInt(2),
		},
		{
			name:     "box to pcs",
			fromUnit: box,
			toUnit:   pcs,
			quantity: decimal.NewFromInt(2),
			expected: decimal.NewFromInt(48),
		},
		{
			name:     "box to pack",
			fromUnit: box,
			toUnit:   pack,
			quantity: decimal.NewFromInt(1),
			expected: decimal.NewFromInt(4), // 24 pcs / 6 pcs = 4 packs
		},
		{
			name:     "pack to box",
			fromUnit: pack,
			toUnit:   box,
			quantity: decimal.NewFromInt(4),
			expected: decimal.NewFromInt(1), // 4 * 6 = 24 pcs = 1 box
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.fromUnit.ConvertTo(tt.quantity, tt.toUnit)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.True(t, result.Equal(tt.expected), "expected %s, got %s", tt.expected, result)
		})
	}
}

func TestUnit_WithName(t *testing.T) {
	original := MustNewBaseUnit("PCS", "Pieces")

	updated, err := original.WithName("Units")
	require.NoError(t, err)
	assert.Equal(t, "PCS", updated.Code())
	assert.Equal(t, "Units", updated.Name())
	// Original should be unchanged (immutable)
	assert.Equal(t, "Pieces", original.Name())

	// Invalid name
	_, err = original.WithName("")
	require.Error(t, err)
}

func TestUnit_WithConversionRate(t *testing.T) {
	original := MustNewBaseUnit("BOX", "Box")

	updated, err := original.WithConversionRate(decimal.NewFromInt(24))
	require.NoError(t, err)
	assert.True(t, updated.ConversionRate().Equal(decimal.NewFromInt(24)))
	// Original should be unchanged (immutable)
	assert.True(t, original.ConversionRate().Equal(decimal.NewFromInt(1)))

	// Invalid rate
	_, err = original.WithConversionRate(decimal.Zero)
	require.Error(t, err)
}

func TestUnit_Equals(t *testing.T) {
	unit1 := MustNewBaseUnit("PCS", "Pieces")
	unit2 := MustNewBaseUnit("PCS", "Units") // Same code, different name
	unit3 := MustNewBaseUnit("BOX", "Box")

	assert.True(t, unit1.Equals(unit2), "units with same code should be equal")
	assert.False(t, unit1.Equals(unit3), "units with different codes should not be equal")
}

func TestUnit_EqualsStrict(t *testing.T) {
	unit1 := MustNewBaseUnit("PCS", "Pieces")
	unit2 := MustNewBaseUnit("PCS", "Pieces")
	unit3 := MustNewBaseUnit("PCS", "Units") // Same code, different name

	assert.True(t, unit1.EqualsStrict(unit2), "identical units should be strictly equal")
	assert.False(t, unit1.EqualsStrict(unit3), "units with different names should not be strictly equal")
}

func TestUnit_MatchesCode(t *testing.T) {
	unit := MustNewBaseUnit("PCS", "Pieces")

	assert.True(t, unit.MatchesCode("PCS"))
	assert.True(t, unit.MatchesCode("pcs"))
	assert.True(t, unit.MatchesCode("  PCS  "))
	assert.False(t, unit.MatchesCode("BOX"))
}

func TestUnit_String(t *testing.T) {
	baseUnit := MustNewBaseUnit("PCS", "Pieces")
	assert.Equal(t, "PCS (Pieces)", baseUnit.String())

	boxUnit := MustNewUnit("BOX", "Box", decimal.NewFromInt(24))
	assert.Equal(t, "BOX (Box, rate: 24)", boxUnit.String())
}

func TestUnit_JSON(t *testing.T) {
	original := MustNewUnit("BOX", "Box", decimal.NewFromInt(24))

	// Marshal
	data, err := json.Marshal(original)
	require.NoError(t, err)

	// Verify JSON structure
	var jsonMap map[string]interface{}
	err = json.Unmarshal(data, &jsonMap)
	require.NoError(t, err)
	assert.Equal(t, "BOX", jsonMap["code"])
	assert.Equal(t, "Box", jsonMap["name"])
	assert.Equal(t, "24", jsonMap["conversionRate"])

	// Unmarshal
	var restored Unit
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)
	assert.True(t, original.EqualsStrict(restored))
}

func TestUnit_JSONUnmarshalError(t *testing.T) {
	var unit Unit

	// Invalid conversion rate
	err := json.Unmarshal([]byte(`{"code":"PCS","name":"Pieces","conversionRate":"invalid"}`), &unit)
	require.Error(t, err)

	// Missing required fields (invalid code)
	err = json.Unmarshal([]byte(`{"code":"","name":"Pieces","conversionRate":"1"}`), &unit)
	require.Error(t, err)
}

func TestUnit_DatabaseOperations(t *testing.T) {
	original := MustNewBaseUnit("PCS", "Pieces")

	// Value (for database write)
	value, err := original.Value()
	require.NoError(t, err)
	assert.Equal(t, "PCS", value)

	// Scan (from database read)
	var restored Unit
	err = restored.Scan("pcs")
	require.NoError(t, err)
	assert.Equal(t, "PCS", restored.Code())
	assert.True(t, restored.IsBaseUnit())

	// Scan nil
	err = restored.Scan(nil)
	require.NoError(t, err)
	assert.True(t, restored.IsZero())

	// Scan []byte
	err = restored.Scan([]byte("BOX"))
	require.NoError(t, err)
	assert.Equal(t, "BOX", restored.Code())
}

func TestUnitDTO(t *testing.T) {
	dto := UnitDTO{
		Code:           "BOX",
		Name:           "Box",
		ConversionRate: decimal.NewFromInt(24),
	}

	// Convert to Unit
	unit, err := dto.ToUnit()
	require.NoError(t, err)
	assert.Equal(t, "BOX", unit.Code())
	assert.Equal(t, "Box", unit.Name())
	assert.True(t, unit.ConversionRate().Equal(decimal.NewFromInt(24)))

	// Convert back to DTO
	backToDTO := unit.ToDTO()
	assert.Equal(t, dto.Code, backToDTO.Code)
	assert.Equal(t, dto.Name, backToDTO.Name)
	assert.True(t, dto.ConversionRate.Equal(backToDTO.ConversionRate))
}

func TestPredefinedUnits(t *testing.T) {
	t.Run("PCSUnit", func(t *testing.T) {
		unit := PCSUnit()
		assert.Equal(t, "PCS", unit.Code())
		assert.True(t, unit.IsBaseUnit())
	})

	t.Run("BoxUnit", func(t *testing.T) {
		unit := BoxUnit(24)
		assert.Equal(t, "BOX", unit.Code())
		assert.True(t, unit.ConversionRate().Equal(decimal.NewFromInt(24)))
	})

	t.Run("KGUnit", func(t *testing.T) {
		unit := KGUnit()
		assert.Equal(t, "KG", unit.Code())
		assert.True(t, unit.IsBaseUnit())
	})

	t.Run("GramUnit", func(t *testing.T) {
		unit := GramUnit()
		assert.Equal(t, "G", unit.Code())
		// 1 g = 0.001 kg
		assert.True(t, unit.ConversionRate().Equal(decimal.NewFromFloat(0.001)))
	})

	t.Run("LiterUnit", func(t *testing.T) {
		unit := LiterUnit()
		assert.Equal(t, "L", unit.Code())
		assert.True(t, unit.IsBaseUnit())
	})

	t.Run("MLUnit", func(t *testing.T) {
		unit := MLUnit()
		assert.Equal(t, "ML", unit.Code())
		// 1 mL = 0.001 L
		assert.True(t, unit.ConversionRate().Equal(decimal.NewFromFloat(0.001)))
	})

	t.Run("MeterUnit", func(t *testing.T) {
		unit := MeterUnit()
		assert.Equal(t, "M", unit.Code())
		assert.True(t, unit.IsBaseUnit())
	})

	t.Run("CMUnit", func(t *testing.T) {
		unit := CMUnit()
		assert.Equal(t, "CM", unit.Code())
		// 1 cm = 0.01 m
		assert.True(t, unit.ConversionRate().Equal(decimal.NewFromFloat(0.01)))
	})
}

func TestUnit_ConversionScenarios(t *testing.T) {
	// Real-world scenario: Product sold by box and piece
	pcs := MustNewBaseUnit("PCS", "Pieces")
	box := MustNewUnit("BOX", "Box", decimal.NewFromInt(24))

	// Customer orders 2 boxes
	ordered := decimal.NewFromInt(2)
	baseQty := box.ConvertToBase(ordered)
	assert.True(t, baseQty.Equal(decimal.NewFromInt(48)), "2 boxes = 48 pieces")

	// Inventory has 60 pieces
	inventory := decimal.NewFromInt(60)
	boxesAvailable := box.ConvertFromBase(inventory)
	assert.True(t, boxesAvailable.Equal(decimal.NewFromFloat(2.5)), "60 pieces = 2.5 boxes")

	// Real-world scenario: Weight-based products
	kg := MustNewBaseUnit("KG", "Kilogram")
	g := MustNewUnit("G", "Gram", decimal.NewFromFloat(0.001))

	// Customer buys 500g
	grams := decimal.NewFromInt(500)
	kgQty := g.ConvertToBase(grams)
	assert.True(t, kgQty.Equal(decimal.NewFromFloat(0.5)), "500g = 0.5kg")

	// Verify unit is unchanged
	_ = pcs
	_ = kg
}

// Helper function to match NewUnit behavior
func normalizeCode(code string) string {
	// Match the implementation: trim and uppercase
	result := ""
	for _, r := range code {
		if r != ' ' && r != '\t' && r != '\n' {
			if r >= 'a' && r <= 'z' {
				r = r - 32 // Convert to uppercase
			}
			result += string(r)
		}
	}
	return result
}

func TestParseUnitFromJSON(t *testing.T) {
	t.Run("valid unit JSON", func(t *testing.T) {
		data := []byte(`{"code":"BOX","name":"Box","conversionRate":"24"}`)
		unit, err := ParseUnitFromJSON(data)
		require.NoError(t, err)
		assert.Equal(t, "BOX", unit.Code())
		assert.Equal(t, "Box", unit.Name())
		assert.True(t, unit.ConversionRate().Equal(decimal.NewFromInt(24)))
	})

	t.Run("valid base unit JSON (rate 1)", func(t *testing.T) {
		data := []byte(`{"code":"PCS","name":"Pieces","conversionRate":"1"}`)
		unit, err := ParseUnitFromJSON(data)
		require.NoError(t, err)
		assert.Equal(t, "PCS", unit.Code())
		assert.True(t, unit.IsBaseUnit())
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		data := []byte(`{invalid json}`)
		_, err := ParseUnitFromJSON(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse unit JSON")
	})

	t.Run("invalid conversion rate returns error", func(t *testing.T) {
		data := []byte(`{"code":"BOX","name":"Box","conversionRate":"not-a-number"}`)
		_, err := ParseUnitFromJSON(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid conversion rate")
	})

	t.Run("empty code returns error", func(t *testing.T) {
		data := []byte(`{"code":"","name":"Box","conversionRate":"24"}`)
		_, err := ParseUnitFromJSON(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unit code cannot be empty")
	})

	t.Run("zero conversion rate returns error", func(t *testing.T) {
		data := []byte(`{"code":"BOX","name":"Box","conversionRate":"0"}`)
		_, err := ParseUnitFromJSON(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be zero")
	})

	t.Run("immutability - returns new value", func(t *testing.T) {
		data := []byte(`{"code":"PACK","name":"Pack","conversionRate":"12"}`)
		unit1, err := ParseUnitFromJSON(data)
		require.NoError(t, err)
		unit2, err := ParseUnitFromJSON(data)
		require.NoError(t, err)

		// Both units should be equal but independent
		assert.True(t, unit1.EqualsStrict(unit2))
	})
}
