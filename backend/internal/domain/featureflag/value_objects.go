package featureflag

import (
	"reflect"

	"github.com/erp/backend/internal/domain/shared"
)

// FlagValue represents the value of a feature flag
// It is a value object that encapsulates the flag's enabled state,
// variant (for A/B testing), and additional metadata
type FlagValue struct {
	Enabled  bool           `json:"enabled"`
	Variant  string         `json:"variant,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// NewBooleanFlagValue creates a new boolean flag value
func NewBooleanFlagValue(enabled bool) FlagValue {
	return FlagValue{
		Enabled:  enabled,
		Metadata: make(map[string]any),
	}
}

// NewVariantFlagValue creates a new variant flag value
func NewVariantFlagValue(variant string) FlagValue {
	return FlagValue{
		Enabled:  true,
		Variant:  variant,
		Metadata: make(map[string]any),
	}
}

// NewFlagValueWithMetadata creates a new flag value with metadata
func NewFlagValueWithMetadata(enabled bool, variant string, metadata map[string]any) FlagValue {
	if metadata == nil {
		metadata = make(map[string]any)
	}
	return FlagValue{
		Enabled:  enabled,
		Variant:  variant,
		Metadata: metadata,
	}
}

// IsEnabled returns whether the flag is enabled
func (v FlagValue) IsEnabled() bool {
	return v.Enabled
}

// GetVariant returns the variant value
func (v FlagValue) GetVariant() string {
	return v.Variant
}

// HasVariant returns true if the flag has a variant set
func (v FlagValue) HasVariant() bool {
	return v.Variant != ""
}

// GetMetadata returns the metadata map
func (v FlagValue) GetMetadata() map[string]any {
	if v.Metadata == nil {
		return make(map[string]any)
	}
	// Return a copy to maintain immutability
	result := make(map[string]any, len(v.Metadata))
	for k, val := range v.Metadata {
		result[k] = val
	}
	return result
}

// GetMetadataValue returns a specific metadata value
func (v FlagValue) GetMetadataValue(key string) (any, bool) {
	if v.Metadata == nil {
		return nil, false
	}
	val, ok := v.Metadata[key]
	return val, ok
}

// WithMetadata returns a new FlagValue with the specified metadata key-value pair
func (v FlagValue) WithMetadata(key string, value any) FlagValue {
	newMetadata := v.GetMetadata()
	newMetadata[key] = value
	return FlagValue{
		Enabled:  v.Enabled,
		Variant:  v.Variant,
		Metadata: newMetadata,
	}
}

// Equals checks if two FlagValues are equal
// Uses reflect.DeepEqual for proper comparison of metadata maps that may contain nested values
func (v FlagValue) Equals(other FlagValue) bool {
	if v.Enabled != other.Enabled || v.Variant != other.Variant {
		return false
	}
	return reflect.DeepEqual(v.Metadata, other.Metadata)
}

// Condition represents a condition for targeting rules
// Conditions are used to match user attributes for targeted rollouts
type Condition struct {
	Attribute string            `json:"attribute"`
	Operator  ConditionOperator `json:"operator"`
	Values    []string          `json:"values"`
}

// NewCondition creates a new condition
func NewCondition(attribute string, operator ConditionOperator, values []string) (Condition, error) {
	if attribute == "" {
		return Condition{}, shared.NewDomainError("INVALID_CONDITION", "Condition attribute cannot be empty")
	}
	if !operator.IsValid() {
		return Condition{}, shared.NewDomainError("INVALID_OPERATOR", "Invalid condition operator")
	}
	if len(values) == 0 {
		return Condition{}, shared.NewDomainError("INVALID_CONDITION", "Condition must have at least one value")
	}

	// Copy values slice to maintain immutability
	valuesCopy := make([]string, len(values))
	copy(valuesCopy, values)

	return Condition{
		Attribute: attribute,
		Operator:  operator,
		Values:    valuesCopy,
	}, nil
}

// GetAttribute returns the attribute name
func (c Condition) GetAttribute() string {
	return c.Attribute
}

// GetOperator returns the operator
func (c Condition) GetOperator() ConditionOperator {
	return c.Operator
}

// GetValues returns a copy of the values
func (c Condition) GetValues() []string {
	if c.Values == nil {
		return []string{}
	}
	result := make([]string, len(c.Values))
	copy(result, c.Values)
	return result
}

// Validate checks if the condition is valid
func (c Condition) Validate() error {
	if c.Attribute == "" {
		return shared.NewDomainError("INVALID_CONDITION", "Condition attribute cannot be empty")
	}
	if !c.Operator.IsValid() {
		return shared.NewDomainError("INVALID_OPERATOR", "Invalid condition operator")
	}
	if len(c.Values) == 0 {
		return shared.NewDomainError("INVALID_CONDITION", "Condition must have at least one value")
	}
	return nil
}

// TargetingRule represents a rule for targeting specific users or groups
// Rules are evaluated in order of priority (lower number = higher priority)
type TargetingRule struct {
	RuleID     string      `json:"rule_id"`
	Priority   int         `json:"priority"`
	Conditions []Condition `json:"conditions"`
	Value      FlagValue   `json:"value"`
	Percentage int         `json:"percentage"` // 0-100, for percentage-based rollouts within matched users
}

// NewTargetingRule creates a new targeting rule
func NewTargetingRule(ruleID string, priority int, conditions []Condition, value FlagValue) (TargetingRule, error) {
	if ruleID == "" {
		return TargetingRule{}, shared.NewDomainError("INVALID_RULE", "Rule ID cannot be empty")
	}
	if priority < 0 {
		return TargetingRule{}, shared.NewDomainError("INVALID_RULE", "Rule priority cannot be negative")
	}

	return TargetingRule{
		RuleID:     ruleID,
		Priority:   priority,
		Conditions: conditions,
		Value:      value,
		Percentage: 100, // Default to 100% of matched users
	}, nil
}

// NewTargetingRuleWithPercentage creates a new targeting rule with percentage
func NewTargetingRuleWithPercentage(ruleID string, priority int, conditions []Condition, value FlagValue, percentage int) (TargetingRule, error) {
	if percentage < 0 || percentage > 100 {
		return TargetingRule{}, shared.NewDomainError("INVALID_PERCENTAGE", "Percentage must be between 0 and 100")
	}

	rule, err := NewTargetingRule(ruleID, priority, conditions, value)
	if err != nil {
		return TargetingRule{}, err
	}
	rule.Percentage = percentage
	return rule, nil
}

// GetRuleID returns the rule ID
func (r TargetingRule) GetRuleID() string {
	return r.RuleID
}

// GetPriority returns the rule priority
func (r TargetingRule) GetPriority() int {
	return r.Priority
}

// GetConditions returns a copy of the conditions
func (r TargetingRule) GetConditions() []Condition {
	if r.Conditions == nil {
		return []Condition{}
	}
	result := make([]Condition, len(r.Conditions))
	copy(result, r.Conditions)
	return result
}

// GetValue returns the flag value for this rule
func (r TargetingRule) GetValue() FlagValue {
	return r.Value
}

// GetPercentage returns the percentage rollout
func (r TargetingRule) GetPercentage() int {
	return r.Percentage
}

// HasConditions returns true if the rule has conditions
func (r TargetingRule) HasConditions() bool {
	return len(r.Conditions) > 0
}

// Validate checks if the rule is valid
func (r TargetingRule) Validate() error {
	if r.RuleID == "" {
		return shared.NewDomainError("INVALID_RULE", "Rule ID cannot be empty")
	}
	if r.Priority < 0 {
		return shared.NewDomainError("INVALID_RULE", "Rule priority cannot be negative")
	}
	if r.Percentage < 0 || r.Percentage > 100 {
		return shared.NewDomainError("INVALID_PERCENTAGE", "Percentage must be between 0 and 100")
	}
	for _, condition := range r.Conditions {
		if err := condition.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// AddCondition returns a new rule with the additional condition
func (r TargetingRule) AddCondition(condition Condition) (TargetingRule, error) {
	if err := condition.Validate(); err != nil {
		return TargetingRule{}, err
	}
	newConditions := make([]Condition, len(r.Conditions)+1)
	copy(newConditions, r.Conditions)
	newConditions[len(r.Conditions)] = condition
	return TargetingRule{
		RuleID:     r.RuleID,
		Priority:   r.Priority,
		Conditions: newConditions,
		Value:      r.Value,
		Percentage: r.Percentage,
	}, nil
}

// WithPriority returns a new rule with the specified priority
func (r TargetingRule) WithPriority(priority int) (TargetingRule, error) {
	if priority < 0 {
		return TargetingRule{}, shared.NewDomainError("INVALID_RULE", "Rule priority cannot be negative")
	}
	return TargetingRule{
		RuleID:     r.RuleID,
		Priority:   priority,
		Conditions: r.GetConditions(),
		Value:      r.Value,
		Percentage: r.Percentage,
	}, nil
}

// WithPercentage returns a new rule with the specified percentage
func (r TargetingRule) WithPercentage(percentage int) (TargetingRule, error) {
	if percentage < 0 || percentage > 100 {
		return TargetingRule{}, shared.NewDomainError("INVALID_PERCENTAGE", "Percentage must be between 0 and 100")
	}
	return TargetingRule{
		RuleID:     r.RuleID,
		Priority:   r.Priority,
		Conditions: r.GetConditions(),
		Value:      r.Value,
		Percentage: percentage,
	}, nil
}
