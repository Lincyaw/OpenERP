// Package telemetry provides OpenTelemetry integration for distributed tracing.
package telemetry

import (
	"context"
	"time"

	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// DBTracingConfig holds configuration for database tracing.
type DBTracingConfig struct {
	Enabled          bool          // Enable database tracing
	LogFullSQL       bool          // Include full SQL statements in spans (dev only, security risk in prod)
	SlowQueryThresh  time.Duration // Threshold for marking queries as slow (default: 200ms)
	DBSystem         string        // Database system name (default: "postgresql")
	WithoutVariables bool          // Exclude query variables from SQL statement (for security)
}

// DefaultDBTracingConfig returns default configuration for database tracing.
func DefaultDBTracingConfig() DBTracingConfig {
	return DBTracingConfig{
		Enabled:          false,
		LogFullSQL:       false,
		SlowQueryThresh:  200 * time.Millisecond,
		DBSystem:         "postgresql",
		WithoutVariables: true, // Default to secure mode
	}
}

// DBTracingPlugin wraps otelgorm plugin with custom slow query detection.
type DBTracingPlugin struct {
	config DBTracingConfig
	logger *zap.Logger
}

// NewDBTracingPlugin creates a new database tracing plugin with the given configuration.
func NewDBTracingPlugin(cfg DBTracingConfig, logger *zap.Logger) *DBTracingPlugin {
	return &DBTracingPlugin{
		config: cfg,
		logger: logger,
	}
}

// RegisterOtelGorm registers the otelgorm plugin with the given GORM DB instance.
// It also registers a custom callback for slow query detection and error marking.
// Returns error if registration fails.
func (p *DBTracingPlugin) RegisterOtelGorm(db *gorm.DB) error {
	if !p.config.Enabled {
		p.logger.Debug("Database tracing disabled, skipping otelgorm registration")
		return nil
	}

	// Build otelgorm options
	opts := []otelgorm.Option{
		otelgorm.WithDBName(p.config.DBSystem),
	}

	// Configure SQL statement visibility
	if !p.config.LogFullSQL {
		// Don't include query parameters in spans for security
		opts = append(opts, otelgorm.WithoutQueryVariables())
	}

	// Create and register the otelgorm plugin
	plugin := otelgorm.NewPlugin(opts...)
	if err := db.Use(plugin); err != nil {
		return err
	}

	// Register before callbacks to set query start time
	if err := p.registerBeforeCallbacks(db); err != nil {
		return err
	}

	// Register custom callback for slow query detection (runs after otelgorm)
	if err := p.registerSlowQueryCallback(db); err != nil {
		return err
	}

	p.logger.Info("Database tracing enabled",
		zap.Bool("log_full_sql", p.config.LogFullSQL),
		zap.Duration("slow_query_threshold", p.config.SlowQueryThresh),
		zap.String("db_system", p.config.DBSystem),
	)

	return nil
}

// registerBeforeCallbacks adds before callbacks to set query start time.
func (p *DBTracingPlugin) registerBeforeCallbacks(db *gorm.DB) error {
	beforeCallback := func(db *gorm.DB) {
		if db.Statement.Context != nil {
			db.Statement.Context = context.WithValue(db.Statement.Context, queryStartTimeKey, time.Now())
		}
	}

	if err := db.Callback().Create().Before("gorm:create").Register("otel_timing:before_create", beforeCallback); err != nil {
		return err
	}
	if err := db.Callback().Query().Before("gorm:query").Register("otel_timing:before_query", beforeCallback); err != nil {
		return err
	}
	if err := db.Callback().Update().Before("gorm:update").Register("otel_timing:before_update", beforeCallback); err != nil {
		return err
	}
	if err := db.Callback().Delete().Before("gorm:delete").Register("otel_timing:before_delete", beforeCallback); err != nil {
		return err
	}
	if err := db.Callback().Row().Before("gorm:row").Register("otel_timing:before_row", beforeCallback); err != nil {
		return err
	}
	if err := db.Callback().Raw().Before("gorm:raw").Register("otel_timing:before_raw", beforeCallback); err != nil {
		return err
	}

	return nil
}

// registerSlowQueryCallback adds a custom callback to detect slow queries and mark errors.
func (p *DBTracingPlugin) registerSlowQueryCallback(db *gorm.DB) error {
	// Register callback after all GORM operations (Create, Query, Update, Delete, Row, Raw)
	callbacks := []struct {
		name     string
		callback func(*gorm.DB)
	}{
		{"otel_slow_query:create", p.slowQueryCallback},
		{"otel_slow_query:query", p.slowQueryCallback},
		{"otel_slow_query:update", p.slowQueryCallback},
		{"otel_slow_query:delete", p.slowQueryCallback},
		{"otel_slow_query:row", p.slowQueryCallback},
		{"otel_slow_query:raw", p.slowQueryCallback},
	}

	// Register for each operation type
	if err := db.Callback().Create().After("gorm:create").Register(callbacks[0].name, callbacks[0].callback); err != nil {
		return err
	}
	if err := db.Callback().Query().After("gorm:query").Register(callbacks[1].name, callbacks[1].callback); err != nil {
		return err
	}
	if err := db.Callback().Update().After("gorm:update").Register(callbacks[2].name, callbacks[2].callback); err != nil {
		return err
	}
	if err := db.Callback().Delete().After("gorm:delete").Register(callbacks[3].name, callbacks[3].callback); err != nil {
		return err
	}
	if err := db.Callback().Row().After("gorm:row").Register(callbacks[4].name, callbacks[4].callback); err != nil {
		return err
	}
	if err := db.Callback().Raw().After("gorm:raw").Register(callbacks[5].name, callbacks[5].callback); err != nil {
		return err
	}

	return nil
}

// slowQueryCallback is called after each database operation to detect slow queries and errors.
func (p *DBTracingPlugin) slowQueryCallback(db *gorm.DB) {
	ctx := db.Statement.Context
	if ctx == nil {
		return
	}

	// Get the current span from context
	span := trace.SpanFromContext(ctx)
	if span == nil || !span.IsRecording() {
		return
	}

	// Add rows affected attribute
	if db.Statement.RowsAffected >= 0 {
		span.SetAttributes(attribute.Int64("db.rows_affected", db.Statement.RowsAffected))
	}

	// Add table name attribute if available
	if db.Statement.Table != "" {
		span.SetAttributes(attribute.String("db.sql.table", db.Statement.Table))
	}

	// Mark errors on the span
	if db.Error != nil && db.Error != gorm.ErrRecordNotFound {
		span.SetStatus(codes.Error, db.Error.Error())
		span.RecordError(db.Error)
	}

	// Check for slow query using the start time from before callback
	if startTime, ok := ctx.Value(queryStartTimeKey).(time.Time); ok {
		elapsed := time.Since(startTime)
		if elapsed > p.config.SlowQueryThresh {
			span.SetAttributes(attribute.Bool("db.slow_query", true))
			span.SetAttributes(attribute.Int64("db.query_duration_ms", elapsed.Milliseconds()))
			span.AddEvent("slow_query_warning", trace.WithAttributes(
				attribute.Int64("duration_ms", elapsed.Milliseconds()),
				attribute.Int64("threshold_ms", p.config.SlowQueryThresh.Milliseconds()),
			))
		}
	}
}

// queryStartTimeKey is the context key for storing query start time.
type contextKey string

const queryStartTimeKey contextKey = "otel_query_start_time"

// WithQueryStartTime returns a context with the query start time set.
// This is used by the slow query callback to calculate elapsed time.
func WithQueryStartTime(ctx context.Context) context.Context {
	return context.WithValue(ctx, queryStartTimeKey, time.Now())
}

// DBTracingCallback provides a GORM callback that tracks query start time
// for accurate slow query detection.
type DBTracingCallback struct {
	slowQueryThresh time.Duration
}

// NewDBTracingCallback creates a new callback for tracking query timing.
func NewDBTracingCallback(slowQueryThresh time.Duration) *DBTracingCallback {
	return &DBTracingCallback{
		slowQueryThresh: slowQueryThresh,
	}
}

// BeforeCallback sets the query start time in context.
func (c *DBTracingCallback) BeforeCallback(db *gorm.DB) {
	if db.Statement.Context != nil {
		db.Statement.Context = context.WithValue(db.Statement.Context, queryStartTimeKey, time.Now())
	}
}

// AfterCallback checks for slow queries and adds attributes to the span.
func (c *DBTracingCallback) AfterCallback(db *gorm.DB) {
	if db.Statement.Context == nil {
		return
	}

	span := trace.SpanFromContext(db.Statement.Context)
	if span == nil || !span.IsRecording() {
		return
	}

	// Add rows affected
	if db.Statement.RowsAffected >= 0 {
		span.SetAttributes(attribute.Int64("db.rows_affected", db.Statement.RowsAffected))
	}

	// Add table name
	if db.Statement.Table != "" {
		span.SetAttributes(attribute.String("db.sql.table", db.Statement.Table))
	}

	// Check for errors (excluding ErrRecordNotFound which is expected behavior)
	if db.Error != nil && db.Error != gorm.ErrRecordNotFound {
		span.SetStatus(codes.Error, db.Error.Error())
		span.RecordError(db.Error)
	}

	// Check for slow query
	if startTime, ok := db.Statement.Context.Value(queryStartTimeKey).(time.Time); ok {
		elapsed := time.Since(startTime)
		if elapsed > c.slowQueryThresh {
			span.SetAttributes(attribute.Bool("db.slow_query", true))
			span.SetAttributes(attribute.Int64("db.query_duration_ms", elapsed.Milliseconds()))
			span.AddEvent("slow_query_warning", trace.WithAttributes(
				attribute.Int64("duration_ms", elapsed.Milliseconds()),
				attribute.Int64("threshold_ms", c.slowQueryThresh.Milliseconds()),
			))
		}
	}
}

// RegisterCallbacks registers the before and after callbacks on the GORM DB instance.
func (c *DBTracingCallback) RegisterCallbacks(db *gorm.DB) error {
	// Register before callbacks to set start time
	if err := db.Callback().Create().Before("gorm:create").Register("otel_timing:before_create", c.BeforeCallback); err != nil {
		return err
	}
	if err := db.Callback().Query().Before("gorm:query").Register("otel_timing:before_query", c.BeforeCallback); err != nil {
		return err
	}
	if err := db.Callback().Update().Before("gorm:update").Register("otel_timing:before_update", c.BeforeCallback); err != nil {
		return err
	}
	if err := db.Callback().Delete().Before("gorm:delete").Register("otel_timing:before_delete", c.BeforeCallback); err != nil {
		return err
	}
	if err := db.Callback().Row().Before("gorm:row").Register("otel_timing:before_row", c.BeforeCallback); err != nil {
		return err
	}
	if err := db.Callback().Raw().Before("gorm:raw").Register("otel_timing:before_raw", c.BeforeCallback); err != nil {
		return err
	}

	// Register after callbacks to check slow queries
	if err := db.Callback().Create().After("gorm:create").Register("otel_timing:after_create", c.AfterCallback); err != nil {
		return err
	}
	if err := db.Callback().Query().After("gorm:query").Register("otel_timing:after_query", c.AfterCallback); err != nil {
		return err
	}
	if err := db.Callback().Update().After("gorm:update").Register("otel_timing:after_update", c.AfterCallback); err != nil {
		return err
	}
	if err := db.Callback().Delete().After("gorm:delete").Register("otel_timing:after_delete", c.AfterCallback); err != nil {
		return err
	}
	if err := db.Callback().Row().After("gorm:row").Register("otel_timing:after_row", c.AfterCallback); err != nil {
		return err
	}
	if err := db.Callback().Raw().After("gorm:raw").Register("otel_timing:after_raw", c.AfterCallback); err != nil {
		return err
	}

	return nil
}
