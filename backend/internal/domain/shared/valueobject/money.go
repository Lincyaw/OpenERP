package valueobject

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

// Currency represents a currency code (ISO 4217)
type Currency string

const (
	CNY Currency = "CNY" // Chinese Yuan (default)
	USD Currency = "USD" // US Dollar
	EUR Currency = "EUR" // Euro
	GBP Currency = "GBP" // British Pound
	JPY Currency = "JPY" // Japanese Yen
	HKD Currency = "HKD" // Hong Kong Dollar
)

// DefaultCurrency is the default currency for the system
const DefaultCurrency = CNY

// Money is a value object representing monetary amounts
// It is immutable - all operations return new Money instances
type Money struct {
	amount   decimal.Decimal
	currency Currency
}

// NewMoney creates a new Money with the specified amount and currency
func NewMoney(amount decimal.Decimal, currency Currency) (Money, error) {
	if currency == "" {
		return Money{}, errors.New("currency cannot be empty")
	}
	return Money{
		amount:   amount,
		currency: currency,
	}, nil
}

// NewMoneyFromFloat creates Money from a float64 value
func NewMoneyFromFloat(amount float64, currency Currency) (Money, error) {
	return NewMoney(decimal.NewFromFloat(amount), currency)
}

// NewMoneyFromInt creates Money from an int64 value (useful for cents)
func NewMoneyFromInt(amount int64, currency Currency) (Money, error) {
	return NewMoney(decimal.NewFromInt(amount), currency)
}

// NewMoneyFromString creates Money from a string representation
func NewMoneyFromString(amount string, currency Currency) (Money, error) {
	d, err := decimal.NewFromString(amount)
	if err != nil {
		return Money{}, fmt.Errorf("invalid amount string: %w", err)
	}
	return NewMoney(d, currency)
}

// NewMoneyCNY creates Money in CNY (Chinese Yuan)
func NewMoneyCNY(amount decimal.Decimal) Money {
	return Money{amount: amount, currency: CNY}
}

// NewMoneyCNYFromFloat creates Money in CNY from float64
func NewMoneyCNYFromFloat(amount float64) Money {
	return Money{amount: decimal.NewFromFloat(amount), currency: CNY}
}

// NewMoneyCNYFromString creates Money in CNY from string
func NewMoneyCNYFromString(amount string) (Money, error) {
	d, err := decimal.NewFromString(amount)
	if err != nil {
		return Money{}, fmt.Errorf("invalid amount string: %w", err)
	}
	return Money{amount: d, currency: CNY}, nil
}

// Zero returns a zero-value Money in the specified currency
func Zero(currency Currency) Money {
	return Money{amount: decimal.Zero, currency: currency}
}

// ZeroCNY returns a zero-value Money in CNY
func ZeroCNY() Money {
	return Zero(CNY)
}

// Amount returns the decimal amount
func (m Money) Amount() decimal.Decimal {
	return m.amount
}

// Currency returns the currency code
func (m Money) Currency() Currency {
	return m.currency
}

// IsZero returns true if the amount is zero
func (m Money) IsZero() bool {
	return m.amount.IsZero()
}

// IsPositive returns true if the amount is positive
func (m Money) IsPositive() bool {
	return m.amount.IsPositive()
}

// IsNegative returns true if the amount is negative
func (m Money) IsNegative() bool {
	return m.amount.IsNegative()
}

// Add returns a new Money with the sum of both amounts
// Returns error if currencies don't match
func (m Money) Add(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, fmt.Errorf("cannot add money with different currencies: %s and %s", m.currency, other.currency)
	}
	return Money{
		amount:   m.amount.Add(other.amount),
		currency: m.currency,
	}, nil
}

// MustAdd adds two Money values, panics if currencies don't match
func (m Money) MustAdd(other Money) Money {
	result, err := m.Add(other)
	if err != nil {
		panic(err)
	}
	return result
}

// Subtract returns a new Money with the difference
// Returns error if currencies don't match
func (m Money) Subtract(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, fmt.Errorf("cannot subtract money with different currencies: %s and %s", m.currency, other.currency)
	}
	return Money{
		amount:   m.amount.Sub(other.amount),
		currency: m.currency,
	}, nil
}

// MustSubtract subtracts two Money values, panics if currencies don't match
func (m Money) MustSubtract(other Money) Money {
	result, err := m.Subtract(other)
	if err != nil {
		panic(err)
	}
	return result
}

// Multiply returns a new Money multiplied by the given factor
func (m Money) Multiply(factor decimal.Decimal) Money {
	return Money{
		amount:   m.amount.Mul(factor),
		currency: m.currency,
	}
}

// MultiplyByInt returns a new Money multiplied by an integer
func (m Money) MultiplyByInt(factor int64) Money {
	return m.Multiply(decimal.NewFromInt(factor))
}

// MultiplyByFloat returns a new Money multiplied by a float
func (m Money) MultiplyByFloat(factor float64) Money {
	return m.Multiply(decimal.NewFromFloat(factor))
}

// Divide returns a new Money divided by the given divisor
// Returns error if divisor is zero
func (m Money) Divide(divisor decimal.Decimal) (Money, error) {
	if divisor.IsZero() {
		return Money{}, errors.New("cannot divide by zero")
	}
	return Money{
		amount:   m.amount.Div(divisor),
		currency: m.currency,
	}, nil
}

// Negate returns a new Money with the sign reversed
func (m Money) Negate() Money {
	return Money{
		amount:   m.amount.Neg(),
		currency: m.currency,
	}
}

// Abs returns a new Money with the absolute value
func (m Money) Abs() Money {
	return Money{
		amount:   m.amount.Abs(),
		currency: m.currency,
	}
}

// Round returns a new Money rounded to the specified decimal places
func (m Money) Round(places int32) Money {
	return Money{
		amount:   m.amount.Round(places),
		currency: m.currency,
	}
}

// RoundBank returns a new Money with banker's rounding to the specified places
func (m Money) RoundBank(places int32) Money {
	return Money{
		amount:   m.amount.RoundBank(places),
		currency: m.currency,
	}
}

// Truncate returns a new Money truncated to the specified decimal places
func (m Money) Truncate(places int32) Money {
	return Money{
		amount:   m.amount.Truncate(places),
		currency: m.currency,
	}
}

// Equals returns true if both Money values are equal (same amount and currency)
func (m Money) Equals(other Money) bool {
	return m.currency == other.currency && m.amount.Equal(other.amount)
}

// LessThan returns true if this Money is less than the other
// Returns error if currencies don't match
func (m Money) LessThan(other Money) (bool, error) {
	if m.currency != other.currency {
		return false, fmt.Errorf("cannot compare money with different currencies: %s and %s", m.currency, other.currency)
	}
	return m.amount.LessThan(other.amount), nil
}

// LessThanOrEqual returns true if this Money is less than or equal to the other
func (m Money) LessThanOrEqual(other Money) (bool, error) {
	if m.currency != other.currency {
		return false, fmt.Errorf("cannot compare money with different currencies: %s and %s", m.currency, other.currency)
	}
	return m.amount.LessThanOrEqual(other.amount), nil
}

// GreaterThan returns true if this Money is greater than the other
func (m Money) GreaterThan(other Money) (bool, error) {
	if m.currency != other.currency {
		return false, fmt.Errorf("cannot compare money with different currencies: %s and %s", m.currency, other.currency)
	}
	return m.amount.GreaterThan(other.amount), nil
}

// GreaterThanOrEqual returns true if this Money is greater than or equal to the other
func (m Money) GreaterThanOrEqual(other Money) (bool, error) {
	if m.currency != other.currency {
		return false, fmt.Errorf("cannot compare money with different currencies: %s and %s", m.currency, other.currency)
	}
	return m.amount.GreaterThanOrEqual(other.amount), nil
}

// String returns a string representation of the Money
func (m Money) String() string {
	return fmt.Sprintf("%s %s", m.amount.StringFixed(2), m.currency)
}

// StringFixed returns the amount as a string with fixed decimal places
func (m Money) StringFixed(places int32) string {
	return m.amount.StringFixed(places)
}

// Float64 returns the amount as a float64 (may lose precision)
func (m Money) Float64() float64 {
	f, _ := m.amount.Float64()
	return f
}

// MarshalJSON implements json.Marshaler
func (m Money) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Amount   string   `json:"amount"`
		Currency Currency `json:"currency"`
	}{
		Amount:   m.amount.String(),
		Currency: m.currency,
	})
}

// UnmarshalJSON implements json.Unmarshaler for deserialization purposes.
//
// IMPORTANT: This method exists ONLY to support JSON deserialization scenarios
// (e.g., API request binding, reading JSON from external sources).
// It is NOT intended for general Money creation from JSON data.
//
// For programmatic JSON parsing where you want explicit error handling and
// clearer intent, use ParseMoneyFromJSON instead.
//
// Note: This method directly assigns fields (amount, currency) without going
// through NewMoney factory, which bypasses validation. This is acceptable because:
// 1. Money allows any decimal amount (including negative for refunds/credits)
// 2. Currency is directly assigned from JSON (empty currency will cause issues later)
// For strict validation, use ParseMoneyFromJSON instead.
func (m *Money) UnmarshalJSON(data []byte) error {
	var v struct {
		Amount   string   `json:"amount"`
		Currency Currency `json:"currency"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	amount, err := decimal.NewFromString(v.Amount)
	if err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}
	m.amount = amount
	m.currency = v.Currency
	return nil
}

// ParseMoneyFromJSON creates a Money from JSON data with full validation.
// This is the recommended way to create Money from JSON when you want
// explicit control over error handling and stricter validation.
//
// Unlike UnmarshalJSON (which is called implicitly by json.Unmarshal),
// this factory function:
// - Makes the parsing operation explicit
// - Validates that currency is not empty
// - Returns a new Money value (not a pointer), maintaining immutability semantics
//
// Example:
//
//	jsonData := []byte(`{"amount":"99.99","currency":"CNY"}`)
//	money, err := valueobject.ParseMoneyFromJSON(jsonData)
//	if err != nil {
//	    // handle parsing error
//	}
func ParseMoneyFromJSON(data []byte) (Money, error) {
	var v struct {
		Amount   string   `json:"amount"`
		Currency Currency `json:"currency"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return Money{}, fmt.Errorf("failed to parse money JSON: %w", err)
	}
	amount, err := decimal.NewFromString(v.Amount)
	if err != nil {
		return Money{}, fmt.Errorf("invalid amount: %w", err)
	}
	return NewMoney(amount, v.Currency)
}

// Value implements driver.Valuer for database storage
// Stores as a numeric value (amount only)
func (m Money) Value() (driver.Value, error) {
	return m.amount.String(), nil
}

// Scan implements sql.Scanner for database retrieval.
//
// IMPORTANT: This method exists ONLY to support GORM/database scanning scenarios.
// It is NOT intended for general Money creation from raw data.
//
// Note: This scans only the amount from database; currency defaults to DefaultCurrency
// if not already set. For complete Money data, consider storing as JSON column
// and using UnmarshalJSON, or store currency in a separate column.
func (m *Money) Scan(value any) error {
	if value == nil {
		m.amount = decimal.Zero
		m.currency = DefaultCurrency
		return nil
	}

	var strVal string
	switch v := value.(type) {
	case string:
		strVal = v
	case []byte:
		strVal = string(v)
	default:
		return fmt.Errorf("cannot scan %T into Money", value)
	}

	amount, err := decimal.NewFromString(strVal)
	if err != nil {
		return fmt.Errorf("invalid decimal value: %w", err)
	}
	m.amount = amount
	if m.currency == "" {
		m.currency = DefaultCurrency
	}
	return nil
}

// Allocate divides money into n parts, handling remainders
// Returns a slice of Money values that sum to the original amount
func (m Money) Allocate(parts int) ([]Money, error) {
	if parts <= 0 {
		return nil, errors.New("parts must be positive")
	}
	if parts == 1 {
		return []Money{m}, nil
	}

	// Calculate base amount per part
	base := m.amount.Div(decimal.NewFromInt(int64(parts))).Truncate(2)
	remainder := m.amount.Sub(base.Mul(decimal.NewFromInt(int64(parts))))

	result := make([]Money, parts)
	remainderCents := remainder.Mul(decimal.NewFromInt(100)).IntPart()

	for i := range parts {
		partAmount := base
		if int64(i) < remainderCents {
			partAmount = partAmount.Add(decimal.NewFromFloat(0.01))
		}
		result[i] = Money{amount: partAmount, currency: m.currency}
	}

	return result, nil
}

// CalculatePercentage returns the percentage of this Money
func (m Money) CalculatePercentage(percent decimal.Decimal) Money {
	return Money{
		amount:   m.amount.Mul(percent).Div(decimal.NewFromInt(100)),
		currency: m.currency,
	}
}

// ApplyDiscount returns the Money after applying a percentage discount
func (m Money) ApplyDiscount(discountPercent decimal.Decimal) Money {
	discount := m.CalculatePercentage(discountPercent)
	return Money{
		amount:   m.amount.Sub(discount.amount),
		currency: m.currency,
	}
}
