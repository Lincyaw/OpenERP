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

	t.Run("sets CORS headers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("handles OPTIONS preflight", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

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
	router := gin.New()
	router.Use(Secure())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
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

	assert.Contains(t, cfg.AllowOrigins, "*")
	assert.Contains(t, cfg.AllowMethods, "GET")
	assert.Contains(t, cfg.AllowMethods, "POST")
	assert.Contains(t, cfg.AllowHeaders, "Content-Type")
	assert.Contains(t, cfg.AllowHeaders, "Authorization")
	assert.True(t, cfg.AllowCredentials)
	assert.Equal(t, 12*time.Hour, cfg.MaxAge)
}
