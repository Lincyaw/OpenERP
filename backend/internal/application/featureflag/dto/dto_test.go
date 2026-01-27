package dto

import (
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/featureflag"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestFlagValueDTO_ToDomain(t *testing.T) {
	dto := FlagValueDTO{
		Enabled:  true,
		Variant:  "control",
		Metadata: map[string]any{"key": "value"},
	}

	domain := dto.ToDomain()

	assert.True(t, domain.Enabled)
	assert.Equal(t, "control", domain.Variant)
	assert.Equal(t, "value", domain.Metadata["key"])
}

func TestToFlagValueDTO(t *testing.T) {
	domain := featureflag.NewFlagValueWithMetadata(true, "variant-a", map[string]any{"foo": "bar"})

	dto := ToFlagValueDTO(domain)

	assert.True(t, dto.Enabled)
	assert.Equal(t, "variant-a", dto.Variant)
	assert.Equal(t, "bar", dto.Metadata["foo"])
}

func TestConditionDTO_ToDomain(t *testing.T) {
	dto := ConditionDTO{
		Attribute: "user_role",
		Operator:  "equals",
		Values:    []string{"admin"},
	}

	domain, err := dto.ToDomain()

	assert.NoError(t, err)
	assert.Equal(t, "user_role", domain.GetAttribute())
	assert.Equal(t, featureflag.ConditionOperatorEquals, domain.GetOperator())
	assert.Equal(t, []string{"admin"}, domain.GetValues())
}

func TestConditionDTO_ToDomain_InvalidOperator(t *testing.T) {
	dto := ConditionDTO{
		Attribute: "user_role",
		Operator:  "invalid",
		Values:    []string{"admin"},
	}

	_, err := dto.ToDomain()

	assert.Error(t, err)
}

func TestToConditionDTO(t *testing.T) {
	domain, _ := featureflag.NewCondition("email", featureflag.ConditionOperatorContains, []string{"@test.com"})

	dto := ToConditionDTO(domain)

	assert.Equal(t, "email", dto.Attribute)
	assert.Equal(t, "contains", dto.Operator)
	assert.Equal(t, []string{"@test.com"}, dto.Values)
}

func TestTargetingRuleDTO_ToDomain(t *testing.T) {
	dto := TargetingRuleDTO{
		RuleID:   "rule-1",
		Priority: 1,
		Conditions: []ConditionDTO{
			{
				Attribute: "plan",
				Operator:  "in",
				Values:    []string{"pro", "enterprise"},
			},
		},
		Value: FlagValueDTO{
			Enabled: true,
		},
		Percentage: 50,
	}

	domain, err := dto.ToDomain()

	assert.NoError(t, err)
	assert.Equal(t, "rule-1", domain.GetRuleID())
	assert.Equal(t, 1, domain.GetPriority())
	assert.Equal(t, 50, domain.GetPercentage())
	assert.True(t, domain.GetValue().Enabled)
	assert.Equal(t, 1, len(domain.GetConditions()))
}

func TestToTargetingRuleDTO(t *testing.T) {
	condition, _ := featureflag.NewCondition("role", featureflag.ConditionOperatorEquals, []string{"admin"})
	rule, _ := featureflag.NewTargetingRuleWithPercentage(
		"rule-1",
		1,
		[]featureflag.Condition{condition},
		featureflag.NewBooleanFlagValue(true),
		100,
	)

	dto := ToTargetingRuleDTO(rule)

	assert.Equal(t, "rule-1", dto.RuleID)
	assert.Equal(t, 1, dto.Priority)
	assert.Equal(t, 100, dto.Percentage)
	assert.Equal(t, 1, len(dto.Conditions))
}

func TestToFlagResponse(t *testing.T) {
	userID := uuid.New()
	flag, _ := featureflag.NewBooleanFlag("test-flag", "Test Flag", false, &userID)
	_ = flag.SetTags([]string{"test", "feature"}, &userID)

	response := ToFlagResponse(flag)

	assert.Equal(t, flag.ID, response.ID)
	assert.Equal(t, "test-flag", response.Key)
	assert.Equal(t, "Test Flag", response.Name)
	assert.Equal(t, "boolean", response.Type)
	assert.Equal(t, "disabled", response.Status)
	assert.Equal(t, []string{"test", "feature"}, response.Tags)
}

func TestToFlagListResponse(t *testing.T) {
	flag1, _ := featureflag.NewBooleanFlag("flag-1", "Flag 1", false, nil)
	flag2, _ := featureflag.NewBooleanFlag("flag-2", "Flag 2", true, nil)
	flags := []featureflag.FeatureFlag{*flag1, *flag2}

	response := ToFlagListResponse(flags, 50, 1, 20)

	assert.Equal(t, 2, len(response.Flags))
	assert.Equal(t, int64(50), response.Total)
	assert.Equal(t, 1, response.Page)
	assert.Equal(t, 20, response.PageSize)
	assert.Equal(t, 3, response.TotalPages) // 50/20 = 2.5, rounded up to 3
}

func TestEvaluationContextDTO_ToDomain(t *testing.T) {
	dto := EvaluationContextDTO{
		TenantID:       "tenant-123",
		UserID:         "user-456",
		UserRole:       "admin",
		UserPlan:       "enterprise",
		UserAttributes: map[string]any{"country": "US"},
		RequestID:      "req-789",
		Environment:    "production",
	}

	domain := dto.ToDomain()

	assert.Equal(t, "tenant-123", domain.TenantID)
	assert.Equal(t, "user-456", domain.UserID)
	assert.Equal(t, "admin", domain.UserRole)
	assert.Equal(t, "enterprise", domain.UserPlan)
	assert.Equal(t, "req-789", domain.RequestID)
	assert.Equal(t, "production", domain.Environment)

	country, ok := domain.GetAttribute("country")
	assert.True(t, ok)
	assert.Equal(t, "US", country)
}

func TestToEvaluateFlagResponse(t *testing.T) {
	result := featureflag.NewEvaluationResult(
		"test-flag",
		featureflag.NewBooleanFlagValue(true),
		featureflag.EvaluationReasonRuleMatch,
		5,
	).WithRuleID("rule-1")

	response := ToEvaluateFlagResponse(result)

	assert.Equal(t, "test-flag", response.Key)
	assert.True(t, response.Enabled)
	assert.Equal(t, "rule_match", response.Reason)
	assert.Equal(t, "rule-1", response.RuleID)
	assert.Equal(t, 5, response.FlagVersion)
}

func TestToBatchEvaluateResponse(t *testing.T) {
	results := map[string]featureflag.EvaluationResult{
		"flag-1": featureflag.NewEvaluationResult("flag-1", featureflag.NewBooleanFlagValue(true), featureflag.EvaluationReasonDefault, 1),
		"flag-2": featureflag.NewEvaluationResult("flag-2", featureflag.NewBooleanFlagValue(false), featureflag.EvaluationReasonDisabled, 2),
	}

	response := ToBatchEvaluateResponse(results)

	assert.Equal(t, 2, len(response.Results))
	assert.True(t, response.Results["flag-1"].Enabled)
	assert.False(t, response.Results["flag-2"].Enabled)
}

func TestToGetClientConfigResponse(t *testing.T) {
	results := map[string]featureflag.EvaluationResult{
		"flag-1": featureflag.NewEvaluationResult("flag-1", featureflag.NewBooleanFlagValue(true), featureflag.EvaluationReasonDefault, 1),
		"flag-2": featureflag.NewEvaluationResult("flag-2", featureflag.NewVariantFlagValue("control"), featureflag.EvaluationReasonDefault, 2),
	}

	response := ToGetClientConfigResponse(results)

	assert.Equal(t, 2, len(response.Flags))
	assert.True(t, response.Flags["flag-1"].Enabled)
	assert.Equal(t, "control", response.Flags["flag-2"].Variant)
}

func TestToOverrideResponse(t *testing.T) {
	targetID := uuid.New()
	userID := uuid.New()
	expiresAt := time.Now().Add(24 * time.Hour)

	override, _ := featureflag.NewFlagOverride(
		"test-flag",
		featureflag.OverrideTargetTypeUser,
		targetID,
		featureflag.NewBooleanFlagValue(true),
		"Testing",
		&expiresAt,
		&userID,
	)

	response := ToOverrideResponse(override)

	assert.Equal(t, override.ID, response.ID)
	assert.Equal(t, "test-flag", response.FlagKey)
	assert.Equal(t, "user", response.TargetType)
	assert.Equal(t, targetID, response.TargetID)
	assert.True(t, response.Value.Enabled)
	assert.Equal(t, "Testing", response.Reason)
	assert.False(t, response.IsExpired)
}

func TestToOverrideListResponse(t *testing.T) {
	override1, _ := featureflag.NewFlagOverride(
		"test-flag",
		featureflag.OverrideTargetTypeUser,
		uuid.New(),
		featureflag.NewBooleanFlagValue(true),
		"reason1",
		nil,
		nil,
	)
	override2, _ := featureflag.NewFlagOverride(
		"test-flag",
		featureflag.OverrideTargetTypeTenant,
		uuid.New(),
		featureflag.NewBooleanFlagValue(false),
		"reason2",
		nil,
		nil,
	)
	overrides := []featureflag.FlagOverride{*override1, *override2}

	response := ToOverrideListResponse(overrides, 25, 2, 10)

	assert.Equal(t, 2, len(response.Overrides))
	assert.Equal(t, int64(25), response.Total)
	assert.Equal(t, 2, response.Page)
	assert.Equal(t, 10, response.PageSize)
	assert.Equal(t, 3, response.TotalPages) // 25/10 = 2.5, rounded up to 3
}

func TestEvaluationContextDTO_ToDomain_EmptyFields(t *testing.T) {
	dto := EvaluationContextDTO{}

	domain := dto.ToDomain()

	assert.Empty(t, domain.TenantID)
	assert.Empty(t, domain.UserID)
	assert.Empty(t, domain.UserRole)
	assert.Empty(t, domain.UserPlan)
	assert.Empty(t, domain.RequestID)
	assert.Empty(t, domain.Environment)
}
