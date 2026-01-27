// Package loadctrl provides load control components including traffic shaping
// and rate limiting for the load generator.
package loadctrl

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// BackpressureState represents the current state of the backpressure handler.
type BackpressureState string

const (
	// BackpressureStateNormal indicates normal operation with no backpressure.
	BackpressureStateNormal BackpressureState = "normal"
	// BackpressureStateWarning indicates elevated error rates or latency.
	BackpressureStateWarning BackpressureState = "warning"
	// BackpressureStateCritical indicates critical conditions requiring immediate action.
	BackpressureStateCritical BackpressureState = "critical"
	// BackpressureStateRecovery indicates the system is recovering from a critical state.
	BackpressureStateRecovery BackpressureState = "recovery"
)

// BackpressureStrategy defines how to respond to backpressure conditions.
type BackpressureStrategy string

const (
	// BackpressureStrategyDrop drops requests when backpressure is detected.
	BackpressureStrategyDrop BackpressureStrategy = "drop"
	// BackpressureStrategyReduce reduces the QPS when backpressure is detected.
	BackpressureStrategyReduce BackpressureStrategy = "reduce"
	// BackpressureStrategyPause pauses all requests when backpressure is detected.
	BackpressureStrategyPause BackpressureStrategy = "pause"
	// BackpressureStrategyCircuit triggers a circuit breaker when backpressure is detected.
	BackpressureStrategyCircuit BackpressureStrategy = "circuit"
)

// BackpressureAction represents an action to take in response to backpressure.
type BackpressureAction struct {
	// ShouldDrop indicates if the request should be dropped.
	ShouldDrop bool
	// ShouldPause indicates if request processing should pause.
	ShouldPause bool
	// QPSMultiplier is the multiplier to apply to QPS (1.0 = no change, 0.5 = reduce by half).
	QPSMultiplier float64
	// PauseDuration is how long to pause if ShouldPause is true.
	PauseDuration time.Duration
	// State is the current backpressure state.
	State BackpressureState
	// Reason describes why this action was taken.
	Reason string
}

// BackpressureHandler manages backpressure detection and response.
// It monitors metrics and triggers appropriate actions based on thresholds.
//
// Thread Safety: Safe for concurrent use.
type BackpressureHandler interface {
	// Check evaluates current metrics and returns an action to take.
	Check() BackpressureAction

	// ShouldAllow returns true if the request should be allowed based on current state.
	ShouldAllow() bool

	// State returns the current backpressure state.
	State() BackpressureState

	// Stats returns statistics about backpressure handling.
	Stats() BackpressureStats

	// SetStrategy sets the backpressure strategy.
	SetStrategy(strategy BackpressureStrategy)

	// SetErrorThreshold sets the error rate threshold (0.0 - 1.0).
	SetErrorThreshold(threshold float64)

	// SetLatencyThreshold sets the P99 latency threshold.
	SetLatencyThreshold(threshold time.Duration)

	// Reset resets the handler to normal state.
	Reset()

	// Start starts the background monitoring loop.
	Start(ctx context.Context)

	// Stop stops the background monitoring loop.
	Stop()
}

// BackpressureStats contains statistics about backpressure handling.
type BackpressureStats struct {
	// CurrentState is the current backpressure state.
	CurrentState BackpressureState
	// TotalStateTransitions is the number of state transitions.
	TotalStateTransitions int64
	// TotalDropped is the number of requests dropped due to backpressure.
	TotalDropped int64
	// TotalPaused is the number of times requests were paused.
	TotalPaused int64
	// TotalReduced is the number of times QPS was reduced.
	TotalReduced int64
	// TimeInNormal is the total time spent in normal state.
	TimeInNormal time.Duration
	// TimeInWarning is the total time spent in warning state.
	TimeInWarning time.Duration
	// TimeInCritical is the total time spent in critical state.
	TimeInCritical time.Duration
	// TimeInRecovery is the total time spent in recovery state.
	TimeInRecovery time.Duration
	// LastErrorRate is the last recorded error rate.
	LastErrorRate float64
	// LastP99Latency is the last recorded P99 latency.
	LastP99Latency time.Duration
	// RecoveryStartTime is when recovery started (if in recovery).
	RecoveryStartTime time.Time
}

// BackpressureConfig holds configuration for the backpressure handler.
type BackpressureConfig struct {
	// Strategy is the backpressure response strategy (default: reduce).
	Strategy BackpressureStrategy `yaml:"strategy" json:"strategy"`
	// ErrorRateThreshold is the error rate threshold to trigger backpressure (default: 0.1 = 10%).
	ErrorRateThreshold float64 `yaml:"errorRateThreshold" json:"errorRateThreshold"`
	// LatencyP99Threshold is the P99 latency threshold to trigger backpressure (default: 1s).
	LatencyP99Threshold time.Duration `yaml:"latencyP99Threshold" json:"latencyP99Threshold"`
	// WarningErrorThreshold is the error rate to enter warning state (default: 0.05 = 5%).
	WarningErrorThreshold float64 `yaml:"warningErrorThreshold,omitempty" json:"warningErrorThreshold,omitempty"`
	// WarningLatencyThreshold is the P99 latency to enter warning state (default: 500ms).
	WarningLatencyThreshold time.Duration `yaml:"warningLatencyThreshold,omitempty" json:"warningLatencyThreshold,omitempty"`
	// RecoveryPeriod is how long to wait in recovery before returning to normal (default: 30s).
	RecoveryPeriod time.Duration `yaml:"recoveryPeriod" json:"recoveryPeriod"`
	// CheckInterval is how often to check metrics (default: 100ms).
	CheckInterval time.Duration `yaml:"checkInterval,omitempty" json:"checkInterval,omitempty"`
	// ReductionFactor is how much to reduce QPS when in reduce mode (default: 0.5).
	ReductionFactor float64 `yaml:"reductionFactor,omitempty" json:"reductionFactor,omitempty"`
	// DropPercentage is what percentage of requests to drop when in drop mode (default: 0.5 = 50%).
	DropPercentage float64 `yaml:"dropPercentage,omitempty" json:"dropPercentage,omitempty"`
	// CircuitOpenDuration is how long to keep the circuit open (default: 10s).
	CircuitOpenDuration time.Duration `yaml:"circuitOpenDuration,omitempty" json:"circuitOpenDuration,omitempty"`
	// ConsecutiveBreachThreshold is how many consecutive breaches before escalating (default: 3).
	ConsecutiveBreachThreshold int `yaml:"consecutiveBreachThreshold,omitempty" json:"consecutiveBreachThreshold,omitempty"`
}

// DefaultBackpressureConfig returns a BackpressureConfig with default values.
func DefaultBackpressureConfig() BackpressureConfig {
	return BackpressureConfig{
		Strategy:                   BackpressureStrategyReduce,
		ErrorRateThreshold:         0.1, // 10%
		LatencyP99Threshold:        time.Second,
		WarningErrorThreshold:      0.05, // 5%
		WarningLatencyThreshold:    500 * time.Millisecond,
		RecoveryPeriod:             30 * time.Second,
		CheckInterval:              100 * time.Millisecond,
		ReductionFactor:            0.5,
		DropPercentage:             0.5,
		CircuitOpenDuration:        10 * time.Second,
		ConsecutiveBreachThreshold: 3,
	}
}

// DefaultBackpressureHandler implements the BackpressureHandler interface.
type DefaultBackpressureHandler struct {
	metrics MetricsCollector
	config  BackpressureConfig

	// State management (protected by stateMu)
	stateMu             sync.RWMutex
	currentState        BackpressureState
	stateEntryTime      time.Time
	recoveryStartTime   time.Time
	consecutiveBreaches int

	// Circuit breaker state (protected by circuitMu)
	circuitMu       sync.RWMutex
	circuitOpen     bool
	circuitOpenTime time.Time
	halfOpenAllowed int32
	halfOpenReset   time.Time // When half-open state started for resetting probes

	// Statistics
	totalTransitions atomic.Int64
	totalDropped     atomic.Int64
	totalPaused      atomic.Int64
	totalReduced     atomic.Int64
	timeInNormal     atomic.Int64 // nanoseconds
	timeInWarning    atomic.Int64
	timeInCritical   atomic.Int64
	timeInRecovery   atomic.Int64
	lastErrorRate    atomic.Uint64 // float64 bits
	lastP99Latency   atomic.Int64  // nanoseconds

	// Drop counter for probabilistic dropping
	dropCounter atomic.Uint64

	// Control (protected by controlMu)
	controlMu sync.Mutex
	stopCh    chan struct{}
	wg        sync.WaitGroup
	isRunning atomic.Bool
}

// NewBackpressureHandler creates a new backpressure handler.
func NewBackpressureHandler(metrics MetricsCollector, config BackpressureConfig) *DefaultBackpressureHandler {
	// Apply defaults
	if config.ErrorRateThreshold <= 0 {
		config.ErrorRateThreshold = 0.1
	}
	if config.LatencyP99Threshold <= 0 {
		config.LatencyP99Threshold = time.Second
	}
	if config.WarningErrorThreshold <= 0 {
		config.WarningErrorThreshold = config.ErrorRateThreshold / 2
	}
	if config.WarningLatencyThreshold <= 0 {
		config.WarningLatencyThreshold = config.LatencyP99Threshold / 2
	}
	if config.RecoveryPeriod <= 0 {
		config.RecoveryPeriod = 30 * time.Second
	}
	if config.CheckInterval <= 0 {
		config.CheckInterval = 100 * time.Millisecond
	}
	if config.ReductionFactor <= 0 || config.ReductionFactor >= 1 {
		config.ReductionFactor = 0.5
	}
	if config.DropPercentage <= 0 || config.DropPercentage > 1 {
		config.DropPercentage = 0.5
	}
	if config.CircuitOpenDuration <= 0 {
		config.CircuitOpenDuration = 10 * time.Second
	}
	if config.ConsecutiveBreachThreshold <= 0 {
		config.ConsecutiveBreachThreshold = 3
	}
	if config.Strategy == "" {
		config.Strategy = BackpressureStrategyReduce
	}

	h := &DefaultBackpressureHandler{
		metrics:        metrics,
		config:         config,
		currentState:   BackpressureStateNormal,
		stateEntryTime: time.Now(),
		stopCh:         make(chan struct{}),
	}

	return h
}

// Check evaluates current metrics and returns an action to take.
func (h *DefaultBackpressureHandler) Check() BackpressureAction {
	// Get current metrics
	var errorRate float64
	var p99Latency time.Duration

	if h.metrics != nil {
		errorRate = h.metrics.GetErrorRate()
		p99Latency = h.metrics.GetP99Latency()
	}

	// Store for stats
	h.storeMetrics(errorRate, p99Latency)

	// Update state based on metrics
	h.updateState(errorRate, p99Latency)

	// Get current state
	h.stateMu.RLock()
	state := h.currentState
	h.stateMu.RUnlock()

	// Determine action based on state and strategy
	return h.determineAction(state, errorRate, p99Latency)
}

// ShouldAllow returns true if the request should be allowed based on current state.
func (h *DefaultBackpressureHandler) ShouldAllow() bool {
	h.stateMu.RLock()
	state := h.currentState
	strategy := h.config.Strategy
	h.stateMu.RUnlock()

	switch state {
	case BackpressureStateNormal, BackpressureStateWarning:
		return true

	case BackpressureStateRecovery:
		// In recovery, allow requests but monitor closely
		return true

	case BackpressureStateCritical:
		// In critical state, behavior depends on strategy
		switch strategy {
		case BackpressureStrategyPause:
			return false

		case BackpressureStrategyCircuit:
			return h.checkCircuit()

		case BackpressureStrategyDrop:
			// Probabilistic dropping
			return !h.shouldDrop()

		case BackpressureStrategyReduce:
			// Always allow, but caller should reduce QPS
			return true

		default:
			return true
		}
	}

	return true
}

// State returns the current backpressure state.
func (h *DefaultBackpressureHandler) State() BackpressureState {
	h.stateMu.RLock()
	defer h.stateMu.RUnlock()
	return h.currentState
}

// Stats returns statistics about backpressure handling.
func (h *DefaultBackpressureHandler) Stats() BackpressureStats {
	h.stateMu.RLock()
	state := h.currentState
	recoveryStart := h.recoveryStartTime
	h.stateMu.RUnlock()

	// Convert atomic float64
	errorRateBits := h.lastErrorRate.Load()
	errorRate := float64FromBits(errorRateBits)

	return BackpressureStats{
		CurrentState:          state,
		TotalStateTransitions: h.totalTransitions.Load(),
		TotalDropped:          h.totalDropped.Load(),
		TotalPaused:           h.totalPaused.Load(),
		TotalReduced:          h.totalReduced.Load(),
		TimeInNormal:          time.Duration(h.timeInNormal.Load()),
		TimeInWarning:         time.Duration(h.timeInWarning.Load()),
		TimeInCritical:        time.Duration(h.timeInCritical.Load()),
		TimeInRecovery:        time.Duration(h.timeInRecovery.Load()),
		LastErrorRate:         errorRate,
		LastP99Latency:        time.Duration(h.lastP99Latency.Load()),
		RecoveryStartTime:     recoveryStart,
	}
}

// SetStrategy sets the backpressure strategy.
func (h *DefaultBackpressureHandler) SetStrategy(strategy BackpressureStrategy) {
	h.stateMu.Lock()
	defer h.stateMu.Unlock()
	h.config.Strategy = strategy
}

// SetErrorThreshold sets the error rate threshold (0.0 - 1.0).
func (h *DefaultBackpressureHandler) SetErrorThreshold(threshold float64) {
	if threshold <= 0 || threshold > 1 {
		return
	}
	h.stateMu.Lock()
	defer h.stateMu.Unlock()
	h.config.ErrorRateThreshold = threshold
	h.config.WarningErrorThreshold = threshold / 2
}

// SetLatencyThreshold sets the P99 latency threshold.
func (h *DefaultBackpressureHandler) SetLatencyThreshold(threshold time.Duration) {
	if threshold <= 0 {
		return
	}
	h.stateMu.Lock()
	defer h.stateMu.Unlock()
	h.config.LatencyP99Threshold = threshold
	h.config.WarningLatencyThreshold = threshold / 2
}

// Reset resets the handler to normal state.
func (h *DefaultBackpressureHandler) Reset() {
	h.stateMu.Lock()
	h.currentState = BackpressureStateNormal
	h.stateEntryTime = time.Now()
	h.recoveryStartTime = time.Time{}
	h.consecutiveBreaches = 0
	h.stateMu.Unlock()

	h.circuitMu.Lock()
	h.circuitOpen = false
	h.circuitOpenTime = time.Time{}
	h.halfOpenAllowed = 0
	h.circuitMu.Unlock()

	h.totalTransitions.Store(0)
	h.totalDropped.Store(0)
	h.totalPaused.Store(0)
	h.totalReduced.Store(0)
	h.timeInNormal.Store(0)
	h.timeInWarning.Store(0)
	h.timeInCritical.Store(0)
	h.timeInRecovery.Store(0)
	h.lastErrorRate.Store(0)
	h.lastP99Latency.Store(0)
	h.dropCounter.Store(0)
}

// Start starts the background monitoring loop.
func (h *DefaultBackpressureHandler) Start(ctx context.Context) {
	h.controlMu.Lock()
	defer h.controlMu.Unlock()

	if h.isRunning.Load() {
		return // Already running
	}

	// Reset the stop channel (ensure previous goroutine has fully stopped)
	h.stopCh = make(chan struct{})
	h.isRunning.Store(true)
	h.wg.Add(1)
	go h.runMonitorLoop(ctx)
}

// Stop stops the background monitoring loop.
func (h *DefaultBackpressureHandler) Stop() {
	h.controlMu.Lock()
	if !h.isRunning.Load() {
		h.controlMu.Unlock()
		return // Not running
	}

	h.isRunning.Store(false)
	close(h.stopCh)
	// Must wait inside lock to prevent Start from adding to WaitGroup
	// before previous Wait() completes
	h.wg.Wait()
	h.controlMu.Unlock()
}

// runMonitorLoop runs the background monitoring loop.
func (h *DefaultBackpressureHandler) runMonitorLoop(ctx context.Context) {
	defer h.wg.Done()

	ticker := time.NewTicker(h.config.CheckInterval)
	defer ticker.Stop()

	lastCheck := time.Now()

	for {
		select {
		case <-ctx.Done():
			h.updateTimeInState(time.Since(lastCheck))
			return
		case <-h.stopCh:
			h.updateTimeInState(time.Since(lastCheck))
			return
		case <-ticker.C:
			elapsed := time.Since(lastCheck)
			h.updateTimeInState(elapsed)
			lastCheck = time.Now()

			h.Check()
		}
	}
}

// updateState updates the state machine based on current metrics.
func (h *DefaultBackpressureHandler) updateState(errorRate float64, p99Latency time.Duration) {
	h.stateMu.Lock()
	defer h.stateMu.Unlock()

	oldState := h.currentState
	newState := h.evaluateState(errorRate, p99Latency)

	if newState != oldState {
		// Update time tracking for old state
		elapsed := time.Since(h.stateEntryTime)
		h.addTimeToState(oldState, elapsed)

		// Transition to new state
		h.currentState = newState
		h.stateEntryTime = time.Now()
		h.totalTransitions.Add(1)

		// Handle state entry
		h.onStateEntry(newState)
	}
}

// evaluateState determines what state we should be in based on metrics.
func (h *DefaultBackpressureHandler) evaluateState(errorRate float64, p99Latency time.Duration) BackpressureState {
	isCritical := errorRate >= h.config.ErrorRateThreshold ||
		(h.config.LatencyP99Threshold > 0 && p99Latency >= h.config.LatencyP99Threshold)

	isWarning := errorRate >= h.config.WarningErrorThreshold ||
		(h.config.WarningLatencyThreshold > 0 && p99Latency >= h.config.WarningLatencyThreshold)

	switch h.currentState {
	case BackpressureStateNormal:
		if isCritical {
			h.consecutiveBreaches++
			if h.consecutiveBreaches >= h.config.ConsecutiveBreachThreshold {
				return BackpressureStateCritical
			}
			return BackpressureStateWarning
		}
		if isWarning {
			h.consecutiveBreaches++
			if h.consecutiveBreaches >= h.config.ConsecutiveBreachThreshold {
				return BackpressureStateWarning
			}
		} else {
			h.consecutiveBreaches = 0
		}
		return BackpressureStateNormal

	case BackpressureStateWarning:
		if isCritical {
			h.consecutiveBreaches++
			if h.consecutiveBreaches >= h.config.ConsecutiveBreachThreshold {
				return BackpressureStateCritical
			}
		} else if !isWarning {
			h.consecutiveBreaches = 0
			return BackpressureStateNormal
		}
		return BackpressureStateWarning

	case BackpressureStateCritical:
		if !isCritical && !isWarning {
			// Start recovery
			h.consecutiveBreaches = 0
			return BackpressureStateRecovery
		}
		return BackpressureStateCritical

	case BackpressureStateRecovery:
		if isCritical {
			// Back to critical
			h.consecutiveBreaches = h.config.ConsecutiveBreachThreshold
			return BackpressureStateCritical
		}
		if isWarning {
			// Stay in recovery but don't reset timer
			return BackpressureStateRecovery
		}
		// Check if recovery period has passed
		if time.Since(h.recoveryStartTime) >= h.config.RecoveryPeriod {
			return BackpressureStateNormal
		}
		return BackpressureStateRecovery
	}

	return BackpressureStateNormal
}

// onStateEntry handles actions when entering a new state.
func (h *DefaultBackpressureHandler) onStateEntry(state BackpressureState) {
	switch state {
	case BackpressureStateRecovery:
		h.recoveryStartTime = time.Now()
		// Close circuit if using circuit strategy
		h.circuitMu.Lock()
		h.circuitOpen = false
		h.circuitMu.Unlock()

	case BackpressureStateCritical:
		// Open circuit if using circuit strategy
		if h.config.Strategy == BackpressureStrategyCircuit {
			h.circuitMu.Lock()
			h.circuitOpen = true
			h.circuitOpenTime = time.Now()
			h.halfOpenAllowed = 0
			h.circuitMu.Unlock()
		}

	case BackpressureStateNormal:
		h.recoveryStartTime = time.Time{}
		h.circuitMu.Lock()
		h.circuitOpen = false
		h.circuitMu.Unlock()
	}
}

// determineAction determines the action to take based on current state and strategy.
func (h *DefaultBackpressureHandler) determineAction(state BackpressureState, _, _ any) BackpressureAction {
	action := BackpressureAction{
		State:         state,
		QPSMultiplier: 1.0,
	}

	switch state {
	case BackpressureStateNormal:
		action.Reason = "operating normally"
		return action

	case BackpressureStateWarning:
		action.Reason = "warning: elevated error rate or latency"
		// Slight reduction in warning state
		action.QPSMultiplier = 0.9
		return action

	case BackpressureStateRecovery:
		action.Reason = "recovering: gradually restoring traffic"
		// Gradual recovery - start at reduced rate
		h.stateMu.RLock()
		elapsed := time.Since(h.recoveryStartTime)
		h.stateMu.RUnlock()

		// Linear recovery from 50% to 100% over recovery period
		progress := float64(elapsed) / float64(h.config.RecoveryPeriod)
		if progress > 1.0 {
			progress = 1.0
		}
		action.QPSMultiplier = 0.5 + (0.5 * progress)
		return action

	case BackpressureStateCritical:
		return h.determineCriticalAction()
	}

	return action
}

// determineCriticalAction determines the action to take in critical state.
func (h *DefaultBackpressureHandler) determineCriticalAction() BackpressureAction {
	// Read config under lock to avoid race
	h.stateMu.RLock()
	strategy := h.config.Strategy
	reductionFactor := h.config.ReductionFactor
	circuitOpenDuration := h.config.CircuitOpenDuration
	h.stateMu.RUnlock()

	action := BackpressureAction{
		State:         BackpressureStateCritical,
		QPSMultiplier: 1.0,
	}

	switch strategy {
	case BackpressureStrategyDrop:
		action.ShouldDrop = h.shouldDrop()
		if action.ShouldDrop {
			h.totalDropped.Add(1)
		}
		action.Reason = "critical: dropping requests"

	case BackpressureStrategyReduce:
		action.QPSMultiplier = reductionFactor
		h.totalReduced.Add(1)
		action.Reason = "critical: reducing QPS"

	case BackpressureStrategyPause:
		action.ShouldPause = true
		action.PauseDuration = circuitOpenDuration
		h.totalPaused.Add(1)
		action.Reason = "critical: pausing requests"

	case BackpressureStrategyCircuit:
		if !h.checkCircuit() {
			action.ShouldDrop = true
			h.totalDropped.Add(1)
			action.Reason = "critical: circuit open"
		} else {
			action.Reason = "critical: circuit half-open, allowing probe"
		}
	}

	return action
}

// shouldDrop returns true if the current request should be dropped.
func (h *DefaultBackpressureHandler) shouldDrop() bool {
	// Read config under lock to avoid race
	h.stateMu.RLock()
	dropPercentage := h.config.DropPercentage
	h.stateMu.RUnlock()

	// Use counter-based probabilistic dropping
	counter := h.dropCounter.Add(1)
	// Drop based on percentage (e.g., 50% means drop every other request)
	dropInterval := max(1, uint64(1.0/dropPercentage))
	return counter%dropInterval == 0
}

// checkCircuit checks if a request should be allowed through the circuit breaker.
// It implements a proper circuit breaker pattern with half-open state resetting.
func (h *DefaultBackpressureHandler) checkCircuit() bool {
	// Read config under lock to avoid race
	h.stateMu.RLock()
	circuitOpenDuration := h.config.CircuitOpenDuration
	h.stateMu.RUnlock()

	h.circuitMu.Lock()
	defer h.circuitMu.Unlock()

	if !h.circuitOpen {
		return true
	}

	// Check if circuit should transition to half-open
	timeSinceOpen := time.Since(h.circuitOpenTime)
	if timeSinceOpen >= circuitOpenDuration {
		// Check if we need to reset the half-open probe counter
		// Reset after another circuit open duration to allow new probes
		if h.halfOpenReset.IsZero() {
			h.halfOpenReset = time.Now()
		} else if time.Since(h.halfOpenReset) >= circuitOpenDuration {
			// Reset probe counter for new round of probes
			h.halfOpenAllowed = 0
			h.halfOpenReset = time.Now()
		}

		// Half-open: allow limited requests to probe
		if h.halfOpenAllowed < 5 { // Allow 5 probe requests per cycle
			h.halfOpenAllowed++
			return true
		}
	}

	return false
}

// storeMetrics stores the current metrics for statistics.
func (h *DefaultBackpressureHandler) storeMetrics(errorRate float64, p99Latency time.Duration) {
	h.lastErrorRate.Store(float64ToBits(errorRate))
	h.lastP99Latency.Store(int64(p99Latency))
}

// updateTimeInState updates time tracking for the current state.
func (h *DefaultBackpressureHandler) updateTimeInState(elapsed time.Duration) {
	h.stateMu.RLock()
	state := h.currentState
	h.stateMu.RUnlock()

	h.addTimeToState(state, elapsed)
}

// addTimeToState adds time to the appropriate state counter.
func (h *DefaultBackpressureHandler) addTimeToState(state BackpressureState, elapsed time.Duration) {
	switch state {
	case BackpressureStateNormal:
		h.timeInNormal.Add(int64(elapsed))
	case BackpressureStateWarning:
		h.timeInWarning.Add(int64(elapsed))
	case BackpressureStateCritical:
		h.timeInCritical.Add(int64(elapsed))
	case BackpressureStateRecovery:
		h.timeInRecovery.Add(int64(elapsed))
	}
}

// float64ToBits converts a float64 to its bit representation.
func float64ToBits(f float64) uint64 {
	return math.Float64bits(f)
}

// float64FromBits converts bits back to float64.
func float64FromBits(bits uint64) float64 {
	return math.Float64frombits(bits)
}

// Compile-time interface check
var _ BackpressureHandler = (*DefaultBackpressureHandler)(nil)
