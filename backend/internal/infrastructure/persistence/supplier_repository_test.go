package persistence

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// newMockSupplierRepository creates a GormSupplierRepository with a mocked SQL connection
func newMockSupplierRepository(t *testing.T) (*GormSupplierRepository, sqlmock.Sqlmock, *sql.DB) {
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

	return NewGormSupplierRepository(gormDB), mock, mockDB
}

func TestNewGormSupplierRepository(t *testing.T) {
	t.Run("creates repository with valid DB", func(t *testing.T) {
		repo, _, mockDB := newMockSupplierRepository(t)
		defer mockDB.Close()

		assert.NotNil(t, repo)
		assert.NotNil(t, repo.db)
	})
}

func TestGormSupplierRepository_FindByID(t *testing.T) {
	t.Run("finds existing supplier", func(t *testing.T) {
		repo, mock, mockDB := newMockSupplierRepository(t)
		defer mockDB.Close()

		supplierID := uuid.New()
		tenantID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "tenant_id", "code", "name", "type", "status", "balance", "credit_limit", "credit_days", "rating"}).
			AddRow(supplierID, tenantID, "SUP001", "Test Supplier", "distributor", "active", decimal.Zero, decimal.Zero, 30, 4)

		mock.ExpectQuery(`SELECT \* FROM "suppliers" WHERE id = \$1 ORDER BY .* LIMIT .*`).
			WithArgs(supplierID, 1).
			WillReturnRows(rows)

		supplier, err := repo.FindByID(context.Background(), supplierID)

		assert.NoError(t, err)
		assert.NotNil(t, supplier)
		assert.Equal(t, supplierID, supplier.ID)
		assert.Equal(t, "SUP001", supplier.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error for non-existent supplier", func(t *testing.T) {
		repo, mock, mockDB := newMockSupplierRepository(t)
		defer mockDB.Close()

		supplierID := uuid.New()

		mock.ExpectQuery(`SELECT \* FROM "suppliers" WHERE id = \$1 ORDER BY .* LIMIT .*`).
			WithArgs(supplierID, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		supplier, err := repo.FindByID(context.Background(), supplierID)

		assert.Error(t, err)
		assert.Nil(t, supplier)
		assert.Equal(t, shared.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormSupplierRepository_FindByIDForTenant(t *testing.T) {
	t.Run("finds supplier within tenant", func(t *testing.T) {
		repo, mock, mockDB := newMockSupplierRepository(t)
		defer mockDB.Close()

		supplierID := uuid.New()
		tenantID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "tenant_id", "code", "name", "type", "status", "balance", "credit_limit", "credit_days", "rating"}).
			AddRow(supplierID, tenantID, "SUP001", "Test Supplier", "distributor", "active", decimal.Zero, decimal.Zero, 30, 4)

		mock.ExpectQuery(`SELECT \* FROM "suppliers" WHERE tenant_id = \$1 AND id = \$2 ORDER BY .* LIMIT .*`).
			WithArgs(tenantID, supplierID, 1).
			WillReturnRows(rows)

		supplier, err := repo.FindByIDForTenant(context.Background(), tenantID, supplierID)

		assert.NoError(t, err)
		assert.NotNil(t, supplier)
		assert.Equal(t, tenantID, supplier.TenantID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormSupplierRepository_FindByCode(t *testing.T) {
	t.Run("finds supplier by code", func(t *testing.T) {
		repo, mock, mockDB := newMockSupplierRepository(t)
		defer mockDB.Close()

		supplierID := uuid.New()
		tenantID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "tenant_id", "code", "name", "type", "status", "balance", "credit_limit", "credit_days", "rating"}).
			AddRow(supplierID, tenantID, "SUP001", "Test Supplier", "distributor", "active", decimal.Zero, decimal.Zero, 30, 4)

		mock.ExpectQuery(`SELECT \* FROM "suppliers" WHERE tenant_id = \$1 AND code = \$2 ORDER BY .* LIMIT .*`).
			WithArgs(tenantID, "SUP001", 1).
			WillReturnRows(rows)

		supplier, err := repo.FindByCode(context.Background(), tenantID, "sup001") // lowercase to test uppercasing

		assert.NoError(t, err)
		assert.NotNil(t, supplier)
		assert.Equal(t, "SUP001", supplier.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormSupplierRepository_FindByPhone(t *testing.T) {
	t.Run("finds supplier by phone", func(t *testing.T) {
		repo, mock, mockDB := newMockSupplierRepository(t)
		defer mockDB.Close()

		supplierID := uuid.New()
		tenantID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "tenant_id", "code", "name", "phone", "type", "status", "balance", "credit_limit", "credit_days", "rating"}).
			AddRow(supplierID, tenantID, "SUP001", "Test Supplier", "13900139000", "distributor", "active", decimal.Zero, decimal.Zero, 30, 4)

		mock.ExpectQuery(`SELECT \* FROM "suppliers" WHERE tenant_id = \$1 AND phone = \$2 ORDER BY .* LIMIT .*`).
			WithArgs(tenantID, "13900139000", 1).
			WillReturnRows(rows)

		supplier, err := repo.FindByPhone(context.Background(), tenantID, "13900139000")

		assert.NoError(t, err)
		assert.NotNil(t, supplier)
		assert.Equal(t, "13900139000", supplier.Phone)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error for empty phone", func(t *testing.T) {
		repo, _, mockDB := newMockSupplierRepository(t)
		defer mockDB.Close()

		_, err := repo.FindByPhone(context.Background(), uuid.New(), "")

		assert.Error(t, err)
	})
}

func TestGormSupplierRepository_FindByEmail(t *testing.T) {
	t.Run("finds supplier by email", func(t *testing.T) {
		repo, mock, mockDB := newMockSupplierRepository(t)
		defer mockDB.Close()

		supplierID := uuid.New()
		tenantID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "tenant_id", "code", "name", "email", "type", "status", "balance", "credit_limit", "credit_days", "rating"}).
			AddRow(supplierID, tenantID, "SUP001", "Test Supplier", "supplier@example.com", "distributor", "active", decimal.Zero, decimal.Zero, 30, 4)

		mock.ExpectQuery(`SELECT \* FROM "suppliers" WHERE tenant_id = \$1 AND email = \$2 ORDER BY .* LIMIT .*`).
			WithArgs(tenantID, "supplier@example.com", 1).
			WillReturnRows(rows)

		supplier, err := repo.FindByEmail(context.Background(), tenantID, "supplier@example.com")

		assert.NoError(t, err)
		assert.NotNil(t, supplier)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error for empty email", func(t *testing.T) {
		repo, _, mockDB := newMockSupplierRepository(t)
		defer mockDB.Close()

		_, err := repo.FindByEmail(context.Background(), uuid.New(), "")

		assert.Error(t, err)
	})
}

func TestGormSupplierRepository_FindByIDs(t *testing.T) {
	t.Run("finds multiple suppliers by IDs", func(t *testing.T) {
		repo, mock, mockDB := newMockSupplierRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		id1 := uuid.New()
		id2 := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "tenant_id", "code", "name", "type", "status", "balance", "credit_limit", "credit_days", "rating"}).
			AddRow(id1, tenantID, "SUP001", "Supplier 1", "manufacturer", "active", decimal.Zero, decimal.Zero, 30, 5).
			AddRow(id2, tenantID, "SUP002", "Supplier 2", "distributor", "active", decimal.Zero, decimal.Zero, 45, 4)

		mock.ExpectQuery(`SELECT \* FROM "suppliers" WHERE tenant_id = \$1 AND id IN \(\$2,\$3\)`).
			WithArgs(tenantID, id1, id2).
			WillReturnRows(rows)

		suppliers, err := repo.FindByIDs(context.Background(), tenantID, []uuid.UUID{id1, id2})

		assert.NoError(t, err)
		assert.Len(t, suppliers, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty slice for empty IDs", func(t *testing.T) {
		repo, _, mockDB := newMockSupplierRepository(t)
		defer mockDB.Close()

		suppliers, err := repo.FindByIDs(context.Background(), uuid.New(), []uuid.UUID{})

		assert.NoError(t, err)
		assert.Empty(t, suppliers)
	})
}

func TestGormSupplierRepository_FindByCodes(t *testing.T) {
	t.Run("returns empty slice for empty codes", func(t *testing.T) {
		repo, _, mockDB := newMockSupplierRepository(t)
		defer mockDB.Close()

		suppliers, err := repo.FindByCodes(context.Background(), uuid.New(), []string{})

		assert.NoError(t, err)
		assert.Empty(t, suppliers)
	})
}

func TestGormSupplierRepository_Save(t *testing.T) {
	t.Run("saves supplier", func(t *testing.T) {
		repo, mock, mockDB := newMockSupplierRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		supplier, _ := partner.NewDistributorSupplier(tenantID, "SUP001", "Test Supplier")

		mock.ExpectExec(`UPDATE "suppliers" SET`).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Save(context.Background(), supplier)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormSupplierRepository_SaveBatch(t *testing.T) {
	t.Run("returns nil for empty batch", func(t *testing.T) {
		repo, _, mockDB := newMockSupplierRepository(t)
		defer mockDB.Close()

		err := repo.SaveBatch(context.Background(), []*partner.Supplier{})

		assert.NoError(t, err)
	})
}

func TestGormSupplierRepository_Delete(t *testing.T) {
	t.Run("deletes existing supplier", func(t *testing.T) {
		repo, mock, mockDB := newMockSupplierRepository(t)
		defer mockDB.Close()

		supplierID := uuid.New()

		mock.ExpectExec(`DELETE FROM "suppliers" WHERE id = \$1`).
			WithArgs(supplierID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Delete(context.Background(), supplierID)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error for non-existent supplier", func(t *testing.T) {
		repo, mock, mockDB := newMockSupplierRepository(t)
		defer mockDB.Close()

		supplierID := uuid.New()

		mock.ExpectExec(`DELETE FROM "suppliers" WHERE id = \$1`).
			WithArgs(supplierID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Delete(context.Background(), supplierID)

		assert.Error(t, err)
		assert.Equal(t, shared.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormSupplierRepository_DeleteForTenant(t *testing.T) {
	t.Run("deletes supplier within tenant", func(t *testing.T) {
		repo, mock, mockDB := newMockSupplierRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		supplierID := uuid.New()

		mock.ExpectExec(`DELETE FROM "suppliers" WHERE tenant_id = \$1 AND id = \$2`).
			WithArgs(tenantID, supplierID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.DeleteForTenant(context.Background(), tenantID, supplierID)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormSupplierRepository_Count(t *testing.T) {
	t.Run("counts suppliers", func(t *testing.T) {
		repo, mock, mockDB := newMockSupplierRepository(t)
		defer mockDB.Close()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "suppliers"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(15))

		count, err := repo.Count(context.Background(), shared.Filter{})

		assert.NoError(t, err)
		assert.Equal(t, int64(15), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormSupplierRepository_CountForTenant(t *testing.T) {
	t.Run("counts suppliers for tenant", func(t *testing.T) {
		repo, mock, mockDB := newMockSupplierRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "suppliers" WHERE tenant_id = \$1`).
			WithArgs(tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(7))

		count, err := repo.CountForTenant(context.Background(), tenantID, shared.Filter{})

		assert.NoError(t, err)
		assert.Equal(t, int64(7), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormSupplierRepository_CountByType(t *testing.T) {
	t.Run("counts suppliers by type", func(t *testing.T) {
		repo, mock, mockDB := newMockSupplierRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "suppliers" WHERE tenant_id = \$1 AND type = \$2`).
			WithArgs(tenantID, partner.SupplierTypeManufacturer).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))

		count, err := repo.CountByType(context.Background(), tenantID, partner.SupplierTypeManufacturer)

		assert.NoError(t, err)
		assert.Equal(t, int64(4), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormSupplierRepository_CountByStatus(t *testing.T) {
	t.Run("counts suppliers by status", func(t *testing.T) {
		repo, mock, mockDB := newMockSupplierRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "suppliers" WHERE tenant_id = \$1 AND status = \$2`).
			WithArgs(tenantID, partner.SupplierStatusActive).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(12))

		count, err := repo.CountByStatus(context.Background(), tenantID, partner.SupplierStatusActive)

		assert.NoError(t, err)
		assert.Equal(t, int64(12), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormSupplierRepository_ExistsByCode(t *testing.T) {
	t.Run("returns true when supplier exists", func(t *testing.T) {
		repo, mock, mockDB := newMockSupplierRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "suppliers" WHERE tenant_id = \$1 AND code = \$2`).
			WithArgs(tenantID, "SUP001").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		exists, err := repo.ExistsByCode(context.Background(), tenantID, "sup001")

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns false when supplier does not exist", func(t *testing.T) {
		repo, mock, mockDB := newMockSupplierRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "suppliers" WHERE tenant_id = \$1 AND code = \$2`).
			WithArgs(tenantID, "NONEXISTENT").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		exists, err := repo.ExistsByCode(context.Background(), tenantID, "nonexistent")

		assert.NoError(t, err)
		assert.False(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormSupplierRepository_ExistsByPhone(t *testing.T) {
	t.Run("returns false for empty phone", func(t *testing.T) {
		repo, _, mockDB := newMockSupplierRepository(t)
		defer mockDB.Close()

		exists, err := repo.ExistsByPhone(context.Background(), uuid.New(), "")

		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestGormSupplierRepository_ExistsByEmail(t *testing.T) {
	t.Run("returns false for empty email", func(t *testing.T) {
		repo, _, mockDB := newMockSupplierRepository(t)
		defer mockDB.Close()

		exists, err := repo.ExistsByEmail(context.Background(), uuid.New(), "")

		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestGormSupplierRepository_InterfaceCompliance(t *testing.T) {
	t.Run("implements SupplierRepository interface", func(t *testing.T) {
		repo, _, mockDB := newMockSupplierRepository(t)
		defer mockDB.Close()

		var _ partner.SupplierRepository = repo
	})
}
