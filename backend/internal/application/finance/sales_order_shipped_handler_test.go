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
	"go.uber.org/zap"
)

// MockAccountReceivableRepository is a mock implementation of AccountReceivableRepository
type MockAccountReceivableRepository struct {
	mock.Mock
}

func (m *MockAccountReceivableRepository) FindByID(ctx context.Context, id uuid.UUID) (*finance.AccountReceivable, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.AccountReceivable), args.Error(1)
}

func (m *MockAccountReceivableRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*finance.AccountReceivable, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.AccountReceivable), args.Error(1)
}

func (m *MockAccountReceivableRepository) FindByReceivableNumber(ctx context.Context, tenantID uuid.UUID, receivableNumber string) (*finance.AccountReceivable, error) {
	args := m.Called(ctx, tenantID, receivableNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.AccountReceivable), args.Error(1)
}

func (m *MockAccountReceivableRepository) FindBySource(ctx context.Context, tenantID uuid.UUID, sourceType finance.SourceType, sourceID uuid.UUID) (*finance.AccountReceivable, error) {
	args := m.Called(ctx, tenantID, sourceType, sourceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.AccountReceivable), args.Error(1)
}

func (m *MockAccountReceivableRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter finance.AccountReceivableFilter) ([]finance.AccountReceivable, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]finance.AccountReceivable), args.Error(1)
}

func (m *MockAccountReceivableRepository) FindByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, filter finance.AccountReceivableFilter) ([]finance.AccountReceivable, error) {
	args := m.Called(ctx, tenantID, customerID, filter)
	return args.Get(0).([]finance.AccountReceivable), args.Error(1)
}

func (m *MockAccountReceivableRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status finance.ReceivableStatus, filter finance.AccountReceivableFilter) ([]finance.AccountReceivable, error) {
	args := m.Called(ctx, tenantID, status, filter)
	return args.Get(0).([]finance.AccountReceivable), args.Error(1)
}

func (m *MockAccountReceivableRepository) FindOutstanding(ctx context.Context, tenantID, customerID uuid.UUID) ([]finance.AccountReceivable, error) {
	args := m.Called(ctx, tenantID, customerID)
	return args.Get(0).([]finance.AccountReceivable), args.Error(1)
}

func (m *MockAccountReceivableRepository) FindOverdue(ctx context.Context, tenantID uuid.UUID, filter finance.AccountReceivableFilter) ([]finance.AccountReceivable, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]finance.AccountReceivable), args.Error(1)
}

func (m *MockAccountReceivableRepository) Save(ctx context.Context, receivable *finance.AccountReceivable) error {
	args := m.Called(ctx, receivable)
	return args.Error(0)
}

func (m *MockAccountReceivableRepository) SaveWithLock(ctx context.Context, receivable *finance.AccountReceivable) error {
	args := m.Called(ctx, receivable)
	return args.Error(0)
}

func (m *MockAccountReceivableRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAccountReceivableRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *MockAccountReceivableRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter finance.AccountReceivableFilter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAccountReceivableRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status finance.ReceivableStatus) (int64, error) {
	args := m.Called(ctx, tenantID, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAccountReceivableRepository) CountByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID, customerID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAccountReceivableRepository) CountOverdue(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAccountReceivableRepository) SumOutstandingByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (decimal.Decimal, error) {
	args := m.Called(ctx, tenantID, customerID)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockAccountReceivableRepository) SumOutstandingForTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockAccountReceivableRepository) SumOverdueForTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockAccountReceivableRepository) ExistsByReceivableNumber(ctx context.Context, tenantID uuid.UUID, receivableNumber string) (bool, error) {
	args := m.Called(ctx, tenantID, receivableNumber)
	return args.Bool(0), args.Error(1)
}

func (m *MockAccountReceivableRepository) ExistsBySource(ctx context.Context, tenantID uuid.UUID, sourceType finance.SourceType, sourceID uuid.UUID) (bool, error) {
	args := m.Called(ctx, tenantID, sourceType, sourceID)
	return args.Bool(0), args.Error(1)
}

func (m *MockAccountReceivableRepository) GenerateReceivableNumber(ctx context.Context, tenantID uuid.UUID) (string, error) {
	args := m.Called(ctx, tenantID)
	return args.String(0), args.Error(1)
}

// Verify interface compliance
var _ finance.AccountReceivableRepository = (*MockAccountReceivableRepository)(nil)

// Test helper functions
func newTestLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func newTestSalesOrderShippedEvent(tenantID, orderID, customerID uuid.UUID) *trade.SalesOrderShippedEvent {
	return &trade.SalesOrderShippedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(trade.EventTypeSalesOrderShipped, trade.AggregateTypeSalesOrder, orderID, tenantID),
		OrderID:         orderID,
		OrderNumber:     "SO-20260124-001",
		CustomerID:      customerID,
		CustomerName:    "Test Customer",
		WarehouseID:     uuid.New(),
		Items: []trade.SalesOrderItemInfo{
			{
				ItemID:      uuid.New(),
				ProductID:   uuid.New(),
				ProductName: "Test Product",
				ProductCode: "P001",
				Quantity:    decimal.NewFromInt(10),
				UnitPrice:   decimal.NewFromFloat(99.99),
				Amount:      decimal.NewFromFloat(999.90),
				Unit:        "pcs",
			},
		},
		TotalAmount:   decimal.NewFromFloat(999.90),
		PayableAmount: decimal.NewFromFloat(999.90),
	}
}

// Tests for SalesOrderShippedHandler
func TestSalesOrderShippedHandler_EventTypes(t *testing.T) {
	mockRepo := new(MockAccountReceivableRepository)
	handler := NewSalesOrderShippedHandler(mockRepo, newTestLogger())

	eventTypes := handler.EventTypes()

	assert.Equal(t, []string{trade.EventTypeSalesOrderShipped}, eventTypes)
}

func TestSalesOrderShippedHandler_Handle_Success(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountReceivableRepository)
	handler := NewSalesOrderShippedHandler(mockRepo, newTestLogger())

	tenantID := uuid.New()
	orderID := uuid.New()
	customerID := uuid.New()
	event := newTestSalesOrderShippedEvent(tenantID, orderID, customerID)

	// Setup expectations
	mockRepo.On("ExistsBySource", ctx, tenantID, finance.SourceTypeSalesOrder, orderID).Return(false, nil)
	mockRepo.On("GenerateReceivableNumber", ctx, tenantID).Return("AR-20260124-001", nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*finance.AccountReceivable")).Return(nil)

	// Execute
	err := handler.Handle(ctx, event)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSalesOrderShippedHandler_Handle_WrongEventType(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountReceivableRepository)
	handler := NewSalesOrderShippedHandler(mockRepo, newTestLogger())

	// Create a different event type
	wrongEvent := &trade.SalesOrderCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(trade.EventTypeSalesOrderCreated, trade.AggregateTypeSalesOrder, uuid.New(), uuid.New()),
	}

	// Execute
	err := handler.Handle(ctx, wrongEvent)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

func TestSalesOrderShippedHandler_Handle_IdempotentWhenExisting(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountReceivableRepository)
	handler := NewSalesOrderShippedHandler(mockRepo, newTestLogger())

	tenantID := uuid.New()
	orderID := uuid.New()
	customerID := uuid.New()
	event := newTestSalesOrderShippedEvent(tenantID, orderID, customerID)

	// Already exists
	mockRepo.On("ExistsBySource", ctx, tenantID, finance.SourceTypeSalesOrder, orderID).Return(true, nil)

	// Execute
	err := handler.Handle(ctx, event)

	// Assert - no error, just skipped
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	// Save should NOT be called
	mockRepo.AssertNotCalled(t, "Save")
}

func TestSalesOrderShippedHandler_Handle_SkipZeroAmount(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountReceivableRepository)
	handler := NewSalesOrderShippedHandler(mockRepo, newTestLogger())

	tenantID := uuid.New()
	orderID := uuid.New()
	customerID := uuid.New()
	event := newTestSalesOrderShippedEvent(tenantID, orderID, customerID)
	event.PayableAmount = decimal.Zero // Fully prepaid

	// Not existing
	mockRepo.On("ExistsBySource", ctx, tenantID, finance.SourceTypeSalesOrder, orderID).Return(false, nil)

	// Execute
	err := handler.Handle(ctx, event)

	// Assert - no error, just skipped
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	// GenerateReceivableNumber and Save should NOT be called
	mockRepo.AssertNotCalled(t, "GenerateReceivableNumber")
	mockRepo.AssertNotCalled(t, "Save")
}

func TestSalesOrderShippedHandler_Handle_ExistsCheckError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountReceivableRepository)
	handler := NewSalesOrderShippedHandler(mockRepo, newTestLogger())

	tenantID := uuid.New()
	orderID := uuid.New()
	customerID := uuid.New()
	event := newTestSalesOrderShippedEvent(tenantID, orderID, customerID)

	// Database error
	mockRepo.On("ExistsBySource", ctx, tenantID, finance.SourceTypeSalesOrder, orderID).Return(false, errors.New("db error"))

	// Execute
	err := handler.Handle(ctx, event)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check existing receivable")
}

func TestSalesOrderShippedHandler_Handle_GenerateNumberError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountReceivableRepository)
	handler := NewSalesOrderShippedHandler(mockRepo, newTestLogger())

	tenantID := uuid.New()
	orderID := uuid.New()
	customerID := uuid.New()
	event := newTestSalesOrderShippedEvent(tenantID, orderID, customerID)

	mockRepo.On("ExistsBySource", ctx, tenantID, finance.SourceTypeSalesOrder, orderID).Return(false, nil)
	mockRepo.On("GenerateReceivableNumber", ctx, tenantID).Return("", errors.New("generate error"))

	// Execute
	err := handler.Handle(ctx, event)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate receivable number")
}

func TestSalesOrderShippedHandler_Handle_SaveError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountReceivableRepository)
	handler := NewSalesOrderShippedHandler(mockRepo, newTestLogger())

	tenantID := uuid.New()
	orderID := uuid.New()
	customerID := uuid.New()
	event := newTestSalesOrderShippedEvent(tenantID, orderID, customerID)

	mockRepo.On("ExistsBySource", ctx, tenantID, finance.SourceTypeSalesOrder, orderID).Return(false, nil)
	mockRepo.On("GenerateReceivableNumber", ctx, tenantID).Return("AR-20260124-001", nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*finance.AccountReceivable")).Return(errors.New("save error"))

	// Execute
	err := handler.Handle(ctx, event)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save account receivable")
}

func TestSalesOrderShippedHandler_Handle_CreatesCorrectReceivable(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountReceivableRepository)
	handler := NewSalesOrderShippedHandler(mockRepo, newTestLogger())

	tenantID := uuid.New()
	orderID := uuid.New()
	customerID := uuid.New()
	event := newTestSalesOrderShippedEvent(tenantID, orderID, customerID)
	receivableNumber := "AR-20260124-001"

	mockRepo.On("ExistsBySource", ctx, tenantID, finance.SourceTypeSalesOrder, orderID).Return(false, nil)
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
	assert.Equal(t, finance.SourceTypeSalesOrder, savedReceivable.SourceType)
	assert.Equal(t, orderID, savedReceivable.SourceID)
	assert.Equal(t, "SO-20260124-001", savedReceivable.SourceNumber)
	assert.True(t, savedReceivable.TotalAmount.Equal(decimal.NewFromFloat(999.90)))
	assert.Equal(t, finance.ReceivableStatusPending, savedReceivable.Status)
	assert.NotNil(t, savedReceivable.DueDate)
	assert.True(t, savedReceivable.DueDate.After(time.Now()))
}
