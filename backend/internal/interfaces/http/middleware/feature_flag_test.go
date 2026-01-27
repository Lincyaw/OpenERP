package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/featureflag"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockFeatureFlagEvaluator is a test implementation of FeatureFlagEvaluator
type mockFeatureFlagEvaluator struct {
	Results map[string]featureflag.EvaluationResult
	Called  bool
	Keys    []string
	EvalCtx *featureflag.EvaluationContext
}

func (m *mockFeatureFlagEvaluator) EvaluateBatch(ctx context.Context, flagKeys []string, evalCtx *featureflag.EvaluationContext) map[string]featureflag.EvaluationResult {
	m.Called = true
	m.Keys = flagKeys
	m.EvalCtx = evalCtx
	return m.Results
}

func TestFeatureFlagMiddleware_BasicEvaluation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockEvaluator := &mockFeatureFlagEvaluator{
		Results: map[string]featureflag.EvaluationResult{
			"enable_new_checkout": {
				Key:     "enable_new_checkout",
				Enabled: true,
				Reason:  featureflag.EvaluationReasonDefault,
			},
			"dark_mode_default": {
				Key:     "dark_mode_default",
				Enabled: false,
				Variant: "light",
				Reason:  featureflag.EvaluationReasonDefault,
			},
		},
	}

	preloadKeys := []string{"enable_new_checkout", "dark_mode_default"}

	router := gin.New()
	router.Use(FeatureFlagMiddleware(mockEvaluator, preloadKeys))

	var capturedFlags map[string]FlagValue
	router.GET("/test", func(c *gin.Context) {
		capturedFlags = GetAllFlags(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, mockEvaluator.Called)
	assert.Equal(t, preloadKeys, mockEvaluator.Keys)
	require.NotNil(t, capturedFlags)
	assert.True(t, capturedFlags["enable_new_checkout"].Enabled)
	assert.False(t, capturedFlags["dark_mode_default"].Enabled)
	assert.Equal(t, "light", capturedFlags["dark_mode_default"].Variant)
}

func TestFeatureFlagMiddleware_ExtractsJWTContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tenantID := uuid.New().String()
	userID := uuid.New().String()
	roleIDs := []string{"admin", "user"}

	mockEvaluator := &mockFeatureFlagEvaluator{
		Results: map[string]featureflag.EvaluationResult{
			"test_flag": {
				Key:     "test_flag",
				Enabled: true,
				Reason:  featureflag.EvaluationReasonDefault,
			},
		},
	}

	router := gin.New()

	// Simulate JWT middleware setting context values
	router.Use(func(c *gin.Context) {
		c.Set(JWTTenantIDKey, tenantID)
		c.Set(JWTUserIDKey, userID)
		c.Set(JWTRoleIDsKey, roleIDs)
		c.Set("request_id", "req-12345")
		c.Next()
	})

	cfg := DefaultFeatureFlagConfig()
	cfg.Evaluator = mockEvaluator
	cfg.PreloadKeys = []string{"test_flag"}
	cfg.Environment = "testing"
	router.Use(FeatureFlagMiddlewareWithConfig(cfg))

	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, mockEvaluator.Called)
	require.NotNil(t, mockEvaluator.EvalCtx)
	assert.Equal(t, tenantID, mockEvaluator.EvalCtx.TenantID)
	assert.Equal(t, userID, mockEvaluator.EvalCtx.UserID)
	assert.Equal(t, "admin", mockEvaluator.EvalCtx.UserRole) // First role
	assert.Equal(t, "req-12345", mockEvaluator.EvalCtx.RequestID)
	assert.Equal(t, "testing", mockEvaluator.EvalCtx.Environment)
}

func TestFeatureFlagMiddleware_SkipPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockEvaluator := &mockFeatureFlagEvaluator{
		Results: map[string]featureflag.EvaluationResult{},
	}

	router := gin.New()
	router.Use(FeatureFlagMiddleware(mockEvaluator, []string{"test_flag"}))

	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.GET("/api", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Test skip path
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, mockEvaluator.Called, "Evaluator should not be called for skip paths")

	// Reset
	mockEvaluator.Called = false

	// Test non-skip path
	req = httptest.NewRequest(http.MethodGet, "/api", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, mockEvaluator.Called, "Evaluator should be called for non-skip paths")
}

func TestFeatureFlagMiddleware_NoEvaluatorOrKeys(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		evaluator   FeatureFlagEvaluator
		preloadKeys []string
	}{
		{
			name:        "nil evaluator",
			evaluator:   nil,
			preloadKeys: []string{"flag1"},
		},
		{
			name: "empty preload keys",
			evaluator: &mockFeatureFlagEvaluator{
				Results: map[string]featureflag.EvaluationResult{},
			},
			preloadKeys: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(FeatureFlagMiddleware(tt.evaluator, tt.preloadKeys))

			var capturedFlags map[string]FlagValue
			router.GET("/test", func(c *gin.Context) {
				capturedFlags = GetAllFlags(c)
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Empty(t, capturedFlags)
		})
	}
}

func TestGetFeatureFlag(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		flags    map[string]FlagValue
		key      string
		expected bool
	}{
		{
			name: "flag exists and enabled",
			flags: map[string]FlagValue{
				"test_flag": {Enabled: true},
			},
			key:      "test_flag",
			expected: true,
		},
		{
			name: "flag exists and disabled",
			flags: map[string]FlagValue{
				"test_flag": {Enabled: false},
			},
			key:      "test_flag",
			expected: false,
		},
		{
			name: "flag does not exist",
			flags: map[string]FlagValue{
				"other_flag": {Enabled: true},
			},
			key:      "test_flag",
			expected: false,
		},
		{
			name:     "no flags in context",
			flags:    nil,
			key:      "test_flag",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			if tt.flags != nil {
				c.Set(FeatureFlagContextKey, tt.flags)
			}

			result := GetFeatureFlag(c, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFeatureVariant(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		flags    map[string]FlagValue
		key      string
		expected string
	}{
		{
			name: "variant exists",
			flags: map[string]FlagValue{
				"test_flag": {Enabled: true, Variant: "A"},
			},
			key:      "test_flag",
			expected: "A",
		},
		{
			name: "no variant",
			flags: map[string]FlagValue{
				"test_flag": {Enabled: true, Variant: ""},
			},
			key:      "test_flag",
			expected: "",
		},
		{
			name: "flag does not exist",
			flags: map[string]FlagValue{
				"other_flag": {Enabled: true, Variant: "B"},
			},
			key:      "test_flag",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			if tt.flags != nil {
				c.Set(FeatureFlagContextKey, tt.flags)
			}

			result := GetFeatureVariant(c, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetAllFlags(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns copy of flags", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		originalFlags := map[string]FlagValue{
			"flag1": {Enabled: true},
			"flag2": {Enabled: false, Variant: "control"},
		}
		c.Set(FeatureFlagContextKey, originalFlags)

		result := GetAllFlags(c)

		// Verify values match
		assert.Equal(t, len(originalFlags), len(result))
		assert.Equal(t, originalFlags["flag1"].Enabled, result["flag1"].Enabled)
		assert.Equal(t, originalFlags["flag2"].Variant, result["flag2"].Variant)

		// Verify it's a copy (modifying result doesn't affect original)
		result["flag1"] = FlagValue{Enabled: false}
		assert.True(t, originalFlags["flag1"].Enabled, "Original should not be modified")
	})

	t.Run("returns empty map when no flags", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		result := GetAllFlags(c)
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})
}

func TestGetFeatureFlagEvalContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns eval context when set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		evalCtx := featureflag.NewEvaluationContext().
			WithTenant("tenant-123").
			WithUser("user-456")
		c.Set(FeatureFlagEvalContextKey, evalCtx)

		result := GetFeatureFlagEvalContext(c)
		require.NotNil(t, result)
		assert.Equal(t, "tenant-123", result.TenantID)
		assert.Equal(t, "user-456", result.UserID)
	})

	t.Run("returns nil when not set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		result := GetFeatureFlagEvalContext(c)
		assert.Nil(t, result)
	})
}

func TestGetFeatureFlagEvalDuration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns duration when set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		duration := 5 * time.Millisecond
		c.Set(FeatureFlagEvalDurationKey, duration)

		result := GetFeatureFlagEvalDuration(c)
		assert.Equal(t, duration, result)
	})

	t.Run("returns zero when not set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		result := GetFeatureFlagEvalDuration(c)
		assert.Equal(t, time.Duration(0), result)
	})
}

func TestMustGetFeatureFlag(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns flag when exists", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		flags := map[string]FlagValue{
			"test_flag": {Enabled: true},
		}
		c.Set(FeatureFlagContextKey, flags)

		result := MustGetFeatureFlag(c, "test_flag")
		assert.True(t, result)
	})

	t.Run("panics when flags not in context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		assert.Panics(t, func() {
			MustGetFeatureFlag(c, "test_flag")
		})
	})
}

func TestWithFeatureFlag(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("executes handler when flag enabled", func(t *testing.T) {
		router := gin.New()

		handlerCalled := false
		router.GET("/test", func(c *gin.Context) {
			c.Set(FeatureFlagContextKey, map[string]FlagValue{
				"my_feature": {Enabled: true},
			})
			c.Next()
		}, WithFeatureFlag("my_feature", func(c *gin.Context) {
			handlerCalled = true
			c.Status(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, handlerCalled)
	})

	t.Run("returns 404 when flag disabled", func(t *testing.T) {
		router := gin.New()

		handlerCalled := false
		router.GET("/test", func(c *gin.Context) {
			c.Set(FeatureFlagContextKey, map[string]FlagValue{
				"my_feature": {Enabled: false},
			})
			c.Next()
		}, WithFeatureFlag("my_feature", func(c *gin.Context) {
			handlerCalled = true
			c.Status(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.False(t, handlerCalled)
	})
}

func TestFeatureFlagMiddleware_WithLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a nop logger for testing (no output)
	logger := zap.NewNop()

	mockEvaluator := &mockFeatureFlagEvaluator{
		Results: map[string]featureflag.EvaluationResult{
			"test_flag": {
				Key:     "test_flag",
				Enabled: true,
				Reason:  featureflag.EvaluationReasonDefault,
			},
		},
	}

	cfg := DefaultFeatureFlagConfig()
	cfg.Evaluator = mockEvaluator
	cfg.PreloadKeys = []string{"test_flag"}
	cfg.Logger = logger

	router := gin.New()
	router.Use(FeatureFlagMiddlewareWithConfig(cfg))

	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, mockEvaluator.Called)
}

func TestBuildEvaluationContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("builds context with all values", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		tenantID := uuid.New().String()
		userID := uuid.New().String()

		c.Set(JWTTenantIDKey, tenantID)
		c.Set(JWTUserIDKey, userID)
		c.Set(JWTRoleIDsKey, []string{"admin", "user"})
		c.Set("request_id", "req-12345")

		evalCtx := buildEvaluationContext(c, "production")

		assert.Equal(t, tenantID, evalCtx.TenantID)
		assert.Equal(t, userID, evalCtx.UserID)
		assert.Equal(t, "admin", evalCtx.UserRole)
		assert.Equal(t, "req-12345", evalCtx.RequestID)
		assert.Equal(t, "production", evalCtx.Environment)
		assert.False(t, evalCtx.Timestamp.IsZero())
	})

	t.Run("handles missing values gracefully", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		evalCtx := buildEvaluationContext(c, "")

		assert.Empty(t, evalCtx.TenantID)
		assert.Empty(t, evalCtx.UserID)
		assert.Empty(t, evalCtx.UserRole)
		assert.Empty(t, evalCtx.RequestID)
		assert.Empty(t, evalCtx.Environment)
	})

	t.Run("handles empty role list", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Set(JWTRoleIDsKey, []string{})

		evalCtx := buildEvaluationContext(c, "")

		assert.Empty(t, evalCtx.UserRole)
	})
}

func TestGetFeatureFlag_InvalidType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("handles wrong type in context gracefully", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		// Set wrong type
		c.Set(FeatureFlagContextKey, "invalid_string_type")

		result := GetFeatureFlag(c, "test_flag")
		assert.False(t, result, "Should return false for invalid type")
	})

	t.Run("handles int type in context gracefully", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		// Set wrong type
		c.Set(FeatureFlagContextKey, 12345)

		result := GetFeatureFlag(c, "test_flag")
		assert.False(t, result, "Should return false for int type")
	})
}

func TestGetFeatureVariant_InvalidType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("handles wrong type in context gracefully", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		// Set wrong type
		c.Set(FeatureFlagContextKey, "invalid_string_type")

		result := GetFeatureVariant(c, "test_flag")
		assert.Empty(t, result, "Should return empty string for invalid type")
	})
}

func TestGetAllFlags_InvalidType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("handles wrong type in context gracefully", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		// Set wrong type
		c.Set(FeatureFlagContextKey, []string{"invalid", "type"})

		result := GetAllFlags(c)
		assert.NotNil(t, result, "Should return empty map, not nil")
		assert.Empty(t, result, "Should return empty map for invalid type")
	})
}

func TestGetFeatureFlagEvalContext_InvalidType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("handles wrong type in context gracefully", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		// Set wrong type
		c.Set(FeatureFlagEvalContextKey, "invalid_string_type")

		result := GetFeatureFlagEvalContext(c)
		assert.Nil(t, result, "Should return nil for invalid type")
	})
}

func TestGetFeatureFlagEvalDuration_InvalidType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("handles wrong type in context gracefully", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		// Set wrong type
		c.Set(FeatureFlagEvalDurationKey, "5ms")

		result := GetFeatureFlagEvalDuration(c)
		assert.Equal(t, time.Duration(0), result, "Should return 0 for invalid type")
	})
}

func TestMustGetFeatureFlag_InvalidType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("panics when flags have wrong type", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		// Set wrong type
		c.Set(FeatureFlagContextKey, "invalid_type")

		assert.Panics(t, func() {
			MustGetFeatureFlag(c, "test_flag")
		})
	})
}
