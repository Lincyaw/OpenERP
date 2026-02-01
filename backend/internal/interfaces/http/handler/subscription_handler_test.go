package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPlanFeatureRepository is a mock implementation of identity.PlanFeatureRepository
type mockPlanFeatureRepository struct {
	features []identity.PlanFeature
	err      error
}

func (m *mockPlanFeatureRepository) FindByID(ctx context.Context, id uuid.UUID) (*identity.PlanFeature, error) {
	return nil, nil
}

func (m *mockPlanFeatureRepository) FindByPlan(ctx context.Context, planID identity.TenantPlan) ([]identity.PlanFeature, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.features, nil
}

func (m *mockPlanFeatureRepository) FindByPlanAndFeature(ctx context.Context, planID identity.TenantPlan, featureKey identity.FeatureKey) (*identity.PlanFeature, error) {
	return nil, nil
}

func (m *mockPlanFeatureRepository) FindEnabledByPlan(ctx context.Context, planID identity.TenantPlan) ([]identity.PlanFeature, error) {
	return nil, nil
}

func (m *mockPlanFeatureRepository) HasFeature(ctx context.Context, planID identity.TenantPlan, featureKey identity.FeatureKey) (bool, error) {
	return false, nil
}

func (m *mockPlanFeatureRepository) GetFeatureLimit(ctx context.Context, planID identity.TenantPlan, featureKey identity.FeatureKey) (*int, error) {
	return nil, nil
}

func (m *mockPlanFeatureRepository) Save(ctx context.Context, feature *identity.PlanFeature) error {
	return nil
}

func (m *mockPlanFeatureRepository) SaveBatch(ctx context.Context, features []identity.PlanFeature) error {
	return nil
}

func (m *mockPlanFeatureRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockPlanFeatureRepository) DeleteByPlan(ctx context.Context, planID identity.TenantPlan) error {
	return nil
}

func createSubscriptionTestTenant(plan identity.TenantPlan) *identity.Tenant {
	tenant, _ := identity.NewTenant("TEST", "Test Tenant")
	_ = tenant.SetPlan(plan)
	return tenant
}

func createTrialTenant() *identity.Tenant {
	tenant, _ := identity.NewTrialTenant("TRIAL", "Trial Tenant", 14)
	return tenant
}

func TestSubscriptionHandler_GetCurrentSubscription(t *testing.T) {
	tenantID := uuid.New()
	tenant := createSubscriptionTestTenant(identity.TenantPlanBasic)
	tenant.ID = tenantID

	enterpriseTenant := createSubscriptionTestTenant(identity.TenantPlanEnterprise)
	enterpriseTenant.ID = tenantID

	trialTenant := createTrialTenant()
	trialTenant.ID = tenantID

	// Create tenant with expiration
	expiringTenant := createSubscriptionTestTenant(identity.TenantPlanPro)
	expiringTenant.ID = tenantID
	expiresAt := time.Now().AddDate(0, 1, 0)
	expiringTenant.SetExpiration(expiresAt)

	tests := []struct {
		name                string
		tenantID            string
		mockTenantRepo      *mockTenantRepository
		mockPlanFeatureRepo *mockPlanFeatureRepository
		mockUserRepo        *mockUserRepository
		mockWarehouse       *mockWarehouseCounter
		mockProduct         *mockProductCounter
		expectedStatus      int
		expectSuccess       bool
		expectedPlanID      string
		expectedStatus_     string
		checkUnlimited      bool
		checkTrialEnds      bool
		checkPeriodDates    bool
	}{
		{
			name:     "valid subscription - basic plan",
			tenantID: tenantID.String(),
			mockTenantRepo: &mockTenantRepository{
				tenant: tenant,
			},
			mockPlanFeatureRepo: &mockPlanFeatureRepository{},
			mockUserRepo:        &mockUserRepository{count: 5},
			mockWarehouse:       &mockWarehouseCounter{count: 2},
			mockProduct:         &mockProductCounter{count: 100},
			expectedStatus:      http.StatusOK,
			expectSuccess:       true,
			expectedPlanID:      "basic",
			expectedStatus_:     "active",
		},
		{
			name:     "valid subscription - enterprise plan (unlimited)",
			tenantID: tenantID.String(),
			mockTenantRepo: &mockTenantRepository{
				tenant: enterpriseTenant,
			},
			mockPlanFeatureRepo: &mockPlanFeatureRepository{},
			mockUserRepo:        &mockUserRepository{count: 50},
			mockWarehouse:       &mockWarehouseCounter{count: 10},
			mockProduct:         &mockProductCounter{count: 10000},
			expectedStatus:      http.StatusOK,
			expectSuccess:       true,
			expectedPlanID:      "enterprise",
			expectedStatus_:     "active",
			checkUnlimited:      true,
		},
		{
			name:     "valid subscription - trial status",
			tenantID: tenantID.String(),
			mockTenantRepo: &mockTenantRepository{
				tenant: trialTenant,
			},
			mockPlanFeatureRepo: &mockPlanFeatureRepository{},
			mockUserRepo:        &mockUserRepository{count: 2},
			mockWarehouse:       &mockWarehouseCounter{count: 1},
			mockProduct:         &mockProductCounter{count: 50},
			expectedStatus:      http.StatusOK,
			expectSuccess:       true,
			expectedPlanID:      "free",
			expectedStatus_:     "trial",
			checkTrialEnds:      true,
		},
		{
			name:     "valid subscription - with expiration dates",
			tenantID: tenantID.String(),
			mockTenantRepo: &mockTenantRepository{
				tenant: expiringTenant,
			},
			mockPlanFeatureRepo: &mockPlanFeatureRepository{},
			mockUserRepo:        &mockUserRepository{count: 10},
			mockWarehouse:       &mockWarehouseCounter{count: 5},
			mockProduct:         &mockProductCounter{count: 500},
			expectedStatus:      http.StatusOK,
			expectSuccess:       true,
			expectedPlanID:      "pro",
			expectedStatus_:     "active",
			checkPeriodDates:    true,
		},
		{
			name:                "missing tenant ID",
			tenantID:            "",
			mockTenantRepo:      &mockTenantRepository{},
			mockPlanFeatureRepo: &mockPlanFeatureRepository{},
			mockUserRepo:        &mockUserRepository{},
			mockWarehouse:       &mockWarehouseCounter{},
			mockProduct:         &mockProductCounter{},
			expectedStatus:      http.StatusUnauthorized,
			expectSuccess:       false,
		},
		{
			name:                "invalid tenant ID format",
			tenantID:            "invalid-uuid",
			mockTenantRepo:      &mockTenantRepository{},
			mockPlanFeatureRepo: &mockPlanFeatureRepository{},
			mockUserRepo:        &mockUserRepository{},
			mockWarehouse:       &mockWarehouseCounter{},
			mockProduct:         &mockProductCounter{},
			expectedStatus:      http.StatusBadRequest,
			expectSuccess:       false,
		},
		{
			name:     "tenant not found",
			tenantID: tenantID.String(),
			mockTenantRepo: &mockTenantRepository{
				err: shared.ErrNotFound,
			},
			mockPlanFeatureRepo: &mockPlanFeatureRepository{},
			mockUserRepo:        &mockUserRepository{},
			mockWarehouse:       &mockWarehouseCounter{},
			mockProduct:         &mockProductCounter{},
			expectedStatus:      http.StatusNotFound,
			expectSuccess:       false,
		},
		{
			name:     "user count error",
			tenantID: tenantID.String(),
			mockTenantRepo: &mockTenantRepository{
				tenant: tenant,
			},
			mockPlanFeatureRepo: &mockPlanFeatureRepository{},
			mockUserRepo:        &mockUserRepository{err: errors.New("db error")},
			mockWarehouse:       &mockWarehouseCounter{count: 2},
			mockProduct:         &mockProductCounter{count: 100},
			expectedStatus:      http.StatusInternalServerError,
			expectSuccess:       false,
		},
		{
			name:     "warehouse count error",
			tenantID: tenantID.String(),
			mockTenantRepo: &mockTenantRepository{
				tenant: tenant,
			},
			mockPlanFeatureRepo: &mockPlanFeatureRepository{},
			mockUserRepo:        &mockUserRepository{count: 5},
			mockWarehouse:       &mockWarehouseCounter{err: errors.New("db error")},
			mockProduct:         &mockProductCounter{count: 100},
			expectedStatus:      http.StatusInternalServerError,
			expectSuccess:       false,
		},
		{
			name:     "product count error",
			tenantID: tenantID.String(),
			mockTenantRepo: &mockTenantRepository{
				tenant: tenant,
			},
			mockPlanFeatureRepo: &mockPlanFeatureRepository{},
			mockUserRepo:        &mockUserRepository{count: 5},
			mockWarehouse:       &mockWarehouseCounter{count: 2},
			mockProduct:         &mockProductCounter{err: errors.New("db error")},
			expectedStatus:      http.StatusInternalServerError,
			expectSuccess:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewSubscriptionHandler(
				tt.mockTenantRepo,
				tt.mockPlanFeatureRepo,
				tt.mockUserRepo,
				tt.mockWarehouse,
				tt.mockProduct,
			)

			router := gin.New()
			router.GET("/billing/subscription/current", func(c *gin.Context) {
				if tt.tenantID != "" {
					c.Set("jwt_tenant_id", tt.tenantID)
				}
				h.GetCurrentSubscription(c)
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/billing/subscription/current", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var resp struct {
				Success bool `json:"success"`
				Data    struct {
					PlanID      string                      `json:"plan_id"`
					PlanName    string                      `json:"plan_name"`
					Status      string                      `json:"status"`
					PeriodStart *string                     `json:"period_start"`
					PeriodEnd   *string                     `json:"period_end"`
					TrialEndsAt *string                     `json:"trial_ends_at"`
					Quotas      []SubscriptionQuotaResponse `json:"quotas"`
					Features    map[string]bool             `json:"features"`
				} `json:"data"`
			}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			if tt.expectSuccess {
				assert.True(t, resp.Success)
				assert.Equal(t, tt.expectedPlanID, resp.Data.PlanID)
				assert.Equal(t, tt.expectedStatus_, resp.Data.Status)
				assert.NotEmpty(t, resp.Data.PlanName)
				assert.Len(t, resp.Data.Quotas, 3)
				assert.NotEmpty(t, resp.Data.Features)

				// Verify quotas
				for _, q := range resp.Data.Quotas {
					assert.Contains(t, []string{"users", "warehouses", "products"}, q.Type)
					assert.NotEmpty(t, q.Unit)
					assert.NotEmpty(t, q.ResetPeriod)
				}

				// Check unlimited quotas for enterprise
				if tt.checkUnlimited {
					for _, q := range resp.Data.Quotas {
						assert.Equal(t, int64(-1), q.Limit)
						assert.Equal(t, int64(-1), q.Remaining)
					}
				}

				// Check trial end date
				if tt.checkTrialEnds {
					assert.NotNil(t, resp.Data.TrialEndsAt)
				}

				// Check period dates
				if tt.checkPeriodDates {
					assert.NotNil(t, resp.Data.PeriodStart)
					assert.NotNil(t, resp.Data.PeriodEnd)
				}
			}
		})
	}
}

func TestGetPlanDisplayName(t *testing.T) {
	tests := []struct {
		plan     identity.TenantPlan
		expected string
	}{
		{identity.TenantPlanFree, "免费版"},
		{identity.TenantPlanBasic, "基础版"},
		{identity.TenantPlanPro, "专业版"},
		{identity.TenantPlanEnterprise, "企业版"},
		{identity.TenantPlan("unknown"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.plan), func(t *testing.T) {
			result := getPlanDisplayName(tt.plan)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetSubscriptionStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   identity.TenantStatus
		expected string
	}{
		{"active", identity.TenantStatusActive, "active"},
		{"inactive", identity.TenantStatusInactive, "cancelled"},
		{"suspended", identity.TenantStatusSuspended, "expired"},
		{"trial", identity.TenantStatusTrial, "trial"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenant := createSubscriptionTestTenant(identity.TenantPlanBasic)
			tenant.Status = tt.status
			result := getSubscriptionStatus(tenant)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetQuotaLimit(t *testing.T) {
	tests := []struct {
		name        string
		limit       int64
		isUnlimited bool
		expected    int64
	}{
		{"normal limit", 100, false, 100},
		{"unlimited", 100, true, -1},
		{"zero limit", 0, false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getQuotaLimit(tt.limit, tt.isUnlimited)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateQuotaRemaining(t *testing.T) {
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
			result := calculateQuotaRemaining(tt.used, tt.limit, tt.isUnlimited)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSubscriptionHandler_GetSubscriptionFeatures(t *testing.T) {
	tenantID := uuid.New()
	tenant := createSubscriptionTestTenant(identity.TenantPlanBasic)
	tenant.ID = tenantID

	tests := []struct {
		name                 string
		mockPlanFeatureRepo  *mockPlanFeatureRepository
		expectedFeatureCount int
	}{
		{
			name: "features from repository",
			mockPlanFeatureRepo: &mockPlanFeatureRepository{
				features: []identity.PlanFeature{
					*identity.NewPlanFeature(identity.TenantPlanBasic, identity.FeatureMultiWarehouse, true, "Multi warehouse"),
					*identity.NewPlanFeature(identity.TenantPlanBasic, identity.FeatureAPIAccess, false, "API access"),
				},
			},
			expectedFeatureCount: 2,
		},
		{
			name: "fallback to defaults on error",
			mockPlanFeatureRepo: &mockPlanFeatureRepository{
				err: errors.New("db error"),
			},
			expectedFeatureCount: len(identity.DefaultPlanFeatures(identity.TenantPlanBasic)),
		},
		{
			name: "fallback to defaults on empty",
			mockPlanFeatureRepo: &mockPlanFeatureRepository{
				features: []identity.PlanFeature{},
			},
			expectedFeatureCount: len(identity.DefaultPlanFeatures(identity.TenantPlanBasic)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewSubscriptionHandler(
				&mockTenantRepository{tenant: tenant},
				tt.mockPlanFeatureRepo,
				&mockUserRepository{count: 5},
				&mockWarehouseCounter{count: 2},
				&mockProductCounter{count: 100},
			)

			router := gin.New()
			router.GET("/billing/subscription/current", func(c *gin.Context) {
				c.Set("jwt_tenant_id", tenantID.String())
				h.GetCurrentSubscription(c)
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/billing/subscription/current", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var resp struct {
				Success bool `json:"success"`
				Data    struct {
					Features map[string]bool `json:"features"`
				} `json:"data"`
			}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			assert.True(t, resp.Success)
			assert.Len(t, resp.Data.Features, tt.expectedFeatureCount)
		})
	}
}
