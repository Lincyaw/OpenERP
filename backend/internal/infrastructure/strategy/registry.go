package strategy

import (
	"fmt"
	"sort"
	"sync"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/strategy"
)

// StrategyRegistry manages strategy registrations
type StrategyRegistry struct {
	mu                   sync.RWMutex
	costStrategies       map[string]strategy.CostCalculationStrategy
	pricingStrategies    map[string]strategy.PricingStrategy
	allocationStrategies map[string]strategy.PaymentAllocationStrategy
	batchStrategies      map[string]strategy.BatchManagementStrategy
	validationStrategies map[string]strategy.ProductValidationStrategy
	defaults             map[strategy.StrategyType]string
}

// NewStrategyRegistry creates a new strategy registry
func NewStrategyRegistry() *StrategyRegistry {
	return &StrategyRegistry{
		costStrategies:       make(map[string]strategy.CostCalculationStrategy),
		pricingStrategies:    make(map[string]strategy.PricingStrategy),
		allocationStrategies: make(map[string]strategy.PaymentAllocationStrategy),
		batchStrategies:      make(map[string]strategy.BatchManagementStrategy),
		validationStrategies: make(map[string]strategy.ProductValidationStrategy),
		defaults:             make(map[strategy.StrategyType]string),
	}
}

// RegisterCostStrategy registers a cost calculation strategy
func (r *StrategyRegistry) RegisterCostStrategy(s strategy.CostCalculationStrategy) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := s.Name()
	if _, exists := r.costStrategies[name]; exists {
		return fmt.Errorf("%w: cost strategy '%s' already registered", shared.ErrAlreadyExists, name)
	}
	r.costStrategies[name] = s
	return nil
}

// GetCostStrategy returns a cost strategy by name, or the default if name is empty
func (r *StrategyRegistry) GetCostStrategy(name string) (strategy.CostCalculationStrategy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if name == "" {
		name = r.defaults[strategy.StrategyTypeCost]
		if name == "" {
			return nil, fmt.Errorf("%w: no default cost strategy set", shared.ErrNotFound)
		}
	}

	s, exists := r.costStrategies[name]
	if !exists {
		return nil, fmt.Errorf("%w: cost strategy '%s' not found", shared.ErrNotFound, name)
	}
	return s, nil
}

// GetCostStrategyOrDefault returns a cost strategy by name, or the default if not found
func (r *StrategyRegistry) GetCostStrategyOrDefault(name string) strategy.CostCalculationStrategy {
	s, err := r.GetCostStrategy(name)
	if err != nil {
		s, _ = r.GetCostStrategy("")
	}
	return s
}

// ListCostStrategies returns all registered cost strategy names
func (r *StrategyRegistry) ListCostStrategies() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.costStrategies))
	for name := range r.costStrategies {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// UnregisterCostStrategy removes a cost strategy
func (r *StrategyRegistry) UnregisterCostStrategy(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.costStrategies[name]; !exists {
		return fmt.Errorf("%w: cost strategy '%s' not found", shared.ErrNotFound, name)
	}
	delete(r.costStrategies, name)

	// Clear default if it was this strategy
	if r.defaults[strategy.StrategyTypeCost] == name {
		delete(r.defaults, strategy.StrategyTypeCost)
	}
	return nil
}

// RegisterPricingStrategy registers a pricing strategy
func (r *StrategyRegistry) RegisterPricingStrategy(s strategy.PricingStrategy) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := s.Name()
	if _, exists := r.pricingStrategies[name]; exists {
		return fmt.Errorf("%w: pricing strategy '%s' already registered", shared.ErrAlreadyExists, name)
	}
	r.pricingStrategies[name] = s
	return nil
}

// GetPricingStrategy returns a pricing strategy by name, or the default if name is empty
func (r *StrategyRegistry) GetPricingStrategy(name string) (strategy.PricingStrategy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if name == "" {
		name = r.defaults[strategy.StrategyTypePricing]
		if name == "" {
			return nil, fmt.Errorf("%w: no default pricing strategy set", shared.ErrNotFound)
		}
	}

	s, exists := r.pricingStrategies[name]
	if !exists {
		return nil, fmt.Errorf("%w: pricing strategy '%s' not found", shared.ErrNotFound, name)
	}
	return s, nil
}

// GetPricingStrategyOrDefault returns a pricing strategy by name, or the default if not found
func (r *StrategyRegistry) GetPricingStrategyOrDefault(name string) strategy.PricingStrategy {
	s, err := r.GetPricingStrategy(name)
	if err != nil {
		s, _ = r.GetPricingStrategy("")
	}
	return s
}

// ListPricingStrategies returns all registered pricing strategy names
func (r *StrategyRegistry) ListPricingStrategies() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.pricingStrategies))
	for name := range r.pricingStrategies {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// UnregisterPricingStrategy removes a pricing strategy
func (r *StrategyRegistry) UnregisterPricingStrategy(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.pricingStrategies[name]; !exists {
		return fmt.Errorf("%w: pricing strategy '%s' not found", shared.ErrNotFound, name)
	}
	delete(r.pricingStrategies, name)

	if r.defaults[strategy.StrategyTypePricing] == name {
		delete(r.defaults, strategy.StrategyTypePricing)
	}
	return nil
}

// RegisterAllocationStrategy registers a payment allocation strategy
func (r *StrategyRegistry) RegisterAllocationStrategy(s strategy.PaymentAllocationStrategy) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := s.Name()
	if _, exists := r.allocationStrategies[name]; exists {
		return fmt.Errorf("%w: allocation strategy '%s' already registered", shared.ErrAlreadyExists, name)
	}
	r.allocationStrategies[name] = s
	return nil
}

// GetAllocationStrategy returns an allocation strategy by name, or the default if name is empty
func (r *StrategyRegistry) GetAllocationStrategy(name string) (strategy.PaymentAllocationStrategy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if name == "" {
		name = r.defaults[strategy.StrategyTypeAllocation]
		if name == "" {
			return nil, fmt.Errorf("%w: no default allocation strategy set", shared.ErrNotFound)
		}
	}

	s, exists := r.allocationStrategies[name]
	if !exists {
		return nil, fmt.Errorf("%w: allocation strategy '%s' not found", shared.ErrNotFound, name)
	}
	return s, nil
}

// GetAllocationStrategyOrDefault returns an allocation strategy by name, or the default if not found
func (r *StrategyRegistry) GetAllocationStrategyOrDefault(name string) strategy.PaymentAllocationStrategy {
	s, err := r.GetAllocationStrategy(name)
	if err != nil {
		s, _ = r.GetAllocationStrategy("")
	}
	return s
}

// ListAllocationStrategies returns all registered allocation strategy names
func (r *StrategyRegistry) ListAllocationStrategies() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.allocationStrategies))
	for name := range r.allocationStrategies {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// UnregisterAllocationStrategy removes an allocation strategy
func (r *StrategyRegistry) UnregisterAllocationStrategy(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.allocationStrategies[name]; !exists {
		return fmt.Errorf("%w: allocation strategy '%s' not found", shared.ErrNotFound, name)
	}
	delete(r.allocationStrategies, name)

	if r.defaults[strategy.StrategyTypeAllocation] == name {
		delete(r.defaults, strategy.StrategyTypeAllocation)
	}
	return nil
}

// RegisterBatchStrategy registers a batch management strategy
func (r *StrategyRegistry) RegisterBatchStrategy(s strategy.BatchManagementStrategy) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := s.Name()
	if _, exists := r.batchStrategies[name]; exists {
		return fmt.Errorf("%w: batch strategy '%s' already registered", shared.ErrAlreadyExists, name)
	}
	r.batchStrategies[name] = s
	return nil
}

// GetBatchStrategy returns a batch strategy by name, or the default if name is empty
func (r *StrategyRegistry) GetBatchStrategy(name string) (strategy.BatchManagementStrategy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if name == "" {
		name = r.defaults[strategy.StrategyTypeBatch]
		if name == "" {
			return nil, fmt.Errorf("%w: no default batch strategy set", shared.ErrNotFound)
		}
	}

	s, exists := r.batchStrategies[name]
	if !exists {
		return nil, fmt.Errorf("%w: batch strategy '%s' not found", shared.ErrNotFound, name)
	}
	return s, nil
}

// GetBatchStrategyOrDefault returns a batch strategy by name, or the default if not found
func (r *StrategyRegistry) GetBatchStrategyOrDefault(name string) strategy.BatchManagementStrategy {
	s, err := r.GetBatchStrategy(name)
	if err != nil {
		s, _ = r.GetBatchStrategy("")
	}
	return s
}

// ListBatchStrategies returns all registered batch strategy names
func (r *StrategyRegistry) ListBatchStrategies() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.batchStrategies))
	for name := range r.batchStrategies {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// UnregisterBatchStrategy removes a batch strategy
func (r *StrategyRegistry) UnregisterBatchStrategy(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.batchStrategies[name]; !exists {
		return fmt.Errorf("%w: batch strategy '%s' not found", shared.ErrNotFound, name)
	}
	delete(r.batchStrategies, name)

	if r.defaults[strategy.StrategyTypeBatch] == name {
		delete(r.defaults, strategy.StrategyTypeBatch)
	}
	return nil
}

// RegisterValidationStrategy registers a product validation strategy
func (r *StrategyRegistry) RegisterValidationStrategy(s strategy.ProductValidationStrategy) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := s.Name()
	if _, exists := r.validationStrategies[name]; exists {
		return fmt.Errorf("%w: validation strategy '%s' already registered", shared.ErrAlreadyExists, name)
	}
	r.validationStrategies[name] = s
	return nil
}

// GetValidationStrategy returns a validation strategy by name, or the default if name is empty
func (r *StrategyRegistry) GetValidationStrategy(name string) (strategy.ProductValidationStrategy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if name == "" {
		name = r.defaults[strategy.StrategyTypeValidation]
		if name == "" {
			return nil, fmt.Errorf("%w: no default validation strategy set", shared.ErrNotFound)
		}
	}

	s, exists := r.validationStrategies[name]
	if !exists {
		return nil, fmt.Errorf("%w: validation strategy '%s' not found", shared.ErrNotFound, name)
	}
	return s, nil
}

// GetValidationStrategyOrDefault returns a validation strategy by name, or the default if not found
func (r *StrategyRegistry) GetValidationStrategyOrDefault(name string) strategy.ProductValidationStrategy {
	s, err := r.GetValidationStrategy(name)
	if err != nil {
		s, _ = r.GetValidationStrategy("")
	}
	return s
}

// ListValidationStrategies returns all registered validation strategy names
func (r *StrategyRegistry) ListValidationStrategies() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.validationStrategies))
	for name := range r.validationStrategies {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// UnregisterValidationStrategy removes a validation strategy
func (r *StrategyRegistry) UnregisterValidationStrategy(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.validationStrategies[name]; !exists {
		return fmt.Errorf("%w: validation strategy '%s' not found", shared.ErrNotFound, name)
	}
	delete(r.validationStrategies, name)

	if r.defaults[strategy.StrategyTypeValidation] == name {
		delete(r.defaults, strategy.StrategyTypeValidation)
	}
	return nil
}

// SetDefault sets the default strategy for a strategy type
func (r *StrategyRegistry) SetDefault(strategyType strategy.StrategyType, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isRegisteredLocked(strategyType, name) {
		return fmt.Errorf("%w: strategy '%s' of type '%s' not found", shared.ErrNotFound, name, strategyType)
	}

	r.defaults[strategyType] = name
	return nil
}

// GetDefault returns the default strategy name for a strategy type
func (r *StrategyRegistry) GetDefault(strategyType strategy.StrategyType) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.defaults[strategyType]
}

// HasDefault returns true if a default is set for the strategy type
func (r *StrategyRegistry) HasDefault(strategyType strategy.StrategyType) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.defaults[strategyType] != ""
}

// IsRegistered returns true if a strategy with the given name is registered for the type
func (r *StrategyRegistry) IsRegistered(strategyType strategy.StrategyType, name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.isRegisteredLocked(strategyType, name)
}

// isRegisteredLocked checks registration without locking (caller must hold lock)
func (r *StrategyRegistry) isRegisteredLocked(strategyType strategy.StrategyType, name string) bool {
	switch strategyType {
	case strategy.StrategyTypeCost:
		_, exists := r.costStrategies[name]
		return exists
	case strategy.StrategyTypePricing:
		_, exists := r.pricingStrategies[name]
		return exists
	case strategy.StrategyTypeAllocation:
		_, exists := r.allocationStrategies[name]
		return exists
	case strategy.StrategyTypeBatch:
		_, exists := r.batchStrategies[name]
		return exists
	case strategy.StrategyTypeValidation:
		_, exists := r.validationStrategies[name]
		return exists
	default:
		return false
	}
}

// Stats returns registration counts for each strategy type
func (r *StrategyRegistry) Stats() map[strategy.StrategyType]int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return map[strategy.StrategyType]int{
		strategy.StrategyTypeCost:       len(r.costStrategies),
		strategy.StrategyTypePricing:    len(r.pricingStrategies),
		strategy.StrategyTypeAllocation: len(r.allocationStrategies),
		strategy.StrategyTypeBatch:      len(r.batchStrategies),
		strategy.StrategyTypeValidation: len(r.validationStrategies),
	}
}
