// Package telemetry provides OpenTelemetry integration for metrics collection.
package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.uber.org/zap"
)

// MetricsConfig holds metrics configuration.
type MetricsConfig struct {
	Enabled           bool
	CollectorEndpoint string
	ExportInterval    time.Duration // Default: 60s
	ServiceName       string
	Insecure          bool
}

// MeterProvider wraps the OpenTelemetry MeterProvider with lifecycle management.
type MeterProvider struct {
	provider *sdkmetric.MeterProvider
	logger   *zap.Logger
	config   MetricsConfig
}

// NewMeterProvider creates and configures a new MeterProvider.
// If metrics are disabled, it returns a provider that wraps the no-op global meter.
func NewMeterProvider(ctx context.Context, cfg MetricsConfig, logger *zap.Logger) (*MeterProvider, error) {
	mp := &MeterProvider{
		logger: logger,
		config: cfg,
	}

	if !cfg.Enabled {
		logger.Info("Metrics disabled, using no-op meter provider")
		return mp, nil
	}

	// Set default export interval if not specified
	exportInterval := cfg.ExportInterval
	if exportInterval == 0 {
		exportInterval = 60 * time.Second
	}

	// Create OTLP gRPC exporter for metrics
	exporterOpts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(cfg.CollectorEndpoint),
	}
	if cfg.Insecure {
		exporterOpts = append(exporterOpts, otlpmetricgrpc.WithInsecure())
	}

	exporter, err := otlpmetricgrpc.New(ctx, exporterOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP metrics exporter: %w", err)
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

	// Create MeterProvider with periodic reader
	mp.provider = sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(
				exporter,
				sdkmetric.WithInterval(exportInterval),
			),
		),
	)

	// Set global meter provider
	otel.SetMeterProvider(mp.provider)

	logger.Info("OpenTelemetry MeterProvider initialized",
		zap.String("collector_endpoint", cfg.CollectorEndpoint),
		zap.Duration("export_interval", exportInterval),
		zap.String("service_name", cfg.ServiceName),
	)

	return mp, nil
}

// Shutdown gracefully shuts down the meter provider, flushing any pending metrics.
// It should be called when the application exits.
func (mp *MeterProvider) Shutdown(ctx context.Context) error {
	if mp.provider == nil {
		mp.logger.Debug("No meter provider to shutdown (metrics disabled)")
		return nil
	}

	mp.logger.Info("Shutting down OpenTelemetry MeterProvider...")

	// Create a timeout context for shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := mp.provider.Shutdown(shutdownCtx); err != nil {
		mp.logger.Error("Error shutting down meter provider", zap.Error(err))
		return fmt.Errorf("failed to shutdown meter provider: %w", err)
	}

	mp.logger.Info("OpenTelemetry MeterProvider shutdown complete")
	return nil
}

// Meter returns a named meter from the provider.
func (mp *MeterProvider) Meter(name string, opts ...metric.MeterOption) metric.Meter {
	if mp.provider == nil {
		return otel.GetMeterProvider().Meter(name, opts...)
	}
	return mp.provider.Meter(name, opts...)
}

// IsEnabled returns whether metrics are enabled.
func (mp *MeterProvider) IsEnabled() bool {
	return mp.config.Enabled && mp.provider != nil
}

// GetConfig returns a copy of the metrics configuration.
func (mp *MeterProvider) GetConfig() MetricsConfig {
	return mp.config
}

// ForceFlush immediately exports all metrics that have not yet been exported.
// This is useful in tests or when you need to ensure all metrics are exported before shutdown.
func (mp *MeterProvider) ForceFlush(ctx context.Context) error {
	if mp.provider == nil {
		return nil
	}
	return mp.provider.ForceFlush(ctx)
}

// =============================================================================
// Base Metric Type Helpers
// =============================================================================

// Counter is a helper for creating and recording counter metrics.
// Counters represent monotonically increasing values (e.g., request count).
type Counter struct {
	counter metric.Int64Counter
}

// NewCounter creates a new Counter metric.
func NewCounter(meter metric.Meter, name, description, unit string) (*Counter, error) {
	c, err := meter.Int64Counter(
		name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create counter %s: %w", name, err)
	}
	return &Counter{counter: c}, nil
}

// Add increments the counter by the given value with optional attributes.
func (c *Counter) Add(ctx context.Context, value int64, attrs ...attribute.KeyValue) {
	c.counter.Add(ctx, value, metric.WithAttributes(attrs...))
}

// Inc increments the counter by 1 with optional attributes.
func (c *Counter) Inc(ctx context.Context, attrs ...attribute.KeyValue) {
	c.counter.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// Histogram is a helper for creating and recording histogram metrics.
// Histograms are used for distributions (e.g., request latency).
type Histogram struct {
	histogram metric.Float64Histogram
}

// HistogramOpts provides options for creating a histogram.
type HistogramOpts struct {
	Name        string
	Description string
	Unit        string
	Boundaries  []float64 // Custom bucket boundaries
}

// NewHistogram creates a new Histogram metric.
func NewHistogram(meter metric.Meter, opts HistogramOpts) (*Histogram, error) {
	histogramOpts := []metric.Float64HistogramOption{
		metric.WithDescription(opts.Description),
		metric.WithUnit(opts.Unit),
	}

	// Add custom boundaries if specified
	if len(opts.Boundaries) > 0 {
		histogramOpts = append(histogramOpts,
			metric.WithExplicitBucketBoundaries(opts.Boundaries...),
		)
	}

	h, err := meter.Float64Histogram(opts.Name, histogramOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create histogram %s: %w", opts.Name, err)
	}
	return &Histogram{histogram: h}, nil
}

// Record records a value to the histogram with optional attributes.
func (h *Histogram) Record(ctx context.Context, value float64, attrs ...attribute.KeyValue) {
	h.histogram.Record(ctx, value, metric.WithAttributes(attrs...))
}

// RecordDuration records a duration (in seconds) to the histogram.
func (h *Histogram) RecordDuration(ctx context.Context, d time.Duration, attrs ...attribute.KeyValue) {
	h.histogram.Record(ctx, d.Seconds(), metric.WithAttributes(attrs...))
}

// Gauge is a helper for creating and recording gauge metrics.
// Gauges represent point-in-time values (e.g., active connections).
type Gauge struct {
	gauge metric.Int64Gauge
}

// NewGauge creates a new Gauge metric.
func NewGauge(meter metric.Meter, name, description, unit string) (*Gauge, error) {
	g, err := meter.Int64Gauge(
		name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gauge %s: %w", name, err)
	}
	return &Gauge{gauge: g}, nil
}

// Record records the current value to the gauge with optional attributes.
func (g *Gauge) Record(ctx context.Context, value int64, attrs ...attribute.KeyValue) {
	g.gauge.Record(ctx, value, metric.WithAttributes(attrs...))
}

// FloatGauge is a helper for creating and recording float64 gauge metrics.
type FloatGauge struct {
	gauge metric.Float64Gauge
}

// NewFloatGauge creates a new Float64 Gauge metric.
func NewFloatGauge(meter metric.Meter, name, description, unit string) (*FloatGauge, error) {
	g, err := meter.Float64Gauge(
		name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create float gauge %s: %w", name, err)
	}
	return &FloatGauge{gauge: g}, nil
}

// Record records the current value to the gauge with optional attributes.
func (g *FloatGauge) Record(ctx context.Context, value float64, attrs ...attribute.KeyValue) {
	g.gauge.Record(ctx, value, metric.WithAttributes(attrs...))
}

// =============================================================================
// Common Metric Labels/Attributes
// =============================================================================

// Common attribute keys for consistency across metrics.
var (
	// Service-level attributes
	AttrTenantID = attribute.Key("tenant_id")
	AttrUserID   = attribute.Key("user_id")

	// HTTP attributes
	AttrHTTPMethod     = attribute.Key("http.method")
	AttrHTTPStatusCode = attribute.Key("http.status_code")
	AttrHTTPRoute      = attribute.Key("http.route")

	// Database attributes
	AttrDBOperation = attribute.Key("db.operation")
	AttrDBTable     = attribute.Key("db.table")
	AttrDBState     = attribute.Key("db.pool.state")

	// Business attributes
	AttrOrderType     = attribute.Key("order_type")
	AttrPaymentMethod = attribute.Key("payment_method")
	AttrPaymentStatus = attribute.Key("payment_status")
	AttrWarehouseID   = attribute.Key("warehouse_id")
	AttrProductID     = attribute.Key("product_id")
	AttrCategoryID    = attribute.Key("category_id")
)

// =============================================================================
// Default Histogram Buckets
// =============================================================================

// Common histogram bucket boundaries for different use cases.
var (
	// HTTPDurationBuckets are bucket boundaries for HTTP request duration (seconds).
	HTTPDurationBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

	// DBDurationBuckets are bucket boundaries for database query duration (seconds).
	DBDurationBuckets = []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5}

	// SmallDurationBuckets are bucket boundaries for fast operations (seconds).
	SmallDurationBuckets = []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1}
)
