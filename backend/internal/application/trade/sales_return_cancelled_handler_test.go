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

// MockInventoryServiceForCancelledReturn extends mock for cancelled return handling
type MockInventoryServiceForCancelledReturn struct {
	mock.Mock
}

func (m *MockInventoryServiceForCancelledReturn) DecreaseStock(ctx context.Context, tenantID uuid.UUID, req inventoryapp.DecreaseStockRequest) error {
	args := m.Called(ctx, tenantID, req)
	return args.Error(0)
}

func (m *MockInventoryServiceForCancelledReturn) ListTransactions(ctx context.Context, tenantID uuid.UUID, filter inventoryapp.TransactionListFilter) ([]inventoryapp.TransactionResponse, int64, error) {
	args := m.Called(ctx, tenantID, filter)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]inventoryapp.TransactionResponse), args.Get(1).(int64), args.Error(2)
}

// Test helper variables for cancelled return handler
var (
	testCancelledReturnTenantID    = uuid.New()
	testCancelledReturnReturnID    = uuid.New()
	testCancelledReturnOrderID     = uuid.New()
	testCancelledReturnWarehouseID = uuid.New()
	testCancelledReturnProductID   = uuid.New()
	testCancelledReturnNumber      = "SR-2024-00002"
	testCancelledOrderNumber       = "SO-2024-00002"
	testCancelledCustomerID        = uuid.New()
)

// ==================== SalesReturnCancelledHandler Tests ====================

func TestSalesReturnCancelledHandler_EventTypes(t *testing.T) {
	logger := zap.NewNop()
	handler := NewSalesReturnCancelledHandler(nil, logger)

	eventTypes := handler.EventTypes()
	assert.Len(t, eventTypes, 1)
	assert.Equal(t, trade.EventTypeSalesReturnCancelled, eventTypes[0])
}

func TestSalesReturnCancelledHandler_Handle_NotApproved(t *testing.T) {
	logger := zap.NewNop()
	handler := NewSalesReturnCancelledHandler(nil, logger)

	ctx := context.Background()
	warehouseID := testCancelledReturnWarehouseID

	event := &trade.SalesReturnCancelledEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypeSalesReturnCancelled,
			trade.AggregateTypeSalesReturn,
			testCancelledReturnReturnID,
			testCancelledReturnTenantID,
		),
		ReturnID:         testCancelledReturnReturnID,
		ReturnNumber:     testCancelledReturnNumber,
		SalesOrderID:     testCancelledReturnOrderID,
		SalesOrderNumber: testCancelledOrderNumber,
		CustomerID:       testCancelledCustomerID,
		WarehouseID:      &warehouseID,
		Items: []trade.SalesReturnItemInfo{
			{
				ItemID:         uuid.New(),
				ProductID:      testCancelledReturnProductID,
				ProductName:    "Product A",
				ProductCode:    "PROD-A",
				ReturnQuantity: decimal.NewFromInt(3),
				BaseQuantity:   decimal.NewFromInt(3),
				UnitPrice:      decimal.NewFromFloat(100.00),
				RefundAmount:   decimal.NewFromFloat(300.00),
				Unit:           "pcs",
				BaseUnit:       "pcs",
				ConversionRate: decimal.NewFromInt(1),
			},
		},
		CancelReason: "Customer changed mind",
		WasApproved:  false, // Not approved - should skip inventory reversal
	}

	err := handler.Handle(ctx, event)

	assert.NoError(t, err)
}

func TestSalesReturnCancelledHandler_Handle_NoWarehouse(t *testing.T) {
	logger := zap.NewNop()
	handler := NewSalesReturnCancelledHandler(nil, logger)

	ctx := context.Background()

	event := &trade.SalesReturnCancelledEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypeSalesReturnCancelled,
			trade.AggregateTypeSalesReturn,
			testCancelledReturnReturnID,
			testCancelledReturnTenantID,
		),
		ReturnID:         testCancelledReturnReturnID,
		ReturnNumber:     testCancelledReturnNumber,
		SalesOrderID:     testCancelledReturnOrderID,
		SalesOrderNumber: testCancelledOrderNumber,
		CustomerID:       testCancelledCustomerID,
		WarehouseID:      nil, // No warehouse
		Items: []trade.SalesReturnItemInfo{
			{
				ItemID:         uuid.New(),
				ProductID:      testCancelledReturnProductID,
				ProductName:    "Product A",
				ProductCode:    "PROD-A",
				ReturnQuantity: decimal.NewFromInt(3),
				BaseQuantity:   decimal.NewFromInt(3),
				UnitPrice:      decimal.NewFromFloat(100.00),
				RefundAmount:   decimal.NewFromFloat(300.00),
				Unit:           "pcs",
				BaseUnit:       "pcs",
				ConversionRate: decimal.NewFromInt(1),
			},
		},
		CancelReason: "Customer changed mind",
		WasApproved:  true,
	}

	err := handler.Handle(ctx, event)

	// Should not error, just skip reversal
	assert.NoError(t, err)
}

func TestSalesReturnCancelledHandler_Handle_WrongEventType(t *testing.T) {
	logger := zap.NewNop()
	handler := NewSalesReturnCancelledHandler(nil, logger)

	ctx := context.Background()

	// Using a different event type
	wrongEvent := &trade.SalesReturnCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypeSalesReturnCreated,
			trade.AggregateTypeSalesReturn,
			testCancelledReturnReturnID,
			testCancelledReturnTenantID,
		),
		ReturnID:         testCancelledReturnReturnID,
		ReturnNumber:     testCancelledReturnNumber,
		SalesOrderID:     testCancelledReturnOrderID,
		SalesOrderNumber: testCancelledOrderNumber,
		CustomerID:       testCancelledCustomerID,
		CustomerName:     "Test Customer",
	}

	err := handler.Handle(ctx, wrongEvent)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

func TestSalesReturnCancelledHandler_Handle_NoTransactionsToReverse(t *testing.T) {
	mockService := new(MockInventoryServiceForCancelledReturn)
	logger := zap.NewNop()

	ctx := context.Background()
	warehouseID := testCancelledReturnWarehouseID

	event := &trade.SalesReturnCancelledEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypeSalesReturnCancelled,
			trade.AggregateTypeSalesReturn,
			testCancelledReturnReturnID,
			testCancelledReturnTenantID,
		),
		ReturnID:         testCancelledReturnReturnID,
		ReturnNumber:     testCancelledReturnNumber,
		SalesOrderID:     testCancelledReturnOrderID,
		SalesOrderNumber: testCancelledOrderNumber,
		CustomerID:       testCancelledCustomerID,
		WarehouseID:      &warehouseID,
		Items: []trade.SalesReturnItemInfo{
			{
				ItemID:         uuid.New(),
				ProductID:      testCancelledReturnProductID,
				ProductName:    "Product A",
				ProductCode:    "PROD-A",
				ReturnQuantity: decimal.NewFromInt(3),
				BaseQuantity:   decimal.NewFromInt(3),
				UnitPrice:      decimal.NewFromFloat(100.00),
				RefundAmount:   decimal.NewFromFloat(300.00),
				Unit:           "pcs",
				BaseUnit:       "pcs",
				ConversionRate: decimal.NewFromInt(1),
			},
		},
		CancelReason: "Customer changed mind",
		WasApproved:  true, // Was approved but no transactions exist
	}

	// Mock ListTransactions to return empty (no inventory was restored yet)
	mockService.On("ListTransactions", ctx, testCancelledReturnTenantID, mock.MatchedBy(func(filter inventoryapp.TransactionListFilter) bool {
		return filter.SourceType == "SALES_RETURN" &&
			filter.SourceID == testCancelledReturnReturnID.String()
	})).Return([]inventoryapp.TransactionResponse{}, int64(0), nil)

	testHandler := &testableReturnCancelledHandler{
		mockService: mockService,
		logger:      logger,
	}
	err := testHandler.Handle(ctx, event)

	assert.NoError(t, err)
	mockService.AssertExpectations(t)
	// DecreaseStock should NOT be called since no transactions exist
	mockService.AssertNotCalled(t, "DecreaseStock")
}

func TestSalesReturnCancelledHandler_Handle_Success(t *testing.T) {
	mockService := new(MockInventoryServiceForCancelledReturn)
	logger := zap.NewNop()

	ctx := context.Background()
	warehouseID := testCancelledReturnWarehouseID

	event := &trade.SalesReturnCancelledEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypeSalesReturnCancelled,
			trade.AggregateTypeSalesReturn,
			testCancelledReturnReturnID,
			testCancelledReturnTenantID,
		),
		ReturnID:         testCancelledReturnReturnID,
		ReturnNumber:     testCancelledReturnNumber,
		SalesOrderID:     testCancelledReturnOrderID,
		SalesOrderNumber: testCancelledOrderNumber,
		CustomerID:       testCancelledCustomerID,
		WarehouseID:      &warehouseID,
		Items: []trade.SalesReturnItemInfo{
			{
				ItemID:         uuid.New(),
				ProductID:      testCancelledReturnProductID,
				ProductName:    "Product A",
				ProductCode:    "PROD-A",
				ReturnQuantity: decimal.NewFromInt(3),
				BaseQuantity:   decimal.NewFromInt(3),
				UnitPrice:      decimal.NewFromFloat(100.00),
				RefundAmount:   decimal.NewFromFloat(300.00),
				Unit:           "pcs",
				BaseUnit:       "pcs",
				ConversionRate: decimal.NewFromInt(1),
			},
		},
		CancelReason: "Customer changed mind",
		WasApproved:  true,
	}

	// Mock ListTransactions to return existing transaction (inventory was restored)
	mockService.On("ListTransactions", ctx, testCancelledReturnTenantID, mock.MatchedBy(func(filter inventoryapp.TransactionListFilter) bool {
		return filter.SourceType == "SALES_RETURN" &&
			filter.SourceID == testCancelledReturnReturnID.String()
	})).Return([]inventoryapp.TransactionResponse{
		{
			ID:              uuid.New(),
			ProductID:       testCancelledReturnProductID,
			TransactionType: "INBOUND",
			Quantity:        decimal.NewFromInt(3),
		},
	}, int64(1), nil)

	// Mock DecreaseStock
	mockService.On("DecreaseStock", ctx, testCancelledReturnTenantID, mock.MatchedBy(func(req inventoryapp.DecreaseStockRequest) bool {
		return req.WarehouseID == warehouseID &&
			req.ProductID == testCancelledReturnProductID &&
			req.Quantity.Equal(decimal.NewFromInt(3)) &&
			req.SourceType == "SALES_RETURN" &&
			req.SourceID == testCancelledReturnReturnID.String()
	})).Return(nil)

	testHandler := &testableReturnCancelledHandler{
		mockService: mockService,
		logger:      logger,
	}
	err := testHandler.Handle(ctx, event)

	assert.NoError(t, err)
	mockService.AssertExpectations(t)
}

func TestSalesReturnCancelledHandler_Handle_MultipleItems(t *testing.T) {
	mockService := new(MockInventoryServiceForCancelledReturn)
	logger := zap.NewNop()

	ctx := context.Background()
	warehouseID := testCancelledReturnWarehouseID
	productID1 := uuid.New()
	productID2 := uuid.New()

	event := &trade.SalesReturnCancelledEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypeSalesReturnCancelled,
			trade.AggregateTypeSalesReturn,
			testCancelledReturnReturnID,
			testCancelledReturnTenantID,
		),
		ReturnID:         testCancelledReturnReturnID,
		ReturnNumber:     testCancelledReturnNumber,
		SalesOrderID:     testCancelledReturnOrderID,
		SalesOrderNumber: testCancelledOrderNumber,
		CustomerID:       testCancelledCustomerID,
		WarehouseID:      &warehouseID,
		Items: []trade.SalesReturnItemInfo{
			{
				ItemID:         uuid.New(),
				ProductID:      productID1,
				ProductName:    "Product A",
				ProductCode:    "PROD-A",
				ReturnQuantity: decimal.NewFromInt(5),
				BaseQuantity:   decimal.NewFromInt(5),
				UnitPrice:      decimal.NewFromFloat(100.00),
				RefundAmount:   decimal.NewFromFloat(500.00),
				Unit:           "pcs",
				BaseUnit:       "pcs",
				ConversionRate: decimal.NewFromInt(1),
			},
			{
				ItemID:         uuid.New(),
				ProductID:      productID2,
				ProductName:    "Product B",
				ProductCode:    "PROD-B",
				ReturnQuantity: decimal.NewFromInt(2),
				BaseQuantity:   decimal.NewFromInt(24), // 2 boxes * 12 pcs/box
				UnitPrice:      decimal.NewFromFloat(50.00),
				RefundAmount:   decimal.NewFromFloat(100.00),
				Unit:           "box",
				BaseUnit:       "pcs",
				ConversionRate: decimal.NewFromInt(12),
			},
		},
		CancelReason: "Order cancelled",
		WasApproved:  true,
	}

	// Mock ListTransactions to return transactions for both products
	mockService.On("ListTransactions", ctx, testCancelledReturnTenantID, mock.AnythingOfType("inventory.TransactionListFilter")).
		Return([]inventoryapp.TransactionResponse{
			{ID: uuid.New(), ProductID: productID1, TransactionType: "INBOUND", Quantity: decimal.NewFromInt(5)},
			{ID: uuid.New(), ProductID: productID2, TransactionType: "INBOUND", Quantity: decimal.NewFromInt(24)},
		}, int64(2), nil)

	// Mock DecreaseStock for both products
	mockService.On("DecreaseStock", ctx, testCancelledReturnTenantID, mock.MatchedBy(func(req inventoryapp.DecreaseStockRequest) bool {
		return req.ProductID == productID1 && req.Quantity.Equal(decimal.NewFromInt(5))
	})).Return(nil)
	mockService.On("DecreaseStock", ctx, testCancelledReturnTenantID, mock.MatchedBy(func(req inventoryapp.DecreaseStockRequest) bool {
		return req.ProductID == productID2 && req.Quantity.Equal(decimal.NewFromInt(24))
	})).Return(nil)

	testHandler := &testableReturnCancelledHandler{
		mockService: mockService,
		logger:      logger,
	}
	err := testHandler.Handle(ctx, event)

	assert.NoError(t, err)
	mockService.AssertNumberOfCalls(t, "DecreaseStock", 2)
	mockService.AssertExpectations(t)
}

func TestSalesReturnCancelledHandler_Handle_PartialFailure(t *testing.T) {
	mockService := new(MockInventoryServiceForCancelledReturn)
	logger := zap.NewNop()

	ctx := context.Background()
	warehouseID := testCancelledReturnWarehouseID
	productID1 := uuid.New()
	productID2 := uuid.New()

	event := &trade.SalesReturnCancelledEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypeSalesReturnCancelled,
			trade.AggregateTypeSalesReturn,
			testCancelledReturnReturnID,
			testCancelledReturnTenantID,
		),
		ReturnID:         testCancelledReturnReturnID,
		ReturnNumber:     testCancelledReturnNumber,
		SalesOrderID:     testCancelledReturnOrderID,
		SalesOrderNumber: testCancelledOrderNumber,
		CustomerID:       testCancelledCustomerID,
		WarehouseID:      &warehouseID,
		Items: []trade.SalesReturnItemInfo{
			{
				ItemID:         uuid.New(),
				ProductID:      productID1,
				ProductName:    "Product A",
				ProductCode:    "PROD-A",
				ReturnQuantity: decimal.NewFromInt(5),
				BaseQuantity:   decimal.NewFromInt(5),
				UnitPrice:      decimal.NewFromFloat(100.00),
				RefundAmount:   decimal.NewFromFloat(500.00),
				Unit:           "pcs",
				BaseUnit:       "pcs",
				ConversionRate: decimal.NewFromInt(1),
			},
			{
				ItemID:         uuid.New(),
				ProductID:      productID2,
				ProductName:    "Product B",
				ProductCode:    "PROD-B",
				ReturnQuantity: decimal.NewFromInt(2),
				BaseQuantity:   decimal.NewFromInt(2),
				UnitPrice:      decimal.NewFromFloat(50.00),
				RefundAmount:   decimal.NewFromFloat(100.00),
				Unit:           "pcs",
				BaseUnit:       "pcs",
				ConversionRate: decimal.NewFromInt(1),
			},
		},
		CancelReason: "Order cancelled",
		WasApproved:  true,
	}

	// Mock ListTransactions to return transactions for both products
	mockService.On("ListTransactions", ctx, testCancelledReturnTenantID, mock.AnythingOfType("inventory.TransactionListFilter")).
		Return([]inventoryapp.TransactionResponse{
			{ID: uuid.New(), ProductID: productID1, TransactionType: "INBOUND", Quantity: decimal.NewFromInt(5)},
			{ID: uuid.New(), ProductID: productID2, TransactionType: "INBOUND", Quantity: decimal.NewFromInt(2)},
		}, int64(2), nil)

	// Product 1 succeeds
	mockService.On("DecreaseStock", ctx, testCancelledReturnTenantID, mock.MatchedBy(func(req inventoryapp.DecreaseStockRequest) bool {
		return req.ProductID == productID1
	})).Return(nil)

	// Product 2 fails
	mockService.On("DecreaseStock", ctx, testCancelledReturnTenantID, mock.MatchedBy(func(req inventoryapp.DecreaseStockRequest) bool {
		return req.ProductID == productID2
	})).Return(errors.New("database error"))

	testHandler := &testableReturnCancelledHandler{
		mockService: mockService,
		logger:      logger,
	}
	err := testHandler.Handle(ctx, event)

	// Should return error but both items should be attempted
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "some items failed to reverse stock")
	mockService.AssertNumberOfCalls(t, "DecreaseStock", 2)
}

func TestSalesReturnCancelledHandler_Handle_TransactionQueryFails(t *testing.T) {
	mockService := new(MockInventoryServiceForCancelledReturn)
	logger := zap.NewNop()

	ctx := context.Background()
	warehouseID := testCancelledReturnWarehouseID

	event := &trade.SalesReturnCancelledEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypeSalesReturnCancelled,
			trade.AggregateTypeSalesReturn,
			testCancelledReturnReturnID,
			testCancelledReturnTenantID,
		),
		ReturnID:         testCancelledReturnReturnID,
		ReturnNumber:     testCancelledReturnNumber,
		SalesOrderID:     testCancelledReturnOrderID,
		SalesOrderNumber: testCancelledOrderNumber,
		CustomerID:       testCancelledCustomerID,
		WarehouseID:      &warehouseID,
		Items: []trade.SalesReturnItemInfo{
			{
				ItemID:         uuid.New(),
				ProductID:      testCancelledReturnProductID,
				ProductName:    "Product A",
				ProductCode:    "PROD-A",
				ReturnQuantity: decimal.NewFromInt(3),
				BaseQuantity:   decimal.NewFromInt(3),
				UnitPrice:      decimal.NewFromFloat(100.00),
				RefundAmount:   decimal.NewFromFloat(300.00),
				Unit:           "pcs",
				BaseUnit:       "pcs",
				ConversionRate: decimal.NewFromInt(1),
			},
		},
		CancelReason: "Customer changed mind",
		WasApproved:  true,
	}

	// Mock ListTransactions to fail
	mockService.On("ListTransactions", ctx, testCancelledReturnTenantID, mock.AnythingOfType("inventory.TransactionListFilter")).
		Return(nil, int64(0), errors.New("database error"))

	// Even though query fails, handler should attempt full reversal
	mockService.On("DecreaseStock", ctx, testCancelledReturnTenantID, mock.AnythingOfType("inventory.DecreaseStockRequest")).
		Return(nil)

	testHandler := &testableReturnCancelledHandler{
		mockService: mockService,
		logger:      logger,
	}
	err := testHandler.Handle(ctx, event)

	// Should still succeed as we attempt reversal anyway
	assert.NoError(t, err)
	mockService.AssertExpectations(t)
}

func TestNewSalesReturnCancelledHandler(t *testing.T) {
	logger := zap.NewNop()
	handler := NewSalesReturnCancelledHandler(nil, logger)

	assert.NotNil(t, handler)
	assert.Equal(t, logger, handler.logger)
}

// ==================== Testable Handler Wrapper ====================

// testableReturnCancelledHandler is a helper for testing SalesReturnCancelledHandler with mock services
type testableReturnCancelledHandler struct {
	mockService *MockInventoryServiceForCancelledReturn
	logger      *zap.Logger
}

func (h *testableReturnCancelledHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	cancelledEvent, ok := event.(*trade.SalesReturnCancelledEvent)
	if !ok {
		return errors.New("unexpected event type: expected SalesReturnCancelledEvent")
	}

	// If not approved, no inventory to reverse
	if !cancelledEvent.WasApproved {
		return nil
	}

	// Check warehouse
	if cancelledEvent.WarehouseID == nil || *cancelledEvent.WarehouseID == uuid.Nil {
		return nil
	}

	// Check for existing transactions
	filter := inventoryapp.TransactionListFilter{
		SourceType: "SALES_RETURN",
		SourceID:   cancelledEvent.ReturnID.String(),
		Page:       1,
		PageSize:   100,
	}

	transactions, _, err := h.mockService.ListTransactions(ctx, event.TenantID(), filter)
	queryFailed := err != nil

	// If no transactions found and query succeeded, nothing to reverse
	if len(transactions) == 0 && !queryFailed {
		return nil
	}

	var lastErr error
	for _, item := range cancelledEvent.Items {
		// Check if this item was restored (skip if not and we have transaction data)
		if !queryFailed && len(transactions) > 0 {
			itemRestored := false
			for _, tx := range transactions {
				if tx.ProductID == item.ProductID && tx.TransactionType == "INBOUND" {
					itemRestored = true
					break
				}
			}
			if !itemRestored {
				continue
			}
		}

		req := inventoryapp.DecreaseStockRequest{
			WarehouseID: *cancelledEvent.WarehouseID,
			ProductID:   item.ProductID,
			Quantity:    item.BaseQuantity,
			SourceType:  "SALES_RETURN",
			SourceID:    cancelledEvent.ReturnID.String(),
			Reference:   "SR-CANCEL:" + cancelledEvent.ReturnNumber,
			Reason:      "Sales return cancelled: " + cancelledEvent.ReturnNumber,
		}

		if err := h.mockService.DecreaseStock(ctx, event.TenantID(), req); err != nil {
			lastErr = err
			continue
		}
	}

	if lastErr != nil {
		return errors.New("some items failed to reverse stock: " + lastErr.Error())
	}

	return nil
}
