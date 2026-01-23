package catalog

import (
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ProductStatus represents the status of a product
type ProductStatus string

const (
	ProductStatusActive       ProductStatus = "active"
	ProductStatusInactive     ProductStatus = "inactive"
	ProductStatusDiscontinued ProductStatus = "discontinued"
)

// Product represents a product/SKU in the catalog
// It is the aggregate root for product-related operations
type Product struct {
	shared.TenantAggregateRoot
	Code          string          `gorm:"type:varchar(50);not null;uniqueIndex:idx_product_tenant_code,priority:2"`
	Name          string          `gorm:"type:varchar(200);not null"`
	Description   string          `gorm:"type:text"`
	Barcode       string          `gorm:"type:varchar(50);index"`
	CategoryID    *uuid.UUID      `gorm:"type:uuid;index"`
	Unit          string          `gorm:"type:varchar(20);not null"`             // Base unit (e.g., "pcs", "kg", "box")
	PurchasePrice decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0"` // Cost price
	SellingPrice  decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0"` // Selling price
	MinStock      decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0"` // Minimum stock level for alerts
	Status        ProductStatus   `gorm:"type:varchar(20);not null;default:'active'"`
	SortOrder     int             `gorm:"not null;default:0"`
	Attributes    string          `gorm:"type:jsonb"` // JSON storage for custom attributes
}

// TableName returns the table name for GORM
func (Product) TableName() string {
	return "products"
}

// NewProduct creates a new product
func NewProduct(tenantID uuid.UUID, code, name, unit string) (*Product, error) {
	if err := validateProductCode(code); err != nil {
		return nil, err
	}
	if err := validateProductName(name); err != nil {
		return nil, err
	}
	if err := validateUnit(unit); err != nil {
		return nil, err
	}

	product := &Product{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		Code:                strings.ToUpper(code),
		Name:                name,
		Unit:                unit,
		PurchasePrice:       decimal.Zero,
		SellingPrice:        decimal.Zero,
		MinStock:            decimal.Zero,
		Status:              ProductStatusActive,
		Attributes:          "{}",
	}

	product.AddDomainEvent(NewProductCreatedEvent(product))

	return product, nil
}

// NewProductWithPrices creates a new product with prices
func NewProductWithPrices(
	tenantID uuid.UUID,
	code, name, unit string,
	purchasePrice, sellingPrice valueobject.Money,
) (*Product, error) {
	product, err := NewProduct(tenantID, code, name, unit)
	if err != nil {
		return nil, err
	}

	if err := product.SetPrices(purchasePrice, sellingPrice); err != nil {
		return nil, err
	}

	return product, nil
}

// Update updates the product's basic information
func (p *Product) Update(name, description string) error {
	if err := validateProductName(name); err != nil {
		return err
	}

	p.Name = name
	p.Description = description
	p.UpdatedAt = time.Now()
	p.IncrementVersion()

	p.AddDomainEvent(NewProductUpdatedEvent(p))

	return nil
}

// UpdateCode updates the product's code
// Note: This should be used with caution as other systems may reference the product code
func (p *Product) UpdateCode(code string) error {
	if err := validateProductCode(code); err != nil {
		return err
	}

	p.Code = strings.ToUpper(code)
	p.UpdatedAt = time.Now()
	p.IncrementVersion()

	p.AddDomainEvent(NewProductUpdatedEvent(p))

	return nil
}

// SetBarcode sets the product barcode
func (p *Product) SetBarcode(barcode string) error {
	if barcode != "" && len(barcode) > 50 {
		return shared.NewDomainError("INVALID_BARCODE", "Barcode cannot exceed 50 characters")
	}

	p.Barcode = barcode
	p.UpdatedAt = time.Now()
	p.IncrementVersion()

	return nil
}

// SetCategory sets the product category
func (p *Product) SetCategory(categoryID *uuid.UUID) {
	p.CategoryID = categoryID
	p.UpdatedAt = time.Now()
	p.IncrementVersion()

	p.AddDomainEvent(NewProductUpdatedEvent(p))
}

// SetPrices sets both purchase and selling prices
func (p *Product) SetPrices(purchasePrice, sellingPrice valueobject.Money) error {
	if purchasePrice.Amount().IsNegative() {
		return shared.NewDomainError("INVALID_PRICE", "Purchase price cannot be negative")
	}
	if sellingPrice.Amount().IsNegative() {
		return shared.NewDomainError("INVALID_PRICE", "Selling price cannot be negative")
	}

	oldPurchasePrice := p.PurchasePrice
	oldSellingPrice := p.SellingPrice

	p.PurchasePrice = purchasePrice.Amount()
	p.SellingPrice = sellingPrice.Amount()
	p.UpdatedAt = time.Now()
	p.IncrementVersion()

	p.AddDomainEvent(NewProductPriceChangedEvent(p, oldPurchasePrice, oldSellingPrice))

	return nil
}

// UpdatePurchasePrice updates only the purchase price
func (p *Product) UpdatePurchasePrice(price valueobject.Money) error {
	if price.Amount().IsNegative() {
		return shared.NewDomainError("INVALID_PRICE", "Purchase price cannot be negative")
	}

	oldPrice := p.PurchasePrice
	p.PurchasePrice = price.Amount()
	p.UpdatedAt = time.Now()
	p.IncrementVersion()

	p.AddDomainEvent(NewProductPriceChangedEvent(p, oldPrice, p.SellingPrice))

	return nil
}

// UpdateSellingPrice updates only the selling price
func (p *Product) UpdateSellingPrice(price valueobject.Money) error {
	if price.Amount().IsNegative() {
		return shared.NewDomainError("INVALID_PRICE", "Selling price cannot be negative")
	}

	oldPrice := p.SellingPrice
	p.SellingPrice = price.Amount()
	p.UpdatedAt = time.Now()
	p.IncrementVersion()

	p.AddDomainEvent(NewProductPriceChangedEvent(p, p.PurchasePrice, oldPrice))

	return nil
}

// SetMinStock sets the minimum stock level for alerts
func (p *Product) SetMinStock(minStock decimal.Decimal) error {
	if minStock.IsNegative() {
		return shared.NewDomainError("INVALID_MIN_STOCK", "Minimum stock cannot be negative")
	}

	p.MinStock = minStock
	p.UpdatedAt = time.Now()
	p.IncrementVersion()

	return nil
}

// SetSortOrder sets the display order of the product
func (p *Product) SetSortOrder(order int) {
	p.SortOrder = order
	p.UpdatedAt = time.Now()
	p.IncrementVersion()
}

// SetAttributes sets custom attributes as JSON
func (p *Product) SetAttributes(attributes string) error {
	if attributes == "" {
		attributes = "{}"
	}
	// Basic JSON validation - should start with { and end with }
	trimmed := strings.TrimSpace(attributes)
	if !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
		return shared.NewDomainError("INVALID_ATTRIBUTES", "Attributes must be valid JSON object")
	}

	p.Attributes = trimmed
	p.UpdatedAt = time.Now()
	p.IncrementVersion()

	return nil
}

// Activate activates the product
func (p *Product) Activate() error {
	if p.Status == ProductStatusActive {
		return shared.NewDomainError("ALREADY_ACTIVE", "Product is already active")
	}
	if p.Status == ProductStatusDiscontinued {
		return shared.NewDomainError("CANNOT_ACTIVATE", "Cannot activate a discontinued product")
	}

	oldStatus := p.Status
	p.Status = ProductStatusActive
	p.UpdatedAt = time.Now()
	p.IncrementVersion()

	p.AddDomainEvent(NewProductStatusChangedEvent(p, oldStatus, ProductStatusActive))

	return nil
}

// Deactivate deactivates the product
func (p *Product) Deactivate() error {
	if p.Status == ProductStatusInactive {
		return shared.NewDomainError("ALREADY_INACTIVE", "Product is already inactive")
	}
	if p.Status == ProductStatusDiscontinued {
		return shared.NewDomainError("CANNOT_DEACTIVATE", "Cannot deactivate a discontinued product")
	}

	oldStatus := p.Status
	p.Status = ProductStatusInactive
	p.UpdatedAt = time.Now()
	p.IncrementVersion()

	p.AddDomainEvent(NewProductStatusChangedEvent(p, oldStatus, ProductStatusInactive))

	return nil
}

// Discontinue marks the product as discontinued
// A discontinued product cannot be reactivated
func (p *Product) Discontinue() error {
	if p.Status == ProductStatusDiscontinued {
		return shared.NewDomainError("ALREADY_DISCONTINUED", "Product is already discontinued")
	}

	oldStatus := p.Status
	p.Status = ProductStatusDiscontinued
	p.UpdatedAt = time.Now()
	p.IncrementVersion()

	p.AddDomainEvent(NewProductStatusChangedEvent(p, oldStatus, ProductStatusDiscontinued))

	return nil
}

// IsActive returns true if the product is active
func (p *Product) IsActive() bool {
	return p.Status == ProductStatusActive
}

// IsInactive returns true if the product is inactive
func (p *Product) IsInactive() bool {
	return p.Status == ProductStatusInactive
}

// IsDiscontinued returns true if the product is discontinued
func (p *Product) IsDiscontinued() bool {
	return p.Status == ProductStatusDiscontinued
}

// HasCategory returns true if the product has a category assigned
func (p *Product) HasCategory() bool {
	return p.CategoryID != nil
}

// GetPurchasePriceMoney returns purchase price as Money value object
func (p *Product) GetPurchasePriceMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(p.PurchasePrice)
}

// GetSellingPriceMoney returns selling price as Money value object
func (p *Product) GetSellingPriceMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(p.SellingPrice)
}

// GetProfitMargin returns the profit margin percentage
// Returns 0 if purchase price is zero
func (p *Product) GetProfitMargin() decimal.Decimal {
	if p.PurchasePrice.IsZero() {
		return decimal.Zero
	}
	profit := p.SellingPrice.Sub(p.PurchasePrice)
	return profit.Div(p.PurchasePrice).Mul(decimal.NewFromInt(100))
}

// validateProductCode validates the product code (SKU)
func validateProductCode(code string) error {
	if code == "" {
		return shared.NewDomainError("INVALID_CODE", "Product code cannot be empty")
	}
	if len(code) > 50 {
		return shared.NewDomainError("INVALID_CODE", "Product code cannot exceed 50 characters")
	}
	// Code should be alphanumeric with underscores and hyphens
	for _, r := range code {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return shared.NewDomainError("INVALID_CODE", "Product code can only contain letters, numbers, underscores, and hyphens")
		}
	}
	return nil
}

// validateProductName validates the product name
func validateProductName(name string) error {
	if name == "" {
		return shared.NewDomainError("INVALID_NAME", "Product name cannot be empty")
	}
	if len(name) > 200 {
		return shared.NewDomainError("INVALID_NAME", "Product name cannot exceed 200 characters")
	}
	return nil
}

// validateUnit validates the unit
func validateUnit(unit string) error {
	if unit == "" {
		return shared.NewDomainError("INVALID_UNIT", "Unit cannot be empty")
	}
	if len(unit) > 20 {
		return shared.NewDomainError("INVALID_UNIT", "Unit cannot exceed 20 characters")
	}
	return nil
}
