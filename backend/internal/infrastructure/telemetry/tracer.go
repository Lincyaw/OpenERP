// Package telemetry provides OpenTelemetry integration for distributed tracing.
package telemetry

import (
	"context"
	"fmt"
	"sync"
	"time"

	otelpyroscope "github.com/grafana/otel-profiling-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Config holds telemetry configuration.
type Config struct {
	Enabled           bool
	CollectorEndpoint string
	SamplingRatio     float64
	ServiceName       string
	Insecure          bool
}

// TracerProvider wraps the OpenTelemetry TracerProvider with lifecycle management.
type TracerProvider struct {
	provider            *sdktrace.TracerProvider
	logger              *zap.Logger
	config              Config
	mu                  sync.RWMutex
	spanProfilesEnabled bool
}

// NewTracerProvider creates and configures a new TracerProvider.
// If telemetry is disabled, it returns a no-op provider.
func NewTracerProvider(ctx context.Context, cfg Config, logger *zap.Logger) (*TracerProvider, error) {
	tp := &TracerProvider{
		logger: logger,
		config: cfg,
	}

	if !cfg.Enabled {
		logger.Info("Telemetry disabled, using no-op tracer provider")
		// Set global tracer provider to no-op (default)
		return tp, nil
	}

	// Create OTLP gRPC exporter
	exporterOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.CollectorEndpoint),
	}
	if cfg.Insecure {
		exporterOpts = append(exporterOpts, otlptracegrpc.WithInsecure())
	}
	exporter, err := otlptracegrpc.New(ctx, exporterOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
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

	// Create sampler based on sampling ratio
	var sampler sdktrace.Sampler
	switch cfg.SamplingRatio {
	case 1.0:
		sampler = sdktrace.AlwaysSample()
	case 0.0:
		sampler = sdktrace.NeverSample()
	default:
		sampler = sdktrace.TraceIDRatioBased(cfg.SamplingRatio)
	}

	// Create TracerProvider with batch span processor
	tp.provider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp.provider)

	// Set global text map propagator for context propagation
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logger.Info("OpenTelemetry TracerProvider initialized",
		zap.String("collector_endpoint", cfg.CollectorEndpoint),
		zap.Float64("sampling_ratio", cfg.SamplingRatio),
		zap.String("service_name", cfg.ServiceName),
	)

	return tp, nil
}

// EnableSpanProfiles wraps the TracerProvider with Pyroscope span profiles integration.
// This allows associating CPU profiles with individual trace spans.
//
// IMPORTANT: This must be called AFTER the Pyroscope profiler is started.
// The profiler must be running for span profiles to be collected.
//
// When enabled:
//   - Each span automatically gets a span_id label in pprof
//   - CPU profiles can be filtered by span_id in Pyroscope
//   - Only CPU profiling is supported for span profiles
//   - Spans shorter than 10ms may not have profile data (100Hz sampling rate)
//
// Requirements:
//   - Telemetry must be enabled (cfg.Enabled = true)
//   - Pyroscope server >= 0.14.0
//   - Pyroscope profiler must be started before calling this method
func (tp *TracerProvider) EnableSpanProfiles() error {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	if tp.provider == nil {
		tp.logger.Debug("Cannot enable span profiles: telemetry disabled")
		return nil
	}

	if tp.spanProfilesEnabled {
		tp.logger.Debug("Span profiles already enabled")
		return nil
	}

	// Wrap the underlying TracerProvider with otelpyroscope
	// This makes every span automatically include span_id as a pprof label
	wrappedProvider := otelpyroscope.NewTracerProvider(tp.provider)

	// Update the global tracer provider to use the wrapped provider
	otel.SetTracerProvider(wrappedProvider)

	tp.spanProfilesEnabled = true

	tp.logger.Info("Span profiles integration enabled",
		zap.String("service_name", tp.config.ServiceName),
		zap.String("note", "CPU profiles will be associated with trace spans"),
	)

	return nil
}

// IsSpanProfilesEnabled returns whether span profiles integration is enabled.
func (tp *TracerProvider) IsSpanProfilesEnabled() bool {
	tp.mu.RLock()
	defer tp.mu.RUnlock()
	return tp.spanProfilesEnabled
}

// Shutdown gracefully shuts down the tracer provider, flushing any pending spans.
// It should be called when the application exits.
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	if tp.provider == nil {
		tp.logger.Debug("No tracer provider to shutdown (telemetry disabled)")
		return nil
	}

	tp.logger.Info("Shutting down OpenTelemetry TracerProvider...")

	// Create a timeout context for shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := tp.provider.Shutdown(shutdownCtx); err != nil {
		tp.logger.Error("Error shutting down tracer provider", zap.Error(err))
		return fmt.Errorf("failed to shutdown tracer provider: %w", err)
	}

	tp.logger.Info("OpenTelemetry TracerProvider shutdown complete")
	return nil
}

// Tracer returns a named tracer from the provider.
func (tp *TracerProvider) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	if tp.provider == nil {
		return otel.GetTracerProvider().Tracer(name, opts...)
	}
	return tp.provider.Tracer(name, opts...)
}

// IsEnabled returns whether telemetry is enabled.
func (tp *TracerProvider) IsEnabled() bool {
	return tp.config.Enabled && tp.provider != nil
}

// GetConfig returns a copy of the telemetry configuration.
func (tp *TracerProvider) GetConfig() Config {
	return tp.config
}

// ForceFlush immediately exports all spans that have not yet been exported.
// This is useful in tests or when you need to ensure all spans are exported before shutdown.
func (tp *TracerProvider) ForceFlush(ctx context.Context) error {
	if tp.provider == nil {
		return nil
	}
	return tp.provider.ForceFlush(ctx)
}
