package tenant

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/erp/backend/internal/infrastructure/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupCallbackMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
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

func TestTenantCallback_RegisterCallbacks(t *testing.T) {
	db, _, mockDB := setupCallbackMockDB(t)
	defer mockDB.Close()

	tc := NewTenantCallback("tenant_id", true)

	// Should not panic
	tc.RegisterCallbacks(db)
}

func TestEnableAutoTenantFilter(t *testing.T) {
	db, _, mockDB := setupCallbackMockDB(t)
	defer mockDB.Close()

	// Should not panic
	EnableAutoTenantFilter(db, true)
}

func TestDisableAutoTenantFilter(t *testing.T) {
	db, _, mockDB := setupCallbackMockDB(t)
	defer mockDB.Close()

	EnableAutoTenantFilter(db, true)

	// Should not panic when removing callbacks
	DisableAutoTenantFilter(db)
}

func TestNewTenantCallback_DefaultColumn(t *testing.T) {
	// Empty column should default to "tenant_id"
	tc := NewTenantCallback("", true)
	assert.Equal(t, "tenant_id", tc.tenantColumn)
	assert.True(t, tc.required)
}

func TestNewTenantCallback_CustomColumn(t *testing.T) {
	tc := NewTenantCallback("org_id", false)
	assert.Equal(t, "org_id", tc.tenantColumn)
	assert.False(t, tc.required)
}

func TestTenantCallback_RequiredEnforcement(t *testing.T) {
	t.Run("errors when tenant required but missing in context", func(t *testing.T) {
		db, _, mockDB := setupCallbackMockDB(t)
		defer mockDB.Close()

		EnableAutoTenantFilter(db, true) // Required=true

		ctx := context.Background() // No tenant ID
		var results []TestModel

		err := db.WithContext(ctx).Find(&results).Error

		assert.ErrorIs(t, err, ErrTenantIDRequired)
	})
}

func TestTenantCallback_InvalidUUID(t *testing.T) {
	t.Run("errors on invalid UUID format", func(t *testing.T) {
		db, _, mockDB := setupCallbackMockDB(t)
		defer mockDB.Close()

		EnableAutoTenantFilter(db, true)

		ctx := createCallbackTestContext("not-a-valid-uuid")
		var results []TestModel

		err := db.WithContext(ctx).Find(&results).Error

		assert.ErrorIs(t, err, ErrInvalidTenantID)
	})
}

func TestTenantCallback_NotRequired(t *testing.T) {
	t.Run("allows query without tenant when not required", func(t *testing.T) {
		db, mock, mockDB := setupCallbackMockDB(t)
		defer mockDB.Close()

		EnableAutoTenantFilter(db, false) // Required=false

		mock.ExpectQuery(`SELECT \* FROM "test_models"`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "name"}))

		ctx := context.Background() // No tenant ID
		var results []TestModel

		err := db.WithContext(ctx).Find(&results).Error

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func createCallbackTestContext(tenantID string) context.Context {
	ctx := context.Background()
	if tenantID != "" {
		log := logger.FromContext(ctx)
		ctx, _ = logger.WithTenantID(ctx, log, tenantID)
	}
	return ctx
}
