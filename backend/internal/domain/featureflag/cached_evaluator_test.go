package featureflag

import (
	"context"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCachedEvaluator_Evaluate(t *testing.T) {
	ctx := context.Background()

	// Setup
	flagRepo := newMockFlagRepo()
	overrideRepo := newMockOverrideRepo()
	cache := newMockCache()

	// Create a test flag
	flag, _ := NewFeatureFlag("test-flag", "Test Flag", FlagTypeBoolean, NewBooleanFlagValue(true), nil)
	flag.Enable(nil)
	flagRepo.flags["test-flag"] = flag

	evaluator := NewCachedEvaluator(flagRepo, overrideRepo, cache, WithCachedEvaluatorLogger(zap.NewNop()))

	// First evaluation - cache miss, should fetch from repo
	result := evaluator.Evaluate(ctx, "test-flag", nil)
	assert.True(t, result.IsEnabled())
	assert.Equal(t, EvaluationReasonDefault, result.Reason)

	// Verify flag was cached
	cachedFlag, _ := cache.Get(ctx, "test-flag")
	require.NotNil(t, cachedFlag)
	assert.Equal(t, "test-flag", cachedFlag.GetKey())

	// Second evaluation - cache hit
	result = evaluator.Evaluate(ctx, "test-flag", nil)
	assert.True(t, result.IsEnabled())
}

func TestCachedEvaluator_EvaluateNotFound(t *testing.T) {
	ctx := context.Background()

	// Setup with empty repos
	flagRepo := newMockFlagRepo()
	overrideRepo := newMockOverrideRepo()
	cache := newMockCache()

	evaluator := NewCachedEvaluator(flagRepo, overrideRepo, cache)

	// Evaluate non-existent flag
	result := evaluator.Evaluate(ctx, "non-existent", nil)
	assert.Equal(t, EvaluationReasonFlagNotFound, result.Reason)
}

func TestCachedEvaluator_EvaluateWithOverrides(t *testing.T) {
	ctx := context.Background()

	// Setup
	flagRepo := newMockFlagRepo()
	overrideRepo := newMockOverrideRepo()
	cache := newMockCache()

	// Create a flag that's enabled with default true
	flag, _ := NewFeatureFlag("test-flag", "Test Flag", FlagTypeBoolean, NewBooleanFlagValue(true), nil)
	flag.Enable(nil)
	flagRepo.flags["test-flag"] = flag

	// Create a user override that sets false
	userID := uuid.New()
	override, _ := NewFlagOverride("test-flag", OverrideTargetTypeUser, userID, NewBooleanFlagValue(false), "test", nil, nil)
	overrideRepo.overrides[overrideRepo.makeKey("test-flag", OverrideTargetTypeUser, userID)] = override

	evaluator := NewCachedEvaluator(flagRepo, overrideRepo, cache)

	// Evaluate without context - should get default (true)
	result := evaluator.Evaluate(ctx, "test-flag", nil)
	assert.True(t, result.IsEnabled())
	assert.Equal(t, EvaluationReasonDefault, result.Reason)

	// Evaluate with user context - should get override (false)
	evalCtx := &EvaluationContext{
		UserID: userID.String(),
	}
	result = evaluator.Evaluate(ctx, "test-flag", evalCtx)
	assert.False(t, result.IsEnabled())
	assert.Equal(t, EvaluationReasonOverrideUser, result.Reason)
}

func TestCachedEvaluator_EvaluateBatch(t *testing.T) {
	ctx := context.Background()

	// Setup
	flagRepo := newMockFlagRepo()
	overrideRepo := newMockOverrideRepo()
	cache := newMockCache()

	// Create test flags
	flag1, _ := NewFeatureFlag("flag-1", "Flag 1", FlagTypeBoolean, NewBooleanFlagValue(true), nil)
	flag1.Enable(nil)
	flag2, _ := NewFeatureFlag("flag-2", "Flag 2", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
	flag2.Enable(nil)
	flagRepo.flags["flag-1"] = flag1
	flagRepo.flags["flag-2"] = flag2

	evaluator := NewCachedEvaluator(flagRepo, overrideRepo, cache)

	// Batch evaluate
	keys := []string{"flag-1", "flag-2", "flag-3"}
	results := evaluator.EvaluateBatch(ctx, keys, nil)

	assert.Len(t, results, 3)
	assert.True(t, results["flag-1"].IsEnabled())
	assert.False(t, results["flag-2"].IsEnabled())
	assert.Equal(t, EvaluationReasonFlagNotFound, results["flag-3"].Reason)
}

func TestCachedEvaluator_EvaluateAll(t *testing.T) {
	ctx := context.Background()

	// Setup
	flagRepo := newMockFlagRepo()
	overrideRepo := newMockOverrideRepo()
	cache := newMockCache()

	// Create enabled and disabled flags
	enabledFlag, _ := NewFeatureFlag("enabled-flag", "Enabled", FlagTypeBoolean, NewBooleanFlagValue(true), nil)
	enabledFlag.Enable(nil)
	disabledFlag, _ := NewFeatureFlag("disabled-flag", "Disabled", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
	flagRepo.flags["enabled-flag"] = enabledFlag
	flagRepo.flags["disabled-flag"] = disabledFlag

	evaluator := NewCachedEvaluator(flagRepo, overrideRepo, cache)

	// Evaluate all - should only get enabled flags
	results, err := evaluator.EvaluateAll(ctx, nil)
	require.NoError(t, err)

	// Only enabled flag should be returned
	assert.Len(t, results, 1)
	assert.Contains(t, results, "enabled-flag")
	assert.True(t, results["enabled-flag"].IsEnabled())
}

func TestCachedEvaluator_InvalidateFlag(t *testing.T) {
	ctx := context.Background()

	// Setup
	flagRepo := newMockFlagRepo()
	overrideRepo := newMockOverrideRepo()
	cache := newMockCache()

	// Create and cache a flag
	flag, _ := NewFeatureFlag("test-flag", "Test", FlagTypeBoolean, NewBooleanFlagValue(true), nil)
	flag.Enable(nil)
	flagRepo.flags["test-flag"] = flag
	cache.Set(ctx, "test-flag", flag, 5*time.Minute)

	evaluator := NewCachedEvaluator(flagRepo, overrideRepo, cache)

	// Verify flag is cached
	cachedFlag, _ := cache.Get(ctx, "test-flag")
	require.NotNil(t, cachedFlag)

	// Invalidate
	err := evaluator.InvalidateFlag(ctx, "test-flag")
	require.NoError(t, err)

	// Verify flag is no longer cached
	cachedFlag, _ = cache.Get(ctx, "test-flag")
	assert.Nil(t, cachedFlag)
}

func TestCachedEvaluator_WarmupCache(t *testing.T) {
	ctx := context.Background()

	// Setup
	flagRepo := newMockFlagRepo()
	overrideRepo := newMockOverrideRepo()
	cache := newMockCache()

	// Create enabled flags
	flag1, _ := NewFeatureFlag("flag-1", "Flag 1", FlagTypeBoolean, NewBooleanFlagValue(true), nil)
	flag1.Enable(nil)
	flag2, _ := NewFeatureFlag("flag-2", "Flag 2", FlagTypeBoolean, NewBooleanFlagValue(true), nil)
	flag2.Enable(nil)
	flagRepo.flags["flag-1"] = flag1
	flagRepo.flags["flag-2"] = flag2

	evaluator := NewCachedEvaluator(flagRepo, overrideRepo, cache, WithCachedEvaluatorLogger(zap.NewNop()))

	// Verify cache is empty
	assert.Equal(t, 0, cache.flagCount)

	// Warmup cache
	err := evaluator.WarmupCache(ctx)
	require.NoError(t, err)

	// Verify flags were cached
	assert.Equal(t, 2, cache.flagCount)
}

func TestCachedEvaluator_NilCache(t *testing.T) {
	ctx := context.Background()

	// Setup with nil cache
	flagRepo := newMockFlagRepo()
	overrideRepo := newMockOverrideRepo()

	flag, _ := NewFeatureFlag("test-flag", "Test", FlagTypeBoolean, NewBooleanFlagValue(true), nil)
	flag.Enable(nil)
	flagRepo.flags["test-flag"] = flag

	evaluator := NewCachedEvaluator(flagRepo, overrideRepo, nil)

	// Should still work without cache
	result := evaluator.Evaluate(ctx, "test-flag", nil)
	assert.True(t, result.IsEnabled())

	// Invalidate should not error
	err := evaluator.InvalidateFlag(ctx, "test-flag")
	assert.NoError(t, err)
}

// Mock implementations

type mockFlagRepo struct {
	flags map[string]*FeatureFlag
}

func newMockFlagRepo() *mockFlagRepo {
	return &mockFlagRepo{
		flags: make(map[string]*FeatureFlag),
	}
}

func (m *mockFlagRepo) Create(ctx context.Context, flag *FeatureFlag) error {
	m.flags[flag.GetKey()] = flag
	return nil
}

func (m *mockFlagRepo) Update(ctx context.Context, flag *FeatureFlag) error {
	m.flags[flag.GetKey()] = flag
	return nil
}

func (m *mockFlagRepo) FindByKey(ctx context.Context, key string) (*FeatureFlag, error) {
	if flag, ok := m.flags[key]; ok {
		return flag, nil
	}
	return nil, shared.NewDomainError("NOT_FOUND", "Flag not found")
}

func (m *mockFlagRepo) FindByID(ctx context.Context, id uuid.UUID) (*FeatureFlag, error) {
	return nil, shared.NewDomainError("NOT_FOUND", "Flag not found")
}

func (m *mockFlagRepo) FindAll(ctx context.Context, filter shared.Filter) ([]FeatureFlag, error) {
	result := make([]FeatureFlag, 0, len(m.flags))
	for _, flag := range m.flags {
		result = append(result, *flag)
	}
	return result, nil
}

func (m *mockFlagRepo) FindByStatus(ctx context.Context, status FlagStatus, filter shared.Filter) ([]FeatureFlag, error) {
	return m.FindAll(ctx, filter)
}

func (m *mockFlagRepo) FindByTags(ctx context.Context, tags []string, filter shared.Filter) ([]FeatureFlag, error) {
	return m.FindAll(ctx, filter)
}

func (m *mockFlagRepo) FindByType(ctx context.Context, flagType FlagType, filter shared.Filter) ([]FeatureFlag, error) {
	return m.FindAll(ctx, filter)
}

func (m *mockFlagRepo) FindEnabled(ctx context.Context, filter shared.Filter) ([]FeatureFlag, error) {
	result := make([]FeatureFlag, 0)
	for _, flag := range m.flags {
		if flag.IsEnabled() {
			result = append(result, *flag)
		}
	}
	return result, nil
}

func (m *mockFlagRepo) Delete(ctx context.Context, key string) error {
	delete(m.flags, key)
	return nil
}

func (m *mockFlagRepo) ExistsByKey(ctx context.Context, key string) (bool, error) {
	_, ok := m.flags[key]
	return ok, nil
}

func (m *mockFlagRepo) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	return int64(len(m.flags)), nil
}

func (m *mockFlagRepo) CountByStatus(ctx context.Context, status FlagStatus) (int64, error) {
	return int64(len(m.flags)), nil
}

type mockOverrideRepo struct {
	overrides map[string]*FlagOverride
}

func newMockOverrideRepo() *mockOverrideRepo {
	return &mockOverrideRepo{
		overrides: make(map[string]*FlagOverride),
	}
}

func (m *mockOverrideRepo) makeKey(flagKey string, targetType OverrideTargetType, targetID uuid.UUID) string {
	return flagKey + ":" + string(targetType) + ":" + targetID.String()
}

func (m *mockOverrideRepo) Create(ctx context.Context, override *FlagOverride) error {
	key := m.makeKey(override.FlagKey, override.TargetType, override.TargetID)
	m.overrides[key] = override
	return nil
}

func (m *mockOverrideRepo) Update(ctx context.Context, override *FlagOverride) error {
	return m.Create(ctx, override)
}

func (m *mockOverrideRepo) FindByID(ctx context.Context, id uuid.UUID) (*FlagOverride, error) {
	return nil, shared.NewDomainError("NOT_FOUND", "Override not found")
}

func (m *mockOverrideRepo) FindByFlagKey(ctx context.Context, flagKey string, filter shared.Filter) ([]FlagOverride, error) {
	return []FlagOverride{}, nil
}

func (m *mockOverrideRepo) FindByTarget(ctx context.Context, targetType OverrideTargetType, targetID uuid.UUID, filter shared.Filter) ([]FlagOverride, error) {
	return []FlagOverride{}, nil
}

func (m *mockOverrideRepo) FindForEvaluation(ctx context.Context, flagKey string, tenantID, userID *uuid.UUID) (*FlagOverride, error) {
	return nil, nil
}

func (m *mockOverrideRepo) FindByFlagKeyAndTarget(ctx context.Context, flagKey string, targetType OverrideTargetType, targetID uuid.UUID) (*FlagOverride, error) {
	key := m.makeKey(flagKey, targetType, targetID)
	if override, ok := m.overrides[key]; ok {
		return override, nil
	}
	return nil, shared.NewDomainError("NOT_FOUND", "Override not found")
}

func (m *mockOverrideRepo) FindExpired(ctx context.Context, filter shared.Filter) ([]FlagOverride, error) {
	return []FlagOverride{}, nil
}

func (m *mockOverrideRepo) FindActive(ctx context.Context, filter shared.Filter) ([]FlagOverride, error) {
	return []FlagOverride{}, nil
}

func (m *mockOverrideRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockOverrideRepo) DeleteByFlagKey(ctx context.Context, flagKey string) (int64, error) {
	return 0, nil
}

func (m *mockOverrideRepo) DeleteExpired(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *mockOverrideRepo) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	return int64(len(m.overrides)), nil
}

func (m *mockOverrideRepo) CountByFlagKey(ctx context.Context, flagKey string) (int64, error) {
	return 0, nil
}

type mockCache struct {
	flags     map[string]*FeatureFlag
	overrides map[string]*FlagOverride
	flagCount int
}

func newMockCache() *mockCache {
	return &mockCache{
		flags:     make(map[string]*FeatureFlag),
		overrides: make(map[string]*FlagOverride),
	}
}

func (m *mockCache) Get(ctx context.Context, key string) (*FeatureFlag, error) {
	if flag, ok := m.flags[key]; ok {
		return flag, nil
	}
	return nil, nil
}

func (m *mockCache) Set(ctx context.Context, key string, flag *FeatureFlag, ttl time.Duration) error {
	if flag != nil {
		m.flags[key] = flag
		m.flagCount = len(m.flags)
	}
	return nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	delete(m.flags, key)
	m.flagCount = len(m.flags)
	return nil
}

func (m *mockCache) GetOverride(ctx context.Context, flagKey string, targetType OverrideTargetType, targetID uuid.UUID) (*FlagOverride, error) {
	key := flagKey + ":" + string(targetType) + ":" + targetID.String()
	if override, ok := m.overrides[key]; ok {
		return override, nil
	}
	return nil, nil
}

func (m *mockCache) SetOverride(ctx context.Context, override *FlagOverride, ttl time.Duration) error {
	if override != nil {
		key := override.FlagKey + ":" + string(override.TargetType) + ":" + override.TargetID.String()
		m.overrides[key] = override
	}
	return nil
}

func (m *mockCache) DeleteOverride(ctx context.Context, flagKey string, targetType OverrideTargetType, targetID uuid.UUID) error {
	key := flagKey + ":" + string(targetType) + ":" + targetID.String()
	delete(m.overrides, key)
	return nil
}

func (m *mockCache) InvalidateAll(ctx context.Context) error {
	m.flags = make(map[string]*FeatureFlag)
	m.overrides = make(map[string]*FlagOverride)
	m.flagCount = 0
	return nil
}

func (m *mockCache) Close() error {
	return nil
}

var _ FlagCache = (*mockCache)(nil)
var _ FeatureFlagRepository = (*mockFlagRepo)(nil)
var _ FlagOverrideRepository = (*mockOverrideRepo)(nil)
