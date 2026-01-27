// Package e2e provides end-to-end integration tests for the loadgen against the ERP backend.
//
// These tests require a running ERP backend. They are skipped by default and can be
// enabled by setting the LOADGEN_E2E_TEST=1 environment variable.
//
// Usage:
//
//	LOADGEN_E2E_TEST=1 go test -v ./internal/e2e/...
//	LOADGEN_E2E_TEST=1 ERP_BASE_URL=http://localhost:8080 go test -v ./internal/e2e/...
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/example/erp/tools/loadgen/internal/config"
	"github.com/example/erp/tools/loadgen/internal/pool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skipUnlessE2E skips the test unless E2E testing is enabled.
func skipUnlessE2E(t *testing.T) {
	t.Helper()
	if os.Getenv("LOADGEN_E2E_TEST") != "1" {
		t.Skip("E2E tests disabled. Set LOADGEN_E2E_TEST=1 to enable.")
	}
}

// getBaseURL returns the ERP backend base URL from environment or default.
func getBaseURL() string {
	if url := os.Getenv("ERP_BASE_URL"); url != "" {
		return url
	}
	return "http://localhost:8080"
}

// TestERPConnectivity verifies basic connectivity to the ERP backend.
func TestERPConnectivity(t *testing.T) {
	skipUnlessE2E(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	baseURL := getBaseURL()
	pingURL := baseURL + "/api/v1/system/ping"

	req, err := http.NewRequestWithContext(ctx, "GET", pingURL, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "Failed to connect to ERP backend at %s", pingURL)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "ERP ping endpoint should return 200")
}

// TestERPAuthentication verifies login flow against the ERP backend.
func TestERPAuthentication(t *testing.T) {
	skipUnlessE2E(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	baseURL := getBaseURL()
	loginURL := baseURL + "/api/v1/auth/login"

	// Load config to get credentials
	cfg, err := loadERPConfig(t)
	require.NoError(t, err)
	require.NotNil(t, cfg.Auth.Login)

	loginBody := map[string]string{
		"username": cfg.Auth.Login.Username,
		"password": cfg.Auth.Login.Password,
	}
	bodyBytes, err := json.Marshal(loginBody)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, "POST", loginURL, bytes.NewReader(bodyBytes))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "Failed to login to ERP backend")
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Login should succeed: %s", string(body))

	// Verify we got a token
	var loginResp struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	err = json.Unmarshal(body, &loginResp)
	require.NoError(t, err)
	assert.True(t, loginResp.Success, "Login response should indicate success")
	assert.NotEmpty(t, loginResp.Data.AccessToken, "Should receive access token")
}

// TestERPWarmupEndpoints verifies that warmup producer endpoints are accessible.
func TestERPWarmupEndpoints(t *testing.T) {
	skipUnlessE2E(t)

	cfg, err := loadERPConfig(t)
	require.NoError(t, err)

	// Get auth token first
	token, err := authenticate(t, cfg)
	require.NoError(t, err, "Authentication failed")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	baseURL := getBaseURL()
	client := &http.Client{Timeout: 30 * time.Second}

	// Test each producer endpoint
	producerEndpoints := getProducerEndpoints(cfg)
	t.Logf("Testing %d producer endpoints", len(producerEndpoints))

	for _, ep := range producerEndpoints {
		t.Run(ep.Name, func(t *testing.T) {
			if ep.Method != "GET" {
				t.Skip("Skipping non-GET producer endpoint")
			}

			url := baseURL + "/api/v1" + ep.Path
			req, err := http.NewRequestWithContext(ctx, ep.Method, url, nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Accept", "application/json")

			resp, err := client.Do(req)
			require.NoError(t, err, "Request failed for %s", ep.Name)
			defer resp.Body.Close()

			// Accept 200 OK or 401 (if endpoint requires specific permissions)
			assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusForbidden,
				"Endpoint %s should return 200 or 403, got %d", ep.Name, resp.StatusCode)
		})
	}
}

// TestERPSimpleLoadRun performs a simple load test run (30 seconds).
func TestERPSimpleLoadRun(t *testing.T) {
	skipUnlessE2E(t)

	cfg, err := loadERPConfig(t)
	require.NoError(t, err)

	// Get auth token
	token, err := authenticate(t, cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	baseURL := getBaseURL()
	client := &http.Client{Timeout: 10 * time.Second}

	// Simple load test: hit a few read endpoints repeatedly for 30 seconds
	readEndpoints := getReadEndpoints(cfg)
	require.NotEmpty(t, readEndpoints, "Should have read endpoints")

	// Track metrics
	var totalRequests, successRequests, failedRequests int
	startTime := time.Now()
	testDuration := 30 * time.Second

	t.Logf("Running simple load test for %s against %d endpoints", testDuration, len(readEndpoints))

	ticker := time.NewTicker(100 * time.Millisecond) // ~10 QPS
	defer ticker.Stop()

	endpointIdx := 0
	for {
		select {
		case <-ctx.Done():
			t.Fatal("Context cancelled")
			return
		case <-ticker.C:
			if time.Since(startTime) >= testDuration {
				goto done
			}

			ep := readEndpoints[endpointIdx%len(readEndpoints)]
			endpointIdx++

			url := baseURL + "/api/v1" + ep.Path
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				failedRequests++
				totalRequests++
				continue
			}
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Accept", "application/json")

			resp, err := client.Do(req)
			totalRequests++
			if err != nil {
				failedRequests++
				continue
			}
			resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				successRequests++
			} else {
				failedRequests++
			}
		}
	}

done:
	elapsed := time.Since(startTime)
	qps := float64(totalRequests) / elapsed.Seconds()
	successRate := float64(successRequests) / float64(totalRequests) * 100

	t.Logf("Load test results:")
	t.Logf("  Duration: %s", elapsed)
	t.Logf("  Total requests: %d", totalRequests)
	t.Logf("  Successful: %d", successRequests)
	t.Logf("  Failed: %d", failedRequests)
	t.Logf("  QPS: %.2f", qps)
	t.Logf("  Success rate: %.2f%%", successRate)

	// Acceptance criteria
	assert.GreaterOrEqual(t, totalRequests, 100, "Should have made at least 100 requests")
	assert.Greater(t, successRate, 90.0, "Success rate should be > 90%%")
}

// TestERPExtendedLoadRun performs a 5-minute load test run.
// This is the main acceptance test for LOADGEN-029.
func TestERPExtendedLoadRun(t *testing.T) {
	skipUnlessE2E(t)

	cfg, err := loadERPConfig(t)
	require.NoError(t, err)

	// Get auth token
	token, err := authenticate(t, cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	baseURL := getBaseURL()
	client := &http.Client{Timeout: 10 * time.Second}

	// Use all enabled endpoints
	endpoints := cfg.GetEnabledEndpoints()
	require.NotEmpty(t, endpoints)

	// Parameter pool for dynamic values
	paramPool := pool.NewShardedPool(&pool.PoolConfig{
		MaxValuesPerType: 1000,
		DefaultTTL:       5 * time.Minute,
	})
	defer paramPool.Close()

	// Metrics
	var totalRequests, successRequests, failedRequests int
	latencies := make([]time.Duration, 0, 10000)
	startTime := time.Now()
	testDuration := 5 * time.Minute

	t.Logf("Running 5-minute load test against %d endpoints", len(endpoints))
	t.Logf("Target: ~50 QPS (5000+ total requests)")

	// Use weighted selection
	weightedEndpoints := buildWeightedList(endpoints)
	ticker := time.NewTicker(20 * time.Millisecond) // ~50 QPS
	defer ticker.Stop()

	reqIdx := 0
	for {
		select {
		case <-ctx.Done():
			t.Fatal("Context cancelled")
			return
		case <-ticker.C:
			elapsed := time.Since(startTime)
			if elapsed >= testDuration {
				goto done
			}

			// Log progress every minute
			if reqIdx%3000 == 0 {
				t.Logf("Progress: %s elapsed, %d requests", elapsed.Round(time.Second), totalRequests)
			}

			ep := weightedEndpoints[reqIdx%len(weightedEndpoints)]
			reqIdx++

			// Skip write operations for safety
			if ep.Method != "GET" {
				continue
			}

			url := baseURL + "/api/v1" + ep.Path
			reqStart := time.Now()

			req, err := http.NewRequestWithContext(ctx, ep.Method, url, nil)
			if err != nil {
				failedRequests++
				totalRequests++
				continue
			}
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Accept", "application/json")

			resp, err := client.Do(req)
			totalRequests++
			latency := time.Since(reqStart)
			latencies = append(latencies, latency)

			if err != nil {
				failedRequests++
				continue
			}
			resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				successRequests++
			} else {
				failedRequests++
			}
		}
	}

done:
	elapsed := time.Since(startTime)
	qps := float64(totalRequests) / elapsed.Seconds()
	successRate := float64(successRequests) / float64(totalRequests) * 100

	// Calculate P95 latency
	p95 := calculateP95(latencies)

	t.Logf("\n===== 5-MINUTE LOAD TEST RESULTS =====")
	t.Logf("Duration: %s", elapsed.Round(time.Second))
	t.Logf("Total requests: %d", totalRequests)
	t.Logf("Successful: %d", successRequests)
	t.Logf("Failed: %d", failedRequests)
	t.Logf("QPS: %.2f", qps)
	t.Logf("Success rate: %.2f%%", successRate)
	t.Logf("P95 latency: %s", p95)
	t.Logf("======================================\n")

	// Acceptance criteria for LOADGEN-029
	assert.GreaterOrEqual(t, totalRequests, 5000, "Should have made at least 5000 requests in 5 minutes")
	assert.Greater(t, successRate, 95.0, "Success rate should be > 95%% for a healthy system")
	assert.Less(t, p95, 2*time.Second, "P95 latency should be < 2s")
}

// Helper functions

func loadERPConfig(t *testing.T) (*config.Config, error) {
	t.Helper()

	// Try to find config file
	paths := []string{
		"../../configs/erp.yaml",
		"../../../configs/erp.yaml",
		"configs/erp.yaml",
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return config.LoadFromFile(p)
		}
	}

	return nil, fmt.Errorf("could not find erp.yaml config file")
}

func authenticate(t *testing.T, cfg *config.Config) (string, error) {
	t.Helper()

	if cfg.Auth.Login == nil {
		return "", fmt.Errorf("no login config")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	baseURL := getBaseURL()
	loginURL := baseURL + "/api/v1" + cfg.Auth.Login.Endpoint

	loginBody := map[string]string{
		"username": cfg.Auth.Login.Username,
		"password": cfg.Auth.Login.Password,
	}
	bodyBytes, _ := json.Marshal(loginBody)

	req, err := http.NewRequestWithContext(ctx, "POST", loginURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	var loginResp struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return "", err
	}

	return loginResp.Data.AccessToken, nil
}

func getProducerEndpoints(cfg *config.Config) []config.EndpointConfig {
	var producers []config.EndpointConfig
	for _, ep := range cfg.Endpoints {
		if len(ep.Produces) > 0 && !ep.Disabled {
			producers = append(producers, ep)
		}
	}
	return producers
}

func getReadEndpoints(cfg *config.Config) []config.EndpointConfig {
	var reads []config.EndpointConfig
	for _, ep := range cfg.Endpoints {
		if ep.Method == "GET" && !ep.Disabled {
			reads = append(reads, ep)
		}
	}
	return reads
}

func buildWeightedList(endpoints []config.EndpointConfig) []config.EndpointConfig {
	var weighted []config.EndpointConfig
	for _, ep := range endpoints {
		weight := ep.Weight
		if weight <= 0 {
			weight = 1
		}
		for i := 0; i < weight; i++ {
			weighted = append(weighted, ep)
		}
	}
	return weighted
}

func calculateP95(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	// Simple sort for P95
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)

	// Bubble sort (simple for test)
	for i := range sorted {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	idx := int(float64(len(sorted)) * 0.95)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
