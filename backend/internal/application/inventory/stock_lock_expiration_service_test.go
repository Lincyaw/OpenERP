package inventory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockEventBus is a mock implementation of EventBus for testing
type MockEventBus struct {
	mock.Mock
}

func (m *MockEventBus) Publish(ctx context.Context, events ...shared.DomainEvent) error {
	args := m.Called(ctx, events)
	return args.Error(0)
}

func (m *MockEventBus) Subscribe(handler shared.EventHandler, eventTypes ...string) {
	m.Called(handler, eventTypes)
}

func (m *MockEventBus) Unsubscribe(handler shared.EventHandler) {
	m.Called(handler)
}

func (m *MockEventBus) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockEventBus) Stop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func createTestExpiredLock(inventoryItemID uuid.UUID) inventory.StockLock {
	lock := inventory.NewStockLock(
		inventoryItemID,
		decimal.NewFromInt(10),
		"sales_order",
		"SO-001",
		time.Now().Add(-time.Hour), // Expired 1 hour ago
	)
	return *lock
}

func createTestInventoryItemForExpiration(tenantID uuid.UUID) *inventory.InventoryItem {
	item, _ := inventory.NewInventoryItem(tenantID, uuid.New(), uuid.New())
	item.AvailableQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(100))
	item.LockedQuantity = inventory.MustNewInventoryQuantity(decimal.NewFromInt(10))
	return item
}

func TestStockLockExpirationService_ReleaseExpiredLocks_NoExpiredLocks(t *testing.T) {
	mockLockRepo := new(MockStockLockRepository)
	mockInventoryRepo := new(MockInventoryItemRepository)
	mockEventBus := new(MockEventBus)
	logger := zap.NewNop()

	service := NewStockLockExpirationService(mockLockRepo, mockInventoryRepo, mockEventBus, logger)

	// No expired locks
	mockLockRepo.On("FindExpired", mock.Anything).Return([]inventory.StockLock{}, nil)

	stats, err := service.ReleaseExpiredLocks(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats.TotalExpired)
	assert.Equal(t, 0, stats.SuccessReleased)
	assert.Equal(t, 0, stats.FailedReleases)
	mockLockRepo.AssertExpectations(t)
}

func TestStockLockExpirationService_ReleaseExpiredLocks_SingleLock(t *testing.T) {
	mockLockRepo := new(MockStockLockRepository)
	mockInventoryRepo := new(MockInventoryItemRepository)
	mockEventBus := new(MockEventBus)
	logger := zap.NewNop()

	service := NewStockLockExpirationService(mockLockRepo, mockInventoryRepo, mockEventBus, logger)

	tenantID := uuid.New()
	inventoryItem := createTestInventoryItemForExpiration(tenantID)
	expiredLock := createTestExpiredLock(inventoryItem.ID)

	// Setup expectations
	mockLockRepo.On("FindExpired", mock.Anything).Return([]inventory.StockLock{expiredLock}, nil)
	mockInventoryRepo.On("FindByID", mock.Anything, expiredLock.InventoryItemID).Return(inventoryItem, nil)
	mockLockRepo.On("Save", mock.Anything, mock.AnythingOfType("*inventory.StockLock")).Return(nil)
	mockInventoryRepo.On("Save", mock.Anything, mock.AnythingOfType("*inventory.InventoryItem")).Return(nil)
	mockEventBus.On("Publish", mock.Anything, mock.Anything).Return(nil)

	stats, err := service.ReleaseExpiredLocks(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 1, stats.TotalExpired)
	assert.Equal(t, 1, stats.SuccessReleased)
	assert.Equal(t, 0, stats.FailedReleases)
	mockLockRepo.AssertExpectations(t)
	mockInventoryRepo.AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestStockLockExpirationService_ReleaseExpiredLocks_MultipleLocks(t *testing.T) {
	mockLockRepo := new(MockStockLockRepository)
	mockInventoryRepo := new(MockInventoryItemRepository)
	mockEventBus := new(MockEventBus)
	logger := zap.NewNop()

	service := NewStockLockExpirationService(mockLockRepo, mockInventoryRepo, mockEventBus, logger)

	tenantID := uuid.New()
	item1 := createTestInventoryItemForExpiration(tenantID)
	item2 := createTestInventoryItemForExpiration(tenantID)
	lock1 := createTestExpiredLock(item1.ID)
	lock2 := createTestExpiredLock(item2.ID)

	// Setup expectations
	mockLockRepo.On("FindExpired", mock.Anything).Return([]inventory.StockLock{lock1, lock2}, nil)
	mockInventoryRepo.On("FindByID", mock.Anything, lock1.InventoryItemID).Return(item1, nil)
	mockInventoryRepo.On("FindByID", mock.Anything, lock2.InventoryItemID).Return(item2, nil)
	mockLockRepo.On("Save", mock.Anything, mock.AnythingOfType("*inventory.StockLock")).Return(nil).Times(2)
	mockInventoryRepo.On("Save", mock.Anything, mock.AnythingOfType("*inventory.InventoryItem")).Return(nil).Times(2)
	mockEventBus.On("Publish", mock.Anything, mock.Anything).Return(nil).Times(2)

	stats, err := service.ReleaseExpiredLocks(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 2, stats.TotalExpired)
	assert.Equal(t, 2, stats.SuccessReleased)
	assert.Equal(t, 0, stats.FailedReleases)
}

func TestStockLockExpirationService_ReleaseExpiredLocks_InventoryItemNotFound(t *testing.T) {
	mockLockRepo := new(MockStockLockRepository)
	mockInventoryRepo := new(MockInventoryItemRepository)
	mockEventBus := new(MockEventBus)
	logger := zap.NewNop()

	service := NewStockLockExpirationService(mockLockRepo, mockInventoryRepo, mockEventBus, logger)

	inventoryItemID := uuid.New()
	expiredLock := createTestExpiredLock(inventoryItemID)

	// Setup expectations - inventory item not found, but lock should still be released
	mockLockRepo.On("FindExpired", mock.Anything).Return([]inventory.StockLock{expiredLock}, nil)
	mockInventoryRepo.On("FindByID", mock.Anything, expiredLock.InventoryItemID).Return(nil, shared.ErrNotFound)
	mockLockRepo.On("Save", mock.Anything, mock.AnythingOfType("*inventory.StockLock")).Return(nil)

	stats, err := service.ReleaseExpiredLocks(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 1, stats.TotalExpired)
	assert.Equal(t, 1, stats.SuccessReleased)
	mockLockRepo.AssertExpectations(t)
	mockInventoryRepo.AssertExpectations(t)
}

func TestStockLockExpirationService_ReleaseExpiredLocks_SaveFails(t *testing.T) {
	mockLockRepo := new(MockStockLockRepository)
	mockInventoryRepo := new(MockInventoryItemRepository)
	mockEventBus := new(MockEventBus)
	logger := zap.NewNop()

	service := NewStockLockExpirationService(mockLockRepo, mockInventoryRepo, mockEventBus, logger)

	tenantID := uuid.New()
	inventoryItem := createTestInventoryItemForExpiration(tenantID)
	expiredLock := createTestExpiredLock(inventoryItem.ID)

	// Setup expectations - save fails
	mockLockRepo.On("FindExpired", mock.Anything).Return([]inventory.StockLock{expiredLock}, nil)
	mockInventoryRepo.On("FindByID", mock.Anything, expiredLock.InventoryItemID).Return(inventoryItem, nil)
	mockLockRepo.On("Save", mock.Anything, mock.AnythingOfType("*inventory.StockLock")).Return(errors.New("database error"))

	stats, err := service.ReleaseExpiredLocks(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 1, stats.TotalExpired)
	assert.Equal(t, 0, stats.SuccessReleased)
	assert.Equal(t, 1, stats.FailedReleases)
}

func TestStockLockExpirationService_BulkReleaseExpiredLocks(t *testing.T) {
	mockLockRepo := new(MockStockLockRepository)
	mockInventoryRepo := new(MockInventoryItemRepository)
	mockEventBus := new(MockEventBus)
	logger := zap.NewNop()

	service := NewStockLockExpirationService(mockLockRepo, mockInventoryRepo, mockEventBus, logger)

	// Bulk release returns count
	mockLockRepo.On("ReleaseExpired", mock.Anything).Return(5, nil)

	count, err := service.BulkReleaseExpiredLocks(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, 5, count)
	mockLockRepo.AssertExpectations(t)
}

func TestStockLockExpirationService_GetExpiredLockCount(t *testing.T) {
	mockLockRepo := new(MockStockLockRepository)
	mockInventoryRepo := new(MockInventoryItemRepository)
	mockEventBus := new(MockEventBus)
	logger := zap.NewNop()

	service := NewStockLockExpirationService(mockLockRepo, mockInventoryRepo, mockEventBus, logger)

	// Return 3 expired locks
	locks := []inventory.StockLock{
		createTestExpiredLock(uuid.New()),
		createTestExpiredLock(uuid.New()),
		createTestExpiredLock(uuid.New()),
	}
	mockLockRepo.On("FindExpired", mock.Anything).Return(locks, nil)

	count, err := service.GetExpiredLockCount(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, 3, count)
	mockLockRepo.AssertExpectations(t)
}

func TestStockLockExpirationService_SetEventBus(t *testing.T) {
	mockLockRepo := new(MockStockLockRepository)
	mockInventoryRepo := new(MockInventoryItemRepository)
	logger := zap.NewNop()

	// Create service without event bus
	service := NewStockLockExpirationService(mockLockRepo, mockInventoryRepo, nil, logger)
	assert.Nil(t, service.eventBus)

	// Set event bus
	mockEventBus := new(MockEventBus)
	service.SetEventBus(mockEventBus)

	assert.NotNil(t, service.eventBus)
}
