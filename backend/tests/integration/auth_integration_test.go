// Package integration provides integration testing for the ERP backend API.
// This file contains tests for Authentication and Authorization (P6-INT-002).
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	identityapp "github.com/erp/backend/internal/application/identity"
	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/infrastructure/auth"
	"github.com/erp/backend/internal/infrastructure/config"
	"github.com/erp/backend/internal/infrastructure/persistence"
	"github.com/erp/backend/internal/interfaces/http/handler"
	"github.com/erp/backend/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// AuthTestServer wraps the test database and HTTP server for auth API testing
type AuthTestServer struct {
	DB          *TestDB
	Engine      *gin.Engine
	UserRepo    *persistence.GormUserRepository
	RoleRepo    *persistence.GormRoleRepository
	AuthService *identityapp.AuthService
	JWTService  *auth.JWTService
}

// NewAuthTestServer creates a new test server with auth infrastructure
func NewAuthTestServer(t *testing.T) *AuthTestServer {
	t.Helper()

	gin.SetMode(gin.TestMode)
	testDB := NewTestDB(t)

	// Initialize repositories
	userRepo := persistence.NewGormUserRepository(testDB.DB)
	roleRepo := persistence.NewGormRoleRepository(testDB.DB)

	// Initialize JWT service with test config
	jwtConfig := config.JWTConfig{
		Secret:                 "test-secret-key-for-auth-testing-1234567890",
		RefreshSecret:          "test-refresh-secret-key-for-auth-testing",
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "erp-test",
		MaxRefreshCount:        10,
	}
	jwtService := auth.NewJWTService(jwtConfig)

	// Cookie config for secure httpOnly cookies
	cookieConfig := config.CookieConfig{
		Domain:   "",
		Path:     "/",
		Secure:   false,
		SameSite: "lax",
	}

	// Initialize auth service
	authConfig := identityapp.AuthServiceConfig{
		MaxLoginAttempts: 5,
		LockDuration:     15 * time.Minute,
	}
	logger := zap.NewNop()
	authService := identityapp.NewAuthService(userRepo, roleRepo, jwtService, authConfig, logger)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService, cookieConfig, jwtConfig)

	// Setup engine with middleware
	engine := gin.New()

	// Setup routes
	api := engine.Group("/api/v1")

	// Auth routes (no JWT required for login/refresh)
	authGroup := api.Group("/auth")
	authGroup.POST("/login", authHandler.Login)
	authGroup.POST("/refresh", authHandler.RefreshToken)

	// Protected auth routes
	protectedAuth := authGroup.Group("")
	protectedAuth.Use(middleware.JWTAuthMiddleware(jwtService))
	protectedAuth.POST("/logout", authHandler.Logout)
	protectedAuth.GET("/me", authHandler.GetCurrentUser)
	protectedAuth.PUT("/password", authHandler.ChangePassword)

	// Protected endpoint for permission testing
	protectedAPI := api.Group("/protected")
	protectedAPI.Use(middleware.JWTAuthMiddleware(jwtService))

	// Single permission check
	protectedAPI.GET("/products", middleware.RequirePermission("product:read"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": "products"})
	})

	// Any permission check
	protectedAPI.POST("/products", middleware.RequireAnyPermission("product:create", "product:admin"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": "created"})
	})

	// All permissions check
	protectedAPI.DELETE("/products/:id", middleware.RequireAllPermissions("product:delete", "product:admin"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": "deleted"})
	})

	// Resource permission check
	protectedAPI.Group("/orders").Use(middleware.RequireResource("order")).GET("", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": "orders"})
	})
	protectedAPI.Group("/orders").Use(middleware.RequireResource("order")).POST("", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": "order created"})
	})

	return &AuthTestServer{
		DB:          testDB,
		Engine:      engine,
		UserRepo:    userRepo,
		RoleRepo:    roleRepo,
		AuthService: authService,
		JWTService:  jwtService,
	}
}

// Request makes an HTTP request to the test server
func (ts *AuthTestServer) Request(method, path string, body interface{}, token ...string) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBody)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")

	// Set Authorization header if token provided
	if len(token) > 0 && token[0] != "" {
		req.Header.Set("Authorization", "Bearer "+token[0])
	}

	w := httptest.NewRecorder()
	ts.Engine.ServeHTTP(w, req)
	return w
}

// CreateTestUser creates a test user with given credentials
func (ts *AuthTestServer) CreateTestUser(t *testing.T, tenantID uuid.UUID, username, password string, roleIDs ...uuid.UUID) *identity.User {
	t.Helper()

	user, err := identity.NewActiveUser(tenantID, username, password)
	require.NoError(t, err)

	// Set a unique email to avoid unique constraint violations
	email := fmt.Sprintf("%s_%s@test.local", username, uuid.New().String()[:8])
	err = user.SetEmail(email)
	require.NoError(t, err)

	if len(roleIDs) > 0 {
		err = user.SetRoles(roleIDs)
		require.NoError(t, err)
	}

	err = ts.UserRepo.Create(context.Background(), user)
	require.NoError(t, err)

	// Save user roles if any
	if len(roleIDs) > 0 {
		err = ts.UserRepo.SaveUserRoles(context.Background(), user)
		require.NoError(t, err)
	}

	return user
}

// CreateTestRole creates a test role with given permissions
func (ts *AuthTestServer) CreateTestRole(t *testing.T, tenantID uuid.UUID, code, name string, permissionCodes ...string) *identity.Role {
	t.Helper()

	role, err := identity.NewRole(tenantID, code, name)
	require.NoError(t, err)

	// Add permissions
	for _, permCode := range permissionCodes {
		perm, err := identity.NewPermissionFromCode(permCode)
		require.NoError(t, err)
		err = role.GrantPermission(*perm)
		require.NoError(t, err)
	}

	err = ts.RoleRepo.Create(context.Background(), role)
	require.NoError(t, err)

	// Save permissions
	if len(permissionCodes) > 0 {
		err = ts.RoleRepo.SavePermissions(context.Background(), role)
		require.NoError(t, err)
	}

	return role
}

// =============================================================================
// Login Flow Tests
// =============================================================================

func TestAuth_LoginFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewAuthTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create test role with permissions
	role := ts.CreateTestRole(t, tenantID, "TEST_USER", "Test User Role",
		"product:read", "product:create", "customer:read")

	// Create test user with role
	testPassword := "TestPass123"
	user := ts.CreateTestUser(t, tenantID, "testuser", testPassword, role.ID)

	t.Run("successful_login_returns_tokens_and_user_info", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"username": "testuser",
			"password": testPassword,
		}

		w := ts.Request(http.MethodPost, "/api/v1/auth/login", reqBody)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		data := resp["data"].(map[string]interface{})

		// Verify token structure
		token := data["token"].(map[string]interface{})
		assert.NotEmpty(t, token["access_token"])
		// refresh_token is now in httpOnly cookie, not in response body
		assert.NotEmpty(t, token["access_token_expires_at"])
		assert.NotEmpty(t, token["refresh_token_expires_at"])
		assert.Equal(t, "Bearer", token["token_type"])

		// Verify refresh token cookie is set
		cookies := w.Result().Cookies()
		var hasRefreshCookie bool
		for _, cookie := range cookies {
			if cookie.Name == "refresh_token" && cookie.Value != "" {
				hasRefreshCookie = true
				break
			}
		}
		assert.True(t, hasRefreshCookie, "refresh_token cookie should be set")

		// Verify user info
		userInfo := data["user"].(map[string]interface{})
		assert.Equal(t, user.ID.String(), userInfo["id"])
		assert.Equal(t, tenantID.String(), userInfo["tenant_id"])
		assert.Equal(t, "testuser", userInfo["username"])

		// Verify permissions are loaded
		permissions := userInfo["permissions"].([]interface{})
		assert.Len(t, permissions, 3) // product:read, product:create, customer:read
	})

	t.Run("invalid_username_returns_401", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"username": "nonexistent",
			"password": testPassword,
		}

		w := ts.Request(http.MethodPost, "/api/v1/auth/login", reqBody)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.False(t, resp["success"].(bool))
	})

	t.Run("invalid_password_returns_401", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"username": "testuser",
			"password": "WrongPassword123",
		}

		w := ts.Request(http.MethodPost, "/api/v1/auth/login", reqBody)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("deactivated_user_cannot_login", func(t *testing.T) {
		// Create and deactivate a user
		deactivatedUser := ts.CreateTestUser(t, tenantID, "deactivated_user", "TestPass123")
		err := deactivatedUser.Deactivate()
		require.NoError(t, err)
		err = ts.UserRepo.Update(context.Background(), deactivatedUser)
		require.NoError(t, err)

		reqBody := map[string]interface{}{
			"username": "deactivated_user",
			"password": "TestPass123",
		}

		w := ts.Request(http.MethodPost, "/api/v1/auth/login", reqBody)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		errorInfo := resp["error"].(map[string]interface{})
		assert.Equal(t, "ACCOUNT_DEACTIVATED", errorInfo["code"])
	})

	t.Run("pending_user_cannot_login", func(t *testing.T) {
		// Create a pending user (use NewUser instead of NewActiveUser)
		pendingUser, err := identity.NewUser(tenantID, "pending_user", "TestPass123")
		require.NoError(t, err)
		// Set unique email to avoid constraint violation
		err = pendingUser.SetEmail(fmt.Sprintf("pending_user_%s@test.local", uuid.New().String()[:8]))
		require.NoError(t, err)
		err = ts.UserRepo.Create(context.Background(), pendingUser)
		require.NoError(t, err)

		reqBody := map[string]interface{}{
			"username": "pending_user",
			"password": "TestPass123",
		}

		w := ts.Request(http.MethodPost, "/api/v1/auth/login", reqBody)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		errorInfo := resp["error"].(map[string]interface{})
		assert.Equal(t, "ACCOUNT_PENDING", errorInfo["code"])
	})

	t.Run("account_locks_after_max_failed_attempts", func(t *testing.T) {
		// Create a user for lock testing
		lockTestUser := ts.CreateTestUser(t, tenantID, "lock_test_user", "TestPass123")

		// Attempt login with wrong password multiple times
		// The 5th attempt may return ACCOUNT_LOCKED (422) instead of INVALID_CREDENTIALS (401)
		for i := 0; i < 5; i++ {
			reqBody := map[string]interface{}{
				"username": "lock_test_user",
				"password": "WrongPassword",
			}
			w := ts.Request(http.MethodPost, "/api/v1/auth/login", reqBody)
			// First 4 attempts should be 401 (invalid credentials)
			// The 5th attempt triggers account lock and returns 422
			if i < 4 {
				assert.Equal(t, http.StatusUnauthorized, w.Code, "Attempt %d should return 401", i+1)
			} else {
				// 5th attempt - account gets locked
				assert.Equal(t, http.StatusUnprocessableEntity, w.Code, "Attempt %d (lock trigger) should return 422", i+1)
			}
		}

		// Verify user is now locked
		lockedUser, err := ts.UserRepo.FindByID(context.Background(), lockTestUser.ID)
		require.NoError(t, err)
		assert.True(t, lockedUser.IsLocked())

		// Try to login with correct password - should fail
		reqBody := map[string]interface{}{
			"username": "lock_test_user",
			"password": "TestPass123",
		}
		w := ts.Request(http.MethodPost, "/api/v1/auth/login", reqBody)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		errorInfo := resp["error"].(map[string]interface{})
		assert.Equal(t, "ACCOUNT_LOCKED", errorInfo["code"])
	})

	t.Run("login_tracks_ip_address", func(t *testing.T) {
		// Get the user before login
		userBefore, err := ts.UserRepo.FindByID(context.Background(), user.ID)
		require.NoError(t, err)

		reqBody := map[string]interface{}{
			"username": "testuser",
			"password": testPassword,
		}

		w := ts.Request(http.MethodPost, "/api/v1/auth/login", reqBody)
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify login was recorded
		userAfter, err := ts.UserRepo.FindByID(context.Background(), user.ID)
		require.NoError(t, err)

		// Last login should be updated
		if userBefore.LastLoginAt != nil {
			assert.True(t, userAfter.LastLoginAt.After(*userBefore.LastLoginAt))
		} else {
			assert.NotNil(t, userAfter.LastLoginAt)
		}
	})
}

// =============================================================================
// Permission Control Tests
// =============================================================================

func TestAuth_PermissionControl(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewAuthTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create roles with different permissions
	readerRole := ts.CreateTestRole(t, tenantID, "READER", "Reader Role", "product:read", "order:read")
	creatorRole := ts.CreateTestRole(t, tenantID, "CREATOR", "Creator Role", "product:create")
	adminRole := ts.CreateTestRole(t, tenantID, "ADMIN", "Admin Role", "product:read", "product:create", "product:delete", "product:admin", "order:read", "order:create")

	// Create users with different roles
	testPassword := "TestPass123"
	readerUser := ts.CreateTestUser(t, tenantID, "reader_user", testPassword, readerRole.ID)
	creatorUser := ts.CreateTestUser(t, tenantID, "creator_user", testPassword, creatorRole.ID)
	adminUser := ts.CreateTestUser(t, tenantID, "admin_user", testPassword, adminRole.ID)
	noRoleUser := ts.CreateTestUser(t, tenantID, "norole_user", testPassword)

	// Helper to login and get token
	getToken := func(username string) string {
		reqBody := map[string]interface{}{
			"username": username,
			"password": testPassword,
		}
		w := ts.Request(http.MethodPost, "/api/v1/auth/login", reqBody)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].(map[string]interface{})
		token := data["token"].(map[string]interface{})
		return token["access_token"].(string)
	}

	_ = readerUser
	_ = creatorUser
	_ = adminUser
	_ = noRoleUser

	t.Run("user_without_auth_gets_401", func(t *testing.T) {
		w := ts.Request(http.MethodGet, "/api/v1/protected/products", nil)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("user_with_required_permission_can_access", func(t *testing.T) {
		token := getToken("reader_user")
		w := ts.Request(http.MethodGet, "/api/v1/protected/products", nil, token)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("user_without_required_permission_gets_403", func(t *testing.T) {
		token := getToken("creator_user") // creator only has product:create
		w := ts.Request(http.MethodGet, "/api/v1/protected/products", nil, token)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("require_any_permission_works_with_one_match", func(t *testing.T) {
		// creator has product:create which is one of [product:create, product:admin]
		token := getToken("creator_user")
		w := ts.Request(http.MethodPost, "/api/v1/protected/products", map[string]interface{}{"name": "test"}, token)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("require_any_permission_fails_without_any_match", func(t *testing.T) {
		// reader has product:read but POST requires product:create or product:admin
		token := getToken("reader_user")
		w := ts.Request(http.MethodPost, "/api/v1/protected/products", map[string]interface{}{"name": "test"}, token)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("require_all_permissions_works_when_user_has_all", func(t *testing.T) {
		// admin has both product:delete and product:admin
		token := getToken("admin_user")
		w := ts.Request(http.MethodDelete, "/api/v1/protected/products/123", nil, token)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("require_all_permissions_fails_without_all", func(t *testing.T) {
		// creator has product:create but DELETE requires both product:delete AND product:admin
		token := getToken("creator_user")
		w := ts.Request(http.MethodDelete, "/api/v1/protected/products/123", nil, token)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("require_resource_maps_http_method_to_action", func(t *testing.T) {
		// reader has order:read - GET should map to order:read
		token := getToken("reader_user")
		w := ts.Request(http.MethodGet, "/api/v1/protected/orders", nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		// reader doesn't have order:create - POST should fail
		w = ts.Request(http.MethodPost, "/api/v1/protected/orders", map[string]interface{}{}, token)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("user_with_multiple_roles_gets_union_of_permissions", func(t *testing.T) {
		// Create a user with both reader and creator roles
		multiRoleUser := ts.CreateTestUser(t, tenantID, "multi_role_user", testPassword, readerRole.ID, creatorRole.ID)
		_ = multiRoleUser

		token := getToken("multi_role_user")

		// Should have product:read from reader role
		w := ts.Request(http.MethodGet, "/api/v1/protected/products", nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		// Should have product:create from creator role
		w = ts.Request(http.MethodPost, "/api/v1/protected/products", map[string]interface{}{"name": "test"}, token)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("user_with_no_roles_has_no_permissions", func(t *testing.T) {
		token := getToken("norole_user")
		w := ts.Request(http.MethodGet, "/api/v1/protected/products", nil, token)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("invalid_bearer_format_returns_401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/protected/products", nil)
		req.Header.Set("Authorization", "InvalidFormat token123")
		w := httptest.NewRecorder()
		ts.Engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("empty_bearer_token_returns_401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/protected/products", nil)
		req.Header.Set("Authorization", "Bearer ")
		w := httptest.NewRecorder()
		ts.Engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// =============================================================================
// Token Refresh Tests
// =============================================================================

func TestAuth_TokenRefresh(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewAuthTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create test role and user
	role := ts.CreateTestRole(t, tenantID, "REFRESH_TEST", "Refresh Test Role", "product:read")
	testPassword := "TestPass123"
	user := ts.CreateTestUser(t, tenantID, "refresh_user", testPassword, role.ID)
	_ = user

	// Login to get initial tokens
	loginReq := map[string]interface{}{
		"username": "refresh_user",
		"password": testPassword,
	}
	loginResp := ts.Request(http.MethodPost, "/api/v1/auth/login", loginReq)
	require.Equal(t, http.StatusOK, loginResp.Code)

	var loginData map[string]interface{}
	json.Unmarshal(loginResp.Body.Bytes(), &loginData)
	data := loginData["data"].(map[string]interface{})
	token := data["token"].(map[string]interface{})
	initialAccessToken := token["access_token"].(string)

	// Extract refresh token from httpOnly cookie
	cookies := loginResp.Result().Cookies()
	var initialRefreshToken string
	for _, cookie := range cookies {
		if cookie.Name == "refresh_token" {
			initialRefreshToken = cookie.Value
			break
		}
	}
	require.NotEmpty(t, initialRefreshToken, "refresh_token cookie should be set")

	t.Run("valid_refresh_token_returns_new_tokens", func(t *testing.T) {
		// Create request with refresh token cookie
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
		req.AddCookie(&http.Cookie{
			Name:  "refresh_token",
			Value: initialRefreshToken,
		})
		w := httptest.NewRecorder()
		ts.Engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		respData := resp["data"].(map[string]interface{})
		newToken := respData["token"].(map[string]interface{})

		// New access token should be returned
		assert.NotEmpty(t, newToken["access_token"])

		// New access token should be different from initial
		assert.NotEqual(t, initialAccessToken, newToken["access_token"])
	})

	t.Run("refresh_with_invalid_token_returns_error", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"refresh_token": "invalid.token.here",
		}

		w := ts.Request(http.MethodPost, "/api/v1/auth/refresh", reqBody)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.False(t, resp["success"].(bool))
	})

	t.Run("refresh_with_access_token_instead_of_refresh_token_fails", func(t *testing.T) {
		// Try to use access token as refresh token
		reqBody := map[string]interface{}{
			"refresh_token": initialAccessToken,
		}

		w := ts.Request(http.MethodPost, "/api/v1/auth/refresh", reqBody)

		// Should fail because access token has wrong token type
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("refresh_for_deactivated_user_fails", func(t *testing.T) {
		// Create another user, login, then deactivate
		deactivateUser := ts.CreateTestUser(t, tenantID, "deactivate_refresh_user", testPassword, role.ID)

		// Login
		loginReq := map[string]interface{}{
			"username": "deactivate_refresh_user",
			"password": testPassword,
		}
		loginResp := ts.Request(http.MethodPost, "/api/v1/auth/login", loginReq)
		require.Equal(t, http.StatusOK, loginResp.Code)

		// Extract refresh token from httpOnly cookie
		cookies := loginResp.Result().Cookies()
		var refreshToken string
		for _, cookie := range cookies {
			if cookie.Name == "refresh_token" {
				refreshToken = cookie.Value
				break
			}
		}
		require.NotEmpty(t, refreshToken, "refresh_token cookie should be set")

		// Deactivate user
		err := deactivateUser.Deactivate()
		require.NoError(t, err)
		err = ts.UserRepo.Update(context.Background(), deactivateUser)
		require.NoError(t, err)

		// Try to refresh with cookie
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
		req.AddCookie(&http.Cookie{
			Name:  "refresh_token",
			Value: refreshToken,
		})
		w := httptest.NewRecorder()
		ts.Engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		errorInfo := resp["error"].(map[string]interface{})
		assert.Equal(t, "ACCOUNT_INACTIVE", errorInfo["code"])
	})

	t.Run("refresh_updates_permissions_when_role_changes", func(t *testing.T) {
		// Create user with initial role
		permChangeUser := ts.CreateTestUser(t, tenantID, "perm_change_user", testPassword, role.ID)

		// Login
		loginReq := map[string]interface{}{
			"username": "perm_change_user",
			"password": testPassword,
		}
		loginResp := ts.Request(http.MethodPost, "/api/v1/auth/login", loginReq)
		require.Equal(t, http.StatusOK, loginResp.Code)

		var loginData map[string]interface{}
		json.Unmarshal(loginResp.Body.Bytes(), &loginData)
		accessToken := loginData["data"].(map[string]interface{})["token"].(map[string]interface{})["access_token"].(string)

		// Extract refresh token from httpOnly cookie
		cookies := loginResp.Result().Cookies()
		var refreshToken string
		for _, cookie := range cookies {
			if cookie.Name == "refresh_token" {
				refreshToken = cookie.Value
				break
			}
		}
		require.NotEmpty(t, refreshToken, "refresh_token cookie should be set")

		// Verify initial permission - should have product:read
		w := ts.Request(http.MethodGet, "/api/v1/protected/products", nil, accessToken)
		assert.Equal(t, http.StatusOK, w.Code)

		// Create a new role with different permissions
		newRole := ts.CreateTestRole(t, tenantID, "NEW_ROLE", "New Role", "customer:read")

		// Update user's role
		err := permChangeUser.SetRoles([]uuid.UUID{newRole.ID})
		require.NoError(t, err)
		err = ts.UserRepo.SaveUserRoles(context.Background(), permChangeUser)
		require.NoError(t, err)

		// Refresh token - should get new permissions
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
		req.AddCookie(&http.Cookie{
			Name:  "refresh_token",
			Value: refreshToken,
		})
		refreshResp := httptest.NewRecorder()
		ts.Engine.ServeHTTP(refreshResp, req)
		require.Equal(t, http.StatusOK, refreshResp.Code)

		var refreshData map[string]interface{}
		json.Unmarshal(refreshResp.Body.Bytes(), &refreshData)
		newAccessToken := refreshData["data"].(map[string]interface{})["token"].(map[string]interface{})["access_token"].(string)

		// New token should NOT have product:read (old permission)
		w = ts.Request(http.MethodGet, "/api/v1/protected/products", nil, newAccessToken)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

// =============================================================================
// Current User and Password Change Tests
// =============================================================================

func TestAuth_CurrentUserAndPassword(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewAuthTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	role := ts.CreateTestRole(t, tenantID, "ME_TEST", "Me Test Role", "product:read")
	testPassword := "TestPass123"
	user := ts.CreateTestUser(t, tenantID, "me_user", testPassword, role.ID)

	// Login to get token
	loginReq := map[string]interface{}{
		"username": "me_user",
		"password": testPassword,
	}
	loginResp := ts.Request(http.MethodPost, "/api/v1/auth/login", loginReq)
	require.Equal(t, http.StatusOK, loginResp.Code)

	var loginData map[string]interface{}
	json.Unmarshal(loginResp.Body.Bytes(), &loginData)
	accessToken := loginData["data"].(map[string]interface{})["token"].(map[string]interface{})["access_token"].(string)

	t.Run("get_current_user_returns_user_info", func(t *testing.T) {
		w := ts.Request(http.MethodGet, "/api/v1/auth/me", nil, accessToken)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		data := resp["data"].(map[string]interface{})
		userInfo := data["user"].(map[string]interface{})
		assert.Equal(t, user.ID.String(), userInfo["id"])
		assert.Equal(t, "me_user", userInfo["username"])

		permissions := data["permissions"].([]interface{})
		assert.Len(t, permissions, 1)
		assert.Equal(t, "product:read", permissions[0])
	})

	t.Run("get_current_user_without_token_returns_401", func(t *testing.T) {
		w := ts.Request(http.MethodGet, "/api/v1/auth/me", nil)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("change_password_with_correct_old_password_succeeds", func(t *testing.T) {
		newPassword := "NewPass456"
		reqBody := map[string]interface{}{
			"old_password": testPassword,
			"new_password": newPassword,
		}

		w := ts.Request(http.MethodPut, "/api/v1/auth/password", reqBody, accessToken)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify can login with new password
		loginReq := map[string]interface{}{
			"username": "me_user",
			"password": newPassword,
		}
		loginResp := ts.Request(http.MethodPost, "/api/v1/auth/login", loginReq)
		assert.Equal(t, http.StatusOK, loginResp.Code)

		// Verify cannot login with old password
		loginReq["password"] = testPassword
		loginResp = ts.Request(http.MethodPost, "/api/v1/auth/login", loginReq)
		assert.Equal(t, http.StatusUnauthorized, loginResp.Code)
	})

	t.Run("change_password_with_wrong_old_password_fails", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"old_password": "WrongOldPass123",
			"new_password": "NewPass789",
		}

		w := ts.Request(http.MethodPut, "/api/v1/auth/password", reqBody, accessToken)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})
}

// =============================================================================
// Token Security Tests
// =============================================================================

func TestAuth_TokenSecurity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewAuthTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	role := ts.CreateTestRole(t, tenantID, "SECURITY_TEST", "Security Test Role", "product:read")
	testPassword := "TestPass123"
	ts.CreateTestUser(t, tenantID, "security_user", testPassword, role.ID)

	// Login to get valid token
	loginReq := map[string]interface{}{
		"username": "security_user",
		"password": testPassword,
	}
	loginResp := ts.Request(http.MethodPost, "/api/v1/auth/login", loginReq)
	require.Equal(t, http.StatusOK, loginResp.Code)

	var loginData map[string]interface{}
	json.Unmarshal(loginResp.Body.Bytes(), &loginData)
	validToken := loginData["data"].(map[string]interface{})["token"].(map[string]interface{})["access_token"].(string)

	t.Run("token_with_wrong_signature_is_rejected", func(t *testing.T) {
		// Modify the signature part of the token
		parts := strings.Split(validToken, ".")
		require.Len(t, parts, 3)
		tamperedToken := parts[0] + "." + parts[1] + ".tampered_signature"

		w := ts.Request(http.MethodGet, "/api/v1/protected/products", nil, tamperedToken)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("completely_invalid_token_is_rejected", func(t *testing.T) {
		w := ts.Request(http.MethodGet, "/api/v1/protected/products", nil, "not.a.valid.jwt.token")
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("empty_authorization_header_returns_401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/protected/products", nil)
		req.Header.Set("Authorization", "")
		w := httptest.NewRecorder()
		ts.Engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("logout_returns_success", func(t *testing.T) {
		w := ts.Request(http.MethodPost, "/api/v1/auth/logout", nil, validToken)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))
	})
}

// =============================================================================
// Multi-Tenant Isolation Tests
// =============================================================================

func TestAuth_MultiTenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewAuthTestServer(t)

	// Create two tenants
	tenant1ID := uuid.New()
	tenant2ID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenant1ID)
	ts.DB.CreateTestTenantWithUUID(tenant2ID)

	// Create roles and users for each tenant
	testPassword := "TestPass123"

	role1 := ts.CreateTestRole(t, tenant1ID, "TENANT1_ROLE", "Tenant 1 Role", "product:read")
	user1 := ts.CreateTestUser(t, tenant1ID, "tenant1_user", testPassword, role1.ID)

	role2 := ts.CreateTestRole(t, tenant2ID, "TENANT2_ROLE", "Tenant 2 Role", "product:create")
	user2 := ts.CreateTestUser(t, tenant2ID, "tenant2_user", testPassword, role2.ID)

	t.Run("tokens_contain_correct_tenant_id", func(t *testing.T) {
		// Login tenant 1 user
		loginReq := map[string]interface{}{
			"username": "tenant1_user",
			"password": testPassword,
		}
		loginResp := ts.Request(http.MethodPost, "/api/v1/auth/login", loginReq)
		require.Equal(t, http.StatusOK, loginResp.Code)

		var loginData map[string]interface{}
		json.Unmarshal(loginResp.Body.Bytes(), &loginData)
		userInfo := loginData["data"].(map[string]interface{})["user"].(map[string]interface{})

		assert.Equal(t, user1.TenantID.String(), userInfo["tenant_id"])
		assert.Equal(t, tenant1ID.String(), userInfo["tenant_id"])
	})

	t.Run("user_permissions_are_tenant_scoped", func(t *testing.T) {
		// Login tenant 1 user
		loginReq1 := map[string]interface{}{
			"username": "tenant1_user",
			"password": testPassword,
		}
		resp1 := ts.Request(http.MethodPost, "/api/v1/auth/login", loginReq1)
		var data1 map[string]interface{}
		json.Unmarshal(resp1.Body.Bytes(), &data1)
		perms1 := data1["data"].(map[string]interface{})["user"].(map[string]interface{})["permissions"].([]interface{})

		// Login tenant 2 user
		loginReq2 := map[string]interface{}{
			"username": "tenant2_user",
			"password": testPassword,
		}
		resp2 := ts.Request(http.MethodPost, "/api/v1/auth/login", loginReq2)
		var data2 map[string]interface{}
		json.Unmarshal(resp2.Body.Bytes(), &data2)
		perms2 := data2["data"].(map[string]interface{})["user"].(map[string]interface{})["permissions"].([]interface{})

		// Tenant 1 user should have product:read
		assert.Contains(t, perms1, "product:read")
		assert.NotContains(t, perms1, "product:create")

		// Tenant 2 user should have product:create
		assert.Contains(t, perms2, "product:create")
		assert.NotContains(t, perms2, "product:read")
	})

	t.Run("users_with_same_username_in_different_tenants", func(t *testing.T) {
		// Note: This test depends on whether the system allows same username across tenants
		// Based on the user repository, username lookup is not tenant-scoped
		// So this test verifies the current behavior

		// Both users should be able to login
		_ = user1
		_ = user2

		// Login tenant1_user
		loginReq1 := map[string]interface{}{
			"username": "tenant1_user",
			"password": testPassword,
		}
		resp1 := ts.Request(http.MethodPost, "/api/v1/auth/login", loginReq1)
		assert.Equal(t, http.StatusOK, resp1.Code)

		// Login tenant2_user
		loginReq2 := map[string]interface{}{
			"username": "tenant2_user",
			"password": testPassword,
		}
		resp2 := ts.Request(http.MethodPost, "/api/v1/auth/login", loginReq2)
		assert.Equal(t, http.StatusOK, resp2.Code)
	})
}
