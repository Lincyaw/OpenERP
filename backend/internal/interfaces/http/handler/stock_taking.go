package handler

import (
	"time"

	inventoryapp "github.com/erp/backend/internal/application/inventory"
	"github.com/erp/backend/internal/domain/inventory"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// StockTakingHandler handles stock taking-related API endpoints
type StockTakingHandler struct {
	BaseHandler
	stockTakingService *inventoryapp.StockTakingService
}

// NewStockTakingHandler creates a new StockTakingHandler
func NewStockTakingHandler(stockTakingService *inventoryapp.StockTakingService) *StockTakingHandler {
	return &StockTakingHandler{
		stockTakingService: stockTakingService,
	}
}

// ===================== Request/Response Types for Swagger =====================

// CreateStockTakingRequest represents a request to create a stock taking
// @Description Request body for creating a new stock taking
type CreateStockTakingRequest struct {
	WarehouseID   string `json:"warehouse_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	WarehouseName string `json:"warehouse_name" binding:"required" example:"Main Warehouse"`
	TakingDate    string `json:"taking_date" example:"2024-01-15"`
	Remark        string `json:"remark" example:"Monthly inventory count"`
	CreatedByID   string `json:"created_by_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440001"`
	CreatedByName string `json:"created_by_name" binding:"required" example:"John Doe"`
}

// UpdateStockTakingRequest represents a request to update a stock taking
// @Description Request body for updating a stock taking
type UpdateStockTakingRequest struct {
	Remark string `json:"remark" example:"Updated remark"`
}

// AddStockTakingItemRequest represents a request to add an item to stock taking
// @Description Request body for adding a product to stock taking
type AddStockTakingItemRequest struct {
	ProductID      string  `json:"product_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440002"`
	ProductName    string  `json:"product_name" binding:"required" example:"Widget A"`
	ProductCode    string  `json:"product_code" binding:"required" example:"WIDGET-001"`
	Unit           string  `json:"unit" binding:"required" example:"pcs"`
	SystemQuantity float64 `json:"system_quantity" binding:"required" example:"100.0"`
	UnitCost       float64 `json:"unit_cost" binding:"required" example:"15.50"`
}

// AddStockTakingItemsRequest represents a bulk request to add items
// @Description Request body for adding multiple products to stock taking
type AddStockTakingItemsRequest struct {
	Items []AddStockTakingItemRequest `json:"items" binding:"required,min=1"`
}

// RecordCountRequest represents a request to record the actual count
// @Description Request body for recording physical count for a product
type RecordCountRequest struct {
	ProductID      string  `json:"product_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440002"`
	ActualQuantity float64 `json:"actual_quantity" binding:"required,gte=0" example:"98.0"`
	Remark         string  `json:"remark" example:"2 units damaged"`
}

// RecordCountsRequest represents a bulk request to record counts
// @Description Request body for recording multiple physical counts
type RecordCountsRequest struct {
	Counts []RecordCountRequest `json:"counts" binding:"required,min=1"`
}

// ApproveStockTakingRequest represents a request to approve a stock taking
// @Description Request body for approving a stock taking
type ApproveStockTakingRequest struct {
	ApproverID   string `json:"approver_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440003"`
	ApproverName string `json:"approver_name" binding:"required" example:"Manager Smith"`
	Note         string `json:"note" example:"Approved after verification"`
}

// RejectStockTakingRequest represents a request to reject a stock taking
// @Description Request body for rejecting a stock taking
type RejectStockTakingRequest struct {
	ApproverID   string `json:"approver_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440003"`
	ApproverName string `json:"approver_name" binding:"required" example:"Manager Smith"`
	Reason       string `json:"reason" binding:"required,min=1,max=500" example:"Counts are inconsistent, please recount"`
}

// CancelStockTakingRequest represents a request to cancel a stock taking
// @Description Request body for cancelling a stock taking
type CancelStockTakingRequest struct {
	Reason string `json:"reason" binding:"max=500" example:"Stock taking no longer required"`
}

// StockTakingItemResponse represents a stock taking item in API responses
// @Description Stock taking item with count and difference information
type StockTakingItemResponse struct {
	ID               string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440010"`
	StockTakingID    string  `json:"stock_taking_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ProductID        string  `json:"product_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	ProductName      string  `json:"product_name" example:"Widget A"`
	ProductCode      string  `json:"product_code" example:"WIDGET-001"`
	Unit             string  `json:"unit" example:"pcs"`
	SystemQuantity   float64 `json:"system_quantity" example:"100.0"`
	ActualQuantity   float64 `json:"actual_quantity" example:"98.0"`
	DifferenceQty    float64 `json:"difference_qty" example:"-2.0"`
	UnitCost         float64 `json:"unit_cost" example:"15.50"`
	DifferenceAmount float64 `json:"difference_amount" example:"-31.0"`
	Counted          bool    `json:"counted" example:"true"`
	Remark           string  `json:"remark,omitempty" example:"2 units damaged"`
	CreatedAt        string  `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt        string  `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

// StockTakingResponse represents a stock taking in API responses
// @Description Stock taking document with full details
type StockTakingResponse struct {
	ID              string                    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantID        string                    `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440100"`
	TakingNumber    string                    `json:"taking_number" example:"ST-20240115-0001"`
	WarehouseID     string                    `json:"warehouse_id" example:"550e8400-e29b-41d4-a716-446655440050"`
	WarehouseName   string                    `json:"warehouse_name" example:"Main Warehouse"`
	Status          string                    `json:"status" example:"DRAFT"`
	TakingDate      string                    `json:"taking_date" example:"2024-01-15T00:00:00Z"`
	StartedAt       string                    `json:"started_at,omitempty" example:"2024-01-15T09:00:00Z"`
	CompletedAt     string                    `json:"completed_at,omitempty" example:"2024-01-15T12:00:00Z"`
	ApprovedAt      string                    `json:"approved_at,omitempty" example:"2024-01-15T14:00:00Z"`
	ApprovedByID    string                    `json:"approved_by_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440003"`
	ApprovedByName  string                    `json:"approved_by_name,omitempty" example:"Manager Smith"`
	CreatedByID     string                    `json:"created_by_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	CreatedByName   string                    `json:"created_by_name" example:"John Doe"`
	TotalItems      int                       `json:"total_items" example:"25"`
	CountedItems    int                       `json:"counted_items" example:"20"`
	DifferenceItems int                       `json:"difference_items" example:"3"`
	TotalDifference float64                   `json:"total_difference" example:"-155.50"`
	Progress        float64                   `json:"progress" example:"80.0"`
	ApprovalNote    string                    `json:"approval_note,omitempty" example:"Approved after verification"`
	Remark          string                    `json:"remark,omitempty" example:"Monthly inventory count"`
	Items           []StockTakingItemResponse `json:"items,omitempty"`
	CreatedAt       string                    `json:"created_at" example:"2024-01-15T08:00:00Z"`
	UpdatedAt       string                    `json:"updated_at" example:"2024-01-15T14:00:00Z"`
	Version         int                       `json:"version" example:"5"`
}

// StockTakingListResponse represents a stock taking in list views
// @Description Stock taking summary for list views
type StockTakingListResponse struct {
	ID              string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TakingNumber    string  `json:"taking_number" example:"ST-20240115-0001"`
	WarehouseID     string  `json:"warehouse_id" example:"550e8400-e29b-41d4-a716-446655440050"`
	WarehouseName   string  `json:"warehouse_name" example:"Main Warehouse"`
	Status          string  `json:"status" example:"COUNTING"`
	TakingDate      string  `json:"taking_date" example:"2024-01-15T00:00:00Z"`
	CreatedByID     string  `json:"created_by_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	CreatedByName   string  `json:"created_by_name" example:"John Doe"`
	TotalItems      int     `json:"total_items" example:"25"`
	CountedItems    int     `json:"counted_items" example:"20"`
	DifferenceItems int     `json:"difference_items" example:"3"`
	TotalDifference float64 `json:"total_difference" example:"-155.50"`
	Progress        float64 `json:"progress" example:"80.0"`
	CreatedAt       string  `json:"created_at" example:"2024-01-15T08:00:00Z"`
	UpdatedAt       string  `json:"updated_at" example:"2024-01-15T12:00:00Z"`
}

// StockTakingProgressResponse represents progress of a stock taking
// @Description Stock taking progress summary
type StockTakingProgressResponse struct {
	ID              string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TakingNumber    string  `json:"taking_number" example:"ST-20240115-0001"`
	Status          string  `json:"status" example:"COUNTING"`
	TotalItems      int     `json:"total_items" example:"25"`
	CountedItems    int     `json:"counted_items" example:"20"`
	DifferenceItems int     `json:"difference_items" example:"3"`
	TotalDifference float64 `json:"total_difference" example:"-155.50"`
	Progress        float64 `json:"progress" example:"80.0"`
	IsComplete      bool    `json:"is_complete" example:"false"`
}

// ===================== Query Handlers =====================

// GetByID godoc
// @Summary      Get stock taking by ID
// @Description  Retrieve a stock taking document by its ID with all items
// @Tags         stock-taking
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Stock Taking ID" format(uuid)
// @Success      200 {object} dto.Response{data=StockTakingResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /inventory/stock-takings/{id} [get]
func (h *StockTakingHandler) GetByID(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid stock taking ID format")
		return
	}

	result, err := h.stockTakingService.GetByID(c.Request.Context(), tenantID, id)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// GetByTakingNumber godoc
// @Summary      Get stock taking by number
// @Description  Retrieve a stock taking document by its taking number
// @Tags         stock-taking
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        taking_number path string true "Taking Number" example(ST-20240115-0001)
// @Success      200 {object} dto.Response{data=StockTakingResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /inventory/stock-takings/by-number/{taking_number} [get]
func (h *StockTakingHandler) GetByTakingNumber(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	takingNumber := c.Param("taking_number")
	if takingNumber == "" {
		h.BadRequest(c, "Taking number is required")
		return
	}

	result, err := h.stockTakingService.GetByTakingNumber(c.Request.Context(), tenantID, takingNumber)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// List godoc
// @Summary      List stock takings
// @Description  Retrieve a paginated list of stock takings with optional filtering
// @Tags         stock-taking
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        search query string false "Search term (taking number, warehouse, creator)"
// @Param        warehouse_id query string false "Filter by warehouse ID" format(uuid)
// @Param        status query string false "Filter by status" Enums(DRAFT, COUNTING, PENDING_APPROVAL, APPROVED, REJECTED, CANCELLED)
// @Param        start_date query string false "Filter by start date" format(date)
// @Param        end_date query string false "Filter by end date" format(date)
// @Param        created_by_id query string false "Filter by creator ID" format(uuid)
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20) maximum(100)
// @Param        order_by query string false "Order by field" default(created_at) Enums(taking_number, taking_date, status, created_at, updated_at, total_items)
// @Param        order_dir query string false "Order direction" Enums(asc, desc) default(desc)
// @Success      200 {object} dto.Response{data=[]StockTakingListResponse,meta=dto.Meta}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /inventory/stock-takings [get]
func (h *StockTakingHandler) List(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var filter inventoryapp.StockTakingListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Parse optional status
	if statusStr := c.Query("status"); statusStr != "" {
		status := inventory.StockTakingStatus(statusStr)
		if !status.IsValid() {
			h.BadRequest(c, "Invalid status value")
			return
		}
		filter.Status = &status
	}

	// Parse optional warehouse ID
	if whIDStr := c.Query("warehouse_id"); whIDStr != "" {
		whID, err := uuid.Parse(whIDStr)
		if err != nil {
			h.BadRequest(c, "Invalid warehouse ID format")
			return
		}
		filter.WarehouseID = &whID
	}

	// Parse optional creator ID
	if creatorIDStr := c.Query("created_by_id"); creatorIDStr != "" {
		creatorID, err := uuid.Parse(creatorIDStr)
		if err != nil {
			h.BadRequest(c, "Invalid creator ID format")
			return
		}
		filter.CreatedByID = &creatorID
	}

	// Parse optional date range
	if startDateStr := c.Query("start_date"); startDateStr != "" {
		startDate, err := parseDateTime(startDateStr)
		if err != nil {
			h.BadRequest(c, "Invalid start_date format")
			return
		}
		filter.StartDate = &startDate
	}
	if endDateStr := c.Query("end_date"); endDateStr != "" {
		endDate, err := parseDateTime(endDateStr)
		if err != nil {
			h.BadRequest(c, "Invalid end_date format")
			return
		}
		filter.EndDate = &endDate
	}

	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	items, total, err := h.stockTakingService.List(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.SuccessWithMeta(c, items, total, filter.Page, filter.PageSize)
}

// ListPendingApproval godoc
// @Summary      List stock takings pending approval
// @Description  Retrieve stock takings that are awaiting approval
// @Tags         stock-taking
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        search query string false "Search term"
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20) maximum(100)
// @Param        order_by query string false "Order by field" default(created_at)
// @Param        order_dir query string false "Order direction" Enums(asc, desc) default(desc)
// @Success      200 {object} dto.Response{data=[]StockTakingListResponse,meta=dto.Meta}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /inventory/stock-takings/pending-approval [get]
func (h *StockTakingHandler) ListPendingApproval(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var filter inventoryapp.StockTakingListFilter
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

	items, total, err := h.stockTakingService.ListPendingApproval(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.SuccessWithMeta(c, items, total, filter.Page, filter.PageSize)
}

// GetProgress godoc
// @Summary      Get stock taking progress
// @Description  Retrieve the counting progress of a stock taking
// @Tags         stock-taking
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Stock Taking ID" format(uuid)
// @Success      200 {object} dto.Response{data=StockTakingProgressResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /inventory/stock-takings/{id}/progress [get]
func (h *StockTakingHandler) GetProgress(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid stock taking ID format")
		return
	}

	result, err := h.stockTakingService.GetProgress(c.Request.Context(), tenantID, id)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// ===================== Command Handlers =====================

// Create godoc
// @Summary      Create stock taking
// @Description  Create a new stock taking document
// @Tags         stock-taking
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        request body CreateStockTakingRequest true "Stock taking creation request"
// @Success      201 {object} dto.Response{data=StockTakingResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /inventory/stock-takings [post]
func (h *StockTakingHandler) Create(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req CreateStockTakingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	warehouseID, err := uuid.Parse(req.WarehouseID)
	if err != nil {
		h.BadRequest(c, "Invalid warehouse ID format")
		return
	}

	createdByID, err := uuid.Parse(req.CreatedByID)
	if err != nil {
		h.BadRequest(c, "Invalid creator ID format")
		return
	}

	appReq := inventoryapp.CreateStockTakingRequest{
		WarehouseID:   warehouseID,
		WarehouseName: req.WarehouseName,
		Remark:        req.Remark,
		CreatedByID:   createdByID,
		CreatedByName: req.CreatedByName,
	}

	// Parse optional taking date
	if req.TakingDate != "" {
		takingDate, err := parseDateTime(req.TakingDate)
		if err != nil {
			h.BadRequest(c, "Invalid taking date format")
			return
		}
		appReq.TakingDate = &takingDate
	}

	result, err := h.stockTakingService.Create(c.Request.Context(), tenantID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Created(c, result)
}

// Update godoc
// @Summary      Update stock taking
// @Description  Update a stock taking document (only in DRAFT status)
// @Tags         stock-taking
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Stock Taking ID" format(uuid)
// @Param        request body UpdateStockTakingRequest true "Stock taking update request"
// @Success      200 {object} dto.Response{data=StockTakingResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /inventory/stock-takings/{id} [put]
func (h *StockTakingHandler) Update(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid stock taking ID format")
		return
	}

	var req UpdateStockTakingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	appReq := inventoryapp.UpdateStockTakingRequest{
		Remark: req.Remark,
	}

	result, err := h.stockTakingService.Update(c.Request.Context(), tenantID, id, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// Delete godoc
// @Summary      Delete stock taking
// @Description  Delete a stock taking document (only in DRAFT status)
// @Tags         stock-taking
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Stock Taking ID" format(uuid)
// @Success      204
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /inventory/stock-takings/{id} [delete]
func (h *StockTakingHandler) Delete(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid stock taking ID format")
		return
	}

	if err := h.stockTakingService.Delete(c.Request.Context(), tenantID, id); err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.NoContent(c)
}

// AddItem godoc
// @Summary      Add item to stock taking
// @Description  Add a product to the stock taking document
// @Tags         stock-taking
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Stock Taking ID" format(uuid)
// @Param        request body AddStockTakingItemRequest true "Item to add"
// @Success      200 {object} dto.Response{data=StockTakingResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /inventory/stock-takings/{id}/items [post]
func (h *StockTakingHandler) AddItem(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	stockTakingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid stock taking ID format")
		return
	}

	var req AddStockTakingItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	appReq := inventoryapp.AddStockTakingItemRequest{
		ProductID:      productID,
		ProductName:    req.ProductName,
		ProductCode:    req.ProductCode,
		Unit:           req.Unit,
		SystemQuantity: decimal.NewFromFloat(req.SystemQuantity),
		UnitCost:       decimal.NewFromFloat(req.UnitCost),
	}

	result, err := h.stockTakingService.AddItem(c.Request.Context(), tenantID, stockTakingID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// AddItems godoc
// @Summary      Add multiple items to stock taking
// @Description  Add multiple products to the stock taking document
// @Tags         stock-taking
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Stock Taking ID" format(uuid)
// @Param        request body AddStockTakingItemsRequest true "Items to add"
// @Success      200 {object} dto.Response{data=StockTakingResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /inventory/stock-takings/{id}/items/bulk [post]
func (h *StockTakingHandler) AddItems(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	stockTakingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid stock taking ID format")
		return
	}

	var req AddStockTakingItemsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Convert to application DTOs
	appItems := make([]inventoryapp.AddStockTakingItemRequest, 0, len(req.Items))
	for _, item := range req.Items {
		productID, err := uuid.Parse(item.ProductID)
		if err != nil {
			h.BadRequest(c, "Invalid product ID format: "+item.ProductID)
			return
		}
		appItems = append(appItems, inventoryapp.AddStockTakingItemRequest{
			ProductID:      productID,
			ProductName:    item.ProductName,
			ProductCode:    item.ProductCode,
			Unit:           item.Unit,
			SystemQuantity: decimal.NewFromFloat(item.SystemQuantity),
			UnitCost:       decimal.NewFromFloat(item.UnitCost),
		})
	}

	appReq := inventoryapp.AddStockTakingItemsRequest{
		Items: appItems,
	}

	result, err := h.stockTakingService.AddItems(c.Request.Context(), tenantID, stockTakingID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// RemoveItem godoc
// @Summary      Remove item from stock taking
// @Description  Remove a product from the stock taking document
// @Tags         stock-taking
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Stock Taking ID" format(uuid)
// @Param        product_id path string true "Product ID" format(uuid)
// @Success      200 {object} dto.Response{data=StockTakingResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /inventory/stock-takings/{id}/items/{product_id} [delete]
func (h *StockTakingHandler) RemoveItem(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	stockTakingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid stock taking ID format")
		return
	}

	productID, err := uuid.Parse(c.Param("product_id"))
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	result, err := h.stockTakingService.RemoveItem(c.Request.Context(), tenantID, stockTakingID, productID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// StartCounting godoc
// @Summary      Start counting
// @Description  Transition the stock taking to counting status
// @Tags         stock-taking
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Stock Taking ID" format(uuid)
// @Success      200 {object} dto.Response{data=StockTakingResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /inventory/stock-takings/{id}/start [post]
func (h *StockTakingHandler) StartCounting(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid stock taking ID format")
		return
	}

	result, err := h.stockTakingService.StartCounting(c.Request.Context(), tenantID, id)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// RecordCount godoc
// @Summary      Record count for an item
// @Description  Record the actual physical count for a product
// @Tags         stock-taking
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Stock Taking ID" format(uuid)
// @Param        request body RecordCountRequest true "Count record"
// @Success      200 {object} dto.Response{data=StockTakingResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /inventory/stock-takings/{id}/count [post]
func (h *StockTakingHandler) RecordCount(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	stockTakingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid stock taking ID format")
		return
	}

	var req RecordCountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	appReq := inventoryapp.RecordCountRequest{
		ProductID:      productID,
		ActualQuantity: decimal.NewFromFloat(req.ActualQuantity),
		Remark:         req.Remark,
	}

	result, err := h.stockTakingService.RecordCount(c.Request.Context(), tenantID, stockTakingID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// RecordCounts godoc
// @Summary      Record counts for multiple items
// @Description  Record actual physical counts for multiple products
// @Tags         stock-taking
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Stock Taking ID" format(uuid)
// @Param        request body RecordCountsRequest true "Count records"
// @Success      200 {object} dto.Response{data=StockTakingResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /inventory/stock-takings/{id}/counts [post]
func (h *StockTakingHandler) RecordCounts(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	stockTakingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid stock taking ID format")
		return
	}

	var req RecordCountsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Convert to application DTOs
	appCounts := make([]inventoryapp.RecordCountRequest, 0, len(req.Counts))
	for _, count := range req.Counts {
		productID, err := uuid.Parse(count.ProductID)
		if err != nil {
			h.BadRequest(c, "Invalid product ID format: "+count.ProductID)
			return
		}
		appCounts = append(appCounts, inventoryapp.RecordCountRequest{
			ProductID:      productID,
			ActualQuantity: decimal.NewFromFloat(count.ActualQuantity),
			Remark:         count.Remark,
		})
	}

	appReq := inventoryapp.RecordCountsRequest{
		Counts: appCounts,
	}

	result, err := h.stockTakingService.RecordCounts(c.Request.Context(), tenantID, stockTakingID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// SubmitForApproval godoc
// @Summary      Submit for approval
// @Description  Submit the stock taking for approval (all items must be counted)
// @Tags         stock-taking
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Stock Taking ID" format(uuid)
// @Success      200 {object} dto.Response{data=StockTakingResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /inventory/stock-takings/{id}/submit [post]
func (h *StockTakingHandler) SubmitForApproval(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid stock taking ID format")
		return
	}

	result, err := h.stockTakingService.SubmitForApproval(c.Request.Context(), tenantID, id)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// Approve godoc
// @Summary      Approve stock taking
// @Description  Approve the stock taking and trigger inventory adjustments
// @Tags         stock-taking
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Stock Taking ID" format(uuid)
// @Param        request body ApproveStockTakingRequest true "Approval request"
// @Success      200 {object} dto.Response{data=StockTakingResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /inventory/stock-takings/{id}/approve [post]
func (h *StockTakingHandler) Approve(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid stock taking ID format")
		return
	}

	var req ApproveStockTakingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	approverID, err := uuid.Parse(req.ApproverID)
	if err != nil {
		h.BadRequest(c, "Invalid approver ID format")
		return
	}

	appReq := inventoryapp.ApproveStockTakingRequest{
		ApproverID:   approverID,
		ApproverName: req.ApproverName,
		Note:         req.Note,
	}

	result, err := h.stockTakingService.Approve(c.Request.Context(), tenantID, id, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// Reject godoc
// @Summary      Reject stock taking
// @Description  Reject the stock taking with a reason
// @Tags         stock-taking
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Stock Taking ID" format(uuid)
// @Param        request body RejectStockTakingRequest true "Rejection request"
// @Success      200 {object} dto.Response{data=StockTakingResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /inventory/stock-takings/{id}/reject [post]
func (h *StockTakingHandler) Reject(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid stock taking ID format")
		return
	}

	var req RejectStockTakingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	approverID, err := uuid.Parse(req.ApproverID)
	if err != nil {
		h.BadRequest(c, "Invalid approver ID format")
		return
	}

	appReq := inventoryapp.RejectStockTakingRequest{
		ApproverID:   approverID,
		ApproverName: req.ApproverName,
		Reason:       req.Reason,
	}

	result, err := h.stockTakingService.Reject(c.Request.Context(), tenantID, id, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// Cancel godoc
// @Summary      Cancel stock taking
// @Description  Cancel the stock taking (only in DRAFT or COUNTING status)
// @Tags         stock-taking
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Stock Taking ID" format(uuid)
// @Param        request body CancelStockTakingRequest true "Cancellation request"
// @Success      200 {object} dto.Response{data=StockTakingResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /inventory/stock-takings/{id}/cancel [post]
func (h *StockTakingHandler) Cancel(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid stock taking ID format")
		return
	}

	var req CancelStockTakingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	appReq := inventoryapp.CancelStockTakingRequest{
		Reason: req.Reason,
	}

	result, err := h.stockTakingService.Cancel(c.Request.Context(), tenantID, id, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// Suppress unused import warning
var _ = time.Time{}
