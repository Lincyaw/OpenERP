package finance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/finance"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Test helper functions for purchase return completed handler
func newTestPurchaseReturnCompletedEvent(tenantID, returnID, purchaseOrderID, supplierID uuid.UUID) *trade.PurchaseReturnCompletedEvent {
	warehouseID := uuid.New()
	return &trade.PurchaseReturnCompletedEvent{
		BaseDomainEvent:     shared.NewBaseDomainEvent(trade.EventTypePurchaseReturnCompleted, trade.AggregateTypePurchaseReturn, returnID, tenantID),
		ReturnID:            returnID,
		ReturnNumber:        "PR-20260124-001",
		PurchaseOrderID:     purchaseOrderID,
		PurchaseOrderNumber: "PO-20260124-001",
		SupplierID:          supplierID,
		SupplierName:        "Test Supplier",
		WarehouseID:         warehouseID,
		Items: []trade.PurchaseReturnItemInfo{
			{
				ItemID:              uuid.New(),
				PurchaseOrderItemID: uuid.New(),
				ProductID:           uuid.New(),
				ProductName:         "Test Product",
				ProductCode:         "P001",
				ReturnQuantity:      decimal.NewFromInt(5),
				UnitCost:            decimal.NewFromFloat(50.00),
				RefundAmount:        decimal.NewFromFloat(250.00),
				Unit:                "pcs",
				BatchNumber:         "BATCH001",
			},
		},
		TotalRefund: decimal.NewFromFloat(250.00),
	}
}

// Tests for PurchaseReturnCompletedHandler
func TestPurchaseReturnCompletedHandler_EventTypes(t *testing.T) {
	mockRepo := new(MockAccountPayableRepository)
	handler := NewPurchaseReturnCompletedHandler(mockRepo, newTestLogger2())

	eventTypes := handler.EventTypes()

	assert.Equal(t, []string{trade.EventTypePurchaseReturnCompleted}, eventTypes)
}

func TestPurchaseReturnCompletedHandler_Handle_Success(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountPayableRepository)
	handler := NewPurchaseReturnCompletedHandler(mockRepo, newTestLogger2())

	tenantID := uuid.New()
	returnID := uuid.New()
	purchaseOrderID := uuid.New()
	supplierID := uuid.New()
	event := newTestPurchaseReturnCompletedEvent(tenantID, returnID, purchaseOrderID, supplierID)

	// Setup expectations
	mockRepo.On("ExistsBySource", ctx, tenantID, finance.PayableSourceTypePurchaseReturn, returnID).Return(false, nil)
	mockRepo.On("GeneratePayableNumber", ctx, tenantID).Return("AP-20260124-002", nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*finance.AccountPayable")).Return(nil)

	// Execute
	err := handler.Handle(ctx, event)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestPurchaseReturnCompletedHandler_Handle_WrongEventType(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountPayableRepository)
	handler := NewPurchaseReturnCompletedHandler(mockRepo, newTestLogger2())

	// Create a different event type
	wrongEvent := &trade.PurchaseOrderReceivedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(trade.EventTypePurchaseOrderReceived, trade.AggregateTypePurchaseOrder, uuid.New(), uuid.New()),
	}

	// Execute
	err := handler.Handle(ctx, wrongEvent)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

func TestPurchaseReturnCompletedHandler_Handle_IdempotentWhenExisting(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountPayableRepository)
	handler := NewPurchaseReturnCompletedHandler(mockRepo, newTestLogger2())

	tenantID := uuid.New()
	returnID := uuid.New()
	purchaseOrderID := uuid.New()
	supplierID := uuid.New()
	event := newTestPurchaseReturnCompletedEvent(tenantID, returnID, purchaseOrderID, supplierID)

	// Already exists
	mockRepo.On("ExistsBySource", ctx, tenantID, finance.PayableSourceTypePurchaseReturn, returnID).Return(true, nil)

	// Execute
	err := handler.Handle(ctx, event)

	// Assert - no error, just skipped
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	// Save should NOT be called
	mockRepo.AssertNotCalled(t, "Save")
}

func TestPurchaseReturnCompletedHandler_Handle_SkipZeroRefund(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountPayableRepository)
	handler := NewPurchaseReturnCompletedHandler(mockRepo, newTestLogger2())

	tenantID := uuid.New()
	returnID := uuid.New()
	purchaseOrderID := uuid.New()
	supplierID := uuid.New()
	event := newTestPurchaseReturnCompletedEvent(tenantID, returnID, purchaseOrderID, supplierID)
	event.TotalRefund = decimal.Zero // Zero refund

	// Not existing
	mockRepo.On("ExistsBySource", ctx, tenantID, finance.PayableSourceTypePurchaseReturn, returnID).Return(false, nil)

	// Execute
	err := handler.Handle(ctx, event)

	// Assert - no error, just skipped
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	// GeneratePayableNumber and Save should NOT be called
	mockRepo.AssertNotCalled(t, "GeneratePayableNumber")
	mockRepo.AssertNotCalled(t, "Save")
}

func TestPurchaseReturnCompletedHandler_Handle_ExistsCheckError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountPayableRepository)
	handler := NewPurchaseReturnCompletedHandler(mockRepo, newTestLogger2())

	tenantID := uuid.New()
	returnID := uuid.New()
	purchaseOrderID := uuid.New()
	supplierID := uuid.New()
	event := newTestPurchaseReturnCompletedEvent(tenantID, returnID, purchaseOrderID, supplierID)

	// Database error
	mockRepo.On("ExistsBySource", ctx, tenantID, finance.PayableSourceTypePurchaseReturn, returnID).Return(false, errors.New("db error"))

	// Execute
	err := handler.Handle(ctx, event)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check existing payable")
}

func TestPurchaseReturnCompletedHandler_Handle_GenerateNumberError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountPayableRepository)
	handler := NewPurchaseReturnCompletedHandler(mockRepo, newTestLogger2())

	tenantID := uuid.New()
	returnID := uuid.New()
	purchaseOrderID := uuid.New()
	supplierID := uuid.New()
	event := newTestPurchaseReturnCompletedEvent(tenantID, returnID, purchaseOrderID, supplierID)

	mockRepo.On("ExistsBySource", ctx, tenantID, finance.PayableSourceTypePurchaseReturn, returnID).Return(false, nil)
	mockRepo.On("GeneratePayableNumber", ctx, tenantID).Return("", errors.New("generate error"))

	// Execute
	err := handler.Handle(ctx, event)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate payable number")
}

func TestPurchaseReturnCompletedHandler_Handle_SaveError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountPayableRepository)
	handler := NewPurchaseReturnCompletedHandler(mockRepo, newTestLogger2())

	tenantID := uuid.New()
	returnID := uuid.New()
	purchaseOrderID := uuid.New()
	supplierID := uuid.New()
	event := newTestPurchaseReturnCompletedEvent(tenantID, returnID, purchaseOrderID, supplierID)

	mockRepo.On("ExistsBySource", ctx, tenantID, finance.PayableSourceTypePurchaseReturn, returnID).Return(false, nil)
	mockRepo.On("GeneratePayableNumber", ctx, tenantID).Return("AP-20260124-002", nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*finance.AccountPayable")).Return(errors.New("save error"))

	// Execute
	err := handler.Handle(ctx, event)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save red-letter account payable")
}

func TestPurchaseReturnCompletedHandler_Handle_CreatesCorrectRedLetterPayable(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountPayableRepository)
	handler := NewPurchaseReturnCompletedHandler(mockRepo, newTestLogger2())

	tenantID := uuid.New()
	returnID := uuid.New()
	purchaseOrderID := uuid.New()
	supplierID := uuid.New()
	event := newTestPurchaseReturnCompletedEvent(tenantID, returnID, purchaseOrderID, supplierID)
	payableNumber := "AP-20260124-002"

	mockRepo.On("ExistsBySource", ctx, tenantID, finance.PayableSourceTypePurchaseReturn, returnID).Return(false, nil)
	mockRepo.On("GeneratePayableNumber", ctx, tenantID).Return(payableNumber, nil)

	var savedPayable *finance.AccountPayable
	mockRepo.On("Save", ctx, mock.AnythingOfType("*finance.AccountPayable")).Run(func(args mock.Arguments) {
		savedPayable = args.Get(1).(*finance.AccountPayable)
	}).Return(nil)

	// Execute
	err := handler.Handle(ctx, event)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, savedPayable)
	assert.Equal(t, tenantID, savedPayable.TenantID)
	assert.Equal(t, payableNumber, savedPayable.PayableNumber)
	assert.Equal(t, supplierID, savedPayable.SupplierID)
	assert.Equal(t, "Test Supplier", savedPayable.SupplierName)
	// Verify it's a purchase return type (red-letter)
	assert.Equal(t, finance.PayableSourceTypePurchaseReturn, savedPayable.SourceType)
	assert.Equal(t, returnID, savedPayable.SourceID)
	assert.Equal(t, "PR-20260124-001", savedPayable.SourceNumber)
	assert.True(t, savedPayable.TotalAmount.Equal(decimal.NewFromFloat(250.00)))
	assert.Equal(t, finance.PayableStatusPending, savedPayable.Status)
	assert.NotNil(t, savedPayable.DueDate)
	// Verify due date is today (immediate for red-letter)
	assert.True(t, savedPayable.DueDate.Before(time.Now().Add(time.Minute)))
	// Verify remark mentions red-letter
	assert.Contains(t, savedPayable.Remark, "Red-letter entry")
	assert.Contains(t, savedPayable.Remark, "PR-20260124-001")
	assert.Contains(t, savedPayable.Remark, "PO-20260124-001")
}
