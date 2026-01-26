package service

import (
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/shopspring/decimal"
)

// UnitConversionResult represents the result of a unit conversion
type UnitConversionResult struct {
	// The quantity in the source unit (what was input)
	SourceQuantity decimal.Decimal
	// The unit code of the source unit
	SourceUnitCode string
	// The conversion rate of the source unit (to base unit)
	ConversionRate decimal.Decimal
	// The quantity in base units (calculated: SourceQuantity * ConversionRate)
	BaseQuantity decimal.Decimal
	// The base unit code
	BaseUnitCode string
}

// UnitInfo contains information needed for unit conversion.
// Deprecated: Use valueobject.Unit directly for new code.
// This struct is maintained for backward compatibility.
type UnitInfo struct {
	UnitCode       string
	UnitName       string
	ConversionRate decimal.Decimal // 1 of this unit = ConversionRate base units
	IsBaseUnit     bool            // If true, ConversionRate should be 1
}

// ToUnit converts UnitInfo to Unit value object.
func (ui UnitInfo) ToUnit() (valueobject.Unit, error) {
	return valueobject.NewUnit(ui.UnitCode, ui.UnitName, ui.ConversionRate)
}

// FromUnit creates UnitInfo from Unit value object.
func FromUnit(u valueobject.Unit) UnitInfo {
	return UnitInfo{
		UnitCode:       u.Code(),
		UnitName:       u.Name(),
		ConversionRate: u.ConversionRate(),
		IsBaseUnit:     u.IsBaseUnit(),
	}
}

// UnitConversionService provides unit conversion operations
// This is a domain service as it operates across multiple aggregates
type UnitConversionService struct{}

// NewUnitConversionService creates a new unit conversion service
func NewUnitConversionService() *UnitConversionService {
	return &UnitConversionService{}
}

// ConvertToBaseUnit converts a quantity from a given unit to base units
// Parameters:
//   - quantity: the quantity in the source unit
//   - sourceUnit: information about the source unit
//   - baseUnitCode: the code of the base unit (for reference)
//
// Returns:
//   - UnitConversionResult containing both source and base quantities
//   - error if conversion fails
func (s *UnitConversionService) ConvertToBaseUnit(
	quantity decimal.Decimal,
	sourceUnit UnitInfo,
	baseUnitCode string,
) (*UnitConversionResult, error) {
	if quantity.IsNegative() {
		return nil, shared.NewDomainError("INVALID_QUANTITY", "Quantity cannot be negative")
	}
	if sourceUnit.ConversionRate.IsZero() {
		return nil, shared.NewDomainError("INVALID_CONVERSION_RATE", "Conversion rate cannot be zero")
	}
	if sourceUnit.ConversionRate.IsNegative() {
		return nil, shared.NewDomainError("INVALID_CONVERSION_RATE", "Conversion rate cannot be negative")
	}

	baseQuantity := quantity.Mul(sourceUnit.ConversionRate).Round(4)

	return &UnitConversionResult{
		SourceQuantity: quantity,
		SourceUnitCode: sourceUnit.UnitCode,
		ConversionRate: sourceUnit.ConversionRate,
		BaseQuantity:   baseQuantity,
		BaseUnitCode:   baseUnitCode,
	}, nil
}

// ConvertFromBaseUnit converts a quantity from base units to a target unit
// Parameters:
//   - baseQuantity: the quantity in base units
//   - targetUnit: information about the target unit
//   - baseUnitCode: the code of the base unit (for reference)
//
// Returns:
//   - UnitConversionResult containing both base and target quantities
//   - error if conversion fails
func (s *UnitConversionService) ConvertFromBaseUnit(
	baseQuantity decimal.Decimal,
	targetUnit UnitInfo,
	baseUnitCode string,
) (*UnitConversionResult, error) {
	if baseQuantity.IsNegative() {
		return nil, shared.NewDomainError("INVALID_QUANTITY", "Quantity cannot be negative")
	}
	if targetUnit.ConversionRate.IsZero() {
		return nil, shared.NewDomainError("INVALID_CONVERSION_RATE", "Conversion rate cannot be zero")
	}
	if targetUnit.ConversionRate.IsNegative() {
		return nil, shared.NewDomainError("INVALID_CONVERSION_RATE", "Conversion rate cannot be negative")
	}

	targetQuantity := baseQuantity.Div(targetUnit.ConversionRate).Round(4)

	return &UnitConversionResult{
		SourceQuantity: targetQuantity,
		SourceUnitCode: targetUnit.UnitCode,
		ConversionRate: targetUnit.ConversionRate,
		BaseQuantity:   baseQuantity,
		BaseUnitCode:   baseUnitCode,
	}, nil
}

// ConvertBetweenUnits converts a quantity between two units
// This goes through the base unit as an intermediary
// Parameters:
//   - quantity: the quantity in the source unit
//   - sourceUnit: information about the source unit
//   - targetUnit: information about the target unit
//   - baseUnitCode: the code of the base unit
//
// Returns:
//   - decimal.Decimal: the quantity in target units
//   - error if conversion fails
func (s *UnitConversionService) ConvertBetweenUnits(
	quantity decimal.Decimal,
	sourceUnit UnitInfo,
	targetUnit UnitInfo,
	baseUnitCode string,
) (decimal.Decimal, error) {
	// First convert to base unit
	baseResult, err := s.ConvertToBaseUnit(quantity, sourceUnit, baseUnitCode)
	if err != nil {
		return decimal.Zero, err
	}

	// Then convert from base unit to target unit
	if targetUnit.ConversionRate.IsZero() {
		return decimal.Zero, shared.NewDomainError("INVALID_CONVERSION_RATE", "Target conversion rate cannot be zero")
	}

	targetQuantity := baseResult.BaseQuantity.Div(targetUnit.ConversionRate).Round(4)
	return targetQuantity, nil
}

// CalculateUnitPrice calculates the unit price for a different unit based on conversion rate
// For example, if base unit price is 1 CNY per piece, and 1 box = 24 pieces,
// then box price should be 24 CNY per box
// Parameters:
//   - baseUnitPrice: price per base unit
//   - conversionRate: how many base units equal 1 of the target unit
//
// Returns:
//   - decimal.Decimal: price per target unit
func (s *UnitConversionService) CalculateUnitPrice(
	baseUnitPrice decimal.Decimal,
	conversionRate decimal.Decimal,
) decimal.Decimal {
	return baseUnitPrice.Mul(conversionRate).Round(4)
}

// CalculateBaseUnitPrice calculates the base unit price from a unit price
// For example, if box price is 24 CNY and 1 box = 24 pieces,
// then piece price should be 1 CNY
// Parameters:
//   - unitPrice: price per unit
//   - conversionRate: how many base units equal 1 of this unit
//
// Returns:
//   - decimal.Decimal: price per base unit
func (s *UnitConversionService) CalculateBaseUnitPrice(
	unitPrice decimal.Decimal,
	conversionRate decimal.Decimal,
) decimal.Decimal {
	if conversionRate.IsZero() {
		return decimal.Zero
	}
	return unitPrice.Div(conversionRate).Round(4)
}

// ValidateConversionRate validates a conversion rate
func (s *UnitConversionService) ValidateConversionRate(rate decimal.Decimal) error {
	if rate.IsZero() {
		return shared.NewDomainError("INVALID_CONVERSION_RATE", "Conversion rate cannot be zero")
	}
	if rate.IsNegative() {
		return shared.NewDomainError("INVALID_CONVERSION_RATE", "Conversion rate cannot be negative")
	}
	return nil
}

// CreateBaseUnitInfo creates a UnitInfo for a base unit (conversion rate = 1)
// Deprecated: Use valueobject.NewBaseUnit directly for new code.
func CreateBaseUnitInfo(unitCode, unitName string) UnitInfo {
	return UnitInfo{
		UnitCode:       unitCode,
		UnitName:       unitName,
		ConversionRate: decimal.NewFromInt(1),
		IsBaseUnit:     true,
	}
}

// CreateUnitInfo creates a UnitInfo for a non-base unit
// Deprecated: Use valueobject.NewUnit directly for new code.
func CreateUnitInfo(unitCode, unitName string, conversionRate decimal.Decimal) UnitInfo {
	return UnitInfo{
		UnitCode:       unitCode,
		UnitName:       unitName,
		ConversionRate: conversionRate,
		IsBaseUnit:     false,
	}
}

// ================================================================
// Methods using Unit value object (preferred for new code)
// ================================================================

// ConvertToBaseUnitVO converts a quantity from a given unit to base units using Unit value objects.
// Parameters:
//   - quantity: the quantity in the source unit
//   - sourceUnit: the source Unit value object
//   - baseUnit: the base Unit value object
//
// Returns:
//   - UnitConversionResult containing both source and base quantities
//   - error if conversion fails
func (s *UnitConversionService) ConvertToBaseUnitVO(
	quantity decimal.Decimal,
	sourceUnit valueobject.Unit,
	baseUnit valueobject.Unit,
) (*UnitConversionResult, error) {
	if quantity.IsNegative() {
		return nil, shared.NewDomainError("INVALID_QUANTITY", "Quantity cannot be negative")
	}

	baseQuantity := sourceUnit.ConvertToBase(quantity)

	return &UnitConversionResult{
		SourceQuantity: quantity,
		SourceUnitCode: sourceUnit.Code(),
		ConversionRate: sourceUnit.ConversionRate(),
		BaseQuantity:   baseQuantity,
		BaseUnitCode:   baseUnit.Code(),
	}, nil
}

// ConvertFromBaseUnitVO converts a quantity from base units to a target unit using Unit value objects.
// Parameters:
//   - baseQuantity: the quantity in base units
//   - targetUnit: the target Unit value object
//   - baseUnit: the base Unit value object
//
// Returns:
//   - UnitConversionResult containing both base and target quantities
//   - error if conversion fails
func (s *UnitConversionService) ConvertFromBaseUnitVO(
	baseQuantity decimal.Decimal,
	targetUnit valueobject.Unit,
	baseUnit valueobject.Unit,
) (*UnitConversionResult, error) {
	if baseQuantity.IsNegative() {
		return nil, shared.NewDomainError("INVALID_QUANTITY", "Quantity cannot be negative")
	}

	targetQuantity := targetUnit.ConvertFromBase(baseQuantity)

	return &UnitConversionResult{
		SourceQuantity: targetQuantity,
		SourceUnitCode: targetUnit.Code(),
		ConversionRate: targetUnit.ConversionRate(),
		BaseQuantity:   baseQuantity,
		BaseUnitCode:   baseUnit.Code(),
	}, nil
}

// ConvertBetweenUnitsVO converts a quantity between two units using Unit value objects.
// Goes through the base unit as an intermediary.
// Parameters:
//   - quantity: the quantity in the source unit
//   - sourceUnit: the source Unit value object
//   - targetUnit: the target Unit value object
//
// Returns:
//   - decimal.Decimal: the quantity in target units
//   - error if conversion fails
func (s *UnitConversionService) ConvertBetweenUnitsVO(
	quantity decimal.Decimal,
	sourceUnit valueobject.Unit,
	targetUnit valueobject.Unit,
) (decimal.Decimal, error) {
	if quantity.IsNegative() {
		return decimal.Zero, shared.NewDomainError("INVALID_QUANTITY", "Quantity cannot be negative")
	}

	return sourceUnit.ConvertTo(quantity, targetUnit)
}

// CalculateUnitPriceVO calculates the unit price for a different unit based on Unit value object.
// For example, if base unit price is 1 CNY per piece, and 1 box = 24 pieces,
// then box price should be 24 CNY per box.
// Parameters:
//   - baseUnitPrice: price per base unit
//   - targetUnit: the target unit
//
// Returns:
//   - decimal.Decimal: price per target unit
func (s *UnitConversionService) CalculateUnitPriceVO(
	baseUnitPrice decimal.Decimal,
	targetUnit valueobject.Unit,
) decimal.Decimal {
	return s.CalculateUnitPrice(baseUnitPrice, targetUnit.ConversionRate())
}

// CalculateBaseUnitPriceVO calculates the base unit price from a unit price using Unit value object.
// For example, if box price is 24 CNY and 1 box = 24 pieces,
// then piece price should be 1 CNY.
// Parameters:
//   - unitPrice: price per unit
//   - sourceUnit: the source unit
//
// Returns:
//   - decimal.Decimal: price per base unit
func (s *UnitConversionService) CalculateBaseUnitPriceVO(
	unitPrice decimal.Decimal,
	sourceUnit valueobject.Unit,
) decimal.Decimal {
	return s.CalculateBaseUnitPrice(unitPrice, sourceUnit.ConversionRate())
}
