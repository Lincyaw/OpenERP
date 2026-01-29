package trade

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPurchaseOrderRepository is a mock implementation of PurchaseOrderRepository
type MockPurchaseOrderRepository struct {
	mock.Mock
}

func (m *MockPurchaseOrderRepository) FindByID(ctx context.Context, id uuid.UUID) (*trade.PurchaseOrder, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*trade.PurchaseOrder), args.Error(1)
}

func (m *MockPurchaseOrderRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*trade.PurchaseOrder, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*trade.PurchaseOrder), args.Error(1)
}

func (m *MockPurchaseOrderRepository) FindByOrderNumber(ctx context.Context, tenantID uuid.UUID, orderNumber string) (*trade.PurchaseOrder, error) {
	args := m.Called(ctx, tenantID, orderNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*trade.PurchaseOrder), args.Error(1)
}

func (m *MockPurchaseOrderRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]trade.PurchaseOrder, error) {
	args := m.Called(ctx, tenantID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.PurchaseOrder), args.Error(1)
}

func (m *MockPurchaseOrderRepository) FindBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID, filter shared.Filter) ([]trade.PurchaseOrder, error) {
	args := m.Called(ctx, tenantID, supplierID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.PurchaseOrder), args.Error(1)
}

func (m *MockPurchaseOrderRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status trade.PurchaseOrderStatus, filter shared.Filter) ([]trade.PurchaseOrder, error) {
	args := m.Called(ctx, tenantID, status, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.PurchaseOrder), args.Error(1)
}

func (m *MockPurchaseOrderRepository) FindByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID, filter shared.Filter) ([]trade.PurchaseOrder, error) {
	args := m.Called(ctx, tenantID, warehouseID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.PurchaseOrder), args.Error(1)
}

func (m *MockPurchaseOrderRepository) FindPendingReceipt(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]trade.PurchaseOrder, error) {
	args := m.Called(ctx, tenantID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.PurchaseOrder), args.Error(1)
}

func (m *MockPurchaseOrderRepository) Save(ctx context.Context, order *trade.PurchaseOrder) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockPurchaseOrderRepository) SaveWithLock(ctx context.Context, order *trade.PurchaseOrder) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockPurchaseOrderRepository) SaveWithLockAndEvents(ctx context.Context, order *trade.PurchaseOrder, events []shared.DomainEvent) error {
	args := m.Called(ctx, order, events)
	return args.Error(0)
}

func (m *MockPurchaseOrderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPurchaseOrderRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *MockPurchaseOrderRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPurchaseOrderRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status trade.PurchaseOrderStatus) (int64, error) {
	args := m.Called(ctx, tenantID, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPurchaseOrderRepository) CountBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID, supplierID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPurchaseOrderRepository) CountIncompleteBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID, supplierID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPurchaseOrderRepository) CountPendingReceipt(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPurchaseOrderRepository) ExistsByOrderNumber(ctx context.Context, tenantID uuid.UUID, orderNumber string) (bool, error) {
	args := m.Called(ctx, tenantID, orderNumber)
	return args.Bool(0), args.Error(1)
}

func (m *MockPurchaseOrderRepository) GenerateOrderNumber(ctx context.Context, tenantID uuid.UUID) (string, error) {
	args := m.Called(ctx, tenantID)
	return args.String(0), args.Error(1)
}

func (m *MockPurchaseOrderRepository) ExistsByProduct(ctx context.Context, tenantID, productID uuid.UUID) (bool, error) {
	args := m.Called(ctx, tenantID, productID)
	return args.Bool(0), args.Error(1)
}

// Purchase Order Test helpers
var (
	testPOTenantID    = uuid.New()
	testSupplierID    = uuid.New()
	testPOProductID   = uuid.New()
	testPOWarehouseID = uuid.New()
	testPOOrderID     = uuid.New()
	testPOOrderNumber = "PO-2024-00001"
	testSupplierName  = "Test Supplier"
	testPOProductName = "Test Product"
	testPOProductCode = "TEST-001"
	testPOUnit        = "pcs"
)

func createTestPurchaseOrder() *trade.PurchaseOrder {
	order, _ := trade.NewPurchaseOrder(testPOTenantID, testPOOrderNumber, testSupplierID, testSupplierName)
	return order
}

func createTestPurchaseOrderWithItem() *trade.PurchaseOrder {
	order := createTestPurchaseOrder()
	order.AddItem(testPOProductID, testPOProductName, testPOProductCode, testPOUnit, testPOUnit, decimal.NewFromInt(10), decimal.NewFromInt(1), newMoneyCNY("100"))
	return order
}

func createConfirmedPurchaseOrder() *trade.PurchaseOrder {
	order := createTestPurchaseOrderWithItem()
	order.SetWarehouse(testPOWarehouseID)
	order.Confirm()
	return order
}

// Tests for Create
func TestPurchaseOrderService_Create(t *testing.T) {
	t.Run("create order successfully", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		repo.On("GenerateOrderNumber", mock.Anything, testPOTenantID).Return(testPOOrderNumber, nil)
		repo.On("Save", mock.Anything, mock.AnythingOfType("*trade.PurchaseOrder")).Return(nil)

		req := CreatePurchaseOrderRequest{
			SupplierID:   testSupplierID,
			SupplierName: testSupplierName,
			Items: []CreatePurchaseOrderItemInput{
				{
					ProductID:      testPOProductID,
					ProductName:    testPOProductName,
					ProductCode:    testPOProductCode,
					Unit:           testPOUnit,
					BaseUnit:       testPOUnit,
					Quantity:       decimal.NewFromInt(5),
					ConversionRate: decimal.NewFromInt(1),
					UnitCost:       decimal.NewFromInt(100),
				},
			},
		}

		result, err := service.Create(ctx, testPOTenantID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, testPOOrderNumber, result.OrderNumber)
		assert.Equal(t, testSupplierName, result.SupplierName)
		assert.Equal(t, 1, result.ItemCount)
		assert.Equal(t, "DRAFT", result.Status)
		repo.AssertExpectations(t)
	})

	t.Run("create order with discount", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		repo.On("GenerateOrderNumber", mock.Anything, testPOTenantID).Return(testPOOrderNumber, nil)
		repo.On("Save", mock.Anything, mock.AnythingOfType("*trade.PurchaseOrder")).Return(nil)

		discount := decimal.NewFromInt(50)
		req := CreatePurchaseOrderRequest{
			SupplierID:   testSupplierID,
			SupplierName: testSupplierName,
			Discount:     &discount,
			Items: []CreatePurchaseOrderItemInput{
				{
					ProductID:      testPOProductID,
					ProductName:    testPOProductName,
					ProductCode:    testPOProductCode,
					Unit:           testPOUnit,
					BaseUnit:       testPOUnit,
					Quantity:       decimal.NewFromInt(5),
					ConversionRate: decimal.NewFromInt(1),
					UnitCost:       decimal.NewFromInt(100),
				},
			},
		}

		result, err := service.Create(ctx, testPOTenantID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, discount, result.DiscountAmount)
		assert.Equal(t, decimal.NewFromInt(450), result.PayableAmount) // 500 - 50
		repo.AssertExpectations(t)
	})

	t.Run("create order with warehouse", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		repo.On("GenerateOrderNumber", mock.Anything, testPOTenantID).Return(testPOOrderNumber, nil)
		repo.On("Save", mock.Anything, mock.AnythingOfType("*trade.PurchaseOrder")).Return(nil)

		warehouseID := testPOWarehouseID
		req := CreatePurchaseOrderRequest{
			SupplierID:   testSupplierID,
			SupplierName: testSupplierName,
			WarehouseID:  &warehouseID,
			Items: []CreatePurchaseOrderItemInput{
				{
					ProductID:      testPOProductID,
					ProductName:    testPOProductName,
					ProductCode:    testPOProductCode,
					Unit:           testPOUnit,
					BaseUnit:       testPOUnit,
					Quantity:       decimal.NewFromInt(1),
					ConversionRate: decimal.NewFromInt(1),
					UnitCost:       decimal.NewFromInt(100),
				},
			},
		}

		result, err := service.Create(ctx, testPOTenantID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.WarehouseID)
		assert.Equal(t, warehouseID, *result.WarehouseID)
		repo.AssertExpectations(t)
	})

	t.Run("fail when generate order number fails", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		repo.On("GenerateOrderNumber", mock.Anything, testPOTenantID).Return("", errors.New("db error"))

		req := CreatePurchaseOrderRequest{
			SupplierID:   testSupplierID,
			SupplierName: testSupplierName,
		}

		result, err := service.Create(ctx, testPOTenantID, req)

		assert.Error(t, err)
		assert.Nil(t, result)
		repo.AssertExpectations(t)
	})
}

// Tests for GetByID
func TestPurchaseOrderService_GetByID(t *testing.T) {
	t.Run("get order successfully", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		order := createTestPurchaseOrderWithItem()
		repo.On("FindByIDForTenant", mock.Anything, testPOTenantID, order.ID).Return(order, nil)

		result, err := service.GetByID(ctx, testPOTenantID, order.ID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, order.OrderNumber, result.OrderNumber)
		repo.AssertExpectations(t)
	})

	t.Run("fail when order not found", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		repo.On("FindByIDForTenant", mock.Anything, testPOTenantID, testPOOrderID).Return(nil, shared.ErrNotFound)

		result, err := service.GetByID(ctx, testPOTenantID, testPOOrderID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, shared.ErrNotFound))
		repo.AssertExpectations(t)
	})
}

// Tests for List
func TestPurchaseOrderService_List(t *testing.T) {
	t.Run("list orders with defaults", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		order1 := createTestPurchaseOrderWithItem()
		order2 := createTestPurchaseOrderWithItem()
		orders := []trade.PurchaseOrder{*order1, *order2}

		repo.On("FindAllForTenant", ctx, testPOTenantID, mock.AnythingOfType("shared.Filter")).Return(orders, nil)
		repo.On("CountForTenant", ctx, testPOTenantID, mock.AnythingOfType("shared.Filter")).Return(int64(2), nil)

		result, total, err := service.List(ctx, testPOTenantID, PurchaseOrderListFilter{})

		assert.NoError(t, err)
		assert.Equal(t, 2, len(result))
		assert.Equal(t, int64(2), total)
		repo.AssertExpectations(t)
	})

	t.Run("list orders with supplier filter", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		supplierID := testSupplierID
		repo.On("FindAllForTenant", ctx, testPOTenantID, mock.MatchedBy(func(f shared.Filter) bool {
			return f.Filters["supplier_id"] == supplierID
		})).Return([]trade.PurchaseOrder{}, nil)
		repo.On("CountForTenant", ctx, testPOTenantID, mock.AnythingOfType("shared.Filter")).Return(int64(0), nil)

		_, _, err := service.List(ctx, testPOTenantID, PurchaseOrderListFilter{SupplierID: &supplierID})

		assert.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

// Tests for ListPendingReceipt
func TestPurchaseOrderService_ListPendingReceipt(t *testing.T) {
	t.Run("list pending receipt orders", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		order := createConfirmedPurchaseOrder()
		orders := []trade.PurchaseOrder{*order}

		repo.On("FindPendingReceipt", ctx, testPOTenantID, mock.AnythingOfType("shared.Filter")).Return(orders, nil)
		repo.On("CountPendingReceipt", ctx, testPOTenantID).Return(int64(1), nil)

		result, total, err := service.ListPendingReceipt(ctx, testPOTenantID, PurchaseOrderListFilter{})

		assert.NoError(t, err)
		assert.Equal(t, 1, len(result))
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "CONFIRMED", result[0].Status)
		repo.AssertExpectations(t)
	})
}

// Tests for AddItem
func TestPurchaseOrderService_AddItem(t *testing.T) {
	t.Run("add item successfully", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		order := createTestPurchaseOrder()
		repo.On("FindByIDForTenant", mock.Anything, testPOTenantID, order.ID).Return(order, nil)
		repo.On("SaveWithLock", mock.Anything, order).Return(nil)

		req := AddPurchaseOrderItemRequest{
			ProductID:   testPOProductID,
			ProductName: testPOProductName,
			ProductCode: testPOProductCode,
			Unit:        testPOUnit,
			Quantity:    decimal.NewFromInt(5),
			UnitCost:    decimal.NewFromInt(100),
		}

		result, err := service.AddItem(ctx, testPOTenantID, order.ID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.ItemCount)
		assert.Equal(t, decimal.NewFromInt(500), result.TotalAmount)
		repo.AssertExpectations(t)
	})

	t.Run("fail to add item to non-draft order", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		order := createConfirmedPurchaseOrder() // CONFIRMED status
		repo.On("FindByIDForTenant", mock.Anything, testPOTenantID, order.ID).Return(order, nil)

		req := AddPurchaseOrderItemRequest{
			ProductID:   uuid.New(),
			ProductName: "New Product",
			ProductCode: "NEW-001",
			Unit:        testPOUnit,
			Quantity:    decimal.NewFromInt(5),
			UnitCost:    decimal.NewFromInt(100),
		}

		result, err := service.AddItem(ctx, testPOTenantID, order.ID, req)

		assert.Error(t, err)
		assert.Nil(t, result)
		repo.AssertExpectations(t)
	})
}

// Tests for Confirm
func TestPurchaseOrderService_Confirm(t *testing.T) {
	t.Run("confirm order successfully", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		order := createTestPurchaseOrderWithItem()
		repo.On("FindByIDForTenant", mock.Anything, testPOTenantID, order.ID).Return(order, nil)
		repo.On("SaveWithLock", mock.Anything, order).Return(nil)

		result, err := service.Confirm(ctx, testPOTenantID, order.ID, ConfirmPurchaseOrderRequest{})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "CONFIRMED", result.Status)
		assert.NotNil(t, result.ConfirmedAt)
		repo.AssertExpectations(t)
	})

	t.Run("confirm order with warehouse", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		order := createTestPurchaseOrderWithItem()
		warehouseID := testPOWarehouseID
		repo.On("FindByIDForTenant", mock.Anything, testPOTenantID, order.ID).Return(order, nil)
		repo.On("SaveWithLock", mock.Anything, order).Return(nil)

		result, err := service.Confirm(ctx, testPOTenantID, order.ID, ConfirmPurchaseOrderRequest{WarehouseID: &warehouseID})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "CONFIRMED", result.Status)
		assert.NotNil(t, result.WarehouseID)
		assert.Equal(t, warehouseID, *result.WarehouseID)
		repo.AssertExpectations(t)
	})

	t.Run("fail to confirm order without items", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		order := createTestPurchaseOrder() // No items
		repo.On("FindByIDForTenant", mock.Anything, testPOTenantID, order.ID).Return(order, nil)

		result, err := service.Confirm(ctx, testPOTenantID, order.ID, ConfirmPurchaseOrderRequest{})

		assert.Error(t, err)
		assert.Nil(t, result)
		repo.AssertExpectations(t)
	})
}

// Tests for Receive
func TestPurchaseOrderService_Receive(t *testing.T) {
	t.Run("receive all items successfully", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		order := createConfirmedPurchaseOrder()
		repo.On("FindByIDForTenant", mock.Anything, testPOTenantID, order.ID).Return(order, nil)
		repo.On("SaveWithLock", mock.Anything, order).Return(nil)

		req := ReceivePurchaseOrderRequest{
			Items: []ReceiveItemInput{
				{
					ProductID:   testPOProductID,
					Quantity:    decimal.NewFromInt(10), // Receive all
					BatchNumber: "BATCH-001",
				},
			},
		}

		result, err := service.Receive(ctx, testPOTenantID, order.ID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.IsFullyReceived)
		assert.Equal(t, "COMPLETED", result.Order.Status)
		assert.Equal(t, 1, len(result.ReceivedItems))
		assert.Equal(t, decimal.NewFromInt(10), result.ReceivedItems[0].Quantity)
		assert.Equal(t, "BATCH-001", result.ReceivedItems[0].BatchNumber)
		repo.AssertExpectations(t)
	})

	t.Run("receive partial items", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		order := createConfirmedPurchaseOrder()
		repo.On("FindByIDForTenant", mock.Anything, testPOTenantID, order.ID).Return(order, nil)
		repo.On("SaveWithLock", mock.Anything, order).Return(nil)

		req := ReceivePurchaseOrderRequest{
			Items: []ReceiveItemInput{
				{
					ProductID: testPOProductID,
					Quantity:  decimal.NewFromInt(5), // Partial
				},
			},
		}

		result, err := service.Receive(ctx, testPOTenantID, order.ID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsFullyReceived)
		assert.Equal(t, "PARTIAL_RECEIVED", result.Order.Status)
		assert.True(t, result.Order.ReceiveProgress.Equal(decimal.NewFromInt(50))) // 50%
		repo.AssertExpectations(t)
	})

	t.Run("receive with cost override", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		order := createConfirmedPurchaseOrder()
		repo.On("FindByIDForTenant", mock.Anything, testPOTenantID, order.ID).Return(order, nil)
		repo.On("SaveWithLock", mock.Anything, order).Return(nil)

		overrideCost := decimal.NewFromInt(120) // Different from original 100
		req := ReceivePurchaseOrderRequest{
			Items: []ReceiveItemInput{
				{
					ProductID: testPOProductID,
					Quantity:  decimal.NewFromInt(10),
					UnitCost:  &overrideCost,
				},
			},
		}

		result, err := service.Receive(ctx, testPOTenantID, order.ID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, overrideCost, result.ReceivedItems[0].UnitCost)
		repo.AssertExpectations(t)
	})

	t.Run("receive with expiry date", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		order := createConfirmedPurchaseOrder()
		repo.On("FindByIDForTenant", mock.Anything, testPOTenantID, order.ID).Return(order, nil)
		repo.On("SaveWithLock", mock.Anything, order).Return(nil)

		expiryDate := time.Now().AddDate(1, 0, 0) // 1 year from now
		req := ReceivePurchaseOrderRequest{
			Items: []ReceiveItemInput{
				{
					ProductID:  testPOProductID,
					Quantity:   decimal.NewFromInt(10),
					ExpiryDate: &expiryDate,
				},
			},
		}

		result, err := service.Receive(ctx, testPOTenantID, order.ID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.ReceivedItems[0].ExpiryDate)
		repo.AssertExpectations(t)
	})

	t.Run("fail to receive without warehouse", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		// Create order without warehouse
		order, _ := trade.NewPurchaseOrder(testPOTenantID, testPOOrderNumber, testSupplierID, testSupplierName)
		order.AddItem(testPOProductID, testPOProductName, testPOProductCode, testPOUnit, testPOUnit, decimal.NewFromInt(10), decimal.NewFromInt(1), valueobject.NewMoneyCNY(decimal.NewFromInt(100)))
		// Confirm with warehouse then clear it (simulate edge case)
		order.SetWarehouse(testPOWarehouseID)
		order.Confirm()
		// Manually clear warehouse for test
		order.WarehouseID = nil

		repo.On("FindByIDForTenant", mock.Anything, testPOTenantID, order.ID).Return(order, nil)

		req := ReceivePurchaseOrderRequest{
			Items: []ReceiveItemInput{
				{
					ProductID: testPOProductID,
					Quantity:  decimal.NewFromInt(5),
				},
			},
		}

		result, err := service.Receive(ctx, testPOTenantID, order.ID, req)

		assert.Error(t, err)
		assert.Nil(t, result)
		repo.AssertExpectations(t)
	})

	t.Run("fail to receive from draft order", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		order := createTestPurchaseOrderWithItem() // DRAFT status
		repo.On("FindByIDForTenant", mock.Anything, testPOTenantID, order.ID).Return(order, nil)

		req := ReceivePurchaseOrderRequest{
			Items: []ReceiveItemInput{
				{
					ProductID: testPOProductID,
					Quantity:  decimal.NewFromInt(5),
				},
			},
		}

		result, err := service.Receive(ctx, testPOTenantID, order.ID, req)

		assert.Error(t, err)
		assert.Nil(t, result)
		repo.AssertExpectations(t)
	})

	t.Run("fail to over-receive", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		order := createConfirmedPurchaseOrder()
		repo.On("FindByIDForTenant", mock.Anything, testPOTenantID, order.ID).Return(order, nil)

		req := ReceivePurchaseOrderRequest{
			Items: []ReceiveItemInput{
				{
					ProductID: testPOProductID,
					Quantity:  decimal.NewFromInt(15), // More than ordered (10)
				},
			},
		}

		result, err := service.Receive(ctx, testPOTenantID, order.ID, req)

		assert.Error(t, err)
		assert.Nil(t, result)
		repo.AssertExpectations(t)
	})
}

// Tests for Cancel
func TestPurchaseOrderService_Cancel(t *testing.T) {
	t.Run("cancel draft order successfully", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		order := createTestPurchaseOrderWithItem()
		repo.On("FindByIDForTenant", mock.Anything, testPOTenantID, order.ID).Return(order, nil)
		repo.On("SaveWithLock", mock.Anything, order).Return(nil)

		result, err := service.Cancel(ctx, testPOTenantID, order.ID, CancelPurchaseOrderRequest{Reason: "Supplier issue"})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "CANCELLED", result.Status)
		assert.Equal(t, "Supplier issue", result.CancelReason)
		assert.NotNil(t, result.CancelledAt)
		repo.AssertExpectations(t)
	})

	t.Run("cancel confirmed order successfully", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		order := createConfirmedPurchaseOrder()
		repo.On("FindByIDForTenant", mock.Anything, testPOTenantID, order.ID).Return(order, nil)
		repo.On("SaveWithLock", mock.Anything, order).Return(nil)

		result, err := service.Cancel(ctx, testPOTenantID, order.ID, CancelPurchaseOrderRequest{Reason: "Price changed"})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "CANCELLED", result.Status)
		repo.AssertExpectations(t)
	})

	t.Run("fail to cancel partial received order", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		order := createConfirmedPurchaseOrder()
		// Partially receive
		order.Receive([]trade.ReceiveItem{{ProductID: testPOProductID, Quantity: decimal.NewFromInt(5)}})

		repo.On("FindByIDForTenant", mock.Anything, testPOTenantID, order.ID).Return(order, nil)

		result, err := service.Cancel(ctx, testPOTenantID, order.ID, CancelPurchaseOrderRequest{Reason: "Too late"})

		assert.Error(t, err)
		assert.Nil(t, result)
		repo.AssertExpectations(t)
	})
}

// Tests for Delete
func TestPurchaseOrderService_Delete(t *testing.T) {
	t.Run("delete draft order successfully", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		order := createTestPurchaseOrder()
		repo.On("FindByIDForTenant", mock.Anything, testPOTenantID, order.ID).Return(order, nil)
		repo.On("DeleteForTenant", ctx, testPOTenantID, order.ID).Return(nil)

		err := service.Delete(ctx, testPOTenantID, order.ID)

		assert.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("fail to delete non-draft order", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		order := createConfirmedPurchaseOrder()
		repo.On("FindByIDForTenant", mock.Anything, testPOTenantID, order.ID).Return(order, nil)

		err := service.Delete(ctx, testPOTenantID, order.ID)

		assert.Error(t, err)
		repo.AssertExpectations(t)
	})
}

// Tests for GetStatusSummary
func TestPurchaseOrderService_GetStatusSummary(t *testing.T) {
	t.Run("get status summary successfully", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		repo.On("CountByStatus", ctx, testPOTenantID, trade.PurchaseOrderStatusDraft).Return(int64(5), nil)
		repo.On("CountByStatus", ctx, testPOTenantID, trade.PurchaseOrderStatusConfirmed).Return(int64(10), nil)
		repo.On("CountByStatus", ctx, testPOTenantID, trade.PurchaseOrderStatusPartialReceived).Return(int64(3), nil)
		repo.On("CountByStatus", ctx, testPOTenantID, trade.PurchaseOrderStatusCompleted).Return(int64(100), nil)
		repo.On("CountByStatus", ctx, testPOTenantID, trade.PurchaseOrderStatusCancelled).Return(int64(2), nil)
		repo.On("CountPendingReceipt", ctx, testPOTenantID).Return(int64(13), nil)

		result, err := service.GetStatusSummary(ctx, testPOTenantID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(5), result.Draft)
		assert.Equal(t, int64(10), result.Confirmed)
		assert.Equal(t, int64(3), result.PartialReceived)
		assert.Equal(t, int64(100), result.Completed)
		assert.Equal(t, int64(2), result.Cancelled)
		assert.Equal(t, int64(120), result.Total)
		assert.Equal(t, int64(13), result.PendingReceipt)
		repo.AssertExpectations(t)
	})
}

// Tests for GetReceivableItems
func TestPurchaseOrderService_GetReceivableItems(t *testing.T) {
	t.Run("get receivable items successfully", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		order := createConfirmedPurchaseOrder()
		repo.On("FindByIDForTenant", mock.Anything, testPOTenantID, order.ID).Return(order, nil)

		result, err := service.GetReceivableItems(ctx, testPOTenantID, order.ID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, len(result))
		assert.Equal(t, decimal.NewFromInt(10), result[0].RemainingQuantity)
		repo.AssertExpectations(t)
	})

	t.Run("fail for draft order", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		order := createTestPurchaseOrderWithItem() // DRAFT
		repo.On("FindByIDForTenant", mock.Anything, testPOTenantID, order.ID).Return(order, nil)

		result, err := service.GetReceivableItems(ctx, testPOTenantID, order.ID)

		assert.Error(t, err)
		assert.Nil(t, result)
		repo.AssertExpectations(t)
	})
}

// Tests for Update
func TestPurchaseOrderService_Update(t *testing.T) {
	t.Run("update order successfully", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		order := createTestPurchaseOrderWithItem()
		repo.On("FindByIDForTenant", mock.Anything, testPOTenantID, order.ID).Return(order, nil)
		repo.On("SaveWithLock", mock.Anything, order).Return(nil)

		warehouseID := testPOWarehouseID
		newRemark := "Updated remark"
		req := UpdatePurchaseOrderRequest{
			WarehouseID: &warehouseID,
			Remark:      &newRemark,
		}

		result, err := service.Update(ctx, testPOTenantID, order.ID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, warehouseID, *result.WarehouseID)
		assert.Equal(t, newRemark, result.Remark)
		repo.AssertExpectations(t)
	})

	t.Run("fail to update non-draft order", func(t *testing.T) {
		repo := new(MockPurchaseOrderRepository)
		service := NewPurchaseOrderService(repo)
		ctx := context.Background()

		order := createConfirmedPurchaseOrder()
		repo.On("FindByIDForTenant", mock.Anything, testPOTenantID, order.ID).Return(order, nil)

		newRemark := "Cannot update"
		req := UpdatePurchaseOrderRequest{
			Remark: &newRemark,
		}

		result, err := service.Update(ctx, testPOTenantID, order.ID, req)

		assert.Error(t, err)
		assert.Nil(t, result)
		repo.AssertExpectations(t)
	})
}
