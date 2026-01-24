package identity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTenant(t *testing.T) {
	t.Run("creates tenant successfully", func(t *testing.T) {
		tenant, err := NewTenant("TENANT001", "Test Company")

		require.NoError(t, err)
		assert.NotNil(t, tenant)
		assert.Equal(t, "TENANT001", tenant.Code)
		assert.Equal(t, "Test Company", tenant.Name)
		assert.Equal(t, TenantStatusActive, tenant.Status)
		assert.Equal(t, TenantPlanFree, tenant.Plan)
		assert.Equal(t, 5, tenant.Config.MaxUsers)
		assert.Equal(t, 3, tenant.Config.MaxWarehouses)
		assert.Equal(t, 1000, tenant.Config.MaxProducts)
		assert.Equal(t, "CNY", tenant.Config.Currency)
		assert.Equal(t, "Asia/Shanghai", tenant.Config.Timezone)
		assert.Equal(t, "zh-CN", tenant.Config.Locale)
		assert.Len(t, tenant.GetDomainEvents(), 1)
	})

	t.Run("converts code to uppercase", func(t *testing.T) {
		tenant, err := NewTenant("tenant002", "Test Company")

		require.NoError(t, err)
		assert.Equal(t, "TENANT002", tenant.Code)
	})

	t.Run("fails with empty code", func(t *testing.T) {
		tenant, err := NewTenant("", "Test Company")

		assert.Error(t, err)
		assert.Nil(t, tenant)
		assert.Contains(t, err.Error(), "code cannot be empty")
	})

	t.Run("fails with invalid code characters", func(t *testing.T) {
		tenant, err := NewTenant("TENANT@001", "Test Company")

		assert.Error(t, err)
		assert.Nil(t, tenant)
		assert.Contains(t, err.Error(), "can only contain")
	})

	t.Run("fails with empty name", func(t *testing.T) {
		tenant, err := NewTenant("TENANT001", "")

		assert.Error(t, err)
		assert.Nil(t, tenant)
		assert.Contains(t, err.Error(), "name cannot be empty")
	})

	t.Run("fails with code exceeding max length", func(t *testing.T) {
		longCode := make([]byte, 51)
		for i := range longCode {
			longCode[i] = 'A'
		}
		tenant, err := NewTenant(string(longCode), "Test Company")

		assert.Error(t, err)
		assert.Nil(t, tenant)
		assert.Contains(t, err.Error(), "cannot exceed 50 characters")
	})
}

func TestNewTrialTenant(t *testing.T) {
	t.Run("creates trial tenant successfully", func(t *testing.T) {
		tenant, err := NewTrialTenant("TRIAL001", "Trial Company", 14)

		require.NoError(t, err)
		assert.NotNil(t, tenant)
		assert.Equal(t, TenantStatusTrial, tenant.Status)
		assert.NotNil(t, tenant.TrialEndsAt)
		assert.True(t, tenant.IsTrial())
	})

	t.Run("fails with zero trial days", func(t *testing.T) {
		tenant, err := NewTrialTenant("TRIAL001", "Trial Company", 0)

		assert.Error(t, err)
		assert.Nil(t, tenant)
		assert.Contains(t, err.Error(), "Trial days must be positive")
	})

	t.Run("fails with negative trial days", func(t *testing.T) {
		tenant, err := NewTrialTenant("TRIAL001", "Trial Company", -5)

		assert.Error(t, err)
		assert.Nil(t, tenant)
	})
}

func TestTenant_Update(t *testing.T) {
	t.Run("updates tenant successfully", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Original Name")
		tenant.ClearDomainEvents()
		initialVersion := tenant.Version

		err := tenant.Update("Updated Name", "Short")

		require.NoError(t, err)
		assert.Equal(t, "Updated Name", tenant.Name)
		assert.Equal(t, "Short", tenant.ShortName)
		assert.Equal(t, initialVersion+1, tenant.Version)
		assert.Len(t, tenant.GetDomainEvents(), 1)
	})

	t.Run("fails with empty name", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Original Name")

		err := tenant.Update("", "Short")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name cannot be empty")
	})

	t.Run("fails with long short name", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Original Name")
		longShortName := make([]byte, 101)
		for i := range longShortName {
			longShortName[i] = 'A'
		}

		err := tenant.Update("New Name", string(longShortName))

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Short name cannot exceed 100 characters")
	})
}

func TestTenant_SetContact(t *testing.T) {
	t.Run("sets contact successfully", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")

		err := tenant.SetContact("John Doe", "+86 13800138000", "john@example.com")

		require.NoError(t, err)
		assert.Equal(t, "John Doe", tenant.ContactName)
		assert.Equal(t, "+86 13800138000", tenant.ContactPhone)
		assert.Equal(t, "john@example.com", tenant.ContactEmail)
	})

	t.Run("fails with long contact name", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")
		longName := make([]byte, 101)
		for i := range longName {
			longName[i] = 'A'
		}

		err := tenant.SetContact(string(longName), "", "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Contact name cannot exceed 100 characters")
	})
}

func TestTenant_SetPlan(t *testing.T) {
	t.Run("sets plan and updates config", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")
		tenant.ClearDomainEvents()

		err := tenant.SetPlan(TenantPlanPro)

		require.NoError(t, err)
		assert.Equal(t, TenantPlanPro, tenant.Plan)
		assert.Equal(t, 50, tenant.Config.MaxUsers)
		assert.Equal(t, 20, tenant.Config.MaxWarehouses)
		assert.Equal(t, 50000, tenant.Config.MaxProducts)
		assert.Len(t, tenant.GetDomainEvents(), 1)
	})

	t.Run("upgrades from trial clears trial status", func(t *testing.T) {
		tenant, _ := NewTrialTenant("TRIAL001", "Trial Company", 14)
		assert.Equal(t, TenantStatusTrial, tenant.Status)

		err := tenant.SetPlan(TenantPlanBasic)

		require.NoError(t, err)
		assert.Equal(t, TenantStatusActive, tenant.Status)
		assert.Nil(t, tenant.TrialEndsAt)
	})

	t.Run("fails with invalid plan", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")

		err := tenant.SetPlan(TenantPlan("invalid"))

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid tenant plan")
	})

	t.Run("enterprise plan has unlimited resources", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")

		err := tenant.SetPlan(TenantPlanEnterprise)

		require.NoError(t, err)
		assert.Equal(t, 9999, tenant.Config.MaxUsers)
		assert.Equal(t, 9999, tenant.Config.MaxWarehouses)
		assert.Equal(t, 999999, tenant.Config.MaxProducts)
	})
}

func TestTenant_StatusTransitions(t *testing.T) {
	t.Run("activate tenant", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")
		tenant.Status = TenantStatusInactive
		tenant.ClearDomainEvents()

		err := tenant.Activate()

		require.NoError(t, err)
		assert.Equal(t, TenantStatusActive, tenant.Status)
		assert.True(t, tenant.IsActive())
		assert.Len(t, tenant.GetDomainEvents(), 1)
	})

	t.Run("fails to activate already active tenant", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")

		err := tenant.Activate()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already active")
	})

	t.Run("deactivate tenant", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")
		tenant.ClearDomainEvents()

		err := tenant.Deactivate()

		require.NoError(t, err)
		assert.Equal(t, TenantStatusInactive, tenant.Status)
		assert.True(t, tenant.IsInactive())
		assert.Len(t, tenant.GetDomainEvents(), 1)
	})

	t.Run("fails to deactivate already inactive tenant", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")
		tenant.Status = TenantStatusInactive

		err := tenant.Deactivate()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already inactive")
	})

	t.Run("suspend tenant", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")
		tenant.ClearDomainEvents()

		err := tenant.Suspend()

		require.NoError(t, err)
		assert.Equal(t, TenantStatusSuspended, tenant.Status)
		assert.True(t, tenant.IsSuspended())
		assert.Len(t, tenant.GetDomainEvents(), 1)
	})

	t.Run("fails to suspend already suspended tenant", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")
		tenant.Status = TenantStatusSuspended

		err := tenant.Suspend()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already suspended")
	})
}

func TestTenant_ConvertFromTrial(t *testing.T) {
	t.Run("converts trial to paid successfully", func(t *testing.T) {
		tenant, _ := NewTrialTenant("TRIAL001", "Trial Company", 14)

		err := tenant.ConvertFromTrial(TenantPlanPro)

		require.NoError(t, err)
		assert.Equal(t, TenantPlanPro, tenant.Plan)
		assert.Equal(t, TenantStatusActive, tenant.Status)
		assert.Nil(t, tenant.TrialEndsAt)
	})

	t.Run("fails to convert non-trial tenant", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")

		err := tenant.ConvertFromTrial(TenantPlanPro)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not in trial status")
	})

	t.Run("fails to convert to free plan", func(t *testing.T) {
		tenant, _ := NewTrialTenant("TRIAL001", "Trial Company", 14)

		err := tenant.ConvertFromTrial(TenantPlanFree)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot convert to free plan")
	})
}

func TestTenant_ExpirationChecks(t *testing.T) {
	t.Run("trial expired", func(t *testing.T) {
		tenant, _ := NewTrialTenant("TRIAL001", "Trial Company", 1)
		// Set trial end to the past
		pastDate := time.Now().AddDate(0, 0, -1)
		tenant.TrialEndsAt = &pastDate

		assert.True(t, tenant.IsTrialExpired())
	})

	t.Run("trial not expired", func(t *testing.T) {
		tenant, _ := NewTrialTenant("TRIAL001", "Trial Company", 14)

		assert.False(t, tenant.IsTrialExpired())
	})

	t.Run("non-trial tenant is not expired", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")

		assert.False(t, tenant.IsTrialExpired())
	})

	t.Run("subscription expired", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")
		pastDate := time.Now().AddDate(0, 0, -1)
		tenant.ExpiresAt = &pastDate

		assert.True(t, tenant.IsSubscriptionExpired())
	})

	t.Run("subscription not expired", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")
		futureDate := time.Now().AddDate(0, 1, 0)
		tenant.ExpiresAt = &futureDate

		assert.False(t, tenant.IsSubscriptionExpired())
	})

	t.Run("no expiration set is not expired", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")

		assert.False(t, tenant.IsSubscriptionExpired())
	})
}

func TestTenant_ResourceLimits(t *testing.T) {
	t.Run("can add user within limit", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")

		assert.True(t, tenant.CanAddUser(4))
		assert.False(t, tenant.CanAddUser(5))
		assert.False(t, tenant.CanAddUser(10))
	})

	t.Run("can add warehouse within limit", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")

		assert.True(t, tenant.CanAddWarehouse(2))
		assert.False(t, tenant.CanAddWarehouse(3))
	})

	t.Run("can add product within limit", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")

		assert.True(t, tenant.CanAddProduct(999))
		assert.False(t, tenant.CanAddProduct(1000))
	})
}

func TestTenant_SetExpiration(t *testing.T) {
	t.Run("sets expiration date", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")
		futureDate := time.Now().AddDate(1, 0, 0)

		tenant.SetExpiration(futureDate)

		assert.NotNil(t, tenant.ExpiresAt)
		assert.Equal(t, futureDate.Unix(), tenant.ExpiresAt.Unix())
	})

	t.Run("clears expiration date", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")
		futureDate := time.Now().AddDate(1, 0, 0)
		tenant.SetExpiration(futureDate)

		tenant.ClearExpiration()

		assert.Nil(t, tenant.ExpiresAt)
	})
}

func TestTenant_UpdateConfig(t *testing.T) {
	t.Run("updates config successfully", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")
		config := TenantConfig{
			MaxUsers:      100,
			MaxWarehouses: 50,
			MaxProducts:   10000,
			Currency:      "USD",
			Timezone:      "America/New_York",
			Locale:        "en-US",
		}

		err := tenant.UpdateConfig(config)

		require.NoError(t, err)
		assert.Equal(t, 100, tenant.Config.MaxUsers)
		assert.Equal(t, "USD", tenant.Config.Currency)
		assert.Equal(t, "en-US", tenant.Config.Locale)
	})

	t.Run("fails with negative max users", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")
		config := TenantConfig{
			MaxUsers: -1,
		}

		err := tenant.UpdateConfig(config)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Max users cannot be negative")
	})

	t.Run("fails with negative max warehouses", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")
		config := TenantConfig{
			MaxWarehouses: -1,
		}

		err := tenant.UpdateConfig(config)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Max warehouses cannot be negative")
	})

	t.Run("fails with negative max products", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")
		config := TenantConfig{
			MaxProducts: -1,
		}

		err := tenant.UpdateConfig(config)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Max products cannot be negative")
	})
}

func TestTenant_SetAddress(t *testing.T) {
	t.Run("sets address successfully", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")

		err := tenant.SetAddress("123 Main Street, City, Province")

		require.NoError(t, err)
		assert.Equal(t, "123 Main Street, City, Province", tenant.Address)
	})

	t.Run("fails with long address", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")
		longAddress := make([]byte, 501)
		for i := range longAddress {
			longAddress[i] = 'A'
		}

		err := tenant.SetAddress(string(longAddress))

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Address cannot exceed 500 characters")
	})
}

func TestTenant_SetLogoURL(t *testing.T) {
	t.Run("sets logo URL successfully", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")

		err := tenant.SetLogoURL("https://example.com/logo.png")

		require.NoError(t, err)
		assert.Equal(t, "https://example.com/logo.png", tenant.LogoURL)
	})

	t.Run("fails with long URL", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")
		longURL := make([]byte, 501)
		for i := range longURL {
			longURL[i] = 'A'
		}

		err := tenant.SetLogoURL(string(longURL))

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Logo URL cannot exceed 500 characters")
	})
}

func TestTenant_SetDomain(t *testing.T) {
	t.Run("sets domain successfully", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")

		err := tenant.SetDomain("mycompany.example.com")

		require.NoError(t, err)
		assert.Equal(t, "mycompany.example.com", tenant.Domain)
	})

	t.Run("converts domain to lowercase", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")

		err := tenant.SetDomain("MyCompany.Example.Com")

		require.NoError(t, err)
		assert.Equal(t, "mycompany.example.com", tenant.Domain)
	})

	t.Run("fails with long domain", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")
		longDomain := make([]byte, 201)
		for i := range longDomain {
			longDomain[i] = 'a'
		}

		err := tenant.SetDomain(string(longDomain))

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Domain cannot exceed 200 characters")
	})
}

func TestTenant_SetNotes(t *testing.T) {
	t.Run("sets notes successfully", func(t *testing.T) {
		tenant, _ := NewTenant("TENANT001", "Test Company")

		tenant.SetNotes("Some important notes about this tenant")

		assert.Equal(t, "Some important notes about this tenant", tenant.Notes)
	})
}

func TestDefaultTenantConfig(t *testing.T) {
	config := DefaultTenantConfig()

	assert.Equal(t, 5, config.MaxUsers)
	assert.Equal(t, 3, config.MaxWarehouses)
	assert.Equal(t, 1000, config.MaxProducts)
	assert.Equal(t, "{}", config.Features)
	assert.Equal(t, "{}", config.Settings)
	assert.Equal(t, "weighted_average", config.CostStrategy)
	assert.Equal(t, "CNY", config.Currency)
	assert.Equal(t, "Asia/Shanghai", config.Timezone)
	assert.Equal(t, "zh-CN", config.Locale)
}

func TestTenant_GetTenantID(t *testing.T) {
	tenant, _ := NewTenant("TENANT001", "Test Company")

	tenantID := tenant.GetTenantID()

	assert.Equal(t, tenant.ID, tenantID)
}
