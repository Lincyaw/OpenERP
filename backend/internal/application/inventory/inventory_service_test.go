package inventory

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockEventPublisher is a mock implementation of shared.EventPublisher
type MockEventPublisher struct {
	mu     sync.Mutex
	events []shared.DomainEvent
}

func NewMockEventPublisher() *MockEventPublisher {
	return &MockEventPublisher{
		events: make([]shared.DomainEvent, 0),
	}
}

func (m *MockEventPublisher) Publish(ctx context.Context, events ...shared.DomainEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, events...)
	return nil
}

func (m *MockEventPublisher) GetEvents() []shared.DomainEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]shared.DomainEvent, len(m.events))
	copy(result, m.events)
	return result
}

func (m *MockEventPublisher) GetEventsByType(eventType string) []shared.DomainEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]shared.DomainEvent, 0)
	for _, e := range m.events {
		if e.EventType() == eventType {
			result = append(result, e)
		}
	}
	return result
}

func (m *MockEventPublisher) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = make([]shared.DomainEvent, 0)
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
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockInventoryItemRepository) GetOrCreate(ctx context.Context, tenantID, warehouseID, productID uuid.UUID) (*inventory.InventoryItem, error) {
	args := m.Called(ctx, tenantID, warehouseID, productID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*inventory.InventoryItem), args.Error(1)
}

// MockStockBatchRepository is a mock implementation of StockBatchRepository
type MockStockBatchRepository struct {
	mock.Mock
}

func (m *MockStockBatchRepository) FindByID(ctx context.Context, id uuid.UUID) (*inventory.StockBatch, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*inventory.StockBatch), args.Error(1)
}

func (m *MockStockBatchRepository) FindByInventoryItem(ctx context.Context, inventoryItemID uuid.UUID, filter shared.Filter) ([]inventory.StockBatch, error) {
	args := m.Called(ctx, inventoryItemID, filter)
	return args.Get(0).([]inventory.StockBatch), args.Error(1)
}

func (m *MockStockBatchRepository) FindAvailable(ctx context.Context, inventoryItemID uuid.UUID) ([]inventory.StockBatch, error) {
	args := m.Called(ctx, inventoryItemID)
	return args.Get(0).([]inventory.StockBatch), args.Error(1)
}

func (m *MockStockBatchRepository) FindExpiringSoon(ctx context.Context, tenantID uuid.UUID, withinDays int, filter shared.Filter) ([]inventory.StockBatch, error) {
	args := m.Called(ctx, tenantID, withinDays, filter)
	return args.Get(0).([]inventory.StockBatch), args.Error(1)
}

func (m *MockStockBatchRepository) FindExpired(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]inventory.StockBatch, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]inventory.StockBatch), args.Error(1)
}

func (m *MockStockBatchRepository) FindByBatchNumber(ctx context.Context, inventoryItemID uuid.UUID, batchNumber string) (*inventory.StockBatch, error) {
	args := m.Called(ctx, inventoryItemID, batchNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*inventory.StockBatch), args.Error(1)
}

func (m *MockStockBatchRepository) Save(ctx context.Context, batch *inventory.StockBatch) error {
	args := m.Called(ctx, batch)
	return args.Error(0)
}

func (m *MockStockBatchRepository) SaveBatch(ctx context.Context, batches []inventory.StockBatch) error {
	args := m.Called(ctx, batches)
	return args.Error(0)
}

func (m *MockStockBatchRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockStockBatchRepository) CountByInventoryItem(ctx context.Context, inventoryItemID uuid.UUID) (int64, error) {
	args := m.Called(ctx, inventoryItemID)
	return args.Get(0).(int64), args.Error(1)
}

// MockStockLockRepository is a mock implementation of StockLockRepository
type MockStockLockRepository struct {
	mock.Mock
}

func (m *MockStockLockRepository) FindByID(ctx context.Context, id uuid.UUID) (*inventory.StockLock, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*inventory.StockLock), args.Error(1)
}

func (m *MockStockLockRepository) FindByInventoryItem(ctx context.Context, inventoryItemID uuid.UUID) ([]inventory.StockLock, error) {
	args := m.Called(ctx, inventoryItemID)
	return args.Get(0).([]inventory.StockLock), args.Error(1)
}

func (m *MockStockLockRepository) FindActive(ctx context.Context, inventoryItemID uuid.UUID) ([]inventory.StockLock, error) {
	args := m.Called(ctx, inventoryItemID)
	return args.Get(0).([]inventory.StockLock), args.Error(1)
}

func (m *MockStockLockRepository) FindExpired(ctx context.Context) ([]inventory.StockLock, error) {
	args := m.Called(ctx)
	return args.Get(0).([]inventory.StockLock), args.Error(1)
}

func (m *MockStockLockRepository) FindBySource(ctx context.Context, sourceType, sourceID string) ([]inventory.StockLock, error) {
	args := m.Called(ctx, sourceType, sourceID)
	return args.Get(0).([]inventory.StockLock), args.Error(1)
}

func (m *MockStockLockRepository) Save(ctx context.Context, lock *inventory.StockLock) error {
	args := m.Called(ctx, lock)
	return args.Error(0)
}

func (m *MockStockLockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockStockLockRepository) ReleaseExpired(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Get(0).(int), args.Error(1)
}

// MockTransactionRepository is a mock implementation of InventoryTransactionRepository
type MockTransactionRepository struct {
	mock.Mock
}

func (m *MockTransactionRepository) FindByID(ctx context.Context, id uuid.UUID) (*inventory.InventoryTransaction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*inventory.InventoryTransaction), args.Error(1)
}

func (m *MockTransactionRepository) FindByInventoryItem(ctx context.Context, inventoryItemID uuid.UUID, filter shared.Filter) ([]inventory.InventoryTransaction, error) {
	args := m.Called(ctx, inventoryItemID, filter)
	return args.Get(0).([]inventory.InventoryTransaction), args.Error(1)
}

func (m *MockTransactionRepository) FindByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID, filter shared.Filter) ([]inventory.InventoryTransaction, error) {
	args := m.Called(ctx, tenantID, warehouseID, filter)
	return args.Get(0).([]inventory.InventoryTransaction), args.Error(1)
}

func (m *MockTransactionRepository) FindByProduct(ctx context.Context, tenantID, productID uuid.UUID, filter shared.Filter) ([]inventory.InventoryTransaction, error) {
	args := m.Called(ctx, tenantID, productID, filter)
	return args.Get(0).([]inventory.InventoryTransaction), args.Error(1)
}

func (m *MockTransactionRepository) FindBySource(ctx context.Context, sourceType inventory.SourceType, sourceID string) ([]inventory.InventoryTransaction, error) {
	args := m.Called(ctx, sourceType, sourceID)
	return args.Get(0).([]inventory.InventoryTransaction), args.Error(1)
}

func (m *MockTransactionRepository) FindByDateRange(ctx context.Context, tenantID uuid.UUID, start, end time.Time, filter shared.Filter) ([]inventory.InventoryTransaction, error) {
	args := m.Called(ctx, tenantID, start, end, filter)
	return args.Get(0).([]inventory.InventoryTransaction), args.Error(1)
}

func (m *MockTransactionRepository) FindByType(ctx context.Context, tenantID uuid.UUID, txType inventory.TransactionType, filter shared.Filter) ([]inventory.InventoryTransaction, error) {
	args := m.Called(ctx, tenantID, txType, filter)
	return args.Get(0).([]inventory.InventoryTransaction), args.Error(1)
}

func (m *MockTransactionRepository) FindForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]inventory.InventoryTransaction, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]inventory.InventoryTransaction), args.Error(1)
}

func (m *MockTransactionRepository) Create(ctx context.Context, tx *inventory.InventoryTransaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockTransactionRepository) CreateBatch(ctx context.Context, txs []*inventory.InventoryTransaction) error {
	args := m.Called(ctx, txs)
	return args.Error(0)
}

func (m *MockTransactionRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTransactionRepository) CountByInventoryItem(ctx context.Context, inventoryItemID uuid.UUID) (int64, error) {
	args := m.Called(ctx, inventoryItemID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTransactionRepository) SumQuantityByTypeAndDateRange(ctx context.Context, tenantID uuid.UUID, txType inventory.TransactionType, start, end time.Time) (decimal.Decimal, error) {
	args := m.Called(ctx, tenantID, txType, start, end)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

// Test helpers
func createTestInventoryItem(tenantID, warehouseID, productID uuid.UUID) *inventory.InventoryItem {
	item, _ := inventory.NewInventoryItem(tenantID, warehouseID, productID)
	return item
}

func createTestInventoryItemWithStock(tenantID, warehouseID, productID uuid.UUID, available, locked decimal.Decimal) *inventory.InventoryItem {
	item, _ := inventory.NewInventoryItem(tenantID, warehouseID, productID)
	item.AvailableQuantity = inventory.MustNewInventoryQuantity(available)
	item.LockedQuantity = inventory.MustNewInventoryQuantity(locked)
	item.UnitCost = decimal.NewFromFloat(10.0)
	return item
}

// Tests

func TestNewInventoryService(t *testing.T) {
	invRepo := new(MockInventoryItemRepository)
	batchRepo := new(MockStockBatchRepository)
	lockRepo := new(MockStockLockRepository)
	txRepo := new(MockTransactionRepository)

	service := NewInventoryService(invRepo, batchRepo, lockRepo, txRepo)

	assert.NotNil(t, service)
}

func TestInventoryService_GetByID(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	itemID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	invRepo := new(MockInventoryItemRepository)
	batchRepo := new(MockStockBatchRepository)
	lockRepo := new(MockStockLockRepository)
	txRepo := new(MockTransactionRepository)

	service := NewInventoryService(invRepo, batchRepo, lockRepo, txRepo)

	item := createTestInventoryItemWithStock(tenantID, warehouseID, productID, decimal.NewFromInt(100), decimal.NewFromInt(10))
	item.ID = itemID

	t.Run("success", func(t *testing.T) {
		invRepo.On("FindByIDForTenant", ctx, tenantID, itemID).Return(item, nil).Once()

		response, err := service.GetByID(ctx, tenantID, itemID)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, itemID, response.ID)
		assert.Equal(t, decimal.NewFromInt(100), response.AvailableQuantity)
		assert.Equal(t, decimal.NewFromInt(10), response.LockedQuantity)
		assert.Equal(t, decimal.NewFromInt(110), response.TotalQuantity)
	})

	t.Run("not found", func(t *testing.T) {
		invRepo.On("FindByIDForTenant", ctx, tenantID, itemID).Return(nil, shared.ErrNotFound).Once()

		response, err := service.GetByID(ctx, tenantID, itemID)

		assert.Error(t, err)
		assert.Nil(t, response)
		assert.True(t, errors.Is(err, shared.ErrNotFound))
	})
}

func TestInventoryService_GetByWarehouseAndProduct(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	invRepo := new(MockInventoryItemRepository)
	batchRepo := new(MockStockBatchRepository)
	lockRepo := new(MockStockLockRepository)
	txRepo := new(MockTransactionRepository)

	service := NewInventoryService(invRepo, batchRepo, lockRepo, txRepo)

	item := createTestInventoryItemWithStock(tenantID, warehouseID, productID, decimal.NewFromInt(50), decimal.Zero)

	t.Run("success", func(t *testing.T) {
		invRepo.On("FindByWarehouseAndProduct", mock.Anything, tenantID, warehouseID, productID).Return(item, nil).Once()

		response, err := service.GetByWarehouseAndProduct(ctx, tenantID, warehouseID, productID)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, warehouseID, response.WarehouseID)
		assert.Equal(t, productID, response.ProductID)
	})

	t.Run("not found", func(t *testing.T) {
		invRepo.On("FindByWarehouseAndProduct", mock.Anything, tenantID, warehouseID, productID).Return(nil, shared.ErrNotFound).Once()

		response, err := service.GetByWarehouseAndProduct(ctx, tenantID, warehouseID, productID)

		assert.Error(t, err)
		assert.Nil(t, response)
	})
}

func TestInventoryService_List(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	invRepo := new(MockInventoryItemRepository)
	batchRepo := new(MockStockBatchRepository)
	lockRepo := new(MockStockLockRepository)
	txRepo := new(MockTransactionRepository)

	service := NewInventoryService(invRepo, batchRepo, lockRepo, txRepo)

	items := []inventory.InventoryItem{
		*createTestInventoryItemWithStock(tenantID, warehouseID, productID, decimal.NewFromInt(100), decimal.Zero),
		*createTestInventoryItemWithStock(tenantID, warehouseID, uuid.New(), decimal.NewFromInt(50), decimal.NewFromInt(10)),
	}

	t.Run("success with defaults", func(t *testing.T) {
		filter := InventoryListFilter{}
		invRepo.On("FindAllForTenant", ctx, tenantID, mock.AnythingOfType("shared.Filter")).Return(items, nil).Once()
		invRepo.On("CountForTenant", ctx, tenantID, mock.AnythingOfType("shared.Filter")).Return(int64(2), nil).Once()

		responses, total, err := service.List(ctx, tenantID, filter)

		assert.NoError(t, err)
		assert.Len(t, responses, 2)
		assert.Equal(t, int64(2), total)
	})

	t.Run("with warehouse filter", func(t *testing.T) {
		filter := InventoryListFilter{WarehouseID: warehouseID.String()}
		invRepo.On("FindAllForTenant", ctx, tenantID, mock.AnythingOfType("shared.Filter")).Return(items, nil).Once()
		invRepo.On("CountForTenant", ctx, tenantID, mock.AnythingOfType("shared.Filter")).Return(int64(2), nil).Once()

		responses, total, err := service.List(ctx, tenantID, filter)

		assert.NoError(t, err)
		assert.Len(t, responses, 2)
		assert.Equal(t, int64(2), total)
	})
}

func TestInventoryService_IncreaseStock(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	invRepo := new(MockInventoryItemRepository)
	batchRepo := new(MockStockBatchRepository)
	lockRepo := new(MockStockLockRepository)
	txRepo := new(MockTransactionRepository)

	service := NewInventoryService(invRepo, batchRepo, lockRepo, txRepo)

	t.Run("success - increase existing stock", func(t *testing.T) {
		item := createTestInventoryItemWithStock(tenantID, warehouseID, productID, decimal.NewFromInt(100), decimal.Zero)

		invRepo.On("GetOrCreate", mock.Anything, tenantID, warehouseID, productID).Return(item, nil).Once()
		invRepo.On("SaveWithLock", mock.Anything, mock.AnythingOfType("*inventory.InventoryItem")).Return(nil).Once()
		txRepo.On("Create", mock.Anything, mock.AnythingOfType("*inventory.InventoryTransaction")).Return(nil).Once()

		req := IncreaseStockRequest{
			WarehouseID: warehouseID,
			ProductID:   productID,
			Quantity:    decimal.NewFromInt(50),
			UnitCost:    decimal.NewFromFloat(15.0),
			SourceType:  "PURCHASE_ORDER",
			SourceID:    "PO-001",
		}

		response, err := service.IncreaseStock(ctx, tenantID, req)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		// Available should have increased by 50
		assert.True(t, response.AvailableQuantity.GreaterThan(decimal.NewFromInt(100)))
	})

	t.Run("invalid source type", func(t *testing.T) {
		req := IncreaseStockRequest{
			WarehouseID: warehouseID,
			ProductID:   productID,
			Quantity:    decimal.NewFromInt(50),
			UnitCost:    decimal.NewFromFloat(15.0),
			SourceType:  "INVALID_TYPE",
			SourceID:    "PO-001",
		}

		response, err := service.IncreaseStock(ctx, tenantID, req)

		assert.Error(t, err)
		assert.Nil(t, response)
	})

	t.Run("records cost method in transaction", func(t *testing.T) {
		item := createTestInventoryItemWithStock(tenantID, warehouseID, productID, decimal.NewFromInt(100), decimal.Zero)
		var capturedTx *inventory.InventoryTransaction

		invRepo.On("GetOrCreate", mock.Anything, tenantID, warehouseID, productID).Return(item, nil).Once()
		invRepo.On("SaveWithLock", mock.Anything, mock.AnythingOfType("*inventory.InventoryItem")).Return(nil).Once()
		txRepo.On("Create", mock.Anything, mock.AnythingOfType("*inventory.InventoryTransaction")).Run(func(args mock.Arguments) {
			capturedTx = args.Get(1).(*inventory.InventoryTransaction)
		}).Return(nil).Once()

		req := IncreaseStockRequest{
			WarehouseID: warehouseID,
			ProductID:   productID,
			Quantity:    decimal.NewFromInt(50),
			UnitCost:    decimal.NewFromFloat(15.0),
			SourceType:  "PURCHASE_ORDER",
			SourceID:    "PO-002",
		}

		response, err := service.IncreaseStock(ctx, tenantID, req)

		require.NoError(t, err)
		assert.NotNil(t, response)
		// Without strategy provider, should use default moving_average
		assert.Equal(t, "moving_average", capturedTx.CostMethod)
	})
}

func TestInventoryService_LockStock(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	invRepo := new(MockInventoryItemRepository)
	batchRepo := new(MockStockBatchRepository)
	lockRepo := new(MockStockLockRepository)
	txRepo := new(MockTransactionRepository)

	service := NewInventoryService(invRepo, batchRepo, lockRepo, txRepo)

	t.Run("success - lock available stock", func(t *testing.T) {
		item := createTestInventoryItemWithStock(tenantID, warehouseID, productID, decimal.NewFromInt(100), decimal.Zero)

		invRepo.On("FindByWarehouseAndProduct", mock.Anything, tenantID, warehouseID, productID).Return(item, nil).Once()
		invRepo.On("SaveWithLock", mock.Anything, mock.AnythingOfType("*inventory.InventoryItem")).Return(nil).Once()
		lockRepo.On("Save", mock.Anything, mock.AnythingOfType("*inventory.StockLock")).Return(nil).Once()
		txRepo.On("Create", mock.Anything, mock.AnythingOfType("*inventory.InventoryTransaction")).Return(nil).Once()

		req := LockStockRequest{
			WarehouseID: warehouseID,
			ProductID:   productID,
			Quantity:    decimal.NewFromInt(30),
			SourceType:  "sales_order",
			SourceID:    "SO-001",
		}

		response, err := service.LockStock(ctx, tenantID, req)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, decimal.NewFromInt(30), response.Quantity)
		assert.Equal(t, warehouseID, response.WarehouseID)
		assert.Equal(t, productID, response.ProductID)
	})

	t.Run("insufficient stock", func(t *testing.T) {
		item := createTestInventoryItemWithStock(tenantID, warehouseID, productID, decimal.NewFromInt(20), decimal.Zero)

		invRepo.On("FindByWarehouseAndProduct", mock.Anything, tenantID, warehouseID, productID).Return(item, nil).Once()

		req := LockStockRequest{
			WarehouseID: warehouseID,
			ProductID:   productID,
			Quantity:    decimal.NewFromInt(50), // More than available
			SourceType:  "sales_order",
			SourceID:    "SO-002",
		}

		response, err := service.LockStock(ctx, tenantID, req)

		assert.Error(t, err)
		assert.Nil(t, response)
	})

	t.Run("no inventory found", func(t *testing.T) {
		invRepo.On("FindByWarehouseAndProduct", mock.Anything, tenantID, warehouseID, productID).Return(nil, shared.ErrNotFound).Once()

		req := LockStockRequest{
			WarehouseID: warehouseID,
			ProductID:   productID,
			Quantity:    decimal.NewFromInt(10),
			SourceType:  "sales_order",
			SourceID:    "SO-003",
		}

		response, err := service.LockStock(ctx, tenantID, req)

		assert.Error(t, err)
		assert.Nil(t, response)
	})
}

func TestInventoryService_UnlockStock(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()
	lockID := uuid.New()
	itemID := uuid.New()

	invRepo := new(MockInventoryItemRepository)
	batchRepo := new(MockStockBatchRepository)
	lockRepo := new(MockStockLockRepository)
	txRepo := new(MockTransactionRepository)

	service := NewInventoryService(invRepo, batchRepo, lockRepo, txRepo)

	t.Run("success - unlock stock", func(t *testing.T) {
		lock := inventory.NewStockLock(itemID, decimal.NewFromInt(30), "sales_order", "SO-001", time.Now().Add(time.Hour))
		lock.ID = lockID

		item := createTestInventoryItemWithStock(tenantID, warehouseID, productID, decimal.NewFromInt(70), decimal.NewFromInt(30))
		item.ID = itemID
		item.Locks = []inventory.StockLock{*lock}

		lockRepo.On("FindByID", mock.Anything, lockID).Return(lock, nil).Once()
		invRepo.On("FindByID", mock.Anything, itemID).Return(item, nil).Once()
		invRepo.On("SaveWithLock", mock.Anything, mock.AnythingOfType("*inventory.InventoryItem")).Return(nil).Once()
		lockRepo.On("Save", mock.Anything, mock.AnythingOfType("*inventory.StockLock")).Return(nil).Once()
		txRepo.On("Create", mock.Anything, mock.AnythingOfType("*inventory.InventoryTransaction")).Return(nil).Once()

		req := UnlockStockRequest{LockID: lockID}

		err := service.UnlockStock(ctx, tenantID, req)

		assert.NoError(t, err)
	})

	t.Run("lock not found", func(t *testing.T) {
		lockRepo.On("FindByID", mock.Anything, lockID).Return(nil, shared.ErrNotFound).Once()

		req := UnlockStockRequest{LockID: lockID}

		err := service.UnlockStock(ctx, tenantID, req)

		assert.Error(t, err)
	})

	t.Run("success - unlock specific lock from multiple locks", func(t *testing.T) {
		// Create multiple locks - the target lock is NOT at index 0
		lock1ID := uuid.New()
		lock1 := inventory.NewStockLock(itemID, decimal.NewFromInt(10), "sales_order", "SO-001", time.Now().Add(time.Hour))
		lock1.ID = lock1ID

		lock2ID := uuid.New()
		lock2 := inventory.NewStockLock(itemID, decimal.NewFromInt(30), "sales_order", "SO-002", time.Now().Add(time.Hour))
		lock2.ID = lock2ID // This is the target lock at index 1

		item := createTestInventoryItemWithStock(tenantID, warehouseID, productID, decimal.NewFromInt(60), decimal.NewFromInt(40))
		item.ID = itemID
		item.Locks = []inventory.StockLock{*lock1, *lock2} // lock2 is at index 1, not 0

		lockRepo.On("FindByID", mock.Anything, lock2ID).Return(lock2, nil).Once()
		invRepo.On("FindByID", mock.Anything, itemID).Return(item, nil).Once()
		invRepo.On("SaveWithLock", mock.Anything, mock.AnythingOfType("*inventory.InventoryItem")).Return(nil).Once()

		// Verify that the correct lock (lock2) is saved, not lock1 at index 0
		lockRepo.On("Save", mock.Anything, mock.MatchedBy(func(l *inventory.StockLock) bool {
			return l.ID == lock2ID && l.Released
		})).Return(nil).Once()

		txRepo.On("Create", mock.Anything, mock.AnythingOfType("*inventory.InventoryTransaction")).Return(nil).Once()

		req := UnlockStockRequest{LockID: lock2ID}

		err := service.UnlockStock(ctx, tenantID, req)

		assert.NoError(t, err)
		lockRepo.AssertExpectations(t)
	})
}

func TestInventoryService_AdjustStock(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	invRepo := new(MockInventoryItemRepository)
	batchRepo := new(MockStockBatchRepository)
	lockRepo := new(MockStockLockRepository)
	txRepo := new(MockTransactionRepository)

	service := NewInventoryService(invRepo, batchRepo, lockRepo, txRepo)

	t.Run("success - increase adjustment", func(t *testing.T) {
		item := createTestInventoryItemWithStock(tenantID, warehouseID, productID, decimal.NewFromInt(100), decimal.Zero)

		invRepo.On("GetOrCreate", mock.Anything, tenantID, warehouseID, productID).Return(item, nil).Once()
		invRepo.On("SaveWithLock", mock.Anything, mock.AnythingOfType("*inventory.InventoryItem")).Return(nil).Once()
		txRepo.On("Create", mock.Anything, mock.AnythingOfType("*inventory.InventoryTransaction")).Return(nil).Once()

		req := AdjustStockRequest{
			WarehouseID:    warehouseID,
			ProductID:      productID,
			ActualQuantity: decimal.NewFromInt(120),
			Reason:         "Stock count variance",
		}

		response, err := service.AdjustStock(ctx, tenantID, req)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, decimal.NewFromInt(120), response.AvailableQuantity)
	})

	t.Run("success - decrease adjustment", func(t *testing.T) {
		item := createTestInventoryItemWithStock(tenantID, warehouseID, productID, decimal.NewFromInt(100), decimal.Zero)

		invRepo.On("GetOrCreate", mock.Anything, tenantID, warehouseID, productID).Return(item, nil).Once()
		invRepo.On("SaveWithLock", mock.Anything, mock.AnythingOfType("*inventory.InventoryItem")).Return(nil).Once()
		txRepo.On("Create", mock.Anything, mock.AnythingOfType("*inventory.InventoryTransaction")).Return(nil).Once()

		req := AdjustStockRequest{
			WarehouseID:    warehouseID,
			ProductID:      productID,
			ActualQuantity: decimal.NewFromInt(80),
			Reason:         "Damaged goods",
		}

		response, err := service.AdjustStock(ctx, tenantID, req)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, decimal.NewFromInt(80), response.AvailableQuantity)
	})

	t.Run("cannot adjust with locked stock", func(t *testing.T) {
		item := createTestInventoryItemWithStock(tenantID, warehouseID, productID, decimal.NewFromInt(100), decimal.NewFromInt(20))

		invRepo.On("GetOrCreate", mock.Anything, tenantID, warehouseID, productID).Return(item, nil).Once()

		req := AdjustStockRequest{
			WarehouseID:    warehouseID,
			ProductID:      productID,
			ActualQuantity: decimal.NewFromInt(80),
			Reason:         "Stock count",
		}

		response, err := service.AdjustStock(ctx, tenantID, req)

		assert.Error(t, err)
		assert.Nil(t, response)
	})
}

func TestInventoryService_SetThresholds(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	invRepo := new(MockInventoryItemRepository)
	batchRepo := new(MockStockBatchRepository)
	lockRepo := new(MockStockLockRepository)
	txRepo := new(MockTransactionRepository)

	service := NewInventoryService(invRepo, batchRepo, lockRepo, txRepo)

	t.Run("success - set min quantity", func(t *testing.T) {
		item := createTestInventoryItemWithStock(tenantID, warehouseID, productID, decimal.NewFromInt(100), decimal.Zero)

		invRepo.On("GetOrCreate", mock.Anything, tenantID, warehouseID, productID).Return(item, nil).Once()
		invRepo.On("Save", mock.Anything, mock.AnythingOfType("*inventory.InventoryItem")).Return(nil).Once()

		minQty := decimal.NewFromInt(10)
		req := SetThresholdsRequest{
			WarehouseID: warehouseID,
			ProductID:   productID,
			MinQuantity: &minQty,
		}

		response, err := service.SetThresholds(ctx, tenantID, req)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, minQty, response.MinQuantity)
	})
}

func TestInventoryService_CheckAvailability(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	invRepo := new(MockInventoryItemRepository)
	batchRepo := new(MockStockBatchRepository)
	lockRepo := new(MockStockLockRepository)
	txRepo := new(MockTransactionRepository)

	service := NewInventoryService(invRepo, batchRepo, lockRepo, txRepo)

	t.Run("sufficient stock", func(t *testing.T) {
		item := createTestInventoryItemWithStock(tenantID, warehouseID, productID, decimal.NewFromInt(100), decimal.Zero)

		invRepo.On("FindByWarehouseAndProduct", mock.Anything, tenantID, warehouseID, productID).Return(item, nil).Once()

		available, qty, err := service.CheckAvailability(ctx, tenantID, warehouseID, productID, decimal.NewFromInt(50))

		assert.NoError(t, err)
		assert.True(t, available)
		assert.Equal(t, decimal.NewFromInt(100), qty)
	})

	t.Run("insufficient stock", func(t *testing.T) {
		item := createTestInventoryItemWithStock(tenantID, warehouseID, productID, decimal.NewFromInt(30), decimal.Zero)

		invRepo.On("FindByWarehouseAndProduct", mock.Anything, tenantID, warehouseID, productID).Return(item, nil).Once()

		available, qty, err := service.CheckAvailability(ctx, tenantID, warehouseID, productID, decimal.NewFromInt(50))

		assert.NoError(t, err)
		assert.False(t, available)
		assert.Equal(t, decimal.NewFromInt(30), qty)
	})

	t.Run("no inventory", func(t *testing.T) {
		invRepo.On("FindByWarehouseAndProduct", mock.Anything, tenantID, warehouseID, productID).Return(nil, shared.ErrNotFound).Once()

		available, qty, err := service.CheckAvailability(ctx, tenantID, warehouseID, productID, decimal.NewFromInt(50))

		assert.NoError(t, err)
		assert.False(t, available)
		assert.Equal(t, decimal.Zero, qty)
	})
}

func TestInventoryService_ListTransactions(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	invRepo := new(MockInventoryItemRepository)
	batchRepo := new(MockStockBatchRepository)
	lockRepo := new(MockStockLockRepository)
	txRepo := new(MockTransactionRepository)

	service := NewInventoryService(invRepo, batchRepo, lockRepo, txRepo)

	t.Run("success", func(t *testing.T) {
		txs := []inventory.InventoryTransaction{}
		txRepo.On("FindForTenant", ctx, tenantID, mock.AnythingOfType("shared.Filter")).Return(txs, nil).Once()
		txRepo.On("CountForTenant", ctx, tenantID, mock.AnythingOfType("shared.Filter")).Return(int64(0), nil).Once()

		filter := TransactionListFilter{
			Page:     1,
			PageSize: 20,
		}

		responses, total, err := service.ListTransactions(ctx, tenantID, filter)

		assert.NoError(t, err)
		assert.Len(t, responses, 0)
		assert.Equal(t, int64(0), total)
	})
}

// Test DTOs conversion functions
func TestToInventoryItemResponse(t *testing.T) {
	tenantID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	item := createTestInventoryItemWithStock(tenantID, warehouseID, productID, decimal.NewFromInt(100), decimal.NewFromInt(20))
	item.MinQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(10))
	item.MaxQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(500))

	response := ToInventoryItemResponse(item)

	assert.Equal(t, item.ID, response.ID)
	assert.Equal(t, tenantID, response.TenantID)
	assert.Equal(t, warehouseID, response.WarehouseID)
	assert.Equal(t, productID, response.ProductID)
	assert.Equal(t, decimal.NewFromInt(100), response.AvailableQuantity)
	assert.Equal(t, decimal.NewFromInt(20), response.LockedQuantity)
	assert.Equal(t, decimal.NewFromInt(120), response.TotalQuantity)
	assert.Equal(t, decimal.NewFromInt(10), response.MinQuantity)
	assert.Equal(t, decimal.NewFromInt(500), response.MaxQuantity)
	assert.False(t, response.IsBelowMinimum)
	assert.False(t, response.IsAboveMaximum)
}

func TestToStockLockResponse(t *testing.T) {
	itemID := uuid.New()
	lock := inventory.NewStockLock(itemID, decimal.NewFromInt(50), "sales_order", "SO-001", time.Now().Add(time.Hour))

	response := ToStockLockResponse(lock)

	assert.Equal(t, lock.ID, response.ID)
	assert.Equal(t, itemID, response.InventoryItemID)
	assert.Equal(t, decimal.NewFromInt(50), response.Quantity)
	assert.Equal(t, "sales_order", response.SourceType)
	assert.Equal(t, "SO-001", response.SourceID)
	assert.True(t, response.IsActive)
	assert.False(t, response.IsExpired)
	assert.False(t, response.Released)
	assert.False(t, response.Consumed)
}

// Test event publishing

func TestInventoryService_AdjustStock_PublishesStockBelowThresholdEvent(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	invRepo := new(MockInventoryItemRepository)
	batchRepo := new(MockStockBatchRepository)
	lockRepo := new(MockStockLockRepository)
	txRepo := new(MockTransactionRepository)
	eventPublisher := NewMockEventPublisher()

	service := NewInventoryService(invRepo, batchRepo, lockRepo, txRepo)
	service.SetEventPublisher(eventPublisher)

	// Create an inventory item with minQuantity set and initial stock above threshold
	item := createTestInventoryItemWithStock(tenantID, warehouseID, productID, decimal.NewFromInt(50), decimal.Zero)
	item.MinQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(10)) // Set minimum threshold

	t.Run("publishes StockBelowThreshold event when stock drops below minimum", func(t *testing.T) {
		eventPublisher.Reset()

		// Mock GetOrCreate to return the item
		invRepo.On("GetOrCreate", mock.Anything, tenantID, warehouseID, productID).Return(item, nil).Once()
		// Mock SaveWithLock to succeed
		invRepo.On("SaveWithLock", mock.Anything, mock.AnythingOfType("*inventory.InventoryItem")).Return(nil).Once()
		// Mock transaction repo
		txRepo.On("Create", mock.Anything, mock.AnythingOfType("*inventory.InventoryTransaction")).Return(nil).Once()

		// Adjust stock to below minimum threshold (5 < 10)
		req := AdjustStockRequest{
			WarehouseID:    warehouseID,
			ProductID:      productID,
			ActualQuantity: decimal.NewFromInt(5),
			Reason:         "Inventory count adjustment",
		}

		_, err := service.AdjustStock(ctx, tenantID, req)
		require.NoError(t, err)

		// Verify events were published
		events := eventPublisher.GetEvents()
		require.GreaterOrEqual(t, len(events), 1, "Expected at least one event to be published")

		// Check for StockBelowThreshold event
		thresholdEvents := eventPublisher.GetEventsByType(inventory.EventTypeStockBelowThreshold)
		require.Len(t, thresholdEvents, 1, "Expected exactly one StockBelowThreshold event")

		thresholdEvent, ok := thresholdEvents[0].(*inventory.StockBelowThresholdEvent)
		require.True(t, ok, "Event should be of type *inventory.StockBelowThresholdEvent")

		assert.Equal(t, warehouseID, thresholdEvent.WarehouseID)
		assert.Equal(t, productID, thresholdEvent.ProductID)
		assert.Equal(t, decimal.NewFromInt(5), thresholdEvent.CurrentQuantity)
		assert.Equal(t, decimal.NewFromInt(10), thresholdEvent.MinimumQuantity)
	})
}

func TestInventoryService_DecreaseStock_PublishesStockBelowThresholdEvent(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	invRepo := new(MockInventoryItemRepository)
	batchRepo := new(MockStockBatchRepository)
	lockRepo := new(MockStockLockRepository)
	txRepo := new(MockTransactionRepository)
	eventPublisher := NewMockEventPublisher()

	service := NewInventoryService(invRepo, batchRepo, lockRepo, txRepo)
	service.SetEventPublisher(eventPublisher)

	// Create an inventory item with minQuantity set and initial stock above threshold
	item := createTestInventoryItemWithStock(tenantID, warehouseID, productID, decimal.NewFromInt(15), decimal.Zero)
	item.MinQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(10)) // Set minimum threshold

	t.Run("publishes StockBelowThreshold event when stock is decreased below minimum", func(t *testing.T) {
		eventPublisher.Reset()

		// Mock FindByWarehouseAndProduct to return the item
		invRepo.On("FindByWarehouseAndProduct", mock.Anything, tenantID, warehouseID, productID).Return(item, nil).Once()
		// Mock SaveWithLock to succeed
		invRepo.On("SaveWithLock", mock.Anything, mock.AnythingOfType("*inventory.InventoryItem")).Return(nil).Once()
		// Mock transaction repo
		txRepo.On("Create", mock.Anything, mock.AnythingOfType("*inventory.InventoryTransaction")).Return(nil).Once()

		// Decrease stock by 10 (15 - 10 = 5, which is below minimum of 10)
		req := DecreaseStockRequest{
			WarehouseID: warehouseID,
			ProductID:   productID,
			Quantity:    decimal.NewFromInt(10),
			SourceType:  "PURCHASE_RETURN",
			SourceID:    "PR-001",
			Reason:      "Purchase return shipped",
		}

		err := service.DecreaseStock(ctx, tenantID, req)
		require.NoError(t, err)

		// Verify StockBelowThreshold event was published
		thresholdEvents := eventPublisher.GetEventsByType(inventory.EventTypeStockBelowThreshold)
		require.Len(t, thresholdEvents, 1, "Expected exactly one StockBelowThreshold event")

		thresholdEvent, ok := thresholdEvents[0].(*inventory.StockBelowThresholdEvent)
		require.True(t, ok)

		assert.Equal(t, warehouseID, thresholdEvent.WarehouseID)
		assert.Equal(t, productID, thresholdEvent.ProductID)
		assert.Equal(t, decimal.NewFromInt(5), thresholdEvent.CurrentQuantity)
	})

	t.Run("does not publish event when stock remains above minimum", func(t *testing.T) {
		eventPublisher.Reset()

		// Reset item stock to above threshold
		item.AvailableQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(100))
		item.ClearDomainEvents()

		// Mock FindByWarehouseAndProduct to return the item
		invRepo.On("FindByWarehouseAndProduct", mock.Anything, tenantID, warehouseID, productID).Return(item, nil).Once()
		// Mock SaveWithLock to succeed
		invRepo.On("SaveWithLock", mock.Anything, mock.AnythingOfType("*inventory.InventoryItem")).Return(nil).Once()
		// Mock transaction repo
		txRepo.On("Create", mock.Anything, mock.AnythingOfType("*inventory.InventoryTransaction")).Return(nil).Once()

		// Decrease stock by 10 (100 - 10 = 90, which is still above minimum of 10)
		req := DecreaseStockRequest{
			WarehouseID: warehouseID,
			ProductID:   productID,
			Quantity:    decimal.NewFromInt(10),
			SourceType:  "PURCHASE_RETURN",
			SourceID:    "PR-002",
			Reason:      "Purchase return shipped",
		}

		err := service.DecreaseStock(ctx, tenantID, req)
		require.NoError(t, err)

		// Verify no StockBelowThreshold event was published
		thresholdEvents := eventPublisher.GetEventsByType(inventory.EventTypeStockBelowThreshold)
		assert.Len(t, thresholdEvents, 0, "Expected no StockBelowThreshold event when stock is above minimum")
	})
}

func TestInventoryService_SetEventPublisher(t *testing.T) {
	invRepo := new(MockInventoryItemRepository)
	batchRepo := new(MockStockBatchRepository)
	lockRepo := new(MockStockLockRepository)
	txRepo := new(MockTransactionRepository)

	service := NewInventoryService(invRepo, batchRepo, lockRepo, txRepo)

	// Initially, eventPublisher should be nil
	assert.Nil(t, service.eventPublisher)

	// Set event publisher
	eventPublisher := NewMockEventPublisher()
	service.SetEventPublisher(eventPublisher)

	// Now eventPublisher should be set
	assert.NotNil(t, service.eventPublisher)
}
