package featureflag

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/erp/backend/internal/domain/shared"
)

// MockFeatureFlagRepository is a mock for FeatureFlagRepository
type MockFeatureFlagRepository struct {
	mock.Mock
}

func (m *MockFeatureFlagRepository) Create(ctx context.Context, flag *FeatureFlag) error {
	args := m.Called(ctx, flag)
	return args.Error(0)
}

func (m *MockFeatureFlagRepository) Update(ctx context.Context, flag *FeatureFlag) error {
	args := m.Called(ctx, flag)
	return args.Error(0)
}

func (m *MockFeatureFlagRepository) FindByKey(ctx context.Context, key string) (*FeatureFlag, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FeatureFlag), args.Error(1)
}

func (m *MockFeatureFlagRepository) FindByID(ctx context.Context, id uuid.UUID) (*FeatureFlag, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FeatureFlag), args.Error(1)
}

func (m *MockFeatureFlagRepository) FindAll(ctx context.Context, filter shared.Filter) ([]FeatureFlag, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]FeatureFlag), args.Error(1)
}

func (m *MockFeatureFlagRepository) FindByStatus(ctx context.Context, status FlagStatus, filter shared.Filter) ([]FeatureFlag, error) {
	args := m.Called(ctx, status, filter)
	return args.Get(0).([]FeatureFlag), args.Error(1)
}

func (m *MockFeatureFlagRepository) FindByTags(ctx context.Context, tags []string, filter shared.Filter) ([]FeatureFlag, error) {
	args := m.Called(ctx, tags, filter)
	return args.Get(0).([]FeatureFlag), args.Error(1)
}

func (m *MockFeatureFlagRepository) FindByType(ctx context.Context, flagType FlagType, filter shared.Filter) ([]FeatureFlag, error) {
	args := m.Called(ctx, flagType, filter)
	return args.Get(0).([]FeatureFlag), args.Error(1)
}

func (m *MockFeatureFlagRepository) FindEnabled(ctx context.Context, filter shared.Filter) ([]FeatureFlag, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]FeatureFlag), args.Error(1)
}

func (m *MockFeatureFlagRepository) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockFeatureFlagRepository) ExistsByKey(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockFeatureFlagRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockFeatureFlagRepository) CountByStatus(ctx context.Context, status FlagStatus) (int64, error) {
	args := m.Called(ctx, status)
	return args.Get(0).(int64), args.Error(1)
}

// MockFlagOverrideRepository is a mock for FlagOverrideRepository
type MockFlagOverrideRepository struct {
	mock.Mock
}

func (m *MockFlagOverrideRepository) Create(ctx context.Context, override *FlagOverride) error {
	args := m.Called(ctx, override)
	return args.Error(0)
}

func (m *MockFlagOverrideRepository) Update(ctx context.Context, override *FlagOverride) error {
	args := m.Called(ctx, override)
	return args.Error(0)
}

func (m *MockFlagOverrideRepository) FindByID(ctx context.Context, id uuid.UUID) (*FlagOverride, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FlagOverride), args.Error(1)
}

func (m *MockFlagOverrideRepository) FindByFlagKey(ctx context.Context, flagKey string, filter shared.Filter) ([]FlagOverride, error) {
	args := m.Called(ctx, flagKey, filter)
	return args.Get(0).([]FlagOverride), args.Error(1)
}

func (m *MockFlagOverrideRepository) FindByTarget(ctx context.Context, targetType OverrideTargetType, targetID uuid.UUID, filter shared.Filter) ([]FlagOverride, error) {
	args := m.Called(ctx, targetType, targetID, filter)
	return args.Get(0).([]FlagOverride), args.Error(1)
}

func (m *MockFlagOverrideRepository) FindForEvaluation(ctx context.Context, flagKey string, tenantID, userID *uuid.UUID) (*FlagOverride, error) {
	args := m.Called(ctx, flagKey, tenantID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FlagOverride), args.Error(1)
}

func (m *MockFlagOverrideRepository) FindByFlagKeyAndTarget(ctx context.Context, flagKey string, targetType OverrideTargetType, targetID uuid.UUID) (*FlagOverride, error) {
	args := m.Called(ctx, flagKey, targetType, targetID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FlagOverride), args.Error(1)
}

func (m *MockFlagOverrideRepository) FindExpired(ctx context.Context, filter shared.Filter) ([]FlagOverride, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]FlagOverride), args.Error(1)
}

func (m *MockFlagOverrideRepository) FindActive(ctx context.Context, filter shared.Filter) ([]FlagOverride, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]FlagOverride), args.Error(1)
}

func (m *MockFlagOverrideRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFlagOverrideRepository) DeleteByFlagKey(ctx context.Context, flagKey string) (int64, error) {
	args := m.Called(ctx, flagKey)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockFlagOverrideRepository) DeleteExpired(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockFlagOverrideRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockFlagOverrideRepository) CountByFlagKey(ctx context.Context, flagKey string) (int64, error) {
	args := m.Called(ctx, flagKey)
	return args.Get(0).(int64), args.Error(1)
}

// Helper to create a test flag
func createTestFlag(t *testing.T, key, name string, flagType FlagType, status FlagStatus) *FeatureFlag {
	flag, err := NewFeatureFlag(key, name, flagType, NewBooleanFlagValue(true), nil)
	require.NoError(t, err)
	if status == FlagStatusEnabled {
		err = flag.Enable(nil)
		require.NoError(t, err)
	}
	return flag
}

// Helper to create a test override
func createTestOverride(t *testing.T, flagKey string, targetType OverrideTargetType, targetID uuid.UUID, enabled bool) *FlagOverride {
	override, err := NewFlagOverride(flagKey, targetType, targetID, NewBooleanFlagValue(enabled), "test override", nil, nil)
	require.NoError(t, err)
	return override
}

func TestEvaluator_NewEvaluator(t *testing.T) {
	flagRepo := &MockFeatureFlagRepository{}
	overrideRepo := &MockFlagOverrideRepository{}

	evaluator := NewEvaluator(flagRepo, overrideRepo)

	assert.NotNil(t, evaluator)
}

func TestEvaluator_Evaluate_FlagNotFound(t *testing.T) {
	flagRepo := &MockFeatureFlagRepository{}
	overrideRepo := &MockFlagOverrideRepository{}

	flagRepo.On("FindByKey", mock.Anything, "nonexistent-flag").Return(nil, shared.NewDomainError("NOT_FOUND", "flag not found"))

	evaluator := NewEvaluator(flagRepo, overrideRepo)
	result := evaluator.Evaluate(context.Background(), "nonexistent-flag", NewEvaluationContext())

	assert.Equal(t, "nonexistent-flag", result.Key)
	assert.False(t, result.Enabled)
	assert.Equal(t, EvaluationReasonFlagNotFound, result.Reason)

	flagRepo.AssertExpectations(t)
}

func TestEvaluator_Evaluate_DatabaseError(t *testing.T) {
	flagRepo := &MockFeatureFlagRepository{}
	overrideRepo := &MockFlagOverrideRepository{}

	dbErr := shared.NewDomainError("DB_ERROR", "database connection failed")
	flagRepo.On("FindByKey", mock.Anything, "some-flag").Return(nil, dbErr)

	evaluator := NewEvaluator(flagRepo, overrideRepo)
	result := evaluator.Evaluate(context.Background(), "some-flag", NewEvaluationContext())

	assert.Equal(t, "some-flag", result.Key)
	assert.False(t, result.Enabled)
	assert.Equal(t, EvaluationReasonError, result.Reason)
	assert.True(t, result.HasError())
	assert.Equal(t, dbErr, result.GetError())

	flagRepo.AssertExpectations(t)
}

func TestEvaluator_Evaluate_DisabledFlag(t *testing.T) {
	flagRepo := &MockFeatureFlagRepository{}
	overrideRepo := &MockFlagOverrideRepository{}

	flag, _ := NewBooleanFlag("disabled-flag", "Disabled Flag", true, nil)
	// Flag is disabled by default

	flagRepo.On("FindByKey", mock.Anything, "disabled-flag").Return(flag, nil)

	evaluator := NewEvaluator(flagRepo, overrideRepo)
	result := evaluator.Evaluate(context.Background(), "disabled-flag", NewEvaluationContext())

	assert.Equal(t, "disabled-flag", result.Key)
	assert.False(t, result.Enabled)
	assert.Equal(t, EvaluationReasonDisabled, result.Reason)

	flagRepo.AssertExpectations(t)
}

func TestEvaluator_Evaluate_EnabledBooleanFlag(t *testing.T) {
	flagRepo := &MockFeatureFlagRepository{}
	overrideRepo := &MockFlagOverrideRepository{}

	flag := createTestFlag(t, "enabled-flag", "Enabled Flag", FlagTypeBoolean, FlagStatusEnabled)

	flagRepo.On("FindByKey", mock.Anything, "enabled-flag").Return(flag, nil)

	evaluator := NewEvaluator(flagRepo, overrideRepo)
	result := evaluator.Evaluate(context.Background(), "enabled-flag", NewEvaluationContext())

	assert.Equal(t, "enabled-flag", result.Key)
	assert.True(t, result.Enabled)
	assert.Equal(t, EvaluationReasonDefault, result.Reason)

	flagRepo.AssertExpectations(t)
}

func TestEvaluator_Evaluate_UserOverride(t *testing.T) {
	flagRepo := &MockFeatureFlagRepository{}
	overrideRepo := &MockFlagOverrideRepository{}

	flag := createTestFlag(t, "override-flag", "Override Flag", FlagTypeBoolean, FlagStatusEnabled)
	userID := uuid.New()

	userOverride, _ := NewFlagOverride(
		"override-flag",
		OverrideTargetTypeUser,
		userID,
		NewBooleanFlagValue(false), // Override to false
		"Test override",
		nil,
		nil,
	)

	flagRepo.On("FindByKey", mock.Anything, "override-flag").Return(flag, nil)
	overrideRepo.On("FindByFlagKeyAndTarget", mock.Anything, "override-flag", OverrideTargetTypeUser, userID).Return(userOverride, nil)

	evaluator := NewEvaluator(flagRepo, overrideRepo)
	evalCtx := NewEvaluationContext().WithUser(userID.String())
	result := evaluator.Evaluate(context.Background(), "override-flag", evalCtx)

	assert.Equal(t, "override-flag", result.Key)
	assert.False(t, result.Enabled) // Override value
	assert.Equal(t, EvaluationReasonOverrideUser, result.Reason)

	flagRepo.AssertExpectations(t)
	overrideRepo.AssertExpectations(t)
}

func TestEvaluator_Evaluate_TenantOverride(t *testing.T) {
	flagRepo := &MockFeatureFlagRepository{}
	overrideRepo := &MockFlagOverrideRepository{}

	flag := createTestFlag(t, "tenant-flag", "Tenant Flag", FlagTypeBoolean, FlagStatusEnabled)
	tenantID := uuid.New()

	tenantOverride, _ := NewFlagOverride(
		"tenant-flag",
		OverrideTargetTypeTenant,
		tenantID,
		NewVariantFlagValue("tenant-variant"),
		"Tenant specific",
		nil,
		nil,
	)

	flagRepo.On("FindByKey", mock.Anything, "tenant-flag").Return(flag, nil)
	overrideRepo.On("FindByFlagKeyAndTarget", mock.Anything, "tenant-flag", OverrideTargetTypeTenant, tenantID).Return(tenantOverride, nil)

	evaluator := NewEvaluator(flagRepo, overrideRepo)
	evalCtx := NewEvaluationContext().WithTenant(tenantID.String())
	result := evaluator.Evaluate(context.Background(), "tenant-flag", evalCtx)

	assert.Equal(t, "tenant-flag", result.Key)
	assert.True(t, result.Enabled)
	assert.Equal(t, "tenant-variant", result.Variant)
	assert.Equal(t, EvaluationReasonOverrideTenant, result.Reason)

	flagRepo.AssertExpectations(t)
	overrideRepo.AssertExpectations(t)
}

func TestEvaluator_Evaluate_UserOverrideTakesPrecedenceOverTenant(t *testing.T) {
	flagRepo := &MockFeatureFlagRepository{}
	overrideRepo := &MockFlagOverrideRepository{}

	flag := createTestFlag(t, "precedence-flag", "Precedence Flag", FlagTypeBoolean, FlagStatusEnabled)
	userID := uuid.New()
	tenantID := uuid.New()

	userOverride, _ := NewFlagOverride(
		"precedence-flag",
		OverrideTargetTypeUser,
		userID,
		NewBooleanFlagValue(false), // User override
		"User override",
		nil,
		nil,
	)

	// Note: tenant override should not be called because user override takes precedence
	flagRepo.On("FindByKey", mock.Anything, "precedence-flag").Return(flag, nil)
	overrideRepo.On("FindByFlagKeyAndTarget", mock.Anything, "precedence-flag", OverrideTargetTypeUser, userID).Return(userOverride, nil)

	evaluator := NewEvaluator(flagRepo, overrideRepo)
	evalCtx := NewEvaluationContext().WithUser(userID.String()).WithTenant(tenantID.String())
	result := evaluator.Evaluate(context.Background(), "precedence-flag", evalCtx)

	assert.Equal(t, EvaluationReasonOverrideUser, result.Reason)
	assert.False(t, result.Enabled) // User override value

	flagRepo.AssertExpectations(t)
	overrideRepo.AssertExpectations(t)
}

func TestEvaluator_Evaluate_ExpiredOverrideIgnored(t *testing.T) {
	flagRepo := &MockFeatureFlagRepository{}
	overrideRepo := &MockFlagOverrideRepository{}

	flag := createTestFlag(t, "expired-override-flag", "Expired Override Flag", FlagTypeBoolean, FlagStatusEnabled)
	userID := uuid.New()

	// Create an expired override
	pastTime := time.Now().Add(-1 * time.Hour)
	expiredOverride, _ := NewFlagOverride(
		"expired-override-flag",
		OverrideTargetTypeUser,
		userID,
		NewBooleanFlagValue(false),
		"Expired override",
		nil, // We'll set ExpiresAt manually
		nil,
	)
	expiredOverride.ExpiresAt = &pastTime

	flagRepo.On("FindByKey", mock.Anything, "expired-override-flag").Return(flag, nil)
	overrideRepo.On("FindByFlagKeyAndTarget", mock.Anything, "expired-override-flag", OverrideTargetTypeUser, userID).Return(expiredOverride, nil)

	evaluator := NewEvaluator(flagRepo, overrideRepo)
	evalCtx := NewEvaluationContext().WithUser(userID.String())
	result := evaluator.Evaluate(context.Background(), "expired-override-flag", evalCtx)

	// Should NOT use expired override, should return default
	assert.Equal(t, EvaluationReasonDefault, result.Reason)
	assert.True(t, result.Enabled) // Flag's default value

	flagRepo.AssertExpectations(t)
}

func TestEvaluator_Evaluate_TargetingRuleMatch(t *testing.T) {
	flagRepo := &MockFeatureFlagRepository{}
	overrideRepo := &MockFlagOverrideRepository{}

	flag := createTestFlag(t, "rule-flag", "Rule Flag", FlagTypeBoolean, FlagStatusEnabled)

	// Add a targeting rule
	condition, _ := NewCondition("user_role", ConditionOperatorEquals, []string{"admin"})
	rule, _ := NewTargetingRule("rule-1", 1, []Condition{condition}, NewBooleanFlagValue(true))
	err := flag.AddRule(rule, nil)
	require.NoError(t, err)

	flagRepo.On("FindByKey", mock.Anything, "rule-flag").Return(flag, nil)

	evaluator := NewEvaluator(flagRepo, overrideRepo)
	evalCtx := NewEvaluationContext().WithUserRole("admin")
	result := evaluator.Evaluate(context.Background(), "rule-flag", evalCtx)

	assert.Equal(t, "rule-flag", result.Key)
	assert.True(t, result.Enabled)
	assert.Equal(t, EvaluationReasonRuleMatch, result.Reason)
	assert.Equal(t, "rule-1", result.RuleID)

	flagRepo.AssertExpectations(t)
}

func TestEvaluator_Evaluate_TargetingRuleNoMatch(t *testing.T) {
	flagRepo := &MockFeatureFlagRepository{}
	overrideRepo := &MockFlagOverrideRepository{}

	flag := createTestFlag(t, "rule-no-match-flag", "Rule No Match Flag", FlagTypeBoolean, FlagStatusEnabled)

	// Add a targeting rule that won't match
	condition, _ := NewCondition("user_role", ConditionOperatorEquals, []string{"admin"})
	rule, _ := NewTargetingRule("rule-1", 1, []Condition{condition}, NewBooleanFlagValue(false))
	err := flag.AddRule(rule, nil)
	require.NoError(t, err)

	flagRepo.On("FindByKey", mock.Anything, "rule-no-match-flag").Return(flag, nil)

	evaluator := NewEvaluator(flagRepo, overrideRepo)
	evalCtx := NewEvaluationContext().WithUserRole("user") // Not admin
	result := evaluator.Evaluate(context.Background(), "rule-no-match-flag", evalCtx)

	assert.Equal(t, EvaluationReasonDefault, result.Reason) // Rule didn't match
	assert.True(t, result.Enabled)                          // Default value

	flagRepo.AssertExpectations(t)
}

func TestEvaluator_Evaluate_RulePriority(t *testing.T) {
	flagRepo := &MockFeatureFlagRepository{}
	overrideRepo := &MockFlagOverrideRepository{}

	flag := createTestFlag(t, "priority-flag", "Priority Flag", FlagTypeVariant, FlagStatusEnabled)

	// Add two rules - both match but different priorities
	condition1, _ := NewCondition("environment", ConditionOperatorEquals, []string{"production"})
	rule1, _ := NewTargetingRule("rule-low-priority", 10, []Condition{condition1}, NewVariantFlagValue("variant-low"))

	condition2, _ := NewCondition("environment", ConditionOperatorEquals, []string{"production"})
	rule2, _ := NewTargetingRule("rule-high-priority", 1, []Condition{condition2}, NewVariantFlagValue("variant-high"))

	// Add in reverse priority order to test sorting
	err := flag.AddRule(rule1, nil)
	require.NoError(t, err)
	err = flag.AddRule(rule2, nil)
	require.NoError(t, err)

	flagRepo.On("FindByKey", mock.Anything, "priority-flag").Return(flag, nil)

	evaluator := NewEvaluator(flagRepo, overrideRepo)
	evalCtx := NewEvaluationContext().WithEnvironment("production")
	result := evaluator.Evaluate(context.Background(), "priority-flag", evalCtx)

	assert.Equal(t, "variant-high", result.Variant) // Higher priority rule wins
	assert.Equal(t, "rule-high-priority", result.RuleID)

	flagRepo.AssertExpectations(t)
}

func TestEvaluator_Evaluate_RuleWithPercentage(t *testing.T) {
	flagRepo := &MockFeatureFlagRepository{}
	overrideRepo := &MockFlagOverrideRepository{}

	flag := createTestFlag(t, "percentage-rule-flag", "Percentage Rule Flag", FlagTypeBoolean, FlagStatusEnabled)

	// Add a rule with 50% rollout
	condition, _ := NewCondition("user_plan", ConditionOperatorEquals, []string{"pro"})
	rule, _ := NewTargetingRuleWithPercentage("rule-50", 1, []Condition{condition}, NewBooleanFlagValue(true), 50)
	err := flag.AddRule(rule, nil)
	require.NoError(t, err)

	flagRepo.On("FindByKey", mock.Anything, "percentage-rule-flag").Return(flag, nil)
	// Mock that no user overrides exist
	overrideRepo.On("FindByFlagKeyAndTarget", mock.Anything, "percentage-rule-flag", OverrideTargetTypeUser, mock.Anything).Return(nil, shared.NewDomainError("NOT_FOUND", "not found"))

	evaluator := NewEvaluator(flagRepo, overrideRepo)

	// Test with many users - some should match, some shouldn't
	matchCount := 0
	totalUsers := 100
	for i := range totalUsers {
		userID := uuid.New().String()
		_ = i
		evalCtx := NewEvaluationContext().WithUser(userID).WithUserPlan("pro")
		result := evaluator.Evaluate(context.Background(), "percentage-rule-flag", evalCtx)

		if result.Reason == EvaluationReasonRuleMatch {
			matchCount++
		}
	}

	// With 50% rollout, we should see roughly 50% matches (with some variance)
	assert.InDelta(t, 50, matchCount, 20, "Should have roughly 50% rule matches")

	flagRepo.AssertExpectations(t)
}

func TestEvaluator_Evaluate_CatchAllRule(t *testing.T) {
	flagRepo := &MockFeatureFlagRepository{}
	overrideRepo := &MockFlagOverrideRepository{}

	flag := createTestFlag(t, "catchall-flag", "Catch All Flag", FlagTypeVariant, FlagStatusEnabled)

	// Add a catch-all rule (no conditions)
	rule, _ := NewTargetingRule("catch-all", 1, []Condition{}, NewVariantFlagValue("catch-all-variant"))
	err := flag.AddRule(rule, nil)
	require.NoError(t, err)

	flagRepo.On("FindByKey", mock.Anything, "catchall-flag").Return(flag, nil)

	evaluator := NewEvaluator(flagRepo, overrideRepo)
	result := evaluator.Evaluate(context.Background(), "catchall-flag", NewEvaluationContext())

	assert.Equal(t, "catch-all-variant", result.Variant)
	assert.Equal(t, EvaluationReasonRuleMatch, result.Reason)
	assert.Equal(t, "catch-all", result.RuleID)

	flagRepo.AssertExpectations(t)
}

func TestEvaluator_EvaluateAll(t *testing.T) {
	flagRepo := &MockFeatureFlagRepository{}
	overrideRepo := &MockFlagOverrideRepository{}

	flag1 := createTestFlag(t, "flag1", "Flag 1", FlagTypeBoolean, FlagStatusEnabled)
	flag2 := createTestFlag(t, "flag2", "Flag 2", FlagTypeBoolean, FlagStatusEnabled)

	flagRepo.On("FindEnabled", mock.Anything, mock.Anything).Return([]FeatureFlag{*flag1, *flag2}, nil)

	evaluator := NewEvaluator(flagRepo, overrideRepo)
	results, err := evaluator.EvaluateAll(context.Background(), NewEvaluationContext())

	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.True(t, results["flag1"].Enabled)
	assert.True(t, results["flag2"].Enabled)

	flagRepo.AssertExpectations(t)
}

func TestEvaluator_EvaluateAll_Error(t *testing.T) {
	flagRepo := &MockFeatureFlagRepository{}
	overrideRepo := &MockFlagOverrideRepository{}

	flagRepo.On("FindEnabled", mock.Anything, mock.Anything).Return([]FeatureFlag{}, shared.NewDomainError("DB_ERROR", "database error"))

	evaluator := NewEvaluator(flagRepo, overrideRepo)
	results, err := evaluator.EvaluateAll(context.Background(), NewEvaluationContext())

	assert.Error(t, err)
	assert.Nil(t, results)

	flagRepo.AssertExpectations(t)
}

func TestEvaluator_EvaluateBatch(t *testing.T) {
	flagRepo := &MockFeatureFlagRepository{}
	overrideRepo := &MockFlagOverrideRepository{}

	flag1 := createTestFlag(t, "flag1", "Flag 1", FlagTypeBoolean, FlagStatusEnabled)
	flag2 := createTestFlag(t, "flag2", "Flag 2", FlagTypeBoolean, FlagStatusDisabled)

	flagRepo.On("FindByKey", mock.Anything, "flag1").Return(flag1, nil)
	flagRepo.On("FindByKey", mock.Anything, "flag2").Return(flag2, nil)
	flagRepo.On("FindByKey", mock.Anything, "flag3").Return(nil, shared.NewDomainError("NOT_FOUND", "not found"))

	evaluator := NewEvaluator(flagRepo, overrideRepo)
	results := evaluator.EvaluateBatch(context.Background(), []string{"flag1", "flag2", "flag3"}, NewEvaluationContext())

	assert.Len(t, results, 3)
	assert.True(t, results["flag1"].Enabled)
	assert.False(t, results["flag2"].Enabled)
	assert.Equal(t, EvaluationReasonDisabled, results["flag2"].Reason)
	assert.Equal(t, EvaluationReasonFlagNotFound, results["flag3"].Reason)

	flagRepo.AssertExpectations(t)
}

func TestEvaluator_IsEnabled(t *testing.T) {
	flagRepo := &MockFeatureFlagRepository{}
	overrideRepo := &MockFlagOverrideRepository{}

	flag := createTestFlag(t, "enabled-check", "Enabled Check", FlagTypeBoolean, FlagStatusEnabled)
	flagRepo.On("FindByKey", mock.Anything, "enabled-check").Return(flag, nil)

	evaluator := NewEvaluator(flagRepo, overrideRepo)
	isEnabled := evaluator.IsEnabled(context.Background(), "enabled-check", NewEvaluationContext())

	assert.True(t, isEnabled)
	flagRepo.AssertExpectations(t)
}

func TestEvaluator_GetVariant(t *testing.T) {
	flagRepo := &MockFeatureFlagRepository{}
	overrideRepo := &MockFlagOverrideRepository{}

	flag, _ := NewVariantFlag("variant-check", "Variant Check", "default-variant", nil)
	_ = flag.Enable(nil)
	flagRepo.On("FindByKey", mock.Anything, "variant-check").Return(flag, nil)

	evaluator := NewEvaluator(flagRepo, overrideRepo)
	variant := evaluator.GetVariant(context.Background(), "variant-check", NewEvaluationContext())

	assert.Equal(t, "default-variant", variant)
	flagRepo.AssertExpectations(t)
}

func TestEvaluator_NilOverrideRepo(t *testing.T) {
	flagRepo := &MockFeatureFlagRepository{}

	flag := createTestFlag(t, "no-override-repo", "No Override Repo", FlagTypeBoolean, FlagStatusEnabled)
	flagRepo.On("FindByKey", mock.Anything, "no-override-repo").Return(flag, nil)

	// Create evaluator with nil override repo
	evaluator := NewEvaluator(flagRepo, nil)
	evalCtx := NewEvaluationContext().WithUser(uuid.New().String())
	result := evaluator.Evaluate(context.Background(), "no-override-repo", evalCtx)

	// Should work fine, just skip override checks
	assert.Equal(t, EvaluationReasonDefault, result.Reason)
	assert.True(t, result.Enabled)

	flagRepo.AssertExpectations(t)
}

func TestPureEvaluator_Evaluate(t *testing.T) {
	evaluator := NewPureEvaluator()

	flag := createTestFlag(t, "pure-eval-flag", "Pure Eval Flag", FlagTypeBoolean, FlagStatusEnabled)
	result := evaluator.Evaluate(flag, NewEvaluationContext(), nil, nil)

	assert.Equal(t, "pure-eval-flag", result.Key)
	assert.True(t, result.Enabled)
	assert.Equal(t, EvaluationReasonDefault, result.Reason)
}

func TestPureEvaluator_Evaluate_WithUserOverride(t *testing.T) {
	evaluator := NewPureEvaluator()

	flag := createTestFlag(t, "pure-override-flag", "Pure Override Flag", FlagTypeBoolean, FlagStatusEnabled)

	userOverride, _ := NewFlagOverride(
		"pure-override-flag",
		OverrideTargetTypeUser,
		uuid.New(),
		NewBooleanFlagValue(false),
		"Test",
		nil,
		nil,
	)

	result := evaluator.Evaluate(flag, NewEvaluationContext(), userOverride, nil)

	assert.Equal(t, EvaluationReasonOverrideUser, result.Reason)
	assert.False(t, result.Enabled)
}

func TestPureEvaluator_Evaluate_WithTenantOverride(t *testing.T) {
	evaluator := NewPureEvaluator()

	flag := createTestFlag(t, "pure-tenant-flag", "Pure Tenant Flag", FlagTypeBoolean, FlagStatusEnabled)

	tenantOverride, _ := NewFlagOverride(
		"pure-tenant-flag",
		OverrideTargetTypeTenant,
		uuid.New(),
		NewBooleanFlagValue(false),
		"Test",
		nil,
		nil,
	)

	result := evaluator.Evaluate(flag, NewEvaluationContext(), nil, tenantOverride)

	assert.Equal(t, EvaluationReasonOverrideTenant, result.Reason)
	assert.False(t, result.Enabled)
}

func TestPureEvaluator_Evaluate_NilFlag(t *testing.T) {
	evaluator := NewPureEvaluator()
	result := evaluator.Evaluate(nil, NewEvaluationContext(), nil, nil)

	assert.Equal(t, EvaluationReasonFlagNotFound, result.Reason)
}

func TestPureEvaluator_Evaluate_DisabledFlag(t *testing.T) {
	evaluator := NewPureEvaluator()

	flag, _ := NewBooleanFlag("pure-disabled", "Pure Disabled", true, nil)
	// Flag is disabled by default

	result := evaluator.Evaluate(flag, NewEvaluationContext(), nil, nil)

	assert.Equal(t, EvaluationReasonDisabled, result.Reason)
	assert.False(t, result.Enabled)
}

func TestPureEvaluator_Evaluate_WithRules(t *testing.T) {
	evaluator := NewPureEvaluator()

	flag := createTestFlag(t, "pure-rules-flag", "Pure Rules Flag", FlagTypeBoolean, FlagStatusEnabled)

	condition, _ := NewCondition("user_role", ConditionOperatorEquals, []string{"admin"})
	rule, _ := NewTargetingRule("admin-rule", 1, []Condition{condition}, NewBooleanFlagValue(true))
	_ = flag.AddRule(rule, nil)

	evalCtx := NewEvaluationContext().WithUserRole("admin")
	result := evaluator.Evaluate(flag, evalCtx, nil, nil)

	assert.Equal(t, EvaluationReasonRuleMatch, result.Reason)
	assert.Equal(t, "admin-rule", result.RuleID)
}

func TestEvaluator_EvaluateFlag_NilFlag(t *testing.T) {
	flagRepo := &MockFeatureFlagRepository{}
	overrideRepo := &MockFlagOverrideRepository{}

	evaluator := NewEvaluator(flagRepo, overrideRepo)
	result := evaluator.EvaluateFlag(context.Background(), nil, NewEvaluationContext())

	assert.Equal(t, EvaluationReasonFlagNotFound, result.Reason)
}

// Tests for plan restriction in evaluation

func TestEvaluator_Evaluate_PlanRestriction(t *testing.T) {
	ctx := context.Background()

	t.Run("flag with no plan restriction allows all plans", func(t *testing.T) {
		flagRepo := &MockFeatureFlagRepository{}
		overrideRepo := &MockFlagOverrideRepository{}
		evaluator := NewEvaluator(flagRepo, overrideRepo)

		flag := createTestFlag(t, "no-restriction", "No Restriction", FlagTypeBoolean, FlagStatusEnabled)
		// No plan restriction set

		evalCtx := NewEvaluationContext().WithUserPlan("free")
		result := evaluator.EvaluateFlag(ctx, flag, evalCtx)

		assert.NotEqual(t, EvaluationReasonPlanRestricted, result.Reason)
		assert.True(t, result.Enabled)
	})

	t.Run("free tenant cannot access pro feature", func(t *testing.T) {
		flagRepo := &MockFeatureFlagRepository{}
		overrideRepo := &MockFlagOverrideRepository{}
		evaluator := NewEvaluator(flagRepo, overrideRepo)

		flag := createTestFlag(t, "pro-feature", "Pro Feature", FlagTypeBoolean, FlagStatusEnabled)
		_ = flag.SetRequiredPlan(RequiredPlanPro, nil)

		evalCtx := NewEvaluationContext().WithUserPlan("free")
		result := evaluator.EvaluateFlag(ctx, flag, evalCtx)

		assert.Equal(t, EvaluationReasonPlanRestricted, result.Reason)
		assert.False(t, result.Enabled)
		assert.True(t, result.IsPlanRestricted())

		// Check that required_plan is in metadata
		requiredPlan, ok := result.Value.GetMetadataValue("required_plan")
		assert.True(t, ok)
		assert.Equal(t, "pro", requiredPlan)
	})

	t.Run("basic tenant cannot access pro feature", func(t *testing.T) {
		flagRepo := &MockFeatureFlagRepository{}
		overrideRepo := &MockFlagOverrideRepository{}
		evaluator := NewEvaluator(flagRepo, overrideRepo)

		flag := createTestFlag(t, "pro-feature", "Pro Feature", FlagTypeBoolean, FlagStatusEnabled)
		_ = flag.SetRequiredPlan(RequiredPlanPro, nil)

		evalCtx := NewEvaluationContext().WithUserPlan("basic")
		result := evaluator.EvaluateFlag(ctx, flag, evalCtx)

		assert.Equal(t, EvaluationReasonPlanRestricted, result.Reason)
		assert.False(t, result.Enabled)
	})

	t.Run("pro tenant can access pro feature", func(t *testing.T) {
		flagRepo := &MockFeatureFlagRepository{}
		overrideRepo := &MockFlagOverrideRepository{}
		evaluator := NewEvaluator(flagRepo, overrideRepo)

		flag := createTestFlag(t, "pro-feature", "Pro Feature", FlagTypeBoolean, FlagStatusEnabled)
		_ = flag.SetRequiredPlan(RequiredPlanPro, nil)

		evalCtx := NewEvaluationContext().WithUserPlan("pro")
		result := evaluator.EvaluateFlag(ctx, flag, evalCtx)

		assert.NotEqual(t, EvaluationReasonPlanRestricted, result.Reason)
		assert.True(t, result.Enabled)
	})

	t.Run("enterprise tenant can access pro feature", func(t *testing.T) {
		flagRepo := &MockFeatureFlagRepository{}
		overrideRepo := &MockFlagOverrideRepository{}
		evaluator := NewEvaluator(flagRepo, overrideRepo)

		flag := createTestFlag(t, "pro-feature", "Pro Feature", FlagTypeBoolean, FlagStatusEnabled)
		_ = flag.SetRequiredPlan(RequiredPlanPro, nil)

		evalCtx := NewEvaluationContext().WithUserPlan("enterprise")
		result := evaluator.EvaluateFlag(ctx, flag, evalCtx)

		assert.NotEqual(t, EvaluationReasonPlanRestricted, result.Reason)
		assert.True(t, result.Enabled)
	})

	t.Run("enterprise feature only accessible by enterprise", func(t *testing.T) {
		flagRepo := &MockFeatureFlagRepository{}
		overrideRepo := &MockFlagOverrideRepository{}
		evaluator := NewEvaluator(flagRepo, overrideRepo)

		flag := createTestFlag(t, "enterprise-feature", "Enterprise Feature", FlagTypeBoolean, FlagStatusEnabled)
		_ = flag.SetRequiredPlan(RequiredPlanEnterprise, nil)

		// Pro tenant cannot access
		evalCtx := NewEvaluationContext().WithUserPlan("pro")
		result := evaluator.EvaluateFlag(ctx, flag, evalCtx)
		assert.Equal(t, EvaluationReasonPlanRestricted, result.Reason)

		// Enterprise tenant can access
		evalCtx = NewEvaluationContext().WithUserPlan("enterprise")
		result = evaluator.EvaluateFlag(ctx, flag, evalCtx)
		assert.NotEqual(t, EvaluationReasonPlanRestricted, result.Reason)
		assert.True(t, result.Enabled)
	})

	t.Run("nil context defaults to no plan (restricted)", func(t *testing.T) {
		flagRepo := &MockFeatureFlagRepository{}
		overrideRepo := &MockFlagOverrideRepository{}
		evaluator := NewEvaluator(flagRepo, overrideRepo)

		flag := createTestFlag(t, "basic-feature", "Basic Feature", FlagTypeBoolean, FlagStatusEnabled)
		_ = flag.SetRequiredPlan(RequiredPlanBasic, nil)

		result := evaluator.EvaluateFlag(ctx, flag, nil)
		assert.Equal(t, EvaluationReasonPlanRestricted, result.Reason)
	})

	t.Run("empty plan in context defaults to free (restricted for basic+)", func(t *testing.T) {
		flagRepo := &MockFeatureFlagRepository{}
		overrideRepo := &MockFlagOverrideRepository{}
		evaluator := NewEvaluator(flagRepo, overrideRepo)

		flag := createTestFlag(t, "basic-feature", "Basic Feature", FlagTypeBoolean, FlagStatusEnabled)
		_ = flag.SetRequiredPlan(RequiredPlanBasic, nil)

		evalCtx := NewEvaluationContext() // No plan set
		result := evaluator.EvaluateFlag(ctx, flag, evalCtx)
		assert.Equal(t, EvaluationReasonPlanRestricted, result.Reason)
	})

	t.Run("plan check happens after disabled check", func(t *testing.T) {
		flagRepo := &MockFeatureFlagRepository{}
		overrideRepo := &MockFlagOverrideRepository{}
		evaluator := NewEvaluator(flagRepo, overrideRepo)

		flag := createTestFlag(t, "disabled-pro", "Disabled Pro", FlagTypeBoolean, FlagStatusDisabled)
		_ = flag.SetRequiredPlan(RequiredPlanPro, nil)

		evalCtx := NewEvaluationContext().WithUserPlan("free")
		result := evaluator.EvaluateFlag(ctx, flag, evalCtx)

		// Should return disabled, not plan_restricted
		assert.Equal(t, EvaluationReasonDisabled, result.Reason)
	})

	t.Run("override bypasses plan restriction", func(t *testing.T) {
		flagRepo := &MockFeatureFlagRepository{}
		overrideRepo := &MockFlagOverrideRepository{}
		evaluator := NewEvaluator(flagRepo, overrideRepo)

		flag := createTestFlag(t, "pro-feature", "Pro Feature", FlagTypeBoolean, FlagStatusEnabled)
		_ = flag.SetRequiredPlan(RequiredPlanPro, nil)

		tenantID := uuid.New()
		override := createTestOverride(t, "pro-feature", OverrideTargetTypeTenant, tenantID, true)

		overrideRepo.On("FindByFlagKeyAndTarget", mock.Anything, "pro-feature", OverrideTargetTypeTenant, tenantID).
			Return(override, nil)

		evalCtx := NewEvaluationContext().WithTenant(tenantID.String()).WithUserPlan("free")
		result := evaluator.EvaluateFlag(ctx, flag, evalCtx)

		// Override should take precedence over plan restriction
		assert.Equal(t, EvaluationReasonOverrideTenant, result.Reason)
		assert.True(t, result.Enabled)
	})
}

func TestPureEvaluator_Evaluate_PlanRestriction(t *testing.T) {
	evaluator := NewPureEvaluator()

	t.Run("free tenant cannot access pro feature", func(t *testing.T) {
		flag := createTestFlag(t, "pro-feature", "Pro Feature", FlagTypeBoolean, FlagStatusEnabled)
		_ = flag.SetRequiredPlan(RequiredPlanPro, nil)

		evalCtx := NewEvaluationContext().WithUserPlan("free")
		result := evaluator.Evaluate(flag, evalCtx, nil, nil)

		assert.Equal(t, EvaluationReasonPlanRestricted, result.Reason)
		assert.False(t, result.Enabled)
	})

	t.Run("pro tenant can access pro feature", func(t *testing.T) {
		flag := createTestFlag(t, "pro-feature", "Pro Feature", FlagTypeBoolean, FlagStatusEnabled)
		_ = flag.SetRequiredPlan(RequiredPlanPro, nil)

		evalCtx := NewEvaluationContext().WithUserPlan("pro")
		result := evaluator.Evaluate(flag, evalCtx, nil, nil)

		assert.NotEqual(t, EvaluationReasonPlanRestricted, result.Reason)
		assert.True(t, result.Enabled)
	})

	t.Run("override bypasses plan restriction", func(t *testing.T) {
		flag := createTestFlag(t, "pro-feature", "Pro Feature", FlagTypeBoolean, FlagStatusEnabled)
		_ = flag.SetRequiredPlan(RequiredPlanPro, nil)

		tenantID := uuid.New()
		override := createTestOverride(t, "pro-feature", OverrideTargetTypeTenant, tenantID, true)

		evalCtx := NewEvaluationContext().WithUserPlan("free")
		result := evaluator.Evaluate(flag, evalCtx, nil, override)

		// Override should take precedence
		assert.Equal(t, EvaluationReasonOverrideTenant, result.Reason)
		assert.True(t, result.Enabled)
	})

	t.Run("upgrade plan grants access", func(t *testing.T) {
		flag := createTestFlag(t, "basic-feature", "Basic Feature", FlagTypeBoolean, FlagStatusEnabled)
		_ = flag.SetRequiredPlan(RequiredPlanBasic, nil)

		// Free tenant - no access
		evalCtx := NewEvaluationContext().WithUserPlan("free")
		result := evaluator.Evaluate(flag, evalCtx, nil, nil)
		assert.Equal(t, EvaluationReasonPlanRestricted, result.Reason)

		// After upgrade to basic - has access
		evalCtx = NewEvaluationContext().WithUserPlan("basic")
		result = evaluator.Evaluate(flag, evalCtx, nil, nil)
		assert.NotEqual(t, EvaluationReasonPlanRestricted, result.Reason)
		assert.True(t, result.Enabled)
	})
}
