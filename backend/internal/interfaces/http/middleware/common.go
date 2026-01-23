package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
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
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins:     []string{"*"},
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
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		allowedOrigin := "*"
		if len(cfg.AllowOrigins) > 0 && cfg.AllowOrigins[0] != "*" {
			for _, o := range cfg.AllowOrigins {
				if o == origin {
					allowedOrigin = origin
					break
				}
			}
			// If specific origins are set but none match, don't set CORS headers
			if allowedOrigin == "*" && origin != "" {
				c.Next()
				return
			}
		}

		c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		if cfg.AllowCredentials && allowedOrigin != "*" {
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		c.Writer.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowHeaders, ", "))
		c.Writer.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowMethods, ", "))

		if len(cfg.ExposeHeaders) > 0 {
			c.Writer.Header().Set("Access-Control-Expose-Headers", strings.Join(cfg.ExposeHeaders, ", "))
		}

		if cfg.MaxAge > 0 {
			c.Writer.Header().Set("Access-Control-Max-Age", string(rune(int(cfg.MaxAge.Seconds()))))
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
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

// Secure adds security headers to responses
func Secure() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent clickjacking
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		// XSS protection
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")
		// Prevent MIME type sniffing
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		// Referrer policy
		c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

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
