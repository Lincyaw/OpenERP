package executor

import (
	"io"
	"net/http"
	"testing"

	"github.com/example/erp/tools/loadgen/internal/circuit"
	"github.com/example/erp/tools/loadgen/internal/pool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestBuilder_SimpleCases(t *testing.T) {
	tests := []struct {
		name    string
		unit    *circuit.EndpointUnit
		setup   func(p pool.ParameterPool)
		wantErr bool
		errMsg  string
		check   func(t *testing.T, req *http.Request)
	}{
		{
			name: "GET request with path parameter",
			unit: &circuit.EndpointUnit{
				Name:      "get-customer",
				Path:      "/customers/{id}",
				Method:    "GET",
				InputPins: []circuit.SemanticType{circuit.EntityCustomerID},
			},
			setup: func(p pool.ParameterPool) {
				p.Add(circuit.EntityCustomerID, "12345", pool.ValueSource{})
			},
			check: func(t *testing.T, req *http.Request) {
				assert.Equal(t, "GET", req.Method)
				assert.Equal(t, "/customers/12345", req.URL.Path)
			},
		},
		{
			name: "POST request with JSON body",
			unit: &circuit.EndpointUnit{
				Name:   "create-customer",
				Path:   "/customers",
				Method: "POST",
				InputPins: []circuit.SemanticType{
					circuit.EntityCustomerName,
					circuit.CommonEmail,
				},
			},
			setup: func(p pool.ParameterPool) {
				p.Add(circuit.EntityCustomerName, "John Doe", pool.ValueSource{})
				p.Add(circuit.CommonEmail, "john@example.com", pool.ValueSource{})
			},
			check: func(t *testing.T, req *http.Request) {
				assert.Equal(t, "POST", req.Method)
				assert.Equal(t, "/customers", req.URL.Path)
				assert.Equal(t, "application/json", req.Header.Get("Content-Type"))

				body, err := io.ReadAll(req.Body)
				require.NoError(t, err)

				// With disambiguation, fields use full semantic type
				assert.Contains(t, string(body), `"entity_customer_name":"John Doe"`)
				assert.Contains(t, string(body), `"common_email":"john@example.com"`)
			},
		},
		{
			name: "Request with headers",
			unit: &circuit.EndpointUnit{
				Name:   "authenticated-request",
				Path:   "/protected",
				Method: "GET",
				InputPins: []circuit.SemanticType{
					circuit.SystemAccessToken,
				},
			},
			setup: func(p pool.ParameterPool) {
				p.Add(circuit.SystemAccessToken, "Bearer token123", pool.ValueSource{})
			},
			check: func(t *testing.T, req *http.Request) {
				assert.Equal(t, "GET", req.Method)
				assert.Equal(t, "/protected", req.URL.Path)
				assert.Equal(t, "Bearer token123", req.Header.Get("Access_Token"))
			},
		},
		{
			name: "Complex path with multiple parameters",
			unit: &circuit.EndpointUnit{
				Name:   "update-inventory",
				Path:   "/warehouses/{warehouse_id}/products/{product_id}/inventory",
				Method: "PUT",
				InputPins: []circuit.SemanticType{
					circuit.EntityWarehouseID,
					circuit.EntityProductID,
					circuit.InventoryStockQuantity,
				},
			},
			setup: func(p pool.ParameterPool) {
				p.Add(circuit.EntityWarehouseID, "WH-001", pool.ValueSource{})
				p.Add(circuit.EntityProductID, "PROD-123", pool.ValueSource{})
				p.Add(circuit.InventoryStockQuantity, 100, pool.ValueSource{})
			},
			check: func(t *testing.T, req *http.Request) {
				assert.Equal(t, "PUT", req.Method)
				assert.Equal(t, "/warehouses/WH-001/products/PROD-123/inventory", req.URL.Path)
				assert.Equal(t, "application/json", req.Header.Get("Content-Type"))

				body, err := io.ReadAll(req.Body)
				require.NoError(t, err)

				// With disambiguation
				assert.Contains(t, string(body), `"inventory_stock_quantity":100`)
			},
		},
		{
			name: "Missing required path parameter",
			unit: &circuit.EndpointUnit{
				Name:      "get-customer",
				Path:      "/customers/{id}",
				Method:    "GET",
				InputPins: []circuit.SemanticType{circuit.EntityCustomerID},
			},
			setup: func(p pool.ParameterPool) {
				// Don't add the required parameter
			},
			wantErr: true,
			errMsg:  "missing required path parameters: id",
		},
		{
			name: "Nil endpoint unit",
			unit: nil,
			wantErr: true,
			errMsg:  "endpoint unit is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create parameter pool
			paramPool := pool.NewShardedPool(nil)
			defer paramPool.Close()

			// Setup test data
			if tt.setup != nil {
				tt.setup(paramPool)
			}

			// Create request builder
			rb := NewRequestBuilder(paramPool)

			// Build request
			req, err := rb.BuildRequest(tt.unit)

			// Check error
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, req)

			// Run checks
			if tt.check != nil {
				tt.check(t, req)
			}
		})
	}
}

func TestRequestBuilder_SalesOrder(t *testing.T) {
	// Create parameter pool with sales order data
	paramPool := pool.NewShardedPool(nil)
	defer paramPool.Close()

	// Add required parameters
	paramPool.Add(circuit.EntityCustomerID, "CUST-12345", pool.ValueSource{})
	paramPool.Add(circuit.OrderSalesNumber, "SO-2024-001", pool.ValueSource{})
	paramPool.Add(circuit.OrderItemQuantity, 5, pool.ValueSource{})
	paramPool.Add(circuit.EntityProductID, "PROD-456", pool.ValueSource{})
	paramPool.Add(circuit.FinancePaymentAmount, 299.95, pool.ValueSource{})

	// Create request builder
	rb := NewRequestBuilder(paramPool)

	// Build sales order request
	req, err := rb.BuildSalesOrderRequest()
	require.NoError(t, err)
	require.NotNil(t, req)

	// Verify request
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "/trade/sales-orders", req.URL.Path)
	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))

	// Parse and verify body
	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)

	// With disambiguation, all fields use full semantic type
	assert.Contains(t, string(body), `"entity_customer_id":"CUST-12345"`)
	assert.Contains(t, string(body), `"order_sales_number":"SO-2024-001"`)
	assert.Contains(t, string(body), `"order_item_quantity":5`)
	assert.Contains(t, string(body), `"entity_product_id":"PROD-456"`)
	assert.Contains(t, string(body), `"finance_payment_amount":299.95`)
}