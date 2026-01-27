package middleware

import (
	"net/http"
	"strings"

	"github.com/erp/backend/internal/infrastructure/auth"
	"github.com/erp/backend/internal/infrastructure/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// JWT context keys
const (
	JWTClaimsKey   = "jwt_claims"
	JWTUserIDKey   = "jwt_user_id"
	JWTTenantIDKey = "jwt_tenant_id"
	JWTUsernameKey = "jwt_username"
	JWTRoleIDsKey  = "jwt_role_ids"
	JWTPermissions = "jwt_permissions"
	AuthHeaderKey  = "Authorization"
	BearerPrefix   = "Bearer "
)

// JWTMiddlewareConfig holds configuration for JWT middleware
type JWTMiddlewareConfig struct {
	// JWTService is required for token validation
	JWTService *auth.JWTService
	// TokenBlacklist is optional for checking revoked tokens
	TokenBlacklist auth.TokenBlacklist
	// SkipPaths are paths that don't require authentication
	SkipPaths []string
	// SkipPathPrefixes are path prefixes that don't require authentication
	SkipPathPrefixes []string
	// Optional callback if token is invalid (default: return 401)
	OnError func(c *gin.Context, err error)
	// Logger for middleware logging
	Logger *zap.Logger
}

// DefaultJWTConfig returns default JWT middleware configuration
func DefaultJWTConfig(jwtService *auth.JWTService) JWTMiddlewareConfig {
	return JWTMiddlewareConfig{
		JWTService:     jwtService,
		TokenBlacklist: nil,
		SkipPaths: []string{
			"/health",
			"/healthz",
			"/ready",
			"/metrics",
			"/api/v1/health",
			"/api/v1/auth/login",
			"/api/v1/auth/refresh",
		},
		SkipPathPrefixes: []string{
			"/swagger",
			"/api-docs",
		},
		OnError: nil,
		Logger:  nil,
	}
}

// JWTAuthMiddleware creates JWT authentication middleware
func JWTAuthMiddleware(jwtService *auth.JWTService) gin.HandlerFunc {
	return JWTAuthMiddlewareWithConfig(DefaultJWTConfig(jwtService))
}

// JWTAuthMiddlewareWithConfig creates JWT authentication middleware with custom config
func JWTAuthMiddlewareWithConfig(cfg JWTMiddlewareConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Check skip paths
		for _, skipPath := range cfg.SkipPaths {
			if path == skipPath {
				c.Next()
				return
			}
		}

		// Check skip path prefixes
		for _, prefix := range cfg.SkipPathPrefixes {
			if strings.HasPrefix(path, prefix) {
				c.Next()
				return
			}
		}

		// Extract token from Authorization header
		authHeader := c.GetHeader(AuthHeaderKey)
		if authHeader == "" {
			handleAuthError(c, cfg, auth.ErrInvalidToken, "Missing authorization header")
			return
		}

		// Check Bearer prefix
		if !strings.HasPrefix(authHeader, BearerPrefix) {
			handleAuthError(c, cfg, auth.ErrInvalidToken, "Invalid authorization header format")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, BearerPrefix)
		if tokenString == "" {
			handleAuthError(c, cfg, auth.ErrInvalidToken, "Missing token")
			return
		}

		// Validate token
		claims, err := cfg.JWTService.ValidateAccessToken(tokenString)
		if err != nil {
			handleAuthError(c, cfg, err, "Token validation failed")
			return
		}

		// Check token blacklist if configured
		if cfg.TokenBlacklist != nil {
			ctx := c.Request.Context()

			// Check if the specific token JTI is blacklisted (individual logout)
			if claims.ID != "" {
				blacklisted, err := cfg.TokenBlacklist.IsBlacklisted(ctx, claims.ID)
				if err != nil {
					// Log error but don't fail the request - fail open for availability
					if cfg.Logger != nil {
						cfg.Logger.Error("Failed to check token blacklist",
							zap.String("jti", claims.ID),
							zap.Error(err))
					}
				} else if blacklisted {
					handleAuthError(c, cfg, auth.ErrTokenBlacklisted, "Token has been revoked")
					return
				}
			}

			// Check if user's tokens have been globally invalidated (force logout, password change)
			if claims.UserID != "" {
				tokenIssuedAt := claims.GetIssuedAtTime()
				invalidated, err := cfg.TokenBlacklist.IsUserTokenInvalidated(ctx, claims.UserID, tokenIssuedAt)
				if err != nil {
					// Log error but don't fail the request - fail open for availability
					if cfg.Logger != nil {
						cfg.Logger.Error("Failed to check user token invalidation",
							zap.String("user_id", claims.UserID),
							zap.Error(err))
					}
				} else if invalidated {
					handleAuthError(c, cfg, auth.ErrTokenBlacklisted, "User session has been invalidated")
					return
				}
			}
		}

		// Store claims in context for downstream use
		c.Set(JWTClaimsKey, claims)
		c.Set(JWTUserIDKey, claims.UserID)
		c.Set(JWTTenantIDKey, claims.TenantID)
		c.Set(JWTUsernameKey, claims.Username)
		c.Set(JWTRoleIDsKey, claims.RoleIDs)
		c.Set(JWTPermissions, claims.Permissions)

		// Also set in request context for logger
		ctx := c.Request.Context()
		log := logger.FromContext(ctx)
		ctx, _ = logger.WithUserID(ctx, log, claims.UserID)
		ctx, _ = logger.WithTenantID(ctx, log, claims.TenantID)
		c.Request = c.Request.WithContext(ctx)

		// Log authentication success if logger is provided
		if cfg.Logger != nil {
			cfg.Logger.Debug("JWT authentication successful",
				zap.String("user_id", claims.UserID),
				zap.String("tenant_id", claims.TenantID),
				zap.String("username", claims.Username),
			)
		}

		c.Next()
	}
}

// handleAuthError handles authentication errors
func handleAuthError(c *gin.Context, cfg JWTMiddlewareConfig, err error, message string) {
	if cfg.OnError != nil {
		cfg.OnError(c, err)
		return
	}

	if cfg.Logger != nil {
		cfg.Logger.Warn("JWT authentication failed",
			zap.Error(err),
			zap.String("message", message),
			zap.String("path", c.Request.URL.Path),
		)
	}

	errorCode := "UNAUTHORIZED"
	errorMessage := "Authentication required"

	switch err {
	case auth.ErrExpiredToken:
		errorCode = "TOKEN_EXPIRED"
		errorMessage = "Token has expired"
	case auth.ErrInvalidToken:
		errorCode = "INVALID_TOKEN"
		errorMessage = "Invalid token"
	case auth.ErrInvalidTokenType:
		errorCode = "INVALID_TOKEN_TYPE"
		errorMessage = "Invalid token type"
	case auth.ErrTokenNotYetValid:
		errorCode = "TOKEN_NOT_VALID"
		errorMessage = "Token is not yet valid"
	case auth.ErrTokenBlacklisted:
		errorCode = "TOKEN_REVOKED"
		errorMessage = "Token has been revoked"
	}

	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"success": false,
		"error": gin.H{
			"code":    errorCode,
			"message": errorMessage,
		},
	})
}

// GetJWTClaims retrieves JWT claims from gin.Context
func GetJWTClaims(c *gin.Context) *auth.Claims {
	if claims, exists := c.Get(JWTClaimsKey); exists {
		if jwtClaims, ok := claims.(*auth.Claims); ok {
			return jwtClaims
		}
	}
	return nil
}

// MustGetJWTClaims retrieves JWT claims from gin.Context or panics if not found
func MustGetJWTClaims(c *gin.Context) *auth.Claims {
	claims := GetJWTClaims(c)
	if claims == nil {
		panic("jwt claims not found in context")
	}
	return claims
}

// GetJWTUserID retrieves the user ID from JWT claims in context
func GetJWTUserID(c *gin.Context) string {
	if userID, exists := c.Get(JWTUserIDKey); exists {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return ""
}

// GetJWTTenantID retrieves the tenant ID from JWT claims in context
func GetJWTTenantID(c *gin.Context) string {
	if tenantID, exists := c.Get(JWTTenantIDKey); exists {
		if id, ok := tenantID.(string); ok {
			return id
		}
	}
	return ""
}

// GetJWTUsername retrieves the username from JWT claims in context
func GetJWTUsername(c *gin.Context) string {
	if username, exists := c.Get(JWTUsernameKey); exists {
		if u, ok := username.(string); ok {
			return u
		}
	}
	return ""
}

// GetJWTRoleIDs retrieves the role IDs from JWT claims in context
func GetJWTRoleIDs(c *gin.Context) []string {
	if roleIDs, exists := c.Get(JWTRoleIDsKey); exists {
		if ids, ok := roleIDs.([]string); ok {
			return ids
		}
	}
	return nil
}

// GetJWTPermissions retrieves the permissions from JWT claims in context
func GetJWTPermissions(c *gin.Context) []string {
	if permissions, exists := c.Get(JWTPermissions); exists {
		if perms, ok := permissions.([]string); ok {
			return perms
		}
	}
	return nil
}

// OptionalJWTAuthMiddleware creates middleware that doesn't require JWT but extracts claims if present
func OptionalJWTAuthMiddleware(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(AuthHeaderKey)
		if authHeader == "" || !strings.HasPrefix(authHeader, BearerPrefix) {
			c.Next()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, BearerPrefix)
		if tokenString == "" {
			c.Next()
			return
		}

		// Try to validate token - don't fail if invalid
		claims, err := jwtService.ValidateAccessToken(tokenString)
		if err != nil {
			c.Next()
			return
		}

		// Store claims in context
		c.Set(JWTClaimsKey, claims)
		c.Set(JWTUserIDKey, claims.UserID)
		c.Set(JWTTenantIDKey, claims.TenantID)
		c.Set(JWTUsernameKey, claims.Username)
		c.Set(JWTRoleIDsKey, claims.RoleIDs)
		c.Set(JWTPermissions, claims.Permissions)

		c.Next()
	}
}
