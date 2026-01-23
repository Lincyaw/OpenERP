package persistence

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// newMockWarehouseRepository creates a GormWarehouseRepository with a mocked SQL connection
func newMockWarehouseRepository(t *testing.T) (*GormWarehouseRepository, sqlmock.Sqlmock, *sql.DB) {
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

	return NewGormWarehouseRepository(gormDB), mock, mockDB
}

func TestNewGormWarehouseRepository(t *testing.T) {
	t.Run("creates repository with valid DB", func(t *testing.T) {
		repo, _, mockDB := newMockWarehouseRepository(t)
		defer mockDB.Close()

		assert.NotNil(t, repo)
		assert.NotNil(t, repo.db)
	})
}

func TestGormWarehouseRepository_FindByID(t *testing.T) {
	t.Run("finds existing warehouse", func(t *testing.T) {
		repo, mock, mockDB := newMockWarehouseRepository(t)
		defer mockDB.Close()

		warehouseID := uuid.New()
		tenantID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "tenant_id", "code", "name", "type", "status", "is_default", "capacity"}).
			AddRow(warehouseID, tenantID, "WH001", "Main Warehouse", "physical", "active", true, 1000)

		mock.ExpectQuery(`SELECT \* FROM "warehouses" WHERE id = \$1 ORDER BY .* LIMIT .*`).
			WithArgs(warehouseID, 1).
			WillReturnRows(rows)

		warehouse, err := repo.FindByID(context.Background(), warehouseID)

		assert.NoError(t, err)
		assert.NotNil(t, warehouse)
		assert.Equal(t, warehouseID, warehouse.ID)
		assert.Equal(t, "WH001", warehouse.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error for non-existent warehouse", func(t *testing.T) {
		repo, mock, mockDB := newMockWarehouseRepository(t)
		defer mockDB.Close()

		warehouseID := uuid.New()

		mock.ExpectQuery(`SELECT \* FROM "warehouses" WHERE id = \$1 ORDER BY .* LIMIT .*`).
			WithArgs(warehouseID, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		warehouse, err := repo.FindByID(context.Background(), warehouseID)

		assert.Error(t, err)
		assert.Nil(t, warehouse)
		assert.Equal(t, shared.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormWarehouseRepository_FindByIDForTenant(t *testing.T) {
	t.Run("finds warehouse within tenant", func(t *testing.T) {
		repo, mock, mockDB := newMockWarehouseRepository(t)
		defer mockDB.Close()

		warehouseID := uuid.New()
		tenantID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "tenant_id", "code", "name", "type", "status", "is_default", "capacity"}).
			AddRow(warehouseID, tenantID, "WH001", "Main Warehouse", "physical", "active", true, 1000)

		mock.ExpectQuery(`SELECT \* FROM "warehouses" WHERE tenant_id = \$1 AND id = \$2 ORDER BY .* LIMIT .*`).
			WithArgs(tenantID, warehouseID, 1).
			WillReturnRows(rows)

		warehouse, err := repo.FindByIDForTenant(context.Background(), tenantID, warehouseID)

		assert.NoError(t, err)
		assert.NotNil(t, warehouse)
		assert.Equal(t, tenantID, warehouse.TenantID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormWarehouseRepository_FindByCode(t *testing.T) {
	t.Run("finds warehouse by code", func(t *testing.T) {
		repo, mock, mockDB := newMockWarehouseRepository(t)
		defer mockDB.Close()

		warehouseID := uuid.New()
		tenantID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "tenant_id", "code", "name", "type", "status", "is_default", "capacity"}).
			AddRow(warehouseID, tenantID, "WH001", "Main Warehouse", "physical", "active", true, 1000)

		mock.ExpectQuery(`SELECT \* FROM "warehouses" WHERE tenant_id = \$1 AND code = \$2 ORDER BY .* LIMIT .*`).
			WithArgs(tenantID, "WH001", 1).
			WillReturnRows(rows)

		warehouse, err := repo.FindByCode(context.Background(), tenantID, "wh001") // lowercase to test uppercasing

		assert.NoError(t, err)
		assert.NotNil(t, warehouse)
		assert.Equal(t, "WH001", warehouse.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error for non-existent code", func(t *testing.T) {
		repo, mock, mockDB := newMockWarehouseRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()

		mock.ExpectQuery(`SELECT \* FROM "warehouses" WHERE tenant_id = \$1 AND code = \$2 ORDER BY .* LIMIT .*`).
			WithArgs(tenantID, "NONEXISTENT", 1).
			WillReturnError(gorm.ErrRecordNotFound)

		warehouse, err := repo.FindByCode(context.Background(), tenantID, "nonexistent")

		assert.Error(t, err)
		assert.Nil(t, warehouse)
		assert.Equal(t, shared.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormWarehouseRepository_FindDefault(t *testing.T) {
	t.Run("finds default warehouse", func(t *testing.T) {
		repo, mock, mockDB := newMockWarehouseRepository(t)
		defer mockDB.Close()

		warehouseID := uuid.New()
		tenantID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "tenant_id", "code", "name", "type", "status", "is_default", "capacity"}).
			AddRow(warehouseID, tenantID, "WH001", "Main Warehouse", "physical", "active", true, 1000)

		mock.ExpectQuery(`SELECT \* FROM "warehouses" WHERE tenant_id = \$1 AND is_default = \$2 ORDER BY .* LIMIT .*`).
			WithArgs(tenantID, true, 1).
			WillReturnRows(rows)

		warehouse, err := repo.FindDefault(context.Background(), tenantID)

		assert.NoError(t, err)
		assert.NotNil(t, warehouse)
		assert.True(t, warehouse.IsDefault)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when no default warehouse", func(t *testing.T) {
		repo, mock, mockDB := newMockWarehouseRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()

		mock.ExpectQuery(`SELECT \* FROM "warehouses" WHERE tenant_id = \$1 AND is_default = \$2 ORDER BY .* LIMIT .*`).
			WithArgs(tenantID, true, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		warehouse, err := repo.FindDefault(context.Background(), tenantID)

		assert.Error(t, err)
		assert.Nil(t, warehouse)
		assert.Equal(t, shared.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormWarehouseRepository_FindByIDs(t *testing.T) {
	t.Run("finds multiple warehouses by IDs", func(t *testing.T) {
		repo, mock, mockDB := newMockWarehouseRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		id1 := uuid.New()
		id2 := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "tenant_id", "code", "name", "type", "status", "is_default", "capacity"}).
			AddRow(id1, tenantID, "WH001", "Warehouse 1", "physical", "active", true, 1000).
			AddRow(id2, tenantID, "WH002", "Warehouse 2", "virtual", "active", false, 0)

		mock.ExpectQuery(`SELECT \* FROM "warehouses" WHERE tenant_id = \$1 AND id IN \(\$2,\$3\)`).
			WithArgs(tenantID, id1, id2).
			WillReturnRows(rows)

		warehouses, err := repo.FindByIDs(context.Background(), tenantID, []uuid.UUID{id1, id2})

		assert.NoError(t, err)
		assert.Len(t, warehouses, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty slice for empty IDs", func(t *testing.T) {
		repo, _, mockDB := newMockWarehouseRepository(t)
		defer mockDB.Close()

		warehouses, err := repo.FindByIDs(context.Background(), uuid.New(), []uuid.UUID{})

		assert.NoError(t, err)
		assert.Empty(t, warehouses)
	})
}

func TestGormWarehouseRepository_FindByCodes(t *testing.T) {
	t.Run("returns empty slice for empty codes", func(t *testing.T) {
		repo, _, mockDB := newMockWarehouseRepository(t)
		defer mockDB.Close()

		warehouses, err := repo.FindByCodes(context.Background(), uuid.New(), []string{})

		assert.NoError(t, err)
		assert.Empty(t, warehouses)
	})
}

func TestGormWarehouseRepository_Save(t *testing.T) {
	t.Run("saves warehouse", func(t *testing.T) {
		repo, mock, mockDB := newMockWarehouseRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		warehouse, _ := partner.NewPhysicalWarehouse(tenantID, "WH001", "Test Warehouse")

		mock.ExpectExec(`UPDATE "warehouses" SET`).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Save(context.Background(), warehouse)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormWarehouseRepository_SaveBatch(t *testing.T) {
	t.Run("returns nil for empty batch", func(t *testing.T) {
		repo, _, mockDB := newMockWarehouseRepository(t)
		defer mockDB.Close()

		err := repo.SaveBatch(context.Background(), []*partner.Warehouse{})

		assert.NoError(t, err)
	})
}

func TestGormWarehouseRepository_Delete(t *testing.T) {
	t.Run("deletes existing warehouse", func(t *testing.T) {
		repo, mock, mockDB := newMockWarehouseRepository(t)
		defer mockDB.Close()

		warehouseID := uuid.New()

		mock.ExpectExec(`DELETE FROM "warehouses" WHERE id = \$1`).
			WithArgs(warehouseID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Delete(context.Background(), warehouseID)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error for non-existent warehouse", func(t *testing.T) {
		repo, mock, mockDB := newMockWarehouseRepository(t)
		defer mockDB.Close()

		warehouseID := uuid.New()

		mock.ExpectExec(`DELETE FROM "warehouses" WHERE id = \$1`).
			WithArgs(warehouseID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Delete(context.Background(), warehouseID)

		assert.Error(t, err)
		assert.Equal(t, shared.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormWarehouseRepository_DeleteForTenant(t *testing.T) {
	t.Run("deletes warehouse within tenant", func(t *testing.T) {
		repo, mock, mockDB := newMockWarehouseRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		warehouseID := uuid.New()

		mock.ExpectExec(`DELETE FROM "warehouses" WHERE tenant_id = \$1 AND id = \$2`).
			WithArgs(tenantID, warehouseID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.DeleteForTenant(context.Background(), tenantID, warehouseID)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormWarehouseRepository_Count(t *testing.T) {
	t.Run("counts warehouses", func(t *testing.T) {
		repo, mock, mockDB := newMockWarehouseRepository(t)
		defer mockDB.Close()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "warehouses"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(8))

		count, err := repo.Count(context.Background(), shared.Filter{})

		assert.NoError(t, err)
		assert.Equal(t, int64(8), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormWarehouseRepository_CountForTenant(t *testing.T) {
	t.Run("counts warehouses for tenant", func(t *testing.T) {
		repo, mock, mockDB := newMockWarehouseRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "warehouses" WHERE tenant_id = \$1`).
			WithArgs(tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

		count, err := repo.CountForTenant(context.Background(), tenantID, shared.Filter{})

		assert.NoError(t, err)
		assert.Equal(t, int64(3), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormWarehouseRepository_CountByType(t *testing.T) {
	t.Run("counts warehouses by type", func(t *testing.T) {
		repo, mock, mockDB := newMockWarehouseRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "warehouses" WHERE tenant_id = \$1 AND type = \$2`).
			WithArgs(tenantID, partner.WarehouseTypePhysical).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

		count, err := repo.CountByType(context.Background(), tenantID, partner.WarehouseTypePhysical)

		assert.NoError(t, err)
		assert.Equal(t, int64(2), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormWarehouseRepository_CountByStatus(t *testing.T) {
	t.Run("counts warehouses by status", func(t *testing.T) {
		repo, mock, mockDB := newMockWarehouseRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "warehouses" WHERE tenant_id = \$1 AND status = \$2`).
			WithArgs(tenantID, partner.WarehouseStatusActive).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		count, err := repo.CountByStatus(context.Background(), tenantID, partner.WarehouseStatusActive)

		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormWarehouseRepository_ExistsByCode(t *testing.T) {
	t.Run("returns true when warehouse exists", func(t *testing.T) {
		repo, mock, mockDB := newMockWarehouseRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "warehouses" WHERE tenant_id = \$1 AND code = \$2`).
			WithArgs(tenantID, "WH001").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		exists, err := repo.ExistsByCode(context.Background(), tenantID, "wh001")

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns false when warehouse does not exist", func(t *testing.T) {
		repo, mock, mockDB := newMockWarehouseRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "warehouses" WHERE tenant_id = \$1 AND code = \$2`).
			WithArgs(tenantID, "NONEXISTENT").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		exists, err := repo.ExistsByCode(context.Background(), tenantID, "nonexistent")

		assert.NoError(t, err)
		assert.False(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormWarehouseRepository_ClearDefault(t *testing.T) {
	t.Run("clears default flag for all warehouses in tenant", func(t *testing.T) {
		repo, mock, mockDB := newMockWarehouseRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()

		mock.ExpectExec(`UPDATE "warehouses" SET "is_default"=.*,"updated_at"=.* WHERE tenant_id = \$. AND is_default = \$.`).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.ClearDefault(context.Background(), tenantID)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("succeeds even when no default warehouse exists", func(t *testing.T) {
		repo, mock, mockDB := newMockWarehouseRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()

		mock.ExpectExec(`UPDATE "warehouses" SET "is_default"=.*,"updated_at"=.* WHERE tenant_id = \$. AND is_default = \$.`).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.ClearDefault(context.Background(), tenantID)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormWarehouseRepository_InterfaceCompliance(t *testing.T) {
	t.Run("implements WarehouseRepository interface", func(t *testing.T) {
		repo, _, mockDB := newMockWarehouseRepository(t)
		defer mockDB.Close()

		var _ partner.WarehouseRepository = repo
	})
}
