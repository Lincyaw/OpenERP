// Package integration provides integration testing for the ERP backend API.
// This file contains tests for the Partner API endpoints (Customer, Supplier, Warehouse)
// against a real database.
package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	partnerapp "github.com/erp/backend/internal/application/partner"
	"github.com/erp/backend/internal/infrastructure/persistence"
	"github.com/erp/backend/internal/interfaces/http/handler"
	"github.com/erp/backend/internal/interfaces/http/router"
	"github.com/erp/backend/tests/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PartnerTestServer wraps the test database and HTTP server for Partner API testing
type PartnerTestServer struct {
	DB     *TestDB
	Engine *gin.Engine
	Router *router.Router
}

// NewPartnerTestServer creates a new test server with Partner APIs registered
func NewPartnerTestServer(t *testing.T) *PartnerTestServer {
	t.Helper()

	gin.SetMode(gin.TestMode)
	testDB := NewTestDB(t)

	// Initialize repositories
	customerRepo := persistence.NewGormCustomerRepository(testDB.DB)
	supplierRepo := persistence.NewGormSupplierRepository(testDB.DB)
	warehouseRepo := persistence.NewGormWarehouseRepository(testDB.DB)

	// Initialize services
	customerService := partnerapp.NewCustomerService(customerRepo)
	supplierService := partnerapp.NewSupplierService(supplierRepo)
	warehouseService := partnerapp.NewWarehouseService(warehouseRepo)

	// Initialize handlers
	customerHandler := handler.NewCustomerHandler(customerService)
	supplierHandler := handler.NewSupplierHandler(supplierService)
	warehouseHandler := handler.NewWarehouseHandler(warehouseService)

	// Setup engine with test authentication middleware
	engine := gin.New()
	engine.Use(testutil.TestAuthMiddleware())

	// Setup routes
	r := router.NewRouter(engine, router.WithAPIVersion("v1"))

	// Register customer routes
	customerRoutes := router.NewDomainGroup("partner", "/partner")
	customerRoutes.POST("/customers", customerHandler.Create)
	customerRoutes.GET("/customers", customerHandler.List)
	customerRoutes.GET("/customers/stats/count", customerHandler.CountByStatus)
	customerRoutes.GET("/customers/:id", customerHandler.GetByID)
	customerRoutes.GET("/customers/code/:code", customerHandler.GetByCode)
	customerRoutes.PUT("/customers/:id", customerHandler.Update)
	customerRoutes.PUT("/customers/:id/code", customerHandler.UpdateCode)
	customerRoutes.DELETE("/customers/:id", customerHandler.Delete)
	customerRoutes.POST("/customers/:id/activate", customerHandler.Activate)
	customerRoutes.POST("/customers/:id/deactivate", customerHandler.Deactivate)
	customerRoutes.POST("/customers/:id/suspend", customerHandler.Suspend)
	customerRoutes.POST("/customers/:id/balance/add", customerHandler.AddBalance)
	customerRoutes.POST("/customers/:id/balance/deduct", customerHandler.DeductBalance)
	customerRoutes.PUT("/customers/:id/level", customerHandler.SetLevel)

	// Register supplier routes
	customerRoutes.POST("/suppliers", supplierHandler.Create)
	customerRoutes.GET("/suppliers", supplierHandler.List)
	customerRoutes.GET("/suppliers/stats/count", supplierHandler.CountByStatus)
	customerRoutes.GET("/suppliers/:id", supplierHandler.GetByID)
	customerRoutes.GET("/suppliers/code/:code", supplierHandler.GetByCode)
	customerRoutes.PUT("/suppliers/:id", supplierHandler.Update)
	customerRoutes.PUT("/suppliers/:id/code", supplierHandler.UpdateCode)
	customerRoutes.DELETE("/suppliers/:id", supplierHandler.Delete)
	customerRoutes.POST("/suppliers/:id/activate", supplierHandler.Activate)
	customerRoutes.POST("/suppliers/:id/deactivate", supplierHandler.Deactivate)
	customerRoutes.POST("/suppliers/:id/block", supplierHandler.Block)
	customerRoutes.PUT("/suppliers/:id/rating", supplierHandler.SetRating)
	customerRoutes.PUT("/suppliers/:id/payment-terms", supplierHandler.SetPaymentTerms)

	// Register warehouse routes
	customerRoutes.POST("/warehouses", warehouseHandler.Create)
	customerRoutes.GET("/warehouses", warehouseHandler.List)
	customerRoutes.GET("/warehouses/stats/count", warehouseHandler.CountByStatus)
	customerRoutes.GET("/warehouses/default", warehouseHandler.GetDefault)
	customerRoutes.GET("/warehouses/:id", warehouseHandler.GetByID)
	customerRoutes.GET("/warehouses/code/:code", warehouseHandler.GetByCode)
	customerRoutes.PUT("/warehouses/:id", warehouseHandler.Update)
	customerRoutes.PUT("/warehouses/:id/code", warehouseHandler.UpdateCode)
	customerRoutes.DELETE("/warehouses/:id", warehouseHandler.Delete)
	customerRoutes.POST("/warehouses/:id/enable", warehouseHandler.Enable)
	customerRoutes.POST("/warehouses/:id/disable", warehouseHandler.Disable)
	customerRoutes.POST("/warehouses/:id/set-default", warehouseHandler.SetDefault)

	r.Register(customerRoutes)
	r.Setup()

	return &PartnerTestServer{
		DB:     testDB,
		Engine: engine,
		Router: r,
	}
}

// Request makes an HTTP request to the test server
func (ts *PartnerTestServer) Request(method, path string, body interface{}, tenantID ...uuid.UUID) *httptest.ResponseRecorder {
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

// =====================================================================
// CUSTOMER API TESTS
// =====================================================================

// TestCustomerAPI_CRUD tests the complete CRUD operations for customers
func TestCustomerAPI_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewPartnerTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	var createdCustomerID string

	t.Run("Create customer", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"code":         "CUST-API-001",
			"name":         "API Test Customer",
			"type":         "organization",
			"contact_name": "John Doe",
			"phone":        "13800138000",
			"email":        "test@example.com",
			"address":      "123 Test Street",
			"city":         "Shanghai",
			"province":     "Shanghai",
			"credit_limit": 10000.00,
		}

		w := ts.Request(http.MethodPost, "/api/v1/partner/customers", reqBody, tenantID)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		createdCustomerID = data["id"].(string)
		assert.NotEmpty(t, createdCustomerID)
		assert.Equal(t, "CUST-API-001", data["code"])
		assert.Equal(t, "API Test Customer", data["name"])
		assert.Equal(t, "organization", data["type"])
		assert.Equal(t, "active", data["status"])
	})

	t.Run("Get customer by ID", func(t *testing.T) {
		require.NotEmpty(t, createdCustomerID, "Customer ID should be set from Create test")

		w := ts.Request(http.MethodGet, "/api/v1/partner/customers/"+createdCustomerID, nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, createdCustomerID, data["id"])
		assert.Equal(t, "CUST-API-001", data["code"])
	})

	t.Run("Get customer by code", func(t *testing.T) {
		w := ts.Request(http.MethodGet, "/api/v1/partner/customers/code/CUST-API-001", nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "CUST-API-001", data["code"])
	})

	t.Run("Update customer", func(t *testing.T) {
		require.NotEmpty(t, createdCustomerID)

		reqBody := map[string]interface{}{
			"name":         "Updated API Customer",
			"contact_name": "Jane Doe",
			"phone":        "13900139000",
		}

		w := ts.Request(http.MethodPut, "/api/v1/partner/customers/"+createdCustomerID, reqBody, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "Updated API Customer", data["name"])
		assert.Equal(t, "Jane Doe", data["contact_name"])
	})

	t.Run("Update customer code", func(t *testing.T) {
		require.NotEmpty(t, createdCustomerID)

		reqBody := map[string]interface{}{
			"code": "CUST-API-UPDATED",
		}

		w := ts.Request(http.MethodPut, "/api/v1/partner/customers/"+createdCustomerID+"/code", reqBody, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "CUST-API-UPDATED", data["code"])
	})

	t.Run("Delete customer", func(t *testing.T) {
		require.NotEmpty(t, createdCustomerID)

		w := ts.Request(http.MethodDelete, "/api/v1/partner/customers/"+createdCustomerID, nil, tenantID)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify customer is deleted
		w = ts.Request(http.MethodGet, "/api/v1/partner/customers/"+createdCustomerID, nil, tenantID)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestCustomerAPI_StatusOperations tests activate/deactivate/suspend operations
func TestCustomerAPI_StatusOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewPartnerTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create a customer first
	reqBody := map[string]interface{}{
		"code": "STATUS-CUST-001",
		"name": "Status Test Customer",
		"type": "individual",
	}

	w := ts.Request(http.MethodPost, "/api/v1/partner/customers", reqBody, tenantID)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)

	customerID := createResp.Data.(map[string]interface{})["id"].(string)

	t.Run("Deactivate active customer", func(t *testing.T) {
		w := ts.Request(http.MethodPost, "/api/v1/partner/customers/"+customerID+"/deactivate", nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "inactive", data["status"])
	})

	t.Run("Activate inactive customer", func(t *testing.T) {
		w := ts.Request(http.MethodPost, "/api/v1/partner/customers/"+customerID+"/activate", nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "active", data["status"])
	})

	t.Run("Suspend customer", func(t *testing.T) {
		w := ts.Request(http.MethodPost, "/api/v1/partner/customers/"+customerID+"/suspend", nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "suspended", data["status"])
	})
}

// TestCustomerAPI_BalanceOperations tests balance add/deduct operations
func TestCustomerAPI_BalanceOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewPartnerTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create a customer first
	reqBody := map[string]interface{}{
		"code": "BALANCE-CUST-001",
		"name": "Balance Test Customer",
		"type": "individual",
	}

	w := ts.Request(http.MethodPost, "/api/v1/partner/customers", reqBody, tenantID)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)

	customerID := createResp.Data.(map[string]interface{})["id"].(string)

	t.Run("Add balance", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"amount": 1000.00,
		}

		w := ts.Request(http.MethodPost, "/api/v1/partner/customers/"+customerID+"/balance/add", reqBody, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "1000", data["balance"])
	})

	t.Run("Deduct balance", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"amount": 300.00,
		}

		w := ts.Request(http.MethodPost, "/api/v1/partner/customers/"+customerID+"/balance/deduct", reqBody, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "700", data["balance"])
	})

	t.Run("Cannot deduct more than balance", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"amount": 10000.00,
		}

		w := ts.Request(http.MethodPost, "/api/v1/partner/customers/"+customerID+"/balance/deduct", reqBody, tenantID)

		// Should fail - insufficient balance
		assert.True(t, w.Code >= 400, "Expected error status code, got %d", w.Code)
	})
}

// TestCustomerAPI_LevelOperations tests customer level setting
func TestCustomerAPI_LevelOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewPartnerTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create a customer first
	reqBody := map[string]interface{}{
		"code": "LEVEL-CUST-001",
		"name": "Level Test Customer",
		"type": "organization",
	}

	w := ts.Request(http.MethodPost, "/api/v1/partner/customers", reqBody, tenantID)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)

	customerID := createResp.Data.(map[string]interface{})["id"].(string)

	t.Run("Set customer level to gold", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"level": "gold",
		}

		w := ts.Request(http.MethodPut, "/api/v1/partner/customers/"+customerID+"/level", reqBody, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "gold", data["level"])
	})

	t.Run("Set customer level to vip", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"level": "vip",
		}

		w := ts.Request(http.MethodPut, "/api/v1/partner/customers/"+customerID+"/level", reqBody, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "vip", data["level"])
	})
}

// TestCustomerAPI_List tests listing with pagination and filtering
func TestCustomerAPI_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewPartnerTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create multiple customers
	for i := 1; i <= 15; i++ {
		customerType := "individual"
		if i%2 == 0 {
			customerType = "organization"
		}
		reqBody := map[string]interface{}{
			"code": fmt.Sprintf("LIST-CUST-%03d", i),
			"name": fmt.Sprintf("List Customer %d", i),
			"type": customerType,
		}
		w := ts.Request(http.MethodPost, "/api/v1/partner/customers", reqBody, tenantID)
		require.Equal(t, http.StatusCreated, w.Code)
	}

	t.Run("List with default pagination", func(t *testing.T) {
		w := ts.Request(http.MethodGet, "/api/v1/partner/customers?page=1&page_size=20", nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotNil(t, resp.Meta)
		assert.Equal(t, int64(15), resp.Meta.Total)
		assert.Equal(t, 1, resp.Meta.Page)
	})

	t.Run("List with custom pagination", func(t *testing.T) {
		w := ts.Request(http.MethodGet, "/api/v1/partner/customers?page=2&page_size=5", nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotNil(t, resp.Meta)
		assert.Equal(t, 2, resp.Meta.Page)
		assert.Equal(t, 5, resp.Meta.PageSize)

		data := resp.Data.([]interface{})
		assert.LessOrEqual(t, len(data), 5)
	})

	t.Run("Get status counts", func(t *testing.T) {
		w := ts.Request(http.MethodGet, "/api/v1/partner/customers/stats/count", nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, float64(15), data["active"])
		assert.Equal(t, float64(0), data["inactive"])
		assert.Equal(t, float64(0), data["suspended"])
		assert.Equal(t, float64(15), data["total"])
	})
}

// TestCustomerAPI_Validation tests request validation errors
func TestCustomerAPI_Validation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewPartnerTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	t.Run("Create with missing required fields", func(t *testing.T) {
		// Missing code
		reqBody := map[string]interface{}{
			"name": "Test Customer",
			"type": "individual",
		}
		w := ts.Request(http.MethodPost, "/api/v1/partner/customers", reqBody, tenantID)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Missing name
		reqBody = map[string]interface{}{
			"code": "TEST-001",
			"type": "individual",
		}
		w = ts.Request(http.MethodPost, "/api/v1/partner/customers", reqBody, tenantID)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Missing type
		reqBody = map[string]interface{}{
			"code": "TEST-001",
			"name": "Test Customer",
		}
		w = ts.Request(http.MethodPost, "/api/v1/partner/customers", reqBody, tenantID)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Create with invalid type", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"code": "TEST-001",
			"name": "Test Customer",
			"type": "invalid_type",
		}
		w := ts.Request(http.MethodPost, "/api/v1/partner/customers", reqBody, tenantID)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Get with invalid UUID", func(t *testing.T) {
		w := ts.Request(http.MethodGet, "/api/v1/partner/customers/not-a-uuid", nil, tenantID)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Update non-existent customer", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		reqBody := map[string]interface{}{
			"name": "Updated Name",
		}
		w := ts.Request(http.MethodPut, "/api/v1/partner/customers/"+nonExistentID, reqBody, tenantID)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestCustomerAPI_DuplicateCode tests duplicate code handling
func TestCustomerAPI_DuplicateCode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewPartnerTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create first customer
	reqBody := map[string]interface{}{
		"code": "DUPE-CUST-001",
		"name": "First Customer",
		"type": "individual",
	}
	w := ts.Request(http.MethodPost, "/api/v1/partner/customers", reqBody, tenantID)
	require.Equal(t, http.StatusCreated, w.Code)

	t.Run("Create with duplicate code fails", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"code": "DUPE-CUST-001",
			"name": "Second Customer",
			"type": "organization",
		}
		w := ts.Request(http.MethodPost, "/api/v1/partner/customers", reqBody, tenantID)
		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

// TestCustomerAPI_TenantIsolation tests that customers are isolated by tenant
func TestCustomerAPI_TenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewPartnerTestServer(t)

	tenant1 := uuid.New()
	tenant2 := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenant1)
	ts.DB.CreateTestTenantWithUUID(tenant2)

	// Create customer for tenant 1
	reqBody := map[string]interface{}{
		"code": "TENANT-CUST-001",
		"name": "Tenant 1 Customer",
		"type": "individual",
	}
	w := ts.Request(http.MethodPost, "/api/v1/partner/customers", reqBody, tenant1)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)
	customerID := createResp.Data.(map[string]interface{})["id"].(string)

	t.Run("Tenant 2 cannot see Tenant 1 customer", func(t *testing.T) {
		w := ts.Request(http.MethodGet, "/api/v1/partner/customers/"+customerID, nil, tenant2)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Tenant 2 cannot update Tenant 1 customer", func(t *testing.T) {
		updateBody := map[string]interface{}{
			"name": "Hacked Name",
		}
		w := ts.Request(http.MethodPut, "/api/v1/partner/customers/"+customerID, updateBody, tenant2)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Same code allowed for different tenants", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"code": "TENANT-CUST-001", // Same code as tenant 1
			"name": "Tenant 2 Same Code",
			"type": "organization",
		}
		w := ts.Request(http.MethodPost, "/api/v1/partner/customers", reqBody, tenant2)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

// =====================================================================
// SUPPLIER API TESTS
// =====================================================================

// TestSupplierAPI_CRUD tests the complete CRUD operations for suppliers
func TestSupplierAPI_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewPartnerTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	var createdSupplierID string

	t.Run("Create supplier", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"code":         "SUP-API-001",
			"name":         "API Test Supplier",
			"type":         "distributor",
			"contact_name": "Mike Johnson",
			"phone":        "13600136000",
			"email":        "supplier@example.com",
			"address":      "789 Industrial Park",
			"city":         "Guangzhou",
			"province":     "Guangdong",
			"credit_days":  30,
			"credit_limit": 50000.00,
			"rating":       4,
		}

		w := ts.Request(http.MethodPost, "/api/v1/partner/suppliers", reqBody, tenantID)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		createdSupplierID = data["id"].(string)
		assert.NotEmpty(t, createdSupplierID)
		assert.Equal(t, "SUP-API-001", data["code"])
		assert.Equal(t, "API Test Supplier", data["name"])
		assert.Equal(t, "distributor", data["type"])
		assert.Equal(t, "active", data["status"])
		assert.Equal(t, float64(4), data["rating"])
	})

	t.Run("Get supplier by ID", func(t *testing.T) {
		require.NotEmpty(t, createdSupplierID)

		w := ts.Request(http.MethodGet, "/api/v1/partner/suppliers/"+createdSupplierID, nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, createdSupplierID, data["id"])
		assert.Equal(t, "SUP-API-001", data["code"])
	})

	t.Run("Get supplier by code", func(t *testing.T) {
		w := ts.Request(http.MethodGet, "/api/v1/partner/suppliers/code/SUP-API-001", nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "SUP-API-001", data["code"])
	})

	t.Run("Update supplier", func(t *testing.T) {
		require.NotEmpty(t, createdSupplierID)

		reqBody := map[string]interface{}{
			"name":         "Updated API Supplier",
			"contact_name": "Sarah Chen",
			"phone":        "13700137000",
		}

		w := ts.Request(http.MethodPut, "/api/v1/partner/suppliers/"+createdSupplierID, reqBody, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "Updated API Supplier", data["name"])
		assert.Equal(t, "Sarah Chen", data["contact_name"])
	})

	t.Run("Update supplier code", func(t *testing.T) {
		require.NotEmpty(t, createdSupplierID)

		reqBody := map[string]interface{}{
			"code": "SUP-API-UPDATED",
		}

		w := ts.Request(http.MethodPut, "/api/v1/partner/suppliers/"+createdSupplierID+"/code", reqBody, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "SUP-API-UPDATED", data["code"])
	})

	t.Run("Delete supplier", func(t *testing.T) {
		require.NotEmpty(t, createdSupplierID)

		w := ts.Request(http.MethodDelete, "/api/v1/partner/suppliers/"+createdSupplierID, nil, tenantID)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify supplier is deleted
		w = ts.Request(http.MethodGet, "/api/v1/partner/suppliers/"+createdSupplierID, nil, tenantID)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestSupplierAPI_StatusOperations tests activate/deactivate/block operations
func TestSupplierAPI_StatusOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewPartnerTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create a supplier first
	reqBody := map[string]interface{}{
		"code": "STATUS-SUP-001",
		"name": "Status Test Supplier",
		"type": "manufacturer",
	}

	w := ts.Request(http.MethodPost, "/api/v1/partner/suppliers", reqBody, tenantID)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)

	supplierID := createResp.Data.(map[string]interface{})["id"].(string)

	t.Run("Deactivate active supplier", func(t *testing.T) {
		w := ts.Request(http.MethodPost, "/api/v1/partner/suppliers/"+supplierID+"/deactivate", nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "inactive", data["status"])
	})

	t.Run("Activate inactive supplier", func(t *testing.T) {
		w := ts.Request(http.MethodPost, "/api/v1/partner/suppliers/"+supplierID+"/activate", nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "active", data["status"])
	})

	t.Run("Block supplier", func(t *testing.T) {
		w := ts.Request(http.MethodPost, "/api/v1/partner/suppliers/"+supplierID+"/block", nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "blocked", data["status"])
	})
}

// TestSupplierAPI_RatingAndPaymentTerms tests rating and payment terms operations
func TestSupplierAPI_RatingAndPaymentTerms(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewPartnerTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create a supplier first
	reqBody := map[string]interface{}{
		"code": "RATING-SUP-001",
		"name": "Rating Test Supplier",
		"type": "retailer",
	}

	w := ts.Request(http.MethodPost, "/api/v1/partner/suppliers", reqBody, tenantID)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)

	supplierID := createResp.Data.(map[string]interface{})["id"].(string)

	t.Run("Set supplier rating", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"rating": 5,
		}

		w := ts.Request(http.MethodPut, "/api/v1/partner/suppliers/"+supplierID+"/rating", reqBody, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, float64(5), data["rating"])
	})

	t.Run("Set payment terms", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"credit_days":  45,
			"credit_limit": 100000.00,
		}

		w := ts.Request(http.MethodPut, "/api/v1/partner/suppliers/"+supplierID+"/payment-terms", reqBody, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, float64(45), data["credit_days"])
		assert.Equal(t, "100000", data["credit_limit"])
	})
}

// TestSupplierAPI_List tests listing with pagination and filtering
func TestSupplierAPI_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewPartnerTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create multiple suppliers
	supplierTypes := []string{"manufacturer", "distributor", "retailer", "service"}
	for i := 1; i <= 12; i++ {
		reqBody := map[string]interface{}{
			"code":   fmt.Sprintf("LIST-SUP-%03d", i),
			"name":   fmt.Sprintf("List Supplier %d", i),
			"type":   supplierTypes[(i-1)%4],
			"rating": (i % 5) + 1,
		}
		w := ts.Request(http.MethodPost, "/api/v1/partner/suppliers", reqBody, tenantID)
		require.Equal(t, http.StatusCreated, w.Code)
	}

	t.Run("List with pagination", func(t *testing.T) {
		w := ts.Request(http.MethodGet, "/api/v1/partner/suppliers?page=1&page_size=5", nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotNil(t, resp.Meta)
		assert.Equal(t, int64(12), resp.Meta.Total)
	})

	t.Run("Get status counts", func(t *testing.T) {
		w := ts.Request(http.MethodGet, "/api/v1/partner/suppliers/stats/count", nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, float64(12), data["active"])
		assert.Equal(t, float64(0), data["inactive"])
		assert.Equal(t, float64(0), data["blocked"])
		assert.Equal(t, float64(12), data["total"])
	})
}

// TestSupplierAPI_Validation tests request validation errors
func TestSupplierAPI_Validation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewPartnerTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	t.Run("Create with missing required fields", func(t *testing.T) {
		// Missing type
		reqBody := map[string]interface{}{
			"code": "TEST-001",
			"name": "Test Supplier",
		}
		w := ts.Request(http.MethodPost, "/api/v1/partner/suppliers", reqBody, tenantID)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Create with invalid type", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"code": "TEST-001",
			"name": "Test Supplier",
			"type": "invalid_type",
		}
		w := ts.Request(http.MethodPost, "/api/v1/partner/suppliers", reqBody, tenantID)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Create with invalid rating", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"code":   "TEST-001",
			"name":   "Test Supplier",
			"type":   "manufacturer",
			"rating": 10, // Max is 5
		}
		w := ts.Request(http.MethodPost, "/api/v1/partner/suppliers", reqBody, tenantID)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestSupplierAPI_TenantIsolation tests that suppliers are isolated by tenant
func TestSupplierAPI_TenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewPartnerTestServer(t)

	tenant1 := uuid.New()
	tenant2 := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenant1)
	ts.DB.CreateTestTenantWithUUID(tenant2)

	// Create supplier for tenant 1
	reqBody := map[string]interface{}{
		"code": "TENANT-SUP-001",
		"name": "Tenant 1 Supplier",
		"type": "distributor",
	}
	w := ts.Request(http.MethodPost, "/api/v1/partner/suppliers", reqBody, tenant1)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)
	supplierID := createResp.Data.(map[string]interface{})["id"].(string)

	t.Run("Tenant 2 cannot see Tenant 1 supplier", func(t *testing.T) {
		w := ts.Request(http.MethodGet, "/api/v1/partner/suppliers/"+supplierID, nil, tenant2)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Same code allowed for different tenants", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"code": "TENANT-SUP-001", // Same code as tenant 1
			"name": "Tenant 2 Same Code",
			"type": "manufacturer",
		}
		w := ts.Request(http.MethodPost, "/api/v1/partner/suppliers", reqBody, tenant2)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

// =====================================================================
// WAREHOUSE API TESTS
// =====================================================================

// TestWarehouseAPI_CRUD tests the complete CRUD operations for warehouses
func TestWarehouseAPI_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewPartnerTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	var createdWarehouseID string

	t.Run("Create warehouse", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"code":         "WH-API-001",
			"name":         "API Test Warehouse",
			"type":         "physical",
			"contact_name": "Wang Wei",
			"phone":        "13500135000",
			"email":        "warehouse@example.com",
			"address":      "Building 5, Industrial Zone",
			"city":         "Shanghai",
			"province":     "Shanghai",
			"is_default":   true,
			"capacity":     10000,
		}

		w := ts.Request(http.MethodPost, "/api/v1/partner/warehouses", reqBody, tenantID)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		createdWarehouseID = data["id"].(string)
		assert.NotEmpty(t, createdWarehouseID)
		assert.Equal(t, "WH-API-001", data["code"])
		assert.Equal(t, "API Test Warehouse", data["name"])
		assert.Equal(t, "physical", data["type"])
		assert.Equal(t, "active", data["status"])
		assert.Equal(t, true, data["is_default"])
	})

	t.Run("Get warehouse by ID", func(t *testing.T) {
		require.NotEmpty(t, createdWarehouseID)

		w := ts.Request(http.MethodGet, "/api/v1/partner/warehouses/"+createdWarehouseID, nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, createdWarehouseID, data["id"])
		assert.Equal(t, "WH-API-001", data["code"])
	})

	t.Run("Get warehouse by code", func(t *testing.T) {
		w := ts.Request(http.MethodGet, "/api/v1/partner/warehouses/code/WH-API-001", nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "WH-API-001", data["code"])
	})

	t.Run("Get default warehouse", func(t *testing.T) {
		w := ts.Request(http.MethodGet, "/api/v1/partner/warehouses/default", nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, createdWarehouseID, data["id"])
		assert.Equal(t, true, data["is_default"])
	})

	t.Run("Update warehouse", func(t *testing.T) {
		require.NotEmpty(t, createdWarehouseID)

		reqBody := map[string]interface{}{
			"name":         "Updated API Warehouse",
			"contact_name": "Li Ming",
			"capacity":     15000,
		}

		w := ts.Request(http.MethodPut, "/api/v1/partner/warehouses/"+createdWarehouseID, reqBody, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "Updated API Warehouse", data["name"])
		assert.Equal(t, "Li Ming", data["contact_name"])
		assert.Equal(t, float64(15000), data["capacity"])
	})

	t.Run("Update warehouse code", func(t *testing.T) {
		require.NotEmpty(t, createdWarehouseID)

		reqBody := map[string]interface{}{
			"code": "WH-API-UPDATED",
		}

		w := ts.Request(http.MethodPut, "/api/v1/partner/warehouses/"+createdWarehouseID+"/code", reqBody, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "WH-API-UPDATED", data["code"])
	})

	t.Run("Delete warehouse", func(t *testing.T) {
		// Create a non-default warehouse to delete
		reqBody := map[string]interface{}{
			"code":       "WH-DELETE-001",
			"name":       "Delete Test Warehouse",
			"type":       "virtual",
			"is_default": false,
		}
		w := ts.Request(http.MethodPost, "/api/v1/partner/warehouses", reqBody, tenantID)
		require.Equal(t, http.StatusCreated, w.Code)

		var createResp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &createResp)
		require.NoError(t, err)
		deleteID := createResp.Data.(map[string]interface{})["id"].(string)

		w = ts.Request(http.MethodDelete, "/api/v1/partner/warehouses/"+deleteID, nil, tenantID)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify warehouse is deleted
		w = ts.Request(http.MethodGet, "/api/v1/partner/warehouses/"+deleteID, nil, tenantID)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestWarehouseAPI_StatusOperations tests enable/disable operations
func TestWarehouseAPI_StatusOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewPartnerTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create a warehouse first
	reqBody := map[string]interface{}{
		"code":       "STATUS-WH-001",
		"name":       "Status Test Warehouse",
		"type":       "physical",
		"is_default": false,
	}

	w := ts.Request(http.MethodPost, "/api/v1/partner/warehouses", reqBody, tenantID)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)

	warehouseID := createResp.Data.(map[string]interface{})["id"].(string)

	t.Run("Disable active warehouse", func(t *testing.T) {
		w := ts.Request(http.MethodPost, "/api/v1/partner/warehouses/"+warehouseID+"/disable", nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "inactive", data["status"])
	})

	t.Run("Enable inactive warehouse", func(t *testing.T) {
		w := ts.Request(http.MethodPost, "/api/v1/partner/warehouses/"+warehouseID+"/enable", nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "active", data["status"])
	})
}

// TestWarehouseAPI_DefaultOperations tests set-default operation
func TestWarehouseAPI_DefaultOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewPartnerTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create first warehouse as default
	reqBody := map[string]interface{}{
		"code":       "DEFAULT-WH-001",
		"name":       "First Warehouse",
		"type":       "physical",
		"is_default": true,
	}
	w := ts.Request(http.MethodPost, "/api/v1/partner/warehouses", reqBody, tenantID)
	require.Equal(t, http.StatusCreated, w.Code)

	var firstResp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &firstResp)
	require.NoError(t, err)
	firstID := firstResp.Data.(map[string]interface{})["id"].(string)

	// Create second warehouse
	reqBody = map[string]interface{}{
		"code":       "DEFAULT-WH-002",
		"name":       "Second Warehouse",
		"type":       "physical",
		"is_default": false,
	}
	w = ts.Request(http.MethodPost, "/api/v1/partner/warehouses", reqBody, tenantID)
	require.Equal(t, http.StatusCreated, w.Code)

	var secondResp APIResponse
	err = json.Unmarshal(w.Body.Bytes(), &secondResp)
	require.NoError(t, err)
	secondID := secondResp.Data.(map[string]interface{})["id"].(string)

	t.Run("Set second warehouse as default", func(t *testing.T) {
		w := ts.Request(http.MethodPost, "/api/v1/partner/warehouses/"+secondID+"/set-default", nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, true, data["is_default"])

		// Verify first warehouse is no longer default
		w = ts.Request(http.MethodGet, "/api/v1/partner/warehouses/"+firstID, nil, tenantID)
		assert.Equal(t, http.StatusOK, w.Code)

		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		data = resp.Data.(map[string]interface{})
		assert.Equal(t, false, data["is_default"])
	})
}

// TestWarehouseAPI_List tests listing with pagination and filtering
func TestWarehouseAPI_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewPartnerTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create multiple warehouses
	warehouseTypes := []string{"physical", "virtual", "consign", "transit"}
	for i := 1; i <= 8; i++ {
		isDefault := i == 1
		reqBody := map[string]interface{}{
			"code":       fmt.Sprintf("LIST-WH-%03d", i),
			"name":       fmt.Sprintf("List Warehouse %d", i),
			"type":       warehouseTypes[(i-1)%4],
			"is_default": isDefault,
		}
		w := ts.Request(http.MethodPost, "/api/v1/partner/warehouses", reqBody, tenantID)
		require.Equal(t, http.StatusCreated, w.Code)
	}

	t.Run("List with pagination", func(t *testing.T) {
		w := ts.Request(http.MethodGet, "/api/v1/partner/warehouses?page=1&page_size=5", nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotNil(t, resp.Meta)
		assert.Equal(t, int64(8), resp.Meta.Total)
	})

	t.Run("Get status counts", func(t *testing.T) {
		w := ts.Request(http.MethodGet, "/api/v1/partner/warehouses/stats/count", nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, float64(8), data["active"])
		assert.Equal(t, float64(0), data["inactive"])
		assert.Equal(t, float64(8), data["total"])
	})
}

// TestWarehouseAPI_Validation tests request validation errors
func TestWarehouseAPI_Validation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewPartnerTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	t.Run("Create with missing required fields", func(t *testing.T) {
		// Missing type
		reqBody := map[string]interface{}{
			"code": "TEST-001",
			"name": "Test Warehouse",
		}
		w := ts.Request(http.MethodPost, "/api/v1/partner/warehouses", reqBody, tenantID)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Create with invalid type", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"code": "TEST-001",
			"name": "Test Warehouse",
			"type": "invalid_type",
		}
		w := ts.Request(http.MethodPost, "/api/v1/partner/warehouses", reqBody, tenantID)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestWarehouseAPI_TenantIsolation tests that warehouses are isolated by tenant
func TestWarehouseAPI_TenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewPartnerTestServer(t)

	tenant1 := uuid.New()
	tenant2 := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenant1)
	ts.DB.CreateTestTenantWithUUID(tenant2)

	// Create warehouse for tenant 1
	reqBody := map[string]interface{}{
		"code":       "TENANT-WH-001",
		"name":       "Tenant 1 Warehouse",
		"type":       "physical",
		"is_default": true,
	}
	w := ts.Request(http.MethodPost, "/api/v1/partner/warehouses", reqBody, tenant1)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)
	warehouseID := createResp.Data.(map[string]interface{})["id"].(string)

	t.Run("Tenant 2 cannot see Tenant 1 warehouse", func(t *testing.T) {
		w := ts.Request(http.MethodGet, "/api/v1/partner/warehouses/"+warehouseID, nil, tenant2)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Same code allowed for different tenants", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"code":       "TENANT-WH-001", // Same code as tenant 1
			"name":       "Tenant 2 Same Code",
			"type":       "virtual",
			"is_default": true,
		}
		w := ts.Request(http.MethodPost, "/api/v1/partner/warehouses", reqBody, tenant2)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}
