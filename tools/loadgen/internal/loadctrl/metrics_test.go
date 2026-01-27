package loadctrl

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSlidingWindowMetrics_BasicOperation(t *testing.T) {
	metrics := NewSlidingWindowMetrics(MetricsConfig{
		WindowSize: 10 * time.Second,
		BucketSize: time.Second,
	})

	// Record some latencies
	metrics.RecordLatency(10 * time.Millisecond)
	metrics.RecordLatency(20 * time.Millisecond)
	metrics.RecordLatency(30 * time.Millisecond)

	// Check average
	avgLatency := metrics.GetAvgLatency()
	assert.Equal(t, 20*time.Millisecond, avgLatency)

	// Check stats
	stats := metrics.GetStats()
	assert.Equal(t, int64(3), stats.TotalRequests)
	assert.Equal(t, int64(0), stats.TotalErrors)
}

func TestSlidingWindowMetrics_ErrorRate(t *testing.T) {
	metrics := NewSlidingWindowMetrics(MetricsConfig{
		WindowSize: 10 * time.Second,
		BucketSize: time.Second,
	})

	// Record 7 successes and 3 errors (30% error rate)
	for range 7 {
		metrics.RecordLatency(10 * time.Millisecond)
	}
	for range 3 {
		metrics.RecordError()
	}

	errorRate := metrics.GetErrorRate()
	assert.InDelta(t, 0.3, errorRate, 0.01)
}

func TestSlidingWindowMetrics_Percentiles(t *testing.T) {
	metrics := NewSlidingWindowMetrics(MetricsConfig{
		WindowSize: 10 * time.Second,
		BucketSize: time.Second,
	})

	// Record latencies from 1ms to 100ms
	for i := 1; i <= 100; i++ {
		metrics.RecordLatency(time.Duration(i) * time.Millisecond)
	}

	p95 := metrics.GetP95Latency()
	p99 := metrics.GetP99Latency()
	avg := metrics.GetAvgLatency()

	// P95 should be around 95ms
	assert.InDelta(t, 95*time.Millisecond, p95, float64(5*time.Millisecond))

	// P99 should be around 99ms
	assert.InDelta(t, 99*time.Millisecond, p99, float64(5*time.Millisecond))

	// Average should be around 50ms
	assert.InDelta(t, 50*time.Millisecond, avg, float64(5*time.Millisecond))
}

func TestSlidingWindowMetrics_Reset(t *testing.T) {
	metrics := NewSlidingWindowMetrics(MetricsConfig{
		WindowSize: 10 * time.Second,
		BucketSize: time.Second,
	})

	// Record some data
	metrics.RecordLatency(10 * time.Millisecond)
	metrics.RecordError()

	// Reset
	metrics.Reset()

	// Should be empty
	stats := metrics.GetStats()
	assert.Equal(t, int64(0), stats.TotalRequests)
	assert.Equal(t, int64(0), stats.TotalErrors)
	assert.Equal(t, time.Duration(0), stats.AvgLatency)
}

func TestSlidingWindowMetrics_EmptyMetrics(t *testing.T) {
	metrics := NewSlidingWindowMetrics(MetricsConfig{
		WindowSize: 10 * time.Second,
		BucketSize: time.Second,
	})

	// Empty metrics should return zero values
	assert.Equal(t, time.Duration(0), metrics.GetAvgLatency())
	assert.Equal(t, time.Duration(0), metrics.GetP95Latency())
	assert.Equal(t, time.Duration(0), metrics.GetP99Latency())
	assert.Equal(t, 0.0, metrics.GetErrorRate())
}

func TestSlidingWindowMetrics_Stats(t *testing.T) {
	metrics := NewSlidingWindowMetrics(MetricsConfig{
		WindowSize: 10 * time.Second,
		BucketSize: time.Second,
	})

	// Record varied latencies
	latencies := []time.Duration{
		5 * time.Millisecond,
		10 * time.Millisecond,
		15 * time.Millisecond,
		20 * time.Millisecond,
		100 * time.Millisecond, // outlier
	}

	for _, lat := range latencies {
		metrics.RecordLatency(lat)
	}

	stats := metrics.GetStats()
	assert.Equal(t, int64(5), stats.TotalRequests)
	assert.Equal(t, 5*time.Millisecond, stats.MinLatency)
	assert.Equal(t, 100*time.Millisecond, stats.MaxLatency)
}

func TestNewMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector()
	assert.NotNil(t, collector)

	// Should work with default config
	collector.RecordLatency(10 * time.Millisecond)
	stats := collector.GetStats()
	assert.Equal(t, int64(1), stats.TotalRequests)
}

func TestSlidingWindowMetrics_RecordSuccess(t *testing.T) {
	metrics := NewSlidingWindowMetrics(MetricsConfig{
		WindowSize: 10 * time.Second,
		BucketSize: time.Second,
	})

	// Record success without latency
	metrics.RecordSuccess()
	metrics.RecordSuccess()
	metrics.RecordSuccess()

	stats := metrics.GetStats()
	assert.Equal(t, int64(3), stats.TotalRequests)
	assert.Equal(t, int64(0), stats.TotalErrors)
	// No latencies recorded
	assert.Equal(t, time.Duration(0), stats.AvgLatency)
}
