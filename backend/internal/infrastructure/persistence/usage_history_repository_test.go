package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/billing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupUsageHistoryTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate the model
	err = db.AutoMigrate(&UsageHistoryModel{})
	require.NoError(t, err)

	return db
}

func TestUsageHistoryRepository_Save(t *testing.T) {
	db := setupUsageHistoryTestDB(t)
	repo := NewUsageHistoryRepository(db)
	ctx := context.Background()

	t.Run("saves new usage history", func(t *testing.T) {
		tenantID := uuid.New()
		history, err := billing.NewUsageHistory(tenantID, time.Now())
		require.NoError(t, err)

		history.WithUsersCount(10).
			WithProductsCount(100).
			WithWarehousesCount(3)

		err = repo.Save(ctx, history)
		require.NoError(t, err)

		// Verify it was saved
		found, err := repo.FindByID(ctx, history.ID)
		require.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, history.ID, found.ID)
		assert.Equal(t, int64(10), found.UsersCount)
		assert.Equal(t, int64(100), found.ProductsCount)
		assert.Equal(t, int64(3), found.WarehousesCount)
	})
}

func TestUsageHistoryRepository_Upsert(t *testing.T) {
	// Skip this test for SQLite as it doesn't support the same ON CONFLICT syntax as PostgreSQL
	// This test should be run against a real PostgreSQL database in integration tests
	t.Skip("Upsert uses PostgreSQL-specific ON CONFLICT syntax, skipping for SQLite")
}

func TestUsageHistoryRepository_FindByTenant(t *testing.T) {
	db := setupUsageHistoryTestDB(t)
	repo := NewUsageHistoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()

	// Create multiple snapshots
	for i := 0; i < 5; i++ {
		date := time.Date(2024, 6, 10+i, 0, 0, 0, 0, time.UTC)
		history, _ := billing.NewUsageHistory(tenantID, date)
		history.WithUsersCount(int64(i + 1))
		err := repo.Save(ctx, history)
		require.NoError(t, err)
	}

	t.Run("returns all snapshots for tenant", func(t *testing.T) {
		filter := billing.DefaultUsageHistoryFilter()
		histories, err := repo.FindByTenant(ctx, tenantID, filter)

		require.NoError(t, err)
		assert.Len(t, histories, 5)
	})

	t.Run("filters by date range", func(t *testing.T) {
		start := time.Date(2024, 6, 11, 0, 0, 0, 0, time.UTC)
		end := time.Date(2024, 6, 13, 0, 0, 0, 0, time.UTC)
		filter := billing.DefaultUsageHistoryFilter().WithDateRange(start, end)

		histories, err := repo.FindByTenant(ctx, tenantID, filter)

		require.NoError(t, err)
		assert.Len(t, histories, 3) // Days 11, 12, 13
	})

	t.Run("applies pagination", func(t *testing.T) {
		filter := billing.DefaultUsageHistoryFilter().WithPagination(1, 2)

		histories, err := repo.FindByTenant(ctx, tenantID, filter)

		require.NoError(t, err)
		assert.Len(t, histories, 2)
	})
}

func TestUsageHistoryRepository_FindLatestByTenant(t *testing.T) {
	db := setupUsageHistoryTestDB(t)
	repo := NewUsageHistoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()

	// Create multiple snapshots
	dates := []time.Time{
		time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 12, 0, 0, 0, 0, time.UTC),
	}

	for _, date := range dates {
		history, _ := billing.NewUsageHistory(tenantID, date)
		err := repo.Save(ctx, history)
		require.NoError(t, err)
	}

	t.Run("returns most recent snapshot", func(t *testing.T) {
		latest, err := repo.FindLatestByTenant(ctx, tenantID)

		require.NoError(t, err)
		assert.NotNil(t, latest)
		assert.Equal(t, time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC), latest.SnapshotDate)
	})

	t.Run("returns nil for tenant with no snapshots", func(t *testing.T) {
		otherTenantID := uuid.New()
		latest, err := repo.FindLatestByTenant(ctx, otherTenantID)

		require.NoError(t, err)
		assert.Nil(t, latest)
	})
}

func TestUsageHistoryRepository_DeleteOlderThan(t *testing.T) {
	db := setupUsageHistoryTestDB(t)
	repo := NewUsageHistoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()

	// Create snapshots with different dates
	dates := []time.Time{
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),  // Old
		time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),  // Old
		time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),  // Recent
		time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC), // Recent
	}

	for _, date := range dates {
		history, _ := billing.NewUsageHistory(tenantID, date)
		err := repo.Save(ctx, history)
		require.NoError(t, err)
	}

	t.Run("deletes old snapshots", func(t *testing.T) {
		cutoff := time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)
		deleted, err := repo.DeleteOlderThan(ctx, cutoff)

		require.NoError(t, err)
		assert.Equal(t, int64(2), deleted)

		// Verify remaining
		filter := billing.DefaultUsageHistoryFilter()
		remaining, err := repo.FindByTenant(ctx, tenantID, filter)
		require.NoError(t, err)
		assert.Len(t, remaining, 2)
	})
}

func TestUsageHistoryRepository_CountByTenant(t *testing.T) {
	db := setupUsageHistoryTestDB(t)
	repo := NewUsageHistoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()

	// Create 5 snapshots
	for i := 0; i < 5; i++ {
		date := time.Date(2024, 6, 10+i, 0, 0, 0, 0, time.UTC)
		history, _ := billing.NewUsageHistory(tenantID, date)
		err := repo.Save(ctx, history)
		require.NoError(t, err)
	}

	t.Run("counts all snapshots", func(t *testing.T) {
		filter := billing.DefaultUsageHistoryFilter()
		count, err := repo.CountByTenant(ctx, tenantID, filter)

		require.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})

	t.Run("counts with date filter", func(t *testing.T) {
		start := time.Date(2024, 6, 12, 0, 0, 0, 0, time.UTC)
		end := time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC)
		filter := billing.DefaultUsageHistoryFilter().WithDateRange(start, end)

		count, err := repo.CountByTenant(ctx, tenantID, filter)

		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})
}

func TestUsageHistoryModel_ToEntity(t *testing.T) {
	model := &UsageHistoryModel{
		ID:              uuid.New(),
		TenantID:        uuid.New(),
		SnapshotDate:    time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
		UsersCount:      10,
		ProductsCount:   100,
		WarehousesCount: 3,
		CustomersCount:  50,
		SuppliersCount:  20,
		OrdersCount:     500,
		StorageBytes:    1024 * 1024,
		APICallsCount:   10000,
		Metadata:        map[string]any{"key": "value"},
		CreatedAt:       time.Now(),
	}

	entity := model.ToEntity()

	assert.Equal(t, model.ID, entity.ID)
	assert.Equal(t, model.TenantID, entity.TenantID)
	assert.Equal(t, model.SnapshotDate, entity.SnapshotDate)
	assert.Equal(t, model.UsersCount, entity.UsersCount)
	assert.Equal(t, model.ProductsCount, entity.ProductsCount)
	assert.Equal(t, model.WarehousesCount, entity.WarehousesCount)
	assert.Equal(t, model.CustomersCount, entity.CustomersCount)
	assert.Equal(t, model.SuppliersCount, entity.SuppliersCount)
	assert.Equal(t, model.OrdersCount, entity.OrdersCount)
	assert.Equal(t, model.StorageBytes, entity.StorageBytes)
	assert.Equal(t, model.APICallsCount, entity.APICallsCount)
	assert.Equal(t, "value", entity.Metadata["key"])
}
