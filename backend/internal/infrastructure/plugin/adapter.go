package plugin

import (
	"github.com/erp/backend/internal/domain/shared/plugin"
	infraStrategy "github.com/erp/backend/internal/infrastructure/strategy"
)

// StrategyRegistryAdapter wraps StrategyRegistry to implement StrategyRegistrar interface
type StrategyRegistryAdapter struct {
	registry *infraStrategy.StrategyRegistry
}

// NewStrategyRegistryAdapter creates a new adapter for StrategyRegistry
func NewStrategyRegistryAdapter(registry *infraStrategy.StrategyRegistry) *StrategyRegistryAdapter {
	return &StrategyRegistryAdapter{registry: registry}
}

// RegisterCostStrategy implements StrategyRegistrar.RegisterCostStrategy
func (a *StrategyRegistryAdapter) RegisterCostStrategy(s any) error {
	return a.registry.RegisterCostStrategyAny(s)
}

// RegisterPricingStrategy implements StrategyRegistrar.RegisterPricingStrategy
func (a *StrategyRegistryAdapter) RegisterPricingStrategy(s any) error {
	return a.registry.RegisterPricingStrategyAny(s)
}

// RegisterAllocationStrategy implements StrategyRegistrar.RegisterAllocationStrategy
func (a *StrategyRegistryAdapter) RegisterAllocationStrategy(s any) error {
	return a.registry.RegisterAllocationStrategyAny(s)
}

// RegisterBatchStrategy implements StrategyRegistrar.RegisterBatchStrategy
func (a *StrategyRegistryAdapter) RegisterBatchStrategy(s any) error {
	return a.registry.RegisterBatchStrategyAny(s)
}

// RegisterValidationStrategy implements StrategyRegistrar.RegisterValidationStrategy
func (a *StrategyRegistryAdapter) RegisterValidationStrategy(s any) error {
	return a.registry.RegisterValidationStrategyAny(s)
}

// Ensure StrategyRegistryAdapter implements StrategyRegistrar interface
var _ plugin.StrategyRegistrar = (*StrategyRegistryAdapter)(nil)
