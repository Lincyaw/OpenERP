package partner

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWarehouse(t *testing.T) {
	tenantID := uuid.New()

	t.Run("creates warehouse with valid input", func(t *testing.T) {
		warehouse, err := NewWarehouse(tenantID, "WH001", "Main Warehouse", WarehouseTypePhysical)
		require.NoError(t, err)
		require.NotNil(t, warehouse)

		assert.NotEqual(t, uuid.Nil, warehouse.ID)
		assert.Equal(t, tenantID, warehouse.TenantID)
		assert.Equal(t, "WH001", warehouse.Code)
		assert.Equal(t, "Main Warehouse", warehouse.Name)
		assert.Equal(t, WarehouseTypePhysical, warehouse.Type)
		assert.Equal(t, WarehouseStatusActive, warehouse.Status)
		assert.False(t, warehouse.IsDefault)
		assert.Equal(t, 0, warehouse.Capacity)
		assert.Equal(t, "中国", warehouse.Country)
		assert.Equal(t, "{}", warehouse.Attributes)

		// Should have created event
		events := warehouse.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeWarehouseCreated, events[0].EventType())
	})

	t.Run("converts code to uppercase", func(t *testing.T) {
		warehouse, err := NewWarehouse(tenantID, "wh001", "Test Warehouse", WarehouseTypePhysical)
		require.NoError(t, err)
		assert.Equal(t, "WH001", warehouse.Code)
	})

	t.Run("fails with empty code", func(t *testing.T) {
		warehouse, err := NewWarehouse(tenantID, "", "Test Warehouse", WarehouseTypePhysical)
		assert.Nil(t, warehouse)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("fails with empty name", func(t *testing.T) {
		warehouse, err := NewWarehouse(tenantID, "WH001", "", WarehouseTypePhysical)
		assert.Nil(t, warehouse)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("fails with invalid type", func(t *testing.T) {
		warehouse, err := NewWarehouse(tenantID, "WH001", "Test Warehouse", WarehouseType("invalid"))
		assert.Nil(t, warehouse)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid warehouse type")
	})

	t.Run("fails with invalid characters in code", func(t *testing.T) {
		warehouse, err := NewWarehouse(tenantID, "WH@001", "Test Warehouse", WarehouseTypePhysical)
		assert.Nil(t, warehouse)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only contain letters, numbers, underscores, and hyphens")
	})

	t.Run("fails with code too long", func(t *testing.T) {
		longCode := make([]byte, 51)
		for i := range longCode {
			longCode[i] = 'A'
		}
		warehouse, err := NewWarehouse(tenantID, string(longCode), "Test Warehouse", WarehouseTypePhysical)
		assert.Nil(t, warehouse)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot exceed 50 characters")
	})

	t.Run("fails with name too long", func(t *testing.T) {
		longName := make([]byte, 201)
		for i := range longName {
			longName[i] = 'a'
		}
		warehouse, err := NewWarehouse(tenantID, "WH001", string(longName), WarehouseTypePhysical)
		assert.Nil(t, warehouse)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot exceed 200 characters")
	})
}

func TestNewPhysicalWarehouse(t *testing.T) {
	tenantID := uuid.New()
	warehouse, err := NewPhysicalWarehouse(tenantID, "WH001", "Physical Warehouse")
	require.NoError(t, err)
	assert.Equal(t, WarehouseTypePhysical, warehouse.Type)
}

func TestNewVirtualWarehouse(t *testing.T) {
	tenantID := uuid.New()
	warehouse, err := NewVirtualWarehouse(tenantID, "VWH001", "Virtual Warehouse")
	require.NoError(t, err)
	assert.Equal(t, WarehouseTypeVirtual, warehouse.Type)
}

func TestNewConsignWarehouse(t *testing.T) {
	tenantID := uuid.New()
	warehouse, err := NewConsignWarehouse(tenantID, "CWH001", "Consignment Warehouse")
	require.NoError(t, err)
	assert.Equal(t, WarehouseTypeConsign, warehouse.Type)
}

func TestNewTransitWarehouse(t *testing.T) {
	tenantID := uuid.New()
	warehouse, err := NewTransitWarehouse(tenantID, "TWH001", "Transit Warehouse")
	require.NoError(t, err)
	assert.Equal(t, WarehouseTypeTransit, warehouse.Type)
}

func TestWarehouse_Update(t *testing.T) {
	warehouse := createTestWarehouse(t)

	t.Run("updates name and short name", func(t *testing.T) {
		warehouse.ClearDomainEvents()
		err := warehouse.Update("New Name", "New Short")
		require.NoError(t, err)
		assert.Equal(t, "New Name", warehouse.Name)
		assert.Equal(t, "New Short", warehouse.ShortName)

		events := warehouse.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeWarehouseUpdated, events[0].EventType())
	})

	t.Run("fails with empty name", func(t *testing.T) {
		err := warehouse.Update("", "Short")
		assert.Error(t, err)
	})

	t.Run("fails with short name too long", func(t *testing.T) {
		longName := make([]byte, 101)
		for i := range longName {
			longName[i] = 'a'
		}
		err := warehouse.Update("Valid Name", string(longName))
		assert.Error(t, err)
	})
}

func TestWarehouse_UpdateCode(t *testing.T) {
	warehouse := createTestWarehouse(t)

	t.Run("updates code", func(t *testing.T) {
		warehouse.ClearDomainEvents()
		err := warehouse.UpdateCode("NEW001")
		require.NoError(t, err)
		assert.Equal(t, "NEW001", warehouse.Code)

		events := warehouse.GetDomainEvents()
		require.Len(t, events, 1)
	})

	t.Run("converts to uppercase", func(t *testing.T) {
		err := warehouse.UpdateCode("new002")
		require.NoError(t, err)
		assert.Equal(t, "NEW002", warehouse.Code)
	})

	t.Run("fails with empty code", func(t *testing.T) {
		err := warehouse.UpdateCode("")
		assert.Error(t, err)
	})
}

func TestWarehouse_SetContact(t *testing.T) {
	warehouse := createTestWarehouse(t)

	t.Run("sets contact information", func(t *testing.T) {
		err := warehouse.SetContact("John Doe", "1234567890", "john@example.com")
		require.NoError(t, err)
		assert.Equal(t, "John Doe", warehouse.ContactName)
		assert.Equal(t, "1234567890", warehouse.Phone)
		assert.Equal(t, "john@example.com", warehouse.Email)
	})

	t.Run("allows empty values", func(t *testing.T) {
		err := warehouse.SetContact("", "", "")
		require.NoError(t, err)
		assert.Empty(t, warehouse.ContactName)
		assert.Empty(t, warehouse.Phone)
		assert.Empty(t, warehouse.Email)
	})

	t.Run("fails with invalid phone", func(t *testing.T) {
		err := warehouse.SetContact("Jane", "invalid@phone", "jane@example.com")
		assert.Error(t, err)
	})

	t.Run("fails with invalid email", func(t *testing.T) {
		err := warehouse.SetContact("Jane", "1234567890", "invalid-email")
		assert.Error(t, err)
	})

	t.Run("fails with contact name too long", func(t *testing.T) {
		longName := make([]byte, 101)
		for i := range longName {
			longName[i] = 'a'
		}
		err := warehouse.SetContact(string(longName), "1234567890", "test@example.com")
		assert.Error(t, err)
	})
}

func TestWarehouse_SetAddress(t *testing.T) {
	warehouse := createTestWarehouse(t)

	t.Run("sets address information", func(t *testing.T) {
		err := warehouse.SetAddress("123 Main St", "Shanghai", "Shanghai", "200000", "China")
		require.NoError(t, err)
		assert.Equal(t, "123 Main St", warehouse.Address)
		assert.Equal(t, "Shanghai", warehouse.City)
		assert.Equal(t, "Shanghai", warehouse.Province)
		assert.Equal(t, "200000", warehouse.PostalCode)
		assert.Equal(t, "China", warehouse.Country)
	})

	t.Run("keeps default country when empty", func(t *testing.T) {
		warehouse.Country = "中国"
		err := warehouse.SetAddress("456 Other St", "Beijing", "Beijing", "100000", "")
		require.NoError(t, err)
		assert.Equal(t, "中国", warehouse.Country)
	})

	t.Run("fails with address too long", func(t *testing.T) {
		longAddress := make([]byte, 501)
		for i := range longAddress {
			longAddress[i] = 'a'
		}
		err := warehouse.SetAddress(string(longAddress), "City", "Province", "12345", "Country")
		assert.Error(t, err)
	})

	t.Run("fails with city too long", func(t *testing.T) {
		longCity := make([]byte, 101)
		for i := range longCity {
			longCity[i] = 'a'
		}
		err := warehouse.SetAddress("Address", string(longCity), "Province", "12345", "Country")
		assert.Error(t, err)
	})

	t.Run("fails with province too long", func(t *testing.T) {
		longProvince := make([]byte, 101)
		for i := range longProvince {
			longProvince[i] = 'a'
		}
		err := warehouse.SetAddress("Address", "City", string(longProvince), "12345", "Country")
		assert.Error(t, err)
	})

	t.Run("fails with postal code too long", func(t *testing.T) {
		longPostalCode := make([]byte, 21)
		for i := range longPostalCode {
			longPostalCode[i] = '1'
		}
		err := warehouse.SetAddress("Address", "City", "Province", string(longPostalCode), "Country")
		assert.Error(t, err)
	})

	t.Run("fails with country too long", func(t *testing.T) {
		longCountry := make([]byte, 101)
		for i := range longCountry {
			longCountry[i] = 'a'
		}
		err := warehouse.SetAddress("Address", "City", "Province", "12345", string(longCountry))
		assert.Error(t, err)
	})
}

func TestWarehouse_SetDefault(t *testing.T) {
	warehouse := createTestWarehouse(t)

	t.Run("sets warehouse as default", func(t *testing.T) {
		warehouse.ClearDomainEvents()
		warehouse.SetDefault(true)
		assert.True(t, warehouse.IsDefault)

		events := warehouse.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeWarehouseSetAsDefault, events[0].EventType())
	})

	t.Run("clears default without event", func(t *testing.T) {
		warehouse.ClearDomainEvents()
		warehouse.SetDefault(false)
		assert.False(t, warehouse.IsDefault)

		events := warehouse.GetDomainEvents()
		assert.Len(t, events, 0)
	})
}

func TestWarehouse_SetCapacity(t *testing.T) {
	warehouse := createTestWarehouse(t)

	t.Run("sets capacity", func(t *testing.T) {
		err := warehouse.SetCapacity(1000)
		require.NoError(t, err)
		assert.Equal(t, 1000, warehouse.Capacity)
	})

	t.Run("allows zero capacity", func(t *testing.T) {
		err := warehouse.SetCapacity(0)
		require.NoError(t, err)
		assert.Equal(t, 0, warehouse.Capacity)
	})

	t.Run("fails with negative capacity", func(t *testing.T) {
		err := warehouse.SetCapacity(-1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be negative")
	})
}

func TestWarehouse_SetNotes(t *testing.T) {
	warehouse := createTestWarehouse(t)
	warehouse.SetNotes("Some notes about the warehouse")
	assert.Equal(t, "Some notes about the warehouse", warehouse.Notes)
}

func TestWarehouse_SetSortOrder(t *testing.T) {
	warehouse := createTestWarehouse(t)
	warehouse.SetSortOrder(5)
	assert.Equal(t, 5, warehouse.SortOrder)
}

func TestWarehouse_SetAttributes(t *testing.T) {
	warehouse := createTestWarehouse(t)

	t.Run("sets valid JSON attributes", func(t *testing.T) {
		err := warehouse.SetAttributes(`{"temperature": "cold", "zones": 3}`)
		require.NoError(t, err)
		assert.Equal(t, `{"temperature": "cold", "zones": 3}`, warehouse.Attributes)
	})

	t.Run("handles empty string by setting default", func(t *testing.T) {
		err := warehouse.SetAttributes("")
		require.NoError(t, err)
		assert.Equal(t, "{}", warehouse.Attributes)
	})

	t.Run("fails with invalid JSON", func(t *testing.T) {
		err := warehouse.SetAttributes("not-json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be valid JSON object")
	})

	t.Run("fails with JSON array", func(t *testing.T) {
		err := warehouse.SetAttributes(`["a", "b"]`)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be valid JSON object")
	})
}

func TestWarehouse_Enable(t *testing.T) {
	warehouse := createTestWarehouse(t)
	warehouse.Status = WarehouseStatusInactive

	t.Run("enables inactive warehouse", func(t *testing.T) {
		warehouse.ClearDomainEvents()
		err := warehouse.Enable()
		require.NoError(t, err)
		assert.Equal(t, WarehouseStatusActive, warehouse.Status)

		events := warehouse.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeWarehouseStatusChanged, events[0].EventType())
	})

	t.Run("fails when already active", func(t *testing.T) {
		err := warehouse.Enable()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already active")
	})
}

func TestWarehouse_Disable(t *testing.T) {
	warehouse := createTestWarehouse(t)

	t.Run("disables active warehouse", func(t *testing.T) {
		warehouse.ClearDomainEvents()
		err := warehouse.Disable()
		require.NoError(t, err)
		assert.Equal(t, WarehouseStatusInactive, warehouse.Status)

		events := warehouse.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeWarehouseStatusChanged, events[0].EventType())
	})

	t.Run("fails when already inactive", func(t *testing.T) {
		err := warehouse.Disable()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already inactive")
	})

	t.Run("fails when warehouse is default", func(t *testing.T) {
		warehouse.Status = WarehouseStatusActive
		warehouse.IsDefault = true
		err := warehouse.Disable()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot disable the default warehouse")
	})
}

func TestWarehouse_StatusChecks(t *testing.T) {
	warehouse := createTestWarehouse(t)

	t.Run("IsActive returns true when active", func(t *testing.T) {
		warehouse.Status = WarehouseStatusActive
		assert.True(t, warehouse.IsActive())
		assert.False(t, warehouse.IsInactive())
	})

	t.Run("IsInactive returns true when inactive", func(t *testing.T) {
		warehouse.Status = WarehouseStatusInactive
		assert.False(t, warehouse.IsActive())
		assert.True(t, warehouse.IsInactive())
	})
}

func TestWarehouse_TypeChecks(t *testing.T) {
	tenantID := uuid.New()

	t.Run("IsPhysical returns true for physical warehouse", func(t *testing.T) {
		warehouse, _ := NewPhysicalWarehouse(tenantID, "WH001", "Physical")
		assert.True(t, warehouse.IsPhysical())
		assert.False(t, warehouse.IsVirtual())
		assert.False(t, warehouse.IsConsign())
		assert.False(t, warehouse.IsTransit())
	})

	t.Run("IsVirtual returns true for virtual warehouse", func(t *testing.T) {
		warehouse, _ := NewVirtualWarehouse(tenantID, "VWH001", "Virtual")
		assert.False(t, warehouse.IsPhysical())
		assert.True(t, warehouse.IsVirtual())
		assert.False(t, warehouse.IsConsign())
		assert.False(t, warehouse.IsTransit())
	})

	t.Run("IsConsign returns true for consignment warehouse", func(t *testing.T) {
		warehouse, _ := NewConsignWarehouse(tenantID, "CWH001", "Consign")
		assert.False(t, warehouse.IsPhysical())
		assert.False(t, warehouse.IsVirtual())
		assert.True(t, warehouse.IsConsign())
		assert.False(t, warehouse.IsTransit())
	})

	t.Run("IsTransit returns true for transit warehouse", func(t *testing.T) {
		warehouse, _ := NewTransitWarehouse(tenantID, "TWH001", "Transit")
		assert.False(t, warehouse.IsPhysical())
		assert.False(t, warehouse.IsVirtual())
		assert.False(t, warehouse.IsConsign())
		assert.True(t, warehouse.IsTransit())
	})
}

func TestWarehouse_HasCapacity(t *testing.T) {
	warehouse := createTestWarehouse(t)

	t.Run("returns false when capacity is zero", func(t *testing.T) {
		warehouse.Capacity = 0
		assert.False(t, warehouse.HasCapacity())
	})

	t.Run("returns true when capacity is set", func(t *testing.T) {
		warehouse.Capacity = 1000
		assert.True(t, warehouse.HasCapacity())
	})
}

func TestWarehouse_GetFullAddress(t *testing.T) {
	warehouse := createTestWarehouse(t)

	t.Run("returns formatted full address", func(t *testing.T) {
		warehouse.Country = "China"
		warehouse.Province = "Shanghai"
		warehouse.City = "Shanghai"
		warehouse.Address = "123 Main St"
		warehouse.PostalCode = "200000"

		fullAddress := warehouse.GetFullAddress()
		assert.Contains(t, fullAddress, "China")
		assert.Contains(t, fullAddress, "Shanghai")
		assert.Contains(t, fullAddress, "123 Main St")
		assert.Contains(t, fullAddress, "200000")
	})

	t.Run("handles missing address parts", func(t *testing.T) {
		warehouse.Country = "China"
		warehouse.Province = ""
		warehouse.City = ""
		warehouse.Address = ""
		warehouse.PostalCode = ""

		fullAddress := warehouse.GetFullAddress()
		assert.Equal(t, "China", fullAddress)
	})

	t.Run("returns empty string when all parts empty", func(t *testing.T) {
		warehouse.Country = ""
		warehouse.Province = ""
		warehouse.City = ""
		warehouse.Address = ""
		warehouse.PostalCode = ""

		fullAddress := warehouse.GetFullAddress()
		assert.Empty(t, fullAddress)
	})
}

func TestWarehouseEvents(t *testing.T) {
	warehouse := createTestWarehouse(t)

	t.Run("created event has correct data", func(t *testing.T) {
		events := warehouse.GetDomainEvents()
		require.Len(t, events, 1)

		event, ok := events[0].(*WarehouseCreatedEvent)
		require.True(t, ok)
		assert.Equal(t, warehouse.ID, event.WarehouseID)
		assert.Equal(t, warehouse.Code, event.Code)
		assert.Equal(t, warehouse.Name, event.Name)
		assert.Equal(t, warehouse.Type, event.Type)
	})

	t.Run("updated event has correct data", func(t *testing.T) {
		warehouse.ClearDomainEvents()
		_ = warehouse.Update("Updated Name", "Short")

		events := warehouse.GetDomainEvents()
		require.Len(t, events, 1)

		event, ok := events[0].(*WarehouseUpdatedEvent)
		require.True(t, ok)
		assert.Equal(t, warehouse.ID, event.WarehouseID)
		assert.Equal(t, "Updated Name", event.Name)
		assert.Equal(t, "Short", event.ShortName)
	})

	t.Run("status changed event has correct data", func(t *testing.T) {
		warehouse.ClearDomainEvents()
		warehouse.Status = WarehouseStatusActive
		warehouse.IsDefault = false
		_ = warehouse.Disable()

		events := warehouse.GetDomainEvents()
		require.Len(t, events, 1)

		event, ok := events[0].(*WarehouseStatusChangedEvent)
		require.True(t, ok)
		assert.Equal(t, warehouse.ID, event.WarehouseID)
		assert.Equal(t, WarehouseStatusActive, event.OldStatus)
		assert.Equal(t, WarehouseStatusInactive, event.NewStatus)
	})

	t.Run("set as default event has correct data", func(t *testing.T) {
		warehouse.ClearDomainEvents()
		warehouse.SetDefault(true)

		events := warehouse.GetDomainEvents()
		require.Len(t, events, 1)

		event, ok := events[0].(*WarehouseSetAsDefaultEvent)
		require.True(t, ok)
		assert.Equal(t, warehouse.ID, event.WarehouseID)
		assert.Equal(t, warehouse.Code, event.Code)
		assert.Equal(t, warehouse.Name, event.Name)
	})
}

func TestWarehouse_ValidateWarehouseType(t *testing.T) {
	tests := []struct {
		warehouseType WarehouseType
		valid         bool
	}{
		{WarehouseTypePhysical, true},
		{WarehouseTypeVirtual, true},
		{WarehouseTypeConsign, true},
		{WarehouseTypeTransit, true},
		{WarehouseType("invalid"), false},
		{WarehouseType(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.warehouseType), func(t *testing.T) {
			err := validateWarehouseType(tt.warehouseType)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// Helper function to create a test warehouse
func createTestWarehouse(t *testing.T) *Warehouse {
	t.Helper()
	tenantID := uuid.New()
	warehouse, err := NewWarehouse(tenantID, "WH001", "Test Warehouse", WarehouseTypePhysical)
	require.NoError(t, err)
	return warehouse
}
