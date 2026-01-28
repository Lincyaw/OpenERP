package handler

import (
	"time"

	"github.com/google/uuid"
)

// =====================
// Tenant Request DTOs
// =====================

// CreateTenantRequest represents the request body for creating a tenant
type CreateTenantRequest struct {
	Code         string `json:"code" binding:"required,min=2,max=50"`
	Name         string `json:"name" binding:"required,min=1,max=200"`
	ShortName    string `json:"short_name" binding:"omitempty,max=100"`
	ContactName  string `json:"contact_name" binding:"omitempty,max=100"`
	ContactPhone string `json:"contact_phone" binding:"omitempty,max=50"`
	ContactEmail string `json:"contact_email" binding:"omitempty,email,max=200"`
	Address      string `json:"address" binding:"omitempty,max=500"`
	LogoURL      string `json:"logo_url" binding:"omitempty,url,max=500"`
	Domain       string `json:"domain" binding:"omitempty,max=200"`
	Plan         string `json:"plan" binding:"omitempty,oneof=free basic pro enterprise"`
	Notes        string `json:"notes" binding:"omitempty"`
	TrialDays    int    `json:"trial_days" binding:"omitempty,min=1,max=365"`
}

// UpdateTenantRequest represents the request body for updating a tenant
// @name HandlerUpdateTenantRequest
type UpdateTenantRequest struct {
	Name         *string `json:"name" binding:"omitempty,min=1,max=200"`
	ShortName    *string `json:"short_name" binding:"omitempty,max=100"`
	ContactName  *string `json:"contact_name" binding:"omitempty,max=100"`
	ContactPhone *string `json:"contact_phone" binding:"omitempty,max=50"`
	ContactEmail *string `json:"contact_email" binding:"omitempty,email,max=200"`
	Address      *string `json:"address" binding:"omitempty,max=500"`
	LogoURL      *string `json:"logo_url" binding:"omitempty,max=500"`
	Domain       *string `json:"domain" binding:"omitempty,max=200"`
	Notes        *string `json:"notes" binding:"omitempty"`
}

// UpdateTenantConfigRequest represents the request body for updating tenant configuration
type UpdateTenantConfigRequest struct {
	MaxUsers      *int    `json:"max_users" binding:"omitempty,min=0"`
	MaxWarehouses *int    `json:"max_warehouses" binding:"omitempty,min=0"`
	MaxProducts   *int    `json:"max_products" binding:"omitempty,min=0"`
	CostStrategy  *string `json:"cost_strategy" binding:"omitempty,oneof=fifo weighted_average"`
	Currency      *string `json:"currency" binding:"omitempty,len=3"`
	Timezone      *string `json:"timezone" binding:"omitempty,max=50"`
	Locale        *string `json:"locale" binding:"omitempty,max=10"`
}

// SetTenantPlanRequest represents the request body for setting tenant plan
type SetTenantPlanRequest struct {
	Plan string `json:"plan" binding:"required,oneof=free basic pro enterprise"`
}

// TenantListQuery represents query parameters for listing tenants
type TenantListQuery struct {
	Keyword  string `form:"keyword" binding:"omitempty"`
	Status   string `form:"status" binding:"omitempty,oneof=active inactive suspended trial"`
	Plan     string `form:"plan" binding:"omitempty,oneof=free basic pro enterprise"`
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	SortBy   string `form:"sort_by" binding:"omitempty,oneof=code name status plan created_at updated_at"`
	SortDir  string `form:"sort_dir" binding:"omitempty,oneof=asc desc"`
}

// =====================
// Tenant Response DTOs
// =====================

// TenantResponse represents a tenant in API responses
type TenantResponse struct {
	ID           uuid.UUID            `json:"id"`
	Code         string               `json:"code"`
	Name         string               `json:"name"`
	ShortName    string               `json:"short_name,omitempty"`
	Status       string               `json:"status"`
	Plan         string               `json:"plan"`
	ContactName  string               `json:"contact_name,omitempty"`
	ContactPhone string               `json:"contact_phone,omitempty"`
	ContactEmail string               `json:"contact_email,omitempty"`
	Address      string               `json:"address,omitempty"`
	LogoURL      string               `json:"logo_url,omitempty"`
	Domain       string               `json:"domain,omitempty"`
	ExpiresAt    *time.Time           `json:"expires_at,omitempty"`
	TrialEndsAt  *time.Time           `json:"trial_ends_at,omitempty"`
	Config       TenantConfigResponse `json:"config"`
	Notes        string               `json:"notes,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
}

// TenantConfigResponse represents tenant configuration in API responses
type TenantConfigResponse struct {
	MaxUsers      int    `json:"max_users"`
	MaxWarehouses int    `json:"max_warehouses"`
	MaxProducts   int    `json:"max_products"`
	CostStrategy  string `json:"cost_strategy"`
	Currency      string `json:"currency"`
	Timezone      string `json:"timezone"`
	Locale        string `json:"locale"`
}

// TenantListResponse represents a paginated list of tenants
type TenantListResponse struct {
	Tenants    []TenantResponse `json:"tenants"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

// TenantStatsResponse represents tenant statistics
type TenantStatsResponse struct {
	Total     int64 `json:"total"`
	Active    int64 `json:"active"`
	Trial     int64 `json:"trial"`
	Inactive  int64 `json:"inactive"`
	Suspended int64 `json:"suspended"`
}
