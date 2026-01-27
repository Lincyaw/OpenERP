package middleware

import (
	"context"
	"maps"
	"slices"
	"time"

	"github.com/erp/backend/internal/domain/featureflag"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Feature flag context keys
const (
	// FeatureFlagContextKey is the key for storing evaluated feature flags in context
	FeatureFlagContextKey = "feature_flags"
	// FeatureFlagEvalContextKey is the key for storing the evaluation context
	FeatureFlagEvalContextKey = "feature_flag_eval_context"
	// FeatureFlagEvalDurationKey is the key for storing evaluation duration
	FeatureFlagEvalDurationKey = "feature_flag_eval_duration"
)

// FlagValue represents an evaluated flag value
type FlagValue struct {
	Enabled bool   `json:"enabled"`
	Variant string `json:"variant,omitempty"`
}

// FeatureFlagEvaluator defines the interface for evaluating feature flags
type FeatureFlagEvaluator interface {
	EvaluateBatch(ctx context.Context, flagKeys []string, evalCtx *featureflag.EvaluationContext) map[string]featureflag.EvaluationResult
}

// FeatureFlagMiddlewareConfig holds configuration for feature flag middleware
type FeatureFlagMiddlewareConfig struct {
	// Evaluator is the feature flag evaluator (required)
	Evaluator FeatureFlagEvaluator
	// PreloadKeys are the feature flag keys to pre-evaluate for each request
	PreloadKeys []string
	// SkipPaths are paths that don't need feature flag evaluation
	SkipPaths []string
	// Logger for middleware logging
	Logger *zap.Logger
	// Environment is the deployment environment (e.g., "development", "production")
	Environment string
}

// DefaultFeatureFlagConfig returns default feature flag middleware configuration
func DefaultFeatureFlagConfig() FeatureFlagMiddlewareConfig {
	return FeatureFlagMiddlewareConfig{
		Evaluator:   nil,
		PreloadKeys: []string{},
		SkipPaths: []string{
			"/health",
			"/healthz",
			"/ready",
			"/metrics",
			"/api/v1/health",
		},
		Logger:      nil,
		Environment: "",
	}
}

// FeatureFlagMiddleware creates a middleware that pre-evaluates feature flags
// and injects them into the request context
func FeatureFlagMiddleware(evaluator FeatureFlagEvaluator, preloadKeys []string) gin.HandlerFunc {
	cfg := DefaultFeatureFlagConfig()
	cfg.Evaluator = evaluator
	cfg.PreloadKeys = preloadKeys
	return FeatureFlagMiddlewareWithConfig(cfg)
}

// FeatureFlagMiddlewareWithConfig creates feature flag middleware with custom configuration
func FeatureFlagMiddlewareWithConfig(cfg FeatureFlagMiddlewareConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip paths that don't need feature flags
		path := c.Request.URL.Path
		if slices.Contains(cfg.SkipPaths, path) {
			c.Next()
			return
		}

		// Skip if no evaluator or no preload keys configured
		if cfg.Evaluator == nil || len(cfg.PreloadKeys) == 0 {
			c.Next()
			return
		}

		// Build evaluation context from auth middleware values
		evalCtx := buildEvaluationContext(c, cfg.Environment)

		// Store evaluation context for downstream use
		c.Set(FeatureFlagEvalContextKey, evalCtx)

		// Evaluate feature flags
		start := time.Now()
		results := cfg.Evaluator.EvaluateBatch(c.Request.Context(), cfg.PreloadKeys, evalCtx)
		evalDuration := time.Since(start)

		// Store evaluation duration for metrics
		c.Set(FeatureFlagEvalDurationKey, evalDuration)

		// Convert results to FlagValue map
		flagValues := make(map[string]FlagValue, len(results))
		for key, result := range results {
			flagValues[key] = FlagValue{
				Enabled: result.Enabled,
				Variant: result.Variant,
			}
		}

		// Store flag values in context
		c.Set(FeatureFlagContextKey, flagValues)

		// Log evaluation if logger is provided
		if cfg.Logger != nil {
			cfg.Logger.Debug("Feature flags evaluated",
				zap.Int("count", len(cfg.PreloadKeys)),
				zap.Duration("duration", evalDuration),
				zap.String("user_id", evalCtx.UserID),
				zap.String("tenant_id", evalCtx.TenantID),
			)
		}

		c.Next()
	}
}

// buildEvaluationContext builds an EvaluationContext from gin.Context
// It extracts user_id, tenant_id, and role from JWT middleware values
func buildEvaluationContext(c *gin.Context, environment string) *featureflag.EvaluationContext {
	evalCtx := featureflag.NewEvaluationContext()

	// Extract tenant_id from JWT middleware
	if tenantID, exists := c.Get(JWTTenantIDKey); exists {
		if tid, ok := tenantID.(string); ok && tid != "" {
			evalCtx = evalCtx.WithTenant(tid)
		}
	}

	// Extract user_id from JWT middleware
	if userID, exists := c.Get(JWTUserIDKey); exists {
		if uid, ok := userID.(string); ok && uid != "" {
			evalCtx = evalCtx.WithUser(uid)
		}
	}

	// Extract role from JWT middleware (first role if multiple)
	if roleIDs, exists := c.Get(JWTRoleIDsKey); exists {
		if roles, ok := roleIDs.([]string); ok && len(roles) > 0 {
			evalCtx = evalCtx.WithUserRole(roles[0])
		}
	}

	// Extract request ID
	if requestID, exists := c.Get("request_id"); exists {
		if rid, ok := requestID.(string); ok && rid != "" {
			evalCtx = evalCtx.WithRequestID(rid)
		}
	}

	// Set environment
	if environment != "" {
		evalCtx = evalCtx.WithEnvironment(environment)
	}

	return evalCtx
}

// GetFeatureFlag retrieves a boolean feature flag value from gin.Context
// Returns false if the flag is not found or is not enabled
func GetFeatureFlag(c *gin.Context, key string) bool {
	flags, exists := c.Get(FeatureFlagContextKey)
	if !exists {
		return false
	}

	flagValues, ok := flags.(map[string]FlagValue)
	if !ok {
		return false
	}

	if value, found := flagValues[key]; found {
		return value.Enabled
	}

	return false
}

// GetFeatureVariant retrieves a feature flag variant from gin.Context
// Returns empty string if the flag is not found or has no variant
func GetFeatureVariant(c *gin.Context, key string) string {
	flags, exists := c.Get(FeatureFlagContextKey)
	if !exists {
		return ""
	}

	flagValues, ok := flags.(map[string]FlagValue)
	if !ok {
		return ""
	}

	if value, found := flagValues[key]; found {
		return value.Variant
	}

	return ""
}

// GetAllFlags retrieves all pre-evaluated feature flags from gin.Context
// Returns an empty map if no flags are found
func GetAllFlags(c *gin.Context) map[string]FlagValue {
	flags, exists := c.Get(FeatureFlagContextKey)
	if !exists {
		return make(map[string]FlagValue)
	}

	flagValues, ok := flags.(map[string]FlagValue)
	if !ok {
		return make(map[string]FlagValue)
	}

	// Return a copy to prevent modification
	result := make(map[string]FlagValue, len(flagValues))
	maps.Copy(result, flagValues)

	return result
}

// GetFeatureFlagEvalContext retrieves the evaluation context from gin.Context
// Returns nil if not found
func GetFeatureFlagEvalContext(c *gin.Context) *featureflag.EvaluationContext {
	evalCtx, exists := c.Get(FeatureFlagEvalContextKey)
	if !exists {
		return nil
	}

	if ctx, ok := evalCtx.(*featureflag.EvaluationContext); ok {
		return ctx
	}

	return nil
}

// GetFeatureFlagEvalDuration retrieves the evaluation duration from gin.Context
// Returns 0 if not found
func GetFeatureFlagEvalDuration(c *gin.Context) time.Duration {
	duration, exists := c.Get(FeatureFlagEvalDurationKey)
	if !exists {
		return 0
	}

	if d, ok := duration.(time.Duration); ok {
		return d
	}

	return 0
}

// MustGetFeatureFlag retrieves a feature flag value or panics if flags not available
// Use this only when you're certain feature flags middleware has run
func MustGetFeatureFlag(c *gin.Context, key string) bool {
	flags, exists := c.Get(FeatureFlagContextKey)
	if !exists {
		panic("feature flags not found in context - ensure FeatureFlagMiddleware is configured")
	}

	flagValues, ok := flags.(map[string]FlagValue)
	if !ok {
		panic("invalid feature flags type in context")
	}

	if value, found := flagValues[key]; found {
		return value.Enabled
	}

	return false
}

// WithFeatureFlag is a helper that wraps a handler and only executes it
// if the specified feature flag is enabled
func WithFeatureFlag(key string, handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !GetFeatureFlag(c, key) {
			c.AbortWithStatusJSON(404, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "FEATURE_NOT_AVAILABLE",
					"message": "This feature is not currently available",
				},
			})
			return
		}
		handler(c)
	}
}
