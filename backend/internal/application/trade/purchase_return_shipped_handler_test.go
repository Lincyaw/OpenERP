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

// MockInventoryServiceForPurchaseReturn extends mock for purchase return handling
type MockInventoryServiceForPurchaseReturn struct {
	mock.Mock
}

func (m *MockInventoryServiceForPurchaseReturn) DecreaseStock(ctx context.Context, tenantID uuid.UUID, req inventoryapp.DecreaseStockRequest) error {
	args := m.Called(ctx, tenantID, req)
	return args.Error(0)
}

// Test helper variables for purchase return handlers
var (
	testPRHandlerTenantID        = uuid.New()
	testPRHandlerReturnID        = uuid.New()
	testPRHandlerOrderID         = uuid.New()
	testPRHandlerWarehouseID     = uuid.New()
	testPRHandlerProductID       = uuid.New()
	testPRHandlerReturnNumber    = "PR-2024-00001"
	testPRHandlerOrderNumber     = "PO-2024-00001"
	testPRHandlerSupplierID      = uuid.New()
	testPRHandlerSupplierName    = "Test Supplier"
)

// ==================== PurchaseReturnShippedHandler Tests ====================

func TestPurchaseReturnShippedHandler_EventTypes(t *testing.T) {
	logger := zap.NewNop()
	handler := NewPurchaseReturnShippedHandler(nil, logger)

	eventTypes := handler.EventTypes()
	assert.Len(t, eventTypes, 1)
	assert.Equal(t, trade.EventTypePurchaseReturnShipped, eventTypes[0])
}

func TestPurchaseReturnShippedHandler_Handle_Success(t *testing.T) {
	mockService := new(MockInventoryServiceForPurchaseReturn)
	logger := zap.NewNop()

	ctx := context.Background()

	event := &trade.PurchaseReturnShippedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypePurchaseReturnShipped,
			trade.AggregateTypePurchaseReturn,
			testPRHandlerReturnID,
			testPRHandlerTenantID,
		),
		ReturnID:            testPRHandlerReturnID,
		ReturnNumber:        testPRHandlerReturnNumber,
		PurchaseOrderID:     testPRHandlerOrderID,
		PurchaseOrderNumber: testPRHandlerOrderNumber,
		SupplierID:          testPRHandlerSupplierID,
		SupplierName:        testPRHandlerSupplierName,
		WarehouseID:         testPRHandlerWarehouseID,
		Items: []trade.PurchaseReturnItemInfo{
			{
				ItemID:         uuid.New(),
				ProductID:      testPRHandlerProductID,
				ProductName:    "Product A",
				ProductCode:    "PROD-A",
				ReturnQuantity: decimal.NewFromInt(3),
				UnitCost:       decimal.NewFromFloat(100.00),
				RefundAmount:   decimal.NewFromFloat(300.00),
				Unit:           "pcs",
			},
		},
		TotalRefund: decimal.NewFromFloat(300.00),
	}

	// Mock DecreaseStock
	mockService.On("DecreaseStock", ctx, testPRHandlerTenantID, mock.MatchedBy(func(req inventoryapp.DecreaseStockRequest) bool {
		return req.WarehouseID == testPRHandlerWarehouseID &&
			req.ProductID == testPRHandlerProductID &&
			req.Quantity.Equal(decimal.NewFromInt(3)) &&
			req.SourceType == "PURCHASE_RETURN" &&
			req.SourceID == testPRHandlerReturnID.String()
	})).Return(nil)

	testHandler := &testablePRShippedHandler{
		mockService: mockService,
		logger:      logger,
	}
	err := testHandler.Handle(ctx, event)

	assert.NoError(t, err)
	mockService.AssertExpectations(t)
}

func TestPurchaseReturnShippedHandler_Handle_MultipleItems(t *testing.T) {
	mockService := new(MockInventoryServiceForPurchaseReturn)
	logger := zap.NewNop()

	ctx := context.Background()
	productID1 := uuid.New()
	productID2 := uuid.New()

	event := &trade.PurchaseReturnShippedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypePurchaseReturnShipped,
			trade.AggregateTypePurchaseReturn,
			testPRHandlerReturnID,
			testPRHandlerTenantID,
		),
		ReturnID:            testPRHandlerReturnID,
		ReturnNumber:        testPRHandlerReturnNumber,
		PurchaseOrderID:     testPRHandlerOrderID,
		PurchaseOrderNumber: testPRHandlerOrderNumber,
		SupplierID:          testPRHandlerSupplierID,
		SupplierName:        testPRHandlerSupplierName,
		WarehouseID:         testPRHandlerWarehouseID,
		Items: []trade.PurchaseReturnItemInfo{
			{
				ItemID:         uuid.New(),
				ProductID:      productID1,
				ProductName:    "Product A",
				ProductCode:    "PROD-A",
				ReturnQuantity: decimal.NewFromInt(5),
				UnitCost:       decimal.NewFromFloat(100.00),
				RefundAmount:   decimal.NewFromFloat(500.00),
				Unit:           "pcs",
			},
			{
				ItemID:         uuid.New(),
				ProductID:      productID2,
				ProductName:    "Product B",
				ProductCode:    "PROD-B",
				ReturnQuantity: decimal.NewFromInt(2),
				UnitCost:       decimal.NewFromFloat(50.00),
				RefundAmount:   decimal.NewFromFloat(100.00),
				Unit:           "box",
			},
		},
		TotalRefund: decimal.NewFromFloat(600.00),
	}

	// Mock for both products
	mockService.On("DecreaseStock", ctx, testPRHandlerTenantID, mock.MatchedBy(func(req inventoryapp.DecreaseStockRequest) bool {
		return req.ProductID == productID1
	})).Return(nil)
	mockService.On("DecreaseStock", ctx, testPRHandlerTenantID, mock.MatchedBy(func(req inventoryapp.DecreaseStockRequest) bool {
		return req.ProductID == productID2
	})).Return(nil)

	testHandler := &testablePRShippedHandler{
		mockService: mockService,
		logger:      logger,
	}
	err := testHandler.Handle(ctx, event)

	assert.NoError(t, err)
	mockService.AssertNumberOfCalls(t, "DecreaseStock", 2)
	mockService.AssertExpectations(t)
}

func TestPurchaseReturnShippedHandler_Handle_MissingWarehouse(t *testing.T) {
	logger := zap.NewNop()
	handler := NewPurchaseReturnShippedHandler(nil, logger)

	ctx := context.Background()

	event := &trade.PurchaseReturnShippedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypePurchaseReturnShipped,
			trade.AggregateTypePurchaseReturn,
			testPRHandlerReturnID,
			testPRHandlerTenantID,
		),
		ReturnID:            testPRHandlerReturnID,
		ReturnNumber:        testPRHandlerReturnNumber,
		PurchaseOrderID:     testPRHandlerOrderID,
		PurchaseOrderNumber: testPRHandlerOrderNumber,
		SupplierID:          testPRHandlerSupplierID,
		SupplierName:        testPRHandlerSupplierName,
		WarehouseID:         uuid.Nil, // Missing warehouse
		Items: []trade.PurchaseReturnItemInfo{
			{
				ItemID:         uuid.New(),
				ProductID:      testPRHandlerProductID,
				ProductName:    "Product A",
				ProductCode:    "PROD-A",
				ReturnQuantity: decimal.NewFromInt(3),
				UnitCost:       decimal.NewFromFloat(100.00),
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

func TestPurchaseReturnShippedHandler_Handle_WrongEventType(t *testing.T) {
	logger := zap.NewNop()
	handler := NewPurchaseReturnShippedHandler(nil, logger)

	ctx := context.Background()

	// Using a different event type
	wrongEvent := &trade.PurchaseReturnCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypePurchaseReturnCreated,
			trade.AggregateTypePurchaseReturn,
			testPRHandlerReturnID,
			testPRHandlerTenantID,
		),
		ReturnID:            testPRHandlerReturnID,
		ReturnNumber:        testPRHandlerReturnNumber,
		PurchaseOrderID:     testPRHandlerOrderID,
		PurchaseOrderNumber: testPRHandlerOrderNumber,
		SupplierID:          testPRHandlerSupplierID,
		SupplierName:        testPRHandlerSupplierName,
	}

	err := handler.Handle(ctx, wrongEvent)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

func TestPurchaseReturnShippedHandler_Handle_PartialFailure(t *testing.T) {
	mockService := new(MockInventoryServiceForPurchaseReturn)
	logger := zap.NewNop()

	ctx := context.Background()
	productID1 := uuid.New()
	productID2 := uuid.New()

	event := &trade.PurchaseReturnShippedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypePurchaseReturnShipped,
			trade.AggregateTypePurchaseReturn,
			testPRHandlerReturnID,
			testPRHandlerTenantID,
		),
		ReturnID:            testPRHandlerReturnID,
		ReturnNumber:        testPRHandlerReturnNumber,
		PurchaseOrderID:     testPRHandlerOrderID,
		PurchaseOrderNumber: testPRHandlerOrderNumber,
		SupplierID:          testPRHandlerSupplierID,
		SupplierName:        testPRHandlerSupplierName,
		WarehouseID:         testPRHandlerWarehouseID,
		Items: []trade.PurchaseReturnItemInfo{
			{
				ItemID:         uuid.New(),
				ProductID:      productID1,
				ProductName:    "Product A",
				ProductCode:    "PROD-A",
				ReturnQuantity: decimal.NewFromInt(5),
				UnitCost:       decimal.NewFromFloat(100.00),
				RefundAmount:   decimal.NewFromFloat(500.00),
				Unit:           "pcs",
			},
			{
				ItemID:         uuid.New(),
				ProductID:      productID2,
				ProductName:    "Product B",
				ProductCode:    "PROD-B",
				ReturnQuantity: decimal.NewFromInt(2),
				UnitCost:       decimal.NewFromFloat(50.00),
				RefundAmount:   decimal.NewFromFloat(100.00),
				Unit:           "box",
			},
		},
		TotalRefund: decimal.NewFromFloat(600.00),
	}

	// Product 1 succeeds
	mockService.On("DecreaseStock", ctx, testPRHandlerTenantID, mock.MatchedBy(func(req inventoryapp.DecreaseStockRequest) bool {
		return req.ProductID == productID1
	})).Return(nil)

	// Product 2 fails
	mockService.On("DecreaseStock", ctx, testPRHandlerTenantID, mock.MatchedBy(func(req inventoryapp.DecreaseStockRequest) bool {
		return req.ProductID == productID2
	})).Return(errors.New("insufficient stock"))

	testHandler := &testablePRShippedHandler{
		mockService: mockService,
		logger:      logger,
	}
	err := testHandler.Handle(ctx, event)

	// Should return error but both items should be attempted
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "some items failed to deduct stock")
	mockService.AssertNumberOfCalls(t, "DecreaseStock", 2)
}

func TestPurchaseReturnShippedHandler_Handle_InsufficientStock(t *testing.T) {
	mockService := new(MockInventoryServiceForPurchaseReturn)
	logger := zap.NewNop()

	ctx := context.Background()

	event := &trade.PurchaseReturnShippedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypePurchaseReturnShipped,
			trade.AggregateTypePurchaseReturn,
			testPRHandlerReturnID,
			testPRHandlerTenantID,
		),
		ReturnID:            testPRHandlerReturnID,
		ReturnNumber:        testPRHandlerReturnNumber,
		PurchaseOrderID:     testPRHandlerOrderID,
		PurchaseOrderNumber: testPRHandlerOrderNumber,
		SupplierID:          testPRHandlerSupplierID,
		SupplierName:        testPRHandlerSupplierName,
		WarehouseID:         testPRHandlerWarehouseID,
		Items: []trade.PurchaseReturnItemInfo{
			{
				ItemID:         uuid.New(),
				ProductID:      testPRHandlerProductID,
				ProductName:    "Product A",
				ProductCode:    "PROD-A",
				ReturnQuantity: decimal.NewFromInt(100), // Large quantity
				UnitCost:       decimal.NewFromFloat(100.00),
				RefundAmount:   decimal.NewFromFloat(10000.00),
				Unit:           "pcs",
			},
		},
		TotalRefund: decimal.NewFromFloat(10000.00),
	}

	// Mock DecreaseStock to return insufficient stock error
	mockService.On("DecreaseStock", ctx, testPRHandlerTenantID, mock.Anything).
		Return(shared.NewDomainError("INSUFFICIENT_STOCK", "Insufficient available stock to decrease"))

	testHandler := &testablePRShippedHandler{
		mockService: mockService,
		logger:      logger,
	}
	err := testHandler.Handle(ctx, event)

	assert.Error(t, err)
	mockService.AssertExpectations(t)
}

func TestNewPurchaseReturnShippedHandler(t *testing.T) {
	logger := zap.NewNop()
	handler := NewPurchaseReturnShippedHandler(nil, logger)

	assert.NotNil(t, handler)
	assert.Equal(t, logger, handler.logger)
}

func TestPurchaseReturnShippedHandler_Handle_EmptyItems(t *testing.T) {
	logger := zap.NewNop()

	ctx := context.Background()

	event := &trade.PurchaseReturnShippedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypePurchaseReturnShipped,
			trade.AggregateTypePurchaseReturn,
			testPRHandlerReturnID,
			testPRHandlerTenantID,
		),
		ReturnID:            testPRHandlerReturnID,
		ReturnNumber:        testPRHandlerReturnNumber,
		PurchaseOrderID:     testPRHandlerOrderID,
		PurchaseOrderNumber: testPRHandlerOrderNumber,
		SupplierID:          testPRHandlerSupplierID,
		SupplierName:        testPRHandlerSupplierName,
		WarehouseID:         testPRHandlerWarehouseID,
		Items:               []trade.PurchaseReturnItemInfo{}, // Empty items
		TotalRefund:         decimal.Zero,
	}

	testHandler := &testablePRShippedHandler{
		mockService: new(MockInventoryServiceForPurchaseReturn),
		logger:      logger,
	}
	err := testHandler.Handle(ctx, event)

	// Should succeed with no items to process
	assert.NoError(t, err)
}

// ==================== Testable Handler Wrapper ====================

// testablePRShippedHandler is a helper for testing PurchaseReturnShippedHandler with mock services
type testablePRShippedHandler struct {
	mockService *MockInventoryServiceForPurchaseReturn
	logger      *zap.Logger
}

func (h *testablePRShippedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	shippedEvent, ok := event.(*trade.PurchaseReturnShippedEvent)
	if !ok {
		return errors.New("unexpected event type: expected PurchaseReturnShippedEvent")
	}

	if shippedEvent.WarehouseID == uuid.Nil {
		return errors.New("warehouse ID is required for stock deduction")
	}

	var lastErr error
	for _, item := range shippedEvent.Items {
		req := inventoryapp.DecreaseStockRequest{
			WarehouseID: shippedEvent.WarehouseID,
			ProductID:   item.ProductID,
			Quantity:    item.ReturnQuantity,
			SourceType:  "PURCHASE_RETURN",
			SourceID:    shippedEvent.ReturnID.String(),
			Reference:   "PR:" + shippedEvent.ReturnNumber,
			Reason:      "Purchase return shipped: " + shippedEvent.ReturnNumber,
		}

		if err := h.mockService.DecreaseStock(ctx, event.TenantID(), req); err != nil {
			lastErr = err
			continue
		}
	}

	if lastErr != nil {
		return errors.New("some items failed to deduct stock: " + lastErr.Error())
	}

	return nil
}
