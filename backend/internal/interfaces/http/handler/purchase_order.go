package handler

import (
	"time"

	tradeapp "github.com/erp/backend/internal/application/trade"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PurchaseOrderHandler handles purchase order-related API endpoints
type PurchaseOrderHandler struct {
	BaseHandler
	orderService *tradeapp.PurchaseOrderService
}

// NewPurchaseOrderHandler creates a new PurchaseOrderHandler
func NewPurchaseOrderHandler(orderService *tradeapp.PurchaseOrderService) *PurchaseOrderHandler {
	return &PurchaseOrderHandler{
		orderService: orderService,
	}
}

// CreatePurchaseOrderRequest represents a request to create a new purchase order
// @Description Request body for creating a new purchase order
type CreatePurchaseOrderRequest struct {
	SupplierID   string                         `json:"supplier_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	SupplierName string                         `json:"supplier_name" binding:"required,min=1,max=200" example:"供应商A"`
	WarehouseID  *string                        `json:"warehouse_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Items        []CreatePurchaseOrderItemInput `json:"items"`
	Discount     *float64                       `json:"discount" example:"100.00"`
	Remark       string                         `json:"remark" example:"备注信息"`
}

// CreatePurchaseOrderItemInput represents an item in the create order request
// @Description Purchase order item for creation
type CreatePurchaseOrderItemInput struct {
	ProductID      string  `json:"product_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440002"`
	ProductName    string  `json:"product_name" binding:"required,min=1,max=200" example:"测试商品"`
	ProductCode    string  `json:"product_code" binding:"required,min=1,max=50" example:"SKU-001"`
	Unit           string  `json:"unit" binding:"required,min=1,max=20" example:"pcs"`
	BaseUnit       string  `json:"base_unit" example:"pcs"`       // Base unit code - defaults to Unit if empty
	ConversionRate float64 `json:"conversion_rate" example:"1.0"` // Conversion rate to base unit - defaults to 1 if empty
	Quantity       float64 `json:"quantity" binding:"required,gt=0" example:"10"`
	UnitCost       float64 `json:"unit_cost" binding:"required,gt=0" example:"50.00"`
	Remark         string  `json:"remark" example:"商品备注"`
}

// UpdatePurchaseOrderRequest represents a request to update a purchase order
// @Description Request body for updating a purchase order (draft only)
type UpdatePurchaseOrderRequest struct {
	WarehouseID *string  `json:"warehouse_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Discount    *float64 `json:"discount" example:"50.00"`
	Remark      *string  `json:"remark" example:"更新备注"`
}

// AddPurchaseOrderItemRequest represents a request to add an item to an order
// @Description Request body for adding an item to a purchase order
type AddPurchaseOrderItemRequest struct {
	ProductID      string  `json:"product_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440002"`
	ProductName    string  `json:"product_name" binding:"required,min=1,max=200" example:"测试商品"`
	ProductCode    string  `json:"product_code" binding:"required,min=1,max=50" example:"SKU-001"`
	Unit           string  `json:"unit" binding:"required,min=1,max=20" example:"pcs"`
	BaseUnit       string  `json:"base_unit" example:"pcs"`       // Base unit code - defaults to Unit if empty
	ConversionRate float64 `json:"conversion_rate" example:"1.0"` // Conversion rate to base unit - defaults to 1 if empty
	Quantity       float64 `json:"quantity" binding:"required,gt=0" example:"5"`
	UnitCost       float64 `json:"unit_cost" binding:"required,gt=0" example:"60.00"`
	Remark         string  `json:"remark" example:"商品备注"`
}

// UpdatePurchaseOrderItemRequest represents a request to update an order item
// @Description Request body for updating a purchase order item
type UpdatePurchaseOrderItemRequest struct {
	Quantity *float64 `json:"quantity" example:"8"`
	UnitCost *float64 `json:"unit_cost" example:"55.00"`
	Remark   *string  `json:"remark" example:"更新商品备注"`
}

// ConfirmPurchaseOrderRequest represents a request to confirm an order
// @Description Request body for confirming a purchase order
type ConfirmPurchaseOrderRequest struct {
	WarehouseID *string `json:"warehouse_id" example:"550e8400-e29b-41d4-a716-446655440001"`
}

// ReceiveItemInput represents an item to receive
// @Description Item to receive in a purchase order
type ReceiveItemInput struct {
	ProductID   string     `json:"product_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440002"`
	Quantity    float64    `json:"quantity" binding:"required,gt=0" example:"5"`
	UnitCost    *float64   `json:"unit_cost" example:"52.00"`
	BatchNumber string     `json:"batch_number" example:"BATCH-2026-001"`
	ExpiryDate  *time.Time `json:"expiry_date" example:"2027-12-31T00:00:00Z"`
}

// ReceivePurchaseOrderRequest represents a request to receive goods
// @Description Request body for receiving goods in a purchase order
type ReceivePurchaseOrderRequest struct {
	WarehouseID *string            `json:"warehouse_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Items       []ReceiveItemInput `json:"items" binding:"required,min=1"`
}

// CancelPurchaseOrderRequest represents a request to cancel an order
// @Description Request body for cancelling a purchase order
type CancelPurchaseOrderRequest struct {
	Reason string `json:"reason" binding:"required,min=1,max=500" example:"供应商无法供货"`
}

// PurchaseOrderResponse represents a purchase order in API responses
// @Description Purchase order response
type PurchaseOrderResponse struct {
	ID               string                      `json:"id" example:"550e8400-e29b-41d4-a716-446655440010"`
	TenantID         string                      `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	OrderNumber      string                      `json:"order_number" example:"PO-2026-00001"`
	SupplierID       string                      `json:"supplier_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	SupplierName     string                      `json:"supplier_name" example:"供应商A"`
	WarehouseID      *string                     `json:"warehouse_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
	Items            []PurchaseOrderItemResponse `json:"items"`
	ItemCount        int                         `json:"item_count" example:"3"`
	TotalQuantity    float64                     `json:"total_quantity" example:"30"`
	ReceivedQuantity float64                     `json:"received_quantity" example:"10"`
	TotalAmount      float64                     `json:"total_amount" example:"1500.00"`
	DiscountAmount   float64                     `json:"discount_amount" example:"100.00"`
	PayableAmount    float64                     `json:"payable_amount" example:"1400.00"`
	Status           string                      `json:"status" example:"draft"`
	ReceiveProgress  float64                     `json:"receive_progress" example:"33.33"`
	Remark           string                      `json:"remark" example:"备注信息"`
	ConfirmedAt      *time.Time                  `json:"confirmed_at,omitempty"`
	CompletedAt      *time.Time                  `json:"completed_at,omitempty"`
	CancelledAt      *time.Time                  `json:"cancelled_at,omitempty"`
	CancelReason     string                      `json:"cancel_reason,omitempty" example:""`
	CreatedAt        time.Time                   `json:"created_at"`
	UpdatedAt        time.Time                   `json:"updated_at"`
	Version          int                         `json:"version" example:"1"`
}

// PurchaseOrderListResponse represents a purchase order in list responses
// @Description Purchase order list item response
type PurchaseOrderListResponse struct {
	ID              string     `json:"id" example:"550e8400-e29b-41d4-a716-446655440010"`
	OrderNumber     string     `json:"order_number" example:"PO-2026-00001"`
	SupplierID      string     `json:"supplier_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	SupplierName    string     `json:"supplier_name" example:"供应商A"`
	WarehouseID     *string    `json:"warehouse_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
	ItemCount       int        `json:"item_count" example:"3"`
	TotalAmount     float64    `json:"total_amount" example:"1500.00"`
	PayableAmount   float64    `json:"payable_amount" example:"1400.00"`
	Status          string     `json:"status" example:"draft"`
	ReceiveProgress float64    `json:"receive_progress" example:"33.33"`
	ConfirmedAt     *time.Time `json:"confirmed_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// PurchaseOrderItemResponse represents an order item in API responses
// @Description Purchase order item response
type PurchaseOrderItemResponse struct {
	ID                string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440020"`
	ProductID         string    `json:"product_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	ProductName       string    `json:"product_name" example:"测试商品"`
	ProductCode       string    `json:"product_code" example:"SKU-001"`
	OrderedQuantity   float64   `json:"ordered_quantity" example:"10"`
	ReceivedQuantity  float64   `json:"received_quantity" example:"5"`
	RemainingQuantity float64   `json:"remaining_quantity" example:"5"`
	UnitCost          float64   `json:"unit_cost" example:"50.00"`
	Amount            float64   `json:"amount" example:"500.00"`
	Unit              string    `json:"unit" example:"pcs"`
	Remark            string    `json:"remark,omitempty" example:"商品备注"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ReceivedItemResponse represents received item info in responses
// @Description Received item info response
type ReceivedItemResponse struct {
	ItemID      string     `json:"item_id" example:"550e8400-e29b-41d4-a716-446655440020"`
	ProductID   string     `json:"product_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	ProductName string     `json:"product_name" example:"测试商品"`
	ProductCode string     `json:"product_code" example:"SKU-001"`
	Quantity    float64    `json:"quantity" example:"5"`
	UnitCost    float64    `json:"unit_cost" example:"50.00"`
	Unit        string     `json:"unit" example:"pcs"`
	BatchNumber string     `json:"batch_number,omitempty" example:"BATCH-2026-001"`
	ExpiryDate  *time.Time `json:"expiry_date,omitempty" example:"2027-12-31T00:00:00Z"`
}

// ReceiveResultResponse represents the result of a receive operation
// @Description Receive operation result response
type ReceiveResultResponse struct {
	Order           PurchaseOrderResponse  `json:"order"`
	ReceivedItems   []ReceivedItemResponse `json:"received_items"`
	IsFullyReceived bool                   `json:"is_fully_received" example:"false"`
}

// PurchaseOrderStatusSummaryResponse represents order count summary by status
// @Description Purchase order status summary response
type PurchaseOrderStatusSummaryResponse struct {
	Draft           int64 `json:"draft" example:"5"`
	Confirmed       int64 `json:"confirmed" example:"10"`
	PartialReceived int64 `json:"partial_received" example:"3"`
	Completed       int64 `json:"completed" example:"100"`
	Cancelled       int64 `json:"cancelled" example:"3"`
	Total           int64 `json:"total" example:"121"`
	PendingReceipt  int64 `json:"pending_receipt" example:"13"`
}

// Create godoc
// @ID           createPurchaseOrder
// @Summary      Create a new purchase order
// @Description  Create a new purchase order with optional items
// @Tags         purchase-orders
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        request body CreatePurchaseOrderRequest true "Purchase order creation request"
// @Success      201 {object} APIResponse[PurchaseOrderResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-orders [post]
func (h *PurchaseOrderHandler) Create(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req CreatePurchaseOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Convert to application DTO
	supplierID, err := uuid.Parse(req.SupplierID)
	if err != nil {
		h.BadRequest(c, "Invalid supplier ID format")
		return
	}

	// Get user ID from JWT context (optional, for data scope)
	userID, _ := getUserID(c)

	appReq := tradeapp.CreatePurchaseOrderRequest{
		SupplierID:   supplierID,
		SupplierName: req.SupplierName,
		Remark:       req.Remark,
	}

	// Set CreatedBy for data scope filtering
	if userID != uuid.Nil {
		appReq.CreatedBy = &userID
	}

	// Convert warehouse ID
	if req.WarehouseID != nil && *req.WarehouseID != "" {
		warehouseID, err := uuid.Parse(*req.WarehouseID)
		if err != nil {
			h.BadRequest(c, "Invalid warehouse ID format")
			return
		}
		appReq.WarehouseID = &warehouseID
	}

	// Convert items
	for _, item := range req.Items {
		productID, err := uuid.Parse(item.ProductID)
		if err != nil {
			h.BadRequest(c, "Invalid product ID format")
			return
		}

		// Set defaults for base_unit and conversion_rate if not provided
		baseUnit := item.BaseUnit
		if baseUnit == "" {
			baseUnit = item.Unit // Default to the same unit
		}
		conversionRate := item.ConversionRate
		if conversionRate <= 0 {
			conversionRate = 1.0 // Default conversion rate
		}

		appReq.Items = append(appReq.Items, tradeapp.CreatePurchaseOrderItemInput{
			ProductID:      productID,
			ProductName:    item.ProductName,
			ProductCode:    item.ProductCode,
			Unit:           item.Unit,
			BaseUnit:       baseUnit,
			Quantity:       decimal.NewFromFloat(item.Quantity),
			ConversionRate: decimal.NewFromFloat(conversionRate),
			UnitCost:       decimal.NewFromFloat(item.UnitCost),
			Remark:         item.Remark,
		})
	}

	// Convert discount
	if req.Discount != nil {
		d := decimal.NewFromFloat(*req.Discount)
		appReq.Discount = &d
	}

	order, err := h.orderService.Create(c.Request.Context(), tenantID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Created(c, toPurchaseOrderResponse(order))
}

// GetByID godoc
// @ID           getPurchaseOrderById
// @Summary      Get purchase order by ID
// @Description  Retrieve a purchase order by its ID
// @Tags         purchase-orders
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Purchase Order ID" format(uuid)
// @Success      200 {object} APIResponse[PurchaseOrderResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-orders/{id} [get]
func (h *PurchaseOrderHandler) GetByID(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid order ID format")
		return
	}

	order, err := h.orderService.GetByID(c.Request.Context(), tenantID, orderID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toPurchaseOrderResponse(order))
}

// GetByOrderNumber godoc
// @ID           getPurchaseOrderByOrderNumber
// @Summary      Get purchase order by order number
// @Description  Retrieve a purchase order by its order number
// @Tags         purchase-orders
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        order_number path string true "Order Number" example:"PO-2026-00001"
// @Success      200 {object} APIResponse[PurchaseOrderResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-orders/number/{order_number} [get]
func (h *PurchaseOrderHandler) GetByOrderNumber(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	orderNumber := c.Param("order_number")
	if orderNumber == "" {
		h.BadRequest(c, "Order number is required")
		return
	}

	order, err := h.orderService.GetByOrderNumber(c.Request.Context(), tenantID, orderNumber)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toPurchaseOrderResponse(order))
}

// List godoc
// @ID           listPurchaseOrders
// @Summary      List purchase orders
// @Description  Retrieve a paginated list of purchase orders with optional filtering
// @Tags         purchase-orders
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        search query string false "Search term (order number, supplier name)"
// @Param        supplier_id query string false "Supplier ID" format(uuid)
// @Param        warehouse_id query string false "Warehouse ID" format(uuid)
// @Param        status query string false "Order status" Enums(draft, confirmed, partial_received, completed, cancelled)
// @Param        statuses query []string false "Multiple order statuses"
// @Param        start_date query string false "Start date (ISO 8601)" format(date-time)
// @Param        end_date query string false "End date (ISO 8601)" format(date-time)
// @Param        min_amount query number false "Minimum payable amount"
// @Param        max_amount query number false "Maximum payable amount"
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20) maximum(100)
// @Param        order_by query string false "Order by field" default(created_at)
// @Param        order_dir query string false "Order direction" Enums(asc, desc) default(desc)
// @Success      200 {object} APIResponse[[]PurchaseOrderListResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-orders [get]
func (h *PurchaseOrderHandler) List(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var filter tradeapp.PurchaseOrderListFilter
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

	orders, total, err := h.orderService.List(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.SuccessWithMeta(c, toPurchaseOrderListResponses(orders), total, filter.Page, filter.PageSize)
}

// ListPendingReceipt godoc
// @ID           listPurchaseOrderPendingReceipt
// @Summary      List purchase orders pending receipt
// @Description  Retrieve purchase orders that are waiting to receive goods (CONFIRMED or PARTIAL_RECEIVED)
// @Tags         purchase-orders
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        search query string false "Search term"
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20) maximum(100)
// @Param        order_by query string false "Order by field" default(created_at)
// @Param        order_dir query string false "Order direction" Enums(asc, desc) default(desc)
// @Success      200 {object} APIResponse[[]PurchaseOrderListResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-orders/pending-receipt [get]
func (h *PurchaseOrderHandler) ListPendingReceipt(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var filter tradeapp.PurchaseOrderListFilter
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

	orders, total, err := h.orderService.ListPendingReceipt(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.SuccessWithMeta(c, toPurchaseOrderListResponses(orders), total, filter.Page, filter.PageSize)
}

// Update godoc
// @ID           updatePurchaseOrder
// @Summary      Update a purchase order
// @Description  Update a purchase order (only allowed in DRAFT status)
// @Tags         purchase-orders
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Purchase Order ID" format(uuid)
// @Param        request body UpdatePurchaseOrderRequest true "Purchase order update request"
// @Success      200 {object} APIResponse[PurchaseOrderResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      422 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-orders/{id} [put]
func (h *PurchaseOrderHandler) Update(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid order ID format")
		return
	}

	var req UpdatePurchaseOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Convert to application DTO
	appReq := tradeapp.UpdatePurchaseOrderRequest{}

	// Convert warehouse ID
	if req.WarehouseID != nil {
		if *req.WarehouseID == "" {
			// Clear warehouse
			nilUUID := uuid.Nil
			appReq.WarehouseID = &nilUUID
		} else {
			warehouseID, err := uuid.Parse(*req.WarehouseID)
			if err != nil {
				h.BadRequest(c, "Invalid warehouse ID format")
				return
			}
			appReq.WarehouseID = &warehouseID
		}
	}

	// Convert discount
	if req.Discount != nil {
		d := decimal.NewFromFloat(*req.Discount)
		appReq.Discount = &d
	}

	appReq.Remark = req.Remark

	order, err := h.orderService.Update(c.Request.Context(), tenantID, orderID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toPurchaseOrderResponse(order))
}

// Delete godoc
// @ID           deletePurchaseOrder
// @Summary      Delete a purchase order
// @Description  Delete a purchase order (only allowed in DRAFT status)
// @Tags         purchase-orders
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Purchase Order ID" format(uuid)
// @Success      204
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      422 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-orders/{id} [delete]
func (h *PurchaseOrderHandler) Delete(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid order ID format")
		return
	}

	err = h.orderService.Delete(c.Request.Context(), tenantID, orderID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.NoContent(c)
}

// AddItem godoc
// @ID           addPurchaseOrderItem
// @Summary      Add item to purchase order
// @Description  Add a new item to a purchase order (only allowed in DRAFT status)
// @Tags         purchase-orders
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Purchase Order ID" format(uuid)
// @Param        request body AddPurchaseOrderItemRequest true "Order item to add"
// @Success      200 {object} APIResponse[PurchaseOrderResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      422 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-orders/{id}/items [post]
func (h *PurchaseOrderHandler) AddItem(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid order ID format")
		return
	}

	var req AddPurchaseOrderItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	// Set defaults for base_unit and conversion_rate if not provided
	baseUnit := req.BaseUnit
	if baseUnit == "" {
		baseUnit = req.Unit // Default to the same unit
	}
	conversionRate := req.ConversionRate
	if conversionRate <= 0 {
		conversionRate = 1.0 // Default conversion rate
	}

	appReq := tradeapp.AddPurchaseOrderItemRequest{
		ProductID:      productID,
		ProductName:    req.ProductName,
		ProductCode:    req.ProductCode,
		Unit:           req.Unit,
		BaseUnit:       baseUnit,
		Quantity:       decimal.NewFromFloat(req.Quantity),
		ConversionRate: decimal.NewFromFloat(conversionRate),
		UnitCost:       decimal.NewFromFloat(req.UnitCost),
		Remark:         req.Remark,
	}

	order, err := h.orderService.AddItem(c.Request.Context(), tenantID, orderID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toPurchaseOrderResponse(order))
}

// UpdateItem godoc
// @ID           updatePurchaseOrderItem
// @Summary      Update order item
// @Description  Update an item in a purchase order (only allowed in DRAFT status)
// @Tags         purchase-orders
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Purchase Order ID" format(uuid)
// @Param        item_id path string true "Order Item ID" format(uuid)
// @Param        request body UpdatePurchaseOrderItemRequest true "Order item update request"
// @Success      200 {object} APIResponse[PurchaseOrderResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      422 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-orders/{id}/items/{item_id} [put]
func (h *PurchaseOrderHandler) UpdateItem(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid order ID format")
		return
	}

	itemID, err := uuid.Parse(c.Param("item_id"))
	if err != nil {
		h.BadRequest(c, "Invalid item ID format")
		return
	}

	var req UpdatePurchaseOrderItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	appReq := tradeapp.UpdatePurchaseOrderItemRequest{
		Remark: req.Remark,
	}

	if req.Quantity != nil {
		d := decimal.NewFromFloat(*req.Quantity)
		appReq.Quantity = &d
	}

	if req.UnitCost != nil {
		d := decimal.NewFromFloat(*req.UnitCost)
		appReq.UnitCost = &d
	}

	order, err := h.orderService.UpdateItem(c.Request.Context(), tenantID, orderID, itemID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toPurchaseOrderResponse(order))
}

// RemoveItem godoc
// @ID           removePurchaseOrderItem
// @Summary      Remove item from purchase order
// @Description  Remove an item from a purchase order (only allowed in DRAFT status)
// @Tags         purchase-orders
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Purchase Order ID" format(uuid)
// @Param        item_id path string true "Order Item ID" format(uuid)
// @Success      200 {object} APIResponse[PurchaseOrderResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      422 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-orders/{id}/items/{item_id} [delete]
func (h *PurchaseOrderHandler) RemoveItem(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid order ID format")
		return
	}

	itemID, err := uuid.Parse(c.Param("item_id"))
	if err != nil {
		h.BadRequest(c, "Invalid item ID format")
		return
	}

	order, err := h.orderService.RemoveItem(c.Request.Context(), tenantID, orderID, itemID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toPurchaseOrderResponse(order))
}

// Confirm godoc
// @ID           confirmPurchaseOrder
// @Summary      Confirm a purchase order
// @Description  Confirm a purchase order (transitions from DRAFT to CONFIRMED)
// @Tags         purchase-orders
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Purchase Order ID" format(uuid)
// @Param        request body ConfirmPurchaseOrderRequest false "Confirm order request"
// @Success      200 {object} APIResponse[PurchaseOrderResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      422 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-orders/{id}/confirm [post]
func (h *PurchaseOrderHandler) Confirm(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid order ID format")
		return
	}

	var req ConfirmPurchaseOrderRequest
	// Allow empty body
	_ = c.ShouldBindJSON(&req)

	appReq := tradeapp.ConfirmPurchaseOrderRequest{}

	if req.WarehouseID != nil && *req.WarehouseID != "" {
		warehouseID, err := uuid.Parse(*req.WarehouseID)
		if err != nil {
			h.BadRequest(c, "Invalid warehouse ID format")
			return
		}
		appReq.WarehouseID = &warehouseID
	}

	order, err := h.orderService.Confirm(c.Request.Context(), tenantID, orderID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toPurchaseOrderResponse(order))
}

// Receive godoc
// @ID           receivePurchaseOrder
// @Summary      Receive goods for a purchase order
// @Description  Process receipt of goods for a purchase order (supports partial receipt)
// @Tags         purchase-orders
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Purchase Order ID" format(uuid)
// @Param        request body ReceivePurchaseOrderRequest true "Receive goods request"
// @Success      200 {object} APIResponse[ReceiveResultResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      422 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-orders/{id}/receive [post]
func (h *PurchaseOrderHandler) Receive(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid order ID format")
		return
	}

	var req ReceivePurchaseOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	appReq := tradeapp.ReceivePurchaseOrderRequest{}

	// Convert warehouse ID
	if req.WarehouseID != nil && *req.WarehouseID != "" {
		warehouseID, err := uuid.Parse(*req.WarehouseID)
		if err != nil {
			h.BadRequest(c, "Invalid warehouse ID format")
			return
		}
		appReq.WarehouseID = &warehouseID
	}

	// Convert items
	for _, item := range req.Items {
		productID, err := uuid.Parse(item.ProductID)
		if err != nil {
			h.BadRequest(c, "Invalid product ID format")
			return
		}
		input := tradeapp.ReceiveItemInput{
			ProductID:   productID,
			Quantity:    decimal.NewFromFloat(item.Quantity),
			BatchNumber: item.BatchNumber,
			ExpiryDate:  item.ExpiryDate,
		}
		if item.UnitCost != nil {
			cost := decimal.NewFromFloat(*item.UnitCost)
			input.UnitCost = &cost
		}
		appReq.Items = append(appReq.Items, input)
	}

	result, err := h.orderService.Receive(c.Request.Context(), tenantID, orderID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toReceiveResultResponse(result))
}

// Cancel godoc
// @ID           cancelPurchaseOrder
// @Summary      Cancel a purchase order
// @Description  Cancel a purchase order (only from DRAFT or CONFIRMED status)
// @Tags         purchase-orders
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Purchase Order ID" format(uuid)
// @Param        request body CancelPurchaseOrderRequest true "Cancel order request"
// @Success      200 {object} APIResponse[PurchaseOrderResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      422 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-orders/{id}/cancel [post]
func (h *PurchaseOrderHandler) Cancel(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid order ID format")
		return
	}

	var req CancelPurchaseOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	appReq := tradeapp.CancelPurchaseOrderRequest{
		Reason: req.Reason,
	}

	order, err := h.orderService.Cancel(c.Request.Context(), tenantID, orderID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toPurchaseOrderResponse(order))
}

// GetStatusSummary godoc
// @ID           getPurchaseOrderStatusSummary
// @Summary      Get purchase order status summary
// @Description  Get count of purchase orders by status for dashboard
// @Tags         purchase-orders
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Success      200 {object} APIResponse[PurchaseOrderStatusSummaryResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-orders/stats/summary [get]
func (h *PurchaseOrderHandler) GetStatusSummary(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	summary, err := h.orderService.GetStatusSummary(c.Request.Context(), tenantID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, PurchaseOrderStatusSummaryResponse{
		Draft:           summary.Draft,
		Confirmed:       summary.Confirmed,
		PartialReceived: summary.PartialReceived,
		Completed:       summary.Completed,
		Cancelled:       summary.Cancelled,
		Total:           summary.Total,
		PendingReceipt:  summary.PendingReceipt,
	})
}

// GetReceivableItems godoc
// @ID           getPurchaseOrderReceivableItems
// @Summary      Get receivable items for a purchase order
// @Description  Get items that can still receive goods for a purchase order
// @Tags         purchase-orders
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Purchase Order ID" format(uuid)
// @Success      200 {object} APIResponse[[]PurchaseOrderItemResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      422 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-orders/{id}/receivable-items [get]
func (h *PurchaseOrderHandler) GetReceivableItems(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid order ID format")
		return
	}

	items, err := h.orderService.GetReceivableItems(c.Request.Context(), tenantID, orderID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toPurchaseOrderItemResponses(items))
}

// toPurchaseOrderResponse converts application response to handler response
func toPurchaseOrderResponse(order *tradeapp.PurchaseOrderResponse) PurchaseOrderResponse {
	items := make([]PurchaseOrderItemResponse, len(order.Items))
	for i, item := range order.Items {
		items[i] = PurchaseOrderItemResponse{
			ID:                item.ID.String(),
			ProductID:         item.ProductID.String(),
			ProductName:       item.ProductName,
			ProductCode:       item.ProductCode,
			OrderedQuantity:   item.OrderedQuantity.InexactFloat64(),
			ReceivedQuantity:  item.ReceivedQuantity.InexactFloat64(),
			RemainingQuantity: item.RemainingQuantity.InexactFloat64(),
			UnitCost:          item.UnitCost.InexactFloat64(),
			Amount:            item.Amount.InexactFloat64(),
			Unit:              item.Unit,
			Remark:            item.Remark,
			CreatedAt:         item.CreatedAt,
			UpdatedAt:         item.UpdatedAt,
		}
	}

	resp := PurchaseOrderResponse{
		ID:               order.ID.String(),
		TenantID:         order.TenantID.String(),
		OrderNumber:      order.OrderNumber,
		SupplierID:       order.SupplierID.String(),
		SupplierName:     order.SupplierName,
		Items:            items,
		ItemCount:        order.ItemCount,
		TotalQuantity:    order.TotalQuantity.InexactFloat64(),
		ReceivedQuantity: order.ReceivedQuantity.InexactFloat64(),
		TotalAmount:      order.TotalAmount.InexactFloat64(),
		DiscountAmount:   order.DiscountAmount.InexactFloat64(),
		PayableAmount:    order.PayableAmount.InexactFloat64(),
		Status:           order.Status,
		ReceiveProgress:  order.ReceiveProgress.InexactFloat64(),
		Remark:           order.Remark,
		ConfirmedAt:      order.ConfirmedAt,
		CompletedAt:      order.CompletedAt,
		CancelledAt:      order.CancelledAt,
		CancelReason:     order.CancelReason,
		CreatedAt:        order.CreatedAt,
		UpdatedAt:        order.UpdatedAt,
		Version:          order.Version,
	}

	if order.WarehouseID != nil {
		warehouseID := order.WarehouseID.String()
		resp.WarehouseID = &warehouseID
	}

	return resp
}

// toPurchaseOrderListResponses converts application list responses to handler responses
func toPurchaseOrderListResponses(orders []tradeapp.PurchaseOrderListItemResponse) []PurchaseOrderListResponse {
	responses := make([]PurchaseOrderListResponse, len(orders))
	for i, order := range orders {
		resp := PurchaseOrderListResponse{
			ID:              order.ID.String(),
			OrderNumber:     order.OrderNumber,
			SupplierID:      order.SupplierID.String(),
			SupplierName:    order.SupplierName,
			ItemCount:       order.ItemCount,
			TotalAmount:     order.TotalAmount.InexactFloat64(),
			PayableAmount:   order.PayableAmount.InexactFloat64(),
			Status:          order.Status,
			ReceiveProgress: order.ReceiveProgress.InexactFloat64(),
			ConfirmedAt:     order.ConfirmedAt,
			CompletedAt:     order.CompletedAt,
			CreatedAt:       order.CreatedAt,
			UpdatedAt:       order.UpdatedAt,
		}

		if order.WarehouseID != nil {
			warehouseID := order.WarehouseID.String()
			resp.WarehouseID = &warehouseID
		}

		responses[i] = resp
	}
	return responses
}

// toPurchaseOrderItemResponses converts application item responses to handler responses
func toPurchaseOrderItemResponses(items []tradeapp.PurchaseOrderItemResponse) []PurchaseOrderItemResponse {
	responses := make([]PurchaseOrderItemResponse, len(items))
	for i, item := range items {
		responses[i] = PurchaseOrderItemResponse{
			ID:                item.ID.String(),
			ProductID:         item.ProductID.String(),
			ProductName:       item.ProductName,
			ProductCode:       item.ProductCode,
			OrderedQuantity:   item.OrderedQuantity.InexactFloat64(),
			ReceivedQuantity:  item.ReceivedQuantity.InexactFloat64(),
			RemainingQuantity: item.RemainingQuantity.InexactFloat64(),
			UnitCost:          item.UnitCost.InexactFloat64(),
			Amount:            item.Amount.InexactFloat64(),
			Unit:              item.Unit,
			Remark:            item.Remark,
			CreatedAt:         item.CreatedAt,
			UpdatedAt:         item.UpdatedAt,
		}
	}
	return responses
}

// toReceiveResultResponse converts application receive result to handler response
func toReceiveResultResponse(result *tradeapp.ReceiveResultResponse) ReceiveResultResponse {
	receivedItems := make([]ReceivedItemResponse, len(result.ReceivedItems))
	for i, item := range result.ReceivedItems {
		receivedItems[i] = ReceivedItemResponse{
			ItemID:      item.ItemID.String(),
			ProductID:   item.ProductID.String(),
			ProductName: item.ProductName,
			ProductCode: item.ProductCode,
			Quantity:    item.Quantity.InexactFloat64(),
			UnitCost:    item.UnitCost.InexactFloat64(),
			Unit:        item.Unit,
			BatchNumber: item.BatchNumber,
			ExpiryDate:  item.ExpiryDate,
		}
	}

	return ReceiveResultResponse{
		Order:           toPurchaseOrderResponse(&result.Order),
		ReceivedItems:   receivedItems,
		IsFullyReceived: result.IsFullyReceived,
	}
}

// Helper function to suppress unused import warning
var _ = dto.Response{}
var _ = trade.PurchaseOrderStatusDraft
