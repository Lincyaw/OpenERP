// Package integration provides integration testing for the ERP backend API.
// This file contains tests for the Product API endpoints against a real database.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	catalogapp "github.com/erp/backend/internal/application/catalog"
	"github.com/erp/backend/internal/infrastructure/persistence"
	"github.com/erp/backend/internal/interfaces/http/handler"
	"github.com/erp/backend/internal/interfaces/http/router"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServer wraps the test database and HTTP server for API testing
type TestServer struct {
	DB     *TestDB
	Engine *gin.Engine
	Router *router.Router
}

// NewTestServer creates a new test server with real database
func NewTestServer(t *testing.T) *TestServer {
	t.Helper()

	gin.SetMode(gin.TestMode)
	testDB := NewTestDB(t)

	// Initialize repositories
	productRepo := persistence.NewGormProductRepository(testDB.DB)
	categoryRepo := persistence.NewGormCategoryRepository(testDB.DB)

	// Initialize services
	productService := catalogapp.NewProductService(productRepo, categoryRepo)

	// Initialize handlers
	productHandler := handler.NewProductHandler(productService)

	// Setup engine
	engine := gin.New()

	// Setup routes
	r := router.NewRouter(engine, router.WithAPIVersion("v1"))

	// Register catalog routes
	catalogRoutes := router.NewDomainGroup("catalog", "/catalog")
	catalogRoutes.POST("/products", productHandler.Create)
	catalogRoutes.GET("/products", productHandler.List)
	catalogRoutes.GET("/products/stats/count", productHandler.CountByStatus)
	catalogRoutes.GET("/products/:id", productHandler.GetByID)
	catalogRoutes.GET("/products/code/:code", productHandler.GetByCode)
	catalogRoutes.PUT("/products/:id", productHandler.Update)
	catalogRoutes.PUT("/products/:id/code", productHandler.UpdateCode)
	catalogRoutes.DELETE("/products/:id", productHandler.Delete)
	catalogRoutes.POST("/products/:id/activate", productHandler.Activate)
	catalogRoutes.POST("/products/:id/deactivate", productHandler.Deactivate)
	catalogRoutes.POST("/products/:id/discontinue", productHandler.Discontinue)

	r.Register(catalogRoutes)
	r.Setup()

	return &TestServer{
		DB:     testDB,
		Engine: engine,
		Router: r,
	}
}

// Request makes an HTTP request to the test server
func (ts *TestServer) Request(method, path string, body interface{}, tenantID ...uuid.UUID) *httptest.ResponseRecorder {
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

// APIResponse represents the standard API response structure
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
	Meta *struct {
		Total    int64 `json:"total"`
		Page     int   `json:"page"`
		PageSize int   `json:"page_size"`
	} `json:"meta,omitempty"`
}

// TestProductAPI_CRUD tests the complete CRUD operations for products
func TestProductAPI_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	var createdProductID string

	t.Run("Create product", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"code":          "API-PROD-001",
			"name":          "API Test Product",
			"unit":          "pcs",
			"description":   "Product created via API test",
			"selling_price": 100.00,
		}

		w := ts.Request(http.MethodPost, "/api/v1/catalog/products", reqBody, tenantID)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		createdProductID = data["id"].(string)
		assert.NotEmpty(t, createdProductID)
		assert.Equal(t, "API-PROD-001", data["code"])
		assert.Equal(t, "API Test Product", data["name"])
		assert.Equal(t, "pcs", data["unit"])
		assert.Equal(t, "active", data["status"])
	})

	t.Run("Get product by ID", func(t *testing.T) {
		require.NotEmpty(t, createdProductID, "Product ID should be set from Create test")

		w := ts.Request(http.MethodGet, "/api/v1/catalog/products/"+createdProductID, nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, createdProductID, data["id"])
		assert.Equal(t, "API-PROD-001", data["code"])
	})

	t.Run("Get product by code", func(t *testing.T) {
		w := ts.Request(http.MethodGet, "/api/v1/catalog/products/code/API-PROD-001", nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "API-PROD-001", data["code"])
	})

	t.Run("Update product", func(t *testing.T) {
		require.NotEmpty(t, createdProductID)

		reqBody := map[string]interface{}{
			"name":        "Updated API Product",
			"description": "Updated description",
		}

		w := ts.Request(http.MethodPut, "/api/v1/catalog/products/"+createdProductID, reqBody, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "Updated API Product", data["name"])
		assert.Equal(t, "Updated description", data["description"])
	})

	t.Run("Update product code", func(t *testing.T) {
		require.NotEmpty(t, createdProductID)

		reqBody := map[string]interface{}{
			"code": "API-PROD-UPDATED",
		}

		w := ts.Request(http.MethodPut, "/api/v1/catalog/products/"+createdProductID+"/code", reqBody, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "API-PROD-UPDATED", data["code"])
	})

	t.Run("Delete product", func(t *testing.T) {
		require.NotEmpty(t, createdProductID)

		w := ts.Request(http.MethodDelete, "/api/v1/catalog/products/"+createdProductID, nil, tenantID)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify product is deleted
		w = ts.Request(http.MethodGet, "/api/v1/catalog/products/"+createdProductID, nil, tenantID)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestProductAPI_StatusOperations tests enable/disable/discontinue operations
func TestProductAPI_StatusOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create a product first
	reqBody := map[string]interface{}{
		"code": "STATUS-PROD-001",
		"name": "Status Test Product",
		"unit": "pcs",
	}

	w := ts.Request(http.MethodPost, "/api/v1/catalog/products", reqBody, tenantID)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)

	productID := createResp.Data.(map[string]interface{})["id"].(string)

	t.Run("Deactivate active product", func(t *testing.T) {
		w := ts.Request(http.MethodPost, "/api/v1/catalog/products/"+productID+"/deactivate", nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "inactive", data["status"])
	})

	t.Run("Activate inactive product", func(t *testing.T) {
		w := ts.Request(http.MethodPost, "/api/v1/catalog/products/"+productID+"/activate", nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "active", data["status"])
	})

	t.Run("Discontinue product", func(t *testing.T) {
		w := ts.Request(http.MethodPost, "/api/v1/catalog/products/"+productID+"/discontinue", nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "discontinued", data["status"])
	})

	t.Run("Cannot activate discontinued product", func(t *testing.T) {
		w := ts.Request(http.MethodPost, "/api/v1/catalog/products/"+productID+"/activate", nil, tenantID)

		// Should fail - discontinued products cannot be reactivated
		// Note: Returns error status (4xx or 5xx), exact status depends on error code mapping
		assert.True(t, w.Code >= 400, "Expected error status code, got %d", w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.False(t, resp.Success)
	})
}

// TestProductAPI_List tests listing with pagination and filtering
func TestProductAPI_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create multiple products
	for i := 1; i <= 15; i++ {
		reqBody := map[string]interface{}{
			"code":          fmt.Sprintf("LIST-PROD-%03d", i),
			"name":          fmt.Sprintf("List Product %d", i),
			"unit":          "pcs",
			"selling_price": float64(i * 10),
		}
		w := ts.Request(http.MethodPost, "/api/v1/catalog/products", reqBody, tenantID)
		require.Equal(t, http.StatusCreated, w.Code)
	}

	t.Run("List with default pagination", func(t *testing.T) {
		// Note: The API requires page and page_size parameters due to validation tags
		w := ts.Request(http.MethodGet, "/api/v1/catalog/products?page=1&page_size=20", nil, tenantID)

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
		w := ts.Request(http.MethodGet, "/api/v1/catalog/products?page=2&page_size=5", nil, tenantID)

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

	t.Run("List with search filter", func(t *testing.T) {
		w := ts.Request(http.MethodGet, "/api/v1/catalog/products?page=1&page_size=20&search=List+Product+1", nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		// Should find products with "List Product 1" in name (1, 10, 11, 12, 13, 14, 15)
		data := resp.Data.([]interface{})
		assert.Greater(t, len(data), 0)
	})

	t.Run("Get status counts", func(t *testing.T) {
		w := ts.Request(http.MethodGet, "/api/v1/catalog/products/stats/count", nil, tenantID)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, float64(15), data["active"])
		assert.Equal(t, float64(0), data["inactive"])
		assert.Equal(t, float64(0), data["discontinued"])
		assert.Equal(t, float64(15), data["total"])
	})
}

// TestProductAPI_Validation tests request validation errors
func TestProductAPI_Validation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	t.Run("Create with missing required fields", func(t *testing.T) {
		// Missing code
		reqBody := map[string]interface{}{
			"name": "Test Product",
			"unit": "pcs",
		}
		w := ts.Request(http.MethodPost, "/api/v1/catalog/products", reqBody, tenantID)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Missing name
		reqBody = map[string]interface{}{
			"code": "TEST-001",
			"unit": "pcs",
		}
		w = ts.Request(http.MethodPost, "/api/v1/catalog/products", reqBody, tenantID)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Missing unit
		reqBody = map[string]interface{}{
			"code": "TEST-001",
			"name": "Test Product",
		}
		w = ts.Request(http.MethodPost, "/api/v1/catalog/products", reqBody, tenantID)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Create with invalid code length", func(t *testing.T) {
		// Code too long (max 50)
		reqBody := map[string]interface{}{
			"code": "THIS-CODE-IS-WAY-TOO-LONG-FOR-A-PRODUCT-CODE-AND-SHOULD-FAIL",
			"name": "Test Product",
			"unit": "pcs",
		}
		w := ts.Request(http.MethodPost, "/api/v1/catalog/products", reqBody, tenantID)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Get with invalid UUID", func(t *testing.T) {
		w := ts.Request(http.MethodGet, "/api/v1/catalog/products/not-a-uuid", nil, tenantID)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Update non-existent product", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		reqBody := map[string]interface{}{
			"name": "Updated Name",
		}
		w := ts.Request(http.MethodPut, "/api/v1/catalog/products/"+nonExistentID, reqBody, tenantID)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Delete non-existent product", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		w := ts.Request(http.MethodDelete, "/api/v1/catalog/products/"+nonExistentID, nil, tenantID)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestProductAPI_DuplicateCode tests duplicate code handling
func TestProductAPI_DuplicateCode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create first product
	reqBody := map[string]interface{}{
		"code": "DUPE-CODE-001",
		"name": "First Product",
		"unit": "pcs",
	}
	w := ts.Request(http.MethodPost, "/api/v1/catalog/products", reqBody, tenantID)
	require.Equal(t, http.StatusCreated, w.Code)

	t.Run("Create with duplicate code fails", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"code": "DUPE-CODE-001",
			"name": "Second Product",
			"unit": "pcs",
		}
		w := ts.Request(http.MethodPost, "/api/v1/catalog/products", reqBody, tenantID)
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("Update code to duplicate fails", func(t *testing.T) {
		// Create another product with different code
		reqBody := map[string]interface{}{
			"code": "DUPE-CODE-002",
			"name": "Another Product",
			"unit": "pcs",
		}
		w := ts.Request(http.MethodPost, "/api/v1/catalog/products", reqBody, tenantID)
		require.Equal(t, http.StatusCreated, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		productID := resp.Data.(map[string]interface{})["id"].(string)

		// Try to update to existing code
		updateBody := map[string]interface{}{
			"code": "DUPE-CODE-001",
		}
		w = ts.Request(http.MethodPut, "/api/v1/catalog/products/"+productID+"/code", updateBody, tenantID)
		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

// TestProductAPI_TenantIsolation tests that products are isolated by tenant
func TestProductAPI_TenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewTestServer(t)

	tenant1 := uuid.New()
	tenant2 := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenant1)
	ts.DB.CreateTestTenantWithUUID(tenant2)

	// Create product for tenant 1
	reqBody := map[string]interface{}{
		"code": "TENANT-PROD-001",
		"name": "Tenant 1 Product",
		"unit": "pcs",
	}
	w := ts.Request(http.MethodPost, "/api/v1/catalog/products", reqBody, tenant1)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)
	productID := createResp.Data.(map[string]interface{})["id"].(string)

	t.Run("Tenant 2 cannot see Tenant 1 product", func(t *testing.T) {
		// Try to get tenant 1's product using tenant 2's context
		w := ts.Request(http.MethodGet, "/api/v1/catalog/products/"+productID, nil, tenant2)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Tenant 2 cannot update Tenant 1 product", func(t *testing.T) {
		updateBody := map[string]interface{}{
			"name": "Hacked Name",
		}
		w := ts.Request(http.MethodPut, "/api/v1/catalog/products/"+productID, updateBody, tenant2)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Tenant 2 cannot delete Tenant 1 product", func(t *testing.T) {
		w := ts.Request(http.MethodDelete, "/api/v1/catalog/products/"+productID, nil, tenant2)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Tenant 1 product count excludes Tenant 2 products", func(t *testing.T) {
		// Create product for tenant 2
		reqBody := map[string]interface{}{
			"code": "TENANT2-PROD-001",
			"name": "Tenant 2 Product",
			"unit": "pcs",
		}
		w := ts.Request(http.MethodPost, "/api/v1/catalog/products", reqBody, tenant2)
		require.Equal(t, http.StatusCreated, w.Code)

		// Get tenant 1's count
		w = ts.Request(http.MethodGet, "/api/v1/catalog/products/stats/count", nil, tenant1)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, float64(1), data["total"]) // Only tenant 1's product

		// Get tenant 2's count
		w = ts.Request(http.MethodGet, "/api/v1/catalog/products/stats/count", nil, tenant2)
		assert.Equal(t, http.StatusOK, w.Code)

		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		data = resp.Data.(map[string]interface{})
		assert.Equal(t, float64(1), data["total"]) // Only tenant 2's product
	})

	t.Run("Same code allowed for different tenants", func(t *testing.T) {
		// Tenant 2 can use the same code as tenant 1
		reqBody := map[string]interface{}{
			"code": "TENANT-PROD-001", // Same code as tenant 1
			"name": "Tenant 2 Same Code",
			"unit": "pcs",
		}
		w := ts.Request(http.MethodPost, "/api/v1/catalog/products", reqBody, tenant2)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

// TestProductAPI_Barcode tests barcode handling
func TestProductAPI_Barcode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	t.Run("Create product with barcode", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"code":    "BARCODE-PROD-001",
			"name":    "Barcode Product",
			"unit":    "pcs",
			"barcode": "1234567890123",
		}
		w := ts.Request(http.MethodPost, "/api/v1/catalog/products", reqBody, tenantID)
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "1234567890123", data["barcode"])
	})

	t.Run("Update product barcode", func(t *testing.T) {
		// Create product without barcode
		reqBody := map[string]interface{}{
			"code": "BARCODE-PROD-002",
			"name": "No Barcode Product",
			"unit": "pcs",
		}
		w := ts.Request(http.MethodPost, "/api/v1/catalog/products", reqBody, tenantID)
		require.Equal(t, http.StatusCreated, w.Code)

		var createResp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &createResp)
		require.NoError(t, err)

		productID := createResp.Data.(map[string]interface{})["id"].(string)

		// Update with barcode
		updateBody := map[string]interface{}{
			"barcode": "9876543210987",
		}
		w = ts.Request(http.MethodPut, "/api/v1/catalog/products/"+productID, updateBody, tenantID)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "9876543210987", data["barcode"])
	})
}

// TestProductAPI_Pricing tests price field handling
func TestProductAPI_Pricing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	t.Run("Create product with prices", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"code":           "PRICE-PROD-001",
			"name":           "Priced Product",
			"unit":           "pcs",
			"purchase_price": 50.00,
			"selling_price":  100.00,
		}
		w := ts.Request(http.MethodPost, "/api/v1/catalog/products", reqBody, tenantID)
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "50", data["purchase_price"])
		assert.Equal(t, "100", data["selling_price"])
	})

	t.Run("Update product prices", func(t *testing.T) {
		// Get the product ID
		w := ts.Request(http.MethodGet, "/api/v1/catalog/products/code/PRICE-PROD-001", nil, tenantID)
		require.Equal(t, http.StatusOK, w.Code)

		var getResp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &getResp)
		require.NoError(t, err)

		productID := getResp.Data.(map[string]interface{})["id"].(string)

		// Update prices
		updateBody := map[string]interface{}{
			"purchase_price": 60.00,
			"selling_price":  120.00,
		}
		w = ts.Request(http.MethodPut, "/api/v1/catalog/products/"+productID, updateBody, tenantID)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "60", data["purchase_price"])
		assert.Equal(t, "120", data["selling_price"])
	})
}

// TestProductAPI_Concurrency tests concurrent operations
func TestProductAPI_Concurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ts := NewTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	t.Run("Concurrent product creation", func(t *testing.T) {
		// Create 10 products concurrently
		done := make(chan bool, 10)
		errors := make(chan error, 10)

		for i := 0; i < 10; i++ {
			go func(idx int) {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				select {
				case <-ctx.Done():
					errors <- ctx.Err()
					done <- false
					return
				default:
				}

				reqBody := map[string]interface{}{
					"code": fmt.Sprintf("CONCURRENT-PROD-%03d", idx),
					"name": fmt.Sprintf("Concurrent Product %d", idx),
					"unit": "pcs",
				}
				w := ts.Request(http.MethodPost, "/api/v1/catalog/products", reqBody, tenantID)
				if w.Code != http.StatusCreated {
					errors <- fmt.Errorf("failed to create product %d: status %d", idx, w.Code)
					done <- false
					return
				}
				errors <- nil
				done <- true
			}(i)
		}

		// Wait for all goroutines
		successCount := 0
		for i := 0; i < 10; i++ {
			if <-done {
				successCount++
			}
			err := <-errors
			if err != nil {
				t.Logf("Error in concurrent creation: %v", err)
			}
		}

		// All 10 should succeed since they have unique codes
		assert.Equal(t, 10, successCount)

		// Verify count
		w := ts.Request(http.MethodGet, "/api/v1/catalog/products/stats/count", nil, tenantID)
		require.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, float64(10), data["total"])
	})
}
