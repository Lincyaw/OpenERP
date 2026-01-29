package handler

import (
	"github.com/erp/backend/internal/application/identity"
	domainIdentity "github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/erp/backend/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RoleHandler handles role management HTTP requests
type RoleHandler struct {
	BaseHandler
	roleService *identity.RoleService
}

// NewRoleHandler creates a new role handler
func NewRoleHandler(roleService *identity.RoleService) *RoleHandler {
	return &RoleHandler{
		roleService: roleService,
	}
}

// Create godoc
//
//	@ID				createRole
//	@Summary		Create a new role
//	@Description	Create a new role in the system
//	@Tags			roles
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateRoleRequest	true	"Role creation request"
//	@Success		201		{object}	APIResponse[RoleResponse]
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		403		{object}	dto.ErrorResponse
//	@Failure		422		{object}	dto.ErrorResponse
//	@Failure		500		{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/identity/roles [post]
func (h *RoleHandler) Create(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body")
		return
	}

	claims := middleware.GetJWTClaims(c)
	if claims == nil {
		h.Unauthorized(c, "Authentication required")
		return
	}

	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	// Get user ID from JWT context (optional, for data scope)
	userID, _ := getUserID(c)

	input := identity.CreateRoleInput{
		TenantID:    tenantID,
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		Permissions: req.Permissions,
		SortOrder:   req.SortOrder,
	}

	// Set CreatedBy for data scope filtering
	if userID != uuid.Nil {
		input.CreatedBy = &userID
	}

	role, err := h.roleService.Create(c.Request.Context(), input)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Created(c, toRoleResponse(role))
}

// GetByID godoc
//
//	@ID				getRoleById
//	@Summary		Get a role by ID
//	@Description	Retrieve a role by its ID
//	@Tags			roles
//	@Produce		json
//	@Param			id	path		string	true	"Role ID"	format(uuid)
//	@Success		200	{object}	APIResponse[RoleResponse]
//	@Failure		400	{object}	dto.ErrorResponse
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Failure		500	{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/identity/roles/{id} [get]
func (h *RoleHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid role ID")
		return
	}

	role, err := h.roleService.GetByID(c.Request.Context(), id)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toRoleResponse(role))
}

// GetByCode godoc
//
//	@ID				getRoleByCode
//	@Summary		Get a role by code
//	@Description	Retrieve a role by its code
//	@Tags			roles
//	@Produce		json
//	@Param			code	path		string	true	"Role code"
//	@Success		200		{object}	APIResponse[RoleResponse]
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		404		{object}	dto.ErrorResponse
//	@Failure		500		{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/identity/roles/code/{code} [get]
func (h *RoleHandler) GetByCode(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		h.BadRequest(c, "Role code is required")
		return
	}

	claims := middleware.GetJWTClaims(c)
	if claims == nil {
		h.Unauthorized(c, "Authentication required")
		return
	}

	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	role, err := h.roleService.GetByCode(c.Request.Context(), tenantID, code)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toRoleResponse(role))
}

// List godoc
//
//	@ID				listRoles
//	@Summary		List roles
//	@Description	Get a paginated list of roles
//	@Tags			roles
//	@Produce		json
//	@Param			keyword			query		string	false	"Search keyword"
//	@Param			is_enabled		query		bool	false	"Filter by enabled status"
//	@Param			is_system_role	query		bool	false	"Filter by system role"
//	@Param			page			query		int		false	"Page number"		default(1)
//	@Param			page_size		query		int		false	"Items per page"	default(20)	maximum(100)
//	@Success		200				{object}	APIResponse[RoleListResponse]
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/identity/roles [get]
func (h *RoleHandler) List(c *gin.Context) {
	var query RoleListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		h.BadRequest(c, "Invalid query parameters")
		return
	}

	claims := middleware.GetJWTClaims(c)
	if claims == nil {
		h.Unauthorized(c, "Authentication required")
		return
	}

	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	// Build filter
	filter := &domainIdentity.RoleFilter{
		Keyword:      query.Keyword,
		IsEnabled:    query.IsEnabled,
		IsSystemRole: query.IsSystemRole,
		Page:         query.Page,
		Limit:        query.PageSize,
	}

	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.Limit == 0 {
		filter.Limit = 20
	}

	result, err := h.roleService.List(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toRoleListResponse(result))
}

// Update godoc
//
//	@ID				updateRole
//	@Summary		Update a role
//	@Description	Update a role's information
//	@Tags			roles
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Role ID"	format(uuid)
//	@Param			request	body		UpdateRoleRequest	true	"Role update request"
//	@Success		200		{object}	APIResponse[RoleResponse]
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		404		{object}	dto.ErrorResponse
//	@Failure		422		{object}	dto.ErrorResponse
//	@Failure		500		{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/identity/roles/{id} [put]
func (h *RoleHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid role ID")
		return
	}

	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body")
		return
	}

	input := identity.UpdateRoleInput{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		SortOrder:   req.SortOrder,
	}

	role, err := h.roleService.Update(c.Request.Context(), input)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toRoleResponse(role))
}

// Delete godoc
//
//	@ID				deleteRole
//	@Summary		Delete a role
//	@Description	Delete a role from the system
//	@Tags			roles
//	@Produce		json
//	@Param			id	path		string	true	"Role ID"	format(uuid)
//	@Success		200	{object}	SuccessResponse
//	@Failure		400	{object}	dto.ErrorResponse
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Failure		422	{object}	dto.ErrorResponse
//	@Failure		500	{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/identity/roles/{id} [delete]
func (h *RoleHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid role ID")
		return
	}

	if err := h.roleService.Delete(c.Request.Context(), id); err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, dto.MessageResponse{Message: "Role deleted successfully"})
}

// Enable godoc
//
//	@ID				enableRole
//	@Summary		Enable a role
//	@Description	Enable a role
//	@Tags			roles
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Role ID"	format(uuid)
//	@Success		200	{object}	APIResponse[RoleResponse]
//	@Failure		400	{object}	dto.ErrorResponse
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Failure		422	{object}	dto.ErrorResponse
//	@Failure		500	{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/identity/roles/{id}/enable [post]
func (h *RoleHandler) Enable(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid role ID")
		return
	}

	role, err := h.roleService.Enable(c.Request.Context(), id)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toRoleResponse(role))
}

// Disable godoc
//
//	@ID				disableRole
//	@Summary		Disable a role
//	@Description	Disable a role
//	@Tags			roles
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Role ID"	format(uuid)
//	@Success		200	{object}	APIResponse[RoleResponse]
//	@Failure		400	{object}	dto.ErrorResponse
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Failure		422	{object}	dto.ErrorResponse
//	@Failure		500	{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/identity/roles/{id}/disable [post]
func (h *RoleHandler) Disable(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid role ID")
		return
	}

	role, err := h.roleService.Disable(c.Request.Context(), id)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toRoleResponse(role))
}

// SetPermissions godoc
//
//	@ID				setPermissionsRole
//	@Summary		Set role permissions
//	@Description	Set permissions for a role (replaces existing permissions)
//	@Tags			roles
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Role ID"	format(uuid)
//	@Param			request	body		SetPermissionsRequest	true	"Permissions"
//	@Success		200		{object}	APIResponse[RoleResponse]
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		404		{object}	dto.ErrorResponse
//	@Failure		422		{object}	dto.ErrorResponse
//	@Failure		500		{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/identity/roles/{id}/permissions [put]
func (h *RoleHandler) SetPermissions(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid role ID")
		return
	}

	var req SetPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body")
		return
	}

	role, err := h.roleService.SetPermissions(c.Request.Context(), id, req.Permissions)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toRoleResponse(role))
}

// GetPermissions godoc
//
//	@ID				getRolePermissions
//	@Summary		Get all available permissions
//	@Description	Get all available permission codes
//	@Tags			roles
//	@Produce		json
//	@Success		200	{object}	APIResponse[PermissionListResponse]
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		500	{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/identity/permissions [get]
func (h *RoleHandler) GetPermissions(c *gin.Context) {
	claims := middleware.GetJWTClaims(c)
	if claims == nil {
		h.Unauthorized(c, "Authentication required")
		return
	}

	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	permissions, err := h.roleService.GetAllPermissionCodes(c.Request.Context(), tenantID)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	// If no permissions found in the database, return predefined permissions
	if len(permissions) == 0 {
		permissions = getPredefinedPermissions()
	}

	h.Success(c, PermissionListResponse{Permissions: permissions})
}

// GetSystemRoles godoc
//
//	@ID				getRoleSystemRoles
//	@Summary		Get system roles
//	@Description	Get all system roles
//	@Tags			roles
//	@Produce		json
//	@Success		200	{object}	APIResponse[[]RoleResponse]
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		500	{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/identity/roles/system [get]
func (h *RoleHandler) GetSystemRoles(c *gin.Context) {
	claims := middleware.GetJWTClaims(c)
	if claims == nil {
		h.Unauthorized(c, "Authentication required")
		return
	}

	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	roles, err := h.roleService.GetSystemRoles(c.Request.Context(), tenantID)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	responses := make([]RoleResponse, len(roles))
	for i, role := range roles {
		responses[i] = *toRoleResponse(&role)
	}

	h.Success(c, responses)
}

// Count godoc
//
//	@ID				countRoles
//	@Summary		Get role count
//	@Description	Get the total number of roles
//	@Tags			roles
//	@Produce		json
//	@Success		200	{object}	APIResponse[CountData]
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		500	{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/identity/roles/stats/count [get]
func (h *RoleHandler) Count(c *gin.Context) {
	claims := middleware.GetJWTClaims(c)
	if claims == nil {
		h.Unauthorized(c, "Authentication required")
		return
	}

	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	count, err := h.roleService.Count(c.Request.Context(), tenantID)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, gin.H{"count": count})
}

// Helper functions for response conversion

func toRoleResponse(role *identity.RoleDTO) *RoleResponse {
	return &RoleResponse{
		ID:           role.ID,
		TenantID:     role.TenantID,
		Code:         role.Code,
		Name:         role.Name,
		Description:  role.Description,
		IsSystemRole: role.IsSystemRole,
		IsEnabled:    role.IsEnabled,
		SortOrder:    role.SortOrder,
		Permissions:  role.Permissions,
		UserCount:    role.UserCount,
		CreatedAt:    role.CreatedAt,
		UpdatedAt:    role.UpdatedAt,
	}
}

func toRoleListResponse(result *identity.RoleListResult) *RoleListResponse {
	roles := make([]RoleResponse, len(result.Roles))
	for i, role := range result.Roles {
		roles[i] = *toRoleResponse(&role)
	}

	return &RoleListResponse{
		Roles:      roles,
		Total:      result.Total,
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalPages: result.TotalPages,
	}
}

// getPredefinedPermissions returns a list of predefined permission codes
func getPredefinedPermissions() []string {
	resources := []string{
		"product", "category", "customer", "supplier", "warehouse",
		"inventory", "sales_order", "purchase_order", "sales_return", "purchase_return",
		"account_receivable", "account_payable", "receipt", "payment",
		"expense", "income", "report", "user", "role", "tenant",
	}

	// Generate common permissions
	commonActions := []string{"create", "read", "update", "delete"}
	permissions := make([]string, 0, len(resources)*len(commonActions)+20)

	for _, resource := range resources {
		for _, action := range commonActions {
			permissions = append(permissions, resource+":"+action)
		}
	}

	// Add special permissions
	specialPerms := []string{
		"product:enable", "product:disable",
		"customer:enable", "customer:disable",
		"supplier:enable", "supplier:disable",
		"warehouse:enable", "warehouse:disable",
		"sales_order:confirm", "sales_order:cancel", "sales_order:ship",
		"purchase_order:confirm", "purchase_order:cancel", "purchase_order:receive",
		"sales_return:approve", "sales_return:reject",
		"inventory:adjust", "inventory:lock", "inventory:unlock",
		"account_receivable:reconcile", "account_payable:reconcile",
		"user:lock", "user:unlock", "user:assign_role",
		"role:enable", "role:disable",
		"report:export", "report:view_all",
	}
	permissions = append(permissions, specialPerms...)

	return permissions
}
