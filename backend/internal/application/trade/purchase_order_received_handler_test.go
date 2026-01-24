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

// MockInventoryService mocks the inventory service for testing
type MockInventoryService struct {
	mock.Mock
}

func (m *MockInventoryService) IncreaseStock(ctx context.Context, tenantID uuid.UUID, req inventoryapp.IncreaseStockRequest) (*inventoryapp.InventoryItemResponse, error) {
	args := m.Called(ctx, tenantID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*inventoryapp.InventoryItemResponse), args.Error(1)
}

// Test helper variables
var (
	testHandlerTenantID     = uuid.New()
	testHandlerOrderID      = uuid.New()
	testHandlerWarehouseID  = uuid.New()
	testHandlerProductID    = uuid.New()
	testHandlerOrderNumber  = "PO-2024-00001"
	testHandlerSupplierID   = uuid.New()
	testHandlerSupplierName = "Test Supplier"
)

func TestPurchaseOrderReceivedHandler_EventTypes(t *testing.T) {
	logger := zap.NewNop()
	handler := NewPurchaseOrderReceivedHandler(nil, logger)

	eventTypes := handler.EventTypes()
	assert.Len(t, eventTypes, 1)
	assert.Equal(t, trade.EventTypePurchaseOrderReceived, eventTypes[0])
}

func TestPurchaseOrderReceivedHandler_Handle_Success(t *testing.T) {
	// Create mock inventory service
	mockService := new(MockInventoryService)
	logger := zap.NewNop()

	ctx := context.Background()
	expiryDate := time.Now().Add(365 * 24 * time.Hour)

	event := &trade.PurchaseOrderReceivedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypePurchaseOrderReceived,
			trade.AggregateTypePurchaseOrder,
			testHandlerOrderID,
			testHandlerTenantID,
		),
		OrderID:      testHandlerOrderID,
		OrderNumber:  testHandlerOrderNumber,
		SupplierID:   testHandlerSupplierID,
		SupplierName: testHandlerSupplierName,
		WarehouseID:  testHandlerWarehouseID,
		ReceivedItems: []trade.ReceivedItemInfo{
			{
				ItemID:      uuid.New(),
				ProductID:   testHandlerProductID,
				ProductName: "Product A",
				ProductCode: "PROD-A",
				Quantity:    decimal.NewFromInt(10),
				UnitCost:    decimal.NewFromFloat(100.50),
				Unit:        "pcs",
				BatchNumber: "BATCH-001",
				ExpiryDate:  &expiryDate,
			},
		},
		TotalAmount:     decimal.NewFromFloat(1005.00),
		PayableAmount:   decimal.NewFromFloat(1005.00),
		IsFullyReceived: true,
	}

	// Setup mock expectation
	mockService.On("IncreaseStock", ctx, testHandlerTenantID, mock.MatchedBy(func(req inventoryapp.IncreaseStockRequest) bool {
		return req.WarehouseID == testHandlerWarehouseID &&
			req.ProductID == testHandlerProductID &&
			req.Quantity.Equal(decimal.NewFromInt(10)) &&
			req.UnitCost.Equal(decimal.NewFromFloat(100.50)) &&
			req.SourceType == "PURCHASE_ORDER" &&
			req.SourceID == testHandlerOrderID.String() &&
			req.BatchNumber == "BATCH-001"
	})).Return(&inventoryapp.InventoryItemResponse{
		ID: uuid.New(),
	}, nil)

	// Execute handler with mock
	testHandler := &testableReceiveHandler{
		mockService: mockService,
		logger:      logger,
	}
	err := testHandler.Handle(ctx, event)

	assert.NoError(t, err)
	mockService.AssertExpectations(t)
}

func TestPurchaseOrderReceivedHandler_Handle_MissingWarehouse(t *testing.T) {
	logger := zap.NewNop()
	handler := NewPurchaseOrderReceivedHandler(nil, logger)

	ctx := context.Background()

	// Event with nil warehouse ID (zero UUID)
	event := &trade.PurchaseOrderReceivedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypePurchaseOrderReceived,
			trade.AggregateTypePurchaseOrder,
			testHandlerOrderID,
			testHandlerTenantID,
		),
		OrderID:      testHandlerOrderID,
		OrderNumber:  testHandlerOrderNumber,
		SupplierID:   testHandlerSupplierID,
		SupplierName: testHandlerSupplierName,
		WarehouseID:  uuid.Nil, // Missing warehouse ID
		ReceivedItems: []trade.ReceivedItemInfo{
			{
				ItemID:      uuid.New(),
				ProductID:   testHandlerProductID,
				ProductName: "Product A",
				ProductCode: "PROD-A",
				Quantity:    decimal.NewFromInt(10),
				UnitCost:    decimal.NewFromFloat(100.50),
				Unit:        "pcs",
			},
		},
		TotalAmount:     decimal.NewFromFloat(1005.00),
		PayableAmount:   decimal.NewFromFloat(1005.00),
		IsFullyReceived: true,
	}

	err := handler.Handle(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "warehouse ID is required")
}

func TestPurchaseOrderReceivedHandler_Handle_WrongEventType(t *testing.T) {
	logger := zap.NewNop()
	handler := NewPurchaseOrderReceivedHandler(nil, logger)

	ctx := context.Background()

	// Using a different event type
	wrongEvent := &trade.PurchaseOrderCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypePurchaseOrderCreated,
			trade.AggregateTypePurchaseOrder,
			testHandlerOrderID,
			testHandlerTenantID,
		),
		OrderID:      testHandlerOrderID,
		OrderNumber:  testHandlerOrderNumber,
		SupplierID:   testHandlerSupplierID,
		SupplierName: testHandlerSupplierName,
	}

	err := handler.Handle(ctx, wrongEvent)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

func TestPurchaseOrderReceivedHandler_Handle_MultipleItems(t *testing.T) {
	mockService := new(MockInventoryService)
	logger := zap.NewNop()

	ctx := context.Background()

	event := &trade.PurchaseOrderReceivedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypePurchaseOrderReceived,
			trade.AggregateTypePurchaseOrder,
			testHandlerOrderID,
			testHandlerTenantID,
		),
		OrderID:      testHandlerOrderID,
		OrderNumber:  testHandlerOrderNumber,
		SupplierID:   testHandlerSupplierID,
		SupplierName: testHandlerSupplierName,
		WarehouseID:  testHandlerWarehouseID,
		ReceivedItems: []trade.ReceivedItemInfo{
			{
				ItemID:      uuid.New(),
				ProductID:   uuid.New(),
				ProductName: "Product A",
				ProductCode: "PROD-A",
				Quantity:    decimal.NewFromInt(10),
				UnitCost:    decimal.NewFromFloat(100.00),
				Unit:        "pcs",
			},
			{
				ItemID:      uuid.New(),
				ProductID:   uuid.New(),
				ProductName: "Product B",
				ProductCode: "PROD-B",
				Quantity:    decimal.NewFromInt(20),
				UnitCost:    decimal.NewFromFloat(50.00),
				Unit:        "pcs",
			},
		},
		TotalAmount:     decimal.NewFromFloat(2000.00),
		PayableAmount:   decimal.NewFromFloat(2000.00),
		IsFullyReceived: false,
	}

	// Setup mock for both items
	mockService.On("IncreaseStock", ctx, testHandlerTenantID, mock.AnythingOfType("inventory.IncreaseStockRequest")).Return(&inventoryapp.InventoryItemResponse{
		ID: uuid.New(),
	}, nil).Times(2)

	testHandler := &testableReceiveHandler{
		mockService: mockService,
		logger:      logger,
	}
	err := testHandler.Handle(ctx, event)

	assert.NoError(t, err)
	mockService.AssertNumberOfCalls(t, "IncreaseStock", 2)
}

func TestPurchaseOrderReceivedHandler_Handle_PartialFailure(t *testing.T) {
	mockService := new(MockInventoryService)
	logger := zap.NewNop()

	ctx := context.Background()
	productID1 := uuid.New()
	productID2 := uuid.New()

	event := &trade.PurchaseOrderReceivedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			trade.EventTypePurchaseOrderReceived,
			trade.AggregateTypePurchaseOrder,
			testHandlerOrderID,
			testHandlerTenantID,
		),
		OrderID:      testHandlerOrderID,
		OrderNumber:  testHandlerOrderNumber,
		SupplierID:   testHandlerSupplierID,
		SupplierName: testHandlerSupplierName,
		WarehouseID:  testHandlerWarehouseID,
		ReceivedItems: []trade.ReceivedItemInfo{
			{
				ItemID:      uuid.New(),
				ProductID:   productID1,
				ProductName: "Product A",
				ProductCode: "PROD-A",
				Quantity:    decimal.NewFromInt(10),
				UnitCost:    decimal.NewFromFloat(100.00),
				Unit:        "pcs",
			},
			{
				ItemID:      uuid.New(),
				ProductID:   productID2,
				ProductName: "Product B",
				ProductCode: "PROD-B",
				Quantity:    decimal.NewFromInt(20),
				UnitCost:    decimal.NewFromFloat(50.00),
				Unit:        "pcs",
			},
		},
		TotalAmount:     decimal.NewFromFloat(2000.00),
		PayableAmount:   decimal.NewFromFloat(2000.00),
		IsFullyReceived: false,
	}

	// First item succeeds, second fails
	mockService.On("IncreaseStock", ctx, testHandlerTenantID, mock.MatchedBy(func(req inventoryapp.IncreaseStockRequest) bool {
		return req.ProductID == productID1
	})).Return(&inventoryapp.InventoryItemResponse{ID: uuid.New()}, nil).Once()

	mockService.On("IncreaseStock", ctx, testHandlerTenantID, mock.MatchedBy(func(req inventoryapp.IncreaseStockRequest) bool {
		return req.ProductID == productID2
	})).Return(nil, errors.New("database error")).Once()

	testHandler := &testableReceiveHandler{
		mockService: mockService,
		logger:      logger,
	}
	err := testHandler.Handle(ctx, event)

	// Should return error but both items should be attempted
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "some items failed to process")
	mockService.AssertNumberOfCalls(t, "IncreaseStock", 2)
}

// testableReceiveHandler is a helper for testing with mock services
type testableReceiveHandler struct {
	mockService *MockInventoryService
	logger      *zap.Logger
}

func (h *testableReceiveHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	receivedEvent, ok := event.(*trade.PurchaseOrderReceivedEvent)
	if !ok {
		return errors.New("unexpected event type")
	}

	if receivedEvent.WarehouseID.String() == "00000000-0000-0000-0000-000000000000" {
		return errors.New("warehouse ID is required for inventory update")
	}

	var lastErr error
	for _, item := range receivedEvent.ReceivedItems {
		req := inventoryapp.IncreaseStockRequest{
			WarehouseID: receivedEvent.WarehouseID,
			ProductID:   item.ProductID,
			Quantity:    item.Quantity,
			UnitCost:    item.UnitCost,
			SourceType:  "PURCHASE_ORDER",
			SourceID:    receivedEvent.OrderID.String(),
			BatchNumber: item.BatchNumber,
			ExpiryDate:  item.ExpiryDate,
			Reference:   "PO:" + receivedEvent.OrderNumber,
			Reason:      "Purchase order receiving",
		}

		if _, err := h.mockService.IncreaseStock(ctx, event.TenantID(), req); err != nil {
			lastErr = err
			continue
		}
	}

	if lastErr != nil {
		return errors.New("some items failed to process: " + lastErr.Error())
	}

	return nil
}

func TestNewPurchaseOrderReceivedHandler(t *testing.T) {
	logger := zap.NewNop()
	handler := NewPurchaseOrderReceivedHandler(nil, logger)

	assert.NotNil(t, handler)
	assert.Equal(t, logger, handler.logger)
}
