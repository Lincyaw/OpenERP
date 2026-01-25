package payment

import (
	"bytes"
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/erp/backend/internal/domain/finance"
)

const (
	wechatAPIBaseURL        = "https://api.mch.weixin.qq.com"
	wechatSandboxAPIBaseURL = "https://api.mch.weixin.qq.com/sandboxnew"
	wechatNativePayPath     = "/v3/pay/transactions/native"
	wechatJSAPIPayPath      = "/v3/pay/transactions/jsapi"
	wechatAppPayPath        = "/v3/pay/transactions/app"
	wechatQueryPayPath      = "/v3/pay/transactions/out-trade-no/%s"
	wechatClosePayPath      = "/v3/pay/transactions/out-trade-no/%s/close"
	wechatRefundPath        = "/v3/refund/domestic/refunds"
	wechatQueryRefundPath   = "/v3/refund/domestic/refunds/%s"
)

// WechatPayAdapter implements PaymentGateway interface for WeChat Pay
type WechatPayAdapter struct {
	config     *WechatPayConfig
	httpClient *http.Client
}

// NewWechatPayAdapter creates a new WeChat Pay adapter
func NewWechatPayAdapter(config *WechatPayConfig) (*WechatPayAdapter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &WechatPayAdapter{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// GatewayType returns the gateway type
func (a *WechatPayAdapter) GatewayType() finance.PaymentGatewayType {
	return finance.PaymentGatewayTypeWechat
}

// CreatePayment creates a payment order in WeChat Pay
func (a *WechatPayAdapter) CreatePayment(ctx context.Context, req *finance.CreatePaymentRequest) (*finance.CreatePaymentResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Determine the correct API path based on channel
	var path string
	switch req.Channel {
	case finance.PaymentChannelWechatNative:
		path = wechatNativePayPath
	case finance.PaymentChannelWechatJSAPI:
		path = wechatJSAPIPayPath
	case finance.PaymentChannelWechatApp:
		path = wechatAppPayPath
	default:
		return nil, finance.ErrPaymentInvalidChannel
	}

	// Build request body
	body := a.buildCreatePaymentBody(req)
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("wechat: failed to marshal request: %w", err)
	}

	// Make API call
	respBody, err := a.doRequest(ctx, "POST", path, bodyBytes)
	if err != nil {
		return nil, err
	}

	// Parse response based on channel
	response := &finance.CreatePaymentResponse{
		GatewayType: finance.PaymentGatewayTypeWechat,
		Status:      finance.GatewayPaymentStatusPending,
		RawResponse: string(respBody),
	}

	var respData map[string]interface{}
	if err := json.Unmarshal(respBody, &respData); err != nil {
		return nil, fmt.Errorf("wechat: failed to parse response: %w", err)
	}

	switch req.Channel {
	case finance.PaymentChannelWechatNative:
		// Native payment returns a code_url for QR code
		if codeURL, ok := respData["code_url"].(string); ok {
			response.QRCodeURL = codeURL
			response.QRCodeData = codeURL
		}
	case finance.PaymentChannelWechatJSAPI:
		// JSAPI returns a prepay_id
		if prepayID, ok := respData["prepay_id"].(string); ok {
			response.PrepayID = prepayID
			// Generate SDK params for frontend
			sdkParams := a.generateJSAPIParams(prepayID)
			if sdkBytes, err := json.Marshal(sdkParams); err == nil {
				response.SDKParams = string(sdkBytes)
			}
		}
	case finance.PaymentChannelWechatApp:
		// App payment returns a prepay_id
		if prepayID, ok := respData["prepay_id"].(string); ok {
			response.PrepayID = prepayID
			// Generate SDK params for app
			sdkParams := a.generateAppParams(prepayID)
			if sdkBytes, err := json.Marshal(sdkParams); err == nil {
				response.SDKParams = string(sdkBytes)
			}
		}
	}

	response.ExpireTime = req.ExpireTime
	response.GatewayOrderID = req.OrderNumber // WeChat uses our order number as reference

	return response, nil
}

// QueryPayment queries payment status from WeChat Pay
func (a *WechatPayAdapter) QueryPayment(ctx context.Context, req *finance.QueryPaymentRequest) (*finance.QueryPaymentResponse, error) {
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

	path := fmt.Sprintf(wechatQueryPayPath, orderNo) + "?mchid=" + a.config.MchID

	respBody, err := a.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	// Parse response
	var respData wechatQueryResponse
	if err := json.Unmarshal(respBody, &respData); err != nil {
		return nil, fmt.Errorf("wechat: failed to parse response: %w", err)
	}

	response := &finance.QueryPaymentResponse{
		GatewayOrderID:       respData.OutTradeNo,
		GatewayType:          finance.PaymentGatewayTypeWechat,
		OrderNumber:          respData.OutTradeNo,
		Status:               mapWechatTradeState(respData.TradeState),
		GatewayTransactionID: respData.TransactionID,
		RawResponse:          string(respBody),
	}

	// Parse amount
	if respData.Amount != nil {
		// WeChat returns amount in cents
		response.Amount = decimal.NewFromInt(int64(respData.Amount.Total)).Div(decimal.NewFromInt(100))
		response.PaidAmount = decimal.NewFromInt(int64(respData.Amount.PayerTotal)).Div(decimal.NewFromInt(100))
		response.Currency = respData.Amount.Currency
	}

	// Parse payer
	if respData.Payer != nil {
		response.PayerAccount = respData.Payer.OpenID
	}

	// Parse success time
	if respData.SuccessTime != "" {
		if t, err := time.Parse(time.RFC3339, respData.SuccessTime); err == nil {
			response.PaidAt = &t
		}
	}

	return response, nil
}

// ClosePayment closes a pending payment order
func (a *WechatPayAdapter) ClosePayment(ctx context.Context, req *finance.ClosePaymentRequest) (*finance.ClosePaymentResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	orderNo := req.OrderNumber
	if orderNo == "" {
		orderNo = req.GatewayOrderID
	}

	path := fmt.Sprintf(wechatClosePayPath, orderNo)

	body := map[string]string{"mchid": a.config.MchID}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("wechat: failed to marshal request: %w", err)
	}

	_, err = a.doRequest(ctx, "POST", path, bodyBytes)
	if err != nil {
		// WeChat returns 204 No Content on success
		if !strings.Contains(err.Error(), "204") {
			return nil, err
		}
	}

	return &finance.ClosePaymentResponse{
		GatewayOrderID: orderNo,
		Success:        true,
	}, nil
}

// CreateRefund creates a refund request
func (a *WechatPayAdapter) CreateRefund(ctx context.Context, req *finance.RefundRequest) (*finance.RefundResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	body := wechatRefundRequest{
		OutTradeNo:  req.GatewayOrderID,
		OutRefundNo: req.RefundNumber,
		Reason:      req.Reason,
		NotifyURL:   a.config.RefundNotifyURL,
		Amount: wechatRefundAmount{
			Refund:   int(req.RefundAmount.Mul(decimal.NewFromInt(100)).IntPart()),
			Total:    int(req.TotalAmount.Mul(decimal.NewFromInt(100)).IntPart()),
			Currency: "CNY",
		},
	}

	if req.GatewayTransactionID != "" {
		body.TransactionID = req.GatewayTransactionID
	}

	if req.NotifyURL != "" {
		body.NotifyURL = req.NotifyURL
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("wechat: failed to marshal request: %w", err)
	}

	respBody, err := a.doRequest(ctx, "POST", wechatRefundPath, bodyBytes)
	if err != nil {
		return nil, err
	}

	var respData wechatRefundResponse
	if err := json.Unmarshal(respBody, &respData); err != nil {
		return nil, fmt.Errorf("wechat: failed to parse response: %w", err)
	}

	response := &finance.RefundResponse{
		GatewayRefundID: respData.RefundID,
		GatewayType:     finance.PaymentGatewayTypeWechat,
		Status:          mapWechatRefundStatus(respData.Status),
		RefundAmount:    decimal.NewFromInt(int64(respData.Amount.Refund)).Div(decimal.NewFromInt(100)),
		RawResponse:     string(respBody),
	}

	if respData.SuccessTime != "" {
		if t, err := time.Parse(time.RFC3339, respData.SuccessTime); err == nil {
			response.RefundedAt = &t
		}
	}

	return response, nil
}

// QueryRefund queries refund status
func (a *WechatPayAdapter) QueryRefund(ctx context.Context, tenantID uuid.UUID, gatewayRefundID string) (*finance.RefundResponse, error) {
	if gatewayRefundID == "" {
		return nil, finance.ErrRefundInvalidRefundID
	}

	path := fmt.Sprintf(wechatQueryRefundPath, gatewayRefundID)

	respBody, err := a.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var respData wechatRefundResponse
	if err := json.Unmarshal(respBody, &respData); err != nil {
		return nil, fmt.Errorf("wechat: failed to parse response: %w", err)
	}

	response := &finance.RefundResponse{
		GatewayRefundID: respData.RefundID,
		GatewayType:     finance.PaymentGatewayTypeWechat,
		Status:          mapWechatRefundStatus(respData.Status),
		RefundAmount:    decimal.NewFromInt(int64(respData.Amount.Refund)).Div(decimal.NewFromInt(100)),
		RawResponse:     string(respBody),
	}

	if respData.SuccessTime != "" {
		if t, err := time.Parse(time.RFC3339, respData.SuccessTime); err == nil {
			response.RefundedAt = &t
		}
	}

	return response, nil
}

// VerifyCallback verifies and parses a payment callback
func (a *WechatPayAdapter) VerifyCallback(ctx context.Context, payload []byte, signature string) (*finance.PaymentCallback, error) {
	// Parse the encrypted notification
	var notification wechatNotification
	if err := json.Unmarshal(payload, &notification); err != nil {
		return nil, fmt.Errorf("wechat: failed to parse notification: %w", err)
	}

	// Decrypt the resource
	decrypted, err := a.decryptResource(&notification.Resource)
	if err != nil {
		return nil, fmt.Errorf("wechat: failed to decrypt resource: %w", err)
	}

	// Parse the decrypted payment data
	var paymentData wechatPaymentNotification
	if err := json.Unmarshal(decrypted, &paymentData); err != nil {
		return nil, fmt.Errorf("wechat: failed to parse payment data: %w", err)
	}

	callback := &finance.PaymentCallback{
		GatewayType:          finance.PaymentGatewayTypeWechat,
		GatewayOrderID:       paymentData.OutTradeNo,
		GatewayTransactionID: paymentData.TransactionID,
		OrderNumber:          paymentData.OutTradeNo,
		Status:               mapWechatTradeState(paymentData.TradeState),
		Currency:             paymentData.Amount.Currency,
		RawPayload:           string(payload),
		Signature:            signature,
	}

	// Parse amount
	callback.Amount = decimal.NewFromInt(int64(paymentData.Amount.Total)).Div(decimal.NewFromInt(100))
	callback.PaidAmount = decimal.NewFromInt(int64(paymentData.Amount.PayerTotal)).Div(decimal.NewFromInt(100))

	// Parse payer
	if paymentData.Payer != nil {
		callback.PayerAccount = paymentData.Payer.OpenID
	}

	// Parse success time
	if paymentData.SuccessTime != "" {
		if t, err := time.Parse(time.RFC3339, paymentData.SuccessTime); err == nil {
			callback.PaidAt = &t
		}
	}

	return callback, nil
}

// VerifyRefundCallback verifies and parses a refund callback
func (a *WechatPayAdapter) VerifyRefundCallback(ctx context.Context, payload []byte, signature string) (*finance.RefundCallback, error) {
	// Parse the encrypted notification
	var notification wechatNotification
	if err := json.Unmarshal(payload, &notification); err != nil {
		return nil, fmt.Errorf("wechat: failed to parse notification: %w", err)
	}

	// Decrypt the resource
	decrypted, err := a.decryptResource(&notification.Resource)
	if err != nil {
		return nil, fmt.Errorf("wechat: failed to decrypt resource: %w", err)
	}

	// Parse the decrypted refund data
	var refundData wechatRefundNotification
	if err := json.Unmarshal(decrypted, &refundData); err != nil {
		return nil, fmt.Errorf("wechat: failed to parse refund data: %w", err)
	}

	callback := &finance.RefundCallback{
		GatewayType:          finance.PaymentGatewayTypeWechat,
		GatewayRefundID:      refundData.RefundID,
		GatewayOrderID:       refundData.OutTradeNo,
		GatewayTransactionID: refundData.TransactionID,
		RefundNumber:         refundData.OutRefundNo,
		Status:               mapWechatRefundStatus(refundData.RefundStatus),
		RefundAmount:         decimal.NewFromInt(int64(refundData.Amount.Refund)).Div(decimal.NewFromInt(100)),
		RawPayload:           string(payload),
		Signature:            signature,
	}

	// Parse success time
	if refundData.SuccessTime != "" {
		if t, err := time.Parse(time.RFC3339, refundData.SuccessTime); err == nil {
			callback.RefundedAt = &t
		}
	}

	return callback, nil
}

// GenerateCallbackResponse generates the response for WeChat callback
func (a *WechatPayAdapter) GenerateCallbackResponse(success bool, message string) []byte {
	resp := map[string]string{
		"code": "SUCCESS",
	}
	if !success {
		resp["code"] = "FAIL"
		resp["message"] = message
	}

	data, _ := json.Marshal(resp)
	return data
}

// buildCreatePaymentBody builds the request body for creating payment
func (a *WechatPayAdapter) buildCreatePaymentBody(req *finance.CreatePaymentRequest) map[string]interface{} {
	body := map[string]interface{}{
		"appid":        a.config.AppID,
		"mchid":        a.config.MchID,
		"description":  req.Subject,
		"out_trade_no": req.OrderNumber,
		"notify_url":   req.NotifyURL,
		"amount": map[string]interface{}{
			"total":    int(req.Amount.Mul(decimal.NewFromInt(100)).IntPart()), // Convert to cents
			"currency": "CNY",
		},
	}

	if req.NotifyURL == "" {
		body["notify_url"] = a.config.NotifyURL
	}

	// Set expiration time
	if !req.ExpireTime.IsZero() {
		body["time_expire"] = req.ExpireTime.Format(time.RFC3339)
	}

	// Add description if provided
	if req.Description != "" {
		body["description"] = req.Description
	}

	// Add metadata as attach
	if len(req.Metadata) > 0 {
		if attachBytes, err := json.Marshal(req.Metadata); err == nil && len(attachBytes) <= 128 {
			body["attach"] = string(attachBytes)
		}
	}

	// Add scene info if client IP provided
	if req.ClientIP != "" {
		body["scene_info"] = map[string]interface{}{
			"payer_client_ip": req.ClientIP,
		}
	}

	return body
}

// generateJSAPIParams generates parameters for JSAPI payment
func (a *WechatPayAdapter) generateJSAPIParams(prepayID string) map[string]string {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonceStr := generateNonceStr()

	// Build sign string
	signStr := fmt.Sprintf("%s\n%s\n%s\nprepay_id=%s\n",
		a.config.AppID, timestamp, nonceStr, prepayID)

	signature := a.sign(signStr)

	return map[string]string{
		"appId":     a.config.AppID,
		"timeStamp": timestamp,
		"nonceStr":  nonceStr,
		"package":   "prepay_id=" + prepayID,
		"signType":  "RSA",
		"paySign":   signature,
	}
}

// generateAppParams generates parameters for App payment
func (a *WechatPayAdapter) generateAppParams(prepayID string) map[string]string {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonceStr := generateNonceStr()

	// Build sign string
	signStr := fmt.Sprintf("%s\n%s\n%s\n%s\n",
		a.config.AppID, timestamp, nonceStr, prepayID)

	signature := a.sign(signStr)

	return map[string]string{
		"appid":     a.config.AppID,
		"partnerid": a.config.MchID,
		"prepayid":  prepayID,
		"package":   "Sign=WXPay",
		"noncestr":  nonceStr,
		"timestamp": timestamp,
		"sign":      signature,
	}
}

// doRequest performs an HTTP request to WeChat Pay API
func (a *WechatPayAdapter) doRequest(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	baseURL := wechatAPIBaseURL
	if a.config.IsSandbox {
		baseURL = wechatSandboxAPIBaseURL
	}

	url := baseURL + path

	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("wechat: failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Generate authorization header
	auth, err := a.generateAuthHeader(method, path, body)
	if err != nil {
		return nil, fmt.Errorf("wechat: failed to generate auth: %w", err)
	}
	req.Header.Set("Authorization", auth)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", finance.ErrGatewayUnavailable, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("wechat: failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		var errResp wechatErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Code != "" {
			return nil, fmt.Errorf("%w: %s - %s", finance.ErrGatewayRequestFailed, errResp.Code, errResp.Message)
		}
		return nil, fmt.Errorf("%w: HTTP %d", finance.ErrGatewayRequestFailed, resp.StatusCode)
	}

	return respBody, nil
}

// generateAuthHeader generates the Authorization header for WeChat Pay API v3
func (a *WechatPayAdapter) generateAuthHeader(method, path string, body []byte) (string, error) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonceStr := generateNonceStr()

	// Build the message to sign
	var bodyStr string
	if body != nil {
		bodyStr = string(body)
	}

	message := fmt.Sprintf("%s\n%s\n%s\n%s\n",
		method, path, timestamp+"\n"+nonceStr, bodyStr)

	signature := a.sign(message)

	return fmt.Sprintf(`WECHATPAY2-SHA256-RSA2048 mchid="%s",nonce_str="%s",signature="%s",timestamp="%s",serial_no="%s"`,
		a.config.MchID, nonceStr, signature, timestamp, a.config.SerialNo), nil
}

// sign signs the message using RSA-SHA256
func (a *WechatPayAdapter) sign(message string) string {
	hash := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, a.config.PrivateKey, crypto.SHA256, hash[:])
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(signature)
}

// decryptResource decrypts the encrypted resource in callback
func (a *WechatPayAdapter) decryptResource(resource *wechatResource) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(resource.Ciphertext)
	if err != nil {
		return nil, err
	}

	nonce := []byte(resource.Nonce)
	associatedData := []byte(resource.AssociatedData)

	block, err := aes.NewCipher([]byte(a.config.APIKey))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, associatedData)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// generateNonceStr generates a random nonce string
func generateNonceStr() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// mapWechatTradeState maps WeChat trade state to our status
func mapWechatTradeState(state string) finance.GatewayPaymentStatus {
	switch state {
	case "SUCCESS":
		return finance.GatewayPaymentStatusPaid
	case "REFUND":
		return finance.GatewayPaymentStatusRefunded
	case "NOTPAY":
		return finance.GatewayPaymentStatusPending
	case "CLOSED":
		return finance.GatewayPaymentStatusClosed
	case "REVOKED":
		return finance.GatewayPaymentStatusCancelled
	case "USERPAYING":
		return finance.GatewayPaymentStatusPending
	case "PAYERROR":
		return finance.GatewayPaymentStatusFailed
	default:
		return finance.GatewayPaymentStatusPending
	}
}

// mapWechatRefundStatus maps WeChat refund status to our status
func mapWechatRefundStatus(status string) finance.RefundStatus {
	switch status {
	case "SUCCESS":
		return finance.RefundStatusSuccess
	case "CLOSED":
		return finance.RefundStatusClosed
	case "PROCESSING":
		return finance.RefundStatusPending
	case "ABNORMAL":
		return finance.RefundStatusFailed
	default:
		return finance.RefundStatusPending
	}
}

// Ensure WechatPayAdapter implements PaymentGateway interface
var _ finance.PaymentGateway = (*WechatPayAdapter)(nil)
