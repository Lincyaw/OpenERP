package partner

import (
	"testing"

	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCustomer(t *testing.T) {
	tenantID := uuid.New()

	t.Run("creates individual customer successfully", func(t *testing.T) {
		customer, err := NewCustomer(tenantID, "CUST001", "Test Customer", CustomerTypeIndividual)

		require.NoError(t, err)
		assert.NotNil(t, customer)
		assert.Equal(t, "CUST001", customer.Code)
		assert.Equal(t, "Test Customer", customer.Name)
		assert.Equal(t, CustomerTypeIndividual, customer.Type)
		assert.True(t, customer.Level.CodeEquals(NormalLevel())) // CustomerLevel is now a Value Object
		assert.Equal(t, CustomerStatusActive, customer.Status)
		assert.Equal(t, tenantID, customer.TenantID)
		assert.True(t, customer.CreditLimit.IsZero())
		assert.True(t, customer.Balance.IsZero())
		assert.Equal(t, "中国", customer.Country)
		assert.Len(t, customer.GetDomainEvents(), 1)
	})

	t.Run("creates organization customer successfully", func(t *testing.T) {
		customer, err := NewCustomer(tenantID, "CUST002", "Test Company Ltd", CustomerTypeOrganization)

		require.NoError(t, err)
		assert.Equal(t, CustomerTypeOrganization, customer.Type)
	})

	t.Run("converts code to uppercase", func(t *testing.T) {
		customer, err := NewCustomer(tenantID, "cust003", "Test Customer", CustomerTypeIndividual)

		require.NoError(t, err)
		assert.Equal(t, "CUST003", customer.Code)
	})

	t.Run("fails with empty code", func(t *testing.T) {
		customer, err := NewCustomer(tenantID, "", "Test Customer", CustomerTypeIndividual)

		assert.Error(t, err)
		assert.Nil(t, customer)
		assert.Contains(t, err.Error(), "code cannot be empty")
	})

	t.Run("fails with invalid code characters", func(t *testing.T) {
		customer, err := NewCustomer(tenantID, "CUST@001", "Test Customer", CustomerTypeIndividual)

		assert.Error(t, err)
		assert.Nil(t, customer)
		assert.Contains(t, err.Error(), "can only contain")
	})

	t.Run("fails with empty name", func(t *testing.T) {
		customer, err := NewCustomer(tenantID, "CUST001", "", CustomerTypeIndividual)

		assert.Error(t, err)
		assert.Nil(t, customer)
		assert.Contains(t, err.Error(), "name cannot be empty")
	})

	t.Run("fails with invalid type", func(t *testing.T) {
		customer, err := NewCustomer(tenantID, "CUST001", "Test", CustomerType("invalid"))

		assert.Error(t, err)
		assert.Nil(t, customer)
		assert.Contains(t, err.Error(), "individual")
	})
}

func TestNewIndividualCustomer(t *testing.T) {
	tenantID := uuid.New()

	customer, err := NewIndividualCustomer(tenantID, "IND001", "Individual Person")

	require.NoError(t, err)
	assert.Equal(t, CustomerTypeIndividual, customer.Type)
	assert.True(t, customer.IsIndividual())
	assert.False(t, customer.IsOrganization())
}

func TestNewOrganizationCustomer(t *testing.T) {
	tenantID := uuid.New()

	customer, err := NewOrganizationCustomer(tenantID, "ORG001", "Organization Inc")

	require.NoError(t, err)
	assert.Equal(t, CustomerTypeOrganization, customer.Type)
	assert.False(t, customer.IsIndividual())
	assert.True(t, customer.IsOrganization())
}

func TestCustomerUpdate(t *testing.T) {
	tenantID := uuid.New()
	customer, _ := NewCustomer(tenantID, "CUST001", "Original Name", CustomerTypeIndividual)
	customer.ClearDomainEvents()
	originalVersion := customer.Version

	t.Run("updates name and short name successfully", func(t *testing.T) {
		err := customer.Update("New Name", "Short")

		require.NoError(t, err)
		assert.Equal(t, "New Name", customer.Name)
		assert.Equal(t, "Short", customer.ShortName)
		// Version is now managed by repository, not domain
		assert.Equal(t, originalVersion, customer.Version)
		assert.Len(t, customer.GetDomainEvents(), 1)
	})

	t.Run("fails with empty name", func(t *testing.T) {
		err := customer.Update("", "Short")

		assert.Error(t, err)
	})
}

func TestCustomerContact(t *testing.T) {
	tenantID := uuid.New()
	customer, _ := NewCustomer(tenantID, "CUST001", "Test Customer", CustomerTypeIndividual)

	t.Run("sets valid contact info", func(t *testing.T) {
		err := customer.SetContact("John Doe", "13800138000", "john@example.com")

		require.NoError(t, err)
		assert.Equal(t, "John Doe", customer.ContactName)
		assert.Equal(t, "13800138000", customer.Phone)
		assert.Equal(t, "john@example.com", customer.Email)
	})

	t.Run("fails with invalid phone", func(t *testing.T) {
		err := customer.SetContact("John Doe", "invalid@phone", "john@example.com")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "phone")
	})

	t.Run("fails with invalid email", func(t *testing.T) {
		err := customer.SetContact("John Doe", "13800138000", "not-an-email")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "email")
	})

	t.Run("allows empty contact fields", func(t *testing.T) {
		err := customer.SetContact("", "", "")

		require.NoError(t, err)
	})
}

func TestCustomerAddress(t *testing.T) {
	tenantID := uuid.New()
	customer, _ := NewCustomer(tenantID, "CUST001", "Test Customer", CustomerTypeIndividual)

	t.Run("sets address successfully", func(t *testing.T) {
		err := customer.SetAddress("123 Main St", "Shanghai", "Shanghai", "200000", "China")

		require.NoError(t, err)
		assert.Equal(t, "123 Main St", customer.Address)
		assert.Equal(t, "Shanghai", customer.City)
		assert.Equal(t, "Shanghai", customer.Province)
		assert.Equal(t, "200000", customer.PostalCode)
		assert.Equal(t, "China", customer.Country)
	})

	t.Run("keeps default country if empty", func(t *testing.T) {
		customer2, _ := NewCustomer(tenantID, "CUST002", "Test", CustomerTypeIndividual)
		err := customer2.SetAddress("123 Main St", "Beijing", "Beijing", "100000", "")

		require.NoError(t, err)
		assert.Equal(t, "中国", customer2.Country)
	})
}

func TestCustomerLevel(t *testing.T) {
	tenantID := uuid.New()
	customer, _ := NewCustomer(tenantID, "CUST001", "Test Customer", CustomerTypeIndividual)
	customer.ClearDomainEvents()

	t.Run("upgrades level successfully", func(t *testing.T) {
		err := customer.SetLevel(GoldLevel())

		require.NoError(t, err)
		assert.True(t, customer.Level.CodeEquals(GoldLevel()))
		assert.Len(t, customer.GetDomainEvents(), 1)
	})

	t.Run("fails with invalid level", func(t *testing.T) {
		invalidLevel := CustomerLevel{} // Empty level
		err := customer.SetLevel(invalidLevel)

		assert.Error(t, err)
	})
}

func TestCustomerBalance(t *testing.T) {
	tenantID := uuid.New()
	customer, _ := NewCustomer(tenantID, "CUST001", "Test Customer", CustomerTypeIndividual)
	customer.ClearDomainEvents()

	t.Run("adds balance successfully", func(t *testing.T) {
		err := customer.AddBalance(decimal.NewFromFloat(100.50))

		require.NoError(t, err)
		assert.True(t, customer.Balance.Equal(decimal.NewFromFloat(100.50)))
		assert.True(t, customer.HasBalance())
		assert.Len(t, customer.GetDomainEvents(), 1)
	})

	t.Run("deducts balance successfully", func(t *testing.T) {
		customer.ClearDomainEvents()
		err := customer.DeductBalance(decimal.NewFromFloat(30.50))

		require.NoError(t, err)
		assert.True(t, customer.Balance.Equal(decimal.NewFromFloat(70.00)))
	})

	t.Run("fails to deduct more than balance", func(t *testing.T) {
		err := customer.DeductBalance(decimal.NewFromFloat(100.00))

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Insufficient")
	})

	t.Run("fails to add negative amount", func(t *testing.T) {
		err := customer.AddBalance(decimal.NewFromFloat(-50))

		assert.Error(t, err)
	})

	t.Run("fails to add zero amount", func(t *testing.T) {
		err := customer.AddBalance(decimal.Zero)

		assert.Error(t, err)
	})

	t.Run("refunds balance successfully", func(t *testing.T) {
		customer.ClearDomainEvents()
		err := customer.RefundBalance(decimal.NewFromFloat(20.00))

		require.NoError(t, err)
		assert.True(t, customer.Balance.Equal(decimal.NewFromFloat(90.00)))
	})
}

func TestCustomerCreditLimit(t *testing.T) {
	tenantID := uuid.New()
	customer, _ := NewCustomer(tenantID, "CUST001", "Test Customer", CustomerTypeIndividual)

	t.Run("sets credit limit successfully", func(t *testing.T) {
		err := customer.SetCreditLimit(decimal.NewFromFloat(5000.00))

		require.NoError(t, err)
		assert.True(t, customer.CreditLimit.Equal(decimal.NewFromFloat(5000.00)))
		assert.True(t, customer.HasCreditLimit())
	})

	t.Run("fails with negative credit limit", func(t *testing.T) {
		err := customer.SetCreditLimit(decimal.NewFromFloat(-100))

		assert.Error(t, err)
	})
}

func TestCustomerStatus(t *testing.T) {
	tenantID := uuid.New()

	t.Run("deactivates customer", func(t *testing.T) {
		customer, _ := NewCustomer(tenantID, "CUST001", "Test", CustomerTypeIndividual)
		customer.ClearDomainEvents()

		err := customer.Deactivate()

		require.NoError(t, err)
		assert.True(t, customer.IsInactive())
		assert.False(t, customer.IsActive())
		assert.Len(t, customer.GetDomainEvents(), 1)
	})

	t.Run("activates customer", func(t *testing.T) {
		customer, _ := NewCustomer(tenantID, "CUST002", "Test", CustomerTypeIndividual)
		customer.Deactivate()
		customer.ClearDomainEvents()

		err := customer.Activate()

		require.NoError(t, err)
		assert.True(t, customer.IsActive())
	})

	t.Run("suspends customer", func(t *testing.T) {
		customer, _ := NewCustomer(tenantID, "CUST003", "Test", CustomerTypeIndividual)
		customer.ClearDomainEvents()

		err := customer.Suspend()

		require.NoError(t, err)
		assert.True(t, customer.IsSuspended())
	})

	t.Run("fails to deactivate already inactive customer", func(t *testing.T) {
		customer, _ := NewCustomer(tenantID, "CUST004", "Test", CustomerTypeIndividual)
		customer.Deactivate()

		err := customer.Deactivate()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already inactive")
	})

	t.Run("fails to activate already active customer", func(t *testing.T) {
		customer, _ := NewCustomer(tenantID, "CUST005", "Test", CustomerTypeIndividual)

		err := customer.Activate()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already active")
	})
}

func TestCustomerAttributes(t *testing.T) {
	tenantID := uuid.New()
	customer, _ := NewCustomer(tenantID, "CUST001", "Test", CustomerTypeIndividual)

	t.Run("sets valid JSON attributes", func(t *testing.T) {
		err := customer.SetAttributes(`{"vip": true, "discount": 10}`)

		require.NoError(t, err)
		assert.Equal(t, `{"vip": true, "discount": 10}`, customer.Attributes)
	})

	t.Run("fails with invalid JSON", func(t *testing.T) {
		err := customer.SetAttributes(`not json`)

		assert.Error(t, err)
	})

	t.Run("converts empty string to empty object", func(t *testing.T) {
		err := customer.SetAttributes("")

		require.NoError(t, err)
		assert.Equal(t, "{}", customer.Attributes)
	})
}

func TestGetFullAddress(t *testing.T) {
	tenantID := uuid.New()
	customer, _ := NewCustomer(tenantID, "CUST001", "Test", CustomerTypeIndividual)
	customer.SetAddress("123 Main Street", "Shanghai", "Shanghai", "200000", "中国")

	fullAddress := customer.GetFullAddress()

	assert.Contains(t, fullAddress, "中国")
	assert.Contains(t, fullAddress, "Shanghai")
	assert.Contains(t, fullAddress, "123 Main Street")
	assert.Contains(t, fullAddress, "200000")
}

func TestCustomerAddressVO(t *testing.T) {
	tenantID := uuid.New()

	t.Run("GetAddressVO returns Address value object", func(t *testing.T) {
		customer, _ := NewCustomer(tenantID, "CUST001", "Test", CustomerTypeIndividual)
		customer.SetAddress("科技园南路123号", "深圳市", "广东省", "518000", "中国")

		addr := customer.GetAddressVO()

		assert.False(t, addr.IsEmpty())
		assert.Equal(t, "广东省", addr.Province())
		assert.Equal(t, "深圳市", addr.City())
		assert.Equal(t, "科技园南路123号", addr.Detail())
		assert.Equal(t, "518000", addr.PostalCode())
	})

	t.Run("GetAddressVO returns empty for customer without address", func(t *testing.T) {
		customer, _ := NewCustomer(tenantID, "CUST002", "Test", CustomerTypeIndividual)

		addr := customer.GetAddressVO()

		assert.True(t, addr.IsEmpty())
	})

	t.Run("SetAddressVO sets address from value object", func(t *testing.T) {
		customer, _ := NewCustomer(tenantID, "CUST003", "Test", CustomerTypeIndividual)
		addr := valueobject.MustNewAddress("广东省", "深圳市", "南山区", "科技园南路123号",
			valueobject.WithPostalCode("518000"), valueobject.WithCountry("中国"))

		customer.SetAddressVO(addr)

		assert.Equal(t, "广东省", customer.Province)
		assert.Equal(t, "深圳市", customer.City)
		assert.Contains(t, customer.Address, "南山区")
		assert.Contains(t, customer.Address, "科技园南路123号")
		assert.Equal(t, "518000", customer.PostalCode)
		assert.Equal(t, "中国", customer.Country)
	})

	t.Run("SetAddressVO clears address with empty value object", func(t *testing.T) {
		customer, _ := NewCustomer(tenantID, "CUST004", "Test", CustomerTypeIndividual)
		customer.SetAddress("Some Address", "City", "Province", "100000", "中国")

		customer.SetAddressVO(valueobject.EmptyAddress())

		assert.Equal(t, "", customer.Province)
		assert.Equal(t, "", customer.City)
		assert.Equal(t, "", customer.Address)
		assert.Equal(t, "", customer.PostalCode)
		// Country should be kept for default
		assert.Equal(t, "中国", customer.Country)
	})
}
