package printing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/erp/backend/internal/domain/finance"
	"github.com/erp/backend/internal/domain/printing"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockAutoPrintRuleRepository is a mock implementation of AutoPrintRuleRepository
type MockAutoPrintRuleRepository struct {
	mock.Mock
}

func (m *MockAutoPrintRuleRepository) FindByID(ctx context.Context, id uuid.UUID) (*printing.AutoPrintRule, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*printing.AutoPrintRule), args.Error(1)
}

func (m *MockAutoPrintRuleRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*printing.AutoPrintRule, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*printing.AutoPrintRule), args.Error(1)
}

func (m *MockAutoPrintRuleRepository) FindAll(ctx context.Context, filter shared.Filter) ([]printing.AutoPrintRule, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]printing.AutoPrintRule), args.Error(1)
}

func (m *MockAutoPrintRuleRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]printing.AutoPrintRule, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]printing.AutoPrintRule), args.Error(1)
}

func (m *MockAutoPrintRuleRepository) FindByDocTypeAndEvent(ctx context.Context, tenantID uuid.UUID, docType printing.DocType, event printing.TriggerEvent) (*printing.AutoPrintRule, error) {
	args := m.Called(ctx, tenantID, docType, event)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*printing.AutoPrintRule), args.Error(1)
}

func (m *MockAutoPrintRuleRepository) FindEnabledByDocTypeAndEvent(ctx context.Context, tenantID uuid.UUID, docType printing.DocType, event printing.TriggerEvent) (*printing.AutoPrintRule, error) {
	args := m.Called(ctx, tenantID, docType, event)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*printing.AutoPrintRule), args.Error(1)
}

func (m *MockAutoPrintRuleRepository) FindEnabledForTenant(ctx context.Context, tenantID uuid.UUID) ([]printing.AutoPrintRule, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]printing.AutoPrintRule), args.Error(1)
}

func (m *MockAutoPrintRuleRepository) ExistsByDocTypeAndEvent(ctx context.Context, tenantID uuid.UUID, docType printing.DocType, event printing.TriggerEvent) (bool, error) {
	args := m.Called(ctx, tenantID, docType, event)
	return args.Bool(0), args.Error(1)
}

func (m *MockAutoPrintRuleRepository) Save(ctx context.Context, rule *printing.AutoPrintRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockAutoPrintRuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAutoPrintRuleRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *MockAutoPrintRuleRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAutoPrintRuleRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

// createTestAutoPrintRule creates a test auto print rule
func createTestAutoPrintRule(tenantID uuid.UUID, docType printing.DocType, triggerEvent printing.TriggerEvent, autoPrint bool) *printing.AutoPrintRule {
	rule, _ := printing.NewAutoPrintRule(
		tenantID,
		docType,
		triggerEvent,
		nil,
		autoPrint,
		1,
		"",
	)
	return rule
}

// createTestHandler creates a handler with mock dependencies for testing.
// This is used for tests that only need to test mapEventToTrigger or other methods
// that don't require the full handler functionality.
func createTestHandler() *AutoPrintHandler {
	mockRepo := new(MockAutoPrintRuleRepository)
	// Create a minimal PrintService - we won't actually call it in most tests
	mockPrintService := &PrintService{}
	return &AutoPrintHandler{
		ruleRepo:     mockRepo,
		printService: mockPrintService,
		redisClient:  nil,
		logger:       zap.NewNop(),
	}
}

func TestAutoPrintHandler_EventTypes(t *testing.T) {
	handler := createTestHandler()

	eventTypes := handler.EventTypes()

	// Verify all expected event types are present
	expectedTypes := []string{
		trade.EventTypeSalesOrderCreated,
		trade.EventTypeSalesOrderConfirmed,
		trade.EventTypeSalesOrderShipped,
		trade.EventTypeSalesOrderCompleted,
		trade.EventTypePurchaseOrderCreated,
		trade.EventTypePurchaseOrderConfirmed,
		trade.EventTypePurchaseOrderReceived,
		trade.EventTypePurchaseOrderCompleted,
		trade.EventTypeSalesReturnCreated,
		trade.EventTypeSalesReturnApproved,
		trade.EventTypeSalesReturnCompleted,
		trade.EventTypePurchaseReturnCreated,
		trade.EventTypePurchaseReturnApproved,
		trade.EventTypePurchaseReturnCompleted,
		"ReceiptVoucherCreated",
		"ReceiptVoucherConfirmed",
		"PaymentVoucherCreated",
		"PaymentVoucherConfirmed",
	}

	assert.Equal(t, len(expectedTypes), len(eventTypes))
	for _, expected := range expectedTypes {
		assert.Contains(t, eventTypes, expected)
	}
}

func TestAutoPrintHandler_MapEventToTrigger_SalesOrder(t *testing.T) {
	handler := createTestHandler()
	tenantID := uuid.New()
	orderID := uuid.New()
	orderNumber := "SO-001"

	tests := []struct {
		name            string
		event           shared.DomainEvent
		expectedDocType printing.DocType
		expectedTrigger printing.TriggerEvent
	}{
		{
			name: "SalesOrderCreated",
			event: &trade.SalesOrderCreatedEvent{
				BaseDomainEvent: shared.NewBaseDomainEvent(trade.EventTypeSalesOrderCreated, trade.AggregateTypeSalesOrder, orderID, tenantID),
				OrderID:         orderID,
				OrderNumber:     orderNumber,
			},
			expectedDocType: printing.DocTypeSalesOrder,
			expectedTrigger: printing.TriggerEventCreated,
		},
		{
			name: "SalesOrderConfirmed",
			event: &trade.SalesOrderConfirmedEvent{
				BaseDomainEvent: shared.NewBaseDomainEvent(trade.EventTypeSalesOrderConfirmed, trade.AggregateTypeSalesOrder, orderID, tenantID),
				OrderID:         orderID,
				OrderNumber:     orderNumber,
			},
			expectedDocType: printing.DocTypeSalesOrder,
			expectedTrigger: printing.TriggerEventConfirmed,
		},
		{
			name: "SalesOrderShipped",
			event: &trade.SalesOrderShippedEvent{
				BaseDomainEvent: shared.NewBaseDomainEvent(trade.EventTypeSalesOrderShipped, trade.AggregateTypeSalesOrder, orderID, tenantID),
				OrderID:         orderID,
				OrderNumber:     orderNumber,
			},
			expectedDocType: printing.DocTypeSalesOrder,
			expectedTrigger: printing.TriggerEventShipped,
		},
		{
			name: "SalesOrderCompleted",
			event: &trade.SalesOrderCompletedEvent{
				BaseDomainEvent: shared.NewBaseDomainEvent(trade.EventTypeSalesOrderCompleted, trade.AggregateTypeSalesOrder, orderID, tenantID),
				OrderID:         orderID,
				OrderNumber:     orderNumber,
			},
			expectedDocType: printing.DocTypeSalesOrder,
			expectedTrigger: printing.TriggerEventCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docType, trigger, docInfo := handler.mapEventToTrigger(tt.event)
			assert.Equal(t, tt.expectedDocType, docType)
			assert.Equal(t, tt.expectedTrigger, trigger)
			assert.Equal(t, orderID, docInfo.DocumentID)
			assert.Equal(t, orderNumber, docInfo.DocumentNumber)
		})
	}
}

func TestAutoPrintHandler_MapEventToTrigger_PurchaseOrder(t *testing.T) {
	handler := createTestHandler()
	tenantID := uuid.New()
	orderID := uuid.New()
	orderNumber := "PO-001"

	tests := []struct {
		name            string
		event           shared.DomainEvent
		expectedDocType printing.DocType
		expectedTrigger printing.TriggerEvent
	}{
		{
			name: "PurchaseOrderCreated",
			event: &trade.PurchaseOrderCreatedEvent{
				BaseDomainEvent: shared.NewBaseDomainEvent(trade.EventTypePurchaseOrderCreated, trade.AggregateTypePurchaseOrder, orderID, tenantID),
				OrderID:         orderID,
				OrderNumber:     orderNumber,
			},
			expectedDocType: printing.DocTypePurchaseOrder,
			expectedTrigger: printing.TriggerEventCreated,
		},
		{
			name: "PurchaseOrderConfirmed",
			event: &trade.PurchaseOrderConfirmedEvent{
				BaseDomainEvent: shared.NewBaseDomainEvent(trade.EventTypePurchaseOrderConfirmed, trade.AggregateTypePurchaseOrder, orderID, tenantID),
				OrderID:         orderID,
				OrderNumber:     orderNumber,
			},
			expectedDocType: printing.DocTypePurchaseOrder,
			expectedTrigger: printing.TriggerEventConfirmed,
		},
		{
			name: "PurchaseOrderReceived",
			event: &trade.PurchaseOrderReceivedEvent{
				BaseDomainEvent: shared.NewBaseDomainEvent(trade.EventTypePurchaseOrderReceived, trade.AggregateTypePurchaseOrder, orderID, tenantID),
				OrderID:         orderID,
				OrderNumber:     orderNumber,
			},
			expectedDocType: printing.DocTypePurchaseOrder,
			expectedTrigger: printing.TriggerEventReceived,
		},
		{
			name: "PurchaseOrderCompleted",
			event: &trade.PurchaseOrderCompletedEvent{
				BaseDomainEvent: shared.NewBaseDomainEvent(trade.EventTypePurchaseOrderCompleted, trade.AggregateTypePurchaseOrder, orderID, tenantID),
				OrderID:         orderID,
				OrderNumber:     orderNumber,
			},
			expectedDocType: printing.DocTypePurchaseOrder,
			expectedTrigger: printing.TriggerEventCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docType, trigger, docInfo := handler.mapEventToTrigger(tt.event)
			assert.Equal(t, tt.expectedDocType, docType)
			assert.Equal(t, tt.expectedTrigger, trigger)
			assert.Equal(t, orderID, docInfo.DocumentID)
			assert.Equal(t, orderNumber, docInfo.DocumentNumber)
		})
	}
}

func TestAutoPrintHandler_MapEventToTrigger_Returns(t *testing.T) {
	handler := createTestHandler()
	tenantID := uuid.New()
	returnID := uuid.New()
	returnNumber := "SR-001"

	tests := []struct {
		name            string
		event           shared.DomainEvent
		expectedDocType printing.DocType
		expectedTrigger printing.TriggerEvent
	}{
		{
			name: "SalesReturnCreated",
			event: &trade.SalesReturnCreatedEvent{
				BaseDomainEvent: shared.NewBaseDomainEvent(trade.EventTypeSalesReturnCreated, trade.AggregateTypeSalesReturn, returnID, tenantID),
				ReturnID:        returnID,
				ReturnNumber:    returnNumber,
			},
			expectedDocType: printing.DocTypeSalesReturn,
			expectedTrigger: printing.TriggerEventCreated,
		},
		{
			name: "SalesReturnApproved",
			event: &trade.SalesReturnApprovedEvent{
				BaseDomainEvent: shared.NewBaseDomainEvent(trade.EventTypeSalesReturnApproved, trade.AggregateTypeSalesReturn, returnID, tenantID),
				ReturnID:        returnID,
				ReturnNumber:    returnNumber,
			},
			expectedDocType: printing.DocTypeSalesReturn,
			expectedTrigger: printing.TriggerEventConfirmed, // Approved maps to Confirmed
		},
		{
			name: "SalesReturnCompleted",
			event: &trade.SalesReturnCompletedEvent{
				BaseDomainEvent: shared.NewBaseDomainEvent(trade.EventTypeSalesReturnCompleted, trade.AggregateTypeSalesReturn, returnID, tenantID),
				ReturnID:        returnID,
				ReturnNumber:    returnNumber,
			},
			expectedDocType: printing.DocTypeSalesReturn,
			expectedTrigger: printing.TriggerEventCompleted,
		},
		{
			name: "PurchaseReturnCreated",
			event: &trade.PurchaseReturnCreatedEvent{
				BaseDomainEvent: shared.NewBaseDomainEvent(trade.EventTypePurchaseReturnCreated, trade.AggregateTypePurchaseReturn, returnID, tenantID),
				ReturnID:        returnID,
				ReturnNumber:    "PR-001",
			},
			expectedDocType: printing.DocTypePurchaseReturn,
			expectedTrigger: printing.TriggerEventCreated,
		},
		{
			name: "PurchaseReturnApproved",
			event: &trade.PurchaseReturnApprovedEvent{
				BaseDomainEvent: shared.NewBaseDomainEvent(trade.EventTypePurchaseReturnApproved, trade.AggregateTypePurchaseReturn, returnID, tenantID),
				ReturnID:        returnID,
				ReturnNumber:    "PR-001",
			},
			expectedDocType: printing.DocTypePurchaseReturn,
			expectedTrigger: printing.TriggerEventConfirmed, // Approved maps to Confirmed
		},
		{
			name: "PurchaseReturnCompleted",
			event: &trade.PurchaseReturnCompletedEvent{
				BaseDomainEvent: shared.NewBaseDomainEvent(trade.EventTypePurchaseReturnCompleted, trade.AggregateTypePurchaseReturn, returnID, tenantID),
				ReturnID:        returnID,
				ReturnNumber:    "PR-001",
			},
			expectedDocType: printing.DocTypePurchaseReturn,
			expectedTrigger: printing.TriggerEventCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docType, trigger, docInfo := handler.mapEventToTrigger(tt.event)
			assert.Equal(t, tt.expectedDocType, docType)
			assert.Equal(t, tt.expectedTrigger, trigger)
			assert.Equal(t, returnID, docInfo.DocumentID)
		})
	}
}

func TestAutoPrintHandler_MapEventToTrigger_Vouchers(t *testing.T) {
	handler := createTestHandler()
	tenantID := uuid.New()
	voucherID := uuid.New()
	voucherNumber := "RV-001"

	tests := []struct {
		name            string
		event           shared.DomainEvent
		expectedDocType printing.DocType
		expectedTrigger printing.TriggerEvent
	}{
		{
			name: "ReceiptVoucherCreated",
			event: &finance.ReceiptVoucherCreatedEvent{
				BaseDomainEvent: shared.NewBaseDomainEvent("ReceiptVoucherCreated", "ReceiptVoucher", voucherID, tenantID),
				VoucherID:       voucherID,
				VoucherNumber:   voucherNumber,
			},
			expectedDocType: printing.DocTypeReceiptVoucher,
			expectedTrigger: printing.TriggerEventCreated,
		},
		{
			name: "ReceiptVoucherConfirmed",
			event: &finance.ReceiptVoucherConfirmedEvent{
				BaseDomainEvent: shared.NewBaseDomainEvent("ReceiptVoucherConfirmed", "ReceiptVoucher", voucherID, tenantID),
				VoucherID:       voucherID,
				VoucherNumber:   voucherNumber,
			},
			expectedDocType: printing.DocTypeReceiptVoucher,
			expectedTrigger: printing.TriggerEventConfirmed,
		},
		{
			name: "PaymentVoucherCreated",
			event: &finance.PaymentVoucherCreatedEvent{
				BaseDomainEvent: shared.NewBaseDomainEvent("PaymentVoucherCreated", "PaymentVoucher", voucherID, tenantID),
				VoucherID:       voucherID,
				VoucherNumber:   "PV-001",
			},
			expectedDocType: printing.DocTypePaymentVoucher,
			expectedTrigger: printing.TriggerEventCreated,
		},
		{
			name: "PaymentVoucherConfirmed",
			event: &finance.PaymentVoucherConfirmedEvent{
				BaseDomainEvent: shared.NewBaseDomainEvent("PaymentVoucherConfirmed", "PaymentVoucher", voucherID, tenantID),
				VoucherID:       voucherID,
				VoucherNumber:   "PV-001",
			},
			expectedDocType: printing.DocTypePaymentVoucher,
			expectedTrigger: printing.TriggerEventConfirmed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docType, trigger, docInfo := handler.mapEventToTrigger(tt.event)
			assert.Equal(t, tt.expectedDocType, docType)
			assert.Equal(t, tt.expectedTrigger, trigger)
			assert.Equal(t, voucherID, docInfo.DocumentID)
		})
	}
}

func TestAutoPrintHandler_MapEventToTrigger_UnknownEvent(t *testing.T) {
	handler := createTestHandler()

	// Create a mock unknown event
	unknownEvent := &shared.BaseDomainEvent{
		ID:            uuid.New(),
		Type:          "UnknownEvent",
		Timestamp:     time.Now(),
		AggID:         uuid.New(),
		AggType:       "Unknown",
		TenantIDValue: uuid.New(),
	}

	docType, trigger, docInfo := handler.mapEventToTrigger(unknownEvent)
	assert.Equal(t, printing.DocType(""), docType)
	assert.Equal(t, printing.TriggerEvent(""), trigger)
	assert.Equal(t, uuid.Nil, docInfo.DocumentID)
}

func TestAutoPrintHandler_BuildIdempotencyKey(t *testing.T) {
	handler := createTestHandler()
	docID := uuid.MustParse("12345678-1234-1234-1234-123456789012")

	key := handler.buildIdempotencyKey(printing.DocTypeSalesOrder, docID, printing.TriggerEventConfirmed)

	expected := "print:SALES_ORDER:12345678-1234-1234-1234-123456789012:CONFIRMED"
	assert.Equal(t, expected, key)
}

func TestAutoPrintHandler_CheckIdempotency_NoRedis(t *testing.T) {
	handler := createTestHandler()
	ctx := context.Background()

	// Without Redis, should always return true (new)
	isNew, err := handler.checkIdempotency(ctx, "test-key")
	assert.NoError(t, err)
	assert.True(t, isNew)
}

func TestAutoPrintHandler_CheckIdempotency_WithRedis(t *testing.T) {
	// Start miniredis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	// Create handler directly to bypass constructor validation for testing
	handler := &AutoPrintHandler{
		ruleRepo:     new(MockAutoPrintRuleRepository),
		printService: &PrintService{},
		redisClient:  redisClient,
		logger:       zap.NewNop(),
	}
	ctx := context.Background()

	// First call should return true (new)
	isNew, err := handler.checkIdempotency(ctx, "test-key")
	assert.NoError(t, err)
	assert.True(t, isNew)

	// Second call should return false (duplicate)
	isNew, err = handler.checkIdempotency(ctx, "test-key")
	assert.NoError(t, err)
	assert.False(t, isNew)

	// Different key should return true
	isNew, err = handler.checkIdempotency(ctx, "different-key")
	assert.NoError(t, err)
	assert.True(t, isNew)
}

func TestAutoPrintHandler_Handle_NoRuleFound(t *testing.T) {
	mockRepo := new(MockAutoPrintRuleRepository)
	// Create handler directly to bypass constructor validation for testing
	handler := &AutoPrintHandler{
		ruleRepo:     mockRepo,
		printService: &PrintService{},
		redisClient:  nil,
		logger:       zap.NewNop(),
	}
	ctx := context.Background()

	tenantID := uuid.New()
	orderID := uuid.New()
	event := &trade.SalesOrderConfirmedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(trade.EventTypeSalesOrderConfirmed, trade.AggregateTypeSalesOrder, orderID, tenantID),
		OrderID:         orderID,
		OrderNumber:     "SO-001",
	}

	// Mock: no rule found
	mockRepo.On("FindEnabledByDocTypeAndEvent", ctx, tenantID, printing.DocTypeSalesOrder, printing.TriggerEventConfirmed).
		Return(nil, shared.ErrNotFound)

	err := handler.Handle(ctx, event)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAutoPrintHandler_Handle_RuleDisabled(t *testing.T) {
	mockRepo := new(MockAutoPrintRuleRepository)
	// Create handler directly to bypass constructor validation for testing
	handler := &AutoPrintHandler{
		ruleRepo:     mockRepo,
		printService: &PrintService{},
		redisClient:  nil,
		logger:       zap.NewNop(),
	}
	ctx := context.Background()

	tenantID := uuid.New()
	orderID := uuid.New()
	event := &trade.SalesOrderConfirmedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(trade.EventTypeSalesOrderConfirmed, trade.AggregateTypeSalesOrder, orderID, tenantID),
		OrderID:         orderID,
		OrderNumber:     "SO-001",
	}

	// Create rule with AutoPrint = false
	rule := createTestAutoPrintRule(tenantID, printing.DocTypeSalesOrder, printing.TriggerEventConfirmed, false)

	mockRepo.On("FindEnabledByDocTypeAndEvent", ctx, tenantID, printing.DocTypeSalesOrder, printing.TriggerEventConfirmed).
		Return(rule, nil)

	err := handler.Handle(ctx, event)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAutoPrintHandler_Handle_DuplicateEvent(t *testing.T) {
	// Start miniredis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	mockRepo := new(MockAutoPrintRuleRepository)
	// Create handler directly to bypass constructor validation for testing
	// Note: PrintService is nil, but we use AutoPrint=false so createPrintJob is never called
	handler := &AutoPrintHandler{
		ruleRepo:     mockRepo,
		printService: &PrintService{},
		redisClient:  redisClient,
		logger:       zap.NewNop(),
	}
	ctx := context.Background()

	tenantID := uuid.New()
	orderID := uuid.New()
	event := &trade.SalesOrderConfirmedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(trade.EventTypeSalesOrderConfirmed, trade.AggregateTypeSalesOrder, orderID, tenantID),
		OrderID:         orderID,
		OrderNumber:     "SO-001",
	}

	// Create rule with AutoPrint = false
	// With the new flow (idempotency check after rule lookup), when AutoPrint=false,
	// the idempotency key is NOT set because we return early before the check.
	// This test verifies that behavior - both calls query the repo because
	// no idempotency key is set when AutoPrint=false.
	rule := createTestAutoPrintRule(tenantID, printing.DocTypeSalesOrder, printing.TriggerEventConfirmed, false)

	// Both calls will query for rule since AutoPrint=false means no idempotency key is set
	mockRepo.On("FindEnabledByDocTypeAndEvent", ctx, tenantID, printing.DocTypeSalesOrder, printing.TriggerEventConfirmed).
		Return(rule, nil)

	// First call - rule found but AutoPrint=false, returns early (no idempotency key set)
	err = handler.Handle(ctx, event)
	assert.NoError(t, err)

	// Second call - same behavior, no idempotency blocking because key was never set
	err = handler.Handle(ctx, event)
	assert.NoError(t, err)

	// Both calls query the repo because AutoPrint=false means idempotency check is skipped
	mockRepo.AssertNumberOfCalls(t, "FindEnabledByDocTypeAndEvent", 2)

	// Verify no idempotency key was set in Redis
	key := handler.buildIdempotencyKey(printing.DocTypeSalesOrder, orderID, printing.TriggerEventConfirmed)
	exists, err := redisClient.Exists(ctx, key).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), exists, "idempotency key should not be set when AutoPrint=false")
}

func TestAutoPrintHandler_Handle_RepositoryError(t *testing.T) {
	mockRepo := new(MockAutoPrintRuleRepository)
	// Create handler directly to bypass constructor validation for testing
	handler := &AutoPrintHandler{
		ruleRepo:     mockRepo,
		printService: &PrintService{},
		redisClient:  nil,
		logger:       zap.NewNop(),
	}
	ctx := context.Background()

	tenantID := uuid.New()
	orderID := uuid.New()
	event := &trade.SalesOrderConfirmedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(trade.EventTypeSalesOrderConfirmed, trade.AggregateTypeSalesOrder, orderID, tenantID),
		OrderID:         orderID,
		OrderNumber:     "SO-001",
	}

	// Mock: repository error
	mockRepo.On("FindEnabledByDocTypeAndEvent", ctx, tenantID, printing.DocTypeSalesOrder, printing.TriggerEventConfirmed).
		Return(nil, errors.New("database error"))

	// Should not return error - auto-print failure should not block business flow
	err := handler.Handle(ctx, event)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAutoPrintHandler_Handle_UnknownEvent(t *testing.T) {
	handler := createTestHandler()
	ctx := context.Background()

	// Create an unknown event
	unknownEvent := &shared.BaseDomainEvent{
		ID:            uuid.New(),
		Type:          "UnknownEvent",
		Timestamp:     time.Now(),
		AggID:         uuid.New(),
		AggType:       "Unknown",
		TenantIDValue: uuid.New(),
	}

	// Should return nil without error
	err := handler.Handle(ctx, unknownEvent)
	assert.NoError(t, err)
}

func TestAutoPrintHandler_RegisterWithEventBus(t *testing.T) {
	handler := createTestHandler()

	// Create a mock event bus
	mockBus := new(MockEventBus)
	mockBus.On("Subscribe", handler).Return()

	handler.RegisterWithEventBus(mockBus)

	mockBus.AssertExpectations(t)
}

func TestAutoPrintHandler_Handle_IdempotencyWithAutoPrintEnabled(t *testing.T) {
	// This test verifies that idempotency works correctly when AutoPrint=true
	// by checking that the idempotency key is set in Redis after the first call

	// Start miniredis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	mockRepo := new(MockAutoPrintRuleRepository)
	// Create handler with nil PrintService - we'll verify behavior before createPrintJob is called
	handler := &AutoPrintHandler{
		ruleRepo:     mockRepo,
		printService: nil, // Will cause panic if createPrintJob is called
		redisClient:  redisClient,
		logger:       zap.NewNop(),
	}
	ctx := context.Background()

	tenantID := uuid.New()
	orderID := uuid.New()
	event := &trade.SalesOrderConfirmedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(trade.EventTypeSalesOrderConfirmed, trade.AggregateTypeSalesOrder, orderID, tenantID),
		OrderID:         orderID,
		OrderNumber:     "SO-001",
	}

	// Create rule with AutoPrint = true
	rule := createTestAutoPrintRule(tenantID, printing.DocTypeSalesOrder, printing.TriggerEventConfirmed, true)

	mockRepo.On("FindEnabledByDocTypeAndEvent", ctx, tenantID, printing.DocTypeSalesOrder, printing.TriggerEventConfirmed).
		Return(rule, nil)

	// First call will panic when trying to create print job (because printService is nil)
	// This is expected - we're testing that the idempotency key gets set
	assert.Panics(t, func() {
		_ = handler.Handle(ctx, event)
	}, "first call should panic when trying to create print job with nil PrintService")

	// Verify idempotency key was set in Redis before the panic
	key := handler.buildIdempotencyKey(printing.DocTypeSalesOrder, orderID, printing.TriggerEventConfirmed)
	exists, err := redisClient.Exists(ctx, key).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), exists, "idempotency key should be set even though createPrintJob panicked")

	// Now create a proper handler with a working PrintService mock
	// The second call should be blocked by idempotency
	handler2 := &AutoPrintHandler{
		ruleRepo:     mockRepo,
		printService: &PrintService{}, // Still not fully initialized, but won't be called
		redisClient:  redisClient,
		logger:       zap.NewNop(),
	}

	// Second call - should be blocked by idempotency check (won't call createPrintJob)
	err = handler2.Handle(ctx, event)
	assert.NoError(t, err)

	// Verify repo was called twice (once per handler)
	mockRepo.AssertNumberOfCalls(t, "FindEnabledByDocTypeAndEvent", 2)
}

// MockEventBus is a mock implementation of EventSubscriber
type MockEventBus struct {
	mock.Mock
}

func (m *MockEventBus) Subscribe(handler shared.EventHandler, eventTypes ...string) {
	m.Called(handler)
}

func (m *MockEventBus) Unsubscribe(handler shared.EventHandler) {
	m.Called(handler)
}
