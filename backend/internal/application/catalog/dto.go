package catalog

import (
	"time"

	"github.com/erp/backend/internal/domain/catalog"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CreateProductRequest represents a request to create a new product
type CreateProductRequest struct {
	Code          string           `json:"code" binding:"required,min=1,max=50"`
	Name          string           `json:"name" binding:"required,min=1,max=200"`
	Description   string           `json:"description" binding:"max=2000"`
	Barcode       string           `json:"barcode" binding:"max=50"`
	CategoryID    *uuid.UUID       `json:"category_id"`
	Unit          string           `json:"unit" binding:"required,min=1,max=20"`
	PurchasePrice *decimal.Decimal `json:"purchase_price"`
	SellingPrice  *decimal.Decimal `json:"selling_price"`
	MinStock      *decimal.Decimal `json:"min_stock"`
	SortOrder     *int             `json:"sort_order"`
	Attributes    string           `json:"attributes"`
}

// UpdateProductRequest represents a request to update a product
type UpdateProductRequest struct {
	Name          *string          `json:"name" binding:"omitempty,min=1,max=200"`
	Description   *string          `json:"description" binding:"omitempty,max=2000"`
	Barcode       *string          `json:"barcode" binding:"omitempty,max=50"`
	CategoryID    *uuid.UUID       `json:"category_id"`
	PurchasePrice *decimal.Decimal `json:"purchase_price"`
	SellingPrice  *decimal.Decimal `json:"selling_price"`
	MinStock      *decimal.Decimal `json:"min_stock"`
	SortOrder     *int             `json:"sort_order"`
	Attributes    *string          `json:"attributes"`
}

// UpdateProductCodeRequest represents a request to update a product's code
type UpdateProductCodeRequest struct {
	Code string `json:"code" binding:"required,min=1,max=50"`
}

// ProductResponse represents a product in API responses
type ProductResponse struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	Code          string          `json:"code"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Barcode       string          `json:"barcode"`
	CategoryID    *uuid.UUID      `json:"category_id"`
	Unit          string          `json:"unit"`
	PurchasePrice decimal.Decimal `json:"purchase_price"`
	SellingPrice  decimal.Decimal `json:"selling_price"`
	MinStock      decimal.Decimal `json:"min_stock"`
	Status        string          `json:"status"`
	SortOrder     int             `json:"sort_order"`
	Attributes    string          `json:"attributes"`
	ProfitMargin  decimal.Decimal `json:"profit_margin"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	Version       int             `json:"version"`
}

// ProductListResponse represents a list item for products
type ProductListResponse struct {
	ID            uuid.UUID       `json:"id"`
	Code          string          `json:"code"`
	Name          string          `json:"name"`
	Barcode       string          `json:"barcode"`
	CategoryID    *uuid.UUID      `json:"category_id"`
	Unit          string          `json:"unit"`
	PurchasePrice decimal.Decimal `json:"purchase_price"`
	SellingPrice  decimal.Decimal `json:"selling_price"`
	Status        string          `json:"status"`
	SortOrder     int             `json:"sort_order"`
	CreatedAt     time.Time       `json:"created_at"`
}

// ProductListFilter represents filter options for product list
type ProductListFilter struct {
	Search     string     `form:"search"`
	Status     string     `form:"status" binding:"omitempty,oneof=active inactive discontinued"`
	CategoryID *uuid.UUID `form:"category_id"`
	Unit       string     `form:"unit"`
	MinPrice   *float64   `form:"min_price"`
	MaxPrice   *float64   `form:"max_price"`
	HasBarcode *bool      `form:"has_barcode"`
	Page       int        `form:"page" binding:"min=1"`
	PageSize   int        `form:"page_size" binding:"min=1,max=100"`
	OrderBy    string     `form:"order_by"`
	OrderDir   string     `form:"order_dir" binding:"omitempty,oneof=asc desc"`
}

// ToProductResponse converts a domain Product to ProductResponse
func ToProductResponse(p *catalog.Product) ProductResponse {
	return ProductResponse{
		ID:            p.ID,
		TenantID:      p.TenantID,
		Code:          p.Code,
		Name:          p.Name,
		Description:   p.Description,
		Barcode:       p.Barcode,
		CategoryID:    p.CategoryID,
		Unit:          p.Unit,
		PurchasePrice: p.PurchasePrice,
		SellingPrice:  p.SellingPrice,
		MinStock:      p.MinStock,
		Status:        string(p.Status),
		SortOrder:     p.SortOrder,
		Attributes:    p.Attributes,
		ProfitMargin:  p.GetProfitMargin(),
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
		Version:       p.Version,
	}
}

// ToProductListResponse converts a domain Product to ProductListResponse
func ToProductListResponse(p *catalog.Product) ProductListResponse {
	return ProductListResponse{
		ID:            p.ID,
		Code:          p.Code,
		Name:          p.Name,
		Barcode:       p.Barcode,
		CategoryID:    p.CategoryID,
		Unit:          p.Unit,
		PurchasePrice: p.PurchasePrice,
		SellingPrice:  p.SellingPrice,
		Status:        string(p.Status),
		SortOrder:     p.SortOrder,
		CreatedAt:     p.CreatedAt,
	}
}

// ToProductListResponses converts a slice of domain Products to ProductListResponses
func ToProductListResponses(products []catalog.Product) []ProductListResponse {
	responses := make([]ProductListResponse, len(products))
	for i, p := range products {
		responses[i] = ToProductListResponse(&p)
	}
	return responses
}

// ============================================================================
// Category DTOs
// ============================================================================

// CreateCategoryRequest represents a request to create a new category
type CreateCategoryRequest struct {
	Code        string     `json:"code" binding:"required,min=1,max=50"`
	Name        string     `json:"name" binding:"required,min=1,max=100"`
	Description string     `json:"description" binding:"max=2000"`
	ParentID    *uuid.UUID `json:"parent_id"`
	SortOrder   *int       `json:"sort_order"`
}

// UpdateCategoryRequest represents a request to update a category
type UpdateCategoryRequest struct {
	Name        string `json:"name" binding:"omitempty,min=1,max=100"`
	Description string `json:"description" binding:"omitempty,max=2000"`
	SortOrder   *int   `json:"sort_order"`
}

// MoveCategoryRequest represents a request to move a category to a new parent
type MoveCategoryRequest struct {
	ParentID *uuid.UUID `json:"parent_id"`
}

// CategoryResponse represents a category in API responses
type CategoryResponse struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	ParentID    *uuid.UUID `json:"parent_id"`
	Path        string     `json:"path"`
	Level       int        `json:"level"`
	SortOrder   int        `json:"sort_order"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Version     int        `json:"version"`
}

// CategoryListResponse represents a list item for categories
type CategoryListResponse struct {
	ID          uuid.UUID  `json:"id"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	ParentID    *uuid.UUID `json:"parent_id"`
	Level       int        `json:"level"`
	SortOrder   int        `json:"sort_order"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
}

// CategoryTreeNode represents a category with children in tree structure
type CategoryTreeNode struct {
	ID          uuid.UUID          `json:"id"`
	Code        string             `json:"code"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	ParentID    *uuid.UUID         `json:"parent_id"`
	Level       int                `json:"level"`
	SortOrder   int                `json:"sort_order"`
	Status      string             `json:"status"`
	Children    []CategoryTreeNode `json:"children"`
}

// CategoryListFilter represents filter options for category list
type CategoryListFilter struct {
	Search   string `form:"search"`
	Status   string `form:"status" binding:"omitempty,oneof=active inactive"`
	Page     int    `form:"page" binding:"min=0"`
	PageSize int    `form:"page_size" binding:"min=0,max=100"`
	SortBy   string `form:"sort_by"`
	SortDesc bool   `form:"sort_desc"`
}

// ToCategoryResponse converts a domain Category to CategoryResponse
func ToCategoryResponse(c *catalog.Category) *CategoryResponse {
	return &CategoryResponse{
		ID:          c.ID,
		TenantID:    c.TenantID,
		Code:        c.Code,
		Name:        c.Name,
		Description: c.Description,
		ParentID:    c.ParentID,
		Path:        c.Path,
		Level:       c.Level,
		SortOrder:   c.SortOrder,
		Status:      string(c.Status),
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
		Version:     c.Version,
	}
}

// ToCategoryListResponse converts a domain Category to CategoryListResponse
func ToCategoryListResponse(c *catalog.Category) CategoryListResponse {
	return CategoryListResponse{
		ID:          c.ID,
		Code:        c.Code,
		Name:        c.Name,
		Description: c.Description,
		ParentID:    c.ParentID,
		Level:       c.Level,
		SortOrder:   c.SortOrder,
		Status:      string(c.Status),
		CreatedAt:   c.CreatedAt,
	}
}

// ============================================================================
// ProductUnit DTOs
// ============================================================================

// CreateProductUnitRequest represents a request to create a product unit
type CreateProductUnitRequest struct {
	UnitCode              string           `json:"unit_code" binding:"required,min=1,max=20"`
	UnitName              string           `json:"unit_name" binding:"required,min=1,max=50"`
	ConversionRate        decimal.Decimal  `json:"conversion_rate" binding:"required,gt=0"`
	DefaultPurchasePrice  *decimal.Decimal `json:"default_purchase_price"`
	DefaultSellingPrice   *decimal.Decimal `json:"default_selling_price"`
	IsDefaultPurchaseUnit bool             `json:"is_default_purchase_unit"`
	IsDefaultSalesUnit    bool             `json:"is_default_sales_unit"`
	SortOrder             *int             `json:"sort_order"`
}

// UpdateProductUnitRequest represents a request to update a product unit
type UpdateProductUnitRequest struct {
	UnitName              *string          `json:"unit_name" binding:"omitempty,min=1,max=50"`
	ConversionRate        *decimal.Decimal `json:"conversion_rate" binding:"omitempty,gt=0"`
	DefaultPurchasePrice  *decimal.Decimal `json:"default_purchase_price"`
	DefaultSellingPrice   *decimal.Decimal `json:"default_selling_price"`
	IsDefaultPurchaseUnit *bool            `json:"is_default_purchase_unit"`
	IsDefaultSalesUnit    *bool            `json:"is_default_sales_unit"`
	SortOrder             *int             `json:"sort_order"`
}

// ProductUnitResponse represents a product unit in API responses
type ProductUnitResponse struct {
	ID                    uuid.UUID       `json:"id"`
	ProductID             uuid.UUID       `json:"product_id"`
	UnitCode              string          `json:"unit_code"`
	UnitName              string          `json:"unit_name"`
	ConversionRate        decimal.Decimal `json:"conversion_rate"`
	DefaultPurchasePrice  decimal.Decimal `json:"default_purchase_price"`
	DefaultSellingPrice   decimal.Decimal `json:"default_selling_price"`
	IsDefaultPurchaseUnit bool            `json:"is_default_purchase_unit"`
	IsDefaultSalesUnit    bool            `json:"is_default_sales_unit"`
	SortOrder             int             `json:"sort_order"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}

// ConvertUnitRequest represents a request to convert quantity between units
type ConvertUnitRequest struct {
	Quantity     decimal.Decimal `json:"quantity" binding:"required"`
	FromUnitCode string          `json:"from_unit_code" binding:"required"`
	ToUnitCode   string          `json:"to_unit_code" binding:"required"`
}

// ConvertUnitResponse represents the result of a unit conversion
type ConvertUnitResponse struct {
	FromQuantity decimal.Decimal `json:"from_quantity"`
	FromUnitCode string          `json:"from_unit_code"`
	ToQuantity   decimal.Decimal `json:"to_quantity"`
	ToUnitCode   string          `json:"to_unit_code"`
}

// ToProductUnitResponse converts a domain ProductUnit to ProductUnitResponse
func ToProductUnitResponse(u *catalog.ProductUnit) ProductUnitResponse {
	return ProductUnitResponse{
		ID:                    u.ID,
		ProductID:             u.ProductID,
		UnitCode:              u.UnitCode,
		UnitName:              u.UnitName,
		ConversionRate:        u.ConversionRate,
		DefaultPurchasePrice:  u.DefaultPurchasePrice,
		DefaultSellingPrice:   u.DefaultSellingPrice,
		IsDefaultPurchaseUnit: u.IsDefaultPurchaseUnit,
		IsDefaultSalesUnit:    u.IsDefaultSalesUnit,
		SortOrder:             u.SortOrder,
		CreatedAt:             u.CreatedAt,
		UpdatedAt:             u.UpdatedAt,
	}
}

// ToProductUnitResponses converts a slice of domain ProductUnits to ProductUnitResponses
func ToProductUnitResponses(units []catalog.ProductUnit) []ProductUnitResponse {
	responses := make([]ProductUnitResponse, len(units))
	for i, u := range units {
		responses[i] = ToProductUnitResponse(&u)
	}
	return responses
}
