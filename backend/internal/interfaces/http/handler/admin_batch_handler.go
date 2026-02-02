package handler

import (
	appIdentity "github.com/erp/backend/internal/application/identity"
	"github.com/gin-gonic/gin"
)

// ============================================================================
// Batch Suspend Endpoints
// ============================================================================

// PreviewBatchSuspend godoc
// @ID          adminPreviewBatchSuspend
// @Summary     Preview batch suspend operation
// @Description Preview the batch suspend operation before execution. Returns which tenants can be suspended and which will be skipped.
// @Description
// @Description **Authentication**: Requires SuperAdmin role (is_super_admin: true in JWT)
// @Description
// @Description **Preview shows**:
// @Description - Total number of tenants in the request
// @Description - How many can be processed
// @Description - How many will be skipped (with reasons)
// @Description - Warnings about potential issues
// @Description
// @Description **Skip reasons**:
// @Description - Tenant is already suspended
// @Description - Tenant is a system tenant (protected)
// @Description - Tenant not found
// @Description
// @Description **Tip**: Always preview before executing batch operations
// @Tags        admin-batch
// @Accept      json
// @Produce     json
// @Param       request body     BatchSuspendRequest true "Batch suspend preview request (max 100 tenants)"
// @Success     200     {object} APIResponse[BatchPreviewResponse] "Preview of batch operation"
// @Failure     400     {object} ErrorResponse "ERR_BAD_REQUEST: Invalid request"
// @Failure     401     {object} ErrorResponse "ERR_UNAUTHORIZED: Missing or invalid token"
// @Failure     403     {object} ErrorResponse "ERR_FORBIDDEN: Not a super admin"
// @Failure     422     {object} ErrorResponse "ERR_BUSINESS_RULE: Business rule violation"
// @Failure     500     {object} ErrorResponse "ERR_INTERNAL: Server error"
// @Security    BearerAuth
// @Router      /admin/batch/suspend/preview [post]
func (h *AdminHandler) PreviewBatchSuspend(c *gin.Context) {
	var req BatchSuspendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	tenantIDs, err := parseUUIDs(req.TenantIDs)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID format")
		return
	}

	input := appIdentity.BatchSuspendInput{
		TenantIDs:             tenantIDs,
		Reason:                req.Reason,
		ScheduledReactivateAt: req.ScheduledReactivateAt,
	}

	result, err := h.adminTenantService.PreviewBatchSuspend(c.Request.Context(), input)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toBatchPreviewResponse(result))
}

// BatchSuspend godoc
// @ID          adminBatchSuspend
// @Summary     Execute batch suspend operation
// @Description Suspend multiple tenants at once. All operations are recorded in audit logs.
// @Description
// @Description **Authentication**: Requires SuperAdmin role (is_super_admin: true in JWT)
// @Description
// @Description **Behavior**:
// @Description - Processes all valid tenants in the list
// @Description - Skips tenants that cannot be suspended
// @Description - Creates individual audit log entries for each tenant
// @Description - Optional reason is applied to all suspensions
// @Description - Optional scheduled_reactivate_at sets auto-reactivation for all
// @Description
// @Description **Limits**: Maximum 100 tenants per batch operation
// @Description
// @Description **Recommendation**: Use preview endpoint first to check impact
// @Tags        admin-batch
// @Accept      json
// @Produce     json
// @Param       request body     BatchSuspendRequest true "Batch suspend request (max 100 tenants)"
// @Success     200     {object} APIResponse[BatchOperationResponse] "Batch operation result"
// @Failure     400     {object} ErrorResponse "ERR_BAD_REQUEST: Invalid request"
// @Failure     401     {object} ErrorResponse "ERR_UNAUTHORIZED: Missing or invalid token"
// @Failure     403     {object} ErrorResponse "ERR_FORBIDDEN: Not a super admin"
// @Failure     422     {object} ErrorResponse "ERR_BUSINESS_RULE: Business rule violation"
// @Failure     500     {object} ErrorResponse "ERR_INTERNAL: Server error"
// @Security    BearerAuth
// @Router      /admin/batch/suspend [post]
func (h *AdminHandler) BatchSuspend(c *gin.Context) {
	var req BatchSuspendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	tenantIDs, err := parseUUIDs(req.TenantIDs)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID format")
		return
	}

	input := appIdentity.BatchSuspendInput{
		TenantIDs:             tenantIDs,
		Reason:                req.Reason,
		ScheduledReactivateAt: req.ScheduledReactivateAt,
	}

	auditCtx := h.getAdminAuditContext(c)
	result, err := h.adminTenantService.BatchSuspend(c.Request.Context(), input, auditCtx)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toBatchOperationResponse(result))
}

// ============================================================================
// Batch Activate Endpoints
// ============================================================================

// PreviewBatchActivate godoc
// @ID          adminPreviewBatchActivate
// @Summary     Preview batch activate operation
// @Description Preview the batch activate operation before execution. Returns which tenants can be activated and which will be skipped.
// @Description
// @Description **Authentication**: Requires SuperAdmin role (is_super_admin: true in JWT)
// @Description
// @Description **Skip reasons**:
// @Description - Tenant is already active
// @Description - Tenant not found
// @Tags        admin-batch
// @Accept      json
// @Produce     json
// @Param       request body     BatchActivateRequest true "Batch activate preview request (max 100 tenants)"
// @Success     200     {object} APIResponse[BatchPreviewResponse] "Preview of batch operation"
// @Failure     400     {object} ErrorResponse "ERR_BAD_REQUEST: Invalid request"
// @Failure     401     {object} ErrorResponse "ERR_UNAUTHORIZED: Missing or invalid token"
// @Failure     403     {object} ErrorResponse "ERR_FORBIDDEN: Not a super admin"
// @Failure     422     {object} ErrorResponse "ERR_BUSINESS_RULE: Business rule violation"
// @Failure     500     {object} ErrorResponse "ERR_INTERNAL: Server error"
// @Security    BearerAuth
// @Router      /admin/batch/activate/preview [post]
func (h *AdminHandler) PreviewBatchActivate(c *gin.Context) {
	var req BatchActivateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	tenantIDs, err := parseUUIDs(req.TenantIDs)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID format")
		return
	}

	input := appIdentity.BatchActivateInput{
		TenantIDs: tenantIDs,
	}

	result, err := h.adminTenantService.PreviewBatchActivate(c.Request.Context(), input)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toBatchPreviewResponse(result))
}

// BatchActivate godoc
// @ID          adminBatchActivate
// @Summary     Execute batch activate operation
// @Description Activate multiple tenants at once. All operations are recorded in audit logs.
// @Description
// @Description **Authentication**: Requires SuperAdmin role (is_super_admin: true in JWT)
// @Description
// @Description **Behavior**:
// @Description - Sets all valid tenants to "active" status
// @Description - Clears any scheduled reactivation dates
// @Description - Creates individual audit log entries for each tenant
// @Description
// @Description **Limits**: Maximum 100 tenants per batch operation
// @Tags        admin-batch
// @Accept      json
// @Produce     json
// @Param       request body     BatchActivateRequest true "Batch activate request (max 100 tenants)"
// @Success     200     {object} APIResponse[BatchOperationResponse] "Batch operation result"
// @Failure     400     {object} ErrorResponse "ERR_BAD_REQUEST: Invalid request"
// @Failure     401     {object} ErrorResponse "ERR_UNAUTHORIZED: Missing or invalid token"
// @Failure     403     {object} ErrorResponse "ERR_FORBIDDEN: Not a super admin"
// @Failure     422     {object} ErrorResponse "ERR_BUSINESS_RULE: Business rule violation"
// @Failure     500     {object} ErrorResponse "ERR_INTERNAL: Server error"
// @Security    BearerAuth
// @Router      /admin/batch/activate [post]
func (h *AdminHandler) BatchActivate(c *gin.Context) {
	var req BatchActivateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	tenantIDs, err := parseUUIDs(req.TenantIDs)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID format")
		return
	}

	input := appIdentity.BatchActivateInput{
		TenantIDs: tenantIDs,
	}

	auditCtx := h.getAdminAuditContext(c)
	result, err := h.adminTenantService.BatchActivate(c.Request.Context(), input, auditCtx)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toBatchOperationResponse(result))
}

// ============================================================================
// Batch Change Plan Endpoints
// ============================================================================

// PreviewBatchChangePlan godoc
// @ID          adminPreviewBatchChangePlan
// @Summary     Preview batch plan change operation
// @Description Preview the batch plan change operation before execution. Returns which tenants can have their plan changed and which will be skipped.
// @Description
// @Description **Authentication**: Requires SuperAdmin role (is_super_admin: true in JWT)
// @Description
// @Description **Skip reasons**:
// @Description - Tenant already on the target plan
// @Description - Tenant not found
// @Description
// @Description **Note**: Upgrades are immediate, downgrades are scheduled
// @Tags        admin-batch
// @Accept      json
// @Produce     json
// @Param       request body     BatchChangePlanRequest true "Batch plan change preview request (max 100 tenants)"
// @Success     200     {object} APIResponse[BatchPreviewResponse] "Preview of batch operation"
// @Failure     400     {object} ErrorResponse "ERR_BAD_REQUEST: Invalid request"
// @Failure     401     {object} ErrorResponse "ERR_UNAUTHORIZED: Missing or invalid token"
// @Failure     403     {object} ErrorResponse "ERR_FORBIDDEN: Not a super admin"
// @Failure     422     {object} ErrorResponse "ERR_BUSINESS_RULE: Invalid plan"
// @Failure     500     {object} ErrorResponse "ERR_INTERNAL: Server error"
// @Security    BearerAuth
// @Router      /admin/batch/change-plan/preview [post]
func (h *AdminHandler) PreviewBatchChangePlan(c *gin.Context) {
	var req BatchChangePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	tenantIDs, err := parseUUIDs(req.TenantIDs)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID format")
		return
	}

	input := appIdentity.BatchChangePlanInput{
		TenantIDs: tenantIDs,
		NewPlan:   req.NewPlan,
		Reason:    req.Reason,
	}

	result, err := h.adminTenantService.PreviewBatchChangePlan(c.Request.Context(), input)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toBatchPreviewResponse(result))
}

// BatchChangePlan godoc
// @ID          adminBatchChangePlan
// @Summary     Execute batch plan change operation
// @Description Change subscription plan for multiple tenants at once. Upgrades are immediate, downgrades are scheduled. All operations are recorded in audit logs.
// @Description
// @Description **Authentication**: Requires SuperAdmin role (is_super_admin: true in JWT)
// @Description
// @Description **Upgrade behavior**:
// @Description - Takes effect immediately for all tenants
// @Description - New quotas apply instantly
// @Description
// @Description **Downgrade behavior**:
// @Description - Scheduled for end of current billing cycle
// @Description - Each tenant's effective date may differ
// @Description
// @Description **Limits**: Maximum 100 tenants per batch operation
// @Tags        admin-batch
// @Accept      json
// @Produce     json
// @Param       request body     BatchChangePlanRequest true "Batch plan change request (max 100 tenants)"
// @Success     200     {object} APIResponse[BatchOperationResponse] "Batch operation result"
// @Failure     400     {object} ErrorResponse "ERR_BAD_REQUEST: Invalid request"
// @Failure     401     {object} ErrorResponse "ERR_UNAUTHORIZED: Missing or invalid token"
// @Failure     403     {object} ErrorResponse "ERR_FORBIDDEN: Not a super admin"
// @Failure     422     {object} ErrorResponse "ERR_BUSINESS_RULE: Invalid plan"
// @Failure     500     {object} ErrorResponse "ERR_INTERNAL: Server error"
// @Security    BearerAuth
// @Router      /admin/batch/change-plan [post]
func (h *AdminHandler) BatchChangePlan(c *gin.Context) {
	var req BatchChangePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	tenantIDs, err := parseUUIDs(req.TenantIDs)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID format")
		return
	}

	input := appIdentity.BatchChangePlanInput{
		TenantIDs: tenantIDs,
		NewPlan:   req.NewPlan,
		Reason:    req.Reason,
	}

	auditCtx := h.getAdminAuditContext(c)
	result, err := h.adminTenantService.BatchChangePlan(c.Request.Context(), input, auditCtx)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toBatchOperationResponse(result))
}

// ============================================================================
// Batch Delete Endpoints
// ============================================================================

// PreviewBatchDelete godoc
// @ID          adminPreviewBatchDelete
// @Summary     Preview batch delete operation
// @Description Preview the batch delete operation before execution. Returns which tenants can be deleted and which will be skipped.
// @Description
// @Description **Authentication**: Requires SuperAdmin role (is_super_admin: true in JWT)
// @Description
// @Description **Skip reasons**:
// @Description - System tenant (protected)
// @Description - Tenant not found
// @Description - Tenant already inactive
// @Tags        admin-batch
// @Accept      json
// @Produce     json
// @Param       request body     BatchDeleteRequest true "Batch delete preview request (max 100 tenants)"
// @Success     200     {object} APIResponse[BatchPreviewResponse] "Preview of batch operation"
// @Failure     400     {object} ErrorResponse "ERR_BAD_REQUEST: Invalid request"
// @Failure     401     {object} ErrorResponse "ERR_UNAUTHORIZED: Missing or invalid token"
// @Failure     403     {object} ErrorResponse "ERR_FORBIDDEN: Not a super admin"
// @Failure     422     {object} ErrorResponse "ERR_BUSINESS_RULE: Business rule violation"
// @Failure     500     {object} ErrorResponse "ERR_INTERNAL: Server error"
// @Security    BearerAuth
// @Router      /admin/batch/delete/preview [post]
func (h *AdminHandler) PreviewBatchDelete(c *gin.Context) {
	var req BatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	tenantIDs, err := parseUUIDs(req.TenantIDs)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID format")
		return
	}

	input := appIdentity.BatchDeleteInput{
		TenantIDs: tenantIDs,
	}

	result, err := h.adminTenantService.PreviewBatchDelete(c.Request.Context(), input)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toBatchPreviewResponse(result))
}

// BatchDelete godoc
// @ID          adminBatchDelete
// @Summary     Execute batch delete operation
// @Description Soft delete multiple tenants at once (marks as inactive, data preserved). All operations are recorded in audit logs.
// @Description
// @Description **Authentication**: Requires SuperAdmin role (is_super_admin: true in JWT)
// @Description
// @Description **Behavior**:
// @Description - Sets all valid tenants to "inactive" status
// @Description - All tenant data is preserved (soft delete)
// @Description - Users of deleted tenants cannot log in
// @Description - Tenants can be reactivated using batch activate
// @Description
// @Description **Limits**: Maximum 100 tenants per batch operation
// @Description
// @Description **Warning**: This operation affects user access immediately
// @Tags        admin-batch
// @Accept      json
// @Produce     json
// @Param       request body     BatchDeleteRequest true "Batch delete request (max 100 tenants)"
// @Success     200     {object} APIResponse[BatchOperationResponse] "Batch operation result"
// @Failure     400     {object} ErrorResponse "ERR_BAD_REQUEST: Invalid request"
// @Failure     401     {object} ErrorResponse "ERR_UNAUTHORIZED: Missing or invalid token"
// @Failure     403     {object} ErrorResponse "ERR_FORBIDDEN: Not a super admin"
// @Failure     422     {object} ErrorResponse "ERR_BUSINESS_RULE: Cannot delete system tenant"
// @Failure     500     {object} ErrorResponse "ERR_INTERNAL: Server error"
// @Security    BearerAuth
// @Router      /admin/batch/delete [post]
func (h *AdminHandler) BatchDelete(c *gin.Context) {
	var req BatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	tenantIDs, err := parseUUIDs(req.TenantIDs)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID format")
		return
	}

	input := appIdentity.BatchDeleteInput{
		TenantIDs: tenantIDs,
	}

	auditCtx := h.getAdminAuditContext(c)
	result, err := h.adminTenantService.BatchDelete(c.Request.Context(), input, auditCtx)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toBatchOperationResponse(result))
}

// RegisterAdminBatchRoutes registers batch operation routes
// All routes are protected by SuperAdminMiddleware
func RegisterAdminBatchRoutes(router *gin.RouterGroup, handler *AdminHandler) {
	batch := router.Group("/batch")

	// Suspend operations
	batch.POST("/suspend/preview", handler.PreviewBatchSuspend)
	batch.POST("/suspend", handler.BatchSuspend)

	// Activate operations
	batch.POST("/activate/preview", handler.PreviewBatchActivate)
	batch.POST("/activate", handler.BatchActivate)

	// Change plan operations
	batch.POST("/change-plan/preview", handler.PreviewBatchChangePlan)
	batch.POST("/change-plan", handler.BatchChangePlan)

	// Delete operations
	batch.POST("/delete/preview", handler.PreviewBatchDelete)
	batch.POST("/delete", handler.BatchDelete)
}
