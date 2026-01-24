package trade

import (
	"context"
	"errors"
	"testing"
	"time"

	inventoryapp "github.com/erp/backend/internal/application/inventory"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockInventoryServiceForSales extends the MockInventoryService with sales-specific methods
type MockInventoryServiceForSales struct {
	mock.Mock
}

func (m *MockInventoryServiceForSales) LockStock(ctx context.Context, tenantID uuid.UUID, req inventoryapp.LockStockRequest) (*inventoryapp.LockStockResponse, error) {
	args := m.Called(ctx, tenantID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*inventoryapp.LockStockResponse), args.Error(1)
}

func (m *MockInventoryServiceForSales) GetLocksBySource(ctx context.Context, sourceType, sourceID string) ([]inventoryapp.StockLockResponse, error) {
	args := m.Called(ctx, sourceType, sourceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]inventoryapp.StockLockResponse), args.Error(1)
}

func (m *MockInventoryServiceForSales) DeductStock(ctx context.Context, tenantID uuid.UUID, req inventoryapp.DeductStockRequest) error {
	args := m.Called(ctx, tenantID, req)
	return args.Error(0)
}

func (m *MockInventoryServiceForSales) UnlockBySource(ctx context.Context, tenantID uuid.UUID, sourceType, sourceID string) (int, error) {
	args := m.Called(ctx, tenantID, sourceType, sourceID)
	return args.Int(0), args.Error(1)
}

// Test helper variables for sales handlers
var (
	testSalesHandlerTenantID     = uuid.New()
	testSalesHandlerOrderID      = uuid.New()
	testSalesHandlerWarehouseID  = uuid.New()
	testSalesHandlerProductID    = uuid.New()
	testSalesHandlerOrderNumber  = "SO-2024-00001"
	testSalesHandlerCustomerID   = uuid.New()
	testSalesHandlerCustomerName = "Test Customer"
)

// ==================== SalesOrderConfirmedHandler Tests ====================

func TestSalesOrderConfirmedHandler_EventTypes(t *testing.T) {
	logger := zap.NewNop()
	handler := NewSalesOrderConfirmedHandler(nil, logger)

	eventTypes := handler.EventTypes()
	assert.Len(t, eventTypes, 1)
	assert.Equal(t, trade.EventTypeSalesOrderConfirmed, eventTypes[0])
}

func TestSalesOrderConfirmedHandler_Handle_Success(t *testing.T) {
	mockService := new(MockInventoryServiceForSales)
	logger := zap.NewNop()

	ctx := context.Background()
	warehouseID := testSalesHandlerWarehouseID

	event := &trade.SalesOrderConfirmedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypeSalesOrderConfirmed,
			trade.AggregateTypeSalesOrder,
			testSalesHandlerOrderID,
			testSalesHandlerTenantID,
		),
		OrderID:      testSalesHandlerOrderID,
		OrderNumber:  testSalesHandlerOrderNumber,
		CustomerID:   testSalesHandlerCustomerID,
		CustomerName: testSalesHandlerCustomerName,
		WarehouseID:  &warehouseID,
		Items: []trade.SalesOrderItemInfo{
			{
				ItemID:      uuid.New(),
				ProductID:   testSalesHandlerProductID,
				ProductName: "Product A",
				ProductCode: "PROD-A",
				Quantity:    decimal.NewFromInt(5),
				UnitPrice:   decimal.NewFromFloat(100.00),
				Amount:      decimal.NewFromFloat(500.00),
				Unit:        "pcs",
			},
		},
		TotalAmount:   decimal.NewFromFloat(500.00),
		PayableAmount: decimal.NewFromFloat(500.00),
	}

	// Setup mock expectation for LockStock
	mockService.On("LockStock", ctx, testSalesHandlerTenantID, mock.MatchedBy(func(req inventoryapp.LockStockRequest) bool {
		return req.WarehouseID == testSalesHandlerWarehouseID &&
			req.ProductID == testSalesHandlerProductID &&
			req.Quantity.Equal(decimal.NewFromInt(5)) &&
			req.SourceType == "SALES_ORDER" &&
			req.SourceID == testSalesHandlerOrderID.String()
	})).Return(&inventoryapp.LockStockResponse{
		LockID:    uuid.New(),
		ProductID: testSalesHandlerProductID,
		Quantity:  decimal.NewFromInt(5),
		ExpireAt:  time.Now().Add(30 * time.Minute),
	}, nil)

	// Execute handler with testable wrapper
	testHandler := &testableConfirmHandler{
		mockService: mockService,
		logger:      logger,
	}
	err := testHandler.Handle(ctx, event)

	assert.NoError(t, err)
	mockService.AssertExpectations(t)
}

func TestSalesOrderConfirmedHandler_Handle_MissingWarehouse(t *testing.T) {
	logger := zap.NewNop()
	handler := NewSalesOrderConfirmedHandler(nil, logger)

	ctx := context.Background()

	// Event with nil warehouse ID
	event := &trade.SalesOrderConfirmedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypeSalesOrderConfirmed,
			trade.AggregateTypeSalesOrder,
			testSalesHandlerOrderID,
			testSalesHandlerTenantID,
		),
		OrderID:      testSalesHandlerOrderID,
		OrderNumber:  testSalesHandlerOrderNumber,
		CustomerID:   testSalesHandlerCustomerID,
		CustomerName: testSalesHandlerCustomerName,
		WarehouseID:  nil, // Missing warehouse
		Items: []trade.SalesOrderItemInfo{
			{
				ItemID:      uuid.New(),
				ProductID:   testSalesHandlerProductID,
				ProductName: "Product A",
				ProductCode: "PROD-A",
				Quantity:    decimal.NewFromInt(5),
				UnitPrice:   decimal.NewFromFloat(100.00),
				Amount:      decimal.NewFromFloat(500.00),
				Unit:        "pcs",
			},
		},
		TotalAmount:   decimal.NewFromFloat(500.00),
		PayableAmount: decimal.NewFromFloat(500.00),
	}

	err := handler.Handle(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "warehouse ID is required")
}

func TestSalesOrderConfirmedHandler_Handle_WrongEventType(t *testing.T) {
	logger := zap.NewNop()
	handler := NewSalesOrderConfirmedHandler(nil, logger)

	ctx := context.Background()

	// Using a different event type
	wrongEvent := &trade.SalesOrderCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypeSalesOrderCreated,
			trade.AggregateTypeSalesOrder,
			testSalesHandlerOrderID,
			testSalesHandlerTenantID,
		),
		OrderID:      testSalesHandlerOrderID,
		OrderNumber:  testSalesHandlerOrderNumber,
		CustomerID:   testSalesHandlerCustomerID,
		CustomerName: testSalesHandlerCustomerName,
	}

	err := handler.Handle(ctx, wrongEvent)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

func TestSalesOrderConfirmedHandler_Handle_PartialFailure(t *testing.T) {
	mockService := new(MockInventoryServiceForSales)
	logger := zap.NewNop()

	ctx := context.Background()
	warehouseID := testSalesHandlerWarehouseID
	productID1 := uuid.New()
	productID2 := uuid.New()

	event := &trade.SalesOrderConfirmedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypeSalesOrderConfirmed,
			trade.AggregateTypeSalesOrder,
			testSalesHandlerOrderID,
			testSalesHandlerTenantID,
		),
		OrderID:      testSalesHandlerOrderID,
		OrderNumber:  testSalesHandlerOrderNumber,
		CustomerID:   testSalesHandlerCustomerID,
		CustomerName: testSalesHandlerCustomerName,
		WarehouseID:  &warehouseID,
		Items: []trade.SalesOrderItemInfo{
			{
				ItemID:      uuid.New(),
				ProductID:   productID1,
				ProductName: "Product A",
				ProductCode: "PROD-A",
				Quantity:    decimal.NewFromInt(5),
				UnitPrice:   decimal.NewFromFloat(100.00),
				Amount:      decimal.NewFromFloat(500.00),
				Unit:        "pcs",
			},
			{
				ItemID:      uuid.New(),
				ProductID:   productID2,
				ProductName: "Product B",
				ProductCode: "PROD-B",
				Quantity:    decimal.NewFromInt(3),
				UnitPrice:   decimal.NewFromFloat(50.00),
				Amount:      decimal.NewFromFloat(150.00),
				Unit:        "pcs",
			},
		},
		TotalAmount:   decimal.NewFromFloat(650.00),
		PayableAmount: decimal.NewFromFloat(650.00),
	}

	// First item succeeds, second fails (insufficient stock)
	mockService.On("LockStock", ctx, testSalesHandlerTenantID, mock.MatchedBy(func(req inventoryapp.LockStockRequest) bool {
		return req.ProductID == productID1
	})).Return(&inventoryapp.LockStockResponse{LockID: uuid.New()}, nil).Once()

	mockService.On("LockStock", ctx, testSalesHandlerTenantID, mock.MatchedBy(func(req inventoryapp.LockStockRequest) bool {
		return req.ProductID == productID2
	})).Return(nil, errors.New("insufficient stock")).Once()

	testHandler := &testableConfirmHandler{
		mockService: mockService,
		logger:      logger,
	}
	err := testHandler.Handle(ctx, event)

	// Should return error but both items should be attempted
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "some items failed to lock")
	mockService.AssertNumberOfCalls(t, "LockStock", 2)
}

// ==================== SalesOrderShippedHandler Tests ====================

func TestSalesOrderShippedHandler_EventTypes(t *testing.T) {
	logger := zap.NewNop()
	handler := NewSalesOrderShippedHandler(nil, logger)

	eventTypes := handler.EventTypes()
	assert.Len(t, eventTypes, 1)
	assert.Equal(t, trade.EventTypeSalesOrderShipped, eventTypes[0])
}

func TestSalesOrderShippedHandler_Handle_Success(t *testing.T) {
	mockService := new(MockInventoryServiceForSales)
	logger := zap.NewNop()

	ctx := context.Background()
	lockID := uuid.New()

	event := &trade.SalesOrderShippedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypeSalesOrderShipped,
			trade.AggregateTypeSalesOrder,
			testSalesHandlerOrderID,
			testSalesHandlerTenantID,
		),
		OrderID:      testSalesHandlerOrderID,
		OrderNumber:  testSalesHandlerOrderNumber,
		CustomerID:   testSalesHandlerCustomerID,
		CustomerName: testSalesHandlerCustomerName,
		WarehouseID:  testSalesHandlerWarehouseID,
		Items: []trade.SalesOrderItemInfo{
			{
				ItemID:      uuid.New(),
				ProductID:   testSalesHandlerProductID,
				ProductName: "Product A",
				ProductCode: "PROD-A",
				Quantity:    decimal.NewFromInt(5),
				UnitPrice:   decimal.NewFromFloat(100.00),
				Amount:      decimal.NewFromFloat(500.00),
				Unit:        "pcs",
			},
		},
		TotalAmount:   decimal.NewFromFloat(500.00),
		PayableAmount: decimal.NewFromFloat(500.00),
	}

	// Mock GetLocksBySource to return active locks
	mockService.On("GetLocksBySource", ctx, "SALES_ORDER", testSalesHandlerOrderID.String()).Return([]inventoryapp.StockLockResponse{
		{
			ID:        lockID,
			ProductID: testSalesHandlerProductID,
			Quantity:  decimal.NewFromInt(5),
			Released:  false,
			Consumed:  false,
		},
	}, nil)

	// Mock DeductStock
	mockService.On("DeductStock", ctx, testSalesHandlerTenantID, mock.MatchedBy(func(req inventoryapp.DeductStockRequest) bool {
		return req.LockID == lockID &&
			req.SourceType == "SALES_ORDER" &&
			req.SourceID == testSalesHandlerOrderID.String()
	})).Return(nil)

	testHandler := &testableShippedHandler{
		mockService: mockService,
		logger:      logger,
	}
	err := testHandler.Handle(ctx, event)

	assert.NoError(t, err)
	mockService.AssertExpectations(t)
}

func TestSalesOrderShippedHandler_Handle_MissingWarehouse(t *testing.T) {
	logger := zap.NewNop()
	handler := NewSalesOrderShippedHandler(nil, logger)

	ctx := context.Background()

	event := &trade.SalesOrderShippedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypeSalesOrderShipped,
			trade.AggregateTypeSalesOrder,
			testSalesHandlerOrderID,
			testSalesHandlerTenantID,
		),
		OrderID:      testSalesHandlerOrderID,
		OrderNumber:  testSalesHandlerOrderNumber,
		CustomerID:   testSalesHandlerCustomerID,
		CustomerName: testSalesHandlerCustomerName,
		WarehouseID:  uuid.Nil, // Missing warehouse
		Items: []trade.SalesOrderItemInfo{
			{
				ItemID:      uuid.New(),
				ProductID:   testSalesHandlerProductID,
				ProductName: "Product A",
				ProductCode: "PROD-A",
				Quantity:    decimal.NewFromInt(5),
				UnitPrice:   decimal.NewFromFloat(100.00),
				Amount:      decimal.NewFromFloat(500.00),
				Unit:        "pcs",
			},
		},
		TotalAmount:   decimal.NewFromFloat(500.00),
		PayableAmount: decimal.NewFromFloat(500.00),
	}

	err := handler.Handle(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "warehouse ID is required")
}

func TestSalesOrderShippedHandler_Handle_WrongEventType(t *testing.T) {
	logger := zap.NewNop()
	handler := NewSalesOrderShippedHandler(nil, logger)

	ctx := context.Background()

	wrongEvent := &trade.SalesOrderCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypeSalesOrderCreated,
			trade.AggregateTypeSalesOrder,
			testSalesHandlerOrderID,
			testSalesHandlerTenantID,
		),
		OrderID:      testSalesHandlerOrderID,
		OrderNumber:  testSalesHandlerOrderNumber,
		CustomerID:   testSalesHandlerCustomerID,
		CustomerName: testSalesHandlerCustomerName,
	}

	err := handler.Handle(ctx, wrongEvent)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

// ==================== SalesOrderCancelledHandler Tests ====================

func TestSalesOrderCancelledHandler_EventTypes(t *testing.T) {
	logger := zap.NewNop()
	handler := NewSalesOrderCancelledHandler(nil, logger)

	eventTypes := handler.EventTypes()
	assert.Len(t, eventTypes, 1)
	assert.Equal(t, trade.EventTypeSalesOrderCancelled, eventTypes[0])
}

func TestSalesOrderCancelledHandler_Handle_WasConfirmed(t *testing.T) {
	mockService := new(MockInventoryServiceForSales)
	logger := zap.NewNop()

	ctx := context.Background()
	warehouseID := testSalesHandlerWarehouseID

	event := &trade.SalesOrderCancelledEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypeSalesOrderCancelled,
			trade.AggregateTypeSalesOrder,
			testSalesHandlerOrderID,
			testSalesHandlerTenantID,
		),
		OrderID:      testSalesHandlerOrderID,
		OrderNumber:  testSalesHandlerOrderNumber,
		CustomerID:   testSalesHandlerCustomerID,
		WarehouseID:  &warehouseID,
		Items: []trade.SalesOrderItemInfo{
			{
				ItemID:      uuid.New(),
				ProductID:   testSalesHandlerProductID,
				ProductName: "Product A",
				ProductCode: "PROD-A",
				Quantity:    decimal.NewFromInt(5),
				UnitPrice:   decimal.NewFromFloat(100.00),
				Amount:      decimal.NewFromFloat(500.00),
				Unit:        "pcs",
			},
		},
		CancelReason: "Customer cancelled",
		WasConfirmed: true, // Order was confirmed, locks need release
	}

	// Mock UnlockBySource
	mockService.On("UnlockBySource", ctx, testSalesHandlerTenantID, "SALES_ORDER", testSalesHandlerOrderID.String()).Return(1, nil)

	testHandler := &testableCancelledHandler{
		mockService: mockService,
		logger:      logger,
	}
	err := testHandler.Handle(ctx, event)

	assert.NoError(t, err)
	mockService.AssertExpectations(t)
}

func TestSalesOrderCancelledHandler_Handle_WasNotConfirmed(t *testing.T) {
	mockService := new(MockInventoryServiceForSales)
	logger := zap.NewNop()

	ctx := context.Background()

	event := &trade.SalesOrderCancelledEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypeSalesOrderCancelled,
			trade.AggregateTypeSalesOrder,
			testSalesHandlerOrderID,
			testSalesHandlerTenantID,
		),
		OrderID:      testSalesHandlerOrderID,
		OrderNumber:  testSalesHandlerOrderNumber,
		CustomerID:   testSalesHandlerCustomerID,
		WarehouseID:  nil,
		Items: []trade.SalesOrderItemInfo{
			{
				ItemID:      uuid.New(),
				ProductID:   testSalesHandlerProductID,
				ProductName: "Product A",
				ProductCode: "PROD-A",
				Quantity:    decimal.NewFromInt(5),
				UnitPrice:   decimal.NewFromFloat(100.00),
				Amount:      decimal.NewFromFloat(500.00),
				Unit:        "pcs",
			},
		},
		CancelReason: "Customer cancelled",
		WasConfirmed: false, // Order was not confirmed, no locks to release
	}

	// UnlockBySource should NOT be called
	testHandler := &testableCancelledHandler{
		mockService: mockService,
		logger:      logger,
	}
	err := testHandler.Handle(ctx, event)

	assert.NoError(t, err)
	mockService.AssertNotCalled(t, "UnlockBySource")
}

func TestSalesOrderCancelledHandler_Handle_WrongEventType(t *testing.T) {
	logger := zap.NewNop()
	handler := NewSalesOrderCancelledHandler(nil, logger)

	ctx := context.Background()

	wrongEvent := &trade.SalesOrderCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypeSalesOrderCreated,
			trade.AggregateTypeSalesOrder,
			testSalesHandlerOrderID,
			testSalesHandlerTenantID,
		),
		OrderID:      testSalesHandlerOrderID,
		OrderNumber:  testSalesHandlerOrderNumber,
		CustomerID:   testSalesHandlerCustomerID,
		CustomerName: testSalesHandlerCustomerName,
	}

	err := handler.Handle(ctx, wrongEvent)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

func TestNewSalesOrderConfirmedHandler(t *testing.T) {
	logger := zap.NewNop()
	handler := NewSalesOrderConfirmedHandler(nil, logger)

	assert.NotNil(t, handler)
	assert.Equal(t, logger, handler.logger)
}

func TestNewSalesOrderShippedHandler(t *testing.T) {
	logger := zap.NewNop()
	handler := NewSalesOrderShippedHandler(nil, logger)

	assert.NotNil(t, handler)
	assert.Equal(t, logger, handler.logger)
}

func TestNewSalesOrderCancelledHandler(t *testing.T) {
	logger := zap.NewNop()
	handler := NewSalesOrderCancelledHandler(nil, logger)

	assert.NotNil(t, handler)
	assert.Equal(t, logger, handler.logger)
}

// ==================== Testable Handler Wrappers ====================

// testableConfirmHandler is a helper for testing SalesOrderConfirmedHandler with mock services
type testableConfirmHandler struct {
	mockService *MockInventoryServiceForSales
	logger      *zap.Logger
}

func (h *testableConfirmHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	confirmedEvent, ok := event.(*trade.SalesOrderConfirmedEvent)
	if !ok {
		return errors.New("unexpected event type")
	}

	if confirmedEvent.WarehouseID == nil || *confirmedEvent.WarehouseID == uuid.Nil {
		return errors.New("warehouse ID is required for stock locking")
	}

	var lastErr error
	for _, item := range confirmedEvent.Items {
		req := inventoryapp.LockStockRequest{
			WarehouseID: *confirmedEvent.WarehouseID,
			ProductID:   item.ProductID,
			Quantity:    item.Quantity,
			SourceType:  "SALES_ORDER",
			SourceID:    confirmedEvent.OrderID.String(),
		}

		if _, err := h.mockService.LockStock(ctx, event.TenantID(), req); err != nil {
			lastErr = err
			continue
		}
	}

	if lastErr != nil {
		return errors.New("some items failed to lock: " + lastErr.Error())
	}

	return nil
}

// testableShippedHandler is a helper for testing SalesOrderShippedHandler with mock services
type testableShippedHandler struct {
	mockService *MockInventoryServiceForSales
	logger      *zap.Logger
}

func (h *testableShippedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	shippedEvent, ok := event.(*trade.SalesOrderShippedEvent)
	if !ok {
		return errors.New("unexpected event type")
	}

	if shippedEvent.WarehouseID == uuid.Nil {
		return errors.New("warehouse ID is required for stock deduction")
	}

	sourceType := "SALES_ORDER"
	sourceID := shippedEvent.OrderID.String()

	locks, err := h.mockService.GetLocksBySource(ctx, sourceType, sourceID)
	if err != nil {
		return err
	}

	lockByProduct := make(map[uuid.UUID]uuid.UUID)
	for _, lock := range locks {
		if !lock.Released && !lock.Consumed {
			lockByProduct[lock.ProductID] = lock.ID
		}
	}

	var lastErr error
	for _, item := range shippedEvent.Items {
		lockID, exists := lockByProduct[item.ProductID]
		if !exists {
			continue
		}

		req := inventoryapp.DeductStockRequest{
			LockID:     lockID,
			SourceType: sourceType,
			SourceID:   sourceID,
		}

		if err := h.mockService.DeductStock(ctx, event.TenantID(), req); err != nil {
			lastErr = err
			continue
		}
	}

	if lastErr != nil {
		return errors.New("some items failed to deduct: " + lastErr.Error())
	}

	return nil
}

// testableCancelledHandler is a helper for testing SalesOrderCancelledHandler with mock services
type testableCancelledHandler struct {
	mockService *MockInventoryServiceForSales
	logger      *zap.Logger
}

func (h *testableCancelledHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	cancelledEvent, ok := event.(*trade.SalesOrderCancelledEvent)
	if !ok {
		return errors.New("unexpected event type")
	}

	if !cancelledEvent.WasConfirmed {
		return nil
	}

	sourceType := "SALES_ORDER"
	sourceID := cancelledEvent.OrderID.String()

	_, err := h.mockService.UnlockBySource(ctx, event.TenantID(), sourceType, sourceID)
	if err != nil {
		return errors.New("failed to release stock locks: " + err.Error())
	}

	return nil
}
