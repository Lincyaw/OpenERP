package models

import (
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/featureflag"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeatureFlagModel_TableName(t *testing.T) {
	model := FeatureFlagModel{}
	assert.Equal(t, "feature_flags", model.TableName())
}

func TestFeatureFlagModel_ToDomain(t *testing.T) {
	userID := uuid.New()
	now := time.Now()

	model := &FeatureFlagModel{
		AggregateModel: AggregateModel{
			BaseModel: BaseModel{
				ID:        uuid.New(),
				CreatedAt: now,
				UpdatedAt: now,
			},
			Version: 2,
		},
		Key:              "test_flag",
		Name:             "Test Flag",
		Description:      "A test feature flag",
		Type:             featureflag.FlagTypeBoolean,
		Status:           featureflag.FlagStatusEnabled,
		DefaultValueJSON: `{"enabled":true}`,
		RulesJSON:        `[]`,
		TagsJSON:         `["test","beta"]`,
		CreatedBy:        &userID,
		UpdatedBy:        &userID,
	}

	domain := model.ToDomain()

	assert.Equal(t, model.ID, domain.ID)
	assert.Equal(t, model.CreatedAt, domain.CreatedAt)
	assert.Equal(t, model.UpdatedAt, domain.UpdatedAt)
	assert.Equal(t, model.Version, domain.Version)
	assert.Equal(t, model.Key, domain.Key)
	assert.Equal(t, model.Name, domain.Name)
	assert.Equal(t, model.Description, domain.Description)
	assert.Equal(t, model.Type, domain.Type)
	assert.Equal(t, model.Status, domain.Status)
	assert.True(t, domain.DefaultValue.Enabled)
	assert.Contains(t, domain.Tags, "test")
	assert.Contains(t, domain.Tags, "beta")
	assert.Equal(t, &userID, domain.CreatedBy)
	assert.Equal(t, &userID, domain.UpdatedBy)
}

func TestFeatureFlagModel_FromDomain(t *testing.T) {
	userID := uuid.New()

	flag, err := featureflag.NewBooleanFlag("test_flag", "Test Flag", true, &userID)
	require.NoError(t, err)
	_ = flag.SetTags([]string{"alpha", "beta"}, &userID)

	model := &FeatureFlagModel{}
	model.FromDomain(flag)

	assert.Equal(t, flag.ID, model.ID)
	assert.Equal(t, flag.Key, model.Key)
	assert.Equal(t, flag.Name, model.Name)
	assert.Equal(t, flag.Type, model.Type)
	assert.Equal(t, flag.Status, model.Status)
	assert.Contains(t, model.DefaultValueJSON, "enabled")
	assert.Contains(t, model.TagsJSON, "alpha")
	assert.Contains(t, model.TagsJSON, "beta")
}

func TestFeatureFlagModelFromDomain(t *testing.T) {
	userID := uuid.New()

	flag, err := featureflag.NewBooleanFlag("test_flag", "Test Flag", false, &userID)
	require.NoError(t, err)

	model := FeatureFlagModelFromDomain(flag)

	assert.NotNil(t, model)
	assert.Equal(t, flag.ID, model.ID)
	assert.Equal(t, flag.Key, model.Key)
}

func TestFeatureFlagModel_RulesJSONRoundTrip(t *testing.T) {
	userID := uuid.New()

	flag, err := featureflag.NewBooleanFlag("test_flag", "Test Flag", true, &userID)
	require.NoError(t, err)

	condition, err := featureflag.NewCondition("country", featureflag.ConditionOperatorEquals, []string{"US"})
	require.NoError(t, err)

	rule, err := featureflag.NewTargetingRule("rule-1", 1, []featureflag.Condition{condition}, featureflag.NewBooleanFlagValue(true))
	require.NoError(t, err)

	err = flag.AddRule(rule, &userID)
	require.NoError(t, err)

	// Convert to model
	model := FeatureFlagModelFromDomain(flag)

	// Convert back to domain
	domainFlag := model.ToDomain()

	assert.Len(t, domainFlag.Rules, 1)
	assert.Equal(t, "rule-1", domainFlag.Rules[0].RuleID)
	assert.Len(t, domainFlag.Rules[0].Conditions, 1)
	assert.Equal(t, "country", domainFlag.Rules[0].Conditions[0].Attribute)
}

func TestFlagOverrideModel_TableName(t *testing.T) {
	model := FlagOverrideModel{}
	assert.Equal(t, "flag_overrides", model.TableName())
}

func TestFlagOverrideModel_ToDomain(t *testing.T) {
	userID := uuid.New()
	targetID := uuid.New()
	expiresAt := time.Now().Add(24 * time.Hour)
	now := time.Now()

	model := &FlagOverrideModel{
		BaseModel: BaseModel{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		FlagKey:    "test_flag",
		TargetType: featureflag.OverrideTargetTypeUser,
		TargetID:   targetID,
		ValueJSON:  `{"enabled":true,"variant":"beta"}`,
		Reason:     "Testing",
		ExpiresAt:  &expiresAt,
		CreatedBy:  &userID,
	}

	domain := model.ToDomain()

	assert.Equal(t, model.ID, domain.ID)
	assert.Equal(t, model.FlagKey, domain.FlagKey)
	assert.Equal(t, model.TargetType, domain.TargetType)
	assert.Equal(t, model.TargetID, domain.TargetID)
	assert.True(t, domain.Value.Enabled)
	assert.Equal(t, "beta", domain.Value.Variant)
	assert.Equal(t, model.Reason, domain.Reason)
	assert.Equal(t, model.ExpiresAt, domain.ExpiresAt)
	assert.Equal(t, model.CreatedBy, domain.CreatedBy)
}

func TestFlagOverrideModel_FromDomain(t *testing.T) {
	userID := uuid.New()
	targetID := uuid.New()
	expiresAt := time.Now().Add(24 * time.Hour)

	override, err := featureflag.NewFlagOverride(
		"test_flag",
		featureflag.OverrideTargetTypeTenant,
		targetID,
		featureflag.NewVariantFlagValue("control"),
		"Experiment",
		&expiresAt,
		&userID,
	)
	require.NoError(t, err)

	model := &FlagOverrideModel{}
	model.FromDomain(override)

	assert.Equal(t, override.ID, model.ID)
	assert.Equal(t, override.FlagKey, model.FlagKey)
	assert.Equal(t, override.TargetType, model.TargetType)
	assert.Equal(t, override.TargetID, model.TargetID)
	assert.Contains(t, model.ValueJSON, "control")
	assert.Equal(t, override.Reason, model.Reason)
	assert.Equal(t, override.ExpiresAt, model.ExpiresAt)
	assert.Equal(t, override.CreatedBy, model.CreatedBy)
}

func TestFlagOverrideModelFromDomain(t *testing.T) {
	targetID := uuid.New()

	override, err := featureflag.NewFlagOverride(
		"test_flag",
		featureflag.OverrideTargetTypeUser,
		targetID,
		featureflag.NewBooleanFlagValue(true),
		"",
		nil,
		nil,
	)
	require.NoError(t, err)

	model := FlagOverrideModelFromDomain(override)

	assert.NotNil(t, model)
	assert.Equal(t, override.ID, model.ID)
	assert.Equal(t, override.FlagKey, model.FlagKey)
}

func TestFlagAuditLogModel_TableName(t *testing.T) {
	model := FlagAuditLogModel{}
	assert.Equal(t, "flag_audit_logs", model.TableName())
}

func TestFlagAuditLogModel_ToDomain(t *testing.T) {
	userID := uuid.New()
	now := time.Now()

	model := &FlagAuditLogModel{
		BaseModel: BaseModel{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		FlagKey:      "test_flag",
		Action:       featureflag.AuditActionEnabled,
		OldValueJSON: `{"status":"disabled"}`,
		NewValueJSON: `{"status":"enabled"}`,
		UserID:       &userID,
		IPAddress:    "192.168.1.1",
		UserAgent:    "Mozilla/5.0",
	}

	domain := model.ToDomain()

	assert.Equal(t, model.ID, domain.ID)
	assert.Equal(t, model.FlagKey, domain.FlagKey)
	assert.Equal(t, model.Action, domain.Action)
	assert.Equal(t, "disabled", domain.OldValue["status"])
	assert.Equal(t, "enabled", domain.NewValue["status"])
	assert.Equal(t, model.UserID, domain.UserID)
	assert.Equal(t, model.IPAddress, domain.IPAddress)
	assert.Equal(t, model.UserAgent, domain.UserAgent)
}

func TestFlagAuditLogModel_FromDomain(t *testing.T) {
	userID := uuid.New()

	log, err := featureflag.NewFlagAuditLog(
		"test_flag",
		featureflag.AuditActionUpdated,
		map[string]any{"old": "value"},
		map[string]any{"new": "value"},
		&userID,
		"10.0.0.1",
		"Test Agent",
	)
	require.NoError(t, err)

	model := &FlagAuditLogModel{}
	model.FromDomain(log)

	assert.Equal(t, log.ID, model.ID)
	assert.Equal(t, log.FlagKey, model.FlagKey)
	assert.Equal(t, log.Action, model.Action)
	assert.Contains(t, model.OldValueJSON, "old")
	assert.Contains(t, model.NewValueJSON, "new")
	assert.Equal(t, log.UserID, model.UserID)
	assert.Equal(t, log.IPAddress, model.IPAddress)
	assert.Equal(t, log.UserAgent, model.UserAgent)
}

func TestFlagAuditLogModelFromDomain(t *testing.T) {
	log, err := featureflag.NewFlagAuditLog(
		"test_flag",
		featureflag.AuditActionCreated,
		nil,
		map[string]any{"key": "value"},
		nil,
		"",
		"",
	)
	require.NoError(t, err)

	model := FlagAuditLogModelFromDomain(log)

	assert.NotNil(t, model)
	assert.Equal(t, log.ID, model.ID)
	assert.Equal(t, log.FlagKey, model.FlagKey)
	assert.Equal(t, log.Action, model.Action)
}

func TestFlagAuditLogModel_EmptyValuesHandling(t *testing.T) {
	log, err := featureflag.NewFlagAuditLog(
		"test_flag",
		featureflag.AuditActionCreated,
		nil,
		nil,
		nil,
		"",
		"",
	)
	require.NoError(t, err)

	// Convert to model
	model := FlagAuditLogModelFromDomain(log)

	// Should have empty JSON objects, not null
	assert.Equal(t, "{}", model.OldValueJSON)
	assert.Equal(t, "{}", model.NewValueJSON)

	// Convert back to domain
	domainLog := model.ToDomain()

	// Should have empty maps, not nil
	assert.NotNil(t, domainLog.OldValue)
	assert.NotNil(t, domainLog.NewValue)
	assert.Empty(t, domainLog.OldValue)
	assert.Empty(t, domainLog.NewValue)
}
