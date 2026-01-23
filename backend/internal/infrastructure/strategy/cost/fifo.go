package cost

import (
	"context"
	"errors"
	"sort"

	"github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/shopspring/decimal"
)

// FIFOCostStrategy implements First-In-First-Out cost calculation
type FIFOCostStrategy struct {
	strategy.BaseStrategy
}

// NewFIFOCostStrategy creates a new FIFO cost strategy
func NewFIFOCostStrategy() *FIFOCostStrategy {
	return &FIFOCostStrategy{
		BaseStrategy: strategy.NewBaseStrategy(
			"fifo",
			strategy.StrategyTypeCost,
			"First-In-First-Out cost calculation",
		),
	}
}

// Method returns the costing method
func (s *FIFOCostStrategy) Method() strategy.CostMethod {
	return strategy.CostMethodFIFO
}

// CalculateCost calculates the cost using FIFO method
func (s *FIFOCostStrategy) CalculateCost(
	ctx context.Context,
	costCtx strategy.CostContext,
	entries []strategy.StockEntry,
) (strategy.CostResult, error) {
	if len(entries) == 0 {
		return strategy.CostResult{}, errors.New("no stock entries provided")
	}

	// Sort entries by entry date (oldest first)
	sortedEntries := make([]strategy.StockEntry, len(entries))
	copy(sortedEntries, entries)
	sort.Slice(sortedEntries, func(i, j int) bool {
		return sortedEntries[i].EntryDate.Before(sortedEntries[j].EntryDate)
	})

	remainingQty := costCtx.Quantity
	totalCost := decimal.Zero
	entriesUsed := make([]strategy.StockEntry, 0)

	for _, entry := range sortedEntries {
		if remainingQty.IsZero() {
			break
		}

		usedQty := decimal.Min(remainingQty, entry.Quantity)
		entryCost := usedQty.Mul(entry.UnitCost)
		totalCost = totalCost.Add(entryCost)
		remainingQty = remainingQty.Sub(usedQty)
		entriesUsed = append(entriesUsed, entry)
	}

	usedQty := costCtx.Quantity.Sub(remainingQty)
	var unitCost decimal.Decimal
	if !usedQty.IsZero() {
		unitCost = totalCost.Div(usedQty)
	}

	return strategy.CostResult{
		UnitCost:     unitCost,
		TotalCost:    totalCost,
		Method:       strategy.CostMethodFIFO,
		EntriesUsed:  entriesUsed,
		RemainingQty: remainingQty,
	}, nil
}

// CalculateAverageCost calculates the weighted average cost (for reporting)
func (s *FIFOCostStrategy) CalculateAverageCost(
	ctx context.Context,
	entries []strategy.StockEntry,
) (decimal.Decimal, error) {
	if len(entries) == 0 {
		return decimal.Zero, errors.New("no stock entries provided")
	}

	totalQty := decimal.Zero
	totalCost := decimal.Zero

	for _, entry := range entries {
		totalQty = totalQty.Add(entry.Quantity)
		totalCost = totalCost.Add(entry.TotalCost)
	}

	if totalQty.IsZero() {
		return decimal.Zero, errors.New("total quantity is zero")
	}

	return totalCost.Div(totalQty), nil
}
