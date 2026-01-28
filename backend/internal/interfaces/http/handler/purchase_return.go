package handler

import (
	"time"

	tradeapp "github.com/erp/backend/internal/application/trade"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/erp/backend/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PurchaseReturnHandler handles purchase return-related API endpoints
type PurchaseReturnHandler struct {
	BaseHandler
	returnService *tradeapp.PurchaseReturnService
}

// NewPurchaseReturnHandler creates a new PurchaseReturnHandler
func NewPurchaseReturnHandler(returnService *tradeapp.PurchaseReturnService) *PurchaseReturnHandler {
	return &PurchaseReturnHandler{
		returnService: returnService,
	}
}

// CreatePurchaseReturnRequest represents a request to create a new purchase return
// @Description Request body for creating a new purchase return
type CreatePurchaseReturnRequest struct {
	PurchaseOrderID string                          `json:"purchase_order_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	WarehouseID     *string                         `json:"warehouse_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Items           []CreatePurchaseReturnItemInput `json:"items" binding:"required,min=1"`
	Reason          string                          `json:"reason" example:"商品质量问题"`
	Remark          string                          `json:"remark" example:"备注信息"`
}

// CreatePurchaseReturnItemInput represents an item in the create return request
// @Description Return item for creation
type CreatePurchaseReturnItemInput struct {
	PurchaseOrderItemID string  `json:"purchase_order_item_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440002"`
	ReturnQuantity      float64 `json:"return_quantity" binding:"required,gt=0" example:"5"`
	Reason              string  `json:"reason" example:"商品损坏"`
	ConditionOnReturn   string  `json:"condition_on_return" example:"defective"`
	BatchNumber         string  `json:"batch_number" example:"BATCH-001"`
}

// UpdatePurchaseReturnRequest represents a request to update a purchase return
// @Description Request body for updating a purchase return (draft only)
type UpdatePurchaseReturnRequest struct {
	WarehouseID *string `json:"warehouse_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Reason      *string `json:"reason" example:"更新退货原因"`
	Remark      *string `json:"remark" example:"更新备注"`
}

// AddPurchaseReturnItemRequest represents a request to add an item to a return
// @Description Request body for adding an item to a return
type AddPurchaseReturnItemRequest struct {
	PurchaseOrderItemID string  `json:"purchase_order_item_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440002"`
	ReturnQuantity      float64 `json:"return_quantity" binding:"required,gt=0" example:"3"`
	Reason              string  `json:"reason" example:"商品有瑕疵"`
	ConditionOnReturn   string  `json:"condition_on_return" example:"defective"`
	BatchNumber         string  `json:"batch_number" example:"BATCH-001"`
}

// UpdatePurchaseReturnItemRequest represents a request to update a return item
// @Description Request body for updating a return item
type UpdatePurchaseReturnItemRequest struct {
	ReturnQuantity    *float64 `json:"return_quantity" example:"4"`
	Reason            *string  `json:"reason" example:"更新原因"`
	ConditionOnReturn *string  `json:"condition_on_return" example:"wrong_item"`
	BatchNumber       *string  `json:"batch_number" example:"BATCH-002"`
}

// ApprovePurchaseReturnRequest represents a request to approve a return
// @Description Request body for approving a return
type ApprovePurchaseReturnRequest struct {
	Note string `json:"note" example:"审批通过，同意退货"`
}

// RejectPurchaseReturnRequest represents a request to reject a return
// @Description Request body for rejecting a return
type RejectPurchaseReturnRequest struct {
	Reason string `json:"reason" binding:"required,min=1,max=500" example:"不符合退货条件"`
}

// CancelPurchaseReturnRequest represents a request to cancel a return
// @Description Request body for cancelling a return
type CancelPurchaseReturnRequest struct {
	Reason string `json:"reason" binding:"required,min=1,max=500" example:"供应商协商取消"`
}

// ShipPurchaseReturnRequest represents a request to ship a return
// @Description Request body for shipping a return back to supplier
type ShipPurchaseReturnRequest struct {
	TrackingNumber string `json:"tracking_number" example:"SF1234567890"`
	Note           string `json:"note" example:"已发货，预计3天到达"`
}

// PurchaseReturnResponse represents a purchase return in API responses
// @Description Purchase return response
type PurchaseReturnResponse struct {
	ID                  string                       `json:"id" example:"550e8400-e29b-41d4-a716-446655440010"`
	TenantID            string                       `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ReturnNumber        string                       `json:"return_number" example:"PR-2026-00001"`
	PurchaseOrderID     string                       `json:"purchase_order_id" example:"550e8400-e29b-41d4-a716-446655440020"`
	PurchaseOrderNumber string                       `json:"purchase_order_number" example:"PO-2026-00001"`
	SupplierID          string                       `json:"supplier_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	SupplierName        string                       `json:"supplier_name" example:"优质供应商"`
	WarehouseID         *string                      `json:"warehouse_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
	Items               []PurchaseReturnItemResponse `json:"items"`
	ItemCount           int                          `json:"item_count" example:"2"`
	TotalQuantity       float64                      `json:"total_quantity" example:"8"`
	TotalRefund         float64                      `json:"total_refund" example:"799.92"`
	Status              string                       `json:"status" example:"draft"`
	Reason              string                       `json:"reason,omitempty" example:"商品质量问题"`
	Remark              string                       `json:"remark,omitempty" example:"备注信息"`
	SubmittedAt         *time.Time                   `json:"submitted_at,omitempty"`
	ApprovedAt          *time.Time                   `json:"approved_at,omitempty"`
	ApprovedBy          *string                      `json:"approved_by,omitempty"`
	ApprovalNote        string                       `json:"approval_note,omitempty"`
	RejectedAt          *time.Time                   `json:"rejected_at,omitempty"`
	RejectedBy          *string                      `json:"rejected_by,omitempty"`
	RejectionReason     string                       `json:"rejection_reason,omitempty"`
	ShippedAt           *time.Time                   `json:"shipped_at,omitempty"`
	ShippedBy           *string                      `json:"shipped_by,omitempty"`
	ShippingNote        string                       `json:"shipping_note,omitempty"`
	TrackingNumber      string                       `json:"tracking_number,omitempty"`
	CompletedAt         *time.Time                   `json:"completed_at,omitempty"`
	CancelledAt         *time.Time                   `json:"cancelled_at,omitempty"`
	CancelReason        string                       `json:"cancel_reason,omitempty"`
	CreatedAt           time.Time                    `json:"created_at"`
	UpdatedAt           time.Time                    `json:"updated_at"`
	Version             int                          `json:"version" example:"1"`
}

// PurchaseReturnListResponse represents a purchase return in list responses
// @Description Purchase return list item response
type PurchaseReturnListResponse struct {
	ID                  string     `json:"id" example:"550e8400-e29b-41d4-a716-446655440010"`
	ReturnNumber        string     `json:"return_number" example:"PR-2026-00001"`
	PurchaseOrderID     string     `json:"purchase_order_id" example:"550e8400-e29b-41d4-a716-446655440020"`
	PurchaseOrderNumber string     `json:"purchase_order_number" example:"PO-2026-00001"`
	SupplierID          string     `json:"supplier_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	SupplierName        string     `json:"supplier_name" example:"优质供应商"`
	WarehouseID         *string    `json:"warehouse_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
	ItemCount           int        `json:"item_count" example:"2"`
	TotalRefund         float64    `json:"total_refund" example:"799.92"`
	Status              string     `json:"status" example:"pending"`
	SubmittedAt         *time.Time `json:"submitted_at,omitempty"`
	ApprovedAt          *time.Time `json:"approved_at,omitempty"`
	ShippedAt           *time.Time `json:"shipped_at,omitempty"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// PurchaseReturnItemResponse represents a return item in API responses
// @Description Purchase return item response
type PurchaseReturnItemResponse struct {
	ID                  string     `json:"id" example:"550e8400-e29b-41d4-a716-446655440030"`
	PurchaseOrderItemID string     `json:"purchase_order_item_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	ProductID           string     `json:"product_id" example:"550e8400-e29b-41d4-a716-446655440003"`
	ProductName         string     `json:"product_name" example:"测试商品"`
	ProductCode         string     `json:"product_code" example:"SKU-001"`
	OriginalQuantity    float64    `json:"original_quantity" example:"10"`
	ReturnQuantity      float64    `json:"return_quantity" example:"5"`
	UnitCost            float64    `json:"unit_cost" example:"99.99"`
	RefundAmount        float64    `json:"refund_amount" example:"499.95"`
	Unit                string     `json:"unit" example:"pcs"`
	Reason              string     `json:"reason,omitempty" example:"商品损坏"`
	ConditionOnReturn   string     `json:"condition_on_return,omitempty" example:"defective"`
	BatchNumber         string     `json:"batch_number,omitempty" example:"BATCH-001"`
	ShippedQuantity     float64    `json:"shipped_quantity" example:"5"`
	ShippedAt           *time.Time `json:"shipped_at,omitempty"`
	SupplierReceivedQty float64    `json:"supplier_received_qty" example:"5"`
	SupplierReceivedAt  *time.Time `json:"supplier_received_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// PurchaseReturnStatusSummaryResponse represents return count summary by status
// @Description Return status summary response
type PurchaseReturnStatusSummaryResponse struct {
	Draft           int64 `json:"draft" example:"3"`
	Pending         int64 `json:"pending" example:"5"`
	Approved        int64 `json:"approved" example:"2"`
	Rejected        int64 `json:"rejected" example:"1"`
	Shipped         int64 `json:"shipped" example:"3"`
	Completed       int64 `json:"completed" example:"50"`
	Cancelled       int64 `json:"cancelled" example:"2"`
	Total           int64 `json:"total" example:"66"`
	PendingApproval int64 `json:"pending_approval" example:"5"`
	PendingShipment int64 `json:"pending_shipment" example:"2"`
}


// Create godoc
// @ID           createPurchaseReturn
// @Summary      Create a new purchase return
// @Description  Create a new purchase return from an existing purchase order
// @Tags         purchase-returns
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        request body CreatePurchaseReturnRequest true "Purchase return creation request"
// @Success      201 {object} APIResponse[PurchaseReturnResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      422 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-returns [post]
func (h *PurchaseReturnHandler) Create(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req CreatePurchaseReturnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Convert to application DTO
	purchaseOrderID, err := uuid.Parse(req.PurchaseOrderID)
	if err != nil {
		h.BadRequest(c, "Invalid purchase order ID format")
		return
	}

	// Get user ID from JWT context (optional, for data scope)
	userID, _ := getUserID(c)

	appReq := tradeapp.CreatePurchaseReturnRequest{
		PurchaseOrderID: purchaseOrderID,
		Reason:          req.Reason,
		Remark:          req.Remark,
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
		orderItemID, err := uuid.Parse(item.PurchaseOrderItemID)
		if err != nil {
			h.BadRequest(c, "Invalid purchase order item ID format")
			return
		}
		appReq.Items = append(appReq.Items, tradeapp.CreatePurchaseReturnItemInput{
			PurchaseOrderItemID: orderItemID,
			ReturnQuantity:      decimal.NewFromFloat(item.ReturnQuantity),
			Reason:              item.Reason,
			ConditionOnReturn:   item.ConditionOnReturn,
			BatchNumber:         item.BatchNumber,
		})
	}

	pr, err := h.returnService.Create(c.Request.Context(), tenantID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Created(c, toPurchaseReturnResponse(pr))
}


// GetByID godoc
// @ID           getPurchaseReturnById
// @Summary      Get purchase return by ID
// @Description  Retrieve a purchase return by its ID
// @Tags         purchase-returns
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Purchase Return ID" format(uuid)
// @Success      200 {object} APIResponse[PurchaseReturnResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-returns/{id} [get]
func (h *PurchaseReturnHandler) GetByID(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	returnID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid return ID format")
		return
	}

	pr, err := h.returnService.GetByID(c.Request.Context(), tenantID, returnID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toPurchaseReturnResponse(pr))
}


// GetByReturnNumber godoc
// @ID           getPurchaseReturnByReturnNumber
// @Summary      Get purchase return by return number
// @Description  Retrieve a purchase return by its return number
// @Tags         purchase-returns
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        return_number path string true "Return Number" example:"PR-2026-00001"
// @Success      200 {object} APIResponse[PurchaseReturnResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-returns/number/{return_number} [get]
func (h *PurchaseReturnHandler) GetByReturnNumber(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	returnNumber := c.Param("return_number")
	if returnNumber == "" {
		h.BadRequest(c, "Return number is required")
		return
	}

	pr, err := h.returnService.GetByReturnNumber(c.Request.Context(), tenantID, returnNumber)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toPurchaseReturnResponse(pr))
}


// List godoc
// @ID           listPurchaseReturns
// @Summary      List purchase returns
// @Description  Retrieve a paginated list of purchase returns with optional filtering
// @Tags         purchase-returns
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        search query string false "Search term (return number, supplier name, order number)"
// @Param        supplier_id query string false "Supplier ID" format(uuid)
// @Param        purchase_order_id query string false "Purchase Order ID" format(uuid)
// @Param        warehouse_id query string false "Warehouse ID" format(uuid)
// @Param        status query string false "Return status" Enums(DRAFT, PENDING, APPROVED, REJECTED, SHIPPED, COMPLETED, CANCELLED)
// @Param        statuses query []string false "Multiple return statuses"
// @Param        start_date query string false "Start date (ISO 8601)" format(date-time)
// @Param        end_date query string false "End date (ISO 8601)" format(date-time)
// @Param        min_amount query number false "Minimum refund amount"
// @Param        max_amount query number false "Maximum refund amount"
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20) maximum(100)
// @Param        order_by query string false "Order by field" default(created_at)
// @Param        order_dir query string false "Order direction" Enums(asc, desc) default(desc)
// @Success      200 {object} APIResponse[[]PurchaseReturnListResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-returns [get]
func (h *PurchaseReturnHandler) List(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var filter tradeapp.PurchaseReturnListFilter
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

	returns, total, err := h.returnService.List(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.SuccessWithMeta(c, toPurchaseReturnListResponses(returns), total, filter.Page, filter.PageSize)
}


// Update godoc
// @ID           updatePurchaseReturn
// @Summary      Update a purchase return
// @Description  Update a purchase return (only allowed in DRAFT status)
// @Tags         purchase-returns
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Purchase Return ID" format(uuid)
// @Param        request body UpdatePurchaseReturnRequest true "Purchase return update request"
// @Success      200 {object} APIResponse[PurchaseReturnResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      422 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-returns/{id} [put]
func (h *PurchaseReturnHandler) Update(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	returnID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid return ID format")
		return
	}

	var req UpdatePurchaseReturnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Convert to application DTO
	appReq := tradeapp.UpdatePurchaseReturnRequest{
		Reason: req.Reason,
		Remark: req.Remark,
	}

	// Convert warehouse ID
	if req.WarehouseID != nil {
		if *req.WarehouseID == "" {
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

	pr, err := h.returnService.Update(c.Request.Context(), tenantID, returnID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toPurchaseReturnResponse(pr))
}


// Delete godoc
// @ID           deletePurchaseReturn
// @Summary      Delete a purchase return
// @Description  Delete a purchase return (only allowed in DRAFT status)
// @Tags         purchase-returns
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Purchase Return ID" format(uuid)
// @Success      204
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      422 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-returns/{id} [delete]
func (h *PurchaseReturnHandler) Delete(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	returnID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid return ID format")
		return
	}

	err = h.returnService.Delete(c.Request.Context(), tenantID, returnID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.NoContent(c)
}


// AddItem godoc
// @ID           addPurchaseReturnItem
// @Summary      Add item to purchase return
// @Description  Add a new item to a purchase return (only allowed in DRAFT status)
// @Tags         purchase-returns
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Purchase Return ID" format(uuid)
// @Param        request body AddPurchaseReturnItemRequest true "Return item to add"
// @Success      200 {object} APIResponse[PurchaseReturnResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      422 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-returns/{id}/items [post]
func (h *PurchaseReturnHandler) AddItem(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	returnID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid return ID format")
		return
	}

	var req AddPurchaseReturnItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	orderItemID, err := uuid.Parse(req.PurchaseOrderItemID)
	if err != nil {
		h.BadRequest(c, "Invalid purchase order item ID format")
		return
	}

	appReq := tradeapp.AddPurchaseReturnItemRequest{
		PurchaseOrderItemID: orderItemID,
		ReturnQuantity:      decimal.NewFromFloat(req.ReturnQuantity),
		Reason:              req.Reason,
		ConditionOnReturn:   req.ConditionOnReturn,
		BatchNumber:         req.BatchNumber,
	}

	pr, err := h.returnService.AddItem(c.Request.Context(), tenantID, returnID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toPurchaseReturnResponse(pr))
}


// UpdateItem godoc
// @ID           updatePurchaseReturnItem
// @Summary      Update return item
// @Description  Update an item in a purchase return (only allowed in DRAFT status)
// @Tags         purchase-returns
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Purchase Return ID" format(uuid)
// @Param        item_id path string true "Return Item ID" format(uuid)
// @Param        request body UpdatePurchaseReturnItemRequest true "Return item update request"
// @Success      200 {object} APIResponse[PurchaseReturnResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      422 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-returns/{id}/items/{item_id} [put]
func (h *PurchaseReturnHandler) UpdateItem(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	returnID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid return ID format")
		return
	}

	itemID, err := uuid.Parse(c.Param("item_id"))
	if err != nil {
		h.BadRequest(c, "Invalid item ID format")
		return
	}

	var req UpdatePurchaseReturnItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	appReq := tradeapp.UpdatePurchaseReturnItemRequest{
		Reason:            req.Reason,
		ConditionOnReturn: req.ConditionOnReturn,
		BatchNumber:       req.BatchNumber,
	}

	if req.ReturnQuantity != nil {
		d := decimal.NewFromFloat(*req.ReturnQuantity)
		appReq.ReturnQuantity = &d
	}

	pr, err := h.returnService.UpdateItem(c.Request.Context(), tenantID, returnID, itemID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toPurchaseReturnResponse(pr))
}


// RemoveItem godoc
// @ID           removePurchaseReturnItem
// @Summary      Remove item from purchase return
// @Description  Remove an item from a purchase return (only allowed in DRAFT status)
// @Tags         purchase-returns
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Purchase Return ID" format(uuid)
// @Param        item_id path string true "Return Item ID" format(uuid)
// @Success      200 {object} APIResponse[PurchaseReturnResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      422 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-returns/{id}/items/{item_id} [delete]
func (h *PurchaseReturnHandler) RemoveItem(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	returnID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid return ID format")
		return
	}

	itemID, err := uuid.Parse(c.Param("item_id"))
	if err != nil {
		h.BadRequest(c, "Invalid item ID format")
		return
	}

	pr, err := h.returnService.RemoveItem(c.Request.Context(), tenantID, returnID, itemID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toPurchaseReturnResponse(pr))
}


// Submit godoc
// @ID           submitPurchaseReturn
// @Summary      Submit a purchase return for approval
// @Description  Submit a purchase return for approval (transitions from DRAFT to PENDING)
// @Tags         purchase-returns
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Purchase Return ID" format(uuid)
// @Success      200 {object} APIResponse[PurchaseReturnResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      422 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-returns/{id}/submit [post]
func (h *PurchaseReturnHandler) Submit(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	returnID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid return ID format")
		return
	}

	pr, err := h.returnService.Submit(c.Request.Context(), tenantID, returnID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toPurchaseReturnResponse(pr))
}


// Approve godoc
// @ID           approvePurchaseReturn
// @Summary      Approve a purchase return
// @Description  Approve a purchase return (transitions from PENDING to APPROVED)
// @Tags         purchase-returns
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Purchase Return ID" format(uuid)
// @Param        request body ApprovePurchaseReturnRequest false "Approval request"
// @Success      200 {object} APIResponse[PurchaseReturnResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      422 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-returns/{id}/approve [post]
func (h *PurchaseReturnHandler) Approve(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	returnID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid return ID format")
		return
	}

	// Get user ID from context (should be set by auth middleware)
	userID, err := getUserID(c)
	if err != nil {
		h.Unauthorized(c, "User not authenticated")
		return
	}

	var req ApprovePurchaseReturnRequest
	// Allow empty body
	_ = c.ShouldBindJSON(&req)

	appReq := tradeapp.ApprovePurchaseReturnRequest{
		Note: req.Note,
	}

	pr, err := h.returnService.Approve(c.Request.Context(), tenantID, returnID, userID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toPurchaseReturnResponse(pr))
}


// Reject godoc
// @ID           rejectPurchaseReturn
// @Summary      Reject a purchase return
// @Description  Reject a purchase return (transitions from PENDING to REJECTED)
// @Tags         purchase-returns
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Purchase Return ID" format(uuid)
// @Param        request body RejectPurchaseReturnRequest true "Rejection request"
// @Success      200 {object} APIResponse[PurchaseReturnResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      422 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-returns/{id}/reject [post]
func (h *PurchaseReturnHandler) Reject(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	returnID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid return ID format")
		return
	}

	// Get user ID from context (should be set by auth middleware)
	userID, err := getUserID(c)
	if err != nil {
		h.Unauthorized(c, "User not authenticated")
		return
	}

	var req RejectPurchaseReturnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	appReq := tradeapp.RejectPurchaseReturnRequest{
		Reason: req.Reason,
	}

	pr, err := h.returnService.Reject(c.Request.Context(), tenantID, returnID, userID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toPurchaseReturnResponse(pr))
}


// Ship godoc
// @ID           shipPurchaseReturn
// @Summary      Ship a purchase return
// @Description  Mark a purchase return as shipped back to supplier (transitions from APPROVED to SHIPPED, triggers inventory deduction)
// @Tags         purchase-returns
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Purchase Return ID" format(uuid)
// @Param        request body ShipPurchaseReturnRequest false "Ship return request"
// @Success      200 {object} APIResponse[PurchaseReturnResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      422 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-returns/{id}/ship [post]
func (h *PurchaseReturnHandler) Ship(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	returnID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid return ID format")
		return
	}

	// Get user ID from context
	userID := uuid.Nil
	userIDStr := middleware.GetJWTUserID(c)
	if userIDStr != "" {
		userID, _ = uuid.Parse(userIDStr)
	}

	var req ShipPurchaseReturnRequest
	// Allow empty body
	_ = c.ShouldBindJSON(&req)

	appReq := tradeapp.ShipPurchaseReturnRequest{
		TrackingNumber: req.TrackingNumber,
		Note:           req.Note,
	}

	pr, err := h.returnService.Ship(c.Request.Context(), tenantID, returnID, userID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toPurchaseReturnResponse(pr))
}


// Complete godoc
// @ID           completePurchaseReturn
// @Summary      Complete a purchase return
// @Description  Mark a purchase return as completed after supplier confirms receipt (transitions from SHIPPED to COMPLETED)
// @Tags         purchase-returns
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Purchase Return ID" format(uuid)
// @Success      200 {object} APIResponse[PurchaseReturnResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      422 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-returns/{id}/complete [post]
func (h *PurchaseReturnHandler) Complete(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	returnID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid return ID format")
		return
	}

	appReq := tradeapp.CompletePurchaseReturnRequest{}

	pr, err := h.returnService.Complete(c.Request.Context(), tenantID, returnID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toPurchaseReturnResponse(pr))
}


// Cancel godoc
// @ID           cancelPurchaseReturn
// @Summary      Cancel a purchase return
// @Description  Cancel a purchase return (from DRAFT, PENDING, or APPROVED status - before shipping)
// @Tags         purchase-returns
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Purchase Return ID" format(uuid)
// @Param        request body CancelPurchaseReturnRequest true "Cancel return request"
// @Success      200 {object} APIResponse[PurchaseReturnResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      422 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-returns/{id}/cancel [post]
func (h *PurchaseReturnHandler) Cancel(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	returnID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid return ID format")
		return
	}

	var req CancelPurchaseReturnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	appReq := tradeapp.CancelPurchaseReturnRequest{
		Reason: req.Reason,
	}

	pr, err := h.returnService.Cancel(c.Request.Context(), tenantID, returnID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toPurchaseReturnResponse(pr))
}


// GetStatusSummary godoc
// @ID           getPurchaseReturnStatusSummary
// @Summary      Get return status summary
// @Description  Get count of returns by status for dashboard
// @Tags         purchase-returns
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Success      200 {object} APIResponse[PurchaseReturnStatusSummaryResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /trade/purchase-returns/stats/summary [get]
func (h *PurchaseReturnHandler) GetStatusSummary(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	summary, err := h.returnService.GetStatusSummary(c.Request.Context(), tenantID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, PurchaseReturnStatusSummaryResponse{
		Draft:           summary.Draft,
		Pending:         summary.Pending,
		Approved:        summary.Approved,
		Rejected:        summary.Rejected,
		Shipped:         summary.Shipped,
		Completed:       summary.Completed,
		Cancelled:       summary.Cancelled,
		Total:           summary.Total,
		PendingApproval: summary.PendingApproval,
		PendingShipment: summary.PendingShipment,
	})
}

// toPurchaseReturnResponse converts application response to handler response
func toPurchaseReturnResponse(pr *tradeapp.PurchaseReturnResponse) PurchaseReturnResponse {
	items := make([]PurchaseReturnItemResponse, len(pr.Items))
	for i, item := range pr.Items {
		items[i] = PurchaseReturnItemResponse{
			ID:                  item.ID.String(),
			PurchaseOrderItemID: item.PurchaseOrderItemID.String(),
			ProductID:           item.ProductID.String(),
			ProductName:         item.ProductName,
			ProductCode:         item.ProductCode,
			OriginalQuantity:    item.OriginalQuantity.InexactFloat64(),
			ReturnQuantity:      item.ReturnQuantity.InexactFloat64(),
			UnitCost:            item.UnitCost.InexactFloat64(),
			RefundAmount:        item.RefundAmount.InexactFloat64(),
			Unit:                item.Unit,
			Reason:              item.Reason,
			ConditionOnReturn:   item.ConditionOnReturn,
			BatchNumber:         item.BatchNumber,
			ShippedQuantity:     item.ShippedQuantity.InexactFloat64(),
			ShippedAt:           item.ShippedAt,
			SupplierReceivedQty: item.SupplierReceivedQty.InexactFloat64(),
			SupplierReceivedAt:  item.SupplierReceivedAt,
			CreatedAt:           item.CreatedAt,
			UpdatedAt:           item.UpdatedAt,
		}
	}

	resp := PurchaseReturnResponse{
		ID:                  pr.ID.String(),
		TenantID:            pr.TenantID.String(),
		ReturnNumber:        pr.ReturnNumber,
		PurchaseOrderID:     pr.PurchaseOrderID.String(),
		PurchaseOrderNumber: pr.PurchaseOrderNumber,
		SupplierID:          pr.SupplierID.String(),
		SupplierName:        pr.SupplierName,
		Items:               items,
		ItemCount:           pr.ItemCount,
		TotalQuantity:       pr.TotalQuantity.InexactFloat64(),
		TotalRefund:         pr.TotalRefund.InexactFloat64(),
		Status:              pr.Status,
		Reason:              pr.Reason,
		Remark:              pr.Remark,
		SubmittedAt:         pr.SubmittedAt,
		ApprovedAt:          pr.ApprovedAt,
		ApprovalNote:        pr.ApprovalNote,
		RejectedAt:          pr.RejectedAt,
		RejectionReason:     pr.RejectionReason,
		ShippedAt:           pr.ShippedAt,
		ShippingNote:        pr.ShippingNote,
		TrackingNumber:      pr.TrackingNumber,
		CompletedAt:         pr.CompletedAt,
		CancelledAt:         pr.CancelledAt,
		CancelReason:        pr.CancelReason,
		CreatedAt:           pr.CreatedAt,
		UpdatedAt:           pr.UpdatedAt,
		Version:             pr.Version,
	}

	if pr.WarehouseID != nil {
		warehouseID := pr.WarehouseID.String()
		resp.WarehouseID = &warehouseID
	}

	if pr.ApprovedBy != nil {
		approvedBy := pr.ApprovedBy.String()
		resp.ApprovedBy = &approvedBy
	}

	if pr.RejectedBy != nil {
		rejectedBy := pr.RejectedBy.String()
		resp.RejectedBy = &rejectedBy
	}

	if pr.ShippedBy != nil {
		shippedBy := pr.ShippedBy.String()
		resp.ShippedBy = &shippedBy
	}

	return resp
}

// toPurchaseReturnListResponses converts application list responses to handler responses
func toPurchaseReturnListResponses(returns []tradeapp.PurchaseReturnListItemResponse) []PurchaseReturnListResponse {
	responses := make([]PurchaseReturnListResponse, len(returns))
	for i, pr := range returns {
		resp := PurchaseReturnListResponse{
			ID:                  pr.ID.String(),
			ReturnNumber:        pr.ReturnNumber,
			PurchaseOrderID:     pr.PurchaseOrderID.String(),
			PurchaseOrderNumber: pr.PurchaseOrderNumber,
			SupplierID:          pr.SupplierID.String(),
			SupplierName:        pr.SupplierName,
			ItemCount:           pr.ItemCount,
			TotalRefund:         pr.TotalRefund.InexactFloat64(),
			Status:              pr.Status,
			SubmittedAt:         pr.SubmittedAt,
			ApprovedAt:          pr.ApprovedAt,
			ShippedAt:           pr.ShippedAt,
			CompletedAt:         pr.CompletedAt,
			CreatedAt:           pr.CreatedAt,
			UpdatedAt:           pr.UpdatedAt,
		}

		if pr.WarehouseID != nil {
			warehouseID := pr.WarehouseID.String()
			resp.WarehouseID = &warehouseID
		}

		responses[i] = resp
	}
	return responses
}

// Helper function to suppress unused import warning
var _ = dto.Response{}
var _ = trade.PurchaseReturnStatusDraft
