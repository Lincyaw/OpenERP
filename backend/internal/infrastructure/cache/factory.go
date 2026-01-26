package cache

import (
	"fmt"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/config"
	"go.uber.org/zap"
)

// IdempotencyStoreFactory creates idempotency stores based on configuration
type IdempotencyStoreFactory struct {
	redisConfig           config.RedisConfig
	logger                *zap.Logger
	allowInMemoryFallback bool
}

// IdempotencyStoreFactoryOption is a functional option for configuring the factory
type IdempotencyStoreFactoryOption func(*IdempotencyStoreFactory)

// WithLogger sets the logger for the factory
func WithLogger(logger *zap.Logger) IdempotencyStoreFactoryOption {
	return func(f *IdempotencyStoreFactory) {
		f.logger = logger
	}
}

// WithInMemoryFallback controls whether to fall back to in-memory store when Redis is unavailable
// Default is true (allow fallback)
func WithInMemoryFallback(allow bool) IdempotencyStoreFactoryOption {
	return func(f *IdempotencyStoreFactory) {
		f.allowInMemoryFallback = allow
	}
}

// NewIdempotencyStoreFactory creates a new factory
func NewIdempotencyStoreFactory(cfg config.RedisConfig, opts ...IdempotencyStoreFactoryOption) *IdempotencyStoreFactory {
	f := &IdempotencyStoreFactory{
		redisConfig:           cfg,
		logger:                zap.NewNop(),
		allowInMemoryFallback: true, // Default to allowing fallback
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// CreateRedisStore creates a Redis-based idempotency store
func (f *IdempotencyStoreFactory) CreateRedisStore() (shared.IdempotencyStore, error) {
	redisCfg := RedisConfig{
		Host:     f.redisConfig.Host,
		Port:     f.redisConfig.Port,
		Password: f.redisConfig.Password,
		DB:       f.redisConfig.DB,
	}

	store, err := NewRedisIdempotencyStore(redisCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis idempotency store: %w", err)
	}

	return store, nil
}

// CreateInMemoryStore creates an in-memory idempotency store
// This is suitable for single-instance deployments and testing
// WARNING: In-memory stores do not share state across process instances,
// which can lead to duplicate event processing in distributed deployments
func (f *IdempotencyStoreFactory) CreateInMemoryStore() shared.IdempotencyStore {
	return NewInMemoryIdempotencyStore()
}

// CreateStore creates an idempotency store based on whether Redis is available
// It tries to create a Redis store first, and falls back to in-memory if Redis is not available
// and AllowInMemoryFallback is true
func (f *IdempotencyStoreFactory) CreateStore() (shared.IdempotencyStore, error) {
	// Try Redis first
	store, err := f.CreateRedisStore()
	if err == nil {
		f.logger.Info("using Redis idempotency store")
		return store, nil
	}

	// Check if fallback is allowed
	if !f.allowInMemoryFallback {
		return nil, fmt.Errorf("Redis required for idempotency but unavailable: %w", err)
	}

	// Fall back to in-memory with warning
	f.logger.Warn("Redis unavailable, falling back to in-memory idempotency store. "+
		"This may cause duplicate event processing in distributed deployments.",
		zap.Error(err),
	)
	return f.CreateInMemoryStore(), nil
}
