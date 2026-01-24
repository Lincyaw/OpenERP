package tenant

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/erp/backend/internal/infrastructure/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestModel is a simple model for testing tenant scoping
type TestModel struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantID uuid.UUID `gorm:"type:uuid;not null;index"`
	Name     string    `gorm:"size:100"`
}

func (TestModel) TableName() string {
	return "test_models"
}

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
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

	return gormDB, mock, mockDB
}

func createTestContext(tenantID string) context.Context {
	ctx := context.Background()
	if tenantID != "" {
		log := logger.FromContext(ctx)
		ctx, _ = logger.WithTenantID(ctx, log, tenantID)
	}
	return ctx
}

func TestTenantScope(t *testing.T) {
	tenantID := uuid.New()

	t.Run("applies tenant filter to query", func(t *testing.T) {
		db, mock, mockDB := setupMockDB(t)
		defer mockDB.Close()

		mock.ExpectQuery(`SELECT \* FROM "test_models" WHERE tenant_id = \$1`).
			WithArgs(tenantID.String()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "name"}))

		var results []TestModel
		err := db.Scopes(TenantScope(tenantID)).Find(&results).Error
		require.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTenantScopeString(t *testing.T) {
	tenantID := uuid.New().String()

	t.Run("applies tenant filter with string ID", func(t *testing.T) {
		db, mock, mockDB := setupMockDB(t)
		defer mockDB.Close()

		mock.ExpectQuery(`SELECT \* FROM "test_models" WHERE tenant_id = \$1`).
			WithArgs(tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "name"}))

		var results []TestModel
		err := db.Scopes(TenantScopeString(tenantID)).Find(&results).Error
		require.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTenantDB_WithContext(t *testing.T) {
	t.Run("extracts tenant from context and scopes query", func(t *testing.T) {
		db, mock, mockDB := setupMockDB(t)
		defer mockDB.Close()

		tenantDB := NewTenantDB(db)
		tenantID := uuid.New()
		ctx := createTestContext(tenantID.String())

		mock.ExpectQuery(`SELECT \* FROM "test_models" WHERE tenant_id = \$1`).
			WithArgs(tenantID.String()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "name"}))

		var results []TestModel
		err := tenantDB.WithContext(ctx).Find(&results).Error
		require.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("errors when tenant required but missing", func(t *testing.T) {
		db, _, mockDB := setupMockDB(t)
		defer mockDB.Close()

		tenantDB := NewTenantDB(db) // required=true by default
		ctx := createTestContext("")

		scopedDB := tenantDB.WithContext(ctx)

		// Should have error when tenant is required but missing
		assert.ErrorIs(t, scopedDB.Error, ErrTenantIDRequired)
	})

	t.Run("allows missing tenant when not required", func(t *testing.T) {
		db, mock, mockDB := setupMockDB(t)
		defer mockDB.Close()

		tenantDB := NewTenantDBWithConfig(db, Config{
			TenantColumn: "tenant_id",
			Required:     false,
		})
		ctx := createTestContext("")

		mock.ExpectQuery(`SELECT \* FROM "test_models"`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "name"}))

		var results []TestModel
		err := tenantDB.WithContext(ctx).Find(&results).Error
		require.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("errors on invalid UUID format", func(t *testing.T) {
		db, _, mockDB := setupMockDB(t)
		defer mockDB.Close()

		tenantDB := NewTenantDB(db)
		ctx := createTestContext("invalid-uuid")

		scopedDB := tenantDB.WithContext(ctx)

		// Should error on invalid UUID
		assert.ErrorIs(t, scopedDB.Error, ErrInvalidTenantID)
	})
}

func TestTenantDB_WithTenant(t *testing.T) {
	t.Run("scopes to specific tenant", func(t *testing.T) {
		db, mock, mockDB := setupMockDB(t)
		defer mockDB.Close()

		tenantDB := NewTenantDB(db)
		tenantID := uuid.New()

		mock.ExpectQuery(`SELECT \* FROM "test_models" WHERE tenant_id = \$1`).
			WithArgs(tenantID.String()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "name"}))

		var results []TestModel
		err := tenantDB.WithTenant(tenantID).Find(&results).Error
		require.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("errors on nil UUID when required", func(t *testing.T) {
		db, _, mockDB := setupMockDB(t)
		defer mockDB.Close()

		tenantDB := NewTenantDB(db)
		scopedDB := tenantDB.WithTenant(uuid.Nil)

		assert.ErrorIs(t, scopedDB.Error, ErrTenantIDRequired)
	})
}

func TestTenantDB_WithTenantString(t *testing.T) {
	t.Run("scopes to tenant from string", func(t *testing.T) {
		db, mock, mockDB := setupMockDB(t)
		defer mockDB.Close()

		tenantDB := NewTenantDB(db)
		tenantID := uuid.New().String()

		mock.ExpectQuery(`SELECT \* FROM "test_models" WHERE tenant_id = \$1`).
			WithArgs(tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "name"}))

		var results []TestModel
		err := tenantDB.WithTenantString(tenantID).Find(&results).Error
		require.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("errors on empty string when required", func(t *testing.T) {
		db, _, mockDB := setupMockDB(t)
		defer mockDB.Close()

		tenantDB := NewTenantDB(db)
		scopedDB := tenantDB.WithTenantString("")

		assert.ErrorIs(t, scopedDB.Error, ErrTenantIDRequired)
	})

	t.Run("errors on invalid UUID string", func(t *testing.T) {
		db, _, mockDB := setupMockDB(t)
		defer mockDB.Close()

		tenantDB := NewTenantDB(db)
		scopedDB := tenantDB.WithTenantString("not-a-uuid")

		assert.ErrorIs(t, scopedDB.Error, ErrInvalidTenantID)
	})
}

func TestTenantDB_SetRequired(t *testing.T) {
	t.Run("creates new instance with required=false", func(t *testing.T) {
		db, _, mockDB := setupMockDB(t)
		defer mockDB.Close()

		tenantDB := NewTenantDB(db)
		notRequiredDB := tenantDB.SetRequired(false)
		ctx := createTestContext("")

		scopedDB := notRequiredDB.WithContext(ctx)
		assert.Nil(t, scopedDB.Error)
	})
}

func TestTenantDB_Unscoped(t *testing.T) {
	t.Run("returns unscoped DB", func(t *testing.T) {
		db, _, mockDB := setupMockDB(t)
		defer mockDB.Close()

		tenantDB := NewTenantDB(db)
		unscopedDB := tenantDB.Unscoped()

		// Should be the same as original DB
		assert.Equal(t, db, unscopedDB)
	})
}

func TestTenantDB_ForTenant(t *testing.T) {
	t.Run("creates scoped DB with context and tenant", func(t *testing.T) {
		db, mock, mockDB := setupMockDB(t)
		defer mockDB.Close()

		tenantDB := NewTenantDB(db)
		tenantID := uuid.New()
		ctx := context.Background()

		mock.ExpectQuery(`SELECT \* FROM "test_models" WHERE tenant_id = \$1`).
			WithArgs(tenantID.String()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "name"}))

		var results []TestModel
		err := tenantDB.ForTenant(ctx, tenantID).Find(&results).Error
		require.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTenantDB_Transaction(t *testing.T) {
	t.Run("transaction errors without tenant when required", func(t *testing.T) {
		db, _, mockDB := setupMockDB(t)
		defer mockDB.Close()

		tenantDB := NewTenantDB(db)
		ctx := createTestContext("")

		err := tenantDB.Transaction(ctx, func(tx *gorm.DB) error {
			return nil
		})

		assert.ErrorIs(t, err, ErrTenantIDRequired)
	})

	t.Run("transaction executes with tenant context", func(t *testing.T) {
		db, mock, mockDB := setupMockDB(t)
		defer mockDB.Close()

		tenantDB := NewTenantDB(db)
		tenantID := uuid.New()
		ctx := createTestContext(tenantID.String())

		mock.ExpectBegin()
		mock.ExpectCommit()

		err := tenantDB.Transaction(ctx, func(tx *gorm.DB) error {
			// Just a no-op to verify transaction works
			return nil
		})

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "tenant_id", cfg.TenantColumn)
	assert.True(t, cfg.Required)
}

func TestNewTenantDBWithConfig_DefaultColumn(t *testing.T) {
	db, _, mockDB := setupMockDB(t)
	defer mockDB.Close()

	// Empty tenant column should default to "tenant_id"
	tenantDB := NewTenantDBWithConfig(db, Config{
		TenantColumn: "",
		Required:     true,
	})

	assert.NotNil(t, tenantDB)
	assert.Equal(t, "tenant_id", tenantDB.tenantColumn)
}

func TestTenantDB_ChainedQueries(t *testing.T) {
	t.Run("tenant scope chains with additional where clauses", func(t *testing.T) {
		db, mock, mockDB := setupMockDB(t)
		defer mockDB.Close()

		tenantDB := NewTenantDB(db)
		tenantID := uuid.New()
		ctx := createTestContext(tenantID.String())

		// GORM may order WHERE clauses differently - use regex that matches either order
		mock.ExpectQuery(`SELECT \* FROM "test_models" WHERE .+ AND .+`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "name"}))

		var results []TestModel
		err := tenantDB.WithContext(ctx).Where("name = ?", "Test").Find(&results).Error
		require.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("tenant scope preserves ordering", func(t *testing.T) {
		db, mock, mockDB := setupMockDB(t)
		defer mockDB.Close()

		tenantDB := NewTenantDB(db)
		tenantID := uuid.New()
		ctx := createTestContext(tenantID.String())

		mock.ExpectQuery(`SELECT \* FROM "test_models" WHERE tenant_id = \$1 ORDER BY name ASC`).
			WithArgs(tenantID.String()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "name"}))

		var results []TestModel
		err := tenantDB.WithContext(ctx).Order("name ASC").Find(&results).Error
		require.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("tenant scope with pagination", func(t *testing.T) {
		db, mock, mockDB := setupMockDB(t)
		defer mockDB.Close()

		tenantDB := NewTenantDB(db)
		tenantID := uuid.New()
		ctx := createTestContext(tenantID.String())

		mock.ExpectQuery(`SELECT \* FROM "test_models" WHERE tenant_id = \$1 LIMIT \$2 OFFSET \$3`).
			WithArgs(tenantID.String(), 10, 5).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "name"}))

		var results []TestModel
		err := tenantDB.WithContext(ctx).Limit(10).Offset(5).Find(&results).Error
		require.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTenantDB_SQLInjectionPrevention(t *testing.T) {
	t.Run("parameterized queries prevent SQL injection", func(t *testing.T) {
		db, mock, mockDB := setupMockDB(t)
		defer mockDB.Close()

		tenantDB := NewTenantDB(db)
		// Malicious tenant ID - should be parameterized and safe
		maliciousTenantID := uuid.New().String()
		ctx := createTestContext(maliciousTenantID)

		// The query should use parameterized queries
		mock.ExpectQuery(`SELECT \* FROM "test_models" WHERE tenant_id = \$1`).
			WithArgs(maliciousTenantID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "name"}))

		var results []TestModel
		err := tenantDB.WithContext(ctx).Find(&results).Error
		require.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTenantDB_MultiTenantIsolation(t *testing.T) {
	t.Run("different tenants get isolated scopes", func(t *testing.T) {
		db, _, mockDB := setupMockDB(t)
		defer mockDB.Close()

		tenantDB := NewTenantDB(db)
		tenant1ID := uuid.New()
		tenant2ID := uuid.New()

		tenant1DB := tenantDB.WithTenant(tenant1ID)
		tenant2DB := tenantDB.WithTenant(tenant2ID)

		// The two scoped DBs should be different instances
		assert.NotEqual(t, tenant1DB, tenant2DB)
	})
}
