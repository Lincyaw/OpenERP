// Package middleware provides HTTP middleware for the ERP system.
package middleware

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/erp/backend/internal/domain/billing"
	"github.com/erp/backend/internal/infrastructure/telemetry"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

// UsageTrackerConfig holds configuration for usage tracking middleware.
type UsageTrackerConfig struct {
	// Enabled controls whether usage tracking is active.
	Enabled bool
	// BufferSize is the size of the async write buffer.
	BufferSize int
	// BatchSize is the number of records to batch before writing.
	BatchSize int
	// FlushInterval is the maximum time to wait before flushing the buffer.
	FlushInterval time.Duration
	// MeterProvider is the OpenTelemetry meter provider for metrics.
	MeterProvider *telemetry.MeterProvider
	// Logger for middleware logging.
	Logger *zap.Logger
	// SkipPaths are paths that should not be tracked.
	SkipPaths []string
}

// DefaultUsageTrackerConfig returns default usage tracker configuration.
func DefaultUsageTrackerConfig() UsageTrackerConfig {
	return UsageTrackerConfig{
		Enabled:       true,
		BufferSize:    10000,
		BatchSize:     100,
		FlushInterval: 5 * time.Second,
		SkipPaths: []string{
			"/health",
			"/healthz",
			"/ready",
			"/metrics",
			"/api/v1/health",
			"/swagger",
		},
	}
}

// UsageTracker manages async usage record collection and batch writing.
type UsageTracker struct {
	config     UsageTrackerConfig
	repository billing.UsageRecordRepository
	buffer     chan *billing.UsageRecord
	logger     *zap.Logger
	metrics    *usageMetrics
	wg         sync.WaitGroup
	stopCh     chan struct{}
	mu         sync.RWMutex
	running    bool
}

// usageMetrics holds OpenTelemetry metrics for usage tracking.
type usageMetrics struct {
	apiCallsTotal     *telemetry.Counter
	resourceCreations *telemetry.Counter
	bufferSize        metric.Int64Gauge
	batchWriteLatency *telemetry.Histogram
	batchWriteErrors  *telemetry.Counter
	recordsWritten    *telemetry.Counter
	recordsDropped    *telemetry.Counter
	trackingOverhead  *telemetry.Histogram
}

// newUsageMetrics creates OpenTelemetry metrics for usage tracking.
func newUsageMetrics(meter metric.Meter) (*usageMetrics, error) {
	apiCallsTotal, err := telemetry.NewCounter(
		meter,
		"usage_api_calls_total",
		"Total number of API calls tracked",
		"{call}",
	)
	if err != nil {
		return nil, err
	}

	resourceCreations, err := telemetry.NewCounter(
		meter,
		"usage_resource_creations_total",
		"Total number of resource creations tracked",
		"{creation}",
	)
	if err != nil {
		return nil, err
	}

	bufferSize, err := meter.Int64Gauge(
		"usage_buffer_size",
		metric.WithDescription("Current size of the usage record buffer"),
		metric.WithUnit("{record}"),
	)
	if err != nil {
		return nil, err
	}

	batchWriteLatency, err := telemetry.NewHistogram(meter, telemetry.HistogramOpts{
		Name:        "usage_batch_write_duration_seconds",
		Description: "Latency of batch writes to usage_records table",
		Unit:        "s",
		Boundaries:  telemetry.DBDurationBuckets,
	})
	if err != nil {
		return nil, err
	}

	batchWriteErrors, err := telemetry.NewCounter(
		meter,
		"usage_batch_write_errors_total",
		"Total number of batch write errors",
		"{error}",
	)
	if err != nil {
		return nil, err
	}

	recordsWritten, err := telemetry.NewCounter(
		meter,
		"usage_records_written_total",
		"Total number of usage records written to database",
		"{record}",
	)
	if err != nil {
		return nil, err
	}

	recordsDropped, err := telemetry.NewCounter(
		meter,
		"usage_records_dropped_total",
		"Total number of usage records dropped due to buffer overflow",
		"{record}",
	)
	if err != nil {
		return nil, err
	}

	trackingOverhead, err := telemetry.NewHistogram(meter, telemetry.HistogramOpts{
		Name:        "usage_tracking_overhead_seconds",
		Description: "Overhead added to request processing by usage tracking",
		Unit:        "s",
		Boundaries:  telemetry.SmallDurationBuckets,
	})
	if err != nil {
		return nil, err
	}

	return &usageMetrics{
		apiCallsTotal:     apiCallsTotal,
		resourceCreations: resourceCreations,
		bufferSize:        bufferSize,
		batchWriteLatency: batchWriteLatency,
		batchWriteErrors:  batchWriteErrors,
		recordsWritten:    recordsWritten,
		recordsDropped:    recordsDropped,
		trackingOverhead:  trackingOverhead,
	}, nil
}

// NewUsageTracker creates a new usage tracker with the given configuration.
func NewUsageTracker(cfg UsageTrackerConfig, repo billing.UsageRecordRepository) (*UsageTracker, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	tracker := &UsageTracker{
		config:     cfg,
		repository: repo,
		buffer:     make(chan *billing.UsageRecord, cfg.BufferSize),
		logger:     logger,
		stopCh:     make(chan struct{}),
	}

	// Initialize metrics if meter provider is available
	if cfg.MeterProvider != nil && cfg.MeterProvider.IsEnabled() {
		meter := cfg.MeterProvider.Meter("usage.tracker")
		metrics, err := newUsageMetrics(meter)
		if err != nil {
			logger.Warn("Failed to create usage metrics, continuing without metrics", zap.Error(err))
		} else {
			tracker.metrics = metrics
		}
	}

	return tracker, nil
}

// Start begins the background batch writer goroutine.
func (t *UsageTracker) Start() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		return
	}

	t.running = true
	t.wg.Add(1)
	go t.batchWriter()

	t.logger.Info("Usage tracker started",
		zap.Int("buffer_size", t.config.BufferSize),
		zap.Int("batch_size", t.config.BatchSize),
		zap.Duration("flush_interval", t.config.FlushInterval),
	)
}

// Stop gracefully stops the usage tracker, flushing remaining records.
func (t *UsageTracker) Stop(ctx context.Context) error {
	t.mu.Lock()
	if !t.running {
		t.mu.Unlock()
		return nil
	}
	t.running = false
	t.mu.Unlock()

	t.logger.Info("Stopping usage tracker...")

	// Signal the batch writer to stop
	close(t.stopCh)

	// Wait for the batch writer to finish with timeout
	done := make(chan struct{})
	go func() {
		t.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.logger.Info("Usage tracker stopped gracefully")
		return nil
	case <-ctx.Done():
		t.logger.Warn("Usage tracker stop timed out")
		return ctx.Err()
	}
}

// batchWriter is the background goroutine that batches and writes usage records.
func (t *UsageTracker) batchWriter() {
	defer t.wg.Done()

	batch := make([]*billing.UsageRecord, 0, t.config.BatchSize)
	ticker := time.NewTicker(t.config.FlushInterval)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		start := time.Now()
		err := t.repository.SaveBatch(ctx, batch)
		duration := time.Since(start)

		if t.metrics != nil {
			t.metrics.batchWriteLatency.RecordDuration(ctx, duration)
		}

		if err != nil {
			t.logger.Error("Failed to write usage batch",
				zap.Int("batch_size", len(batch)),
				zap.Error(err),
			)
			if t.metrics != nil {
				t.metrics.batchWriteErrors.Inc(ctx)
			}
		} else {
			t.logger.Debug("Wrote usage batch",
				zap.Int("batch_size", len(batch)),
				zap.Duration("duration", duration),
			)
			if t.metrics != nil {
				t.metrics.recordsWritten.Add(ctx, int64(len(batch)))
			}
		}

		// Clear the batch
		batch = batch[:0]
	}

	for {
		select {
		case record, ok := <-t.buffer:
			if !ok {
				// Channel closed, flush remaining and exit
				flush()
				return
			}

			batch = append(batch, record)

			// Flush if batch is full
			if len(batch) >= t.config.BatchSize {
				flush()
			}

		case <-ticker.C:
			// Periodic flush
			flush()

			// Update buffer size metric
			if t.metrics != nil {
				t.metrics.bufferSize.Record(context.Background(), int64(len(t.buffer)))
			}

		case <-t.stopCh:
			// Drain remaining records from buffer
			for {
				select {
				case record := <-t.buffer:
					batch = append(batch, record)
					if len(batch) >= t.config.BatchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}
		}
	}
}

// Track adds a usage record to the buffer for async writing.
// Returns true if the record was added, false if the buffer is full.
func (t *UsageTracker) Track(record *billing.UsageRecord) bool {
	t.mu.RLock()
	running := t.running
	t.mu.RUnlock()

	if !running || !t.config.Enabled {
		return false
	}

	select {
	case t.buffer <- record:
		return true
	default:
		// Buffer is full, drop the record
		if t.metrics != nil {
			t.metrics.recordsDropped.Inc(context.Background())
		}
		t.logger.Warn("Usage buffer full, dropping record",
			zap.String("usage_type", string(record.UsageType)),
			zap.String("tenant_id", record.TenantID.String()),
		)
		return false
	}
}

// TrackAPIUsage returns a Gin middleware that tracks API call usage.
// This middleware should be placed after authentication middleware
// to have access to tenant and user information.
func TrackAPIUsage(tracker *UsageTracker) gin.HandlerFunc {
	if tracker == nil || !tracker.config.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		// Check if path should be skipped
		path := c.Request.URL.Path
		for _, skipPath := range tracker.config.SkipPaths {
			if path == skipPath || strings.HasPrefix(path, skipPath+"/") {
				c.Next()
				return
			}
		}

		start := time.Now()

		// Process request first
		c.Next()

		// Track usage asynchronously after request completes
		trackingStart := time.Now()

		// Get tenant ID from context
		tenantIDStr := GetTenantID(c)
		if tenantIDStr == "" {
			// No tenant context, skip tracking
			return
		}

		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			return
		}

		// Create usage record
		record, err := billing.NewUsageRecordSimple(tenantID, billing.UsageTypeAPICalls, 1)
		if err != nil {
			tracker.logger.Debug("Failed to create API usage record", zap.Error(err))
			return
		}

		// Add request context
		route := getRoutePattern(c)
		record.WithSource("api_request", route)
		record.WithRequestInfo(c.ClientIP(), c.Request.UserAgent())
		record.WithMetadata("method", c.Request.Method)
		record.WithMetadata("status_code", c.Writer.Status())
		record.WithMetadata("response_time_ms", time.Since(start).Milliseconds())

		// Get user ID if available
		if userIDStr, exists := c.Get(JWTUserIDKey); exists {
			if uid, ok := userIDStr.(string); ok {
				if userID, err := uuid.Parse(uid); err == nil {
					record.WithUser(userID)
				}
			}
		}

		// Track the record
		tracker.Track(record)

		// Record tracking overhead
		if tracker.metrics != nil {
			overhead := time.Since(trackingStart)
			tracker.metrics.trackingOverhead.RecordDuration(c.Request.Context(), overhead)
			tracker.metrics.apiCallsTotal.Inc(c.Request.Context(),
				telemetry.AttrHTTPMethod.String(c.Request.Method),
				telemetry.AttrHTTPRoute.String(route),
				telemetry.AttrTenantID.String(tenantIDStr),
			)
		}
	}
}

// ResourceType represents the type of resource being created.
type ResourceType string

const (
	ResourceTypeSalesOrder    ResourceType = "sales_order"
	ResourceTypePurchaseOrder ResourceType = "purchase_order"
	ResourceTypeProduct       ResourceType = "product"
	ResourceTypeCustomer      ResourceType = "customer"
	ResourceTypeSupplier      ResourceType = "supplier"
	ResourceTypeWarehouse     ResourceType = "warehouse"
	ResourceTypeReport        ResourceType = "report"
	ResourceTypeDataExport    ResourceType = "data_export"
	ResourceTypeDataImport    ResourceType = "data_import"
)

// ToUsageType converts a ResourceType to the corresponding UsageType.
func (r ResourceType) ToUsageType() billing.UsageType {
	switch r {
	case ResourceTypeSalesOrder, ResourceTypePurchaseOrder:
		return billing.UsageTypeOrdersCreated
	case ResourceTypeProduct:
		return billing.UsageTypeProductsSKU
	case ResourceTypeCustomer:
		return billing.UsageTypeCustomers
	case ResourceTypeSupplier:
		return billing.UsageTypeSuppliers
	case ResourceTypeWarehouse:
		return billing.UsageTypeWarehouses
	case ResourceTypeReport:
		return billing.UsageTypeReportsGenerated
	case ResourceTypeDataExport:
		return billing.UsageTypeDataExports
	case ResourceTypeDataImport:
		return billing.UsageTypeDataImportRows
	default:
		return billing.UsageTypeAPICalls
	}
}

// ResourceCreationContext holds context for resource creation tracking.
type ResourceCreationContext struct {
	ResourceType ResourceType
	ResourceID   string
	Quantity     int64 // For imports, this is the number of rows
	Metadata     map[string]any
}

// ResourceCreationContextKey is the key for storing resource creation context.
const ResourceCreationContextKey = "resource_creation_context"

// TrackResourceCreation returns a Gin middleware that tracks resource creation.
// This middleware should be used on specific routes that create resources.
// The handler should set the ResourceCreationContext in the gin context
// after successfully creating the resource.
func TrackResourceCreation(tracker *UsageTracker, resourceType ResourceType) gin.HandlerFunc {
	if tracker == nil || !tracker.config.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		// Process request first
		c.Next()

		// Only track successful creations (2xx status codes)
		if c.Writer.Status() < 200 || c.Writer.Status() >= 300 {
			return
		}

		// Get tenant ID from context
		tenantIDStr := GetTenantID(c)
		if tenantIDStr == "" {
			return
		}

		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			return
		}

		// Get resource creation context if set by handler
		var resourceID string
		var quantity int64 = 1
		var metadata map[string]any

		if ctx, exists := c.Get(ResourceCreationContextKey); exists {
			if rcc, ok := ctx.(*ResourceCreationContext); ok {
				resourceID = rcc.ResourceID
				if rcc.Quantity > 0 {
					quantity = rcc.Quantity
				}
				metadata = rcc.Metadata
			}
		}

		// Create usage record
		usageType := resourceType.ToUsageType()
		record, err := billing.NewUsageRecordSimple(tenantID, usageType, quantity)
		if err != nil {
			tracker.logger.Debug("Failed to create resource usage record", zap.Error(err))
			return
		}

		// Add source information
		record.WithSource(string(resourceType), resourceID)

		// Add metadata
		record.WithMetadata("resource_type", string(resourceType))
		for k, v := range metadata {
			record.WithMetadata(k, v)
		}

		// Get user ID if available
		if userIDStr, exists := c.Get(JWTUserIDKey); exists {
			if uid, ok := userIDStr.(string); ok {
				if userID, err := uuid.Parse(uid); err == nil {
					record.WithUser(userID)
				}
			}
		}

		// Track the record
		tracker.Track(record)

		// Record metrics
		if tracker.metrics != nil {
			tracker.metrics.resourceCreations.Inc(c.Request.Context(),
				attribute.String("resource_type", string(resourceType)),
				telemetry.AttrTenantID.String(tenantIDStr),
			)
		}
	}
}

// SetResourceCreationContext is a helper function for handlers to set
// the resource creation context for tracking.
func SetResourceCreationContext(c *gin.Context, ctx *ResourceCreationContext) {
	c.Set(ResourceCreationContextKey, ctx)
}

// TrackDataImport is a helper middleware for tracking data import operations.
// It expects the handler to set the row count in the ResourceCreationContext.
func TrackDataImport(tracker *UsageTracker) gin.HandlerFunc {
	return TrackResourceCreation(tracker, ResourceTypeDataImport)
}

// TrackDataExport is a helper middleware for tracking data export operations.
func TrackDataExport(tracker *UsageTracker) gin.HandlerFunc {
	return TrackResourceCreation(tracker, ResourceTypeDataExport)
}

// TrackReportGeneration is a helper middleware for tracking report generation.
func TrackReportGeneration(tracker *UsageTracker) gin.HandlerFunc {
	return TrackResourceCreation(tracker, ResourceTypeReport)
}

// UsageTrackerStats returns current statistics about the usage tracker.
type UsageTrackerStats struct {
	BufferSize     int
	BufferCapacity int
	BufferUsage    float64
	Running        bool
}

// Stats returns current statistics about the usage tracker.
func (t *UsageTracker) Stats() UsageTrackerStats {
	t.mu.RLock()
	running := t.running
	t.mu.RUnlock()

	bufferLen := len(t.buffer)
	bufferCap := cap(t.buffer)

	var usage float64
	if bufferCap > 0 {
		usage = float64(bufferLen) / float64(bufferCap) * 100
	}

	return UsageTrackerStats{
		BufferSize:     bufferLen,
		BufferCapacity: bufferCap,
		BufferUsage:    usage,
		Running:        running,
	}
}

// IsRunning returns whether the usage tracker is currently running.
func (t *UsageTracker) IsRunning() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.running
}
