package batch

import (
	"context"
	"sort"

	"github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/shopspring/decimal"
)

// FIFOBatchStrategy implements First In First Out batch selection
// Batches are selected based on manufacture date (oldest first)
type FIFOBatchStrategy struct {
	strategy.BaseStrategy
}

// NewFIFOBatchStrategy creates a new FIFO batch strategy
func NewFIFOBatchStrategy() *FIFOBatchStrategy {
	return &FIFOBatchStrategy{
		BaseStrategy: strategy.NewBaseStrategy(
			"fifo",
			strategy.StrategyTypeBatch,
			"First In First Out - selects batches by manufacture date (oldest first)",
		),
	}
}

// SelectBatches selects batches in FIFO order by manufacture date
func (s *FIFOBatchStrategy) SelectBatches(
	ctx context.Context,
	selCtx strategy.BatchSelectionContext,
	batches []strategy.Batch,
) (strategy.BatchSelectionResult, error) {
	// Filter batches for the product and warehouse with available quantity
	filtered := filterAvailableBatches(batches, selCtx.ProductID, selCtx.WarehouseID)

	// Handle preferred batch if specified
	if selCtx.PreferBatch != "" {
		filtered = prioritizePreferredBatch(filtered, selCtx.PreferBatch)
	} else {
		// Sort by manufacture date (oldest first) for FIFO
		sort.Slice(filtered, func(i, j int) bool {
			// If manufacture date is zero, use received date as fallback
			iDate := filtered[i].ManufactureDate
			jDate := filtered[j].ManufactureDate
			if iDate.IsZero() {
				iDate = filtered[i].ReceivedDate
			}
			if jDate.IsZero() {
				jDate = filtered[j].ReceivedDate
			}
			return iDate.Before(jDate)
		})
	}

	return selectFromBatches(filtered, selCtx.Quantity)
}

// ConsidersExpiry returns false as FIFO doesn't consider expiry dates
func (s *FIFOBatchStrategy) ConsidersExpiry() bool {
	return false
}

// SupportsFEFO returns false as this is FIFO not FEFO
func (s *FIFOBatchStrategy) SupportsFEFO() bool {
	return false
}

// filterAvailableBatches filters batches by product, warehouse, and positive available quantity
func filterAvailableBatches(batches []strategy.Batch, productID, warehouseID string) []strategy.Batch {
	filtered := make([]strategy.Batch, 0)
	for _, b := range batches {
		if b.ProductID == productID && b.WarehouseID == warehouseID && b.AvailableQty.IsPositive() {
			filtered = append(filtered, b)
		}
	}
	return filtered
}

// prioritizePreferredBatch moves the preferred batch to the front of the list
func prioritizePreferredBatch(batches []strategy.Batch, preferredBatch string) []strategy.Batch {
	result := make([]strategy.Batch, 0, len(batches))
	var preferred *strategy.Batch

	for i := range batches {
		if batches[i].BatchNumber == preferredBatch {
			preferred = &batches[i]
		} else {
			result = append(result, batches[i])
		}
	}

	if preferred != nil {
		result = append([]strategy.Batch{*preferred}, result...)
	}

	return result
}

// selectFromBatches selects quantity from sorted batches
func selectFromBatches(batches []strategy.Batch, quantity decimal.Decimal) (strategy.BatchSelectionResult, error) {
	remainingQty := quantity
	selections := make([]strategy.BatchSelection, 0)
	totalQty := decimal.Zero

	for _, batch := range batches {
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
