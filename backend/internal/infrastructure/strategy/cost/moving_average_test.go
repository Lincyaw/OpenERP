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

func TestNewMovingAverageCostStrategy(t *testing.T) {
	s := NewMovingAverageCostStrategy()

	assert.NotNil(t, s)
	assert.Equal(t, "moving_average", s.Name())
	assert.Equal(t, strategy.StrategyTypeCost, s.Type())
	assert.Equal(t, strategy.CostMethodMovingAverage, s.Method())
	assert.NotEmpty(t, s.Description())
}

func TestMovingAverageCostStrategy_CalculateAverageCost(t *testing.T) {
	s := NewMovingAverageCostStrategy()
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
			name: "multiple entries with same cost",
			entries: []strategy.StockEntry{
				{
					ID:        "1",
					Quantity:  decimal.NewFromInt(50),
					UnitCost:  decimal.NewFromFloat(10.00),
					TotalCost: decimal.NewFromFloat(500.00),
				},
				{
					ID:        "2",
					Quantity:  decimal.NewFromInt(50),
					UnitCost:  decimal.NewFromFloat(10.00),
					TotalCost: decimal.NewFromFloat(500.00),
				},
			},
			expected:    decimal.NewFromFloat(10.00),
			expectError: false,
		},
		{
			name: "multiple entries with different costs - weighted average",
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
			// Total: 3000 / 200 = 15.00
			expected:    decimal.NewFromFloat(15.00),
			expectError: false,
		},
		{
			name: "weighted average with unequal quantities",
			entries: []strategy.StockEntry{
				{
					ID:        "1",
					Quantity:  decimal.NewFromInt(100),
					UnitCost:  decimal.NewFromFloat(10.00),
					TotalCost: decimal.NewFromFloat(1000.00),
				},
				{
					ID:        "2",
					Quantity:  decimal.NewFromInt(50),
					UnitCost:  decimal.NewFromFloat(20.00),
					TotalCost: decimal.NewFromFloat(1000.00),
				},
			},
			// Total: 2000 / 150 = 13.333...
			expected:    decimal.NewFromFloat(2000).Div(decimal.NewFromInt(150)),
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

func TestMovingAverageCostStrategy_CalculateCost(t *testing.T) {
	s := NewMovingAverageCostStrategy()
	ctx := context.Background()
	now := time.Now()

	tests := []struct {
		name         string
		costCtx      strategy.CostContext
		entries      []strategy.StockEntry
		wantUnitCost decimal.Decimal
		wantTotal    decimal.Decimal
		wantError    bool
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
			name: "single entry - calculate cost for partial quantity",
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
			wantUnitCost: decimal.NewFromFloat(10.00),
			wantTotal:    decimal.NewFromFloat(500.00), // 50 * 10.00
			wantError:    false,
		},
		{
			name: "multiple entries - uses weighted average",
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
					EntryDate:   now.Add(-time.Hour),
				},
				{
					ID:          "2",
					ProductID:   "product1",
					WarehouseID: "wh1",
					Quantity:    decimal.NewFromInt(100),
					UnitCost:    decimal.NewFromFloat(20.00),
					TotalCost:   decimal.NewFromFloat(2000.00),
					EntryDate:   now,
				},
			},
			// Average: 3000/200 = 15.00, Total: 100 * 15 = 1500
			wantUnitCost: decimal.NewFromFloat(15.00),
			wantTotal:    decimal.NewFromFloat(1500.00),
			wantError:    false,
		},
		{
			name: "request more than available - still uses average cost",
			costCtx: strategy.CostContext{
				TenantID:    "tenant1",
				ProductID:   "product1",
				WarehouseID: "wh1",
				Quantity:    decimal.NewFromInt(500), // More than 200 available
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
				{
					ID:          "2",
					ProductID:   "product1",
					WarehouseID: "wh1",
					Quantity:    decimal.NewFromInt(100),
					UnitCost:    decimal.NewFromFloat(20.00),
					TotalCost:   decimal.NewFromFloat(2000.00),
					EntryDate:   now,
				},
			},
			// Average cost: 15.00, Total: 500 * 15 = 7500
			wantUnitCost: decimal.NewFromFloat(15.00),
			wantTotal:    decimal.NewFromFloat(7500.00),
			wantError:    false,
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
			assert.Equal(t, strategy.CostMethodMovingAverage, result.Method)
			assert.True(t, tt.wantUnitCost.Equal(result.UnitCost),
				"expected unit cost %s but got %s", tt.wantUnitCost.String(), result.UnitCost.String())
			assert.True(t, tt.wantTotal.Equal(result.TotalCost),
				"expected total cost %s but got %s", tt.wantTotal.String(), result.TotalCost.String())
			assert.True(t, result.RemainingQty.IsZero())
			assert.Len(t, result.EntriesUsed, len(tt.entries))
		})
	}
}

func TestMovingAverageCostStrategy_RealWorldScenario(t *testing.T) {
	// Simulate a real-world inventory cost scenario
	s := NewMovingAverageCostStrategy()
	ctx := context.Background()
	now := time.Now()

	// Scenario: Product has been purchased at different prices over time
	// Purchase 1: 100 units at $10.00
	// Purchase 2: 200 units at $12.00
	// Purchase 3: 50 units at $15.00
	// Total: 350 units, Total Cost: $1000 + $2400 + $750 = $4150
	// Average Cost: $4150 / 350 = $11.857...

	entries := []strategy.StockEntry{
		{
			ID:            "pur-001",
			ProductID:     "prod-001",
			WarehouseID:   "wh-001",
			Quantity:      decimal.NewFromInt(100),
			UnitCost:      decimal.NewFromFloat(10.00),
			TotalCost:     decimal.NewFromFloat(1000.00),
			EntryDate:     now.Add(-72 * time.Hour),
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
			EntryDate:     now.Add(-48 * time.Hour),
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
			EntryDate:     now.Add(-24 * time.Hour),
			BatchNumber:   "BATCH-003",
			ReferenceID:   "PO-003",
			ReferenceType: "purchase_order",
		},
	}

	// Calculate average cost
	avgCost, err := s.CalculateAverageCost(ctx, entries)
	require.NoError(t, err)

	// Expected: 4150 / 350 = 11.857142857142857
	expectedAvg := decimal.NewFromFloat(4150.00).Div(decimal.NewFromInt(350))
	assert.True(t, expectedAvg.Equal(avgCost),
		"expected average cost %s but got %s", expectedAvg.String(), avgCost.String())

	// Now calculate cost for selling 100 units
	costCtx := strategy.CostContext{
		TenantID:    "tenant1",
		ProductID:   "prod-001",
		WarehouseID: "wh-001",
		Quantity:    decimal.NewFromInt(100),
		Date:        now,
	}

	result, err := s.CalculateCost(ctx, costCtx, entries)
	require.NoError(t, err)

	// Cost for 100 units at average price
	expectedTotal := avgCost.Mul(decimal.NewFromInt(100))
	assert.True(t, result.UnitCost.Equal(avgCost))
	assert.True(t, result.TotalCost.Round(4).Equal(expectedTotal.Round(4)),
		"expected total cost %s but got %s", expectedTotal.String(), result.TotalCost.String())
}
