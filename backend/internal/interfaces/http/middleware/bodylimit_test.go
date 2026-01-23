package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestBodyLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("allows request within limit", func(t *testing.T) {
		router := gin.New()
		router.Use(BodyLimit(1024)) // 1KB limit
		router.POST("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		body := bytes.NewReader([]byte("small body"))
		req := httptest.NewRequest("POST", "/test", body)
		req.Header.Set("Content-Length", "10")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("rejects request exceeding Content-Length limit", func(t *testing.T) {
		router := gin.New()
		router.Use(BodyLimit(100)) // 100 bytes limit
		router.POST("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		body := bytes.NewReader([]byte(strings.Repeat("x", 200)))
		req := httptest.NewRequest("POST", "/test", body)
		req.ContentLength = 200
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
		assert.Contains(t, w.Body.String(), "REQUEST_TOO_LARGE")
	})

	t.Run("allows GET requests", func(t *testing.T) {
		router := gin.New()
		router.Use(BodyLimit(10))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("uses MaxBytesReader for streaming protection", func(t *testing.T) {
		router := gin.New()
		router.Use(BodyLimit(50))
		router.POST("/test", func(c *gin.Context) {
			// Try to read the body
			buf := make([]byte, 200)
			_, err := c.Request.Body.Read(buf)
			if err != nil {
				c.String(http.StatusBadRequest, "body too large")
				return
			}
			c.String(http.StatusOK, "ok")
		})

		// Request without Content-Length header (streaming)
		body := strings.NewReader(strings.Repeat("x", 100))
		req := httptest.NewRequest("POST", "/test", body)
		// Don't set Content-Length to simulate streaming
		req.ContentLength = -1
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should hit the MaxBytesReader limit when reading
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
