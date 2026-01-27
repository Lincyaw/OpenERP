package auth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// TokenBlacklist defines the interface for token blacklisting operations
// This is used to invalidate JWT tokens before they expire (e.g., on logout)
type TokenBlacklist interface {
	// AddToBlacklist adds a token's JTI (JWT ID) to the blacklist
	// ttl should be set to the remaining time until token expiration
	AddToBlacklist(ctx context.Context, jti string, ttl time.Duration) error

	// IsBlacklisted checks if a token's JTI is in the blacklist
	IsBlacklisted(ctx context.Context, jti string) (bool, error)

	// AddUserTokensToBlacklist blacklists all tokens for a user (force logout all sessions)
	// This stores the invalidation timestamp, and tokens issued before this time are invalid
	AddUserTokensToBlacklist(ctx context.Context, userID string, ttl time.Duration) error

	// IsUserTokenInvalidated checks if a user's tokens have been invalidated
	// Returns true if tokens issued before the invalidation timestamp should be rejected
	IsUserTokenInvalidated(ctx context.Context, userID string, tokenIssuedAt time.Time) (bool, error)
}

// RedisTokenBlacklist implements TokenBlacklist using Redis
type RedisTokenBlacklist struct {
	client    *redis.Client
	keyPrefix string
}

// RedisTokenBlacklistConfig holds configuration for Redis token blacklist
type RedisTokenBlacklistConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// NewRedisTokenBlacklist creates a new Redis-based token blacklist
func NewRedisTokenBlacklist(cfg RedisTokenBlacklistConfig) (*RedisTokenBlacklist, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     10, // Connection pool size
		MinIdleConns: 3,  // Minimum idle connections
		MaxRetries:   3,  // Max retries on failure
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis for token blacklist: %w", err)
	}

	return &RedisTokenBlacklist{
		client:    client,
		keyPrefix: "token:blacklist:",
	}, nil
}

// NewRedisTokenBlacklistWithClient creates a token blacklist with an existing Redis client
func NewRedisTokenBlacklistWithClient(client *redis.Client) *RedisTokenBlacklist {
	return &RedisTokenBlacklist{
		client:    client,
		keyPrefix: "token:blacklist:",
	}
}

// jtiKey returns the Redis key for a JTI
func (b *RedisTokenBlacklist) jtiKey(jti string) string {
	return b.keyPrefix + "jti:" + jti
}

// userKey returns the Redis key for user token invalidation
func (b *RedisTokenBlacklist) userKey(userID string) string {
	return b.keyPrefix + "user:" + userID
}

// AddToBlacklist adds a token's JTI to the blacklist
func (b *RedisTokenBlacklist) AddToBlacklist(ctx context.Context, jti string, ttl time.Duration) error {
	key := b.jtiKey(jti)

	// Store with TTL - value "1" indicates blacklisted
	err := b.client.Set(ctx, key, "1", ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to add token to blacklist: %w", err)
	}

	return nil
}

// IsBlacklisted checks if a token's JTI is in the blacklist
func (b *RedisTokenBlacklist) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	key := b.jtiKey(jti)

	exists, err := b.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check token blacklist: %w", err)
	}

	return exists > 0, nil
}

// AddUserTokensToBlacklist invalidates all tokens for a user by storing the current timestamp
// Any token issued before this timestamp will be considered invalid
func (b *RedisTokenBlacklist) AddUserTokensToBlacklist(ctx context.Context, userID string, ttl time.Duration) error {
	key := b.userKey(userID)

	// Store current Unix timestamp as invalidation time
	invalidationTime := time.Now().Unix()
	err := b.client.Set(ctx, key, invalidationTime, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to invalidate user tokens: %w", err)
	}

	return nil
}

// IsUserTokenInvalidated checks if a token was issued before the user's invalidation timestamp
func (b *RedisTokenBlacklist) IsUserTokenInvalidated(ctx context.Context, userID string, tokenIssuedAt time.Time) (bool, error) {
	key := b.userKey(userID)

	invalidationTimeStr, err := b.client.Get(ctx, key).Result()
	if err == redis.Nil {
		// No invalidation timestamp, token is valid
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check user token invalidation: %w", err)
	}

	var invalidationTime int64
	_, err = fmt.Sscanf(invalidationTimeStr, "%d", &invalidationTime)
	if err != nil {
		return false, fmt.Errorf("failed to parse invalidation timestamp: %w", err)
	}

	// Token is invalid if it was issued before or at the invalidation time
	return tokenIssuedAt.Unix() <= invalidationTime, nil
}

// Close closes the Redis client
func (b *RedisTokenBlacklist) Close() error {
	return b.client.Close()
}

// GetClient returns the underlying Redis client (for testing/monitoring)
func (b *RedisTokenBlacklist) GetClient() *redis.Client {
	return b.client
}

// Ensure RedisTokenBlacklist implements TokenBlacklist
var _ TokenBlacklist = (*RedisTokenBlacklist)(nil)

// InMemoryTokenBlacklist provides an in-memory implementation for testing
// WARNING: This should not be used in production with multiple instances
type InMemoryTokenBlacklist struct {
	mu                    sync.RWMutex
	jtiBlacklist          map[string]time.Time // JTI -> expiration time
	userInvalidationTimes map[string]time.Time // userID -> invalidation time
}

// NewInMemoryTokenBlacklist creates a new in-memory token blacklist
func NewInMemoryTokenBlacklist() *InMemoryTokenBlacklist {
	return &InMemoryTokenBlacklist{
		jtiBlacklist:          make(map[string]time.Time),
		userInvalidationTimes: make(map[string]time.Time),
	}
}

// AddToBlacklist adds a token's JTI to the in-memory blacklist
func (b *InMemoryTokenBlacklist) AddToBlacklist(_ context.Context, jti string, ttl time.Duration) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.jtiBlacklist[jti] = time.Now().Add(ttl)
	return nil
}

// IsBlacklisted checks if a token's JTI is blacklisted (and not expired)
func (b *InMemoryTokenBlacklist) IsBlacklisted(_ context.Context, jti string) (bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	expiration, exists := b.jtiBlacklist[jti]
	if !exists {
		return false, nil
	}

	// Check if the blacklist entry has expired
	if time.Now().After(expiration) {
		delete(b.jtiBlacklist, jti)
		return false, nil
	}

	return true, nil
}

// AddUserTokensToBlacklist invalidates all tokens for a user
func (b *InMemoryTokenBlacklist) AddUserTokensToBlacklist(_ context.Context, userID string, _ time.Duration) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.userInvalidationTimes[userID] = time.Now()
	return nil
}

// IsUserTokenInvalidated checks if a token was issued before the user's invalidation timestamp
func (b *InMemoryTokenBlacklist) IsUserTokenInvalidated(_ context.Context, userID string, tokenIssuedAt time.Time) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	invalidationTime, exists := b.userInvalidationTimes[userID]
	if !exists {
		return false, nil
	}

	// Use UnixNano for sub-second precision in testing
	// Token is invalid if it was issued at or before the invalidation time
	return tokenIssuedAt.UnixNano() <= invalidationTime.UnixNano(), nil
}

// Ensure InMemoryTokenBlacklist implements TokenBlacklist
var _ TokenBlacklist = (*InMemoryTokenBlacklist)(nil)
