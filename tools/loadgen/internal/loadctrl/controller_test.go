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

// mockTrafficShaper is a simple constant QPS shaper for testing.
type mockTrafficShaper struct {
	qps   float64
	phase string
}

func (m *mockTrafficShaper) GetTargetQPS(_ time.Duration) float64 {
	return m.qps
}

func (m *mockTrafficShaper) GetPhase(_ time.Duration) string {
	return m.phase
}

func (m *mockTrafficShaper) Name() string {
	return "mock"
}

func (m *mockTrafficShaper) Config() ShaperConfig {
	return ShaperConfig{Type: "mock", BaseQPS: m.qps}
}

// variableShaper allows changing QPS at runtime.
type variableShaper struct {
	qps atomic.Int64 // stored as int64 (QPS * 1000) for atomic access
}

func newVariableShaper(qps float64) *variableShaper {
	v := &variableShaper{}
	v.qps.Store(int64(qps * 1000))
	return v
}

func (v *variableShaper) SetQPS(qps float64) {
	v.qps.Store(int64(qps * 1000))
}

func (v *variableShaper) GetTargetQPS(_ time.Duration) float64 {
	return float64(v.qps.Load()) / 1000
}

func (v *variableShaper) GetPhase(_ time.Duration) string {
	return "variable"
}

func (v *variableShaper) Name() string {
	return "variable"
}

func (v *variableShaper) Config() ShaperConfig {
	return ShaperConfig{Type: "variable"}
}

func TestLoadController_BasicOperation(t *testing.T) {
	rateLimiter := NewTokenBucketLimiter(100, 10)
	shaper := &mockTrafficShaper{qps: 100, phase: "test"}
	metrics := NewMetricsCollector()

	controller := NewLoadController(rateLimiter, shaper, nil, metrics, LoadControllerConfig{
		AdjustInterval: 50 * time.Millisecond,
	})

	ctx := context.Background()
	controller.Start(ctx)
	defer controller.Stop()

	// Wait for at least one adjustment cycle
	time.Sleep(100 * time.Millisecond)

	stats := controller.Stats()
	assert.True(t, stats.IsRunning)
	assert.Equal(t, "test", stats.CurrentPhase)
	assert.InDelta(t, 100.0, stats.ActualQPS, 1.0)
}

func TestLoadController_QPSAdjustment(t *testing.T) {
	rateLimiter := NewTokenBucketLimiter(50, 10)
	shaper := newVariableShaper(50)
	metrics := NewMetricsCollector()

	var lastTarget, lastActual atomic.Int64 // Store as int64 (QPS * 1000)
	var adjustCount atomic.Int32

	controller := NewLoadController(rateLimiter, shaper, nil, metrics, LoadControllerConfig{
		AdjustInterval: 50 * time.Millisecond,
	})
	controller.SetOnQPSAdjust(func(target, actual float64) {
		lastTarget.Store(int64(target * 1000))
		lastActual.Store(int64(actual * 1000))
		adjustCount.Add(1)
	})

	ctx := context.Background()
	controller.Start(ctx)
	defer controller.Stop()

	// Wait for initial adjustment
	time.Sleep(100 * time.Millisecond)
	assert.InDelta(t, 50.0, float64(lastTarget.Load())/1000, 1.0)
	assert.InDelta(t, 50.0, float64(lastActual.Load())/1000, 1.0)

	// Change shaper QPS
	shaper.SetQPS(200)

	// Wait for adjustment
	time.Sleep(100 * time.Millisecond)
	assert.InDelta(t, 200.0, float64(lastTarget.Load())/1000, 1.0)
	assert.InDelta(t, 200.0, float64(lastActual.Load())/1000, 1.0)

	// Verify multiple adjustments occurred
	assert.True(t, adjustCount.Load() >= 2)
}

func TestLoadController_AdaptiveControl(t *testing.T) {
	rateLimiter := NewTokenBucketLimiter(100, 10)
	shaper := &mockTrafficShaper{qps: 100, phase: "test"}
	metrics := NewSlidingWindowMetrics(MetricsConfig{
		WindowSize: 10 * time.Second,
		BucketSize: 100 * time.Millisecond,
	})

	controller := NewLoadController(rateLimiter, shaper, nil, metrics, LoadControllerConfig{
		AdjustInterval:             50 * time.Millisecond,
		Adaptive:                   true,
		TargetP95:                  50 * time.Millisecond, // 50ms target
		AdaptiveReductionFactor:    0.1,                   // 10% reduction
		ConsecutiveBreachThreshold: 2,
		MinQPS:                     10,
	})

	ctx := context.Background()
	controller.Start(ctx)
	defer controller.Stop()

	// Record latencies within target
	for range 10 {
		metrics.RecordLatency(30 * time.Millisecond)
	}

	time.Sleep(100 * time.Millisecond)
	stats := controller.Stats()
	assert.InDelta(t, 100.0, stats.ActualQPS, 5.0)
	assert.False(t, stats.AdaptiveActive)

	// Record high latencies to trigger adaptive control
	for range 50 {
		metrics.RecordLatency(100 * time.Millisecond) // Above 50ms target
	}

	// Wait for multiple adjustment cycles
	time.Sleep(200 * time.Millisecond)

	stats = controller.Stats()
	// QPS should be reduced
	assert.Less(t, stats.ActualQPS, 100.0)
	assert.True(t, stats.AdaptiveActive)
}

func TestLoadController_WorkerPoolAutoScale(t *testing.T) {
	rateLimiter := NewTokenBucketLimiter(100, 10)
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
		AdjustInterval:      50 * time.Millisecond,
		WorkerAutoScale:     true,
		WorkerLatencyBuffer: 1.5,
		DefaultAvgLatency:   50 * time.Millisecond,
	})

	ctx := context.Background()
	controller.Start(ctx)
	defer controller.Stop()

	// Record some latencies
	for range 10 {
		metrics.RecordLatency(100 * time.Millisecond)
	}

	time.Sleep(150 * time.Millisecond)

	// With 100 QPS and 100ms latency, optimal workers = 100 * 0.1 * 1.5 = 15
	stats := controller.Stats()
	require.NotNil(t, stats.WorkerPoolStats)

	// Increase QPS
	shaper.SetQPS(500)

	time.Sleep(150 * time.Millisecond)

	// Workers should increase
	stats = controller.Stats()
	require.NotNil(t, stats.WorkerPoolStats)
	// With 500 QPS and 100ms latency, optimal workers = 500 * 0.1 * 1.5 = 75, clamped to 50
	assert.GreaterOrEqual(t, stats.WorkerPoolStats.CurrentSize, 10)
}

func TestLoadController_Acquire(t *testing.T) {
	rateLimiter := NewTokenBucketLimiter(100, 10)
	shaper := &mockTrafficShaper{qps: 100, phase: "test"}
	metrics := NewMetricsCollector()

	controller := NewLoadController(rateLimiter, shaper, nil, metrics, LoadControllerConfig{
		AdjustInterval: 50 * time.Millisecond,
	})

	ctx := context.Background()
	controller.Start(ctx)
	defer controller.Stop()

	// Acquire should work
	acquireCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	err := controller.Acquire(acquireCtx)
	assert.NoError(t, err)

	// TryAcquire should work
	assert.True(t, controller.TryAcquire())
}

func TestLoadController_StopCleanly(t *testing.T) {
	rateLimiter := NewTokenBucketLimiter(100, 10)
	shaper := &mockTrafficShaper{qps: 100, phase: "test"}
	workerPool := NewWorkerPool(WorkerPoolConfig{
		MinSize:     5,
		MaxSize:     20,
		InitialSize: 10,
	})

	controller := NewLoadController(rateLimiter, shaper, workerPool, nil, LoadControllerConfig{
		AdjustInterval: 50 * time.Millisecond,
	})

	ctx := context.Background()
	controller.Start(ctx)

	// Stop should complete without hanging
	done := make(chan struct{})
	go func() {
		controller.Stop()
		close(done)
	}()

	select {
	case <-done:
		// OK
	case <-time.After(time.Second):
		t.Fatal("Stop() hung")
	}

	stats := controller.Stats()
	assert.False(t, stats.IsRunning)
}

func TestLoadController_MinQPSFloor(t *testing.T) {
	rateLimiter := NewTokenBucketLimiter(100, 10)
	shaper := &mockTrafficShaper{qps: 100, phase: "test"}
	metrics := NewSlidingWindowMetrics(MetricsConfig{
		WindowSize: 10 * time.Second,
		BucketSize: 100 * time.Millisecond,
	})

	controller := NewLoadController(rateLimiter, shaper, nil, metrics, LoadControllerConfig{
		AdjustInterval:             50 * time.Millisecond,
		Adaptive:                   true,
		TargetP95:                  10 * time.Millisecond, // Very low target
		AdaptiveReductionFactor:    0.5,                   // Aggressive reduction
		ConsecutiveBreachThreshold: 1,
		MinQPS:                     50, // Floor at 50 QPS
	})

	ctx := context.Background()
	controller.Start(ctx)
	defer controller.Stop()

	// Record very high latencies
	for range 100 {
		metrics.RecordLatency(1 * time.Second)
	}

	// Wait for adjustment
	time.Sleep(200 * time.Millisecond)

	stats := controller.Stats()
	// QPS should not go below MinQPS
	assert.GreaterOrEqual(t, stats.ActualQPS, 50.0)
}

func TestLoadController_Responsiveness(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping responsiveness test in short mode")
	}

	rateLimiter := NewTokenBucketLimiter(100, 10)
	shaper := newVariableShaper(100)
	metrics := NewMetricsCollector()

	type adjustment struct {
		target float64
		actual float64
		time   time.Time
	}
	var adjustmentsMu sync.Mutex
	var adjustments []adjustment

	controller := NewLoadController(rateLimiter, shaper, nil, metrics, LoadControllerConfig{
		AdjustInterval: 100 * time.Millisecond, // 100ms interval as per spec
	})
	controller.SetOnQPSAdjust(func(target, actual float64) {
		adjustmentsMu.Lock()
		adjustments = append(adjustments, adjustment{target, actual, time.Now()})
		adjustmentsMu.Unlock()
	})

	ctx := context.Background()
	controller.Start(ctx)
	defer controller.Stop()

	// Wait for stable operation
	time.Sleep(300 * time.Millisecond)

	// Record change time
	changeTime := time.Now()
	shaper.SetQPS(200)

	// Wait for response
	time.Sleep(300 * time.Millisecond)

	// Find when the QPS change was detected
	adjustmentsMu.Lock()
	adjCopy := make([]adjustment, len(adjustments))
	copy(adjCopy, adjustments)
	adjustmentsMu.Unlock()

	var responseTime time.Duration
	for _, adj := range adjCopy {
		if adj.time.After(changeTime) && adj.target > 150 {
			responseTime = adj.time.Sub(changeTime)
			break
		}
	}

	// Response should be within 200ms (2 adjustment cycles)
	assert.Less(t, responseTime, 200*time.Millisecond,
		"Controller should respond to QPS change within 200ms, got %v", responseTime)
}

func TestLoadController_CalculateOptimalWorkers(t *testing.T) {
	tests := []struct {
		name          string
		targetQPS     float64
		avgLatency    time.Duration
		latencyBuffer float64
		minWorkers    int
		maxWorkers    int
		expected      int
	}{
		{
			name:          "basic calculation",
			targetQPS:     100,
			avgLatency:    100 * time.Millisecond,
			latencyBuffer: 1.5,
			minWorkers:    5,
			maxWorkers:    50,
			expected:      15, // 100 * 0.1 * 1.5 = 15
		},
		{
			name:          "clamped to min",
			targetQPS:     10,
			avgLatency:    10 * time.Millisecond,
			latencyBuffer: 1.5,
			minWorkers:    5,
			maxWorkers:    50,
			expected:      5, // 10 * 0.01 * 1.5 = 0.15, clamped to 5
		},
		{
			name:          "clamped to max",
			targetQPS:     1000,
			avgLatency:    500 * time.Millisecond,
			latencyBuffer: 1.5,
			minWorkers:    5,
			maxWorkers:    50,
			expected:      50, // 1000 * 0.5 * 1.5 = 750, clamped to 50
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rateLimiter := NewTokenBucketLimiter(tt.targetQPS, 10)
			shaper := &mockTrafficShaper{qps: tt.targetQPS, phase: "test"}
			metrics := NewSlidingWindowMetrics(MetricsConfig{
				WindowSize: 10 * time.Second,
				BucketSize: time.Second,
			})
			workerPool := NewWorkerPool(WorkerPoolConfig{
				MinSize:     tt.minWorkers,
				MaxSize:     tt.maxWorkers,
				InitialSize: 10,
			})

			controller := NewLoadController(rateLimiter, shaper, workerPool, metrics, LoadControllerConfig{
				WorkerAutoScale:     true,
				WorkerLatencyBuffer: tt.latencyBuffer,
			})

			// Record latencies to establish average
			for range 10 {
				metrics.RecordLatency(tt.avgLatency)
			}

			optimal := controller.calculateOptimalWorkers(tt.targetQPS)
			assert.Equal(t, tt.expected, optimal)
		})
	}
}

func TestLoadController_IntegrationWithShapers(t *testing.T) {
	// Test with actual SineWaveShaper
	shaperConfig := ShaperConfig{
		Type:      "sine",
		BaseQPS:   100,
		Amplitude: 50,
		Period:    1 * time.Second,
	}
	shaper, err := NewTrafficShaper(shaperConfig)
	require.NoError(t, err)

	rateLimiter := NewTokenBucketLimiter(100, 20)
	metrics := NewMetricsCollector()

	controller := NewLoadController(rateLimiter, shaper, nil, metrics, LoadControllerConfig{
		AdjustInterval: 50 * time.Millisecond,
	})

	ctx := context.Background()
	controller.Start(ctx)
	defer controller.Stop()

	// Run for a full cycle
	time.Sleep(1100 * time.Millisecond)

	// QPS should have varied
	stats := controller.Stats()
	assert.True(t, stats.IsRunning)
	// The actual QPS will be somewhere between 50 and 150
	assert.True(t, stats.ActualQPS >= 50 && stats.ActualQPS <= 150,
		"Expected QPS between 50-150, got %f", stats.ActualQPS)
}
