package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockFeatureChecker is a test implementation of FeatureChecker
type mockFeatureChecker struct {
	Features map[identity.FeatureKey]bool
	Limits   map[identity.FeatureKey]*int
	Err      error
	Called   bool
}

func (m *mockFeatureChecker) HasFeature(ctx context.Context, planID identity.TenantPlan, featureKey identity.FeatureKey) (bool, error) {
	m.Called = true
	if m.Err != nil {
		return false, m.Err
	}
	return m.Features[featureKey], nil
}

func (m *mockFeatureChecker) GetFeatureLimit(ctx context.Context, planID identity.TenantPlan, featureKey identity.FeatureKey) (*int, error) {
	m.Called = true
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Limits[featureKey], nil
}

// mockTenantPlanProvider is a test implementation of TenantPlanProvider
type mockTenantPlanProvider struct {
	Plan   identity.TenantPlan
	Err    error
	Called bool
}

func (m *mockTenantPlanProvider) GetTenantPlan(ctx context.Context, tenantID string) (identity.TenantPlan, error) {
	m.Called = true
	if m.Err != nil {
		return "", m.Err
	}
	return m.Plan, nil
}

// mockFeatureCache is a test implementation of FeatureCache
type mockFeatureCache struct {
	Data         map[string]bool
	GetCalled    bool
	SetCalled    bool
	DeleteCalled bool
	GetErr       error
	SetErr       error
}

func newMockFeatureCache() *mockFeatureCache {
	return &mockFeatureCache{
		Data: make(map[string]bool),
	}
}

func (m *mockFeatureCache) Get(ctx context.Context, key string) (bool, bool, error) {
	m.GetCalled = true
	if m.GetErr != nil {
		return false, false, m.GetErr
	}
	value, found := m.Data[key]
	return value, found, nil
}

func (m *mockFeatureCache) Set(ctx context.Context, key string, value bool, ttl time.Duration) error {
	m.SetCalled = true
	if m.SetErr != nil {
		return m.SetErr
	}
	m.Data[key] = value
	return nil
}

func (m *mockFeatureCache) Delete(ctx context.Context, key string) error {
	m.DeleteCalled = true
	delete(m.Data, key)
	return nil
}

// setupTestRouter creates a test router with JWT context middleware
func setupTestRouter(tenantID, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Simulate JWT middleware setting context values
	router.Use(func(c *gin.Context) {
		if tenantID != "" {
			c.Set(JWTTenantIDKey, tenantID)
		}
		if userID != "" {
			c.Set(JWTUserIDKey, userID)
		}
		c.Next()
	})

	return router
}

func TestRequireFeature_FeatureEnabled(t *testing.T) {
	tenantID := uuid.New().String()

	mockChecker := &mockFeatureChecker{
		Features: map[identity.FeatureKey]bool{
			identity.FeatureMultiWarehouse: true,
		},
	}

	mockProvider := &mockTenantPlanProvider{
		Plan: identity.TenantPlanPro,
	}

	cfg := FeatureMiddlewareConfig{
		FeatureChecker:     mockChecker,
		TenantPlanProvider: mockProvider,
	}

	router := setupTestRouter(tenantID, "user-123")

	handlerCalled := false
	router.GET("/test", RequireFeatureWithConfig(identity.FeatureMultiWarehouse, cfg), func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, handlerCalled)
	assert.True(t, mockChecker.Called)
	assert.True(t, mockProvider.Called)
}

func TestRequireFeature_FeatureDisabled(t *testing.T) {
	tenantID := uuid.New().String()

	mockChecker := &mockFeatureChecker{
		Features: map[identity.FeatureKey]bool{
			identity.FeatureMultiWarehouse: false,
		},
	}

	mockProvider := &mockTenantPlanProvider{
		Plan: identity.TenantPlanFree,
	}

	cfg := FeatureMiddlewareConfig{
		FeatureChecker:     mockChecker,
		TenantPlanProvider: mockProvider,
	}

	router := setupTestRouter(tenantID, "user-123")

	handlerCalled := false
	router.GET("/test", RequireFeatureWithConfig(identity.FeatureMultiWarehouse, cfg), func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, handlerCalled)
	assert.Contains(t, w.Body.String(), "ERR_FEATURE_NOT_AVAILABLE")
	assert.Contains(t, w.Body.String(), "upgrade")
}

func TestRequireFeature_NoTenantContext(t *testing.T) {
	mockChecker := &mockFeatureChecker{
		Features: map[identity.FeatureKey]bool{
			identity.FeatureMultiWarehouse: true,
		},
	}

	cfg := FeatureMiddlewareConfig{
		FeatureChecker: mockChecker,
	}

	// Router without tenant ID
	router := setupTestRouter("", "user-123")

	handlerCalled := false
	router.GET("/test", RequireFeatureWithConfig(identity.FeatureMultiWarehouse, cfg), func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, handlerCalled)
	assert.Contains(t, w.Body.String(), "No tenant context found")
}

func TestRequireFeature_TenantPlanProviderError(t *testing.T) {
	tenantID := uuid.New().String()

	mockProvider := &mockTenantPlanProvider{
		Err: errors.New("database error"),
	}

	cfg := FeatureMiddlewareConfig{
		TenantPlanProvider: mockProvider,
	}

	router := setupTestRouter(tenantID, "user-123")

	handlerCalled := false
	router.GET("/test", RequireFeatureWithConfig(identity.FeatureMultiWarehouse, cfg), func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, handlerCalled)
	assert.Contains(t, w.Body.String(), "Failed to determine subscription plan")
}

func TestRequireFeature_FeatureCheckerError(t *testing.T) {
	tenantID := uuid.New().String()

	mockChecker := &mockFeatureChecker{
		Err: errors.New("feature check error"),
	}

	mockProvider := &mockTenantPlanProvider{
		Plan: identity.TenantPlanPro,
	}

	cfg := FeatureMiddlewareConfig{
		FeatureChecker:     mockChecker,
		TenantPlanProvider: mockProvider,
	}

	router := setupTestRouter(tenantID, "user-123")

	handlerCalled := false
	router.GET("/test", RequireFeatureWithConfig(identity.FeatureMultiWarehouse, cfg), func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, handlerCalled)
}

func TestRequireFeature_UsesDefaultPlanFeatures(t *testing.T) {
	tenantID := uuid.New().String()

	// No FeatureChecker configured - should use default plan features
	mockProvider := &mockTenantPlanProvider{
		Plan: identity.TenantPlanPro,
	}

	cfg := FeatureMiddlewareConfig{
		TenantPlanProvider: mockProvider,
	}

	router := setupTestRouter(tenantID, "user-123")

	handlerCalled := false
	// Multi-warehouse is enabled for Pro plan by default
	router.GET("/test", RequireFeatureWithConfig(identity.FeatureMultiWarehouse, cfg), func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, handlerCalled)
}

func TestRequireFeature_WithCache(t *testing.T) {
	tenantID := uuid.New().String()

	mockChecker := &mockFeatureChecker{
		Features: map[identity.FeatureKey]bool{
			identity.FeatureMultiWarehouse: true,
		},
	}

	mockProvider := &mockTenantPlanProvider{
		Plan: identity.TenantPlanPro,
	}

	mockCache := newMockFeatureCache()

	cfg := FeatureMiddlewareConfig{
		FeatureChecker:     mockChecker,
		TenantPlanProvider: mockProvider,
		Cache:              mockCache,
		CacheTTL:           5 * time.Minute,
	}

	router := setupTestRouter(tenantID, "user-123")

	router.GET("/test", RequireFeatureWithConfig(identity.FeatureMultiWarehouse, cfg), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// First request - should check feature and cache result
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, mockCache.GetCalled)
	assert.True(t, mockCache.SetCalled)
	assert.True(t, mockChecker.Called)

	// Reset mock state
	mockChecker.Called = false
	mockCache.GetCalled = false
	mockCache.SetCalled = false

	// Second request - should use cached result
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, mockCache.GetCalled)
	assert.False(t, mockChecker.Called) // Should not call checker - used cache
}

func TestRequireAnyFeature_OneFeatureEnabled(t *testing.T) {
	tenantID := uuid.New().String()

	mockChecker := &mockFeatureChecker{
		Features: map[identity.FeatureKey]bool{
			identity.FeatureMultiWarehouse:  false,
			identity.FeatureBatchManagement: true,
			identity.FeatureSerialTracking:  false,
		},
	}

	mockProvider := &mockTenantPlanProvider{
		Plan: identity.TenantPlanBasic,
	}

	cfg := FeatureMiddlewareConfig{
		FeatureChecker:     mockChecker,
		TenantPlanProvider: mockProvider,
	}

	router := setupTestRouter(tenantID, "user-123")

	handlerCalled := false
	router.GET("/test", RequireAnyFeatureWithConfig(cfg,
		identity.FeatureMultiWarehouse,
		identity.FeatureBatchManagement,
		identity.FeatureSerialTracking,
	), func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, handlerCalled)
}

func TestRequireAnyFeature_NoFeaturesEnabled(t *testing.T) {
	tenantID := uuid.New().String()

	mockChecker := &mockFeatureChecker{
		Features: map[identity.FeatureKey]bool{
			identity.FeatureMultiWarehouse:  false,
			identity.FeatureBatchManagement: false,
			identity.FeatureSerialTracking:  false,
		},
	}

	mockProvider := &mockTenantPlanProvider{
		Plan: identity.TenantPlanFree,
	}

	cfg := FeatureMiddlewareConfig{
		FeatureChecker:     mockChecker,
		TenantPlanProvider: mockProvider,
	}

	router := setupTestRouter(tenantID, "user-123")

	handlerCalled := false
	router.GET("/test", RequireAnyFeatureWithConfig(cfg,
		identity.FeatureMultiWarehouse,
		identity.FeatureBatchManagement,
		identity.FeatureSerialTracking,
	), func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, handlerCalled)
	assert.Contains(t, w.Body.String(), "ERR_FEATURES_NOT_AVAILABLE")
}

func TestRequireAnyFeature_EmptyFeatureList(t *testing.T) {
	tenantID := uuid.New().String()

	cfg := FeatureMiddlewareConfig{}

	router := setupTestRouter(tenantID, "user-123")

	handlerCalled := false
	router.GET("/test", RequireAnyFeatureWithConfig(cfg), func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, handlerCalled)
}

func TestRequireAllFeatures_AllFeaturesEnabled(t *testing.T) {
	tenantID := uuid.New().String()

	mockChecker := &mockFeatureChecker{
		Features: map[identity.FeatureKey]bool{
			identity.FeatureMultiWarehouse:  true,
			identity.FeatureBatchManagement: true,
			identity.FeatureSerialTracking:  true,
		},
	}

	mockProvider := &mockTenantPlanProvider{
		Plan: identity.TenantPlanEnterprise,
	}

	cfg := FeatureMiddlewareConfig{
		FeatureChecker:     mockChecker,
		TenantPlanProvider: mockProvider,
	}

	router := setupTestRouter(tenantID, "user-123")

	handlerCalled := false
	router.GET("/test", RequireAllFeaturesWithConfig(cfg,
		identity.FeatureMultiWarehouse,
		identity.FeatureBatchManagement,
		identity.FeatureSerialTracking,
	), func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, handlerCalled)
}

func TestRequireAllFeatures_SomeFeaturesDisabled(t *testing.T) {
	tenantID := uuid.New().String()

	mockChecker := &mockFeatureChecker{
		Features: map[identity.FeatureKey]bool{
			identity.FeatureMultiWarehouse:  true,
			identity.FeatureBatchManagement: false, // This one is disabled
			identity.FeatureSerialTracking:  true,
		},
	}

	mockProvider := &mockTenantPlanProvider{
		Plan: identity.TenantPlanBasic,
	}

	cfg := FeatureMiddlewareConfig{
		FeatureChecker:     mockChecker,
		TenantPlanProvider: mockProvider,
	}

	router := setupTestRouter(tenantID, "user-123")

	handlerCalled := false
	router.GET("/test", RequireAllFeaturesWithConfig(cfg,
		identity.FeatureMultiWarehouse,
		identity.FeatureBatchManagement,
		identity.FeatureSerialTracking,
	), func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, handlerCalled)
	// Only one feature is missing, so it uses the singular error code
	assert.Contains(t, w.Body.String(), "ERR_FEATURE_NOT_AVAILABLE")
}

func TestRequireAllFeatures_EmptyFeatureList(t *testing.T) {
	tenantID := uuid.New().String()

	cfg := FeatureMiddlewareConfig{}

	router := setupTestRouter(tenantID, "user-123")

	handlerCalled := false
	router.GET("/test", RequireAllFeaturesWithConfig(cfg), func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, handlerCalled)
}

func TestRequireFeature_CustomOnDenied(t *testing.T) {
	tenantID := uuid.New().String()

	mockChecker := &mockFeatureChecker{
		Features: map[identity.FeatureKey]bool{
			identity.FeatureMultiWarehouse: false,
		},
	}

	mockProvider := &mockTenantPlanProvider{
		Plan: identity.TenantPlanFree,
	}

	onDeniedCalled := false
	var capturedFeatureKey identity.FeatureKey
	var capturedPlan identity.TenantPlan

	cfg := FeatureMiddlewareConfig{
		FeatureChecker:     mockChecker,
		TenantPlanProvider: mockProvider,
		OnDenied: func(c *gin.Context, featureKey identity.FeatureKey, tenantPlan identity.TenantPlan) {
			onDeniedCalled = true
			capturedFeatureKey = featureKey
			capturedPlan = tenantPlan
			c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
				"message": "Custom upgrade message",
			})
		},
	}

	router := setupTestRouter(tenantID, "user-123")

	router.GET("/test", RequireFeatureWithConfig(identity.FeatureMultiWarehouse, cfg), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusPaymentRequired, w.Code)
	assert.True(t, onDeniedCalled)
	assert.Equal(t, identity.FeatureMultiWarehouse, capturedFeatureKey)
	assert.Equal(t, identity.TenantPlanFree, capturedPlan)
	assert.Contains(t, w.Body.String(), "Custom upgrade message")
}

func TestRequireFeature_WithLogger(t *testing.T) {
	tenantID := uuid.New().String()

	mockChecker := &mockFeatureChecker{
		Features: map[identity.FeatureKey]bool{
			identity.FeatureMultiWarehouse: true,
		},
	}

	mockProvider := &mockTenantPlanProvider{
		Plan: identity.TenantPlanPro,
	}

	logger := zap.NewNop()

	cfg := FeatureMiddlewareConfig{
		FeatureChecker:     mockChecker,
		TenantPlanProvider: mockProvider,
		Logger:             logger,
	}

	router := setupTestRouter(tenantID, "user-123")

	router.GET("/test", RequireFeatureWithConfig(identity.FeatureMultiWarehouse, cfg), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetTenantPlan(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns plan when set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Set(TenantPlanKey, identity.TenantPlanPro)

		result := GetTenantPlan(c)
		assert.Equal(t, identity.TenantPlanPro, result)
	})

	t.Run("returns empty when not set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		result := GetTenantPlan(c)
		assert.Equal(t, identity.TenantPlan(""), result)
	})

	t.Run("returns empty for wrong type", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Set(TenantPlanKey, "invalid_type")

		result := GetTenantPlan(c)
		assert.Equal(t, identity.TenantPlan(""), result)
	})
}

func TestHasFeature(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns true when feature is enabled", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

		c.Set(TenantPlanKey, identity.TenantPlanPro)

		mockChecker := &mockFeatureChecker{
			Features: map[identity.FeatureKey]bool{
				identity.FeatureMultiWarehouse: true,
			},
		}

		cfg := FeatureMiddlewareConfig{
			FeatureChecker: mockChecker,
		}

		result := HasFeature(c, cfg, identity.FeatureMultiWarehouse)
		assert.True(t, result)
	})

	t.Run("returns false when no plan in context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

		cfg := FeatureMiddlewareConfig{}

		result := HasFeature(c, cfg, identity.FeatureMultiWarehouse)
		assert.False(t, result)
	})
}

func TestFormatFeatureName(t *testing.T) {
	tests := []struct {
		input    identity.FeatureKey
		expected string
	}{
		{identity.FeatureMultiWarehouse, "Multi Warehouse"},
		{identity.FeatureBatchManagement, "Batch Management"},
		{identity.FeatureSerialTracking, "Serial Tracking"},
		{identity.FeatureAPIAccess, "Api Access"},
		{identity.FeatureSLA, "Sla"},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := formatFeatureName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildFeatureCacheKey(t *testing.T) {
	key := buildFeatureCacheKey(identity.TenantPlanPro, identity.FeatureMultiWarehouse)
	assert.Equal(t, "feature_check:pro:multi_warehouse", key)
}

func TestWithFeature(t *testing.T) {
	tenantID := uuid.New().String()

	t.Run("executes handler when feature enabled", func(t *testing.T) {
		mockChecker := &mockFeatureChecker{
			Features: map[identity.FeatureKey]bool{
				identity.FeatureMultiWarehouse: true,
			},
		}

		mockProvider := &mockTenantPlanProvider{
			Plan: identity.TenantPlanPro,
		}

		cfg := FeatureMiddlewareConfig{
			FeatureChecker:     mockChecker,
			TenantPlanProvider: mockProvider,
		}

		router := setupTestRouter(tenantID, "user-123")

		handlerCalled := false
		router.GET("/test",
			RequireFeatureWithConfig(identity.FeatureMultiWarehouse, cfg),
			WithFeature(identity.FeatureMultiWarehouse, cfg, func(c *gin.Context) {
				handlerCalled = true
				c.Status(http.StatusOK)
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, handlerCalled)
	})
}

func TestRequireFeature_DefaultsToFreePlan(t *testing.T) {
	tenantID := uuid.New().String()

	// No TenantPlanProvider configured - should default to free plan
	mockChecker := &mockFeatureChecker{
		Features: map[identity.FeatureKey]bool{
			identity.FeatureSalesOrders: true, // Sales orders are enabled for free plan
		},
	}

	cfg := FeatureMiddlewareConfig{
		FeatureChecker: mockChecker,
	}

	router := setupTestRouter(tenantID, "user-123")

	handlerCalled := false
	router.GET("/test", RequireFeatureWithConfig(identity.FeatureSalesOrders, cfg), func(c *gin.Context) {
		handlerCalled = true
		// Verify the plan was set to free
		plan := GetTenantPlan(c)
		require.Equal(t, identity.TenantPlanFree, plan)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, handlerCalled)
}

func TestRequireFeature_TenantPlanAlreadyInContext(t *testing.T) {
	tenantID := uuid.New().String()

	mockChecker := &mockFeatureChecker{
		Features: map[identity.FeatureKey]bool{
			identity.FeatureMultiWarehouse: true,
		},
	}

	mockProvider := &mockTenantPlanProvider{
		Plan: identity.TenantPlanBasic, // This should NOT be used
	}

	cfg := FeatureMiddlewareConfig{
		FeatureChecker:     mockChecker,
		TenantPlanProvider: mockProvider,
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Simulate previous middleware setting tenant plan
	router.Use(func(c *gin.Context) {
		c.Set(JWTTenantIDKey, tenantID)
		c.Set(TenantPlanKey, identity.TenantPlanEnterprise) // Pre-set plan
		c.Next()
	})

	handlerCalled := false
	router.GET("/test", RequireFeatureWithConfig(identity.FeatureMultiWarehouse, cfg), func(c *gin.Context) {
		handlerCalled = true
		// Verify the pre-set plan was used
		plan := GetTenantPlan(c)
		require.Equal(t, identity.TenantPlanEnterprise, plan)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, handlerCalled)
	// Provider should not be called since plan was already in context
	assert.False(t, mockProvider.Called)
}

func TestRequireFeature_InvalidFeatureKeyPanics(t *testing.T) {
	assert.Panics(t, func() {
		RequireFeature(identity.FeatureKey("invalid_feature_key"))
	})
}

func TestRequireAnyFeature_InvalidFeatureKeyPanics(t *testing.T) {
	assert.Panics(t, func() {
		RequireAnyFeature(
			identity.FeatureMultiWarehouse,
			identity.FeatureKey("invalid_feature_key"),
		)
	})
}

func TestRequireAllFeatures_InvalidFeatureKeyPanics(t *testing.T) {
	assert.Panics(t, func() {
		RequireAllFeatures(
			identity.FeatureMultiWarehouse,
			identity.FeatureKey("invalid_feature_key"),
		)
	})
}

func TestRequireAnyFeature_AllChecksFailWithErrors(t *testing.T) {
	tenantID := uuid.New().String()

	mockChecker := &mockFeatureChecker{
		Err: errors.New("database error"),
	}

	mockProvider := &mockTenantPlanProvider{
		Plan: identity.TenantPlanPro,
	}

	cfg := FeatureMiddlewareConfig{
		FeatureChecker:     mockChecker,
		TenantPlanProvider: mockProvider,
	}

	router := setupTestRouter(tenantID, "user-123")

	handlerCalled := false
	router.GET("/test", RequireAnyFeatureWithConfig(cfg,
		identity.FeatureMultiWarehouse,
		identity.FeatureBatchManagement,
	), func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should fail securely when all checks fail with errors
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, handlerCalled)
	assert.Contains(t, w.Body.String(), "Failed to verify feature access")
}

func TestRequireAllFeatures_CheckErrorFailsSecurely(t *testing.T) {
	tenantID := uuid.New().String()

	mockChecker := &mockFeatureChecker{
		Err: errors.New("database error"),
	}

	mockProvider := &mockTenantPlanProvider{
		Plan: identity.TenantPlanPro,
	}

	cfg := FeatureMiddlewareConfig{
		FeatureChecker:     mockChecker,
		TenantPlanProvider: mockProvider,
	}

	router := setupTestRouter(tenantID, "user-123")

	handlerCalled := false
	router.GET("/test", RequireAllFeaturesWithConfig(cfg,
		identity.FeatureMultiWarehouse,
		identity.FeatureBatchManagement,
	), func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should fail securely when feature check errors
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, handlerCalled)
	assert.Contains(t, w.Body.String(), "Failed to verify feature access")
}

func TestRequireFeature_CacheGetError(t *testing.T) {
	tenantID := uuid.New().String()

	mockChecker := &mockFeatureChecker{
		Features: map[identity.FeatureKey]bool{
			identity.FeatureMultiWarehouse: true,
		},
	}

	mockProvider := &mockTenantPlanProvider{
		Plan: identity.TenantPlanPro,
	}

	mockCache := newMockFeatureCache()
	mockCache.GetErr = errors.New("cache connection failed")

	cfg := FeatureMiddlewareConfig{
		FeatureChecker:     mockChecker,
		TenantPlanProvider: mockProvider,
		Cache:              mockCache,
	}

	router := setupTestRouter(tenantID, "user-123")

	handlerCalled := false
	router.GET("/test", RequireFeatureWithConfig(identity.FeatureMultiWarehouse, cfg), func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should still work when cache fails - falls back to checker
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, handlerCalled)
	assert.True(t, mockChecker.Called)
}
