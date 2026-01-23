package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter implements a simple in-memory rate limiter using token bucket algorithm
type RateLimiter struct {
	mu          sync.Mutex
	clients     map[string]*client
	limit       int           // Maximum requests per window
	window      time.Duration // Time window
	cleanupTick time.Duration // Cleanup interval
}

type client struct {
	tokens    int
	lastReset time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		clients:     make(map[string]*client),
		limit:       limit,
		window:      window,
		cleanupTick: window * 2, // Cleanup every 2 windows
	}
	go rl.cleanup()
	return rl
}

// cleanup removes expired clients periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupTick)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, c := range rl.clients {
			if now.Sub(c.lastReset) > rl.window*2 {
				delete(rl.clients, key)
			}
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request from the given key should be allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	c, exists := rl.clients[key]

	if !exists {
		rl.clients[key] = &client{
			tokens:    rl.limit - 1,
			lastReset: now,
		}
		return true
	}

	// Reset tokens if window has passed
	if now.Sub(c.lastReset) >= rl.window {
		c.tokens = rl.limit - 1
		c.lastReset = now
		return true
	}

	// Check if tokens are available
	if c.tokens > 0 {
		c.tokens--
		return true
	}

	return false
}

// Remaining returns the number of remaining requests for the given key
func (rl *RateLimiter) Remaining(key string) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	c, exists := rl.clients[key]
	if !exists {
		return rl.limit
	}

	if time.Since(c.lastReset) >= rl.window {
		return rl.limit
	}

	return c.tokens
}

// RateLimit returns a rate limiting middleware
func RateLimit(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use client IP as rate limit key
		key := c.ClientIP()

		// Add tenant ID to key if available for per-tenant limits
		if tenantID := c.GetHeader("X-Tenant-ID"); tenantID != "" {
			key = tenantID + ":" + key
		}

		if !limiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Too many requests. Please try again later.",
				},
			})
			return
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", limiter.Remaining(key)))

		c.Next()
	}
}

// RateLimitByKey returns a rate limiting middleware with custom key extractor
func RateLimitByKey(limiter *RateLimiter, keyFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := keyFunc(c)

		if !limiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Too many requests. Please try again later.",
				},
			})
			return
		}

		c.Next()
	}
}
