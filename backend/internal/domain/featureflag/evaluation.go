package featureflag

import (
	"maps"
	"time"
)

// EvaluationReason describes why a particular evaluation result was returned
type EvaluationReason string

const (
	// EvaluationReasonOverrideUser indicates the result came from a user-level override
	EvaluationReasonOverrideUser EvaluationReason = "override_user"
	// EvaluationReasonOverrideTenant indicates the result came from a tenant-level override
	EvaluationReasonOverrideTenant EvaluationReason = "override_tenant"
	// EvaluationReasonRuleMatch indicates the result came from a matched targeting rule
	EvaluationReasonRuleMatch EvaluationReason = "rule_match"
	// EvaluationReasonPercentage indicates the result came from percentage rollout
	EvaluationReasonPercentage EvaluationReason = "percentage"
	// EvaluationReasonDefault indicates the result came from the default value
	EvaluationReasonDefault EvaluationReason = "default"
	// EvaluationReasonDisabled indicates the flag was disabled
	EvaluationReasonDisabled EvaluationReason = "disabled"
	// EvaluationReasonFlagNotFound indicates the flag was not found
	EvaluationReasonFlagNotFound EvaluationReason = "flag_not_found"
	// EvaluationReasonError indicates an error occurred during evaluation
	EvaluationReasonError EvaluationReason = "error"
)

// String returns the string representation of the evaluation reason
func (r EvaluationReason) String() string {
	return string(r)
}

// EvaluationContext represents the context for evaluating a feature flag.
// It contains all the information needed to determine which value to return.
type EvaluationContext struct {
	// TenantID is the ID of the tenant making the request
	TenantID string `json:"tenant_id,omitempty"`

	// UserID is the ID of the user making the request
	UserID string `json:"user_id,omitempty"`

	// UserRole is the role of the user (e.g., "admin", "user", "viewer")
	UserRole string `json:"user_role,omitempty"`

	// UserPlan is the subscription plan of the user (e.g., "free", "pro", "enterprise")
	UserPlan string `json:"user_plan,omitempty"`

	// UserAttributes contains additional user attributes for targeting rules
	UserAttributes map[string]any `json:"user_attributes,omitempty"`

	// RequestID is the unique identifier for the current request (for tracing)
	RequestID string `json:"request_id,omitempty"`

	// Timestamp is the time of evaluation (useful for time-based rules)
	Timestamp time.Time `json:"timestamp"`

	// Environment is the deployment environment (e.g., "development", "staging", "production")
	Environment string `json:"environment,omitempty"`
}

// NewEvaluationContext creates a new evaluation context with the current timestamp
func NewEvaluationContext() *EvaluationContext {
	return &EvaluationContext{
		Timestamp:      time.Now(),
		UserAttributes: make(map[string]any),
	}
}

// WithTenant returns a copy of the context with the tenant ID set
func (c *EvaluationContext) WithTenant(tenantID string) *EvaluationContext {
	return &EvaluationContext{
		TenantID:       tenantID,
		UserID:         c.UserID,
		UserRole:       c.UserRole,
		UserPlan:       c.UserPlan,
		UserAttributes: copyMap(c.UserAttributes),
		RequestID:      c.RequestID,
		Timestamp:      c.Timestamp,
		Environment:    c.Environment,
	}
}

// WithUser returns a copy of the context with the user ID set
func (c *EvaluationContext) WithUser(userID string) *EvaluationContext {
	return &EvaluationContext{
		TenantID:       c.TenantID,
		UserID:         userID,
		UserRole:       c.UserRole,
		UserPlan:       c.UserPlan,
		UserAttributes: copyMap(c.UserAttributes),
		RequestID:      c.RequestID,
		Timestamp:      c.Timestamp,
		Environment:    c.Environment,
	}
}

// WithUserRole returns a copy of the context with the user role set
func (c *EvaluationContext) WithUserRole(role string) *EvaluationContext {
	return &EvaluationContext{
		TenantID:       c.TenantID,
		UserID:         c.UserID,
		UserRole:       role,
		UserPlan:       c.UserPlan,
		UserAttributes: copyMap(c.UserAttributes),
		RequestID:      c.RequestID,
		Timestamp:      c.Timestamp,
		Environment:    c.Environment,
	}
}

// WithUserPlan returns a copy of the context with the user plan set
func (c *EvaluationContext) WithUserPlan(plan string) *EvaluationContext {
	return &EvaluationContext{
		TenantID:       c.TenantID,
		UserID:         c.UserID,
		UserRole:       c.UserRole,
		UserPlan:       plan,
		UserAttributes: copyMap(c.UserAttributes),
		RequestID:      c.RequestID,
		Timestamp:      c.Timestamp,
		Environment:    c.Environment,
	}
}

// WithAttribute returns a copy of the context with an additional user attribute
func (c *EvaluationContext) WithAttribute(key string, value any) *EvaluationContext {
	newAttrs := copyMap(c.UserAttributes)
	newAttrs[key] = value
	return &EvaluationContext{
		TenantID:       c.TenantID,
		UserID:         c.UserID,
		UserRole:       c.UserRole,
		UserPlan:       c.UserPlan,
		UserAttributes: newAttrs,
		RequestID:      c.RequestID,
		Timestamp:      c.Timestamp,
		Environment:    c.Environment,
	}
}

// WithRequestID returns a copy of the context with the request ID set
func (c *EvaluationContext) WithRequestID(requestID string) *EvaluationContext {
	return &EvaluationContext{
		TenantID:       c.TenantID,
		UserID:         c.UserID,
		UserRole:       c.UserRole,
		UserPlan:       c.UserPlan,
		UserAttributes: copyMap(c.UserAttributes),
		RequestID:      requestID,
		Timestamp:      c.Timestamp,
		Environment:    c.Environment,
	}
}

// WithEnvironment returns a copy of the context with the environment set
func (c *EvaluationContext) WithEnvironment(env string) *EvaluationContext {
	return &EvaluationContext{
		TenantID:       c.TenantID,
		UserID:         c.UserID,
		UserRole:       c.UserRole,
		UserPlan:       c.UserPlan,
		UserAttributes: copyMap(c.UserAttributes),
		RequestID:      c.RequestID,
		Timestamp:      c.Timestamp,
		Environment:    env,
	}
}

// WithTimestamp returns a copy of the context with a custom timestamp
func (c *EvaluationContext) WithTimestamp(timestamp time.Time) *EvaluationContext {
	return &EvaluationContext{
		TenantID:       c.TenantID,
		UserID:         c.UserID,
		UserRole:       c.UserRole,
		UserPlan:       c.UserPlan,
		UserAttributes: copyMap(c.UserAttributes),
		RequestID:      c.RequestID,
		Timestamp:      timestamp,
		Environment:    c.Environment,
	}
}

// GetAttribute returns the value of a user attribute
func (c *EvaluationContext) GetAttribute(key string) (any, bool) {
	if c.UserAttributes == nil {
		return nil, false
	}
	val, ok := c.UserAttributes[key]
	return val, ok
}

// GetAttributeString returns the value of a user attribute as a string
func (c *EvaluationContext) GetAttributeString(key string) string {
	val, ok := c.GetAttribute(key)
	if !ok {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

// HasUser returns true if the context has a user ID
func (c *EvaluationContext) HasUser() bool {
	return c.UserID != ""
}

// HasTenant returns true if the context has a tenant ID
func (c *EvaluationContext) HasTenant() bool {
	return c.TenantID != ""
}

// EvaluationResult represents the result of evaluating a feature flag
type EvaluationResult struct {
	// Key is the key of the evaluated flag
	Key string `json:"key"`

	// Enabled indicates whether the feature is enabled
	Enabled bool `json:"enabled"`

	// Variant is the variant name for A/B testing flags
	Variant string `json:"variant,omitempty"`

	// Value is the complete flag value (for more complex use cases)
	Value FlagValue `json:"value"`

	// Reason explains why this result was returned
	Reason EvaluationReason `json:"reason"`

	// RuleID is the ID of the rule that matched (if applicable)
	RuleID string `json:"rule_id,omitempty"`

	// FlagVersion is the version of the flag at the time of evaluation
	FlagVersion int `json:"flag_version"`

	// EvaluatedAt is the timestamp when the evaluation occurred
	EvaluatedAt time.Time `json:"evaluated_at"`

	// Error contains any error that occurred during evaluation
	Error error `json:"-"`
}

// NewEvaluationResult creates a new evaluation result
func NewEvaluationResult(key string, value FlagValue, reason EvaluationReason, flagVersion int) EvaluationResult {
	return EvaluationResult{
		Key:         key,
		Enabled:     value.Enabled,
		Variant:     value.Variant,
		Value:       value,
		Reason:      reason,
		FlagVersion: flagVersion,
		EvaluatedAt: time.Now(),
	}
}

// NewDisabledResult creates an evaluation result for a disabled flag
func NewDisabledResult(key string, defaultValue FlagValue, flagVersion int) EvaluationResult {
	return EvaluationResult{
		Key:         key,
		Enabled:     false,
		Variant:     defaultValue.Variant,
		Value:       NewBooleanFlagValue(false),
		Reason:      EvaluationReasonDisabled,
		FlagVersion: flagVersion,
		EvaluatedAt: time.Now(),
	}
}

// NewFlagNotFoundResult creates an evaluation result for a flag that was not found
func NewFlagNotFoundResult(key string) EvaluationResult {
	return EvaluationResult{
		Key:         key,
		Enabled:     false,
		Value:       NewBooleanFlagValue(false),
		Reason:      EvaluationReasonFlagNotFound,
		FlagVersion: 0,
		EvaluatedAt: time.Now(),
	}
}

// NewErrorResult creates an evaluation result for an error
func NewErrorResult(key string, err error) EvaluationResult {
	return EvaluationResult{
		Key:         key,
		Enabled:     false,
		Value:       NewBooleanFlagValue(false),
		Reason:      EvaluationReasonError,
		FlagVersion: 0,
		EvaluatedAt: time.Now(),
		Error:       err,
	}
}

// WithRuleID returns a copy of the result with the rule ID set
func (r EvaluationResult) WithRuleID(ruleID string) EvaluationResult {
	return EvaluationResult{
		Key:         r.Key,
		Enabled:     r.Enabled,
		Variant:     r.Variant,
		Value:       r.Value,
		Reason:      r.Reason,
		RuleID:      ruleID,
		FlagVersion: r.FlagVersion,
		EvaluatedAt: r.EvaluatedAt,
		Error:       r.Error,
	}
}

// IsEnabled returns true if the feature is enabled
func (r EvaluationResult) IsEnabled() bool {
	return r.Enabled
}

// IsDisabled returns true if the feature is disabled
func (r EvaluationResult) IsDisabled() bool {
	return !r.Enabled
}

// HasVariant returns true if the result has a variant
func (r EvaluationResult) HasVariant() bool {
	return r.Variant != ""
}

// HasError returns true if an error occurred during evaluation
func (r EvaluationResult) HasError() bool {
	return r.Error != nil
}

// GetError returns the error that occurred during evaluation
func (r EvaluationResult) GetError() error {
	return r.Error
}

// IsFromOverride returns true if the result came from an override
func (r EvaluationResult) IsFromOverride() bool {
	return r.Reason == EvaluationReasonOverrideUser || r.Reason == EvaluationReasonOverrideTenant
}

// IsFromRule returns true if the result came from a targeting rule
func (r EvaluationResult) IsFromRule() bool {
	return r.Reason == EvaluationReasonRuleMatch || r.Reason == EvaluationReasonPercentage
}

// IsDefault returns true if the result is the default value
func (r EvaluationResult) IsDefault() bool {
	return r.Reason == EvaluationReasonDefault
}

// copyMap creates a shallow copy of a map
func copyMap(m map[string]any) map[string]any {
	if m == nil {
		return make(map[string]any)
	}
	result := make(map[string]any, len(m))
	maps.Copy(result, m)
	return result
}
