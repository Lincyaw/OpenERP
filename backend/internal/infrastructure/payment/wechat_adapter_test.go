package payment

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erp/backend/internal/domain/finance"
)

func TestWechatPayConfig_Validate(t *testing.T) {
	// Generate a test private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	tests := []struct {
		name    string
		config  *WechatPayConfig
		wantErr error
	}{
		{
			name: "valid config",
			config: &WechatPayConfig{
				MchID:      "1234567890",
				AppID:      "wx1234567890abcdef",
				APIKey:     "12345678901234567890123456789012", // 32 bytes
				SerialNo:   "ABCDEF1234567890",
				PrivateKey: privateKey,
				NotifyURL:  "https://example.com/notify",
			},
			wantErr: nil,
		},
		{
			name: "missing MchID",
			config: &WechatPayConfig{
				AppID:      "wx1234567890abcdef",
				APIKey:     "12345678901234567890123456789012",
				SerialNo:   "ABCDEF1234567890",
				PrivateKey: privateKey,
				NotifyURL:  "https://example.com/notify",
			},
			wantErr: ErrWechatMissingMchID,
		},
		{
			name: "missing AppID",
			config: &WechatPayConfig{
				MchID:      "1234567890",
				APIKey:     "12345678901234567890123456789012",
				SerialNo:   "ABCDEF1234567890",
				PrivateKey: privateKey,
				NotifyURL:  "https://example.com/notify",
			},
			wantErr: ErrWechatMissingAppID,
		},
		{
			name: "invalid API key length",
			config: &WechatPayConfig{
				MchID:      "1234567890",
				AppID:      "wx1234567890abcdef",
				APIKey:     "tooshort",
				SerialNo:   "ABCDEF1234567890",
				PrivateKey: privateKey,
				NotifyURL:  "https://example.com/notify",
			},
			wantErr: ErrWechatInvalidAPIKey,
		},
		{
			name: "missing serial number",
			config: &WechatPayConfig{
				MchID:      "1234567890",
				AppID:      "wx1234567890abcdef",
				APIKey:     "12345678901234567890123456789012",
				PrivateKey: privateKey,
				NotifyURL:  "https://example.com/notify",
			},
			wantErr: ErrWechatMissingSerialNo,
		},
		{
			name: "missing private key",
			config: &WechatPayConfig{
				MchID:     "1234567890",
				AppID:     "wx1234567890abcdef",
				APIKey:    "12345678901234567890123456789012",
				SerialNo:  "ABCDEF1234567890",
				NotifyURL: "https://example.com/notify",
			},
			wantErr: ErrWechatMissingPrivateKey,
		},
		{
			name: "missing notify URL",
			config: &WechatPayConfig{
				MchID:      "1234567890",
				AppID:      "wx1234567890abcdef",
				APIKey:     "12345678901234567890123456789012",
				SerialNo:   "ABCDEF1234567890",
				PrivateKey: privateKey,
			},
			wantErr: ErrWechatMissingNotifyURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewWechatPayAdapter(t *testing.T) {
	// Generate a test private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	t.Run("valid config", func(t *testing.T) {
		config := &WechatPayConfig{
			MchID:      "1234567890",
			AppID:      "wx1234567890abcdef",
			APIKey:     "12345678901234567890123456789012",
			SerialNo:   "ABCDEF1234567890",
			PrivateKey: privateKey,
			NotifyURL:  "https://example.com/notify",
		}

		adapter, err := NewWechatPayAdapter(config)
		require.NoError(t, err)
		assert.NotNil(t, adapter)
		assert.Equal(t, finance.PaymentGatewayTypeWechat, adapter.GatewayType())
	})

	t.Run("invalid config", func(t *testing.T) {
		config := &WechatPayConfig{} // Missing required fields
		adapter, err := NewWechatPayAdapter(config)
		assert.Error(t, err)
		assert.Nil(t, adapter)
	})
}

func TestWechatPayAdapter_GatewayType(t *testing.T) {
	adapter := createTestAdapter(t)
	assert.Equal(t, finance.PaymentGatewayTypeWechat, adapter.GatewayType())
}

func TestWechatPayAdapter_CreatePayment_InvalidRequest(t *testing.T) {
	adapter := createTestAdapter(t)
	ctx := context.Background()

	tests := []struct {
		name    string
		req     *finance.CreatePaymentRequest
		wantErr error
	}{
		{
			name:    "nil tenant ID",
			req:     &finance.CreatePaymentRequest{},
			wantErr: finance.ErrPaymentInvalidTenantID,
		},
		{
			name: "nil order ID",
			req: &finance.CreatePaymentRequest{
				TenantID: uuid.New(),
			},
			wantErr: finance.ErrPaymentInvalidOrderID,
		},
		{
			name: "empty order number",
			req: &finance.CreatePaymentRequest{
				TenantID: uuid.New(),
				OrderID:  uuid.New(),
			},
			wantErr: finance.ErrPaymentInvalidOrderNumber,
		},
		{
			name: "invalid amount",
			req: &finance.CreatePaymentRequest{
				TenantID:    uuid.New(),
				OrderID:     uuid.New(),
				OrderNumber: "ORDER-001",
				Amount:      decimal.Zero,
			},
			wantErr: finance.ErrPaymentInvalidAmount,
		},
		{
			name: "invalid channel",
			req: &finance.CreatePaymentRequest{
				TenantID:    uuid.New(),
				OrderID:     uuid.New(),
				OrderNumber: "ORDER-001",
				Amount:      decimal.NewFromFloat(100.00),
				Channel:     "INVALID",
			},
			wantErr: finance.ErrPaymentInvalidChannel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := adapter.CreatePayment(ctx, tt.req)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestWechatPayAdapter_CreatePayment_NativePayment(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/v3/pay/transactions/native")

		resp := map[string]string{
			"code_url": "weixin://wxpay/bizpayurl?pr=abc123",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := createTestAdapterWithServer(t, server.URL)
	ctx := context.Background()

	req := &finance.CreatePaymentRequest{
		TenantID:    uuid.New(),
		OrderID:     uuid.New(),
		OrderNumber: "ORDER-001",
		Amount:      decimal.NewFromFloat(100.00),
		Channel:     finance.PaymentChannelWechatNative,
		Subject:     "Test Payment",
		NotifyURL:   "https://example.com/notify",
		ExpireTime:  time.Now().Add(30 * time.Minute),
	}

	resp, err := adapter.CreatePayment(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, finance.PaymentGatewayTypeWechat, resp.GatewayType)
	assert.Equal(t, finance.GatewayPaymentStatusPending, resp.Status)
	assert.Equal(t, "weixin://wxpay/bizpayurl?pr=abc123", resp.QRCodeURL)
}

func TestWechatPayAdapter_QueryPayment(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/v3/pay/transactions/out-trade-no")

		resp := wechatQueryResponse{
			AppID:         "wx1234567890abcdef",
			MchID:         "1234567890",
			OutTradeNo:    "ORDER-001",
			TransactionID: "TXN123456789",
			TradeState:    "SUCCESS",
			SuccessTime:   "2024-01-01T12:00:00+08:00",
			Payer:         &wechatPayer{OpenID: "oWx123456"},
			Amount: &wechatAmount{
				Total:      10000,
				PayerTotal: 10000,
				Currency:   "CNY",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := createTestAdapterWithServer(t, server.URL)
	ctx := context.Background()

	req := &finance.QueryPaymentRequest{
		TenantID:    uuid.New(),
		OrderNumber: "ORDER-001",
		GatewayType: finance.PaymentGatewayTypeWechat,
	}

	resp, err := adapter.QueryPayment(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "ORDER-001", resp.OrderNumber)
	assert.Equal(t, "TXN123456789", resp.GatewayTransactionID)
	assert.Equal(t, finance.GatewayPaymentStatusPaid, resp.Status)
	assert.True(t, resp.Amount.Equal(decimal.NewFromFloat(100.00)))
	assert.NotNil(t, resp.PaidAt)
}

func TestWechatPayAdapter_ClosePayment(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/close")

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	adapter := createTestAdapterWithServer(t, server.URL)
	ctx := context.Background()

	req := &finance.ClosePaymentRequest{
		TenantID:    uuid.New(),
		OrderNumber: "ORDER-001",
		GatewayType: finance.PaymentGatewayTypeWechat,
	}

	resp, err := adapter.ClosePayment(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
}

func TestWechatPayAdapter_CreateRefund(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/v3/refund/domestic/refunds")

		resp := wechatRefundResponse{
			RefundID:    "REFUND123456",
			OutRefundNo: "REFUND-001",
			Status:      "SUCCESS",
			SuccessTime: "2024-01-01T13:00:00+08:00",
			Amount: struct {
				Total            int    `json:"total"`
				Refund           int    `json:"refund"`
				PayerTotal       int    `json:"payer_total"`
				PayerRefund      int    `json:"payer_refund"`
				SettlementTotal  int    `json:"settlement_total"`
				SettlementRefund int    `json:"settlement_refund"`
				DiscountRefund   int    `json:"discount_refund"`
				Currency         string `json:"currency"`
			}{
				Total:    10000,
				Refund:   5000,
				Currency: "CNY",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := createTestAdapterWithServer(t, server.URL)
	ctx := context.Background()

	req := &finance.RefundRequest{
		TenantID:       uuid.New(),
		GatewayOrderID: "ORDER-001",
		RefundID:       uuid.New(),
		RefundNumber:   "REFUND-001",
		TotalAmount:    decimal.NewFromFloat(100.00),
		RefundAmount:   decimal.NewFromFloat(50.00),
		Reason:         "Customer request",
		GatewayType:    finance.PaymentGatewayTypeWechat,
	}

	resp, err := adapter.CreateRefund(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "REFUND123456", resp.GatewayRefundID)
	assert.Equal(t, finance.RefundStatusSuccess, resp.Status)
	assert.True(t, resp.RefundAmount.Equal(decimal.NewFromFloat(50.00)))
}

func TestWechatPayAdapter_GenerateCallbackResponse(t *testing.T) {
	adapter := createTestAdapter(t)

	t.Run("success response", func(t *testing.T) {
		resp := adapter.GenerateCallbackResponse(true, "")
		var result map[string]string
		err := json.Unmarshal(resp, &result)
		require.NoError(t, err)
		assert.Equal(t, "SUCCESS", result["code"])
	})

	t.Run("failure response", func(t *testing.T) {
		resp := adapter.GenerateCallbackResponse(false, "Processing failed")
		var result map[string]string
		err := json.Unmarshal(resp, &result)
		require.NoError(t, err)
		assert.Equal(t, "FAIL", result["code"])
		assert.Equal(t, "Processing failed", result["message"])
	})
}

func TestMapWechatTradeState(t *testing.T) {
	tests := []struct {
		state    string
		expected finance.GatewayPaymentStatus
	}{
		{"SUCCESS", finance.GatewayPaymentStatusPaid},
		{"REFUND", finance.GatewayPaymentStatusRefunded},
		{"NOTPAY", finance.GatewayPaymentStatusPending},
		{"CLOSED", finance.GatewayPaymentStatusClosed},
		{"REVOKED", finance.GatewayPaymentStatusCancelled},
		{"USERPAYING", finance.GatewayPaymentStatusPending},
		{"PAYERROR", finance.GatewayPaymentStatusFailed},
		{"UNKNOWN", finance.GatewayPaymentStatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			result := mapWechatTradeState(tt.state)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapWechatRefundStatus(t *testing.T) {
	tests := []struct {
		status   string
		expected finance.RefundStatus
	}{
		{"SUCCESS", finance.RefundStatusSuccess},
		{"CLOSED", finance.RefundStatusClosed},
		{"PROCESSING", finance.RefundStatusPending},
		{"ABNORMAL", finance.RefundStatusFailed},
		{"UNKNOWN", finance.RefundStatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := mapWechatRefundStatus(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// createTestAdapter creates a test adapter with a mock configuration
func createTestAdapter(t *testing.T) *WechatPayAdapter {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	config := &WechatPayConfig{
		MchID:      "1234567890",
		AppID:      "wx1234567890abcdef",
		APIKey:     "12345678901234567890123456789012",
		SerialNo:   "ABCDEF1234567890",
		PrivateKey: privateKey,
		NotifyURL:  "https://example.com/notify",
	}

	adapter, err := NewWechatPayAdapter(config)
	require.NoError(t, err)
	return adapter
}

// createTestAdapterWithServer creates a test adapter that uses a mock server
func createTestAdapterWithServer(t *testing.T, serverURL string) *WechatPayAdapter {
	adapter := createTestAdapter(t)

	// Override the base URL for testing
	// In a real implementation, you might inject the HTTP client or use a test hook
	adapter.httpClient = &http.Client{
		Transport: &testTransport{baseURL: serverURL},
		Timeout:   30 * time.Second,
	}

	return adapter
}

// testTransport is a custom transport that rewrites URLs for testing
type testTransport struct {
	baseURL string
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite the URL to use our test server
	req.URL.Scheme = "http"
	req.URL.Host = t.baseURL[7:] // Remove "http://" prefix

	return http.DefaultTransport.RoundTrip(req)
}
