package persistence

import (
	"fmt"
	"time"

	"github.com/erp/backend/internal/infrastructure/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database holds the database connection and provides methods for database operations
type Database struct {
	DB *gorm.DB
}

// NewDatabase creates a new database connection with the given configuration
func NewDatabase(cfg *config.DatabaseConfig) (*Database, error) {
	return newDatabaseWithLogLevel(cfg, logger.Silent)
}

// NewDatabaseWithLogger creates a new database connection with custom logger settings
func NewDatabaseWithLogger(cfg *config.DatabaseConfig, logLevel logger.LogLevel) (*Database, error) {
	return newDatabaseWithLogLevel(cfg, logLevel)
}

// newDatabaseWithLogLevel is the internal function that creates database connections
func newDatabaseWithLogLevel(cfg *config.DatabaseConfig, logLevel logger.LogLevel) (*Database, error) {
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

	return &Database{DB: db}, nil
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
