package billing

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrInvalidTenantID is returned when a tenant ID is invalid or empty
var ErrInvalidTenantID = errors.New("billing: tenant ID cannot be empty")

// UsageHistory represents a daily snapshot of tenant usage metrics.
// These snapshots are created daily by a scheduled job and used for
// historical trend analysis and reporting.
type UsageHistory struct {
	ID              uuid.UUID      `json:"id"`
	TenantID        uuid.UUID      `json:"tenant_id"`
	SnapshotDate    time.Time      `json:"snapshot_date"`
	UsersCount      int64          `json:"users_count"`
	ProductsCount   int64          `json:"products_count"`
	WarehousesCount int64          `json:"warehouses_count"`
	CustomersCount  int64          `json:"customers_count"`
	SuppliersCount  int64          `json:"suppliers_count"`
	OrdersCount     int64          `json:"orders_count"`
	StorageBytes    int64          `json:"storage_bytes"`
	APICallsCount   int64          `json:"api_calls_count"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
}

// NewUsageHistory creates a new usage history snapshot with validation
func NewUsageHistory(tenantID uuid.UUID, snapshotDate time.Time) (*UsageHistory, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}

	// Normalize snapshot date to start of day (UTC)
	normalizedDate := time.Date(
		snapshotDate.Year(),
		snapshotDate.Month(),
		snapshotDate.Day(),
		0, 0, 0, 0,
		time.UTC,
	)

	return &UsageHistory{
		ID:           uuid.New(),
		TenantID:     tenantID,
		SnapshotDate: normalizedDate,
		Metadata:     make(map[string]any),
		CreatedAt:    time.Now().UTC(),
	}, nil
}

// WithUsersCount sets the users count
func (h *UsageHistory) WithUsersCount(count int64) *UsageHistory {
	if count >= 0 {
		h.UsersCount = count
	}
	return h
}

// WithProductsCount sets the products count
func (h *UsageHistory) WithProductsCount(count int64) *UsageHistory {
	if count >= 0 {
		h.ProductsCount = count
	}
	return h
}

// WithWarehousesCount sets the warehouses count
func (h *UsageHistory) WithWarehousesCount(count int64) *UsageHistory {
	if count >= 0 {
		h.WarehousesCount = count
	}
	return h
}

// WithCustomersCount sets the customers count
func (h *UsageHistory) WithCustomersCount(count int64) *UsageHistory {
	if count >= 0 {
		h.CustomersCount = count
	}
	return h
}

// WithSuppliersCount sets the suppliers count
func (h *UsageHistory) WithSuppliersCount(count int64) *UsageHistory {
	if count >= 0 {
		h.SuppliersCount = count
	}
	return h
}

// WithOrdersCount sets the orders count
func (h *UsageHistory) WithOrdersCount(count int64) *UsageHistory {
	if count >= 0 {
		h.OrdersCount = count
	}
	return h
}

// WithStorageBytes sets the storage bytes
func (h *UsageHistory) WithStorageBytes(bytes int64) *UsageHistory {
	if bytes >= 0 {
		h.StorageBytes = bytes
	}
	return h
}

// WithAPICallsCount sets the API calls count
func (h *UsageHistory) WithAPICallsCount(count int64) *UsageHistory {
	if count >= 0 {
		h.APICallsCount = count
	}
	return h
}

// WithMetadata adds a metadata entry
func (h *UsageHistory) WithMetadata(key string, value any) *UsageHistory {
	if h.Metadata == nil {
		h.Metadata = make(map[string]any)
	}
	h.Metadata[key] = value
	return h
}

// SetCounts sets all counts at once for convenience
// Negative values are ignored (counts remain unchanged)
func (h *UsageHistory) SetCounts(users, products, warehouses, customers, suppliers, orders int64) *UsageHistory {
	if users >= 0 {
		h.UsersCount = users
	}
	if products >= 0 {
		h.ProductsCount = products
	}
	if warehouses >= 0 {
		h.WarehousesCount = warehouses
	}
	if customers >= 0 {
		h.CustomersCount = customers
	}
	if suppliers >= 0 {
		h.SuppliersCount = suppliers
	}
	if orders >= 0 {
		h.OrdersCount = orders
	}
	return h
}

// UsageHistoryFilter defines filtering options for usage history queries
type UsageHistoryFilter struct {
	StartDate *time.Time // Filter snapshots from this date (inclusive)
	EndDate   *time.Time // Filter snapshots until this date (inclusive)
	Page      int        // Page number (1-based)
	PageSize  int        // Number of records per page
}

// DefaultUsageHistoryFilter returns a filter with default values
func DefaultUsageHistoryFilter() UsageHistoryFilter {
	return UsageHistoryFilter{
		Page:     1,
		PageSize: 100,
	}
}

// WithDateRange sets the date range for the filter
func (f UsageHistoryFilter) WithDateRange(start, end time.Time) UsageHistoryFilter {
	f.StartDate = &start
	f.EndDate = &end
	return f
}

// WithPagination sets pagination options
func (f UsageHistoryFilter) WithPagination(page, pageSize int) UsageHistoryFilter {
	f.Page = page
	f.PageSize = pageSize
	return f
}

// UsageHistoryRepository defines the interface for persisting and querying usage history
type UsageHistoryRepository interface {
	// Save persists a new usage history snapshot
	Save(ctx context.Context, history *UsageHistory) error

	// SaveBatch persists multiple usage history snapshots in a single transaction
	SaveBatch(ctx context.Context, histories []*UsageHistory) error

	// Upsert creates or updates a usage history snapshot for a tenant and date
	Upsert(ctx context.Context, history *UsageHistory) error

	// FindByID retrieves a usage history snapshot by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*UsageHistory, error)

	// FindByTenantAndDate retrieves a specific snapshot for a tenant and date
	FindByTenantAndDate(ctx context.Context, tenantID uuid.UUID, date time.Time) (*UsageHistory, error)

	// FindByTenant retrieves all snapshots for a tenant within a date range
	FindByTenant(ctx context.Context, tenantID uuid.UUID, filter UsageHistoryFilter) ([]*UsageHistory, error)

	// FindLatestByTenant retrieves the most recent snapshot for a tenant
	FindLatestByTenant(ctx context.Context, tenantID uuid.UUID) (*UsageHistory, error)

	// CountByTenant counts snapshots for a tenant within a date range
	CountByTenant(ctx context.Context, tenantID uuid.UUID, filter UsageHistoryFilter) (int64, error)

	// DeleteOlderThan removes snapshots older than the specified date (for data retention)
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)

	// DeleteByTenant removes all snapshots for a tenant
	DeleteByTenant(ctx context.Context, tenantID uuid.UUID) error

	// GetAllTenantIDs retrieves all unique tenant IDs that have usage history
	GetAllTenantIDs(ctx context.Context) ([]uuid.UUID, error)
}
