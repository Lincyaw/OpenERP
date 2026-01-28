// Package executor provides request building and execution functionality for the load generator.
package executor

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/example/erp/tools/loadgen/internal/circuit"
	"github.com/example/erp/tools/loadgen/internal/loadctrl"
)

// Executor errors.
var (
	// ErrExecutorClosed is returned when operations are attempted on a closed executor.
	ErrExecutorClosed = errors.New("executor: executor is closed")
	// ErrExecutorNotRunning is returned when the executor is not running.
	ErrExecutorNotRunning = errors.New("executor: executor is not running")
	// ErrExecutorAlreadyRunning is returned when trying to start an already running executor.
	ErrExecutorAlreadyRunning = errors.New("executor: executor is already running")
)

// ExecutorConfig holds configuration for the Executor.
type ExecutorConfig struct {
	// BaseURL is the base URL for API requests.
	BaseURL string

	// Timeout is the default request timeout.
	// Default: 30s
	Timeout time.Duration

	// MaxRetries is the maximum number of retries for failed requests.
	// Default: 0 (no retries)
	MaxRetries int

	// RetryDelay is the delay between retries.
	// Default: 1s
	RetryDelay time.Duration

	// EnableCircuitBoard enables automatic dependency resolution via CircuitBoard.
	// Default: true
	EnableCircuitBoard bool

	// OnRequest is called before each request is sent.
	OnRequest func(req *http.Request, endpointName string)

	// OnResponse is called after each response is received.
	OnResponse func(result *ExecutionResult)

	// OnError is called when an error occurs.
	OnError func(err error, endpointName string)

	// DefaultHeaders are headers added to all requests.
	DefaultHeaders map[string]string

	// AuthToken is the bearer token for authentication.
	AuthToken string
}

// DefaultExecutorConfig returns default executor configuration.
func DefaultExecutorConfig() ExecutorConfig {
	return ExecutorConfig{
		Timeout:            30 * time.Second,
		MaxRetries:         0,
		RetryDelay:         time.Second,
		EnableCircuitBoard: true,
		DefaultHeaders:     make(map[string]string),
	}
}

// ExecutionResult holds the result of executing a request.
type ExecutionResult struct {
	// EndpointName is the name of the endpoint that was executed.
	EndpointName string

	// Method is the HTTP method used.
	Method string

	// Path is the URL path.
	Path string

	// StatusCode is the HTTP response status code.
	StatusCode int

	// Latency is the time taken for the request.
	Latency time.Duration

	// Error holds any error that occurred.
	Error error

	// Success indicates if the request was successful (2xx status).
	Success bool

	// ResponseSize is the size of the response body in bytes.
	ResponseSize int64

	// Retries is the number of retries attempted.
	Retries int

	// Timestamp is when the request was made.
	Timestamp time.Time

	// ResponseBody holds the raw response body (may be truncated).
	ResponseBody []byte
}

// ExecutorStats holds statistics about the executor.
type ExecutorStats struct {
	// TotalRequests is the total number of requests made.
	TotalRequests int64

	// SuccessfulRequests is the number of successful requests.
	SuccessfulRequests int64

	// FailedRequests is the number of failed requests.
	FailedRequests int64

	// TotalLatency is the cumulative latency in nanoseconds.
	TotalLatency int64

	// MinLatency is the minimum request latency.
	MinLatency time.Duration

	// MaxLatency is the maximum request latency.
	MaxLatency time.Duration

	// TotalBytes is the total bytes received.
	TotalBytes int64

	// RateLimiterStats contains rate limiter statistics.
	RateLimiterStats loadctrl.RateLimiterStats

	// WorkerPoolStats contains worker pool statistics.
	WorkerPoolStats loadctrl.WorkerPoolStats

	// SchedulerStats contains scheduler statistics.
	SchedulerStats SchedulerStats

	// StatusCodes tracks response status code distribution.
	StatusCodes map[int]int64

	// EndpointStats tracks per-endpoint statistics.
	EndpointStats map[string]*EndpointExecutionStats

	// StartTime is when the executor started.
	StartTime time.Time

	// Duration is how long the executor has been running.
	Duration time.Duration
}

// EndpointExecutionStats holds per-endpoint statistics.
type EndpointExecutionStats struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	TotalLatency       int64
	MinLatency         time.Duration
	MaxLatency         time.Duration
}

// AverageLatency returns the average latency for an endpoint.
func (s *EndpointExecutionStats) AverageLatency() time.Duration {
	if s.TotalRequests == 0 {
		return 0
	}
	return time.Duration(s.TotalLatency / s.TotalRequests)
}

// SuccessRate returns the success rate as a percentage (0-100).
func (s *EndpointExecutionStats) SuccessRate() float64 {
	if s.TotalRequests == 0 {
		return 0
	}
	return float64(s.SuccessfulRequests) / float64(s.TotalRequests) * 100
}

// Executor orchestrates load generation by integrating:
// - Scheduler: Selects endpoints based on weights
// - RateLimiter: Controls request rate
// - WorkerPool: Manages concurrent workers
// - CircuitBoard: Handles dependencies and request building
// - HTTPClient: Executes HTTP requests
//
// Thread Safety: Safe for concurrent use.
type Executor struct {
	config ExecutorConfig

	// Components
	scheduler   *Scheduler
	rateLimiter loadctrl.RateLimiter
	workerPool  *loadctrl.WorkerPool
	board       *circuit.CircuitBoard
	httpClient  *http.Client
	builder     *RequestBuilder

	// State
	mu        sync.RWMutex
	isRunning atomic.Bool
	isClosed  atomic.Bool
	stopCh    chan struct{}
	startTime time.Time
	ctx       context.Context
	cancel    context.CancelFunc

	// Statistics
	totalRequests      atomic.Int64
	successfulRequests atomic.Int64
	failedRequests     atomic.Int64
	totalLatency       atomic.Int64
	minLatency         atomic.Int64 // in nanoseconds
	maxLatency         atomic.Int64 // in nanoseconds
	totalBytes         atomic.Int64
	statusCodes        sync.Map // map[int]*atomic.Int64
	endpointStats      sync.Map // map[string]*EndpointExecutionStats
	endpointStatsMu    sync.Map // map[string]*sync.RWMutex for per-endpoint locking
}

// NewExecutor creates a new executor with the given components.
func NewExecutor(
	config ExecutorConfig,
	scheduler *Scheduler,
	rateLimiter loadctrl.RateLimiter,
	workerPool *loadctrl.WorkerPool,
	board *circuit.CircuitBoard,
	httpClient *http.Client,
) (*Executor, error) {
	if scheduler == nil {
		return nil, errors.New("executor: scheduler is required")
	}
	if rateLimiter == nil {
		return nil, errors.New("executor: rate limiter is required")
	}
	if workerPool == nil {
		return nil, errors.New("executor: worker pool is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: config.Timeout,
		}
	}

	// Apply defaults
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.DefaultHeaders == nil {
		config.DefaultHeaders = make(map[string]string)
	}

	e := &Executor{
		config:      config,
		scheduler:   scheduler,
		rateLimiter: rateLimiter,
		workerPool:  workerPool,
		board:       board,
		httpClient:  httpClient,
		stopCh:      make(chan struct{}),
	}

	return e, nil
}

// SetRequestBuilder sets the request builder for the executor.
func (e *Executor) SetRequestBuilder(builder *RequestBuilder) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.builder = builder
}

// Start begins the execution loop.
// The executor continuously:
// 1. Acquires rate limit token
// 2. Selects an endpoint via the scheduler
// 3. Submits the request to the worker pool
func (e *Executor) Start(ctx context.Context) error {
	if e.isClosed.Load() {
		return ErrExecutorClosed
	}
	if e.isRunning.Swap(true) {
		return ErrExecutorAlreadyRunning
	}

	e.mu.Lock()
	e.startTime = time.Now()
	e.stopCh = make(chan struct{})
	e.ctx, e.cancel = context.WithCancel(ctx)
	e.mu.Unlock()

	// Start the worker pool
	e.workerPool.Start(e.ctx)

	// Start the main execution loop
	go e.runLoop(e.ctx)

	return nil
}

// Stop stops the execution loop and waits for pending requests to complete.
func (e *Executor) Stop() {
	if !e.isRunning.Swap(false) {
		return // Not running
	}

	// Signal stop
	e.mu.Lock()
	if e.cancel != nil {
		e.cancel()
	}
	close(e.stopCh)
	e.mu.Unlock()

	// Stop worker pool
	e.workerPool.Stop()
}

// Close stops the executor and releases resources.
func (e *Executor) Close() {
	if e.isClosed.Swap(true) {
		return // Already closed
	}

	e.Stop()

	// Close circuit board if available
	if e.board != nil {
		e.board.Close()
	}
}

// IsRunning returns whether the executor is currently running.
func (e *Executor) IsRunning() bool {
	return e.isRunning.Load()
}

// runLoop is the main execution loop.
func (e *Executor) runLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		default:
			// Acquire rate limit token
			if err := e.rateLimiter.Acquire(ctx); err != nil {
				// Context cancelled or rate limiter error
				if ctx.Err() != nil {
					return
				}
				// Log rate limiter errors via callback
				if e.config.OnError != nil {
					e.config.OnError(err, "[rate_limiter]")
				}
				continue
			}

			// Select an endpoint
			endpoint, err := e.scheduler.Select()
			if err != nil {
				if e.config.OnError != nil {
					e.config.OnError(err, "[scheduler]")
				}
				continue
			}

			// Submit task to worker pool
			task := e.createTask(ctx, endpoint)
			if !e.workerPool.Submit(task) {
				// Queue is full, try with blocking
				if err := e.workerPool.SubmitWait(ctx, task); err != nil {
					// Context cancelled
					if ctx.Err() != nil {
						return
					}
				}
			}
		}
	}
}

// createTask creates a task function for the worker pool.
func (e *Executor) createTask(ctx context.Context, endpoint *EndpointInfo) loadctrl.Task {
	return func(taskCtx context.Context) error {
		result := e.executeEndpoint(taskCtx, endpoint)

		// Update statistics
		e.updateStats(result)

		// Call callback if configured
		if e.config.OnResponse != nil {
			e.config.OnResponse(result)
		}

		if result.Error != nil && e.config.OnError != nil {
			e.config.OnError(result.Error, endpoint.Name)
		}

		return result.Error
	}
}

// executeEndpoint executes a single endpoint request.
func (e *Executor) executeEndpoint(ctx context.Context, endpoint *EndpointInfo) *ExecutionResult {
	result := &ExecutionResult{
		EndpointName: endpoint.Name,
		Method:       endpoint.Method,
		Path:         endpoint.Path,
		Timestamp:    time.Now(),
	}

	startTime := time.Now()

	// Try with retries
	maxRetries := e.config.MaxRetries
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			result.Retries = attempt
			// Wait before retry
			select {
			case <-ctx.Done():
				result.Error = ctx.Err()
				result.Latency = time.Since(startTime)
				return result
			case <-time.After(e.config.RetryDelay):
			}
		}

		// Execute the request
		statusCode, responseBody, err := e.doRequest(ctx, endpoint)
		result.StatusCode = statusCode
		result.ResponseBody = responseBody
		result.ResponseSize = int64(len(responseBody))

		if err != nil {
			lastErr = err
			continue
		}

		// Check for success (2xx status)
		if statusCode >= 200 && statusCode < 300 {
			result.Success = true
			result.Latency = time.Since(startTime)
			return result
		}

		// Non-2xx status is an error
		lastErr = fmt.Errorf("HTTP %d", statusCode)
	}

	// All retries failed
	result.Error = lastErr
	result.Latency = time.Since(startTime)
	return result
}

// doRequest performs the HTTP request.
func (e *Executor) doRequest(ctx context.Context, endpoint *EndpointInfo) (int, []byte, error) {
	var req *http.Request
	var err error

	// Build the request
	if e.config.EnableCircuitBoard && e.board != nil && endpoint.Unit != nil {
		// Use circuit board for dependency resolution
		execResult, execErr := e.board.Execute(ctx, endpoint.Name)
		if execErr != nil {
			return 0, nil, fmt.Errorf("circuit board execution failed: %w", execErr)
		}
		return execResult.StatusCode, execResult.ResponseBody, nil
	} else if e.builder != nil && endpoint.Unit != nil {
		// Use request builder
		req, err = e.builder.BuildRequest(endpoint.Unit)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to build request: %w", err)
		}
		// Ensure context is set for request cancellation
		req = req.WithContext(ctx)
	} else {
		// Build simple request
		url := e.config.BaseURL + endpoint.Path
		req, err = http.NewRequestWithContext(ctx, endpoint.Method, url, nil)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to create request: %w", err)
		}
	}

	// Add base URL if not present
	if req.URL.Host == "" && e.config.BaseURL != "" {
		fullURL := e.config.BaseURL + req.URL.Path
		if req.URL.RawQuery != "" {
			fullURL += "?" + req.URL.RawQuery
		}
		req, err = http.NewRequestWithContext(ctx, req.Method, fullURL, req.Body)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to rebuild request with base URL: %w", err)
		}
	}

	// Add default headers
	for k, v := range e.config.DefaultHeaders {
		req.Header.Set(k, v)
	}

	// Add auth token
	if e.config.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+e.config.AuthToken)
	}

	// Set content type for POST/PUT/PATCH if not set
	if req.Body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Call request callback
	if e.config.OnRequest != nil {
		e.config.OnRequest(req, endpoint.Name)
	}

	// Execute request
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body (with limit)
	const maxBodySize = 10 * 1024 * 1024 // 10MB
	body := make([]byte, 0, 1024)
	buf := make([]byte, 1024)
	totalRead := int64(0)

	for totalRead < maxBodySize {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			body = append(body, buf[:n]...)
			totalRead += int64(n)
		}
		if readErr != nil {
			break
		}
	}

	return resp.StatusCode, body, nil
}

// updateStats updates executor statistics with the result.
func (e *Executor) updateStats(result *ExecutionResult) {
	e.totalRequests.Add(1)

	if result.Success {
		e.successfulRequests.Add(1)
	} else {
		e.failedRequests.Add(1)
	}

	latencyNs := result.Latency.Nanoseconds()
	e.totalLatency.Add(latencyNs)

	// Update min latency (use compare-and-swap for thread safety)
	for {
		current := e.minLatency.Load()
		if current != 0 && current <= latencyNs {
			break
		}
		if e.minLatency.CompareAndSwap(current, latencyNs) {
			break
		}
	}

	// Update max latency
	for {
		current := e.maxLatency.Load()
		if current >= latencyNs {
			break
		}
		if e.maxLatency.CompareAndSwap(current, latencyNs) {
			break
		}
	}

	e.totalBytes.Add(result.ResponseSize)

	// Update status code counter
	if result.StatusCode > 0 {
		counter, _ := e.statusCodes.LoadOrStore(result.StatusCode, &atomic.Int64{})
		counter.(*atomic.Int64).Add(1)
	}

	// Update per-endpoint stats
	e.updateEndpointStats(result)
}

// updateEndpointStats updates per-endpoint statistics.
func (e *Executor) updateEndpointStats(result *ExecutionResult) {
	// Get or create mutex for this endpoint
	mutexI, _ := e.endpointStatsMu.LoadOrStore(result.EndpointName, &sync.RWMutex{})
	mu := mutexI.(*sync.RWMutex)

	mu.Lock()
	defer mu.Unlock()

	// Get or create stats inside the lock to avoid race conditions
	statsI, _ := e.endpointStats.LoadOrStore(result.EndpointName, &EndpointExecutionStats{})
	stats := statsI.(*EndpointExecutionStats)

	stats.TotalRequests++
	if result.Success {
		stats.SuccessfulRequests++
	} else {
		stats.FailedRequests++
	}
	stats.TotalLatency += result.Latency.Nanoseconds()

	if stats.MinLatency == 0 || result.Latency < stats.MinLatency {
		stats.MinLatency = result.Latency
	}
	if result.Latency > stats.MaxLatency {
		stats.MaxLatency = result.Latency
	}
}

// Stats returns current executor statistics.
func (e *Executor) Stats() ExecutorStats {
	e.mu.RLock()
	startTime := e.startTime
	e.mu.RUnlock()

	stats := ExecutorStats{
		TotalRequests:      e.totalRequests.Load(),
		SuccessfulRequests: e.successfulRequests.Load(),
		FailedRequests:     e.failedRequests.Load(),
		TotalLatency:       e.totalLatency.Load(),
		MinLatency:         time.Duration(e.minLatency.Load()),
		MaxLatency:         time.Duration(e.maxLatency.Load()),
		TotalBytes:         e.totalBytes.Load(),
		StatusCodes:        make(map[int]int64),
		EndpointStats:      make(map[string]*EndpointExecutionStats),
		StartTime:          startTime,
	}

	if !startTime.IsZero() {
		stats.Duration = time.Since(startTime)
	}

	// Collect status code distribution
	e.statusCodes.Range(func(key, value any) bool {
		if counter, ok := value.(*atomic.Int64); ok {
			stats.StatusCodes[key.(int)] = counter.Load()
		}
		return true
	})

	// Collect endpoint stats
	e.endpointStats.Range(func(key, value any) bool {
		if epStats, ok := value.(*EndpointExecutionStats); ok {
			// Get mutex for this endpoint
			if mutexI, ok := e.endpointStatsMu.Load(key); ok {
				mu := mutexI.(*sync.RWMutex)
				mu.RLock()
				statsCopy := &EndpointExecutionStats{
					TotalRequests:      epStats.TotalRequests,
					SuccessfulRequests: epStats.SuccessfulRequests,
					FailedRequests:     epStats.FailedRequests,
					TotalLatency:       epStats.TotalLatency,
					MinLatency:         epStats.MinLatency,
					MaxLatency:         epStats.MaxLatency,
				}
				mu.RUnlock()
				stats.EndpointStats[key.(string)] = statsCopy
			}
		}
		return true
	})

	// Collect component stats
	if e.rateLimiter != nil {
		stats.RateLimiterStats = e.rateLimiter.Stats()
	}
	if e.workerPool != nil {
		stats.WorkerPoolStats = e.workerPool.Stats()
	}
	if e.scheduler != nil {
		stats.SchedulerStats = e.scheduler.Stats()
	}

	return stats
}

// AverageLatency returns the average request latency.
func (s *ExecutorStats) AverageLatency() time.Duration {
	if s.TotalRequests == 0 {
		return 0
	}
	return time.Duration(s.TotalLatency / s.TotalRequests)
}

// SuccessRate returns the success rate as a percentage (0-100).
func (s *ExecutorStats) SuccessRate() float64 {
	if s.TotalRequests == 0 {
		return 0
	}
	return float64(s.SuccessfulRequests) / float64(s.TotalRequests) * 100
}

// RequestsPerSecond returns the requests per second.
func (s *ExecutorStats) RequestsPerSecond() float64 {
	if s.Duration == 0 {
		return 0
	}
	return float64(s.TotalRequests) / s.Duration.Seconds()
}

// GetScheduler returns the scheduler.
func (e *Executor) GetScheduler() *Scheduler {
	return e.scheduler
}

// GetRateLimiter returns the rate limiter.
func (e *Executor) GetRateLimiter() loadctrl.RateLimiter {
	return e.rateLimiter
}

// GetWorkerPool returns the worker pool.
func (e *Executor) GetWorkerPool() *loadctrl.WorkerPool {
	return e.workerPool
}

// GetCircuitBoard returns the circuit board.
func (e *Executor) GetCircuitBoard() *circuit.CircuitBoard {
	return e.board
}

// ExecuteOnce executes a single request for the given endpoint (synchronously).
// This is useful for testing or one-off requests outside the main loop.
func (e *Executor) ExecuteOnce(ctx context.Context, endpointName string) (*ExecutionResult, error) {
	endpoint, err := e.scheduler.GetEndpoint(endpointName)
	if err != nil {
		return nil, err
	}

	result := e.executeEndpoint(ctx, endpoint)
	e.updateStats(result)

	// Call callbacks
	if e.config.OnResponse != nil {
		e.config.OnResponse(result)
	}

	if result.Error != nil && e.config.OnError != nil {
		e.config.OnError(result.Error, endpointName)
	}

	return result, result.Error
}

// ResetStats resets all statistics.
func (e *Executor) ResetStats() {
	e.totalRequests.Store(0)
	e.successfulRequests.Store(0)
	e.failedRequests.Store(0)
	e.totalLatency.Store(0)
	e.minLatency.Store(0)
	e.maxLatency.Store(0)
	e.totalBytes.Store(0)

	// Clear status codes
	e.statusCodes.Range(func(key, _ any) bool {
		e.statusCodes.Delete(key)
		return true
	})

	// Clear endpoint stats
	e.endpointStats.Range(func(key, _ any) bool {
		e.endpointStats.Delete(key)
		e.endpointStatsMu.Delete(key)
		return true
	})

	e.mu.Lock()
	e.startTime = time.Now()
	e.mu.Unlock()
}
