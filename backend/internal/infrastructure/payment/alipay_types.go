package payment

// alipayErrorResponse represents an error response from Alipay
type alipayErrorResponse struct {
	Code    string `json:"code"`
	Msg     string `json:"msg"`
	SubCode string `json:"sub_code,omitempty"`
	SubMsg  string `json:"sub_msg,omitempty"`
}

// IsSuccess returns true if the response indicates success
func (r *alipayErrorResponse) IsSuccess() bool {
	return r.Code == "10000"
}

// alipayTradeCreateResponse represents response for creating a trade
type alipayTradeCreateResponse struct {
	Response struct {
		alipayErrorResponse
		TradeNo    string `json:"trade_no,omitempty"`
		OutTradeNo string `json:"out_trade_no,omitempty"`
	} `json:"alipay_trade_create_response"`
	Sign string `json:"sign"`
}

// alipayTradePagePayResponse represents response for page payment (PC web)
type alipayTradePagePayResponse struct {
	// Page pay returns an HTML form, not JSON
	FormHTML string
}

// alipayTradeWapPayResponse represents response for WAP payment (mobile web)
type alipayTradeWapPayResponse struct {
	// WAP pay returns an HTML form, not JSON
	FormHTML string
}

// alipayTradeAppPayResponse represents response for APP payment
type alipayTradeAppPayResponse struct {
	// App pay returns a signed order string
	OrderString string
}

// alipayTradePrecreateResponse represents response for precreate (QR code)
type alipayTradePrecreateResponse struct {
	Response struct {
		alipayErrorResponse
		OutTradeNo string `json:"out_trade_no,omitempty"`
		QRCode     string `json:"qr_code,omitempty"`
	} `json:"alipay_trade_precreate_response"`
	Sign string `json:"sign"`
}

// alipayTradeQueryResponse represents response for querying a trade
type alipayTradeQueryResponse struct {
	Response struct {
		alipayErrorResponse
		TradeNo        string `json:"trade_no,omitempty"`
		OutTradeNo     string `json:"out_trade_no,omitempty"`
		BuyerLogonID   string `json:"buyer_logon_id,omitempty"`
		TradeStatus    string `json:"trade_status,omitempty"`
		TotalAmount    string `json:"total_amount,omitempty"`
		ReceiptAmount  string `json:"receipt_amount,omitempty"`
		BuyerPayAmount string `json:"buyer_pay_amount,omitempty"`
		PointAmount    string `json:"point_amount,omitempty"`
		InvoiceAmount  string `json:"invoice_amount,omitempty"`
		SendPayDate    string `json:"send_pay_date,omitempty"`
		BuyerUserID    string `json:"buyer_user_id,omitempty"`
		BuyerUserType  string `json:"buyer_user_type,omitempty"`
	} `json:"alipay_trade_query_response"`
	Sign string `json:"sign"`
}

// alipayTradeCloseResponse represents response for closing a trade
type alipayTradeCloseResponse struct {
	Response struct {
		alipayErrorResponse
		TradeNo    string `json:"trade_no,omitempty"`
		OutTradeNo string `json:"out_trade_no,omitempty"`
	} `json:"alipay_trade_close_response"`
	Sign string `json:"sign"`
}

// alipayTradeRefundResponse represents response for refund
type alipayTradeRefundResponse struct {
	Response struct {
		alipayErrorResponse
		TradeNo              string `json:"trade_no,omitempty"`
		OutTradeNo           string `json:"out_trade_no,omitempty"`
		BuyerLogonID         string `json:"buyer_logon_id,omitempty"`
		FundChange           string `json:"fund_change,omitempty"`
		RefundFee            string `json:"refund_fee,omitempty"`
		RefundCurrency       string `json:"refund_currency,omitempty"`
		GmtRefundPay         string `json:"gmt_refund_pay,omitempty"`
		RefundDetailItemList []struct {
			FundChannel string `json:"fund_channel"`
			Amount      string `json:"amount"`
			RealAmount  string `json:"real_amount"`
		} `json:"refund_detail_item_list,omitempty"`
		StoreName               string `json:"store_name,omitempty"`
		BuyerUserID             string `json:"buyer_user_id,omitempty"`
		RefundPresetPaytoolList []struct {
			Amount         []string `json:"amount"`
			AssertTypeCode string   `json:"assert_type_code"`
		} `json:"refund_preset_paytool_list,omitempty"`
		RefundSettlementID           string `json:"refund_settlement_id,omitempty"`
		PresentRefundBuyerAmount     string `json:"present_refund_buyer_amount,omitempty"`
		PresentRefundDiscountAmount  string `json:"present_refund_discount_amount,omitempty"`
		PresentRefundMdiscountAmount string `json:"present_refund_mdiscount_amount,omitempty"`
	} `json:"alipay_trade_refund_response"`
	Sign string `json:"sign"`
}

// alipayTradeFastpayRefundQueryResponse represents response for refund query
type alipayTradeFastpayRefundQueryResponse struct {
	Response struct {
		alipayErrorResponse
		TradeNo        string `json:"trade_no,omitempty"`
		OutTradeNo     string `json:"out_trade_no,omitempty"`
		OutRequestNo   string `json:"out_request_no,omitempty"`
		RefundStatus   string `json:"refund_status,omitempty"`
		TotalAmount    string `json:"total_amount,omitempty"`
		RefundAmount   string `json:"refund_amount,omitempty"`
		RefundRoyaltys []struct {
			RefundAmount string `json:"refund_amount"`
			RoyaltyType  string `json:"royalty_type"`
			ResultCode   string `json:"result_code"`
			TransOut     string `json:"trans_out"`
			TransIn      string `json:"trans_in"`
		} `json:"refund_royaltys,omitempty"`
		GmtRefundPay         string `json:"gmt_refund_pay,omitempty"`
		RefundDetailItemList []struct {
			FundChannel string `json:"fund_channel"`
			Amount      string `json:"amount"`
			RealAmount  string `json:"real_amount"`
		} `json:"refund_detail_item_list,omitempty"`
		SendBackFee                  string `json:"send_back_fee,omitempty"`
		RefundSettlementID           string `json:"refund_settlement_id,omitempty"`
		PresentRefundBuyerAmount     string `json:"present_refund_buyer_amount,omitempty"`
		PresentRefundDiscountAmount  string `json:"present_refund_discount_amount,omitempty"`
		PresentRefundMdiscountAmount string `json:"present_refund_mdiscount_amount,omitempty"`
	} `json:"alipay_trade_fastpay_refund_query_response"`
	Sign string `json:"sign"`
}

// alipayNotification represents a payment notification from Alipay
// Alipay sends notifications as URL-encoded form data, not JSON
type alipayNotification struct {
	// Common fields
	NotifyTime string `json:"notify_time"` // Notification time
	NotifyType string `json:"notify_type"` // Notification type
	NotifyID   string `json:"notify_id"`   // Notification ID
	AppID      string `json:"app_id"`      // App ID
	Charset    string `json:"charset"`     // Encoding
	Version    string `json:"version"`     // API version
	SignType   string `json:"sign_type"`   // Signature type (RSA2 or RSA)
	Sign       string `json:"sign"`        // Signature

	// Trade fields
	TradeNo        string `json:"trade_no"`         // Alipay trade number
	OutTradeNo     string `json:"out_trade_no"`     // Merchant order number
	OutBizNo       string `json:"out_biz_no"`       // Merchant business number
	BuyerID        string `json:"buyer_id"`         // Buyer Alipay user ID
	BuyerLogonID   string `json:"buyer_logon_id"`   // Buyer Alipay account
	SellerID       string `json:"seller_id"`        // Seller Alipay user ID
	SellerEmail    string `json:"seller_email"`     // Seller Alipay email
	TradeStatus    string `json:"trade_status"`     // Trade status
	TotalAmount    string `json:"total_amount"`     // Total amount
	ReceiptAmount  string `json:"receipt_amount"`   // Receipt amount
	InvoiceAmount  string `json:"invoice_amount"`   // Invoice amount
	BuyerPayAmount string `json:"buyer_pay_amount"` // Buyer payment amount
	PointAmount    string `json:"point_amount"`     // Points used
	RefundFee      string `json:"refund_fee"`       // Refund amount
	Subject        string `json:"subject"`          // Order subject
	Body           string `json:"body"`             // Order description
	GmtCreate      string `json:"gmt_create"`       // Order creation time
	GmtPayment     string `json:"gmt_payment"`      // Payment time
	GmtRefund      string `json:"gmt_refund"`       // Refund time
	GmtClose       string `json:"gmt_close"`        // Close time
	FundBillList   string `json:"fund_bill_list"`   // Payment fund list (JSON)
	PassbackParams string `json:"passback_params"`  // Passback parameters

	// Refund fields (for refund notifications)
	OutRequestNo string `json:"out_request_no"` // Refund request number
	RefundStatus string `json:"refund_status"`  // Refund status
}

// alipayBizContent represents the biz_content parameter for API requests
type alipayBizContent struct {
	OutTradeNo     string `json:"out_trade_no,omitempty"`
	ProductCode    string `json:"product_code,omitempty"`
	TotalAmount    string `json:"total_amount,omitempty"`
	Subject        string `json:"subject,omitempty"`
	Body           string `json:"body,omitempty"`
	TimeoutExpress string `json:"timeout_express,omitempty"`
	TimeExpire     string `json:"time_expire,omitempty"`
	PassbackParams string `json:"passback_params,omitempty"`
	QuitURL        string `json:"quit_url,omitempty"`
	TradeNo        string `json:"trade_no,omitempty"`
	RefundAmount   string `json:"refund_amount,omitempty"`
	RefundReason   string `json:"refund_reason,omitempty"`
	OutRequestNo   string `json:"out_request_no,omitempty"`
}

// Alipay API methods
const (
	alipayMethodPagePay     = "alipay.trade.page.pay"             // PC web payment
	alipayMethodWapPay      = "alipay.trade.wap.pay"              // Mobile web payment
	alipayMethodAppPay      = "alipay.trade.app.pay"              // App payment
	alipayMethodPrecreate   = "alipay.trade.precreate"            // QR code payment
	alipayMethodCreate      = "alipay.trade.create"               // F2F payment
	alipayMethodQuery       = "alipay.trade.query"                // Query payment
	alipayMethodClose       = "alipay.trade.close"                // Close payment
	alipayMethodRefund      = "alipay.trade.refund"               // Refund
	alipayMethodRefundQuery = "alipay.trade.fastpay.refund.query" // Query refund
)

// Alipay product codes
const (
	alipayProductCodeFastInstantTradePay = "FAST_INSTANT_TRADE_PAY" // PC web
	alipayProductCodeQuickWapWay         = "QUICK_WAP_WAY"          // Mobile web
	alipayProductCodeQuickMSecPay        = "QUICK_MSECURITY_PAY"    // App
	alipayProductCodeFaceToFacePayment   = "FACE_TO_FACE_PAYMENT"   // QR/F2F
)

// Alipay trade status
const (
	alipayTradeStatusWaitBuyerPay  = "WAIT_BUYER_PAY" // Waiting for payment
	alipayTradeStatusTradeClosed   = "TRADE_CLOSED"   // Trade closed
	alipayTradeStatusTradeSuccess  = "TRADE_SUCCESS"  // Trade success
	alipayTradeStatusTradeFinished = "TRADE_FINISHED" // Trade finished (no refund allowed)
)

// Alipay refund status
const (
	alipayRefundStatusRefundSuccess = "REFUND_SUCCESS" // Refund success
)
