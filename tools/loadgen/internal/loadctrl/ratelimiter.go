// Package loadctrl provides load control components including traffic shaping
// and rate limiting for the load generator.
package loadctrl

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiterType defines the type of rate limiter algorithm.
type RateLimiterType string

const (
	// RateLimiterTokenBucket uses token bucket algorithm (recommended).
	RateLimiterTokenBucket RateLimiterType = "token_bucket"
	// RateLimiterLeakyBucket uses leaky bucket algorithm.
	RateLimiterLeakyBucket RateLimiterType = "leaky_bucket"
	// RateLimiterSlidingWindow uses sliding window algorithm.
	RateLimiterSlidingWindow RateLimiterType = "sliding_window"
)

// RateLimiter defines the interface for rate limiting strategies.
// Implementations control the rate at which requests can be sent.
//
// Thread Safety: Implementations must be safe for concurrent use by multiple goroutines.
type RateLimiter interface {
	// Acquire blocks until a request slot is available or context is cancelled.
	// Returns an error if the context is cancelled or deadline exceeded.
	Acquire(ctx context.Context) error

	// TryAcquire attempts to acquire a request slot without blocking.
	// Returns true if a slot was acquired, false otherwise.
	TryAcquire() bool

	// SetRate dynamically adjusts the rate limit.
	// The new rate takes effect immediately.
	SetRate(qps float64)

	// CurrentRate returns the current rate limit in QPS.
	CurrentRate() float64

	// Stats returns current statistics about the rate limiter.
	Stats() RateLimiterStats
}

// RateLimiterStats contains statistics about rate limiter usage.
type RateLimiterStats struct {
	// TotalAcquired is the total number of successful acquisitions.
	TotalAcquired int64
	// TotalRejected is the total number of rejected acquisitions (TryAcquire failures).
	TotalRejected int64
	// CurrentQPS is the current configured QPS.
	CurrentQPS float64
	// AvgWaitTime is the average time spent waiting in Acquire calls.
	AvgWaitTime time.Duration
}

// RateLimiterConfig holds configuration for creating a rate limiter.
type RateLimiterConfig struct {
	// Type specifies the rate limiter algorithm.
	Type RateLimiterType `yaml:"type" json:"type"`
	// QPS is the target requests per second.
	QPS float64 `yaml:"qps" json:"qps"`
	// BurstSize is the maximum burst size (only for token bucket).
	BurstSize int `yaml:"burstSize,omitempty" json:"burstSize,omitempty"`
}

// TokenBucketLimiter implements RateLimiter using the token bucket algorithm.
// It uses golang.org/x/time/rate under the hood.
//
// Thread Safety: Safe for concurrent use.
type TokenBucketLimiter struct {
	limiter   *rate.Limiter
	burstSize int
	qps       float64
	mu        sync.RWMutex

	// Statistics
	totalAcquired atomic.Int64
	totalRejected atomic.Int64
	totalWaitTime atomic.Int64 // in nanoseconds
	waitCount     atomic.Int64
}

// NewTokenBucketLimiter creates a new token bucket rate limiter.
// qps is the target requests per second, burst is the maximum burst size.
// If burst is 0, it defaults to max(1, int(qps)).
func NewTokenBucketLimiter(qps float64, burst int) *TokenBucketLimiter {
	if burst <= 0 {
		burst = max(1, int(qps))
	}
	if qps <= 0 {
		qps = 1 // Minimum QPS
	}
	return &TokenBucketLimiter{
		limiter:   rate.NewLimiter(rate.Limit(qps), burst),
		burstSize: burst,
		qps:       qps,
	}
}

// Acquire blocks until a request slot is available or context is cancelled.
func (l *TokenBucketLimiter) Acquire(ctx context.Context) error {
	start := time.Now()
	err := l.limiter.Wait(ctx)
	if err == nil {
		l.totalAcquired.Add(1)
		waitTime := time.Since(start)
		l.totalWaitTime.Add(int64(waitTime))
		l.waitCount.Add(1)
	}
	return err
}

// TryAcquire attempts to acquire a request slot without blocking.
func (l *TokenBucketLimiter) TryAcquire() bool {
	if l.limiter.Allow() {
		l.totalAcquired.Add(1)
		return true
	}
	l.totalRejected.Add(1)
	return false
}

// SetRate dynamically adjusts the rate limit.
func (l *TokenBucketLimiter) SetRate(qps float64) {
	if qps <= 0 {
		qps = 1 // Minimum QPS
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.qps = qps
	l.limiter.SetLimit(rate.Limit(qps))
}

// CurrentRate returns the current rate limit in QPS.
func (l *TokenBucketLimiter) CurrentRate() float64 {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.qps
}

// Stats returns current statistics about the rate limiter.
func (l *TokenBucketLimiter) Stats() RateLimiterStats {
	acquired := l.totalAcquired.Load()
	rejected := l.totalRejected.Load()
	totalWait := l.totalWaitTime.Load()
	waitCnt := l.waitCount.Load()

	var avgWait time.Duration
	if waitCnt > 0 {
		avgWait = time.Duration(totalWait / waitCnt)
	}

	l.mu.RLock()
	currentQPS := l.qps
	l.mu.RUnlock()

	return RateLimiterStats{
		TotalAcquired: acquired,
		TotalRejected: rejected,
		CurrentQPS:    currentQPS,
		AvgWaitTime:   avgWait,
	}
}

// BurstSize returns the configured burst size.
func (l *TokenBucketLimiter) BurstSize() int {
	return l.burstSize
}

// SetBurst dynamically adjusts the burst size.
func (l *TokenBucketLimiter) SetBurst(burst int) {
	if burst <= 0 {
		burst = 1
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.burstSize = burst
	l.limiter.SetBurst(burst)
}

// LeakyBucketLimiter implements RateLimiter using the leaky bucket algorithm.
// It ensures a constant rate of requests by spacing them evenly.
//
// Thread Safety: Safe for concurrent use.
type LeakyBucketLimiter struct {
	qps         float64
	interval    time.Duration
	lastRequest time.Time
	mu          sync.Mutex

	// Statistics
	totalAcquired atomic.Int64
	totalRejected atomic.Int64
	totalWaitTime atomic.Int64
	waitCount     atomic.Int64
}

// NewLeakyBucketLimiter creates a new leaky bucket rate limiter.
func NewLeakyBucketLimiter(qps float64) *LeakyBucketLimiter {
	if qps <= 0 {
		qps = 1
	}
	return &LeakyBucketLimiter{
		qps:      qps,
		interval: time.Duration(float64(time.Second) / qps),
	}
}

// Acquire blocks until a request slot is available or context is cancelled.
func (l *LeakyBucketLimiter) Acquire(ctx context.Context) error {
	start := time.Now()

	for {
		l.mu.Lock()
		now := time.Now()
		nextAllowed := l.lastRequest.Add(l.interval)

		if !now.Before(nextAllowed) {
			// Slot available, acquire it
			l.lastRequest = now
			l.mu.Unlock()
			l.totalAcquired.Add(1)
			l.totalWaitTime.Add(int64(time.Since(start)))
			l.waitCount.Add(1)
			return nil
		}

		waitTime := nextAllowed.Sub(now)
		l.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Loop back and recheck
		}
	}
}

// TryAcquire attempts to acquire a request slot without blocking.
func (l *LeakyBucketLimiter) TryAcquire() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	nextAllowed := l.lastRequest.Add(l.interval)

	if now.Before(nextAllowed) {
		l.totalRejected.Add(1)
		return false
	}

	l.lastRequest = now
	l.totalAcquired.Add(1)
	return true
}

// SetRate dynamically adjusts the rate limit.
func (l *LeakyBucketLimiter) SetRate(qps float64) {
	if qps <= 0 {
		qps = 1
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.qps = qps
	l.interval = time.Duration(float64(time.Second) / qps)
}

// CurrentRate returns the current rate limit in QPS.
func (l *LeakyBucketLimiter) CurrentRate() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.qps
}

// Stats returns current statistics about the rate limiter.
func (l *LeakyBucketLimiter) Stats() RateLimiterStats {
	acquired := l.totalAcquired.Load()
	rejected := l.totalRejected.Load()
	totalWait := l.totalWaitTime.Load()
	waitCnt := l.waitCount.Load()

	var avgWait time.Duration
	if waitCnt > 0 {
		avgWait = time.Duration(totalWait / waitCnt)
	}

	l.mu.Lock()
	currentQPS := l.qps
	l.mu.Unlock()

	return RateLimiterStats{
		TotalAcquired: acquired,
		TotalRejected: rejected,
		CurrentQPS:    currentQPS,
		AvgWaitTime:   avgWait,
	}
}

// SlidingWindowLimiter implements RateLimiter using the sliding window algorithm.
// It tracks requests in a sliding time window for more accurate rate limiting.
//
// Thread Safety: Safe for concurrent use.
type SlidingWindowLimiter struct {
	qps        float64
	windowSize time.Duration
	timestamps []time.Time
	mu         sync.Mutex

	// Statistics
	totalAcquired atomic.Int64
	totalRejected atomic.Int64
	totalWaitTime atomic.Int64
	waitCount     atomic.Int64
}

// NewSlidingWindowLimiter creates a new sliding window rate limiter.
// windowSize is the duration of the sliding window (typically 1 second).
func NewSlidingWindowLimiter(qps float64, windowSize time.Duration) *SlidingWindowLimiter {
	if qps <= 0 {
		qps = 1
	}
	if windowSize <= 0 {
		windowSize = time.Second
	}
	return &SlidingWindowLimiter{
		qps:        qps,
		windowSize: windowSize,
		timestamps: make([]time.Time, 0, int(qps)+1),
	}
}

// Acquire blocks until a request slot is available or context is cancelled.
func (l *SlidingWindowLimiter) Acquire(ctx context.Context) error {
	start := time.Now()

	for {
		if l.tryAcquireInternal() {
			l.totalAcquired.Add(1)
			l.totalWaitTime.Add(int64(time.Since(start)))
			l.waitCount.Add(1)
			return nil
		}

		// Calculate wait time until oldest request falls out of window
		l.mu.Lock()
		var waitTime time.Duration
		if len(l.timestamps) > 0 {
			oldestExpiry := l.timestamps[0].Add(l.windowSize)
			waitTime = time.Until(oldestExpiry)
			if waitTime < 0 {
				waitTime = time.Millisecond
			}
		} else {
			waitTime = time.Millisecond
		}
		l.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
		}
	}
}

// TryAcquire attempts to acquire a request slot without blocking.
func (l *SlidingWindowLimiter) TryAcquire() bool {
	if l.tryAcquireInternal() {
		l.totalAcquired.Add(1)
		return true
	}
	l.totalRejected.Add(1)
	return false
}

// tryAcquireInternal is the internal implementation without statistics tracking.
func (l *SlidingWindowLimiter) tryAcquireInternal() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-l.windowSize)

	// Remove expired timestamps - find first non-expired timestamp
	validIdx := 0
	for _, ts := range l.timestamps {
		if ts.After(windowStart) {
			break
		}
		validIdx++
	}
	if validIdx > 0 {
		l.timestamps = l.timestamps[validIdx:]
	}

	// Check if we can accept a new request
	maxRequests := max(1, int(l.qps*l.windowSize.Seconds()))

	if len(l.timestamps) >= maxRequests {
		return false
	}

	l.timestamps = append(l.timestamps, now)
	return true
}

// SetRate dynamically adjusts the rate limit.
func (l *SlidingWindowLimiter) SetRate(qps float64) {
	if qps <= 0 {
		qps = 1
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.qps = qps
}

// CurrentRate returns the current rate limit in QPS.
func (l *SlidingWindowLimiter) CurrentRate() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.qps
}

// Stats returns current statistics about the rate limiter.
func (l *SlidingWindowLimiter) Stats() RateLimiterStats {
	acquired := l.totalAcquired.Load()
	rejected := l.totalRejected.Load()
	totalWait := l.totalWaitTime.Load()
	waitCnt := l.waitCount.Load()

	var avgWait time.Duration
	if waitCnt > 0 {
		avgWait = time.Duration(totalWait / waitCnt)
	}

	l.mu.Lock()
	currentQPS := l.qps
	l.mu.Unlock()

	return RateLimiterStats{
		TotalAcquired: acquired,
		TotalRejected: rejected,
		CurrentQPS:    currentQPS,
		AvgWaitTime:   avgWait,
	}
}

// NewRateLimiter creates a new rate limiter based on the configuration.
func NewRateLimiter(config RateLimiterConfig) (RateLimiter, error) {
	switch config.Type {
	case RateLimiterTokenBucket, "":
		// Token bucket is the default
		return NewTokenBucketLimiter(config.QPS, config.BurstSize), nil
	case RateLimiterLeakyBucket:
		return NewLeakyBucketLimiter(config.QPS), nil
	case RateLimiterSlidingWindow:
		return NewSlidingWindowLimiter(config.QPS, time.Second), nil
	default:
		return NewTokenBucketLimiter(config.QPS, config.BurstSize), nil
	}
}
