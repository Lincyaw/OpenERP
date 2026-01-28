package executor

import (
	"encoding/json"
	"io"
	"testing"

	"github.com/example/erp/tools/loadgen/internal/circuit"
	"github.com/example/erp/tools/loadgen/internal/pool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Extended Request Builder Tests
// ============================================================================

func TestBuildRequestFromExtended_PathParameters(t *testing.T) {
	tests := []struct {
		name     string
		unit     *ExtendedEndpointUnit
		poolData map[circuit.SemanticType]any
		wantPath string
		wantErr  bool
	}{
		{
			name: "single path parameter",
			unit: &ExtendedEndpointUnit{
				Name:   "get-customer",
				Path:   "/customers/{customer_id}",
				Method: "GET",
				InputPins: []InputPin{
					{
						Name:         "customer_id",
						Location:     paramLocationPath,
						SemanticType: circuit.EntityCustomerID,
						Required:     true,
					},
				},
			},
			poolData: map[circuit.SemanticType]any{
				circuit.EntityCustomerID: "CUST-12345",
			},
			wantPath: "/customers/CUST-12345",
		},
		{
			name: "multiple path parameters",
			unit: &ExtendedEndpointUnit{
				Name:   "get-warehouse-product",
				Path:   "/warehouses/{warehouse_id}/products/{product_id}",
				Method: "GET",
				InputPins: []InputPin{
					{
						Name:         "warehouse_id",
						Location:     paramLocationPath,
						SemanticType: circuit.EntityWarehouseID,
						Required:     true,
					},
					{
						Name:         "product_id",
						Location:     paramLocationPath,
						SemanticType: circuit.EntityProductID,
						Required:     true,
					},
				},
			},
			poolData: map[circuit.SemanticType]any{
				circuit.EntityWarehouseID: "WH-001",
				circuit.EntityProductID:   "PROD-456",
			},
			wantPath: "/warehouses/WH-001/products/PROD-456",
		},
		{
			name: "path parameter with special characters",
			unit: &ExtendedEndpointUnit{
				Name:   "get-customer",
				Path:   "/customers/{id}",
				Method: "GET",
				InputPins: []InputPin{
					{
						Name:         "id",
						Location:     paramLocationPath,
						SemanticType: circuit.EntityCustomerID,
						Required:     true,
					},
				},
			},
			poolData: map[circuit.SemanticType]any{
				circuit.EntityCustomerID: "user@example.com",
			},
			// Note: req.URL.Path returns decoded path, but PathEscape ensures safe URL transmission
			// The @ character is percent-encoded in the actual request URL
			wantPath: "/customers/user@example.com",
		},
		{
			name: "colon-style path parameter",
			unit: &ExtendedEndpointUnit{
				Name:   "get-customer",
				Path:   "/customers/:id",
				Method: "GET",
				InputPins: []InputPin{
					{
						Name:         "id",
						Location:     paramLocationPath,
						SemanticType: circuit.EntityCustomerID,
						Required:     true,
					},
				},
			},
			poolData: map[circuit.SemanticType]any{
				circuit.EntityCustomerID: "12345",
			},
			wantPath: "/customers/12345",
		},
		{
			name: "missing required path parameter",
			unit: &ExtendedEndpointUnit{
				Name:   "get-customer",
				Path:   "/customers/{id}",
				Method: "GET",
				InputPins: []InputPin{
					{
						Name:         "id",
						Location:     paramLocationPath,
						SemanticType: circuit.EntityCustomerID,
						Required:     true,
					},
				},
			},
			poolData: map[circuit.SemanticType]any{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paramPool := pool.NewShardedPool(nil)
			defer paramPool.Close()

			for semType, value := range tt.poolData {
				paramPool.Add(semType, value, pool.ValueSource{})
			}

			rb := NewRequestBuilder(paramPool)
			req, err := rb.BuildRequestFromExtended(tt.unit)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, req)
			assert.Equal(t, tt.wantPath, req.URL.Path)
		})
	}
}

func TestBuildRequestFromExtended_QueryParameters(t *testing.T) {
	tests := []struct {
		name      string
		unit      *ExtendedEndpointUnit
		poolData  map[circuit.SemanticType]any
		wantQuery map[string]string
		wantErr   bool
	}{
		{
			name: "single query parameter",
			unit: &ExtendedEndpointUnit{
				Name:   "list-customers",
				Path:   "/customers",
				Method: "GET",
				InputPins: []InputPin{
					{
						Name:         "page",
						Location:     paramLocationQuery,
						SemanticType: circuit.CommonPage,
						Required:     false,
					},
				},
			},
			poolData: map[circuit.SemanticType]any{
				circuit.CommonPage: 1,
			},
			wantQuery: map[string]string{"page": "1"},
		},
		{
			name: "multiple query parameters",
			unit: &ExtendedEndpointUnit{
				Name:   "list-customers",
				Path:   "/customers",
				Method: "GET",
				InputPins: []InputPin{
					{
						Name:         "page",
						Location:     paramLocationQuery,
						SemanticType: circuit.CommonPage,
					},
					{
						Name:         "page_size",
						Location:     paramLocationQuery,
						SemanticType: circuit.CommonPageSize,
					},
					{
						Name:         "sort_by",
						Location:     paramLocationQuery,
						SemanticType: circuit.CommonSortBy,
					},
				},
			},
			poolData: map[circuit.SemanticType]any{
				circuit.CommonPage:     2,
				circuit.CommonPageSize: 50,
				circuit.CommonSortBy:   "name",
			},
			wantQuery: map[string]string{
				"page":      "2",
				"page_size": "50",
				"sort_by":   "name",
			},
		},
		{
			name: "optional query parameter missing",
			unit: &ExtendedEndpointUnit{
				Name:   "list-customers",
				Path:   "/customers",
				Method: "GET",
				InputPins: []InputPin{
					{
						Name:         "keyword",
						Location:     paramLocationQuery,
						SemanticType: circuit.CommonKeyword,
						Required:     false,
					},
				},
			},
			poolData:  map[circuit.SemanticType]any{},
			wantQuery: map[string]string{},
		},
		{
			name: "required query parameter missing",
			unit: &ExtendedEndpointUnit{
				Name:   "search-customers",
				Path:   "/customers/search",
				Method: "GET",
				InputPins: []InputPin{
					{
						Name:         "keyword",
						Location:     paramLocationQuery,
						SemanticType: circuit.CommonKeyword,
						Required:     true,
					},
				},
			},
			poolData: map[circuit.SemanticType]any{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paramPool := pool.NewShardedPool(nil)
			defer paramPool.Close()

			for semType, value := range tt.poolData {
				paramPool.Add(semType, value, pool.ValueSource{})
			}

			rb := NewRequestBuilder(paramPool)
			req, err := rb.BuildRequestFromExtended(tt.unit)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, req)

			query := req.URL.Query()
			for key, expectedValue := range tt.wantQuery {
				assert.Equal(t, expectedValue, query.Get(key), "query param %s", key)
			}
		})
	}
}

func TestBuildRequestFromExtended_HeaderParameters(t *testing.T) {
	tests := []struct {
		name        string
		unit        *ExtendedEndpointUnit
		poolData    map[circuit.SemanticType]any
		wantHeaders map[string]string
		wantErr     bool
	}{
		{
			name: "authorization header",
			unit: &ExtendedEndpointUnit{
				Name:   "protected-resource",
				Path:   "/protected",
				Method: "GET",
				InputPins: []InputPin{
					{
						Name:         "Authorization",
						Location:     paramLocationHeader,
						SemanticType: circuit.SystemAccessToken,
						Required:     true,
					},
				},
			},
			poolData: map[circuit.SemanticType]any{
				circuit.SystemAccessToken: "Bearer token123",
			},
			wantHeaders: map[string]string{
				"Authorization": "Bearer token123",
			},
		},
		{
			name: "custom headers",
			unit: &ExtendedEndpointUnit{
				Name:   "api-request",
				Path:   "/api/data",
				Method: "GET",
				InputPins: []InputPin{
					{
						Name:         "X-Api-Key",
						Location:     paramLocationHeader,
						SemanticType: circuit.SystemAPIKey,
						Required:     true,
					},
					{
						Name:         "X-Request-ID",
						Location:     paramLocationHeader,
						SemanticType: circuit.SemanticType("common.request_id"),
						Required:     false,
					},
				},
			},
			poolData: map[circuit.SemanticType]any{
				circuit.SystemAPIKey:                      "api-key-value",
				circuit.SemanticType("common.request_id"): "req-12345",
			},
			wantHeaders: map[string]string{
				"X-Api-Key":    "api-key-value",
				"X-Request-Id": "req-12345",
			},
		},
		{
			name: "missing required header",
			unit: &ExtendedEndpointUnit{
				Name:   "protected-resource",
				Path:   "/protected",
				Method: "GET",
				InputPins: []InputPin{
					{
						Name:         "Authorization",
						Location:     paramLocationHeader,
						SemanticType: circuit.SystemAccessToken,
						Required:     true,
					},
				},
			},
			poolData: map[circuit.SemanticType]any{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paramPool := pool.NewShardedPool(nil)
			defer paramPool.Close()

			for semType, value := range tt.poolData {
				paramPool.Add(semType, value, pool.ValueSource{})
			}

			rb := NewRequestBuilder(paramPool)
			req, err := rb.BuildRequestFromExtended(tt.unit)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, req)

			for key, expectedValue := range tt.wantHeaders {
				assert.Equal(t, expectedValue, req.Header.Get(key), "header %s", key)
			}
		})
	}
}

func TestBuildRequestFromExtended_BodyParameters(t *testing.T) {
	tests := []struct {
		name     string
		unit     *ExtendedEndpointUnit
		poolData map[circuit.SemanticType]any
		wantBody map[string]any
		wantErr  bool
	}{
		{
			name: "simple body parameters",
			unit: &ExtendedEndpointUnit{
				Name:   "create-customer",
				Path:   "/customers",
				Method: "POST",
				InputPins: []InputPin{
					{
						Name:         "name",
						Location:     paramLocationBody,
						SemanticType: circuit.EntityCustomerName,
						Required:     true,
					},
					{
						Name:         "email",
						Location:     paramLocationBody,
						SemanticType: circuit.CommonEmail,
						Required:     true,
					},
				},
			},
			poolData: map[circuit.SemanticType]any{
				circuit.EntityCustomerName: "John Doe",
				circuit.CommonEmail:        "john@example.com",
			},
			wantBody: map[string]any{
				"name":  "John Doe",
				"email": "john@example.com",
			},
		},
		{
			name: "body with numeric values",
			unit: &ExtendedEndpointUnit{
				Name:   "update-inventory",
				Path:   "/inventory",
				Method: "PUT",
				InputPins: []InputPin{
					{
						Name:         "product_id",
						Location:     paramLocationBody,
						SemanticType: circuit.EntityProductID,
						Required:     true,
					},
					{
						Name:         "quantity",
						Location:     paramLocationBody,
						SemanticType: circuit.InventoryStockQuantity,
						Required:     true,
					},
				},
			},
			poolData: map[circuit.SemanticType]any{
				circuit.EntityProductID:        "PROD-123",
				circuit.InventoryStockQuantity: 100,
			},
			wantBody: map[string]any{
				"product_id": "PROD-123",
				"quantity":   float64(100), // JSON unmarshals numbers as float64
			},
		},
		{
			name: "missing required body parameter",
			unit: &ExtendedEndpointUnit{
				Name:   "create-customer",
				Path:   "/customers",
				Method: "POST",
				InputPins: []InputPin{
					{
						Name:         "name",
						Location:     paramLocationBody,
						SemanticType: circuit.EntityCustomerName,
						Required:     true,
					},
				},
			},
			poolData: map[circuit.SemanticType]any{},
			wantErr:  true,
		},
		{
			name: "optional body parameter missing",
			unit: &ExtendedEndpointUnit{
				Name:   "create-customer",
				Path:   "/customers",
				Method: "POST",
				InputPins: []InputPin{
					{
						Name:         "name",
						Location:     paramLocationBody,
						SemanticType: circuit.EntityCustomerName,
						Required:     true,
					},
					{
						Name:         "notes",
						Location:     paramLocationBody,
						SemanticType: circuit.CommonNote,
						Required:     false,
					},
				},
			},
			poolData: map[circuit.SemanticType]any{
				circuit.EntityCustomerName: "John Doe",
			},
			wantBody: map[string]any{
				"name": "John Doe",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paramPool := pool.NewShardedPool(nil)
			defer paramPool.Close()

			for semType, value := range tt.poolData {
				paramPool.Add(semType, value, pool.ValueSource{})
			}

			rb := NewRequestBuilder(paramPool)
			req, err := rb.BuildRequestFromExtended(tt.unit)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, req)
			assert.Equal(t, "application/json", req.Header.Get("Content-Type"))

			// Parse and verify body
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)

			var bodyMap map[string]any
			err = json.Unmarshal(body, &bodyMap)
			require.NoError(t, err)

			for key, expectedValue := range tt.wantBody {
				assert.Equal(t, expectedValue, bodyMap[key], "body field %s", key)
			}
		})
	}
}

func TestBuildRequestFromExtended_NestedBody(t *testing.T) {
	tests := []struct {
		name     string
		unit     *ExtendedEndpointUnit
		poolData map[circuit.SemanticType]any
		wantBody map[string]any
		wantErr  bool
	}{
		{
			name: "nested object using JSONPath",
			unit: &ExtendedEndpointUnit{
				Name:   "create-order",
				Path:   "/orders",
				Method: "POST",
				InputPins: []InputPin{
					{
						Name:         "customer_id",
						Location:     paramLocationBody,
						SemanticType: circuit.EntityCustomerID,
						Required:     true,
					},
					{
						Name:         "city",
						Location:     paramLocationBody,
						SemanticType: circuit.SemanticType("common.city"),
						JSONPath:     "shipping_address.city",
						Required:     false,
					},
				},
			},
			poolData: map[circuit.SemanticType]any{
				circuit.EntityCustomerID:            "CUST-123",
				circuit.SemanticType("common.city"): "New York",
			},
			wantBody: map[string]any{
				"customer_id": "CUST-123",
				"shipping_address": map[string]any{
					"city": "New York",
				},
			},
		},
		{
			name: "deeply nested path",
			unit: &ExtendedEndpointUnit{
				Name:   "create-order",
				Path:   "/orders",
				Method: "POST",
				InputPins: []InputPin{
					{
						Name:         "postal_code",
						Location:     paramLocationBody,
						SemanticType: circuit.SemanticType("common.postal_code"),
						JSONPath:     "billing.address.postal_code",
						Required:     true,
					},
				},
			},
			poolData: map[circuit.SemanticType]any{
				circuit.SemanticType("common.postal_code"): "10001",
			},
			wantBody: map[string]any{
				"billing": map[string]any{
					"address": map[string]any{
						"postal_code": "10001",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paramPool := pool.NewShardedPool(nil)
			defer paramPool.Close()

			for semType, value := range tt.poolData {
				paramPool.Add(semType, value, pool.ValueSource{})
			}

			rb := NewRequestBuilder(paramPool)
			req, err := rb.BuildRequestFromExtended(tt.unit)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, req)

			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)

			var bodyMap map[string]any
			err = json.Unmarshal(body, &bodyMap)
			require.NoError(t, err)

			// Deep compare
			assertDeepEqual(t, tt.wantBody, bodyMap)
		})
	}
}

// assertDeepEqual recursively compares two maps.
func assertDeepEqual(t *testing.T, expected, actual map[string]any) {
	for key, expectedValue := range expected {
		actualValue, exists := actual[key]
		assert.True(t, exists, "missing key: %s", key)

		switch ev := expectedValue.(type) {
		case map[string]any:
			av, ok := actualValue.(map[string]any)
			require.True(t, ok, "expected map for key %s, got %T", key, actualValue)
			assertDeepEqual(t, ev, av)
		default:
			assert.Equal(t, expectedValue, actualValue, "value mismatch for key %s", key)
		}
	}
}

func TestBuildRequestFromExtended_MixedParameters(t *testing.T) {
	t.Run("path, query, header, and body parameters", func(t *testing.T) {
		paramPool := pool.NewShardedPool(nil)
		defer paramPool.Close()

		// Add all parameters to pool
		paramPool.Add(circuit.EntityCustomerID, "CUST-12345", pool.ValueSource{})
		paramPool.Add(circuit.CommonPage, 1, pool.ValueSource{})
		paramPool.Add(circuit.CommonPageSize, 20, pool.ValueSource{})
		paramPool.Add(circuit.SystemAccessToken, "Bearer token123", pool.ValueSource{})
		paramPool.Add(circuit.CommonNote, "Test note", pool.ValueSource{})

		unit := &ExtendedEndpointUnit{
			Name:   "update-customer-orders",
			Path:   "/customers/{customer_id}/orders",
			Method: "POST",
			InputPins: []InputPin{
				{
					Name:         "customer_id",
					Location:     paramLocationPath,
					SemanticType: circuit.EntityCustomerID,
					Required:     true,
				},
				{
					Name:         "page",
					Location:     paramLocationQuery,
					SemanticType: circuit.CommonPage,
				},
				{
					Name:         "page_size",
					Location:     paramLocationQuery,
					SemanticType: circuit.CommonPageSize,
				},
				{
					Name:         "Authorization",
					Location:     paramLocationHeader,
					SemanticType: circuit.SystemAccessToken,
					Required:     true,
				},
				{
					Name:         "notes",
					Location:     paramLocationBody,
					SemanticType: circuit.CommonNote,
				},
			},
		}

		rb := NewRequestBuilder(paramPool)
		req, err := rb.BuildRequestFromExtended(unit)
		require.NoError(t, err)
		require.NotNil(t, req)

		// Verify path
		assert.Equal(t, "/customers/CUST-12345/orders", req.URL.Path)

		// Verify query
		assert.Equal(t, "1", req.URL.Query().Get("page"))
		assert.Equal(t, "20", req.URL.Query().Get("page_size"))

		// Verify headers
		assert.Equal(t, "Bearer token123", req.Header.Get("Authorization"))
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))

		// Verify body
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)

		var bodyMap map[string]any
		err = json.Unmarshal(body, &bodyMap)
		require.NoError(t, err)
		assert.Equal(t, "Test note", bodyMap["notes"])
	})
}

func TestBuildRequestFromExtended_Validation(t *testing.T) {
	t.Run("nil unit returns error", func(t *testing.T) {
		paramPool := pool.NewShardedPool(nil)
		defer paramPool.Close()

		rb := NewRequestBuilder(paramPool)
		_, err := rb.BuildRequestFromExtended(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "endpoint unit is nil")
	})

	t.Run("empty path returns error", func(t *testing.T) {
		paramPool := pool.NewShardedPool(nil)
		defer paramPool.Close()

		unit := &ExtendedEndpointUnit{
			Name:   "test",
			Method: "GET",
			Path:   "",
		}

		rb := NewRequestBuilder(paramPool)
		_, err := rb.BuildRequestFromExtended(unit)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "path is empty")
	})

	t.Run("invalid HTTP method returns error", func(t *testing.T) {
		paramPool := pool.NewShardedPool(nil)
		defer paramPool.Close()

		unit := &ExtendedEndpointUnit{
			Name:   "test",
			Method: "INVALID",
			Path:   "/test",
		}

		rb := NewRequestBuilder(paramPool)
		_, err := rb.BuildRequestFromExtended(unit)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid HTTP method")
	})
}

// ============================================================================
// Sales Order Specific Tests
// ============================================================================

func TestBuildSalesOrderFromPool(t *testing.T) {
	t.Run("complete sales order", func(t *testing.T) {
		paramPool := pool.NewShardedPool(nil)
		defer paramPool.Close()

		// Add required parameters
		paramPool.Add(circuit.EntityCustomerID, "CUST-001", pool.ValueSource{})
		paramPool.Add(circuit.EntityProductID, "PROD-100", pool.ValueSource{})
		paramPool.Add(circuit.OrderItemQuantity, 5, pool.ValueSource{})
		paramPool.Add(circuit.OrderItemPrice, 29.99, pool.ValueSource{})
		paramPool.Add(circuit.CommonNote, "Rush order", pool.ValueSource{})

		rb := NewRequestBuilder(paramPool)
		req, err := rb.BuildSalesOrderFromPool()
		require.NoError(t, err)
		require.NotNil(t, req)

		// Verify request basics
		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "/trade/sales-orders", req.URL.Path)
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))

		// Parse body
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)

		var bodyMap map[string]any
		err = json.Unmarshal(body, &bodyMap)
		require.NoError(t, err)

		// Verify customer_id
		assert.Equal(t, "CUST-001", bodyMap["customer_id"])

		// Verify items array
		items, ok := bodyMap["items"].([]any)
		require.True(t, ok, "items should be an array")
		require.Len(t, items, 1, "should have 1 item")

		item := items[0].(map[string]any)
		assert.Equal(t, "PROD-100", item["product_id"])
		assert.Equal(t, float64(5), item["quantity"])
		assert.Equal(t, float64(29.99), item["unit_price"])

		// Verify notes
		assert.Equal(t, "Rush order", bodyMap["notes"])
	})

	t.Run("sales order with shipping address", func(t *testing.T) {
		paramPool := pool.NewShardedPool(nil)
		defer paramPool.Close()

		// Add required parameters
		paramPool.Add(circuit.EntityCustomerID, "CUST-002", pool.ValueSource{})
		paramPool.Add(circuit.EntityProductID, "PROD-200", pool.ValueSource{})
		paramPool.Add(circuit.OrderItemQuantity, 2, pool.ValueSource{})

		// Add address components
		paramPool.Add(circuit.CommonAddress, "123 Main St", pool.ValueSource{})
		paramPool.Add(circuit.SemanticType("common.city"), "Boston", pool.ValueSource{})
		paramPool.Add(circuit.SemanticType("common.state"), "MA", pool.ValueSource{})
		paramPool.Add(circuit.SemanticType("common.postal_code"), "02101", pool.ValueSource{})
		paramPool.Add(circuit.SemanticType("common.country"), "USA", pool.ValueSource{})

		rb := NewRequestBuilder(paramPool)
		req, err := rb.BuildSalesOrderFromPool()
		require.NoError(t, err)
		require.NotNil(t, req)

		// Parse body
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)

		var bodyMap map[string]any
		err = json.Unmarshal(body, &bodyMap)
		require.NoError(t, err)

		// Verify shipping address
		address, ok := bodyMap["shipping_address"].(map[string]any)
		require.True(t, ok, "shipping_address should be an object")
		assert.Equal(t, "123 Main St", address["street"])
		assert.Equal(t, "Boston", address["city"])
		assert.Equal(t, "MA", address["state"])
		assert.Equal(t, "02101", address["postal_code"])
		assert.Equal(t, "USA", address["country"])
	})

	t.Run("sales order with pre-built items array", func(t *testing.T) {
		paramPool := pool.NewShardedPool(nil)
		defer paramPool.Close()

		// Add required parameters
		paramPool.Add(circuit.EntityCustomerID, "CUST-003", pool.ValueSource{})

		// Add items as array
		items := []any{
			map[string]any{
				"product_id": "PROD-A",
				"quantity":   3,
				"unit_price": 15.00,
			},
			map[string]any{
				"product_id": "PROD-B",
				"quantity":   1,
				"unit_price": 99.99,
			},
		}
		paramPool.Add(circuit.SemanticType("order.items"), items, pool.ValueSource{})

		rb := NewRequestBuilder(paramPool)
		req, err := rb.BuildSalesOrderFromPool()
		require.NoError(t, err)
		require.NotNil(t, req)

		// Parse body
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)

		var bodyMap map[string]any
		err = json.Unmarshal(body, &bodyMap)
		require.NoError(t, err)

		// Verify items array
		resultItems, ok := bodyMap["items"].([]any)
		require.True(t, ok, "items should be an array")
		require.Len(t, resultItems, 2, "should have 2 items")
	})

	t.Run("sales order missing customer_id fails", func(t *testing.T) {
		paramPool := pool.NewShardedPool(nil)
		defer paramPool.Close()

		// Only add product ID, not customer ID
		paramPool.Add(circuit.EntityProductID, "PROD-100", pool.ValueSource{})

		rb := NewRequestBuilder(paramPool)
		_, err := rb.BuildSalesOrderFromPool()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "customer_id")
	})
}

// ============================================================================
// JSON Path Parsing Tests
// ============================================================================

func TestParseJSONPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []any
	}{
		{
			name:     "simple field",
			path:     "name",
			expected: []any{"name"},
		},
		{
			name:     "nested fields",
			path:     "address.city",
			expected: []any{"address", "city"},
		},
		{
			name:     "deeply nested",
			path:     "a.b.c.d",
			expected: []any{"a", "b", "c", "d"},
		},
		{
			name:     "array index",
			path:     "items[0]",
			expected: []any{"items", 0},
		},
		{
			name:     "array with nested field",
			path:     "items[0].product_id",
			expected: []any{"items", 0, "product_id"},
		},
		{
			name:     "multiple arrays",
			path:     "orders[0].items[1].name",
			expected: []any{"orders", 0, "items", 1, "name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseJSONPath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSetNestedValue(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		value    any
		expected map[string]any
	}{
		{
			name:  "simple field",
			path:  "name",
			value: "John",
			expected: map[string]any{
				"name": "John",
			},
		},
		{
			name:  "nested field",
			path:  "address.city",
			value: "Boston",
			expected: map[string]any{
				"address": map[string]any{
					"city": "Boston",
				},
			},
		},
		{
			name:  "deeply nested",
			path:  "a.b.c",
			value: 123,
			expected: map[string]any{
				"a": map[string]any{
					"b": map[string]any{
						"c": 123,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := make(map[string]any)
			err := setNestedValue(m, tt.path, tt.value)
			require.NoError(t, err)

			assertDeepEqual(t, tt.expected, m)
		})
	}
}

// ============================================================================
// Acceptance Criteria Test: POST /trade/sales-orders
// ============================================================================

func TestAcceptanceCriteria_SalesOrderRequest(t *testing.T) {
	t.Run("正确构建 POST /trade/sales-orders 请求", func(t *testing.T) {
		// Setup pool with realistic sales order data
		paramPool := pool.NewShardedPool(nil)
		defer paramPool.Close()

		// Customer data
		paramPool.Add(circuit.EntityCustomerID, "CUST-2024-001", pool.ValueSource{})

		// Product and item data
		paramPool.Add(circuit.EntityProductID, "PROD-SKU-100", pool.ValueSource{})
		paramPool.Add(circuit.OrderItemQuantity, 10, pool.ValueSource{})
		paramPool.Add(circuit.CommonPrice, 199.99, pool.ValueSource{})

		// Order metadata
		paramPool.Add(circuit.CommonNote, "Priority shipping requested", pool.ValueSource{})

		// Build the sales order request
		rb := NewRequestBuilder(paramPool)
		req, err := rb.BuildSalesOrderFromPool()

		// Verify request construction
		require.NoError(t, err, "should build request without error")
		require.NotNil(t, req, "request should not be nil")

		// Verify HTTP method and path
		assert.Equal(t, "POST", req.Method, "should use POST method")
		assert.Equal(t, "/trade/sales-orders", req.URL.Path, "should target /trade/sales-orders")

		// Verify content type
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"), "should set Content-Type to application/json")

		// Parse and verify body structure
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err, "should read body")

		var salesOrder map[string]any
		err = json.Unmarshal(body, &salesOrder)
		require.NoError(t, err, "body should be valid JSON")

		// Verify customer_id
		assert.Equal(t, "CUST-2024-001", salesOrder["customer_id"], "should include customer_id")

		// Verify items array
		items, ok := salesOrder["items"].([]any)
		require.True(t, ok, "items should be an array")
		require.GreaterOrEqual(t, len(items), 1, "should have at least one item")

		item := items[0].(map[string]any)
		assert.Equal(t, "PROD-SKU-100", item["product_id"], "item should have product_id")
		assert.Equal(t, float64(10), item["quantity"], "item should have quantity")

		// Verify notes (optional)
		if notes, ok := salesOrder["notes"]; ok {
			assert.Equal(t, "Priority shipping requested", notes, "should include notes when provided")
		}

		t.Logf("Successfully built POST /trade/sales-orders request")
		t.Logf("Request body: %s", string(body))
	})

	t.Run("使用 ExtendedEndpointUnit 构建复杂销售订单", func(t *testing.T) {
		paramPool := pool.NewShardedPool(nil)
		defer paramPool.Close()

		// Setup parameters
		paramPool.Add(circuit.EntityCustomerID, "CUST-COMPLEX-001", pool.ValueSource{})
		paramPool.Add(circuit.EntityProductID, "PROD-A1", pool.ValueSource{})
		paramPool.Add(circuit.OrderItemQuantity, 5, pool.ValueSource{})
		paramPool.Add(circuit.OrderItemPrice, 49.99, pool.ValueSource{})
		paramPool.Add(circuit.SemanticType("common.city"), "Shanghai", pool.ValueSource{})
		paramPool.Add(circuit.SemanticType("common.postal_code"), "200000", pool.ValueSource{})

		// Define endpoint with explicit structure
		unit := &ExtendedEndpointUnit{
			Name:   "create-sales-order",
			Path:   "/trade/sales-orders",
			Method: "POST",
			InputPins: []InputPin{
				{
					Name:         "customer_id",
					Location:     paramLocationBody,
					SemanticType: circuit.EntityCustomerID,
					Required:     true,
				},
				{
					Name:         "product_id",
					Location:     paramLocationBody,
					SemanticType: circuit.EntityProductID,
					JSONPath:     "items[0].product_id",
					Required:     true,
				},
				{
					Name:         "quantity",
					Location:     paramLocationBody,
					SemanticType: circuit.OrderItemQuantity,
					JSONPath:     "items[0].quantity",
					Required:     true,
				},
				{
					Name:         "unit_price",
					Location:     paramLocationBody,
					SemanticType: circuit.OrderItemPrice,
					JSONPath:     "items[0].unit_price",
					Required:     false,
				},
				{
					Name:         "city",
					Location:     paramLocationBody,
					SemanticType: circuit.SemanticType("common.city"),
					JSONPath:     "shipping_address.city",
					Required:     false,
				},
				{
					Name:         "postal_code",
					Location:     paramLocationBody,
					SemanticType: circuit.SemanticType("common.postal_code"),
					JSONPath:     "shipping_address.postal_code",
					Required:     false,
				},
			},
		}

		rb := NewRequestBuilder(paramPool)
		req, err := rb.BuildRequestFromExtended(unit)
		require.NoError(t, err)
		require.NotNil(t, req)

		// Verify request
		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "/trade/sales-orders", req.URL.Path)

		body, _ := io.ReadAll(req.Body)
		var salesOrder map[string]any
		json.Unmarshal(body, &salesOrder)

		assert.Equal(t, "CUST-COMPLEX-001", salesOrder["customer_id"])

		// Verify nested structure
		if shippingAddr, ok := salesOrder["shipping_address"].(map[string]any); ok {
			assert.Equal(t, "Shanghai", shippingAddr["city"])
			assert.Equal(t, "200000", shippingAddr["postal_code"])
		}

		t.Logf("Complex sales order body: %s", string(body))
	})
}

// ============================================================================
// Additional Coverage Tests
// ============================================================================

func TestMatchesPathParam(t *testing.T) {
	tests := []struct {
		name      string
		paramName string
		pin       InputPin
		expected  bool
	}{
		{
			name:      "direct match",
			paramName: "id",
			pin:       InputPin{Name: "id", SemanticType: circuit.CommonID},
			expected:  true,
		},
		{
			name:      "snake_case match",
			paramName: "customerId",
			pin:       InputPin{Name: "customer_id", SemanticType: circuit.EntityCustomerID},
			expected:  true,
		},
		{
			name:      "entity id suffix match",
			paramName: "warehouse_id",
			pin:       InputPin{Name: "warehouse_id", SemanticType: circuit.EntityWarehouseID},
			expected:  true,
		},
		{
			name:      "semantic type entity match",
			paramName: "product_id",
			pin:       InputPin{Name: "id", SemanticType: circuit.EntityProductID},
			expected:  true,
		},
		{
			name:      "no match",
			paramName: "user_id",
			pin:       InputPin{Name: "order_id", SemanticType: circuit.OrderSalesID},
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesPathParam(tt.paramName, tt.pin)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestApplySchemaDefaults(t *testing.T) {
	t.Run("apply object schema defaults", func(t *testing.T) {
		body := make(map[string]any)
		schema := &SchemaInfo{
			Type: "object",
			Properties: map[string]*SchemaInfo{
				"name": {Type: "string"},
				"details": {
					Type: "object",
					Properties: map[string]*SchemaInfo{
						"description": {Type: "string"},
					},
				},
				"tags": {
					Type:  "array",
					Items: &SchemaInfo{Type: "string"},
				},
			},
		}

		result := applySchemaDefaults(body, schema)

		// Object properties should be initialized
		assert.NotNil(t, result["details"])
		assert.IsType(t, map[string]any{}, result["details"])

		// Array properties should be initialized
		assert.NotNil(t, result["tags"])
		assert.IsType(t, []any{}, result["tags"])
	})

	t.Run("nil schema returns body unchanged", func(t *testing.T) {
		body := map[string]any{"key": "value"}
		result := applySchemaDefaults(body, nil)
		assert.Equal(t, body, result)
	})

	t.Run("existing values not overwritten", func(t *testing.T) {
		body := map[string]any{
			"name": "existing",
			"details": map[string]any{
				"custom": "data",
			},
		}
		schema := &SchemaInfo{
			Type: "object",
			Properties: map[string]*SchemaInfo{
				"name": {Type: "string"},
				"details": {
					Type: "object",
					Properties: map[string]*SchemaInfo{
						"description": {Type: "string"},
					},
				},
			},
		}

		result := applySchemaDefaults(body, schema)
		assert.Equal(t, "existing", result["name"])

		details := result["details"].(map[string]any)
		assert.Equal(t, "data", details["custom"])
	})
}

func TestBuildQueryFromPins_ArrayValues(t *testing.T) {
	paramPool := pool.NewShardedPool(nil)
	defer paramPool.Close()

	// Add array value
	paramPool.Add(circuit.SemanticType("common.ids"), []string{"id1", "id2", "id3"}, pool.ValueSource{})

	unit := &ExtendedEndpointUnit{
		Name:   "bulk-get",
		Path:   "/items",
		Method: "GET",
		InputPins: []InputPin{
			{
				Name:         "ids",
				Location:     paramLocationQuery,
				SemanticType: circuit.SemanticType("common.ids"),
			},
		},
	}

	rb := NewRequestBuilder(paramPool)
	req, err := rb.BuildRequestFromExtended(unit)
	require.NoError(t, err)

	query := req.URL.Query()
	ids := query["ids"]
	assert.Len(t, ids, 3)
	assert.Contains(t, ids, "id1")
	assert.Contains(t, ids, "id2")
	assert.Contains(t, ids, "id3")
}

func TestValidateEndpointUnit_EdgeCases(t *testing.T) {
	paramPool := pool.NewShardedPool(nil)
	defer paramPool.Close()

	rb := NewRequestBuilder(paramPool)

	t.Run("empty method", func(t *testing.T) {
		unit := &circuit.EndpointUnit{
			Name:   "test",
			Path:   "/test",
			Method: "",
		}
		_, err := rb.BuildRequest(unit)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "HTTP method is empty")
	})

	t.Run("invalid method gets error", func(t *testing.T) {
		unit := &circuit.EndpointUnit{
			Name:   "test",
			Path:   "/test",
			Method: "INVALID",
		}
		_, err := rb.BuildRequest(unit)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid HTTP method")
	})
}

func TestGetValueForPin_FallbackToDefault(t *testing.T) {
	paramPool := pool.NewShardedPool(nil)
	defer paramPool.Close()

	// Don't add any values to pool

	unit := &ExtendedEndpointUnit{
		Name:   "test",
		Path:   "/test",
		Method: "POST",
		InputPins: []InputPin{
			{
				Name:     "optional_field",
				Location: paramLocationBody,
				Default:  "default_value",
				Required: false,
			},
		},
	}

	rb := NewRequestBuilder(paramPool)
	req, err := rb.BuildRequestFromExtended(unit)
	require.NoError(t, err)

	body, _ := io.ReadAll(req.Body)
	var bodyMap map[string]any
	json.Unmarshal(body, &bodyMap)

	assert.Equal(t, "default_value", bodyMap["optional_field"])
}
