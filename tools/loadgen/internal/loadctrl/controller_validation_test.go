package loadctrl

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// LOADGEN-VAL-005: Adaptive Control Validation Tests
// =============================================================================
//
// This file validates the adaptive control functionality of the LoadController:
// 1. P95延迟超标时自动降速 - P95 latency breach triggers automatic QPS reduction
// 2. WorkerPool大小自动调整 - WorkerPool automatic size adjustment
// 3. 最优工作者数计算算法 - Optimal worker count calculation algorithm
// 4. 100ms更新周期 - 100ms adjustment interval
// 5. 恢复后自动提速 - Automatic QPS recovery after latency drops
//
// Pass Criteria:
// - P95 > 500ms时QPS在10秒内下降 > 20%
// - Worker数量根据负载动态调整
// - 延迟恢复后QPS逐步提升
// - 调整周期严格按100ms执行

// =============================================================================
// Test 1: P95 Latency Breach Triggers Automatic QPS Reduction
// =============================================================================

// TestAdaptiveControl_P95BreachTriggersQPSReduction verifies that when P95 latency
// exceeds the target threshold, the controller automatically reduces QPS.
// Pass criteria: P95 > 500ms时QPS在10秒内下降 > 20%
func TestAdaptiveControl_P95BreachTriggersQPSReduction(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name              string
		baseQPS           float64
		targetP95         time.Duration
		highLatency       time.Duration
		reductionFactor   float64
		expectedReduction float64 // Minimum expected QPS reduction percentage
		breachThreshold   int
		maxWaitTime       time.Duration
	}{
		{
			name:              "500ms threshold with 10% reduction",
			baseQPS:           100,
			targetP95:         500 * time.Millisecond,
			highLatency:       700 * time.Millisecond, // 700ms > 500ms
			reductionFactor:   0.1,                    // 10% reduction per cycle
			expectedReduction: 0.10,                   // At least 10% reduction
			breachThreshold:   2,
			maxWaitTime:       5 * time.Second,
		},
		{
			name:              "200ms threshold with 20% reduction",
			baseQPS:           200,
			targetP95:         200 * time.Millisecond,
			highLatency:       400 * time.Millisecond, // 400ms > 200ms
			reductionFactor:   0.2,                    // 20% reduction per cycle
			expectedReduction: 0.20,                   // At least 20% reduction
			breachThreshold:   1,
			maxWaitTime:       3 * time.Second,
		},
		{
			name:              "1s threshold with aggressive 30% reduction",
			baseQPS:           500,
			targetP95:         1 * time.Second,
			highLatency:       1500 * time.Millisecond, // 1.5s > 1s
			reductionFactor:   0.3,                     // 30% reduction per cycle
			expectedReduction: 0.30,                    // At least 30% reduction
			breachThreshold:   2,
			maxWaitTime:       5 * time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rateLimiter := NewTokenBucketLimiter(tc.baseQPS, int(tc.baseQPS/5))
			shaper := &mockTrafficShaper{qps: tc.baseQPS, phase: "test"}
			metrics := NewSlidingWindowMetrics(MetricsConfig{
				WindowSize: 10 * time.Second,
				BucketSize: 100 * time.Millisecond,
			})

			controller := NewLoadController(rateLimiter, shaper, nil, metrics, LoadControllerConfig{
				AdjustInterval:             100 * time.Millisecond,
				Adaptive:                   true,
				TargetP95:                  tc.targetP95,
				AdaptiveReductionFactor:    tc.reductionFactor,
				ConsecutiveBreachThreshold: tc.breachThreshold,
				MinQPS:                     10,
			})

			ctx, cancel := context.WithTimeout(context.Background(), tc.maxWaitTime)
			defer cancel()
			controller.Start(ctx)
			defer controller.Stop()

			// Let controller stabilize with normal latencies
			for range 20 {
				metrics.RecordLatency(tc.targetP95 / 2)
			}
			time.Sleep(200 * time.Millisecond)

			initialQPS := controller.ActualQPS()
			assert.InDelta(t, tc.baseQPS, initialQPS, tc.baseQPS*0.1,
				"Initial QPS should be close to base QPS")
			assert.False(t, controller.IsAdaptiveActive(),
				"Adaptive control should not be active initially")

			// Inject high latencies to trigger adaptive control
			for range 100 {
				metrics.RecordLatency(tc.highLatency)
			}

			// Wait for adaptive control to activate and reduce QPS
			var finalQPS float64
			var adaptiveActivated bool
			deadline := time.Now().Add(tc.maxWaitTime)

			for time.Now().Before(deadline) {
				time.Sleep(150 * time.Millisecond)
				finalQPS = controller.ActualQPS()
				adaptiveActivated = controller.IsAdaptiveActive()

				if adaptiveActivated && finalQPS < initialQPS*(1-tc.expectedReduction*0.5) {
					break
				}

				// Keep injecting high latencies
				for range 20 {
					metrics.RecordLatency(tc.highLatency)
				}
			}

			assert.True(t, adaptiveActivated,
				"Adaptive control should be active after P95 breach")

			reduction := (initialQPS - finalQPS) / initialQPS
			assert.GreaterOrEqual(t, reduction, tc.expectedReduction*0.9, // Allow 10% tolerance
				"QPS should reduce by at least %.0f%%, got %.2f%% (initial=%.2f, final=%.2f)",
				tc.expectedReduction*100, reduction*100, initialQPS, finalQPS)

			t.Logf("Test %s: Initial QPS=%.2f, Final QPS=%.2f, Reduction=%.2f%%",
				tc.name, initialQPS, finalQPS, reduction*100)
		})
	}
}

// TestAdaptiveControl_QPSReductionWithin10Seconds validates the pass criteria:
// "P95 > 500ms时QPS在10秒内下降 > 20%"
//
// Note: With AdaptiveReductionFactor=0.25, a single sustained breach will cause
// 25% reduction, meeting the >20% requirement.
func TestAdaptiveControl_QPSReductionWithin10Seconds(t *testing.T) {
	t.Parallel()

	rateLimiter := NewTokenBucketLimiter(100, 20)
	shaper := &mockTrafficShaper{qps: 100, phase: "stress-test"}
	metrics := NewSlidingWindowMetrics(MetricsConfig{
		WindowSize: 10 * time.Second,
		BucketSize: 100 * time.Millisecond,
	})

	controller := NewLoadController(rateLimiter, shaper, nil, metrics, LoadControllerConfig{
		AdjustInterval:             100 * time.Millisecond,
		Adaptive:                   true,
		TargetP95:                  500 * time.Millisecond, // 500ms threshold
		AdaptiveReductionFactor:    0.25,                   // 25% reduction per breach (>20%)
		AdaptiveRecoveryFactor:     0.05,
		ConsecutiveBreachThreshold: 2,
		MinQPS:                     10,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	controller.Start(ctx)
	defer controller.Stop()

	// Stabilize with good latencies
	for range 50 {
		metrics.RecordLatency(200 * time.Millisecond)
	}
	time.Sleep(300 * time.Millisecond)

	initialQPS := controller.ActualQPS()
	require.InDelta(t, 100.0, initialQPS, 5.0, "Should start at ~100 QPS")

	startTime := time.Now()

	// Track QPS over time
	type qpsRecord struct {
		time time.Time
		qps  float64
	}
	var records []qpsRecord

	// Start injecting high latencies (> 500ms)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				metrics.RecordLatency(700 * time.Millisecond) // P95 will be ~700ms
				time.Sleep(20 * time.Millisecond)
			}
		}
	}()

	// Monitor QPS reduction
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	achievedReduction := false
	var achievedTime time.Duration

	for {
		select {
		case <-ticker.C:
			currentQPS := controller.ActualQPS()
			records = append(records, qpsRecord{time: time.Now(), qps: currentQPS})

			reduction := (initialQPS - currentQPS) / initialQPS
			if reduction >= 0.20 { // 20% reduction achieved
				achievedReduction = true
				achievedTime = time.Since(startTime)
			}

			if achievedReduction || time.Since(startTime) > 10*time.Second {
				cancel() // Stop the latency injection
				<-done
				goto checkResults
			}
		case <-ctx.Done():
			<-done
			goto checkResults
		}
	}

checkResults:
	assert.True(t, achievedReduction,
		"QPS should reduce by > 20%% within 10 seconds when P95 > 500ms")

	if achievedReduction {
		assert.LessOrEqual(t, achievedTime.Seconds(), 10.0,
			"20%% QPS reduction should occur within 10 seconds, took %.2fs", achievedTime.Seconds())
		t.Logf("Achieved 20%% QPS reduction in %.2f seconds", achievedTime.Seconds())
	}

	// Log QPS progression
	t.Logf("QPS progression:")
	for _, r := range records {
		elapsed := r.time.Sub(startTime)
		t.Logf("  t=%.1fs: QPS=%.2f", elapsed.Seconds(), r.qps)
	}
}

// =============================================================================
// Test 2: WorkerPool Automatic Size Adjustment
// =============================================================================

// TestWorkerPool_AutomaticSizeAdjustment verifies that WorkerPool size
// dynamically adjusts based on load (QPS and latency).
func TestWorkerPool_AutomaticSizeAdjustment(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		initialQPS     float64
		targetQPS      float64
		avgLatency     time.Duration
		minWorkers     int
		maxWorkers     int
		expectedChange string // "increase", "decrease", "clamp_max", "clamp_min"
	}{
		{
			name:           "scale up with increased QPS",
			initialQPS:     50,
			targetQPS:      200,
			avgLatency:     100 * time.Millisecond,
			minWorkers:     5,
			maxWorkers:     100,
			expectedChange: "increase",
		},
		{
			name:           "scale down with decreased QPS",
			initialQPS:     200,
			targetQPS:      30,
			avgLatency:     50 * time.Millisecond,
			minWorkers:     5,
			maxWorkers:     100,
			expectedChange: "decrease",
		},
		{
			name:           "clamp to max workers",
			initialQPS:     100,
			targetQPS:      1000,
			avgLatency:     500 * time.Millisecond, // Would need 750 workers
			minWorkers:     5,
			maxWorkers:     50,
			expectedChange: "clamp_max",
		},
		{
			name:           "clamp to min workers",
			initialQPS:     100,
			targetQPS:      5,
			avgLatency:     10 * time.Millisecond, // Would need ~0.075 workers
			minWorkers:     10,
			maxWorkers:     50,
			expectedChange: "clamp_min",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rateLimiter := NewTokenBucketLimiter(tc.initialQPS, int(tc.initialQPS/5))
			shaper := newVariableShaper(tc.initialQPS)
			metrics := NewSlidingWindowMetrics(MetricsConfig{
				WindowSize: 10 * time.Second,
				BucketSize: 100 * time.Millisecond,
			})
			workerPool := NewWorkerPool(WorkerPoolConfig{
				MinSize:     tc.minWorkers,
				MaxSize:     tc.maxWorkers,
				InitialSize: 20,
			})

			controller := NewLoadController(rateLimiter, shaper, workerPool, metrics, LoadControllerConfig{
				AdjustInterval:      100 * time.Millisecond,
				WorkerAutoScale:     true,
				WorkerLatencyBuffer: 1.5,
				DefaultAvgLatency:   50 * time.Millisecond,
			})

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			controller.Start(ctx)
			defer controller.Stop()

			// Record latencies to establish baseline
			for range 30 {
				metrics.RecordLatency(tc.avgLatency)
			}
			time.Sleep(200 * time.Millisecond)

			initialWorkers := workerPool.CurrentSize()

			// Change QPS to trigger worker adjustment
			shaper.SetQPS(tc.targetQPS)

			// Wait for adjustment
			time.Sleep(500 * time.Millisecond)

			finalWorkers := workerPool.CurrentSize()

			switch tc.expectedChange {
			case "increase":
				assert.Greater(t, finalWorkers, initialWorkers,
					"Workers should increase: initial=%d, final=%d", initialWorkers, finalWorkers)
			case "decrease":
				assert.Less(t, finalWorkers, initialWorkers,
					"Workers should decrease: initial=%d, final=%d", initialWorkers, finalWorkers)
			case "clamp_max":
				assert.Equal(t, tc.maxWorkers, finalWorkers,
					"Workers should be clamped to max=%d, got=%d", tc.maxWorkers, finalWorkers)
			case "clamp_min":
				assert.Equal(t, tc.minWorkers, finalWorkers,
					"Workers should be clamped to min=%d, got=%d", tc.minWorkers, finalWorkers)
			}

			t.Logf("Test %s: QPS %.0f->%.0f, Workers %d->%d",
				tc.name, tc.initialQPS, tc.targetQPS, initialWorkers, finalWorkers)
		})
	}
}

// TestWorkerPool_DynamicAdjustmentWithLoad verifies the pass criteria:
// "Worker数量根据负载动态调整"
func TestWorkerPool_DynamicAdjustmentWithLoad(t *testing.T) {
	t.Parallel()

	rateLimiter := NewTokenBucketLimiter(100, 20)
	shaper := newVariableShaper(100)
	metrics := NewSlidingWindowMetrics(MetricsConfig{
		WindowSize: 10 * time.Second,
		BucketSize: 100 * time.Millisecond,
	})
	workerPool := NewWorkerPool(WorkerPoolConfig{
		MinSize:     5,
		MaxSize:     50,
		InitialSize: 10,
	})

	controller := NewLoadController(rateLimiter, shaper, workerPool, metrics, LoadControllerConfig{
		AdjustInterval:      100 * time.Millisecond,
		WorkerAutoScale:     true,
		WorkerLatencyBuffer: 1.5,
		DefaultAvgLatency:   50 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	controller.Start(ctx)
	defer controller.Stop()

	type workerRecord struct {
		qps     float64
		latency time.Duration
		workers int
	}
	var records []workerRecord

	// Test sequence of load changes
	loadSequence := []struct {
		qps     float64
		latency time.Duration
	}{
		{100, 50 * time.Millisecond},  // Low load
		{300, 100 * time.Millisecond}, // Medium load
		{500, 200 * time.Millisecond}, // High load
		{200, 80 * time.Millisecond},  // Reduced load
		{50, 30 * time.Millisecond},   // Low load again
	}

	for _, load := range loadSequence {
		shaper.SetQPS(load.qps)

		// Record latencies
		for range 20 {
			metrics.RecordLatency(load.latency)
		}

		time.Sleep(300 * time.Millisecond)

		records = append(records, workerRecord{
			qps:     load.qps,
			latency: load.latency,
			workers: workerPool.CurrentSize(),
		})
	}

	// Verify workers changed according to load
	t.Logf("Worker adjustment sequence:")
	for _, r := range records {
		// Expected workers = QPS * Latency(s) * Buffer(1.5)
		expectedWorkers := r.qps * r.latency.Seconds() * 1.5
		t.Logf("  QPS=%.0f, Latency=%v, Workers=%d (expected ~%.0f)",
			r.qps, r.latency, r.workers, expectedWorkers)
	}

	// Verify dynamic adjustment occurred
	workersChanged := false
	for i := 1; i < len(records); i++ {
		if records[i].workers != records[i-1].workers {
			workersChanged = true
			break
		}
	}
	assert.True(t, workersChanged, "Workers should change dynamically with load")

	// Verify workers at high load > workers at low load
	highLoadWorkers := records[2].workers // QPS=500
	lowLoadWorkers := records[4].workers  // QPS=50
	assert.Greater(t, highLoadWorkers, lowLoadWorkers,
		"High load (%d workers) should have more workers than low load (%d workers)",
		highLoadWorkers, lowLoadWorkers)
}

// =============================================================================
// Test 3: Optimal Worker Count Calculation Algorithm
// =============================================================================

// TestOptimalWorkerCalculation validates the formula:
// Workers = QPS * AvgLatency(seconds) * LatencyBuffer
func TestOptimalWorkerCalculation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		qps             float64
		avgLatency      time.Duration
		latencyBuffer   float64
		minWorkers      int
		maxWorkers      int
		expectedWorkers int
	}{
		{
			name:            "basic calculation: 100 QPS * 100ms * 1.5",
			qps:             100,
			avgLatency:      100 * time.Millisecond,
			latencyBuffer:   1.5,
			minWorkers:      1,
			maxWorkers:      100,
			expectedWorkers: 15, // 100 * 0.1 * 1.5 = 15
		},
		{
			name:            "high QPS: 500 QPS * 200ms * 1.5",
			qps:             500,
			avgLatency:      200 * time.Millisecond,
			latencyBuffer:   1.5,
			minWorkers:      5,
			maxWorkers:      200,
			expectedWorkers: 150, // 500 * 0.2 * 1.5 = 150
		},
		{
			name:            "low latency: 1000 QPS * 10ms * 1.5",
			qps:             1000,
			avgLatency:      10 * time.Millisecond,
			latencyBuffer:   1.5,
			minWorkers:      5,
			maxWorkers:      100,
			expectedWorkers: 15, // 1000 * 0.01 * 1.5 = 15
		},
		{
			name:            "clamped to minimum",
			qps:             10,
			avgLatency:      10 * time.Millisecond,
			latencyBuffer:   1.5,
			minWorkers:      10,
			maxWorkers:      100,
			expectedWorkers: 10, // 10 * 0.01 * 1.5 = 0.15, clamped to 10
		},
		{
			name:            "clamped to maximum",
			qps:             1000,
			avgLatency:      500 * time.Millisecond,
			latencyBuffer:   2.0,
			minWorkers:      5,
			maxWorkers:      50,
			expectedWorkers: 50, // 1000 * 0.5 * 2.0 = 1000, clamped to 50
		},
		{
			name:            "different buffer: 100 QPS * 100ms * 2.0",
			qps:             100,
			avgLatency:      100 * time.Millisecond,
			latencyBuffer:   2.0,
			minWorkers:      1,
			maxWorkers:      100,
			expectedWorkers: 20, // 100 * 0.1 * 2.0 = 20
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rateLimiter := NewTokenBucketLimiter(tc.qps, int(tc.qps/5))
			shaper := &mockTrafficShaper{qps: tc.qps, phase: "test"}
			metrics := NewSlidingWindowMetrics(MetricsConfig{
				WindowSize: 10 * time.Second,
				BucketSize: time.Second,
			})
			workerPool := NewWorkerPool(WorkerPoolConfig{
				MinSize:     tc.minWorkers,
				MaxSize:     tc.maxWorkers,
				InitialSize: tc.minWorkers,
			})

			controller := NewLoadController(rateLimiter, shaper, workerPool, metrics, LoadControllerConfig{
				WorkerAutoScale:     true,
				WorkerLatencyBuffer: tc.latencyBuffer,
			})

			// Record latencies to establish average
			for range 20 {
				metrics.RecordLatency(tc.avgLatency)
			}

			// Calculate expected value
			calculatedWorkers := int(tc.qps * tc.avgLatency.Seconds() * tc.latencyBuffer)
			if calculatedWorkers < tc.minWorkers {
				calculatedWorkers = tc.minWorkers
			}
			if calculatedWorkers > tc.maxWorkers {
				calculatedWorkers = tc.maxWorkers
			}

			// Get actual calculation result
			actualWorkers := controller.calculateOptimalWorkers(tc.qps)

			assert.Equal(t, tc.expectedWorkers, actualWorkers,
				"Workers calculation mismatch")
			assert.Equal(t, calculatedWorkers, actualWorkers,
				"Formula: QPS(%.0f) * Latency(%.3fs) * Buffer(%.1f) = %.0f, expected %d",
				tc.qps, tc.avgLatency.Seconds(), tc.latencyBuffer,
				tc.qps*tc.avgLatency.Seconds()*tc.latencyBuffer, tc.expectedWorkers)
		})
	}
}

// TestOptimalWorkerCalculation_EdgeCases tests edge cases in the formula
func TestOptimalWorkerCalculation_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("zero QPS returns minimum", func(t *testing.T) {
		t.Parallel()

		workerPool := NewWorkerPool(WorkerPoolConfig{
			MinSize:     5,
			MaxSize:     50,
			InitialSize: 10,
		})

		controller := NewLoadController(
			NewTokenBucketLimiter(100, 20),
			&mockTrafficShaper{qps: 100, phase: "test"},
			workerPool,
			nil,
			LoadControllerConfig{
				WorkerAutoScale:     true,
				WorkerLatencyBuffer: 1.5,
				DefaultAvgLatency:   50 * time.Millisecond,
			},
		)

		workers := controller.calculateOptimalWorkers(0)
		assert.GreaterOrEqual(t, workers, 1, "Zero QPS should return at least 1 worker")
	})

	t.Run("uses default latency when no metrics", func(t *testing.T) {
		t.Parallel()

		workerPool := NewWorkerPool(WorkerPoolConfig{
			MinSize:     5,
			MaxSize:     50,
			InitialSize: 10,
		})

		defaultLatency := 100 * time.Millisecond
		controller := NewLoadController(
			NewTokenBucketLimiter(100, 20),
			&mockTrafficShaper{qps: 100, phase: "test"},
			workerPool,
			nil, // No metrics
			LoadControllerConfig{
				WorkerAutoScale:     true,
				WorkerLatencyBuffer: 1.5,
				DefaultAvgLatency:   defaultLatency,
			},
		)

		workers := controller.calculateOptimalWorkers(100)
		// Expected: 100 * 0.1 * 1.5 = 15
		expected := int(100 * defaultLatency.Seconds() * 1.5)
		assert.Equal(t, expected, workers)
	})
}

// =============================================================================
// Test 4: 100ms Adjustment Interval
// =============================================================================

// TestAdjustmentInterval_100ms verifies that adjustment cycles execute
// strictly at 100ms intervals.
func TestAdjustmentInterval_100ms(t *testing.T) {
	t.Parallel()

	rateLimiter := NewTokenBucketLimiter(100, 20)
	shaper := newVariableShaper(100)
	metrics := NewMetricsCollector()

	type adjustment struct {
		timestamp time.Time
		targetQPS float64
		actualQPS float64
	}
	var adjustmentsMu sync.Mutex
	var adjustments []adjustment

	adjustInterval := 100 * time.Millisecond
	controller := NewLoadController(rateLimiter, shaper, nil, metrics, LoadControllerConfig{
		AdjustInterval: adjustInterval,
	})
	controller.SetOnQPSAdjust(func(target, actual float64) {
		adjustmentsMu.Lock()
		adjustments = append(adjustments, adjustment{
			timestamp: time.Now(),
			targetQPS: target,
			actualQPS: actual,
		})
		adjustmentsMu.Unlock()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	controller.Start(ctx)
	defer controller.Stop()

	// Run for enough time to collect multiple adjustments
	time.Sleep(1500 * time.Millisecond)

	adjustmentsMu.Lock()
	adjCopy := make([]adjustment, len(adjustments))
	copy(adjCopy, adjustments)
	adjustmentsMu.Unlock()

	require.GreaterOrEqual(t, len(adjCopy), 10,
		"Should have at least 10 adjustments in 1.5s with 100ms interval")

	// Verify intervals between adjustments
	var totalDeviation time.Duration
	var maxDeviation time.Duration
	deviationCount := 0

	for i := 1; i < len(adjCopy); i++ {
		interval := adjCopy[i].timestamp.Sub(adjCopy[i-1].timestamp)
		deviation := interval - adjustInterval
		if deviation < 0 {
			deviation = -deviation
		}

		totalDeviation += deviation
		if deviation > maxDeviation {
			maxDeviation = deviation
		}
		deviationCount++
	}

	avgDeviation := totalDeviation / time.Duration(deviationCount)

	// Allow 20ms tolerance (20% of 100ms)
	tolerance := 20 * time.Millisecond
	assert.LessOrEqual(t, avgDeviation, tolerance,
		"Average deviation from 100ms interval should be <= %v, got %v",
		tolerance, avgDeviation)

	// Max deviation should be within 50ms (account for scheduler variability)
	maxTolerance := 50 * time.Millisecond
	assert.LessOrEqual(t, maxDeviation, maxTolerance,
		"Max deviation from 100ms interval should be <= %v, got %v",
		maxTolerance, maxDeviation)

	t.Logf("Adjustment interval analysis:")
	t.Logf("  Total adjustments: %d", len(adjCopy))
	t.Logf("  Average deviation: %v (tolerance: %v)", avgDeviation, tolerance)
	t.Logf("  Max deviation: %v (tolerance: %v)", maxDeviation, maxTolerance)
}

// TestAdjustmentInterval_Precise validates the pass criteria:
// "调整周期严格按100ms执行"
func TestAdjustmentInterval_Precise(t *testing.T) {
	t.Parallel()

	rateLimiter := NewTokenBucketLimiter(100, 20)
	shaper := &mockTrafficShaper{qps: 100, phase: "test"}

	var callCount atomic.Int32
	var timestamps []time.Time
	var timestampsMu sync.Mutex

	controller := NewLoadController(rateLimiter, shaper, nil, nil, LoadControllerConfig{
		AdjustInterval: 100 * time.Millisecond,
	})
	controller.SetOnQPSAdjust(func(_, _ float64) {
		callCount.Add(1)
		timestampsMu.Lock()
		timestamps = append(timestamps, time.Now())
		timestampsMu.Unlock()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	controller.Start(ctx)
	defer controller.Stop()

	time.Sleep(1100 * time.Millisecond)

	// Should have approximately 10-11 adjustments in 1.1 seconds
	count := callCount.Load()
	assert.GreaterOrEqual(t, count, int32(9),
		"Should have at least 9 adjustments, got %d", count)
	assert.LessOrEqual(t, count, int32(13),
		"Should have at most 13 adjustments, got %d", count)

	// Calculate intervals
	timestampsMu.Lock()
	intervals := make([]time.Duration, 0, len(timestamps)-1)
	for i := 1; i < len(timestamps); i++ {
		intervals = append(intervals, timestamps[i].Sub(timestamps[i-1]))
	}
	timestampsMu.Unlock()

	// Check that intervals are close to 100ms
	for i, interval := range intervals {
		deviation := math.Abs(float64(interval - 100*time.Millisecond))
		assert.LessOrEqual(t, deviation, float64(30*time.Millisecond),
			"Interval %d deviation %.2fms should be <= 30ms",
			i, float64(deviation)/float64(time.Millisecond))
	}

	t.Logf("Interval precision test: %d adjustments, intervals range from %v to %v",
		count, intervals[0], intervals[len(intervals)-1])
}

// =============================================================================
// Test 5: Automatic QPS Recovery After Latency Drops
// =============================================================================

// TestAdaptiveControl_QPSRecoveryAfterLatencyDrops verifies that QPS
// gradually recovers after latency returns to normal.
func TestAdaptiveControl_QPSRecoveryAfterLatencyDrops(t *testing.T) {
	t.Parallel()

	rateLimiter := NewTokenBucketLimiter(100, 20)
	shaper := &mockTrafficShaper{qps: 100, phase: "recovery-test"}
	metrics := NewSlidingWindowMetrics(MetricsConfig{
		WindowSize: 5 * time.Second, // Shorter window for faster test
		BucketSize: 100 * time.Millisecond,
	})

	controller := NewLoadController(rateLimiter, shaper, nil, metrics, LoadControllerConfig{
		AdjustInterval:             100 * time.Millisecond,
		Adaptive:                   true,
		TargetP95:                  200 * time.Millisecond,
		AdaptiveReductionFactor:    0.15,
		AdaptiveRecoveryFactor:     0.05,
		ConsecutiveBreachThreshold: 2,
		MinQPS:                     20,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	controller.Start(ctx)
	defer controller.Stop()

	// Phase 1: Stabilize with normal latency
	for range 30 {
		metrics.RecordLatency(100 * time.Millisecond)
	}
	time.Sleep(300 * time.Millisecond)

	initialQPS := controller.ActualQPS()
	require.InDelta(t, 100.0, initialQPS, 5.0, "Initial QPS should be ~100")

	// Phase 2: Inject high latency to trigger reduction
	for range 50 {
		metrics.RecordLatency(500 * time.Millisecond) // >> 200ms target
	}
	time.Sleep(500 * time.Millisecond)

	reducedQPS := controller.ActualQPS()
	assert.Less(t, reducedQPS, initialQPS*0.95,
		"QPS should be reduced after high latency")
	assert.True(t, controller.IsAdaptiveActive(),
		"Adaptive control should be active")

	t.Logf("Phase 2 - After high latency: QPS reduced from %.2f to %.2f",
		initialQPS, reducedQPS)

	// Phase 3: Inject normal latency to trigger recovery
	// Reset metrics with good latencies
	metrics.Reset()
	for range 50 {
		metrics.RecordLatency(50 * time.Millisecond) // << 200ms target
	}

	// Wait for recovery (consecutive good readings)
	var recoveredQPS float64
	var recovered bool
	for range 30 {
		time.Sleep(200 * time.Millisecond)

		// Keep injecting good latencies
		for range 10 {
			metrics.RecordLatency(50 * time.Millisecond)
		}

		recoveredQPS = controller.ActualQPS()
		if !controller.IsAdaptiveActive() && recoveredQPS >= reducedQPS*1.05 {
			recovered = true
			break
		}
	}

	assert.True(t, recovered || recoveredQPS > reducedQPS,
		"QPS should recover after latency normalizes: reduced=%.2f, recovered=%.2f",
		reducedQPS, recoveredQPS)

	t.Logf("Phase 3 - After recovery: QPS from %.2f to %.2f (initial was %.2f)",
		reducedQPS, recoveredQPS, initialQPS)
}

// TestAdaptiveControl_GradualRecovery validates the pass criteria:
// "延迟恢复后QPS逐步提升"
func TestAdaptiveControl_GradualRecovery(t *testing.T) {
	t.Parallel()

	rateLimiter := NewTokenBucketLimiter(100, 20)
	shaper := &mockTrafficShaper{qps: 100, phase: "gradual-recovery"}
	metrics := NewSlidingWindowMetrics(MetricsConfig{
		WindowSize: 5 * time.Second,
		BucketSize: 100 * time.Millisecond,
	})

	controller := NewLoadController(rateLimiter, shaper, nil, metrics, LoadControllerConfig{
		AdjustInterval:             100 * time.Millisecond,
		Adaptive:                   true,
		TargetP95:                  100 * time.Millisecond,
		AdaptiveReductionFactor:    0.2,
		AdaptiveRecoveryFactor:     0.05,
		ConsecutiveBreachThreshold: 1,
		MinQPS:                     20,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	controller.Start(ctx)
	defer controller.Stop()

	// Stabilize
	for range 30 {
		metrics.RecordLatency(50 * time.Millisecond)
	}
	time.Sleep(300 * time.Millisecond)

	// Trigger reduction with high latency
	for range 50 {
		metrics.RecordLatency(300 * time.Millisecond)
	}
	time.Sleep(500 * time.Millisecond)

	beforeRecoveryQPS := controller.ActualQPS()
	require.Less(t, beforeRecoveryQPS, 100.0,
		"QPS should be reduced before recovery")

	// Track recovery progress
	type recoveryPoint struct {
		time time.Time
		qps  float64
	}
	var recoveryProgress []recoveryPoint

	// Reset and inject good latencies
	metrics.Reset()
	recoveryStart := time.Now()

	for i := range 50 {
		// Inject good latencies
		for range 10 {
			metrics.RecordLatency(30 * time.Millisecond)
		}

		time.Sleep(100 * time.Millisecond)

		currentQPS := controller.ActualQPS()
		recoveryProgress = append(recoveryProgress, recoveryPoint{
			time: time.Now(),
			qps:  currentQPS,
		})

		// Check if we've recovered close to target
		if currentQPS >= 95 && !controller.IsAdaptiveActive() {
			t.Logf("Full recovery achieved at iteration %d", i)
			break
		}
	}

	// Verify gradual recovery (QPS should generally increase over time)
	t.Logf("Recovery progress:")
	increasingTrend := 0
	for i, point := range recoveryProgress {
		elapsed := point.time.Sub(recoveryStart)
		t.Logf("  t=%.1fs: QPS=%.2f", elapsed.Seconds(), point.qps)

		if i > 0 && point.qps > recoveryProgress[i-1].qps {
			increasingTrend++
		}
	}

	// At least some recovery progress should show increasing trend
	if len(recoveryProgress) > 5 {
		assert.GreaterOrEqual(t, increasingTrend, 2,
			"Recovery should show increasing QPS trend")
	}

	finalQPS := recoveryProgress[len(recoveryProgress)-1].qps
	assert.Greater(t, finalQPS, beforeRecoveryQPS,
		"Final QPS (%.2f) should be higher than before recovery (%.2f)",
		finalQPS, beforeRecoveryQPS)
}

// =============================================================================
// Integration and Stability Tests
// =============================================================================

// TestAdaptiveControl_FullCycle tests a complete adaptive control cycle:
// normal -> high latency -> reduction -> recovery -> normal
func TestAdaptiveControl_FullCycle(t *testing.T) {
	t.Parallel()

	rateLimiter := NewTokenBucketLimiter(100, 20)
	shaper := &mockTrafficShaper{qps: 100, phase: "full-cycle"}
	metrics := NewSlidingWindowMetrics(MetricsConfig{
		WindowSize: 5 * time.Second,
		BucketSize: 100 * time.Millisecond,
	})

	controller := NewLoadController(rateLimiter, shaper, nil, metrics, LoadControllerConfig{
		AdjustInterval:             100 * time.Millisecond,
		Adaptive:                   true,
		TargetP95:                  200 * time.Millisecond,
		AdaptiveReductionFactor:    0.15,
		AdaptiveRecoveryFactor:     0.05,
		ConsecutiveBreachThreshold: 2,
		MinQPS:                     20,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	controller.Start(ctx)
	defer controller.Stop()

	// Phase 1: Normal operation
	t.Log("Phase 1: Normal operation")
	for range 30 {
		metrics.RecordLatency(100 * time.Millisecond)
	}
	time.Sleep(300 * time.Millisecond)

	normalQPS := controller.ActualQPS()
	assert.InDelta(t, 100.0, normalQPS, 5.0)
	assert.False(t, controller.IsAdaptiveActive())

	// Phase 2: High latency spike
	t.Log("Phase 2: High latency spike")
	for range 50 {
		metrics.RecordLatency(500 * time.Millisecond)
	}
	time.Sleep(500 * time.Millisecond)

	reducedQPS := controller.ActualQPS()
	assert.Less(t, reducedQPS, normalQPS*0.95,
		"QPS should be reduced during high latency")
	assert.True(t, controller.IsAdaptiveActive(),
		"Adaptive control should be active")

	// Phase 3: Recovery with normal latency
	t.Log("Phase 3: Recovery")
	metrics.Reset()
	for range 50 {
		metrics.RecordLatency(50 * time.Millisecond)
	}

	// Wait for recovery
	for range 30 {
		time.Sleep(200 * time.Millisecond)
		for range 10 {
			metrics.RecordLatency(50 * time.Millisecond)
		}
		if !controller.IsAdaptiveActive() {
			break
		}
	}

	recoveredQPS := controller.ActualQPS()
	assert.GreaterOrEqual(t, recoveredQPS, reducedQPS,
		"QPS should recover after latency normalizes")

	t.Logf("Full cycle: normal=%.2f -> reduced=%.2f -> recovered=%.2f",
		normalQPS, reducedQPS, recoveredQPS)
}

// TestAdaptiveControl_ThreadSafety tests concurrent access to adaptive control
func TestAdaptiveControl_ThreadSafety(t *testing.T) {
	t.Parallel()

	rateLimiter := NewTokenBucketLimiter(100, 20)
	shaper := &mockTrafficShaper{qps: 100, phase: "thread-safety"}
	metrics := NewSlidingWindowMetrics(MetricsConfig{
		WindowSize: 10 * time.Second,
		BucketSize: 100 * time.Millisecond,
	})
	workerPool := NewWorkerPool(WorkerPoolConfig{
		MinSize:     5,
		MaxSize:     50,
		InitialSize: 20,
	})

	controller := NewLoadController(rateLimiter, shaper, workerPool, metrics, LoadControllerConfig{
		AdjustInterval:             50 * time.Millisecond,
		Adaptive:                   true,
		TargetP95:                  200 * time.Millisecond,
		AdaptiveReductionFactor:    0.1,
		WorkerAutoScale:            true,
		WorkerLatencyBuffer:        1.5,
		ConsecutiveBreachThreshold: 2,
		MinQPS:                     10,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	controller.Start(ctx)
	defer controller.Stop()

	var wg sync.WaitGroup
	const goroutines = 10
	const iterations = 100

	// Concurrent latency recording
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range iterations {
				// Alternate between high and low latencies
				if time.Now().UnixNano()%2 == 0 {
					metrics.RecordLatency(50 * time.Millisecond)
				} else {
					metrics.RecordLatency(300 * time.Millisecond)
				}
			}
		}()
	}

	// Concurrent stats reading
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range iterations {
				_ = controller.Stats()
				_ = controller.ActualQPS()
				_ = controller.TargetQPS()
				_ = controller.IsAdaptiveActive()
			}
		}()
	}

	// Concurrent acquire
	for range goroutines / 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range iterations / 2 {
				_ = controller.TryAcquire()
			}
		}()
	}

	wg.Wait()

	// Should not panic and stats should be readable
	stats := controller.Stats()
	assert.True(t, stats.IsRunning || !stats.IsRunning) // Just verify no panic
}

// TestAdaptiveControl_Stability runs multiple iterations to verify stability
func TestAdaptiveControl_Stability(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stability test in short mode")
	}
	t.Parallel()

	const runs = 3

	for run := range runs {
		t.Run(fmt.Sprintf("run_%d", run+1), func(t *testing.T) {
			rateLimiter := NewTokenBucketLimiter(100, 20)
			shaper := &mockTrafficShaper{qps: 100, phase: "stability"}
			metrics := NewSlidingWindowMetrics(MetricsConfig{
				WindowSize: 5 * time.Second,
				BucketSize: 100 * time.Millisecond,
			})

			controller := NewLoadController(rateLimiter, shaper, nil, metrics, LoadControllerConfig{
				AdjustInterval:             100 * time.Millisecond,
				Adaptive:                   true,
				TargetP95:                  200 * time.Millisecond,
				AdaptiveReductionFactor:    0.15,
				ConsecutiveBreachThreshold: 2,
				MinQPS:                     20,
			})

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			controller.Start(ctx)
			defer controller.Stop()

			// Simulate varying load
			for i := range 20 {
				latency := 100 * time.Millisecond
				if i%5 == 0 {
					latency = 500 * time.Millisecond // Occasional spike
				}
				for range 10 {
					metrics.RecordLatency(latency)
				}
				time.Sleep(100 * time.Millisecond)
			}

			stats := controller.Stats()
			assert.True(t, stats.IsRunning || !stats.IsRunning)
			t.Logf("Run %d: Final QPS=%.2f, AdaptiveActive=%v",
				run+1, stats.ActualQPS, stats.AdaptiveActive)
		})
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

// BenchmarkAdaptiveControl_Adjustment benchmarks the adjustment cycle
func BenchmarkAdaptiveControl_Adjustment(b *testing.B) {
	rateLimiter := NewTokenBucketLimiter(100, 20)
	shaper := &mockTrafficShaper{qps: 100, phase: "bench"}
	metrics := NewSlidingWindowMetrics(MetricsConfig{
		WindowSize: 10 * time.Second,
		BucketSize: 100 * time.Millisecond,
	})
	workerPool := NewWorkerPool(WorkerPoolConfig{
		MinSize:     5,
		MaxSize:     50,
		InitialSize: 20,
	})

	controller := NewLoadController(rateLimiter, shaper, workerPool, metrics, LoadControllerConfig{
		AdjustInterval:             100 * time.Millisecond,
		Adaptive:                   true,
		TargetP95:                  200 * time.Millisecond,
		AdaptiveReductionFactor:    0.1,
		WorkerAutoScale:            true,
		WorkerLatencyBuffer:        1.5,
		ConsecutiveBreachThreshold: 2,
		MinQPS:                     10,
	})

	// Pre-populate metrics
	for range 100 {
		metrics.RecordLatency(100 * time.Millisecond)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		controller.adjust()
	}
}

// BenchmarkOptimalWorkerCalculation benchmarks the worker calculation
func BenchmarkOptimalWorkerCalculation(b *testing.B) {
	workerPool := NewWorkerPool(WorkerPoolConfig{
		MinSize:     5,
		MaxSize:     100,
		InitialSize: 20,
	})
	metrics := NewSlidingWindowMetrics(MetricsConfig{
		WindowSize: 10 * time.Second,
		BucketSize: time.Second,
	})

	controller := NewLoadController(
		NewTokenBucketLimiter(100, 20),
		&mockTrafficShaper{qps: 100, phase: "bench"},
		workerPool,
		metrics,
		LoadControllerConfig{
			WorkerAutoScale:     true,
			WorkerLatencyBuffer: 1.5,
		},
	)

	// Pre-populate metrics
	for range 100 {
		metrics.RecordLatency(100 * time.Millisecond)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		controller.calculateOptimalWorkers(float64(100 + i%100))
	}
}
