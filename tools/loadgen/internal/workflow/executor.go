// Package workflow implements business workflow support for the load generator.
package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// HTTPClient is the interface for making HTTP requests.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// ExecutorConfig holds configuration for the workflow executor.
type ExecutorConfig struct {
	// Client is the HTTP client for making requests.
	Client HTTPClient

	// BaseURL is the base URL for API requests.
	BaseURL string

	// Token is the authentication token (added to Authorization header).
	Token string

	// DefaultHeaders are headers added to all requests.
	DefaultHeaders map[string]string

	// OnStepStart is called before each step starts.
	OnStepStart func(workflowName string, stepIndex int, step Step)

	// OnStepComplete is called after each step completes.
	OnStepComplete func(workflowName string, stepIndex int, step Step, result StepResult)

	// OnWorkflowStart is called when a workflow starts.
	OnWorkflowStart func(workflowName string, def Definition)

	// OnWorkflowComplete is called when a workflow completes.
	OnWorkflowComplete func(workflowName string, def Definition, result Result)

	// DefaultTimeout is the default timeout for workflow execution.
	// Default: 60s
	DefaultTimeout time.Duration

	// MaxRetryDelay is the maximum delay between retries.
	// Default: 30s
	MaxRetryDelay time.Duration
}

// Executor executes workflows.
type Executor struct {
	config ExecutorConfig

	// Statistics
	workflowsExecuted  atomic.Int64
	workflowsSucceeded atomic.Int64
	workflowsFailed    atomic.Int64
	stepsExecuted      atomic.Int64
	stepsSucceeded     atomic.Int64
	stepsFailed        atomic.Int64

	// For testing
	nowFunc func() time.Time
}

// StepResult holds the result of executing a single step.
type StepResult struct {
	// StepIndex is the 0-based index of the step.
	StepIndex int

	// StepName is the name of the step (if provided).
	StepName string

	// Endpoint is the endpoint that was called.
	Endpoint string

	// Success indicates whether the step succeeded.
	Success bool

	// StatusCode is the HTTP status code received.
	StatusCode int

	// ExpectedStatus is the expected HTTP status code.
	ExpectedStatus int

	// Duration is the time taken to execute the step.
	Duration time.Duration

	// ExtractedValues holds values extracted from the response.
	ExtractedValues map[string]any

	// Error holds any error that occurred.
	Error error

	// ResponseBody is the response body (for debugging).
	ResponseBody string

	// Retries is the number of retries attempted.
	Retries int
}

// Result holds the result of executing a workflow.
type Result struct {
	// WorkflowName is the name of the executed workflow.
	WorkflowName string

	// Success indicates whether all steps completed successfully.
	Success bool

	// StepResults holds results for each step.
	StepResults []StepResult

	// Duration is the total time taken for the workflow.
	Duration time.Duration

	// Error holds any error that caused the workflow to abort.
	Error error

	// FinalContext holds the final state of extracted values.
	FinalContext map[string]any

	// CompletedSteps is the number of steps that completed (successfully or not).
	CompletedSteps int

	// SuccessfulSteps is the number of steps that succeeded.
	SuccessfulSteps int
}

// NewExecutor creates a new workflow executor.
func NewExecutor(config ExecutorConfig) (*Executor, error) {
	if config.Client == nil {
		return nil, fmt.Errorf("HTTP client is required")
	}

	if config.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	// Apply defaults
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = 60 * time.Second
	}
	if config.MaxRetryDelay == 0 {
		config.MaxRetryDelay = 30 * time.Second
	}

	return &Executor{
		config:  config,
		nowFunc: time.Now,
	}, nil
}

// Execute runs a workflow with the given initial context.
// The context map can contain initial values for placeholders.
func (e *Executor) Execute(ctx context.Context, def Definition, initialContext map[string]any) (*Result, error) {
	startTime := e.nowFunc()
	workflowName := def.Name

	// Create workflow context (copy initial context)
	workflowCtx := make(map[string]any, len(initialContext))
	maps.Copy(workflowCtx, initialContext)

	// Parse timeout
	timeout := e.config.DefaultTimeout
	if def.Timeout != "" {
		parsed, err := time.ParseDuration(def.Timeout)
		if err == nil {
			timeout = parsed
		}
	}

	// Create timeout context
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result := &Result{
		WorkflowName: workflowName,
		StepResults:  make([]StepResult, 0, len(def.Steps)),
		FinalContext: workflowCtx,
	}

	e.workflowsExecuted.Add(1)

	// Notify workflow start
	if e.config.OnWorkflowStart != nil {
		e.config.OnWorkflowStart(workflowName, def)
	}

	// Execute each step
	for i, step := range def.Steps {
		// Check context cancellation
		if execCtx.Err() != nil {
			result.Error = fmt.Errorf("%w: %v", ErrWorkflowAborted, execCtx.Err())
			break
		}

		// Execute step with delay if configured
		if step.Delay != "" {
			delay, err := time.ParseDuration(step.Delay)
			if err == nil && delay > 0 {
				select {
				case <-execCtx.Done():
					result.Error = fmt.Errorf("%w: %v", ErrWorkflowAborted, execCtx.Err())
					break
				case <-time.After(delay):
				}
			}
		}

		// Check condition if present
		if step.Condition != "" {
			conditionMet := evaluateCondition(step.Condition, workflowCtx)
			if !conditionMet {
				// Skip this step
				stepResult := StepResult{
					StepIndex: i,
					StepName:  step.Name,
					Endpoint:  step.Endpoint,
					Success:   true, // Consider skipped steps as successful
				}
				result.StepResults = append(result.StepResults, stepResult)
				result.CompletedSteps++
				result.SuccessfulSteps++
				continue
			}
		}

		// Notify step start
		if e.config.OnStepStart != nil {
			e.config.OnStepStart(workflowName, i, step)
		}

		// Execute the step
		stepResult := e.executeStep(execCtx, i, step, workflowCtx)
		result.StepResults = append(result.StepResults, stepResult)
		result.CompletedSteps++

		e.stepsExecuted.Add(1)

		// Notify step complete
		if e.config.OnStepComplete != nil {
			e.config.OnStepComplete(workflowName, i, step, stepResult)
		}

		if stepResult.Success {
			e.stepsSucceeded.Add(1)
			result.SuccessfulSteps++

			// Add extracted values to workflow context
			maps.Copy(workflowCtx, stepResult.ExtractedValues)
		} else {
			e.stepsFailed.Add(1)

			// Handle failure based on OnFailure setting
			onFailure := strings.ToLower(step.OnFailure)
			if onFailure == "" {
				onFailure = "abort"
			}

			switch onFailure {
			case "abort":
				if !def.ContinueOnError {
					result.Error = stepResult.Error
					break
				}
			case "continue":
				// Continue to next step
			case "retry":
				// Retry is handled in executeStep
			}

			// If we're not continuing on error and step failed, break
			if !def.ContinueOnError && onFailure == "abort" {
				break
			}
		}
	}

	result.Duration = e.nowFunc().Sub(startTime)
	result.Success = result.Error == nil && result.SuccessfulSteps == len(def.Steps)
	result.FinalContext = workflowCtx

	if result.Success {
		e.workflowsSucceeded.Add(1)
	} else {
		e.workflowsFailed.Add(1)
	}

	// Notify workflow complete
	if e.config.OnWorkflowComplete != nil {
		e.config.OnWorkflowComplete(workflowName, def, *result)
	}

	return result, nil
}

// executeStep executes a single step with retry support.
func (e *Executor) executeStep(ctx context.Context, index int, step Step, workflowCtx map[string]any) StepResult {
	startTime := e.nowFunc()

	result := StepResult{
		StepIndex:       index,
		StepName:        step.Name,
		Endpoint:        step.Endpoint,
		ExpectedStatus:  step.ExpectedStatus,
		ExtractedValues: make(map[string]any),
	}

	maxRetries := 0
	if strings.ToLower(step.OnFailure) == "retry" {
		maxRetries = step.RetryCount
	}

	retryDelay := time.Second
	if step.RetryDelay != "" {
		if parsed, err := time.ParseDuration(step.RetryDelay); err == nil {
			retryDelay = parsed
		}
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			result.Retries = attempt
			// Wait before retry
			select {
			case <-ctx.Done():
				result.Error = fmt.Errorf("%w: %v", ErrWorkflowAborted, ctx.Err())
				result.Duration = e.nowFunc().Sub(startTime)
				return result
			case <-time.After(retryDelay):
			}
		}

		// Execute the step
		statusCode, responseBody, err := e.doRequest(ctx, step, workflowCtx)
		result.StatusCode = statusCode
		result.ResponseBody = responseBody

		if err != nil {
			lastErr = err
			continue
		}

		// Check status code
		expectedStatus := step.ExpectedStatus
		if expectedStatus == 0 {
			step.ApplyDefaults()
			expectedStatus = step.ExpectedStatus
		}

		if statusCode != expectedStatus {
			lastErr = fmt.Errorf("%w: expected status %d, got %d", ErrStepFailed, expectedStatus, statusCode)
			continue
		}

		// Extract values from response
		if len(step.Extract) > 0 && responseBody != "" {
			extracted, err := extractValues(responseBody, step.Extract)
			if err != nil {
				lastErr = fmt.Errorf("%w: %v", ErrExtractionFailed, err)
				continue
			}
			result.ExtractedValues = extracted
		}

		// Success!
		result.Success = true
		result.Duration = e.nowFunc().Sub(startTime)
		return result
	}

	// All retries failed
	result.Error = lastErr
	result.Duration = e.nowFunc().Sub(startTime)
	return result
}

// doRequest performs the HTTP request for a step.
func (e *Executor) doRequest(ctx context.Context, step Step, workflowCtx map[string]any) (int, string, error) {
	method := step.GetMethod()
	path := step.GetPath()

	// Replace placeholders in path
	path = ReplacePlaceholders(path, workflowCtx)

	// Build URL
	url := strings.TrimSuffix(e.config.BaseURL, "/") + path

	// Prepare body
	var bodyReader io.Reader
	if step.Body != "" {
		body := ReplacePlaceholders(step.Body, workflowCtx)
		bodyReader = strings.NewReader(body)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return 0, "", fmt.Errorf("creating request: %w", err)
	}

	// Add default headers
	for k, v := range e.config.DefaultHeaders {
		req.Header.Set(k, v)
	}

	// Add step headers
	for k, v := range step.Headers {
		req.Header.Set(k, ReplacePlaceholders(v, workflowCtx))
	}

	// Add query parameters
	if len(step.QueryParams) > 0 {
		q := req.URL.Query()
		for k, v := range step.QueryParams {
			q.Set(k, ReplacePlaceholders(v, workflowCtx))
		}
		req.URL.RawQuery = q.Encode()
	}

	// Add authorization if present
	if e.config.Token != "" {
		req.Header.Set("Authorization", "Bearer "+e.config.Token)
	}

	// Set content type for POST/PUT/PATCH
	if step.Body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute request
	resp, err := e.config.Client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body with size limit to prevent memory exhaustion
	const maxResponseSize = 10 * 1024 * 1024 // 10 MB
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return resp.StatusCode, "", fmt.Errorf("reading response: %w", err)
	}

	return resp.StatusCode, string(bodyBytes), nil
}

// extractValues extracts values from a JSON response using JSONPath expressions.
func extractValues(responseBody string, extractions map[string]string) (map[string]any, error) {
	result := make(map[string]any)

	// Parse response as JSON
	var data any
	if err := json.Unmarshal([]byte(responseBody), &data); err != nil {
		return nil, fmt.Errorf("parsing response JSON: %w", err)
	}

	for varName, jsonPath := range extractions {
		value, err := evaluateJSONPath(data, jsonPath)
		if err != nil {
			return nil, fmt.Errorf("extracting %q with path %q: %w", varName, jsonPath, err)
		}
		result[varName] = value
	}

	return result, nil
}

// evaluateJSONPath evaluates a simple JSONPath expression.
// Supports basic paths like $.data.id, $.data.items[0].id, $.data.items[*].id
func evaluateJSONPath(data any, path string) (any, error) {
	if !strings.HasPrefix(path, "$") {
		return nil, fmt.Errorf("%w: path must start with '$'", ErrInvalidJSONPath)
	}

	// Remove leading $
	path = strings.TrimPrefix(path, "$")
	if path == "" {
		return data, nil
	}

	// Remove leading . if present
	path = strings.TrimPrefix(path, ".")

	return navigatePath(data, path)
}

// navigatePath navigates through the JSON structure following the path.
func navigatePath(data any, path string) (any, error) {
	if path == "" {
		return data, nil
	}

	// Split path into segments
	segments := parsePathSegments(path)
	if len(segments) == 0 {
		return data, nil
	}

	// Limit path depth to prevent abuse with deeply nested structures
	const maxPathDepth = 20
	if len(segments) > maxPathDepth {
		return nil, fmt.Errorf("JSONPath depth limit exceeded (max %d segments)", maxPathDepth)
	}

	current := data
	for _, segment := range segments {
		var err error
		current, err = navigateSegment(current, segment)
		if err != nil {
			return nil, err
		}
	}

	return current, nil
}

// pathSegment represents a single segment of a JSONPath.
type pathSegment struct {
	Key        string
	IsArray    bool
	Index      int
	IsWildcard bool
}

// parsePathSegments parses a path into segments.
func parsePathSegments(path string) []pathSegment {
	var segments []pathSegment

	i := 0
	for i < len(path) {
		if path[i] == '.' {
			i++
			continue
		}

		// Check for array access
		if path[i] == '[' {
			endBracket := strings.Index(path[i:], "]")
			if endBracket == -1 {
				break
			}

			content := path[i+1 : i+endBracket]
			if content == "*" {
				segments = append(segments, pathSegment{IsArray: true, IsWildcard: true})
			} else {
				var idx int
				fmt.Sscanf(content, "%d", &idx)
				segments = append(segments, pathSegment{IsArray: true, Index: idx})
			}
			i += endBracket + 1
			continue
		}

		// Find key end
		end := i
		for end < len(path) && path[end] != '.' && path[end] != '[' {
			end++
		}

		key := path[i:end]
		if key != "" {
			segments = append(segments, pathSegment{Key: key})
		}
		i = end
	}

	return segments
}

// navigateSegment navigates a single path segment.
func navigateSegment(data any, segment pathSegment) (any, error) {
	if segment.IsArray {
		arr, ok := data.([]any)
		if !ok {
			return nil, fmt.Errorf("expected array, got %T", data)
		}

		if segment.IsWildcard {
			// Return all elements
			return arr, nil
		}

		if segment.Index < 0 || segment.Index >= len(arr) {
			return nil, fmt.Errorf("array index %d out of bounds (length %d)", segment.Index, len(arr))
		}
		return arr[segment.Index], nil
	}

	// Navigate object key
	obj, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected object, got %T", data)
	}

	value, ok := obj[segment.Key]
	if !ok {
		return nil, fmt.Errorf("key %q not found", segment.Key)
	}

	return value, nil
}

// evaluateCondition evaluates a simple condition against the workflow context.
// Returns true if the condition is met.
// Supported conditions:
//   - {{.varName}} - true if varName exists and is not empty
//   - Simple string matching (future enhancement)
func evaluateCondition(condition string, ctx map[string]any) bool {
	// Simple placeholder check: {{.varName}}
	if strings.HasPrefix(condition, "{{.") && strings.HasSuffix(condition, "}}") {
		varName := condition[3 : len(condition)-2]
		val, ok := ctx[varName]
		if !ok {
			return false
		}
		// Check if value is non-empty
		switch v := val.(type) {
		case string:
			return v != ""
		case nil:
			return false
		default:
			return true
		}
	}

	// Default to true if condition format is not recognized
	return true
}

// Stats returns execution statistics.
func (e *Executor) Stats() ExecutorStats {
	return ExecutorStats{
		WorkflowsExecuted:  e.workflowsExecuted.Load(),
		WorkflowsSucceeded: e.workflowsSucceeded.Load(),
		WorkflowsFailed:    e.workflowsFailed.Load(),
		StepsExecuted:      e.stepsExecuted.Load(),
		StepsSucceeded:     e.stepsSucceeded.Load(),
		StepsFailed:        e.stepsFailed.Load(),
	}
}

// ExecutorStats holds execution statistics.
type ExecutorStats struct {
	WorkflowsExecuted  int64
	WorkflowsSucceeded int64
	WorkflowsFailed    int64
	StepsExecuted      int64
	StepsSucceeded     int64
	StepsFailed        int64
}

// SuccessRate returns the workflow success rate.
func (s ExecutorStats) SuccessRate() float64 {
	if s.WorkflowsExecuted == 0 {
		return 0
	}
	return float64(s.WorkflowsSucceeded) / float64(s.WorkflowsExecuted)
}

// StepSuccessRate returns the step success rate.
func (s ExecutorStats) StepSuccessRate() float64 {
	if s.StepsExecuted == 0 {
		return 0
	}
	return float64(s.StepsSucceeded) / float64(s.StepsExecuted)
}

// WithNowFunc sets a custom time function for testing.
// IMPORTANT: This method is NOT thread-safe. It must be called during initialization
// before any concurrent operations begin.
func (e *Executor) WithNowFunc(fn func() time.Time) *Executor {
	e.nowFunc = fn
	return e
}

// ExecutionContext holds the context for a workflow execution.
type ExecutionContext struct {
	mu     sync.RWMutex
	values map[string]any
}

// NewExecutionContext creates a new execution context with initial values.
func NewExecutionContext(initial map[string]any) *ExecutionContext {
	ctx := &ExecutionContext{
		values: make(map[string]any, len(initial)),
	}
	maps.Copy(ctx.values, initial)
	return ctx
}

// Set sets a value in the context.
func (c *ExecutionContext) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = value
}

// Get retrieves a value from the context.
func (c *ExecutionContext) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.values[key]
	return val, ok
}

// GetAll returns a copy of all values in the context.
func (c *ExecutionContext) GetAll() map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make(map[string]any, len(c.values))
	maps.Copy(result, c.values)
	return result
}
