package partner

import (
	"context"
	"testing"

	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// =============================================================================
// Mock Repositories
// =============================================================================

// MockWarehouseRepository is a mock implementation of WarehouseRepository
type MockWarehouseRepository struct {
	mock.Mock
}

func (m *MockWarehouseRepository) FindByID(ctx context.Context, id uuid.UUID) (*partner.Warehouse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*partner.Warehouse, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*partner.Warehouse, error) {
	args := m.Called(ctx, tenantID, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) FindDefault(ctx context.Context, tenantID uuid.UUID) (*partner.Warehouse, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Warehouse, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]partner.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) FindByType(ctx context.Context, tenantID uuid.UUID, warehouseType partner.WarehouseType, filter shared.Filter) ([]partner.Warehouse, error) {
	args := m.Called(ctx, tenantID, warehouseType, filter)
	return args.Get(0).([]partner.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status partner.WarehouseStatus, filter shared.Filter) ([]partner.Warehouse, error) {
	args := m.Called(ctx, tenantID, status, filter)
	return args.Get(0).([]partner.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) FindActive(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Warehouse, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]partner.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) FindAll(ctx context.Context, filter shared.Filter) ([]partner.Warehouse, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]partner.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]partner.Warehouse, error) {
	args := m.Called(ctx, tenantID, ids)
	return args.Get(0).([]partner.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) FindByCodes(ctx context.Context, tenantID uuid.UUID, codes []string) ([]partner.Warehouse, error) {
	args := m.Called(ctx, tenantID, codes)
	return args.Get(0).([]partner.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) Save(ctx context.Context, warehouse *partner.Warehouse) error {
	args := m.Called(ctx, warehouse)
	return args.Error(0)
}

func (m *MockWarehouseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockWarehouseRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *MockWarehouseRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockWarehouseRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status partner.WarehouseStatus) (int64, error) {
	args := m.Called(ctx, tenantID, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockWarehouseRepository) CountByType(ctx context.Context, tenantID uuid.UUID, warehouseType partner.WarehouseType) (int64, error) {
	args := m.Called(ctx, tenantID, warehouseType)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockWarehouseRepository) ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error) {
	args := m.Called(ctx, tenantID, code)
	return args.Bool(0), args.Error(1)
}

func (m *MockWarehouseRepository) ClearDefault(ctx context.Context, tenantID uuid.UUID) error {
	args := m.Called(ctx, tenantID)
	return args.Error(0)
}

func (m *MockWarehouseRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockWarehouseRepository) SaveBatch(ctx context.Context, warehouses []*partner.Warehouse) error {
	args := m.Called(ctx, warehouses)
	return args.Error(0)
}

// MockInventoryItemRepository is a mock implementation of InventoryItemRepository
type MockInventoryItemRepository struct {
	mock.Mock
}

func (m *MockInventoryItemRepository) FindByID(ctx context.Context, id uuid.UUID) (*inventory.InventoryItem, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*inventory.InventoryItem), args.Error(1)
}

func (m *MockInventoryItemRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*inventory.InventoryItem, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*inventory.InventoryItem), args.Error(1)
}

func (m *MockInventoryItemRepository) FindByWarehouseAndProduct(ctx context.Context, tenantID, warehouseID, productID uuid.UUID) (*inventory.InventoryItem, error) {
	args := m.Called(ctx, tenantID, warehouseID, productID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*inventory.InventoryItem), args.Error(1)
}

func (m *MockInventoryItemRepository) FindByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID, filter shared.Filter) ([]inventory.InventoryItem, error) {
	args := m.Called(ctx, tenantID, warehouseID, filter)
	return args.Get(0).([]inventory.InventoryItem), args.Error(1)
}

func (m *MockInventoryItemRepository) FindByProduct(ctx context.Context, tenantID, productID uuid.UUID, filter shared.Filter) ([]inventory.InventoryItem, error) {
	args := m.Called(ctx, tenantID, productID, filter)
	return args.Get(0).([]inventory.InventoryItem), args.Error(1)
}

func (m *MockInventoryItemRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]inventory.InventoryItem, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]inventory.InventoryItem), args.Error(1)
}

func (m *MockInventoryItemRepository) FindBelowMinimum(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]inventory.InventoryItem, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]inventory.InventoryItem), args.Error(1)
}

func (m *MockInventoryItemRepository) FindWithAvailableStock(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]inventory.InventoryItem, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]inventory.InventoryItem), args.Error(1)
}

func (m *MockInventoryItemRepository) FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]inventory.InventoryItem, error) {
	args := m.Called(ctx, tenantID, ids)
	return args.Get(0).([]inventory.InventoryItem), args.Error(1)
}

func (m *MockInventoryItemRepository) Save(ctx context.Context, item *inventory.InventoryItem) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *MockInventoryItemRepository) SaveWithLock(ctx context.Context, item *inventory.InventoryItem) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *MockInventoryItemRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockInventoryItemRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *MockInventoryItemRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockInventoryItemRepository) CountByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID, warehouseID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockInventoryItemRepository) CountByProduct(ctx context.Context, tenantID, productID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID, productID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockInventoryItemRepository) SumQuantityByProduct(ctx context.Context, tenantID, productID uuid.UUID) (decimal.Decimal, error) {
	args := m.Called(ctx, tenantID, productID)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockInventoryItemRepository) SumValueByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID) (decimal.Decimal, error) {
	args := m.Called(ctx, tenantID, warehouseID)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockInventoryItemRepository) ExistsByWarehouseAndProduct(ctx context.Context, tenantID, warehouseID, productID uuid.UUID) (bool, error) {
	args := m.Called(ctx, tenantID, warehouseID, productID)
	return args.Bool(0), args.Error(1)
}

func (m *MockInventoryItemRepository) GetOrCreate(ctx context.Context, tenantID, warehouseID, productID uuid.UUID) (*inventory.InventoryItem, error) {
	args := m.Called(ctx, tenantID, warehouseID, productID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*inventory.InventoryItem), args.Error(1)
}

// =============================================================================
// Test Helper Functions
// =============================================================================

func newWarehouseTestTenantID() uuid.UUID {
	return uuid.MustParse("11111111-1111-1111-1111-111111111111")
}

func newTestWarehouseID() uuid.UUID {
	return uuid.MustParse("33333333-3333-3333-3333-333333333333")
}

func createTestWarehouseEntity(tenantID uuid.UUID) *partner.Warehouse {
	warehouse, _ := partner.NewWarehouse(tenantID, "WH-001", "Test Warehouse", partner.WarehouseTypePhysical)
	return warehouse
}

// =============================================================================
// WarehouseService Delete Tests
// =============================================================================

func TestWarehouseService_Delete_Success_NoInventory(t *testing.T) {
	mockWarehouseRepo := new(MockWarehouseRepository)
	mockInventoryRepo := new(MockInventoryItemRepository)
	service := NewWarehouseService(mockWarehouseRepo, mockInventoryRepo)

	ctx := context.Background()
	tenantID := newWarehouseTestTenantID()
	warehouseID := newTestWarehouseID()
	warehouse := createTestWarehouseEntity(tenantID)

	mockWarehouseRepo.On("FindByIDForTenant", ctx, tenantID, warehouseID).Return(warehouse, nil)
	mockInventoryRepo.On("CountByWarehouse", ctx, tenantID, warehouseID).Return(int64(0), nil)
	mockWarehouseRepo.On("DeleteForTenant", ctx, tenantID, warehouseID).Return(nil)

	err := service.Delete(ctx, tenantID, warehouseID)

	assert.NoError(t, err)
	mockWarehouseRepo.AssertExpectations(t)
	mockInventoryRepo.AssertExpectations(t)
}

func TestWarehouseService_Delete_FailsWhenHasInventory(t *testing.T) {
	mockWarehouseRepo := new(MockWarehouseRepository)
	mockInventoryRepo := new(MockInventoryItemRepository)
	service := NewWarehouseService(mockWarehouseRepo, mockInventoryRepo)

	ctx := context.Background()
	tenantID := newWarehouseTestTenantID()
	warehouseID := newTestWarehouseID()
	warehouse := createTestWarehouseEntity(tenantID)

	mockWarehouseRepo.On("FindByIDForTenant", ctx, tenantID, warehouseID).Return(warehouse, nil)
	mockInventoryRepo.On("CountByWarehouse", ctx, tenantID, warehouseID).Return(int64(5), nil)

	err := service.Delete(ctx, tenantID, warehouseID)

	assert.Error(t, err)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "HAS_INVENTORY", domainErr.Code)
	assert.Contains(t, domainErr.Message, "5 inventory item(s)")
	mockWarehouseRepo.AssertExpectations(t)
	mockInventoryRepo.AssertExpectations(t)
}

func TestWarehouseService_Delete_FailsWhenDefaultWarehouse(t *testing.T) {
	mockWarehouseRepo := new(MockWarehouseRepository)
	mockInventoryRepo := new(MockInventoryItemRepository)
	service := NewWarehouseService(mockWarehouseRepo, mockInventoryRepo)

	ctx := context.Background()
	tenantID := newWarehouseTestTenantID()
	warehouseID := newTestWarehouseID()
	warehouse := createTestWarehouseEntity(tenantID)
	warehouse.SetDefault(true)

	mockWarehouseRepo.On("FindByIDForTenant", ctx, tenantID, warehouseID).Return(warehouse, nil)

	err := service.Delete(ctx, tenantID, warehouseID)

	assert.Error(t, err)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "CANNOT_DELETE", domainErr.Code)
	assert.Contains(t, domainErr.Message, "default warehouse")
	mockWarehouseRepo.AssertExpectations(t)
}

func TestWarehouseService_Delete_FailsWhenWarehouseNotFound(t *testing.T) {
	mockWarehouseRepo := new(MockWarehouseRepository)
	mockInventoryRepo := new(MockInventoryItemRepository)
	service := NewWarehouseService(mockWarehouseRepo, mockInventoryRepo)

	ctx := context.Background()
	tenantID := newWarehouseTestTenantID()
	warehouseID := newTestWarehouseID()

	mockWarehouseRepo.On("FindByIDForTenant", ctx, tenantID, warehouseID).Return(nil, shared.ErrNotFound)

	err := service.Delete(ctx, tenantID, warehouseID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, shared.ErrNotFound)
	mockWarehouseRepo.AssertExpectations(t)
}

func TestWarehouseService_DeleteWithOptions_ForceDelete_Success(t *testing.T) {
	mockWarehouseRepo := new(MockWarehouseRepository)
	mockInventoryRepo := new(MockInventoryItemRepository)
	service := NewWarehouseService(mockWarehouseRepo, mockInventoryRepo)

	ctx := context.Background()
	tenantID := newWarehouseTestTenantID()
	warehouseID := newTestWarehouseID()
	warehouse := createTestWarehouseEntity(tenantID)

	mockWarehouseRepo.On("FindByIDForTenant", ctx, tenantID, warehouseID).Return(warehouse, nil)
	// Note: CountByWarehouse is NOT called when force=true
	mockWarehouseRepo.On("DeleteForTenant", ctx, tenantID, warehouseID).Return(nil)

	opts := DeleteOptions{Force: true}
	err := service.DeleteWithOptions(ctx, tenantID, warehouseID, opts)

	assert.NoError(t, err)
	mockWarehouseRepo.AssertExpectations(t)
	// Inventory repo should NOT be called when force=true
	mockInventoryRepo.AssertNotCalled(t, "CountByWarehouse", mock.Anything, mock.Anything, mock.Anything)
}

func TestWarehouseService_DeleteWithOptions_ForceDeleteStillBlocksDefaultWarehouse(t *testing.T) {
	mockWarehouseRepo := new(MockWarehouseRepository)
	mockInventoryRepo := new(MockInventoryItemRepository)
	service := NewWarehouseService(mockWarehouseRepo, mockInventoryRepo)

	ctx := context.Background()
	tenantID := newWarehouseTestTenantID()
	warehouseID := newTestWarehouseID()
	warehouse := createTestWarehouseEntity(tenantID)
	warehouse.SetDefault(true)

	mockWarehouseRepo.On("FindByIDForTenant", ctx, tenantID, warehouseID).Return(warehouse, nil)

	opts := DeleteOptions{Force: true}
	err := service.DeleteWithOptions(ctx, tenantID, warehouseID, opts)

	assert.Error(t, err)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "CANNOT_DELETE", domainErr.Code)
	mockWarehouseRepo.AssertExpectations(t)
}

func TestWarehouseService_Delete_InventoryCheckError(t *testing.T) {
	mockWarehouseRepo := new(MockWarehouseRepository)
	mockInventoryRepo := new(MockInventoryItemRepository)
	service := NewWarehouseService(mockWarehouseRepo, mockInventoryRepo)

	ctx := context.Background()
	tenantID := newWarehouseTestTenantID()
	warehouseID := newTestWarehouseID()
	warehouse := createTestWarehouseEntity(tenantID)

	mockWarehouseRepo.On("FindByIDForTenant", ctx, tenantID, warehouseID).Return(warehouse, nil)
	mockInventoryRepo.On("CountByWarehouse", ctx, tenantID, warehouseID).Return(int64(0), assert.AnError)

	err := service.Delete(ctx, tenantID, warehouseID)

	assert.Error(t, err)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "INVENTORY_CHECK_FAILED", domainErr.Code)
	mockWarehouseRepo.AssertExpectations(t)
	mockInventoryRepo.AssertExpectations(t)
}

func TestWarehouseService_Delete_WithNilInventoryRepo(t *testing.T) {
	mockWarehouseRepo := new(MockWarehouseRepository)
	// Pass nil for inventory repo to test backward compatibility
	service := NewWarehouseService(mockWarehouseRepo, nil)

	ctx := context.Background()
	tenantID := newWarehouseTestTenantID()
	warehouseID := newTestWarehouseID()
	warehouse := createTestWarehouseEntity(tenantID)

	mockWarehouseRepo.On("FindByIDForTenant", ctx, tenantID, warehouseID).Return(warehouse, nil)
	mockWarehouseRepo.On("DeleteForTenant", ctx, tenantID, warehouseID).Return(nil)

	err := service.Delete(ctx, tenantID, warehouseID)

	assert.NoError(t, err)
	mockWarehouseRepo.AssertExpectations(t)
}

// =============================================================================
// WarehouseService Create Tests
// =============================================================================

func TestWarehouseService_Create_Success(t *testing.T) {
	mockWarehouseRepo := new(MockWarehouseRepository)
	mockInventoryRepo := new(MockInventoryItemRepository)
	service := NewWarehouseService(mockWarehouseRepo, mockInventoryRepo)

	ctx := context.Background()
	tenantID := newWarehouseTestTenantID()
	req := CreateWarehouseRequest{
		Code: "NEW-WH-001",
		Name: "New Warehouse",
		Type: "physical",
	}

	mockWarehouseRepo.On("ExistsByCode", ctx, tenantID, req.Code).Return(false, nil)
	mockWarehouseRepo.On("Save", ctx, mock.AnythingOfType("*partner.Warehouse")).Return(nil)

	result, err := service.Create(ctx, tenantID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "NEW-WH-001", result.Code)
	assert.Equal(t, "New Warehouse", result.Name)
	assert.Equal(t, "physical", result.Type)
	mockWarehouseRepo.AssertExpectations(t)
}

func TestWarehouseService_Create_DuplicateCode(t *testing.T) {
	mockWarehouseRepo := new(MockWarehouseRepository)
	mockInventoryRepo := new(MockInventoryItemRepository)
	service := NewWarehouseService(mockWarehouseRepo, mockInventoryRepo)

	ctx := context.Background()
	tenantID := newWarehouseTestTenantID()
	req := CreateWarehouseRequest{
		Code: "EXISTING-WH-001",
		Name: "Warehouse",
		Type: "physical",
	}

	mockWarehouseRepo.On("ExistsByCode", ctx, tenantID, req.Code).Return(true, nil)

	result, err := service.Create(ctx, tenantID, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "ALREADY_EXISTS", domainErr.Code)
	mockWarehouseRepo.AssertExpectations(t)
}

// =============================================================================
// WarehouseService GetByID Tests
// =============================================================================

func TestWarehouseService_GetByID_Success(t *testing.T) {
	mockWarehouseRepo := new(MockWarehouseRepository)
	mockInventoryRepo := new(MockInventoryItemRepository)
	service := NewWarehouseService(mockWarehouseRepo, mockInventoryRepo)

	ctx := context.Background()
	tenantID := newWarehouseTestTenantID()
	warehouseID := newTestWarehouseID()
	warehouse := createTestWarehouseEntity(tenantID)

	mockWarehouseRepo.On("FindByIDForTenant", ctx, tenantID, warehouseID).Return(warehouse, nil)

	result, err := service.GetByID(ctx, tenantID, warehouseID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, warehouse.Code, result.Code)
	mockWarehouseRepo.AssertExpectations(t)
}

func TestWarehouseService_GetByID_NotFound(t *testing.T) {
	mockWarehouseRepo := new(MockWarehouseRepository)
	mockInventoryRepo := new(MockInventoryItemRepository)
	service := NewWarehouseService(mockWarehouseRepo, mockInventoryRepo)

	ctx := context.Background()
	tenantID := newWarehouseTestTenantID()
	warehouseID := newTestWarehouseID()

	mockWarehouseRepo.On("FindByIDForTenant", ctx, tenantID, warehouseID).Return(nil, shared.ErrNotFound)

	result, err := service.GetByID(ctx, tenantID, warehouseID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, shared.ErrNotFound)
	mockWarehouseRepo.AssertExpectations(t)
}
