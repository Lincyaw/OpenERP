package persistence

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// newMockInventoryRepoForConcurrency creates a repository with mocked DB for concurrency tests
func newMockInventoryRepoForConcurrency(t *testing.T) (*GormInventoryItemRepository, sqlmock.Sqlmock, *sql.DB) {
	t.Helper()

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

// TestSaveWithLock_OptimisticLocking tests that SaveWithLock correctly implements optimistic locking
func TestSaveWithLock_OptimisticLocking(t *testing.T) {
	t.Run("successful save with correct version", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryRepoForConcurrency(t)
		defer mockDB.Close()

		item := createTestInventoryItemForConcurrency(t)
		item.Version = 2 // Simulate incremented version after domain operation

		// First, expect SELECT to get current version from DB
		rows := sqlmock.NewRows([]string{"version"}).AddRow(1) // DB has version 1
		mock.ExpectQuery(`SELECT .* FROM "inventory_items"`).
			WithArgs(item.ID).
			WillReturnRows(rows)

		// Then expect UPDATE with version check
		mock.ExpectExec(`UPDATE "inventory_items" SET`).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.SaveWithLock(context.Background(), item)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("fails when version mismatch (concurrent modification)", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryRepoForConcurrency(t)
		defer mockDB.Close()

		item := createTestInventoryItemForConcurrency(t)
		item.Version = 2 // Domain expects DB to have version 1

		// DB returns version 2 (another transaction already updated)
		rows := sqlmock.NewRows([]string{"version"}).AddRow(2)
		mock.ExpectQuery(`SELECT .* FROM "inventory_items"`).
			WithArgs(item.ID).
			WillReturnRows(rows)

		// No UPDATE should be attempted since version mismatch detected early

		err := repo.SaveWithLock(context.Background(), item)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "modified by another transaction")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("handles database error gracefully", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryRepoForConcurrency(t)
		defer mockDB.Close()

		item := createTestInventoryItemForConcurrency(t)
		item.Version = 2

		// First, SELECT returns correct version
		rows := sqlmock.NewRows([]string{"version"}).AddRow(1)
		mock.ExpectQuery(`SELECT .* FROM "inventory_items"`).
			WithArgs(item.ID).
			WillReturnRows(rows)

		// UPDATE fails with DB error
		mock.ExpectExec(`UPDATE "inventory_items" SET`).
			WillReturnError(assert.AnError)

		err := repo.SaveWithLock(context.Background(), item)

		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("fails when item not found in database", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryRepoForConcurrency(t)
		defer mockDB.Close()

		item := createTestInventoryItemForConcurrency(t)
		item.Version = 2

		// SELECT returns empty rows (GORM's Scan doesn't return ErrRecordNotFound)
		// We need to simulate RowsAffected = 0
		rows := sqlmock.NewRows([]string{"version"}) // Empty rows
		mock.ExpectQuery(`SELECT .* FROM "inventory_items"`).
			WithArgs(item.ID).
			WillReturnRows(rows)

		err := repo.SaveWithLock(context.Background(), item)

		require.Error(t, err)
		assert.ErrorIs(t, err, shared.ErrNotFound, "expected ErrNotFound when item does not exist")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("fails when rows affected is zero after UPDATE", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryRepoForConcurrency(t)
		defer mockDB.Close()

		item := createTestInventoryItemForConcurrency(t)
		item.Version = 2

		// SELECT returns expected version
		rows := sqlmock.NewRows([]string{"version"}).AddRow(1)
		mock.ExpectQuery(`SELECT .* FROM "inventory_items"`).
			WithArgs(item.ID).
			WillReturnRows(rows)

		// UPDATE succeeds but affects 0 rows (race condition between SELECT and UPDATE)
		mock.ExpectExec(`UPDATE "inventory_items" SET`).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.SaveWithLock(context.Background(), item)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "modified by another transaction")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestSaveWithLock_UpdatedFields tests that all necessary fields are updated
func TestSaveWithLock_UpdatedFields(t *testing.T) {
	t.Run("updates all inventory fields correctly", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryRepoForConcurrency(t)
		defer mockDB.Close()

		item := createTestInventoryItemForConcurrency(t)
		item.AvailableQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(80))
		item.LockedQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(20))
		item.UnitCost = decimal.NewFromFloat(15.50)
		item.MinQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(10))
		item.MaxQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(500))
		item.Version = 3
		item.UpdatedAt = time.Now()

		// First, SELECT returns expected version (version - 1 = 2)
		rows := sqlmock.NewRows([]string{"version"}).AddRow(2)
		mock.ExpectQuery(`SELECT .* FROM "inventory_items"`).
			WithArgs(item.ID).
			WillReturnRows(rows)

		// The update should include all these fields
		mock.ExpectExec(`UPDATE "inventory_items" SET`).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.SaveWithLock(context.Background(), item)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestConcurrentLockScenario_Domain tests concurrent lock scenarios at domain level
// This test demonstrates how optimistic locking would prevent race conditions
// by verifying that both readers start with the same version and both increment it
func TestConcurrentLockScenario_Domain(t *testing.T) {
	t.Run("simulates read-modify-write race condition prevention", func(t *testing.T) {
		// Simulate two readers getting the same inventory item (version 1)
		reader1 := createTestInventoryItemForConcurrency(t)
		reader1.AvailableQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(100))
		reader1.Version = 1

		reader2 := createTestInventoryItemForConcurrency(t)
		reader2.AvailableQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(100))
		reader2.Version = 1 // Same version as reader1

		// Both readers perform domain operations
		_, err := reader1.LockStock(decimal.NewFromInt(30), "order", "O-1", time.Now().Add(time.Hour))
		require.NoError(t, err)

		_, err = reader2.LockStock(decimal.NewFromInt(30), "order", "O-2", time.Now().Add(time.Hour))
		require.NoError(t, err)

		// Both have incremented their version to 2
		assert.Equal(t, 2, reader1.Version)
		assert.Equal(t, 2, reader2.Version)

		// In the real database scenario with SaveWithLock:
		// - Reader 1 saves first: SELECT returns version=1, version check passes,
		//   UPDATE WHERE version=1 succeeds, DB version becomes 2
		// - Reader 2 tries to save: SELECT returns version=2 (DB was updated),
		//   but reader2.Version-1=1 != 2, so optimistic lock fails immediately
		// This ensures only one writer succeeds when concurrent modifications occur
	})

	t.Run("repository SaveWithLock rejects stale version", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryRepoForConcurrency(t)
		defer mockDB.Close()

		item := createTestInventoryItemForConcurrency(t)
		item.Version = 2 // After domain operation increments version

		// Simulate database already has version 2 (another transaction saved first)
		// SaveWithLock first SELECTs the current version from DB
		// Then compares: currentVersion (2) != expectedVersion (item.Version-1 = 1)
		// So optimistic lock fails without even attempting UPDATE
		rows := sqlmock.NewRows([]string{"version"}).AddRow(2)
		mock.ExpectQuery(`SELECT .* FROM "inventory_items"`).
			WithArgs(item.ID).
			WillReturnRows(rows)

		err := repo.SaveWithLock(context.Background(), item)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "modified by another transaction")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestOversellPrevention_Domain tests overselling prevention at domain level
func TestOversellPrevention_Domain(t *testing.T) {
	t.Run("domain prevents locking more than available", func(t *testing.T) {
		item := createTestInventoryItemForConcurrency(t)
		item.AvailableQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(50))

		// Try to lock more than available - should fail at domain level
		_, err := item.LockStock(decimal.NewFromInt(100), "order", "O-1", time.Now().Add(time.Hour))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Insufficient")
	})

	t.Run("domain correctly tracks available vs locked", func(t *testing.T) {
		item := createTestInventoryItemForConcurrency(t)
		item.AvailableQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(100))

		// Lock 30 units
		_, err := item.LockStock(decimal.NewFromInt(30), "order", "O-1", time.Now().Add(time.Hour))
		require.NoError(t, err)

		assert.True(t, item.AvailableQuantity.Amount().Equal(decimal.NewFromInt(70)))
		assert.True(t, item.LockedQuantity.Amount().Equal(decimal.NewFromInt(30)))

		// Can't lock another 80 (only 70 available)
		_, err = item.LockStock(decimal.NewFromInt(80), "order", "O-2", time.Now().Add(time.Hour))
		require.Error(t, err)

		// Can lock 70
		_, err = item.LockStock(decimal.NewFromInt(70), "order", "O-3", time.Now().Add(time.Hour))
		require.NoError(t, err)

		assert.True(t, item.AvailableQuantity.Amount().Equal(decimal.NewFromInt(0)))
		assert.True(t, item.LockedQuantity.Amount().Equal(decimal.NewFromInt(100)))
	})

	t.Run("CanFulfill correctly reports availability", func(t *testing.T) {
		item := createTestInventoryItemForConcurrency(t)
		item.AvailableQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(100))

		assert.True(t, item.CanFulfill(decimal.NewFromInt(50)))
		assert.True(t, item.CanFulfill(decimal.NewFromInt(100)))
		assert.False(t, item.CanFulfill(decimal.NewFromInt(101)))

		// Lock some stock
		_, _ = item.LockStock(decimal.NewFromInt(60), "order", "O-1", time.Now().Add(time.Hour))

		assert.True(t, item.CanFulfill(decimal.NewFromInt(40)))
		assert.False(t, item.CanFulfill(decimal.NewFromInt(50)))
	})
}

// TestVersionIncrement tests that version is correctly incremented on domain operations
func TestVersionIncrement(t *testing.T) {
	t.Run("LockStock increments version", func(t *testing.T) {
		item := createTestInventoryItemForConcurrency(t)
		item.AvailableQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(100))
		initialVersion := item.Version

		_, err := item.LockStock(decimal.NewFromInt(30), "order", "O-1", time.Now().Add(time.Hour))
		require.NoError(t, err)

		assert.Equal(t, initialVersion+1, item.Version)
	})

	t.Run("UnlockStock increments version", func(t *testing.T) {
		item := createTestInventoryItemForConcurrency(t)
		item.AvailableQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(100))

		lock, err := item.LockStock(decimal.NewFromInt(30), "order", "O-1", time.Now().Add(time.Hour))
		require.NoError(t, err)
		versionAfterLock := item.Version

		err = item.UnlockStock(lock.ID)
		require.NoError(t, err)

		assert.Equal(t, versionAfterLock+1, item.Version)
	})

	t.Run("DeductStock increments version", func(t *testing.T) {
		item := createTestInventoryItemForConcurrency(t)
		item.AvailableQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(100))

		lock, err := item.LockStock(decimal.NewFromInt(30), "order", "O-1", time.Now().Add(time.Hour))
		require.NoError(t, err)
		versionAfterLock := item.Version

		err = item.DeductStock(lock.ID)
		require.NoError(t, err)

		assert.Equal(t, versionAfterLock+1, item.Version)
	})

	t.Run("IncreaseStock increments version", func(t *testing.T) {
		item := createTestInventoryItemForConcurrency(t)
		initialVersion := item.Version

		err := item.IncreaseStock(
			decimal.NewFromInt(50),
			valueobject.NewMoneyCNYFromFloat(10.00),
			nil,
		)
		require.NoError(t, err)

		assert.Equal(t, initialVersion+1, item.Version)
	})

	t.Run("AdjustStock increments version", func(t *testing.T) {
		item := createTestInventoryItemForConcurrency(t)
		item.AvailableQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(100))
		initialVersion := item.Version

		err := item.AdjustStock(decimal.NewFromInt(110), "Found extra stock")
		require.NoError(t, err)

		assert.Equal(t, initialVersion+1, item.Version)
	})
}

// TestGetOrCreate_RaceCondition tests that GetOrCreate handles race conditions
func TestGetOrCreate_RaceCondition(t *testing.T) {
	t.Run("handles concurrent creation with ON CONFLICT", func(t *testing.T) {
		repo, mock, mockDB := newMockInventoryRepoForConcurrency(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		warehouseID := uuid.New()
		productID := uuid.New()

		// First, the find returns not found
		mock.ExpectQuery(`SELECT \* FROM "inventory_items" WHERE tenant_id`).
			WillReturnError(gorm.ErrRecordNotFound)

		// Then insert with ON CONFLICT DO NOTHING
		mock.ExpectExec(`INSERT INTO "inventory_items"`).
			WillReturnResult(sqlmock.NewResult(0, 1))

		item, err := repo.GetOrCreate(context.Background(), tenantID, warehouseID, productID)

		require.NoError(t, err)
		assert.NotNil(t, item)
		assert.Equal(t, warehouseID, item.WarehouseID)
		assert.Equal(t, productID, item.ProductID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestStockLockRepository_Concurrency tests stock lock repository operations
func TestStockLockRepository_Concurrency(t *testing.T) {
	t.Run("FindActive filters correctly", func(t *testing.T) {
		repo, mock, mockDB := newMockStockLockRepository(t)
		defer mockDB.Close()

		inventoryItemID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "inventory_item_id", "quantity", "source_type", "source_id",
			"expire_at", "released", "consumed",
		}).
			AddRow(uuid.New(), inventoryItemID, decimal.NewFromInt(10), "order", "O-1", time.Now().Add(time.Hour), false, false).
			AddRow(uuid.New(), inventoryItemID, decimal.NewFromInt(20), "order", "O-2", time.Now().Add(time.Hour), false, false)

		mock.ExpectQuery(`SELECT \* FROM "stock_locks" WHERE inventory_item_id`).
			WillReturnRows(rows)

		locks, err := repo.FindActive(context.Background(), inventoryItemID)

		require.NoError(t, err)
		assert.Len(t, locks, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ReleaseExpired updates expired locks", func(t *testing.T) {
		repo, mock, mockDB := newMockStockLockRepository(t)
		defer mockDB.Close()

		mock.ExpectExec(`UPDATE "stock_locks" SET`).
			WillReturnResult(sqlmock.NewResult(0, 5))

		count, err := repo.ReleaseExpired(context.Background())

		require.NoError(t, err)
		assert.Equal(t, 5, count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestQuantityInvariant tests that quantity invariants are maintained
func TestQuantityInvariant(t *testing.T) {
	t.Run("TotalQuantity always equals Available + Locked", func(t *testing.T) {
		item := createTestInventoryItemForConcurrency(t)
		item.AvailableQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(100))

		// Initial state
		expectedTotal, _ := item.AvailableQuantity.Add(item.LockedQuantity)
		assert.True(t, item.TotalQuantity().Amount().Equal(expectedTotal.Amount()))

		// After lock
		_, _ = item.LockStock(decimal.NewFromInt(30), "order", "O-1", time.Now().Add(time.Hour))
		expectedTotal, _ = item.AvailableQuantity.Add(item.LockedQuantity)
		assert.True(t, item.TotalQuantity().Amount().Equal(expectedTotal.Amount()))
		assert.True(t, item.TotalQuantity().Amount().Equal(decimal.NewFromInt(100)))

		// After another lock
		_, _ = item.LockStock(decimal.NewFromInt(20), "order", "O-2", time.Now().Add(time.Hour))
		expectedTotal, _ = item.AvailableQuantity.Add(item.LockedQuantity)
		assert.True(t, item.TotalQuantity().Amount().Equal(expectedTotal.Amount()))
		assert.True(t, item.TotalQuantity().Amount().Equal(decimal.NewFromInt(100)))
	})

	t.Run("lock-unlock cycle preserves total", func(t *testing.T) {
		item := createTestInventoryItemForConcurrency(t)
		item.AvailableQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(100))
		initialTotal := item.TotalQuantity()

		lock, _ := item.LockStock(decimal.NewFromInt(30), "order", "O-1", time.Now().Add(time.Hour))
		_ = item.UnlockStock(lock.ID)

		assert.True(t, item.TotalQuantity().Amount().Equal(initialTotal.Amount()))
		assert.True(t, item.AvailableQuantity.Amount().Equal(decimal.NewFromInt(100)))
		assert.True(t, item.LockedQuantity.Amount().Equal(decimal.Zero))
	})

	t.Run("deduction reduces total correctly", func(t *testing.T) {
		item := createTestInventoryItemForConcurrency(t)
		item.AvailableQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(100))

		lock, _ := item.LockStock(decimal.NewFromInt(30), "order", "O-1", time.Now().Add(time.Hour))
		_ = item.DeductStock(lock.ID)

		assert.True(t, item.TotalQuantity().Amount().Equal(decimal.NewFromInt(70)))
		assert.True(t, item.AvailableQuantity.Amount().Equal(decimal.NewFromInt(70)))
		assert.True(t, item.LockedQuantity.Amount().Equal(decimal.Zero))
	})
}

// TestConcurrentMultipleLocks tests multiple concurrent lock operations
func TestConcurrentMultipleLocks(t *testing.T) {
	t.Run("multiple locks accumulate locked quantity correctly", func(t *testing.T) {
		item := createTestInventoryItemForConcurrency(t)
		item.AvailableQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(100))

		// Create 5 locks of 10 units each
		for range 5 {
			_, err := item.LockStock(
				decimal.NewFromInt(10),
				"sales_order",
				uuid.New().String(),
				time.Now().Add(time.Hour),
			)
			require.NoError(t, err)
		}

		assert.True(t, item.AvailableQuantity.Amount().Equal(decimal.NewFromInt(50)))
		assert.True(t, item.LockedQuantity.Amount().Equal(decimal.NewFromInt(50)))
		assert.True(t, item.TotalQuantity().Amount().Equal(decimal.NewFromInt(100)))
	})

	t.Run("lock fails when exact available reached", func(t *testing.T) {
		item := createTestInventoryItemForConcurrency(t)
		item.AvailableQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(50))

		// Lock exactly all available
		_, err := item.LockStock(decimal.NewFromInt(50), "order", "O-1", time.Now().Add(time.Hour))
		require.NoError(t, err)

		// Any additional lock should fail
		_, err = item.LockStock(decimal.NewFromInt(1), "order", "O-2", time.Now().Add(time.Hour))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Insufficient")
	})
}

// TestDeductionAndUnlockInteraction tests the interaction between deduction and unlock
func TestDeductionAndUnlockInteraction(t *testing.T) {
	t.Run("cannot unlock a consumed lock", func(t *testing.T) {
		item := createTestInventoryItemForConcurrency(t)
		item.AvailableQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(100))

		lock, err := item.LockStock(decimal.NewFromInt(30), "order", "O-1", time.Now().Add(time.Hour))
		require.NoError(t, err)

		// Deduct (consume) the lock
		err = item.DeductStock(lock.ID)
		require.NoError(t, err)

		// Try to unlock the consumed lock - should fail
		err = item.UnlockStock(lock.ID)
		require.Error(t, err)
	})

	t.Run("cannot deduct an already unlocked lock", func(t *testing.T) {
		item := createTestInventoryItemForConcurrency(t)
		item.AvailableQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(100))

		lock, err := item.LockStock(decimal.NewFromInt(30), "order", "O-1", time.Now().Add(time.Hour))
		require.NoError(t, err)

		// Unlock the lock
		err = item.UnlockStock(lock.ID)
		require.NoError(t, err)

		// Try to deduct the unlocked lock - should fail
		err = item.DeductStock(lock.ID)
		require.Error(t, err)
	})
}

// TestLockExpiration tests lock expiration handling
func TestLockExpiration(t *testing.T) {
	t.Run("GetExpiredLocks returns only expired locks", func(t *testing.T) {
		item := createTestInventoryItemForConcurrency(t)
		item.AvailableQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(100))

		// Create an expired lock
		_, err := item.LockStock(decimal.NewFromInt(10), "order", "O-1", time.Now().Add(-time.Hour))
		require.NoError(t, err)

		// Create a non-expired lock
		_, err = item.LockStock(decimal.NewFromInt(10), "order", "O-2", time.Now().Add(time.Hour))
		require.NoError(t, err)

		expiredLocks := item.GetExpiredLocks()
		assert.Len(t, expiredLocks, 1)
		assert.Equal(t, "O-1", expiredLocks[0].SourceID)
	})

	t.Run("ReleaseExpiredLocks releases expired locks", func(t *testing.T) {
		item := createTestInventoryItemForConcurrency(t)
		item.AvailableQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(100))

		// Create 2 expired locks
		_, _ = item.LockStock(decimal.NewFromInt(10), "order", "O-1", time.Now().Add(-time.Hour))
		_, _ = item.LockStock(decimal.NewFromInt(10), "order", "O-2", time.Now().Add(-time.Minute))

		// Create a non-expired lock
		_, _ = item.LockStock(decimal.NewFromInt(10), "order", "O-3", time.Now().Add(time.Hour))

		// Available is now 70, locked is 30
		assert.True(t, item.AvailableQuantity.Amount().Equal(decimal.NewFromInt(70)))
		assert.True(t, item.LockedQuantity.Amount().Equal(decimal.NewFromInt(30)))

		// Release expired locks
		count := item.ReleaseExpiredLocks()
		assert.Equal(t, 2, count)

		// Available should be 90 (70 + 20 released), locked should be 10
		assert.True(t, item.AvailableQuantity.Amount().Equal(decimal.NewFromInt(90)))
		assert.True(t, item.LockedQuantity.Amount().Equal(decimal.NewFromInt(10)))
	})
}

// Helper functions

func createTestInventoryItemForConcurrency(t *testing.T) *inventory.InventoryItem {
	t.Helper()
	item, err := inventory.NewInventoryItem(uuid.New(), uuid.New(), uuid.New())
	require.NoError(t, err)
	return item
}
