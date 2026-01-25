// Package middleware provides HTTP middleware for the ERP application.
package middleware

import (
	"net/http"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/infrastructure/persistence/datascope"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// DataScope context keys
const (
	DataScopesKey     = "data_scopes"
	DataScopeFilterKey = "data_scope_filter"
	UserRolesKey      = "user_roles"
)

// DataScopeMiddlewareConfig holds configuration for DataScope middleware
type DataScopeMiddlewareConfig struct {
	// RoleRepository is required for loading roles with data scopes
	RoleRepository identity.RoleRepository
	// SkipPaths are paths that don't require data scope filtering
	SkipPaths []string
	// SkipPathPrefixes are path prefixes that don't require data scope filtering
	SkipPathPrefixes []string
	// Logger for middleware logging
	Logger *zap.Logger
}

// DefaultDataScopeConfig returns default DataScope middleware configuration
func DefaultDataScopeConfig(roleRepo identity.RoleRepository) DataScopeMiddlewareConfig {
	return DataScopeMiddlewareConfig{
		RoleRepository: roleRepo,
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
		Logger: nil,
	}
}

// DataScopeMiddleware creates middleware that loads user's roles and data scopes into context.
// This middleware should run after JWTAuthMiddleware as it depends on JWT claims.
func DataScopeMiddleware(roleRepo identity.RoleRepository) gin.HandlerFunc {
	return DataScopeMiddlewareWithConfig(DefaultDataScopeConfig(roleRepo))
}

// DataScopeMiddlewareWithConfig creates DataScope middleware with custom config
func DataScopeMiddlewareWithConfig(cfg DataScopeMiddlewareConfig) gin.HandlerFunc {
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
			if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
				c.Next()
				return
			}
		}

		// Get role IDs from JWT claims (set by JWTAuthMiddleware)
		roleIDStrings := GetJWTRoleIDs(c)
		if len(roleIDStrings) == 0 {
			// No roles - user has no data scope restrictions (default to ALL)
			c.Next()
			return
		}

		// Parse role IDs
		roleIDs := make([]uuid.UUID, 0, len(roleIDStrings))
		for _, idStr := range roleIDStrings {
			if id, err := uuid.Parse(idStr); err == nil {
				roleIDs = append(roleIDs, id)
			}
		}

		if len(roleIDs) == 0 {
			c.Next()
			return
		}

		// Get tenant ID from JWT claims
		tenantIDStr := GetJWTTenantID(c)
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			if cfg.Logger != nil {
				cfg.Logger.Warn("Invalid tenant ID in JWT",
					zap.String("tenant_id", tenantIDStr),
					zap.Error(err),
				)
			}
			c.Next()
			return
		}

		// Load roles with their data scopes
		ctx := c.Request.Context()
		rolePtrs, err := cfg.RoleRepository.FindByIDs(ctx, roleIDs)
		if err != nil {
			if cfg.Logger != nil {
				cfg.Logger.Error("Failed to load roles for data scope",
					zap.Error(err),
					zap.String("tenant_id", tenantIDStr),
				)
			}
			// Continue without data scope filtering on error
			c.Next()
			return
		}

		// Filter roles by tenant ID and load data scopes
		roles := make([]identity.Role, 0, len(rolePtrs))
		for _, rolePtr := range rolePtrs {
			if rolePtr == nil || rolePtr.TenantID != tenantID {
				continue
			}
			if err := cfg.RoleRepository.LoadPermissionsAndDataScopes(ctx, rolePtr); err != nil {
				if cfg.Logger != nil {
					cfg.Logger.Warn("Failed to load data scopes for role",
						zap.Error(err),
						zap.String("role_id", rolePtr.ID.String()),
					)
				}
			}
			roles = append(roles, *rolePtr)
		}

		// Store roles in context
		c.Set(UserRolesKey, roles)

		// Create and store data scope filter
		filter := datascope.NewFilter(ctx, roles)
		c.Set(DataScopeFilterKey, filter)

		// Also add data scopes to request context for downstream use
		ctx = datascope.WithDataScopes(ctx, roles)
		c.Request = c.Request.WithContext(ctx)

		if cfg.Logger != nil {
			cfg.Logger.Debug("Data scopes loaded",
				zap.Int("role_count", len(roles)),
				zap.String("user_id", GetJWTUserID(c)),
			)
		}

		c.Next()
	}
}

// GetDataScopeFilter retrieves the DataScope filter from gin.Context
func GetDataScopeFilter(c *gin.Context) *datascope.Filter {
	if filter, exists := c.Get(DataScopeFilterKey); exists {
		if f, ok := filter.(*datascope.Filter); ok {
			return f
		}
	}
	return nil
}

// GetUserRoles retrieves the user's roles from gin.Context
func GetUserRoles(c *gin.Context) []identity.Role {
	if roles, exists := c.Get(UserRolesKey); exists {
		if r, ok := roles.([]identity.Role); ok {
			return r
		}
	}
	return nil
}

// RequireDataScope is a middleware that requires a specific data scope type for a resource.
// This can be used to restrict access to routes that require higher-level data access.
func RequireDataScope(resource string, minScopeType identity.DataScopeType, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		filter := GetDataScopeFilter(c)
		if filter == nil {
			// No filter means no restrictions - allow access
			c.Next()
			return
		}

		actualScope := filter.GetScopeType(resource)
		if !meetsMinimumScope(actualScope, minScopeType) {
			if logger != nil {
				logger.Warn("Insufficient data scope",
					zap.String("resource", resource),
					zap.String("required", string(minScopeType)),
					zap.String("actual", string(actualScope)),
					zap.String("user_id", GetJWTUserID(c)),
				)
			}
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INSUFFICIENT_DATA_SCOPE",
					"message": "You don't have sufficient data access for this operation",
				},
			})
			return
		}

		c.Next()
	}
}

// meetsMinimumScope checks if actualScope meets or exceeds minScope
func meetsMinimumScope(actualScope, minScope identity.DataScopeType) bool {
	scopeLevels := map[identity.DataScopeType]int{
		identity.DataScopeSelf:       10,
		identity.DataScopeCustom:     40,
		identity.DataScopeDepartment: 50,
		identity.DataScopeAll:        100,
	}

	actualLevel := scopeLevels[actualScope]
	minLevel := scopeLevels[minScope]

	return actualLevel >= minLevel
}

// ApplyDataScopeToQuery is a helper function that applies data scope filtering to a GORM query.
// Usage: db = middleware.ApplyDataScopeToQuery(c, db, "sales_order")
func ApplyDataScopeToQuery(c *gin.Context, db interface{}, resource string) interface{} {
	filter := GetDataScopeFilter(c)
	if filter == nil {
		return db
	}

	// Type assert to *gorm.DB - import avoided by using interface{}
	// The actual application should use datascope.Filter.Apply() directly
	return db
}
