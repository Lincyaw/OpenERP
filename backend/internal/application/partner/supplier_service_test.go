package partner

import (
	"context"
	"errors"
	"testing"

	"github.com/erp/backend/internal/domain/finance"
	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// =============================================================================
// Mock Repositories
// =============================================================================

// MockSupplierRepository is a mock implementation of SupplierRepository
type MockSupplierRepository struct {
	mock.Mock
}

func (m *MockSupplierRepository) FindByID(ctx context.Context, id uuid.UUID) (*partner.Supplier, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*partner.Supplier, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*partner.Supplier, error) {
	args := m.Called(ctx, tenantID, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) FindByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (*partner.Supplier, error) {
	args := m.Called(ctx, tenantID, phone)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) FindByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*partner.Supplier, error) {
	args := m.Called(ctx, tenantID, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) FindAll(ctx context.Context, filter shared.Filter) ([]partner.Supplier, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Supplier, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) FindByType(ctx context.Context, tenantID uuid.UUID, supplierType partner.SupplierType, filter shared.Filter) ([]partner.Supplier, error) {
	args := m.Called(ctx, tenantID, supplierType, filter)
	return args.Get(0).([]partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status partner.SupplierStatus, filter shared.Filter) ([]partner.Supplier, error) {
	args := m.Called(ctx, tenantID, status, filter)
	return args.Get(0).([]partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) FindActive(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Supplier, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]partner.Supplier, error) {
	args := m.Called(ctx, tenantID, ids)
	return args.Get(0).([]partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) FindByCodes(ctx context.Context, tenantID uuid.UUID, codes []string) ([]partner.Supplier, error) {
	args := m.Called(ctx, tenantID, codes)
	return args.Get(0).([]partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) FindWithOutstandingBalance(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Supplier, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) FindOverCreditLimit(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Supplier, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) Save(ctx context.Context, supplier *partner.Supplier) error {
	args := m.Called(ctx, supplier)
	return args.Error(0)
}

func (m *MockSupplierRepository) SaveBatch(ctx context.Context, suppliers []*partner.Supplier) error {
	args := m.Called(ctx, suppliers)
	return args.Error(0)
}

func (m *MockSupplierRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSupplierRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *MockSupplierRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSupplierRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSupplierRepository) CountByType(ctx context.Context, tenantID uuid.UUID, supplierType partner.SupplierType) (int64, error) {
	args := m.Called(ctx, tenantID, supplierType)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSupplierRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status partner.SupplierStatus) (int64, error) {
	args := m.Called(ctx, tenantID, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSupplierRepository) ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error) {
	args := m.Called(ctx, tenantID, code)
	return args.Bool(0), args.Error(1)
}

func (m *MockSupplierRepository) ExistsByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (bool, error) {
	args := m.Called(ctx, tenantID, phone)
	return args.Bool(0), args.Error(1)
}

func (m *MockSupplierRepository) ExistsByEmail(ctx context.Context, tenantID uuid.UUID, email string) (bool, error) {
	args := m.Called(ctx, tenantID, email)
	return args.Bool(0), args.Error(1)
}

// Verify interface compliance
var _ partner.SupplierRepository = (*MockSupplierRepository)(nil)

// MockAccountPayableRepositoryForSupplier is a mock implementation of AccountPayableRepository
// Only the methods needed for supplier delete validation are implemented
type MockAccountPayableRepositoryForSupplier struct {
	mock.Mock
}

func (m *MockAccountPayableRepositoryForSupplier) FindByID(ctx context.Context, id uuid.UUID) (*finance.AccountPayable, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.AccountPayable), args.Error(1)
}

func (m *MockAccountPayableRepositoryForSupplier) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*finance.AccountPayable, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.AccountPayable), args.Error(1)
}

func (m *MockAccountPayableRepositoryForSupplier) FindByPayableNumber(ctx context.Context, tenantID uuid.UUID, payableNumber string) (*finance.AccountPayable, error) {
	args := m.Called(ctx, tenantID, payableNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.AccountPayable), args.Error(1)
}

func (m *MockAccountPayableRepositoryForSupplier) FindBySource(ctx context.Context, tenantID uuid.UUID, sourceType finance.PayableSourceType, sourceID uuid.UUID) (*finance.AccountPayable, error) {
	args := m.Called(ctx, tenantID, sourceType, sourceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.AccountPayable), args.Error(1)
}

func (m *MockAccountPayableRepositoryForSupplier) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter finance.AccountPayableFilter) ([]finance.AccountPayable, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]finance.AccountPayable), args.Error(1)
}

func (m *MockAccountPayableRepositoryForSupplier) FindBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID, filter finance.AccountPayableFilter) ([]finance.AccountPayable, error) {
	args := m.Called(ctx, tenantID, supplierID, filter)
	return args.Get(0).([]finance.AccountPayable), args.Error(1)
}

func (m *MockAccountPayableRepositoryForSupplier) FindByStatus(ctx context.Context, tenantID uuid.UUID, status finance.PayableStatus, filter finance.AccountPayableFilter) ([]finance.AccountPayable, error) {
	args := m.Called(ctx, tenantID, status, filter)
	return args.Get(0).([]finance.AccountPayable), args.Error(1)
}

func (m *MockAccountPayableRepositoryForSupplier) FindOutstanding(ctx context.Context, tenantID, supplierID uuid.UUID) ([]finance.AccountPayable, error) {
	args := m.Called(ctx, tenantID, supplierID)
	return args.Get(0).([]finance.AccountPayable), args.Error(1)
}

func (m *MockAccountPayableRepositoryForSupplier) FindOverdue(ctx context.Context, tenantID uuid.UUID, filter finance.AccountPayableFilter) ([]finance.AccountPayable, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]finance.AccountPayable), args.Error(1)
}

func (m *MockAccountPayableRepositoryForSupplier) Save(ctx context.Context, payable *finance.AccountPayable) error {
	args := m.Called(ctx, payable)
	return args.Error(0)
}

func (m *MockAccountPayableRepositoryForSupplier) SaveWithLock(ctx context.Context, payable *finance.AccountPayable) error {
	args := m.Called(ctx, payable)
	return args.Error(0)
}

func (m *MockAccountPayableRepositoryForSupplier) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAccountPayableRepositoryForSupplier) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *MockAccountPayableRepositoryForSupplier) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter finance.AccountPayableFilter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAccountPayableRepositoryForSupplier) CountByStatus(ctx context.Context, tenantID uuid.UUID, status finance.PayableStatus) (int64, error) {
	args := m.Called(ctx, tenantID, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAccountPayableRepositoryForSupplier) CountBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID, supplierID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAccountPayableRepositoryForSupplier) CountOverdue(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAccountPayableRepositoryForSupplier) CountOutstandingBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID, supplierID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAccountPayableRepositoryForSupplier) SumOutstandingBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (decimal.Decimal, error) {
	args := m.Called(ctx, tenantID, supplierID)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockAccountPayableRepositoryForSupplier) SumOutstandingForTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockAccountPayableRepositoryForSupplier) SumOverdueForTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockAccountPayableRepositoryForSupplier) ExistsByPayableNumber(ctx context.Context, tenantID uuid.UUID, payableNumber string) (bool, error) {
	args := m.Called(ctx, tenantID, payableNumber)
	return args.Bool(0), args.Error(1)
}

func (m *MockAccountPayableRepositoryForSupplier) ExistsBySource(ctx context.Context, tenantID uuid.UUID, sourceType finance.PayableSourceType, sourceID uuid.UUID) (bool, error) {
	args := m.Called(ctx, tenantID, sourceType, sourceID)
	return args.Bool(0), args.Error(1)
}

func (m *MockAccountPayableRepositoryForSupplier) GeneratePayableNumber(ctx context.Context, tenantID uuid.UUID) (string, error) {
	args := m.Called(ctx, tenantID)
	return args.String(0), args.Error(1)
}

// Verify interface compliance
var _ finance.AccountPayableRepository = (*MockAccountPayableRepositoryForSupplier)(nil)

// MockPurchaseOrderRepositoryForSupplier is a mock implementation of PurchaseOrderRepository
// Only the methods needed for supplier delete validation are implemented
type MockPurchaseOrderRepositoryForSupplier struct {
	mock.Mock
}

func (m *MockPurchaseOrderRepositoryForSupplier) FindByID(ctx context.Context, id uuid.UUID) (*trade.PurchaseOrder, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*trade.PurchaseOrder), args.Error(1)
}

func (m *MockPurchaseOrderRepositoryForSupplier) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*trade.PurchaseOrder, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*trade.PurchaseOrder), args.Error(1)
}

func (m *MockPurchaseOrderRepositoryForSupplier) FindByOrderNumber(ctx context.Context, tenantID uuid.UUID, orderNumber string) (*trade.PurchaseOrder, error) {
	args := m.Called(ctx, tenantID, orderNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*trade.PurchaseOrder), args.Error(1)
}

func (m *MockPurchaseOrderRepositoryForSupplier) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]trade.PurchaseOrder, error) {
	args := m.Called(ctx, tenantID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.PurchaseOrder), args.Error(1)
}

func (m *MockPurchaseOrderRepositoryForSupplier) FindBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID, filter shared.Filter) ([]trade.PurchaseOrder, error) {
	args := m.Called(ctx, tenantID, supplierID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.PurchaseOrder), args.Error(1)
}

func (m *MockPurchaseOrderRepositoryForSupplier) FindByStatus(ctx context.Context, tenantID uuid.UUID, status trade.PurchaseOrderStatus, filter shared.Filter) ([]trade.PurchaseOrder, error) {
	args := m.Called(ctx, tenantID, status, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.PurchaseOrder), args.Error(1)
}

func (m *MockPurchaseOrderRepositoryForSupplier) FindByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID, filter shared.Filter) ([]trade.PurchaseOrder, error) {
	args := m.Called(ctx, tenantID, warehouseID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.PurchaseOrder), args.Error(1)
}

func (m *MockPurchaseOrderRepositoryForSupplier) FindPendingReceipt(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]trade.PurchaseOrder, error) {
	args := m.Called(ctx, tenantID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.PurchaseOrder), args.Error(1)
}

func (m *MockPurchaseOrderRepositoryForSupplier) Save(ctx context.Context, order *trade.PurchaseOrder) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockPurchaseOrderRepositoryForSupplier) SaveWithLock(ctx context.Context, order *trade.PurchaseOrder) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockPurchaseOrderRepositoryForSupplier) SaveWithLockAndEvents(ctx context.Context, order *trade.PurchaseOrder, events []shared.DomainEvent) error {
	args := m.Called(ctx, order, events)
	return args.Error(0)
}

func (m *MockPurchaseOrderRepositoryForSupplier) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPurchaseOrderRepositoryForSupplier) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *MockPurchaseOrderRepositoryForSupplier) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPurchaseOrderRepositoryForSupplier) CountByStatus(ctx context.Context, tenantID uuid.UUID, status trade.PurchaseOrderStatus) (int64, error) {
	args := m.Called(ctx, tenantID, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPurchaseOrderRepositoryForSupplier) CountBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID, supplierID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPurchaseOrderRepositoryForSupplier) CountIncompleteBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID, supplierID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPurchaseOrderRepositoryForSupplier) CountPendingReceipt(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPurchaseOrderRepositoryForSupplier) ExistsByOrderNumber(ctx context.Context, tenantID uuid.UUID, orderNumber string) (bool, error) {
	args := m.Called(ctx, tenantID, orderNumber)
	return args.Bool(0), args.Error(1)
}

func (m *MockPurchaseOrderRepositoryForSupplier) GenerateOrderNumber(ctx context.Context, tenantID uuid.UUID) (string, error) {
	args := m.Called(ctx, tenantID)
	return args.String(0), args.Error(1)
}

func (m *MockPurchaseOrderRepositoryForSupplier) ExistsByProduct(ctx context.Context, tenantID, productID uuid.UUID) (bool, error) {
	args := m.Called(ctx, tenantID, productID)
	return args.Bool(0), args.Error(1)
}

// Verify interface compliance
var _ trade.PurchaseOrderRepository = (*MockPurchaseOrderRepositoryForSupplier)(nil)

// =============================================================================
// Test Helper Functions
// =============================================================================

func newSupplierTestTenantID() uuid.UUID {
	return uuid.MustParse("11111111-1111-1111-1111-111111111111")
}

func newSupplierTestSupplierID() uuid.UUID {
	return uuid.MustParse("33333333-3333-3333-3333-333333333333")
}

func createTestSupplier(tenantID uuid.UUID) *partner.Supplier {
	supplier, _ := partner.NewSupplier(tenantID, "SUP-001", "Test Supplier", partner.SupplierTypeManufacturer)
	return supplier
}

// =============================================================================
// SupplierService Delete Tests
// =============================================================================

func TestSupplierService_Delete_Success(t *testing.T) {
	mockRepo := new(MockSupplierRepository)
	service := NewSupplierService(mockRepo)

	ctx := context.Background()
	tenantID := newSupplierTestTenantID()
	supplierID := newSupplierTestSupplierID()
	supplier := createTestSupplier(tenantID)

	mockRepo.On("FindByIDForTenant", ctx, tenantID, supplierID).Return(supplier, nil)
	mockRepo.On("DeleteForTenant", ctx, tenantID, supplierID).Return(nil)

	err := service.Delete(ctx, tenantID, supplierID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSupplierService_Delete_HasBalance(t *testing.T) {
	mockRepo := new(MockSupplierRepository)
	service := NewSupplierService(mockRepo)

	ctx := context.Background()
	tenantID := newSupplierTestTenantID()
	supplierID := newSupplierTestSupplierID()
	supplier := createTestSupplier(tenantID)
	supplier.AddBalance(decimal.NewFromFloat(100.00)) // Add prepaid balance

	mockRepo.On("FindByIDForTenant", ctx, tenantID, supplierID).Return(supplier, nil)

	err := service.Delete(ctx, tenantID, supplierID)

	assert.Error(t, err)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "HAS_BALANCE", domainErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestSupplierService_Delete_HasOutstandingPayables(t *testing.T) {
	mockRepo := new(MockSupplierRepository)
	mockPayableRepo := new(MockAccountPayableRepositoryForSupplier)
	service := NewSupplierService(mockRepo)
	service.SetAccountPayableRepo(mockPayableRepo)

	ctx := context.Background()
	tenantID := newSupplierTestTenantID()
	supplierID := newSupplierTestSupplierID()
	supplier := createTestSupplier(tenantID)

	mockRepo.On("FindByIDForTenant", ctx, tenantID, supplierID).Return(supplier, nil)
	mockPayableRepo.On("CountOutstandingBySupplier", ctx, tenantID, supplierID).Return(int64(3), nil)
	mockPayableRepo.On("SumOutstandingBySupplier", ctx, tenantID, supplierID).Return(decimal.NewFromFloat(1500.00), nil)

	err := service.Delete(ctx, tenantID, supplierID)

	assert.Error(t, err)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "HAS_OUTSTANDING_PAYABLES", domainErr.Code)
	assert.Contains(t, domainErr.Message, "3 outstanding payable")
	mockRepo.AssertExpectations(t)
	mockPayableRepo.AssertExpectations(t)
}

func TestSupplierService_Delete_HasIncompleteOrders(t *testing.T) {
	mockRepo := new(MockSupplierRepository)
	mockPayableRepo := new(MockAccountPayableRepositoryForSupplier)
	mockOrderRepo := new(MockPurchaseOrderRepositoryForSupplier)
	service := NewSupplierService(mockRepo)
	service.SetAccountPayableRepo(mockPayableRepo)
	service.SetPurchaseOrderRepo(mockOrderRepo)

	ctx := context.Background()
	tenantID := newSupplierTestTenantID()
	supplierID := newSupplierTestSupplierID()
	supplier := createTestSupplier(tenantID)

	mockRepo.On("FindByIDForTenant", ctx, tenantID, supplierID).Return(supplier, nil)
	mockPayableRepo.On("CountOutstandingBySupplier", ctx, tenantID, supplierID).Return(int64(0), nil)
	mockOrderRepo.On("CountIncompleteBySupplier", ctx, tenantID, supplierID).Return(int64(2), nil)

	err := service.Delete(ctx, tenantID, supplierID)

	assert.Error(t, err)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "HAS_INCOMPLETE_ORDERS", domainErr.Code)
	assert.Contains(t, domainErr.Message, "2 incomplete order")
	mockRepo.AssertExpectations(t)
	mockPayableRepo.AssertExpectations(t)
	mockOrderRepo.AssertExpectations(t)
}

func TestSupplierService_Delete_SuccessWithRepositoryChecks(t *testing.T) {
	mockRepo := new(MockSupplierRepository)
	mockPayableRepo := new(MockAccountPayableRepositoryForSupplier)
	mockOrderRepo := new(MockPurchaseOrderRepositoryForSupplier)
	service := NewSupplierService(mockRepo)
	service.SetAccountPayableRepo(mockPayableRepo)
	service.SetPurchaseOrderRepo(mockOrderRepo)

	ctx := context.Background()
	tenantID := newSupplierTestTenantID()
	supplierID := newSupplierTestSupplierID()
	supplier := createTestSupplier(tenantID)

	mockRepo.On("FindByIDForTenant", ctx, tenantID, supplierID).Return(supplier, nil)
	mockPayableRepo.On("CountOutstandingBySupplier", ctx, tenantID, supplierID).Return(int64(0), nil)
	mockOrderRepo.On("CountIncompleteBySupplier", ctx, tenantID, supplierID).Return(int64(0), nil)
	mockRepo.On("DeleteForTenant", ctx, tenantID, supplierID).Return(nil)

	err := service.Delete(ctx, tenantID, supplierID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockPayableRepo.AssertExpectations(t)
	mockOrderRepo.AssertExpectations(t)
}

func TestSupplierService_Delete_PayableCheckFailed(t *testing.T) {
	mockRepo := new(MockSupplierRepository)
	mockPayableRepo := new(MockAccountPayableRepositoryForSupplier)
	service := NewSupplierService(mockRepo)
	service.SetAccountPayableRepo(mockPayableRepo)

	ctx := context.Background()
	tenantID := newSupplierTestTenantID()
	supplierID := newSupplierTestSupplierID()
	supplier := createTestSupplier(tenantID)

	mockRepo.On("FindByIDForTenant", ctx, tenantID, supplierID).Return(supplier, nil)
	mockPayableRepo.On("CountOutstandingBySupplier", ctx, tenantID, supplierID).Return(int64(0), errors.New("db error"))

	err := service.Delete(ctx, tenantID, supplierID)

	assert.Error(t, err)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "PAYABLE_CHECK_FAILED", domainErr.Code)
	mockRepo.AssertExpectations(t)
	mockPayableRepo.AssertExpectations(t)
}

func TestSupplierService_Delete_OrderCheckFailed(t *testing.T) {
	mockRepo := new(MockSupplierRepository)
	mockPayableRepo := new(MockAccountPayableRepositoryForSupplier)
	mockOrderRepo := new(MockPurchaseOrderRepositoryForSupplier)
	service := NewSupplierService(mockRepo)
	service.SetAccountPayableRepo(mockPayableRepo)
	service.SetPurchaseOrderRepo(mockOrderRepo)

	ctx := context.Background()
	tenantID := newSupplierTestTenantID()
	supplierID := newSupplierTestSupplierID()
	supplier := createTestSupplier(tenantID)

	mockRepo.On("FindByIDForTenant", ctx, tenantID, supplierID).Return(supplier, nil)
	mockPayableRepo.On("CountOutstandingBySupplier", ctx, tenantID, supplierID).Return(int64(0), nil)
	mockOrderRepo.On("CountIncompleteBySupplier", ctx, tenantID, supplierID).Return(int64(0), errors.New("db error"))

	err := service.Delete(ctx, tenantID, supplierID)

	assert.Error(t, err)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "ORDER_CHECK_FAILED", domainErr.Code)
	mockRepo.AssertExpectations(t)
	mockPayableRepo.AssertExpectations(t)
	mockOrderRepo.AssertExpectations(t)
}

func TestSupplierService_Delete_SupplierNotFound(t *testing.T) {
	mockRepo := new(MockSupplierRepository)
	service := NewSupplierService(mockRepo)

	ctx := context.Background()
	tenantID := newSupplierTestTenantID()
	supplierID := newSupplierTestSupplierID()

	mockRepo.On("FindByIDForTenant", ctx, tenantID, supplierID).Return(nil, shared.ErrNotFound)

	err := service.Delete(ctx, tenantID, supplierID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, shared.ErrNotFound)
	mockRepo.AssertExpectations(t)
}
