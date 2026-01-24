package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GormRoleRepository implements RoleRepository using GORM
type GormRoleRepository struct {
	db *gorm.DB
}

// NewGormRoleRepository creates a new GormRoleRepository
func NewGormRoleRepository(db *gorm.DB) *GormRoleRepository {
	return &GormRoleRepository{db: db}
}

// Create creates a new role
func (r *GormRoleRepository) Create(ctx context.Context, role *identity.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

// Update updates an existing role
func (r *GormRoleRepository) Update(ctx context.Context, role *identity.Role) error {
	result := r.db.WithContext(ctx).Save(role)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// Delete deletes a role by ID
func (r *GormRoleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete role permissions
		if err := tx.Where("role_id = ?", id).Delete(&identity.RolePermission{}).Error; err != nil {
			return err
		}

		// Delete role data scopes
		if err := tx.Where("role_id = ?", id).Delete(&identity.RoleDataScope{}).Error; err != nil {
			return err
		}

		// Delete role
		result := tx.Delete(&identity.Role{}, "id = ?", id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return shared.ErrNotFound
		}
		return nil
	})
}

// FindByID finds a role by ID
func (r *GormRoleRepository) FindByID(ctx context.Context, id uuid.UUID) (*identity.Role, error) {
	var role identity.Role
	if err := r.db.WithContext(ctx).First(&role, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &role, nil
}

// FindByCode finds a role by code within a tenant
func (r *GormRoleRepository) FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*identity.Role, error) {
	var role identity.Role
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND UPPER(code) = ?", tenantID, strings.ToUpper(code)).
		First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &role, nil
}

// FindAll finds all roles for a tenant with optional filtering
func (r *GormRoleRepository) FindAll(ctx context.Context, tenantID uuid.UUID, filter *identity.RoleFilter) ([]*identity.Role, error) {
	var roles []*identity.Role
	query := r.db.WithContext(ctx).Model(&identity.Role{}).Where("tenant_id = ?", tenantID)

	query = r.applyFilter(query, filter)

	// Default sorting
	query = query.Order("sort_order ASC, name ASC")

	// Apply pagination
	if filter != nil {
		if filter.Limit > 0 {
			query = query.Limit(filter.Limit)
		}
		if filter.Page > 0 && filter.Limit > 0 {
			offset := (filter.Page - 1) * filter.Limit
			query = query.Offset(offset)
		}
	}

	if err := query.Find(&roles).Error; err != nil {
		return nil, err
	}

	return roles, nil
}

// Count counts roles matching the filter
func (r *GormRoleRepository) Count(ctx context.Context, tenantID uuid.UUID, filter *identity.RoleFilter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&identity.Role{}).Where("tenant_id = ?", tenantID)

	query = r.applyFilter(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ExistsByCode checks if a role with the given code exists in a tenant
func (r *GormRoleRepository) ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&identity.Role{}).
		Where("tenant_id = ? AND UPPER(code) = ?", tenantID, strings.ToUpper(code)).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ExistsByID checks if a role with the given ID exists
func (r *GormRoleRepository) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&identity.Role{}).
		Where("id = ?", id).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// FindByIDs finds multiple roles by IDs
func (r *GormRoleRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*identity.Role, error) {
	if len(ids) == 0 {
		return []*identity.Role{}, nil
	}

	var roles []*identity.Role
	if err := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// FindSystemRoles finds all system roles for a tenant
func (r *GormRoleRepository) FindSystemRoles(ctx context.Context, tenantID uuid.UUID) ([]*identity.Role, error) {
	var roles []*identity.Role
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND is_system_role = ?", tenantID, true).
		Order("sort_order ASC, name ASC").
		Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// SavePermissions saves all permissions for a role (replaces existing)
func (r *GormRoleRepository) SavePermissions(ctx context.Context, role *identity.Role) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete existing permissions
		if err := tx.Where("role_id = ?", role.ID).Delete(&identity.RolePermission{}).Error; err != nil {
			return err
		}

		// Insert new permissions
		if len(role.Permissions) > 0 {
			rolePerms := make([]identity.RolePermission, len(role.Permissions))
			for i, perm := range role.Permissions {
				rolePerms[i] = identity.RolePermission{
					RoleID:      role.ID,
					TenantID:    role.TenantID,
					Code:        perm.Code,
					Resource:    perm.Resource,
					Action:      perm.Action,
					Description: perm.Description,
					CreatedAt:   time.Now(),
				}
			}
			if err := tx.Create(&rolePerms).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// LoadPermissions loads permissions for a role
func (r *GormRoleRepository) LoadPermissions(ctx context.Context, role *identity.Role) error {
	var rolePerms []identity.RolePermission
	if err := r.db.WithContext(ctx).
		Where("role_id = ?", role.ID).
		Find(&rolePerms).Error; err != nil {
		return err
	}

	permissions := make([]identity.Permission, len(rolePerms))
	for i, rp := range rolePerms {
		permissions[i] = identity.Permission{
			Code:        rp.Code,
			Resource:    rp.Resource,
			Action:      rp.Action,
			Description: rp.Description,
		}
	}
	role.Permissions = permissions

	return nil
}

// SaveDataScopes saves all data scopes for a role (replaces existing)
func (r *GormRoleRepository) SaveDataScopes(ctx context.Context, role *identity.Role) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete existing data scopes
		if err := tx.Where("role_id = ?", role.ID).Delete(&identity.RoleDataScope{}).Error; err != nil {
			return err
		}

		// Insert new data scopes
		if len(role.DataScopes) > 0 {
			roleScopes := make([]identity.RoleDataScope, len(role.DataScopes))
			for i, scope := range role.DataScopes {
				scopeValues := ""
				if len(scope.ScopeValues) > 0 {
					bytes, _ := json.Marshal(scope.ScopeValues)
					scopeValues = string(bytes)
				}
				roleScopes[i] = identity.RoleDataScope{
					RoleID:      role.ID,
					TenantID:    role.TenantID,
					Resource:    scope.Resource,
					ScopeType:   scope.ScopeType,
					ScopeValues: scopeValues,
					Description: scope.Description,
					CreatedAt:   time.Now(),
				}
			}
			if err := tx.Create(&roleScopes).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// LoadDataScopes loads data scopes for a role
func (r *GormRoleRepository) LoadDataScopes(ctx context.Context, role *identity.Role) error {
	var roleScopes []identity.RoleDataScope
	if err := r.db.WithContext(ctx).
		Where("role_id = ?", role.ID).
		Find(&roleScopes).Error; err != nil {
		return err
	}

	dataScopes := make([]identity.DataScope, len(roleScopes))
	for i, rs := range roleScopes {
		var scopeValues []string
		if rs.ScopeValues != "" {
			_ = json.Unmarshal([]byte(rs.ScopeValues), &scopeValues)
		}
		dataScopes[i] = identity.DataScope{
			Resource:    rs.Resource,
			ScopeType:   rs.ScopeType,
			ScopeValues: scopeValues,
			Description: rs.Description,
		}
	}
	role.DataScopes = dataScopes

	return nil
}

// LoadPermissionsAndDataScopes loads both permissions and data scopes for a role
func (r *GormRoleRepository) LoadPermissionsAndDataScopes(ctx context.Context, role *identity.Role) error {
	if err := r.LoadPermissions(ctx, role); err != nil {
		return err
	}
	return r.LoadDataScopes(ctx, role)
}

// FindUsersWithRole finds all user IDs that have this role
func (r *GormRoleRepository) FindUsersWithRole(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error) {
	var userRoles []identity.UserRole
	if err := r.db.WithContext(ctx).
		Where("role_id = ?", roleID).
		Find(&userRoles).Error; err != nil {
		return nil, err
	}

	userIDs := make([]uuid.UUID, len(userRoles))
	for i, ur := range userRoles {
		userIDs[i] = ur.UserID
	}
	return userIDs, nil
}

// CountUsersWithRole counts how many users have this role
func (r *GormRoleRepository) CountUsersWithRole(ctx context.Context, roleID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&identity.UserRole{}).
		Where("role_id = ?", roleID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// FindRolesWithPermission finds all roles that have a specific permission
func (r *GormRoleRepository) FindRolesWithPermission(ctx context.Context, tenantID uuid.UUID, permissionCode string) ([]*identity.Role, error) {
	var rolePerms []identity.RolePermission
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND code = ?", tenantID, permissionCode).
		Find(&rolePerms).Error; err != nil {
		return nil, err
	}

	if len(rolePerms) == 0 {
		return []*identity.Role{}, nil
	}

	roleIDs := make([]uuid.UUID, len(rolePerms))
	for i, rp := range rolePerms {
		roleIDs[i] = rp.RoleID
	}

	return r.FindByIDs(ctx, roleIDs)
}

// GetAllPermissionCodes returns all distinct permission codes used in a tenant
func (r *GormRoleRepository) GetAllPermissionCodes(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	var codes []string
	if err := r.db.WithContext(ctx).
		Model(&identity.RolePermission{}).
		Where("tenant_id = ?", tenantID).
		Distinct("code").
		Pluck("code", &codes).Error; err != nil {
		return nil, err
	}
	return codes, nil
}

// applyFilter applies filter options to the query
func (r *GormRoleRepository) applyFilter(query *gorm.DB, filter *identity.RoleFilter) *gorm.DB {
	if filter == nil {
		return query
	}

	// Apply keyword search
	if filter.Keyword != "" {
		searchPattern := "%" + filter.Keyword + "%"
		query = query.Where("code ILIKE ? OR name ILIKE ?", searchPattern, searchPattern)
	}

	// Apply enabled filter
	if filter.IsEnabled != nil {
		query = query.Where("is_enabled = ?", *filter.IsEnabled)
	}

	// Apply system role filter
	if filter.IsSystemRole != nil {
		query = query.Where("is_system_role = ?", *filter.IsSystemRole)
	}

	return query
}

// Ensure GormRoleRepository implements RoleRepository
var _ identity.RoleRepository = (*GormRoleRepository)(nil)
