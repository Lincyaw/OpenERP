package finance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/finance"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// =============================================================================
// Mock Gateway for Payment Callback Tests
// =============================================================================

type MockPaymentGateway struct {
	mock.Mock
}

func (m *MockPaymentGateway) GatewayType() finance.PaymentGatewayType {
	args := m.Called()
	return args.Get(0).(finance.PaymentGatewayType)
}

func (m *MockPaymentGateway) CreatePayment(ctx context.Context, req *finance.CreatePaymentRequest) (*finance.CreatePaymentResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.CreatePaymentResponse), args.Error(1)
}

func (m *MockPaymentGateway) QueryPayment(ctx context.Context, req *finance.QueryPaymentRequest) (*finance.QueryPaymentResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.QueryPaymentResponse), args.Error(1)
}

func (m *MockPaymentGateway) ClosePayment(ctx context.Context, req *finance.ClosePaymentRequest) (*finance.ClosePaymentResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.ClosePaymentResponse), args.Error(1)
}

func (m *MockPaymentGateway) CreateRefund(ctx context.Context, req *finance.RefundRequest) (*finance.RefundResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.RefundResponse), args.Error(1)
}

func (m *MockPaymentGateway) QueryRefund(ctx context.Context, tenantID uuid.UUID, gatewayRefundID string) (*finance.RefundResponse, error) {
	args := m.Called(ctx, tenantID, gatewayRefundID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.RefundResponse), args.Error(1)
}

func (m *MockPaymentGateway) VerifyCallback(ctx context.Context, payload []byte, signature string) (*finance.PaymentCallback, error) {
	args := m.Called(ctx, payload, signature)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.PaymentCallback), args.Error(1)
}

func (m *MockPaymentGateway) VerifyRefundCallback(ctx context.Context, payload []byte, signature string) (*finance.RefundCallback, error) {
	args := m.Called(ctx, payload, signature)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.RefundCallback), args.Error(1)
}

func (m *MockPaymentGateway) GenerateCallbackResponse(success bool, message string) []byte {
	args := m.Called(success, message)
	return args.Get(0).([]byte)
}

// =============================================================================
// Mock Receipt Voucher Repository
// =============================================================================

type MockReceiptVoucherRepository struct {
	mock.Mock
}

func (m *MockReceiptVoucherRepository) FindByID(ctx context.Context, id uuid.UUID) (*finance.ReceiptVoucher, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.ReceiptVoucher), args.Error(1)
}

func (m *MockReceiptVoucherRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*finance.ReceiptVoucher, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.ReceiptVoucher), args.Error(1)
}

func (m *MockReceiptVoucherRepository) FindByVoucherNumber(ctx context.Context, tenantID uuid.UUID, voucherNumber string) (*finance.ReceiptVoucher, error) {
	args := m.Called(ctx, tenantID, voucherNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.ReceiptVoucher), args.Error(1)
}

func (m *MockReceiptVoucherRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter finance.ReceiptVoucherFilter) ([]finance.ReceiptVoucher, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]finance.ReceiptVoucher), args.Error(1)
}

func (m *MockReceiptVoucherRepository) FindByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, filter finance.ReceiptVoucherFilter) ([]finance.ReceiptVoucher, error) {
	args := m.Called(ctx, tenantID, customerID, filter)
	return args.Get(0).([]finance.ReceiptVoucher), args.Error(1)
}

func (m *MockReceiptVoucherRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status finance.VoucherStatus, filter finance.ReceiptVoucherFilter) ([]finance.ReceiptVoucher, error) {
	args := m.Called(ctx, tenantID, status, filter)
	return args.Get(0).([]finance.ReceiptVoucher), args.Error(1)
}

func (m *MockReceiptVoucherRepository) FindWithUnallocatedAmount(ctx context.Context, tenantID, customerID uuid.UUID) ([]finance.ReceiptVoucher, error) {
	args := m.Called(ctx, tenantID, customerID)
	return args.Get(0).([]finance.ReceiptVoucher), args.Error(1)
}

func (m *MockReceiptVoucherRepository) FindByPaymentReference(ctx context.Context, paymentReference string) (*finance.ReceiptVoucher, error) {
	args := m.Called(ctx, paymentReference)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.ReceiptVoucher), args.Error(1)
}

func (m *MockReceiptVoucherRepository) Save(ctx context.Context, voucher *finance.ReceiptVoucher) error {
	args := m.Called(ctx, voucher)
	return args.Error(0)
}

func (m *MockReceiptVoucherRepository) SaveWithLock(ctx context.Context, voucher *finance.ReceiptVoucher) error {
	args := m.Called(ctx, voucher)
	return args.Error(0)
}

func (m *MockReceiptVoucherRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockReceiptVoucherRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *MockReceiptVoucherRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter finance.ReceiptVoucherFilter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockReceiptVoucherRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status finance.VoucherStatus) (int64, error) {
	args := m.Called(ctx, tenantID, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockReceiptVoucherRepository) CountByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID, customerID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockReceiptVoucherRepository) SumByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (decimal.Decimal, error) {
	args := m.Called(ctx, tenantID, customerID)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockReceiptVoucherRepository) SumForTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockReceiptVoucherRepository) SumUnallocatedByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (decimal.Decimal, error) {
	args := m.Called(ctx, tenantID, customerID)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockReceiptVoucherRepository) ExistsByVoucherNumber(ctx context.Context, tenantID uuid.UUID, voucherNumber string) (bool, error) {
	args := m.Called(ctx, tenantID, voucherNumber)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockReceiptVoucherRepository) GenerateVoucherNumber(ctx context.Context, tenantID uuid.UUID) (string, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).(string), args.Error(1)
}

// =============================================================================
// Mock Event Publisher
// =============================================================================

type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) Publish(ctx context.Context, events ...shared.DomainEvent) error {
	args := m.Called(ctx, events)
	return args.Error(0)
}

// =============================================================================
// Test Cases
// =============================================================================

func TestPaymentCallbackService_GetGateway(t *testing.T) {
	tests := []struct {
		name        string
		gatewayType finance.PaymentGatewayType
		registered  bool
		wantErr     bool
	}{
		{
			name:        "registered gateway",
			gatewayType: finance.PaymentGatewayTypeWechat,
			registered:  true,
			wantErr:     false,
		},
		{
			name:        "unregistered gateway",
			gatewayType: finance.PaymentGatewayTypeAlipay,
			registered:  false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create service with or without gateway
			gateways := []finance.PaymentGateway{}
			if tt.registered {
				mockGateway := &MockPaymentGateway{}
				mockGateway.On("GatewayType").Return(tt.gatewayType)
				gateways = append(gateways, mockGateway)
			}

			svc := NewPaymentCallbackService(PaymentCallbackServiceConfig{
				Gateways: gateways,
			})

			// Get gateway
			gateway, err := svc.GetGateway(tt.gatewayType)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, ErrCallbackGatewayNotRegistered, err)
				assert.Nil(t, gateway)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, gateway)
			}
		})
	}
}

func TestPaymentCallbackService_RegisterGateway(t *testing.T) {
	svc := NewPaymentCallbackService(PaymentCallbackServiceConfig{})

	mockGateway := &MockPaymentGateway{}
	mockGateway.On("GatewayType").Return(finance.PaymentGatewayTypeWechat)

	// Register gateway
	svc.RegisterGateway(mockGateway)

	// Verify it's registered
	gateway, err := svc.GetGateway(finance.PaymentGatewayTypeWechat)
	assert.NoError(t, err)
	assert.NotNil(t, gateway)
}

func TestPaymentCallbackService_ProcessPaymentCallback_VerificationFailed(t *testing.T) {
	// Setup
	mockGateway := &MockPaymentGateway{}
	mockGateway.On("GatewayType").Return(finance.PaymentGatewayTypeWechat)
	mockGateway.On("VerifyCallback", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("invalid signature"))

	svc := NewPaymentCallbackService(PaymentCallbackServiceConfig{
		Gateways: []finance.PaymentGateway{mockGateway},
	})

	// Execute
	result, err := svc.ProcessPaymentCallback(
		context.Background(),
		finance.PaymentGatewayTypeWechat,
		[]byte(`{"test": "payload"}`),
		"invalid_signature",
	)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signature verification failed")
	assert.Nil(t, result)
}

func TestPaymentCallbackService_ProcessPaymentCallback_GatewayNotRegistered(t *testing.T) {
	svc := NewPaymentCallbackService(PaymentCallbackServiceConfig{})

	// Execute
	result, err := svc.ProcessPaymentCallback(
		context.Background(),
		finance.PaymentGatewayTypeWechat,
		[]byte(`{"test": "payload"}`),
		"signature",
	)

	// Verify
	assert.Error(t, err)
	assert.Equal(t, ErrCallbackGatewayNotRegistered, err)
	assert.Nil(t, result)
}

func TestPaymentCallbackService_HandlePaymentCallback_NonSuccessStatus(t *testing.T) {
	// Setup
	mockVoucherRepo := &MockReceiptVoucherRepository{}
	svc := NewPaymentCallbackService(PaymentCallbackServiceConfig{
		ReceiptVoucherRepo: mockVoucherRepo,
	})

	callback := &finance.PaymentCallback{
		GatewayType:    finance.PaymentGatewayTypeWechat,
		GatewayOrderID: "test-order-123",
		OrderNumber:    "test-order-123",
		Status:         finance.GatewayPaymentStatusPending, // Not success
		Amount:         decimal.NewFromFloat(100.00),
	}

	// Execute
	err := svc.HandlePaymentCallback(context.Background(), callback)

	// Verify - should succeed but not process (non-successful payment)
	assert.NoError(t, err)
	mockVoucherRepo.AssertNotCalled(t, "FindByPaymentReference")
}

func TestPaymentCallbackService_HandlePaymentCallback_VoucherNotFound(t *testing.T) {
	// Setup
	mockVoucherRepo := &MockReceiptVoucherRepository{}
	mockVoucherRepo.On("FindByPaymentReference", mock.Anything, "test-order-123").
		Return(nil, nil)

	svc := NewPaymentCallbackService(PaymentCallbackServiceConfig{
		ReceiptVoucherRepo: mockVoucherRepo,
	})

	callback := &finance.PaymentCallback{
		GatewayType:    finance.PaymentGatewayTypeWechat,
		GatewayOrderID: "test-order-123",
		OrderNumber:    "test-order-123",
		Status:         finance.GatewayPaymentStatusPaid,
		Amount:         decimal.NewFromFloat(100.00),
	}

	// Execute
	err := svc.HandlePaymentCallback(context.Background(), callback)

	// Verify
	assert.Error(t, err)
	assert.Equal(t, ErrCallbackOrderNotFound, err)
}

func TestPaymentCallbackService_HandlePaymentCallback_AlreadyConfirmed(t *testing.T) {
	// Setup
	tenantID := uuid.New()
	customerID := uuid.New()
	voucherID := uuid.New()

	voucher := &finance.ReceiptVoucher{
		Status: finance.VoucherStatusConfirmed, // Already confirmed
	}
	voucher.ID = voucherID
	voucher.TenantID = tenantID
	voucher.CustomerID = customerID

	mockVoucherRepo := &MockReceiptVoucherRepository{}
	mockVoucherRepo.On("FindByPaymentReference", mock.Anything, "test-order-123").
		Return(voucher, nil)

	svc := NewPaymentCallbackService(PaymentCallbackServiceConfig{
		ReceiptVoucherRepo: mockVoucherRepo,
	})

	callback := &finance.PaymentCallback{
		GatewayType:          finance.PaymentGatewayTypeWechat,
		GatewayOrderID:       "test-order-123",
		GatewayTransactionID: "txn-456",
		OrderNumber:          "test-order-123",
		Status:               finance.GatewayPaymentStatusPaid,
		Amount:               decimal.NewFromFloat(100.00),
	}

	// Execute
	err := svc.HandlePaymentCallback(context.Background(), callback)

	// Verify - should succeed (idempotent)
	assert.NoError(t, err)
	mockVoucherRepo.AssertNotCalled(t, "SaveWithLock")
}

func TestPaymentCallbackService_HandleRefundCallback_NonSuccessStatus(t *testing.T) {
	// Setup
	svc := NewPaymentCallbackService(PaymentCallbackServiceConfig{})

	callback := &finance.RefundCallback{
		GatewayType:     finance.PaymentGatewayTypeWechat,
		GatewayRefundID: "refund-123",
		RefundNumber:    "REF-001",
		Status:          finance.RefundStatusPending, // Not success
		RefundAmount:    decimal.NewFromFloat(50.00),
	}

	// Execute
	err := svc.HandleRefundCallback(context.Background(), callback)

	// Verify - should succeed but not process
	assert.NoError(t, err)
}

func TestPaymentCallbackService_HandleRefundCallback_Success(t *testing.T) {
	// Setup
	mockPublisher := &MockEventPublisher{}
	mockPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil)

	svc := NewPaymentCallbackService(PaymentCallbackServiceConfig{
		EventPublisher: mockPublisher,
	})

	now := time.Now()
	callback := &finance.RefundCallback{
		GatewayType:     finance.PaymentGatewayTypeWechat,
		GatewayRefundID: "refund-123",
		RefundNumber:    "REF-001",
		Status:          finance.RefundStatusSuccess,
		RefundAmount:    decimal.NewFromFloat(50.00),
		RefundedAt:      &now,
	}

	// Execute
	err := svc.HandleRefundCallback(context.Background(), callback)

	// Verify
	assert.NoError(t, err)
	mockPublisher.AssertCalled(t, "Publish", mock.Anything, mock.Anything)
}

func TestPaymentCallbackService_Idempotency(t *testing.T) {
	// Setup
	mockGateway := &MockPaymentGateway{}
	mockGateway.On("GatewayType").Return(finance.PaymentGatewayTypeWechat)

	callback := &finance.PaymentCallback{
		GatewayType:          finance.PaymentGatewayTypeWechat,
		GatewayOrderID:       "test-order-123",
		GatewayTransactionID: "unique-txn-id",
		OrderNumber:          "test-order-123",
		Status:               finance.GatewayPaymentStatusPaid,
		Amount:               decimal.NewFromFloat(100.00),
	}

	mockGateway.On("VerifyCallback", mock.Anything, mock.Anything, mock.Anything).
		Return(callback, nil)
	mockGateway.On("GenerateCallbackResponse", true, "").
		Return([]byte(`{"code":"SUCCESS"}`))
	mockGateway.On("GenerateCallbackResponse", false, mock.Anything).
		Return([]byte(`{"code":"FAIL"}`))

	// Create a voucher that will be confirmed (already confirmed to avoid confirmation logic issues)
	voucher := createTestReceiptVoucher()
	voucher.Status = finance.VoucherStatusConfirmed // Already confirmed - idempotent case
	mockVoucherRepo := &MockReceiptVoucherRepository{}
	mockVoucherRepo.On("FindByPaymentReference", mock.Anything, "test-order-123").
		Return(voucher, nil)

	svc := NewPaymentCallbackService(PaymentCallbackServiceConfig{
		Gateways:           []finance.PaymentGateway{mockGateway},
		ReceiptVoucherRepo: mockVoucherRepo,
	})

	// Process first callback
	result1, err1 := svc.ProcessPaymentCallback(
		context.Background(),
		finance.PaymentGatewayTypeWechat,
		[]byte(`{"test": "payload"}`),
		"valid_signature",
	)

	assert.NoError(t, err1)
	assert.True(t, result1.Success)
	assert.False(t, result1.AlreadyProcessed)

	// Process duplicate callback with same transaction ID
	result2, err2 := svc.ProcessPaymentCallback(
		context.Background(),
		finance.PaymentGatewayTypeWechat,
		[]byte(`{"test": "payload"}`),
		"valid_signature",
	)

	assert.NoError(t, err2)
	assert.True(t, result2.Success)
	assert.True(t, result2.AlreadyProcessed) // Should be marked as already processed
}

// Helper function to create test receipt voucher
func createTestReceiptVoucher() *finance.ReceiptVoucher {
	voucher := &finance.ReceiptVoucher{
		VoucherNumber:     "RV-TEST-001",
		CustomerID:        uuid.New(),
		CustomerName:      "Test Customer",
		Amount:            decimal.NewFromFloat(100.00),
		AllocatedAmount:   decimal.Zero,
		UnallocatedAmount: decimal.NewFromFloat(100.00),
		PaymentMethod:     finance.PaymentMethodWechat,
		PaymentReference:  "test-order-123",
		Status:            finance.VoucherStatusDraft,
		ReceiptDate:       time.Now(),
	}
	voucher.ID = uuid.New()
	voucher.TenantID = uuid.New()
	voucher.Version = 1
	return voucher
}
