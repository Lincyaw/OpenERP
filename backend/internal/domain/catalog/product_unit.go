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
	ID                    uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	TenantID              uuid.UUID       `gorm:"type:uuid;not null;index"`
	ProductID             uuid.UUID       `gorm:"type:uuid;not null;index;uniqueIndex:idx_product_unit_code,priority:2"`
	UnitCode              string          `gorm:"type:varchar(20);not null;uniqueIndex:idx_product_unit_code,priority:3"`
	UnitName              string          `gorm:"type:varchar(50);not null"`
	ConversionRate        decimal.Decimal `gorm:"type:decimal(18,6);not null"` // Rate to convert to base unit
	DefaultPurchasePrice  decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0"`
	DefaultSellingPrice   decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0"`
	IsDefaultPurchaseUnit bool            `gorm:"not null;default:false"`
	IsDefaultSalesUnit    bool            `gorm:"not null;default:false"`
	SortOrder             int             `gorm:"not null;default:0"`
	CreatedAt             time.Time       `gorm:"not null;autoCreateTime"`
	UpdatedAt             time.Time       `gorm:"not null;autoUpdateTime"`
}

// TableName returns the table name for GORM
func (ProductUnit) TableName() string {
	return "product_units"
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
