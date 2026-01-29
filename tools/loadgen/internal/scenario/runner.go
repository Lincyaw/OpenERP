// Package scenario provides scenario management for the load generator.
package scenario

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/example/erp/tools/loadgen/internal/circuit"
	"github.com/example/erp/tools/loadgen/internal/config"
	"github.com/example/erp/tools/loadgen/internal/workflow"
)

// Runner manages the execution of a scenario against a configuration.
// It applies scenario overrides to the base configuration and tracks
// execution state.
type Runner struct {
	mu         sync.RWMutex
	scenario   *Definition
	baseConfig *config.Config
	appliedCfg *config.Config
	startTime  time.Time
	endTime    time.Time
	running    bool
	completed  bool
	cancelFunc context.CancelFunc
	stats      *RunStats
}

// RunStats contains statistics about a scenario run.
type RunStats struct {
	ScenarioName      string        `json:"scenarioName"`
	StartTime         time.Time     `json:"startTime"`
	EndTime           time.Time     `json:"endTime,omitempty"`
	Duration          time.Duration `json:"duration,omitempty"`
	EffectiveDuration time.Duration `json:"effectiveDuration,omitempty"`
	EndpointsActive   int           `json:"endpointsActive"`
	EndpointsDisabled int           `json:"endpointsDisabled"`
	OverridesApplied  []string      `json:"overridesApplied"`
}

// NewRunner creates a new scenario runner.
func NewRunner(scenario *Definition, baseConfig *config.Config) *Runner {
	return &Runner{
		scenario:   scenario,
		baseConfig: baseConfig,
		stats: &RunStats{
			ScenarioName:     scenario.Name,
			OverridesApplied: make([]string, 0),
		},
	}
}

// Scenario returns the scenario being run.
func (r *Runner) Scenario() *Definition {
	return r.scenario
}

// BaseConfig returns the base configuration before overrides.
func (r *Runner) BaseConfig() *config.Config {
	return r.baseConfig
}

// AppliedConfig returns the configuration with scenario overrides applied.
// This must be called after ApplyOverrides().
func (r *Runner) AppliedConfig() *config.Config {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.appliedCfg
}

// Stats returns the run statistics.
func (r *Runner) Stats() *RunStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Make a copy
	stats := *r.stats
	if r.running && !r.completed {
		stats.Duration = time.Since(r.startTime)
	} else if r.completed {
		stats.Duration = r.endTime.Sub(r.startTime)
	}
	return &stats
}

// IsRunning returns true if the scenario is currently running.
func (r *Runner) IsRunning() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.running
}

// IsCompleted returns true if the scenario has completed.
func (r *Runner) IsCompleted() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.completed
}

// ApplyOverrides applies scenario overrides to the base configuration
// and returns the modified configuration. The original base config is not modified.
func (r *Runner) ApplyOverrides() (*config.Config, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Deep copy the base config to avoid modifying it
	applied, err := deepCopyConfig(r.baseConfig)
	if err != nil {
		return nil, fmt.Errorf("copying base config: %w", err)
	}

	// Apply duration override
	if r.scenario.HasDurationOverride() {
		applied.Duration = r.scenario.Duration
		r.stats.OverridesApplied = append(r.stats.OverridesApplied, "duration")
	}

	// Apply traffic shaper override
	if r.scenario.HasTrafficOverride() {
		applied.TrafficShaper = *r.scenario.TrafficShaper
		r.stats.OverridesApplied = append(r.stats.OverridesApplied, "trafficShaper")
	}

	// Apply endpoint focus
	if r.scenario.HasFocusEndpoints() {
		r.stats.OverridesApplied = append(r.stats.OverridesApplied, "focusEndpoints")
	}

	// Apply endpoint overrides
	activeCount := 0
	disabledCount := 0

	for i := range applied.Endpoints {
		ep := &applied.Endpoints[i]

		// Check if disabled by scenario
		if r.scenario.IsEndpointDisabled(ep.Name) {
			ep.Disabled = true
			disabledCount++
			continue
		}

		// Check if focused (if focus list is specified)
		if !r.scenario.IsEndpointFocused(ep.Name) {
			ep.Disabled = true
			disabledCount++
			continue
		}

		// Apply weight override if specified
		if weight := r.scenario.GetEndpointWeight(ep.Name); weight >= 0 {
			ep.Weight = weight
		}

		if !ep.Disabled {
			activeCount++
		}
	}

	// Update stats
	r.stats.EndpointsActive = activeCount
	r.stats.EndpointsDisabled = disabledCount

	// Apply warmup override
	if r.scenario.Warmup != nil {
		if r.scenario.Warmup.Enabled != nil && !*r.scenario.Warmup.Enabled {
			applied.Warmup.Iterations = 0
		}
		if r.scenario.Warmup.Iterations > 0 {
			applied.Warmup.Iterations = r.scenario.Warmup.Iterations
		}
		if r.scenario.Warmup.Timeout > 0 {
			applied.Warmup.Timeout = r.scenario.Warmup.Timeout
		}
		r.stats.OverridesApplied = append(r.stats.OverridesApplied, "warmup")
	}

	// Apply assertion override
	if r.scenario.Assertions != nil {
		if applied.Assertions.Global == nil {
			applied.Assertions.Global = &config.GlobalAssertions{}
		}
		if r.scenario.Assertions.MaxErrorRate != nil {
			applied.Assertions.Global.MaxErrorRate = r.scenario.Assertions.MaxErrorRate
		}
		if r.scenario.Assertions.MinSuccessRate != nil {
			applied.Assertions.Global.MinSuccessRate = r.scenario.Assertions.MinSuccessRate
		}
		if r.scenario.Assertions.MaxP95Latency > 0 {
			applied.Assertions.Global.MaxP95Latency = r.scenario.Assertions.MaxP95Latency
		}
		if r.scenario.Assertions.MaxP99Latency > 0 {
			applied.Assertions.Global.MaxP99Latency = r.scenario.Assertions.MaxP99Latency
		}
		r.stats.OverridesApplied = append(r.stats.OverridesApplied, "assertions")
	}

	// Update config name to include scenario
	applied.Name = fmt.Sprintf("%s [Scenario: %s]", applied.Name, r.scenario.Name)

	r.appliedCfg = applied
	return applied, nil
}

// Start marks the scenario as started. This is typically called
// right before the load test begins executing.
func (r *Runner) Start(ctx context.Context) (context.Context, context.CancelFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.startTime = time.Now()
	r.stats.StartTime = r.startTime
	r.running = true
	r.completed = false

	ctx, cancel := context.WithCancel(ctx)
	r.cancelFunc = cancel
	return ctx, cancel
}

// Complete marks the scenario as completed.
func (r *Runner) Complete() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.endTime = time.Now()
	r.stats.EndTime = r.endTime
	r.stats.Duration = r.endTime.Sub(r.startTime)
	r.stats.EffectiveDuration = r.stats.Duration
	r.running = false
	r.completed = true
}

// Cancel cancels the running scenario.
func (r *Runner) Cancel() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cancelFunc != nil {
		r.cancelFunc()
	}
	r.running = false
}

// deepCopyConfig creates a deep copy of a config.
func deepCopyConfig(src *config.Config) (*config.Config, error) {
	// Create a new config and copy fields
	dst := &config.Config{
		Name:            src.Name,
		Description:     src.Description,
		Version:         src.Version,
		Target:          src.Target,
		Auth:            src.Auth,
		Duration:        src.Duration,
		Warmup:          src.Warmup,
		TrafficShaper:   src.TrafficShaper,
		RateLimiter:     src.RateLimiter,
		WorkerPool:      src.WorkerPool,
		Controller:      src.Controller,
		Backpressure:    src.Backpressure,
		Metrics:         src.Metrics,
		Output:          src.Output,
		Assertions:      src.Assertions,
		InferenceConfig: src.InferenceConfig,
	}

	// Deep copy target headers
	if src.Target.Headers != nil {
		dst.Target.Headers = make(map[string]string, len(src.Target.Headers))
		for k, v := range src.Target.Headers {
			dst.Target.Headers[k] = v
		}
	}

	// Deep copy endpoints
	dst.Endpoints = make([]config.EndpointConfig, len(src.Endpoints))
	for i, ep := range src.Endpoints {
		dst.Endpoints[i] = copyEndpoint(ep)
	}

	// Deep copy scenarios
	dst.Scenarios = make([]config.ScenarioConfig, len(src.Scenarios))
	copy(dst.Scenarios, src.Scenarios)

	// Deep copy data generators
	if src.DataGenerators != nil {
		dst.DataGenerators = make(map[string]config.GeneratorConfig, len(src.DataGenerators))
		for k, v := range src.DataGenerators {
			dst.DataGenerators[k] = v
		}
	}

	// Deep copy semantic overrides
	if src.SemanticOverrides != nil {
		dst.SemanticOverrides = make(map[string]string, len(src.SemanticOverrides))
		for k, v := range src.SemanticOverrides {
			dst.SemanticOverrides[k] = v
		}
	}

	// Deep copy workflows
	if src.Workflows != nil {
		dst.Workflows = make(map[string]workflow.Definition, len(src.Workflows))
		for k, v := range src.Workflows {
			dst.Workflows[k] = v
		}
	}

	// Deep copy assertions
	if src.Assertions.Global != nil {
		globalCopy := *src.Assertions.Global
		dst.Assertions.Global = &globalCopy
	}
	if src.Assertions.EndpointOverrides != nil {
		dst.Assertions.EndpointOverrides = make(map[string]config.EndpointAssertions, len(src.Assertions.EndpointOverrides))
		for k, v := range src.Assertions.EndpointOverrides {
			dst.Assertions.EndpointOverrides[k] = v
		}
	}

	return dst, nil
}

// copyEndpoint creates a copy of an endpoint config.
func copyEndpoint(src config.EndpointConfig) config.EndpointConfig {
	dst := src // Shallow copy first

	// Deep copy slices and maps
	if src.Tags != nil {
		dst.Tags = make([]string, len(src.Tags))
		copy(dst.Tags, src.Tags)
	}

	if src.Headers != nil {
		dst.Headers = make(map[string]string, len(src.Headers))
		for k, v := range src.Headers {
			dst.Headers[k] = v
		}
	}

	if src.QueryParams != nil {
		dst.QueryParams = make(map[string]config.ParameterConfig, len(src.QueryParams))
		for k, v := range src.QueryParams {
			dst.QueryParams[k] = v
		}
	}

	if src.PathParams != nil {
		dst.PathParams = make(map[string]config.ParameterConfig, len(src.PathParams))
		for k, v := range src.PathParams {
			dst.PathParams[k] = v
		}
	}

	if src.Produces != nil {
		dst.Produces = make([]config.ProducesConfig, len(src.Produces))
		copy(dst.Produces, src.Produces)
	}

	if src.Consumes != nil {
		dst.Consumes = make([]circuit.SemanticType, len(src.Consumes))
		copy(dst.Consumes, src.Consumes)
	}

	if src.DependsOn != nil {
		dst.DependsOn = make([]string, len(src.DependsOn))
		copy(dst.DependsOn, src.DependsOn)
	}

	if src.Schedule != nil {
		dst.Schedule = make([]config.ScheduleWeightConfig, len(src.Schedule))
		copy(dst.Schedule, src.Schedule)
	}

	return dst
}
