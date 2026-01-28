package handler

import (
	"time"

	"github.com/google/uuid"
)

// =====================
// User Request DTOs
// =====================

// CreateUserRequest represents the request body for creating a user
// @Name HandlerCreateUserRequest
type CreateUserRequest struct {
	Username    string   `json:"username" binding:"required,min=3,max=100"`
	Password    string   `json:"password" binding:"required,min=8,max=128"`
	Email       string   `json:"email" binding:"omitempty,email,max=200"`
	Phone       string   `json:"phone" binding:"omitempty,max=50"`
	DisplayName string   `json:"display_name" binding:"omitempty,max=200"`
	Notes       string   `json:"notes" binding:"omitempty"`
	RoleIDs     []string `json:"role_ids" binding:"omitempty"`
}

// UpdateUserRequest represents the request body for updating a user
// @Name HandlerUpdateUserRequest
type UpdateUserRequest struct {
	Email       *string `json:"email" binding:"omitempty,email,max=200"`
	Phone       *string `json:"phone" binding:"omitempty,max=50"`
	DisplayName *string `json:"display_name" binding:"omitempty,max=200"`
	Notes       *string `json:"notes" binding:"omitempty"`
}

// ResetPasswordRequest represents the request body for resetting a user's password
// @Name HandlerResetPasswordRequest
type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=8,max=128"`
}

// AssignRolesRequest represents the request body for assigning roles to a user
// @Name HandlerAssignRolesRequest
type AssignRolesRequest struct {
	RoleIDs []string `json:"role_ids" binding:"required"`
}

// LockUserRequest represents the request body for locking a user
// @Name HandlerLockUserRequest
type LockUserRequest struct {
	DurationMinutes int `json:"duration_minutes" binding:"omitempty,min=1"`
}

// UserListQuery represents query parameters for listing users
// @Name HandlerUserListQuery
type UserListQuery struct {
	Keyword  string `form:"keyword" binding:"omitempty"`
	Status   string `form:"status" binding:"omitempty,oneof=pending active locked deactivated"`
	RoleID   string `form:"role_id" binding:"omitempty,uuid"`
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	SortBy   string `form:"sort_by" binding:"omitempty,oneof=username email display_name created_at updated_at last_login_at"`
	SortDir  string `form:"sort_dir" binding:"omitempty,oneof=asc desc"`
}

// =====================
// User Response DTOs
// =====================

// UserResponse represents a user in API responses
// @Name HandlerUserListQuery
type UserResponse struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	Username    string     `json:"username"`
	Email       string     `json:"email,omitempty"`
	Phone       string     `json:"phone,omitempty"`
	DisplayName string     `json:"display_name"`
	Avatar      string     `json:"avatar,omitempty"`
	Status      string     `json:"status"`
	RoleIDs     []string   `json:"role_ids"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// UserListResponse represents a paginated list of users
// @Name HandlerUserListQuery
type UserListResponse struct {
	Users      []UserResponse `json:"users"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

// =====================
// Role Request DTOs
// =====================

// CreateRoleRequest represents the request body for creating a role
// @Name HandlerCreateRoleRequest
type CreateRoleRequest struct {
	Code        string   `json:"code" binding:"required,min=2,max=50"`
	Name        string   `json:"name" binding:"required,min=1,max=100"`
	Description string   `json:"description" binding:"omitempty"`
	Permissions []string `json:"permissions" binding:"omitempty"`
	SortOrder   int      `json:"sort_order" binding:"omitempty"`
}

// UpdateRoleRequest represents the request body for updating a role
// @Name HandlerUpdateRoleRequest
type UpdateRoleRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=1,max=100"`
	Description *string `json:"description" binding:"omitempty"`
	SortOrder   *int    `json:"sort_order" binding:"omitempty"`
}

// SetPermissionsRequest represents the request body for setting role permissions
// @Name HandlerUpdateRoleRequest
type SetPermissionsRequest struct {
	Permissions []string `json:"permissions" binding:"required"`
}

// RoleListQuery represents query parameters for listing roles
// @Name HandlerUpdateRoleRequest
type RoleListQuery struct {
	Keyword      string `form:"keyword" binding:"omitempty"`
	IsEnabled    *bool  `form:"is_enabled" binding:"omitempty"`
	IsSystemRole *bool  `form:"is_system_role" binding:"omitempty"`
	Page         int    `form:"page" binding:"omitempty,min=1"`
	PageSize     int    `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// =====================
// Role Response DTOs
// =====================

// RoleResponse represents a role in API responses
// @Name HandlerRoleResponse
type RoleResponse struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	IsSystemRole bool      `json:"is_system_role"`
	IsEnabled    bool      `json:"is_enabled"`
	SortOrder    int       `json:"sort_order"`
	Permissions  []string  `json:"permissions"`
	UserCount    int64     `json:"user_count,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// RoleListResponse represents a paginated list of roles
// @Name HandlerRoleResponse
type RoleListResponse struct {
	Roles      []RoleResponse `json:"roles"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

// PermissionResponse represents a permission in API responses
// @Name HandlerRoleResponse
type PermissionResponse struct {
	Code        string `json:"code"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Description string `json:"description,omitempty"`
}

// PermissionListResponse represents a list of permissions
// @Name HandlerPermissionListResponse
type PermissionListResponse struct {
	Permissions []string `json:"permissions"`
}
