package featureflag

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// FlagCache defines the interface for feature flag caching.
// Implementations should provide fast access to feature flags and overrides
// to achieve sub-millisecond evaluation latency.
//
// The cache operates as part of a multi-tier caching strategy:
// - L1: Local in-memory cache (sync.Map or similar) for ultra-fast access
// - L2: Redis cache for distributed consistency
// - L3: Database as the source of truth
//
// Cache keys follow the pattern:
// - Flags: feature_flag:{key}
// - Overrides: feature_flag:override:{flag_key}:{target_type}:{target_id}
type FlagCache interface {
	// Get retrieves a feature flag from cache by its key.
	// Returns nil, nil if the flag is not in cache (cache miss).
	// Returns nil, error if there was an error accessing the cache.
	Get(ctx context.Context, key string) (*FeatureFlag, error)

	// Set stores a feature flag in cache with the specified TTL.
	// If ttl is 0, implementation should use a default TTL.
	Set(ctx context.Context, key string, flag *FeatureFlag, ttl time.Duration) error

	// Delete removes a feature flag from cache by its key.
	Delete(ctx context.Context, key string) error

	// GetOverride retrieves a flag override from cache.
	// Returns nil, nil if the override is not in cache (cache miss).
	// Returns nil, error if there was an error accessing the cache.
	GetOverride(ctx context.Context, flagKey string, targetType OverrideTargetType, targetID uuid.UUID) (*FlagOverride, error)

	// SetOverride stores a flag override in cache with the specified TTL.
	// If ttl is 0, implementation should use a default TTL.
	SetOverride(ctx context.Context, override *FlagOverride, ttl time.Duration) error

	// DeleteOverride removes a flag override from cache.
	DeleteOverride(ctx context.Context, flagKey string, targetType OverrideTargetType, targetID uuid.UUID) error

	// InvalidateAll removes all cached feature flags and overrides.
	// This is typically used for cache warm-up or emergency cache clear.
	InvalidateAll(ctx context.Context) error

	// Close releases any resources held by the cache.
	Close() error
}

// CacheUpdateAction represents the type of cache update notification
type CacheUpdateAction string

const (
	// CacheUpdateActionUpdated indicates a flag was created or updated
	CacheUpdateActionUpdated CacheUpdateAction = "updated"
	// CacheUpdateActionDeleted indicates a flag was deleted
	CacheUpdateActionDeleted CacheUpdateAction = "deleted"
	// CacheUpdateActionOverrideUpdated indicates an override was created or updated
	CacheUpdateActionOverrideUpdated CacheUpdateAction = "override_updated"
	// CacheUpdateActionOverrideDeleted indicates an override was deleted
	CacheUpdateActionOverrideDeleted CacheUpdateAction = "override_deleted"
	// CacheUpdateActionInvalidateAll indicates all cache should be cleared
	CacheUpdateActionInvalidateAll CacheUpdateAction = "invalidate_all"
)

// CacheUpdateMessage represents a cache invalidation message
// sent via Pub/Sub to notify other instances of cache changes.
type CacheUpdateMessage struct {
	Action     CacheUpdateAction `json:"action"`
	FlagKey    string            `json:"flag_key,omitempty"`
	TargetType string            `json:"target_type,omitempty"`
	TargetID   string            `json:"target_id,omitempty"`
	Timestamp  int64             `json:"timestamp"`
}

// CacheInvalidator provides cache invalidation functionality.
// It allows publishing cache update notifications to other instances
// and subscribing to receive notifications from other instances.
type CacheInvalidator interface {
	// Publish sends a cache update notification to all subscribers.
	Publish(ctx context.Context, msg CacheUpdateMessage) error

	// Subscribe starts listening for cache update notifications.
	// The callback function is invoked for each received message.
	// This method should be called in a goroutine as it blocks.
	// Returns a cancel function to stop the subscription.
	Subscribe(ctx context.Context, callback func(msg CacheUpdateMessage)) error

	// Close releases any resources held by the invalidator.
	Close() error
}

// TieredFlagCache combines multiple cache layers for optimal performance.
// It follows a read-through, write-around pattern:
// - Reads: Check L1 -> Check L2 -> Database
// - Writes: Write to L2, invalidate L1 via Pub/Sub
type TieredFlagCache interface {
	FlagCache

	// GetL1 directly accesses the L1 (local) cache, bypassing L2.
	// This is useful for hot paths where you want minimal latency.
	GetL1(ctx context.Context, key string) (*FeatureFlag, error)

	// SetL1 directly sets a value in the L1 (local) cache.
	// This is typically called when receiving Pub/Sub notifications.
	SetL1(ctx context.Context, key string, flag *FeatureFlag, ttl time.Duration) error

	// InvalidateL1 removes an entry from the L1 (local) cache only.
	InvalidateL1(ctx context.Context, key string) error

	// GetCacheStats returns statistics about cache hits, misses, and other metrics.
	GetCacheStats(ctx context.Context) CacheStats
}

// CacheStats holds cache performance statistics
type CacheStats struct {
	L1Hits       int64   `json:"l1_hits"`
	L1Misses     int64   `json:"l1_misses"`
	L2Hits       int64   `json:"l2_hits"`
	L2Misses     int64   `json:"l2_misses"`
	TotalHits    int64   `json:"total_hits"`
	TotalMisses  int64   `json:"total_misses"`
	HitRatio     float64 `json:"hit_ratio"`
	CacheEntries int64   `json:"cache_entries"`
}

// CacheConfig holds configuration for the feature flag cache
type CacheConfig struct {
	// FlagTTL is the time-to-live for cached flags (default: 60s)
	FlagTTL time.Duration
	// OverrideTTL is the time-to-live for cached overrides (default: 60s)
	OverrideTTL time.Duration
	// L1TTL is the time-to-live for L1 (local) cache (default: 10s)
	L1TTL time.Duration
	// L1MaxSize is the maximum number of entries in L1 cache (default: 10000)
	L1MaxSize int
	// PubSubChannel is the Redis Pub/Sub channel name (default: "feature_flag:updates")
	PubSubChannel string
}

// DefaultCacheConfig returns the default cache configuration
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		FlagTTL:       60 * time.Second,
		OverrideTTL:   60 * time.Second,
		L1TTL:         10 * time.Second,
		L1MaxSize:     10000,
		PubSubChannel: "feature_flag:updates",
	}
}
