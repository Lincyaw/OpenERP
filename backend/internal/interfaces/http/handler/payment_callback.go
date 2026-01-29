package handler

import (
	"io"
	"net/http"

	financeapp "github.com/erp/backend/internal/application/finance"
	"github.com/erp/backend/internal/domain/finance"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/gin-gonic/gin"
)

// PaymentCallbackHandler handles payment gateway callback endpoints
// These endpoints are called by external payment gateways (WeChat Pay, Alipay)
// and do not require authentication
type PaymentCallbackHandler struct {
	BaseHandler
	callbackService *financeapp.PaymentCallbackService
}

// NewPaymentCallbackHandler creates a new PaymentCallbackHandler
func NewPaymentCallbackHandler(callbackService *financeapp.PaymentCallbackService) *PaymentCallbackHandler {
	return &PaymentCallbackHandler{
		callbackService: callbackService,
	}
}

// PaymentCallbackResponse represents the response for payment callback status
//
//	@Description	Payment callback status response
type PaymentCallbackResponse struct {
	Success          bool   `json:"success" example:"true"`
	AlreadyProcessed bool   `json:"already_processed,omitempty" example:"false"`
	Message          string `json:"message,omitempty" example:"Payment processed successfully"`
}

// HandleWechatPaymentCallback godoc
//
//	@ID				handleWechatPaymentCallbackPaymentCallback
//	@Summary		Handle WeChat Pay payment callback
//	@Description	Receive and process payment notification from WeChat Pay
//	@Tags			payment-callbacks
//	@Accept			json
//	@Produce		json
//	@Param			Wechatpay-Signature	header		string				true	"WeChat Pay signature"
//	@Param			Wechatpay-Timestamp	header		string				true	"Timestamp"
//	@Param			Wechatpay-Nonce		header		string				true	"Nonce"
//	@Param			Wechatpay-Serial	header		string				true	"Certificate serial number"
//	@Success		200					{object}	map[string]string	"code=SUCCESS"
//	@Failure		500					{object}	map[string]string	"code=FAIL"
//	@Router			/payment/callback/wechat [post]
func (h *PaymentCallbackHandler) HandleWechatPaymentCallback(c *gin.Context) {
	h.handlePaymentCallback(c, finance.PaymentGatewayTypeWechat)
}

// HandleAlipayPaymentCallback godoc
//
//	@ID				handleAlipayPaymentCallbackPaymentCallback
//	@Summary		Handle Alipay payment callback
//	@Description	Receive and process payment notification from Alipay
//	@Tags			payment-callbacks
//	@Accept			application/x-www-form-urlencoded
//	@Produce		text/plain
//	@Success		200	{string}	string	"success"
//	@Failure		500	{string}	string	"fail"
//	@Router			/payment/callback/alipay [post]
func (h *PaymentCallbackHandler) HandleAlipayPaymentCallback(c *gin.Context) {
	h.handlePaymentCallback(c, finance.PaymentGatewayTypeAlipay)
}

// HandleWechatRefundCallback godoc
//
//	@ID				handleWechatRefundCallbackPaymentCallback
//	@Summary		Handle WeChat Pay refund callback
//	@Description	Receive and process refund notification from WeChat Pay
//	@Tags			payment-callbacks
//	@Accept			json
//	@Produce		json
//	@Param			Wechatpay-Signature	header		string				true	"WeChat Pay signature"
//	@Param			Wechatpay-Timestamp	header		string				true	"Timestamp"
//	@Param			Wechatpay-Nonce		header		string				true	"Nonce"
//	@Param			Wechatpay-Serial	header		string				true	"Certificate serial number"
//	@Success		200					{object}	map[string]string	"code=SUCCESS"
//	@Failure		500					{object}	map[string]string	"code=FAIL"
//	@Router			/payment/callback/wechat/refund [post]
func (h *PaymentCallbackHandler) HandleWechatRefundCallback(c *gin.Context) {
	h.handleRefundCallback(c, finance.PaymentGatewayTypeWechat)
}

// HandleAlipayRefundCallback godoc
//
//	@ID				handleAlipayRefundCallbackPaymentCallback
//	@Summary		Handle Alipay refund callback
//	@Description	Receive and process refund notification from Alipay
//	@Tags			payment-callbacks
//	@Accept			application/x-www-form-urlencoded
//	@Produce		text/plain
//	@Success		200	{string}	string	"success"
//	@Failure		500	{string}	string	"fail"
//	@Router			/payment/callback/alipay/refund [post]
func (h *PaymentCallbackHandler) HandleAlipayRefundCallback(c *gin.Context) {
	h.handleRefundCallback(c, finance.PaymentGatewayTypeAlipay)
}

// handlePaymentCallback is the internal handler for payment callbacks
func (h *PaymentCallbackHandler) handlePaymentCallback(c *gin.Context, gatewayType finance.PaymentGatewayType) {
	// Read the raw request body
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.respondToGateway(c, gatewayType, false, "Failed to read request body")
		return
	}

	// Get signature from headers (varies by gateway)
	signature := h.extractSignature(c, gatewayType)

	// Process the callback
	result, err := h.callbackService.ProcessPaymentCallback(
		c.Request.Context(),
		gatewayType,
		payload,
		signature,
	)

	if err != nil {
		h.respondToGateway(c, gatewayType, false, err.Error())
		return
	}

	// Respond with gateway-specific format
	if result != nil && result.GatewayResponse != nil {
		c.Data(http.StatusOK, h.getContentType(gatewayType), result.GatewayResponse)
		return
	}

	h.respondToGateway(c, gatewayType, true, "")
}

// handleRefundCallback is the internal handler for refund callbacks
func (h *PaymentCallbackHandler) handleRefundCallback(c *gin.Context, gatewayType finance.PaymentGatewayType) {
	// Read the raw request body
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.respondToGateway(c, gatewayType, false, "Failed to read request body")
		return
	}

	// Get signature from headers
	signature := h.extractSignature(c, gatewayType)

	// Process the refund callback
	result, err := h.callbackService.ProcessRefundCallback(
		c.Request.Context(),
		gatewayType,
		payload,
		signature,
	)

	if err != nil {
		h.respondToGateway(c, gatewayType, false, err.Error())
		return
	}

	// Respond with gateway-specific format
	if result != nil && result.GatewayResponse != nil {
		c.Data(http.StatusOK, h.getContentType(gatewayType), result.GatewayResponse)
		return
	}

	h.respondToGateway(c, gatewayType, true, "")
}

// extractSignature extracts the signature from headers based on gateway type
func (h *PaymentCallbackHandler) extractSignature(c *gin.Context, gatewayType finance.PaymentGatewayType) string {
	switch gatewayType {
	case finance.PaymentGatewayTypeWechat:
		// WeChat Pay uses multiple headers for signature verification
		// The signature verification is done in the adapter using these headers
		return c.GetHeader("Wechatpay-Signature")
	case finance.PaymentGatewayTypeAlipay:
		// Alipay signature is in the form data
		return c.PostForm("sign")
	default:
		return ""
	}
}

// getContentType returns the content type for the gateway response
func (h *PaymentCallbackHandler) getContentType(gatewayType finance.PaymentGatewayType) string {
	switch gatewayType {
	case finance.PaymentGatewayTypeWechat:
		return "application/json"
	case finance.PaymentGatewayTypeAlipay:
		return "text/plain"
	default:
		return "text/plain"
	}
}

// respondToGateway sends a response in the format expected by the gateway
func (h *PaymentCallbackHandler) respondToGateway(c *gin.Context, gatewayType finance.PaymentGatewayType, success bool, message string) {
	gateway, err := h.callbackService.GetGateway(gatewayType)
	if err != nil {
		// Gateway not registered, use a generic response
		if success {
			c.String(http.StatusOK, "success")
		} else {
			c.String(http.StatusInternalServerError, "fail")
		}
		return
	}

	response := gateway.GenerateCallbackResponse(success, message)
	c.Data(http.StatusOK, h.getContentType(gatewayType), response)
}

// Suppress unused import warning
var _ = dto.Response{}
