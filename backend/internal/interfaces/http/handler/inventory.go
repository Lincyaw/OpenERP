package handler

import (
	"time"

	inventoryapp "github.com/erp/backend/internal/application/inventory"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// parseDateTime parses a datetime string in various formats
func parseDateTime(s string) (time.Time, error) {
	// Try RFC3339 first
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	// Try ISO date format
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	// Try datetime without timezone
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t, nil
	}
	// Default to RFC3339 parsing error
	return time.Parse(time.RFC3339, s)
}

// InventoryHandler handles inventory-related API endpoints
type InventoryHandler struct {
	BaseHandler
	inventoryService *inventoryapp.InventoryService
}

// NewInventoryHandler creates a new InventoryHandler
func NewInventoryHandler(inventoryService *inventoryapp.InventoryService) *InventoryHandler {
	return &InventoryHandler{
		inventoryService: inventoryService,
	}
}

// ===================== Request/Response Types for Swagger =====================

// InventoryItemResponse represents an inventory item in API responses
// @Description Inventory item response with stock quantities and cost information
type InventoryItemResponse struct {
	ID                string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantID          string  `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	WarehouseID       string  `json:"warehouse_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	ProductID         string  `json:"product_id" example:"550e8400-e29b-41d4-a716-446655440003"`
	AvailableQuantity float64 `json:"available_quantity" example:"100.5"`
	LockedQuantity    float64 `json:"locked_quantity" example:"10.0"`
	TotalQuantity     float64 `json:"total_quantity" example:"110.5"`
	UnitCost          float64 `json:"unit_cost" example:"15.50"`
	TotalValue        float64 `json:"total_value" example:"1712.75"`
	MinQuantity       float64 `json:"min_quantity" example:"20.0"`
	MaxQuantity       float64 `json:"max_quantity" example:"500.0"`
	IsBelowMinimum    bool    `json:"is_below_minimum" example:"false"`
	IsAboveMaximum    bool    `json:"is_above_maximum" example:"false"`
	CreatedAt         string  `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt         string  `json:"updated_at" example:"2024-01-15T10:30:00Z"`
	Version           int     `json:"version" example:"1"`
}

// IncreaseStockRequest represents a request to increase stock
// @Description Request body for increasing stock
type IncreaseStockRequest struct {
	WarehouseID string  `json:"warehouse_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	ProductID   string  `json:"product_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440001"`
	Quantity    float64 `json:"quantity" binding:"required,gt=0" example:"50.0"`
	UnitCost    float64 `json:"unit_cost" binding:"required,gte=0" example:"15.50"`
	SourceType  string  `json:"source_type" binding:"required" example:"PURCHASE_ORDER"`
	SourceID    string  `json:"source_id" binding:"required" example:"PO-2024-001"`
	BatchNumber string  `json:"batch_number" example:"BATCH-001"`
	ExpiryDate  string  `json:"expiry_date" example:"2025-12-31"`
	Reference   string  `json:"reference" example:"Received from supplier ABC"`
	Reason      string  `json:"reason" example:"Regular purchase"`
	OperatorID  string  `json:"operator_id" example:"550e8400-e29b-41d4-a716-446655440002"`
}

// LockStockRequest represents a request to lock stock
// @Description Request body for locking stock for a pending order
type LockStockRequest struct {
	WarehouseID string  `json:"warehouse_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	ProductID   string  `json:"product_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440001"`
	Quantity    float64 `json:"quantity" binding:"required,gt=0" example:"10.0"`
	SourceType  string  `json:"source_type" binding:"required" example:"sales_order"`
	SourceID    string  `json:"source_id" binding:"required" example:"SO-2024-001"`
	ExpireAt    string  `json:"expire_at" example:"2024-01-15T11:00:00Z"`
}

// LockStockResponse represents the response after locking stock
// @Description Response after successfully locking stock
type LockStockResponse struct {
	LockID          string  `json:"lock_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	InventoryItemID string  `json:"inventory_item_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	WarehouseID     string  `json:"warehouse_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	ProductID       string  `json:"product_id" example:"550e8400-e29b-41d4-a716-446655440003"`
	Quantity        float64 `json:"quantity" example:"10.0"`
	ExpireAt        string  `json:"expire_at" example:"2024-01-15T11:00:00Z"`
	SourceType      string  `json:"source_type" example:"sales_order"`
	SourceID        string  `json:"source_id" example:"SO-2024-001"`
}

// UnlockStockRequest represents a request to unlock stock
// @Description Request body for releasing locked stock
type UnlockStockRequest struct {
	LockID string `json:"lock_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// DeductStockRequest represents a request to deduct locked stock
// @Description Request body for consuming locked stock (shipment/fulfillment)
type DeductStockRequest struct {
	LockID     string `json:"lock_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	SourceType string `json:"source_type" binding:"required" example:"SALES_ORDER"`
	SourceID   string `json:"source_id" binding:"required" example:"SO-2024-001"`
	Reference  string `json:"reference" example:"Shipped via SF Express"`
	OperatorID string `json:"operator_id" example:"550e8400-e29b-41d4-a716-446655440002"`
}

// AdjustStockRequest represents a request to adjust stock
// @Description Request body for stock count adjustment
type AdjustStockRequest struct {
	WarehouseID    string  `json:"warehouse_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	ProductID      string  `json:"product_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440001"`
	ActualQuantity float64 `json:"actual_quantity" binding:"required,gte=0" example:"95.0"`
	Reason         string  `json:"reason" binding:"required,min=1,max=255" example:"Stock count variance - 5 units damaged"`
	SourceType     string  `json:"source_type" example:"STOCK_TAKE"`
	SourceID       string  `json:"source_id" example:"ST-2024-001"`
	OperatorID     string  `json:"operator_id" example:"550e8400-e29b-41d4-a716-446655440002"`
}

// SetThresholdsRequest represents a request to set min/max thresholds
// @Description Request body for setting inventory alert thresholds
type SetThresholdsRequest struct {
	WarehouseID string   `json:"warehouse_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	ProductID   string   `json:"product_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440001"`
	MinQuantity *float64 `json:"min_quantity" example:"20.0"`
	MaxQuantity *float64 `json:"max_quantity" example:"500.0"`
}

// CheckAvailabilityRequest represents a request to check stock availability
// @Description Request body for checking if a quantity is available
type CheckAvailabilityRequest struct {
	WarehouseID string  `json:"warehouse_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	ProductID   string  `json:"product_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440001"`
	Quantity    float64 `json:"quantity" binding:"required,gt=0" example:"10.0"`
}

// CheckAvailabilityResponse represents the availability check result
// @Description Response indicating if requested quantity is available
type CheckAvailabilityResponse struct {
	Available         bool    `json:"available" example:"true"`
	AvailableQuantity float64 `json:"available_quantity" example:"100.0"`
	RequestedQuantity float64 `json:"requested_quantity" example:"10.0"`
}

// StockLockResponse represents a stock lock in API responses
// @Description Stock lock details
type StockLockResponse struct {
	ID              string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	InventoryItemID string  `json:"inventory_item_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Quantity        float64 `json:"quantity" example:"10.0"`
	SourceType      string  `json:"source_type" example:"sales_order"`
	SourceID        string  `json:"source_id" example:"SO-2024-001"`
	ExpireAt        string  `json:"expire_at" example:"2024-01-15T11:00:00Z"`
	Released        bool    `json:"released" example:"false"`
	Consumed        bool    `json:"consumed" example:"false"`
	IsActive        bool    `json:"is_active" example:"true"`
	IsExpired       bool    `json:"is_expired" example:"false"`
	CreatedAt       string  `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

// TransactionResponse represents an inventory transaction in API responses
// @Description Inventory transaction record for audit trail
type TransactionResponse struct {
	ID              string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantID        string  `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	InventoryItemID string  `json:"inventory_item_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	WarehouseID     string  `json:"warehouse_id" example:"550e8400-e29b-41d4-a716-446655440003"`
	ProductID       string  `json:"product_id" example:"550e8400-e29b-41d4-a716-446655440004"`
	TransactionType string  `json:"transaction_type" example:"INBOUND"`
	Quantity        float64 `json:"quantity" example:"50.0"`
	SignedQuantity  float64 `json:"signed_quantity" example:"50.0"`
	UnitCost        float64 `json:"unit_cost" example:"15.50"`
	TotalCost       float64 `json:"total_cost" example:"775.0"`
	BalanceBefore   float64 `json:"balance_before" example:"50.0"`
	BalanceAfter    float64 `json:"balance_after" example:"100.0"`
	SourceType      string  `json:"source_type" example:"PURCHASE_ORDER"`
	SourceID        string  `json:"source_id" example:"PO-2024-001"`
	Reference       string  `json:"reference,omitempty" example:"Received from supplier ABC"`
	Reason          string  `json:"reason,omitempty" example:"Regular purchase"`
	TransactionDate string  `json:"transaction_date" example:"2024-01-15T10:30:00Z"`
	CreatedAt       string  `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

// ===================== Query Handlers =====================

// ===================== Query Handlers =====================

// GetByID godoc
// @ID           getInventoryById
// @Summary      Get inventory item by ID
// @Description  Retrieve an inventory item by its ID
// @Tags         inventory
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Inventory Item ID" format(uuid)
// @Success      200 {object} APIResponse[InventoryItemResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /inventory/items/{id} [get]
func (h *InventoryHandler) GetByID(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid inventory item ID format")
		return
	}

	item, err := h.inventoryService.GetByID(c.Request.Context(), tenantID, itemID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, item)
}

// GetByWarehouseAndProduct godoc
// @ID           getInventoryByWarehouseAndProduct
// @Summary      Get inventory by warehouse and product
// @Description  Retrieve inventory for a specific warehouse-product combination
// @Tags         inventory
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        warehouse_id query string true "Warehouse ID" format(uuid)
// @Param        product_id query string true "Product ID" format(uuid)
// @Success      200 {object} APIResponse[InventoryItemResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /inventory/items/lookup [get]
func (h *InventoryHandler) GetByWarehouseAndProduct(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	warehouseIDStr := c.Query("warehouse_id")
	productIDStr := c.Query("product_id")

	if warehouseIDStr == "" || productIDStr == "" {
		h.BadRequest(c, "warehouse_id and product_id are required")
		return
	}

	warehouseID, err := uuid.Parse(warehouseIDStr)
	if err != nil {
		h.BadRequest(c, "Invalid warehouse ID format")
		return
	}

	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	item, err := h.inventoryService.GetByWarehouseAndProduct(c.Request.Context(), tenantID, warehouseID, productID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, item)
}

// List godoc
// @ID           listInventories
// @Summary      List inventory items
// @Description  Retrieve a paginated list of inventory items with optional filtering
// @Tags         inventory
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        search query string false "Search term"
// @Param        warehouse_id query string false "Filter by warehouse ID" format(uuid)
// @Param        product_id query string false "Filter by product ID" format(uuid)
// @Param        below_minimum query boolean false "Filter items below minimum threshold"
// @Param        has_stock query boolean false "Filter by stock availability"
// @Param        min_quantity query number false "Minimum quantity filter"
// @Param        max_quantity query number false "Maximum quantity filter"
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20) maximum(100)
// @Param        order_by query string false "Order by field" default(updated_at)
// @Param        order_dir query string false "Order direction" Enums(asc, desc) default(desc)
// @Success      200 {object} APIResponse[[]InventoryItemResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /inventory/items [get]
func (h *InventoryHandler) List(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var filter inventoryapp.InventoryListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	items, total, err := h.inventoryService.List(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.SuccessWithMeta(c, items, total, filter.Page, filter.PageSize)
}

// ListByWarehouse godoc
// @ID           listInventoryByWarehouse
// @Summary      List inventory by warehouse
// @Description  Retrieve inventory items for a specific warehouse
// @Tags         inventory
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        warehouse_id path string true "Warehouse ID" format(uuid)
// @Param        search query string false "Search term"
// @Param        below_minimum query boolean false "Filter items below minimum threshold"
// @Param        has_stock query boolean false "Filter by stock availability"
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20) maximum(100)
// @Param        order_by query string false "Order by field" default(updated_at)
// @Param        order_dir query string false "Order direction" Enums(asc, desc) default(desc)
// @Success      200 {object} APIResponse[[]InventoryItemResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /inventory/warehouses/{warehouse_id}/items [get]
func (h *InventoryHandler) ListByWarehouse(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	warehouseID, err := uuid.Parse(c.Param("warehouse_id"))
	if err != nil {
		h.BadRequest(c, "Invalid warehouse ID format")
		return
	}

	var filter inventoryapp.InventoryListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	items, total, err := h.inventoryService.ListByWarehouse(c.Request.Context(), tenantID, warehouseID, filter)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.SuccessWithMeta(c, items, total, filter.Page, filter.PageSize)
}

// ListByProduct godoc
// @ID           listInventoryByProduct
// @Summary      List inventory by product
// @Description  Retrieve inventory items for a specific product across all warehouses
// @Tags         inventory
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        product_id path string true "Product ID" format(uuid)
// @Param        search query string false "Search term"
// @Param        below_minimum query boolean false "Filter items below minimum threshold"
// @Param        has_stock query boolean false "Filter by stock availability"
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20) maximum(100)
// @Param        order_by query string false "Order by field" default(updated_at)
// @Param        order_dir query string false "Order direction" Enums(asc, desc) default(desc)
// @Success      200 {object} APIResponse[[]InventoryItemResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /inventory/products/{product_id}/items [get]
func (h *InventoryHandler) ListByProduct(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	productID, err := uuid.Parse(c.Param("product_id"))
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	var filter inventoryapp.InventoryListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	items, total, err := h.inventoryService.ListByProduct(c.Request.Context(), tenantID, productID, filter)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.SuccessWithMeta(c, items, total, filter.Page, filter.PageSize)
}

// ListBelowMinimum godoc
// @ID           listInventoryBelowMinimum
// @Summary      List inventory below minimum threshold
// @Description  Retrieve inventory items that are below their minimum threshold (low stock alerts)
// @Tags         inventory
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        warehouse_id query string false "Filter by warehouse ID" format(uuid)
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20) maximum(100)
// @Param        order_by query string false "Order by field" default(updated_at)
// @Param        order_dir query string false "Order direction" Enums(asc, desc) default(desc)
// @Success      200 {object} APIResponse[[]InventoryItemResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /inventory/items/alerts/low-stock [get]
func (h *InventoryHandler) ListBelowMinimum(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var filter inventoryapp.InventoryListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	items, total, err := h.inventoryService.ListBelowMinimum(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.SuccessWithMeta(c, items, total, filter.Page, filter.PageSize)
}

// CheckAvailability godoc
// @ID           checkAvailabilityInventory
// @Summary      Check stock availability
// @Description  Check if a specific quantity is available for a product in a warehouse
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        request body CheckAvailabilityRequest true "Availability check request"
// @Success      200 {object} APIResponse[CheckAvailabilityResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /inventory/availability/check [post]
func (h *InventoryHandler) CheckAvailability(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req CheckAvailabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	warehouseID, err := uuid.Parse(req.WarehouseID)
	if err != nil {
		h.BadRequest(c, "Invalid warehouse ID format")
		return
	}

	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	quantity := decimal.NewFromFloat(req.Quantity)
	available, availableQty, err := h.inventoryService.CheckAvailability(c.Request.Context(), tenantID, warehouseID, productID, quantity)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	availableFloat, _ := availableQty.Float64()
	h.Success(c, CheckAvailabilityResponse{
		Available:         available,
		AvailableQuantity: availableFloat,
		RequestedQuantity: req.Quantity,
	})
}

// ===================== Stock Operation Handlers =====================

// ===================== Stock Operation Handlers =====================

// IncreaseStock godoc
// @ID           increaseStockInventory
// @Summary      Increase stock
// @Description  Increase stock for a product in a warehouse (e.g., purchase receiving, returns)
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        request body IncreaseStockRequest true "Stock increase request"
// @Success      200 {object} APIResponse[InventoryItemResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /inventory/stock/increase [post]
func (h *InventoryHandler) IncreaseStock(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req IncreaseStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	warehouseID, err := uuid.Parse(req.WarehouseID)
	if err != nil {
		h.BadRequest(c, "Invalid warehouse ID format")
		return
	}

	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	// Convert to application DTO
	appReq := inventoryapp.IncreaseStockRequest{
		WarehouseID: warehouseID,
		ProductID:   productID,
		Quantity:    decimal.NewFromFloat(req.Quantity),
		UnitCost:    decimal.NewFromFloat(req.UnitCost),
		SourceType:  req.SourceType,
		SourceID:    req.SourceID,
		BatchNumber: req.BatchNumber,
		Reference:   req.Reference,
		Reason:      req.Reason,
	}

	// Parse optional expiry date
	if req.ExpiryDate != "" {
		expiryTime, err := parseDateTime(req.ExpiryDate)
		if err != nil {
			h.BadRequest(c, "Invalid expiry date format")
			return
		}
		appReq.ExpiryDate = &expiryTime
	}

	// Parse optional operator ID
	if req.OperatorID != "" {
		opID, err := uuid.Parse(req.OperatorID)
		if err != nil {
			h.BadRequest(c, "Invalid operator ID format")
			return
		}
		appReq.OperatorID = &opID
	}

	item, err := h.inventoryService.IncreaseStock(c.Request.Context(), tenantID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, item)
}

// LockStock godoc
// @ID           lockStockInventory
// @Summary      Lock stock
// @Description  Lock stock for a pending order (reserve inventory)
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        request body LockStockRequest true "Stock lock request"
// @Success      200 {object} APIResponse[LockStockResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      422 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /inventory/stock/lock [post]
func (h *InventoryHandler) LockStock(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req LockStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	warehouseID, err := uuid.Parse(req.WarehouseID)
	if err != nil {
		h.BadRequest(c, "Invalid warehouse ID format")
		return
	}

	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	// Convert to application DTO
	appReq := inventoryapp.LockStockRequest{
		WarehouseID: warehouseID,
		ProductID:   productID,
		Quantity:    decimal.NewFromFloat(req.Quantity),
		SourceType:  req.SourceType,
		SourceID:    req.SourceID,
	}

	// Parse optional expire time
	if req.ExpireAt != "" {
		expireTime, err := parseDateTime(req.ExpireAt)
		if err != nil {
			h.BadRequest(c, "Invalid expire_at format")
			return
		}
		appReq.ExpireAt = &expireTime
	}

	result, err := h.inventoryService.LockStock(c.Request.Context(), tenantID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// UnlockStock godoc
// @ID           unlockStockInventory
// @Summary      Unlock stock
// @Description  Release previously locked stock back to available (e.g., order cancelled)
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        request body UnlockStockRequest true "Stock unlock request"
// @Success      204
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      422 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /inventory/stock/unlock [post]
func (h *InventoryHandler) UnlockStock(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req UnlockStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	lockID, err := uuid.Parse(req.LockID)
	if err != nil {
		h.BadRequest(c, "Invalid lock ID format")
		return
	}

	appReq := inventoryapp.UnlockStockRequest{
		LockID: lockID,
	}

	err = h.inventoryService.UnlockStock(c.Request.Context(), tenantID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.NoContent(c)
}

// DeductStock godoc
// @ID           deductStockInventory
// @Summary      Deduct stock
// @Description  Deduct locked stock (actual shipment/consumption)
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        request body DeductStockRequest true "Stock deduction request"
// @Success      204
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      422 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /inventory/stock/deduct [post]
func (h *InventoryHandler) DeductStock(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req DeductStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	lockID, err := uuid.Parse(req.LockID)
	if err != nil {
		h.BadRequest(c, "Invalid lock ID format")
		return
	}

	appReq := inventoryapp.DeductStockRequest{
		LockID:     lockID,
		SourceType: req.SourceType,
		SourceID:   req.SourceID,
		Reference:  req.Reference,
	}

	// Parse optional operator ID
	if req.OperatorID != "" {
		opID, err := uuid.Parse(req.OperatorID)
		if err != nil {
			h.BadRequest(c, "Invalid operator ID format")
			return
		}
		appReq.OperatorID = &opID
	}

	err = h.inventoryService.DeductStock(c.Request.Context(), tenantID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.NoContent(c)
}

// AdjustStock godoc
// @ID           adjustStockInventory
// @Summary      Adjust stock
// @Description  Adjust stock to match actual quantity (stock count/adjustment)
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        request body AdjustStockRequest true "Stock adjustment request"
// @Success      200 {object} APIResponse[InventoryItemResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      422 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /inventory/stock/adjust [post]
func (h *InventoryHandler) AdjustStock(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req AdjustStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	warehouseID, err := uuid.Parse(req.WarehouseID)
	if err != nil {
		h.BadRequest(c, "Invalid warehouse ID format")
		return
	}

	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	appReq := inventoryapp.AdjustStockRequest{
		WarehouseID:    warehouseID,
		ProductID:      productID,
		ActualQuantity: decimal.NewFromFloat(req.ActualQuantity),
		Reason:         req.Reason,
		SourceType:     req.SourceType,
		SourceID:       req.SourceID,
	}

	// Parse optional operator ID
	if req.OperatorID != "" {
		opID, err := uuid.Parse(req.OperatorID)
		if err != nil {
			h.BadRequest(c, "Invalid operator ID format")
			return
		}
		appReq.OperatorID = &opID
	}

	item, err := h.inventoryService.AdjustStock(c.Request.Context(), tenantID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, item)
}

// SetThresholds godoc
// @ID           setThresholdsInventory
// @Summary      Set inventory thresholds
// @Description  Set min/max quantity thresholds for inventory alerts
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        request body SetThresholdsRequest true "Threshold settings request"
// @Success      200 {object} APIResponse[InventoryItemResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      422 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /inventory/thresholds [put]
func (h *InventoryHandler) SetThresholds(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req SetThresholdsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	warehouseID, err := uuid.Parse(req.WarehouseID)
	if err != nil {
		h.BadRequest(c, "Invalid warehouse ID format")
		return
	}

	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	appReq := inventoryapp.SetThresholdsRequest{
		WarehouseID: warehouseID,
		ProductID:   productID,
	}

	if req.MinQuantity != nil {
		minQty := decimal.NewFromFloat(*req.MinQuantity)
		appReq.MinQuantity = &minQty
	}
	if req.MaxQuantity != nil {
		maxQty := decimal.NewFromFloat(*req.MaxQuantity)
		appReq.MaxQuantity = &maxQty
	}

	item, err := h.inventoryService.SetThresholds(c.Request.Context(), tenantID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, item)
}

// ===================== Lock Management Handlers =====================

// ===================== Lock Management Handlers =====================

// GetActiveLocks godoc
// @ID           getInventoryActiveLocks
// @Summary      Get active locks
// @Description  Retrieve all active (unexpired, unreleased) locks for an inventory item
// @Tags         inventory
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        warehouse_id query string true "Warehouse ID" format(uuid)
// @Param        product_id query string true "Product ID" format(uuid)
// @Success      200 {object} APIResponse[[]StockLockResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /inventory/locks [get]
func (h *InventoryHandler) GetActiveLocks(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	warehouseIDStr := c.Query("warehouse_id")
	productIDStr := c.Query("product_id")

	if warehouseIDStr == "" || productIDStr == "" {
		h.BadRequest(c, "warehouse_id and product_id are required")
		return
	}

	warehouseID, err := uuid.Parse(warehouseIDStr)
	if err != nil {
		h.BadRequest(c, "Invalid warehouse ID format")
		return
	}

	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	locks, err := h.inventoryService.GetActiveLocks(c.Request.Context(), tenantID, warehouseID, productID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, locks)
}

// GetLockByID godoc
// @ID           getInventoryLockByID
// @Summary      Get lock by ID
// @Description  Retrieve a specific stock lock by ID
// @Tags         inventory
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Lock ID" format(uuid)
// @Success      200 {object} APIResponse[StockLockResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /inventory/locks/{id} [get]
func (h *InventoryHandler) GetLockByID(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	lockID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid lock ID format")
		return
	}

	lock, err := h.inventoryService.GetLockByID(c.Request.Context(), tenantID, lockID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, lock)
}

// ===================== Transaction Handlers =====================

// ===================== Transaction Handlers =====================

// ListTransactions godoc
// @ID           listInventoryTransactions
// @Summary      List inventory transactions
// @Description  Retrieve a paginated list of inventory transactions with optional filtering
// @Tags         inventory
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        warehouse_id query string false "Filter by warehouse ID" format(uuid)
// @Param        product_id query string false "Filter by product ID" format(uuid)
// @Param        transaction_type query string false "Filter by transaction type" Enums(INBOUND, OUTBOUND, LOCK, UNLOCK, ADJUSTMENT)
// @Param        source_type query string false "Filter by source type" Enums(PURCHASE_ORDER, SALES_ORDER, SALES_RETURN, PURCHASE_RETURN, MANUAL_ADJUSTMENT, STOCK_TAKE)
// @Param        source_id query string false "Filter by source ID"
// @Param        start_date query string false "Filter by start date" format(date-time)
// @Param        end_date query string false "Filter by end date" format(date-time)
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20) maximum(100)
// @Param        order_by query string false "Order by field" default(transaction_date)
// @Param        order_dir query string false "Order direction" Enums(asc, desc) default(desc)
// @Success      200 {object} APIResponse[[]TransactionResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /inventory/transactions [get]
func (h *InventoryHandler) ListTransactions(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var filter inventoryapp.TransactionListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	txs, total, err := h.inventoryService.ListTransactions(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.SuccessWithMeta(c, txs, total, filter.Page, filter.PageSize)
}

// ListTransactionsByItem godoc
// @ID           listInventoryTransactionsByItem
// @Summary      List transactions by inventory item
// @Description  Retrieve transactions for a specific inventory item
// @Tags         inventory
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Inventory Item ID" format(uuid)
// @Param        transaction_type query string false "Filter by transaction type" Enums(INBOUND, OUTBOUND, LOCK, UNLOCK, ADJUSTMENT)
// @Param        start_date query string false "Filter by start date" format(date-time)
// @Param        end_date query string false "Filter by end date" format(date-time)
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20) maximum(100)
// @Param        order_by query string false "Order by field" default(transaction_date)
// @Param        order_dir query string false "Order direction" Enums(asc, desc) default(desc)
// @Success      200 {object} APIResponse[[]TransactionResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /inventory/items/{id}/transactions [get]
func (h *InventoryHandler) ListTransactionsByItem(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid inventory item ID format")
		return
	}

	var filter inventoryapp.TransactionListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	txs, total, err := h.inventoryService.ListTransactionsByInventoryItem(c.Request.Context(), tenantID, itemID, filter)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.SuccessWithMeta(c, txs, total, filter.Page, filter.PageSize)
}

// GetTransactionByID godoc
// @ID           getInventoryTransactionByID
// @Summary      Get transaction by ID
// @Description  Retrieve a specific inventory transaction by ID
// @Tags         inventory
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Transaction ID" format(uuid)
// @Success      200 {object} APIResponse[TransactionResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /inventory/transactions/{id} [get]
func (h *InventoryHandler) GetTransactionByID(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	txID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid transaction ID format")
		return
	}

	tx, err := h.inventoryService.GetTransactionByID(c.Request.Context(), tenantID, txID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, tx)
}

// Helper function to suppress unused import warning
var _ = dto.Response{}
