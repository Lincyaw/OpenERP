package valueobject

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

// Quantity is a value object representing quantities (inventory, order items, etc.)
// It supports decimal quantities for items sold by weight/volume
// It is immutable - all operations return new Quantity instances
type Quantity struct {
	value decimal.Decimal
	unit  string
}

// NewQuantity creates a new Quantity with the specified value and unit
func NewQuantity(value decimal.Decimal, unit string) (Quantity, error) {
	if value.IsNegative() {
		return Quantity{}, errors.New("quantity cannot be negative")
	}
	return Quantity{
		value: value,
		unit:  unit,
	}, nil
}

// NewQuantityFromFloat creates Quantity from a float64 value
func NewQuantityFromFloat(value float64, unit string) (Quantity, error) {
	return NewQuantity(decimal.NewFromFloat(value), unit)
}

// NewQuantityFromInt creates Quantity from an int64 value
func NewQuantityFromInt(value int64, unit string) (Quantity, error) {
	return NewQuantity(decimal.NewFromInt(value), unit)
}

// NewQuantityFromString creates Quantity from a string representation
func NewQuantityFromString(value string, unit string) (Quantity, error) {
	d, err := decimal.NewFromString(value)
	if err != nil {
		return Quantity{}, fmt.Errorf("invalid quantity string: %w", err)
	}
	return NewQuantity(d, unit)
}

// NewIntegerQuantity creates a Quantity for countable items (pieces, units, etc.)
func NewIntegerQuantity(value int64, unit string) (Quantity, error) {
	if value < 0 {
		return Quantity{}, errors.New("quantity cannot be negative")
	}
	return Quantity{
		value: decimal.NewFromInt(value),
		unit:  unit,
	}, nil
}

// MustNewQuantity creates a Quantity and panics on error
func MustNewQuantity(value decimal.Decimal, unit string) Quantity {
	q, err := NewQuantity(value, unit)
	if err != nil {
		panic(err)
	}
	return q
}

// MustNewQuantityFromInt creates a Quantity from int64 and panics on error
func MustNewQuantityFromInt(value int64, unit string) Quantity {
	q, err := NewQuantityFromInt(value, unit)
	if err != nil {
		panic(err)
	}
	return q
}

// Zero returns a zero quantity with the specified unit
func ZeroQuantity(unit string) Quantity {
	return Quantity{value: decimal.Zero, unit: unit}
}

// Amount returns the decimal value
func (q Quantity) Amount() decimal.Decimal {
	return q.value
}

// Unit returns the unit of measurement
func (q Quantity) Unit() string {
	return q.unit
}

// IsZero returns true if the quantity is zero
func (q Quantity) IsZero() bool {
	return q.value.IsZero()
}

// IsPositive returns true if the quantity is positive
func (q Quantity) IsPositive() bool {
	return q.value.IsPositive()
}

// IntValue returns the quantity as an int64 (truncated)
func (q Quantity) IntValue() int64 {
	return q.value.IntPart()
}

// Float64 returns the quantity as a float64 (may lose precision)
func (q Quantity) Float64() float64 {
	f, _ := q.value.Float64()
	return f
}

// Add returns a new Quantity with the sum of both quantities
// Returns error if units don't match
func (q Quantity) Add(other Quantity) (Quantity, error) {
	if q.unit != other.unit {
		return Quantity{}, fmt.Errorf("cannot add quantities with different units: %s and %s", q.unit, other.unit)
	}
	return Quantity{
		value: q.value.Add(other.value),
		unit:  q.unit,
	}, nil
}

// MustAdd adds two quantities, panics if units don't match
func (q Quantity) MustAdd(other Quantity) Quantity {
	result, err := q.Add(other)
	if err != nil {
		panic(err)
	}
	return result
}

// Subtract returns a new Quantity with the difference
// Returns error if units don't match or result would be negative
func (q Quantity) Subtract(other Quantity) (Quantity, error) {
	if q.unit != other.unit {
		return Quantity{}, fmt.Errorf("cannot subtract quantities with different units: %s and %s", q.unit, other.unit)
	}
	result := q.value.Sub(other.value)
	if result.IsNegative() {
		return Quantity{}, errors.New("resulting quantity would be negative")
	}
	return Quantity{
		value: result,
		unit:  q.unit,
	}, nil
}

// MustSubtract subtracts quantities, panics if error
func (q Quantity) MustSubtract(other Quantity) Quantity {
	result, err := q.Subtract(other)
	if err != nil {
		panic(err)
	}
	return result
}

// SubtractAllowNegative returns a new Quantity with the difference
// Allows negative results (useful for calculating deficits)
func (q Quantity) SubtractAllowNegative(other Quantity) (Quantity, error) {
	if q.unit != other.unit {
		return Quantity{}, fmt.Errorf("cannot subtract quantities with different units: %s and %s", q.unit, other.unit)
	}
	return Quantity{
		value: q.value.Sub(other.value),
		unit:  q.unit,
	}, nil
}

// Multiply returns a new Quantity multiplied by the given factor
func (q Quantity) Multiply(factor decimal.Decimal) (Quantity, error) {
	result := q.value.Mul(factor)
	if result.IsNegative() {
		return Quantity{}, errors.New("resulting quantity would be negative")
	}
	return Quantity{
		value: result,
		unit:  q.unit,
	}, nil
}

// MultiplyByInt returns a new Quantity multiplied by an integer
func (q Quantity) MultiplyByInt(factor int64) (Quantity, error) {
	return q.Multiply(decimal.NewFromInt(factor))
}

// MultiplyByFloat returns a new Quantity multiplied by a float
func (q Quantity) MultiplyByFloat(factor float64) (Quantity, error) {
	return q.Multiply(decimal.NewFromFloat(factor))
}

// Divide returns a new Quantity divided by the given divisor
func (q Quantity) Divide(divisor decimal.Decimal) (Quantity, error) {
	if divisor.IsZero() {
		return Quantity{}, errors.New("cannot divide by zero")
	}
	if divisor.IsNegative() {
		return Quantity{}, errors.New("cannot divide by negative number")
	}
	return Quantity{
		value: q.value.Div(divisor),
		unit:  q.unit,
	}, nil
}

// Convert converts the quantity to a different unit using the given conversion ratio
// newValue = oldValue * ratio
func (q Quantity) Convert(newUnit string, ratio decimal.Decimal) (Quantity, error) {
	if ratio.IsZero() || ratio.IsNegative() {
		return Quantity{}, errors.New("conversion ratio must be positive")
	}
	return Quantity{
		value: q.value.Mul(ratio),
		unit:  newUnit,
	}, nil
}

// ConvertByFloat converts using a float ratio
func (q Quantity) ConvertByFloat(newUnit string, ratio float64) (Quantity, error) {
	return q.Convert(newUnit, decimal.NewFromFloat(ratio))
}

// Round returns a new Quantity rounded to the specified decimal places
func (q Quantity) Round(places int32) Quantity {
	return Quantity{
		value: q.value.Round(places),
		unit:  q.unit,
	}
}

// Truncate returns a new Quantity truncated to the specified decimal places
func (q Quantity) Truncate(places int32) Quantity {
	return Quantity{
		value: q.value.Truncate(places),
		unit:  q.unit,
	}
}

// Ceiling returns a new Quantity rounded up to an integer
func (q Quantity) Ceiling() Quantity {
	return Quantity{
		value: q.value.Ceil(),
		unit:  q.unit,
	}
}

// Floor returns a new Quantity rounded down to an integer
func (q Quantity) Floor() Quantity {
	return Quantity{
		value: q.value.Floor(),
		unit:  q.unit,
	}
}

// Equals returns true if both quantities are equal (same value and unit)
func (q Quantity) Equals(other Quantity) bool {
	return q.unit == other.unit && q.value.Equal(other.value)
}

// LessThan returns true if this quantity is less than the other
func (q Quantity) LessThan(other Quantity) (bool, error) {
	if q.unit != other.unit {
		return false, fmt.Errorf("cannot compare quantities with different units: %s and %s", q.unit, other.unit)
	}
	return q.value.LessThan(other.value), nil
}

// LessThanOrEqual returns true if this quantity is less than or equal to the other
func (q Quantity) LessThanOrEqual(other Quantity) (bool, error) {
	if q.unit != other.unit {
		return false, fmt.Errorf("cannot compare quantities with different units: %s and %s", q.unit, other.unit)
	}
	return q.value.LessThanOrEqual(other.value), nil
}

// GreaterThan returns true if this quantity is greater than the other
func (q Quantity) GreaterThan(other Quantity) (bool, error) {
	if q.unit != other.unit {
		return false, fmt.Errorf("cannot compare quantities with different units: %s and %s", q.unit, other.unit)
	}
	return q.value.GreaterThan(other.value), nil
}

// GreaterThanOrEqual returns true if this quantity is greater than or equal to the other
func (q Quantity) GreaterThanOrEqual(other Quantity) (bool, error) {
	if q.unit != other.unit {
		return false, fmt.Errorf("cannot compare quantities with different units: %s and %s", q.unit, other.unit)
	}
	return q.value.GreaterThanOrEqual(other.value), nil
}

// String returns a string representation of the Quantity
func (q Quantity) String() string {
	if q.unit == "" {
		return q.value.String()
	}
	return fmt.Sprintf("%s %s", q.value.String(), q.unit)
}

// StringFixed returns the value as a string with fixed decimal places
func (q Quantity) StringFixed(places int32) string {
	return q.value.StringFixed(places)
}

// MarshalJSON implements json.Marshaler
func (q Quantity) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Value string `json:"value"`
		Unit  string `json:"unit"`
	}{
		Value: q.value.String(),
		Unit:  q.unit,
	})
}

// UnmarshalJSON implements json.Unmarshaler for deserialization purposes.
//
// IMPORTANT: This method exists ONLY to support JSON deserialization scenarios
// (e.g., API request binding, reading JSON from external sources).
// It is NOT intended for general Quantity creation from JSON data.
//
// For programmatic JSON parsing where you want explicit error handling and
// clearer intent, use ParseQuantityFromJSON instead.
//
// This method validates that quantity is non-negative during unmarshaling,
// maintaining the domain invariant that quantities cannot be negative.
func (q *Quantity) UnmarshalJSON(data []byte) error {
	var v struct {
		Value string `json:"value"`
		Unit  string `json:"unit"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	value, err := decimal.NewFromString(v.Value)
	if err != nil {
		return fmt.Errorf("invalid value: %w", err)
	}
	if value.IsNegative() {
		return errors.New("quantity cannot be negative")
	}
	q.value = value
	q.unit = v.Unit
	return nil
}

// ParseQuantityFromJSON creates a Quantity from JSON data with full validation.
// This is the recommended way to create Quantity from JSON when you want
// explicit control over error handling and clearer intent in your code.
//
// Unlike UnmarshalJSON (which is called implicitly by json.Unmarshal),
// this factory function:
// - Makes the parsing operation explicit
// - Returns a new Quantity value (not a pointer), maintaining immutability semantics
// - Uses the NewQuantity factory for consistent validation
//
// Example:
//
//	jsonData := []byte(`{"value":"10.5","unit":"KG"}`)
//	qty, err := valueobject.ParseQuantityFromJSON(jsonData)
//	if err != nil {
//	    // handle parsing error
//	}
func ParseQuantityFromJSON(data []byte) (Quantity, error) {
	var v struct {
		Value string `json:"value"`
		Unit  string `json:"unit"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return Quantity{}, fmt.Errorf("failed to parse quantity JSON: %w", err)
	}
	value, err := decimal.NewFromString(v.Value)
	if err != nil {
		return Quantity{}, fmt.Errorf("invalid value: %w", err)
	}
	return NewQuantity(value, v.Unit)
}

// Value implements driver.Valuer for database storage
func (q Quantity) Value() (driver.Value, error) {
	return q.value.String(), nil
}

// Scan implements sql.Scanner for database retrieval.
//
// IMPORTANT: This method exists ONLY to support GORM/database scanning scenarios.
// It is NOT intended for general Quantity creation from raw data.
//
// Note: This scans only the numeric value from database; unit is not restored
// from this operation. For complete Quantity data with unit, consider storing
// as JSON column or storing unit in a separate column.
func (q *Quantity) Scan(value any) error {
	if value == nil {
		q.value = decimal.Zero
		return nil
	}

	var strVal string
	switch v := value.(type) {
	case string:
		strVal = v
	case []byte:
		strVal = string(v)
	default:
		return fmt.Errorf("cannot scan %T into Quantity", value)
	}

	val, err := decimal.NewFromString(strVal)
	if err != nil {
		return fmt.Errorf("invalid decimal value: %w", err)
	}
	q.value = val
	return nil
}

// SufficientFor returns true if this quantity is sufficient for the required amount
func (q Quantity) SufficientFor(required Quantity) (bool, error) {
	return q.GreaterThanOrEqual(required)
}

// Deficit returns how much more is needed to meet the required amount
// Returns zero if quantity is sufficient
func (q Quantity) Deficit(required Quantity) (Quantity, error) {
	if q.unit != required.unit {
		return Quantity{}, fmt.Errorf("cannot calculate deficit with different units: %s and %s", q.unit, required.unit)
	}
	if q.value.GreaterThanOrEqual(required.value) {
		return ZeroQuantity(q.unit), nil
	}
	return Quantity{
		value: required.value.Sub(q.value),
		unit:  q.unit,
	}, nil
}

// Split divides the quantity into n equal parts
func (q Quantity) Split(parts int) ([]Quantity, error) {
	if parts <= 0 {
		return nil, errors.New("parts must be positive")
	}
	if parts == 1 {
		return []Quantity{q}, nil
	}

	partValue := q.value.Div(decimal.NewFromInt(int64(parts)))
	result := make([]Quantity, parts)
	for i := range parts {
		result[i] = Quantity{value: partValue, unit: q.unit}
	}
	return result, nil
}
