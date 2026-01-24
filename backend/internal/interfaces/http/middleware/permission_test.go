package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/erp/backend/internal/infrastructure/auth"
	"github.com/erp/backend/internal/infrastructure/config"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func newTestJWTServiceForPermission() *auth.JWTService {
	cfg := config.JWTConfig{
		Secret:                 "test-secret-key-at-least-32-chars",
		RefreshSecret:          "test-refresh-secret-key-32-chars",
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test-issuer",
		MaxRefreshCount:        10,
	}
	return auth.NewJWTService(cfg)
}

func newTestTokenWithPermissions(jwtService *auth.JWTService, permissions []string) *auth.TokenPair {
	input := auth.GenerateTokenInput{
		TenantID:    uuid.New(),
		UserID:      uuid.New(),
		Username:    "testuser",
		RoleIDs:     []uuid.UUID{uuid.New()},
		Permissions: permissions,
	}
	pair, _ := jwtService.GenerateTokenPair(input)
	return pair
}

func setupRouterWithJWT(jwtService *auth.JWTService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(JWTAuthMiddleware(jwtService))
	return router
}

// Test RequirePermission
func TestRequirePermission_WithValidPermission(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"product:read", "product:create"})

	router := setupRouterWithJWT(jwtService)
	router.GET("/products", RequirePermission("product:read"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequirePermission_WithoutPermission(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"product:read"})

	router := setupRouterWithJWT(jwtService)
	router.GET("/products", RequirePermission("product:delete"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response["success"].(bool))
	assert.NotNil(t, response["error"])
}

func TestRequirePermission_WithoutAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// No JWT middleware, claims will be nil
	router.GET("/products", RequirePermission("product:read"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// Test RequireAnyPermission
func TestRequireAnyPermission_WithOneMatch(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"product:read"})

	router := setupRouterWithJWT(jwtService)
	router.GET("/products", RequireAnyPermission("product:read", "product:create"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireAnyPermission_WithNoMatch(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"customer:read"})

	router := setupRouterWithJWT(jwtService)
	router.GET("/products", RequireAnyPermission("product:read", "product:create"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// Test RequireAllPermissions
func TestRequireAllPermissions_WithAllMatching(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"product:read", "product:create", "product:update"})

	router := setupRouterWithJWT(jwtService)
	router.GET("/products", RequireAllPermissions("product:read", "product:create"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireAllPermissions_WithPartialMatch(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"product:read"})

	router := setupRouterWithJWT(jwtService)
	router.GET("/products", RequireAllPermissions("product:read", "product:create"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// Test RequireResource
func TestRequireResource_GET(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"product:read"})

	router := setupRouterWithJWT(jwtService)
	router.GET("/products", RequireResource("product"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireResource_POST(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"product:create"})

	router := setupRouterWithJWT(jwtService)
	router.POST("/products", RequireResource("product"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodPost, "/products", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireResource_PUT(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"product:update"})

	router := setupRouterWithJWT(jwtService)
	router.PUT("/products/:id", RequireResource("product"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodPut, "/products/123", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireResource_PATCH(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"product:update"})

	router := setupRouterWithJWT(jwtService)
	router.PATCH("/products/:id", RequireResource("product"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodPatch, "/products/123", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireResource_DELETE(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"product:delete"})

	router := setupRouterWithJWT(jwtService)
	router.DELETE("/products/:id", RequireResource("product"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodDelete, "/products/123", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireResource_WrongPermission(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"product:read"})

	router := setupRouterWithJWT(jwtService)
	router.DELETE("/products/:id", RequireResource("product"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodDelete, "/products/123", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// Test RequireResourceAction
func TestRequireResourceAction(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"product:confirm"})

	router := setupRouterWithJWT(jwtService)
	router.POST("/products/:id/confirm", RequireResourceAction("product", "confirm"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodPost, "/products/123/confirm", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// Test RoutePermissionMiddleware
func TestRoutePermissionMiddleware_ExactPathMatch(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"product:read"})

	cfg := RoutePermissionConfig{
		Routes: []RoutePermission{
			{Method: "GET", Path: "/api/v1/products", Permissions: []string{"product:read"}},
		},
	}

	router := setupRouterWithJWT(jwtService)
	router.Use(RoutePermissionMiddleware(cfg))
	router.GET("/api/v1/products", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRoutePermissionMiddleware_PrefixMatch(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"product:read"})

	cfg := RoutePermissionConfig{
		Routes: []RoutePermission{
			{Method: "GET", Path: "/api/v1/products*", Permissions: []string{"product:read"}},
		},
	}

	router := setupRouterWithJWT(jwtService)
	router.Use(RoutePermissionMiddleware(cfg))
	router.GET("/api/v1/products/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products/123", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRoutePermissionMiddleware_WildcardMethod(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"product:manage"})

	cfg := RoutePermissionConfig{
		Routes: []RoutePermission{
			{Method: "*", Path: "/api/v1/products", Permissions: []string{"product:manage"}},
		},
	}

	router := setupRouterWithJWT(jwtService)
	router.Use(RoutePermissionMiddleware(cfg))
	router.GET("/api/v1/products", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.POST("/api/v1/products", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Test GET
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Test POST
	req = httptest.NewRequest(http.MethodPost, "/api/v1/products", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRoutePermissionMiddleware_RequireAll(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"product:read", "product:create"})

	cfg := RoutePermissionConfig{
		Routes: []RoutePermission{
			{Method: "GET", Path: "/api/v1/products", Permissions: []string{"product:read", "product:create"}, RequireAll: true},
		},
	}

	router := setupRouterWithJWT(jwtService)
	router.Use(RoutePermissionMiddleware(cfg))
	router.GET("/api/v1/products", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRoutePermissionMiddleware_RequireAll_Fail(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"product:read"})

	cfg := RoutePermissionConfig{
		Routes: []RoutePermission{
			{Method: "GET", Path: "/api/v1/products", Permissions: []string{"product:read", "product:create"}, RequireAll: true},
		},
	}

	router := setupRouterWithJWT(jwtService)
	router.Use(RoutePermissionMiddleware(cfg))
	router.GET("/api/v1/products", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRoutePermissionMiddleware_NoMatchingRoute_Allow(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{})

	cfg := RoutePermissionConfig{
		Routes: []RoutePermission{
			{Method: "GET", Path: "/api/v1/products", Permissions: []string{"product:read"}},
		},
		DefaultDeny: false, // Allow unmatched routes
	}

	router := setupRouterWithJWT(jwtService)
	router.Use(RoutePermissionMiddleware(cfg))
	router.GET("/api/v1/other", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/other", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRoutePermissionMiddleware_NoMatchingRoute_Deny(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{})

	cfg := RoutePermissionConfig{
		Routes: []RoutePermission{
			{Method: "GET", Path: "/api/v1/products", Permissions: []string{"product:read"}},
		},
		DefaultDeny: true, // Deny unmatched routes
	}

	router := setupRouterWithJWT(jwtService)
	router.Use(RoutePermissionMiddleware(cfg))
	router.GET("/api/v1/other", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/other", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// Test helper functions
func TestHasPermission(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"product:read", "product:create"})

	router := setupRouterWithJWT(jwtService)
	router.GET("/test", func(c *gin.Context) {
		assert.True(t, HasPermission(c, "product:read"))
		assert.True(t, HasPermission(c, "product:create"))
		assert.False(t, HasPermission(c, "product:delete"))
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHasAnyPermission(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"product:read"})

	router := setupRouterWithJWT(jwtService)
	router.GET("/test", func(c *gin.Context) {
		assert.True(t, HasAnyPermission(c, "product:read", "product:create"))
		assert.False(t, HasAnyPermission(c, "customer:read", "customer:create"))
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHasAllPermissions(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"product:read", "product:create"})

	router := setupRouterWithJWT(jwtService)
	router.GET("/test", func(c *gin.Context) {
		assert.True(t, HasAllPermissions(c, "product:read", "product:create"))
		assert.False(t, HasAllPermissions(c, "product:read", "product:delete"))
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMustHavePermission_Success(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"product:read"})

	router := setupRouterWithJWT(jwtService)
	router.GET("/test", func(c *gin.Context) {
		if MustHavePermission(c, "product:read") {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMustHavePermission_Fail(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"product:read"})

	router := setupRouterWithJWT(jwtService)
	router.GET("/test", func(c *gin.Context) {
		if MustHavePermission(c, "product:delete") {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// Test RequireCustomPermission
func TestRequireCustomPermission_Success(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"product:read"})

	// Custom check: allow if user has any product permission
	customCheck := func(claims *auth.Claims, c *gin.Context) bool {
		for _, p := range claims.Permissions {
			if len(p) >= 7 && p[:7] == "product" {
				return true
			}
		}
		return false
	}

	router := setupRouterWithJWT(jwtService)
	router.GET("/test", RequireCustomPermission(customCheck), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireCustomPermission_Fail(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"customer:read"})

	// Custom check: allow if user has any product permission
	customCheck := func(claims *auth.Claims, c *gin.Context) bool {
		for _, p := range claims.Permissions {
			if len(p) >= 7 && p[:7] == "product" {
				return true
			}
		}
		return false
	}

	router := setupRouterWithJWT(jwtService)
	router.GET("/test", RequireCustomPermission(customCheck), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// Test methodToAction
func TestMethodToAction(t *testing.T) {
	tests := []struct {
		method   string
		expected string
	}{
		{http.MethodGet, "read"},
		{http.MethodPost, "create"},
		{http.MethodPut, "update"},
		{http.MethodPatch, "update"},
		{http.MethodDelete, "delete"},
		{http.MethodHead, "read"},
		{http.MethodOptions, "read"},
		{"UNKNOWN", "read"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			assert.Equal(t, tt.expected, methodToAction(tt.method))
		})
	}
}

// Test with logger
func TestRequirePermission_WithLogger(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"product:read"})
	logger := zaptest.NewLogger(t)

	cfg := PermissionConfig{
		Logger: logger,
	}

	router := setupRouterWithJWT(jwtService)
	router.GET("/products", RequireAnyPermissionWithConfig(cfg, "product:read"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// Test custom OnDenied callback
func TestRequirePermission_WithOnDeniedCallback(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"customer:read"})

	customDeniedCalled := false
	cfg := PermissionConfig{
		OnDenied: func(c *gin.Context, requiredPerms []string) {
			customDeniedCalled = true
			c.AbortWithStatusJSON(http.StatusTeapot, gin.H{
				"custom": true,
				"required": requiredPerms,
			})
		},
	}

	router := setupRouterWithJWT(jwtService)
	router.GET("/products", RequireAnyPermissionWithConfig(cfg, "product:read"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.True(t, customDeniedCalled)
	assert.Equal(t, http.StatusTeapot, rec.Code)
}

// Test error response format
func TestPermissionDenied_ResponseFormat(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{})

	router := setupRouterWithJWT(jwtService)
	router.GET("/products", RequirePermission("product:read"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))

	errInfo := response["error"].(map[string]interface{})
	assert.Equal(t, "ERR_FORBIDDEN", errInfo["code"])
	assert.Contains(t, errInfo["message"], "insufficient permissions")
}

// Test HasPermission without claims
func TestHasPermission_WithoutClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		// No claims in context
		assert.False(t, HasPermission(c, "product:read"))
		assert.False(t, HasAnyPermission(c, "product:read"))
		assert.False(t, HasAllPermissions(c, "product:read"))
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// Test RoutePermissionMiddleware with logger
func TestRoutePermissionMiddleware_WithLogger(t *testing.T) {
	jwtService := newTestJWTServiceForPermission()
	pair := newTestTokenWithPermissions(jwtService, []string{"product:read"})
	logger := zaptest.NewLogger(t)

	cfg := RoutePermissionConfig{
		Routes: []RoutePermission{
			{Method: "GET", Path: "/api/v1/products", Permissions: []string{"product:read"}},
		},
		Logger: logger,
	}

	router := setupRouterWithJWT(jwtService)
	router.Use(RoutePermissionMiddleware(cfg))
	router.GET("/api/v1/products", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// Test matchRoute
func TestMatchRoute(t *testing.T) {
	tests := []struct {
		name     string
		route    RoutePermission
		method   string
		path     string
		expected bool
	}{
		{
			name:     "exact match",
			route:    RoutePermission{Method: "GET", Path: "/api/products"},
			method:   "GET",
			path:     "/api/products",
			expected: true,
		},
		{
			name:     "method mismatch",
			route:    RoutePermission{Method: "GET", Path: "/api/products"},
			method:   "POST",
			path:     "/api/products",
			expected: false,
		},
		{
			name:     "path mismatch",
			route:    RoutePermission{Method: "GET", Path: "/api/products"},
			method:   "GET",
			path:     "/api/customers",
			expected: false,
		},
		{
			name:     "wildcard method",
			route:    RoutePermission{Method: "*", Path: "/api/products"},
			method:   "DELETE",
			path:     "/api/products",
			expected: true,
		},
		{
			name:     "prefix match",
			route:    RoutePermission{Method: "GET", Path: "/api/products*"},
			method:   "GET",
			path:     "/api/products/123",
			expected: true,
		},
		{
			name:     "prefix match exact",
			route:    RoutePermission{Method: "GET", Path: "/api/products*"},
			method:   "GET",
			path:     "/api/products",
			expected: true,
		},
		{
			name:     "case insensitive method",
			route:    RoutePermission{Method: "get", Path: "/api/products"},
			method:   "GET",
			path:     "/api/products",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchRoute(&tt.route, tt.method, tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}
