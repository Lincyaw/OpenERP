package catalog

import (
	"context"
	"testing"

	"github.com/erp/backend/internal/domain/catalog"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockProductRepository is a mock implementation of ProductRepository
type MockProductRepository struct {
	mock.Mock
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

// MockCategoryRepository is a mock implementation of CategoryRepository
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

func (m *MockCategoryRepository) FindAll(ctx context.Context, filter shared.Filter) ([]catalog.Category, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]catalog.Category), args.Error(1)
}

func (m *MockCategoryRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]catalog.Category, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]catalog.Category), args.Error(1)
}

func (m *MockCategoryRepository) FindChildren(ctx context.Context, tenantID, parentID uuid.UUID) ([]catalog.Category, error) {
	args := m.Called(ctx, tenantID, parentID)
	return args.Get(0).([]catalog.Category), args.Error(1)
}

func (m *MockCategoryRepository) FindRootCategories(ctx context.Context, tenantID uuid.UUID) ([]catalog.Category, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]catalog.Category), args.Error(1)
}

func (m *MockCategoryRepository) FindDescendants(ctx context.Context, tenantID, id uuid.UUID) ([]catalog.Category, error) {
	args := m.Called(ctx, tenantID, id)
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

func (m *MockCategoryRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCategoryRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCategoryRepository) ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error) {
	args := m.Called(ctx, tenantID, code)
	return args.Bool(0), args.Error(1)
}

func (m *MockCategoryRepository) HasChildren(ctx context.Context, tenantID, categoryID uuid.UUID) (bool, error) {
	args := m.Called(ctx, tenantID, categoryID)
	return args.Bool(0), args.Error(1)
}

func (m *MockCategoryRepository) HasProducts(ctx context.Context, tenantID, categoryID uuid.UUID) (bool, error) {
	args := m.Called(ctx, tenantID, categoryID)
	return args.Bool(0), args.Error(1)
}

func (m *MockCategoryRepository) UpdatePath(ctx context.Context, tenantID, categoryID uuid.UUID, newPath string, levelDelta int) error {
	args := m.Called(ctx, tenantID, categoryID, newPath, levelDelta)
	return args.Error(0)
}

func (m *MockCategoryRepository) FindByPath(ctx context.Context, tenantID uuid.UUID, path string) (*catalog.Category, error) {
	args := m.Called(ctx, tenantID, path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*catalog.Category), args.Error(1)
}

// Test helper functions
func newTestTenantID() uuid.UUID {
	return uuid.MustParse("11111111-1111-1111-1111-111111111111")
}

func newTestProductID() uuid.UUID {
	return uuid.MustParse("22222222-2222-2222-2222-222222222222")
}

func newTestCategoryID() uuid.UUID {
	return uuid.MustParse("33333333-3333-3333-3333-333333333333")
}

func createTestProduct(tenantID uuid.UUID) *catalog.Product {
	product, _ := catalog.NewProduct(tenantID, "TEST-001", "Test Product", "pcs")
	return product
}

// Tests for ProductService.Create
func TestProductService_Create_Success(t *testing.T) {
	mockProductRepo := new(MockProductRepository)
	mockCategoryRepo := new(MockCategoryRepository)
	service := NewProductService(mockProductRepo, mockCategoryRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	req := CreateProductRequest{
		Code: "NEW-001",
		Name: "New Product",
		Unit: "pcs",
	}

	mockProductRepo.On("ExistsByCode", ctx, tenantID, req.Code).Return(false, nil)
	mockProductRepo.On("Save", ctx, mock.AnythingOfType("*catalog.Product")).Return(nil)

	result, err := service.Create(ctx, tenantID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "NEW-001", result.Code)
	assert.Equal(t, "New Product", result.Name)
	assert.Equal(t, "pcs", result.Unit)
	assert.Equal(t, "active", result.Status)
	mockProductRepo.AssertExpectations(t)
}

func TestProductService_Create_WithAllFields(t *testing.T) {
	mockProductRepo := new(MockProductRepository)
	mockCategoryRepo := new(MockCategoryRepository)
	service := NewProductService(mockProductRepo, mockCategoryRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	categoryID := newTestCategoryID()
	purchasePrice := decimal.NewFromFloat(50.00)
	sellingPrice := decimal.NewFromFloat(100.00)
	minStock := decimal.NewFromFloat(10)
	sortOrder := 5

	req := CreateProductRequest{
		Code:          "FULL-001",
		Name:          "Full Product",
		Description:   "A product with all fields",
		Barcode:       "1234567890123",
		CategoryID:    &categoryID,
		Unit:          "pcs",
		PurchasePrice: &purchasePrice,
		SellingPrice:  &sellingPrice,
		MinStock:      &minStock,
		SortOrder:     &sortOrder,
		Attributes:    `{"color": "red"}`,
	}

	category, _ := catalog.NewCategory(tenantID, "CAT-001", "Test Category")

	mockProductRepo.On("ExistsByCode", ctx, tenantID, req.Code).Return(false, nil)
	mockProductRepo.On("ExistsByBarcode", ctx, tenantID, req.Barcode).Return(false, nil)
	mockCategoryRepo.On("FindByIDForTenant", ctx, tenantID, categoryID).Return(category, nil)
	mockProductRepo.On("Save", ctx, mock.AnythingOfType("*catalog.Product")).Return(nil)

	result, err := service.Create(ctx, tenantID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "FULL-001", result.Code)
	assert.Equal(t, "Full Product", result.Name)
	assert.Equal(t, "A product with all fields", result.Description)
	assert.Equal(t, "1234567890123", result.Barcode)
	assert.Equal(t, &categoryID, result.CategoryID)
	assert.True(t, result.PurchasePrice.Equal(purchasePrice))
	assert.True(t, result.SellingPrice.Equal(sellingPrice))
	assert.True(t, result.MinStock.Equal(minStock))
	assert.Equal(t, sortOrder, result.SortOrder)
	mockProductRepo.AssertExpectations(t)
	mockCategoryRepo.AssertExpectations(t)
}

func TestProductService_Create_DuplicateCode(t *testing.T) {
	mockProductRepo := new(MockProductRepository)
	mockCategoryRepo := new(MockCategoryRepository)
	service := NewProductService(mockProductRepo, mockCategoryRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	req := CreateProductRequest{
		Code: "EXISTING-001",
		Name: "New Product",
		Unit: "pcs",
	}

	mockProductRepo.On("ExistsByCode", ctx, tenantID, req.Code).Return(true, nil)

	result, err := service.Create(ctx, tenantID, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "ALREADY_EXISTS", domainErr.Code)
	mockProductRepo.AssertExpectations(t)
}

func TestProductService_Create_DuplicateBarcode(t *testing.T) {
	mockProductRepo := new(MockProductRepository)
	mockCategoryRepo := new(MockCategoryRepository)
	service := NewProductService(mockProductRepo, mockCategoryRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	req := CreateProductRequest{
		Code:    "NEW-001",
		Name:    "New Product",
		Unit:    "pcs",
		Barcode: "EXISTING-BARCODE",
	}

	mockProductRepo.On("ExistsByCode", ctx, tenantID, req.Code).Return(false, nil)
	mockProductRepo.On("ExistsByBarcode", ctx, tenantID, req.Barcode).Return(true, nil)

	result, err := service.Create(ctx, tenantID, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "ALREADY_EXISTS", domainErr.Code)
	mockProductRepo.AssertExpectations(t)
}

func TestProductService_Create_InvalidCategory(t *testing.T) {
	mockProductRepo := new(MockProductRepository)
	mockCategoryRepo := new(MockCategoryRepository)
	service := NewProductService(mockProductRepo, mockCategoryRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	invalidCategoryID := uuid.New()
	req := CreateProductRequest{
		Code:       "NEW-001",
		Name:       "New Product",
		Unit:       "pcs",
		CategoryID: &invalidCategoryID,
	}

	mockProductRepo.On("ExistsByCode", ctx, tenantID, req.Code).Return(false, nil)
	mockCategoryRepo.On("FindByIDForTenant", ctx, tenantID, invalidCategoryID).Return(nil, shared.ErrNotFound)

	result, err := service.Create(ctx, tenantID, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "INVALID_CATEGORY", domainErr.Code)
	mockProductRepo.AssertExpectations(t)
	mockCategoryRepo.AssertExpectations(t)
}

// Tests for ProductService.GetByID
func TestProductService_GetByID_Success(t *testing.T) {
	mockProductRepo := new(MockProductRepository)
	mockCategoryRepo := new(MockCategoryRepository)
	service := NewProductService(mockProductRepo, mockCategoryRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	productID := newTestProductID()
	product := createTestProduct(tenantID)

	mockProductRepo.On("FindByIDForTenant", ctx, tenantID, productID).Return(product, nil)

	result, err := service.GetByID(ctx, tenantID, productID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, product.Code, result.Code)
	mockProductRepo.AssertExpectations(t)
}

func TestProductService_GetByID_NotFound(t *testing.T) {
	mockProductRepo := new(MockProductRepository)
	mockCategoryRepo := new(MockCategoryRepository)
	service := NewProductService(mockProductRepo, mockCategoryRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	productID := newTestProductID()

	mockProductRepo.On("FindByIDForTenant", ctx, tenantID, productID).Return(nil, shared.ErrNotFound)

	result, err := service.GetByID(ctx, tenantID, productID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, shared.ErrNotFound)
	mockProductRepo.AssertExpectations(t)
}

// Tests for ProductService.List
func TestProductService_List_Success(t *testing.T) {
	mockProductRepo := new(MockProductRepository)
	mockCategoryRepo := new(MockCategoryRepository)
	service := NewProductService(mockProductRepo, mockCategoryRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	filter := ProductListFilter{
		Page:     1,
		PageSize: 10,
	}

	products := []catalog.Product{
		*createTestProduct(tenantID),
	}

	mockProductRepo.On("FindAllForTenant", ctx, tenantID, mock.AnythingOfType("shared.Filter")).Return(products, nil)
	mockProductRepo.On("CountForTenant", ctx, tenantID, mock.AnythingOfType("shared.Filter")).Return(int64(1), nil)

	result, total, err := service.List(ctx, tenantID, filter)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, int64(1), total)
	mockProductRepo.AssertExpectations(t)
}

func TestProductService_List_WithFilters(t *testing.T) {
	mockProductRepo := new(MockProductRepository)
	mockCategoryRepo := new(MockCategoryRepository)
	service := NewProductService(mockProductRepo, mockCategoryRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	categoryID := newTestCategoryID()
	minPrice := 10.0
	maxPrice := 100.0
	hasBarcode := true

	filter := ProductListFilter{
		Search:     "test",
		Status:     "active",
		CategoryID: &categoryID,
		Unit:       "pcs",
		MinPrice:   &minPrice,
		MaxPrice:   &maxPrice,
		HasBarcode: &hasBarcode,
		Page:       1,
		PageSize:   20,
		OrderBy:    "name",
		OrderDir:   "asc",
	}

	products := []catalog.Product{}

	mockProductRepo.On("FindAllForTenant", ctx, tenantID, mock.AnythingOfType("shared.Filter")).Return(products, nil)
	mockProductRepo.On("CountForTenant", ctx, tenantID, mock.AnythingOfType("shared.Filter")).Return(int64(0), nil)

	result, total, err := service.List(ctx, tenantID, filter)

	assert.NoError(t, err)
	assert.Len(t, result, 0)
	assert.Equal(t, int64(0), total)
	mockProductRepo.AssertExpectations(t)
}

// Tests for ProductService.Update
func TestProductService_Update_Success(t *testing.T) {
	mockProductRepo := new(MockProductRepository)
	mockCategoryRepo := new(MockCategoryRepository)
	service := NewProductService(mockProductRepo, mockCategoryRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	productID := newTestProductID()
	product := createTestProduct(tenantID)

	newName := "Updated Name"
	newDescription := "Updated Description"
	req := UpdateProductRequest{
		Name:        &newName,
		Description: &newDescription,
	}

	mockProductRepo.On("FindByIDForTenant", ctx, tenantID, productID).Return(product, nil)
	mockProductRepo.On("Save", ctx, mock.AnythingOfType("*catalog.Product")).Return(nil)

	result, err := service.Update(ctx, tenantID, productID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, newName, result.Name)
	assert.Equal(t, newDescription, result.Description)
	mockProductRepo.AssertExpectations(t)
}

func TestProductService_Update_NotFound(t *testing.T) {
	mockProductRepo := new(MockProductRepository)
	mockCategoryRepo := new(MockCategoryRepository)
	service := NewProductService(mockProductRepo, mockCategoryRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	productID := newTestProductID()

	newName := "Updated Name"
	req := UpdateProductRequest{
		Name: &newName,
	}

	mockProductRepo.On("FindByIDForTenant", ctx, tenantID, productID).Return(nil, shared.ErrNotFound)

	result, err := service.Update(ctx, tenantID, productID, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, shared.ErrNotFound)
	mockProductRepo.AssertExpectations(t)
}

// Tests for ProductService.UpdateCode
func TestProductService_UpdateCode_Success(t *testing.T) {
	mockProductRepo := new(MockProductRepository)
	mockCategoryRepo := new(MockCategoryRepository)
	service := NewProductService(mockProductRepo, mockCategoryRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	productID := newTestProductID()
	product := createTestProduct(tenantID)
	newCode := "NEW-CODE-001"

	mockProductRepo.On("FindByIDForTenant", ctx, tenantID, productID).Return(product, nil)
	mockProductRepo.On("ExistsByCode", ctx, tenantID, newCode).Return(false, nil)
	mockProductRepo.On("Save", ctx, mock.AnythingOfType("*catalog.Product")).Return(nil)

	result, err := service.UpdateCode(ctx, tenantID, productID, newCode)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "NEW-CODE-001", result.Code)
	mockProductRepo.AssertExpectations(t)
}

func TestProductService_UpdateCode_DuplicateCode(t *testing.T) {
	mockProductRepo := new(MockProductRepository)
	mockCategoryRepo := new(MockCategoryRepository)
	service := NewProductService(mockProductRepo, mockCategoryRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	productID := newTestProductID()
	product := createTestProduct(tenantID)
	newCode := "EXISTING-CODE"

	mockProductRepo.On("FindByIDForTenant", ctx, tenantID, productID).Return(product, nil)
	mockProductRepo.On("ExistsByCode", ctx, tenantID, newCode).Return(true, nil)

	result, err := service.UpdateCode(ctx, tenantID, productID, newCode)

	assert.Error(t, err)
	assert.Nil(t, result)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "ALREADY_EXISTS", domainErr.Code)
	mockProductRepo.AssertExpectations(t)
}

// Tests for ProductService.Delete
func TestProductService_Delete_Success(t *testing.T) {
	mockProductRepo := new(MockProductRepository)
	mockCategoryRepo := new(MockCategoryRepository)
	service := NewProductService(mockProductRepo, mockCategoryRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	productID := newTestProductID()
	product := createTestProduct(tenantID)

	mockProductRepo.On("FindByIDForTenant", ctx, tenantID, productID).Return(product, nil)
	mockProductRepo.On("DeleteForTenant", ctx, tenantID, productID).Return(nil)

	err := service.Delete(ctx, tenantID, productID)

	assert.NoError(t, err)
	mockProductRepo.AssertExpectations(t)
}

func TestProductService_Delete_NotFound(t *testing.T) {
	mockProductRepo := new(MockProductRepository)
	mockCategoryRepo := new(MockCategoryRepository)
	service := NewProductService(mockProductRepo, mockCategoryRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	productID := newTestProductID()

	mockProductRepo.On("FindByIDForTenant", ctx, tenantID, productID).Return(nil, shared.ErrNotFound)

	err := service.Delete(ctx, tenantID, productID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, shared.ErrNotFound)
	mockProductRepo.AssertExpectations(t)
}

// Tests for ProductService.Activate
func TestProductService_Activate_Success(t *testing.T) {
	mockProductRepo := new(MockProductRepository)
	mockCategoryRepo := new(MockCategoryRepository)
	service := NewProductService(mockProductRepo, mockCategoryRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	productID := newTestProductID()
	product := createTestProduct(tenantID)
	product.Deactivate()

	mockProductRepo.On("FindByIDForTenant", ctx, tenantID, productID).Return(product, nil)
	mockProductRepo.On("Save", ctx, mock.AnythingOfType("*catalog.Product")).Return(nil)

	result, err := service.Activate(ctx, tenantID, productID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "active", result.Status)
	mockProductRepo.AssertExpectations(t)
}

func TestProductService_Activate_AlreadyActive(t *testing.T) {
	mockProductRepo := new(MockProductRepository)
	mockCategoryRepo := new(MockCategoryRepository)
	service := NewProductService(mockProductRepo, mockCategoryRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	productID := newTestProductID()
	product := createTestProduct(tenantID) // Already active by default

	mockProductRepo.On("FindByIDForTenant", ctx, tenantID, productID).Return(product, nil)

	result, err := service.Activate(ctx, tenantID, productID)

	assert.Error(t, err)
	assert.Nil(t, result)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "ALREADY_ACTIVE", domainErr.Code)
	mockProductRepo.AssertExpectations(t)
}

// Tests for ProductService.Deactivate
func TestProductService_Deactivate_Success(t *testing.T) {
	mockProductRepo := new(MockProductRepository)
	mockCategoryRepo := new(MockCategoryRepository)
	service := NewProductService(mockProductRepo, mockCategoryRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	productID := newTestProductID()
	product := createTestProduct(tenantID) // Active by default

	mockProductRepo.On("FindByIDForTenant", ctx, tenantID, productID).Return(product, nil)
	mockProductRepo.On("Save", ctx, mock.AnythingOfType("*catalog.Product")).Return(nil)

	result, err := service.Deactivate(ctx, tenantID, productID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "inactive", result.Status)
	mockProductRepo.AssertExpectations(t)
}

// Tests for ProductService.Discontinue
func TestProductService_Discontinue_Success(t *testing.T) {
	mockProductRepo := new(MockProductRepository)
	mockCategoryRepo := new(MockCategoryRepository)
	service := NewProductService(mockProductRepo, mockCategoryRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	productID := newTestProductID()
	product := createTestProduct(tenantID)

	mockProductRepo.On("FindByIDForTenant", ctx, tenantID, productID).Return(product, nil)
	mockProductRepo.On("Save", ctx, mock.AnythingOfType("*catalog.Product")).Return(nil)

	result, err := service.Discontinue(ctx, tenantID, productID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "discontinued", result.Status)
	mockProductRepo.AssertExpectations(t)
}

func TestProductService_Discontinue_CannotActivateAfter(t *testing.T) {
	mockProductRepo := new(MockProductRepository)
	mockCategoryRepo := new(MockCategoryRepository)
	service := NewProductService(mockProductRepo, mockCategoryRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	productID := newTestProductID()
	product := createTestProduct(tenantID)
	product.Discontinue()

	mockProductRepo.On("FindByIDForTenant", ctx, tenantID, productID).Return(product, nil)

	result, err := service.Activate(ctx, tenantID, productID)

	assert.Error(t, err)
	assert.Nil(t, result)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "CANNOT_ACTIVATE", domainErr.Code)
	mockProductRepo.AssertExpectations(t)
}

// Tests for ProductService.CountByStatus
func TestProductService_CountByStatus_Success(t *testing.T) {
	mockProductRepo := new(MockProductRepository)
	mockCategoryRepo := new(MockCategoryRepository)
	service := NewProductService(mockProductRepo, mockCategoryRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()

	mockProductRepo.On("CountByStatus", ctx, tenantID, catalog.ProductStatusActive).Return(int64(10), nil)
	mockProductRepo.On("CountByStatus", ctx, tenantID, catalog.ProductStatusInactive).Return(int64(5), nil)
	mockProductRepo.On("CountByStatus", ctx, tenantID, catalog.ProductStatusDiscontinued).Return(int64(2), nil)

	counts, err := service.CountByStatus(ctx, tenantID)

	assert.NoError(t, err)
	assert.NotNil(t, counts)
	assert.Equal(t, int64(10), counts["active"])
	assert.Equal(t, int64(5), counts["inactive"])
	assert.Equal(t, int64(2), counts["discontinued"])
	assert.Equal(t, int64(17), counts["total"])
	mockProductRepo.AssertExpectations(t)
}

// Tests for ToProductResponse and ToProductListResponse
func TestToProductResponse(t *testing.T) {
	tenantID := newTestTenantID()
	product := createTestProduct(tenantID)

	result := ToProductResponse(product)

	assert.Equal(t, product.ID, result.ID)
	assert.Equal(t, product.TenantID, result.TenantID)
	assert.Equal(t, product.Code, result.Code)
	assert.Equal(t, product.Name, result.Name)
	assert.Equal(t, product.Unit, result.Unit)
	assert.Equal(t, string(product.Status), result.Status)
}

func TestToProductListResponses(t *testing.T) {
	tenantID := newTestTenantID()
	products := []catalog.Product{
		*createTestProduct(tenantID),
		*createTestProduct(tenantID),
	}

	results := ToProductListResponses(products)

	assert.Len(t, results, 2)
	assert.Equal(t, products[0].Code, results[0].Code)
	assert.Equal(t, products[1].Code, results[1].Code)
}
