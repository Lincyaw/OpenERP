// Package circuit provides circuit-board-like components for the load generator.
// The ProducerChainGuard prevents cascade overload when auto-triggering producers.
package circuit

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// Errors returned by ProducerChainGuard
var (
	// ErrMaxDepthExceeded is returned when recursion depth exceeds the limit.
	ErrMaxDepthExceeded = errors.New("producer chain: max recursion depth exceeded")
	// ErrCooldownActive is returned when attempting refill during cooldown period.
	ErrCooldownActive = errors.New("producer chain: cooldown period active")
	// ErrGuardClosed is returned when operations are attempted on a closed guard.
	ErrGuardClosed = errors.New("producer chain: guard is closed")
)

// Note: SemanticType is defined in semantic.go

// ProducerChainGuardConfig holds configuration for the guard.
type ProducerChainGuardConfig struct {
	// MaxDepth is the maximum recursion depth allowed for producer chains.
	// Default: 3
	MaxDepth int

	// CooldownPeriod is the minimum time between refill attempts for the same semantic type.
	// Default: 1s
	CooldownPeriod time.Duration

	// MinPoolSize is the threshold below which refill is triggered.
	// Default: 5
	MinPoolSize int

	// RefillBatchSize is the number of values to create in a single refill operation.
	// Default: 10
	RefillBatchSize int
}

// DefaultProducerChainGuardConfig returns the default configuration.
func DefaultProducerChainGuardConfig() ProducerChainGuardConfig {
	return ProducerChainGuardConfig{
		MaxDepth:        3,
		CooldownPeriod:  time.Second,
		MinPoolSize:     5,
		RefillBatchSize: 10,
	}
}

// ProducerChainGuard protects against cascade overload when auto-triggering producers.
// It limits recursion depth and implements cooldown periods to prevent system overload.
//
// Thread Safety: All public methods are safe for concurrent use by multiple goroutines,
// except WithNowFunc which must be called during initialization before concurrent access.
type ProducerChainGuard struct {
	config ProducerChainGuardConfig

	// currentDepth tracks the current recursion depth using atomic operations.
	currentDepth atomic.Int32

	// lastRefillTime tracks the last refill time for each semantic type.
	lastRefillTime sync.Map // map[SemanticType]time.Time

	// refillMu protects the check-and-store operation in TryRefill for concurrent safety.
	refillMu sync.Map // map[SemanticType]*sync.Mutex

	// stats tracks guard statistics.
	stats GuardStats

	// mu protects stats updates.
	mu sync.RWMutex

	// closed indicates whether the guard has been closed.
	closed atomic.Bool

	// nowFunc allows injecting time for testing.
	nowFunc func() time.Time
}

// GuardStats holds statistics about the guard's operation.
type GuardStats struct {
	// TotalEnterAttempts is the total number of Enter() calls.
	TotalEnterAttempts int64

	// TotalEnterSuccess is the number of successful Enter() calls.
	TotalEnterSuccess int64

	// TotalDepthRejections is the number of rejections due to max depth exceeded.
	TotalDepthRejections int64

	// TotalCooldownSkips is the number of refills skipped due to cooldown.
	TotalCooldownSkips int64

	// TotalRefillsTriggered is the number of refills that were allowed.
	TotalRefillsTriggered int64

	// PeakDepth is the maximum recursion depth observed.
	PeakDepth int32
}

// NewProducerChainGuard creates a new guard with the given configuration.
// If config is nil, default configuration is used.
func NewProducerChainGuard(config *ProducerChainGuardConfig) *ProducerChainGuard {
	cfg := DefaultProducerChainGuardConfig()
	if config != nil {
		if config.MaxDepth > 0 {
			cfg.MaxDepth = config.MaxDepth
		}
		if config.CooldownPeriod > 0 {
			cfg.CooldownPeriod = config.CooldownPeriod
		}
		if config.MinPoolSize > 0 {
			cfg.MinPoolSize = config.MinPoolSize
		}
		if config.RefillBatchSize > 0 {
			cfg.RefillBatchSize = config.RefillBatchSize
		}
	}

	return &ProducerChainGuard{
		config:  cfg,
		nowFunc: time.Now,
	}
}

// Enter attempts to enter a producer chain level.
// Returns nil if successful, or ErrMaxDepthExceeded if the depth limit is reached.
// Must be paired with Exit() on success.
func (g *ProducerChainGuard) Enter(ctx context.Context) error {
	if g.closed.Load() {
		return ErrGuardClosed
	}

	// Check context cancellation
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Atomically increment stats counter
	g.mu.Lock()
	g.stats.TotalEnterAttempts++
	g.mu.Unlock()

	// Atomically increment depth and check limit
	newDepth := g.currentDepth.Add(1)

	// Track peak depth
	g.mu.Lock()
	if newDepth > g.stats.PeakDepth {
		g.stats.PeakDepth = newDepth
	}
	g.mu.Unlock()

	if int(newDepth) > g.config.MaxDepth {
		// Roll back the increment
		g.currentDepth.Add(-1)

		g.mu.Lock()
		g.stats.TotalDepthRejections++
		g.mu.Unlock()

		return ErrMaxDepthExceeded
	}

	g.mu.Lock()
	g.stats.TotalEnterSuccess++
	g.mu.Unlock()

	return nil
}

// Exit decrements the current recursion depth.
// Must be called after a successful Enter().
func (g *ProducerChainGuard) Exit() {
	if g.closed.Load() {
		return
	}

	newDepth := g.currentDepth.Add(-1)
	if newDepth < 0 {
		// Reset to 0 if somehow we went negative (should not happen in normal operation)
		g.currentDepth.Store(0)
	}
}

// CurrentDepth returns the current recursion depth.
func (g *ProducerChainGuard) CurrentDepth() int {
	return int(g.currentDepth.Load())
}

// CanRefill checks if refill is allowed for the given semantic type.
// Returns true if refill is allowed, false if in cooldown period.
func (g *ProducerChainGuard) CanRefill(semantic SemanticType) bool {
	if g.closed.Load() {
		return false
	}

	now := g.nowFunc()

	lastTime, ok := g.lastRefillTime.Load(semantic)
	if !ok {
		return true
	}

	lt := lastTime.(time.Time)
	return now.Sub(lt) >= g.config.CooldownPeriod
}

// ShouldRefill checks if the pool should be refilled based on current size.
// Returns true if currentSize is below MinPoolSize.
func (g *ProducerChainGuard) ShouldRefill(currentSize int) bool {
	return currentSize < g.config.MinPoolSize
}

// TryRefill attempts to start a refill operation for the given semantic type.
// Returns the batch size and nil if allowed, or 0 and ErrCooldownActive if in cooldown.
// On success, the last refill time is updated.
// This method is safe for concurrent calls with the same semantic type.
func (g *ProducerChainGuard) TryRefill(ctx context.Context, semantic SemanticType) (int, error) {
	if g.closed.Load() {
		return 0, ErrGuardClosed
	}

	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	// Get or create a mutex for this semantic type
	muI, _ := g.refillMu.LoadOrStore(semantic, &sync.Mutex{})
	mu := muI.(*sync.Mutex)

	// Lock to make check-and-store atomic for this semantic type
	mu.Lock()
	defer mu.Unlock()

	now := g.nowFunc()

	// Check cooldown
	lastTime, ok := g.lastRefillTime.Load(semantic)
	if ok {
		lt := lastTime.(time.Time)
		if now.Sub(lt) < g.config.CooldownPeriod {
			g.mu.Lock()
			g.stats.TotalCooldownSkips++
			g.mu.Unlock()
			return 0, ErrCooldownActive
		}
	}

	// Update last refill time
	g.lastRefillTime.Store(semantic, now)

	g.mu.Lock()
	g.stats.TotalRefillsTriggered++
	g.mu.Unlock()

	return g.config.RefillBatchSize, nil
}

// ExecuteWithGuard executes a producer function with depth protection.
// It handles Enter/Exit automatically and returns an error if depth is exceeded.
func (g *ProducerChainGuard) ExecuteWithGuard(ctx context.Context, fn func(ctx context.Context) error) error {
	if err := g.Enter(ctx); err != nil {
		return err
	}
	defer g.Exit()

	return fn(ctx)
}

// Config returns a copy of the current configuration.
func (g *ProducerChainGuard) Config() ProducerChainGuardConfig {
	return g.config
}

// Stats returns a copy of the current statistics.
func (g *ProducerChainGuard) Stats() GuardStats {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.stats
}

// ResetStats resets all statistics to zero.
func (g *ProducerChainGuard) ResetStats() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.stats = GuardStats{}
}

// ResetCooldown resets the cooldown timer for a specific semantic type.
// This allows immediate refill for that type.
func (g *ProducerChainGuard) ResetCooldown(semantic SemanticType) {
	g.lastRefillTime.Delete(semantic)
	g.refillMu.Delete(semantic) // Clean up associated mutex
}

// ResetAllCooldowns resets all cooldown timers.
func (g *ProducerChainGuard) ResetAllCooldowns() {
	g.lastRefillTime.Range(func(key, _ any) bool {
		g.lastRefillTime.Delete(key)
		g.refillMu.Delete(key) // Clean up associated mutex
		return true
	})
}

// Close closes the guard and prevents further operations.
func (g *ProducerChainGuard) Close() {
	g.closed.Store(true)
}

// IsClosed returns whether the guard has been closed.
func (g *ProducerChainGuard) IsClosed() bool {
	return g.closed.Load()
}

// WithNowFunc sets a custom time function for testing.
// IMPORTANT: Must be called before concurrent access begins.
// Returns the guard for chaining.
func (g *ProducerChainGuard) WithNowFunc(fn func() time.Time) *ProducerChainGuard {
	g.nowFunc = fn
	return g
}
