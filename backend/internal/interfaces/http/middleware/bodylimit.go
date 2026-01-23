package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BodyLimit returns a middleware that limits request body size
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "REQUEST_TOO_LARGE",
					"message": "Request body exceeds maximum allowed size",
				},
			})
			return
		}

		// Wrap the body with a limited reader for streaming requests
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
