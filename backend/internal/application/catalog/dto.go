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
