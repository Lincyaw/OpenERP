package middleware

import (
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
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestJWTService() *auth.JWTService {
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

func newTestTokenPair(jwtService *auth.JWTService) (*auth.TokenPair, auth.GenerateTokenInput) {
	input := auth.GenerateTokenInput{
		TenantID:    uuid.New(),
		UserID:      uuid.New(),
		Username:    "testuser",
		RoleIDs:     []uuid.UUID{uuid.New()},
		Permissions: []string{"product:read", "product:create"},
	}
	pair, _ := jwtService.GenerateTokenPair(input)
	return pair, input
}

func TestJWTAuthMiddleware_ValidToken(t *testing.T) {
	jwtService := newTestJWTService()
	pair, input := newTestTokenPair(jwtService)

	router := gin.New()
	router.Use(JWTAuthMiddleware(jwtService))
	router.GET("/test", func(c *gin.Context) {
		claims := GetJWTClaims(c)
		assert.NotNil(t, claims)
		assert.Equal(t, input.UserID.String(), claims.UserID)
		assert.Equal(t, input.TenantID.String(), claims.TenantID)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestJWTAuthMiddleware_MissingHeader(t *testing.T) {
	jwtService := newTestJWTService()

	router := gin.New()
	router.Use(JWTAuthMiddleware(jwtService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuthMiddleware_InvalidHeaderFormat(t *testing.T) {
	jwtService := newTestJWTService()

	router := gin.New()
	router.Use(JWTAuthMiddleware(jwtService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "InvalidFormat token123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuthMiddleware_EmptyToken(t *testing.T) {
	jwtService := newTestJWTService()

	router := gin.New()
	router.Use(JWTAuthMiddleware(jwtService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuthMiddleware_InvalidToken(t *testing.T) {
	jwtService := newTestJWTService()

	router := gin.New()
	router.Use(JWTAuthMiddleware(jwtService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuthMiddleware_ExpiredToken(t *testing.T) {
	cfg := config.JWTConfig{
		Secret:                 "test-secret-key-at-least-32-chars",
		AccessTokenExpiration:  -1 * time.Hour, // Already expired
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test-issuer",
	}
	jwtService := auth.NewJWTService(cfg)
	pair, _ := newTestTokenPair(jwtService)

	router := gin.New()
	router.Use(JWTAuthMiddleware(jwtService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuthMiddleware_RefreshTokenUsedAsAccess(t *testing.T) {
	jwtService := newTestJWTService()
	pair, _ := newTestTokenPair(jwtService)

	router := gin.New()
	router.Use(JWTAuthMiddleware(jwtService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+pair.RefreshToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuthMiddleware_SkipPaths(t *testing.T) {
	jwtService := newTestJWTService()

	cfg := DefaultJWTConfig(jwtService)
	cfg.SkipPaths = append(cfg.SkipPaths, "/public")

	router := gin.New()
	router.Use(JWTAuthMiddlewareWithConfig(cfg))
	router.GET("/public", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/public", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestJWTAuthMiddleware_SkipPathPrefixes(t *testing.T) {
	jwtService := newTestJWTService()

	cfg := DefaultJWTConfig(jwtService)
	cfg.SkipPathPrefixes = append(cfg.SkipPathPrefixes, "/static")

	router := gin.New()
	router.Use(JWTAuthMiddlewareWithConfig(cfg))
	router.GET("/static/assets/image.png", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/static/assets/image.png", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestJWTAuthMiddleware_DefaultSkipPaths(t *testing.T) {
	jwtService := newTestJWTService()

	router := gin.New()
	router.Use(JWTAuthMiddleware(jwtService))

	defaultSkipPaths := []string{
		"/health",
		"/healthz",
		"/ready",
		"/api/v1/health",
		"/api/v1/auth/login",
		"/api/v1/auth/refresh",
	}

	for _, path := range defaultSkipPaths {
		router.GET(path, func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
	}

	for _, path := range defaultSkipPaths {
		t.Run("SkipPath_"+path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code, "Path %s should be skipped", path)
		})
	}
}

func TestJWTAuthMiddleware_ContextValues(t *testing.T) {
	jwtService := newTestJWTService()
	pair, input := newTestTokenPair(jwtService)

	var capturedUserID, capturedTenantID, capturedUsername string
	var capturedRoleIDs []string

	router := gin.New()
	router.Use(JWTAuthMiddleware(jwtService))
	router.GET("/test", func(c *gin.Context) {
		capturedUserID = GetJWTUserID(c)
		capturedTenantID = GetJWTTenantID(c)
		capturedUsername = GetJWTUsername(c)
		capturedRoleIDs = GetJWTRoleIDs(c)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, input.UserID.String(), capturedUserID)
	assert.Equal(t, input.TenantID.String(), capturedTenantID)
	assert.Equal(t, input.Username, capturedUsername)
	require.Len(t, capturedRoleIDs, 1)
	assert.Equal(t, input.RoleIDs[0].String(), capturedRoleIDs[0])
}

func TestGetJWTClaims_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	claims := GetJWTClaims(c)

	assert.Nil(t, claims)
}

func TestMustGetJWTClaims_Panics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	assert.Panics(t, func() {
		MustGetJWTClaims(c)
	})
}

func TestGetJWTUserID_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	userID := GetJWTUserID(c)

	assert.Empty(t, userID)
}

func TestGetJWTTenantID_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	tenantID := GetJWTTenantID(c)

	assert.Empty(t, tenantID)
}

func TestGetJWTUsername_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	username := GetJWTUsername(c)

	assert.Empty(t, username)
}

func TestGetJWTRoleIDs_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	roleIDs := GetJWTRoleIDs(c)

	assert.Nil(t, roleIDs)
}

func TestGetJWTPermissions_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	permissions := GetJWTPermissions(c)

	assert.Nil(t, permissions)
}

func TestOptionalJWTAuthMiddleware_NoToken(t *testing.T) {
	jwtService := newTestJWTService()

	var capturedClaims *auth.Claims

	router := gin.New()
	router.Use(OptionalJWTAuthMiddleware(jwtService))
	router.GET("/test", func(c *gin.Context) {
		capturedClaims = GetJWTClaims(c)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Nil(t, capturedClaims)
}

func TestOptionalJWTAuthMiddleware_ValidToken(t *testing.T) {
	jwtService := newTestJWTService()
	pair, input := newTestTokenPair(jwtService)

	var capturedClaims *auth.Claims

	router := gin.New()
	router.Use(OptionalJWTAuthMiddleware(jwtService))
	router.GET("/test", func(c *gin.Context) {
		capturedClaims = GetJWTClaims(c)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, capturedClaims)
	assert.Equal(t, input.UserID.String(), capturedClaims.UserID)
}

func TestOptionalJWTAuthMiddleware_InvalidToken(t *testing.T) {
	jwtService := newTestJWTService()

	var capturedClaims *auth.Claims

	router := gin.New()
	router.Use(OptionalJWTAuthMiddleware(jwtService))
	router.GET("/test", func(c *gin.Context) {
		capturedClaims = GetJWTClaims(c)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Nil(t, capturedClaims) // Invalid token, no claims
}

func TestJWTAuthMiddleware_CustomOnError(t *testing.T) {
	jwtService := newTestJWTService()

	customErrorCalled := false
	cfg := DefaultJWTConfig(jwtService)
	cfg.OnError = func(c *gin.Context, err error) {
		customErrorCalled = true
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"custom": "error"})
	}

	router := gin.New()
	router.Use(JWTAuthMiddlewareWithConfig(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.True(t, customErrorCalled)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestJWTAuthMiddleware_GetPermissions(t *testing.T) {
	jwtService := newTestJWTService()
	pair, input := newTestTokenPair(jwtService)

	var capturedPermissions []string

	router := gin.New()
	router.Use(JWTAuthMiddleware(jwtService))
	router.GET("/test", func(c *gin.Context) {
		capturedPermissions = GetJWTPermissions(c)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, input.Permissions, capturedPermissions)
}
