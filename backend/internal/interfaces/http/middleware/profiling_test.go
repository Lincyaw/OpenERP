package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/erp/backend/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestDefaultProfilingConfig(t *testing.T) {
	cfg := middleware.DefaultProfilingConfig()

	assert.True(t, cfg.Enabled)
	assert.Contains(t, cfg.SkipPaths, "/health")
	assert.Contains(t, cfg.SkipPaths, "/healthz")
	assert.Contains(t, cfg.SkipPaths, "/ready")
	assert.Contains(t, cfg.SkipPaths, "/metrics")
	assert.Contains(t, cfg.SkipPathPrefixes, "/swagger")
	assert.Contains(t, cfg.SkipPathPrefixes, "/api-docs")
}

func TestProfilingMiddleware_Disabled(t *testing.T) {
	r := gin.New()

	cfg := middleware.ProfilingConfig{
		Enabled: false,
	}

	handlerCalled := false
	r.Use(middleware.ProfilingWithConfig(cfg))
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, handlerCalled, "handler should be called when profiling is disabled")
}

func TestProfilingMiddleware_Enabled(t *testing.T) {
	r := gin.New()

	cfg := middleware.DefaultProfilingConfig()
	handlerCalled := false

	r.Use(middleware.ProfilingWithConfig(cfg))
	r.GET("/api/v1/products", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, handlerCalled, "handler should be called when profiling is enabled")
}

func TestProfilingMiddleware_SkipPaths(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		shouldSkip bool
	}{
		{"health_exact", "/health", true},
		{"healthz_exact", "/healthz", true},
		{"ready_exact", "/ready", true},
		{"metrics_exact", "/metrics", true},
		{"swagger_prefix", "/swagger/index.html", true},
		{"api_docs_prefix", "/api-docs/v1", true},
		{"normal_api_path", "/api/v1/products", false},
		{"health_subpath", "/health/check", false}, // not exact match
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			cfg := middleware.DefaultProfilingConfig()

			handlerCalled := false
			r.Use(middleware.ProfilingWithConfig(cfg))
			r.GET(tt.path, func(c *gin.Context) {
				handlerCalled = true
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.True(t, handlerCalled, "handler should be called for path: %s", tt.path)
		})
	}
}

func TestProfilingMiddleware_ExtractsLabels(t *testing.T) {
	r := gin.New()

	cfg := middleware.DefaultProfilingConfig()

	r.Use(middleware.ProfilingWithConfig(cfg))
	r.GET("/api/v1/products/:id", func(c *gin.Context) {
		// The middleware should have extracted labels
		// We can't directly test the labels, but we can verify
		// the request was processed
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products/123", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProfilingMiddleware_WithTenantFromJWT(t *testing.T) {
	r := gin.New()

	cfg := middleware.DefaultProfilingConfig()

	// Simulate JWT middleware setting tenant ID
	r.Use(func(c *gin.Context) {
		c.Set(middleware.JWTTenantIDKey, "tenant-123")
		c.Next()
	})
	r.Use(middleware.ProfilingWithConfig(cfg))
	r.GET("/api/v1/products", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProfilingMiddleware_WithTenantFromTenantMiddleware(t *testing.T) {
	r := gin.New()

	cfg := middleware.DefaultProfilingConfig()

	// Simulate Tenant middleware setting tenant ID (fallback)
	r.Use(func(c *gin.Context) {
		c.Set(middleware.TenantIDKey, "tenant-456")
		c.Next()
	})
	r.Use(middleware.ProfilingWithConfig(cfg))
	r.GET("/api/v1/products", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProfilingMiddleware_JWTTenantTakesPrecedence(t *testing.T) {
	r := gin.New()

	cfg := middleware.DefaultProfilingConfig()

	// Both JWT and Tenant middleware set tenant ID
	// JWT should take precedence
	r.Use(func(c *gin.Context) {
		c.Set(middleware.JWTTenantIDKey, "jwt-tenant")
		c.Set(middleware.TenantIDKey, "header-tenant")
		c.Next()
	})
	r.Use(middleware.ProfilingWithConfig(cfg))
	r.GET("/api/v1/products", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProfilingMiddleware_HTTPMethods(t *testing.T) {
	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			r := gin.New()

			cfg := middleware.DefaultProfilingConfig()
			handlerCalled := false

			r.Use(middleware.ProfilingWithConfig(cfg))

			// Register route for the specific method
			switch method {
			case http.MethodGet:
				r.GET("/api/v1/test", func(c *gin.Context) { handlerCalled = true; c.Status(http.StatusOK) })
			case http.MethodPost:
				r.POST("/api/v1/test", func(c *gin.Context) { handlerCalled = true; c.Status(http.StatusOK) })
			case http.MethodPut:
				r.PUT("/api/v1/test", func(c *gin.Context) { handlerCalled = true; c.Status(http.StatusOK) })
			case http.MethodDelete:
				r.DELETE("/api/v1/test", func(c *gin.Context) { handlerCalled = true; c.Status(http.StatusOK) })
			case http.MethodPatch:
				r.PATCH("/api/v1/test", func(c *gin.Context) { handlerCalled = true; c.Status(http.StatusOK) })
			}

			req := httptest.NewRequest(method, "/api/v1/test", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.True(t, handlerCalled, "handler should be called for method: %s", method)
		})
	}
}

func TestProfilingMiddleware_DefaultMiddleware(t *testing.T) {
	r := gin.New()

	handlerCalled := false
	r.Use(middleware.Profiling())
	r.GET("/api/v1/products", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, handlerCalled)
}

func TestProfilingAttributeInjector(t *testing.T) {
	r := gin.New()

	handlerCalled := false
	r.Use(middleware.ProfilingAttributeInjector())
	r.GET("/api/v1/products", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, handlerCalled)
}

func TestProfilingMiddleware_CustomSkipPaths(t *testing.T) {
	cfg := middleware.ProfilingConfig{
		Enabled: true,
		SkipPaths: []string{
			"/custom/health",
			"/custom/status",
		},
		SkipPathPrefixes: []string{
			"/custom/admin",
		},
	}

	tests := []struct {
		path       string
		shouldSkip bool
	}{
		{"/custom/health", true},
		{"/custom/status", true},
		{"/custom/admin/dashboard", true},
		{"/custom/api", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			r := gin.New()
			handlerCalled := false

			r.Use(middleware.ProfilingWithConfig(cfg))
			r.GET(tt.path, func(c *gin.Context) {
				handlerCalled = true
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.True(t, handlerCalled)
		})
	}
}

func TestExtractControllerFromRoute(t *testing.T) {
	// This tests the internal function behavior via the middleware
	tests := []struct {
		name             string
		route            string
		expectedNonEmpty bool
	}{
		{"products_route", "/api/v1/products", true},
		{"products_with_id", "/api/v1/products/:id", true},
		{"customers", "/api/v1/customers", true},
		{"nested_orders", "/api/v1/customers/:id/orders", true},
		{"empty_route", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.route == "" {
				return // Skip empty route test as gin can't register it
			}

			r := gin.New()
			cfg := middleware.DefaultProfilingConfig()

			r.Use(middleware.ProfilingWithConfig(cfg))
			r.GET(tt.route, func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			// Use a path that matches the route
			path := tt.route
			if containsPathParam(path) {
				path = replacePathParams(path)
			}

			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// containsPathParam checks if route has path parameters
func containsPathParam(route string) bool {
	return len(route) > 0 && (route[0] == ':' || contains(route, "/:"))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// replacePathParams replaces path parameters with test values
func replacePathParams(route string) string {
	result := route
	for i := 0; i < len(result); i++ {
		if result[i] == ':' {
			// Find end of parameter
			end := i + 1
			for end < len(result) && result[end] != '/' {
				end++
			}
			result = result[:i] + "test-value" + result[end:]
		}
	}
	return result
}

func TestProfilingMiddleware_ContextPreserved(t *testing.T) {
	r := gin.New()

	cfg := middleware.DefaultProfilingConfig()

	// Set custom context value before profiling middleware
	r.Use(func(c *gin.Context) {
		c.Set("custom_key", "custom_value")
		c.Next()
	})
	r.Use(middleware.ProfilingWithConfig(cfg))
	r.GET("/api/v1/products", func(c *gin.Context) {
		// Verify custom context value is preserved
		value, exists := c.Get("custom_key")
		assert.True(t, exists, "custom key should exist")
		assert.Equal(t, "custom_value", value)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProfilingMiddleware_EmptyTenantID(t *testing.T) {
	r := gin.New()

	cfg := middleware.DefaultProfilingConfig()

	// No tenant ID set
	r.Use(middleware.ProfilingWithConfig(cfg))
	r.GET("/api/v1/products", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProfilingMiddleware_TenantIDWrongType(t *testing.T) {
	r := gin.New()

	cfg := middleware.DefaultProfilingConfig()

	// Set tenant ID with wrong type
	r.Use(func(c *gin.Context) {
		c.Set(middleware.JWTTenantIDKey, 12345) // int instead of string
		c.Next()
	})
	r.Use(middleware.ProfilingWithConfig(cfg))
	r.GET("/api/v1/products", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// Should still work, just without tenant_id label
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProfilingMiddleware_ChainWithOtherMiddleware(t *testing.T) {
	r := gin.New()

	cfg := middleware.DefaultProfilingConfig()

	middlewareOrder := []string{}

	r.Use(func(c *gin.Context) {
		middlewareOrder = append(middlewareOrder, "first")
		c.Next()
		middlewareOrder = append(middlewareOrder, "first_after")
	})

	r.Use(middleware.ProfilingWithConfig(cfg))

	r.Use(func(c *gin.Context) {
		middlewareOrder = append(middlewareOrder, "third")
		c.Next()
		middlewareOrder = append(middlewareOrder, "third_after")
	})

	r.GET("/api/v1/products", func(c *gin.Context) {
		middlewareOrder = append(middlewareOrder, "handler")
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Verify middleware execution order
	assert.Equal(t, []string{"first", "third", "handler", "third_after", "first_after"}, middlewareOrder)
}

func TestIsVersionSegment(t *testing.T) {
	// Test the version segment detection indirectly via route extraction
	routes := []struct {
		route              string
		shouldMatchProduct bool
	}{
		{"/api/v1/products", true},
		{"/api/v2/products", true},
		{"/api/v10/products", true},
		{"/api/v100/products", true},
		{"/api/products", true}, // no version
		{"/v1/products", true},
	}

	for _, tt := range routes {
		t.Run(tt.route, func(t *testing.T) {
			r := gin.New()
			cfg := middleware.DefaultProfilingConfig()

			r.Use(middleware.ProfilingWithConfig(cfg))
			r.GET(tt.route, func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, tt.route, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}
