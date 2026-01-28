package handler

import (
	partnerapp "github.com/erp/backend/internal/application/partner"
	"github.com/erp/backend/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// WarehouseHandler handles warehouse-related API endpoints
type WarehouseHandler struct {
	BaseHandler
	warehouseService *partnerapp.WarehouseService
}

// NewWarehouseHandler creates a new WarehouseHandler
func NewWarehouseHandler(warehouseService *partnerapp.WarehouseService) *WarehouseHandler {
	return &WarehouseHandler{
		warehouseService: warehouseService,
	}
}

// CreateWarehouseRequest represents a request to create a new warehouse
// @Description	Request body for creating a new warehouse
// @Name HandlerCreateWarehouseRequest
type CreateWarehouseRequest struct {
	Code        string `json:"code" binding:"required,min=1,max=50" example:"WH-001"`
	Name        string `json:"name" binding:"required,min=1,max=200" example:"Main Warehouse"`
	ShortName   string `json:"short_name" binding:"max=100" example:"Main"`
	Type        string `json:"type" binding:"required,oneof=physical virtual consign transit" example:"physical"`
	ContactName string `json:"contact_name" binding:"max=100" example:"Wang Wei"`
	Phone       string `json:"phone" binding:"max=50" example:"13500135000"`
	Email       string `json:"email" binding:"omitempty,email,max=200" example:"warehouse@company.com"`
	Address     string `json:"address" binding:"max=500" example:"Building 5, Industrial Zone"`
	City        string `json:"city" binding:"max=100" example:"Shanghai"`
	Province    string `json:"province" binding:"max=100" example:"Shanghai"`
	PostalCode  string `json:"postal_code" binding:"max=20" example:"201100"`
	Country     string `json:"country" binding:"max=100" example:"China"`
	IsDefault   *bool  `json:"is_default" example:"true"`
	Capacity    *int   `json:"capacity" example:"10000"`
	Notes       string `json:"notes" example:"Primary storage facility"`
	SortOrder   *int   `json:"sort_order" example:"0"`
	Attributes  string `json:"attributes" example:"{}"`
}

// UpdateWarehouseRequest represents a request to update a warehouse
//
//	@Description	Request body for updating a warehouse
type UpdateWarehouseRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=1,max=200" example:"Main Distribution Center"`
	ShortName   *string `json:"short_name" binding:"omitempty,max=100" example:"MDC"`
	ContactName *string `json:"contact_name" binding:"omitempty,max=100" example:"Li Ming"`
	Phone       *string `json:"phone" binding:"omitempty,max=50" example:"13600136000"`
	Email       *string `json:"email" binding:"omitempty,email,max=200" example:"mdc@company.com"`
	Address     *string `json:"address" binding:"omitempty,max=500" example:"Block A, Logistics Park"`
	City        *string `json:"city" binding:"omitempty,max=100" example:"Hangzhou"`
	Province    *string `json:"province" binding:"omitempty,max=100" example:"Zhejiang"`
	PostalCode  *string `json:"postal_code" binding:"omitempty,max=20" example:"310000"`
	Country     *string `json:"country" binding:"omitempty,max=100" example:"China"`
	IsDefault   *bool   `json:"is_default" example:"false"`
	Capacity    *int    `json:"capacity" example:"15000"`
	Notes       *string `json:"notes" example:"Upgraded capacity"`
	SortOrder   *int    `json:"sort_order" example:"1"`
	Attributes  *string `json:"attributes" example:"{}"`
}

// UpdateWarehouseCodeRequest represents a request to update a warehouse's code
//
//	@Description	Request body for updating a warehouse's code
type UpdateWarehouseCodeRequest struct {
	Code string `json:"code" binding:"required,min=1,max=50" example:"WH-002"`
}

// Create godoc
// @ID           createWarehouse
//
//	@Summary		Create a new warehouse
//	@Description	Create a new warehouse in the partner module
//	@Tags			warehouses
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string					false	"Tenant ID (optional for dev)"
//	@Param			request		body		CreateWarehouseRequest	true	"Warehouse creation request"
//	@Success		201			{object}	APIResponse[WarehouseResponse]
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/partner/warehouses [post]
func (h *WarehouseHandler) Create(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req CreateWarehouseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Get user ID from JWT context (optional, for data scope)
	userID, _ := getUserID(c)

	// Convert to application DTO
	appReq := partnerapp.CreateWarehouseRequest{
		Code:        req.Code,
		Name:        req.Name,
		ShortName:   req.ShortName,
		Type:        req.Type,
		ContactName: req.ContactName,
		Phone:       req.Phone,
		Email:       req.Email,
		Address:     req.Address,
		City:        req.City,
		Province:    req.Province,
		PostalCode:  req.PostalCode,
		Country:     req.Country,
		IsDefault:   req.IsDefault,
		Capacity:    req.Capacity,
		Notes:       req.Notes,
		SortOrder:   req.SortOrder,
		Attributes:  req.Attributes,
	}

	// Set CreatedBy for data scope filtering
	if userID != uuid.Nil {
		appReq.CreatedBy = &userID
	}

	warehouse, err := h.warehouseService.Create(c.Request.Context(), tenantID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Created(c, warehouse)
}

// GetByID godoc
// @ID           getWarehouseById
//
//	@Summary		Get warehouse by ID
//	@Description	Retrieve a warehouse by its ID
//	@Tags			warehouses
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			id			path		string	true	"Warehouse ID"	format(uuid)
//	@Success		200			{object}	APIResponse[WarehouseResponse]
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/partner/warehouses/{id} [get]
func (h *WarehouseHandler) GetByID(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	warehouseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid warehouse ID format")
		return
	}

	warehouse, err := h.warehouseService.GetByID(c.Request.Context(), tenantID, warehouseID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, warehouse)
}

// GetByCode godoc
// @ID           getWarehouseByCode
//
//	@Summary		Get warehouse by code
//	@Description	Retrieve a warehouse by its code
//	@Tags			warehouses
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			code		path		string	true	"Warehouse Code"
//	@Success		200			{object}	APIResponse[WarehouseResponse]
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/partner/warehouses/code/{code} [get]
func (h *WarehouseHandler) GetByCode(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	code := c.Param("code")
	if code == "" {
		h.BadRequest(c, "Warehouse code is required")
		return
	}

	warehouse, err := h.warehouseService.GetByCode(c.Request.Context(), tenantID, code)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, warehouse)
}

// GetDefault godoc
// @ID           getWarehouseDefault
//
//	@Summary		Get default warehouse
//	@Description	Retrieve the default warehouse for the tenant
//	@Tags			warehouses
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Success		200			{object}	APIResponse[WarehouseResponse]
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/partner/warehouses/default [get]
func (h *WarehouseHandler) GetDefault(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	warehouse, err := h.warehouseService.GetDefault(c.Request.Context(), tenantID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, warehouse)
}

// List godoc
// @ID           listWarehouses
//
//	@Summary		List warehouses
//	@Description	Retrieve a paginated list of warehouses with optional filtering
//	@Tags			warehouses
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			search		query		string	false	"Search term (name, code)"
//	@Param			status		query		string	false	"Warehouse status"	Enums(enabled, disabled)
//	@Param			type		query		string	false	"Warehouse type"	Enums(physical, virtual, consign, transit)
//	@Param			city		query		string	false	"City"
//	@Param			province	query		string	false	"Province"
//	@Param			is_default	query		boolean	false	"Filter by default status"
//	@Param			page		query		int		false	"Page number"		default(1)
//	@Param			page_size	query		int		false	"Page size"			default(20)	maximum(100)
//	@Param			order_by	query		string	false	"Order by field"	default(is_default)
//	@Param			order_dir	query		string	false	"Order direction"	Enums(asc, desc)	default(desc)
//	@Success		200			{object}	APIResponse[[]WarehouseListResponse]
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/partner/warehouses [get]
func (h *WarehouseHandler) List(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var filter partnerapp.WarehouseListFilter
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

	warehouses, total, err := h.warehouseService.List(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.SuccessWithMeta(c, warehouses, total, filter.Page, filter.PageSize)
}

// Update godoc
// @ID           updateWarehouse
//
//	@Summary		Update a warehouse
//	@Description	Update an existing warehouse's details
//	@Tags			warehouses
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string					false	"Tenant ID (optional for dev)"
//	@Param			id			path		string					true	"Warehouse ID"	format(uuid)
//	@Param			request		body		UpdateWarehouseRequest	true	"Warehouse update request"
//	@Success		200			{object}	APIResponse[WarehouseResponse]
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/partner/warehouses/{id} [put]
func (h *WarehouseHandler) Update(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	warehouseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid warehouse ID format")
		return
	}

	var req UpdateWarehouseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Convert to application DTO
	appReq := partnerapp.UpdateWarehouseRequest{
		Name:        req.Name,
		ShortName:   req.ShortName,
		ContactName: req.ContactName,
		Phone:       req.Phone,
		Email:       req.Email,
		Address:     req.Address,
		City:        req.City,
		Province:    req.Province,
		PostalCode:  req.PostalCode,
		Country:     req.Country,
		IsDefault:   req.IsDefault,
		Capacity:    req.Capacity,
		Notes:       req.Notes,
		SortOrder:   req.SortOrder,
		Attributes:  req.Attributes,
	}

	warehouse, err := h.warehouseService.Update(c.Request.Context(), tenantID, warehouseID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, warehouse)
}

// UpdateCode godoc
// @ID           updateWarehouseCode
//
//	@Summary		Update warehouse code
//	@Description	Update a warehouse's code
//	@Tags			warehouses
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string						false	"Tenant ID (optional for dev)"
//	@Param			id			path		string						true	"Warehouse ID"	format(uuid)
//	@Param			request		body		UpdateWarehouseCodeRequest	true	"New warehouse code"
//	@Success		200			{object}	APIResponse[WarehouseResponse]
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/partner/warehouses/{id}/code [put]
func (h *WarehouseHandler) UpdateCode(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	warehouseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid warehouse ID format")
		return
	}

	var req UpdateWarehouseCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	warehouse, err := h.warehouseService.UpdateCode(c.Request.Context(), tenantID, warehouseID, req.Code)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, warehouse)
}

// Delete godoc
// @ID           deleteWarehouse
//
//	@Summary		Delete a warehouse
//	@Description	Delete a warehouse by ID (cannot delete default warehouse or warehouse with inventory unless force=true with admin permission)
//	@Tags			warehouses
//	@Produce		json
//	@Param			X-Tenant-ID	header	string	false	"Tenant ID (optional for dev)"
//	@Param			id			path	string	true	"Warehouse ID"	format(uuid)
//	@Param			force		query	bool	false	"Force delete even if warehouse has inventory (requires admin permission)"
//	@Success		204
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		422	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/partner/warehouses/{id} [delete]
func (h *WarehouseHandler) Delete(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	warehouseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid warehouse ID format")
		return
	}

	// Check for force delete parameter
	forceStr := c.Query("force")
	force := forceStr == "true" || forceStr == "1"

	// If force delete is requested, check for admin permission
	if force {
		// Require warehouse:force_delete permission for force delete
		// This permission should only be granted to admin roles
		if !middleware.HasPermission(c, "warehouse:force_delete") {
			h.Forbidden(c, "Force delete requires admin permission (warehouse:force_delete)")
			return
		}
	}

	opts := partnerapp.DeleteOptions{
		Force: force,
	}

	err = h.warehouseService.DeleteWithOptions(c.Request.Context(), tenantID, warehouseID, opts)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.NoContent(c)
}

// Enable godoc
// @ID           enableWarehouse
//
//	@Summary		Enable a warehouse
//	@Description	Enable an inactive warehouse
//	@Tags			warehouses
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			id			path		string	true	"Warehouse ID"	format(uuid)
//	@Success		200			{object}	APIResponse[WarehouseResponse]
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		422			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/partner/warehouses/{id}/enable [post]
func (h *WarehouseHandler) Enable(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	warehouseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid warehouse ID format")
		return
	}

	warehouse, err := h.warehouseService.Enable(c.Request.Context(), tenantID, warehouseID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, warehouse)
}

// Disable godoc
// @ID           disableWarehouse
//
//	@Summary		Disable a warehouse
//	@Description	Disable an active warehouse
//	@Tags			warehouses
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			id			path		string	true	"Warehouse ID"	format(uuid)
//	@Success		200			{object}	APIResponse[WarehouseResponse]
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		422			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/partner/warehouses/{id}/disable [post]
func (h *WarehouseHandler) Disable(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	warehouseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid warehouse ID format")
		return
	}

	warehouse, err := h.warehouseService.Disable(c.Request.Context(), tenantID, warehouseID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, warehouse)
}

// SetDefault godoc
// @ID           setDefaultWarehouse
//
//	@Summary		Set default warehouse
//	@Description	Set a warehouse as the default warehouse
//	@Tags			warehouses
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			id			path		string	true	"Warehouse ID"	format(uuid)
//	@Success		200			{object}	APIResponse[WarehouseResponse]
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		422			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/partner/warehouses/{id}/set-default [post]
func (h *WarehouseHandler) SetDefault(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	warehouseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid warehouse ID format")
		return
	}

	warehouse, err := h.warehouseService.SetDefault(c.Request.Context(), tenantID, warehouseID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, warehouse)
}

// WarehouseCountByStatusResponse represents the warehouse count by status response
type WarehouseCountByStatusResponse struct {
	Active   int64 `json:"active" example:"8"`
	Inactive int64 `json:"inactive" example:"2"`
	Total    int64 `json:"total" example:"10"`
}

// CountByStatus godoc
// @ID           countWarehouseByStatus
//
//	@Summary		Get warehouse counts by status
//	@Description	Get the count of warehouses grouped by status
//	@Tags			warehouses
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Success		200			{object}	APIResponse[WarehouseCountByStatusResponse]
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/partner/warehouses/stats/count [get]
func (h *WarehouseHandler) CountByStatus(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	counts, err := h.warehouseService.CountByStatus(c.Request.Context(), tenantID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	response := WarehouseCountByStatusResponse{
		Active:   counts["active"],
		Inactive: counts["inactive"],
		Total:    counts["total"],
	}

	h.Success(c, response)
}
