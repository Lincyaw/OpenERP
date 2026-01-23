package persistence

import (
	"context"
	"errors"
	"strings"

	"github.com/erp/backend/internal/domain/catalog"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GormProductRepository implements ProductRepository using GORM
type GormProductRepository struct {
	db *gorm.DB
}

// NewGormProductRepository creates a new GormProductRepository
func NewGormProductRepository(db *gorm.DB) *GormProductRepository {
	return &GormProductRepository{db: db}
}

// FindByID finds a product by its ID
func (r *GormProductRepository) FindByID(ctx context.Context, id uuid.UUID) (*catalog.Product, error) {
	var product catalog.Product
	if err := r.db.WithContext(ctx).First(&product, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &product, nil
}

// FindByIDForTenant finds a product by ID within a tenant
func (r *GormProductRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*catalog.Product, error) {
	var product catalog.Product
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		First(&product).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &product, nil
}

// FindByCode finds a product by its code within a tenant
func (r *GormProductRepository) FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*catalog.Product, error) {
	var product catalog.Product
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND code = ?", tenantID, strings.ToUpper(code)).
		First(&product).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &product, nil
}

// FindByBarcode finds a product by its barcode within a tenant
func (r *GormProductRepository) FindByBarcode(ctx context.Context, tenantID uuid.UUID, barcode string) (*catalog.Product, error) {
	if barcode == "" {
		return nil, shared.NewDomainError("INVALID_BARCODE", "Barcode cannot be empty")
	}
	var product catalog.Product
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND barcode = ?", tenantID, barcode).
		First(&product).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &product, nil
}

// FindAll finds all products matching the filter
func (r *GormProductRepository) FindAll(ctx context.Context, filter shared.Filter) ([]catalog.Product, error) {
	var products []catalog.Product
	query := r.applyFilter(r.db.WithContext(ctx).Model(&catalog.Product{}), filter)

	if err := query.Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

// FindAllForTenant finds all products for a tenant
func (r *GormProductRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]catalog.Product, error) {
	var products []catalog.Product
	query := r.applyFilter(r.db.WithContext(ctx).Model(&catalog.Product{}).Where("tenant_id = ?", tenantID), filter)

	if err := query.Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

// FindByCategory finds all products in a specific category
func (r *GormProductRepository) FindByCategory(ctx context.Context, tenantID, categoryID uuid.UUID, filter shared.Filter) ([]catalog.Product, error) {
	var products []catalog.Product
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&catalog.Product{}).
			Where("tenant_id = ? AND category_id = ?", tenantID, categoryID),
		filter,
	)

	if err := query.Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

// FindByCategories finds all products in multiple categories
func (r *GormProductRepository) FindByCategories(ctx context.Context, tenantID uuid.UUID, categoryIDs []uuid.UUID, filter shared.Filter) ([]catalog.Product, error) {
	if len(categoryIDs) == 0 {
		return []catalog.Product{}, nil
	}

	var products []catalog.Product
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&catalog.Product{}).
			Where("tenant_id = ? AND category_id IN ?", tenantID, categoryIDs),
		filter,
	)

	if err := query.Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

// FindActive finds all active products for a tenant
func (r *GormProductRepository) FindActive(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]catalog.Product, error) {
	var products []catalog.Product
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&catalog.Product{}).
			Where("tenant_id = ? AND status = ?", tenantID, catalog.ProductStatusActive),
		filter,
	)

	if err := query.Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

// FindByStatus finds products by status for a tenant
func (r *GormProductRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status catalog.ProductStatus, filter shared.Filter) ([]catalog.Product, error) {
	var products []catalog.Product
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&catalog.Product{}).
			Where("tenant_id = ? AND status = ?", tenantID, status),
		filter,
	)

	if err := query.Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

// FindByIDs finds multiple products by their IDs
func (r *GormProductRepository) FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]catalog.Product, error) {
	if len(ids) == 0 {
		return []catalog.Product{}, nil
	}

	var products []catalog.Product
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id IN ?", tenantID, ids).
		Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

// FindByCodes finds multiple products by their codes
func (r *GormProductRepository) FindByCodes(ctx context.Context, tenantID uuid.UUID, codes []string) ([]catalog.Product, error) {
	if len(codes) == 0 {
		return []catalog.Product{}, nil
	}

	// Convert codes to uppercase
	upperCodes := make([]string, len(codes))
	for i, code := range codes {
		upperCodes[i] = strings.ToUpper(code)
	}

	var products []catalog.Product
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND code IN ?", tenantID, upperCodes).
		Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

// Save creates or updates a product
func (r *GormProductRepository) Save(ctx context.Context, product *catalog.Product) error {
	return r.db.WithContext(ctx).Save(product).Error
}

// SaveBatch creates or updates multiple products
func (r *GormProductRepository) SaveBatch(ctx context.Context, products []*catalog.Product) error {
	if len(products) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Save(products).Error
}

// Delete deletes a product
func (r *GormProductRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&catalog.Product{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// DeleteForTenant deletes a product within a tenant
func (r *GormProductRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&catalog.Product{}, "tenant_id = ? AND id = ?", tenantID, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// Count counts products matching the filter
func (r *GormProductRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&catalog.Product{})
	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountForTenant counts products for a tenant
func (r *GormProductRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&catalog.Product{}).Where("tenant_id = ?", tenantID)
	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByCategory counts products in a specific category
func (r *GormProductRepository) CountByCategory(ctx context.Context, tenantID, categoryID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&catalog.Product{}).
		Where("tenant_id = ? AND category_id = ?", tenantID, categoryID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByStatus counts products by status for a tenant
func (r *GormProductRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status catalog.ProductStatus) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&catalog.Product{}).
		Where("tenant_id = ? AND status = ?", tenantID, status).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ExistsByCode checks if a product with the given code exists in the tenant
func (r *GormProductRepository) ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&catalog.Product{}).
		Where("tenant_id = ? AND code = ?", tenantID, strings.ToUpper(code)).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ExistsByBarcode checks if a product with the given barcode exists in the tenant
func (r *GormProductRepository) ExistsByBarcode(ctx context.Context, tenantID uuid.UUID, barcode string) (bool, error) {
	if barcode == "" {
		return false, nil
	}
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&catalog.Product{}).
		Where("tenant_id = ? AND barcode = ?", tenantID, barcode).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// applyFilter applies filter options to the query
func (r *GormProductRepository) applyFilter(query *gorm.DB, filter shared.Filter) *gorm.DB {
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
	} else {
		// Default ordering
		query = query.Order("sort_order ASC, name ASC")
	}

	return query
}

// applyFilterWithoutPagination applies filter options without pagination
func (r *GormProductRepository) applyFilterWithoutPagination(query *gorm.DB, filter shared.Filter) *gorm.DB {
	// Apply search
	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		query = query.Where("name ILIKE ? OR code ILIKE ? OR barcode ILIKE ?", searchPattern, searchPattern, searchPattern)
	}

	// Apply additional filters
	for key, value := range filter.Filters {
		switch key {
		case "status":
			query = query.Where("status = ?", value)
		case "category_id":
			if value == nil {
				query = query.Where("category_id IS NULL")
			} else {
				query = query.Where("category_id = ?", value)
			}
		case "unit":
			query = query.Where("unit = ?", value)
		case "min_price":
			query = query.Where("selling_price >= ?", value)
		case "max_price":
			query = query.Where("selling_price <= ?", value)
		case "has_barcode":
			if value == true {
				query = query.Where("barcode IS NOT NULL AND barcode != ''")
			} else {
				query = query.Where("barcode IS NULL OR barcode = ''")
			}
		}
	}

	return query
}

// Ensure GormProductRepository implements ProductRepository
var _ catalog.ProductRepository = (*GormProductRepository)(nil)
