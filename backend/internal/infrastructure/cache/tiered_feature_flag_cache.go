package cache

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/erp/backend/internal/domain/featureflag"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// TieredFeatureFlagCache implements a two-tier caching strategy
// L1: Local in-memory cache (fast, but local to instance)
// L2: Redis cache (slower, but shared across instances)
// This follows a read-through, write-around pattern with Pub/Sub invalidation
type TieredFeatureFlagCache struct {
	l1Cache     *InMemoryFeatureFlagCache
	l2Cache     *RedisFeatureFlagCache
	invalidator *RedisFlagCacheInvalidator
	config      featureflag.CacheConfig
	logger      *zap.Logger

	// Stats for monitoring
	l1Hits   int64
	l1Misses int64
	l2Hits   int64
	l2Misses int64
}

// TieredFeatureFlagCacheOption is a functional option for configuring the cache
type TieredFeatureFlagCacheOption func(*TieredFeatureFlagCache)

// WithTieredConfig sets the cache configuration
func WithTieredConfig(config featureflag.CacheConfig) TieredFeatureFlagCacheOption {
	return func(c *TieredFeatureFlagCache) {
		c.config = config
	}
}

// WithTieredLogger sets the logger for the cache
func WithTieredLogger(logger *zap.Logger) TieredFeatureFlagCacheOption {
	return func(c *TieredFeatureFlagCache) {
		c.logger = logger
	}
}

// NewTieredFeatureFlagCache creates a new tiered feature flag cache
func NewTieredFeatureFlagCache(
	l1Cache *InMemoryFeatureFlagCache,
	l2Cache *RedisFeatureFlagCache,
	invalidator *RedisFlagCacheInvalidator,
	opts ...TieredFeatureFlagCacheOption,
) *TieredFeatureFlagCache {
	cache := &TieredFeatureFlagCache{
		l1Cache:     l1Cache,
		l2Cache:     l2Cache,
		invalidator: invalidator,
		config:      featureflag.DefaultCacheConfig(),
		logger:      zap.NewNop(),
	}

	for _, opt := range opts {
		opt(cache)
	}

	return cache
}

// StartInvalidationSubscription starts listening for cache invalidation messages
// This should be called after creating the cache, typically in a goroutine
func (c *TieredFeatureFlagCache) StartInvalidationSubscription(ctx context.Context) error {
	if c.invalidator == nil {
		return nil
	}

	return c.invalidator.Subscribe(ctx, func(msg featureflag.CacheUpdateMessage) {
		c.handleInvalidationMessage(msg)
	})
}

// handleInvalidationMessage processes cache invalidation messages
func (c *TieredFeatureFlagCache) handleInvalidationMessage(msg featureflag.CacheUpdateMessage) {
	ctx := context.Background()

	switch msg.Action {
	case featureflag.CacheUpdateActionUpdated, featureflag.CacheUpdateActionDeleted:
		// Invalidate L1 cache for the flag
		if err := c.l1Cache.Delete(ctx, msg.FlagKey); err != nil {
			c.logger.Error("Failed to invalidate L1 cache for flag",
				zap.String("flag_key", msg.FlagKey),
				zap.Error(err))
		}
		c.logger.Debug("Invalidated L1 cache for flag",
			zap.String("action", string(msg.Action)),
			zap.String("flag_key", msg.FlagKey))

	case featureflag.CacheUpdateActionOverrideUpdated, featureflag.CacheUpdateActionOverrideDeleted:
		// Invalidate L1 cache for the override
		if msg.TargetType != "" && msg.TargetID != "" {
			targetID, err := uuid.Parse(msg.TargetID)
			if err != nil {
				c.logger.Error("Failed to parse target ID in invalidation message",
					zap.String("target_id", msg.TargetID),
					zap.Error(err))
				return
			}
			if err := c.l1Cache.DeleteOverride(ctx, msg.FlagKey, featureflag.OverrideTargetType(msg.TargetType), targetID); err != nil {
				c.logger.Error("Failed to invalidate L1 cache for override",
					zap.String("flag_key", msg.FlagKey),
					zap.Error(err))
			}
			c.logger.Debug("Invalidated L1 cache for override",
				zap.String("action", string(msg.Action)),
				zap.String("flag_key", msg.FlagKey),
				zap.String("target_type", msg.TargetType),
				zap.String("target_id", msg.TargetID))
		}

	case featureflag.CacheUpdateActionInvalidateAll:
		// Invalidate all L1 cache
		if err := c.l1Cache.InvalidateAll(ctx); err != nil {
			c.logger.Error("Failed to invalidate all L1 cache", zap.Error(err))
		}
		c.logger.Info("Invalidated all L1 cache")
	}
}

// Get retrieves a feature flag from cache (L1 -> L2)
func (c *TieredFeatureFlagCache) Get(ctx context.Context, key string) (*featureflag.FeatureFlag, error) {
	// Try L1 first
	flag, err := c.l1Cache.Get(ctx, key)
	if err != nil {
		c.logger.Warn("L1 cache error", zap.String("key", key), zap.Error(err))
	}
	if flag != nil {
		atomic.AddInt64(&c.l1Hits, 1)
		return flag, nil
	}
	atomic.AddInt64(&c.l1Misses, 1)

	// Try L2
	flag, err = c.l2Cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if flag != nil {
		atomic.AddInt64(&c.l2Hits, 1)
		// Populate L1 cache
		if err := c.l1Cache.Set(ctx, key, flag, c.config.L1TTL); err != nil {
			c.logger.Warn("Failed to populate L1 cache", zap.String("key", key), zap.Error(err))
		}
		return flag, nil
	}
	atomic.AddInt64(&c.l2Misses, 1)

	return nil, nil
}

// Set stores a feature flag in cache (L2 only, L1 populated on read)
func (c *TieredFeatureFlagCache) Set(ctx context.Context, key string, flag *featureflag.FeatureFlag, ttl time.Duration) error {
	// Set in L2
	if err := c.l2Cache.Set(ctx, key, flag, ttl); err != nil {
		return err
	}

	// Also set in L1 for immediate local access
	if err := c.l1Cache.Set(ctx, key, flag, c.config.L1TTL); err != nil {
		c.logger.Warn("Failed to set L1 cache", zap.String("key", key), zap.Error(err))
	}

	// Publish invalidation for other instances
	if c.invalidator != nil {
		if err := c.invalidator.PublishFlagUpdate(ctx, key); err != nil {
			c.logger.Warn("Failed to publish flag update", zap.String("key", key), zap.Error(err))
		}
	}

	return nil
}

// Delete removes a feature flag from cache (both L1 and L2)
func (c *TieredFeatureFlagCache) Delete(ctx context.Context, key string) error {
	// Delete from L2
	if err := c.l2Cache.Delete(ctx, key); err != nil {
		return err
	}

	// Delete from L1
	if err := c.l1Cache.Delete(ctx, key); err != nil {
		c.logger.Warn("Failed to delete from L1 cache", zap.String("key", key), zap.Error(err))
	}

	// Publish invalidation for other instances
	if c.invalidator != nil {
		if err := c.invalidator.PublishFlagDelete(ctx, key); err != nil {
			c.logger.Warn("Failed to publish flag delete", zap.String("key", key), zap.Error(err))
		}
	}

	return nil
}

// GetOverride retrieves a flag override from cache (L1 -> L2)
func (c *TieredFeatureFlagCache) GetOverride(ctx context.Context, flagKey string, targetType featureflag.OverrideTargetType, targetID uuid.UUID) (*featureflag.FlagOverride, error) {
	// Try L1 first
	override, err := c.l1Cache.GetOverride(ctx, flagKey, targetType, targetID)
	if err != nil {
		c.logger.Warn("L1 cache error for override",
			zap.String("flag_key", flagKey),
			zap.Error(err))
	}
	if override != nil {
		atomic.AddInt64(&c.l1Hits, 1)
		return override, nil
	}
	atomic.AddInt64(&c.l1Misses, 1)

	// Try L2
	override, err = c.l2Cache.GetOverride(ctx, flagKey, targetType, targetID)
	if err != nil {
		return nil, err
	}
	if override != nil {
		atomic.AddInt64(&c.l2Hits, 1)
		// Populate L1 cache
		if err := c.l1Cache.SetOverride(ctx, override, c.config.L1TTL); err != nil {
			c.logger.Warn("Failed to populate L1 cache for override",
				zap.String("flag_key", flagKey),
				zap.Error(err))
		}
		return override, nil
	}
	atomic.AddInt64(&c.l2Misses, 1)

	return nil, nil
}

// SetOverride stores a flag override in cache
func (c *TieredFeatureFlagCache) SetOverride(ctx context.Context, override *featureflag.FlagOverride, ttl time.Duration) error {
	// Set in L2
	if err := c.l2Cache.SetOverride(ctx, override, ttl); err != nil {
		return err
	}

	// Also set in L1 for immediate local access
	if err := c.l1Cache.SetOverride(ctx, override, c.config.L1TTL); err != nil {
		c.logger.Warn("Failed to set L1 cache for override",
			zap.String("flag_key", override.FlagKey),
			zap.Error(err))
	}

	// Publish invalidation for other instances
	if c.invalidator != nil {
		if err := c.invalidator.PublishOverrideUpdate(ctx, override.FlagKey, string(override.TargetType), override.TargetID.String()); err != nil {
			c.logger.Warn("Failed to publish override update",
				zap.String("flag_key", override.FlagKey),
				zap.Error(err))
		}
	}

	return nil
}

// DeleteOverride removes a flag override from cache
func (c *TieredFeatureFlagCache) DeleteOverride(ctx context.Context, flagKey string, targetType featureflag.OverrideTargetType, targetID uuid.UUID) error {
	// Delete from L2
	if err := c.l2Cache.DeleteOverride(ctx, flagKey, targetType, targetID); err != nil {
		return err
	}

	// Delete from L1
	if err := c.l1Cache.DeleteOverride(ctx, flagKey, targetType, targetID); err != nil {
		c.logger.Warn("Failed to delete override from L1 cache",
			zap.String("flag_key", flagKey),
			zap.Error(err))
	}

	// Publish invalidation for other instances
	if c.invalidator != nil {
		if err := c.invalidator.PublishOverrideDelete(ctx, flagKey, string(targetType), targetID.String()); err != nil {
			c.logger.Warn("Failed to publish override delete",
				zap.String("flag_key", flagKey),
				zap.Error(err))
		}
	}

	return nil
}

// InvalidateAll removes all cached feature flags and overrides
func (c *TieredFeatureFlagCache) InvalidateAll(ctx context.Context) error {
	// Invalidate L2
	if err := c.l2Cache.InvalidateAll(ctx); err != nil {
		return err
	}

	// Invalidate L1
	if err := c.l1Cache.InvalidateAll(ctx); err != nil {
		c.logger.Warn("Failed to invalidate L1 cache", zap.Error(err))
	}

	// Publish invalidation for other instances
	if c.invalidator != nil {
		if err := c.invalidator.PublishInvalidateAll(ctx); err != nil {
			c.logger.Warn("Failed to publish invalidate all", zap.Error(err))
		}
	}

	return nil
}

// Close releases any resources held by the cache
func (c *TieredFeatureFlagCache) Close() error {
	var lastErr error

	if c.invalidator != nil {
		if err := c.invalidator.Close(); err != nil {
			lastErr = err
		}
	}

	if err := c.l2Cache.Close(); err != nil {
		lastErr = err
	}

	if err := c.l1Cache.Close(); err != nil {
		lastErr = err
	}

	return lastErr
}

// TieredFlagCache interface implementation

// GetL1 directly accesses the L1 (local) cache
func (c *TieredFeatureFlagCache) GetL1(ctx context.Context, key string) (*featureflag.FeatureFlag, error) {
	return c.l1Cache.Get(ctx, key)
}

// SetL1 directly sets a value in the L1 (local) cache
func (c *TieredFeatureFlagCache) SetL1(ctx context.Context, key string, flag *featureflag.FeatureFlag, ttl time.Duration) error {
	return c.l1Cache.Set(ctx, key, flag, ttl)
}

// InvalidateL1 removes an entry from the L1 (local) cache only
func (c *TieredFeatureFlagCache) InvalidateL1(ctx context.Context, key string) error {
	return c.l1Cache.Delete(ctx, key)
}

// GetCacheStats returns statistics about cache hits, misses, and other metrics
func (c *TieredFeatureFlagCache) GetCacheStats(ctx context.Context) featureflag.CacheStats {
	l1Hits := atomic.LoadInt64(&c.l1Hits)
	l1Misses := atomic.LoadInt64(&c.l1Misses)
	l2Hits := atomic.LoadInt64(&c.l2Hits)
	l2Misses := atomic.LoadInt64(&c.l2Misses)

	totalHits := l1Hits + l2Hits
	totalMisses := l2Misses // Only count final misses

	var hitRatio float64
	totalRequests := totalHits + totalMisses
	if totalRequests > 0 {
		hitRatio = float64(totalHits) / float64(totalRequests)
	}

	flagCount, overrideCount := c.l1Cache.Count()

	return featureflag.CacheStats{
		L1Hits:       l1Hits,
		L1Misses:     l1Misses,
		L2Hits:       l2Hits,
		L2Misses:     l2Misses,
		TotalHits:    totalHits,
		TotalMisses:  totalMisses,
		HitRatio:     hitRatio,
		CacheEntries: int64(flagCount + overrideCount),
	}
}

// ResetStats resets the cache statistics
func (c *TieredFeatureFlagCache) ResetStats() {
	atomic.StoreInt64(&c.l1Hits, 0)
	atomic.StoreInt64(&c.l1Misses, 0)
	atomic.StoreInt64(&c.l2Hits, 0)
	atomic.StoreInt64(&c.l2Misses, 0)
	c.l1Cache.ResetStats()
}

// Ensure TieredFeatureFlagCache implements both FlagCache and TieredFlagCache
var _ featureflag.FlagCache = (*TieredFeatureFlagCache)(nil)
var _ featureflag.TieredFlagCache = (*TieredFeatureFlagCache)(nil)
