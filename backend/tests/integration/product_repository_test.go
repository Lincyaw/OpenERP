package integration

import (
	"context"
	"os"
	"testing"

	"github.com/erp/backend/internal/domain/catalog"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/persistence"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain runs before any tests and handles cleanup
func TestMain(m *testing.M) {
	code := m.Run()
	CleanupSharedContainer()
	os.Exit(code)
}

// TestProductRepository_Integration tests the ProductRepository against a real PostgreSQL database
func TestProductRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := NewTestDB(t)
	repo := persistence.NewGormProductRepository(testDB.DB)
	ctx := context.Background()
	tenantID := uuid.New()

	// Create tenant first (required for foreign key)
	testDB.CreateTestTenantWithUUID(tenantID)

	t.Run("Save and FindByID", func(t *testing.T) {
		product, err := catalog.NewProduct(tenantID, "PROD-001", "Test Product", "pcs")
		require.NoError(t, err)

		err = repo.Save(ctx, product)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, product.ID)
		require.NoError(t, err)
		assert.Equal(t, product.ID, found.ID)
		assert.Equal(t, product.Code, found.Code)
		assert.Equal(t, product.Name, found.Name)
		assert.Equal(t, product.TenantID, found.TenantID)
	})

	t.Run("FindByIDForTenant", func(t *testing.T) {
		product, err := catalog.NewProduct(tenantID, "PROD-002", "Tenant Product", "kg")
		require.NoError(t, err)

		err = repo.Save(ctx, product)
		require.NoError(t, err)

		// Should find with correct tenant
		found, err := repo.FindByIDForTenant(ctx, tenantID, product.ID)
		require.NoError(t, err)
		assert.Equal(t, product.ID, found.ID)

		// Should not find with different tenant
		otherTenant := uuid.New()
		_, err = repo.FindByIDForTenant(ctx, otherTenant, product.ID)
		assert.ErrorIs(t, err, shared.ErrNotFound)
	})

	t.Run("FindByCode", func(t *testing.T) {
		product, err := catalog.NewProduct(tenantID, "PROD-003", "Code Product", "box")
		require.NoError(t, err)

		err = repo.Save(ctx, product)
		require.NoError(t, err)

		// Should find by code (case-insensitive - code is stored uppercase)
		found, err := repo.FindByCode(ctx, tenantID, "prod-003")
		require.NoError(t, err)
		assert.Equal(t, "PROD-003", found.Code)
	})

	t.Run("FindByBarcode", func(t *testing.T) {
		product, err := catalog.NewProduct(tenantID, "PROD-004", "Barcode Product", "pcs")
		require.NoError(t, err)
		err = product.SetBarcode("1234567890123")
		require.NoError(t, err)

		err = repo.Save(ctx, product)
		require.NoError(t, err)

		found, err := repo.FindByBarcode(ctx, tenantID, "1234567890123")
		require.NoError(t, err)
		assert.Equal(t, product.ID, found.ID)
		assert.Equal(t, "1234567890123", found.Barcode)
	})

	t.Run("FindAllForTenant with pagination", func(t *testing.T) {
		// Create multiple products
		for i := 5; i < 15; i++ {
			product, err := catalog.NewProduct(tenantID, "BULK-PROD-"+string(rune('A'+i)), "Bulk Product "+string(rune('A'+i)), "pcs")
			require.NoError(t, err)
			err = repo.Save(ctx, product)
			require.NoError(t, err)
		}

		// Test pagination
		filter := shared.Filter{
			Page:     1,
			PageSize: 5,
		}
		products, err := repo.FindAllForTenant(ctx, tenantID, filter)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(products), 5)

		// Second page
		filter.Page = 2
		page2Products, err := repo.FindAllForTenant(ctx, tenantID, filter)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(page2Products), 5)
	})

	t.Run("FindByStatus", func(t *testing.T) {
		// Create active product
		activeProduct, err := catalog.NewProduct(tenantID, "STATUS-ACTIVE", "Active Product", "pcs")
		require.NoError(t, err)
		err = repo.Save(ctx, activeProduct)
		require.NoError(t, err)

		// Create inactive product
		inactiveProduct, err := catalog.NewProduct(tenantID, "STATUS-INACTIVE", "Inactive Product", "pcs")
		require.NoError(t, err)
		err = inactiveProduct.Deactivate()
		require.NoError(t, err)
		err = repo.Save(ctx, inactiveProduct)
		require.NoError(t, err)

		// Find active products
		activeProducts, err := repo.FindByStatus(ctx, tenantID, catalog.ProductStatusActive, shared.Filter{})
		require.NoError(t, err)
		assert.True(t, len(activeProducts) >= 1)
		for _, p := range activeProducts {
			assert.Equal(t, catalog.ProductStatusActive, p.Status)
		}

		// Find inactive products
		inactiveProducts, err := repo.FindByStatus(ctx, tenantID, catalog.ProductStatusInactive, shared.Filter{})
		require.NoError(t, err)
		assert.True(t, len(inactiveProducts) >= 1)
		for _, p := range inactiveProducts {
			assert.Equal(t, catalog.ProductStatusInactive, p.Status)
		}
	})

	t.Run("Update product", func(t *testing.T) {
		product, err := catalog.NewProduct(tenantID, "UPDATE-PROD", "Original Name", "pcs")
		require.NoError(t, err)

		err = repo.Save(ctx, product)
		require.NoError(t, err)

		// Update the product
		err = product.Update("Updated Name", "Updated description")
		require.NoError(t, err)

		err = repo.Save(ctx, product)
		require.NoError(t, err)

		// Verify update
		found, err := repo.FindByID(ctx, product.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", found.Name)
		assert.Equal(t, "Updated description", found.Description)
	})

	t.Run("Delete product", func(t *testing.T) {
		product, err := catalog.NewProduct(tenantID, "DELETE-PROD", "To Delete", "pcs")
		require.NoError(t, err)

		err = repo.Save(ctx, product)
		require.NoError(t, err)

		// Delete
		err = repo.Delete(ctx, product.ID)
		require.NoError(t, err)

		// Verify deletion
		_, err = repo.FindByID(ctx, product.ID)
		assert.ErrorIs(t, err, shared.ErrNotFound)
	})

	t.Run("DeleteForTenant", func(t *testing.T) {
		product, err := catalog.NewProduct(tenantID, "DELETE-TENANT-PROD", "To Delete by Tenant", "pcs")
		require.NoError(t, err)

		err = repo.Save(ctx, product)
		require.NoError(t, err)

		// Try to delete with wrong tenant - should fail
		otherTenant := uuid.New()
		err = repo.DeleteForTenant(ctx, otherTenant, product.ID)
		assert.ErrorIs(t, err, shared.ErrNotFound)

		// Delete with correct tenant - should succeed
		err = repo.DeleteForTenant(ctx, tenantID, product.ID)
		require.NoError(t, err)

		_, err = repo.FindByID(ctx, product.ID)
		assert.ErrorIs(t, err, shared.ErrNotFound)
	})

	t.Run("CountForTenant", func(t *testing.T) {
		// Create a separate tenant for counting
		countTenant := uuid.New()
		testDB.CreateTestTenantWithUUID(countTenant)

		for i := range 5 {
			product, err := catalog.NewProduct(countTenant, "COUNT-"+string(rune('A'+i)), "Count Product "+string(rune('A'+i)), "pcs")
			require.NoError(t, err)
			err = repo.Save(ctx, product)
			require.NoError(t, err)
		}

		count, err := repo.CountForTenant(ctx, countTenant, shared.Filter{})
		require.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})

	t.Run("ExistsByCode", func(t *testing.T) {
		product, err := catalog.NewProduct(tenantID, "EXISTS-CODE", "Exists Product", "pcs")
		require.NoError(t, err)
		err = repo.Save(ctx, product)
		require.NoError(t, err)

		exists, err := repo.ExistsByCode(ctx, tenantID, "EXISTS-CODE")
		require.NoError(t, err)
		assert.True(t, exists)

		exists, err = repo.ExistsByCode(ctx, tenantID, "NONEXISTENT-CODE")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("ExistsByBarcode", func(t *testing.T) {
		product, err := catalog.NewProduct(tenantID, "EXISTS-BARCODE", "Barcode Exists", "pcs")
		require.NoError(t, err)
		err = product.SetBarcode("9876543210123")
		require.NoError(t, err)
		err = repo.Save(ctx, product)
		require.NoError(t, err)

		exists, err := repo.ExistsByBarcode(ctx, tenantID, "9876543210123")
		require.NoError(t, err)
		assert.True(t, exists)

		exists, err = repo.ExistsByBarcode(ctx, tenantID, "0000000000000")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("FindByIDs", func(t *testing.T) {
		idsTenant := uuid.New()
		testDB.CreateTestTenantWithUUID(idsTenant)
		var ids []uuid.UUID

		for i := range 3 {
			product, err := catalog.NewProduct(idsTenant, "IDS-"+string(rune('A'+i)), "IDs Product "+string(rune('A'+i)), "pcs")
			require.NoError(t, err)
			err = repo.Save(ctx, product)
			require.NoError(t, err)
			ids = append(ids, product.ID)
		}

		found, err := repo.FindByIDs(ctx, idsTenant, ids)
		require.NoError(t, err)
		assert.Equal(t, 3, len(found))
	})

	t.Run("FindByCodes", func(t *testing.T) {
		codesTenant := uuid.New()
		testDB.CreateTestTenantWithUUID(codesTenant)
		codes := []string{"CODES-A", "CODES-B", "CODES-C"}

		for _, code := range codes {
			product, err := catalog.NewProduct(codesTenant, code, "Codes Product", "pcs")
			require.NoError(t, err)
			err = repo.Save(ctx, product)
			require.NoError(t, err)
		}

		found, err := repo.FindByCodes(ctx, codesTenant, codes)
		require.NoError(t, err)
		assert.Equal(t, 3, len(found))
	})

	t.Run("Search with filter", func(t *testing.T) {
		searchTenant := uuid.New()
		testDB.CreateTestTenantWithUUID(searchTenant)

		// Create products with various prices
		products := []struct {
			code   string
			name   string
			price  decimal.Decimal
			status catalog.ProductStatus
		}{
			{"SEARCH-CHEAP", "Cheap Widget", decimal.NewFromFloat(10.00), catalog.ProductStatusActive},
			{"SEARCH-MID", "Medium Widget", decimal.NewFromFloat(50.00), catalog.ProductStatusActive},
			{"SEARCH-EXPENSIVE", "Expensive Widget", decimal.NewFromFloat(100.00), catalog.ProductStatusInactive},
		}

		for _, p := range products {
			product, err := catalog.NewProduct(searchTenant, p.code, p.name, "pcs")
			require.NoError(t, err)
			product.SellingPrice = p.price
			if p.status == catalog.ProductStatusInactive {
				err = product.Deactivate()
				require.NoError(t, err)
			}
			err = repo.Save(ctx, product)
			require.NoError(t, err)
		}

		// Search by name
		filter := shared.Filter{
			Search: "Widget",
		}
		found, err := repo.FindAllForTenant(ctx, searchTenant, filter)
		require.NoError(t, err)
		assert.Equal(t, 3, len(found))

		// Filter by price range
		filter = shared.Filter{
			Filters: map[string]interface{}{
				"min_price": decimal.NewFromFloat(20.00),
				"max_price": decimal.NewFromFloat(80.00),
			},
		}
		found, err = repo.FindAllForTenant(ctx, searchTenant, filter)
		require.NoError(t, err)
		assert.Equal(t, 1, len(found))
		assert.Equal(t, "SEARCH-MID", found[0].Code)
	})
}

// TestProductRepository_TenantIsolation tests that data is properly isolated between tenants
func TestProductRepository_TenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := NewTestDB(t)
	repo := persistence.NewGormProductRepository(testDB.DB)
	ctx := context.Background()

	tenant1 := uuid.New()
	tenant2 := uuid.New()

	// Create tenants first (required for foreign keys)
	testDB.CreateTestTenantWithUUID(tenant1)
	testDB.CreateTestTenantWithUUID(tenant2)

	// Create products for tenant 1
	for i := range 3 {
		product, err := catalog.NewProduct(tenant1, "T1-PROD-"+string(rune('A'+i)), "Tenant 1 Product", "pcs")
		require.NoError(t, err)
		err = repo.Save(ctx, product)
		require.NoError(t, err)
	}

	// Create products for tenant 2
	for i := range 2 {
		product, err := catalog.NewProduct(tenant2, "T2-PROD-"+string(rune('A'+i)), "Tenant 2 Product", "pcs")
		require.NoError(t, err)
		err = repo.Save(ctx, product)
		require.NoError(t, err)
	}

	// Verify tenant 1 only sees their products
	t1Products, err := repo.FindAllForTenant(ctx, tenant1, shared.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 3, len(t1Products))
	for _, p := range t1Products {
		assert.Equal(t, tenant1, p.TenantID)
	}

	// Verify tenant 2 only sees their products
	t2Products, err := repo.FindAllForTenant(ctx, tenant2, shared.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 2, len(t2Products))
	for _, p := range t2Products {
		assert.Equal(t, tenant2, p.TenantID)
	}

	// Verify count is correct per tenant
	count1, err := repo.CountForTenant(ctx, tenant1, shared.Filter{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), count1)

	count2, err := repo.CountForTenant(ctx, tenant2, shared.Filter{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), count2)
}

// TestProductRepository_OptimisticLocking tests optimistic locking behavior
func TestProductRepository_OptimisticLocking(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := NewTestDB(t)
	repo := persistence.NewGormProductRepository(testDB.DB)
	ctx := context.Background()
	tenantID := uuid.New()

	// Create tenant first (required for foreign key)
	testDB.CreateTestTenantWithUUID(tenantID)

	// Create a product
	product, err := catalog.NewProduct(tenantID, "LOCK-PROD", "Locking Product", "pcs")
	require.NoError(t, err)
	err = repo.Save(ctx, product)
	require.NoError(t, err)

	// Load the same product twice
	instance1, err := repo.FindByID(ctx, product.ID)
	require.NoError(t, err)

	instance2, err := repo.FindByID(ctx, product.ID)
	require.NoError(t, err)

	// Update instance 1
	err = instance1.Update("Updated by Instance 1", "")
	require.NoError(t, err)
	err = repo.Save(ctx, instance1)
	require.NoError(t, err)

	// Verify version was incremented
	updated, err := repo.FindByID(ctx, product.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated by Instance 1", updated.Name)
	assert.Greater(t, updated.Version, instance2.Version)
}
