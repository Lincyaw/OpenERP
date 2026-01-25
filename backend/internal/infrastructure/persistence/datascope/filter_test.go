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
