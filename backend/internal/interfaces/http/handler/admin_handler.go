package handler

import (
	"time"

	appIdentity "github.com/erp/backend/internal/application/identity"
	"github.com/erp/backend/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminHandler handles all admin HTTP requests for tenant management
// All endpoints require SuperAdminMiddleware
type AdminHandler struct {
	BaseHandler
	adminTenantService *appIdentity.AdminTenantService
	tenantStatsService *appIdentity.TenantStatsService
	auditService       *appIdentity.AuditService
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(
	adminTenantService *appIdentity.AdminTenantService,
	tenantStatsService *appIdentity.TenantStatsService,
	auditService *appIdentity.AuditService,
) *AdminHandler {
	return &AdminHandler{
		adminTenantService: adminTenantService,
		tenantStatsService: tenantStatsService,
		auditService:       auditService,
	}
}

// getAdminAuditContext extracts audit context from the request
func (h *AdminHandler) getAdminAuditContext(c *gin.Context) appIdentity.AuditContext {
	userID, _ := getUserID(c)
	return appIdentity.AuditContext{
		AdminUserID: userID,
		IPAddress:   c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
	}
}

// ============================================================================
// Tenant CRUD Endpoints
// ============================================================================

// ListTenants godoc
// @ID          adminListTenants
// @Summary     List all tenants
// @Description Get a paginated list of all tenants with filtering options
// @Tags        admin-tenants
// @Produce     json
// @Param       page          query    int     false  "Page number" default(1)
// @Param       page_size     query    int     false  "Items per page" default(20) maximum(100)
// @Param       search        query    string  false  "Search by name or code"
// @Param       status        query    string  false  "Filter by status" Enums(active, inactive, suspended, trial)
// @Param       plan          query    string  false  "Filter by plan" Enums(free, basic, pro, enterprise)
// @Param       order_by      query    string  false  "Sort by field" Enums(name, code, status, plan, created_at)
// @Param       order_dir     query    string  false  "Sort direction" Enums(asc, desc)
// @Success     200           {object} APIResponse[AdminTenantListResponse]
// @Failure     400           {object} ErrorResponse
// @Failure     401           {object} ErrorResponse
// @Failure     403           {object} ErrorResponse
// @Failure     500           {object} ErrorResponse
// @Security    BearerAuth
// @Router      /admin/tenants [get]
func (h *AdminHandler) ListTenants(c *gin.Context) {
	var query AdminTenantListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		h.BadRequest(c, "Invalid query parameters: "+err.Error())
		return
	}

	// Build filter input
	input := appIdentity.AdminTenantFilterInput{
		Page:     query.Page,
		PageSize: query.PageSize,
		OrderBy:  query.OrderBy,
		OrderDir: query.OrderDir,
		Search:   query.Search,
	}
	if query.Status != "" {
		input.Status = &query.Status
	}
	if query.Plan != "" {
		input.Plan = &query.Plan
	}

	// Call service
	result, err := h.adminTenantService.ListTenants(c.Request.Context(), input)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	// Build response
	response := AdminTenantListResponse{
		Tenants:    make([]AdminTenantResponse, len(result.Tenants)),
		Total:      result.Total,
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalPages: result.TotalPages,
	}

	for i, tenant := range result.Tenants {
		response.Tenants[i] = toAdminTenantResponse(&tenant)
	}

	h.Success(c, response)
}

// CreateTenant godoc
// @ID          adminCreateTenant
// @Summary     Create a new tenant
// @Description Create a new tenant in the system
// @Tags        admin-tenants
// @Accept      json
// @Produce     json
// @Param       request body     AdminCreateTenantRequest true "Tenant creation request"
// @Success     201     {object} APIResponse[AdminTenantResponse]
// @Failure     400     {object} ErrorResponse
// @Failure     401     {object} ErrorResponse
// @Failure     403     {object} ErrorResponse
// @Failure     409     {object} ErrorResponse
// @Failure     422     {object} ErrorResponse
// @Failure     500     {object} ErrorResponse
// @Security    BearerAuth
// @Router      /admin/tenants [post]
func (h *AdminHandler) CreateTenant(c *gin.Context) {
	var req AdminCreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	// Build input
	input := appIdentity.AdminCreateTenantInput{
		Code:         req.Code,
		Name:         req.Name,
		ShortName:    req.ShortName,
		ContactName:  req.ContactName,
		ContactPhone: req.ContactPhone,
		ContactEmail: req.ContactEmail,
		Address:      req.Address,
		Plan:         req.Plan,
		TrialDays:    req.TrialDays,
		Notes:        req.Notes,
	}

	// Get audit context
	auditCtx := h.getAdminAuditContext(c)

	// Call service
	result, err := h.adminTenantService.CreateTenant(c.Request.Context(), input, auditCtx)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Created(c, toAdminTenantResponse(result))
}

// GetTenant godoc
// @ID          adminGetTenant
// @Summary     Get tenant details
// @Description Get detailed information about a specific tenant
// @Tags        admin-tenants
// @Produce     json
// @Param       id   path     string true "Tenant ID" format(uuid)
// @Success     200  {object} APIResponse[AdminTenantResponse]
// @Failure     400  {object} ErrorResponse
// @Failure     401  {object} ErrorResponse
// @Failure     403  {object} ErrorResponse
// @Failure     404  {object} ErrorResponse
// @Failure     500  {object} ErrorResponse
// @Security    BearerAuth
// @Router      /admin/tenants/{id} [get]
func (h *AdminHandler) GetTenant(c *gin.Context) {
	// Parse tenant ID
	tenantIDStr := c.Param("id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID format")
		return
	}

	// Call service
	result, err := h.adminTenantService.GetTenant(c.Request.Context(), tenantID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toAdminTenantResponse(result))
}

// UpdateTenant godoc
// @ID          adminUpdateTenant
// @Summary     Update tenant information
// @Description Update a tenant's basic information
// @Tags        admin-tenants
// @Accept      json
// @Produce     json
// @Param       id      path     string                   true "Tenant ID" format(uuid)
// @Param       request body     AdminUpdateTenantRequest true "Tenant update request"
// @Success     200     {object} APIResponse[AdminTenantResponse]
// @Failure     400     {object} ErrorResponse
// @Failure     401     {object} ErrorResponse
// @Failure     403     {object} ErrorResponse
// @Failure     404     {object} ErrorResponse
// @Failure     422     {object} ErrorResponse
// @Failure     500     {object} ErrorResponse
// @Security    BearerAuth
// @Router      /admin/tenants/{id} [put]
func (h *AdminHandler) UpdateTenant(c *gin.Context) {
	// Parse tenant ID
	tenantIDStr := c.Param("id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID format")
		return
	}

	// Parse request body
	var req AdminUpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	// Build input
	input := appIdentity.AdminUpdateTenantInput{
		Name:         req.Name,
		ShortName:    req.ShortName,
		ContactName:  req.ContactName,
		ContactPhone: req.ContactPhone,
		ContactEmail: req.ContactEmail,
		Address:      req.Address,
		Notes:        req.Notes,
	}

	// Get audit context
	auditCtx := h.getAdminAuditContext(c)

	// Call service
	result, err := h.adminTenantService.UpdateTenant(c.Request.Context(), tenantID, input, auditCtx)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toAdminTenantResponse(result))
}

// DeleteTenant godoc
// @ID          adminDeleteTenant
// @Summary     Delete a tenant
// @Description Soft delete a tenant (marks as inactive, data preserved)
// @Tags        admin-tenants
// @Produce     json
// @Param       id   path     string true "Tenant ID" format(uuid)
// @Success     200  {object} APIResponse[MessageResponse]
// @Failure     400  {object} ErrorResponse
// @Failure     401  {object} ErrorResponse
// @Failure     403  {object} ErrorResponse
// @Failure     404  {object} ErrorResponse
// @Failure     422  {object} ErrorResponse
// @Failure     500  {object} ErrorResponse
// @Security    BearerAuth
// @Router      /admin/tenants/{id} [delete]
func (h *AdminHandler) DeleteTenant(c *gin.Context) {
	// Parse tenant ID
	tenantIDStr := c.Param("id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID format")
		return
	}

	// Get audit context
	auditCtx := h.getAdminAuditContext(c)

	// Call service
	if err := h.adminTenantService.DeleteTenant(c.Request.Context(), tenantID, auditCtx); err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, MessageResponse{Message: "Tenant deleted successfully"})
}

// ============================================================================
// Tenant Status Management Endpoints
// ============================================================================

// SuspendTenant godoc
// @ID          adminSuspendTenant
// @Summary     Suspend a tenant
// @Description Suspend a tenant with optional reason and scheduled reactivation
// @Tags        admin-tenants
// @Accept      json
// @Produce     json
// @Param       id      path     string                true "Tenant ID" format(uuid)
// @Param       request body     SuspendTenantRequest  false "Suspension details"
// @Success     200     {object} APIResponse[AdminTenantResponse]
// @Failure     400     {object} ErrorResponse
// @Failure     401     {object} ErrorResponse
// @Failure     403     {object} ErrorResponse
// @Failure     404     {object} ErrorResponse
// @Failure     422     {object} ErrorResponse
// @Failure     500     {object} ErrorResponse
// @Security    BearerAuth
// @Router      /admin/tenants/{id}/suspend [post]
func (h *AdminHandler) SuspendTenant(c *gin.Context) {
	// Parse tenant ID
	tenantIDStr := c.Param("id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID format")
		return
	}

	// Parse optional request body
	var req SuspendTenantRequest
	_ = c.ShouldBindJSON(&req) // Ignore error - body is optional

	// Build input
	input := appIdentity.SuspendTenantInput{
		TenantID:              tenantID,
		Reason:                req.Reason,
		ScheduledReactivateAt: req.ScheduledReactivateAt,
	}

	// Get audit context
	auditCtx := h.getAdminAuditContext(c)

	// Call service
	result, err := h.adminTenantService.SuspendTenantWithReason(c.Request.Context(), input, auditCtx)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toAdminTenantResponse(result))
}

// ActivateTenant godoc
// @ID          adminActivateTenant
// @Summary     Activate a tenant
// @Description Activate a suspended or inactive tenant
// @Tags        admin-tenants
// @Produce     json
// @Param       id   path     string true "Tenant ID" format(uuid)
// @Success     200  {object} APIResponse[AdminTenantResponse]
// @Failure     400  {object} ErrorResponse
// @Failure     401  {object} ErrorResponse
// @Failure     403  {object} ErrorResponse
// @Failure     404  {object} ErrorResponse
// @Failure     422  {object} ErrorResponse
// @Failure     500  {object} ErrorResponse
// @Security    BearerAuth
// @Router      /admin/tenants/{id}/activate [post]
func (h *AdminHandler) ActivateTenant(c *gin.Context) {
	// Parse tenant ID
	tenantIDStr := c.Param("id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID format")
		return
	}

	// Get audit context
	auditCtx := h.getAdminAuditContext(c)

	// Call service
	result, err := h.adminTenantService.ActivateTenant(c.Request.Context(), tenantID, auditCtx)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toAdminTenantResponse(result))
}

// ============================================================================
// Statistics Endpoints
// ============================================================================

// GetTenantStats godoc
// @ID          adminGetTenantStats
// @Summary     Get tenant usage statistics
// @Description Get detailed usage statistics for a specific tenant
// @Tags        admin-tenants
// @Produce     json
// @Param       id   path     string true "Tenant ID" format(uuid)
// @Success     200  {object} APIResponse[TenantUsageStatsResponse]
// @Failure     400  {object} ErrorResponse
// @Failure     401  {object} ErrorResponse
// @Failure     403  {object} ErrorResponse
// @Failure     404  {object} ErrorResponse
// @Failure     500  {object} ErrorResponse
// @Security    BearerAuth
// @Router      /admin/tenants/{id}/stats [get]
func (h *AdminHandler) GetTenantStats(c *gin.Context) {
	// Parse tenant ID
	tenantIDStr := c.Param("id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID format")
		return
	}

	// Call service
	result, err := h.tenantStatsService.GetTenantStats(c.Request.Context(), tenantID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	// Build response
	response := TenantUsageStatsResponse{
		TenantID:       result.TenantID,
		TenantName:     result.TenantName,
		TenantCode:     result.TenantCode,
		Plan:           result.Plan,
		Status:         result.Status,
		UserCount:      result.UserCount,
		ProductCount:   result.ProductCount,
		OrderCount:     result.OrderCount,
		WarehouseCount: result.WarehouseCount,
		StorageUsage:   result.StorageUsage,
		APICallCount:   result.APICallCount,
		MaxUsers:       result.MaxUsers,
		MaxProducts:    result.MaxProducts,
		MaxWarehouses:  result.MaxWarehouses,
		LastUpdated:    time.Unix(result.LastUpdated, 0),
	}

	h.Success(c, response)
}

// GetPlatformStats godoc
// @ID          adminGetPlatformStats
// @Summary     Get platform-wide statistics
// @Description Get aggregated statistics across all tenants
// @Tags        admin-stats
// @Produce     json
// @Success     200  {object} APIResponse[PlatformStatsResponse]
// @Failure     401  {object} ErrorResponse
// @Failure     403  {object} ErrorResponse
// @Failure     500  {object} ErrorResponse
// @Security    BearerAuth
// @Router      /admin/stats [get]
func (h *AdminHandler) GetPlatformStats(c *gin.Context) {
	// Call service
	result, err := h.tenantStatsService.GetPlatformStats(c.Request.Context())
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	// Build response
	response := PlatformStatsResponse{
		TotalTenants:      result.TotalTenants,
		ActiveTenants:     result.ActiveTenants,
		TrialTenants:      result.TrialTenants,
		SuspendedTenants:  result.SuspendedTenants,
		InactiveTenants:   result.InactiveTenants,
		TenantsByPlan:     result.TenantsByPlan,
		TotalUsers:        result.TotalUsers,
		TotalProducts:     result.TotalProducts,
		TotalOrders:       result.TotalOrders,
		TotalStorageUsage: result.TotalStorageUsage,
		LastUpdated:       time.Unix(result.LastUpdated, 0),
	}

	h.Success(c, response)
}

// GetTenantGrowth godoc
// @ID          adminGetTenantGrowth
// @Summary     Get tenant growth trend
// @Description Get tenant growth trend data for charts
// @Tags        admin-stats
// @Produce     json
// @Param       period query    string false "Period type" Enums(daily, weekly, monthly) default(daily)
// @Param       days   query    int    false "Number of days to look back" default(30) maximum(365)
// @Success     200    {object} APIResponse[TenantGrowthTrendResponse]
// @Failure     400    {object} ErrorResponse
// @Failure     401    {object} ErrorResponse
// @Failure     403    {object} ErrorResponse
// @Failure     500    {object} ErrorResponse
// @Security    BearerAuth
// @Router      /admin/stats/growth [get]
func (h *AdminHandler) GetTenantGrowth(c *gin.Context) {
	var query TenantGrowthQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		h.BadRequest(c, "Invalid query parameters: "+err.Error())
		return
	}

	// Set defaults
	if query.Period == "" {
		query.Period = "daily"
	}
	if query.Days == 0 {
		query.Days = 30
	}

	// Build input
	input := appIdentity.TenantGrowthInput{
		Period: query.Period,
		Days:   query.Days,
	}

	// Call service
	result, err := h.tenantStatsService.GetTenantGrowth(c.Request.Context(), input)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	// Build response
	dataPoints := make([]TenantGrowthPointResponse, len(result.DataPoints))
	for i, point := range result.DataPoints {
		dataPoints[i] = TenantGrowthPointResponse{
			Date:         point.Date,
			TotalTenants: point.TotalTenants,
			NewTenants:   point.NewTenants,
			ChurnedCount: point.ChurnedCount,
		}
	}

	response := TenantGrowthTrendResponse{
		Period:     result.Period,
		StartDate:  result.StartDate,
		EndDate:    result.EndDate,
		DataPoints: dataPoints,
	}

	h.Success(c, response)
}

// ============================================================================
// Audit Log Endpoints
// ============================================================================

// ListAuditLogs godoc
// @ID          adminListAuditLogs
// @Summary     List audit logs
// @Description Get a paginated list of admin audit logs
// @Tags        admin-audit
// @Produce     json
// @Param       page          query    int     false  "Page number" default(1)
// @Param       page_size     query    int     false  "Items per page" default(20) maximum(100)
// @Param       admin_user_id query    string  false  "Filter by admin user ID" format(uuid)
// @Param       action        query    string  false  "Filter by action" Enums(tenant_create, tenant_update, tenant_delete, tenant_suspend, tenant_activate, subscription_change, quota_update)
// @Param       target_type   query    string  false  "Filter by target type" Enums(tenant, subscription, quota)
// @Param       target_id     query    string  false  "Filter by target ID" format(uuid)
// @Param       start_time    query    string  false  "Filter by start time" format(date-time)
// @Param       end_time      query    string  false  "Filter by end time" format(date-time)
// @Param       sort_by       query    string  false  "Sort by field" Enums(created_at, action, target_type)
// @Param       sort_order    query    string  false  "Sort order" Enums(asc, desc)
// @Success     200           {object} APIResponse[AdminAuditLogListResponse]
// @Failure     400           {object} ErrorResponse
// @Failure     401           {object} ErrorResponse
// @Failure     403           {object} ErrorResponse
// @Failure     500           {object} ErrorResponse
// @Security    BearerAuth
// @Router      /admin/audit-logs [get]
func (h *AdminHandler) ListAuditLogs(c *gin.Context) {
	var query AuditLogListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		h.BadRequest(c, "Invalid query parameters: "+err.Error())
		return
	}

	// Build filter input
	input := appIdentity.AuditLogFilterInput{
		Page:      query.Page,
		PageSize:  query.PageSize,
		IPAddress: query.IPAddress,
		SortBy:    query.SortBy,
		SortOrder: query.SortOrder,
	}

	if query.AdminUserID != "" {
		adminUserID, err := uuid.Parse(query.AdminUserID)
		if err != nil {
			h.BadRequest(c, "Invalid admin_user_id format")
			return
		}
		input.AdminUserID = &adminUserID
	}

	if query.Action != "" {
		input.Action = &query.Action
	}

	if query.TargetType != "" {
		input.TargetType = &query.TargetType
	}

	if query.TargetID != "" {
		targetID, err := uuid.Parse(query.TargetID)
		if err != nil {
			h.BadRequest(c, "Invalid target_id format")
			return
		}
		input.TargetID = &targetID
	}

	if query.StartTime != nil {
		input.StartTime = query.StartTime
	}

	if query.EndTime != nil {
		input.EndTime = query.EndTime
	}

	// Call service
	result, err := h.auditService.List(c.Request.Context(), input)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	// Build response
	response := AdminAuditLogListResponse{
		Logs:       make([]AdminAuditLogResponse, len(result.Logs)),
		Total:      result.Total,
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalPages: result.TotalPages,
	}

	for i, log := range result.Logs {
		response.Logs[i] = AdminAuditLogResponse{
			ID:          log.ID,
			AdminUserID: log.AdminUserID,
			Action:      log.Action,
			TargetType:  log.TargetType,
			TargetID:    log.TargetID,
			OldValue:    log.OldValue,
			NewValue:    log.NewValue,
			IPAddress:   log.IPAddress,
			UserAgent:   log.UserAgent,
			CreatedAt:   log.CreatedAt,
		}
	}

	h.Success(c, response)
}

// ============================================================================
// Route Registration
// ============================================================================

// RegisterAdminRoutes registers all admin routes
// All routes are protected by SuperAdminMiddleware
func RegisterAdminRoutes(router *gin.RouterGroup, handler *AdminHandler) {
	// Apply SuperAdminMiddleware to all admin routes
	admin := router.Group("/admin")
	admin.Use(middleware.SuperAdminMiddleware())

	// Tenant CRUD
	tenants := admin.Group("/tenants")
	tenants.GET("", handler.ListTenants)
	tenants.POST("", handler.CreateTenant)
	tenants.GET("/:id", handler.GetTenant)
	tenants.PUT("/:id", handler.UpdateTenant)
	tenants.DELETE("/:id", handler.DeleteTenant)

	// Tenant status management
	tenants.POST("/:id/suspend", handler.SuspendTenant)
	tenants.POST("/:id/activate", handler.ActivateTenant)

	// Tenant subscription management
	tenants.PUT("/:id/plan", handler.ChangePlan)
	tenants.PUT("/:id/quota", handler.UpdateQuota)
	tenants.GET("/:id/subscription-history", handler.GetSubscriptionHistory)
	tenants.DELETE("/:id/scheduled-plan", handler.CancelScheduledPlanChange)

	// Tenant statistics
	tenants.GET("/:id/stats", handler.GetTenantStats)

	// Platform statistics
	admin.GET("/stats", handler.GetPlatformStats)
	admin.GET("/stats/growth", handler.GetTenantGrowth)

	// Audit logs
	admin.GET("/audit-logs", handler.ListAuditLogs)
}

// ============================================================================
// Subscription Management Methods
// ============================================================================

// ChangePlan godoc
// @ID          adminChangeTenantPlanV2
// @Summary     Change tenant subscription plan
// @Description Change a tenant's subscription plan. Upgrades take effect immediately, downgrades are scheduled for cycle end.
// @Tags        admin-tenants
// @Accept      json
// @Produce     json
// @Param       id      path     string            true  "Tenant ID" format(uuid)
// @Param       request body     ChangePlanRequest true  "Plan change request"
// @Success     200     {object} APIResponse[ChangePlanResponse]
// @Failure     400     {object} ErrorResponse
// @Failure     401     {object} ErrorResponse
// @Failure     403     {object} ErrorResponse
// @Failure     404     {object} ErrorResponse
// @Failure     422     {object} ErrorResponse
// @Failure     500     {object} ErrorResponse
// @Security    BearerAuth
// @Router      /admin/tenants/{id}/plan [put]
func (h *AdminHandler) ChangePlan(c *gin.Context) {
	// Parse tenant ID
	tenantIDStr := c.Param("id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID format")
		return
	}

	// Parse request body
	var req ChangePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	// Build input
	input := appIdentity.ChangePlanInput{
		TenantID: tenantID,
		NewPlan:  req.Plan,
		Reason:   req.Reason,
	}

	// Get audit context
	auditCtx := h.getAdminAuditContext(c)

	// Call service
	result, err := h.adminTenantService.ChangePlan(c.Request.Context(), input, auditCtx)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	// Build response
	response := ChangePlanResponse{
		Tenant:       toAdminTenantResponse(result.Tenant),
		ChangeType:   result.ChangeType,
		EffectiveAt:  result.EffectiveAt,
		PreviousPlan: result.PreviousPlan,
		NewPlan:      result.NewPlan,
	}

	h.Success(c, response)
}

// UpdateQuota godoc
// @ID          adminUpdateTenantQuotaV2
// @Summary     Update tenant quota limits
// @Description Update a tenant's quota limits. Changes take effect immediately.
// @Tags        admin-tenants
// @Accept      json
// @Produce     json
// @Param       id      path     string             true  "Tenant ID" format(uuid)
// @Param       request body     UpdateQuotaRequest true  "Quota update request"
// @Success     200     {object} APIResponse[AdminTenantResponse]
// @Failure     400     {object} ErrorResponse
// @Failure     401     {object} ErrorResponse
// @Failure     403     {object} ErrorResponse
// @Failure     404     {object} ErrorResponse
// @Failure     422     {object} ErrorResponse
// @Failure     500     {object} ErrorResponse
// @Security    BearerAuth
// @Router      /admin/tenants/{id}/quota [put]
func (h *AdminHandler) UpdateQuota(c *gin.Context) {
	// Parse tenant ID
	tenantIDStr := c.Param("id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID format")
		return
	}

	// Parse request body
	var req UpdateQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	// Build input
	input := appIdentity.UpdateQuotaInput{
		TenantID:      tenantID,
		MaxUsers:      req.MaxUsers,
		MaxWarehouses: req.MaxWarehouses,
		MaxProducts:   req.MaxProducts,
		Reason:        req.Reason,
	}

	// Get audit context
	auditCtx := h.getAdminAuditContext(c)

	// Call service
	result, err := h.adminTenantService.UpdateQuota(c.Request.Context(), input, auditCtx)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toAdminTenantResponse(result))
}

// GetSubscriptionHistory godoc
// @ID          adminGetSubscriptionHistoryV2
// @Summary     Get tenant subscription history
// @Description Retrieve subscription change history for a tenant
// @Tags        admin-tenants
// @Produce     json
// @Param       id          path     string  true   "Tenant ID" format(uuid)
// @Param       page        query    int     false  "Page number" default(1)
// @Param       page_size   query    int     false  "Items per page" default(20) maximum(100)
// @Param       change_type query    string  false  "Filter by change type" Enums(plan_upgrade, plan_downgrade, quota_update)
// @Success     200         {object} APIResponse[SubscriptionHistoryListResponse]
// @Failure     400         {object} ErrorResponse
// @Failure     401         {object} ErrorResponse
// @Failure     403         {object} ErrorResponse
// @Failure     404         {object} ErrorResponse
// @Failure     500         {object} ErrorResponse
// @Security    BearerAuth
// @Router      /admin/tenants/{id}/subscription-history [get]
func (h *AdminHandler) GetSubscriptionHistory(c *gin.Context) {
	// Parse tenant ID
	tenantIDStr := c.Param("id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID format")
		return
	}

	// Parse query parameters
	var query SubscriptionHistoryQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		h.BadRequest(c, "Invalid query parameters: "+err.Error())
		return
	}

	// Build filter input
	input := appIdentity.SubscriptionHistoryFilterInput{
		Page:     query.Page,
		PageSize: query.PageSize,
	}
	if query.ChangeType != "" {
		input.ChangeType = &query.ChangeType
	}

	// Call service
	result, err := h.adminTenantService.GetSubscriptionHistory(c.Request.Context(), tenantID, input)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	// Build response
	response := SubscriptionHistoryListResponse{
		History:    make([]SubscriptionHistoryResponse, len(result.History)),
		Total:      result.Total,
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalPages: result.TotalPages,
	}

	for i, hist := range result.History {
		response.History[i] = toSubscriptionHistoryResponse(&hist)
	}

	h.Success(c, response)
}

// CancelScheduledPlanChange godoc
// @ID          adminCancelScheduledPlanChangeV2
// @Summary     Cancel scheduled plan change
// @Description Cancel a pending plan downgrade for a tenant
// @Tags        admin-tenants
// @Produce     json
// @Param       id   path     string  true  "Tenant ID" format(uuid)
// @Success     200  {object} APIResponse[AdminTenantResponse]
// @Failure     400  {object} ErrorResponse
// @Failure     401  {object} ErrorResponse
// @Failure     403  {object} ErrorResponse
// @Failure     404  {object} ErrorResponse
// @Failure     422  {object} ErrorResponse
// @Failure     500  {object} ErrorResponse
// @Security    BearerAuth
// @Router      /admin/tenants/{id}/scheduled-plan [delete]
func (h *AdminHandler) CancelScheduledPlanChange(c *gin.Context) {
	// Parse tenant ID
	tenantIDStr := c.Param("id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID format")
		return
	}

	// Get audit context
	auditCtx := h.getAdminAuditContext(c)

	// Call service
	result, err := h.adminTenantService.CancelScheduledPlanChange(c.Request.Context(), tenantID, auditCtx)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toAdminTenantResponse(result))
}

// Helper function to check if user is super admin
func isAdminSuperAdmin(c *gin.Context) bool {
	return middleware.IsSuperAdmin(c)
}
