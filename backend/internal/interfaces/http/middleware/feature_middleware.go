package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Feature middleware context keys
const (
	// TenantPlanKey is the key for storing tenant plan in context
	TenantPlanKey = "tenant_plan"
	// FeatureCheckCachePrefix is the Redis key prefix for feature check cache
	FeatureCheckCachePrefix = "feature_check:"
)

// FeatureChecker defines the interface for checking plan features
type FeatureChecker interface {
	// HasFeature checks if a plan has a specific feature enabled
	HasFeature(ctx context.Context, planID identity.TenantPlan, featureKey identity.FeatureKey) (bool, error)
	// GetFeatureLimit returns the limit for a feature in a plan (nil if unlimited)
	GetFeatureLimit(ctx context.Context, planID identity.TenantPlan, featureKey identity.FeatureKey) (*int, error)
}

// TenantPlanProvider defines the interface for getting tenant plan
type TenantPlanProvider interface {
	// GetTenantPlan returns the plan for a tenant
	GetTenantPlan(ctx context.Context, tenantID string) (identity.TenantPlan, error)
}

// FeatureCache defines the interface for caching feature check results
type FeatureCache interface {
	// Get retrieves a cached feature check result
	Get(ctx context.Context, key string) (bool, bool, error) // value, found, error
	// Set stores a feature check result in cache
	Set(ctx context.Context, key string, value bool, ttl time.Duration) error
	// Delete removes a cached feature check result
	Delete(ctx context.Context, key string) error
}

// FeatureMiddlewareConfig holds configuration for feature middleware
type FeatureMiddlewareConfig struct {
	// FeatureChecker is required for checking plan features
	FeatureChecker FeatureChecker
	// TenantPlanProvider is required for getting tenant plan
	TenantPlanProvider TenantPlanProvider
	// Cache is optional for caching feature check results
	Cache FeatureCache
	// CacheTTL is the TTL for cached feature check results (default: 5 minutes)
	CacheTTL time.Duration
	// Logger for middleware logging
	Logger *zap.Logger
	// OnDenied is called when feature access is denied (optional)
	OnDenied func(c *gin.Context, featureKey identity.FeatureKey, tenantPlan identity.TenantPlan)
}

// DefaultFeatureMiddlewareConfig returns default feature middleware configuration
func DefaultFeatureMiddlewareConfig() FeatureMiddlewareConfig {
	return FeatureMiddlewareConfig{
		FeatureChecker:     nil,
		TenantPlanProvider: nil,
		Cache:              nil,
		CacheTTL:           5 * time.Minute,
		Logger:             nil,
		OnDenied:           nil,
	}
}

// RequireFeature creates middleware that requires a specific feature to be enabled
// for the tenant's subscription plan.
// Panics if featureKey is not a valid feature key (fail fast at startup).
func RequireFeature(featureKey identity.FeatureKey) gin.HandlerFunc {
	return RequireFeatureWithConfig(featureKey, DefaultFeatureMiddlewareConfig())
}

// RequireFeatureWithConfig creates feature middleware with custom configuration.
// Panics if featureKey is not a valid feature key (fail fast at startup).
func RequireFeatureWithConfig(featureKey identity.FeatureKey, cfg FeatureMiddlewareConfig) gin.HandlerFunc {
	// Validate feature key at middleware creation time (fail fast)
	if !identity.IsValidFeatureKey(featureKey) {
		panic(fmt.Sprintf("invalid feature key: %s", featureKey))
	}

	return func(c *gin.Context) {
		// Get tenant ID from JWT context
		tenantID := GetJWTTenantID(c)
		if tenantID == "" {
			handleFeatureDenied(c, cfg, featureKey, "", "No tenant context found")
			return
		}

		// Get tenant plan
		tenantPlan, err := getTenantPlan(c, cfg, tenantID)
		if err != nil {
			if cfg.Logger != nil {
				cfg.Logger.Error("Failed to get tenant plan",
					zap.String("tenant_id", tenantID),
					zap.Error(err),
				)
			}
			handleFeatureDenied(c, cfg, featureKey, "", "Failed to determine subscription plan")
			return
		}

		// Store tenant plan in context for downstream use
		c.Set(TenantPlanKey, tenantPlan)

		// Check if feature is enabled for the plan
		hasFeature, err := checkFeature(c, cfg, tenantPlan, featureKey)
		if err != nil {
			if cfg.Logger != nil {
				cfg.Logger.Error("Failed to check feature",
					zap.String("tenant_id", tenantID),
					zap.String("plan", string(tenantPlan)),
					zap.String("feature", string(featureKey)),
					zap.Error(err),
				)
			}
			handleFeatureDenied(c, cfg, featureKey, tenantPlan, "Failed to check feature availability")
			return
		}

		if !hasFeature {
			if cfg.Logger != nil {
				cfg.Logger.Info("Feature access denied",
					zap.String("tenant_id", tenantID),
					zap.String("plan", string(tenantPlan)),
					zap.String("feature", string(featureKey)),
				)
			}
			handleFeatureDenied(c, cfg, featureKey, tenantPlan, "")
			return
		}

		if cfg.Logger != nil {
			cfg.Logger.Debug("Feature access granted",
				zap.String("tenant_id", tenantID),
				zap.String("plan", string(tenantPlan)),
				zap.String("feature", string(featureKey)),
			)
		}

		c.Next()
	}
}

// RequireAnyFeature creates middleware that requires any of the specified features
// to be enabled for the tenant's subscription plan
// RequireAnyFeature creates middleware that requires any of the specified features
// to be enabled for the tenant's subscription plan.
// Panics if any featureKey is not a valid feature key (fail fast at startup).
func RequireAnyFeature(featureKeys ...identity.FeatureKey) gin.HandlerFunc {
	return RequireAnyFeatureWithConfig(DefaultFeatureMiddlewareConfig(), featureKeys...)
}

// RequireAnyFeatureWithConfig creates middleware with custom configuration.
// Panics if any featureKey is not a valid feature key (fail fast at startup).
func RequireAnyFeatureWithConfig(cfg FeatureMiddlewareConfig, featureKeys ...identity.FeatureKey) gin.HandlerFunc {
	// Validate feature keys at middleware creation time (fail fast)
	for _, featureKey := range featureKeys {
		if !identity.IsValidFeatureKey(featureKey) {
			panic(fmt.Sprintf("invalid feature key: %s", featureKey))
		}
	}

	return func(c *gin.Context) {
		if len(featureKeys) == 0 {
			c.Next()
			return
		}

		// Get tenant ID from JWT context
		tenantID := GetJWTTenantID(c)
		if tenantID == "" {
			handleFeatureDenied(c, cfg, featureKeys[0], "", "No tenant context found")
			return
		}

		// Get tenant plan
		tenantPlan, err := getTenantPlan(c, cfg, tenantID)
		if err != nil {
			if cfg.Logger != nil {
				cfg.Logger.Error("Failed to get tenant plan",
					zap.String("tenant_id", tenantID),
					zap.Error(err),
				)
			}
			handleFeatureDenied(c, cfg, featureKeys[0], "", "Failed to determine subscription plan")
			return
		}

		// Store tenant plan in context for downstream use
		c.Set(TenantPlanKey, tenantPlan)

		// Check if any feature is enabled for the plan
		// Track errors to fail securely if all checks fail with errors
		var lastErr error
		var checkedAny bool

		for _, featureKey := range featureKeys {
			hasFeature, err := checkFeature(c, cfg, tenantPlan, featureKey)
			if err != nil {
				lastErr = err
				if cfg.Logger != nil {
					cfg.Logger.Error("Failed to check feature",
						zap.String("tenant_id", tenantID),
						zap.String("plan", string(tenantPlan)),
						zap.String("feature", string(featureKey)),
						zap.Error(err),
					)
				}
				continue // Try next feature
			}

			checkedAny = true
			if hasFeature {
				if cfg.Logger != nil {
					cfg.Logger.Debug("Feature access granted (any)",
						zap.String("tenant_id", tenantID),
						zap.String("plan", string(tenantPlan)),
						zap.String("feature", string(featureKey)),
					)
				}
				c.Next()
				return
			}
		}

		// If no features were successfully checked and there were errors, fail securely
		if !checkedAny && lastErr != nil {
			handleFeatureDenied(c, cfg, featureKeys[0], tenantPlan, "Failed to verify feature access")
			return
		}

		// No feature was enabled
		if cfg.Logger != nil {
			featureKeyStrings := make([]string, len(featureKeys))
			for i, k := range featureKeys {
				featureKeyStrings[i] = string(k)
			}
			cfg.Logger.Info("Feature access denied (none of required features enabled)",
				zap.String("tenant_id", tenantID),
				zap.String("plan", string(tenantPlan)),
				zap.Strings("features", featureKeyStrings),
			)
		}
		handleFeatureDeniedMultiple(c, cfg, featureKeys, tenantPlan, "")
	}
}

// RequireAllFeatures creates middleware that requires all of the specified features
// to be enabled for the tenant's subscription plan.
// Panics if any featureKey is not a valid feature key (fail fast at startup).
func RequireAllFeatures(featureKeys ...identity.FeatureKey) gin.HandlerFunc {
	return RequireAllFeaturesWithConfig(DefaultFeatureMiddlewareConfig(), featureKeys...)
}

// RequireAllFeaturesWithConfig creates middleware with custom configuration.
// Panics if any featureKey is not a valid feature key (fail fast at startup).
func RequireAllFeaturesWithConfig(cfg FeatureMiddlewareConfig, featureKeys ...identity.FeatureKey) gin.HandlerFunc {
	// Validate feature keys at middleware creation time (fail fast)
	for _, featureKey := range featureKeys {
		if !identity.IsValidFeatureKey(featureKey) {
			panic(fmt.Sprintf("invalid feature key: %s", featureKey))
		}
	}

	return func(c *gin.Context) {
		if len(featureKeys) == 0 {
			c.Next()
			return
		}

		// Get tenant ID from JWT context
		tenantID := GetJWTTenantID(c)
		if tenantID == "" {
			handleFeatureDenied(c, cfg, featureKeys[0], "", "No tenant context found")
			return
		}

		// Get tenant plan
		tenantPlan, err := getTenantPlan(c, cfg, tenantID)
		if err != nil {
			if cfg.Logger != nil {
				cfg.Logger.Error("Failed to get tenant plan",
					zap.String("tenant_id", tenantID),
					zap.Error(err),
				)
			}
			handleFeatureDenied(c, cfg, featureKeys[0], "", "Failed to determine subscription plan")
			return
		}

		// Store tenant plan in context for downstream use
		c.Set(TenantPlanKey, tenantPlan)

		// Check if all features are enabled for the plan
		// Distinguish between errors and disabled features for better error messages
		var missingFeatures []identity.FeatureKey
		var errorFeatures []identity.FeatureKey

		for _, featureKey := range featureKeys {
			hasFeature, err := checkFeature(c, cfg, tenantPlan, featureKey)
			if err != nil {
				if cfg.Logger != nil {
					cfg.Logger.Error("Failed to check feature",
						zap.String("tenant_id", tenantID),
						zap.String("plan", string(tenantPlan)),
						zap.String("feature", string(featureKey)),
						zap.Error(err),
					)
				}
				errorFeatures = append(errorFeatures, featureKey)
				continue
			}

			if !hasFeature {
				missingFeatures = append(missingFeatures, featureKey)
			}
		}

		// If there were errors checking features, fail securely
		if len(errorFeatures) > 0 {
			handleFeatureDenied(c, cfg, errorFeatures[0], tenantPlan, "Failed to verify feature access")
			return
		}

		if len(missingFeatures) > 0 {
			if cfg.Logger != nil {
				missingKeyStrings := make([]string, len(missingFeatures))
				for i, k := range missingFeatures {
					missingKeyStrings[i] = string(k)
				}
				cfg.Logger.Info("Feature access denied (missing required features)",
					zap.String("tenant_id", tenantID),
					zap.String("plan", string(tenantPlan)),
					zap.Strings("missing_features", missingKeyStrings),
				)
			}
			handleFeatureDeniedMultiple(c, cfg, missingFeatures, tenantPlan, "")
			return
		}

		if cfg.Logger != nil {
			cfg.Logger.Debug("Feature access granted (all)",
				zap.String("tenant_id", tenantID),
				zap.String("plan", string(tenantPlan)),
				zap.Int("feature_count", len(featureKeys)),
			)
		}

		c.Next()
	}
}

// getTenantPlan retrieves the tenant plan, using cache if available
func getTenantPlan(c *gin.Context, cfg FeatureMiddlewareConfig, tenantID string) (identity.TenantPlan, error) {
	// Check if plan is already in context (set by previous middleware)
	if plan, exists := c.Get(TenantPlanKey); exists {
		if tenantPlan, ok := plan.(identity.TenantPlan); ok {
			return tenantPlan, nil
		}
	}

	// If no provider is configured, use default (free) plan
	if cfg.TenantPlanProvider == nil {
		return identity.TenantPlanFree, nil
	}

	// Get plan from provider
	return cfg.TenantPlanProvider.GetTenantPlan(c.Request.Context(), tenantID)
}

// checkFeature checks if a feature is enabled for a plan, using cache if available
func checkFeature(c *gin.Context, cfg FeatureMiddlewareConfig, tenantPlan identity.TenantPlan, featureKey identity.FeatureKey) (bool, error) {
	ctx := c.Request.Context()

	// Build cache key
	cacheKey := buildFeatureCacheKey(tenantPlan, featureKey)

	// Check cache first
	if cfg.Cache != nil {
		if value, found, err := cfg.Cache.Get(ctx, cacheKey); err == nil && found {
			return value, nil
		}
	}

	// Check feature using checker or default
	var hasFeature bool
	if cfg.FeatureChecker != nil {
		var err error
		hasFeature, err = cfg.FeatureChecker.HasFeature(ctx, tenantPlan, featureKey)
		if err != nil {
			return false, err
		}
	} else {
		// Use default plan features if no checker is configured
		hasFeature = identity.PlanHasFeature(tenantPlan, featureKey)
	}

	// Store in cache
	if cfg.Cache != nil {
		ttl := cfg.CacheTTL
		if ttl == 0 {
			ttl = 5 * time.Minute
		}
		// Ignore cache set errors - feature check still succeeded
		_ = cfg.Cache.Set(ctx, cacheKey, hasFeature, ttl)
	}

	return hasFeature, nil
}

// buildFeatureCacheKey builds a cache key for feature check.
// Note: Cache key is based on plan, not tenant, because features are plan-level.
// All tenants on the same plan have the same features enabled.
// If tenant-specific feature overrides are needed in the future, include tenantID in the key.
func buildFeatureCacheKey(tenantPlan identity.TenantPlan, featureKey identity.FeatureKey) string {
	return fmt.Sprintf("%s%s:%s", FeatureCheckCachePrefix, tenantPlan, featureKey)
}

// handleFeatureDenied handles feature access denied scenarios
func handleFeatureDenied(c *gin.Context, cfg FeatureMiddlewareConfig, featureKey identity.FeatureKey, tenantPlan identity.TenantPlan, customMessage string) {
	if cfg.OnDenied != nil {
		cfg.OnDenied(c, featureKey, tenantPlan)
		return
	}

	message := customMessage
	if message == "" {
		message = buildUpgradeMessage(featureKey, tenantPlan)
	}

	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "ERR_FEATURE_NOT_AVAILABLE",
			"message": message,
			"details": gin.H{
				"feature":      string(featureKey),
				"current_plan": string(tenantPlan),
				"upgrade_hint": "Please upgrade your subscription plan to access this feature",
			},
		},
	})
}

// handleFeatureDeniedMultiple handles feature access denied for multiple features
func handleFeatureDeniedMultiple(c *gin.Context, cfg FeatureMiddlewareConfig, featureKeys []identity.FeatureKey, tenantPlan identity.TenantPlan, customMessage string) {
	if len(featureKeys) == 1 {
		handleFeatureDenied(c, cfg, featureKeys[0], tenantPlan, customMessage)
		return
	}

	if cfg.OnDenied != nil {
		cfg.OnDenied(c, featureKeys[0], tenantPlan)
		return
	}

	featureKeyStrings := make([]string, len(featureKeys))
	for i, k := range featureKeys {
		featureKeyStrings[i] = string(k)
	}

	message := customMessage
	if message == "" {
		message = fmt.Sprintf("The following features are not available in your current plan (%s): %s",
			tenantPlan, strings.Join(featureKeyStrings, ", "))
	}

	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "ERR_FEATURES_NOT_AVAILABLE",
			"message": message,
			"details": gin.H{
				"features":     featureKeyStrings,
				"current_plan": string(tenantPlan),
				"upgrade_hint": "Please upgrade your subscription plan to access these features",
			},
		},
	})
}

// buildUpgradeMessage builds a user-friendly upgrade message
func buildUpgradeMessage(featureKey identity.FeatureKey, tenantPlan identity.TenantPlan) string {
	featureName := formatFeatureName(featureKey)

	if tenantPlan == "" {
		return fmt.Sprintf("The %s feature is not available. Please contact support.", featureName)
	}

	return fmt.Sprintf("The %s feature is not available in your current plan (%s). Please upgrade to access this feature.",
		featureName, tenantPlan)
}

// formatFeatureName converts a feature key to a human-readable name
func formatFeatureName(featureKey identity.FeatureKey) string {
	// Convert snake_case to Title Case
	name := string(featureKey)
	name = strings.ReplaceAll(name, "_", " ")
	words := strings.Fields(name)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

// GetTenantPlan retrieves the tenant plan from gin.Context
func GetTenantPlan(c *gin.Context) identity.TenantPlan {
	if plan, exists := c.Get(TenantPlanKey); exists {
		if tenantPlan, ok := plan.(identity.TenantPlan); ok {
			return tenantPlan
		}
	}
	return ""
}

// HasFeature is a helper function to check if the current tenant has a feature
// This can be used in handlers after the feature middleware has run
func HasFeature(c *gin.Context, cfg FeatureMiddlewareConfig, featureKey identity.FeatureKey) bool {
	tenantPlan := GetTenantPlan(c)
	if tenantPlan == "" {
		return false
	}

	hasFeature, err := checkFeature(c, cfg, tenantPlan, featureKey)
	if err != nil {
		return false
	}

	return hasFeature
}

// MustHaveFeature aborts the request if the tenant doesn't have the feature
// Returns true if the tenant has the feature, false if aborted
func MustHaveFeature(c *gin.Context, cfg FeatureMiddlewareConfig, featureKey identity.FeatureKey) bool {
	if !HasFeature(c, cfg, featureKey) {
		tenantPlan := GetTenantPlan(c)
		handleFeatureDenied(c, cfg, featureKey, tenantPlan, "")
		return false
	}
	return true
}

// WithFeature is a helper that wraps a handler and only executes it
// if the specified feature is enabled for the tenant's plan
func WithFeature(featureKey identity.FeatureKey, cfg FeatureMiddlewareConfig, handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !MustHaveFeature(c, cfg, featureKey) {
			return
		}
		handler(c)
	}
}
