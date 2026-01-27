package trade

import (
	"context"
	"errors"
	"testing"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSalesOrderRepository is a mock implementation of SalesOrderRepository
type MockSalesOrderRepository struct {
	mock.Mock
}

func (m *MockSalesOrderRepository) FindByID(ctx context.Context, id uuid.UUID) (*trade.SalesOrder, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*trade.SalesOrder), args.Error(1)
}

func (m *MockSalesOrderRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*trade.SalesOrder, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*trade.SalesOrder), args.Error(1)
}

func (m *MockSalesOrderRepository) FindByOrderNumber(ctx context.Context, tenantID uuid.UUID, orderNumber string) (*trade.SalesOrder, error) {
	args := m.Called(ctx, tenantID, orderNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*trade.SalesOrder), args.Error(1)
}

func (m *MockSalesOrderRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]trade.SalesOrder, error) {
	args := m.Called(ctx, tenantID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.SalesOrder), args.Error(1)
}

func (m *MockSalesOrderRepository) FindByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, filter shared.Filter) ([]trade.SalesOrder, error) {
	args := m.Called(ctx, tenantID, customerID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.SalesOrder), args.Error(1)
}

func (m *MockSalesOrderRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status trade.OrderStatus, filter shared.Filter) ([]trade.SalesOrder, error) {
	args := m.Called(ctx, tenantID, status, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.SalesOrder), args.Error(1)
}

func (m *MockSalesOrderRepository) FindByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID, filter shared.Filter) ([]trade.SalesOrder, error) {
	args := m.Called(ctx, tenantID, warehouseID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.SalesOrder), args.Error(1)
}

func (m *MockSalesOrderRepository) Save(ctx context.Context, order *trade.SalesOrder) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockSalesOrderRepository) SaveWithLock(ctx context.Context, order *trade.SalesOrder) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockSalesOrderRepository) SaveWithLockAndEvents(ctx context.Context, order *trade.SalesOrder, events []shared.DomainEvent) error {
	args := m.Called(ctx, order, events)
	return args.Error(0)
}

func (m *MockSalesOrderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSalesOrderRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *MockSalesOrderRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSalesOrderRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status trade.OrderStatus) (int64, error) {
	args := m.Called(ctx, tenantID, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSalesOrderRepository) CountByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID, customerID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSalesOrderRepository) CountIncompleteByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID, customerID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSalesOrderRepository) ExistsByOrderNumber(ctx context.Context, tenantID uuid.UUID, orderNumber string) (bool, error) {
	args := m.Called(ctx, tenantID, orderNumber)
	return args.Bool(0), args.Error(1)
}

func (m *MockSalesOrderRepository) GenerateOrderNumber(ctx context.Context, tenantID uuid.UUID) (string, error) {
	args := m.Called(ctx, tenantID)
	return args.String(0), args.Error(1)
}

// Test helpers
var (
	testTenantID     = uuid.New()
	testCustomerID   = uuid.New()
	testProductID    = uuid.New()
	testWarehouseID  = uuid.New()
	testOrderID      = uuid.New()
	testOrderNumber  = "SO-2024-00001"
	testCustomerName = "Test Customer"
	testProductName  = "Test Product"
	testProductCode  = "TEST-001"
	testUnit         = "pcs"
)

func createTestOrder() *trade.SalesOrder {
	order, _ := trade.NewSalesOrder(testTenantID, testOrderNumber, testCustomerID, testCustomerName)
	return order
}

func createTestOrderWithItem() *trade.SalesOrder {
	order := createTestOrder()
	order.AddItem(testProductID, testProductName, testProductCode, testUnit, testUnit, decimal.NewFromInt(10), decimal.NewFromInt(1), newMoneyCNY("100"))
	return order
}

func newMoneyCNY(amount string) valueobject.Money {
	amt, _ := decimal.NewFromString(amount)
	return valueobject.NewMoneyCNY(amt)
}

// Tests for Create
func TestSalesOrderService_Create(t *testing.T) {
	t.Run("create order successfully", func(t *testing.T) {
		repo := new(MockSalesOrderRepository)
		service := NewSalesOrderService(repo)
		ctx := context.Background()

		repo.On("GenerateOrderNumber", ctx, testTenantID).Return(testOrderNumber, nil)
		repo.On("Save", ctx, mock.AnythingOfType("*trade.SalesOrder")).Return(nil)

		req := CreateSalesOrderRequest{
			CustomerID:   testCustomerID,
			CustomerName: testCustomerName,
			Items: []CreateSalesOrderItemInput{
				{
					ProductID:   testProductID,
					ProductName: testProductName,
					ProductCode: testProductCode,
					Unit:        testUnit,
					Quantity:    decimal.NewFromInt(5),
					UnitPrice:   decimal.NewFromInt(100),
				},
			},
		}

		result, err := service.Create(ctx, testTenantID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, testOrderNumber, result.OrderNumber)
		assert.Equal(t, testCustomerName, result.CustomerName)
		assert.Equal(t, 1, result.ItemCount)
		assert.Equal(t, "draft", result.Status)
		repo.AssertExpectations(t)
	})

	t.Run("create order with discount", func(t *testing.T) {
		repo := new(MockSalesOrderRepository)
		service := NewSalesOrderService(repo)
		ctx := context.Background()

		repo.On("GenerateOrderNumber", ctx, testTenantID).Return(testOrderNumber, nil)
		repo.On("Save", ctx, mock.AnythingOfType("*trade.SalesOrder")).Return(nil)

		discount := decimal.NewFromInt(50)
		req := CreateSalesOrderRequest{
			CustomerID:   testCustomerID,
			CustomerName: testCustomerName,
			Discount:     &discount,
			Items: []CreateSalesOrderItemInput{
				{
					ProductID:   testProductID,
					ProductName: testProductName,
					ProductCode: testProductCode,
					Unit:        testUnit,
					Quantity:    decimal.NewFromInt(5),
					UnitPrice:   decimal.NewFromInt(100),
				},
			},
		}

		result, err := service.Create(ctx, testTenantID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, discount, result.DiscountAmount)
		assert.Equal(t, decimal.NewFromInt(450), result.PayableAmount) // 500 - 50
		repo.AssertExpectations(t)
	})

	t.Run("create order with warehouse", func(t *testing.T) {
		repo := new(MockSalesOrderRepository)
		service := NewSalesOrderService(repo)
		ctx := context.Background()

		repo.On("GenerateOrderNumber", ctx, testTenantID).Return(testOrderNumber, nil)
		repo.On("Save", ctx, mock.AnythingOfType("*trade.SalesOrder")).Return(nil)

		warehouseID := testWarehouseID
		req := CreateSalesOrderRequest{
			CustomerID:   testCustomerID,
			CustomerName: testCustomerName,
			WarehouseID:  &warehouseID,
			Items: []CreateSalesOrderItemInput{
				{
					ProductID:   testProductID,
					ProductName: testProductName,
					ProductCode: testProductCode,
					Unit:        testUnit,
					Quantity:    decimal.NewFromInt(1),
					UnitPrice:   decimal.NewFromInt(100),
				},
			},
		}

		result, err := service.Create(ctx, testTenantID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.WarehouseID)
		assert.Equal(t, warehouseID, *result.WarehouseID)
		repo.AssertExpectations(t)
	})

	t.Run("fail when generate order number fails", func(t *testing.T) {
		repo := new(MockSalesOrderRepository)
		service := NewSalesOrderService(repo)
		ctx := context.Background()

		repo.On("GenerateOrderNumber", ctx, testTenantID).Return("", errors.New("db error"))

		req := CreateSalesOrderRequest{
			CustomerID:   testCustomerID,
			CustomerName: testCustomerName,
		}

		result, err := service.Create(ctx, testTenantID, req)

		assert.Error(t, err)
		assert.Nil(t, result)
		repo.AssertExpectations(t)
	})
}

// Tests for GetByID
func TestSalesOrderService_GetByID(t *testing.T) {
	t.Run("get order successfully", func(t *testing.T) {
		repo := new(MockSalesOrderRepository)
		service := NewSalesOrderService(repo)
		ctx := context.Background()

		order := createTestOrderWithItem()
		repo.On("FindByIDForTenant", ctx, testTenantID, order.ID).Return(order, nil)

		result, err := service.GetByID(ctx, testTenantID, order.ID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, order.OrderNumber, result.OrderNumber)
		repo.AssertExpectations(t)
	})

	t.Run("fail when order not found", func(t *testing.T) {
		repo := new(MockSalesOrderRepository)
		service := NewSalesOrderService(repo)
		ctx := context.Background()

		repo.On("FindByIDForTenant", ctx, testTenantID, testOrderID).Return(nil, shared.ErrNotFound)

		result, err := service.GetByID(ctx, testTenantID, testOrderID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, shared.ErrNotFound))
		repo.AssertExpectations(t)
	})
}

// Tests for List
func TestSalesOrderService_List(t *testing.T) {
	t.Run("list orders with defaults", func(t *testing.T) {
		repo := new(MockSalesOrderRepository)
		service := NewSalesOrderService(repo)
		ctx := context.Background()

		order1 := createTestOrderWithItem()
		order2 := createTestOrderWithItem()
		orders := []trade.SalesOrder{*order1, *order2}

		repo.On("FindAllForTenant", ctx, testTenantID, mock.AnythingOfType("shared.Filter")).Return(orders, nil)
		repo.On("CountForTenant", ctx, testTenantID, mock.AnythingOfType("shared.Filter")).Return(int64(2), nil)

		result, total, err := service.List(ctx, testTenantID, SalesOrderListFilter{})

		assert.NoError(t, err)
		assert.Equal(t, 2, len(result))
		assert.Equal(t, int64(2), total)
		repo.AssertExpectations(t)
	})

	t.Run("list orders with customer filter", func(t *testing.T) {
		repo := new(MockSalesOrderRepository)
		service := NewSalesOrderService(repo)
		ctx := context.Background()

		customerID := testCustomerID
		repo.On("FindAllForTenant", ctx, testTenantID, mock.MatchedBy(func(f shared.Filter) bool {
			return f.Filters["customer_id"] == customerID
		})).Return([]trade.SalesOrder{}, nil)
		repo.On("CountForTenant", ctx, testTenantID, mock.AnythingOfType("shared.Filter")).Return(int64(0), nil)

		_, _, err := service.List(ctx, testTenantID, SalesOrderListFilter{CustomerID: &customerID})

		assert.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

// Tests for AddItem
func TestSalesOrderService_AddItem(t *testing.T) {
	t.Run("add item successfully", func(t *testing.T) {
		repo := new(MockSalesOrderRepository)
		service := NewSalesOrderService(repo)
		ctx := context.Background()

		order := createTestOrder()
		repo.On("FindByIDForTenant", ctx, testTenantID, order.ID).Return(order, nil)
		repo.On("SaveWithLock", ctx, order).Return(nil)

		req := AddOrderItemRequest{
			ProductID:   testProductID,
			ProductName: testProductName,
			ProductCode: testProductCode,
			Unit:        testUnit,
			Quantity:    decimal.NewFromInt(5),
			UnitPrice:   decimal.NewFromInt(100),
		}

		result, err := service.AddItem(ctx, testTenantID, order.ID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.ItemCount)
		assert.Equal(t, decimal.NewFromInt(500), result.TotalAmount)
		repo.AssertExpectations(t)
	})

	t.Run("fail to add item to non-draft order", func(t *testing.T) {
		repo := new(MockSalesOrderRepository)
		service := NewSalesOrderService(repo)
		ctx := context.Background()

		order := createTestOrderWithItem()
		order.Confirm() // Move to CONFIRMED status
		repo.On("FindByIDForTenant", ctx, testTenantID, order.ID).Return(order, nil)

		req := AddOrderItemRequest{
			ProductID:   uuid.New(),
			ProductName: "New Product",
			ProductCode: "NEW-001",
			Unit:        testUnit,
			Quantity:    decimal.NewFromInt(5),
			UnitPrice:   decimal.NewFromInt(100),
		}

		result, err := service.AddItem(ctx, testTenantID, order.ID, req)

		assert.Error(t, err)
		assert.Nil(t, result)
		repo.AssertExpectations(t)
	})
}

// Tests for Confirm
func TestSalesOrderService_Confirm(t *testing.T) {
	t.Run("confirm order successfully", func(t *testing.T) {
		repo := new(MockSalesOrderRepository)
		service := NewSalesOrderService(repo)
		ctx := context.Background()

		order := createTestOrderWithItem()
		repo.On("FindByIDForTenant", ctx, testTenantID, order.ID).Return(order, nil)
		repo.On("SaveWithLock", ctx, order).Return(nil)

		result, err := service.Confirm(ctx, testTenantID, order.ID, ConfirmOrderRequest{})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "confirmed", result.Status)
		assert.NotNil(t, result.ConfirmedAt)
		repo.AssertExpectations(t)
	})

	t.Run("confirm order with warehouse", func(t *testing.T) {
		repo := new(MockSalesOrderRepository)
		service := NewSalesOrderService(repo)
		ctx := context.Background()

		order := createTestOrderWithItem()
		warehouseID := testWarehouseID
		repo.On("FindByIDForTenant", ctx, testTenantID, order.ID).Return(order, nil)
		repo.On("SaveWithLock", ctx, order).Return(nil)

		result, err := service.Confirm(ctx, testTenantID, order.ID, ConfirmOrderRequest{WarehouseID: &warehouseID})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "confirmed", result.Status)
		assert.NotNil(t, result.WarehouseID)
		assert.Equal(t, warehouseID, *result.WarehouseID)
		repo.AssertExpectations(t)
	})

	t.Run("fail to confirm order without items", func(t *testing.T) {
		repo := new(MockSalesOrderRepository)
		service := NewSalesOrderService(repo)
		ctx := context.Background()

		order := createTestOrder() // No items
		repo.On("FindByIDForTenant", ctx, testTenantID, order.ID).Return(order, nil)

		result, err := service.Confirm(ctx, testTenantID, order.ID, ConfirmOrderRequest{})

		assert.Error(t, err)
		assert.Nil(t, result)
		repo.AssertExpectations(t)
	})
}

// Tests for Ship
func TestSalesOrderService_Ship(t *testing.T) {
	t.Run("ship order successfully", func(t *testing.T) {
		repo := new(MockSalesOrderRepository)
		service := NewSalesOrderService(repo)
		ctx := context.Background()

		order := createTestOrderWithItem()
		order.SetWarehouse(testWarehouseID)
		order.Confirm()
		repo.On("FindByIDForTenant", ctx, testTenantID, order.ID).Return(order, nil)
		repo.On("SaveWithLock", ctx, order).Return(nil)

		result, err := service.Ship(ctx, testTenantID, order.ID, ShipOrderRequest{})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "shipped", result.Status)
		assert.NotNil(t, result.ShippedAt)
		repo.AssertExpectations(t)
	})

	t.Run("fail to ship order without warehouse", func(t *testing.T) {
		repo := new(MockSalesOrderRepository)
		service := NewSalesOrderService(repo)
		ctx := context.Background()

		order := createTestOrderWithItem()
		order.Confirm() // Confirm without warehouse
		repo.On("FindByIDForTenant", ctx, testTenantID, order.ID).Return(order, nil)

		result, err := service.Ship(ctx, testTenantID, order.ID, ShipOrderRequest{})

		assert.Error(t, err)
		assert.Nil(t, result)
		repo.AssertExpectations(t)
	})
}

// Tests for Complete
func TestSalesOrderService_Complete(t *testing.T) {
	t.Run("complete order successfully", func(t *testing.T) {
		repo := new(MockSalesOrderRepository)
		service := NewSalesOrderService(repo)
		ctx := context.Background()

		order := createTestOrderWithItem()
		order.SetWarehouse(testWarehouseID)
		order.Confirm()
		order.Ship()
		repo.On("FindByIDForTenant", ctx, testTenantID, order.ID).Return(order, nil)
		repo.On("SaveWithLock", ctx, order).Return(nil)

		result, err := service.Complete(ctx, testTenantID, order.ID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "completed", result.Status)
		assert.NotNil(t, result.CompletedAt)
		repo.AssertExpectations(t)
	})
}

// Tests for Cancel
func TestSalesOrderService_Cancel(t *testing.T) {
	t.Run("cancel draft order successfully", func(t *testing.T) {
		repo := new(MockSalesOrderRepository)
		service := NewSalesOrderService(repo)
		ctx := context.Background()

		order := createTestOrderWithItem()
		repo.On("FindByIDForTenant", ctx, testTenantID, order.ID).Return(order, nil)
		repo.On("SaveWithLock", ctx, order).Return(nil)

		result, err := service.Cancel(ctx, testTenantID, order.ID, CancelOrderRequest{Reason: "Customer request"})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "cancelled", result.Status)
		assert.Equal(t, "Customer request", result.CancelReason)
		assert.NotNil(t, result.CancelledAt)
		repo.AssertExpectations(t)
	})

	t.Run("cancel confirmed order successfully", func(t *testing.T) {
		repo := new(MockSalesOrderRepository)
		service := NewSalesOrderService(repo)
		ctx := context.Background()

		order := createTestOrderWithItem()
		order.Confirm()
		repo.On("FindByIDForTenant", ctx, testTenantID, order.ID).Return(order, nil)
		repo.On("SaveWithLock", ctx, order).Return(nil)

		result, err := service.Cancel(ctx, testTenantID, order.ID, CancelOrderRequest{Reason: "Out of stock"})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "cancelled", result.Status)
		repo.AssertExpectations(t)
	})

	t.Run("fail to cancel shipped order", func(t *testing.T) {
		repo := new(MockSalesOrderRepository)
		service := NewSalesOrderService(repo)
		ctx := context.Background()

		order := createTestOrderWithItem()
		order.SetWarehouse(testWarehouseID)
		order.Confirm()
		order.Ship()
		repo.On("FindByIDForTenant", ctx, testTenantID, order.ID).Return(order, nil)

		result, err := service.Cancel(ctx, testTenantID, order.ID, CancelOrderRequest{Reason: "Too late"})

		assert.Error(t, err)
		assert.Nil(t, result)
		repo.AssertExpectations(t)
	})
}

// Tests for Delete
func TestSalesOrderService_Delete(t *testing.T) {
	t.Run("delete draft order successfully", func(t *testing.T) {
		repo := new(MockSalesOrderRepository)
		service := NewSalesOrderService(repo)
		ctx := context.Background()

		order := createTestOrder()
		repo.On("FindByIDForTenant", ctx, testTenantID, order.ID).Return(order, nil)
		repo.On("DeleteForTenant", ctx, testTenantID, order.ID).Return(nil)

		err := service.Delete(ctx, testTenantID, order.ID)

		assert.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("fail to delete non-draft order", func(t *testing.T) {
		repo := new(MockSalesOrderRepository)
		service := NewSalesOrderService(repo)
		ctx := context.Background()

		order := createTestOrderWithItem()
		order.Confirm()
		repo.On("FindByIDForTenant", ctx, testTenantID, order.ID).Return(order, nil)

		err := service.Delete(ctx, testTenantID, order.ID)

		assert.Error(t, err)
		repo.AssertExpectations(t)
	})
}

// Tests for GetStatusSummary
func TestSalesOrderService_GetStatusSummary(t *testing.T) {
	t.Run("get status summary successfully", func(t *testing.T) {
		repo := new(MockSalesOrderRepository)
		service := NewSalesOrderService(repo)
		ctx := context.Background()

		repo.On("CountByStatus", ctx, testTenantID, trade.OrderStatusDraft).Return(int64(5), nil)
		repo.On("CountByStatus", ctx, testTenantID, trade.OrderStatusConfirmed).Return(int64(10), nil)
		repo.On("CountByStatus", ctx, testTenantID, trade.OrderStatusShipped).Return(int64(3), nil)
		repo.On("CountByStatus", ctx, testTenantID, trade.OrderStatusCompleted).Return(int64(100), nil)
		repo.On("CountByStatus", ctx, testTenantID, trade.OrderStatusCancelled).Return(int64(2), nil)

		result, err := service.GetStatusSummary(ctx, testTenantID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(5), result.Draft)
		assert.Equal(t, int64(10), result.Confirmed)
		assert.Equal(t, int64(3), result.Shipped)
		assert.Equal(t, int64(100), result.Completed)
		assert.Equal(t, int64(2), result.Cancelled)
		assert.Equal(t, int64(120), result.Total)
		repo.AssertExpectations(t)
	})
}
