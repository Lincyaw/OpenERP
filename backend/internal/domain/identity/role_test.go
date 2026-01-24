package identity

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helpers

func createTestRole(t *testing.T) *Role {
	tenantID := uuid.New()
	role, err := NewRole(tenantID, "TEST_ROLE", "Test Role")
	require.NoError(t, err)
	require.NotNil(t, role)
	return role
}

func createTestSystemRole(t *testing.T) *Role {
	tenantID := uuid.New()
	role, err := NewSystemRole(tenantID, "ADMIN", "Administrator")
	require.NoError(t, err)
	require.NotNil(t, role)
	return role
}

// Permission Value Object Tests

func TestNewPermission(t *testing.T) {
	tests := []struct {
		name        string
		resource    string
		action      string
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid permission",
			resource: "product",
			action:   "create",
			wantErr:  false,
		},
		{
			name:     "valid permission with underscore",
			resource: "sales_order",
			action:   "confirm",
			wantErr:  false,
		},
		{
			name:        "empty resource",
			resource:    "",
			action:      "create",
			wantErr:     true,
			errContains: "resource cannot be empty",
		},
		{
			name:        "empty action",
			resource:    "product",
			action:      "",
			wantErr:     true,
			errContains: "action cannot be empty",
		},
		{
			name:        "resource starting with number",
			resource:    "1product",
			action:      "create",
			wantErr:     true,
			errContains: "must start with a letter",
		},
		{
			name:        "action with invalid characters",
			resource:    "product",
			action:      "create-item",
			wantErr:     true,
			errContains: "must start with a letter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perm, err := NewPermission(tt.resource, tt.action)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, perm)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, perm)
				assert.Equal(t, tt.resource+":"+tt.action, perm.Code)
				assert.Equal(t, tt.resource, perm.Resource)
				assert.Equal(t, tt.action, perm.Action)
			}
		})
	}
}

func TestNewPermissionFromCode(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid code",
			code:    "product:create",
			wantErr: false,
		},
		{
			name:    "valid code with underscore",
			code:    "sales_order:confirm",
			wantErr: false,
		},
		{
			name:        "invalid code format - no colon",
			code:        "productcreate",
			wantErr:     true,
			errContains: "format 'resource:action'",
		},
		{
			name:        "invalid code format - empty",
			code:        "",
			wantErr:     true,
			errContains: "format 'resource:action'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perm, err := NewPermissionFromCode(tt.code)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.code, perm.Code)
			}
		})
	}
}

func TestPermission_Equals(t *testing.T) {
	perm1, _ := NewPermission("product", "create")
	perm2, _ := NewPermission("product", "create")
	perm3, _ := NewPermission("product", "read")

	assert.True(t, perm1.Equals(*perm2))
	assert.False(t, perm1.Equals(*perm3))
}

func TestPermission_IsEmpty(t *testing.T) {
	perm1, _ := NewPermission("product", "create")
	perm2 := Permission{}

	assert.False(t, perm1.IsEmpty())
	assert.True(t, perm2.IsEmpty())
}

// DataScope Value Object Tests

func TestNewDataScope(t *testing.T) {
	tests := []struct {
		name        string
		resource    string
		scopeType   DataScopeType
		wantErr     bool
		errContains string
	}{
		{
			name:      "valid data scope - all",
			resource:  "sales_order",
			scopeType: DataScopeAll,
			wantErr:   false,
		},
		{
			name:      "valid data scope - self",
			resource:  "sales_order",
			scopeType: DataScopeSelf,
			wantErr:   false,
		},
		{
			name:      "valid data scope - department",
			resource:  "sales_order",
			scopeType: DataScopeDepartment,
			wantErr:   false,
		},
		{
			name:        "empty resource",
			resource:    "",
			scopeType:   DataScopeAll,
			wantErr:     true,
			errContains: "resource cannot be empty",
		},
		{
			name:        "invalid scope type",
			resource:    "sales_order",
			scopeType:   DataScopeType("invalid"),
			wantErr:     true,
			errContains: "Invalid data scope type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds, err := NewDataScope(tt.resource, tt.scopeType)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.resource, ds.Resource)
				assert.Equal(t, tt.scopeType, ds.ScopeType)
			}
		})
	}
}

func TestNewCustomDataScope(t *testing.T) {
	// Valid custom data scope
	ds, err := NewCustomDataScope("sales_order", []string{"region_1", "region_2"})
	require.NoError(t, err)
	assert.Equal(t, DataScopeCustom, ds.ScopeType)
	assert.Len(t, ds.ScopeValues, 2)

	// Empty scope values
	_, err = NewCustomDataScope("sales_order", []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must have at least one scope value")
}

func TestDataScope_Equals(t *testing.T) {
	ds1, _ := NewDataScope("sales_order", DataScopeAll)
	ds2, _ := NewDataScope("sales_order", DataScopeAll)
	ds3, _ := NewDataScope("sales_order", DataScopeSelf)
	ds4, _ := NewDataScope("purchase_order", DataScopeAll)

	assert.True(t, ds1.Equals(*ds2))
	assert.False(t, ds1.Equals(*ds3)) // Different scope type
	assert.False(t, ds1.Equals(*ds4)) // Different resource
}

// Role Aggregate Tests

func TestNewRole(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		roleName    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid role",
			code:     "SALES",
			roleName: "Sales Representative",
			wantErr:  false,
		},
		{
			name:     "valid role with underscore",
			code:     "SALES_MANAGER",
			roleName: "Sales Manager",
			wantErr:  false,
		},
		{
			name:        "empty code",
			code:        "",
			roleName:    "Test Role",
			wantErr:     true,
			errContains: "Role code cannot be empty",
		},
		{
			name:        "code too short",
			code:        "A",
			roleName:    "Test Role",
			wantErr:     true,
			errContains: "at least 2 characters",
		},
		{
			name:        "code starting with number",
			code:        "1ROLE",
			roleName:    "Test Role",
			wantErr:     true,
			errContains: "must start with a letter",
		},
		{
			name:        "code with invalid characters",
			code:        "ROLE-TEST",
			roleName:    "Test Role",
			wantErr:     true,
			errContains: "must start with a letter",
		},
		{
			name:        "empty name",
			code:        "TEST",
			roleName:    "",
			wantErr:     true,
			errContains: "Role name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenantID := uuid.New()
			role, err := NewRole(tenantID, tt.code, tt.roleName)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, role)
				assert.Equal(t, tenantID, role.TenantID)
				assert.NotEqual(t, uuid.Nil, role.ID)
				assert.False(t, role.IsSystemRole)
				assert.True(t, role.IsEnabled)
				assert.Empty(t, role.Permissions)
				assert.Empty(t, role.DataScopes)

				// Check events
				events := role.GetDomainEvents()
				require.Len(t, events, 1)
				_, ok := events[0].(*RoleCreatedEvent)
				assert.True(t, ok)
			}
		})
	}
}

func TestNewSystemRole(t *testing.T) {
	tenantID := uuid.New()
	role, err := NewSystemRole(tenantID, "ADMIN", "Administrator")
	require.NoError(t, err)
	assert.True(t, role.IsSystemRole)
	assert.True(t, role.IsEnabled)
	assert.False(t, role.CanDelete())
}

func TestRole_SetName(t *testing.T) {
	role := createTestRole(t)
	oldVersion := role.Version

	err := role.SetName("Updated Name")
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", role.Name)
	assert.Equal(t, oldVersion+1, role.Version)

	// Empty name
	err = role.SetName("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestRole_SetDescription(t *testing.T) {
	role := createTestRole(t)
	oldVersion := role.Version

	role.SetDescription("This is a test role")
	assert.Equal(t, "This is a test role", role.Description)
	assert.Equal(t, oldVersion+1, role.Version)
}

func TestRole_EnableDisable(t *testing.T) {
	role := createTestRole(t)
	role.ClearDomainEvents()

	// Disable
	err := role.Disable()
	require.NoError(t, err)
	assert.False(t, role.IsEnabled)
	events := role.GetDomainEvents()
	require.Len(t, events, 1)
	_, ok := events[0].(*RoleDisabledEvent)
	assert.True(t, ok)

	// Try to disable again
	err = role.Disable()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already disabled")

	role.ClearDomainEvents()

	// Enable
	err = role.Enable()
	require.NoError(t, err)
	assert.True(t, role.IsEnabled)
	events = role.GetDomainEvents()
	require.Len(t, events, 1)
	_, ok = events[0].(*RoleEnabledEvent)
	assert.True(t, ok)

	// Try to enable again
	err = role.Enable()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already enabled")
}

func TestRole_GrantPermission(t *testing.T) {
	role := createTestRole(t)
	role.ClearDomainEvents()

	// Grant a permission
	perm, _ := NewPermission("product", "create")
	err := role.GrantPermission(*perm)
	require.NoError(t, err)
	assert.Len(t, role.Permissions, 1)
	assert.True(t, role.HasPermission("product:create"))

	// Check event
	events := role.GetDomainEvents()
	require.Len(t, events, 1)
	grantedEvent, ok := events[0].(*RolePermissionGrantedEvent)
	assert.True(t, ok)
	assert.Equal(t, "product:create", grantedEvent.PermissionCode)

	// Try to grant the same permission
	err = role.GrantPermission(*perm)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already has this permission")

	// Grant using code
	err = role.GrantPermissionByCode("product:read")
	require.NoError(t, err)
	assert.True(t, role.HasPermission("product:read"))
}

func TestRole_RevokePermission(t *testing.T) {
	role := createTestRole(t)

	// Grant some permissions
	role.GrantPermissionByCode("product:create")
	role.GrantPermissionByCode("product:read")
	role.ClearDomainEvents()

	// Revoke a permission
	err := role.RevokePermission("product:create")
	require.NoError(t, err)
	assert.Len(t, role.Permissions, 1)
	assert.False(t, role.HasPermission("product:create"))
	assert.True(t, role.HasPermission("product:read"))

	// Check event
	events := role.GetDomainEvents()
	require.Len(t, events, 1)
	revokedEvent, ok := events[0].(*RolePermissionRevokedEvent)
	assert.True(t, ok)
	assert.Equal(t, "product:create", revokedEvent.PermissionCode)

	// Try to revoke a permission that doesn't exist
	err = role.RevokePermission("product:delete")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not have this permission")
}

func TestRole_SetPermissions(t *testing.T) {
	role := createTestRole(t)

	// Set multiple permissions
	perm1, _ := NewPermission("product", "create")
	perm2, _ := NewPermission("product", "read")
	perm3, _ := NewPermission("product", "create") // Duplicate

	err := role.SetPermissions([]Permission{*perm1, *perm2, *perm3})
	require.NoError(t, err)

	// Should deduplicate
	assert.Len(t, role.Permissions, 2)
	assert.True(t, role.HasPermission("product:create"))
	assert.True(t, role.HasPermission("product:read"))
}

func TestRole_HasPermissionForResource(t *testing.T) {
	role := createTestRole(t)

	role.GrantPermissionByCode("product:create")
	role.GrantPermissionByCode("product:read")
	role.GrantPermissionByCode("customer:read")

	assert.True(t, role.HasPermissionForResource("product"))
	assert.True(t, role.HasPermissionForResource("customer"))
	assert.False(t, role.HasPermissionForResource("sales_order"))
}

func TestRole_GetPermissionsForResource(t *testing.T) {
	role := createTestRole(t)

	role.GrantPermissionByCode("product:create")
	role.GrantPermissionByCode("product:read")
	role.GrantPermissionByCode("customer:read")

	productPerms := role.GetPermissionsForResource("product")
	assert.Len(t, productPerms, 2)

	customerPerms := role.GetPermissionsForResource("customer")
	assert.Len(t, customerPerms, 1)

	orderPerms := role.GetPermissionsForResource("sales_order")
	assert.Len(t, orderPerms, 0)
}

func TestRole_SetDataScope(t *testing.T) {
	role := createTestRole(t)
	role.ClearDomainEvents()

	// Set a data scope
	ds, _ := NewDataScope("sales_order", DataScopeSelf)
	err := role.SetDataScope(*ds)
	require.NoError(t, err)
	assert.Len(t, role.DataScopes, 1)
	assert.True(t, role.HasDataScope("sales_order"))

	// Check event
	events := role.GetDomainEvents()
	require.Len(t, events, 1)
	changedEvent, ok := events[0].(*RoleDataScopeChangedEvent)
	assert.True(t, ok)
	assert.Equal(t, "sales_order", changedEvent.Resource)
	assert.Equal(t, DataScopeSelf, changedEvent.ScopeType)

	role.ClearDomainEvents()

	// Update the same resource's data scope
	ds2, _ := NewDataScope("sales_order", DataScopeAll)
	err = role.SetDataScope(*ds2)
	require.NoError(t, err)
	assert.Len(t, role.DataScopes, 1) // Still only one

	retrievedDS := role.GetDataScope("sales_order")
	require.NotNil(t, retrievedDS)
	assert.Equal(t, DataScopeAll, retrievedDS.ScopeType)
}

func TestRole_RemoveDataScope(t *testing.T) {
	role := createTestRole(t)

	// Set some data scopes
	ds1, _ := NewDataScope("sales_order", DataScopeSelf)
	ds2, _ := NewDataScope("purchase_order", DataScopeAll)
	role.SetDataScope(*ds1)
	role.SetDataScope(*ds2)

	// Remove one
	err := role.RemoveDataScope("sales_order")
	require.NoError(t, err)
	assert.Len(t, role.DataScopes, 1)
	assert.False(t, role.HasDataScope("sales_order"))
	assert.True(t, role.HasDataScope("purchase_order"))

	// Try to remove one that doesn't exist
	err = role.RemoveDataScope("inventory")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not have data scope")
}

func TestRole_SetDataScopes(t *testing.T) {
	role := createTestRole(t)

	ds1, _ := NewDataScope("sales_order", DataScopeSelf)
	ds2, _ := NewDataScope("purchase_order", DataScopeAll)
	ds3, _ := NewDataScope("sales_order", DataScopeDepartment) // Duplicate resource

	err := role.SetDataScopes([]DataScope{*ds1, *ds2, *ds3})
	require.NoError(t, err)

	// Should deduplicate by resource (keep first)
	assert.Len(t, role.DataScopes, 2)
	salesDS := role.GetDataScope("sales_order")
	require.NotNil(t, salesDS)
	assert.Equal(t, DataScopeSelf, salesDS.ScopeType) // First one wins
}

func TestRole_Update(t *testing.T) {
	role := createTestRole(t)
	role.ClearDomainEvents()

	err := role.Update("New Name", "New Description")
	require.NoError(t, err)
	assert.Equal(t, "New Name", role.Name)
	assert.Equal(t, "New Description", role.Description)

	// Check event
	events := role.GetDomainEvents()
	require.Len(t, events, 1)
	_, ok := events[0].(*RoleUpdatedEvent)
	assert.True(t, ok)
}

func TestRole_CanDelete(t *testing.T) {
	// Regular role can be deleted
	regularRole := createTestRole(t)
	assert.True(t, regularRole.CanDelete())

	// System role cannot be deleted
	systemRole := createTestSystemRole(t)
	assert.False(t, systemRole.CanDelete())
}

func TestRoleCodeConstants(t *testing.T) {
	// Verify predefined role codes are valid
	codes := []string{
		RoleCodeAdmin,
		RoleCodeOwner,
		RoleCodeManager,
		RoleCodeSales,
		RoleCodePurchaser,
		RoleCodeWarehouse,
		RoleCodeCashier,
		RoleCodeAccountant,
	}

	tenantID := uuid.New()
	for _, code := range codes {
		role, err := NewRole(tenantID, code, "Test Role")
		require.NoError(t, err, "Failed to create role with code: %s", code)
		assert.NotNil(t, role)
	}
}

func TestResourceAndActionConstants(t *testing.T) {
	// Verify predefined resource/action combinations are valid
	resources := []string{
		ResourceProduct,
		ResourceCategory,
		ResourceCustomer,
		ResourceSupplier,
		ResourceWarehouse,
		ResourceInventory,
		ResourceSalesOrder,
		ResourcePurchaseOrder,
	}

	actions := []string{
		ActionCreate,
		ActionRead,
		ActionUpdate,
		ActionDelete,
		ActionEnable,
		ActionDisable,
		ActionConfirm,
		ActionCancel,
	}

	for _, resource := range resources {
		for _, action := range actions {
			perm, err := NewPermission(resource, action)
			require.NoError(t, err, "Failed to create permission for %s:%s", resource, action)
			assert.NotNil(t, perm)
		}
	}
}

func TestRole_ConcurrentPermissionOperations(t *testing.T) {
	role := createTestRole(t)

	// Grant multiple permissions
	permissions := []string{
		"product:create",
		"product:read",
		"product:update",
		"customer:read",
		"customer:create",
		"sales_order:create",
		"sales_order:confirm",
		"sales_order:ship",
	}

	for _, code := range permissions {
		err := role.GrantPermissionByCode(code)
		require.NoError(t, err)
	}

	assert.Len(t, role.Permissions, len(permissions))

	// Verify all permissions exist
	for _, code := range permissions {
		assert.True(t, role.HasPermission(code), "Missing permission: %s", code)
	}

	// Revoke some permissions
	err := role.RevokePermission("product:update")
	require.NoError(t, err)
	err = role.RevokePermission("sales_order:ship")
	require.NoError(t, err)

	assert.Len(t, role.Permissions, len(permissions)-2)
	assert.False(t, role.HasPermission("product:update"))
	assert.False(t, role.HasPermission("sales_order:ship"))
	assert.True(t, role.HasPermission("product:create"))
}

func TestRole_VersionIncrement(t *testing.T) {
	role := createTestRole(t)
	initialVersion := role.Version

	// Each operation should increment version
	role.SetDescription("desc")
	assert.Equal(t, initialVersion+1, role.Version)

	role.SetSortOrder(10)
	assert.Equal(t, initialVersion+2, role.Version)

	role.GrantPermissionByCode("product:read")
	assert.Equal(t, initialVersion+3, role.Version)

	role.RevokePermission("product:read")
	assert.Equal(t, initialVersion+4, role.Version)
}

func TestRole_CodeNormalization(t *testing.T) {
	tenantID := uuid.New()

	// Code should be normalized to uppercase
	role, err := NewRole(tenantID, "sales_rep", "Sales Rep")
	require.NoError(t, err)
	assert.Equal(t, "SALES_REP", role.Code)

	// Code with mixed case
	role2, err := NewRole(tenantID, "SalesManager", "Sales Manager")
	require.NoError(t, err)
	assert.Equal(t, "SALESMANAGER", role2.Code)
}

func TestRole_EmptyPermission(t *testing.T) {
	role := createTestRole(t)

	// Cannot grant empty permission
	err := role.GrantPermission(Permission{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Permission cannot be empty")
}

func TestRole_EmptyDataScope(t *testing.T) {
	role := createTestRole(t)

	// Cannot set empty data scope
	err := role.SetDataScope(DataScope{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Data scope cannot be empty")
}
