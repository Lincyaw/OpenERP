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

func TestFEFOBatchStrategy_SelectBatches(t *testing.T) {
	s := NewFEFOBatchStrategy()
	ctx := context.Background()

	// Create test batches with different expiry dates
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
			ManufactureDate: now.Add(-30 * 24 * time.Hour),
			ExpiryDate:      now.Add(60 * 24 * time.Hour), // Expires in 60 days
			ReceivedDate:    now.Add(-25 * 24 * time.Hour),
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
			ManufactureDate: now.Add(-20 * 24 * time.Hour),
			ExpiryDate:      now.Add(10 * 24 * time.Hour), // Expires in 10 days (earliest)
			ReceivedDate:    now.Add(-18 * 24 * time.Hour),
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
			ManufactureDate: now.Add(-10 * 24 * time.Hour),
			ExpiryDate:      now.Add(30 * 24 * time.Hour), // Expires in 30 days
			ReceivedDate:    now.Add(-8 * 24 * time.Hour),
		},
	}

	t.Run("selects batches in FEFO order by expiry date", func(t *testing.T) {
		selCtx := strategy.BatchSelectionContext{
			TenantID:    "tenant-1",
			ProductID:   "product-1",
			WarehouseID: "warehouse-1",
			Quantity:    decimal.NewFromInt(60),
			Date:        now,
		}

		result, err := s.SelectBatches(ctx, selCtx, batches)
		require.NoError(t, err)

		// Should select earliest expiring batch first (batch-1), then next earliest (batch-2)
		assert.Len(t, result.Selections, 2)
		assert.Equal(t, "batch-1", result.Selections[0].BatchID) // Expires in 10 days
		assert.Equal(t, decimal.NewFromInt(50), result.Selections[0].Quantity)
		assert.Equal(t, "batch-2", result.Selections[1].BatchID) // Expires in 30 days
		assert.Equal(t, decimal.NewFromInt(10), result.Selections[1].Quantity)
		assert.True(t, result.TotalQty.Equal(decimal.NewFromInt(60)))
		assert.True(t, result.ShortfallQty.IsZero())
	})

	t.Run("excludes already expired batches", func(t *testing.T) {
		expiredBatches := append(batches, strategy.Batch{
			ID:              "batch-expired",
			TenantID:        "tenant-1",
			ProductID:       "product-1",
			WarehouseID:     "warehouse-1",
			BatchNumber:     "B999",
			Quantity:        decimal.NewFromInt(100),
			AvailableQty:    decimal.NewFromInt(100),
			UnitCost:        decimal.NewFromFloat(5.0),
			ManufactureDate: now.Add(-90 * 24 * time.Hour),
			ExpiryDate:      now.Add(-1 * 24 * time.Hour), // Expired yesterday
			ReceivedDate:    now.Add(-80 * 24 * time.Hour),
		})

		selCtx := strategy.BatchSelectionContext{
			TenantID:    "tenant-1",
			ProductID:   "product-1",
			WarehouseID: "warehouse-1",
			Quantity:    decimal.NewFromInt(200),
			Date:        now,
		}

		result, err := s.SelectBatches(ctx, selCtx, expiredBatches)
		require.NoError(t, err)

		// Should not include expired batch
		for _, sel := range result.Selections {
			assert.NotEqual(t, "batch-expired", sel.BatchID)
		}
		assert.True(t, result.TotalQty.Equal(decimal.NewFromInt(120))) // Only non-expired batches
	})

	t.Run("handles batches without expiry dates (put them last)", func(t *testing.T) {
		batchesWithMixed := []strategy.Batch{
			{
				ID:              "batch-no-expiry",
				ProductID:       "product-1",
				WarehouseID:     "warehouse-1",
				BatchNumber:     "BNE",
				AvailableQty:    decimal.NewFromInt(50),
				ManufactureDate: now.Add(-5 * 24 * time.Hour), // Newer manufacture
				// No ExpiryDate
			},
			{
				ID:              "batch-with-expiry",
				ProductID:       "product-1",
				WarehouseID:     "warehouse-1",
				BatchNumber:     "BWE",
				AvailableQty:    decimal.NewFromInt(30),
				ManufactureDate: now.Add(-30 * 24 * time.Hour), // Older manufacture
				ExpiryDate:      now.Add(90 * 24 * time.Hour),
			},
		}

		selCtx := strategy.BatchSelectionContext{
			ProductID:   "product-1",
			WarehouseID: "warehouse-1",
			Quantity:    decimal.NewFromInt(70),
			Date:        now,
		}

		result, err := s.SelectBatches(ctx, selCtx, batchesWithMixed)
		require.NoError(t, err)

		// Batch with expiry should come first
		assert.Equal(t, "batch-with-expiry", result.Selections[0].BatchID)
		assert.Equal(t, "batch-no-expiry", result.Selections[1].BatchID)
	})

	t.Run("falls back to manufacture date when no expiry dates", func(t *testing.T) {
		batchesNoExpiry := []strategy.Batch{
			{
				ID:              "batch-newer",
				ProductID:       "product-1",
				WarehouseID:     "warehouse-1",
				BatchNumber:     "BN",
				AvailableQty:    decimal.NewFromInt(30),
				ManufactureDate: now.Add(-5 * 24 * time.Hour), // Newer
			},
			{
				ID:              "batch-older",
				ProductID:       "product-1",
				WarehouseID:     "warehouse-1",
				BatchNumber:     "BO",
				AvailableQty:    decimal.NewFromInt(30),
				ManufactureDate: now.Add(-15 * 24 * time.Hour), // Older
			},
		}

		selCtx := strategy.BatchSelectionContext{
			ProductID:   "product-1",
			WarehouseID: "warehouse-1",
			Quantity:    decimal.NewFromInt(50),
			Date:        now,
		}

		result, err := s.SelectBatches(ctx, selCtx, batchesNoExpiry)
		require.NoError(t, err)

		// When both have no expiry, fall back to manufacture date (older first)
		assert.Equal(t, "batch-older", result.Selections[0].BatchID)
	})

	t.Run("respects preferred batch", func(t *testing.T) {
		selCtx := strategy.BatchSelectionContext{
			TenantID:    "tenant-1",
			ProductID:   "product-1",
			WarehouseID: "warehouse-1",
			Quantity:    decimal.NewFromInt(30),
			Date:        now,
			PreferBatch: "B003", // Prefer batch with latest expiry
		}

		result, err := s.SelectBatches(ctx, selCtx, batches)
		require.NoError(t, err)

		// Should select preferred batch first regardless of expiry
		assert.Len(t, result.Selections, 1)
		assert.Equal(t, "batch-3", result.Selections[0].BatchID)
		assert.Equal(t, "B003", result.Selections[0].BatchNumber)
	})

	t.Run("reports shortfall when insufficient non-expired quantity", func(t *testing.T) {
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
}

func TestFEFOBatchStrategy_Metadata(t *testing.T) {
	s := NewFEFOBatchStrategy()

	t.Run("returns correct name", func(t *testing.T) {
		assert.Equal(t, "fefo", s.Name())
	})

	t.Run("returns correct type", func(t *testing.T) {
		assert.Equal(t, strategy.StrategyTypeBatch, s.Type())
	})

	t.Run("considers expiry", func(t *testing.T) {
		assert.True(t, s.ConsidersExpiry())
	})

	t.Run("supports FEFO", func(t *testing.T) {
		assert.True(t, s.SupportsFEFO())
	})

	t.Run("has description", func(t *testing.T) {
		assert.NotEmpty(t, s.Description())
		assert.Contains(t, s.Description(), "First Expired First Out")
	})
}
