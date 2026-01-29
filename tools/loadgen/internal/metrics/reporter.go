// Package metrics provides metrics collection and reporting for the load generator.
package metrics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// JSONReport represents a complete load test report in JSON format.
// It contains all information needed to analyze test results and can be
// parsed by external tools for further processing.
type JSONReport struct {
	// Metadata about the report
	Metadata ReportMetadata `json:"metadata"`

	// Configuration used for the test
	Configuration ReportConfiguration `json:"configuration"`

	// Summary statistics
	Summary ReportSummary `json:"summary"`

	// Per-endpoint statistics
	Endpoints []EndpointReport `json:"endpoints"`

	// Status code distribution
	StatusCodes map[string]int64 `json:"statusCodes"`

	// Latency distribution buckets
	LatencyDistribution LatencyDistribution `json:"latencyDistribution"`

	// Errors encountered during the test
	Errors []ErrorEntry `json:"errors,omitempty"`
}

// ReportMetadata contains metadata about the report.
type ReportMetadata struct {
	Version     string    `json:"version"`
	GeneratedAt time.Time `json:"generatedAt"`
	Generator   string    `json:"generator"`
}

// ReportConfiguration captures the test configuration.
type ReportConfiguration struct {
	Name          string           `json:"name"`
	Description   string           `json:"description,omitempty"`
	TargetBaseURL string           `json:"targetBaseURL"`
	Duration      Duration         `json:"duration"`
	TrafficShaper *TrafficShaper   `json:"trafficShaper,omitempty"`
	RateLimiter   *RateLimiter     `json:"rateLimiter,omitempty"`
	WorkerPool    *WorkerPool      `json:"workerPool,omitempty"`
	EndpointCount int              `json:"endpointCount"`
	TotalWeight   int              `json:"totalWeight"`
	Warmup        *WarmupConfig    `json:"warmup,omitempty"`
	Workflows     []WorkflowConfig `json:"workflows,omitempty"`
}

// Duration wraps time.Duration for JSON serialization.
type Duration struct {
	time.Duration
}

// MarshalJSON implements json.Marshaler for Duration.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"seconds": d.Seconds(),
		"display": formatDurationString(d.Duration),
	})
}

// UnmarshalJSON implements json.Unmarshaler for Duration.
func (d *Duration) UnmarshalJSON(data []byte) error {
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	if seconds, ok := obj["seconds"].(float64); ok {
		d.Duration = time.Duration(seconds * float64(time.Second))
	}
	return nil
}

// TrafficShaper captures traffic shaping configuration.
type TrafficShaper struct {
	Type    string  `json:"type"`
	BaseQPS float64 `json:"baseQPS"`
}

// RateLimiter captures rate limiter configuration.
type RateLimiter struct {
	Type  string  `json:"type"`
	QPS   float64 `json:"qps"`
	Burst int     `json:"burst"`
}

// WorkerPool captures worker pool configuration.
type WorkerPool struct {
	MinSize     int `json:"minSize"`
	MaxSize     int `json:"maxSize"`
	InitialSize int `json:"initialSize"`
}

// WarmupConfig captures warmup phase configuration.
type WarmupConfig struct {
	Enabled    bool     `json:"enabled"`
	Iterations int      `json:"iterations,omitempty"`
	FillTypes  []string `json:"fillTypes,omitempty"`
}

// WorkflowConfig captures workflow configuration.
type WorkflowConfig struct {
	Name       string `json:"name"`
	Weight     int    `json:"weight"`
	StepsCount int    `json:"stepsCount"`
}

// ReportSummary contains overall test statistics.
type ReportSummary struct {
	// Timing information
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	Duration  Duration  `json:"duration"`

	// Request counts
	TotalRequests   int64 `json:"totalRequests"`
	SuccessRequests int64 `json:"successRequests"`
	FailedRequests  int64 `json:"failedRequests"`
	TotalBytes      int64 `json:"totalBytes"`

	// Derived metrics
	SuccessRate float64 `json:"successRate"`
	QPS         float64 `json:"qps"`
	BytesPerSec float64 `json:"bytesPerSecond"`

	// Latency statistics (in milliseconds for readability)
	Latency LatencyStats `json:"latency"`
}

// LatencyStats contains latency statistics in milliseconds.
type LatencyStats struct {
	MinMs float64 `json:"minMs"`
	AvgMs float64 `json:"avgMs"`
	P50Ms float64 `json:"p50Ms"`
	P95Ms float64 `json:"p95Ms"`
	P99Ms float64 `json:"p99Ms"`
	MaxMs float64 `json:"maxMs"`
}

// LatencyDistribution contains latency histogram buckets.
type LatencyDistribution struct {
	Buckets []LatencyBucket `json:"buckets"`
}

// LatencyBucket represents a latency histogram bucket.
type LatencyBucket struct {
	Label    string  `json:"label"`
	MaxMs    float64 `json:"maxMs"`
	Count    int64   `json:"count"`
	Percent  float64 `json:"percent"`
	CumCount int64   `json:"cumulativeCount"`
}

// EndpointReport contains statistics for a single endpoint.
type EndpointReport struct {
	Name            string       `json:"name"`
	TotalRequests   int64        `json:"totalRequests"`
	SuccessRequests int64        `json:"successRequests"`
	FailedRequests  int64        `json:"failedRequests"`
	TotalBytes      int64        `json:"totalBytes"`
	SuccessRate     float64      `json:"successRate"`
	QPS             float64      `json:"qps"`
	Latency         LatencyStats `json:"latency"`
}

// ErrorEntry represents an error encountered during testing.
type ErrorEntry struct {
	StatusCode int    `json:"statusCode,omitempty"`
	Endpoint   string `json:"endpoint"`
	Count      int64  `json:"count"`
	Message    string `json:"message,omitempty"`
}

// Reporter generates JSON reports from test metrics.
type Reporter struct {
	version string
}

// NewReporter creates a new Reporter.
func NewReporter() *Reporter {
	return &Reporter{
		version: "1.0.0",
	}
}

// ReportOptions configures report generation.
type ReportOptions struct {
	// ConfigName is the name of the configuration used.
	ConfigName string

	// ConfigDescription is the description of the configuration.
	ConfigDescription string

	// TargetBaseURL is the target system URL.
	TargetBaseURL string

	// TestDuration is the configured test duration.
	TestDuration time.Duration

	// TrafficShaperType is the traffic shaper type.
	TrafficShaperType string

	// TrafficShaperBaseQPS is the base QPS.
	TrafficShaperBaseQPS float64

	// RateLimiterType is the rate limiter type.
	RateLimiterType string

	// RateLimiterQPS is the rate limiter QPS.
	RateLimiterQPS float64

	// RateLimiterBurst is the rate limiter burst size.
	RateLimiterBurst int

	// WorkerPoolMinSize is the minimum worker pool size.
	WorkerPoolMinSize int

	// WorkerPoolMaxSize is the maximum worker pool size.
	WorkerPoolMaxSize int

	// WorkerPoolInitialSize is the initial worker pool size.
	WorkerPoolInitialSize int

	// EndpointCount is the number of endpoints.
	EndpointCount int

	// TotalWeight is the total endpoint weight.
	TotalWeight int

	// WarmupEnabled indicates if warmup was enabled.
	WarmupEnabled bool

	// WarmupIterations is the number of warmup iterations.
	WarmupIterations int

	// WarmupFillTypes is the list of semantic types to fill during warmup.
	WarmupFillTypes []string

	// Workflows is the list of workflow configurations.
	Workflows []WorkflowConfig

	// Errors is the list of errors encountered.
	Errors []ErrorEntry
}

// GenerateReport creates a JSON report from a metrics snapshot.
func (r *Reporter) GenerateReport(snapshot Snapshot, opts ReportOptions) *JSONReport {
	report := &JSONReport{
		Metadata: ReportMetadata{
			Version:     r.version,
			GeneratedAt: time.Now().UTC(),
			Generator:   "loadgen",
		},
		Configuration: ReportConfiguration{
			Name:          opts.ConfigName,
			Description:   opts.ConfigDescription,
			TargetBaseURL: opts.TargetBaseURL,
			Duration:      Duration{opts.TestDuration},
			EndpointCount: opts.EndpointCount,
			TotalWeight:   opts.TotalWeight,
		},
		Summary: r.buildSummary(snapshot),
		StatusCodes: func() map[string]int64 {
			result := make(map[string]int64)
			for code, count := range snapshot.StatusCodes {
				result[fmt.Sprintf("%d", code)] = count
			}
			return result
		}(),
		LatencyDistribution: r.buildLatencyDistribution(snapshot),
		Errors:              opts.Errors,
	}

	// Add traffic shaper config if provided
	if opts.TrafficShaperType != "" {
		report.Configuration.TrafficShaper = &TrafficShaper{
			Type:    opts.TrafficShaperType,
			BaseQPS: opts.TrafficShaperBaseQPS,
		}
	}

	// Add rate limiter config if provided
	if opts.RateLimiterType != "" {
		report.Configuration.RateLimiter = &RateLimiter{
			Type:  opts.RateLimiterType,
			QPS:   opts.RateLimiterQPS,
			Burst: opts.RateLimiterBurst,
		}
	}

	// Add worker pool config if provided
	if opts.WorkerPoolMaxSize > 0 {
		report.Configuration.WorkerPool = &WorkerPool{
			MinSize:     opts.WorkerPoolMinSize,
			MaxSize:     opts.WorkerPoolMaxSize,
			InitialSize: opts.WorkerPoolInitialSize,
		}
	}

	// Add warmup config if enabled
	if opts.WarmupEnabled {
		report.Configuration.Warmup = &WarmupConfig{
			Enabled:    true,
			Iterations: opts.WarmupIterations,
			FillTypes:  opts.WarmupFillTypes,
		}
	}

	// Add workflow configs
	if len(opts.Workflows) > 0 {
		report.Configuration.Workflows = opts.Workflows
	}

	// Build endpoint reports
	report.Endpoints = r.buildEndpointReports(snapshot)

	return report
}

// buildSummary creates the summary section from a snapshot.
func (r *Reporter) buildSummary(snapshot Snapshot) ReportSummary {
	var bytesPerSec float64
	if snapshot.Duration > 0 {
		bytesPerSec = float64(snapshot.TotalBytes) / snapshot.Duration.Seconds()
	}

	return ReportSummary{
		StartTime:       snapshot.StartTime,
		EndTime:         snapshot.EndTime,
		Duration:        Duration{snapshot.Duration},
		TotalRequests:   snapshot.TotalRequests,
		SuccessRequests: snapshot.SuccessRequests,
		FailedRequests:  snapshot.FailedRequests,
		TotalBytes:      snapshot.TotalBytes,
		SuccessRate:     snapshot.SuccessRate,
		QPS:             snapshot.QPS,
		BytesPerSec:     bytesPerSec,
		Latency:         r.convertLatencyStats(snapshot),
	}
}

// convertLatencyStats converts duration-based latencies to milliseconds.
func (r *Reporter) convertLatencyStats(snapshot Snapshot) LatencyStats {
	return LatencyStats{
		MinMs: float64(snapshot.MinLatency.Nanoseconds()) / 1e6,
		AvgMs: float64(snapshot.AvgLatency.Nanoseconds()) / 1e6,
		P50Ms: float64(snapshot.P50Latency.Nanoseconds()) / 1e6,
		P95Ms: float64(snapshot.P95Latency.Nanoseconds()) / 1e6,
		P99Ms: float64(snapshot.P99Latency.Nanoseconds()) / 1e6,
		MaxMs: float64(snapshot.MaxLatency.Nanoseconds()) / 1e6,
	}
}

// buildLatencyDistribution creates latency histogram buckets from a snapshot.
func (r *Reporter) buildLatencyDistribution(snapshot Snapshot) LatencyDistribution {
	if snapshot.TotalRequests == 0 {
		return LatencyDistribution{Buckets: []LatencyBucket{}}
	}

	// Define standard latency buckets
	buckets := []struct {
		label string
		maxMs float64
	}{
		{"< 10ms", 10},
		{"< 50ms", 50},
		{"< 100ms", 100},
		{"< 500ms", 500},
		{"< 1s", 1000},
		{">= 1s", 0}, // 0 indicates infinity
	}

	// Estimate distribution based on percentiles
	// This is approximate since we only have percentile data
	latencyMs := []float64{
		float64(snapshot.MinLatency.Nanoseconds()) / 1e6,
		float64(snapshot.P50Latency.Nanoseconds()) / 1e6,
		float64(snapshot.P95Latency.Nanoseconds()) / 1e6,
		float64(snapshot.P99Latency.Nanoseconds()) / 1e6,
		float64(snapshot.MaxLatency.Nanoseconds()) / 1e6,
	}

	result := make([]LatencyBucket, len(buckets))
	var cumCount int64

	for i, bucket := range buckets {
		count := int64(0)
		for _, lat := range latencyMs {
			maxMs := bucket.maxMs
			if maxMs == 0 {
				maxMs = 1e9 // Effectively infinity
			}
			if lat <= maxMs {
				count++
			}
		}

		pct := float64(count) / float64(len(latencyMs)) * 100
		cumCount += count

		maxMs := bucket.maxMs
		if maxMs == 0 {
			maxMs = -1 // Indicate unbounded
		}

		result[i] = LatencyBucket{
			Label:    bucket.label,
			MaxMs:    maxMs,
			Count:    count,
			Percent:  pct,
			CumCount: cumCount,
		}
	}

	return LatencyDistribution{Buckets: result}
}

// buildEndpointReports creates endpoint reports from a snapshot.
// Endpoints are sorted by name for deterministic output across runs.
func (r *Reporter) buildEndpointReports(snapshot Snapshot) []EndpointReport {
	// Sort endpoint names for deterministic ordering
	names := make([]string, 0, len(snapshot.EndpointStats))
	for name := range snapshot.EndpointStats {
		names = append(names, name)
	}
	sort.Strings(names)

	reports := make([]EndpointReport, 0, len(names))

	for _, name := range names {
		stats := snapshot.EndpointStats[name]
		report := EndpointReport{
			Name:            name,
			TotalRequests:   stats.TotalRequests,
			SuccessRequests: stats.SuccessRequests,
			FailedRequests:  stats.FailedRequests,
			TotalBytes:      stats.TotalBytes,
			SuccessRate:     stats.SuccessRate,
			QPS:             stats.QPS,
			Latency: LatencyStats{
				MinMs: float64(stats.MinLatency.Nanoseconds()) / 1e6,
				AvgMs: float64(stats.AvgLatency.Nanoseconds()) / 1e6,
				P50Ms: float64(stats.P50Latency.Nanoseconds()) / 1e6,
				P95Ms: float64(stats.P95Latency.Nanoseconds()) / 1e6,
				P99Ms: float64(stats.P99Latency.Nanoseconds()) / 1e6,
				MaxMs: float64(stats.MaxLatency.Nanoseconds()) / 1e6,
			},
		}
		reports = append(reports, report)
	}

	return reports
}

// ToJSON serializes a report to JSON bytes.
func (r *Reporter) ToJSON(report *JSONReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

// WriteToFile writes a report to a file.
// The path supports template variables:
// - {{.Timestamp}} - Current timestamp in format YYYYMMDD-HHMMSS
// - {{.Date}} - Current date in format YYYY-MM-DD
// - {{.Time}} - Current time in format HHMMSS
func (r *Reporter) WriteToFile(report *JSONReport, path string) error {
	// Expand path templates
	expandedPath := expandPathTemplate(path)

	// Clean the path to normalize and prevent path traversal issues
	expandedPath = filepath.Clean(expandedPath)

	// Ensure directory exists
	dir := filepath.Dir(expandedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Marshal to JSON
	data, err := r.ToJSON(report)
	if err != nil {
		return fmt.Errorf("marshaling report to JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(expandedPath, data, 0644); err != nil {
		return fmt.Errorf("writing report file: %w", err)
	}

	return nil
}

// expandPathTemplate expands template variables in a path.
func expandPathTemplate(path string) string {
	now := time.Now()

	replacements := map[string]string{
		"{{.Timestamp}}": now.Format("20060102-150405"),
		"{{.Date}}":      now.Format("2006-01-02"),
		"{{.Time}}":      now.Format("150405"),
	}

	result := path
	for template, value := range replacements {
		result = strings.ReplaceAll(result, template, value)
	}

	return result
}

// formatDurationString formats a duration for display.
func formatDurationString(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", hours, minutes)
}
