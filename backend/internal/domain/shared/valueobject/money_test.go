package valueobject

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMoney(t *testing.T) {
	t.Run("creates money with valid amount and currency", func(t *testing.T) {
		m, err := NewMoney(decimal.NewFromFloat(100.50), CNY)
		require.NoError(t, err)
		assert.Equal(t, CNY, m.Currency())
		assert.True(t, m.Amount().Equal(decimal.NewFromFloat(100.50)))
	})

	t.Run("returns error for empty currency", func(t *testing.T) {
		_, err := NewMoney(decimal.NewFromFloat(100), "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "currency cannot be empty")
	})
}

func TestNewMoneyFromFloat(t *testing.T) {
	m, err := NewMoneyFromFloat(99.99, USD)
	require.NoError(t, err)
	assert.Equal(t, USD, m.Currency())
	assert.True(t, m.Amount().Equal(decimal.NewFromFloat(99.99)))
}

func TestNewMoneyFromInt(t *testing.T) {
	m, err := NewMoneyFromInt(1000, EUR)
	require.NoError(t, err)
	assert.Equal(t, EUR, m.Currency())
	assert.Equal(t, int64(1000), m.Amount().IntPart())
}

func TestNewMoneyFromString(t *testing.T) {
	t.Run("valid string", func(t *testing.T) {
		m, err := NewMoneyFromString("123.45", CNY)
		require.NoError(t, err)
		assert.True(t, m.Amount().Equal(decimal.NewFromFloat(123.45)))
	})

	t.Run("invalid string", func(t *testing.T) {
		_, err := NewMoneyFromString("not-a-number", CNY)
		assert.Error(t, err)
	})
}

func TestNewMoneyCNY(t *testing.T) {
	m := NewMoneyCNY(decimal.NewFromFloat(50.00))
	assert.Equal(t, CNY, m.Currency())
	assert.True(t, m.Amount().Equal(decimal.NewFromFloat(50.00)))
}

func TestNewMoneyCNYFromFloat(t *testing.T) {
	m := NewMoneyCNYFromFloat(75.50)
	assert.Equal(t, CNY, m.Currency())
	assert.Equal(t, 75.5, m.Float64())
}

func TestNewMoneyCNYFromString(t *testing.T) {
	m, err := NewMoneyCNYFromString("199.99")
	require.NoError(t, err)
	assert.Equal(t, CNY, m.Currency())
}

func TestZero(t *testing.T) {
	m := Zero(USD)
	assert.True(t, m.IsZero())
	assert.Equal(t, USD, m.Currency())
}

func TestZeroCNY(t *testing.T) {
	m := ZeroCNY()
	assert.True(t, m.IsZero())
	assert.Equal(t, CNY, m.Currency())
}

func TestMoneyIsPositiveNegativeZero(t *testing.T) {
	positive := NewMoneyCNYFromFloat(100)
	negative := NewMoneyCNYFromFloat(-100)
	zero := ZeroCNY()

	assert.True(t, positive.IsPositive())
	assert.False(t, positive.IsNegative())
	assert.False(t, positive.IsZero())

	assert.False(t, negative.IsPositive())
	assert.True(t, negative.IsNegative())
	assert.False(t, negative.IsZero())

	assert.False(t, zero.IsPositive())
	assert.False(t, zero.IsNegative())
	assert.True(t, zero.IsZero())
}

func TestMoneyAdd(t *testing.T) {
	t.Run("adds same currency", func(t *testing.T) {
		m1 := NewMoneyCNYFromFloat(100.50)
		m2 := NewMoneyCNYFromFloat(50.25)
		result, err := m1.Add(m2)
		require.NoError(t, err)
		assert.True(t, result.Amount().Equal(decimal.NewFromFloat(150.75)))
	})

	t.Run("fails for different currencies", func(t *testing.T) {
		m1, _ := NewMoneyFromFloat(100, CNY)
		m2, _ := NewMoneyFromFloat(50, USD)
		_, err := m1.Add(m2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "different currencies")
	})
}

func TestMoneyMustAdd(t *testing.T) {
	t.Run("adds same currency", func(t *testing.T) {
		m1 := NewMoneyCNYFromFloat(100)
		m2 := NewMoneyCNYFromFloat(50)
		result := m1.MustAdd(m2)
		assert.Equal(t, 150.0, result.Float64())
	})

	t.Run("panics for different currencies", func(t *testing.T) {
		m1, _ := NewMoneyFromFloat(100, CNY)
		m2, _ := NewMoneyFromFloat(50, USD)
		assert.Panics(t, func() {
			m1.MustAdd(m2)
		})
	})
}

func TestMoneySubtract(t *testing.T) {
	t.Run("subtracts same currency", func(t *testing.T) {
		m1 := NewMoneyCNYFromFloat(100.50)
		m2 := NewMoneyCNYFromFloat(50.25)
		result, err := m1.Subtract(m2)
		require.NoError(t, err)
		assert.True(t, result.Amount().Equal(decimal.NewFromFloat(50.25)))
	})

	t.Run("fails for different currencies", func(t *testing.T) {
		m1, _ := NewMoneyFromFloat(100, CNY)
		m2, _ := NewMoneyFromFloat(50, USD)
		_, err := m1.Subtract(m2)
		assert.Error(t, err)
	})
}

func TestMoneyMultiply(t *testing.T) {
	m := NewMoneyCNYFromFloat(100)

	t.Run("multiply by decimal", func(t *testing.T) {
		result := m.Multiply(decimal.NewFromFloat(1.5))
		assert.Equal(t, 150.0, result.Float64())
	})

	t.Run("multiply by int", func(t *testing.T) {
		result := m.MultiplyByInt(3)
		assert.Equal(t, 300.0, result.Float64())
	})

	t.Run("multiply by float", func(t *testing.T) {
		result := m.MultiplyByFloat(0.5)
		assert.Equal(t, 50.0, result.Float64())
	})
}

func TestMoneyDivide(t *testing.T) {
	m := NewMoneyCNYFromFloat(100)

	t.Run("divide by positive number", func(t *testing.T) {
		result, err := m.Divide(decimal.NewFromInt(4))
		require.NoError(t, err)
		assert.Equal(t, 25.0, result.Float64())
	})

	t.Run("fails for division by zero", func(t *testing.T) {
		_, err := m.Divide(decimal.Zero)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "divide by zero")
	})
}

func TestMoneyNegate(t *testing.T) {
	m := NewMoneyCNYFromFloat(100)
	result := m.Negate()
	assert.Equal(t, -100.0, result.Float64())
	assert.Equal(t, CNY, result.Currency())
}

func TestMoneyAbs(t *testing.T) {
	negative := NewMoneyCNYFromFloat(-100)
	result := negative.Abs()
	assert.Equal(t, 100.0, result.Float64())
}

func TestMoneyRound(t *testing.T) {
	m := NewMoneyCNYFromFloat(100.456)

	t.Run("round to 2 places", func(t *testing.T) {
		result := m.Round(2)
		assert.Equal(t, "100.46", result.StringFixed(2))
	})

	t.Run("truncate to 2 places", func(t *testing.T) {
		result := m.Truncate(2)
		assert.Equal(t, "100.45", result.StringFixed(2))
	})
}

func TestMoneyComparisons(t *testing.T) {
	m100 := NewMoneyCNYFromFloat(100)
	m50 := NewMoneyCNYFromFloat(50)
	m100b := NewMoneyCNYFromFloat(100)

	t.Run("equals", func(t *testing.T) {
		assert.True(t, m100.Equals(m100b))
		assert.False(t, m100.Equals(m50))
	})

	t.Run("less than", func(t *testing.T) {
		result, err := m50.LessThan(m100)
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("greater than", func(t *testing.T) {
		result, err := m100.GreaterThan(m50)
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("comparison fails for different currencies", func(t *testing.T) {
		usd, _ := NewMoneyFromFloat(100, USD)
		_, err := m100.LessThan(usd)
		assert.Error(t, err)
	})
}

func TestMoneyString(t *testing.T) {
	m := NewMoneyCNYFromFloat(123.45)
	assert.Equal(t, "123.45 CNY", m.String())
}

func TestMoneyJSON(t *testing.T) {
	original := NewMoneyCNYFromFloat(99.99)

	t.Run("marshal", func(t *testing.T) {
		data, err := json.Marshal(original)
		require.NoError(t, err)
		assert.Contains(t, string(data), "99.99")
		assert.Contains(t, string(data), "CNY")
	})

	t.Run("unmarshal", func(t *testing.T) {
		data := `{"amount":"123.45","currency":"USD"}`
		var m Money
		err := json.Unmarshal([]byte(data), &m)
		require.NoError(t, err)
		assert.Equal(t, USD, m.Currency())
		assert.True(t, m.Amount().Equal(decimal.NewFromFloat(123.45)))
	})
}

func TestMoneyAllocate(t *testing.T) {
	t.Run("allocates evenly", func(t *testing.T) {
		m := NewMoneyCNYFromFloat(100)
		parts, err := m.Allocate(4)
		require.NoError(t, err)
		assert.Len(t, parts, 4)
		// Sum should equal original
		sum := ZeroCNY()
		for _, p := range parts {
			sum = sum.MustAdd(p)
		}
		assert.True(t, sum.Amount().Equal(m.Amount()))
	})

	t.Run("handles remainder", func(t *testing.T) {
		m := NewMoneyCNYFromFloat(100)
		parts, err := m.Allocate(3)
		require.NoError(t, err)
		assert.Len(t, parts, 3)
	})

	t.Run("fails for zero parts", func(t *testing.T) {
		m := NewMoneyCNYFromFloat(100)
		_, err := m.Allocate(0)
		assert.Error(t, err)
	})

	t.Run("single part returns original", func(t *testing.T) {
		m := NewMoneyCNYFromFloat(100)
		parts, err := m.Allocate(1)
		require.NoError(t, err)
		assert.Len(t, parts, 1)
		assert.True(t, parts[0].Equals(m))
	})
}

func TestMoneyCalculatePercentage(t *testing.T) {
	m := NewMoneyCNYFromFloat(200)
	result := m.CalculatePercentage(decimal.NewFromInt(10))
	assert.Equal(t, 20.0, result.Float64())
}

func TestMoneyApplyDiscount(t *testing.T) {
	m := NewMoneyCNYFromFloat(100)
	result := m.ApplyDiscount(decimal.NewFromInt(20)) // 20% discount
	assert.Equal(t, 80.0, result.Float64())
}

func TestMoneyScan(t *testing.T) {
	t.Run("scan string", func(t *testing.T) {
		var m Money
		err := m.Scan("123.45")
		require.NoError(t, err)
		assert.True(t, m.Amount().Equal(decimal.NewFromFloat(123.45)))
		assert.Equal(t, DefaultCurrency, m.Currency())
	})

	t.Run("scan bytes", func(t *testing.T) {
		var m Money
		err := m.Scan([]byte("99.99"))
		require.NoError(t, err)
		assert.True(t, m.Amount().Equal(decimal.NewFromFloat(99.99)))
	})

	t.Run("scan nil", func(t *testing.T) {
		var m Money
		err := m.Scan(nil)
		require.NoError(t, err)
		assert.True(t, m.IsZero())
	})

	t.Run("scan invalid type", func(t *testing.T) {
		var m Money
		err := m.Scan(12345)
		assert.Error(t, err)
	})
}

func TestMoneyValue(t *testing.T) {
	m := NewMoneyCNYFromFloat(123.45)
	val, err := m.Value()
	require.NoError(t, err)
	assert.Equal(t, "123.45", val)
}

func TestParseMoneyFromJSON(t *testing.T) {
	t.Run("valid money JSON", func(t *testing.T) {
		data := []byte(`{"amount":"99.99","currency":"CNY"}`)
		money, err := ParseMoneyFromJSON(data)
		require.NoError(t, err)
		assert.True(t, money.Amount().Equal(decimal.NewFromFloat(99.99)))
		assert.Equal(t, CNY, money.Currency())
	})

	t.Run("valid money with USD currency", func(t *testing.T) {
		data := []byte(`{"amount":"150.00","currency":"USD"}`)
		money, err := ParseMoneyFromJSON(data)
		require.NoError(t, err)
		assert.True(t, money.Amount().Equal(decimal.NewFromFloat(150.00)))
		assert.Equal(t, USD, money.Currency())
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		data := []byte(`{invalid json}`)
		_, err := ParseMoneyFromJSON(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse money JSON")
	})

	t.Run("invalid amount string returns error", func(t *testing.T) {
		data := []byte(`{"amount":"not-a-number","currency":"CNY"}`)
		_, err := ParseMoneyFromJSON(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid amount")
	})

	t.Run("empty currency returns error", func(t *testing.T) {
		data := []byte(`{"amount":"100.00","currency":""}`)
		_, err := ParseMoneyFromJSON(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "currency cannot be empty")
	})

	t.Run("immutability - returns new value", func(t *testing.T) {
		data := []byte(`{"amount":"50.00","currency":"CNY"}`)
		money1, err := ParseMoneyFromJSON(data)
		require.NoError(t, err)
		money2, err := ParseMoneyFromJSON(data)
		require.NoError(t, err)

		// Both money values should be equal but independent
		assert.True(t, money1.Equals(money2))
	})
}
