package ecommerce

import (
	"encoding/json"

	"github.com/shopspring/decimal"
)

// ---------------------------------------------------------------------------
// Common Taobao API Response Types
// ---------------------------------------------------------------------------

// TaobaoResponse is the base response wrapper for all Taobao API calls
type TaobaoResponse struct {
	// ErrorResponse contains error information if the request failed
	ErrorResponse *TaobaoErrorResponse `json:"error_response,omitempty"`
}

// TaobaoErrorResponse represents an error response from Taobao API
type TaobaoErrorResponse struct {
	Code      string `json:"code"`
	Msg       string `json:"msg"`
	SubCode   string `json:"sub_code,omitempty"`
	SubMsg    string `json:"sub_msg,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// IsSuccess returns true if the response indicates success
func (r *TaobaoResponse) IsSuccess() bool {
	return r.ErrorResponse == nil
}

// ---------------------------------------------------------------------------
// Trade/Order Related Types
// ---------------------------------------------------------------------------

// TaobaoTradesGetResponse is the response for taobao.trades.sold.get API
type TaobaoTradesGetResponse struct {
	TaobaoResponse
	TradesSoldGetResponse *TradesSoldGetResponse `json:"trades_sold_get_response,omitempty"`
}

// TradesSoldGetResponse contains the sold trades data
type TradesSoldGetResponse struct {
	TotalResults int64         `json:"total_results"`
	HasNext      bool          `json:"has_next"`
	Trades       *TaobaoTrades `json:"trades,omitempty"`
	RequestID    string        `json:"request_id"`
}

// TaobaoTrades is a wrapper for trade list
type TaobaoTrades struct {
	Trade []TaobaoTrade `json:"trade"`
}

// TaobaoTrade represents a trade/order from Taobao
type TaobaoTrade struct {
	// Basic order info
	Tid          int64  `json:"tid"`                      // Trade ID (order number)
	Status       string `json:"status"`                   // Order status
	Type         string `json:"type,omitempty"`           // Order type
	BuyerNick    string `json:"buyer_nick"`               // Buyer nickname
	BuyerOpenUID string `json:"buyer_open_uid,omitempty"` // Buyer open UID

	// Timestamps
	Created     string `json:"created,omitempty"`      // Order creation time
	Modified    string `json:"modified,omitempty"`     // Last modified time
	PayTime     string `json:"pay_time,omitempty"`     // Payment time
	ConsignTime string `json:"consign_time,omitempty"` // Shipping time
	EndTime     string `json:"end_time,omitempty"`     // Order end time

	// Amounts
	Payment     string `json:"payment,omitempty"`      // Total payment amount
	TotalFee    string `json:"total_fee,omitempty"`    // Total order fee
	PostFee     string `json:"post_fee,omitempty"`     // Shipping fee
	DiscountFee string `json:"discount_fee,omitempty"` // Discount amount
	AdjustFee   string `json:"adjust_fee,omitempty"`   // Adjustment fee
	BuyerRate   bool   `json:"buyer_rate,omitempty"`   // Whether buyer has rated
	SellerRate  bool   `json:"seller_rate,omitempty"`  // Whether seller has rated

	// Receiver info
	ReceiverName     string `json:"receiver_name,omitempty"`
	ReceiverState    string `json:"receiver_state,omitempty"`
	ReceiverCity     string `json:"receiver_city,omitempty"`
	ReceiverDistrict string `json:"receiver_district,omitempty"`
	ReceiverAddress  string `json:"receiver_address,omitempty"`
	ReceiverZip      string `json:"receiver_zip,omitempty"`
	ReceiverMobile   string `json:"receiver_mobile,omitempty"`
	ReceiverPhone    string `json:"receiver_phone,omitempty"`

	// Shipping info
	ShippingType string `json:"shipping_type,omitempty"`
	Sid          string `json:"sid,omitempty"`          // Shipping ID
	CompanyCode  string `json:"company_code,omitempty"` // Shipping company code

	// Buyer/Seller notes
	BuyerMemo    string `json:"buyer_memo,omitempty"`
	SellerMemo   string `json:"seller_memo,omitempty"`
	BuyerMessage string `json:"buyer_message,omitempty"`

	// Order items
	Orders *TaobaoOrders `json:"orders,omitempty"`

	// Other fields
	NumIid  int64  `json:"num_iid,omitempty"`  // Item ID (for single-item orders)
	Title   string `json:"title,omitempty"`    // Item title
	Price   string `json:"price,omitempty"`    // Item price
	Num     int64  `json:"num,omitempty"`      // Item quantity
	PicPath string `json:"pic_path,omitempty"` // Item image path
}

// TaobaoOrders is a wrapper for order items
type TaobaoOrders struct {
	Order []TaobaoOrder `json:"order"`
}

// TaobaoOrder represents an order line item
type TaobaoOrder struct {
	Oid               int64  `json:"oid"`                           // Order item ID
	NumIid            int64  `json:"num_iid"`                       // Item ID
	SkuID             string `json:"sku_id,omitempty"`              // SKU ID
	Title             string `json:"title"`                         // Item title
	SkuPropertiesName string `json:"sku_properties_name,omitempty"` // SKU properties
	Price             string `json:"price"`                         // Unit price
	Num               int64  `json:"num"`                           // Quantity
	TotalFee          string `json:"total_fee"`                     // Total fee
	Payment           string `json:"payment"`                       // Payment amount
	DiscountFee       string `json:"discount_fee,omitempty"`        // Discount
	AdjustFee         string `json:"adjust_fee,omitempty"`          // Adjustment
	RefundStatus      string `json:"refund_status,omitempty"`       // Refund status
	RefundID          int64  `json:"refund_id,omitempty"`           // Refund ID
	Status            string `json:"status,omitempty"`              // Item status
	PicPath           string `json:"pic_path,omitempty"`            // Image path
	OuterIid          string `json:"outer_iid,omitempty"`           // Outer item ID (seller code)
	OuterSkuID        string `json:"outer_sku_id,omitempty"`        // Outer SKU ID
}

// TaobaoTradeGetResponse is the response for taobao.trade.fullinfo.get API
type TaobaoTradeGetResponse struct {
	TaobaoResponse
	TradeFullinfoGetResponse *TradeFullinfoGetResponse `json:"trade_fullinfo_get_response,omitempty"`
}

// TradeFullinfoGetResponse contains full trade info
type TradeFullinfoGetResponse struct {
	Trade     *TaobaoTrade `json:"trade,omitempty"`
	RequestID string       `json:"request_id"`
}

// ---------------------------------------------------------------------------
// Logistics/Shipping Related Types
// ---------------------------------------------------------------------------

// TaobaoLogisticsSendResponse is the response for taobao.logistics.offline.send API
type TaobaoLogisticsSendResponse struct {
	TaobaoResponse
	LogisticsOfflineSendResponse *LogisticsOfflineSendResponse `json:"logistics_offline_send_response,omitempty"`
}

// LogisticsOfflineSendResponse contains offline send result
type LogisticsOfflineSendResponse struct {
	Shipping  *TaobaoShipping `json:"shipping,omitempty"`
	RequestID string          `json:"request_id"`
}

// TaobaoShipping represents shipping information
type TaobaoShipping struct {
	IsSuccess bool `json:"is_success"`
}

// TaobaoLogisticsCompaniesResponse is the response for taobao.logistics.companies.get API
type TaobaoLogisticsCompaniesResponse struct {
	TaobaoResponse
	LogisticsCompaniesGetResponse *LogisticsCompaniesGetResponse `json:"logistics_companies_get_response,omitempty"`
}

// LogisticsCompaniesGetResponse contains logistics companies list
type LogisticsCompaniesGetResponse struct {
	LogisticsCompanies *LogisticsCompanies `json:"logistics_companies,omitempty"`
	RequestID          string              `json:"request_id"`
}

// LogisticsCompanies is a wrapper for logistics company list
type LogisticsCompanies struct {
	LogisticsCompany []LogisticsCompany `json:"logistics_company"`
}

// LogisticsCompany represents a logistics/shipping company
type LogisticsCompany struct {
	ID      int64  `json:"id"`
	Code    string `json:"code"`
	Name    string `json:"name"`
	RegMail string `json:"reg_mail_no,omitempty"`
}

// ---------------------------------------------------------------------------
// Inventory Related Types
// ---------------------------------------------------------------------------

// TaobaoInventoryUpdateResponse is the response for taobao.inventory.adjust.external API
type TaobaoInventoryUpdateResponse struct {
	TaobaoResponse
	InventoryAdjustExternalResponse *InventoryAdjustExternalResponse `json:"inventory_adjust_external_response,omitempty"`
}

// InventoryAdjustExternalResponse contains inventory update result
type InventoryAdjustExternalResponse struct {
	OperateCode string    `json:"operate_code,omitempty"`
	TipInfos    *TipInfos `json:"tip_infos,omitempty"`
	RequestID   string    `json:"request_id"`
}

// TipInfos contains operation result details
type TipInfos struct {
	TipInfo []TipInfo `json:"tip_info"`
}

// TipInfo represents a single operation result
type TipInfo struct {
	ScItemID  int64  `json:"sc_item_id,omitempty"`
	OuterCode string `json:"outer_code,omitempty"`
	Info      string `json:"info,omitempty"`
}

// TaobaoSkuQuantityUpdateResponse is the response for taobao.item.sku.update API
type TaobaoSkuQuantityUpdateResponse struct {
	TaobaoResponse
	ItemSkuUpdateResponse *ItemSkuUpdateResponse `json:"item_sku_update_response,omitempty"`
}

// ItemSkuUpdateResponse contains SKU update result
type ItemSkuUpdateResponse struct {
	Sku       *TaobaoSku `json:"sku,omitempty"`
	RequestID string     `json:"request_id"`
}

// TaobaoItemQuantityUpdateResponse is the response for taobao.item.quantity.update API
type TaobaoItemQuantityUpdateResponse struct {
	TaobaoResponse
	ItemQuantityUpdateResponse *ItemQuantityUpdateResponse `json:"item_quantity_update_response,omitempty"`
}

// ItemQuantityUpdateResponse contains item quantity update result
type ItemQuantityUpdateResponse struct {
	Item      *TaobaoItem `json:"item,omitempty"`
	RequestID string      `json:"request_id"`
}

// TaobaoItem represents a product item
type TaobaoItem struct {
	NumIid   int64  `json:"num_iid"`
	Num      int64  `json:"num,omitempty"`
	Modified string `json:"modified,omitempty"`
}

// TaobaoSku represents a SKU
type TaobaoSku struct {
	SkuID    int64  `json:"sku_id"`
	NumIid   int64  `json:"num_iid"`
	Quantity int64  `json:"quantity,omitempty"`
	Modified string `json:"modified,omitempty"`
}

// ---------------------------------------------------------------------------
// Product Related Types
// ---------------------------------------------------------------------------

// TaobaoItemGetResponse is the response for taobao.item.get API
type TaobaoItemGetResponse struct {
	TaobaoResponse
	ItemGetResponse *ItemGetResponse `json:"item_get_response,omitempty"`
}

// ItemGetResponse contains item details
type ItemGetResponse struct {
	Item      *TaobaoFullItem `json:"item,omitempty"`
	RequestID string          `json:"request_id"`
}

// TaobaoFullItem represents full product details
type TaobaoFullItem struct {
	NumIid        int64       `json:"num_iid"`
	Title         string      `json:"title"`
	Nick          string      `json:"nick,omitempty"`
	Type          string      `json:"type,omitempty"`
	Cid           int64       `json:"cid,omitempty"`
	SellerCids    string      `json:"seller_cids,omitempty"`
	Props         string      `json:"props,omitempty"`
	Num           int64       `json:"num,omitempty"`
	ValidThru     int64       `json:"valid_thru,omitempty"`
	ListTime      string      `json:"list_time,omitempty"`
	DelistTime    string      `json:"delist_time,omitempty"`
	Desc          string      `json:"desc,omitempty"`
	Price         string      `json:"price,omitempty"`
	PostFee       string      `json:"post_fee,omitempty"`
	ExpressFee    string      `json:"express_fee,omitempty"`
	EmsFee        string      `json:"ems_fee,omitempty"`
	HasDiscount   bool        `json:"has_discount,omitempty"`
	HasInvoice    bool        `json:"has_invoice,omitempty"`
	HasWarranty   bool        `json:"has_warranty,omitempty"`
	HasShowcase   bool        `json:"has_showcase,omitempty"`
	Modified      string      `json:"modified,omitempty"`
	ApproveStatus string      `json:"approve_status,omitempty"`
	ItemImg       *ItemImgs   `json:"item_imgs,omitempty"`
	PropImgs      *PropImgs   `json:"prop_imgs,omitempty"`
	Skus          *TaobaoSkus `json:"skus,omitempty"`
	OuterId       string      `json:"outer_id,omitempty"`
	IsVirtual     bool        `json:"is_virtual,omitempty"`
}

// ItemImgs is a wrapper for item images
type ItemImgs struct {
	ItemImg []ItemImg `json:"item_img"`
}

// ItemImg represents an item image
type ItemImg struct {
	ID       int64  `json:"id,omitempty"`
	URL      string `json:"url"`
	Position int    `json:"position,omitempty"`
}

// PropImgs is a wrapper for property images
type PropImgs struct {
	PropImg []PropImg `json:"prop_img"`
}

// PropImg represents a property image
type PropImg struct {
	ID         int64  `json:"id,omitempty"`
	URL        string `json:"url"`
	Properties string `json:"properties,omitempty"`
}

// TaobaoSkus is a wrapper for SKU list
type TaobaoSkus struct {
	Sku []TaobaoFullSku `json:"sku"`
}

// TaobaoFullSku represents full SKU details
type TaobaoFullSku struct {
	SkuID          int64  `json:"sku_id"`
	NumIid         int64  `json:"num_iid"`
	Properties     string `json:"properties,omitempty"`
	PropertiesName string `json:"properties_name,omitempty"`
	Quantity       int64  `json:"quantity,omitempty"`
	Price          string `json:"price,omitempty"`
	OuterId        string `json:"outer_id,omitempty"`
	Status         string `json:"status,omitempty"`
	Created        string `json:"created,omitempty"`
	Modified       string `json:"modified,omitempty"`
}

// ---------------------------------------------------------------------------
// Helper Functions
// ---------------------------------------------------------------------------

// ParseDecimal safely parses a string to decimal
func ParseDecimal(s string) decimal.Decimal {
	if s == "" {
		return decimal.Zero
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

// MarshalJSON returns JSON representation of the type
func (t *TaobaoTrade) MarshalJSON() ([]byte, error) {
	type Alias TaobaoTrade
	return json.Marshal((*Alias)(t))
}
