// Package telemetry provides OpenTelemetry integration for logs collection.
// This file implements the Zap -> OpenTelemetry log bridge.
package telemetry

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogsConfig holds logs bridge configuration.
type LogsConfig struct {
	Enabled           bool
	CollectorEndpoint string
	ServiceName       string
	Insecure          bool
}

// LoggerProvider wraps the OpenTelemetry LoggerProvider with lifecycle management.
type LoggerProvider struct {
	provider *sdklog.LoggerProvider
	logger   *zap.Logger
	config   LogsConfig
}

// NewLoggerProvider creates and configures a new OpenTelemetry LoggerProvider.
// If logs are disabled, it returns a provider that wraps the no-op global logger.
func NewLoggerProvider(ctx context.Context, cfg LogsConfig, logger *zap.Logger) (*LoggerProvider, error) {
	lp := &LoggerProvider{
		logger: logger,
		config: cfg,
	}

	if !cfg.Enabled {
		logger.Info("OTEL Logs disabled, using no-op logger provider")
		return lp, nil
	}

	// Create OTLP gRPC exporter for logs
	exporterOpts := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(cfg.CollectorEndpoint),
	}
	if cfg.Insecure {
		exporterOpts = append(exporterOpts, otlploggrpc.WithInsecure())
	}

	exporter, err := otlploggrpc.New(ctx, exporterOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP logs exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create LoggerProvider with batch processor
	lp.provider = sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(
			sdklog.NewBatchProcessor(exporter),
		),
	)

	// Set global logger provider
	global.SetLoggerProvider(lp.provider)

	logger.Info("OpenTelemetry LoggerProvider initialized",
		zap.String("collector_endpoint", cfg.CollectorEndpoint),
		zap.String("service_name", cfg.ServiceName),
	)

	return lp, nil
}

// Shutdown gracefully shuts down the logger provider, flushing any pending logs.
// It should be called when the application exits.
func (lp *LoggerProvider) Shutdown(ctx context.Context) error {
	if lp.provider == nil {
		lp.logger.Debug("No logger provider to shutdown (logs disabled)")
		return nil
	}

	lp.logger.Info("Shutting down OpenTelemetry LoggerProvider...")

	// Create a timeout context for shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := lp.provider.Shutdown(shutdownCtx); err != nil {
		lp.logger.Error("Error shutting down logger provider", zap.Error(err))
		return fmt.Errorf("failed to shutdown logger provider: %w", err)
	}

	lp.logger.Info("OpenTelemetry LoggerProvider shutdown complete")
	return nil
}

// IsEnabled returns whether OTEL logs are enabled.
func (lp *LoggerProvider) IsEnabled() bool {
	return lp.config.Enabled && lp.provider != nil
}

// GetConfig returns a copy of the logs configuration.
func (lp *LoggerProvider) GetConfig() LogsConfig {
	return lp.config
}

// ForceFlush immediately exports all logs that have not yet been exported.
// This is useful in tests or when you need to ensure all logs are exported before shutdown.
func (lp *LoggerProvider) ForceFlush(ctx context.Context) error {
	if lp.provider == nil {
		return nil
	}
	return lp.provider.ForceFlush(ctx)
}

// GetLoggerProvider returns the underlying SDK LoggerProvider.
// Returns nil if logs are disabled.
func (lp *LoggerProvider) GetLoggerProvider() *sdklog.LoggerProvider {
	return lp.provider
}

// =============================================================================
// Zap Logger Bridge
// =============================================================================

// ZapBridgeConfig holds configuration for the Zap -> OTEL bridge.
type ZapBridgeConfig struct {
	// ServiceName is used as the logger name in OpenTelemetry
	ServiceName string
	// LoggerProvider is the OpenTelemetry LoggerProvider to use
	LoggerProvider *LoggerProvider
	// Level is the minimum log level to emit
	Level zapcore.Level
}

// NewZapOTELCore creates a new zapcore.Core that bridges Zap logs to OpenTelemetry.
// This core should be combined with existing cores using zapcore.NewTee for
// dual output to both stdout and OTEL.
//
// Example usage:
//
//	// Create OTEL core
//	otelCore := telemetry.NewZapOTELCore(telemetry.ZapBridgeConfig{
//	    ServiceName:    "erp-backend",
//	    LoggerProvider: loggerProvider,
//	    Level:          zapcore.InfoLevel,
//	})
//
//	// Combine with existing stdout core
//	combinedCore := zapcore.NewTee(stdoutCore, otelCore)
//	logger := zap.New(combinedCore)
func NewZapOTELCore(cfg ZapBridgeConfig) zapcore.Core {
	if cfg.LoggerProvider == nil || !cfg.LoggerProvider.IsEnabled() {
		// Return a no-op core if OTEL logs are disabled
		return zapcore.NewNopCore()
	}

	// Build otelzap options
	opts := []otelzap.Option{
		otelzap.WithLoggerProvider(cfg.LoggerProvider.provider),
	}

	// Create the otelzap core
	// The name should be the package/service name for identification
	core := otelzap.NewCore(cfg.ServiceName, opts...)

	// Wrap with level enabler if a specific level is configured
	// The otelzap core itself doesn't have min level, so we wrap it
	if cfg.Level != zapcore.DebugLevel {
		return &levelFilterCore{
			Core:     core,
			minLevel: cfg.Level,
		}
	}

	return core
}

// levelFilterCore wraps a zapcore.Core with level filtering.
type levelFilterCore struct {
	zapcore.Core
	minLevel zapcore.Level
}

// Enabled implements zapcore.Core.
func (c *levelFilterCore) Enabled(lvl zapcore.Level) bool {
	return lvl >= c.minLevel && c.Core.Enabled(lvl)
}

// Check implements zapcore.Core.
func (c *levelFilterCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if !c.Enabled(entry.Level) {
		return ce
	}
	return c.Core.Check(entry, ce)
}

// With implements zapcore.Core.
func (c *levelFilterCore) With(fields []zapcore.Field) zapcore.Core {
	return &levelFilterCore{
		Core:     c.Core.With(fields),
		minLevel: c.minLevel,
	}
}

// NewBridgedLogger creates a new Zap logger that outputs to both the original
// destination (stdout/file) and OpenTelemetry.
//
// Parameters:
//   - baseCore: The original zapcore.Core (e.g., from existing zap.Logger)
//   - otelCore: The OTEL bridge core created by NewZapOTELCore
//   - opts: Additional zap options (caller, stacktrace, etc.)
//
// Example usage:
//
//	otelCore := telemetry.NewZapOTELCore(cfg)
//	logger := telemetry.NewBridgedLogger(existingLogger.Core(), otelCore,
//	    zap.AddCaller(),
//	    zap.AddStacktrace(zapcore.ErrorLevel),
//	)
func NewBridgedLogger(baseCore, otelCore zapcore.Core, opts ...zap.Option) *zap.Logger {
	// Combine cores using Tee - logs will be written to both
	combinedCore := zapcore.NewTee(baseCore, otelCore)
	return zap.New(combinedCore, opts...)
}

// =============================================================================
// Helper Functions
// =============================================================================

// CreateBridgedLoggerFromConfig creates a complete bridged logger setup.
// This is a convenience function that creates both the base logger and OTEL bridge.
//
// Parameters:
//   - baseConfig: Configuration for the base Zap logger (stdout/file output)
//   - logsProvider: The LoggerProvider for OTEL export
//   - serviceName: Service name for OTEL logger identification
//
// Returns a logger that outputs to both the configured output and OTEL Collector.
func CreateBridgedLoggerFromConfig(
	baseConfig *BaseLoggerConfig,
	logsProvider *LoggerProvider,
	serviceName string,
) (*zap.Logger, error) {
	// Create base core
	baseCore := createBaseCore(baseConfig)

	// Create OTEL core
	otelCore := NewZapOTELCore(ZapBridgeConfig{
		ServiceName:    serviceName,
		LoggerProvider: logsProvider,
		Level:          parseLogLevel(baseConfig.Level),
	})

	// Create combined logger
	opts := []zap.Option{
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	}

	return NewBridgedLogger(baseCore, otelCore, opts...), nil
}

// BaseLoggerConfig holds configuration for the base Zap logger.
type BaseLoggerConfig struct {
	Level      string // debug, info, warn, error
	Format     string // json, console
	Output     string // stdout, stderr, or file path
	TimeFormat string // time format string
}

// DefaultBaseLoggerConfig returns a default configuration for development.
func DefaultBaseLoggerConfig() *BaseLoggerConfig {
	return &BaseLoggerConfig{
		Level:      "info",
		Format:     "console",
		Output:     "stdout",
		TimeFormat: "2006-01-02T15:04:05.000Z07:00",
	}
}

// createBaseCore creates a zapcore.Core from BaseLoggerConfig.
func createBaseCore(cfg *BaseLoggerConfig) zapcore.Core {
	level := parseLogLevel(cfg.Level)
	encoder := createLogEncoder(cfg)
	writer := createLogWriter(cfg.Output)

	return zapcore.NewCore(encoder, writer, level)
}

// parseLogLevel converts a string level to zapcore.Level.
func parseLogLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// createLogEncoder creates the appropriate encoder based on format.
func createLogEncoder(cfg *BaseLoggerConfig) zapcore.Encoder {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout(cfg.TimeFormat),
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	if cfg.Format == "console" {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		return zapcore.NewConsoleEncoder(encoderConfig)
	}

	return zapcore.NewJSONEncoder(encoderConfig)
}

// createLogWriter creates the appropriate writer based on output.
func createLogWriter(output string) zapcore.WriteSyncer {
	switch output {
	case "stdout":
		return zapcore.AddSync(os.Stdout)
	case "stderr":
		return zapcore.AddSync(os.Stderr)
	default:
		// Unsupported output, fallback to stdout
		// TODO: Implement file output if needed
		return zapcore.AddSync(os.Stdout)
	}
}
