package ecommerce

import (
	"encoding/json"
)

// ---------------------------------------------------------------------------
// Common Douyin API Response Types
// ---------------------------------------------------------------------------

// DouyinResponse is the base response wrapper for all Douyin API calls
type DouyinResponse struct {
	// ErrNo is the error code (0 for success)
	ErrNo int `json:"err_no"`
	// Message is the error message
	Message string `json:"message"`
	// LogID is the request trace ID for debugging
	LogID string `json:"log_id,omitempty"`
}

// IsSuccess returns true if the response indicates success
func (r *DouyinResponse) IsSuccess() bool {
	return r.ErrNo == 0
}

// ---------------------------------------------------------------------------
// Order Related Types
// ---------------------------------------------------------------------------

// DouyinOrderListResponse is the response for order.searchList API
type DouyinOrderListResponse struct {
	DouyinResponse
	Data *DouyinOrderListData `json:"data,omitempty"`
}

// DouyinOrderListData contains the order list data
type DouyinOrderListData struct {
	Total int64         `json:"total"`
	List  []DouyinOrder `json:"list,omitempty"`
}

// DouyinOrderDetailResponse is the response for order.orderDetail API
type DouyinOrderDetailResponse struct {
	DouyinResponse
	Data *DouyinOrderDetailData `json:"data,omitempty"`
}

// DouyinOrderDetailData contains the order detail data
type DouyinOrderDetailData struct {
	ShopOrderDetail *DouyinOrder `json:"shop_order_detail,omitempty"`
}

// DouyinOrder represents an order from Douyin platform
type DouyinOrder struct {
	// Basic order info
	OrderID     string `json:"order_id"`     // Order ID (string format)
	ShopID      int64  `json:"shop_id"`      // Shop ID
	OrderStatus int    `json:"order_status"` // Order status code
	OrderType   int    `json:"order_type"`   // Order type

	// Timestamps (Unix milliseconds)
	CreateTime  int64 `json:"create_time"`   // Order creation time
	UpdateTime  int64 `json:"update_time"`   // Order update time
	PayTime     int64 `json:"pay_time"`      // Payment time
	ExpShipTime int64 `json:"exp_ship_time"` // Expected shipping time
	FinishTime  int64 `json:"finish_time"`   // Order finish time

	// Amounts (in cents/fen)
	OrderAmount      int64 `json:"order_amount"`       // Total order amount
	PayAmount        int64 `json:"pay_amount"`         // Actual payment amount
	PostAmount       int64 `json:"post_amount"`        // Shipping fee
	CouponAmount     int64 `json:"coupon_amount"`      // Coupon discount
	ShopCouponAmount int64 `json:"shop_coupon_amount"` // Shop coupon discount
	PlatformDiscount int64 `json:"platform_discount"`  // Platform discount

	// Buyer info
	BuyerWords   string `json:"buyer_words,omitempty"`  // Buyer message
	SellerWords  string `json:"seller_words,omitempty"` // Seller notes
	BuyerOpenUID string `json:"open_id,omitempty"`      // Buyer open UID

	// Receiver info
	PostReceiver *DouyinPostReceiver `json:"post_receiver,omitempty"`

	// Logistics info
	LogisticsInfo *DouyinLogisticsInfo `json:"logistics_info,omitempty"`

	// Order items
	SkuOrderList []DouyinSkuOrder `json:"sku_order_list,omitempty"`

	// Channel info
	ChannelPaymentNo string `json:"channel_payment_no,omitempty"` // Payment channel order number
	BOrderID         string `json:"b_order_id,omitempty"`         // Business order ID

	// Promotion info
	CouponInfo []DouyinCouponInfo `json:"coupon_info,omitempty"`
}

// DouyinPostReceiver contains receiver/shipping address information
type DouyinPostReceiver struct {
	Name         string `json:"name"`          // Receiver name
	Phone        string `json:"phone"`         // Receiver phone (encrypted)
	Province     string `json:"province"`      // Province
	City         string `json:"city"`          // City
	Town         string `json:"town"`          // Town/District
	Street       string `json:"street"`        // Street address
	Detail       string `json:"detail"`        // Detailed address
	PostCode     string `json:"post_code"`     // Postal code
	EncryptPhone string `json:"encrypt_phone"` // Encrypted phone number
}

// DouyinLogisticsInfo contains logistics/shipping information
type DouyinLogisticsInfo struct {
	LogisticsID string `json:"logistics_id"` // Logistics ID
	Company     string `json:"company"`      // Logistics company
	TrackingNo  string `json:"tracking_no"`  // Tracking number
	ShipTime    int64  `json:"ship_time"`    // Ship time
	DeliveryID  string `json:"delivery_id"`  // Delivery ID
	CompanyCode string `json:"company_code"` // Logistics company code
}

// DouyinSkuOrder represents a SKU order item
type DouyinSkuOrder struct {
	OrderID       string `json:"order_id"`        // Parent order ID
	ParentOrderID string `json:"parent_order_id"` // Parent order ID
	SkuOrderID    string `json:"sku_order_id"`    // SKU order ID
	ProductID     int64  `json:"product_id"`      // Product ID
	ProductName   string `json:"product_name"`    // Product name
	SkuID         int64  `json:"sku_id"`          // SKU ID
	Code          string `json:"code"`            // SKU code
	SkuSpec       string `json:"sku_spec"`        // SKU spec (JSON format)
	ItemNum       int    `json:"item_num"`        // Quantity
	OriginAmount  int64  `json:"origin_amount"`   // Original amount
	PayAmount     int64  `json:"pay_amount"`      // Payment amount
	CouponAmount  int64  `json:"coupon_amount"`   // Coupon discount
	CampaignInfo  string `json:"campaign_info"`   // Promotion info
	ProductPic    string `json:"product_pic"`     // Product image URL
	OutSkuID      string `json:"out_sku_id"`      // External SKU ID
	OutProductID  string `json:"out_product_id"`  // External product ID
	OrderStatus   int    `json:"order_status"`    // Item order status
}

// DouyinCouponInfo represents coupon/promotion information
type DouyinCouponInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Amount     int64  `json:"amount"`
	CouponType int    `json:"coupon_type"`
}

// ---------------------------------------------------------------------------
// Logistics/Shipping Related Types
// ---------------------------------------------------------------------------

// DouyinShipResponse is the response for logistics.addOrder API
type DouyinShipResponse struct {
	DouyinResponse
	Data *DouyinShipData `json:"data,omitempty"`
}

// DouyinShipData contains the shipping result
type DouyinShipData struct {
	PackID string `json:"pack_id,omitempty"`
}

// DouyinLogisticsCompaniesResponse is the response for logistics.listCompany API
type DouyinLogisticsCompaniesResponse struct {
	DouyinResponse
	Data *DouyinLogisticsCompaniesData `json:"data,omitempty"`
}

// DouyinLogisticsCompaniesData contains logistics companies list
type DouyinLogisticsCompaniesData struct {
	List []DouyinLogisticsCompany `json:"list,omitempty"`
}

// DouyinLogisticsCompany represents a logistics/shipping company
type DouyinLogisticsCompany struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// ---------------------------------------------------------------------------
// Product Related Types
// ---------------------------------------------------------------------------

// DouyinProductListResponse is the response for product.listV2 API
type DouyinProductListResponse struct {
	DouyinResponse
	Data *DouyinProductListData `json:"data,omitempty"`
}

// DouyinProductListData contains product list data
type DouyinProductListData struct {
	Total int64           `json:"total"`
	Data  []DouyinProduct `json:"data,omitempty"`
}

// DouyinProductDetailResponse is the response for product.detail API
type DouyinProductDetailResponse struct {
	DouyinResponse
	Data *DouyinProductDetailData `json:"data,omitempty"`
}

// DouyinProductDetailData contains product detail
type DouyinProductDetailData struct {
	Product *DouyinProduct `json:"product,omitempty"`
}

// DouyinProduct represents a product from Douyin platform
type DouyinProduct struct {
	ProductID     int64               `json:"product_id"`     // Product ID
	OutProductID  string              `json:"out_product_id"` // External product ID
	Name          string              `json:"name"`           // Product name
	Img           string              `json:"img"`            // Main image URL
	MarketPrice   int64               `json:"market_price"`   // Market price (cents)
	DiscountPrice int64               `json:"discount_price"` // Discount price (cents)
	Status        int                 `json:"status"`         // Product status
	CreateTime    int64               `json:"create_time"`    // Creation time
	UpdateTime    int64               `json:"update_time"`    // Update time
	Description   string              `json:"description"`    // Product description
	CategoryID    int64               `json:"category_id"`    // Category ID
	CategoryName  string              `json:"category_name"`  // Category name
	Specs         []DouyinProductSpec `json:"specs,omitempty"`
	SkuList       []DouyinSku         `json:"sku_list,omitempty"`
}

// DouyinProductSpec represents a product specification
type DouyinProductSpec struct {
	ID     int64                    `json:"id"`
	Name   string                   `json:"name"`
	Values []DouyinProductSpecValue `json:"values,omitempty"`
}

// DouyinProductSpecValue represents a specification value
type DouyinProductSpecValue struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Image string `json:"image,omitempty"`
}

// DouyinSku represents a SKU
type DouyinSku struct {
	SkuID       int64              `json:"sku_id"`       // SKU ID
	OutSkuID    string             `json:"out_sku_id"`   // External SKU ID
	Code        string             `json:"code"`         // SKU code
	StockNum    int64              `json:"stock_num"`    // Stock quantity
	Price       int64              `json:"price"`        // SKU price (cents)
	SettlePrice int64              `json:"settle_price"` // Settlement price
	SpecDetail  []DouyinSpecDetail `json:"spec_detail,omitempty"`
}

// DouyinSpecDetail represents SKU specification detail
type DouyinSpecDetail struct {
	SpecID    int64  `json:"spec_id"`
	SpecName  string `json:"spec_name"`
	ValueID   int64  `json:"value_id"`
	ValueName string `json:"value_name"`
}

// ---------------------------------------------------------------------------
// Inventory Related Types
// ---------------------------------------------------------------------------

// DouyinStockUpdateResponse is the response for sku.stockNum API
type DouyinStockUpdateResponse struct {
	DouyinResponse
	Data *DouyinStockUpdateData `json:"data,omitempty"`
}

// DouyinStockUpdateData contains stock update result
type DouyinStockUpdateData struct {
	Success bool `json:"success"`
}

// DouyinStockSyncRequest represents a stock sync request
type DouyinStockSyncRequest struct {
	OutWarehouseID string            `json:"out_warehouse_id,omitempty"`
	StockList      []DouyinStockItem `json:"stock_list"`
}

// DouyinStockItem represents a single stock item to update
type DouyinStockItem struct {
	SkuID    int64  `json:"sku_id"`
	OutSkuID string `json:"out_sku_id,omitempty"`
	StockNum int64  `json:"stock_num"`
}

// ---------------------------------------------------------------------------
// Order Status Constants
// ---------------------------------------------------------------------------

const (
	// DouyinOrderStatusPendingPayment indicates order is waiting for payment
	DouyinOrderStatusPendingPayment = 1
	// DouyinOrderStatusPendingShipment indicates order is paid and waiting for shipment
	DouyinOrderStatusPendingShipment = 2
	// DouyinOrderStatusShipped indicates order has been shipped
	DouyinOrderStatusShipped = 3
	// DouyinOrderStatusCompleted indicates order is completed
	DouyinOrderStatusCompleted = 4
	// DouyinOrderStatusCancelled indicates order is cancelled
	DouyinOrderStatusCancelled = 5
	// DouyinOrderStatusRefunding indicates refund in progress
	DouyinOrderStatusRefunding = 6
	// DouyinOrderStatusRefunded indicates refund completed
	DouyinOrderStatusRefunded = 7
)

// ---------------------------------------------------------------------------
// Helper Functions
// ---------------------------------------------------------------------------

// MarshalJSON returns JSON representation of the type
func (o *DouyinOrder) MarshalJSON() ([]byte, error) {
	type Alias DouyinOrder
	return json.Marshal((*Alias)(o))
}

// ParseDouyinOrderStatus converts Douyin status code to display string
func ParseDouyinOrderStatus(status int) string {
	switch status {
	case DouyinOrderStatusPendingPayment:
		return "待付款"
	case DouyinOrderStatusPendingShipment:
		return "待发货"
	case DouyinOrderStatusShipped:
		return "已发货"
	case DouyinOrderStatusCompleted:
		return "已完成"
	case DouyinOrderStatusCancelled:
		return "已取消"
	case DouyinOrderStatusRefunding:
		return "退款中"
	case DouyinOrderStatusRefunded:
		return "已退款"
	default:
		return "未知状态"
	}
}
