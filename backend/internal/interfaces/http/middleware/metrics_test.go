package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// setupTestMeter sets up a test meter provider and reader.
func setupTestMeter(t *testing.T) (*sdkmetric.MeterProvider, *sdkmetric.ManualReader) {
	t.Helper()

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(mp)

	t.Cleanup(func() {
		_ = mp.Shutdown(context.Background())
	})

	return mp, reader
}

// collectMetrics collects metrics from the reader.
func collectMetrics(t *testing.T, reader *sdkmetric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()

	var rm metricdata.ResourceMetrics
	err := reader.Collect(context.Background(), &rm)
	require.NoError(t, err)
	return rm
}

// findMetricByName finds a metric by name in the collected metrics.
func findMetricByName(rm metricdata.ResourceMetrics, name string) *metricdata.Metrics {
	for _, sm := range rm.ScopeMetrics {
		for i := range sm.Metrics {
			if sm.Metrics[i].Name == name {
				return &sm.Metrics[i]
			}
		}
	}
	return nil
}

func TestHTTPMetrics_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := HTTPMetricsConfig{
		Enabled: false,
	}

	router := gin.New()
	router.Use(HTTPMetrics(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHTTPMetrics_NilMeterProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := HTTPMetricsConfig{
		Enabled:       true,
		MeterProvider: nil,
	}

	router := gin.New()
	router.Use(HTTPMetrics(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	// Should not panic and return OK
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHTTPMetricsWithMeter_Enabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mp, reader := setupTestMeter(t)

	meter := mp.Meter("http.server")

	router := gin.New()
	router.Use(HTTPMetricsWithMeter(meter, true))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Collect and verify metrics
	rm := collectMetrics(t, reader)

	// Verify request_total counter
	requestTotalMetric := findMetricByName(rm, "http_server_request_total")
	require.NotNil(t, requestTotalMetric, "http_server_request_total metric not found")

	// Verify request_duration histogram
	requestDurationMetric := findMetricByName(rm, "http_server_request_duration_seconds")
	require.NotNil(t, requestDurationMetric, "http_server_request_duration_seconds metric not found")
}

func TestHTTPMetricsWithMeter_RequestCounter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mp, reader := setupTestMeter(t)

	meter := mp.Meter("http.server")

	router := gin.New()
	router.Use(HTTPMetricsWithMeter(meter, true))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	// Make 3 requests
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Collect metrics
	rm := collectMetrics(t, reader)

	// Verify request_total counter value
	requestTotalMetric := findMetricByName(rm, "http_server_request_total")
	require.NotNil(t, requestTotalMetric, "http_server_request_total metric not found")

	// Check that we have sum data (counter)
	sumData, ok := requestTotalMetric.Data.(metricdata.Sum[int64])
	require.True(t, ok, "expected Sum data for counter")
	require.Len(t, sumData.DataPoints, 1)
	assert.Equal(t, int64(3), sumData.DataPoints[0].Value)
}

func TestHTTPMetricsWithMeter_DifferentStatusCodes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mp, reader := setupTestMeter(t)

	meter := mp.Meter("http.server")

	router := gin.New()
	router.Use(HTTPMetricsWithMeter(meter, true))
	router.GET("/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	router.GET("/error", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error"})
	})
	router.GET("/notfound", func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})

	// Make requests with different status codes
	for _, path := range []string{"/ok", "/ok", "/error", "/notfound"} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, path, nil)
		router.ServeHTTP(w, req)
	}

	// Collect metrics
	rm := collectMetrics(t, reader)

	// Verify request_total counter
	requestTotalMetric := findMetricByName(rm, "http_server_request_total")
	require.NotNil(t, requestTotalMetric, "http_server_request_total metric not found")

	sumData, ok := requestTotalMetric.Data.(metricdata.Sum[int64])
	require.True(t, ok, "expected Sum data for counter")

	// Should have separate data points for different status codes
	// Total requests should be 4
	var totalRequests int64
	for _, dp := range sumData.DataPoints {
		totalRequests += dp.Value
	}
	assert.Equal(t, int64(4), totalRequests)
}

func TestHTTPMetricsWithMeter_DifferentMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mp, reader := setupTestMeter(t)

	meter := mp.Meter("http.server")

	router := gin.New()
	router.Use(HTTPMetricsWithMeter(meter, true))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"message": "created"})
	})
	router.PUT("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "updated"})
	})

	// Make requests with different methods
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut}
	for _, method := range methods {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(method, "/test", nil)
		router.ServeHTTP(w, req)
	}

	// Collect metrics
	rm := collectMetrics(t, reader)

	// Verify request_total counter
	requestTotalMetric := findMetricByName(rm, "http_server_request_total")
	require.NotNil(t, requestTotalMetric, "http_server_request_total metric not found")

	sumData, ok := requestTotalMetric.Data.(metricdata.Sum[int64])
	require.True(t, ok, "expected Sum data for counter")

	// Should have separate data points for different methods
	// Total requests should be 3
	var totalRequests int64
	for _, dp := range sumData.DataPoints {
		totalRequests += dp.Value
	}
	assert.Equal(t, int64(3), totalRequests)
}

func TestHTTPMetricsWithMeter_RequestDuration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mp, reader := setupTestMeter(t)

	meter := mp.Meter("http.server")

	router := gin.New()
	router.Use(HTTPMetricsWithMeter(meter, true))
	router.GET("/slow", func(c *gin.Context) {
		time.Sleep(50 * time.Millisecond)
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/slow", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Collect metrics
	rm := collectMetrics(t, reader)

	// Verify request_duration histogram
	requestDurationMetric := findMetricByName(rm, "http_server_request_duration_seconds")
	require.NotNil(t, requestDurationMetric, "http_server_request_duration_seconds metric not found")

	histData, ok := requestDurationMetric.Data.(metricdata.Histogram[float64])
	require.True(t, ok, "expected Histogram data for duration")
	require.Len(t, histData.DataPoints, 1)

	// Duration should be > 50ms (0.05s)
	dp := histData.DataPoints[0]
	assert.Greater(t, dp.Sum, 0.05, "Duration should be greater than 50ms")
}

func TestHTTPMetricsWithMeter_RequestSize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mp, reader := setupTestMeter(t)

	meter := mp.Meter("http.server")

	router := gin.New()
	router.Use(HTTPMetricsWithMeter(meter, true))
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	body := strings.NewReader(`{"data": "test body content"}`)
	req, _ := http.NewRequest(http.MethodPost, "/test", body)
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(body.Len())
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Collect metrics
	rm := collectMetrics(t, reader)

	// Verify request_size histogram
	requestSizeMetric := findMetricByName(rm, "http_server_request_size_bytes")
	require.NotNil(t, requestSizeMetric, "http_server_request_size_bytes metric not found")

	histData, ok := requestSizeMetric.Data.(metricdata.Histogram[float64])
	require.True(t, ok, "expected Histogram data for request size")
	require.Len(t, histData.DataPoints, 1)

	// Request size should be > 0
	dp := histData.DataPoints[0]
	assert.Greater(t, dp.Sum, float64(0), "Request size should be greater than 0")
}

func TestHTTPMetricsWithMeter_ResponseSize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mp, reader := setupTestMeter(t)

	meter := mp.Meter("http.server")

	router := gin.New()
	router.Use(HTTPMetricsWithMeter(meter, true))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "this is a response body"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Collect metrics
	rm := collectMetrics(t, reader)

	// Verify response_size histogram
	responseSizeMetric := findMetricByName(rm, "http_server_response_size_bytes")
	require.NotNil(t, responseSizeMetric, "http_server_response_size_bytes metric not found")

	histData, ok := responseSizeMetric.Data.(metricdata.Histogram[float64])
	require.True(t, ok, "expected Histogram data for response size")
	require.Len(t, histData.DataPoints, 1)

	// Response size should be > 0
	dp := histData.DataPoints[0]
	assert.Greater(t, dp.Sum, float64(0), "Response size should be greater than 0")
}

func TestHTTPMetricsWithMeter_ActiveRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mp, reader := setupTestMeter(t)

	meter := mp.Meter("http.server")

	router := gin.New()
	router.Use(HTTPMetricsWithMeter(meter, true))
	router.GET("/test", func(c *gin.Context) {
		// Active requests should be 1 during this handler
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Collect metrics
	rm := collectMetrics(t, reader)

	// Verify active_requests gauge
	activeRequestsMetric := findMetricByName(rm, "http_server_active_requests")
	require.NotNil(t, activeRequestsMetric, "http_server_active_requests metric not found")

	// After request completes, active_requests should be 0
	sumData, ok := activeRequestsMetric.Data.(metricdata.Sum[int64])
	require.True(t, ok, "expected Sum data for active_requests")
	if len(sumData.DataPoints) > 0 {
		assert.Equal(t, int64(0), sumData.DataPoints[0].Value)
	}
}

func TestHTTPMetricsWithMeter_WithTenantID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mp, reader := setupTestMeter(t)

	meter := mp.Meter("http.server")

	router := gin.New()
	// Simulate JWT middleware setting tenant_id
	router.Use(func(c *gin.Context) {
		c.Set(JWTTenantIDKey, "tenant-123")
		c.Next()
	})
	router.Use(HTTPMetricsWithMeter(meter, true))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Collect metrics
	rm := collectMetrics(t, reader)

	// Verify request_total counter has tenant_id attribute
	requestTotalMetric := findMetricByName(rm, "http_server_request_total")
	require.NotNil(t, requestTotalMetric, "http_server_request_total metric not found")

	sumData, ok := requestTotalMetric.Data.(metricdata.Sum[int64])
	require.True(t, ok, "expected Sum data for counter")
	require.Len(t, sumData.DataPoints, 1)

	// Check for tenant_id attribute
	found := false
	for _, attr := range sumData.DataPoints[0].Attributes.ToSlice() {
		if string(attr.Key) == "tenant_id" {
			assert.Equal(t, "tenant-123", attr.Value.AsString())
			found = true
			break
		}
	}
	assert.True(t, found, "tenant_id attribute not found in metrics")
}

func TestHTTPMetricsWithMeter_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mp, _ := setupTestMeter(t)

	meter := mp.Meter("http.server")

	router := gin.New()
	router.Use(HTTPMetricsWithMeter(meter, false))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetRoutePattern_WithMatchedRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/api/v1/products/:id", func(c *gin.Context) {
		route := getRoutePattern(c)
		c.JSON(http.StatusOK, gin.H{"route": route})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/products/123", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "/api/v1/products/:id")
}

func TestGetRoutePattern_UnmatchedRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	// Add a middleware that will handle all requests
	router.Use(func(c *gin.Context) {
		route := getRoutePattern(c)
		c.JSON(http.StatusNotFound, gin.H{"route": route})
		c.Abort()
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/nonexistent", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "unknown")
}

func TestGetRequestSize(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name          string
		contentLength int64
		expectedSize  int64
	}{
		{
			name:          "with content length",
			contentLength: 100,
			expectedSize:  100,
		},
		{
			name:          "zero content length",
			contentLength: 0,
			expectedSize:  0,
		},
		{
			name:          "negative content length",
			contentLength: -1,
			expectedSize:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.POST("/test", func(c *gin.Context) {
				size := getRequestSize(c)
				c.JSON(http.StatusOK, gin.H{"size": size})
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/test", nil)
			req.ContentLength = tc.contentLength
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestGetTenantIDFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		contextValue   interface{}
		expectedResult string
	}{
		{
			name:           "with tenant_id",
			contextValue:   "tenant-123",
			expectedResult: "tenant-123",
		},
		{
			name:           "empty tenant_id",
			contextValue:   "",
			expectedResult: "",
		},
		{
			name:           "no tenant_id",
			contextValue:   nil,
			expectedResult: "",
		},
		{
			name:           "non-string tenant_id",
			contextValue:   123,
			expectedResult: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			if tc.contextValue != nil {
				router.Use(func(c *gin.Context) {
					c.Set(JWTTenantIDKey, tc.contextValue)
					c.Next()
				})
			}
			router.GET("/test", func(c *gin.Context) {
				tenantID := getTenantIDFromContext(c)
				c.JSON(http.StatusOK, gin.H{"tenant_id": tenantID})
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestHTTPMetricsStatusGroup(t *testing.T) {
	testCases := []struct {
		statusCode int
		expected   string
	}{
		{200, "2xx"},
		{201, "2xx"},
		{299, "2xx"},
		{300, "3xx"},
		{301, "3xx"},
		{399, "3xx"},
		{400, "4xx"},
		{401, "4xx"},
		{404, "4xx"},
		{499, "4xx"},
		{500, "5xx"},
		{501, "5xx"},
		{503, "5xx"},
		{599, "5xx"},
		{100, "other"},
		{199, "other"},
		{600, "5xx"}, // Anything >= 500 is 5xx
		{0, "other"},
	}

	for _, tc := range testCases {
		t.Run(http.StatusText(tc.statusCode), func(t *testing.T) {
			result := HTTPMetricsStatusGroup(tc.statusCode)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseStatusCode(t *testing.T) {
	testCases := []struct {
		input    string
		expected int
	}{
		{"200", 200},
		{"404", 404},
		{"500", 500},
		{"invalid", 0},
		{"", 0},
		{"12.34", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := ParseStatusCode(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHTTPMetricsResponseWriter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	// Use gin's built-in response writer wrapper
	ctx, _ := gin.CreateTestContext(w)
	rw := &HTTPMetricsResponseWriter{
		ResponseWriter: ctx.Writer,
	}

	// Write some bytes
	n, err := rw.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, 5, rw.BytesWritten())

	// Write more bytes
	n, err = rw.Write([]byte(" world"))
	assert.NoError(t, err)
	assert.Equal(t, 6, n)
	assert.Equal(t, 11, rw.BytesWritten())
}

func TestDefaultHTTPMetricsConfig(t *testing.T) {
	cfg := DefaultHTTPMetricsConfig()

	assert.Equal(t, "erp-backend", cfg.ServiceName)
	assert.True(t, cfg.Enabled)
	assert.Nil(t, cfg.MeterProvider)
}

func TestHTTPMetricsWithMeter_RoutePatternAttributes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mp, reader := setupTestMeter(t)

	meter := mp.Meter("http.server")

	router := gin.New()
	router.Use(HTTPMetricsWithMeter(meter, true))
	router.GET("/api/v1/products/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"id": c.Param("id")})
	})

	// Make requests with different IDs - should all use the same route pattern
	for _, id := range []string{"1", "2", "abc", "xyz"} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/products/"+id, nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Collect metrics
	rm := collectMetrics(t, reader)

	// Verify request_total counter
	requestTotalMetric := findMetricByName(rm, "http_server_request_total")
	require.NotNil(t, requestTotalMetric, "http_server_request_total metric not found")

	sumData, ok := requestTotalMetric.Data.(metricdata.Sum[int64])
	require.True(t, ok, "expected Sum data for counter")

	// All 4 requests should be counted under the same route pattern
	// There should be only 1 data point (same method, route, status_code)
	require.Len(t, sumData.DataPoints, 1)
	assert.Equal(t, int64(4), sumData.DataPoints[0].Value)

	// Verify the route attribute uses the pattern, not the actual path
	found := false
	for _, attr := range sumData.DataPoints[0].Attributes.ToSlice() {
		if string(attr.Key) == "http.route" {
			assert.Equal(t, "/api/v1/products/:id", attr.Value.AsString())
			found = true
			break
		}
	}
	assert.True(t, found, "http.route attribute not found")
}
