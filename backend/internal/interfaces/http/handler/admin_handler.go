package handler

import (
	"time"

	appIdentity "github.com/erp/backend/internal/application/identity"
	"github.com/erp/backend/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminHandler handles all admin HTTP requests for tenant management.
//
// All endpoints in this handler require SuperAdmin privileges.
// Authentication is performed via JWT Bearer token with is_super_admin: true claim.
//
// Error Codes:
//   - ERR_BAD_REQUEST (400): Invalid request parameters or body
//   - ERR_UNAUTHORIZED (401): Missing or invalid authentication token
//   - ERR_FORBIDDEN (403): User lacks super admin privileges
//   - ERR_NOT_FOUND (404): Requested tenant not found
//   - ERR_ALREADY_EXISTS (409): Tenant code or name already exists
//   - ERR_BUSINESS_RULE (422): Business rule violation (e.g., cannot modify system tenant)
//   - ERR_INTERNAL (500): Internal server error
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
// @Description Get a paginated list of all tenants with filtering and sorting options.
// @Description
// @Description **Authentication**: Requires SuperAdmin role (is_super_admin: true in JWT)
// @Description
// @Description **Filtering**:
// @Description - Search by tenant name or code (partial match)
// @Description - Filter by status: active, inactive, suspended, trial
// @Description - Filter by plan: free, basic, pro, enterprise
// @Description
// @Description **Sorting**: Results can be sorted by name, code, status, plan, or created_at
// @Tags        admin-tenants
// @Produce     json
// @Param       page          query    int     false  "Page number (1-based)" default(1) minimum(1)
// @Param       page_size     query    int     false  "Items per page (max 100)" default(20) minimum(1) maximum(100)
// @Param       search        query    string  false  "Search by name or code (case-insensitive)"
// @Param       status        query    string  false  "Filter by tenant status" Enums(active, inactive, suspended, trial)
// @Param       plan          query    string  false  "Filter by subscription plan" Enums(free, basic, pro, enterprise)
// @Param       order_by      query    string  false  "Field to sort by" Enums(name, code, status, plan, created_at) default(created_at)
// @Param       order_dir     query    string  false  "Sort direction" Enums(asc, desc) default(desc)
// @Success     200           {object} APIResponse[AdminTenantListResponse] "Paginated list of tenants"
// @Failure     400           {object} ErrorResponse "ERR_BAD_REQUEST: Invalid query parameters"
// @Failure     401           {object} ErrorResponse "ERR_UNAUTHORIZED: Missing or invalid token"
// @Failure     403           {object} ErrorResponse "ERR_FORBIDDEN: Not a super admin"
// @Failure     500           {object} ErrorResponse "ERR_INTERNAL: Server error"
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
// @Description Create a new tenant in the system with the specified configuration.
// @Description
// @Description **Authentication**: Requires SuperAdmin role (is_super_admin: true in JWT)
// @Description
// @Description **Plan Options**:
// @Description - free: Limited features, ideal for evaluation
// @Description - basic: Standard features for small businesses
// @Description - pro: Advanced features for growing businesses
// @Description - enterprise: Full features with custom quotas
// @Description
// @Description **Trial**: Optionally specify trial_days (max 90) for trial period
// @Description
// @Description **Audit**: All create operations are logged in the audit trail
// @Tags        admin-tenants
// @Accept      json
// @Produce     json
// @Param       request body     AdminCreateTenantRequest true "Tenant creation request"
// @Success     201     {object} APIResponse[AdminTenantResponse] "Tenant created successfully"
// @Failure     400     {object} ErrorResponse "ERR_BAD_REQUEST: Invalid request body"
// @Failure     401     {object} ErrorResponse "ERR_UNAUTHORIZED: Missing or invalid token"
// @Failure     403     {object} ErrorResponse "ERR_FORBIDDEN: Not a super admin"
// @Failure     409     {object} ErrorResponse "ERR_ALREADY_EXISTS: Tenant code or name already exists"
// @Failure     422     {object} ErrorResponse "ERR_BUSINESS_RULE: Business rule violation"
// @Failure     500     {object} ErrorResponse "ERR_INTERNAL: Server error"
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
// @Description Get detailed information about a specific tenant including configuration and statistics.
// @Description
// @Description **Authentication**: Requires SuperAdmin role (is_super_admin: true in JWT)
// @Description
// @Description **Response includes**:
// @Description - Basic tenant info (name, code, status, plan)
// @Description - Contact information
// @Description - Configuration (quotas, cost strategy, regional settings)
// @Description - Usage statistics (user count, product count, etc.)
// @Tags        admin-tenants
// @Produce     json
// @Param       id   path     string true "Tenant ID (UUID format)" format(uuid)
// @Success     200  {object} APIResponse[AdminTenantResponse] "Tenant details"
// @Failure     400  {object} ErrorResponse "ERR_BAD_REQUEST: Invalid tenant ID format"
// @Failure     401  {object} ErrorResponse "ERR_UNAUTHORIZED: Missing or invalid token"
// @Failure     403  {object} ErrorResponse "ERR_FORBIDDEN: Not a super admin"
// @Failure     404  {object} ErrorResponse "ERR_NOT_FOUND: Tenant not found"
// @Failure     500  {object} ErrorResponse "ERR_INTERNAL: Server error"
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
// @Description Update a tenant's basic information. All fields are optional - only provided fields will be updated.
// @Description
// @Description **Authentication**: Requires SuperAdmin role (is_super_admin: true in JWT)
// @Description
// @Description **Updatable fields**:
// @Description - name, short_name: Display names
// @Description - contact_name, contact_phone, contact_email: Contact info
// @Description - address: Physical address
// @Description - notes: Internal notes
// @Description
// @Description **Note**: To change plan or quotas, use the dedicated endpoints
// @Description
// @Description **Audit**: All update operations are logged in the audit trail
// @Tags        admin-tenants
// @Accept      json
// @Produce     json
// @Param       id      path     string                   true "Tenant ID (UUID format)" format(uuid)
// @Param       request body     AdminUpdateTenantRequest true "Tenant update request (all fields optional)"
// @Success     200     {object} APIResponse[AdminTenantResponse] "Updated tenant details"
// @Failure     400     {object} ErrorResponse "ERR_BAD_REQUEST: Invalid request"
// @Failure     401     {object} ErrorResponse "ERR_UNAUTHORIZED: Missing or invalid token"
// @Failure     403     {object} ErrorResponse "ERR_FORBIDDEN: Not a super admin"
// @Failure     404     {object} ErrorResponse "ERR_NOT_FOUND: Tenant not found"
// @Failure     422     {object} ErrorResponse "ERR_BUSINESS_RULE: Business rule violation"
// @Failure     500     {object} ErrorResponse "ERR_INTERNAL: Server error"
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
// @Description Soft delete a tenant. The tenant is marked as inactive but data is preserved.
// @Description
// @Description **Authentication**: Requires SuperAdmin role (is_super_admin: true in JWT)
// @Description
// @Description **Behavior**:
// @Description - Sets tenant status to "inactive"
// @Description - All tenant data is preserved (soft delete)
// @Description - Users of the tenant can no longer log in
// @Description - Tenant can be reactivated using the activate endpoint
// @Description
// @Description **Restrictions**:
// @Description - System tenant cannot be deleted (ERR_FORBIDDEN)
// @Description
// @Description **Audit**: All delete operations are logged in the audit trail
// @Tags        admin-tenants
// @Produce     json
// @Param       id   path     string true "Tenant ID (UUID format)" format(uuid)
// @Success     200  {object} APIResponse[MessageResponse] "Tenant deleted successfully"
// @Failure     400  {object} ErrorResponse "ERR_BAD_REQUEST: Invalid tenant ID format"
// @Failure     401  {object} ErrorResponse "ERR_UNAUTHORIZED: Missing or invalid token"
// @Failure     403  {object} ErrorResponse "ERR_FORBIDDEN: Cannot delete system tenant"
// @Failure     404  {object} ErrorResponse "ERR_NOT_FOUND: Tenant not found"
// @Failure     422  {object} ErrorResponse "ERR_BUSINESS_RULE: Business rule violation"
// @Failure     500  {object} ErrorResponse "ERR_INTERNAL: Server error"
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
// @Description Suspend a tenant with optional reason and scheduled reactivation date.
// @Description
// @Description **Authentication**: Requires SuperAdmin role (is_super_admin: true in JWT)
// @Description
// @Description **Behavior**:
// @Description - Sets tenant status to "suspended"
// @Description - All users of the tenant are immediately blocked from login
// @Description - Optional reason is stored and visible in audit logs
// @Description - Optional scheduled_reactivate_at sets automatic reactivation
// @Description
// @Description **Restrictions**:
// @Description - System tenant cannot be suspended (ERR_FORBIDDEN)
// @Description - Already suspended tenants return success (idempotent)
// @Description
// @Description **Audit**: All suspend operations are logged with reason
// @Tags        admin-tenants
// @Accept      json
// @Produce     json
// @Param       id      path     string                true "Tenant ID (UUID format)" format(uuid)
// @Param       request body     SuspendTenantRequest  false "Suspension details (optional)"
// @Success     200     {object} APIResponse[AdminTenantResponse] "Tenant suspended"
// @Failure     400     {object} ErrorResponse "ERR_BAD_REQUEST: Invalid request"
// @Failure     401     {object} ErrorResponse "ERR_UNAUTHORIZED: Missing or invalid token"
// @Failure     403     {object} ErrorResponse "ERR_FORBIDDEN: Cannot suspend system tenant"
// @Failure     404     {object} ErrorResponse "ERR_NOT_FOUND: Tenant not found"
// @Failure     422     {object} ErrorResponse "ERR_BUSINESS_RULE: Business rule violation"
// @Failure     500     {object} ErrorResponse "ERR_INTERNAL: Server error"
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
// @Description Activate a suspended or inactive tenant, restoring full access.
// @Description
// @Description **Authentication**: Requires SuperAdmin role (is_super_admin: true in JWT)
// @Description
// @Description **Behavior**:
// @Description - Sets tenant status to "active"
// @Description - All users of the tenant can log in again
// @Description - Clears any scheduled reactivation date
// @Description
// @Description **Use cases**:
// @Description - Reactivating suspended tenants after payment
// @Description - Restoring soft-deleted (inactive) tenants
// @Description - Ending trial period early (converting to active)
// @Description
// @Description **Audit**: All activation operations are logged
// @Tags        admin-tenants
// @Produce     json
// @Param       id   path     string true "Tenant ID (UUID format)" format(uuid)
// @Success     200  {object} APIResponse[AdminTenantResponse] "Tenant activated"
// @Failure     400  {object} ErrorResponse "ERR_BAD_REQUEST: Invalid tenant ID format"
// @Failure     401  {object} ErrorResponse "ERR_UNAUTHORIZED: Missing or invalid token"
// @Failure     403  {object} ErrorResponse "ERR_FORBIDDEN: Not a super admin"
// @Failure     404  {object} ErrorResponse "ERR_NOT_FOUND: Tenant not found"
// @Failure     422  {object} ErrorResponse "ERR_BUSINESS_RULE: Cannot activate (e.g., already active)"
// @Failure     500  {object} ErrorResponse "ERR_INTERNAL: Server error"
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
// @Description Get detailed usage statistics for a specific tenant including quota utilization.
// @Description
// @Description **Authentication**: Requires SuperAdmin role (is_super_admin: true in JWT)
// @Description
// @Description **Statistics included**:
// @Description - User count vs max_users quota
// @Description - Product count vs max_products quota
// @Description - Warehouse count vs max_warehouses quota
// @Description - Order count (historical)
// @Description - Storage usage in bytes
// @Description - API call count
// @Description
// @Description **Use cases**:
// @Description - Monitoring tenant resource usage
// @Description - Identifying tenants approaching quota limits
// @Description - Capacity planning
// @Tags        admin-tenants
// @Produce     json
// @Param       id   path     string true "Tenant ID (UUID format)" format(uuid)
// @Success     200  {object} APIResponse[TenantUsageStatsResponse] "Tenant statistics"
// @Failure     400  {object} ErrorResponse "ERR_BAD_REQUEST: Invalid tenant ID format"
// @Failure     401  {object} ErrorResponse "ERR_UNAUTHORIZED: Missing or invalid token"
// @Failure     403  {object} ErrorResponse "ERR_FORBIDDEN: Not a super admin"
// @Failure     404  {object} ErrorResponse "ERR_NOT_FOUND: Tenant not found"
// @Failure     500  {object} ErrorResponse "ERR_INTERNAL: Server error"
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
// @Description Get aggregated statistics across all tenants in the platform.
// @Description
// @Description **Authentication**: Requires SuperAdmin role (is_super_admin: true in JWT)
// @Description
// @Description **Statistics included**:
// @Description - Tenant counts by status (active, trial, suspended, inactive)
// @Description - Tenant distribution by plan
// @Description - Aggregate user count across all tenants
// @Description - Aggregate product count across all tenants
// @Description - Aggregate order count across all tenants
// @Description - Total storage usage in bytes
// @Description
// @Description **Use cases**:
// @Description - Platform health dashboard
// @Description - Revenue analysis (by plan distribution)
// @Description - Growth monitoring
// @Tags        admin-stats
// @Produce     json
// @Success     200  {object} APIResponse[PlatformStatsResponse] "Platform statistics"
// @Failure     401  {object} ErrorResponse "ERR_UNAUTHORIZED: Missing or invalid token"
// @Failure     403  {object} ErrorResponse "ERR_FORBIDDEN: Not a super admin"
// @Failure     500  {object} ErrorResponse "ERR_INTERNAL: Server error"
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
// @Description Get tenant growth trend data for charts and analytics.
// @Description
// @Description **Authentication**: Requires SuperAdmin role (is_super_admin: true in JWT)
// @Description
// @Description **Data included per period**:
// @Description - Total tenant count
// @Description - New tenants added
// @Description - Churned tenants (deleted/inactive)
// @Description
// @Description **Period options**:
// @Description - daily: Day-by-day data points
// @Description - weekly: Aggregated by week
// @Description - monthly: Aggregated by month
// @Description
// @Description **Use cases**:
// @Description - Growth rate analysis
// @Description - Churn monitoring
// @Description - Trend visualization in dashboards
// @Tags        admin-stats
// @Produce     json
// @Param       period query    string false "Aggregation period" Enums(daily, weekly, monthly) default(daily)
// @Param       days   query    int    false "Number of days to look back (max 365)" default(30) minimum(1) maximum(365)
// @Success     200    {object} APIResponse[TenantGrowthTrendResponse] "Growth trend data"
// @Failure     400    {object} ErrorResponse "ERR_BAD_REQUEST: Invalid parameters"
// @Failure     401    {object} ErrorResponse "ERR_UNAUTHORIZED: Missing or invalid token"
// @Failure     403    {object} ErrorResponse "ERR_FORBIDDEN: Not a super admin"
// @Failure     500    {object} ErrorResponse "ERR_INTERNAL: Server error"
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
// @Description Get a paginated list of admin audit logs with filtering options.
// @Description
// @Description **Authentication**: Requires SuperAdmin role (is_super_admin: true in JWT)
// @Description
// @Description **Logged actions**:
// @Description - tenant_create, tenant_update, tenant_delete
// @Description - tenant_suspend, tenant_activate
// @Description - subscription_change, quota_update
// @Description
// @Description **Target types**:
// @Description - tenant: Tenant entity changes
// @Description - subscription: Plan changes
// @Description - quota: Quota limit changes
// @Description
// @Description **Each log entry contains**:
// @Description - Admin user who performed the action
// @Description - Action type and target
// @Description - Old and new values (for tracking changes)
// @Description - IP address and user agent
// @Description - Timestamp
// @Tags        admin-audit
// @Produce     json
// @Param       page          query    int     false  "Page number (1-based)" default(1) minimum(1)
// @Param       page_size     query    int     false  "Items per page (max 100)" default(20) minimum(1) maximum(100)
// @Param       admin_user_id query    string  false  "Filter by admin user ID" format(uuid)
// @Param       action        query    string  false  "Filter by action type" Enums(tenant_create, tenant_update, tenant_delete, tenant_suspend, tenant_activate, subscription_change, quota_update)
// @Param       target_type   query    string  false  "Filter by target entity type" Enums(tenant, subscription, quota)
// @Param       target_id     query    string  false  "Filter by target entity ID" format(uuid)
// @Param       start_time    query    string  false  "Filter by start time (RFC3339)" format(date-time)
// @Param       end_time      query    string  false  "Filter by end time (RFC3339)" format(date-time)
// @Param       sort_by       query    string  false  "Sort by field" Enums(created_at, action, target_type) default(created_at)
// @Param       sort_order    query    string  false  "Sort order" Enums(asc, desc) default(desc)
// @Param       ip_address    query    string  false  "Filter by IP address"
// @Success     200           {object} APIResponse[AdminAuditLogListResponse] "Paginated audit logs"
// @Failure     400           {object} ErrorResponse "ERR_BAD_REQUEST: Invalid parameters"
// @Failure     401           {object} ErrorResponse "ERR_UNAUTHORIZED: Missing or invalid token"
// @Failure     403           {object} ErrorResponse "ERR_FORBIDDEN: Not a super admin"
// @Failure     500           {object} ErrorResponse "ERR_INTERNAL: Server error"
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

	// Batch operations
	RegisterAdminBatchRoutes(admin, handler)

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
// @Description Change a tenant's subscription plan with automatic upgrade/downgrade handling.
// @Description
// @Description **Authentication**: Requires SuperAdmin role (is_super_admin: true in JWT)
// @Description
// @Description **Plan hierarchy** (low to high):
// @Description - free → basic → pro → enterprise
// @Description
// @Description **Upgrade behavior**:
// @Description - Takes effect immediately
// @Description - New quotas apply instantly
// @Description - change_type: "immediate"
// @Description
// @Description **Downgrade behavior**:
// @Description - Scheduled for end of current billing cycle
// @Description - Current quotas remain until effective date
// @Description - change_type: "scheduled"
// @Description - Can be cancelled before effective date
// @Description
// @Description **Audit**: All plan changes are logged with reason
// @Tags        admin-tenants
// @Accept      json
// @Produce     json
// @Param       id      path     string            true  "Tenant ID (UUID format)" format(uuid)
// @Param       request body     ChangePlanRequest true  "Plan change request"
// @Success     200     {object} APIResponse[ChangePlanResponse] "Plan change result"
// @Failure     400     {object} ErrorResponse "ERR_BAD_REQUEST: Invalid request"
// @Failure     401     {object} ErrorResponse "ERR_UNAUTHORIZED: Missing or invalid token"
// @Failure     403     {object} ErrorResponse "ERR_FORBIDDEN: Not a super admin"
// @Failure     404     {object} ErrorResponse "ERR_NOT_FOUND: Tenant not found"
// @Failure     422     {object} ErrorResponse "ERR_BUSINESS_RULE: Same plan or invalid transition"
// @Failure     500     {object} ErrorResponse "ERR_INTERNAL: Server error"
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
// @Description
// @Description **Authentication**: Requires SuperAdmin role (is_super_admin: true in JWT)
// @Description
// @Description **Updatable quotas**:
// @Description - max_users: Maximum number of users
// @Description - max_warehouses: Maximum number of warehouses
// @Description - max_products: Maximum number of products
// @Description
// @Description **Behavior**:
// @Description - All fields are optional - only provided fields are updated
// @Description - Changes take effect immediately
// @Description - Does not affect existing data exceeding new limits
// @Description
// @Description **Note**: Consider tenant's current plan limits as baseline
// @Description
// @Description **Audit**: All quota changes are logged with old/new values
// @Tags        admin-tenants
// @Accept      json
// @Produce     json
// @Param       id      path     string             true  "Tenant ID (UUID format)" format(uuid)
// @Param       request body     UpdateQuotaRequest true  "Quota update request (all fields optional)"
// @Success     200     {object} APIResponse[AdminTenantResponse] "Updated tenant with new quotas"
// @Failure     400     {object} ErrorResponse "ERR_BAD_REQUEST: Invalid request"
// @Failure     401     {object} ErrorResponse "ERR_UNAUTHORIZED: Missing or invalid token"
// @Failure     403     {object} ErrorResponse "ERR_FORBIDDEN: Not a super admin"
// @Failure     404     {object} ErrorResponse "ERR_NOT_FOUND: Tenant not found"
// @Failure     422     {object} ErrorResponse "ERR_BUSINESS_RULE: Invalid quota values"
// @Failure     500     {object} ErrorResponse "ERR_INTERNAL: Server error"
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
// @Description Retrieve subscription change history for a tenant with filtering.
// @Description
// @Description **Authentication**: Requires SuperAdmin role (is_super_admin: true in JWT)
// @Description
// @Description **History includes**:
// @Description - Plan upgrades and downgrades
// @Description - Quota updates
// @Description - Effective dates and scheduled changes
// @Description - Admin who made the change
// @Description - Reason for change (if provided)
// @Description
// @Description **Change types**:
// @Description - plan_upgrade: Plan was upgraded
// @Description - plan_downgrade: Plan was downgraded
// @Description - quota_update: Quota limits were changed
// @Tags        admin-tenants
// @Produce     json
// @Param       id          path     string  true   "Tenant ID (UUID format)" format(uuid)
// @Param       page        query    int     false  "Page number (1-based)" default(1) minimum(1)
// @Param       page_size   query    int     false  "Items per page (max 100)" default(20) minimum(1) maximum(100)
// @Param       change_type query    string  false  "Filter by change type" Enums(plan_upgrade, plan_downgrade, quota_update)
// @Success     200         {object} APIResponse[SubscriptionHistoryListResponse] "Subscription history"
// @Failure     400         {object} ErrorResponse "ERR_BAD_REQUEST: Invalid parameters"
// @Failure     401         {object} ErrorResponse "ERR_UNAUTHORIZED: Missing or invalid token"
// @Failure     403         {object} ErrorResponse "ERR_FORBIDDEN: Not a super admin"
// @Failure     404         {object} ErrorResponse "ERR_NOT_FOUND: Tenant not found"
// @Failure     500         {object} ErrorResponse "ERR_INTERNAL: Server error"
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
// @Description Cancel a pending plan downgrade for a tenant before it takes effect.
// @Description
// @Description **Authentication**: Requires SuperAdmin role (is_super_admin: true in JWT)
// @Description
// @Description **Behavior**:
// @Description - Clears scheduled_plan and scheduled_plan_effective_at
// @Description - Tenant remains on current plan indefinitely
// @Description - No effect if no scheduled change exists
// @Description
// @Description **Use case**:
// @Description - Customer decides to stay on current plan
// @Description - Downgrade was scheduled in error
// @Description
// @Description **Audit**: Cancellation is logged in audit trail
// @Tags        admin-tenants
// @Produce     json
// @Param       id   path     string  true  "Tenant ID (UUID format)" format(uuid)
// @Success     200  {object} APIResponse[AdminTenantResponse] "Tenant with no scheduled plan"
// @Failure     400  {object} ErrorResponse "ERR_BAD_REQUEST: Invalid tenant ID format"
// @Failure     401  {object} ErrorResponse "ERR_UNAUTHORIZED: Missing or invalid token"
// @Failure     403  {object} ErrorResponse "ERR_FORBIDDEN: Not a super admin"
// @Failure     404  {object} ErrorResponse "ERR_NOT_FOUND: Tenant not found"
// @Failure     422  {object} ErrorResponse "ERR_BUSINESS_RULE: No scheduled change to cancel"
// @Failure     500  {object} ErrorResponse "ERR_INTERNAL: Server error"
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
