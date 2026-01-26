package ecommerce

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/erp/backend/internal/domain/integration"
)

// maxResponseSize is the maximum allowed response size from Taobao API (10MB)
const maxResponseSize = 10 * 1024 * 1024

// ErrTaobaoInvalidProductID indicates an invalid product ID format
var ErrTaobaoInvalidProductID = errors.New("taobao: invalid product ID format")

// TaobaoAdapter implements EcommercePlatform interface for Taobao/Tmall platform
type TaobaoAdapter struct {
	config     *TaobaoConfig
	httpClient *http.Client

	// tenantConfigs stores per-tenant configurations
	// In production, this would be loaded from database
	tenantConfigs map[uuid.UUID]*TaobaoConfig
	mu            sync.RWMutex // Protects tenantConfigs map
}

// NewTaobaoAdapter creates a new Taobao adapter with the given configuration
func NewTaobaoAdapter(config *TaobaoConfig) (*TaobaoAdapter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &TaobaoAdapter{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.TimeoutSeconds) * time.Second,
		},
		tenantConfigs: make(map[uuid.UUID]*TaobaoConfig),
	}, nil
}

// SetTenantConfig sets the configuration for a specific tenant
func (a *TaobaoAdapter) SetTenantConfig(tenantID uuid.UUID, config *TaobaoConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tenantConfigs[tenantID] = config
	return nil
}

// getTenantConfig retrieves the configuration for a tenant
func (a *TaobaoAdapter) getTenantConfig(tenantID uuid.UUID) (*TaobaoConfig, error) {
	a.mu.RLock()
	config, ok := a.tenantConfigs[tenantID]
	a.mu.RUnlock()
	if ok {
		return config, nil
	}
	// Fall back to default config
	if a.config != nil {
		return a.config, nil
	}
	return nil, integration.ErrPlatformNotConfigured
}

// validateNumericID validates that a string is a valid numeric ID
func validateNumericID(id string) error {
	if id == "" {
		return ErrTaobaoInvalidProductID
	}
	if _, err := strconv.ParseInt(id, 10, 64); err != nil {
		return fmt.Errorf("%w: %s", ErrTaobaoInvalidProductID, id)
	}
	return nil
}

// PlatformCode returns the platform code this adapter handles
func (a *TaobaoAdapter) PlatformCode() integration.PlatformCode {
	return integration.PlatformCodeTaobao
}

// IsEnabled returns true if this platform is enabled for the tenant
func (a *TaobaoAdapter) IsEnabled(ctx context.Context, tenantID uuid.UUID) (bool, error) {
	_, err := a.getTenantConfig(tenantID)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// ---------------------------------------------------------------------------
// Product Operations
// ---------------------------------------------------------------------------

// SyncProducts synchronizes products to the Taobao platform
func (a *TaobaoAdapter) SyncProducts(ctx context.Context, tenantID uuid.UUID, products []integration.ProductSync) (*integration.SyncResult, error) {
	config, err := a.getTenantConfig(tenantID)
	if err != nil {
		return nil, err
	}

	result := &integration.SyncResult{
		Status:       integration.SyncStatusInProgress,
		TotalCount:   len(products),
		SuccessCount: 0,
		FailedCount:  0,
		FailedItems:  make([]integration.SyncFailure, 0),
		SyncedAt:     time.Now(),
	}

	for _, product := range products {
		// For Taobao, we use taobao.item.update API for existing products
		// and taobao.item.add for new products
		// Here we implement a simplified version focusing on inventory sync

		if product.PlatformProductID != "" {
			// Update existing product
			err := a.updateProductOnPlatform(ctx, config, &product)
			if err != nil {
				result.FailedCount++
				result.FailedItems = append(result.FailedItems, integration.SyncFailure{
					ItemID:       product.PlatformProductID,
					ErrorMessage: err.Error(),
				})
				continue
			}
		}
		result.SuccessCount++
	}

	// Set final status
	if result.FailedCount == 0 {
		result.Status = integration.SyncStatusSuccess
	} else if result.SuccessCount > 0 {
		result.Status = integration.SyncStatusPartial
	} else {
		result.Status = integration.SyncStatusFailed
	}

	return result, nil
}

// updateProductOnPlatform updates a product on Taobao
func (a *TaobaoAdapter) updateProductOnPlatform(ctx context.Context, config *TaobaoConfig, product *integration.ProductSync) error {
	// Use taobao.item.update API
	params := map[string]string{
		"method":  "taobao.item.update",
		"num_iid": product.PlatformProductID,
		"title":   product.ProductName,
		"num":     strconv.FormatInt(product.Quantity.IntPart(), 10),
	}

	if product.Price.IsPositive() {
		params["price"] = product.Price.StringFixed(2)
	}

	_, err := a.doRequest(ctx, config, params)
	return err
}

// GetProduct retrieves a product from Taobao platform
func (a *TaobaoAdapter) GetProduct(ctx context.Context, tenantID uuid.UUID, platformProductID string) (*integration.ProductSync, error) {
	// Validate input
	if err := validateNumericID(platformProductID); err != nil {
		return nil, err
	}

	config, err := a.getTenantConfig(tenantID)
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"method":  "taobao.item.get",
		"num_iid": platformProductID,
		"fields":  "num_iid,title,nick,type,cid,num,price,desc,item_imgs,skus,outer_id",
	}

	respBody, err := a.doRequest(ctx, config, params)
	if err != nil {
		return nil, err
	}

	var resp TaobaoItemGetResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("%w: failed to parse response: %v", integration.ErrPlatformInvalidResponse, err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("%w: %s - %s", integration.ErrPlatformRequestFailed, resp.ErrorResponse.Code, resp.ErrorResponse.Msg)
	}

	if resp.ItemGetResponse == nil || resp.ItemGetResponse.Item == nil {
		return nil, integration.ErrProductSyncMappingNotFound
	}

	item := resp.ItemGetResponse.Item
	product := &integration.ProductSync{
		PlatformProductID: strconv.FormatInt(item.NumIid, 10),
		ProductCode:       item.OuterId,
		ProductName:       item.Title,
		Description:       item.Desc,
		Price:             ParseDecimal(item.Price),
		Quantity:          decimal.NewFromInt(item.Num),
		IsOnSale:          item.ApproveStatus == "onsale",
		SKUs:              make([]integration.ProductSkuSync, 0),
	}

	// Parse images
	if item.ItemImg != nil {
		for _, img := range item.ItemImg.ItemImg {
			product.ImageURLs = append(product.ImageURLs, img.URL)
		}
	}

	// Parse SKUs
	if item.Skus != nil {
		for _, sku := range item.Skus.Sku {
			product.SKUs = append(product.SKUs, integration.ProductSkuSync{
				PlatformSkuID: strconv.FormatInt(sku.SkuID, 10),
				SkuCode:       sku.OuterId,
				SkuName:       sku.PropertiesName,
				Price:         ParseDecimal(sku.Price),
				Quantity:      decimal.NewFromInt(sku.Quantity),
			})
		}
	}

	return product, nil
}

// ---------------------------------------------------------------------------
// Order Operations
// ---------------------------------------------------------------------------

// PullOrders pulls orders from Taobao platform within the specified time range
func (a *TaobaoAdapter) PullOrders(ctx context.Context, req *integration.OrderPullRequest) (*integration.OrderPullResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	config, err := a.getTenantConfig(req.TenantID)
	if err != nil {
		return nil, err
	}

	// Build API parameters
	params := map[string]string{
		"method":        "taobao.trades.sold.get",
		"fields":        taobaoTradeFields,
		"start_created": req.StartTime.Format("2006-01-02 15:04:05"),
		"end_created":   req.EndTime.Format("2006-01-02 15:04:05"),
		"page_no":       strconv.Itoa(req.PageNo),
		"page_size":     strconv.Itoa(req.PageSize),
	}

	// Add status filter if specified
	if req.Status != nil {
		params["status"] = mapToTaobaoOrderStatus(*req.Status)
	}

	respBody, err := a.doRequest(ctx, config, params)
	if err != nil {
		return nil, err
	}

	var resp TaobaoTradesGetResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("%w: failed to parse response: %v", integration.ErrPlatformInvalidResponse, err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("%w: %s - %s", integration.ErrPlatformRequestFailed, resp.ErrorResponse.Code, resp.ErrorResponse.Msg)
	}

	if resp.TradesSoldGetResponse == nil {
		return nil, integration.ErrPlatformInvalidResponse
	}

	// Convert to domain models
	response := &integration.OrderPullResponse{
		Orders:     make([]integration.PlatformOrder, 0),
		TotalCount: resp.TradesSoldGetResponse.TotalResults,
		HasMore:    resp.TradesSoldGetResponse.HasNext,
		NextPageNo: req.PageNo + 1,
	}

	if resp.TradesSoldGetResponse.Trades != nil {
		for _, trade := range resp.TradesSoldGetResponse.Trades.Trade {
			order := a.convertTaobaoTradeToPlatformOrder(&trade)
			response.Orders = append(response.Orders, order)
		}
	}

	return response, nil
}

// GetOrder retrieves a single order from Taobao platform
func (a *TaobaoAdapter) GetOrder(ctx context.Context, tenantID uuid.UUID, platformOrderID string) (*integration.PlatformOrder, error) {
	// Validate input
	if err := validateNumericID(platformOrderID); err != nil {
		return nil, err
	}

	config, err := a.getTenantConfig(tenantID)
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"method": "taobao.trade.fullinfo.get",
		"tid":    platformOrderID,
		"fields": taobaoTradeFields,
	}

	respBody, err := a.doRequest(ctx, config, params)
	if err != nil {
		return nil, err
	}

	var resp TaobaoTradeGetResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("%w: failed to parse response: %v", integration.ErrPlatformInvalidResponse, err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("%w: %s - %s", integration.ErrPlatformRequestFailed, resp.ErrorResponse.Code, resp.ErrorResponse.Msg)
	}

	if resp.TradeFullinfoGetResponse == nil || resp.TradeFullinfoGetResponse.Trade == nil {
		return nil, integration.ErrOrderSyncOrderNotFound
	}

	order := a.convertTaobaoTradeToPlatformOrder(resp.TradeFullinfoGetResponse.Trade)
	return &order, nil
}

// UpdateOrderStatus updates the order status on Taobao platform
func (a *TaobaoAdapter) UpdateOrderStatus(ctx context.Context, req *integration.OrderStatusUpdateRequest) error {
	if err := req.Validate(); err != nil {
		return err
	}

	config, err := a.getTenantConfig(req.TenantID)
	if err != nil {
		return err
	}

	// For shipped status, use logistics API
	if req.Status == integration.PlatformOrderStatusShipped {
		return a.sendShipment(ctx, config, req)
	}

	// Other status updates are typically not supported by Taobao API
	// (orders transition through natural workflow on the platform)
	return nil
}

// sendShipment sends shipment information to Taobao
func (a *TaobaoAdapter) sendShipment(ctx context.Context, config *TaobaoConfig, req *integration.OrderStatusUpdateRequest) error {
	// Map shipping company name to Taobao code
	companyCode := mapShippingCompanyToTaobaoCode(req.ShippingCompany)

	params := map[string]string{
		"method":       "taobao.logistics.offline.send",
		"tid":          req.PlatformOrderID,
		"out_sid":      req.TrackingNumber,
		"company_code": companyCode,
	}

	respBody, err := a.doRequest(ctx, config, params)
	if err != nil {
		return err
	}

	var resp TaobaoLogisticsSendResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return fmt.Errorf("%w: failed to parse response: %v", integration.ErrPlatformInvalidResponse, err)
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("%w: %s - %s", integration.ErrPlatformRequestFailed, resp.ErrorResponse.Code, resp.ErrorResponse.Msg)
	}

	if resp.LogisticsOfflineSendResponse != nil &&
		resp.LogisticsOfflineSendResponse.Shipping != nil &&
		!resp.LogisticsOfflineSendResponse.Shipping.IsSuccess {
		return integration.ErrOrderSyncStatusUpdateFailed
	}

	return nil
}

// ---------------------------------------------------------------------------
// Inventory Operations
// ---------------------------------------------------------------------------

// SyncInventory synchronizes inventory levels to Taobao platform
func (a *TaobaoAdapter) SyncInventory(ctx context.Context, tenantID uuid.UUID, items []integration.InventorySync) (*integration.SyncResult, error) {
	config, err := a.getTenantConfig(tenantID)
	if err != nil {
		return nil, err
	}

	result := &integration.SyncResult{
		Status:       integration.SyncStatusInProgress,
		TotalCount:   len(items),
		SuccessCount: 0,
		FailedCount:  0,
		FailedItems:  make([]integration.SyncFailure, 0),
		SyncedAt:     time.Now(),
	}

	for _, item := range items {
		var err error

		if item.PlatformSkuID != "" {
			// Update SKU-level inventory
			err = a.updateSkuQuantity(ctx, config, item.PlatformProductID, item.PlatformSkuID, item.AvailableQuantity)
		} else {
			// Update item-level inventory
			err = a.updateItemQuantity(ctx, config, item.PlatformProductID, item.AvailableQuantity)
		}

		if err != nil {
			result.FailedCount++
			result.FailedItems = append(result.FailedItems, integration.SyncFailure{
				ItemID:       item.PlatformProductID,
				ErrorMessage: err.Error(),
			})
			continue
		}
		result.SuccessCount++
	}

	// Set final status
	if result.FailedCount == 0 {
		result.Status = integration.SyncStatusSuccess
	} else if result.SuccessCount > 0 {
		result.Status = integration.SyncStatusPartial
	} else {
		result.Status = integration.SyncStatusFailed
	}

	return result, nil
}

// updateItemQuantity updates item-level inventory on Taobao
func (a *TaobaoAdapter) updateItemQuantity(ctx context.Context, config *TaobaoConfig, numIid string, quantity decimal.Decimal) error {
	params := map[string]string{
		"method":   "taobao.item.quantity.update",
		"num_iid":  numIid,
		"quantity": strconv.FormatInt(quantity.IntPart(), 10),
	}

	respBody, err := a.doRequest(ctx, config, params)
	if err != nil {
		return err
	}

	var resp TaobaoItemQuantityUpdateResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return fmt.Errorf("%w: failed to parse response: %v", integration.ErrPlatformInvalidResponse, err)
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("%w: %s - %s", integration.ErrPlatformRequestFailed, resp.ErrorResponse.Code, resp.ErrorResponse.Msg)
	}

	return nil
}

// updateSkuQuantity updates SKU-level inventory on Taobao
func (a *TaobaoAdapter) updateSkuQuantity(ctx context.Context, config *TaobaoConfig, numIid, skuId string, quantity decimal.Decimal) error {
	params := map[string]string{
		"method":   "taobao.item.sku.update",
		"num_iid":  numIid,
		"sku_id":   skuId,
		"quantity": strconv.FormatInt(quantity.IntPart(), 10),
	}

	respBody, err := a.doRequest(ctx, config, params)
	if err != nil {
		return err
	}

	var resp TaobaoSkuQuantityUpdateResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return fmt.Errorf("%w: failed to parse response: %v", integration.ErrPlatformInvalidResponse, err)
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("%w: %s - %s", integration.ErrPlatformRequestFailed, resp.ErrorResponse.Code, resp.ErrorResponse.Msg)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Internal Helpers
// ---------------------------------------------------------------------------

// doRequest performs an HTTP request to Taobao API
func (a *TaobaoAdapter) doRequest(ctx context.Context, config *TaobaoConfig, params map[string]string) ([]byte, error) {
	// Add common parameters
	params["app_key"] = config.AppKey
	params["session"] = config.SessionKey
	params["timestamp"] = time.Now().Format("2006-01-02 15:04:05")
	params["format"] = "json"
	params["v"] = "2.0"
	params["sign_method"] = "md5"

	// Generate signature
	params["sign"] = config.Sign(params)

	// Build URL-encoded body
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", config.APIBaseURL, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, fmt.Errorf("taobao: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", integration.ErrPlatformUnavailable, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("taobao: failed to read response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: HTTP %d", integration.ErrPlatformRequestFailed, resp.StatusCode)
	}

	return body, nil
}

// convertTaobaoTradeToPlatformOrder converts a Taobao trade to PlatformOrder
func (a *TaobaoAdapter) convertTaobaoTradeToPlatformOrder(trade *TaobaoTrade) integration.PlatformOrder {
	order := integration.PlatformOrder{
		PlatformOrderID:    strconv.FormatInt(trade.Tid, 10),
		PlatformCode:       integration.PlatformCodeTaobao,
		Status:             mapTaobaoOrderStatusToPlatformStatus(trade.Status),
		BuyerNickname:      trade.BuyerNick,
		BuyerPhone:         trade.ReceiverMobile,
		ReceiverName:       trade.ReceiverName,
		ReceiverPhone:      trade.ReceiverMobile,
		ReceiverProvince:   trade.ReceiverState,
		ReceiverCity:       trade.ReceiverCity,
		ReceiverDistrict:   trade.ReceiverDistrict,
		ReceiverAddress:    trade.ReceiverAddress,
		ReceiverPostalCode: trade.ReceiverZip,
		TotalAmount:        ParseDecimal(trade.Payment),
		ProductAmount:      ParseDecimal(trade.TotalFee),
		FreightAmount:      ParseDecimal(trade.PostFee),
		DiscountAmount:     ParseDecimal(trade.DiscountFee),
		Currency:           "CNY",
		Items:              make([]integration.PlatformOrderItem, 0),
		BuyerMessage:       trade.BuyerMessage,
		SellerMemo:         trade.SellerMemo,
	}

	// Parse timestamps
	if trade.Created != "" {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", trade.Created, time.Local); err == nil {
			order.CreatedAt = t
		}
	}
	if trade.PayTime != "" {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", trade.PayTime, time.Local); err == nil {
			order.PaidAt = &t
		}
	}
	if trade.ConsignTime != "" {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", trade.ConsignTime, time.Local); err == nil {
			order.ShippedAt = &t
		}
	}
	if trade.EndTime != "" {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", trade.EndTime, time.Local); err == nil {
			order.CompletedAt = &t
		}
	}

	// Convert order items
	if trade.Orders != nil {
		for _, item := range trade.Orders.Order {
			orderItem := integration.PlatformOrderItem{
				PlatformItemID:    strconv.FormatInt(item.Oid, 10),
				PlatformProductID: strconv.FormatInt(item.NumIid, 10),
				PlatformSkuID:     item.SkuID,
				ProductName:       item.Title,
				SkuName:           item.SkuPropertiesName,
				ImageURL:          item.PicPath,
				Quantity:          decimal.NewFromInt(item.Num),
				UnitPrice:         ParseDecimal(item.Price),
				TotalPrice:        ParseDecimal(item.TotalFee),
				DiscountAmount:    ParseDecimal(item.DiscountFee),
			}
			order.Items = append(order.Items, orderItem)
		}
	} else if trade.NumIid > 0 {
		// Single item order (no nested orders structure)
		orderItem := integration.PlatformOrderItem{
			PlatformProductID: strconv.FormatInt(trade.NumIid, 10),
			ProductName:       trade.Title,
			ImageURL:          trade.PicPath,
			Quantity:          decimal.NewFromInt(trade.Num),
			UnitPrice:         ParseDecimal(trade.Price),
			TotalPrice:        ParseDecimal(trade.TotalFee),
		}
		order.Items = append(order.Items, orderItem)
	}

	// Store raw data as JSON
	if rawBytes, err := json.Marshal(trade); err == nil {
		order.RawData = string(rawBytes)
	}

	return order
}

// ---------------------------------------------------------------------------
// Status Mapping
// ---------------------------------------------------------------------------

// mapTaobaoOrderStatusToPlatformStatus maps Taobao order status to platform status
func mapTaobaoOrderStatusToPlatformStatus(status string) integration.PlatformOrderStatus {
	switch status {
	case "WAIT_BUYER_PAY":
		return integration.PlatformOrderStatusPending
	case "WAIT_SELLER_SEND_GOODS":
		return integration.PlatformOrderStatusPaid
	case "WAIT_BUYER_CONFIRM_GOODS":
		return integration.PlatformOrderStatusShipped
	case "TRADE_BUYER_SIGNED":
		return integration.PlatformOrderStatusDelivered
	case "TRADE_FINISHED":
		return integration.PlatformOrderStatusCompleted
	case "TRADE_CLOSED":
		return integration.PlatformOrderStatusClosed
	case "TRADE_CLOSED_BY_TAOBAO":
		return integration.PlatformOrderStatusCancelled
	default:
		return integration.PlatformOrderStatusPending
	}
}

// mapToTaobaoOrderStatus maps platform status to Taobao status
func mapToTaobaoOrderStatus(status integration.PlatformOrderStatus) string {
	switch status {
	case integration.PlatformOrderStatusPending:
		return "WAIT_BUYER_PAY"
	case integration.PlatformOrderStatusPaid:
		return "WAIT_SELLER_SEND_GOODS"
	case integration.PlatformOrderStatusShipped:
		return "WAIT_BUYER_CONFIRM_GOODS"
	case integration.PlatformOrderStatusDelivered:
		return "TRADE_BUYER_SIGNED"
	case integration.PlatformOrderStatusCompleted:
		return "TRADE_FINISHED"
	case integration.PlatformOrderStatusClosed:
		return "TRADE_CLOSED"
	case integration.PlatformOrderStatusCancelled:
		return "TRADE_CLOSED_BY_TAOBAO"
	default:
		return ""
	}
}

// mapShippingCompanyToTaobaoCode maps common shipping company names to Taobao codes
func mapShippingCompanyToTaobaoCode(company string) string {
	// Common Chinese shipping companies
	companyMap := map[string]string{
		"顺丰":   "SF",
		"顺丰速运": "SF",
		"圆通":   "YTO",
		"圆通速递": "YTO",
		"中通":   "ZTO",
		"中通快递": "ZTO",
		"申通":   "STO",
		"申通快递": "STO",
		"韵达":   "YUNDA",
		"韵达快递": "YUNDA",
		"邮政":   "POSTB",
		"EMS":  "EMS",
		"ems":  "EMS",
		"京东":   "JD",
		"京东物流": "JD",
		"百世":   "HTKY",
		"百世快递": "HTKY",
		"德邦":   "DBL",
		"德邦物流": "DBL",
		"极兔":   "JTSD",
		"极兔速递": "JTSD",
		"菜鸟":   "CAINIAO",
	}

	// Try direct match
	if code, ok := companyMap[company]; ok {
		return code
	}

	// Try case-insensitive match
	upperCompany := strings.ToUpper(company)
	for name, code := range companyMap {
		if strings.Contains(upperCompany, strings.ToUpper(name)) {
			return code
		}
	}

	// Default to OTHER if not found
	return "OTHER"
}

// taobaoTradeFields is the comma-separated list of fields to request from Taobao API
const taobaoTradeFields = "tid,status,type,buyer_nick,buyer_open_uid,created,modified,pay_time,consign_time,end_time," +
	"payment,total_fee,post_fee,discount_fee,adjust_fee,buyer_rate,seller_rate," +
	"receiver_name,receiver_state,receiver_city,receiver_district,receiver_address,receiver_zip,receiver_mobile,receiver_phone," +
	"shipping_type,sid,company_code,buyer_memo,seller_memo,buyer_message," +
	"orders.oid,orders.num_iid,orders.sku_id,orders.title,orders.sku_properties_name,orders.price,orders.num,orders.total_fee," +
	"orders.payment,orders.discount_fee,orders.adjust_fee,orders.refund_status,orders.refund_id,orders.status,orders.pic_path," +
	"orders.outer_iid,orders.outer_sku_id"

// Ensure TaobaoAdapter implements EcommercePlatform interface
var _ integration.EcommercePlatform = (*TaobaoAdapter)(nil)
