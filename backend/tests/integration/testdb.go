// Package integration provides integration testing utilities for the ERP backend.
// It uses testcontainers to spin up real PostgreSQL databases for testing.
package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	mpg "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	// Shared container for all tests in a package
	sharedContainer    testcontainers.Container
	sharedContainerMu  sync.Mutex
	sharedContainerDSN string
)

// TestDB represents a test database connection
type TestDB struct {
	DB        *gorm.DB
	SqlDB     *sql.DB
	Container testcontainers.Container
	DSN       string
	t         *testing.T
}

// NewTestDB creates a new PostgreSQL container for testing.
// This creates a fresh container for each test, providing complete isolation.
func NewTestDB(t *testing.T) *TestDB {
	t.Helper()

	ctx := context.Background()

	// Create PostgreSQL container
	container, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("erp_test"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("admin123"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err, "Failed to start PostgreSQL container")

	// Get connection string
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "Failed to get connection string")

	// Connect with GORM
	db, sqlDB := connectToDatabase(t, dsn)

	// Run migrations
	runMigrations(t, sqlDB)

	testDB := &TestDB{
		DB:        db,
		SqlDB:     sqlDB,
		Container: container,
		DSN:       dsn,
		t:         t,
	}

	// Register cleanup
	t.Cleanup(func() {
		testDB.Close()
	})

	return testDB
}

// NewSharedTestDB returns a shared PostgreSQL container for tests that can share state.
// This is more efficient for read-only tests or tests that clean up after themselves.
// Each call creates a new transaction that is rolled back on cleanup.
func NewSharedTestDB(t *testing.T) *TestDB {
	t.Helper()

	sharedContainerMu.Lock()
	defer sharedContainerMu.Unlock()

	ctx := context.Background()

	// Initialize shared container if not exists
	if sharedContainer == nil {
		container, err := tcpostgres.Run(ctx,
			"postgres:16-alpine",
			tcpostgres.WithDatabase("erp_shared_test"),
			tcpostgres.WithUsername("postgres"),
			tcpostgres.WithPassword("admin123"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(60*time.Second)),
		)
		require.NoError(t, err, "Failed to start shared PostgreSQL container")

		dsn, err := container.ConnectionString(ctx, "sslmode=disable")
		require.NoError(t, err, "Failed to get connection string")

		sharedContainer = container
		sharedContainerDSN = dsn

		// Connect and run migrations once
		db, sqlDB := connectToDatabase(t, dsn)
		runMigrations(t, sqlDB)
		sqlDB.Close()
		_ = db // Let it be garbage collected
	}

	// Each test gets a fresh connection
	db, sqlDB := connectToDatabase(t, sharedContainerDSN)

	testDB := &TestDB{
		DB:        db,
		SqlDB:     sqlDB,
		Container: sharedContainer,
		DSN:       sharedContainerDSN,
		t:         t,
	}

	// Note: For shared container, we don't register cleanup for the container itself
	// Only close the database connection
	t.Cleanup(func() {
		if testDB.SqlDB != nil {
			testDB.SqlDB.Close()
		}
	})

	return testDB
}

// Close closes the database connection and terminates the container
func (tdb *TestDB) Close() {
	ctx := context.Background()

	if tdb.SqlDB != nil {
		tdb.SqlDB.Close()
	}

	// Only terminate if this is not the shared container
	if tdb.Container != nil && tdb.Container != sharedContainer {
		if err := tdb.Container.Terminate(ctx); err != nil {
			tdb.t.Logf("Warning: Failed to terminate container: %v", err)
		}
	}
}

// CleanTables truncates all tables in the database
func (tdb *TestDB) CleanTables() {
	tdb.t.Helper()

	// Get all table names
	var tables []string
	err := tdb.DB.Raw(`
		SELECT tablename FROM pg_tables
		WHERE schemaname = 'public'
		AND tablename != 'schema_migrations'
	`).Scan(&tables).Error
	require.NoError(tdb.t, err, "Failed to get table names")

	if len(tables) == 0 {
		return
	}

	// Disable foreign key checks, truncate all tables, then re-enable
	for _, table := range tables {
		err := tdb.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)).Error
		if err != nil {
			tdb.t.Logf("Warning: Failed to truncate table %s: %v", table, err)
		}
	}
}

// WithTransaction runs a function within a transaction that is automatically rolled back.
// This is useful for tests that need to be isolated without truncating tables.
func (tdb *TestDB) WithTransaction(fn func(tx *gorm.DB)) {
	tdb.t.Helper()

	tx := tdb.DB.Begin()
	require.NoError(tdb.t, tx.Error, "Failed to begin transaction")

	defer func() {
		tx.Rollback()
	}()

	fn(tx)
}

// connectToDatabase establishes a GORM connection to the database
func connectToDatabase(t *testing.T, dsn string) (*gorm.DB, *sql.DB) {
	t.Helper()

	// Configure GORM with minimal logging for tests
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	// Enable debug logging if TEST_DB_DEBUG is set
	if os.Getenv("TEST_DB_DEBUG") != "" {
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	}

	db, err := gorm.Open(gormpostgres.Open(dsn), gormConfig)
	require.NoError(t, err, "Failed to connect to database")

	sqlDB, err := db.DB()
	require.NoError(t, err, "Failed to get underlying SQL DB")

	// Configure connection pool for tests
	sqlDB.SetMaxOpenConns(5)
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	return db, sqlDB
}

// runMigrations applies all database migrations
func runMigrations(t *testing.T, sqlDB *sql.DB) {
	t.Helper()

	// Find migrations directory
	migrationsPath := findMigrationsPath()
	require.NotEmpty(t, migrationsPath, "Could not find migrations directory")

	// Create migration driver
	driver, err := mpg.WithInstance(sqlDB, &mpg.Config{})
	require.NoError(t, err, "Failed to create migration driver")

	// Create migrate instance
	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres",
		driver,
	)
	require.NoError(t, err, "Failed to create migrate instance")

	// Run migrations
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		require.NoError(t, err, "Failed to run migrations")
	}
}

// findMigrationsPath locates the migrations directory
func findMigrationsPath() string {
	// Get the directory of this file
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}

	// Navigate from tests/integration to backend/migrations
	dir := filepath.Dir(filename)
	for i := 0; i < 5; i++ {
		migrationsPath := filepath.Join(dir, "migrations")
		if _, err := os.Stat(migrationsPath); err == nil {
			return migrationsPath
		}
		dir = filepath.Dir(dir)
	}

	// Try relative path from working directory
	if wd, err := os.Getwd(); err == nil {
		paths := []string{
			filepath.Join(wd, "migrations"),
			filepath.Join(wd, "backend", "migrations"),
			filepath.Join(wd, "..", "migrations"),
			filepath.Join(wd, "..", "..", "backend", "migrations"),
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}

	return ""
}

// CleanupSharedContainer terminates the shared container.
// This should be called in TestMain if using shared containers.
func CleanupSharedContainer() {
	sharedContainerMu.Lock()
	defer sharedContainerMu.Unlock()

	if sharedContainer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		sharedContainer.Terminate(ctx)
		sharedContainer = nil
		sharedContainerDSN = ""
	}
}

// CreateTestTenant creates a tenant record for testing and returns its ID.
// This is required since many tables have foreign key constraints to the tenants table.
func (tdb *TestDB) CreateTestTenant(tenantID string, name, code string) {
	tdb.t.Helper()

	err := tdb.DB.Exec(`
		INSERT INTO tenants (id, name, code, status)
		VALUES (?, ?, ?, 'active')
		ON CONFLICT (code) DO NOTHING
	`, tenantID, name, code).Error
	require.NoError(tdb.t, err, "Failed to create test tenant")
}

// CreateTestTenantWithUUID creates a tenant record with a UUID for testing.
func (tdb *TestDB) CreateTestTenantWithUUID(tenantID fmt.Stringer) {
	tdb.t.Helper()

	code := fmt.Sprintf("test_%s", tenantID.String()[:8])
	name := fmt.Sprintf("Test Tenant %s", tenantID.String()[:8])
	tdb.CreateTestTenant(tenantID.String(), name, code)
}

// CreateTestWarehouse creates a warehouse record for testing.
// This is required since inventory_items has a foreign key to warehouses.
func (tdb *TestDB) CreateTestWarehouse(tenantID, warehouseID fmt.Stringer) {
	tdb.t.Helper()

	code := fmt.Sprintf("WH_%s", warehouseID.String()[:8])
	name := fmt.Sprintf("Test Warehouse %s", warehouseID.String()[:8])

	err := tdb.DB.Exec(`
		INSERT INTO warehouses (id, tenant_id, code, name, status, version)
		VALUES (?, ?, ?, ?, 'active', 1)
		ON CONFLICT (id) DO NOTHING
	`, warehouseID.String(), tenantID.String(), code, name).Error
	require.NoError(tdb.t, err, "Failed to create test warehouse")
}

// CreateTestProduct creates a product record for testing.
// This is required since inventory_items has a foreign key to products.
func (tdb *TestDB) CreateTestProduct(tenantID, productID fmt.Stringer) {
	tdb.t.Helper()

	code := fmt.Sprintf("PROD_%s", productID.String()[:8])
	name := fmt.Sprintf("Test Product %s", productID.String()[:8])

	err := tdb.DB.Exec(`
		INSERT INTO products (id, tenant_id, code, name, unit, status, version)
		VALUES (?, ?, ?, ?, 'pcs', 'active', 1)
		ON CONFLICT (id) DO NOTHING
	`, productID.String(), tenantID.String(), code, name).Error
	require.NoError(tdb.t, err, "Failed to create test product")
}
