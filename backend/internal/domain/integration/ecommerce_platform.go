package integration

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ---------------------------------------------------------------------------
// EcommercePlatform Errors
// ---------------------------------------------------------------------------

var (
	// Platform errors
	ErrPlatformNotConfigured    = errors.New("integration: platform not configured")
	ErrPlatformNotEnabled       = errors.New("integration: platform not enabled")
	ErrPlatformUnavailable      = errors.New("integration: platform temporarily unavailable")
	ErrPlatformRequestFailed    = errors.New("integration: platform request failed")
	ErrPlatformInvalidResponse  = errors.New("integration: invalid platform response")
	ErrPlatformAuthFailed       = errors.New("integration: platform authentication failed")
	ErrPlatformTokenExpired     = errors.New("integration: platform token expired")
	ErrPlatformRateLimited      = errors.New("integration: platform rate limited")
	ErrPlatformInvalidSignature = errors.New("integration: invalid platform signature")

	// Product sync errors
	ErrProductSyncInvalidProduct  = errors.New("integration: invalid product for sync")
	ErrProductSyncMappingNotFound = errors.New("integration: product mapping not found")
	ErrProductSyncFailed          = errors.New("integration: product sync failed")

	// Order sync errors
	ErrOrderSyncInvalidOrder       = errors.New("integration: invalid order for sync")
	ErrOrderSyncOrderNotFound      = errors.New("integration: platform order not found")
	ErrOrderSyncDuplicateOrder     = errors.New("integration: order already synced")
	ErrOrderSyncInvalidStatus      = errors.New("integration: invalid order status transition")
	ErrOrderSyncStatusUpdateFailed = errors.New("integration: order status update failed")

	// Inventory sync errors
	ErrInventorySyncFailed = errors.New("integration: inventory sync failed")

	// Mapping errors
	ErrMappingInvalidTenantID        = errors.New("integration: invalid tenant ID")
	ErrMappingInvalidProductID       = errors.New("integration: invalid product ID")
	ErrMappingInvalidPlatformCode    = errors.New("integration: invalid platform code")
	ErrMappingInvalidPlatformID      = errors.New("integration: invalid platform product ID")
	ErrMappingAlreadyExists          = errors.New("integration: product mapping already exists")
	ErrMappingNotFound               = errors.New("integration: product mapping not found")
	ErrMappingSkuMappingInvalid      = errors.New("integration: invalid SKU mapping")
	ErrMappingPlatformCodeNotAllowed = errors.New("integration: platform code not allowed for tenant")
)

// ---------------------------------------------------------------------------
// PlatformCode represents the type of e-commerce platform
// ---------------------------------------------------------------------------

// PlatformCode represents the type of e-commerce platform
type PlatformCode string

const (
	// PlatformCodeTaobao represents Taobao/Tmall platform
	PlatformCodeTaobao PlatformCode = "TAOBAO"
	// PlatformCodeJD represents JD.com platform
	PlatformCodeJD PlatformCode = "JD"
	// PlatformCodePDD represents Pinduoduo platform
	PlatformCodePDD PlatformCode = "PDD"
	// PlatformCodeDouyin represents Douyin/TikTok shop platform
	PlatformCodeDouyin PlatformCode = "DOUYIN"
	// PlatformCodeWechat represents WeChat Mini Program shop
	PlatformCodeWechat PlatformCode = "WECHAT"
	// PlatformCodeKuaishou represents Kuaishou shop platform
	PlatformCodeKuaishou PlatformCode = "KUAISHOU"
)

// IsValid returns true if the platform code is valid
func (c PlatformCode) IsValid() bool {
	switch c {
	case PlatformCodeTaobao, PlatformCodeJD, PlatformCodePDD,
		PlatformCodeDouyin, PlatformCodeWechat, PlatformCodeKuaishou:
		return true
	default:
		return false
	}
}

// String returns the string representation of PlatformCode
func (c PlatformCode) String() string {
	return string(c)
}

// DisplayName returns a human-readable name for the platform
func (c PlatformCode) DisplayName() string {
	switch c {
	case PlatformCodeTaobao:
		return "淘宝/天猫"
	case PlatformCodeJD:
		return "京东"
	case PlatformCodePDD:
		return "拼多多"
	case PlatformCodeDouyin:
		return "抖音"
	case PlatformCodeWechat:
		return "微信小程序"
	case PlatformCodeKuaishou:
		return "快手"
	default:
		return string(c)
	}
}

// ---------------------------------------------------------------------------
// PlatformOrderStatus represents the status of an order on the platform
// ---------------------------------------------------------------------------

// PlatformOrderStatus represents the status of an order on the platform
type PlatformOrderStatus string

const (
	// PlatformOrderStatusPending indicates order is pending payment
	PlatformOrderStatusPending PlatformOrderStatus = "PENDING"
	// PlatformOrderStatusPaid indicates payment received, pending shipment
	PlatformOrderStatusPaid PlatformOrderStatus = "PAID"
	// PlatformOrderStatusShipped indicates order has been shipped
	PlatformOrderStatusShipped PlatformOrderStatus = "SHIPPED"
	// PlatformOrderStatusDelivered indicates order delivered
	PlatformOrderStatusDelivered PlatformOrderStatus = "DELIVERED"
	// PlatformOrderStatusCompleted indicates order completed (buyer confirmed)
	PlatformOrderStatusCompleted PlatformOrderStatus = "COMPLETED"
	// PlatformOrderStatusCancelled indicates order was cancelled
	PlatformOrderStatusCancelled PlatformOrderStatus = "CANCELLED"
	// PlatformOrderStatusRefunding indicates refund in progress
	PlatformOrderStatusRefunding PlatformOrderStatus = "REFUNDING"
	// PlatformOrderStatusRefunded indicates order was refunded
	PlatformOrderStatusRefunded PlatformOrderStatus = "REFUNDED"
	// PlatformOrderStatusClosed indicates order was closed
	PlatformOrderStatusClosed PlatformOrderStatus = "CLOSED"
)

// IsValid returns true if the status is valid
func (s PlatformOrderStatus) IsValid() bool {
	switch s {
	case PlatformOrderStatusPending, PlatformOrderStatusPaid, PlatformOrderStatusShipped,
		PlatformOrderStatusDelivered, PlatformOrderStatusCompleted, PlatformOrderStatusCancelled,
		PlatformOrderStatusRefunding, PlatformOrderStatusRefunded, PlatformOrderStatusClosed:
		return true
	default:
		return false
	}
}

// String returns the string representation of PlatformOrderStatus
func (s PlatformOrderStatus) String() string {
	return string(s)
}

// IsFinal returns true if the status is a final (terminal) state
func (s PlatformOrderStatus) IsFinal() bool {
	switch s {
	case PlatformOrderStatusCompleted, PlatformOrderStatusCancelled,
		PlatformOrderStatusRefunded, PlatformOrderStatusClosed:
		return true
	default:
		return false
	}
}

// RequiresShipment returns true if the status requires shipment processing
func (s PlatformOrderStatus) RequiresShipment() bool {
	return s == PlatformOrderStatusPaid
}

// ---------------------------------------------------------------------------
// SyncStatus represents the synchronization status
// ---------------------------------------------------------------------------

// SyncStatus represents the synchronization status
type SyncStatus string

const (
	// SyncStatusPending indicates sync is pending
	SyncStatusPending SyncStatus = "PENDING"
	// SyncStatusInProgress indicates sync is in progress
	SyncStatusInProgress SyncStatus = "IN_PROGRESS"
	// SyncStatusSuccess indicates sync was successful
	SyncStatusSuccess SyncStatus = "SUCCESS"
	// SyncStatusPartial indicates partial sync success
	SyncStatusPartial SyncStatus = "PARTIAL"
	// SyncStatusFailed indicates sync failed
	SyncStatusFailed SyncStatus = "FAILED"
)

// IsValid returns true if the status is valid
func (s SyncStatus) IsValid() bool {
	switch s {
	case SyncStatusPending, SyncStatusInProgress, SyncStatusSuccess, SyncStatusPartial, SyncStatusFailed:
		return true
	default:
		return false
	}
}

// String returns the string representation of SyncStatus
func (s SyncStatus) String() string {
	return string(s)
}

// ---------------------------------------------------------------------------
// Value Objects
// ---------------------------------------------------------------------------

// PlatformOrder represents an order from an external e-commerce platform
type PlatformOrder struct {
	// PlatformOrderID is the order ID on the platform
	PlatformOrderID string
	// PlatformCode identifies which platform this order is from
	PlatformCode PlatformCode
	// Status is the current order status on the platform
	Status PlatformOrderStatus
	// BuyerNickname is the buyer's platform nickname
	BuyerNickname string
	// BuyerPhone is the buyer's phone number
	BuyerPhone string
	// ReceiverName is the recipient's name
	ReceiverName string
	// ReceiverPhone is the recipient's phone number
	ReceiverPhone string
	// ReceiverProvince is the delivery province
	ReceiverProvince string
	// ReceiverCity is the delivery city
	ReceiverCity string
	// ReceiverDistrict is the delivery district
	ReceiverDistrict string
	// ReceiverAddress is the detailed delivery address
	ReceiverAddress string
	// ReceiverPostalCode is the postal code
	ReceiverPostalCode string
	// TotalAmount is the total order amount (what buyer paid)
	TotalAmount decimal.Decimal
	// ProductAmount is the total product amount (before discounts)
	ProductAmount decimal.Decimal
	// FreightAmount is the shipping fee
	FreightAmount decimal.Decimal
	// DiscountAmount is the total discount amount
	DiscountAmount decimal.Decimal
	// PlatformDiscount is discount provided by platform
	PlatformDiscount decimal.Decimal
	// SellerDiscount is discount provided by seller
	SellerDiscount decimal.Decimal
	// Currency is the payment currency (default: CNY)
	Currency string
	// Items contains the order line items
	Items []PlatformOrderItem
	// BuyerMessage is the message from buyer
	BuyerMessage string
	// SellerMemo is the seller's note
	SellerMemo string
	// CreatedAt is when the order was created on the platform
	CreatedAt time.Time
	// PaidAt is when the payment was received
	PaidAt *time.Time
	// ShippedAt is when the order was shipped
	ShippedAt *time.Time
	// CompletedAt is when the order was completed
	CompletedAt *time.Time
	// RawData is the original platform response (JSON)
	RawData string
}

// PlatformOrderItem represents a line item in a platform order
type PlatformOrderItem struct {
	// PlatformItemID is the item ID on the platform
	PlatformItemID string
	// PlatformProductID is the product ID on the platform
	PlatformProductID string
	// PlatformSkuID is the SKU ID on the platform
	PlatformSkuID string
	// ProductName is the product name
	ProductName string
	// SkuName is the SKU specification name
	SkuName string
	// ImageURL is the product image URL
	ImageURL string
	// Quantity is the ordered quantity
	Quantity decimal.Decimal
	// UnitPrice is the unit price
	UnitPrice decimal.Decimal
	// TotalPrice is the total price for this item
	TotalPrice decimal.Decimal
	// DiscountAmount is the discount for this item
	DiscountAmount decimal.Decimal
}

// ProductSync represents a product to be synchronized to a platform
type ProductSync struct {
	// PlatformProductID is the existing product ID on platform (empty for new)
	PlatformProductID string
	// LocalProductID is our internal product ID
	LocalProductID uuid.UUID
	// ProductCode is our internal product code
	ProductCode string
	// ProductName is the product name
	ProductName string
	// Description is the product description
	Description string
	// CategoryID is the platform category ID
	CategoryID string
	// Price is the selling price
	Price decimal.Decimal
	// OriginalPrice is the original/compare price
	OriginalPrice decimal.Decimal
	// Quantity is the available quantity
	Quantity decimal.Decimal
	// IsOnSale indicates if the product should be listed for sale
	IsOnSale bool
	// Weight is the product weight in grams
	Weight decimal.Decimal
	// ImageURLs contains product image URLs
	ImageURLs []string
	// SKUs contains the SKU variants
	SKUs []ProductSkuSync
}

// ProductSkuSync represents a SKU variant for synchronization
type ProductSkuSync struct {
	// PlatformSkuID is the existing SKU ID on platform (empty for new)
	PlatformSkuID string
	// LocalSkuID is our internal SKU/variant ID
	LocalSkuID uuid.UUID
	// SkuCode is our internal SKU code
	SkuCode string
	// SkuName is the SKU specification name
	SkuName string
	// Attributes contains SKU attributes (e.g., color, size)
	Attributes map[string]string
	// Price is the SKU price
	Price decimal.Decimal
	// Quantity is the SKU quantity
	Quantity decimal.Decimal
	// ImageURL is the SKU-specific image URL
	ImageURL string
}

// InventorySync represents inventory data to sync to a platform
type InventorySync struct {
	// PlatformProductID is the product ID on the platform
	PlatformProductID string
	// PlatformSkuID is the SKU ID on the platform (optional)
	PlatformSkuID string
	// AvailableQuantity is the available stock quantity
	AvailableQuantity decimal.Decimal
}

// ---------------------------------------------------------------------------
// Request/Response DTOs
// ---------------------------------------------------------------------------

// OrderPullRequest represents a request to pull orders from a platform
type OrderPullRequest struct {
	// TenantID is the tenant making the request
	TenantID uuid.UUID
	// PlatformCode specifies which platform to pull from
	PlatformCode PlatformCode
	// StartTime is the start of the time range for orders
	StartTime time.Time
	// EndTime is the end of the time range for orders
	EndTime time.Time
	// Status filters by order status (optional)
	Status *PlatformOrderStatus
	// PageNo is the page number (1-indexed)
	PageNo int
	// PageSize is the number of orders per page
	PageSize int
}

// Validate validates the order pull request
func (r *OrderPullRequest) Validate() error {
	if r.TenantID == uuid.Nil {
		return ErrMappingInvalidTenantID
	}
	if !r.PlatformCode.IsValid() {
		return ErrMappingInvalidPlatformCode
	}
	if r.StartTime.IsZero() || r.EndTime.IsZero() {
		return errors.New("integration: start time and end time are required")
	}
	if r.StartTime.After(r.EndTime) {
		return errors.New("integration: start time must be before end time")
	}
	if r.PageNo < 1 {
		r.PageNo = 1
	}
	if r.PageSize < 1 || r.PageSize > 100 {
		r.PageSize = 50
	}
	return nil
}

// OrderPullResponse represents the response from pulling orders
type OrderPullResponse struct {
	// Orders contains the pulled orders
	Orders []PlatformOrder
	// TotalCount is the total number of orders matching criteria
	TotalCount int64
	// HasMore indicates if there are more pages
	HasMore bool
	// NextPageNo is the next page number (if HasMore is true)
	NextPageNo int
}

// SyncResult represents the result of a sync operation
type SyncResult struct {
	// Status is the overall sync status
	Status SyncStatus
	// TotalCount is the total number of items to sync
	TotalCount int
	// SuccessCount is the number of successfully synced items
	SuccessCount int
	// FailedCount is the number of failed items
	FailedCount int
	// FailedItems contains details about failed items
	FailedItems []SyncFailure
	// SyncedAt is when the sync completed
	SyncedAt time.Time
}

// SyncFailure represents a failed sync item
type SyncFailure struct {
	// ItemID is the identifier of the failed item
	ItemID string
	// ErrorCode is the platform error code
	ErrorCode string
	// ErrorMessage is the error description
	ErrorMessage string
}

// OrderStatusUpdateRequest represents a request to update order status on platform
type OrderStatusUpdateRequest struct {
	// TenantID is the tenant making the request
	TenantID uuid.UUID
	// PlatformCode specifies which platform
	PlatformCode PlatformCode
	// PlatformOrderID is the order ID on the platform
	PlatformOrderID string
	// Status is the new status to set
	Status PlatformOrderStatus
	// ShippingCompany is the shipping company name (for SHIPPED status)
	ShippingCompany string
	// TrackingNumber is the shipping tracking number (for SHIPPED status)
	TrackingNumber string
}

// Validate validates the order status update request
func (r *OrderStatusUpdateRequest) Validate() error {
	if r.TenantID == uuid.Nil {
		return ErrMappingInvalidTenantID
	}
	if !r.PlatformCode.IsValid() {
		return ErrMappingInvalidPlatformCode
	}
	if r.PlatformOrderID == "" {
		return ErrOrderSyncOrderNotFound
	}
	if !r.Status.IsValid() {
		return ErrOrderSyncInvalidStatus
	}
	// Shipping info required for SHIPPED status
	if r.Status == PlatformOrderStatusShipped {
		if r.ShippingCompany == "" || r.TrackingNumber == "" {
			return errors.New("integration: shipping company and tracking number required for shipped status")
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// EcommercePlatform Port Interface
// ---------------------------------------------------------------------------

// EcommercePlatform defines the port interface for external e-commerce platforms.
// This interface follows the Ports & Adapters pattern - it's defined in the domain
// layer, and concrete implementations (Taobao, JD, PDD, Douyin) are in the infrastructure layer.
type EcommercePlatform interface {
	// PlatformCode returns the platform code this adapter handles
	PlatformCode() PlatformCode

	// IsEnabled returns true if this platform is enabled for the tenant
	IsEnabled(ctx context.Context, tenantID uuid.UUID) (bool, error)

	// ---------------------------------------------------------------------------
	// Product Operations
	// ---------------------------------------------------------------------------

	// SyncProducts synchronizes products to the platform
	// Creates new products or updates existing ones based on PlatformProductID
	SyncProducts(ctx context.Context, tenantID uuid.UUID, products []ProductSync) (*SyncResult, error)

	// GetProduct retrieves a product from the platform
	GetProduct(ctx context.Context, tenantID uuid.UUID, platformProductID string) (*ProductSync, error)

	// ---------------------------------------------------------------------------
	// Order Operations
	// ---------------------------------------------------------------------------

	// PullOrders pulls orders from the platform within the specified time range
	PullOrders(ctx context.Context, req *OrderPullRequest) (*OrderPullResponse, error)

	// GetOrder retrieves a single order from the platform
	GetOrder(ctx context.Context, tenantID uuid.UUID, platformOrderID string) (*PlatformOrder, error)

	// UpdateOrderStatus updates the order status on the platform
	// Used for shipping confirmation, delivery confirmation, etc.
	UpdateOrderStatus(ctx context.Context, req *OrderStatusUpdateRequest) error

	// ---------------------------------------------------------------------------
	// Inventory Operations
	// ---------------------------------------------------------------------------

	// SyncInventory synchronizes inventory levels to the platform
	SyncInventory(ctx context.Context, tenantID uuid.UUID, items []InventorySync) (*SyncResult, error)
}

// EcommercePlatformRegistry provides access to configured e-commerce platforms
// This allows selecting the appropriate platform adapter based on platform code
type EcommercePlatformRegistry interface {
	// GetPlatform returns the platform adapter for the specified code
	GetPlatform(platformCode PlatformCode) (EcommercePlatform, error)

	// ListPlatforms returns all registered platform adapters
	ListPlatforms() []EcommercePlatform

	// ListEnabledPlatforms returns all enabled platforms for a tenant
	ListEnabledPlatforms(ctx context.Context, tenantID uuid.UUID) ([]EcommercePlatform, error)

	// IsEnabled returns true if the platform is enabled for the tenant
	IsEnabled(ctx context.Context, tenantID uuid.UUID, platformCode PlatformCode) (bool, error)
}
