package featureflag

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBooleanFlagValue(t *testing.T) {
	t.Run("enabled flag", func(t *testing.T) {
		fv := NewBooleanFlagValue(true)
		assert.True(t, fv.Enabled)
		assert.Empty(t, fv.Variant)
		assert.NotNil(t, fv.Metadata)
		assert.Len(t, fv.Metadata, 0)
	})

	t.Run("disabled flag", func(t *testing.T) {
		fv := NewBooleanFlagValue(false)
		assert.False(t, fv.Enabled)
		assert.Empty(t, fv.Variant)
	})
}

func TestNewVariantFlagValue(t *testing.T) {
	fv := NewVariantFlagValue("control")
	assert.True(t, fv.Enabled)
	assert.Equal(t, "control", fv.Variant)
	assert.NotNil(t, fv.Metadata)
}

func TestNewFlagValueWithMetadata(t *testing.T) {
	metadata := map[string]any{"key": "value"}
	fv := NewFlagValueWithMetadata(true, "variant_a", metadata)

	assert.True(t, fv.Enabled)
	assert.Equal(t, "variant_a", fv.Variant)
	assert.Equal(t, "value", fv.Metadata["key"])
}

func TestNewFlagValueWithMetadata_NilMetadata(t *testing.T) {
	fv := NewFlagValueWithMetadata(true, "variant_a", nil)
	assert.NotNil(t, fv.Metadata)
	assert.Len(t, fv.Metadata, 0)
}

func TestFlagValue_IsEnabled(t *testing.T) {
	fv := NewBooleanFlagValue(true)
	assert.True(t, fv.IsEnabled())

	fv = NewBooleanFlagValue(false)
	assert.False(t, fv.IsEnabled())
}

func TestFlagValue_GetVariant(t *testing.T) {
	fv := NewVariantFlagValue("test_variant")
	assert.Equal(t, "test_variant", fv.GetVariant())
}

func TestFlagValue_HasVariant(t *testing.T) {
	fv := NewVariantFlagValue("test")
	assert.True(t, fv.HasVariant())

	fv = NewBooleanFlagValue(true)
	assert.False(t, fv.HasVariant())
}

func TestFlagValue_GetMetadata(t *testing.T) {
	metadata := map[string]any{"key": "value"}
	fv := NewFlagValueWithMetadata(true, "", metadata)

	result := fv.GetMetadata()
	assert.Equal(t, "value", result["key"])

	// Ensure it returns a copy
	result["key"] = "modified"
	assert.Equal(t, "value", fv.Metadata["key"])
}

func TestFlagValue_GetMetadata_NilMetadata(t *testing.T) {
	fv := FlagValue{}
	result := fv.GetMetadata()
	assert.NotNil(t, result)
	assert.Len(t, result, 0)
}

func TestFlagValue_GetMetadataValue(t *testing.T) {
	metadata := map[string]any{"key": "value"}
	fv := NewFlagValueWithMetadata(true, "", metadata)

	val, ok := fv.GetMetadataValue("key")
	assert.True(t, ok)
	assert.Equal(t, "value", val)

	val, ok = fv.GetMetadataValue("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestFlagValue_GetMetadataValue_NilMetadata(t *testing.T) {
	fv := FlagValue{}
	val, ok := fv.GetMetadataValue("key")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestFlagValue_WithMetadata(t *testing.T) {
	fv := NewBooleanFlagValue(true)
	newFv := fv.WithMetadata("key", "value")

	assert.Equal(t, "value", newFv.Metadata["key"])
	assert.Empty(t, fv.Metadata) // Original unchanged
}

func TestFlagValue_Equals(t *testing.T) {
	fv1 := NewFlagValueWithMetadata(true, "variant", map[string]any{"k": "v"})
	fv2 := NewFlagValueWithMetadata(true, "variant", map[string]any{"k": "v"})
	assert.True(t, fv1.Equals(fv2))

	// Different enabled
	fv3 := NewFlagValueWithMetadata(false, "variant", map[string]any{"k": "v"})
	assert.False(t, fv1.Equals(fv3))

	// Different variant
	fv4 := NewFlagValueWithMetadata(true, "other", map[string]any{"k": "v"})
	assert.False(t, fv1.Equals(fv4))

	// Different metadata
	fv5 := NewFlagValueWithMetadata(true, "variant", map[string]any{"k": "other"})
	assert.False(t, fv1.Equals(fv5))

	// Different metadata length
	fv6 := NewFlagValueWithMetadata(true, "variant", map[string]any{"k": "v", "k2": "v2"})
	assert.False(t, fv1.Equals(fv6))
}

func TestNewCondition(t *testing.T) {
	t.Run("valid condition", func(t *testing.T) {
		cond, err := NewCondition("user_role", ConditionOperatorEquals, []string{"admin"})
		assert.NoError(t, err)
		assert.Equal(t, "user_role", cond.Attribute)
		assert.Equal(t, ConditionOperatorEquals, cond.Operator)
		assert.Equal(t, []string{"admin"}, cond.Values)
	})

	t.Run("empty attribute", func(t *testing.T) {
		_, err := NewCondition("", ConditionOperatorEquals, []string{"admin"})
		assert.Error(t, err)
	})

	t.Run("invalid operator", func(t *testing.T) {
		_, err := NewCondition("user_role", ConditionOperator("invalid"), []string{"admin"})
		assert.Error(t, err)
	})

	t.Run("empty values", func(t *testing.T) {
		_, err := NewCondition("user_role", ConditionOperatorEquals, []string{})
		assert.Error(t, err)
	})
}

func TestCondition_Getters(t *testing.T) {
	cond, _ := NewCondition("user_role", ConditionOperatorIn, []string{"admin", "manager"})

	assert.Equal(t, "user_role", cond.GetAttribute())
	assert.Equal(t, ConditionOperatorIn, cond.GetOperator())
	values := cond.GetValues()
	assert.Equal(t, []string{"admin", "manager"}, values)

	// Ensure GetValues returns a copy
	values[0] = "modified"
	assert.Equal(t, "admin", cond.Values[0])
}

func TestCondition_GetValues_NilValues(t *testing.T) {
	cond := Condition{}
	values := cond.GetValues()
	assert.NotNil(t, values)
	assert.Len(t, values, 0)
}

func TestCondition_Validate(t *testing.T) {
	t.Run("valid condition", func(t *testing.T) {
		cond, _ := NewCondition("attr", ConditionOperatorEquals, []string{"val"})
		assert.NoError(t, cond.Validate())
	})

	t.Run("empty attribute", func(t *testing.T) {
		cond := Condition{Operator: ConditionOperatorEquals, Values: []string{"val"}}
		assert.Error(t, cond.Validate())
	})

	t.Run("invalid operator", func(t *testing.T) {
		cond := Condition{Attribute: "attr", Operator: ConditionOperator("invalid"), Values: []string{"val"}}
		assert.Error(t, cond.Validate())
	})

	t.Run("empty values", func(t *testing.T) {
		cond := Condition{Attribute: "attr", Operator: ConditionOperatorEquals, Values: []string{}}
		assert.Error(t, cond.Validate())
	})
}

func TestNewTargetingRule(t *testing.T) {
	t.Run("valid rule", func(t *testing.T) {
		conditions := []Condition{
			{Attribute: "role", Operator: ConditionOperatorEquals, Values: []string{"admin"}},
		}
		rule, err := NewTargetingRule("rule1", 1, conditions, NewBooleanFlagValue(true))
		assert.NoError(t, err)
		assert.Equal(t, "rule1", rule.RuleID)
		assert.Equal(t, 1, rule.Priority)
		assert.Len(t, rule.Conditions, 1)
		assert.Equal(t, 100, rule.Percentage) // Default 100%
	})

	t.Run("empty rule ID", func(t *testing.T) {
		_, err := NewTargetingRule("", 1, nil, NewBooleanFlagValue(true))
		assert.Error(t, err)
	})

	t.Run("negative priority", func(t *testing.T) {
		_, err := NewTargetingRule("rule1", -1, nil, NewBooleanFlagValue(true))
		assert.Error(t, err)
	})
}

func TestNewTargetingRuleWithPercentage(t *testing.T) {
	t.Run("valid percentage", func(t *testing.T) {
		rule, err := NewTargetingRuleWithPercentage("rule1", 1, nil, NewBooleanFlagValue(true), 50)
		assert.NoError(t, err)
		assert.Equal(t, 50, rule.Percentage)
	})

	t.Run("negative percentage", func(t *testing.T) {
		_, err := NewTargetingRuleWithPercentage("rule1", 1, nil, NewBooleanFlagValue(true), -1)
		assert.Error(t, err)
	})

	t.Run("percentage over 100", func(t *testing.T) {
		_, err := NewTargetingRuleWithPercentage("rule1", 1, nil, NewBooleanFlagValue(true), 101)
		assert.Error(t, err)
	})
}

func TestTargetingRule_Getters(t *testing.T) {
	conditions := []Condition{
		{Attribute: "role", Operator: ConditionOperatorEquals, Values: []string{"admin"}},
	}
	rule, _ := NewTargetingRuleWithPercentage("rule1", 1, conditions, NewBooleanFlagValue(true), 75)

	assert.Equal(t, "rule1", rule.GetRuleID())
	assert.Equal(t, 1, rule.GetPriority())
	assert.Equal(t, 75, rule.GetPercentage())
	assert.True(t, rule.GetValue().Enabled)
	assert.True(t, rule.HasConditions())

	// Ensure GetConditions returns a copy
	conds := rule.GetConditions()
	assert.Len(t, conds, 1)
	conds[0].Attribute = "modified"
	assert.Equal(t, "role", rule.Conditions[0].Attribute)
}

func TestTargetingRule_GetConditions_NilConditions(t *testing.T) {
	rule := TargetingRule{RuleID: "rule1", Priority: 1, Percentage: 100}
	conds := rule.GetConditions()
	assert.NotNil(t, conds)
	assert.Len(t, conds, 0)
}

func TestTargetingRule_HasConditions(t *testing.T) {
	rule := TargetingRule{RuleID: "rule1"}
	assert.False(t, rule.HasConditions())

	rule.Conditions = []Condition{{Attribute: "test"}}
	assert.True(t, rule.HasConditions())
}

func TestTargetingRule_Validate(t *testing.T) {
	t.Run("valid rule", func(t *testing.T) {
		rule, _ := NewTargetingRule("rule1", 1, nil, NewBooleanFlagValue(true))
		assert.NoError(t, rule.Validate())
	})

	t.Run("empty rule ID", func(t *testing.T) {
		rule := TargetingRule{Priority: 1, Percentage: 100}
		assert.Error(t, rule.Validate())
	})

	t.Run("negative priority", func(t *testing.T) {
		rule := TargetingRule{RuleID: "rule1", Priority: -1, Percentage: 100}
		assert.Error(t, rule.Validate())
	})

	t.Run("invalid percentage", func(t *testing.T) {
		rule := TargetingRule{RuleID: "rule1", Priority: 1, Percentage: 101}
		assert.Error(t, rule.Validate())
	})

	t.Run("invalid condition", func(t *testing.T) {
		rule := TargetingRule{
			RuleID:     "rule1",
			Priority:   1,
			Percentage: 100,
			Conditions: []Condition{{Attribute: "", Operator: ConditionOperatorEquals, Values: []string{"val"}}},
		}
		assert.Error(t, rule.Validate())
	})
}

func TestTargetingRule_AddCondition(t *testing.T) {
	rule, _ := NewTargetingRule("rule1", 1, nil, NewBooleanFlagValue(true))

	cond, _ := NewCondition("role", ConditionOperatorEquals, []string{"admin"})
	newRule, err := rule.AddCondition(cond)
	assert.NoError(t, err)
	assert.Len(t, newRule.Conditions, 1)
	assert.Len(t, rule.Conditions, 0) // Original unchanged
}

func TestTargetingRule_AddCondition_InvalidCondition(t *testing.T) {
	rule, _ := NewTargetingRule("rule1", 1, nil, NewBooleanFlagValue(true))
	cond := Condition{Attribute: "", Operator: ConditionOperatorEquals, Values: []string{"val"}}
	_, err := rule.AddCondition(cond)
	assert.Error(t, err)
}

func TestTargetingRule_WithPriority(t *testing.T) {
	rule, _ := NewTargetingRule("rule1", 1, nil, NewBooleanFlagValue(true))

	newRule, err := rule.WithPriority(5)
	assert.NoError(t, err)
	assert.Equal(t, 5, newRule.Priority)
	assert.Equal(t, 1, rule.Priority) // Original unchanged
}

func TestTargetingRule_WithPriority_Negative(t *testing.T) {
	rule, _ := NewTargetingRule("rule1", 1, nil, NewBooleanFlagValue(true))
	_, err := rule.WithPriority(-1)
	assert.Error(t, err)
}

func TestTargetingRule_WithPercentage(t *testing.T) {
	rule, _ := NewTargetingRule("rule1", 1, nil, NewBooleanFlagValue(true))

	newRule, err := rule.WithPercentage(50)
	assert.NoError(t, err)
	assert.Equal(t, 50, newRule.Percentage)
	assert.Equal(t, 100, rule.Percentage) // Original unchanged
}

func TestTargetingRule_WithPercentage_Invalid(t *testing.T) {
	rule, _ := NewTargetingRule("rule1", 1, nil, NewBooleanFlagValue(true))

	_, err := rule.WithPercentage(-1)
	assert.Error(t, err)

	_, err = rule.WithPercentage(101)
	assert.Error(t, err)
}
