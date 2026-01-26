package integration

import (
	"context"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/erp/backend/internal/infrastructure/persistence"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInventoryRepository_Integration tests the InventoryRepository against a real PostgreSQL database
func TestInventoryRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := NewTestDB(t)
	repo := persistence.NewGormInventoryItemRepository(testDB.DB)
	ctx := context.Background()
	tenantID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	// Create tenant and warehouse first (required for foreign keys)
	testDB.CreateTestTenantWithUUID(tenantID)
	testDB.CreateTestWarehouse(tenantID, warehouseID)
	testDB.CreateTestProduct(tenantID, productID)

	t.Run("Save and FindByID", func(t *testing.T) {
		item, err := inventory.NewInventoryItem(tenantID, warehouseID, productID)
		require.NoError(t, err)

		err = repo.Save(ctx, item)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, item.ID)
		require.NoError(t, err)
		assert.Equal(t, item.ID, found.ID)
		assert.Equal(t, item.WarehouseID, found.WarehouseID)
		assert.Equal(t, item.ProductID, found.ProductID)
		assert.Equal(t, item.TenantID, found.TenantID)
	})

	t.Run("FindByWarehouseAndProduct", func(t *testing.T) {
		newWarehouseID := uuid.New()
		testDB.CreateTestWarehouse(tenantID, newWarehouseID)
		newProductID := uuid.New()
		testDB.CreateTestProduct(tenantID, newProductID)

		item, err := inventory.NewInventoryItem(tenantID, newWarehouseID, newProductID)
		require.NoError(t, err)

		err = repo.Save(ctx, item)
		require.NoError(t, err)

		found, err := repo.FindByWarehouseAndProduct(ctx, tenantID, newWarehouseID, newProductID)
		require.NoError(t, err)
		assert.Equal(t, item.ID, found.ID)

		// Should not find with different warehouse
		_, err = repo.FindByWarehouseAndProduct(ctx, tenantID, uuid.New(), newProductID)
		assert.ErrorIs(t, err, shared.ErrNotFound)
	})

	t.Run("Stock increase and update", func(t *testing.T) {
		stockWarehouseID := uuid.New()
		testDB.CreateTestWarehouse(tenantID, stockWarehouseID)
		stockProductID := uuid.New()
		testDB.CreateTestProduct(tenantID, stockProductID)

		item, err := inventory.NewInventoryItem(tenantID, stockWarehouseID, stockProductID)
		require.NoError(t, err)

		err = repo.Save(ctx, item)
		require.NoError(t, err)

		// Increase stock
		unitCost := valueobject.NewMoneyCNY(decimal.NewFromFloat(10.00))
		err = item.IncreaseStock(decimal.NewFromFloat(100), unitCost, nil)
		require.NoError(t, err)

		err = repo.Save(ctx, item)
		require.NoError(t, err)

		// Verify update
		found, err := repo.FindByID(ctx, item.ID)
		require.NoError(t, err)
		assert.True(t, found.AvailableQuantity.Amount().Equal(decimal.NewFromFloat(100)))
		assert.True(t, found.UnitCost.Equal(decimal.NewFromFloat(10.00)))
	})

	t.Run("Stock lock and release", func(t *testing.T) {
		lockWarehouseID := uuid.New()
		testDB.CreateTestWarehouse(tenantID, lockWarehouseID)
		lockProductID := uuid.New()
		testDB.CreateTestProduct(tenantID, lockProductID)

		item, err := inventory.NewInventoryItem(tenantID, lockWarehouseID, lockProductID)
		require.NoError(t, err)

		// First add some stock
		unitCost := valueobject.NewMoneyCNY(decimal.NewFromFloat(10.00))
		err = item.IncreaseStock(decimal.NewFromFloat(100), unitCost, nil)
		require.NoError(t, err)

		err = repo.Save(ctx, item)
		require.NoError(t, err)

		// Lock some quantity
		orderID := uuid.New()
		expireAt := time.Now().Add(24 * time.Hour)
		lock, err := item.LockStock(decimal.NewFromFloat(30), "sales_order", orderID.String(), expireAt)
		require.NoError(t, err)
		require.NotNil(t, lock)

		err = repo.Save(ctx, item)
		require.NoError(t, err)

		// Verify lock persisted - check quantities
		found, err := repo.FindByID(ctx, item.ID)
		require.NoError(t, err)
		assert.True(t, found.AvailableQuantity.Amount().Equal(decimal.NewFromFloat(70)))
		assert.True(t, found.LockedQuantity.Amount().Equal(decimal.NewFromFloat(30)))

		// Release lock using original item (which has the lock reference)
		// Note: Repository doesn't preload Locks association, so we use the original item
		err = item.UnlockStock(lock.ID)
		require.NoError(t, err)

		err = repo.Save(ctx, item)
		require.NoError(t, err)

		// Verify unlock
		found2, err := repo.FindByID(ctx, item.ID)
		require.NoError(t, err)
		assert.True(t, found2.AvailableQuantity.Amount().Equal(decimal.NewFromFloat(100)))
		assert.True(t, found2.LockedQuantity.Amount().Equal(decimal.NewFromFloat(0)))
	})

	t.Run("FindByWarehouse", func(t *testing.T) {
		warehouseForList := uuid.New()
		testDB.CreateTestWarehouse(tenantID, warehouseForList)

		// Create multiple inventory items for the same warehouse
		for i := range 5 {
			prodID := uuid.New()
			testDB.CreateTestProduct(tenantID, prodID)
			item, err := inventory.NewInventoryItem(tenantID, warehouseForList, prodID)
			require.NoError(t, err)

			unitCost := valueobject.NewMoneyCNY(decimal.NewFromFloat(10.00 * float64(i+1)))
			err = item.IncreaseStock(decimal.NewFromFloat(float64((i+1)*10)), unitCost, nil)
			require.NoError(t, err)

			err = repo.Save(ctx, item)
			require.NoError(t, err)
		}

		// Find all items in warehouse
		items, err := repo.FindByWarehouse(ctx, tenantID, warehouseForList, shared.Filter{})
		require.NoError(t, err)
		assert.Equal(t, 5, len(items))
		for _, item := range items {
			assert.Equal(t, warehouseForList, item.WarehouseID)
		}
	})

	t.Run("FindByProduct", func(t *testing.T) {
		productForMultiWarehouse := uuid.New()
		testDB.CreateTestProduct(tenantID, productForMultiWarehouse)

		// Create inventory items in different warehouses
		for i := range 3 {
			wh := uuid.New()
			testDB.CreateTestWarehouse(tenantID, wh)
			item, err := inventory.NewInventoryItem(tenantID, wh, productForMultiWarehouse)
			require.NoError(t, err)

			unitCost := valueobject.NewMoneyCNY(decimal.NewFromFloat(10.00))
			err = item.IncreaseStock(decimal.NewFromFloat(float64((i+1)*20)), unitCost, nil)
			require.NoError(t, err)

			err = repo.Save(ctx, item)
			require.NoError(t, err)
		}

		// Find all inventory for product across warehouses
		items, err := repo.FindByProduct(ctx, tenantID, productForMultiWarehouse, shared.Filter{})
		require.NoError(t, err)
		assert.Equal(t, 3, len(items))
		for _, item := range items {
			assert.Equal(t, productForMultiWarehouse, item.ProductID)
		}
	})

	t.Run("FindBelowMinimum", func(t *testing.T) {
		lowStockTenant := uuid.New()
		testDB.CreateTestTenantWithUUID(lowStockTenant)

		// Create inventory items with different stock levels
		items := []struct {
			minQty  float64
			current float64
		}{
			{10, 5},  // Low stock
			{10, 15}, // OK
			{20, 10}, // Low stock
			{5, 10},  // OK
		}

		for _, data := range items {
			wh := uuid.New()
			testDB.CreateTestWarehouse(lowStockTenant, wh)
			prodID := uuid.New()
			testDB.CreateTestProduct(lowStockTenant, prodID)
			item, err := inventory.NewInventoryItem(lowStockTenant, wh, prodID)
			require.NoError(t, err)

			item.MinQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromFloat(data.minQty))
			unitCost := valueobject.NewMoneyCNY(decimal.NewFromFloat(10.00))
			err = item.IncreaseStock(decimal.NewFromFloat(data.current), unitCost, nil)
			require.NoError(t, err)

			err = repo.Save(ctx, item)
			require.NoError(t, err)
		}

		// Find items with low stock
		lowStockItems, err := repo.FindBelowMinimum(ctx, lowStockTenant, shared.Filter{})
		require.NoError(t, err)
		assert.Equal(t, 2, len(lowStockItems))
	})

	t.Run("Delete inventory item", func(t *testing.T) {
		delWh := uuid.New()
		testDB.CreateTestWarehouse(tenantID, delWh)
		delProd := uuid.New()
		testDB.CreateTestProduct(tenantID, delProd)
		item, err := inventory.NewInventoryItem(tenantID, delWh, delProd)
		require.NoError(t, err)

		err = repo.Save(ctx, item)
		require.NoError(t, err)

		err = repo.Delete(ctx, item.ID)
		require.NoError(t, err)

		_, err = repo.FindByID(ctx, item.ID)
		assert.ErrorIs(t, err, shared.ErrNotFound)
	})

	t.Run("CountByWarehouse", func(t *testing.T) {
		countWarehouse := uuid.New()
		testDB.CreateTestWarehouse(tenantID, countWarehouse)

		for i := range 7 {
			prodID := uuid.New()
			testDB.CreateTestProduct(tenantID, prodID)
			item, err := inventory.NewInventoryItem(tenantID, countWarehouse, prodID)
			require.NoError(t, err)

			unitCost := valueobject.NewMoneyCNY(decimal.NewFromFloat(10.00))
			err = item.IncreaseStock(decimal.NewFromFloat(float64(i+1)*10), unitCost, nil)
			require.NoError(t, err)

			err = repo.Save(ctx, item)
			require.NoError(t, err)
		}

		count, err := repo.CountByWarehouse(ctx, tenantID, countWarehouse)
		require.NoError(t, err)
		assert.Equal(t, int64(7), count)
	})
}

// TestInventoryRepository_TenantIsolation tests that inventory data is isolated between tenants
func TestInventoryRepository_TenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := NewTestDB(t)
	repo := persistence.NewGormInventoryItemRepository(testDB.DB)
	ctx := context.Background()

	tenant1 := uuid.New()
	tenant2 := uuid.New()
	sharedWarehouseID := uuid.New() // Simulating if two tenants accidentally use same warehouse ID

	// Create tenants first (required for foreign keys)
	testDB.CreateTestTenantWithUUID(tenant1)
	testDB.CreateTestTenantWithUUID(tenant2)

	// Create shared warehouse for both tenants
	testDB.CreateTestWarehouse(tenant1, sharedWarehouseID)

	// Create inventory items for tenant 1
	for i := range 3 {
		prodID := uuid.New()
		testDB.CreateTestProduct(tenant1, prodID)
		item, err := inventory.NewInventoryItem(tenant1, sharedWarehouseID, prodID)
		require.NoError(t, err)

		unitCost := valueobject.NewMoneyCNY(decimal.NewFromFloat(10.00))
		err = item.IncreaseStock(decimal.NewFromFloat(float64(i+1)*10), unitCost, nil)
		require.NoError(t, err)

		err = repo.Save(ctx, item)
		require.NoError(t, err)
	}

	// Create inventory items for tenant 2
	// Note: Need to create warehouse for tenant 2 as well
	testDB.CreateTestWarehouse(tenant2, sharedWarehouseID)
	for i := range 2 {
		prodID := uuid.New()
		testDB.CreateTestProduct(tenant2, prodID)
		item, err := inventory.NewInventoryItem(tenant2, sharedWarehouseID, prodID)
		require.NoError(t, err)

		unitCost := valueobject.NewMoneyCNY(decimal.NewFromFloat(20.00))
		err = item.IncreaseStock(decimal.NewFromFloat(float64(i+1)*5), unitCost, nil)
		require.NoError(t, err)

		err = repo.Save(ctx, item)
		require.NoError(t, err)
	}

	// Verify tenant 1 only sees their inventory
	t1Items, err := repo.FindByWarehouse(ctx, tenant1, sharedWarehouseID, shared.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 3, len(t1Items))
	for _, item := range t1Items {
		assert.Equal(t, tenant1, item.TenantID)
	}

	// Verify tenant 2 only sees their inventory
	t2Items, err := repo.FindByWarehouse(ctx, tenant2, sharedWarehouseID, shared.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 2, len(t2Items))
	for _, item := range t2Items {
		assert.Equal(t, tenant2, item.TenantID)
	}
}

// TestInventoryRepository_ConcurrentUpdates tests optimistic locking for concurrent inventory updates
func TestInventoryRepository_ConcurrentUpdates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := NewTestDB(t)
	repo := persistence.NewGormInventoryItemRepository(testDB.DB)
	ctx := context.Background()
	tenantID := uuid.New()

	// Create tenant first (required for foreign key)
	testDB.CreateTestTenantWithUUID(tenantID)

	// Create warehouse (required for foreign key)
	warehouseID := uuid.New()
	testDB.CreateTestWarehouse(tenantID, warehouseID)

	// Create product (required for foreign key)
	productID := uuid.New()
	testDB.CreateTestProduct(tenantID, productID)

	// Create inventory item
	item, err := inventory.NewInventoryItem(tenantID, warehouseID, productID)
	require.NoError(t, err)

	unitCost := valueobject.NewMoneyCNY(decimal.NewFromFloat(10.00))
	err = item.IncreaseStock(decimal.NewFromFloat(100), unitCost, nil)
	require.NoError(t, err)

	err = repo.Save(ctx, item)
	require.NoError(t, err)

	// Simulate concurrent access - load item twice
	item1, err := repo.FindByID(ctx, item.ID)
	require.NoError(t, err)

	item2, err := repo.FindByID(ctx, item.ID)
	require.NoError(t, err)

	// First update succeeds
	expireAt := time.Now().Add(24 * time.Hour)
	_, err = item1.LockStock(decimal.NewFromFloat(20), "sales_order", "order-1", expireAt)
	require.NoError(t, err)
	err = repo.Save(ctx, item1)
	require.NoError(t, err)

	// Verify version was incremented
	updated, err := repo.FindByID(ctx, item.ID)
	require.NoError(t, err)
	assert.Greater(t, updated.Version, item2.Version)
}
