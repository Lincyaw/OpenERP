package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/erp/backend/internal/domain/featureflag"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Constants for in-memory cache configuration
const (
	defaultCleanupInterval = 30 * time.Second
)

// InMemoryFeatureFlagCache implements FlagCache using in-memory storage
// This is designed to be used as L1 cache in front of Redis
type InMemoryFeatureFlagCache struct {
	flags     sync.Map // map[string]*cacheEntry[featureflag.FeatureFlag]
	overrides sync.Map // map[string]*cacheEntry[featureflag.FlagOverride]
	config    featureflag.CacheConfig
	logger    *zap.Logger
	stopCh    chan struct{} // Channel to stop the cleanup goroutine
	stopped   int32         // Atomic flag to track if cache is stopped

	// Stats for monitoring
	hits   int64
	misses int64
}

// cacheEntry wraps a cached value with expiration time
type cacheEntry[T any] struct {
	value     *T
	expiresAt time.Time
}

// isExpired checks if the cache entry has expired
func (e *cacheEntry[T]) isExpired() bool {
	return time.Now().After(e.expiresAt)
}

// InMemoryFeatureFlagCacheOption is a functional option for configuring the cache
type InMemoryFeatureFlagCacheOption func(*InMemoryFeatureFlagCache)

// WithInMemoryConfig sets the cache configuration
func WithInMemoryConfig(config featureflag.CacheConfig) InMemoryFeatureFlagCacheOption {
	return func(c *InMemoryFeatureFlagCache) {
		c.config = config
	}
}

// WithInMemoryLogger sets the logger for the cache
func WithInMemoryLogger(logger *zap.Logger) InMemoryFeatureFlagCacheOption {
	return func(c *InMemoryFeatureFlagCache) {
		c.logger = logger
	}
}

// NewInMemoryFeatureFlagCache creates a new in-memory feature flag cache
func NewInMemoryFeatureFlagCache(opts ...InMemoryFeatureFlagCacheOption) *InMemoryFeatureFlagCache {
	cache := &InMemoryFeatureFlagCache{
		config: featureflag.DefaultCacheConfig(),
		logger: zap.NewNop(),
		stopCh: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(cache)
	}

	// Start background cleanup goroutine
	go cache.cleanupExpired()

	return cache
}

// flagCacheKey generates the cache key for a feature flag
func (c *InMemoryFeatureFlagCache) flagCacheKey(key string) string {
	return "flag:" + key
}

// overrideCacheKey generates the cache key for an override
func (c *InMemoryFeatureFlagCache) overrideCacheKey(flagKey string, targetType featureflag.OverrideTargetType, targetID uuid.UUID) string {
	return "override:" + flagKey + ":" + string(targetType) + ":" + targetID.String()
}

// Get retrieves a feature flag from cache
func (c *InMemoryFeatureFlagCache) Get(ctx context.Context, key string) (*featureflag.FeatureFlag, error) {
	cacheKey := c.flagCacheKey(key)

	if value, ok := c.flags.Load(cacheKey); ok {
		entry := value.(*cacheEntry[featureflag.FeatureFlag])
		if !entry.isExpired() {
			atomic.AddInt64(&c.hits, 1)
			c.logger.Debug("L1 cache hit for feature flag", zap.String("key", key))
			return entry.value, nil
		}
		// Expired, remove from cache
		c.flags.Delete(cacheKey)
	}

	atomic.AddInt64(&c.misses, 1)
	c.logger.Debug("L1 cache miss for feature flag", zap.String("key", key))
	return nil, nil
}

// Set stores a feature flag in cache
func (c *InMemoryFeatureFlagCache) Set(ctx context.Context, key string, flag *featureflag.FeatureFlag, ttl time.Duration) error {
	if flag == nil {
		return nil
	}

	if ttl == 0 {
		ttl = c.config.L1TTL
	}

	cacheKey := c.flagCacheKey(key)
	entry := &cacheEntry[featureflag.FeatureFlag]{
		value:     flag,
		expiresAt: time.Now().Add(ttl),
	}

	c.flags.Store(cacheKey, entry)
	c.logger.Debug("Cached feature flag in L1",
		zap.String("key", key),
		zap.Duration("ttl", ttl))
	return nil
}

// Delete removes a feature flag from cache
func (c *InMemoryFeatureFlagCache) Delete(ctx context.Context, key string) error {
	cacheKey := c.flagCacheKey(key)
	c.flags.Delete(cacheKey)
	c.logger.Debug("Deleted feature flag from L1 cache", zap.String("key", key))
	return nil
}

// GetOverride retrieves a flag override from cache
func (c *InMemoryFeatureFlagCache) GetOverride(ctx context.Context, flagKey string, targetType featureflag.OverrideTargetType, targetID uuid.UUID) (*featureflag.FlagOverride, error) {
	cacheKey := c.overrideCacheKey(flagKey, targetType, targetID)

	if value, ok := c.overrides.Load(cacheKey); ok {
		entry := value.(*cacheEntry[featureflag.FlagOverride])
		if !entry.isExpired() {
			atomic.AddInt64(&c.hits, 1)
			c.logger.Debug("L1 cache hit for flag override",
				zap.String("flag_key", flagKey),
				zap.String("target_type", string(targetType)))
			return entry.value, nil
		}
		// Expired, remove from cache
		c.overrides.Delete(cacheKey)
	}

	atomic.AddInt64(&c.misses, 1)
	c.logger.Debug("L1 cache miss for flag override",
		zap.String("flag_key", flagKey),
		zap.String("target_type", string(targetType)))
	return nil, nil
}

// SetOverride stores a flag override in cache
func (c *InMemoryFeatureFlagCache) SetOverride(ctx context.Context, override *featureflag.FlagOverride, ttl time.Duration) error {
	if override == nil {
		return nil
	}

	if ttl == 0 {
		ttl = c.config.L1TTL
	}

	cacheKey := c.overrideCacheKey(override.FlagKey, override.TargetType, override.TargetID)
	entry := &cacheEntry[featureflag.FlagOverride]{
		value:     override,
		expiresAt: time.Now().Add(ttl),
	}

	c.overrides.Store(cacheKey, entry)
	c.logger.Debug("Cached flag override in L1",
		zap.String("flag_key", override.FlagKey),
		zap.String("target_type", string(override.TargetType)),
		zap.Duration("ttl", ttl))
	return nil
}

// DeleteOverride removes a flag override from cache
func (c *InMemoryFeatureFlagCache) DeleteOverride(ctx context.Context, flagKey string, targetType featureflag.OverrideTargetType, targetID uuid.UUID) error {
	cacheKey := c.overrideCacheKey(flagKey, targetType, targetID)
	c.overrides.Delete(cacheKey)
	c.logger.Debug("Deleted flag override from L1 cache",
		zap.String("flag_key", flagKey),
		zap.String("target_type", string(targetType)))
	return nil
}

// InvalidateAll removes all cached feature flags and overrides
func (c *InMemoryFeatureFlagCache) InvalidateAll(ctx context.Context) error {
	// Clear all entries
	c.flags.Range(func(key, _ any) bool {
		c.flags.Delete(key)
		return true
	})
	c.overrides.Range(func(key, _ any) bool {
		c.overrides.Delete(key)
		return true
	})

	c.logger.Info("Invalidated all L1 feature flag cache")
	return nil
}

// Close releases any resources held by the cache
func (c *InMemoryFeatureFlagCache) Close() error {
	// Only close once
	if atomic.CompareAndSwapInt32(&c.stopped, 0, 1) {
		close(c.stopCh)
	}
	return nil
}

// GetStats returns cache statistics
func (c *InMemoryFeatureFlagCache) GetStats() (hits, misses int64) {
	return atomic.LoadInt64(&c.hits), atomic.LoadInt64(&c.misses)
}

// ResetStats resets the cache statistics
func (c *InMemoryFeatureFlagCache) ResetStats() {
	atomic.StoreInt64(&c.hits, 0)
	atomic.StoreInt64(&c.misses, 0)
}

// Count returns the number of entries in the cache
func (c *InMemoryFeatureFlagCache) Count() (flags, overrides int) {
	c.flags.Range(func(_, _ any) bool {
		flags++
		return true
	})
	c.overrides.Range(func(_, _ any) bool {
		overrides++
		return true
	})
	return flags, overrides
}

// cleanupExpired periodically removes expired entries from the cache
func (c *InMemoryFeatureFlagCache) cleanupExpired() {
	ticker := time.NewTicker(defaultCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			func() {
				defer func() {
					if r := recover(); r != nil {
						c.logger.Error("Panic in cache cleanup",
							zap.Any("panic", r))
					}
				}()
				c.doCleanup()
			}()
		}
	}
}

// doCleanup removes expired entries from both caches
func (c *InMemoryFeatureFlagCache) doCleanup() {
	var flagsRemoved, overridesRemoved int

	c.flags.Range(func(key, value any) bool {
		entry := value.(*cacheEntry[featureflag.FeatureFlag])
		if entry.isExpired() {
			c.flags.Delete(key)
			flagsRemoved++
		}
		return true
	})

	c.overrides.Range(func(key, value any) bool {
		entry := value.(*cacheEntry[featureflag.FlagOverride])
		if entry.isExpired() {
			c.overrides.Delete(key)
			overridesRemoved++
		}
		return true
	})

	if flagsRemoved > 0 || overridesRemoved > 0 {
		c.logger.Debug("Cleaned up expired L1 cache entries",
			zap.Int("flags_removed", flagsRemoved),
			zap.Int("overrides_removed", overridesRemoved))
	}
}

// Ensure InMemoryFeatureFlagCache implements FlagCache
var _ featureflag.FlagCache = (*InMemoryFeatureFlagCache)(nil)
