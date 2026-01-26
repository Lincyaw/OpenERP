package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestCORS(t *testing.T) {
	router := gin.New()
	router.Use(CORS())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	t.Run("rejects cross-origin request with empty whitelist default", func(t *testing.T) {
		// DefaultCORSConfig now uses empty whitelist for security
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://malicious.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// Empty whitelist should NOT set CORS headers for cross-origin requests
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("allows same-origin request with empty whitelist default", func(t *testing.T) {
		// Same-origin requests (no Origin header) should be allowed
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("handles OPTIONS preflight with empty whitelist", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "http://some-origin.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// OPTIONS should still return 204 but without CORS headers
		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

func TestCORSWithConfig(t *testing.T) {
	t.Run("allows specific origin", func(t *testing.T) {
		cfg := CORSConfig{
			AllowOrigins:     []string{"http://localhost:3000"},
			AllowMethods:     []string{"GET", "POST"},
			AllowHeaders:     []string{"Content-Type"},
			AllowCredentials: true,
		}

		router := gin.New()
		router.Use(CORSWithConfig(cfg))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("allows multiple specific origins", func(t *testing.T) {
		cfg := CORSConfig{
			AllowOrigins:     []string{"http://localhost:3000", "http://example.com"},
			AllowMethods:     []string{"GET", "POST"},
			AllowHeaders:     []string{"Content-Type"},
			AllowCredentials: true,
		}

		router := gin.New()
		router.Use(CORSWithConfig(cfg))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		// Test first origin
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))

		// Test second origin
		req = httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, "http://example.com", w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("rejects non-allowed origin", func(t *testing.T) {
		cfg := CORSConfig{
			AllowOrigins: []string{"http://allowed.com"},
		}

		router := gin.New()
		router.Use(CORSWithConfig(cfg))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://not-allowed.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Non-matching origin should not get CORS headers
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("empty whitelist rejects all cross-origin requests", func(t *testing.T) {
		cfg := CORSConfig{
			AllowOrigins: []string{}, // Empty whitelist - most secure
			AllowMethods: []string{"GET"},
			AllowHeaders: []string{"Content-Type"},
		}

		router := gin.New()
		router.Use(CORSWithConfig(cfg))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://any-origin.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Empty whitelist should NOT set any CORS headers
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("wildcard allows all origins", func(t *testing.T) {
		cfg := CORSConfig{
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{"GET", "POST"},
			AllowHeaders:     []string{"Content-Type"},
			AllowCredentials: false, // Note: credentials with wildcard is insecure
		}

		router := gin.New()
		router.Use(CORSWithConfig(cfg))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://any-origin.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		// Credentials should not be set with wildcard origin
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("credentials not set with wildcard origin even if configured", func(t *testing.T) {
		cfg := CORSConfig{
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{"GET"},
			AllowHeaders:     []string{"Content-Type"},
			AllowCredentials: true, // Should be ignored with wildcard
		}

		router := gin.New()
		router.Use(CORSWithConfig(cfg))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Credentials should NOT be set with wildcard (browser would reject it)
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("sets Max-Age header correctly", func(t *testing.T) {
		cfg := CORSConfig{
			AllowOrigins: []string{"http://localhost:3000"},
			AllowMethods: []string{"GET"},
			AllowHeaders: []string{"Content-Type"},
			MaxAge:       12 * time.Hour, // 43200 seconds
		}

		router := gin.New()
		router.Use(CORSWithConfig(cfg))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Max-Age should be "43200" (12 hours in seconds), NOT a Unicode character
		assert.Equal(t, "43200", w.Header().Get("Access-Control-Max-Age"))
	})

	t.Run("sets expose headers correctly", func(t *testing.T) {
		cfg := CORSConfig{
			AllowOrigins:  []string{"http://localhost:3000"},
			AllowMethods:  []string{"GET"},
			AllowHeaders:  []string{"Content-Type"},
			ExposeHeaders: []string{"X-Request-ID", "X-Custom-Header"},
		}

		router := gin.New()
		router.Use(CORSWithConfig(cfg))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, "X-Request-ID, X-Custom-Header", w.Header().Get("Access-Control-Expose-Headers"))
	})

	t.Run("handles OPTIONS preflight with allowed origin", func(t *testing.T) {
		cfg := CORSConfig{
			AllowOrigins: []string{"http://localhost:3000"},
			AllowMethods: []string{"GET", "POST", "PUT"},
			AllowHeaders: []string{"Content-Type", "Authorization"},
		}

		router := gin.New()
		router.Use(CORSWithConfig(cfg))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "GET, POST, PUT", w.Header().Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "Content-Type, Authorization", w.Header().Get("Access-Control-Allow-Headers"))
	})

	t.Run("handles OPTIONS preflight with disallowed origin", func(t *testing.T) {
		cfg := CORSConfig{
			AllowOrigins: []string{"http://allowed.com"},
			AllowMethods: []string{"GET", "POST"},
		}

		router := gin.New()
		router.Use(CORSWithConfig(cfg))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "http://not-allowed.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should still return 204 but without CORS headers
		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})
}

func TestRequestID(t *testing.T) {
	router := gin.New()
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, c.GetString("request_id"))
	})

	t.Run("generates request ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
		assert.NotEmpty(t, w.Body.String())
	})

	t.Run("uses provided request ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Request-ID", "test-request-id")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, "test-request-id", w.Header().Get("X-Request-ID"))
		assert.Equal(t, "test-request-id", w.Body.String())
	})
}

func TestSecure(t *testing.T) {
	t.Run("default configuration sets basic security headers and CSP/Permissions-Policy", func(t *testing.T) {
		router := gin.New()
		router.Use(Secure())
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Legacy security headers
		assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
		assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
		assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))

		// SEC-006: New security headers
		// CSP should be set with default directive
		csp := w.Header().Get("Content-Security-Policy")
		assert.NotEmpty(t, csp)
		assert.Contains(t, csp, "default-src 'self'")
		assert.Contains(t, csp, "frame-ancestors 'none'")

		// HSTS is disabled by default (requires HTTPS verification)
		assert.Empty(t, w.Header().Get("Strict-Transport-Security"))

		// Permissions-Policy should be set with restrictive defaults
		permPolicy := w.Header().Get("Permissions-Policy")
		assert.NotEmpty(t, permPolicy)
		assert.Contains(t, permPolicy, "camera=()")
		assert.Contains(t, permPolicy, "microphone=()")
	})
}

func TestSecureWithConfig(t *testing.T) {
	t.Run("custom CSP directive", func(t *testing.T) {
		cfg := SecurityConfig{
			CSPEnabled:               true,
			CSPDirective:             "default-src 'none'; script-src 'self'",
			PermissionsPolicyEnabled: false,
			HSTSEnabled:              false,
		}

		router := gin.New()
		router.Use(SecureWithConfig(cfg))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, "default-src 'none'; script-src 'self'", w.Header().Get("Content-Security-Policy"))
		assert.Empty(t, w.Header().Get("Permissions-Policy"))
		assert.Empty(t, w.Header().Get("Strict-Transport-Security"))
	})

	t.Run("HSTS enabled with all options", func(t *testing.T) {
		cfg := SecurityConfig{
			HSTSEnabled:              true,
			HSTSMaxAge:               63072000, // 2 years
			HSTSIncludeSubdomains:    true,
			HSTSPreload:              true,
			CSPEnabled:               false,
			PermissionsPolicyEnabled: false,
		}

		router := gin.New()
		router.Use(SecureWithConfig(cfg))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		hsts := w.Header().Get("Strict-Transport-Security")
		assert.Equal(t, "max-age=63072000; includeSubDomains; preload", hsts)
	})

	t.Run("HSTS enabled without optional flags", func(t *testing.T) {
		cfg := SecurityConfig{
			HSTSEnabled:              true,
			HSTSMaxAge:               31536000, // 1 year
			HSTSIncludeSubdomains:    false,
			HSTSPreload:              false,
			CSPEnabled:               false,
			PermissionsPolicyEnabled: false,
		}

		router := gin.New()
		router.Use(SecureWithConfig(cfg))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		hsts := w.Header().Get("Strict-Transport-Security")
		assert.Equal(t, "max-age=31536000", hsts)
	})

	t.Run("custom Permissions-Policy directive", func(t *testing.T) {
		cfg := SecurityConfig{
			PermissionsPolicyEnabled:   true,
			PermissionsPolicyDirective: "geolocation=(self), microphone=()",
			CSPEnabled:                 false,
			HSTSEnabled:                false,
		}

		router := gin.New()
		router.Use(SecureWithConfig(cfg))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, "geolocation=(self), microphone=()", w.Header().Get("Permissions-Policy"))
	})

	t.Run("all headers disabled", func(t *testing.T) {
		cfg := SecurityConfig{
			HSTSEnabled:              false,
			CSPEnabled:               false,
			PermissionsPolicyEnabled: false,
		}

		router := gin.New()
		router.Use(SecureWithConfig(cfg))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Basic security headers should still be present
		assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
		assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))

		// New headers should be absent
		assert.Empty(t, w.Header().Get("Content-Security-Policy"))
		assert.Empty(t, w.Header().Get("Strict-Transport-Security"))
		assert.Empty(t, w.Header().Get("Permissions-Policy"))
	})

	t.Run("all headers enabled with full config", func(t *testing.T) {
		cfg := SecurityConfig{
			HSTSEnabled:                true,
			HSTSMaxAge:                 31536000,
			HSTSIncludeSubdomains:      true,
			HSTSPreload:                false,
			CSPEnabled:                 true,
			CSPDirective:               "default-src 'self'",
			PermissionsPolicyEnabled:   true,
			PermissionsPolicyDirective: "camera=(), microphone=()",
		}

		router := gin.New()
		router.Use(SecureWithConfig(cfg))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// All headers should be present
		assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
		assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "default-src 'self'", w.Header().Get("Content-Security-Policy"))
		assert.Equal(t, "max-age=31536000; includeSubDomains", w.Header().Get("Strict-Transport-Security"))
		assert.Equal(t, "camera=(), microphone=()", w.Header().Get("Permissions-Policy"))
	})
}

func TestDefaultSecurityConfig(t *testing.T) {
	cfg := DefaultSecurityConfig()

	// HSTS should be disabled by default (requires HTTPS setup)
	assert.False(t, cfg.HSTSEnabled)
	assert.Equal(t, 31536000, cfg.HSTSMaxAge) // 1 year default
	assert.True(t, cfg.HSTSIncludeSubdomains)
	assert.False(t, cfg.HSTSPreload)

	// CSP should be enabled with secure defaults
	assert.True(t, cfg.CSPEnabled)
	assert.NotEmpty(t, cfg.CSPDirective)
	assert.Contains(t, cfg.CSPDirective, "default-src 'self'")
	assert.Contains(t, cfg.CSPDirective, "frame-ancestors 'none'")

	// Permissions-Policy should be enabled with restrictive defaults
	assert.True(t, cfg.PermissionsPolicyEnabled)
	assert.NotEmpty(t, cfg.PermissionsPolicyDirective)
	assert.Contains(t, cfg.PermissionsPolicyDirective, "camera=()")
	assert.Contains(t, cfg.PermissionsPolicyDirective, "microphone=()")
}

func TestTimeout(t *testing.T) {
	router := gin.New()
	router.Use(Timeout(30 * time.Second))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, "30s", w.Header().Get("X-Request-Timeout"))
}

func TestGenerateRequestID(t *testing.T) {
	id1 := generateRequestID()
	id2 := generateRequestID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Len(t, id1, 32) // 16 bytes hex encoded = 32 chars
}

func TestDefaultCORSConfig(t *testing.T) {
	cfg := DefaultCORSConfig()

	// SEC-005: DefaultCORSConfig now uses empty whitelist for security
	assert.Empty(t, cfg.AllowOrigins, "DefaultCORSConfig should have empty AllowOrigins for security")
	assert.Contains(t, cfg.AllowMethods, "GET")
	assert.Contains(t, cfg.AllowMethods, "POST")
	assert.Contains(t, cfg.AllowHeaders, "Content-Type")
	assert.Contains(t, cfg.AllowHeaders, "Authorization")
	assert.True(t, cfg.AllowCredentials)
	assert.Equal(t, 12*time.Hour, cfg.MaxAge)
}

// TestMaxAgeHeaderFormat specifically tests the fix for the string(rune()) bug
func TestMaxAgeHeaderFormat(t *testing.T) {
	testCases := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"1 hour", 1 * time.Hour, "3600"},
		{"12 hours", 12 * time.Hour, "43200"},
		{"24 hours", 24 * time.Hour, "86400"},
		{"1 minute", 1 * time.Minute, "60"},
		{"30 seconds", 30 * time.Second, "30"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := CORSConfig{
				AllowOrigins: []string{"http://localhost:3000"},
				AllowMethods: []string{"GET"},
				AllowHeaders: []string{"Content-Type"},
				MaxAge:       tc.duration,
			}

			router := gin.New()
			router.Use(CORSWithConfig(cfg))
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "ok")
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", "http://localhost:3000")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Verify Max-Age is a proper decimal string, not a Unicode character
			maxAge := w.Header().Get("Access-Control-Max-Age")
			assert.Equal(t, tc.expected, maxAge, "Max-Age should be decimal string, not Unicode character")

			// Verify it's not a single character (which would indicate the bug)
			assert.Greater(t, len(maxAge), 1, "Max-Age should not be a single character")
		})
	}
}
