package integration

import (
	"context"
	"testing"

	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/persistence"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCustomerRepository_Integration tests the CustomerRepository against a real PostgreSQL database
func TestCustomerRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := NewTestDB(t)
	repo := persistence.NewGormCustomerRepository(testDB.DB)
	ctx := context.Background()
	tenantID := uuid.New()

	// Create tenant first (required for foreign key)
	testDB.CreateTestTenantWithUUID(tenantID)

	t.Run("Save and FindByID", func(t *testing.T) {
		customer, err := partner.NewIndividualCustomer(tenantID, "CUST-001", "Test Customer")
		require.NoError(t, err)

		err = repo.Save(ctx, customer)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, customer.ID)
		require.NoError(t, err)
		assert.Equal(t, customer.ID, found.ID)
		assert.Equal(t, customer.Code, found.Code)
		assert.Equal(t, customer.Name, found.Name)
		assert.Equal(t, customer.TenantID, found.TenantID)
	})

	t.Run("FindByIDForTenant", func(t *testing.T) {
		customer, err := partner.NewOrganizationCustomer(tenantID, "CUST-002", "Organization Customer")
		require.NoError(t, err)

		err = repo.Save(ctx, customer)
		require.NoError(t, err)

		// Should find with correct tenant
		found, err := repo.FindByIDForTenant(ctx, tenantID, customer.ID)
		require.NoError(t, err)
		assert.Equal(t, customer.ID, found.ID)

		// Should not find with different tenant
		otherTenant := uuid.New()
		_, err = repo.FindByIDForTenant(ctx, otherTenant, customer.ID)
		assert.ErrorIs(t, err, shared.ErrNotFound)
	})

	t.Run("FindByCode", func(t *testing.T) {
		customer, err := partner.NewIndividualCustomer(tenantID, "CUST-003", "Code Customer")
		require.NoError(t, err)

		err = repo.Save(ctx, customer)
		require.NoError(t, err)

		// Should find by code (case-insensitive)
		found, err := repo.FindByCode(ctx, tenantID, "cust-003")
		require.NoError(t, err)
		assert.Equal(t, "CUST-003", found.Code)
	})

	t.Run("FindAllForTenant with pagination", func(t *testing.T) {
		// Create customers for pagination test
		paginationTenant := uuid.New()
		testDB.CreateTestTenantWithUUID(paginationTenant)
		for i := range 10 {
			customer, err := partner.NewIndividualCustomer(paginationTenant, "PAGE-CUST-"+string(rune('A'+i)), "Page Customer "+string(rune('A'+i)))
			require.NoError(t, err)
			err = repo.Save(ctx, customer)
			require.NoError(t, err)
		}

		// Test pagination
		filter := shared.Filter{
			Page:     1,
			PageSize: 5,
		}
		customers, err := repo.FindAllForTenant(ctx, paginationTenant, filter)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(customers), 5)

		// Second page
		filter.Page = 2
		page2Customers, err := repo.FindAllForTenant(ctx, paginationTenant, filter)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(page2Customers), 5)
	})

	t.Run("FindByStatus", func(t *testing.T) {
		statusTenant := uuid.New()
		testDB.CreateTestTenantWithUUID(statusTenant)

		// Create active customer
		activeCustomer, err := partner.NewIndividualCustomer(statusTenant, "STATUS-ACTIVE", "Active Customer")
		require.NoError(t, err)
		err = repo.Save(ctx, activeCustomer)
		require.NoError(t, err)

		// Create inactive customer
		inactiveCustomer, err := partner.NewIndividualCustomer(statusTenant, "STATUS-INACTIVE", "Inactive Customer")
		require.NoError(t, err)
		err = inactiveCustomer.Deactivate()
		require.NoError(t, err)
		err = repo.Save(ctx, inactiveCustomer)
		require.NoError(t, err)

		// Find active customers
		activeCustomers, err := repo.FindByStatus(ctx, statusTenant, partner.CustomerStatusActive, shared.Filter{})
		require.NoError(t, err)
		assert.True(t, len(activeCustomers) >= 1)
		for _, c := range activeCustomers {
			assert.Equal(t, partner.CustomerStatusActive, c.Status)
		}

		// Find inactive customers
		inactiveCustomers, err := repo.FindByStatus(ctx, statusTenant, partner.CustomerStatusInactive, shared.Filter{})
		require.NoError(t, err)
		assert.True(t, len(inactiveCustomers) >= 1)
		for _, c := range inactiveCustomers {
			assert.Equal(t, partner.CustomerStatusInactive, c.Status)
		}
	})

	t.Run("FindByType", func(t *testing.T) {
		typeTenant := uuid.New()
		testDB.CreateTestTenantWithUUID(typeTenant)

		// Create individual customer
		individual, err := partner.NewIndividualCustomer(typeTenant, "TYPE-IND", "Individual")
		require.NoError(t, err)
		err = repo.Save(ctx, individual)
		require.NoError(t, err)

		// Create organization customer
		org, err := partner.NewOrganizationCustomer(typeTenant, "TYPE-ORG", "Organization")
		require.NoError(t, err)
		err = repo.Save(ctx, org)
		require.NoError(t, err)

		// Find individual customers
		individuals, err := repo.FindByType(ctx, typeTenant, partner.CustomerTypeIndividual, shared.Filter{})
		require.NoError(t, err)
		assert.True(t, len(individuals) >= 1)
		for _, c := range individuals {
			assert.Equal(t, partner.CustomerTypeIndividual, c.Type)
		}

		// Find organization customers
		orgs, err := repo.FindByType(ctx, typeTenant, partner.CustomerTypeOrganization, shared.Filter{})
		require.NoError(t, err)
		assert.True(t, len(orgs) >= 1)
		for _, c := range orgs {
			assert.Equal(t, partner.CustomerTypeOrganization, c.Type)
		}
	})

	t.Run("Update customer", func(t *testing.T) {
		customer, err := partner.NewIndividualCustomer(tenantID, "UPDATE-CUST", "Original Name")
		require.NoError(t, err)

		err = repo.Save(ctx, customer)
		require.NoError(t, err)

		// Update the customer
		err = customer.Update("Updated Name", "Short Name")
		require.NoError(t, err)
		err = customer.SetContact("John Doe", "13800138000", "john@example.com")
		require.NoError(t, err)

		err = repo.Save(ctx, customer)
		require.NoError(t, err)

		// Verify update
		found, err := repo.FindByID(ctx, customer.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", found.Name)
		assert.Equal(t, "Short Name", found.ShortName)
		assert.Equal(t, "John Doe", found.ContactName)
		assert.Equal(t, "13800138000", found.Phone)
		assert.Equal(t, "john@example.com", found.Email)
	})

	t.Run("Customer balance operations", func(t *testing.T) {
		customer, err := partner.NewIndividualCustomer(tenantID, "BALANCE-CUST", "Balance Customer")
		require.NoError(t, err)

		err = repo.Save(ctx, customer)
		require.NoError(t, err)

		// Add balance
		err = customer.AddBalance(decimal.NewFromFloat(1000.00))
		require.NoError(t, err)
		err = repo.Save(ctx, customer)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, customer.ID)
		require.NoError(t, err)
		assert.True(t, found.Balance.Equal(decimal.NewFromFloat(1000.00)))

		// Deduct balance
		err = found.DeductBalance(decimal.NewFromFloat(300.00))
		require.NoError(t, err)
		err = repo.Save(ctx, found)
		require.NoError(t, err)

		found2, err := repo.FindByID(ctx, customer.ID)
		require.NoError(t, err)
		assert.True(t, found2.Balance.Equal(decimal.NewFromFloat(700.00)))
	})

	t.Run("Delete customer", func(t *testing.T) {
		customer, err := partner.NewIndividualCustomer(tenantID, "DELETE-CUST", "To Delete")
		require.NoError(t, err)

		err = repo.Save(ctx, customer)
		require.NoError(t, err)

		// Delete
		err = repo.Delete(ctx, customer.ID)
		require.NoError(t, err)

		// Verify deletion
		_, err = repo.FindByID(ctx, customer.ID)
		assert.ErrorIs(t, err, shared.ErrNotFound)
	})

	t.Run("CountForTenant", func(t *testing.T) {
		countTenant := uuid.New()
		testDB.CreateTestTenantWithUUID(countTenant)

		for i := range 5 {
			customer, err := partner.NewIndividualCustomer(countTenant, "COUNT-"+string(rune('A'+i)), "Count Customer "+string(rune('A'+i)))
			require.NoError(t, err)
			err = repo.Save(ctx, customer)
			require.NoError(t, err)
		}

		count, err := repo.CountForTenant(ctx, countTenant, shared.Filter{})
		require.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})

	t.Run("ExistsByCode", func(t *testing.T) {
		customer, err := partner.NewIndividualCustomer(tenantID, "EXISTS-CODE", "Exists Customer")
		require.NoError(t, err)
		err = repo.Save(ctx, customer)
		require.NoError(t, err)

		exists, err := repo.ExistsByCode(ctx, tenantID, "EXISTS-CODE")
		require.NoError(t, err)
		assert.True(t, exists)

		exists, err = repo.ExistsByCode(ctx, tenantID, "NONEXISTENT-CODE")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Search with filter", func(t *testing.T) {
		searchTenant := uuid.New()
		testDB.CreateTestTenantWithUUID(searchTenant)

		// Create customers with various levels
		customers := []struct {
			code  string
			name  string
			level partner.CustomerLevel
		}{
			{"SEARCH-NORMAL", "Normal Customer", partner.CustomerLevelNormal},
			{"SEARCH-VIP", "VIP Customer", partner.CustomerLevelVIP},
			{"SEARCH-GOLD", "Gold Customer", partner.CustomerLevelGold},
		}

		for _, c := range customers {
			customer, err := partner.NewIndividualCustomer(searchTenant, c.code, c.name)
			require.NoError(t, err)
			if c.level != partner.CustomerLevelNormal {
				err = customer.SetLevel(c.level)
				require.NoError(t, err)
			}
			err = repo.Save(ctx, customer)
			require.NoError(t, err)
		}

		// Search by name
		filter := shared.Filter{
			Search: "Customer",
		}
		found, err := repo.FindAllForTenant(ctx, searchTenant, filter)
		require.NoError(t, err)
		assert.Equal(t, 3, len(found))

		// Filter by level
		filter = shared.Filter{
			Filters: map[string]any{
				"level": partner.CustomerLevelVIP,
			},
		}
		found, err = repo.FindAllForTenant(ctx, searchTenant, filter)
		require.NoError(t, err)
		assert.Equal(t, 1, len(found))
		assert.Equal(t, "SEARCH-VIP", found[0].Code)
	})
}

// TestCustomerRepository_TenantIsolation tests tenant data isolation
func TestCustomerRepository_TenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := NewTestDB(t)
	repo := persistence.NewGormCustomerRepository(testDB.DB)
	ctx := context.Background()

	tenant1 := uuid.New()
	tenant2 := uuid.New()

	// Create tenants first (required for foreign keys)
	testDB.CreateTestTenantWithUUID(tenant1)
	testDB.CreateTestTenantWithUUID(tenant2)

	// Create customers for tenant 1
	for i := range 3 {
		customer, err := partner.NewIndividualCustomer(tenant1, "T1-CUST-"+string(rune('A'+i)), "Tenant 1 Customer")
		require.NoError(t, err)
		err = repo.Save(ctx, customer)
		require.NoError(t, err)
	}

	// Create customers for tenant 2
	for i := range 2 {
		customer, err := partner.NewIndividualCustomer(tenant2, "T2-CUST-"+string(rune('A'+i)), "Tenant 2 Customer")
		require.NoError(t, err)
		err = repo.Save(ctx, customer)
		require.NoError(t, err)
	}

	// Verify tenant 1 only sees their customers
	t1Customers, err := repo.FindAllForTenant(ctx, tenant1, shared.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 3, len(t1Customers))
	for _, c := range t1Customers {
		assert.Equal(t, tenant1, c.TenantID)
	}

	// Verify tenant 2 only sees their customers
	t2Customers, err := repo.FindAllForTenant(ctx, tenant2, shared.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 2, len(t2Customers))
	for _, c := range t2Customers {
		assert.Equal(t, tenant2, c.TenantID)
	}
}
