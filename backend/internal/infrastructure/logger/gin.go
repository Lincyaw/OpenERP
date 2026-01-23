package logger

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GinMiddleware returns a gin middleware that logs HTTP requests
func GinMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Get request ID from context (set by RequestID middleware)
		requestID, _ := c.Get("request_id")
		requestIDStr, _ := requestID.(string)

		// Create request-scoped logger
		reqLogger := logger.With(
			zap.String("request_id", requestIDStr),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
		)

		// Store logger in gin context for handlers to use
		c.Set("logger", reqLogger)

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()
		bodySize := c.Writer.Size()

		// Build log fields
		fields := []zap.Field{
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("client_ip", clientIP),
			zap.String("user_agent", userAgent),
			zap.Int("body_size", bodySize),
		}

		if query != "" {
			fields = append(fields, zap.String("query", query))
		}

		// Log errors if any
		if len(c.Errors) > 0 {
			fields = append(fields, zap.Strings("errors", c.Errors.Errors()))
		}

		// Choose log level based on status code
		msg := "HTTP Request"
		switch {
		case status >= 500:
			reqLogger.Error(msg, fields...)
		case status >= 400:
			reqLogger.Warn(msg, fields...)
		default:
			reqLogger.Info(msg, fields...)
		}
	}
}

// Recovery returns a gin middleware that recovers from panics and logs them
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Get request ID if available
				requestID, _ := c.Get("request_id")
				requestIDStr, _ := requestID.(string)

				logger.Error("Panic recovered",
					zap.String("request_id", requestIDStr),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.Any("error", err),
					zap.Stack("stacktrace"),
				)

				c.AbortWithStatus(500)
			}
		}()
		c.Next()
	}
}

// GetGinLogger retrieves the logger from gin context
func GetGinLogger(c *gin.Context) *zap.Logger {
	if logger, exists := c.Get("logger"); exists {
		if l, ok := logger.(*zap.Logger); ok {
			return l
		}
	}
	return zap.NewNop()
}
