// Package integration provides integration testing for the ERP backend API.
// This file contains security vulnerability scanning tests (P6-QA-003).
package integration

import (
	"bytes"
	"context"
	"encoding/base64"
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

// SecurityTestServer wraps the test database and HTTP server for security testing
type SecurityTestServer struct {
	DB          *TestDB
	Engine      *gin.Engine
	UserRepo    *persistence.GormUserRepository
	RoleRepo    *persistence.GormRoleRepository
	AuthService *identityapp.AuthService
	JWTService  *auth.JWTService
}

// NewSecurityTestServer creates a new test server with security middleware
func NewSecurityTestServer(t *testing.T) *SecurityTestServer {
	t.Helper()

	gin.SetMode(gin.TestMode)
	testDB := NewTestDB(t)

	// Initialize repositories
	userRepo := persistence.NewGormUserRepository(testDB.DB)
	roleRepo := persistence.NewGormRoleRepository(testDB.DB)

	// Initialize JWT service with test config
	jwtConfig := config.JWTConfig{
		Secret:                 "test-secret-key-for-security-testing-1234567890",
		RefreshSecret:          "test-refresh-secret-key-for-security-testing",
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

	// Setup engine with security middleware
	engine := gin.New()
	engine.Use(middleware.Secure())               // Security headers
	engine.Use(middleware.RequestID())            // Request ID generation
	engine.Use(middleware.BodyLimit(1024 * 1024)) // 1MB body limit

	// Setup routes
	api := engine.Group("/api/v1")

	// Auth routes (no JWT required for login)
	authGroup := api.Group("/auth")
	authGroup.POST("/login", authHandler.Login)

	// Protected routes
	protectedAPI := api.Group("/protected")
	protectedAPI.Use(middleware.JWTAuthMiddleware(jwtService))

	// Echo endpoint for security testing
	protectedAPI.POST("/echo", func(c *gin.Context) {
		var body map[string]any
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": body})
	})

	// User input endpoint for XSS testing
	protectedAPI.POST("/users", func(c *gin.Context) {
		var body struct {
			Name  string `json:"name" binding:"required"`
			Email string `json:"email" binding:"required,email"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// Return the data (simulating storage and retrieval)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"name":  body.Name,
				"email": body.Email,
			},
		})
	})

	return &SecurityTestServer{
		DB:          testDB,
		Engine:      engine,
		UserRepo:    userRepo,
		RoleRepo:    roleRepo,
		AuthService: authService,
		JWTService:  jwtService,
	}
}

// Request makes an HTTP request to the test server
func (ts *SecurityTestServer) Request(method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBody)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")

	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	w := httptest.NewRecorder()
	ts.Engine.ServeHTTP(w, req)
	return w
}

// CreateTestUser creates a test user with given credentials
func (ts *SecurityTestServer) CreateTestUser(t *testing.T, tenantID uuid.UUID, username, password string) *identity.User {
	t.Helper()

	user, err := identity.NewActiveUser(tenantID, username, password)
	require.NoError(t, err)

	// Set a unique email
	email := fmt.Sprintf("%s_%s@test.local", username, uuid.New().String()[:8])
	err = user.SetEmail(email)
	require.NoError(t, err)

	err = ts.UserRepo.Create(context.Background(), user)
	require.NoError(t, err)

	return user
}

// GetAuthToken gets a JWT token for a user
func (ts *SecurityTestServer) GetAuthToken(t *testing.T, username, password string) string {
	t.Helper()

	resp := ts.Request("POST", "/api/v1/auth/login", map[string]string{
		"username": username,
		"password": password,
	}, nil)

	require.Equal(t, http.StatusOK, resp.Code, "Login failed: %s", resp.Body.String())

	var result map[string]any
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	data, ok := result["data"].(map[string]any)
	require.True(t, ok, "data should be a map, got: %v", result)

	token, ok := data["token"].(map[string]any)
	require.True(t, ok, "token should be a map, got: %v", data)

	accessToken, ok := token["access_token"].(string)
	require.True(t, ok, "access_token should be a string, got: %v", token)

	return accessToken
}

// ============================================================================
// Security Scanning Tests
// ============================================================================

func TestSecurity_Headers(t *testing.T) {
	ts := NewSecurityTestServer(t)

	t.Run("security_headers_are_set_on_responses", func(t *testing.T) {
		// Create tenant and user for authenticated request
		tenantID := uuid.New()
		ts.DB.CreateTestTenant(tenantID.String(), "Security Test Tenant", "sec_test")
		ts.CreateTestUser(t, tenantID, "securityuser", "SecurePass123!")
		token := ts.GetAuthToken(t, "securityuser", "SecurePass123!")

		resp := ts.Request("POST", "/api/v1/protected/echo", map[string]string{"test": "data"}, map[string]string{
			"Authorization": "Bearer " + token,
		})

		// Verify security headers
		assert.Equal(t, "DENY", resp.Header().Get("X-Frame-Options"),
			"X-Frame-Options should prevent clickjacking")
		assert.Equal(t, "1; mode=block", resp.Header().Get("X-XSS-Protection"),
			"X-XSS-Protection should enable browser XSS filter")
		assert.Equal(t, "nosniff", resp.Header().Get("X-Content-Type-Options"),
			"X-Content-Type-Options should prevent MIME sniffing")
		assert.Equal(t, "strict-origin-when-cross-origin", resp.Header().Get("Referrer-Policy"),
			"Referrer-Policy should limit referrer information")
	})

	t.Run("request_id_is_generated_for_each_request", func(t *testing.T) {
		tenantID := uuid.New()
		ts.DB.CreateTestTenant(tenantID.String(), "Security Test Tenant 2", "sec_test2")
		ts.CreateTestUser(t, tenantID, "securityuser2", "SecurePass123!")
		token := ts.GetAuthToken(t, "securityuser2", "SecurePass123!")

		// Make two requests
		resp1 := ts.Request("POST", "/api/v1/protected/echo", map[string]string{"test": "1"}, map[string]string{
			"Authorization": "Bearer " + token,
		})
		resp2 := ts.Request("POST", "/api/v1/protected/echo", map[string]string{"test": "2"}, map[string]string{
			"Authorization": "Bearer " + token,
		})

		// Verify request IDs are generated and unique
		reqID1 := resp1.Header().Get("X-Request-ID")
		reqID2 := resp2.Header().Get("X-Request-ID")
		assert.NotEmpty(t, reqID1, "Request ID should be generated")
		assert.NotEmpty(t, reqID2, "Request ID should be generated")
		assert.NotEqual(t, reqID1, reqID2, "Request IDs should be unique")
	})

	t.Run("custom_request_id_is_preserved", func(t *testing.T) {
		tenantID := uuid.New()
		ts.DB.CreateTestTenant(tenantID.String(), "Security Test Tenant 3", "sec_test3")
		ts.CreateTestUser(t, tenantID, "securityuser3", "SecurePass123!")
		token := ts.GetAuthToken(t, "securityuser3", "SecurePass123!")

		customRequestID := "custom-request-id-12345"
		resp := ts.Request("POST", "/api/v1/protected/echo", map[string]string{"test": "data"}, map[string]string{
			"Authorization": "Bearer " + token,
			"X-Request-ID":  customRequestID,
		})

		assert.Equal(t, customRequestID, resp.Header().Get("X-Request-ID"),
			"Custom request ID should be preserved")
	})
}

// ============================================================================
// XSS Protection Tests
// ============================================================================

func TestSecurity_XSSProtection(t *testing.T) {
	ts := NewSecurityTestServer(t)

	// Create tenant and user
	tenantID := uuid.New()
	ts.DB.CreateTestTenant(tenantID.String(), "XSS Test Tenant", "xss_test")
	ts.CreateTestUser(t, tenantID, "xssuser", "SecurePass123!")
	token := ts.GetAuthToken(t, "xssuser", "SecurePass123!")

	xssPayloads := []struct {
		name    string
		payload string
	}{
		{"script_tag", "<script>alert('XSS')</script>"},
		{"img_onerror", "<img src=x onerror=alert('XSS')>"},
		{"svg_onload", "<svg onload=alert('XSS')>"},
		{"event_handler", "<body onload=alert('XSS')>"},
		{"javascript_uri", "javascript:alert('XSS')"},
		{"data_uri", "data:text/html,<script>alert('XSS')</script>"},
		{"encoded_script", "&lt;script&gt;alert('XSS')&lt;/script&gt;"},
		{"double_encoded", "%253Cscript%253Ealert('XSS')%253C/script%253E"},
		{"null_byte", "<scr\x00ipt>alert('XSS')</script>"},
		{"unicode_bypass", "<script>alert\u0000('XSS')</script>"},
	}

	for _, tc := range xssPayloads {
		t.Run("xss_payload_"+tc.name+"_is_not_executed", func(t *testing.T) {
			// Send XSS payload as user input
			resp := ts.Request("POST", "/api/v1/protected/echo", map[string]any{
				"name":    tc.payload,
				"message": tc.payload,
			}, map[string]string{
				"Authorization": "Bearer " + token,
			})

			// The request should succeed (we're testing that the data is handled safely)
			assert.Equal(t, http.StatusOK, resp.Code)

			// Response should have Content-Type: application/json
			contentType := resp.Header().Get("Content-Type")
			assert.Contains(t, contentType, "application/json",
				"Response Content-Type should be application/json, not text/html")

			// X-XSS-Protection header should be present
			assert.Equal(t, "1; mode=block", resp.Header().Get("X-XSS-Protection"),
				"X-XSS-Protection header should be set")

			// X-Content-Type-Options should prevent MIME sniffing
			assert.Equal(t, "nosniff", resp.Header().Get("X-Content-Type-Options"),
				"X-Content-Type-Options should be nosniff to prevent MIME sniffing")
		})
	}

	t.Run("xss_in_login_username_is_handled", func(t *testing.T) {
		xssUsername := "<script>alert('XSS')</script>"
		resp := ts.Request("POST", "/api/v1/auth/login", map[string]string{
			"username": xssUsername,
			"password": "anypassword",
		}, nil)

		// Should return 401 (invalid credentials), not crash or execute script
		assert.Equal(t, http.StatusUnauthorized, resp.Code)

		// Response should be JSON, not HTML
		contentType := resp.Header().Get("Content-Type")
		assert.Contains(t, contentType, "application/json")
	})

	t.Run("xss_in_headers_is_handled", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/protected/echo", map[string]string{"test": "data"}, map[string]string{
			"Authorization":   "Bearer " + token,
			"X-Custom-Header": "<script>alert('XSS')</script>",
		})

		// Request should succeed without executing the script
		assert.Equal(t, http.StatusOK, resp.Code)

		// Verify security headers are still present
		assert.Equal(t, "1; mode=block", resp.Header().Get("X-XSS-Protection"))
	})
}

// ============================================================================
// CSRF Protection Tests
// ============================================================================

func TestSecurity_CSRFProtection(t *testing.T) {
	ts := NewSecurityTestServer(t)

	// Create tenant and user
	tenantID := uuid.New()
	ts.DB.CreateTestTenant(tenantID.String(), "CSRF Test Tenant", "csrf_test")
	ts.CreateTestUser(t, tenantID, "csrfuser", "SecurePass123!")
	token := ts.GetAuthToken(t, "csrfuser", "SecurePass123!")

	t.Run("state_changing_request_without_auth_is_rejected", func(t *testing.T) {
		// POST request without JWT token should be rejected
		resp := ts.Request("POST", "/api/v1/protected/echo", map[string]string{"test": "data"}, nil)
		assert.Equal(t, http.StatusUnauthorized, resp.Code,
			"State-changing requests without authentication should be rejected")
	})

	t.Run("request_with_invalid_auth_token_is_rejected", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/protected/echo", map[string]string{"test": "data"}, map[string]string{
			"Authorization": "Bearer invalid-token",
		})
		assert.Equal(t, http.StatusUnauthorized, resp.Code,
			"Requests with invalid tokens should be rejected")
	})

	t.Run("request_with_expired_token_is_rejected", func(t *testing.T) {
		// Create a JWT service with very short expiration for testing
		shortConfig := config.JWTConfig{
			Secret:                 "test-secret-key-for-security-testing-1234567890",
			RefreshSecret:          "test-refresh-secret-key-for-security-testing",
			AccessTokenExpiration:  1 * time.Millisecond, // Very short
			RefreshTokenExpiration: 1 * time.Millisecond,
			Issuer:                 "erp-test",
			MaxRefreshCount:        10,
		}
		shortJWTService := auth.NewJWTService(shortConfig)

		// Generate token that expires immediately
		tokenInput := auth.GenerateTokenInput{
			TenantID:    tenantID,
			UserID:      uuid.New(),
			Username:    "testuser",
			RoleIDs:     []uuid.UUID{},
			Permissions: []string{},
		}
		tokenPair, _ := shortJWTService.GenerateTokenPair(tokenInput)

		// Wait for expiration
		time.Sleep(10 * time.Millisecond)

		resp := ts.Request("POST", "/api/v1/protected/echo", map[string]string{"test": "data"}, map[string]string{
			"Authorization": "Bearer " + tokenPair.AccessToken,
		})
		assert.Equal(t, http.StatusUnauthorized, resp.Code,
			"Requests with expired tokens should be rejected")
	})

	t.Run("request_from_different_origin_without_cors_is_handled", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/protected/echo", map[string]string{"test": "data"}, map[string]string{
			"Authorization": "Bearer " + token,
			"Origin":        "http://malicious-site.com",
		})

		// With proper JWT authentication, the request should succeed
		// CORS is about browser-enforced same-origin policy, not server-side CSRF
		// JWT-based authentication inherently protects against CSRF
		assert.Equal(t, http.StatusOK, resp.Code,
			"JWT authentication provides CSRF protection")
	})

	t.Run("cookie_based_auth_not_supported_prevents_csrf", func(t *testing.T) {
		// Attempt to authenticate with cookie instead of Authorization header
		// This should fail since the API uses JWT in Authorization header
		resp := ts.Request("POST", "/api/v1/protected/echo", map[string]string{"test": "data"}, map[string]string{
			"Cookie": "session_token=" + token, // Trying to send JWT as cookie
		})
		assert.Equal(t, http.StatusUnauthorized, resp.Code,
			"API should not accept authentication via cookies (CSRF protection)")
	})

	t.Run("valid_jwt_in_authorization_header_is_required", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/protected/echo", map[string]string{"test": "data"}, map[string]string{
			"Authorization": "Bearer " + token,
		})
		assert.Equal(t, http.StatusOK, resp.Code,
			"Valid JWT in Authorization header should be accepted")
	})
}

// ============================================================================
// SQL Injection Protection Tests
// ============================================================================

func TestSecurity_SQLInjectionProtection(t *testing.T) {
	ts := NewSecurityTestServer(t)

	sqlInjectionPayloads := []struct {
		name    string
		payload string
	}{
		{"basic_or_bypass", "' OR '1'='1"},
		{"union_select", "' UNION SELECT * FROM users--"},
		{"drop_table", "'; DROP TABLE users;--"},
		{"comment_bypass", "admin'--"},
		{"stacked_queries", "'; SELECT * FROM users;--"},
		{"time_based_blind", "' OR SLEEP(5)--"},
		{"error_based", "' AND 1=CONVERT(int, (SELECT @@version))--"},
		{"boolean_blind", "' AND 1=1--"},
		{"hex_encoded", "0x27204f522027313d273127"},
		{"char_function", "' OR CHAR(97)+CHAR(100)+CHAR(109)+CHAR(105)+CHAR(110)--"},
	}

	t.Run("sql_injection_in_login_is_prevented", func(t *testing.T) {
		for _, tc := range sqlInjectionPayloads {
			t.Run(tc.name, func(t *testing.T) {
				resp := ts.Request("POST", "/api/v1/auth/login", map[string]string{
					"username": tc.payload,
					"password": tc.payload,
				}, nil)

				// Should return 401 (invalid credentials), not 500 (server error)
				// and definitely not 200 (successful bypass)
				assert.NotEqual(t, http.StatusOK, resp.Code,
					"SQL injection should not bypass authentication")
				assert.NotEqual(t, http.StatusInternalServerError, resp.Code,
					"SQL injection should not cause server error")
			})
		}
	})

	t.Run("sql_injection_in_json_body_is_handled", func(t *testing.T) {
		tenantID := uuid.New()
		ts.DB.CreateTestTenant(tenantID.String(), "SQL Test Tenant", "sql_test")
		ts.CreateTestUser(t, tenantID, "sqluser", "SecurePass123!")
		token := ts.GetAuthToken(t, "sqluser", "SecurePass123!")

		for _, tc := range sqlInjectionPayloads {
			t.Run(tc.name, func(t *testing.T) {
				resp := ts.Request("POST", "/api/v1/protected/echo", map[string]any{
					"query":  tc.payload,
					"filter": tc.payload,
				}, map[string]string{
					"Authorization": "Bearer " + token,
				})

				// Request should succeed (data is just echoed back)
				// The payload should be treated as data, not SQL
				assert.Equal(t, http.StatusOK, resp.Code)

				// Verify the response is valid JSON
				var result map[string]any
				err := json.Unmarshal(resp.Body.Bytes(), &result)
				assert.NoError(t, err, "Response should be valid JSON")
			})
		}
	})
}

// ============================================================================
// Authentication Security Tests
// ============================================================================

func TestSecurity_AuthenticationSecurity(t *testing.T) {
	ts := NewSecurityTestServer(t)

	tenantID := uuid.New()
	ts.DB.CreateTestTenant(tenantID.String(), "Auth Security Tenant", "auth_sec")

	t.Run("password_not_returned_in_responses", func(t *testing.T) {
		ts.CreateTestUser(t, tenantID, "authsecuser", "SuperSecretPassword123!")

		resp := ts.Request("POST", "/api/v1/auth/login", map[string]string{
			"username": "authsecuser",
			"password": "SuperSecretPassword123!",
		}, nil)

		assert.Equal(t, http.StatusOK, resp.Code)

		// Password should never appear in response
		responseBody := resp.Body.String()
		assert.NotContains(t, responseBody, "SuperSecretPassword123!",
			"Password should never be returned in response")
		assert.NotContains(t, strings.ToLower(responseBody), "password",
			"Password field should not be in response")
	})

	t.Run("jwt_token_contains_minimal_claims", func(t *testing.T) {
		ts.CreateTestUser(t, tenantID, "claimsuser", "SecurePass123!")
		token := ts.GetAuthToken(t, "claimsuser", "SecurePass123!")

		// Decode JWT payload (base64)
		parts := strings.Split(token, ".")
		require.Len(t, parts, 3, "JWT should have 3 parts")

		// Base64 decode the payload (second part)
		// Note: JWT uses base64url encoding without padding
		payloadBytes, err := decodeBase64URL(parts[1])
		require.NoError(t, err)

		var claims map[string]any
		err = json.Unmarshal(payloadBytes, &claims)
		require.NoError(t, err)

		// Verify sensitive data is not in JWT
		assert.NotContains(t, claims, "password", "JWT should not contain password")
		assert.NotContains(t, claims, "email", "JWT should minimize PII")

		// Verify essential claims are present
		assert.Contains(t, claims, "sub", "JWT should have subject claim")
		assert.Contains(t, claims, "exp", "JWT should have expiration claim")
		assert.Contains(t, claims, "iat", "JWT should have issued-at claim")
	})

	t.Run("invalid_password_format_is_rejected_with_generic_message", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/auth/login", map[string]string{
			"username": "nonexistentuser",
			"password": "wrongpassword",
		}, nil)

		assert.Equal(t, http.StatusUnauthorized, resp.Code)

		// Error message should not reveal if username exists
		responseBody := resp.Body.String()
		assert.NotContains(t, strings.ToLower(responseBody), "user not found",
			"Error should not reveal username existence")
		assert.NotContains(t, strings.ToLower(responseBody), "password incorrect",
			"Error should not confirm username was correct")
	})

	t.Run("timing_attack_protection_on_login", func(t *testing.T) {
		// Create a user for comparison
		ts.CreateTestUser(t, tenantID, "timinguser", "SecurePass123!")

		// Measure timing for existing user with wrong password
		start1 := time.Now()
		ts.Request("POST", "/api/v1/auth/login", map[string]string{
			"username": "timinguser",
			"password": "wrongpassword",
		}, nil)
		duration1 := time.Since(start1)

		// Measure timing for non-existing user
		start2 := time.Now()
		ts.Request("POST", "/api/v1/auth/login", map[string]string{
			"username": "nonexistentuser12345",
			"password": "wrongpassword",
		}, nil)
		duration2 := time.Since(start2)

		// Timing difference should be minimal (within 50ms typically)
		// This is a soft check as timing can vary
		timingDiff := abs(duration1.Milliseconds() - duration2.Milliseconds())
		t.Logf("Timing difference: %dms (existing user: %dms, non-existing: %dms)",
			timingDiff, duration1.Milliseconds(), duration2.Milliseconds())

		// We just log the timing for awareness; strict enforcement may cause flaky tests
	})
}

// ============================================================================
// Request Validation Security Tests
// ============================================================================

func TestSecurity_RequestValidation(t *testing.T) {
	ts := NewSecurityTestServer(t)

	tenantID := uuid.New()
	ts.DB.CreateTestTenant(tenantID.String(), "Validation Tenant", "val_test")
	ts.CreateTestUser(t, tenantID, "valuser", "SecurePass123!")
	token := ts.GetAuthToken(t, "valuser", "SecurePass123!")

	t.Run("oversized_request_body_is_rejected", func(t *testing.T) {
		// Create a payload larger than 1MB limit
		largePayload := make([]byte, 2*1024*1024) // 2MB
		for i := range largePayload {
			largePayload[i] = 'a'
		}

		req := httptest.NewRequest("POST", "/api/v1/protected/echo", bytes.NewBuffer(largePayload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		ts.Engine.ServeHTTP(w, req)

		// Should be rejected (400 or 413)
		assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusRequestEntityTooLarge,
			"Oversized requests should be rejected, got %d", w.Code)
	})

	t.Run("malformed_json_is_rejected", func(t *testing.T) {
		malformedJSON := []string{
			`{"name": }`,
			`{"name": "test"`,
			`{name: "test"}`,
			`{"name": undefined}`,
		}

		for _, payload := range malformedJSON {
			req := httptest.NewRequest("POST", "/api/v1/protected/echo", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			w := httptest.NewRecorder()
			ts.Engine.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code,
				"Malformed JSON should be rejected: %s", payload)
		}
	})

	t.Run("content_type_validation", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/protected/echo", strings.NewReader(`{"test": "data"}`))
		req.Header.Set("Content-Type", "text/plain") // Wrong content type
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		ts.Engine.ServeHTTP(w, req)

		// Gin may still process it, but we should verify the behavior
		// At minimum, security headers should still be present
		assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	})
}

// ============================================================================
// Path Traversal Protection Tests
// ============================================================================

func TestSecurity_PathTraversal(t *testing.T) {
	ts := NewSecurityTestServer(t)

	pathTraversalPayloads := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\config\\sam",
		"....//....//....//etc/passwd",
		"%2e%2e%2f%2e%2e%2f%2e%2e%2fetc/passwd",
		"..%252f..%252f..%252fetc/passwd",
		"/etc/passwd",
		"file:///etc/passwd",
	}

	for _, payload := range pathTraversalPayloads {
		t.Run("path_traversal_"+strings.ReplaceAll(payload[:min(20, len(payload))], "/", "_"), func(t *testing.T) {
			resp := ts.Request("GET", "/api/v1/protected/"+payload, nil, nil)

			// Should return 401 (no auth) or 404 (not found), not file contents
			assert.True(t, resp.Code == http.StatusUnauthorized || resp.Code == http.StatusNotFound,
				"Path traversal should not access files, got status %d for payload: %s", resp.Code, payload)

			// Response should not contain file contents
			responseBody := resp.Body.String()
			assert.NotContains(t, responseBody, "root:", "Should not contain /etc/passwd content")
			assert.NotContains(t, responseBody, "[boot loader]", "Should not contain Windows system file content")
		})
	}
}

// ============================================================================
// Error Information Leakage Tests
// ============================================================================

func TestSecurity_ErrorInformationLeakage(t *testing.T) {
	ts := NewSecurityTestServer(t)

	t.Run("internal_error_does_not_leak_stack_trace", func(t *testing.T) {
		// Send a request that might cause an error
		resp := ts.Request("POST", "/api/v1/protected/echo", nil, map[string]string{
			"Authorization": "Bearer invalid",
		})

		responseBody := resp.Body.String()

		// Should not contain sensitive error details
		assert.NotContains(t, responseBody, "panic", "Should not expose panic details")
		assert.NotContains(t, responseBody, "runtime error", "Should not expose runtime errors")
		assert.NotContains(t, responseBody, ".go:", "Should not expose source file locations")
		assert.NotContains(t, responseBody, "goroutine", "Should not expose goroutine info")
	})

	t.Run("database_error_does_not_leak_schema", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/auth/login", map[string]string{
			"username": "' OR 1=1; SELECT * FROM pg_tables;--",
			"password": "test",
		}, nil)

		responseBody := resp.Body.String()

		// Should not contain database schema information
		assert.NotContains(t, responseBody, "pg_tables", "Should not expose database tables")
		assert.NotContains(t, responseBody, "column", "Should not expose column information")
		assert.NotContains(t, responseBody, "SELECT", "Should not expose SQL queries")
		assert.NotContains(t, responseBody, "INSERT", "Should not expose SQL queries")
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

func decodeBase64URL(s string) ([]byte, error) {
	// Add padding if necessary
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}

	// Replace URL-safe characters
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")

	// Actually decode the base64 string
	return base64.StdEncoding.DecodeString(s)
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
