package identity

import (
	"context"

	"github.com/google/uuid"
)

// RoleFilter defines the filter criteria for role queries
type RoleFilter struct {
	Keyword      string // Search in code and name
	IsEnabled    *bool  // Filter by enabled status
	IsSystemRole *bool  // Filter by system role flag
	// Pagination
	Page  int
	Limit int
}

// RoleRepository defines the interface for role persistence operations
type RoleRepository interface {
	// Create creates a new role
	Create(ctx context.Context, role *Role) error

	// Update updates an existing role
	Update(ctx context.Context, role *Role) error

	// Delete deletes a role by ID
	Delete(ctx context.Context, id uuid.UUID) error

	// FindByID finds a role by ID
	FindByID(ctx context.Context, id uuid.UUID) (*Role, error)

	// FindByCode finds a role by code within a tenant
	FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*Role, error)

	// FindAll finds all roles for a tenant with optional filtering
	FindAll(ctx context.Context, tenantID uuid.UUID, filter *RoleFilter) ([]*Role, error)

	// Count counts roles matching the filter
	Count(ctx context.Context, tenantID uuid.UUID, filter *RoleFilter) (int64, error)

	// ExistsByCode checks if a role with the given code exists in a tenant
	ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error)

	// ExistsByID checks if a role with the given ID exists
	ExistsByID(ctx context.Context, id uuid.UUID) (bool, error)

	// FindByIDs finds multiple roles by IDs
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*Role, error)

	// FindSystemRoles finds all system roles for a tenant
	FindSystemRoles(ctx context.Context, tenantID uuid.UUID) ([]*Role, error)

	// SavePermissions saves all permissions for a role (replaces existing)
	SavePermissions(ctx context.Context, role *Role) error

	// LoadPermissions loads permissions for a role
	LoadPermissions(ctx context.Context, role *Role) error

	// SaveDataScopes saves all data scopes for a role (replaces existing)
	SaveDataScopes(ctx context.Context, role *Role) error

	// LoadDataScopes loads data scopes for a role
	LoadDataScopes(ctx context.Context, role *Role) error

	// LoadPermissionsAndDataScopes loads both permissions and data scopes for a role
	LoadPermissionsAndDataScopes(ctx context.Context, role *Role) error

	// FindUsersWithRole finds all user IDs that have this role
	FindUsersWithRole(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error)

	// CountUsersWithRole counts how many users have this role
	CountUsersWithRole(ctx context.Context, roleID uuid.UUID) (int64, error)

	// FindRolesWithPermission finds all roles that have a specific permission
	FindRolesWithPermission(ctx context.Context, tenantID uuid.UUID, permissionCode string) ([]*Role, error)

	// GetAllPermissionCodes returns all distinct permission codes used in a tenant
	GetAllPermissionCodes(ctx context.Context, tenantID uuid.UUID) ([]string, error)
}

// RoleWithUserCount represents a role with the count of users assigned to it
type RoleWithUserCount struct {
	Role      *Role
	UserCount int64
}

// PermissionSummary represents a summary of permission usage
type PermissionSummary struct {
	Code       string
	Resource   string
	Action     string
	RoleCount  int64 // Number of roles that have this permission
	RoleCodes  []string
}
