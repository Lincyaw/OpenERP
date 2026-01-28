// Package circuit provides circuit-board-like components for the load generator.
// This file implements the CircuitBoard - the main controller that orchestrates
// endpoint execution with automatic dependency resolution and self-healing capabilities.
package circuit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Errors returned by CircuitBoard operations.
var (
	// ErrBoardClosed is returned when operations are attempted on a closed board.
	ErrBoardClosed = errors.New("circuit board: board is closed")
	// ErrEndpointNotFoundOnBoard is returned when the requested endpoint is not found.
	ErrEndpointNotFoundOnBoard = errors.New("circuit board: endpoint not found")
	// ErrMissingDependency is returned when a required dependency cannot be satisfied.
	ErrMissingDependency = errors.New("circuit board: missing dependency")
	// ErrExecutionFailed is returned when endpoint execution fails.
	ErrExecutionFailed = errors.New("circuit board: execution failed")
	// ErrNoProducerAvailable is returned when no producer can satisfy a semantic type.
	ErrNoProducerAvailable = errors.New("circuit board: no producer available for semantic type")
	// ErrMaxRetryExceeded is returned when auto-healing retries are exhausted.
	ErrMaxRetryExceeded = errors.New("circuit board: max retry exceeded for auto-healing")
)

// PoolValue represents a value retrieved from the parameter pool.
// This interface breaks the import cycle between circuit and pool packages.
type PoolValue interface {
	// GetData returns the underlying value data.
	GetData() any
}

// ValueSource describes the origin of a parameter value.
type ValueSource struct {
	// Endpoint is the API endpoint that produced this value.
	Endpoint string
	// ResponseField is the JSONPath to the field in the response.
	ResponseField string
	// RequestID is the ID of the request that produced this value.
	RequestID string
}

// ParameterPool is an interface for parameter storage and retrieval.
// This interface breaks the import cycle between circuit and pool packages.
type ParameterPool interface {
	// Add adds a value to the pool for the given semantic type.
	Add(semantic SemanticType, value any, source ValueSource)

	// Get retrieves a value for the given semantic type.
	// Returns an error if no value is available.
	Get(semantic SemanticType) (PoolValue, error)

	// Size returns the number of values for the given semantic type.
	Size(semantic SemanticType) int
}

// CircuitBoardConfig holds configuration for the CircuitBoard.
type CircuitBoardConfig struct {
	// BaseURL is the base URL for API requests.
	BaseURL string

	// DefaultTimeout is the default timeout for HTTP requests.
	DefaultTimeout time.Duration

	// MaxAutoHealRetries is the maximum number of retries for auto-healing.
	// Default: 3
	MaxAutoHealRetries int

	// EnableAutoHeal enables automatic producer triggering when parameters are missing.
	// Default: true
	EnableAutoHeal bool

	// ResponseExtractors configures how to extract values from responses.
	// Maps endpoint name to extraction config.
	ResponseExtractors map[string][]ResponseExtractor
}

// DefaultCircuitBoardConfig returns the default configuration.
func DefaultCircuitBoardConfig() CircuitBoardConfig {
	return CircuitBoardConfig{
		DefaultTimeout:     30 * time.Second,
		MaxAutoHealRetries: 3,
		EnableAutoHeal:     true,
		ResponseExtractors: make(map[string][]ResponseExtractor),
	}
}

// ResponseExtractor defines how to extract values from API responses.
type ResponseExtractor struct {
	// JSONPath is the path to extract the value (e.g., "$.data.id", "data.id").
	JSONPath string

	// SemanticType is the semantic classification for the extracted value.
	SemanticType SemanticType

	// ValueSource describes the source for pool storage.
	Source ValueSource
}

// RequestBuilder is an interface for building HTTP requests.
type RequestBuilder interface {
	// BuildRequestForEndpoint builds an HTTP request for the given endpoint.
	BuildRequestForEndpoint(unit *EndpointUnit) (*http.Request, error)
}

// HTTPClient is an interface for executing HTTP requests.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// CircuitBoard is the main controller that orchestrates endpoint execution.
// It integrates EndpointUnits, ParameterPool, and DependencyGraph to provide:
// - Automatic dependency resolution
// - Self-healing parameter pool (auto-triggers producers when values are missing)
// - Response value extraction and storage
// - Execution statistics tracking
//
// Thread Safety: All public methods are safe for concurrent use.
type CircuitBoard struct {
	config  CircuitBoardConfig
	pool    ParameterPool
	graph   *DependencyGraph
	guard   *ProducerChainGuard
	client  HTTPClient
	builder RequestBuilder

	// mu protects access to mutable state
	mu sync.RWMutex

	// units maps endpoint name to unit with statistics
	units map[string]*UnitWithStats

	// closed indicates if the board has been closed
	closed atomic.Bool

	// Statistics
	stats BoardStats
}

// UnitWithStats wraps an EndpointUnit with execution statistics.
type UnitWithStats struct {
	*EndpointUnit
	Stats EndpointStats
	mu    sync.RWMutex
}

// EndpointStats holds execution statistics for an endpoint.
type EndpointStats struct {
	// TotalExecutions is the total number of executions.
	TotalExecutions int64

	// SuccessfulExecutions is the number of successful executions.
	SuccessfulExecutions int64

	// FailedExecutions is the number of failed executions.
	FailedExecutions int64

	// TotalLatency is the cumulative latency in nanoseconds.
	TotalLatency int64

	// MinLatency is the minimum execution latency.
	MinLatency time.Duration

	// MaxLatency is the maximum execution latency.
	MaxLatency time.Duration

	// LastExecutedAt is when the endpoint was last executed.
	LastExecutedAt time.Time

	// LastError is the last error message if any.
	LastError string

	// AutoHealTriggers is how many times this endpoint was triggered for auto-healing.
	AutoHealTriggers int64

	// ValuesProduced is how many values this endpoint has produced.
	ValuesProduced int64
}

// AverageLatency returns the average execution latency.
func (s *EndpointStats) AverageLatency() time.Duration {
	if s.TotalExecutions == 0 {
		return 0
	}
	return time.Duration(s.TotalLatency / s.TotalExecutions)
}

// SuccessRate returns the success rate as a percentage (0-100).
func (s *EndpointStats) SuccessRate() float64 {
	if s.TotalExecutions == 0 {
		return 0
	}
	return float64(s.SuccessfulExecutions) / float64(s.TotalExecutions) * 100
}

// BoardStats holds overall statistics for the circuit board.
type BoardStats struct {
	// TotalExecutions is the total number of endpoint executions.
	TotalExecutions int64

	// SuccessfulExecutions is the total successful executions.
	SuccessfulExecutions int64

	// FailedExecutions is the total failed executions.
	FailedExecutions int64

	// AutoHealAttempts is the number of auto-healing attempts.
	AutoHealAttempts int64

	// AutoHealSuccesses is the number of successful auto-healing attempts.
	AutoHealSuccesses int64

	// AutoHealFailures is the number of failed auto-healing attempts.
	AutoHealFailures int64

	// ValueExtracted is the total number of values extracted from responses.
	ValueExtracted int64
}

// NewCircuitBoard creates a new circuit board with the given components.
func NewCircuitBoard(
	config *CircuitBoardConfig,
	paramPool ParameterPool,
	graph *DependencyGraph,
	guard *ProducerChainGuard,
	client HTTPClient,
) *CircuitBoard {
	cfg := DefaultCircuitBoardConfig()
	if config != nil {
		if config.BaseURL != "" {
			cfg.BaseURL = config.BaseURL
		}
		if config.DefaultTimeout > 0 {
			cfg.DefaultTimeout = config.DefaultTimeout
		}
		if config.MaxAutoHealRetries > 0 {
			cfg.MaxAutoHealRetries = config.MaxAutoHealRetries
		}
		cfg.EnableAutoHeal = config.EnableAutoHeal
		if config.ResponseExtractors != nil {
			cfg.ResponseExtractors = config.ResponseExtractors
		}
	}

	if guard == nil {
		guard = NewProducerChainGuard(nil)
	}

	board := &CircuitBoard{
		config: cfg,
		pool:   paramPool,
		graph:  graph,
		guard:  guard,
		client: client,
		units:  make(map[string]*UnitWithStats),
	}

	// Initialize units from the graph
	if graph != nil {
		for _, unit := range graph.GetAllEndpoints() {
			board.units[unit.Name] = &UnitWithStats{
				EndpointUnit: unit,
			}
		}
	}

	return board
}

// SetRequestBuilder sets the request builder for the board.
func (b *CircuitBoard) SetRequestBuilder(builder RequestBuilder) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.builder = builder
}

// GetOrCreate retrieves a value for the semantic type from the pool.
// If no value exists and auto-healing is enabled, it triggers the appropriate
// producer endpoint to create the value.
func (b *CircuitBoard) GetOrCreate(ctx context.Context, semanticType SemanticType) (PoolValue, error) {
	if b.closed.Load() {
		return nil, ErrBoardClosed
	}

	// First, try to get from pool
	value, err := b.pool.Get(semanticType)
	if err == nil {
		return value, nil
	}

	// If pool is empty and auto-healing is disabled, return error
	if !b.config.EnableAutoHeal {
		return nil, fmt.Errorf("%w: %s", ErrMissingDependency, semanticType)
	}

	// Auto-heal: trigger a producer
	b.mu.Lock()
	b.stats.AutoHealAttempts++
	b.mu.Unlock()

	// Find producers for this semantic type
	producers := b.graph.GetProducers(semanticType)
	if len(producers) == 0 {
		b.mu.Lock()
		b.stats.AutoHealFailures++
		b.mu.Unlock()
		return nil, fmt.Errorf("%w: %s", ErrNoProducerAvailable, semanticType)
	}

	// Use the guard to prevent cascade overload
	var lastErr error
	err = b.guard.ExecuteWithGuard(ctx, func(ctx context.Context) error {
		// Try each producer until one succeeds
		for _, producer := range producers {
			// Execute the producer
			_, execErr := b.Execute(ctx, producer.Name)
			if execErr == nil {
				return nil
			}
			lastErr = execErr
		}
		return fmt.Errorf("all producers failed for %s: last error: %w", semanticType, lastErr)
	})

	if err != nil {
		b.mu.Lock()
		b.stats.AutoHealFailures++
		b.mu.Unlock()
		return nil, fmt.Errorf("%w: failed to trigger producer: %v", ErrMissingDependency, err)
	}

	// Now try to get the value again
	value, err = b.pool.Get(semanticType)
	if err != nil {
		b.mu.Lock()
		b.stats.AutoHealFailures++
		b.mu.Unlock()
		return nil, fmt.Errorf("%w: producer executed but value not found: %s", ErrMissingDependency, semanticType)
	}

	b.mu.Lock()
	b.stats.AutoHealSuccesses++
	b.mu.Unlock()

	return value, nil
}

// CheckDependencies verifies all input dependencies for an endpoint are satisfied.
// Returns a list of missing semantic types and whether all dependencies are met.
func (b *CircuitBoard) CheckDependencies(ctx context.Context, endpointName string) (missing []SemanticType, satisfied bool) {
	if b.closed.Load() {
		return nil, false
	}

	b.mu.RLock()
	unit, exists := b.units[endpointName]
	b.mu.RUnlock()

	if !exists {
		return nil, false
	}

	missing = make([]SemanticType, 0)
	for _, inputPin := range unit.InputPins {
		if inputPin == "" || inputPin == UnknownSemanticType {
			continue
		}
		_, err := b.pool.Get(inputPin)
		if err != nil {
			missing = append(missing, inputPin)
		}
	}

	return missing, len(missing) == 0
}

// SatisfyDependencies attempts to satisfy all missing dependencies for an endpoint.
// It triggers producers for each missing semantic type.
func (b *CircuitBoard) SatisfyDependencies(ctx context.Context, endpointName string) error {
	if b.closed.Load() {
		return ErrBoardClosed
	}

	missing, satisfied := b.CheckDependencies(ctx, endpointName)
	if satisfied {
		return nil
	}

	// Try to satisfy each missing dependency
	for _, semanticType := range missing {
		_, err := b.GetOrCreate(ctx, semanticType)
		if err != nil {
			return fmt.Errorf("cannot satisfy dependency %s: %w", semanticType, err)
		}
	}

	return nil
}

// Execute runs an endpoint and extracts output values to the pool.
// It first checks and satisfies dependencies if auto-healing is enabled.
func (b *CircuitBoard) Execute(ctx context.Context, endpointName string) (*ExecutionResult, error) {
	if b.closed.Load() {
		return nil, ErrBoardClosed
	}

	b.mu.RLock()
	unit, exists := b.units[endpointName]
	b.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrEndpointNotFoundOnBoard, endpointName)
	}

	// Start timing
	startTime := time.Now()

	// Update execution count
	unit.mu.Lock()
	unit.Stats.TotalExecutions++
	unit.mu.Unlock()

	b.mu.Lock()
	b.stats.TotalExecutions++
	b.mu.Unlock()

	// Check and satisfy dependencies if auto-healing is enabled
	if b.config.EnableAutoHeal {
		if err := b.SatisfyDependencies(ctx, endpointName); err != nil {
			b.recordFailure(unit, startTime, err)
			return nil, fmt.Errorf("dependency satisfaction failed: %w", err)
		}
	} else {
		// Just check if dependencies are satisfied
		missing, satisfied := b.CheckDependencies(ctx, endpointName)
		if !satisfied {
			err := fmt.Errorf("%w: missing types: %v", ErrMissingDependency, missing)
			b.recordFailure(unit, startTime, err)
			return nil, err
		}
	}

	// Build the request
	if b.builder == nil {
		err := errors.New("request builder not configured")
		b.recordFailure(unit, startTime, err)
		return nil, err
	}

	req, err := b.builder.BuildRequestForEndpoint(unit.EndpointUnit)
	if err != nil {
		b.recordFailure(unit, startTime, err)
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	// Add base URL if configured
	if b.config.BaseURL != "" && req.URL.Host == "" {
		req.URL.Scheme = "http"
		req.URL.Host = b.config.BaseURL
	}

	// Set timeout
	if b.config.DefaultTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, b.config.DefaultTimeout)
		defer cancel()
	}
	req = req.WithContext(ctx)

	// Execute the request
	resp, err := b.client.Do(req)
	if err != nil {
		b.recordFailure(unit, startTime, err)
		return nil, fmt.Errorf("%w: %v", ErrExecutionFailed, err)
	}
	defer resp.Body.Close()

	// Read response body with size limit to prevent memory exhaustion
	const maxResponseBodySize = 10 * 1024 * 1024 // 10MB
	limitedReader := io.LimitReader(resp.Body, maxResponseBodySize)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		b.recordFailure(unit, startTime, err)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
		b.recordFailure(unit, startTime, err)
		return nil, fmt.Errorf("%w: %v", ErrExecutionFailed, err)
	}

	// Extract output values
	extractedCount := b.extractOutputValues(unit.EndpointUnit, bodyBytes)

	// Record success
	latency := time.Since(startTime)
	b.recordSuccess(unit, latency, extractedCount)

	return &ExecutionResult{
		EndpointName:    endpointName,
		StatusCode:      resp.StatusCode,
		ResponseBody:    bodyBytes,
		Latency:         latency,
		ValuesExtracted: extractedCount,
	}, nil
}

// ExecuteWithPlan executes an endpoint with its full dependency chain.
// Returns all execution results in order.
func (b *CircuitBoard) ExecuteWithPlan(ctx context.Context, targetEndpoint string) ([]*ExecutionResult, error) {
	if b.closed.Load() {
		return nil, ErrBoardClosed
	}

	// Get execution plan from graph
	plan, err := b.graph.GetExecutionPlan(targetEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution plan: %w", err)
	}

	results := make([]*ExecutionResult, 0, len(plan.Steps))

	// Execute each step in order
	for _, step := range plan.Steps {
		// Skip disabled endpoints
		if step.Disabled {
			continue
		}

		result, err := b.Execute(ctx, step.Name)
		if err != nil {
			return results, fmt.Errorf("execution failed at step %s: %w", step.Name, err)
		}
		results = append(results, result)
	}

	return results, nil
}

// extractOutputValues extracts values from the response and stores them in the pool.
func (b *CircuitBoard) extractOutputValues(unit *EndpointUnit, responseBody []byte) int64 {
	if len(unit.OutputPins) == 0 {
		return 0
	}

	// Parse response as JSON
	var responseData map[string]any
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		return 0
	}

	var extractedCount int64

	// Check for configured extractors
	extractors := b.config.ResponseExtractors[unit.Name]

	if len(extractors) > 0 {
		// Use configured extractors
		for _, extractor := range extractors {
			value := extractJSONPath(responseData, extractor.JSONPath)
			if value != nil {
				source := extractor.Source
				if source.Endpoint == "" {
					source.Endpoint = unit.Name
				}
				b.pool.Add(extractor.SemanticType, value, source)
				extractedCount++
			}
		}
	} else {
		// Auto-extract based on output pins
		for _, outputPin := range unit.OutputPins {
			// Try common paths for the value
			paths := getCommonJSONPaths(outputPin)
			for _, path := range paths {
				value := extractJSONPath(responseData, path)
				if value != nil {
					source := ValueSource{
						Endpoint:      unit.Name,
						ResponseField: path,
					}
					b.pool.Add(outputPin, value, source)
					extractedCount++
					break
				}
			}
		}
	}

	b.mu.Lock()
	b.stats.ValueExtracted += extractedCount
	b.mu.Unlock()

	return extractedCount
}

// getCommonJSONPaths returns common JSON paths for a semantic type.
func getCommonJSONPaths(semanticType SemanticType) []string {
	field := semanticType.Field()
	entity := semanticType.Entity()

	paths := []string{
		// Direct field
		field,
		// Nested in data
		"data." + field,
	}

	if entity != "" {
		// Entity-specific paths
		paths = append(paths, entity+"."+field)
		paths = append(paths, "data."+entity+"."+field)
	}

	// Common response wrapper patterns
	paths = append(paths, "result."+field)
	paths = append(paths, "response."+field)

	return paths
}

// extractJSONPath extracts a value from JSON data using a dot-notation path.
func extractJSONPath(data map[string]any, path string) any {
	if path == "" {
		return nil
	}

	parts := splitPath(path)
	current := any(data)

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			next, exists := v[part]
			if !exists {
				return nil
			}
			current = next
		case []any:
			// For arrays, we might want the first element
			if len(v) > 0 {
				if nested, ok := v[0].(map[string]any); ok {
					next, exists := nested[part]
					if !exists {
						return nil
					}
					current = next
				} else {
					return nil
				}
			} else {
				return nil
			}
		default:
			return nil
		}
	}

	return current
}

// splitPath splits a dot-notation path into parts.
func splitPath(path string) []string {
	if path == "" {
		return nil
	}
	return strings.Split(path, ".")
}

// recordSuccess records a successful execution.
func (b *CircuitBoard) recordSuccess(unit *UnitWithStats, latency time.Duration, valuesProduced int64) {
	unit.mu.Lock()
	unit.Stats.SuccessfulExecutions++
	unit.Stats.TotalLatency += latency.Nanoseconds()
	unit.Stats.LastExecutedAt = time.Now()
	unit.Stats.ValuesProduced += valuesProduced
	if unit.Stats.MinLatency == 0 || latency < unit.Stats.MinLatency {
		unit.Stats.MinLatency = latency
	}
	if latency > unit.Stats.MaxLatency {
		unit.Stats.MaxLatency = latency
	}
	unit.mu.Unlock()

	b.mu.Lock()
	b.stats.SuccessfulExecutions++
	b.mu.Unlock()
}

// recordFailure records a failed execution.
func (b *CircuitBoard) recordFailure(unit *UnitWithStats, startTime time.Time, err error) {
	latency := time.Since(startTime)

	unit.mu.Lock()
	unit.Stats.FailedExecutions++
	unit.Stats.TotalLatency += latency.Nanoseconds()
	unit.Stats.LastExecutedAt = time.Now()
	unit.Stats.LastError = err.Error()
	unit.mu.Unlock()

	b.mu.Lock()
	b.stats.FailedExecutions++
	b.mu.Unlock()
}

// GetEndpointStats returns statistics for a specific endpoint.
func (b *CircuitBoard) GetEndpointStats(endpointName string) (*EndpointStats, error) {
	b.mu.RLock()
	unit, exists := b.units[endpointName]
	b.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrEndpointNotFoundOnBoard, endpointName)
	}

	unit.mu.RLock()
	stats := unit.Stats // Copy
	unit.mu.RUnlock()

	return &stats, nil
}

// GetAllEndpointStats returns statistics for all endpoints.
func (b *CircuitBoard) GetAllEndpointStats() map[string]EndpointStats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make(map[string]EndpointStats, len(b.units))
	for name, unit := range b.units {
		unit.mu.RLock()
		result[name] = unit.Stats
		unit.mu.RUnlock()
	}
	return result
}

// GetBoardStats returns overall board statistics.
func (b *CircuitBoard) GetBoardStats() BoardStats {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.stats
}

// AddEndpoint adds a new endpoint to the board.
func (b *CircuitBoard) AddEndpoint(unit *EndpointUnit) {
	if unit == nil || unit.Name == "" {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.units[unit.Name] = &UnitWithStats{
		EndpointUnit: unit,
	}

	// Also add to graph
	if b.graph != nil {
		b.graph.AddEndpoint(unit)
	}
}

// RemoveEndpoint removes an endpoint from the board.
func (b *CircuitBoard) RemoveEndpoint(endpointName string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.units, endpointName)
}

// GetEndpoint returns an endpoint by name.
func (b *CircuitBoard) GetEndpoint(name string) *EndpointUnit {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if unit, exists := b.units[name]; exists {
		return unit.EndpointUnit
	}
	return nil
}

// GetAllEndpoints returns all endpoints.
func (b *CircuitBoard) GetAllEndpoints() []*EndpointUnit {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]*EndpointUnit, 0, len(b.units))
	for _, unit := range b.units {
		result = append(result, unit.EndpointUnit)
	}
	return result
}

// GetPool returns the underlying parameter pool.
func (b *CircuitBoard) GetPool() ParameterPool {
	return b.pool
}

// GetGraph returns the underlying dependency graph.
func (b *CircuitBoard) GetGraph() *DependencyGraph {
	return b.graph
}

// GetGuard returns the underlying producer chain guard.
func (b *CircuitBoard) GetGuard() *ProducerChainGuard {
	return b.guard
}

// RebuildDependencies rebuilds the dependency graph.
func (b *CircuitBoard) RebuildDependencies() {
	if b.graph != nil {
		b.graph.BuildDependencies()
	}
}

// ResetStats resets all statistics.
func (b *CircuitBoard) ResetStats() {
	b.mu.Lock()
	b.stats = BoardStats{}
	for _, unit := range b.units {
		unit.mu.Lock()
		unit.Stats = EndpointStats{}
		unit.mu.Unlock()
	}
	b.mu.Unlock()
}

// Close closes the circuit board and releases resources.
func (b *CircuitBoard) Close() {
	if b.closed.Swap(true) {
		return // Already closed
	}

	if b.guard != nil {
		b.guard.Close()
	}
}

// IsClosed returns whether the board has been closed.
func (b *CircuitBoard) IsClosed() bool {
	return b.closed.Load()
}

// ExecutionResult holds the result of an endpoint execution.
type ExecutionResult struct {
	// EndpointName is the name of the executed endpoint.
	EndpointName string

	// StatusCode is the HTTP status code.
	StatusCode int

	// ResponseBody is the raw response body.
	ResponseBody []byte

	// Latency is the execution duration.
	Latency time.Duration

	// ValuesExtracted is the number of values extracted to the pool.
	ValuesExtracted int64

	// Error is set if the execution failed.
	Error error
}

// IsSuccess returns true if the execution was successful.
func (r *ExecutionResult) IsSuccess() bool {
	return r.Error == nil && r.StatusCode >= 200 && r.StatusCode < 300
}

// ParseJSON parses the response body as JSON.
func (r *ExecutionResult) ParseJSON(v any) error {
	return json.Unmarshal(r.ResponseBody, v)
}

// SimpleRequestBuilder is a simple implementation of RequestBuilder.
// It builds requests based on EndpointUnit configuration.
type SimpleRequestBuilder struct {
	pool    ParameterPool
	baseURL string
}

// NewSimpleRequestBuilder creates a new simple request builder.
func NewSimpleRequestBuilder(paramPool ParameterPool, baseURL string) *SimpleRequestBuilder {
	return &SimpleRequestBuilder{
		pool:    paramPool,
		baseURL: baseURL,
	}
}

// BuildRequestForEndpoint builds an HTTP request for the given endpoint.
func (rb *SimpleRequestBuilder) BuildRequestForEndpoint(unit *EndpointUnit) (*http.Request, error) {
	if unit == nil {
		return nil, errors.New("endpoint unit is nil")
	}

	// Build URL
	url := rb.baseURL + unit.Path

	// Create request
	req, err := http.NewRequest(unit.Method, url, nil)
	if err != nil {
		return nil, err
	}

	// Set content type for POST/PUT/PATCH
	if unit.Method == "POST" || unit.Method == "PUT" || unit.Method == "PATCH" {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}
