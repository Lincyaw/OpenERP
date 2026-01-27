package featureflag

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEvaluationContext(t *testing.T) {
	ctx := NewEvaluationContext()

	assert.NotNil(t, ctx)
	assert.NotZero(t, ctx.Timestamp)
	assert.NotNil(t, ctx.UserAttributes)
	assert.Empty(t, ctx.TenantID)
	assert.Empty(t, ctx.UserID)
}

func TestEvaluationContext_WithTenant(t *testing.T) {
	ctx := NewEvaluationContext()
	ctx = ctx.WithTenant("tenant-123")

	assert.Equal(t, "tenant-123", ctx.TenantID)
	assert.True(t, ctx.HasTenant())
}

func TestEvaluationContext_WithUser(t *testing.T) {
	ctx := NewEvaluationContext()
	ctx = ctx.WithUser("user-456")

	assert.Equal(t, "user-456", ctx.UserID)
	assert.True(t, ctx.HasUser())
}

func TestEvaluationContext_WithUserRole(t *testing.T) {
	ctx := NewEvaluationContext().WithUserRole("admin")
	assert.Equal(t, "admin", ctx.UserRole)
}

func TestEvaluationContext_WithUserPlan(t *testing.T) {
	ctx := NewEvaluationContext().WithUserPlan("enterprise")
	assert.Equal(t, "enterprise", ctx.UserPlan)
}

func TestEvaluationContext_WithAttribute(t *testing.T) {
	ctx := NewEvaluationContext().
		WithAttribute("country", "US").
		WithAttribute("beta_tester", true)

	val, ok := ctx.GetAttribute("country")
	assert.True(t, ok)
	assert.Equal(t, "US", val)

	val, ok = ctx.GetAttribute("beta_tester")
	assert.True(t, ok)
	assert.Equal(t, true, val)

	val, ok = ctx.GetAttribute("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestEvaluationContext_GetAttributeString(t *testing.T) {
	ctx := NewEvaluationContext().
		WithAttribute("country", "US").
		WithAttribute("numeric", 123)

	assert.Equal(t, "US", ctx.GetAttributeString("country"))
	assert.Equal(t, "", ctx.GetAttributeString("numeric")) // Not a string
	assert.Equal(t, "", ctx.GetAttributeString("nonexistent"))
}

func TestEvaluationContext_WithRequestID(t *testing.T) {
	ctx := NewEvaluationContext().WithRequestID("req-789")
	assert.Equal(t, "req-789", ctx.RequestID)
}

func TestEvaluationContext_WithEnvironment(t *testing.T) {
	ctx := NewEvaluationContext().WithEnvironment("production")
	assert.Equal(t, "production", ctx.Environment)
}

func TestEvaluationContext_WithTimestamp(t *testing.T) {
	customTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	ctx := NewEvaluationContext().WithTimestamp(customTime)
	assert.Equal(t, customTime, ctx.Timestamp)
}

func TestEvaluationContext_Immutability(t *testing.T) {
	original := NewEvaluationContext().
		WithUser("user-1").
		WithAttribute("key", "value1")

	modified := original.WithUser("user-2").
		WithAttribute("key", "value2")

	// Original should not be modified
	assert.Equal(t, "user-1", original.UserID)
	val, _ := original.GetAttribute("key")
	assert.Equal(t, "value1", val)

	// Modified should have new values
	assert.Equal(t, "user-2", modified.UserID)
	val2, _ := modified.GetAttribute("key")
	assert.Equal(t, "value2", val2)
}

func TestEvaluationContext_Chaining(t *testing.T) {
	ctx := NewEvaluationContext().
		WithTenant("tenant-123").
		WithUser("user-456").
		WithUserRole("admin").
		WithUserPlan("enterprise").
		WithEnvironment("production").
		WithRequestID("req-789").
		WithAttribute("country", "US").
		WithAttribute("beta", true)

	assert.Equal(t, "tenant-123", ctx.TenantID)
	assert.Equal(t, "user-456", ctx.UserID)
	assert.Equal(t, "admin", ctx.UserRole)
	assert.Equal(t, "enterprise", ctx.UserPlan)
	assert.Equal(t, "production", ctx.Environment)
	assert.Equal(t, "req-789", ctx.RequestID)
	assert.True(t, ctx.HasTenant())
	assert.True(t, ctx.HasUser())
}

func TestNewEvaluationResult(t *testing.T) {
	value := NewBooleanFlagValue(true)
	result := NewEvaluationResult("my-flag", value, EvaluationReasonDefault, 5)

	assert.Equal(t, "my-flag", result.Key)
	assert.True(t, result.Enabled)
	assert.Equal(t, EvaluationReasonDefault, result.Reason)
	assert.Equal(t, 5, result.FlagVersion)
	assert.NotZero(t, result.EvaluatedAt)
}

func TestNewEvaluationResult_WithVariant(t *testing.T) {
	value := NewVariantFlagValue("variant-A")
	result := NewEvaluationResult("my-flag", value, EvaluationReasonRuleMatch, 3)

	assert.Equal(t, "my-flag", result.Key)
	assert.True(t, result.Enabled)
	assert.Equal(t, "variant-A", result.Variant)
	assert.True(t, result.HasVariant())
	assert.Equal(t, EvaluationReasonRuleMatch, result.Reason)
}

func TestNewDisabledResult(t *testing.T) {
	defaultValue := NewBooleanFlagValue(true)
	result := NewDisabledResult("my-flag", defaultValue, 2)

	assert.Equal(t, "my-flag", result.Key)
	assert.False(t, result.Enabled)
	assert.True(t, result.IsDisabled())
	assert.Equal(t, EvaluationReasonDisabled, result.Reason)
	assert.Equal(t, 2, result.FlagVersion)
}

func TestNewFlagNotFoundResult(t *testing.T) {
	result := NewFlagNotFoundResult("missing-flag")

	assert.Equal(t, "missing-flag", result.Key)
	assert.False(t, result.Enabled)
	assert.Equal(t, EvaluationReasonFlagNotFound, result.Reason)
	assert.Equal(t, 0, result.FlagVersion)
}

func TestNewErrorResult(t *testing.T) {
	err := assert.AnError
	result := NewErrorResult("error-flag", err)

	assert.Equal(t, "error-flag", result.Key)
	assert.False(t, result.Enabled)
	assert.Equal(t, EvaluationReasonError, result.Reason)
	assert.True(t, result.HasError())
	assert.Equal(t, err, result.GetError())
}

func TestEvaluationResult_WithRuleID(t *testing.T) {
	value := NewBooleanFlagValue(true)
	result := NewEvaluationResult("my-flag", value, EvaluationReasonRuleMatch, 1)
	resultWithRule := result.WithRuleID("rule-123")

	assert.Empty(t, result.RuleID) // Original unchanged
	assert.Equal(t, "rule-123", resultWithRule.RuleID)
}

func TestEvaluationResult_IsFromOverride(t *testing.T) {
	tests := []struct {
		reason   EvaluationReason
		expected bool
	}{
		{EvaluationReasonOverrideUser, true},
		{EvaluationReasonOverrideTenant, true},
		{EvaluationReasonRuleMatch, false},
		{EvaluationReasonDefault, false},
	}

	for _, tc := range tests {
		value := NewBooleanFlagValue(true)
		result := NewEvaluationResult("test", value, tc.reason, 1)
		assert.Equal(t, tc.expected, result.IsFromOverride(), "reason: %s", tc.reason)
	}
}

func TestEvaluationResult_IsFromRule(t *testing.T) {
	tests := []struct {
		reason   EvaluationReason
		expected bool
	}{
		{EvaluationReasonRuleMatch, true},
		{EvaluationReasonPercentage, true},
		{EvaluationReasonOverrideUser, false},
		{EvaluationReasonDefault, false},
	}

	for _, tc := range tests {
		value := NewBooleanFlagValue(true)
		result := NewEvaluationResult("test", value, tc.reason, 1)
		assert.Equal(t, tc.expected, result.IsFromRule(), "reason: %s", tc.reason)
	}
}

func TestEvaluationResult_IsDefault(t *testing.T) {
	tests := []struct {
		reason   EvaluationReason
		expected bool
	}{
		{EvaluationReasonDefault, true},
		{EvaluationReasonRuleMatch, false},
		{EvaluationReasonOverrideUser, false},
	}

	for _, tc := range tests {
		value := NewBooleanFlagValue(true)
		result := NewEvaluationResult("test", value, tc.reason, 1)
		assert.Equal(t, tc.expected, result.IsDefault(), "reason: %s", tc.reason)
	}
}

func TestEvaluationReason_String(t *testing.T) {
	tests := []struct {
		reason   EvaluationReason
		expected string
	}{
		{EvaluationReasonOverrideUser, "override_user"},
		{EvaluationReasonOverrideTenant, "override_tenant"},
		{EvaluationReasonRuleMatch, "rule_match"},
		{EvaluationReasonPercentage, "percentage"},
		{EvaluationReasonDefault, "default"},
		{EvaluationReasonDisabled, "disabled"},
		{EvaluationReasonFlagNotFound, "flag_not_found"},
		{EvaluationReasonError, "error"},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.expected, tc.reason.String())
	}
}

func TestCopyMap(t *testing.T) {
	t.Run("nil map", func(t *testing.T) {
		result := copyMap(nil)
		require.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("empty map", func(t *testing.T) {
		original := make(map[string]any)
		result := copyMap(original)
		require.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("map with values", func(t *testing.T) {
		original := map[string]any{
			"key1": "value1",
			"key2": 123,
		}
		result := copyMap(original)

		// Verify copy has same values
		assert.Equal(t, "value1", result["key1"])
		assert.Equal(t, 123, result["key2"])

		// Verify modifying copy doesn't affect original
		result["key1"] = "modified"
		assert.Equal(t, "value1", original["key1"])
	})
}
