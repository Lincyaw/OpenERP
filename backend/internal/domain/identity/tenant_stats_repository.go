package identity

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// TenantUsageStats represents usage statistics for a single tenant
type TenantUsageStats struct {
	TenantID       uuid.UUID `json:"tenant_id"`
	UserCount      int64     `json:"user_count"`
	ProductCount   int64     `json:"product_count"`
	OrderCount     int64     `json:"order_count"` // Sales + Purchase orders
	WarehouseCount int64     `json:"warehouse_count"`
	StorageUsage   int64     `json:"storage_usage"`  // In bytes
	APICallCount   int64     `json:"api_call_count"` // Last 30 days
	LastUpdated    time.Time `json:"last_updated"`
}

// PlatformStats represents platform-wide statistics
type PlatformStats struct {
	TotalTenants      int64            `json:"total_tenants"`
	ActiveTenants     int64            `json:"active_tenants"`
	TrialTenants      int64            `json:"trial_tenants"`
	SuspendedTenants  int64            `json:"suspended_tenants"`
	InactiveTenants   int64            `json:"inactive_tenants"`
	TenantsByPlan     map[string]int64 `json:"tenants_by_plan"`
	TotalUsers        int64            `json:"total_users"`
	TotalProducts     int64            `json:"total_products"`
	TotalOrders       int64            `json:"total_orders"`
	TotalStorageUsage int64            `json:"total_storage_usage"` // In bytes
	LastUpdated       time.Time        `json:"last_updated"`
}

// TenantGrowthPoint represents a single data point in growth trend
type TenantGrowthPoint struct {
	Date         time.Time `json:"date"`
	TotalTenants int64     `json:"total_tenants"`
	NewTenants   int64     `json:"new_tenants"`
	ChurnedCount int64     `json:"churned_count"` // Tenants that became inactive
}

// TenantGrowthTrend represents tenant growth over time
type TenantGrowthTrend struct {
	Period     string              `json:"period"` // "daily", "weekly", "monthly"
	StartDate  time.Time           `json:"start_date"`
	EndDate    time.Time           `json:"end_date"`
	DataPoints []TenantGrowthPoint `json:"data_points"`
}

// TenantStatsRepository defines the interface for cross-tenant statistics queries
// This repository bypasses tenant isolation and is only accessible to super admins
// IMPORTANT: All methods in this interface must be protected by SuperAdminMiddleware
type TenantStatsRepository interface {
	// GetTenantUsageStats retrieves usage statistics for a specific tenant
	GetTenantUsageStats(ctx context.Context, tenantID uuid.UUID) (*TenantUsageStats, error)

	// GetPlatformStats retrieves platform-wide statistics
	GetPlatformStats(ctx context.Context) (*PlatformStats, error)

	// GetTenantGrowthTrend retrieves tenant growth trend for a period
	// period: "daily", "weekly", "monthly"
	// days: number of days to look back (e.g., 30, 90, 365)
	GetTenantGrowthTrend(ctx context.Context, period string, days int) (*TenantGrowthTrend, error)

	// CountUsersByTenant counts users for a specific tenant
	CountUsersByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error)

	// CountProductsByTenant counts products for a specific tenant
	CountProductsByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error)

	// CountOrdersByTenant counts orders (sales + purchase) for a specific tenant
	CountOrdersByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error)

	// CountWarehousesByTenant counts warehouses for a specific tenant
	CountWarehousesByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error)

	// GetStorageUsageByTenant gets storage usage in bytes for a specific tenant
	GetStorageUsageByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error)

	// GetAPICallCountByTenant gets API call count for a specific tenant (last 30 days)
	GetAPICallCountByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error)

	// CountTotalUsers counts all users across all tenants
	CountTotalUsers(ctx context.Context) (int64, error)

	// CountTotalProducts counts all products across all tenants
	CountTotalProducts(ctx context.Context) (int64, error)

	// CountTotalOrders counts all orders across all tenants
	CountTotalOrders(ctx context.Context) (int64, error)

	// GetTotalStorageUsage gets total storage usage across all tenants
	GetTotalStorageUsage(ctx context.Context) (int64, error)

	// CountNewTenantsByDateRange counts new tenants created within a date range
	CountNewTenantsByDateRange(ctx context.Context, start, end time.Time) (int64, error)

	// CountChurnedTenantsByDateRange counts tenants that became inactive within a date range
	CountChurnedTenantsByDateRange(ctx context.Context, start, end time.Time) (int64, error)

	// CountTenantsByDateRange counts total tenants as of a specific date
	CountTenantsByDateRange(ctx context.Context, asOf time.Time) (int64, error)
}
