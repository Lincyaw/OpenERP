package middleware

import (
	"net/http"
	"strings"

	"github.com/erp/backend/internal/infrastructure/auth"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// PermissionConfig holds configuration for permission middleware
type PermissionConfig struct {
	// Logger for middleware logging
	Logger *zap.Logger
	// OnDenied is called when permission is denied (optional)
	OnDenied func(c *gin.Context, requiredPerms []string)
}

// RequirePermission creates middleware that requires a specific permission
// This is a convenience function for single permission requirement
func RequirePermission(permission string) gin.HandlerFunc {
	return RequireAnyPermission(permission)
}

// RequirePermissionWithConfig creates middleware with custom config
func RequirePermissionWithConfig(permission string, cfg PermissionConfig) gin.HandlerFunc {
	return RequireAnyPermissionWithConfig(cfg, permission)
}

// RequireAnyPermission creates middleware that requires any of the specified permissions
// User must have at least one of the listed permissions to proceed
func RequireAnyPermission(permissions ...string) gin.HandlerFunc {
	return RequireAnyPermissionWithConfig(PermissionConfig{}, permissions...)
}

// RequireAnyPermissionWithConfig creates middleware that requires any of the specified permissions with custom config
func RequireAnyPermissionWithConfig(cfg PermissionConfig, permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetJWTClaims(c)
		if claims == nil {
			handlePermissionDenied(c, cfg, permissions, "No authentication claims found")
			return
		}

		if !claims.HasAnyPermission(permissions...) {
			handlePermissionDenied(c, cfg, permissions, "User lacks required permission")
			return
		}

		if cfg.Logger != nil {
			cfg.Logger.Debug("Permission check passed",
				zap.String("user_id", claims.UserID),
				zap.Strings("required_any", permissions),
				zap.Strings("user_permissions", claims.Permissions),
			)
		}

		c.Next()
	}
}

// RequireAllPermissions creates middleware that requires all of the specified permissions
// User must have every listed permission to proceed
func RequireAllPermissions(permissions ...string) gin.HandlerFunc {
	return RequireAllPermissionsWithConfig(PermissionConfig{}, permissions...)
}

// RequireAllPermissionsWithConfig creates middleware that requires all permissions with custom config
func RequireAllPermissionsWithConfig(cfg PermissionConfig, permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetJWTClaims(c)
		if claims == nil {
			handlePermissionDenied(c, cfg, permissions, "No authentication claims found")
			return
		}

		if !claims.HasAllPermissions(permissions...) {
			handlePermissionDenied(c, cfg, permissions, "User lacks one or more required permissions")
			return
		}

		if cfg.Logger != nil {
			cfg.Logger.Debug("All permissions check passed",
				zap.String("user_id", claims.UserID),
				zap.Strings("required_all", permissions),
				zap.Strings("user_permissions", claims.Permissions),
			)
		}

		c.Next()
	}
}

// RequireResource creates middleware that checks permission for a resource with dynamic action
// The action is determined by the HTTP method:
// - GET -> read
// - POST -> create
// - PUT/PATCH -> update
// - DELETE -> delete
func RequireResource(resource string) gin.HandlerFunc {
	return RequireResourceWithConfig(resource, PermissionConfig{})
}

// RequireResourceWithConfig creates middleware with custom config
func RequireResourceWithConfig(resource string, cfg PermissionConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		action := methodToAction(c.Request.Method)
		permission := resource + ":" + action

		claims := GetJWTClaims(c)
		if claims == nil {
			handlePermissionDenied(c, cfg, []string{permission}, "No authentication claims found")
			return
		}

		if !claims.HasPermission(permission) {
			handlePermissionDenied(c, cfg, []string{permission}, "User lacks required permission for resource")
			return
		}

		if cfg.Logger != nil {
			cfg.Logger.Debug("Resource permission check passed",
				zap.String("user_id", claims.UserID),
				zap.String("resource", resource),
				zap.String("action", action),
				zap.String("permission", permission),
			)
		}

		c.Next()
	}
}

// RequireResourceAction creates middleware that checks a specific resource:action permission
func RequireResourceAction(resource, action string) gin.HandlerFunc {
	return RequirePermission(resource + ":" + action)
}

// methodToAction converts HTTP method to permission action
func methodToAction(method string) string {
	switch strings.ToUpper(method) {
	case http.MethodGet:
		return "read"
	case http.MethodPost:
		return "create"
	case http.MethodPut, http.MethodPatch:
		return "update"
	case http.MethodDelete:
		return "delete"
	default:
		return "read"
	}
}

// handlePermissionDenied handles permission denied scenarios
func handlePermissionDenied(c *gin.Context, cfg PermissionConfig, requiredPerms []string, reason string) {
	if cfg.OnDenied != nil {
		cfg.OnDenied(c, requiredPerms)
		return
	}

	if cfg.Logger != nil {
		claims := GetJWTClaims(c)
		userID := ""
		userPerms := []string{}
		if claims != nil {
			userID = claims.UserID
			userPerms = claims.Permissions
		}

		cfg.Logger.Warn("Permission denied",
			zap.String("reason", reason),
			zap.String("user_id", userID),
			zap.Strings("required_permissions", requiredPerms),
			zap.Strings("user_permissions", userPerms),
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
		)
	}

	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "ERR_FORBIDDEN",
			"message": "Access denied: insufficient permissions",
		},
	})
}

// RoutePermission defines permission requirement for a route
type RoutePermission struct {
	Method      string   // HTTP method (GET, POST, etc.) or "*" for all methods
	Path        string   // Route path pattern
	Permissions []string // Required permissions (any of these)
	RequireAll  bool     // If true, require all permissions instead of any
}

// RoutePermissionConfig holds configuration for route-based permission checking
type RoutePermissionConfig struct {
	// Routes defines permission requirements for specific routes
	Routes []RoutePermission
	// Logger for middleware logging
	Logger *zap.Logger
	// DefaultDeny if true, denies access when no matching route is found
	// If false, allows access when no matching route is found
	DefaultDeny bool
	// OnDenied is called when permission is denied (optional)
	OnDenied func(c *gin.Context, route *RoutePermission)
}

// RoutePermissionMiddleware creates middleware that checks permissions based on route configuration
// This allows centralized permission management for all routes
func RoutePermissionMiddleware(cfg RoutePermissionConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.FullPath() // Use FullPath to get the route pattern with parameters
		if path == "" {
			path = c.Request.URL.Path
		}
		method := c.Request.Method

		// Find matching route permission
		var matchedRoute *RoutePermission
		for i := range cfg.Routes {
			route := &cfg.Routes[i]
			if matchRoute(route, method, path) {
				matchedRoute = route
				break
			}
		}

		// No matching route found
		if matchedRoute == nil {
			if cfg.DefaultDeny {
				if cfg.Logger != nil {
					cfg.Logger.Warn("No route permission defined, access denied",
						zap.String("path", path),
						zap.String("method", method),
					)
				}
				handleRoutePermissionDenied(c, cfg, nil)
				return
			}
			// Allow access if DefaultDeny is false
			c.Next()
			return
		}

		// Check permissions
		claims := GetJWTClaims(c)
		if claims == nil {
			handleRoutePermissionDenied(c, cfg, matchedRoute)
			return
		}

		var hasPermission bool
		if matchedRoute.RequireAll {
			hasPermission = claims.HasAllPermissions(matchedRoute.Permissions...)
		} else {
			hasPermission = claims.HasAnyPermission(matchedRoute.Permissions...)
		}

		if !hasPermission {
			handleRoutePermissionDenied(c, cfg, matchedRoute)
			return
		}

		if cfg.Logger != nil {
			cfg.Logger.Debug("Route permission check passed",
				zap.String("user_id", claims.UserID),
				zap.String("path", path),
				zap.String("method", method),
				zap.Strings("required_permissions", matchedRoute.Permissions),
			)
		}

		c.Next()
	}
}

// matchRoute checks if a route permission matches the request
func matchRoute(route *RoutePermission, method, path string) bool {
	// Check method match
	if route.Method != "*" && !strings.EqualFold(route.Method, method) {
		return false
	}

	// Check path match
	// Support exact match and prefix match (with trailing *)
	if strings.HasSuffix(route.Path, "*") {
		prefix := strings.TrimSuffix(route.Path, "*")
		return strings.HasPrefix(path, prefix)
	}

	return route.Path == path
}

// handleRoutePermissionDenied handles route-based permission denied scenarios
func handleRoutePermissionDenied(c *gin.Context, cfg RoutePermissionConfig, route *RoutePermission) {
	if cfg.OnDenied != nil {
		cfg.OnDenied(c, route)
		return
	}

	if cfg.Logger != nil {
		claims := GetJWTClaims(c)
		userID := ""
		userPerms := []string{}
		if claims != nil {
			userID = claims.UserID
			userPerms = claims.Permissions
		}

		requiredPerms := []string{}
		if route != nil {
			requiredPerms = route.Permissions
		}

		cfg.Logger.Warn("Route permission denied",
			zap.String("user_id", userID),
			zap.Strings("required_permissions", requiredPerms),
			zap.Strings("user_permissions", userPerms),
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
		)
	}

	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "ERR_FORBIDDEN",
			"message": "Access denied: insufficient permissions",
		},
	})
}

// HasPermission is a helper function to check permission in handlers
// Returns true if the user has the specified permission
func HasPermission(c *gin.Context, permission string) bool {
	claims := GetJWTClaims(c)
	if claims == nil {
		return false
	}
	return claims.HasPermission(permission)
}

// HasAnyPermission is a helper function to check if user has any of the permissions
func HasAnyPermission(c *gin.Context, permissions ...string) bool {
	claims := GetJWTClaims(c)
	if claims == nil {
		return false
	}
	return claims.HasAnyPermission(permissions...)
}

// HasAllPermissions is a helper function to check if user has all of the permissions
func HasAllPermissions(c *gin.Context, permissions ...string) bool {
	claims := GetJWTClaims(c)
	if claims == nil {
		return false
	}
	return claims.HasAllPermissions(permissions...)
}

// MustHavePermission aborts the request if the user doesn't have the permission
// Returns true if the user has permission, false if aborted
func MustHavePermission(c *gin.Context, permission string) bool {
	if !HasPermission(c, permission) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ERR_FORBIDDEN",
				"message": "Access denied: insufficient permissions",
			},
		})
		return false
	}
	return true
}

// CheckPermissionFunc is a function type for custom permission checking
type CheckPermissionFunc func(claims *auth.Claims, c *gin.Context) bool

// RequireCustomPermission creates middleware with a custom permission check function
// This allows for complex permission logic that can't be expressed with simple permission strings
func RequireCustomPermission(checkFunc CheckPermissionFunc) gin.HandlerFunc {
	return RequireCustomPermissionWithConfig(checkFunc, PermissionConfig{})
}

// RequireCustomPermissionWithConfig creates custom permission middleware with config
func RequireCustomPermissionWithConfig(checkFunc CheckPermissionFunc, cfg PermissionConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetJWTClaims(c)
		if claims == nil {
			handlePermissionDenied(c, cfg, []string{"custom"}, "No authentication claims found")
			return
		}

		if !checkFunc(claims, c) {
			handlePermissionDenied(c, cfg, []string{"custom"}, "Custom permission check failed")
			return
		}

		c.Next()
	}
}
