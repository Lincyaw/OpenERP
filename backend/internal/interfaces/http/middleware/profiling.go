// Package middleware provides HTTP middleware for the ERP system.
package middleware

import (
	"context"
	"strings"

	"github.com/erp/backend/internal/infrastructure/telemetry"
	"github.com/gin-gonic/gin"
)

// ProfilingConfig holds configuration for the profiling middleware.
type ProfilingConfig struct {
	// Enabled controls whether profiling labels are added to requests.
	Enabled bool
	// SkipPaths are paths that don't need profiling labels (e.g., health checks).
	SkipPaths []string
	// SkipPathPrefixes are path prefixes that don't need profiling labels.
	SkipPathPrefixes []string
}

// DefaultProfilingConfig returns default profiling middleware configuration.
func DefaultProfilingConfig() ProfilingConfig {
	return ProfilingConfig{
		Enabled: true,
		SkipPaths: []string{
			"/health",
			"/healthz",
			"/ready",
			"/metrics",
		},
		SkipPathPrefixes: []string{
			"/swagger",
			"/api-docs",
		},
	}
}

// Profiling returns profiling middleware with default configuration.
// This middleware adds Pyroscope labels to the request context for
// continuous profiling analysis.
func Profiling() gin.HandlerFunc {
	return ProfilingWithConfig(DefaultProfilingConfig())
}

// ProfilingWithConfig returns profiling middleware with custom configuration.
// The middleware adds the following labels to the profiling context:
//   - controller: Handler name (e.g., "ProductHandler")
//   - route: Route pattern (e.g., "/api/v1/products/:id")
//   - method: HTTP method (GET, POST, PUT, DELETE)
//   - tenant_id: Tenant ID (from JWT or header)
//
// These labels can be used in Pyroscope UI to filter and analyze profiles
// by different dimensions.
func ProfilingWithConfig(cfg ProfilingConfig) gin.HandlerFunc {
	if !cfg.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

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

		// Extract profiling labels
		labels := extractProfilingLabels(c)

		// Wrap the handler execution with profiling labels
		telemetry.WithProfilingLabels(c.Request.Context(), labels, func(ctx context.Context) {
			// Update request context with labeled context
			c.Request = c.Request.WithContext(ctx)
			c.Next()
		})
	}
}

// extractProfilingLabels extracts profiling labels from the gin context.
func extractProfilingLabels(c *gin.Context) map[string]string {
	labels := make(map[string]string, 4)

	// HTTP method (low cardinality: GET, POST, PUT, DELETE, PATCH)
	method := c.Request.Method
	if method != "" {
		labels[telemetry.ProfilingLabelMethod] = method
	}

	// Route pattern (from gin's matched route, e.g., "/api/v1/products/:id")
	// This is low cardinality as it uses the pattern, not the actual path
	route := c.FullPath()
	if route != "" {
		labels[telemetry.ProfilingLabelRoute] = route
	}

	// Controller/handler name - derive from route pattern
	// e.g., "/api/v1/products" -> "products"
	controller := extractControllerFromRoute(route)
	if controller != "" {
		labels[telemetry.ProfilingLabelController] = controller
	}

	// Tenant ID (from JWT claims or header - set by upstream middleware)
	// This is low-to-medium cardinality depending on the number of tenants
	tenantID := getTenantIDForProfiling(c)
	if tenantID != "" {
		labels[telemetry.ProfilingLabelTenantID] = tenantID
	}

	return labels
}

// extractControllerFromRoute derives a controller name from the route pattern.
// Example: "/api/v1/products/:id" -> "products"
// Example: "/api/v1/customers/:id/orders" -> "customers"
func extractControllerFromRoute(route string) string {
	if route == "" {
		return ""
	}

	// Split route by '/'
	parts := strings.Split(route, "/")

	// Find the first meaningful path segment after "api" and version
	// Expected format: /api/v1/{resource}/...
	for i, part := range parts {
		// Skip empty parts, "api", and version segments (v1, v2, etc.)
		if part == "" || part == "api" || isVersionSegment(part) {
			continue
		}

		// Skip path parameters (start with ':' or are in curly braces)
		if strings.HasPrefix(part, ":") || strings.HasPrefix(part, "{") {
			continue
		}

		// Found the resource name
		// If the next part is a path parameter, this is likely the controller
		// e.g., "/api/v1/products/:id" -> products
		if i+1 < len(parts) && (strings.HasPrefix(parts[i+1], ":") || strings.HasPrefix(parts[i+1], "{")) {
			return part
		}

		// Return the first meaningful segment
		return part
	}

	return ""
}

// isVersionSegment checks if a path segment is an API version (v1, v2, etc.)
func isVersionSegment(segment string) bool {
	if len(segment) < 2 {
		return false
	}
	if segment[0] != 'v' && segment[0] != 'V' {
		return false
	}
	// Check if remaining characters are digits
	for i := 1; i < len(segment); i++ {
		if segment[i] < '0' || segment[i] > '9' {
			return false
		}
	}
	return true
}

// getTenantIDForProfiling retrieves the tenant ID from the gin context.
// It checks both JWT claims and the tenant middleware context values.
func getTenantIDForProfiling(c *gin.Context) string {
	// First check JWT claims (set by JWT middleware)
	if tenantID, exists := c.Get(JWTTenantIDKey); exists {
		if id, ok := tenantID.(string); ok && id != "" {
			return id
		}
	}

	// Fallback to tenant middleware key
	if tenantID, exists := c.Get(TenantIDKey); exists {
		if id, ok := tenantID.(string); ok && id != "" {
			return id
		}
	}

	return ""
}

// ProfilingAttributeInjector returns a middleware that injects profiling labels
// after authentication middleware has run (so tenant_id is available).
// Use this middleware AFTER both JWT and Tenant middleware in the chain.
func ProfilingAttributeInjector() gin.HandlerFunc {
	return ProfilingWithConfig(DefaultProfilingConfig())
}
