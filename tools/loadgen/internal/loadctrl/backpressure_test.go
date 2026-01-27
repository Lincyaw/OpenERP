package loadctrl

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMetricsCollector is a mock implementation of MetricsCollector for testing.
type mockMetricsCollector struct {
	mu         sync.RWMutex
	errorRate  float64
	p99Latency time.Duration
	p95Latency time.Duration
	avgLatency time.Duration
}

func newMockMetrics() *mockMetricsCollector {
	return &mockMetricsCollector{}
}

func (m *mockMetricsCollector) setErrorRate(rate float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorRate = rate
}

func (m *mockMetricsCollector) setP99Latency(latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.p99Latency = latency
}

func (m *mockMetricsCollector) setP95Latency(latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.p95Latency = latency
}

func (m *mockMetricsCollector) RecordLatency(latency time.Duration) {}
func (m *mockMetricsCollector) RecordError()                        {}
func (m *mockMetricsCollector) RecordSuccess()                      {}

func (m *mockMetricsCollector) GetP95Latency() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.p95Latency
}

func (m *mockMetricsCollector) GetP99Latency() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.p99Latency
}

func (m *mockMetricsCollector) GetAvgLatency() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.avgLatency
}

func (m *mockMetricsCollector) GetErrorRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.errorRate
}

func (m *mockMetricsCollector) GetStats() MetricsStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return MetricsStats{
		ErrorRate:  m.errorRate,
		P99Latency: m.p99Latency,
		P95Latency: m.p95Latency,
		AvgLatency: m.avgLatency,
	}
}

func (m *mockMetricsCollector) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorRate = 0
	m.p99Latency = 0
	m.p95Latency = 0
	m.avgLatency = 0
}

func TestNewBackpressureHandler(t *testing.T) {
	t.Parallel()

	t.Run("creates handler with default config", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		handler := NewBackpressureHandler(metrics, BackpressureConfig{})

		require.NotNil(t, handler)
		assert.Equal(t, BackpressureStateNormal, handler.State())
		assert.Equal(t, BackpressureStrategyReduce, handler.config.Strategy)
		assert.InDelta(t, 0.1, handler.config.ErrorRateThreshold, 0.001)
		assert.Equal(t, time.Second, handler.config.LatencyP99Threshold)
		assert.Equal(t, 30*time.Second, handler.config.RecoveryPeriod)
	})

	t.Run("creates handler with custom config", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		config := BackpressureConfig{
			Strategy:            BackpressureStrategyDrop,
			ErrorRateThreshold:  0.2,
			LatencyP99Threshold: 2 * time.Second,
			RecoveryPeriod:      60 * time.Second,
		}
		handler := NewBackpressureHandler(metrics, config)

		require.NotNil(t, handler)
		assert.Equal(t, BackpressureStrategyDrop, handler.config.Strategy)
		assert.InDelta(t, 0.2, handler.config.ErrorRateThreshold, 0.001)
		assert.Equal(t, 2*time.Second, handler.config.LatencyP99Threshold)
		assert.Equal(t, 60*time.Second, handler.config.RecoveryPeriod)
	})

	t.Run("applies warning thresholds as half of main thresholds", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		config := BackpressureConfig{
			ErrorRateThreshold:  0.2,
			LatencyP99Threshold: 2 * time.Second,
		}
		handler := NewBackpressureHandler(metrics, config)

		assert.InDelta(t, 0.1, handler.config.WarningErrorThreshold, 0.001)
		assert.Equal(t, time.Second, handler.config.WarningLatencyThreshold)
	})
}

func TestBackpressureHandler_StateTransitions(t *testing.T) {
	t.Parallel()

	t.Run("normal to warning on elevated error rate", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		config := BackpressureConfig{
			ErrorRateThreshold:         0.1,
			WarningErrorThreshold:      0.05,
			ConsecutiveBreachThreshold: 1,
		}
		handler := NewBackpressureHandler(metrics, config)

		// Set error rate above warning but below critical
		metrics.setErrorRate(0.06)

		// Check should transition to warning
		handler.Check()
		assert.Equal(t, BackpressureStateWarning, handler.State())
	})

	t.Run("normal to critical on high error rate", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		config := BackpressureConfig{
			ErrorRateThreshold:         0.1,
			ConsecutiveBreachThreshold: 1,
		}
		handler := NewBackpressureHandler(metrics, config)

		// Set error rate above critical threshold
		metrics.setErrorRate(0.15)

		// First check: normal -> warning (even with critical error rate, needs consecutive breaches)
		handler.Check()
		// Second check: should go to critical since we meet breach threshold
		handler.Check()

		// With consecutive breach threshold of 1, first check that exceeds
		// critical threshold should trigger warning, second should trigger critical
		assert.Equal(t, BackpressureStateCritical, handler.State())
	})

	t.Run("normal to critical on high P99 latency", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		config := BackpressureConfig{
			LatencyP99Threshold:        time.Second,
			ConsecutiveBreachThreshold: 1,
		}
		handler := NewBackpressureHandler(metrics, config)

		// Set P99 latency above threshold
		metrics.setP99Latency(1500 * time.Millisecond)

		// Trigger state transitions
		handler.Check()
		handler.Check()

		assert.Equal(t, BackpressureStateCritical, handler.State())
	})

	t.Run("critical to recovery when metrics improve", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		config := BackpressureConfig{
			ErrorRateThreshold:         0.1,
			WarningErrorThreshold:      0.05,
			ConsecutiveBreachThreshold: 1,
			RecoveryPeriod:             100 * time.Millisecond,
		}
		handler := NewBackpressureHandler(metrics, config)

		// Drive to critical state
		metrics.setErrorRate(0.15)
		for range 5 {
			handler.Check()
		}
		require.Equal(t, BackpressureStateCritical, handler.State())

		// Improve metrics
		metrics.setErrorRate(0.01)

		// Should transition to recovery
		handler.Check()
		assert.Equal(t, BackpressureStateRecovery, handler.State())
	})

	t.Run("recovery to normal after recovery period", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		config := BackpressureConfig{
			ErrorRateThreshold:         0.1,
			WarningErrorThreshold:      0.05,
			ConsecutiveBreachThreshold: 1,
			RecoveryPeriod:             50 * time.Millisecond,
		}
		handler := NewBackpressureHandler(metrics, config)

		// Drive to critical state
		metrics.setErrorRate(0.15)
		for range 3 {
			handler.Check()
		}
		require.Equal(t, BackpressureStateCritical, handler.State())

		// Improve metrics
		metrics.setErrorRate(0.01)
		handler.Check()
		require.Equal(t, BackpressureStateRecovery, handler.State())

		// Wait for recovery period
		time.Sleep(60 * time.Millisecond)

		// Should transition to normal
		handler.Check()
		assert.Equal(t, BackpressureStateNormal, handler.State())
	})

	t.Run("recovery back to critical on spike", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		config := BackpressureConfig{
			ErrorRateThreshold:         0.1,
			WarningErrorThreshold:      0.05,
			ConsecutiveBreachThreshold: 1,
			RecoveryPeriod:             5 * time.Second,
		}
		handler := NewBackpressureHandler(metrics, config)

		// Drive to critical, then recovery
		metrics.setErrorRate(0.15)
		for range 3 {
			handler.Check()
		}
		metrics.setErrorRate(0.01)
		handler.Check()
		require.Equal(t, BackpressureStateRecovery, handler.State())

		// Spike in errors
		metrics.setErrorRate(0.15)
		handler.Check()

		assert.Equal(t, BackpressureStateCritical, handler.State())
	})
}

func TestBackpressureHandler_Strategies(t *testing.T) {
	t.Parallel()

	t.Run("reduce strategy returns QPS multiplier", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		config := BackpressureConfig{
			Strategy:                   BackpressureStrategyReduce,
			ErrorRateThreshold:         0.1,
			ConsecutiveBreachThreshold: 1,
			ReductionFactor:            0.5,
		}
		handler := NewBackpressureHandler(metrics, config)

		// Drive to critical
		metrics.setErrorRate(0.15)
		for range 3 {
			handler.Check()
		}
		require.Equal(t, BackpressureStateCritical, handler.State())

		action := handler.Check()
		assert.InDelta(t, 0.5, action.QPSMultiplier, 0.001)
		assert.False(t, action.ShouldDrop)
		assert.False(t, action.ShouldPause)
	})

	t.Run("drop strategy marks requests for dropping", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		config := BackpressureConfig{
			Strategy:                   BackpressureStrategyDrop,
			ErrorRateThreshold:         0.1,
			ConsecutiveBreachThreshold: 1,
			DropPercentage:             0.5,
		}
		handler := NewBackpressureHandler(metrics, config)

		// Drive to critical
		metrics.setErrorRate(0.15)
		for range 3 {
			handler.Check()
		}
		require.Equal(t, BackpressureStateCritical, handler.State())

		// Check multiple times to verify probabilistic dropping
		dropCount := 0
		for range 100 {
			action := handler.Check()
			if action.ShouldDrop {
				dropCount++
			}
		}

		// With 50% drop rate, expect roughly half to be dropped
		assert.InDelta(t, 50, dropCount, 15)
	})

	t.Run("pause strategy sets should pause", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		config := BackpressureConfig{
			Strategy:                   BackpressureStrategyPause,
			ErrorRateThreshold:         0.1,
			ConsecutiveBreachThreshold: 1,
			CircuitOpenDuration:        5 * time.Second,
		}
		handler := NewBackpressureHandler(metrics, config)

		// Drive to critical
		metrics.setErrorRate(0.15)
		for range 3 {
			handler.Check()
		}
		require.Equal(t, BackpressureStateCritical, handler.State())

		action := handler.Check()
		assert.True(t, action.ShouldPause)
		assert.Equal(t, 5*time.Second, action.PauseDuration)
	})

	t.Run("circuit strategy opens circuit breaker", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		config := BackpressureConfig{
			Strategy:                   BackpressureStrategyCircuit,
			ErrorRateThreshold:         0.1,
			ConsecutiveBreachThreshold: 1,
			CircuitOpenDuration:        50 * time.Millisecond,
		}
		handler := NewBackpressureHandler(metrics, config)

		// Drive to critical
		metrics.setErrorRate(0.15)
		for range 3 {
			handler.Check()
		}
		require.Equal(t, BackpressureStateCritical, handler.State())

		// Circuit should be open, requests dropped
		action := handler.Check()
		assert.True(t, action.ShouldDrop)

		// Wait for circuit open duration
		time.Sleep(60 * time.Millisecond)

		// Circuit should be half-open, allow some probes
		allowedCount := 0
		for range 10 {
			action = handler.Check()
			if !action.ShouldDrop {
				allowedCount++
			}
		}
		assert.Greater(t, allowedCount, 0)
	})
}

func TestBackpressureHandler_ShouldAllow(t *testing.T) {
	t.Parallel()

	t.Run("allows in normal state", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		handler := NewBackpressureHandler(metrics, BackpressureConfig{})

		assert.True(t, handler.ShouldAllow())
	})

	t.Run("allows in warning state", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		config := BackpressureConfig{
			ErrorRateThreshold:         0.1,
			WarningErrorThreshold:      0.05,
			ConsecutiveBreachThreshold: 1,
		}
		handler := NewBackpressureHandler(metrics, config)

		metrics.setErrorRate(0.06)
		handler.Check()
		require.Equal(t, BackpressureStateWarning, handler.State())

		assert.True(t, handler.ShouldAllow())
	})

	t.Run("allows in recovery state", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		config := BackpressureConfig{
			ErrorRateThreshold:         0.1,
			ConsecutiveBreachThreshold: 1,
		}
		handler := NewBackpressureHandler(metrics, config)

		// Drive to critical then recovery
		metrics.setErrorRate(0.15)
		for range 3 {
			handler.Check()
		}
		metrics.setErrorRate(0.01)
		handler.Check()
		require.Equal(t, BackpressureStateRecovery, handler.State())

		assert.True(t, handler.ShouldAllow())
	})

	t.Run("blocks in critical state with pause strategy", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		config := BackpressureConfig{
			Strategy:                   BackpressureStrategyPause,
			ErrorRateThreshold:         0.1,
			ConsecutiveBreachThreshold: 1,
		}
		handler := NewBackpressureHandler(metrics, config)

		metrics.setErrorRate(0.15)
		for range 3 {
			handler.Check()
		}
		require.Equal(t, BackpressureStateCritical, handler.State())

		assert.False(t, handler.ShouldAllow())
	})
}

func TestBackpressureHandler_Stats(t *testing.T) {
	t.Parallel()

	t.Run("tracks state transitions", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		config := BackpressureConfig{
			ErrorRateThreshold:         0.1,
			WarningErrorThreshold:      0.05,
			ConsecutiveBreachThreshold: 1,
		}
		handler := NewBackpressureHandler(metrics, config)

		// Drive through state transitions
		metrics.setErrorRate(0.06) // warning
		handler.Check()
		metrics.setErrorRate(0.15) // critical
		handler.Check()
		handler.Check()
		metrics.setErrorRate(0.01) // recovery
		handler.Check()

		stats := handler.Stats()
		assert.Greater(t, stats.TotalStateTransitions, int64(0))
	})

	t.Run("tracks dropped requests", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		config := BackpressureConfig{
			Strategy:                   BackpressureStrategyDrop,
			ErrorRateThreshold:         0.1,
			ConsecutiveBreachThreshold: 1,
			DropPercentage:             1.0, // Drop all
		}
		handler := NewBackpressureHandler(metrics, config)

		// Drive to critical
		metrics.setErrorRate(0.15)
		for range 3 {
			handler.Check()
		}

		// Check multiple times
		for range 10 {
			handler.Check()
		}

		stats := handler.Stats()
		assert.Greater(t, stats.TotalDropped, int64(0))
	})

	t.Run("tracks last metrics", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		handler := NewBackpressureHandler(metrics, BackpressureConfig{})

		metrics.setErrorRate(0.05)
		metrics.setP99Latency(200 * time.Millisecond)
		handler.Check()

		stats := handler.Stats()
		assert.InDelta(t, 0.05, stats.LastErrorRate, 0.001)
		assert.Equal(t, 200*time.Millisecond, stats.LastP99Latency)
	})
}

func TestBackpressureHandler_Configuration(t *testing.T) {
	t.Parallel()

	t.Run("SetStrategy changes strategy", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		handler := NewBackpressureHandler(metrics, BackpressureConfig{
			Strategy: BackpressureStrategyReduce,
		})

		handler.SetStrategy(BackpressureStrategyDrop)
		assert.Equal(t, BackpressureStrategyDrop, handler.config.Strategy)
	})

	t.Run("SetErrorThreshold updates threshold", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		handler := NewBackpressureHandler(metrics, BackpressureConfig{})

		handler.SetErrorThreshold(0.2)
		assert.InDelta(t, 0.2, handler.config.ErrorRateThreshold, 0.001)
		assert.InDelta(t, 0.1, handler.config.WarningErrorThreshold, 0.001)
	})

	t.Run("SetLatencyThreshold updates threshold", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		handler := NewBackpressureHandler(metrics, BackpressureConfig{})

		handler.SetLatencyThreshold(2 * time.Second)
		assert.Equal(t, 2*time.Second, handler.config.LatencyP99Threshold)
		assert.Equal(t, time.Second, handler.config.WarningLatencyThreshold)
	})

	t.Run("invalid thresholds are ignored", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		handler := NewBackpressureHandler(metrics, BackpressureConfig{
			ErrorRateThreshold:  0.1,
			LatencyP99Threshold: time.Second,
		})

		handler.SetErrorThreshold(-0.1)
		assert.InDelta(t, 0.1, handler.config.ErrorRateThreshold, 0.001)

		handler.SetErrorThreshold(1.5)
		assert.InDelta(t, 0.1, handler.config.ErrorRateThreshold, 0.001)

		handler.SetLatencyThreshold(-time.Second)
		assert.Equal(t, time.Second, handler.config.LatencyP99Threshold)
	})
}

func TestBackpressureHandler_Reset(t *testing.T) {
	t.Parallel()

	t.Run("reset returns to normal state", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		config := BackpressureConfig{
			ErrorRateThreshold:         0.1,
			ConsecutiveBreachThreshold: 1,
		}
		handler := NewBackpressureHandler(metrics, config)

		// Drive to critical
		metrics.setErrorRate(0.15)
		for range 3 {
			handler.Check()
		}
		require.Equal(t, BackpressureStateCritical, handler.State())

		// Reset
		handler.Reset()

		assert.Equal(t, BackpressureStateNormal, handler.State())
		stats := handler.Stats()
		assert.Equal(t, int64(0), stats.TotalStateTransitions)
		assert.Equal(t, int64(0), stats.TotalDropped)
	})
}

func TestBackpressureHandler_StartStop(t *testing.T) {
	t.Parallel()

	t.Run("starts and stops monitoring loop", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		config := BackpressureConfig{
			CheckInterval:              10 * time.Millisecond,
			ErrorRateThreshold:         0.1,
			ConsecutiveBreachThreshold: 1,
		}
		handler := NewBackpressureHandler(metrics, config)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		handler.Start(ctx)
		assert.True(t, handler.isRunning.Load())

		// Let it run for a bit
		metrics.setErrorRate(0.15)
		time.Sleep(50 * time.Millisecond)

		handler.Stop()
		assert.False(t, handler.isRunning.Load())

		// State should have transitioned due to monitoring
		stats := handler.Stats()
		assert.Greater(t, stats.TotalStateTransitions, int64(0))
	})

	t.Run("stop is idempotent", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		handler := NewBackpressureHandler(metrics, BackpressureConfig{})

		// Stop without starting
		handler.Stop()
		assert.False(t, handler.isRunning.Load())
	})

	t.Run("start is idempotent", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		config := BackpressureConfig{
			CheckInterval: 10 * time.Millisecond,
		}
		handler := NewBackpressureHandler(metrics, config)

		ctx := context.Background()
		handler.Start(ctx)
		handler.Start(ctx) // Second start should be no-op

		assert.True(t, handler.isRunning.Load())
		handler.Stop()
	})
}

func TestBackpressureHandler_RecoveryAction(t *testing.T) {
	t.Parallel()

	t.Run("recovery QPS multiplier increases over time", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		config := BackpressureConfig{
			ErrorRateThreshold:         0.1,
			ConsecutiveBreachThreshold: 1,
			RecoveryPeriod:             200 * time.Millisecond,
		}
		handler := NewBackpressureHandler(metrics, config)

		// Drive to recovery
		metrics.setErrorRate(0.15)
		for range 3 {
			handler.Check()
		}
		metrics.setErrorRate(0.01)
		handler.Check()
		require.Equal(t, BackpressureStateRecovery, handler.State())

		// Check QPS multiplier at start of recovery
		action1 := handler.Check()
		assert.InDelta(t, 0.5, action1.QPSMultiplier, 0.1)

		// Wait for half the recovery period
		time.Sleep(100 * time.Millisecond)

		// Check QPS multiplier mid-recovery
		action2 := handler.Check()
		assert.Greater(t, action2.QPSMultiplier, action1.QPSMultiplier)

		// Wait for full recovery
		time.Sleep(150 * time.Millisecond)

		// Check QPS multiplier at end of recovery
		action3 := handler.Check()
		// Should be close to 1.0 or already transitioned to normal
		assert.GreaterOrEqual(t, action3.QPSMultiplier, 0.9)
	})
}

func TestBackpressureHandler_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	t.Run("handles concurrent Check calls", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		config := BackpressureConfig{
			ErrorRateThreshold:         0.1,
			ConsecutiveBreachThreshold: 1,
		}
		handler := NewBackpressureHandler(metrics, config)

		var wg sync.WaitGroup
		for range 100 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range 100 {
					handler.Check()
					handler.ShouldAllow()
					handler.State()
					handler.Stats()
				}
			}()
		}
		wg.Wait()

		// Should complete without race conditions
		assert.NotNil(t, handler.State())
	})

	t.Run("handles concurrent state changes", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		config := BackpressureConfig{
			ErrorRateThreshold:         0.1,
			ConsecutiveBreachThreshold: 1,
		}
		handler := NewBackpressureHandler(metrics, config)

		var wg sync.WaitGroup

		// Change metrics rapidly
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				metrics.setErrorRate(0.15)
				time.Sleep(time.Millisecond)
				metrics.setErrorRate(0.01)
				time.Sleep(time.Millisecond)
			}
		}()

		// Check concurrently
		for range 10 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range 100 {
					handler.Check()
				}
			}()
		}

		wg.Wait()
	})

	t.Run("handles concurrent start and stop", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		config := BackpressureConfig{
			CheckInterval: time.Millisecond,
		}
		handler := NewBackpressureHandler(metrics, config)

		ctx := context.Background()
		var wg sync.WaitGroup

		// Rapidly call Start and Stop from multiple goroutines
		for range 50 {
			wg.Add(2)
			go func() {
				defer wg.Done()
				handler.Start(ctx)
			}()
			go func() {
				defer wg.Done()
				handler.Stop()
			}()
		}

		wg.Wait()
		handler.Stop() // Ensure stopped at end
		assert.False(t, handler.isRunning.Load())
	})

	t.Run("handles concurrent config modifications", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		handler := NewBackpressureHandler(metrics, BackpressureConfig{})

		var wg sync.WaitGroup

		// Modify config from multiple goroutines
		for range 50 {
			wg.Add(3)
			go func() {
				defer wg.Done()
				for range 100 {
					handler.SetStrategy(BackpressureStrategyDrop)
					handler.SetStrategy(BackpressureStrategyReduce)
				}
			}()
			go func() {
				defer wg.Done()
				for range 100 {
					handler.SetErrorThreshold(0.1)
					handler.SetErrorThreshold(0.2)
				}
			}()
			go func() {
				defer wg.Done()
				for range 100 {
					handler.Check()
					handler.ShouldAllow()
				}
			}()
		}

		wg.Wait()
	})
}

func TestDefaultBackpressureConfig(t *testing.T) {
	t.Parallel()

	config := DefaultBackpressureConfig()

	assert.Equal(t, BackpressureStrategyReduce, config.Strategy)
	assert.InDelta(t, 0.1, config.ErrorRateThreshold, 0.001)
	assert.Equal(t, time.Second, config.LatencyP99Threshold)
	assert.InDelta(t, 0.05, config.WarningErrorThreshold, 0.001)
	assert.Equal(t, 500*time.Millisecond, config.WarningLatencyThreshold)
	assert.Equal(t, 30*time.Second, config.RecoveryPeriod)
	assert.Equal(t, 100*time.Millisecond, config.CheckInterval)
	assert.InDelta(t, 0.5, config.ReductionFactor, 0.001)
	assert.InDelta(t, 0.5, config.DropPercentage, 0.001)
	assert.Equal(t, 10*time.Second, config.CircuitOpenDuration)
	assert.Equal(t, 3, config.ConsecutiveBreachThreshold)
}

func TestBackpressureHandler_WarningState(t *testing.T) {
	t.Parallel()

	t.Run("warning state has slight QPS reduction", func(t *testing.T) {
		t.Parallel()
		metrics := newMockMetrics()
		config := BackpressureConfig{
			ErrorRateThreshold:         0.1,
			WarningErrorThreshold:      0.05,
			ConsecutiveBreachThreshold: 1,
		}
		handler := NewBackpressureHandler(metrics, config)

		// Drive to warning
		metrics.setErrorRate(0.06)
		handler.Check()
		require.Equal(t, BackpressureStateWarning, handler.State())

		action := handler.Check()
		assert.InDelta(t, 0.9, action.QPSMultiplier, 0.001)
		assert.Contains(t, action.Reason, "warning")
	})
}

func TestBackpressureHandler_NilMetrics(t *testing.T) {
	t.Parallel()

	t.Run("handles nil metrics gracefully", func(t *testing.T) {
		t.Parallel()
		handler := NewBackpressureHandler(nil, BackpressureConfig{})

		// Should not panic
		action := handler.Check()
		assert.Equal(t, BackpressureStateNormal, action.State)
		assert.True(t, handler.ShouldAllow())
	})
}

func TestFloat64Conversion(t *testing.T) {
	t.Parallel()

	testCases := []float64{0.0, 0.1, 0.5, 0.999, 1.0, 100.0, -0.5}

	for _, tc := range testCases {
		bits := float64ToBits(tc)
		result := float64FromBits(bits)
		assert.InDelta(t, tc, result, 0.0001)
	}
}
