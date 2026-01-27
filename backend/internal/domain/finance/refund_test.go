package finance_test

import (
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/finance"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRefundRecord(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()
	sourceID := uuid.New()
	orderID := uuid.New()

	tests := []struct {
		name            string
		tenantID        uuid.UUID
		refundNumber    string
		orderID         uuid.UUID
		orderNumber     string
		sourceType      finance.RefundSourceType
		sourceID        uuid.UUID
		sourceNumber    string
		customerID      uuid.UUID
		customerName    string
		refundAmount    decimal.Decimal
		gatewayType     finance.PaymentGatewayType
		reason          string
		expectedErr     bool
		expectedErrCode string
	}{
		{
			name:         "valid refund record",
			tenantID:     tenantID,
			refundNumber: "RF-2026-01-0001",
			orderID:      orderID,
			orderNumber:  "SO-2026-0001",
			sourceType:   finance.RefundSourceTypeSalesReturn,
			sourceID:     sourceID,
			sourceNumber: "SR-2026-0001",
			customerID:   customerID,
			customerName: "Test Customer",
			refundAmount: decimal.NewFromFloat(100.00),
			gatewayType:  finance.PaymentGatewayTypeWechat,
			reason:       "Customer return",
			expectedErr:  false,
		},
		{
			name:            "empty refund number",
			tenantID:        tenantID,
			refundNumber:    "",
			orderID:         orderID,
			orderNumber:     "SO-2026-0001",
			sourceType:      finance.RefundSourceTypeSalesReturn,
			sourceID:        sourceID,
			sourceNumber:    "SR-2026-0001",
			customerID:      customerID,
			customerName:    "Test Customer",
			refundAmount:    decimal.NewFromFloat(100.00),
			gatewayType:     finance.PaymentGatewayTypeWechat,
			reason:          "Customer return",
			expectedErr:     true,
			expectedErrCode: "Refund number cannot be empty",
		},
		{
			name:            "nil order ID",
			tenantID:        tenantID,
			refundNumber:    "RF-2026-01-0001",
			orderID:         uuid.Nil,
			orderNumber:     "SO-2026-0001",
			sourceType:      finance.RefundSourceTypeSalesReturn,
			sourceID:        sourceID,
			sourceNumber:    "SR-2026-0001",
			customerID:      customerID,
			customerName:    "Test Customer",
			refundAmount:    decimal.NewFromFloat(100.00),
			gatewayType:     finance.PaymentGatewayTypeWechat,
			reason:          "Customer return",
			expectedErr:     true,
			expectedErrCode: "Original order ID cannot be empty",
		},
		{
			name:            "nil customer ID",
			tenantID:        tenantID,
			refundNumber:    "RF-2026-01-0001",
			orderID:         orderID,
			orderNumber:     "SO-2026-0001",
			sourceType:      finance.RefundSourceTypeSalesReturn,
			sourceID:        sourceID,
			sourceNumber:    "SR-2026-0001",
			customerID:      uuid.Nil,
			customerName:    "Test Customer",
			refundAmount:    decimal.NewFromFloat(100.00),
			gatewayType:     finance.PaymentGatewayTypeWechat,
			reason:          "Customer return",
			expectedErr:     true,
			expectedErrCode: "Customer ID cannot be empty",
		},
		{
			name:            "zero refund amount",
			tenantID:        tenantID,
			refundNumber:    "RF-2026-01-0001",
			orderID:         orderID,
			orderNumber:     "SO-2026-0001",
			sourceType:      finance.RefundSourceTypeSalesReturn,
			sourceID:        sourceID,
			sourceNumber:    "SR-2026-0001",
			customerID:      customerID,
			customerName:    "Test Customer",
			refundAmount:    decimal.Zero,
			gatewayType:     finance.PaymentGatewayTypeWechat,
			reason:          "Customer return",
			expectedErr:     true,
			expectedErrCode: "Refund amount must be positive",
		},
		{
			name:            "negative refund amount",
			tenantID:        tenantID,
			refundNumber:    "RF-2026-01-0001",
			orderID:         orderID,
			orderNumber:     "SO-2026-0001",
			sourceType:      finance.RefundSourceTypeSalesReturn,
			sourceID:        sourceID,
			sourceNumber:    "SR-2026-0001",
			customerID:      customerID,
			customerName:    "Test Customer",
			refundAmount:    decimal.NewFromFloat(-100.00),
			gatewayType:     finance.PaymentGatewayTypeWechat,
			reason:          "Customer return",
			expectedErr:     true,
			expectedErrCode: "Refund amount must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, err := finance.NewRefundRecord(
				tt.tenantID,
				tt.refundNumber,
				tt.orderID,
				tt.orderNumber,
				tt.sourceType,
				tt.sourceID,
				tt.sourceNumber,
				tt.customerID,
				tt.customerName,
				tt.refundAmount,
				tt.gatewayType,
				tt.reason,
			)

			if tt.expectedErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrCode)
				assert.Nil(t, record)
			} else {
				require.NoError(t, err)
				require.NotNil(t, record)
				assert.Equal(t, tt.refundNumber, record.RefundNumber)
				assert.Equal(t, tt.orderID, record.OriginalOrderID)
				assert.Equal(t, tt.sourceType, record.SourceType)
				assert.Equal(t, tt.customerID, record.CustomerID)
				assert.True(t, tt.refundAmount.Equal(record.RefundAmount))
				assert.Equal(t, tt.gatewayType, record.GatewayType)
				assert.Equal(t, finance.RefundRecordStatusPending, record.Status)
				assert.NotNil(t, record.RequestedAt)
				assert.True(t, record.ActualRefundAmount.IsZero())
				assert.NotEmpty(t, record.ID)
			}
		})
	}
}

func TestRefundRecord_Complete(t *testing.T) {
	record := createTestRefundRecord(t)

	actualAmount := decimal.NewFromFloat(100.00)
	rawResponse := `{"status": "success"}`

	err := record.Complete(actualAmount, rawResponse)
	require.NoError(t, err)

	assert.Equal(t, finance.RefundRecordStatusSuccess, record.Status)
	assert.True(t, actualAmount.Equal(record.ActualRefundAmount))
	assert.Equal(t, rawResponse, record.RawResponse)
	assert.NotNil(t, record.CompletedAt)
}

func TestRefundRecord_Complete_Idempotent(t *testing.T) {
	record := createTestRefundRecord(t)

	// Complete first time
	err := record.Complete(decimal.NewFromFloat(100.00), `{}`)
	require.NoError(t, err)
	firstCompletedAt := record.CompletedAt

	// Complete again should be idempotent
	err = record.Complete(decimal.NewFromFloat(100.00), `{}`)
	require.NoError(t, err)

	assert.Equal(t, finance.RefundRecordStatusSuccess, record.Status)
	assert.Equal(t, firstCompletedAt, record.CompletedAt)
}

func TestRefundRecord_Complete_FromFailedState(t *testing.T) {
	record := createTestRefundRecord(t)

	// First fail it
	err := record.Fail("Gateway error", `{"error": "timeout"}`)
	require.NoError(t, err)

	// Cannot complete a failed refund
	err = record.Complete(decimal.NewFromFloat(100.00), `{}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Cannot complete refund")
}

func TestRefundRecord_Fail(t *testing.T) {
	record := createTestRefundRecord(t)

	failReason := "Gateway timeout"
	rawResponse := `{"error": "timeout"}`

	err := record.Fail(failReason, rawResponse)
	require.NoError(t, err)

	assert.Equal(t, finance.RefundRecordStatusFailed, record.Status)
	assert.Equal(t, failReason, record.FailReason)
	assert.Equal(t, rawResponse, record.RawResponse)
	assert.NotNil(t, record.FailedAt)
}

func TestRefundRecord_Fail_FromCompletedState(t *testing.T) {
	record := createTestRefundRecord(t)

	// First complete it
	err := record.Complete(decimal.NewFromFloat(100.00), `{}`)
	require.NoError(t, err)

	// Cannot fail a completed refund
	err = record.Fail("Some error", `{}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Cannot fail refund")
}

func TestRefundRecord_MarkProcessing(t *testing.T) {
	record := createTestRefundRecord(t)

	gatewayRefundID := "wx_refund_123456"
	err := record.MarkProcessing(gatewayRefundID)
	require.NoError(t, err)

	assert.Equal(t, finance.RefundRecordStatusProcessing, record.Status)
	assert.Equal(t, gatewayRefundID, record.GatewayRefundID)
}

func TestRefundRecord_Close(t *testing.T) {
	record := createTestRefundRecord(t)

	reason := "Cancelled by user"
	err := record.Close(reason)
	require.NoError(t, err)

	assert.Equal(t, finance.RefundRecordStatusClosed, record.Status)
	assert.Equal(t, reason, record.FailReason)
}

func TestRefundRecord_Close_FromCompletedState(t *testing.T) {
	record := createTestRefundRecord(t)

	// First complete it
	err := record.Complete(decimal.NewFromFloat(100.00), `{}`)
	require.NoError(t, err)

	// Cannot close a completed refund
	err = record.Close("Cancelled")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Cannot close refund")
}

func TestRefundRecord_SetOriginalPayment(t *testing.T) {
	record := createTestRefundRecord(t)

	paymentID := uuid.New()
	err := record.SetOriginalPayment(paymentID)
	require.NoError(t, err)

	assert.NotNil(t, record.OriginalPaymentID)
	assert.Equal(t, paymentID, *record.OriginalPaymentID)
}

func TestRefundRecord_SetOriginalPayment_NilID(t *testing.T) {
	record := createTestRefundRecord(t)

	err := record.SetOriginalPayment(uuid.Nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Payment ID cannot be empty")
}

func TestRefundRecord_SetGatewayInfo(t *testing.T) {
	record := createTestRefundRecord(t)

	gatewayRefundID := "wx_refund_123"
	gatewayOrderID := "wx_order_456"
	gatewayTransactionID := "wx_txn_789"

	record.SetGatewayInfo(gatewayRefundID, gatewayOrderID, gatewayTransactionID)

	assert.Equal(t, gatewayRefundID, record.GatewayRefundID)
	assert.Equal(t, gatewayOrderID, record.GatewayOrderID)
	assert.Equal(t, gatewayTransactionID, record.GatewayTransactionID)
}

func TestRefundRecord_StatusHelpers(t *testing.T) {
	record := createTestRefundRecord(t)

	// Initial state is pending
	assert.True(t, record.IsPending())
	assert.False(t, record.IsProcessing())
	assert.False(t, record.IsSuccess())
	assert.False(t, record.IsFailed())
	assert.False(t, record.IsClosed())

	// Mark processing
	err := record.MarkProcessing("gw_123")
	require.NoError(t, err)
	assert.False(t, record.IsPending())
	assert.True(t, record.IsProcessing())

	// Complete
	err = record.Complete(decimal.NewFromFloat(100), `{}`)
	require.NoError(t, err)
	assert.True(t, record.IsSuccess())
}

func TestRefundRecord_CanRetry(t *testing.T) {
	record := createTestRefundRecord(t)

	// Pending cannot be retried (it's still in progress)
	assert.False(t, record.CanRetry())

	// Failed can be retried
	err := record.Fail("Error", `{}`)
	require.NoError(t, err)
	assert.True(t, record.CanRetry())
}

func TestRefundRecordStatus_IsValid(t *testing.T) {
	tests := []struct {
		status finance.RefundRecordStatus
		valid  bool
	}{
		{finance.RefundRecordStatusPending, true},
		{finance.RefundRecordStatusProcessing, true},
		{finance.RefundRecordStatusSuccess, true},
		{finance.RefundRecordStatusFailed, true},
		{finance.RefundRecordStatusClosed, true},
		{finance.RefundRecordStatus("INVALID"), false},
		{finance.RefundRecordStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.status.IsValid())
		})
	}
}

func TestRefundRecordStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status   finance.RefundRecordStatus
		terminal bool
	}{
		{finance.RefundRecordStatusPending, false},
		{finance.RefundRecordStatusProcessing, false},
		{finance.RefundRecordStatusSuccess, true},
		{finance.RefundRecordStatusFailed, true},
		{finance.RefundRecordStatusClosed, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.terminal, tt.status.IsTerminal())
		})
	}
}

func TestRefundSourceType_IsValid(t *testing.T) {
	tests := []struct {
		sourceType finance.RefundSourceType
		valid      bool
	}{
		{finance.RefundSourceTypeSalesReturn, true},
		{finance.RefundSourceTypeCreditMemo, true},
		{finance.RefundSourceTypeOrderCancel, true},
		{finance.RefundSourceTypeManual, true},
		{finance.RefundSourceType("INVALID"), false},
		{finance.RefundSourceType(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.sourceType), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.sourceType.IsValid())
		})
	}
}

func TestNewRefundRecordFromCallback(t *testing.T) {
	tenantID := uuid.New()
	refundNumber := "RF-CB-12345678"
	now := time.Now()

	callback := &finance.RefundCallback{
		GatewayType:          finance.PaymentGatewayTypeWechat,
		GatewayRefundID:      "wx_refund_123",
		GatewayOrderID:       "wx_order_456",
		GatewayTransactionID: "wx_txn_789",
		RefundNumber:         "internal-123",
		Status:               finance.RefundStatusSuccess,
		RefundAmount:         decimal.NewFromFloat(100.00),
		RefundedAt:           &now,
		RawPayload:           `{"status": "success"}`,
	}

	record, err := finance.NewRefundRecordFromCallback(tenantID, refundNumber, callback)
	require.NoError(t, err)
	require.NotNil(t, record)

	assert.Equal(t, refundNumber, record.RefundNumber)
	assert.Equal(t, callback.GatewayRefundID, record.GatewayRefundID)
	assert.Equal(t, callback.GatewayOrderID, record.GatewayOrderID)
	assert.Equal(t, callback.GatewayTransactionID, record.GatewayTransactionID)
	assert.True(t, callback.RefundAmount.Equal(record.RefundAmount))
	assert.True(t, callback.RefundAmount.Equal(record.ActualRefundAmount))
	assert.Equal(t, finance.RefundRecordStatusSuccess, record.Status)
	assert.NotNil(t, record.CompletedAt)
}

func TestNewRefundRecordFromCallback_NilCallback(t *testing.T) {
	record, err := finance.NewRefundRecordFromCallback(uuid.New(), "RF-123", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Callback cannot be nil")
	assert.Nil(t, record)
}

func TestNewRefundRecordFromCallback_EmptyRefundNumber(t *testing.T) {
	callback := &finance.RefundCallback{
		GatewayType:  finance.PaymentGatewayTypeWechat,
		RefundAmount: decimal.NewFromFloat(100.00),
	}

	record, err := finance.NewRefundRecordFromCallback(uuid.New(), "", callback)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Refund number cannot be empty")
	assert.Nil(t, record)
}

func TestRefundRecord_CompleteFromCallback(t *testing.T) {
	record := createTestRefundRecord(t)
	now := time.Now()

	callback := &finance.RefundCallback{
		GatewayType:          finance.PaymentGatewayTypeWechat,
		GatewayRefundID:      "wx_refund_new",
		GatewayOrderID:       "wx_order_new",
		GatewayTransactionID: "wx_txn_new",
		Status:               finance.RefundStatusSuccess,
		RefundAmount:         decimal.NewFromFloat(100.00),
		RefundedAt:           &now,
		RawPayload:           `{"result": "success"}`,
	}

	err := record.CompleteFromCallback(callback)
	require.NoError(t, err)

	assert.Equal(t, finance.RefundRecordStatusSuccess, record.Status)
	assert.Equal(t, callback.GatewayRefundID, record.GatewayRefundID)
	assert.Equal(t, callback.GatewayOrderID, record.GatewayOrderID)
	assert.Equal(t, callback.GatewayTransactionID, record.GatewayTransactionID)
	assert.True(t, callback.RefundAmount.Equal(record.ActualRefundAmount))
}

func TestRefundRecord_CompleteFromCallback_NilCallback(t *testing.T) {
	record := createTestRefundRecord(t)

	err := record.CompleteFromCallback(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Callback cannot be nil")
}

// Helper function to create a test refund record
func createTestRefundRecord(t *testing.T) *finance.RefundRecord {
	t.Helper()

	record, err := finance.NewRefundRecord(
		uuid.New(),
		"RF-2026-01-0001",
		uuid.New(),
		"SO-2026-0001",
		finance.RefundSourceTypeSalesReturn,
		uuid.New(),
		"SR-2026-0001",
		uuid.New(),
		"Test Customer",
		decimal.NewFromFloat(100.00),
		finance.PaymentGatewayTypeWechat,
		"Test refund",
	)
	require.NoError(t, err)
	return record
}
