package loadctrl

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenBucketLimiter_BasicOperation(t *testing.T) {
	limiter := NewTokenBucketLimiter(100, 10)

	// Should be able to acquire
	assert.True(t, limiter.TryAcquire())

	// Current rate should be 100
	assert.Equal(t, 100.0, limiter.CurrentRate())

	// Stats should reflect the acquisition
	stats := limiter.Stats()
	assert.Equal(t, int64(1), stats.TotalAcquired)
	assert.Equal(t, int64(0), stats.TotalRejected)
}

func TestTokenBucketLimiter_SetRate(t *testing.T) {
	limiter := NewTokenBucketLimiter(100, 10)

	// Change rate
	limiter.SetRate(200)
	assert.Equal(t, 200.0, limiter.CurrentRate())

	// Change to very low rate
	limiter.SetRate(0.5)
	assert.Equal(t, 0.5, limiter.CurrentRate())

	// Zero rate should default to 1
	limiter.SetRate(0)
	assert.Equal(t, 1.0, limiter.CurrentRate())
}

func TestTokenBucketLimiter_Acquire(t *testing.T) {
	limiter := NewTokenBucketLimiter(100, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// First acquire should be immediate
	err := limiter.Acquire(ctx)
	assert.NoError(t, err)

	stats := limiter.Stats()
	assert.Equal(t, int64(1), stats.TotalAcquired)
}

func TestTokenBucketLimiter_AcquireWithContext(t *testing.T) {
	// Create a very slow limiter
	limiter := NewTokenBucketLimiter(1, 1)

	// Exhaust the burst
	limiter.TryAcquire()

	// Try to acquire with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := limiter.Acquire(ctx)
	assert.Error(t, err)
	// Error could be context.DeadlineExceeded or rate limiter error
}

func TestTokenBucketLimiter_TryAcquire_Rejection(t *testing.T) {
	// Create limiter with very low rate and burst of 1
	limiter := NewTokenBucketLimiter(1, 1)

	// First should succeed
	assert.True(t, limiter.TryAcquire())

	// Second should fail (no tokens)
	assert.False(t, limiter.TryAcquire())

	stats := limiter.Stats()
	assert.Equal(t, int64(1), stats.TotalAcquired)
	assert.Equal(t, int64(1), stats.TotalRejected)
}

func TestTokenBucketLimiter_BurstSize(t *testing.T) {
	limiter := NewTokenBucketLimiter(10, 5)
	assert.Equal(t, 5, limiter.BurstSize())

	limiter.SetBurst(10)
	assert.Equal(t, 10, limiter.BurstSize())
}

func TestLeakyBucketLimiter_BasicOperation(t *testing.T) {
	limiter := NewLeakyBucketLimiter(100)

	// Should be able to acquire
	assert.True(t, limiter.TryAcquire())

	// Current rate should be 100
	assert.Equal(t, 100.0, limiter.CurrentRate())

	// Stats should reflect the acquisition
	stats := limiter.Stats()
	assert.Equal(t, int64(1), stats.TotalAcquired)
}

func TestLeakyBucketLimiter_SetRate(t *testing.T) {
	limiter := NewLeakyBucketLimiter(100)

	limiter.SetRate(200)
	assert.Equal(t, 200.0, limiter.CurrentRate())
}

func TestLeakyBucketLimiter_Spacing(t *testing.T) {
	limiter := NewLeakyBucketLimiter(100) // 10ms between requests

	// First acquire
	assert.True(t, limiter.TryAcquire())

	// Immediate second should fail
	assert.False(t, limiter.TryAcquire())

	// Wait for interval
	time.Sleep(15 * time.Millisecond)

	// Now should succeed
	assert.True(t, limiter.TryAcquire())
}

func TestSlidingWindowLimiter_BasicOperation(t *testing.T) {
	limiter := NewSlidingWindowLimiter(100, time.Second)

	// Should be able to acquire
	assert.True(t, limiter.TryAcquire())

	// Current rate should be 100
	assert.Equal(t, 100.0, limiter.CurrentRate())

	// Stats should reflect the acquisition
	stats := limiter.Stats()
	assert.Equal(t, int64(1), stats.TotalAcquired)
}

func TestSlidingWindowLimiter_WindowLimit(t *testing.T) {
	// 5 QPS with 1 second window
	limiter := NewSlidingWindowLimiter(5, time.Second)

	// Should be able to acquire 5 times
	for i := 0; i < 5; i++ {
		assert.True(t, limiter.TryAcquire(), "acquire %d should succeed", i)
	}

	// 6th should fail
	assert.False(t, limiter.TryAcquire())

	stats := limiter.Stats()
	assert.Equal(t, int64(5), stats.TotalAcquired)
	assert.Equal(t, int64(1), stats.TotalRejected)
}

func TestNewRateLimiter_Factory(t *testing.T) {
	tests := []struct {
		name   string
		config RateLimiterConfig
	}{
		{
			name: "token_bucket",
			config: RateLimiterConfig{
				Type:      RateLimiterTokenBucket,
				QPS:       100,
				BurstSize: 10,
			},
		},
		{
			name: "leaky_bucket",
			config: RateLimiterConfig{
				Type: RateLimiterLeakyBucket,
				QPS:  100,
			},
		},
		{
			name: "sliding_window",
			config: RateLimiterConfig{
				Type: RateLimiterSlidingWindow,
				QPS:  100,
			},
		},
		{
			name: "empty_type_defaults_to_token_bucket",
			config: RateLimiterConfig{
				QPS:       100,
				BurstSize: 10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter, err := NewRateLimiter(tt.config)
			require.NoError(t, err)
			assert.NotNil(t, limiter)
			assert.Equal(t, 100.0, limiter.CurrentRate())
		})
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	limiter := NewTokenBucketLimiter(1000, 100)

	var wg sync.WaitGroup
	const goroutines = 10
	const iterations = 100

	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range iterations {
				limiter.TryAcquire()
				limiter.CurrentRate()
				limiter.Stats()
			}
		}()
	}

	wg.Wait()

	stats := limiter.Stats()
	assert.True(t, stats.TotalAcquired > 0)
}

func TestRateLimiter_RateAccuracy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping rate accuracy test in short mode")
	}

	const targetQPS = 50.0
	const testDuration = 500 * time.Millisecond

	// Use burst size of 1 to test rate limiting more accurately
	limiter := NewTokenBucketLimiter(targetQPS, 1)

	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	var count int64
	start := time.Now()

	// Use blocking Acquire to properly test rate limiting
	for {
		err := limiter.Acquire(ctx)
		if err != nil {
			break
		}
		count++
	}

	elapsed := time.Since(start)
	actualQPS := float64(count) / elapsed.Seconds()

	// Allow 30% tolerance for CI environments
	tolerance := targetQPS * 0.3
	assert.InDelta(t, targetQPS, actualQPS, tolerance,
		"Expected QPS around %f, got %f (count=%d, elapsed=%v)",
		targetQPS, actualQPS, count, elapsed)
}
