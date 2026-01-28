package metrics

import (
	"bytes"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCollector(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		c := NewCollector(CollectorConfig{})
		require.NotNil(t, c)
		assert.Equal(t, 100000, c.maxLatencies)
	})

	t.Run("custom config", func(t *testing.T) {
		c := NewCollector(CollectorConfig{
			MaxLatencies:        50000,
			EnableEndpointStats: true,
		})
		require.NotNil(t, c)
		assert.Equal(t, 50000, c.maxLatencies)
	})
}

func TestCollector_Record(t *testing.T) {
	t.Run("success request", func(t *testing.T) {
		c := NewCollector(DefaultCollectorConfig())
		c.Start()

		c.Record(Result{
			EndpointName: "test-endpoint",
			Method:       "GET",
			Path:         "/api/test",
			StatusCode:   200,
			Latency:      100 * time.Millisecond,
			Success:      true,
			ResponseSize: 1024,
			Timestamp:    time.Now(),
		})

		assert.Equal(t, int64(1), c.GetTotalRequests())
		assert.Equal(t, int64(1), c.GetSuccessRequests())
		assert.Equal(t, int64(0), c.GetFailedRequests())
		assert.Equal(t, 100.0, c.GetSuccessRate())
	})

	t.Run("failed request", func(t *testing.T) {
		c := NewCollector(DefaultCollectorConfig())
		c.Start()

		c.Record(Result{
			EndpointName: "test-endpoint",
			Method:       "GET",
			Path:         "/api/test",
			StatusCode:   500,
			Latency:      50 * time.Millisecond,
			Success:      false,
			ResponseSize: 256,
			Timestamp:    time.Now(),
		})

		assert.Equal(t, int64(1), c.GetTotalRequests())
		assert.Equal(t, int64(0), c.GetSuccessRequests())
		assert.Equal(t, int64(1), c.GetFailedRequests())
		assert.Equal(t, 0.0, c.GetSuccessRate())
	})

	t.Run("mixed requests", func(t *testing.T) {
		c := NewCollector(DefaultCollectorConfig())
		c.Start()

		// Record 8 successes and 2 failures = 80% success rate
		for range 8 {
			c.Record(Result{Success: true, StatusCode: 200, Latency: 10 * time.Millisecond})
		}
		for range 2 {
			c.Record(Result{Success: false, StatusCode: 500, Latency: 50 * time.Millisecond})
		}

		assert.Equal(t, int64(10), c.GetTotalRequests())
		assert.Equal(t, int64(8), c.GetSuccessRequests())
		assert.Equal(t, int64(2), c.GetFailedRequests())
		assert.Equal(t, 80.0, c.GetSuccessRate())
	})
}

func TestCollector_Snapshot(t *testing.T) {
	t.Run("basic snapshot", func(t *testing.T) {
		c := NewCollector(DefaultCollectorConfig())
		c.Start()

		// Record varied latencies
		latencies := []time.Duration{
			10 * time.Millisecond,
			20 * time.Millisecond,
			30 * time.Millisecond,
			40 * time.Millisecond,
			50 * time.Millisecond,
			100 * time.Millisecond,
			200 * time.Millisecond,
			300 * time.Millisecond,
			400 * time.Millisecond,
			500 * time.Millisecond,
		}

		for _, lat := range latencies {
			c.Record(Result{
				Success:      true,
				StatusCode:   200,
				Latency:      lat,
				ResponseSize: 100,
			})
		}

		snapshot := c.Snapshot()

		assert.Equal(t, int64(10), snapshot.TotalRequests)
		assert.Equal(t, int64(10), snapshot.SuccessRequests)
		assert.Equal(t, int64(0), snapshot.FailedRequests)
		assert.Equal(t, 100.0, snapshot.SuccessRate)

		// Verify latency stats
		assert.Equal(t, 10*time.Millisecond, snapshot.MinLatency)
		assert.Equal(t, 500*time.Millisecond, snapshot.MaxLatency)
		assert.True(t, snapshot.AvgLatency > 0)
		assert.True(t, snapshot.P50Latency > 0)
		assert.True(t, snapshot.P95Latency > 0)
		assert.True(t, snapshot.P99Latency > 0)

		// Verify percentiles are in order
		assert.True(t, snapshot.MinLatency <= snapshot.P50Latency)
		assert.True(t, snapshot.P50Latency <= snapshot.P95Latency)
		assert.True(t, snapshot.P95Latency <= snapshot.P99Latency)
		assert.True(t, snapshot.P99Latency <= snapshot.MaxLatency)
	})

	t.Run("status code distribution", func(t *testing.T) {
		c := NewCollector(DefaultCollectorConfig())
		c.Start()

		// Record various status codes
		for range 5 {
			c.Record(Result{Success: true, StatusCode: 200, Latency: 10 * time.Millisecond})
		}
		for range 3 {
			c.Record(Result{Success: true, StatusCode: 201, Latency: 10 * time.Millisecond})
		}
		for range 2 {
			c.Record(Result{Success: false, StatusCode: 500, Latency: 10 * time.Millisecond})
		}

		snapshot := c.Snapshot()

		assert.Equal(t, int64(5), snapshot.StatusCodes[200])
		assert.Equal(t, int64(3), snapshot.StatusCodes[201])
		assert.Equal(t, int64(2), snapshot.StatusCodes[500])
	})

	t.Run("endpoint statistics", func(t *testing.T) {
		c := NewCollector(DefaultCollectorConfig())
		c.Start()

		// Record for different endpoints
		for range 5 {
			c.Record(Result{
				EndpointName: "endpoint-a",
				Success:      true,
				StatusCode:   200,
				Latency:      50 * time.Millisecond,
			})
		}
		for range 3 {
			c.Record(Result{
				EndpointName: "endpoint-b",
				Success:      true,
				StatusCode:   200,
				Latency:      100 * time.Millisecond,
			})
		}
		c.Record(Result{
			EndpointName: "endpoint-b",
			Success:      false,
			StatusCode:   500,
			Latency:      200 * time.Millisecond,
		})

		snapshot := c.Snapshot()

		// Check endpoint-a
		epA := snapshot.EndpointStats["endpoint-a"]
		require.NotNil(t, epA)
		assert.Equal(t, int64(5), epA.TotalRequests)
		assert.Equal(t, int64(5), epA.SuccessRequests)
		assert.Equal(t, 100.0, epA.SuccessRate)

		// Check endpoint-b
		epB := snapshot.EndpointStats["endpoint-b"]
		require.NotNil(t, epB)
		assert.Equal(t, int64(4), epB.TotalRequests)
		assert.Equal(t, int64(3), epB.SuccessRequests)
		assert.Equal(t, int64(1), epB.FailedRequests)
		assert.Equal(t, 75.0, epB.SuccessRate)
	})
}

func TestCollector_Concurrent(t *testing.T) {
	c := NewCollector(DefaultCollectorConfig())
	c.Start()

	numGoroutines := 10
	recordsPerGoroutine := 1000

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()
			for range recordsPerGoroutine {
				c.Record(Result{
					EndpointName: "concurrent-test",
					Success:      true,
					StatusCode:   200,
					Latency:      10 * time.Millisecond,
					ResponseSize: 100,
				})
			}
		}()
	}

	wg.Wait()

	expectedTotal := int64(numGoroutines * recordsPerGoroutine)
	assert.Equal(t, expectedTotal, c.GetTotalRequests())
	assert.Equal(t, expectedTotal, c.GetSuccessRequests())
}

func TestCollector_Reset(t *testing.T) {
	c := NewCollector(DefaultCollectorConfig())
	c.Start()

	// Record some data
	for range 10 {
		c.Record(Result{
			EndpointName: "test-endpoint",
			Success:      true,
			StatusCode:   200,
			Latency:      10 * time.Millisecond,
		})
	}

	assert.Equal(t, int64(10), c.GetTotalRequests())

	// Reset
	c.Reset()

	assert.Equal(t, int64(0), c.GetTotalRequests())
	assert.Equal(t, int64(0), c.GetSuccessRequests())
	assert.Equal(t, int64(0), c.GetFailedRequests())

	snapshot := c.Snapshot()
	assert.Equal(t, int64(0), snapshot.TotalRequests)
	assert.Empty(t, snapshot.StatusCodes)
	assert.Empty(t, snapshot.EndpointStats)
}

func TestCollector_LatencyPercentiles(t *testing.T) {
	c := NewCollector(DefaultCollectorConfig())
	c.Start()

	// Create a controlled distribution of 100 latencies
	// 0-49: 1-50ms (50 samples)
	// 50-94: 51-95ms (45 samples)
	// 95-98: 100-400ms (4 samples)
	// 99: 1000ms (1 sample)
	for i := range 50 {
		c.Record(Result{
			Success: true,
			Latency: time.Duration(i+1) * time.Millisecond,
		})
	}
	for i := range 45 {
		c.Record(Result{
			Success: true,
			Latency: time.Duration(51+i) * time.Millisecond,
		})
	}
	for i := range 4 {
		c.Record(Result{
			Success: true,
			Latency: time.Duration(100*(i+1)) * time.Millisecond,
		})
	}
	c.Record(Result{
		Success: true,
		Latency: 1000 * time.Millisecond,
	})

	snapshot := c.Snapshot()

	// P50 should be around 50ms
	assert.True(t, snapshot.P50Latency >= 40*time.Millisecond && snapshot.P50Latency <= 60*time.Millisecond,
		"P50 should be around 50ms, got %v", snapshot.P50Latency)

	// P95 should be high
	assert.True(t, snapshot.P95Latency >= 95*time.Millisecond,
		"P95 should be >= 95ms, got %v", snapshot.P95Latency)

	// P99 should be very high
	assert.True(t, snapshot.P99Latency >= 100*time.Millisecond,
		"P99 should be >= 100ms, got %v", snapshot.P99Latency)
}

func TestCollector_QPS(t *testing.T) {
	c := NewCollector(DefaultCollectorConfig())
	c.Start()

	// Record 100 requests
	for range 100 {
		c.Record(Result{Success: true, StatusCode: 200, Latency: 1 * time.Millisecond})
	}

	// Wait a bit to ensure duration is measurable
	time.Sleep(100 * time.Millisecond)

	qps := c.GetCurrentQPS()
	assert.True(t, qps > 0, "QPS should be positive")

	// QPS should be roughly 100 / 0.1s = 1000
	// But due to test timing, just verify it's reasonable
	assert.True(t, qps > 100, "QPS should be > 100, got %v", qps)
}

func TestCollector_Duration(t *testing.T) {
	c := NewCollector(DefaultCollectorConfig())

	// Before start
	assert.Equal(t, time.Duration(0), c.Duration())

	c.Start()
	time.Sleep(50 * time.Millisecond)

	duration := c.Duration()
	assert.True(t, duration >= 50*time.Millisecond, "Duration should be >= 50ms, got %v", duration)

	c.Stop()
	finalDuration := c.Duration()
	assert.True(t, finalDuration >= 50*time.Millisecond)

	// Duration shouldn't increase after stop
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, finalDuration, c.Duration())
}

func TestCollector_LatencySampling(t *testing.T) {
	// Test that reservoir sampling works when exceeding maxLatencies
	c := NewCollector(CollectorConfig{
		MaxLatencies: 100,
	})
	c.Start()

	// Record more than maxLatencies
	for range 200 {
		c.Record(Result{
			Success: true,
			Latency: 10 * time.Millisecond,
		})
	}

	snapshot := c.Snapshot()
	assert.Equal(t, int64(200), snapshot.TotalRequests)

	// Latency stats should still be calculated
	assert.True(t, snapshot.AvgLatency > 0)
}

// TestConsole tests console output functionality
func TestConsole_NewConsole(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		console := NewConsole(ConsoleConfig{})
		require.NotNil(t, console)
		assert.NotNil(t, console.writer)
	})

	t.Run("custom config", func(t *testing.T) {
		buf := &bytes.Buffer{}
		console := NewConsole(ConsoleConfig{
			Writer:          buf,
			RefreshInterval: 100 * time.Millisecond,
			MaxEndpoints:    5,
		})
		require.NotNil(t, console)
		assert.Equal(t, buf, console.writer)
		assert.Equal(t, 100*time.Millisecond, console.config.RefreshInterval)
		assert.Equal(t, 5, console.config.MaxEndpoints)
	})
}

func TestConsole_StartStop(t *testing.T) {
	buf := &bytes.Buffer{}
	console := NewConsole(ConsoleConfig{
		Writer:          buf,
		RefreshInterval: 50 * time.Millisecond,
		UseColors:       false,
	})

	collector := NewCollector(DefaultCollectorConfig())
	collector.Start()

	// Record some data
	for range 10 {
		collector.Record(Result{
			EndpointName: "test",
			Success:      true,
			StatusCode:   200,
			Latency:      10 * time.Millisecond,
		})
	}

	// Start console
	console.Start(collector, func() string { return "rising to peak" })

	// Wait for at least one update
	time.Sleep(100 * time.Millisecond)

	// Stop console
	console.Stop()

	// Verify output was produced
	output := buf.String()
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "Requests:")
	assert.Contains(t, output, "Latency:")
}

func TestConsole_PrintFinalReport(t *testing.T) {
	buf := &bytes.Buffer{}
	console := NewConsole(ConsoleConfig{
		Writer:    buf,
		UseColors: false,
	})

	collector := NewCollector(DefaultCollectorConfig())
	collector.Start()

	// Record varied data
	for range 8 {
		collector.Record(Result{
			EndpointName: "GET /api/products",
			Success:      true,
			StatusCode:   200,
			Latency:      50 * time.Millisecond,
			ResponseSize: 1024,
		})
	}
	for range 2 {
		collector.Record(Result{
			EndpointName: "POST /api/orders",
			Success:      false,
			StatusCode:   500,
			Latency:      200 * time.Millisecond,
			ResponseSize: 256,
		})
	}

	collector.Stop()
	snapshot := collector.Snapshot()

	console.PrintFinalReport(snapshot)

	output := buf.String()

	// Verify report sections
	assert.Contains(t, output, "LOAD TEST FINAL REPORT")
	assert.Contains(t, output, "Test Duration")
	assert.Contains(t, output, "Request Statistics")
	assert.Contains(t, output, "Total Requests:")
	assert.Contains(t, output, "Successful:")
	assert.Contains(t, output, "Failed:")
	assert.Contains(t, output, "Success Rate:")
	assert.Contains(t, output, "Latency Distribution")
	assert.Contains(t, output, "Min:")
	assert.Contains(t, output, "Max:")
	assert.Contains(t, output, "P95:")
	assert.Contains(t, output, "P99:")
	assert.Contains(t, output, "Status Code Distribution")
	assert.Contains(t, output, "Per-Endpoint Statistics")
}

func TestFormatLatency(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{0, "0ms"},
		{500 * time.Nanosecond, "500ns"},
		{10 * time.Microsecond, "10.00µs"},
		{10 * time.Millisecond, "10.00ms"},
		{1 * time.Second, "1.00s"},
		{1500 * time.Millisecond, "1.50s"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatLatency(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{30 * time.Second, "30.0s"},
		{90 * time.Second, "1m30s"},
		{3600 * time.Second, "1h0m"},
		{3661 * time.Second, "1h1m"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatDuration(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatBytes(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestStatisticalAccuracy verifies the statistical calculations are accurate
func TestStatisticalAccuracy(t *testing.T) {
	c := NewCollector(DefaultCollectorConfig())
	c.Start()

	// Record a known set of latencies
	// Mean should be (1+2+3+4+5+6+7+8+9+10)/10 = 5.5
	for i := 1; i <= 10; i++ {
		c.Record(Result{
			Success: true,
			Latency: time.Duration(i) * time.Millisecond,
		})
	}

	snapshot := c.Snapshot()

	// Verify average
	expectedAvg := 5.5 * float64(time.Millisecond)
	actualAvg := float64(snapshot.AvgLatency)
	tolerance := 0.1 * float64(time.Millisecond) // 0.1ms tolerance
	assert.True(t, math.Abs(actualAvg-expectedAvg) < tolerance,
		"Average should be ~5.5ms, got %v", snapshot.AvgLatency)

	// Verify min/max
	assert.Equal(t, time.Millisecond, snapshot.MinLatency)
	assert.Equal(t, 10*time.Millisecond, snapshot.MaxLatency)

	// Verify success rate calculation
	c.Record(Result{Success: false, Latency: 1 * time.Millisecond})

	snapshot = c.Snapshot()
	expectedSuccessRate := 10.0 / 11.0 * 100 // 90.909...%
	tolerance = 0.01
	assert.True(t, math.Abs(snapshot.SuccessRate-expectedSuccessRate) < tolerance,
		"Success rate should be ~90.91%%, got %.2f%%", snapshot.SuccessRate)
}

func TestEndpointSnapshot_Percentiles(t *testing.T) {
	c := NewCollector(DefaultCollectorConfig())
	c.Start()

	// Create consistent latency pattern for endpoint
	for i := range 100 {
		c.Record(Result{
			EndpointName: "test-endpoint",
			Success:      true,
			StatusCode:   200,
			Latency:      time.Duration(i+1) * time.Millisecond,
		})
	}

	snapshot := c.Snapshot()
	epStats := snapshot.EndpointStats["test-endpoint"]
	require.NotNil(t, epStats)

	// P50 should be around 50ms
	assert.True(t, epStats.P50Latency >= 45*time.Millisecond && epStats.P50Latency <= 55*time.Millisecond,
		"P50 should be ~50ms, got %v", epStats.P50Latency)

	// P95 should be around 95ms
	assert.True(t, epStats.P95Latency >= 90*time.Millisecond,
		"P95 should be ~95ms, got %v", epStats.P95Latency)

	// P99 should be around 99ms
	assert.True(t, epStats.P99Latency >= 95*time.Millisecond,
		"P99 should be ~99ms, got %v", epStats.P99Latency)
}
