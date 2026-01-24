package inventory

import (
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInventoryItem(t *testing.T) {
	tenantID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	t.Run("creates inventory item successfully", func(t *testing.T) {
		item, err := NewInventoryItem(tenantID, warehouseID, productID)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, item.ID)
		assert.Equal(t, tenantID, item.TenantID)
		assert.Equal(t, warehouseID, item.WarehouseID)
		assert.Equal(t, productID, item.ProductID)
		assert.True(t, item.AvailableQuantity.IsZero())
		assert.True(t, item.LockedQuantity.IsZero())
		assert.True(t, item.UnitCost.IsZero())
	})

	t.Run("fails with nil warehouse ID", func(t *testing.T) {
		item, err := NewInventoryItem(tenantID, uuid.Nil, productID)

		require.Error(t, err)
		assert.Nil(t, item)
		assert.Contains(t, err.Error(), "Warehouse ID")
	})

	t.Run("fails with nil product ID", func(t *testing.T) {
		item, err := NewInventoryItem(tenantID, warehouseID, uuid.Nil)

		require.Error(t, err)
		assert.Nil(t, item)
		assert.Contains(t, err.Error(), "Product ID")
	})
}

func TestInventoryItem_TotalQuantity(t *testing.T) {
	item := createTestInventoryItem(t)
	item.AvailableQuantity = decimal.NewFromInt(100)
	item.LockedQuantity = decimal.NewFromInt(20)

	total := item.TotalQuantity()

	assert.Equal(t, decimal.NewFromInt(120), total)
}

func TestInventoryItem_IncreaseStock(t *testing.T) {
	t.Run("increases stock and calculates weighted average cost", func(t *testing.T) {
		item := createTestInventoryItem(t)

		// First increase: 100 units at 10.00
		err := item.IncreaseStock(
			decimal.NewFromInt(100),
			valueobject.NewMoneyCNYFromFloat(10.00),
			nil,
		)

		require.NoError(t, err)
		assert.Equal(t, decimal.NewFromInt(100), item.AvailableQuantity)
		assert.Equal(t, "10", item.UnitCost.String())

		// Second increase: 100 units at 20.00
		// New cost = (100*10 + 100*20) / 200 = 15
		err = item.IncreaseStock(
			decimal.NewFromInt(100),
			valueobject.NewMoneyCNYFromFloat(20.00),
			nil,
		)

		require.NoError(t, err)
		assert.Equal(t, decimal.NewFromInt(200), item.AvailableQuantity)
		assert.Equal(t, "15", item.UnitCost.String())
	})

	t.Run("emits StockIncreased event", func(t *testing.T) {
		item := createTestInventoryItem(t)

		err := item.IncreaseStock(
			decimal.NewFromInt(50),
			valueobject.NewMoneyCNYFromFloat(10.00),
			nil,
		)

		require.NoError(t, err)
		events := item.GetDomainEvents()
		// First increase emits StockIncreased and InventoryCostChanged (cost changes from 0)
		require.GreaterOrEqual(t, len(events), 1)
		assert.Equal(t, EventTypeStockIncreased, events[0].EventType())
	})

	t.Run("emits InventoryCostChanged event when cost changes", func(t *testing.T) {
		item := createTestInventoryItem(t)
		item.AvailableQuantity = decimal.NewFromInt(100)
		item.UnitCost = decimal.NewFromFloat(10.00)

		err := item.IncreaseStock(
			decimal.NewFromInt(100),
			valueobject.NewMoneyCNYFromFloat(20.00),
			nil,
		)

		require.NoError(t, err)
		events := item.GetDomainEvents()
		require.Len(t, events, 2)
		assert.Equal(t, EventTypeStockIncreased, events[0].EventType())
		assert.Equal(t, EventTypeInventoryCostChanged, events[1].EventType())
	})

	t.Run("creates batch when batch info provided", func(t *testing.T) {
		item := createTestInventoryItem(t)
		batchInfo := &BatchInfo{
			BatchNumber: "BATCH-001",
		}

		err := item.IncreaseStock(
			decimal.NewFromInt(50),
			valueobject.NewMoneyCNYFromFloat(10.00),
			batchInfo,
		)

		require.NoError(t, err)
		assert.Len(t, item.Batches, 1)
		assert.Equal(t, "BATCH-001", item.Batches[0].BatchNumber)
		assert.Equal(t, decimal.NewFromInt(50), item.Batches[0].Quantity)
	})

	t.Run("fails with zero quantity", func(t *testing.T) {
		item := createTestInventoryItem(t)

		err := item.IncreaseStock(
			decimal.Zero,
			valueobject.NewMoneyCNYFromFloat(10.00),
			nil,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "positive")
	})

	t.Run("fails with negative quantity", func(t *testing.T) {
		item := createTestInventoryItem(t)

		err := item.IncreaseStock(
			decimal.NewFromInt(-10),
			valueobject.NewMoneyCNYFromFloat(10.00),
			nil,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "positive")
	})

	t.Run("fails with negative cost", func(t *testing.T) {
		item := createTestInventoryItem(t)

		err := item.IncreaseStock(
			decimal.NewFromInt(10),
			valueobject.NewMoneyCNYFromFloat(-5.00),
			nil,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "negative")
	})
}

func TestInventoryItem_LockStock(t *testing.T) {
	t.Run("locks stock successfully", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)
		expireAt := time.Now().Add(time.Hour)

		lock, err := item.LockStock(decimal.NewFromInt(30), "sales_order", "SO-001", expireAt)

		require.NoError(t, err)
		assert.NotNil(t, lock)
		assert.Equal(t, decimal.NewFromInt(70), item.AvailableQuantity)
		assert.Equal(t, decimal.NewFromInt(30), item.LockedQuantity)
		assert.Equal(t, "sales_order", lock.SourceType)
		assert.Equal(t, "SO-001", lock.SourceID)
	})

	t.Run("emits StockLocked event", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)
		item.ClearDomainEvents()

		_, err := item.LockStock(decimal.NewFromInt(30), "sales_order", "SO-001", time.Now().Add(time.Hour))

		require.NoError(t, err)
		events := item.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeStockLocked, events[0].EventType())
	})

	t.Run("fails with insufficient stock", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)

		_, err := item.LockStock(decimal.NewFromInt(150), "sales_order", "SO-001", time.Now().Add(time.Hour))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Insufficient")
	})

	t.Run("fails with zero quantity", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)

		_, err := item.LockStock(decimal.Zero, "sales_order", "SO-001", time.Now().Add(time.Hour))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "positive")
	})

	t.Run("fails with empty source type", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)

		_, err := item.LockStock(decimal.NewFromInt(10), "", "SO-001", time.Now().Add(time.Hour))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Source type")
	})

	t.Run("fails with empty source ID", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)

		_, err := item.LockStock(decimal.NewFromInt(10), "sales_order", "", time.Now().Add(time.Hour))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Source")
	})
}

func TestInventoryItem_UnlockStock(t *testing.T) {
	t.Run("unlocks stock successfully", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)
		lock, _ := item.LockStock(decimal.NewFromInt(30), "sales_order", "SO-001", time.Now().Add(time.Hour))
		item.ClearDomainEvents()

		err := item.UnlockStock(lock.ID)

		require.NoError(t, err)
		assert.Equal(t, decimal.NewFromInt(100), item.AvailableQuantity)
		assert.True(t, item.LockedQuantity.IsZero())
	})

	t.Run("emits StockUnlocked event", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)
		lock, _ := item.LockStock(decimal.NewFromInt(30), "sales_order", "SO-001", time.Now().Add(time.Hour))
		item.ClearDomainEvents()

		err := item.UnlockStock(lock.ID)

		require.NoError(t, err)
		events := item.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeStockUnlocked, events[0].EventType())
	})

	t.Run("fails with non-existent lock", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)

		err := item.UnlockStock(uuid.New())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("fails with already released lock", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)
		lock, _ := item.LockStock(decimal.NewFromInt(30), "sales_order", "SO-001", time.Now().Add(time.Hour))
		_ = item.UnlockStock(lock.ID)

		err := item.UnlockStock(lock.ID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestInventoryItem_DeductStock(t *testing.T) {
	t.Run("deducts locked stock successfully", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)
		lock, _ := item.LockStock(decimal.NewFromInt(30), "sales_order", "SO-001", time.Now().Add(time.Hour))
		item.ClearDomainEvents()

		err := item.DeductStock(lock.ID)

		require.NoError(t, err)
		assert.Equal(t, decimal.NewFromInt(70), item.AvailableQuantity)
		assert.True(t, item.LockedQuantity.IsZero())
		assert.Equal(t, decimal.NewFromInt(70), item.TotalQuantity())
	})

	t.Run("emits StockDeducted event", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)
		lock, _ := item.LockStock(decimal.NewFromInt(30), "sales_order", "SO-001", time.Now().Add(time.Hour))
		item.ClearDomainEvents()

		err := item.DeductStock(lock.ID)

		require.NoError(t, err)
		events := item.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeStockDeducted, events[0].EventType())
	})

	t.Run("emits StockBelowThreshold when below minimum", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)
		item.MinQuantity = decimal.NewFromInt(80) // Set minimum threshold
		lock, _ := item.LockStock(decimal.NewFromInt(30), "sales_order", "SO-001", time.Now().Add(time.Hour))
		item.ClearDomainEvents()

		err := item.DeductStock(lock.ID)

		require.NoError(t, err)
		events := item.GetDomainEvents()
		require.Len(t, events, 2)
		assert.Equal(t, EventTypeStockDeducted, events[0].EventType())
		assert.Equal(t, EventTypeStockBelowThreshold, events[1].EventType())
	})

	t.Run("fails with non-existent lock", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)

		err := item.DeductStock(uuid.New())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestInventoryItem_DecreaseStock(t *testing.T) {
	t.Run("decreases available stock successfully", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)
		item.ClearDomainEvents()

		err := item.DecreaseStock(decimal.NewFromInt(30), "PURCHASE_RETURN", "PR-001", "Purchase return")

		require.NoError(t, err)
		assert.Equal(t, decimal.NewFromInt(70), item.AvailableQuantity)
		assert.True(t, item.LockedQuantity.IsZero())
	})

	t.Run("emits StockDecreased event", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)
		item.ClearDomainEvents()

		err := item.DecreaseStock(decimal.NewFromInt(30), "PURCHASE_RETURN", "PR-001", "Purchase return")

		require.NoError(t, err)
		events := item.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeStockDecreased, events[0].EventType())
	})

	t.Run("emits StockBelowThreshold when below minimum", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)
		item.MinQuantity = decimal.NewFromInt(80) // Set minimum threshold
		item.ClearDomainEvents()

		err := item.DecreaseStock(decimal.NewFromInt(30), "PURCHASE_RETURN", "PR-001", "Purchase return")

		require.NoError(t, err)
		events := item.GetDomainEvents()
		require.Len(t, events, 2)
		assert.Equal(t, EventTypeStockDecreased, events[0].EventType())
		assert.Equal(t, EventTypeStockBelowThreshold, events[1].EventType())
	})

	t.Run("fails with zero quantity", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)

		err := item.DecreaseStock(decimal.Zero, "PURCHASE_RETURN", "PR-001", "Purchase return")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "positive")
	})

	t.Run("fails with negative quantity", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)

		err := item.DecreaseStock(decimal.NewFromInt(-10), "PURCHASE_RETURN", "PR-001", "Purchase return")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "positive")
	})

	t.Run("fails with insufficient stock", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)

		err := item.DecreaseStock(decimal.NewFromInt(150), "PURCHASE_RETURN", "PR-001", "Purchase return")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Insufficient")
	})

	t.Run("fails with empty source type", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)

		err := item.DecreaseStock(decimal.NewFromInt(30), "", "PR-001", "Purchase return")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Source type and ID are required")
	})

	t.Run("fails with empty source ID", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)

		err := item.DecreaseStock(decimal.NewFromInt(30), "PURCHASE_RETURN", "", "Purchase return")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Source type and ID are required")
	})

	t.Run("decreases all available stock", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)
		item.ClearDomainEvents()

		err := item.DecreaseStock(decimal.NewFromInt(100), "PURCHASE_RETURN", "PR-001", "Purchase return")

		require.NoError(t, err)
		assert.True(t, item.AvailableQuantity.IsZero())
	})
}

func TestInventoryItem_AdjustStock(t *testing.T) {
	t.Run("adjusts stock successfully (increase)", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)
		item.ClearDomainEvents()

		err := item.AdjustStock(decimal.NewFromInt(120), "Stock taking - found extra units")

		require.NoError(t, err)
		assert.Equal(t, decimal.NewFromInt(120), item.AvailableQuantity)
	})

	t.Run("adjusts stock successfully (decrease)", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)
		item.ClearDomainEvents()

		err := item.AdjustStock(decimal.NewFromInt(80), "Stock taking - missing units")

		require.NoError(t, err)
		assert.Equal(t, decimal.NewFromInt(80), item.AvailableQuantity)
	})

	t.Run("emits StockAdjusted event", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)
		item.ClearDomainEvents()

		err := item.AdjustStock(decimal.NewFromInt(120), "Adjustment reason")

		require.NoError(t, err)
		events := item.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeStockAdjusted, events[0].EventType())
	})

	t.Run("fails with negative quantity", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)

		err := item.AdjustStock(decimal.NewFromInt(-10), "Invalid adjustment")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "negative")
	})

	t.Run("fails with empty reason", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)

		err := item.AdjustStock(decimal.NewFromInt(80), "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "reason")
	})

	t.Run("fails when stock is locked", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)
		_, _ = item.LockStock(decimal.NewFromInt(30), "sales_order", "SO-001", time.Now().Add(time.Hour))

		err := item.AdjustStock(decimal.NewFromInt(80), "Cannot adjust with locks")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "locks")
	})
}

func TestInventoryItem_ThresholdChecks(t *testing.T) {
	t.Run("IsBelowMinimum returns true when below threshold", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 50)
		item.MinQuantity = decimal.NewFromInt(100)

		assert.True(t, item.IsBelowMinimum())
	})

	t.Run("IsBelowMinimum returns false when at or above threshold", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)
		item.MinQuantity = decimal.NewFromInt(100)

		assert.False(t, item.IsBelowMinimum())
	})

	t.Run("IsBelowMinimum returns false when threshold not set", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 50)

		assert.False(t, item.IsBelowMinimum())
	})

	t.Run("IsAboveMaximum returns true when above threshold", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 150)
		item.MaxQuantity = decimal.NewFromInt(100)

		assert.True(t, item.IsAboveMaximum())
	})

	t.Run("IsAboveMaximum returns false when at or below threshold", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)
		item.MaxQuantity = decimal.NewFromInt(100)

		assert.False(t, item.IsAboveMaximum())
	})
}

func TestInventoryItem_CanFulfill(t *testing.T) {
	t.Run("returns true when sufficient available stock", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)

		assert.True(t, item.CanFulfill(decimal.NewFromInt(50)))
		assert.True(t, item.CanFulfill(decimal.NewFromInt(100)))
	})

	t.Run("returns false when insufficient available stock", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)

		assert.False(t, item.CanFulfill(decimal.NewFromInt(150)))
	})

	t.Run("considers only available not locked", func(t *testing.T) {
		item := createTestInventoryItemWithStock(t, 100)
		_, _ = item.LockStock(decimal.NewFromInt(30), "sales_order", "SO-001", time.Now().Add(time.Hour))

		assert.True(t, item.CanFulfill(decimal.NewFromInt(70)))
		assert.False(t, item.CanFulfill(decimal.NewFromInt(80)))
	})
}

func TestInventoryItem_GetActiveLocks(t *testing.T) {
	item := createTestInventoryItemWithStock(t, 100)
	lock1, _ := item.LockStock(decimal.NewFromInt(20), "sales_order", "SO-001", time.Now().Add(time.Hour))
	_, _ = item.LockStock(decimal.NewFromInt(20), "sales_order", "SO-002", time.Now().Add(time.Hour))
	_ = item.UnlockStock(lock1.ID)

	activeLocks := item.GetActiveLocks()

	assert.Len(t, activeLocks, 1)
	assert.Equal(t, "SO-002", activeLocks[0].SourceID)
}

func TestInventoryItem_GetTotalValue(t *testing.T) {
	item := createTestInventoryItemWithStock(t, 100)
	item.UnitCost = decimal.NewFromFloat(25.50)
	item.LockedQuantity = decimal.NewFromInt(20)

	value := item.GetTotalValue()

	// Total = 120 * 25.50 = 3060
	expected := decimal.NewFromFloat(3060.00)
	assert.True(t, value.Amount().Equal(expected))
}

// Helper functions

func createTestInventoryItem(t *testing.T) *InventoryItem {
	t.Helper()
	item, err := NewInventoryItem(uuid.New(), uuid.New(), uuid.New())
	require.NoError(t, err)
	return item
}

func createTestInventoryItemWithStock(t *testing.T, quantity int64) *InventoryItem {
	t.Helper()
	item := createTestInventoryItem(t)
	err := item.IncreaseStock(
		decimal.NewFromInt(quantity),
		valueobject.NewMoneyCNYFromFloat(10.00),
		nil,
	)
	require.NoError(t, err)
	return item
}
