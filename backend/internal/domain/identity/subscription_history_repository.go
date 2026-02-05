package identity

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SubscriptionHistoryFilter defines filter options for querying subscription history
type SubscriptionHistoryFilter struct {
	TenantID   *uuid.UUID
	ChangeType *SubscriptionChangeType
	StartDate  *time.Time
	EndDate    *time.Time
	Page       int
	PageSize   int
	SortBy     string // created_at, effective_at
	SortOrder  string // asc, desc
}

// DefaultSubscriptionHistoryFilter returns default filter values
func DefaultSubscriptionHistoryFilter() SubscriptionHistoryFilter {
	return SubscriptionHistoryFilter{
		Page:      1,
		PageSize:  20,
		SortBy:    "created_at",
		SortOrder: "desc",
	}
}

// Limit returns the page size with a maximum cap
func (f SubscriptionHistoryFilter) Limit() int {
	if f.PageSize <= 0 {
		return 20
	}
	if f.PageSize > 100 {
		return 100
	}
	return f.PageSize
}

// Offset returns the offset for pagination
func (f SubscriptionHistoryFilter) Offset() int {
	if f.Page <= 0 {
		return 0
	}
	return (f.Page - 1) * f.Limit()
}

// SubscriptionHistoryRepository defines persistence operations for subscription history
type SubscriptionHistoryRepository interface {
	// Create creates a new subscription history record
	Create(ctx context.Context, history *SubscriptionHistory) error

	// FindByID finds a subscription history record by ID
	FindByID(ctx context.Context, id uuid.UUID) (*SubscriptionHistory, error)

	// FindByTenantID finds subscription history for a tenant with filtering and pagination
	FindByTenantID(ctx context.Context, tenantID uuid.UUID, filter SubscriptionHistoryFilter) ([]SubscriptionHistory, int64, error)

	// FindPendingDowngrades finds all tenants with scheduled plan downgrades that are due
	// This is used by a background job to apply scheduled downgrades
	FindPendingDowngrades(ctx context.Context, beforeTime time.Time) ([]SubscriptionHistory, error)

	// FindLatestByTenantID finds the most recent subscription history for a tenant
	FindLatestByTenantID(ctx context.Context, tenantID uuid.UUID) (*SubscriptionHistory, error)
}
