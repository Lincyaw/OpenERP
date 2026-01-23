package logger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestGinMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	core, recorded := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(core)

	router := gin.New()
	router.Use(GinMiddleware(zapLogger))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check that request was logged
	logs := recorded.All()
	require.NotEmpty(t, logs)

	// Find the HTTP Request log
	var httpLog *observer.LoggedEntry
	for i := range logs {
		if logs[i].Message == "HTTP Request" {
			httpLog = &logs[i]
			break
		}
	}
	require.NotNil(t, httpLog, "HTTP Request log should exist")
	assert.Equal(t, zapcore.InfoLevel, httpLog.Level)
}

func TestGinMiddleware_WithRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	core, recorded := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(core)

	router := gin.New()

	// Add request ID first (simulating RequestID middleware)
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "test-req-123")
		c.Next()
	})
	router.Use(GinMiddleware(zapLogger))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	logs := recorded.All()
	require.NotEmpty(t, logs)

	// Find the HTTP Request log and verify request_id
	var httpLog *observer.LoggedEntry
	for i := range logs {
		if logs[i].Message == "HTTP Request" {
			httpLog = &logs[i]
			break
		}
	}
	require.NotNil(t, httpLog)

	hasRequestID := false
	for _, field := range httpLog.Context {
		if field.Key == "request_id" {
			hasRequestID = true
			assert.Equal(t, "test-req-123", field.String)
		}
	}
	assert.True(t, hasRequestID, "request_id should be in log fields")
}

func TestGinMiddleware_ErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	core, recorded := observer.New(zapcore.WarnLevel)
	zapLogger := zap.New(core)

	router := gin.New()
	router.Use(GinMiddleware(zapLogger))
	router.GET("/error", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/error", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 4xx responses should be logged as warnings
	logs := recorded.All()
	require.NotEmpty(t, logs)

	var httpLog *observer.LoggedEntry
	for i := range logs {
		if logs[i].Message == "HTTP Request" {
			httpLog = &logs[i]
			break
		}
	}
	require.NotNil(t, httpLog)
	assert.Equal(t, zapcore.WarnLevel, httpLog.Level)
}

func TestGinMiddleware_ServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	core, recorded := observer.New(zapcore.ErrorLevel)
	zapLogger := zap.New(core)

	router := gin.New()
	router.Use(GinMiddleware(zapLogger))
	router.GET("/error", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/error", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 5xx responses should be logged as errors
	logs := recorded.All()
	require.NotEmpty(t, logs)

	var httpLog *observer.LoggedEntry
	for i := range logs {
		if logs[i].Message == "HTTP Request" {
			httpLog = &logs[i]
			break
		}
	}
	require.NotNil(t, httpLog)
	assert.Equal(t, zapcore.ErrorLevel, httpLog.Level)
}

func TestGinMiddleware_WithQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	core, recorded := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(core)

	router := gin.New()
	router.Use(GinMiddleware(zapLogger))
	router.GET("/search", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/search?q=test&page=1", nil)
	router.ServeHTTP(w, req)

	logs := recorded.All()
	require.NotEmpty(t, logs)

	// Find query field
	var httpLog *observer.LoggedEntry
	for i := range logs {
		if logs[i].Message == "HTTP Request" {
			httpLog = &logs[i]
			break
		}
	}
	require.NotNil(t, httpLog)

	hasQuery := false
	for _, field := range httpLog.Context {
		if field.Key == "query" {
			hasQuery = true
			assert.Contains(t, field.String, "q=test")
		}
	}
	assert.True(t, hasQuery, "query should be in log fields")
}

func TestRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	core, recorded := observer.New(zapcore.ErrorLevel)
	zapLogger := zap.New(core)

	router := gin.New()
	router.Use(Recovery(zapLogger))
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/panic", nil)

	// Should not panic
	assert.NotPanics(t, func() {
		router.ServeHTTP(w, req)
	})

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// Should log the panic
	logs := recorded.All()
	require.NotEmpty(t, logs)
	assert.Contains(t, logs[0].Message, "Panic recovered")
}

func TestGetGinLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)

	core, _ := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(core)

	var retrievedLogger *zap.Logger

	router := gin.New()
	router.Use(GinMiddleware(zapLogger))
	router.GET("/test", func(c *gin.Context) {
		retrievedLogger = GetGinLogger(c)
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.NotNil(t, retrievedLogger)
}

func TestGetGinLogger_NotSet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var retrievedLogger *zap.Logger

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		retrievedLogger = GetGinLogger(c)
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	// Should return no-op logger, not nil
	assert.NotNil(t, retrievedLogger)

	// Should not panic when used
	assert.NotPanics(t, func() {
		retrievedLogger.Info("test")
	})
}

func TestGinMiddleware_LogsCorrectFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	core, recorded := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(core)

	router := gin.New()
	router.Use(GinMiddleware(zapLogger))
	router.POST("/api/users", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"id": 1})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/users", nil)
	req.Header.Set("User-Agent", "Test-Agent/1.0")
	router.ServeHTTP(w, req)

	logs := recorded.All()
	require.NotEmpty(t, logs)

	var httpLog *observer.LoggedEntry
	for i := range logs {
		if logs[i].Message == "HTTP Request" {
			httpLog = &logs[i]
			break
		}
	}
	require.NotNil(t, httpLog)

	// Check expected fields
	fieldMap := make(map[string]any)
	for _, field := range httpLog.Context {
		fieldMap[field.Key] = field
	}

	assert.Contains(t, fieldMap, "status")
	assert.Contains(t, fieldMap, "latency")
	assert.Contains(t, fieldMap, "client_ip")
	assert.Contains(t, fieldMap, "user_agent")
	assert.Contains(t, fieldMap, "method")
	assert.Contains(t, fieldMap, "path")
}
