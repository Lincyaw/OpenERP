package metrics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewReporter(t *testing.T) {
	r := NewReporter()
	assert.NotNil(t, r)
	assert.Equal(t, "1.0.0", r.version)
}

func TestReporter_GenerateReport(t *testing.T) {
	r := NewReporter()

	// Create a test snapshot
	snapshot := createTestSnapshot()

	opts := ReportOptions{
		ConfigName:            "test-config",
		ConfigDescription:     "Test configuration",
		TargetBaseURL:         "http://localhost:8080",
		TestDuration:          5 * time.Minute,
		TrafficShaperType:     "sine",
		TrafficShaperBaseQPS:  100,
		RateLimiterType:       "token_bucket",
		RateLimiterQPS:        150,
		RateLimiterBurst:      20,
		WorkerPoolMinSize:     5,
		WorkerPoolMaxSize:     50,
		WorkerPoolInitialSize: 10,
		EndpointCount:         10,
		TotalWeight:           50,
		WarmupEnabled:         true,
		WarmupIterations:      3,
		WarmupFillTypes:       []string{"entity.product.id", "entity.customer.id"},
		Workflows: []WorkflowConfig{
			{Name: "sales_cycle", Weight: 10, StepsCount: 4},
			{Name: "purchase_cycle", Weight: 5, StepsCount: 3},
		},
		Errors: []ErrorEntry{
			{StatusCode: 500, Endpoint: "POST /api/v1/orders", Count: 5, Message: "Internal server error"},
		},
	}

	report := r.GenerateReport(snapshot, opts)

	// Verify metadata
	assert.Equal(t, "1.0.0", report.Metadata.Version)
	assert.Equal(t, "loadgen", report.Metadata.Generator)
	assert.False(t, report.Metadata.GeneratedAt.IsZero())

	// Verify configuration
	assert.Equal(t, "test-config", report.Configuration.Name)
	assert.Equal(t, "Test configuration", report.Configuration.Description)
	assert.Equal(t, "http://localhost:8080", report.Configuration.TargetBaseURL)
	assert.Equal(t, 5*time.Minute, report.Configuration.Duration.Duration)
	assert.Equal(t, 10, report.Configuration.EndpointCount)
	assert.Equal(t, 50, report.Configuration.TotalWeight)

	// Verify traffic shaper config
	require.NotNil(t, report.Configuration.TrafficShaper)
	assert.Equal(t, "sine", report.Configuration.TrafficShaper.Type)
	assert.Equal(t, 100.0, report.Configuration.TrafficShaper.BaseQPS)

	// Verify rate limiter config
	require.NotNil(t, report.Configuration.RateLimiter)
	assert.Equal(t, "token_bucket", report.Configuration.RateLimiter.Type)
	assert.Equal(t, 150.0, report.Configuration.RateLimiter.QPS)
	assert.Equal(t, 20, report.Configuration.RateLimiter.Burst)

	// Verify worker pool config
	require.NotNil(t, report.Configuration.WorkerPool)
	assert.Equal(t, 5, report.Configuration.WorkerPool.MinSize)
	assert.Equal(t, 50, report.Configuration.WorkerPool.MaxSize)
	assert.Equal(t, 10, report.Configuration.WorkerPool.InitialSize)

	// Verify warmup config
	require.NotNil(t, report.Configuration.Warmup)
	assert.True(t, report.Configuration.Warmup.Enabled)
	assert.Equal(t, 3, report.Configuration.Warmup.Iterations)
	assert.Equal(t, []string{"entity.product.id", "entity.customer.id"}, report.Configuration.Warmup.FillTypes)

	// Verify workflows
	require.Len(t, report.Configuration.Workflows, 2)
	assert.Equal(t, "sales_cycle", report.Configuration.Workflows[0].Name)
	assert.Equal(t, 10, report.Configuration.Workflows[0].Weight)

	// Verify summary
	assert.Equal(t, snapshot.TotalRequests, report.Summary.TotalRequests)
	assert.Equal(t, snapshot.SuccessRequests, report.Summary.SuccessRequests)
	assert.Equal(t, snapshot.FailedRequests, report.Summary.FailedRequests)
	assert.InDelta(t, snapshot.SuccessRate, report.Summary.SuccessRate, 0.01)
	assert.InDelta(t, snapshot.QPS, report.Summary.QPS, 0.01)

	// Verify latency stats
	assert.InDelta(t, float64(snapshot.MinLatency.Nanoseconds())/1e6, report.Summary.Latency.MinMs, 0.01)
	assert.InDelta(t, float64(snapshot.P95Latency.Nanoseconds())/1e6, report.Summary.Latency.P95Ms, 0.01)

	// Verify status codes
	assert.NotEmpty(t, report.StatusCodes)
	assert.Equal(t, snapshot.StatusCodes[200], report.StatusCodes["200"])

	// Verify endpoints
	assert.NotEmpty(t, report.Endpoints)

	// Verify errors
	require.Len(t, report.Errors, 1)
	assert.Equal(t, 500, report.Errors[0].StatusCode)
	assert.Equal(t, "POST /api/v1/orders", report.Errors[0].Endpoint)
}

func TestReporter_ToJSON(t *testing.T) {
	r := NewReporter()
	snapshot := createTestSnapshot()
	report := r.GenerateReport(snapshot, ReportOptions{
		ConfigName:    "test",
		TargetBaseURL: "http://localhost:8080",
		TestDuration:  time.Minute,
	})

	// Convert to JSON
	data, err := r.ToJSON(report)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Verify it's valid JSON by unmarshaling
	var parsed map[string]any
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	// Verify top-level structure
	assert.Contains(t, parsed, "metadata")
	assert.Contains(t, parsed, "configuration")
	assert.Contains(t, parsed, "summary")
	assert.Contains(t, parsed, "endpoints")
	assert.Contains(t, parsed, "statusCodes")
	assert.Contains(t, parsed, "latencyDistribution")
}

func TestReporter_WriteToFile(t *testing.T) {
	r := NewReporter()
	snapshot := createTestSnapshot()
	report := r.GenerateReport(snapshot, ReportOptions{
		ConfigName:    "test",
		TargetBaseURL: "http://localhost:8080",
		TestDuration:  time.Minute,
	})

	// Create temp directory
	tmpDir := t.TempDir()

	t.Run("simple path", func(t *testing.T) {
		path := filepath.Join(tmpDir, "report.json")
		err := r.WriteToFile(report, path)
		require.NoError(t, err)

		// Verify file exists and is valid JSON
		data, err := os.ReadFile(path)
		require.NoError(t, err)

		var parsed JSONReport
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)
		assert.Equal(t, report.Configuration.Name, parsed.Configuration.Name)
	})

	t.Run("nested directory", func(t *testing.T) {
		path := filepath.Join(tmpDir, "nested", "dir", "report.json")
		err := r.WriteToFile(report, path)
		require.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(path)
		require.NoError(t, err)
	})

	t.Run("timestamp template", func(t *testing.T) {
		path := filepath.Join(tmpDir, "report-{{.Timestamp}}.json")
		err := r.WriteToFile(report, path)
		require.NoError(t, err)

		// Find the created file (should have timestamp expanded)
		files, err := os.ReadDir(tmpDir)
		require.NoError(t, err)

		found := false
		for _, f := range files {
			if strings.HasPrefix(f.Name(), "report-") && strings.HasSuffix(f.Name(), ".json") && f.Name() != "report-{{.Timestamp}}.json" {
				found = true
				// Verify timestamp format (YYYYMMDD-HHMMSS)
				name := strings.TrimPrefix(f.Name(), "report-")
				name = strings.TrimSuffix(name, ".json")
				assert.Regexp(t, `^\d{8}-\d{6}$`, name)
				break
			}
		}
		assert.True(t, found, "File with expanded timestamp should exist")
	})

	t.Run("date template", func(t *testing.T) {
		path := filepath.Join(tmpDir, "output", "report-{{.Date}}.json")
		err := r.WriteToFile(report, path)
		require.NoError(t, err)

		// Find the created file
		files, err := os.ReadDir(filepath.Join(tmpDir, "output"))
		require.NoError(t, err)

		found := false
		for _, f := range files {
			if strings.HasPrefix(f.Name(), "report-") {
				found = true
				// Verify date format (YYYY-MM-DD)
				name := strings.TrimPrefix(f.Name(), "report-")
				name = strings.TrimSuffix(name, ".json")
				assert.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, name)
				break
			}
		}
		assert.True(t, found, "File with expanded date should exist")
	})
}

func TestExpandPathTemplate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(t *testing.T, result string)
	}{
		{
			name:  "no template",
			input: "path/to/file.json",
			validate: func(t *testing.T, result string) {
				assert.Equal(t, "path/to/file.json", result)
			},
		},
		{
			name:  "timestamp template",
			input: "results/{{.Timestamp}}.json",
			validate: func(t *testing.T, result string) {
				assert.NotContains(t, result, "{{.Timestamp}}")
				assert.Regexp(t, `results/\d{8}-\d{6}\.json`, result)
			},
		},
		{
			name:  "date template",
			input: "results/{{.Date}}/report.json",
			validate: func(t *testing.T, result string) {
				assert.NotContains(t, result, "{{.Date}}")
				assert.Regexp(t, `results/\d{4}-\d{2}-\d{2}/report\.json`, result)
			},
		},
		{
			name:  "time template",
			input: "results/{{.Time}}.json",
			validate: func(t *testing.T, result string) {
				assert.NotContains(t, result, "{{.Time}}")
				assert.Regexp(t, `results/\d{6}\.json`, result)
			},
		},
		{
			name:  "multiple templates",
			input: "results/{{.Date}}/loadgen-{{.Timestamp}}.json",
			validate: func(t *testing.T, result string) {
				assert.NotContains(t, result, "{{")
				assert.Regexp(t, `results/\d{4}-\d{2}-\d{2}/loadgen-\d{8}-\d{6}\.json`, result)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := expandPathTemplate(tc.input)
			tc.validate(t, result)
		})
	}
}

func TestDuration_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		contains []string
	}{
		{
			name:     "seconds only",
			duration: 30 * time.Second,
			contains: []string{`"seconds":30`, `"display":"30.0s"`},
		},
		{
			name:     "minutes and seconds",
			duration: 2*time.Minute + 30*time.Second,
			contains: []string{`"seconds":150`, `"display":"2m30s"`},
		},
		{
			name:     "hours and minutes",
			duration: 1*time.Hour + 30*time.Minute,
			contains: []string{`"seconds":5400`, `"display":"1h30m"`},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := Duration{tc.duration}
			data, err := json.Marshal(d)
			require.NoError(t, err)

			for _, expected := range tc.contains {
				assert.Contains(t, string(data), expected)
			}
		})
	}
}

func TestDuration_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected time.Duration
	}{
		{
			name:     "30 seconds",
			json:     `{"seconds":30,"display":"30.0s"}`,
			expected: 30 * time.Second,
		},
		{
			name:     "5 minutes",
			json:     `{"seconds":300,"display":"5m0s"}`,
			expected: 5 * time.Minute,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var d Duration
			err := json.Unmarshal([]byte(tc.json), &d)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, d.Duration)
		})
	}
}

func TestJSONReport_RoundTrip(t *testing.T) {
	// Create a full report
	r := NewReporter()
	snapshot := createTestSnapshot()
	report := r.GenerateReport(snapshot, ReportOptions{
		ConfigName:            "integration-test",
		ConfigDescription:     "Full integration test",
		TargetBaseURL:         "http://localhost:8080/api/v1",
		TestDuration:          10 * time.Minute,
		TrafficShaperType:     "step",
		TrafficShaperBaseQPS:  50,
		RateLimiterType:       "leaky_bucket",
		RateLimiterQPS:        100,
		RateLimiterBurst:      10,
		WorkerPoolMinSize:     2,
		WorkerPoolMaxSize:     100,
		WorkerPoolInitialSize: 10,
		EndpointCount:         25,
		TotalWeight:           100,
		WarmupEnabled:         true,
		WarmupIterations:      5,
		WarmupFillTypes:       []string{"entity.product.id"},
		Workflows: []WorkflowConfig{
			{Name: "order_flow", Weight: 20, StepsCount: 5},
		},
		Errors: []ErrorEntry{
			{StatusCode: 429, Endpoint: "GET /api/products", Count: 10, Message: "Rate limited"},
			{StatusCode: 503, Endpoint: "POST /api/orders", Count: 3, Message: "Service unavailable"},
		},
	})

	// Marshal to JSON
	data, err := r.ToJSON(report)
	require.NoError(t, err)

	// Unmarshal back
	var parsed JSONReport
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	// Verify all fields preserved
	assert.Equal(t, report.Metadata.Version, parsed.Metadata.Version)
	assert.Equal(t, report.Metadata.Generator, parsed.Metadata.Generator)

	assert.Equal(t, report.Configuration.Name, parsed.Configuration.Name)
	assert.Equal(t, report.Configuration.Description, parsed.Configuration.Description)
	assert.Equal(t, report.Configuration.TargetBaseURL, parsed.Configuration.TargetBaseURL)
	assert.Equal(t, report.Configuration.EndpointCount, parsed.Configuration.EndpointCount)

	assert.Equal(t, report.Summary.TotalRequests, parsed.Summary.TotalRequests)
	assert.Equal(t, report.Summary.SuccessRate, parsed.Summary.SuccessRate)

	assert.Equal(t, len(report.StatusCodes), len(parsed.StatusCodes))
	assert.Equal(t, len(report.Endpoints), len(parsed.Endpoints))
	assert.Equal(t, len(report.Errors), len(parsed.Errors))
}

func TestJSONReport_ExternalToolCompatibility(t *testing.T) {
	// This test verifies that the JSON output can be parsed by external tools
	// by using the standard library's json.Unmarshal into generic types

	r := NewReporter()
	snapshot := createTestSnapshot()
	report := r.GenerateReport(snapshot, ReportOptions{
		ConfigName:    "external-tool-test",
		TargetBaseURL: "http://localhost:8080",
		TestDuration:  time.Minute,
	})

	data, err := r.ToJSON(report)
	require.NoError(t, err)

	// Parse as generic map (like jq would)
	var generic map[string]any
	err = json.Unmarshal(data, &generic)
	require.NoError(t, err)

	// Verify we can access nested fields using standard JSON paths
	t.Run("access metadata.version", func(t *testing.T) {
		metadata, ok := generic["metadata"].(map[string]any)
		require.True(t, ok)
		version, ok := metadata["version"].(string)
		require.True(t, ok)
		assert.Equal(t, "1.0.0", version)
	})

	t.Run("access summary.totalRequests", func(t *testing.T) {
		summary, ok := generic["summary"].(map[string]any)
		require.True(t, ok)
		totalRequests, ok := summary["totalRequests"].(float64) // JSON numbers are float64
		require.True(t, ok)
		assert.Equal(t, float64(snapshot.TotalRequests), totalRequests)
	})

	t.Run("access summary.latency.p95Ms", func(t *testing.T) {
		summary, ok := generic["summary"].(map[string]any)
		require.True(t, ok)
		latency, ok := summary["latency"].(map[string]any)
		require.True(t, ok)
		p95Ms, ok := latency["p95Ms"].(float64)
		require.True(t, ok)
		assert.Greater(t, p95Ms, 0.0)
	})

	t.Run("access endpoints array", func(t *testing.T) {
		endpoints, ok := generic["endpoints"].([]any)
		require.True(t, ok)
		assert.NotEmpty(t, endpoints)

		// Access first endpoint
		if len(endpoints) > 0 {
			ep, ok := endpoints[0].(map[string]any)
			require.True(t, ok)
			_, ok = ep["name"].(string)
			require.True(t, ok)
		}
	})

	t.Run("access statusCodes map", func(t *testing.T) {
		statusCodes, ok := generic["statusCodes"].(map[string]any)
		require.True(t, ok)
		// Status codes are stored with string keys (e.g., "200", "404")
		for key := range statusCodes {
			assert.Regexp(t, `^\d+$`, key)
		}
	})
}

func TestLatencyDistribution(t *testing.T) {
	r := NewReporter()
	snapshot := Snapshot{
		TotalRequests: 1000,
		MinLatency:    5 * time.Millisecond,
		AvgLatency:    25 * time.Millisecond,
		P50Latency:    20 * time.Millisecond,
		P95Latency:    80 * time.Millisecond,
		P99Latency:    200 * time.Millisecond,
		MaxLatency:    500 * time.Millisecond,
	}

	dist := r.buildLatencyDistribution(snapshot)

	assert.NotEmpty(t, dist.Buckets)

	// Verify bucket structure
	for _, bucket := range dist.Buckets {
		assert.NotEmpty(t, bucket.Label)
		assert.GreaterOrEqual(t, bucket.Count, int64(0))
		assert.GreaterOrEqual(t, bucket.Percent, 0.0)
		assert.LessOrEqual(t, bucket.Percent, 100.0)
	}
}

func TestReporter_EmptySnapshot(t *testing.T) {
	r := NewReporter()
	snapshot := Snapshot{} // Empty snapshot

	report := r.GenerateReport(snapshot, ReportOptions{
		ConfigName:    "empty-test",
		TargetBaseURL: "http://localhost:8080",
	})

	// Should still produce valid JSON
	data, err := r.ToJSON(report)
	require.NoError(t, err)

	var parsed JSONReport
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	// Verify zero values are handled properly
	assert.Equal(t, int64(0), parsed.Summary.TotalRequests)
	assert.Equal(t, 0.0, parsed.Summary.SuccessRate)
	assert.Empty(t, parsed.Endpoints)
}

func TestFormatDurationString(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Second, "30.0s"},
		{90 * time.Second, "1m30s"},
		{5 * time.Minute, "5m0s"},
		{time.Hour + 30*time.Minute, "1h30m"},
		{2*time.Hour + 45*time.Minute, "2h45m"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			result := formatDurationString(tc.duration)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// createTestSnapshot creates a realistic test snapshot for testing.
func createTestSnapshot() Snapshot {
	now := time.Now()
	return Snapshot{
		StartTime:       now.Add(-5 * time.Minute),
		EndTime:         now,
		Duration:        5 * time.Minute,
		TotalRequests:   10000,
		SuccessRequests: 9800,
		FailedRequests:  200,
		TotalBytes:      5000000,
		MinLatency:      5 * time.Millisecond,
		AvgLatency:      25 * time.Millisecond,
		P50Latency:      20 * time.Millisecond,
		P95Latency:      80 * time.Millisecond,
		P99Latency:      150 * time.Millisecond,
		MaxLatency:      500 * time.Millisecond,
		SuccessRate:     98.0,
		QPS:             33.33,
		StatusCodes: map[int]int64{
			200: 9500,
			201: 300,
			400: 50,
			404: 100,
			500: 50,
		},
		EndpointStats: map[string]*EndpointSnapshot{
			"GET /api/v1/products": {
				Name:            "GET /api/v1/products",
				TotalRequests:   5000,
				SuccessRequests: 4900,
				FailedRequests:  100,
				TotalBytes:      2500000,
				MinLatency:      3 * time.Millisecond,
				AvgLatency:      20 * time.Millisecond,
				P50Latency:      15 * time.Millisecond,
				P95Latency:      60 * time.Millisecond,
				P99Latency:      120 * time.Millisecond,
				MaxLatency:      400 * time.Millisecond,
				SuccessRate:     98.0,
				QPS:             16.67,
			},
			"POST /api/v1/orders": {
				Name:            "POST /api/v1/orders",
				TotalRequests:   3000,
				SuccessRequests: 2900,
				FailedRequests:  100,
				TotalBytes:      1500000,
				MinLatency:      10 * time.Millisecond,
				AvgLatency:      35 * time.Millisecond,
				P50Latency:      30 * time.Millisecond,
				P95Latency:      100 * time.Millisecond,
				P99Latency:      200 * time.Millisecond,
				MaxLatency:      500 * time.Millisecond,
				SuccessRate:     96.67,
				QPS:             10.0,
			},
		},
	}
}
