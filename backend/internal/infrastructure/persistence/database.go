package persistence

import (
	"fmt"
	"time"

	"github.com/erp/backend/internal/infrastructure/config"
	"github.com/erp/backend/internal/infrastructure/telemetry"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database holds the database connection and provides methods for database operations
type Database struct {
	DB     *gorm.DB
	logger *zap.Logger
}

// NewDatabase creates a new database connection with the given configuration
func NewDatabase(cfg *config.DatabaseConfig) (*Database, error) {
	return newDatabaseWithLogLevel(cfg, logger.Silent, nil, nil)
}

// NewDatabaseWithLogger creates a new database connection with custom logger settings
func NewDatabaseWithLogger(cfg *config.DatabaseConfig, logLevel logger.LogLevel) (*Database, error) {
	return newDatabaseWithLogLevel(cfg, logLevel, nil, nil)
}

// NewDatabaseWithCustomLogger creates a new database connection with a custom GORM logger
func NewDatabaseWithCustomLogger(cfg *config.DatabaseConfig, gormLogger logger.Interface) (*Database, error) {
	return newDatabaseWithCustomLogger(cfg, gormLogger, nil, nil)
}

// NewDatabaseWithTracing creates a new database connection with OpenTelemetry tracing enabled.
// The telemetryCfg controls database tracing options (slow query threshold, SQL logging).
// The zapLogger is used for logging tracing-related messages.
func NewDatabaseWithTracing(cfg *config.DatabaseConfig, gormLogger logger.Interface, telemetryCfg *config.TelemetryConfig, zapLogger *zap.Logger) (*Database, error) {
	return newDatabaseWithCustomLogger(cfg, gormLogger, telemetryCfg, zapLogger)
}

// newDatabaseWithLogLevel is the internal function that creates database connections
func newDatabaseWithLogLevel(cfg *config.DatabaseConfig, logLevel logger.LogLevel, telemetryCfg *config.TelemetryConfig, zapLogger *zap.Logger) (*Database, error) {
	dsn := cfg.DSN()
	gormLogger := logger.Default.LogMode(logLevel)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                 gormLogger,
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return configureConnectionPool(db, cfg, telemetryCfg, zapLogger)
}

// newDatabaseWithCustomLogger creates database connection with a custom logger
func newDatabaseWithCustomLogger(cfg *config.DatabaseConfig, customLogger logger.Interface, telemetryCfg *config.TelemetryConfig, zapLogger *zap.Logger) (*Database, error) {
	dsn := cfg.DSN()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                 customLogger,
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return configureConnectionPool(db, cfg, telemetryCfg, zapLogger)
}

// configureConnectionPool sets up the connection pool and pings the database
func configureConnectionPool(db *gorm.DB, cfg *config.DatabaseConfig, telemetryCfg *config.TelemetryConfig, zapLogger *zap.Logger) (*Database, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Duration(cfg.ConnMaxIdleTime) * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure database tracing if enabled
	if telemetryCfg != nil && telemetryCfg.DBTraceEnabled && telemetryCfg.Enabled {
		if zapLogger == nil {
			zapLogger = zap.NewNop()
		}
		tracingCfg := telemetry.DBTracingConfig{
			Enabled:          true,
			LogFullSQL:       telemetryCfg.DBLogFullSQL,
			SlowQueryThresh:  telemetryCfg.DBSlowQueryThresh,
			DBSystem:         "postgresql",
			WithoutVariables: !telemetryCfg.DBLogFullSQL, // Hide variables unless full SQL is requested
		}
		plugin := telemetry.NewDBTracingPlugin(tracingCfg, zapLogger)
		if err := plugin.RegisterOtelGorm(db); err != nil {
			return nil, fmt.Errorf("failed to register database tracing plugin: %w", err)
		}
		// Note: DBTracingPlugin.RegisterOtelGorm already registers slow query callbacks
		// via otelgorm and custom callbacks, so we don't need additional timing callbacks
	}

	return &Database{DB: db, logger: zapLogger}, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	return sqlDB.Close()
}

// Ping checks if the database connection is alive
func (d *Database) Ping() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	return sqlDB.Ping()
}

// Stats returns database connection pool statistics and an error if unable to retrieve
func (d *Database) Stats() (ConnectionStats, error) {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return ConnectionStats{}, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	stats := sqlDB.Stats()
	return ConnectionStats{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration,
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxIdleTimeClosed:  stats.MaxIdleTimeClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
	}, nil
}

// ConnectionStats holds database connection pool statistics
type ConnectionStats struct {
	MaxOpenConnections int
	OpenConnections    int
	InUse              int
	Idle               int
	WaitCount          int64
	WaitDuration       time.Duration
	MaxIdleClosed      int64
	MaxIdleTimeClosed  int64
	MaxLifetimeClosed  int64
}

// Transaction executes a function within a database transaction
func (d *Database) Transaction(fn func(tx *gorm.DB) error) error {
	return d.DB.Transaction(fn)
}

// WithTenant returns a new GORM DB instance scoped to a specific tenant.
// Panics if tenantID is empty to prevent data leakage.
func (d *Database) WithTenant(tenantID string) *gorm.DB {
	if tenantID == "" {
		panic("WithTenant called with empty tenant ID - this is a programming error")
	}
	return d.DB.Where("tenant_id = ?", tenantID)
}
