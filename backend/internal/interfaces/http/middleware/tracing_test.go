package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace/noop"
)

// setupTestTracer sets up a test tracer provider and returns the span recorder.
func setupTestTracer(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()

	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	t.Cleanup(func() {
		_ = tp.Shutdown(t.Context())
	})

	return sr
}

func TestTracingWithConfig_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	}

	router := gin.New()
	router.Use(TracingWithConfig(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTracingWithConfig_Enabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sr := setupTestTracer(t)

	cfg := TracingConfig{
		Enabled:     true,
		ServiceName: "test-service",
	}

	router := gin.New()
	router.Use(TracingWithConfig(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check that a span was created
	spans := sr.Ended()
	require.GreaterOrEqual(t, len(spans), 1)

	// Find the HTTP span
	var httpSpan sdktrace.ReadOnlySpan
	for _, span := range spans {
		if span.Name() == "GET /test" {
			httpSpan = span
			break
		}
	}
	require.NotNil(t, httpSpan, "HTTP span not found")
}

func TestTracingWithConfig_WithRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sr := setupTestTracer(t)

	cfg := TracingConfig{
		Enabled:     true,
		ServiceName: "test-service",
	}

	router := gin.New()
	// Add RequestID middleware first
	router.Use(RequestID())
	router.Use(TracingWithConfig(cfg))
	router.Use(TracingAttributeInjector())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "test-request-id-123")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check that spans were created with request_id attribute
	spans := sr.Ended()
	require.GreaterOrEqual(t, len(spans), 1)

	// Find the HTTP span and check for request_id attribute
	found := false
	for _, span := range spans {
		if span.Name() == "GET /test" {
			attrs := span.Attributes()
			for _, attr := range attrs {
				if attr.Key == "request_id" {
					assert.Equal(t, "test-request-id-123", attr.Value.AsString())
					found = true
					break
				}
			}
			break
		}
	}
	assert.True(t, found, "request_id attribute not found in span")
}

func TestTracingWithConfig_WithJWTClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sr := setupTestTracer(t)

	cfg := TracingConfig{
		Enabled:     true,
		ServiceName: "test-service",
	}

	router := gin.New()
	router.Use(TracingWithConfig(cfg))
	// Simulate JWT middleware setting claims
	router.Use(func(c *gin.Context) {
		c.Set(JWTUserIDKey, "user-123")
		c.Set(JWTTenantIDKey, "tenant-456")
		c.Next()
	})
	router.Use(TracingAttributeInjector())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check that spans were created with user_id and tenant_id attributes
	spans := sr.Ended()
	require.GreaterOrEqual(t, len(spans), 1)

	// Find the HTTP span and check for attributes
	userIDFound := false
	tenantIDFound := false
	for _, span := range spans {
		if span.Name() == "GET /test" {
			attrs := span.Attributes()
			for _, attr := range attrs {
				if attr.Key == "user_id" {
					assert.Equal(t, "user-123", attr.Value.AsString())
					userIDFound = true
				}
				if attr.Key == "tenant_id" {
					assert.Equal(t, "tenant-456", attr.Value.AsString())
					tenantIDFound = true
				}
			}
			break
		}
	}
	assert.True(t, userIDFound, "user_id attribute not found in span")
	assert.True(t, tenantIDFound, "tenant_id attribute not found in span")
}

func TestTracingWithConfig_WithTenantHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sr := setupTestTracer(t)

	cfg := TracingConfig{
		Enabled:     true,
		ServiceName: "test-service",
	}

	router := gin.New()
	router.Use(TracingWithConfig(cfg))
	router.Use(TracingAttributeInjector())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	// Use valid UUID format for tenant ID header
	req.Header.Set("X-Tenant-ID", "12345678-1234-1234-1234-123456789abc")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check that spans were created with tenant_id from header
	spans := sr.Ended()
	require.GreaterOrEqual(t, len(spans), 1)

	// Find the HTTP span and check for tenant_id attribute
	found := false
	for _, span := range spans {
		if span.Name() == "GET /test" {
			attrs := span.Attributes()
			for _, attr := range attrs {
				if attr.Key == "tenant_id" {
					assert.Equal(t, "12345678-1234-1234-1234-123456789abc", attr.Value.AsString())
					found = true
					break
				}
			}
			break
		}
	}
	assert.True(t, found, "tenant_id attribute not found in span")
}

func TestSpanErrorMarker_4xxError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sr := setupTestTracer(t)

	cfg := TracingConfig{
		Enabled:     true,
		ServiceName: "test-service",
	}

	router := gin.New()
	router.Use(TracingWithConfig(cfg))
	router.Use(SpanErrorMarker())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	// Check that the span was marked with error status
	spans := sr.Ended()
	require.GreaterOrEqual(t, len(spans), 1)

	// Find the HTTP span and check for error status
	for _, span := range spans {
		if span.Name() == "GET /test" {
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, "Not Found", span.Status().Description)
			break
		}
	}
}

func TestSpanErrorMarker_5xxError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sr := setupTestTracer(t)

	cfg := TracingConfig{
		Enabled:     true,
		ServiceName: "test-service",
	}

	router := gin.New()
	router.Use(TracingWithConfig(cfg))
	router.Use(SpanErrorMarker())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// Check that the span was marked with error status
	spans := sr.Ended()
	require.GreaterOrEqual(t, len(spans), 1)

	// Find the HTTP span and check for error status
	// Note: otelgin may already set the error status, so we just verify it's Error
	for _, span := range spans {
		if span.Name() == "GET /test" {
			assert.Equal(t, codes.Error, span.Status().Code)
			// Description may vary depending on whether otelgin or our middleware set it
			// The important thing is the error code is set
			break
		}
	}
}

func TestSpanErrorMarker_401Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sr := setupTestTracer(t)

	cfg := TracingConfig{
		Enabled:     true,
		ServiceName: "test-service",
	}

	router := gin.New()
	router.Use(TracingWithConfig(cfg))
	router.Use(SpanErrorMarker())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Check that the span was marked with error status
	spans := sr.Ended()
	require.GreaterOrEqual(t, len(spans), 1)

	// Find the HTTP span and check for error status
	for _, span := range spans {
		if span.Name() == "GET /test" {
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, "Unauthorized", span.Status().Description)
			break
		}
	}
}

func TestSpanErrorMarker_403Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sr := setupTestTracer(t)

	cfg := TracingConfig{
		Enabled:     true,
		ServiceName: "test-service",
	}

	router := gin.New()
	router.Use(TracingWithConfig(cfg))
	router.Use(SpanErrorMarker())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	// Check that the span was marked with error status
	spans := sr.Ended()
	require.GreaterOrEqual(t, len(spans), 1)

	// Find the HTTP span and check for error status
	for _, span := range spans {
		if span.Name() == "GET /test" {
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, "Forbidden", span.Status().Description)
			break
		}
	}
}

func TestSpanErrorMarker_SuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sr := setupTestTracer(t)

	cfg := TracingConfig{
		Enabled:     true,
		ServiceName: "test-service",
	}

	router := gin.New()
	router.Use(TracingWithConfig(cfg))
	router.Use(SpanErrorMarker())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check that the span was NOT marked with error status
	spans := sr.Ended()
	require.GreaterOrEqual(t, len(spans), 1)

	// Find the HTTP span and check status is Unset (not Error)
	for _, span := range spans {
		if span.Name() == "GET /test" {
			assert.NotEqual(t, codes.Error, span.Status().Code)
			break
		}
	}
}

func TestDefaultTracingConfig(t *testing.T) {
	cfg := DefaultTracingConfig()

	assert.Equal(t, "erp-backend", cfg.ServiceName)
	assert.True(t, cfg.Enabled)
}

func TestTracing_DefaultConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sr := setupTestTracer(t)

	router := gin.New()
	router.Use(Tracing())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check that a span was created
	spans := sr.Ended()
	require.GreaterOrEqual(t, len(spans), 1)
}

func TestGetRequestID_FromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "context-request-id")
		c.Next()
	})
	router.GET("/test", func(c *gin.Context) {
		requestID := getRequestID(c)
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "context-request-id")
}

func TestGetRequestID_FromHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		requestID := getRequestID(c)
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "header-request-id")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "header-request-id")
}

func TestGetTenantID_FromJWT(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(JWTTenantIDKey, "jwt-tenant-id")
		c.Next()
	})
	router.GET("/test", func(c *gin.Context) {
		tenantID := getTenantID(c)
		c.JSON(http.StatusOK, gin.H{"tenant_id": tenantID})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "jwt-tenant-id")
}

func TestGetTenantID_FromHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		tenantID := getTenantID(c)
		c.JSON(http.StatusOK, gin.H{"tenant_id": tenantID})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	// Use valid UUID format
	req.Header.Set("X-Tenant-ID", "12345678-1234-1234-1234-123456789abc")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "12345678-1234-1234-1234-123456789abc")
}

func TestGetTenantID_FromHeader_InvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		tenantID := getTenantID(c)
		c.JSON(http.StatusOK, gin.H{"tenant_id": tenantID})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	// Invalid UUID format - should be rejected
	req.Header.Set("X-Tenant-ID", "invalid-tenant-id")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Should return empty tenant_id since the header value is invalid
	assert.Contains(t, w.Body.String(), `"tenant_id":""`)
}

func TestGetUserID_FromJWT(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(JWTUserIDKey, "jwt-user-id")
		c.Next()
	})
	router.GET("/test", func(c *gin.Context) {
		userID := getUserID(c)
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "jwt-user-id")
}

func TestGetUserID_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		userID := getUserID(c)
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"user_id":""`)
}

func TestTracingAttributeInjector_WithNoSpan(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Don't set up a tracer provider, so there's no recording span
	router := gin.New()
	router.Use(TracingAttributeInjector())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	// Should not panic and should return 200
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSpanErrorMarker_WithNoSpan(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Use a no-op tracer provider
	noopTp := noop.NewTracerProvider()
	otel.SetTracerProvider(noopTp)

	router := gin.New()
	router.Use(SpanErrorMarker())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	// Should not panic and should return 500
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSpanErrorMarker_BadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sr := setupTestTracer(t)

	cfg := TracingConfig{
		Enabled:     true,
		ServiceName: "test-service",
	}

	router := gin.New()
	router.Use(TracingWithConfig(cfg))
	router.Use(SpanErrorMarker())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Check that the span was marked with error status
	spans := sr.Ended()
	require.GreaterOrEqual(t, len(spans), 1)

	// Find the HTTP span and check for error status
	for _, span := range spans {
		if span.Name() == "GET /test" {
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, "Client Error", span.Status().Description)
			break
		}
	}
}

func TestGetRequestID_LongHeader_Truncated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		requestID := getRequestID(c)
		c.JSON(http.StatusOK, gin.H{"request_id": requestID, "length": len(requestID)})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	// Create a very long request ID that exceeds MaxRequestIDLength (128)
	longRequestID := "a"
	for i := 0; i < 200; i++ {
		longRequestID += "b"
	}
	req.Header.Set("X-Request-ID", longRequestID)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// The request ID should be truncated to MaxRequestIDLength (128)
	assert.Contains(t, w.Body.String(), `"length":128`)
}

func TestIsValidTenantID_ValidUUID(t *testing.T) {
	testCases := []struct {
		name     string
		tenantID string
		expected bool
	}{
		{
			name:     "valid lowercase UUID",
			tenantID: "12345678-1234-1234-1234-123456789abc",
			expected: true,
		},
		{
			name:     "valid uppercase UUID",
			tenantID: "12345678-1234-1234-1234-123456789ABC",
			expected: true,
		},
		{
			name:     "valid mixed case UUID",
			tenantID: "12345678-1234-1234-1234-123456789AbC",
			expected: true,
		},
		{
			name:     "invalid - too short",
			tenantID: "12345678-1234-1234",
			expected: false,
		},
		{
			name:     "invalid - no dashes",
			tenantID: "12345678123412341234123456789abc",
			expected: false,
		},
		{
			name:     "invalid - contains special characters",
			tenantID: "12345678-1234-1234-1234-123456789<>!",
			expected: false,
		},
		{
			name:     "invalid - script injection attempt",
			tenantID: "<script>alert(1)</script>",
			expected: false,
		},
		{
			name:     "empty string",
			tenantID: "",
			expected: false,
		},
		{
			name:     "invalid - contains spaces",
			tenantID: "12345678-1234 -1234-1234-123456789abc",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidTenantID(tc.tenantID)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsValidTenantID_TooLong(t *testing.T) {
	// Create a string that looks like a UUID but is too long
	longTenantID := "12345678-1234-1234-1234-123456789abc"
	for i := 0; i < 100; i++ {
		longTenantID += "extra"
	}

	result := isValidTenantID(longTenantID)
	assert.False(t, result, "Should reject tenant IDs exceeding MaxTenantIDLength")
}
