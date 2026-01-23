package migration

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"
)

// Migrator handles database migrations using golang-migrate
type Migrator struct {
	migrate *migrate.Migrate
	logger  *zap.Logger
}

// Config holds migration configuration
type Config struct {
	DatabaseURL    string
	MigrationsPath string
}

// New creates a new Migrator instance
func New(db *sql.DB, migrationsPath string, logger *zap.Logger) (*Migrator, error) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return &Migrator{
		migrate: m,
		logger:  logger,
	}, nil
}

// NewFromURL creates a Migrator from database URL
func NewFromURL(databaseURL, migrationsPath string, logger *zap.Logger) (*Migrator, error) {
	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsPath),
		databaseURL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return &Migrator{
		migrate: m,
		logger:  logger,
	}, nil
}

// Up runs all pending migrations
func (m *Migrator) Up() error {
	m.logger.Info("Running migrations up")

	err := m.migrate.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration up failed: %w", err)
	}

	if err == migrate.ErrNoChange {
		m.logger.Info("No migrations to apply")
		return nil
	}

	version, dirty, err := m.migrate.Version()
	if err != nil {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	m.logger.Info("Migrations completed",
		zap.Uint("version", version),
		zap.Bool("dirty", dirty),
	)

	return nil
}

// Down rolls back all migrations
func (m *Migrator) Down() error {
	m.logger.Info("Running migrations down")

	err := m.migrate.Down()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration down failed: %w", err)
	}

	if err == migrate.ErrNoChange {
		m.logger.Info("No migrations to roll back")
		return nil
	}

	m.logger.Info("All migrations rolled back")
	return nil
}

// Steps applies n migrations (positive = up, negative = down)
func (m *Migrator) Steps(n int) error {
	m.logger.Info("Running migration steps", zap.Int("steps", n))

	err := m.migrate.Steps(n)
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration steps failed: %w", err)
	}

	if err == migrate.ErrNoChange {
		m.logger.Info("No migrations to apply")
		return nil
	}

	version, dirty, err := m.migrate.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	m.logger.Info("Migration steps completed",
		zap.Uint("version", version),
		zap.Bool("dirty", dirty),
	)

	return nil
}

// GoTo migrates to a specific version
func (m *Migrator) GoTo(version uint) error {
	m.logger.Info("Migrating to version", zap.Uint("target_version", version))

	err := m.migrate.Migrate(version)
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration to version %d failed: %w", version, err)
	}

	if err == migrate.ErrNoChange {
		m.logger.Info("Already at target version")
		return nil
	}

	m.logger.Info("Migration to version completed", zap.Uint("version", version))
	return nil
}

// Version returns the current migration version
func (m *Migrator) Version() (uint, bool, error) {
	version, dirty, err := m.migrate.Version()
	if err != nil {
		if err == migrate.ErrNilVersion {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}
	return version, dirty, nil
}

// Force sets the migration version without running migrations
// Use with caution - this is for fixing dirty database state
func (m *Migrator) Force(version int) error {
	m.logger.Warn("Forcing migration version", zap.Int("version", version))

	err := m.migrate.Force(version)
	if err != nil {
		return fmt.Errorf("failed to force version %d: %w", version, err)
	}

	m.logger.Info("Migration version forced", zap.Int("version", version))
	return nil
}

// Drop drops the entire database
// Use with extreme caution - this destroys all data
func (m *Migrator) Drop() error {
	m.logger.Warn("Dropping database - all data will be lost")

	err := m.migrate.Drop()
	if err != nil {
		return fmt.Errorf("failed to drop database: %w", err)
	}

	m.logger.Info("Database dropped")
	return nil
}

// Close closes the migrator and releases resources
func (m *Migrator) Close() error {
	sourceErr, dbErr := m.migrate.Close()
	if sourceErr != nil {
		return fmt.Errorf("failed to close source: %w", sourceErr)
	}
	if dbErr != nil {
		return fmt.Errorf("failed to close database: %w", dbErr)
	}
	return nil
}
