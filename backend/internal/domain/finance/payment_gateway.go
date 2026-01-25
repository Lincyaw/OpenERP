package finance

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ---------------------------------------------------------------------------
// Payment Gateway Errors
// ---------------------------------------------------------------------------

var (
	// Payment creation errors
	ErrPaymentInvalidTenantID    = errors.New("payment: invalid tenant ID")
	ErrPaymentInvalidOrderID     = errors.New("payment: invalid order ID")
	ErrPaymentInvalidOrderNumber = errors.New("payment: invalid order number")
	ErrPaymentInvalidAmount      = errors.New("payment: invalid payment amount")
	ErrPaymentInvalidChannel     = errors.New("payment: invalid payment channel")
	ErrPaymentInvalidSubject     = errors.New("payment: invalid payment subject")
	ErrPaymentInvalidNotifyURL   = errors.New("payment: invalid notify URL")
	ErrPaymentInvalidReturnURL   = errors.New("payment: invalid return URL")

	// Payment query errors
	ErrPaymentInvalidQueryParams = errors.New("payment: invalid query parameters, need gateway order ID, order ID, or order number")
	ErrPaymentInvalidGatewayType = errors.New("payment: invalid gateway type")
	ErrPaymentNotFound           = errors.New("payment: payment order not found")
	ErrPaymentAlreadyPaid        = errors.New("payment: payment already completed")
	ErrPaymentAlreadyClosed      = errors.New("payment: payment already closed")
	ErrPaymentExpired            = errors.New("payment: payment order expired")

	// Refund errors
	ErrRefundInvalidOriginalPayment = errors.New("refund: invalid original payment reference")
	ErrRefundInvalidRefundID        = errors.New("refund: invalid refund ID")
	ErrRefundInvalidTotalAmount     = errors.New("refund: invalid total amount")
	ErrRefundInvalidAmount          = errors.New("refund: invalid refund amount")
	ErrRefundAmountExceedsTotal     = errors.New("refund: refund amount exceeds total payment")
	ErrRefundNotFound               = errors.New("refund: refund not found")
	ErrRefundNotAllowed             = errors.New("refund: refund not allowed for this payment")

	// Gateway errors
	ErrGatewayNotConfigured   = errors.New("payment: gateway not configured")
	ErrGatewayNotEnabled      = errors.New("payment: gateway not enabled")
	ErrGatewayUnavailable     = errors.New("payment: gateway temporarily unavailable")
	ErrGatewayRequestFailed   = errors.New("payment: gateway request failed")
	ErrGatewayInvalidResponse = errors.New("payment: invalid gateway response")
	ErrGatewayInvalidCallback = errors.New("payment: invalid callback signature")
)

// ---------------------------------------------------------------------------
// PaymentGatewayType represents the type of payment gateway
type PaymentGatewayType string

const (
	// PaymentGatewayTypeWechat represents WeChat Pay gateway
	PaymentGatewayTypeWechat PaymentGatewayType = "WECHAT"
	// PaymentGatewayTypeAlipay represents Alipay gateway
	PaymentGatewayTypeAlipay PaymentGatewayType = "ALIPAY"
)

// IsValid returns true if the gateway type is valid
func (t PaymentGatewayType) IsValid() bool {
	switch t {
	case PaymentGatewayTypeWechat, PaymentGatewayTypeAlipay:
		return true
	default:
		return false
	}
}

// String returns the string representation of PaymentGatewayType
func (t PaymentGatewayType) String() string {
	return string(t)
}

// PaymentChannel represents the specific payment channel within a gateway
type PaymentChannel string

const (
	// WeChat Payment Channels
	PaymentChannelWechatNative PaymentChannel = "WECHAT_NATIVE" // Native QR code payment
	PaymentChannelWechatJSAPI  PaymentChannel = "WECHAT_JSAPI"  // WeChat mini-program/H5 payment
	PaymentChannelWechatApp    PaymentChannel = "WECHAT_APP"    // In-app payment

	// Alipay Payment Channels
	PaymentChannelAlipayPage       PaymentChannel = "ALIPAY_PAGE"   // PC web payment
	PaymentChannelAlipayWap        PaymentChannel = "ALIPAY_WAP"    // Mobile web payment
	PaymentChannelAlipayApp        PaymentChannel = "ALIPAY_APP"    // In-app payment
	PaymentChannelAlipayQRCode     PaymentChannel = "ALIPAY_QRCODE" // QR code payment
	PaymentChannelAlipayFaceToFace PaymentChannel = "ALIPAY_F2F"    // Face-to-face (merchant presents QR)
)

// IsValid returns true if the payment channel is valid
func (c PaymentChannel) IsValid() bool {
	switch c {
	case PaymentChannelWechatNative, PaymentChannelWechatJSAPI, PaymentChannelWechatApp,
		PaymentChannelAlipayPage, PaymentChannelAlipayWap, PaymentChannelAlipayApp,
		PaymentChannelAlipayQRCode, PaymentChannelAlipayFaceToFace:
		return true
	default:
		return false
	}
}

// String returns the string representation of PaymentChannel
func (c PaymentChannel) String() string {
	return string(c)
}

// GetGatewayType returns the gateway type for this channel
func (c PaymentChannel) GetGatewayType() PaymentGatewayType {
	switch c {
	case PaymentChannelWechatNative, PaymentChannelWechatJSAPI, PaymentChannelWechatApp:
		return PaymentGatewayTypeWechat
	case PaymentChannelAlipayPage, PaymentChannelAlipayWap, PaymentChannelAlipayApp,
		PaymentChannelAlipayQRCode, PaymentChannelAlipayFaceToFace:
		return PaymentGatewayTypeAlipay
	default:
		return ""
	}
}

// GatewayPaymentStatus represents the status of a payment in the gateway
type GatewayPaymentStatus string

const (
	// GatewayPaymentStatusPending indicates payment is pending (waiting for user)
	GatewayPaymentStatusPending GatewayPaymentStatus = "PENDING"
	// GatewayPaymentStatusPaid indicates payment was successful
	GatewayPaymentStatusPaid GatewayPaymentStatus = "PAID"
	// GatewayPaymentStatusFailed indicates payment failed
	GatewayPaymentStatusFailed GatewayPaymentStatus = "FAILED"
	// GatewayPaymentStatusCancelled indicates payment was cancelled
	GatewayPaymentStatusCancelled GatewayPaymentStatus = "CANCELLED"
	// GatewayPaymentStatusRefunded indicates payment was refunded (full)
	GatewayPaymentStatusRefunded GatewayPaymentStatus = "REFUNDED"
	// GatewayPaymentStatusPartialRefunded indicates payment was partially refunded
	GatewayPaymentStatusPartialRefunded GatewayPaymentStatus = "PARTIAL_REFUNDED"
	// GatewayPaymentStatusClosed indicates payment order was closed
	GatewayPaymentStatusClosed GatewayPaymentStatus = "CLOSED"
)

// IsValid returns true if the status is valid
func (s GatewayPaymentStatus) IsValid() bool {
	switch s {
	case GatewayPaymentStatusPending, GatewayPaymentStatusPaid, GatewayPaymentStatusFailed,
		GatewayPaymentStatusCancelled, GatewayPaymentStatusRefunded, GatewayPaymentStatusPartialRefunded,
		GatewayPaymentStatusClosed:
		return true
	default:
		return false
	}
}

// String returns the string representation of GatewayPaymentStatus
func (s GatewayPaymentStatus) String() string {
	return string(s)
}

// IsFinal returns true if the status is a final (terminal) state
func (s GatewayPaymentStatus) IsFinal() bool {
	switch s {
	case GatewayPaymentStatusPaid, GatewayPaymentStatusFailed, GatewayPaymentStatusCancelled,
		GatewayPaymentStatusRefunded, GatewayPaymentStatusClosed:
		return true
	default:
		return false
	}
}

// IsSuccess returns true if the payment was successful
func (s GatewayPaymentStatus) IsSuccess() bool {
	return s == GatewayPaymentStatusPaid
}

// ---------------------------------------------------------------------------
// Payment Request/Response DTOs
// ---------------------------------------------------------------------------

// CreatePaymentRequest represents a request to create a payment order
type CreatePaymentRequest struct {
	// TenantID is the tenant making the payment
	TenantID uuid.UUID
	// OrderID is our internal order/reference ID
	OrderID uuid.UUID
	// OrderNumber is our internal order number (for display)
	OrderNumber string
	// Amount is the payment amount
	Amount decimal.Decimal
	// Currency is the payment currency (default: CNY)
	Currency string
	// Channel specifies the payment channel to use
	Channel PaymentChannel
	// Subject is the payment subject/title (shown to user)
	Subject string
	// Description is an optional detailed description
	Description string
	// NotifyURL is the callback URL for payment notifications
	NotifyURL string
	// ReturnURL is the URL to redirect user after payment (for web payments)
	ReturnURL string
	// ExpireTime is when the payment order should expire
	ExpireTime time.Time
	// ClientIP is the payer's IP address
	ClientIP string
	// Metadata is additional key-value data to associate with payment
	Metadata map[string]string
}

// Validate validates the create payment request
func (r *CreatePaymentRequest) Validate() error {
	if r.TenantID == uuid.Nil {
		return ErrPaymentInvalidTenantID
	}
	if r.OrderID == uuid.Nil {
		return ErrPaymentInvalidOrderID
	}
	if r.OrderNumber == "" {
		return ErrPaymentInvalidOrderNumber
	}
	if r.Amount.LessThanOrEqual(decimal.Zero) {
		return ErrPaymentInvalidAmount
	}
	if !r.Channel.IsValid() {
		return ErrPaymentInvalidChannel
	}
	if r.Subject == "" {
		return ErrPaymentInvalidSubject
	}
	if r.NotifyURL == "" {
		return ErrPaymentInvalidNotifyURL
	}
	return nil
}

// CreatePaymentResponse represents the response from creating a payment order
type CreatePaymentResponse struct {
	// GatewayOrderID is the payment order ID in the gateway
	GatewayOrderID string
	// GatewayType identifies which gateway processed this
	GatewayType PaymentGatewayType
	// Status is the initial payment status
	Status GatewayPaymentStatus
	// QRCodeURL is the QR code URL for scanning (for QR-based payments)
	QRCodeURL string
	// QRCodeData is the raw QR code data
	QRCodeData string
	// PaymentURL is the URL to redirect user for payment (for web payments)
	PaymentURL string
	// PrepayID is the prepay ID (for WeChat JSAPI/Mini Program)
	PrepayID string
	// SDKParams contains parameters for native SDK integration (JSON)
	SDKParams string
	// ExpireTime is when this payment order expires
	ExpireTime time.Time
	// RawResponse is the original gateway response (JSON)
	RawResponse string
}

// QueryPaymentRequest represents a request to query payment status
type QueryPaymentRequest struct {
	// TenantID is the tenant who owns the payment
	TenantID uuid.UUID
	// GatewayOrderID is the payment order ID in the gateway
	GatewayOrderID string
	// OrderID is our internal order/reference ID (alternative to GatewayOrderID)
	OrderID uuid.UUID
	// OrderNumber is our internal order number (alternative to GatewayOrderID)
	OrderNumber string
	// GatewayType specifies which gateway to query
	GatewayType PaymentGatewayType
}

// Validate validates the query payment request
func (r *QueryPaymentRequest) Validate() error {
	if r.TenantID == uuid.Nil {
		return ErrPaymentInvalidTenantID
	}
	if r.GatewayOrderID == "" && r.OrderID == uuid.Nil && r.OrderNumber == "" {
		return ErrPaymentInvalidQueryParams
	}
	if !r.GatewayType.IsValid() {
		return ErrPaymentInvalidGatewayType
	}
	return nil
}

// QueryPaymentResponse represents the response from querying payment status
type QueryPaymentResponse struct {
	// GatewayOrderID is the payment order ID in the gateway
	GatewayOrderID string
	// GatewayType identifies which gateway processed this
	GatewayType PaymentGatewayType
	// OrderID is our internal order/reference ID
	OrderID uuid.UUID
	// OrderNumber is our internal order number
	OrderNumber string
	// Status is the current payment status
	Status GatewayPaymentStatus
	// Amount is the payment amount
	Amount decimal.Decimal
	// Currency is the payment currency
	Currency string
	// PaidAmount is the actual amount paid (may differ from Amount for partial payments)
	PaidAmount decimal.Decimal
	// PayerAccount is the payer's account identifier (masked)
	PayerAccount string
	// GatewayTransactionID is the transaction ID from the gateway
	GatewayTransactionID string
	// PaidAt is when the payment was completed
	PaidAt *time.Time
	// RawResponse is the original gateway response (JSON)
	RawResponse string
}

// RefundRequest represents a request to refund a payment
type RefundRequest struct {
	// TenantID is the tenant requesting the refund
	TenantID uuid.UUID
	// GatewayOrderID is the original payment order ID in the gateway
	GatewayOrderID string
	// GatewayTransactionID is the original transaction ID
	GatewayTransactionID string
	// RefundID is our internal refund reference ID
	RefundID uuid.UUID
	// RefundNumber is our internal refund number
	RefundNumber string
	// TotalAmount is the original payment amount
	TotalAmount decimal.Decimal
	// RefundAmount is the amount to refund
	RefundAmount decimal.Decimal
	// Currency is the refund currency (should match original)
	Currency string
	// Reason is the reason for refund
	Reason string
	// NotifyURL is the callback URL for refund notifications
	NotifyURL string
	// GatewayType specifies which gateway to use
	GatewayType PaymentGatewayType
}

// Validate validates the refund request
func (r *RefundRequest) Validate() error {
	if r.TenantID == uuid.Nil {
		return ErrPaymentInvalidTenantID
	}
	if r.GatewayOrderID == "" && r.GatewayTransactionID == "" {
		return ErrRefundInvalidOriginalPayment
	}
	if r.RefundID == uuid.Nil {
		return ErrRefundInvalidRefundID
	}
	if r.TotalAmount.LessThanOrEqual(decimal.Zero) {
		return ErrRefundInvalidTotalAmount
	}
	if r.RefundAmount.LessThanOrEqual(decimal.Zero) {
		return ErrRefundInvalidAmount
	}
	if r.RefundAmount.GreaterThan(r.TotalAmount) {
		return ErrRefundAmountExceedsTotal
	}
	if !r.GatewayType.IsValid() {
		return ErrPaymentInvalidGatewayType
	}
	return nil
}

// RefundResponse represents the response from a refund request
type RefundResponse struct {
	// GatewayRefundID is the refund ID in the gateway
	GatewayRefundID string
	// GatewayType identifies which gateway processed this
	GatewayType PaymentGatewayType
	// Status is the refund status
	Status RefundStatus
	// RefundAmount is the refunded amount
	RefundAmount decimal.Decimal
	// RefundedAt is when the refund was completed
	RefundedAt *time.Time
	// RawResponse is the original gateway response (JSON)
	RawResponse string
}

// RefundStatus represents the status of a refund
type RefundStatus string

const (
	// RefundStatusPending indicates refund is being processed
	RefundStatusPending RefundStatus = "PENDING"
	// RefundStatusSuccess indicates refund was successful
	RefundStatusSuccess RefundStatus = "SUCCESS"
	// RefundStatusFailed indicates refund failed
	RefundStatusFailed RefundStatus = "FAILED"
	// RefundStatusClosed indicates refund was closed/cancelled
	RefundStatusClosed RefundStatus = "CLOSED"
)

// IsValid returns true if the refund status is valid
func (s RefundStatus) IsValid() bool {
	switch s {
	case RefundStatusPending, RefundStatusSuccess, RefundStatusFailed, RefundStatusClosed:
		return true
	default:
		return false
	}
}

// String returns the string representation of RefundStatus
func (s RefundStatus) String() string {
	return string(s)
}

// ClosePaymentRequest represents a request to close/cancel a payment order
type ClosePaymentRequest struct {
	// TenantID is the tenant who owns the payment
	TenantID uuid.UUID
	// GatewayOrderID is the payment order ID in the gateway
	GatewayOrderID string
	// OrderNumber is our internal order number
	OrderNumber string
	// GatewayType specifies which gateway to use
	GatewayType PaymentGatewayType
}

// Validate validates the close payment request
func (r *ClosePaymentRequest) Validate() error {
	if r.TenantID == uuid.Nil {
		return ErrPaymentInvalidTenantID
	}
	if r.GatewayOrderID == "" && r.OrderNumber == "" {
		return ErrPaymentInvalidQueryParams
	}
	if !r.GatewayType.IsValid() {
		return ErrPaymentInvalidGatewayType
	}
	return nil
}

// ClosePaymentResponse represents the response from closing a payment order
type ClosePaymentResponse struct {
	// GatewayOrderID is the payment order ID in the gateway
	GatewayOrderID string
	// Success indicates if the close operation was successful
	Success bool
	// RawResponse is the original gateway response (JSON)
	RawResponse string
}

// ---------------------------------------------------------------------------
// Payment Callback Types
// ---------------------------------------------------------------------------

// PaymentCallback represents a payment notification from the gateway
type PaymentCallback struct {
	// GatewayType identifies which gateway sent this callback
	GatewayType PaymentGatewayType
	// GatewayOrderID is the payment order ID in the gateway
	GatewayOrderID string
	// GatewayTransactionID is the transaction ID from the gateway
	GatewayTransactionID string
	// OrderNumber is our internal order number
	OrderNumber string
	// Status is the payment status
	Status GatewayPaymentStatus
	// Amount is the payment amount
	Amount decimal.Decimal
	// Currency is the payment currency
	Currency string
	// PaidAmount is the actual amount paid
	PaidAmount decimal.Decimal
	// PayerAccount is the payer's account identifier (masked)
	PayerAccount string
	// PaidAt is when the payment was completed
	PaidAt *time.Time
	// RawPayload is the original callback payload
	RawPayload string
	// Signature is the callback signature for verification
	Signature string
}

// RefundCallback represents a refund notification from the gateway
type RefundCallback struct {
	// GatewayType identifies which gateway sent this callback
	GatewayType PaymentGatewayType
	// GatewayRefundID is the refund ID in the gateway
	GatewayRefundID string
	// GatewayOrderID is the original payment order ID
	GatewayOrderID string
	// GatewayTransactionID is the original transaction ID
	GatewayTransactionID string
	// RefundNumber is our internal refund number
	RefundNumber string
	// Status is the refund status
	Status RefundStatus
	// RefundAmount is the refunded amount
	RefundAmount decimal.Decimal
	// RefundedAt is when the refund was completed
	RefundedAt *time.Time
	// RawPayload is the original callback payload
	RawPayload string
	// Signature is the callback signature for verification
	Signature string
}

// ---------------------------------------------------------------------------
// PaymentGateway Port Interface
// ---------------------------------------------------------------------------

// PaymentGateway defines the port interface for external payment gateways
// This interface follows the Ports & Adapters pattern - it's defined in the domain
// layer, and concrete implementations (WeChat Pay, Alipay) are in the infrastructure layer.
type PaymentGateway interface {
	// GatewayType returns the type of this payment gateway
	GatewayType() PaymentGatewayType

	// CreatePayment creates a new payment order in the gateway
	// Returns payment URL/QR code for user to complete payment
	CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*CreatePaymentResponse, error)

	// QueryPayment queries the current status of a payment
	QueryPayment(ctx context.Context, req *QueryPaymentRequest) (*QueryPaymentResponse, error)

	// ClosePayment closes/cancels a pending payment order
	ClosePayment(ctx context.Context, req *ClosePaymentRequest) (*ClosePaymentResponse, error)

	// CreateRefund initiates a refund for a completed payment
	CreateRefund(ctx context.Context, req *RefundRequest) (*RefundResponse, error)

	// QueryRefund queries the status of a refund
	QueryRefund(ctx context.Context, tenantID uuid.UUID, gatewayRefundID string) (*RefundResponse, error)

	// VerifyCallback verifies and parses a payment callback notification
	// Returns nil if signature is invalid
	VerifyCallback(ctx context.Context, payload []byte, signature string) (*PaymentCallback, error)

	// VerifyRefundCallback verifies and parses a refund callback notification
	// Returns nil if signature is invalid
	VerifyRefundCallback(ctx context.Context, payload []byte, signature string) (*RefundCallback, error)

	// GenerateCallbackResponse generates the response to send back to gateway
	// after processing a callback (acknowledges receipt)
	GenerateCallbackResponse(success bool, message string) []byte
}

// PaymentCallbackHandler defines the interface for handling payment callbacks
// This is implemented by the application service layer to process payment notifications
type PaymentCallbackHandler interface {
	// HandlePaymentCallback processes a payment completion notification
	// Updates the corresponding receipt voucher and triggers reconciliation
	HandlePaymentCallback(ctx context.Context, callback *PaymentCallback) error

	// HandleRefundCallback processes a refund completion notification
	// Updates the corresponding records
	HandleRefundCallback(ctx context.Context, callback *RefundCallback) error
}

// PaymentGatewayRegistry provides access to configured payment gateways
// This allows selecting the appropriate gateway based on payment channel
type PaymentGatewayRegistry interface {
	// GetGateway returns the gateway for the specified type
	GetGateway(gatewayType PaymentGatewayType) (PaymentGateway, error)

	// GetGatewayByChannel returns the gateway for the specified channel
	GetGatewayByChannel(channel PaymentChannel) (PaymentGateway, error)

	// ListGateways returns all registered gateways
	ListGateways() []PaymentGateway

	// IsEnabled returns true if the gateway type is enabled
	IsEnabled(gatewayType PaymentGatewayType) bool
}
