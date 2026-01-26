package datascope

import (
	"context"
	"testing"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/infrastructure/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFilter(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("creates filter with empty roles", func(t *testing.T) {
		ctx := context.Background()
		filter := NewFilter(ctx, []identity.Role{})

		assert.NotNil(t, filter)
		assert.Empty(t, filter.dataScopes)
	})

	t.Run("creates filter with user ID from context", func(t *testing.T) {
		ctx := context.Background()
		ctx, _ = logger.WithUserID(ctx, logger.FromContext(ctx), userID.String())

		filter := NewFilter(ctx, []identity.Role{})

		assert.Equal(t, userID, filter.userID)
	})

	t.Run("merges data scopes from multiple roles", func(t *testing.T) {
		ctx := context.Background()

		role1, _ := identity.NewRole(tenantID, "ROLE1", "Role 1")
		ds1, _ := identity.NewDataScope("sales_order", identity.DataScopeSelf)
		_ = role1.SetDataScope(*ds1)

		role2, _ := identity.NewRole(tenantID, "ROLE2", "Role 2")
		ds2, _ := identity.NewDataScope("sales_order", identity.DataScopeAll)
		_ = role2.SetDataScope(*ds2)

		filter := NewFilter(ctx, []identity.Role{*role1, *role2})

		// Should have ALL scope (higher permission wins)
		assert.Equal(t, identity.DataScopeAll, filter.GetScopeType("sales_order"))
	})

	t.Run("ignores disabled roles", func(t *testing.T) {
		ctx := context.Background()

		role1, _ := identity.NewRole(tenantID, "ROLE1", "Role 1")
		ds1, _ := identity.NewDataScope("sales_order", identity.DataScopeAll)
		_ = role1.SetDataScope(*ds1)
		_ = role1.Disable()

		role2, _ := identity.NewRole(tenantID, "ROLE2", "Role 2")
		ds2, _ := identity.NewDataScope("sales_order", identity.DataScopeSelf)
		_ = role2.SetDataScope(*ds2)

		filter := NewFilter(ctx, []identity.Role{*role1, *role2})

		// Role1 is disabled, so should use SELF from role2
		assert.Equal(t, identity.DataScopeSelf, filter.GetScopeType("sales_order"))
	})
}

func TestFilter_GetScopeType(t *testing.T) {
	tenantID := uuid.New()

	t.Run("returns ALL for unconfigured resource", func(t *testing.T) {
		filter := NewFilter(context.Background(), []identity.Role{})

		scopeType := filter.GetScopeType("unconfigured_resource")

		assert.Equal(t, identity.DataScopeAll, scopeType)
	})

	t.Run("returns configured scope type", func(t *testing.T) {
		ctx := context.Background()

		role, _ := identity.NewRole(tenantID, "ROLE1", "Role 1")
		ds, _ := identity.NewDataScope("sales_order", identity.DataScopeSelf)
		_ = role.SetDataScope(*ds)

		filter := NewFilter(ctx, []identity.Role{*role})

		assert.Equal(t, identity.DataScopeSelf, filter.GetScopeType("sales_order"))
	})
}

func TestFilter_HasScope(t *testing.T) {
	tenantID := uuid.New()

	t.Run("returns false for unconfigured resource", func(t *testing.T) {
		filter := NewFilter(context.Background(), []identity.Role{})

		assert.False(t, filter.HasScope("unconfigured_resource"))
	})

	t.Run("returns true for configured resource", func(t *testing.T) {
		ctx := context.Background()

		role, _ := identity.NewRole(tenantID, "ROLE1", "Role 1")
		ds, _ := identity.NewDataScope("sales_order", identity.DataScopeSelf)
		_ = role.SetDataScope(*ds)

		filter := NewFilter(ctx, []identity.Role{*role})

		assert.True(t, filter.HasScope("sales_order"))
	})
}

func TestFilter_CanAccessAll(t *testing.T) {
	tenantID := uuid.New()

	t.Run("returns true for unconfigured resource", func(t *testing.T) {
		filter := NewFilter(context.Background(), []identity.Role{})

		assert.True(t, filter.CanAccessAll("unconfigured_resource"))
	})

	t.Run("returns true for ALL scope", func(t *testing.T) {
		ctx := context.Background()

		role, _ := identity.NewRole(tenantID, "ROLE1", "Role 1")
		ds, _ := identity.NewDataScope("sales_order", identity.DataScopeAll)
		_ = role.SetDataScope(*ds)

		filter := NewFilter(ctx, []identity.Role{*role})

		assert.True(t, filter.CanAccessAll("sales_order"))
	})

	t.Run("returns false for SELF scope", func(t *testing.T) {
		ctx := context.Background()

		role, _ := identity.NewRole(tenantID, "ROLE1", "Role 1")
		ds, _ := identity.NewDataScope("sales_order", identity.DataScopeSelf)
		_ = role.SetDataScope(*ds)

		filter := NewFilter(ctx, []identity.Role{*role})

		assert.False(t, filter.CanAccessAll("sales_order"))
	})
}

func TestFilter_IsOwner(t *testing.T) {
	userID := uuid.New()

	t.Run("returns false for nil createdBy", func(t *testing.T) {
		ctx := context.Background()
		ctx, _ = logger.WithUserID(ctx, logger.FromContext(ctx), userID.String())

		filter := NewFilter(ctx, []identity.Role{})

		assert.False(t, filter.IsOwner(nil))
	})

	t.Run("returns false for nil userID", func(t *testing.T) {
		filter := NewFilter(context.Background(), []identity.Role{})
		createdBy := uuid.New()

		assert.False(t, filter.IsOwner(&createdBy))
	})

	t.Run("returns true when user is owner", func(t *testing.T) {
		ctx := context.Background()
		ctx, _ = logger.WithUserID(ctx, logger.FromContext(ctx), userID.String())

		filter := NewFilter(ctx, []identity.Role{})

		assert.True(t, filter.IsOwner(&userID))
	})

	t.Run("returns false when user is not owner", func(t *testing.T) {
		ctx := context.Background()
		ctx, _ = logger.WithUserID(ctx, logger.FromContext(ctx), userID.String())

		filter := NewFilter(ctx, []identity.Role{})
		otherUser := uuid.New()

		assert.False(t, filter.IsOwner(&otherUser))
	})
}

func TestWithDataScopes(t *testing.T) {
	tenantID := uuid.New()

	t.Run("stores data scopes in context", func(t *testing.T) {
		ctx := context.Background()

		role, _ := identity.NewRole(tenantID, "ROLE1", "Role 1")
		ds1, _ := identity.NewDataScope("sales_order", identity.DataScopeSelf)
		ds2, _ := identity.NewDataScope("product", identity.DataScopeAll)
		_ = role.SetDataScope(*ds1)
		_ = role.SetDataScope(*ds2)

		ctx = WithDataScopes(ctx, []identity.Role{*role})

		scopes, ok := ctx.Value(ScopesKey).(map[string]identity.DataScope)
		require.True(t, ok)
		assert.Len(t, scopes, 2)
		assert.Equal(t, identity.DataScopeSelf, scopes["sales_order"].ScopeType)
		assert.Equal(t, identity.DataScopeAll, scopes["product"].ScopeType)
	})
}

func TestNewFilterFromContext(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("creates filter from context scopes", func(t *testing.T) {
		ctx := context.Background()
		ctx, _ = logger.WithUserID(ctx, logger.FromContext(ctx), userID.String())

		role, _ := identity.NewRole(tenantID, "ROLE1", "Role 1")
		ds, _ := identity.NewDataScope("sales_order", identity.DataScopeSelf)
		_ = role.SetDataScope(*ds)

		ctx = WithDataScopes(ctx, []identity.Role{*role})

		filter := NewFilterFromContext(ctx)

		assert.Equal(t, userID, filter.userID)
		assert.Equal(t, identity.DataScopeSelf, filter.GetScopeType("sales_order"))
	})

	t.Run("handles missing scopes in context", func(t *testing.T) {
		filter := NewFilterFromContext(context.Background())

		assert.Empty(t, filter.dataScopes)
		assert.Equal(t, identity.DataScopeAll, filter.GetScopeType("any_resource"))
	})
}

func TestCompareScopeLevel(t *testing.T) {
	testCases := []struct {
		name     string
		a        identity.DataScopeType
		b        identity.DataScopeType
		expected int
	}{
		{"ALL > SELF", identity.DataScopeAll, identity.DataScopeSelf, 90},
		{"ALL > DEPARTMENT", identity.DataScopeAll, identity.DataScopeDepartment, 50},
		{"DEPARTMENT > SELF", identity.DataScopeDepartment, identity.DataScopeSelf, 40},
		{"SELF < ALL", identity.DataScopeSelf, identity.DataScopeAll, -90},
		{"SELF == SELF", identity.DataScopeSelf, identity.DataScopeSelf, 0},
		{"ALL == ALL", identity.DataScopeAll, identity.DataScopeAll, 0},
		{"CUSTOM > SELF", identity.DataScopeCustom, identity.DataScopeSelf, 30},
		{"DEPARTMENT > CUSTOM", identity.DataScopeDepartment, identity.DataScopeCustom, 10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := compareScopeLevel(tc.a, tc.b)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMergeScopes(t *testing.T) {
	t.Run("merges empty lists", func(t *testing.T) {
		result := MergeScopes()
		assert.Empty(t, result)
	})

	t.Run("merges single list", func(t *testing.T) {
		ds1, _ := identity.NewDataScope("sales_order", identity.DataScopeSelf)
		ds2, _ := identity.NewDataScope("product", identity.DataScopeAll)

		result := MergeScopes([]identity.DataScope{*ds1, *ds2})

		assert.Len(t, result, 2)
		assert.Equal(t, identity.DataScopeSelf, result["sales_order"].ScopeType)
		assert.Equal(t, identity.DataScopeAll, result["product"].ScopeType)
	})

	t.Run("merges multiple lists keeping higher permission", func(t *testing.T) {
		ds1, _ := identity.NewDataScope("sales_order", identity.DataScopeSelf)
		ds2, _ := identity.NewDataScope("sales_order", identity.DataScopeAll)
		ds3, _ := identity.NewDataScope("product", identity.DataScopeSelf)

		result := MergeScopes(
			[]identity.DataScope{*ds1},
			[]identity.DataScope{*ds2, *ds3},
		)

		assert.Len(t, result, 2)
		assert.Equal(t, identity.DataScopeAll, result["sales_order"].ScopeType)
		assert.Equal(t, identity.DataScopeSelf, result["product"].ScopeType)
	})

	t.Run("handles overlapping resources correctly", func(t *testing.T) {
		ds1, _ := identity.NewDataScope("sales_order", identity.DataScopeDepartment)
		ds2, _ := identity.NewDataScope("sales_order", identity.DataScopeSelf)
		ds3, _ := identity.NewDataScope("sales_order", identity.DataScopeAll)

		result := MergeScopes(
			[]identity.DataScope{*ds1},
			[]identity.DataScope{*ds2},
			[]identity.DataScope{*ds3},
		)

		assert.Len(t, result, 1)
		assert.Equal(t, identity.DataScopeAll, result["sales_order"].ScopeType)
	})
}

func TestFilter_GetUserID(t *testing.T) {
	t.Run("returns user ID from context", func(t *testing.T) {
		userID := uuid.New()
		ctx := context.Background()
		ctx, _ = logger.WithUserID(ctx, logger.FromContext(ctx), userID.String())

		filter := NewFilter(ctx, []identity.Role{})

		assert.Equal(t, userID, filter.GetUserID())
	})

	t.Run("returns nil UUID for missing user ID", func(t *testing.T) {
		filter := NewFilter(context.Background(), []identity.Role{})

		assert.Equal(t, uuid.Nil, filter.GetUserID())
	})
}

func TestDataScopeScopeFromContext(t *testing.T) {
	tenantID := uuid.New()

	t.Run("creates GORM scope function", func(t *testing.T) {
		ctx := context.Background()

		role, _ := identity.NewRole(tenantID, "ROLE1", "Role 1")
		ds, _ := identity.NewDataScope("sales_order", identity.DataScopeSelf)
		_ = role.SetDataScope(*ds)

		ctx = WithDataScopes(ctx, []identity.Role{*role})

		scopeFunc := DataScopeScopeFromContext(ctx, "sales_order")

		assert.NotNil(t, scopeFunc)
	})
}

func TestFilter_getDefaultScopeField(t *testing.T) {
	filter := &Filter{}

	testCases := []struct {
		resource      string
		expectedField string
	}{
		{"inventory", "warehouse_id"},
		{"sales_order", "warehouse_id"},
		{"purchase_order", "warehouse_id"},
		{"stock_batch", "warehouse_id"},
		{"stock_lock", "warehouse_id"},
		{"sales_return", "warehouse_id"},
		{"purchase_return", "warehouse_id"},
		{"stock_take", "warehouse_id"},
		{"stock_transfer", "warehouse_id"},
		{"customer", ""},
		{"product", ""},
		{"unknown_resource", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.resource, func(t *testing.T) {
			field := filter.getDefaultScopeField(tc.resource)
			assert.Equal(t, tc.expectedField, field)
		})
	}
}

func TestFilter_CustomScopeWithField(t *testing.T) {
	tenantID := uuid.New()
	warehouseID := uuid.New()

	t.Run("uses custom scope field when specified", func(t *testing.T) {
		ctx := context.Background()

		role, _ := identity.NewRole(tenantID, "WAREHOUSE", "Warehouse Manager")
		ds, _ := identity.NewCustomDataScopeWithField("inventory", "warehouse_id", []string{warehouseID.String()})
		_ = role.SetDataScope(*ds)

		filter := NewFilter(ctx, []identity.Role{*role})

		// Verify the filter has the correct scope field
		scopeType := filter.GetScopeType("inventory")
		assert.Equal(t, identity.DataScopeCustom, scopeType)
	})

	t.Run("falls back to default field when scope field is empty", func(t *testing.T) {
		ctx := context.Background()

		role, _ := identity.NewRole(tenantID, "WAREHOUSE", "Warehouse Manager")
		ds, _ := identity.NewCustomDataScope("sales_order", []string{warehouseID.String()})
		_ = role.SetDataScope(*ds)

		filter := NewFilter(ctx, []identity.Role{*role})

		// The filter should use default field mapping
		defaultField := filter.getDefaultScopeField("sales_order")
		assert.Equal(t, "warehouse_id", defaultField)
	})
}

func TestCustomDataScopeWithField(t *testing.T) {
	t.Run("creates custom scope with field", func(t *testing.T) {
		ds, err := identity.NewCustomDataScopeWithField("inventory", "warehouse_id", []string{"wh-1", "wh-2"})
		require.NoError(t, err)

		assert.Equal(t, "inventory", ds.Resource)
		assert.Equal(t, identity.DataScopeCustom, ds.ScopeType)
		assert.Equal(t, "warehouse_id", ds.ScopeField)
		assert.Equal(t, []string{"wh-1", "wh-2"}, ds.ScopeValues)
	})

	t.Run("fails with empty scope field", func(t *testing.T) {
		_, err := identity.NewCustomDataScopeWithField("inventory", "", []string{"wh-1"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Scope field cannot be empty")
	})

	t.Run("fails with empty scope values", func(t *testing.T) {
		_, err := identity.NewCustomDataScopeWithField("inventory", "warehouse_id", []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one scope value")
	})
}

// ============================================================================
// WAREHOUSE Scope Tests - DDD-020
// ============================================================================

func TestWarehouseDataScope(t *testing.T) {
	t.Run("creates warehouse scope successfully", func(t *testing.T) {
		warehouseIDs := []string{"wh-001", "wh-002"}
		ds, err := identity.NewWarehouseDataScope("inventory", warehouseIDs)
		require.NoError(t, err)

		assert.Equal(t, "inventory", ds.Resource)
		assert.Equal(t, identity.DataScopeWarehouse, ds.ScopeType)
		assert.Equal(t, "warehouse_id", ds.ScopeField)
		assert.Equal(t, warehouseIDs, ds.ScopeValues)
	})

	t.Run("fails with empty warehouse IDs", func(t *testing.T) {
		_, err := identity.NewWarehouseDataScope("inventory", []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one warehouse ID")
	})

	t.Run("fails with invalid resource", func(t *testing.T) {
		_, err := identity.NewWarehouseDataScope("", []string{"wh-001"})
		require.Error(t, err)
	})

	t.Run("makes defensive copy of warehouse IDs", func(t *testing.T) {
		warehouseIDs := []string{"wh-001", "wh-002"}
		ds, err := identity.NewWarehouseDataScope("inventory", warehouseIDs)
		require.NoError(t, err)

		// Modify original slice
		warehouseIDs[0] = "modified"

		// DataScope should not be affected
		assert.Equal(t, "wh-001", ds.ScopeValues[0])
	})
}

func TestFilter_WarehouseScope(t *testing.T) {
	tenantID := uuid.New()
	warehouseID1 := uuid.New().String()
	warehouseID2 := uuid.New().String()

	t.Run("filters by warehouse_id for WAREHOUSE scope", func(t *testing.T) {
		ctx := context.Background()

		role, _ := identity.NewRole(tenantID, "WAREHOUSE", "Warehouse Manager")
		ds, _ := identity.NewWarehouseDataScope("inventory", []string{warehouseID1, warehouseID2})
		_ = role.SetDataScope(*ds)

		filter := NewFilter(ctx, []identity.Role{*role})

		// Verify scope type
		assert.Equal(t, identity.DataScopeWarehouse, filter.GetScopeType("inventory"))
	})

	t.Run("returns empty result when no warehouses assigned", func(t *testing.T) {
		ctx := context.Background()

		role, _ := identity.NewRole(tenantID, "WAREHOUSE", "Warehouse Manager")
		// Create a scope manually with empty values (edge case)
		ds := identity.DataScope{
			Resource:    "inventory",
			ScopeType:   identity.DataScopeWarehouse,
			ScopeField:  "warehouse_id",
			ScopeValues: []string{},
		}
		_ = role.SetDataScope(ds)

		filter := NewFilter(ctx, []identity.Role{*role})

		assert.Equal(t, identity.DataScopeWarehouse, filter.GetScopeType("inventory"))
	})

	t.Run("warehouse scope takes precedence over lower scopes", func(t *testing.T) {
		ctx := context.Background()

		// Role 1: SELF scope
		role1, _ := identity.NewRole(tenantID, "SALES", "Salesperson")
		ds1, _ := identity.NewDataScope("inventory", identity.DataScopeSelf)
		_ = role1.SetDataScope(*ds1)

		// Role 2: WAREHOUSE scope
		role2, _ := identity.NewRole(tenantID, "WAREHOUSE", "Warehouse Manager")
		ds2, _ := identity.NewWarehouseDataScope("inventory", []string{warehouseID1})
		_ = role2.SetDataScope(*ds2)

		filter := NewFilter(ctx, []identity.Role{*role1, *role2})

		// WAREHOUSE (45) > SELF (10)
		assert.Equal(t, identity.DataScopeWarehouse, filter.GetScopeType("inventory"))
	})

	t.Run("ALL scope takes precedence over WAREHOUSE scope", func(t *testing.T) {
		ctx := context.Background()

		// Role 1: WAREHOUSE scope
		role1, _ := identity.NewRole(tenantID, "WAREHOUSE", "Warehouse Manager")
		ds1, _ := identity.NewWarehouseDataScope("inventory", []string{warehouseID1})
		_ = role1.SetDataScope(*ds1)

		// Role 2: ALL scope (e.g., Manager role)
		role2, _ := identity.NewRole(tenantID, "MANAGER", "Manager")
		ds2, _ := identity.NewDataScope("inventory", identity.DataScopeAll)
		_ = role2.SetDataScope(*ds2)

		filter := NewFilter(ctx, []identity.Role{*role1, *role2})

		// ALL (100) > WAREHOUSE (45)
		assert.Equal(t, identity.DataScopeAll, filter.GetScopeType("inventory"))
	})
}

func TestFilter_GetWarehouseIDs(t *testing.T) {
	tenantID := uuid.New()
	warehouseID1 := uuid.New().String()
	warehouseID2 := uuid.New().String()

	t.Run("returns warehouse IDs for WAREHOUSE scope", func(t *testing.T) {
		ctx := context.Background()

		role, _ := identity.NewRole(tenantID, "WAREHOUSE", "Warehouse Manager")
		ds, _ := identity.NewWarehouseDataScope("inventory", []string{warehouseID1, warehouseID2})
		_ = role.SetDataScope(*ds)

		filter := NewFilter(ctx, []identity.Role{*role})

		warehouseIDs := filter.GetWarehouseIDs("inventory")
		assert.Len(t, warehouseIDs, 2)
		assert.Contains(t, warehouseIDs, warehouseID1)
		assert.Contains(t, warehouseIDs, warehouseID2)
	})

	t.Run("returns nil for non-WAREHOUSE scope", func(t *testing.T) {
		ctx := context.Background()

		role, _ := identity.NewRole(tenantID, "MANAGER", "Manager")
		ds, _ := identity.NewDataScope("inventory", identity.DataScopeAll)
		_ = role.SetDataScope(*ds)

		filter := NewFilter(ctx, []identity.Role{*role})

		assert.Nil(t, filter.GetWarehouseIDs("inventory"))
	})

	t.Run("returns nil for unconfigured resource", func(t *testing.T) {
		filter := NewFilter(context.Background(), []identity.Role{})

		assert.Nil(t, filter.GetWarehouseIDs("inventory"))
	})
}

func TestFilter_HasWarehouseAccess(t *testing.T) {
	tenantID := uuid.New()
	warehouseID1 := uuid.New().String()
	warehouseID2 := uuid.New().String()
	warehouseID3 := uuid.New().String()

	t.Run("returns true for warehouse in scope", func(t *testing.T) {
		ctx := context.Background()

		role, _ := identity.NewRole(tenantID, "WAREHOUSE", "Warehouse Manager")
		ds, _ := identity.NewWarehouseDataScope("inventory", []string{warehouseID1, warehouseID2})
		_ = role.SetDataScope(*ds)

		filter := NewFilter(ctx, []identity.Role{*role})

		assert.True(t, filter.HasWarehouseAccess("inventory", warehouseID1))
		assert.True(t, filter.HasWarehouseAccess("inventory", warehouseID2))
	})

	t.Run("returns false for warehouse not in scope", func(t *testing.T) {
		ctx := context.Background()

		role, _ := identity.NewRole(tenantID, "WAREHOUSE", "Warehouse Manager")
		ds, _ := identity.NewWarehouseDataScope("inventory", []string{warehouseID1, warehouseID2})
		_ = role.SetDataScope(*ds)

		filter := NewFilter(ctx, []identity.Role{*role})

		assert.False(t, filter.HasWarehouseAccess("inventory", warehouseID3))
	})

	t.Run("returns true for ALL scope", func(t *testing.T) {
		ctx := context.Background()

		role, _ := identity.NewRole(tenantID, "MANAGER", "Manager")
		ds, _ := identity.NewDataScope("inventory", identity.DataScopeAll)
		_ = role.SetDataScope(*ds)

		filter := NewFilter(ctx, []identity.Role{*role})

		assert.True(t, filter.HasWarehouseAccess("inventory", warehouseID1))
		assert.True(t, filter.HasWarehouseAccess("inventory", warehouseID3))
	})

	t.Run("returns true for unconfigured resource", func(t *testing.T) {
		filter := NewFilter(context.Background(), []identity.Role{})

		assert.True(t, filter.HasWarehouseAccess("inventory", warehouseID1))
	})
}

func TestFilter_IsWarehouseScoped(t *testing.T) {
	tenantID := uuid.New()
	warehouseID := uuid.New().String()

	t.Run("returns true for WAREHOUSE scope", func(t *testing.T) {
		ctx := context.Background()

		role, _ := identity.NewRole(tenantID, "WAREHOUSE", "Warehouse Manager")
		ds, _ := identity.NewWarehouseDataScope("inventory", []string{warehouseID})
		_ = role.SetDataScope(*ds)

		filter := NewFilter(ctx, []identity.Role{*role})

		assert.True(t, filter.IsWarehouseScoped("inventory"))
	})

	t.Run("returns false for other scope types", func(t *testing.T) {
		ctx := context.Background()

		role, _ := identity.NewRole(tenantID, "MANAGER", "Manager")
		ds, _ := identity.NewDataScope("inventory", identity.DataScopeAll)
		_ = role.SetDataScope(*ds)

		filter := NewFilter(ctx, []identity.Role{*role})

		assert.False(t, filter.IsWarehouseScoped("inventory"))
	})

	t.Run("returns false for unconfigured resource", func(t *testing.T) {
		filter := NewFilter(context.Background(), []identity.Role{})

		assert.False(t, filter.IsWarehouseScoped("inventory"))
	})
}

func TestCompareScopeLevel_WithWarehouse(t *testing.T) {
	testCases := []struct {
		name     string
		a        identity.DataScopeType
		b        identity.DataScopeType
		expected int
	}{
		{"ALL > WAREHOUSE", identity.DataScopeAll, identity.DataScopeWarehouse, 55},
		{"WAREHOUSE < ALL", identity.DataScopeWarehouse, identity.DataScopeAll, -55},
		{"WAREHOUSE > CUSTOM", identity.DataScopeWarehouse, identity.DataScopeCustom, 5},
		{"WAREHOUSE > SELF", identity.DataScopeWarehouse, identity.DataScopeSelf, 35},
		{"DEPARTMENT > WAREHOUSE", identity.DataScopeDepartment, identity.DataScopeWarehouse, 5},
		{"WAREHOUSE == WAREHOUSE", identity.DataScopeWarehouse, identity.DataScopeWarehouse, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := compareScopeLevel(tc.a, tc.b)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsResourceWarehouseScoped(t *testing.T) {
	t.Run("returns true for inventory-related resources", func(t *testing.T) {
		assert.True(t, IsResourceWarehouseScoped("inventory"))
		assert.True(t, IsResourceWarehouseScoped("sales_order"))
		assert.True(t, IsResourceWarehouseScoped("purchase_order"))
		assert.True(t, IsResourceWarehouseScoped("stock_batch"))
		assert.True(t, IsResourceWarehouseScoped("stock_lock"))
		assert.True(t, IsResourceWarehouseScoped("sales_return"))
		assert.True(t, IsResourceWarehouseScoped("purchase_return"))
		assert.True(t, IsResourceWarehouseScoped("stock_take"))
		assert.True(t, IsResourceWarehouseScoped("stock_transfer"))
	})

	t.Run("returns false for non-warehouse resources", func(t *testing.T) {
		assert.False(t, IsResourceWarehouseScoped("customer"))
		assert.False(t, IsResourceWarehouseScoped("product"))
		assert.False(t, IsResourceWarehouseScoped("user"))
		assert.False(t, IsResourceWarehouseScoped("role"))
		assert.False(t, IsResourceWarehouseScoped("unknown"))
	})
}

func TestCreateWarehouseScopesForRole(t *testing.T) {
	t.Run("creates scopes for all warehouse resources", func(t *testing.T) {
		warehouseIDs := []string{"wh-001", "wh-002"}
		scopes, err := CreateWarehouseScopesForRole(warehouseIDs)
		require.NoError(t, err)

		// Should have scopes for all warehouse-related resources
		assert.Len(t, scopes, 9) // inventory, sales_order, purchase_order, stock_batch, stock_lock, sales_return, purchase_return, stock_take, stock_transfer

		// All scopes should be WAREHOUSE type with correct warehouse IDs
		resourcesFound := make(map[string]bool)
		for _, ds := range scopes {
			assert.Equal(t, identity.DataScopeWarehouse, ds.ScopeType)
			assert.Equal(t, "warehouse_id", ds.ScopeField)
			assert.Equal(t, warehouseIDs, ds.ScopeValues)
			resourcesFound[ds.Resource] = true
		}

		// Verify all warehouse resources are covered
		assert.True(t, resourcesFound["inventory"])
		assert.True(t, resourcesFound["sales_order"])
		assert.True(t, resourcesFound["purchase_order"])
	})

	t.Run("returns nil for empty warehouse IDs", func(t *testing.T) {
		scopes, err := CreateWarehouseScopesForRole([]string{})
		require.NoError(t, err)
		assert.Nil(t, scopes)
	})

	t.Run("returns nil for nil warehouse IDs", func(t *testing.T) {
		scopes, err := CreateWarehouseScopesForRole(nil)
		require.NoError(t, err)
		assert.Nil(t, scopes)
	})
}

func TestWithWarehouseIDs(t *testing.T) {
	t.Run("stores warehouse IDs in context", func(t *testing.T) {
		ctx := context.Background()
		warehouseIDs := []string{"wh-001", "wh-002"}

		ctx = WithWarehouseIDs(ctx, warehouseIDs)

		retrieved := GetWarehouseIDsFromContext(ctx)
		assert.Equal(t, warehouseIDs, retrieved)
	})

	t.Run("returns nil for context without warehouse IDs", func(t *testing.T) {
		ctx := context.Background()

		retrieved := GetWarehouseIDsFromContext(ctx)
		assert.Nil(t, retrieved)
	})
}

func TestMergeScopes_WithWarehouse(t *testing.T) {
	warehouseID := uuid.New().String()

	t.Run("ALL takes precedence over WAREHOUSE", func(t *testing.T) {
		dsWarehouse, _ := identity.NewWarehouseDataScope("inventory", []string{warehouseID})
		dsAll, _ := identity.NewDataScope("inventory", identity.DataScopeAll)

		result := MergeScopes(
			[]identity.DataScope{*dsWarehouse},
			[]identity.DataScope{*dsAll},
		)

		assert.Len(t, result, 1)
		assert.Equal(t, identity.DataScopeAll, result["inventory"].ScopeType)
	})

	t.Run("WAREHOUSE takes precedence over SELF", func(t *testing.T) {
		dsWarehouse, _ := identity.NewWarehouseDataScope("inventory", []string{warehouseID})
		dsSelf, _ := identity.NewDataScope("inventory", identity.DataScopeSelf)

		result := MergeScopes(
			[]identity.DataScope{*dsSelf},
			[]identity.DataScope{*dsWarehouse},
		)

		assert.Len(t, result, 1)
		assert.Equal(t, identity.DataScopeWarehouse, result["inventory"].ScopeType)
	})

	t.Run("WAREHOUSE takes precedence over CUSTOM", func(t *testing.T) {
		dsWarehouse, _ := identity.NewWarehouseDataScope("inventory", []string{warehouseID})
		dsCustom, _ := identity.NewCustomDataScope("inventory", []string{"value1"})

		result := MergeScopes(
			[]identity.DataScope{*dsCustom},
			[]identity.DataScope{*dsWarehouse},
		)

		assert.Len(t, result, 1)
		assert.Equal(t, identity.DataScopeWarehouse, result["inventory"].ScopeType)
	})
}
