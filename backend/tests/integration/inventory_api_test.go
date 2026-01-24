// Package integration provides integration testing for the ERP backend API.
// This file contains tests for the Inventory API endpoints against a real database.
package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	inventoryapp "github.com/erp/backend/internal/application/inventory"
	"github.com/erp/backend/internal/infrastructure/persistence"
	"github.com/erp/backend/internal/interfaces/http/handler"
	"github.com/erp/backend/internal/interfaces/http/router"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// InventoryTestServer wraps the test database and HTTP server for Inventory API testing
type InventoryTestServer struct {
	DB     *TestDB
	Engine *gin.Engine
	Router *router.Router
}

// NewInventoryTestServer creates a new test server with Inventory APIs registered
func NewInventoryTestServer(t *testing.T) *InventoryTestServer {
	t.Helper()

	gin.SetMode(gin.TestMode)
	testDB := NewTestDB(t)

	// Initialize repositories
	inventoryItemRepo := persistence.NewGormInventoryItemRepository(testDB.DB)
	stockBatchRepo := persistence.NewGormStockBatchRepository(testDB.DB)
	stockLockRepo := persistence.NewGormStockLockRepository(testDB.DB)
	inventoryTxRepo := persistence.NewGormInventoryTransactionRepository(testDB.DB)

	// Initialize service
	inventoryService := inventoryapp.NewInventoryService(
		inventoryItemRepo,
		stockBatchRepo,
		stockLockRepo,
		inventoryTxRepo,
	)

	// Initialize handler
	inventoryHandler := handler.NewInventoryHandler(inventoryService)

	// Setup engine
	engine := gin.New()

	// Setup routes
	r := router.NewRouter(engine, router.WithAPIVersion("v1"))

	// Register inventory routes matching main.go setup
	inventoryRoutes := router.NewDomainGroup("inventory", "/inventory")

	// Query routes
	inventoryRoutes.GET("/items", inventoryHandler.List)
	inventoryRoutes.GET("/items/lookup", inventoryHandler.GetByWarehouseAndProduct)
	inventoryRoutes.GET("/items/alerts/low-stock", inventoryHandler.ListBelowMinimum)
	inventoryRoutes.GET("/items/:id", inventoryHandler.GetByID)
	inventoryRoutes.GET("/items/:id/transactions", inventoryHandler.ListTransactionsByItem)
	inventoryRoutes.GET("/warehouses/:warehouse_id/items", inventoryHandler.ListByWarehouse)
	inventoryRoutes.GET("/products/:product_id/items", inventoryHandler.ListByProduct)

	// Stock operations
	inventoryRoutes.POST("/availability/check", inventoryHandler.CheckAvailability)
	inventoryRoutes.POST("/stock/increase", inventoryHandler.IncreaseStock)
	inventoryRoutes.POST("/stock/lock", inventoryHandler.LockStock)
	inventoryRoutes.POST("/stock/unlock", inventoryHandler.UnlockStock)
	inventoryRoutes.POST("/stock/deduct", inventoryHandler.DeductStock)
	inventoryRoutes.POST("/stock/adjust", inventoryHandler.AdjustStock)
	inventoryRoutes.PUT("/thresholds", inventoryHandler.SetThresholds)

	// Lock management
	inventoryRoutes.GET("/locks", inventoryHandler.GetActiveLocks)
	inventoryRoutes.GET("/locks/:id", inventoryHandler.GetLockByID)

	// Transactions
	inventoryRoutes.GET("/transactions", inventoryHandler.ListTransactions)
	inventoryRoutes.GET("/transactions/:id", inventoryHandler.GetTransactionByID)

	r.Register(inventoryRoutes)
	r.Setup()

	return &InventoryTestServer{
		DB:     testDB,
		Engine: engine,
		Router: r,
	}
}

// Request makes an HTTP request to the test server
func (ts *InventoryTestServer) Request(method, path string, body any, tenantID ...uuid.UUID) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBody)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")

	// Set tenant ID header if provided
	if len(tenantID) > 0 {
		req.Header.Set("X-Tenant-ID", tenantID[0].String())
	}

	w := httptest.NewRecorder()
	ts.Engine.ServeHTTP(w, req)
	return w
}

// Helper function to assert decimal values (returned as strings in JSON)
func assertDecimalEqual(t *testing.T, expected float64, actual any, msg ...string) {
	t.Helper()
	msgStr := ""
	if len(msg) > 0 {
		msgStr = msg[0]
	}
	// The API returns decimal values as strings
	switch v := actual.(type) {
	case string:
		assert.Equal(t, fmt.Sprintf("%.0f", expected), v, msgStr)
	case float64:
		assert.Equal(t, expected, v, msgStr)
	default:
		t.Errorf("unexpected type for decimal: %T, value: %v, %s", actual, actual, msgStr)
	}
}

// =====================================================================
// INVENTORY QUERY API TESTS
// =====================================================================

// TestInventoryAPI_IncreaseAndGetByID tests stock increase and retrieval
func TestInventoryAPI_IncreaseAndGetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewInventoryTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create warehouse and product
	warehouseID := uuid.New()
	productID := uuid.New()
	ts.DB.CreateTestWarehouse(tenantID, warehouseID)
	ts.DB.CreateTestProduct(tenantID, productID)

	t.Run("Increase stock creates inventory item", func(t *testing.T) {
		// Increase stock
		increaseReq := map[string]any{
			"warehouse_id": warehouseID.String(),
			"product_id":   productID.String(),
			"quantity":     100.0,
			"unit_cost":    15.50,
			"source_type":  "PURCHASE_ORDER",
			"source_id":    "PO-2024-001",
			"reference":    "Initial stock",
			"reason":       "Purchase receiving",
		}

		w := ts.Request("POST", "/api/v1/inventory/stock/increase", increaseReq, tenantID)
		require.Equal(t, http.StatusOK, w.Code, "Response body: %s", w.Body.String())

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]any)
		assert.NotEmpty(t, data["id"])
		assert.Equal(t, warehouseID.String(), data["warehouse_id"])
		assert.Equal(t, productID.String(), data["product_id"])
		// Decimal values are returned as strings
		assert.Equal(t, "100", data["available_quantity"])
		assert.Equal(t, "0", data["locked_quantity"])
		assert.Equal(t, "15.5", data["unit_cost"])

		// Get by ID
		itemID := data["id"].(string)
		w2 := ts.Request("GET", "/api/v1/inventory/items/"+itemID, nil, tenantID)
		require.Equal(t, http.StatusOK, w2.Code)

		var getResponse map[string]any
		err = json.Unmarshal(w2.Body.Bytes(), &getResponse)
		require.NoError(t, err)
		assert.True(t, getResponse["success"].(bool))

		getData := getResponse["data"].(map[string]any)
		assert.Equal(t, itemID, getData["id"])
		assert.Equal(t, "100", getData["available_quantity"])
	})

	t.Run("Get by warehouse and product", func(t *testing.T) {
		// Query by warehouse and product
		path := fmt.Sprintf("/api/v1/inventory/items/lookup?warehouse_id=%s&product_id=%s", warehouseID.String(), productID.String())
		w := ts.Request("GET", path, nil, tenantID)
		require.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response["success"].(bool))

		data := response["data"].(map[string]any)
		assert.Equal(t, warehouseID.String(), data["warehouse_id"])
		assert.Equal(t, productID.String(), data["product_id"])
	})

	t.Run("Get non-existent item returns 404", func(t *testing.T) {
		fakeID := uuid.New().String()
		w := ts.Request("GET", "/api/v1/inventory/items/"+fakeID, nil, tenantID)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Invalid UUID returns 400", func(t *testing.T) {
		w := ts.Request("GET", "/api/v1/inventory/items/invalid-uuid", nil, tenantID)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestInventoryAPI_List tests listing inventory items with filtering
func TestInventoryAPI_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewInventoryTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create warehouse and multiple products
	warehouseID := uuid.New()
	ts.DB.CreateTestWarehouse(tenantID, warehouseID)

	productIDs := make([]uuid.UUID, 5)
	for i := range 5 {
		productIDs[i] = uuid.New()
		ts.DB.CreateTestProduct(tenantID, productIDs[i])

		// Add stock for each product
		increaseReq := map[string]any{
			"warehouse_id": warehouseID.String(),
			"product_id":   productIDs[i].String(),
			"quantity":     float64((i + 1) * 10),
			"unit_cost":    float64((i + 1) * 5),
			"source_type":  "PURCHASE_ORDER",
			"source_id":    fmt.Sprintf("PO-2024-%03d", i+1),
		}
		w := ts.Request("POST", "/api/v1/inventory/stock/increase", increaseReq, tenantID)
		require.Equal(t, http.StatusOK, w.Code, "Failed to create inventory item %d: %s", i, w.Body.String())
	}

	t.Run("List all inventory items", func(t *testing.T) {
		w := ts.Request("GET", "/api/v1/inventory/items?page=1&page_size=10", nil, tenantID)
		require.Equal(t, http.StatusOK, w.Code, "Response: %s", w.Body.String())

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response["success"].(bool))

		data := response["data"].([]any)
		assert.Len(t, data, 5)

		meta := response["meta"].(map[string]any)
		assert.Equal(t, float64(5), meta["total"])
	})

	t.Run("List by warehouse", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/inventory/warehouses/%s/items?page=1&page_size=10", warehouseID.String())
		w := ts.Request("GET", path, nil, tenantID)
		require.Equal(t, http.StatusOK, w.Code, "Response: %s", w.Body.String())

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response["success"].(bool))

		data := response["data"].([]any)
		assert.Len(t, data, 5)
	})

	t.Run("List by product", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/inventory/products/%s/items?page=1&page_size=10", productIDs[0].String())
		w := ts.Request("GET", path, nil, tenantID)
		require.Equal(t, http.StatusOK, w.Code, "Response: %s", w.Body.String())

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response["success"].(bool))

		data := response["data"].([]any)
		assert.Len(t, data, 1)
	})

	t.Run("Pagination works correctly", func(t *testing.T) {
		w := ts.Request("GET", "/api/v1/inventory/items?page=1&page_size=2", nil, tenantID)
		require.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].([]any)
		assert.Len(t, data, 2)

		meta := response["meta"].(map[string]any)
		assert.Equal(t, float64(5), meta["total"])
		assert.Equal(t, float64(1), meta["page"])
		assert.Equal(t, float64(2), meta["page_size"])
	})
}

// =====================================================================
// STOCK OPERATION API TESTS
// =====================================================================

// TestInventoryAPI_StockLockUnlock tests stock locking and unlocking
func TestInventoryAPI_StockLockUnlock(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewInventoryTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create warehouse and product with initial stock
	warehouseID := uuid.New()
	productID := uuid.New()
	ts.DB.CreateTestWarehouse(tenantID, warehouseID)
	ts.DB.CreateTestProduct(tenantID, productID)

	// Add initial stock
	increaseReq := map[string]any{
		"warehouse_id": warehouseID.String(),
		"product_id":   productID.String(),
		"quantity":     100.0,
		"unit_cost":    10.0,
		"source_type":  "PURCHASE_ORDER",
		"source_id":    "PO-2024-001",
	}
	w := ts.Request("POST", "/api/v1/inventory/stock/increase", increaseReq, tenantID)
	require.Equal(t, http.StatusOK, w.Code, "Response: %s", w.Body.String())

	var lockID string

	t.Run("Lock stock for pending order", func(t *testing.T) {
		expireAt := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
		lockReq := map[string]any{
			"warehouse_id": warehouseID.String(),
			"product_id":   productID.String(),
			"quantity":     30.0,
			"source_type":  "sales_order",
			"source_id":    "SO-2024-001",
			"expire_at":    expireAt,
		}

		w := ts.Request("POST", "/api/v1/inventory/stock/lock", lockReq, tenantID)
		require.Equal(t, http.StatusOK, w.Code, "Response: %s", w.Body.String())

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response["success"].(bool))

		data := response["data"].(map[string]any)
		lockID = data["lock_id"].(string)
		assert.NotEmpty(t, lockID)
		assert.Equal(t, "30", data["quantity"])
		assert.Equal(t, "sales_order", data["source_type"])

		// Verify available quantity decreased
		path := fmt.Sprintf("/api/v1/inventory/items/lookup?warehouse_id=%s&product_id=%s", warehouseID.String(), productID.String())
		w2 := ts.Request("GET", path, nil, tenantID)
		require.Equal(t, http.StatusOK, w2.Code)

		var getResponse map[string]any
		err = json.Unmarshal(w2.Body.Bytes(), &getResponse)
		require.NoError(t, err)
		getData := getResponse["data"].(map[string]any)
		assert.Equal(t, "70", getData["available_quantity"])
		assert.Equal(t, "30", getData["locked_quantity"])
	})

	t.Run("Get active locks", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/inventory/locks?warehouse_id=%s&product_id=%s", warehouseID.String(), productID.String())
		w := ts.Request("GET", path, nil, tenantID)
		require.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].([]any)
		assert.Len(t, data, 1)
		lock := data[0].(map[string]any)
		assert.Equal(t, lockID, lock["id"])
		assert.Equal(t, "30", lock["quantity"])
	})

	t.Run("Get lock by ID", func(t *testing.T) {
		path := "/api/v1/inventory/locks/" + lockID
		w := ts.Request("GET", path, nil, tenantID)
		require.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].(map[string]any)
		assert.Equal(t, lockID, data["id"])
		assert.Equal(t, "30", data["quantity"])
	})

	t.Run("Unlock stock (order cancelled)", func(t *testing.T) {
		unlockReq := map[string]any{
			"lock_id": lockID,
		}

		w := ts.Request("POST", "/api/v1/inventory/stock/unlock", unlockReq, tenantID)
		require.Equal(t, http.StatusNoContent, w.Code)

		// Verify available quantity restored
		path := fmt.Sprintf("/api/v1/inventory/items/lookup?warehouse_id=%s&product_id=%s", warehouseID.String(), productID.String())
		w2 := ts.Request("GET", path, nil, tenantID)
		require.Equal(t, http.StatusOK, w2.Code)

		var getResponse map[string]any
		err := json.Unmarshal(w2.Body.Bytes(), &getResponse)
		require.NoError(t, err)
		getData := getResponse["data"].(map[string]any)
		assert.Equal(t, "100", getData["available_quantity"])
		assert.Equal(t, "0", getData["locked_quantity"])
	})
}

// TestInventoryAPI_StockDeduct tests stock deduction (fulfillment)
func TestInventoryAPI_StockDeduct(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewInventoryTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create warehouse and product with initial stock
	warehouseID := uuid.New()
	productID := uuid.New()
	ts.DB.CreateTestWarehouse(tenantID, warehouseID)
	ts.DB.CreateTestProduct(tenantID, productID)

	// Add initial stock
	increaseReq := map[string]any{
		"warehouse_id": warehouseID.String(),
		"product_id":   productID.String(),
		"quantity":     100.0,
		"unit_cost":    10.0,
		"source_type":  "PURCHASE_ORDER",
		"source_id":    "PO-2024-001",
	}
	w := ts.Request("POST", "/api/v1/inventory/stock/increase", increaseReq, tenantID)
	require.Equal(t, http.StatusOK, w.Code, "Response: %s", w.Body.String())

	// Lock stock first
	expireAt := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	lockReq := map[string]any{
		"warehouse_id": warehouseID.String(),
		"product_id":   productID.String(),
		"quantity":     20.0,
		"source_type":  "sales_order",
		"source_id":    "SO-2024-001",
		"expire_at":    expireAt,
	}
	w = ts.Request("POST", "/api/v1/inventory/stock/lock", lockReq, tenantID)
	require.Equal(t, http.StatusOK, w.Code, "Response: %s", w.Body.String())

	var lockResponse map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &lockResponse)
	require.NoError(t, err)
	lockID := lockResponse["data"].(map[string]any)["lock_id"].(string)

	t.Run("Deduct locked stock (shipment)", func(t *testing.T) {
		deductReq := map[string]any{
			"lock_id":     lockID,
			"source_type": "SALES_ORDER",
			"source_id":   "SO-2024-001",
			"reference":   "Shipped via SF Express",
		}

		w := ts.Request("POST", "/api/v1/inventory/stock/deduct", deductReq, tenantID)
		require.Equal(t, http.StatusNoContent, w.Code, "Response: %s", w.Body.String())

		// Verify total quantity decreased, locked quantity back to 0
		path := fmt.Sprintf("/api/v1/inventory/items/lookup?warehouse_id=%s&product_id=%s", warehouseID.String(), productID.String())
		w2 := ts.Request("GET", path, nil, tenantID)
		require.Equal(t, http.StatusOK, w2.Code)

		var getResponse map[string]any
		err := json.Unmarshal(w2.Body.Bytes(), &getResponse)
		require.NoError(t, err)
		getData := getResponse["data"].(map[string]any)
		assert.Equal(t, "80", getData["available_quantity"])
		assert.Equal(t, "0", getData["locked_quantity"])
		assert.Equal(t, "80", getData["total_quantity"])
	})
}

// TestInventoryAPI_StockAdjust tests stock adjustment (stock counting)
func TestInventoryAPI_StockAdjust(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewInventoryTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create warehouse and product with initial stock
	warehouseID := uuid.New()
	productID := uuid.New()
	ts.DB.CreateTestWarehouse(tenantID, warehouseID)
	ts.DB.CreateTestProduct(tenantID, productID)

	// Add initial stock
	increaseReq := map[string]any{
		"warehouse_id": warehouseID.String(),
		"product_id":   productID.String(),
		"quantity":     100.0,
		"unit_cost":    10.0,
		"source_type":  "PURCHASE_ORDER",
		"source_id":    "PO-2024-001",
	}
	w := ts.Request("POST", "/api/v1/inventory/stock/increase", increaseReq, tenantID)
	require.Equal(t, http.StatusOK, w.Code, "Response: %s", w.Body.String())

	t.Run("Adjust stock down (damaged goods)", func(t *testing.T) {
		adjustReq := map[string]any{
			"warehouse_id":    warehouseID.String(),
			"product_id":      productID.String(),
			"actual_quantity": 95.0,
			"reason":          "5 units damaged during inspection",
			"source_type":     "STOCK_TAKE",
			"source_id":       "ST-2024-001",
		}

		w := ts.Request("POST", "/api/v1/inventory/stock/adjust", adjustReq, tenantID)
		require.Equal(t, http.StatusOK, w.Code, "Response: %s", w.Body.String())

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response["success"].(bool))

		data := response["data"].(map[string]any)
		assert.Equal(t, "95", data["available_quantity"])
	})

	t.Run("Adjust stock up (found extra items)", func(t *testing.T) {
		adjustReq := map[string]any{
			"warehouse_id":    warehouseID.String(),
			"product_id":      productID.String(),
			"actual_quantity": 100.0,
			"reason":          "Found 5 additional units",
			"source_type":     "STOCK_TAKE",
			"source_id":       "ST-2024-002",
		}

		w := ts.Request("POST", "/api/v1/inventory/stock/adjust", adjustReq, tenantID)
		require.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].(map[string]any)
		assert.Equal(t, "100", data["available_quantity"])
	})
}

// TestInventoryAPI_CheckAvailability tests stock availability checking
func TestInventoryAPI_CheckAvailability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewInventoryTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create warehouse and product with initial stock
	warehouseID := uuid.New()
	productID := uuid.New()
	ts.DB.CreateTestWarehouse(tenantID, warehouseID)
	ts.DB.CreateTestProduct(tenantID, productID)

	// Add initial stock
	increaseReq := map[string]any{
		"warehouse_id": warehouseID.String(),
		"product_id":   productID.String(),
		"quantity":     50.0,
		"unit_cost":    10.0,
		"source_type":  "PURCHASE_ORDER",
		"source_id":    "PO-2024-001",
	}
	w := ts.Request("POST", "/api/v1/inventory/stock/increase", increaseReq, tenantID)
	require.Equal(t, http.StatusOK, w.Code, "Response: %s", w.Body.String())

	t.Run("Check availability - sufficient stock", func(t *testing.T) {
		checkReq := map[string]any{
			"warehouse_id": warehouseID.String(),
			"product_id":   productID.String(),
			"quantity":     30.0,
		}

		w := ts.Request("POST", "/api/v1/inventory/availability/check", checkReq, tenantID)
		require.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].(map[string]any)
		assert.True(t, data["available"].(bool))
		assert.Equal(t, float64(50), data["available_quantity"])
		assert.Equal(t, float64(30), data["requested_quantity"])
	})

	t.Run("Check availability - insufficient stock", func(t *testing.T) {
		checkReq := map[string]any{
			"warehouse_id": warehouseID.String(),
			"product_id":   productID.String(),
			"quantity":     100.0,
		}

		w := ts.Request("POST", "/api/v1/inventory/availability/check", checkReq, tenantID)
		require.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].(map[string]any)
		assert.False(t, data["available"].(bool))
		assert.Equal(t, float64(50), data["available_quantity"])
		assert.Equal(t, float64(100), data["requested_quantity"])
	})
}

// TestInventoryAPI_SetThresholds tests setting min/max thresholds
func TestInventoryAPI_SetThresholds(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewInventoryTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create warehouse and product with initial stock
	warehouseID := uuid.New()
	productID := uuid.New()
	ts.DB.CreateTestWarehouse(tenantID, warehouseID)
	ts.DB.CreateTestProduct(tenantID, productID)

	// Add initial stock
	increaseReq := map[string]any{
		"warehouse_id": warehouseID.String(),
		"product_id":   productID.String(),
		"quantity":     50.0,
		"unit_cost":    10.0,
		"source_type":  "PURCHASE_ORDER",
		"source_id":    "PO-2024-001",
	}
	w := ts.Request("POST", "/api/v1/inventory/stock/increase", increaseReq, tenantID)
	require.Equal(t, http.StatusOK, w.Code, "Response: %s", w.Body.String())

	t.Run("Set thresholds", func(t *testing.T) {
		thresholdReq := map[string]any{
			"warehouse_id": warehouseID.String(),
			"product_id":   productID.String(),
			"min_quantity": 20.0,
			"max_quantity": 200.0,
		}

		w := ts.Request("PUT", "/api/v1/inventory/thresholds", thresholdReq, tenantID)
		require.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response["success"].(bool))

		data := response["data"].(map[string]any)
		assert.Equal(t, "20", data["min_quantity"])
		assert.Equal(t, "200", data["max_quantity"])
		assert.False(t, data["is_below_minimum"].(bool)) // 50 > 20
		assert.False(t, data["is_above_maximum"].(bool)) // 50 < 200
	})

	t.Run("Low stock alert shows in list", func(t *testing.T) {
		// Reduce stock below minimum
		adjustReq := map[string]any{
			"warehouse_id":    warehouseID.String(),
			"product_id":      productID.String(),
			"actual_quantity": 15.0,
			"reason":          "Test low stock alert",
			"source_type":     "MANUAL_ADJUSTMENT",
			"source_id":       "ADJ-001",
		}
		w := ts.Request("POST", "/api/v1/inventory/stock/adjust", adjustReq, tenantID)
		require.Equal(t, http.StatusOK, w.Code)

		// Check low stock alert list
		w2 := ts.Request("GET", "/api/v1/inventory/items/alerts/low-stock?page=1&page_size=10", nil, tenantID)
		require.Equal(t, http.StatusOK, w2.Code)

		var response map[string]any
		err := json.Unmarshal(w2.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].([]any)
		assert.Len(t, data, 1) // Our item should be in the low stock list

		item := data[0].(map[string]any)
		assert.Equal(t, "15", item["available_quantity"])
		assert.True(t, item["is_below_minimum"].(bool))
	})
}

// TestInventoryAPI_Transactions tests inventory transaction history
func TestInventoryAPI_Transactions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewInventoryTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create warehouse and product
	warehouseID := uuid.New()
	productID := uuid.New()
	ts.DB.CreateTestWarehouse(tenantID, warehouseID)
	ts.DB.CreateTestProduct(tenantID, productID)

	// Perform multiple stock operations to create transactions
	// 1. Increase stock
	increaseReq := map[string]any{
		"warehouse_id": warehouseID.String(),
		"product_id":   productID.String(),
		"quantity":     100.0,
		"unit_cost":    10.0,
		"source_type":  "PURCHASE_ORDER",
		"source_id":    "PO-2024-001",
	}
	w := ts.Request("POST", "/api/v1/inventory/stock/increase", increaseReq, tenantID)
	require.Equal(t, http.StatusOK, w.Code, "Response: %s", w.Body.String())

	// Get item ID for transaction query
	var increaseResponse map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &increaseResponse)
	require.NoError(t, err)
	itemID := increaseResponse["data"].(map[string]any)["id"].(string)

	// 2. Adjust stock
	adjustReq := map[string]any{
		"warehouse_id":    warehouseID.String(),
		"product_id":      productID.String(),
		"actual_quantity": 95.0,
		"reason":          "Damaged goods",
		"source_type":     "MANUAL_ADJUSTMENT",
		"source_id":       "ADJ-001",
	}
	w = ts.Request("POST", "/api/v1/inventory/stock/adjust", adjustReq, tenantID)
	require.Equal(t, http.StatusOK, w.Code)

	t.Run("List all transactions", func(t *testing.T) {
		w := ts.Request("GET", "/api/v1/inventory/transactions?page=1&page_size=10", nil, tenantID)
		require.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].([]any)
		assert.GreaterOrEqual(t, len(data), 2) // At least increase + adjustment
	})

	t.Run("List transactions by item", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/inventory/items/%s/transactions?page=1&page_size=10", itemID)
		w := ts.Request("GET", path, nil, tenantID)
		require.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].([]any)
		assert.GreaterOrEqual(t, len(data), 2) // At least increase + adjustment
	})

	t.Run("Filter transactions by type", func(t *testing.T) {
		path := "/api/v1/inventory/transactions?transaction_type=INBOUND&page=1&page_size=10"
		w := ts.Request("GET", path, nil, tenantID)
		require.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].([]any)
		// Verify all returned transactions are INBOUND
		for _, tx := range data {
			txMap := tx.(map[string]any)
			assert.Equal(t, "INBOUND", txMap["transaction_type"])
		}
	})
}

// =====================================================================
// ERROR SCENARIO TESTS
// =====================================================================

// TestInventoryAPI_ValidationErrors tests input validation
func TestInventoryAPI_ValidationErrors(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewInventoryTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	t.Run("Increase stock - missing required fields", func(t *testing.T) {
		increaseReq := map[string]any{
			"warehouse_id": uuid.New().String(),
			// Missing product_id, quantity, etc.
		}

		w := ts.Request("POST", "/api/v1/inventory/stock/increase", increaseReq, tenantID)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Increase stock - negative quantity", func(t *testing.T) {
		increaseReq := map[string]any{
			"warehouse_id": uuid.New().String(),
			"product_id":   uuid.New().String(),
			"quantity":     -10.0,
			"unit_cost":    10.0,
			"source_type":  "PURCHASE_ORDER",
			"source_id":    "PO-2024-001",
		}

		w := ts.Request("POST", "/api/v1/inventory/stock/increase", increaseReq, tenantID)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Increase stock - invalid UUID", func(t *testing.T) {
		increaseReq := map[string]any{
			"warehouse_id": "not-a-valid-uuid",
			"product_id":   uuid.New().String(),
			"quantity":     10.0,
			"unit_cost":    10.0,
			"source_type":  "PURCHASE_ORDER",
			"source_id":    "PO-2024-001",
		}

		w := ts.Request("POST", "/api/v1/inventory/stock/increase", increaseReq, tenantID)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Lock stock - insufficient stock", func(t *testing.T) {
		warehouseID := uuid.New()
		productID := uuid.New()
		ts.DB.CreateTestWarehouse(tenantID, warehouseID)
		ts.DB.CreateTestProduct(tenantID, productID)

		// Add only 10 units
		increaseReq := map[string]any{
			"warehouse_id": warehouseID.String(),
			"product_id":   productID.String(),
			"quantity":     10.0,
			"unit_cost":    10.0,
			"source_type":  "PURCHASE_ORDER",
			"source_id":    "PO-2024-001",
		}
		w := ts.Request("POST", "/api/v1/inventory/stock/increase", increaseReq, tenantID)
		require.Equal(t, http.StatusOK, w.Code, "Response: %s", w.Body.String())

		// Try to lock 100 units
		lockReq := map[string]any{
			"warehouse_id": warehouseID.String(),
			"product_id":   productID.String(),
			"quantity":     100.0,
			"source_type":  "sales_order",
			"source_id":    "SO-2024-001",
		}
		w = ts.Request("POST", "/api/v1/inventory/stock/lock", lockReq, tenantID)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("Unlock non-existent lock", func(t *testing.T) {
		unlockReq := map[string]any{
			"lock_id": uuid.New().String(),
		}

		w := ts.Request("POST", "/api/v1/inventory/stock/unlock", unlockReq, tenantID)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Adjust stock - missing reason", func(t *testing.T) {
		adjustReq := map[string]any{
			"warehouse_id":    uuid.New().String(),
			"product_id":      uuid.New().String(),
			"actual_quantity": 50.0,
			// Missing reason
		}

		w := ts.Request("POST", "/api/v1/inventory/stock/adjust", adjustReq, tenantID)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestInventoryAPI_TenantIsolation tests that inventory data is isolated between tenants
func TestInventoryAPI_TenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewInventoryTestServer(t)

	tenant1 := uuid.New()
	tenant2 := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenant1)
	ts.DB.CreateTestTenantWithUUID(tenant2)

	// Create warehouse and product for tenant 1
	warehouse1 := uuid.New()
	product1 := uuid.New()
	ts.DB.CreateTestWarehouse(tenant1, warehouse1)
	ts.DB.CreateTestProduct(tenant1, product1)

	// Add stock for tenant 1
	increaseReq := map[string]any{
		"warehouse_id": warehouse1.String(),
		"product_id":   product1.String(),
		"quantity":     100.0,
		"unit_cost":    10.0,
		"source_type":  "PURCHASE_ORDER",
		"source_id":    "PO-2024-001",
	}
	w := ts.Request("POST", "/api/v1/inventory/stock/increase", increaseReq, tenant1)
	require.Equal(t, http.StatusOK, w.Code, "Response: %s", w.Body.String())

	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	itemID := response["data"].(map[string]any)["id"].(string)

	t.Run("Tenant 1 can see their inventory", func(t *testing.T) {
		w := ts.Request("GET", "/api/v1/inventory/items?page=1&page_size=10", nil, tenant1)
		require.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].([]any)
		assert.Len(t, data, 1)
	})

	t.Run("Tenant 2 cannot see tenant 1's inventory", func(t *testing.T) {
		w := ts.Request("GET", "/api/v1/inventory/items?page=1&page_size=10", nil, tenant2)
		require.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].([]any)
		assert.Len(t, data, 0)
	})

	t.Run("Tenant 2 cannot access tenant 1's item by ID", func(t *testing.T) {
		w := ts.Request("GET", "/api/v1/inventory/items/"+itemID, nil, tenant2)
		// Should return 404 or 403 (item not found for this tenant)
		assert.True(t, w.Code == http.StatusNotFound || w.Code == http.StatusForbidden)
	})

	t.Run("Tenant 2 cannot lock tenant 1's inventory", func(t *testing.T) {
		lockReq := map[string]any{
			"warehouse_id": warehouse1.String(),
			"product_id":   product1.String(),
			"quantity":     10.0,
			"source_type":  "sales_order",
			"source_id":    "SO-2024-001",
		}
		w := ts.Request("POST", "/api/v1/inventory/stock/lock", lockReq, tenant2)
		// Should fail - inventory doesn't exist for tenant2
		assert.True(t, w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError)
	})
}

// TestInventoryAPI_ConcurrentOperations tests handling of concurrent stock operations
func TestInventoryAPI_ConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewInventoryTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create warehouse and product with initial stock
	warehouseID := uuid.New()
	productID := uuid.New()
	ts.DB.CreateTestWarehouse(tenantID, warehouseID)
	ts.DB.CreateTestProduct(tenantID, productID)

	// Add initial stock
	increaseReq := map[string]any{
		"warehouse_id": warehouseID.String(),
		"product_id":   productID.String(),
		"quantity":     100.0,
		"unit_cost":    10.0,
		"source_type":  "PURCHASE_ORDER",
		"source_id":    "PO-2024-001",
	}
	w := ts.Request("POST", "/api/v1/inventory/stock/increase", increaseReq, tenantID)
	require.Equal(t, http.StatusOK, w.Code, "Response: %s", w.Body.String())

	t.Run("Multiple small locks should succeed", func(t *testing.T) {
		// Lock 20 units 3 times = 60 total
		for i := range 3 {
			lockReq := map[string]any{
				"warehouse_id": warehouseID.String(),
				"product_id":   productID.String(),
				"quantity":     20.0,
				"source_type":  "sales_order",
				"source_id":    fmt.Sprintf("SO-2024-%03d", i+1),
			}
			w := ts.Request("POST", "/api/v1/inventory/stock/lock", lockReq, tenantID)
			require.Equal(t, http.StatusOK, w.Code, "Lock %d failed: %s", i+1, w.Body.String())
		}

		// Verify remaining available quantity
		path := fmt.Sprintf("/api/v1/inventory/items/lookup?warehouse_id=%s&product_id=%s", warehouseID.String(), productID.String())
		w := ts.Request("GET", path, nil, tenantID)
		require.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		data := response["data"].(map[string]any)
		assert.Equal(t, "40", data["available_quantity"])   // 100 - 60
		assert.Equal(t, "60", data["locked_quantity"])      // 3 * 20
		assert.Equal(t, "100", data["total_quantity"])      // unchanged
	})
}
