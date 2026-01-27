package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/featureflag"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Constants for Redis cache configuration
const (
	defaultScanBatchSize = 100
)

// RedisFeatureFlagCache implements FlagCache using Redis
type RedisFeatureFlagCache struct {
	client     *redis.Client
	ownsClient bool // true if we created the client and should close it
	config     featureflag.CacheConfig
	logger     *zap.Logger
}

// RedisFeatureFlagCacheOption is a functional option for configuring the cache
type RedisFeatureFlagCacheOption func(*RedisFeatureFlagCache)

// WithCacheConfig sets the cache configuration
func WithCacheConfig(config featureflag.CacheConfig) RedisFeatureFlagCacheOption {
	return func(c *RedisFeatureFlagCache) {
		c.config = config
	}
}

// WithCacheLogger sets the logger for the cache
func WithCacheLogger(logger *zap.Logger) RedisFeatureFlagCacheOption {
	return func(c *RedisFeatureFlagCache) {
		c.logger = logger
	}
}

// NewRedisFeatureFlagCache creates a new Redis-based feature flag cache
func NewRedisFeatureFlagCache(cfg RedisConfig, opts ...RedisFeatureFlagCacheOption) (*RedisFeatureFlagCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	cache := &RedisFeatureFlagCache{
		client:     client,
		ownsClient: true, // We created this client, so we own it
		config:     featureflag.DefaultCacheConfig(),
		logger:     zap.NewNop(),
	}

	for _, opt := range opts {
		opt(cache)
	}

	return cache, nil
}

// NewRedisFeatureFlagCacheWithClient creates a cache with an existing Redis client
// Note: The caller retains ownership of the client and is responsible for closing it
func NewRedisFeatureFlagCacheWithClient(client *redis.Client, opts ...RedisFeatureFlagCacheOption) *RedisFeatureFlagCache {
	cache := &RedisFeatureFlagCache{
		client:     client,
		ownsClient: false, // Client is shared, don't close it
		config:     featureflag.DefaultCacheConfig(),
		logger:     zap.NewNop(),
	}

	for _, opt := range opts {
		opt(cache)
	}

	return cache
}

// flagCacheKey generates the cache key for a feature flag
func (c *RedisFeatureFlagCache) flagCacheKey(key string) string {
	return fmt.Sprintf("feature_flag:%s", key)
}

// overrideCacheKey generates the cache key for an override
func (c *RedisFeatureFlagCache) overrideCacheKey(flagKey string, targetType featureflag.OverrideTargetType, targetID uuid.UUID) string {
	return fmt.Sprintf("feature_flag:override:%s:%s:%s", flagKey, targetType, targetID.String())
}

// Get retrieves a feature flag from cache
func (c *RedisFeatureFlagCache) Get(ctx context.Context, key string) (*featureflag.FeatureFlag, error) {
	cacheKey := c.flagCacheKey(key)

	data, err := c.client.Get(ctx, cacheKey).Bytes()
	if err == redis.Nil {
		// Cache miss
		c.logger.Debug("Cache miss for feature flag", zap.String("key", key))
		return nil, nil
	}
	if err != nil {
		c.logger.Error("Failed to get feature flag from cache",
			zap.String("key", key),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get flag from cache: %w", err)
	}

	var flag featureflag.FeatureFlag
	if err := json.Unmarshal(data, &flag); err != nil {
		c.logger.Error("Failed to unmarshal feature flag",
			zap.String("key", key),
			zap.Error(err))
		// Delete corrupted cache entry
		_ = c.client.Del(ctx, cacheKey)
		return nil, fmt.Errorf("failed to unmarshal flag: %w", err)
	}

	c.logger.Debug("Cache hit for feature flag", zap.String("key", key))
	return &flag, nil
}

// Set stores a feature flag in cache
func (c *RedisFeatureFlagCache) Set(ctx context.Context, key string, flag *featureflag.FeatureFlag, ttl time.Duration) error {
	if flag == nil {
		return nil
	}

	if ttl == 0 {
		ttl = c.config.FlagTTL
	}

	cacheKey := c.flagCacheKey(key)

	data, err := json.Marshal(flag)
	if err != nil {
		c.logger.Error("Failed to marshal feature flag",
			zap.String("key", key),
			zap.Error(err))
		return fmt.Errorf("failed to marshal flag: %w", err)
	}

	if err := c.client.Set(ctx, cacheKey, data, ttl).Err(); err != nil {
		c.logger.Error("Failed to set feature flag in cache",
			zap.String("key", key),
			zap.Error(err))
		return fmt.Errorf("failed to set flag in cache: %w", err)
	}

	c.logger.Debug("Cached feature flag",
		zap.String("key", key),
		zap.Duration("ttl", ttl))
	return nil
}

// Delete removes a feature flag from cache
func (c *RedisFeatureFlagCache) Delete(ctx context.Context, key string) error {
	cacheKey := c.flagCacheKey(key)

	if err := c.client.Del(ctx, cacheKey).Err(); err != nil {
		c.logger.Error("Failed to delete feature flag from cache",
			zap.String("key", key),
			zap.Error(err))
		return fmt.Errorf("failed to delete flag from cache: %w", err)
	}

	c.logger.Debug("Deleted feature flag from cache", zap.String("key", key))
	return nil
}

// GetOverride retrieves a flag override from cache
func (c *RedisFeatureFlagCache) GetOverride(ctx context.Context, flagKey string, targetType featureflag.OverrideTargetType, targetID uuid.UUID) (*featureflag.FlagOverride, error) {
	cacheKey := c.overrideCacheKey(flagKey, targetType, targetID)

	data, err := c.client.Get(ctx, cacheKey).Bytes()
	if err == redis.Nil {
		// Cache miss
		c.logger.Debug("Cache miss for flag override",
			zap.String("flag_key", flagKey),
			zap.String("target_type", string(targetType)),
			zap.String("target_id", targetID.String()))
		return nil, nil
	}
	if err != nil {
		c.logger.Error("Failed to get flag override from cache",
			zap.String("flag_key", flagKey),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get override from cache: %w", err)
	}

	var override featureflag.FlagOverride
	if err := json.Unmarshal(data, &override); err != nil {
		c.logger.Error("Failed to unmarshal flag override",
			zap.String("flag_key", flagKey),
			zap.Error(err))
		// Delete corrupted cache entry
		_ = c.client.Del(ctx, cacheKey)
		return nil, fmt.Errorf("failed to unmarshal override: %w", err)
	}

	c.logger.Debug("Cache hit for flag override",
		zap.String("flag_key", flagKey),
		zap.String("target_type", string(targetType)),
		zap.String("target_id", targetID.String()))
	return &override, nil
}

// SetOverride stores a flag override in cache
func (c *RedisFeatureFlagCache) SetOverride(ctx context.Context, override *featureflag.FlagOverride, ttl time.Duration) error {
	if override == nil {
		return nil
	}

	if ttl == 0 {
		ttl = c.config.OverrideTTL
	}

	cacheKey := c.overrideCacheKey(override.FlagKey, override.TargetType, override.TargetID)

	data, err := json.Marshal(override)
	if err != nil {
		c.logger.Error("Failed to marshal flag override",
			zap.String("flag_key", override.FlagKey),
			zap.Error(err))
		return fmt.Errorf("failed to marshal override: %w", err)
	}

	if err := c.client.Set(ctx, cacheKey, data, ttl).Err(); err != nil {
		c.logger.Error("Failed to set flag override in cache",
			zap.String("flag_key", override.FlagKey),
			zap.Error(err))
		return fmt.Errorf("failed to set override in cache: %w", err)
	}

	c.logger.Debug("Cached flag override",
		zap.String("flag_key", override.FlagKey),
		zap.String("target_type", string(override.TargetType)),
		zap.String("target_id", override.TargetID.String()),
		zap.Duration("ttl", ttl))
	return nil
}

// DeleteOverride removes a flag override from cache
func (c *RedisFeatureFlagCache) DeleteOverride(ctx context.Context, flagKey string, targetType featureflag.OverrideTargetType, targetID uuid.UUID) error {
	cacheKey := c.overrideCacheKey(flagKey, targetType, targetID)

	if err := c.client.Del(ctx, cacheKey).Err(); err != nil {
		c.logger.Error("Failed to delete flag override from cache",
			zap.String("flag_key", flagKey),
			zap.Error(err))
		return fmt.Errorf("failed to delete override from cache: %w", err)
	}

	c.logger.Debug("Deleted flag override from cache",
		zap.String("flag_key", flagKey),
		zap.String("target_type", string(targetType)),
		zap.String("target_id", targetID.String()))
	return nil
}

// InvalidateAll removes all cached feature flags and overrides
func (c *RedisFeatureFlagCache) InvalidateAll(ctx context.Context) error {
	// Use SCAN to find all feature flag keys to avoid blocking Redis with KEYS command
	var cursor uint64
	var deletedCount int64

	for {
		var keys []string
		var err error
		keys, cursor, err = c.client.Scan(ctx, cursor, "feature_flag:*", defaultScanBatchSize).Result()
		if err != nil {
			c.logger.Error("Failed to scan feature flag keys", zap.Error(err))
			return fmt.Errorf("failed to scan cache keys: %w", err)
		}

		if len(keys) > 0 {
			deleted, err := c.client.Del(ctx, keys...).Result()
			if err != nil {
				c.logger.Error("Failed to delete feature flag keys", zap.Error(err))
				return fmt.Errorf("failed to delete cache keys: %w", err)
			}
			deletedCount += deleted
		}

		if cursor == 0 {
			break
		}
	}

	c.logger.Info("Invalidated all feature flag cache",
		zap.Int64("deleted_count", deletedCount))
	return nil
}

// Close releases any resources held by the cache
func (c *RedisFeatureFlagCache) Close() error {
	// Only close client if we own it
	if c.ownsClient {
		return c.client.Close()
	}
	return nil
}

// GetClient returns the underlying Redis client (for testing/monitoring)
func (c *RedisFeatureFlagCache) GetClient() *redis.Client {
	return c.client
}

// Ensure RedisFeatureFlagCache implements FlagCache
var _ featureflag.FlagCache = (*RedisFeatureFlagCache)(nil)
