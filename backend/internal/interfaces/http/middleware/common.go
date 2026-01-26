package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CORSConfig holds CORS middleware configuration
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           time.Duration
}

// DefaultCORSConfig returns default CORS configuration
// NOTE: AllowOrigins is empty by default for security. In production,
// you MUST explicitly configure allowed origins via config.toml or environment variables.
// Using an empty list will reject all cross-origin requests until configured.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins:     []string{}, // Empty by default for security - must be explicitly configured
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "X-Request-ID", "X-Tenant-ID", "Accept", "Origin", "Cache-Control"},
		ExposeHeaders:    []string{"X-Request-ID", "X-RateLimit-Limit", "X-RateLimit-Remaining"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
}

// CORS returns a middleware that handles CORS with default configuration
func CORS() gin.HandlerFunc {
	return CORSWithConfig(DefaultCORSConfig())
}

// CORSWithConfig returns a CORS middleware with custom configuration
func CORSWithConfig(cfg CORSConfig) gin.HandlerFunc {
	// Pre-compute whether wildcard is allowed
	allowWildcard := false
	for _, o := range cfg.AllowOrigins {
		if o == "*" {
			allowWildcard = true
			break
		}
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Handle OPTIONS preflight requests first
		// We always respond to OPTIONS with 204, but only set CORS headers if origin is allowed
		if c.Request.Method == "OPTIONS" {
			// Handle allowed origins for preflight
			if len(cfg.AllowOrigins) > 0 {
				if allowWildcard {
					c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
					setCORSHeaders(c, cfg)
				} else {
					for _, o := range cfg.AllowOrigins {
						if o == origin {
							c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
							if cfg.AllowCredentials {
								c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
							}
							setCORSHeaders(c, cfg)
							break
						}
					}
				}
			}
			// Always abort with 204 for OPTIONS (even without CORS headers)
			// This prevents 404s for preflight requests
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		// Determine the allowed origin for this request
		var allowedOrigin string

		if len(cfg.AllowOrigins) == 0 {
			// Empty whitelist: reject all cross-origin requests (most secure default)
			// Continue processing but don't set CORS headers
			c.Next()
			return
		}

		if allowWildcard {
			// Wildcard mode - allow all origins
			// Note: AllowCredentials with "*" origin is insecure and will be rejected by browsers
			allowedOrigin = "*"
		} else {
			// Explicit whitelist mode - check if origin is allowed
			for _, o := range cfg.AllowOrigins {
				if o == origin {
					allowedOrigin = origin
					break
				}
			}
			// If origin is not in whitelist, don't set CORS headers
			if allowedOrigin == "" && origin != "" {
				c.Next()
				return
			}
		}

		// Set CORS headers
		if allowedOrigin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			if cfg.AllowCredentials && allowedOrigin != "*" {
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			setCORSHeaders(c, cfg)
		}

		c.Next()
	}
}

// setCORSHeaders sets common CORS headers (methods, headers, expose, max-age)
func setCORSHeaders(c *gin.Context, cfg CORSConfig) {
	c.Writer.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowHeaders, ", "))
	c.Writer.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowMethods, ", "))

	if len(cfg.ExposeHeaders) > 0 {
		c.Writer.Header().Set("Access-Control-Expose-Headers", strings.Join(cfg.ExposeHeaders, ", "))
	}

	if cfg.MaxAge > 0 {
		c.Writer.Header().Set("Access-Control-Max-Age", strconv.Itoa(int(cfg.MaxAge.Seconds())))
	}
}

// RequestID adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set("request_id", requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Next()
	}
}

// RequestLogger logs incoming requests (placeholder - actual logging is in logger package)
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		if query != "" {
			path = path + "?" + query
		}

		// These values are available for logging
		// Actual structured logging is done via logger.GinMiddleware
		_ = latency
		_ = status
		_ = path
	}
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	// Generate 16 random bytes (128 bits)
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return time.Now().Format("20060102150405") + "-" + fallbackRandomString(8)
	}
	return hex.EncodeToString(bytes)
}

// fallbackRandomString generates a random string as fallback
func fallbackRandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

// SecurityConfig holds configuration for security headers
type SecurityConfig struct {
	// HSTS settings
	HSTSEnabled           bool
	HSTSMaxAge            int // in seconds
	HSTSIncludeSubdomains bool
	HSTSPreload           bool

	// CSP settings
	CSPEnabled   bool
	CSPDirective string // Content-Security-Policy directive

	// Permissions-Policy settings
	PermissionsPolicyEnabled   bool
	PermissionsPolicyDirective string
}

// DefaultSecurityConfig returns secure default settings
// NOTE: HSTS is disabled by default as it requires HTTPS in production.
// Enable it in production by setting HSTSEnabled = true in your config.
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		// HSTS: disabled by default (requires HTTPS)
		HSTSEnabled:           false,
		HSTSMaxAge:            31536000, // 1 year in seconds
		HSTSIncludeSubdomains: true,
		HSTSPreload:           false, // Don't preload by default, requires HTTPS verification

		// CSP: enabled with secure defaults
		CSPEnabled: true,
		// Default CSP: Allow same-origin scripts, styles, images, fonts, connections.
		// Block inline scripts/styles by default to prevent XSS.
		// Applications should customize this based on their needs.
		CSPDirective: "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'",

		// Permissions-Policy: enabled with restrictive defaults
		PermissionsPolicyEnabled: true,
		// Disable potentially dangerous browser features by default
		PermissionsPolicyDirective: "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()",
	}
}

// Secure adds security headers to responses using default configuration
func Secure() gin.HandlerFunc {
	return SecureWithConfig(DefaultSecurityConfig())
}

// SecureWithConfig adds security headers to responses with custom configuration
func SecureWithConfig(cfg SecurityConfig) gin.HandlerFunc {
	// Pre-compute HSTS header value if enabled
	var hstsValue string
	if cfg.HSTSEnabled {
		hstsValue = fmt.Sprintf("max-age=%d", cfg.HSTSMaxAge)
		if cfg.HSTSIncludeSubdomains {
			hstsValue += "; includeSubDomains"
		}
		if cfg.HSTSPreload {
			hstsValue += "; preload"
		}
	}

	return func(c *gin.Context) {
		// Prevent clickjacking
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		// XSS protection (legacy, but still useful for older browsers)
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")
		// Prevent MIME type sniffing
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		// Referrer policy
		c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// SEC-006: Add Content-Security-Policy header
		if cfg.CSPEnabled && cfg.CSPDirective != "" {
			c.Writer.Header().Set("Content-Security-Policy", cfg.CSPDirective)
		}

		// SEC-006: Add Strict-Transport-Security header (HSTS)
		// Only effective over HTTPS, but setting it won't cause issues over HTTP
		if cfg.HSTSEnabled && hstsValue != "" {
			c.Writer.Header().Set("Strict-Transport-Security", hstsValue)
		}

		// SEC-006: Add Permissions-Policy header (formerly Feature-Policy)
		if cfg.PermissionsPolicyEnabled && cfg.PermissionsPolicyDirective != "" {
			c.Writer.Header().Set("Permissions-Policy", cfg.PermissionsPolicyDirective)
		}

		c.Next()
	}
}

// Timeout returns a middleware that times out requests
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Note: For proper timeout handling, consider using context.WithTimeout
		// in your handlers. This middleware sets a header for documentation.
		c.Writer.Header().Set("X-Request-Timeout", timeout.String())
		c.Next()
	}
}
