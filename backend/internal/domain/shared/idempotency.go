package shared

import (
	"context"
	"time"
)

// IdempotencyStore stores processed event IDs to prevent duplicate processing
type IdempotencyStore interface {
	// MarkProcessed marks an event as processed with a TTL
	// Returns true if the event was newly marked, false if it was already processed
	MarkProcessed(ctx context.Context, eventID string, ttl time.Duration) (bool, error)

	// IsProcessed checks if an event has already been processed
	IsProcessed(ctx context.Context, eventID string) (bool, error)

	// Close closes the store and releases resources
	Close() error
}

// IdempotencyConfig holds configuration for idempotency handling
type IdempotencyConfig struct {
	// TTL is the time-to-live for processed event IDs
	// After this duration, the same event ID can be processed again
	// Default: 24 hours
	TTL time.Duration

	// Enabled determines whether idempotency checking is enabled
	// Default: true
	Enabled bool
}

// DefaultIdempotencyConfig returns the default idempotency configuration
func DefaultIdempotencyConfig() IdempotencyConfig {
	return IdempotencyConfig{
		TTL:     24 * time.Hour,
		Enabled: true,
	}
}
