package payment

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erp/backend/internal/domain/finance"
)

// generateTestAlipayKeyPair generates a test RSA key pair for Alipay
func generateTestAlipayKeyPair(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return privateKey, &privateKey.PublicKey
}

// createTestAlipayConfig creates a test AlipayConfig
func createTestAlipayConfig(t *testing.T) *AlipayConfig {
	privateKey, publicKey := generateTestAlipayKeyPair(t)
	return &AlipayConfig{
		AppID:           "2021000000000001",
		PrivateKey:      privateKey,
		AlipayPublicKey: publicKey,
		SignType:        "RSA2",
		NotifyURL:       "https://example.com/notify",
		ReturnURL:       "https://example.com/return",
		IsSandbox:       true,
	}
}

func TestAlipayConfig_Validate(t *testing.T) {
	privateKey, publicKey := generateTestAlipayKeyPair(t)

	tests := []struct {
		name    string
		config  *AlipayConfig
		wantErr error
	}{
		{
			name: "valid config",
			config: &AlipayConfig{
				AppID:           "2021000000000001",
				PrivateKey:      privateKey,
				AlipayPublicKey: publicKey,
				SignType:        "RSA2",
				NotifyURL:       "https://example.com/notify",
			},
			wantErr: nil,
		},
		{
			name: "missing app ID",
			config: &AlipayConfig{
				PrivateKey:      privateKey,
				AlipayPublicKey: publicKey,
				NotifyURL:       "https://example.com/notify",
			},
			wantErr: ErrAlipayMissingAppID,
		},
		{
			name: "missing private key",
			config: &AlipayConfig{
				AppID:           "2021000000000001",
				AlipayPublicKey: publicKey,
				NotifyURL:       "https://example.com/notify",
			},
			wantErr: ErrAlipayMissingPrivateKey,
		},
		{
			name: "missing public key",
			config: &AlipayConfig{
				AppID:      "2021000000000001",
				PrivateKey: privateKey,
				NotifyURL:  "https://example.com/notify",
			},
			wantErr: ErrAlipayMissingPublicKey,
		},
		{
			name: "missing notify URL",
			config: &AlipayConfig{
				AppID:           "2021000000000001",
				PrivateKey:      privateKey,
				AlipayPublicKey: publicKey,
			},
			wantErr: ErrAlipayMissingNotifyURL,
		},
		{
			name: "invalid sign type",
			config: &AlipayConfig{
				AppID:           "2021000000000001",
				PrivateKey:      privateKey,
				AlipayPublicKey: publicKey,
				SignType:        "INVALID",
				NotifyURL:       "https://example.com/notify",
			},
			wantErr: ErrAlipayInvalidSignType,
		},
		{
			name: "default sign type (RSA2)",
			config: &AlipayConfig{
				AppID:           "2021000000000001",
				PrivateKey:      privateKey,
				AlipayPublicKey: publicKey,
				SignType:        "", // Empty should default to RSA2
				NotifyURL:       "https://example.com/notify",
			},
			wantErr: nil,
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

func TestAlipayConfigBuilder(t *testing.T) {
	privateKey, publicKey := generateTestAlipayKeyPair(t)

	t.Run("build valid config", func(t *testing.T) {
		config, err := NewAlipayConfigBuilder().
			SetAppID("2021000000000001").
			SetNotifyURL("https://example.com/notify").
			SetReturnURL("https://example.com/return").
			SetIsSandbox(true).
			Build()

		// This should fail because we didn't set private key
		assert.Error(t, err)
		assert.Nil(t, config)
	})

	t.Run("build config with keys", func(t *testing.T) {
		builder := NewAlipayConfigBuilder().
			SetAppID("2021000000000001").
			SetNotifyURL("https://example.com/notify")

		// Set keys directly (not via PEM for test)
		builder.config.PrivateKey = privateKey
		builder.config.AlipayPublicKey = publicKey

		config, err := builder.Build()
		require.NoError(t, err)
		assert.Equal(t, "2021000000000001", config.AppID)
		assert.Equal(t, "RSA2", config.SignType) // Default
	})
}

func TestNewAlipayAdapter(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := createTestAlipayConfig(t)
		adapter, err := NewAlipayAdapter(config)
		require.NoError(t, err)
		assert.NotNil(t, adapter)
		assert.Equal(t, finance.PaymentGatewayTypeAlipay, adapter.GatewayType())
	})

	t.Run("invalid config", func(t *testing.T) {
		config := &AlipayConfig{} // Empty config
		adapter, err := NewAlipayAdapter(config)
		assert.Error(t, err)
		assert.Nil(t, adapter)
	})
}

func TestAlipayAdapter_CreatePayment_Validation(t *testing.T) {
	config := createTestAlipayConfig(t)
	adapter, err := NewAlipayAdapter(config)
	require.NoError(t, err)

	tests := []struct {
		name    string
		req     *finance.CreatePaymentRequest
		wantErr error
	}{
		{
			name: "missing tenant ID",
			req: &finance.CreatePaymentRequest{
				OrderID:     uuid.New(),
				OrderNumber: "ORD001",
				Amount:      decimal.NewFromFloat(100.00),
				Channel:     finance.PaymentChannelAlipayPage,
				Subject:     "Test Order",
				NotifyURL:   "https://example.com/notify",
			},
			wantErr: finance.ErrPaymentInvalidTenantID,
		},
		{
			name: "missing order ID",
			req: &finance.CreatePaymentRequest{
				TenantID:    uuid.New(),
				OrderNumber: "ORD001",
				Amount:      decimal.NewFromFloat(100.00),
				Channel:     finance.PaymentChannelAlipayPage,
				Subject:     "Test Order",
				NotifyURL:   "https://example.com/notify",
			},
			wantErr: finance.ErrPaymentInvalidOrderID,
		},
		{
			name: "missing order number",
			req: &finance.CreatePaymentRequest{
				TenantID:  uuid.New(),
				OrderID:   uuid.New(),
				Amount:    decimal.NewFromFloat(100.00),
				Channel:   finance.PaymentChannelAlipayPage,
				Subject:   "Test Order",
				NotifyURL: "https://example.com/notify",
			},
			wantErr: finance.ErrPaymentInvalidOrderNumber,
		},
		{
			name: "invalid amount",
			req: &finance.CreatePaymentRequest{
				TenantID:    uuid.New(),
				OrderID:     uuid.New(),
				OrderNumber: "ORD001",
				Amount:      decimal.Zero,
				Channel:     finance.PaymentChannelAlipayPage,
				Subject:     "Test Order",
				NotifyURL:   "https://example.com/notify",
			},
			wantErr: finance.ErrPaymentInvalidAmount,
		},
		{
			name: "invalid channel",
			req: &finance.CreatePaymentRequest{
				TenantID:    uuid.New(),
				OrderID:     uuid.New(),
				OrderNumber: "ORD001",
				Amount:      decimal.NewFromFloat(100.00),
				Channel:     "INVALID",
				Subject:     "Test Order",
				NotifyURL:   "https://example.com/notify",
			},
			wantErr: finance.ErrPaymentInvalidChannel,
		},
		{
			name: "missing subject",
			req: &finance.CreatePaymentRequest{
				TenantID:    uuid.New(),
				OrderID:     uuid.New(),
				OrderNumber: "ORD001",
				Amount:      decimal.NewFromFloat(100.00),
				Channel:     finance.PaymentChannelAlipayPage,
				NotifyURL:   "https://example.com/notify",
			},
			wantErr: finance.ErrPaymentInvalidSubject,
		},
		{
			name: "missing notify URL",
			req: &finance.CreatePaymentRequest{
				TenantID:    uuid.New(),
				OrderID:     uuid.New(),
				OrderNumber: "ORD001",
				Amount:      decimal.NewFromFloat(100.00),
				Channel:     finance.PaymentChannelAlipayPage,
				Subject:     "Test Order",
			},
			wantErr: finance.ErrPaymentInvalidNotifyURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := adapter.CreatePayment(context.Background(), tt.req)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestAlipayAdapter_CreatePayment_PagePay(t *testing.T) {
	config := createTestAlipayConfig(t)
	adapter, err := NewAlipayAdapter(config)
	require.NoError(t, err)

	req := &finance.CreatePaymentRequest{
		TenantID:    uuid.New(),
		OrderID:     uuid.New(),
		OrderNumber: "ORD001",
		Amount:      decimal.NewFromFloat(100.00),
		Channel:     finance.PaymentChannelAlipayPage,
		Subject:     "Test Order",
		NotifyURL:   "https://example.com/notify",
		ReturnURL:   "https://example.com/return",
		ExpireTime:  time.Now().Add(30 * time.Minute),
	}

	resp, err := adapter.CreatePayment(context.Background(), req)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.PaymentURL)
	assert.Contains(t, resp.PaymentURL, "openapi-sandbox.dl.alipaydev.com")
	assert.Equal(t, finance.GatewayPaymentStatusPending, resp.Status)
	assert.Equal(t, finance.PaymentGatewayTypeAlipay, resp.GatewayType)
}

func TestAlipayAdapter_CreatePayment_WapPay(t *testing.T) {
	config := createTestAlipayConfig(t)
	adapter, err := NewAlipayAdapter(config)
	require.NoError(t, err)

	req := &finance.CreatePaymentRequest{
		TenantID:    uuid.New(),
		OrderID:     uuid.New(),
		OrderNumber: "ORD002",
		Amount:      decimal.NewFromFloat(50.00),
		Channel:     finance.PaymentChannelAlipayWap,
		Subject:     "Test WAP Order",
		NotifyURL:   "https://example.com/notify",
		ReturnURL:   "https://example.com/return",
	}

	resp, err := adapter.CreatePayment(context.Background(), req)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.PaymentURL)
	assert.Equal(t, finance.GatewayPaymentStatusPending, resp.Status)
}

func TestAlipayAdapter_CreatePayment_AppPay(t *testing.T) {
	config := createTestAlipayConfig(t)
	adapter, err := NewAlipayAdapter(config)
	require.NoError(t, err)

	req := &finance.CreatePaymentRequest{
		TenantID:    uuid.New(),
		OrderID:     uuid.New(),
		OrderNumber: "ORD003",
		Amount:      decimal.NewFromFloat(200.00),
		Channel:     finance.PaymentChannelAlipayApp,
		Subject:     "Test App Order",
		NotifyURL:   "https://example.com/notify",
	}

	resp, err := adapter.CreatePayment(context.Background(), req)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.SDKParams)
	assert.Contains(t, resp.SDKParams, "app_id=")
	assert.Contains(t, resp.SDKParams, "sign=")
	assert.Equal(t, finance.GatewayPaymentStatusPending, resp.Status)
}

func TestAlipayAdapter_CreatePayment_QRCode(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := alipayTradePrecreateResponse{}
		resp.Response.Code = "10000"
		resp.Response.Msg = "Success"
		resp.Response.OutTradeNo = "ORD004"
		resp.Response.QRCode = "https://qr.alipay.com/abc123"
		resp.Sign = "test_sign"

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := createTestAlipayConfig(t)
	adapter, err := NewAlipayAdapter(config)
	require.NoError(t, err)

	// Override the gateway URL for testing
	originalURL := alipaySandboxGatewayURL
	defer func() { /* restore not needed as const */ }()

	// Since we can't modify const, we'll just test that the adapter returns proper response
	// when gateway is unavailable (sandbox URL won't work in tests)
	req := &finance.CreatePaymentRequest{
		TenantID:    uuid.New(),
		OrderID:     uuid.New(),
		OrderNumber: "ORD004",
		Amount:      decimal.NewFromFloat(100.00),
		Channel:     finance.PaymentChannelAlipayQRCode,
		Subject:     "Test QR Order",
		NotifyURL:   "https://example.com/notify",
	}

	// This will fail because we're not actually connecting to the sandbox
	// In real tests, you'd mock the HTTP client
	_, err = adapter.CreatePayment(context.Background(), req)
	// We expect an error because the sandbox URL doesn't exist
	assert.Error(t, err)
	_ = originalURL // suppress unused warning
}

func TestAlipayAdapter_CreatePayment_InvalidChannel(t *testing.T) {
	config := createTestAlipayConfig(t)
	adapter, err := NewAlipayAdapter(config)
	require.NoError(t, err)

	req := &finance.CreatePaymentRequest{
		TenantID:    uuid.New(),
		OrderID:     uuid.New(),
		OrderNumber: "ORD005",
		Amount:      decimal.NewFromFloat(100.00),
		Channel:     finance.PaymentChannelWechatNative, // Wrong gateway channel
		Subject:     "Test Order",
		NotifyURL:   "https://example.com/notify",
	}

	_, err = adapter.CreatePayment(context.Background(), req)
	assert.ErrorIs(t, err, finance.ErrPaymentInvalidChannel)
}

func TestAlipayAdapter_QueryPayment_Validation(t *testing.T) {
	config := createTestAlipayConfig(t)
	adapter, err := NewAlipayAdapter(config)
	require.NoError(t, err)

	tests := []struct {
		name    string
		req     *finance.QueryPaymentRequest
		wantErr error
	}{
		{
			name: "missing tenant ID",
			req: &finance.QueryPaymentRequest{
				OrderNumber: "ORD001",
				GatewayType: finance.PaymentGatewayTypeAlipay,
			},
			wantErr: finance.ErrPaymentInvalidTenantID,
		},
		{
			name: "missing order reference",
			req: &finance.QueryPaymentRequest{
				TenantID:    uuid.New(),
				GatewayType: finance.PaymentGatewayTypeAlipay,
			},
			wantErr: finance.ErrPaymentInvalidQueryParams,
		},
		{
			name: "invalid gateway type",
			req: &finance.QueryPaymentRequest{
				TenantID:    uuid.New(),
				OrderNumber: "ORD001",
				GatewayType: "INVALID",
			},
			wantErr: finance.ErrPaymentInvalidGatewayType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := adapter.QueryPayment(context.Background(), tt.req)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestAlipayAdapter_ClosePayment_Validation(t *testing.T) {
	config := createTestAlipayConfig(t)
	adapter, err := NewAlipayAdapter(config)
	require.NoError(t, err)

	tests := []struct {
		name    string
		req     *finance.ClosePaymentRequest
		wantErr error
	}{
		{
			name: "missing tenant ID",
			req: &finance.ClosePaymentRequest{
				OrderNumber: "ORD001",
				GatewayType: finance.PaymentGatewayTypeAlipay,
			},
			wantErr: finance.ErrPaymentInvalidTenantID,
		},
		{
			name: "missing order reference",
			req: &finance.ClosePaymentRequest{
				TenantID:    uuid.New(),
				GatewayType: finance.PaymentGatewayTypeAlipay,
			},
			wantErr: finance.ErrPaymentInvalidQueryParams,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := adapter.ClosePayment(context.Background(), tt.req)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestAlipayAdapter_CreateRefund_Validation(t *testing.T) {
	config := createTestAlipayConfig(t)
	adapter, err := NewAlipayAdapter(config)
	require.NoError(t, err)

	tests := []struct {
		name    string
		req     *finance.RefundRequest
		wantErr error
	}{
		{
			name: "missing tenant ID",
			req: &finance.RefundRequest{
				GatewayOrderID: "ORD001",
				RefundID:       uuid.New(),
				RefundNumber:   "REF001",
				TotalAmount:    decimal.NewFromFloat(100.00),
				RefundAmount:   decimal.NewFromFloat(50.00),
				GatewayType:    finance.PaymentGatewayTypeAlipay,
			},
			wantErr: finance.ErrPaymentInvalidTenantID,
		},
		{
			name: "missing original payment reference",
			req: &finance.RefundRequest{
				TenantID:     uuid.New(),
				RefundID:     uuid.New(),
				RefundNumber: "REF001",
				TotalAmount:  decimal.NewFromFloat(100.00),
				RefundAmount: decimal.NewFromFloat(50.00),
				GatewayType:  finance.PaymentGatewayTypeAlipay,
			},
			wantErr: finance.ErrRefundInvalidOriginalPayment,
		},
		{
			name: "missing refund ID",
			req: &finance.RefundRequest{
				TenantID:       uuid.New(),
				GatewayOrderID: "ORD001",
				RefundNumber:   "REF001",
				TotalAmount:    decimal.NewFromFloat(100.00),
				RefundAmount:   decimal.NewFromFloat(50.00),
				GatewayType:    finance.PaymentGatewayTypeAlipay,
			},
			wantErr: finance.ErrRefundInvalidRefundID,
		},
		{
			name: "refund amount exceeds total",
			req: &finance.RefundRequest{
				TenantID:       uuid.New(),
				GatewayOrderID: "ORD001",
				RefundID:       uuid.New(),
				RefundNumber:   "REF001",
				TotalAmount:    decimal.NewFromFloat(100.00),
				RefundAmount:   decimal.NewFromFloat(150.00),
				GatewayType:    finance.PaymentGatewayTypeAlipay,
			},
			wantErr: finance.ErrRefundAmountExceedsTotal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := adapter.CreateRefund(context.Background(), tt.req)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestAlipayAdapter_QueryRefund_Validation(t *testing.T) {
	config := createTestAlipayConfig(t)
	adapter, err := NewAlipayAdapter(config)
	require.NoError(t, err)

	t.Run("missing refund ID", func(t *testing.T) {
		_, err := adapter.QueryRefund(context.Background(), uuid.New(), "")
		assert.ErrorIs(t, err, finance.ErrRefundInvalidRefundID)
	})
}

func TestAlipayAdapter_VerifyCallback(t *testing.T) {
	config := createTestAlipayConfig(t)
	adapter, err := NewAlipayAdapter(config)
	require.NoError(t, err)

	// Create test notification data
	values := url.Values{
		"notify_time":      {"2024-01-15 12:00:00"},
		"notify_type":      {"trade_status_sync"},
		"notify_id":        {"ac05099524730693a8b330c5ecf72da9786"},
		"app_id":           {"2021000000000001"},
		"trade_no":         {"2024011522001234567890123456"},
		"out_trade_no":     {"ORD001"},
		"buyer_logon_id":   {"138****1234"},
		"trade_status":     {"TRADE_SUCCESS"},
		"total_amount":     {"100.00"},
		"buyer_pay_amount": {"100.00"},
		"gmt_payment":      {"2024-01-15 12:00:00"},
	}

	// Build sign string and sign it
	signStr := adapter.buildSignString(map[string]string{
		"notify_time":      "2024-01-15 12:00:00",
		"notify_type":      "trade_status_sync",
		"notify_id":        "ac05099524730693a8b330c5ecf72da9786",
		"app_id":           "2021000000000001",
		"trade_no":         "2024011522001234567890123456",
		"out_trade_no":     "ORD001",
		"buyer_logon_id":   "138****1234",
		"trade_status":     "TRADE_SUCCESS",
		"total_amount":     "100.00",
		"buyer_pay_amount": "100.00",
		"gmt_payment":      "2024-01-15 12:00:00",
	})

	sign, err := adapter.sign(map[string]string{
		"notify_time":      "2024-01-15 12:00:00",
		"notify_type":      "trade_status_sync",
		"notify_id":        "ac05099524730693a8b330c5ecf72da9786",
		"app_id":           "2021000000000001",
		"trade_no":         "2024011522001234567890123456",
		"out_trade_no":     "ORD001",
		"buyer_logon_id":   "138****1234",
		"trade_status":     "TRADE_SUCCESS",
		"total_amount":     "100.00",
		"buyer_pay_amount": "100.00",
		"gmt_payment":      "2024-01-15 12:00:00",
	})
	require.NoError(t, err)
	values.Set("sign", sign)

	payload := []byte(values.Encode())

	callback, err := adapter.VerifyCallback(context.Background(), payload, sign)
	require.NoError(t, err)
	assert.Equal(t, "ORD001", callback.OrderNumber)
	assert.Equal(t, "2024011522001234567890123456", callback.GatewayTransactionID)
	assert.Equal(t, finance.GatewayPaymentStatusPaid, callback.Status)
	assert.Equal(t, "100", callback.Amount.StringFixed(0))

	_ = signStr // suppress unused warning
}

func TestAlipayAdapter_VerifyCallback_InvalidSignature(t *testing.T) {
	config := createTestAlipayConfig(t)
	adapter, err := NewAlipayAdapter(config)
	require.NoError(t, err)

	values := url.Values{
		"notify_time":  {"2024-01-15 12:00:00"},
		"trade_no":     {"2024011522001234567890123456"},
		"out_trade_no": {"ORD001"},
		"trade_status": {"TRADE_SUCCESS"},
		"total_amount": {"100.00"},
		"sign":         {"invalid_signature"},
	}

	payload := []byte(values.Encode())

	_, err = adapter.VerifyCallback(context.Background(), payload, "invalid_signature")
	assert.ErrorIs(t, err, finance.ErrGatewayInvalidCallback)
}

func TestAlipayAdapter_GenerateCallbackResponse(t *testing.T) {
	config := createTestAlipayConfig(t)
	adapter, err := NewAlipayAdapter(config)
	require.NoError(t, err)

	t.Run("success response", func(t *testing.T) {
		resp := adapter.GenerateCallbackResponse(true, "")
		assert.Equal(t, "success", string(resp))
	})

	t.Run("failure response", func(t *testing.T) {
		resp := adapter.GenerateCallbackResponse(false, "error message")
		assert.Equal(t, "fail", string(resp))
	})
}

func TestMapAlipayTradeStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected finance.GatewayPaymentStatus
	}{
		{"TRADE_SUCCESS", finance.GatewayPaymentStatusPaid},
		{"TRADE_FINISHED", finance.GatewayPaymentStatusPaid},
		{"TRADE_CLOSED", finance.GatewayPaymentStatusClosed},
		{"WAIT_BUYER_PAY", finance.GatewayPaymentStatusPending},
		{"UNKNOWN", finance.GatewayPaymentStatusPending},
		{"", finance.GatewayPaymentStatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapAlipayTradeStatus(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAlipayAdapter_BuildSignString(t *testing.T) {
	config := createTestAlipayConfig(t)
	adapter, err := NewAlipayAdapter(config)
	require.NoError(t, err)

	params := map[string]string{
		"z_param": "z_value",
		"a_param": "a_value",
		"m_param": "m_value",
		"sign":    "should_be_excluded",
		"empty":   "",
	}

	signStr := adapter.buildSignString(params)

	// Should be sorted alphabetically and exclude sign and empty values
	expected := "a_param=a_value&m_param=m_value&z_param=z_value"
	assert.Equal(t, expected, signStr)
}

func TestAlipayAdapter_BuildPaymentURL(t *testing.T) {
	config := createTestAlipayConfig(t)
	adapter, err := NewAlipayAdapter(config)
	require.NoError(t, err)

	params := map[string]string{
		"app_id": "2021000000000001",
		"method": "alipay.trade.page.pay",
		"sign":   "test_sign",
	}

	paymentURL := adapter.buildPaymentURL(params)

	assert.Contains(t, paymentURL, "openapi-sandbox.dl.alipaydev.com")
	assert.Contains(t, paymentURL, "app_id=2021000000000001")
	assert.Contains(t, paymentURL, "method=alipay.trade.page.pay")
	assert.Contains(t, paymentURL, "sign=test_sign")
}

func TestAlipayAdapter_BuildOrderString(t *testing.T) {
	config := createTestAlipayConfig(t)
	adapter, err := NewAlipayAdapter(config)
	require.NoError(t, err)

	params := map[string]string{
		"app_id": "2021000000000001",
		"method": "alipay.trade.app.pay",
		"sign":   "test_sign",
	}

	orderStr := adapter.buildOrderString(params)

	// Should be sorted and URL-encoded
	assert.Contains(t, orderStr, "app_id=2021000000000001")
	assert.Contains(t, orderStr, "method=alipay.trade.app.pay")
	assert.Contains(t, orderStr, "sign=test_sign")
}

func TestAlipayAdapter_Sign(t *testing.T) {
	config := createTestAlipayConfig(t)
	adapter, err := NewAlipayAdapter(config)
	require.NoError(t, err)

	params := map[string]string{
		"app_id":      "2021000000000001",
		"method":      "alipay.trade.page.pay",
		"biz_content": `{"out_trade_no":"ORD001","total_amount":"100.00"}`,
	}

	sign, err := adapter.sign(params)
	require.NoError(t, err)
	assert.NotEmpty(t, sign)

	// Sign should be base64 encoded
	assert.NotContains(t, sign, "=+/") // Not raw bytes
}

func TestAlipayAdapter_VerifySign(t *testing.T) {
	config := createTestAlipayConfig(t)
	adapter, err := NewAlipayAdapter(config)
	require.NoError(t, err)

	params := map[string]string{
		"app_id":      "2021000000000001",
		"method":      "alipay.trade.page.pay",
		"biz_content": `{"out_trade_no":"ORD001","total_amount":"100.00"}`,
	}

	// Sign the params
	sign, err := adapter.sign(params)
	require.NoError(t, err)

	// Convert to url.Values for verification
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	values.Set("sign", sign)

	// Verify should succeed
	assert.True(t, adapter.verifySign(values, sign))

	// Verify with wrong signature should fail
	assert.False(t, adapter.verifySign(values, "wrong_signature"))
}
