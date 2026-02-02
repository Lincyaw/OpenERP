package handler

import (
	appIdentity "github.com/erp/backend/internal/application/identity"
	"github.com/erp/backend/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminTenantHandler handles admin tenant management HTTP requests.
// This handler manages subscription and quota operations.
//
// All endpoints require SuperAdmin privileges (is_super_admin: true in JWT).
type AdminTenantHandler struct {
	BaseHandler
	adminTenantService *appIdentity.AdminTenantService
}

// NewAdminTenantHandler creates a new admin tenant handler
func NewAdminTenantHandler(adminTenantService *appIdentity.AdminTenantService) *AdminTenantHandler {
	return &AdminTenantHandler{
		adminTenantService: adminTenantService,
	}
}

// getAuditContext extracts audit context from the request
func (h *AdminTenantHandler) getAuditContext(c *gin.Context) appIdentity.AuditContext {
	userID, _ := getUserID(c)
	return appIdentity.AuditContext{
		AdminUserID: userID,
		IPAddress:   c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
	}
}

// ChangePlan godoc
// @ID          adminChangeTenantPlan
// @Summary     Change tenant subscription plan
// @Description Change a tenant's subscription plan. Upgrades take effect immediately, downgrades are scheduled for cycle end.
// @Tags        admin-tenants
// @Accept      json
// @Produce     json
// @Param       id      path     string                    true  "Tenant ID" format(uuid)
// @Param       request body     ChangePlanRequest         true  "Plan change request"
// @Success     200     {object} APIResponse[ChangePlanResponse]
// @Failure     400     {object} ErrorResponse
// @Failure     401     {object} ErrorResponse
// @Failure     403     {object} ErrorResponse
// @Failure     404     {object} ErrorResponse
// @Failure     422     {object} ErrorResponse
// @Failure     500     {object} ErrorResponse
// @Security    BearerAuth
// @Router      /admin/tenants/{id}/plan [put]
func (h *AdminTenantHandler) ChangePlan(c *gin.Context) {
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
	auditCtx := h.getAuditContext(c)

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
// @ID          adminUpdateTenantQuota
// @Summary     Update tenant quota limits
// @Description Update a tenant's quota limits. Changes take effect immediately.
// @Tags        admin-tenants
// @Accept      json
// @Produce     json
// @Param       id      path     string                    true  "Tenant ID" format(uuid)
// @Param       request body     UpdateQuotaRequest        true  "Quota update request"
// @Success     200     {object} APIResponse[AdminTenantResponse]
// @Failure     400     {object} ErrorResponse
// @Failure     401     {object} ErrorResponse
// @Failure     403     {object} ErrorResponse
// @Failure     404     {object} ErrorResponse
// @Failure     422     {object} ErrorResponse
// @Failure     500     {object} ErrorResponse
// @Security    BearerAuth
// @Router      /admin/tenants/{id}/quota [put]
func (h *AdminTenantHandler) UpdateQuota(c *gin.Context) {
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
	auditCtx := h.getAuditContext(c)

	// Call service
	result, err := h.adminTenantService.UpdateQuota(c.Request.Context(), input, auditCtx)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toAdminTenantResponse(result))
}

// GetSubscriptionHistory godoc
// @ID          adminGetSubscriptionHistory
// @Summary     Get tenant subscription history
// @Description Retrieve subscription change history for a tenant
// @Tags        admin-tenants
// @Produce     json
// @Param       id        path     string  true   "Tenant ID" format(uuid)
// @Param       page      query    int     false  "Page number" default(1)
// @Param       page_size query    int     false  "Items per page" default(20) maximum(100)
// @Param       change_type query  string  false  "Filter by change type (plan_upgrade, plan_downgrade, quota_update)"
// @Success     200       {object} APIResponse[SubscriptionHistoryListResponse]
// @Failure     400       {object} ErrorResponse
// @Failure     401       {object} ErrorResponse
// @Failure     403       {object} ErrorResponse
// @Failure     404       {object} ErrorResponse
// @Failure     500       {object} ErrorResponse
// @Security    BearerAuth
// @Router      /admin/tenants/{id}/subscription-history [get]
func (h *AdminTenantHandler) GetSubscriptionHistory(c *gin.Context) {
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

	for i, h := range result.History {
		response.History[i] = toSubscriptionHistoryResponse(&h)
	}

	h.Success(c, response)
}

// CancelScheduledPlanChange godoc
// @ID          adminCancelScheduledPlanChange
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
func (h *AdminTenantHandler) CancelScheduledPlanChange(c *gin.Context) {
	// Parse tenant ID
	tenantIDStr := c.Param("id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID format")
		return
	}

	// Get audit context
	auditCtx := h.getAuditContext(c)

	// Call service
	result, err := h.adminTenantService.CancelScheduledPlanChange(c.Request.Context(), tenantID, auditCtx)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toAdminTenantResponse(result))
}

// RegisterAdminTenantRoutes registers admin tenant routes
func RegisterAdminTenantRoutes(router *gin.RouterGroup, handler *AdminTenantHandler) {
	// All routes require super admin middleware (should be applied at group level)
	router.PUT("/:id/plan", handler.ChangePlan)
	router.PUT("/:id/quota", handler.UpdateQuota)
	router.GET("/:id/subscription-history", handler.GetSubscriptionHistory)
	router.DELETE("/:id/scheduled-plan", handler.CancelScheduledPlanChange)
}

// Helper function to check if user is super admin
func isSuperAdmin(c *gin.Context) bool {
	return middleware.IsSuperAdmin(c)
}
