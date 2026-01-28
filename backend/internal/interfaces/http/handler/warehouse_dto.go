package handler

// WarehouseResponse represents a warehouse in API responses
// @Description Warehouse details returned by the API
// @Name HandlerWarehouseResponse
type WarehouseResponse struct {
	ID          string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantID    string `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Code        string `json:"code" example:"WH-001"`
	Name        string `json:"name" example:"Main Warehouse"`
	ShortName   string `json:"short_name" example:"Main WH"`
	Type        string `json:"type" example:"normal" enums:"normal,virtual,transit"`
	Status      string `json:"status" example:"enabled" enums:"enabled,disabled"`
	IsDefault   bool   `json:"is_default" example:"true"`
	ManagerName string `json:"manager_name" example:"John Manager"`
	Phone       string `json:"phone" example:"13800138000"`
	Email       string `json:"email" example:"warehouse@company.com"`
	Address     string `json:"address" example:"789 Storage Road"`
	City        string `json:"city" example:"Guangzhou"`
	Province    string `json:"province" example:"Guangdong"`
	PostalCode  string `json:"postal_code" example:"510000"`
	Country     string `json:"country" example:"China"`
	FullAddress string `json:"full_address" example:"789 Storage Road, Guangzhou, Guangdong 510000, China"`
	Notes       string `json:"notes" example:"Main distribution center"`
	SortOrder   int    `json:"sort_order" example:"0"`
	Attributes  string `json:"attributes" example:"{}"`
	CreatedAt   string `json:"created_at" example:"2026-01-24T12:00:00Z"`
	UpdatedAt   string `json:"updated_at" example:"2026-01-24T12:00:00Z"`
	Version     int    `json:"version" example:"1"`
}

// WarehouseListResponse represents a warehouse list item
// @Description Warehouse list item with basic information
// @Name HandlerWarehouseListResponse
type WarehouseListResponse struct {
	ID        string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Code      string `json:"code" example:"WH-001"`
	Name      string `json:"name" example:"Main Warehouse"`
	ShortName string `json:"short_name" example:"Main WH"`
	Type      string `json:"type" example:"normal" enums:"normal,virtual,transit"`
	Status    string `json:"status" example:"enabled" enums:"enabled,disabled"`
	IsDefault bool   `json:"is_default" example:"true"`
	City      string `json:"city" example:"Guangzhou"`
	Province  string `json:"province" example:"Guangdong"`
	SortOrder int    `json:"sort_order" example:"0"`
	CreatedAt string `json:"created_at" example:"2026-01-24T12:00:00Z"`
}

// WarehouseCountResponse represents warehouse count statistics
// @Description Warehouse counts by status
// @Name HandlerWarehouseCountResponse
type WarehouseCountResponse struct {
	Enabled  int64 `json:"enabled" example:"3"`
	Disabled int64 `json:"disabled" example:"1"`
	Total    int64 `json:"total" example:"4"`
}
