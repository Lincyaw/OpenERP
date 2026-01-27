// Package middleware provides HTTP middleware for the ERP system.
package middleware

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Constants for trace attribute validation.
const (
	// MaxRequestIDLength is the maximum length for request IDs to prevent DoS via large headers.
	MaxRequestIDLength = 128
	// MaxTenantIDLength is the maximum length for tenant IDs.
	MaxTenantIDLength = 64
)

// uuidRegex validates UUID format for tenant IDs from headers.
var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// TracingConfig holds configuration for the tracing middleware.
type TracingConfig struct {
	// ServiceName is the name of the service for trace identification.
	ServiceName string
	// Enabled controls whether tracing is active.
	Enabled bool
}

// DefaultTracingConfig returns default tracing configuration.
func DefaultTracingConfig() TracingConfig {
	return TracingConfig{
		ServiceName: "erp-backend",
		Enabled:     true,
	}
}

// Tracing returns OpenTelemetry tracing middleware with default configuration.
func Tracing() gin.HandlerFunc {
	return TracingWithConfig(DefaultTracingConfig())
}

// TracingWithConfig returns OpenTelemetry tracing middleware with custom configuration.
// This middleware wraps otelgin and adds custom span attributes:
//   - tenant_id: from JWT claims or X-Tenant-ID header
//   - user_id: from JWT claims
//   - request_id: from X-Request-ID header or generated
//
// The span name follows the format: "HTTP METHOD route_pattern" (e.g., "GET /api/v1/products/:id")
// Error responses (4xx/5xx) are marked with codes.Error status.
func TracingWithConfig(cfg TracingConfig) gin.HandlerFunc {
	if !cfg.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// Create the base otelgin middleware
	baseMiddleware := otelgin.Middleware(cfg.ServiceName)

	return func(c *gin.Context) {
		// Execute the base otelgin middleware first to create the span
		baseMiddleware(c)

		// After otelgin has created the span, enrich it with custom attributes
		span := trace.SpanFromContext(c.Request.Context())
		if span.IsRecording() {
			enrichSpanWithAttributes(c, span)
		}
	}
}

// enrichSpanWithAttributes adds custom attributes to the span from the request context.
func enrichSpanWithAttributes(c *gin.Context, span trace.Span) {
	// Add request_id attribute
	if requestID := getRequestID(c); requestID != "" {
		span.SetAttributes(attribute.String("request_id", requestID))
	}

	// Add tenant_id attribute (from JWT claims or header)
	if tenantID := getTenantID(c); tenantID != "" {
		span.SetAttributes(attribute.String("tenant_id", tenantID))
	}

	// Add user_id attribute (from JWT claims)
	if userID := getUserID(c); userID != "" {
		span.SetAttributes(attribute.String("user_id", userID))
	}
}

// getRequestID retrieves the request ID from the gin context or header.
// Header values are validated and truncated to prevent abuse.
func getRequestID(c *gin.Context) string {
	// First check gin context (set by RequestID middleware)
	if requestID, exists := c.Get("request_id"); exists {
		if id, ok := requestID.(string); ok && id != "" {
			return id
		}
	}

	// Fallback to header with length validation to prevent DoS
	headerID := c.GetHeader("X-Request-ID")
	if len(headerID) > MaxRequestIDLength {
		return headerID[:MaxRequestIDLength]
	}
	return headerID
}

// getTenantID retrieves the tenant ID from JWT claims or header.
// Header values are validated as UUIDs to prevent injection attacks.
func getTenantID(c *gin.Context) string {
	// First check JWT claims (set by JWT middleware) - trusted source
	if tenantID, exists := c.Get(JWTTenantIDKey); exists {
		if id, ok := tenantID.(string); ok && id != "" {
			return id
		}
	}

	// Fallback to header (for unauthenticated requests)
	// Validate format to prevent trace data injection
	headerTenantID := c.GetHeader("X-Tenant-ID")
	if headerTenantID != "" && isValidTenantID(headerTenantID) {
		return headerTenantID
	}
	return ""
}

// isValidTenantID validates that a tenant ID is a proper UUID format.
// This prevents injection of malicious data into trace attributes.
func isValidTenantID(tenantID string) bool {
	if len(tenantID) > MaxTenantIDLength {
		return false
	}
	return uuidRegex.MatchString(tenantID)
}

// getUserID retrieves the user ID from JWT claims.
func getUserID(c *gin.Context) string {
	if userID, exists := c.Get(JWTUserIDKey); exists {
		if id, ok := userID.(string); ok && id != "" {
			return id
		}
	}
	return ""
}

// SpanErrorMarker returns a middleware that marks spans with error status
// for HTTP error responses (4xx/5xx).
// This should be placed AFTER the Tracing middleware in the middleware chain.
func SpanErrorMarker() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// After the request is processed, check the response status
		span := trace.SpanFromContext(c.Request.Context())
		if !span.IsRecording() {
			return
		}

		statusCode := c.Writer.Status()

		// Mark error status for 4xx and 5xx responses
		if statusCode >= http.StatusBadRequest {
			var errorMessage string
			if statusCode >= http.StatusInternalServerError {
				errorMessage = "Internal Server Error"
			} else if statusCode == http.StatusUnauthorized {
				errorMessage = "Unauthorized"
			} else if statusCode == http.StatusForbidden {
				errorMessage = "Forbidden"
			} else if statusCode == http.StatusNotFound {
				errorMessage = "Not Found"
			} else {
				errorMessage = "Client Error"
			}

			span.SetStatus(codes.Error, errorMessage)
			span.SetAttributes(attribute.Int("http.status_code", statusCode))
		}
	}
}

// TracingAttributeInjector returns a middleware that injects custom attributes
// into the current span after authentication middleware has run.
// This should be placed AFTER both Tracing and JWT middleware in the chain.
func TracingAttributeInjector() gin.HandlerFunc {
	return func(c *gin.Context) {
		span := trace.SpanFromContext(c.Request.Context())
		if span.IsRecording() {
			enrichSpanWithAttributes(c, span)
		}
		c.Next()
	}
}
