package plugin

// AttributeDefinition defines a product attribute for industry-specific validation
type AttributeDefinition struct {
	// Key is the attribute key, e.g., "registration_number"
	Key string
	// Label is the display name, e.g., "农药登记证号"
	Label string
	// Required indicates if this attribute is mandatory
	Required bool
	// Regex is an optional validation pattern
	Regex string
	// CategoryCodes specifies which product categories require this attribute
	// If empty, applies to all products in the industry
	CategoryCodes []string
}

// IndustryPlugin defines the interface for industry-specific plugins
// Plugins extend the system to support specific industry requirements
type IndustryPlugin interface {
	// Name returns the unique identifier for the plugin
	Name() string
	// DisplayName returns the human-readable name for the plugin
	DisplayName() string
	// RegisterStrategies registers industry-specific strategies with the registry
	RegisterStrategies(registry StrategyRegistrar)
	// GetRequiredProductAttributes returns the attribute definitions for this industry
	GetRequiredProductAttributes() []AttributeDefinition
}

// StrategyRegistrar is the interface for registering strategies
// This is implemented by the infrastructure StrategyRegistry
type StrategyRegistrar interface {
	// RegisterCostStrategy registers a cost calculation strategy
	RegisterCostStrategy(s any) error
	// RegisterPricingStrategy registers a pricing strategy
	RegisterPricingStrategy(s any) error
	// RegisterAllocationStrategy registers a payment allocation strategy
	RegisterAllocationStrategy(s any) error
	// RegisterBatchStrategy registers a batch management strategy
	RegisterBatchStrategy(s any) error
	// RegisterValidationStrategy registers a product validation strategy
	RegisterValidationStrategy(s any) error
}
