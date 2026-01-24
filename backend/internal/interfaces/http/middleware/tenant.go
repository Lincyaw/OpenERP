package middleware

import (
	"net/http"
	"strings"

	"github.com/erp/backend/internal/infrastructure/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// TenantContextKey is the key used to store tenant information in gin.Context
const (
	TenantIDKey     = "tenant_id"
	TenantCodeKey   = "tenant_code"
	TenantHeaderKey = "X-Tenant-ID"
)

// TenantInfo holds the extracted tenant information
type TenantInfo struct {
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code"`
}

// TenantExtractor defines the interface for extracting tenant information
type TenantExtractor interface {
	ExtractTenantID(c *gin.Context) (string, error)
}

// TenantValidator defines the interface for validating tenant
type TenantValidator interface {
	ValidateTenant(tenantID string) (*TenantInfo, error)
}

// TenantMiddlewareConfig holds configuration for tenant middleware
type TenantMiddlewareConfig struct {
	// HeaderEnabled enables X-Tenant-ID header extraction
	HeaderEnabled bool
	// JWTEnabled enables JWT claim extraction (requires JWT middleware to run first)
	JWTEnabled bool
	// SubdomainEnabled enables subdomain extraction
	SubdomainEnabled bool
	// BaseDomain is the base domain for subdomain extraction (e.g., "erp.com")
	BaseDomain string
	// SkipPaths are paths that don't require tenant context (e.g., health check)
	SkipPaths []string
	// Required determines if tenant context is mandatory
	Required bool
	// Validator is an optional validator to check if tenant exists and is active
	Validator TenantValidator
	// Logger for middleware logging
	Logger *zap.Logger
}

// DefaultTenantConfig returns default tenant middleware configuration
func DefaultTenantConfig() TenantMiddlewareConfig {
	return TenantMiddlewareConfig{
		HeaderEnabled:    true,
		JWTEnabled:       true,
		SubdomainEnabled: false,
		BaseDomain:       "",
		SkipPaths:        []string{"/health", "/healthz", "/ready", "/metrics", "/api/v1/health"},
		Required:         true,
		Validator:        nil,
		Logger:           nil,
	}
}

// TenantMiddleware extracts tenant information from the request
// Extraction order: JWT claims > X-Tenant-ID header > subdomain
func TenantMiddleware() gin.HandlerFunc {
	return TenantMiddlewareWithConfig(DefaultTenantConfig())
}

// TenantMiddlewareWithConfig returns tenant middleware with custom configuration
func TenantMiddlewareWithConfig(cfg TenantMiddlewareConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if path should be skipped
		path := c.Request.URL.Path
		for _, skipPath := range cfg.SkipPaths {
			if path == skipPath || strings.HasPrefix(path, skipPath+"/") {
				c.Next()
				return
			}
		}

		var tenantID string
		var extractionMethod string

		// Priority 1: JWT claims (if JWT middleware has already run)
		if cfg.JWTEnabled {
			if jwtTenantID, exists := c.Get("jwt_tenant_id"); exists {
				if tid, ok := jwtTenantID.(string); ok && tid != "" {
					tenantID = tid
					extractionMethod = "jwt"
				}
			}
		}

		// Priority 2: X-Tenant-ID header
		if tenantID == "" && cfg.HeaderEnabled {
			if headerTenantID := c.GetHeader(TenantHeaderKey); headerTenantID != "" {
				tenantID = headerTenantID
				extractionMethod = "header"
			}
		}

		// Priority 3: Subdomain extraction
		if tenantID == "" && cfg.SubdomainEnabled && cfg.BaseDomain != "" {
			if subdomainTenantID := extractTenantFromSubdomain(c.Request.Host, cfg.BaseDomain); subdomainTenantID != "" {
				tenantID = subdomainTenantID
				extractionMethod = "subdomain"
			}
		}

		// Validate tenant ID format if present
		if tenantID != "" {
			if err := validateTenantIDFormat(tenantID); err != nil {
				respondUnauthorized(c, "Invalid tenant ID format")
				return
			}
		}

		// Check if tenant is required
		if tenantID == "" && cfg.Required {
			respondUnauthorized(c, "Tenant identification required")
			return
		}

		// Optional: Validate tenant exists and is active
		var tenantInfo *TenantInfo
		if tenantID != "" && cfg.Validator != nil {
			var err error
			tenantInfo, err = cfg.Validator.ValidateTenant(tenantID)
			if err != nil {
				log := cfg.Logger
				if log == nil {
					log = logger.FromContext(c.Request.Context())
				}
				log.Warn("Tenant validation failed",
					zap.String("tenant_id", tenantID),
					zap.Error(err),
				)
				respondUnauthorized(c, "Invalid or inactive tenant")
				return
			}
		}

		// Set tenant information in context
		if tenantID != "" {
			// Set in gin context for easy access in handlers
			c.Set(TenantIDKey, tenantID)
			if tenantInfo != nil {
				c.Set(TenantCodeKey, tenantInfo.Code)
			}

			// Set in request context for service layer access
			ctx := c.Request.Context()
			log := logger.FromContext(ctx)
			ctx, _ = logger.WithTenantID(ctx, log, tenantID)
			c.Request = c.Request.WithContext(ctx)

			// Log extraction method for debugging
			if cfg.Logger != nil {
				cfg.Logger.Debug("Tenant identified",
					zap.String("tenant_id", tenantID),
					zap.String("method", extractionMethod),
				)
			}
		}

		c.Next()
	}
}

// extractTenantFromSubdomain extracts tenant code from subdomain
// e.g., "acme.erp.com" with baseDomain "erp.com" returns "acme"
func extractTenantFromSubdomain(host, baseDomain string) string {
	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Check if host ends with base domain
	if !strings.HasSuffix(host, baseDomain) {
		return ""
	}

	// Extract subdomain
	subdomain := strings.TrimSuffix(host, "."+baseDomain)
	if subdomain == host || subdomain == "" || subdomain == "www" {
		return ""
	}

	// Return the first part of subdomain (in case of multi-level subdomains)
	parts := strings.Split(subdomain, ".")
	return parts[0]
}

// validateTenantIDFormat validates that the tenant ID is a valid UUID
func validateTenantIDFormat(tenantID string) error {
	_, err := uuid.Parse(tenantID)
	return err
}

// respondUnauthorized sends an unauthorized response
func respondUnauthorized(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "UNAUTHORIZED",
			"message": message,
		},
	})
}

// GetTenantID retrieves the tenant ID from gin.Context
func GetTenantID(c *gin.Context) string {
	if tenantID, exists := c.Get(TenantIDKey); exists {
		if tid, ok := tenantID.(string); ok {
			return tid
		}
	}
	return ""
}

// GetTenantUUID retrieves the tenant ID as UUID from gin.Context
func GetTenantUUID(c *gin.Context) (uuid.UUID, error) {
	tenantID := GetTenantID(c)
	if tenantID == "" {
		return uuid.Nil, nil
	}
	return uuid.Parse(tenantID)
}

// GetTenantCode retrieves the tenant code from gin.Context
func GetTenantCode(c *gin.Context) string {
	if tenantCode, exists := c.Get(TenantCodeKey); exists {
		if code, ok := tenantCode.(string); ok {
			return code
		}
	}
	return ""
}

// MustGetTenantID retrieves the tenant ID from gin.Context or panics if not found
// Use this only in handlers where tenant is guaranteed to exist
func MustGetTenantID(c *gin.Context) string {
	tenantID := GetTenantID(c)
	if tenantID == "" {
		panic("tenant_id not found in context")
	}
	return tenantID
}

// MustGetTenantUUID retrieves the tenant ID as UUID or panics if not found
func MustGetTenantUUID(c *gin.Context) uuid.UUID {
	tenantUUID, err := GetTenantUUID(c)
	if err != nil || tenantUUID == uuid.Nil {
		panic("valid tenant_id not found in context")
	}
	return tenantUUID
}

// OptionalTenantMiddleware creates middleware that doesn't require tenant
func OptionalTenantMiddleware() gin.HandlerFunc {
	cfg := DefaultTenantConfig()
	cfg.Required = false
	return TenantMiddlewareWithConfig(cfg)
}
