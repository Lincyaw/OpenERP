// Package telemetry provides OpenTelemetry integration for database metrics collection.
package telemetry

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// DBMetricsConfig holds configuration for database metrics collection.
type DBMetricsConfig struct {
	// Enabled controls whether metrics collection is active.
	Enabled bool
	// SlowQueryThreshold defines the threshold for slow query detection (default: 200ms).
	SlowQueryThreshold time.Duration
	// PoolStatsInterval defines how often to collect connection pool stats (default: 15s).
	PoolStatsInterval time.Duration
}

// DefaultDBMetricsConfig returns default configuration for database metrics.
func DefaultDBMetricsConfig() DBMetricsConfig {
	return DBMetricsConfig{
		Enabled:            true,
		SlowQueryThreshold: 200 * time.Millisecond,
		PoolStatsInterval:  15 * time.Second,
	}
}

// DBMetrics holds all database-related metrics instruments.
type DBMetrics struct {
	// Connection pool metrics
	poolConnections    *Gauge // db_pool_connections with state label
	poolConnectionsMax *Gauge // db_pool_connections_max

	// Query metrics
	queryTotal     *Counter   // db_query_total
	queryDuration  *Histogram // db_query_duration_seconds
	slowQueryTotal *Counter   // db_slow_query_total

	// Internal state
	config   DBMetricsConfig
	logger   *zap.Logger
	sqlDB    *sql.DB
	stopCh   chan struct{}
	wg       sync.WaitGroup
	mu       sync.RWMutex
	stopOnce sync.Once // Ensures Stop() is idempotent
}

// NewDBMetrics creates a new DBMetrics instance with the given meter.
func NewDBMetrics(meter metric.Meter, cfg DBMetricsConfig, logger *zap.Logger) (*DBMetrics, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	// Apply defaults
	if cfg.SlowQueryThreshold == 0 {
		cfg.SlowQueryThreshold = 200 * time.Millisecond
	}
	if cfg.PoolStatsInterval == 0 {
		cfg.PoolStatsInterval = 15 * time.Second
	}

	// Create connection pool metrics
	poolConnections, err := NewGauge(
		meter,
		"db_pool_connections",
		"Number of connections in the pool by state",
		"{connection}",
	)
	if err != nil {
		return nil, err
	}

	poolConnectionsMax, err := NewGauge(
		meter,
		"db_pool_connections_max",
		"Maximum number of connections in the pool",
		"{connection}",
	)
	if err != nil {
		return nil, err
	}

	// Create query metrics
	queryTotal, err := NewCounter(
		meter,
		"db_query_total",
		"Total number of database queries by operation type",
		"{query}",
	)
	if err != nil {
		return nil, err
	}

	queryDuration, err := NewHistogram(meter, HistogramOpts{
		Name:        "db_query_duration_seconds",
		Description: "Database query latency distribution in seconds",
		Unit:        "s",
		Boundaries:  DBDurationBuckets, // [0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5]
	})
	if err != nil {
		return nil, err
	}

	slowQueryTotal, err := NewCounter(
		meter,
		"db_slow_query_total",
		"Total number of slow database queries (>200ms by default)",
		"{query}",
	)
	if err != nil {
		return nil, err
	}

	return &DBMetrics{
		poolConnections:    poolConnections,
		poolConnectionsMax: poolConnectionsMax,
		queryTotal:         queryTotal,
		queryDuration:      queryDuration,
		slowQueryTotal:     slowQueryTotal,
		config:             cfg,
		logger:             logger,
		stopCh:             make(chan struct{}),
	}, nil
}

// SetSQLDB sets the sql.DB instance for connection pool metrics collection.
// This must be called before StartPoolStatsCollection.
func (m *DBMetrics) SetSQLDB(sqlDB *sql.DB) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sqlDB = sqlDB
}

// StartPoolStatsCollection starts a goroutine that periodically collects
// connection pool statistics. Call Stop() to terminate.
func (m *DBMetrics) StartPoolStatsCollection(ctx context.Context) {
	m.mu.RLock()
	sqlDB := m.sqlDB
	m.mu.RUnlock()

	if sqlDB == nil {
		m.logger.Warn("Cannot start pool stats collection: sqlDB not set")
		return
	}

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()

		ticker := time.NewTicker(m.config.PoolStatsInterval)
		defer ticker.Stop()

		// Collect immediately on start
		m.collectPoolStats(ctx)

		for {
			select {
			case <-ticker.C:
				m.collectPoolStats(ctx)
			case <-m.stopCh:
				m.logger.Debug("Stopping pool stats collection")
				return
			case <-ctx.Done():
				m.logger.Debug("Pool stats collection context cancelled")
				return
			}
		}
	}()

	m.logger.Info("Started database connection pool stats collection",
		zap.Duration("interval", m.config.PoolStatsInterval),
	)
}

// collectPoolStats collects and records connection pool statistics.
func (m *DBMetrics) collectPoolStats(ctx context.Context) {
	m.mu.RLock()
	sqlDB := m.sqlDB
	m.mu.RUnlock()

	if sqlDB == nil {
		return
	}

	stats := sqlDB.Stats()

	// Record max connections
	m.poolConnectionsMax.Record(ctx, int64(stats.MaxOpenConnections))

	// Record connection states with labels
	// Note: sql.DB.Stats() provides:
	// - Idle: number of idle connections
	// - InUse: number of connections currently in use
	// - OpenConnections = Idle + InUse (total open connections)
	// WaitCount is cumulative and not a current state, so we don't use it here
	m.poolConnections.Record(ctx, int64(stats.Idle), AttrDBState.String("idle"))
	m.poolConnections.Record(ctx, int64(stats.InUse), AttrDBState.String("in_use"))
	// Record total open connections for overall pool utilization monitoring
	m.poolConnections.Record(ctx, int64(stats.OpenConnections), AttrDBState.String("open"))
}

// Stop stops the pool stats collection goroutine. Safe to call multiple times.
func (m *DBMetrics) Stop() {
	m.stopOnce.Do(func() {
		close(m.stopCh)
		m.wg.Wait()
		m.logger.Debug("Database metrics stopped")
	})
}

// RecordQuery records metrics for a database query.
func (m *DBMetrics) RecordQuery(ctx context.Context, operation string, table string, duration time.Duration, err error) {
	// Normalize operation to uppercase
	operation = strings.ToUpper(operation)
	if operation == "" {
		operation = "UNKNOWN"
	}

	// Record query count
	m.queryTotal.Inc(ctx, AttrDBOperation.String(operation))

	// Record query duration
	m.queryDuration.RecordDuration(ctx, duration, AttrDBOperation.String(operation))

	// Check for slow query
	if duration > m.config.SlowQueryThreshold {
		tableName := table
		if tableName == "" {
			tableName = "unknown"
		}
		m.slowQueryTotal.Inc(ctx, AttrDBTable.String(tableName))
	}
}

// =============================================================================
// GORM Plugin for Query Metrics
// =============================================================================

// DBMetricsPlugin is a GORM plugin that collects query metrics.
type DBMetricsPlugin struct {
	metrics *DBMetrics
	logger  *zap.Logger
}

// NewDBMetricsPlugin creates a new GORM plugin for database metrics.
func NewDBMetricsPlugin(metrics *DBMetrics, logger *zap.Logger) *DBMetricsPlugin {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &DBMetricsPlugin{
		metrics: metrics,
		logger:  logger,
	}
}

// Name returns the plugin name.
func (p *DBMetricsPlugin) Name() string {
	return "db_metrics"
}

// Initialize registers the GORM callbacks for metrics collection.
func (p *DBMetricsPlugin) Initialize(db *gorm.DB) error {
	// Register before callbacks to set query start time
	if err := p.registerBeforeCallbacks(db); err != nil {
		return err
	}

	// Register after callbacks to record metrics
	if err := p.registerAfterCallbacks(db); err != nil {
		return err
	}

	p.logger.Info("Database metrics plugin initialized")
	return nil
}

// registerBeforeCallbacks registers callbacks that run before DB operations.
func (p *DBMetricsPlugin) registerBeforeCallbacks(db *gorm.DB) error {
	beforeCallback := func(db *gorm.DB) {
		ctx := db.Statement.Context
		if ctx == nil {
			ctx = context.Background()
		}
		db.Statement.Context = context.WithValue(
			ctx,
			dbMetricsStartTimeKey,
			time.Now(),
		)
	}

	// Register before callbacks for each operation type
	if err := db.Callback().Create().Before("gorm:create").Register("db_metrics:before_create", beforeCallback); err != nil {
		return err
	}
	if err := db.Callback().Query().Before("gorm:query").Register("db_metrics:before_query", beforeCallback); err != nil {
		return err
	}
	if err := db.Callback().Update().Before("gorm:update").Register("db_metrics:before_update", beforeCallback); err != nil {
		return err
	}
	if err := db.Callback().Delete().Before("gorm:delete").Register("db_metrics:before_delete", beforeCallback); err != nil {
		return err
	}
	if err := db.Callback().Row().Before("gorm:row").Register("db_metrics:before_row", beforeCallback); err != nil {
		return err
	}
	if err := db.Callback().Raw().Before("gorm:raw").Register("db_metrics:before_raw", beforeCallback); err != nil {
		return err
	}

	return nil
}

// registerAfterCallbacks registers callbacks that run after DB operations.
func (p *DBMetricsPlugin) registerAfterCallbacks(db *gorm.DB) error {
	// Create operation callback (INSERT)
	createCallback := func(db *gorm.DB) {
		p.recordMetrics(db, "INSERT")
	}

	// Query operation callback (SELECT)
	queryCallback := func(db *gorm.DB) {
		p.recordMetrics(db, "SELECT")
	}

	// Update operation callback (UPDATE)
	updateCallback := func(db *gorm.DB) {
		p.recordMetrics(db, "UPDATE")
	}

	// Delete operation callback (DELETE)
	deleteCallback := func(db *gorm.DB) {
		p.recordMetrics(db, "DELETE")
	}

	// Row/Raw operations - detect operation from SQL
	rawCallback := func(db *gorm.DB) {
		operation := detectOperationType(db.Statement.SQL.String())
		p.recordMetrics(db, operation)
	}

	// Register after callbacks
	if err := db.Callback().Create().After("gorm:create").Register("db_metrics:after_create", createCallback); err != nil {
		return err
	}
	if err := db.Callback().Query().After("gorm:query").Register("db_metrics:after_query", queryCallback); err != nil {
		return err
	}
	if err := db.Callback().Update().After("gorm:update").Register("db_metrics:after_update", updateCallback); err != nil {
		return err
	}
	if err := db.Callback().Delete().After("gorm:delete").Register("db_metrics:after_delete", deleteCallback); err != nil {
		return err
	}
	if err := db.Callback().Row().After("gorm:row").Register("db_metrics:after_row", rawCallback); err != nil {
		return err
	}
	if err := db.Callback().Raw().After("gorm:raw").Register("db_metrics:after_raw", rawCallback); err != nil {
		return err
	}

	return nil
}

// recordMetrics records metrics for a completed database operation.
func (p *DBMetricsPlugin) recordMetrics(db *gorm.DB, operation string) {
	ctx := db.Statement.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Calculate duration
	var duration time.Duration
	if startTime, ok := ctx.Value(dbMetricsStartTimeKey).(time.Time); ok {
		duration = time.Since(startTime)
	}

	// Get table name
	table := db.Statement.Table

	// Record the metrics
	p.metrics.RecordQuery(ctx, operation, table, duration, db.Error)
}

// detectOperationType attempts to detect the SQL operation type from the query.
func detectOperationType(sql string) string {
	sql = strings.TrimSpace(strings.ToUpper(sql))

	switch {
	case strings.HasPrefix(sql, "SELECT"):
		return "SELECT"
	case strings.HasPrefix(sql, "INSERT"):
		return "INSERT"
	case strings.HasPrefix(sql, "UPDATE"):
		return "UPDATE"
	case strings.HasPrefix(sql, "DELETE"):
		return "DELETE"
	default:
		return "OTHER"
	}
}

// dbMetricsStartTimeKey is the context key for storing query start time.
type dbMetricsContextKey string

const dbMetricsStartTimeKey dbMetricsContextKey = "db_metrics_start_time"

// =============================================================================
// Helper Functions for Integration
// =============================================================================

// RegisterDBMetrics creates and registers database metrics on a GORM DB instance.
// It returns the DBMetrics instance for lifecycle management (call Stop() on shutdown).
func RegisterDBMetrics(db *gorm.DB, meterProvider *MeterProvider, cfg DBMetricsConfig, logger *zap.Logger) (*DBMetrics, error) {
	if !cfg.Enabled {
		logger.Debug("Database metrics disabled, skipping registration")
		return nil, nil
	}

	if meterProvider == nil || !meterProvider.IsEnabled() {
		logger.Debug("MeterProvider not available, skipping database metrics")
		return nil, nil
	}

	// Create meter
	meter := meterProvider.Meter("db.client")

	// Create metrics
	metrics, err := NewDBMetrics(meter, cfg, logger)
	if err != nil {
		return nil, err
	}

	// Get sql.DB for pool stats
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	metrics.SetSQLDB(sqlDB)

	// Register GORM plugin for query metrics
	plugin := NewDBMetricsPlugin(metrics, logger)
	if err := db.Use(plugin); err != nil {
		return nil, err
	}

	logger.Info("Database metrics registered",
		zap.Duration("slow_query_threshold", cfg.SlowQueryThreshold),
		zap.Duration("pool_stats_interval", cfg.PoolStatsInterval),
	)

	return metrics, nil
}
