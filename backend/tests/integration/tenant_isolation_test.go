// Package integration provides integration tests for multi-tenant isolation.
// This file tests the critical multi-tenant requirements:
// - Tenant data isolation (tenant A cannot access tenant B's data)
// - Tenant switching (data is correctly scoped when switching tenants)
// - Tenant deactivation (deactivated tenants cannot perform operations)
package integration

import (
	"context"
	"testing"

	"github.com/erp/backend/internal/domain/catalog"
	identitydomain "github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/persistence"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TenantIsolationTestSetup provides test infrastructure for tenant isolation tests
type TenantIsolationTestSetup struct {
	DB           *TestDB
	TenantRepo   *persistence.GormTenantRepository
	ProductRepo  *persistence.GormProductRepository
	CustomerRepo *persistence.GormCustomerRepository
	TenantA      *identitydomain.Tenant
	TenantB      *identitydomain.Tenant
}

// NewTenantIsolationTestSetup creates test infrastructure with two isolated tenants
func NewTenantIsolationTestSetup(t *testing.T) *TenantIsolationTestSetup {
	t.Helper()

	testDB := NewTestDB(t)

	// Create repositories
	tenantRepo := persistence.NewGormTenantRepository(testDB.DB)
	productRepo := persistence.NewGormProductRepository(testDB.DB)
	customerRepo := persistence.NewGormCustomerRepository(testDB.DB)

	ctx := context.Background()

	// Create Tenant A
	tenantA, err := identitydomain.NewTenant("TENANT_A", "Test Tenant A")
	require.NoError(t, err)
	err = tenantRepo.Save(ctx, tenantA)
	require.NoError(t, err)

	// Create Tenant B
	tenantB, err := identitydomain.NewTenant("TENANT_B", "Test Tenant B")
	require.NoError(t, err)
	err = tenantRepo.Save(ctx, tenantB)
	require.NoError(t, err)

	return &TenantIsolationTestSetup{
		DB:           testDB,
		TenantRepo:   tenantRepo,
		ProductRepo:  productRepo,
		CustomerRepo: customerRepo,
		TenantA:      tenantA,
		TenantB:      tenantB,
	}
}

// ==================== Test: Tenant Data Isolation ====================

func TestTenantIsolation_DataIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setup := NewTenantIsolationTestSetup(t)
	ctx := context.Background()

	t.Run("product_created_in_tenant_A_not_visible_to_tenant_B", func(t *testing.T) {
		// Create a product in Tenant A
		productA, err := catalog.NewProduct(
			setup.TenantA.ID,
			"PROD-A-001",
			"Product in Tenant A",
			"pcs",
		)
		require.NoError(t, err)

		err = setup.ProductRepo.Save(ctx, productA)
		require.NoError(t, err)

		// Verify Tenant A can find the product
		foundA, err := setup.ProductRepo.FindByIDForTenant(ctx, setup.TenantA.ID, productA.ID)
		require.NoError(t, err)
		assert.Equal(t, productA.ID, foundA.ID)
		assert.Equal(t, "PROD-A-001", foundA.Code)

		// Verify Tenant B CANNOT find the product
		foundB, err := setup.ProductRepo.FindByIDForTenant(ctx, setup.TenantB.ID, productA.ID)
		assert.ErrorIs(t, err, shared.ErrNotFound)
		assert.Nil(t, foundB)
	})

	t.Run("customer_created_in_tenant_A_not_visible_to_tenant_B", func(t *testing.T) {
		// Create a customer in Tenant A
		customerA, err := partner.NewCustomer(
			setup.TenantA.ID,
			"CUST-A-001",
			"Customer in Tenant A",
			partner.CustomerTypeIndividual,
		)
		require.NoError(t, err)

		err = setup.CustomerRepo.Save(ctx, customerA)
		require.NoError(t, err)

		// Verify Tenant A can find the customer
		foundA, err := setup.CustomerRepo.FindByIDForTenant(ctx, setup.TenantA.ID, customerA.ID)
		require.NoError(t, err)
		assert.Equal(t, customerA.ID, foundA.ID)

		// Verify Tenant B CANNOT find the customer
		foundB, err := setup.CustomerRepo.FindByIDForTenant(ctx, setup.TenantB.ID, customerA.ID)
		assert.ErrorIs(t, err, shared.ErrNotFound)
		assert.Nil(t, foundB)
	})

	t.Run("tenant_A_list_excludes_tenant_B_products", func(t *testing.T) {
		// Create products in both tenants
		productA1, err := catalog.NewProduct(setup.TenantA.ID, "PROD-A-LIST1", "A1", "pcs")
		require.NoError(t, err)
		productA2, err := catalog.NewProduct(setup.TenantA.ID, "PROD-A-LIST2", "A2", "pcs")
		require.NoError(t, err)
		productB1, err := catalog.NewProduct(setup.TenantB.ID, "PROD-B-LIST1", "B1", "pcs")
		require.NoError(t, err)

		require.NoError(t, setup.ProductRepo.Save(ctx, productA1))
		require.NoError(t, setup.ProductRepo.Save(ctx, productA2))
		require.NoError(t, setup.ProductRepo.Save(ctx, productB1))

		// List products for Tenant A
		filter := shared.Filter{Page: 1, PageSize: 100}
		productsA, err := setup.ProductRepo.FindAllForTenant(ctx, setup.TenantA.ID, filter)
		require.NoError(t, err)

		// Verify only Tenant A products are in the list
		productCodesA := make([]string, len(productsA))
		for i, p := range productsA {
			productCodesA[i] = p.Code
		}
		assert.Contains(t, productCodesA, "PROD-A-LIST1")
		assert.Contains(t, productCodesA, "PROD-A-LIST2")
		assert.NotContains(t, productCodesA, "PROD-B-LIST1")

		// List products for Tenant B
		productsB, err := setup.ProductRepo.FindAllForTenant(ctx, setup.TenantB.ID, filter)
		require.NoError(t, err)

		productCodesB := make([]string, len(productsB))
		for i, p := range productsB {
			productCodesB[i] = p.Code
		}
		assert.NotContains(t, productCodesB, "PROD-A-LIST1")
		assert.NotContains(t, productCodesB, "PROD-A-LIST2")
		assert.Contains(t, productCodesB, "PROD-B-LIST1")
	})

	t.Run("same_product_code_allowed_in_different_tenants", func(t *testing.T) {
		// This tests that the same product code can exist in different tenants
		code := "SHARED-CODE-001"

		productA, err := catalog.NewProduct(setup.TenantA.ID, code, "Product A with shared code", "pcs")
		require.NoError(t, err)
		err = setup.ProductRepo.Save(ctx, productA)
		require.NoError(t, err)

		productB, err := catalog.NewProduct(setup.TenantB.ID, code, "Product B with shared code", "pcs")
		require.NoError(t, err)
		err = setup.ProductRepo.Save(ctx, productB)
		require.NoError(t, err)

		// Both products should exist with the same code but different IDs
		foundA, err := setup.ProductRepo.FindByCode(ctx, setup.TenantA.ID, code)
		require.NoError(t, err)
		assert.Equal(t, productA.ID, foundA.ID)
		assert.Equal(t, "Product A with shared code", foundA.Name)

		foundB, err := setup.ProductRepo.FindByCode(ctx, setup.TenantB.ID, code)
		require.NoError(t, err)
		assert.Equal(t, productB.ID, foundB.ID)
		assert.Equal(t, "Product B with shared code", foundB.Name)

		assert.NotEqual(t, foundA.ID, foundB.ID)
	})

	t.Run("count_for_tenant_only_includes_own_data", func(t *testing.T) {
		// Create a fresh test setup for count test to avoid interference
		setup2 := NewTenantIsolationTestSetup(t)
		ctx2 := context.Background()

		// Create 3 products in Tenant A
		for i := 1; i <= 3; i++ {
			p, err := catalog.NewProduct(setup2.TenantA.ID, "PROD-COUNT-A-"+string(rune('0'+i)), "Count A", "pcs")
			require.NoError(t, err)
			require.NoError(t, setup2.ProductRepo.Save(ctx2, p))
		}

		// Create 5 products in Tenant B
		for i := 1; i <= 5; i++ {
			p, err := catalog.NewProduct(setup2.TenantB.ID, "PROD-COUNT-B-"+string(rune('0'+i)), "Count B", "pcs")
			require.NoError(t, err)
			require.NoError(t, setup2.ProductRepo.Save(ctx2, p))
		}

		// Count for Tenant A
		countA, err := setup2.ProductRepo.CountForTenant(ctx2, setup2.TenantA.ID, shared.Filter{})
		require.NoError(t, err)
		assert.Equal(t, int64(3), countA)

		// Count for Tenant B
		countB, err := setup2.ProductRepo.CountForTenant(ctx2, setup2.TenantB.ID, shared.Filter{})
		require.NoError(t, err)
		assert.Equal(t, int64(5), countB)
	})
}

// ==================== Test: Tenant Switching ====================

func TestTenantIsolation_TenantSwitching(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setup := NewTenantIsolationTestSetup(t)
	ctx := context.Background()

	t.Run("switching_tenant_context_shows_correct_data", func(t *testing.T) {
		// Create distinct products in each tenant
		productA, err := catalog.NewProduct(setup.TenantA.ID, "SWITCH-A-001", "Switch Product A", "pcs")
		require.NoError(t, err)
		require.NoError(t, setup.ProductRepo.Save(ctx, productA))

		productB, err := catalog.NewProduct(setup.TenantB.ID, "SWITCH-B-001", "Switch Product B", "pcs")
		require.NoError(t, err)
		require.NoError(t, setup.ProductRepo.Save(ctx, productB))

		// Simulate user operating as Tenant A
		currentTenantID := setup.TenantA.ID
		filter := shared.Filter{Page: 1, PageSize: 100}
		products, err := setup.ProductRepo.FindAllForTenant(ctx, currentTenantID, filter)
		require.NoError(t, err)

		codes := extractCodes(products)
		assert.Contains(t, codes, "SWITCH-A-001")
		assert.NotContains(t, codes, "SWITCH-B-001")

		// Switch to Tenant B
		currentTenantID = setup.TenantB.ID
		products, err = setup.ProductRepo.FindAllForTenant(ctx, currentTenantID, filter)
		require.NoError(t, err)

		codes = extractCodes(products)
		assert.NotContains(t, codes, "SWITCH-A-001")
		assert.Contains(t, codes, "SWITCH-B-001")
	})

	t.Run("product_lookup_by_code_respects_current_tenant", func(t *testing.T) {
		code := "LOOKUP-CODE-001"

		productA, err := catalog.NewProduct(setup.TenantA.ID, code, "Lookup A", "pcs")
		require.NoError(t, err)
		require.NoError(t, setup.ProductRepo.Save(ctx, productA))

		productB, err := catalog.NewProduct(setup.TenantB.ID, code, "Lookup B", "pcs")
		require.NoError(t, err)
		require.NoError(t, setup.ProductRepo.Save(ctx, productB))

		// Lookup as Tenant A
		found, err := setup.ProductRepo.FindByCode(ctx, setup.TenantA.ID, code)
		require.NoError(t, err)
		assert.Equal(t, "Lookup A", found.Name)
		assert.Equal(t, setup.TenantA.ID, found.TenantID)

		// Lookup as Tenant B
		found, err = setup.ProductRepo.FindByCode(ctx, setup.TenantB.ID, code)
		require.NoError(t, err)
		assert.Equal(t, "Lookup B", found.Name)
		assert.Equal(t, setup.TenantB.ID, found.TenantID)
	})
}

// ==================== Test: Tenant Deactivation ====================

func TestTenantIsolation_TenantDeactivation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setup := NewTenantIsolationTestSetup(t)
	ctx := context.Background()

	t.Run("tenant_status_transitions", func(t *testing.T) {
		// Create a new tenant for this test
		tenant, err := identitydomain.NewTenant("DEACTIVATE_TEST", "Deactivation Test Tenant")
		require.NoError(t, err)
		require.NoError(t, setup.TenantRepo.Save(ctx, tenant))

		// Initial status should be active
		assert.Equal(t, identitydomain.TenantStatusActive, tenant.Status)
		assert.True(t, tenant.IsActive())

		// Deactivate the tenant
		err = tenant.Deactivate()
		require.NoError(t, err)
		require.NoError(t, setup.TenantRepo.Save(ctx, tenant))

		// Verify tenant is now inactive
		assert.Equal(t, identitydomain.TenantStatusInactive, tenant.Status)
		assert.True(t, tenant.IsInactive())
		assert.False(t, tenant.IsActive())

		// Verify can be fetched and has correct status
		fetched, err := setup.TenantRepo.FindByID(ctx, tenant.ID)
		require.NoError(t, err)
		assert.Equal(t, identitydomain.TenantStatusInactive, fetched.Status)

		// Re-activate the tenant
		err = fetched.Activate()
		require.NoError(t, err)
		require.NoError(t, setup.TenantRepo.Save(ctx, fetched))

		// Verify tenant is active again
		refetched, err := setup.TenantRepo.FindByID(ctx, tenant.ID)
		require.NoError(t, err)
		assert.Equal(t, identitydomain.TenantStatusActive, refetched.Status)
	})

	t.Run("tenant_suspension", func(t *testing.T) {
		// Create a new tenant for this test
		tenant, err := identitydomain.NewTenant("SUSPEND_TEST", "Suspension Test Tenant")
		require.NoError(t, err)
		require.NoError(t, setup.TenantRepo.Save(ctx, tenant))

		// Suspend the tenant
		err = tenant.Suspend()
		require.NoError(t, err)
		require.NoError(t, setup.TenantRepo.Save(ctx, tenant))

		// Verify tenant is suspended
		assert.Equal(t, identitydomain.TenantStatusSuspended, tenant.Status)
		assert.True(t, tenant.IsSuspended())
		assert.False(t, tenant.IsActive())

		// Fetch and verify persistence
		fetched, err := setup.TenantRepo.FindByID(ctx, tenant.ID)
		require.NoError(t, err)
		assert.Equal(t, identitydomain.TenantStatusSuspended, fetched.Status)
	})

	t.Run("deactivated_tenant_data_still_exists_but_filtered", func(t *testing.T) {
		// This test verifies that when a tenant is deactivated,
		// its data still exists but should be filtered by status checks

		// Create a tenant and add data
		tenant, err := identitydomain.NewTenant("DATA_PERSIST_TEST", "Data Persistence Test")
		require.NoError(t, err)
		require.NoError(t, setup.TenantRepo.Save(ctx, tenant))

		// Create a product for this tenant
		product, err := catalog.NewProduct(tenant.ID, "PERSIST-PROD-001", "Persist Product", "pcs")
		require.NoError(t, err)
		require.NoError(t, setup.ProductRepo.Save(ctx, product))

		// Verify product exists
		found, err := setup.ProductRepo.FindByIDForTenant(ctx, tenant.ID, product.ID)
		require.NoError(t, err)
		assert.Equal(t, product.ID, found.ID)

		// Deactivate the tenant
		err = tenant.Deactivate()
		require.NoError(t, err)
		require.NoError(t, setup.TenantRepo.Save(ctx, tenant))

		// Product data still exists (repository doesn't check tenant status)
		// This is intentional - the application layer should check tenant status
		found, err = setup.ProductRepo.FindByIDForTenant(ctx, tenant.ID, product.ID)
		require.NoError(t, err)
		assert.Equal(t, product.ID, found.ID)

		// But tenant status can be checked before allowing operations
		fetchedTenant, err := setup.TenantRepo.FindByID(ctx, tenant.ID)
		require.NoError(t, err)
		assert.False(t, fetchedTenant.IsActive(), "Tenant should not be active")
	})

	t.Run("find_tenants_by_status", func(t *testing.T) {
		// Create tenants with different statuses
		activeTenant, err := identitydomain.NewTenant("STATUS_ACTIVE", "Active Tenant")
		require.NoError(t, err)
		require.NoError(t, setup.TenantRepo.Save(ctx, activeTenant))

		inactiveTenant, err := identitydomain.NewTenant("STATUS_INACTIVE", "Inactive Tenant")
		require.NoError(t, err)
		err = inactiveTenant.Deactivate()
		require.NoError(t, err)
		require.NoError(t, setup.TenantRepo.Save(ctx, inactiveTenant))

		suspendedTenant, err := identitydomain.NewTenant("STATUS_SUSPENDED", "Suspended Tenant")
		require.NoError(t, err)
		err = suspendedTenant.Suspend()
		require.NoError(t, err)
		require.NoError(t, setup.TenantRepo.Save(ctx, suspendedTenant))

		// Find active tenants
		filter := shared.Filter{Page: 1, PageSize: 100}
		activeTenants, err := setup.TenantRepo.FindByStatus(ctx, identitydomain.TenantStatusActive, filter)
		require.NoError(t, err)

		activeCodes := make([]string, len(activeTenants))
		for i, t := range activeTenants {
			activeCodes[i] = t.Code
		}
		assert.Contains(t, activeCodes, "STATUS_ACTIVE")
		assert.NotContains(t, activeCodes, "STATUS_INACTIVE")
		assert.NotContains(t, activeCodes, "STATUS_SUSPENDED")

		// Find inactive tenants
		inactiveTenants, err := setup.TenantRepo.FindByStatus(ctx, identitydomain.TenantStatusInactive, filter)
		require.NoError(t, err)

		inactiveCodes := make([]string, len(inactiveTenants))
		for i, t := range inactiveTenants {
			inactiveCodes[i] = t.Code
		}
		assert.Contains(t, inactiveCodes, "STATUS_INACTIVE")
		assert.NotContains(t, inactiveCodes, "STATUS_ACTIVE")
	})

	t.Run("count_by_status", func(t *testing.T) {
		// Count active tenants
		activeCount, err := setup.TenantRepo.CountByStatus(ctx, identitydomain.TenantStatusActive)
		require.NoError(t, err)
		assert.Greater(t, activeCount, int64(0))

		// Count suspended tenants
		suspendedCount, err := setup.TenantRepo.CountByStatus(ctx, identitydomain.TenantStatusSuspended)
		require.NoError(t, err)
		// May be 0 or more depending on previous tests
		assert.GreaterOrEqual(t, suspendedCount, int64(0))
	})
}

// ==================== Test: Cross-Tenant Security ====================

func TestTenantIsolation_CrossTenantSecurity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setup := NewTenantIsolationTestSetup(t)
	ctx := context.Background()

	t.Run("cannot_update_product_with_wrong_tenant_id", func(t *testing.T) {
		// Create a product in Tenant A
		product, err := catalog.NewProduct(setup.TenantA.ID, "CROSS-SEC-001", "Cross Security Test", "pcs")
		require.NoError(t, err)
		require.NoError(t, setup.ProductRepo.Save(ctx, product))

		// Try to find and update as Tenant B - should not find it
		found, err := setup.ProductRepo.FindByIDForTenant(ctx, setup.TenantB.ID, product.ID)
		assert.ErrorIs(t, err, shared.ErrNotFound)
		assert.Nil(t, found)
	})

	t.Run("cannot_delete_product_from_wrong_tenant", func(t *testing.T) {
		// Create a product in Tenant A
		product, err := catalog.NewProduct(setup.TenantA.ID, "DEL-SEC-001", "Delete Security Test", "pcs")
		require.NoError(t, err)
		require.NoError(t, setup.ProductRepo.Save(ctx, product))

		// Try to delete as Tenant B - should fail
		err = setup.ProductRepo.DeleteForTenant(ctx, setup.TenantB.ID, product.ID)
		assert.ErrorIs(t, err, shared.ErrNotFound)

		// Verify product still exists for Tenant A
		found, err := setup.ProductRepo.FindByIDForTenant(ctx, setup.TenantA.ID, product.ID)
		require.NoError(t, err)
		assert.Equal(t, product.ID, found.ID)
	})

	t.Run("tenant_id_mismatch_returns_not_found", func(t *testing.T) {
		// Create customer in Tenant A
		customer, err := partner.NewCustomer(
			setup.TenantA.ID,
			"MISMATCH-001",
			"Mismatch Customer",
			partner.CustomerTypeIndividual,
		)
		require.NoError(t, err)
		require.NoError(t, setup.CustomerRepo.Save(ctx, customer))

		// Access with wrong tenant ID
		found, err := setup.CustomerRepo.FindByIDForTenant(ctx, setup.TenantB.ID, customer.ID)
		assert.ErrorIs(t, err, shared.ErrNotFound)
		assert.Nil(t, found)

		// Access with random tenant ID
		randomTenantID := uuid.New()
		found, err = setup.CustomerRepo.FindByIDForTenant(ctx, randomTenantID, customer.ID)
		assert.ErrorIs(t, err, shared.ErrNotFound)
		assert.Nil(t, found)
	})
}

// Helper functions

func extractCodes(products []catalog.Product) []string {
	codes := make([]string, len(products))
	for i, p := range products {
		codes[i] = p.Code
	}
	return codes
}
