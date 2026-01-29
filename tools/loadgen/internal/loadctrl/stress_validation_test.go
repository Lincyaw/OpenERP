package loadctrl

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// LOADGEN-VAL-009: Stress Testing and Extreme Scenario Validation Tests
// =============================================================================
//
// This file contains comprehensive validation tests for stress testing and
// extreme scenarios. These tests verify:
//
// 1. 测试逐步增压到系统极限 (Ramp-up to system limits)
// 2. 验证优雅降级行为 (Graceful degradation behavior)
// 3. 测试1小时持续负载 (Simulated 1-hour sustained load)
// 4. 监控内存泄漏 (Memory leak monitoring)
// 5. 测试快速停止和恢复 (Fast stop and recovery)
// 6. 验证数据一致性 (Data consistency under stress)
//
// Pass Criteria:
// - 系统在极限负载下不崩溃
// - 降级时保持核心功能
// - 长时间运行无内存泄漏
// - 停止后可快速恢复
// =============================================================================

// stressTestShaper is a test traffic shaper for stress tests
type stressTestShaper struct {
	qps   float64
	phase string
	mu    sync.RWMutex
}

func newStressTestShaper(qps float64) *stressTestShaper {
	return &stressTestShaper{qps: qps, phase: "stress"}
}

func (s *stressTestShaper) GetTargetQPS(_ time.Duration) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.qps
}

func (s *stressTestShaper) SetQPS(qps float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.qps = qps
}

func (s *stressTestShaper) GetPhase(_ time.Duration) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.phase
}

func (s *stressTestShaper) Name() string {
	return "stress"
}

func (s *stressTestShaper) Config() ShaperConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return ShaperConfig{Type: "constant", BaseQPS: s.qps}
}

// stressMetricsCollector is a thread-safe metrics collector for stress tests
type stressMetricsCollector struct {
	mu         sync.RWMutex
	errorRate  float64
	p99Latency time.Duration
	p95Latency time.Duration
	avgLatency time.Duration
}

func newStressMetrics() *stressMetricsCollector {
	return &stressMetricsCollector{}
}

func (m *stressMetricsCollector) setErrorRate(rate float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorRate = rate
}

func (m *stressMetricsCollector) setP99Latency(latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.p99Latency = latency
}

func (m *stressMetricsCollector) RecordLatency(_ time.Duration) {}
func (m *stressMetricsCollector) RecordError()                  {}
func (m *stressMetricsCollector) RecordSuccess()                {}

func (m *stressMetricsCollector) GetP95Latency() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.p95Latency
}

func (m *stressMetricsCollector) GetP99Latency() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.p99Latency
}

func (m *stressMetricsCollector) GetAvgLatency() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.avgLatency
}

func (m *stressMetricsCollector) GetErrorRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.errorRate
}

func (m *stressMetricsCollector) GetStats() MetricsStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return MetricsStats{
		ErrorRate:  m.errorRate,
		P99Latency: m.p99Latency,
		P95Latency: m.p95Latency,
		AvgLatency: m.avgLatency,
	}
}

func (m *stressMetricsCollector) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorRate = 0
	m.p99Latency = 0
	m.p95Latency = 0
	m.avgLatency = 0
}

// =============================================================================
// Test 1: Ramp-up to System Limits (逐步增压到系统极限)
// =============================================================================

func TestStressValidation_LOADGEN_VAL_009(t *testing.T) {
	t.Parallel()

	t.Run("Ramp_up_to_high_QPS_without_crash", func(t *testing.T) {
		t.Parallel()

		// Create rate limiter that can handle high QPS
		rateLimiter := NewTokenBucketLimiter(5000, 1000)

		// Create shaper that progressively increases QPS
		shaper := newStressTestShaper(100)

		// Create worker pool
		workerPool := NewWorkerPool(WorkerPoolConfig{
			MinSize:     10,
			MaxSize:     500,
			InitialSize: 50,
		})

		// Create metrics collector
		metrics := newStressMetrics()

		// Create controller with adaptive control
		config := LoadControllerConfig{
			AdjustInterval:  10 * time.Millisecond,
			WorkerAutoScale: true,
		}
		controller := NewLoadController(rateLimiter, shaper, workerPool, metrics, config)

		// Start the controller
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		controller.Start(ctx)
		defer controller.Stop()

		// Track QPS levels reached
		var maxQPSReached atomic.Int64
		var totalAcquired atomic.Int64
		var panicOccurred atomic.Bool

		// Monitor for 1 second
		var wg sync.WaitGroup
		monitorCtx, monitorCancel := context.WithTimeout(ctx, time.Second)
		defer monitorCancel()

		// Progressively increase QPS
		go func() {
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			qps := float64(100)
			for {
				select {
				case <-monitorCtx.Done():
					return
				case <-ticker.C:
					qps = min(qps*1.5, 5000)
					shaper.SetQPS(qps)
					rateLimiter.SetRate(qps)
				}
			}
		}()

		// Multiple goroutines trying to acquire at high rate
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						panicOccurred.Store(true)
					}
					wg.Done()
				}()
				for {
					select {
					case <-monitorCtx.Done():
						return
					default:
						if controller.TryAcquire() {
							totalAcquired.Add(1)
						}
						time.Sleep(100 * time.Microsecond) // ~10000 attempts/sec per goroutine
					}
				}
			}()
		}

		// Monitor QPS progression
		go func() {
			ticker := time.NewTicker(50 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-monitorCtx.Done():
					return
				case <-ticker.C:
					stats := controller.Stats()
					current := int64(stats.TargetQPS)
					for {
						old := maxQPSReached.Load()
						if current <= old || maxQPSReached.CompareAndSwap(old, current) {
							break
						}
					}
				}
			}
		}()

		wg.Wait()

		assert.False(t, panicOccurred.Load(), "System should not panic under high load ramp-up")
		assert.Greater(t, maxQPSReached.Load(), int64(1000), "Should reach at least 1000 QPS during ramp-up")
		assert.Greater(t, totalAcquired.Load(), int64(100), "Should successfully acquire many requests")

		t.Logf("Ramp-up validation: Max QPS reached = %d, Total acquired = %d",
			maxQPSReached.Load(), totalAcquired.Load())
	})

	t.Run("System_handles_peak_QPS_of_5000", func(t *testing.T) {
		t.Parallel()

		// Create rate limiter for 5000 QPS
		rateLimiter := NewTokenBucketLimiter(5000, 500)

		// Constant shaper at peak QPS
		shaper := newStressTestShaper(5000)

		config := LoadControllerConfig{
			AdjustInterval: 10 * time.Millisecond,
		}
		controller := NewLoadController(rateLimiter, shaper, nil, nil, config)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		controller.Start(ctx)
		defer controller.Stop()

		// Wait for controller to stabilize
		time.Sleep(50 * time.Millisecond)

		stats := controller.Stats()
		assert.InDelta(t, 5000, stats.TargetQPS, 100, "Should reach target QPS of 5000")
		assert.InDelta(t, 5000, stats.ActualQPS, 100, "Rate limiter should be set to 5000 QPS")

		t.Logf("Peak QPS test: Target=%v, Actual=%v", stats.TargetQPS, stats.ActualQPS)
	})

	t.Run("Gradual_ramp_up_300s_simulation", func(t *testing.T) {
		t.Parallel()

		// Create step shaper that simulates 300s ramp-up compressed to 300ms
		stepConfig := ShaperConfig{
			Type:    "step",
			BaseQPS: 100,
			Step: &StepConfig{
				Steps: []StepLevel{
					{QPS: 100, Duration: 60 * time.Millisecond, RampDuration: 30 * time.Millisecond},
					{QPS: 500, Duration: 60 * time.Millisecond, RampDuration: 30 * time.Millisecond},
					{QPS: 1000, Duration: 60 * time.Millisecond, RampDuration: 30 * time.Millisecond},
					{QPS: 1000, Duration: 30 * time.Millisecond, RampDuration: 0},
				},
			},
		}
		trafficShaper, err := NewStepShaper(stepConfig)
		require.NoError(t, err)

		rateLimiter := NewTokenBucketLimiter(2000, 200)

		config := LoadControllerConfig{
			AdjustInterval: 5 * time.Millisecond,
		}
		controller := NewLoadController(rateLimiter, trafficShaper, nil, nil, config)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		controller.Start(ctx)
		defer controller.Stop()

		// Track QPS progression
		var qpsHistory []float64
		ticker := time.NewTicker(30 * time.Millisecond)
		defer ticker.Stop()

		done := time.After(320 * time.Millisecond)
		for {
			select {
			case <-done:
				goto verify
			case <-ticker.C:
				stats := controller.Stats()
				qpsHistory = append(qpsHistory, stats.TargetQPS)
			}
		}

	verify:
		require.GreaterOrEqual(t, len(qpsHistory), 3, "Should have multiple QPS samples")

		// Verify progression (should generally increase)
		hasIncrease := false
		for i := 1; i < len(qpsHistory); i++ {
			if qpsHistory[i] > qpsHistory[0]+50 {
				hasIncrease = true
				break
			}
		}
		assert.True(t, hasIncrease, "QPS should increase during ramp-up")

		t.Logf("300s ramp-up simulation (scaled): QPS history = %v", qpsHistory)
	})
}

// =============================================================================
// Test 2: Graceful Degradation Behavior (优雅降级行为)
// =============================================================================

func TestStress_GracefulDegradation(t *testing.T) {
	t.Parallel()

	t.Run("Maintains_core_functionality_during_degradation", func(t *testing.T) {
		t.Parallel()

		metrics := newStressMetrics()
		config := BackpressureConfig{
			Strategy:                   BackpressureStrategyReduce,
			ErrorRateThreshold:         0.1,
			ConsecutiveBreachThreshold: 1,
			ReductionFactor:            0.5,
		}
		handler := NewBackpressureHandler(metrics, config)

		// Simulate high error rate
		metrics.setErrorRate(0.2)
		for i := 0; i < 3; i++ {
			handler.Check()
		}

		// Verify degradation is active
		require.Equal(t, BackpressureStateCritical, handler.State())

		// Verify core functionality still works
		action := handler.Check()
		assert.NotNil(t, action, "Check should still return action")
		assert.Greater(t, action.QPSMultiplier, 0.0, "QPS multiplier should be > 0")
		assert.Less(t, action.QPSMultiplier, 1.0, "QPS multiplier should be < 1.0 (degraded)")

		// Verify ShouldAllow still responds
		result := handler.ShouldAllow()
		assert.True(t, result, "Reduce strategy should allow requests")

		// Verify Stats still accessible
		stats := handler.Stats()
		assert.Equal(t, BackpressureStateCritical, stats.CurrentState)

		t.Logf("Degradation: QPSMultiplier=%v, ShouldAllow=%v, State=%v",
			action.QPSMultiplier, result, stats.CurrentState)
	})

	t.Run("Degradation_reduces_load_proportionally", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			reductionFactor float64
			expectedRange   [2]float64
		}{
			{0.25, [2]float64{0.20, 0.30}},
			{0.50, [2]float64{0.45, 0.55}},
			{0.75, [2]float64{0.70, 0.80}},
		}

		for _, tc := range testCases {
			metrics := newStressMetrics()
			config := BackpressureConfig{
				Strategy:                   BackpressureStrategyReduce,
				ErrorRateThreshold:         0.1,
				ConsecutiveBreachThreshold: 1,
				ReductionFactor:            tc.reductionFactor,
			}
			handler := NewBackpressureHandler(metrics, config)

			// Trigger degradation
			metrics.setErrorRate(0.2)
			for i := 0; i < 3; i++ {
				handler.Check()
			}

			action := handler.Check()
			assert.GreaterOrEqual(t, action.QPSMultiplier, tc.expectedRange[0],
				"QPSMultiplier should be >= %v for reduction factor %v", tc.expectedRange[0], tc.reductionFactor)
			assert.LessOrEqual(t, action.QPSMultiplier, tc.expectedRange[1],
				"QPSMultiplier should be <= %v for reduction factor %v", tc.expectedRange[1], tc.reductionFactor)
		}
	})

	t.Run("Recovery_from_degradation_is_gradual", func(t *testing.T) {
		t.Parallel()

		metrics := newStressMetrics()
		config := BackpressureConfig{
			Strategy:                   BackpressureStrategyReduce,
			ErrorRateThreshold:         0.1,
			WarningErrorThreshold:      0.05,
			ConsecutiveBreachThreshold: 1,
			RecoveryPeriod:             100 * time.Millisecond,
		}
		handler := NewBackpressureHandler(metrics, config)

		// Trigger and then recover
		metrics.setErrorRate(0.2)
		for i := 0; i < 3; i++ {
			handler.Check()
		}
		require.Equal(t, BackpressureStateCritical, handler.State())

		// Start recovery
		metrics.setErrorRate(0.01)
		handler.Check()
		require.Equal(t, BackpressureStateRecovery, handler.State())

		// Track QPSMultiplier over recovery period
		var multipliers []float64
		start := time.Now()
		for time.Since(start) < 120*time.Millisecond {
			action := handler.Check()
			multipliers = append(multipliers, action.QPSMultiplier)
			time.Sleep(10 * time.Millisecond)
		}

		require.GreaterOrEqual(t, len(multipliers), 5, "Should have multiple samples")

		// Verify gradual increase
		for i := 1; i < len(multipliers); i++ {
			assert.GreaterOrEqual(t, multipliers[i], multipliers[i-1]-0.1,
				"QPSMultiplier should not decrease during recovery (allowing small variance)")
		}

		// First should be lower, last should be higher
		assert.Less(t, multipliers[0], multipliers[len(multipliers)-1],
			"Final multiplier should be higher than initial")

		t.Logf("Recovery multipliers: %v", multipliers)
	})

	t.Run("System_does_not_crash_under_extreme_degradation", func(t *testing.T) {
		t.Parallel()

		metrics := newStressMetrics()
		config := BackpressureConfig{
			Strategy:                   BackpressureStrategyCircuit,
			ErrorRateThreshold:         0.1,
			ConsecutiveBreachThreshold: 1,
			CircuitOpenDuration:        50 * time.Millisecond,
		}
		handler := NewBackpressureHandler(metrics, config)

		var panicOccurred atomic.Bool

		// Extreme scenario: rapid oscillation between states
		done := make(chan struct{})
		go func() {
			defer func() {
				if r := recover(); r != nil {
					panicOccurred.Store(true)
				}
				close(done)
			}()

			for i := 0; i < 1000; i++ {
				if i%2 == 0 {
					metrics.setErrorRate(0.3) // Extreme error
				} else {
					metrics.setErrorRate(0.01) // Recovery
				}
				handler.Check()
				handler.ShouldAllow()
				handler.Stats()
			}
		}()

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Test timed out")
		}

		assert.False(t, panicOccurred.Load(), "Should not panic under extreme state oscillation")
	})
}

// =============================================================================
// Test 3: Sustained Load Simulation (1小时持续负载模拟)
// =============================================================================

func TestStress_SustainedLoad(t *testing.T) {
	t.Parallel()

	t.Run("Simulated_1_hour_sustained_load", func(t *testing.T) {
		t.Parallel()

		// Simulate 1 hour compressed to 1 second
		// Scale: 1 hour = 1 second (3600:1 compression)
		const (
			compressionRatio = 3600
		)

		rateLimiter := NewTokenBucketLimiter(100, 50)
		shaper := newStressTestShaper(100)
		workerPool := NewWorkerPool(WorkerPoolConfig{
			MinSize:     5,
			MaxSize:     50,
			InitialSize: 10,
		})

		config := LoadControllerConfig{
			AdjustInterval:  10 * time.Millisecond,
			WorkerAutoScale: true,
		}
		controller := NewLoadController(rateLimiter, shaper, workerPool, nil, config)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		controller.Start(ctx)
		defer controller.Stop()

		// Track metrics over simulated hour
		type sample struct {
			simulatedTime time.Duration
			isRunning     bool
			targetQPS     float64
			workerSize    int
		}
		var samples []sample

		startTime := time.Now()
		ticker := time.NewTicker(100 * time.Millisecond) // Sample every ~6 simulated minutes
		defer ticker.Stop()

		testEnd := time.After(time.Second)
		for {
			select {
			case <-testEnd:
				goto analyze
			case <-ticker.C:
				elapsed := time.Since(startTime)
				stats := controller.Stats()

				var workerSize int
				if stats.WorkerPoolStats != nil {
					workerSize = stats.WorkerPoolStats.CurrentSize
				}

				samples = append(samples, sample{
					simulatedTime: time.Duration(float64(elapsed) * compressionRatio),
					isRunning:     stats.IsRunning,
					targetQPS:     stats.TargetQPS,
					workerSize:    workerSize,
				})
			}
		}

	analyze:
		require.GreaterOrEqual(t, len(samples), 5, "Should have multiple samples over simulated hour")

		// Verify system remained running throughout
		allRunning := true
		for _, s := range samples {
			if !s.isRunning {
				allRunning = false
				break
			}
		}
		assert.True(t, allRunning, "System should remain running throughout simulated hour")

		// Verify QPS remained stable
		var minQPS, maxQPS float64 = 10000, 0
		for _, s := range samples {
			if s.targetQPS < minQPS {
				minQPS = s.targetQPS
			}
			if s.targetQPS > maxQPS {
				maxQPS = s.targetQPS
			}
		}
		assert.InDelta(t, 100, (minQPS+maxQPS)/2, 20, "Average QPS should be stable around 100")

		t.Logf("Sustained load (1 hour simulated): %d samples, QPS range: %.0f-%.0f",
			len(samples), minQPS, maxQPS)
	})

	t.Run("Continuous_operation_without_resource_exhaustion", func(t *testing.T) {
		t.Parallel()

		rateLimiter := NewTokenBucketLimiter(500, 100)
		shaper := newStressTestShaper(500)

		config := LoadControllerConfig{
			AdjustInterval: 10 * time.Millisecond,
		}
		controller := NewLoadController(rateLimiter, shaper, nil, nil, config)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		controller.Start(ctx)
		defer controller.Stop()

		// High-frequency operations
		var opsCount atomic.Int64
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					default:
						controller.TryAcquire()
						controller.Stats()
						opsCount.Add(2)
					}
				}
			}()
		}

		wg.Wait()

		assert.Greater(t, opsCount.Load(), int64(1000), "Should complete many operations without exhaustion")
		t.Logf("Completed %d operations in 1 second", opsCount.Load())
	})
}

// =============================================================================
// Test 4: Memory Leak Monitoring (监控内存泄漏)
// =============================================================================

func TestStress_MemoryLeakMonitoring(t *testing.T) {
	t.Parallel()

	t.Run("No_memory_leak_during_sustained_operations", func(t *testing.T) {
		t.Parallel()

		// Force GC and get baseline
		runtime.GC()
		var baselineStats runtime.MemStats
		runtime.ReadMemStats(&baselineStats)
		baselineHeap := baselineStats.HeapAlloc

		rateLimiter := NewTokenBucketLimiter(1000, 200)
		shaper := newStressTestShaper(1000)
		workerPool := NewWorkerPool(WorkerPoolConfig{
			MinSize:     5,
			MaxSize:     100,
			InitialSize: 20,
		})

		config := LoadControllerConfig{
			AdjustInterval:  10 * time.Millisecond,
			WorkerAutoScale: true,
		}
		controller := NewLoadController(rateLimiter, shaper, workerPool, nil, config)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		controller.Start(ctx)
		defer controller.Stop()

		// Perform many operations
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					default:
						controller.TryAcquire()
						controller.Stats()
					}
				}
			}()
		}

		wg.Wait()
		controller.Stop()

		// Force GC and measure final
		runtime.GC()
		time.Sleep(10 * time.Millisecond) // Allow GC to complete
		runtime.GC()

		var finalStats runtime.MemStats
		runtime.ReadMemStats(&finalStats)
		finalHeap := finalStats.HeapAlloc

		// Check for significant memory growth (allow some variance)
		// A leak would show continuous growth; normal usage might fluctuate
		heapGrowth := int64(finalHeap) - int64(baselineHeap)
		heapGrowthMB := float64(heapGrowth) / (1024 * 1024)

		// Allow up to 10MB growth for normal operation overhead
		assert.Less(t, heapGrowthMB, 10.0,
			"Heap should not grow significantly (grew %.2f MB)", heapGrowthMB)

		t.Logf("Memory: Baseline=%d KB, Final=%d KB, Growth=%.2f MB",
			baselineHeap/1024, finalHeap/1024, heapGrowthMB)
	})

	t.Run("Backpressure_handler_no_memory_leak", func(t *testing.T) {
		t.Parallel()

		runtime.GC()
		var baselineStats runtime.MemStats
		runtime.ReadMemStats(&baselineStats)
		baselineHeap := baselineStats.HeapAlloc

		metrics := newStressMetrics()
		config := BackpressureConfig{
			Strategy:                   BackpressureStrategyReduce,
			ErrorRateThreshold:         0.1,
			ConsecutiveBreachThreshold: 1,
			CheckInterval:              time.Millisecond,
		}
		handler := NewBackpressureHandler(metrics, config)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		handler.Start(ctx)
		defer handler.Stop()

		// Perform many state transitions
		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 10000; j++ {
					select {
					case <-ctx.Done():
						return
					default:
						if j%2 == 0 {
							metrics.setErrorRate(0.2)
						} else {
							metrics.setErrorRate(0.01)
						}
						handler.Check()
						handler.Stats()
					}
				}
			}()
		}

		wg.Wait()
		handler.Stop()

		runtime.GC()
		time.Sleep(10 * time.Millisecond)
		runtime.GC()

		var finalStats runtime.MemStats
		runtime.ReadMemStats(&finalStats)
		finalHeap := finalStats.HeapAlloc

		heapGrowth := int64(finalHeap) - int64(baselineHeap)
		heapGrowthMB := float64(heapGrowth) / (1024 * 1024)

		assert.Less(t, heapGrowthMB, 5.0,
			"Backpressure handler should not leak memory (grew %.2f MB)", heapGrowthMB)

		t.Logf("Backpressure memory: Baseline=%d KB, Final=%d KB, Growth=%.2f MB",
			baselineHeap/1024, finalHeap/1024, heapGrowthMB)
	})

	t.Run("Worker_pool_no_goroutine_leak", func(t *testing.T) {
		// NOTE: This test cannot be run in parallel with other tests that create goroutines
		// because other parallel tests affect the goroutine count. Run sequentially.

		// Wait for other goroutines to stabilize first
		time.Sleep(100 * time.Millisecond)
		runtime.GC()
		runtime.Gosched()

		initialGoroutines := runtime.NumGoroutine()

		// Create and destroy worker pools multiple times
		for i := 0; i < 5; i++ {
			pool := NewWorkerPool(WorkerPoolConfig{
				MinSize:     5,
				MaxSize:     50,
				InitialSize: 20,
			})

			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			pool.Start(ctx)

			// Submit some tasks
			for j := 0; j < 50; j++ {
				pool.Submit(func(ctx context.Context) error {
					time.Sleep(time.Microsecond)
					return nil
				})
			}

			pool.Stop()
			cancel()

			// Give goroutines time to clean up
			time.Sleep(20 * time.Millisecond)
		}

		// Force scheduler to run multiple times
		for i := 0; i < 3; i++ {
			runtime.Gosched()
			time.Sleep(50 * time.Millisecond)
		}

		finalGoroutines := runtime.NumGoroutine()

		// Allow larger variance due to test parallelism and other system goroutines
		goroutineGrowth := finalGoroutines - initialGoroutines

		// NOTE: In parallel test environment, goroutine count can vary significantly
		// We check that the growth is within reasonable bounds (not hundreds)
		// A true leak would show linear growth with iterations
		assert.Less(t, goroutineGrowth, 100,
			"Should not leak many goroutines (grew by %d)", goroutineGrowth)

		t.Logf("Goroutines: Initial=%d, Final=%d, Growth=%d",
			initialGoroutines, finalGoroutines, goroutineGrowth)
	})
}

// =============================================================================
// Test 5: Fast Stop and Recovery (快速停止和恢复)
// =============================================================================

func TestStress_FastStopAndRecovery(t *testing.T) {
	t.Parallel()

	t.Run("Controller_stops_quickly", func(t *testing.T) {
		t.Parallel()

		rateLimiter := NewTokenBucketLimiter(1000, 200)
		shaper := newStressTestShaper(1000)
		workerPool := NewWorkerPool(WorkerPoolConfig{
			MinSize:     10,
			MaxSize:     100,
			InitialSize: 50,
		})

		config := LoadControllerConfig{
			AdjustInterval: 10 * time.Millisecond,
		}
		controller := NewLoadController(rateLimiter, shaper, workerPool, nil, config)

		ctx := context.Background()
		controller.Start(ctx)

		// Let it run briefly
		time.Sleep(100 * time.Millisecond)

		// Measure stop time
		startStop := time.Now()
		controller.Stop()
		stopDuration := time.Since(startStop)

		assert.Less(t, stopDuration, 500*time.Millisecond,
			"Controller should stop within 500ms (took %v)", stopDuration)

		// Verify stopped state
		stats := controller.Stats()
		assert.False(t, stats.IsRunning, "Controller should not be running after Stop()")

		t.Logf("Stop duration: %v", stopDuration)
	})

	t.Run("Controller_restarts_after_stop", func(t *testing.T) {
		t.Parallel()

		rateLimiter := NewTokenBucketLimiter(500, 100)
		shaper := newStressTestShaper(500)
		workerPool := NewWorkerPool(WorkerPoolConfig{
			MinSize:     5,
			MaxSize:     50,
			InitialSize: 10,
		})

		config := LoadControllerConfig{
			AdjustInterval: 10 * time.Millisecond,
		}
		controller := NewLoadController(rateLimiter, shaper, workerPool, nil, config)

		ctx := context.Background()

		// Start-stop-start cycle
		controller.Start(ctx)
		time.Sleep(50 * time.Millisecond)
		assert.True(t, controller.Stats().IsRunning, "Should be running after first Start")

		controller.Stop()
		assert.False(t, controller.Stats().IsRunning, "Should be stopped after Stop")

		// Restart
		controller.Start(ctx)
		time.Sleep(50 * time.Millisecond)
		assert.True(t, controller.Stats().IsRunning, "Should be running after restart")

		// Verify functionality restored
		acquired := controller.TryAcquire()
		stats := controller.Stats()

		assert.True(t, acquired || stats.RateLimiterStats.TotalAcquired > 0,
			"Should be able to acquire after restart")
		assert.Greater(t, stats.TargetQPS, float64(0), "Target QPS should be set after restart")

		controller.Stop()

		t.Log("Restart cycle completed successfully")
	})

	t.Run("Multiple_stop_start_cycles_stable", func(t *testing.T) {
		t.Parallel()

		rateLimiter := NewTokenBucketLimiter(100, 50)
		shaper := newStressTestShaper(100)
		workerPool := NewWorkerPool(WorkerPoolConfig{
			MinSize:     2,
			MaxSize:     20,
			InitialSize: 5,
		})

		config := LoadControllerConfig{
			AdjustInterval: 10 * time.Millisecond,
		}
		controller := NewLoadController(rateLimiter, shaper, workerPool, nil, config)

		ctx := context.Background()

		var panicOccurred atomic.Bool
		cycles := 10

		func() {
			defer func() {
				if r := recover(); r != nil {
					panicOccurred.Store(true)
				}
			}()

			for i := 0; i < cycles; i++ {
				controller.Start(ctx)
				time.Sleep(20 * time.Millisecond)

				// Perform some operations while running
				for j := 0; j < 10; j++ {
					controller.TryAcquire()
					controller.Stats()
				}

				controller.Stop()
				time.Sleep(10 * time.Millisecond)
			}
		}()

		assert.False(t, panicOccurred.Load(),
			"Multiple stop/start cycles should not cause panic")

		t.Logf("Completed %d stop/start cycles without issues", cycles)
	})

	t.Run("Backpressure_handler_fast_stop_recovery", func(t *testing.T) {
		t.Parallel()

		metrics := newStressMetrics()
		config := BackpressureConfig{
			Strategy:                   BackpressureStrategyReduce,
			ErrorRateThreshold:         0.1,
			ConsecutiveBreachThreshold: 1,
			CheckInterval:              5 * time.Millisecond,
		}
		handler := NewBackpressureHandler(metrics, config)

		ctx := context.Background()

		// Start-stop-start multiple times
		for i := 0; i < 5; i++ {
			handler.Start(ctx)
			time.Sleep(20 * time.Millisecond)

			// Trigger some state changes
			metrics.setErrorRate(0.2)
			handler.Check()

			startStop := time.Now()
			handler.Stop()
			stopDuration := time.Since(startStop)

			assert.Less(t, stopDuration, 100*time.Millisecond,
				"Backpressure handler should stop quickly (cycle %d took %v)", i, stopDuration)

			// Verify can restart
			handler.Reset()
			handler.Start(ctx)
			assert.Equal(t, BackpressureStateNormal, handler.State(),
				"Should reset to normal state")
			handler.Stop()
		}

		t.Log("Backpressure handler stop/recovery cycles completed")
	})
}

// =============================================================================
// Test 6: Data Consistency Under Stress (验证数据一致性)
// =============================================================================

func TestStress_DataConsistency(t *testing.T) {
	t.Parallel()

	t.Run("Rate_limiter_counters_consistent", func(t *testing.T) {
		t.Parallel()

		rateLimiter := NewTokenBucketLimiter(1000, 100)

		var acquiredCount atomic.Int64
		var rejectedCount atomic.Int64

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		var wg sync.WaitGroup
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					default:
						if rateLimiter.TryAcquire() {
							acquiredCount.Add(1)
						} else {
							rejectedCount.Add(1)
						}
					}
				}
			}()
		}

		wg.Wait()

		stats := rateLimiter.Stats()
		expectedTotal := acquiredCount.Load() + rejectedCount.Load()

		// Verify internal counter consistency
		assert.Greater(t, acquiredCount.Load(), int64(0), "Should have acquired some tokens")
		assert.Greater(t, stats.TotalAcquired, int64(0), "Stats should show tokens acquired")

		// Stats.TotalAcquired tracks successful acquires
		assert.InDelta(t, acquiredCount.Load(), stats.TotalAcquired, float64(acquiredCount.Load())*0.1,
			"Acquired count should roughly match stats")

		t.Logf("Consistency: Acquired=%d, Rejected=%d, Total=%d, StatsTotalAcquired=%d",
			acquiredCount.Load(), rejectedCount.Load(), expectedTotal, stats.TotalAcquired)
	})

	t.Run("Worker_pool_task_counts_consistent", func(t *testing.T) {
		t.Parallel()

		pool := NewWorkerPool(WorkerPoolConfig{
			MinSize:     5,
			MaxSize:     50,
			InitialSize: 20,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		pool.Start(ctx)
		defer pool.Stop()

		var submittedCount atomic.Int64
		var executedCount atomic.Int64
		var failedCount atomic.Int64

		// Submit tasks
		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					select {
					case <-ctx.Done():
						return
					default:
						submitted := pool.Submit(func(ctx context.Context) error {
							executedCount.Add(1)
							return nil
						})
						if submitted {
							submittedCount.Add(1)
						}
					}
				}
			}()
		}

		wg.Wait()
		time.Sleep(100 * time.Millisecond) // Allow tasks to complete

		stats := pool.Stats()
		expectedTotal := stats.TotalExecuted + stats.TotalFailed

		// Verify consistency
		assert.GreaterOrEqual(t, submittedCount.Load(), expectedTotal,
			"Submitted should be >= executed + failed")
		assert.Equal(t, stats.TotalFailed, failedCount.Load(),
			"Failed count should match")

		// executedCount tracked by tasks might slightly differ from stats due to timing
		assert.InDelta(t, stats.TotalExecuted, executedCount.Load(), float64(stats.TotalExecuted)*0.2,
			"Executed counts should be roughly consistent")

		t.Logf("Worker pool: Submitted=%d, Executed=%d, Failed=%d, StatsExecuted=%d, StatsFailed=%d",
			submittedCount.Load(), executedCount.Load(), failedCount.Load(),
			stats.TotalExecuted, stats.TotalFailed)
	})

	t.Run("Backpressure_stats_consistent", func(t *testing.T) {
		t.Parallel()

		metrics := newStressMetrics()
		config := BackpressureConfig{
			Strategy:                   BackpressureStrategyDrop,
			ErrorRateThreshold:         0.1,
			ConsecutiveBreachThreshold: 1,
			DropPercentage:             0.5,
		}
		handler := NewBackpressureHandler(metrics, config)

		// Drive to critical
		metrics.setErrorRate(0.2)
		for i := 0; i < 3; i++ {
			handler.Check()
		}
		require.Equal(t, BackpressureStateCritical, handler.State())

		var droppedCount atomic.Int64
		var allowedCount atomic.Int64

		// Perform many checks
		for i := 0; i < 1000; i++ {
			action := handler.Check()
			if action.ShouldDrop {
				droppedCount.Add(1)
			} else {
				allowedCount.Add(1)
			}
		}

		stats := handler.Stats()

		// Verify stats track drops
		assert.GreaterOrEqual(t, stats.TotalDropped, droppedCount.Load()-100,
			"Stats TotalDropped should be close to actual drops")

		// Verify drop rate is approximately correct
		actualDropRate := float64(droppedCount.Load()) / 1000.0
		assert.InDelta(t, 0.5, actualDropRate, 0.15,
			"Drop rate should be approximately 50%%")

		t.Logf("Drop consistency: Dropped=%d, Allowed=%d, Rate=%.2f%%, StatsDropped=%d",
			droppedCount.Load(), allowedCount.Load(), actualDropRate*100, stats.TotalDropped)
	})

	t.Run("State_transitions_counted_correctly", func(t *testing.T) {
		t.Parallel()

		metrics := newStressMetrics()
		config := BackpressureConfig{
			ErrorRateThreshold:         0.1,
			WarningErrorThreshold:      0.05,
			ConsecutiveBreachThreshold: 1,
			RecoveryPeriod:             20 * time.Millisecond,
		}
		handler := NewBackpressureHandler(metrics, config)

		// Count expected transitions manually
		var expectedTransitions int64

		// normal -> warning
		metrics.setErrorRate(0.06)
		handler.Check()
		if handler.State() == BackpressureStateWarning {
			expectedTransitions++
		}

		// warning -> critical
		metrics.setErrorRate(0.15)
		handler.Check()
		handler.Check()
		if handler.State() == BackpressureStateCritical {
			expectedTransitions++
		}

		// critical -> recovery
		metrics.setErrorRate(0.01)
		handler.Check()
		if handler.State() == BackpressureStateRecovery {
			expectedTransitions++
		}

		// recovery -> normal (after period)
		time.Sleep(30 * time.Millisecond)
		handler.Check()
		if handler.State() == BackpressureStateNormal {
			expectedTransitions++
		}

		stats := handler.Stats()

		assert.GreaterOrEqual(t, stats.TotalStateTransitions, expectedTransitions,
			"Stats should track at least %d transitions", expectedTransitions)

		t.Logf("State transitions: Expected>=%d, Actual=%d",
			expectedTransitions, stats.TotalStateTransitions)
	})
}

// =============================================================================
// Test 7: Combined Stress Scenarios
// =============================================================================

func TestStress_CombinedScenarios(t *testing.T) {
	t.Parallel()

	t.Run("Full_system_stress_test", func(t *testing.T) {
		t.Parallel()

		// Create full load control stack
		rateLimiter := NewTokenBucketLimiter(500, 100)
		shaper := newStressTestShaper(500)

		workerPool := NewWorkerPool(WorkerPoolConfig{
			MinSize:     5,
			MaxSize:     100,
			InitialSize: 20,
		})

		metrics := newStressMetrics()

		config := LoadControllerConfig{
			AdjustInterval:  10 * time.Millisecond,
			WorkerAutoScale: true,
		}
		controller := NewLoadController(rateLimiter, shaper, workerPool, metrics, config)

		// Backpressure handler
		bpConfig := BackpressureConfig{
			Strategy:                   BackpressureStrategyReduce,
			ErrorRateThreshold:         0.1,
			ConsecutiveBreachThreshold: 2,
			ReductionFactor:            0.5,
			CheckInterval:              10 * time.Millisecond,
		}
		backpressure := NewBackpressureHandler(metrics, bpConfig)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		controller.Start(ctx)
		backpressure.Start(ctx)
		defer func() {
			controller.Stop()
			backpressure.Stop()
		}()

		// Run concurrent operations
		var wg sync.WaitGroup
		var totalOps atomic.Int64
		var panicOccurred atomic.Bool

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						panicOccurred.Store(true)
					}
					wg.Done()
				}()
				for {
					select {
					case <-ctx.Done():
						return
					default:
						// Simulate varying load conditions
						if totalOps.Load()%100 < 10 {
							metrics.setErrorRate(0.15) // 10% of time: high errors
						} else {
							metrics.setErrorRate(0.02) // 90% of time: normal
						}

						controller.TryAcquire()
						controller.Stats()
						backpressure.Check()
						backpressure.ShouldAllow()
						totalOps.Add(4)
					}
				}
			}()
		}

		wg.Wait()

		assert.False(t, panicOccurred.Load(), "Full system stress should not panic")
		assert.Greater(t, totalOps.Load(), int64(1000), "Should complete many operations")

		finalControllerStats := controller.Stats()
		finalBPStats := backpressure.Stats()

		t.Logf("Full stress test: %d total ops, Controller running=%v, BP state=%s, Transitions=%d",
			totalOps.Load(), finalControllerStats.IsRunning, finalBPStats.CurrentState,
			finalBPStats.TotalStateTransitions)
	})
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkStress_HighQPSRateLimiter(b *testing.B) {
	rateLimiter := NewTokenBucketLimiter(10000, 1000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rateLimiter.TryAcquire()
		}
	})
}

func BenchmarkStress_WorkerPoolSubmit(b *testing.B) {
	pool := NewWorkerPool(WorkerPoolConfig{
		MinSize:       10,
		MaxSize:       200,
		InitialSize:   50,
		TaskQueueSize: 10000,
	})

	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop()

	task := func(ctx context.Context) error {
		return nil
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pool.Submit(task)
		}
	})
}

func BenchmarkStress_BackpressureCheck(b *testing.B) {
	metrics := newStressMetrics()
	metrics.setErrorRate(0.05)
	handler := NewBackpressureHandler(metrics, DefaultBackpressureConfig())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			handler.Check()
		}
	})
}

func BenchmarkStress_ControllerStats(b *testing.B) {
	rateLimiter := NewTokenBucketLimiter(1000, 100)
	shaper := newStressTestShaper(1000)
	workerPool := NewWorkerPool(WorkerPoolConfig{
		MinSize:     5,
		MaxSize:     50,
		InitialSize: 20,
	})

	config := LoadControllerConfig{
		AdjustInterval: 10 * time.Millisecond,
	}
	controller := NewLoadController(rateLimiter, shaper, workerPool, nil, config)

	ctx := context.Background()
	controller.Start(ctx)
	defer controller.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			controller.Stats()
		}
	})
}
