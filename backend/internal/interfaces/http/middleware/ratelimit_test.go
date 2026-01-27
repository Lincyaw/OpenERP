package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		limiter := NewRateLimiter(5, time.Minute)

		for i := 0; i < 5; i++ {
			assert.True(t, limiter.Allow("client1"), "request %d should be allowed", i+1)
		}
	})

	t.Run("blocks requests exceeding limit", func(t *testing.T) {
		limiter := NewRateLimiter(3, time.Minute)

		// Use up all tokens
		for i := 0; i < 3; i++ {
			assert.True(t, limiter.Allow("client2"))
		}

		// Next request should be blocked
		assert.False(t, limiter.Allow("client2"))
	})

	t.Run("separate limits per client", func(t *testing.T) {
		limiter := NewRateLimiter(2, time.Minute)

		assert.True(t, limiter.Allow("clientA"))
		assert.True(t, limiter.Allow("clientA"))
		assert.False(t, limiter.Allow("clientA"))

		// clientB should still have tokens
		assert.True(t, limiter.Allow("clientB"))
		assert.True(t, limiter.Allow("clientB"))
	})

	t.Run("resets after window", func(t *testing.T) {
		limiter := NewRateLimiter(2, 50*time.Millisecond)

		assert.True(t, limiter.Allow("client3"))
		assert.True(t, limiter.Allow("client3"))
		assert.False(t, limiter.Allow("client3"))

		// Wait for window to pass
		time.Sleep(60 * time.Millisecond)

		// Should be allowed again
		assert.True(t, limiter.Allow("client3"))
	})

	t.Run("remaining returns correct count", func(t *testing.T) {
		limiter := NewRateLimiter(5, time.Minute)

		assert.Equal(t, 5, limiter.Remaining("newclient"))

		limiter.Allow("newclient") // 4 remaining
		limiter.Allow("newclient") // 3 remaining

		assert.Equal(t, 3, limiter.Remaining("newclient"))
	})

	t.Run("concurrent access is safe", func(t *testing.T) {
		limiter := NewRateLimiter(100, time.Minute)
		var wg sync.WaitGroup
		allowed := 0
		var mu sync.Mutex

		for i := 0; i < 150; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if limiter.Allow("concurrent-client") {
					mu.Lock()
					allowed++
					mu.Unlock()
				}
			}()
		}

		wg.Wait()
		assert.Equal(t, 100, allowed)
	})
}

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("allows requests within limit", func(t *testing.T) {
		limiter := NewRateLimiter(3, time.Minute)
		router := gin.New()
		router.Use(RateLimit(limiter))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		for i := 0; i < 3; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}
	})

	t.Run("returns 429 when limit exceeded", func(t *testing.T) {
		limiter := NewRateLimiter(2, time.Minute)
		router := gin.New()
		router.Use(RateLimit(limiter))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		// Use up tokens
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}

		// Next request should be rate limited
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Contains(t, w.Body.String(), "RATE_LIMIT_EXCEEDED")
	})

	t.Run("uses tenant ID in rate limit key", func(t *testing.T) {
		limiter := NewRateLimiter(1, time.Minute)
		router := gin.New()
		router.Use(RateLimit(limiter))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		// First request for tenant1
		req1 := httptest.NewRequest("GET", "/test", nil)
		req1.Header.Set("X-Tenant-ID", "tenant1")
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)

		// Second request for tenant1 should be limited
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.Header.Set("X-Tenant-ID", "tenant1")
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusTooManyRequests, w2.Code)

		// Request for tenant2 should still work
		req3 := httptest.NewRequest("GET", "/test", nil)
		req3.Header.Set("X-Tenant-ID", "tenant2")
		w3 := httptest.NewRecorder()
		router.ServeHTTP(w3, req3)
		assert.Equal(t, http.StatusOK, w3.Code)
	})
}

func TestRateLimitByKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("uses custom key function", func(t *testing.T) {
		limiter := NewRateLimiter(1, time.Minute)
		keyFunc := func(c *gin.Context) string {
			return c.GetHeader("X-User-ID")
		}

		router := gin.New()
		router.Use(RateLimitByKey(limiter, keyFunc))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		// First request for user1
		req1 := httptest.NewRequest("GET", "/test", nil)
		req1.Header.Set("X-User-ID", "user1")
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)

		// Second request for user1 should be limited
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.Header.Set("X-User-ID", "user1")
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusTooManyRequests, w2.Code)
	})
}

func TestAuthRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("allows requests within auth limit", func(t *testing.T) {
		limiter := NewRateLimiter(5, time.Minute) // 5 attempts per minute
		router := gin.New()
		router.Use(AuthRateLimit(limiter))
		router.POST("/login", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"success": true})
		})

		// Should allow 5 requests
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("POST", "/login", nil)
			req.RemoteAddr = "192.168.1.100:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code, "request %d should be allowed", i+1)
		}
	})

	t.Run("returns 429 with AUTH_RATE_LIMIT_EXCEEDED when auth limit exceeded", func(t *testing.T) {
		limiter := NewRateLimiter(3, time.Minute)
		router := gin.New()
		router.Use(AuthRateLimit(limiter))
		router.POST("/login", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"success": true})
		})

		// Use up all tokens
		for i := 0; i < 3; i++ {
			req := httptest.NewRequest("POST", "/login", nil)
			req.RemoteAddr = "192.168.1.100:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}

		// Next request should be rate limited with AUTH-specific error
		req := httptest.NewRequest("POST", "/login", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Contains(t, w.Body.String(), "AUTH_RATE_LIMIT_EXCEEDED")
		assert.Contains(t, w.Body.String(), "Too many authentication attempts")
	})

	t.Run("includes rate limit headers", func(t *testing.T) {
		limiter := NewRateLimiter(5, time.Minute)
		router := gin.New()
		router.Use(AuthRateLimit(limiter))
		router.POST("/login", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"success": true})
		})

		req := httptest.NewRequest("POST", "/login", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "5", w.Header().Get("X-RateLimit-Limit"))
		assert.Equal(t, "4", w.Header().Get("X-RateLimit-Remaining"))
	})

	t.Run("includes Retry-After header when blocked", func(t *testing.T) {
		limiter := NewRateLimiter(1, time.Minute)
		router := gin.New()
		router.Use(AuthRateLimit(limiter))
		router.POST("/login", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"success": true})
		})

		// Use up token
		req1 := httptest.NewRequest("POST", "/login", nil)
		req1.RemoteAddr = "192.168.1.100:12345"
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		// Next request should have Retry-After header
		req2 := httptest.NewRequest("POST", "/login", nil)
		req2.RemoteAddr = "192.168.1.100:12345"
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusTooManyRequests, w2.Code)
		assert.Equal(t, "60", w2.Header().Get("Retry-After")) // 60 seconds = 1 minute
	})

	t.Run("separate limits per IP address", func(t *testing.T) {
		limiter := NewRateLimiter(2, time.Minute)
		router := gin.New()
		router.Use(AuthRateLimit(limiter))
		router.POST("/login", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"success": true})
		})

		// IP 1: Use up tokens
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("POST", "/login", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}

		// IP 1: Should be blocked
		req1 := httptest.NewRequest("POST", "/login", nil)
		req1.RemoteAddr = "192.168.1.1:12345"
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusTooManyRequests, w1.Code)

		// IP 2: Should still work
		req2 := httptest.NewRequest("POST", "/login", nil)
		req2.RemoteAddr = "192.168.1.2:12345"
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusOK, w2.Code)
	})

	t.Run("uses auth prefix in key to isolate from other rate limiters", func(t *testing.T) {
		// Create two separate limiters (simulating global and auth)
		globalLimiter := NewRateLimiter(100, time.Minute)
		authLimiter := NewRateLimiter(2, time.Minute)

		router := gin.New()

		// Auth route with auth limiter
		authGroup := router.Group("/auth")
		authGroup.Use(AuthRateLimit(authLimiter))
		authGroup.POST("/login", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"success": true})
		})

		// Other route with global limiter
		router.Use(RateLimit(globalLimiter))
		router.GET("/api/data", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": "test"})
		})

		// Exhaust auth limit
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("POST", "/auth/login", nil)
			req.RemoteAddr = "192.168.1.100:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}

		// Auth should be blocked
		req1 := httptest.NewRequest("POST", "/auth/login", nil)
		req1.RemoteAddr = "192.168.1.100:12345"
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusTooManyRequests, w1.Code)

		// Other API should still work (different limiter)
		req2 := httptest.NewRequest("GET", "/api/data", nil)
		req2.RemoteAddr = "192.168.1.100:12345"
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusOK, w2.Code)
	})
}
