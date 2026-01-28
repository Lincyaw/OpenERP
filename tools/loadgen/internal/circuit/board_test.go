// Package circuit provides circuit-board-like components for the load generator.
// This file contains integration tests for the CircuitBoard.
package circuit

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Mock Implementations for Testing
// =============================================================================

// mockPoolValue implements PoolValue interface for testing.
type mockPoolValue struct {
	data any
}

func (v *mockPoolValue) GetData() any {
	return v.data
}

// mockParameterPool implements ParameterPool interface for testing.
type mockParameterPool struct {
	mu     sync.RWMutex
	values map[SemanticType][]any
}

func newMockParameterPool() *mockParameterPool {
	return &mockParameterPool{
		values: make(map[SemanticType][]any),
	}
}

func (p *mockParameterPool) Add(semantic SemanticType, value any, source ValueSource) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.values[semantic] = append(p.values[semantic], value)
}

func (p *mockParameterPool) Get(semantic SemanticType) (PoolValue, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	vals, exists := p.values[semantic]
	if !exists || len(vals) == 0 {
		return nil, fmt.Errorf("no value for %s", semantic)
	}
	return &mockPoolValue{data: vals[len(vals)-1]}, nil
}

func (p *mockParameterPool) Size(semantic SemanticType) int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.values[semantic])
}

// mockHTTPClient implements HTTPClient interface for testing.
type mockHTTPClient struct {
	mu        sync.Mutex
	responses map[string]*http.Response
	requests  []*http.Request
	handler   func(req *http.Request) (*http.Response, error)
}

func newMockHTTPClient() *mockHTTPClient {
	return &mockHTTPClient{
		responses: make(map[string]*http.Response),
	}
}

func (c *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	c.mu.Lock()
	c.requests = append(c.requests, req)
	c.mu.Unlock()

	if c.handler != nil {
		return c.handler(req)
	}

	key := req.Method + ":" + req.URL.Path
	if resp, exists := c.responses[key]; exists {
		return resp, nil
	}

	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"success":true}`))),
	}, nil
}

func (c *mockHTTPClient) SetResponse(method, path string, statusCode int, body []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := method + ":" + path
	c.responses[key] = &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

func (c *mockHTTPClient) SetHandler(handler func(req *http.Request) (*http.Response, error)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handler = handler
}

// mockRequestBuilder implements RequestBuilder interface for testing.
type mockRequestBuilder struct {
	baseURL string
}

func (b *mockRequestBuilder) BuildRequestForEndpoint(unit *EndpointUnit) (*http.Request, error) {
	url := b.baseURL + unit.Path
	return http.NewRequest(unit.Method, url, nil)
}

// =============================================================================
// Test Cases for CircuitBoard
// =============================================================================

func TestNewCircuitBoard(t *testing.T) {
	pool := newMockParameterPool()
	graph := NewDependencyGraph()
	client := newMockHTTPClient()

	board := NewCircuitBoard(nil, pool, graph, nil, client)

	assert.NotNil(t, board)
	assert.NotNil(t, board.GetPool())
	assert.NotNil(t, board.GetGraph())
	assert.NotNil(t, board.GetGuard())
	assert.False(t, board.IsClosed())
}

func TestCircuitBoard_WithConfig(t *testing.T) {
	pool := newMockParameterPool()
	graph := NewDependencyGraph()
	client := newMockHTTPClient()

	config := &CircuitBoardConfig{
		BaseURL:            "http://localhost:8080",
		DefaultTimeout:     10 * time.Second,
		MaxAutoHealRetries: 5,
		EnableAutoHeal:     false,
	}

	board := NewCircuitBoard(config, pool, graph, nil, client)

	assert.Equal(t, "http://localhost:8080", board.config.BaseURL)
	assert.Equal(t, 10*time.Second, board.config.DefaultTimeout)
	assert.Equal(t, 5, board.config.MaxAutoHealRetries)
	assert.False(t, board.config.EnableAutoHeal)
}

func TestCircuitBoard_AddEndpoint(t *testing.T) {
	board := NewCircuitBoard(nil, newMockParameterPool(), NewDependencyGraph(), nil, newMockHTTPClient())

	unit := &EndpointUnit{
		Name:   "test-endpoint",
		Path:   "/test",
		Method: "GET",
	}

	board.AddEndpoint(unit)

	retrieved := board.GetEndpoint("test-endpoint")
	assert.NotNil(t, retrieved)
	assert.Equal(t, "test-endpoint", retrieved.Name)
}

func TestCircuitBoard_CheckDependencies_AllSatisfied(t *testing.T) {
	pool := newMockParameterPool()
	pool.Add(EntityCustomerID, "customer-123", ValueSource{})
	pool.Add(EntityProductID, "product-456", ValueSource{})

	graph := NewDependencyGraph()
	board := NewCircuitBoard(nil, pool, graph, nil, newMockHTTPClient())

	unit := &EndpointUnit{
		Name:      "create-order",
		Path:      "/orders",
		Method:    "POST",
		InputPins: []SemanticType{EntityCustomerID, EntityProductID},
	}
	board.AddEndpoint(unit)

	ctx := context.Background()
	missing, satisfied := board.CheckDependencies(ctx, "create-order")

	assert.True(t, satisfied)
	assert.Empty(t, missing)
}

func TestCircuitBoard_CheckDependencies_MissingValues(t *testing.T) {
	pool := newMockParameterPool()
	pool.Add(EntityCustomerID, "customer-123", ValueSource{})
	// Note: EntityProductID is NOT added

	graph := NewDependencyGraph()
	board := NewCircuitBoard(nil, pool, graph, nil, newMockHTTPClient())

	unit := &EndpointUnit{
		Name:      "create-order",
		Path:      "/orders",
		Method:    "POST",
		InputPins: []SemanticType{EntityCustomerID, EntityProductID},
	}
	board.AddEndpoint(unit)

	ctx := context.Background()
	missing, satisfied := board.CheckDependencies(ctx, "create-order")

	assert.False(t, satisfied)
	assert.Contains(t, missing, EntityProductID)
}

func TestCircuitBoard_GetOrCreate_ExistingValue(t *testing.T) {
	pool := newMockParameterPool()
	pool.Add(EntityCustomerID, "customer-123", ValueSource{})

	graph := NewDependencyGraph()
	board := NewCircuitBoard(nil, pool, graph, nil, newMockHTTPClient())

	ctx := context.Background()
	value, err := board.GetOrCreate(ctx, EntityCustomerID)

	require.NoError(t, err)
	assert.NotNil(t, value)
	assert.Equal(t, "customer-123", value.GetData())
}

func TestCircuitBoard_GetOrCreate_AutoHealDisabled(t *testing.T) {
	pool := newMockParameterPool()
	graph := NewDependencyGraph()

	config := &CircuitBoardConfig{EnableAutoHeal: false}
	board := NewCircuitBoard(config, pool, graph, nil, newMockHTTPClient())

	ctx := context.Background()
	value, err := board.GetOrCreate(ctx, EntityCustomerID)

	assert.Error(t, err)
	assert.Nil(t, value)
	assert.ErrorIs(t, err, ErrMissingDependency)
}

func TestCircuitBoard_GetOrCreate_NoProducer(t *testing.T) {
	pool := newMockParameterPool()
	graph := NewDependencyGraph()

	config := &CircuitBoardConfig{EnableAutoHeal: true}
	board := NewCircuitBoard(config, pool, graph, nil, newMockHTTPClient())

	ctx := context.Background()
	value, err := board.GetOrCreate(ctx, EntityCustomerID)

	assert.Error(t, err)
	assert.Nil(t, value)
	assert.ErrorIs(t, err, ErrNoProducerAvailable)
}

func TestCircuitBoard_Execute_Success(t *testing.T) {
	pool := newMockParameterPool()
	client := newMockHTTPClient()

	responseBody := `{"success":true,"data":{"id":"customer-123"}}`
	client.SetResponse("POST", "/customers", 201, []byte(responseBody))

	graph := NewDependencyGraph()
	config := &CircuitBoardConfig{
		EnableAutoHeal: false, // Disable auto-healing for this test
	}
	board := NewCircuitBoard(config, pool, graph, nil, client)
	board.SetRequestBuilder(&mockRequestBuilder{baseURL: ""})

	unit := &EndpointUnit{
		Name:       "create-customer",
		Path:       "/customers",
		Method:     "POST",
		InputPins:  []SemanticType{}, // No dependencies
		OutputPins: []SemanticType{EntityCustomerID},
	}
	board.AddEndpoint(unit)

	ctx := context.Background()
	result, err := board.Execute(ctx, "create-customer")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 201, result.StatusCode)
	assert.True(t, result.IsSuccess())
}

func TestCircuitBoard_Execute_HTTPError(t *testing.T) {
	pool := newMockParameterPool()
	client := newMockHTTPClient()

	client.SetResponse("POST", "/customers", 500, []byte(`{"error":"internal server error"}`))

	graph := NewDependencyGraph()
	config := &CircuitBoardConfig{EnableAutoHeal: false}
	board := NewCircuitBoard(config, pool, graph, nil, client)
	board.SetRequestBuilder(&mockRequestBuilder{baseURL: ""})

	unit := &EndpointUnit{
		Name:   "create-customer",
		Path:   "/customers",
		Method: "POST",
	}
	board.AddEndpoint(unit)

	ctx := context.Background()
	result, err := board.Execute(ctx, "create-customer")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrExecutionFailed)
}

func TestCircuitBoard_Execute_ExtractsOutputValues(t *testing.T) {
	pool := newMockParameterPool()
	client := newMockHTTPClient()

	responseBody := `{"data":{"id":"customer-456","name":"Test Customer"}}`
	client.SetResponse("POST", "/customers", 201, []byte(responseBody))

	graph := NewDependencyGraph()
	config := &CircuitBoardConfig{
		EnableAutoHeal: false,
		ResponseExtractors: map[string][]ResponseExtractor{
			"create-customer": {
				{JSONPath: "data.id", SemanticType: EntityCustomerID},
				{JSONPath: "data.name", SemanticType: EntityCustomerName},
			},
		},
	}
	board := NewCircuitBoard(config, pool, graph, nil, client)
	board.SetRequestBuilder(&mockRequestBuilder{baseURL: ""})

	unit := &EndpointUnit{
		Name:       "create-customer",
		Path:       "/customers",
		Method:     "POST",
		OutputPins: []SemanticType{EntityCustomerID, EntityCustomerName},
	}
	board.AddEndpoint(unit)

	ctx := context.Background()
	result, err := board.Execute(ctx, "create-customer")

	require.NoError(t, err)
	assert.Equal(t, int64(2), result.ValuesExtracted)

	// Verify values were added to pool
	customerID, err := pool.Get(EntityCustomerID)
	require.NoError(t, err)
	assert.Equal(t, "customer-456", customerID.GetData())

	customerName, err := pool.Get(EntityCustomerName)
	require.NoError(t, err)
	assert.Equal(t, "Test Customer", customerName.GetData())
}

func TestCircuitBoard_Statistics(t *testing.T) {
	pool := newMockParameterPool()
	client := newMockHTTPClient()

	client.SetResponse("GET", "/customers/123", 200, []byte(`{"id":"123"}`))

	graph := NewDependencyGraph()
	config := &CircuitBoardConfig{EnableAutoHeal: false}
	board := NewCircuitBoard(config, pool, graph, nil, client)
	board.SetRequestBuilder(&mockRequestBuilder{baseURL: ""})

	unit := &EndpointUnit{
		Name:   "get-customer",
		Path:   "/customers/123",
		Method: "GET",
	}
	board.AddEndpoint(unit)

	ctx := context.Background()

	// Execute multiple times
	for i := 0; i < 3; i++ {
		_, _ = board.Execute(ctx, "get-customer")
	}

	// Check endpoint stats
	stats, err := board.GetEndpointStats("get-customer")
	require.NoError(t, err)
	assert.Equal(t, int64(3), stats.TotalExecutions)
	assert.Equal(t, int64(3), stats.SuccessfulExecutions)
	assert.Equal(t, int64(0), stats.FailedExecutions)

	// Check board stats
	boardStats := board.GetBoardStats()
	assert.Equal(t, int64(3), boardStats.TotalExecutions)
	assert.Equal(t, int64(3), boardStats.SuccessfulExecutions)
}

func TestCircuitBoard_Close(t *testing.T) {
	board := NewCircuitBoard(nil, newMockParameterPool(), NewDependencyGraph(), nil, newMockHTTPClient())

	assert.False(t, board.IsClosed())

	board.Close()

	assert.True(t, board.IsClosed())

	// Operations on closed board should fail
	ctx := context.Background()
	_, err := board.GetOrCreate(ctx, EntityCustomerID)
	assert.ErrorIs(t, err, ErrBoardClosed)
}

// =============================================================================
// Integration Tests for Self-Healing Capability
// =============================================================================

// TestIntegration_SelfHealing_AutoCreateCustomerBeforeOrder is the acceptance test
// that verifies: "参数池为空时自动创建客户后再创建订单"
// (When parameter pool is empty, auto-create customer before creating order)
func TestIntegration_SelfHealing_AutoCreateCustomerBeforeOrder(t *testing.T) {
	pool := newMockParameterPool()
	client := newMockHTTPClient()

	// Track execution order
	var executionOrder []string
	var mu sync.Mutex

	// Setup HTTP responses for each endpoint
	client.SetHandler(func(req *http.Request) (*http.Response, error) {
		mu.Lock()
		executionOrder = append(executionOrder, req.Method+":"+req.URL.Path)
		mu.Unlock()

		switch req.URL.Path {
		case "/customers":
			if req.Method == "POST" {
				// Create customer returns new customer ID
				body := `{"data":{"id":"cust-auto-123","name":"Auto Created Customer"}}`
				return &http.Response{
					StatusCode: 201,
					Body:       io.NopCloser(bytes.NewReader([]byte(body))),
				}, nil
			}
		case "/orders":
			if req.Method == "POST" {
				// Create order requires customer_id in the request
				body := `{"data":{"id":"order-456","customer_id":"cust-auto-123"}}`
				return &http.Response{
					StatusCode: 201,
					Body:       io.NopCloser(bytes.NewReader([]byte(body))),
				}, nil
			}
		}
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":"not found"}`))),
		}, nil
	})

	// Create dependency graph with producer-consumer relationships
	graph := NewDependencyGraph()

	// Customer endpoint produces EntityCustomerID
	createCustomer := &EndpointUnit{
		Name:       "create-customer",
		Path:       "/customers",
		Method:     "POST",
		InputPins:  []SemanticType{}, // No dependencies
		OutputPins: []SemanticType{EntityCustomerID},
	}
	graph.AddEndpoint(createCustomer)

	// Order endpoint consumes EntityCustomerID
	createOrder := &EndpointUnit{
		Name:       "create-order",
		Path:       "/orders",
		Method:     "POST",
		InputPins:  []SemanticType{EntityCustomerID},
		OutputPins: []SemanticType{OrderSalesID},
	}
	graph.AddEndpoint(createOrder)

	// Build dependencies
	graph.BuildDependencies()

	// Create board with auto-healing enabled
	config := &CircuitBoardConfig{
		EnableAutoHeal: true,
		ResponseExtractors: map[string][]ResponseExtractor{
			"create-customer": {
				{JSONPath: "data.id", SemanticType: EntityCustomerID},
			},
			"create-order": {
				{JSONPath: "data.id", SemanticType: OrderSalesID},
			},
		},
	}

	board := NewCircuitBoard(config, pool, graph, nil, client)
	board.SetRequestBuilder(&mockRequestBuilder{baseURL: ""})

	// Verify pool is empty initially
	assert.Equal(t, 0, pool.Size(EntityCustomerID))

	// Execute create-order - this should auto-trigger create-customer first
	ctx := context.Background()
	result, err := board.Execute(ctx, "create-order")

	// Verify execution succeeded
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsSuccess())

	// Verify execution order: customer created BEFORE order
	mu.Lock()
	defer mu.Unlock()
	require.GreaterOrEqual(t, len(executionOrder), 2, "Expected at least 2 requests")

	// Find indices
	var customerIdx, orderIdx int = -1, -1
	for i, exec := range executionOrder {
		if exec == "POST:/customers" && customerIdx == -1 {
			customerIdx = i
		}
		if exec == "POST:/orders" && orderIdx == -1 {
			orderIdx = i
		}
	}

	assert.Greater(t, orderIdx, customerIdx,
		"Customer should be created before order. Execution order: %v", executionOrder)

	// Verify customer ID was extracted and stored
	customerID, err := pool.Get(EntityCustomerID)
	require.NoError(t, err)
	assert.Equal(t, "cust-auto-123", customerID.GetData())

	// Verify auto-heal statistics
	stats := board.GetBoardStats()
	assert.GreaterOrEqual(t, stats.AutoHealAttempts, int64(1), "Should have attempted auto-healing")
	assert.GreaterOrEqual(t, stats.AutoHealSuccesses, int64(1), "Should have successful auto-healing")
}

// TestIntegration_SelfHealing_ChainedDependencies tests a chain of dependencies:
// Order -> Customer -> Tenant
func TestIntegration_SelfHealing_ChainedDependencies(t *testing.T) {
	pool := newMockParameterPool()
	client := newMockHTTPClient()

	var executionOrder []string
	var mu sync.Mutex

	client.SetHandler(func(req *http.Request) (*http.Response, error) {
		mu.Lock()
		executionOrder = append(executionOrder, req.URL.Path)
		mu.Unlock()

		switch req.URL.Path {
		case "/tenants":
			return &http.Response{
				StatusCode: 201,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"data":{"id":"tenant-001"}}`))),
			}, nil
		case "/customers":
			return &http.Response{
				StatusCode: 201,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"data":{"id":"customer-001"}}`))),
			}, nil
		case "/orders":
			return &http.Response{
				StatusCode: 201,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"data":{"id":"order-001"}}`))),
			}, nil
		}
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{}`))),
		}, nil
	})

	graph := NewDependencyGraph()

	// Tenant (no dependencies)
	graph.AddEndpoint(&EndpointUnit{
		Name:       "create-tenant",
		Path:       "/tenants",
		Method:     "POST",
		OutputPins: []SemanticType{EntityTenantID},
	})

	// Customer depends on Tenant
	graph.AddEndpoint(&EndpointUnit{
		Name:       "create-customer",
		Path:       "/customers",
		Method:     "POST",
		InputPins:  []SemanticType{EntityTenantID},
		OutputPins: []SemanticType{EntityCustomerID},
	})

	// Order depends on Customer
	graph.AddEndpoint(&EndpointUnit{
		Name:       "create-order",
		Path:       "/orders",
		Method:     "POST",
		InputPins:  []SemanticType{EntityCustomerID},
		OutputPins: []SemanticType{OrderSalesID},
	})

	graph.BuildDependencies()

	config := &CircuitBoardConfig{
		EnableAutoHeal: true,
		ResponseExtractors: map[string][]ResponseExtractor{
			"create-tenant":   {{JSONPath: "data.id", SemanticType: EntityTenantID}},
			"create-customer": {{JSONPath: "data.id", SemanticType: EntityCustomerID}},
			"create-order":    {{JSONPath: "data.id", SemanticType: OrderSalesID}},
		},
	}

	board := NewCircuitBoard(config, pool, graph, nil, client)
	board.SetRequestBuilder(&mockRequestBuilder{baseURL: ""})

	// Execute create-order with empty pool
	ctx := context.Background()
	result, err := board.Execute(ctx, "create-order")

	require.NoError(t, err)
	assert.True(t, result.IsSuccess())

	// Verify all entities were created in correct order
	mu.Lock()
	defer mu.Unlock()

	tenantIdx := indexOf(executionOrder, "/tenants")
	customerIdx := indexOf(executionOrder, "/customers")
	orderIdx := indexOf(executionOrder, "/orders")

	assert.Greater(t, customerIdx, tenantIdx, "Tenant should be created before customer")
	assert.Greater(t, orderIdx, customerIdx, "Customer should be created before order")
}

// TestIntegration_SelfHealing_MaxDepthGuard tests that the guard prevents infinite recursion.
func TestIntegration_SelfHealing_MaxDepthGuard(t *testing.T) {
	pool := newMockParameterPool()
	client := newMockHTTPClient()

	// Create a scenario where producer fails to produce the required value
	// This could lead to infinite recursion without the guard
	client.SetHandler(func(req *http.Request) (*http.Response, error) {
		// Producer endpoint succeeds but doesn't produce the expected value
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"data":{}}`))),
		}, nil
	})

	graph := NewDependencyGraph()

	// Endpoint that produces nothing useful
	graph.AddEndpoint(&EndpointUnit{
		Name:       "broken-producer",
		Path:       "/broken",
		Method:     "POST",
		OutputPins: []SemanticType{EntityCustomerID},
	})

	// Consumer that needs the value
	graph.AddEndpoint(&EndpointUnit{
		Name:      "consumer",
		Path:      "/consumer",
		Method:    "POST",
		InputPins: []SemanticType{EntityCustomerID},
	})

	graph.BuildDependencies()

	// Configure with low max retries
	guardConfig := &ProducerChainGuardConfig{
		MaxDepth:       2,
		CooldownPeriod: 10 * time.Millisecond,
	}
	guard := NewProducerChainGuard(guardConfig)

	config := &CircuitBoardConfig{EnableAutoHeal: true}
	board := NewCircuitBoard(config, pool, graph, guard, client)
	board.SetRequestBuilder(&mockRequestBuilder{baseURL: ""})

	// This should eventually fail, not hang
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := board.Execute(ctx, "consumer")

	// Should fail due to missing dependency (producer didn't produce the value)
	assert.Error(t, err)
}

func TestCircuitBoard_ExecuteWithPlan(t *testing.T) {
	pool := newMockParameterPool()
	client := newMockHTTPClient()

	var executionOrder []string
	var mu sync.Mutex

	client.SetHandler(func(req *http.Request) (*http.Response, error) {
		mu.Lock()
		executionOrder = append(executionOrder, req.URL.Path)
		mu.Unlock()

		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"data":{"id":"123"}}`))),
		}, nil
	})

	graph := NewDependencyGraph()

	// Step 1: Create tenant
	graph.AddEndpoint(&EndpointUnit{
		Name:       "step1",
		Path:       "/step1",
		Method:     "POST",
		OutputPins: []SemanticType{EntityTenantID},
	})

	// Step 2: Create customer (depends on tenant)
	graph.AddEndpoint(&EndpointUnit{
		Name:       "step2",
		Path:       "/step2",
		Method:     "POST",
		InputPins:  []SemanticType{EntityTenantID},
		OutputPins: []SemanticType{EntityCustomerID},
	})

	// Step 3: Create order (depends on customer)
	graph.AddEndpoint(&EndpointUnit{
		Name:      "step3",
		Path:      "/step3",
		Method:    "POST",
		InputPins: []SemanticType{EntityCustomerID},
	})

	graph.BuildDependencies()

	config := &CircuitBoardConfig{
		EnableAutoHeal: false, // Disable auto-heal to test explicit plan
		ResponseExtractors: map[string][]ResponseExtractor{
			"step1": {{JSONPath: "data.id", SemanticType: EntityTenantID}},
			"step2": {{JSONPath: "data.id", SemanticType: EntityCustomerID}},
		},
	}

	board := NewCircuitBoard(config, pool, graph, nil, client)
	board.SetRequestBuilder(&mockRequestBuilder{baseURL: ""})

	ctx := context.Background()
	results, err := board.ExecuteWithPlan(ctx, "step3")

	require.NoError(t, err)
	assert.Len(t, results, 3)

	// Verify order: step1 -> step2 -> step3
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"/step1", "/step2", "/step3"}, executionOrder)
}

// =============================================================================
// Test Helpers
// =============================================================================

func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkCircuitBoard_Execute(b *testing.B) {
	pool := newMockParameterPool()
	client := newMockHTTPClient()
	client.SetResponse("GET", "/test", 200, []byte(`{"data":{"id":"123"}}`))

	graph := NewDependencyGraph()
	config := &CircuitBoardConfig{EnableAutoHeal: false}
	board := NewCircuitBoard(config, pool, graph, nil, client)
	board.SetRequestBuilder(&mockRequestBuilder{baseURL: ""})

	unit := &EndpointUnit{
		Name:   "test",
		Path:   "/test",
		Method: "GET",
	}
	board.AddEndpoint(unit)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = board.Execute(ctx, "test")
	}
}

func BenchmarkCircuitBoard_GetOrCreate_Hit(b *testing.B) {
	pool := newMockParameterPool()
	pool.Add(EntityCustomerID, "customer-123", ValueSource{})

	graph := NewDependencyGraph()
	board := NewCircuitBoard(nil, pool, graph, nil, newMockHTTPClient())

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = board.GetOrCreate(ctx, EntityCustomerID)
	}
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestCircuitBoard_EndpointStats_AverageLatency(t *testing.T) {
	stats := &EndpointStats{
		TotalExecutions: 10,
		TotalLatency:    int64(100 * time.Millisecond),
	}

	avg := stats.AverageLatency()
	assert.Equal(t, 10*time.Millisecond, avg)
}

func TestCircuitBoard_EndpointStats_SuccessRate(t *testing.T) {
	stats := &EndpointStats{
		TotalExecutions:      100,
		SuccessfulExecutions: 95,
	}

	rate := stats.SuccessRate()
	assert.Equal(t, 95.0, rate)
}

func TestCircuitBoard_EndpointStats_ZeroExecutions(t *testing.T) {
	stats := &EndpointStats{}

	assert.Equal(t, time.Duration(0), stats.AverageLatency())
	assert.Equal(t, 0.0, stats.SuccessRate())
}

func TestCircuitBoard_ExtractJSONPath(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]any
		path     string
		expected any
	}{
		{
			name:     "simple path",
			data:     map[string]any{"id": "123"},
			path:     "id",
			expected: "123",
		},
		{
			name:     "nested path",
			data:     map[string]any{"data": map[string]any{"id": "456"}},
			path:     "data.id",
			expected: "456",
		},
		{
			name:     "deep nested path",
			data:     map[string]any{"data": map[string]any{"user": map[string]any{"name": "John"}}},
			path:     "data.user.name",
			expected: "John",
		},
		{
			name:     "missing path",
			data:     map[string]any{"id": "123"},
			path:     "nonexistent",
			expected: nil,
		},
		{
			name:     "empty path",
			data:     map[string]any{"id": "123"},
			path:     "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONPath(tt.data, tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCircuitBoard_ResetStats(t *testing.T) {
	pool := newMockParameterPool()
	client := newMockHTTPClient()
	client.SetResponse("GET", "/test", 200, []byte(`{}`))

	board := NewCircuitBoard(nil, pool, NewDependencyGraph(), nil, client)
	board.SetRequestBuilder(&mockRequestBuilder{baseURL: ""})

	unit := &EndpointUnit{Name: "test", Path: "/test", Method: "GET"}
	board.AddEndpoint(unit)

	ctx := context.Background()
	_, _ = board.Execute(ctx, "test")

	// Verify stats are non-zero
	stats := board.GetBoardStats()
	assert.Equal(t, int64(1), stats.TotalExecutions)

	// Reset
	board.ResetStats()

	// Verify stats are reset
	stats = board.GetBoardStats()
	assert.Equal(t, int64(0), stats.TotalExecutions)
}
