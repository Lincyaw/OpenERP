package batch

import (
	"context"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFIFOBatchStrategy_SelectBatches(t *testing.T) {
	s := NewFIFOBatchStrategy()
	ctx := context.Background()

	// Create test batches with different manufacture dates
	now := time.Now()
	batches := []strategy.Batch{
		{
			ID:              "batch-3",
			TenantID:        "tenant-1",
			ProductID:       "product-1",
			WarehouseID:     "warehouse-1",
			BatchNumber:     "B003",
			Quantity:        decimal.NewFromInt(100),
			AvailableQty:    decimal.NewFromInt(30),
			UnitCost:        decimal.NewFromFloat(10.0),
			ManufactureDate: now.Add(-24 * time.Hour), // Oldest - 1 day ago
			ReceivedDate:    now.Add(-20 * time.Hour),
		},
		{
			ID:              "batch-1",
			TenantID:        "tenant-1",
			ProductID:       "product-1",
			WarehouseID:     "warehouse-1",
			BatchNumber:     "B001",
			Quantity:        decimal.NewFromInt(100),
			AvailableQty:    decimal.NewFromInt(50),
			UnitCost:        decimal.NewFromFloat(12.0),
			ManufactureDate: now.Add(-12 * time.Hour), // 12 hours ago
			ReceivedDate:    now.Add(-10 * time.Hour),
		},
		{
			ID:              "batch-2",
			TenantID:        "tenant-1",
			ProductID:       "product-1",
			WarehouseID:     "warehouse-1",
			BatchNumber:     "B002",
			Quantity:        decimal.NewFromInt(100),
			AvailableQty:    decimal.NewFromInt(40),
			UnitCost:        decimal.NewFromFloat(11.0),
			ManufactureDate: now.Add(-6 * time.Hour), // Newest - 6 hours ago
			ReceivedDate:    now.Add(-5 * time.Hour),
		},
	}

	t.Run("selects batches in FIFO order by manufacture date", func(t *testing.T) {
		selCtx := strategy.BatchSelectionContext{
			TenantID:    "tenant-1",
			ProductID:   "product-1",
			WarehouseID: "warehouse-1",
			Quantity:    decimal.NewFromInt(60),
			Date:        now,
		}

		result, err := s.SelectBatches(ctx, selCtx, batches)
		require.NoError(t, err)

		// Should select oldest batch first (batch-3, 30 units), then next oldest (batch-1, 30 units)
		assert.Len(t, result.Selections, 2)
		assert.Equal(t, "batch-3", result.Selections[0].BatchID)
		assert.Equal(t, decimal.NewFromInt(30), result.Selections[0].Quantity)
		assert.Equal(t, "batch-1", result.Selections[1].BatchID)
		assert.Equal(t, decimal.NewFromInt(30), result.Selections[1].Quantity)
		assert.True(t, result.TotalQty.Equal(decimal.NewFromInt(60)))
		assert.True(t, result.ShortfallQty.IsZero())
	})

	t.Run("reports shortfall when insufficient quantity", func(t *testing.T) {
		selCtx := strategy.BatchSelectionContext{
			TenantID:    "tenant-1",
			ProductID:   "product-1",
			WarehouseID: "warehouse-1",
			Quantity:    decimal.NewFromInt(200), // More than total available (120)
			Date:        now,
		}

		result, err := s.SelectBatches(ctx, selCtx, batches)
		require.NoError(t, err)

		assert.Len(t, result.Selections, 3)
		assert.True(t, result.TotalQty.Equal(decimal.NewFromInt(120)))
		assert.True(t, result.ShortfallQty.Equal(decimal.NewFromInt(80)))
	})

	t.Run("respects preferred batch", func(t *testing.T) {
		selCtx := strategy.BatchSelectionContext{
			TenantID:    "tenant-1",
			ProductID:   "product-1",
			WarehouseID: "warehouse-1",
			Quantity:    decimal.NewFromInt(30),
			Date:        now,
			PreferBatch: "B002", // Prefer the newest batch
		}

		result, err := s.SelectBatches(ctx, selCtx, batches)
		require.NoError(t, err)

		// Should select preferred batch first
		assert.Len(t, result.Selections, 1)
		assert.Equal(t, "batch-2", result.Selections[0].BatchID)
		assert.Equal(t, "B002", result.Selections[0].BatchNumber)
	})

	t.Run("filters by product and warehouse", func(t *testing.T) {
		mixedBatches := append(batches, strategy.Batch{
			ID:              "batch-other",
			TenantID:        "tenant-1",
			ProductID:       "product-2", // Different product
			WarehouseID:     "warehouse-1",
			BatchNumber:     "B999",
			Quantity:        decimal.NewFromInt(100),
			AvailableQty:    decimal.NewFromInt(100),
			UnitCost:        decimal.NewFromFloat(5.0),
			ManufactureDate: now.Add(-48 * time.Hour), // Oldest overall but different product
		})

		selCtx := strategy.BatchSelectionContext{
			TenantID:    "tenant-1",
			ProductID:   "product-1",
			WarehouseID: "warehouse-1",
			Quantity:    decimal.NewFromInt(200),
			Date:        now,
		}

		result, err := s.SelectBatches(ctx, selCtx, mixedBatches)
		require.NoError(t, err)

		// Should only select batches for product-1
		for _, sel := range result.Selections {
			assert.NotEqual(t, "batch-other", sel.BatchID)
		}
		assert.True(t, result.TotalQty.Equal(decimal.NewFromInt(120)))
	})

	t.Run("uses received date as fallback when manufacture date is zero", func(t *testing.T) {
		batchesWithoutMfgDate := []strategy.Batch{
			{
				ID:           "batch-a",
				ProductID:    "product-1",
				WarehouseID:  "warehouse-1",
				BatchNumber:  "BA",
				AvailableQty: decimal.NewFromInt(20),
				ReceivedDate: now.Add(-10 * time.Hour), // Older received date
			},
			{
				ID:           "batch-b",
				ProductID:    "product-1",
				WarehouseID:  "warehouse-1",
				BatchNumber:  "BB",
				AvailableQty: decimal.NewFromInt(20),
				ReceivedDate: now.Add(-5 * time.Hour), // Newer received date
			},
		}

		selCtx := strategy.BatchSelectionContext{
			ProductID:   "product-1",
			WarehouseID: "warehouse-1",
			Quantity:    decimal.NewFromInt(15),
		}

		result, err := s.SelectBatches(ctx, selCtx, batchesWithoutMfgDate)
		require.NoError(t, err)

		// Should select batch-a first (older received date)
		assert.Equal(t, "batch-a", result.Selections[0].BatchID)
	})
}

func TestFIFOBatchStrategy_Metadata(t *testing.T) {
	s := NewFIFOBatchStrategy()

	t.Run("returns correct name", func(t *testing.T) {
		assert.Equal(t, "fifo", s.Name())
	})

	t.Run("returns correct type", func(t *testing.T) {
		assert.Equal(t, strategy.StrategyTypeBatch, s.Type())
	})

	t.Run("does not consider expiry", func(t *testing.T) {
		assert.False(t, s.ConsidersExpiry())
	})

	t.Run("does not support FEFO", func(t *testing.T) {
		assert.False(t, s.SupportsFEFO())
	})

	t.Run("has description", func(t *testing.T) {
		assert.NotEmpty(t, s.Description())
		assert.Contains(t, s.Description(), "First In First Out")
	})
}
