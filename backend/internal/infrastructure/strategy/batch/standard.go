package batch

import (
	"context"
	"sort"

	"github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/shopspring/decimal"
)

// StandardBatchStrategy implements FIFO batch selection
type StandardBatchStrategy struct {
	strategy.BaseStrategy
}

// NewStandardBatchStrategy creates a new standard batch strategy
func NewStandardBatchStrategy() *StandardBatchStrategy {
	return &StandardBatchStrategy{
		BaseStrategy: strategy.NewBaseStrategy(
			"standard",
			strategy.StrategyTypeBatch,
			"FIFO batch selection by received date",
		),
	}
}

// SelectBatches selects batches in FIFO order by received date
func (s *StandardBatchStrategy) SelectBatches(
	ctx context.Context,
	selCtx strategy.BatchSelectionContext,
	batches []strategy.Batch,
) (strategy.BatchSelectionResult, error) {
	// Filter batches for the product and warehouse
	filtered := make([]strategy.Batch, 0)
	for _, b := range batches {
		if b.ProductID == selCtx.ProductID && b.WarehouseID == selCtx.WarehouseID && b.AvailableQty.IsPositive() {
			filtered = append(filtered, b)
		}
	}

	// Sort by received date (oldest first)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].ReceivedDate.Before(filtered[j].ReceivedDate)
	})

	remainingQty := selCtx.Quantity
	selections := make([]strategy.BatchSelection, 0)
	totalQty := decimal.Zero

	for _, batch := range filtered {
		if remainingQty.IsZero() || remainingQty.IsNegative() {
			break
		}

		selectedQty := decimal.Min(remainingQty, batch.AvailableQty)
		selections = append(selections, strategy.BatchSelection{
			BatchID:     batch.ID,
			BatchNumber: batch.BatchNumber,
			Quantity:    selectedQty,
			ExpiryDate:  batch.ExpiryDate,
			UnitCost:    batch.UnitCost,
		})

		remainingQty = remainingQty.Sub(selectedQty)
		totalQty = totalQty.Add(selectedQty)
	}

	return strategy.BatchSelectionResult{
		Selections:   selections,
		TotalQty:     totalQty,
		ShortfallQty: remainingQty,
	}, nil
}

// ConsidersExpiry returns false as standard strategy doesn't consider expiry
func (s *StandardBatchStrategy) ConsidersExpiry() bool {
	return false
}

// SupportsFEFO returns false as this is FIFO not FEFO
func (s *StandardBatchStrategy) SupportsFEFO() bool {
	return false
}
