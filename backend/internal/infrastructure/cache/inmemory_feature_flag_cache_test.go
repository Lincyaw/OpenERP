package cache

import (
	"context"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/featureflag"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryFeatureFlagCache_Get(t *testing.T) {
	cache := NewInMemoryFeatureFlagCache()

	ctx := context.Background()
	key := "test-flag"

	// Test cache miss
	flag, err := cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Nil(t, flag)

	// Create and set a flag
	testFlag := createTestFlag(key)
	err = cache.Set(ctx, key, testFlag, 5*time.Second)
	require.NoError(t, err)

	// Test cache hit
	flag, err = cache.Get(ctx, key)
	require.NoError(t, err)
	require.NotNil(t, flag)
	assert.Equal(t, key, flag.GetKey())
}

func TestInMemoryFeatureFlagCache_Set(t *testing.T) {
	cache := NewInMemoryFeatureFlagCache()

	ctx := context.Background()
	key := "test-flag"
	testFlag := createTestFlag(key)

	// Set with explicit TTL
	err := cache.Set(ctx, key, testFlag, 5*time.Second)
	require.NoError(t, err)

	// Verify it was set
	flag, err := cache.Get(ctx, key)
	require.NoError(t, err)
	require.NotNil(t, flag)
	assert.Equal(t, key, flag.GetKey())

	// Set nil flag (should be no-op)
	err = cache.Set(ctx, "nil-flag", nil, 5*time.Second)
	require.NoError(t, err)
}

func TestInMemoryFeatureFlagCache_Delete(t *testing.T) {
	cache := NewInMemoryFeatureFlagCache()

	ctx := context.Background()
	key := "test-flag"
	testFlag := createTestFlag(key)

	// Set a flag
	err := cache.Set(ctx, key, testFlag, 5*time.Second)
	require.NoError(t, err)

	// Delete it
	err = cache.Delete(ctx, key)
	require.NoError(t, err)

	// Verify it's gone
	flag, err := cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Nil(t, flag)
}

func TestInMemoryFeatureFlagCache_Expiration(t *testing.T) {
	cache := NewInMemoryFeatureFlagCache()

	ctx := context.Background()
	key := "test-flag"
	testFlag := createTestFlag(key)

	// Set with very short TTL
	err := cache.Set(ctx, key, testFlag, 50*time.Millisecond)
	require.NoError(t, err)

	// Verify it's there
	flag, err := cache.Get(ctx, key)
	require.NoError(t, err)
	require.NotNil(t, flag)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Verify it's expired
	flag, err = cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Nil(t, flag)
}

func TestInMemoryFeatureFlagCache_GetOverride(t *testing.T) {
	cache := NewInMemoryFeatureFlagCache()

	ctx := context.Background()
	flagKey := "test-flag"
	targetType := featureflag.OverrideTargetTypeUser
	targetID := uuid.New()

	// Test cache miss
	override, err := cache.GetOverride(ctx, flagKey, targetType, targetID)
	require.NoError(t, err)
	assert.Nil(t, override)

	// Create and set an override
	testOverride := createTestOverride(flagKey, targetType, targetID)
	err = cache.SetOverride(ctx, testOverride, 5*time.Second)
	require.NoError(t, err)

	// Test cache hit
	override, err = cache.GetOverride(ctx, flagKey, targetType, targetID)
	require.NoError(t, err)
	require.NotNil(t, override)
	assert.Equal(t, flagKey, override.GetFlagKey())
	assert.Equal(t, targetType, override.GetTargetType())
	assert.Equal(t, targetID, override.GetTargetID())
}

func TestInMemoryFeatureFlagCache_SetOverride(t *testing.T) {
	cache := NewInMemoryFeatureFlagCache()

	ctx := context.Background()
	flagKey := "test-flag"
	targetType := featureflag.OverrideTargetTypeTenant
	targetID := uuid.New()
	testOverride := createTestOverride(flagKey, targetType, targetID)

	// Set with explicit TTL
	err := cache.SetOverride(ctx, testOverride, 5*time.Second)
	require.NoError(t, err)

	// Verify it was set
	override, err := cache.GetOverride(ctx, flagKey, targetType, targetID)
	require.NoError(t, err)
	require.NotNil(t, override)

	// Set nil override (should be no-op)
	err = cache.SetOverride(ctx, nil, 5*time.Second)
	require.NoError(t, err)
}

func TestInMemoryFeatureFlagCache_DeleteOverride(t *testing.T) {
	cache := NewInMemoryFeatureFlagCache()

	ctx := context.Background()
	flagKey := "test-flag"
	targetType := featureflag.OverrideTargetTypeUser
	targetID := uuid.New()
	testOverride := createTestOverride(flagKey, targetType, targetID)

	// Set an override
	err := cache.SetOverride(ctx, testOverride, 5*time.Second)
	require.NoError(t, err)

	// Delete it
	err = cache.DeleteOverride(ctx, flagKey, targetType, targetID)
	require.NoError(t, err)

	// Verify it's gone
	override, err := cache.GetOverride(ctx, flagKey, targetType, targetID)
	require.NoError(t, err)
	assert.Nil(t, override)
}

func TestInMemoryFeatureFlagCache_InvalidateAll(t *testing.T) {
	cache := NewInMemoryFeatureFlagCache()

	ctx := context.Background()

	// Set multiple flags and overrides
	flag1 := createTestFlag("flag-1")
	flag2 := createTestFlag("flag-2")
	override1 := createTestOverride("flag-1", featureflag.OverrideTargetTypeUser, uuid.New())
	override2 := createTestOverride("flag-2", featureflag.OverrideTargetTypeTenant, uuid.New())

	require.NoError(t, cache.Set(ctx, "flag-1", flag1, 5*time.Second))
	require.NoError(t, cache.Set(ctx, "flag-2", flag2, 5*time.Second))
	require.NoError(t, cache.SetOverride(ctx, override1, 5*time.Second))
	require.NoError(t, cache.SetOverride(ctx, override2, 5*time.Second))

	// Verify they're there
	flags, overrides := cache.Count()
	assert.Equal(t, 2, flags)
	assert.Equal(t, 2, overrides)

	// Invalidate all
	err := cache.InvalidateAll(ctx)
	require.NoError(t, err)

	// Verify all are gone
	flags, overrides = cache.Count()
	assert.Equal(t, 0, flags)
	assert.Equal(t, 0, overrides)
}

func TestInMemoryFeatureFlagCache_Stats(t *testing.T) {
	cache := NewInMemoryFeatureFlagCache()

	ctx := context.Background()
	key := "test-flag"
	testFlag := createTestFlag(key)

	// Initial stats should be zero
	hits, misses := cache.GetStats()
	assert.Equal(t, int64(0), hits)
	assert.Equal(t, int64(0), misses)

	// Cache miss
	_, _ = cache.Get(ctx, key)
	hits, misses = cache.GetStats()
	assert.Equal(t, int64(0), hits)
	assert.Equal(t, int64(1), misses)

	// Set flag
	require.NoError(t, cache.Set(ctx, key, testFlag, 5*time.Second))

	// Cache hit
	_, _ = cache.Get(ctx, key)
	hits, misses = cache.GetStats()
	assert.Equal(t, int64(1), hits)
	assert.Equal(t, int64(1), misses)

	// Reset stats
	cache.ResetStats()
	hits, misses = cache.GetStats()
	assert.Equal(t, int64(0), hits)
	assert.Equal(t, int64(0), misses)
}

func TestInMemoryFeatureFlagCache_DefaultTTL(t *testing.T) {
	config := featureflag.CacheConfig{
		L1TTL: 100 * time.Millisecond,
	}
	cache := NewInMemoryFeatureFlagCache(WithInMemoryConfig(config))

	ctx := context.Background()
	key := "test-flag"
	testFlag := createTestFlag(key)

	// Set with TTL=0 (should use default)
	err := cache.Set(ctx, key, testFlag, 0)
	require.NoError(t, err)

	// Verify it's there
	flag, err := cache.Get(ctx, key)
	require.NoError(t, err)
	require.NotNil(t, flag)

	// Wait for default TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Verify it's expired
	flag, err = cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Nil(t, flag)
}

func TestInMemoryFeatureFlagCache_Close(t *testing.T) {
	cache := NewInMemoryFeatureFlagCache()

	// Close should return nil
	err := cache.Close()
	require.NoError(t, err)

	// Close again should be safe (idempotent)
	err = cache.Close()
	require.NoError(t, err)
}

// Helper functions

func createTestFlag(key string) *featureflag.FeatureFlag {
	flag, _ := featureflag.NewFeatureFlag(
		key,
		"Test Flag",
		featureflag.FlagTypeBoolean,
		featureflag.NewBooleanFlagValue(true),
		nil,
	)
	return flag
}

func createTestOverride(flagKey string, targetType featureflag.OverrideTargetType, targetID uuid.UUID) *featureflag.FlagOverride {
	override, _ := featureflag.NewFlagOverride(
		flagKey,
		targetType,
		targetID,
		featureflag.NewBooleanFlagValue(false),
		"test override",
		nil,
		nil,
	)
	return override
}

// Mock implementations for testing

type mockFeatureFlagRepository struct {
	flags map[string]*featureflag.FeatureFlag
}

func newMockFeatureFlagRepository() *mockFeatureFlagRepository {
	return &mockFeatureFlagRepository{
		flags: make(map[string]*featureflag.FeatureFlag),
	}
}

func (m *mockFeatureFlagRepository) Create(ctx context.Context, flag *featureflag.FeatureFlag) error {
	m.flags[flag.GetKey()] = flag
	return nil
}

func (m *mockFeatureFlagRepository) Update(ctx context.Context, flag *featureflag.FeatureFlag) error {
	m.flags[flag.GetKey()] = flag
	return nil
}

func (m *mockFeatureFlagRepository) FindByKey(ctx context.Context, key string) (*featureflag.FeatureFlag, error) {
	if flag, ok := m.flags[key]; ok {
		return flag, nil
	}
	return nil, shared.NewDomainError("NOT_FOUND", "Flag not found")
}

func (m *mockFeatureFlagRepository) FindByID(ctx context.Context, id uuid.UUID) (*featureflag.FeatureFlag, error) {
	for _, flag := range m.flags {
		if flag.GetID() == id {
			return flag, nil
		}
	}
	return nil, shared.NewDomainError("NOT_FOUND", "Flag not found")
}

func (m *mockFeatureFlagRepository) FindAll(ctx context.Context, filter shared.Filter) ([]featureflag.FeatureFlag, error) {
	result := make([]featureflag.FeatureFlag, 0, len(m.flags))
	for _, flag := range m.flags {
		result = append(result, *flag)
	}
	return result, nil
}

func (m *mockFeatureFlagRepository) FindByStatus(ctx context.Context, status featureflag.FlagStatus, filter shared.Filter) ([]featureflag.FeatureFlag, error) {
	return m.FindAll(ctx, filter)
}

func (m *mockFeatureFlagRepository) FindByTags(ctx context.Context, tags []string, filter shared.Filter) ([]featureflag.FeatureFlag, error) {
	return m.FindAll(ctx, filter)
}

func (m *mockFeatureFlagRepository) FindByType(ctx context.Context, flagType featureflag.FlagType, filter shared.Filter) ([]featureflag.FeatureFlag, error) {
	return m.FindAll(ctx, filter)
}

func (m *mockFeatureFlagRepository) FindEnabled(ctx context.Context, filter shared.Filter) ([]featureflag.FeatureFlag, error) {
	result := make([]featureflag.FeatureFlag, 0)
	for _, flag := range m.flags {
		if flag.IsEnabled() {
			result = append(result, *flag)
		}
	}
	return result, nil
}

func (m *mockFeatureFlagRepository) Delete(ctx context.Context, key string) error {
	delete(m.flags, key)
	return nil
}

func (m *mockFeatureFlagRepository) ExistsByKey(ctx context.Context, key string) (bool, error) {
	_, ok := m.flags[key]
	return ok, nil
}

func (m *mockFeatureFlagRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	return int64(len(m.flags)), nil
}

func (m *mockFeatureFlagRepository) CountByStatus(ctx context.Context, status featureflag.FlagStatus) (int64, error) {
	return int64(len(m.flags)), nil
}

type mockFlagOverrideRepository struct {
	overrides map[string]*featureflag.FlagOverride
}

func newMockFlagOverrideRepository() *mockFlagOverrideRepository {
	return &mockFlagOverrideRepository{
		overrides: make(map[string]*featureflag.FlagOverride),
	}
}

func (m *mockFlagOverrideRepository) makeKey(flagKey string, targetType featureflag.OverrideTargetType, targetID uuid.UUID) string {
	return flagKey + ":" + string(targetType) + ":" + targetID.String()
}

func (m *mockFlagOverrideRepository) Create(ctx context.Context, override *featureflag.FlagOverride) error {
	key := m.makeKey(override.FlagKey, override.TargetType, override.TargetID)
	m.overrides[key] = override
	return nil
}

func (m *mockFlagOverrideRepository) Update(ctx context.Context, override *featureflag.FlagOverride) error {
	return m.Create(ctx, override)
}

func (m *mockFlagOverrideRepository) FindByID(ctx context.Context, id uuid.UUID) (*featureflag.FlagOverride, error) {
	for _, override := range m.overrides {
		if override.GetID() == id {
			return override, nil
		}
	}
	return nil, shared.NewDomainError("NOT_FOUND", "Override not found")
}

func (m *mockFlagOverrideRepository) FindByFlagKey(ctx context.Context, flagKey string, filter shared.Filter) ([]featureflag.FlagOverride, error) {
	result := make([]featureflag.FlagOverride, 0)
	for _, override := range m.overrides {
		if override.FlagKey == flagKey {
			result = append(result, *override)
		}
	}
	return result, nil
}

func (m *mockFlagOverrideRepository) FindByTarget(ctx context.Context, targetType featureflag.OverrideTargetType, targetID uuid.UUID, filter shared.Filter) ([]featureflag.FlagOverride, error) {
	result := make([]featureflag.FlagOverride, 0)
	for _, override := range m.overrides {
		if override.TargetType == targetType && override.TargetID == targetID {
			result = append(result, *override)
		}
	}
	return result, nil
}

func (m *mockFlagOverrideRepository) FindForEvaluation(ctx context.Context, flagKey string, tenantID, userID *uuid.UUID) (*featureflag.FlagOverride, error) {
	// User override has priority
	if userID != nil {
		key := m.makeKey(flagKey, featureflag.OverrideTargetTypeUser, *userID)
		if override, ok := m.overrides[key]; ok && override.IsActive() {
			return override, nil
		}
	}
	if tenantID != nil {
		key := m.makeKey(flagKey, featureflag.OverrideTargetTypeTenant, *tenantID)
		if override, ok := m.overrides[key]; ok && override.IsActive() {
			return override, nil
		}
	}
	return nil, nil
}

func (m *mockFlagOverrideRepository) FindByFlagKeyAndTarget(ctx context.Context, flagKey string, targetType featureflag.OverrideTargetType, targetID uuid.UUID) (*featureflag.FlagOverride, error) {
	key := m.makeKey(flagKey, targetType, targetID)
	if override, ok := m.overrides[key]; ok {
		return override, nil
	}
	return nil, shared.NewDomainError("NOT_FOUND", "Override not found")
}

func (m *mockFlagOverrideRepository) FindExpired(ctx context.Context, filter shared.Filter) ([]featureflag.FlagOverride, error) {
	return []featureflag.FlagOverride{}, nil
}

func (m *mockFlagOverrideRepository) FindActive(ctx context.Context, filter shared.Filter) ([]featureflag.FlagOverride, error) {
	result := make([]featureflag.FlagOverride, 0)
	for _, override := range m.overrides {
		if override.IsActive() {
			result = append(result, *override)
		}
	}
	return result, nil
}

func (m *mockFlagOverrideRepository) Delete(ctx context.Context, id uuid.UUID) error {
	for key, override := range m.overrides {
		if override.GetID() == id {
			delete(m.overrides, key)
			return nil
		}
	}
	return nil
}

func (m *mockFlagOverrideRepository) DeleteByFlagKey(ctx context.Context, flagKey string) (int64, error) {
	var count int64
	for key, override := range m.overrides {
		if override.FlagKey == flagKey {
			delete(m.overrides, key)
			count++
		}
	}
	return count, nil
}

func (m *mockFlagOverrideRepository) DeleteExpired(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *mockFlagOverrideRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	return int64(len(m.overrides)), nil
}

func (m *mockFlagOverrideRepository) CountByFlagKey(ctx context.Context, flagKey string) (int64, error) {
	var count int64
	for _, override := range m.overrides {
		if override.FlagKey == flagKey {
			count++
		}
	}
	return count, nil
}

var _ featureflag.FeatureFlagRepository = (*mockFeatureFlagRepository)(nil)
var _ featureflag.FlagOverrideRepository = (*mockFlagOverrideRepository)(nil)
