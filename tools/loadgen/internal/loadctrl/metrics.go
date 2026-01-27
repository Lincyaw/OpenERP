// Package loadctrl provides load control components including traffic shaping
// and rate limiting for the load generator.
package loadctrl

import (
	"slices"
	"sync"
	"time"
)

// MetricsCollector defines the interface for collecting and reporting metrics.
// The LoadController uses this to get latency information for adaptive control.
//
// Thread Safety: Implementations must be safe for concurrent use.
type MetricsCollector interface {
	// RecordLatency records a latency measurement.
	RecordLatency(latency time.Duration)

	// RecordError records an error occurrence.
	RecordError()

	// RecordSuccess records a successful request.
	RecordSuccess()

	// GetP95Latency returns the 95th percentile latency over the window.
	GetP95Latency() time.Duration

	// GetP99Latency returns the 99th percentile latency over the window.
	GetP99Latency() time.Duration

	// GetAvgLatency returns the average latency over the window.
	GetAvgLatency() time.Duration

	// GetErrorRate returns the error rate (0.0 - 1.0) over the window.
	GetErrorRate() float64

	// GetStats returns a snapshot of all collected metrics.
	GetStats() MetricsStats

	// Reset clears all collected metrics.
	Reset()
}

// MetricsStats contains a snapshot of collected metrics.
type MetricsStats struct {
	// TotalRequests is the total number of requests recorded.
	TotalRequests int64
	// TotalErrors is the total number of errors recorded.
	TotalErrors int64
	// AvgLatency is the average latency.
	AvgLatency time.Duration
	// P50Latency is the 50th percentile latency.
	P50Latency time.Duration
	// P95Latency is the 95th percentile latency.
	P95Latency time.Duration
	// P99Latency is the 99th percentile latency.
	P99Latency time.Duration
	// MinLatency is the minimum latency recorded.
	MinLatency time.Duration
	// MaxLatency is the maximum latency recorded.
	MaxLatency time.Duration
	// ErrorRate is the error rate (0.0 - 1.0).
	ErrorRate float64
	// RequestsPerSecond is the calculated QPS.
	RequestsPerSecond float64
}

// SlidingWindowMetrics implements MetricsCollector using a sliding time window.
// It maintains latency measurements within a configurable time window.
//
// Thread Safety: Safe for concurrent use.
type SlidingWindowMetrics struct {
	windowSize   time.Duration
	bucketSize   time.Duration
	buckets      []*metricsBucket
	bucketCount  int
	currentIdx   int
	lastRotation time.Time
	mu           sync.RWMutex

	// Aggregate counters
	totalRequests int64
	totalErrors   int64
}

// metricsBucket holds metrics for a single time bucket.
type metricsBucket struct {
	latencies    []time.Duration
	errorCount   int64
	successCount int64
	timestamp    time.Time
}

// MetricsConfig holds configuration for creating a metrics collector.
type MetricsConfig struct {
	// WindowSize is the duration of the sliding window (default: 10s).
	WindowSize time.Duration `yaml:"windowSize" json:"windowSize"`
	// BucketSize is the duration of each bucket (default: 1s).
	BucketSize time.Duration `yaml:"bucketSize" json:"bucketSize"`
}

// NewSlidingWindowMetrics creates a new sliding window metrics collector.
func NewSlidingWindowMetrics(config MetricsConfig) *SlidingWindowMetrics {
	if config.WindowSize <= 0 {
		config.WindowSize = 10 * time.Second
	}
	if config.BucketSize <= 0 {
		config.BucketSize = time.Second
	}
	if config.BucketSize > config.WindowSize {
		config.BucketSize = config.WindowSize
	}

	bucketCount := max(1, int(config.WindowSize/config.BucketSize))

	buckets := make([]*metricsBucket, bucketCount)
	now := time.Now()
	for i := range buckets {
		buckets[i] = &metricsBucket{
			latencies: make([]time.Duration, 0, 100),
			timestamp: now,
		}
	}

	return &SlidingWindowMetrics{
		windowSize:   config.WindowSize,
		bucketSize:   config.BucketSize,
		buckets:      buckets,
		bucketCount:  bucketCount,
		currentIdx:   0,
		lastRotation: now,
	}
}

// RecordLatency records a latency measurement.
func (m *SlidingWindowMetrics) RecordLatency(latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.rotateBucketsLocked()
	bucket := m.buckets[m.currentIdx]
	bucket.latencies = append(bucket.latencies, latency)
	bucket.successCount++
	m.totalRequests++
}

// RecordError records an error occurrence.
func (m *SlidingWindowMetrics) RecordError() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.rotateBucketsLocked()
	bucket := m.buckets[m.currentIdx]
	bucket.errorCount++
	m.totalErrors++
	m.totalRequests++
}

// RecordSuccess records a successful request without latency.
func (m *SlidingWindowMetrics) RecordSuccess() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.rotateBucketsLocked()
	bucket := m.buckets[m.currentIdx]
	bucket.successCount++
	m.totalRequests++
}

// GetP95Latency returns the 95th percentile latency over the window.
func (m *SlidingWindowMetrics) GetP95Latency() time.Duration {
	return m.getPercentileLatency(0.95)
}

// GetP99Latency returns the 99th percentile latency over the window.
func (m *SlidingWindowMetrics) GetP99Latency() time.Duration {
	return m.getPercentileLatency(0.99)
}

// GetAvgLatency returns the average latency over the window.
func (m *SlidingWindowMetrics) GetAvgLatency() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var total time.Duration
	var count int64

	cutoff := time.Now().Add(-m.windowSize)
	for _, bucket := range m.buckets {
		if bucket.timestamp.Before(cutoff) {
			continue
		}
		for _, lat := range bucket.latencies {
			total += lat
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return time.Duration(int64(total) / count)
}

// GetErrorRate returns the error rate (0.0 - 1.0) over the window.
func (m *SlidingWindowMetrics) GetErrorRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var totalReqs, totalErrs int64
	cutoff := time.Now().Add(-m.windowSize)

	for _, bucket := range m.buckets {
		if bucket.timestamp.Before(cutoff) {
			continue
		}
		totalReqs += bucket.successCount + bucket.errorCount
		totalErrs += bucket.errorCount
	}

	if totalReqs == 0 {
		return 0
	}
	return float64(totalErrs) / float64(totalReqs)
}

// GetStats returns a snapshot of all collected metrics.
func (m *SlidingWindowMetrics) GetStats() MetricsStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Collect all latencies from valid buckets
	var allLatencies []time.Duration
	var totalReqs, totalErrs int64
	cutoff := time.Now().Add(-m.windowSize)
	var minTime, maxTime time.Time

	for _, bucket := range m.buckets {
		if bucket.timestamp.Before(cutoff) {
			continue
		}
		allLatencies = append(allLatencies, bucket.latencies...)
		totalReqs += bucket.successCount + bucket.errorCount
		totalErrs += bucket.errorCount

		if minTime.IsZero() || bucket.timestamp.Before(minTime) {
			minTime = bucket.timestamp
		}
		if maxTime.IsZero() || bucket.timestamp.After(maxTime) {
			maxTime = bucket.timestamp
		}
	}

	stats := MetricsStats{
		TotalRequests: m.totalRequests,
		TotalErrors:   m.totalErrors,
	}

	if totalReqs > 0 {
		stats.ErrorRate = float64(totalErrs) / float64(totalReqs)
	}

	if len(allLatencies) > 0 {
		// Sort for percentile calculation
		slices.Sort(allLatencies)

		// Calculate statistics
		var sum time.Duration
		for _, lat := range allLatencies {
			sum += lat
		}
		stats.AvgLatency = time.Duration(int64(sum) / int64(len(allLatencies)))
		stats.MinLatency = allLatencies[0]
		stats.MaxLatency = allLatencies[len(allLatencies)-1]
		stats.P50Latency = allLatencies[int(float64(len(allLatencies))*0.5)]
		stats.P95Latency = allLatencies[int(float64(len(allLatencies))*0.95)]
		stats.P99Latency = allLatencies[max(0, int(float64(len(allLatencies))*0.99)-1)]
	}

	// Calculate RPS
	if !minTime.IsZero() && !maxTime.IsZero() && maxTime.After(minTime) {
		duration := maxTime.Sub(minTime)
		if duration > 0 {
			stats.RequestsPerSecond = float64(totalReqs) / duration.Seconds()
		}
	}

	return stats
}

// Reset clears all collected metrics.
func (m *SlidingWindowMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for i := range m.buckets {
		m.buckets[i] = &metricsBucket{
			latencies: make([]time.Duration, 0, 100),
			timestamp: now,
		}
	}
	m.currentIdx = 0
	m.lastRotation = now
	m.totalRequests = 0
	m.totalErrors = 0
}

// getPercentileLatency returns the latency at the given percentile (0.0 - 1.0).
func (m *SlidingWindowMetrics) getPercentileLatency(percentile float64) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Collect all latencies from valid buckets
	var allLatencies []time.Duration
	cutoff := time.Now().Add(-m.windowSize)

	for _, bucket := range m.buckets {
		if bucket.timestamp.Before(cutoff) {
			continue
		}
		allLatencies = append(allLatencies, bucket.latencies...)
	}

	if len(allLatencies) == 0 {
		return 0
	}

	// Sort for percentile calculation
	slices.Sort(allLatencies)

	idx := int(float64(len(allLatencies)) * percentile)
	if idx >= len(allLatencies) {
		idx = len(allLatencies) - 1
	}
	return allLatencies[idx]
}

// rotateBucketsLocked rotates buckets if needed. Must be called with lock held.
func (m *SlidingWindowMetrics) rotateBucketsLocked() {
	now := time.Now()
	elapsed := now.Sub(m.lastRotation)

	// Determine how many buckets to rotate
	bucketsToRotate := int(elapsed / m.bucketSize)
	if bucketsToRotate <= 0 {
		return
	}

	// Rotate buckets
	for range min(bucketsToRotate, m.bucketCount) {
		m.currentIdx = (m.currentIdx + 1) % m.bucketCount
		// Clear the new current bucket
		m.buckets[m.currentIdx] = &metricsBucket{
			latencies: make([]time.Duration, 0, 100),
			timestamp: now,
		}
	}

	m.lastRotation = now
}

// NewMetricsCollector creates a new metrics collector with default configuration.
func NewMetricsCollector() MetricsCollector {
	return NewSlidingWindowMetrics(MetricsConfig{
		WindowSize: 10 * time.Second,
		BucketSize: time.Second,
	})
}
