package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/erp/backend/internal/infrastructure/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockTenantValidator is a test implementation of TenantValidator
type mockTenantValidator struct {
	ValidTenants map[string]*TenantInfo
	ShouldFail   bool
	FailError    error
}

func (m *mockTenantValidator) ValidateTenant(tenantID string) (*TenantInfo, error) {
	if m.ShouldFail {
		return nil, m.FailError
	}
	if info, exists := m.ValidTenants[tenantID]; exists {
		return info, nil
	}
	return nil, errors.New("tenant not found")
}

func TestTenantMiddleware_HeaderExtraction(t *testing.T) {
	tests := []struct {
		name           string
		tenantID       string
		expectedStatus int
		expectedID     string
	}{
		{
			name:           "valid tenant ID in header",
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing tenant ID",
			tenantID:       "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid tenant ID format",
			tenantID:       "invalid-uuid",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(TenantMiddleware())

			var capturedTenantID string
			router.GET("/test", func(c *gin.Context) {
				capturedTenantID = GetTenantID(c)
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.tenantID != "" {
				req.Header.Set(TenantHeaderKey, tt.tenantID)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, tt.tenantID, capturedTenantID)
			}
		})
	}
}

func TestTenantMiddleware_JWTExtraction(t *testing.T) {
	tenantID := uuid.New().String()

	router := gin.New()

	// Simulate JWT middleware that sets tenant_id
	router.Use(func(c *gin.Context) {
		c.Set("jwt_tenant_id", tenantID)
		c.Next()
	})
	router.Use(TenantMiddleware())

	var capturedTenantID string
	router.GET("/test", func(c *gin.Context) {
		capturedTenantID = GetTenantID(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, tenantID, capturedTenantID)
}

func TestTenantMiddleware_JWTOverridesHeader(t *testing.T) {
	jwtTenantID := uuid.New().String()
	headerTenantID := uuid.New().String()

	router := gin.New()

	// JWT sets one tenant ID
	router.Use(func(c *gin.Context) {
		c.Set("jwt_tenant_id", jwtTenantID)
		c.Next()
	})
	router.Use(TenantMiddleware())

	var capturedTenantID string
	router.GET("/test", func(c *gin.Context) {
		capturedTenantID = GetTenantID(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// Header sets a different tenant ID
	req.Header.Set(TenantHeaderKey, headerTenantID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// JWT should take priority over header
	assert.Equal(t, jwtTenantID, capturedTenantID)
}

func TestTenantMiddleware_SkipPaths(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		skipPaths      []string
		tenantID       string
		expectedStatus int
	}{
		{
			name:           "health endpoint skipped",
			path:           "/health",
			skipPaths:      []string{"/health"},
			tenantID:       "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "api health endpoint skipped",
			path:           "/api/v1/health",
			skipPaths:      []string{"/api/v1/health"},
			tenantID:       "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "metrics endpoint skipped",
			path:           "/metrics",
			skipPaths:      []string{"/metrics"},
			tenantID:       "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "nested health path skipped",
			path:           "/health/ready",
			skipPaths:      []string{"/health"},
			tenantID:       "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "non-skipped path requires tenant",
			path:           "/api/test",
			skipPaths:      []string{"/health"},
			tenantID:       "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			cfg := DefaultTenantConfig()
			cfg.SkipPaths = tt.skipPaths
			router.Use(TenantMiddlewareWithConfig(cfg))

			router.GET(tt.path, func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.tenantID != "" {
				req.Header.Set(TenantHeaderKey, tt.tenantID)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestTenantMiddleware_OptionalTenant(t *testing.T) {
	router := gin.New()
	router.Use(OptionalTenantMiddleware())

	var capturedTenantID string
	router.GET("/test", func(c *gin.Context) {
		capturedTenantID = GetTenantID(c)
		c.Status(http.StatusOK)
	})

	// Request without tenant ID should succeed
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, capturedTenantID)
}

func TestTenantMiddleware_WithValidator(t *testing.T) {
	validTenantID := uuid.New().String()
	invalidTenantID := uuid.New().String()

	validator := &mockTenantValidator{
		ValidTenants: map[string]*TenantInfo{
			validTenantID: {
				ID:   uuid.MustParse(validTenantID),
				Code: "ACME",
			},
		},
	}

	tests := []struct {
		name           string
		tenantID       string
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "valid tenant passes validation",
			tenantID:       validTenantID,
			expectedStatus: http.StatusOK,
			expectedCode:   "ACME",
		},
		{
			name:           "invalid tenant fails validation",
			tenantID:       invalidTenantID,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			cfg := DefaultTenantConfig()
			cfg.Validator = validator
			router.Use(TenantMiddlewareWithConfig(cfg))

			var capturedCode string
			router.GET("/test", func(c *gin.Context) {
				capturedCode = GetTenantCode(c)
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set(TenantHeaderKey, tt.tenantID)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, tt.expectedCode, capturedCode)
			}
		})
	}
}

func TestTenantMiddleware_SubdomainExtraction(t *testing.T) {
	// Note: The tenant ID for subdomain extraction returns the subdomain as tenant code,
	// which then needs to be resolved to a tenant ID by the validator
	// For this test, we test the extraction logic directly

	tests := []struct {
		name       string
		host       string
		baseDomain string
		expected   string
	}{
		{
			name:       "simple subdomain",
			host:       "acme.erp.com",
			baseDomain: "erp.com",
			expected:   "acme",
		},
		{
			name:       "subdomain with port",
			host:       "acme.erp.com:8080",
			baseDomain: "erp.com",
			expected:   "acme",
		},
		{
			name:       "no subdomain",
			host:       "erp.com",
			baseDomain: "erp.com",
			expected:   "",
		},
		{
			name:       "www subdomain ignored",
			host:       "www.erp.com",
			baseDomain: "erp.com",
			expected:   "",
		},
		{
			name:       "different base domain",
			host:       "acme.other.com",
			baseDomain: "erp.com",
			expected:   "",
		},
		{
			name:       "multi-level subdomain",
			host:       "app.acme.erp.com",
			baseDomain: "erp.com",
			expected:   "app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTenantFromSubdomain(tt.host, tt.baseDomain)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateTenantIDFormat(t *testing.T) {
	tests := []struct {
		name      string
		tenantID  string
		wantError bool
	}{
		{
			name:      "valid UUID",
			tenantID:  uuid.New().String(),
			wantError: false,
		},
		{
			name:      "invalid UUID - too short",
			tenantID:  "invalid",
			wantError: true,
		},
		{
			name:      "invalid UUID - wrong format",
			tenantID:  "not-a-valid-uuid-format",
			wantError: true,
		},
		{
			name:      "empty string",
			tenantID:  "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTenantIDFormat(tt.tenantID)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetTenantID(t *testing.T) {
	tenantID := uuid.New().String()

	router := gin.New()
	router.Use(TenantMiddleware())

	router.GET("/test", func(c *gin.Context) {
		// Test GetTenantID
		gotID := GetTenantID(c)
		assert.Equal(t, tenantID, gotID)

		// Test GetTenantUUID
		gotUUID, err := GetTenantUUID(c)
		require.NoError(t, err)
		assert.Equal(t, uuid.MustParse(tenantID), gotUUID)

		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(TenantHeaderKey, tenantID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMustGetTenantID_Panics(t *testing.T) {
	router := gin.New()
	// No tenant middleware, so no tenant_id in context

	router.GET("/test", func(c *gin.Context) {
		assert.Panics(t, func() {
			MustGetTenantID(c)
		})
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMustGetTenantUUID_Panics(t *testing.T) {
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		assert.Panics(t, func() {
			MustGetTenantUUID(c)
		})
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDefaultTenantConfig(t *testing.T) {
	cfg := DefaultTenantConfig()

	assert.True(t, cfg.HeaderEnabled)
	assert.True(t, cfg.JWTEnabled)
	assert.False(t, cfg.SubdomainEnabled)
	assert.Empty(t, cfg.BaseDomain)
	assert.True(t, cfg.Required)
	assert.Nil(t, cfg.Validator)
	assert.Nil(t, cfg.Logger)
	assert.Contains(t, cfg.SkipPaths, "/health")
	assert.Contains(t, cfg.SkipPaths, "/metrics")
}

func TestTenantMiddleware_ContextPropagation(t *testing.T) {
	tenantID := uuid.New().String()

	router := gin.New()
	router.Use(TenantMiddleware())

	router.GET("/test", func(c *gin.Context) {
		// Test that tenant ID is also available in the request context
		// via the logger package utility
		ctx := c.Request.Context()
		ctxTenantID := logger.GetTenantID(ctx)
		assert.Equal(t, tenantID, ctxTenantID)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(TenantHeaderKey, tenantID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTenantMiddleware_DisabledMethods(t *testing.T) {
	tenantID := uuid.New().String()

	t.Run("header disabled", func(t *testing.T) {
		router := gin.New()
		cfg := DefaultTenantConfig()
		cfg.HeaderEnabled = false
		cfg.Required = false
		router.Use(TenantMiddlewareWithConfig(cfg))

		var capturedTenantID string
		router.GET("/test", func(c *gin.Context) {
			capturedTenantID = GetTenantID(c)
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set(TenantHeaderKey, tenantID)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// Header extraction disabled, so tenant ID should be empty
		assert.Empty(t, capturedTenantID)
	})

	t.Run("jwt disabled", func(t *testing.T) {
		router := gin.New()

		// Simulate JWT middleware
		router.Use(func(c *gin.Context) {
			c.Set("jwt_tenant_id", tenantID)
			c.Next()
		})

		cfg := DefaultTenantConfig()
		cfg.JWTEnabled = false
		cfg.Required = false
		router.Use(TenantMiddlewareWithConfig(cfg))

		var capturedTenantID string
		router.GET("/test", func(c *gin.Context) {
			capturedTenantID = GetTenantID(c)
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// JWT extraction disabled, so tenant ID should be empty
		assert.Empty(t, capturedTenantID)
	})
}

func TestTenantMiddleware_ValidatorError(t *testing.T) {
	tenantID := uuid.New().String()
	validatorError := errors.New("database connection failed")

	validator := &mockTenantValidator{
		ShouldFail: true,
		FailError:  validatorError,
	}

	router := gin.New()
	cfg := DefaultTenantConfig()
	cfg.Validator = validator
	router.Use(TenantMiddlewareWithConfig(cfg))

	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(TenantHeaderKey, tenantID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
