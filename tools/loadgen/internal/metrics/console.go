// Package metrics provides metrics collection and console reporting for the load generator.
package metrics

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// Console provides real-time console output for load test metrics.
// It displays progress bars, QPS, success rate, latency, and traffic shaping phase.
//
// Thread Safety: Safe for concurrent use.
type Console struct {
	mu sync.Mutex

	writer   io.Writer
	config   ConsoleConfig
	lastLine int // Track lines for clearing

	// State tracking
	isRunning bool
	stopCh    chan struct{}
	doneCh    chan struct{}
}

// ConsoleConfig holds configuration for console output.
type ConsoleConfig struct {
	// Writer is the output destination. Default: os.Stdout
	Writer io.Writer

	// RefreshInterval is how often to update the display. Default: 500ms
	RefreshInterval time.Duration

	// ShowEndpointStats shows per-endpoint breakdown. Default: true
	ShowEndpointStats bool

	// MaxEndpoints limits how many endpoints to show. Default: 10
	MaxEndpoints int

	// ProgressBarWidth is the width of the progress bar. Default: 50
	ProgressBarWidth int

	// ShowTimestamp shows timestamp in output. Default: true
	ShowTimestamp bool

	// UseColors enables ANSI color codes. Default: true
	UseColors bool

	// TotalDuration is the expected test duration (for progress bar).
	// If zero, progress bar shows elapsed time only.
	TotalDuration time.Duration
}

// DefaultConsoleConfig returns default configuration.
func DefaultConsoleConfig() ConsoleConfig {
	return ConsoleConfig{
		Writer:            os.Stdout,
		RefreshInterval:   500 * time.Millisecond,
		ShowEndpointStats: true,
		MaxEndpoints:      10,
		ProgressBarWidth:  50,
		ShowTimestamp:     true,
		UseColors:         true,
	}
}

// ANSI color codes.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

// NewConsole creates a new console output handler.
func NewConsole(config ConsoleConfig) *Console {
	if config.Writer == nil {
		config.Writer = os.Stdout
	}
	if config.RefreshInterval <= 0 {
		config.RefreshInterval = 500 * time.Millisecond
	}
	if config.MaxEndpoints <= 0 {
		config.MaxEndpoints = 10
	}
	if config.ProgressBarWidth <= 0 {
		config.ProgressBarWidth = 50
	}

	return &Console{
		writer: config.Writer,
		config: config,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

// color returns the color code if colors are enabled, empty string otherwise.
func (c *Console) color(code string) string {
	if c.config.UseColors {
		return code
	}
	return ""
}

// Start begins real-time console updates.
func (c *Console) Start(collector *Collector, shaperPhaseFunc func() string) {
	c.mu.Lock()
	if c.isRunning {
		c.mu.Unlock()
		return
	}
	c.isRunning = true
	c.stopCh = make(chan struct{})
	c.doneCh = make(chan struct{})
	c.mu.Unlock()

	go c.updateLoop(collector, shaperPhaseFunc)
}

// Stop stops the console updates.
func (c *Console) Stop() {
	c.mu.Lock()
	if !c.isRunning {
		c.mu.Unlock()
		return
	}
	c.isRunning = false
	close(c.stopCh)
	c.mu.Unlock()

	// Wait for update loop to finish
	<-c.doneCh
}

// updateLoop continuously updates the console display.
func (c *Console) updateLoop(collector *Collector, shaperPhaseFunc func() string) {
	defer close(c.doneCh)

	ticker := time.NewTicker(c.config.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.update(collector, shaperPhaseFunc)
		}
	}
}

// update renders the current metrics to the console.
func (c *Console) update(collector *Collector, shaperPhaseFunc func() string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	snapshot := collector.Snapshot()

	// Clear previous output
	c.clearPreviousOutput()

	var lines []string

	// Header with timestamp
	if c.config.ShowTimestamp {
		lines = append(lines, fmt.Sprintf("%s[%s]%s Load Test Progress",
			c.color(colorDim), time.Now().Format("15:04:05"), c.color(colorReset)))
	}

	// Progress bar
	lines = append(lines, c.formatProgressBar(snapshot.Duration))

	// Traffic phase
	if shaperPhaseFunc != nil {
		phase := shaperPhaseFunc()
		lines = append(lines, fmt.Sprintf("  Phase: %s%s%s",
			c.color(colorCyan), phase, c.color(colorReset)))
	}

	// Main stats line
	lines = append(lines, c.formatMainStats(snapshot))

	// Latency line
	lines = append(lines, c.formatLatencyStats(snapshot))

	// Status code distribution
	lines = append(lines, c.formatStatusCodes(snapshot))

	// Endpoint stats
	if c.config.ShowEndpointStats && len(snapshot.EndpointStats) > 0 {
		lines = append(lines, "")
		lines = append(lines, c.formatEndpointStats(snapshot.EndpointStats)...)
	}

	// Write all lines
	output := strings.Join(lines, "\n") + "\n"
	fmt.Fprint(c.writer, output)
	c.lastLine = len(lines)
}

// clearPreviousOutput clears the previous console output.
func (c *Console) clearPreviousOutput() {
	if c.lastLine > 0 {
		// Move cursor up and clear lines
		for range c.lastLine {
			fmt.Fprint(c.writer, "\033[A\033[K")
		}
	}
}

// formatProgressBar creates a visual progress bar.
func (c *Console) formatProgressBar(elapsed time.Duration) string {
	width := c.config.ProgressBarWidth

	var progress float64
	var barContent string

	if c.config.TotalDuration > 0 {
		// Show progress towards total duration
		progress = float64(elapsed) / float64(c.config.TotalDuration)
		if progress > 1 {
			progress = 1
		}

		filled := int(progress * float64(width))
		empty := width - filled

		barContent = fmt.Sprintf("%s%s%s%s",
			c.color(colorGreen),
			strings.Repeat("█", filled),
			c.color(colorDim)+strings.Repeat("░", empty)+c.color(colorReset),
			"")
	} else {
		// Show elapsed time with animated indicator
		animChars := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
		animIdx := int(elapsed.Milliseconds()/100) % len(animChars)

		// Show running indicator
		barContent = fmt.Sprintf("%s%s%s Running... %s",
			c.color(colorCyan), animChars[animIdx], c.color(colorReset),
			formatDuration(elapsed))
		return fmt.Sprintf("  %s", barContent)
	}

	return fmt.Sprintf("  [%s] %.1f%% (%s / %s)",
		barContent, progress*100, formatDuration(elapsed), formatDuration(c.config.TotalDuration))
}

// formatMainStats formats the main statistics line.
func (c *Console) formatMainStats(snapshot Snapshot) string {
	return fmt.Sprintf("  Requests: %s%d%s | QPS: %s%.1f%s | Success: %s%.1f%%%s (%d/%d)",
		c.color(colorBold), snapshot.TotalRequests, c.color(colorReset),
		c.color(colorBlue), snapshot.QPS, c.color(colorReset),
		c.successRateColor(snapshot.SuccessRate), snapshot.SuccessRate, c.color(colorReset),
		snapshot.SuccessRequests, snapshot.TotalRequests)
}

// formatLatencyStats formats latency statistics.
func (c *Console) formatLatencyStats(snapshot Snapshot) string {
	return fmt.Sprintf("  Latency: min=%s avg=%s p50=%s p95=%s p99=%s max=%s",
		formatLatency(snapshot.MinLatency),
		formatLatency(snapshot.AvgLatency),
		formatLatency(snapshot.P50Latency),
		c.color(colorYellow)+formatLatency(snapshot.P95Latency)+c.color(colorReset),
		c.color(colorRed)+formatLatency(snapshot.P99Latency)+c.color(colorReset),
		formatLatency(snapshot.MaxLatency))
}

// formatStatusCodes formats status code distribution.
func (c *Console) formatStatusCodes(snapshot Snapshot) string {
	if len(snapshot.StatusCodes) == 0 {
		return "  Status: (no responses yet)"
	}

	// Sort status codes
	codes := make([]int, 0, len(snapshot.StatusCodes))
	for code := range snapshot.StatusCodes {
		codes = append(codes, code)
	}
	sort.Ints(codes)

	parts := make([]string, 0, len(codes))
	for _, code := range codes {
		count := snapshot.StatusCodes[code]
		color := c.color(colorGreen)
		if code >= 400 && code < 500 {
			color = c.color(colorYellow)
		} else if code >= 500 {
			color = c.color(colorRed)
		}
		parts = append(parts, fmt.Sprintf("%s%d%s:%d", color, code, c.color(colorReset), count))
	}

	return fmt.Sprintf("  Status: %s", strings.Join(parts, " "))
}

// formatEndpointStats formats per-endpoint statistics.
func (c *Console) formatEndpointStats(stats map[string]*EndpointSnapshot) []string {
	lines := []string{
		fmt.Sprintf("  %sEndpoint Statistics:%s", c.color(colorBold), c.color(colorReset)),
	}

	// Sort endpoints by request count (descending)
	type endpointEntry struct {
		name  string
		stats *EndpointSnapshot
	}

	entries := make([]endpointEntry, 0, len(stats))
	for name, s := range stats {
		entries = append(entries, endpointEntry{name, s})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].stats.TotalRequests > entries[j].stats.TotalRequests
	})

	// Limit endpoints shown
	maxEndpoints := min(c.config.MaxEndpoints, len(entries))

	for i := range maxEndpoints {
		e := entries[i]
		successColor := c.color(colorGreen)
		if e.stats.SuccessRate < 95 {
			successColor = c.color(colorYellow)
		}
		if e.stats.SuccessRate < 80 {
			successColor = c.color(colorRed)
		}

		// Truncate endpoint name if too long
		name := e.name
		if len(name) > 30 {
			name = name[:27] + "..."
		}

		lines = append(lines, fmt.Sprintf("    %-30s reqs=%5d %s%.1f%%%s p95=%s",
			name,
			e.stats.TotalRequests,
			successColor, e.stats.SuccessRate, c.color(colorReset),
			formatLatency(e.stats.P95Latency)))
	}

	if len(entries) > c.config.MaxEndpoints {
		lines = append(lines, fmt.Sprintf("    %s... and %d more endpoints%s",
			c.color(colorDim), len(entries)-c.config.MaxEndpoints, c.color(colorReset)))
	}

	return lines
}

// PrintFinalReport prints a comprehensive final report.
func (c *Console) PrintFinalReport(snapshot Snapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clear any previous real-time output
	c.clearPreviousOutput()
	c.lastLine = 0

	w := c.writer

	// Header
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s╔══════════════════════════════════════════════════════════════╗%s\n",
		c.color(colorBold), c.color(colorReset))
	fmt.Fprintf(w, "%s║               LOAD TEST FINAL REPORT                        ║%s\n",
		c.color(colorBold), c.color(colorReset))
	fmt.Fprintf(w, "%s╚══════════════════════════════════════════════════════════════╝%s\n",
		c.color(colorBold), c.color(colorReset))
	fmt.Fprintln(w)

	// Test Duration
	fmt.Fprintf(w, "%s── Test Duration ─────────────────────────────────────────────%s\n",
		c.color(colorCyan), c.color(colorReset))
	if !snapshot.StartTime.IsZero() {
		fmt.Fprintf(w, "  Start Time:     %s\n", snapshot.StartTime.Format("2006-01-02 15:04:05"))
	}
	if !snapshot.EndTime.IsZero() {
		fmt.Fprintf(w, "  End Time:       %s\n", snapshot.EndTime.Format("2006-01-02 15:04:05"))
	}
	fmt.Fprintf(w, "  Duration:       %s\n", formatDuration(snapshot.Duration))
	fmt.Fprintln(w)

	// Request Statistics
	fmt.Fprintf(w, "%s── Request Statistics ────────────────────────────────────────%s\n",
		c.color(colorCyan), c.color(colorReset))
	fmt.Fprintf(w, "  Total Requests:    %s%d%s\n",
		c.color(colorBold), snapshot.TotalRequests, c.color(colorReset))
	fmt.Fprintf(w, "  Successful:        %s%d%s\n",
		c.color(colorGreen), snapshot.SuccessRequests, c.color(colorReset))
	fmt.Fprintf(w, "  Failed:            %s%d%s\n",
		c.color(colorRed), snapshot.FailedRequests, c.color(colorReset))
	fmt.Fprintf(w, "  Success Rate:      %s%.2f%%%s\n",
		c.successRateColor(snapshot.SuccessRate), snapshot.SuccessRate, c.color(colorReset))
	fmt.Fprintf(w, "  Throughput:        %s%.2f req/s%s\n",
		c.color(colorBlue), snapshot.QPS, c.color(colorReset))
	fmt.Fprintf(w, "  Data Transferred:  %s\n", formatBytes(snapshot.TotalBytes))
	fmt.Fprintln(w)

	// Latency Distribution
	fmt.Fprintf(w, "%s── Latency Distribution ──────────────────────────────────────%s\n",
		c.color(colorCyan), c.color(colorReset))
	fmt.Fprintf(w, "  Min:    %12s\n", formatLatency(snapshot.MinLatency))
	fmt.Fprintf(w, "  Avg:    %12s\n", formatLatency(snapshot.AvgLatency))
	fmt.Fprintf(w, "  P50:    %12s\n", formatLatency(snapshot.P50Latency))
	fmt.Fprintf(w, "  P95:    %12s  %s(target < 500ms)%s\n",
		formatLatency(snapshot.P95Latency), c.color(colorDim), c.color(colorReset))
	fmt.Fprintf(w, "  P99:    %12s  %s(target < 1s)%s\n",
		formatLatency(snapshot.P99Latency), c.color(colorDim), c.color(colorReset))
	fmt.Fprintf(w, "  Max:    %12s\n", formatLatency(snapshot.MaxLatency))
	fmt.Fprintln(w)

	// Latency histogram (visual)
	fmt.Fprintln(w, c.formatLatencyHistogram(snapshot))

	// Status Code Distribution
	if len(snapshot.StatusCodes) > 0 {
		fmt.Fprintf(w, "%s── Status Code Distribution ──────────────────────────────────%s\n",
			c.color(colorCyan), c.color(colorReset))

		codes := make([]int, 0, len(snapshot.StatusCodes))
		for code := range snapshot.StatusCodes {
			codes = append(codes, code)
		}
		sort.Ints(codes)

		for _, code := range codes {
			count := snapshot.StatusCodes[code]
			pct := float64(count) / float64(snapshot.TotalRequests) * 100
			color := c.statusCodeColor(code)

			// Visual bar
			barWidth := int(pct / 2)
			if barWidth < 1 && count > 0 {
				barWidth = 1
			}

			fmt.Fprintf(w, "  %s%d%s: %6d (%5.1f%%) %s%s%s\n",
				color, code, c.color(colorReset),
				count, pct,
				color, strings.Repeat("█", barWidth), c.color(colorReset))
		}
		fmt.Fprintln(w)
	}

	// Per-Endpoint Statistics
	if len(snapshot.EndpointStats) > 0 {
		fmt.Fprintf(w, "%s── Per-Endpoint Statistics ───────────────────────────────────%s\n",
			c.color(colorCyan), c.color(colorReset))

		// Sort by request count
		type endpointEntry struct {
			name  string
			stats *EndpointSnapshot
		}

		entries := make([]endpointEntry, 0, len(snapshot.EndpointStats))
		for name, s := range snapshot.EndpointStats {
			entries = append(entries, endpointEntry{name, s})
		}

		sort.Slice(entries, func(i, j int) bool {
			return entries[i].stats.TotalRequests > entries[j].stats.TotalRequests
		})

		// Header
		fmt.Fprintf(w, "  %-35s %8s %8s %8s %10s\n",
			"Endpoint", "Requests", "Success%", "P95", "Avg")
		fmt.Fprintf(w, "  %s%s%s\n",
			c.color(colorDim), strings.Repeat("─", 70), c.color(colorReset))

		for _, e := range entries {
			name := e.name
			if len(name) > 35 {
				name = name[:32] + "..."
			}

			fmt.Fprintf(w, "  %-35s %8d %s%7.1f%%%s %8s %10s\n",
				name,
				e.stats.TotalRequests,
				c.successRateColor(e.stats.SuccessRate), e.stats.SuccessRate, c.color(colorReset),
				formatLatency(e.stats.P95Latency),
				formatLatency(e.stats.AvgLatency))
		}
		fmt.Fprintln(w)
	}

	// Footer
	fmt.Fprintf(w, "%s══════════════════════════════════════════════════════════════%s\n",
		c.color(colorDim), c.color(colorReset))
	fmt.Fprintln(w)
}

// formatLatencyHistogram creates a visual latency histogram.
func (c *Console) formatLatencyHistogram(snapshot Snapshot) string {
	if snapshot.TotalRequests == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %s── Latency Histogram ─────────────────────────────────────────%s\n",
		c.color(colorCyan), c.color(colorReset)))

	// Simple bucket visualization
	buckets := []struct {
		label string
		max   time.Duration
	}{
		{"< 10ms", 10 * time.Millisecond},
		{"< 50ms", 50 * time.Millisecond},
		{"< 100ms", 100 * time.Millisecond},
		{"< 500ms", 500 * time.Millisecond},
		{"< 1s", time.Second},
		{">= 1s", time.Hour * 24}, // Effectively infinity
	}

	// Estimate distribution based on percentiles
	// This is approximate since we only have percentile data
	latencies := []time.Duration{snapshot.MinLatency, snapshot.P50Latency, snapshot.P95Latency, snapshot.P99Latency, snapshot.MaxLatency}

	for _, bucket := range buckets {
		count := 0
		for _, lat := range latencies {
			if lat <= bucket.max {
				count++
			}
		}
		pct := float64(count) / float64(len(latencies)) * 100

		barWidth := int(pct / 2)
		if barWidth < 1 && count > 0 {
			barWidth = 1
		}

		color := c.color(colorGreen)
		if bucket.max > 100*time.Millisecond {
			color = c.color(colorYellow)
		}
		if bucket.max > 500*time.Millisecond {
			color = c.color(colorRed)
		}

		sb.WriteString(fmt.Sprintf("  %8s: %s%s%s\n",
			bucket.label, color, strings.Repeat("█", barWidth), c.color(colorReset)))
	}

	return sb.String()
}

// Success rate thresholds for color coding.
const (
	successRateExcellent = 99.0 // Green: >= 99%
	successRateGood      = 95.0 // Yellow: >= 95%, < 99%
	// Red: < 95%
)

// successRateColor returns the appropriate color for a success rate.
func (c *Console) successRateColor(rate float64) string {
	switch {
	case rate >= successRateExcellent:
		return c.color(colorGreen)
	case rate >= successRateGood:
		return c.color(colorYellow)
	default:
		return c.color(colorRed)
	}
}

// statusCodeColor returns the appropriate color for a status code.
func (c *Console) statusCodeColor(code int) string {
	switch {
	case code >= 200 && code < 300:
		return c.color(colorGreen)
	case code >= 300 && code < 400:
		return c.color(colorBlue)
	case code >= 400 && code < 500:
		return c.color(colorYellow)
	default:
		return c.color(colorRed)
	}
}

// formatLatency formats a duration for display.
func formatLatency(d time.Duration) string {
	if d == 0 {
		return "0ms"
	}
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.2fµs", float64(d.Nanoseconds())/1000)
	}
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1e6)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// formatDuration formats a duration for display.
func formatDuration(d time.Duration) string {
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

// formatBytes formats a byte count for display.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
