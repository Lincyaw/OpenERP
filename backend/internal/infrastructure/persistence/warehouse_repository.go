package persistence

import (
	"context"
	"errors"
	"strings"

	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GormWarehouseRepository implements WarehouseRepository using GORM
type GormWarehouseRepository struct {
	db *gorm.DB
}

// NewGormWarehouseRepository creates a new GormWarehouseRepository
func NewGormWarehouseRepository(db *gorm.DB) *GormWarehouseRepository {
	return &GormWarehouseRepository{db: db}
}

// FindByID finds a warehouse by its ID
func (r *GormWarehouseRepository) FindByID(ctx context.Context, id uuid.UUID) (*partner.Warehouse, error) {
	var model models.WarehouseModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByIDForTenant finds a warehouse by ID within a tenant
func (r *GormWarehouseRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*partner.Warehouse, error) {
	var model models.WarehouseModel
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

// FindByCode finds a warehouse by its code within a tenant
func (r *GormWarehouseRepository) FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*partner.Warehouse, error) {
	var model models.WarehouseModel
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

// FindAll finds all warehouses matching the filter
func (r *GormWarehouseRepository) FindAll(ctx context.Context, filter shared.Filter) ([]partner.Warehouse, error) {
	var warehouseModels []models.WarehouseModel
	query := r.applyFilter(r.db.WithContext(ctx).Model(&models.WarehouseModel{}), filter)

	if err := query.Find(&warehouseModels).Error; err != nil {
		return nil, err
	}

	warehouses := make([]partner.Warehouse, len(warehouseModels))
	for i, model := range warehouseModels {
		warehouses[i] = *model.ToDomain()
	}
	return warehouses, nil
}

// FindAllForTenant finds all warehouses for a tenant
func (r *GormWarehouseRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Warehouse, error) {
	var warehouseModels []models.WarehouseModel
	query := r.applyFilter(r.db.WithContext(ctx).Model(&models.WarehouseModel{}).Where("tenant_id = ?", tenantID), filter)

	if err := query.Find(&warehouseModels).Error; err != nil {
		return nil, err
	}

	warehouses := make([]partner.Warehouse, len(warehouseModels))
	for i, model := range warehouseModels {
		warehouses[i] = *model.ToDomain()
	}
	return warehouses, nil
}

// FindByType finds warehouses by type (physical/virtual/consign/transit)
func (r *GormWarehouseRepository) FindByType(ctx context.Context, tenantID uuid.UUID, warehouseType partner.WarehouseType, filter shared.Filter) ([]partner.Warehouse, error) {
	var warehouseModels []models.WarehouseModel
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&models.WarehouseModel{}).
			Where("tenant_id = ? AND type = ?", tenantID, warehouseType),
		filter,
	)

	if err := query.Find(&warehouseModels).Error; err != nil {
		return nil, err
	}

	warehouses := make([]partner.Warehouse, len(warehouseModels))
	for i, model := range warehouseModels {
		warehouses[i] = *model.ToDomain()
	}
	return warehouses, nil
}

// FindByStatus finds warehouses by status for a tenant
func (r *GormWarehouseRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status partner.WarehouseStatus, filter shared.Filter) ([]partner.Warehouse, error) {
	var warehouseModels []models.WarehouseModel
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&models.WarehouseModel{}).
			Where("tenant_id = ? AND status = ?", tenantID, status),
		filter,
	)

	if err := query.Find(&warehouseModels).Error; err != nil {
		return nil, err
	}

	warehouses := make([]partner.Warehouse, len(warehouseModels))
	for i, model := range warehouseModels {
		warehouses[i] = *model.ToDomain()
	}
	return warehouses, nil
}

// FindActive finds all active warehouses for a tenant
func (r *GormWarehouseRepository) FindActive(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Warehouse, error) {
	return r.FindByStatus(ctx, tenantID, partner.WarehouseStatusActive, filter)
}

// FindDefault finds the default warehouse for a tenant
func (r *GormWarehouseRepository) FindDefault(ctx context.Context, tenantID uuid.UUID) (*partner.Warehouse, error) {
	var model models.WarehouseModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND is_default = ?", tenantID, true).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByIDs finds multiple warehouses by their IDs
func (r *GormWarehouseRepository) FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]partner.Warehouse, error) {
	if len(ids) == 0 {
		return []partner.Warehouse{}, nil
	}

	var warehouseModels []models.WarehouseModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id IN ?", tenantID, ids).
		Find(&warehouseModels).Error; err != nil {
		return nil, err
	}

	warehouses := make([]partner.Warehouse, len(warehouseModels))
	for i, model := range warehouseModels {
		warehouses[i] = *model.ToDomain()
	}
	return warehouses, nil
}

// FindByCodes finds multiple warehouses by their codes
func (r *GormWarehouseRepository) FindByCodes(ctx context.Context, tenantID uuid.UUID, codes []string) ([]partner.Warehouse, error) {
	if len(codes) == 0 {
		return []partner.Warehouse{}, nil
	}

	upperCodes := make([]string, len(codes))
	for i, code := range codes {
		upperCodes[i] = strings.ToUpper(code)
	}

	var warehouseModels []models.WarehouseModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND code IN ?", tenantID, upperCodes).
		Find(&warehouseModels).Error; err != nil {
		return nil, err
	}

	warehouses := make([]partner.Warehouse, len(warehouseModels))
	for i, model := range warehouseModels {
		warehouses[i] = *model.ToDomain()
	}
	return warehouses, nil
}

// Save creates or updates a warehouse
func (r *GormWarehouseRepository) Save(ctx context.Context, warehouse *partner.Warehouse) error {
	model := models.WarehouseModelFromDomain(warehouse)
	return r.db.WithContext(ctx).Save(model).Error
}

// SaveBatch creates or updates multiple warehouses
func (r *GormWarehouseRepository) SaveBatch(ctx context.Context, warehouses []*partner.Warehouse) error {
	if len(warehouses) == 0 {
		return nil
	}
	warehouseModels := make([]*models.WarehouseModel, len(warehouses))
	for i, w := range warehouses {
		warehouseModels[i] = models.WarehouseModelFromDomain(w)
	}
	return r.db.WithContext(ctx).Save(warehouseModels).Error
}

// Delete deletes a warehouse
func (r *GormWarehouseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.WarehouseModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// DeleteForTenant deletes a warehouse within a tenant
func (r *GormWarehouseRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.WarehouseModel{}, "tenant_id = ? AND id = ?", tenantID, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// Count counts warehouses matching the filter
func (r *GormWarehouseRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.WarehouseModel{})
	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountForTenant counts warehouses for a tenant
func (r *GormWarehouseRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.WarehouseModel{}).Where("tenant_id = ?", tenantID)
	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByType counts warehouses by type for a tenant
func (r *GormWarehouseRepository) CountByType(ctx context.Context, tenantID uuid.UUID, warehouseType partner.WarehouseType) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.WarehouseModel{}).
		Where("tenant_id = ? AND type = ?", tenantID, warehouseType).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByStatus counts warehouses by status for a tenant
func (r *GormWarehouseRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status partner.WarehouseStatus) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.WarehouseModel{}).
		Where("tenant_id = ? AND status = ?", tenantID, status).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ExistsByCode checks if a warehouse with the given code exists in the tenant
func (r *GormWarehouseRepository) ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.WarehouseModel{}).
		Where("tenant_id = ? AND code = ?", tenantID, strings.ToUpper(code)).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ClearDefault clears the default flag for all warehouses in a tenant
func (r *GormWarehouseRepository) ClearDefault(ctx context.Context, tenantID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&models.WarehouseModel{}).
		Where("tenant_id = ? AND is_default = ?", tenantID, true).
		Update("is_default", false).Error
}

// applyFilter applies filter options to the query
func (r *GormWarehouseRepository) applyFilter(query *gorm.DB, filter shared.Filter) *gorm.DB {
	query = r.applyFilterWithoutPagination(query, filter)

	// Apply pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// Apply ordering with whitelist validation to prevent SQL injection
	if filter.OrderBy != "" {
		sortField := ValidateSortField(filter.OrderBy, WarehouseSortFields, "")
		if sortField != "" {
			sortOrder := ValidateSortOrder(filter.OrderDir)
			query = query.Order(sortField + " " + sortOrder)
		} else {
			// Default ordering if invalid field
			query = query.Order("is_default DESC, sort_order ASC, name ASC")
		}
	} else {
		// Default ordering: default warehouse first, then by sort_order and name
		query = query.Order("is_default DESC, sort_order ASC, name ASC")
	}

	return query
}

// applyFilterWithoutPagination applies filter options without pagination
func (r *GormWarehouseRepository) applyFilterWithoutPagination(query *gorm.DB, filter shared.Filter) *gorm.DB {
	// Apply search
	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		query = query.Where("name ILIKE ? OR code ILIKE ? OR address ILIKE ? OR city ILIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern)
	}

	// Apply additional filters
	for key, value := range filter.Filters {
		switch key {
		case "status":
			query = query.Where("status = ?", value)
		case "type":
			query = query.Where("type = ?", value)
		case "city":
			query = query.Where("city = ?", value)
		case "province":
			query = query.Where("province = ?", value)
		case "is_default":
			query = query.Where("is_default = ?", value)
		case "has_capacity":
			if value == true {
				query = query.Where("capacity > 0")
			} else {
				query = query.Where("capacity = 0")
			}
		}
	}

	return query
}

// Ensure GormWarehouseRepository implements WarehouseRepository
var _ partner.WarehouseRepository = (*GormWarehouseRepository)(nil)
