package valueobject

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// Unit is a value object representing a unit of measurement.
// It is immutable - all operations return new Unit instances.
// A Unit has a code (identifier), name (display), and conversion rate to base unit.
type Unit struct {
	code           string
	name           string
	conversionRate decimal.Decimal
}

// Common unit codes for convenience
const (
	UnitCodePCS  = "PCS"  // Pieces (commonly used base unit)
	UnitCodeKG   = "KG"   // Kilograms
	UnitCodeG    = "G"    // Grams
	UnitCodeL    = "L"    // Liters
	UnitCodeML   = "ML"   // Milliliters
	UnitCodeM    = "M"    // Meters
	UnitCodeCM   = "CM"   // Centimeters
	UnitCodeBOX  = "BOX"  // Box
	UnitCodePACK = "PACK" // Pack
	UnitCodeSET  = "SET"  // Set
)

// NewUnit creates a new Unit with the specified code, name, and conversion rate.
// Parameters:
//   - code: unique identifier for the unit (e.g., "PCS", "BOX")
//   - name: human-readable name (e.g., "Pieces", "Box")
//   - conversionRate: how many base units equal 1 of this unit (must be positive)
//
// Returns error if:
//   - code is empty or too long (max 20 chars)
//   - name is empty or too long (max 50 chars)
//   - conversionRate is zero or negative
func NewUnit(code, name string, conversionRate decimal.Decimal) (Unit, error) {
	// Normalize code: trim and uppercase
	code = strings.TrimSpace(strings.ToUpper(code))
	name = strings.TrimSpace(name)

	if err := validateUnitCode(code); err != nil {
		return Unit{}, err
	}
	if err := validateUnitName(name); err != nil {
		return Unit{}, err
	}
	if err := validateUnitConversionRate(conversionRate); err != nil {
		return Unit{}, err
	}

	return Unit{
		code:           code,
		name:           name,
		conversionRate: conversionRate,
	}, nil
}

// NewBaseUnit creates a new Unit with conversion rate of 1 (base unit).
// Use this for creating the primary/base unit for a product.
func NewBaseUnit(code, name string) (Unit, error) {
	return NewUnit(code, name, decimal.NewFromInt(1))
}

// NewUnitFromFloat creates a Unit with conversion rate from float64.
func NewUnitFromFloat(code, name string, conversionRate float64) (Unit, error) {
	return NewUnit(code, name, decimal.NewFromFloat(conversionRate))
}

// NewUnitFromInt creates a Unit with conversion rate from int64.
func NewUnitFromInt(code, name string, conversionRate int64) (Unit, error) {
	return NewUnit(code, name, decimal.NewFromInt(conversionRate))
}

// MustNewUnit creates a Unit and panics on error.
// Use only when you're certain the inputs are valid.
func MustNewUnit(code, name string, conversionRate decimal.Decimal) Unit {
	u, err := NewUnit(code, name, conversionRate)
	if err != nil {
		panic(err)
	}
	return u
}

// MustNewBaseUnit creates a base Unit and panics on error.
func MustNewBaseUnit(code, name string) Unit {
	u, err := NewBaseUnit(code, name)
	if err != nil {
		panic(err)
	}
	return u
}

// Code returns the unit code (normalized to uppercase).
func (u Unit) Code() string {
	return u.code
}

// Name returns the unit name.
func (u Unit) Name() string {
	return u.name
}

// ConversionRate returns the conversion rate to base unit.
// 1 of this unit = ConversionRate base units.
func (u Unit) ConversionRate() decimal.Decimal {
	return u.conversionRate
}

// IsBaseUnit returns true if this is a base unit (conversion rate = 1).
func (u Unit) IsBaseUnit() bool {
	return u.conversionRate.Equal(decimal.NewFromInt(1))
}

// IsZero returns true if this is a zero-value Unit.
func (u Unit) IsZero() bool {
	return u.code == "" && u.name == "" && u.conversionRate.IsZero()
}

// ConvertToBase converts a quantity from this unit to base units.
// Formula: baseQuantity = quantity * conversionRate
func (u Unit) ConvertToBase(quantity decimal.Decimal) decimal.Decimal {
	return quantity.Mul(u.conversionRate).Round(4)
}

// ConvertFromBase converts a quantity from base units to this unit.
// Formula: unitQuantity = baseQuantity / conversionRate
func (u Unit) ConvertFromBase(baseQuantity decimal.Decimal) decimal.Decimal {
	if u.conversionRate.IsZero() {
		return decimal.Zero
	}
	return baseQuantity.Div(u.conversionRate).Round(4)
}

// ConvertTo converts a quantity from this unit to another unit.
// Goes through base unit as intermediary.
// Formula: targetQuantity = (quantity * thisRate) / targetRate
func (u Unit) ConvertTo(quantity decimal.Decimal, targetUnit Unit) (decimal.Decimal, error) {
	if targetUnit.conversionRate.IsZero() {
		return decimal.Zero, errors.New("target unit conversion rate cannot be zero")
	}
	baseQuantity := u.ConvertToBase(quantity)
	return baseQuantity.Div(targetUnit.conversionRate).Round(4), nil
}

// WithName returns a new Unit with an updated name.
func (u Unit) WithName(name string) (Unit, error) {
	name = strings.TrimSpace(name)
	if err := validateUnitName(name); err != nil {
		return Unit{}, err
	}
	return Unit{
		code:           u.code,
		name:           name,
		conversionRate: u.conversionRate,
	}, nil
}

// WithConversionRate returns a new Unit with an updated conversion rate.
func (u Unit) WithConversionRate(rate decimal.Decimal) (Unit, error) {
	if err := validateUnitConversionRate(rate); err != nil {
		return Unit{}, err
	}
	return Unit{
		code:           u.code,
		name:           u.name,
		conversionRate: rate,
	}, nil
}

// Equals returns true if both Units have the same code (case-insensitive).
// Name and conversion rate may differ for units with the same code.
func (u Unit) Equals(other Unit) bool {
	return u.code == other.code
}

// EqualsStrict returns true if both Units are exactly equal
// (same code, name, and conversion rate).
func (u Unit) EqualsStrict(other Unit) bool {
	return u.code == other.code &&
		u.name == other.name &&
		u.conversionRate.Equal(other.conversionRate)
}

// MatchesCode returns true if the unit code matches (case-insensitive).
func (u Unit) MatchesCode(code string) bool {
	return u.code == strings.TrimSpace(strings.ToUpper(code))
}

// String returns a string representation of the Unit.
func (u Unit) String() string {
	if u.IsBaseUnit() {
		return fmt.Sprintf("%s (%s)", u.code, u.name)
	}
	return fmt.Sprintf("%s (%s, rate: %s)", u.code, u.name, u.conversionRate.String())
}

// MarshalJSON implements json.Marshaler.
func (u Unit) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Code           string `json:"code"`
		Name           string `json:"name"`
		ConversionRate string `json:"conversionRate"`
	}{
		Code:           u.code,
		Name:           u.name,
		ConversionRate: u.conversionRate.String(),
	})
}

// UnmarshalJSON implements json.Unmarshaler.
func (u *Unit) UnmarshalJSON(data []byte) error {
	var v struct {
		Code           string `json:"code"`
		Name           string `json:"name"`
		ConversionRate string `json:"conversionRate"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	rate, err := decimal.NewFromString(v.ConversionRate)
	if err != nil {
		return fmt.Errorf("invalid conversion rate: %w", err)
	}

	parsed, err := NewUnit(v.Code, v.Name, rate)
	if err != nil {
		return err
	}

	*u = parsed
	return nil
}

// Value implements driver.Valuer for database storage.
// Stores the code only (name and rate should be stored separately if needed).
func (u Unit) Value() (driver.Value, error) {
	return u.code, nil
}

// Scan implements sql.Scanner for database retrieval.
// Only reads the code; sets default name and rate=1.
// Use ScanFull for complete unit data.
func (u *Unit) Scan(value any) error {
	if value == nil {
		u.code = ""
		u.name = ""
		u.conversionRate = decimal.Zero
		return nil
	}

	var strVal string
	switch v := value.(type) {
	case string:
		strVal = v
	case []byte:
		strVal = string(v)
	default:
		return fmt.Errorf("cannot scan %T into Unit", value)
	}

	u.code = strings.TrimSpace(strings.ToUpper(strVal))
	u.name = u.code // Default name to code
	u.conversionRate = decimal.NewFromInt(1)
	return nil
}

// UnitDTO is a data transfer object for Unit (for serialization/deserialization).
type UnitDTO struct {
	Code           string          `json:"code"`
	Name           string          `json:"name"`
	ConversionRate decimal.Decimal `json:"conversionRate"`
}

// ToUnit converts UnitDTO to Unit value object.
func (dto UnitDTO) ToUnit() (Unit, error) {
	return NewUnit(dto.Code, dto.Name, dto.ConversionRate)
}

// ToDTO converts Unit to UnitDTO.
func (u Unit) ToDTO() UnitDTO {
	return UnitDTO{
		Code:           u.code,
		Name:           u.name,
		ConversionRate: u.conversionRate,
	}
}

// Validation functions

func validateUnitCode(code string) error {
	if code == "" {
		return errors.New("unit code cannot be empty")
	}
	if len(code) > 20 {
		return errors.New("unit code cannot exceed 20 characters")
	}
	return nil
}

func validateUnitName(name string) error {
	if name == "" {
		return errors.New("unit name cannot be empty")
	}
	if len(name) > 50 {
		return errors.New("unit name cannot exceed 50 characters")
	}
	return nil
}

func validateUnitConversionRate(rate decimal.Decimal) error {
	if rate.IsNegative() {
		return errors.New("unit conversion rate cannot be negative")
	}
	if rate.IsZero() {
		return errors.New("unit conversion rate cannot be zero")
	}
	return nil
}

// Common predefined units

// PCSUnit returns a standard pieces unit (base unit).
func PCSUnit() Unit {
	return MustNewBaseUnit(UnitCodePCS, "Pieces")
}

// BoxUnit returns a standard box unit with the given conversion rate.
func BoxUnit(pcsPerBox int64) Unit {
	return MustNewUnit(UnitCodeBOX, "Box", decimal.NewFromInt(pcsPerBox))
}

// KGUnit returns a standard kilogram unit (base unit for weight).
func KGUnit() Unit {
	return MustNewBaseUnit(UnitCodeKG, "Kilogram")
}

// GramUnit returns a standard gram unit (1 kg = 1000 g).
func GramUnit() Unit {
	return MustNewUnit(UnitCodeG, "Gram", decimal.NewFromFloat(0.001))
}

// LiterUnit returns a standard liter unit (base unit for volume).
func LiterUnit() Unit {
	return MustNewBaseUnit(UnitCodeL, "Liter")
}

// MLUnit returns a standard milliliter unit (1 L = 1000 mL).
func MLUnit() Unit {
	return MustNewUnit(UnitCodeML, "Milliliter", decimal.NewFromFloat(0.001))
}

// MeterUnit returns a standard meter unit (base unit for length).
func MeterUnit() Unit {
	return MustNewBaseUnit(UnitCodeM, "Meter")
}

// CMUnit returns a standard centimeter unit (1 m = 100 cm).
func CMUnit() Unit {
	return MustNewUnit(UnitCodeCM, "Centimeter", decimal.NewFromFloat(0.01))
}
