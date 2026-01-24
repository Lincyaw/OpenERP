package inventory

import (
	"sort"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// BatchOutboundStrategyType defines the type of batch outbound strategy
type BatchOutboundStrategyType string

const (
	// BatchOutboundStrategyTypeFIFO uses First In First Out (by production date, then creation date)
	BatchOutboundStrategyTypeFIFO BatchOutboundStrategyType = "FIFO"
	// BatchOutboundStrategyTypeFEFO uses First Expire First Out (prioritizes batches closer to expiry)
	BatchOutboundStrategyTypeFEFO BatchOutboundStrategyType = "FEFO"
	// BatchOutboundStrategyTypeSpecified allows manual selection of specific batches
	BatchOutboundStrategyTypeSpecified BatchOutboundStrategyType = "SPECIFIED"
)

// IsValid checks if the strategy type is valid
func (t BatchOutboundStrategyType) IsValid() bool {
	switch t {
	case BatchOutboundStrategyTypeFIFO, BatchOutboundStrategyTypeFEFO, BatchOutboundStrategyTypeSpecified:
		return true
	}
	return false
}

// String returns the string representation
func (t BatchOutboundStrategyType) String() string {
	return string(t)
}

// AllBatchOutboundStrategyTypes returns all valid batch outbound strategy types
func AllBatchOutboundStrategyTypes() []BatchOutboundStrategyType {
	return []BatchOutboundStrategyType{
		BatchOutboundStrategyTypeFIFO,
		BatchOutboundStrategyTypeFEFO,
		BatchOutboundStrategyTypeSpecified,
	}
}

// BatchDeductionRequest represents a request to deduct from a specific batch
type BatchDeductionRequest struct {
	BatchID  uuid.UUID       // ID of the specific batch to deduct from
	Quantity decimal.Decimal // Quantity to deduct (if zero, deduct as much as possible)
}

// BatchDeductionResult represents the result of deducting from a single batch
type BatchDeductionResult struct {
	BatchID        uuid.UUID       // ID of the batch
	BatchNumber    string          // Batch number for display
	DeductedAmount decimal.Decimal // Amount actually deducted
	UnitCost       decimal.Decimal // Unit cost of this batch (for cost calculation)
	TotalCost      decimal.Decimal // Total cost (DeductedAmount * UnitCost)
	RemainingInBatch decimal.Decimal // Remaining quantity in batch after deduction
	FullyConsumed  bool            // True if batch is now fully consumed
}

// BatchOutboundResult represents the complete result of a batch outbound operation
type BatchOutboundResult struct {
	Deductions          []BatchDeductionResult // List of batch deductions
	TotalDeducted       decimal.Decimal        // Total quantity deducted
	TotalCost           decimal.Decimal        // Total cost of deducted stock
	WeightedAverageCost decimal.Decimal        // Weighted average cost per unit
	RemainingQuantity   decimal.Decimal        // Quantity that could not be fulfilled
	FullyFulfilled      bool                   // True if all requested quantity was fulfilled
	BatchesConsumed     []uuid.UUID            // Batches that were fully consumed
	BatchesPartial      []uuid.UUID            // Batches that were partially consumed
}

// BatchOutboundStrategy is the interface for batch outbound strategies
type BatchOutboundStrategy interface {
	strategy.Strategy
	// StrategyType returns the batch outbound strategy type
	StrategyType() BatchOutboundStrategyType
	// SelectBatches calculates which batches to use and how much to deduct from each
	SelectBatches(requestedQuantity decimal.Decimal, batches []StockBatch) (*BatchOutboundResult, error)
}

// FIFOBatchOutboundStrategy implements FIFO (First In First Out) batch selection
// It selects batches by production date (oldest first), falling back to creation date
type FIFOBatchOutboundStrategy struct {
	strategy.BaseStrategy
}

// NewFIFOBatchOutboundStrategy creates a new FIFO batch outbound strategy
func NewFIFOBatchOutboundStrategy() *FIFOBatchOutboundStrategy {
	return &FIFOBatchOutboundStrategy{
		BaseStrategy: strategy.NewBaseStrategy(
			"fifo_batch_outbound",
			strategy.StrategyTypeBatch,
			"FIFO batch outbound strategy - selects oldest batches first by production date, then creation date",
		),
	}
}

// StrategyType returns the batch outbound strategy type
func (s *FIFOBatchOutboundStrategy) StrategyType() BatchOutboundStrategyType {
	return BatchOutboundStrategyTypeFIFO
}

// SelectBatches selects batches using FIFO order (oldest first by production date)
func (s *FIFOBatchOutboundStrategy) SelectBatches(requestedQuantity decimal.Decimal, batches []StockBatch) (*BatchOutboundResult, error) {
	if requestedQuantity.LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("INVALID_QUANTITY", "Requested quantity must be positive")
	}

	// Filter available batches
	availableBatches := filterAvailableBatches(batches)
	if len(availableBatches) == 0 {
		return &BatchOutboundResult{
			Deductions:        make([]BatchDeductionResult, 0),
			TotalDeducted:     decimal.Zero,
			TotalCost:         decimal.Zero,
			RemainingQuantity: requestedQuantity,
			FullyFulfilled:    false,
			BatchesConsumed:   make([]uuid.UUID, 0),
			BatchesPartial:    make([]uuid.UUID, 0),
		}, nil
	}

	// Sort by FIFO: production date first (oldest first), then creation date
	sortedBatches := make([]StockBatch, len(availableBatches))
	copy(sortedBatches, availableBatches)
	sort.Slice(sortedBatches, func(i, j int) bool {
		// Sort by production date first (nil production dates go to the end)
		if sortedBatches[i].ProductionDate != nil && sortedBatches[j].ProductionDate != nil {
			if !sortedBatches[i].ProductionDate.Equal(*sortedBatches[j].ProductionDate) {
				return sortedBatches[i].ProductionDate.Before(*sortedBatches[j].ProductionDate)
			}
		} else if sortedBatches[i].ProductionDate != nil {
			return true // i has production date, j doesn't - i comes first
		} else if sortedBatches[j].ProductionDate != nil {
			return false // j has production date, i doesn't - j comes first
		}
		// Fall back to creation date
		return sortedBatches[i].CreatedAt.Before(sortedBatches[j].CreatedAt)
	})

	return calculateDeductions(requestedQuantity, sortedBatches)
}

// FEFOBatchOutboundStrategy implements FEFO (First Expire First Out) batch selection
// It prioritizes batches that will expire soonest
type FEFOBatchOutboundStrategy struct {
	strategy.BaseStrategy
}

// NewFEFOBatchOutboundStrategy creates a new FEFO batch outbound strategy
func NewFEFOBatchOutboundStrategy() *FEFOBatchOutboundStrategy {
	return &FEFOBatchOutboundStrategy{
		BaseStrategy: strategy.NewBaseStrategy(
			"fefo_batch_outbound",
			strategy.StrategyTypeBatch,
			"FEFO batch outbound strategy - selects batches closest to expiry first",
		),
	}
}

// StrategyType returns the batch outbound strategy type
func (s *FEFOBatchOutboundStrategy) StrategyType() BatchOutboundStrategyType {
	return BatchOutboundStrategyTypeFEFO
}

// SelectBatches selects batches using FEFO order (earliest expiry first)
func (s *FEFOBatchOutboundStrategy) SelectBatches(requestedQuantity decimal.Decimal, batches []StockBatch) (*BatchOutboundResult, error) {
	if requestedQuantity.LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("INVALID_QUANTITY", "Requested quantity must be positive")
	}

	// Filter available batches
	availableBatches := filterAvailableBatches(batches)
	if len(availableBatches) == 0 {
		return &BatchOutboundResult{
			Deductions:        make([]BatchDeductionResult, 0),
			TotalDeducted:     decimal.Zero,
			TotalCost:         decimal.Zero,
			RemainingQuantity: requestedQuantity,
			FullyFulfilled:    false,
			BatchesConsumed:   make([]uuid.UUID, 0),
			BatchesPartial:    make([]uuid.UUID, 0),
		}, nil
	}

	// Sort by FEFO: expiry date first (earliest first), then production date, then creation date
	sortedBatches := make([]StockBatch, len(availableBatches))
	copy(sortedBatches, availableBatches)
	sort.Slice(sortedBatches, func(i, j int) bool {
		// Sort by expiry date first (batches with expiry come first, nil expiry goes to end)
		if sortedBatches[i].ExpiryDate != nil && sortedBatches[j].ExpiryDate != nil {
			if !sortedBatches[i].ExpiryDate.Equal(*sortedBatches[j].ExpiryDate) {
				return sortedBatches[i].ExpiryDate.Before(*sortedBatches[j].ExpiryDate)
			}
		} else if sortedBatches[i].ExpiryDate != nil {
			return true // i has expiry, j doesn't - i comes first (use expiring stock first)
		} else if sortedBatches[j].ExpiryDate != nil {
			return false // j has expiry, i doesn't - j comes first
		}
		// Fall back to production date (FIFO)
		if sortedBatches[i].ProductionDate != nil && sortedBatches[j].ProductionDate != nil {
			if !sortedBatches[i].ProductionDate.Equal(*sortedBatches[j].ProductionDate) {
				return sortedBatches[i].ProductionDate.Before(*sortedBatches[j].ProductionDate)
			}
		}
		// Final fallback to creation date
		return sortedBatches[i].CreatedAt.Before(sortedBatches[j].CreatedAt)
	})

	return calculateDeductions(requestedQuantity, sortedBatches)
}

// SpecifiedBatchOutboundStrategy allows manual selection of specific batches
type SpecifiedBatchOutboundStrategy struct {
	strategy.BaseStrategy
	requests []BatchDeductionRequest
}

// NewSpecifiedBatchOutboundStrategy creates a new specified batch outbound strategy
func NewSpecifiedBatchOutboundStrategy(requests []BatchDeductionRequest) *SpecifiedBatchOutboundStrategy {
	return &SpecifiedBatchOutboundStrategy{
		BaseStrategy: strategy.NewBaseStrategy(
			"specified_batch_outbound",
			strategy.StrategyTypeBatch,
			"Specified batch outbound strategy - uses user-specified batches in order",
		),
		requests: requests,
	}
}

// StrategyType returns the batch outbound strategy type
func (s *SpecifiedBatchOutboundStrategy) StrategyType() BatchOutboundStrategyType {
	return BatchOutboundStrategyTypeSpecified
}

// GetRequests returns the configured batch deduction requests
func (s *SpecifiedBatchOutboundStrategy) GetRequests() []BatchDeductionRequest {
	return s.requests
}

// SelectBatches selects batches based on user-specified requests
func (s *SpecifiedBatchOutboundStrategy) SelectBatches(requestedQuantity decimal.Decimal, batches []StockBatch) (*BatchOutboundResult, error) {
	if requestedQuantity.LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("INVALID_QUANTITY", "Requested quantity must be positive")
	}

	if len(s.requests) == 0 {
		return nil, shared.NewDomainError("NO_BATCH_REQUESTS", "Specified strategy requires batch deduction requests")
	}

	// Build batch map for quick lookup
	batchMap := make(map[uuid.UUID]*StockBatch)
	for i := range batches {
		if batches[i].IsAvailable() {
			batchMap[batches[i].ID] = &batches[i]
		}
	}

	deductions := make([]BatchDeductionResult, 0)
	batchesConsumed := make([]uuid.UUID, 0)
	batchesPartial := make([]uuid.UUID, 0)
	remaining := requestedQuantity
	totalDeducted := decimal.Zero
	totalCost := decimal.Zero

	// Process specified batch requests in order
	for _, req := range s.requests {
		if remaining.IsZero() {
			break
		}

		batch, exists := batchMap[req.BatchID]
		if !exists {
			continue // Skip non-existent or unavailable batches
		}
		if batch.Quantity.LessThanOrEqual(decimal.Zero) {
			continue
		}

		// Determine deduction amount
		var deductAmount decimal.Decimal
		if req.Quantity.IsZero() || req.Quantity.IsNegative() {
			// If quantity not specified, deduct as much as possible
			deductAmount = decimal.Min(remaining, batch.Quantity)
		} else {
			// Use specified quantity, but cap at available/remaining
			deductAmount = decimal.Min(req.Quantity, remaining)
			deductAmount = decimal.Min(deductAmount, batch.Quantity)
		}

		if deductAmount.LessThanOrEqual(decimal.Zero) {
			continue
		}

		remainingInBatch := batch.Quantity.Sub(deductAmount)
		fullyConsumed := remainingInBatch.IsZero()
		batchCost := deductAmount.Mul(batch.UnitCost)

		deductions = append(deductions, BatchDeductionResult{
			BatchID:          batch.ID,
			BatchNumber:      batch.BatchNumber,
			DeductedAmount:   deductAmount,
			UnitCost:         batch.UnitCost,
			TotalCost:        batchCost,
			RemainingInBatch: remainingInBatch,
			FullyConsumed:    fullyConsumed,
		})

		totalDeducted = totalDeducted.Add(deductAmount)
		totalCost = totalCost.Add(batchCost)
		remaining = remaining.Sub(deductAmount)

		if fullyConsumed {
			batchesConsumed = append(batchesConsumed, batch.ID)
		} else {
			batchesPartial = append(batchesPartial, batch.ID)
		}

		// Update batch quantity for subsequent iterations (in case same batch appears twice)
		batch.Quantity = remainingInBatch
	}

	// Calculate weighted average cost
	var weightedAvgCost decimal.Decimal
	if totalDeducted.GreaterThan(decimal.Zero) {
		weightedAvgCost = totalCost.Div(totalDeducted).Round(4)
	}

	return &BatchOutboundResult{
		Deductions:          deductions,
		TotalDeducted:       totalDeducted,
		TotalCost:           totalCost,
		WeightedAverageCost: weightedAvgCost,
		RemainingQuantity:   remaining,
		FullyFulfilled:      remaining.IsZero(),
		BatchesConsumed:     batchesConsumed,
		BatchesPartial:      batchesPartial,
	}, nil
}

// filterAvailableBatches returns batches that are available (not consumed and not expired)
func filterAvailableBatches(batches []StockBatch) []StockBatch {
	available := make([]StockBatch, 0, len(batches))
	for _, batch := range batches {
		if batch.IsAvailable() {
			available = append(available, batch)
		}
	}
	return available
}

// calculateDeductions is a helper function to calculate batch deductions
func calculateDeductions(requestedQuantity decimal.Decimal, sortedBatches []StockBatch) (*BatchOutboundResult, error) {
	deductions := make([]BatchDeductionResult, 0)
	batchesConsumed := make([]uuid.UUID, 0)
	batchesPartial := make([]uuid.UUID, 0)
	remaining := requestedQuantity
	totalDeducted := decimal.Zero
	totalCost := decimal.Zero

	for _, batch := range sortedBatches {
		if remaining.IsZero() {
			break
		}
		if batch.Quantity.LessThanOrEqual(decimal.Zero) {
			continue
		}

		// Calculate how much to take from this batch
		deductAmount := decimal.Min(remaining, batch.Quantity)
		remainingInBatch := batch.Quantity.Sub(deductAmount)
		fullyConsumed := remainingInBatch.IsZero()
		batchCost := deductAmount.Mul(batch.UnitCost)

		deductions = append(deductions, BatchDeductionResult{
			BatchID:          batch.ID,
			BatchNumber:      batch.BatchNumber,
			DeductedAmount:   deductAmount,
			UnitCost:         batch.UnitCost,
			TotalCost:        batchCost,
			RemainingInBatch: remainingInBatch,
			FullyConsumed:    fullyConsumed,
		})

		totalDeducted = totalDeducted.Add(deductAmount)
		totalCost = totalCost.Add(batchCost)
		remaining = remaining.Sub(deductAmount)

		if fullyConsumed {
			batchesConsumed = append(batchesConsumed, batch.ID)
		} else {
			batchesPartial = append(batchesPartial, batch.ID)
		}
	}

	// Calculate weighted average cost
	var weightedAvgCost decimal.Decimal
	if totalDeducted.GreaterThan(decimal.Zero) {
		weightedAvgCost = totalCost.Div(totalDeducted).Round(4)
	}

	return &BatchOutboundResult{
		Deductions:          deductions,
		TotalDeducted:       totalDeducted,
		TotalCost:           totalCost,
		WeightedAverageCost: weightedAvgCost,
		RemainingQuantity:   remaining,
		FullyFulfilled:      remaining.IsZero(),
		BatchesConsumed:     batchesConsumed,
		BatchesPartial:      batchesPartial,
	}, nil
}

// BatchOutboundStrategyFactory creates batch outbound strategies
type BatchOutboundStrategyFactory struct{}

// NewBatchOutboundStrategyFactory creates a new factory
func NewBatchOutboundStrategyFactory() *BatchOutboundStrategyFactory {
	return &BatchOutboundStrategyFactory{}
}

// CreateFIFOStrategy creates a FIFO batch outbound strategy
func (f *BatchOutboundStrategyFactory) CreateFIFOStrategy() *FIFOBatchOutboundStrategy {
	return NewFIFOBatchOutboundStrategy()
}

// CreateFEFOStrategy creates a FEFO batch outbound strategy
func (f *BatchOutboundStrategyFactory) CreateFEFOStrategy() *FEFOBatchOutboundStrategy {
	return NewFEFOBatchOutboundStrategy()
}

// CreateSpecifiedStrategy creates a specified batch outbound strategy with given requests
func (f *BatchOutboundStrategyFactory) CreateSpecifiedStrategy(requests []BatchDeductionRequest) *SpecifiedBatchOutboundStrategy {
	return NewSpecifiedBatchOutboundStrategy(requests)
}

// GetStrategy returns a strategy by type
func (f *BatchOutboundStrategyFactory) GetStrategy(strategyType BatchOutboundStrategyType, requests []BatchDeductionRequest) (BatchOutboundStrategy, error) {
	switch strategyType {
	case BatchOutboundStrategyTypeFIFO:
		return f.CreateFIFOStrategy(), nil
	case BatchOutboundStrategyTypeFEFO:
		return f.CreateFEFOStrategy(), nil
	case BatchOutboundStrategyTypeSpecified:
		if len(requests) == 0 {
			return nil, shared.NewDomainError("INVALID_REQUESTS", "Specified strategy requires batch deduction requests")
		}
		return f.CreateSpecifiedStrategy(requests), nil
	default:
		return nil, shared.NewDomainError("INVALID_STRATEGY", "Unknown batch outbound strategy type")
	}
}

// GetDefaultStrategy returns the default strategy (FIFO)
func (f *BatchOutboundStrategyFactory) GetDefaultStrategy() BatchOutboundStrategy {
	return f.CreateFIFOStrategy()
}

// ApplyBatchDeductions applies the deduction results to actual batches
// This is a helper function to execute the strategy result on real batch entities
func ApplyBatchDeductions(batches []*StockBatch, result *BatchOutboundResult) error {
	if result == nil {
		return shared.NewDomainError("INVALID_RESULT", "Batch outbound result cannot be nil")
	}

	// Build batch map
	batchMap := make(map[uuid.UUID]*StockBatch)
	for _, batch := range batches {
		batchMap[batch.ID] = batch
	}

	// Apply each deduction
	for _, deduction := range result.Deductions {
		batch, exists := batchMap[deduction.BatchID]
		if !exists {
			return shared.NewDomainError("BATCH_NOT_FOUND", "Batch not found: "+deduction.BatchID.String())
		}

		actualDeducted := batch.Deduct(deduction.DeductedAmount)
		if !actualDeducted.Equal(deduction.DeductedAmount) {
			return shared.NewDomainError("DEDUCTION_MISMATCH", "Batch deduction amount mismatch")
		}
	}

	return nil
}

// ValidateBatchAvailability checks if the batches have sufficient quantity for the request
func ValidateBatchAvailability(batches []StockBatch, requestedQuantity decimal.Decimal) (bool, decimal.Decimal) {
	totalAvailable := decimal.Zero
	for _, batch := range batches {
		if batch.IsAvailable() {
			totalAvailable = totalAvailable.Add(batch.Quantity)
		}
	}
	return totalAvailable.GreaterThanOrEqual(requestedQuantity), totalAvailable
}

// GetBatchesByExpiryWindow returns batches that will expire within the given duration
func GetBatchesByExpiryWindow(batches []StockBatch, window time.Duration) []StockBatch {
	expiring := make([]StockBatch, 0)
	deadline := time.Now().Add(window)
	for _, batch := range batches {
		if batch.IsAvailable() && batch.ExpiryDate != nil && batch.ExpiryDate.Before(deadline) {
			expiring = append(expiring, batch)
		}
	}
	return expiring
}
