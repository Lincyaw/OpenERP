package catalog

import (
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ProductUnit represents an alternate unit for a product with conversion rate
// It defines how different units relate to the base unit (e.g., 1 box = 24 pcs)
type ProductUnit struct {
	ID                    uuid.UUID       // Primary key
	TenantID              uuid.UUID       // Tenant ID reference
	ProductID             uuid.UUID       // Product ID reference
	UnitCode              string          // Unit code (e.g., "box", "case")
	UnitName              string          // Unit display name
	ConversionRate        decimal.Decimal // Rate to convert to base unit
	DefaultPurchasePrice  decimal.Decimal // Default purchase price for this unit
	DefaultSellingPrice   decimal.Decimal // Default selling price for this unit
	IsDefaultPurchaseUnit bool            // Whether this is the default purchase unit
	IsDefaultSalesUnit    bool            // Whether this is the default sales unit
	SortOrder             int             // Display order
	CreatedAt             time.Time       // Creation timestamp
	UpdatedAt             time.Time       // Last update timestamp
}

// NewProductUnit creates a new product unit
func NewProductUnit(tenantID, productID uuid.UUID, unitCode, unitName string, conversionRate decimal.Decimal) (*ProductUnit, error) {
	if err := validateUnitCode(unitCode); err != nil {
		return nil, err
	}
	if err := validateUnitName(unitName); err != nil {
		return nil, err
	}
	if err := validateConversionRate(conversionRate); err != nil {
		return nil, err
	}

	return &ProductUnit{
		ID:                    uuid.New(),
		TenantID:              tenantID,
		ProductID:             productID,
		UnitCode:              unitCode,
		UnitName:              unitName,
		ConversionRate:        conversionRate,
		DefaultPurchasePrice:  decimal.Zero,
		DefaultSellingPrice:   decimal.Zero,
		IsDefaultPurchaseUnit: false,
		IsDefaultSalesUnit:    false,
		SortOrder:             0,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}, nil
}

// Update updates the product unit's basic information
func (pu *ProductUnit) Update(unitName string, conversionRate decimal.Decimal) error {
	if err := validateUnitName(unitName); err != nil {
		return err
	}
	if err := validateConversionRate(conversionRate); err != nil {
		return err
	}

	pu.UnitName = unitName
	pu.ConversionRate = conversionRate
	pu.UpdatedAt = time.Now()

	return nil
}

// SetPrices sets default prices for this unit
func (pu *ProductUnit) SetPrices(purchasePrice, sellingPrice decimal.Decimal) error {
	if purchasePrice.IsNegative() {
		return shared.NewDomainError("INVALID_PRICE", "Purchase price cannot be negative")
	}
	if sellingPrice.IsNegative() {
		return shared.NewDomainError("INVALID_PRICE", "Selling price cannot be negative")
	}

	pu.DefaultPurchasePrice = purchasePrice
	pu.DefaultSellingPrice = sellingPrice
	pu.UpdatedAt = time.Now()

	return nil
}

// SetAsDefaultPurchaseUnit marks this unit as the default purchase unit
func (pu *ProductUnit) SetAsDefaultPurchaseUnit(isDefault bool) {
	pu.IsDefaultPurchaseUnit = isDefault
	pu.UpdatedAt = time.Now()
}

// SetAsDefaultSalesUnit marks this unit as the default sales unit
func (pu *ProductUnit) SetAsDefaultSalesUnit(isDefault bool) {
	pu.IsDefaultSalesUnit = isDefault
	pu.UpdatedAt = time.Now()
}

// SetSortOrder sets the display order
func (pu *ProductUnit) SetSortOrder(order int) {
	pu.SortOrder = order
	pu.UpdatedAt = time.Now()
}

// ConvertToBaseUnit converts quantity from this unit to base unit
// Formula: baseQuantity = quantity * conversionRate
func (pu *ProductUnit) ConvertToBaseUnit(quantity decimal.Decimal) decimal.Decimal {
	return quantity.Mul(pu.ConversionRate).Round(4)
}

// ConvertFromBaseUnit converts quantity from base unit to this unit
// Formula: unitQuantity = baseQuantity / conversionRate
func (pu *ProductUnit) ConvertFromBaseUnit(baseQuantity decimal.Decimal) decimal.Decimal {
	if pu.ConversionRate.IsZero() {
		return decimal.Zero
	}
	return baseQuantity.Div(pu.ConversionRate).Round(4)
}

// validateUnitCode validates the unit code
func validateUnitCode(code string) error {
	if code == "" {
		return shared.NewDomainError("INVALID_UNIT_CODE", "Unit code cannot be empty")
	}
	if len(code) > 20 {
		return shared.NewDomainError("INVALID_UNIT_CODE", "Unit code cannot exceed 20 characters")
	}
	return nil
}

// validateUnitName validates the unit name
func validateUnitName(name string) error {
	if name == "" {
		return shared.NewDomainError("INVALID_UNIT_NAME", "Unit name cannot be empty")
	}
	if len(name) > 50 {
		return shared.NewDomainError("INVALID_UNIT_NAME", "Unit name cannot exceed 50 characters")
	}
	return nil
}

// validateConversionRate validates the conversion rate
func validateConversionRate(rate decimal.Decimal) error {
	if rate.IsNegative() {
		return shared.NewDomainError("INVALID_CONVERSION_RATE", "Conversion rate cannot be negative")
	}
	if rate.IsZero() {
		return shared.NewDomainError("INVALID_CONVERSION_RATE", "Conversion rate cannot be zero")
	}
	return nil
}
