package batch

import (
	"context"
	"sort"
	"time"

	"github.com/erp/backend/internal/domain/shared/strategy"
)

// FEFOBatchStrategy implements First Expired First Out batch selection
// Batches are selected based on expiry date (earliest expiry first)
// Ideal for perishable goods, pharmaceuticals, and food products
type FEFOBatchStrategy struct {
	strategy.BaseStrategy
}

// NewFEFOBatchStrategy creates a new FEFO batch strategy
func NewFEFOBatchStrategy() *FEFOBatchStrategy {
	return &FEFOBatchStrategy{
		BaseStrategy: strategy.NewBaseStrategy(
			"fefo",
			strategy.StrategyTypeBatch,
			"First Expired First Out - selects batches by expiry date (earliest expiry first)",
		),
	}
}

// SelectBatches selects batches in FEFO order by expiry date
func (s *FEFOBatchStrategy) SelectBatches(
	ctx context.Context,
	selCtx strategy.BatchSelectionContext,
	batches []strategy.Batch,
) (strategy.BatchSelectionResult, error) {
	// Filter batches for the product and warehouse with available quantity
	filtered := filterAvailableBatches(batches, selCtx.ProductID, selCtx.WarehouseID)

	// Further filter out already expired batches
	filtered = filterNonExpiredBatches(filtered, selCtx.Date)

	// Handle preferred batch if specified
	if selCtx.PreferBatch != "" {
		filtered = prioritizePreferredBatch(filtered, selCtx.PreferBatch)
	} else {
		// Sort by expiry date (earliest expiry first) for FEFO
		sort.Slice(filtered, func(i, j int) bool {
			// Batches without expiry date go last
			iExpiry := filtered[i].ExpiryDate
			jExpiry := filtered[j].ExpiryDate

			// If both have no expiry, fall back to manufacture date
			if iExpiry.IsZero() && jExpiry.IsZero() {
				return filtered[i].ManufactureDate.Before(filtered[j].ManufactureDate)
			}
			// If only one has no expiry, it goes last
			if iExpiry.IsZero() {
				return false
			}
			if jExpiry.IsZero() {
				return true
			}
			return iExpiry.Before(jExpiry)
		})
	}

	return selectFromBatches(filtered, selCtx.Quantity)
}

// ConsidersExpiry returns true as FEFO considers expiry dates
func (s *FEFOBatchStrategy) ConsidersExpiry() bool {
	return true
}

// SupportsFEFO returns true as this is FEFO strategy
func (s *FEFOBatchStrategy) SupportsFEFO() bool {
	return true
}

// filterNonExpiredBatches filters out batches that have already expired
func filterNonExpiredBatches(batches []strategy.Batch, currentDate time.Time) []strategy.Batch {
	if currentDate.IsZero() {
		currentDate = time.Now()
	}

	filtered := make([]strategy.Batch, 0, len(batches))
	for _, b := range batches {
		// Include batches with no expiry date or expiry date in the future
		if b.ExpiryDate.IsZero() || b.ExpiryDate.After(currentDate) {
			filtered = append(filtered, b)
		}
	}
	return filtered
}
