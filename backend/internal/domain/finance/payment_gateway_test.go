package finance

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestPaymentGatewayType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		gateway  PaymentGatewayType
		expected bool
	}{
		{"wechat", PaymentGatewayTypeWechat, true},
		{"alipay", PaymentGatewayTypeAlipay, true},
		{"empty", "", false},
		{"invalid", "INVALID", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.gateway.IsValid())
		})
	}
}

func TestPaymentChannel_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		channel  PaymentChannel
		expected bool
	}{
		{"wechat_native", PaymentChannelWechatNative, true},
		{"wechat_jsapi", PaymentChannelWechatJSAPI, true},
		{"wechat_app", PaymentChannelWechatApp, true},
		{"alipay_page", PaymentChannelAlipayPage, true},
		{"alipay_wap", PaymentChannelAlipayWap, true},
		{"alipay_app", PaymentChannelAlipayApp, true},
		{"alipay_qrcode", PaymentChannelAlipayQRCode, true},
		{"alipay_f2f", PaymentChannelAlipayFaceToFace, true},
		{"empty", "", false},
		{"invalid", "INVALID", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.channel.IsValid())
		})
	}
}

func TestPaymentChannel_GetGatewayType(t *testing.T) {
	tests := []struct {
		name     string
		channel  PaymentChannel
		expected PaymentGatewayType
	}{
		{"wechat_native", PaymentChannelWechatNative, PaymentGatewayTypeWechat},
		{"wechat_jsapi", PaymentChannelWechatJSAPI, PaymentGatewayTypeWechat},
		{"wechat_app", PaymentChannelWechatApp, PaymentGatewayTypeWechat},
		{"alipay_page", PaymentChannelAlipayPage, PaymentGatewayTypeAlipay},
		{"alipay_wap", PaymentChannelAlipayWap, PaymentGatewayTypeAlipay},
		{"alipay_app", PaymentChannelAlipayApp, PaymentGatewayTypeAlipay},
		{"alipay_qrcode", PaymentChannelAlipayQRCode, PaymentGatewayTypeAlipay},
		{"alipay_f2f", PaymentChannelAlipayFaceToFace, PaymentGatewayTypeAlipay},
		{"invalid", "INVALID", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.channel.GetGatewayType())
		})
	}
}

func TestGatewayPaymentStatus_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		status   GatewayPaymentStatus
		expected bool
	}{
		{"pending", GatewayPaymentStatusPending, true},
		{"paid", GatewayPaymentStatusPaid, true},
		{"failed", GatewayPaymentStatusFailed, true},
		{"cancelled", GatewayPaymentStatusCancelled, true},
		{"refunded", GatewayPaymentStatusRefunded, true},
		{"partial_refunded", GatewayPaymentStatusPartialRefunded, true},
		{"closed", GatewayPaymentStatusClosed, true},
		{"empty", "", false},
		{"invalid", "INVALID", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.IsValid())
		})
	}
}

func TestGatewayPaymentStatus_IsFinal(t *testing.T) {
	tests := []struct {
		name     string
		status   GatewayPaymentStatus
		expected bool
	}{
		{"pending", GatewayPaymentStatusPending, false},
		{"paid", GatewayPaymentStatusPaid, true},
		{"failed", GatewayPaymentStatusFailed, true},
		{"cancelled", GatewayPaymentStatusCancelled, true},
		{"refunded", GatewayPaymentStatusRefunded, true},
		{"partial_refunded", GatewayPaymentStatusPartialRefunded, false},
		{"closed", GatewayPaymentStatusClosed, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.IsFinal())
		})
	}
}

func TestGatewayPaymentStatus_IsSuccess(t *testing.T) {
	tests := []struct {
		name     string
		status   GatewayPaymentStatus
		expected bool
	}{
		{"pending", GatewayPaymentStatusPending, false},
		{"paid", GatewayPaymentStatusPaid, true},
		{"failed", GatewayPaymentStatusFailed, false},
		{"cancelled", GatewayPaymentStatusCancelled, false},
		{"refunded", GatewayPaymentStatusRefunded, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.IsSuccess())
		})
	}
}

func TestRefundStatus_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		status   RefundStatus
		expected bool
	}{
		{"pending", RefundStatusPending, true},
		{"success", RefundStatusSuccess, true},
		{"failed", RefundStatusFailed, true},
		{"closed", RefundStatusClosed, true},
		{"empty", "", false},
		{"invalid", "INVALID", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.IsValid())
		})
	}
}

func TestCreatePaymentRequest_Validate(t *testing.T) {
	validRequest := CreatePaymentRequest{
		TenantID:    uuid.New(),
		OrderID:     uuid.New(),
		OrderNumber: "SO-2024-001",
		Amount:      decimal.NewFromFloat(100.00),
		Channel:     PaymentChannelWechatNative,
		Subject:     "Test Payment",
		NotifyURL:   "https://example.com/callback",
		ExpireTime:  time.Now().Add(30 * time.Minute),
	}

	t.Run("valid_request", func(t *testing.T) {
		err := validRequest.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing_tenant_id", func(t *testing.T) {
		req := validRequest
		req.TenantID = uuid.Nil
		err := req.Validate()
		assert.Equal(t, ErrPaymentInvalidTenantID, err)
	})

	t.Run("missing_order_id", func(t *testing.T) {
		req := validRequest
		req.OrderID = uuid.Nil
		err := req.Validate()
		assert.Equal(t, ErrPaymentInvalidOrderID, err)
	})

	t.Run("missing_order_number", func(t *testing.T) {
		req := validRequest
		req.OrderNumber = ""
		err := req.Validate()
		assert.Equal(t, ErrPaymentInvalidOrderNumber, err)
	})

	t.Run("invalid_amount_zero", func(t *testing.T) {
		req := validRequest
		req.Amount = decimal.Zero
		err := req.Validate()
		assert.Equal(t, ErrPaymentInvalidAmount, err)
	})

	t.Run("invalid_amount_negative", func(t *testing.T) {
		req := validRequest
		req.Amount = decimal.NewFromFloat(-100.00)
		err := req.Validate()
		assert.Equal(t, ErrPaymentInvalidAmount, err)
	})

	t.Run("invalid_channel", func(t *testing.T) {
		req := validRequest
		req.Channel = "INVALID"
		err := req.Validate()
		assert.Equal(t, ErrPaymentInvalidChannel, err)
	})

	t.Run("missing_subject", func(t *testing.T) {
		req := validRequest
		req.Subject = ""
		err := req.Validate()
		assert.Equal(t, ErrPaymentInvalidSubject, err)
	})

	t.Run("missing_notify_url", func(t *testing.T) {
		req := validRequest
		req.NotifyURL = ""
		err := req.Validate()
		assert.Equal(t, ErrPaymentInvalidNotifyURL, err)
	})
}

func TestQueryPaymentRequest_Validate(t *testing.T) {
	t.Run("valid_with_gateway_order_id", func(t *testing.T) {
		req := QueryPaymentRequest{
			TenantID:       uuid.New(),
			GatewayOrderID: "wx_pay_123",
			GatewayType:    PaymentGatewayTypeWechat,
		}
		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("valid_with_order_id", func(t *testing.T) {
		req := QueryPaymentRequest{
			TenantID:    uuid.New(),
			OrderID:     uuid.New(),
			GatewayType: PaymentGatewayTypeAlipay,
		}
		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("valid_with_order_number", func(t *testing.T) {
		req := QueryPaymentRequest{
			TenantID:    uuid.New(),
			OrderNumber: "SO-2024-001",
			GatewayType: PaymentGatewayTypeWechat,
		}
		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing_tenant_id", func(t *testing.T) {
		req := QueryPaymentRequest{
			TenantID:       uuid.Nil,
			GatewayOrderID: "wx_pay_123",
			GatewayType:    PaymentGatewayTypeWechat,
		}
		err := req.Validate()
		assert.Equal(t, ErrPaymentInvalidTenantID, err)
	})

	t.Run("missing_all_identifiers", func(t *testing.T) {
		req := QueryPaymentRequest{
			TenantID:    uuid.New(),
			GatewayType: PaymentGatewayTypeWechat,
		}
		err := req.Validate()
		assert.Equal(t, ErrPaymentInvalidQueryParams, err)
	})

	t.Run("invalid_gateway_type", func(t *testing.T) {
		req := QueryPaymentRequest{
			TenantID:       uuid.New(),
			GatewayOrderID: "wx_pay_123",
			GatewayType:    "INVALID",
		}
		err := req.Validate()
		assert.Equal(t, ErrPaymentInvalidGatewayType, err)
	})
}

func TestRefundRequest_Validate(t *testing.T) {
	validRequest := RefundRequest{
		TenantID:       uuid.New(),
		GatewayOrderID: "wx_pay_123",
		RefundID:       uuid.New(),
		TotalAmount:    decimal.NewFromFloat(100.00),
		RefundAmount:   decimal.NewFromFloat(50.00),
		GatewayType:    PaymentGatewayTypeWechat,
	}

	t.Run("valid_request", func(t *testing.T) {
		err := validRequest.Validate()
		assert.NoError(t, err)
	})

	t.Run("valid_with_transaction_id", func(t *testing.T) {
		req := validRequest
		req.GatewayOrderID = ""
		req.GatewayTransactionID = "tx_123"
		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing_tenant_id", func(t *testing.T) {
		req := validRequest
		req.TenantID = uuid.Nil
		err := req.Validate()
		assert.Equal(t, ErrPaymentInvalidTenantID, err)
	})

	t.Run("missing_original_payment_reference", func(t *testing.T) {
		req := validRequest
		req.GatewayOrderID = ""
		req.GatewayTransactionID = ""
		err := req.Validate()
		assert.Equal(t, ErrRefundInvalidOriginalPayment, err)
	})

	t.Run("missing_refund_id", func(t *testing.T) {
		req := validRequest
		req.RefundID = uuid.Nil
		err := req.Validate()
		assert.Equal(t, ErrRefundInvalidRefundID, err)
	})

	t.Run("invalid_total_amount", func(t *testing.T) {
		req := validRequest
		req.TotalAmount = decimal.Zero
		err := req.Validate()
		assert.Equal(t, ErrRefundInvalidTotalAmount, err)
	})

	t.Run("invalid_refund_amount", func(t *testing.T) {
		req := validRequest
		req.RefundAmount = decimal.Zero
		err := req.Validate()
		assert.Equal(t, ErrRefundInvalidAmount, err)
	})

	t.Run("refund_exceeds_total", func(t *testing.T) {
		req := validRequest
		req.RefundAmount = decimal.NewFromFloat(150.00)
		err := req.Validate()
		assert.Equal(t, ErrRefundAmountExceedsTotal, err)
	})

	t.Run("invalid_gateway_type", func(t *testing.T) {
		req := validRequest
		req.GatewayType = "INVALID"
		err := req.Validate()
		assert.Equal(t, ErrPaymentInvalidGatewayType, err)
	})
}

func TestClosePaymentRequest_Validate(t *testing.T) {
	t.Run("valid_with_gateway_order_id", func(t *testing.T) {
		req := ClosePaymentRequest{
			TenantID:       uuid.New(),
			GatewayOrderID: "wx_pay_123",
			GatewayType:    PaymentGatewayTypeWechat,
		}
		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("valid_with_order_number", func(t *testing.T) {
		req := ClosePaymentRequest{
			TenantID:    uuid.New(),
			OrderNumber: "SO-2024-001",
			GatewayType: PaymentGatewayTypeAlipay,
		}
		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing_tenant_id", func(t *testing.T) {
		req := ClosePaymentRequest{
			TenantID:       uuid.Nil,
			GatewayOrderID: "wx_pay_123",
			GatewayType:    PaymentGatewayTypeWechat,
		}
		err := req.Validate()
		assert.Equal(t, ErrPaymentInvalidTenantID, err)
	})

	t.Run("missing_identifiers", func(t *testing.T) {
		req := ClosePaymentRequest{
			TenantID:    uuid.New(),
			GatewayType: PaymentGatewayTypeWechat,
		}
		err := req.Validate()
		assert.Equal(t, ErrPaymentInvalidQueryParams, err)
	})

	t.Run("invalid_gateway_type", func(t *testing.T) {
		req := ClosePaymentRequest{
			TenantID:       uuid.New(),
			GatewayOrderID: "wx_pay_123",
			GatewayType:    "INVALID",
		}
		err := req.Validate()
		assert.Equal(t, ErrPaymentInvalidGatewayType, err)
	})
}
