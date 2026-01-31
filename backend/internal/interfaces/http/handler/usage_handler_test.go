package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockUserRepository is a mock implementation of identity.UserRepository for testing
type mockUserRepository struct {
	count int64
	err   error
}

func (m *mockUserRepository) Create(ctx context.Context, user *identity.User) error {
	return nil
}

func (m *mockUserRepository) Update(ctx context.Context, user *identity.User) error {
	return nil
}

func (m *mockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*identity.User, error) {
	return nil, nil
}

func (m *mockUserRepository) FindByUsername(ctx context.Context, username string) (*identity.User, error) {
	return nil, nil
}

func (m *mockUserRepository) FindByEmail(ctx context.Context, email string) (*identity.User, error) {
	return nil, nil
}

func (m *mockUserRepository) FindByPhone(ctx context.Context, phone string) (*identity.User, error) {
	return nil, nil
}

func (m *mockUserRepository) FindAll(ctx context.Context, filter identity.UserFilter) ([]*identity.User, int64, error) {
	return nil, 0, nil
}

func (m *mockUserRepository) FindByRoleID(ctx context.Context, roleID uuid.UUID) ([]*identity.User, error) {
	return nil, nil
}

func (m *mockUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	return false, nil
}

func (m *mockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return false, nil
}

func (m *mockUserRepository) SaveUserRoles(ctx context.Context, user *identity.User) error {
	return nil
}

func (m *mockUserRepository) LoadUserRoles(ctx context.Context, user *identity.User) error {
	return nil
}

func (m *mockUserRepository) Count(ctx context.Context) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.count, nil
}

// mockWarehouseCounter is a mock implementation of WarehouseCounter
type mockWarehouseCounter struct {
	count int64
	err   error
}

func (m *mockWarehouseCounter) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.count, nil
}

// mockProductCounter is a mock implementation of ProductCounter
type mockProductCounter struct {
	count int64
	err   error
}

func (m *mockProductCounter) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.count, nil
}

func createTestTenant(plan identity.TenantPlan) *identity.Tenant {
	tenant, _ := identity.NewTenant("TEST", "Test Tenant")
	_ = tenant.SetPlan(plan)
	return tenant
}

func TestUsageHandler_GetCurrentUsage(t *testing.T) {
	tenantID := uuid.New()
	tenant := createTestTenant(identity.TenantPlanBasic)
	tenant.ID = tenantID

	tests := []struct {
		name           string
		tenantID       string
		mockTenantRepo *mockTenantRepository
		mockUserRepo   *mockUserRepository
		mockWarehouse  *mockWarehouseCounter
		mockProduct    *mockProductCounter
		expectedStatus int
		expectSuccess  bool
	}{
		{
			name:     "valid usage retrieval",
			tenantID: tenantID.String(),
			mockTenantRepo: &mockTenantRepository{
				tenant: tenant,
			},
			mockUserRepo:   &mockUserRepository{count: 5},
			mockWarehouse:  &mockWarehouseCounter{count: 2},
			mockProduct:    &mockProductCounter{count: 100},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name:           "missing tenant ID",
			tenantID:       "",
			mockTenantRepo: &mockTenantRepository{},
			mockUserRepo:   &mockUserRepository{},
			mockWarehouse:  &mockWarehouseCounter{},
			mockProduct:    &mockProductCounter{},
			expectedStatus: http.StatusUnauthorized,
			expectSuccess:  false,
		},
		{
			name:           "invalid tenant ID format",
			tenantID:       "invalid-uuid",
			mockTenantRepo: &mockTenantRepository{},
			mockUserRepo:   &mockUserRepository{},
			mockWarehouse:  &mockWarehouseCounter{},
			mockProduct:    &mockProductCounter{},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
		{
			name:     "tenant not found",
			tenantID: tenantID.String(),
			mockTenantRepo: &mockTenantRepository{
				err: shared.ErrNotFound,
			},
			mockUserRepo:   &mockUserRepository{},
			mockWarehouse:  &mockWarehouseCounter{},
			mockProduct:    &mockProductCounter{},
			expectedStatus: http.StatusNotFound,
			expectSuccess:  false,
		},
		{
			name:     "user count error",
			tenantID: tenantID.String(),
			mockTenantRepo: &mockTenantRepository{
				tenant: tenant,
			},
			mockUserRepo:   &mockUserRepository{err: errors.New("db error")},
			mockWarehouse:  &mockWarehouseCounter{count: 2},
			mockProduct:    &mockProductCounter{count: 100},
			expectedStatus: http.StatusInternalServerError,
			expectSuccess:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewUsageHandler(tt.mockTenantRepo, tt.mockUserRepo, tt.mockWarehouse, tt.mockProduct)

			router := gin.New()
			router.GET("/tenants/current/usage", func(c *gin.Context) {
				if tt.tenantID != "" {
					c.Set("jwt_tenant_id", tt.tenantID)
				}
				h.GetCurrentUsage(c)
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/tenants/current/usage", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var resp struct {
				Success bool `json:"success"`
				Data    struct {
					TenantID   string        `json:"tenant_id"`
					TenantName string        `json:"tenant_name"`
					Plan       string        `json:"plan"`
					Metrics    []UsageMetric `json:"metrics"`
				} `json:"data"`
			}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			if tt.expectSuccess {
				assert.True(t, resp.Success)
				assert.Equal(t, tenantID.String(), resp.Data.TenantID)
				assert.Equal(t, "basic", resp.Data.Plan)
				assert.Len(t, resp.Data.Metrics, 3)

				// Verify metrics
				for _, m := range resp.Data.Metrics {
					assert.Contains(t, []string{"users", "warehouses", "products"}, m.Name)
				}
			}
		})
	}
}

func TestUsageHandler_GetUsageHistory(t *testing.T) {
	tenantID := uuid.New()
	tenant := createTestTenant(identity.TenantPlanBasic)
	tenant.ID = tenantID

	tests := []struct {
		name           string
		tenantID       string
		queryParams    string
		mockTenantRepo *mockTenantRepository
		mockUserRepo   *mockUserRepository
		mockWarehouse  *mockWarehouseCounter
		mockProduct    *mockProductCounter
		expectedStatus int
		expectSuccess  bool
	}{
		{
			name:           "valid daily history",
			tenantID:       tenantID.String(),
			queryParams:    "?period=daily",
			mockTenantRepo: &mockTenantRepository{tenant: tenant},
			mockUserRepo:   &mockUserRepository{count: 5},
			mockWarehouse:  &mockWarehouseCounter{count: 2},
			mockProduct:    &mockProductCounter{count: 100},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name:           "valid weekly history",
			tenantID:       tenantID.String(),
			queryParams:    "?period=weekly",
			mockTenantRepo: &mockTenantRepository{tenant: tenant},
			mockUserRepo:   &mockUserRepository{count: 5},
			mockWarehouse:  &mockWarehouseCounter{count: 2},
			mockProduct:    &mockProductCounter{count: 100},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name:           "valid monthly history",
			tenantID:       tenantID.String(),
			queryParams:    "?period=monthly",
			mockTenantRepo: &mockTenantRepository{tenant: tenant},
			mockUserRepo:   &mockUserRepository{count: 5},
			mockWarehouse:  &mockWarehouseCounter{count: 2},
			mockProduct:    &mockProductCounter{count: 100},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name:           "invalid period",
			tenantID:       tenantID.String(),
			queryParams:    "?period=invalid",
			mockTenantRepo: &mockTenantRepository{tenant: tenant},
			mockUserRepo:   &mockUserRepository{count: 5},
			mockWarehouse:  &mockWarehouseCounter{count: 2},
			mockProduct:    &mockProductCounter{count: 100},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
		{
			name:           "invalid start_date format",
			tenantID:       tenantID.String(),
			queryParams:    "?start_date=invalid",
			mockTenantRepo: &mockTenantRepository{tenant: tenant},
			mockUserRepo:   &mockUserRepository{count: 5},
			mockWarehouse:  &mockWarehouseCounter{count: 2},
			mockProduct:    &mockProductCounter{count: 100},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
		{
			name:           "invalid end_date format",
			tenantID:       tenantID.String(),
			queryParams:    "?end_date=invalid",
			mockTenantRepo: &mockTenantRepository{tenant: tenant},
			mockUserRepo:   &mockUserRepository{count: 5},
			mockWarehouse:  &mockWarehouseCounter{count: 2},
			mockProduct:    &mockProductCounter{count: 100},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
		{
			name:           "start_date after end_date",
			tenantID:       tenantID.String(),
			queryParams:    "?start_date=2024-12-31&end_date=2024-01-01",
			mockTenantRepo: &mockTenantRepository{tenant: tenant},
			mockUserRepo:   &mockUserRepository{count: 5},
			mockWarehouse:  &mockWarehouseCounter{count: 2},
			mockProduct:    &mockProductCounter{count: 100},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
		{
			name:           "missing tenant ID",
			tenantID:       "",
			queryParams:    "",
			mockTenantRepo: &mockTenantRepository{},
			mockUserRepo:   &mockUserRepository{},
			mockWarehouse:  &mockWarehouseCounter{},
			mockProduct:    &mockProductCounter{},
			expectedStatus: http.StatusUnauthorized,
			expectSuccess:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewUsageHandler(tt.mockTenantRepo, tt.mockUserRepo, tt.mockWarehouse, tt.mockProduct)

			router := gin.New()
			router.GET("/tenants/current/usage/history", func(c *gin.Context) {
				if tt.tenantID != "" {
					c.Set("jwt_tenant_id", tt.tenantID)
				}
				h.GetUsageHistory(c)
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/tenants/current/usage/history"+tt.queryParams, nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var resp struct {
				Success bool `json:"success"`
				Data    struct {
					TenantID   string              `json:"tenant_id"`
					Period     string              `json:"period"`
					DataPoints []UsageHistoryPoint `json:"data_points"`
				} `json:"data"`
			}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			if tt.expectSuccess {
				assert.True(t, resp.Success)
				assert.NotEmpty(t, resp.Data.DataPoints)
			}
		})
	}
}

func TestUsageHandler_GetQuotas(t *testing.T) {
	tenantID := uuid.New()
	tenant := createTestTenant(identity.TenantPlanBasic)
	tenant.ID = tenantID

	enterpriseTenant := createTestTenant(identity.TenantPlanEnterprise)
	enterpriseTenant.ID = tenantID

	tests := []struct {
		name           string
		tenantID       string
		mockTenantRepo *mockTenantRepository
		mockUserRepo   *mockUserRepository
		mockWarehouse  *mockWarehouseCounter
		mockProduct    *mockProductCounter
		expectedStatus int
		expectSuccess  bool
		checkUnlimited bool
	}{
		{
			name:     "valid quota retrieval - basic plan",
			tenantID: tenantID.String(),
			mockTenantRepo: &mockTenantRepository{
				tenant: tenant,
			},
			mockUserRepo:   &mockUserRepository{count: 5},
			mockWarehouse:  &mockWarehouseCounter{count: 2},
			mockProduct:    &mockProductCounter{count: 100},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
			checkUnlimited: false,
		},
		{
			name:     "valid quota retrieval - enterprise plan (unlimited)",
			tenantID: tenantID.String(),
			mockTenantRepo: &mockTenantRepository{
				tenant: enterpriseTenant,
			},
			mockUserRepo:   &mockUserRepository{count: 50},
			mockWarehouse:  &mockWarehouseCounter{count: 10},
			mockProduct:    &mockProductCounter{count: 10000},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
			checkUnlimited: true,
		},
		{
			name:           "missing tenant ID",
			tenantID:       "",
			mockTenantRepo: &mockTenantRepository{},
			mockUserRepo:   &mockUserRepository{},
			mockWarehouse:  &mockWarehouseCounter{},
			mockProduct:    &mockProductCounter{},
			expectedStatus: http.StatusUnauthorized,
			expectSuccess:  false,
		},
		{
			name:     "tenant not found",
			tenantID: tenantID.String(),
			mockTenantRepo: &mockTenantRepository{
				err: shared.ErrNotFound,
			},
			mockUserRepo:   &mockUserRepository{},
			mockWarehouse:  &mockWarehouseCounter{},
			mockProduct:    &mockProductCounter{},
			expectedStatus: http.StatusNotFound,
			expectSuccess:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewUsageHandler(tt.mockTenantRepo, tt.mockUserRepo, tt.mockWarehouse, tt.mockProduct)

			router := gin.New()
			router.GET("/tenants/current/quotas", func(c *gin.Context) {
				if tt.tenantID != "" {
					c.Set("jwt_tenant_id", tt.tenantID)
				}
				h.GetQuotas(c)
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/tenants/current/quotas", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var resp struct {
				Success bool `json:"success"`
				Data    struct {
					TenantID string      `json:"tenant_id"`
					Plan     string      `json:"plan"`
					Quotas   []QuotaItem `json:"quotas"`
				} `json:"data"`
			}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			if tt.expectSuccess {
				assert.True(t, resp.Success)
				assert.Len(t, resp.Data.Quotas, 3)

				if tt.checkUnlimited {
					for _, q := range resp.Data.Quotas {
						assert.True(t, q.IsUnlimited)
						assert.Equal(t, int64(-1), q.Remaining)
					}
				}
			}
		})
	}
}

func TestUsageHandler_GetTenantUsageByAdmin(t *testing.T) {
	tenantID := uuid.New()
	tenant := createTestTenant(identity.TenantPlanPro)
	tenant.ID = tenantID

	tests := []struct {
		name           string
		pathTenantID   string
		mockTenantRepo *mockTenantRepository
		mockUserRepo   *mockUserRepository
		mockWarehouse  *mockWarehouseCounter
		mockProduct    *mockProductCounter
		expectedStatus int
		expectSuccess  bool
	}{
		{
			name:         "valid admin usage retrieval",
			pathTenantID: tenantID.String(),
			mockTenantRepo: &mockTenantRepository{
				tenant: tenant,
			},
			mockUserRepo:   &mockUserRepository{count: 10},
			mockWarehouse:  &mockWarehouseCounter{count: 5},
			mockProduct:    &mockProductCounter{count: 500},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name:           "missing tenant ID",
			pathTenantID:   "",
			mockTenantRepo: &mockTenantRepository{},
			mockUserRepo:   &mockUserRepository{},
			mockWarehouse:  &mockWarehouseCounter{},
			mockProduct:    &mockProductCounter{},
			expectedStatus: http.StatusBadRequest, // Empty ID is caught by handler validation
			expectSuccess:  false,
		},
		{
			name:           "invalid tenant ID format",
			pathTenantID:   "invalid-uuid",
			mockTenantRepo: &mockTenantRepository{},
			mockUserRepo:   &mockUserRepository{},
			mockWarehouse:  &mockWarehouseCounter{},
			mockProduct:    &mockProductCounter{},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
		{
			name:         "tenant not found",
			pathTenantID: uuid.New().String(),
			mockTenantRepo: &mockTenantRepository{
				err: shared.ErrNotFound,
			},
			mockUserRepo:   &mockUserRepository{},
			mockWarehouse:  &mockWarehouseCounter{},
			mockProduct:    &mockProductCounter{},
			expectedStatus: http.StatusNotFound,
			expectSuccess:  false,
		},
		{
			name:         "internal error on metrics",
			pathTenantID: tenantID.String(),
			mockTenantRepo: &mockTenantRepository{
				tenant: tenant,
			},
			mockUserRepo:   &mockUserRepository{err: errors.New("db error")},
			mockWarehouse:  &mockWarehouseCounter{count: 5},
			mockProduct:    &mockProductCounter{count: 500},
			expectedStatus: http.StatusInternalServerError,
			expectSuccess:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewUsageHandler(tt.mockTenantRepo, tt.mockUserRepo, tt.mockWarehouse, tt.mockProduct)

			router := gin.New()
			router.GET("/admin/tenants/:id/usage", h.GetTenantUsageByAdmin)

			w := httptest.NewRecorder()
			path := "/admin/tenants/" + tt.pathTenantID + "/usage"
			if tt.pathTenantID == "" {
				path = "/admin/tenants//usage"
			}
			req := httptest.NewRequest("GET", path, nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectSuccess {
				var resp struct {
					Success bool `json:"success"`
					Data    struct {
						TenantID   string        `json:"tenant_id"`
						TenantCode string        `json:"tenant_code"`
						TenantName string        `json:"tenant_name"`
						Plan       string        `json:"plan"`
						Status     string        `json:"status"`
						Metrics    []UsageMetric `json:"metrics"`
						Quotas     []QuotaItem   `json:"quotas"`
					} `json:"data"`
				}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)

				assert.True(t, resp.Success)
				assert.Equal(t, tenantID.String(), resp.Data.TenantID)
				assert.Equal(t, "pro", resp.Data.Plan)
				assert.Len(t, resp.Data.Metrics, 3)
				assert.Len(t, resp.Data.Quotas, 3)
			}
		})
	}
}

func TestCalculatePercentage(t *testing.T) {
	tests := []struct {
		name     string
		current  int64
		limit    int64
		expected float64
	}{
		{"50%", 5, 10, 50.0},
		{"100%", 10, 10, 100.0},
		{"0%", 0, 10, 0.0},
		{"over 100%", 15, 10, 150.0},
		{"zero limit", 5, 0, 0.0},
		{"negative limit", 5, -1, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculatePercentage(tt.current, tt.limit)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateRemaining(t *testing.T) {
	tests := []struct {
		name        string
		used        int64
		limit       int64
		isUnlimited bool
		expected    int64
	}{
		{"normal remaining", 5, 10, false, 5},
		{"no remaining", 10, 10, false, 0},
		{"over limit", 15, 10, false, 0},
		{"unlimited", 100, 10, true, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateRemaining(tt.used, tt.limit, tt.isUnlimited)
			assert.Equal(t, tt.expected, result)
		})
	}
}
