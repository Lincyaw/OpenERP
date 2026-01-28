package executor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/erp/tools/loadgen/internal/circuit"
	"github.com/example/erp/tools/loadgen/internal/loadctrl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHTTPServer creates a test server that responds with configurable delays and statuses
func mockHTTPServer(t *testing.T) *httptest.Server {
	requestCount := &atomic.Int32{}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)

		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)

		// Return different responses based on path
		switch r.URL.Path {
		case "/error":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": "internal error"}`))
		case "/slow":
			time.Sleep(500 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status": "slow response"}`))
		case "/auth-required":
			auth := r.Header.Get("Authorization")
			if auth == "" {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error": "unauthorized"}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status": "authenticated"}`))
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			resp := map[string]any{
				"status":  "ok",
				"path":    r.URL.Path,
				"method":  r.Method,
				"request": count,
			}
			_ = json.NewEncoder(w).Encode(resp)
		}
	}))
}

func TestNewExecutor(t *testing.T) {
	t.Run("creates executor with required components", func(t *testing.T) {
		scheduler := NewScheduler(DefaultSchedulerConfig())
		rateLimiter := loadctrl.NewTokenBucketLimiter(100, 10)
		workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{
			MinSize: 1,
			MaxSize: 10,
		})

		executor, err := NewExecutor(
			DefaultExecutorConfig(),
			scheduler,
			rateLimiter,
			workerPool,
			nil, // no circuit board
			nil, // use default http client
		)

		require.NoError(t, err)
		require.NotNil(t, executor)
	})

	t.Run("returns error when scheduler is nil", func(t *testing.T) {
		rateLimiter := loadctrl.NewTokenBucketLimiter(100, 10)
		workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{MinSize: 1, MaxSize: 10})

		_, err := NewExecutor(
			DefaultExecutorConfig(),
			nil, // nil scheduler
			rateLimiter,
			workerPool,
			nil,
			nil,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "scheduler is required")
	})

	t.Run("returns error when rate limiter is nil", func(t *testing.T) {
		scheduler := NewScheduler(DefaultSchedulerConfig())
		workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{MinSize: 1, MaxSize: 10})

		_, err := NewExecutor(
			DefaultExecutorConfig(),
			scheduler,
			nil, // nil rate limiter
			workerPool,
			nil,
			nil,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "rate limiter is required")
	})

	t.Run("returns error when worker pool is nil", func(t *testing.T) {
		scheduler := NewScheduler(DefaultSchedulerConfig())
		rateLimiter := loadctrl.NewTokenBucketLimiter(100, 10)

		_, err := NewExecutor(
			DefaultExecutorConfig(),
			scheduler,
			rateLimiter,
			nil, // nil worker pool
			nil,
			nil,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "worker pool is required")
	})
}

func TestExecutor_StartStop(t *testing.T) {
	t.Run("starts and stops successfully", func(t *testing.T) {
		server := mockHTTPServer(t)
		defer server.Close()

		scheduler := NewScheduler(DefaultSchedulerConfig())
		_ = scheduler.Register(&EndpointInfo{
			Name:   "test",
			Method: "GET",
			Path:   "/test",
			Weight: 1,
		})

		rateLimiter := loadctrl.NewTokenBucketLimiter(10, 5)
		workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{MinSize: 1, MaxSize: 5})

		config := DefaultExecutorConfig()
		config.BaseURL = server.URL

		executor, err := NewExecutor(config, scheduler, rateLimiter, workerPool, nil, nil)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = executor.Start(ctx)
		require.NoError(t, err)
		assert.True(t, executor.IsRunning())

		// Let it run for a bit
		time.Sleep(100 * time.Millisecond)

		executor.Stop()
		assert.False(t, executor.IsRunning())

		// Check some requests were made
		stats := executor.Stats()
		assert.Greater(t, stats.TotalRequests, int64(0))
	})

	t.Run("prevents double start", func(t *testing.T) {
		scheduler := NewScheduler(DefaultSchedulerConfig())
		rateLimiter := loadctrl.NewTokenBucketLimiter(10, 5)
		workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{MinSize: 1, MaxSize: 5})

		executor, _ := NewExecutor(DefaultExecutorConfig(), scheduler, rateLimiter, workerPool, nil, nil)

		ctx := context.Background()
		_ = executor.Start(ctx)

		err := executor.Start(ctx)
		assert.ErrorIs(t, err, ErrExecutorAlreadyRunning)

		executor.Stop()
	})

	t.Run("stop is idempotent", func(t *testing.T) {
		scheduler := NewScheduler(DefaultSchedulerConfig())
		rateLimiter := loadctrl.NewTokenBucketLimiter(10, 5)
		workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{MinSize: 1, MaxSize: 5})

		executor, _ := NewExecutor(DefaultExecutorConfig(), scheduler, rateLimiter, workerPool, nil, nil)

		ctx := context.Background()
		_ = executor.Start(ctx)

		executor.Stop()
		executor.Stop() // Should not panic
		executor.Stop() // Should not panic
	})
}

func TestExecutor_ExecuteOnce(t *testing.T) {
	t.Run("executes single request successfully", func(t *testing.T) {
		server := mockHTTPServer(t)
		defer server.Close()

		scheduler := NewScheduler(DefaultSchedulerConfig())
		_ = scheduler.Register(&EndpointInfo{
			Name:   "test",
			Method: "GET",
			Path:   "/test",
			Weight: 1,
		})

		rateLimiter := loadctrl.NewTokenBucketLimiter(100, 10)
		workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{MinSize: 1, MaxSize: 5})

		config := DefaultExecutorConfig()
		config.BaseURL = server.URL

		executor, _ := NewExecutor(config, scheduler, rateLimiter, workerPool, nil, nil)

		result, err := executor.ExecuteOnce(context.Background(), "test")

		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, http.StatusOK, result.StatusCode)
		assert.Equal(t, "test", result.EndpointName)
		assert.Greater(t, result.Latency, time.Duration(0))
	})

	t.Run("returns error for nonexistent endpoint", func(t *testing.T) {
		scheduler := NewScheduler(DefaultSchedulerConfig())
		rateLimiter := loadctrl.NewTokenBucketLimiter(100, 10)
		workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{MinSize: 1, MaxSize: 5})

		executor, _ := NewExecutor(DefaultExecutorConfig(), scheduler, rateLimiter, workerPool, nil, nil)

		_, err := executor.ExecuteOnce(context.Background(), "nonexistent")

		assert.ErrorIs(t, err, ErrEndpointNotFound)
	})

	t.Run("handles error responses", func(t *testing.T) {
		server := mockHTTPServer(t)
		defer server.Close()

		scheduler := NewScheduler(DefaultSchedulerConfig())
		_ = scheduler.Register(&EndpointInfo{
			Name:   "error-endpoint",
			Method: "GET",
			Path:   "/error",
			Weight: 1,
		})

		rateLimiter := loadctrl.NewTokenBucketLimiter(100, 10)
		workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{MinSize: 1, MaxSize: 5})

		config := DefaultExecutorConfig()
		config.BaseURL = server.URL

		executor, _ := NewExecutor(config, scheduler, rateLimiter, workerPool, nil, nil)

		result, _ := executor.ExecuteOnce(context.Background(), "error-endpoint")

		assert.False(t, result.Success)
		assert.Equal(t, http.StatusInternalServerError, result.StatusCode)
	})
}

func TestExecutor_Stats(t *testing.T) {
	t.Run("tracks request statistics", func(t *testing.T) {
		server := mockHTTPServer(t)
		defer server.Close()

		scheduler := NewScheduler(DefaultSchedulerConfig())
		_ = scheduler.RegisterAll([]*EndpointInfo{
			{Name: "ep1", Method: "GET", Path: "/ep1", Weight: 1},
			{Name: "ep2", Method: "GET", Path: "/ep2", Weight: 1},
		})

		rateLimiter := loadctrl.NewTokenBucketLimiter(1000, 100)
		workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{MinSize: 5, MaxSize: 10})

		config := DefaultExecutorConfig()
		config.BaseURL = server.URL

		executor, _ := NewExecutor(config, scheduler, rateLimiter, workerPool, nil, nil)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		_ = executor.Start(ctx)
		<-ctx.Done()
		executor.Stop()

		stats := executor.Stats()

		assert.Greater(t, stats.TotalRequests, int64(0))
		assert.Equal(t, stats.SuccessfulRequests+stats.FailedRequests, stats.TotalRequests)
		assert.Greater(t, stats.TotalLatency, int64(0))
		assert.Greater(t, stats.MinLatency, time.Duration(0))
		assert.GreaterOrEqual(t, stats.MaxLatency, stats.MinLatency)
	})

	t.Run("tracks status code distribution", func(t *testing.T) {
		server := mockHTTPServer(t)
		defer server.Close()

		scheduler := NewScheduler(DefaultSchedulerConfig())
		_ = scheduler.RegisterAll([]*EndpointInfo{
			{Name: "ok", Method: "GET", Path: "/ok", Weight: 1},
			{Name: "error", Method: "GET", Path: "/error", Weight: 1},
		})

		rateLimiter := loadctrl.NewTokenBucketLimiter(1000, 100)
		workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{MinSize: 2, MaxSize: 5})

		config := DefaultExecutorConfig()
		config.BaseURL = server.URL

		executor, _ := NewExecutor(config, scheduler, rateLimiter, workerPool, nil, nil)

		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()

		_ = executor.Start(ctx)
		<-ctx.Done()
		executor.Stop()

		stats := executor.Stats()

		// Should have both 200 and 500 status codes
		assert.Greater(t, len(stats.StatusCodes), 0)
	})

	t.Run("tracks per-endpoint statistics", func(t *testing.T) {
		server := mockHTTPServer(t)
		defer server.Close()

		scheduler := NewScheduler(DefaultSchedulerConfig())
		_ = scheduler.Register(&EndpointInfo{
			Name:   "single",
			Method: "GET",
			Path:   "/single",
			Weight: 1,
		})

		rateLimiter := loadctrl.NewTokenBucketLimiter(100, 10)
		workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{MinSize: 1, MaxSize: 2})

		config := DefaultExecutorConfig()
		config.BaseURL = server.URL

		executor, _ := NewExecutor(config, scheduler, rateLimiter, workerPool, nil, nil)

		// Execute a few requests
		for range 5 {
			_, _ = executor.ExecuteOnce(context.Background(), "single")
		}

		stats := executor.Stats()

		epStats, exists := stats.EndpointStats["single"]
		require.True(t, exists)
		assert.Equal(t, int64(5), epStats.TotalRequests)
	})

	t.Run("provides derived statistics", func(t *testing.T) {
		server := mockHTTPServer(t)
		defer server.Close()

		scheduler := NewScheduler(DefaultSchedulerConfig())
		_ = scheduler.Register(&EndpointInfo{
			Name:   "test",
			Method: "GET",
			Path:   "/test",
			Weight: 1,
		})

		rateLimiter := loadctrl.NewTokenBucketLimiter(100, 10)
		workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{MinSize: 1, MaxSize: 2})

		config := DefaultExecutorConfig()
		config.BaseURL = server.URL

		executor, _ := NewExecutor(config, scheduler, rateLimiter, workerPool, nil, nil)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		_ = executor.Start(ctx)
		<-ctx.Done()
		executor.Stop()

		stats := executor.Stats()

		// Test derived metrics
		avgLatency := stats.AverageLatency()
		successRate := stats.SuccessRate()
		rps := stats.RequestsPerSecond()

		assert.Greater(t, avgLatency, time.Duration(0))
		assert.GreaterOrEqual(t, successRate, 0.0)
		assert.LessOrEqual(t, successRate, 100.0)
		assert.Greater(t, rps, 0.0)
	})
}

func TestExecutor_ResetStats(t *testing.T) {
	server := mockHTTPServer(t)
	defer server.Close()

	scheduler := NewScheduler(DefaultSchedulerConfig())
	_ = scheduler.Register(&EndpointInfo{
		Name:   "test",
		Method: "GET",
		Path:   "/test",
		Weight: 1,
	})

	rateLimiter := loadctrl.NewTokenBucketLimiter(100, 10)
	workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{MinSize: 1, MaxSize: 2})

	config := DefaultExecutorConfig()
	config.BaseURL = server.URL

	executor, _ := NewExecutor(config, scheduler, rateLimiter, workerPool, nil, nil)

	// Execute some requests
	for range 5 {
		_, _ = executor.ExecuteOnce(context.Background(), "test")
	}

	stats := executor.Stats()
	assert.Equal(t, int64(5), stats.TotalRequests)

	// Reset stats
	executor.ResetStats()

	stats = executor.Stats()
	assert.Equal(t, int64(0), stats.TotalRequests)
	assert.Equal(t, int64(0), stats.SuccessfulRequests)
	assert.Equal(t, int64(0), stats.FailedRequests)
}

func TestExecutor_Callbacks(t *testing.T) {
	t.Run("calls OnRequest callback", func(t *testing.T) {
		server := mockHTTPServer(t)
		defer server.Close()

		var requestCalled atomic.Bool

		scheduler := NewScheduler(DefaultSchedulerConfig())
		_ = scheduler.Register(&EndpointInfo{
			Name:   "test",
			Method: "GET",
			Path:   "/test",
			Weight: 1,
		})

		rateLimiter := loadctrl.NewTokenBucketLimiter(100, 10)
		workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{MinSize: 1, MaxSize: 2})

		config := DefaultExecutorConfig()
		config.BaseURL = server.URL
		config.OnRequest = func(req *http.Request, endpointName string) {
			requestCalled.Store(true)
			assert.Equal(t, "test", endpointName)
		}

		executor, _ := NewExecutor(config, scheduler, rateLimiter, workerPool, nil, nil)
		_, _ = executor.ExecuteOnce(context.Background(), "test")

		assert.True(t, requestCalled.Load())
	})

	t.Run("calls OnResponse callback", func(t *testing.T) {
		server := mockHTTPServer(t)
		defer server.Close()

		var responseCalled atomic.Bool
		var receivedResult *ExecutionResult

		scheduler := NewScheduler(DefaultSchedulerConfig())
		_ = scheduler.Register(&EndpointInfo{
			Name:   "test",
			Method: "GET",
			Path:   "/test",
			Weight: 1,
		})

		rateLimiter := loadctrl.NewTokenBucketLimiter(100, 10)
		workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{MinSize: 1, MaxSize: 2})

		config := DefaultExecutorConfig()
		config.BaseURL = server.URL
		config.OnResponse = func(result *ExecutionResult) {
			responseCalled.Store(true)
			receivedResult = result
		}

		executor, _ := NewExecutor(config, scheduler, rateLimiter, workerPool, nil, nil)
		_, _ = executor.ExecuteOnce(context.Background(), "test")

		assert.True(t, responseCalled.Load())
		require.NotNil(t, receivedResult)
		assert.Equal(t, "test", receivedResult.EndpointName)
	})

	t.Run("calls OnError callback on failure", func(t *testing.T) {
		server := mockHTTPServer(t)
		defer server.Close()

		var errorCalled atomic.Bool

		scheduler := NewScheduler(DefaultSchedulerConfig())
		_ = scheduler.Register(&EndpointInfo{
			Name:   "error-endpoint",
			Method: "GET",
			Path:   "/error",
			Weight: 1,
		})

		rateLimiter := loadctrl.NewTokenBucketLimiter(100, 10)
		workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{MinSize: 1, MaxSize: 2})

		config := DefaultExecutorConfig()
		config.BaseURL = server.URL
		config.OnError = func(err error, endpointName string) {
			errorCalled.Store(true)
			assert.Equal(t, "error-endpoint", endpointName)
		}

		executor, _ := NewExecutor(config, scheduler, rateLimiter, workerPool, nil, nil)
		_, _ = executor.ExecuteOnce(context.Background(), "error-endpoint")

		assert.True(t, errorCalled.Load())
	})
}

func TestExecutor_Authentication(t *testing.T) {
	server := mockHTTPServer(t)
	defer server.Close()

	scheduler := NewScheduler(DefaultSchedulerConfig())
	_ = scheduler.Register(&EndpointInfo{
		Name:   "auth-required",
		Method: "GET",
		Path:   "/auth-required",
		Weight: 1,
	})

	rateLimiter := loadctrl.NewTokenBucketLimiter(100, 10)
	workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{MinSize: 1, MaxSize: 2})

	t.Run("request fails without auth token", func(t *testing.T) {
		config := DefaultExecutorConfig()
		config.BaseURL = server.URL
		// No auth token

		executor, _ := NewExecutor(config, scheduler, rateLimiter, workerPool, nil, nil)
		result, _ := executor.ExecuteOnce(context.Background(), "auth-required")

		assert.False(t, result.Success)
		assert.Equal(t, http.StatusUnauthorized, result.StatusCode)
	})

	t.Run("request succeeds with auth token", func(t *testing.T) {
		config := DefaultExecutorConfig()
		config.BaseURL = server.URL
		config.AuthToken = "test-token"

		executor, _ := NewExecutor(config, scheduler, rateLimiter, workerPool, nil, nil)
		result, _ := executor.ExecuteOnce(context.Background(), "auth-required")

		assert.True(t, result.Success)
		assert.Equal(t, http.StatusOK, result.StatusCode)
	})
}

func TestExecutor_Retries(t *testing.T) {
	t.Run("retries on failure", func(t *testing.T) {
		requestCount := &atomic.Int32{}

		// Server that fails first 2 requests, then succeeds
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count := requestCount.Add(1)
			if count < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status": "ok"}`))
		}))
		defer server.Close()

		scheduler := NewScheduler(DefaultSchedulerConfig())
		_ = scheduler.Register(&EndpointInfo{
			Name:   "retry-test",
			Method: "GET",
			Path:   "/test",
			Weight: 1,
		})

		rateLimiter := loadctrl.NewTokenBucketLimiter(100, 10)
		workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{MinSize: 1, MaxSize: 2})

		config := DefaultExecutorConfig()
		config.BaseURL = server.URL
		config.MaxRetries = 3
		config.RetryDelay = 10 * time.Millisecond

		executor, _ := NewExecutor(config, scheduler, rateLimiter, workerPool, nil, nil)
		result, _ := executor.ExecuteOnce(context.Background(), "retry-test")

		assert.True(t, result.Success)
		assert.Equal(t, 2, result.Retries)
	})

	t.Run("gives up after max retries", func(t *testing.T) {
		// Server that always fails
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		scheduler := NewScheduler(DefaultSchedulerConfig())
		_ = scheduler.Register(&EndpointInfo{
			Name:   "always-fail",
			Method: "GET",
			Path:   "/test",
			Weight: 1,
		})

		rateLimiter := loadctrl.NewTokenBucketLimiter(100, 10)
		workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{MinSize: 1, MaxSize: 2})

		config := DefaultExecutorConfig()
		config.BaseURL = server.URL
		config.MaxRetries = 2
		config.RetryDelay = 10 * time.Millisecond

		executor, _ := NewExecutor(config, scheduler, rateLimiter, workerPool, nil, nil)
		result, _ := executor.ExecuteOnce(context.Background(), "always-fail")

		assert.False(t, result.Success)
		assert.Equal(t, 2, result.Retries)
	})
}

func TestExecutor_Concurrency(t *testing.T) {
	t.Run("handles concurrent requests safely", func(t *testing.T) {
		requestCount := &atomic.Int32{}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount.Add(1)
			time.Sleep(5 * time.Millisecond) // Simulate processing
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status": "ok"}`))
		}))
		defer server.Close()

		scheduler := NewScheduler(DefaultSchedulerConfig())
		for i := range 5 {
			_ = scheduler.Register(&EndpointInfo{
				Name:   "ep" + string(rune('0'+i)),
				Method: "GET",
				Path:   "/ep" + string(rune('0'+i)),
				Weight: 1,
			})
		}

		rateLimiter := loadctrl.NewTokenBucketLimiter(1000, 100)
		workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{MinSize: 5, MaxSize: 20})

		config := DefaultExecutorConfig()
		config.BaseURL = server.URL

		executor, _ := NewExecutor(config, scheduler, rateLimiter, workerPool, nil, nil)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		_ = executor.Start(ctx)
		<-ctx.Done()
		executor.Stop()

		stats := executor.Stats()

		// Should have processed many requests concurrently
		assert.Greater(t, stats.TotalRequests, int64(10))
		// All requests should be accounted for (success + failed = total)
		assert.Equal(t, stats.TotalRequests, stats.SuccessfulRequests+stats.FailedRequests)
	})
}

func TestExecutor_WeightDistribution(t *testing.T) {
	t.Run("distributes requests according to weights", func(t *testing.T) {
		requestCounts := sync.Map{}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			counter, _ := requestCounts.LoadOrStore(r.URL.Path, &atomic.Int32{})
			counter.(*atomic.Int32).Add(1)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		scheduler := NewScheduler(DefaultSchedulerConfig())
		// Register with 1:10 weight ratio - high should get ~10x more requests
		_ = scheduler.RegisterAll([]*EndpointInfo{
			{Name: "low", Method: "GET", Path: "/low", Weight: 1},
			{Name: "high", Method: "GET", Path: "/high", Weight: 10},
		})

		rateLimiter := loadctrl.NewTokenBucketLimiter(1000, 100)
		workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{MinSize: 5, MaxSize: 10})

		config := DefaultExecutorConfig()
		config.BaseURL = server.URL

		executor, _ := NewExecutor(config, scheduler, rateLimiter, workerPool, nil, nil)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		_ = executor.Start(ctx)
		<-ctx.Done()
		executor.Stop()

		// Get counts
		lowCountI, _ := requestCounts.Load("/low")
		highCountI, _ := requestCounts.Load("/high")

		lowCount := int64(0)
		highCount := int64(0)
		if lowCountI != nil {
			lowCount = int64(lowCountI.(*atomic.Int32).Load())
		}
		if highCountI != nil {
			highCount = int64(highCountI.(*atomic.Int32).Load())
		}

		// High should have significantly more requests
		if lowCount > 0 {
			ratio := float64(highCount) / float64(lowCount)
			// Allow wide tolerance due to randomness, but ratio should trend towards 10
			assert.Greater(t, ratio, 3.0, "high weight endpoint should receive more requests")
		}
	})
}

func TestExecutor_Close(t *testing.T) {
	t.Run("close stops executor and releases resources", func(t *testing.T) {
		scheduler := NewScheduler(DefaultSchedulerConfig())
		rateLimiter := loadctrl.NewTokenBucketLimiter(100, 10)
		workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{MinSize: 1, MaxSize: 5})

		executor, _ := NewExecutor(DefaultExecutorConfig(), scheduler, rateLimiter, workerPool, nil, nil)

		_ = executor.Start(context.Background())
		assert.True(t, executor.IsRunning())

		executor.Close()
		assert.False(t, executor.IsRunning())
	})

	t.Run("close is idempotent", func(t *testing.T) {
		scheduler := NewScheduler(DefaultSchedulerConfig())
		rateLimiter := loadctrl.NewTokenBucketLimiter(100, 10)
		workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{MinSize: 1, MaxSize: 5})

		executor, _ := NewExecutor(DefaultExecutorConfig(), scheduler, rateLimiter, workerPool, nil, nil)

		executor.Close()
		executor.Close() // Should not panic
		executor.Close()
	})

	t.Run("start after close returns error", func(t *testing.T) {
		scheduler := NewScheduler(DefaultSchedulerConfig())
		rateLimiter := loadctrl.NewTokenBucketLimiter(100, 10)
		workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{MinSize: 1, MaxSize: 5})

		executor, _ := NewExecutor(DefaultExecutorConfig(), scheduler, rateLimiter, workerPool, nil, nil)

		executor.Close()

		err := executor.Start(context.Background())
		assert.ErrorIs(t, err, ErrExecutorClosed)
	})
}

func TestExecutor_WithCircuitBoard(t *testing.T) {
	t.Run("uses circuit board for execution when enabled", func(t *testing.T) {
		server := mockHTTPServer(t)
		defer server.Close()

		// Create a simple parameter pool implementation
		paramPool := &mockParameterPool{values: make(map[circuit.SemanticType]any)}

		// Create dependency graph
		graph := circuit.NewDependencyGraph()

		// Create circuit board
		boardConfig := &circuit.CircuitBoardConfig{
			BaseURL:        server.URL,
			EnableAutoHeal: false,
		}
		board := circuit.NewCircuitBoard(boardConfig, paramPool, graph, nil, &http.Client{})

		// Add endpoint to board
		unit := &circuit.EndpointUnit{
			Name:   "board-test",
			Method: "GET",
			Path:   "/board-test",
		}
		board.AddEndpoint(unit)
		board.SetRequestBuilder(circuit.NewSimpleRequestBuilder(paramPool, server.URL))

		// Create scheduler and register endpoint
		scheduler := NewScheduler(DefaultSchedulerConfig())
		_ = scheduler.Register(&EndpointInfo{
			Name:   "board-test",
			Method: "GET",
			Path:   "/board-test",
			Weight: 1,
			Unit:   unit,
		})

		rateLimiter := loadctrl.NewTokenBucketLimiter(100, 10)
		workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{MinSize: 1, MaxSize: 2})

		config := DefaultExecutorConfig()
		config.BaseURL = server.URL
		config.EnableCircuitBoard = true

		executor, _ := NewExecutor(config, scheduler, rateLimiter, workerPool, board, nil)

		result, err := executor.ExecuteOnce(context.Background(), "board-test")

		require.NoError(t, err)
		assert.True(t, result.Success)
	})
}

// mockParameterPool is a simple implementation for testing
type mockParameterPool struct {
	values map[circuit.SemanticType]any
	mu     sync.RWMutex
}

func (p *mockParameterPool) Add(semantic circuit.SemanticType, value any, _ circuit.ValueSource) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.values[semantic] = value
}

func (p *mockParameterPool) Get(semantic circuit.SemanticType) (circuit.PoolValue, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if v, ok := p.values[semantic]; ok {
		return &mockPoolValue{data: v}, nil
	}
	return nil, ErrEndpointNotFound
}

func (p *mockParameterPool) Size(semantic circuit.SemanticType) int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if _, ok := p.values[semantic]; ok {
		return 1
	}
	return 0
}

type mockPoolValue struct {
	data any
}

func (v *mockPoolValue) GetData() any {
	return v.data
}

// Benchmark tests
func BenchmarkExecutor_ExecuteOnce(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	scheduler := NewScheduler(DefaultSchedulerConfig())
	_ = scheduler.Register(&EndpointInfo{
		Name:   "bench",
		Method: "GET",
		Path:   "/bench",
		Weight: 1,
	})

	rateLimiter := loadctrl.NewTokenBucketLimiter(100000, 10000)
	workerPool := loadctrl.NewWorkerPool(loadctrl.WorkerPoolConfig{MinSize: 1, MaxSize: 10})

	config := DefaultExecutorConfig()
	config.BaseURL = server.URL

	executor, _ := NewExecutor(config, scheduler, rateLimiter, workerPool, nil, nil)

	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		_, _ = executor.ExecuteOnce(ctx, "bench")
	}
}
