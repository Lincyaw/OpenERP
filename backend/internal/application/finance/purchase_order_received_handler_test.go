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

// MockAccountPayableRepository is a mock implementation of AccountPayableRepository
type MockAccountPayableRepository struct {
	mock.Mock
}

func (m *MockAccountPayableRepository) FindByID(ctx context.Context, id uuid.UUID) (*finance.AccountPayable, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.AccountPayable), args.Error(1)
}

func (m *MockAccountPayableRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*finance.AccountPayable, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.AccountPayable), args.Error(1)
}

func (m *MockAccountPayableRepository) FindByPayableNumber(ctx context.Context, tenantID uuid.UUID, payableNumber string) (*finance.AccountPayable, error) {
	args := m.Called(ctx, tenantID, payableNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.AccountPayable), args.Error(1)
}

func (m *MockAccountPayableRepository) FindBySource(ctx context.Context, tenantID uuid.UUID, sourceType finance.PayableSourceType, sourceID uuid.UUID) (*finance.AccountPayable, error) {
	args := m.Called(ctx, tenantID, sourceType, sourceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.AccountPayable), args.Error(1)
}

func (m *MockAccountPayableRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter finance.AccountPayableFilter) ([]finance.AccountPayable, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]finance.AccountPayable), args.Error(1)
}

func (m *MockAccountPayableRepository) FindBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID, filter finance.AccountPayableFilter) ([]finance.AccountPayable, error) {
	args := m.Called(ctx, tenantID, supplierID, filter)
	return args.Get(0).([]finance.AccountPayable), args.Error(1)
}

func (m *MockAccountPayableRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status finance.PayableStatus, filter finance.AccountPayableFilter) ([]finance.AccountPayable, error) {
	args := m.Called(ctx, tenantID, status, filter)
	return args.Get(0).([]finance.AccountPayable), args.Error(1)
}

func (m *MockAccountPayableRepository) FindOutstanding(ctx context.Context, tenantID, supplierID uuid.UUID) ([]finance.AccountPayable, error) {
	args := m.Called(ctx, tenantID, supplierID)
	return args.Get(0).([]finance.AccountPayable), args.Error(1)
}

func (m *MockAccountPayableRepository) FindOverdue(ctx context.Context, tenantID uuid.UUID, filter finance.AccountPayableFilter) ([]finance.AccountPayable, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]finance.AccountPayable), args.Error(1)
}

func (m *MockAccountPayableRepository) Save(ctx context.Context, payable *finance.AccountPayable) error {
	args := m.Called(ctx, payable)
	return args.Error(0)
}

func (m *MockAccountPayableRepository) SaveWithLock(ctx context.Context, payable *finance.AccountPayable) error {
	args := m.Called(ctx, payable)
	return args.Error(0)
}

func (m *MockAccountPayableRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAccountPayableRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *MockAccountPayableRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter finance.AccountPayableFilter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAccountPayableRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status finance.PayableStatus) (int64, error) {
	args := m.Called(ctx, tenantID, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAccountPayableRepository) CountBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID, supplierID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAccountPayableRepository) CountOverdue(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAccountPayableRepository) SumOutstandingBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (decimal.Decimal, error) {
	args := m.Called(ctx, tenantID, supplierID)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockAccountPayableRepository) SumOutstandingForTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockAccountPayableRepository) SumOverdueForTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockAccountPayableRepository) ExistsByPayableNumber(ctx context.Context, tenantID uuid.UUID, payableNumber string) (bool, error) {
	args := m.Called(ctx, tenantID, payableNumber)
	return args.Bool(0), args.Error(1)
}

func (m *MockAccountPayableRepository) ExistsBySource(ctx context.Context, tenantID uuid.UUID, sourceType finance.PayableSourceType, sourceID uuid.UUID) (bool, error) {
	args := m.Called(ctx, tenantID, sourceType, sourceID)
	return args.Bool(0), args.Error(1)
}

func (m *MockAccountPayableRepository) GeneratePayableNumber(ctx context.Context, tenantID uuid.UUID) (string, error) {
	args := m.Called(ctx, tenantID)
	return args.String(0), args.Error(1)
}

// Verify interface compliance
var _ finance.AccountPayableRepository = (*MockAccountPayableRepository)(nil)

// Test helper
func newTestLogger2() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func newTestPurchaseOrderReceivedEvent(tenantID, orderID, supplierID uuid.UUID, isFullyReceived bool) *trade.PurchaseOrderReceivedEvent {
	now := time.Now().AddDate(0, 1, 0) // 1 month from now
	return &trade.PurchaseOrderReceivedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(trade.EventTypePurchaseOrderReceived, trade.AggregateTypePurchaseOrder, orderID, tenantID),
		OrderID:         orderID,
		OrderNumber:     "PO-20260124-001",
		SupplierID:      supplierID,
		SupplierName:    "Test Supplier",
		WarehouseID:     uuid.New(),
		ReceivedItems: []trade.ReceivedItemInfo{
			{
				ItemID:      uuid.New(),
				ProductID:   uuid.New(),
				ProductName: "Test Product",
				ProductCode: "P001",
				Quantity:    decimal.NewFromInt(10),
				UnitCost:    decimal.NewFromFloat(50.00),
				Unit:        "pcs",
				BatchNumber: "BATCH001",
				ExpiryDate:  &now,
			},
		},
		TotalAmount:     decimal.NewFromFloat(500.00),
		PayableAmount:   decimal.NewFromFloat(500.00),
		IsFullyReceived: isFullyReceived,
	}
}

// Tests for PurchaseOrderReceivedHandler
func TestPurchaseOrderReceivedHandler_EventTypes(t *testing.T) {
	mockRepo := new(MockAccountPayableRepository)
	handler := NewPurchaseOrderReceivedHandler(mockRepo, newTestLogger2())

	eventTypes := handler.EventTypes()

	assert.Equal(t, []string{trade.EventTypePurchaseOrderReceived}, eventTypes)
}

func TestPurchaseOrderReceivedHandler_Handle_Success(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountPayableRepository)
	handler := NewPurchaseOrderReceivedHandler(mockRepo, newTestLogger2())

	tenantID := uuid.New()
	orderID := uuid.New()
	supplierID := uuid.New()
	event := newTestPurchaseOrderReceivedEvent(tenantID, orderID, supplierID, true) // Fully received

	// Setup expectations
	mockRepo.On("FindBySource", ctx, tenantID, finance.PayableSourceTypePurchaseOrder, orderID).Return(nil, shared.NewDomainError("NOT_FOUND", "not found"))
	mockRepo.On("GeneratePayableNumber", ctx, tenantID).Return("AP-20260124-001", nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*finance.AccountPayable")).Return(nil)

	// Execute
	err := handler.Handle(ctx, event)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestPurchaseOrderReceivedHandler_Handle_WrongEventType(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountPayableRepository)
	handler := NewPurchaseOrderReceivedHandler(mockRepo, newTestLogger2())

	// Create a different event type
	wrongEvent := &trade.PurchaseOrderCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(trade.EventTypePurchaseOrderCreated, trade.AggregateTypePurchaseOrder, uuid.New(), uuid.New()),
	}

	// Execute
	err := handler.Handle(ctx, wrongEvent)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

func TestPurchaseOrderReceivedHandler_Handle_SkipPartialReceive(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountPayableRepository)
	handler := NewPurchaseOrderReceivedHandler(mockRepo, newTestLogger2())

	tenantID := uuid.New()
	orderID := uuid.New()
	supplierID := uuid.New()
	event := newTestPurchaseOrderReceivedEvent(tenantID, orderID, supplierID, false) // Not fully received

	// No existing payable
	mockRepo.On("FindBySource", ctx, tenantID, finance.PayableSourceTypePurchaseOrder, orderID).Return(nil, shared.NewDomainError("NOT_FOUND", "not found"))

	// Execute
	err := handler.Handle(ctx, event)

	// Assert - no error, just skipped
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	// GeneratePayableNumber and Save should NOT be called
	mockRepo.AssertNotCalled(t, "GeneratePayableNumber")
	mockRepo.AssertNotCalled(t, "Save")
}

func TestPurchaseOrderReceivedHandler_Handle_IdempotentWhenExisting(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountPayableRepository)
	handler := NewPurchaseOrderReceivedHandler(mockRepo, newTestLogger2())

	tenantID := uuid.New()
	orderID := uuid.New()
	supplierID := uuid.New()
	event := newTestPurchaseOrderReceivedEvent(tenantID, orderID, supplierID, true)

	// Already exists - return existing payable
	existingPayable := &finance.AccountPayable{}
	existingPayable.ID = uuid.New()
	mockRepo.On("FindBySource", ctx, tenantID, finance.PayableSourceTypePurchaseOrder, orderID).Return(existingPayable, nil)

	// Execute
	err := handler.Handle(ctx, event)

	// Assert - no error, just skipped
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	// Save should NOT be called
	mockRepo.AssertNotCalled(t, "Save")
}

func TestPurchaseOrderReceivedHandler_Handle_SkipZeroAmount(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountPayableRepository)
	handler := NewPurchaseOrderReceivedHandler(mockRepo, newTestLogger2())

	tenantID := uuid.New()
	orderID := uuid.New()
	supplierID := uuid.New()
	event := newTestPurchaseOrderReceivedEvent(tenantID, orderID, supplierID, true)
	// Set received items to have zero quantity
	event.ReceivedItems = []trade.ReceivedItemInfo{
		{
			Quantity: decimal.Zero,
			UnitCost: decimal.NewFromFloat(50.00),
		},
	}

	// Execute
	err := handler.Handle(ctx, event)

	// Assert - no error, just skipped
	assert.NoError(t, err)
	// FindBySource should NOT be called since we skip early
	mockRepo.AssertNotCalled(t, "FindBySource")
	mockRepo.AssertNotCalled(t, "Save")
}

func TestPurchaseOrderReceivedHandler_Handle_FindSourceError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountPayableRepository)
	handler := NewPurchaseOrderReceivedHandler(mockRepo, newTestLogger2())

	tenantID := uuid.New()
	orderID := uuid.New()
	supplierID := uuid.New()
	event := newTestPurchaseOrderReceivedEvent(tenantID, orderID, supplierID, true)

	// Database error (not a NOT_FOUND error)
	mockRepo.On("FindBySource", ctx, tenantID, finance.PayableSourceTypePurchaseOrder, orderID).Return(nil, errors.New("database error"))

	// Execute
	err := handler.Handle(ctx, event)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check existing payable")
}

func TestPurchaseOrderReceivedHandler_Handle_GenerateNumberError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountPayableRepository)
	handler := NewPurchaseOrderReceivedHandler(mockRepo, newTestLogger2())

	tenantID := uuid.New()
	orderID := uuid.New()
	supplierID := uuid.New()
	event := newTestPurchaseOrderReceivedEvent(tenantID, orderID, supplierID, true)

	mockRepo.On("FindBySource", ctx, tenantID, finance.PayableSourceTypePurchaseOrder, orderID).Return(nil, shared.NewDomainError("NOT_FOUND", "not found"))
	mockRepo.On("GeneratePayableNumber", ctx, tenantID).Return("", errors.New("generate error"))

	// Execute
	err := handler.Handle(ctx, event)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate payable number")
}

func TestPurchaseOrderReceivedHandler_Handle_SaveError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountPayableRepository)
	handler := NewPurchaseOrderReceivedHandler(mockRepo, newTestLogger2())

	tenantID := uuid.New()
	orderID := uuid.New()
	supplierID := uuid.New()
	event := newTestPurchaseOrderReceivedEvent(tenantID, orderID, supplierID, true)

	mockRepo.On("FindBySource", ctx, tenantID, finance.PayableSourceTypePurchaseOrder, orderID).Return(nil, shared.NewDomainError("NOT_FOUND", "not found"))
	mockRepo.On("GeneratePayableNumber", ctx, tenantID).Return("AP-20260124-001", nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*finance.AccountPayable")).Return(errors.New("save error"))

	// Execute
	err := handler.Handle(ctx, event)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save account payable")
}

func TestPurchaseOrderReceivedHandler_Handle_CreatesCorrectPayable(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockAccountPayableRepository)
	handler := NewPurchaseOrderReceivedHandler(mockRepo, newTestLogger2())

	tenantID := uuid.New()
	orderID := uuid.New()
	supplierID := uuid.New()
	event := newTestPurchaseOrderReceivedEvent(tenantID, orderID, supplierID, true)
	payableNumber := "AP-20260124-001"

	mockRepo.On("FindBySource", ctx, tenantID, finance.PayableSourceTypePurchaseOrder, orderID).Return(nil, shared.NewDomainError("NOT_FOUND", "not found"))
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
	assert.Equal(t, finance.PayableSourceTypePurchaseOrder, savedPayable.SourceType)
	assert.Equal(t, orderID, savedPayable.SourceID)
	assert.Equal(t, "PO-20260124-001", savedPayable.SourceNumber)
	assert.True(t, savedPayable.TotalAmount.Equal(decimal.NewFromFloat(500.00)))
	assert.Equal(t, finance.PayableStatusPending, savedPayable.Status)
	assert.NotNil(t, savedPayable.DueDate)
	assert.True(t, savedPayable.DueDate.After(time.Now()))
}

// Test isNotFoundError helper function
func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "NOT_FOUND domain error",
			err:      shared.NewDomainError("NOT_FOUND", "not found"),
			expected: true,
		},
		{
			name:     "other domain error",
			err:      shared.NewDomainError("INVALID_STATE", "invalid"),
			expected: false,
		},
		{
			name:     "standard error",
			err:      errors.New("some error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNotFoundError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
