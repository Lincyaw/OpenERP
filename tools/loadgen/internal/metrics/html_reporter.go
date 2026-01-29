// Package metrics provides metrics collection and reporting for the load generator.
package metrics

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
)

// HTMLReporter generates HTML reports from test metrics.
type HTMLReporter struct {
	version string
}

// NewHTMLReporter creates a new HTMLReporter.
func NewHTMLReporter() *HTMLReporter {
	return &HTMLReporter{
		version: "1.0.0",
	}
}

// HTMLReport represents data for HTML template rendering.
type HTMLReport struct {
	// Report metadata
	Title       string
	Version     string
	GeneratedAt string
	Generator   string

	// Configuration
	ConfigName    string
	ConfigDesc    string
	TargetBaseURL string
	Duration      string
	DurationSec   float64

	// Traffic shaper
	HasTrafficShaper bool
	ShaperType       string
	ShaperBaseQPS    float64

	// Rate limiter
	HasRateLimiter   bool
	RateLimiterType  string
	RateLimiterQPS   float64
	RateLimiterBurst int

	// Worker pool
	HasWorkerPool     bool
	WorkerPoolMinSize int
	WorkerPoolMaxSize int
	WorkerPoolInitial int

	// Warmup
	HasWarmup        bool
	WarmupIterations int
	WarmupFillTypes  string

	// Summary statistics
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TotalBytes      int64
	SuccessRate     float64
	SuccessRateStr  string
	QPS             float64
	QPSStr          string
	BytesPerSec     float64
	BytesPerSecStr  string
	StartTime       string
	EndTime         string

	// Latency statistics
	MinLatencyMs float64
	AvgLatencyMs float64
	P50LatencyMs float64
	P95LatencyMs float64
	P99LatencyMs float64
	MaxLatencyMs float64

	// Status codes for chart
	StatusCodes     []StatusCodeData
	StatusCodesJSON template.JS

	// Latency distribution for chart
	LatencyBuckets     []LatencyBucketData
	LatencyBucketsJSON template.JS

	// Endpoint statistics
	Endpoints     []EndpointData
	EndpointsJSON template.JS

	// Errors
	HasErrors bool
	Errors    []ErrorData

	// Workflows
	HasWorkflows bool
	Workflows    []WorkflowData
}

// StatusCodeData represents status code distribution data.
type StatusCodeData struct {
	Code    string
	Count   int64
	Percent float64
}

// LatencyBucketData represents latency distribution bucket data.
type LatencyBucketData struct {
	Label   string
	Count   int64
	Percent float64
}

// EndpointData represents endpoint statistics for HTML rendering.
type EndpointData struct {
	Name            string
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	SuccessRate     string
	QPS             string
	MinLatencyMs    string
	AvgLatencyMs    string
	P50LatencyMs    string
	P95LatencyMs    string
	P99LatencyMs    string
	MaxLatencyMs    string
}

// ErrorData represents error information for HTML rendering.
type ErrorData struct {
	StatusCode int
	Endpoint   string
	Count      int64
	Message    string
}

// WorkflowData represents workflow configuration for HTML rendering.
type WorkflowData struct {
	Name       string
	Weight     int
	StepsCount int
}

// GenerateHTML generates an HTML report from a JSON report.
func (r *HTMLReporter) GenerateHTML(report *JSONReport) ([]byte, error) {
	htmlData := r.buildHTMLReport(report)

	tmpl, err := template.New("report").Parse(htmlTemplate)
	if err != nil {
		return nil, fmt.Errorf("parsing HTML template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, htmlData); err != nil {
		return nil, fmt.Errorf("executing HTML template: %w", err)
	}

	return buf.Bytes(), nil
}

// WriteHTMLToFile writes an HTML report to a file.
func (r *HTMLReporter) WriteHTMLToFile(report *JSONReport, path string) error {
	// Expand path templates
	expandedPath := expandPathTemplate(path)

	// Clean the path to normalize and prevent path traversal issues
	expandedPath = filepath.Clean(expandedPath)

	// Ensure directory exists
	dir := filepath.Dir(expandedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Generate HTML
	data, err := r.GenerateHTML(report)
	if err != nil {
		return fmt.Errorf("generating HTML report: %w", err)
	}

	// Write to file
	if err := os.WriteFile(expandedPath, data, 0644); err != nil {
		return fmt.Errorf("writing HTML report file: %w", err)
	}

	return nil
}

// buildHTMLReport converts a JSONReport to HTMLReport.
func (r *HTMLReporter) buildHTMLReport(report *JSONReport) *HTMLReport {
	html := &HTMLReport{
		Title:       fmt.Sprintf("Load Test Report - %s", report.Configuration.Name),
		Version:     report.Metadata.Version,
		GeneratedAt: report.Metadata.GeneratedAt.Format("2006-01-02 15:04:05 UTC"),
		Generator:   report.Metadata.Generator,

		ConfigName:    report.Configuration.Name,
		ConfigDesc:    report.Configuration.Description,
		TargetBaseURL: report.Configuration.TargetBaseURL,
		Duration:      formatDurationString(report.Configuration.Duration.Duration),
		DurationSec:   report.Configuration.Duration.Seconds(),

		TotalRequests:   report.Summary.TotalRequests,
		SuccessRequests: report.Summary.SuccessRequests,
		FailedRequests:  report.Summary.FailedRequests,
		TotalBytes:      report.Summary.TotalBytes,
		SuccessRate:     report.Summary.SuccessRate,
		SuccessRateStr:  fmt.Sprintf("%.2f%%", report.Summary.SuccessRate),
		QPS:             report.Summary.QPS,
		QPSStr:          fmt.Sprintf("%.2f", report.Summary.QPS),
		BytesPerSec:     report.Summary.BytesPerSec,
		BytesPerSecStr:  formatBytesPerSec(report.Summary.BytesPerSec),
		StartTime:       report.Summary.StartTime.Format("2006-01-02 15:04:05"),
		EndTime:         report.Summary.EndTime.Format("2006-01-02 15:04:05"),

		MinLatencyMs: report.Summary.Latency.MinMs,
		AvgLatencyMs: report.Summary.Latency.AvgMs,
		P50LatencyMs: report.Summary.Latency.P50Ms,
		P95LatencyMs: report.Summary.Latency.P95Ms,
		P99LatencyMs: report.Summary.Latency.P99Ms,
		MaxLatencyMs: report.Summary.Latency.MaxMs,
	}

	// Traffic shaper
	if report.Configuration.TrafficShaper != nil {
		html.HasTrafficShaper = true
		html.ShaperType = report.Configuration.TrafficShaper.Type
		html.ShaperBaseQPS = report.Configuration.TrafficShaper.BaseQPS
	}

	// Rate limiter
	if report.Configuration.RateLimiter != nil {
		html.HasRateLimiter = true
		html.RateLimiterType = report.Configuration.RateLimiter.Type
		html.RateLimiterQPS = report.Configuration.RateLimiter.QPS
		html.RateLimiterBurst = report.Configuration.RateLimiter.Burst
	}

	// Worker pool
	if report.Configuration.WorkerPool != nil {
		html.HasWorkerPool = true
		html.WorkerPoolMinSize = report.Configuration.WorkerPool.MinSize
		html.WorkerPoolMaxSize = report.Configuration.WorkerPool.MaxSize
		html.WorkerPoolInitial = report.Configuration.WorkerPool.InitialSize
	}

	// Warmup
	if report.Configuration.Warmup != nil && report.Configuration.Warmup.Enabled {
		html.HasWarmup = true
		html.WarmupIterations = report.Configuration.Warmup.Iterations
		if len(report.Configuration.Warmup.FillTypes) > 0 {
			html.WarmupFillTypes = fmt.Sprintf("%v", report.Configuration.Warmup.FillTypes)
		}
	}

	// Status codes
	html.StatusCodes = r.buildStatusCodes(report.StatusCodes, report.Summary.TotalRequests)
	html.StatusCodesJSON = r.statusCodesToJSON(html.StatusCodes)

	// Latency buckets
	html.LatencyBuckets = r.buildLatencyBuckets(report.LatencyDistribution)
	html.LatencyBucketsJSON = r.latencyBucketsToJSON(html.LatencyBuckets)

	// Endpoints
	html.Endpoints = r.buildEndpoints(report.Endpoints)
	html.EndpointsJSON = r.endpointsToJSON(html.Endpoints)

	// Errors
	if len(report.Errors) > 0 {
		html.HasErrors = true
		html.Errors = r.buildErrors(report.Errors)
	}

	// Workflows
	if len(report.Configuration.Workflows) > 0 {
		html.HasWorkflows = true
		html.Workflows = r.buildWorkflows(report.Configuration.Workflows)
	}

	return html
}

// buildStatusCodes converts status codes map to sorted slice.
func (r *HTMLReporter) buildStatusCodes(codes map[string]int64, total int64) []StatusCodeData {
	var data []StatusCodeData
	for code, count := range codes {
		pct := 0.0
		if total > 0 {
			pct = float64(count) / float64(total) * 100
		}
		data = append(data, StatusCodeData{
			Code:    code,
			Count:   count,
			Percent: pct,
		})
	}
	sort.Slice(data, func(i, j int) bool {
		return data[i].Code < data[j].Code
	})
	return data
}

// buildLatencyBuckets converts latency distribution to bucket data.
func (r *HTMLReporter) buildLatencyBuckets(dist LatencyDistribution) []LatencyBucketData {
	var data []LatencyBucketData
	for _, bucket := range dist.Buckets {
		data = append(data, LatencyBucketData{
			Label:   bucket.Label,
			Count:   bucket.Count,
			Percent: bucket.Percent,
		})
	}
	return data
}

// buildEndpoints converts endpoint reports to HTML data.
func (r *HTMLReporter) buildEndpoints(endpoints []EndpointReport) []EndpointData {
	var data []EndpointData
	for _, ep := range endpoints {
		data = append(data, EndpointData{
			Name:            ep.Name,
			TotalRequests:   ep.TotalRequests,
			SuccessRequests: ep.SuccessRequests,
			FailedRequests:  ep.FailedRequests,
			SuccessRate:     fmt.Sprintf("%.2f%%", ep.SuccessRate),
			QPS:             fmt.Sprintf("%.2f", ep.QPS),
			MinLatencyMs:    fmt.Sprintf("%.2f", ep.Latency.MinMs),
			AvgLatencyMs:    fmt.Sprintf("%.2f", ep.Latency.AvgMs),
			P50LatencyMs:    fmt.Sprintf("%.2f", ep.Latency.P50Ms),
			P95LatencyMs:    fmt.Sprintf("%.2f", ep.Latency.P95Ms),
			P99LatencyMs:    fmt.Sprintf("%.2f", ep.Latency.P99Ms),
			MaxLatencyMs:    fmt.Sprintf("%.2f", ep.Latency.MaxMs),
		})
	}
	return data
}

// buildErrors converts error entries to HTML data.
func (r *HTMLReporter) buildErrors(errors []ErrorEntry) []ErrorData {
	var data []ErrorData
	for _, err := range errors {
		data = append(data, ErrorData{
			StatusCode: err.StatusCode,
			Endpoint:   err.Endpoint,
			Count:      err.Count,
			Message:    err.Message,
		})
	}
	return data
}

// buildWorkflows converts workflow configs to HTML data.
func (r *HTMLReporter) buildWorkflows(workflows []WorkflowConfig) []WorkflowData {
	var data []WorkflowData
	for _, wf := range workflows {
		data = append(data, WorkflowData{
			Name:       wf.Name,
			Weight:     wf.Weight,
			StepsCount: wf.StepsCount,
		})
	}
	return data
}

// statusCodesToJSON converts status codes to JSON for chart.
func (r *HTMLReporter) statusCodesToJSON(codes []StatusCodeData) template.JS {
	var labels, counts []string
	for _, code := range codes {
		labels = append(labels, fmt.Sprintf(`"%s"`, code.Code))
		counts = append(counts, fmt.Sprintf("%d", code.Count))
	}
	return template.JS(fmt.Sprintf(`{labels: [%s], data: [%s]}`,
		joinStrings(labels, ","), joinStrings(counts, ",")))
}

// latencyBucketsToJSON converts latency buckets to JSON for chart.
func (r *HTMLReporter) latencyBucketsToJSON(buckets []LatencyBucketData) template.JS {
	var labels, counts []string
	for _, bucket := range buckets {
		labels = append(labels, fmt.Sprintf(`"%s"`, bucket.Label))
		counts = append(counts, fmt.Sprintf("%d", bucket.Count))
	}
	return template.JS(fmt.Sprintf(`{labels: [%s], data: [%s]}`,
		joinStrings(labels, ","), joinStrings(counts, ",")))
}

// endpointsToJSON converts endpoints to JSON for chart.
func (r *HTMLReporter) endpointsToJSON(endpoints []EndpointData) template.JS {
	var labels, requests []string
	for _, ep := range endpoints {
		labels = append(labels, fmt.Sprintf(`"%s"`, ep.Name))
		requests = append(requests, fmt.Sprintf("%d", ep.TotalRequests))
	}
	return template.JS(fmt.Sprintf(`{labels: [%s], data: [%s]}`,
		joinStrings(labels, ","), joinStrings(requests, ",")))
}

// joinStrings joins strings with separator.
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// formatBytesPerSec formats bytes per second to human readable string.
func formatBytesPerSec(bytesPerSec float64) string {
	const unit = 1024
	if bytesPerSec < unit {
		return fmt.Sprintf("%.2f B/s", bytesPerSec)
	}
	div := float64(unit)
	exp := 0
	for n := bytesPerSec / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB/s", bytesPerSec/div, "KMGTPE"[exp])
}

// htmlTemplate is the embedded HTML template with CSS styling.
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.1/dist/chart.umd.min.js"></script>
    <style>
        :root {
            --primary-color: #3b82f6;
            --success-color: #10b981;
            --warning-color: #f59e0b;
            --error-color: #ef4444;
            --bg-color: #f8fafc;
            --card-bg: #ffffff;
            --text-primary: #1e293b;
            --text-secondary: #64748b;
            --border-color: #e2e8f0;
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background-color: var(--bg-color);
            color: var(--text-primary);
            line-height: 1.6;
        }

        .container {
            max-width: 1400px;
            margin: 0 auto;
            padding: 2rem;
        }

        header {
            background: linear-gradient(135deg, var(--primary-color), #1d4ed8);
            color: white;
            padding: 2rem;
            border-radius: 12px;
            margin-bottom: 2rem;
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
        }

        header h1 {
            font-size: 1.875rem;
            font-weight: 700;
            margin-bottom: 0.5rem;
        }

        header .meta {
            opacity: 0.9;
            font-size: 0.875rem;
        }

        .grid {
            display: grid;
            gap: 1.5rem;
        }

        .grid-2 { grid-template-columns: repeat(2, 1fr); }
        .grid-3 { grid-template-columns: repeat(3, 1fr); }
        .grid-4 { grid-template-columns: repeat(4, 1fr); }

        @media (max-width: 1024px) {
            .grid-4 { grid-template-columns: repeat(2, 1fr); }
            .grid-3 { grid-template-columns: repeat(2, 1fr); }
        }

        @media (max-width: 640px) {
            .grid-4, .grid-3, .grid-2 { grid-template-columns: 1fr; }
            .container { padding: 1rem; }
        }

        .card {
            background: var(--card-bg);
            border-radius: 12px;
            padding: 1.5rem;
            box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
            border: 1px solid var(--border-color);
        }

        .card h2 {
            font-size: 1rem;
            font-weight: 600;
            color: var(--text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.05em;
            margin-bottom: 1rem;
            padding-bottom: 0.75rem;
            border-bottom: 1px solid var(--border-color);
        }

        .stat-card {
            text-align: center;
        }

        .stat-card .value {
            font-size: 2rem;
            font-weight: 700;
            color: var(--primary-color);
        }

        .stat-card .label {
            color: var(--text-secondary);
            font-size: 0.875rem;
            margin-top: 0.25rem;
        }

        .stat-card.success .value { color: var(--success-color); }
        .stat-card.warning .value { color: var(--warning-color); }
        .stat-card.error .value { color: var(--error-color); }

        .config-item {
            display: flex;
            justify-content: space-between;
            padding: 0.5rem 0;
            border-bottom: 1px solid var(--border-color);
        }

        .config-item:last-child {
            border-bottom: none;
        }

        .config-item .key {
            color: var(--text-secondary);
            font-weight: 500;
        }

        .config-item .value {
            font-weight: 600;
        }

        .chart-container {
            position: relative;
            height: 300px;
        }

        table {
            width: 100%;
            border-collapse: collapse;
            font-size: 0.875rem;
        }

        th, td {
            padding: 0.75rem;
            text-align: left;
            border-bottom: 1px solid var(--border-color);
        }

        th {
            background: var(--bg-color);
            font-weight: 600;
            color: var(--text-secondary);
            text-transform: uppercase;
            font-size: 0.75rem;
            letter-spacing: 0.05em;
        }

        tr:hover {
            background: var(--bg-color);
        }

        .badge {
            display: inline-block;
            padding: 0.25rem 0.5rem;
            border-radius: 9999px;
            font-size: 0.75rem;
            font-weight: 600;
        }

        .badge-success { background: #d1fae5; color: #065f46; }
        .badge-warning { background: #fef3c7; color: #92400e; }
        .badge-error { background: #fee2e2; color: #991b1b; }

        .section {
            margin-bottom: 2rem;
        }

        .latency-grid {
            display: grid;
            grid-template-columns: repeat(6, 1fr);
            gap: 1rem;
            text-align: center;
        }

        @media (max-width: 768px) {
            .latency-grid { grid-template-columns: repeat(3, 1fr); }
        }

        .latency-item .value {
            font-size: 1.5rem;
            font-weight: 700;
            color: var(--primary-color);
        }

        .latency-item .label {
            font-size: 0.75rem;
            color: var(--text-secondary);
            text-transform: uppercase;
        }

        .footer {
            text-align: center;
            padding: 2rem;
            color: var(--text-secondary);
            font-size: 0.875rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>{{.Title}}</h1>
            <div class="meta">
                Generated: {{.GeneratedAt}} | Version: {{.Version}} | Generator: {{.Generator}}
            </div>
        </header>

        <!-- Summary Stats -->
        <section class="section">
            <div class="grid grid-4">
                <div class="card stat-card">
                    <div class="value">{{.TotalRequests}}</div>
                    <div class="label">Total Requests</div>
                </div>
                <div class="card stat-card success">
                    <div class="value">{{.SuccessRateStr}}</div>
                    <div class="label">Success Rate</div>
                </div>
                <div class="card stat-card">
                    <div class="value">{{.QPSStr}}</div>
                    <div class="label">Requests/sec</div>
                </div>
                <div class="card stat-card">
                    <div class="value">{{.BytesPerSecStr}}</div>
                    <div class="label">Throughput</div>
                </div>
            </div>
        </section>

        <!-- Configuration & Latency -->
        <section class="section">
            <div class="grid grid-2">
                <div class="card">
                    <h2>Configuration</h2>
                    <div class="config-item">
                        <span class="key">Test Name</span>
                        <span class="value">{{.ConfigName}}</span>
                    </div>
                    <div class="config-item">
                        <span class="key">Target URL</span>
                        <span class="value">{{.TargetBaseURL}}</span>
                    </div>
                    <div class="config-item">
                        <span class="key">Duration</span>
                        <span class="value">{{.Duration}}</span>
                    </div>
                    <div class="config-item">
                        <span class="key">Start Time</span>
                        <span class="value">{{.StartTime}}</span>
                    </div>
                    <div class="config-item">
                        <span class="key">End Time</span>
                        <span class="value">{{.EndTime}}</span>
                    </div>
                    {{if .HasTrafficShaper}}
                    <div class="config-item">
                        <span class="key">Traffic Shaper</span>
                        <span class="value">{{.ShaperType}} ({{.ShaperBaseQPS}} QPS)</span>
                    </div>
                    {{end}}
                    {{if .HasRateLimiter}}
                    <div class="config-item">
                        <span class="key">Rate Limiter</span>
                        <span class="value">{{.RateLimiterType}} ({{.RateLimiterQPS}} QPS)</span>
                    </div>
                    {{end}}
                    {{if .HasWorkerPool}}
                    <div class="config-item">
                        <span class="key">Worker Pool</span>
                        <span class="value">{{.WorkerPoolMinSize}}-{{.WorkerPoolMaxSize}} (init: {{.WorkerPoolInitial}})</span>
                    </div>
                    {{end}}
                </div>

                <div class="card">
                    <h2>Latency Distribution</h2>
                    <div class="latency-grid">
                        <div class="latency-item">
                            <div class="value">{{printf "%.1f" .MinLatencyMs}}</div>
                            <div class="label">Min (ms)</div>
                        </div>
                        <div class="latency-item">
                            <div class="value">{{printf "%.1f" .AvgLatencyMs}}</div>
                            <div class="label">Avg (ms)</div>
                        </div>
                        <div class="latency-item">
                            <div class="value">{{printf "%.1f" .P50LatencyMs}}</div>
                            <div class="label">P50 (ms)</div>
                        </div>
                        <div class="latency-item">
                            <div class="value">{{printf "%.1f" .P95LatencyMs}}</div>
                            <div class="label">P95 (ms)</div>
                        </div>
                        <div class="latency-item">
                            <div class="value">{{printf "%.1f" .P99LatencyMs}}</div>
                            <div class="label">P99 (ms)</div>
                        </div>
                        <div class="latency-item">
                            <div class="value">{{printf "%.1f" .MaxLatencyMs}}</div>
                            <div class="label">Max (ms)</div>
                        </div>
                    </div>
                </div>
            </div>
        </section>

        <!-- Charts -->
        <section class="section">
            <div class="grid grid-3">
                <div class="card">
                    <h2>Status Code Distribution</h2>
                    <div class="chart-container">
                        <canvas id="statusCodeChart"></canvas>
                    </div>
                </div>
                <div class="card">
                    <h2>Latency Histogram</h2>
                    <div class="chart-container">
                        <canvas id="latencyChart"></canvas>
                    </div>
                </div>
                <div class="card">
                    <h2>Endpoint Distribution</h2>
                    <div class="chart-container">
                        <canvas id="endpointChart"></canvas>
                    </div>
                </div>
            </div>
        </section>

        <!-- Endpoint Statistics Table -->
        <section class="section">
            <div class="card">
                <h2>Endpoint Statistics</h2>
                <div style="overflow-x: auto;">
                    <table>
                        <thead>
                            <tr>
                                <th>Endpoint</th>
                                <th>Total</th>
                                <th>Success</th>
                                <th>Failed</th>
                                <th>Success Rate</th>
                                <th>QPS</th>
                                <th>P50</th>
                                <th>P95</th>
                                <th>P99</th>
                            </tr>
                        </thead>
                        <tbody>
                            {{range .Endpoints}}
                            <tr>
                                <td><strong>{{.Name}}</strong></td>
                                <td>{{.TotalRequests}}</td>
                                <td>{{.SuccessRequests}}</td>
                                <td>{{.FailedRequests}}</td>
                                <td>{{.SuccessRate}}</td>
                                <td>{{.QPS}}</td>
                                <td>{{.P50LatencyMs}}ms</td>
                                <td>{{.P95LatencyMs}}ms</td>
                                <td>{{.P99LatencyMs}}ms</td>
                            </tr>
                            {{end}}
                        </tbody>
                    </table>
                </div>
            </div>
        </section>

        {{if .HasErrors}}
        <!-- Errors -->
        <section class="section">
            <div class="card">
                <h2>Errors</h2>
                <div style="overflow-x: auto;">
                    <table>
                        <thead>
                            <tr>
                                <th>Status</th>
                                <th>Endpoint</th>
                                <th>Count</th>
                                <th>Message</th>
                            </tr>
                        </thead>
                        <tbody>
                            {{range .Errors}}
                            <tr>
                                <td><span class="badge badge-error">{{.StatusCode}}</span></td>
                                <td>{{.Endpoint}}</td>
                                <td>{{.Count}}</td>
                                <td>{{.Message}}</td>
                            </tr>
                            {{end}}
                        </tbody>
                    </table>
                </div>
            </div>
        </section>
        {{end}}

        {{if .HasWorkflows}}
        <!-- Workflows -->
        <section class="section">
            <div class="card">
                <h2>Workflows</h2>
                <table>
                    <thead>
                        <tr>
                            <th>Name</th>
                            <th>Weight</th>
                            <th>Steps</th>
                        </tr>
                    </thead>
                    <tbody>
                        {{range .Workflows}}
                        <tr>
                            <td><strong>{{.Name}}</strong></td>
                            <td>{{.Weight}}</td>
                            <td>{{.StepsCount}}</td>
                        </tr>
                        {{end}}
                    </tbody>
                </table>
            </div>
        </section>
        {{end}}

        <footer class="footer">
            <p>Generated by {{.Generator}} v{{.Version}}</p>
        </footer>
    </div>

    <script>
        // Chart.js configuration
        const statusCodeData = {{.StatusCodesJSON}};
        const latencyData = {{.LatencyBucketsJSON}};
        const endpointData = {{.EndpointsJSON}};

        const chartColors = [
            '#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6',
            '#ec4899', '#06b6d4', '#84cc16', '#f97316', '#6366f1'
        ];

        // Status Code Chart (Doughnut)
        new Chart(document.getElementById('statusCodeChart'), {
            type: 'doughnut',
            data: {
                labels: statusCodeData.labels,
                datasets: [{
                    data: statusCodeData.data,
                    backgroundColor: chartColors,
                    borderWidth: 0
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        position: 'bottom',
                        labels: {
                            padding: 15,
                            usePointStyle: true
                        }
                    }
                }
            }
        });

        // Latency Histogram (Bar)
        new Chart(document.getElementById('latencyChart'), {
            type: 'bar',
            data: {
                labels: latencyData.labels,
                datasets: [{
                    label: 'Requests',
                    data: latencyData.data,
                    backgroundColor: '#3b82f6',
                    borderRadius: 4
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        display: false
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true,
                        grid: {
                            display: true,
                            color: '#e2e8f0'
                        }
                    },
                    x: {
                        grid: {
                            display: false
                        }
                    }
                }
            }
        });

        // Endpoint Distribution (Horizontal Bar)
        new Chart(document.getElementById('endpointChart'), {
            type: 'bar',
            data: {
                labels: endpointData.labels,
                datasets: [{
                    label: 'Requests',
                    data: endpointData.data,
                    backgroundColor: chartColors,
                    borderRadius: 4
                }]
            },
            options: {
                indexAxis: 'y',
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        display: false
                    }
                },
                scales: {
                    x: {
                        beginAtZero: true,
                        grid: {
                            display: true,
                            color: '#e2e8f0'
                        }
                    },
                    y: {
                        grid: {
                            display: false
                        }
                    }
                }
            }
        });
    </script>
</body>
</html>`
