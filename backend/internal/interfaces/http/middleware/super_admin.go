package middleware

import (
	"net/http"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/infrastructure/auth"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Super admin context keys
const (
	// IsSuperAdminKey is the context key for the super admin flag
	IsSuperAdminKey = "is_super_admin"
)

// SuperAdminConfig holds configuration for super admin middleware
type SuperAdminConfig struct {
	// Logger for middleware logging
	Logger *zap.Logger
	// OnDenied is called when access is denied (optional)
	OnDenied func(c *gin.Context, reason string)
}

// SuperAdminMiddleware creates middleware that requires super admin access
// This middleware should be applied to /api/v1/admin/* routes
// It verifies:
// 1. User belongs to the system tenant (tenant_id = SystemTenantID)
// 2. User has super_admin role OR tenant:* permissions
func SuperAdminMiddleware() gin.HandlerFunc {
	return SuperAdminMiddlewareWithConfig(SuperAdminConfig{})
}

// SuperAdminMiddlewareWithConfig creates super admin middleware with custom config
func SuperAdminMiddlewareWithConfig(cfg SuperAdminConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get JWT claims from context (requires JWTAuthMiddleware to run first)
		claims := GetJWTClaims(c)
		if claims == nil {
			handleSuperAdminUnauthorized(c, cfg, "No authentication claims found")
			return
		}

		// Check 1: Verify user belongs to system tenant
		if claims.TenantID != identity.SystemTenantID {
			if cfg.Logger != nil {
				cfg.Logger.Warn("Super admin access denied: not system tenant",
					zap.String("user_id", claims.UserID),
					zap.String("tenant_id", claims.TenantID),
					zap.String("expected_tenant_id", identity.SystemTenantID),
					zap.String("path", c.Request.URL.Path),
				)
			}
			handleSuperAdminForbidden(c, cfg, "Access denied: not a system administrator")
			return
		}

		// Check 2: Verify user has super_admin role OR tenant:* permissions
		isSuperAdmin := checkSuperAdminAccess(claims)
		if !isSuperAdmin {
			if cfg.Logger != nil {
				cfg.Logger.Warn("Super admin access denied: insufficient permissions",
					zap.String("user_id", claims.UserID),
					zap.String("tenant_id", claims.TenantID),
					zap.Strings("role_ids", claims.RoleIDs),
					zap.Strings("permissions", claims.Permissions),
					zap.String("path", c.Request.URL.Path),
				)
			}
			handleSuperAdminForbidden(c, cfg, "Access denied: insufficient super admin permissions")
			return
		}

		// Set IsSuperAdmin flag in context
		c.Set(IsSuperAdminKey, true)

		if cfg.Logger != nil {
			cfg.Logger.Debug("Super admin access granted",
				zap.String("user_id", claims.UserID),
				zap.String("username", claims.Username),
				zap.String("path", c.Request.URL.Path),
			)
		}

		c.Next()
	}
}

// checkSuperAdminAccess checks if the user has super admin access
// Returns true if user has:
// - super_admin role ID in their role list, OR
// - Any tenant:* permission (tenant:read, tenant:create, tenant:update, tenant:delete, tenant:suspend, tenant:manage)
func checkSuperAdminAccess(claims *auth.Claims) bool {
	// Check for super admin role ID
	for _, roleID := range claims.RoleIDs {
		if roleID == identity.SuperAdminRoleID {
			return true
		}
	}

	// Check for any tenant:* permission
	tenantPermissions := identity.TenantPermissions()
	for _, perm := range claims.Permissions {
		for _, tenantPerm := range tenantPermissions {
			if perm == tenantPerm {
				return true
			}
		}
	}

	return false
}

// handleSuperAdminUnauthorized handles 401 Unauthorized responses
func handleSuperAdminUnauthorized(c *gin.Context, cfg SuperAdminConfig, reason string) {
	if cfg.OnDenied != nil {
		cfg.OnDenied(c, reason)
		return
	}

	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "ERR_UNAUTHORIZED",
			"message": "Authentication required for admin access",
		},
	})
}

// handleSuperAdminForbidden handles 403 Forbidden responses
func handleSuperAdminForbidden(c *gin.Context, cfg SuperAdminConfig, reason string) {
	if cfg.OnDenied != nil {
		cfg.OnDenied(c, reason)
		return
	}

	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "ERR_FORBIDDEN",
			"message": reason,
		},
	})
}

// IsSuperAdmin returns true if the current user is a super admin
// This checks the context flag set by SuperAdminMiddleware
func IsSuperAdmin(c *gin.Context) bool {
	if isSuperAdmin, exists := c.Get(IsSuperAdminKey); exists {
		if flag, ok := isSuperAdmin.(bool); ok {
			return flag
		}
	}
	return false
}

// RequireSuperAdmin is a helper function that can be used in handlers
// to verify super admin access and abort if not authorized
// Returns true if the user is a super admin, false if aborted
func RequireSuperAdmin(c *gin.Context) bool {
	if !IsSuperAdmin(c) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ERR_FORBIDDEN",
				"message": "Access denied: super admin privileges required",
			},
		})
		return false
	}
	return true
}

// CheckSuperAdminFromClaims checks if the given claims represent a super admin
// This is useful for checking super admin status without the middleware
func CheckSuperAdminFromClaims(claims *auth.Claims) bool {
	if claims == nil {
		return false
	}

	// Must be from system tenant
	if claims.TenantID != identity.SystemTenantID {
		return false
	}

	return checkSuperAdminAccess(claims)
}
