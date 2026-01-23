package cost

import (
	"context"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFIFOCostStrategy(t *testing.T) {
	s := NewFIFOCostStrategy()

	assert.NotNil(t, s)
	assert.Equal(t, "fifo", s.Name())
	assert.Equal(t, strategy.StrategyTypeCost, s.Type())
	assert.Equal(t, strategy.CostMethodFIFO, s.Method())
	assert.NotEmpty(t, s.Description())
}

func TestFIFOCostStrategy_CalculateAverageCost(t *testing.T) {
	s := NewFIFOCostStrategy()
	ctx := context.Background()

	tests := []struct {
		name        string
		entries     []strategy.StockEntry
		expected    decimal.Decimal
		expectError bool
	}{
		{
			name:        "empty entries",
			entries:     []strategy.StockEntry{},
			expected:    decimal.Zero,
			expectError: true,
		},
		{
			name: "single entry",
			entries: []strategy.StockEntry{
				{
					ID:        "1",
					Quantity:  decimal.NewFromInt(100),
					UnitCost:  decimal.NewFromFloat(10.00),
					TotalCost: decimal.NewFromFloat(1000.00),
				},
			},
			expected:    decimal.NewFromFloat(10.00),
			expectError: false,
		},
		{
			name: "multiple entries - weighted average",
			entries: []strategy.StockEntry{
				{
					ID:        "1",
					Quantity:  decimal.NewFromInt(100),
					UnitCost:  decimal.NewFromFloat(10.00),
					TotalCost: decimal.NewFromFloat(1000.00),
				},
				{
					ID:        "2",
					Quantity:  decimal.NewFromInt(100),
					UnitCost:  decimal.NewFromFloat(20.00),
					TotalCost: decimal.NewFromFloat(2000.00),
				},
			},
			// For average calculation (reporting), uses weighted average
			expected:    decimal.NewFromFloat(15.00),
			expectError: false,
		},
		{
			name: "zero total quantity",
			entries: []strategy.StockEntry{
				{
					ID:        "1",
					Quantity:  decimal.Zero,
					UnitCost:  decimal.NewFromFloat(10.00),
					TotalCost: decimal.Zero,
				},
			},
			expected:    decimal.Zero,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := s.CalculateAverageCost(ctx, tt.entries)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.True(t, tt.expected.Equal(result),
					"expected %s but got %s", tt.expected.String(), result.String())
			}
		})
	}
}

func TestFIFOCostStrategy_CalculateCost(t *testing.T) {
	s := NewFIFOCostStrategy()
	ctx := context.Background()
	now := time.Now()

	tests := []struct {
		name          string
		costCtx       strategy.CostContext
		entries       []strategy.StockEntry
		wantUnitCost  decimal.Decimal
		wantTotal     decimal.Decimal
		wantRemaining decimal.Decimal
		wantUsedCount int
		wantError     bool
	}{
		{
			name: "empty entries",
			costCtx: strategy.CostContext{
				Quantity: decimal.NewFromInt(10),
			},
			entries:   []strategy.StockEntry{},
			wantError: true,
		},
		{
			name: "single entry - request partial quantity",
			costCtx: strategy.CostContext{
				TenantID:    "tenant1",
				ProductID:   "product1",
				WarehouseID: "wh1",
				Quantity:    decimal.NewFromInt(50),
				Date:        now,
			},
			entries: []strategy.StockEntry{
				{
					ID:          "1",
					ProductID:   "product1",
					WarehouseID: "wh1",
					Quantity:    decimal.NewFromInt(100),
					UnitCost:    decimal.NewFromFloat(10.00),
					TotalCost:   decimal.NewFromFloat(1000.00),
					EntryDate:   now,
				},
			},
			wantUnitCost:  decimal.NewFromFloat(10.00),
			wantTotal:     decimal.NewFromFloat(500.00),
			wantRemaining: decimal.Zero,
			wantUsedCount: 1,
			wantError:     false,
		},
		{
			name: "FIFO - oldest stock consumed first",
			costCtx: strategy.CostContext{
				TenantID:    "tenant1",
				ProductID:   "product1",
				WarehouseID: "wh1",
				Quantity:    decimal.NewFromInt(100),
				Date:        now,
			},
			entries: []strategy.StockEntry{
				{
					ID:          "1",
					ProductID:   "product1",
					WarehouseID: "wh1",
					Quantity:    decimal.NewFromInt(100),
					UnitCost:    decimal.NewFromFloat(10.00),
					TotalCost:   decimal.NewFromFloat(1000.00),
					EntryDate:   now.Add(-48 * time.Hour), // Older
				},
				{
					ID:          "2",
					ProductID:   "product1",
					WarehouseID: "wh1",
					Quantity:    decimal.NewFromInt(100),
					UnitCost:    decimal.NewFromFloat(20.00),
					TotalCost:   decimal.NewFromFloat(2000.00),
					EntryDate:   now.Add(-24 * time.Hour), // Newer
				},
			},
			// FIFO uses oldest first: 100 units at $10.00
			wantUnitCost:  decimal.NewFromFloat(10.00),
			wantTotal:     decimal.NewFromFloat(1000.00),
			wantRemaining: decimal.Zero,
			wantUsedCount: 1,
			wantError:     false,
		},
		{
			name: "FIFO - spans multiple batches",
			costCtx: strategy.CostContext{
				TenantID:    "tenant1",
				ProductID:   "product1",
				WarehouseID: "wh1",
				Quantity:    decimal.NewFromInt(150),
				Date:        now,
			},
			entries: []strategy.StockEntry{
				{
					ID:          "1",
					ProductID:   "product1",
					WarehouseID: "wh1",
					Quantity:    decimal.NewFromInt(100),
					UnitCost:    decimal.NewFromFloat(10.00),
					TotalCost:   decimal.NewFromFloat(1000.00),
					EntryDate:   now.Add(-48 * time.Hour), // Oldest
				},
				{
					ID:          "2",
					ProductID:   "product1",
					WarehouseID: "wh1",
					Quantity:    decimal.NewFromInt(100),
					UnitCost:    decimal.NewFromFloat(20.00),
					TotalCost:   decimal.NewFromFloat(2000.00),
					EntryDate:   now.Add(-24 * time.Hour), // Newer
				},
			},
			// FIFO: 100 @ $10 + 50 @ $20 = $1000 + $1000 = $2000
			// Unit cost: $2000 / 150 = $13.333...
			wantUnitCost:  decimal.NewFromFloat(2000.00).Div(decimal.NewFromInt(150)),
			wantTotal:     decimal.NewFromFloat(2000.00),
			wantRemaining: decimal.Zero,
			wantUsedCount: 2,
			wantError:     false,
		},
		{
			name: "FIFO - request more than available",
			costCtx: strategy.CostContext{
				TenantID:    "tenant1",
				ProductID:   "product1",
				WarehouseID: "wh1",
				Quantity:    decimal.NewFromInt(300), // More than 200 available
				Date:        now,
			},
			entries: []strategy.StockEntry{
				{
					ID:          "1",
					ProductID:   "product1",
					WarehouseID: "wh1",
					Quantity:    decimal.NewFromInt(100),
					UnitCost:    decimal.NewFromFloat(10.00),
					TotalCost:   decimal.NewFromFloat(1000.00),
					EntryDate:   now.Add(-48 * time.Hour),
				},
				{
					ID:          "2",
					ProductID:   "product1",
					WarehouseID: "wh1",
					Quantity:    decimal.NewFromInt(100),
					UnitCost:    decimal.NewFromFloat(20.00),
					TotalCost:   decimal.NewFromFloat(2000.00),
					EntryDate:   now.Add(-24 * time.Hour),
				},
			},
			// Only 200 available: 100 @ $10 + 100 @ $20 = $3000
			// Unit cost: $3000 / 200 = $15
			wantUnitCost:  decimal.NewFromFloat(15.00),
			wantTotal:     decimal.NewFromFloat(3000.00),
			wantRemaining: decimal.NewFromInt(100), // 300 - 200 = 100 remaining
			wantUsedCount: 2,
			wantError:     false,
		},
		{
			name: "FIFO - entries in wrong order (should sort by date)",
			costCtx: strategy.CostContext{
				TenantID:    "tenant1",
				ProductID:   "product1",
				WarehouseID: "wh1",
				Quantity:    decimal.NewFromInt(50),
				Date:        now,
			},
			entries: []strategy.StockEntry{
				{
					ID:          "2",
					ProductID:   "product1",
					WarehouseID: "wh1",
					Quantity:    decimal.NewFromInt(100),
					UnitCost:    decimal.NewFromFloat(20.00),
					TotalCost:   decimal.NewFromFloat(2000.00),
					EntryDate:   now.Add(-24 * time.Hour), // Newer, but listed first
				},
				{
					ID:          "1",
					ProductID:   "product1",
					WarehouseID: "wh1",
					Quantity:    decimal.NewFromInt(100),
					UnitCost:    decimal.NewFromFloat(10.00),
					TotalCost:   decimal.NewFromFloat(1000.00),
					EntryDate:   now.Add(-48 * time.Hour), // Older
				},
			},
			// FIFO should sort and use oldest first: 50 @ $10 = $500
			wantUnitCost:  decimal.NewFromFloat(10.00),
			wantTotal:     decimal.NewFromFloat(500.00),
			wantRemaining: decimal.Zero,
			wantUsedCount: 1,
			wantError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := s.CalculateCost(ctx, tt.costCtx, tt.entries)

			if tt.wantError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, strategy.CostMethodFIFO, result.Method)
			assert.True(t, tt.wantUnitCost.Round(4).Equal(result.UnitCost.Round(4)),
				"expected unit cost %s but got %s", tt.wantUnitCost.String(), result.UnitCost.String())
			assert.True(t, tt.wantTotal.Round(4).Equal(result.TotalCost.Round(4)),
				"expected total cost %s but got %s", tt.wantTotal.String(), result.TotalCost.String())
			assert.True(t, tt.wantRemaining.Equal(result.RemainingQty),
				"expected remaining %s but got %s", tt.wantRemaining.String(), result.RemainingQty.String())
			assert.Len(t, result.EntriesUsed, tt.wantUsedCount)
		})
	}
}

func TestFIFOCostStrategy_RealWorldScenario(t *testing.T) {
	// Simulate a real-world FIFO inventory cost scenario
	s := NewFIFOCostStrategy()
	ctx := context.Background()
	now := time.Now()

	// Scenario: Product purchased at different prices over time
	// Day 1: Purchase 100 units at $10.00
	// Day 2: Purchase 200 units at $12.00
	// Day 3: Purchase 50 units at $15.00
	// Total: 350 units available

	entries := []strategy.StockEntry{
		{
			ID:            "pur-001",
			ProductID:     "prod-001",
			WarehouseID:   "wh-001",
			Quantity:      decimal.NewFromInt(100),
			UnitCost:      decimal.NewFromFloat(10.00),
			TotalCost:     decimal.NewFromFloat(1000.00),
			EntryDate:     now.Add(-72 * time.Hour), // 3 days ago
			BatchNumber:   "BATCH-001",
			ReferenceID:   "PO-001",
			ReferenceType: "purchase_order",
		},
		{
			ID:            "pur-002",
			ProductID:     "prod-001",
			WarehouseID:   "wh-001",
			Quantity:      decimal.NewFromInt(200),
			UnitCost:      decimal.NewFromFloat(12.00),
			TotalCost:     decimal.NewFromFloat(2400.00),
			EntryDate:     now.Add(-48 * time.Hour), // 2 days ago
			BatchNumber:   "BATCH-002",
			ReferenceID:   "PO-002",
			ReferenceType: "purchase_order",
		},
		{
			ID:            "pur-003",
			ProductID:     "prod-001",
			WarehouseID:   "wh-001",
			Quantity:      decimal.NewFromInt(50),
			UnitCost:      decimal.NewFromFloat(15.00),
			TotalCost:     decimal.NewFromFloat(750.00),
			EntryDate:     now.Add(-24 * time.Hour), // 1 day ago
			BatchNumber:   "BATCH-003",
			ReferenceID:   "PO-003",
			ReferenceType: "purchase_order",
		},
	}

	// Test 1: Sell 50 units - should use only first batch
	t.Run("sell 50 units - first batch only", func(t *testing.T) {
		costCtx := strategy.CostContext{
			TenantID:    "tenant1",
			ProductID:   "prod-001",
			WarehouseID: "wh-001",
			Quantity:    decimal.NewFromInt(50),
			Date:        now,
		}

		result, err := s.CalculateCost(ctx, costCtx, entries)
		require.NoError(t, err)

		// Should use 50 from first batch at $10.00
		assert.True(t, decimal.NewFromFloat(10.00).Equal(result.UnitCost))
		assert.True(t, decimal.NewFromFloat(500.00).Equal(result.TotalCost))
		assert.True(t, result.RemainingQty.IsZero())
		assert.Len(t, result.EntriesUsed, 1)
	})

	// Test 2: Sell 100 units - should exhaust first batch exactly
	t.Run("sell 100 units - exhaust first batch", func(t *testing.T) {
		costCtx := strategy.CostContext{
			TenantID:    "tenant1",
			ProductID:   "prod-001",
			WarehouseID: "wh-001",
			Quantity:    decimal.NewFromInt(100),
			Date:        now,
		}

		result, err := s.CalculateCost(ctx, costCtx, entries)
		require.NoError(t, err)

		// Should use all 100 from first batch at $10.00
		assert.True(t, decimal.NewFromFloat(10.00).Equal(result.UnitCost))
		assert.True(t, decimal.NewFromFloat(1000.00).Equal(result.TotalCost))
		assert.True(t, result.RemainingQty.IsZero())
		assert.Len(t, result.EntriesUsed, 1)
	})

	// Test 3: Sell 150 units - spans two batches
	t.Run("sell 150 units - spans two batches", func(t *testing.T) {
		costCtx := strategy.CostContext{
			TenantID:    "tenant1",
			ProductID:   "prod-001",
			WarehouseID: "wh-001",
			Quantity:    decimal.NewFromInt(150),
			Date:        now,
		}

		result, err := s.CalculateCost(ctx, costCtx, entries)
		require.NoError(t, err)

		// Should use: 100 @ $10 + 50 @ $12 = $1000 + $600 = $1600
		// Average: $1600 / 150 = $10.6666...
		expectedTotal := decimal.NewFromFloat(1600.00)
		expectedUnitCost := expectedTotal.Div(decimal.NewFromInt(150))

		assert.True(t, expectedUnitCost.Round(4).Equal(result.UnitCost.Round(4)))
		assert.True(t, expectedTotal.Equal(result.TotalCost))
		assert.True(t, result.RemainingQty.IsZero())
		assert.Len(t, result.EntriesUsed, 2)
	})

	// Test 4: Sell 320 units - spans all batches
	t.Run("sell 320 units - spans all batches", func(t *testing.T) {
		costCtx := strategy.CostContext{
			TenantID:    "tenant1",
			ProductID:   "prod-001",
			WarehouseID: "wh-001",
			Quantity:    decimal.NewFromInt(320),
			Date:        now,
		}

		result, err := s.CalculateCost(ctx, costCtx, entries)
		require.NoError(t, err)

		// Should use: 100 @ $10 + 200 @ $12 + 20 @ $15 = $1000 + $2400 + $300 = $3700
		// Average: $3700 / 320 = $11.5625
		expectedTotal := decimal.NewFromFloat(3700.00)
		expectedUnitCost := expectedTotal.Div(decimal.NewFromInt(320))

		assert.True(t, expectedUnitCost.Round(4).Equal(result.UnitCost.Round(4)))
		assert.True(t, expectedTotal.Equal(result.TotalCost))
		assert.True(t, result.RemainingQty.IsZero())
		assert.Len(t, result.EntriesUsed, 3)
	})

	// Test 5: Sell 400 units - more than available
	t.Run("sell 400 units - more than available", func(t *testing.T) {
		costCtx := strategy.CostContext{
			TenantID:    "tenant1",
			ProductID:   "prod-001",
			WarehouseID: "wh-001",
			Quantity:    decimal.NewFromInt(400), // Only 350 available
			Date:        now,
		}

		result, err := s.CalculateCost(ctx, costCtx, entries)
		require.NoError(t, err)

		// Should use all available: 100 @ $10 + 200 @ $12 + 50 @ $15 = $1000 + $2400 + $750 = $4150
		// Average: $4150 / 350 = $11.857...
		// Remaining: 400 - 350 = 50
		expectedTotal := decimal.NewFromFloat(4150.00)
		expectedUnitCost := expectedTotal.Div(decimal.NewFromInt(350))

		assert.True(t, expectedUnitCost.Round(4).Equal(result.UnitCost.Round(4)))
		assert.True(t, expectedTotal.Equal(result.TotalCost))
		assert.True(t, decimal.NewFromInt(50).Equal(result.RemainingQty))
		assert.Len(t, result.EntriesUsed, 3)
	})
}

func TestFIFOCostStrategy_CompareWithMovingAverage(t *testing.T) {
	// Compare FIFO vs Moving Average to demonstrate the difference
	fifo := NewFIFOCostStrategy()
	movingAvg := NewMovingAverageCostStrategy()
	ctx := context.Background()
	now := time.Now()

	// Entries: Prices increasing over time
	// This demonstrates when FIFO gives lower COGS (rising prices)
	entries := []strategy.StockEntry{
		{
			ID:        "1",
			Quantity:  decimal.NewFromInt(100),
			UnitCost:  decimal.NewFromFloat(10.00),
			TotalCost: decimal.NewFromFloat(1000.00),
			EntryDate: now.Add(-48 * time.Hour),
		},
		{
			ID:        "2",
			Quantity:  decimal.NewFromInt(100),
			UnitCost:  decimal.NewFromFloat(20.00),
			TotalCost: decimal.NewFromFloat(2000.00),
			EntryDate: now.Add(-24 * time.Hour),
		},
	}

	costCtx := strategy.CostContext{
		Quantity: decimal.NewFromInt(50),
		Date:     now,
	}

	fifoResult, err := fifo.CalculateCost(ctx, costCtx, entries)
	require.NoError(t, err)

	movingAvgResult, err := movingAvg.CalculateCost(ctx, costCtx, entries)
	require.NoError(t, err)

	// With rising prices:
	// - FIFO uses oldest (cheaper) stock first: $10.00 * 50 = $500
	// - Moving average: $15.00 * 50 = $750
	assert.True(t, fifoResult.TotalCost.LessThan(movingAvgResult.TotalCost),
		"FIFO cost (%s) should be less than Moving Average cost (%s) with rising prices",
		fifoResult.TotalCost.String(), movingAvgResult.TotalCost.String())

	// Verify specific values
	assert.True(t, decimal.NewFromFloat(500.00).Equal(fifoResult.TotalCost))
	assert.True(t, decimal.NewFromFloat(750.00).Equal(movingAvgResult.TotalCost))
}
