// Package metrics provides metrics collection and reporting for the load generator.
package metrics

import (
	"maps"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

// Collector aggregates and reports load test metrics.
// It provides comprehensive statistics including:
// - Request counts (total, success, failure)
// - Latency distribution (min, avg, p50, p95, p99, max)
// - Per-endpoint breakdown
// - Real-time QPS calculation
//
// Thread Safety: Safe for concurrent use by multiple goroutines.
type Collector struct {
	mu sync.RWMutex

	// Global counters
	totalRequests   atomic.Int64
	successRequests atomic.Int64
	failedRequests  atomic.Int64
	totalBytes      atomic.Int64

	// Latency tracking (stored as nanoseconds for precision)
	latencies    []int64
	latencyMu    sync.RWMutex // Use RWMutex for read-heavy access pattern
	maxLatencies int          // Maximum latencies to track for percentile calculation

	// Per-endpoint statistics
	endpointStats   map[string]*EndpointStats
	endpointStatsMu sync.RWMutex

	// Status code tracking
	statusCodes   map[int]int64
	statusCodesMu sync.RWMutex

	// Timing
	startTime time.Time
	endTime   time.Time

	// Configuration
	config CollectorConfig
}

// CollectorConfig holds configuration for the metrics collector.
type CollectorConfig struct {
	// MaxLatencies is the maximum number of latency samples to retain
	// for percentile calculations. Default: 100000.
	MaxLatencies int

	// EnableEndpointStats enables per-endpoint statistics.
	// Default: true
	EnableEndpointStats bool
}

// Default configuration values.
const (
	defaultMaxLatencies         = 100000
	defaultEndpointMaxLatencies = 10000
)

// DefaultCollectorConfig returns default configuration.
func DefaultCollectorConfig() CollectorConfig {
	return CollectorConfig{
		MaxLatencies:        defaultMaxLatencies,
		EnableEndpointStats: true,
	}
}

// EndpointStats holds statistics for a single endpoint.
type EndpointStats struct {
	mu sync.RWMutex

	Name             string
	TotalRequests    int64
	SuccessRequests  int64
	FailedRequests   int64
	TotalLatencyNs   int64
	MinLatency       time.Duration
	MaxLatency       time.Duration
	TotalBytes       int64
	latencies        []int64
	maxLatencySample int
}

// Result represents the result of a single request.
type Result struct {
	EndpointName string
	Method       string
	Path         string
	StatusCode   int
	Latency      time.Duration
	Success      bool
	ResponseSize int64
	Timestamp    time.Time
	Error        error
}

// Snapshot represents a point-in-time snapshot of all metrics.
type Snapshot struct {
	// Timing
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration

	// Request counts
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TotalBytes      int64

	// Latency distribution
	MinLatency time.Duration
	AvgLatency time.Duration
	P50Latency time.Duration
	P95Latency time.Duration
	P99Latency time.Duration
	MaxLatency time.Duration

	// Derived metrics
	SuccessRate float64 // 0.0 - 100.0 percentage
	QPS         float64 // Requests per second

	// Status code distribution
	StatusCodes map[int]int64

	// Per-endpoint statistics
	EndpointStats map[string]*EndpointSnapshot
}

// EndpointSnapshot represents a snapshot of endpoint statistics.
type EndpointSnapshot struct {
	Name            string
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TotalBytes      int64
	MinLatency      time.Duration
	AvgLatency      time.Duration
	P50Latency      time.Duration
	P95Latency      time.Duration
	P99Latency      time.Duration
	MaxLatency      time.Duration
	SuccessRate     float64
	QPS             float64
}

// NewCollector creates a new metrics collector.
func NewCollector(config CollectorConfig) *Collector {
	if config.MaxLatencies <= 0 {
		config.MaxLatencies = 100000
	}

	return &Collector{
		latencies:     make([]int64, 0, config.MaxLatencies),
		maxLatencies:  config.MaxLatencies,
		endpointStats: make(map[string]*EndpointStats),
		statusCodes:   make(map[int]int64),
		config:        config,
	}
}

// Start marks the beginning of metrics collection.
func (c *Collector) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.startTime = time.Now()
}

// Stop marks the end of metrics collection.
func (c *Collector) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.endTime = time.Now()
}

// Record records a request result.
func (c *Collector) Record(result Result) {
	// Update global counters
	c.totalRequests.Add(1)
	if result.Success {
		c.successRequests.Add(1)
	} else {
		c.failedRequests.Add(1)
	}
	c.totalBytes.Add(result.ResponseSize)

	// Record latency
	latencyNs := result.Latency.Nanoseconds()
	c.recordLatency(latencyNs)

	// Record status code
	if result.StatusCode > 0 {
		c.recordStatusCode(result.StatusCode)
	}

	// Record endpoint stats if enabled
	if c.config.EnableEndpointStats && result.EndpointName != "" {
		c.recordEndpointResult(result)
	}
}

// recordLatency adds a latency sample using a sliding window approach.
// When capacity is exceeded, older samples are discarded to maintain a view
// of recent performance rather than historical average. This is intentional
// for load testing where current latency distribution is more relevant.
func (c *Collector) recordLatency(latencyNs int64) {
	c.latencyMu.Lock()
	defer c.latencyMu.Unlock()

	// Sliding window: when capacity is exceeded, keep the most recent half
	if len(c.latencies) >= c.maxLatencies {
		if len(c.latencies) > 0 {
			halfSize := c.maxLatencies / 2
			c.latencies = c.latencies[len(c.latencies)-halfSize:]
		}
	}

	c.latencies = append(c.latencies, latencyNs)
}

// recordStatusCode increments the count for a status code.
func (c *Collector) recordStatusCode(code int) {
	c.statusCodesMu.Lock()
	defer c.statusCodesMu.Unlock()
	c.statusCodes[code]++
}

// recordEndpointResult records statistics for a specific endpoint.
func (c *Collector) recordEndpointResult(result Result) {
	c.endpointStatsMu.Lock()
	stats, ok := c.endpointStats[result.EndpointName]
	if !ok {
		stats = &EndpointStats{
			Name:             result.EndpointName,
			latencies:        make([]int64, 0, defaultEndpointMaxLatencies),
			maxLatencySample: defaultEndpointMaxLatencies,
		}
		c.endpointStats[result.EndpointName] = stats
	}
	c.endpointStatsMu.Unlock()

	stats.mu.Lock()
	defer stats.mu.Unlock()

	stats.TotalRequests++
	if result.Success {
		stats.SuccessRequests++
	} else {
		stats.FailedRequests++
	}

	latencyNs := result.Latency.Nanoseconds()
	stats.TotalLatencyNs += latencyNs
	stats.TotalBytes += result.ResponseSize

	// Update min/max
	if stats.MinLatency == 0 || result.Latency < stats.MinLatency {
		stats.MinLatency = result.Latency
	}
	if result.Latency > stats.MaxLatency {
		stats.MaxLatency = result.Latency
	}

	// Record latency sample (with reservoir sampling)
	if len(stats.latencies) >= stats.maxLatencySample {
		halfSize := stats.maxLatencySample / 2
		stats.latencies = stats.latencies[len(stats.latencies)-halfSize:]
	}
	stats.latencies = append(stats.latencies, latencyNs)
}

// Snapshot returns a point-in-time snapshot of all metrics.
func (c *Collector) Snapshot() Snapshot {
	c.mu.RLock()
	startTime := c.startTime
	endTime := c.endTime
	c.mu.RUnlock()

	// Calculate duration
	var duration time.Duration
	if !startTime.IsZero() {
		if endTime.IsZero() {
			duration = time.Since(startTime)
		} else {
			duration = endTime.Sub(startTime)
		}
	}

	// Get current values
	totalRequests := c.totalRequests.Load()
	successRequests := c.successRequests.Load()
	failedRequests := c.failedRequests.Load()
	totalBytes := c.totalBytes.Load()

	// Calculate latency percentiles
	minLat, avgLat, p50Lat, p95Lat, p99Lat, maxLat := c.calculateLatencyStats()

	// Calculate success rate
	var successRate float64
	if totalRequests > 0 {
		successRate = float64(successRequests) / float64(totalRequests) * 100
	}

	// Calculate QPS
	var qps float64
	if duration > 0 {
		qps = float64(totalRequests) / duration.Seconds()
	}

	// Copy status codes
	statusCodes := c.copyStatusCodes()

	// Copy endpoint stats
	endpointStats := c.copyEndpointStats(duration)

	return Snapshot{
		StartTime:       startTime,
		EndTime:         endTime,
		Duration:        duration,
		TotalRequests:   totalRequests,
		SuccessRequests: successRequests,
		FailedRequests:  failedRequests,
		TotalBytes:      totalBytes,
		MinLatency:      minLat,
		AvgLatency:      avgLat,
		P50Latency:      p50Lat,
		P95Latency:      p95Lat,
		P99Latency:      p99Lat,
		MaxLatency:      maxLat,
		SuccessRate:     successRate,
		QPS:             qps,
		StatusCodes:     statusCodes,
		EndpointStats:   endpointStats,
	}
}

// calculateLatencyStats computes latency statistics from collected samples.
func (c *Collector) calculateLatencyStats() (min, avg, p50, p95, p99, max time.Duration) {
	c.latencyMu.RLock()
	// Make a copy to avoid holding the lock during sorting
	latenciesCopy := make([]int64, len(c.latencies))
	copy(latenciesCopy, c.latencies)
	c.latencyMu.RUnlock()

	if len(latenciesCopy) == 0 {
		return 0, 0, 0, 0, 0, 0
	}

	// Sort for percentile calculation
	slices.Sort(latenciesCopy)

	// Calculate statistics
	var sum int64
	for _, lat := range latenciesCopy {
		sum += lat
	}

	n := len(latenciesCopy)
	min = time.Duration(latenciesCopy[0])
	max = time.Duration(latenciesCopy[n-1])
	avg = time.Duration(sum / int64(n))
	p50 = time.Duration(latenciesCopy[percentileIndex(n, 0.50)])
	p95 = time.Duration(latenciesCopy[percentileIndex(n, 0.95)])
	p99 = time.Duration(latenciesCopy[percentileIndex(n, 0.99)])

	return min, avg, p50, p95, p99, max
}

// percentileIndex returns the index for a given percentile.
func percentileIndex(n int, percentile float64) int {
	idx := int(float64(n) * percentile)
	if idx >= n {
		idx = n - 1
	}
	if idx < 0 {
		idx = 0
	}
	return idx
}

// copyStatusCodes creates a copy of the status code map.
func (c *Collector) copyStatusCodes() map[int]int64 {
	c.statusCodesMu.RLock()
	defer c.statusCodesMu.RUnlock()

	result := make(map[int]int64, len(c.statusCodes))
	maps.Copy(result, c.statusCodes)
	return result
}

// copyEndpointStats creates snapshots of all endpoint statistics.
func (c *Collector) copyEndpointStats(totalDuration time.Duration) map[string]*EndpointSnapshot {
	c.endpointStatsMu.RLock()
	defer c.endpointStatsMu.RUnlock()

	result := make(map[string]*EndpointSnapshot, len(c.endpointStats))
	for name, stats := range c.endpointStats {
		result[name] = stats.snapshot(totalDuration)
	}
	return result
}

// snapshot creates a snapshot of endpoint statistics.
func (s *EndpointStats) snapshot(totalDuration time.Duration) *EndpointSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := &EndpointSnapshot{
		Name:            s.Name,
		TotalRequests:   s.TotalRequests,
		SuccessRequests: s.SuccessRequests,
		FailedRequests:  s.FailedRequests,
		TotalBytes:      s.TotalBytes,
		MinLatency:      s.MinLatency,
		MaxLatency:      s.MaxLatency,
	}

	// Calculate average latency
	if s.TotalRequests > 0 {
		snapshot.AvgLatency = time.Duration(s.TotalLatencyNs / s.TotalRequests)
		snapshot.SuccessRate = float64(s.SuccessRequests) / float64(s.TotalRequests) * 100
	}

	// Calculate QPS
	if totalDuration > 0 {
		snapshot.QPS = float64(s.TotalRequests) / totalDuration.Seconds()
	}

	// Calculate percentiles from samples
	if len(s.latencies) > 0 {
		latenciesCopy := make([]int64, len(s.latencies))
		copy(latenciesCopy, s.latencies)
		slices.Sort(latenciesCopy)

		n := len(latenciesCopy)
		snapshot.P50Latency = time.Duration(latenciesCopy[percentileIndex(n, 0.50)])
		snapshot.P95Latency = time.Duration(latenciesCopy[percentileIndex(n, 0.95)])
		snapshot.P99Latency = time.Duration(latenciesCopy[percentileIndex(n, 0.99)])
	}

	return snapshot
}

// Reset clears all collected metrics.
func (c *Collector) Reset() {
	c.mu.Lock()
	c.startTime = time.Time{}
	c.endTime = time.Time{}
	c.mu.Unlock()

	c.totalRequests.Store(0)
	c.successRequests.Store(0)
	c.failedRequests.Store(0)
	c.totalBytes.Store(0)

	c.latencyMu.Lock()
	c.latencies = c.latencies[:0]
	c.latencyMu.Unlock()

	c.statusCodesMu.Lock()
	c.statusCodes = make(map[int]int64)
	c.statusCodesMu.Unlock()

	c.endpointStatsMu.Lock()
	c.endpointStats = make(map[string]*EndpointStats)
	c.endpointStatsMu.Unlock()
}

// GetTotalRequests returns the current total request count.
func (c *Collector) GetTotalRequests() int64 {
	return c.totalRequests.Load()
}

// GetSuccessRequests returns the current successful request count.
func (c *Collector) GetSuccessRequests() int64 {
	return c.successRequests.Load()
}

// GetFailedRequests returns the current failed request count.
func (c *Collector) GetFailedRequests() int64 {
	return c.failedRequests.Load()
}

// GetSuccessRate returns the current success rate (0.0 - 100.0).
func (c *Collector) GetSuccessRate() float64 {
	total := c.totalRequests.Load()
	if total == 0 {
		return 0
	}
	return float64(c.successRequests.Load()) / float64(total) * 100
}

// GetCurrentQPS returns the current requests per second.
func (c *Collector) GetCurrentQPS() float64 {
	c.mu.RLock()
	startTime := c.startTime
	c.mu.RUnlock()

	if startTime.IsZero() {
		return 0
	}

	duration := time.Since(startTime)
	if duration <= 0 {
		return 0
	}

	return float64(c.totalRequests.Load()) / duration.Seconds()
}

// StartTime returns the start time.
func (c *Collector) StartTime() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.startTime
}

// Duration returns the elapsed duration since start.
func (c *Collector) Duration() time.Duration {
	c.mu.RLock()
	startTime := c.startTime
	endTime := c.endTime
	c.mu.RUnlock()

	if startTime.IsZero() {
		return 0
	}
	if endTime.IsZero() {
		return time.Since(startTime)
	}
	return endTime.Sub(startTime)
}
