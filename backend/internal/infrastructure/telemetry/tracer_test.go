package telemetry_test

import (
	"context"
	"testing"
	"time"

	"github.com/erp/backend/internal/infrastructure/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestNewTracerProvider_Disabled(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	cfg := telemetry.Config{
		Enabled:           false,
		CollectorEndpoint: "localhost:4317",
		SamplingRatio:     1.0,
		ServiceName:       "test-service",
	}

	tp, err := telemetry.NewTracerProvider(ctx, cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, tp)

	// Verify telemetry is disabled
	assert.False(t, tp.IsEnabled())

	// GetConfig should return the config
	gotCfg := tp.GetConfig()
	assert.Equal(t, cfg.ServiceName, gotCfg.ServiceName)
	assert.False(t, gotCfg.Enabled)

	// Shutdown should succeed with no-op
	err = tp.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestNewTracerProvider_Enabled(t *testing.T) {
	// Skip this test in CI as it requires a real OTEL collector
	// This test is for local development with `make otel-up`
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	cfg := telemetry.Config{
		Enabled:           true,
		CollectorEndpoint: "localhost:4317",
		SamplingRatio:     1.0,
		ServiceName:       "test-service",
	}

	tp, err := telemetry.NewTracerProvider(ctx, cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, tp)

	// Verify telemetry is enabled
	assert.True(t, tp.IsEnabled())

	// Get a tracer and create a span
	tracer := tp.Tracer("test")
	_, span := tracer.Start(ctx, "test-span")
	span.End()

	// Force flush to ensure spans are exported
	err = tp.ForceFlush(ctx)
	assert.NoError(t, err)

	// Shutdown should succeed
	err = tp.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestNewTracerProvider_SamplingRatios(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	tests := []struct {
		name          string
		samplingRatio float64
		wantEnabled   bool
	}{
		{"always_sample", 1.0, false}, // disabled telemetry
		{"never_sample", 0.0, false},
		{"ratio_sample", 0.5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := telemetry.Config{
				Enabled:           false, // Keep disabled for unit tests
				CollectorEndpoint: "localhost:4317",
				SamplingRatio:     tt.samplingRatio,
				ServiceName:       "test-service",
			}

			tp, err := telemetry.NewTracerProvider(ctx, cfg, logger)
			require.NoError(t, err)
			assert.Equal(t, tt.wantEnabled, tp.IsEnabled())

			// Shutdown should always succeed
			err = tp.Shutdown(ctx)
			assert.NoError(t, err)
		})
	}
}

func TestTracerProvider_Tracer(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	cfg := telemetry.Config{
		Enabled:           false,
		CollectorEndpoint: "localhost:4317",
		SamplingRatio:     1.0,
		ServiceName:       "test-service",
	}

	tp, err := telemetry.NewTracerProvider(ctx, cfg, logger)
	require.NoError(t, err)

	// Get a tracer even when disabled (should return no-op tracer)
	tracer := tp.Tracer("test-tracer")
	require.NotNil(t, tracer)

	// Creating a span should work (no-op)
	_, span := tracer.Start(ctx, "test-span")
	span.End()
}

func TestTracerProvider_ForceFlush_Disabled(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	cfg := telemetry.Config{
		Enabled:           false,
		CollectorEndpoint: "localhost:4317",
		SamplingRatio:     1.0,
		ServiceName:       "test-service",
	}

	tp, err := telemetry.NewTracerProvider(ctx, cfg, logger)
	require.NoError(t, err)

	// ForceFlush should succeed when disabled
	err = tp.ForceFlush(ctx)
	assert.NoError(t, err)
}

func TestTracerProvider_ShutdownTimeout(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	cfg := telemetry.Config{
		Enabled:           false,
		CollectorEndpoint: "localhost:4317",
		SamplingRatio:     1.0,
		ServiceName:       "test-service",
	}

	tp, err := telemetry.NewTracerProvider(ctx, cfg, logger)
	require.NoError(t, err)

	// Shutdown with a cancelled context should still succeed for disabled provider
	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()

	err = tp.Shutdown(cancelledCtx)
	assert.NoError(t, err)
}

func TestConfig_Defaults(t *testing.T) {
	cfg := telemetry.Config{}

	// Verify zero values
	assert.False(t, cfg.Enabled)
	assert.Empty(t, cfg.CollectorEndpoint)
	assert.Zero(t, cfg.SamplingRatio)
	assert.Empty(t, cfg.ServiceName)
}

func TestNewTracerProvider_InvalidEndpoint(t *testing.T) {
	// Skip in short mode as this may try to connect
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := zaptest.NewLogger(t, zaptest.Level(zap.ErrorLevel))
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	cfg := telemetry.Config{
		Enabled:           true,
		CollectorEndpoint: "invalid-host:99999", // Invalid endpoint
		SamplingRatio:     1.0,
		ServiceName:       "test-service",
	}

	// Creation may succeed but spans won't be exported
	// The exporter handles connection errors gracefully
	tp, err := telemetry.NewTracerProvider(ctx, cfg, logger)
	if err != nil {
		// Connection error is expected with invalid endpoint
		t.Logf("Expected connection error: %v", err)
		return
	}

	// If creation succeeded, shutdown should still work
	_ = tp.Shutdown(context.Background())
}
