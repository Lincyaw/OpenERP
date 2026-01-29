// Package metrics provides metrics collection and reporting for the load generator.
package metrics

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Exit codes for assertion results.
const (
	// ExitCodeSuccess indicates all assertions passed.
	ExitCodeSuccess = 0
	// ExitCodeAssertionFailure indicates one or more assertions failed.
	ExitCodeAssertionFailure = 2
)

// Errors returned by the assertions package.
var (
	// ErrAssertionFailed is returned when one or more assertions fail.
	ErrAssertionFailed = errors.New("assertion failed")
)

// AssertionResult represents the result of evaluating a single assertion.
type AssertionResult struct {
	// Name is the assertion name (e.g., "global.maxErrorRate", "endpoint:create-product.maxP95Latency").
	Name string

	// Description describes what was being checked.
	Description string

	// Passed indicates whether the assertion passed.
	Passed bool

	// Expected is the expected/threshold value.
	Expected string

	// Actual is the actual measured value.
	Actual string

	// Endpoint is the endpoint name (empty for global assertions).
	Endpoint string
}

// AssertionResults holds all assertion results from a validation run.
type AssertionResults struct {
	// Results is the list of all assertion results.
	Results []AssertionResult

	// PassedCount is the number of passed assertions.
	PassedCount int

	// FailedCount is the number of failed assertions.
	FailedCount int

	// TotalCount is the total number of assertions evaluated.
	TotalCount int

	// AllPassed indicates whether all assertions passed.
	AllPassed bool
}

// FailedResults returns only the failed assertion results.
func (r *AssertionResults) FailedResults() []AssertionResult {
	failed := make([]AssertionResult, 0, r.FailedCount)
	for _, result := range r.Results {
		if !result.Passed {
			failed = append(failed, result)
		}
	}
	return failed
}

// PassedResults returns only the passed assertion results.
func (r *AssertionResults) PassedResults() []AssertionResult {
	passed := make([]AssertionResult, 0, r.PassedCount)
	for _, result := range r.Results {
		if result.Passed {
			passed = append(passed, result)
		}
	}
	return passed
}

// Summary returns a human-readable summary of the assertion results.
func (r *AssertionResults) Summary() string {
	if r.TotalCount == 0 {
		return "No assertions configured"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Assertions: %d/%d passed", r.PassedCount, r.TotalCount))

	if r.FailedCount > 0 {
		sb.WriteString(fmt.Sprintf(" (%d FAILED)", r.FailedCount))
	}

	return sb.String()
}

// AssertionValidatorConfig is a simplified interface for assertion configuration
// to avoid circular imports with the config package.
type AssertionValidatorConfig struct {
	// Global assertions apply to overall test results.
	Global *GlobalAssertions

	// EndpointOverrides allows endpoint-specific assertion overrides.
	EndpointOverrides map[string]EndpointAssertions

	// ExitOnFailure causes the program to return a non-zero exit code
	// if any assertion fails. Default: true
	ExitOnFailure *bool
}

// GlobalAssertions defines SLO thresholds for the entire test.
// This type mirrors config.GlobalAssertions without YAML tags for internal use.
type GlobalAssertions struct {
	MaxErrorRate   *float64
	MinSuccessRate *float64
	MaxP50Latency  int64 // Nanoseconds
	MaxP95Latency  int64 // Nanoseconds
	MaxP99Latency  int64 // Nanoseconds
	MaxAvgLatency  int64 // Nanoseconds
	MinThroughput  *float64
}

// EndpointAssertions defines SLO thresholds for a specific endpoint.
type EndpointAssertions struct {
	MaxErrorRate   *float64
	MinSuccessRate *float64
	MaxP50Latency  int64 // Nanoseconds
	MaxP95Latency  int64 // Nanoseconds
	MaxP99Latency  int64 // Nanoseconds
	MaxAvgLatency  int64 // Nanoseconds
	MinThroughput  *float64
	Disabled       bool
}

// AssertionValidator validates metrics against SLO assertions.
type AssertionValidator struct {
	config AssertionValidatorConfig
}

// NewAssertionValidator creates a new assertion validator.
func NewAssertionValidator(config AssertionValidatorConfig) *AssertionValidator {
	return &AssertionValidator{
		config: config,
	}
}

// DefaultAssertionValidatorConfig returns a configuration with default values.
func DefaultAssertionValidatorConfig() AssertionValidatorConfig {
	exitOnFailure := true
	return AssertionValidatorConfig{
		ExitOnFailure: &exitOnFailure,
	}
}

// ExitOnFailure returns whether the program should exit with non-zero code on failure.
func (v *AssertionValidator) ExitOnFailure() bool {
	if v.config.ExitOnFailure == nil {
		return true // Default to true
	}
	return *v.config.ExitOnFailure
}

// Validate evaluates all assertions against the provided metrics snapshot.
func (v *AssertionValidator) Validate(snapshot Snapshot) *AssertionResults {
	results := &AssertionResults{
		Results: make([]AssertionResult, 0),
	}

	// Validate global assertions
	v.validateGlobalAssertions(snapshot, results)

	// Validate endpoint-specific assertions
	v.validateEndpointAssertions(snapshot, results)

	// Calculate totals
	results.TotalCount = len(results.Results)
	for _, r := range results.Results {
		if r.Passed {
			results.PassedCount++
		} else {
			results.FailedCount++
		}
	}
	results.AllPassed = results.FailedCount == 0

	return results
}

// validateGlobalAssertions validates global SLO assertions.
func (v *AssertionValidator) validateGlobalAssertions(snapshot Snapshot, results *AssertionResults) {
	global := v.config.Global
	if global == nil {
		return
	}

	// MaxErrorRate
	if global.MaxErrorRate != nil {
		errorRate := 100.0 - snapshot.SuccessRate // SuccessRate is 0-100, so errorRate = 100 - success
		passed := errorRate <= *global.MaxErrorRate
		results.Results = append(results.Results, AssertionResult{
			Name:        "global.maxErrorRate",
			Description: "Maximum error rate",
			Passed:      passed,
			Expected:    fmt.Sprintf("<= %.2f%%", *global.MaxErrorRate),
			Actual:      fmt.Sprintf("%.2f%%", errorRate),
		})
	}

	// MinSuccessRate
	if global.MinSuccessRate != nil {
		passed := snapshot.SuccessRate >= *global.MinSuccessRate
		results.Results = append(results.Results, AssertionResult{
			Name:        "global.minSuccessRate",
			Description: "Minimum success rate",
			Passed:      passed,
			Expected:    fmt.Sprintf(">= %.2f%%", *global.MinSuccessRate),
			Actual:      fmt.Sprintf("%.2f%%", snapshot.SuccessRate),
		})
	}

	// MaxP50Latency
	if global.MaxP50Latency > 0 {
		passed := snapshot.P50Latency.Nanoseconds() <= global.MaxP50Latency
		results.Results = append(results.Results, AssertionResult{
			Name:        "global.maxP50Latency",
			Description: "Maximum P50 latency",
			Passed:      passed,
			Expected:    fmt.Sprintf("<= %s", formatDurationNs(global.MaxP50Latency)),
			Actual:      snapshot.P50Latency.String(),
		})
	}

	// MaxP95Latency
	if global.MaxP95Latency > 0 {
		passed := snapshot.P95Latency.Nanoseconds() <= global.MaxP95Latency
		results.Results = append(results.Results, AssertionResult{
			Name:        "global.maxP95Latency",
			Description: "Maximum P95 latency",
			Passed:      passed,
			Expected:    fmt.Sprintf("<= %s", formatDurationNs(global.MaxP95Latency)),
			Actual:      snapshot.P95Latency.String(),
		})
	}

	// MaxP99Latency
	if global.MaxP99Latency > 0 {
		passed := snapshot.P99Latency.Nanoseconds() <= global.MaxP99Latency
		results.Results = append(results.Results, AssertionResult{
			Name:        "global.maxP99Latency",
			Description: "Maximum P99 latency",
			Passed:      passed,
			Expected:    fmt.Sprintf("<= %s", formatDurationNs(global.MaxP99Latency)),
			Actual:      snapshot.P99Latency.String(),
		})
	}

	// MaxAvgLatency
	if global.MaxAvgLatency > 0 {
		passed := snapshot.AvgLatency.Nanoseconds() <= global.MaxAvgLatency
		results.Results = append(results.Results, AssertionResult{
			Name:        "global.maxAvgLatency",
			Description: "Maximum average latency",
			Passed:      passed,
			Expected:    fmt.Sprintf("<= %s", formatDurationNs(global.MaxAvgLatency)),
			Actual:      snapshot.AvgLatency.String(),
		})
	}

	// MinThroughput
	if global.MinThroughput != nil {
		passed := snapshot.QPS >= *global.MinThroughput
		results.Results = append(results.Results, AssertionResult{
			Name:        "global.minThroughput",
			Description: "Minimum throughput (QPS)",
			Passed:      passed,
			Expected:    fmt.Sprintf(">= %.2f req/s", *global.MinThroughput),
			Actual:      fmt.Sprintf("%.2f req/s", snapshot.QPS),
		})
	}
}

// validateEndpointAssertions validates endpoint-specific SLO assertions.
func (v *AssertionValidator) validateEndpointAssertions(snapshot Snapshot, results *AssertionResults) {
	if v.config.EndpointOverrides == nil {
		return
	}

	// Sort endpoint names for deterministic output
	endpointNames := make([]string, 0, len(v.config.EndpointOverrides))
	for name := range v.config.EndpointOverrides {
		endpointNames = append(endpointNames, name)
	}
	sort.Strings(endpointNames)

	for _, endpointName := range endpointNames {
		assertions := v.config.EndpointOverrides[endpointName]

		// Skip if disabled
		if assertions.Disabled {
			continue
		}

		// Get endpoint stats from snapshot
		epStats, exists := snapshot.EndpointStats[endpointName]
		if !exists {
			// Endpoint not found in results - skip silently
			continue
		}

		// Calculate endpoint error rate
		var errorRate float64
		if epStats.TotalRequests > 0 {
			errorRate = 100.0 - epStats.SuccessRate
		}

		// MaxErrorRate for endpoint
		if assertions.MaxErrorRate != nil {
			passed := errorRate <= *assertions.MaxErrorRate
			results.Results = append(results.Results, AssertionResult{
				Name:        fmt.Sprintf("endpoint:%s.maxErrorRate", endpointName),
				Description: fmt.Sprintf("Maximum error rate for %s", endpointName),
				Passed:      passed,
				Expected:    fmt.Sprintf("<= %.2f%%", *assertions.MaxErrorRate),
				Actual:      fmt.Sprintf("%.2f%%", errorRate),
				Endpoint:    endpointName,
			})
		}

		// MinSuccessRate for endpoint
		if assertions.MinSuccessRate != nil {
			passed := epStats.SuccessRate >= *assertions.MinSuccessRate
			results.Results = append(results.Results, AssertionResult{
				Name:        fmt.Sprintf("endpoint:%s.minSuccessRate", endpointName),
				Description: fmt.Sprintf("Minimum success rate for %s", endpointName),
				Passed:      passed,
				Expected:    fmt.Sprintf(">= %.2f%%", *assertions.MinSuccessRate),
				Actual:      fmt.Sprintf("%.2f%%", epStats.SuccessRate),
				Endpoint:    endpointName,
			})
		}

		// MaxP50Latency for endpoint
		if assertions.MaxP50Latency > 0 {
			passed := epStats.P50Latency.Nanoseconds() <= assertions.MaxP50Latency
			results.Results = append(results.Results, AssertionResult{
				Name:        fmt.Sprintf("endpoint:%s.maxP50Latency", endpointName),
				Description: fmt.Sprintf("Maximum P50 latency for %s", endpointName),
				Passed:      passed,
				Expected:    fmt.Sprintf("<= %s", formatDurationNs(assertions.MaxP50Latency)),
				Actual:      epStats.P50Latency.String(),
				Endpoint:    endpointName,
			})
		}

		// MaxP95Latency for endpoint
		if assertions.MaxP95Latency > 0 {
			passed := epStats.P95Latency.Nanoseconds() <= assertions.MaxP95Latency
			results.Results = append(results.Results, AssertionResult{
				Name:        fmt.Sprintf("endpoint:%s.maxP95Latency", endpointName),
				Description: fmt.Sprintf("Maximum P95 latency for %s", endpointName),
				Passed:      passed,
				Expected:    fmt.Sprintf("<= %s", formatDurationNs(assertions.MaxP95Latency)),
				Actual:      epStats.P95Latency.String(),
				Endpoint:    endpointName,
			})
		}

		// MaxP99Latency for endpoint
		if assertions.MaxP99Latency > 0 {
			passed := epStats.P99Latency.Nanoseconds() <= assertions.MaxP99Latency
			results.Results = append(results.Results, AssertionResult{
				Name:        fmt.Sprintf("endpoint:%s.maxP99Latency", endpointName),
				Description: fmt.Sprintf("Maximum P99 latency for %s", endpointName),
				Passed:      passed,
				Expected:    fmt.Sprintf("<= %s", formatDurationNs(assertions.MaxP99Latency)),
				Actual:      epStats.P99Latency.String(),
				Endpoint:    endpointName,
			})
		}

		// MaxAvgLatency for endpoint
		if assertions.MaxAvgLatency > 0 {
			passed := epStats.AvgLatency.Nanoseconds() <= assertions.MaxAvgLatency
			results.Results = append(results.Results, AssertionResult{
				Name:        fmt.Sprintf("endpoint:%s.maxAvgLatency", endpointName),
				Description: fmt.Sprintf("Maximum average latency for %s", endpointName),
				Passed:      passed,
				Expected:    fmt.Sprintf("<= %s", formatDurationNs(assertions.MaxAvgLatency)),
				Actual:      epStats.AvgLatency.String(),
				Endpoint:    endpointName,
			})
		}

		// MinThroughput for endpoint
		if assertions.MinThroughput != nil {
			passed := epStats.QPS >= *assertions.MinThroughput
			results.Results = append(results.Results, AssertionResult{
				Name:        fmt.Sprintf("endpoint:%s.minThroughput", endpointName),
				Description: fmt.Sprintf("Minimum throughput for %s", endpointName),
				Passed:      passed,
				Expected:    fmt.Sprintf(">= %.2f req/s", *assertions.MinThroughput),
				Actual:      fmt.Sprintf("%.2f req/s", epStats.QPS),
				Endpoint:    endpointName,
			})
		}
	}
}

// HasAssertions returns true if any assertions are configured.
func (v *AssertionValidator) HasAssertions() bool {
	global := v.config.Global

	// Check global assertions
	if global != nil {
		if global.MaxErrorRate != nil ||
			global.MinSuccessRate != nil ||
			global.MaxP50Latency > 0 ||
			global.MaxP95Latency > 0 ||
			global.MaxP99Latency > 0 ||
			global.MaxAvgLatency > 0 ||
			global.MinThroughput != nil {
			return true
		}
	}

	// Check endpoint assertions
	for _, assertions := range v.config.EndpointOverrides {
		if assertions.Disabled {
			continue
		}
		if assertions.MaxErrorRate != nil ||
			assertions.MinSuccessRate != nil ||
			assertions.MaxP50Latency > 0 ||
			assertions.MaxP95Latency > 0 ||
			assertions.MaxP99Latency > 0 ||
			assertions.MaxAvgLatency > 0 ||
			assertions.MinThroughput != nil {
			return true
		}
	}

	return false
}

// Config returns the assertion configuration.
func (v *AssertionValidator) Config() AssertionValidatorConfig {
	return v.config
}

// formatDurationNs formats nanoseconds as a human-readable duration.
func formatDurationNs(ns int64) string {
	if ns < 1000 {
		return fmt.Sprintf("%dns", ns)
	}
	if ns < 1000000 {
		return fmt.Sprintf("%.2fµs", float64(ns)/1000)
	}
	if ns < 1000000000 {
		return fmt.Sprintf("%.2fms", float64(ns)/1000000)
	}
	return fmt.Sprintf("%.2fs", float64(ns)/1000000000)
}

// FormatResults formats assertion results for display.
func FormatResults(results *AssertionResults, verbose bool) string {
	if results.TotalCount == 0 {
		return "No assertions configured"
	}

	var sb strings.Builder

	// Header
	sb.WriteString("═══════════════════════════════════════════════════════════════════════════════\n")
	sb.WriteString("                              ASSERTION RESULTS\n")
	sb.WriteString("═══════════════════════════════════════════════════════════════════════════════\n\n")

	// Summary
	if results.AllPassed {
		sb.WriteString(fmt.Sprintf("✓ All %d assertions PASSED\n\n", results.TotalCount))
	} else {
		sb.WriteString(fmt.Sprintf("✗ %d/%d assertions FAILED\n\n", results.FailedCount, results.TotalCount))
	}

	// Failed assertions first
	if results.FailedCount > 0 {
		sb.WriteString("FAILED ASSERTIONS:\n")
		sb.WriteString("─────────────────────────────────────────────────────────────────────────────────\n")
		for _, r := range results.FailedResults() {
			sb.WriteString(fmt.Sprintf("  ✗ %s\n", r.Name))
			sb.WriteString(fmt.Sprintf("    Description: %s\n", r.Description))
			sb.WriteString(fmt.Sprintf("    Expected:    %s\n", r.Expected))
			sb.WriteString(fmt.Sprintf("    Actual:      %s\n", r.Actual))
			sb.WriteString("\n")
		}
	}

	// Passed assertions (only in verbose mode)
	if verbose && results.PassedCount > 0 {
		sb.WriteString("PASSED ASSERTIONS:\n")
		sb.WriteString("─────────────────────────────────────────────────────────────────────────────────\n")
		for _, r := range results.PassedResults() {
			sb.WriteString(fmt.Sprintf("  ✓ %s\n", r.Name))
			sb.WriteString(fmt.Sprintf("    Expected: %s | Actual: %s\n", r.Expected, r.Actual))
		}
	}

	sb.WriteString("═══════════════════════════════════════════════════════════════════════════════\n")

	return sb.String()
}

// NewAssertionValidatorConfigFromDurations creates an AssertionValidatorConfig from
// config values with time.Duration fields, converting them to nanoseconds.
// This helper is useful when integrating with the config package.
func NewAssertionValidatorConfigFromDurations(
	global *GlobalAssertionsDuration,
	endpoints map[string]EndpointAssertionsDuration,
	exitOnFailure *bool,
) AssertionValidatorConfig {
	config := AssertionValidatorConfig{
		ExitOnFailure: exitOnFailure,
	}

	if global != nil {
		config.Global = &GlobalAssertions{
			MaxErrorRate:   global.MaxErrorRate,
			MinSuccessRate: global.MinSuccessRate,
			MaxP50Latency:  global.MaxP50Latency.Nanoseconds(),
			MaxP95Latency:  global.MaxP95Latency.Nanoseconds(),
			MaxP99Latency:  global.MaxP99Latency.Nanoseconds(),
			MaxAvgLatency:  global.MaxAvgLatency.Nanoseconds(),
			MinThroughput:  global.MinThroughput,
		}
	}

	if len(endpoints) > 0 {
		config.EndpointOverrides = make(map[string]EndpointAssertions, len(endpoints))
		for name, ep := range endpoints {
			config.EndpointOverrides[name] = EndpointAssertions{
				MaxErrorRate:   ep.MaxErrorRate,
				MinSuccessRate: ep.MinSuccessRate,
				MaxP50Latency:  ep.MaxP50Latency.Nanoseconds(),
				MaxP95Latency:  ep.MaxP95Latency.Nanoseconds(),
				MaxP99Latency:  ep.MaxP99Latency.Nanoseconds(),
				MaxAvgLatency:  ep.MaxAvgLatency.Nanoseconds(),
				MinThroughput:  ep.MinThroughput,
				Disabled:       ep.Disabled,
			}
		}
	}

	return config
}

// GlobalAssertionsDuration is like GlobalAssertions but uses time.Duration
// for latency fields (for use with config package).
type GlobalAssertionsDuration struct {
	MaxErrorRate   *float64
	MinSuccessRate *float64
	MaxP50Latency  time.Duration
	MaxP95Latency  time.Duration
	MaxP99Latency  time.Duration
	MaxAvgLatency  time.Duration
	MinThroughput  *float64
}

// EndpointAssertionsDuration is like EndpointAssertions but uses time.Duration
// for latency fields (for use with config package).
type EndpointAssertionsDuration struct {
	MaxErrorRate   *float64
	MinSuccessRate *float64
	MaxP50Latency  time.Duration
	MaxP95Latency  time.Duration
	MaxP99Latency  time.Duration
	MaxAvgLatency  time.Duration
	MinThroughput  *float64
	Disabled       bool
}
