package pool

import (
	"context"
	"time"
)

// EvictionPolicy defines how values are evicted when the pool is full.
type EvictionPolicy int

const (
	// EvictionFIFO evicts the oldest values first (First-In-First-Out).
	EvictionFIFO EvictionPolicy = iota

	// EvictionLRU evicts the least recently used values first.
	EvictionLRU

	// EvictionRandom evicts values at random.
	EvictionRandom
)

func (e EvictionPolicy) String() string {
	switch e {
	case EvictionFIFO:
		return "FIFO"
	case EvictionLRU:
		return "LRU"
	case EvictionRandom:
		return "Random"
	default:
		return "Unknown"
	}
}

// ParseEvictionPolicy parses a string into an EvictionPolicy.
func ParseEvictionPolicy(s string) EvictionPolicy {
	switch s {
	case "LRU", "lru":
		return EvictionLRU
	case "Random", "random", "RANDOM":
		return EvictionRandom
	default:
		return EvictionFIFO
	}
}

// Stats represents statistics about a parameter pool.
type Stats struct {
	// TotalValues is the total number of values in the pool
	TotalValues int64

	// ValuesByType is the count of values per semantic type
	ValuesByType map[SemanticType]int64

	// HitCount is the number of successful Get operations
	HitCount int64

	// MissCount is the number of failed Get operations (value not found)
	MissCount int64

	// EvictionCount is the number of values that have been evicted
	EvictionCount int64

	// ExpiredCount is the number of values that have expired
	ExpiredCount int64

	// AddCount is the total number of values added
	AddCount int64

	// Uptime is how long the pool has been running
	Uptime time.Duration
}

// HitRate returns the hit rate as a percentage (0-100).
func (s Stats) HitRate() float64 {
	total := s.HitCount + s.MissCount
	if total == 0 {
		return 0
	}
	return float64(s.HitCount) / float64(total) * 100
}

// ParameterPool defines the interface for parameter storage and retrieval.
type ParameterPool interface {
	// Add adds a value to the pool for the given semantic type.
	// Returns the number of values evicted (if any) to make room.
	Add(ctx context.Context, value *ParameterValue) (evicted int, err error)

	// Get retrieves a value for the given semantic type.
	// Returns nil if no value is available.
	Get(ctx context.Context, semanticType SemanticType) (*ParameterValue, error)

	// GetRandom retrieves a random value for the given semantic type.
	// Returns nil if no value is available.
	GetRandom(ctx context.Context, semanticType SemanticType) (*ParameterValue, error)

	// GetAll retrieves all values for the given semantic type.
	GetAll(ctx context.Context, semanticType SemanticType) ([]*ParameterValue, error)

	// Count returns the number of values for the given semantic type.
	Count(ctx context.Context, semanticType SemanticType) (int, error)

	// Remove removes a specific value from the pool.
	// Returns true if the value was found and removed.
	Remove(ctx context.Context, value *ParameterValue) (bool, error)

	// Clear removes all values for the given semantic type.
	// Returns the number of values removed.
	Clear(ctx context.Context, semanticType SemanticType) (int, error)

	// ClearAll removes all values from the pool.
	ClearAll(ctx context.Context) error

	// Cleanup removes expired values from the pool.
	// Returns the number of values removed.
	Cleanup(ctx context.Context) (int, error)

	// Stats returns statistics about the pool.
	Stats(ctx context.Context) (Stats, error)

	// Types returns all semantic types that have values in the pool.
	Types(ctx context.Context) ([]SemanticType, error)

	// Close releases resources held by the pool.
	Close() error
}

// PoolConfig holds configuration options for parameter pools.
type PoolConfig struct {
	// DefaultTTL is the default time-to-live for values (0 means no expiration)
	DefaultTTL time.Duration

	// MaxValuesPerType is the maximum number of values per semantic type (0 means unlimited)
	MaxValuesPerType int

	// EvictionPolicy determines how values are evicted when the pool is full
	EvictionPolicy EvictionPolicy

	// CleanupInterval is how often to run cleanup of expired values (0 means no automatic cleanup)
	CleanupInterval time.Duration

	// ShardCount is the number of shards for ShardedParameterPool (must be power of 2)
	ShardCount int
}

// DefaultPoolConfig returns a PoolConfig with sensible defaults.
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		DefaultTTL:       5 * time.Minute,
		MaxValuesPerType: 1000,
		EvictionPolicy:   EvictionFIFO,
		CleanupInterval:  1 * time.Minute,
		ShardCount:       16,
	}
}
