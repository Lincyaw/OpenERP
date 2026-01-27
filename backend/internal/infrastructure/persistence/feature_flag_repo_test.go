package persistence

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/erp/backend/internal/domain/featureflag"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// newMockFeatureFlagRepository creates a GormFeatureFlagRepository with a mocked SQL connection
func newMockFeatureFlagRepository(t *testing.T) (*GormFeatureFlagRepository, sqlmock.Sqlmock, *sql.DB) {
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

	return NewGormFeatureFlagRepository(gormDB), mock, mockDB
}

func TestNewGormFeatureFlagRepository(t *testing.T) {
	t.Run("creates repository with valid DB", func(t *testing.T) {
		repo, _, mockDB := newMockFeatureFlagRepository(t)
		defer mockDB.Close()

		assert.NotNil(t, repo)
		assert.NotNil(t, repo.db)
	})
}

func TestGormFeatureFlagRepository_WithTx(t *testing.T) {
	t.Run("returns new repository with transaction", func(t *testing.T) {
		repo, _, mockDB := newMockFeatureFlagRepository(t)
		defer mockDB.Close()

		txRepo := repo.WithTx(repo.db)
		assert.NotNil(t, txRepo)
		// Returns a new instance, but pointing to same DB
		assert.NotNil(t, txRepo.db)
	})
}

func TestGormFeatureFlagRepository_FindByKey(t *testing.T) {
	t.Run("finds existing feature flag", func(t *testing.T) {
		repo, mock, mockDB := newMockFeatureFlagRepository(t)
		defer mockDB.Close()

		flagID := uuid.New()
		userID := uuid.New()
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "version",
			"key", "name", "description", "type", "status",
			"default_value", "rules", "tags",
			"created_by", "updated_by",
		}).AddRow(
			flagID, now, now, 1,
			"test_flag", "Test Flag", "Description", "boolean", "enabled",
			`{"enabled":true}`, `[]`, `["test"]`,
			userID, userID,
		)

		mock.ExpectQuery(`SELECT \* FROM "feature_flags" WHERE key = \$1`).
			WithArgs("test_flag", 1).
			WillReturnRows(rows)

		flag, err := repo.FindByKey(context.Background(), "test_flag")

		assert.NoError(t, err)
		assert.NotNil(t, flag)
		assert.Equal(t, "test_flag", flag.Key)
		assert.Equal(t, featureflag.FlagStatusEnabled, flag.Status)
		assert.True(t, flag.DefaultValue.Enabled)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error for non-existent flag", func(t *testing.T) {
		repo, mock, mockDB := newMockFeatureFlagRepository(t)
		defer mockDB.Close()

		mock.ExpectQuery(`SELECT \* FROM "feature_flags" WHERE key = \$1`).
			WithArgs("non_existent", 1).
			WillReturnError(gorm.ErrRecordNotFound)

		flag, err := repo.FindByKey(context.Background(), "non_existent")

		assert.Error(t, err)
		assert.Nil(t, flag)
		assert.Equal(t, shared.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormFeatureFlagRepository_FindByID(t *testing.T) {
	t.Run("finds existing feature flag by ID", func(t *testing.T) {
		repo, mock, mockDB := newMockFeatureFlagRepository(t)
		defer mockDB.Close()

		flagID := uuid.New()
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "version",
			"key", "name", "description", "type", "status",
			"default_value", "rules", "tags",
			"created_by", "updated_by",
		}).AddRow(
			flagID, now, now, 1,
			"my_flag", "My Flag", "", "boolean", "disabled",
			`{"enabled":false}`, `[]`, `[]`,
			nil, nil,
		)

		mock.ExpectQuery(`SELECT \* FROM "feature_flags" WHERE id = \$1`).
			WithArgs(flagID, 1).
			WillReturnRows(rows)

		flag, err := repo.FindByID(context.Background(), flagID)

		assert.NoError(t, err)
		assert.NotNil(t, flag)
		assert.Equal(t, flagID, flag.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormFeatureFlagRepository_Delete(t *testing.T) {
	t.Run("deletes existing flag", func(t *testing.T) {
		repo, mock, mockDB := newMockFeatureFlagRepository(t)
		defer mockDB.Close()

		mock.ExpectExec(`DELETE FROM "feature_flags" WHERE key = \$1`).
			WithArgs("test_flag").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Delete(context.Background(), "test_flag")

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error for non-existent flag", func(t *testing.T) {
		repo, mock, mockDB := newMockFeatureFlagRepository(t)
		defer mockDB.Close()

		mock.ExpectExec(`DELETE FROM "feature_flags" WHERE key = \$1`).
			WithArgs("non_existent").
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Delete(context.Background(), "non_existent")

		assert.Error(t, err)
		assert.Equal(t, shared.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormFeatureFlagRepository_ExistsByKey(t *testing.T) {
	t.Run("returns true for existing flag", func(t *testing.T) {
		repo, mock, mockDB := newMockFeatureFlagRepository(t)
		defer mockDB.Close()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "feature_flags" WHERE key = \$1`).
			WithArgs("test_flag").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		exists, err := repo.ExistsByKey(context.Background(), "test_flag")

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns false for non-existent flag", func(t *testing.T) {
		repo, mock, mockDB := newMockFeatureFlagRepository(t)
		defer mockDB.Close()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "feature_flags" WHERE key = \$1`).
			WithArgs("non_existent").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		exists, err := repo.ExistsByKey(context.Background(), "non_existent")

		assert.NoError(t, err)
		assert.False(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormFeatureFlagRepository_CountByStatus(t *testing.T) {
	t.Run("counts flags by status", func(t *testing.T) {
		repo, mock, mockDB := newMockFeatureFlagRepository(t)
		defer mockDB.Close()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "feature_flags" WHERE status = \$1`).
			WithArgs(featureflag.FlagStatusEnabled).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		count, err := repo.CountByStatus(context.Background(), featureflag.FlagStatusEnabled)

		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ======================================================================
// FlagOverrideRepository Tests
// ======================================================================

// newMockFlagOverrideRepository creates a GormFlagOverrideRepository with a mocked SQL connection
func newMockFlagOverrideRepository(t *testing.T) (*GormFlagOverrideRepository, sqlmock.Sqlmock, *sql.DB) {
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

	return NewGormFlagOverrideRepository(gormDB), mock, mockDB
}

func TestNewGormFlagOverrideRepository(t *testing.T) {
	t.Run("creates repository with valid DB", func(t *testing.T) {
		repo, _, mockDB := newMockFlagOverrideRepository(t)
		defer mockDB.Close()

		assert.NotNil(t, repo)
		assert.NotNil(t, repo.db)
	})
}

func TestGormFlagOverrideRepository_FindByID(t *testing.T) {
	t.Run("finds existing override", func(t *testing.T) {
		repo, mock, mockDB := newMockFlagOverrideRepository(t)
		defer mockDB.Close()

		overrideID := uuid.New()
		targetID := uuid.New()
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "created_at", "updated_at",
			"flag_key", "target_type", "target_id",
			"value", "reason", "expires_at", "created_by",
		}).AddRow(
			overrideID, now, now,
			"test_flag", "user", targetID,
			`{"enabled":true}`, "Testing", nil, nil,
		)

		mock.ExpectQuery(`SELECT \* FROM "flag_overrides" WHERE id = \$1`).
			WithArgs(overrideID, 1).
			WillReturnRows(rows)

		override, err := repo.FindByID(context.Background(), overrideID)

		assert.NoError(t, err)
		assert.NotNil(t, override)
		assert.Equal(t, "test_flag", override.FlagKey)
		assert.Equal(t, featureflag.OverrideTargetTypeUser, override.TargetType)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error for non-existent override", func(t *testing.T) {
		repo, mock, mockDB := newMockFlagOverrideRepository(t)
		defer mockDB.Close()

		overrideID := uuid.New()

		mock.ExpectQuery(`SELECT \* FROM "flag_overrides" WHERE id = \$1`).
			WithArgs(overrideID, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		override, err := repo.FindByID(context.Background(), overrideID)

		assert.Error(t, err)
		assert.Nil(t, override)
		assert.Equal(t, shared.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormFlagOverrideRepository_Delete(t *testing.T) {
	t.Run("deletes existing override", func(t *testing.T) {
		repo, mock, mockDB := newMockFlagOverrideRepository(t)
		defer mockDB.Close()

		overrideID := uuid.New()

		mock.ExpectExec(`DELETE FROM "flag_overrides" WHERE id = \$1`).
			WithArgs(overrideID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Delete(context.Background(), overrideID)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error for non-existent override", func(t *testing.T) {
		repo, mock, mockDB := newMockFlagOverrideRepository(t)
		defer mockDB.Close()

		overrideID := uuid.New()

		mock.ExpectExec(`DELETE FROM "flag_overrides" WHERE id = \$1`).
			WithArgs(overrideID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Delete(context.Background(), overrideID)

		assert.Error(t, err)
		assert.Equal(t, shared.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormFlagOverrideRepository_DeleteByFlagKey(t *testing.T) {
	t.Run("deletes all overrides for a flag", func(t *testing.T) {
		repo, mock, mockDB := newMockFlagOverrideRepository(t)
		defer mockDB.Close()

		mock.ExpectExec(`DELETE FROM "flag_overrides" WHERE flag_key = \$1`).
			WithArgs("test_flag").
			WillReturnResult(sqlmock.NewResult(0, 3))

		count, err := repo.DeleteByFlagKey(context.Background(), "test_flag")

		assert.NoError(t, err)
		assert.Equal(t, int64(3), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormFlagOverrideRepository_CountByFlagKey(t *testing.T) {
	t.Run("counts overrides for a flag", func(t *testing.T) {
		repo, mock, mockDB := newMockFlagOverrideRepository(t)
		defer mockDB.Close()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "flag_overrides" WHERE flag_key = \$1`).
			WithArgs("test_flag").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(7))

		count, err := repo.CountByFlagKey(context.Background(), "test_flag")

		assert.NoError(t, err)
		assert.Equal(t, int64(7), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ======================================================================
// FlagAuditLogRepository Tests
// ======================================================================

// newMockFlagAuditLogRepository creates a GormFlagAuditLogRepository with a mocked SQL connection
func newMockFlagAuditLogRepository(t *testing.T) (*GormFlagAuditLogRepository, sqlmock.Sqlmock, *sql.DB) {
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

	return NewGormFlagAuditLogRepository(gormDB), mock, mockDB
}

func TestNewGormFlagAuditLogRepository(t *testing.T) {
	t.Run("creates repository with valid DB", func(t *testing.T) {
		repo, _, mockDB := newMockFlagAuditLogRepository(t)
		defer mockDB.Close()

		assert.NotNil(t, repo)
		assert.NotNil(t, repo.db)
	})
}

func TestGormFlagAuditLogRepository_CreateBatch(t *testing.T) {
	t.Run("handles empty batch", func(t *testing.T) {
		repo, _, mockDB := newMockFlagAuditLogRepository(t)
		defer mockDB.Close()

		err := repo.CreateBatch(context.Background(), []*featureflag.FlagAuditLog{})

		assert.NoError(t, err)
	})
}

func TestGormFlagAuditLogRepository_CountByFlagKey(t *testing.T) {
	t.Run("counts audit logs for a flag", func(t *testing.T) {
		repo, mock, mockDB := newMockFlagAuditLogRepository(t)
		defer mockDB.Close()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "flag_audit_logs" WHERE flag_key = \$1`).
			WithArgs("test_flag").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(15))

		count, err := repo.CountByFlagKey(context.Background(), "test_flag")

		assert.NoError(t, err)
		assert.Equal(t, int64(15), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ======================================================================
// Sort Field Tests
// ======================================================================

func TestFeatureFlagSortFields(t *testing.T) {
	t.Run("contains expected fields", func(t *testing.T) {
		expectedFields := []string{"id", "created_at", "updated_at", "key", "name", "type", "status"}
		for _, field := range expectedFields {
			assert.True(t, FeatureFlagSortFields[field], "Expected field %s to be in FeatureFlagSortFields", field)
		}
	})

	t.Run("does not contain invalid fields", func(t *testing.T) {
		assert.False(t, FeatureFlagSortFields["invalid_field"])
		assert.False(t, FeatureFlagSortFields["password"])
	})
}

func TestFlagOverrideSortFields(t *testing.T) {
	t.Run("contains expected fields", func(t *testing.T) {
		expectedFields := []string{"id", "created_at", "updated_at", "flag_key", "target_type", "expires_at"}
		for _, field := range expectedFields {
			assert.True(t, FlagOverrideSortFields[field], "Expected field %s to be in FlagOverrideSortFields", field)
		}
	})
}

func TestFlagAuditLogSortFields(t *testing.T) {
	t.Run("contains expected fields", func(t *testing.T) {
		expectedFields := []string{"id", "created_at", "flag_key", "action"}
		for _, field := range expectedFields {
			assert.True(t, FlagAuditLogSortFields[field], "Expected field %s to be in FlagAuditLogSortFields", field)
		}
	})
}

func TestGormFeatureFlagRepository_FindByTags(t *testing.T) {
	t.Run("returns empty slice for empty tags", func(t *testing.T) {
		repo, _, mockDB := newMockFeatureFlagRepository(t)
		defer mockDB.Close()

		flags, err := repo.FindByTags(context.Background(), []string{}, shared.Filter{})

		assert.NoError(t, err)
		assert.Empty(t, flags)
	})

	t.Run("returns error for too many tags", func(t *testing.T) {
		repo, _, mockDB := newMockFeatureFlagRepository(t)
		defer mockDB.Close()

		// Create 51 tags (exceeds MaxFindByTagsLimit of 50)
		tags := make([]string, 51)
		for i := range 51 {
			tags[i] = "tag" + string(rune('a'+i%26))
		}

		_, err := repo.FindByTags(context.Background(), tags, shared.Filter{})

		assert.Error(t, err)
		// Check that error message mentions the limit
		assert.Contains(t, err.Error(), "50 tags")
	})
}

func TestMaxFindByTagsLimit(t *testing.T) {
	t.Run("constant is set to 50", func(t *testing.T) {
		assert.Equal(t, 50, MaxFindByTagsLimit)
	})
}
