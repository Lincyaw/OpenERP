package payment

// wechatErrorResponse represents an error response from WeChat Pay
type wechatErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  struct {
		Field    string `json:"field,omitempty"`
		Value    string `json:"value,omitempty"`
		Issue    string `json:"issue,omitempty"`
		Location string `json:"location,omitempty"`
	} `json:"detail,omitempty"`
}

// wechatQueryResponse represents the response from querying a payment
type wechatQueryResponse struct {
	AppID          string        `json:"appid"`
	MchID          string        `json:"mchid"`
	OutTradeNo     string        `json:"out_trade_no"`
	TransactionID  string        `json:"transaction_id,omitempty"`
	TradeType      string        `json:"trade_type,omitempty"`
	TradeState     string        `json:"trade_state"`
	TradeStateDesc string        `json:"trade_state_desc,omitempty"`
	BankType       string        `json:"bank_type,omitempty"`
	Attach         string        `json:"attach,omitempty"`
	SuccessTime    string        `json:"success_time,omitempty"`
	Payer          *wechatPayer  `json:"payer,omitempty"`
	Amount         *wechatAmount `json:"amount,omitempty"`
}

// wechatPayer represents payer information
type wechatPayer struct {
	OpenID string `json:"openid"`
}

// wechatAmount represents amount information in responses
type wechatAmount struct {
	Total         int    `json:"total"`
	PayerTotal    int    `json:"payer_total"`
	Currency      string `json:"currency"`
	PayerCurrency string `json:"payer_currency"`
}

// wechatRefundRequest represents a refund request
type wechatRefundRequest struct {
	TransactionID string             `json:"transaction_id,omitempty"`
	OutTradeNo    string             `json:"out_trade_no,omitempty"`
	OutRefundNo   string             `json:"out_refund_no"`
	Reason        string             `json:"reason,omitempty"`
	NotifyURL     string             `json:"notify_url,omitempty"`
	FundsAccount  string             `json:"funds_account,omitempty"`
	Amount        wechatRefundAmount `json:"amount"`
}

// wechatRefundAmount represents refund amount information
type wechatRefundAmount struct {
	Refund   int    `json:"refund"`
	Total    int    `json:"total"`
	Currency string `json:"currency"`
}

// wechatRefundResponse represents a refund response
type wechatRefundResponse struct {
	RefundID            string `json:"refund_id"`
	OutRefundNo         string `json:"out_refund_no"`
	TransactionID       string `json:"transaction_id"`
	OutTradeNo          string `json:"out_trade_no"`
	Channel             string `json:"channel"`
	UserReceivedAccount string `json:"user_received_account"`
	SuccessTime         string `json:"success_time,omitempty"`
	CreateTime          string `json:"create_time"`
	Status              string `json:"status"`
	FundsAccount        string `json:"funds_account,omitempty"`
	Amount              struct {
		Total            int    `json:"total"`
		Refund           int    `json:"refund"`
		PayerTotal       int    `json:"payer_total"`
		PayerRefund      int    `json:"payer_refund"`
		SettlementTotal  int    `json:"settlement_total"`
		SettlementRefund int    `json:"settlement_refund"`
		DiscountRefund   int    `json:"discount_refund"`
		Currency         string `json:"currency"`
	} `json:"amount"`
}

// wechatNotification represents a notification from WeChat Pay
type wechatNotification struct {
	ID           string         `json:"id"`
	CreateTime   string         `json:"create_time"`
	EventType    string         `json:"event_type"`
	ResourceType string         `json:"resource_type"`
	Resource     wechatResource `json:"resource"`
	Summary      string         `json:"summary"`
}

// wechatResource represents encrypted resource in notification
type wechatResource struct {
	Algorithm      string `json:"algorithm"`
	Ciphertext     string `json:"ciphertext"`
	AssociatedData string `json:"associated_data"`
	OriginalType   string `json:"original_type"`
	Nonce          string `json:"nonce"`
}

// wechatPaymentNotification represents decrypted payment notification
type wechatPaymentNotification struct {
	AppID          string        `json:"appid"`
	MchID          string        `json:"mchid"`
	OutTradeNo     string        `json:"out_trade_no"`
	TransactionID  string        `json:"transaction_id"`
	TradeType      string        `json:"trade_type"`
	TradeState     string        `json:"trade_state"`
	TradeStateDesc string        `json:"trade_state_desc"`
	BankType       string        `json:"bank_type"`
	Attach         string        `json:"attach"`
	SuccessTime    string        `json:"success_time"`
	Payer          *wechatPayer  `json:"payer"`
	Amount         *wechatAmount `json:"amount"`
}

// wechatRefundNotification represents decrypted refund notification
type wechatRefundNotification struct {
	MchID               string `json:"mchid"`
	OutTradeNo          string `json:"out_trade_no"`
	TransactionID       string `json:"transaction_id"`
	OutRefundNo         string `json:"out_refund_no"`
	RefundID            string `json:"refund_id"`
	RefundStatus        string `json:"refund_status"`
	SuccessTime         string `json:"success_time,omitempty"`
	UserReceivedAccount string `json:"user_received_account"`
	Amount              struct {
		Total       int `json:"total"`
		Refund      int `json:"refund"`
		PayerTotal  int `json:"payer_total"`
		PayerRefund int `json:"payer_refund"`
	} `json:"amount"`
}
