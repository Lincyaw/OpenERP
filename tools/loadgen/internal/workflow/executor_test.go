package workflow

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHTTPClient is a mock HTTP client for testing.
type mockHTTPClient struct {
	mu        sync.Mutex
	responses []mockResponse
	calls     []mockCall
	callIndex int
}

type mockResponse struct {
	statusCode int
	body       string
	err        error
}

type mockCall struct {
	method  string
	url     string
	body    string
	headers http.Header
}

func newMockHTTPClient(responses ...mockResponse) *mockHTTPClient {
	return &mockHTTPClient{
		responses: responses,
		calls:     make([]mockCall, 0),
	}
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Record the call
	body := ""
	if req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		body = string(bodyBytes)
	}
	m.calls = append(m.calls, mockCall{
		method:  req.Method,
		url:     req.URL.String(),
		body:    body,
		headers: req.Header.Clone(),
	})

	// Return mock response
	if m.callIndex >= len(m.responses) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"success": true}`)),
		}, nil
	}

	resp := m.responses[m.callIndex]
	m.callIndex++

	if resp.err != nil {
		return nil, resp.err
	}

	return &http.Response{
		StatusCode: resp.statusCode,
		Body:       io.NopCloser(strings.NewReader(resp.body)),
	}, nil
}

func (m *mockHTTPClient) getCalls() []mockCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]mockCall{}, m.calls...)
}

func TestNewExecutor(t *testing.T) {
	tests := []struct {
		name    string
		config  ExecutorConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: ExecutorConfig{
				Client:  &http.Client{},
				BaseURL: "http://localhost:8080",
			},
			wantErr: false,
		},
		{
			name: "missing client",
			config: ExecutorConfig{
				BaseURL: "http://localhost:8080",
			},
			wantErr: true,
			errMsg:  "HTTP client is required",
		},
		{
			name: "missing base URL",
			config: ExecutorConfig{
				Client: &http.Client{},
			},
			wantErr: true,
			errMsg:  "base URL is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec, err := NewExecutor(tt.config)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, exec)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, exec)
			}
		})
	}
}

func TestExecutor_Execute_SimpleWorkflow(t *testing.T) {
	mockClient := newMockHTTPClient(
		mockResponse{statusCode: 200, body: `{"data": {"id": "order-123"}}`},
	)

	exec, err := NewExecutor(ExecutorConfig{
		Client:  mockClient,
		BaseURL: "http://localhost:8080",
	})
	require.NoError(t, err)

	// Simple workflow with one GET step
	def := Definition{
		Name: "simple_workflow",
		Steps: []Step{
			{
				Name:           "get_order",
				Endpoint:       "GET /api/orders/123",
				ExpectedStatus: 200,
			},
		},
	}

	result, err := exec.Execute(context.Background(), def, nil)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, 1, result.CompletedSteps)
	assert.Equal(t, 1, result.SuccessfulSteps)
	assert.Len(t, result.StepResults, 1)
	assert.True(t, result.StepResults[0].Success)

	// Verify HTTP call
	calls := mockClient.getCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, "GET", calls[0].method)
	assert.Contains(t, calls[0].url, "/api/orders/123")
}

func TestExecutor_Execute_MultiStepWorkflow(t *testing.T) {
	mockClient := newMockHTTPClient(
		mockResponse{statusCode: 201, body: `{"data": {"id": "order-001", "status": "draft"}}`},
		mockResponse{statusCode: 200, body: `{"data": {"id": "order-001", "status": "confirmed"}}`},
		mockResponse{statusCode: 200, body: `{"data": {"id": "order-001", "status": "shipped"}}`},
	)

	exec, err := NewExecutor(ExecutorConfig{
		Client:  mockClient,
		BaseURL: "http://localhost:8080",
	})
	require.NoError(t, err)

	// Multi-step workflow: Create → Confirm → Ship
	def := Definition{
		Name: "sales_cycle",
		Steps: []Step{
			{
				Name:           "create_order",
				Endpoint:       "POST /api/orders",
				Body:           `{"customer_id": "cust-123"}`,
				ExpectedStatus: 201,
				Extract:        map[string]string{"order_id": "$.data.id"},
			},
			{
				Name:           "confirm_order",
				Endpoint:       "POST /api/orders/{order_id}/confirm",
				ExpectedStatus: 200,
			},
			{
				Name:           "ship_order",
				Endpoint:       "POST /api/orders/{order_id}/ship",
				ExpectedStatus: 200,
			},
		},
	}

	result, err := exec.Execute(context.Background(), def, nil)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, 3, result.CompletedSteps)
	assert.Equal(t, 3, result.SuccessfulSteps)

	// Verify all HTTP calls
	calls := mockClient.getCalls()
	require.Len(t, calls, 3)

	// Step 1: Create order
	assert.Equal(t, "POST", calls[0].method)
	assert.Contains(t, calls[0].url, "/api/orders")
	assert.Contains(t, calls[0].body, "cust-123")

	// Step 2: Confirm order (should use extracted order_id)
	assert.Equal(t, "POST", calls[1].method)
	assert.Contains(t, calls[1].url, "/api/orders/order-001/confirm")

	// Step 3: Ship order (should use extracted order_id)
	assert.Equal(t, "POST", calls[2].method)
	assert.Contains(t, calls[2].url, "/api/orders/order-001/ship")

	// Verify final context has extracted values
	assert.Equal(t, "order-001", result.FinalContext["order_id"])
}

func TestExecutor_Execute_WithInitialContext(t *testing.T) {
	mockClient := newMockHTTPClient(
		mockResponse{statusCode: 200, body: `{"data": {"id": "order-123"}}`},
	)

	exec, err := NewExecutor(ExecutorConfig{
		Client:  mockClient,
		BaseURL: "http://localhost:8080",
	})
	require.NoError(t, err)

	def := Definition{
		Name: "test_workflow",
		Steps: []Step{
			{
				Name:           "get_order",
				Endpoint:       "GET /api/orders/{order_id}",
				ExpectedStatus: 200,
			},
		},
	}

	// Pass initial context with order_id
	initialContext := map[string]any{
		"order_id": "initial-order-456",
	}

	result, err := exec.Execute(context.Background(), def, initialContext)
	require.NoError(t, err)
	assert.True(t, result.Success)

	// Verify the order_id from initial context was used
	calls := mockClient.getCalls()
	require.Len(t, calls, 1)
	assert.Contains(t, calls[0].url, "/api/orders/initial-order-456")
}

func TestExecutor_Execute_StepFailure_Abort(t *testing.T) {
	mockClient := newMockHTTPClient(
		mockResponse{statusCode: 201, body: `{"data": {"id": "order-001"}}`},
		mockResponse{statusCode: 500, body: `{"error": "internal error"}`}, // Fails
		mockResponse{statusCode: 200, body: `{}`},                          // Should not be called
	)

	exec, err := NewExecutor(ExecutorConfig{
		Client:  mockClient,
		BaseURL: "http://localhost:8080",
	})
	require.NoError(t, err)

	def := Definition{
		Name: "failing_workflow",
		Steps: []Step{
			{
				Name:           "step1",
				Endpoint:       "POST /api/step1",
				ExpectedStatus: 201,
				Extract:        map[string]string{"id": "$.data.id"},
			},
			{
				Name:           "step2_fails",
				Endpoint:       "POST /api/step2",
				ExpectedStatus: 200, // Will fail because we get 500
				OnFailure:      "abort",
			},
			{
				Name:           "step3_not_called",
				Endpoint:       "POST /api/step3",
				ExpectedStatus: 200,
			},
		},
	}

	result, err := exec.Execute(context.Background(), def, nil)
	require.NoError(t, err) // Execute returns no error, but Result.Success is false
	assert.False(t, result.Success)
	assert.Equal(t, 2, result.CompletedSteps)
	assert.Equal(t, 1, result.SuccessfulSteps)
	assert.NotNil(t, result.Error)

	// Step 3 should not be executed
	calls := mockClient.getCalls()
	assert.Len(t, calls, 2)
}

func TestExecutor_Execute_StepFailure_Continue(t *testing.T) {
	mockClient := newMockHTTPClient(
		mockResponse{statusCode: 200, body: `{}`},
		mockResponse{statusCode: 500, body: `{}`}, // Fails but continue
		mockResponse{statusCode: 200, body: `{}`}, // Should be called
	)

	exec, err := NewExecutor(ExecutorConfig{
		Client:  mockClient,
		BaseURL: "http://localhost:8080",
	})
	require.NoError(t, err)

	def := Definition{
		Name:            "continue_workflow",
		ContinueOnError: true, // Continue even on step failure
		Steps: []Step{
			{Endpoint: "GET /api/step1", ExpectedStatus: 200},
			{Endpoint: "GET /api/step2", ExpectedStatus: 200, OnFailure: "continue"},
			{Endpoint: "GET /api/step3", ExpectedStatus: 200},
		},
	}

	result, err := exec.Execute(context.Background(), def, nil)
	require.NoError(t, err)
	assert.False(t, result.Success) // Not all steps succeeded
	assert.Equal(t, 3, result.CompletedSteps)
	assert.Equal(t, 2, result.SuccessfulSteps)

	// All 3 steps should be executed
	calls := mockClient.getCalls()
	assert.Len(t, calls, 3)
}

func TestExecutor_Execute_WithRetry(t *testing.T) {
	mockClient := newMockHTTPClient(
		mockResponse{statusCode: 500, body: `{}`}, // Fail 1st
		mockResponse{statusCode: 500, body: `{}`}, // Fail 2nd
		mockResponse{statusCode: 200, body: `{}`}, // Success 3rd
	)

	exec, err := NewExecutor(ExecutorConfig{
		Client:  mockClient,
		BaseURL: "http://localhost:8080",
	})
	require.NoError(t, err)

	def := Definition{
		Name: "retry_workflow",
		Steps: []Step{
			{
				Endpoint:       "GET /api/test",
				ExpectedStatus: 200,
				OnFailure:      "retry",
				RetryCount:     3,
				RetryDelay:     "10ms",
			},
		},
	}

	result, err := exec.Execute(context.Background(), def, nil)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, 2, result.StepResults[0].Retries)

	// Should have 3 HTTP calls total
	calls := mockClient.getCalls()
	assert.Len(t, calls, 3)
}

func TestExecutor_Execute_ContextCancellation(t *testing.T) {
	mockClient := newMockHTTPClient(
		mockResponse{statusCode: 200, body: `{}`},
	)

	exec, err := NewExecutor(ExecutorConfig{
		Client:  mockClient,
		BaseURL: "http://localhost:8080",
	})
	require.NoError(t, err)

	def := Definition{
		Name: "cancel_workflow",
		Steps: []Step{
			{Endpoint: "GET /api/step1", ExpectedStatus: 200, Delay: "1s"},
			{Endpoint: "GET /api/step2", ExpectedStatus: 200},
		},
	}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := exec.Execute(ctx, def, nil)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error.Error(), "aborted")
}

func TestExecutor_Execute_WithTimeout(t *testing.T) {
	mockClient := newMockHTTPClient(
		mockResponse{statusCode: 200, body: `{}`},
	)

	exec, err := NewExecutor(ExecutorConfig{
		Client:         mockClient,
		BaseURL:        "http://localhost:8080",
		DefaultTimeout: 10 * time.Millisecond,
	})
	require.NoError(t, err)

	def := Definition{
		Name:    "timeout_workflow",
		Timeout: "10ms",
		Steps: []Step{
			{Endpoint: "GET /api/step1", ExpectedStatus: 200, Delay: "100ms"}, // Delay longer than timeout
		},
	}

	result, err := exec.Execute(context.Background(), def, nil)
	require.NoError(t, err)
	assert.False(t, result.Success)
}

func TestExecutor_Execute_WithCondition(t *testing.T) {
	mockClient := newMockHTTPClient(
		mockResponse{statusCode: 200, body: `{}`},
	)

	exec, err := NewExecutor(ExecutorConfig{
		Client:  mockClient,
		BaseURL: "http://localhost:8080",
	})
	require.NoError(t, err)

	def := Definition{
		Name: "conditional_workflow",
		Steps: []Step{
			{
				Name:           "conditional_step",
				Endpoint:       "GET /api/test",
				ExpectedStatus: 200,
				Condition:      "{{.order_id}}", // Only run if order_id exists
			},
		},
	}

	// Test with condition NOT met (no order_id in context)
	result, err := exec.Execute(context.Background(), def, nil)
	require.NoError(t, err)
	assert.True(t, result.Success) // Skipped steps count as successful
	assert.Equal(t, 1, result.SuccessfulSteps)

	// No HTTP calls because condition was not met
	calls := mockClient.getCalls()
	assert.Len(t, calls, 0)

	// Reset mock and test with condition MET
	mockClient2 := newMockHTTPClient(
		mockResponse{statusCode: 200, body: `{}`},
	)
	exec2, _ := NewExecutor(ExecutorConfig{
		Client:  mockClient2,
		BaseURL: "http://localhost:8080",
	})

	result2, err := exec2.Execute(context.Background(), def, map[string]any{"order_id": "123"})
	require.NoError(t, err)
	assert.True(t, result2.Success)

	calls2 := mockClient2.getCalls()
	assert.Len(t, calls2, 1) // HTTP call was made because condition was met
}

func TestExecutor_Execute_ExtractValues(t *testing.T) {
	responseBody := `{
		"success": true,
		"data": {
			"id": "ord-123",
			"customer": {
				"id": "cust-456",
				"name": "Test Customer"
			},
			"items": [
				{"id": "item-1", "product_id": "prod-1"},
				{"id": "item-2", "product_id": "prod-2"}
			]
		}
	}`

	mockClient := newMockHTTPClient(
		mockResponse{statusCode: 200, body: responseBody},
	)

	exec, err := NewExecutor(ExecutorConfig{
		Client:  mockClient,
		BaseURL: "http://localhost:8080",
	})
	require.NoError(t, err)

	def := Definition{
		Name: "extract_workflow",
		Steps: []Step{
			{
				Endpoint:       "GET /api/order",
				ExpectedStatus: 200,
				Extract: map[string]string{
					"order_id":      "$.data.id",
					"customer_id":   "$.data.customer.id",
					"customer_name": "$.data.customer.name",
					"first_item_id": "$.data.items[0].id",
				},
			},
		},
	}

	result, err := exec.Execute(context.Background(), def, nil)
	require.NoError(t, err)
	assert.True(t, result.Success)

	// Verify extracted values
	assert.Equal(t, "ord-123", result.FinalContext["order_id"])
	assert.Equal(t, "cust-456", result.FinalContext["customer_id"])
	assert.Equal(t, "Test Customer", result.FinalContext["customer_name"])
	assert.Equal(t, "item-1", result.FinalContext["first_item_id"])
}

func TestExecutor_Execute_WithHeaders(t *testing.T) {
	mockClient := newMockHTTPClient(
		mockResponse{statusCode: 200, body: `{}`},
	)

	exec, err := NewExecutor(ExecutorConfig{
		Client:  mockClient,
		BaseURL: "http://localhost:8080",
		Token:   "test-token",
		DefaultHeaders: map[string]string{
			"X-Request-ID": "global-123",
		},
	})
	require.NoError(t, err)

	def := Definition{
		Name: "headers_workflow",
		Steps: []Step{
			{
				Endpoint:       "GET /api/test",
				ExpectedStatus: 200,
				Headers: map[string]string{
					"X-Custom-Header": "custom-value",
				},
			},
		},
	}

	result, err := exec.Execute(context.Background(), def, nil)
	require.NoError(t, err)
	assert.True(t, result.Success)

	calls := mockClient.getCalls()
	require.Len(t, calls, 1)

	// Check headers
	assert.Equal(t, "Bearer test-token", calls[0].headers.Get("Authorization"))
	assert.Equal(t, "global-123", calls[0].headers.Get("X-Request-ID"))
	assert.Equal(t, "custom-value", calls[0].headers.Get("X-Custom-Header"))
}

func TestExecutor_Execute_WithQueryParams(t *testing.T) {
	mockClient := newMockHTTPClient(
		mockResponse{statusCode: 200, body: `{}`},
	)

	exec, err := NewExecutor(ExecutorConfig{
		Client:  mockClient,
		BaseURL: "http://localhost:8080",
	})
	require.NoError(t, err)

	def := Definition{
		Name: "query_workflow",
		Steps: []Step{
			{
				Endpoint:       "GET /api/orders",
				ExpectedStatus: 200,
				QueryParams: map[string]string{
					"status":      "pending",
					"customer_id": "{customer_id}",
				},
			},
		},
	}

	result, err := exec.Execute(context.Background(), def, map[string]any{"customer_id": "cust-789"})
	require.NoError(t, err)
	assert.True(t, result.Success)

	calls := mockClient.getCalls()
	require.Len(t, calls, 1)
	assert.Contains(t, calls[0].url, "status=pending")
	assert.Contains(t, calls[0].url, "customer_id=cust-789")
}

func TestExecutor_Stats(t *testing.T) {
	mockClient := newMockHTTPClient(
		mockResponse{statusCode: 200, body: `{}`},
		mockResponse{statusCode: 200, body: `{}`},
		mockResponse{statusCode: 500, body: `{}`}, // This will fail
	)

	exec, err := NewExecutor(ExecutorConfig{
		Client:  mockClient,
		BaseURL: "http://localhost:8080",
	})
	require.NoError(t, err)

	// Execute first workflow (success)
	def1 := Definition{
		Name:  "workflow1",
		Steps: []Step{{Endpoint: "GET /api/test", ExpectedStatus: 200}},
	}
	_, _ = exec.Execute(context.Background(), def1, nil)

	// Execute second workflow (success)
	def2 := Definition{
		Name:  "workflow2",
		Steps: []Step{{Endpoint: "GET /api/test", ExpectedStatus: 200}},
	}
	_, _ = exec.Execute(context.Background(), def2, nil)

	// Execute third workflow (fail)
	def3 := Definition{
		Name:  "workflow3",
		Steps: []Step{{Endpoint: "GET /api/test", ExpectedStatus: 200}}, // Will get 500
	}
	_, _ = exec.Execute(context.Background(), def3, nil)

	stats := exec.Stats()
	assert.Equal(t, int64(3), stats.WorkflowsExecuted)
	assert.Equal(t, int64(2), stats.WorkflowsSucceeded)
	assert.Equal(t, int64(1), stats.WorkflowsFailed)
	assert.Equal(t, int64(3), stats.StepsExecuted)
	assert.Equal(t, int64(2), stats.StepsSucceeded)
	assert.Equal(t, int64(1), stats.StepsFailed)

	// Test success rates
	assert.InDelta(t, 0.6666, stats.SuccessRate(), 0.01)
	assert.InDelta(t, 0.6666, stats.StepSuccessRate(), 0.01)
}

func TestExecutor_Callbacks(t *testing.T) {
	mockClient := newMockHTTPClient(
		mockResponse{statusCode: 200, body: `{"data": {"id": "123"}}`},
		mockResponse{statusCode: 200, body: `{}`},
	)

	var workflowStartCalled bool
	var workflowCompleteCalled bool
	var stepStartCalls []int
	var stepCompleteCalls []int

	exec, err := NewExecutor(ExecutorConfig{
		Client:  mockClient,
		BaseURL: "http://localhost:8080",
		OnWorkflowStart: func(name string, def Definition) {
			workflowStartCalled = true
			assert.Equal(t, "callback_test", name)
		},
		OnWorkflowComplete: func(name string, def Definition, result Result) {
			workflowCompleteCalled = true
			assert.True(t, result.Success)
		},
		OnStepStart: func(workflowName string, stepIndex int, step Step) {
			stepStartCalls = append(stepStartCalls, stepIndex)
		},
		OnStepComplete: func(workflowName string, stepIndex int, step Step, result StepResult) {
			stepCompleteCalls = append(stepCompleteCalls, stepIndex)
		},
	})
	require.NoError(t, err)

	def := Definition{
		Name: "callback_test",
		Steps: []Step{
			{Endpoint: "GET /api/step1", ExpectedStatus: 200},
			{Endpoint: "GET /api/step2", ExpectedStatus: 200},
		},
	}

	_, _ = exec.Execute(context.Background(), def, nil)

	assert.True(t, workflowStartCalled)
	assert.True(t, workflowCompleteCalled)
	assert.Equal(t, []int{0, 1}, stepStartCalls)
	assert.Equal(t, []int{0, 1}, stepCompleteCalls)
}

func TestEvaluateJSONPath(t *testing.T) {
	jsonData := `{
		"success": true,
		"data": {
			"id": "123",
			"name": "Test",
			"nested": {
				"value": 42
			},
			"items": [
				{"id": "item1"},
				{"id": "item2"},
				{"id": "item3"}
			]
		}
	}`

	var data any
	err := json.Unmarshal([]byte(jsonData), &data)
	require.NoError(t, err)

	tests := []struct {
		name     string
		path     string
		expected any
		wantErr  bool
	}{
		{
			name:    "root object",
			path:    "$",
			wantErr: false,
		},
		{
			name:     "simple field",
			path:     "$.success",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "nested field",
			path:     "$.data.id",
			expected: "123",
			wantErr:  false,
		},
		{
			name:     "deeply nested field",
			path:     "$.data.nested.value",
			expected: float64(42),
			wantErr:  false,
		},
		{
			name:     "array index",
			path:     "$.data.items[0].id",
			expected: "item1",
			wantErr:  false,
		},
		{
			name:     "array index 2",
			path:     "$.data.items[2].id",
			expected: "item3",
			wantErr:  false,
		},
		{
			name:    "invalid path - missing $",
			path:    "data.id",
			wantErr: true,
		},
		{
			name:    "nonexistent field",
			path:    "$.nonexistent",
			wantErr: true,
		},
		{
			name:    "array out of bounds",
			path:    "$.data.items[99].id",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluateJSONPath(data, tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.expected != nil {
					assert.Equal(t, tt.expected, result)
				}
			}
		})
	}
}

func TestExecutionContext(t *testing.T) {
	initial := map[string]any{
		"key1": "value1",
		"key2": 42,
	}

	ctx := NewExecutionContext(initial)

	// Test Get
	val, ok := ctx.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)

	// Test Get non-existent
	_, ok = ctx.Get("nonexistent")
	assert.False(t, ok)

	// Test Set
	ctx.Set("key3", "value3")
	val, ok = ctx.Get("key3")
	assert.True(t, ok)
	assert.Equal(t, "value3", val)

	// Test GetAll (should be a copy)
	all := ctx.GetAll()
	assert.Len(t, all, 3)
	assert.Equal(t, "value1", all["key1"])
	assert.Equal(t, 42, all["key2"])
	assert.Equal(t, "value3", all["key3"])

	// Modify the copy and verify original is unchanged
	all["key1"] = "modified"
	val, _ = ctx.Get("key1")
	assert.Equal(t, "value1", val)
}

// TestExecutor_Execute_SalesCycle tests the complete sales order workflow
// This is the acceptance test for the LOADGEN-023 task
func TestExecutor_Execute_SalesCycle(t *testing.T) {
	// Mock responses for the sales cycle workflow
	mockClient := newMockHTTPClient(
		// Step 1: Create sales order
		mockResponse{
			statusCode: 201,
			body: `{
				"success": true,
				"data": {
					"id": "so-20260129-001",
					"order_number": "SO-2026-0001",
					"status": "draft",
					"customer_id": "cust-123"
				}
			}`,
		},
		// Step 2: Add item to order
		mockResponse{
			statusCode: 201,
			body: `{
				"success": true,
				"data": {
					"id": "item-001",
					"product_id": "prod-abc",
					"quantity": 10,
					"unit_price": 99.99
				}
			}`,
		},
		// Step 3: Confirm order
		mockResponse{
			statusCode: 200,
			body: `{
				"success": true,
				"data": {
					"id": "so-20260129-001",
					"status": "confirmed"
				}
			}`,
		},
		// Step 4: Ship order
		mockResponse{
			statusCode: 200,
			body: `{
				"success": true,
				"data": {
					"id": "so-20260129-001",
					"status": "shipped"
				}
			}`,
		},
	)

	exec, err := NewExecutor(ExecutorConfig{
		Client:  mockClient,
		BaseURL: "http://localhost:8080/api/v1",
		Token:   "test-token",
	})
	require.NoError(t, err)

	// Define the sales_cycle workflow
	salesCycleWorkflow := Definition{
		Name:        "sales_cycle",
		Description: "Complete sales order lifecycle: create -> add item -> confirm -> ship",
		Weight:      10,
		Timeout:     "60s",
		Steps: []Step{
			{
				Name:           "create_order",
				Endpoint:       "POST /trade/sales-orders",
				Body:           `{"customer_id": "{customer_id}", "warehouse_id": "{warehouse_id}", "remark": "Test order"}`,
				ExpectedStatus: 201,
				Extract: map[string]string{
					"order_id":     "$.data.id",
					"order_number": "$.data.order_number",
				},
			},
			{
				Name:           "add_item",
				Endpoint:       "POST /trade/sales-orders/{order_id}/items",
				Body:           `{"product_id": "{product_id}", "quantity": 10, "unit_price": 99.99}`,
				ExpectedStatus: 201,
				Extract: map[string]string{
					"item_id": "$.data.id",
				},
			},
			{
				Name:           "confirm_order",
				Endpoint:       "POST /trade/sales-orders/{order_id}/confirm",
				ExpectedStatus: 200,
			},
			{
				Name:           "ship_order",
				Endpoint:       "POST /trade/sales-orders/{order_id}/ship",
				ExpectedStatus: 200,
			},
		},
	}

	// Execute with initial context (simulating values from parameter pool)
	initialContext := map[string]any{
		"customer_id":  "cust-123",
		"warehouse_id": "wh-001",
		"product_id":   "prod-abc",
	}

	result, err := exec.Execute(context.Background(), salesCycleWorkflow, initialContext)
	require.NoError(t, err)

	// ACCEPTANCE CRITERIA: 100% success rate
	assert.True(t, result.Success, "Sales cycle workflow should complete successfully")
	assert.Equal(t, 4, result.CompletedSteps, "All 4 steps should be completed")
	assert.Equal(t, 4, result.SuccessfulSteps, "All 4 steps should succeed")
	assert.Nil(t, result.Error, "No errors should occur")

	// Verify workflow name
	assert.Equal(t, "sales_cycle", result.WorkflowName)

	// Verify each step succeeded
	for i, stepResult := range result.StepResults {
		assert.True(t, stepResult.Success, "Step %d (%s) should succeed", i, stepResult.StepName)
	}

	// Verify extracted values are in final context
	assert.Equal(t, "so-20260129-001", result.FinalContext["order_id"])
	assert.Equal(t, "SO-2026-0001", result.FinalContext["order_number"])
	assert.Equal(t, "item-001", result.FinalContext["item_id"])

	// Verify HTTP calls were made correctly
	calls := mockClient.getCalls()
	require.Len(t, calls, 4)

	// Step 1: Create order
	assert.Equal(t, "POST", calls[0].method)
	assert.Contains(t, calls[0].url, "/trade/sales-orders")
	assert.Contains(t, calls[0].body, "cust-123")
	assert.Contains(t, calls[0].body, "wh-001")

	// Step 2: Add item (uses extracted order_id)
	assert.Equal(t, "POST", calls[1].method)
	assert.Contains(t, calls[1].url, "/trade/sales-orders/so-20260129-001/items")
	assert.Contains(t, calls[1].body, "prod-abc")

	// Step 3: Confirm (uses extracted order_id)
	assert.Equal(t, "POST", calls[2].method)
	assert.Contains(t, calls[2].url, "/trade/sales-orders/so-20260129-001/confirm")

	// Step 4: Ship (uses extracted order_id)
	assert.Equal(t, "POST", calls[3].method)
	assert.Contains(t, calls[3].url, "/trade/sales-orders/so-20260129-001/ship")

	// Log success rate for verification
	t.Logf("Sales cycle workflow success rate: %.0f%%", float64(result.SuccessfulSteps)/float64(result.CompletedSteps)*100)
}
