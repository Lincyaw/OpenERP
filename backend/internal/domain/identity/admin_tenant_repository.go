package identity

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// AdminTenantFilter represents filter options for cross-tenant queries
// This filter is used by super admins to query all tenants across the platform
type AdminTenantFilter struct {
	// Pagination
	Page     int
	PageSize int

	// Sorting
	OrderBy  string
	OrderDir string

	// Search (fuzzy match on name, code, contact_email)
	Search string

	// Filter by status (active, inactive, suspended, trial)
	Status *TenantStatus

	// Filter by plan type (free, basic, pro, enterprise)
	PlanType *TenantPlan

	// Filter by creation date range
	CreatedAfter  *time.Time
	CreatedBefore *time.Time

	// Filter by expiration date range
	ExpiresAfter  *time.Time
	ExpiresBefore *time.Time

	// Filter by trial status
	IsTrialExpiring *bool // true = trial expiring within 7 days
	IsExpired       *bool // true = subscription expired
}

// DefaultAdminTenantFilter returns a filter with default values
func DefaultAdminTenantFilter() AdminTenantFilter {
	return AdminTenantFilter{
		Page:     1,
		PageSize: 20,
		OrderBy:  "created_at",
		OrderDir: "desc",
	}
}

// ToSharedFilter converts AdminTenantFilter to shared.Filter for compatibility
func (f AdminTenantFilter) ToSharedFilter() shared.Filter {
	return shared.Filter{
		Page:     f.Page,
		PageSize: f.PageSize,
		OrderBy:  f.OrderBy,
		OrderDir: f.OrderDir,
		Search:   f.Search,
	}
}

// AdminTenantRepository defines the interface for cross-tenant queries
// This repository bypasses tenant isolation and is only accessible to super admins
// IMPORTANT: All methods in this interface must be protected by SuperAdminMiddleware
type AdminTenantRepository interface {
	// FindAll finds all tenants matching the filter with pagination
	// This method bypasses tenant isolation
	FindAll(ctx context.Context, filter AdminTenantFilter) ([]Tenant, error)

	// FindByID finds a tenant by its ID
	// This method bypasses tenant isolation
	FindByID(ctx context.Context, id uuid.UUID) (*Tenant, error)

	// FindByIDs finds multiple tenants by their IDs
	// This method bypasses tenant isolation
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]Tenant, error)

	// Count counts tenants matching the filter
	// This method bypasses tenant isolation
	Count(ctx context.Context, filter AdminTenantFilter) (int64, error)

	// Search searches tenants by name, code, or contact email with fuzzy matching
	// This method bypasses tenant isolation
	Search(ctx context.Context, query string, filter AdminTenantFilter) ([]Tenant, error)

	// CountByStatus returns the count of tenants grouped by status
	// Useful for dashboard statistics
	CountByStatus(ctx context.Context) (map[TenantStatus]int64, error)

	// CountByPlan returns the count of tenants grouped by plan
	// Useful for dashboard statistics
	CountByPlan(ctx context.Context) (map[TenantPlan]int64, error)

	// FindTrialExpiring finds tenants whose trial is expiring within the given days
	FindTrialExpiring(ctx context.Context, withinDays int) ([]Tenant, error)

	// FindSubscriptionExpiring finds tenants whose subscription is expiring within the given days
	FindSubscriptionExpiring(ctx context.Context, withinDays int) ([]Tenant, error)

	// FindExpired finds tenants whose subscription has expired
	FindExpired(ctx context.Context) ([]Tenant, error)

	// GetStatistics returns aggregated statistics for all tenants
	GetStatistics(ctx context.Context) (*TenantStatistics, error)
}

// TenantStatistics holds aggregated statistics for all tenants
type TenantStatistics struct {
	TotalTenants     int64            `json:"total_tenants"`
	ActiveTenants    int64            `json:"active_tenants"`
	InactiveTenants  int64            `json:"inactive_tenants"`
	SuspendedTenants int64            `json:"suspended_tenants"`
	TrialTenants     int64            `json:"trial_tenants"`
	ByPlan           map[string]int64 `json:"by_plan"`
	TrialExpiring7d  int64            `json:"trial_expiring_7d"`
	SubExpiring30d   int64            `json:"sub_expiring_30d"`
	ExpiredTenants   int64            `json:"expired_tenants"`
}
