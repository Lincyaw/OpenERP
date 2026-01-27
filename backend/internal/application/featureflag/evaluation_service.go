package featureflag

import (
	"context"

	"github.com/erp/backend/internal/application/featureflag/dto"
	"github.com/erp/backend/internal/domain/featureflag"
	"github.com/erp/backend/internal/domain/shared"
	"go.uber.org/zap"
)

// EvaluatorInterface defines the interface for flag evaluation
// This allows switching between cached and non-cached evaluators
type EvaluatorInterface interface {
	Evaluate(ctx context.Context, flagKey string, evalCtx *featureflag.EvaluationContext) featureflag.EvaluationResult
	EvaluateBatch(ctx context.Context, flagKeys []string, evalCtx *featureflag.EvaluationContext) map[string]featureflag.EvaluationResult
	EvaluateAll(ctx context.Context, evalCtx *featureflag.EvaluationContext) (map[string]featureflag.EvaluationResult, error)
}

// EvaluationService handles feature flag evaluation operations
type EvaluationService struct {
	flagRepo     featureflag.FeatureFlagRepository
	overrideRepo featureflag.FlagOverrideRepository
	evaluator    EvaluatorInterface
	logger       *zap.Logger
}

// NewEvaluationService creates a new evaluation service without caching
func NewEvaluationService(
	flagRepo featureflag.FeatureFlagRepository,
	overrideRepo featureflag.FlagOverrideRepository,
	logger *zap.Logger,
) *EvaluationService {
	return &EvaluationService{
		flagRepo:     flagRepo,
		overrideRepo: overrideRepo,
		evaluator:    featureflag.NewEvaluator(flagRepo, overrideRepo),
		logger:       logger,
	}
}

// NewCachedEvaluationService creates a new evaluation service with caching
func NewCachedEvaluationService(
	flagRepo featureflag.FeatureFlagRepository,
	overrideRepo featureflag.FlagOverrideRepository,
	cache featureflag.FlagCache,
	logger *zap.Logger,
) *EvaluationService {
	cachedEvaluator := featureflag.NewCachedEvaluator(
		flagRepo,
		overrideRepo,
		cache,
		featureflag.WithCachedEvaluatorLogger(logger),
	)
	return &EvaluationService{
		flagRepo:     flagRepo,
		overrideRepo: overrideRepo,
		evaluator:    cachedEvaluator,
		logger:       logger,
	}
}

// NewCachedEvaluationServiceWithConfig creates a new evaluation service with caching and custom config
func NewCachedEvaluationServiceWithConfig(
	flagRepo featureflag.FeatureFlagRepository,
	overrideRepo featureflag.FlagOverrideRepository,
	cache featureflag.FlagCache,
	config featureflag.CacheConfig,
	logger *zap.Logger,
) *EvaluationService {
	cachedEvaluator := featureflag.NewCachedEvaluator(
		flagRepo,
		overrideRepo,
		cache,
		featureflag.WithCachedEvaluatorLogger(logger),
		featureflag.WithCachedEvaluatorConfig(config),
	)
	return &EvaluationService{
		flagRepo:     flagRepo,
		overrideRepo: overrideRepo,
		evaluator:    cachedEvaluator,
		logger:       logger,
	}
}

// Evaluate evaluates a single feature flag
func (s *EvaluationService) Evaluate(ctx context.Context, key string, evalCtxDTO dto.EvaluationContextDTO) (*dto.EvaluateFlagResponse, error) {
	s.logger.Debug("Evaluating feature flag",
		zap.String("key", key),
		zap.String("user_id", evalCtxDTO.UserID),
		zap.String("tenant_id", evalCtxDTO.TenantID))

	// Convert DTO to domain context
	evalCtx := evalCtxDTO.ToDomain()

	// Evaluate the flag
	result := s.evaluator.Evaluate(ctx, key, evalCtx)

	// Check for errors
	if result.HasError() {
		s.logger.Error("Flag evaluation error",
			zap.String("key", key),
			zap.Error(result.GetError()))
		return nil, shared.NewDomainError("EVALUATION_ERROR", "Failed to evaluate feature flag")
	}

	// Check for flag not found
	if result.Reason == featureflag.EvaluationReasonFlagNotFound {
		return nil, shared.NewDomainError("FLAG_NOT_FOUND", "Feature flag not found")
	}

	s.logger.Debug("Flag evaluation complete",
		zap.String("key", key),
		zap.Bool("enabled", result.Enabled),
		zap.String("reason", string(result.Reason)))

	return dto.ToEvaluateFlagResponse(result), nil
}

// BatchEvaluate evaluates multiple feature flags at once
func (s *EvaluationService) BatchEvaluate(ctx context.Context, keys []string, evalCtxDTO dto.EvaluationContextDTO) (*dto.BatchEvaluateResponse, error) {
	s.logger.Debug("Batch evaluating feature flags",
		zap.Int("count", len(keys)),
		zap.String("user_id", evalCtxDTO.UserID),
		zap.String("tenant_id", evalCtxDTO.TenantID))

	// Validate input
	if len(keys) == 0 {
		return nil, shared.NewDomainError("INVALID_REQUEST", "At least one flag key is required")
	}
	if len(keys) > 100 {
		return nil, shared.NewDomainError("INVALID_REQUEST", "Cannot evaluate more than 100 flags at once")
	}

	// Convert DTO to domain context
	evalCtx := evalCtxDTO.ToDomain()

	// Evaluate all flags
	results := s.evaluator.EvaluateBatch(ctx, keys, evalCtx)

	s.logger.Debug("Batch evaluation complete",
		zap.Int("count", len(results)))

	return dto.ToBatchEvaluateResponse(results), nil
}

// GetClientConfig returns all enabled flags for a client SDK
// This is optimized for client applications that need all flag values upfront
func (s *EvaluationService) GetClientConfig(ctx context.Context, evalCtxDTO dto.EvaluationContextDTO) (*dto.GetClientConfigResponse, error) {
	s.logger.Debug("Getting client config",
		zap.String("user_id", evalCtxDTO.UserID),
		zap.String("tenant_id", evalCtxDTO.TenantID))

	// Convert DTO to domain context
	evalCtx := evalCtxDTO.ToDomain()

	// Evaluate all enabled flags
	results, err := s.evaluator.EvaluateAll(ctx, evalCtx)
	if err != nil {
		s.logger.Error("Failed to evaluate all flags", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to get client config")
	}

	s.logger.Debug("Client config retrieved",
		zap.Int("flag_count", len(results)))

	return dto.ToGetClientConfigResponse(results), nil
}

// IsEnabled is a convenience method to check if a flag is enabled
func (s *EvaluationService) IsEnabled(ctx context.Context, key string, evalCtxDTO dto.EvaluationContextDTO) (bool, error) {
	s.logger.Debug("Checking if flag is enabled", zap.String("key", key))

	evalCtx := evalCtxDTO.ToDomain()
	result := s.evaluator.Evaluate(ctx, key, evalCtx)

	if result.HasError() {
		return false, shared.NewDomainError("EVALUATION_ERROR", "Failed to evaluate feature flag")
	}

	if result.Reason == featureflag.EvaluationReasonFlagNotFound {
		return false, shared.NewDomainError("FLAG_NOT_FOUND", "Feature flag not found")
	}

	return result.IsEnabled(), nil
}

// GetVariant is a convenience method to get the variant value for a flag
func (s *EvaluationService) GetVariant(ctx context.Context, key string, evalCtxDTO dto.EvaluationContextDTO) (string, error) {
	s.logger.Debug("Getting flag variant", zap.String("key", key))

	evalCtx := evalCtxDTO.ToDomain()
	result := s.evaluator.Evaluate(ctx, key, evalCtx)

	if result.HasError() {
		return "", shared.NewDomainError("EVALUATION_ERROR", "Failed to evaluate feature flag")
	}

	if result.Reason == featureflag.EvaluationReasonFlagNotFound {
		return "", shared.NewDomainError("FLAG_NOT_FOUND", "Feature flag not found")
	}

	return result.Variant, nil
}

// WarmupCache pre-populates the cache with all enabled flags
// This should be called at application startup for optimal performance
// Returns nil if the evaluator doesn't support caching
func (s *EvaluationService) WarmupCache(ctx context.Context) error {
	if cachedEval, ok := s.evaluator.(*featureflag.CachedEvaluator); ok {
		return cachedEval.WarmupCache(ctx)
	}
	return nil
}

// InvalidateFlag invalidates a specific flag in the cache
// Returns nil if the evaluator doesn't support caching
func (s *EvaluationService) InvalidateFlag(ctx context.Context, key string) error {
	if cachedEval, ok := s.evaluator.(*featureflag.CachedEvaluator); ok {
		return cachedEval.InvalidateFlag(ctx, key)
	}
	return nil
}

// GetCacheStats returns cache statistics if available
// Returns nil if the evaluator doesn't support caching
func (s *EvaluationService) GetCacheStats(ctx context.Context) *featureflag.CacheStats {
	if cachedEval, ok := s.evaluator.(*featureflag.CachedEvaluator); ok {
		cache := cachedEval.GetCache()
		if tieredCache, ok := cache.(featureflag.TieredFlagCache); ok {
			stats := tieredCache.GetCacheStats(ctx)
			return &stats
		}
	}
	return nil
}
