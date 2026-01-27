package featureflag

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchCondition_Equals(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		ctx       *EvaluationContext
		expected  bool
	}{
		{
			name: "equals user_id - match",
			condition: Condition{
				Attribute: "user_id",
				Operator:  ConditionOperatorEquals,
				Values:    []string{"user-123"},
			},
			ctx:      NewEvaluationContext().WithUser("user-123"),
			expected: true,
		},
		{
			name: "equals user_id - no match",
			condition: Condition{
				Attribute: "user_id",
				Operator:  ConditionOperatorEquals,
				Values:    []string{"user-123"},
			},
			ctx:      NewEvaluationContext().WithUser("user-456"),
			expected: false,
		},
		{
			name: "equals case insensitive",
			condition: Condition{
				Attribute: "user_role",
				Operator:  ConditionOperatorEquals,
				Values:    []string{"ADMIN"},
			},
			ctx:      NewEvaluationContext().WithUserRole("admin"),
			expected: true,
		},
		{
			name: "equals multiple values - match first",
			condition: Condition{
				Attribute: "environment",
				Operator:  ConditionOperatorEquals,
				Values:    []string{"production", "staging"},
			},
			ctx:      NewEvaluationContext().WithEnvironment("production"),
			expected: true,
		},
		{
			name: "equals multiple values - match second",
			condition: Condition{
				Attribute: "environment",
				Operator:  ConditionOperatorEquals,
				Values:    []string{"production", "staging"},
			},
			ctx:      NewEvaluationContext().WithEnvironment("staging"),
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := MatchCondition(tc.condition, tc.ctx)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMatchCondition_NotEquals(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		ctx       *EvaluationContext
		expected  bool
	}{
		{
			name: "not equals - different value",
			condition: Condition{
				Attribute: "user_id",
				Operator:  ConditionOperatorNotEquals,
				Values:    []string{"user-123"},
			},
			ctx:      NewEvaluationContext().WithUser("user-456"),
			expected: true,
		},
		{
			name: "not equals - same value",
			condition: Condition{
				Attribute: "user_id",
				Operator:  ConditionOperatorNotEquals,
				Values:    []string{"user-123"},
			},
			ctx:      NewEvaluationContext().WithUser("user-123"),
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := MatchCondition(tc.condition, tc.ctx)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMatchCondition_In(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		ctx       *EvaluationContext
		expected  bool
	}{
		{
			name: "in list - match",
			condition: Condition{
				Attribute: "user_plan",
				Operator:  ConditionOperatorIn,
				Values:    []string{"pro", "enterprise", "unlimited"},
			},
			ctx:      NewEvaluationContext().WithUserPlan("enterprise"),
			expected: true,
		},
		{
			name: "in list - no match",
			condition: Condition{
				Attribute: "user_plan",
				Operator:  ConditionOperatorIn,
				Values:    []string{"pro", "enterprise", "unlimited"},
			},
			ctx:      NewEvaluationContext().WithUserPlan("free"),
			expected: false,
		},
		{
			name: "in list - case insensitive",
			condition: Condition{
				Attribute: "user_plan",
				Operator:  ConditionOperatorIn,
				Values:    []string{"PRO", "ENTERPRISE"},
			},
			ctx:      NewEvaluationContext().WithUserPlan("pro"),
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := MatchCondition(tc.condition, tc.ctx)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMatchCondition_NotIn(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		ctx       *EvaluationContext
		expected  bool
	}{
		{
			name: "not in list - value not in list",
			condition: Condition{
				Attribute: "user_plan",
				Operator:  ConditionOperatorNotIn,
				Values:    []string{"pro", "enterprise"},
			},
			ctx:      NewEvaluationContext().WithUserPlan("free"),
			expected: true,
		},
		{
			name: "not in list - value in list",
			condition: Condition{
				Attribute: "user_plan",
				Operator:  ConditionOperatorNotIn,
				Values:    []string{"pro", "enterprise"},
			},
			ctx:      NewEvaluationContext().WithUserPlan("pro"),
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := MatchCondition(tc.condition, tc.ctx)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMatchCondition_Contains(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		ctx       *EvaluationContext
		expected  bool
	}{
		{
			name: "contains - match",
			condition: Condition{
				Attribute: "email",
				Operator:  ConditionOperatorContains,
				Values:    []string{"@company.com"},
			},
			ctx:      NewEvaluationContext().WithAttribute("email", "user@company.com"),
			expected: true,
		},
		{
			name: "contains - no match",
			condition: Condition{
				Attribute: "email",
				Operator:  ConditionOperatorContains,
				Values:    []string{"@company.com"},
			},
			ctx:      NewEvaluationContext().WithAttribute("email", "user@other.com"),
			expected: false,
		},
		{
			name: "contains - case insensitive",
			condition: Condition{
				Attribute: "email",
				Operator:  ConditionOperatorContains,
				Values:    []string{"@COMPANY.COM"},
			},
			ctx:      NewEvaluationContext().WithAttribute("email", "user@company.com"),
			expected: true,
		},
		{
			name: "contains - multiple values, any match",
			condition: Condition{
				Attribute: "email",
				Operator:  ConditionOperatorContains,
				Values:    []string{"@company.com", "@partner.com"},
			},
			ctx:      NewEvaluationContext().WithAttribute("email", "user@partner.com"),
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := MatchCondition(tc.condition, tc.ctx)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMatchCondition_GreaterThan(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		ctx       *EvaluationContext
		expected  bool
	}{
		{
			name: "greater than - numeric match",
			condition: Condition{
				Attribute: "age",
				Operator:  ConditionOperatorGreaterThan,
				Values:    []string{"18"},
			},
			ctx:      NewEvaluationContext().WithAttribute("age", 25),
			expected: true,
		},
		{
			name: "greater than - numeric no match",
			condition: Condition{
				Attribute: "age",
				Operator:  ConditionOperatorGreaterThan,
				Values:    []string{"18"},
			},
			ctx:      NewEvaluationContext().WithAttribute("age", 16),
			expected: false,
		},
		{
			name: "greater than - equal value",
			condition: Condition{
				Attribute: "age",
				Operator:  ConditionOperatorGreaterThan,
				Values:    []string{"18"},
			},
			ctx:      NewEvaluationContext().WithAttribute("age", 18),
			expected: false,
		},
		{
			name: "greater than - float values",
			condition: Condition{
				Attribute: "score",
				Operator:  ConditionOperatorGreaterThan,
				Values:    []string{"9.5"},
			},
			ctx:      NewEvaluationContext().WithAttribute("score", 9.8),
			expected: true,
		},
		{
			name: "greater than - string comparison fallback",
			condition: Condition{
				Attribute: "version",
				Operator:  ConditionOperatorGreaterThan,
				Values:    []string{"1.0"},
			},
			ctx:      NewEvaluationContext().WithAttribute("version", "2.0"),
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := MatchCondition(tc.condition, tc.ctx)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMatchCondition_LessThan(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		ctx       *EvaluationContext
		expected  bool
	}{
		{
			name: "less than - numeric match",
			condition: Condition{
				Attribute: "age",
				Operator:  ConditionOperatorLessThan,
				Values:    []string{"18"},
			},
			ctx:      NewEvaluationContext().WithAttribute("age", 16),
			expected: true,
		},
		{
			name: "less than - numeric no match",
			condition: Condition{
				Attribute: "age",
				Operator:  ConditionOperatorLessThan,
				Values:    []string{"18"},
			},
			ctx:      NewEvaluationContext().WithAttribute("age", 25),
			expected: false,
		},
		{
			name: "less than - equal value",
			condition: Condition{
				Attribute: "age",
				Operator:  ConditionOperatorLessThan,
				Values:    []string{"18"},
			},
			ctx:      NewEvaluationContext().WithAttribute("age", 18),
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := MatchCondition(tc.condition, tc.ctx)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMatchCondition_BuiltInAttributes(t *testing.T) {
	tests := []struct {
		name      string
		attribute string
		value     string
		setupCtx  func() *EvaluationContext
	}{
		{
			name:      "tenant_id",
			attribute: "tenant_id",
			value:     "tenant-123",
			setupCtx:  func() *EvaluationContext { return NewEvaluationContext().WithTenant("tenant-123") },
		},
		{
			name:      "tenantid (alias)",
			attribute: "tenantid",
			value:     "tenant-123",
			setupCtx:  func() *EvaluationContext { return NewEvaluationContext().WithTenant("tenant-123") },
		},
		{
			name:      "user_id",
			attribute: "user_id",
			value:     "user-456",
			setupCtx:  func() *EvaluationContext { return NewEvaluationContext().WithUser("user-456") },
		},
		{
			name:      "userid (alias)",
			attribute: "userid",
			value:     "user-456",
			setupCtx:  func() *EvaluationContext { return NewEvaluationContext().WithUser("user-456") },
		},
		{
			name:      "user_role",
			attribute: "user_role",
			value:     "admin",
			setupCtx:  func() *EvaluationContext { return NewEvaluationContext().WithUserRole("admin") },
		},
		{
			name:      "role (alias)",
			attribute: "role",
			value:     "admin",
			setupCtx:  func() *EvaluationContext { return NewEvaluationContext().WithUserRole("admin") },
		},
		{
			name:      "user_plan",
			attribute: "user_plan",
			value:     "enterprise",
			setupCtx:  func() *EvaluationContext { return NewEvaluationContext().WithUserPlan("enterprise") },
		},
		{
			name:      "plan (alias)",
			attribute: "plan",
			value:     "enterprise",
			setupCtx:  func() *EvaluationContext { return NewEvaluationContext().WithUserPlan("enterprise") },
		},
		{
			name:      "environment",
			attribute: "environment",
			value:     "production",
			setupCtx:  func() *EvaluationContext { return NewEvaluationContext().WithEnvironment("production") },
		},
		{
			name:      "env (alias)",
			attribute: "env",
			value:     "production",
			setupCtx:  func() *EvaluationContext { return NewEvaluationContext().WithEnvironment("production") },
		},
		{
			name:      "request_id",
			attribute: "request_id",
			value:     "req-789",
			setupCtx:  func() *EvaluationContext { return NewEvaluationContext().WithRequestID("req-789") },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			condition := Condition{
				Attribute: tc.attribute,
				Operator:  ConditionOperatorEquals,
				Values:    []string{tc.value},
			}
			result := MatchCondition(condition, tc.setupCtx())
			assert.True(t, result)
		})
	}
}

func TestMatchCondition_UserAttributes(t *testing.T) {
	ctx := NewEvaluationContext().
		WithAttribute("country", "US").
		WithAttribute("beta_tester", "true").
		WithAttribute("score", 85)

	tests := []struct {
		name      string
		condition Condition
		expected  bool
	}{
		{
			name: "string attribute",
			condition: Condition{
				Attribute: "country",
				Operator:  ConditionOperatorEquals,
				Values:    []string{"US"},
			},
			expected: true,
		},
		{
			name: "boolean-like attribute",
			condition: Condition{
				Attribute: "beta_tester",
				Operator:  ConditionOperatorEquals,
				Values:    []string{"true"},
			},
			expected: true,
		},
		{
			name: "numeric attribute with greater than",
			condition: Condition{
				Attribute: "score",
				Operator:  ConditionOperatorGreaterThan,
				Values:    []string{"80"},
			},
			expected: true,
		},
		{
			name: "nonexistent attribute",
			condition: Condition{
				Attribute: "nonexistent",
				Operator:  ConditionOperatorEquals,
				Values:    []string{"value"},
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := MatchCondition(tc.condition, ctx)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMatchCondition_NilContext(t *testing.T) {
	condition := Condition{
		Attribute: "user_id",
		Operator:  ConditionOperatorEquals,
		Values:    []string{"user-123"},
	}

	result := MatchCondition(condition, nil)
	assert.False(t, result)
}

func TestMatchCondition_NilAttributeValue(t *testing.T) {
	condition := Condition{
		Attribute: "missing_attr",
		Operator:  ConditionOperatorEquals,
		Values:    []string{"some_value"},
	}

	ctx := NewEvaluationContext()
	result := MatchCondition(condition, ctx)
	assert.False(t, result)
}

func TestMatchCondition_EmptyValues(t *testing.T) {
	condition := Condition{
		Attribute: "user_id",
		Operator:  ConditionOperatorEquals,
		Values:    []string{},
	}

	ctx := NewEvaluationContext().WithUser("user-123")
	result := MatchCondition(condition, ctx)
	assert.False(t, result)
}

func TestMatchAllConditions(t *testing.T) {
	conditions := []Condition{
		{Attribute: "user_role", Operator: ConditionOperatorEquals, Values: []string{"admin"}},
		{Attribute: "environment", Operator: ConditionOperatorEquals, Values: []string{"production"}},
	}

	tests := []struct {
		name     string
		ctx      *EvaluationContext
		expected bool
	}{
		{
			name: "all conditions match",
			ctx: NewEvaluationContext().
				WithUserRole("admin").
				WithEnvironment("production"),
			expected: true,
		},
		{
			name: "first condition fails",
			ctx: NewEvaluationContext().
				WithUserRole("user").
				WithEnvironment("production"),
			expected: false,
		},
		{
			name: "second condition fails",
			ctx: NewEvaluationContext().
				WithUserRole("admin").
				WithEnvironment("staging"),
			expected: false,
		},
		{
			name: "both conditions fail",
			ctx: NewEvaluationContext().
				WithUserRole("user").
				WithEnvironment("staging"),
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := MatchAllConditions(conditions, tc.ctx)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMatchAllConditions_EmptyConditions(t *testing.T) {
	ctx := NewEvaluationContext()
	result := MatchAllConditions([]Condition{}, ctx)
	assert.True(t, result, "Empty conditions should always match")
}

func TestMatchAllConditions_NilContext(t *testing.T) {
	conditions := []Condition{
		{Attribute: "user_id", Operator: ConditionOperatorEquals, Values: []string{"user-123"}},
	}
	result := MatchAllConditions(conditions, nil)
	assert.False(t, result)
}

func TestMatchAnyCondition(t *testing.T) {
	conditions := []Condition{
		{Attribute: "user_role", Operator: ConditionOperatorEquals, Values: []string{"admin"}},
		{Attribute: "user_role", Operator: ConditionOperatorEquals, Values: []string{"moderator"}},
	}

	tests := []struct {
		name     string
		ctx      *EvaluationContext
		expected bool
	}{
		{
			name:     "first condition matches",
			ctx:      NewEvaluationContext().WithUserRole("admin"),
			expected: true,
		},
		{
			name:     "second condition matches",
			ctx:      NewEvaluationContext().WithUserRole("moderator"),
			expected: true,
		},
		{
			name:     "no conditions match",
			ctx:      NewEvaluationContext().WithUserRole("user"),
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := MatchAnyCondition(conditions, tc.ctx)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMatchAnyCondition_EmptyConditions(t *testing.T) {
	ctx := NewEvaluationContext()
	result := MatchAnyCondition([]Condition{}, ctx)
	assert.False(t, result, "Empty conditions should not match for ANY")
}

func TestMatchAnyCondition_NilContext(t *testing.T) {
	conditions := []Condition{
		{Attribute: "user_id", Operator: ConditionOperatorEquals, Values: []string{"user-123"}},
	}
	result := MatchAnyCondition(conditions, nil)
	assert.False(t, result)
}

func TestConditionMatcher(t *testing.T) {
	matcher := NewConditionMatcher()

	condition := Condition{
		Attribute: "user_role",
		Operator:  ConditionOperatorEquals,
		Values:    []string{"admin"},
	}
	ctx := NewEvaluationContext().WithUserRole("admin")

	assert.True(t, matcher.Match(condition, ctx))

	conditions := []Condition{
		{Attribute: "user_role", Operator: ConditionOperatorEquals, Values: []string{"admin"}},
		{Attribute: "environment", Operator: ConditionOperatorEquals, Values: []string{"production"}},
	}

	ctxBoth := ctx.WithEnvironment("production")
	assert.True(t, matcher.MatchAll(conditions, ctxBoth))
	assert.True(t, matcher.MatchAny(conditions, ctxBoth))

	ctxPartial := ctx.WithEnvironment("staging")
	assert.False(t, matcher.MatchAll(conditions, ctxPartial))
	assert.True(t, matcher.MatchAny(conditions, ctxPartial))
}

func TestToString(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{"nil", nil, ""},
		{"string", "hello", "hello"},
		{"int", 42, "42"},
		{"int32", int32(42), "42"},
		{"int64", int64(42), "42"},
		{"float32", float32(3.14), "3.14"},
		{"float64", 3.14159, "3.14159"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"struct", struct{ Name string }{"test"}, "{test}"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := toString(tc.value)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected float64
		ok       bool
	}{
		{"nil", nil, 0, false},
		{"int", 42, 42.0, true},
		{"int32", int32(42), 42.0, true},
		{"int64", int64(42), 42.0, true},
		{"float32", float32(3.14), 3.140000104904175, true}, // float32 precision
		{"float64", 3.14, 3.14, true},
		{"string numeric", "3.14", 3.14, true},
		{"string non-numeric", "abc", 0, false},
		{"bool", true, 0, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, ok := toFloat64(tc.value)
			assert.Equal(t, tc.ok, ok)
			if ok {
				assert.InDelta(t, tc.expected, result, 0.0001)
			}
		})
	}
}

func TestInvalidOperator(t *testing.T) {
	condition := Condition{
		Attribute: "user_id",
		Operator:  ConditionOperator("invalid_operator"),
		Values:    []string{"value"},
	}

	ctx := NewEvaluationContext().WithUser("user-123")
	result := MatchCondition(condition, ctx)
	assert.False(t, result, "Invalid operator should return false")
}
