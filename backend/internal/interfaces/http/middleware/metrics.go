// Package middleware provides HTTP middleware for the ERP system.
package middleware

import (
	"context"
	"strconv"
	"time"

	"github.com/erp/backend/internal/infrastructure/telemetry"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// HTTPMetricsConfig holds configuration for HTTP metrics middleware.
type HTTPMetricsConfig struct {
	// MeterProvider is the OpenTelemetry meter provider.
	MeterProvider *telemetry.MeterProvider
	// ServiceName is the name of the service for metric identification.
	ServiceName string
	// Enabled controls whether metrics collection is active.
	Enabled bool
}

// DefaultHTTPMetricsConfig returns default HTTP metrics configuration.
func DefaultHTTPMetricsConfig() HTTPMetricsConfig {
	return HTTPMetricsConfig{
		ServiceName: "erp-backend",
		Enabled:     true,
	}
}

// httpMetrics holds all HTTP-related metrics instruments.
type httpMetrics struct {
	requestTotal        *telemetry.Counter
	requestDuration     *telemetry.Histogram
	requestSize         *telemetry.Histogram
	responseSize        *telemetry.Histogram
	activeRequests      metric.Int64UpDownCounter
	activeRequestsGauge *telemetry.Gauge
}

// newHTTPMetrics creates all HTTP metrics instruments from a meter.
func newHTTPMetrics(meter metric.Meter) (*httpMetrics, error) {
	// 1. http_server_request_total (Counter)
	requestTotal, err := telemetry.NewCounter(
		meter,
		"http_server_request_total",
		"Total number of HTTP requests",
		"{request}",
	)
	if err != nil {
		return nil, err
	}

	// 2. http_server_request_duration_seconds (Histogram)
	requestDuration, err := telemetry.NewHistogram(meter, telemetry.HistogramOpts{
		Name:        "http_server_request_duration_seconds",
		Description: "HTTP request latency distribution in seconds",
		Unit:        "s",
		Boundaries:  telemetry.HTTPDurationBuckets,
	})
	if err != nil {
		return nil, err
	}

	// 3. http_server_request_size_bytes (Histogram)
	// Using appropriate bucket boundaries for request sizes
	requestSizeBuckets := []float64{100, 500, 1000, 5000, 10000, 50000, 100000, 500000, 1000000}
	requestSize, err := telemetry.NewHistogram(meter, telemetry.HistogramOpts{
		Name:        "http_server_request_size_bytes",
		Description: "HTTP request body size distribution in bytes",
		Unit:        "By",
		Boundaries:  requestSizeBuckets,
	})
	if err != nil {
		return nil, err
	}

	// 4. http_server_response_size_bytes (Histogram)
	responseSizeBuckets := []float64{100, 500, 1000, 5000, 10000, 50000, 100000, 500000, 1000000, 5000000}
	responseSize, err := telemetry.NewHistogram(meter, telemetry.HistogramOpts{
		Name:        "http_server_response_size_bytes",
		Description: "HTTP response body size distribution in bytes",
		Unit:        "By",
		Boundaries:  responseSizeBuckets,
	})
	if err != nil {
		return nil, err
	}

	// 5. http_server_active_requests (UpDownCounter for concurrent requests)
	activeRequests, err := meter.Int64UpDownCounter(
		"http_server_active_requests",
		metric.WithDescription("Number of currently active HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	return &httpMetrics{
		requestTotal:    requestTotal,
		requestDuration: requestDuration,
		requestSize:     requestSize,
		responseSize:    responseSize,
		activeRequests:  activeRequests,
	}, nil
}

// HTTPMetrics returns a Gin middleware that collects HTTP metrics.
// This middleware tracks:
// - http_server_request_total: Total request count with method, route, status_code, tenant_id labels
// - http_server_request_duration_seconds: Request latency histogram with method, route labels
// - http_server_request_size_bytes: Request body size histogram with method, route labels
// - http_server_response_size_bytes: Response body size histogram with method, route labels
// - http_server_active_requests: Gauge of currently processing requests
func HTTPMetrics(cfg HTTPMetricsConfig) gin.HandlerFunc {
	if !cfg.Enabled || cfg.MeterProvider == nil || !cfg.MeterProvider.IsEnabled() {
		// Return no-op middleware when metrics are disabled
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// Create meter and instruments
	meter := cfg.MeterProvider.Meter("http.server")
	metrics, err := newHTTPMetrics(meter)
	if err != nil {
		// If metrics setup fails, return no-op middleware
		// In production, this should be logged
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		ctx := c.Request.Context()
		start := time.Now()

		// Record request size (Content-Length header if available)
		requestSize := getRequestSize(c)

		// Increment active requests
		metrics.activeRequests.Add(ctx, 1)

		// Process request
		c.Next()

		// Decrement active requests
		metrics.activeRequests.Add(ctx, -1)

		// Calculate duration
		duration := time.Since(start)

		// Get route pattern (not actual path to avoid high cardinality)
		route := getRoutePattern(c)

		// Get attributes
		method := c.Request.Method
		statusCode := c.Writer.Status()
		tenantID := getTenantIDFromContext(c)

		// Record http_server_request_total
		requestAttrs := []attribute.KeyValue{
			telemetry.AttrHTTPMethod.String(method),
			telemetry.AttrHTTPRoute.String(route),
			telemetry.AttrHTTPStatusCode.Int(statusCode),
		}
		if tenantID != "" {
			requestAttrs = append(requestAttrs, telemetry.AttrTenantID.String(tenantID))
		}
		metrics.requestTotal.Inc(ctx, requestAttrs...)

		// Record http_server_request_duration_seconds (only method and route to reduce cardinality)
		durationAttrs := []attribute.KeyValue{
			telemetry.AttrHTTPMethod.String(method),
			telemetry.AttrHTTPRoute.String(route),
		}
		metrics.requestDuration.RecordDuration(ctx, duration, durationAttrs...)

		// Record http_server_request_size_bytes
		if requestSize > 0 {
			metrics.requestSize.Record(ctx, float64(requestSize), durationAttrs...)
		}

		// Record http_server_response_size_bytes
		responseSize := c.Writer.Size()
		if responseSize > 0 {
			metrics.responseSize.Record(ctx, float64(responseSize), durationAttrs...)
		}
	}
}

// getRoutePattern returns the route pattern (e.g., "/api/v1/products/:id")
// instead of the actual path to avoid high cardinality issues.
func getRoutePattern(c *gin.Context) string {
	// Gin's FullPath returns the matched route pattern
	route := c.FullPath()
	if route == "" {
		// For unmatched routes, use a generic pattern
		return "unknown"
	}
	return route
}

// getRequestSize returns the size of the request body.
func getRequestSize(c *gin.Context) int64 {
	// Try Content-Length header first
	if cl := c.Request.ContentLength; cl > 0 {
		return cl
	}
	return 0
}

// getTenantIDFromContext retrieves tenant_id from the gin context.
// This relies on the JWT middleware having set the tenant_id.
func getTenantIDFromContext(c *gin.Context) string {
	if tenantID, exists := c.Get(JWTTenantIDKey); exists {
		if id, ok := tenantID.(string); ok && id != "" {
			return id
		}
	}
	return ""
}

// HTTPMetricsWithMeter returns HTTP metrics middleware using an existing meter.
// This is useful for testing or when you want to provide a custom meter.
func HTTPMetricsWithMeter(meter metric.Meter, enabled bool) gin.HandlerFunc {
	if !enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	metrics, err := newHTTPMetrics(meter)
	if err != nil {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return httpMetricsMiddleware(metrics)
}

// httpMetricsMiddleware is the core middleware logic, extracted for reuse.
func httpMetricsMiddleware(metrics *httpMetrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		start := time.Now()

		// Record request size
		requestSize := getRequestSize(c)

		// Increment active requests
		metrics.activeRequests.Add(ctx, 1)

		// Process request
		c.Next()

		// Decrement active requests
		metrics.activeRequests.Add(ctx, -1)

		// Calculate duration
		duration := time.Since(start)

		// Get route pattern
		route := getRoutePattern(c)

		// Get attributes
		method := c.Request.Method
		statusCode := c.Writer.Status()
		tenantID := getTenantIDFromContext(c)

		// Record metrics
		recordHTTPMetrics(ctx, metrics, method, route, statusCode, tenantID, duration, requestSize, c.Writer.Size())
	}
}

// recordHTTPMetrics records all HTTP metrics for a request.
func recordHTTPMetrics(
	ctx context.Context,
	metrics *httpMetrics,
	method, route string,
	statusCode int,
	tenantID string,
	duration time.Duration,
	requestSize int64,
	responseSize int,
) {
	// Request counter attributes (includes status_code and tenant_id)
	requestAttrs := []attribute.KeyValue{
		telemetry.AttrHTTPMethod.String(method),
		telemetry.AttrHTTPRoute.String(route),
		telemetry.AttrHTTPStatusCode.Int(statusCode),
	}
	if tenantID != "" {
		requestAttrs = append(requestAttrs, telemetry.AttrTenantID.String(tenantID))
	}
	metrics.requestTotal.Inc(ctx, requestAttrs...)

	// Duration and size attributes (only method and route for lower cardinality)
	baseAttrs := []attribute.KeyValue{
		telemetry.AttrHTTPMethod.String(method),
		telemetry.AttrHTTPRoute.String(route),
	}
	metrics.requestDuration.RecordDuration(ctx, duration, baseAttrs...)

	if requestSize > 0 {
		metrics.requestSize.Record(ctx, float64(requestSize), baseAttrs...)
	}

	if responseSize > 0 {
		metrics.responseSize.Record(ctx, float64(responseSize), baseAttrs...)
	}
}

// HTTPMetricsStatusGroup returns a helper for grouping status codes.
// This can be used to calculate error rates by status class (2xx, 4xx, 5xx).
func HTTPMetricsStatusGroup(statusCode int) string {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return "2xx"
	case statusCode >= 300 && statusCode < 400:
		return "3xx"
	case statusCode >= 400 && statusCode < 500:
		return "4xx"
	case statusCode >= 500:
		return "5xx"
	default:
		return "other"
	}
}

// HTTPMetricsResponseWriter wraps gin.ResponseWriter to capture response size
// when the standard Size() method returns -1.
type HTTPMetricsResponseWriter struct {
	gin.ResponseWriter
	bytesWritten int
}

// Write captures the number of bytes written.
func (w *HTTPMetricsResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += n
	return n, err
}

// BytesWritten returns the total bytes written to the response.
func (w *HTTPMetricsResponseWriter) BytesWritten() int {
	return w.bytesWritten
}

// ParseStatusCode safely parses a status code string to int.
func ParseStatusCode(s string) int {
	code, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return code
}
