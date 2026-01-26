package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/persistence/models"
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
	model := models.RoleModelFromDomain(role)
	return r.db.WithContext(ctx).Create(model).Error
}

// Update updates an existing role
func (r *GormRoleRepository) Update(ctx context.Context, role *identity.Role) error {
	model := models.RoleModelFromDomain(role)
	result := r.db.WithContext(ctx).Save(model)
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
		if err := tx.Where("role_id = ?", id).Delete(&models.RolePermissionModel{}).Error; err != nil {
			return err
		}

		// Delete role data scopes
		if err := tx.Where("role_id = ?", id).Delete(&models.RoleDataScopeModel{}).Error; err != nil {
			return err
		}

		// Delete role
		result := tx.Delete(&models.RoleModel{}, "id = ?", id)
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
	var model models.RoleModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByCode finds a role by code within a tenant
func (r *GormRoleRepository) FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*identity.Role, error) {
	var model models.RoleModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND UPPER(code) = ?", tenantID, strings.ToUpper(code)).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindAll finds all roles for a tenant with optional filtering
func (r *GormRoleRepository) FindAll(ctx context.Context, tenantID uuid.UUID, filter *identity.RoleFilter) ([]*identity.Role, error) {
	var roleModels []*models.RoleModel
	query := r.db.WithContext(ctx).Model(&models.RoleModel{}).Where("tenant_id = ?", tenantID)

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

	if err := query.Find(&roleModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	roles := make([]*identity.Role, len(roleModels))
	for i, model := range roleModels {
		roles[i] = model.ToDomain()
	}

	return roles, nil
}

// Count counts roles matching the filter
func (r *GormRoleRepository) Count(ctx context.Context, tenantID uuid.UUID, filter *identity.RoleFilter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.RoleModel{}).Where("tenant_id = ?", tenantID)

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
		Model(&models.RoleModel{}).
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
		Model(&models.RoleModel{}).
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

	var roleModels []*models.RoleModel
	if err := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&roleModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	roles := make([]*identity.Role, len(roleModels))
	for i, model := range roleModels {
		roles[i] = model.ToDomain()
	}

	return roles, nil
}

// FindSystemRoles finds all system roles for a tenant
func (r *GormRoleRepository) FindSystemRoles(ctx context.Context, tenantID uuid.UUID) ([]*identity.Role, error) {
	var roleModels []*models.RoleModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND is_system_role = ?", tenantID, true).
		Order("sort_order ASC, name ASC").
		Find(&roleModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	roles := make([]*identity.Role, len(roleModels))
	for i, model := range roleModels {
		roles[i] = model.ToDomain()
	}

	return roles, nil
}

// SavePermissions saves all permissions for a role (replaces existing)
func (r *GormRoleRepository) SavePermissions(ctx context.Context, role *identity.Role) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete existing permissions
		if err := tx.Where("role_id = ?", role.ID).Delete(&models.RolePermissionModel{}).Error; err != nil {
			return err
		}

		// Insert new permissions
		if len(role.Permissions) > 0 {
			rolePermModels := make([]models.RolePermissionModel, len(role.Permissions))
			for i, perm := range role.Permissions {
				rolePermModels[i] = models.RolePermissionModel{
					RoleID:      role.ID,
					TenantID:    role.TenantID,
					Code:        perm.Code,
					Resource:    perm.Resource,
					Action:      perm.Action,
					Description: perm.Description,
					CreatedAt:   time.Now(),
				}
			}
			if err := tx.Create(&rolePermModels).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// LoadPermissions loads permissions for a role
func (r *GormRoleRepository) LoadPermissions(ctx context.Context, role *identity.Role) error {
	var rolePermModels []models.RolePermissionModel
	if err := r.db.WithContext(ctx).
		Where("role_id = ?", role.ID).
		Find(&rolePermModels).Error; err != nil {
		return err
	}

	permissions := make([]identity.Permission, len(rolePermModels))
	for i, model := range rolePermModels {
		permissions[i] = model.ToDomain()
	}
	role.Permissions = permissions

	return nil
}

// SaveDataScopes saves all data scopes for a role (replaces existing)
func (r *GormRoleRepository) SaveDataScopes(ctx context.Context, role *identity.Role) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete existing data scopes
		if err := tx.Where("role_id = ?", role.ID).Delete(&models.RoleDataScopeModel{}).Error; err != nil {
			return err
		}

		// Insert new data scopes
		if len(role.DataScopes) > 0 {
			roleScopeModels := make([]models.RoleDataScopeModel, len(role.DataScopes))
			for i, scope := range role.DataScopes {
				scopeValues := ""
				if len(scope.ScopeValues) > 0 {
					bytes, _ := json.Marshal(scope.ScopeValues)
					scopeValues = string(bytes)
				}
				roleScopeModels[i] = models.RoleDataScopeModel{
					RoleID:      role.ID,
					TenantID:    role.TenantID,
					Resource:    scope.Resource,
					ScopeType:   scope.ScopeType,
					ScopeField:  scope.ScopeField,
					ScopeValues: scopeValues,
					Description: scope.Description,
					CreatedAt:   time.Now(),
				}
			}
			if err := tx.Create(&roleScopeModels).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// LoadDataScopes loads data scopes for a role
func (r *GormRoleRepository) LoadDataScopes(ctx context.Context, role *identity.Role) error {
	var roleScopeModels []models.RoleDataScopeModel
	if err := r.db.WithContext(ctx).
		Where("role_id = ?", role.ID).
		Find(&roleScopeModels).Error; err != nil {
		return err
	}

	dataScopes := make([]identity.DataScope, len(roleScopeModels))
	for i, model := range roleScopeModels {
		var scopeValues []string
		if model.ScopeValues != "" {
			_ = json.Unmarshal([]byte(model.ScopeValues), &scopeValues)
		}
		dataScopes[i] = identity.DataScope{
			Resource:    model.Resource,
			ScopeType:   model.ScopeType,
			ScopeField:  model.ScopeField,
			ScopeValues: scopeValues,
			Description: model.Description,
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
	var userRoleModels []models.UserRoleModel
	if err := r.db.WithContext(ctx).
		Where("role_id = ?", roleID).
		Find(&userRoleModels).Error; err != nil {
		return nil, err
	}

	userIDs := make([]uuid.UUID, len(userRoleModels))
	for i, model := range userRoleModels {
		userIDs[i] = model.UserID
	}
	return userIDs, nil
}

// CountUsersWithRole counts how many users have this role
func (r *GormRoleRepository) CountUsersWithRole(ctx context.Context, roleID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.UserRoleModel{}).
		Where("role_id = ?", roleID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// FindRolesWithPermission finds all roles that have a specific permission
func (r *GormRoleRepository) FindRolesWithPermission(ctx context.Context, tenantID uuid.UUID, permissionCode string) ([]*identity.Role, error) {
	var rolePermModels []models.RolePermissionModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND code = ?", tenantID, permissionCode).
		Find(&rolePermModels).Error; err != nil {
		return nil, err
	}

	if len(rolePermModels) == 0 {
		return []*identity.Role{}, nil
	}

	roleIDs := make([]uuid.UUID, len(rolePermModels))
	for i, model := range rolePermModels {
		roleIDs[i] = model.RoleID
	}

	return r.FindByIDs(ctx, roleIDs)
}

// GetAllPermissionCodes returns all distinct permission codes used in a tenant
func (r *GormRoleRepository) GetAllPermissionCodes(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	var codes []string
	if err := r.db.WithContext(ctx).
		Model(&models.RolePermissionModel{}).
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
