// Package metrics provides metrics collection and reporting for the load generator.
package metrics

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
)

// Prometheus metric names.
const (
	MetricRequestsTotal           = "loadgen_requests_total"
	MetricRequestDurationSeconds  = "loadgen_request_duration_seconds"
	MetricCurrentQPS              = "loadgen_current_qps"
	MetricTargetQPS               = "loadgen_target_qps"
	MetricPoolSize                = "loadgen_pool_size"
	MetricBackpressureState       = "loadgen_backpressure_state"
	MetricSuccessRate             = "loadgen_success_rate"
	MetricActiveWorkers           = "loadgen_active_workers"
	MetricRequestBytesTotal       = "loadgen_request_bytes_total"
	MetricEndpointRequestsTotal   = "loadgen_endpoint_requests_total"
	MetricEndpointDurationSeconds = "loadgen_endpoint_duration_seconds"
)

// PrometheusExporter exports metrics to Prometheus via an HTTP endpoint.
// It provides real-time metrics that can be scraped by Prometheus.
//
// Thread Safety: Safe for concurrent use by multiple goroutines.
type PrometheusExporter struct {
	mu sync.RWMutex

	// Configuration
	config PrometheusExporterConfig

	// Prometheus registry
	registry *prometheus.Registry

	// Metrics
	requestsTotal          *prometheus.CounterVec
	requestDurationSeconds *prometheus.HistogramVec
	currentQPS             prometheus.Gauge
	targetQPS              prometheus.Gauge
	poolSize               prometheus.Gauge
	backpressureState      prometheus.Gauge
	successRate            prometheus.Gauge
	activeWorkers          prometheus.Gauge
	requestBytesTotal      prometheus.Counter

	// HTTP server
	server *http.Server
	ln     net.Listener

	// State tracking
	running bool

	// Error handling
	lastError error
}

// PrometheusExporterConfig holds configuration for the Prometheus exporter.
type PrometheusExporterConfig struct {
	// Port is the HTTP port for the metrics endpoint.
	// Default: 9090
	Port int

	// Path is the URL path for the metrics endpoint.
	// Default: /metrics
	Path string

	// Namespace is the prefix for all metrics.
	// Default: "" (no prefix beyond "loadgen_")
	Namespace string

	// Subsystem is the subsystem label.
	// Default: "" (no subsystem)
	Subsystem string

	// HistogramBuckets are the histogram buckets for request duration.
	// Default: prometheus.DefBuckets
	HistogramBuckets []float64
}

// DefaultPrometheusExporterConfig returns default configuration.
func DefaultPrometheusExporterConfig() PrometheusExporterConfig {
	return PrometheusExporterConfig{
		Port:             9090,
		Path:             "/metrics",
		HistogramBuckets: prometheus.DefBuckets,
	}
}

// NewPrometheusExporter creates a new Prometheus exporter.
func NewPrometheusExporter(config PrometheusExporterConfig) *PrometheusExporter {
	if config.Port == 0 {
		config.Port = 9090
	}
	if config.Path == "" {
		config.Path = "/metrics"
	}
	if len(config.HistogramBuckets) == 0 {
		config.HistogramBuckets = prometheus.DefBuckets
	}

	// Create a new registry to avoid conflicts with default metrics
	registry := prometheus.NewRegistry()

	exporter := &PrometheusExporter{
		config:   config,
		registry: registry,
	}

	// Initialize metrics
	exporter.initMetrics()

	return exporter
}

// initMetrics initializes all Prometheus metrics.
func (e *PrometheusExporter) initMetrics() {
	// Counter for total requests
	e.requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: e.config.Namespace,
			Subsystem: e.config.Subsystem,
			Name:      "requests_total",
			Help:      "Total number of HTTP requests made by the load generator.",
		},
		[]string{"status", "success"},
	)

	// Histogram for request duration
	e.requestDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: e.config.Namespace,
			Subsystem: e.config.Subsystem,
			Name:      "request_duration_seconds",
			Help:      "Duration of HTTP requests in seconds.",
			Buckets:   e.config.HistogramBuckets,
		},
		[]string{"endpoint"},
	)

	// Gauge for current QPS
	e.currentQPS = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: e.config.Namespace,
			Subsystem: e.config.Subsystem,
			Name:      "current_qps",
			Help:      "Current requests per second rate.",
		},
	)

	// Gauge for target QPS
	e.targetQPS = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: e.config.Namespace,
			Subsystem: e.config.Subsystem,
			Name:      "target_qps",
			Help:      "Target requests per second rate.",
		},
	)

	// Gauge for pool size
	e.poolSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: e.config.Namespace,
			Subsystem: e.config.Subsystem,
			Name:      "pool_size",
			Help:      "Current worker pool size.",
		},
	)

	// Gauge for backpressure state
	e.backpressureState = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: e.config.Namespace,
			Subsystem: e.config.Subsystem,
			Name:      "backpressure_state",
			Help:      "Current backpressure state (0=normal, 1=warning, 2=critical).",
		},
	)

	// Gauge for success rate
	e.successRate = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: e.config.Namespace,
			Subsystem: e.config.Subsystem,
			Name:      "success_rate",
			Help:      "Current request success rate (0.0-100.0).",
		},
	)

	// Gauge for active workers
	e.activeWorkers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: e.config.Namespace,
			Subsystem: e.config.Subsystem,
			Name:      "active_workers",
			Help:      "Number of currently active workers.",
		},
	)

	// Counter for total bytes
	e.requestBytesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: e.config.Namespace,
			Subsystem: e.config.Subsystem,
			Name:      "request_bytes_total",
			Help:      "Total bytes received from all requests.",
		},
	)

	// Register all metrics with the registry
	e.registry.MustRegister(
		e.requestsTotal,
		e.requestDurationSeconds,
		e.currentQPS,
		e.targetQPS,
		e.poolSize,
		e.backpressureState,
		e.successRate,
		e.activeWorkers,
		e.requestBytesTotal,
	)
}

// Start starts the HTTP server for the metrics endpoint.
func (e *PrometheusExporter) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return nil
	}

	// Create listener
	addr := fmt.Sprintf(":%d", e.config.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("starting Prometheus exporter: %w", err)
	}
	e.ln = ln

	// Create handler
	mux := http.NewServeMux()
	mux.Handle(e.config.Path, promhttp.HandlerFor(e.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}))

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	e.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		if err := e.server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			// Store error for retrieval via LastError()
			e.mu.Lock()
			e.lastError = err
			e.mu.Unlock()
		}
	}()

	e.running = true
	return nil
}

// Stop stops the HTTP server.
func (e *PrometheusExporter) Stop(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return nil
	}

	e.running = false

	if e.server != nil {
		return e.server.Shutdown(ctx)
	}
	return nil
}

// RecordRequest records a single request result.
func (e *PrometheusExporter) RecordRequest(result Result) {
	// Record request count
	statusLabel := fmt.Sprintf("%d", result.StatusCode)
	successLabel := "true"
	if !result.Success {
		successLabel = "false"
	}
	e.requestsTotal.WithLabelValues(statusLabel, successLabel).Inc()

	// Record request duration
	durationSeconds := result.Latency.Seconds()
	e.requestDurationSeconds.WithLabelValues(result.EndpointName).Observe(durationSeconds)

	// Record bytes
	e.requestBytesTotal.Add(float64(result.ResponseSize))
}

// UpdateCurrentQPS updates the current QPS gauge.
func (e *PrometheusExporter) UpdateCurrentQPS(qps float64) {
	e.currentQPS.Set(qps)
}

// UpdateTargetQPS updates the target QPS gauge.
func (e *PrometheusExporter) UpdateTargetQPS(qps float64) {
	e.targetQPS.Set(qps)
}

// UpdatePoolSize updates the pool size gauge.
func (e *PrometheusExporter) UpdatePoolSize(size int) {
	e.poolSize.Set(float64(size))
}

// UpdateBackpressureState updates the backpressure state gauge.
// State values: 0=normal, 1=warning, 2=critical
func (e *PrometheusExporter) UpdateBackpressureState(state int) {
	e.backpressureState.Set(float64(state))
}

// UpdateSuccessRate updates the success rate gauge.
func (e *PrometheusExporter) UpdateSuccessRate(rate float64) {
	e.successRate.Set(rate)
}

// UpdateActiveWorkers updates the active workers gauge.
func (e *PrometheusExporter) UpdateActiveWorkers(count int) {
	e.activeWorkers.Set(float64(count))
}

// UpdateFromSnapshot updates all metrics from a collector snapshot.
func (e *PrometheusExporter) UpdateFromSnapshot(snapshot Snapshot) {
	e.UpdateCurrentQPS(snapshot.QPS)
	e.UpdateSuccessRate(snapshot.SuccessRate)
}

// GetPort returns the configured port.
func (e *PrometheusExporter) GetPort() int {
	return e.config.Port
}

// GetPath returns the configured path.
func (e *PrometheusExporter) GetPath() string {
	return e.config.Path
}

// GetAddress returns the full address for the metrics endpoint.
func (e *PrometheusExporter) GetAddress() string {
	return fmt.Sprintf("http://localhost:%d%s", e.config.Port, e.config.Path)
}

// IsRunning returns whether the exporter is running.
func (e *PrometheusExporter) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.running
}

// LastError returns the last error from the HTTP server, if any.
// This is useful for checking if the server encountered any issues after starting.
func (e *PrometheusExporter) LastError() error {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.lastError
}

// Registry returns the Prometheus registry (for testing).
func (e *PrometheusExporter) Registry() *prometheus.Registry {
	return e.registry
}

// Gather collects all metrics from the registry (for testing).
// Returns metric families for inspection.
func (e *PrometheusExporter) Gather() ([]*dto.MetricFamily, error) {
	return e.registry.Gather()
}
