package handler

import (
	"bytes"
	"context"
	"encoding/json"
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

// mockTenantRepository is a mock implementation of identity.TenantRepository
type mockTenantRepository struct {
	tenant *identity.Tenant
	err    error
}

func (m *mockTenantRepository) FindByID(ctx context.Context, id uuid.UUID) (*identity.Tenant, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.tenant, nil
}

func (m *mockTenantRepository) FindByCode(ctx context.Context, code string) (*identity.Tenant, error) {
	return nil, nil
}

func (m *mockTenantRepository) FindByDomain(ctx context.Context, domain string) (*identity.Tenant, error) {
	return nil, nil
}

func (m *mockTenantRepository) FindAll(ctx context.Context, filter shared.Filter) ([]identity.Tenant, error) {
	return nil, nil
}

func (m *mockTenantRepository) FindByStatus(ctx context.Context, status identity.TenantStatus, filter shared.Filter) ([]identity.Tenant, error) {
	return nil, nil
}

func (m *mockTenantRepository) FindByPlan(ctx context.Context, plan identity.TenantPlan, filter shared.Filter) ([]identity.Tenant, error) {
	return nil, nil
}

func (m *mockTenantRepository) FindActive(ctx context.Context, filter shared.Filter) ([]identity.Tenant, error) {
	return nil, nil
}

func (m *mockTenantRepository) FindTrialExpiring(ctx context.Context, withinDays int) ([]identity.Tenant, error) {
	return nil, nil
}

func (m *mockTenantRepository) FindSubscriptionExpiring(ctx context.Context, withinDays int) ([]identity.Tenant, error) {
	return nil, nil
}

func (m *mockTenantRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]identity.Tenant, error) {
	return nil, nil
}

func (m *mockTenantRepository) Save(ctx context.Context, tenant *identity.Tenant) error {
	return nil
}

func (m *mockTenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockTenantRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	return 0, nil
}

func (m *mockTenantRepository) CountByStatus(ctx context.Context, status identity.TenantStatus) (int64, error) {
	return 0, nil
}

func (m *mockTenantRepository) CountByPlan(ctx context.Context, plan identity.TenantPlan) (int64, error) {
	return 0, nil
}

func (m *mockTenantRepository) ExistsByCode(ctx context.Context, code string) (bool, error) {
	return false, nil
}

func (m *mockTenantRepository) ExistsByDomain(ctx context.Context, domain string) (bool, error) {
	return false, nil
}

func (m *mockTenantRepository) FindByStripeCustomerID(ctx context.Context, customerID string) (*identity.Tenant, error) {
	return nil, nil
}

func (m *mockTenantRepository) FindByStripeSubscriptionID(ctx context.Context, subscriptionID string) (*identity.Tenant, error) {
	return nil, nil
}

func TestPlanFeatureHandler_ListPlans(t *testing.T) {
	h := NewPlanFeatureHandler(&mockTenantRepository{}, nil)

	router := gin.New()
	router.GET("/admin/plans", h.ListPlans)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/plans", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Plans []PlanResponse `json:"plans"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Len(t, resp.Data.Plans, 4)

	// Verify plan codes
	planCodes := make([]string, len(resp.Data.Plans))
	for i, p := range resp.Data.Plans {
		planCodes[i] = p.Code
	}
	assert.Contains(t, planCodes, "free")
	assert.Contains(t, planCodes, "basic")
	assert.Contains(t, planCodes, "pro")
	assert.Contains(t, planCodes, "enterprise")
}

func TestPlanFeatureHandler_GetPlanFeatures(t *testing.T) {
	h := NewPlanFeatureHandler(&mockTenantRepository{}, nil)

	tests := []struct {
		name           string
		planCode       string
		expectedStatus int
		expectSuccess  bool
	}{
		{
			name:           "valid free plan",
			planCode:       "free",
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name:           "valid basic plan",
			planCode:       "basic",
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name:           "valid pro plan",
			planCode:       "pro",
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name:           "valid enterprise plan",
			planCode:       "enterprise",
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name:           "invalid plan code",
			planCode:       "invalid",
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/admin/plans/:plan/features", h.GetPlanFeatures)

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/admin/plans/"+tt.planCode+"/features", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var resp struct {
				Success bool `json:"success"`
				Data    struct {
					Plan     string                `json:"plan"`
					Features []PlanFeatureResponse `json:"features"`
				} `json:"data"`
			}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			if tt.expectSuccess {
				assert.True(t, resp.Success)
				assert.Equal(t, tt.planCode, resp.Data.Plan)
				assert.NotEmpty(t, resp.Data.Features)
			}
		})
	}
}

func TestPlanFeatureHandler_GetPlanFeatures_EmptyPlan(t *testing.T) {
	h := NewPlanFeatureHandler(&mockTenantRepository{}, nil)

	router := gin.New()
	router.GET("/admin/plans/:plan/features", h.GetPlanFeatures)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/plans//features", nil)
	router.ServeHTTP(w, req)

	// Empty plan should result in 404 (route not matched) or redirect
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestPlanFeatureHandler_UpdatePlanFeatures(t *testing.T) {
	h := NewPlanFeatureHandler(&mockTenantRepository{}, nil)

	tests := []struct {
		name           string
		planCode       string
		requestBody    interface{}
		expectedStatus int
		expectSuccess  bool
	}{
		{
			name:     "valid update",
			planCode: "basic",
			requestBody: UpdatePlanFeaturesRequest{
				Features: []UpdatePlanFeatureRequest{
					{
						FeatureKey: "multi_warehouse",
						Enabled:    true,
						Limit:      nil,
					},
				},
			},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name:     "update with limit",
			planCode: "basic",
			requestBody: UpdatePlanFeaturesRequest{
				Features: []UpdatePlanFeatureRequest{
					{
						FeatureKey: "data_import",
						Enabled:    true,
						Limit:      intPtr(5000),
					},
				},
			},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name:     "invalid plan code",
			planCode: "invalid",
			requestBody: UpdatePlanFeaturesRequest{
				Features: []UpdatePlanFeatureRequest{
					{
						FeatureKey: "multi_warehouse",
						Enabled:    true,
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
		{
			name:     "invalid feature key",
			planCode: "basic",
			requestBody: UpdatePlanFeaturesRequest{
				Features: []UpdatePlanFeatureRequest{
					{
						FeatureKey: "invalid_feature",
						Enabled:    true,
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
		{
			name:     "negative limit",
			planCode: "basic",
			requestBody: UpdatePlanFeaturesRequest{
				Features: []UpdatePlanFeatureRequest{
					{
						FeatureKey: "data_import",
						Enabled:    true,
						Limit:      intPtr(-1),
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
		{
			name:           "empty features",
			planCode:       "basic",
			requestBody:    UpdatePlanFeaturesRequest{Features: []UpdatePlanFeatureRequest{}},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.PUT("/admin/plans/:plan/features", h.UpdatePlanFeatures)

			body, _ := json.Marshal(tt.requestBody)
			w := httptest.NewRecorder()
			req := httptest.NewRequest("PUT", "/admin/plans/"+tt.planCode+"/features", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var resp struct {
				Success bool `json:"success"`
			}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, tt.expectSuccess, resp.Success)
		})
	}
}

func TestPlanFeatureHandler_UpdatePlanFeatures_InvalidJSON(t *testing.T) {
	h := NewPlanFeatureHandler(&mockTenantRepository{}, nil)

	router := gin.New()
	router.PUT("/admin/plans/:plan/features", h.UpdatePlanFeatures)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/admin/plans/basic/features", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPlanFeatureHandler_GetCurrentTenantFeatures(t *testing.T) {
	tenantID := uuid.New()
	tenant, _ := identity.NewTenant("TEST", "Test Tenant")
	_ = tenant.SetPlan(identity.TenantPlanPro)
	tenant.ID = tenantID

	tests := []struct {
		name           string
		tenantID       string
		mockRepo       *mockTenantRepository
		expectedStatus int
		expectSuccess  bool
	}{
		{
			name:     "valid tenant",
			tenantID: tenantID.String(),
			mockRepo: &mockTenantRepository{
				tenant: tenant,
			},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name:     "tenant not found",
			tenantID: uuid.New().String(),
			mockRepo: &mockTenantRepository{
				err: shared.ErrNotFound,
			},
			expectedStatus: http.StatusNotFound,
			expectSuccess:  false,
		},
		{
			name:           "missing tenant ID",
			tenantID:       "",
			mockRepo:       &mockTenantRepository{},
			expectedStatus: http.StatusUnauthorized,
			expectSuccess:  false,
		},
		{
			name:           "invalid tenant ID format",
			tenantID:       "invalid-uuid",
			mockRepo:       &mockTenantRepository{},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewPlanFeatureHandler(tt.mockRepo, nil)

			router := gin.New()
			router.GET("/tenants/current/features", func(c *gin.Context) {
				if tt.tenantID != "" {
					c.Set("jwt_tenant_id", tt.tenantID)
				}
				h.GetCurrentTenantFeatures(c)
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/tenants/current/features", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var resp struct {
				Success bool `json:"success"`
				Data    struct {
					TenantID string                  `json:"tenant_id"`
					Plan     string                  `json:"plan"`
					Features []TenantFeatureResponse `json:"features"`
				} `json:"data"`
			}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			if tt.expectSuccess {
				assert.True(t, resp.Success)
				assert.Equal(t, tenantID.String(), resp.Data.TenantID)
				assert.Equal(t, "pro", resp.Data.Plan)
				assert.NotEmpty(t, resp.Data.Features)
			}
		})
	}
}

func TestPlanFeatureHandler_GetCurrentTenantFeatures_InternalError(t *testing.T) {
	tenantID := uuid.New()
	h := NewPlanFeatureHandler(&mockTenantRepository{
		err: assert.AnError,
	}, nil)

	router := gin.New()
	router.GET("/tenants/current/features", func(c *gin.Context) {
		c.Set("jwt_tenant_id", tenantID.String())
		h.GetCurrentTenantFeatures(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/tenants/current/features", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestIsValidPlan(t *testing.T) {
	tests := []struct {
		plan     identity.TenantPlan
		expected bool
	}{
		{identity.TenantPlanFree, true},
		{identity.TenantPlanBasic, true},
		{identity.TenantPlanPro, true},
		{identity.TenantPlanEnterprise, true},
		{identity.TenantPlan("invalid"), false},
		{identity.TenantPlan(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.plan), func(t *testing.T) {
			result := isValidPlan(tt.plan)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
