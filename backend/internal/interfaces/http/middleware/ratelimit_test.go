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
