package strategy

// StrategyType represents the type of strategy
type StrategyType string

const (
	StrategyTypeCost       StrategyType = "cost"
	StrategyTypePricing    StrategyType = "pricing"
	StrategyTypeAllocation StrategyType = "allocation"
	StrategyTypeBatch      StrategyType = "batch"
	StrategyTypeValidation StrategyType = "validation"
)

// String returns the string representation of the strategy type
func (t StrategyType) String() string {
	return string(t)
}

// IsValid returns true if the strategy type is valid
func (t StrategyType) IsValid() bool {
	switch t {
	case StrategyTypeCost, StrategyTypePricing, StrategyTypeAllocation, StrategyTypeBatch, StrategyTypeValidation:
		return true
	default:
		return false
	}
}

// AllStrategyTypes returns all valid strategy types
func AllStrategyTypes() []StrategyType {
	return []StrategyType{
		StrategyTypeCost,
		StrategyTypePricing,
		StrategyTypeAllocation,
		StrategyTypeBatch,
		StrategyTypeValidation,
	}
}

// Strategy is the base interface for all strategies
type Strategy interface {
	// Name returns the unique name of the strategy
	Name() string
	// Type returns the type of the strategy
	Type() StrategyType
	// Description returns a human-readable description
	Description() string
}

// BaseStrategy provides common implementation for strategies
type BaseStrategy struct {
	name         string
	strategyType StrategyType
	description  string
}

// NewBaseStrategy creates a new BaseStrategy
func NewBaseStrategy(name string, strategyType StrategyType, description string) BaseStrategy {
	return BaseStrategy{
		name:         name,
		strategyType: strategyType,
		description:  description,
	}
}

// Name returns the strategy name
func (s BaseStrategy) Name() string {
	return s.name
}

// Type returns the strategy type
func (s BaseStrategy) Type() StrategyType {
	return s.strategyType
}

// Description returns the strategy description
func (s BaseStrategy) Description() string {
	return s.description
}
