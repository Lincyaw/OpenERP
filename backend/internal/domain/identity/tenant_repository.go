package identity

import (
	"context"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// TenantRepository defines the interface for tenant persistence
type TenantRepository interface {
	// FindByID finds a tenant by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*Tenant, error)

	// FindByCode finds a tenant by its unique code
	FindByCode(ctx context.Context, code string) (*Tenant, error)

	// FindByDomain finds a tenant by its custom domain
	FindByDomain(ctx context.Context, domain string) (*Tenant, error)

	// FindAll finds all tenants matching the filter
	FindAll(ctx context.Context, filter shared.Filter) ([]Tenant, error)

	// FindByStatus finds tenants by status
	FindByStatus(ctx context.Context, status TenantStatus, filter shared.Filter) ([]Tenant, error)

	// FindByPlan finds tenants by subscription plan
	FindByPlan(ctx context.Context, plan TenantPlan, filter shared.Filter) ([]Tenant, error)

	// FindActive finds all active tenants
	FindActive(ctx context.Context, filter shared.Filter) ([]Tenant, error)

	// FindTrialExpiring finds tenants whose trial is expiring within the given days
	FindTrialExpiring(ctx context.Context, withinDays int) ([]Tenant, error)

	// FindSubscriptionExpiring finds tenants whose subscription is expiring within the given days
	FindSubscriptionExpiring(ctx context.Context, withinDays int) ([]Tenant, error)

	// FindByIDs finds multiple tenants by their IDs
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]Tenant, error)

	// Save creates or updates a tenant
	Save(ctx context.Context, tenant *Tenant) error

	// Delete deletes a tenant
	Delete(ctx context.Context, id uuid.UUID) error

	// Count counts tenants matching the filter
	Count(ctx context.Context, filter shared.Filter) (int64, error)

	// CountByStatus counts tenants by status
	CountByStatus(ctx context.Context, status TenantStatus) (int64, error)

	// CountByPlan counts tenants by plan
	CountByPlan(ctx context.Context, plan TenantPlan) (int64, error)

	// ExistsByCode checks if a tenant with the given code exists
	ExistsByCode(ctx context.Context, code string) (bool, error)

	// ExistsByDomain checks if a tenant with the given domain exists
	ExistsByDomain(ctx context.Context, domain string) (bool, error)
}
