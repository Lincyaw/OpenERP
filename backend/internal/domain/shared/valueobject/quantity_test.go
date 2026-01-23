package valueobject

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewQuantity(t *testing.T) {
	t.Run("creates quantity with valid value and unit", func(t *testing.T) {
		q, err := NewQuantity(decimal.NewFromFloat(10.5), "kg")
		require.NoError(t, err)
		assert.Equal(t, "kg", q.Unit())
		assert.True(t, q.Amount().Equal(decimal.NewFromFloat(10.5)))
	})

	t.Run("returns error for negative quantity", func(t *testing.T) {
		_, err := NewQuantity(decimal.NewFromFloat(-5), "kg")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be negative")
	})

	t.Run("allows zero quantity", func(t *testing.T) {
		q, err := NewQuantity(decimal.Zero, "pcs")
		require.NoError(t, err)
		assert.True(t, q.IsZero())
	})

	t.Run("allows empty unit", func(t *testing.T) {
		q, err := NewQuantity(decimal.NewFromInt(5), "")
		require.NoError(t, err)
		assert.Equal(t, "", q.Unit())
	})
}

func TestNewQuantityFromFloat(t *testing.T) {
	q, err := NewQuantityFromFloat(25.5, "lbs")
	require.NoError(t, err)
	assert.Equal(t, 25.5, q.Float64())
	assert.Equal(t, "lbs", q.Unit())
}

func TestNewQuantityFromInt(t *testing.T) {
	q, err := NewQuantityFromInt(100, "pcs")
	require.NoError(t, err)
	assert.Equal(t, int64(100), q.IntValue())
	assert.Equal(t, "pcs", q.Unit())
}

func TestNewQuantityFromString(t *testing.T) {
	t.Run("valid string", func(t *testing.T) {
		q, err := NewQuantityFromString("50.25", "kg")
		require.NoError(t, err)
		assert.True(t, q.Amount().Equal(decimal.NewFromFloat(50.25)))
	})

	t.Run("invalid string", func(t *testing.T) {
		_, err := NewQuantityFromString("not-a-number", "kg")
		assert.Error(t, err)
	})
}

func TestNewIntegerQuantity(t *testing.T) {
	t.Run("creates integer quantity", func(t *testing.T) {
		q, err := NewIntegerQuantity(50, "units")
		require.NoError(t, err)
		assert.Equal(t, int64(50), q.IntValue())
	})

	t.Run("fails for negative", func(t *testing.T) {
		_, err := NewIntegerQuantity(-5, "units")
		assert.Error(t, err)
	})
}

func TestMustNewQuantity(t *testing.T) {
	t.Run("creates quantity", func(t *testing.T) {
		q := MustNewQuantity(decimal.NewFromInt(10), "pcs")
		assert.Equal(t, int64(10), q.IntValue())
	})

	t.Run("panics for negative", func(t *testing.T) {
		assert.Panics(t, func() {
			MustNewQuantity(decimal.NewFromInt(-1), "pcs")
		})
	})
}

func TestZeroQuantity(t *testing.T) {
	q := ZeroQuantity("kg")
	assert.True(t, q.IsZero())
	assert.Equal(t, "kg", q.Unit())
}

func TestQuantityIsPositiveZero(t *testing.T) {
	positive, _ := NewQuantityFromInt(10, "pcs")
	zero := ZeroQuantity("pcs")

	assert.True(t, positive.IsPositive())
	assert.False(t, positive.IsZero())

	assert.False(t, zero.IsPositive())
	assert.True(t, zero.IsZero())
}

func TestQuantityAdd(t *testing.T) {
	t.Run("adds same unit", func(t *testing.T) {
		q1, _ := NewQuantityFromInt(10, "pcs")
		q2, _ := NewQuantityFromInt(5, "pcs")
		result, err := q1.Add(q2)
		require.NoError(t, err)
		assert.Equal(t, int64(15), result.IntValue())
	})

	t.Run("fails for different units", func(t *testing.T) {
		q1, _ := NewQuantityFromInt(10, "kg")
		q2, _ := NewQuantityFromInt(5, "lbs")
		_, err := q1.Add(q2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "different units")
	})
}

func TestQuantityMustAdd(t *testing.T) {
	t.Run("adds same unit", func(t *testing.T) {
		q1, _ := NewQuantityFromInt(10, "pcs")
		q2, _ := NewQuantityFromInt(5, "pcs")
		result := q1.MustAdd(q2)
		assert.Equal(t, int64(15), result.IntValue())
	})

	t.Run("panics for different units", func(t *testing.T) {
		q1, _ := NewQuantityFromInt(10, "kg")
		q2, _ := NewQuantityFromInt(5, "lbs")
		assert.Panics(t, func() {
			q1.MustAdd(q2)
		})
	})
}

func TestQuantitySubtract(t *testing.T) {
	t.Run("subtracts same unit", func(t *testing.T) {
		q1, _ := NewQuantityFromInt(10, "pcs")
		q2, _ := NewQuantityFromInt(3, "pcs")
		result, err := q1.Subtract(q2)
		require.NoError(t, err)
		assert.Equal(t, int64(7), result.IntValue())
	})

	t.Run("fails for negative result", func(t *testing.T) {
		q1, _ := NewQuantityFromInt(5, "pcs")
		q2, _ := NewQuantityFromInt(10, "pcs")
		_, err := q1.Subtract(q2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "negative")
	})

	t.Run("fails for different units", func(t *testing.T) {
		q1, _ := NewQuantityFromInt(10, "kg")
		q2, _ := NewQuantityFromInt(5, "lbs")
		_, err := q1.Subtract(q2)
		assert.Error(t, err)
	})
}

func TestQuantitySubtractAllowNegative(t *testing.T) {
	q1, _ := NewQuantityFromInt(5, "pcs")
	q2, _ := NewQuantityFromInt(10, "pcs")
	result, err := q1.SubtractAllowNegative(q2)
	require.NoError(t, err)
	assert.Equal(t, int64(-5), result.IntValue())
}

func TestQuantityMultiply(t *testing.T) {
	q, _ := NewQuantityFromInt(10, "pcs")

	t.Run("multiply by positive", func(t *testing.T) {
		result, err := q.Multiply(decimal.NewFromInt(3))
		require.NoError(t, err)
		assert.Equal(t, int64(30), result.IntValue())
	})

	t.Run("fails for negative factor", func(t *testing.T) {
		_, err := q.Multiply(decimal.NewFromInt(-2))
		assert.Error(t, err)
	})

	t.Run("multiply by int", func(t *testing.T) {
		result, err := q.MultiplyByInt(5)
		require.NoError(t, err)
		assert.Equal(t, int64(50), result.IntValue())
	})

	t.Run("multiply by float", func(t *testing.T) {
		result, err := q.MultiplyByFloat(1.5)
		require.NoError(t, err)
		assert.Equal(t, 15.0, result.Float64())
	})
}

func TestQuantityDivide(t *testing.T) {
	q, _ := NewQuantityFromInt(100, "pcs")

	t.Run("divide by positive", func(t *testing.T) {
		result, err := q.Divide(decimal.NewFromInt(4))
		require.NoError(t, err)
		assert.Equal(t, 25.0, result.Float64())
	})

	t.Run("fails for zero divisor", func(t *testing.T) {
		_, err := q.Divide(decimal.Zero)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "divide by zero")
	})

	t.Run("fails for negative divisor", func(t *testing.T) {
		_, err := q.Divide(decimal.NewFromInt(-2))
		assert.Error(t, err)
	})
}

func TestQuantityConvert(t *testing.T) {
	q, _ := NewQuantityFromInt(1000, "g")

	t.Run("converts to different unit", func(t *testing.T) {
		// 1000g = 1kg (ratio 0.001)
		result, err := q.Convert("kg", decimal.NewFromFloat(0.001))
		require.NoError(t, err)
		assert.Equal(t, "kg", result.Unit())
		assert.Equal(t, 1.0, result.Float64())
	})

	t.Run("fails for zero ratio", func(t *testing.T) {
		_, err := q.Convert("kg", decimal.Zero)
		assert.Error(t, err)
	})

	t.Run("fails for negative ratio", func(t *testing.T) {
		_, err := q.Convert("kg", decimal.NewFromInt(-1))
		assert.Error(t, err)
	})
}

func TestQuantityRounding(t *testing.T) {
	q, _ := NewQuantityFromFloat(10.567, "kg")

	t.Run("round", func(t *testing.T) {
		result := q.Round(2)
		assert.Equal(t, "10.57", result.StringFixed(2))
	})

	t.Run("truncate", func(t *testing.T) {
		result := q.Truncate(2)
		assert.Equal(t, "10.56", result.StringFixed(2))
	})

	t.Run("ceiling", func(t *testing.T) {
		result := q.Ceiling()
		assert.Equal(t, int64(11), result.IntValue())
	})

	t.Run("floor", func(t *testing.T) {
		result := q.Floor()
		assert.Equal(t, int64(10), result.IntValue())
	})
}

func TestQuantityComparisons(t *testing.T) {
	q100, _ := NewQuantityFromInt(100, "pcs")
	q50, _ := NewQuantityFromInt(50, "pcs")
	q100b, _ := NewQuantityFromInt(100, "pcs")

	t.Run("equals", func(t *testing.T) {
		assert.True(t, q100.Equals(q100b))
		assert.False(t, q100.Equals(q50))
	})

	t.Run("less than", func(t *testing.T) {
		result, err := q50.LessThan(q100)
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("greater than", func(t *testing.T) {
		result, err := q100.GreaterThan(q50)
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("less than or equal", func(t *testing.T) {
		result, err := q100.LessThanOrEqual(q100b)
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("greater than or equal", func(t *testing.T) {
		result, err := q100.GreaterThanOrEqual(q50)
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("comparison fails for different units", func(t *testing.T) {
		kg, _ := NewQuantityFromInt(100, "kg")
		_, err := q100.LessThan(kg)
		assert.Error(t, err)
	})
}

func TestQuantityString(t *testing.T) {
	t.Run("with unit", func(t *testing.T) {
		q, _ := NewQuantityFromFloat(10.5, "kg")
		assert.Equal(t, "10.5 kg", q.String())
	})

	t.Run("without unit", func(t *testing.T) {
		q, _ := NewQuantityFromInt(10, "")
		assert.Equal(t, "10", q.String())
	})
}

func TestQuantityJSON(t *testing.T) {
	original, _ := NewQuantityFromFloat(25.5, "kg")

	t.Run("marshal", func(t *testing.T) {
		data, err := json.Marshal(original)
		require.NoError(t, err)
		assert.Contains(t, string(data), "25.5")
		assert.Contains(t, string(data), "kg")
	})

	t.Run("unmarshal valid", func(t *testing.T) {
		data := `{"value":"50.25","unit":"lbs"}`
		var q Quantity
		err := json.Unmarshal([]byte(data), &q)
		require.NoError(t, err)
		assert.Equal(t, "lbs", q.Unit())
		assert.True(t, q.Amount().Equal(decimal.NewFromFloat(50.25)))
	})

	t.Run("unmarshal negative fails", func(t *testing.T) {
		data := `{"value":"-10","unit":"pcs"}`
		var q Quantity
		err := json.Unmarshal([]byte(data), &q)
		assert.Error(t, err)
	})
}

func TestQuantitySufficientFor(t *testing.T) {
	available, _ := NewQuantityFromInt(100, "pcs")
	required, _ := NewQuantityFromInt(50, "pcs")
	exceeded, _ := NewQuantityFromInt(150, "pcs")

	t.Run("sufficient", func(t *testing.T) {
		result, err := available.SufficientFor(required)
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("not sufficient", func(t *testing.T) {
		result, err := available.SufficientFor(exceeded)
		require.NoError(t, err)
		assert.False(t, result)
	})
}

func TestQuantityDeficit(t *testing.T) {
	available, _ := NewQuantityFromInt(30, "pcs")
	required, _ := NewQuantityFromInt(50, "pcs")
	lesser, _ := NewQuantityFromInt(20, "pcs")

	t.Run("has deficit", func(t *testing.T) {
		deficit, err := available.Deficit(required)
		require.NoError(t, err)
		assert.Equal(t, int64(20), deficit.IntValue())
	})

	t.Run("no deficit", func(t *testing.T) {
		deficit, err := available.Deficit(lesser)
		require.NoError(t, err)
		assert.True(t, deficit.IsZero())
	})

	t.Run("fails for different units", func(t *testing.T) {
		kg, _ := NewQuantityFromInt(50, "kg")
		_, err := available.Deficit(kg)
		assert.Error(t, err)
	})
}

func TestQuantitySplit(t *testing.T) {
	q, _ := NewQuantityFromInt(100, "pcs")

	t.Run("splits evenly", func(t *testing.T) {
		parts, err := q.Split(4)
		require.NoError(t, err)
		assert.Len(t, parts, 4)
		for _, p := range parts {
			assert.Equal(t, 25.0, p.Float64())
		}
	})

	t.Run("single part returns original", func(t *testing.T) {
		parts, err := q.Split(1)
		require.NoError(t, err)
		assert.Len(t, parts, 1)
		assert.True(t, parts[0].Equals(q))
	})

	t.Run("fails for zero parts", func(t *testing.T) {
		_, err := q.Split(0)
		assert.Error(t, err)
	})
}

func TestQuantityScan(t *testing.T) {
	t.Run("scan string", func(t *testing.T) {
		var q Quantity
		err := q.Scan("50.25")
		require.NoError(t, err)
		assert.True(t, q.Amount().Equal(decimal.NewFromFloat(50.25)))
	})

	t.Run("scan bytes", func(t *testing.T) {
		var q Quantity
		err := q.Scan([]byte("100"))
		require.NoError(t, err)
		assert.Equal(t, int64(100), q.IntValue())
	})

	t.Run("scan nil", func(t *testing.T) {
		var q Quantity
		err := q.Scan(nil)
		require.NoError(t, err)
		assert.True(t, q.IsZero())
	})

	t.Run("scan invalid type", func(t *testing.T) {
		var q Quantity
		err := q.Scan(12345)
		assert.Error(t, err)
	})
}

func TestQuantityValue(t *testing.T) {
	q, _ := NewQuantityFromFloat(50.25, "kg")
	val, err := q.Value()
	require.NoError(t, err)
	assert.Equal(t, "50.25", val)
}
