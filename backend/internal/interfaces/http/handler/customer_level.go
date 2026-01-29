package handler

import (
	"regexp"

	partnerapp "github.com/erp/backend/internal/application/partner"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// validCodePattern matches alphanumeric codes with underscores and hyphens
var validCodePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// isValidLevelCode validates a customer level code
func isValidLevelCode(code string) bool {
	return len(code) > 0 && len(code) <= 50 && validCodePattern.MatchString(code)
}

// CustomerLevelHandler handles customer level-related API endpoints
type CustomerLevelHandler struct {
	BaseHandler
	levelService *partnerapp.CustomerLevelService
}

// NewCustomerLevelHandler creates a new CustomerLevelHandler
func NewCustomerLevelHandler(levelService *partnerapp.CustomerLevelService) *CustomerLevelHandler {
	return &CustomerLevelHandler{
		levelService: levelService,
	}
}

// =============================================================================
// Customer Level Response DTOs (for Swagger)
// =============================================================================

// CustomerLevelResponse represents a customer level in API responses
//
//	@Description	Customer level details
type CustomerLevelResponse struct {
	ID              string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantID        string  `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Code            string  `json:"code" example:"gold"`
	Name            string  `json:"name" example:"金卡会员"`
	DiscountRate    float64 `json:"discount_rate" example:"0.05"`
	DiscountPercent float64 `json:"discount_percent" example:"5"`
	SortOrder       int     `json:"sort_order" example:"2"`
	IsDefault       bool    `json:"is_default" example:"false"`
	IsActive        bool    `json:"is_active" example:"true"`
	Description     string  `json:"description" example:"5% discount on all purchases"`
	CustomerCount   int64   `json:"customer_count,omitempty" example:"150"`
	CreatedAt       string  `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt       string  `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

// CustomerLevelListResponse represents a customer level list item
//
//	@Description	Customer level list item with customer count
type CustomerLevelListResponse struct {
	ID              string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Code            string  `json:"code" example:"gold"`
	Name            string  `json:"name" example:"金卡会员"`
	DiscountRate    float64 `json:"discount_rate" example:"0.05"`
	DiscountPercent float64 `json:"discount_percent" example:"5"`
	SortOrder       int     `json:"sort_order" example:"2"`
	IsDefault       bool    `json:"is_default" example:"false"`
	IsActive        bool    `json:"is_active" example:"true"`
	CustomerCount   int64   `json:"customer_count" example:"150"`
}

// =============================================================================
// Customer Level Request DTOs
// =============================================================================

// CreateCustomerLevelRequest represents a request to create a customer level
//
//	@Description	Request body for creating a customer level
type CreateCustomerLevelRequest struct {
	Code         string  `json:"code" binding:"required,min=1,max=50" example:"gold"`
	Name         string  `json:"name" binding:"required,min=1,max=100" example:"金卡会员"`
	DiscountRate float64 `json:"discount_rate" binding:"gte=0,lte=1" example:"0.05"`
	SortOrder    *int    `json:"sort_order" example:"2"`
	IsDefault    bool    `json:"is_default" example:"false"`
	IsActive     bool    `json:"is_active" example:"true"`
	Description  string  `json:"description" binding:"max=500" example:"5% discount on all purchases"`
}

// UpdateCustomerLevelRequest represents a request to update a customer level
//
//	@Description	Request body for updating a customer level
type UpdateCustomerLevelRequest struct {
	Name         *string  `json:"name" binding:"omitempty,min=1,max=100" example:"金卡会员"`
	DiscountRate *float64 `json:"discount_rate" binding:"omitempty,gte=0,lte=1" example:"0.08"`
	SortOrder    *int     `json:"sort_order" example:"3"`
	IsDefault    *bool    `json:"is_default" example:"false"`
	IsActive     *bool    `json:"is_active" example:"true"`
	Description  *string  `json:"description" binding:"omitempty,max=500" example:"Updated description"`
}

// Create godoc
//
//	@ID				createCustomerLevel
//	@Summary		Create a new customer level
//	@Description	Create a new customer level for the tenant
//	@Tags			customer-levels
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string						false	"Tenant ID (optional for dev)"
//	@Param			request		body		CreateCustomerLevelRequest	true	"Customer level creation request"
//	@Success		201			{object}	APIResponse[CustomerLevelResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		409			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/partner/customer-levels [post]
func (h *CustomerLevelHandler) Create(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req CreateCustomerLevelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	appReq := partnerapp.CreateCustomerLevelRequest{
		Code:         req.Code,
		Name:         req.Name,
		DiscountRate: req.DiscountRate,
		SortOrder:    req.SortOrder,
		IsDefault:    req.IsDefault,
		IsActive:     req.IsActive,
		Description:  req.Description,
	}

	level, err := h.levelService.Create(c.Request.Context(), tenantID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Created(c, level)
}

// GetByID godoc
//
//	@ID				getCustomerLevelById
//	@Summary		Get customer level by ID
//	@Description	Retrieve a customer level by its ID
//	@Tags			customer-levels
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			id			path		string	true	"Customer Level ID"	format(uuid)
//	@Success		200			{object}	APIResponse[CustomerLevelResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/partner/customer-levels/{id} [get]
func (h *CustomerLevelHandler) GetByID(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	levelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid customer level ID format")
		return
	}

	level, err := h.levelService.GetByID(c.Request.Context(), tenantID, levelID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, level)
}

// GetByCode godoc
//
//	@ID				getCustomerLevelByCode
//	@Summary		Get customer level by code
//	@Description	Retrieve a customer level by its code
//	@Tags			customer-levels
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			code		path		string	true	"Customer Level Code"
//	@Success		200			{object}	APIResponse[CustomerLevelResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/partner/customer-levels/code/{code} [get]
func (h *CustomerLevelHandler) GetByCode(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	code := c.Param("code")
	if code == "" {
		h.BadRequest(c, "Customer level code is required")
		return
	}
	if !isValidLevelCode(code) {
		h.BadRequest(c, "Invalid customer level code format")
		return
	}

	level, err := h.levelService.GetByCode(c.Request.Context(), tenantID, code)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, level)
}

// GetDefault godoc
//
//	@ID				getCustomerLevelDefault
//	@Summary		Get default customer level
//	@Description	Retrieve the default customer level for the tenant
//	@Tags			customer-levels
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Success		200			{object}	APIResponse[CustomerLevelResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/partner/customer-levels/default [get]
func (h *CustomerLevelHandler) GetDefault(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	level, err := h.levelService.GetDefault(c.Request.Context(), tenantID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, level)
}

// List godoc
//
//	@ID				listCustomerLevels
//	@Summary		List customer levels
//	@Description	Retrieve all customer levels for the tenant
//	@Tags			customer-levels
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			active_only	query		bool	false	"Only return active levels"	default(false)
//	@Success		200			{object}	APIResponse[[]CustomerLevelListResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/partner/customer-levels [get]
func (h *CustomerLevelHandler) List(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	activeOnly := c.Query("active_only") == "true"

	levels, err := h.levelService.List(c.Request.Context(), tenantID, activeOnly)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, levels)
}

// Update godoc
//
//	@ID				updateCustomerLevel
//	@Summary		Update a customer level
//	@Description	Update an existing customer level's details
//	@Tags			customer-levels
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string						false	"Tenant ID (optional for dev)"
//	@Param			id			path		string						true	"Customer Level ID"	format(uuid)
//	@Param			request		body		UpdateCustomerLevelRequest	true	"Customer level update request"
//	@Success		200			{object}	APIResponse[CustomerLevelResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/partner/customer-levels/{id} [put]
func (h *CustomerLevelHandler) Update(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	levelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid customer level ID format")
		return
	}

	var req UpdateCustomerLevelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	appReq := partnerapp.UpdateCustomerLevelRequest{
		Name:         req.Name,
		DiscountRate: req.DiscountRate,
		SortOrder:    req.SortOrder,
		IsDefault:    req.IsDefault,
		IsActive:     req.IsActive,
		Description:  req.Description,
	}

	level, err := h.levelService.Update(c.Request.Context(), tenantID, levelID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, level)
}

// Delete godoc
//
//	@ID				deleteCustomerLevel
//	@Summary		Delete a customer level
//	@Description	Delete a customer level by ID
//	@Tags			customer-levels
//	@Produce		json
//	@Param			X-Tenant-ID	header	string	false	"Tenant ID (optional for dev)"
//	@Param			id			path	string	true	"Customer Level ID"	format(uuid)
//	@Success		204
//	@Failure		400	{object}	dto.ErrorResponse
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Failure		422	{object}	dto.ErrorResponse
//	@Failure		500	{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/partner/customer-levels/{id} [delete]
func (h *CustomerLevelHandler) Delete(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	levelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid customer level ID format")
		return
	}

	err = h.levelService.Delete(c.Request.Context(), tenantID, levelID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.NoContent(c)
}

// SetDefault godoc
//
//	@ID				setDefaultCustomerLevel
//	@Summary		Set default customer level
//	@Description	Set a customer level as the default for new customers
//	@Tags			customer-levels
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			id			path		string	true	"Customer Level ID"	format(uuid)
//	@Success		200			{object}	APIResponse[CustomerLevelResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		422			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/partner/customer-levels/{id}/set-default [post]
func (h *CustomerLevelHandler) SetDefault(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	levelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid customer level ID format")
		return
	}

	level, err := h.levelService.SetDefault(c.Request.Context(), tenantID, levelID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, level)
}

// Activate godoc
//
//	@ID				activateCustomerLevel
//	@Summary		Activate a customer level
//	@Description	Activate an inactive customer level
//	@Tags			customer-levels
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			id			path		string	true	"Customer Level ID"	format(uuid)
//	@Success		200			{object}	APIResponse[CustomerLevelResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/partner/customer-levels/{id}/activate [post]
func (h *CustomerLevelHandler) Activate(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	levelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid customer level ID format")
		return
	}

	level, err := h.levelService.Activate(c.Request.Context(), tenantID, levelID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, level)
}

// Deactivate godoc
//
//	@ID				deactivateCustomerLevel
//	@Summary		Deactivate a customer level
//	@Description	Deactivate an active customer level
//	@Tags			customer-levels
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			id			path		string	true	"Customer Level ID"	format(uuid)
//	@Success		200			{object}	APIResponse[CustomerLevelResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		422			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/partner/customer-levels/{id}/deactivate [post]
func (h *CustomerLevelHandler) Deactivate(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	levelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid customer level ID format")
		return
	}

	level, err := h.levelService.Deactivate(c.Request.Context(), tenantID, levelID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, level)
}

// InitializeDefaultLevels godoc
//
//	@ID				initializeDefaultLevelsCustomerLevel
//	@Summary		Initialize default customer levels
//	@Description	Create the default customer levels for a tenant
//	@Tags			customer-levels
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Success		200			{object}	APIResponse[map[string]string]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/partner/customer-levels/initialize [post]
func (h *CustomerLevelHandler) InitializeDefaultLevels(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	err = h.levelService.InitializeDefaultLevels(c.Request.Context(), tenantID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, map[string]string{"message": "Default customer levels initialized successfully"})
}
