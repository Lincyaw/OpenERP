package inventory

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStockAllocationService(t *testing.T) {
	t.Run("creates with default options", func(t *testing.T) {
		service := NewStockAllocationService()

		assert.NotNil(t, service)
		assert.Equal(t, 30*time.Minute, service.GetDefaultLockDuration())
	})

	t.Run("creates with custom lock duration", func(t *testing.T) {
		customDuration := 1 * time.Hour
		service := NewStockAllocationService(WithDefaultLockDuration(customDuration))

		assert.Equal(t, customDuration, service.GetDefaultLockDuration())
	})

	t.Run("ignores invalid lock duration", func(t *testing.T) {
		service := NewStockAllocationService(WithDefaultLockDuration(-10 * time.Minute))

		assert.Equal(t, 30*time.Minute, service.GetDefaultLockDuration())
	})
}

func TestAllocationRequest_Validate(t *testing.T) {
	tenantID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	createInventoryItem := func() *InventoryItem {
		item, _ := NewInventoryItem(tenantID, warehouseID, productID)
		item.AvailableQuantity = MustNewInventoryQuantity(decimal.NewFromInt(100))
		return item
	}

	t.Run("valid request", func(t *testing.T) {
		req := AllocationRequest{
			Items: []AllocationItem{
				{
					InventoryItem: createInventoryItem(),
					Quantity:      decimal.NewFromInt(10),
				},
			},
			SourceType: "SALES_ORDER",
			SourceID:   uuid.New().String(),
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("empty items", func(t *testing.T) {
		req := AllocationRequest{
			Items:      []AllocationItem{},
			SourceType: "SALES_ORDER",
			SourceID:   uuid.New().String(),
		}

		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "At least one item is required")
	})

	t.Run("missing source type", func(t *testing.T) {
		req := AllocationRequest{
			Items: []AllocationItem{
				{
					InventoryItem: createInventoryItem(),
					Quantity:      decimal.NewFromInt(10),
				},
			},
			SourceType: "",
			SourceID:   uuid.New().String(),
		}

		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Source type is required")
	})

	t.Run("missing source ID", func(t *testing.T) {
		req := AllocationRequest{
			Items: []AllocationItem{
				{
					InventoryItem: createInventoryItem(),
					Quantity:      decimal.NewFromInt(10),
				},
			},
			SourceType: "SALES_ORDER",
			SourceID:   "",
		}

		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Source ID is required")
	})

	t.Run("nil inventory item", func(t *testing.T) {
		req := AllocationRequest{
			Items: []AllocationItem{
				{
					InventoryItem: nil,
					Quantity:      decimal.NewFromInt(10),
				},
			},
			SourceType: "SALES_ORDER",
			SourceID:   uuid.New().String(),
		}

		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Inventory item at index 0 is nil")
	})

	t.Run("zero quantity", func(t *testing.T) {
		req := AllocationRequest{
			Items: []AllocationItem{
				{
					InventoryItem: createInventoryItem(),
					Quantity:      decimal.Zero,
				},
			},
			SourceType: "SALES_ORDER",
			SourceID:   uuid.New().String(),
		}

		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Quantity at index 0 must be positive")
	})

	t.Run("negative quantity", func(t *testing.T) {
		req := AllocationRequest{
			Items: []AllocationItem{
				{
					InventoryItem: createInventoryItem(),
					Quantity:      decimal.NewFromInt(-10),
				},
			},
			SourceType: "SALES_ORDER",
			SourceID:   uuid.New().String(),
		}

		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Quantity at index 0 must be positive")
	})
}

func TestStockAllocationService_AllocateStock(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	warehouseID := uuid.New()
	sourceID := uuid.New().String()

	createInventoryItem := func(productID uuid.UUID, availableQty int64) *InventoryItem {
		item, _ := NewInventoryItem(tenantID, warehouseID, productID)
		item.AvailableQuantity = MustNewInventoryQuantity(decimal.NewFromInt(availableQty))
		return item
	}

	t.Run("successful allocation of single item", func(t *testing.T) {
		service := NewStockAllocationService()
		productID := uuid.New()
		item := createInventoryItem(productID, 100)

		req := AllocationRequest{
			Items: []AllocationItem{
				{InventoryItem: item, Quantity: decimal.NewFromInt(50)},
			},
			SourceType: "SALES_ORDER",
			SourceID:   sourceID,
		}

		result, err := service.AllocateStock(ctx, req)

		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.False(t, result.PartialSuccess)
		assert.False(t, result.Compensated)
		assert.Len(t, result.Items, 1)
		assert.True(t, result.Items[0].Success)
		assert.NotEqual(t, uuid.Nil, result.Items[0].LockID)
		assert.Equal(t, decimal.NewFromInt(50), result.TotalAllocated)
		assert.Empty(t, result.FailedItems)

		// Verify inventory item state
		assert.True(t, decimal.NewFromInt(50).Equal(item.AvailableQuantity.Amount()))
		assert.True(t, decimal.NewFromInt(50).Equal(item.LockedQuantity.Amount()))
	})

	t.Run("successful allocation of multiple items", func(t *testing.T) {
		service := NewStockAllocationService()
		item1 := createInventoryItem(uuid.New(), 100)
		item2 := createInventoryItem(uuid.New(), 200)

		req := AllocationRequest{
			Items: []AllocationItem{
				{InventoryItem: item1, Quantity: decimal.NewFromInt(50)},
				{InventoryItem: item2, Quantity: decimal.NewFromInt(100)},
			},
			SourceType: "SALES_ORDER",
			SourceID:   sourceID,
		}

		result, err := service.AllocateStock(ctx, req)

		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Len(t, result.Items, 2)
		assert.True(t, result.Items[0].Success)
		assert.True(t, result.Items[1].Success)
		assert.Equal(t, decimal.NewFromInt(150), result.TotalAllocated)
		assert.Equal(t, decimal.NewFromInt(150), result.TotalRequested)
		assert.Empty(t, result.FailedItems)

		// Verify both items have correct state
		assert.True(t, decimal.NewFromInt(50).Equal(item1.AvailableQuantity.Amount()))
		assert.True(t, decimal.NewFromInt(50).Equal(item1.LockedQuantity.Amount()))
		assert.True(t, decimal.NewFromInt(100).Equal(item2.AvailableQuantity.Amount()))
		assert.True(t, decimal.NewFromInt(100).Equal(item2.LockedQuantity.Amount()))
	})

	t.Run("partial allocation triggers compensation", func(t *testing.T) {
		service := NewStockAllocationService()
		item1 := createInventoryItem(uuid.New(), 100)
		item2 := createInventoryItem(uuid.New(), 10) // Insufficient stock

		req := AllocationRequest{
			Items: []AllocationItem{
				{InventoryItem: item1, Quantity: decimal.NewFromInt(50)},
				{InventoryItem: item2, Quantity: decimal.NewFromInt(50)}, // Will fail
			},
			SourceType: "SALES_ORDER",
			SourceID:   sourceID,
		}

		result, err := service.AllocateStock(ctx, req)

		require.NoError(t, err)
		assert.False(t, result.Success)
		assert.True(t, result.PartialSuccess)
		assert.True(t, result.Compensated)
		assert.Len(t, result.FailedItems, 1)
		assert.Equal(t, 1, result.FailedItems[0])
		assert.Len(t, result.CompensationResults, 1)
		assert.True(t, result.CompensationResults[0].Success) // Compensation succeeded

		// Verify item1's lock was compensated (rolled back)
		assert.True(t, decimal.NewFromInt(100).Equal(item1.AvailableQuantity.Amount()))
		assert.True(t, decimal.Zero.Equal(item1.LockedQuantity.Amount()))
	})

	t.Run("complete failure - no items allocated", func(t *testing.T) {
		service := NewStockAllocationService()
		item1 := createInventoryItem(uuid.New(), 10) // Insufficient
		item2 := createInventoryItem(uuid.New(), 10) // Insufficient

		req := AllocationRequest{
			Items: []AllocationItem{
				{InventoryItem: item1, Quantity: decimal.NewFromInt(50)},
				{InventoryItem: item2, Quantity: decimal.NewFromInt(50)},
			},
			SourceType: "SALES_ORDER",
			SourceID:   sourceID,
		}

		result, err := service.AllocateStock(ctx, req)

		require.NoError(t, err)
		assert.False(t, result.Success)
		assert.False(t, result.PartialSuccess) // No partial success, complete failure
		assert.False(t, result.Compensated)    // No compensation needed
		assert.Len(t, result.FailedItems, 2)
		assert.Equal(t, decimal.Zero, result.TotalAllocated)
		assert.Empty(t, result.CompensationResults)
	})

	t.Run("custom lock duration", func(t *testing.T) {
		service := NewStockAllocationService()
		item := createInventoryItem(uuid.New(), 100)

		customDuration := 2 * time.Hour
		req := AllocationRequest{
			Items: []AllocationItem{
				{InventoryItem: item, Quantity: decimal.NewFromInt(50)},
			},
			SourceType:   "SALES_ORDER",
			SourceID:     sourceID,
			LockDuration: customDuration,
		}

		result, err := service.AllocateStock(ctx, req)

		require.NoError(t, err)
		assert.True(t, result.Success)

		// Check that lock expires in approximately 2 hours
		expectedExpiry := time.Now().Add(customDuration)
		actualExpiry := result.Items[0].ExpireAt
		assert.WithinDuration(t, expectedExpiry, actualExpiry, 5*time.Second)
	})

	t.Run("returns validation error for invalid request", func(t *testing.T) {
		service := NewStockAllocationService()

		req := AllocationRequest{
			Items:      []AllocationItem{},
			SourceType: "SALES_ORDER",
			SourceID:   sourceID,
		}

		result, err := service.AllocateStock(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "At least one item is required")
	})

	t.Run("generates domain events on success", func(t *testing.T) {
		service := NewStockAllocationService()
		item := createInventoryItem(uuid.New(), 100)

		req := AllocationRequest{
			Items: []AllocationItem{
				{InventoryItem: item, Quantity: decimal.NewFromInt(50)},
			},
			SourceType: "SALES_ORDER",
			SourceID:   sourceID,
		}

		result, err := service.AllocateStock(ctx, req)

		require.NoError(t, err)
		assert.Len(t, result.DomainEvents, 1)
		assert.Equal(t, EventTypeStockAllocationCompleted, result.DomainEvents[0].EventType())
	})

	t.Run("generates domain events on partial failure", func(t *testing.T) {
		service := NewStockAllocationService()
		item1 := createInventoryItem(uuid.New(), 100)
		item2 := createInventoryItem(uuid.New(), 10) // Insufficient

		req := AllocationRequest{
			Items: []AllocationItem{
				{InventoryItem: item1, Quantity: decimal.NewFromInt(50)},
				{InventoryItem: item2, Quantity: decimal.NewFromInt(50)},
			},
			SourceType: "SALES_ORDER",
			SourceID:   sourceID,
		}

		result, err := service.AllocateStock(ctx, req)

		require.NoError(t, err)
		// Should have partial event and compensation event
		assert.Len(t, result.DomainEvents, 2)
		assert.Equal(t, EventTypeStockAllocationPartial, result.DomainEvents[0].EventType())
		assert.Equal(t, EventTypeStockAllocationCompensated, result.DomainEvents[1].EventType())
	})
}

func TestStockAllocationService_PreviewAllocation(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	warehouseID := uuid.New()
	sourceID := uuid.New().String()

	createInventoryItem := func(productID uuid.UUID, availableQty int64) *InventoryItem {
		item, _ := NewInventoryItem(tenantID, warehouseID, productID)
		item.AvailableQuantity = MustNewInventoryQuantity(decimal.NewFromInt(availableQty))
		return item
	}

	t.Run("preview with sufficient stock", func(t *testing.T) {
		service := NewStockAllocationService()
		item := createInventoryItem(uuid.New(), 100)

		req := AllocationRequest{
			Items: []AllocationItem{
				{InventoryItem: item, Quantity: decimal.NewFromInt(50)},
			},
			SourceType: "SALES_ORDER",
			SourceID:   sourceID,
		}

		preview, err := service.PreviewAllocation(ctx, req)

		require.NoError(t, err)
		assert.True(t, preview.CanFulfillAll)
		assert.Len(t, preview.Items, 1)
		assert.True(t, preview.Items[0].CanFulfill)
		assert.Equal(t, decimal.NewFromInt(50), preview.TotalRequested)
		assert.Equal(t, decimal.NewFromInt(100), preview.TotalAvailable)
		assert.Empty(t, preview.ShortageItems)

		// Verify inventory item is NOT modified
		assert.True(t, decimal.NewFromInt(100).Equal(item.AvailableQuantity.Amount()))
		assert.True(t, item.LockedQuantity.IsZero())
	})

	t.Run("preview with insufficient stock", func(t *testing.T) {
		service := NewStockAllocationService()
		item := createInventoryItem(uuid.New(), 30)

		req := AllocationRequest{
			Items: []AllocationItem{
				{InventoryItem: item, Quantity: decimal.NewFromInt(50)},
			},
			SourceType: "SALES_ORDER",
			SourceID:   sourceID,
		}

		preview, err := service.PreviewAllocation(ctx, req)

		require.NoError(t, err)
		assert.False(t, preview.CanFulfillAll)
		assert.Len(t, preview.Items, 1)
		assert.False(t, preview.Items[0].CanFulfill)
		assert.Equal(t, decimal.NewFromInt(20), preview.Items[0].ShortageQuantity)
		assert.Len(t, preview.ShortageItems, 1)
		assert.Equal(t, 0, preview.ShortageItems[0])
	})

	t.Run("preview with partial fulfillment possible", func(t *testing.T) {
		service := NewStockAllocationService()
		item1 := createInventoryItem(uuid.New(), 100) // Sufficient
		item2 := createInventoryItem(uuid.New(), 10)  // Insufficient

		req := AllocationRequest{
			Items: []AllocationItem{
				{InventoryItem: item1, Quantity: decimal.NewFromInt(50)},
				{InventoryItem: item2, Quantity: decimal.NewFromInt(50)},
			},
			SourceType: "SALES_ORDER",
			SourceID:   sourceID,
		}

		preview, err := service.PreviewAllocation(ctx, req)

		require.NoError(t, err)
		assert.False(t, preview.CanFulfillAll)
		assert.True(t, preview.Items[0].CanFulfill)
		assert.False(t, preview.Items[1].CanFulfill)
		assert.Len(t, preview.ShortageItems, 1)
		assert.Equal(t, 1, preview.ShortageItems[0])
	})
}

func TestStockAllocationService_ReleaseAllocation(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	warehouseID := uuid.New()
	sourceID := uuid.New().String()

	createInventoryItemWithLock := func(productID uuid.UUID, availableQty, lockedQty int64, sourceType, sourceID string) *InventoryItem {
		item, _ := NewInventoryItem(tenantID, warehouseID, productID)
		// Set initial available to total (available + locked) so after locking we have the desired state
		item.AvailableQuantity = MustNewInventoryQuantity(decimal.NewFromInt(availableQty + lockedQty))
		if lockedQty > 0 {
			// Create a lock - this moves lockedQty from available to locked
			_, _ = item.LockStock(decimal.NewFromInt(lockedQty), sourceType, sourceID, time.Now().Add(30*time.Minute))
		}
		return item
	}

	t.Run("release existing locks", func(t *testing.T) {
		service := NewStockAllocationService()
		item := createInventoryItemWithLock(uuid.New(), 50, 50, "SALES_ORDER", sourceID)

		// After creation: available=50, locked=50 (total 100)
		assert.True(t, decimal.NewFromInt(50).Equal(item.AvailableQuantity.Amount()))
		assert.True(t, decimal.NewFromInt(50).Equal(item.LockedQuantity.Amount()))
		assert.Len(t, item.GetActiveLocks(), 1)

		result, err := service.ReleaseAllocation(ctx, []*InventoryItem{item}, "SALES_ORDER", sourceID)

		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Len(t, result.Items, 1)
		assert.True(t, result.Items[0].Success)
		assert.True(t, decimal.NewFromInt(50).Equal(result.TotalReleased))

		// Verify lock is released - available should be 100 (50 + 50 released), locked should be 0
		assert.Empty(t, item.GetActiveLocks())
		assert.True(t, decimal.NewFromInt(100).Equal(item.AvailableQuantity.Amount()))
		assert.True(t, decimal.Zero.Equal(item.LockedQuantity.Amount()))
	})

	t.Run("release from multiple items", func(t *testing.T) {
		service := NewStockAllocationService()
		item1 := createInventoryItemWithLock(uuid.New(), 50, 50, "SALES_ORDER", sourceID)
		item2 := createInventoryItemWithLock(uuid.New(), 100, 100, "SALES_ORDER", sourceID)

		result, err := service.ReleaseAllocation(ctx,
			[]*InventoryItem{item1, item2}, "SALES_ORDER", sourceID)

		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Len(t, result.Items, 2)
		assert.True(t, decimal.NewFromInt(150).Equal(result.TotalReleased))
	})

	t.Run("no matching locks", func(t *testing.T) {
		service := NewStockAllocationService()
		item := createInventoryItemWithLock(uuid.New(), 50, 50, "TRANSFER", "different-source")

		result, err := service.ReleaseAllocation(ctx, []*InventoryItem{item}, "SALES_ORDER", sourceID)

		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Empty(t, result.Items) // No matching locks found
		assert.True(t, result.TotalReleased.IsZero())

		// Original lock should still exist
		assert.Len(t, item.GetActiveLocks(), 1)
	})

	t.Run("returns error for empty items", func(t *testing.T) {
		service := NewStockAllocationService()

		result, err := service.ReleaseAllocation(ctx, []*InventoryItem{}, "SALES_ORDER", sourceID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "At least one item is required")
	})

	t.Run("returns error for missing source", func(t *testing.T) {
		service := NewStockAllocationService()
		item := createInventoryItemWithLock(uuid.New(), 100, 50, "SALES_ORDER", sourceID)

		result, err := service.ReleaseAllocation(ctx, []*InventoryItem{item}, "", sourceID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "Source type and ID are required")
	})

	t.Run("generates domain events", func(t *testing.T) {
		service := NewStockAllocationService()
		item := createInventoryItemWithLock(uuid.New(), 50, 50, "SALES_ORDER", sourceID)

		result, err := service.ReleaseAllocation(ctx, []*InventoryItem{item}, "SALES_ORDER", sourceID)

		require.NoError(t, err)
		assert.Len(t, result.DomainEvents, 1)
		assert.Equal(t, EventTypeStockAllocationReleased, result.DomainEvents[0].EventType())
	})
}

func TestAllocationResult_GetSuccessfulLocks(t *testing.T) {
	result := &AllocationResult{
		Items: []AllocationItemResult{
			{InventoryItemID: uuid.New(), Success: true, LockID: uuid.New()},
			{InventoryItemID: uuid.New(), Success: false},
			{InventoryItemID: uuid.New(), Success: true, LockID: uuid.New()},
		},
	}

	successfulLocks := result.GetSuccessfulLocks()

	assert.Len(t, successfulLocks, 2)
	assert.True(t, successfulLocks[0].Success)
	assert.True(t, successfulLocks[1].Success)
}
