package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/infrastructure/persistence/datascope"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRoleRepository implements identity.RoleRepository for testing
type mockRoleRepository struct {
	roles       map[uuid.UUID]*identity.Role
	findByIDsFn func(ctx context.Context, ids []uuid.UUID) ([]*identity.Role, error)
}

func newMockRoleRepository() *mockRoleRepository {
	return &mockRoleRepository{
		roles: make(map[uuid.UUID]*identity.Role),
	}
}

func (m *mockRoleRepository) Create(ctx context.Context, role *identity.Role) error {
	m.roles[role.ID] = role
	return nil
}

func (m *mockRoleRepository) Update(ctx context.Context, role *identity.Role) error {
	m.roles[role.ID] = role
	return nil
}

func (m *mockRoleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.roles, id)
	return nil
}

func (m *mockRoleRepository) FindByID(ctx context.Context, id uuid.UUID) (*identity.Role, error) {
	if role, ok := m.roles[id]; ok {
		return role, nil
	}
	return nil, nil
}

func (m *mockRoleRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*identity.Role, error) {
	if m.findByIDsFn != nil {
		return m.findByIDsFn(ctx, ids)
	}
	var result []*identity.Role
	for _, id := range ids {
		if role, ok := m.roles[id]; ok {
			result = append(result, role)
		}
	}
	return result, nil
}

func (m *mockRoleRepository) FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*identity.Role, error) {
	for _, role := range m.roles {
		if role.TenantID == tenantID && role.Code == code {
			return role, nil
		}
	}
	return nil, nil
}

func (m *mockRoleRepository) FindAll(ctx context.Context, tenantID uuid.UUID, filter *identity.RoleFilter) ([]*identity.Role, error) {
	var result []*identity.Role
	for _, role := range m.roles {
		if role.TenantID == tenantID {
			result = append(result, role)
		}
	}
	return result, nil
}

func (m *mockRoleRepository) Count(ctx context.Context, tenantID uuid.UUID, filter *identity.RoleFilter) (int64, error) {
	count := int64(0)
	for _, role := range m.roles {
		if role.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

func (m *mockRoleRepository) ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error) {
	for _, role := range m.roles {
		if role.TenantID == tenantID && role.Code == code {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockRoleRepository) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	_, exists := m.roles[id]
	return exists, nil
}

func (m *mockRoleRepository) FindSystemRoles(ctx context.Context, tenantID uuid.UUID) ([]*identity.Role, error) {
	var result []*identity.Role
	for _, role := range m.roles {
		if role.TenantID == tenantID && role.IsSystemRole {
			result = append(result, role)
		}
	}
	return result, nil
}

func (m *mockRoleRepository) SavePermissions(ctx context.Context, role *identity.Role) error {
	return nil
}

func (m *mockRoleRepository) LoadPermissions(ctx context.Context, role *identity.Role) error {
	return nil
}

func (m *mockRoleRepository) SaveDataScopes(ctx context.Context, role *identity.Role) error {
	return nil
}

func (m *mockRoleRepository) LoadDataScopes(ctx context.Context, role *identity.Role) error {
	return nil
}

func (m *mockRoleRepository) LoadPermissionsAndDataScopes(ctx context.Context, role *identity.Role) error {
	// Just return the role as-is, data scopes should already be set
	return nil
}

func (m *mockRoleRepository) FindUsersWithRole(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

func (m *mockRoleRepository) CountUsersWithRole(ctx context.Context, roleID uuid.UUID) (int64, error) {
	return 0, nil
}

func (m *mockRoleRepository) FindRolesWithPermission(ctx context.Context, tenantID uuid.UUID, permissionCode string) ([]*identity.Role, error) {
	return nil, nil
}

func (m *mockRoleRepository) GetAllPermissionCodes(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	return nil, nil
}

func TestDataScopeMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("skips configured paths", func(t *testing.T) {
		mockRepo := newMockRoleRepository()
		middleware := DataScopeMiddlewareWithConfig(DataScopeMiddlewareConfig{
			RoleRepository: mockRepo,
			SkipPaths:      []string{"/health"},
		})

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)

		middleware(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Nil(t, GetDataScopeFilter(c))
	})

	t.Run("skips configured path prefixes", func(t *testing.T) {
		mockRepo := newMockRoleRepository()
		middleware := DataScopeMiddlewareWithConfig(DataScopeMiddlewareConfig{
			RoleRepository:   mockRepo,
			SkipPathPrefixes: []string{"/swagger"},
		})

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)

		middleware(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("continues without roles in context", func(t *testing.T) {
		mockRepo := newMockRoleRepository()
		middleware := DataScopeMiddleware(mockRepo)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)

		middleware(c)

		// No error, just continues
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("loads data scopes from roles", func(t *testing.T) {
		mockRepo := newMockRoleRepository()

		// Create a role with data scope
		role, _ := identity.NewRole(tenantID, "SALES", "Sales Role")
		ds, _ := identity.NewDataScope("sales_order", identity.DataScopeSelf)
		_ = role.SetDataScope(*ds)
		_ = mockRepo.Create(context.Background(), role)

		middleware := DataScopeMiddleware(mockRepo)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)

		// Set JWT context values
		c.Set(JWTTenantIDKey, tenantID.String())
		c.Set(JWTUserIDKey, userID.String())
		c.Set(JWTRoleIDsKey, []string{role.ID.String()})

		middleware(c)

		// Check that filter was created
		filter := GetDataScopeFilter(c)
		require.NotNil(t, filter)
		assert.Equal(t, identity.DataScopeSelf, filter.GetScopeType("sales_order"))
	})

	t.Run("merges data scopes from multiple roles", func(t *testing.T) {
		mockRepo := newMockRoleRepository()

		// Create role with SELF scope
		role1, _ := identity.NewRole(tenantID, "SALES", "Sales Role")
		ds1, _ := identity.NewDataScope("sales_order", identity.DataScopeSelf)
		_ = role1.SetDataScope(*ds1)
		_ = mockRepo.Create(context.Background(), role1)

		// Create role with ALL scope
		role2, _ := identity.NewRole(tenantID, "MANAGER", "Manager Role")
		ds2, _ := identity.NewDataScope("sales_order", identity.DataScopeAll)
		_ = role2.SetDataScope(*ds2)
		_ = mockRepo.Create(context.Background(), role2)

		middleware := DataScopeMiddleware(mockRepo)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)

		c.Set(JWTTenantIDKey, tenantID.String())
		c.Set(JWTUserIDKey, userID.String())
		c.Set(JWTRoleIDsKey, []string{role1.ID.String(), role2.ID.String()})

		middleware(c)

		filter := GetDataScopeFilter(c)
		require.NotNil(t, filter)
		// Higher scope (ALL) should win
		assert.Equal(t, identity.DataScopeAll, filter.GetScopeType("sales_order"))
	})

	t.Run("stores roles in context", func(t *testing.T) {
		mockRepo := newMockRoleRepository()

		role, _ := identity.NewRole(tenantID, "SALES", "Sales Role")
		_ = mockRepo.Create(context.Background(), role)

		middleware := DataScopeMiddleware(mockRepo)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)

		c.Set(JWTTenantIDKey, tenantID.String())
		c.Set(JWTUserIDKey, userID.String())
		c.Set(JWTRoleIDsKey, []string{role.ID.String()})

		middleware(c)

		roles := GetUserRoles(c)
		require.Len(t, roles, 1)
		assert.Equal(t, role.Code, roles[0].Code)
	})

	t.Run("filters roles by tenant ID", func(t *testing.T) {
		mockRepo := newMockRoleRepository()
		otherTenantID := uuid.New()

		// Create a role for a different tenant
		role, _ := identity.NewRole(otherTenantID, "SALES", "Sales Role")
		ds, _ := identity.NewDataScope("sales_order", identity.DataScopeSelf)
		_ = role.SetDataScope(*ds)
		_ = mockRepo.Create(context.Background(), role)

		middleware := DataScopeMiddleware(mockRepo)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)

		// Request is for a different tenant
		c.Set(JWTTenantIDKey, tenantID.String())
		c.Set(JWTUserIDKey, userID.String())
		c.Set(JWTRoleIDsKey, []string{role.ID.String()})

		middleware(c)

		// Role should be filtered out - no data scope filter
		roles := GetUserRoles(c)
		assert.Empty(t, roles)
	})
}

func TestGetDataScopeFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns nil when no filter set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		filter := GetDataScopeFilter(c)
		assert.Nil(t, filter)
	})

	t.Run("returns filter when set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		ctx := context.Background()
		expectedFilter := datascope.NewFilter(ctx, []identity.Role{})
		c.Set(DataScopeFilterKey, expectedFilter)

		filter := GetDataScopeFilter(c)
		assert.Equal(t, expectedFilter, filter)
	})

	t.Run("returns nil for wrong type", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Set(DataScopeFilterKey, "not a filter")

		filter := GetDataScopeFilter(c)
		assert.Nil(t, filter)
	})
}

func TestGetUserRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()

	t.Run("returns nil when no roles set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		roles := GetUserRoles(c)
		assert.Nil(t, roles)
	})

	t.Run("returns roles when set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		role, _ := identity.NewRole(tenantID, "SALES", "Sales Role")
		c.Set(UserRolesKey, []identity.Role{*role})

		roles := GetUserRoles(c)
		require.Len(t, roles, 1)
		assert.Equal(t, "SALES", roles[0].Code)
	})
}

func TestRequireDataScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()

	t.Run("allows access when no filter (no restrictions)", func(t *testing.T) {
		middleware := RequireDataScope("sales_order", identity.DataScopeAll, nil)

		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		called := false
		r.GET("/test", middleware, func(c *gin.Context) {
			called = true
			c.Status(http.StatusOK)
		})

		c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
		r.ServeHTTP(w, c.Request)

		assert.True(t, called)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("allows access when scope meets requirement", func(t *testing.T) {
		middleware := RequireDataScope("sales_order", identity.DataScopeSelf, nil)

		role, _ := identity.NewRole(tenantID, "SALES", "Sales Role")
		ds, _ := identity.NewDataScope("sales_order", identity.DataScopeAll)
		_ = role.SetDataScope(*ds)

		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		filter := datascope.NewFilter(context.Background(), []identity.Role{*role})

		called := false
		r.GET("/test", func(c *gin.Context) {
			c.Set(DataScopeFilterKey, filter)
			c.Next()
		}, middleware, func(c *gin.Context) {
			called = true
			c.Status(http.StatusOK)
		})

		c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
		r.ServeHTTP(w, c.Request)

		assert.True(t, called)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("denies access when scope insufficient", func(t *testing.T) {
		middleware := RequireDataScope("sales_order", identity.DataScopeAll, nil)

		role, _ := identity.NewRole(tenantID, "SALES", "Sales Role")
		ds, _ := identity.NewDataScope("sales_order", identity.DataScopeSelf)
		_ = role.SetDataScope(*ds)

		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		filter := datascope.NewFilter(context.Background(), []identity.Role{*role})

		called := false
		r.GET("/test", func(c *gin.Context) {
			c.Set(DataScopeFilterKey, filter)
			c.Next()
		}, middleware, func(c *gin.Context) {
			called = true
			c.Status(http.StatusOK)
		})

		c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
		r.ServeHTTP(w, c.Request)

		assert.False(t, called)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestMeetsMinimumScope(t *testing.T) {
	testCases := []struct {
		name     string
		actual   identity.DataScopeType
		min      identity.DataScopeType
		expected bool
	}{
		{"ALL meets ALL", identity.DataScopeAll, identity.DataScopeAll, true},
		{"ALL meets SELF", identity.DataScopeAll, identity.DataScopeSelf, true},
		{"ALL meets DEPARTMENT", identity.DataScopeAll, identity.DataScopeDepartment, true},
		{"DEPARTMENT meets SELF", identity.DataScopeDepartment, identity.DataScopeSelf, true},
		{"SELF meets SELF", identity.DataScopeSelf, identity.DataScopeSelf, true},
		{"SELF does not meet ALL", identity.DataScopeSelf, identity.DataScopeAll, false},
		{"SELF does not meet DEPARTMENT", identity.DataScopeSelf, identity.DataScopeDepartment, false},
		{"DEPARTMENT does not meet ALL", identity.DataScopeDepartment, identity.DataScopeAll, false},
		{"CUSTOM meets SELF", identity.DataScopeCustom, identity.DataScopeSelf, true},
		{"CUSTOM does not meet DEPARTMENT", identity.DataScopeCustom, identity.DataScopeDepartment, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := meetsMinimumScope(tc.actual, tc.min)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDefaultDataScopeConfig(t *testing.T) {
	mockRepo := newMockRoleRepository()
	config := DefaultDataScopeConfig(mockRepo)

	assert.Equal(t, mockRepo, config.RoleRepository)
	assert.Contains(t, config.SkipPaths, "/health")
	assert.Contains(t, config.SkipPaths, "/api/v1/auth/login")
	assert.Contains(t, config.SkipPathPrefixes, "/swagger")
}
