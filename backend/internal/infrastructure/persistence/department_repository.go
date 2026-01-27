package persistence

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GormDepartmentRepository implements DepartmentRepository using GORM
type GormDepartmentRepository struct {
	db *gorm.DB
}

// NewGormDepartmentRepository creates a new GormDepartmentRepository
func NewGormDepartmentRepository(db *gorm.DB) *GormDepartmentRepository {
	return &GormDepartmentRepository{db: db}
}

// Create saves a new department
func (r *GormDepartmentRepository) Create(ctx context.Context, dept *identity.Department) error {
	metadataJSON, err := json.Marshal(dept.Metadata)
	if err != nil {
		return err
	}

	model := models.DepartmentModelFromDomain(dept, string(metadataJSON))

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return err
	}

	// Update path after creation with the new ID
	if dept.ParentID == nil {
		dept.Path = "/" + dept.ID.String()
	}

	return nil
}

// Update updates an existing department
func (r *GormDepartmentRepository) Update(ctx context.Context, dept *identity.Department) error {
	metadataJSON, err := json.Marshal(dept.Metadata)
	if err != nil {
		return err
	}

	model := models.DepartmentModelFromDomain(dept, string(metadataJSON))

	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// Delete removes a department by ID
func (r *GormDepartmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Check if there are any child departments
	var childCount int64
	if err := r.db.WithContext(ctx).
		Model(&models.DepartmentModel{}).
		Where("parent_id = ?", id).
		Count(&childCount).Error; err != nil {
		return err
	}
	if childCount > 0 {
		return shared.NewDomainError("HAS_CHILDREN", "Cannot delete department with child departments")
	}

	// Check if there are any users in this department
	var userCount int64
	if err := r.db.WithContext(ctx).
		Model(&models.UserModel{}).
		Where("department_id = ?", id).
		Count(&userCount).Error; err != nil {
		return err
	}
	if userCount > 0 {
		return shared.NewDomainError("HAS_USERS", "Cannot delete department with assigned users")
	}

	result := r.db.WithContext(ctx).Delete(&models.DepartmentModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// FindByID finds a department by ID
func (r *GormDepartmentRepository) FindByID(ctx context.Context, id uuid.UUID) (*identity.Department, error) {
	var model models.DepartmentModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return r.modelToDomain(&model)
}

// FindByCode finds a department by code within a tenant
func (r *GormDepartmentRepository) FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*identity.Department, error) {
	var model models.DepartmentModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND code = ?", tenantID, code).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return r.modelToDomain(&model)
}

// FindByTenantID finds all departments for a tenant
func (r *GormDepartmentRepository) FindByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*identity.Department, error) {
	var deptModels []*models.DepartmentModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("level ASC, sort_order ASC, name ASC").
		Find(&deptModels).Error; err != nil {
		return nil, err
	}

	return r.modelsToDomain(deptModels)
}

// FindChildren finds all direct children of a department
func (r *GormDepartmentRepository) FindChildren(ctx context.Context, parentID uuid.UUID) ([]*identity.Department, error) {
	var deptModels []*models.DepartmentModel
	if err := r.db.WithContext(ctx).
		Where("parent_id = ?", parentID).
		Order("sort_order ASC, name ASC").
		Find(&deptModels).Error; err != nil {
		return nil, err
	}

	return r.modelsToDomain(deptModels)
}

// FindDescendants finds all descendants of a department (using materialized path)
func (r *GormDepartmentRepository) FindDescendants(ctx context.Context, dept *identity.Department) ([]*identity.Department, error) {
	var deptModels []*models.DepartmentModel

	// Use path prefix matching to find all descendants
	// Pattern: "/parent-path/%" matches all descendants
	pathPattern := dept.Path + "/%"

	if err := r.db.WithContext(ctx).
		Where("path LIKE ?", pathPattern).
		Order("level ASC, sort_order ASC, name ASC").
		Find(&deptModels).Error; err != nil {
		return nil, err
	}

	return r.modelsToDomain(deptModels)
}

// FindByIDs finds departments by multiple IDs
func (r *GormDepartmentRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*identity.Department, error) {
	if len(ids) == 0 {
		return []*identity.Department{}, nil
	}

	var deptModels []*models.DepartmentModel
	if err := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&deptModels).Error; err != nil {
		return nil, err
	}

	return r.modelsToDomain(deptModels)
}

// FindRootDepartments finds all root departments (no parent) for a tenant
func (r *GormDepartmentRepository) FindRootDepartments(ctx context.Context, tenantID uuid.UUID) ([]*identity.Department, error) {
	var deptModels []*models.DepartmentModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND parent_id IS NULL", tenantID).
		Order("sort_order ASC, name ASC").
		Find(&deptModels).Error; err != nil {
		return nil, err
	}

	return r.modelsToDomain(deptModels)
}

// FindByManagerID finds departments managed by a specific user
func (r *GormDepartmentRepository) FindByManagerID(ctx context.Context, managerID uuid.UUID) ([]*identity.Department, error) {
	var deptModels []*models.DepartmentModel
	if err := r.db.WithContext(ctx).
		Where("manager_id = ?", managerID).
		Find(&deptModels).Error; err != nil {
		return nil, err
	}

	return r.modelsToDomain(deptModels)
}

// CountByTenantID counts departments for a tenant
func (r *GormDepartmentRepository) CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.DepartmentModel{}).
		Where("tenant_id = ?", tenantID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ExistsByCode checks if a department code exists within a tenant
func (r *GormDepartmentRepository) ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.DepartmentModel{}).
		Where("tenant_id = ? AND code = ?", tenantID, code).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetAllDepartmentIDsInSubtree returns all department IDs in a subtree (including the root)
// This is used for department-scoped data filtering
func (r *GormDepartmentRepository) GetAllDepartmentIDsInSubtree(ctx context.Context, departmentID uuid.UUID) ([]uuid.UUID, error) {
	// First, get the department to get its path
	dept, err := r.FindByID(ctx, departmentID)
	if err != nil {
		return nil, err
	}

	// Find all departments with paths starting with the current department's path
	var deptModels []*models.DepartmentModel
	pathPattern := dept.Path + "%"

	if err := r.db.WithContext(ctx).
		Select("id").
		Where("path LIKE ?", pathPattern).
		Find(&deptModels).Error; err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, len(deptModels))
	for i, model := range deptModels {
		ids[i] = model.ID
	}

	return ids, nil
}

// modelToDomain converts a single model to domain entity with metadata parsing
func (r *GormDepartmentRepository) modelToDomain(model *models.DepartmentModel) (*identity.Department, error) {
	dept := model.ToDomain()

	// Parse metadata JSON
	if model.Metadata != "" && model.Metadata != "{}" {
		var metadata map[string]string
		if err := json.Unmarshal([]byte(model.Metadata), &metadata); err != nil {
			return nil, err
		}
		dept.Metadata = metadata
	}

	return dept, nil
}

// modelsToDomain converts multiple models to domain entities
func (r *GormDepartmentRepository) modelsToDomain(deptModels []*models.DepartmentModel) ([]*identity.Department, error) {
	departments := make([]*identity.Department, len(deptModels))
	for i, model := range deptModels {
		dept, err := r.modelToDomain(model)
		if err != nil {
			return nil, err
		}
		departments[i] = dept
	}
	return departments, nil
}

// Ensure GormDepartmentRepository implements DepartmentRepository
var _ identity.DepartmentRepository = (*GormDepartmentRepository)(nil)
