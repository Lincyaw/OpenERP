package strategy

import (
	"github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/erp/backend/internal/infrastructure/strategy/allocation"
	"github.com/erp/backend/internal/infrastructure/strategy/batch"
	"github.com/erp/backend/internal/infrastructure/strategy/cost"
	"github.com/erp/backend/internal/infrastructure/strategy/pricing"
	"github.com/erp/backend/internal/infrastructure/strategy/validation"
)

// NewRegistryWithDefaults creates a new registry with default strategies registered.
// The customer level pricing strategy uses static fallback discounts.
// For production use with dynamic discount lookup, use NewRegistryWithProvider.
func NewRegistryWithDefaults() (*StrategyRegistry, error) {
	return NewRegistryWithProvider(nil)
}

// NewRegistryWithProvider creates a new registry with default strategies registered,
// using the provided CustomerLevelProvider for dynamic discount lookups.
// If provider is nil, the customer level strategy uses static fallback discounts.
func NewRegistryWithProvider(customerLevelProvider pricing.CustomerLevelProvider) (*StrategyRegistry, error) {
	r := NewStrategyRegistry()

	// Register cost strategies
	movingAvg := cost.NewMovingAverageCostStrategy()
	if err := r.RegisterCostStrategy(movingAvg); err != nil {
		return nil, err
	}

	fifoCost := cost.NewFIFOCostStrategy()
	if err := r.RegisterCostStrategy(fifoCost); err != nil {
		return nil, err
	}

	// Register pricing strategies
	standardPricing := pricing.NewStandardPricingStrategy()
	if err := r.RegisterPricingStrategy(standardPricing); err != nil {
		return nil, err
	}

	tieredPricing := pricing.DefaultTieredPricingStrategy()
	if err := r.RegisterPricingStrategy(tieredPricing); err != nil {
		return nil, err
	}

	// Create customer level pricing strategy with provider (if provided)
	customerLevelPricing := pricing.NewCustomerLevelPricingStrategy(customerLevelProvider)
	if err := r.RegisterPricingStrategy(customerLevelPricing); err != nil {
		return nil, err
	}

	// Register allocation strategies
	fifoAlloc := allocation.NewFIFOAllocationStrategy()
	if err := r.RegisterAllocationStrategy(fifoAlloc); err != nil {
		return nil, err
	}

	// Register batch strategies
	standardBatch := batch.NewStandardBatchStrategy()
	if err := r.RegisterBatchStrategy(standardBatch); err != nil {
		return nil, err
	}

	fifoBatch := batch.NewFIFOBatchStrategy()
	if err := r.RegisterBatchStrategy(fifoBatch); err != nil {
		return nil, err
	}

	fefoBatch := batch.NewFEFOBatchStrategy()
	if err := r.RegisterBatchStrategy(fefoBatch); err != nil {
		return nil, err
	}

	// Register validation strategies
	standardValidator := validation.NewStandardProductValidator()
	if err := r.RegisterValidationStrategy(standardValidator); err != nil {
		return nil, err
	}

	// Set defaults
	if err := r.SetDefault(strategy.StrategyTypeCost, movingAvg.Name()); err != nil {
		return nil, err
	}
	if err := r.SetDefault(strategy.StrategyTypePricing, standardPricing.Name()); err != nil {
		return nil, err
	}
	if err := r.SetDefault(strategy.StrategyTypeAllocation, fifoAlloc.Name()); err != nil {
		return nil, err
	}
	if err := r.SetDefault(strategy.StrategyTypeBatch, standardBatch.Name()); err != nil {
		return nil, err
	}
	if err := r.SetDefault(strategy.StrategyTypeValidation, standardValidator.Name()); err != nil {
		return nil, err
	}

	return r, nil
}
