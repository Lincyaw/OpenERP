package handler

import (
	"time"

	"github.com/erp/backend/internal/application/identity"
	domainIdentity "github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/erp/backend/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UserHandler handles user management HTTP requests
type UserHandler struct {
	BaseHandler
	userService *identity.UserService
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService *identity.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}


// Create godoc
// @ID           createUser
// @Summary      Create a new user
// @Description  Create a new user in the system
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request body CreateUserRequest true "User creation request"
// @Success      201 {object} APIResponse[UserResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Failure      422 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /identity/users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req CreateUserRequest
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

	// Parse role IDs
	roleIDs := make([]uuid.UUID, 0, len(req.RoleIDs))
	for _, ridStr := range req.RoleIDs {
		rid, err := uuid.Parse(ridStr)
		if err != nil {
			h.BadRequest(c, "Invalid role ID: "+ridStr)
			return
		}
		roleIDs = append(roleIDs, rid)
	}

	// Get user ID from JWT context (optional, for data scope)
	userID, _ := getUserID(c)

	input := identity.CreateUserInput{
		TenantID:    tenantID,
		Username:    req.Username,
		Password:    req.Password,
		Email:       req.Email,
		Phone:       req.Phone,
		DisplayName: req.DisplayName,
		Notes:       req.Notes,
		RoleIDs:     roleIDs,
	}

	// Set CreatedBy for data scope filtering
	if userID != uuid.Nil {
		input.CreatedBy = &userID
	}

	user, err := h.userService.Create(c.Request.Context(), input)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Created(c, toUserResponse(user))
}


// GetByID godoc
// @ID           getUserById
// @Summary      Get a user by ID
// @Description  Retrieve a user by their ID
// @Tags         users
// @Produce      json
// @Param        id path string true "User ID" format(uuid)
// @Success      200 {object} APIResponse[UserResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /identity/users/{id} [get]
func (h *UserHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid user ID")
		return
	}

	user, err := h.userService.GetByID(c.Request.Context(), id)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toUserResponse(user))
}


// List godoc
// @ID           listUsers
// @Summary      List users
// @Description  Get a paginated list of users
// @Tags         users
// @Produce      json
// @Param        keyword query string false "Search keyword"
// @Param        status query string false "User status" Enums(pending, active, locked, deactivated)
// @Param        role_id query string false "Filter by role ID" format(uuid)
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Items per page" default(20) maximum(100)
// @Param        sort_by query string false "Sort by field" Enums(username, email, display_name, created_at, updated_at, last_login_at)
// @Param        sort_dir query string false "Sort direction" Enums(asc, desc)
// @Success      200 {object} APIResponse[UserListResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /identity/users [get]
func (h *UserHandler) List(c *gin.Context) {
	var query UserListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		h.BadRequest(c, "Invalid query parameters")
		return
	}

	// Build filter
	filter := domainIdentity.NewUserFilter()
	if query.Keyword != "" {
		filter = filter.WithKeyword(query.Keyword)
	}
	if query.Status != "" {
		status := domainIdentity.UserStatus(query.Status)
		filter = filter.WithStatus(status)
	}
	if query.RoleID != "" {
		roleID, err := uuid.Parse(query.RoleID)
		if err != nil {
			h.BadRequest(c, "Invalid role ID")
			return
		}
		filter = filter.WithRoleID(roleID)
	}
	if query.Page > 0 {
		filter.Page = query.Page
	}
	if query.PageSize > 0 {
		filter.PageSize = query.PageSize
	}
	if query.SortBy != "" {
		filter.SortBy = query.SortBy
	}
	if query.SortDir != "" {
		filter.SortOrder = query.SortDir
	}

	result, err := h.userService.List(c.Request.Context(), filter)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toUserListResponse(result))
}


// Update godoc
// @ID           updateUser
// @Summary      Update a user
// @Description  Update a user's information
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id path string true "User ID" format(uuid)
// @Param        request body UpdateUserRequest true "User update request"
// @Success      200 {object} APIResponse[UserResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      422 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /identity/users/{id} [put]
func (h *UserHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid user ID")
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body")
		return
	}

	input := identity.UpdateUserInput{
		ID:          id,
		Email:       req.Email,
		Phone:       req.Phone,
		DisplayName: req.DisplayName,
		Notes:       req.Notes,
	}

	user, err := h.userService.Update(c.Request.Context(), input)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toUserResponse(user))
}


// Delete godoc
// @ID           deleteUser
// @Summary      Delete a user
// @Description  Delete a user from the system
// @Tags         users
// @Produce      json
// @Param        id path string true "User ID" format(uuid)
// @Success      200 {object} SuccessResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /identity/users/{id} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid user ID")
		return
	}

	if err := h.userService.Delete(c.Request.Context(), id); err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, dto.MessageResponse{Message: "User deleted successfully"})
}


// Activate godoc
// @ID           activateUser
// @Summary      Activate a user
// @Description  Activate a user account
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id path string true "User ID" format(uuid)
// @Success      200 {object} APIResponse[UserResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      422 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /identity/users/{id}/activate [post]
func (h *UserHandler) Activate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid user ID")
		return
	}

	user, err := h.userService.Activate(c.Request.Context(), id)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toUserResponse(user))
}


// Deactivate godoc
// @ID           deactivateUser
// @Summary      Deactivate a user
// @Description  Deactivate a user account
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id path string true "User ID" format(uuid)
// @Success      200 {object} APIResponse[UserResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      422 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /identity/users/{id}/deactivate [post]
func (h *UserHandler) Deactivate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid user ID")
		return
	}

	user, err := h.userService.Deactivate(c.Request.Context(), id)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toUserResponse(user))
}


// Lock godoc
// @ID           lockUser
// @Summary      Lock a user
// @Description  Lock a user account for a specified duration
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id path string true "User ID" format(uuid)
// @Param        request body LockUserRequest false "Lock duration"
// @Success      200 {object} APIResponse[UserResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      422 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /identity/users/{id}/lock [post]
func (h *UserHandler) Lock(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid user ID")
		return
	}

	var req LockUserRequest
	_ = c.ShouldBindJSON(&req) // Optional body

	duration := time.Duration(0)
	if req.DurationMinutes > 0 {
		duration = time.Duration(req.DurationMinutes) * time.Minute
	}

	user, err := h.userService.Lock(c.Request.Context(), id, duration)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toUserResponse(user))
}


// Unlock godoc
// @ID           unlockUser
// @Summary      Unlock a user
// @Description  Unlock a locked user account
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id path string true "User ID" format(uuid)
// @Success      200 {object} APIResponse[UserResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      422 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /identity/users/{id}/unlock [post]
func (h *UserHandler) Unlock(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid user ID")
		return
	}

	user, err := h.userService.Unlock(c.Request.Context(), id)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toUserResponse(user))
}


// ResetPassword godoc
// @ID           resetPasswordUser
// @Summary      Reset user password
// @Description  Reset a user's password (admin action)
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id path string true "User ID" format(uuid)
// @Param        request body ResetPasswordRequest true "New password"
// @Success      200 {object} SuccessResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      422 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /identity/users/{id}/reset-password [post]
func (h *UserHandler) ResetPassword(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid user ID")
		return
	}

	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body")
		return
	}

	if err := h.userService.ResetPassword(c.Request.Context(), id, req.NewPassword); err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, dto.MessageResponse{Message: "Password reset successfully. User must change password on next login."})
}


// AssignRoles godoc
// @ID           assignRolesUser
// @Summary      Assign roles to a user
// @Description  Assign roles to a user (replaces existing roles)
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id path string true "User ID" format(uuid)
// @Param        request body AssignRolesRequest true "Role IDs"
// @Success      200 {object} APIResponse[UserResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      422 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /identity/users/{id}/roles [put]
func (h *UserHandler) AssignRoles(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid user ID")
		return
	}

	var req AssignRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body")
		return
	}

	// Parse role IDs
	roleIDs := make([]uuid.UUID, 0, len(req.RoleIDs))
	for _, ridStr := range req.RoleIDs {
		rid, err := uuid.Parse(ridStr)
		if err != nil {
			h.BadRequest(c, "Invalid role ID: "+ridStr)
			return
		}
		roleIDs = append(roleIDs, rid)
	}

	user, err := h.userService.AssignRoles(c.Request.Context(), id, roleIDs)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toUserResponse(user))
}


// Count godoc
// @ID           countUsers
// @Summary      Get user count
// @Description  Get the total number of users
// @Tags         users
// @Produce      json
// @Success      200 {object} APIResponse[CountData]
// @Failure      401 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /identity/users/stats/count [get]
func (h *UserHandler) Count(c *gin.Context) {
	count, err := h.userService.Count(c.Request.Context())
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, gin.H{"count": count})
}

// Helper functions for response conversion

func toUserResponse(user *identity.UserDTO) *UserResponse {
	roleIDStrings := make([]string, len(user.RoleIDs))
	for i, rid := range user.RoleIDs {
		roleIDStrings[i] = rid.String()
	}

	return &UserResponse{
		ID:          user.ID,
		TenantID:    user.TenantID,
		Username:    user.Username,
		Email:       user.Email,
		Phone:       user.Phone,
		DisplayName: user.DisplayName,
		Avatar:      user.Avatar,
		Status:      user.Status,
		RoleIDs:     roleIDStrings,
		LastLoginAt: user.LastLoginAt,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}

func toUserListResponse(result *identity.UserListResult) *UserListResponse {
	users := make([]UserResponse, len(result.Users))
	for i, user := range result.Users {
		users[i] = *toUserResponse(&user)
	}

	return &UserListResponse{
		Users:      users,
		Total:      result.Total,
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalPages: result.TotalPages,
	}
}
