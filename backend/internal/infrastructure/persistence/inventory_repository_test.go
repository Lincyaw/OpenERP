package persistence

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// newMockInventoryItemRepository creates a GormInventoryItemRepository with a mocked SQL connection
func newMockInventoryItemRepository(t *testing.T) (*GormInventoryItemRepository, sqlmock.Sqlmock, *sql.DB) {
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

	return NewGormInventoryItemRepository(gormDB), mock, mockDB
}

func TestNewGormInventoryItemRepository(t *testing.T) {
	t.Run("creates repository with valid DB", func(t *testing.T) {
		repo, _, mockDB := newMockInventoryItemRepository(t)
		defer mockDB.Close()

		assert.NotNil(t, repo)
		assert.NotNil(t, repo.db)
	})
}

func TestGormInventoryItemRepository_FindByID(t *testing.T) {
	t.Run("finds existing inventory item", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryItemRepository(t)
		defer mockDB.Close()

		itemID := uuid.New()
		tenantID := uuid.New()
		warehouseID := uuid.New()
		productID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "warehouse_id", "product_id",
			"available_quantity", "locked_quantity", "unit_cost",
			"min_quantity", "max_quantity", "version",
		}).AddRow(
			itemID, tenantID, warehouseID, productID,
			decimal.NewFromInt(100), decimal.NewFromInt(10), decimal.NewFromFloat(15.50),
			decimal.NewFromInt(20), decimal.NewFromInt(500), 1,
		)

		mock.ExpectQuery(`SELECT \* FROM "inventory_items" WHERE id = \$1`).
			WithArgs(itemID, 1).
			WillReturnRows(rows)

		item, err := repo.FindByID(context.Background(), itemID)

		assert.NoError(t, err)
		assert.NotNil(t, item)
		assert.Equal(t, itemID, item.ID)
		assert.Equal(t, warehouseID, item.WarehouseID)
		assert.Equal(t, productID, item.ProductID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error for non-existent item", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryItemRepository(t)
		defer mockDB.Close()

		itemID := uuid.New()

		mock.ExpectQuery(`SELECT \* FROM "inventory_items" WHERE id = \$1`).
			WithArgs(itemID, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		item, err := repo.FindByID(context.Background(), itemID)

		assert.Error(t, err)
		assert.Nil(t, item)
		assert.Equal(t, shared.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormInventoryItemRepository_FindByIDForTenant(t *testing.T) {
	t.Run("finds inventory item within tenant", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryItemRepository(t)
		defer mockDB.Close()

		itemID := uuid.New()
		tenantID := uuid.New()
		warehouseID := uuid.New()
		productID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "warehouse_id", "product_id",
			"available_quantity", "locked_quantity", "unit_cost",
			"min_quantity", "max_quantity", "version",
		}).AddRow(
			itemID, tenantID, warehouseID, productID,
			decimal.NewFromInt(100), decimal.NewFromInt(0), decimal.NewFromFloat(10.00),
			decimal.Zero, decimal.Zero, 1,
		)

		mock.ExpectQuery(`SELECT \* FROM "inventory_items" WHERE tenant_id = \$1 AND id = \$2`).
			WithArgs(tenantID, itemID, 1).
			WillReturnRows(rows)

		item, err := repo.FindByIDForTenant(context.Background(), tenantID, itemID)

		assert.NoError(t, err)
		assert.NotNil(t, item)
		assert.Equal(t, tenantID, item.TenantID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormInventoryItemRepository_FindByWarehouseAndProduct(t *testing.T) {
	t.Run("finds inventory by warehouse-product combination", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryItemRepository(t)
		defer mockDB.Close()

		itemID := uuid.New()
		tenantID := uuid.New()
		warehouseID := uuid.New()
		productID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "warehouse_id", "product_id",
			"available_quantity", "locked_quantity", "unit_cost",
			"min_quantity", "max_quantity", "version",
		}).AddRow(
			itemID, tenantID, warehouseID, productID,
			decimal.NewFromInt(50), decimal.NewFromInt(5), decimal.NewFromFloat(25.00),
			decimal.NewFromInt(10), decimal.NewFromInt(200), 1,
		)

		mock.ExpectQuery(`SELECT \* FROM "inventory_items" WHERE tenant_id = \$1 AND warehouse_id = \$2 AND product_id = \$3`).
			WithArgs(tenantID, warehouseID, productID, 1).
			WillReturnRows(rows)

		item, err := repo.FindByWarehouseAndProduct(context.Background(), tenantID, warehouseID, productID)

		assert.NoError(t, err)
		assert.NotNil(t, item)
		assert.Equal(t, warehouseID, item.WarehouseID)
		assert.Equal(t, productID, item.ProductID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found for missing combination", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryItemRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		warehouseID := uuid.New()
		productID := uuid.New()

		mock.ExpectQuery(`SELECT \* FROM "inventory_items" WHERE tenant_id = \$1 AND warehouse_id = \$2 AND product_id = \$3`).
			WithArgs(tenantID, warehouseID, productID, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		item, err := repo.FindByWarehouseAndProduct(context.Background(), tenantID, warehouseID, productID)

		assert.Error(t, err)
		assert.Nil(t, item)
		assert.Equal(t, shared.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormInventoryItemRepository_FindByIDs(t *testing.T) {
	t.Run("finds multiple inventory items by IDs", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryItemRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		id1 := uuid.New()
		id2 := uuid.New()
		warehouseID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "warehouse_id", "product_id",
			"available_quantity", "locked_quantity", "unit_cost",
			"min_quantity", "max_quantity", "version",
		}).
			AddRow(id1, tenantID, warehouseID, uuid.New(), decimal.NewFromInt(100), decimal.Zero, decimal.NewFromFloat(10.00), decimal.Zero, decimal.Zero, 1).
			AddRow(id2, tenantID, warehouseID, uuid.New(), decimal.NewFromInt(200), decimal.Zero, decimal.NewFromFloat(20.00), decimal.Zero, decimal.Zero, 1)

		mock.ExpectQuery(`SELECT \* FROM "inventory_items" WHERE tenant_id = \$1 AND id IN \(\$2,\$3\)`).
			WithArgs(tenantID, id1, id2).
			WillReturnRows(rows)

		items, err := repo.FindByIDs(context.Background(), tenantID, []uuid.UUID{id1, id2})

		assert.NoError(t, err)
		assert.Len(t, items, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty slice for empty IDs", func(t *testing.T) {
		repo, _, mockDB := newMockInventoryItemRepository(t)
		defer mockDB.Close()

		items, err := repo.FindByIDs(context.Background(), uuid.New(), []uuid.UUID{})

		assert.NoError(t, err)
		assert.Empty(t, items)
	})
}

func TestGormInventoryItemRepository_Save(t *testing.T) {
	t.Run("saves inventory item", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryItemRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		warehouseID := uuid.New()
		productID := uuid.New()
		item, _ := inventory.NewInventoryItem(tenantID, warehouseID, productID)

		mock.ExpectExec(`UPDATE "inventory_items" SET`).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Save(context.Background(), item)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormInventoryItemRepository_Delete(t *testing.T) {
	t.Run("deletes existing inventory item", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryItemRepository(t)
		defer mockDB.Close()

		itemID := uuid.New()

		mock.ExpectExec(`DELETE FROM "inventory_items" WHERE id = \$1`).
			WithArgs(itemID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Delete(context.Background(), itemID)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error for non-existent item", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryItemRepository(t)
		defer mockDB.Close()

		itemID := uuid.New()

		mock.ExpectExec(`DELETE FROM "inventory_items" WHERE id = \$1`).
			WithArgs(itemID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Delete(context.Background(), itemID)

		assert.Error(t, err)
		assert.Equal(t, shared.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormInventoryItemRepository_DeleteForTenant(t *testing.T) {
	t.Run("deletes inventory item within tenant", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryItemRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		itemID := uuid.New()

		mock.ExpectExec(`DELETE FROM "inventory_items" WHERE tenant_id = \$1 AND id = \$2`).
			WithArgs(tenantID, itemID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.DeleteForTenant(context.Background(), tenantID, itemID)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormInventoryItemRepository_CountForTenant(t *testing.T) {
	t.Run("counts inventory items for tenant", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryItemRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "inventory_items" WHERE tenant_id = \$1`).
			WithArgs(tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(15))

		count, err := repo.CountForTenant(context.Background(), tenantID, shared.Filter{})

		assert.NoError(t, err)
		assert.Equal(t, int64(15), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormInventoryItemRepository_CountByWarehouse(t *testing.T) {
	t.Run("counts inventory items in warehouse", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryItemRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		warehouseID := uuid.New()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "inventory_items" WHERE tenant_id = \$1 AND warehouse_id = \$2`).
			WithArgs(tenantID, warehouseID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(8))

		count, err := repo.CountByWarehouse(context.Background(), tenantID, warehouseID)

		assert.NoError(t, err)
		assert.Equal(t, int64(8), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormInventoryItemRepository_CountByProduct(t *testing.T) {
	t.Run("counts inventory items for product", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryItemRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		productID := uuid.New()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "inventory_items" WHERE tenant_id = \$1 AND product_id = \$2`).
			WithArgs(tenantID, productID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

		count, err := repo.CountByProduct(context.Background(), tenantID, productID)

		assert.NoError(t, err)
		assert.Equal(t, int64(3), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormInventoryItemRepository_ExistsByWarehouseAndProduct(t *testing.T) {
	t.Run("returns true when inventory exists", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryItemRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		warehouseID := uuid.New()
		productID := uuid.New()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "inventory_items" WHERE tenant_id = \$1 AND warehouse_id = \$2 AND product_id = \$3`).
			WithArgs(tenantID, warehouseID, productID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		exists, err := repo.ExistsByWarehouseAndProduct(context.Background(), tenantID, warehouseID, productID)

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns false when inventory does not exist", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryItemRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		warehouseID := uuid.New()
		productID := uuid.New()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "inventory_items" WHERE tenant_id = \$1 AND warehouse_id = \$2 AND product_id = \$3`).
			WithArgs(tenantID, warehouseID, productID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		exists, err := repo.ExistsByWarehouseAndProduct(context.Background(), tenantID, warehouseID, productID)

		assert.NoError(t, err)
		assert.False(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormInventoryItemRepository_InterfaceCompliance(t *testing.T) {
	t.Run("implements InventoryItemRepository interface", func(t *testing.T) {
		repo, _, mockDB := newMockInventoryItemRepository(t)
		defer mockDB.Close()

		var _ inventory.InventoryItemRepository = repo
	})
}

// Test StockBatchRepository

func newMockStockBatchRepository(t *testing.T) (*GormStockBatchRepository, sqlmock.Sqlmock, *sql.DB) {
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

	return NewGormStockBatchRepository(gormDB), mock, mockDB
}

func TestGormStockBatchRepository_FindByID(t *testing.T) {
	t.Run("finds existing batch", func(t *testing.T) {
		repo, mock, mockDB := newMockStockBatchRepository(t)
		defer mockDB.Close()

		batchID := uuid.New()
		inventoryItemID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "inventory_item_id", "batch_number", "quantity", "unit_cost", "consumed",
		}).AddRow(
			batchID, inventoryItemID, "BATCH001", decimal.NewFromInt(100), decimal.NewFromFloat(10.00), false,
		)

		mock.ExpectQuery(`SELECT \* FROM "stock_batches" WHERE id = \$1`).
			WithArgs(batchID, 1).
			WillReturnRows(rows)

		batch, err := repo.FindByID(context.Background(), batchID)

		assert.NoError(t, err)
		assert.NotNil(t, batch)
		assert.Equal(t, batchID, batch.ID)
		assert.Equal(t, "BATCH001", batch.BatchNumber)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error for non-existent batch", func(t *testing.T) {
		repo, mock, mockDB := newMockStockBatchRepository(t)
		defer mockDB.Close()

		batchID := uuid.New()

		mock.ExpectQuery(`SELECT \* FROM "stock_batches" WHERE id = \$1`).
			WithArgs(batchID, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		batch, err := repo.FindByID(context.Background(), batchID)

		assert.Error(t, err)
		assert.Nil(t, batch)
		assert.Equal(t, shared.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormStockBatchRepository_FindByBatchNumber(t *testing.T) {
	t.Run("finds batch by batch number", func(t *testing.T) {
		repo, mock, mockDB := newMockStockBatchRepository(t)
		defer mockDB.Close()

		batchID := uuid.New()
		inventoryItemID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "inventory_item_id", "batch_number", "quantity", "unit_cost", "consumed",
		}).AddRow(
			batchID, inventoryItemID, "BATCH001", decimal.NewFromInt(50), decimal.NewFromFloat(15.00), false,
		)

		mock.ExpectQuery(`SELECT \* FROM "stock_batches" WHERE inventory_item_id = \$1 AND batch_number = \$2`).
			WithArgs(inventoryItemID, "BATCH001", 1).
			WillReturnRows(rows)

		batch, err := repo.FindByBatchNumber(context.Background(), inventoryItemID, "BATCH001")

		assert.NoError(t, err)
		assert.NotNil(t, batch)
		assert.Equal(t, "BATCH001", batch.BatchNumber)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormStockBatchRepository_Save(t *testing.T) {
	t.Run("saves batch", func(t *testing.T) {
		repo, mock, mockDB := newMockStockBatchRepository(t)
		defer mockDB.Close()

		batch := inventory.NewStockBatch(uuid.New(), "BATCH001", nil, nil, decimal.NewFromInt(100), decimal.NewFromFloat(10.00))

		mock.ExpectExec(`UPDATE "stock_batches" SET`).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Save(context.Background(), batch)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormStockBatchRepository_SaveBatch(t *testing.T) {
	t.Run("returns nil for empty batch", func(t *testing.T) {
		repo, _, mockDB := newMockStockBatchRepository(t)
		defer mockDB.Close()

		err := repo.SaveBatch(context.Background(), []inventory.StockBatch{})

		assert.NoError(t, err)
	})
}

func TestGormStockBatchRepository_Delete(t *testing.T) {
	t.Run("deletes existing batch", func(t *testing.T) {
		repo, mock, mockDB := newMockStockBatchRepository(t)
		defer mockDB.Close()

		batchID := uuid.New()

		mock.ExpectExec(`DELETE FROM "stock_batches" WHERE id = \$1`).
			WithArgs(batchID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Delete(context.Background(), batchID)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormStockBatchRepository_InterfaceCompliance(t *testing.T) {
	t.Run("implements StockBatchRepository interface", func(t *testing.T) {
		repo, _, mockDB := newMockStockBatchRepository(t)
		defer mockDB.Close()

		var _ inventory.StockBatchRepository = repo
	})
}

// Test StockLockRepository

func newMockStockLockRepository(t *testing.T) (*GormStockLockRepository, sqlmock.Sqlmock, *sql.DB) {
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

	return NewGormStockLockRepository(gormDB), mock, mockDB
}

func TestGormStockLockRepository_FindByID(t *testing.T) {
	t.Run("finds existing lock", func(t *testing.T) {
		repo, mock, mockDB := newMockStockLockRepository(t)
		defer mockDB.Close()

		lockID := uuid.New()
		inventoryItemID := uuid.New()
		expireAt := time.Now().Add(24 * time.Hour)

		rows := sqlmock.NewRows([]string{
			"id", "inventory_item_id", "quantity", "source_type", "source_id",
			"expire_at", "released", "consumed",
		}).AddRow(
			lockID, inventoryItemID, decimal.NewFromInt(10), "sales_order", "SO-001",
			expireAt, false, false,
		)

		mock.ExpectQuery(`SELECT \* FROM "stock_locks" WHERE id = \$1`).
			WithArgs(lockID, 1).
			WillReturnRows(rows)

		lock, err := repo.FindByID(context.Background(), lockID)

		assert.NoError(t, err)
		assert.NotNil(t, lock)
		assert.Equal(t, lockID, lock.ID)
		assert.Equal(t, "sales_order", lock.SourceType)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error for non-existent lock", func(t *testing.T) {
		repo, mock, mockDB := newMockStockLockRepository(t)
		defer mockDB.Close()

		lockID := uuid.New()

		mock.ExpectQuery(`SELECT \* FROM "stock_locks" WHERE id = \$1`).
			WithArgs(lockID, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		lock, err := repo.FindByID(context.Background(), lockID)

		assert.Error(t, err)
		assert.Nil(t, lock)
		assert.Equal(t, shared.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormStockLockRepository_Save(t *testing.T) {
	t.Run("saves lock", func(t *testing.T) {
		repo, mock, mockDB := newMockStockLockRepository(t)
		defer mockDB.Close()

		lock := inventory.NewStockLock(
			uuid.New(),
			decimal.NewFromInt(10),
			"sales_order",
			"SO-001",
			time.Now().Add(24*time.Hour),
		)

		mock.ExpectExec(`UPDATE "stock_locks" SET`).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Save(context.Background(), lock)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormStockLockRepository_Delete(t *testing.T) {
	t.Run("deletes existing lock", func(t *testing.T) {
		repo, mock, mockDB := newMockStockLockRepository(t)
		defer mockDB.Close()

		lockID := uuid.New()

		mock.ExpectExec(`DELETE FROM "stock_locks" WHERE id = \$1`).
			WithArgs(lockID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Delete(context.Background(), lockID)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormStockLockRepository_InterfaceCompliance(t *testing.T) {
	t.Run("implements StockLockRepository interface", func(t *testing.T) {
		repo, _, mockDB := newMockStockLockRepository(t)
		defer mockDB.Close()

		var _ inventory.StockLockRepository = repo
	})
}

// Test InventoryTransactionRepository

func newMockInventoryTransactionRepository(t *testing.T) (*GormInventoryTransactionRepository, sqlmock.Sqlmock, *sql.DB) {
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

	return NewGormInventoryTransactionRepository(gormDB), mock, mockDB
}

func TestGormInventoryTransactionRepository_FindByID(t *testing.T) {
	t.Run("finds existing transaction", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryTransactionRepository(t)
		defer mockDB.Close()

		txID := uuid.New()
		tenantID := uuid.New()
		inventoryItemID := uuid.New()
		warehouseID := uuid.New()
		productID := uuid.New()
		txDate := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "inventory_item_id", "warehouse_id", "product_id",
			"transaction_type", "quantity", "unit_cost", "total_cost",
			"balance_before", "balance_after", "source_type", "source_id", "transaction_date",
		}).AddRow(
			txID, tenantID, inventoryItemID, warehouseID, productID,
			"INBOUND", decimal.NewFromInt(100), decimal.NewFromFloat(10.00), decimal.NewFromFloat(1000.00),
			decimal.Zero, decimal.NewFromInt(100), "PURCHASE_ORDER", "PO-001", txDate,
		)

		mock.ExpectQuery(`SELECT \* FROM "inventory_transactions" WHERE id = \$1`).
			WithArgs(txID, 1).
			WillReturnRows(rows)

		tx, err := repo.FindByID(context.Background(), txID)

		assert.NoError(t, err)
		assert.NotNil(t, tx)
		assert.Equal(t, txID, tx.ID)
		assert.Equal(t, inventory.TransactionTypeInbound, tx.TransactionType)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error for non-existent transaction", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryTransactionRepository(t)
		defer mockDB.Close()

		txID := uuid.New()

		mock.ExpectQuery(`SELECT \* FROM "inventory_transactions" WHERE id = \$1`).
			WithArgs(txID, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		tx, err := repo.FindByID(context.Background(), txID)

		assert.Error(t, err)
		assert.Nil(t, tx)
		assert.Equal(t, shared.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormInventoryTransactionRepository_Create(t *testing.T) {
	t.Run("creates transaction", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryTransactionRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		inventoryItemID := uuid.New()
		warehouseID := uuid.New()
		productID := uuid.New()

		tx, _ := inventory.NewInventoryTransaction(
			tenantID,
			inventoryItemID,
			warehouseID,
			productID,
			inventory.TransactionTypeInbound,
			decimal.NewFromInt(100),
			decimal.NewFromFloat(10.00),
			decimal.Zero,
			decimal.NewFromInt(100),
			inventory.SourceTypePurchaseOrder,
			"PO-001",
		)

		mock.ExpectExec(`INSERT INTO "inventory_transactions"`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(context.Background(), tx)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormInventoryTransactionRepository_CreateBatch(t *testing.T) {
	t.Run("returns nil for empty batch", func(t *testing.T) {
		repo, _, mockDB := newMockInventoryTransactionRepository(t)
		defer mockDB.Close()

		err := repo.CreateBatch(context.Background(), []*inventory.InventoryTransaction{})

		assert.NoError(t, err)
	})
}

func TestGormInventoryTransactionRepository_CountForTenant(t *testing.T) {
	t.Run("counts transactions for tenant", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryTransactionRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "inventory_transactions" WHERE tenant_id = \$1`).
			WithArgs(tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(50))

		count, err := repo.CountForTenant(context.Background(), tenantID, shared.Filter{})

		assert.NoError(t, err)
		assert.Equal(t, int64(50), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormInventoryTransactionRepository_CountByInventoryItem(t *testing.T) {
	t.Run("counts transactions for inventory item", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryTransactionRepository(t)
		defer mockDB.Close()

		inventoryItemID := uuid.New()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "inventory_transactions" WHERE inventory_item_id = \$1`).
			WithArgs(inventoryItemID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(25))

		count, err := repo.CountByInventoryItem(context.Background(), inventoryItemID)

		assert.NoError(t, err)
		assert.Equal(t, int64(25), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormInventoryTransactionRepository_InterfaceCompliance(t *testing.T) {
	t.Run("implements InventoryTransactionRepository interface", func(t *testing.T) {
		repo, _, mockDB := newMockInventoryTransactionRepository(t)
		defer mockDB.Close()

		var _ inventory.InventoryTransactionRepository = repo
	})
}
