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

// SalesReturnHandler handles sales return-related API endpoints
type SalesReturnHandler struct {
	BaseHandler
	returnService *tradeapp.SalesReturnService
}

// NewSalesReturnHandler creates a new SalesReturnHandler
func NewSalesReturnHandler(returnService *tradeapp.SalesReturnService) *SalesReturnHandler {
	return &SalesReturnHandler{
		returnService: returnService,
	}
}

// CreateSalesReturnRequest represents a request to create a new sales return
//
//	@Description	Request body for creating a new sales return
type CreateSalesReturnRequest struct {
	SalesOrderID string                       `json:"sales_order_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	WarehouseID  *string                      `json:"warehouse_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Items        []CreateSalesReturnItemInput `json:"items" binding:"required,min=1"`
	Reason       string                       `json:"reason" example:"商品质量问题"`
	Remark       string                       `json:"remark" example:"备注信息"`
}

// CreateSalesReturnItemInput represents an item in the create return request
//
//	@Description	Return item for creation
type CreateSalesReturnItemInput struct {
	SalesOrderItemID  string  `json:"sales_order_item_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440002"`
	ReturnQuantity    float64 `json:"return_quantity" binding:"required,gt=0" example:"5"`
	Reason            string  `json:"reason" example:"商品损坏"`
	ConditionOnReturn string  `json:"condition_on_return" example:"damaged"`
}

// UpdateSalesReturnRequest represents a request to update a sales return
//
//	@Description	Request body for updating a sales return (draft only)
type UpdateSalesReturnRequest struct {
	WarehouseID *string `json:"warehouse_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Reason      *string `json:"reason" example:"更新退货原因"`
	Remark      *string `json:"remark" example:"更新备注"`
}

// AddReturnItemRequest represents a request to add an item to a return
//
//	@Description	Request body for adding an item to a return
type AddReturnItemRequest struct {
	SalesOrderItemID  string  `json:"sales_order_item_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440002"`
	ReturnQuantity    float64 `json:"return_quantity" binding:"required,gt=0" example:"3"`
	Reason            string  `json:"reason" example:"商品有瑕疵"`
	ConditionOnReturn string  `json:"condition_on_return" example:"defective"`
}

// UpdateReturnItemRequest represents a request to update a return item
//
//	@Description	Request body for updating a return item
type UpdateReturnItemRequest struct {
	ReturnQuantity    *float64 `json:"return_quantity" example:"4"`
	Reason            *string  `json:"reason" example:"更新原因"`
	ConditionOnReturn *string  `json:"condition_on_return" example:"wrong_item"`
}

// ApproveReturnRequest represents a request to approve a return
//
//	@Description	Request body for approving a return
type ApproveReturnRequest struct {
	Note string `json:"note" example:"审批通过，同意退货"`
}

// RejectReturnRequest represents a request to reject a return
//
//	@Description	Request body for rejecting a return
type RejectReturnRequest struct {
	Reason string `json:"reason" binding:"required,min=1,max=500" example:"不符合退货条件"`
}

// CancelReturnRequest represents a request to cancel a return
//
//	@Description	Request body for cancelling a return
type CancelReturnRequest struct {
	Reason string `json:"reason" binding:"required,min=1,max=500" example:"客户撤销退货申请"`
}

// CompleteReturnRequest represents a request to complete a return
//
//	@Description	Request body for completing a return
type CompleteReturnRequest struct {
	WarehouseID *string `json:"warehouse_id" example:"550e8400-e29b-41d4-a716-446655440001"`
}

// ReceiveReturnRequest represents a request to receive returned goods
//
//	@Description	Request body for receiving returned goods
type ReceiveReturnRequest struct {
	WarehouseID *string `json:"warehouse_id" example:"550e8400-e29b-41d4-a716-446655440001"`
}

// SalesReturnResponse represents a sales return in API responses
//
//	@Description	Sales return response
type SalesReturnResponse struct {
	ID               string                    `json:"id" example:"550e8400-e29b-41d4-a716-446655440010"`
	TenantID         string                    `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ReturnNumber     string                    `json:"return_number" example:"SR-2026-00001"`
	SalesOrderID     string                    `json:"sales_order_id" example:"550e8400-e29b-41d4-a716-446655440020"`
	SalesOrderNumber string                    `json:"sales_order_number" example:"SO-2026-00001"`
	CustomerID       string                    `json:"customer_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	CustomerName     string                    `json:"customer_name" example:"张三"`
	WarehouseID      *string                   `json:"warehouse_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
	Items            []SalesReturnItemResponse `json:"items"`
	ItemCount        int                       `json:"item_count" example:"2"`
	TotalQuantity    float64                   `json:"total_quantity" example:"8"`
	TotalRefund      float64                   `json:"total_refund" example:"799.92"`
	Status           string                    `json:"status" example:"draft"`
	Reason           string                    `json:"reason,omitempty" example:"商品质量问题"`
	Remark           string                    `json:"remark,omitempty" example:"备注信息"`
	SubmittedAt      *time.Time                `json:"submitted_at,omitempty"`
	ApprovedAt       *time.Time                `json:"approved_at,omitempty"`
	ApprovedBy       *string                   `json:"approved_by,omitempty"`
	ApprovalNote     string                    `json:"approval_note,omitempty"`
	RejectedAt       *time.Time                `json:"rejected_at,omitempty"`
	RejectedBy       *string                   `json:"rejected_by,omitempty"`
	RejectionReason  string                    `json:"rejection_reason,omitempty"`
	ReceivedAt       *time.Time                `json:"received_at,omitempty"`
	CompletedAt      *time.Time                `json:"completed_at,omitempty"`
	CancelledAt      *time.Time                `json:"cancelled_at,omitempty"`
	CancelReason     string                    `json:"cancel_reason,omitempty"`
	CreatedAt        time.Time                 `json:"created_at"`
	UpdatedAt        time.Time                 `json:"updated_at"`
	Version          int                       `json:"version" example:"1"`
}

// SalesReturnListResponse represents a sales return in list responses
//
//	@Description	Sales return list item response
type SalesReturnListResponse struct {
	ID               string     `json:"id" example:"550e8400-e29b-41d4-a716-446655440010"`
	ReturnNumber     string     `json:"return_number" example:"SR-2026-00001"`
	SalesOrderID     string     `json:"sales_order_id" example:"550e8400-e29b-41d4-a716-446655440020"`
	SalesOrderNumber string     `json:"sales_order_number" example:"SO-2026-00001"`
	CustomerID       string     `json:"customer_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	CustomerName     string     `json:"customer_name" example:"张三"`
	WarehouseID      *string    `json:"warehouse_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
	ItemCount        int        `json:"item_count" example:"2"`
	TotalRefund      float64    `json:"total_refund" example:"799.92"`
	Status           string     `json:"status" example:"pending"`
	SubmittedAt      *time.Time `json:"submitted_at,omitempty"`
	ApprovedAt       *time.Time `json:"approved_at,omitempty"`
	ReceivedAt       *time.Time `json:"received_at,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// SalesReturnItemResponse represents a return item in API responses
//
//	@Description	Sales return item response
type SalesReturnItemResponse struct {
	ID                string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440030"`
	SalesOrderItemID  string    `json:"sales_order_item_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	ProductID         string    `json:"product_id" example:"550e8400-e29b-41d4-a716-446655440003"`
	ProductName       string    `json:"product_name" example:"测试商品"`
	ProductCode       string    `json:"product_code" example:"SKU-001"`
	OriginalQuantity  float64   `json:"original_quantity" example:"10"`
	ReturnQuantity    float64   `json:"return_quantity" example:"5"`
	UnitPrice         float64   `json:"unit_price" example:"99.99"`
	RefundAmount      float64   `json:"refund_amount" example:"499.95"`
	Unit              string    `json:"unit" example:"pcs"`
	Reason            string    `json:"reason,omitempty" example:"商品损坏"`
	ConditionOnReturn string    `json:"condition_on_return,omitempty" example:"damaged"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ReturnStatusSummaryResponse represents return count summary by status
//
//	@Description	Return status summary response
type ReturnStatusSummaryResponse struct {
	Draft           int64 `json:"draft" example:"3"`
	Pending         int64 `json:"pending" example:"5"`
	Approved        int64 `json:"approved" example:"2"`
	Receiving       int64 `json:"receiving" example:"1"`
	Rejected        int64 `json:"rejected" example:"1"`
	Completed       int64 `json:"completed" example:"50"`
	Cancelled       int64 `json:"cancelled" example:"2"`
	Total           int64 `json:"total" example:"63"`
	PendingApproval int64 `json:"pending_approval" example:"5"`
}

// Create godoc
//
//	@ID				createSalesReturn
//	@Summary		Create a new sales return
//	@Description	Create a new sales return from an existing sales order
//	@Tags			sales-returns
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string						false	"Tenant ID (optional for dev)"
//	@Param			request		body		CreateSalesReturnRequest	true	"Sales return creation request"
//	@Success		201			{object}	APIResponse[SalesReturnResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		422			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/trade/sales-returns [post]
func (h *SalesReturnHandler) Create(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req CreateSalesReturnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Convert to application DTO
	salesOrderID, err := uuid.Parse(req.SalesOrderID)
	if err != nil {
		h.BadRequest(c, "Invalid sales order ID format")
		return
	}

	// Get user ID from JWT context (optional, for data scope)
	userID, _ := getUserID(c)

	appReq := tradeapp.CreateSalesReturnRequest{
		SalesOrderID: salesOrderID,
		Reason:       req.Reason,
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
		orderItemID, err := uuid.Parse(item.SalesOrderItemID)
		if err != nil {
			h.BadRequest(c, "Invalid sales order item ID format")
			return
		}
		appReq.Items = append(appReq.Items, tradeapp.CreateSalesReturnItemInput{
			SalesOrderItemID:  orderItemID,
			ReturnQuantity:    decimal.NewFromFloat(item.ReturnQuantity),
			Reason:            item.Reason,
			ConditionOnReturn: item.ConditionOnReturn,
		})
	}

	sr, err := h.returnService.Create(c.Request.Context(), tenantID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Created(c, toSalesReturnResponse(sr))
}

// GetByID godoc
//
//	@ID				getSalesReturnById
//	@Summary		Get sales return by ID
//	@Description	Retrieve a sales return by its ID
//	@Tags			sales-returns
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			id			path		string	true	"Sales Return ID"	format(uuid)
//	@Success		200			{object}	APIResponse[SalesReturnResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/trade/sales-returns/{id} [get]
func (h *SalesReturnHandler) GetByID(c *gin.Context) {
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

	sr, err := h.returnService.GetByID(c.Request.Context(), tenantID, returnID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toSalesReturnResponse(sr))
}

// GetByReturnNumber godoc
//
//	@ID				getSalesReturnByReturnNumber
//	@Summary		Get sales return by return number
//	@Description	Retrieve a sales return by its return number
//	@Tags			sales-returns
//	@Produce		json
//	@Param			X-Tenant-ID		header		string	false	"Tenant ID (optional for dev)"
//	@Param			return_number	path		string	true	"Return Number"	example:"SR-2026-00001"
//	@Success		200				{object}	APIResponse[SalesReturnResponse]
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		404				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/trade/sales-returns/number/{return_number} [get]
func (h *SalesReturnHandler) GetByReturnNumber(c *gin.Context) {
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

	sr, err := h.returnService.GetByReturnNumber(c.Request.Context(), tenantID, returnNumber)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toSalesReturnResponse(sr))
}

// List godoc
//
//	@ID				listSalesReturns
//	@Summary		List sales returns
//	@Description	Retrieve a paginated list of sales returns with optional filtering
//	@Tags			sales-returns
//	@Produce		json
//	@Param			X-Tenant-ID		header		string		false	"Tenant ID (optional for dev)"
//	@Param			search			query		string		false	"Search term (return number, customer name, order number)"
//	@Param			customer_id		query		string		false	"Customer ID"		format(uuid)
//	@Param			sales_order_id	query		string		false	"Sales Order ID"	format(uuid)
//	@Param			warehouse_id	query		string		false	"Warehouse ID"		format(uuid)
//	@Param			status			query		string		false	"Return status"		Enums(DRAFT, PENDING, APPROVED, REJECTED, COMPLETED, CANCELLED)
//	@Param			statuses		query		[]string	false	"Multiple return statuses"
//	@Param			start_date		query		string		false	"Start date (ISO 8601)"	format(date-time)
//	@Param			end_date		query		string		false	"End date (ISO 8601)"	format(date-time)
//	@Param			min_amount		query		number		false	"Minimum refund amount"
//	@Param			max_amount		query		number		false	"Maximum refund amount"
//	@Param			page			query		int			false	"Page number"		default(1)
//	@Param			page_size		query		int			false	"Page size"			default(20)	maximum(100)
//	@Param			order_by		query		string		false	"Order by field"	default(created_at)
//	@Param			order_dir		query		string		false	"Order direction"	Enums(asc, desc)	default(desc)
//	@Success		200				{object}	APIResponse[[]SalesReturnListResponse]
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/trade/sales-returns [get]
func (h *SalesReturnHandler) List(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var filter tradeapp.SalesReturnListFilter
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

	h.SuccessWithMeta(c, toSalesReturnListResponses(returns), total, filter.Page, filter.PageSize)
}

// Update godoc
//
//	@ID				updateSalesReturn
//	@Summary		Update a sales return
//	@Description	Update a sales return (only allowed in DRAFT status)
//	@Tags			sales-returns
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string						false	"Tenant ID (optional for dev)"
//	@Param			id			path		string						true	"Sales Return ID"	format(uuid)
//	@Param			request		body		UpdateSalesReturnRequest	true	"Sales return update request"
//	@Success		200			{object}	APIResponse[SalesReturnResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		422			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/trade/sales-returns/{id} [put]
func (h *SalesReturnHandler) Update(c *gin.Context) {
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

	var req UpdateSalesReturnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Convert to application DTO
	appReq := tradeapp.UpdateSalesReturnRequest{
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

	sr, err := h.returnService.Update(c.Request.Context(), tenantID, returnID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toSalesReturnResponse(sr))
}

// Delete godoc
//
//	@ID				deleteSalesReturn
//	@Summary		Delete a sales return
//	@Description	Delete a sales return (only allowed in DRAFT status)
//	@Tags			sales-returns
//	@Produce		json
//	@Param			X-Tenant-ID	header	string	false	"Tenant ID (optional for dev)"
//	@Param			id			path	string	true	"Sales Return ID"	format(uuid)
//	@Success		204
//	@Failure		400	{object}	dto.ErrorResponse
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Failure		422	{object}	dto.ErrorResponse
//	@Failure		500	{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/trade/sales-returns/{id} [delete]
func (h *SalesReturnHandler) Delete(c *gin.Context) {
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
//
//	@ID				addSalesReturnItem
//	@Summary		Add item to sales return
//	@Description	Add a new item to a sales return (only allowed in DRAFT status)
//	@Tags			sales-returns
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string					false	"Tenant ID (optional for dev)"
//	@Param			id			path		string					true	"Sales Return ID"	format(uuid)
//	@Param			request		body		AddReturnItemRequest	true	"Return item to add"
//	@Success		200			{object}	APIResponse[SalesReturnResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		422			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/trade/sales-returns/{id}/items [post]
func (h *SalesReturnHandler) AddItem(c *gin.Context) {
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

	var req AddReturnItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	orderItemID, err := uuid.Parse(req.SalesOrderItemID)
	if err != nil {
		h.BadRequest(c, "Invalid sales order item ID format")
		return
	}

	appReq := tradeapp.AddReturnItemRequest{
		SalesOrderItemID:  orderItemID,
		ReturnQuantity:    decimal.NewFromFloat(req.ReturnQuantity),
		Reason:            req.Reason,
		ConditionOnReturn: req.ConditionOnReturn,
	}

	sr, err := h.returnService.AddItem(c.Request.Context(), tenantID, returnID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toSalesReturnResponse(sr))
}

// UpdateItem godoc
//
//	@ID				updateSalesReturnItem
//	@Summary		Update return item
//	@Description	Update an item in a sales return (only allowed in DRAFT status)
//	@Tags			sales-returns
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string					false	"Tenant ID (optional for dev)"
//	@Param			id			path		string					true	"Sales Return ID"	format(uuid)
//	@Param			item_id		path		string					true	"Return Item ID"	format(uuid)
//	@Param			request		body		UpdateReturnItemRequest	true	"Return item update request"
//	@Success		200			{object}	APIResponse[SalesReturnResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		422			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/trade/sales-returns/{id}/items/{item_id} [put]
func (h *SalesReturnHandler) UpdateItem(c *gin.Context) {
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

	var req UpdateReturnItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	appReq := tradeapp.UpdateReturnItemRequest{
		Reason:            req.Reason,
		ConditionOnReturn: req.ConditionOnReturn,
	}

	if req.ReturnQuantity != nil {
		d := decimal.NewFromFloat(*req.ReturnQuantity)
		appReq.ReturnQuantity = &d
	}

	sr, err := h.returnService.UpdateItem(c.Request.Context(), tenantID, returnID, itemID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toSalesReturnResponse(sr))
}

// RemoveItem godoc
//
//	@ID				removeSalesReturnItem
//	@Summary		Remove item from sales return
//	@Description	Remove an item from a sales return (only allowed in DRAFT status)
//	@Tags			sales-returns
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			id			path		string	true	"Sales Return ID"	format(uuid)
//	@Param			item_id		path		string	true	"Return Item ID"	format(uuid)
//	@Success		200			{object}	APIResponse[SalesReturnResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		422			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/trade/sales-returns/{id}/items/{item_id} [delete]
func (h *SalesReturnHandler) RemoveItem(c *gin.Context) {
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

	sr, err := h.returnService.RemoveItem(c.Request.Context(), tenantID, returnID, itemID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toSalesReturnResponse(sr))
}

// Submit godoc
//
//	@ID				submitSalesReturn
//	@Summary		Submit a sales return for approval
//	@Description	Submit a sales return for approval (transitions from DRAFT to PENDING)
//	@Tags			sales-returns
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			id			path		string	true	"Sales Return ID"	format(uuid)
//	@Success		200			{object}	APIResponse[SalesReturnResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		422			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/trade/sales-returns/{id}/submit [post]
func (h *SalesReturnHandler) Submit(c *gin.Context) {
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

	sr, err := h.returnService.Submit(c.Request.Context(), tenantID, returnID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toSalesReturnResponse(sr))
}

// Approve godoc
//
//	@ID				approveSalesReturn
//	@Summary		Approve a sales return
//	@Description	Approve a sales return (transitions from PENDING to APPROVED)
//	@Tags			sales-returns
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string					false	"Tenant ID (optional for dev)"
//	@Param			id			path		string					true	"Sales Return ID"	format(uuid)
//	@Param			request		body		ApproveReturnRequest	false	"Approval request"
//	@Success		200			{object}	APIResponse[SalesReturnResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		422			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/trade/sales-returns/{id}/approve [post]
func (h *SalesReturnHandler) Approve(c *gin.Context) {
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

	var req ApproveReturnRequest
	// Allow empty body
	_ = c.ShouldBindJSON(&req)

	appReq := tradeapp.ApproveReturnRequest{
		Note: req.Note,
	}

	sr, err := h.returnService.Approve(c.Request.Context(), tenantID, returnID, userID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toSalesReturnResponse(sr))
}

// Receive godoc
//
//	@ID				receiveSalesReturn
//	@Summary		Receive returned goods
//	@Description	Start receiving returned goods into warehouse (transitions from APPROVED to RECEIVING)
//	@Tags			sales-returns
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string					false	"Tenant ID (optional for dev)"
//	@Param			id			path		string					true	"Sales Return ID"	format(uuid)
//	@Param			request		body		ReceiveReturnRequest	false	"Receive return request"
//	@Success		200			{object}	APIResponse[SalesReturnResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		422			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/trade/sales-returns/{id}/receive [post]
func (h *SalesReturnHandler) Receive(c *gin.Context) {
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

	var req ReceiveReturnRequest
	// Allow empty body
	_ = c.ShouldBindJSON(&req)

	appReq := tradeapp.ReceiveReturnRequest{}

	if req.WarehouseID != nil && *req.WarehouseID != "" {
		warehouseID, err := uuid.Parse(*req.WarehouseID)
		if err != nil {
			h.BadRequest(c, "Invalid warehouse ID format")
			return
		}
		appReq.WarehouseID = &warehouseID
	}

	sr, err := h.returnService.Receive(c.Request.Context(), tenantID, returnID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toSalesReturnResponse(sr))
}

// Reject godoc
//
//	@ID				rejectSalesReturn
//	@Summary		Reject a sales return
//	@Description	Reject a sales return (transitions from PENDING to REJECTED)
//	@Tags			sales-returns
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string				false	"Tenant ID (optional for dev)"
//	@Param			id			path		string				true	"Sales Return ID"	format(uuid)
//	@Param			request		body		RejectReturnRequest	true	"Rejection request"
//	@Success		200			{object}	APIResponse[SalesReturnResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		422			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/trade/sales-returns/{id}/reject [post]
func (h *SalesReturnHandler) Reject(c *gin.Context) {
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

	var req RejectReturnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	appReq := tradeapp.RejectReturnRequest{
		Reason: req.Reason,
	}

	sr, err := h.returnService.Reject(c.Request.Context(), tenantID, returnID, userID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toSalesReturnResponse(sr))
}

// Complete godoc
//
//	@ID				completeSalesReturn
//	@Summary		Complete a sales return
//	@Description	Mark a sales return as completed (transitions from APPROVED to COMPLETED)
//	@Tags			sales-returns
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string					false	"Tenant ID (optional for dev)"
//	@Param			id			path		string					true	"Sales Return ID"	format(uuid)
//	@Param			request		body		CompleteReturnRequest	false	"Complete return request"
//	@Success		200			{object}	APIResponse[SalesReturnResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		422			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/trade/sales-returns/{id}/complete [post]
func (h *SalesReturnHandler) Complete(c *gin.Context) {
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

	var req CompleteReturnRequest
	// Allow empty body
	_ = c.ShouldBindJSON(&req)

	appReq := tradeapp.CompleteReturnRequest{}

	if req.WarehouseID != nil && *req.WarehouseID != "" {
		warehouseID, err := uuid.Parse(*req.WarehouseID)
		if err != nil {
			h.BadRequest(c, "Invalid warehouse ID format")
			return
		}
		appReq.WarehouseID = &warehouseID
	}

	sr, err := h.returnService.Complete(c.Request.Context(), tenantID, returnID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toSalesReturnResponse(sr))
}

// Cancel godoc
//
//	@ID				cancelSalesReturn
//	@Summary		Cancel a sales return
//	@Description	Cancel a sales return (from DRAFT, PENDING, or APPROVED status)
//	@Tags			sales-returns
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string				false	"Tenant ID (optional for dev)"
//	@Param			id			path		string				true	"Sales Return ID"	format(uuid)
//	@Param			request		body		CancelReturnRequest	true	"Cancel return request"
//	@Success		200			{object}	APIResponse[SalesReturnResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		422			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/trade/sales-returns/{id}/cancel [post]
func (h *SalesReturnHandler) Cancel(c *gin.Context) {
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

	var req CancelReturnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	appReq := tradeapp.CancelReturnRequest{
		Reason: req.Reason,
	}

	sr, err := h.returnService.Cancel(c.Request.Context(), tenantID, returnID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toSalesReturnResponse(sr))
}

// GetStatusSummary godoc
//
//	@ID				getSalesReturnStatusSummary
//	@Summary		Get return status summary
//	@Description	Get count of returns by status for dashboard
//	@Tags			sales-returns
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Success		200			{object}	APIResponse[ReturnStatusSummaryResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/trade/sales-returns/stats/summary [get]
func (h *SalesReturnHandler) GetStatusSummary(c *gin.Context) {
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

	h.Success(c, ReturnStatusSummaryResponse{
		Draft:           summary.Draft,
		Pending:         summary.Pending,
		Approved:        summary.Approved,
		Receiving:       summary.Receiving,
		Rejected:        summary.Rejected,
		Completed:       summary.Completed,
		Cancelled:       summary.Cancelled,
		Total:           summary.Total,
		PendingApproval: summary.PendingApproval,
	})
}

// toSalesReturnResponse converts application response to handler response
func toSalesReturnResponse(sr *tradeapp.SalesReturnResponse) SalesReturnResponse {
	items := make([]SalesReturnItemResponse, len(sr.Items))
	for i, item := range sr.Items {
		items[i] = SalesReturnItemResponse{
			ID:                item.ID.String(),
			SalesOrderItemID:  item.SalesOrderItemID.String(),
			ProductID:         item.ProductID.String(),
			ProductName:       item.ProductName,
			ProductCode:       item.ProductCode,
			OriginalQuantity:  item.OriginalQuantity.InexactFloat64(),
			ReturnQuantity:    item.ReturnQuantity.InexactFloat64(),
			UnitPrice:         item.UnitPrice.InexactFloat64(),
			RefundAmount:      item.RefundAmount.InexactFloat64(),
			Unit:              item.Unit,
			Reason:            item.Reason,
			ConditionOnReturn: item.ConditionOnReturn,
			CreatedAt:         item.CreatedAt,
			UpdatedAt:         item.UpdatedAt,
		}
	}

	resp := SalesReturnResponse{
		ID:               sr.ID.String(),
		TenantID:         sr.TenantID.String(),
		ReturnNumber:     sr.ReturnNumber,
		SalesOrderID:     sr.SalesOrderID.String(),
		SalesOrderNumber: sr.SalesOrderNumber,
		CustomerID:       sr.CustomerID.String(),
		CustomerName:     sr.CustomerName,
		Items:            items,
		ItemCount:        sr.ItemCount,
		TotalQuantity:    sr.TotalQuantity.InexactFloat64(),
		TotalRefund:      sr.TotalRefund.InexactFloat64(),
		Status:           sr.Status,
		Reason:           sr.Reason,
		Remark:           sr.Remark,
		SubmittedAt:      sr.SubmittedAt,
		ApprovedAt:       sr.ApprovedAt,
		ApprovalNote:     sr.ApprovalNote,
		RejectedAt:       sr.RejectedAt,
		RejectionReason:  sr.RejectionReason,
		ReceivedAt:       sr.ReceivedAt,
		CompletedAt:      sr.CompletedAt,
		CancelledAt:      sr.CancelledAt,
		CancelReason:     sr.CancelReason,
		CreatedAt:        sr.CreatedAt,
		UpdatedAt:        sr.UpdatedAt,
		Version:          sr.Version,
	}

	if sr.WarehouseID != nil {
		warehouseID := sr.WarehouseID.String()
		resp.WarehouseID = &warehouseID
	}

	if sr.ApprovedBy != nil {
		approvedBy := sr.ApprovedBy.String()
		resp.ApprovedBy = &approvedBy
	}

	if sr.RejectedBy != nil {
		rejectedBy := sr.RejectedBy.String()
		resp.RejectedBy = &rejectedBy
	}

	return resp
}

// toSalesReturnListResponses converts application list responses to handler responses
func toSalesReturnListResponses(returns []tradeapp.SalesReturnListItemResponse) []SalesReturnListResponse {
	responses := make([]SalesReturnListResponse, len(returns))
	for i, sr := range returns {
		resp := SalesReturnListResponse{
			ID:               sr.ID.String(),
			ReturnNumber:     sr.ReturnNumber,
			SalesOrderID:     sr.SalesOrderID.String(),
			SalesOrderNumber: sr.SalesOrderNumber,
			CustomerID:       sr.CustomerID.String(),
			CustomerName:     sr.CustomerName,
			ItemCount:        sr.ItemCount,
			TotalRefund:      sr.TotalRefund.InexactFloat64(),
			Status:           sr.Status,
			SubmittedAt:      sr.SubmittedAt,
			ApprovedAt:       sr.ApprovedAt,
			ReceivedAt:       sr.ReceivedAt,
			CompletedAt:      sr.CompletedAt,
			CreatedAt:        sr.CreatedAt,
			UpdatedAt:        sr.UpdatedAt,
		}

		if sr.WarehouseID != nil {
			warehouseID := sr.WarehouseID.String()
			resp.WarehouseID = &warehouseID
		}

		responses[i] = resp
	}
	return responses
}

// Helper function to suppress unused import warning
var _ = dto.Response{}
var _ = trade.ReturnStatusDraft
