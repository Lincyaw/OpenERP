package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/erp/backend/internal/domain/featureflag"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Constants for invalidator configuration
const (
	defaultCloseTimeout = 5 * time.Second
)

// RedisFlagCacheInvalidator implements CacheInvalidator using Redis Pub/Sub
type RedisFlagCacheInvalidator struct {
	client     *redis.Client
	ownsClient bool // true if we created the client and should close it
	channel    string
	logger     *zap.Logger
	cancelFn   context.CancelFunc
	doneCh     chan struct{}
	doneOnce   sync.Once
	mu         sync.Mutex
	isRunning  bool
}

// RedisFlagCacheInvalidatorOption is a functional option for configuring the invalidator
type RedisFlagCacheInvalidatorOption func(*RedisFlagCacheInvalidator)

// WithInvalidatorChannel sets the Pub/Sub channel name
func WithInvalidatorChannel(channel string) RedisFlagCacheInvalidatorOption {
	return func(i *RedisFlagCacheInvalidator) {
		i.channel = channel
	}
}

// WithInvalidatorLogger sets the logger for the invalidator
func WithInvalidatorLogger(logger *zap.Logger) RedisFlagCacheInvalidatorOption {
	return func(i *RedisFlagCacheInvalidator) {
		i.logger = logger
	}
}

// NewRedisFlagCacheInvalidator creates a new Redis Pub/Sub cache invalidator
func NewRedisFlagCacheInvalidator(cfg RedisConfig, opts ...RedisFlagCacheInvalidatorOption) (*RedisFlagCacheInvalidator, error) {
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

	invalidator := &RedisFlagCacheInvalidator{
		client:     client,
		ownsClient: true, // We created this client, so we own it
		channel:    featureflag.DefaultCacheConfig().PubSubChannel,
		logger:     zap.NewNop(),
		doneCh:     make(chan struct{}),
	}

	for _, opt := range opts {
		opt(invalidator)
	}

	return invalidator, nil
}

// NewRedisFlagCacheInvalidatorWithClient creates an invalidator with an existing Redis client
// Note: The caller retains ownership of the client and is responsible for closing it
func NewRedisFlagCacheInvalidatorWithClient(client *redis.Client, opts ...RedisFlagCacheInvalidatorOption) *RedisFlagCacheInvalidator {
	invalidator := &RedisFlagCacheInvalidator{
		client:     client,
		ownsClient: false, // Client is shared, don't close it
		channel:    featureflag.DefaultCacheConfig().PubSubChannel,
		logger:     zap.NewNop(),
		doneCh:     make(chan struct{}),
	}

	for _, opt := range opts {
		opt(invalidator)
	}

	return invalidator
}

// Publish sends a cache update notification to all subscribers
func (i *RedisFlagCacheInvalidator) Publish(ctx context.Context, msg featureflag.CacheUpdateMessage) error {
	// Set timestamp if not set
	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().UnixNano()
	}

	data, err := json.Marshal(msg)
	if err != nil {
		i.logger.Error("Failed to marshal cache update message",
			zap.String("action", string(msg.Action)),
			zap.Error(err))
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := i.client.Publish(ctx, i.channel, data).Err(); err != nil {
		i.logger.Error("Failed to publish cache update message",
			zap.String("channel", i.channel),
			zap.Error(err))
		return fmt.Errorf("failed to publish message: %w", err)
	}

	i.logger.Debug("Published cache update message",
		zap.String("action", string(msg.Action)),
		zap.String("flag_key", msg.FlagKey),
		zap.String("channel", i.channel))

	return nil
}

// Subscribe starts listening for cache update notifications
// The callback function is invoked for each received message
// This method should be called in a goroutine as it blocks
func (i *RedisFlagCacheInvalidator) Subscribe(ctx context.Context, callback func(msg featureflag.CacheUpdateMessage)) error {
	i.mu.Lock()
	if i.isRunning {
		i.mu.Unlock()
		return fmt.Errorf("subscription already running")
	}
	i.isRunning = true
	i.mu.Unlock()

	// Create a cancellable context
	subCtx, cancel := context.WithCancel(ctx)
	i.mu.Lock()
	i.cancelFn = cancel
	i.mu.Unlock()

	pubsub := i.client.Subscribe(subCtx, i.channel)
	defer pubsub.Close()

	// Wait for subscription confirmation
	_, err := pubsub.Receive(subCtx)
	if err != nil {
		i.mu.Lock()
		i.isRunning = false
		i.mu.Unlock()
		return fmt.Errorf("failed to subscribe to channel: %w", err)
	}

	i.logger.Info("Subscribed to cache invalidation channel",
		zap.String("channel", i.channel))

	// Get the message channel
	ch := pubsub.Channel()

	for {
		select {
		case <-subCtx.Done():
			i.logger.Info("Cache invalidation subscription stopped")
			i.mu.Lock()
			i.isRunning = false
			i.mu.Unlock()
			i.markDone()
			return subCtx.Err()
		case msg, ok := <-ch:
			if !ok {
				i.logger.Warn("Cache invalidation channel closed")
				i.mu.Lock()
				i.isRunning = false
				i.mu.Unlock()
				i.markDone()
				return nil
			}

			var updateMsg featureflag.CacheUpdateMessage
			if err := json.Unmarshal([]byte(msg.Payload), &updateMsg); err != nil {
				i.logger.Error("Failed to unmarshal cache update message",
					zap.String("payload", msg.Payload),
					zap.Error(err))
				continue
			}

			i.logger.Debug("Received cache update message",
				zap.String("action", string(updateMsg.Action)),
				zap.String("flag_key", updateMsg.FlagKey))

			// Call the callback in a separate goroutine to prevent blocking
			go func(m featureflag.CacheUpdateMessage) {
				defer func() {
					if r := recover(); r != nil {
						i.logger.Error("Panic in cache update callback",
							zap.Any("panic", r))
					}
				}()
				callback(m)
			}(updateMsg)
		}
	}
}

// markDone safely marks the invalidator as done
func (i *RedisFlagCacheInvalidator) markDone() {
	i.doneOnce.Do(func() {
		close(i.doneCh)
	})
}

// Close releases any resources held by the invalidator
func (i *RedisFlagCacheInvalidator) Close() error {
	i.mu.Lock()
	cancelFn := i.cancelFn
	i.mu.Unlock()

	if cancelFn != nil {
		cancelFn()
		// Wait for subscription to stop with timeout
		select {
		case <-i.doneCh:
		case <-time.After(defaultCloseTimeout):
			i.logger.Warn("Timeout waiting for subscription to stop")
		}
	}

	// Only close client if we own it
	if i.ownsClient {
		return i.client.Close()
	}
	return nil
}

// PublishFlagUpdate publishes a flag update notification
func (i *RedisFlagCacheInvalidator) PublishFlagUpdate(ctx context.Context, flagKey string) error {
	return i.Publish(ctx, featureflag.CacheUpdateMessage{
		Action:  featureflag.CacheUpdateActionUpdated,
		FlagKey: flagKey,
	})
}

// PublishFlagDelete publishes a flag deletion notification
func (i *RedisFlagCacheInvalidator) PublishFlagDelete(ctx context.Context, flagKey string) error {
	return i.Publish(ctx, featureflag.CacheUpdateMessage{
		Action:  featureflag.CacheUpdateActionDeleted,
		FlagKey: flagKey,
	})
}

// PublishOverrideUpdate publishes an override update notification
func (i *RedisFlagCacheInvalidator) PublishOverrideUpdate(ctx context.Context, flagKey, targetType, targetID string) error {
	return i.Publish(ctx, featureflag.CacheUpdateMessage{
		Action:     featureflag.CacheUpdateActionOverrideUpdated,
		FlagKey:    flagKey,
		TargetType: targetType,
		TargetID:   targetID,
	})
}

// PublishOverrideDelete publishes an override deletion notification
func (i *RedisFlagCacheInvalidator) PublishOverrideDelete(ctx context.Context, flagKey, targetType, targetID string) error {
	return i.Publish(ctx, featureflag.CacheUpdateMessage{
		Action:     featureflag.CacheUpdateActionOverrideDeleted,
		FlagKey:    flagKey,
		TargetType: targetType,
		TargetID:   targetID,
	})
}

// PublishInvalidateAll publishes an invalidate-all notification
func (i *RedisFlagCacheInvalidator) PublishInvalidateAll(ctx context.Context) error {
	return i.Publish(ctx, featureflag.CacheUpdateMessage{
		Action: featureflag.CacheUpdateActionInvalidateAll,
	})
}

// GetClient returns the underlying Redis client (for testing/monitoring)
func (i *RedisFlagCacheInvalidator) GetClient() *redis.Client {
	return i.client
}

// Ensure RedisFlagCacheInvalidator implements CacheInvalidator
var _ featureflag.CacheInvalidator = (*RedisFlagCacheInvalidator)(nil)
