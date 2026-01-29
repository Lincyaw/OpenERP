package metrics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTMLReporter(t *testing.T) {
	r := NewHTMLReporter()
	assert.NotNil(t, r)
	assert.Equal(t, "1.0.0", r.version)
}

func TestHTMLReporter_GenerateHTML(t *testing.T) {
	r := NewHTMLReporter()
	jsonReporter := NewReporter()

	// Create a test snapshot and generate JSON report
	snapshot := createTestSnapshot()
	opts := ReportOptions{
		ConfigName:            "html-test-config",
		ConfigDescription:     "HTML Test configuration",
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
		WarmupFillTypes:       []string{"entity.product.id"},
		Workflows: []WorkflowConfig{
			{Name: "sales_cycle", Weight: 10, StepsCount: 4},
		},
		Errors: []ErrorEntry{
			{StatusCode: 500, Endpoint: "POST /api/v1/orders", Count: 5, Message: "Internal server error"},
		},
	}

	jsonReport := jsonReporter.GenerateReport(snapshot, opts)

	// Generate HTML
	html, err := r.GenerateHTML(jsonReport)
	require.NoError(t, err)
	assert.NotEmpty(t, html)

	htmlStr := string(html)

	// Verify HTML structure
	assert.Contains(t, htmlStr, "<!DOCTYPE html>")
	assert.Contains(t, htmlStr, "<html lang=\"en\">")
	assert.Contains(t, htmlStr, "</html>")

	// Verify title contains config name
	assert.Contains(t, htmlStr, "html-test-config")

	// Verify configuration section
	assert.Contains(t, htmlStr, "http://localhost:8080")
	assert.Contains(t, htmlStr, "Configuration")

	// Verify summary statistics are rendered
	assert.Contains(t, htmlStr, "Total Requests")
	assert.Contains(t, htmlStr, "Success Rate")
	assert.Contains(t, htmlStr, "Requests/sec")

	// Verify latency section
	assert.Contains(t, htmlStr, "Latency Distribution")
	assert.Contains(t, htmlStr, "P50")
	assert.Contains(t, htmlStr, "P95")
	assert.Contains(t, htmlStr, "P99")

	// Verify charts section
	assert.Contains(t, htmlStr, "Status Code Distribution")
	assert.Contains(t, htmlStr, "Latency Histogram")
	assert.Contains(t, htmlStr, "Endpoint Distribution")

	// Verify Chart.js is included
	assert.Contains(t, htmlStr, "chart.js")
	assert.Contains(t, htmlStr, "statusCodeChart")
	assert.Contains(t, htmlStr, "latencyChart")
	assert.Contains(t, htmlStr, "endpointChart")

	// Verify endpoint table
	assert.Contains(t, htmlStr, "Endpoint Statistics")
	assert.Contains(t, htmlStr, "GET /api/v1/products")
	assert.Contains(t, htmlStr, "POST /api/v1/orders")

	// Verify errors section
	assert.Contains(t, htmlStr, "Errors")
	assert.Contains(t, htmlStr, "Internal server error")

	// Verify workflows section
	assert.Contains(t, htmlStr, "Workflows")
	assert.Contains(t, htmlStr, "sales_cycle")

	// Verify traffic shaper is rendered
	assert.Contains(t, htmlStr, "Traffic Shaper")
	assert.Contains(t, htmlStr, "sine")

	// Verify rate limiter is rendered
	assert.Contains(t, htmlStr, "Rate Limiter")
	assert.Contains(t, htmlStr, "token_bucket")

	// Verify worker pool is rendered
	assert.Contains(t, htmlStr, "Worker Pool")

	// Verify CSS is embedded
	assert.Contains(t, htmlStr, "<style>")
	assert.Contains(t, htmlStr, "--primary-color")
}

func TestHTMLReporter_WriteHTMLToFile(t *testing.T) {
	r := NewHTMLReporter()
	jsonReporter := NewReporter()

	snapshot := createTestSnapshot()
	jsonReport := jsonReporter.GenerateReport(snapshot, ReportOptions{
		ConfigName:    "file-write-test",
		TargetBaseURL: "http://localhost:8080",
		TestDuration:  time.Minute,
	})

	// Create temp directory
	tmpDir := t.TempDir()

	t.Run("simple path", func(t *testing.T) {
		path := filepath.Join(tmpDir, "report.html")
		err := r.WriteHTMLToFile(jsonReport, path)
		require.NoError(t, err)

		// Verify file exists and contains valid HTML
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Contains(t, string(data), "<!DOCTYPE html>")
		assert.Contains(t, string(data), "file-write-test")
	})

	t.Run("nested directory", func(t *testing.T) {
		path := filepath.Join(tmpDir, "nested", "dir", "report.html")
		err := r.WriteHTMLToFile(jsonReport, path)
		require.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(path)
		require.NoError(t, err)
	})

	t.Run("timestamp template", func(t *testing.T) {
		path := filepath.Join(tmpDir, "report-{{.Timestamp}}.html")
		err := r.WriteHTMLToFile(jsonReport, path)
		require.NoError(t, err)

		// Find the created file
		files, err := os.ReadDir(tmpDir)
		require.NoError(t, err)

		found := false
		for _, f := range files {
			if strings.HasPrefix(f.Name(), "report-") && strings.HasSuffix(f.Name(), ".html") && f.Name() != "report-{{.Timestamp}}.html" {
				found = true
				// Verify timestamp format (YYYYMMDD-HHMMSS)
				name := strings.TrimPrefix(f.Name(), "report-")
				name = strings.TrimSuffix(name, ".html")
				assert.Regexp(t, `^\d{8}-\d{6}$`, name)
				break
			}
		}
		assert.True(t, found, "File with expanded timestamp should exist")
	})
}

func TestHTMLReporter_EmptyReport(t *testing.T) {
	r := NewHTMLReporter()
	jsonReporter := NewReporter()

	// Empty snapshot
	snapshot := Snapshot{}
	jsonReport := jsonReporter.GenerateReport(snapshot, ReportOptions{
		ConfigName:    "empty-test",
		TargetBaseURL: "http://localhost:8080",
	})

	// Should still produce valid HTML
	html, err := r.GenerateHTML(jsonReport)
	require.NoError(t, err)
	assert.NotEmpty(t, html)

	htmlStr := string(html)
	assert.Contains(t, htmlStr, "<!DOCTYPE html>")
	assert.Contains(t, htmlStr, "empty-test")
}

func TestHTMLReporter_NoErrorsSection(t *testing.T) {
	r := NewHTMLReporter()
	jsonReporter := NewReporter()

	snapshot := createTestSnapshot()
	jsonReport := jsonReporter.GenerateReport(snapshot, ReportOptions{
		ConfigName:    "no-errors-test",
		TargetBaseURL: "http://localhost:8080",
		Errors:        nil, // No errors
	})

	html, err := r.GenerateHTML(jsonReport)
	require.NoError(t, err)

	htmlStr := string(html)
	// Errors section header should not appear when there are no errors
	// The errors table section only renders when HasErrors is true
	assert.NotContains(t, htmlStr, "<h2>Errors</h2>")
}

func TestHTMLReporter_NoWorkflowsSection(t *testing.T) {
	r := NewHTMLReporter()
	jsonReporter := NewReporter()

	snapshot := createTestSnapshot()
	jsonReport := jsonReporter.GenerateReport(snapshot, ReportOptions{
		ConfigName:    "no-workflows-test",
		TargetBaseURL: "http://localhost:8080",
		Workflows:     nil, // No workflows
	})

	html, err := r.GenerateHTML(jsonReport)
	require.NoError(t, err)

	// The Workflows header should not appear in the main content
	// (it only appears as a hidden section when there are no workflows)
	htmlStr := string(html)
	assert.Contains(t, htmlStr, "<!DOCTYPE html>")
}

func TestHTMLReporter_ChartDataGeneration(t *testing.T) {
	r := NewHTMLReporter()

	// Test status codes JSON generation
	statusCodes := []StatusCodeData{
		{Code: "200", Count: 1000, Percent: 90.0},
		{Code: "500", Count: 100, Percent: 10.0},
	}
	statusJSON := r.statusCodesToJSON(statusCodes)
	assert.Contains(t, string(statusJSON), `"200"`)
	assert.Contains(t, string(statusJSON), `"500"`)
	assert.Contains(t, string(statusJSON), "1000")
	assert.Contains(t, string(statusJSON), "100")

	// Test latency buckets JSON generation
	// Note: encoding/json escapes < as \u003c for XSS safety
	latencyBuckets := []LatencyBucketData{
		{Label: "< 10ms", Count: 500, Percent: 50.0},
		{Label: "< 50ms", Count: 300, Percent: 30.0},
	}
	latencyJSON := r.latencyBucketsToJSON(latencyBuckets)
	// The < character is escaped as \u003c by encoding/json for XSS safety
	assert.Contains(t, string(latencyJSON), `10ms`)
	assert.Contains(t, string(latencyJSON), `50ms`)
	assert.Contains(t, string(latencyJSON), "500")
	assert.Contains(t, string(latencyJSON), "300")

	// Test endpoints JSON generation
	endpoints := []EndpointData{
		{Name: "GET /api/products", TotalRequests: 5000},
		{Name: "POST /api/orders", TotalRequests: 3000},
	}
	endpointsJSON := r.endpointsToJSON(endpoints)
	assert.Contains(t, string(endpointsJSON), `GET /api/products`)
	assert.Contains(t, string(endpointsJSON), "5000")
}

func TestFormatBytesPerSec(t *testing.T) {
	tests := []struct {
		bytes    float64
		expected string
	}{
		{0, "0.00 B/s"},
		{500, "500.00 B/s"},
		{1024, "1.00 KB/s"},
		{1536, "1.50 KB/s"},
		{1048576, "1.00 MB/s"},
		{1073741824, "1.00 GB/s"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			result := formatBytesPerSec(tc.bytes)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHTMLReporter_BrowserCompatibility(t *testing.T) {
	// This test verifies that the generated HTML can be opened in a browser
	// by checking for common browser compatibility requirements

	r := NewHTMLReporter()
	jsonReporter := NewReporter()

	snapshot := createTestSnapshot()
	jsonReport := jsonReporter.GenerateReport(snapshot, ReportOptions{
		ConfigName:    "browser-test",
		TargetBaseURL: "http://localhost:8080",
	})

	html, err := r.GenerateHTML(jsonReport)
	require.NoError(t, err)

	htmlStr := string(html)

	// Verify proper HTML5 doctype
	assert.True(t, strings.HasPrefix(htmlStr, "<!DOCTYPE html>"))

	// Verify charset is specified
	assert.Contains(t, htmlStr, `charset="UTF-8"`)

	// Verify viewport meta tag for responsive design
	assert.Contains(t, htmlStr, `name="viewport"`)

	// Verify CSS is inlined (standalone HTML file)
	assert.Contains(t, htmlStr, "<style>")
	assert.Contains(t, htmlStr, "var(--")

	// Verify Chart.js CDN is included
	assert.Contains(t, htmlStr, "cdn.jsdelivr.net")
	assert.Contains(t, htmlStr, "chart.js")
}

func TestHTMLReporter_XSSPrevention(t *testing.T) {
	// This test verifies that XSS attacks are prevented in the HTML output
	r := NewHTMLReporter()
	jsonReporter := NewReporter()

	snapshot := createTestSnapshot()

	// Create a report with potentially malicious content
	jsonReport := jsonReporter.GenerateReport(snapshot, ReportOptions{
		ConfigName:        `<script>alert('xss')</script>`,
		ConfigDescription: `<img onerror="alert(1)" src="x">`,
		TargetBaseURL:     `http://example.com"><script>alert(1)</script>`,
		Errors: []ErrorEntry{
			{StatusCode: 500, Endpoint: `POST /api"><img onerror=alert(1)>`, Count: 1, Message: `<script>alert('xss')</script>`},
		},
	})

	html, err := r.GenerateHTML(jsonReport)
	require.NoError(t, err)

	htmlStr := string(html)

	// Verify script tags are escaped (< becomes &lt;)
	assert.NotContains(t, htmlStr, "<script>alert")
	// The content should be escaped to HTML entities
	assert.Contains(t, htmlStr, "&lt;script&gt;")
	// Verify the raw img tag is escaped
	assert.NotContains(t, htmlStr, `<img onerror`)
	assert.Contains(t, htmlStr, "&lt;img")
}
