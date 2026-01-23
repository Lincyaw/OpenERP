package cost

import (
	"context"
	"errors"

	"github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/shopspring/decimal"
)

// MovingAverageCostStrategy implements weighted average cost calculation
type MovingAverageCostStrategy struct {
	strategy.BaseStrategy
}

// NewMovingAverageCostStrategy creates a new moving average cost strategy
func NewMovingAverageCostStrategy() *MovingAverageCostStrategy {
	return &MovingAverageCostStrategy{
		BaseStrategy: strategy.NewBaseStrategy(
			"moving_average",
			strategy.StrategyTypeCost,
			"Weighted moving average cost calculation",
		),
	}
}

// Method returns the costing method
func (s *MovingAverageCostStrategy) Method() strategy.CostMethod {
	return strategy.CostMethodMovingAverage
}

// CalculateCost calculates the cost using weighted average
func (s *MovingAverageCostStrategy) CalculateCost(
	ctx context.Context,
	costCtx strategy.CostContext,
	entries []strategy.StockEntry,
) (strategy.CostResult, error) {
	if len(entries) == 0 {
		return strategy.CostResult{}, errors.New("no stock entries provided")
	}

	avgCost, err := s.CalculateAverageCost(ctx, entries)
	if err != nil {
		return strategy.CostResult{}, err
	}

	totalCost := avgCost.Mul(costCtx.Quantity)

	return strategy.CostResult{
		UnitCost:     avgCost,
		TotalCost:    totalCost,
		Method:       strategy.CostMethodMovingAverage,
		EntriesUsed:  entries,
		RemainingQty: decimal.Zero,
	}, nil
}

// CalculateAverageCost calculates the weighted average cost
func (s *MovingAverageCostStrategy) CalculateAverageCost(
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
