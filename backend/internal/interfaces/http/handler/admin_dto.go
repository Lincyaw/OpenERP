package handler

import (
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Query DTOs
// ============================================================================

// AdminTenantListQuery represents query parameters for listing tenants
type AdminTenantListQuery struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Search   string `form:"search" binding:"omitempty,max=200"`
	Status   string `form:"status" binding:"omitempty,oneof=active inactive suspended trial"`
	Plan     string `form:"plan" binding:"omitempty,oneof=free basic pro enterprise"`
	OrderBy  string `form:"order_by" binding:"omitempty,oneof=name code status plan created_at"`
	OrderDir string `form:"order_dir" binding:"omitempty,oneof=asc desc"`
}

// TenantGrowthQuery represents query parameters for growth trend
type TenantGrowthQuery struct {
	Period string `form:"period" binding:"omitempty,oneof=daily weekly monthly"`
	Days   int    `form:"days" binding:"omitempty,min=1,max=365"`
}

// AuditLogListQuery represents query parameters for listing audit logs
type AuditLogListQuery struct {
	Page        int        `form:"page" binding:"omitempty,min=1"`
	PageSize    int        `form:"page_size" binding:"omitempty,min=1,max=100"`
	AdminUserID string     `form:"admin_user_id" binding:"omitempty"`
	Action      string     `form:"action" binding:"omitempty"`
	TargetType  string     `form:"target_type" binding:"omitempty"`
	TargetID    string     `form:"target_id" binding:"omitempty"`
	StartTime   *time.Time `form:"start_time" binding:"omitempty"`
	EndTime     *time.Time `form:"end_time" binding:"omitempty"`
	IPAddress   string     `form:"ip_address" binding:"omitempty"`
	SortBy      string     `form:"sort_by" binding:"omitempty,oneof=created_at action target_type"`
	SortOrder   string     `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

// ============================================================================
// Request DTOs
// ============================================================================

// AdminCreateTenantRequest represents a request to create a tenant
type AdminCreateTenantRequest struct {
	Code         string `json:"code" binding:"required,min=2,max=50"`
	Name         string `json:"name" binding:"required,min=2,max=200"`
	ShortName    string `json:"short_name,omitempty" binding:"omitempty,max=50"`
	ContactName  string `json:"contact_name,omitempty" binding:"omitempty,max=100"`
	ContactPhone string `json:"contact_phone,omitempty" binding:"omitempty,max=20"`
	ContactEmail string `json:"contact_email,omitempty" binding:"omitempty,email,max=200"`
	Address      string `json:"address,omitempty" binding:"omitempty,max=500"`
	Plan         string `json:"plan,omitempty" binding:"omitempty,oneof=free basic pro enterprise"`
	TrialDays    int    `json:"trial_days,omitempty" binding:"omitempty,min=0,max=90"`
	Notes        string `json:"notes,omitempty" binding:"omitempty,max=1000"`
}

// AdminUpdateTenantRequest represents a request to update a tenant
type AdminUpdateTenantRequest struct {
	Name         *string `json:"name,omitempty" binding:"omitempty,min=2,max=200"`
	ShortName    *string `json:"short_name,omitempty" binding:"omitempty,max=50"`
	ContactName  *string `json:"contact_name,omitempty" binding:"omitempty,max=100"`
	ContactPhone *string `json:"contact_phone,omitempty" binding:"omitempty,max=20"`
	ContactEmail *string `json:"contact_email,omitempty" binding:"omitempty,email,max=200"`
	Address      *string `json:"address,omitempty" binding:"omitempty,max=500"`
	Notes        *string `json:"notes,omitempty" binding:"omitempty,max=1000"`
}

// SuspendTenantRequest represents a request to suspend a tenant
type SuspendTenantRequest struct {
	Reason                string     `json:"reason,omitempty" binding:"omitempty,max=500"`
	ScheduledReactivateAt *time.Time `json:"scheduled_reactivate_at,omitempty"`
}

// ============================================================================
// Response DTOs
// ============================================================================

// AdminTenantListResponse represents a paginated list of tenants
type AdminTenantListResponse struct {
	Tenants    []AdminTenantResponse `json:"tenants"`
	Total      int64                 `json:"total"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
	TotalPages int                   `json:"total_pages"`
}

// TenantUsageStatsResponse represents tenant usage statistics
type TenantUsageStatsResponse struct {
	TenantID       uuid.UUID `json:"tenant_id"`
	TenantName     string    `json:"tenant_name"`
	TenantCode     string    `json:"tenant_code"`
	Plan           string    `json:"plan"`
	Status         string    `json:"status"`
	UserCount      int64     `json:"user_count"`
	ProductCount   int64     `json:"product_count"`
	OrderCount     int64     `json:"order_count"`
	WarehouseCount int64     `json:"warehouse_count"`
	StorageUsage   int64     `json:"storage_usage"`
	APICallCount   int64     `json:"api_call_count"`
	MaxUsers       int       `json:"max_users"`
	MaxProducts    int       `json:"max_products"`
	MaxWarehouses  int       `json:"max_warehouses"`
	LastUpdated    time.Time `json:"last_updated"`
}

// PlatformStatsResponse represents platform-wide statistics
type PlatformStatsResponse struct {
	TotalTenants      int64            `json:"total_tenants"`
	ActiveTenants     int64            `json:"active_tenants"`
	TrialTenants      int64            `json:"trial_tenants"`
	SuspendedTenants  int64            `json:"suspended_tenants"`
	InactiveTenants   int64            `json:"inactive_tenants"`
	TenantsByPlan     map[string]int64 `json:"tenants_by_plan"`
	TotalUsers        int64            `json:"total_users"`
	TotalProducts     int64            `json:"total_products"`
	TotalOrders       int64            `json:"total_orders"`
	TotalStorageUsage int64            `json:"total_storage_usage"`
	LastUpdated       time.Time        `json:"last_updated"`
}

// TenantGrowthTrendResponse represents tenant growth trend data
type TenantGrowthTrendResponse struct {
	Period     string                      `json:"period"`
	StartDate  string                      `json:"start_date"`
	EndDate    string                      `json:"end_date"`
	DataPoints []TenantGrowthPointResponse `json:"data_points"`
}

// TenantGrowthPointResponse represents a single data point in growth trend
type TenantGrowthPointResponse struct {
	Date         string `json:"date"`
	TotalTenants int64  `json:"total_tenants"`
	NewTenants   int64  `json:"new_tenants"`
	ChurnedCount int64  `json:"churned_count"`
}

// AdminAuditLogListResponse represents a paginated list of audit logs
type AdminAuditLogListResponse struct {
	Logs       []AdminAuditLogResponse `json:"logs"`
	Total      int64                   `json:"total"`
	Page       int                     `json:"page"`
	PageSize   int                     `json:"page_size"`
	TotalPages int                     `json:"total_pages"`
}

// AdminAuditLogResponse represents a single audit log entry
type AdminAuditLogResponse struct {
	ID          uuid.UUID              `json:"id"`
	AdminUserID uuid.UUID              `json:"admin_user_id"`
	Action      string                 `json:"action"`
	TargetType  string                 `json:"target_type"`
	TargetID    *uuid.UUID             `json:"target_id,omitempty"`
	OldValue    map[string]interface{} `json:"old_value,omitempty"`
	NewValue    map[string]interface{} `json:"new_value,omitempty"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// MessageResponse represents a simple message response
type MessageResponse struct {
	Message string `json:"message"`
}
