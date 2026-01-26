package persistence

import (
	"context"
	"errors"
	"strings"

	"github.com/erp/backend/internal/domain/catalog"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GormCategoryRepository implements CategoryRepository using GORM
type GormCategoryRepository struct {
	db *gorm.DB
}

// NewGormCategoryRepository creates a new GormCategoryRepository
func NewGormCategoryRepository(db *gorm.DB) *GormCategoryRepository {
	return &GormCategoryRepository{db: db}
}

// FindByID finds a category by its ID
func (r *GormCategoryRepository) FindByID(ctx context.Context, id uuid.UUID) (*catalog.Category, error) {
	var model models.CategoryModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByIDForTenant finds a category by ID within a tenant
func (r *GormCategoryRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*catalog.Category, error) {
	var model models.CategoryModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByCode finds a category by its code within a tenant
func (r *GormCategoryRepository) FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*catalog.Category, error) {
	var model models.CategoryModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND code = ?", tenantID, strings.ToUpper(code)).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindAll finds all categories matching the filter
func (r *GormCategoryRepository) FindAll(ctx context.Context, filter shared.Filter) ([]catalog.Category, error) {
	var categoryModels []models.CategoryModel
	query := r.applyFilter(r.db.WithContext(ctx).Model(&models.CategoryModel{}), filter)

	if err := query.Find(&categoryModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	categories := make([]catalog.Category, len(categoryModels))
	for i, model := range categoryModels {
		categories[i] = *model.ToDomain()
	}
	return categories, nil
}

// FindAllForTenant finds all categories for a tenant
func (r *GormCategoryRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]catalog.Category, error) {
	var categoryModels []models.CategoryModel
	query := r.applyFilter(r.db.WithContext(ctx).Model(&models.CategoryModel{}).Where("tenant_id = ?", tenantID), filter)

	if err := query.Find(&categoryModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	categories := make([]catalog.Category, len(categoryModels))
	for i, model := range categoryModels {
		categories[i] = *model.ToDomain()
	}
	return categories, nil
}

// FindChildren finds all direct children of a category
func (r *GormCategoryRepository) FindChildren(ctx context.Context, tenantID, parentID uuid.UUID) ([]catalog.Category, error) {
	var categoryModels []models.CategoryModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND parent_id = ?", tenantID, parentID).
		Order("sort_order ASC, name ASC").
		Find(&categoryModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	categories := make([]catalog.Category, len(categoryModels))
	for i, model := range categoryModels {
		categories[i] = *model.ToDomain()
	}
	return categories, nil
}

// FindRootCategories finds all root categories for a tenant
func (r *GormCategoryRepository) FindRootCategories(ctx context.Context, tenantID uuid.UUID) ([]catalog.Category, error) {
	var categoryModels []models.CategoryModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND parent_id IS NULL", tenantID).
		Order("sort_order ASC, name ASC").
		Find(&categoryModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	categories := make([]catalog.Category, len(categoryModels))
	for i, model := range categoryModels {
		categories[i] = *model.ToDomain()
	}
	return categories, nil
}

// FindDescendants finds all descendants of a category (using materialized path)
func (r *GormCategoryRepository) FindDescendants(ctx context.Context, tenantID, categoryID uuid.UUID) ([]catalog.Category, error) {
	// First, get the category to obtain its path
	parent, err := r.FindByIDForTenant(ctx, tenantID, categoryID)
	if err != nil {
		return nil, err
	}

	var categoryModels []models.CategoryModel
	// Find all categories whose path starts with the parent's path followed by /
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND path LIKE ?", tenantID, parent.Path+"/%").
		Order("level ASC, sort_order ASC, name ASC").
		Find(&categoryModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	categories := make([]catalog.Category, len(categoryModels))
	for i, model := range categoryModels {
		categories[i] = *model.ToDomain()
	}
	return categories, nil
}

// FindByPath finds a category by its materialized path
func (r *GormCategoryRepository) FindByPath(ctx context.Context, tenantID uuid.UUID, path string) (*catalog.Category, error) {
	var model models.CategoryModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND path = ?", tenantID, path).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// Save creates or updates a category
func (r *GormCategoryRepository) Save(ctx context.Context, category *catalog.Category) error {
	model := models.CategoryModelFromDomain(category)
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete deletes a category
func (r *GormCategoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.CategoryModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// DeleteForTenant deletes a category within a tenant
func (r *GormCategoryRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.CategoryModel{}, "tenant_id = ? AND id = ?", tenantID, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// HasChildren checks if a category has any children
func (r *GormCategoryRepository) HasChildren(ctx context.Context, tenantID, categoryID uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.CategoryModel{}).
		Where("tenant_id = ? AND parent_id = ?", tenantID, categoryID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// HasProducts checks if a category has any associated products
// Note: This will be implemented properly when Product entity is available
func (r *GormCategoryRepository) HasProducts(ctx context.Context, tenantID, categoryID uuid.UUID) (bool, error) {
	// TODO: Implement when Product repository is available
	// For now, return false (no products)
	return false, nil
}

// Count counts categories matching the filter
func (r *GormCategoryRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.CategoryModel{})
	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountForTenant counts categories for a tenant
func (r *GormCategoryRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.CategoryModel{}).Where("tenant_id = ?", tenantID)
	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ExistsByCode checks if a category with the given code exists in the tenant
func (r *GormCategoryRepository) ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.CategoryModel{}).
		Where("tenant_id = ? AND code = ?", tenantID, strings.ToUpper(code)).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// UpdatePath updates the path for a category and its descendants
func (r *GormCategoryRepository) UpdatePath(ctx context.Context, tenantID, categoryID uuid.UUID, newPath string, levelDelta int) error {
	// Get the current category
	current, err := r.FindByIDForTenant(ctx, tenantID, categoryID)
	if err != nil {
		return err
	}

	oldPath := current.Path

	// Update the category itself
	if err := r.db.WithContext(ctx).
		Model(&models.CategoryModel{}).
		Where("id = ?", categoryID).
		Updates(map[string]interface{}{
			"path":  newPath,
			"level": gorm.Expr("level + ?", levelDelta),
		}).Error; err != nil {
		return err
	}

	// Update all descendants
	// Replace the old path prefix with the new path prefix
	if err := r.db.WithContext(ctx).
		Model(&models.CategoryModel{}).
		Where("tenant_id = ? AND path LIKE ?", tenantID, oldPath+"/%").
		Updates(map[string]interface{}{
			"path":  gorm.Expr("REPLACE(path, ?, ?)", oldPath, newPath),
			"level": gorm.Expr("level + ?", levelDelta),
		}).Error; err != nil {
		return err
	}

	return nil
}

// applyFilter applies filter options to the query
func (r *GormCategoryRepository) applyFilter(query *gorm.DB, filter shared.Filter) *gorm.DB {
	query = r.applyFilterWithoutPagination(query, filter)

	// Apply pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// Apply ordering
	if filter.OrderBy != "" {
		orderDir := "ASC"
		if strings.ToLower(filter.OrderDir) == "desc" {
			orderDir = "DESC"
		}
		query = query.Order(filter.OrderBy + " " + orderDir)
	}

	return query
}

// applyFilterWithoutPagination applies filter options without pagination
func (r *GormCategoryRepository) applyFilterWithoutPagination(query *gorm.DB, filter shared.Filter) *gorm.DB {
	// Apply search
	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		query = query.Where("name ILIKE ? OR code ILIKE ?", searchPattern, searchPattern)
	}

	// Apply additional filters
	for key, value := range filter.Filters {
		switch key {
		case "status":
			query = query.Where("status = ?", value)
		case "parent_id":
			if value == nil {
				query = query.Where("parent_id IS NULL")
			} else {
				query = query.Where("parent_id = ?", value)
			}
		case "level":
			query = query.Where("level = ?", value)
		}
	}

	return query
}

// Ensure GormCategoryRepository implements CategoryRepository
var _ catalog.CategoryRepository = (*GormCategoryRepository)(nil)
