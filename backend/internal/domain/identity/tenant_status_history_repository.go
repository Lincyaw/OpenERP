package identity

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// TenantStatusHistoryFilter contains filter options for querying status history
type TenantStatusHistoryFilter struct {
	Page       int
	PageSize   int
	ChangeType *TenantStatusChangeType
	StartDate  *time.Time
	EndDate    *time.Time
}

// DefaultTenantStatusHistoryFilter returns a filter with default values
func DefaultTenantStatusHistoryFilter() TenantStatusHistoryFilter {
	return TenantStatusHistoryFilter{
		Page:     1,
		PageSize: 20,
	}
}

// Limit returns the page size (for compatibility)
func (f TenantStatusHistoryFilter) Limit() int {
	if f.PageSize <= 0 {
		return 20
	}
	if f.PageSize > 100 {
		return 100
	}
	return f.PageSize
}

// Offset returns the offset for pagination
func (f TenantStatusHistoryFilter) Offset() int {
	if f.Page <= 0 {
		return 0
	}
	return (f.Page - 1) * f.Limit()
}

// TenantStatusHistoryRepository defines the interface for tenant status history persistence
type TenantStatusHistoryRepository interface {
	// Create creates a new status history record
	Create(ctx context.Context, history *TenantStatusHistory) error

	// FindByTenantID retrieves status history for a tenant with pagination
	FindByTenantID(ctx context.Context, tenantID uuid.UUID, filter TenantStatusHistoryFilter) ([]TenantStatusHistory, int64, error)

	// FindLatestByTenantID retrieves the most recent status change for a tenant
	FindLatestByTenantID(ctx context.Context, tenantID uuid.UUID) (*TenantStatusHistory, error)

	// FindPendingReactivations finds all tenants with scheduled reactivation before the given time
	FindPendingReactivations(ctx context.Context, before time.Time) ([]TenantStatusHistory, error)
}
