// Package loadctrl provides load control components including traffic shaping
// and rate limiting for the load generator.
package loadctrl

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// LoadController coordinates traffic shaping and rate limiting.
// It acts as the central control point for managing load generation.
//
// Thread Safety: Safe for concurrent use.
type LoadController struct {
	rateLimiter   RateLimiter
	trafficShaper TrafficShaper
	workerPool    *WorkerPool
	metrics       MetricsCollector

	startTime    time.Time
	adjustTicker *time.Ticker
	stopCh       chan struct{}
	wg           sync.WaitGroup
	isRunning    atomic.Bool

	// Configuration
	config LoadControllerConfig

	// Callbacks (protected by callbackMu)
	callbackMu  sync.RWMutex
	onQPSAdjust func(targetQPS, actualQPS float64)

	// Internal state (protected by stateMu)
	stateMu         sync.RWMutex
	lastTargetQPS   float64
	consecutiveHigh int // Count of consecutive P95 threshold breaches
}

// LoadControllerConfig holds configuration for the LoadController.
type LoadControllerConfig struct {
	// AdjustInterval is how often to adjust the rate limiter (default: 100ms).
	AdjustInterval time.Duration `yaml:"adjustInterval,omitempty" json:"adjustInterval,omitempty"`

	// Adaptive enables adaptive control based on latency.
	Adaptive bool `yaml:"adaptive" json:"adaptive"`

	// TargetP95 is the target P95 latency for adaptive control.
	TargetP95 time.Duration `yaml:"targetP95,omitempty" json:"targetP95,omitempty"`

	// AdaptiveReductionFactor is how much to reduce QPS when P95 exceeds target (0.0-1.0).
	// Default: 0.1 (reduce by 10%)
	AdaptiveReductionFactor float64 `yaml:"adaptiveReductionFactor,omitempty" json:"adaptiveReductionFactor,omitempty"`

	// AdaptiveRecoveryFactor is how much to increase QPS when recovering (0.0-1.0).
	// Default: 0.05 (increase by 5%)
	AdaptiveRecoveryFactor float64 `yaml:"adaptiveRecoveryFactor,omitempty" json:"adaptiveRecoveryFactor,omitempty"`

	// MinQPS is the minimum QPS floor for adaptive control.
	MinQPS float64 `yaml:"minQPS,omitempty" json:"minQPS,omitempty"`

	// ConsecutiveBreachThreshold is how many consecutive P95 breaches before reducing QPS.
	// Default: 2
	ConsecutiveBreachThreshold int `yaml:"consecutiveBreachThreshold,omitempty" json:"consecutiveBreachThreshold,omitempty"`

	// WorkerAutoScale enables automatic worker pool scaling.
	WorkerAutoScale bool `yaml:"workerAutoScale" json:"workerAutoScale"`

	// WorkerLatencyBuffer is the buffer multiplier for worker calculation.
	// Workers = QPS * AvgLatency * WorkerLatencyBuffer
	// Default: 1.5
	WorkerLatencyBuffer float64 `yaml:"workerLatencyBuffer,omitempty" json:"workerLatencyBuffer,omitempty"`

	// DefaultAvgLatency is the assumed average latency when no metrics are available.
	// Default: 50ms
	DefaultAvgLatency time.Duration `yaml:"defaultAvgLatency,omitempty" json:"defaultAvgLatency,omitempty"`
}

// LoadControllerStats contains statistics about the load controller.
type LoadControllerStats struct {
	// IsRunning indicates if the controller is currently running.
	IsRunning bool
	// ElapsedTime is the time since the controller started.
	ElapsedTime time.Duration
	// CurrentPhase is the current traffic shaping phase.
	CurrentPhase string
	// TargetQPS is the current target QPS from traffic shaper.
	TargetQPS float64
	// ActualQPS is the actual QPS being enforced by rate limiter.
	ActualQPS float64
	// RateLimiterStats contains rate limiter statistics.
	RateLimiterStats RateLimiterStats
	// WorkerPoolStats contains worker pool statistics.
	WorkerPoolStats *WorkerPoolStats
	// MetricsStats contains metrics statistics.
	MetricsStats *MetricsStats
	// AdaptiveActive indicates if adaptive control is currently reducing QPS.
	AdaptiveActive bool
}

// NewLoadController creates a new LoadController.
func NewLoadController(
	rateLimiter RateLimiter,
	trafficShaper TrafficShaper,
	workerPool *WorkerPool,
	metrics MetricsCollector,
	config LoadControllerConfig,
) *LoadController {
	// Validate required dependencies
	if rateLimiter == nil {
		panic("loadctrl: rateLimiter cannot be nil")
	}
	if trafficShaper == nil {
		panic("loadctrl: trafficShaper cannot be nil")
	}

	// Apply defaults
	if config.AdjustInterval <= 0 {
		config.AdjustInterval = 100 * time.Millisecond
	}
	if config.AdaptiveReductionFactor <= 0 {
		config.AdaptiveReductionFactor = 0.1
	}
	if config.AdaptiveRecoveryFactor <= 0 {
		config.AdaptiveRecoveryFactor = 0.05
	}
	if config.ConsecutiveBreachThreshold <= 0 {
		config.ConsecutiveBreachThreshold = 2
	}
	if config.WorkerLatencyBuffer <= 0 {
		config.WorkerLatencyBuffer = 1.5
	}
	if config.DefaultAvgLatency <= 0 {
		config.DefaultAvgLatency = 50 * time.Millisecond
	}

	return &LoadController{
		rateLimiter:   rateLimiter,
		trafficShaper: trafficShaper,
		workerPool:    workerPool,
		metrics:       metrics,
		config:        config,
		stopCh:        make(chan struct{}),
	}
}

// Start starts the load controller.
// It begins the periodic adjustment loop and starts the worker pool.
func (lc *LoadController) Start(ctx context.Context) {
	if lc.isRunning.Swap(true) {
		return // Already running
	}

	lc.startTime = time.Now()
	lc.adjustTicker = time.NewTicker(lc.config.AdjustInterval)
	lc.stopCh = make(chan struct{})

	// Start worker pool if available
	if lc.workerPool != nil {
		lc.workerPool.Start(ctx)
	}

	// Start the adjustment loop
	lc.wg.Add(1)
	go lc.runAdjustmentLoop(ctx)
}

// Stop stops the load controller.
func (lc *LoadController) Stop() {
	if !lc.isRunning.Swap(false) {
		return // Not running
	}

	close(lc.stopCh)
	if lc.adjustTicker != nil {
		lc.adjustTicker.Stop()
	}

	// Stop worker pool if available
	if lc.workerPool != nil {
		lc.workerPool.Stop()
	}

	lc.wg.Wait()
}

// Acquire acquires a request slot, blocking until available.
func (lc *LoadController) Acquire(ctx context.Context) error {
	return lc.rateLimiter.Acquire(ctx)
}

// TryAcquire attempts to acquire a request slot without blocking.
func (lc *LoadController) TryAcquire() bool {
	return lc.rateLimiter.TryAcquire()
}

// Stats returns statistics about the load controller.
func (lc *LoadController) Stats() LoadControllerStats {
	lc.stateMu.RLock()
	lastTargetQPS := lc.lastTargetQPS
	consecutiveHigh := lc.consecutiveHigh
	lc.stateMu.RUnlock()

	stats := LoadControllerStats{
		IsRunning: lc.isRunning.Load(),
		TargetQPS: lastTargetQPS,
		ActualQPS: lc.rateLimiter.CurrentRate(),
	}

	if lc.isRunning.Load() {
		stats.ElapsedTime = time.Since(lc.startTime)
		stats.CurrentPhase = lc.trafficShaper.GetPhase(stats.ElapsedTime)
	}

	stats.RateLimiterStats = lc.rateLimiter.Stats()

	if lc.workerPool != nil {
		wpStats := lc.workerPool.Stats()
		stats.WorkerPoolStats = &wpStats
	}

	if lc.metrics != nil {
		metricsStats := lc.metrics.GetStats()
		stats.MetricsStats = &metricsStats
	}

	stats.AdaptiveActive = consecutiveHigh >= lc.config.ConsecutiveBreachThreshold

	return stats
}

// SetOnQPSAdjust sets a callback that's called when QPS is adjusted.
func (lc *LoadController) SetOnQPSAdjust(callback func(targetQPS, actualQPS float64)) {
	lc.callbackMu.Lock()
	defer lc.callbackMu.Unlock()
	lc.onQPSAdjust = callback
}

// RateLimiter returns the underlying rate limiter.
func (lc *LoadController) RateLimiter() RateLimiter {
	return lc.rateLimiter
}

// TrafficShaper returns the underlying traffic shaper.
func (lc *LoadController) TrafficShaper() TrafficShaper {
	return lc.trafficShaper
}

// WorkerPool returns the underlying worker pool.
func (lc *LoadController) WorkerPool() *WorkerPool {
	return lc.workerPool
}

// Metrics returns the underlying metrics collector.
func (lc *LoadController) Metrics() MetricsCollector {
	return lc.metrics
}

// runAdjustmentLoop runs the periodic adjustment loop.
func (lc *LoadController) runAdjustmentLoop(ctx context.Context) {
	defer lc.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-lc.stopCh:
			return
		case <-lc.adjustTicker.C:
			lc.adjust()
		}
	}
}

// adjust performs a single adjustment cycle.
func (lc *LoadController) adjust() {
	elapsed := time.Since(lc.startTime)

	// Get target QPS from traffic shaper
	targetQPS := lc.trafficShaper.GetTargetQPS(elapsed)

	lc.stateMu.Lock()
	lc.lastTargetQPS = targetQPS
	lc.stateMu.Unlock()

	actualQPS := targetQPS

	// Apply adaptive control if enabled
	if lc.config.Adaptive && lc.metrics != nil {
		actualQPS = lc.applyAdaptiveControl(targetQPS)
	}

	// Update rate limiter
	lc.rateLimiter.SetRate(actualQPS)

	// Adjust worker pool size if enabled
	if lc.config.WorkerAutoScale && lc.workerPool != nil {
		optimalWorkers := lc.calculateOptimalWorkers(actualQPS)
		lc.workerPool.AdjustSize(optimalWorkers)
	}

	// Callback if set
	lc.callbackMu.RLock()
	cb := lc.onQPSAdjust
	lc.callbackMu.RUnlock()
	if cb != nil {
		cb(targetQPS, actualQPS)
	}
}

// applyAdaptiveControl applies adaptive control based on latency metrics.
// Returns the adjusted QPS.
func (lc *LoadController) applyAdaptiveControl(targetQPS float64) float64 {
	if lc.config.TargetP95 <= 0 {
		return targetQPS
	}

	currentP95 := lc.metrics.GetP95Latency()
	if currentP95 <= 0 {
		// No latency data yet, use target
		lc.stateMu.Lock()
		lc.consecutiveHigh = 0
		lc.stateMu.Unlock()
		return targetQPS
	}

	lc.stateMu.Lock()
	defer lc.stateMu.Unlock()

	if currentP95 > lc.config.TargetP95 {
		// P95 exceeds target
		lc.consecutiveHigh++
		if lc.consecutiveHigh >= lc.config.ConsecutiveBreachThreshold {
			// Reduce QPS
			reduction := 1.0 - lc.config.AdaptiveReductionFactor
			adjustedQPS := targetQPS * reduction

			// Apply minimum floor
			if lc.config.MinQPS > 0 && adjustedQPS < lc.config.MinQPS {
				adjustedQPS = lc.config.MinQPS
			}

			return adjustedQPS
		}
	} else {
		// P95 is within target
		if lc.consecutiveHigh > 0 {
			// We were previously reducing, now recovering
			lc.consecutiveHigh = 0
		}
	}

	return targetQPS
}

// calculateOptimalWorkers calculates the optimal number of workers based on QPS and latency.
// Formula: Workers = QPS * AvgLatency(seconds) * LatencyBuffer
func (lc *LoadController) calculateOptimalWorkers(targetQPS float64) int {
	avgLatency := lc.config.DefaultAvgLatency

	if lc.metrics != nil {
		measuredLatency := lc.metrics.GetAvgLatency()
		if measuredLatency > 0 {
			avgLatency = measuredLatency
		}
	}

	// Calculate optimal workers
	// Workers needed = QPS * average processing time (in seconds)
	// Add buffer for variability
	optimal := int(targetQPS * avgLatency.Seconds() * lc.config.WorkerLatencyBuffer)

	// Clamp to pool limits
	if lc.workerPool != nil {
		minSize := lc.workerPool.MinSize()
		maxSize := lc.workerPool.MaxSize()

		if optimal < minSize {
			optimal = minSize
		}
		if optimal > maxSize {
			optimal = maxSize
		}
	}

	return max(1, optimal)
}

// Elapsed returns the elapsed time since the controller started.
func (lc *LoadController) Elapsed() time.Duration {
	if !lc.isRunning.Load() {
		return 0
	}
	return time.Since(lc.startTime)
}

// CurrentPhase returns the current traffic shaping phase.
func (lc *LoadController) CurrentPhase() string {
	if !lc.isRunning.Load() {
		return "stopped"
	}
	return lc.trafficShaper.GetPhase(time.Since(lc.startTime))
}

// TargetQPS returns the current target QPS from the traffic shaper.
func (lc *LoadController) TargetQPS() float64 {
	lc.stateMu.RLock()
	defer lc.stateMu.RUnlock()
	return lc.lastTargetQPS
}

// ActualQPS returns the current QPS being enforced by the rate limiter.
func (lc *LoadController) ActualQPS() float64 {
	return lc.rateLimiter.CurrentRate()
}

// IsAdaptiveActive returns true if adaptive control is currently reducing QPS.
func (lc *LoadController) IsAdaptiveActive() bool {
	lc.stateMu.RLock()
	defer lc.stateMu.RUnlock()
	return lc.consecutiveHigh >= lc.config.ConsecutiveBreachThreshold
}
