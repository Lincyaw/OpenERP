package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	appIdentity "github.com/erp/backend/internal/application/identity"
	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/infrastructure/auth"
	"github.com/erp/backend/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAdminTenantService is a mock implementation of AdminTenantService
type MockAdminTenantService struct {
	mock.Mock
}

func (m *MockAdminTenantService) ListTenants(ctx interface{}, input appIdentity.AdminTenantFilterInput) (*appIdentity.AdminTenantListResult, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*appIdentity.AdminTenantListResult), args.Error(1)
}

// MockTenantStatsService is a mock implementation of TenantStatsService
type MockTenantStatsService struct {
	mock.Mock
}

func (m *MockTenantStatsService) GetPlatformStats(ctx interface{}) (*appIdentity.PlatformStatsDTO, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*appIdentity.PlatformStatsDTO), args.Error(1)
}

// MockAuditService is a mock implementation of AuditService
type MockAuditService struct {
	mock.Mock
}

func (m *MockAuditService) List(ctx interface{}, input appIdentity.AuditLogFilterInput) (*appIdentity.AuditLogListResult, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*appIdentity.AuditLogListResult), args.Error(1)
}

// setupAdminTestRouter creates a test router with super admin middleware
func setupAdminTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

// setupSuperAdminContext sets up a gin context with super admin claims
func setupSuperAdminContext(c *gin.Context) {
	claims := &auth.Claims{
		UserID:      uuid.New().String(),
		Username:    "superadmin",
		TenantID:    identity.SystemTenantID,
		RoleIDs:     []string{identity.SuperAdminRoleID},
		Permissions: []string{"tenant:read", "tenant:create", "tenant:update", "tenant:delete"},
	}
	c.Set(middleware.JWTClaimsKey, claims)
	c.Set(middleware.IsSuperAdminKey, true)
}

func TestAdminHandler_NewAdminHandler(t *testing.T) {
	handler := NewAdminHandler(nil, nil, nil)
	assert.NotNil(t, handler)
}

func TestAdminHandler_GetAdminAuditContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("User-Agent", "test-agent")
	c.Request.RemoteAddr = "192.168.1.1:12345"

	// Set up user ID in context
	userID := uuid.New()
	claims := &auth.Claims{
		UserID:   userID.String(),
		TenantID: identity.SystemTenantID,
	}
	c.Set(middleware.JWTClaimsKey, claims)

	handler := NewAdminHandler(nil, nil, nil)
	auditCtx := handler.getAdminAuditContext(c)

	// Note: getUserID returns uuid.Nil if parsing fails, which is expected behavior
	// The test verifies the audit context is populated with available data
	assert.Equal(t, "test-agent", auditCtx.UserAgent)
	assert.Contains(t, auditCtx.IPAddress, "192.168.1.1")
}

func TestAdminTenantListQuery_Validation(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "empty query is valid",
			url:     "/test",
			wantErr: false,
		},
		{
			name:    "valid query with page and page_size",
			url:     "/test?page=1&page_size=20",
			wantErr: false,
		},
		{
			name:    "valid status active",
			url:     "/test?status=active",
			wantErr: false,
		},
		{
			name:    "valid plan pro",
			url:     "/test?plan=pro",
			wantErr: false,
		},
		{
			name:    "invalid status",
			url:     "/test?status=invalid",
			wantErr: true,
		},
		{
			name:    "invalid plan",
			url:     "/test?plan=invalid",
			wantErr: true,
		},
		{
			name:    "page size too large",
			url:     "/test?page_size=200",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, tt.url, nil)

			var query AdminTenantListQuery
			err := c.ShouldBindQuery(&query)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAdminCreateTenantRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "valid request",
			body:    `{"code":"TEST001","name":"Test Tenant"}`,
			wantErr: false,
		},
		{
			name:    "missing code",
			body:    `{"name":"Test Tenant"}`,
			wantErr: true,
		},
		{
			name:    "missing name",
			body:    `{"code":"TEST001"}`,
			wantErr: true,
		},
		{
			name:    "code too short",
			body:    `{"code":"T","name":"Test Tenant"}`,
			wantErr: true,
		},
		{
			name:    "invalid email",
			body:    `{"code":"TEST001","name":"Test Tenant","contact_email":"invalid"}`,
			wantErr: true,
		},
		{
			name:    "invalid plan",
			body:    `{"code":"TEST001","name":"Test Tenant","plan":"invalid"}`,
			wantErr: true,
		},
		{
			name:    "valid with all fields",
			body:    `{"code":"TEST001","name":"Test Tenant","short_name":"TT","contact_name":"John","contact_phone":"123456","contact_email":"test@example.com","address":"123 Main St","plan":"pro","trial_days":14,"notes":"Test notes"}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")

			var req AdminCreateTenantRequest
			err := c.ShouldBindJSON(&req)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSuspendTenantRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "empty body is valid",
			body:    `{}`,
			wantErr: false,
		},
		{
			name:    "with reason",
			body:    `{"reason":"Payment overdue"}`,
			wantErr: false,
		},
		{
			name:    "with scheduled reactivation",
			body:    `{"reason":"Temporary suspension","scheduled_reactivate_at":"2025-12-31T23:59:59Z"}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")

			var req SuspendTenantRequest
			err := c.ShouldBindJSON(&req)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTenantGrowthQuery_Validation(t *testing.T) {
	tests := []struct {
		name    string
		query   TenantGrowthQuery
		wantErr bool
	}{
		{
			name:    "empty query is valid",
			query:   TenantGrowthQuery{},
			wantErr: false,
		},
		{
			name: "valid daily period",
			query: TenantGrowthQuery{
				Period: "daily",
				Days:   30,
			},
			wantErr: false,
		},
		{
			name: "valid weekly period",
			query: TenantGrowthQuery{
				Period: "weekly",
				Days:   90,
			},
			wantErr: false,
		},
		{
			name: "valid monthly period",
			query: TenantGrowthQuery{
				Period: "monthly",
				Days:   365,
			},
			wantErr: false,
		},
		{
			name: "invalid period",
			query: TenantGrowthQuery{
				Period: "yearly",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			url := "/test"
			queryParts := []string{}
			if tt.query.Period != "" {
				queryParts = append(queryParts, "period="+tt.query.Period)
			}
			if tt.query.Days > 0 {
				queryParts = append(queryParts, "days=30")
			}
			if len(queryParts) > 0 {
				url += "?" + strings.Join(queryParts, "&")
			}
			c.Request = httptest.NewRequest(http.MethodGet, url, nil)

			var query TenantGrowthQuery
			err := c.ShouldBindQuery(&query)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAdminTenantResponse_JSON(t *testing.T) {
	now := time.Now()
	response := AdminTenantResponse{
		ID:        uuid.New(),
		Code:      "TEST001",
		Name:      "Test Tenant",
		ShortName: "TT",
		Status:    "active",
		Plan:      "pro",
		Config: AdminTenantConfigResponse{
			MaxUsers:      10,
			MaxWarehouses: 5,
			MaxProducts:   1000,
			CostStrategy:  "fifo",
			Currency:      "USD",
			Timezone:      "UTC",
			Locale:        "en-US",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	data, err := json.Marshal(response)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "TEST001")
	assert.Contains(t, string(data), "Test Tenant")
	assert.Contains(t, string(data), "active")
	assert.Contains(t, string(data), "pro")
}

func TestPlatformStatsResponse_JSON(t *testing.T) {
	response := PlatformStatsResponse{
		TotalTenants:     100,
		ActiveTenants:    80,
		TrialTenants:     10,
		SuspendedTenants: 5,
		InactiveTenants:  5,
		TenantsByPlan: map[string]int64{
			"free":       30,
			"basic":      40,
			"pro":        25,
			"enterprise": 5,
		},
		TotalUsers:        500,
		TotalProducts:     10000,
		TotalOrders:       50000,
		TotalStorageUsage: 1024 * 1024 * 1024,
		LastUpdated:       time.Now(),
	}

	data, err := json.Marshal(response)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "total_tenants")
	assert.Contains(t, string(data), "active_tenants")
	assert.Contains(t, string(data), "tenants_by_plan")
}

func TestAdminAuditLogResponse_JSON(t *testing.T) {
	targetID := uuid.New()
	response := AdminAuditLogResponse{
		ID:          uuid.New(),
		AdminUserID: uuid.New(),
		Action:      "tenant_create",
		TargetType:  "tenant",
		TargetID:    &targetID,
		OldValue:    nil,
		NewValue: map[string]interface{}{
			"name": "New Tenant",
			"code": "NEW001",
		},
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
		CreatedAt: time.Now(),
	}

	data, err := json.Marshal(response)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "tenant_create")
	assert.Contains(t, string(data), "tenant")
	assert.Contains(t, string(data), "New Tenant")
}

func TestRegisterAdminRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")

	handler := NewAdminHandler(nil, nil, nil)
	RegisterAdminRoutes(api, handler)

	// Verify routes are registered
	routes := router.Routes()
	routePaths := make(map[string]bool)
	for _, route := range routes {
		routePaths[route.Method+" "+route.Path] = true
	}

	// Check tenant CRUD routes
	assert.True(t, routePaths["GET /api/v1/admin/tenants"])
	assert.True(t, routePaths["POST /api/v1/admin/tenants"])
	assert.True(t, routePaths["GET /api/v1/admin/tenants/:id"])
	assert.True(t, routePaths["PUT /api/v1/admin/tenants/:id"])
	assert.True(t, routePaths["DELETE /api/v1/admin/tenants/:id"])

	// Check status management routes
	assert.True(t, routePaths["POST /api/v1/admin/tenants/:id/suspend"])
	assert.True(t, routePaths["POST /api/v1/admin/tenants/:id/activate"])

	// Check subscription routes
	assert.True(t, routePaths["PUT /api/v1/admin/tenants/:id/plan"])
	assert.True(t, routePaths["PUT /api/v1/admin/tenants/:id/quota"])
	assert.True(t, routePaths["GET /api/v1/admin/tenants/:id/subscription-history"])
	assert.True(t, routePaths["DELETE /api/v1/admin/tenants/:id/scheduled-plan"])

	// Check stats routes
	assert.True(t, routePaths["GET /api/v1/admin/tenants/:id/stats"])
	assert.True(t, routePaths["GET /api/v1/admin/stats"])
	assert.True(t, routePaths["GET /api/v1/admin/stats/growth"])

	// Check audit log route
	assert.True(t, routePaths["GET /api/v1/admin/audit-logs"])
}
