package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/infrastructure/auth"
	"github.com/erp/backend/internal/infrastructure/config"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// Helper function to create a test JWT service
func newTestJWTServiceForSuperAdmin() *auth.JWTService {
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

// Helper to create a super admin token (system tenant + super admin role)
func newSuperAdminToken(jwtService *auth.JWTService) (*auth.TokenPair, auth.GenerateTokenInput) {
	systemTenantUUID, _ := uuid.Parse(identity.SystemTenantID)
	superAdminRoleUUID, _ := uuid.Parse(identity.SuperAdminRoleID)
	superAdminUserUUID, _ := uuid.Parse(identity.SuperAdminUserID)

	input := auth.GenerateTokenInput{
		TenantID:    systemTenantUUID,
		UserID:      superAdminUserUUID,
		Username:    "superadmin",
		RoleIDs:     []uuid.UUID{superAdminRoleUUID},
		Permissions: identity.TenantPermissions(),
	}
	pair, _ := jwtService.GenerateTokenPair(input)
	return pair, input
}

// Helper to create a system tenant user with tenant permissions but no super admin role
func newSystemTenantUserWithPermissions(jwtService *auth.JWTService) (*auth.TokenPair, auth.GenerateTokenInput) {
	systemTenantUUID, _ := uuid.Parse(identity.SystemTenantID)

	input := auth.GenerateTokenInput{
		TenantID:    systemTenantUUID,
		UserID:      uuid.New(),
		Username:    "system_user",
		RoleIDs:     []uuid.UUID{uuid.New()},                  // Not super admin role
		Permissions: []string{"tenant:read", "tenant:create"}, // Has some tenant permissions
	}
	pair, _ := jwtService.GenerateTokenPair(input)
	return pair, input
}

// Helper to create a regular tenant user (not system tenant)
func newRegularTenantUser(jwtService *auth.JWTService) (*auth.TokenPair, auth.GenerateTokenInput) {
	input := auth.GenerateTokenInput{
		TenantID:    uuid.New(), // Regular tenant, not system tenant
		UserID:      uuid.New(),
		Username:    "regular_user",
		RoleIDs:     []uuid.UUID{uuid.New()},
		Permissions: []string{"product:read", "product:create"},
	}
	pair, _ := jwtService.GenerateTokenPair(input)
	return pair, input
}

// Helper to create a system tenant user without super admin permissions
func newSystemTenantUserWithoutPermissions(jwtService *auth.JWTService) (*auth.TokenPair, auth.GenerateTokenInput) {
	systemTenantUUID, _ := uuid.Parse(identity.SystemTenantID)

	input := auth.GenerateTokenInput{
		TenantID:    systemTenantUUID,
		UserID:      uuid.New(),
		Username:    "system_user_no_perms",
		RoleIDs:     []uuid.UUID{uuid.New()},  // Not super admin role
		Permissions: []string{"product:read"}, // No tenant permissions
	}
	pair, _ := jwtService.GenerateTokenPair(input)
	return pair, input
}

// setupRouterWithMiddleware creates a test router with JWT and SuperAdmin middleware
func setupRouterWithMiddleware(jwtService *auth.JWTService, superAdminCfg SuperAdminConfig) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(JWTAuthMiddleware(jwtService))
	router.Use(SuperAdminMiddlewareWithConfig(superAdminCfg))
	return router
}

// TestSuperAdminMiddleware_ValidSuperAdmin tests that super admin can access admin routes
func TestSuperAdminMiddleware_ValidSuperAdmin(t *testing.T) {
	jwtService := newTestJWTServiceForSuperAdmin()
	pair, _ := newSuperAdminToken(jwtService)

	router := setupRouterWithMiddleware(jwtService, SuperAdminConfig{})
	router.GET("/api/v1/admin/tenants", func(c *gin.Context) {
		assert.True(t, IsSuperAdmin(c))
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tenants", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestSuperAdminMiddleware_SystemTenantWithPermissions tests system tenant user with tenant permissions
func TestSuperAdminMiddleware_SystemTenantWithPermissions(t *testing.T) {
	jwtService := newTestJWTServiceForSuperAdmin()
	pair, _ := newSystemTenantUserWithPermissions(jwtService)

	router := setupRouterWithMiddleware(jwtService, SuperAdminConfig{})
	router.GET("/api/v1/admin/tenants", func(c *gin.Context) {
		assert.True(t, IsSuperAdmin(c))
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tenants", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestSuperAdminMiddleware_RegularTenantDenied tests that regular tenant users are denied
func TestSuperAdminMiddleware_RegularTenantDenied(t *testing.T) {
	jwtService := newTestJWTServiceForSuperAdmin()
	pair, _ := newRegularTenantUser(jwtService)

	router := setupRouterWithMiddleware(jwtService, SuperAdminConfig{})
	router.GET("/api/v1/admin/tenants", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tenants", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "ERR_FORBIDDEN")
	assert.Contains(t, rec.Body.String(), "not a system administrator")
}

// TestSuperAdminMiddleware_SystemTenantWithoutPermissions tests system tenant user without permissions
func TestSuperAdminMiddleware_SystemTenantWithoutPermissions(t *testing.T) {
	jwtService := newTestJWTServiceForSuperAdmin()
	pair, _ := newSystemTenantUserWithoutPermissions(jwtService)

	router := setupRouterWithMiddleware(jwtService, SuperAdminConfig{})
	router.GET("/api/v1/admin/tenants", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tenants", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "ERR_FORBIDDEN")
	assert.Contains(t, rec.Body.String(), "insufficient super admin permissions")
}

// TestSuperAdminMiddleware_NoAuthentication tests request without authentication
func TestSuperAdminMiddleware_NoAuthentication(t *testing.T) {
	jwtService := newTestJWTServiceForSuperAdmin()

	router := setupRouterWithMiddleware(jwtService, SuperAdminConfig{})
	router.GET("/api/v1/admin/tenants", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tenants", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// JWT middleware should return 401 first
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestSuperAdminMiddleware_InvalidToken tests request with invalid token
func TestSuperAdminMiddleware_InvalidToken(t *testing.T) {
	jwtService := newTestJWTServiceForSuperAdmin()

	router := setupRouterWithMiddleware(jwtService, SuperAdminConfig{})
	router.GET("/api/v1/admin/tenants", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tenants", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// JWT middleware should return 401 first
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestSuperAdminMiddleware_WithLogger tests middleware with logger
func TestSuperAdminMiddleware_WithLogger(t *testing.T) {
	jwtService := newTestJWTServiceForSuperAdmin()
	pair, _ := newSuperAdminToken(jwtService)
	logger := zaptest.NewLogger(t)

	router := setupRouterWithMiddleware(jwtService, SuperAdminConfig{Logger: logger})
	router.GET("/api/v1/admin/tenants", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tenants", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestSuperAdminMiddleware_WithLoggerDenied tests middleware logging on denied access
func TestSuperAdminMiddleware_WithLoggerDenied(t *testing.T) {
	jwtService := newTestJWTServiceForSuperAdmin()
	pair, _ := newRegularTenantUser(jwtService)
	logger := zaptest.NewLogger(t)

	router := setupRouterWithMiddleware(jwtService, SuperAdminConfig{Logger: logger})
	router.GET("/api/v1/admin/tenants", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tenants", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// TestSuperAdminMiddleware_CustomOnDenied tests custom OnDenied callback
func TestSuperAdminMiddleware_CustomOnDenied(t *testing.T) {
	jwtService := newTestJWTServiceForSuperAdmin()
	pair, _ := newRegularTenantUser(jwtService)

	customDeniedCalled := false
	customReason := ""

	cfg := SuperAdminConfig{
		OnDenied: func(c *gin.Context, reason string) {
			customDeniedCalled = true
			customReason = reason
			c.AbortWithStatusJSON(http.StatusTeapot, gin.H{"custom": "denied"})
		},
	}

	router := setupRouterWithMiddleware(jwtService, cfg)
	router.GET("/api/v1/admin/tenants", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tenants", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.True(t, customDeniedCalled)
	assert.Contains(t, customReason, "not a system administrator")
	assert.Equal(t, http.StatusTeapot, rec.Code)
}

// TestIsSuperAdmin_True tests IsSuperAdmin helper when user is super admin
func TestIsSuperAdmin_True(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set(IsSuperAdminKey, true)

	assert.True(t, IsSuperAdmin(c))
}

// TestIsSuperAdmin_False tests IsSuperAdmin helper when user is not super admin
func TestIsSuperAdmin_False(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set(IsSuperAdminKey, false)

	assert.False(t, IsSuperAdmin(c))
}

// TestIsSuperAdmin_NotSet tests IsSuperAdmin helper when flag is not set
func TestIsSuperAdmin_NotSet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	assert.False(t, IsSuperAdmin(c))
}

// TestIsSuperAdmin_WrongType tests IsSuperAdmin helper when flag is wrong type
func TestIsSuperAdmin_WrongType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set(IsSuperAdminKey, "not a bool")

	assert.False(t, IsSuperAdmin(c))
}

// TestRequireSuperAdmin_Success tests RequireSuperAdmin helper when user is super admin
func TestRequireSuperAdmin_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set(IsSuperAdminKey, true)

	result := RequireSuperAdmin(c)

	assert.True(t, result)
	assert.Equal(t, http.StatusOK, rec.Code) // Not aborted
}

// TestRequireSuperAdmin_Denied tests RequireSuperAdmin helper when user is not super admin
func TestRequireSuperAdmin_Denied(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set(IsSuperAdminKey, false)

	result := RequireSuperAdmin(c)

	assert.False(t, result)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "ERR_FORBIDDEN")
}

// TestCheckSuperAdminFromClaims_ValidSuperAdmin tests CheckSuperAdminFromClaims with valid super admin
func TestCheckSuperAdminFromClaims_ValidSuperAdmin(t *testing.T) {
	claims := &auth.Claims{
		TenantID:    identity.SystemTenantID,
		UserID:      identity.SuperAdminUserID,
		RoleIDs:     []string{identity.SuperAdminRoleID},
		Permissions: identity.TenantPermissions(),
	}

	assert.True(t, CheckSuperAdminFromClaims(claims))
}

// TestCheckSuperAdminFromClaims_SystemTenantWithPermissions tests with tenant permissions
func TestCheckSuperAdminFromClaims_SystemTenantWithPermissions(t *testing.T) {
	claims := &auth.Claims{
		TenantID:    identity.SystemTenantID,
		UserID:      uuid.New().String(),
		RoleIDs:     []string{uuid.New().String()},
		Permissions: []string{"tenant:read"},
	}

	assert.True(t, CheckSuperAdminFromClaims(claims))
}

// TestCheckSuperAdminFromClaims_RegularTenant tests with regular tenant
func TestCheckSuperAdminFromClaims_RegularTenant(t *testing.T) {
	claims := &auth.Claims{
		TenantID:    uuid.New().String(),
		UserID:      uuid.New().String(),
		RoleIDs:     []string{uuid.New().String()},
		Permissions: []string{"product:read"},
	}

	assert.False(t, CheckSuperAdminFromClaims(claims))
}

// TestCheckSuperAdminFromClaims_SystemTenantNoPermissions tests system tenant without permissions
func TestCheckSuperAdminFromClaims_SystemTenantNoPermissions(t *testing.T) {
	claims := &auth.Claims{
		TenantID:    identity.SystemTenantID,
		UserID:      uuid.New().String(),
		RoleIDs:     []string{uuid.New().String()},
		Permissions: []string{"product:read"},
	}

	assert.False(t, CheckSuperAdminFromClaims(claims))
}

// TestCheckSuperAdminFromClaims_NilClaims tests with nil claims
func TestCheckSuperAdminFromClaims_NilClaims(t *testing.T) {
	assert.False(t, CheckSuperAdminFromClaims(nil))
}

// TestSuperAdminMiddleware_AllTenantPermissions tests each tenant permission grants access
func TestSuperAdminMiddleware_AllTenantPermissions(t *testing.T) {
	jwtService := newTestJWTServiceForSuperAdmin()
	systemTenantUUID, _ := uuid.Parse(identity.SystemTenantID)

	tenantPermissions := identity.TenantPermissions()

	for _, perm := range tenantPermissions {
		t.Run("Permission_"+perm, func(t *testing.T) {
			input := auth.GenerateTokenInput{
				TenantID:    systemTenantUUID,
				UserID:      uuid.New(),
				Username:    "test_user",
				RoleIDs:     []uuid.UUID{uuid.New()},
				Permissions: []string{perm}, // Only this permission
			}
			pair, _ := jwtService.GenerateTokenPair(input)

			router := setupRouterWithMiddleware(jwtService, SuperAdminConfig{})
			router.GET("/api/v1/admin/tenants", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tenants", nil)
			req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code, "Permission %s should grant access", perm)
		})
	}
}

// TestSuperAdminMiddleware_SuperAdminRoleOnly tests access with only super admin role (no permissions)
func TestSuperAdminMiddleware_SuperAdminRoleOnly(t *testing.T) {
	jwtService := newTestJWTServiceForSuperAdmin()
	systemTenantUUID, _ := uuid.Parse(identity.SystemTenantID)
	superAdminRoleUUID, _ := uuid.Parse(identity.SuperAdminRoleID)

	input := auth.GenerateTokenInput{
		TenantID:    systemTenantUUID,
		UserID:      uuid.New(),
		Username:    "super_admin_role_only",
		RoleIDs:     []uuid.UUID{superAdminRoleUUID},
		Permissions: []string{}, // No permissions, but has super admin role
	}
	pair, _ := jwtService.GenerateTokenPair(input)

	router := setupRouterWithMiddleware(jwtService, SuperAdminConfig{})
	router.GET("/api/v1/admin/tenants", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tenants", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestSuperAdminMiddleware_ContextFlagSet tests that IsSuperAdmin flag is set in context
func TestSuperAdminMiddleware_ContextFlagSet(t *testing.T) {
	jwtService := newTestJWTServiceForSuperAdmin()
	pair, _ := newSuperAdminToken(jwtService)

	var capturedIsSuperAdmin bool

	router := setupRouterWithMiddleware(jwtService, SuperAdminConfig{})
	router.GET("/api/v1/admin/tenants", func(c *gin.Context) {
		capturedIsSuperAdmin = IsSuperAdmin(c)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tenants", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, capturedIsSuperAdmin)
}

// TestSuperAdminMiddleware_MultipleRoutes tests middleware on multiple admin routes
func TestSuperAdminMiddleware_MultipleRoutes(t *testing.T) {
	jwtService := newTestJWTServiceForSuperAdmin()
	pair, _ := newSuperAdminToken(jwtService)

	router := setupRouterWithMiddleware(jwtService, SuperAdminConfig{})

	routes := []string{
		"/api/v1/admin/tenants",
		"/api/v1/admin/tenants/:id",
		"/api/v1/admin/users",
		"/api/v1/admin/audit-logs",
	}

	for _, route := range routes {
		router.GET(route, func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
	}

	for _, route := range routes {
		t.Run("Route_"+route, func(t *testing.T) {
			// Replace :id with actual ID for parameterized routes
			testPath := route
			if route == "/api/v1/admin/tenants/:id" {
				testPath = "/api/v1/admin/tenants/" + uuid.New().String()
			}

			req := httptest.NewRequest(http.MethodGet, testPath, nil)
			req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code, "Route %s should be accessible", route)
		})
	}
}

// TestSuperAdminMiddleware_ErrorResponseFormat tests error response format
func TestSuperAdminMiddleware_ErrorResponseFormat(t *testing.T) {
	jwtService := newTestJWTServiceForSuperAdmin()
	pair, _ := newRegularTenantUser(jwtService)

	router := setupRouterWithMiddleware(jwtService, SuperAdminConfig{})
	router.GET("/api/v1/admin/tenants", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tenants", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)

	// Verify response structure
	body := rec.Body.String()
	assert.Contains(t, body, `"success":false`)
	assert.Contains(t, body, `"error"`)
	assert.Contains(t, body, `"code":"ERR_FORBIDDEN"`)
	assert.Contains(t, body, `"message"`)
}

// TestSuperAdminMiddleware_Performance tests middleware performance
func TestSuperAdminMiddleware_Performance(t *testing.T) {
	jwtService := newTestJWTServiceForSuperAdmin()
	pair, _ := newSuperAdminToken(jwtService)

	router := setupRouterWithMiddleware(jwtService, SuperAdminConfig{})
	router.GET("/api/v1/admin/tenants", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Warm up
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tenants", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// Measure performance
	iterations := 1000
	start := time.Now()

	for i := 0; i < iterations; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tenants", nil)
		req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
	}

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(iterations)

	// Acceptance criteria: < 1ms per request
	assert.Less(t, avgTime, 1*time.Millisecond, "Average middleware overhead should be < 1ms, got %v", avgTime)
	t.Logf("Average middleware overhead: %v", avgTime)
}

// TestCheckSuperAdminAccess_EmptyRoleIDs tests with empty role IDs
func TestCheckSuperAdminAccess_EmptyRoleIDs(t *testing.T) {
	claims := &auth.Claims{
		TenantID:    identity.SystemTenantID,
		UserID:      uuid.New().String(),
		RoleIDs:     []string{},
		Permissions: []string{"tenant:read"},
	}

	// Should still pass because of tenant:read permission
	assert.True(t, CheckSuperAdminFromClaims(claims))
}

// TestCheckSuperAdminAccess_EmptyPermissions tests with empty permissions
func TestCheckSuperAdminAccess_EmptyPermissions(t *testing.T) {
	claims := &auth.Claims{
		TenantID:    identity.SystemTenantID,
		UserID:      uuid.New().String(),
		RoleIDs:     []string{identity.SuperAdminRoleID},
		Permissions: []string{},
	}

	// Should still pass because of super admin role
	assert.True(t, CheckSuperAdminFromClaims(claims))
}

// TestCheckSuperAdminAccess_BothEmpty tests with both empty
func TestCheckSuperAdminAccess_BothEmpty(t *testing.T) {
	claims := &auth.Claims{
		TenantID:    identity.SystemTenantID,
		UserID:      uuid.New().String(),
		RoleIDs:     []string{},
		Permissions: []string{},
	}

	// Should fail - no role and no permissions
	assert.False(t, CheckSuperAdminFromClaims(claims))
}

// TestSuperAdminMiddleware_WithZapLogger tests with production zap logger
func TestSuperAdminMiddleware_WithZapLogger(t *testing.T) {
	jwtService := newTestJWTServiceForSuperAdmin()
	pair, _ := newSuperAdminToken(jwtService)

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	router := setupRouterWithMiddleware(jwtService, SuperAdminConfig{Logger: logger})
	router.GET("/api/v1/admin/tenants", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tenants", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestSuperAdminMiddleware_DefaultConfig tests the default SuperAdminMiddleware function
func TestSuperAdminMiddleware_DefaultConfig(t *testing.T) {
	jwtService := newTestJWTServiceForSuperAdmin()
	pair, _ := newSuperAdminToken(jwtService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(JWTAuthMiddleware(jwtService))
	router.Use(SuperAdminMiddleware()) // Use default config
	router.GET("/api/v1/admin/tenants", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tenants", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestSuperAdminMiddleware_UnauthorizedCustomCallback tests custom callback for unauthorized
func TestSuperAdminMiddleware_UnauthorizedCustomCallback(t *testing.T) {
	customDeniedCalled := false

	cfg := SuperAdminConfig{
		OnDenied: func(c *gin.Context, reason string) {
			customDeniedCalled = true
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"custom": "unauthorized"})
		},
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Skip JWT middleware to test unauthorized path in SuperAdmin middleware
	router.Use(SuperAdminMiddlewareWithConfig(cfg))
	router.GET("/api/v1/admin/tenants", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tenants", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.True(t, customDeniedCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestSuperAdminMiddleware_UnauthorizedDefaultResponse tests default unauthorized response
func TestSuperAdminMiddleware_UnauthorizedDefaultResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Skip JWT middleware to test unauthorized path in SuperAdmin middleware
	router.Use(SuperAdminMiddlewareWithConfig(SuperAdminConfig{}))
	router.GET("/api/v1/admin/tenants", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tenants", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "ERR_UNAUTHORIZED")
	assert.Contains(t, rec.Body.String(), "Authentication required for admin access")
}

// TestSuperAdminMiddleware_LoggerOnInsufficientPermissions tests logger on insufficient permissions
func TestSuperAdminMiddleware_LoggerOnInsufficientPermissions(t *testing.T) {
	jwtService := newTestJWTServiceForSuperAdmin()
	pair, _ := newSystemTenantUserWithoutPermissions(jwtService)
	logger := zaptest.NewLogger(t)

	router := setupRouterWithMiddleware(jwtService, SuperAdminConfig{Logger: logger})
	router.GET("/api/v1/admin/tenants", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tenants", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}
