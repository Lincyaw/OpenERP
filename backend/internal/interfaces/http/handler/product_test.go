package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	catalogapp "github.com/erp/backend/internal/application/catalog"
	"github.com/erp/backend/internal/domain/catalog"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockProductRepository implements catalog.ProductRepository for testing
type MockProductRepository struct {
	mock.Mock
}

// MockValidationStrategyGetter implements catalogapp.ValidationStrategyGetter for testing
type MockValidationStrategyGetter struct {
	mock.Mock
}

func (m *MockValidationStrategyGetter) GetValidationStrategyOrDefault(name string) strategy.ProductValidationStrategy {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(strategy.ProductValidationStrategy)
}

func (m *MockProductRepository) FindByID(ctx context.Context, id uuid.UUID) (*catalog.Product, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*catalog.Product), args.Error(1)
}

func (m *MockProductRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*catalog.Product, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*catalog.Product), args.Error(1)
}

func (m *MockProductRepository) FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*catalog.Product, error) {
	args := m.Called(ctx, tenantID, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*catalog.Product), args.Error(1)
}

func (m *MockProductRepository) FindByBarcode(ctx context.Context, tenantID uuid.UUID, barcode string) (*catalog.Product, error) {
	args := m.Called(ctx, tenantID, barcode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*catalog.Product), args.Error(1)
}

func (m *MockProductRepository) FindAll(ctx context.Context, filter shared.Filter) ([]catalog.Product, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]catalog.Product), args.Error(1)
}

func (m *MockProductRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]catalog.Product, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]catalog.Product), args.Error(1)
}

func (m *MockProductRepository) FindByCategory(ctx context.Context, tenantID, categoryID uuid.UUID, filter shared.Filter) ([]catalog.Product, error) {
	args := m.Called(ctx, tenantID, categoryID, filter)
	return args.Get(0).([]catalog.Product), args.Error(1)
}

func (m *MockProductRepository) FindByCategories(ctx context.Context, tenantID uuid.UUID, categoryIDs []uuid.UUID, filter shared.Filter) ([]catalog.Product, error) {
	args := m.Called(ctx, tenantID, categoryIDs, filter)
	return args.Get(0).([]catalog.Product), args.Error(1)
}

func (m *MockProductRepository) FindActive(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]catalog.Product, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]catalog.Product), args.Error(1)
}

func (m *MockProductRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status catalog.ProductStatus, filter shared.Filter) ([]catalog.Product, error) {
	args := m.Called(ctx, tenantID, status, filter)
	return args.Get(0).([]catalog.Product), args.Error(1)
}

func (m *MockProductRepository) FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]catalog.Product, error) {
	args := m.Called(ctx, tenantID, ids)
	return args.Get(0).([]catalog.Product), args.Error(1)
}

func (m *MockProductRepository) FindByCodes(ctx context.Context, tenantID uuid.UUID, codes []string) ([]catalog.Product, error) {
	args := m.Called(ctx, tenantID, codes)
	return args.Get(0).([]catalog.Product), args.Error(1)
}

func (m *MockProductRepository) Save(ctx context.Context, product *catalog.Product) error {
	args := m.Called(ctx, product)
	return args.Error(0)
}

func (m *MockProductRepository) SaveBatch(ctx context.Context, products []*catalog.Product) error {
	args := m.Called(ctx, products)
	return args.Error(0)
}

func (m *MockProductRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockProductRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *MockProductRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockProductRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockProductRepository) CountByCategory(ctx context.Context, tenantID, categoryID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID, categoryID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockProductRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status catalog.ProductStatus) (int64, error) {
	args := m.Called(ctx, tenantID, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockProductRepository) ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error) {
	args := m.Called(ctx, tenantID, code)
	return args.Bool(0), args.Error(1)
}

func (m *MockProductRepository) ExistsByBarcode(ctx context.Context, tenantID uuid.UUID, barcode string) (bool, error) {
	args := m.Called(ctx, tenantID, barcode)
	return args.Bool(0), args.Error(1)
}

// MockCategoryRepository implements catalog.CategoryRepository for testing
type MockCategoryRepository struct {
	mock.Mock
}

func (m *MockCategoryRepository) FindByID(ctx context.Context, id uuid.UUID) (*catalog.Category, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*catalog.Category), args.Error(1)
}

func (m *MockCategoryRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*catalog.Category, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*catalog.Category), args.Error(1)
}

func (m *MockCategoryRepository) FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*catalog.Category, error) {
	args := m.Called(ctx, tenantID, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*catalog.Category), args.Error(1)
}

func (m *MockCategoryRepository) FindChildren(ctx context.Context, tenantID, parentID uuid.UUID) ([]catalog.Category, error) {
	args := m.Called(ctx, tenantID, parentID)
	return args.Get(0).([]catalog.Category), args.Error(1)
}

func (m *MockCategoryRepository) FindRootCategories(ctx context.Context, tenantID uuid.UUID) ([]catalog.Category, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]catalog.Category), args.Error(1)
}

func (m *MockCategoryRepository) FindDescendants(ctx context.Context, tenantID, parentID uuid.UUID) ([]catalog.Category, error) {
	args := m.Called(ctx, tenantID, parentID)
	return args.Get(0).([]catalog.Category), args.Error(1)
}

func (m *MockCategoryRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]catalog.Category, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]catalog.Category), args.Error(1)
}

func (m *MockCategoryRepository) Save(ctx context.Context, category *catalog.Category) error {
	args := m.Called(ctx, category)
	return args.Error(0)
}

func (m *MockCategoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCategoryRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *MockCategoryRepository) ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error) {
	args := m.Called(ctx, tenantID, code)
	return args.Bool(0), args.Error(1)
}

func (m *MockCategoryRepository) HasChildren(ctx context.Context, tenantID, parentID uuid.UUID) (bool, error) {
	args := m.Called(ctx, tenantID, parentID)
	return args.Bool(0), args.Error(1)
}

func (m *MockCategoryRepository) HasProducts(ctx context.Context, tenantID, categoryID uuid.UUID) (bool, error) {
	args := m.Called(ctx, tenantID, categoryID)
	return args.Bool(0), args.Error(1)
}

func (m *MockCategoryRepository) UpdatePath(ctx context.Context, tenantID, categoryID uuid.UUID, newPath string, newLevel int) error {
	args := m.Called(ctx, tenantID, categoryID, newPath, newLevel)
	return args.Error(0)
}

func (m *MockCategoryRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCategoryRepository) FindAll(ctx context.Context, filter shared.Filter) ([]catalog.Category, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]catalog.Category), args.Error(1)
}

func (m *MockCategoryRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCategoryRepository) FindByPath(ctx context.Context, tenantID uuid.UUID, path string) (*catalog.Category, error) {
	args := m.Called(ctx, tenantID, path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*catalog.Category), args.Error(1)
}

// Test setup helpers
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Add test authentication middleware that sets JWT context values
	// Uses a default test tenant and user ID for all requests
	router.Use(func(c *gin.Context) {
		setJWTContext(c, uuid.MustParse("00000000-0000-0000-0000-000000000001"), uuid.New())
		c.Next()
	})
	return router
}

func setupProductHandler(productRepo *MockProductRepository, categoryRepo *MockCategoryRepository) *ProductHandler {
	mockStrategyGetter := new(MockValidationStrategyGetter)
	mockStrategyGetter.On("GetValidationStrategyOrDefault", mock.Anything).Return(nil)
	productService := catalogapp.NewProductService(productRepo, categoryRepo, mockStrategyGetter)
	return NewProductHandler(productService)
}

func createTestProduct(tenantID uuid.UUID) *catalog.Product {
	product, _ := catalog.NewProduct(tenantID, "SKU-001", "Test Product", "pcs")
	return product
}

// Tests

func TestProductHandler_Create_Success(t *testing.T) {
	productRepo := new(MockProductRepository)
	categoryRepo := new(MockCategoryRepository)
	handler := setupProductHandler(productRepo, categoryRepo)

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	productRepo.On("ExistsByCode", mock.Anything, tenantID, "SKU-001").Return(false, nil)
	productRepo.On("Save", mock.Anything, mock.AnythingOfType("*catalog.Product")).Return(nil)

	router := setupTestRouter()
	router.POST("/products", handler.Create)

	reqBody := CreateProductRequest{
		Code: "SKU-001",
		Name: "Test Product",
		Unit: "pcs",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	productRepo.AssertExpectations(t)
}

func TestProductHandler_Create_DuplicateCode(t *testing.T) {
	productRepo := new(MockProductRepository)
	categoryRepo := new(MockCategoryRepository)
	handler := setupProductHandler(productRepo, categoryRepo)

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	productRepo.On("ExistsByCode", mock.Anything, tenantID, "SKU-001").Return(true, nil)

	router := setupTestRouter()
	router.POST("/products", handler.Create)

	reqBody := CreateProductRequest{
		Code: "SKU-001",
		Name: "Test Product",
		Unit: "pcs",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	productRepo.AssertExpectations(t)
}

func TestProductHandler_Create_InvalidJSON(t *testing.T) {
	productRepo := new(MockProductRepository)
	categoryRepo := new(MockCategoryRepository)
	handler := setupProductHandler(productRepo, categoryRepo)

	router := setupTestRouter()
	router.POST("/products", handler.Create)

	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProductHandler_GetByID_Success(t *testing.T) {
	productRepo := new(MockProductRepository)
	categoryRepo := new(MockCategoryRepository)
	handler := setupProductHandler(productRepo, categoryRepo)

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	productID := uuid.New()
	product := createTestProduct(tenantID)
	product.ID = productID

	productRepo.On("FindByIDForTenant", mock.Anything, tenantID, productID).Return(product, nil)

	router := setupTestRouter()
	router.GET("/products/:id", handler.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/products/"+productID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	productRepo.AssertExpectations(t)
}

func TestProductHandler_GetByID_NotFound(t *testing.T) {
	productRepo := new(MockProductRepository)
	categoryRepo := new(MockCategoryRepository)
	handler := setupProductHandler(productRepo, categoryRepo)

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	productID := uuid.New()

	productRepo.On("FindByIDForTenant", mock.Anything, tenantID, productID).Return(nil, shared.ErrNotFound)

	router := setupTestRouter()
	router.GET("/products/:id", handler.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/products/"+productID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	productRepo.AssertExpectations(t)
}

func TestProductHandler_GetByID_InvalidID(t *testing.T) {
	productRepo := new(MockProductRepository)
	categoryRepo := new(MockCategoryRepository)
	handler := setupProductHandler(productRepo, categoryRepo)

	router := setupTestRouter()
	router.GET("/products/:id", handler.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/products/invalid-uuid", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProductHandler_List_Success(t *testing.T) {
	productRepo := new(MockProductRepository)
	categoryRepo := new(MockCategoryRepository)
	handler := setupProductHandler(productRepo, categoryRepo)

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	product1 := createTestProduct(tenantID)
	product2 := createTestProduct(tenantID)
	product2.Code = "SKU-002"
	product2.Name = "Test Product 2"

	products := []catalog.Product{*product1, *product2}

	productRepo.On("FindAllForTenant", mock.Anything, tenantID, mock.AnythingOfType("shared.Filter")).Return(products, nil)
	productRepo.On("CountForTenant", mock.Anything, tenantID, mock.AnythingOfType("shared.Filter")).Return(int64(2), nil)

	router := setupTestRouter()
	router.GET("/products", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/products?page=1&page_size=20", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["meta"])

	productRepo.AssertExpectations(t)
}

func TestProductHandler_Update_Success(t *testing.T) {
	productRepo := new(MockProductRepository)
	categoryRepo := new(MockCategoryRepository)
	handler := setupProductHandler(productRepo, categoryRepo)

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	productID := uuid.New()
	product := createTestProduct(tenantID)
	product.ID = productID

	productRepo.On("FindByIDForTenant", mock.Anything, tenantID, productID).Return(product, nil)
	productRepo.On("Save", mock.Anything, mock.AnythingOfType("*catalog.Product")).Return(nil)

	router := setupTestRouter()
	router.PUT("/products/:id", handler.Update)

	newName := "Updated Product Name"
	reqBody := UpdateProductRequest{
		Name: &newName,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/products/"+productID.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	productRepo.AssertExpectations(t)
}

func TestProductHandler_Delete_Success(t *testing.T) {
	productRepo := new(MockProductRepository)
	categoryRepo := new(MockCategoryRepository)
	handler := setupProductHandler(productRepo, categoryRepo)

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	productID := uuid.New()
	product := createTestProduct(tenantID)
	product.ID = productID

	productRepo.On("FindByIDForTenant", mock.Anything, tenantID, productID).Return(product, nil)
	productRepo.On("DeleteForTenant", mock.Anything, tenantID, productID).Return(nil)

	router := setupTestRouter()
	router.DELETE("/products/:id", handler.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/products/"+productID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	productRepo.AssertExpectations(t)
}

func TestProductHandler_Activate_Success(t *testing.T) {
	productRepo := new(MockProductRepository)
	categoryRepo := new(MockCategoryRepository)
	handler := setupProductHandler(productRepo, categoryRepo)

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	productID := uuid.New()
	product := createTestProduct(tenantID)
	product.ID = productID
	product.Status = catalog.ProductStatusInactive

	productRepo.On("FindByIDForTenant", mock.Anything, tenantID, productID).Return(product, nil)
	productRepo.On("Save", mock.Anything, mock.AnythingOfType("*catalog.Product")).Return(nil)

	router := setupTestRouter()
	router.POST("/products/:id/activate", handler.Activate)

	req := httptest.NewRequest(http.MethodPost, "/products/"+productID.String()+"/activate", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	productRepo.AssertExpectations(t)
}

func TestProductHandler_Deactivate_Success(t *testing.T) {
	productRepo := new(MockProductRepository)
	categoryRepo := new(MockCategoryRepository)
	handler := setupProductHandler(productRepo, categoryRepo)

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	productID := uuid.New()
	product := createTestProduct(tenantID)
	product.ID = productID
	// Default status is active

	productRepo.On("FindByIDForTenant", mock.Anything, tenantID, productID).Return(product, nil)
	productRepo.On("Save", mock.Anything, mock.AnythingOfType("*catalog.Product")).Return(nil)

	router := setupTestRouter()
	router.POST("/products/:id/deactivate", handler.Deactivate)

	req := httptest.NewRequest(http.MethodPost, "/products/"+productID.String()+"/deactivate", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	productRepo.AssertExpectations(t)
}

func TestProductHandler_Discontinue_Success(t *testing.T) {
	productRepo := new(MockProductRepository)
	categoryRepo := new(MockCategoryRepository)
	handler := setupProductHandler(productRepo, categoryRepo)

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	productID := uuid.New()
	product := createTestProduct(tenantID)
	product.ID = productID

	productRepo.On("FindByIDForTenant", mock.Anything, tenantID, productID).Return(product, nil)
	productRepo.On("Save", mock.Anything, mock.AnythingOfType("*catalog.Product")).Return(nil)

	router := setupTestRouter()
	router.POST("/products/:id/discontinue", handler.Discontinue)

	req := httptest.NewRequest(http.MethodPost, "/products/"+productID.String()+"/discontinue", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	productRepo.AssertExpectations(t)
}

func TestProductHandler_CountByStatus_Success(t *testing.T) {
	productRepo := new(MockProductRepository)
	categoryRepo := new(MockCategoryRepository)
	handler := setupProductHandler(productRepo, categoryRepo)

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	productRepo.On("CountByStatus", mock.Anything, tenantID, catalog.ProductStatusActive).Return(int64(50), nil)
	productRepo.On("CountByStatus", mock.Anything, tenantID, catalog.ProductStatusInactive).Return(int64(10), nil)
	productRepo.On("CountByStatus", mock.Anything, tenantID, catalog.ProductStatusDiscontinued).Return(int64(5), nil)

	router := setupTestRouter()
	router.GET("/products/stats/count", handler.CountByStatus)

	req := httptest.NewRequest(http.MethodGet, "/products/stats/count", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(50), data["active"])
	assert.Equal(t, float64(10), data["inactive"])
	assert.Equal(t, float64(5), data["discontinued"])
	assert.Equal(t, float64(65), data["total"])

	productRepo.AssertExpectations(t)
}

func TestProductHandler_GetByCode_Success(t *testing.T) {
	productRepo := new(MockProductRepository)
	categoryRepo := new(MockCategoryRepository)
	handler := setupProductHandler(productRepo, categoryRepo)

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	product := createTestProduct(tenantID)

	productRepo.On("FindByCode", mock.Anything, tenantID, "SKU-001").Return(product, nil)

	router := setupTestRouter()
	router.GET("/products/code/:code", handler.GetByCode)

	req := httptest.NewRequest(http.MethodGet, "/products/code/SKU-001", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	productRepo.AssertExpectations(t)
}

func TestProductHandler_UpdateCode_Success(t *testing.T) {
	productRepo := new(MockProductRepository)
	categoryRepo := new(MockCategoryRepository)
	handler := setupProductHandler(productRepo, categoryRepo)

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	productID := uuid.New()
	product := createTestProduct(tenantID)
	product.ID = productID

	productRepo.On("FindByIDForTenant", mock.Anything, tenantID, productID).Return(product, nil)
	productRepo.On("ExistsByCode", mock.Anything, tenantID, "SKU-NEW").Return(false, nil)
	productRepo.On("Save", mock.Anything, mock.AnythingOfType("*catalog.Product")).Return(nil)

	router := setupTestRouter()
	router.PUT("/products/:id/code", handler.UpdateCode)

	reqBody := UpdateProductCodeRequest{
		Code: "SKU-NEW",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/products/"+productID.String()+"/code", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	productRepo.AssertExpectations(t)
}

// Suppress unused imports warnings
var (
	_ = decimal.Decimal{}
	_ = errors.New
	_ = time.Now
)
