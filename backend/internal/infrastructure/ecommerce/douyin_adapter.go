package ecommerce

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/erp/backend/internal/domain/integration"
)

// Constants for Douyin API
const (
	// maxDouyinResponseSize limits the response body size to prevent memory exhaustion
	maxDouyinResponseSize = 10 * 1024 * 1024 // 10MB max response
	// centsPerYuan is the conversion factor for Chinese currency
	centsPerYuan = 100
)

// DouyinAdapter implements EcommercePlatform interface for Douyin (TikTok Shop) platform
type DouyinAdapter struct {
	config     *DouyinConfig
	httpClient *http.Client

	// tenantConfigs stores per-tenant configurations
	// In production, this would be loaded from database
	tenantConfigs map[uuid.UUID]*DouyinConfig
	mu            sync.RWMutex // Protects tenantConfigs map access
}

// NewDouyinAdapter creates a new Douyin adapter with the given configuration
func NewDouyinAdapter(config *DouyinConfig) (*DouyinAdapter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &DouyinAdapter{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.TimeoutSeconds) * time.Second,
		},
		tenantConfigs: make(map[uuid.UUID]*DouyinConfig),
	}, nil
}

// SetTenantConfig sets the configuration for a specific tenant
func (a *DouyinAdapter) SetTenantConfig(tenantID uuid.UUID, config *DouyinConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tenantConfigs[tenantID] = config
	return nil
}

// getTenantConfig retrieves the configuration for a tenant
func (a *DouyinAdapter) getTenantConfig(tenantID uuid.UUID) (*DouyinConfig, error) {
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

// PlatformCode returns the platform code this adapter handles
func (a *DouyinAdapter) PlatformCode() integration.PlatformCode {
	return integration.PlatformCodeDouyin
}

// IsEnabled returns true if this platform is enabled for the tenant
func (a *DouyinAdapter) IsEnabled(ctx context.Context, tenantID uuid.UUID) (bool, error) {
	_, err := a.getTenantConfig(tenantID)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// ---------------------------------------------------------------------------
// Product Operations
// ---------------------------------------------------------------------------

// SyncProducts synchronizes products to the Douyin platform
func (a *DouyinAdapter) SyncProducts(ctx context.Context, tenantID uuid.UUID, products []integration.ProductSync) (*integration.SyncResult, error) {
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

// updateProductOnPlatform updates a product on Douyin (placeholder - full implementation would use product.edit API)
func (a *DouyinAdapter) updateProductOnPlatform(_ context.Context, _ *DouyinConfig, _ *integration.ProductSync) error {
	// Douyin product update requires complex parameters
	// This is a simplified implementation focusing on price/quantity updates
	// Full implementation would use /product/editV2 API

	// For now, we only update inventory through the dedicated inventory sync
	// Product title/price changes require the full product edit API
	return nil
}

// GetProduct retrieves a product from Douyin platform
func (a *DouyinAdapter) GetProduct(ctx context.Context, tenantID uuid.UUID, platformProductID string) (*integration.ProductSync, error) {
	config, err := a.getTenantConfig(tenantID)
	if err != nil {
		return nil, err
	}

	productID, err := strconv.ParseInt(platformProductID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("douyin: invalid product ID: %w", err)
	}

	params := map[string]any{
		"product_id": productID,
	}

	respBody, err := a.doRequest(ctx, config, "/product/detail", params)
	if err != nil {
		return nil, err
	}

	var resp DouyinProductDetailResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("douyin: failed to parse response: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("douyin: %d - %s", resp.ErrNo, resp.Message)
	}

	if resp.Data == nil || resp.Data.Product == nil {
		return nil, integration.ErrProductSyncMappingNotFound
	}

	item := resp.Data.Product
	product := &integration.ProductSync{
		PlatformProductID: strconv.FormatInt(item.ProductID, 10),
		ProductCode:       item.OutProductID,
		ProductName:       item.Name,
		Description:       item.Description,
		// Price in cents, convert to yuan
		Price:         decimal.NewFromInt(item.DiscountPrice).Div(decimal.NewFromInt(centsPerYuan)),
		OriginalPrice: decimal.NewFromInt(item.MarketPrice).Div(decimal.NewFromInt(centsPerYuan)),
		IsOnSale:      item.Status == 0, // 0 = online, 1 = offline
		SKUs:          make([]integration.ProductSkuSync, 0),
	}

	// Main image
	if item.Img != "" {
		product.ImageURLs = append(product.ImageURLs, item.Img)
	}

	// Parse SKUs
	var totalQuantity int64
	for _, sku := range item.SkuList {
		totalQuantity += sku.StockNum

		// Build SKU name from spec details
		var specNames []string
		for _, spec := range sku.SpecDetail {
			specNames = append(specNames, fmt.Sprintf("%s:%s", spec.SpecName, spec.ValueName))
		}

		product.SKUs = append(product.SKUs, integration.ProductSkuSync{
			PlatformSkuID: strconv.FormatInt(sku.SkuID, 10),
			SkuCode:       sku.OutSkuID,
			SkuName:       strings.Join(specNames, ";"),
			Price:         decimal.NewFromInt(sku.Price).Div(decimal.NewFromInt(centsPerYuan)),
			Quantity:      decimal.NewFromInt(sku.StockNum),
		})
	}
	product.Quantity = decimal.NewFromInt(totalQuantity)

	return product, nil
}

// ---------------------------------------------------------------------------
// Order Operations
// ---------------------------------------------------------------------------

// PullOrders pulls orders from Douyin platform within the specified time range
func (a *DouyinAdapter) PullOrders(ctx context.Context, req *integration.OrderPullRequest) (*integration.OrderPullResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	config, err := a.getTenantConfig(req.TenantID)
	if err != nil {
		return nil, err
	}

	// Build API parameters
	params := map[string]any{
		"start_time": req.StartTime.Unix(),
		"end_time":   req.EndTime.Unix(),
		"page":       req.PageNo - 1, // Douyin uses 0-indexed page
		"size":       req.PageSize,
		"order_by":   "create_time",
		"is_desc":    "1",
	}

	// Add status filter if specified
	if req.Status != nil {
		params["order_status"] = mapToDouyinOrderStatus(*req.Status)
	}

	respBody, err := a.doRequest(ctx, config, "/order/searchList", params)
	if err != nil {
		return nil, err
	}

	var resp DouyinOrderListResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("douyin: failed to parse response: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("douyin: %d - %s", resp.ErrNo, resp.Message)
	}

	if resp.Data == nil {
		return nil, integration.ErrPlatformInvalidResponse
	}

	// Convert to domain models
	response := &integration.OrderPullResponse{
		Orders:     make([]integration.PlatformOrder, 0),
		TotalCount: resp.Data.Total,
		HasMore:    int64(req.PageNo*req.PageSize) < resp.Data.Total,
		NextPageNo: req.PageNo + 1,
	}

	for _, order := range resp.Data.List {
		platformOrder := a.convertDouyinOrderToPlatformOrder(&order)
		response.Orders = append(response.Orders, platformOrder)
	}

	return response, nil
}

// GetOrder retrieves a single order from Douyin platform
func (a *DouyinAdapter) GetOrder(ctx context.Context, tenantID uuid.UUID, platformOrderID string) (*integration.PlatformOrder, error) {
	if platformOrderID == "" {
		return nil, integration.ErrOrderSyncOrderNotFound
	}

	config, err := a.getTenantConfig(tenantID)
	if err != nil {
		return nil, err
	}

	params := map[string]any{
		"shop_order_id": platformOrderID,
	}

	respBody, err := a.doRequest(ctx, config, "/order/orderDetail", params)
	if err != nil {
		return nil, err
	}

	var resp DouyinOrderDetailResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("douyin: failed to parse response: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("douyin: %d - %s", resp.ErrNo, resp.Message)
	}

	if resp.Data == nil || resp.Data.ShopOrderDetail == nil {
		return nil, integration.ErrOrderSyncOrderNotFound
	}

	order := a.convertDouyinOrderToPlatformOrder(resp.Data.ShopOrderDetail)
	return &order, nil
}

// UpdateOrderStatus updates the order status on Douyin platform
func (a *DouyinAdapter) UpdateOrderStatus(ctx context.Context, req *integration.OrderStatusUpdateRequest) error {
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

	// Other status updates are typically not supported by Douyin API
	// (orders transition through natural workflow on the platform)
	return nil
}

// sendShipment sends shipment information to Douyin
func (a *DouyinAdapter) sendShipment(ctx context.Context, config *DouyinConfig, req *integration.OrderStatusUpdateRequest) error {
	// Map shipping company name to Douyin code
	companyCode := mapShippingCompanyToDouyinCode(req.ShippingCompany)

	params := map[string]any{
		"order_id":     req.PlatformOrderID,
		"logistics_id": companyCode,
		"company":      req.ShippingCompany,
		"tracking_no":  req.TrackingNumber,
	}

	respBody, err := a.doRequest(ctx, config, "/order/logisticsAdd", params)
	if err != nil {
		return err
	}

	var resp DouyinShipResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return fmt.Errorf("douyin: failed to parse response: %w", err)
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("douyin: %d - %s", resp.ErrNo, resp.Message)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Inventory Operations
// ---------------------------------------------------------------------------

// SyncInventory synchronizes inventory levels to Douyin platform
func (a *DouyinAdapter) SyncInventory(ctx context.Context, tenantID uuid.UUID, items []integration.InventorySync) (*integration.SyncResult, error) {
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
		err := a.updateSkuStock(ctx, config, &item)
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

// updateSkuStock updates SKU stock on Douyin
func (a *DouyinAdapter) updateSkuStock(ctx context.Context, config *DouyinConfig, item *integration.InventorySync) error {
	skuID, err := strconv.ParseInt(item.PlatformSkuID, 10, 64)
	if err != nil {
		// Try using product ID if SKU ID is not provided
		skuID, err = strconv.ParseInt(item.PlatformProductID, 10, 64)
		if err != nil {
			return fmt.Errorf("douyin: invalid SKU ID: %w", err)
		}
	}

	params := map[string]any{
		"sku_id":    skuID,
		"stock_num": item.AvailableQuantity.IntPart(),
	}

	respBody, err := a.doRequest(ctx, config, "/sku/stockNum", params)
	if err != nil {
		return err
	}

	var resp DouyinStockUpdateResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return fmt.Errorf("douyin: failed to parse response: %w", err)
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("douyin: %d - %s", resp.ErrNo, resp.Message)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Internal Helpers
// ---------------------------------------------------------------------------

// doRequest performs an HTTP request to Douyin API
func (a *DouyinAdapter) doRequest(ctx context.Context, config *DouyinConfig, method string, params map[string]any) ([]byte, error) {
	// Serialize params to JSON
	paramJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("douyin: failed to marshal params: %w", err)
	}

	// Prepare common parameters
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	version := "2"

	// Generate signature
	sign := config.Sign(method, string(paramJSON), timestamp, version)

	// Build URL
	url := fmt.Sprintf("%s%s", config.APIBaseURL, method)

	// Build request body
	requestBody := map[string]any{
		"app_key":      config.AppKey,
		"access_token": config.AccessToken,
		"method":       method,
		"param_json":   string(paramJSON),
		"timestamp":    timestamp,
		"v":            version,
		"sign":         sign,
		"sign_method":  "hmac-sha256",
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("douyin: failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("douyin: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", integration.ErrPlatformUnavailable, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDouyinResponseSize))
	if err != nil {
		return nil, fmt.Errorf("douyin: failed to read response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: HTTP %d", integration.ErrPlatformRequestFailed, resp.StatusCode)
	}

	return body, nil
}

// convertDouyinOrderToPlatformOrder converts a Douyin order to PlatformOrder
func (a *DouyinAdapter) convertDouyinOrderToPlatformOrder(order *DouyinOrder) integration.PlatformOrder {
	platformOrder := integration.PlatformOrder{
		PlatformOrderID:  order.OrderID,
		PlatformCode:     integration.PlatformCodeDouyin,
		Status:           mapDouyinOrderStatusToPlatformStatus(order.OrderStatus),
		BuyerNickname:    order.BuyerOpenUID,
		TotalAmount:      decimal.NewFromInt(order.PayAmount).Div(decimal.NewFromInt(centsPerYuan)),
		ProductAmount:    decimal.NewFromInt(order.OrderAmount).Div(decimal.NewFromInt(centsPerYuan)),
		FreightAmount:    decimal.NewFromInt(order.PostAmount).Div(decimal.NewFromInt(centsPerYuan)),
		DiscountAmount:   decimal.NewFromInt(order.CouponAmount).Div(decimal.NewFromInt(centsPerYuan)),
		PlatformDiscount: decimal.NewFromInt(order.PlatformDiscount).Div(decimal.NewFromInt(centsPerYuan)),
		SellerDiscount:   decimal.NewFromInt(order.ShopCouponAmount).Div(decimal.NewFromInt(centsPerYuan)),
		Currency:         "CNY",
		Items:            make([]integration.PlatformOrderItem, 0),
		BuyerMessage:     order.BuyerWords,
		SellerMemo:       order.SellerWords,
	}

	// Parse timestamps (Unix milliseconds)
	if order.CreateTime > 0 {
		platformOrder.CreatedAt = time.Unix(order.CreateTime, 0)
	}
	if order.PayTime > 0 {
		paidAt := time.Unix(order.PayTime, 0)
		platformOrder.PaidAt = &paidAt
	}
	if order.FinishTime > 0 {
		completedAt := time.Unix(order.FinishTime, 0)
		platformOrder.CompletedAt = &completedAt
	}

	// Parse receiver info
	if order.PostReceiver != nil {
		receiver := order.PostReceiver
		platformOrder.ReceiverName = receiver.Name
		platformOrder.ReceiverPhone = receiver.Phone
		platformOrder.ReceiverProvince = receiver.Province
		platformOrder.ReceiverCity = receiver.City
		platformOrder.ReceiverDistrict = receiver.Town
		platformOrder.ReceiverAddress = receiver.Street + " " + receiver.Detail
		platformOrder.ReceiverPostalCode = receiver.PostCode
		platformOrder.BuyerPhone = receiver.Phone
	}

	// Parse logistics info for shipped time
	if order.LogisticsInfo != nil && order.LogisticsInfo.ShipTime > 0 {
		shippedAt := time.Unix(order.LogisticsInfo.ShipTime, 0)
		platformOrder.ShippedAt = &shippedAt
	}

	// Convert order items
	for _, item := range order.SkuOrderList {
		// Calculate unit price safely (avoid division by zero)
		var unitPrice decimal.Decimal
		if item.ItemNum > 0 {
			unitPrice = decimal.NewFromInt(item.OriginAmount).Div(decimal.NewFromInt(centsPerYuan * int64(item.ItemNum)))
		} else {
			unitPrice = decimal.Zero
		}

		orderItem := integration.PlatformOrderItem{
			PlatformItemID:    item.SkuOrderID,
			PlatformProductID: strconv.FormatInt(item.ProductID, 10),
			PlatformSkuID:     strconv.FormatInt(item.SkuID, 10),
			ProductName:       item.ProductName,
			SkuName:           item.SkuSpec,
			ImageURL:          item.ProductPic,
			Quantity:          decimal.NewFromInt(int64(item.ItemNum)),
			UnitPrice:         unitPrice,
			TotalPrice:        decimal.NewFromInt(item.OriginAmount).Div(decimal.NewFromInt(centsPerYuan)),
			DiscountAmount:    decimal.NewFromInt(item.CouponAmount).Div(decimal.NewFromInt(centsPerYuan)),
		}
		platformOrder.Items = append(platformOrder.Items, orderItem)
	}

	// Store raw data as JSON
	if rawBytes, err := json.Marshal(order); err == nil {
		platformOrder.RawData = string(rawBytes)
	}

	return platformOrder
}

// ---------------------------------------------------------------------------
// Status Mapping
// ---------------------------------------------------------------------------

// mapDouyinOrderStatusToPlatformStatus maps Douyin order status to platform status
func mapDouyinOrderStatusToPlatformStatus(status int) integration.PlatformOrderStatus {
	switch status {
	case DouyinOrderStatusPendingPayment:
		return integration.PlatformOrderStatusPending
	case DouyinOrderStatusPendingShipment:
		return integration.PlatformOrderStatusPaid
	case DouyinOrderStatusShipped:
		return integration.PlatformOrderStatusShipped
	case DouyinOrderStatusCompleted:
		return integration.PlatformOrderStatusCompleted
	case DouyinOrderStatusCancelled:
		return integration.PlatformOrderStatusCancelled
	case DouyinOrderStatusRefunding:
		return integration.PlatformOrderStatusRefunding
	case DouyinOrderStatusRefunded:
		return integration.PlatformOrderStatusRefunded
	default:
		return integration.PlatformOrderStatusPending
	}
}

// mapToDouyinOrderStatus maps platform status to Douyin status
func mapToDouyinOrderStatus(status integration.PlatformOrderStatus) int {
	switch status {
	case integration.PlatformOrderStatusPending:
		return DouyinOrderStatusPendingPayment
	case integration.PlatformOrderStatusPaid:
		return DouyinOrderStatusPendingShipment
	case integration.PlatformOrderStatusShipped:
		return DouyinOrderStatusShipped
	case integration.PlatformOrderStatusCompleted:
		return DouyinOrderStatusCompleted
	case integration.PlatformOrderStatusCancelled:
		return DouyinOrderStatusCancelled
	case integration.PlatformOrderStatusRefunding:
		return DouyinOrderStatusRefunding
	case integration.PlatformOrderStatusRefunded:
		return DouyinOrderStatusRefunded
	default:
		return 0
	}
}

// mapShippingCompanyToDouyinCode maps common shipping company names to Douyin codes
func mapShippingCompanyToDouyinCode(company string) string {
	// Common Chinese shipping companies and their Douyin codes
	companyMap := map[string]string{
		"顺丰":   "shunfeng",
		"顺丰速运": "shunfeng",
		"圆通":   "yuantong",
		"圆通速递": "yuantong",
		"中通":   "zhongtong",
		"中通快递": "zhongtong",
		"申通":   "shentong",
		"申通快递": "shentong",
		"韵达":   "yunda",
		"韵达快递": "yunda",
		"邮政":   "ems",
		"EMS":  "ems",
		"ems":  "ems",
		"京东":   "jd",
		"京东物流": "jd",
		"百世":   "huitongkuaidi",
		"百世快递": "huitongkuaidi",
		"德邦":   "debangkuaidi",
		"德邦物流": "debangkuaidi",
		"极兔":   "jtexpress",
		"极兔速递": "jtexpress",
		"菜鸟":   "cainiaowuliu",
		"丹鸟":   "danniao",
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
	return "other"
}

// Ensure DouyinAdapter implements EcommercePlatform interface
var _ integration.EcommercePlatform = (*DouyinAdapter)(nil)
