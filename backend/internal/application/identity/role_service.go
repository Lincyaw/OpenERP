package identity

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RoleService handles role management operations
type RoleService struct {
	roleRepo identity.RoleRepository
	userRepo identity.UserRepository
	logger   *zap.Logger
}

// NewRoleService creates a new role service
func NewRoleService(
	roleRepo identity.RoleRepository,
	userRepo identity.UserRepository,
	logger *zap.Logger,
) *RoleService {
	return &RoleService{
		roleRepo: roleRepo,
		userRepo: userRepo,
		logger:   logger,
	}
}

// CreateRoleInput contains input for creating a role
type CreateRoleInput struct {
	TenantID    uuid.UUID
	Code        string
	Name        string
	Description string
	Permissions []string // Permission codes like "product:create"
	SortOrder   int
}

// UpdateRoleInput contains input for updating a role
type UpdateRoleInput struct {
	ID          uuid.UUID
	Name        *string
	Description *string
	SortOrder   *int
}

// RoleDTO represents role data transfer object
type RoleDTO struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	Code         string     `json:"code"`
	Name         string     `json:"name"`
	Description  string     `json:"description,omitempty"`
	IsSystemRole bool       `json:"is_system_role"`
	IsEnabled    bool       `json:"is_enabled"`
	SortOrder    int        `json:"sort_order"`
	Permissions  []string   `json:"permissions"`
	UserCount    int64      `json:"user_count,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// RoleListResult represents paginated role list result
type RoleListResult struct {
	Roles      []RoleDTO `json:"roles"`
	Total      int64     `json:"total"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
	TotalPages int       `json:"total_pages"`
}

// Create creates a new role
func (s *RoleService) Create(ctx context.Context, input CreateRoleInput) (*RoleDTO, error) {
	s.logger.Info("Creating new role",
		zap.String("code", input.Code),
		zap.String("tenant_id", input.TenantID.String()))

	// Check if code already exists
	exists, err := s.roleRepo.ExistsByCode(ctx, input.TenantID, input.Code)
	if err != nil {
		s.logger.Error("Failed to check role code existence", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to check role code availability")
	}
	if exists {
		return nil, shared.NewDomainError("ROLE_CODE_EXISTS", "Role code already exists")
	}

	// Create role
	role, err := identity.NewRole(input.TenantID, input.Code, input.Name)
	if err != nil {
		return nil, err
	}

	// Set optional fields
	if input.Description != "" {
		role.SetDescription(input.Description)
	}
	if input.SortOrder != 0 {
		role.SetSortOrder(input.SortOrder)
	}

	// Add permissions
	for _, permCode := range input.Permissions {
		if err := role.GrantPermissionByCode(permCode); err != nil {
			// Skip duplicate errors silently
			if domainErr, ok := err.(*shared.DomainError); ok && domainErr.Code == "PERMISSION_ALREADY_GRANTED" {
				continue
			}
			return nil, err
		}
	}

	// Save role
	if err := s.roleRepo.Create(ctx, role); err != nil {
		s.logger.Error("Failed to create role", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to create role")
	}

	// Save permissions
	if len(role.Permissions) > 0 {
		if err := s.roleRepo.SavePermissions(ctx, role); err != nil {
			s.logger.Error("Failed to save role permissions", zap.Error(err))
			// Delete the role if permission assignment failed
			_ = s.roleRepo.Delete(ctx, role.ID)
			return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to save role permissions")
		}
	}

	s.logger.Info("Role created successfully",
		zap.String("role_id", role.ID.String()),
		zap.String("code", role.Code))

	return toRoleDTO(role), nil
}

// GetByID retrieves a role by ID
func (s *RoleService) GetByID(ctx context.Context, id uuid.UUID) (*RoleDTO, error) {
	role, err := s.roleRepo.FindByID(ctx, id)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("ROLE_NOT_FOUND", "Role not found")
		}
		s.logger.Error("Failed to find role", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find role")
	}

	// Load permissions
	if err := s.roleRepo.LoadPermissions(ctx, role); err != nil {
		s.logger.Error("Failed to load role permissions", zap.Error(err))
	}

	dto := toRoleDTO(role)

	// Get user count
	userCount, err := s.roleRepo.CountUsersWithRole(ctx, id)
	if err == nil {
		dto.UserCount = userCount
	}

	return dto, nil
}

// GetByCode retrieves a role by code
func (s *RoleService) GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*RoleDTO, error) {
	role, err := s.roleRepo.FindByCode(ctx, tenantID, code)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("ROLE_NOT_FOUND", "Role not found")
		}
		s.logger.Error("Failed to find role", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find role")
	}

	// Load permissions
	if err := s.roleRepo.LoadPermissions(ctx, role); err != nil {
		s.logger.Error("Failed to load role permissions", zap.Error(err))
	}

	dto := toRoleDTO(role)

	// Get user count
	userCount, err := s.roleRepo.CountUsersWithRole(ctx, role.ID)
	if err == nil {
		dto.UserCount = userCount
	}

	return dto, nil
}

// List retrieves a paginated list of roles
func (s *RoleService) List(ctx context.Context, tenantID uuid.UUID, filter *identity.RoleFilter) (*RoleListResult, error) {
	roles, err := s.roleRepo.FindAll(ctx, tenantID, filter)
	if err != nil {
		s.logger.Error("Failed to list roles", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to list roles")
	}

	total, err := s.roleRepo.Count(ctx, tenantID, filter)
	if err != nil {
		s.logger.Error("Failed to count roles", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to count roles")
	}

	// Load permissions for each role
	for _, role := range roles {
		if err := s.roleRepo.LoadPermissions(ctx, role); err != nil {
			s.logger.Error("Failed to load role permissions",
				zap.String("role_id", role.ID.String()),
				zap.Error(err))
		}
	}

	// Calculate pagination
	pageSize := 20
	page := 1
	if filter != nil {
		if filter.Limit > 0 {
			pageSize = filter.Limit
		}
		if filter.Page > 0 {
			page = filter.Page
		}
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	roleDTOs := make([]RoleDTO, len(roles))
	for i, role := range roles {
		dto := toRoleDTO(role)
		// Get user count for each role
		userCount, err := s.roleRepo.CountUsersWithRole(ctx, role.ID)
		if err == nil {
			dto.UserCount = userCount
		}
		roleDTOs[i] = *dto
	}

	return &RoleListResult{
		Roles:      roleDTOs,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// Update updates a role's information
func (s *RoleService) Update(ctx context.Context, input UpdateRoleInput) (*RoleDTO, error) {
	role, err := s.roleRepo.FindByID(ctx, input.ID)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("ROLE_NOT_FOUND", "Role not found")
		}
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find role")
	}

	// Update fields
	if input.Name != nil {
		if err := role.SetName(*input.Name); err != nil {
			return nil, err
		}
	}

	if input.Description != nil {
		role.SetDescription(*input.Description)
	}

	if input.SortOrder != nil {
		role.SetSortOrder(*input.SortOrder)
	}

	if err := s.roleRepo.Update(ctx, role); err != nil {
		s.logger.Error("Failed to update role", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to update role")
	}

	// Load permissions
	if err := s.roleRepo.LoadPermissions(ctx, role); err != nil {
		s.logger.Error("Failed to load role permissions", zap.Error(err))
	}

	s.logger.Info("Role updated", zap.String("role_id", input.ID.String()))

	return toRoleDTO(role), nil
}

// Delete deletes a role
func (s *RoleService) Delete(ctx context.Context, id uuid.UUID) error {
	role, err := s.roleRepo.FindByID(ctx, id)
	if err != nil {
		if err == shared.ErrNotFound {
			return shared.NewDomainError("ROLE_NOT_FOUND", "Role not found")
		}
		return shared.NewDomainError("INTERNAL_ERROR", "Failed to find role")
	}

	// Check if role can be deleted
	if !role.CanDelete() {
		return shared.NewDomainError("CANNOT_DELETE_SYSTEM_ROLE", "System roles cannot be deleted")
	}

	// Check if any users have this role
	userCount, err := s.roleRepo.CountUsersWithRole(ctx, id)
	if err != nil {
		s.logger.Error("Failed to count users with role", zap.Error(err))
		return shared.NewDomainError("INTERNAL_ERROR", "Failed to check role usage")
	}
	if userCount > 0 {
		return shared.NewDomainError("ROLE_IN_USE", "Cannot delete role that is assigned to users")
	}

	if err := s.roleRepo.Delete(ctx, id); err != nil {
		s.logger.Error("Failed to delete role", zap.Error(err))
		return shared.NewDomainError("INTERNAL_ERROR", "Failed to delete role")
	}

	s.logger.Info("Role deleted", zap.String("role_id", id.String()))

	return nil
}

// Enable enables a role
func (s *RoleService) Enable(ctx context.Context, id uuid.UUID) (*RoleDTO, error) {
	role, err := s.roleRepo.FindByID(ctx, id)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("ROLE_NOT_FOUND", "Role not found")
		}
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find role")
	}

	if err := role.Enable(); err != nil {
		return nil, err
	}

	if err := s.roleRepo.Update(ctx, role); err != nil {
		s.logger.Error("Failed to enable role", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to enable role")
	}

	// Load permissions
	if err := s.roleRepo.LoadPermissions(ctx, role); err != nil {
		s.logger.Error("Failed to load role permissions", zap.Error(err))
	}

	s.logger.Info("Role enabled", zap.String("role_id", id.String()))

	return toRoleDTO(role), nil
}

// Disable disables a role
func (s *RoleService) Disable(ctx context.Context, id uuid.UUID) (*RoleDTO, error) {
	role, err := s.roleRepo.FindByID(ctx, id)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("ROLE_NOT_FOUND", "Role not found")
		}
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find role")
	}

	if err := role.Disable(); err != nil {
		return nil, err
	}

	if err := s.roleRepo.Update(ctx, role); err != nil {
		s.logger.Error("Failed to disable role", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to disable role")
	}

	// Load permissions
	if err := s.roleRepo.LoadPermissions(ctx, role); err != nil {
		s.logger.Error("Failed to load role permissions", zap.Error(err))
	}

	s.logger.Info("Role disabled", zap.String("role_id", id.String()))

	return toRoleDTO(role), nil
}

// SetPermissions sets permissions for a role (replaces existing permissions)
func (s *RoleService) SetPermissions(ctx context.Context, roleID uuid.UUID, permissionCodes []string) (*RoleDTO, error) {
	role, err := s.roleRepo.FindByID(ctx, roleID)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("ROLE_NOT_FOUND", "Role not found")
		}
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find role")
	}

	// Build new permission list
	permissions := make([]identity.Permission, 0, len(permissionCodes))
	for _, code := range permissionCodes {
		perm, err := identity.NewPermissionFromCode(code)
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, *perm)
	}

	// Set permissions
	if err := role.SetPermissions(permissions); err != nil {
		return nil, err
	}

	// Save permissions
	if err := s.roleRepo.SavePermissions(ctx, role); err != nil {
		s.logger.Error("Failed to save role permissions", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to save permissions")
	}

	// Update role version
	if err := s.roleRepo.Update(ctx, role); err != nil {
		s.logger.Error("Failed to update role", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to update role")
	}

	s.logger.Info("Role permissions updated",
		zap.String("role_id", roleID.String()),
		zap.Int("permission_count", len(permissions)))

	return toRoleDTO(role), nil
}

// GetAllPermissionCodes returns all available permission codes for a tenant
func (s *RoleService) GetAllPermissionCodes(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	return s.roleRepo.GetAllPermissionCodes(ctx, tenantID)
}

// GetSystemRoles returns all system roles for a tenant
func (s *RoleService) GetSystemRoles(ctx context.Context, tenantID uuid.UUID) ([]RoleDTO, error) {
	roles, err := s.roleRepo.FindSystemRoles(ctx, tenantID)
	if err != nil {
		s.logger.Error("Failed to find system roles", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find system roles")
	}

	roleDTOs := make([]RoleDTO, len(roles))
	for i, role := range roles {
		if err := s.roleRepo.LoadPermissions(ctx, role); err != nil {
			s.logger.Error("Failed to load role permissions", zap.Error(err))
		}
		roleDTOs[i] = *toRoleDTO(role)
	}

	return roleDTOs, nil
}

// Count returns the total number of roles for a tenant
func (s *RoleService) Count(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	return s.roleRepo.Count(ctx, tenantID, nil)
}

// toRoleDTO converts domain Role to RoleDTO
func toRoleDTO(role *identity.Role) *RoleDTO {
	permissions := make([]string, len(role.Permissions))
	for i, perm := range role.Permissions {
		permissions[i] = perm.Code
	}

	return &RoleDTO{
		ID:           role.ID,
		TenantID:     role.TenantID,
		Code:         role.Code,
		Name:         role.Name,
		Description:  role.Description,
		IsSystemRole: role.IsSystemRole,
		IsEnabled:    role.IsEnabled,
		SortOrder:    role.SortOrder,
		Permissions:  permissions,
		CreatedAt:    role.CreatedAt,
		UpdatedAt:    role.UpdatedAt,
	}
}
