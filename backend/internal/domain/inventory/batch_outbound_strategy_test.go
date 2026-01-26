package inventory

import (
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestBatch(batchNumber string, quantity, unitCost float64, productionDate, expiryDate *time.Time) StockBatch {
	batch := StockBatch{
		BaseEntity:     shared.NewBaseEntity(),
		BatchNumber:    batchNumber,
		ProductionDate: productionDate,
		ExpiryDate:     expiryDate,
		Quantity:       decimal.NewFromFloat(quantity),
		UnitCost:       decimal.NewFromFloat(unitCost),
		Consumed:       false,
	}
	return batch
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func TestBatchOutboundStrategyType(t *testing.T) {
	t.Run("IsValid returns true for valid types", func(t *testing.T) {
		assert.True(t, BatchOutboundStrategyTypeFIFO.IsValid())
		assert.True(t, BatchOutboundStrategyTypeFEFO.IsValid())
		assert.True(t, BatchOutboundStrategyTypeSpecified.IsValid())
	})

	t.Run("IsValid returns false for invalid type", func(t *testing.T) {
		invalidType := BatchOutboundStrategyType("INVALID")
		assert.False(t, invalidType.IsValid())
	})

	t.Run("String returns correct string", func(t *testing.T) {
		assert.Equal(t, "FIFO", BatchOutboundStrategyTypeFIFO.String())
		assert.Equal(t, "FEFO", BatchOutboundStrategyTypeFEFO.String())
		assert.Equal(t, "SPECIFIED", BatchOutboundStrategyTypeSpecified.String())
	})

	t.Run("AllBatchOutboundStrategyTypes returns all types", func(t *testing.T) {
		types := AllBatchOutboundStrategyTypes()
		assert.Len(t, types, 3)
		assert.Contains(t, types, BatchOutboundStrategyTypeFIFO)
		assert.Contains(t, types, BatchOutboundStrategyTypeFEFO)
		assert.Contains(t, types, BatchOutboundStrategyTypeSpecified)
	})
}

func TestFIFOBatchOutboundStrategy(t *testing.T) {
	strategy := NewFIFOBatchOutboundStrategy()

	t.Run("Strategy metadata is correct", func(t *testing.T) {
		assert.Equal(t, "fifo_batch_outbound", strategy.Name())
		assert.Equal(t, BatchOutboundStrategyTypeFIFO, strategy.StrategyType())
		assert.NotEmpty(t, strategy.Description())
	})

	t.Run("Returns error for zero quantity", func(t *testing.T) {
		batches := []StockBatch{createTestBatch("B001", 100, 10, nil, nil)}
		_, err := strategy.SelectBatches(decimal.Zero, batches)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "quantity must be positive")
	})

	t.Run("Returns error for negative quantity", func(t *testing.T) {
		batches := []StockBatch{createTestBatch("B001", 100, 10, nil, nil)}
		_, err := strategy.SelectBatches(decimal.NewFromFloat(-10), batches)
		assert.Error(t, err)
	})

	t.Run("Returns empty result for no batches", func(t *testing.T) {
		result, err := strategy.SelectBatches(decimal.NewFromFloat(10), []StockBatch{})
		require.NoError(t, err)
		assert.Len(t, result.Deductions, 0)
		assert.True(t, result.RemainingQuantity.Equal(decimal.NewFromFloat(10)))
		assert.False(t, result.FullyFulfilled)
	})

	t.Run("Selects single batch when sufficient", func(t *testing.T) {
		batches := []StockBatch{createTestBatch("B001", 100, 10, nil, nil)}
		result, err := strategy.SelectBatches(decimal.NewFromFloat(50), batches)
		require.NoError(t, err)
		assert.Len(t, result.Deductions, 1)
		assert.True(t, result.TotalDeducted.Equal(decimal.NewFromFloat(50)))
		assert.True(t, result.RemainingQuantity.IsZero())
		assert.True(t, result.FullyFulfilled)
		assert.Equal(t, "B001", result.Deductions[0].BatchNumber)
	})

	t.Run("Selects oldest batch first by production date", func(t *testing.T) {
		now := time.Now()
		oldDate := now.AddDate(0, -2, 0)
		newDate := now.AddDate(0, -1, 0)

		batches := []StockBatch{
			createTestBatch("B002-NEW", 100, 12, timePtr(newDate), nil),
			createTestBatch("B001-OLD", 100, 10, timePtr(oldDate), nil),
		}

		result, err := strategy.SelectBatches(decimal.NewFromFloat(50), batches)
		require.NoError(t, err)
		assert.Len(t, result.Deductions, 1)
		assert.Equal(t, "B001-OLD", result.Deductions[0].BatchNumber)
		assert.True(t, result.Deductions[0].UnitCost.Equal(decimal.NewFromFloat(10)))
	})

	t.Run("Falls back to creation date when no production date", func(t *testing.T) {
		batch1 := createTestBatch("B001-OLDER", 100, 10, nil, nil)
		batch2 := createTestBatch("B002-NEWER", 100, 12, nil, nil)
		batch2.CreatedAt = batch1.CreatedAt.Add(time.Hour)

		batches := []StockBatch{batch2, batch1}

		result, err := strategy.SelectBatches(decimal.NewFromFloat(50), batches)
		require.NoError(t, err)
		assert.Equal(t, "B001-OLDER", result.Deductions[0].BatchNumber)
	})

	t.Run("Selects multiple batches when needed", func(t *testing.T) {
		now := time.Now()
		batches := []StockBatch{
			createTestBatch("B001", 30, 10, timePtr(now.AddDate(0, -3, 0)), nil),
			createTestBatch("B002", 40, 12, timePtr(now.AddDate(0, -2, 0)), nil),
			createTestBatch("B003", 50, 15, timePtr(now.AddDate(0, -1, 0)), nil),
		}

		result, err := strategy.SelectBatches(decimal.NewFromFloat(60), batches)
		require.NoError(t, err)
		assert.Len(t, result.Deductions, 2)
		assert.True(t, result.TotalDeducted.Equal(decimal.NewFromFloat(60)))
		assert.True(t, result.FullyFulfilled)
		// First batch (30) fully consumed, second batch (30 out of 40) partially
		assert.Equal(t, "B001", result.Deductions[0].BatchNumber)
		assert.True(t, result.Deductions[0].DeductedAmount.Equal(decimal.NewFromFloat(30)))
		assert.True(t, result.Deductions[0].FullyConsumed)
		assert.Equal(t, "B002", result.Deductions[1].BatchNumber)
		assert.True(t, result.Deductions[1].DeductedAmount.Equal(decimal.NewFromFloat(30)))
		assert.False(t, result.Deductions[1].FullyConsumed)
	})

	t.Run("Handles partial fulfillment", func(t *testing.T) {
		batches := []StockBatch{
			createTestBatch("B001", 30, 10, nil, nil),
			createTestBatch("B002", 20, 12, nil, nil),
		}

		result, err := strategy.SelectBatches(decimal.NewFromFloat(100), batches)
		require.NoError(t, err)
		assert.True(t, result.TotalDeducted.Equal(decimal.NewFromFloat(50)))
		assert.True(t, result.RemainingQuantity.Equal(decimal.NewFromFloat(50)))
		assert.False(t, result.FullyFulfilled)
		assert.Len(t, result.BatchesConsumed, 2)
	})

	t.Run("Skips expired batches", func(t *testing.T) {
		now := time.Now()
		expiredDate := now.AddDate(0, 0, -1)
		futureDate := now.AddDate(0, 0, 30)

		batches := []StockBatch{
			createTestBatch("B001-EXPIRED", 100, 10, nil, timePtr(expiredDate)),
			createTestBatch("B002-VALID", 100, 12, nil, timePtr(futureDate)),
		}

		result, err := strategy.SelectBatches(decimal.NewFromFloat(50), batches)
		require.NoError(t, err)
		assert.Len(t, result.Deductions, 1)
		assert.Equal(t, "B002-VALID", result.Deductions[0].BatchNumber)
	})

	t.Run("Skips consumed batches", func(t *testing.T) {
		batch1 := createTestBatch("B001-CONSUMED", 100, 10, nil, nil)
		batch1.Consumed = true
		batch2 := createTestBatch("B002-AVAILABLE", 100, 12, nil, nil)

		batches := []StockBatch{batch1, batch2}

		result, err := strategy.SelectBatches(decimal.NewFromFloat(50), batches)
		require.NoError(t, err)
		assert.Len(t, result.Deductions, 1)
		assert.Equal(t, "B002-AVAILABLE", result.Deductions[0].BatchNumber)
	})

	t.Run("Calculates weighted average cost correctly", func(t *testing.T) {
		now := time.Now()
		batches := []StockBatch{
			createTestBatch("B001", 30, 10, timePtr(now.AddDate(0, -2, 0)), nil),  // 30 * 10 = 300
			createTestBatch("B002", 100, 20, timePtr(now.AddDate(0, -1, 0)), nil), // 20 * 20 = 400
		}

		result, err := strategy.SelectBatches(decimal.NewFromFloat(50), batches)
		require.NoError(t, err)
		// Total cost: 300 + 400 = 700
		// Total qty: 50
		// Weighted avg: 700 / 50 = 14
		assert.True(t, result.TotalCost.Equal(decimal.NewFromFloat(700)))
		assert.True(t, result.WeightedAverageCost.Equal(decimal.NewFromFloat(14)))
	})
}

func TestFEFOBatchOutboundStrategy(t *testing.T) {
	strategy := NewFEFOBatchOutboundStrategy()

	t.Run("Strategy metadata is correct", func(t *testing.T) {
		assert.Equal(t, "fefo_batch_outbound", strategy.Name())
		assert.Equal(t, BatchOutboundStrategyTypeFEFO, strategy.StrategyType())
		assert.NotEmpty(t, strategy.Description())
	})

	t.Run("Selects batch with earliest expiry first", func(t *testing.T) {
		now := time.Now()
		batches := []StockBatch{
			createTestBatch("B001-LATER", 100, 10, nil, timePtr(now.AddDate(0, 3, 0))),
			createTestBatch("B002-SOONER", 100, 12, nil, timePtr(now.AddDate(0, 1, 0))),
			createTestBatch("B003-NO-EXPIRY", 100, 15, nil, nil),
		}

		result, err := strategy.SelectBatches(decimal.NewFromFloat(50), batches)
		require.NoError(t, err)
		assert.Len(t, result.Deductions, 1)
		assert.Equal(t, "B002-SOONER", result.Deductions[0].BatchNumber)
	})

	t.Run("Batches with expiry come before those without", func(t *testing.T) {
		now := time.Now()
		batches := []StockBatch{
			createTestBatch("B001-NO-EXPIRY", 100, 10, nil, nil),
			createTestBatch("B002-HAS-EXPIRY", 100, 12, nil, timePtr(now.AddDate(0, 6, 0))),
		}

		result, err := strategy.SelectBatches(decimal.NewFromFloat(50), batches)
		require.NoError(t, err)
		assert.Equal(t, "B002-HAS-EXPIRY", result.Deductions[0].BatchNumber)
	})

	t.Run("Falls back to production date when expiry is same", func(t *testing.T) {
		now := time.Now()
		expiry := now.AddDate(0, 3, 0)
		batches := []StockBatch{
			createTestBatch("B001-NEWER", 100, 10, timePtr(now.AddDate(0, -1, 0)), timePtr(expiry)),
			createTestBatch("B002-OLDER", 100, 12, timePtr(now.AddDate(0, -3, 0)), timePtr(expiry)),
		}

		result, err := strategy.SelectBatches(decimal.NewFromFloat(50), batches)
		require.NoError(t, err)
		assert.Equal(t, "B002-OLDER", result.Deductions[0].BatchNumber)
	})

	t.Run("Handles multiple batches with mixed expiry", func(t *testing.T) {
		now := time.Now()
		batches := []StockBatch{
			createTestBatch("B001", 20, 10, nil, timePtr(now.AddDate(0, 3, 0))),
			createTestBatch("B002", 30, 12, nil, timePtr(now.AddDate(0, 1, 0))),
			createTestBatch("B003", 50, 15, nil, nil),
		}

		result, err := strategy.SelectBatches(decimal.NewFromFloat(60), batches)
		require.NoError(t, err)
		// Should pick: B002 (earliest expiry, 30), then B001 (next expiry, 30 of 20 = all 20),
		// then B003 (no expiry, 10 needed)
		assert.Len(t, result.Deductions, 3)
		assert.Equal(t, "B002", result.Deductions[0].BatchNumber)
		assert.Equal(t, "B001", result.Deductions[1].BatchNumber)
		assert.Equal(t, "B003", result.Deductions[2].BatchNumber)
	})
}

func TestSpecifiedBatchOutboundStrategy(t *testing.T) {
	t.Run("Strategy metadata is correct", func(t *testing.T) {
		requests := []BatchDeductionRequest{}
		strategy := NewSpecifiedBatchOutboundStrategy(requests)
		assert.Equal(t, "specified_batch_outbound", strategy.Name())
		assert.Equal(t, BatchOutboundStrategyTypeSpecified, strategy.StrategyType())
	})

	t.Run("Returns error when no requests provided", func(t *testing.T) {
		strategy := NewSpecifiedBatchOutboundStrategy(nil)
		batches := []StockBatch{createTestBatch("B001", 100, 10, nil, nil)}
		_, err := strategy.SelectBatches(decimal.NewFromFloat(50), batches)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "requires batch deduction requests")
	})

	t.Run("Deducts from specified batch", func(t *testing.T) {
		batch := createTestBatch("B001", 100, 10, nil, nil)
		requests := []BatchDeductionRequest{
			{BatchID: batch.ID, Quantity: decimal.NewFromFloat(30)},
		}
		strategy := NewSpecifiedBatchOutboundStrategy(requests)

		result, err := strategy.SelectBatches(decimal.NewFromFloat(30), []StockBatch{batch})
		require.NoError(t, err)
		assert.Len(t, result.Deductions, 1)
		assert.Equal(t, batch.ID, result.Deductions[0].BatchID)
		assert.True(t, result.Deductions[0].DeductedAmount.Equal(decimal.NewFromFloat(30)))
	})

	t.Run("Respects request order", func(t *testing.T) {
		batch1 := createTestBatch("B001", 50, 10, nil, nil)
		batch2 := createTestBatch("B002", 50, 15, nil, nil)

		// Request B002 first, then B001
		requests := []BatchDeductionRequest{
			{BatchID: batch2.ID, Quantity: decimal.NewFromFloat(30)},
			{BatchID: batch1.ID, Quantity: decimal.NewFromFloat(20)},
		}
		strategy := NewSpecifiedBatchOutboundStrategy(requests)

		result, err := strategy.SelectBatches(decimal.NewFromFloat(50), []StockBatch{batch1, batch2})
		require.NoError(t, err)
		assert.Len(t, result.Deductions, 2)
		assert.Equal(t, batch2.ID, result.Deductions[0].BatchID)
		assert.Equal(t, batch1.ID, result.Deductions[1].BatchID)
	})

	t.Run("Skips non-existent batch", func(t *testing.T) {
		batch := createTestBatch("B001", 100, 10, nil, nil)
		nonExistentID := uuid.New()
		requests := []BatchDeductionRequest{
			{BatchID: nonExistentID, Quantity: decimal.NewFromFloat(30)},
			{BatchID: batch.ID, Quantity: decimal.NewFromFloat(30)},
		}
		strategy := NewSpecifiedBatchOutboundStrategy(requests)

		result, err := strategy.SelectBatches(decimal.NewFromFloat(30), []StockBatch{batch})
		require.NoError(t, err)
		assert.Len(t, result.Deductions, 1)
		assert.Equal(t, batch.ID, result.Deductions[0].BatchID)
	})

	t.Run("Deducts max available when quantity is zero", func(t *testing.T) {
		batch := createTestBatch("B001", 100, 10, nil, nil)
		requests := []BatchDeductionRequest{
			{BatchID: batch.ID, Quantity: decimal.Zero}, // Zero means take as much as possible
		}
		strategy := NewSpecifiedBatchOutboundStrategy(requests)

		result, err := strategy.SelectBatches(decimal.NewFromFloat(50), []StockBatch{batch})
		require.NoError(t, err)
		assert.True(t, result.TotalDeducted.Equal(decimal.NewFromFloat(50)))
	})

	t.Run("Caps at available quantity", func(t *testing.T) {
		batch := createTestBatch("B001", 30, 10, nil, nil)
		requests := []BatchDeductionRequest{
			{BatchID: batch.ID, Quantity: decimal.NewFromFloat(100)}, // More than available
		}
		strategy := NewSpecifiedBatchOutboundStrategy(requests)

		result, err := strategy.SelectBatches(decimal.NewFromFloat(100), []StockBatch{batch})
		require.NoError(t, err)
		assert.True(t, result.TotalDeducted.Equal(decimal.NewFromFloat(30)))
		assert.True(t, result.RemainingQuantity.Equal(decimal.NewFromFloat(70)))
		assert.False(t, result.FullyFulfilled)
	})

	t.Run("GetRequests returns configured requests", func(t *testing.T) {
		requests := []BatchDeductionRequest{
			{BatchID: uuid.New(), Quantity: decimal.NewFromFloat(10)},
		}
		strategy := NewSpecifiedBatchOutboundStrategy(requests)
		assert.Equal(t, requests, strategy.GetRequests())
	})
}

func TestBatchOutboundStrategyFactory(t *testing.T) {
	factory := NewBatchOutboundStrategyFactory()

	t.Run("CreateFIFOStrategy returns FIFO strategy", func(t *testing.T) {
		strategy := factory.CreateFIFOStrategy()
		assert.NotNil(t, strategy)
		assert.Equal(t, BatchOutboundStrategyTypeFIFO, strategy.StrategyType())
	})

	t.Run("CreateFEFOStrategy returns FEFO strategy", func(t *testing.T) {
		strategy := factory.CreateFEFOStrategy()
		assert.NotNil(t, strategy)
		assert.Equal(t, BatchOutboundStrategyTypeFEFO, strategy.StrategyType())
	})

	t.Run("CreateSpecifiedStrategy returns specified strategy", func(t *testing.T) {
		requests := []BatchDeductionRequest{{BatchID: uuid.New()}}
		strategy := factory.CreateSpecifiedStrategy(requests)
		assert.NotNil(t, strategy)
		assert.Equal(t, BatchOutboundStrategyTypeSpecified, strategy.StrategyType())
	})

	t.Run("GetStrategy returns correct strategy", func(t *testing.T) {
		fifo, err := factory.GetStrategy(BatchOutboundStrategyTypeFIFO, nil)
		require.NoError(t, err)
		assert.Equal(t, BatchOutboundStrategyTypeFIFO, fifo.StrategyType())

		fefo, err := factory.GetStrategy(BatchOutboundStrategyTypeFEFO, nil)
		require.NoError(t, err)
		assert.Equal(t, BatchOutboundStrategyTypeFEFO, fefo.StrategyType())

		requests := []BatchDeductionRequest{{BatchID: uuid.New()}}
		specified, err := factory.GetStrategy(BatchOutboundStrategyTypeSpecified, requests)
		require.NoError(t, err)
		assert.Equal(t, BatchOutboundStrategyTypeSpecified, specified.StrategyType())
	})

	t.Run("GetStrategy returns error for specified without requests", func(t *testing.T) {
		_, err := factory.GetStrategy(BatchOutboundStrategyTypeSpecified, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "requires batch deduction requests")
	})

	t.Run("GetStrategy returns error for invalid type", func(t *testing.T) {
		_, err := factory.GetStrategy(BatchOutboundStrategyType("INVALID"), nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Unknown batch outbound strategy type")
	})

	t.Run("GetDefaultStrategy returns FIFO", func(t *testing.T) {
		strategy := factory.GetDefaultStrategy()
		assert.Equal(t, BatchOutboundStrategyTypeFIFO, strategy.StrategyType())
	})
}

func TestValidateBatchAvailability(t *testing.T) {
	t.Run("Returns true when sufficient stock", func(t *testing.T) {
		batches := []StockBatch{
			createTestBatch("B001", 50, 10, nil, nil),
			createTestBatch("B002", 50, 10, nil, nil),
		}
		sufficient, total := ValidateBatchAvailability(batches, decimal.NewFromFloat(80))
		assert.True(t, sufficient)
		assert.True(t, total.Equal(decimal.NewFromFloat(100)))
	})

	t.Run("Returns false when insufficient stock", func(t *testing.T) {
		batches := []StockBatch{
			createTestBatch("B001", 50, 10, nil, nil),
		}
		sufficient, total := ValidateBatchAvailability(batches, decimal.NewFromFloat(80))
		assert.False(t, sufficient)
		assert.True(t, total.Equal(decimal.NewFromFloat(50)))
	})

	t.Run("Excludes unavailable batches", func(t *testing.T) {
		now := time.Now()
		expiredBatch := createTestBatch("B001-EXPIRED", 50, 10, nil, timePtr(now.AddDate(0, 0, -1)))
		consumedBatch := createTestBatch("B002-CONSUMED", 50, 10, nil, nil)
		consumedBatch.Consumed = true
		validBatch := createTestBatch("B003-VALID", 30, 10, nil, nil)

		batches := []StockBatch{expiredBatch, consumedBatch, validBatch}
		sufficient, total := ValidateBatchAvailability(batches, decimal.NewFromFloat(30))
		assert.True(t, sufficient)
		assert.True(t, total.Equal(decimal.NewFromFloat(30)))
	})
}

func TestGetBatchesByExpiryWindow(t *testing.T) {
	now := time.Now()

	t.Run("Returns batches expiring within window", func(t *testing.T) {
		batches := []StockBatch{
			createTestBatch("B001", 50, 10, nil, timePtr(now.Add(5*24*time.Hour))),  // 5 days
			createTestBatch("B002", 50, 10, nil, timePtr(now.Add(15*24*time.Hour))), // 15 days
			createTestBatch("B003", 50, 10, nil, timePtr(now.Add(45*24*time.Hour))), // 45 days
			createTestBatch("B004", 50, 10, nil, nil),                               // No expiry
		}

		// Get batches expiring within 30 days
		expiring := GetBatchesByExpiryWindow(batches, 30*24*time.Hour)
		assert.Len(t, expiring, 2)
		assert.Equal(t, "B001", expiring[0].BatchNumber)
		assert.Equal(t, "B002", expiring[1].BatchNumber)
	})

	t.Run("Returns empty for no expiring batches", func(t *testing.T) {
		batches := []StockBatch{
			createTestBatch("B001", 50, 10, nil, timePtr(now.Add(60*24*time.Hour))), // 60 days
			createTestBatch("B002", 50, 10, nil, nil),                               // No expiry
		}

		expiring := GetBatchesByExpiryWindow(batches, 30*24*time.Hour)
		assert.Len(t, expiring, 0)
	})

	t.Run("Excludes unavailable batches", func(t *testing.T) {
		consumedBatch := createTestBatch("B001", 50, 10, nil, timePtr(now.Add(5*24*time.Hour)))
		consumedBatch.Consumed = true
		validBatch := createTestBatch("B002", 50, 10, nil, timePtr(now.Add(10*24*time.Hour)))

		batches := []StockBatch{consumedBatch, validBatch}
		expiring := GetBatchesByExpiryWindow(batches, 30*24*time.Hour)
		assert.Len(t, expiring, 1)
		assert.Equal(t, "B002", expiring[0].BatchNumber)
	})
}

func TestApplyBatchDeductions(t *testing.T) {
	t.Run("Applies deductions to batches", func(t *testing.T) {
		batch1 := createTestBatch("B001", 50, 10, nil, nil)
		batch2 := createTestBatch("B002", 30, 15, nil, nil)
		batches := []*StockBatch{&batch1, &batch2}

		result := &BatchOutboundResult{
			Deductions: []BatchDeductionResult{
				{BatchID: batch1.ID, DeductedAmount: decimal.NewFromFloat(30)},
				{BatchID: batch2.ID, DeductedAmount: decimal.NewFromFloat(20)},
			},
		}

		err := ApplyBatchDeductions(batches, result)
		require.NoError(t, err)
		assert.True(t, batch1.Quantity.Equal(decimal.NewFromFloat(20)))
		assert.True(t, batch2.Quantity.Equal(decimal.NewFromFloat(10)))
	})

	t.Run("Returns error for nil result", func(t *testing.T) {
		err := ApplyBatchDeductions(nil, nil)
		assert.Error(t, err)
	})

	t.Run("Returns error for non-existent batch", func(t *testing.T) {
		batch := createTestBatch("B001", 50, 10, nil, nil)
		batches := []*StockBatch{&batch}

		result := &BatchOutboundResult{
			Deductions: []BatchDeductionResult{
				{BatchID: uuid.New(), DeductedAmount: decimal.NewFromFloat(30)},
			},
		}

		err := ApplyBatchDeductions(batches, result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Batch not found")
	})
}
