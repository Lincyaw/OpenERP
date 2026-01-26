package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/redis/go-redis/v9"
)

// RedisIdempotencyStore implements IdempotencyStore using Redis
// This is suitable for distributed deployments where multiple instances
// need to share idempotency state
type RedisIdempotencyStore struct {
	client    *redis.Client
	keyPrefix string
}

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// NewRedisIdempotencyStore creates a new Redis-based idempotency store
func NewRedisIdempotencyStore(cfg RedisConfig) (*RedisIdempotencyStore, error) {
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

	return &RedisIdempotencyStore{
		client:    client,
		keyPrefix: "event:idempotency:",
	}, nil
}

// NewRedisIdempotencyStoreWithClient creates a store with an existing Redis client
// This is useful for testing or when sharing a client across components
func NewRedisIdempotencyStoreWithClient(client *redis.Client, keyPrefix string) *RedisIdempotencyStore {
	if keyPrefix == "" {
		keyPrefix = "event:idempotency:"
	}
	return &RedisIdempotencyStore{
		client:    client,
		keyPrefix: keyPrefix,
	}
}

// MarkProcessed marks an event as processed with a TTL
// Returns true if the event was newly marked, false if it was already processed
// Uses SETNX (SET if Not eXists) for atomic operation
func (s *RedisIdempotencyStore) MarkProcessed(ctx context.Context, eventID string, ttl time.Duration) (bool, error) {
	key := s.keyPrefix + eventID

	// Use SETNX with TTL in a single atomic operation
	// Returns true if key was set, false if it already existed
	result, err := s.client.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to mark event as processed: %w", err)
	}

	return result, nil
}

// IsProcessed checks if an event has already been processed
func (s *RedisIdempotencyStore) IsProcessed(ctx context.Context, eventID string) (bool, error) {
	key := s.keyPrefix + eventID

	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check if event is processed: %w", err)
	}

	return exists > 0, nil
}

// Close closes the Redis client
func (s *RedisIdempotencyStore) Close() error {
	return s.client.Close()
}

// GetClient returns the underlying Redis client (for testing/monitoring)
func (s *RedisIdempotencyStore) GetClient() *redis.Client {
	return s.client
}

// Ensure RedisIdempotencyStore implements IdempotencyStore
var _ shared.IdempotencyStore = (*RedisIdempotencyStore)(nil)
