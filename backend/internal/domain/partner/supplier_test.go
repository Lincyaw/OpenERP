package partner

import (
	"testing"

	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSupplier(t *testing.T) {
	tenantID := uuid.New()

	t.Run("creates supplier with valid input", func(t *testing.T) {
		supplier, err := NewSupplier(tenantID, "SUP001", "Test Supplier", SupplierTypeDistributor)
		require.NoError(t, err)
		require.NotNil(t, supplier)

		assert.NotEqual(t, uuid.Nil, supplier.ID)
		assert.Equal(t, tenantID, supplier.TenantID)
		assert.Equal(t, "SUP001", supplier.Code)
		assert.Equal(t, "Test Supplier", supplier.Name)
		assert.Equal(t, SupplierTypeDistributor, supplier.Type)
		assert.Equal(t, SupplierStatusActive, supplier.Status)
		assert.Equal(t, 0, supplier.CreditDays)
		assert.True(t, supplier.CreditLimit.IsZero())
		assert.True(t, supplier.Balance.IsZero())
		assert.Equal(t, 0, supplier.Rating)
		assert.Equal(t, "中国", supplier.Country)
		assert.Equal(t, "{}", supplier.Attributes)

		// Should have created event
		events := supplier.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeSupplierCreated, events[0].EventType())
	})

	t.Run("converts code to uppercase", func(t *testing.T) {
		supplier, err := NewSupplier(tenantID, "sup001", "Test Supplier", SupplierTypeManufacturer)
		require.NoError(t, err)
		assert.Equal(t, "SUP001", supplier.Code)
	})

	t.Run("fails with empty code", func(t *testing.T) {
		supplier, err := NewSupplier(tenantID, "", "Test Supplier", SupplierTypeDistributor)
		assert.Nil(t, supplier)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("fails with empty name", func(t *testing.T) {
		supplier, err := NewSupplier(tenantID, "SUP001", "", SupplierTypeDistributor)
		assert.Nil(t, supplier)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("fails with invalid type", func(t *testing.T) {
		supplier, err := NewSupplier(tenantID, "SUP001", "Test Supplier", SupplierType("invalid"))
		assert.Nil(t, supplier)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid supplier type")
	})

	t.Run("fails with invalid characters in code", func(t *testing.T) {
		supplier, err := NewSupplier(tenantID, "SUP@001", "Test Supplier", SupplierTypeDistributor)
		assert.Nil(t, supplier)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only contain letters, numbers, underscores, and hyphens")
	})
}

func TestNewManufacturerSupplier(t *testing.T) {
	tenantID := uuid.New()
	supplier, err := NewManufacturerSupplier(tenantID, "MFG001", "Factory Inc")
	require.NoError(t, err)
	assert.Equal(t, SupplierTypeManufacturer, supplier.Type)
}

func TestNewDistributorSupplier(t *testing.T) {
	tenantID := uuid.New()
	supplier, err := NewDistributorSupplier(tenantID, "DST001", "Distributor Corp")
	require.NoError(t, err)
	assert.Equal(t, SupplierTypeDistributor, supplier.Type)
}

func TestSupplier_Update(t *testing.T) {
	supplier := createTestSupplier(t)

	t.Run("updates name and short name", func(t *testing.T) {
		supplier.ClearDomainEvents()
		err := supplier.Update("New Name", "New Short")
		require.NoError(t, err)
		assert.Equal(t, "New Name", supplier.Name)
		assert.Equal(t, "New Short", supplier.ShortName)

		events := supplier.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeSupplierUpdated, events[0].EventType())
	})

	t.Run("fails with empty name", func(t *testing.T) {
		err := supplier.Update("", "Short")
		assert.Error(t, err)
	})

	t.Run("fails with short name too long", func(t *testing.T) {
		longName := make([]byte, 101)
		for i := range longName {
			longName[i] = 'a'
		}
		err := supplier.Update("Valid Name", string(longName))
		assert.Error(t, err)
	})
}

func TestSupplier_UpdateCode(t *testing.T) {
	supplier := createTestSupplier(t)

	t.Run("updates code", func(t *testing.T) {
		supplier.ClearDomainEvents()
		err := supplier.UpdateCode("NEW001")
		require.NoError(t, err)
		assert.Equal(t, "NEW001", supplier.Code)

		events := supplier.GetDomainEvents()
		require.Len(t, events, 1)
	})

	t.Run("converts to uppercase", func(t *testing.T) {
		err := supplier.UpdateCode("new002")
		require.NoError(t, err)
		assert.Equal(t, "NEW002", supplier.Code)
	})

	t.Run("fails with empty code", func(t *testing.T) {
		err := supplier.UpdateCode("")
		assert.Error(t, err)
	})
}

func TestSupplier_SetContact(t *testing.T) {
	supplier := createTestSupplier(t)

	t.Run("sets contact information", func(t *testing.T) {
		err := supplier.SetContact("John Doe", "1234567890", "john@example.com")
		require.NoError(t, err)
		assert.Equal(t, "John Doe", supplier.ContactName)
		assert.Equal(t, "1234567890", supplier.Phone)
		assert.Equal(t, "john@example.com", supplier.Email)
	})

	t.Run("allows empty values", func(t *testing.T) {
		err := supplier.SetContact("", "", "")
		require.NoError(t, err)
		assert.Empty(t, supplier.ContactName)
		assert.Empty(t, supplier.Phone)
		assert.Empty(t, supplier.Email)
	})

	t.Run("fails with invalid phone", func(t *testing.T) {
		err := supplier.SetContact("Jane", "invalid@phone", "jane@example.com")
		assert.Error(t, err)
	})

	t.Run("fails with invalid email", func(t *testing.T) {
		err := supplier.SetContact("Jane", "1234567890", "invalid-email")
		assert.Error(t, err)
	})

	t.Run("fails with contact name too long", func(t *testing.T) {
		longName := make([]byte, 101)
		for i := range longName {
			longName[i] = 'a'
		}
		err := supplier.SetContact(string(longName), "1234567890", "test@example.com")
		assert.Error(t, err)
	})
}

func TestSupplier_SetAddress(t *testing.T) {
	supplier := createTestSupplier(t)

	t.Run("sets address information", func(t *testing.T) {
		err := supplier.SetAddress("123 Main St", "Shanghai", "Shanghai", "200000", "China")
		require.NoError(t, err)
		assert.Equal(t, "123 Main St", supplier.Address)
		assert.Equal(t, "Shanghai", supplier.City)
		assert.Equal(t, "Shanghai", supplier.Province)
		assert.Equal(t, "200000", supplier.PostalCode)
		assert.Equal(t, "China", supplier.Country)
	})

	t.Run("keeps default country when empty", func(t *testing.T) {
		supplier.Country = "中国"
		err := supplier.SetAddress("456 Other St", "Beijing", "Beijing", "100000", "")
		require.NoError(t, err)
		assert.Equal(t, "中国", supplier.Country)
	})

	t.Run("fails with address too long", func(t *testing.T) {
		longAddress := make([]byte, 501)
		for i := range longAddress {
			longAddress[i] = 'a'
		}
		err := supplier.SetAddress(string(longAddress), "City", "Province", "12345", "Country")
		assert.Error(t, err)
	})
}

func TestSupplier_SetTaxID(t *testing.T) {
	supplier := createTestSupplier(t)

	t.Run("sets tax ID", func(t *testing.T) {
		err := supplier.SetTaxID("91310000MA1K4GCFXQ")
		require.NoError(t, err)
		assert.Equal(t, "91310000MA1K4GCFXQ", supplier.TaxID)
	})

	t.Run("allows empty tax ID", func(t *testing.T) {
		err := supplier.SetTaxID("")
		require.NoError(t, err)
		assert.Empty(t, supplier.TaxID)
	})

	t.Run("fails with tax ID too long", func(t *testing.T) {
		longTaxID := make([]byte, 51)
		for i := range longTaxID {
			longTaxID[i] = '1'
		}
		err := supplier.SetTaxID(string(longTaxID))
		assert.Error(t, err)
	})
}

func TestSupplier_SetBankInfo(t *testing.T) {
	supplier := createTestSupplier(t)

	t.Run("sets bank information", func(t *testing.T) {
		err := supplier.SetBankInfo("Bank of China", "1234567890123456")
		require.NoError(t, err)
		assert.Equal(t, "Bank of China", supplier.BankName)
		assert.Equal(t, "1234567890123456", supplier.BankAccount)
	})

	t.Run("allows empty values", func(t *testing.T) {
		err := supplier.SetBankInfo("", "")
		require.NoError(t, err)
		assert.Empty(t, supplier.BankName)
		assert.Empty(t, supplier.BankAccount)
	})

	t.Run("fails with bank name too long", func(t *testing.T) {
		longName := make([]byte, 201)
		for i := range longName {
			longName[i] = 'a'
		}
		err := supplier.SetBankInfo(string(longName), "1234567890")
		assert.Error(t, err)
	})

	t.Run("fails with bank account too long", func(t *testing.T) {
		longAccount := make([]byte, 101)
		for i := range longAccount {
			longAccount[i] = '1'
		}
		err := supplier.SetBankInfo("Bank", string(longAccount))
		assert.Error(t, err)
	})
}

func TestSupplier_SetPaymentTerms(t *testing.T) {
	supplier := createTestSupplier(t)

	t.Run("sets payment terms", func(t *testing.T) {
		supplier.ClearDomainEvents()
		err := supplier.SetPaymentTerms(30, decimal.NewFromInt(50000))
		require.NoError(t, err)
		assert.Equal(t, 30, supplier.CreditDays)
		assert.True(t, supplier.CreditLimit.Equal(decimal.NewFromInt(50000)))

		events := supplier.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeSupplierPaymentTermsChanged, events[0].EventType())
	})

	t.Run("fails with negative credit days", func(t *testing.T) {
		err := supplier.SetPaymentTerms(-1, decimal.NewFromInt(1000))
		assert.Error(t, err)
	})

	t.Run("fails with credit days exceeding 365", func(t *testing.T) {
		err := supplier.SetPaymentTerms(400, decimal.NewFromInt(1000))
		assert.Error(t, err)
	})

	t.Run("fails with negative credit limit", func(t *testing.T) {
		err := supplier.SetPaymentTerms(30, decimal.NewFromInt(-1000))
		assert.Error(t, err)
	})
}

func TestSupplier_SetCreditDays(t *testing.T) {
	supplier := createTestSupplier(t)
	supplier.CreditLimit = decimal.NewFromInt(10000)

	err := supplier.SetCreditDays(45)
	require.NoError(t, err)
	assert.Equal(t, 45, supplier.CreditDays)
	assert.True(t, supplier.CreditLimit.Equal(decimal.NewFromInt(10000))) // Limit should not change
}

func TestSupplier_SetCreditLimit(t *testing.T) {
	supplier := createTestSupplier(t)
	supplier.CreditDays = 30

	err := supplier.SetCreditLimit(decimal.NewFromInt(25000))
	require.NoError(t, err)
	assert.True(t, supplier.CreditLimit.Equal(decimal.NewFromInt(25000)))
	assert.Equal(t, 30, supplier.CreditDays) // Days should not change
}

func TestSupplier_AddBalance(t *testing.T) {
	supplier := createTestSupplier(t)

	t.Run("adds to balance", func(t *testing.T) {
		supplier.ClearDomainEvents()
		err := supplier.AddBalance(decimal.NewFromInt(1000))
		require.NoError(t, err)
		assert.True(t, supplier.Balance.Equal(decimal.NewFromInt(1000)))

		events := supplier.GetDomainEvents()
		require.Len(t, events, 1)
		balanceEvent := events[0].(*SupplierBalanceChangedEvent)
		assert.Equal(t, "purchase", balanceEvent.Reason)
	})

	t.Run("accumulates balance", func(t *testing.T) {
		supplier.Balance = decimal.NewFromInt(500)
		err := supplier.AddBalance(decimal.NewFromInt(300))
		require.NoError(t, err)
		assert.True(t, supplier.Balance.Equal(decimal.NewFromInt(800)))
	})

	t.Run("fails with negative amount", func(t *testing.T) {
		err := supplier.AddBalance(decimal.NewFromInt(-100))
		assert.Error(t, err)
	})

	t.Run("fails with zero amount", func(t *testing.T) {
		err := supplier.AddBalance(decimal.Zero)
		assert.Error(t, err)
	})
}

func TestSupplier_DeductBalance(t *testing.T) {
	supplier := createTestSupplier(t)
	supplier.Balance = decimal.NewFromInt(1000)

	t.Run("deducts from balance", func(t *testing.T) {
		supplier.ClearDomainEvents()
		err := supplier.DeductBalance(decimal.NewFromInt(300))
		require.NoError(t, err)
		assert.True(t, supplier.Balance.Equal(decimal.NewFromInt(700)))

		events := supplier.GetDomainEvents()
		require.Len(t, events, 1)
		balanceEvent := events[0].(*SupplierBalanceChangedEvent)
		assert.Equal(t, "payment", balanceEvent.Reason)
	})

	t.Run("fails when amount exceeds balance", func(t *testing.T) {
		supplier.Balance = decimal.NewFromInt(100)
		err := supplier.DeductBalance(decimal.NewFromInt(200))
		assert.Error(t, err)
	})

	t.Run("fails with negative amount", func(t *testing.T) {
		err := supplier.DeductBalance(decimal.NewFromInt(-100))
		assert.Error(t, err)
	})

	t.Run("fails with zero amount", func(t *testing.T) {
		err := supplier.DeductBalance(decimal.Zero)
		assert.Error(t, err)
	})
}

func TestSupplier_AdjustBalance(t *testing.T) {
	supplier := createTestSupplier(t)
	supplier.Balance = decimal.NewFromInt(1000)

	t.Run("adjusts balance positive", func(t *testing.T) {
		supplier.ClearDomainEvents()
		err := supplier.AdjustBalance(decimal.NewFromInt(500), "correction")
		require.NoError(t, err)
		assert.True(t, supplier.Balance.Equal(decimal.NewFromInt(1500)))

		events := supplier.GetDomainEvents()
		require.Len(t, events, 1)
		balanceEvent := events[0].(*SupplierBalanceChangedEvent)
		assert.Equal(t, "correction", balanceEvent.Reason)
	})

	t.Run("adjusts balance negative", func(t *testing.T) {
		supplier.Balance = decimal.NewFromInt(1000)
		err := supplier.AdjustBalance(decimal.NewFromInt(-300), "write-off")
		require.NoError(t, err)
		assert.True(t, supplier.Balance.Equal(decimal.NewFromInt(700)))
	})

	t.Run("fails with zero amount", func(t *testing.T) {
		err := supplier.AdjustBalance(decimal.Zero, "reason")
		assert.Error(t, err)
	})

	t.Run("fails if result would be negative", func(t *testing.T) {
		supplier.Balance = decimal.NewFromInt(500)
		err := supplier.AdjustBalance(decimal.NewFromInt(-600), "write-off")
		assert.Error(t, err)
	})
}

func TestSupplier_SetRating(t *testing.T) {
	supplier := createTestSupplier(t)

	t.Run("sets valid rating", func(t *testing.T) {
		for rating := 0; rating <= 5; rating++ {
			err := supplier.SetRating(rating)
			require.NoError(t, err)
			assert.Equal(t, rating, supplier.Rating)
		}
	})

	t.Run("fails with rating below 0", func(t *testing.T) {
		err := supplier.SetRating(-1)
		assert.Error(t, err)
	})

	t.Run("fails with rating above 5", func(t *testing.T) {
		err := supplier.SetRating(6)
		assert.Error(t, err)
	})
}

func TestSupplier_SetNotes(t *testing.T) {
	supplier := createTestSupplier(t)
	supplier.SetNotes("Important note")
	assert.Equal(t, "Important note", supplier.Notes)
}

func TestSupplier_SetSortOrder(t *testing.T) {
	supplier := createTestSupplier(t)
	supplier.SetSortOrder(10)
	assert.Equal(t, 10, supplier.SortOrder)
}

func TestSupplier_SetAttributes(t *testing.T) {
	supplier := createTestSupplier(t)

	t.Run("sets valid JSON", func(t *testing.T) {
		err := supplier.SetAttributes(`{"key": "value"}`)
		require.NoError(t, err)
		assert.Equal(t, `{"key": "value"}`, supplier.Attributes)
	})

	t.Run("sets empty to default", func(t *testing.T) {
		err := supplier.SetAttributes("")
		require.NoError(t, err)
		assert.Equal(t, "{}", supplier.Attributes)
	})

	t.Run("trims whitespace", func(t *testing.T) {
		err := supplier.SetAttributes("  {\"trimmed\": true}  ")
		require.NoError(t, err)
		assert.Equal(t, `{"trimmed": true}`, supplier.Attributes)
	})

	t.Run("fails with invalid JSON", func(t *testing.T) {
		err := supplier.SetAttributes("not json")
		assert.Error(t, err)
	})

	t.Run("fails with array instead of object", func(t *testing.T) {
		err := supplier.SetAttributes("[1,2,3]")
		assert.Error(t, err)
	})
}

func TestSupplier_StatusChanges(t *testing.T) {
	t.Run("activate", func(t *testing.T) {
		supplier := createTestSupplier(t)
		supplier.Status = SupplierStatusInactive
		supplier.ClearDomainEvents()

		err := supplier.Activate()
		require.NoError(t, err)
		assert.Equal(t, SupplierStatusActive, supplier.Status)

		events := supplier.GetDomainEvents()
		require.Len(t, events, 1)
		statusEvent := events[0].(*SupplierStatusChangedEvent)
		assert.Equal(t, SupplierStatusInactive, statusEvent.OldStatus)
		assert.Equal(t, SupplierStatusActive, statusEvent.NewStatus)
	})

	t.Run("activate already active fails", func(t *testing.T) {
		supplier := createTestSupplier(t)
		err := supplier.Activate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already active")
	})

	t.Run("deactivate", func(t *testing.T) {
		supplier := createTestSupplier(t)
		supplier.ClearDomainEvents()

		err := supplier.Deactivate()
		require.NoError(t, err)
		assert.Equal(t, SupplierStatusInactive, supplier.Status)

		events := supplier.GetDomainEvents()
		require.Len(t, events, 1)
	})

	t.Run("deactivate already inactive fails", func(t *testing.T) {
		supplier := createTestSupplier(t)
		supplier.Status = SupplierStatusInactive
		err := supplier.Deactivate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already inactive")
	})

	t.Run("block", func(t *testing.T) {
		supplier := createTestSupplier(t)
		supplier.ClearDomainEvents()

		err := supplier.Block()
		require.NoError(t, err)
		assert.Equal(t, SupplierStatusBlocked, supplier.Status)

		events := supplier.GetDomainEvents()
		require.Len(t, events, 1)
	})

	t.Run("block already blocked fails", func(t *testing.T) {
		supplier := createTestSupplier(t)
		supplier.Status = SupplierStatusBlocked
		err := supplier.Block()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already blocked")
	})
}

func TestSupplier_StatusQueries(t *testing.T) {
	supplier := createTestSupplier(t)

	t.Run("IsActive", func(t *testing.T) {
		supplier.Status = SupplierStatusActive
		assert.True(t, supplier.IsActive())
		assert.False(t, supplier.IsInactive())
		assert.False(t, supplier.IsBlocked())
	})

	t.Run("IsInactive", func(t *testing.T) {
		supplier.Status = SupplierStatusInactive
		assert.False(t, supplier.IsActive())
		assert.True(t, supplier.IsInactive())
		assert.False(t, supplier.IsBlocked())
	})

	t.Run("IsBlocked", func(t *testing.T) {
		supplier.Status = SupplierStatusBlocked
		assert.False(t, supplier.IsActive())
		assert.False(t, supplier.IsInactive())
		assert.True(t, supplier.IsBlocked())
	})
}

func TestSupplier_TypeQueries(t *testing.T) {
	tenantID := uuid.New()

	t.Run("IsManufacturer", func(t *testing.T) {
		supplier, _ := NewSupplier(tenantID, "MFG001", "Manufacturer", SupplierTypeManufacturer)
		assert.True(t, supplier.IsManufacturer())
		assert.False(t, supplier.IsDistributor())
	})

	t.Run("IsDistributor", func(t *testing.T) {
		supplier, _ := NewSupplier(tenantID, "DST001", "Distributor", SupplierTypeDistributor)
		assert.False(t, supplier.IsManufacturer())
		assert.True(t, supplier.IsDistributor())
	})
}

func TestSupplier_CreditQueries(t *testing.T) {
	supplier := createTestSupplier(t)

	t.Run("HasCreditTerms", func(t *testing.T) {
		assert.False(t, supplier.HasCreditTerms())

		supplier.CreditDays = 30
		assert.True(t, supplier.HasCreditTerms())

		supplier.CreditDays = 0
		supplier.CreditLimit = decimal.NewFromInt(10000)
		assert.True(t, supplier.HasCreditTerms())
	})

	t.Run("HasBalance", func(t *testing.T) {
		assert.False(t, supplier.HasBalance())

		supplier.Balance = decimal.NewFromInt(100)
		assert.True(t, supplier.HasBalance())
	})

	t.Run("GetAvailableCredit", func(t *testing.T) {
		// No credit limit set
		supplier.CreditLimit = decimal.Zero
		assert.True(t, supplier.GetAvailableCredit().IsZero())

		// Credit limit with no balance
		supplier.CreditLimit = decimal.NewFromInt(10000)
		supplier.Balance = decimal.Zero
		assert.True(t, supplier.GetAvailableCredit().Equal(decimal.NewFromInt(10000)))

		// Credit limit with partial balance
		supplier.Balance = decimal.NewFromInt(3000)
		assert.True(t, supplier.GetAvailableCredit().Equal(decimal.NewFromInt(7000)))

		// Balance exceeds limit
		supplier.Balance = decimal.NewFromInt(15000)
		assert.True(t, supplier.GetAvailableCredit().IsZero())
	})

	t.Run("IsOverCreditLimit", func(t *testing.T) {
		supplier.CreditLimit = decimal.Zero
		supplier.Balance = decimal.NewFromInt(10000)
		assert.False(t, supplier.IsOverCreditLimit()) // No limit set

		supplier.CreditLimit = decimal.NewFromInt(5000)
		assert.True(t, supplier.IsOverCreditLimit()) // Over limit

		supplier.Balance = decimal.NewFromInt(3000)
		assert.False(t, supplier.IsOverCreditLimit()) // Under limit
	})
}

func TestSupplier_GetFullAddress(t *testing.T) {
	supplier := createTestSupplier(t)
	supplier.SetAddress("123 Main Street", "Shanghai", "Shanghai", "200000", "中国")

	fullAddress := supplier.GetFullAddress()
	assert.Equal(t, "中国 Shanghai Shanghai 123 Main Street 200000", fullAddress)
}

func TestSupplierAddressVO(t *testing.T) {
	t.Run("GetAddressVO returns Address value object", func(t *testing.T) {
		supplier := createTestSupplier(t)
		supplier.SetAddress("科技园南路123号", "深圳市", "广东省", "518000", "中国")

		addr := supplier.GetAddressVO()

		assert.False(t, addr.IsEmpty())
		assert.Equal(t, "广东省", addr.Province())
		assert.Equal(t, "深圳市", addr.City())
		assert.Equal(t, "科技园南路123号", addr.Detail())
		assert.Equal(t, "518000", addr.PostalCode())
	})

	t.Run("GetAddressVO returns empty for supplier without address", func(t *testing.T) {
		supplier := createTestSupplier(t)

		addr := supplier.GetAddressVO()

		assert.True(t, addr.IsEmpty())
	})

	t.Run("SetAddressVO sets address from value object", func(t *testing.T) {
		supplier := createTestSupplier(t)
		addr := valueobject.MustNewAddress("广东省", "深圳市", "南山区", "科技园南路123号",
			valueobject.WithPostalCode("518000"), valueobject.WithCountry("中国"))

		supplier.SetAddressVO(addr)

		assert.Equal(t, "广东省", supplier.Province)
		assert.Equal(t, "深圳市", supplier.City)
		assert.Contains(t, supplier.Address, "南山区")
		assert.Contains(t, supplier.Address, "科技园南路123号")
		assert.Equal(t, "518000", supplier.PostalCode)
		assert.Equal(t, "中国", supplier.Country)
	})

	t.Run("SetAddressVO clears address with empty value object", func(t *testing.T) {
		supplier := createTestSupplier(t)
		supplier.SetAddress("Some Address", "City", "Province", "100000", "中国")

		supplier.SetAddressVO(valueobject.EmptyAddress())

		assert.Equal(t, "", supplier.Province)
		assert.Equal(t, "", supplier.City)
		assert.Equal(t, "", supplier.Address)
		assert.Equal(t, "", supplier.PostalCode)
		// Country should be kept for default
		assert.Equal(t, "中国", supplier.Country)
	})
}

func TestSupplier_TableName(t *testing.T) {
	supplier := &Supplier{}
	assert.Equal(t, "suppliers", supplier.TableName())
}

// Helper function to create a test supplier
func createTestSupplier(t *testing.T) *Supplier {
	t.Helper()
	tenantID := uuid.New()
	supplier, err := NewSupplier(tenantID, "TEST001", "Test Supplier", SupplierTypeDistributor)
	require.NoError(t, err)
	supplier.ClearDomainEvents()
	return supplier
}
