package handler

import (
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// API Documentation
// ============================================================================

// Admin API requires SuperAdmin role (is_super_admin: true in JWT claims).
// All endpoints return standardized error responses with the following codes:
//
// | HTTP Status | Error Code         | Description                                    |
// |-------------|--------------------|-------------------------------------------------|
// | 400         | ERR_BAD_REQUEST    | Invalid request parameters or body             |
// | 401         | ERR_UNAUTHORIZED   | Missing or invalid authentication token        |
// | 403         | ERR_FORBIDDEN      | User lacks super admin privileges              |
// | 404         | ERR_NOT_FOUND      | Requested tenant/resource not found            |
// | 409         | ERR_ALREADY_EXISTS | Tenant code/name already exists                |
// | 422         | ERR_BUSINESS_RULE  | Business rule violation (e.g., system tenant)  |
// | 500         | ERR_INTERNAL       | Internal server error                          |

// ============================================================================
// API Changelog
// ============================================================================
//
// Version 1.0.0 (2026-02-02) - Initial Admin API Release
//
// ## Tenant Management
// - GET    /admin/tenants             - List all tenants with pagination and filtering
// - POST   /admin/tenants             - Create a new tenant
// - GET    /admin/tenants/{id}        - Get tenant details
// - PUT    /admin/tenants/{id}        - Update tenant information
// - DELETE /admin/tenants/{id}        - Soft delete a tenant
//
// ## Tenant Status Management
// - POST   /admin/tenants/{id}/suspend   - Suspend a tenant
// - POST   /admin/tenants/{id}/activate  - Activate a tenant
//
// ## Subscription Management
// - PUT    /admin/tenants/{id}/plan      - Change subscription plan
// - PUT    /admin/tenants/{id}/quota     - Update quota limits
// - GET    /admin/tenants/{id}/subscription-history - Get subscription history
// - DELETE /admin/tenants/{id}/scheduled-plan       - Cancel scheduled plan change
//
// ## Statistics
// - GET    /admin/tenants/{id}/stats  - Get tenant usage statistics
// - GET    /admin/stats               - Get platform-wide statistics
// - GET    /admin/stats/growth        - Get tenant growth trend
//
// ## Audit Logs
// - GET    /admin/audit-logs          - List audit logs with filtering
//
// ## Batch Operations
// - POST   /admin/batch/suspend/preview      - Preview batch suspend
// - POST   /admin/batch/suspend              - Execute batch suspend
// - POST   /admin/batch/activate/preview     - Preview batch activate
// - POST   /admin/batch/activate             - Execute batch activate
// - POST   /admin/batch/change-plan/preview  - Preview batch plan change
// - POST   /admin/batch/change-plan          - Execute batch plan change
// - POST   /admin/batch/delete/preview       - Preview batch delete
// - POST   /admin/batch/delete               - Execute batch delete
//
// ## Breaking Changes
// None (initial release)
//
// ## Deprecations
// None
//
// ## Known Issues
// - Routes are defined but not yet registered in main.go
// - Frontend integration pending (P3-ADMIN-014 through P3-ADMIN-022)

// ============================================================================
// Query DTOs
// ============================================================================

// AdminTenantListQuery represents query parameters for listing tenants
//
//	@Description Query parameters for tenant list endpoint with filtering and pagination
type AdminTenantListQuery struct {
	Page     int    `form:"page" binding:"omitempty,min=1" example:"1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100" example:"20"`
	Search   string `form:"search" binding:"omitempty,max=200" example:"acme"`
	Status   string `form:"status" binding:"omitempty,oneof=active inactive suspended trial" example:"active"`
	Plan     string `form:"plan" binding:"omitempty,oneof=free basic pro enterprise" example:"pro"`
	OrderBy  string `form:"order_by" binding:"omitempty,oneof=name code status plan created_at" example:"created_at"`
	OrderDir string `form:"order_dir" binding:"omitempty,oneof=asc desc" example:"desc"`
}

// TenantGrowthQuery represents query parameters for growth trend
//
//	@Description Query parameters for tenant growth trend endpoint
type TenantGrowthQuery struct {
	Period string `form:"period" binding:"omitempty,oneof=daily weekly monthly" example:"daily"`
	Days   int    `form:"days" binding:"omitempty,min=1,max=365" example:"30"`
}

// AuditLogListQuery represents query parameters for listing audit logs
//
//	@Description Query parameters for audit log list with filtering options
type AuditLogListQuery struct {
	Page        int        `form:"page" binding:"omitempty,min=1" example:"1"`
	PageSize    int        `form:"page_size" binding:"omitempty,min=1,max=100" example:"50"`
	AdminUserID string     `form:"admin_user_id" binding:"omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	Action      string     `form:"action" binding:"omitempty" example:"tenant_create"`
	TargetType  string     `form:"target_type" binding:"omitempty" example:"tenant"`
	TargetID    string     `form:"target_id" binding:"omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	StartTime   *time.Time `form:"start_time" binding:"omitempty"`
	EndTime     *time.Time `form:"end_time" binding:"omitempty"`
	IPAddress   string     `form:"ip_address" binding:"omitempty" example:"192.168.1.100"`
	SortBy      string     `form:"sort_by" binding:"omitempty,oneof=created_at action target_type" example:"created_at"`
	SortOrder   string     `form:"sort_order" binding:"omitempty,oneof=asc desc" example:"desc"`
}

// ============================================================================
// Request DTOs
// ============================================================================

// AdminCreateTenantRequest represents a request to create a tenant
//
//	@Description Request body for creating a new tenant in the platform
type AdminCreateTenantRequest struct {
	Code         string `json:"code" binding:"required,min=2,max=50" example:"ACME001"`
	Name         string `json:"name" binding:"required,min=2,max=200" example:"Acme Corporation"`
	ShortName    string `json:"short_name,omitempty" binding:"omitempty,max=50" example:"Acme"`
	ContactName  string `json:"contact_name,omitempty" binding:"omitempty,max=100" example:"John Smith"`
	ContactPhone string `json:"contact_phone,omitempty" binding:"omitempty,max=20" example:"+1-555-123-4567"`
	ContactEmail string `json:"contact_email,omitempty" binding:"omitempty,email,max=200" example:"john.smith@acme.com"`
	Address      string `json:"address,omitempty" binding:"omitempty,max=500" example:"123 Business Ave, Suite 100, New York, NY 10001"`
	Plan         string `json:"plan,omitempty" binding:"omitempty,oneof=free basic pro enterprise" example:"pro"`
	TrialDays    int    `json:"trial_days,omitempty" binding:"omitempty,min=0,max=90" example:"14"`
	Notes        string `json:"notes,omitempty" binding:"omitempty,max=1000" example:"Enterprise customer, signed 2-year contract"`
}

// AdminUpdateTenantRequest represents a request to update a tenant
//
//	@Description Request body for updating tenant information. All fields are optional.
type AdminUpdateTenantRequest struct {
	Name         *string `json:"name,omitempty" binding:"omitempty,min=2,max=200" example:"Acme Corporation Inc."`
	ShortName    *string `json:"short_name,omitempty" binding:"omitempty,max=50" example:"Acme Inc"`
	ContactName  *string `json:"contact_name,omitempty" binding:"omitempty,max=100" example:"Jane Doe"`
	ContactPhone *string `json:"contact_phone,omitempty" binding:"omitempty,max=20" example:"+1-555-987-6543"`
	ContactEmail *string `json:"contact_email,omitempty" binding:"omitempty,email,max=200" example:"jane.doe@acme.com"`
	Address      *string `json:"address,omitempty" binding:"omitempty,max=500" example:"456 Enterprise Blvd, Floor 20, New York, NY 10002"`
	Notes        *string `json:"notes,omitempty" binding:"omitempty,max=1000" example:"Upgraded to enterprise plan in Q2 2024"`
}

// SuspendTenantRequest represents a request to suspend a tenant
//
//	@Description Request body for suspending a tenant with optional reason and reactivation schedule
type SuspendTenantRequest struct {
	Reason                string     `json:"reason,omitempty" binding:"omitempty,max=500" example:"Payment overdue for 30+ days"`
	ScheduledReactivateAt *time.Time `json:"scheduled_reactivate_at,omitempty" example:"2024-03-15T00:00:00Z"`
}

// ============================================================================
// Response DTOs
// ============================================================================

// AdminTenantListResponse represents a paginated list of tenants
//
//	@Description Paginated list of tenants with metadata
type AdminTenantListResponse struct {
	Tenants    []AdminTenantResponse `json:"tenants"`
	Total      int64                 `json:"total" example:"156"`
	Page       int                   `json:"page" example:"1"`
	PageSize   int                   `json:"page_size" example:"20"`
	TotalPages int                   `json:"total_pages" example:"8"`
}

// TenantUsageStatsResponse represents tenant usage statistics
//
//	@Description Detailed usage statistics for a specific tenant including quotas
type TenantUsageStatsResponse struct {
	TenantID       uuid.UUID `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantName     string    `json:"tenant_name" example:"Acme Corporation"`
	TenantCode     string    `json:"tenant_code" example:"ACME001"`
	Plan           string    `json:"plan" example:"pro"`
	Status         string    `json:"status" example:"active"`
	UserCount      int64     `json:"user_count" example:"25"`
	ProductCount   int64     `json:"product_count" example:"1500"`
	OrderCount     int64     `json:"order_count" example:"3200"`
	WarehouseCount int64     `json:"warehouse_count" example:"3"`
	StorageUsage   int64     `json:"storage_usage" example:"5368709120"`
	APICallCount   int64     `json:"api_call_count" example:"125000"`
	MaxUsers       int       `json:"max_users" example:"50"`
	MaxProducts    int       `json:"max_products" example:"5000"`
	MaxWarehouses  int       `json:"max_warehouses" example:"10"`
	LastUpdated    time.Time `json:"last_updated" example:"2024-02-01T12:00:00Z"`
}

// PlatformStatsResponse represents platform-wide statistics
//
//	@Description Aggregated statistics across all tenants in the platform
type PlatformStatsResponse struct {
	TotalTenants      int64            `json:"total_tenants" example:"156"`
	ActiveTenants     int64            `json:"active_tenants" example:"120"`
	TrialTenants      int64            `json:"trial_tenants" example:"25"`
	SuspendedTenants  int64            `json:"suspended_tenants" example:"8"`
	InactiveTenants   int64            `json:"inactive_tenants" example:"3"`
	TenantsByPlan     map[string]int64 `json:"tenants_by_plan" example:"free:45,basic:60,pro:40,enterprise:11"`
	TotalUsers        int64            `json:"total_users" example:"2500"`
	TotalProducts     int64            `json:"total_products" example:"125000"`
	TotalOrders       int64            `json:"total_orders" example:"450000"`
	TotalStorageUsage int64            `json:"total_storage_usage" example:"107374182400"`
	LastUpdated       time.Time        `json:"last_updated" example:"2024-02-01T12:00:00Z"`
}

// TenantGrowthTrendResponse represents tenant growth trend data
//
//	@Description Time-series data showing tenant growth over a period
type TenantGrowthTrendResponse struct {
	Period     string                      `json:"period" example:"daily"`
	StartDate  string                      `json:"start_date" example:"2024-01-01"`
	EndDate    string                      `json:"end_date" example:"2024-01-31"`
	DataPoints []TenantGrowthPointResponse `json:"data_points"`
}

// TenantGrowthPointResponse represents a single data point in growth trend
//
//	@Description Single data point representing tenant metrics for a specific date
type TenantGrowthPointResponse struct {
	Date         string `json:"date" example:"2024-01-15"`
	TotalTenants int64  `json:"total_tenants" example:"150"`
	NewTenants   int64  `json:"new_tenants" example:"5"`
	ChurnedCount int64  `json:"churned_count" example:"1"`
}

// AdminAuditLogListResponse represents a paginated list of audit logs
//
//	@Description Paginated list of admin audit logs with metadata
type AdminAuditLogListResponse struct {
	Logs       []AdminAuditLogResponse `json:"logs"`
	Total      int64                   `json:"total" example:"1250"`
	Page       int                     `json:"page" example:"1"`
	PageSize   int                     `json:"page_size" example:"50"`
	TotalPages int                     `json:"total_pages" example:"25"`
}

// AdminAuditLogResponse represents a single audit log entry
//
//	@Description Single audit log entry recording admin action with before/after values
type AdminAuditLogResponse struct {
	ID          uuid.UUID              `json:"id" example:"550e8400-e29b-41d4-a716-446655440099"`
	AdminUserID uuid.UUID              `json:"admin_user_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Action      string                 `json:"action" example:"tenant_create"`
	TargetType  string                 `json:"target_type" example:"tenant"`
	TargetID    *uuid.UUID             `json:"target_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	OldValue    map[string]interface{} `json:"old_value,omitempty"`
	NewValue    map[string]interface{} `json:"new_value,omitempty"`
	IPAddress   string                 `json:"ip_address,omitempty" example:"192.168.1.100"`
	UserAgent   string                 `json:"user_agent,omitempty" example:"Mozilla/5.0 (Windows NT 10.0; Win64; x64)"`
	CreatedAt   time.Time              `json:"created_at" example:"2024-02-01T10:30:00Z"`
}

// MessageResponse represents a simple message response
//
//	@Description Simple response containing a success message
type MessageResponse struct {
	Message string `json:"message" example:"Tenant deleted successfully"`
}
