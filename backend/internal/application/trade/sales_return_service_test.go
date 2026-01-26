package trade

import (
	"context"
	"testing"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSalesReturnRepository is a mock implementation of SalesReturnRepository
type MockSalesReturnRepository struct {
	mock.Mock
}

func (m *MockSalesReturnRepository) FindByID(ctx context.Context, id uuid.UUID) (*trade.SalesReturn, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*trade.SalesReturn), args.Error(1)
}

func (m *MockSalesReturnRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*trade.SalesReturn, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*trade.SalesReturn), args.Error(1)
}

func (m *MockSalesReturnRepository) FindByReturnNumber(ctx context.Context, tenantID uuid.UUID, returnNumber string) (*trade.SalesReturn, error) {
	args := m.Called(ctx, tenantID, returnNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*trade.SalesReturn), args.Error(1)
}

func (m *MockSalesReturnRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]trade.SalesReturn, error) {
	args := m.Called(ctx, tenantID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.SalesReturn), args.Error(1)
}

func (m *MockSalesReturnRepository) FindByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, filter shared.Filter) ([]trade.SalesReturn, error) {
	args := m.Called(ctx, tenantID, customerID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.SalesReturn), args.Error(1)
}

func (m *MockSalesReturnRepository) FindBySalesOrder(ctx context.Context, tenantID, salesOrderID uuid.UUID) ([]trade.SalesReturn, error) {
	args := m.Called(ctx, tenantID, salesOrderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.SalesReturn), args.Error(1)
}

func (m *MockSalesReturnRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status trade.ReturnStatus, filter shared.Filter) ([]trade.SalesReturn, error) {
	args := m.Called(ctx, tenantID, status, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.SalesReturn), args.Error(1)
}

func (m *MockSalesReturnRepository) FindPendingApproval(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]trade.SalesReturn, error) {
	args := m.Called(ctx, tenantID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.SalesReturn), args.Error(1)
}

func (m *MockSalesReturnRepository) Save(ctx context.Context, sr *trade.SalesReturn) error {
	args := m.Called(ctx, sr)
	return args.Error(0)
}

func (m *MockSalesReturnRepository) SaveWithLock(ctx context.Context, sr *trade.SalesReturn) error {
	args := m.Called(ctx, sr)
	return args.Error(0)
}

func (m *MockSalesReturnRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSalesReturnRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *MockSalesReturnRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSalesReturnRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status trade.ReturnStatus) (int64, error) {
	args := m.Called(ctx, tenantID, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSalesReturnRepository) CountByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID, customerID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSalesReturnRepository) CountBySalesOrder(ctx context.Context, tenantID, salesOrderID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID, salesOrderID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSalesReturnRepository) CountPendingApproval(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSalesReturnRepository) ExistsByReturnNumber(ctx context.Context, tenantID uuid.UUID, returnNumber string) (bool, error) {
	args := m.Called(ctx, tenantID, returnNumber)
	return args.Bool(0), args.Error(1)
}

func (m *MockSalesReturnRepository) GenerateReturnNumber(ctx context.Context, tenantID uuid.UUID) (string, error) {
	args := m.Called(ctx, tenantID)
	return args.String(0), args.Error(1)
}

func (m *MockSalesReturnRepository) GetReturnedQuantityByOrderItem(ctx context.Context, tenantID, salesOrderItemID uuid.UUID) (map[uuid.UUID]decimal.Decimal, error) {
	args := m.Called(ctx, tenantID, salesOrderItemID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[uuid.UUID]decimal.Decimal), args.Error(1)
}

func (m *MockSalesReturnRepository) GetReturnedQuantityByOrderItems(ctx context.Context, tenantID uuid.UUID, salesOrderItemIDs []uuid.UUID) (map[uuid.UUID]decimal.Decimal, error) {
	args := m.Called(ctx, tenantID, salesOrderItemIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[uuid.UUID]decimal.Decimal), args.Error(1)
}

// Ensure mock implements the interface
var _ trade.SalesReturnRepository = (*MockSalesReturnRepository)(nil)

// Helper to create a test sales order with items
func createTestSalesOrderForReturn(tenantID uuid.UUID, orderID uuid.UUID, itemID uuid.UUID, quantity decimal.Decimal) *trade.SalesOrder {
	warehouseID := uuid.New()
	order := &trade.SalesOrder{
		OrderNumber:  "SO-2026-00001",
		CustomerID:   uuid.New(),
		CustomerName: "Test Customer",
		WarehouseID:  &warehouseID,
		Items: []trade.SalesOrderItem{
			{
				ID:             itemID,
				OrderID:        orderID,
				ProductID:      uuid.New(),
				ProductName:    "Test Product",
				ProductCode:    "SKU-001",
				Quantity:       quantity,
				UnitPrice:      decimal.NewFromFloat(99.99),
				Amount:         quantity.Mul(decimal.NewFromFloat(99.99)),
				Unit:           "pcs",
				BaseUnit:       "pcs",
				ConversionRate: decimal.NewFromInt(1),
			},
		},
		Status: trade.OrderStatusShipped,
	}
	order.ID = orderID
	order.TenantID = tenantID
	order.Version = 1
	return order
}

func TestSalesReturnService_Create_ValidatesReturnQuantity(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	orderID := uuid.New()
	orderItemID := uuid.New()
	originalQuantity := decimal.NewFromInt(100)

	t.Run("should allow return when no previous returns exist", func(t *testing.T) {
		mockReturnRepo := new(MockSalesReturnRepository)
		mockOrderRepo := new(MockSalesOrderRepository)
		service := NewSalesReturnService(mockReturnRepo, mockOrderRepo)

		order := createTestSalesOrderForReturn(tenantID, orderID, orderItemID, originalQuantity)

		// No previous returns
		alreadyReturned := map[uuid.UUID]decimal.Decimal{
			orderItemID: decimal.Zero,
		}

		mockOrderRepo.On("FindByIDForTenant", ctx, tenantID, orderID).Return(order, nil)
		mockReturnRepo.On("GetReturnedQuantityByOrderItems", ctx, tenantID, []uuid.UUID{orderItemID}).Return(alreadyReturned, nil)
		mockReturnRepo.On("GenerateReturnNumber", ctx, tenantID).Return("SR-2026-00001", nil)
		mockReturnRepo.On("Save", ctx, mock.AnythingOfType("*trade.SalesReturn")).Return(nil)

		req := CreateSalesReturnRequest{
			SalesOrderID: orderID,
			Items: []CreateSalesReturnItemInput{
				{
					SalesOrderItemID: orderItemID,
					ReturnQuantity:   decimal.NewFromInt(10),
					Reason:           "Damaged",
				},
			},
		}

		result, err := service.Create(ctx, tenantID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		mockReturnRepo.AssertExpectations(t)
		mockOrderRepo.AssertExpectations(t)
	})

	t.Run("should allow return when within remaining quantity", func(t *testing.T) {
		mockReturnRepo := new(MockSalesReturnRepository)
		mockOrderRepo := new(MockSalesOrderRepository)
		service := NewSalesReturnService(mockReturnRepo, mockOrderRepo)

		order := createTestSalesOrderForReturn(tenantID, orderID, orderItemID, originalQuantity)

		// Previous returns of 50 units
		alreadyReturned := map[uuid.UUID]decimal.Decimal{
			orderItemID: decimal.NewFromInt(50),
		}

		mockOrderRepo.On("FindByIDForTenant", ctx, tenantID, orderID).Return(order, nil)
		mockReturnRepo.On("GetReturnedQuantityByOrderItems", ctx, tenantID, []uuid.UUID{orderItemID}).Return(alreadyReturned, nil)
		mockReturnRepo.On("GenerateReturnNumber", ctx, tenantID).Return("SR-2026-00002", nil)
		mockReturnRepo.On("Save", ctx, mock.AnythingOfType("*trade.SalesReturn")).Return(nil)

		req := CreateSalesReturnRequest{
			SalesOrderID: orderID,
			Items: []CreateSalesReturnItemInput{
				{
					SalesOrderItemID: orderItemID,
					ReturnQuantity:   decimal.NewFromInt(50), // Exactly remaining quantity
					Reason:           "Damaged",
				},
			},
		}

		result, err := service.Create(ctx, tenantID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		mockReturnRepo.AssertExpectations(t)
		mockOrderRepo.AssertExpectations(t)
	})

	t.Run("should reject return when exceeds remaining quantity", func(t *testing.T) {
		mockReturnRepo := new(MockSalesReturnRepository)
		mockOrderRepo := new(MockSalesOrderRepository)
		service := NewSalesReturnService(mockReturnRepo, mockOrderRepo)

		order := createTestSalesOrderForReturn(tenantID, orderID, orderItemID, originalQuantity)

		// Previous returns of 50 units, only 50 remaining
		alreadyReturned := map[uuid.UUID]decimal.Decimal{
			orderItemID: decimal.NewFromInt(50),
		}

		mockOrderRepo.On("FindByIDForTenant", ctx, tenantID, orderID).Return(order, nil)
		mockReturnRepo.On("GetReturnedQuantityByOrderItems", ctx, tenantID, []uuid.UUID{orderItemID}).Return(alreadyReturned, nil)

		req := CreateSalesReturnRequest{
			SalesOrderID: orderID,
			Items: []CreateSalesReturnItemInput{
				{
					SalesOrderItemID: orderItemID,
					ReturnQuantity:   decimal.NewFromInt(51), // Exceeds remaining (50)
					Reason:           "Damaged",
				},
			},
		}

		result, err := service.Create(ctx, tenantID, req)

		assert.Error(t, err)
		assert.Nil(t, result)

		// Check the error type
		var domainErr *shared.DomainError
		assert.ErrorAs(t, err, &domainErr)
		assert.Equal(t, "EXCESSIVE_RETURN_QUANTITY", domainErr.Code)

		mockOrderRepo.AssertExpectations(t)
		mockReturnRepo.AssertExpectations(t)
	})

	t.Run("should reject return when all quantity already returned", func(t *testing.T) {
		mockReturnRepo := new(MockSalesReturnRepository)
		mockOrderRepo := new(MockSalesOrderRepository)
		service := NewSalesReturnService(mockReturnRepo, mockOrderRepo)

		order := createTestSalesOrderForReturn(tenantID, orderID, orderItemID, originalQuantity)

		// All 100 units already returned
		alreadyReturned := map[uuid.UUID]decimal.Decimal{
			orderItemID: decimal.NewFromInt(100),
		}

		mockOrderRepo.On("FindByIDForTenant", ctx, tenantID, orderID).Return(order, nil)
		mockReturnRepo.On("GetReturnedQuantityByOrderItems", ctx, tenantID, []uuid.UUID{orderItemID}).Return(alreadyReturned, nil)

		req := CreateSalesReturnRequest{
			SalesOrderID: orderID,
			Items: []CreateSalesReturnItemInput{
				{
					SalesOrderItemID: orderItemID,
					ReturnQuantity:   decimal.NewFromInt(1), // Even 1 unit is too many
					Reason:           "Damaged",
				},
			},
		}

		result, err := service.Create(ctx, tenantID, req)

		assert.Error(t, err)
		assert.Nil(t, result)

		var domainErr *shared.DomainError
		assert.ErrorAs(t, err, &domainErr)
		assert.Equal(t, "EXCESSIVE_RETURN_QUANTITY", domainErr.Code)

		mockOrderRepo.AssertExpectations(t)
		mockReturnRepo.AssertExpectations(t)
	})
}

func TestSalesReturnService_AddItem_ValidatesReturnQuantity(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	orderID := uuid.New()
	returnID := uuid.New()
	orderItemID := uuid.New()
	originalQuantity := decimal.NewFromInt(100)

	t.Run("should allow adding item when within remaining quantity", func(t *testing.T) {
		mockReturnRepo := new(MockSalesReturnRepository)
		mockOrderRepo := new(MockSalesOrderRepository)
		service := NewSalesReturnService(mockReturnRepo, mockOrderRepo)

		order := createTestSalesOrderForReturn(tenantID, orderID, orderItemID, originalQuantity)

		// Create a draft return without items
		warehouseID := uuid.New()
		sr := &trade.SalesReturn{
			ReturnNumber:     "SR-2026-00001",
			SalesOrderID:     orderID,
			SalesOrderNumber: "SO-2026-00001",
			CustomerID:       uuid.New(),
			CustomerName:     "Test Customer",
			WarehouseID:      &warehouseID,
			Items:            []trade.SalesReturnItem{},
			Status:           trade.ReturnStatusDraft,
		}
		sr.ID = returnID
		sr.TenantID = tenantID
		sr.Version = 1

		// 30 units already returned
		alreadyReturned := map[uuid.UUID]decimal.Decimal{
			orderItemID: decimal.NewFromInt(30),
		}

		mockReturnRepo.On("FindByIDForTenant", ctx, tenantID, returnID).Return(sr, nil)
		mockOrderRepo.On("FindByIDForTenant", ctx, tenantID, orderID).Return(order, nil)
		mockReturnRepo.On("GetReturnedQuantityByOrderItem", ctx, tenantID, orderItemID).Return(alreadyReturned, nil)
		mockReturnRepo.On("SaveWithLock", ctx, mock.AnythingOfType("*trade.SalesReturn")).Return(nil)

		req := AddReturnItemRequest{
			SalesOrderItemID:  orderItemID,
			ReturnQuantity:    decimal.NewFromInt(70), // Exactly remaining (100 - 30)
			Reason:            "Wrong item",
			ConditionOnReturn: "defective",
		}

		result, err := service.AddItem(ctx, tenantID, returnID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		mockReturnRepo.AssertExpectations(t)
		mockOrderRepo.AssertExpectations(t)
	})

	t.Run("should reject adding item when exceeds remaining quantity", func(t *testing.T) {
		mockReturnRepo := new(MockSalesReturnRepository)
		mockOrderRepo := new(MockSalesOrderRepository)
		service := NewSalesReturnService(mockReturnRepo, mockOrderRepo)

		order := createTestSalesOrderForReturn(tenantID, orderID, orderItemID, originalQuantity)

		// Create a draft return without items
		warehouseID := uuid.New()
		sr := &trade.SalesReturn{
			ReturnNumber:     "SR-2026-00001",
			SalesOrderID:     orderID,
			SalesOrderNumber: "SO-2026-00001",
			CustomerID:       uuid.New(),
			CustomerName:     "Test Customer",
			WarehouseID:      &warehouseID,
			Items:            []trade.SalesReturnItem{},
			Status:           trade.ReturnStatusDraft,
		}
		sr.ID = returnID
		sr.TenantID = tenantID
		sr.Version = 1

		// 90 units already returned, only 10 remaining
		alreadyReturned := map[uuid.UUID]decimal.Decimal{
			orderItemID: decimal.NewFromInt(90),
		}

		mockReturnRepo.On("FindByIDForTenant", ctx, tenantID, returnID).Return(sr, nil)
		mockOrderRepo.On("FindByIDForTenant", ctx, tenantID, orderID).Return(order, nil)
		mockReturnRepo.On("GetReturnedQuantityByOrderItem", ctx, tenantID, orderItemID).Return(alreadyReturned, nil)

		req := AddReturnItemRequest{
			SalesOrderItemID:  orderItemID,
			ReturnQuantity:    decimal.NewFromInt(20), // Exceeds remaining (10)
			Reason:            "Wrong item",
			ConditionOnReturn: "defective",
		}

		result, err := service.AddItem(ctx, tenantID, returnID, req)

		assert.Error(t, err)
		assert.Nil(t, result)

		var domainErr *shared.DomainError
		assert.ErrorAs(t, err, &domainErr)
		assert.Equal(t, "EXCESSIVE_RETURN_QUANTITY", domainErr.Code)

		mockReturnRepo.AssertExpectations(t)
		mockOrderRepo.AssertExpectations(t)
	})
}

func TestSalesReturnService_UpdateItem_ValidatesReturnQuantity(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	orderID := uuid.New()
	returnID := uuid.New()
	orderItemID := uuid.New()
	returnItemID := uuid.New()
	originalQuantity := decimal.NewFromInt(100)

	t.Run("should allow updating item quantity when within remaining", func(t *testing.T) {
		mockReturnRepo := new(MockSalesReturnRepository)
		mockOrderRepo := new(MockSalesOrderRepository)
		service := NewSalesReturnService(mockReturnRepo, mockOrderRepo)

		order := createTestSalesOrderForReturn(tenantID, orderID, orderItemID, originalQuantity)

		// Create a draft return with an existing item of quantity 20
		warehouseID := uuid.New()
		sr := &trade.SalesReturn{
			ReturnNumber:     "SR-2026-00001",
			SalesOrderID:     orderID,
			SalesOrderNumber: "SO-2026-00001",
			CustomerID:       uuid.New(),
			CustomerName:     "Test Customer",
			WarehouseID:      &warehouseID,
			Items: []trade.SalesReturnItem{
				{
					ID:               returnItemID,
					ReturnID:         returnID,
					SalesOrderItemID: orderItemID,
					ProductID:        uuid.New(),
					ProductName:      "Test Product",
					ProductCode:      "SKU-001",
					OriginalQuantity: originalQuantity,
					ReturnQuantity:   decimal.NewFromInt(20),
					UnitPrice:        decimal.NewFromFloat(99.99),
					RefundAmount:     decimal.NewFromFloat(1999.80),
					Unit:             "pcs",
					BaseUnit:         "pcs",
					ConversionRate:   decimal.NewFromInt(1),
					BaseQuantity:     decimal.NewFromInt(20),
				},
			},
			Status: trade.ReturnStatusDraft,
		}
		sr.ID = returnID
		sr.TenantID = tenantID
		sr.Version = 1

		// Current return has 20 (this item), no other returns exist
		// Total already returned = 20 (from this item only)
		alreadyReturned := map[uuid.UUID]decimal.Decimal{
			orderItemID: decimal.NewFromInt(20),
		}

		mockReturnRepo.On("FindByIDForTenant", ctx, tenantID, returnID).Return(sr, nil)
		mockOrderRepo.On("FindByIDForTenant", ctx, tenantID, orderID).Return(order, nil)
		mockReturnRepo.On("GetReturnedQuantityByOrderItem", ctx, tenantID, orderItemID).Return(alreadyReturned, nil)
		mockReturnRepo.On("SaveWithLock", ctx, mock.AnythingOfType("*trade.SalesReturn")).Return(nil)

		// Update to 80 (remaining was 80 + 20 current = 100)
		newQty := decimal.NewFromInt(80)
		req := UpdateReturnItemRequest{
			ReturnQuantity: &newQty,
		}

		result, err := service.UpdateItem(ctx, tenantID, returnID, returnItemID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		mockReturnRepo.AssertExpectations(t)
		mockOrderRepo.AssertExpectations(t)
	})

	t.Run("should reject updating item quantity when exceeds remaining", func(t *testing.T) {
		mockReturnRepo := new(MockSalesReturnRepository)
		mockOrderRepo := new(MockSalesOrderRepository)
		service := NewSalesReturnService(mockReturnRepo, mockOrderRepo)

		order := createTestSalesOrderForReturn(tenantID, orderID, orderItemID, originalQuantity)

		// Create a draft return with an existing item of quantity 20
		warehouseID := uuid.New()
		sr := &trade.SalesReturn{
			ReturnNumber:     "SR-2026-00001",
			SalesOrderID:     orderID,
			SalesOrderNumber: "SO-2026-00001",
			CustomerID:       uuid.New(),
			CustomerName:     "Test Customer",
			WarehouseID:      &warehouseID,
			Items: []trade.SalesReturnItem{
				{
					ID:               returnItemID,
					ReturnID:         returnID,
					SalesOrderItemID: orderItemID,
					ProductID:        uuid.New(),
					ProductName:      "Test Product",
					ProductCode:      "SKU-001",
					OriginalQuantity: originalQuantity,
					ReturnQuantity:   decimal.NewFromInt(20),
					UnitPrice:        decimal.NewFromFloat(99.99),
					RefundAmount:     decimal.NewFromFloat(1999.80),
					Unit:             "pcs",
					BaseUnit:         "pcs",
					ConversionRate:   decimal.NewFromInt(1),
					BaseQuantity:     decimal.NewFromInt(20),
				},
			},
			Status: trade.ReturnStatusDraft,
		}
		sr.ID = returnID
		sr.TenantID = tenantID
		sr.Version = 1

		// Another return already has 50 units, plus this item has 20
		// Total already returned = 70 (50 other + 20 this item)
		alreadyReturned := map[uuid.UUID]decimal.Decimal{
			orderItemID: decimal.NewFromInt(70),
		}

		mockReturnRepo.On("FindByIDForTenant", ctx, tenantID, returnID).Return(sr, nil)
		mockOrderRepo.On("FindByIDForTenant", ctx, tenantID, orderID).Return(order, nil)
		mockReturnRepo.On("GetReturnedQuantityByOrderItem", ctx, tenantID, orderItemID).Return(alreadyReturned, nil)

		// Try to update to 60 (remaining without this item is 50, so max is 50)
		// Other returns: 50, current item: 20 -> total 70
		// Remaining = 100 - 50 = 50 (excluding current item)
		// Requested 60 > 50, so should fail
		newQty := decimal.NewFromInt(60)
		req := UpdateReturnItemRequest{
			ReturnQuantity: &newQty,
		}

		result, err := service.UpdateItem(ctx, tenantID, returnID, returnItemID, req)

		assert.Error(t, err)
		assert.Nil(t, result)

		var domainErr *shared.DomainError
		assert.ErrorAs(t, err, &domainErr)
		assert.Equal(t, "EXCESSIVE_RETURN_QUANTITY", domainErr.Code)

		mockReturnRepo.AssertExpectations(t)
		mockOrderRepo.AssertExpectations(t)
	})
}
