package telemetry

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestNewLoggerProvider_Disabled(t *testing.T) {
	ctx := context.Background()
	baseLogger := zap.NewNop()

	cfg := LogsConfig{
		Enabled:           false,
		CollectorEndpoint: "localhost:14317",
		ServiceName:       "test-service",
		Insecure:          true,
	}

	provider, err := NewLoggerProvider(ctx, cfg, baseLogger)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.False(t, provider.IsEnabled())
	assert.Nil(t, provider.GetLoggerProvider())

	// Shutdown should be safe
	err = provider.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestLoggerProvider_GetConfig(t *testing.T) {
	ctx := context.Background()
	baseLogger := zap.NewNop()

	cfg := LogsConfig{
		Enabled:           false,
		CollectorEndpoint: "localhost:14317",
		ServiceName:       "test-service",
		Insecure:          true,
	}

	provider, err := NewLoggerProvider(ctx, cfg, baseLogger)
	require.NoError(t, err)

	returnedConfig := provider.GetConfig()
	assert.Equal(t, cfg.Enabled, returnedConfig.Enabled)
	assert.Equal(t, cfg.CollectorEndpoint, returnedConfig.CollectorEndpoint)
	assert.Equal(t, cfg.ServiceName, returnedConfig.ServiceName)
	assert.Equal(t, cfg.Insecure, returnedConfig.Insecure)
}

func TestLoggerProvider_ForceFlush_Disabled(t *testing.T) {
	ctx := context.Background()
	baseLogger := zap.NewNop()

	cfg := LogsConfig{
		Enabled: false,
	}

	provider, err := NewLoggerProvider(ctx, cfg, baseLogger)
	require.NoError(t, err)

	// ForceFlush on disabled provider should not error
	err = provider.ForceFlush(ctx)
	assert.NoError(t, err)
}

func TestNewZapOTELCore_NilProvider(t *testing.T) {
	cfg := ZapBridgeConfig{
		ServiceName:    "test-service",
		LoggerProvider: nil,
		Level:          zapcore.InfoLevel,
	}

	core := NewZapOTELCore(cfg)
	assert.NotNil(t, core)

	// Should return nop core
	assert.False(t, core.Enabled(zapcore.InfoLevel))
}

func TestNewZapOTELCore_DisabledProvider(t *testing.T) {
	ctx := context.Background()
	baseLogger := zap.NewNop()

	logsProvider, err := NewLoggerProvider(ctx, LogsConfig{Enabled: false}, baseLogger)
	require.NoError(t, err)

	cfg := ZapBridgeConfig{
		ServiceName:    "test-service",
		LoggerProvider: logsProvider,
		Level:          zapcore.InfoLevel,
	}

	core := NewZapOTELCore(cfg)
	assert.NotNil(t, core)

	// Should return nop core because provider is disabled
	assert.False(t, core.Enabled(zapcore.InfoLevel))
}

func TestNewBridgedLogger(t *testing.T) {
	// Create an observer core to capture logs
	observedZapCore, observedLogs := observer.New(zapcore.InfoLevel)

	// Create a nop core for OTEL (since we don't have a real collector)
	otelCore := zapcore.NewNopCore()

	// Create bridged logger
	logger := NewBridgedLogger(observedZapCore, otelCore, zap.AddCaller())

	// Log some messages
	logger.Info("test message", zap.String("key", "value"))
	logger.Debug("debug message") // Should not appear (below InfoLevel)
	logger.Warn("warning message")

	// Verify logs were captured
	logs := observedLogs.All()
	assert.Len(t, logs, 2) // Info and Warn only

	assert.Equal(t, "test message", logs[0].Message)
	assert.Equal(t, zapcore.InfoLevel, logs[0].Level)
	assert.Contains(t, logs[0].Context, zap.String("key", "value"))

	assert.Equal(t, "warning message", logs[1].Message)
	assert.Equal(t, zapcore.WarnLevel, logs[1].Level)
}

func TestCreateBridgedLoggerFromConfig(t *testing.T) {
	ctx := context.Background()
	baseLogger := zap.NewNop()

	// Create disabled logs provider
	logsProvider, err := NewLoggerProvider(ctx, LogsConfig{Enabled: false}, baseLogger)
	require.NoError(t, err)

	baseConfig := &BaseLoggerConfig{
		Level:      "info",
		Format:     "json",
		Output:     "stdout",
		TimeFormat: "2006-01-02T15:04:05.000Z07:00",
	}

	logger, err := CreateBridgedLoggerFromConfig(baseConfig, logsProvider, "test-service")
	require.NoError(t, err)
	assert.NotNil(t, logger)
}

func TestDefaultBaseLoggerConfig(t *testing.T) {
	cfg := DefaultBaseLoggerConfig()
	assert.Equal(t, "info", cfg.Level)
	assert.Equal(t, "console", cfg.Format)
	assert.Equal(t, "stdout", cfg.Output)
	assert.NotEmpty(t, cfg.TimeFormat)
}

func TestParseLogLevel(t *testing.T) {
	testCases := []struct {
		input    string
		expected zapcore.Level
	}{
		{"debug", zapcore.DebugLevel},
		{"info", zapcore.InfoLevel},
		{"warn", zapcore.WarnLevel},
		{"warning", zapcore.WarnLevel},
		{"error", zapcore.ErrorLevel},
		{"fatal", zapcore.FatalLevel},
		{"unknown", zapcore.InfoLevel}, // default
		{"", zapcore.InfoLevel},        // default
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := parseLogLevel(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCreateLogEncoder_JSON(t *testing.T) {
	cfg := &BaseLoggerConfig{
		Format:     "json",
		TimeFormat: "2006-01-02T15:04:05.000Z07:00",
	}

	encoder := createLogEncoder(cfg)
	assert.NotNil(t, encoder)

	// Verify it creates JSON output
	buf, err := encoder.EncodeEntry(
		zapcore.Entry{
			Level:   zapcore.InfoLevel,
			Message: "test",
		},
		nil,
	)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"level":"info"`)
	assert.Contains(t, buf.String(), `"msg":"test"`)
}

func TestCreateLogEncoder_Console(t *testing.T) {
	cfg := &BaseLoggerConfig{
		Format:     "console",
		TimeFormat: "2006-01-02T15:04:05.000Z07:00",
	}

	encoder := createLogEncoder(cfg)
	assert.NotNil(t, encoder)

	// Verify it creates console output (not JSON)
	buf, err := encoder.EncodeEntry(
		zapcore.Entry{
			Level:   zapcore.InfoLevel,
			Message: "test",
		},
		nil,
	)
	require.NoError(t, err)
	// Console encoder doesn't produce JSON format
	assert.NotContains(t, buf.String(), `"level"`)
}

func TestCreateLogWriter(t *testing.T) {
	// Test stdout
	writer := createLogWriter("stdout")
	assert.NotNil(t, writer)

	// Test stderr
	writer = createLogWriter("stderr")
	assert.NotNil(t, writer)

	// Test file path (falls back to stdout)
	writer = createLogWriter("/tmp/test.log")
	assert.NotNil(t, writer)
}

func TestCreateBaseCore(t *testing.T) {
	cfg := &BaseLoggerConfig{
		Level:      "info",
		Format:     "json",
		Output:     "stdout",
		TimeFormat: "2006-01-02T15:04:05.000Z07:00",
	}

	core := createBaseCore(cfg)
	assert.NotNil(t, core)

	// Test that level filtering works
	assert.True(t, core.Enabled(zapcore.InfoLevel))
	assert.True(t, core.Enabled(zapcore.WarnLevel))
	assert.False(t, core.Enabled(zapcore.DebugLevel))
}

func TestLevelFilterCore(t *testing.T) {
	// Create an observer core to capture logs
	observedZapCore, observedLogs := observer.New(zapcore.DebugLevel)

	// Wrap with level filter
	filteredCore := &levelFilterCore{
		Core:     observedZapCore,
		minLevel: zapcore.WarnLevel,
	}

	// Test Enabled
	assert.True(t, filteredCore.Enabled(zapcore.WarnLevel))
	assert.True(t, filteredCore.Enabled(zapcore.ErrorLevel))
	assert.False(t, filteredCore.Enabled(zapcore.InfoLevel))
	assert.False(t, filteredCore.Enabled(zapcore.DebugLevel))

	// Test logging through the filtered core
	logger := zap.New(filteredCore)
	logger.Debug("debug") // Should not appear
	logger.Info("info")   // Should not appear
	logger.Warn("warn")   // Should appear
	logger.Error("error") // Should appear

	logs := observedLogs.All()
	assert.Len(t, logs, 2)
	assert.Equal(t, "warn", logs[0].Message)
	assert.Equal(t, "error", logs[1].Message)
}

func TestLevelFilterCore_With(t *testing.T) {
	// Create an observer core
	observedZapCore, observedLogs := observer.New(zapcore.DebugLevel)

	// Wrap with level filter
	filteredCore := &levelFilterCore{
		Core:     observedZapCore,
		minLevel: zapcore.WarnLevel,
	}

	// Add fields with With
	childCore := filteredCore.With([]zapcore.Field{zap.String("service", "test")})
	assert.NotNil(t, childCore)

	// Verify it's still a levelFilterCore
	lfCore, ok := childCore.(*levelFilterCore)
	assert.True(t, ok)
	assert.Equal(t, zapcore.WarnLevel, lfCore.minLevel)

	// Log with child core
	logger := zap.New(childCore)
	logger.Warn("test message")

	logs := observedLogs.All()
	require.Len(t, logs, 1)
	assert.Equal(t, "test message", logs[0].Message)

	// Check that the field was added
	hasServiceField := false
	for _, field := range logs[0].Context {
		if field.Key == "service" && field.String == "test" {
			hasServiceField = true
			break
		}
	}
	assert.True(t, hasServiceField, "service field should be present")
}

func TestCreateLogWriter_Stdout(t *testing.T) {
	writer := createLogWriter("stdout")
	assert.NotNil(t, writer)
}

func TestCreateLogWriter_Stderr(t *testing.T) {
	writer := createLogWriter("stderr")
	assert.NotNil(t, writer)
}

func TestLoggerBridge_LevelMapping(t *testing.T) {
	// This test verifies that zap levels are correctly enabled
	// We can't test actual OTEL output without a collector, but we can
	// test that the core configuration is correct

	testCases := []struct {
		name          string
		configLevel   zapcore.Level
		testLevel     zapcore.Level
		shouldBeValid bool
	}{
		{"debug config, debug test", zapcore.DebugLevel, zapcore.DebugLevel, true},
		{"debug config, info test", zapcore.DebugLevel, zapcore.InfoLevel, true},
		{"info config, debug test", zapcore.InfoLevel, zapcore.DebugLevel, false},
		{"info config, info test", zapcore.InfoLevel, zapcore.InfoLevel, true},
		{"warn config, info test", zapcore.WarnLevel, zapcore.InfoLevel, false},
		{"warn config, warn test", zapcore.WarnLevel, zapcore.WarnLevel, true},
		{"error config, warn test", zapcore.ErrorLevel, zapcore.WarnLevel, false},
		{"error config, error test", zapcore.ErrorLevel, zapcore.ErrorLevel, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a level enabler
			core := zapcore.NewCore(
				zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
				zapcore.AddSync(&bytes.Buffer{}),
				tc.configLevel,
			)

			result := core.Enabled(tc.testLevel)
			assert.Equal(t, tc.shouldBeValid, result)
		})
	}
}

func TestLoggerBridge_Integration(t *testing.T) {
	// Integration test to verify the full setup flow
	ctx := context.Background()
	baseLogger := zap.NewNop()

	// Create disabled logs provider (no collector needed)
	logsProvider, err := NewLoggerProvider(ctx, LogsConfig{
		Enabled:           false,
		CollectorEndpoint: "localhost:14317",
		ServiceName:       "test-service",
		Insecure:          true,
	}, baseLogger)
	require.NoError(t, err)

	// Create base config
	baseConfig := &BaseLoggerConfig{
		Level:      "debug",
		Format:     "json",
		Output:     "stdout",
		TimeFormat: "2006-01-02T15:04:05.000Z07:00",
	}

	// Create bridged logger
	logger, err := CreateBridgedLoggerFromConfig(baseConfig, logsProvider, "integration-test")
	require.NoError(t, err)
	require.NotNil(t, logger)

	// Verify logger works (writes to stdout, OTEL core is nop)
	logger.Info("integration test message",
		zap.String("request_id", "req-123"),
		zap.String("tenant_id", "tenant-456"),
		zap.String("user_id", "user-789"),
	)

	// Cleanup
	logger.Sync()
}

func TestLoggerProvider_Shutdown_MultipleCalls(t *testing.T) {
	ctx := context.Background()
	baseLogger := zap.NewNop()

	provider, err := NewLoggerProvider(ctx, LogsConfig{Enabled: false}, baseLogger)
	require.NoError(t, err)

	// Multiple shutdown calls should be safe
	err = provider.Shutdown(ctx)
	assert.NoError(t, err)

	err = provider.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestZapBridgeConfig_AllFields(t *testing.T) {
	cfg := ZapBridgeConfig{
		ServiceName:    "test-service",
		LoggerProvider: nil,
		Level:          zapcore.WarnLevel,
	}

	assert.Equal(t, "test-service", cfg.ServiceName)
	assert.Nil(t, cfg.LoggerProvider)
	assert.Equal(t, zapcore.WarnLevel, cfg.Level)
}

func TestLogsConfig_AllFields(t *testing.T) {
	cfg := LogsConfig{
		Enabled:           true,
		CollectorEndpoint: "localhost:14317",
		ServiceName:       "test-service",
		Insecure:          false,
	}

	assert.True(t, cfg.Enabled)
	assert.Equal(t, "localhost:14317", cfg.CollectorEndpoint)
	assert.Equal(t, "test-service", cfg.ServiceName)
	assert.False(t, cfg.Insecure)
}

// TestNewLoggerProvider_EnabledButNoCollector tests that creating an enabled
// provider without a running collector still works (it will buffer logs).
func TestNewLoggerProvider_EnabledButNoCollector(t *testing.T) {
	ctx := context.Background()
	baseLogger := zap.NewNop()

	cfg := LogsConfig{
		Enabled:           true,
		CollectorEndpoint: "localhost:19999", // Non-existent endpoint
		ServiceName:       "test-service",
		Insecure:          true,
	}

	// This should succeed even without a collector - it creates the provider
	// and exporter, which will buffer logs until they can be sent
	provider, err := NewLoggerProvider(ctx, cfg, baseLogger)
	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.True(t, provider.IsEnabled())
	assert.NotNil(t, provider.GetLoggerProvider())

	// Clean shutdown
	err = provider.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestNewZapOTELCore_WithEnabledProvider(t *testing.T) {
	ctx := context.Background()
	baseLogger := zap.NewNop()

	// Create enabled provider (even without collector, the core should work)
	logsProvider, err := NewLoggerProvider(ctx, LogsConfig{
		Enabled:           true,
		CollectorEndpoint: "localhost:19999",
		ServiceName:       "test-service",
		Insecure:          true,
	}, baseLogger)
	require.NoError(t, err)
	defer logsProvider.Shutdown(ctx)

	// Test with default level (DebugLevel)
	cfg := ZapBridgeConfig{
		ServiceName:    "test-service",
		LoggerProvider: logsProvider,
		Level:          zapcore.DebugLevel,
	}

	core := NewZapOTELCore(cfg)
	assert.NotNil(t, core)

	// The core should be enabled for all levels
	assert.True(t, core.Enabled(zapcore.DebugLevel))
	assert.True(t, core.Enabled(zapcore.InfoLevel))
	assert.True(t, core.Enabled(zapcore.WarnLevel))
	assert.True(t, core.Enabled(zapcore.ErrorLevel))
}

func TestNewZapOTELCore_WithLevelFilter(t *testing.T) {
	ctx := context.Background()
	baseLogger := zap.NewNop()

	// Create enabled provider
	logsProvider, err := NewLoggerProvider(ctx, LogsConfig{
		Enabled:           true,
		CollectorEndpoint: "localhost:19999",
		ServiceName:       "test-service",
		Insecure:          true,
	}, baseLogger)
	require.NoError(t, err)
	defer logsProvider.Shutdown(ctx)

	// Test with WarnLevel filter
	cfg := ZapBridgeConfig{
		ServiceName:    "test-service",
		LoggerProvider: logsProvider,
		Level:          zapcore.WarnLevel,
	}

	core := NewZapOTELCore(cfg)
	assert.NotNil(t, core)

	// The core should be wrapped with level filter
	_, isFiltered := core.(*levelFilterCore)
	assert.True(t, isFiltered, "core should be wrapped with levelFilterCore")

	// Should only be enabled for Warn and above
	assert.False(t, core.Enabled(zapcore.DebugLevel))
	assert.False(t, core.Enabled(zapcore.InfoLevel))
	assert.True(t, core.Enabled(zapcore.WarnLevel))
	assert.True(t, core.Enabled(zapcore.ErrorLevel))
}

func TestLogAttributeMapping(t *testing.T) {
	// Test that various zap field types are handled correctly
	// This tests the basic encoder functionality with our configuration

	var buf bytes.Buffer
	encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.DebugLevel)
	logger := zap.New(core)

	// Log with various field types
	logger.Info("test",
		zap.String("string_field", "value"),
		zap.Int("int_field", 42),
		zap.Float64("float_field", 3.14),
		zap.Bool("bool_field", true),
		zap.Strings("strings_field", []string{"a", "b"}),
	)

	output := buf.String()

	// Verify fields are present in output
	assert.Contains(t, output, `"string_field":"value"`)
	assert.Contains(t, output, `"int_field":42`)
	assert.True(t, strings.Contains(output, `"float_field":3.14`) || strings.Contains(output, `"float_field":3.1`))
	assert.Contains(t, output, `"bool_field":true`)
	assert.Contains(t, output, `"strings_field":["a","b"]`)
}
