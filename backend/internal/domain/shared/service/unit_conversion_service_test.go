package service

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnitConversionService_ConvertToBaseUnit(t *testing.T) {
	svc := NewUnitConversionService()

	tests := []struct {
		name         string
		quantity     decimal.Decimal
		sourceUnit   UnitInfo
		baseUnitCode string
		wantBase     decimal.Decimal
		wantErr      bool
	}{
		{
			name:     "convert boxes to pieces (1 box = 24 pieces)",
			quantity: decimal.NewFromInt(10),
			sourceUnit: UnitInfo{
				UnitCode:       "BOX",
				UnitName:       "Box",
				ConversionRate: decimal.NewFromInt(24),
				IsBaseUnit:     false,
			},
			baseUnitCode: "PCS",
			wantBase:     decimal.NewFromInt(240),
			wantErr:      false,
		},
		{
			name:     "convert base unit to itself (rate = 1)",
			quantity: decimal.NewFromInt(100),
			sourceUnit: UnitInfo{
				UnitCode:       "PCS",
				UnitName:       "Piece",
				ConversionRate: decimal.NewFromInt(1),
				IsBaseUnit:     true,
			},
			baseUnitCode: "PCS",
			wantBase:     decimal.NewFromInt(100),
			wantErr:      false,
		},
		{
			name:     "convert with decimal conversion rate (1 kg = 1000 g)",
			quantity: decimal.NewFromFloat(2.5),
			sourceUnit: UnitInfo{
				UnitCode:       "KG",
				UnitName:       "Kilogram",
				ConversionRate: decimal.NewFromInt(1000),
				IsBaseUnit:     false,
			},
			baseUnitCode: "G",
			wantBase:     decimal.NewFromInt(2500),
			wantErr:      false,
		},
		{
			name:     "convert with fractional rate (1 dozen = 12 pieces)",
			quantity: decimal.NewFromFloat(1.5),
			sourceUnit: UnitInfo{
				UnitCode:       "DZ",
				UnitName:       "Dozen",
				ConversionRate: decimal.NewFromInt(12),
				IsBaseUnit:     false,
			},
			baseUnitCode: "PCS",
			wantBase:     decimal.NewFromInt(18),
			wantErr:      false,
		},
		{
			name:     "error on zero conversion rate",
			quantity: decimal.NewFromInt(10),
			sourceUnit: UnitInfo{
				UnitCode:       "BOX",
				UnitName:       "Box",
				ConversionRate: decimal.Zero,
				IsBaseUnit:     false,
			},
			baseUnitCode: "PCS",
			wantBase:     decimal.Zero,
			wantErr:      true,
		},
		{
			name:     "error on negative quantity",
			quantity: decimal.NewFromInt(-10),
			sourceUnit: UnitInfo{
				UnitCode:       "BOX",
				UnitName:       "Box",
				ConversionRate: decimal.NewFromInt(24),
				IsBaseUnit:     false,
			},
			baseUnitCode: "PCS",
			wantBase:     decimal.Zero,
			wantErr:      true,
		},
		{
			name:     "error on negative conversion rate",
			quantity: decimal.NewFromInt(10),
			sourceUnit: UnitInfo{
				UnitCode:       "BOX",
				UnitName:       "Box",
				ConversionRate: decimal.NewFromInt(-24),
				IsBaseUnit:     false,
			},
			baseUnitCode: "PCS",
			wantBase:     decimal.Zero,
			wantErr:      true,
		},
		{
			name:     "zero quantity is valid",
			quantity: decimal.Zero,
			sourceUnit: UnitInfo{
				UnitCode:       "BOX",
				UnitName:       "Box",
				ConversionRate: decimal.NewFromInt(24),
				IsBaseUnit:     false,
			},
			baseUnitCode: "PCS",
			wantBase:     decimal.Zero,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.ConvertToBaseUnit(tt.quantity, tt.sourceUnit, tt.baseUnitCode)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.True(t, tt.wantBase.Equal(result.BaseQuantity),
				"expected base quantity %s, got %s", tt.wantBase, result.BaseQuantity)
			assert.True(t, tt.quantity.Equal(result.SourceQuantity))
			assert.Equal(t, tt.sourceUnit.UnitCode, result.SourceUnitCode)
			assert.Equal(t, tt.baseUnitCode, result.BaseUnitCode)
		})
	}
}

func TestUnitConversionService_ConvertFromBaseUnit(t *testing.T) {
	svc := NewUnitConversionService()

	tests := []struct {
		name         string
		baseQuantity decimal.Decimal
		targetUnit   UnitInfo
		baseUnitCode string
		wantTarget   decimal.Decimal
		wantErr      bool
	}{
		{
			name:         "convert pieces to boxes (24 pieces = 1 box)",
			baseQuantity: decimal.NewFromInt(240),
			targetUnit: UnitInfo{
				UnitCode:       "BOX",
				UnitName:       "Box",
				ConversionRate: decimal.NewFromInt(24),
				IsBaseUnit:     false,
			},
			baseUnitCode: "PCS",
			wantTarget:   decimal.NewFromInt(10),
			wantErr:      false,
		},
		{
			name:         "convert pieces to boxes with remainder",
			baseQuantity: decimal.NewFromInt(30),
			targetUnit: UnitInfo{
				UnitCode:       "BOX",
				UnitName:       "Box",
				ConversionRate: decimal.NewFromInt(24),
				IsBaseUnit:     false,
			},
			baseUnitCode: "PCS",
			wantTarget:   decimal.NewFromFloat(1.25), // 30/24 = 1.25
			wantErr:      false,
		},
		{
			name:         "convert grams to kilograms",
			baseQuantity: decimal.NewFromInt(2500),
			targetUnit: UnitInfo{
				UnitCode:       "KG",
				UnitName:       "Kilogram",
				ConversionRate: decimal.NewFromInt(1000),
				IsBaseUnit:     false,
			},
			baseUnitCode: "G",
			wantTarget:   decimal.NewFromFloat(2.5),
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.ConvertFromBaseUnit(tt.baseQuantity, tt.targetUnit, tt.baseUnitCode)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.True(t, tt.wantTarget.Equal(result.SourceQuantity),
				"expected target quantity %s, got %s", tt.wantTarget, result.SourceQuantity)
			assert.True(t, tt.baseQuantity.Equal(result.BaseQuantity))
		})
	}
}

func TestUnitConversionService_ConvertBetweenUnits(t *testing.T) {
	svc := NewUnitConversionService()

	tests := []struct {
		name         string
		quantity     decimal.Decimal
		sourceUnit   UnitInfo
		targetUnit   UnitInfo
		baseUnitCode string
		wantTarget   decimal.Decimal
		wantErr      bool
	}{
		{
			name:     "convert boxes to dozens (1 box = 24 pcs, 1 dozen = 12 pcs)",
			quantity: decimal.NewFromInt(1),
			sourceUnit: UnitInfo{
				UnitCode:       "BOX",
				UnitName:       "Box",
				ConversionRate: decimal.NewFromInt(24),
				IsBaseUnit:     false,
			},
			targetUnit: UnitInfo{
				UnitCode:       "DZ",
				UnitName:       "Dozen",
				ConversionRate: decimal.NewFromInt(12),
				IsBaseUnit:     false,
			},
			baseUnitCode: "PCS",
			wantTarget:   decimal.NewFromInt(2), // 1 box = 24 pcs = 2 dozens
			wantErr:      false,
		},
		{
			name:     "convert same unit to itself",
			quantity: decimal.NewFromInt(10),
			sourceUnit: UnitInfo{
				UnitCode:       "BOX",
				UnitName:       "Box",
				ConversionRate: decimal.NewFromInt(24),
				IsBaseUnit:     false,
			},
			targetUnit: UnitInfo{
				UnitCode:       "BOX",
				UnitName:       "Box",
				ConversionRate: decimal.NewFromInt(24),
				IsBaseUnit:     false,
			},
			baseUnitCode: "PCS",
			wantTarget:   decimal.NewFromInt(10),
			wantErr:      false,
		},
		{
			name:     "convert kilograms to pounds (approximate)",
			quantity: decimal.NewFromInt(1),
			sourceUnit: UnitInfo{
				UnitCode:       "KG",
				UnitName:       "Kilogram",
				ConversionRate: decimal.NewFromInt(1000), // 1 kg = 1000 g
				IsBaseUnit:     false,
			},
			targetUnit: UnitInfo{
				UnitCode:       "LB",
				UnitName:       "Pound",
				ConversionRate: decimal.NewFromFloat(453.592), // 1 lb = 453.592 g
				IsBaseUnit:     false,
			},
			baseUnitCode: "G",
			wantTarget:   decimal.NewFromFloat(2.2046).Round(4), // 1000/453.592 ≈ 2.2046
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.ConvertBetweenUnits(tt.quantity, tt.sourceUnit, tt.targetUnit, tt.baseUnitCode)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			// Use approximate comparison for floating point
			diff := tt.wantTarget.Sub(result).Abs()
			assert.True(t, diff.LessThan(decimal.NewFromFloat(0.001)),
				"expected target quantity %s, got %s", tt.wantTarget, result)
		})
	}
}

func TestUnitConversionService_CalculateUnitPrice(t *testing.T) {
	svc := NewUnitConversionService()

	tests := []struct {
		name           string
		baseUnitPrice  decimal.Decimal
		conversionRate decimal.Decimal
		wantPrice      decimal.Decimal
	}{
		{
			name:           "box price from piece price (24 pcs/box)",
			baseUnitPrice:  decimal.NewFromFloat(1.50), // 1.50 per piece
			conversionRate: decimal.NewFromInt(24),
			wantPrice:      decimal.NewFromFloat(36.00), // 24 * 1.50 = 36.00 per box
		},
		{
			name:           "kilogram price from gram price",
			baseUnitPrice:  decimal.NewFromFloat(0.01), // 0.01 per gram
			conversionRate: decimal.NewFromInt(1000),
			wantPrice:      decimal.NewFromFloat(10.00), // 1000 * 0.01 = 10.00 per kg
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.CalculateUnitPrice(tt.baseUnitPrice, tt.conversionRate)
			assert.True(t, tt.wantPrice.Equal(result),
				"expected price %s, got %s", tt.wantPrice, result)
		})
	}
}

func TestUnitConversionService_CalculateBaseUnitPrice(t *testing.T) {
	svc := NewUnitConversionService()

	tests := []struct {
		name           string
		unitPrice      decimal.Decimal
		conversionRate decimal.Decimal
		wantPrice      decimal.Decimal
	}{
		{
			name:           "piece price from box price (24 pcs/box)",
			unitPrice:      decimal.NewFromFloat(36.00), // 36.00 per box
			conversionRate: decimal.NewFromInt(24),
			wantPrice:      decimal.NewFromFloat(1.50), // 36.00 / 24 = 1.50 per piece
		},
		{
			name:           "gram price from kilogram price",
			unitPrice:      decimal.NewFromFloat(10.00), // 10.00 per kg
			conversionRate: decimal.NewFromInt(1000),
			wantPrice:      decimal.NewFromFloat(0.01), // 10.00 / 1000 = 0.01 per gram
		},
		{
			name:           "zero conversion rate returns zero",
			unitPrice:      decimal.NewFromFloat(10.00),
			conversionRate: decimal.Zero,
			wantPrice:      decimal.Zero,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.CalculateBaseUnitPrice(tt.unitPrice, tt.conversionRate)
			assert.True(t, tt.wantPrice.Equal(result),
				"expected price %s, got %s", tt.wantPrice, result)
		})
	}
}

func TestCreateUnitInfo(t *testing.T) {
	t.Run("create base unit info", func(t *testing.T) {
		info := CreateBaseUnitInfo("PCS", "Piece")
		assert.Equal(t, "PCS", info.UnitCode)
		assert.Equal(t, "Piece", info.UnitName)
		assert.True(t, decimal.NewFromInt(1).Equal(info.ConversionRate))
		assert.True(t, info.IsBaseUnit)
	})

	t.Run("create non-base unit info", func(t *testing.T) {
		rate := decimal.NewFromInt(24)
		info := CreateUnitInfo("BOX", "Box", rate)
		assert.Equal(t, "BOX", info.UnitCode)
		assert.Equal(t, "Box", info.UnitName)
		assert.True(t, rate.Equal(info.ConversionRate))
		assert.False(t, info.IsBaseUnit)
	})
}

func TestUnitConversionService_ValidateConversionRate(t *testing.T) {
	svc := NewUnitConversionService()

	tests := []struct {
		name    string
		rate    decimal.Decimal
		wantErr bool
	}{
		{"valid positive rate", decimal.NewFromInt(24), false},
		{"valid decimal rate", decimal.NewFromFloat(0.5), false},
		{"valid rate of 1", decimal.NewFromInt(1), false},
		{"zero rate", decimal.Zero, true},
		{"negative rate", decimal.NewFromInt(-1), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.ValidateConversionRate(tt.rate)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
