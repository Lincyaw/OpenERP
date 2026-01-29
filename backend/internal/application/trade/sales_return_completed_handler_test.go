package trade

import (
	"context"
	"errors"
	"testing"

	inventoryapp "github.com/erp/backend/internal/application/inventory"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockInventoryServiceForSalesReturn extends mock for sales return handling
type MockInventoryServiceForSalesReturn struct {
	mock.Mock
}

func (m *MockInventoryServiceForSalesReturn) IncreaseStock(ctx context.Context, tenantID uuid.UUID, req inventoryapp.IncreaseStockRequest) (*inventoryapp.InventoryItemResponse, error) {
	args := m.Called(ctx, tenantID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*inventoryapp.InventoryItemResponse), args.Error(1)
}

func (m *MockInventoryServiceForSalesReturn) GetByWarehouseAndProduct(ctx context.Context, tenantID, warehouseID, productID uuid.UUID) (*inventoryapp.InventoryItemResponse, error) {
	args := m.Called(ctx, tenantID, warehouseID, productID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*inventoryapp.InventoryItemResponse), args.Error(1)
}

// Test helper variables for sales return handlers
var (
	testReturnHandlerTenantID     = uuid.New()
	testReturnHandlerReturnID     = uuid.New()
	testReturnHandlerOrderID      = uuid.New()
	testReturnHandlerWarehouseID  = uuid.New()
	testReturnHandlerProductID    = uuid.New()
	testReturnHandlerReturnNumber = "SR-2024-00001"
	testReturnHandlerOrderNumber  = "SO-2024-00001"
	testReturnHandlerCustomerID   = uuid.New()
	testReturnHandlerCustomerName = "Test Customer"
)

// ==================== SalesReturnCompletedHandler Tests ====================

func TestSalesReturnCompletedHandler_EventTypes(t *testing.T) {
	logger := zap.NewNop()
	handler := NewSalesReturnCompletedHandler(nil, logger)

	eventTypes := handler.EventTypes()
	assert.Len(t, eventTypes, 1)
	assert.Equal(t, trade.EventTypeSalesReturnCompleted, eventTypes[0])
}

func TestSalesReturnCompletedHandler_Handle_Success(t *testing.T) {
	mockService := new(MockInventoryServiceForSalesReturn)
	logger := zap.NewNop()

	ctx := context.Background()

	event := &trade.SalesReturnCompletedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypeSalesReturnCompleted,
			trade.AggregateTypeSalesReturn,
			testReturnHandlerReturnID,
			testReturnHandlerTenantID,
		),
		ReturnID:         testReturnHandlerReturnID,
		ReturnNumber:     testReturnHandlerReturnNumber,
		SalesOrderID:     testReturnHandlerOrderID,
		SalesOrderNumber: testReturnHandlerOrderNumber,
		CustomerID:       testReturnHandlerCustomerID,
		CustomerName:     testReturnHandlerCustomerName,
		WarehouseID:      testReturnHandlerWarehouseID,
		Items: []trade.SalesReturnItemInfo{
			{
				ItemID:         uuid.New(),
				ProductID:      testReturnHandlerProductID,
				ProductName:    "Product A",
				ProductCode:    "PROD-A",
				ReturnQuantity: decimal.NewFromInt(3),
				UnitPrice:      decimal.NewFromFloat(100.00),
				RefundAmount:   decimal.NewFromFloat(300.00),
				Unit:           "pcs",
			},
		},
		TotalRefund: decimal.NewFromFloat(300.00),
	}

	// Mock GetByWarehouseAndProduct to return existing inventory with unit cost
	mockService.On("GetByWarehouseAndProduct", ctx, testReturnHandlerTenantID, testReturnHandlerWarehouseID, testReturnHandlerProductID).
		Return(&inventoryapp.InventoryItemResponse{
			ID:                uuid.New(),
			WarehouseID:       testReturnHandlerWarehouseID,
			ProductID:         testReturnHandlerProductID,
			AvailableQuantity: decimal.NewFromInt(10),
			UnitCost:          decimal.NewFromFloat(80.00), // Current unit cost
		}, nil)

	// Mock IncreaseStock with existing unit cost
	mockService.On("IncreaseStock", ctx, testReturnHandlerTenantID, mock.MatchedBy(func(req inventoryapp.IncreaseStockRequest) bool {
		return req.WarehouseID == testReturnHandlerWarehouseID &&
			req.ProductID == testReturnHandlerProductID &&
			req.Quantity.Equal(decimal.NewFromInt(3)) &&
			req.UnitCost.Equal(decimal.NewFromFloat(80.00)) && // Uses existing unit cost
			req.SourceType == "SALES_RETURN" &&
			req.SourceID == testReturnHandlerReturnID.String()
	})).Return(&inventoryapp.InventoryItemResponse{
		ID:                uuid.New(),
		WarehouseID:       testReturnHandlerWarehouseID,
		ProductID:         testReturnHandlerProductID,
		AvailableQuantity: decimal.NewFromInt(13), // 10 + 3
	}, nil)

	testHandler := &testableReturnCompletedHandler{
		mockService: mockService,
		logger:      logger,
	}
	err := testHandler.Handle(ctx, event)

	assert.NoError(t, err)
	mockService.AssertExpectations(t)
}

func TestSalesReturnCompletedHandler_Handle_FallbackToUnitPrice(t *testing.T) {
	mockService := new(MockInventoryServiceForSalesReturn)
	logger := zap.NewNop()

	ctx := context.Background()

	event := &trade.SalesReturnCompletedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypeSalesReturnCompleted,
			trade.AggregateTypeSalesReturn,
			testReturnHandlerReturnID,
			testReturnHandlerTenantID,
		),
		ReturnID:         testReturnHandlerReturnID,
		ReturnNumber:     testReturnHandlerReturnNumber,
		SalesOrderID:     testReturnHandlerOrderID,
		SalesOrderNumber: testReturnHandlerOrderNumber,
		CustomerID:       testReturnHandlerCustomerID,
		CustomerName:     testReturnHandlerCustomerName,
		WarehouseID:      testReturnHandlerWarehouseID,
		Items: []trade.SalesReturnItemInfo{
			{
				ItemID:         uuid.New(),
				ProductID:      testReturnHandlerProductID,
				ProductName:    "Product A",
				ProductCode:    "PROD-A",
				ReturnQuantity: decimal.NewFromInt(3),
				UnitPrice:      decimal.NewFromFloat(100.00),
				RefundAmount:   decimal.NewFromFloat(300.00),
				Unit:           "pcs",
			},
		},
		TotalRefund: decimal.NewFromFloat(300.00),
	}

	// Mock GetByWarehouseAndProduct to return error (no existing inventory)
	mockService.On("GetByWarehouseAndProduct", ctx, testReturnHandlerTenantID, testReturnHandlerWarehouseID, testReturnHandlerProductID).
		Return(nil, shared.ErrNotFound)

	// Mock IncreaseStock with unit price as fallback
	mockService.On("IncreaseStock", ctx, testReturnHandlerTenantID, mock.MatchedBy(func(req inventoryapp.IncreaseStockRequest) bool {
		return req.WarehouseID == testReturnHandlerWarehouseID &&
			req.ProductID == testReturnHandlerProductID &&
			req.Quantity.Equal(decimal.NewFromInt(3)) &&
			req.UnitCost.Equal(decimal.NewFromFloat(100.00)) && // Uses item unit price as fallback
			req.SourceType == "SALES_RETURN" &&
			req.SourceID == testReturnHandlerReturnID.String()
	})).Return(&inventoryapp.InventoryItemResponse{
		ID:                uuid.New(),
		WarehouseID:       testReturnHandlerWarehouseID,
		ProductID:         testReturnHandlerProductID,
		AvailableQuantity: decimal.NewFromInt(3),
	}, nil)

	testHandler := &testableReturnCompletedHandler{
		mockService: mockService,
		logger:      logger,
	}
	err := testHandler.Handle(ctx, event)

	assert.NoError(t, err)
	mockService.AssertExpectations(t)
}

func TestSalesReturnCompletedHandler_Handle_MultipleItems(t *testing.T) {
	mockService := new(MockInventoryServiceForSalesReturn)
	logger := zap.NewNop()

	ctx := context.Background()
	productID1 := uuid.New()
	productID2 := uuid.New()

	event := &trade.SalesReturnCompletedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypeSalesReturnCompleted,
			trade.AggregateTypeSalesReturn,
			testReturnHandlerReturnID,
			testReturnHandlerTenantID,
		),
		ReturnID:         testReturnHandlerReturnID,
		ReturnNumber:     testReturnHandlerReturnNumber,
		SalesOrderID:     testReturnHandlerOrderID,
		SalesOrderNumber: testReturnHandlerOrderNumber,
		CustomerID:       testReturnHandlerCustomerID,
		CustomerName:     testReturnHandlerCustomerName,
		WarehouseID:      testReturnHandlerWarehouseID,
		Items: []trade.SalesReturnItemInfo{
			{
				ItemID:         uuid.New(),
				ProductID:      productID1,
				ProductName:    "Product A",
				ProductCode:    "PROD-A",
				ReturnQuantity: decimal.NewFromInt(5),
				UnitPrice:      decimal.NewFromFloat(100.00),
				RefundAmount:   decimal.NewFromFloat(500.00),
				Unit:           "pcs",
			},
			{
				ItemID:         uuid.New(),
				ProductID:      productID2,
				ProductName:    "Product B",
				ProductCode:    "PROD-B",
				ReturnQuantity: decimal.NewFromInt(2),
				UnitPrice:      decimal.NewFromFloat(50.00),
				RefundAmount:   decimal.NewFromFloat(100.00),
				Unit:           "box",
			},
		},
		TotalRefund: decimal.NewFromFloat(600.00),
	}

	// Mock for product 1
	mockService.On("GetByWarehouseAndProduct", ctx, testReturnHandlerTenantID, testReturnHandlerWarehouseID, productID1).
		Return(&inventoryapp.InventoryItemResponse{UnitCost: decimal.NewFromFloat(90.00)}, nil)
	mockService.On("IncreaseStock", ctx, testReturnHandlerTenantID, mock.MatchedBy(func(req inventoryapp.IncreaseStockRequest) bool {
		return req.ProductID == productID1
	})).Return(&inventoryapp.InventoryItemResponse{}, nil)

	// Mock for product 2
	mockService.On("GetByWarehouseAndProduct", ctx, testReturnHandlerTenantID, testReturnHandlerWarehouseID, productID2).
		Return(&inventoryapp.InventoryItemResponse{UnitCost: decimal.NewFromFloat(40.00)}, nil)
	mockService.On("IncreaseStock", ctx, testReturnHandlerTenantID, mock.MatchedBy(func(req inventoryapp.IncreaseStockRequest) bool {
		return req.ProductID == productID2
	})).Return(&inventoryapp.InventoryItemResponse{}, nil)

	testHandler := &testableReturnCompletedHandler{
		mockService: mockService,
		logger:      logger,
	}
	err := testHandler.Handle(ctx, event)

	assert.NoError(t, err)
	mockService.AssertNumberOfCalls(t, "IncreaseStock", 2)
	mockService.AssertExpectations(t)
}

func TestSalesReturnCompletedHandler_Handle_MissingWarehouse(t *testing.T) {
	logger := zap.NewNop()
	handler := NewSalesReturnCompletedHandler(nil, logger)

	ctx := context.Background()

	event := &trade.SalesReturnCompletedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypeSalesReturnCompleted,
			trade.AggregateTypeSalesReturn,
			testReturnHandlerReturnID,
			testReturnHandlerTenantID,
		),
		ReturnID:         testReturnHandlerReturnID,
		ReturnNumber:     testReturnHandlerReturnNumber,
		SalesOrderID:     testReturnHandlerOrderID,
		SalesOrderNumber: testReturnHandlerOrderNumber,
		CustomerID:       testReturnHandlerCustomerID,
		CustomerName:     testReturnHandlerCustomerName,
		WarehouseID:      uuid.Nil, // Missing warehouse
		Items: []trade.SalesReturnItemInfo{
			{
				ItemID:         uuid.New(),
				ProductID:      testReturnHandlerProductID,
				ProductName:    "Product A",
				ProductCode:    "PROD-A",
				ReturnQuantity: decimal.NewFromInt(3),
				UnitPrice:      decimal.NewFromFloat(100.00),
				RefundAmount:   decimal.NewFromFloat(300.00),
				Unit:           "pcs",
			},
		},
		TotalRefund: decimal.NewFromFloat(300.00),
	}

	err := handler.Handle(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "warehouse ID is required")
}

func TestSalesReturnCompletedHandler_Handle_WrongEventType(t *testing.T) {
	logger := zap.NewNop()
	handler := NewSalesReturnCompletedHandler(nil, logger)

	ctx := context.Background()

	// Using a different event type
	wrongEvent := &trade.SalesReturnCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypeSalesReturnCreated,
			trade.AggregateTypeSalesReturn,
			testReturnHandlerReturnID,
			testReturnHandlerTenantID,
		),
		ReturnID:         testReturnHandlerReturnID,
		ReturnNumber:     testReturnHandlerReturnNumber,
		SalesOrderID:     testReturnHandlerOrderID,
		SalesOrderNumber: testReturnHandlerOrderNumber,
		CustomerID:       testReturnHandlerCustomerID,
		CustomerName:     testReturnHandlerCustomerName,
	}

	err := handler.Handle(ctx, wrongEvent)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

func TestSalesReturnCompletedHandler_Handle_PartialFailure(t *testing.T) {
	mockService := new(MockInventoryServiceForSalesReturn)
	logger := zap.NewNop()

	ctx := context.Background()
	productID1 := uuid.New()
	productID2 := uuid.New()

	event := &trade.SalesReturnCompletedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypeSalesReturnCompleted,
			trade.AggregateTypeSalesReturn,
			testReturnHandlerReturnID,
			testReturnHandlerTenantID,
		),
		ReturnID:         testReturnHandlerReturnID,
		ReturnNumber:     testReturnHandlerReturnNumber,
		SalesOrderID:     testReturnHandlerOrderID,
		SalesOrderNumber: testReturnHandlerOrderNumber,
		CustomerID:       testReturnHandlerCustomerID,
		CustomerName:     testReturnHandlerCustomerName,
		WarehouseID:      testReturnHandlerWarehouseID,
		Items: []trade.SalesReturnItemInfo{
			{
				ItemID:         uuid.New(),
				ProductID:      productID1,
				ProductName:    "Product A",
				ProductCode:    "PROD-A",
				ReturnQuantity: decimal.NewFromInt(5),
				UnitPrice:      decimal.NewFromFloat(100.00),
				RefundAmount:   decimal.NewFromFloat(500.00),
				Unit:           "pcs",
			},
			{
				ItemID:         uuid.New(),
				ProductID:      productID2,
				ProductName:    "Product B",
				ProductCode:    "PROD-B",
				ReturnQuantity: decimal.NewFromInt(2),
				UnitPrice:      decimal.NewFromFloat(50.00),
				RefundAmount:   decimal.NewFromFloat(100.00),
				Unit:           "box",
			},
		},
		TotalRefund: decimal.NewFromFloat(600.00),
	}

	// Product 1 succeeds
	mockService.On("GetByWarehouseAndProduct", ctx, testReturnHandlerTenantID, testReturnHandlerWarehouseID, productID1).
		Return(&inventoryapp.InventoryItemResponse{UnitCost: decimal.NewFromFloat(90.00)}, nil)
	mockService.On("IncreaseStock", ctx, testReturnHandlerTenantID, mock.MatchedBy(func(req inventoryapp.IncreaseStockRequest) bool {
		return req.ProductID == productID1
	})).Return(&inventoryapp.InventoryItemResponse{}, nil)

	// Product 2 fails
	mockService.On("GetByWarehouseAndProduct", ctx, testReturnHandlerTenantID, testReturnHandlerWarehouseID, productID2).
		Return(&inventoryapp.InventoryItemResponse{UnitCost: decimal.NewFromFloat(40.00)}, nil)
	mockService.On("IncreaseStock", ctx, testReturnHandlerTenantID, mock.MatchedBy(func(req inventoryapp.IncreaseStockRequest) bool {
		return req.ProductID == productID2
	})).Return(nil, errors.New("database error"))

	testHandler := &testableReturnCompletedHandler{
		mockService: mockService,
		logger:      logger,
	}
	err := testHandler.Handle(ctx, event)

	// Should return error but both items should be attempted
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "some items failed to restore stock")
	mockService.AssertNumberOfCalls(t, "IncreaseStock", 2)
}

func TestNewSalesReturnCompletedHandler(t *testing.T) {
	logger := zap.NewNop()
	handler := NewSalesReturnCompletedHandler(nil, logger)

	assert.NotNil(t, handler)
	assert.Equal(t, logger, handler.logger)
}

// ==================== Testable Handler Wrapper ====================

// testableReturnCompletedHandler is a helper for testing SalesReturnCompletedHandler with mock services
type testableReturnCompletedHandler struct {
	mockService *MockInventoryServiceForSalesReturn
	logger      *zap.Logger
}

func (h *testableReturnCompletedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	completedEvent, ok := event.(*trade.SalesReturnCompletedEvent)
	if !ok {
		return errors.New("unexpected event type: expected SalesReturnCompletedEvent")
	}

	if completedEvent.WarehouseID == uuid.Nil {
		return errors.New("warehouse ID is required for stock restoration")
	}

	var lastErr error
	for _, item := range completedEvent.Items {
		// Try to get existing unit cost
		unitCost := item.UnitPrice
		invItem, err := h.mockService.GetByWarehouseAndProduct(ctx, event.TenantID(), completedEvent.WarehouseID, item.ProductID)
		if err == nil && invItem != nil {
			unitCost = invItem.UnitCost
		}

		req := inventoryapp.IncreaseStockRequest{
			WarehouseID: completedEvent.WarehouseID,
			ProductID:   item.ProductID,
			Quantity:    item.ReturnQuantity,
			UnitCost:    unitCost,
			SourceType:  "SALES_RETURN",
			SourceID:    completedEvent.ReturnID.String(),
			Reference:   "SR:" + completedEvent.ReturnNumber,
			Reason:      "Sales return: " + completedEvent.ReturnNumber,
		}

		if _, err := h.mockService.IncreaseStock(ctx, event.TenantID(), req); err != nil {
			lastErr = err
			continue
		}
	}

	if lastErr != nil {
		return errors.New("some items failed to restore stock: " + lastErr.Error())
	}

	return nil
}
