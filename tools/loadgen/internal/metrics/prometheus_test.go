package metrics

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPrometheusExporter(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		exporter := NewPrometheusExporter(PrometheusExporterConfig{})

		assert.Equal(t, 9090, exporter.GetPort())
		assert.Equal(t, "/metrics", exporter.GetPath())
		assert.NotNil(t, exporter.registry)
		assert.False(t, exporter.IsRunning())
	})

	t.Run("custom config", func(t *testing.T) {
		config := PrometheusExporterConfig{
			Port:             8080,
			Path:             "/custom-metrics",
			Namespace:        "test",
			Subsystem:        "loadgen",
			HistogramBuckets: []float64{0.001, 0.01, 0.1, 1, 10},
		}
		exporter := NewPrometheusExporter(config)

		assert.Equal(t, 8080, exporter.GetPort())
		assert.Equal(t, "/custom-metrics", exporter.GetPath())
	})
}

func TestDefaultPrometheusExporterConfig(t *testing.T) {
	config := DefaultPrometheusExporterConfig()

	assert.Equal(t, 9090, config.Port)
	assert.Equal(t, "/metrics", config.Path)
	assert.Equal(t, prometheus.DefBuckets, config.HistogramBuckets)
}

func TestPrometheusExporter_StartStop(t *testing.T) {
	// Use a random high port to avoid conflicts
	config := PrometheusExporterConfig{
		Port: 19090 + int(time.Now().UnixNano()%1000),
		Path: "/metrics",
	}
	exporter := NewPrometheusExporter(config)

	// Start server
	err := exporter.Start()
	require.NoError(t, err)
	assert.True(t, exporter.IsRunning())

	// Starting again should be idempotent
	err = exporter.Start()
	require.NoError(t, err)

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Verify metrics endpoint is accessible
	resp, err := http.Get(exporter.GetAddress())
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify health endpoint
	healthURL := fmt.Sprintf("http://localhost:%d/health", config.Port)
	resp, err = http.Get(healthURL)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = exporter.Stop(ctx)
	require.NoError(t, err)
	assert.False(t, exporter.IsRunning())

	// Stopping again should be idempotent
	err = exporter.Stop(ctx)
	require.NoError(t, err)
}

func TestPrometheusExporter_RecordRequest(t *testing.T) {
	exporter := NewPrometheusExporter(PrometheusExporterConfig{})

	// Record successful request
	exporter.RecordRequest(Result{
		EndpointName: "test-endpoint",
		StatusCode:   200,
		Success:      true,
		Latency:      100 * time.Millisecond,
		ResponseSize: 1024,
	})

	// Record failed request
	exporter.RecordRequest(Result{
		EndpointName: "test-endpoint",
		StatusCode:   500,
		Success:      false,
		Latency:      50 * time.Millisecond,
		ResponseSize: 256,
	})

	// Gather metrics and verify
	metricFamilies, err := exporter.Gather()
	require.NoError(t, err)

	// Find and verify requests_total counter
	requestsTotal := findMetricFamily(metricFamilies, "requests_total")
	require.NotNil(t, requestsTotal, "requests_total metric should exist")

	// Find success counter
	successMetric := findMetricByLabels(requestsTotal, map[string]string{
		"status":  "200",
		"success": "true",
	})
	require.NotNil(t, successMetric, "success metric should exist")
	assert.Equal(t, 1.0, successMetric.GetCounter().GetValue())

	// Find failure counter
	failureMetric := findMetricByLabels(requestsTotal, map[string]string{
		"status":  "500",
		"success": "false",
	})
	require.NotNil(t, failureMetric, "failure metric should exist")
	assert.Equal(t, 1.0, failureMetric.GetCounter().GetValue())

	// Verify request duration histogram
	durationHist := findMetricFamily(metricFamilies, "request_duration_seconds")
	require.NotNil(t, durationHist, "request_duration_seconds metric should exist")
	assert.Equal(t, dto.MetricType_HISTOGRAM, *durationHist.Type)

	// Verify bytes counter
	bytesTotal := findMetricFamily(metricFamilies, "request_bytes_total")
	require.NotNil(t, bytesTotal, "request_bytes_total metric should exist")
	assert.Equal(t, float64(1024+256), bytesTotal.Metric[0].GetCounter().GetValue())
}

func TestPrometheusExporter_UpdateGauges(t *testing.T) {
	exporter := NewPrometheusExporter(PrometheusExporterConfig{})

	// Update all gauges
	exporter.UpdateCurrentQPS(150.5)
	exporter.UpdateTargetQPS(200.0)
	exporter.UpdatePoolSize(10)
	exporter.UpdateBackpressureState(1)
	exporter.UpdateSuccessRate(95.5)
	exporter.UpdateActiveWorkers(8)

	// Gather metrics
	metricFamilies, err := exporter.Gather()
	require.NoError(t, err)

	// Verify current_qps
	currentQPS := findMetricFamily(metricFamilies, "current_qps")
	require.NotNil(t, currentQPS)
	assert.Equal(t, 150.5, currentQPS.Metric[0].GetGauge().GetValue())

	// Verify target_qps
	targetQPS := findMetricFamily(metricFamilies, "target_qps")
	require.NotNil(t, targetQPS)
	assert.Equal(t, 200.0, targetQPS.Metric[0].GetGauge().GetValue())

	// Verify pool_size
	poolSize := findMetricFamily(metricFamilies, "pool_size")
	require.NotNil(t, poolSize)
	assert.Equal(t, 10.0, poolSize.Metric[0].GetGauge().GetValue())

	// Verify backpressure_state
	backpressure := findMetricFamily(metricFamilies, "backpressure_state")
	require.NotNil(t, backpressure)
	assert.Equal(t, 1.0, backpressure.Metric[0].GetGauge().GetValue())

	// Verify success_rate
	successRate := findMetricFamily(metricFamilies, "success_rate")
	require.NotNil(t, successRate)
	assert.Equal(t, 95.5, successRate.Metric[0].GetGauge().GetValue())

	// Verify active_workers
	activeWorkers := findMetricFamily(metricFamilies, "active_workers")
	require.NotNil(t, activeWorkers)
	assert.Equal(t, 8.0, activeWorkers.Metric[0].GetGauge().GetValue())
}

func TestPrometheusExporter_UpdateFromSnapshot(t *testing.T) {
	exporter := NewPrometheusExporter(PrometheusExporterConfig{})

	snapshot := Snapshot{
		QPS:         123.45,
		SuccessRate: 98.76,
	}

	exporter.UpdateFromSnapshot(snapshot)

	metricFamilies, err := exporter.Gather()
	require.NoError(t, err)

	// Verify current_qps
	currentQPS := findMetricFamily(metricFamilies, "current_qps")
	require.NotNil(t, currentQPS)
	assert.InDelta(t, 123.45, currentQPS.Metric[0].GetGauge().GetValue(), 0.001)

	// Verify success_rate
	successRate := findMetricFamily(metricFamilies, "success_rate")
	require.NotNil(t, successRate)
	assert.InDelta(t, 98.76, successRate.Metric[0].GetGauge().GetValue(), 0.001)
}

func TestPrometheusExporter_GetAddress(t *testing.T) {
	config := PrometheusExporterConfig{
		Port: 8080,
		Path: "/custom",
	}
	exporter := NewPrometheusExporter(config)

	assert.Equal(t, "http://localhost:8080/custom", exporter.GetAddress())
}

func TestPrometheusExporter_MetricsEndpointContent(t *testing.T) {
	// Use a random high port to avoid conflicts
	config := PrometheusExporterConfig{
		Port: 19090 + int(time.Now().UnixNano()%1000),
		Path: "/metrics",
	}
	exporter := NewPrometheusExporter(config)

	// Record some data
	exporter.RecordRequest(Result{
		EndpointName: "test-endpoint",
		StatusCode:   200,
		Success:      true,
		Latency:      100 * time.Millisecond,
		ResponseSize: 1024,
	})
	exporter.UpdateCurrentQPS(50.0)
	exporter.UpdateTargetQPS(100.0)
	exporter.UpdatePoolSize(5)

	// Start server
	err := exporter.Start()
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = exporter.Stop(ctx)
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Fetch metrics
	resp, err := http.Get(exporter.GetAddress())
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	content := string(body)

	// Verify expected metrics are present
	expectedMetrics := []string{
		"requests_total",
		"request_duration_seconds",
		"current_qps",
		"target_qps",
		"pool_size",
		"backpressure_state",
		"success_rate",
		"active_workers",
		"request_bytes_total",
	}

	for _, metric := range expectedMetrics {
		assert.Contains(t, content, metric, "Metrics should contain %s", metric)
	}

	// Verify labels are present
	assert.Contains(t, content, `status="200"`)
	assert.Contains(t, content, `success="true"`)
	assert.Contains(t, content, `endpoint="test-endpoint"`)
}

func TestPrometheusExporter_MetricsWithNamespace(t *testing.T) {
	config := PrometheusExporterConfig{
		Port:      19090 + int(time.Now().UnixNano()%1000),
		Path:      "/metrics",
		Namespace: "myapp",
		Subsystem: "loadgen",
	}
	exporter := NewPrometheusExporter(config)

	exporter.UpdateCurrentQPS(50.0)

	// Start server
	err := exporter.Start()
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = exporter.Stop(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Fetch metrics
	resp, err := http.Get(exporter.GetAddress())
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	content := string(body)

	// Verify namespaced metric names
	assert.Contains(t, content, "myapp_loadgen_current_qps")
}

func TestPrometheusExporter_CustomHistogramBuckets(t *testing.T) {
	config := PrometheusExporterConfig{
		HistogramBuckets: []float64{0.001, 0.01, 0.1, 0.5, 1.0},
	}
	exporter := NewPrometheusExporter(config)

	// Record requests with different latencies
	latencies := []time.Duration{
		500 * time.Microsecond, // < 0.001s
		5 * time.Millisecond,   // < 0.01s
		50 * time.Millisecond,  // < 0.1s
		200 * time.Millisecond, // < 0.5s
		800 * time.Millisecond, // < 1.0s
	}

	for _, lat := range latencies {
		exporter.RecordRequest(Result{
			EndpointName: "test",
			StatusCode:   200,
			Success:      true,
			Latency:      lat,
		})
	}

	// Gather and verify histogram
	metricFamilies, err := exporter.Gather()
	require.NoError(t, err)

	durationHist := findMetricFamily(metricFamilies, "request_duration_seconds")
	require.NotNil(t, durationHist)

	// Verify histogram has data
	hist := durationHist.Metric[0].GetHistogram()
	require.NotNil(t, hist)
	assert.Equal(t, uint64(5), hist.GetSampleCount())
}

func TestPrometheusExporter_ConcurrentAccess(t *testing.T) {
	exporter := NewPrometheusExporter(PrometheusExporterConfig{})

	// Start multiple goroutines recording requests concurrently
	done := make(chan bool)
	const goroutines = 10
	const requestsPerGoroutine = 100

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			for j := 0; j < requestsPerGoroutine; j++ {
				exporter.RecordRequest(Result{
					EndpointName: fmt.Sprintf("endpoint-%d", id),
					StatusCode:   200,
					Success:      true,
					Latency:      time.Duration(j) * time.Millisecond,
					ResponseSize: int64(j * 100),
				})
				exporter.UpdateCurrentQPS(float64(j))
				exporter.UpdatePoolSize(j)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}

	// Verify total requests
	metricFamilies, err := exporter.Gather()
	require.NoError(t, err)

	requestsTotal := findMetricFamily(metricFamilies, "requests_total")
	require.NotNil(t, requestsTotal)

	// Sum all counters
	var total float64
	for _, m := range requestsTotal.Metric {
		total += m.GetCounter().GetValue()
	}
	assert.Equal(t, float64(goroutines*requestsPerGoroutine), total)
}

func TestPrometheusExporter_BackpressureStates(t *testing.T) {
	exporter := NewPrometheusExporter(PrometheusExporterConfig{})

	states := []struct {
		state int
		name  string
	}{
		{0, "normal"},
		{1, "warning"},
		{2, "critical"},
	}

	for _, s := range states {
		t.Run(s.name, func(t *testing.T) {
			exporter.UpdateBackpressureState(s.state)

			metricFamilies, err := exporter.Gather()
			require.NoError(t, err)

			backpressure := findMetricFamily(metricFamilies, "backpressure_state")
			require.NotNil(t, backpressure)
			assert.Equal(t, float64(s.state), backpressure.Metric[0].GetGauge().GetValue())
		})
	}
}

func TestPrometheusExporter_Registry(t *testing.T) {
	exporter := NewPrometheusExporter(PrometheusExporterConfig{})

	registry := exporter.Registry()
	require.NotNil(t, registry)

	// Verify we can gather from the registry directly
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)
	assert.NotEmpty(t, metricFamilies)
}

// Helper functions

func findMetricFamily(families []*dto.MetricFamily, name string) *dto.MetricFamily {
	for _, f := range families {
		if strings.HasSuffix(f.GetName(), name) {
			return f
		}
	}
	return nil
}

func findMetricByLabels(family *dto.MetricFamily, labels map[string]string) *dto.Metric {
	for _, m := range family.Metric {
		match := true
		for wantKey, wantValue := range labels {
			found := false
			for _, l := range m.Label {
				if l.GetName() == wantKey && l.GetValue() == wantValue {
					found = true
					break
				}
			}
			if !found {
				match = false
				break
			}
		}
		if match {
			return m
		}
	}
	return nil
}
