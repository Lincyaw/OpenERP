package telemetry_test

import (
	"context"
	"testing"
	"time"

	"github.com/erp/backend/internal/infrastructure/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestNewMeterProvider_Disabled(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	cfg := telemetry.MetricsConfig{
		Enabled:           false,
		CollectorEndpoint: "localhost:14317",
		ExportInterval:    60 * time.Second,
		ServiceName:       "test-service",
	}

	mp, err := telemetry.NewMeterProvider(ctx, cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, mp)

	// Verify metrics are disabled
	assert.False(t, mp.IsEnabled())

	// GetConfig should return the config
	gotCfg := mp.GetConfig()
	assert.Equal(t, cfg.ServiceName, gotCfg.ServiceName)
	assert.False(t, gotCfg.Enabled)

	// Shutdown should succeed with no-op
	err = mp.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestNewMeterProvider_Enabled(t *testing.T) {
	// Skip this test in CI as it requires a real OTEL collector
	// This test is for local development with `make otel-up`
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	cfg := telemetry.MetricsConfig{
		Enabled:           true,
		CollectorEndpoint: "localhost:14317",
		ExportInterval:    1 * time.Second, // Short interval for testing
		ServiceName:       "test-service",
		Insecure:          true,
	}

	mp, err := telemetry.NewMeterProvider(ctx, cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, mp)

	// Verify metrics are enabled
	assert.True(t, mp.IsEnabled())

	// Get a meter and create metrics
	meter := mp.Meter("test")
	require.NotNil(t, meter)

	// Force flush to ensure metrics are exported
	err = mp.ForceFlush(ctx)
	assert.NoError(t, err)

	// Shutdown should succeed
	err = mp.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestMeterProvider_Meter(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	cfg := telemetry.MetricsConfig{
		Enabled:           false,
		CollectorEndpoint: "localhost:14317",
		ExportInterval:    60 * time.Second,
		ServiceName:       "test-service",
	}

	mp, err := telemetry.NewMeterProvider(ctx, cfg, logger)
	require.NoError(t, err)

	// Get a meter even when disabled (should return no-op meter)
	meter := mp.Meter("test-meter")
	require.NotNil(t, meter)
}

func TestMeterProvider_ForceFlush_Disabled(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	cfg := telemetry.MetricsConfig{
		Enabled:           false,
		CollectorEndpoint: "localhost:14317",
		ExportInterval:    60 * time.Second,
		ServiceName:       "test-service",
	}

	mp, err := telemetry.NewMeterProvider(ctx, cfg, logger)
	require.NoError(t, err)

	// ForceFlush should succeed when disabled
	err = mp.ForceFlush(ctx)
	assert.NoError(t, err)
}

func TestMeterProvider_ShutdownTimeout(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	cfg := telemetry.MetricsConfig{
		Enabled:           false,
		CollectorEndpoint: "localhost:14317",
		ExportInterval:    60 * time.Second,
		ServiceName:       "test-service",
	}

	mp, err := telemetry.NewMeterProvider(ctx, cfg, logger)
	require.NoError(t, err)

	// Shutdown with a cancelled context should still succeed for disabled provider
	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()

	err = mp.Shutdown(cancelledCtx)
	assert.NoError(t, err)
}

func TestMetricsConfig_Defaults(t *testing.T) {
	cfg := telemetry.MetricsConfig{}

	// Verify zero values
	assert.False(t, cfg.Enabled)
	assert.Empty(t, cfg.CollectorEndpoint)
	assert.Zero(t, cfg.ExportInterval)
	assert.Empty(t, cfg.ServiceName)
}

func TestMeterProvider_DefaultExportInterval(t *testing.T) {
	// Skip this test in CI as it requires a real OTEL collector
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	// Create with zero export interval (should default to 60s)
	cfg := telemetry.MetricsConfig{
		Enabled:           true,
		CollectorEndpoint: "localhost:14317",
		ExportInterval:    0, // Zero - should default to 60s
		ServiceName:       "test-service",
		Insecure:          true,
	}

	mp, err := telemetry.NewMeterProvider(ctx, cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, mp)

	// Cleanup
	_ = mp.Shutdown(ctx)
}

func TestNewMeterProvider_InvalidEndpoint(t *testing.T) {
	// Skip in short mode as this may try to connect
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := zaptest.NewLogger(t, zaptest.Level(zap.ErrorLevel))
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	cfg := telemetry.MetricsConfig{
		Enabled:           true,
		CollectorEndpoint: "invalid-host:99999", // Invalid endpoint
		ExportInterval:    1 * time.Second,
		ServiceName:       "test-service",
	}

	// Creation may succeed but metrics won't be exported
	// The exporter handles connection errors gracefully
	mp, err := telemetry.NewMeterProvider(ctx, cfg, logger)
	if err != nil {
		// Connection error is expected with invalid endpoint
		t.Logf("Expected connection error: %v", err)
		return
	}

	// If creation succeeded, shutdown should still work
	_ = mp.Shutdown(context.Background())
}

// ============================================================================
// Counter Tests
// ============================================================================

func TestCounter_Add(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	cfg := telemetry.MetricsConfig{
		Enabled:           false,
		CollectorEndpoint: "localhost:14317",
		ServiceName:       "test-service",
	}

	mp, err := telemetry.NewMeterProvider(ctx, cfg, logger)
	require.NoError(t, err)

	meter := mp.Meter("test")
	counter, err := telemetry.NewCounter(meter, "test_counter", "Test counter description", "1")
	require.NoError(t, err)
	require.NotNil(t, counter)

	// Add values with attributes
	counter.Add(ctx, 5, attribute.String("method", "GET"))
	counter.Add(ctx, 10, attribute.String("method", "POST"))

	// Inc should work
	counter.Inc(ctx, attribute.String("method", "DELETE"))
}

func TestCounter_Inc(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	cfg := telemetry.MetricsConfig{
		Enabled:           false,
		CollectorEndpoint: "localhost:14317",
		ServiceName:       "test-service",
	}

	mp, err := telemetry.NewMeterProvider(ctx, cfg, logger)
	require.NoError(t, err)

	meter := mp.Meter("test")
	counter, err := telemetry.NewCounter(meter, "request_count", "Request count", "{request}")
	require.NoError(t, err)

	// Inc multiple times
	counter.Inc(ctx)
	counter.Inc(ctx, attribute.String("status", "success"))
	counter.Inc(ctx, attribute.String("status", "error"))
}

// ============================================================================
// Histogram Tests
// ============================================================================

func TestHistogram_Record(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	cfg := telemetry.MetricsConfig{
		Enabled:           false,
		CollectorEndpoint: "localhost:14317",
		ServiceName:       "test-service",
	}

	mp, err := telemetry.NewMeterProvider(ctx, cfg, logger)
	require.NoError(t, err)

	meter := mp.Meter("test")
	histogram, err := telemetry.NewHistogram(meter, telemetry.HistogramOpts{
		Name:        "http_request_duration_seconds",
		Description: "HTTP request duration",
		Unit:        "s",
		Boundaries:  telemetry.HTTPDurationBuckets,
	})
	require.NoError(t, err)
	require.NotNil(t, histogram)

	// Record values
	histogram.Record(ctx, 0.005)
	histogram.Record(ctx, 0.1, attribute.String("route", "/api/v1/products"))
	histogram.Record(ctx, 2.5, attribute.String("route", "/api/v1/orders"))
}

func TestHistogram_RecordDuration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	cfg := telemetry.MetricsConfig{
		Enabled:           false,
		CollectorEndpoint: "localhost:14317",
		ServiceName:       "test-service",
	}

	mp, err := telemetry.NewMeterProvider(ctx, cfg, logger)
	require.NoError(t, err)

	meter := mp.Meter("test")
	histogram, err := telemetry.NewHistogram(meter, telemetry.HistogramOpts{
		Name:        "db_query_duration_seconds",
		Description: "Database query duration",
		Unit:        "s",
		Boundaries:  telemetry.DBDurationBuckets,
	})
	require.NoError(t, err)

	// Record durations
	histogram.RecordDuration(ctx, 5*time.Millisecond)
	histogram.RecordDuration(ctx, 100*time.Millisecond, attribute.String("operation", "SELECT"))
	histogram.RecordDuration(ctx, 1*time.Second, attribute.String("operation", "INSERT"))
}

func TestHistogram_CustomBoundaries(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	cfg := telemetry.MetricsConfig{
		Enabled:           false,
		CollectorEndpoint: "localhost:14317",
		ServiceName:       "test-service",
	}

	mp, err := telemetry.NewMeterProvider(ctx, cfg, logger)
	require.NoError(t, err)

	meter := mp.Meter("test")

	// Create histogram with custom boundaries
	customBoundaries := []float64{0.1, 0.5, 1.0, 5.0, 10.0}
	histogram, err := telemetry.NewHistogram(meter, telemetry.HistogramOpts{
		Name:        "custom_histogram",
		Description: "Custom histogram with specific boundaries",
		Unit:        "s",
		Boundaries:  customBoundaries,
	})
	require.NoError(t, err)
	require.NotNil(t, histogram)

	histogram.Record(ctx, 0.25)
}

func TestHistogram_NoBoundaries(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	cfg := telemetry.MetricsConfig{
		Enabled:           false,
		CollectorEndpoint: "localhost:14317",
		ServiceName:       "test-service",
	}

	mp, err := telemetry.NewMeterProvider(ctx, cfg, logger)
	require.NoError(t, err)

	meter := mp.Meter("test")

	// Create histogram without custom boundaries (uses SDK defaults)
	histogram, err := telemetry.NewHistogram(meter, telemetry.HistogramOpts{
		Name:        "default_histogram",
		Description: "Histogram with default boundaries",
		Unit:        "s",
	})
	require.NoError(t, err)
	require.NotNil(t, histogram)

	histogram.Record(ctx, 1.5)
}

// ============================================================================
// Gauge Tests
// ============================================================================

func TestGauge_Record(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	cfg := telemetry.MetricsConfig{
		Enabled:           false,
		CollectorEndpoint: "localhost:14317",
		ServiceName:       "test-service",
	}

	mp, err := telemetry.NewMeterProvider(ctx, cfg, logger)
	require.NoError(t, err)

	meter := mp.Meter("test")
	gauge, err := telemetry.NewGauge(meter, "active_connections", "Number of active connections", "{connection}")
	require.NoError(t, err)
	require.NotNil(t, gauge)

	// Record values
	gauge.Record(ctx, 10)
	gauge.Record(ctx, 15, attribute.String("pool", "db"))
	gauge.Record(ctx, 5, attribute.String("pool", "redis"))
}

func TestFloatGauge_Record(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	cfg := telemetry.MetricsConfig{
		Enabled:           false,
		CollectorEndpoint: "localhost:14317",
		ServiceName:       "test-service",
	}

	mp, err := telemetry.NewMeterProvider(ctx, cfg, logger)
	require.NoError(t, err)

	meter := mp.Meter("test")
	gauge, err := telemetry.NewFloatGauge(meter, "cpu_usage_percent", "CPU usage percentage", "%")
	require.NoError(t, err)
	require.NotNil(t, gauge)

	// Record values
	gauge.Record(ctx, 45.5)
	gauge.Record(ctx, 78.2, attribute.String("core", "0"))
	gauge.Record(ctx, 23.1, attribute.String("core", "1"))
}

// ============================================================================
// Common Attributes Tests
// ============================================================================

func TestCommonAttributes(t *testing.T) {
	// Verify attribute keys are properly defined
	assert.Equal(t, "tenant_id", string(telemetry.AttrTenantID))
	assert.Equal(t, "user_id", string(telemetry.AttrUserID))
	assert.Equal(t, "http.method", string(telemetry.AttrHTTPMethod))
	assert.Equal(t, "http.status_code", string(telemetry.AttrHTTPStatusCode))
	assert.Equal(t, "http.route", string(telemetry.AttrHTTPRoute))
	assert.Equal(t, "db.operation", string(telemetry.AttrDBOperation))
	assert.Equal(t, "db.table", string(telemetry.AttrDBTable))
	assert.Equal(t, "db.pool.state", string(telemetry.AttrDBState))
	assert.Equal(t, "order_type", string(telemetry.AttrOrderType))
	assert.Equal(t, "payment_method", string(telemetry.AttrPaymentMethod))
	assert.Equal(t, "payment_status", string(telemetry.AttrPaymentStatus))
	assert.Equal(t, "warehouse_id", string(telemetry.AttrWarehouseID))
	assert.Equal(t, "product_id", string(telemetry.AttrProductID))
	assert.Equal(t, "category_id", string(telemetry.AttrCategoryID))
}

// ============================================================================
// Default Bucket Boundaries Tests
// ============================================================================

func TestDefaultBuckets(t *testing.T) {
	// Verify HTTP duration buckets
	assert.Equal(t, []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}, telemetry.HTTPDurationBuckets)

	// Verify DB duration buckets
	assert.Equal(t, []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5}, telemetry.DBDurationBuckets)

	// Verify small duration buckets
	assert.Equal(t, []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1}, telemetry.SmallDurationBuckets)
}

func TestHistogram_WithHTTPDurationBuckets(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	cfg := telemetry.MetricsConfig{
		Enabled:           false,
		CollectorEndpoint: "localhost:14317",
		ServiceName:       "test-service",
	}

	mp, err := telemetry.NewMeterProvider(ctx, cfg, logger)
	require.NoError(t, err)

	meter := mp.Meter("test")
	histogram, err := telemetry.NewHistogram(meter, telemetry.HistogramOpts{
		Name:        "http_server_request_duration_seconds",
		Description: "HTTP server request duration",
		Unit:        "s",
		Boundaries:  telemetry.HTTPDurationBuckets,
	})
	require.NoError(t, err)

	// Record some typical HTTP request durations
	histogram.Record(ctx, 0.005, telemetry.AttrHTTPMethod.String("GET"))  // 5ms
	histogram.Record(ctx, 0.05, telemetry.AttrHTTPMethod.String("POST"))  // 50ms
	histogram.Record(ctx, 0.5, telemetry.AttrHTTPMethod.String("PUT"))    // 500ms
	histogram.Record(ctx, 5.0, telemetry.AttrHTTPMethod.String("DELETE")) // 5s (slow)
}

func TestHistogram_WithDBDurationBuckets(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	cfg := telemetry.MetricsConfig{
		Enabled:           false,
		CollectorEndpoint: "localhost:14317",
		ServiceName:       "test-service",
	}

	mp, err := telemetry.NewMeterProvider(ctx, cfg, logger)
	require.NoError(t, err)

	meter := mp.Meter("test")
	histogram, err := telemetry.NewHistogram(meter, telemetry.HistogramOpts{
		Name:        "db_query_duration_seconds",
		Description: "Database query duration",
		Unit:        "s",
		Boundaries:  telemetry.DBDurationBuckets,
	})
	require.NoError(t, err)

	// Record some typical DB query durations
	histogram.Record(ctx, 0.001, telemetry.AttrDBOperation.String("SELECT")) // 1ms
	histogram.Record(ctx, 0.01, telemetry.AttrDBOperation.String("INSERT"))  // 10ms
	histogram.Record(ctx, 0.1, telemetry.AttrDBOperation.String("UPDATE"))   // 100ms
	histogram.Record(ctx, 1.0, telemetry.AttrDBOperation.String("DELETE"))   // 1s (slow)
}
