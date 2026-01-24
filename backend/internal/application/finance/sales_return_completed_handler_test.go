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

// Test helper functions for sales return completed handler
func newTestSalesReturnCompletedEvent(tenantID, returnID, salesOrderID, customerID uuid.UUID) *trade.SalesReturnCompletedEvent {
	warehouseID := uuid.New()
	return &trade.SalesReturnCompletedEvent{
		BaseDomainEvent:  shared.NewBaseDomainEvent(trade.EventTypeSalesReturnCompleted, trade.AggregateTypeSalesReturn, returnID, tenantID),
		ReturnID:         returnID,
		ReturnNumber:     "SR-20260124-001",
		SalesOrderID:     salesOrderID,
		SalesOrderNumber: "SO-20260124-001",
		CustomerID:       customerID,
		CustomerName:     "Test Customer",
		WarehouseID:      warehouseID,
		Items: []trade.SalesReturnItemInfo{
			{
				ItemID:           uuid.New(),
				SalesOrderItemID: uuid.New(),
				ProductID:        uuid.New(),
				ProductName:      "Test Product",
				ProductCode:      "P001",
				ReturnQuantity:   decimal.NewFromInt(5),
				UnitPrice:        decimal.NewFromFloat(99.99),
				RefundAmount:     decimal.NewFromFloat(499.95),
				Unit:             "pcs",
			},
		},
		TotalRefund: decimal.NewFromFloat(499.95),
	}
}

// Tests for SalesReturnCompletedHandler
func TestSalesReturnCompletedHandler_EventTypes(t *testing.T) {
	mockRepo := new(MockAccountReceivableRepository)
	handler := NewSalesReturnCompletedHandler(mockRepo, newTestLogger())

	eventTypes := handler.EventTypes()

	assert.Equal(t, []string{trade.EventTypeSalesReturnCompleted}, eventTypes)
}

func TestSalesReturnCompletedHandler_Handle_Success(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountReceivableRepository)
	handler := NewSalesReturnCompletedHandler(mockRepo, newTestLogger())

	tenantID := uuid.New()
	returnID := uuid.New()
	salesOrderID := uuid.New()
	customerID := uuid.New()
	event := newTestSalesReturnCompletedEvent(tenantID, returnID, salesOrderID, customerID)

	// Setup expectations
	mockRepo.On("ExistsBySource", ctx, tenantID, finance.SourceTypeSalesReturn, returnID).Return(false, nil)
	mockRepo.On("GenerateReceivableNumber", ctx, tenantID).Return("AR-20260124-002", nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*finance.AccountReceivable")).Return(nil)

	// Execute
	err := handler.Handle(ctx, event)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSalesReturnCompletedHandler_Handle_WrongEventType(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountReceivableRepository)
	handler := NewSalesReturnCompletedHandler(mockRepo, newTestLogger())

	// Create a different event type
	wrongEvent := &trade.SalesOrderShippedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(trade.EventTypeSalesOrderShipped, trade.AggregateTypeSalesOrder, uuid.New(), uuid.New()),
	}

	// Execute
	err := handler.Handle(ctx, wrongEvent)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

func TestSalesReturnCompletedHandler_Handle_IdempotentWhenExisting(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountReceivableRepository)
	handler := NewSalesReturnCompletedHandler(mockRepo, newTestLogger())

	tenantID := uuid.New()
	returnID := uuid.New()
	salesOrderID := uuid.New()
	customerID := uuid.New()
	event := newTestSalesReturnCompletedEvent(tenantID, returnID, salesOrderID, customerID)

	// Already exists
	mockRepo.On("ExistsBySource", ctx, tenantID, finance.SourceTypeSalesReturn, returnID).Return(true, nil)

	// Execute
	err := handler.Handle(ctx, event)

	// Assert - no error, just skipped
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	// Save should NOT be called
	mockRepo.AssertNotCalled(t, "Save")
}

func TestSalesReturnCompletedHandler_Handle_SkipZeroRefund(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountReceivableRepository)
	handler := NewSalesReturnCompletedHandler(mockRepo, newTestLogger())

	tenantID := uuid.New()
	returnID := uuid.New()
	salesOrderID := uuid.New()
	customerID := uuid.New()
	event := newTestSalesReturnCompletedEvent(tenantID, returnID, salesOrderID, customerID)
	event.TotalRefund = decimal.Zero // Zero refund

	// Not existing
	mockRepo.On("ExistsBySource", ctx, tenantID, finance.SourceTypeSalesReturn, returnID).Return(false, nil)

	// Execute
	err := handler.Handle(ctx, event)

	// Assert - no error, just skipped
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	// GenerateReceivableNumber and Save should NOT be called
	mockRepo.AssertNotCalled(t, "GenerateReceivableNumber")
	mockRepo.AssertNotCalled(t, "Save")
}

func TestSalesReturnCompletedHandler_Handle_ExistsCheckError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountReceivableRepository)
	handler := NewSalesReturnCompletedHandler(mockRepo, newTestLogger())

	tenantID := uuid.New()
	returnID := uuid.New()
	salesOrderID := uuid.New()
	customerID := uuid.New()
	event := newTestSalesReturnCompletedEvent(tenantID, returnID, salesOrderID, customerID)

	// Database error
	mockRepo.On("ExistsBySource", ctx, tenantID, finance.SourceTypeSalesReturn, returnID).Return(false, errors.New("db error"))

	// Execute
	err := handler.Handle(ctx, event)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check existing receivable")
}

func TestSalesReturnCompletedHandler_Handle_GenerateNumberError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountReceivableRepository)
	handler := NewSalesReturnCompletedHandler(mockRepo, newTestLogger())

	tenantID := uuid.New()
	returnID := uuid.New()
	salesOrderID := uuid.New()
	customerID := uuid.New()
	event := newTestSalesReturnCompletedEvent(tenantID, returnID, salesOrderID, customerID)

	mockRepo.On("ExistsBySource", ctx, tenantID, finance.SourceTypeSalesReturn, returnID).Return(false, nil)
	mockRepo.On("GenerateReceivableNumber", ctx, tenantID).Return("", errors.New("generate error"))

	// Execute
	err := handler.Handle(ctx, event)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate receivable number")
}

func TestSalesReturnCompletedHandler_Handle_SaveError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountReceivableRepository)
	handler := NewSalesReturnCompletedHandler(mockRepo, newTestLogger())

	tenantID := uuid.New()
	returnID := uuid.New()
	salesOrderID := uuid.New()
	customerID := uuid.New()
	event := newTestSalesReturnCompletedEvent(tenantID, returnID, salesOrderID, customerID)

	mockRepo.On("ExistsBySource", ctx, tenantID, finance.SourceTypeSalesReturn, returnID).Return(false, nil)
	mockRepo.On("GenerateReceivableNumber", ctx, tenantID).Return("AR-20260124-002", nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*finance.AccountReceivable")).Return(errors.New("save error"))

	// Execute
	err := handler.Handle(ctx, event)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save red-letter account receivable")
}

func TestSalesReturnCompletedHandler_Handle_CreatesCorrectRedLetterReceivable(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountReceivableRepository)
	handler := NewSalesReturnCompletedHandler(mockRepo, newTestLogger())

	tenantID := uuid.New()
	returnID := uuid.New()
	salesOrderID := uuid.New()
	customerID := uuid.New()
	event := newTestSalesReturnCompletedEvent(tenantID, returnID, salesOrderID, customerID)
	receivableNumber := "AR-20260124-002"

	mockRepo.On("ExistsBySource", ctx, tenantID, finance.SourceTypeSalesReturn, returnID).Return(false, nil)
	mockRepo.On("GenerateReceivableNumber", ctx, tenantID).Return(receivableNumber, nil)

	var savedReceivable *finance.AccountReceivable
	mockRepo.On("Save", ctx, mock.AnythingOfType("*finance.AccountReceivable")).Run(func(args mock.Arguments) {
		savedReceivable = args.Get(1).(*finance.AccountReceivable)
	}).Return(nil)

	// Execute
	err := handler.Handle(ctx, event)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, savedReceivable)
	assert.Equal(t, tenantID, savedReceivable.TenantID)
	assert.Equal(t, receivableNumber, savedReceivable.ReceivableNumber)
	assert.Equal(t, customerID, savedReceivable.CustomerID)
	assert.Equal(t, "Test Customer", savedReceivable.CustomerName)
	// Verify it's a sales return type (red-letter)
	assert.Equal(t, finance.SourceTypeSalesReturn, savedReceivable.SourceType)
	assert.Equal(t, returnID, savedReceivable.SourceID)
	assert.Equal(t, "SR-20260124-001", savedReceivable.SourceNumber)
	assert.True(t, savedReceivable.TotalAmount.Equal(decimal.NewFromFloat(499.95)))
	assert.Equal(t, finance.ReceivableStatusPending, savedReceivable.Status)
	assert.NotNil(t, savedReceivable.DueDate)
	// Verify due date is today (immediate for red-letter)
	assert.True(t, savedReceivable.DueDate.Before(time.Now().Add(time.Minute)))
	// Verify remark mentions red-letter
	assert.Contains(t, savedReceivable.Remark, "Red-letter entry")
	assert.Contains(t, savedReceivable.Remark, "SR-20260124-001")
	assert.Contains(t, savedReceivable.Remark, "SO-20260124-001")
}
