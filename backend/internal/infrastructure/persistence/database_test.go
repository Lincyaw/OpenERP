package persistence

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestConnectionStats_Struct tests that ConnectionStats struct can be properly initialized
func TestConnectionStats_Struct(t *testing.T) {
	t.Run("creates ConnectionStats with zero values", func(t *testing.T) {
		stats := ConnectionStats{}

		assert.Equal(t, 0, stats.MaxOpenConnections)
		assert.Equal(t, 0, stats.OpenConnections)
		assert.Equal(t, 0, stats.InUse)
		assert.Equal(t, 0, stats.Idle)
		assert.Equal(t, int64(0), stats.WaitCount)
		assert.Equal(t, time.Duration(0), stats.WaitDuration)
		assert.Equal(t, int64(0), stats.MaxIdleClosed)
		assert.Equal(t, int64(0), stats.MaxIdleTimeClosed)
		assert.Equal(t, int64(0), stats.MaxLifetimeClosed)
	})

	t.Run("creates ConnectionStats with custom values", func(t *testing.T) {
		stats := ConnectionStats{
			MaxOpenConnections: 25,
			OpenConnections:    10,
			InUse:              5,
			Idle:               5,
			WaitCount:          100,
			WaitDuration:       5 * time.Second,
			MaxIdleClosed:      50,
			MaxIdleTimeClosed:  30,
			MaxLifetimeClosed:  20,
		}

		assert.Equal(t, 25, stats.MaxOpenConnections)
		assert.Equal(t, 10, stats.OpenConnections)
		assert.Equal(t, 5, stats.InUse)
		assert.Equal(t, 5, stats.Idle)
		assert.Equal(t, int64(100), stats.WaitCount)
		assert.Equal(t, 5*time.Second, stats.WaitDuration)
		assert.Equal(t, int64(50), stats.MaxIdleClosed)
		assert.Equal(t, int64(30), stats.MaxIdleTimeClosed)
		assert.Equal(t, int64(20), stats.MaxLifetimeClosed)
	})

	t.Run("InUse plus Idle equals OpenConnections", func(t *testing.T) {
		stats := ConnectionStats{
			OpenConnections: 10,
			InUse:           6,
			Idle:            4,
		}

		assert.Equal(t, stats.OpenConnections, stats.InUse+stats.Idle)
	})
}

// TestDatabase_Struct tests the Database struct
func TestDatabase_Struct(t *testing.T) {
	t.Run("creates Database with nil DB", func(t *testing.T) {
		db := &Database{DB: nil}
		assert.Nil(t, db.DB)
	})
}

// newMockDatabase creates a Database instance with a mocked SQL connection
func newMockDatabase(t *testing.T) (*Database, sqlmock.Sqlmock, *sql.DB) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})

	gormDB, err := gorm.Open(dialector, &gorm.Config{
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)

	return &Database{DB: gormDB}, mock, mockDB
}

// TestDatabase_WithTenant tests the WithTenant method
func TestDatabase_WithTenant(t *testing.T) {
	t.Run("returns scoped GORM DB with tenant filter", func(t *testing.T) {
		db, mock, mockDB := newMockDatabase(t)
		defer mockDB.Close()

		tenantID := "tenant-123"

		// Create a test struct for the query
		type TestModel struct {
			ID       uint
			TenantID string
			Name     string
		}

		// Expect a query with tenant_id filter
		mock.ExpectQuery(`SELECT \* FROM "test_models" WHERE tenant_id = \$1`).
			WithArgs(tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "name"}).
				AddRow(1, tenantID, "Test Item"))

		// Use WithTenant and execute a query
		scopedDB := db.WithTenant(tenantID)
		require.NotNil(t, scopedDB)

		var results []TestModel
		err := scopedDB.Find(&results).Error
		require.NoError(t, err)

		// Verify all expectations were met
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("WithTenant does not modify original DB", func(t *testing.T) {
		db, _, mockDB := newMockDatabase(t)
		defer mockDB.Close()

		tenantID := "tenant-456"
		originalDB := db.DB

		scopedDB := db.WithTenant(tenantID)

		// Original DB should remain unchanged
		assert.NotEqual(t, originalDB, scopedDB)
		assert.Equal(t, originalDB, db.DB)
	})

	t.Run("WithTenant with empty tenant ID panics", func(t *testing.T) {
		db, _, mockDB := newMockDatabase(t)
		defer mockDB.Close()

		// WithTenant should panic when called with empty tenant ID
		assert.Panics(t, func() {
			db.WithTenant("")
		})
	})

	t.Run("WithTenant with special characters in tenant ID", func(t *testing.T) {
		db, mock, mockDB := newMockDatabase(t)
		defer mockDB.Close()

		// Test SQL injection prevention - the parameterized query should handle this safely
		tenantID := "tenant'; DROP TABLE users; --"

		type TestModel struct {
			ID       uint
			TenantID string
		}

		mock.ExpectQuery(`SELECT \* FROM "test_models" WHERE tenant_id = \$1`).
			WithArgs(tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id"}))

		scopedDB := db.WithTenant(tenantID)
		var results []TestModel
		err := scopedDB.Find(&results).Error
		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

// TestDatabase_Stats tests the Stats method
func TestDatabase_Stats(t *testing.T) {
	t.Run("returns ConnectionStats from underlying DB", func(t *testing.T) {
		db, _, mockDB := newMockDatabase(t)
		defer mockDB.Close()

		// Stats should return values (mock provides default stats)
		stats, err := db.Stats()

		// The stats should be a valid ConnectionStats struct
		// With mock, values are typically zero but the method should work
		assert.NoError(t, err)
		assert.IsType(t, ConnectionStats{}, stats)
	})
}

// TestDatabase_Ping tests the Ping method
func TestDatabase_Ping(t *testing.T) {
	t.Run("successful ping", func(t *testing.T) {
		db, mock, mockDB := newMockDatabase(t)
		defer mockDB.Close()

		mock.ExpectPing()

		err := db.Ping()
		assert.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

// TestDatabase_Close tests the Close method
func TestDatabase_Close(t *testing.T) {
	t.Run("successful close", func(t *testing.T) {
		db, mock, mockDB := newMockDatabase(t)
		_ = mockDB // We don't close mockDB here since db.Close() will do it

		mock.ExpectClose()

		err := db.Close()
		assert.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

// TestDatabase_Transaction tests the Transaction method
func TestDatabase_Transaction(t *testing.T) {
	t.Run("successful transaction", func(t *testing.T) {
		db, mock, mockDB := newMockDatabase(t)
		defer mockDB.Close()

		type TestModel struct {
			ID   uint
			Name string
		}

		mock.ExpectBegin()
		// PostgreSQL GORM uses Query with RETURNING clause instead of Exec
		mock.ExpectQuery(`INSERT INTO "test_models"`).
			WithArgs("test").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectCommit()

		err := db.Transaction(func(tx *gorm.DB) error {
			return tx.Create(&TestModel{Name: "test"}).Error
		})

		assert.NoError(t, err)
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("transaction rollback on error", func(t *testing.T) {
		db, mock, mockDB := newMockDatabase(t)
		defer mockDB.Close()

		mock.ExpectBegin()
		mock.ExpectRollback()

		err := db.Transaction(func(tx *gorm.DB) error {
			return assert.AnError
		})

		assert.Error(t, err)
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

// TestDatabase_WithTenant_ChainedQueries tests chaining WithTenant with other query methods
func TestDatabase_WithTenant_ChainedQueries(t *testing.T) {
	t.Run("WithTenant can be chained with other Where clauses", func(t *testing.T) {
		db, mock, mockDB := newMockDatabase(t)
		defer mockDB.Close()

		tenantID := "tenant-789"

		type Product struct {
			ID       uint
			TenantID string
			Name     string
			Active   bool
		}

		// Expect a query with both tenant_id and active filters
		// GORM generates: SELECT * FROM "products" WHERE tenant_id = $1 AND active = $2
		mock.ExpectQuery(`SELECT \* FROM "products" WHERE tenant_id = \$1 AND active = \$2`).
			WithArgs(tenantID, true).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "name", "active"}).
				AddRow(1, tenantID, "Product A", true))

		scopedDB := db.WithTenant(tenantID)
		var results []Product
		err := scopedDB.Where("active = ?", true).Find(&results).Error
		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("WithTenant preserves ordering", func(t *testing.T) {
		db, mock, mockDB := newMockDatabase(t)
		defer mockDB.Close()

		tenantID := "tenant-order"

		type Item struct {
			ID       uint
			TenantID string
			Name     string
		}

		mock.ExpectQuery(`SELECT \* FROM "items" WHERE tenant_id = \$1 ORDER BY name ASC`).
			WithArgs(tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "name"}).
				AddRow(1, tenantID, "Alpha").
				AddRow(2, tenantID, "Beta"))

		scopedDB := db.WithTenant(tenantID)
		var results []Item
		err := scopedDB.Order("name ASC").Find(&results).Error
		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("WithTenant with limit and offset", func(t *testing.T) {
		db, mock, mockDB := newMockDatabase(t)
		defer mockDB.Close()

		tenantID := "tenant-pagination"

		type Record struct {
			ID       uint
			TenantID string
		}

		mock.ExpectQuery(`SELECT \* FROM "records" WHERE tenant_id = \$1 LIMIT \$2 OFFSET \$3`).
			WithArgs(tenantID, 10, 5).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id"}).
				AddRow(6, tenantID))

		scopedDB := db.WithTenant(tenantID)
		var results []Record
		err := scopedDB.Limit(10).Offset(5).Find(&results).Error
		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

// TestDatabase_Stats_EdgeCases tests Stats method edge cases
func TestDatabase_Stats_EdgeCases(t *testing.T) {
	t.Run("Stats returns valid struct with all fields", func(t *testing.T) {
		db, _, mockDB := newMockDatabase(t)
		defer mockDB.Close()

		stats, err := db.Stats()

		// Verify the stats struct has the correct type for all fields
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, stats.MaxOpenConnections, 0)
		assert.GreaterOrEqual(t, stats.OpenConnections, 0)
		assert.GreaterOrEqual(t, stats.InUse, 0)
		assert.GreaterOrEqual(t, stats.Idle, 0)
		assert.GreaterOrEqual(t, stats.WaitCount, int64(0))
		assert.GreaterOrEqual(t, stats.WaitDuration, time.Duration(0))
		assert.GreaterOrEqual(t, stats.MaxIdleClosed, int64(0))
		assert.GreaterOrEqual(t, stats.MaxIdleTimeClosed, int64(0))
		assert.GreaterOrEqual(t, stats.MaxLifetimeClosed, int64(0))
	})
}

// TestDatabase_MultiTenant tests multi-tenant isolation scenarios
func TestDatabase_MultiTenant(t *testing.T) {
	t.Run("different tenants get isolated scopes", func(t *testing.T) {
		db, _, mockDB := newMockDatabase(t)
		defer mockDB.Close()

		tenant1DB := db.WithTenant("tenant-1")
		tenant2DB := db.WithTenant("tenant-2")

		// The two scoped DBs should be different instances
		assert.NotEqual(t, tenant1DB, tenant2DB)
	})

	t.Run("WithTenant with UUID format tenant ID", func(t *testing.T) {
		db, mock, mockDB := newMockDatabase(t)
		defer mockDB.Close()

		tenantID := "550e8400-e29b-41d4-a716-446655440000"

		type Entity struct {
			ID       uint
			TenantID string
		}

		mock.ExpectQuery(`SELECT \* FROM "entities" WHERE tenant_id = \$1`).
			WithArgs(tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id"}).
				AddRow(1, tenantID))

		scopedDB := db.WithTenant(tenantID)
		var results []Entity
		err := scopedDB.Find(&results).Error
		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

// TestDatabase_Ping_EdgeCases tests Ping method edge cases
func TestDatabase_Ping_EdgeCases(t *testing.T) {
	t.Run("ping with MonitorPingsOption enabled", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer mockDB.Close()

		// GORM may ping during Open, so expect it first
		mock.ExpectPing()

		dialector := postgres.New(postgres.Config{
			Conn:       mockDB,
			DriverName: "postgres",
		})

		gormDB, err := gorm.Open(dialector, &gorm.Config{
			SkipDefaultTransaction: true,
		})
		require.NoError(t, err)

		db := &Database{DB: gormDB}

		// Now expect the actual Ping call
		mock.ExpectPing()

		err = db.Ping()
		assert.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}
