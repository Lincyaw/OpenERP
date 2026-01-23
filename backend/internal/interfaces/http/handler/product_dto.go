package handler

import "time"

// ProductResponse represents a product in API responses
// @Description Product details returned by the API
type ProductResponse struct {
	ID            string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantID      string  `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Code          string  `json:"code" example:"SKU-001"`
	Name          string  `json:"name" example:"Sample Product"`
	Description   string  `json:"description" example:"This is a sample product description"`
	Barcode       string  `json:"barcode" example:"6901234567890"`
	CategoryID    *string `json:"category_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	Unit          string  `json:"unit" example:"pcs"`
	PurchasePrice float64 `json:"purchase_price" example:"50.00"`
	SellingPrice  float64 `json:"selling_price" example:"100.00"`
	MinStock      float64 `json:"min_stock" example:"10"`
	Status        string  `json:"status" example:"active" enums:"active,inactive,discontinued"`
	SortOrder     int     `json:"sort_order" example:"0"`
	Attributes    string  `json:"attributes" example:"{}"`
	ProfitMargin  float64 `json:"profit_margin" example:"100.00"`
	CreatedAt     string  `json:"created_at" example:"2026-01-24T12:00:00Z"`
	UpdatedAt     string  `json:"updated_at" example:"2026-01-24T12:00:00Z"`
	Version       int     `json:"version" example:"1"`
}

// ProductListResponse represents a product list item
// @Description Product list item with basic information
type ProductListResponse struct {
	ID            string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Code          string  `json:"code" example:"SKU-001"`
	Name          string  `json:"name" example:"Sample Product"`
	Barcode       string  `json:"barcode" example:"6901234567890"`
	CategoryID    *string `json:"category_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	Unit          string  `json:"unit" example:"pcs"`
	PurchasePrice float64 `json:"purchase_price" example:"50.00"`
	SellingPrice  float64 `json:"selling_price" example:"100.00"`
	Status        string  `json:"status" example:"active" enums:"active,inactive,discontinued"`
	SortOrder     int     `json:"sort_order" example:"0"`
	CreatedAt     string  `json:"created_at" example:"2026-01-24T12:00:00Z"`
}

// Helper to suppress unused import warning
var _ = time.Now
