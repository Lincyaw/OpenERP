package handler

import (
	"github.com/erp/backend/internal/application/identity"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TenantHandler handles tenant management HTTP requests
type TenantHandler struct {
	BaseHandler
	tenantService *identity.TenantService
}

// NewTenantHandler creates a new tenant handler
func NewTenantHandler(tenantService *identity.TenantService) *TenantHandler {
	return &TenantHandler{
		tenantService: tenantService,
	}
}

// Create godoc
// @Summary      Create a new tenant
// @Description  Create a new tenant in the system
// @Tags         tenants
// @Accept       json
// @Produce      json
// @Param        request body CreateTenantRequest true "Tenant creation request"
// @Success      201 {object} dto.Response{data=TenantResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      403 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /identity/tenants [post]
func (h *TenantHandler) Create(c *gin.Context) {
	var req CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body")
		return
	}

	input := identity.CreateTenantInput{
		Code:         req.Code,
		Name:         req.Name,
		ShortName:    req.ShortName,
		ContactName:  req.ContactName,
		ContactPhone: req.ContactPhone,
		ContactEmail: req.ContactEmail,
		Address:      req.Address,
		LogoURL:      req.LogoURL,
		Domain:       req.Domain,
		Plan:         req.Plan,
		Notes:        req.Notes,
		TrialDays:    req.TrialDays,
	}

	tenant, err := h.tenantService.Create(c.Request.Context(), input)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Created(c, toTenantResponse(tenant))
}

// GetByID godoc
// @Summary      Get a tenant by ID
// @Description  Retrieve a tenant by its ID
// @Tags         tenants
// @Accept       json
// @Produce      json
// @Param        id path string true "Tenant ID" format(uuid)
// @Success      200 {object} dto.Response{data=TenantResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /identity/tenants/{id} [get]
func (h *TenantHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	tenant, err := h.tenantService.GetByID(c.Request.Context(), id)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toTenantResponse(tenant))
}

// GetByCode godoc
// @Summary      Get a tenant by code
// @Description  Retrieve a tenant by its unique code
// @Tags         tenants
// @Accept       json
// @Produce      json
// @Param        code path string true "Tenant code"
// @Success      200 {object} dto.Response{data=TenantResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /identity/tenants/code/{code} [get]
func (h *TenantHandler) GetByCode(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		h.BadRequest(c, "Tenant code is required")
		return
	}

	tenant, err := h.tenantService.GetByCode(c.Request.Context(), code)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toTenantResponse(tenant))
}

// List godoc
// @Summary      List tenants
// @Description  Get a paginated list of tenants
// @Tags         tenants
// @Accept       json
// @Produce      json
// @Param        keyword query string false "Search keyword"
// @Param        status query string false "Tenant status" Enums(active, inactive, suspended, trial)
// @Param        plan query string false "Subscription plan" Enums(free, basic, pro, enterprise)
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Items per page" default(20) maximum(100)
// @Param        sort_by query string false "Sort by field" Enums(code, name, status, plan, created_at, updated_at)
// @Param        sort_dir query string false "Sort direction" Enums(asc, desc)
// @Success      200 {object} dto.Response{data=TenantListResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /identity/tenants [get]
func (h *TenantHandler) List(c *gin.Context) {
	var query TenantListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		h.BadRequest(c, "Invalid query parameters")
		return
	}

	filter := identity.TenantFilter{
		Page:     query.Page,
		PageSize: query.PageSize,
		SortBy:   query.SortBy,
		SortDir:  query.SortDir,
		Keyword:  query.Keyword,
		Status:   query.Status,
		Plan:     query.Plan,
	}

	result, err := h.tenantService.List(c.Request.Context(), filter)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toTenantListResponse(result))
}

// Update godoc
// @Summary      Update a tenant
// @Description  Update a tenant's information
// @Tags         tenants
// @Accept       json
// @Produce      json
// @Param        id path string true "Tenant ID" format(uuid)
// @Param        request body UpdateTenantRequest true "Tenant update request"
// @Success      200 {object} dto.Response{data=TenantResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /identity/tenants/{id} [put]
func (h *TenantHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body")
		return
	}

	input := identity.UpdateTenantInput{
		ID:           id,
		Name:         req.Name,
		ShortName:    req.ShortName,
		ContactName:  req.ContactName,
		ContactPhone: req.ContactPhone,
		ContactEmail: req.ContactEmail,
		Address:      req.Address,
		LogoURL:      req.LogoURL,
		Domain:       req.Domain,
		Notes:        req.Notes,
	}

	tenant, err := h.tenantService.Update(c.Request.Context(), input)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toTenantResponse(tenant))
}

// UpdateConfig godoc
// @Summary      Update tenant configuration
// @Description  Update a tenant's configuration settings
// @Tags         tenants
// @Accept       json
// @Produce      json
// @Param        id path string true "Tenant ID" format(uuid)
// @Param        request body UpdateTenantConfigRequest true "Configuration update request"
// @Success      200 {object} dto.Response{data=TenantResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /identity/tenants/{id}/config [put]
func (h *TenantHandler) UpdateConfig(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req UpdateTenantConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body")
		return
	}

	input := identity.TenantConfigInput{
		MaxUsers:      req.MaxUsers,
		MaxWarehouses: req.MaxWarehouses,
		MaxProducts:   req.MaxProducts,
		CostStrategy:  req.CostStrategy,
		Currency:      req.Currency,
		Timezone:      req.Timezone,
		Locale:        req.Locale,
	}

	tenant, err := h.tenantService.UpdateConfig(c.Request.Context(), id, input)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toTenantResponse(tenant))
}

// SetPlan godoc
// @Summary      Set tenant plan
// @Description  Update a tenant's subscription plan
// @Tags         tenants
// @Accept       json
// @Produce      json
// @Param        id path string true "Tenant ID" format(uuid)
// @Param        request body SetTenantPlanRequest true "Plan update request"
// @Success      200 {object} dto.Response{data=TenantResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /identity/tenants/{id}/plan [put]
func (h *TenantHandler) SetPlan(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req SetTenantPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body")
		return
	}

	tenant, err := h.tenantService.SetPlan(c.Request.Context(), id, req.Plan)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toTenantResponse(tenant))
}

// Activate godoc
// @Summary      Activate a tenant
// @Description  Activate a tenant account
// @Tags         tenants
// @Accept       json
// @Produce      json
// @Param        id path string true "Tenant ID" format(uuid)
// @Success      200 {object} dto.Response{data=TenantResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /identity/tenants/{id}/activate [post]
func (h *TenantHandler) Activate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	tenant, err := h.tenantService.Activate(c.Request.Context(), id)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toTenantResponse(tenant))
}

// Deactivate godoc
// @Summary      Deactivate a tenant
// @Description  Deactivate a tenant account
// @Tags         tenants
// @Accept       json
// @Produce      json
// @Param        id path string true "Tenant ID" format(uuid)
// @Success      200 {object} dto.Response{data=TenantResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /identity/tenants/{id}/deactivate [post]
func (h *TenantHandler) Deactivate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	tenant, err := h.tenantService.Deactivate(c.Request.Context(), id)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toTenantResponse(tenant))
}

// Suspend godoc
// @Summary      Suspend a tenant
// @Description  Suspend a tenant account (e.g., due to payment issues)
// @Tags         tenants
// @Accept       json
// @Produce      json
// @Param        id path string true "Tenant ID" format(uuid)
// @Success      200 {object} dto.Response{data=TenantResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /identity/tenants/{id}/suspend [post]
func (h *TenantHandler) Suspend(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	tenant, err := h.tenantService.Suspend(c.Request.Context(), id)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toTenantResponse(tenant))
}

// Delete godoc
// @Summary      Delete a tenant
// @Description  Delete a tenant from the system (only inactive tenants can be deleted)
// @Tags         tenants
// @Accept       json
// @Produce      json
// @Param        id path string true "Tenant ID" format(uuid)
// @Success      200 {object} dto.Response
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /identity/tenants/{id} [delete]
func (h *TenantHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	if err := h.tenantService.Delete(c.Request.Context(), id); err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, dto.MessageResponse{Message: "Tenant deleted successfully"})
}

// GetStats godoc
// @Summary      Get tenant statistics
// @Description  Get statistics about tenants in the system
// @Tags         tenants
// @Accept       json
// @Produce      json
// @Success      200 {object} dto.Response{data=TenantStatsResponse}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /identity/tenants/stats [get]
func (h *TenantHandler) GetStats(c *gin.Context) {
	stats, err := h.tenantService.GetStats(c.Request.Context())
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, TenantStatsResponse{
		Total:     stats.Total,
		Active:    stats.Active,
		Trial:     stats.Trial,
		Inactive:  stats.Inactive,
		Suspended: stats.Suspended,
	})
}

// Count godoc
// @Summary      Get tenant count
// @Description  Get the total number of tenants
// @Tags         tenants
// @Accept       json
// @Produce      json
// @Success      200 {object} dto.Response{data=object{count=int64}}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /identity/tenants/stats/count [get]
func (h *TenantHandler) Count(c *gin.Context) {
	count, err := h.tenantService.Count(c.Request.Context())
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, gin.H{"count": count})
}

// Helper functions for response conversion

func toTenantResponse(tenant *identity.TenantDTO) *TenantResponse {
	return &TenantResponse{
		ID:           tenant.ID,
		Code:         tenant.Code,
		Name:         tenant.Name,
		ShortName:    tenant.ShortName,
		Status:       tenant.Status,
		Plan:         tenant.Plan,
		ContactName:  tenant.ContactName,
		ContactPhone: tenant.ContactPhone,
		ContactEmail: tenant.ContactEmail,
		Address:      tenant.Address,
		LogoURL:      tenant.LogoURL,
		Domain:       tenant.Domain,
		ExpiresAt:    tenant.ExpiresAt,
		TrialEndsAt:  tenant.TrialEndsAt,
		Config: TenantConfigResponse{
			MaxUsers:      tenant.Config.MaxUsers,
			MaxWarehouses: tenant.Config.MaxWarehouses,
			MaxProducts:   tenant.Config.MaxProducts,
			CostStrategy:  tenant.Config.CostStrategy,
			Currency:      tenant.Config.Currency,
			Timezone:      tenant.Config.Timezone,
			Locale:        tenant.Config.Locale,
		},
		Notes:     tenant.Notes,
		CreatedAt: tenant.CreatedAt,
		UpdatedAt: tenant.UpdatedAt,
	}
}

func toTenantListResponse(result *identity.TenantListResult) *TenantListResponse {
	tenants := make([]TenantResponse, len(result.Tenants))
	for i, tenant := range result.Tenants {
		tenants[i] = *toTenantResponse(&tenant)
	}

	return &TenantListResponse{
		Tenants:    tenants,
		Total:      result.Total,
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalPages: result.TotalPages,
	}
}
