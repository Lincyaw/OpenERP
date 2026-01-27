package featureflag

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// CachedEvaluator wraps an Evaluator with caching capabilities.
// It follows a read-through caching pattern:
// 1. Check cache for flag/override
// 2. If cache miss, fetch from repository
// 3. Populate cache with fetched data
// 4. Evaluate using cached data
type CachedEvaluator struct {
	flagRepo     FeatureFlagRepository
	overrideRepo FlagOverrideRepository
	cache        FlagCache
	logger       *zap.Logger
	config       CacheConfig
}

// CachedEvaluatorOption is a functional option for configuring the cached evaluator
type CachedEvaluatorOption func(*CachedEvaluator)

// WithCachedEvaluatorLogger sets the logger
func WithCachedEvaluatorLogger(logger *zap.Logger) CachedEvaluatorOption {
	return func(e *CachedEvaluator) {
		e.logger = logger
	}
}

// WithCachedEvaluatorConfig sets the cache config
func WithCachedEvaluatorConfig(config CacheConfig) CachedEvaluatorOption {
	return func(e *CachedEvaluator) {
		e.config = config
	}
}

// NewCachedEvaluator creates a new cached evaluator
func NewCachedEvaluator(
	flagRepo FeatureFlagRepository,
	overrideRepo FlagOverrideRepository,
	cache FlagCache,
	opts ...CachedEvaluatorOption,
) *CachedEvaluator {
	e := &CachedEvaluator{
		flagRepo:     flagRepo,
		overrideRepo: overrideRepo,
		cache:        cache,
		logger:       zap.NewNop(),
		config:       DefaultCacheConfig(),
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// Evaluate evaluates a single feature flag for the given context with caching
func (e *CachedEvaluator) Evaluate(ctx context.Context, flagKey string, evalCtx *EvaluationContext) EvaluationResult {
	// Get the feature flag (with caching)
	flag, err := e.getFlag(ctx, flagKey)
	if err != nil {
		if domainErr, ok := err.(*shared.DomainError); ok && domainErr.Code == "NOT_FOUND" {
			return NewFlagNotFoundResult(flagKey)
		}
		return NewErrorResult(flagKey, err)
	}

	// Get overrides (with caching)
	var userOverride, tenantOverride *FlagOverride

	if evalCtx != nil && evalCtx.HasUser() {
		userID, err := uuid.Parse(evalCtx.UserID)
		if err == nil {
			userOverride, _ = e.getOverride(ctx, flagKey, OverrideTargetTypeUser, userID)
		}
	}

	if evalCtx != nil && evalCtx.HasTenant() {
		tenantID, err := uuid.Parse(evalCtx.TenantID)
		if err == nil {
			tenantOverride, _ = e.getOverride(ctx, flagKey, OverrideTargetTypeTenant, tenantID)
		}
	}

	// Use PureEvaluator for evaluation with pre-fetched data
	pureEval := NewPureEvaluator()
	return pureEval.Evaluate(flag, evalCtx, userOverride, tenantOverride)
}

// EvaluateBatch evaluates multiple feature flags at once with caching
func (e *CachedEvaluator) EvaluateBatch(ctx context.Context, flagKeys []string, evalCtx *EvaluationContext) map[string]EvaluationResult {
	results := make(map[string]EvaluationResult, len(flagKeys))

	for _, key := range flagKeys {
		results[key] = e.Evaluate(ctx, key, evalCtx)
	}

	return results
}

// EvaluateAll evaluates all enabled feature flags with caching
func (e *CachedEvaluator) EvaluateAll(ctx context.Context, evalCtx *EvaluationContext) (map[string]EvaluationResult, error) {
	// For EvaluateAll, we still need to query the repository
	// as we don't know which flags are enabled without checking the database
	filter := shared.Filter{
		Page:     1,
		PageSize: 1000,
	}
	flags, err := e.flagRepo.FindEnabled(ctx, filter)
	if err != nil {
		return nil, err
	}

	results := make(map[string]EvaluationResult, len(flags))

	// Pre-fetch overrides if we have user/tenant context
	var userOverride, tenantOverride *FlagOverride

	for i := range flags {
		flag := &flags[i]
		flagKey := flag.GetKey()

		// Cache the flag for future single evaluations
		if e.cache != nil {
			if err := e.cache.Set(ctx, flagKey, flag, e.config.FlagTTL); err != nil {
				e.logger.Warn("Failed to cache flag during EvaluateAll",
					zap.String("key", flagKey),
					zap.Error(err))
			}
		}

		// Get overrides for this flag
		userOverride = nil
		tenantOverride = nil

		if evalCtx != nil && evalCtx.HasUser() {
			userID, err := uuid.Parse(evalCtx.UserID)
			if err == nil {
				userOverride, _ = e.getOverride(ctx, flagKey, OverrideTargetTypeUser, userID)
			}
		}

		if evalCtx != nil && evalCtx.HasTenant() {
			tenantID, err := uuid.Parse(evalCtx.TenantID)
			if err == nil {
				tenantOverride, _ = e.getOverride(ctx, flagKey, OverrideTargetTypeTenant, tenantID)
			}
		}

		// Evaluate
		pureEval := NewPureEvaluator()
		results[flagKey] = pureEval.Evaluate(flag, evalCtx, userOverride, tenantOverride)
	}

	return results, nil
}

// getFlag retrieves a flag from cache or repository
func (e *CachedEvaluator) getFlag(ctx context.Context, key string) (*FeatureFlag, error) {
	// Try cache first
	if e.cache != nil {
		flag, err := e.cache.Get(ctx, key)
		if err != nil {
			e.logger.Warn("Cache get error",
				zap.String("key", key),
				zap.Error(err))
		} else if flag != nil {
			e.logger.Debug("Cache hit for flag", zap.String("key", key))
			return flag, nil
		}
		e.logger.Debug("Cache miss for flag", zap.String("key", key))
	}

	// Fetch from repository
	flag, err := e.flagRepo.FindByKey(ctx, key)
	if err != nil {
		return nil, err
	}

	// Populate cache
	if e.cache != nil && flag != nil {
		if err := e.cache.Set(ctx, key, flag, e.config.FlagTTL); err != nil {
			e.logger.Warn("Failed to cache flag",
				zap.String("key", key),
				zap.Error(err))
		}
	}

	return flag, nil
}

// getOverride retrieves an override from cache or repository
func (e *CachedEvaluator) getOverride(ctx context.Context, flagKey string, targetType OverrideTargetType, targetID uuid.UUID) (*FlagOverride, error) {
	// Try cache first
	if e.cache != nil {
		override, err := e.cache.GetOverride(ctx, flagKey, targetType, targetID)
		if err != nil {
			e.logger.Warn("Cache get error for override",
				zap.String("flag_key", flagKey),
				zap.String("target_type", string(targetType)),
				zap.Error(err))
		} else if override != nil {
			e.logger.Debug("Cache hit for override",
				zap.String("flag_key", flagKey),
				zap.String("target_type", string(targetType)))
			return override, nil
		}
		e.logger.Debug("Cache miss for override",
			zap.String("flag_key", flagKey),
			zap.String("target_type", string(targetType)))
	}

	// Fetch from repository
	if e.overrideRepo == nil {
		return nil, nil
	}

	override, err := e.overrideRepo.FindByFlagKeyAndTarget(ctx, flagKey, targetType, targetID)
	if err != nil {
		// Not found is not an error for overrides
		if domainErr, ok := err.(*shared.DomainError); ok && domainErr.Code == "NOT_FOUND" {
			return nil, nil
		}
		return nil, err
	}

	// Populate cache
	if e.cache != nil && override != nil {
		if err := e.cache.SetOverride(ctx, override, e.config.OverrideTTL); err != nil {
			e.logger.Warn("Failed to cache override",
				zap.String("flag_key", flagKey),
				zap.Error(err))
		}
	}

	return override, nil
}

// IsEnabled is a convenience method to check if a flag is enabled
func (e *CachedEvaluator) IsEnabled(ctx context.Context, flagKey string, evalCtx *EvaluationContext) bool {
	result := e.Evaluate(ctx, flagKey, evalCtx)
	return result.IsEnabled()
}

// GetVariant is a convenience method to get the variant value
func (e *CachedEvaluator) GetVariant(ctx context.Context, flagKey string, evalCtx *EvaluationContext) string {
	result := e.Evaluate(ctx, flagKey, evalCtx)
	return result.Variant
}

// InvalidateFlag invalidates a flag in the cache
func (e *CachedEvaluator) InvalidateFlag(ctx context.Context, key string) error {
	if e.cache == nil {
		return nil
	}
	return e.cache.Delete(ctx, key)
}

// InvalidateOverride invalidates an override in the cache
func (e *CachedEvaluator) InvalidateOverride(ctx context.Context, flagKey string, targetType OverrideTargetType, targetID uuid.UUID) error {
	if e.cache == nil {
		return nil
	}
	return e.cache.DeleteOverride(ctx, flagKey, targetType, targetID)
}

// WarmupCache pre-populates the cache with all enabled flags
// This should be called at application startup for optimal performance
func (e *CachedEvaluator) WarmupCache(ctx context.Context) error {
	if e.cache == nil {
		return nil
	}

	e.logger.Info("Starting cache warmup")
	start := time.Now()

	filter := shared.Filter{
		Page:     1,
		PageSize: 1000,
	}
	flags, err := e.flagRepo.FindEnabled(ctx, filter)
	if err != nil {
		return err
	}

	for i := range flags {
		flag := &flags[i]
		if err := e.cache.Set(ctx, flag.GetKey(), flag, e.config.FlagTTL); err != nil {
			e.logger.Warn("Failed to cache flag during warmup",
				zap.String("key", flag.GetKey()),
				zap.Error(err))
		}
	}

	e.logger.Info("Cache warmup completed",
		zap.Int("flags_cached", len(flags)),
		zap.Duration("duration", time.Since(start)))

	return nil
}

// GetCache returns the underlying cache (for testing/monitoring)
func (e *CachedEvaluator) GetCache() FlagCache {
	return e.cache
}
