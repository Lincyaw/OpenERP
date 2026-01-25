package payment

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/erp/backend/internal/domain/finance"
)

const (
	alipayGatewayURL        = "https://openapi.alipay.com/gateway.do"
	alipaySandboxGatewayURL = "https://openapi-sandbox.dl.alipaydev.com/gateway.do"
	alipayFormat            = "JSON"
	alipayCharset           = "utf-8"
	alipayVersion           = "1.0"
	alipayTimeLayout        = "2006-01-02 15:04:05"
)

// AlipayAdapter implements PaymentGateway interface for Alipay
type AlipayAdapter struct {
	config     *AlipayConfig
	httpClient *http.Client
}

// NewAlipayAdapter creates a new Alipay adapter
func NewAlipayAdapter(config *AlipayConfig) (*AlipayAdapter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &AlipayAdapter{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// GatewayType returns the gateway type
func (a *AlipayAdapter) GatewayType() finance.PaymentGatewayType {
	return finance.PaymentGatewayTypeAlipay
}

// CreatePayment creates a payment order in Alipay
func (a *AlipayAdapter) CreatePayment(ctx context.Context, req *finance.CreatePaymentRequest) (*finance.CreatePaymentResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Determine the method and product code based on channel
	var method, productCode string
	switch req.Channel {
	case finance.PaymentChannelAlipayPage:
		method = alipayMethodPagePay
		productCode = alipayProductCodeFastInstantTradePay
	case finance.PaymentChannelAlipayWap:
		method = alipayMethodWapPay
		productCode = alipayProductCodeQuickWapWay
	case finance.PaymentChannelAlipayApp:
		method = alipayMethodAppPay
		productCode = alipayProductCodeQuickMSecPay
	case finance.PaymentChannelAlipayQRCode:
		method = alipayMethodPrecreate
		productCode = alipayProductCodeFaceToFacePayment
	case finance.PaymentChannelAlipayFaceToFace:
		method = alipayMethodPrecreate
		productCode = alipayProductCodeFaceToFacePayment
	default:
		return nil, finance.ErrPaymentInvalidChannel
	}

	// Build biz_content
	bizContent := alipayBizContent{
		OutTradeNo:  req.OrderNumber,
		ProductCode: productCode,
		TotalAmount: req.Amount.StringFixed(2),
		Subject:     req.Subject,
	}

	if req.Description != "" {
		bizContent.Body = req.Description
	}

	// Set expiration time
	if !req.ExpireTime.IsZero() {
		bizContent.TimeExpire = req.ExpireTime.Format(alipayTimeLayout)
	}

	// Add metadata as passback_params
	if len(req.Metadata) > 0 {
		if metaBytes, err := json.Marshal(req.Metadata); err == nil {
			bizContent.PassbackParams = url.QueryEscape(string(metaBytes))
		}
	}

	// For WAP pay, set quit_url
	if req.Channel == finance.PaymentChannelAlipayWap && req.ReturnURL != "" {
		bizContent.QuitURL = req.ReturnURL
	}

	// Build common parameters
	params := a.buildCommonParams(method)

	// Set notify URL
	if req.NotifyURL != "" {
		params["notify_url"] = req.NotifyURL
	} else {
		params["notify_url"] = a.config.NotifyURL
	}

	// Set return URL for web payments
	if req.ReturnURL != "" && (req.Channel == finance.PaymentChannelAlipayPage || req.Channel == finance.PaymentChannelAlipayWap) {
		params["return_url"] = req.ReturnURL
	} else if a.config.ReturnURL != "" {
		params["return_url"] = a.config.ReturnURL
	}

	// Set biz_content
	bizContentBytes, err := json.Marshal(bizContent)
	if err != nil {
		return nil, fmt.Errorf("alipay: failed to marshal biz_content: %w", err)
	}
	params["biz_content"] = string(bizContentBytes)

	// Sign the request
	sign, err := a.sign(params)
	if err != nil {
		return nil, fmt.Errorf("alipay: failed to sign request: %w", err)
	}
	params["sign"] = sign

	// Create response
	response := &finance.CreatePaymentResponse{
		GatewayType:    finance.PaymentGatewayTypeAlipay,
		Status:         finance.GatewayPaymentStatusPending,
		GatewayOrderID: req.OrderNumber, // Alipay uses our order number as reference
		ExpireTime:     req.ExpireTime,
	}

	// Handle different payment channels
	switch req.Channel {
	case finance.PaymentChannelAlipayPage, finance.PaymentChannelAlipayWap:
		// For web payments, return a payment URL
		paymentURL := a.buildPaymentURL(params)
		response.PaymentURL = paymentURL
		response.RawResponse = paymentURL

	case finance.PaymentChannelAlipayApp:
		// For app payment, return the signed order string
		orderString := a.buildOrderString(params)
		response.SDKParams = orderString
		response.RawResponse = orderString

	case finance.PaymentChannelAlipayQRCode, finance.PaymentChannelAlipayFaceToFace:
		// For QR code payment, need to call API and get QR code
		respBody, err := a.doRequest(ctx, params)
		if err != nil {
			return nil, err
		}

		var precreateResp alipayTradePrecreateResponse
		if err := json.Unmarshal(respBody, &precreateResp); err != nil {
			return nil, fmt.Errorf("alipay: failed to parse response: %w", err)
		}

		if !precreateResp.Response.IsSuccess() {
			return nil, fmt.Errorf("%w: %s - %s", finance.ErrGatewayRequestFailed,
				precreateResp.Response.SubCode, precreateResp.Response.SubMsg)
		}

		response.QRCodeURL = precreateResp.Response.QRCode
		response.QRCodeData = precreateResp.Response.QRCode
		response.RawResponse = string(respBody)
	}

	return response, nil
}

// QueryPayment queries payment status from Alipay
func (a *AlipayAdapter) QueryPayment(ctx context.Context, req *finance.QueryPaymentRequest) (*finance.QueryPaymentResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Determine the order number to query
	orderNo := req.OrderNumber
	if orderNo == "" && req.GatewayOrderID != "" {
		orderNo = req.GatewayOrderID
	}
	if orderNo == "" {
		return nil, finance.ErrPaymentInvalidQueryParams
	}

	// Build biz_content
	bizContent := alipayBizContent{
		OutTradeNo: orderNo,
	}

	// Build common parameters
	params := a.buildCommonParams(alipayMethodQuery)

	// Set biz_content
	bizContentBytes, err := json.Marshal(bizContent)
	if err != nil {
		return nil, fmt.Errorf("alipay: failed to marshal biz_content: %w", err)
	}
	params["biz_content"] = string(bizContentBytes)

	// Sign the request
	sign, err := a.sign(params)
	if err != nil {
		return nil, fmt.Errorf("alipay: failed to sign request: %w", err)
	}
	params["sign"] = sign

	// Make API call
	respBody, err := a.doRequest(ctx, params)
	if err != nil {
		return nil, err
	}

	// Parse response
	var queryResp alipayTradeQueryResponse
	if err := json.Unmarshal(respBody, &queryResp); err != nil {
		return nil, fmt.Errorf("alipay: failed to parse response: %w", err)
	}

	if !queryResp.Response.IsSuccess() {
		if queryResp.Response.SubCode == "ACQ.TRADE_NOT_EXIST" {
			return nil, finance.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("%w: %s - %s", finance.ErrGatewayRequestFailed,
			queryResp.Response.SubCode, queryResp.Response.SubMsg)
	}

	response := &finance.QueryPaymentResponse{
		GatewayOrderID:       queryResp.Response.OutTradeNo,
		GatewayType:          finance.PaymentGatewayTypeAlipay,
		OrderNumber:          queryResp.Response.OutTradeNo,
		Status:               mapAlipayTradeStatus(queryResp.Response.TradeStatus),
		GatewayTransactionID: queryResp.Response.TradeNo,
		PayerAccount:         queryResp.Response.BuyerLogonID,
		RawResponse:          string(respBody),
	}

	// Parse amounts
	if queryResp.Response.TotalAmount != "" {
		if amount, err := decimal.NewFromString(queryResp.Response.TotalAmount); err == nil {
			response.Amount = amount
		}
	}
	if queryResp.Response.BuyerPayAmount != "" {
		if amount, err := decimal.NewFromString(queryResp.Response.BuyerPayAmount); err == nil {
			response.PaidAmount = amount
		}
	} else {
		response.PaidAmount = response.Amount
	}

	response.Currency = "CNY"

	// Parse payment time
	if queryResp.Response.SendPayDate != "" {
		if t, err := time.Parse(alipayTimeLayout, queryResp.Response.SendPayDate); err == nil {
			response.PaidAt = &t
		}
	}

	return response, nil
}

// ClosePayment closes a pending payment order
func (a *AlipayAdapter) ClosePayment(ctx context.Context, req *finance.ClosePaymentRequest) (*finance.ClosePaymentResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	orderNo := req.OrderNumber
	if orderNo == "" {
		orderNo = req.GatewayOrderID
	}

	// Build biz_content
	bizContent := alipayBizContent{
		OutTradeNo: orderNo,
	}

	// Build common parameters
	params := a.buildCommonParams(alipayMethodClose)

	// Set biz_content
	bizContentBytes, err := json.Marshal(bizContent)
	if err != nil {
		return nil, fmt.Errorf("alipay: failed to marshal biz_content: %w", err)
	}
	params["biz_content"] = string(bizContentBytes)

	// Sign the request
	sign, err := a.sign(params)
	if err != nil {
		return nil, fmt.Errorf("alipay: failed to sign request: %w", err)
	}
	params["sign"] = sign

	// Make API call
	respBody, err := a.doRequest(ctx, params)
	if err != nil {
		return nil, err
	}

	// Parse response
	var closeResp alipayTradeCloseResponse
	if err := json.Unmarshal(respBody, &closeResp); err != nil {
		return nil, fmt.Errorf("alipay: failed to parse response: %w", err)
	}

	if !closeResp.Response.IsSuccess() {
		return nil, fmt.Errorf("%w: %s - %s", finance.ErrGatewayRequestFailed,
			closeResp.Response.SubCode, closeResp.Response.SubMsg)
	}

	return &finance.ClosePaymentResponse{
		GatewayOrderID: orderNo,
		Success:        true,
		RawResponse:    string(respBody),
	}, nil
}

// CreateRefund creates a refund request
func (a *AlipayAdapter) CreateRefund(ctx context.Context, req *finance.RefundRequest) (*finance.RefundResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Build biz_content
	bizContent := alipayBizContent{
		OutTradeNo:   req.GatewayOrderID,
		RefundAmount: req.RefundAmount.StringFixed(2),
		OutRequestNo: req.RefundNumber,
	}

	if req.GatewayTransactionID != "" {
		bizContent.TradeNo = req.GatewayTransactionID
	}

	if req.Reason != "" {
		bizContent.RefundReason = req.Reason
	}

	// Build common parameters
	params := a.buildCommonParams(alipayMethodRefund)

	// Set biz_content
	bizContentBytes, err := json.Marshal(bizContent)
	if err != nil {
		return nil, fmt.Errorf("alipay: failed to marshal biz_content: %w", err)
	}
	params["biz_content"] = string(bizContentBytes)

	// Sign the request
	sign, err := a.sign(params)
	if err != nil {
		return nil, fmt.Errorf("alipay: failed to sign request: %w", err)
	}
	params["sign"] = sign

	// Make API call
	respBody, err := a.doRequest(ctx, params)
	if err != nil {
		return nil, err
	}

	// Parse response
	var refundResp alipayTradeRefundResponse
	if err := json.Unmarshal(respBody, &refundResp); err != nil {
		return nil, fmt.Errorf("alipay: failed to parse response: %w", err)
	}

	if !refundResp.Response.IsSuccess() {
		return nil, fmt.Errorf("%w: %s - %s", finance.ErrGatewayRequestFailed,
			refundResp.Response.SubCode, refundResp.Response.SubMsg)
	}

	response := &finance.RefundResponse{
		GatewayRefundID: req.RefundNumber, // Alipay uses out_request_no as refund ID
		GatewayType:     finance.PaymentGatewayTypeAlipay,
		Status:          finance.RefundStatusSuccess, // Alipay refund is synchronous
		RawResponse:     string(respBody),
	}

	// Parse refund amount
	if refundResp.Response.RefundFee != "" {
		if amount, err := decimal.NewFromString(refundResp.Response.RefundFee); err == nil {
			response.RefundAmount = amount
		}
	}

	// Parse refund time
	if refundResp.Response.GmtRefundPay != "" {
		if t, err := time.Parse(alipayTimeLayout, refundResp.Response.GmtRefundPay); err == nil {
			response.RefundedAt = &t
		}
	}

	return response, nil
}

// QueryRefund queries refund status
func (a *AlipayAdapter) QueryRefund(ctx context.Context, tenantID uuid.UUID, gatewayRefundID string) (*finance.RefundResponse, error) {
	if gatewayRefundID == "" {
		return nil, finance.ErrRefundInvalidRefundID
	}

	// Build biz_content
	bizContent := alipayBizContent{
		OutRequestNo: gatewayRefundID,
	}

	// Build common parameters
	params := a.buildCommonParams(alipayMethodRefundQuery)

	// Set biz_content
	bizContentBytes, err := json.Marshal(bizContent)
	if err != nil {
		return nil, fmt.Errorf("alipay: failed to marshal biz_content: %w", err)
	}
	params["biz_content"] = string(bizContentBytes)

	// Sign the request
	sign, err := a.sign(params)
	if err != nil {
		return nil, fmt.Errorf("alipay: failed to sign request: %w", err)
	}
	params["sign"] = sign

	// Make API call
	respBody, err := a.doRequest(ctx, params)
	if err != nil {
		return nil, err
	}

	// Parse response
	var queryResp alipayTradeFastpayRefundQueryResponse
	if err := json.Unmarshal(respBody, &queryResp); err != nil {
		return nil, fmt.Errorf("alipay: failed to parse response: %w", err)
	}

	if !queryResp.Response.IsSuccess() {
		if queryResp.Response.SubCode == "ACQ.TRADE_NOT_EXIST" {
			return nil, finance.ErrRefundNotFound
		}
		return nil, fmt.Errorf("%w: %s - %s", finance.ErrGatewayRequestFailed,
			queryResp.Response.SubCode, queryResp.Response.SubMsg)
	}

	response := &finance.RefundResponse{
		GatewayRefundID: queryResp.Response.OutRequestNo,
		GatewayType:     finance.PaymentGatewayTypeAlipay,
		RawResponse:     string(respBody),
	}

	// Parse refund status
	if queryResp.Response.RefundStatus == alipayRefundStatusRefundSuccess {
		response.Status = finance.RefundStatusSuccess
	} else if queryResp.Response.RefundAmount != "" {
		// If there's a refund amount, consider it successful
		response.Status = finance.RefundStatusSuccess
	} else {
		response.Status = finance.RefundStatusPending
	}

	// Parse refund amount
	if queryResp.Response.RefundAmount != "" {
		if amount, err := decimal.NewFromString(queryResp.Response.RefundAmount); err == nil {
			response.RefundAmount = amount
		}
	}

	// Parse refund time
	if queryResp.Response.GmtRefundPay != "" {
		if t, err := time.Parse(alipayTimeLayout, queryResp.Response.GmtRefundPay); err == nil {
			response.RefundedAt = &t
		}
	}

	return response, nil
}

// VerifyCallback verifies and parses a payment callback
func (a *AlipayAdapter) VerifyCallback(ctx context.Context, payload []byte, signature string) (*finance.PaymentCallback, error) {
	// Parse the notification (URL-encoded form data)
	values, err := url.ParseQuery(string(payload))
	if err != nil {
		return nil, fmt.Errorf("alipay: failed to parse notification: %w", err)
	}

	// Get signature from values if not provided
	if signature == "" {
		signature = values.Get("sign")
	}

	// Verify signature
	if !a.verifySign(values, signature) {
		return nil, finance.ErrGatewayInvalidCallback
	}

	// Build callback from notification
	notification := parseAlipayNotification(values)

	callback := &finance.PaymentCallback{
		GatewayType:          finance.PaymentGatewayTypeAlipay,
		GatewayOrderID:       notification.OutTradeNo,
		GatewayTransactionID: notification.TradeNo,
		OrderNumber:          notification.OutTradeNo,
		Status:               mapAlipayTradeStatus(notification.TradeStatus),
		Currency:             "CNY",
		PayerAccount:         notification.BuyerLogonID,
		RawPayload:           string(payload),
		Signature:            signature,
	}

	// Parse amounts
	if notification.TotalAmount != "" {
		if amount, err := decimal.NewFromString(notification.TotalAmount); err == nil {
			callback.Amount = amount
		}
	}
	if notification.BuyerPayAmount != "" {
		if amount, err := decimal.NewFromString(notification.BuyerPayAmount); err == nil {
			callback.PaidAmount = amount
		}
	} else {
		callback.PaidAmount = callback.Amount
	}

	// Parse payment time
	if notification.GmtPayment != "" {
		if t, err := time.Parse(alipayTimeLayout, notification.GmtPayment); err == nil {
			callback.PaidAt = &t
		}
	}

	return callback, nil
}

// VerifyRefundCallback verifies and parses a refund callback
func (a *AlipayAdapter) VerifyRefundCallback(ctx context.Context, payload []byte, signature string) (*finance.RefundCallback, error) {
	// Parse the notification (URL-encoded form data)
	values, err := url.ParseQuery(string(payload))
	if err != nil {
		return nil, fmt.Errorf("alipay: failed to parse notification: %w", err)
	}

	// Get signature from values if not provided
	if signature == "" {
		signature = values.Get("sign")
	}

	// Verify signature
	if !a.verifySign(values, signature) {
		return nil, finance.ErrGatewayInvalidCallback
	}

	// Build callback from notification
	notification := parseAlipayNotification(values)

	callback := &finance.RefundCallback{
		GatewayType:          finance.PaymentGatewayTypeAlipay,
		GatewayRefundID:      notification.OutRequestNo,
		GatewayOrderID:       notification.OutTradeNo,
		GatewayTransactionID: notification.TradeNo,
		RefundNumber:         notification.OutRequestNo,
		RawPayload:           string(payload),
		Signature:            signature,
	}

	// Determine refund status
	if notification.RefundStatus == alipayRefundStatusRefundSuccess {
		callback.Status = finance.RefundStatusSuccess
	} else if notification.TradeStatus == alipayTradeStatusTradeSuccess && notification.RefundFee != "" {
		// Partial refund
		callback.Status = finance.RefundStatusSuccess
	} else {
		callback.Status = finance.RefundStatusPending
	}

	// Parse refund amount
	if notification.RefundFee != "" {
		if amount, err := decimal.NewFromString(notification.RefundFee); err == nil {
			callback.RefundAmount = amount
		}
	}

	// Parse refund time
	if notification.GmtRefund != "" {
		if t, err := time.Parse(alipayTimeLayout, notification.GmtRefund); err == nil {
			callback.RefundedAt = &t
		}
	}

	return callback, nil
}

// GenerateCallbackResponse generates the response for Alipay callback
func (a *AlipayAdapter) GenerateCallbackResponse(success bool, message string) []byte {
	if success {
		return []byte("success")
	}
	return []byte("fail")
}

// buildCommonParams builds common parameters for API requests
func (a *AlipayAdapter) buildCommonParams(method string) map[string]string {
	params := map[string]string{
		"app_id":    a.config.AppID,
		"method":    method,
		"format":    alipayFormat,
		"charset":   alipayCharset,
		"sign_type": a.config.SignType,
		"timestamp": time.Now().Format(alipayTimeLayout),
		"version":   alipayVersion,
	}

	// Add certificate SN if using certificate mode
	if a.config.AppCertSN != "" {
		params["app_cert_sn"] = a.config.AppCertSN
	}
	if a.config.AlipayCertSN != "" {
		params["alipay_root_cert_sn"] = a.config.AlipayCertSN
	}

	return params
}

// sign signs the parameters
func (a *AlipayAdapter) sign(params map[string]string) (string, error) {
	// Build sign string
	signStr := a.buildSignString(params)

	// Sign using SHA256 with RSA (RSA2)
	hash := sha256.Sum256([]byte(signStr))
	signature, err := rsa.SignPKCS1v15(rand.Reader, a.config.PrivateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(signature), nil
}

// verifySign verifies the signature from Alipay
func (a *AlipayAdapter) verifySign(values url.Values, signature string) bool {
	// Build params map excluding sign and sign_type
	params := make(map[string]string)
	for key := range values {
		if key != "sign" && key != "sign_type" {
			params[key] = values.Get(key)
		}
	}

	// Build sign string
	signStr := a.buildSignString(params)

	// Decode signature
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false
	}

	// Verify using SHA256 with RSA (RSA2)
	hash := sha256.Sum256([]byte(signStr))
	err = rsa.VerifyPKCS1v15(a.config.AlipayPublicKey, crypto.SHA256, hash[:], sigBytes)
	return err == nil
}

// buildSignString builds the string to sign
func (a *AlipayAdapter) buildSignString(params map[string]string) string {
	// Sort keys
	keys := make([]string, 0, len(params))
	for key := range params {
		if params[key] != "" && key != "sign" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	// Build string
	var parts []string
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, params[key]))
	}

	return strings.Join(parts, "&")
}

// buildPaymentURL builds the payment URL for web payments
func (a *AlipayAdapter) buildPaymentURL(params map[string]string) string {
	gatewayURL := alipayGatewayURL
	if a.config.IsSandbox {
		gatewayURL = alipaySandboxGatewayURL
	}

	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}

	return gatewayURL + "?" + values.Encode()
}

// buildOrderString builds the order string for app payment
func (a *AlipayAdapter) buildOrderString(params map[string]string) string {
	// Sort keys
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Build string with URL encoding
	var parts []string
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, url.QueryEscape(params[key])))
	}

	return strings.Join(parts, "&")
}

// doRequest performs an HTTP request to Alipay API
func (a *AlipayAdapter) doRequest(ctx context.Context, params map[string]string) ([]byte, error) {
	gatewayURL := alipayGatewayURL
	if a.config.IsSandbox {
		gatewayURL = alipaySandboxGatewayURL
	}

	// Build form data
	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", gatewayURL, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, fmt.Errorf("alipay: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", finance.ErrGatewayUnavailable, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("alipay: failed to read response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: HTTP %d", finance.ErrGatewayRequestFailed, resp.StatusCode)
	}

	return respBody, nil
}

// parseAlipayNotification parses URL values into alipayNotification
func parseAlipayNotification(values url.Values) *alipayNotification {
	return &alipayNotification{
		NotifyTime:     values.Get("notify_time"),
		NotifyType:     values.Get("notify_type"),
		NotifyID:       values.Get("notify_id"),
		AppID:          values.Get("app_id"),
		Charset:        values.Get("charset"),
		Version:        values.Get("version"),
		SignType:       values.Get("sign_type"),
		Sign:           values.Get("sign"),
		TradeNo:        values.Get("trade_no"),
		OutTradeNo:     values.Get("out_trade_no"),
		OutBizNo:       values.Get("out_biz_no"),
		BuyerID:        values.Get("buyer_id"),
		BuyerLogonID:   values.Get("buyer_logon_id"),
		SellerID:       values.Get("seller_id"),
		SellerEmail:    values.Get("seller_email"),
		TradeStatus:    values.Get("trade_status"),
		TotalAmount:    values.Get("total_amount"),
		ReceiptAmount:  values.Get("receipt_amount"),
		InvoiceAmount:  values.Get("invoice_amount"),
		BuyerPayAmount: values.Get("buyer_pay_amount"),
		PointAmount:    values.Get("point_amount"),
		RefundFee:      values.Get("refund_fee"),
		Subject:        values.Get("subject"),
		Body:           values.Get("body"),
		GmtCreate:      values.Get("gmt_create"),
		GmtPayment:     values.Get("gmt_payment"),
		GmtRefund:      values.Get("gmt_refund"),
		GmtClose:       values.Get("gmt_close"),
		FundBillList:   values.Get("fund_bill_list"),
		PassbackParams: values.Get("passback_params"),
		OutRequestNo:   values.Get("out_request_no"),
		RefundStatus:   values.Get("refund_status"),
	}
}

// mapAlipayTradeStatus maps Alipay trade status to our status
func mapAlipayTradeStatus(status string) finance.GatewayPaymentStatus {
	switch status {
	case alipayTradeStatusTradeSuccess:
		return finance.GatewayPaymentStatusPaid
	case alipayTradeStatusTradeFinished:
		return finance.GatewayPaymentStatusPaid
	case alipayTradeStatusTradeClosed:
		return finance.GatewayPaymentStatusClosed
	case alipayTradeStatusWaitBuyerPay:
		return finance.GatewayPaymentStatusPending
	default:
		return finance.GatewayPaymentStatusPending
	}
}

// Ensure AlipayAdapter implements PaymentGateway interface
var _ finance.PaymentGateway = (*AlipayAdapter)(nil)
