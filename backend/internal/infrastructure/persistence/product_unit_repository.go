package persistence

import (
	"context"
	"errors"

	"github.com/erp/backend/internal/domain/catalog"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GormProductUnitRepository implements ProductUnitRepository using GORM
type GormProductUnitRepository struct {
	db *gorm.DB
}

// NewGormProductUnitRepository creates a new GormProductUnitRepository
func NewGormProductUnitRepository(db *gorm.DB) *GormProductUnitRepository {
	return &GormProductUnitRepository{db: db}
}

// FindByID finds a product unit by its ID
func (r *GormProductUnitRepository) FindByID(ctx context.Context, id uuid.UUID) (*catalog.ProductUnit, error) {
	var model models.ProductUnitModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByIDForTenant finds a product unit by ID within a tenant
func (r *GormProductUnitRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*catalog.ProductUnit, error) {
	var model models.ProductUnitModel
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

// FindByProductID finds all units for a product
func (r *GormProductUnitRepository) FindByProductID(ctx context.Context, tenantID, productID uuid.UUID) ([]catalog.ProductUnit, error) {
	var unitModels []models.ProductUnitModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND product_id = ?", tenantID, productID).
		Order("sort_order ASC, unit_name ASC").
		Find(&unitModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	units := make([]catalog.ProductUnit, len(unitModels))
	for i, model := range unitModels {
		units[i] = *model.ToDomain()
	}
	return units, nil
}

// FindByProductIDAndCode finds a specific unit for a product by code
func (r *GormProductUnitRepository) FindByProductIDAndCode(ctx context.Context, tenantID, productID uuid.UUID, unitCode string) (*catalog.ProductUnit, error) {
	var model models.ProductUnitModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND product_id = ? AND unit_code = ?", tenantID, productID, unitCode).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindDefaultPurchaseUnit finds the default purchase unit for a product
func (r *GormProductUnitRepository) FindDefaultPurchaseUnit(ctx context.Context, tenantID, productID uuid.UUID) (*catalog.ProductUnit, error) {
	var model models.ProductUnitModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND product_id = ? AND is_default_purchase_unit = true", tenantID, productID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindDefaultSalesUnit finds the default sales unit for a product
func (r *GormProductUnitRepository) FindDefaultSalesUnit(ctx context.Context, tenantID, productID uuid.UUID) (*catalog.ProductUnit, error) {
	var model models.ProductUnitModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND product_id = ? AND is_default_sales_unit = true", tenantID, productID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// Save creates or updates a product unit
func (r *GormProductUnitRepository) Save(ctx context.Context, unit *catalog.ProductUnit) error {
	model := models.ProductUnitModelFromDomain(unit)
	return r.db.WithContext(ctx).Save(model).Error
}

// SaveBatch creates or updates multiple product units
func (r *GormProductUnitRepository) SaveBatch(ctx context.Context, units []*catalog.ProductUnit) error {
	if len(units) == 0 {
		return nil
	}
	unitModels := make([]*models.ProductUnitModel, len(units))
	for i, u := range units {
		unitModels[i] = models.ProductUnitModelFromDomain(u)
	}
	return r.db.WithContext(ctx).Save(unitModels).Error
}

// Delete deletes a product unit
func (r *GormProductUnitRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.ProductUnitModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// DeleteForTenant deletes a product unit within a tenant
func (r *GormProductUnitRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.ProductUnitModel{}, "tenant_id = ? AND id = ?", tenantID, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// DeleteByProductID deletes all units for a product
func (r *GormProductUnitRepository) DeleteByProductID(ctx context.Context, tenantID, productID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Delete(&models.ProductUnitModel{}, "tenant_id = ? AND product_id = ?", tenantID, productID).Error
}

// ClearDefaultPurchaseUnit clears the default purchase unit flag for all units of a product
func (r *GormProductUnitRepository) ClearDefaultPurchaseUnit(ctx context.Context, tenantID, productID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&models.ProductUnitModel{}).
		Where("tenant_id = ? AND product_id = ?", tenantID, productID).
		Update("is_default_purchase_unit", false).Error
}

// ClearDefaultSalesUnit clears the default sales unit flag for all units of a product
func (r *GormProductUnitRepository) ClearDefaultSalesUnit(ctx context.Context, tenantID, productID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&models.ProductUnitModel{}).
		Where("tenant_id = ? AND product_id = ?", tenantID, productID).
		Update("is_default_sales_unit", false).Error
}

// CountByProductID counts units for a product
func (r *GormProductUnitRepository) CountByProductID(ctx context.Context, tenantID, productID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.ProductUnitModel{}).
		Where("tenant_id = ? AND product_id = ?", tenantID, productID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ExistsByProductIDAndCode checks if a unit with the given code exists for a product
func (r *GormProductUnitRepository) ExistsByProductIDAndCode(ctx context.Context, tenantID, productID uuid.UUID, unitCode string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.ProductUnitModel{}).
		Where("tenant_id = ? AND product_id = ? AND unit_code = ?", tenantID, productID, unitCode).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// Ensure GormProductUnitRepository implements ProductUnitRepository
var _ catalog.ProductUnitRepository = (*GormProductUnitRepository)(nil)
