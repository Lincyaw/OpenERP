package featureflag

import (
	"fmt"
	"strconv"
	"strings"
)

// MatchCondition evaluates whether the given condition matches the evaluation context.
// It supports all operators: EQUALS, NOT_EQUALS, IN, NOT_IN, CONTAINS, GREATER_THAN, LESS_THAN.
//
// The attribute is looked up in the following order:
// 1. Built-in attributes: tenant_id, user_id, user_role, user_plan, environment
// 2. User attributes from the UserAttributes map
func MatchCondition(condition Condition, ctx *EvaluationContext) bool {
	if ctx == nil {
		return false
	}

	// Get the attribute value from the context
	attrValue := getAttributeValue(condition.Attribute, ctx)

	// Apply the operator
	return applyOperator(condition.Operator, attrValue, condition.Values)
}

// MatchAllConditions returns true if ALL conditions match the context (AND logic)
func MatchAllConditions(conditions []Condition, ctx *EvaluationContext) bool {
	if ctx == nil {
		return false
	}

	for _, condition := range conditions {
		if !MatchCondition(condition, ctx) {
			return false
		}
	}
	return true
}

// MatchAnyCondition returns true if ANY condition matches the context (OR logic)
func MatchAnyCondition(conditions []Condition, ctx *EvaluationContext) bool {
	if ctx == nil || len(conditions) == 0 {
		return false
	}

	for _, condition := range conditions {
		if MatchCondition(condition, ctx) {
			return true
		}
	}
	return false
}

// getAttributeValue retrieves the value of an attribute from the evaluation context
func getAttributeValue(attribute string, ctx *EvaluationContext) any {
	// Check built-in attributes first
	switch strings.ToLower(attribute) {
	case "tenant_id", "tenantid":
		return ctx.TenantID
	case "user_id", "userid":
		return ctx.UserID
	case "user_role", "userrole", "role":
		return ctx.UserRole
	case "user_plan", "userplan", "plan":
		return ctx.UserPlan
	case "environment", "env":
		return ctx.Environment
	case "request_id", "requestid":
		return ctx.RequestID
	}

	// Check user attributes
	if ctx.UserAttributes != nil {
		if val, ok := ctx.UserAttributes[attribute]; ok {
			return val
		}
	}

	return nil
}

// applyOperator applies the given operator to compare the attribute value with the condition values
func applyOperator(op ConditionOperator, attrValue any, condValues []string) bool {
	switch op {
	case ConditionOperatorEquals:
		return operatorEquals(attrValue, condValues)
	case ConditionOperatorNotEquals:
		return !operatorEquals(attrValue, condValues)
	case ConditionOperatorIn:
		return operatorIn(attrValue, condValues)
	case ConditionOperatorNotIn:
		return !operatorIn(attrValue, condValues)
	case ConditionOperatorContains:
		return operatorContains(attrValue, condValues)
	case ConditionOperatorGreaterThan:
		return operatorGreaterThan(attrValue, condValues)
	case ConditionOperatorLessThan:
		return operatorLessThan(attrValue, condValues)
	default:
		return false
	}
}

// operatorEquals checks if the attribute value equals any of the condition values
// For a single condition value, checks exact equality.
// For multiple condition values, checks if the attribute value equals any of them.
func operatorEquals(attrValue any, condValues []string) bool {
	if attrValue == nil || len(condValues) == 0 {
		return false
	}

	attrStr := toString(attrValue)
	for _, condValue := range condValues {
		if strings.EqualFold(attrStr, condValue) {
			return true
		}
	}
	return false
}

// operatorIn checks if the attribute value is in the list of condition values
// Case-insensitive comparison
func operatorIn(attrValue any, condValues []string) bool {
	if attrValue == nil || len(condValues) == 0 {
		return false
	}

	attrStr := strings.ToLower(toString(attrValue))
	for _, condValue := range condValues {
		if strings.ToLower(condValue) == attrStr {
			return true
		}
	}
	return false
}

// operatorContains checks if the attribute value contains any of the condition values
// Case-insensitive comparison
func operatorContains(attrValue any, condValues []string) bool {
	if attrValue == nil || len(condValues) == 0 {
		return false
	}

	attrStr := strings.ToLower(toString(attrValue))
	for _, condValue := range condValues {
		if strings.Contains(attrStr, strings.ToLower(condValue)) {
			return true
		}
	}
	return false
}

// operatorGreaterThan checks if the attribute value is greater than the condition value
// Supports numeric and string comparisons
func operatorGreaterThan(attrValue any, condValues []string) bool {
	if attrValue == nil || len(condValues) == 0 {
		return false
	}

	// Try numeric comparison first
	if attrNum, ok := toFloat64(attrValue); ok {
		if condNum, err := strconv.ParseFloat(condValues[0], 64); err == nil {
			return attrNum > condNum
		}
	}

	// Fall back to string comparison
	attrStr := toString(attrValue)
	return attrStr > condValues[0]
}

// operatorLessThan checks if the attribute value is less than the condition value
// Supports numeric and string comparisons
func operatorLessThan(attrValue any, condValues []string) bool {
	if attrValue == nil || len(condValues) == 0 {
		return false
	}

	// Try numeric comparison first
	if attrNum, ok := toFloat64(attrValue); ok {
		if condNum, err := strconv.ParseFloat(condValues[0], 64); err == nil {
			return attrNum < condNum
		}
	}

	// Fall back to string comparison
	attrStr := toString(attrValue)
	return attrStr < condValues[0]
}

// toString converts any value to a string representation
func toString(value any) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// toFloat64 attempts to convert any value to float64
func toFloat64(value any) (float64, bool) {
	if value == nil {
		return 0, false
	}

	switch v := value.(type) {
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, true
		}
		return 0, false
	default:
		return 0, false
	}
}

// ConditionMatcher provides a stateful condition matcher that can be configured
type ConditionMatcher struct{}

// NewConditionMatcher creates a new condition matcher
func NewConditionMatcher() *ConditionMatcher {
	return &ConditionMatcher{}
}

// Match evaluates whether the given condition matches the evaluation context
func (m *ConditionMatcher) Match(condition Condition, ctx *EvaluationContext) bool {
	return MatchCondition(condition, ctx)
}

// MatchAll returns true if ALL conditions match the context (AND logic)
func (m *ConditionMatcher) MatchAll(conditions []Condition, ctx *EvaluationContext) bool {
	return MatchAllConditions(conditions, ctx)
}

// MatchAny returns true if ANY condition matches the context (OR logic)
func (m *ConditionMatcher) MatchAny(conditions []Condition, ctx *EvaluationContext) bool {
	return MatchAnyCondition(conditions, ctx)
}
