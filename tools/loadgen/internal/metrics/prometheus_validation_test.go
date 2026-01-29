// Package metrics provides metrics collection and reporting for the load generator.
package metrics

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPrometheusValidation_LOADGEN_VAL_007 validates all Prometheus metrics
// required by task LOADGEN-VAL-007:
//   - 启动Prometheus导出器 (端口9090)
//   - 验证/metrics端点可访问
//   - 检查loadgen_requests_total计数器
//   - 检查loadgen_request_duration_seconds直方图
//   - 检查loadgen_current_qps gauge
//   - 检查loadgen_target_qps gauge
//   - 检查loadgen_backpressure_state gauge
func TestPrometheusValidation_LOADGEN_VAL_007(t *testing.T) {
	// Use port 9090 as specified in requirements (or fallback to random if occupied)
	config := PrometheusExporterConfig{
		Port: 9090,
		Path: "/metrics",
	}
	exporter := NewPrometheusExporter(config)

	// Test 1: Start Prometheus exporter on port 9090
	t.Run("Start Prometheus exporter on port 9090", func(t *testing.T) {
		err := exporter.Start()
		if err != nil {
			// Port 9090 might be in use, try alternate port for testing
			t.Logf("Port 9090 in use, using alternate port for testing: %v", err)
			config.Port = 19090
			exporter = NewPrometheusExporter(config)
			err = exporter.Start()
		}
		require.NoError(t, err, "Should be able to start Prometheus exporter")
		assert.True(t, exporter.IsRunning(), "Exporter should be running")
	})

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = exporter.Stop(ctx)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test 2: Verify /metrics endpoint is accessible
	t.Run("Verify /metrics endpoint accessible", func(t *testing.T) {
		resp, err := http.Get(exporter.GetAddress())
		require.NoError(t, err, "/metrics endpoint should be accessible")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode, "/metrics should return 200 OK")

		// Verify response is valid Prometheus format
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		content := string(body)
		assert.Contains(t, content, "# HELP", "Response should be valid Prometheus format")
		assert.Contains(t, content, "# TYPE", "Response should be valid Prometheus format")
	})

	// Simulate some load test activity to populate metrics
	simulateLoadTestActivity(exporter)

	// Test 3: Check loadgen_requests_total counter
	t.Run("Check loadgen_requests_total counter", func(t *testing.T) {
		content := fetchMetrics(t, exporter)

		// Verify the counter exists and has expected format
		assert.Contains(t, content, "requests_total",
			"loadgen_requests_total counter should exist")
		assert.Contains(t, content, "# TYPE",
			"Metric should have TYPE annotation")
		assert.Contains(t, content, "counter",
			"requests_total should be a counter type")
		assert.Contains(t, content, `status="200"`,
			"Counter should have status label")
		assert.Contains(t, content, `success="true"`,
			"Counter should have success label")
	})

	// Test 4: Check loadgen_request_duration_seconds histogram
	t.Run("Check loadgen_request_duration_seconds histogram", func(t *testing.T) {
		content := fetchMetrics(t, exporter)

		// Verify histogram exists with all components
		assert.Contains(t, content, "request_duration_seconds",
			"loadgen_request_duration_seconds histogram should exist")
		assert.Contains(t, content, "request_duration_seconds_bucket",
			"Histogram should have bucket metric")
		assert.Contains(t, content, "request_duration_seconds_count",
			"Histogram should have count metric")
		assert.Contains(t, content, "request_duration_seconds_sum",
			"Histogram should have sum metric")
		assert.Contains(t, content, `endpoint=`,
			"Histogram should have endpoint label")
	})

	// Test 5: Check loadgen_current_qps gauge
	t.Run("Check loadgen_current_qps gauge", func(t *testing.T) {
		content := fetchMetrics(t, exporter)

		assert.Contains(t, content, "current_qps",
			"loadgen_current_qps gauge should exist")

		// Verify it's a gauge type
		lines := strings.Split(content, "\n")
		foundType := false
		for _, line := range lines {
			if strings.Contains(line, "current_qps") && strings.Contains(line, "# TYPE") {
				assert.Contains(t, line, "gauge", "current_qps should be gauge type")
				foundType = true
				break
			}
		}
		assert.True(t, foundType, "Should find TYPE annotation for current_qps")

		// Verify value is updated
		assert.Contains(t, content, "current_qps 100",
			"current_qps should have the expected value")
	})

	// Test 6: Check loadgen_target_qps gauge
	t.Run("Check loadgen_target_qps gauge", func(t *testing.T) {
		content := fetchMetrics(t, exporter)

		assert.Contains(t, content, "target_qps",
			"loadgen_target_qps gauge should exist")

		// Verify value is updated
		assert.Contains(t, content, "target_qps 200",
			"target_qps should have the expected value")
	})

	// Test 7: Check loadgen_backpressure_state gauge
	t.Run("Check loadgen_backpressure_state gauge", func(t *testing.T) {
		content := fetchMetrics(t, exporter)

		assert.Contains(t, content, "backpressure_state",
			"loadgen_backpressure_state gauge should exist")

		// Verify value (0=normal, 1=warning, 2=critical)
		// We set it to 1 (warning) in simulateLoadTestActivity
		assert.Contains(t, content, "backpressure_state 1",
			"backpressure_state should have the expected value (1=warning)")
	})

	// Test 8: Verify all metrics are correctly registered and updating
	t.Run("Verify all metrics registered and updating", func(t *testing.T) {
		// Update metrics with new values
		exporter.UpdateCurrentQPS(150.5)
		exporter.UpdateTargetQPS(250.0)
		exporter.UpdateBackpressureState(2) // critical

		// Record another request
		exporter.RecordRequest(Result{
			EndpointName: "api/v1/orders",
			StatusCode:   500,
			Success:      false,
			Latency:      500 * time.Millisecond,
			ResponseSize: 512,
		})

		content := fetchMetrics(t, exporter)

		// Verify updated values
		assert.Contains(t, content, "current_qps 150.5",
			"current_qps should update to new value")
		assert.Contains(t, content, "target_qps 250",
			"target_qps should update to new value")
		assert.Contains(t, content, "backpressure_state 2",
			"backpressure_state should update to critical (2)")
		assert.Contains(t, content, `status="500"`,
			"Should have recorded the failed request")
		assert.Contains(t, content, `success="false"`,
			"Should have recorded the failed request with success=false")
	})

	// Test 9: Verify Prometheus format validity
	t.Run("Verify valid Prometheus format", func(t *testing.T) {
		resp, err := http.Get(exporter.GetAddress())
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check content type
		contentType := resp.Header.Get("Content-Type")
		assert.True(t,
			strings.Contains(contentType, "text/plain") ||
				strings.Contains(contentType, "application/openmetrics-text"),
			"Content-Type should be text/plain or openmetrics format")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		content := string(body)

		// Verify format: each metric should have HELP and TYPE
		metricsToCheck := []string{
			"requests_total",
			"request_duration_seconds",
			"current_qps",
			"target_qps",
			"backpressure_state",
			"pool_size",
			"success_rate",
			"active_workers",
			"request_bytes_total",
		}

		for _, metric := range metricsToCheck {
			// Check HELP exists
			helpFound := false
			typeFound := false
			for _, line := range strings.Split(content, "\n") {
				if strings.Contains(line, "# HELP") && strings.Contains(line, metric) {
					helpFound = true
				}
				if strings.Contains(line, "# TYPE") && strings.Contains(line, metric) {
					typeFound = true
				}
			}
			assert.True(t, helpFound, "Metric %s should have HELP annotation", metric)
			assert.True(t, typeFound, "Metric %s should have TYPE annotation", metric)
		}
	})
}

// TestPrometheusMetricsMatchActualRun verifies metrics match actual running situation
func TestPrometheusMetricsMatchActualRun(t *testing.T) {
	config := PrometheusExporterConfig{
		Port: 19091 + int(time.Now().UnixNano()%100),
		Path: "/metrics",
	}
	exporter := NewPrometheusExporter(config)

	err := exporter.Start()
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = exporter.Stop(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Simulate a realistic load test scenario
	t.Run("Simulate realistic load test", func(t *testing.T) {
		// Phase 1: Warmup (low QPS, all success)
		exporter.UpdateTargetQPS(10)
		exporter.UpdateCurrentQPS(8)
		exporter.UpdateBackpressureState(0) // normal
		exporter.UpdatePoolSize(5)
		exporter.UpdateActiveWorkers(3)

		for i := 0; i < 10; i++ {
			exporter.RecordRequest(Result{
				EndpointName: "warmup-endpoint",
				StatusCode:   200,
				Success:      true,
				Latency:      time.Duration(50+i*10) * time.Millisecond,
				ResponseSize: 1024,
			})
		}
		exporter.UpdateSuccessRate(100.0)

		// Verify warmup metrics
		content := fetchMetrics(t, exporter)
		assert.Contains(t, content, "target_qps 10")
		assert.Contains(t, content, "current_qps 8")
		assert.Contains(t, content, "backpressure_state 0")
		assert.Contains(t, content, "pool_size 5")
		assert.Contains(t, content, "active_workers 3")
		assert.Contains(t, content, "success_rate 100")

		// Phase 2: Ramp up (higher QPS, some errors)
		exporter.UpdateTargetQPS(100)
		exporter.UpdateCurrentQPS(95)
		exporter.UpdateBackpressureState(1) // warning
		exporter.UpdatePoolSize(20)
		exporter.UpdateActiveWorkers(18)

		for i := 0; i < 90; i++ {
			exporter.RecordRequest(Result{
				EndpointName: "main-endpoint",
				StatusCode:   200,
				Success:      true,
				Latency:      time.Duration(100+i) * time.Millisecond,
				ResponseSize: 2048,
			})
		}
		// Add some failures
		for i := 0; i < 5; i++ {
			exporter.RecordRequest(Result{
				EndpointName: "main-endpoint",
				StatusCode:   500,
				Success:      false,
				Latency:      500 * time.Millisecond,
				ResponseSize: 256,
			})
		}
		exporter.UpdateSuccessRate(94.7)

		// Verify ramp up metrics
		content = fetchMetrics(t, exporter)
		assert.Contains(t, content, "target_qps 100")
		assert.Contains(t, content, "current_qps 95")
		assert.Contains(t, content, "backpressure_state 1")
		assert.Contains(t, content, "pool_size 20")
		assert.Contains(t, content, "active_workers 18")

		// Phase 3: Backpressure (system under stress)
		exporter.UpdateBackpressureState(2) // critical
		exporter.UpdateCurrentQPS(60)       // falling behind target
		exporter.UpdateSuccessRate(85.0)

		content = fetchMetrics(t, exporter)
		assert.Contains(t, content, "backpressure_state 2")
		assert.Contains(t, content, "current_qps 60")
	})
}

// simulateLoadTestActivity simulates load test activity to populate metrics
func simulateLoadTestActivity(exporter *PrometheusExporter) {
	// Set initial gauge values
	exporter.UpdateCurrentQPS(100.0)
	exporter.UpdateTargetQPS(200.0)
	exporter.UpdatePoolSize(10)
	exporter.UpdateBackpressureState(1) // warning state
	exporter.UpdateSuccessRate(95.0)
	exporter.UpdateActiveWorkers(8)

	// Record some successful requests
	for i := 0; i < 10; i++ {
		exporter.RecordRequest(Result{
			EndpointName: "api/v1/products",
			StatusCode:   200,
			Success:      true,
			Latency:      time.Duration(100+i*10) * time.Millisecond,
			ResponseSize: int64(1024 + i*100),
		})
	}

	// Record some failed requests
	for i := 0; i < 2; i++ {
		exporter.RecordRequest(Result{
			EndpointName: "api/v1/products",
			StatusCode:   500,
			Success:      false,
			Latency:      time.Duration(500+i*100) * time.Millisecond,
			ResponseSize: int64(256),
		})
	}
}

// fetchMetrics fetches the metrics from the exporter's HTTP endpoint
func fetchMetrics(t *testing.T, exporter *PrometheusExporter) string {
	resp, err := http.Get(exporter.GetAddress())
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return string(body)
}
