package billing

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUsageHistory(t *testing.T) {
	t.Run("creates valid usage history", func(t *testing.T) {
		tenantID := uuid.New()
		snapshotDate := time.Now()

		history, err := NewUsageHistory(tenantID, snapshotDate)

		require.NoError(t, err)
		assert.NotNil(t, history)
		assert.NotEqual(t, uuid.Nil, history.ID)
		assert.Equal(t, tenantID, history.TenantID)
		assert.Equal(t, int64(0), history.UsersCount)
		assert.Equal(t, int64(0), history.ProductsCount)
		assert.Equal(t, int64(0), history.WarehousesCount)
		assert.NotNil(t, history.Metadata)
	})

	t.Run("normalizes snapshot date to start of day UTC", func(t *testing.T) {
		tenantID := uuid.New()
		// Create a date with time component
		snapshotDate := time.Date(2024, 6, 15, 14, 30, 45, 123456789, time.Local)

		history, err := NewUsageHistory(tenantID, snapshotDate)

		require.NoError(t, err)
		// Should be normalized to start of day in UTC
		assert.Equal(t, 0, history.SnapshotDate.Hour())
		assert.Equal(t, 0, history.SnapshotDate.Minute())
		assert.Equal(t, 0, history.SnapshotDate.Second())
		assert.Equal(t, 0, history.SnapshotDate.Nanosecond())
		assert.Equal(t, time.UTC, history.SnapshotDate.Location())
	})

	t.Run("returns error for nil tenant ID", func(t *testing.T) {
		history, err := NewUsageHistory(uuid.Nil, time.Now())

		assert.Error(t, err)
		assert.Nil(t, history)
		assert.Equal(t, ErrInvalidTenantID, err)
	})
}

func TestUsageHistory_WithMethods(t *testing.T) {
	tenantID := uuid.New()
	history, _ := NewUsageHistory(tenantID, time.Now())

	t.Run("WithUsersCount sets users count", func(t *testing.T) {
		result := history.WithUsersCount(10)
		assert.Equal(t, int64(10), history.UsersCount)
		assert.Same(t, history, result) // Returns same instance for chaining
	})

	t.Run("WithUsersCount ignores negative values", func(t *testing.T) {
		history.UsersCount = 5
		history.WithUsersCount(-1)
		assert.Equal(t, int64(5), history.UsersCount)
	})

	t.Run("WithProductsCount sets products count", func(t *testing.T) {
		history.WithProductsCount(100)
		assert.Equal(t, int64(100), history.ProductsCount)
	})

	t.Run("WithWarehousesCount sets warehouses count", func(t *testing.T) {
		history.WithWarehousesCount(3)
		assert.Equal(t, int64(3), history.WarehousesCount)
	})

	t.Run("WithCustomersCount sets customers count", func(t *testing.T) {
		history.WithCustomersCount(50)
		assert.Equal(t, int64(50), history.CustomersCount)
	})

	t.Run("WithSuppliersCount sets suppliers count", func(t *testing.T) {
		history.WithSuppliersCount(20)
		assert.Equal(t, int64(20), history.SuppliersCount)
	})

	t.Run("WithOrdersCount sets orders count", func(t *testing.T) {
		history.WithOrdersCount(500)
		assert.Equal(t, int64(500), history.OrdersCount)
	})

	t.Run("WithStorageBytes sets storage bytes", func(t *testing.T) {
		history.WithStorageBytes(1024 * 1024)
		assert.Equal(t, int64(1024*1024), history.StorageBytes)
	})

	t.Run("WithAPICallsCount sets API calls count", func(t *testing.T) {
		history.WithAPICallsCount(10000)
		assert.Equal(t, int64(10000), history.APICallsCount)
	})

	t.Run("WithMetadata adds metadata entry", func(t *testing.T) {
		history.WithMetadata("custom_key", "custom_value")
		assert.Equal(t, "custom_value", history.Metadata["custom_key"])
	})
}

func TestUsageHistory_SetCounts(t *testing.T) {
	tenantID := uuid.New()
	history, _ := NewUsageHistory(tenantID, time.Now())

	result := history.SetCounts(10, 100, 3, 50, 20, 500)

	assert.Same(t, history, result)
	assert.Equal(t, int64(10), history.UsersCount)
	assert.Equal(t, int64(100), history.ProductsCount)
	assert.Equal(t, int64(3), history.WarehousesCount)
	assert.Equal(t, int64(50), history.CustomersCount)
	assert.Equal(t, int64(20), history.SuppliersCount)
	assert.Equal(t, int64(500), history.OrdersCount)
}

func TestUsageHistoryFilter(t *testing.T) {
	t.Run("DefaultUsageHistoryFilter returns correct defaults", func(t *testing.T) {
		filter := DefaultUsageHistoryFilter()

		assert.Equal(t, 1, filter.Page)
		assert.Equal(t, 100, filter.PageSize)
		assert.Nil(t, filter.StartDate)
		assert.Nil(t, filter.EndDate)
	})

	t.Run("WithDateRange sets date range", func(t *testing.T) {
		filter := DefaultUsageHistoryFilter()
		start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

		result := filter.WithDateRange(start, end)

		assert.NotNil(t, result.StartDate)
		assert.NotNil(t, result.EndDate)
		assert.Equal(t, start, *result.StartDate)
		assert.Equal(t, end, *result.EndDate)
	})

	t.Run("WithPagination sets pagination", func(t *testing.T) {
		filter := DefaultUsageHistoryFilter()

		result := filter.WithPagination(2, 50)

		assert.Equal(t, 2, result.Page)
		assert.Equal(t, 50, result.PageSize)
	})
}
