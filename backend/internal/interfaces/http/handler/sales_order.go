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

// SalesOrderHandler handles sales order-related API endpoints
type SalesOrderHandler struct {
	BaseHandler
	orderService *tradeapp.SalesOrderService
}

// NewSalesOrderHandler creates a new SalesOrderHandler
func NewSalesOrderHandler(orderService *tradeapp.SalesOrderService) *SalesOrderHandler {
	return &SalesOrderHandler{
		orderService: orderService,
	}
}

// CreateSalesOrderRequest represents a request to create a new sales order
// @Description Request body for creating a new sales order
type CreateSalesOrderRequest struct {
	CustomerID   string                      `json:"customer_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	CustomerName string                      `json:"customer_name" binding:"required,min=1,max=200" example:"张三"`
	WarehouseID  *string                     `json:"warehouse_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Items        []CreateSalesOrderItemInput `json:"items"`
	Discount     *float64                    `json:"discount" example:"100.00"`
	Remark       string                      `json:"remark" example:"备注信息"`
}

// CreateSalesOrderItemInput represents an item in the create order request
// @Description Order item for creation
type CreateSalesOrderItemInput struct {
	ProductID   string  `json:"product_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440002"`
	ProductName string  `json:"product_name" binding:"required,min=1,max=200" example:"测试商品"`
	ProductCode string  `json:"product_code" binding:"required,min=1,max=50" example:"SKU-001"`
	Unit        string  `json:"unit" binding:"required,min=1,max=20" example:"pcs"`
	Quantity    float64 `json:"quantity" binding:"required,gt=0" example:"10"`
	UnitPrice   float64 `json:"unit_price" binding:"required,gt=0" example:"99.99"`
	Remark      string  `json:"remark" example:"商品备注"`
}

// UpdateSalesOrderRequest represents a request to update a sales order
// @Description Request body for updating a sales order (draft only)
type UpdateSalesOrderRequest struct {
	WarehouseID *string  `json:"warehouse_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Discount    *float64 `json:"discount" example:"50.00"`
	Remark      *string  `json:"remark" example:"更新备注"`
}

// AddOrderItemRequest represents a request to add an item to an order
// @Description Request body for adding an item to an order
type AddOrderItemRequest struct {
	ProductID   string  `json:"product_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440002"`
	ProductName string  `json:"product_name" binding:"required,min=1,max=200" example:"测试商品"`
	ProductCode string  `json:"product_code" binding:"required,min=1,max=50" example:"SKU-001"`
	Unit        string  `json:"unit" binding:"required,min=1,max=20" example:"pcs"`
	Quantity    float64 `json:"quantity" binding:"required,gt=0" example:"5"`
	UnitPrice   float64 `json:"unit_price" binding:"required,gt=0" example:"199.99"`
	Remark      string  `json:"remark" example:"商品备注"`
}

// UpdateOrderItemRequest represents a request to update an order item
// @Description Request body for updating an order item
type UpdateOrderItemRequest struct {
	Quantity  *float64 `json:"quantity" example:"8"`
	UnitPrice *float64 `json:"unit_price" example:"89.99"`
	Remark    *string  `json:"remark" example:"更新商品备注"`
}

// ConfirmOrderRequest represents a request to confirm an order
// @Description Request body for confirming an order
type ConfirmOrderRequest struct {
	WarehouseID *string `json:"warehouse_id" example:"550e8400-e29b-41d4-a716-446655440001"`
}

// ShipOrderRequest represents a request to ship an order
// @Description Request body for shipping an order
type ShipOrderRequest struct {
	WarehouseID *string `json:"warehouse_id" example:"550e8400-e29b-41d4-a716-446655440001"`
}

// CancelOrderRequest represents a request to cancel an order
// @Description Request body for cancelling an order
type CancelOrderRequest struct {
	Reason string `json:"reason" binding:"required,min=1,max=500" example:"客户取消订单"`
}

// SalesOrderResponse represents a sales order in API responses
// @Description Sales order response
type SalesOrderResponse struct {
	ID             string                   `json:"id" example:"550e8400-e29b-41d4-a716-446655440010"`
	TenantID       string                   `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	OrderNumber    string                   `json:"order_number" example:"SO-2026-00001"`
	CustomerID     string                   `json:"customer_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	CustomerName   string                   `json:"customer_name" example:"张三"`
	WarehouseID    *string                  `json:"warehouse_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
	Items          []SalesOrderItemResponse `json:"items"`
	ItemCount      int                      `json:"item_count" example:"3"`
	TotalQuantity  float64                  `json:"total_quantity" example:"30"`
	TotalAmount    float64                  `json:"total_amount" example:"2999.70"`
	DiscountAmount float64                  `json:"discount_amount" example:"100.00"`
	PayableAmount  float64                  `json:"payable_amount" example:"2899.70"`
	Status         string                   `json:"status" example:"draft"`
	Remark         string                   `json:"remark" example:"备注信息"`
	ConfirmedAt    *time.Time               `json:"confirmed_at,omitempty"`
	ShippedAt      *time.Time               `json:"shipped_at,omitempty"`
	CompletedAt    *time.Time               `json:"completed_at,omitempty"`
	CancelledAt    *time.Time               `json:"cancelled_at,omitempty"`
	CancelReason   string                   `json:"cancel_reason,omitempty" example:""`
	CreatedAt      time.Time                `json:"created_at"`
	UpdatedAt      time.Time                `json:"updated_at"`
	Version        int                      `json:"version" example:"1"`
}

// SalesOrderListResponse represents a sales order in list responses
// @Description Sales order list item response
type SalesOrderListResponse struct {
	ID            string     `json:"id" example:"550e8400-e29b-41d4-a716-446655440010"`
	OrderNumber   string     `json:"order_number" example:"SO-2026-00001"`
	CustomerID    string     `json:"customer_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	CustomerName  string     `json:"customer_name" example:"张三"`
	WarehouseID   *string    `json:"warehouse_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
	ItemCount     int        `json:"item_count" example:"3"`
	TotalAmount   float64    `json:"total_amount" example:"2999.70"`
	PayableAmount float64    `json:"payable_amount" example:"2899.70"`
	Status        string     `json:"status" example:"draft"`
	ConfirmedAt   *time.Time `json:"confirmed_at,omitempty"`
	ShippedAt     *time.Time `json:"shipped_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// SalesOrderItemResponse represents an order item in API responses
// @Description Sales order item response
type SalesOrderItemResponse struct {
	ID          string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440020"`
	ProductID   string    `json:"product_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	ProductName string    `json:"product_name" example:"测试商品"`
	ProductCode string    `json:"product_code" example:"SKU-001"`
	Quantity    float64   `json:"quantity" example:"10"`
	UnitPrice   float64   `json:"unit_price" example:"99.99"`
	Amount      float64   `json:"amount" example:"999.90"`
	Unit        string    `json:"unit" example:"pcs"`
	Remark      string    `json:"remark,omitempty" example:"商品备注"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// OrderStatusSummaryResponse represents order count summary by status
// @Description Order status summary response
type OrderStatusSummaryResponse struct {
	Draft     int64 `json:"draft" example:"5"`
	Confirmed int64 `json:"confirmed" example:"10"`
	Shipped   int64 `json:"shipped" example:"8"`
	Completed int64 `json:"completed" example:"100"`
	Cancelled int64 `json:"cancelled" example:"3"`
	Total     int64 `json:"total" example:"126"`
}

// Create godoc
// @Summary      Create a new sales order
// @Description  Create a new sales order with optional items
// @Tags         sales-orders
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        request body CreateSalesOrderRequest true "Sales order creation request"
// @Success      201 {object} dto.Response{data=SalesOrderResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /trade/sales-orders [post]
func (h *SalesOrderHandler) Create(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req CreateSalesOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Convert to application DTO
	customerID, err := uuid.Parse(req.CustomerID)
	if err != nil {
		h.BadRequest(c, "Invalid customer ID format")
		return
	}

	appReq := tradeapp.CreateSalesOrderRequest{
		CustomerID:   customerID,
		CustomerName: req.CustomerName,
		Remark:       req.Remark,
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
		appReq.Items = append(appReq.Items, tradeapp.CreateSalesOrderItemInput{
			ProductID:   productID,
			ProductName: item.ProductName,
			ProductCode: item.ProductCode,
			Unit:        item.Unit,
			Quantity:    decimal.NewFromFloat(item.Quantity),
			UnitPrice:   decimal.NewFromFloat(item.UnitPrice),
			Remark:      item.Remark,
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

	h.Created(c, toSalesOrderResponse(order))
}

// GetByID godoc
// @Summary      Get sales order by ID
// @Description  Retrieve a sales order by its ID
// @Tags         sales-orders
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Sales Order ID" format(uuid)
// @Success      200 {object} dto.Response{data=SalesOrderResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /trade/sales-orders/{id} [get]
func (h *SalesOrderHandler) GetByID(c *gin.Context) {
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

	h.Success(c, toSalesOrderResponse(order))
}

// GetByOrderNumber godoc
// @Summary      Get sales order by order number
// @Description  Retrieve a sales order by its order number
// @Tags         sales-orders
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        order_number path string true "Order Number" example:"SO-2026-00001"
// @Success      200 {object} dto.Response{data=SalesOrderResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /trade/sales-orders/number/{order_number} [get]
func (h *SalesOrderHandler) GetByOrderNumber(c *gin.Context) {
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

	h.Success(c, toSalesOrderResponse(order))
}

// List godoc
// @Summary      List sales orders
// @Description  Retrieve a paginated list of sales orders with optional filtering
// @Tags         sales-orders
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        search query string false "Search term (order number, customer name)"
// @Param        customer_id query string false "Customer ID" format(uuid)
// @Param        warehouse_id query string false "Warehouse ID" format(uuid)
// @Param        status query string false "Order status" Enums(draft, confirmed, shipped, completed, cancelled)
// @Param        statuses query []string false "Multiple order statuses"
// @Param        start_date query string false "Start date (ISO 8601)" format(date-time)
// @Param        end_date query string false "End date (ISO 8601)" format(date-time)
// @Param        min_amount query number false "Minimum payable amount"
// @Param        max_amount query number false "Maximum payable amount"
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20) maximum(100)
// @Param        order_by query string false "Order by field" default(created_at)
// @Param        order_dir query string false "Order direction" Enums(asc, desc) default(desc)
// @Success      200 {object} dto.Response{data=[]SalesOrderListResponse,meta=dto.Meta}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /trade/sales-orders [get]
func (h *SalesOrderHandler) List(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var filter tradeapp.SalesOrderListFilter
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

	h.SuccessWithMeta(c, toSalesOrderListResponses(orders), total, filter.Page, filter.PageSize)
}

// Update godoc
// @Summary      Update a sales order
// @Description  Update a sales order (only allowed in DRAFT status)
// @Tags         sales-orders
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Sales Order ID" format(uuid)
// @Param        request body UpdateSalesOrderRequest true "Sales order update request"
// @Success      200 {object} dto.Response{data=SalesOrderResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /trade/sales-orders/{id} [put]
func (h *SalesOrderHandler) Update(c *gin.Context) {
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

	var req UpdateSalesOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Convert to application DTO
	appReq := tradeapp.UpdateSalesOrderRequest{}

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

	h.Success(c, toSalesOrderResponse(order))
}

// Delete godoc
// @Summary      Delete a sales order
// @Description  Delete a sales order (only allowed in DRAFT status)
// @Tags         sales-orders
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Sales Order ID" format(uuid)
// @Success      204
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /trade/sales-orders/{id} [delete]
func (h *SalesOrderHandler) Delete(c *gin.Context) {
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
// @Summary      Add item to sales order
// @Description  Add a new item to a sales order (only allowed in DRAFT status)
// @Tags         sales-orders
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Sales Order ID" format(uuid)
// @Param        request body AddOrderItemRequest true "Order item to add"
// @Success      200 {object} dto.Response{data=SalesOrderResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /trade/sales-orders/{id}/items [post]
func (h *SalesOrderHandler) AddItem(c *gin.Context) {
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

	var req AddOrderItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	appReq := tradeapp.AddOrderItemRequest{
		ProductID:   productID,
		ProductName: req.ProductName,
		ProductCode: req.ProductCode,
		Unit:        req.Unit,
		Quantity:    decimal.NewFromFloat(req.Quantity),
		UnitPrice:   decimal.NewFromFloat(req.UnitPrice),
		Remark:      req.Remark,
	}

	order, err := h.orderService.AddItem(c.Request.Context(), tenantID, orderID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toSalesOrderResponse(order))
}

// UpdateItem godoc
// @Summary      Update order item
// @Description  Update an item in a sales order (only allowed in DRAFT status)
// @Tags         sales-orders
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Sales Order ID" format(uuid)
// @Param        item_id path string true "Order Item ID" format(uuid)
// @Param        request body UpdateOrderItemRequest true "Order item update request"
// @Success      200 {object} dto.Response{data=SalesOrderResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /trade/sales-orders/{id}/items/{item_id} [put]
func (h *SalesOrderHandler) UpdateItem(c *gin.Context) {
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

	var req UpdateOrderItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	appReq := tradeapp.UpdateOrderItemRequest{
		Remark: req.Remark,
	}

	if req.Quantity != nil {
		d := decimal.NewFromFloat(*req.Quantity)
		appReq.Quantity = &d
	}

	if req.UnitPrice != nil {
		d := decimal.NewFromFloat(*req.UnitPrice)
		appReq.UnitPrice = &d
	}

	order, err := h.orderService.UpdateItem(c.Request.Context(), tenantID, orderID, itemID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toSalesOrderResponse(order))
}

// RemoveItem godoc
// @Summary      Remove item from sales order
// @Description  Remove an item from a sales order (only allowed in DRAFT status)
// @Tags         sales-orders
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Sales Order ID" format(uuid)
// @Param        item_id path string true "Order Item ID" format(uuid)
// @Success      200 {object} dto.Response{data=SalesOrderResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /trade/sales-orders/{id}/items/{item_id} [delete]
func (h *SalesOrderHandler) RemoveItem(c *gin.Context) {
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

	h.Success(c, toSalesOrderResponse(order))
}

// Confirm godoc
// @Summary      Confirm a sales order
// @Description  Confirm a sales order (transitions from DRAFT to CONFIRMED)
// @Tags         sales-orders
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Sales Order ID" format(uuid)
// @Param        request body ConfirmOrderRequest false "Confirm order request"
// @Success      200 {object} dto.Response{data=SalesOrderResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /trade/sales-orders/{id}/confirm [post]
func (h *SalesOrderHandler) Confirm(c *gin.Context) {
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

	var req ConfirmOrderRequest
	// Allow empty body
	_ = c.ShouldBindJSON(&req)

	appReq := tradeapp.ConfirmOrderRequest{}

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

	h.Success(c, toSalesOrderResponse(order))
}

// Ship godoc
// @Summary      Ship a sales order
// @Description  Mark a sales order as shipped (transitions from CONFIRMED to SHIPPED)
// @Tags         sales-orders
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Sales Order ID" format(uuid)
// @Param        request body ShipOrderRequest false "Ship order request"
// @Success      200 {object} dto.Response{data=SalesOrderResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /trade/sales-orders/{id}/ship [post]
func (h *SalesOrderHandler) Ship(c *gin.Context) {
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

	var req ShipOrderRequest
	// Allow empty body
	_ = c.ShouldBindJSON(&req)

	appReq := tradeapp.ShipOrderRequest{}

	if req.WarehouseID != nil && *req.WarehouseID != "" {
		warehouseID, err := uuid.Parse(*req.WarehouseID)
		if err != nil {
			h.BadRequest(c, "Invalid warehouse ID format")
			return
		}
		appReq.WarehouseID = &warehouseID
	}

	order, err := h.orderService.Ship(c.Request.Context(), tenantID, orderID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toSalesOrderResponse(order))
}

// Complete godoc
// @Summary      Complete a sales order
// @Description  Mark a sales order as completed (transitions from SHIPPED to COMPLETED)
// @Tags         sales-orders
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Sales Order ID" format(uuid)
// @Success      200 {object} dto.Response{data=SalesOrderResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /trade/sales-orders/{id}/complete [post]
func (h *SalesOrderHandler) Complete(c *gin.Context) {
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

	order, err := h.orderService.Complete(c.Request.Context(), tenantID, orderID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toSalesOrderResponse(order))
}

// Cancel godoc
// @Summary      Cancel a sales order
// @Description  Cancel a sales order (from DRAFT or CONFIRMED status)
// @Tags         sales-orders
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Sales Order ID" format(uuid)
// @Param        request body CancelOrderRequest true "Cancel order request"
// @Success      200 {object} dto.Response{data=SalesOrderResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /trade/sales-orders/{id}/cancel [post]
func (h *SalesOrderHandler) Cancel(c *gin.Context) {
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

	var req CancelOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	appReq := tradeapp.CancelOrderRequest{
		Reason: req.Reason,
	}

	order, err := h.orderService.Cancel(c.Request.Context(), tenantID, orderID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toSalesOrderResponse(order))
}

// GetStatusSummary godoc
// @Summary      Get order status summary
// @Description  Get count of orders by status for dashboard
// @Tags         sales-orders
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Success      200 {object} dto.Response{data=OrderStatusSummaryResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /trade/sales-orders/stats/summary [get]
func (h *SalesOrderHandler) GetStatusSummary(c *gin.Context) {
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

	h.Success(c, OrderStatusSummaryResponse{
		Draft:     summary.Draft,
		Confirmed: summary.Confirmed,
		Shipped:   summary.Shipped,
		Completed: summary.Completed,
		Cancelled: summary.Cancelled,
		Total:     summary.Total,
	})
}

// toSalesOrderResponse converts application response to handler response
func toSalesOrderResponse(order *tradeapp.SalesOrderResponse) SalesOrderResponse {
	items := make([]SalesOrderItemResponse, len(order.Items))
	for i, item := range order.Items {
		items[i] = SalesOrderItemResponse{
			ID:          item.ID.String(),
			ProductID:   item.ProductID.String(),
			ProductName: item.ProductName,
			ProductCode: item.ProductCode,
			Quantity:    item.Quantity.InexactFloat64(),
			UnitPrice:   item.UnitPrice.InexactFloat64(),
			Amount:      item.Amount.InexactFloat64(),
			Unit:        item.Unit,
			Remark:      item.Remark,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		}
	}

	resp := SalesOrderResponse{
		ID:             order.ID.String(),
		TenantID:       order.TenantID.String(),
		OrderNumber:    order.OrderNumber,
		CustomerID:     order.CustomerID.String(),
		CustomerName:   order.CustomerName,
		Items:          items,
		ItemCount:      order.ItemCount,
		TotalQuantity:  order.TotalQuantity.InexactFloat64(),
		TotalAmount:    order.TotalAmount.InexactFloat64(),
		DiscountAmount: order.DiscountAmount.InexactFloat64(),
		PayableAmount:  order.PayableAmount.InexactFloat64(),
		Status:         order.Status,
		Remark:         order.Remark,
		ConfirmedAt:    order.ConfirmedAt,
		ShippedAt:      order.ShippedAt,
		CompletedAt:    order.CompletedAt,
		CancelledAt:    order.CancelledAt,
		CancelReason:   order.CancelReason,
		CreatedAt:      order.CreatedAt,
		UpdatedAt:      order.UpdatedAt,
		Version:        order.Version,
	}

	if order.WarehouseID != nil {
		warehouseID := order.WarehouseID.String()
		resp.WarehouseID = &warehouseID
	}

	return resp
}

// toSalesOrderListResponses converts application list responses to handler responses
func toSalesOrderListResponses(orders []tradeapp.SalesOrderListItemResponse) []SalesOrderListResponse {
	responses := make([]SalesOrderListResponse, len(orders))
	for i, order := range orders {
		resp := SalesOrderListResponse{
			ID:            order.ID.String(),
			OrderNumber:   order.OrderNumber,
			CustomerID:    order.CustomerID.String(),
			CustomerName:  order.CustomerName,
			ItemCount:     order.ItemCount,
			TotalAmount:   order.TotalAmount.InexactFloat64(),
			PayableAmount: order.PayableAmount.InexactFloat64(),
			Status:        order.Status,
			ConfirmedAt:   order.ConfirmedAt,
			ShippedAt:     order.ShippedAt,
			CreatedAt:     order.CreatedAt,
			UpdatedAt:     order.UpdatedAt,
		}

		if order.WarehouseID != nil {
			warehouseID := order.WarehouseID.String()
			resp.WarehouseID = &warehouseID
		}

		responses[i] = resp
	}
	return responses
}

// Helper function to suppress unused import warning
var _ = dto.Response{}
var _ = trade.OrderStatusDraft
