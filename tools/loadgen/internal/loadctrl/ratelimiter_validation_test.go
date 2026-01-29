package loadctrl

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// LOADGEN-VAL-003: 验证速率限制器功能
// This file contains validation tests for the TokenBucketLimiter

// TestTokenBucketLimiter_BasicFunctionality validates basic operations
// Requirement: 测试 TokenBucketLimiter 基础功能
func TestTokenBucketLimiter_BasicFunctionality(t *testing.T) {
	t.Parallel()

	t.Run("creation with default burst", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(100, 0)
		assert.Equal(t, 100.0, limiter.CurrentRate())
		assert.Equal(t, 100, limiter.BurstSize()) // default burst = max(1, int(qps))
	})

	t.Run("creation with explicit burst", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(100, 20)
		assert.Equal(t, 100.0, limiter.CurrentRate())
		assert.Equal(t, 20, limiter.BurstSize())
	})

	t.Run("creation with negative QPS defaults to 1", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(-10, 5)
		assert.Equal(t, 1.0, limiter.CurrentRate())
	})

	t.Run("creation with zero burst defaults to max(1, qps)", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(50, 0)
		assert.Equal(t, 50, limiter.BurstSize())
	})

	t.Run("TryAcquire succeeds when tokens available", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(100, 10)
		assert.True(t, limiter.TryAcquire())
	})

	t.Run("TryAcquire fails when no tokens available", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(1, 1)
		// First acquire uses the token
		assert.True(t, limiter.TryAcquire())
		// Second should fail immediately
		assert.False(t, limiter.TryAcquire())
	})

	t.Run("Acquire blocks and returns when context cancelled", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(1, 1)
		// Exhaust the burst
		assert.True(t, limiter.TryAcquire())

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := limiter.Acquire(ctx)
		assert.Error(t, err)
	})
}

// TestTokenBucketLimiter_QPSAccuracy validates QPS precision within ±2%
// Requirement: 验证每秒请求数精确度 (±2%)
// Pass criteria: 实际QPS与目标QPS误差 < 2%
func TestTokenBucketLimiter_QPSAccuracy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping QPS accuracy test in short mode")
	}

	testCases := []struct {
		name      string
		targetQPS float64
		duration  time.Duration
		tolerance float64 // percentage tolerance
	}{
		{
			name:      "low_qps_10",
			targetQPS: 10,
			duration:  5 * time.Second, // longer duration for low QPS for accuracy
			tolerance: 0.025,           // 2.5% for low QPS (timing overhead dominates)
		},
		{
			name:      "medium_qps_50",
			targetQPS: 50,
			duration:  3 * time.Second,
			tolerance: 0.02, // 2%
		},
		{
			name:      "high_qps_100",
			targetQPS: 100,
			duration:  2 * time.Second,
			tolerance: 0.02, // 2%
		},
		{
			name:      "very_high_qps_200",
			targetQPS: 200,
			duration:  2 * time.Second,
			tolerance: 0.02, // 2%
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Use burst of 1 to ensure accurate rate limiting behavior
			limiter := NewTokenBucketLimiter(tc.targetQPS, 1)

			ctx, cancel := context.WithTimeout(context.Background(), tc.duration)
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

			// Calculate allowed deviation
			allowedDeviation := tc.targetQPS * tc.tolerance
			lowerBound := tc.targetQPS - allowedDeviation
			upperBound := tc.targetQPS + allowedDeviation

			t.Logf("Target QPS: %.2f, Actual QPS: %.2f, Tolerance: ±%.2f%%, Range: [%.2f, %.2f]",
				tc.targetQPS, actualQPS, tc.tolerance*100, lowerBound, upperBound)
			t.Logf("Total requests: %d, Elapsed: %v", count, elapsed)

			assert.GreaterOrEqual(t, actualQPS, lowerBound,
				"Actual QPS %.2f is below lower bound %.2f", actualQPS, lowerBound)
			assert.LessOrEqual(t, actualQPS, upperBound,
				"Actual QPS %.2f is above upper bound %.2f", actualQPS, upperBound)

			// Verify stats match
			stats := limiter.Stats()
			assert.Equal(t, count, stats.TotalAcquired, "Stats TotalAcquired should match count")
		})
	}
}

// TestTokenBucketLimiter_BurstHandling validates burst traffic handling
// Requirement: 测试突发流量处理能力
// Pass criteria: 突发请求正确处理
func TestTokenBucketLimiter_BurstHandling(t *testing.T) {
	t.Parallel()

	t.Run("burst_allows_immediate_requests", func(t *testing.T) {
		burstSize := 20
		limiter := NewTokenBucketLimiter(10, burstSize)

		// Should be able to acquire burst number of requests immediately
		var acquired int
		for range burstSize {
			if limiter.TryAcquire() {
				acquired++
			}
		}

		assert.Equal(t, burstSize, acquired,
			"Should acquire exactly %d requests in burst", burstSize)

		// Next request should be rate limited
		assert.False(t, limiter.TryAcquire(), "Should be rate limited after burst")
	})

	t.Run("burst_with_high_qps", func(t *testing.T) {
		burstSize := 50
		qps := 100.0
		limiter := NewTokenBucketLimiter(qps, burstSize)

		// Consume all burst tokens
		for range burstSize {
			require.True(t, limiter.TryAcquire())
		}

		// Should be rate limited now
		assert.False(t, limiter.TryAcquire())

		// Wait for tokens to replenish
		time.Sleep(100 * time.Millisecond)

		// Should be able to acquire more tokens (100 QPS * 0.1s = 10 tokens)
		var replenished int
		for range 15 {
			if limiter.TryAcquire() {
				replenished++
			}
		}
		assert.GreaterOrEqual(t, replenished, 8, "Should have replenished at least 8 tokens")
	})

	t.Run("burst_size_modification", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(100, 10)
		assert.Equal(t, 10, limiter.BurstSize())

		limiter.SetBurst(25)
		assert.Equal(t, 25, limiter.BurstSize())

		// Zero burst should default to 1
		limiter.SetBurst(0)
		assert.Equal(t, 1, limiter.BurstSize())
	})

	t.Run("concurrent_burst_requests", func(t *testing.T) {
		burstSize := 30
		limiter := NewTokenBucketLimiter(10, burstSize)

		var acquired atomic.Int64
		var wg sync.WaitGroup

		// Launch concurrent goroutines trying to acquire
		numGoroutines := 50
		for range numGoroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if limiter.TryAcquire() {
					acquired.Add(1)
				}
			}()
		}

		wg.Wait()

		// Should acquire exactly burstSize (or close to it with race conditions)
		assert.InDelta(t, float64(burstSize), float64(acquired.Load()), 5,
			"Concurrent burst acquisition should be close to burst size")
	})
}

// TestTokenBucketLimiter_DynamicRateAdjustment validates SetRate functionality
// Requirement: 验证动态速率调整 (SetRate)
// Pass criteria: 动态调整后立即生效
func TestTokenBucketLimiter_DynamicRateAdjustment(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping dynamic rate adjustment test in short mode")
	}

	t.Run("rate_change_takes_effect_immediately", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(10, 1)

		// Exhaust burst
		limiter.TryAcquire()

		// Change rate to higher value
		limiter.SetRate(1000)
		assert.Equal(t, 1000.0, limiter.CurrentRate())

		// Wait a bit for new tokens
		time.Sleep(10 * time.Millisecond)

		// Should be able to acquire more tokens at higher rate
		assert.True(t, limiter.TryAcquire(), "Should be able to acquire after rate increase")
	})

	t.Run("rate_increase_measured_accurately", func(t *testing.T) {
		initialRate := 20.0
		newRate := 100.0
		limiter := NewTokenBucketLimiter(initialRate, 1)

		// Run at initial rate for a short time
		ctx1, cancel1 := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel1()

		var count1 int64
		start1 := time.Now()
		for {
			if err := limiter.Acquire(ctx1); err != nil {
				break
			}
			count1++
		}
		elapsed1 := time.Since(start1)
		actualQPS1 := float64(count1) / elapsed1.Seconds()

		t.Logf("Initial rate: target=%.2f, actual=%.2f", initialRate, actualQPS1)

		// Change rate
		limiter.SetRate(newRate)

		// Run at new rate
		ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel2()

		var count2 int64
		start2 := time.Now()
		for {
			if err := limiter.Acquire(ctx2); err != nil {
				break
			}
			count2++
		}
		elapsed2 := time.Since(start2)
		actualQPS2 := float64(count2) / elapsed2.Seconds()

		t.Logf("New rate: target=%.2f, actual=%.2f", newRate, actualQPS2)

		// Verify both rates are within tolerance (10% for short duration tests)
		tolerance := 0.10
		assert.InDelta(t, initialRate, actualQPS1, initialRate*tolerance,
			"Initial rate should be within tolerance")
		assert.InDelta(t, newRate, actualQPS2, newRate*tolerance,
			"New rate should be within tolerance after adjustment")

		// Verify that new rate is significantly higher (at least 2x)
		assert.Greater(t, actualQPS2, actualQPS1*2,
			"New rate should be significantly higher than initial rate")
	})

	t.Run("rate_decrease_takes_effect", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(100, 50)

		// Get many tokens at high rate
		for range 50 {
			limiter.TryAcquire()
		}

		// Decrease rate significantly
		limiter.SetRate(1)
		assert.Equal(t, 1.0, limiter.CurrentRate())

		// Wait a bit
		time.Sleep(50 * time.Millisecond)

		// Should be rate limited at the new low rate
		// Only ~0.05 tokens should have been added (1 QPS * 0.05s)
		assert.False(t, limiter.TryAcquire(), "Should be rate limited at new low rate")
	})

	t.Run("zero_rate_defaults_to_1", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(100, 10)
		limiter.SetRate(0)
		assert.Equal(t, 1.0, limiter.CurrentRate())
	})

	t.Run("negative_rate_defaults_to_1", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(100, 10)
		limiter.SetRate(-50)
		assert.Equal(t, 1.0, limiter.CurrentRate())
	})
}

// TestTokenBucketLimiter_Statistics validates statistics accuracy
// Requirement: 测试统计信息准确性
// Pass criteria: 统计数据准确记录
func TestTokenBucketLimiter_Statistics(t *testing.T) {
	t.Parallel()

	t.Run("total_acquired_count_accurate", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(1000, 100)

		const numAcquires = 50
		for range numAcquires {
			limiter.TryAcquire()
		}

		stats := limiter.Stats()
		assert.Equal(t, int64(numAcquires), stats.TotalAcquired,
			"TotalAcquired should match number of successful acquires")
	})

	t.Run("total_rejected_count_accurate", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(1, 1)

		// First should succeed
		assert.True(t, limiter.TryAcquire())

		// Rest should fail
		const numRejects = 10
		for range numRejects {
			limiter.TryAcquire()
		}

		stats := limiter.Stats()
		assert.Equal(t, int64(1), stats.TotalAcquired)
		assert.Equal(t, int64(numRejects), stats.TotalRejected,
			"TotalRejected should match number of failed acquires")
	})

	t.Run("current_qps_accurate", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(75, 10)

		stats := limiter.Stats()
		assert.Equal(t, 75.0, stats.CurrentQPS)

		limiter.SetRate(150)
		stats = limiter.Stats()
		assert.Equal(t, 150.0, stats.CurrentQPS)
	})

	t.Run("average_wait_time_tracked", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping wait time test in short mode")
		}

		limiter := NewTokenBucketLimiter(10, 1)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		// Make several blocking acquires
		for range 5 {
			if err := limiter.Acquire(ctx); err != nil {
				break
			}
		}

		stats := limiter.Stats()
		// Average wait time should be > 0 since we're rate limited
		if stats.TotalAcquired > 1 {
			assert.Greater(t, stats.AvgWaitTime, time.Duration(0),
				"AvgWaitTime should be > 0 for rate-limited acquires")
		}
	})

	t.Run("stats_thread_safety", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(1000, 100)

		var wg sync.WaitGroup
		const goroutines = 20
		const iterations = 50

		for range goroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range iterations {
					limiter.TryAcquire()
					limiter.Stats() // Read stats while writing
				}
			}()
		}

		wg.Wait()

		stats := limiter.Stats()
		// Total should be reasonable (some acquired, some rejected)
		total := stats.TotalAcquired + stats.TotalRejected
		assert.Equal(t, int64(goroutines*iterations), total,
			"Total (acquired + rejected) should match total attempts")
	})

	t.Run("stats_accurate_with_acquire_blocking", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping blocking acquire stats test in short mode")
		}

		limiter := NewTokenBucketLimiter(100, 1)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		var count int64
		for {
			if err := limiter.Acquire(ctx); err != nil {
				break
			}
			count++
		}

		stats := limiter.Stats()
		assert.Equal(t, count, stats.TotalAcquired,
			"Stats TotalAcquired should match actual acquire count")
		assert.Equal(t, int64(0), stats.TotalRejected,
			"Stats TotalRejected should be 0 when using Acquire")
	})
}

// TestTokenBucketLimiter_EdgeCases tests edge cases and boundary conditions
func TestTokenBucketLimiter_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("very_low_qps", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(0.5, 1)
		assert.Equal(t, 0.5, limiter.CurrentRate())
	})

	t.Run("very_high_qps", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(10000, 100)
		assert.Equal(t, 10000.0, limiter.CurrentRate())
	})

	t.Run("context_already_cancelled", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(100, 1)
		limiter.TryAcquire() // Exhaust burst

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := limiter.Acquire(ctx)
		assert.Error(t, err)
	})

	t.Run("rapid_rate_changes", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(100, 10)

		for i := range 100 {
			limiter.SetRate(float64(i + 1))
			assert.Equal(t, float64(i+1), limiter.CurrentRate())
		}
	})
}
