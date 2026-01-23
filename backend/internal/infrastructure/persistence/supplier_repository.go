package persistence

import (
	"context"
	"errors"
	"strings"

	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// GormSupplierRepository implements SupplierRepository using GORM
type GormSupplierRepository struct {
	db *gorm.DB
}

// NewGormSupplierRepository creates a new GormSupplierRepository
func NewGormSupplierRepository(db *gorm.DB) *GormSupplierRepository {
	return &GormSupplierRepository{db: db}
}

// FindByID finds a supplier by its ID
func (r *GormSupplierRepository) FindByID(ctx context.Context, id uuid.UUID) (*partner.Supplier, error) {
	var supplier partner.Supplier
	if err := r.db.WithContext(ctx).First(&supplier, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &supplier, nil
}

// FindByIDForTenant finds a supplier by ID within a tenant
func (r *GormSupplierRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*partner.Supplier, error) {
	var supplier partner.Supplier
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		First(&supplier).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &supplier, nil
}

// FindByCode finds a supplier by its code within a tenant
func (r *GormSupplierRepository) FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*partner.Supplier, error) {
	var supplier partner.Supplier
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND code = ?", tenantID, strings.ToUpper(code)).
		First(&supplier).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &supplier, nil
}

// FindByPhone finds a supplier by phone number within a tenant
func (r *GormSupplierRepository) FindByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (*partner.Supplier, error) {
	if phone == "" {
		return nil, shared.NewDomainError("INVALID_PHONE", "Phone cannot be empty")
	}
	var supplier partner.Supplier
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND phone = ?", tenantID, phone).
		First(&supplier).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &supplier, nil
}

// FindByEmail finds a supplier by email within a tenant
func (r *GormSupplierRepository) FindByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*partner.Supplier, error) {
	if email == "" {
		return nil, shared.NewDomainError("INVALID_EMAIL", "Email cannot be empty")
	}
	var supplier partner.Supplier
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND email = ?", tenantID, strings.ToLower(email)).
		First(&supplier).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &supplier, nil
}

// FindAll finds all suppliers matching the filter
func (r *GormSupplierRepository) FindAll(ctx context.Context, filter shared.Filter) ([]partner.Supplier, error) {
	var suppliers []partner.Supplier
	query := r.applyFilter(r.db.WithContext(ctx).Model(&partner.Supplier{}), filter)

	if err := query.Find(&suppliers).Error; err != nil {
		return nil, err
	}
	return suppliers, nil
}

// FindAllForTenant finds all suppliers for a tenant
func (r *GormSupplierRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Supplier, error) {
	var suppliers []partner.Supplier
	query := r.applyFilter(r.db.WithContext(ctx).Model(&partner.Supplier{}).Where("tenant_id = ?", tenantID), filter)

	if err := query.Find(&suppliers).Error; err != nil {
		return nil, err
	}
	return suppliers, nil
}

// FindByType finds suppliers by type (manufacturer/distributor/retailer/service)
func (r *GormSupplierRepository) FindByType(ctx context.Context, tenantID uuid.UUID, supplierType partner.SupplierType, filter shared.Filter) ([]partner.Supplier, error) {
	var suppliers []partner.Supplier
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&partner.Supplier{}).
			Where("tenant_id = ? AND type = ?", tenantID, supplierType),
		filter,
	)

	if err := query.Find(&suppliers).Error; err != nil {
		return nil, err
	}
	return suppliers, nil
}

// FindByStatus finds suppliers by status for a tenant
func (r *GormSupplierRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status partner.SupplierStatus, filter shared.Filter) ([]partner.Supplier, error) {
	var suppliers []partner.Supplier
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&partner.Supplier{}).
			Where("tenant_id = ? AND status = ?", tenantID, status),
		filter,
	)

	if err := query.Find(&suppliers).Error; err != nil {
		return nil, err
	}
	return suppliers, nil
}

// FindActive finds all active suppliers for a tenant
func (r *GormSupplierRepository) FindActive(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Supplier, error) {
	return r.FindByStatus(ctx, tenantID, partner.SupplierStatusActive, filter)
}

// FindByIDs finds multiple suppliers by their IDs
func (r *GormSupplierRepository) FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]partner.Supplier, error) {
	if len(ids) == 0 {
		return []partner.Supplier{}, nil
	}

	var suppliers []partner.Supplier
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id IN ?", tenantID, ids).
		Find(&suppliers).Error; err != nil {
		return nil, err
	}
	return suppliers, nil
}

// FindByCodes finds multiple suppliers by their codes
func (r *GormSupplierRepository) FindByCodes(ctx context.Context, tenantID uuid.UUID, codes []string) ([]partner.Supplier, error) {
	if len(codes) == 0 {
		return []partner.Supplier{}, nil
	}

	upperCodes := make([]string, len(codes))
	for i, code := range codes {
		upperCodes[i] = strings.ToUpper(code)
	}

	var suppliers []partner.Supplier
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND code IN ?", tenantID, upperCodes).
		Find(&suppliers).Error; err != nil {
		return nil, err
	}
	return suppliers, nil
}

// FindWithOutstandingBalance finds suppliers with accounts payable balance > 0
func (r *GormSupplierRepository) FindWithOutstandingBalance(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Supplier, error) {
	var suppliers []partner.Supplier
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&partner.Supplier{}).
			Where("tenant_id = ? AND balance > ?", tenantID, decimal.Zero),
		filter,
	)

	if err := query.Find(&suppliers).Error; err != nil {
		return nil, err
	}
	return suppliers, nil
}

// FindOverCreditLimit finds suppliers whose balance exceeds their credit limit
func (r *GormSupplierRepository) FindOverCreditLimit(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Supplier, error) {
	var suppliers []partner.Supplier
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&partner.Supplier{}).
			Where("tenant_id = ? AND credit_limit > 0 AND balance > credit_limit", tenantID),
		filter,
	)

	if err := query.Find(&suppliers).Error; err != nil {
		return nil, err
	}
	return suppliers, nil
}

// Save creates or updates a supplier
func (r *GormSupplierRepository) Save(ctx context.Context, supplier *partner.Supplier) error {
	return r.db.WithContext(ctx).Save(supplier).Error
}

// SaveBatch creates or updates multiple suppliers
func (r *GormSupplierRepository) SaveBatch(ctx context.Context, suppliers []*partner.Supplier) error {
	if len(suppliers) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Save(suppliers).Error
}

// Delete deletes a supplier
func (r *GormSupplierRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&partner.Supplier{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// DeleteForTenant deletes a supplier within a tenant
func (r *GormSupplierRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&partner.Supplier{}, "tenant_id = ? AND id = ?", tenantID, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// Count counts suppliers matching the filter
func (r *GormSupplierRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&partner.Supplier{})
	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountForTenant counts suppliers for a tenant
func (r *GormSupplierRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&partner.Supplier{}).Where("tenant_id = ?", tenantID)
	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByType counts suppliers by type for a tenant
func (r *GormSupplierRepository) CountByType(ctx context.Context, tenantID uuid.UUID, supplierType partner.SupplierType) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&partner.Supplier{}).
		Where("tenant_id = ? AND type = ?", tenantID, supplierType).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByStatus counts suppliers by status for a tenant
func (r *GormSupplierRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status partner.SupplierStatus) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&partner.Supplier{}).
		Where("tenant_id = ? AND status = ?", tenantID, status).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ExistsByCode checks if a supplier with the given code exists in the tenant
func (r *GormSupplierRepository) ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&partner.Supplier{}).
		Where("tenant_id = ? AND code = ?", tenantID, strings.ToUpper(code)).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ExistsByPhone checks if a supplier with the given phone exists in the tenant
func (r *GormSupplierRepository) ExistsByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (bool, error) {
	if phone == "" {
		return false, nil
	}
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&partner.Supplier{}).
		Where("tenant_id = ? AND phone = ?", tenantID, phone).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ExistsByEmail checks if a supplier with the given email exists in the tenant
func (r *GormSupplierRepository) ExistsByEmail(ctx context.Context, tenantID uuid.UUID, email string) (bool, error) {
	if email == "" {
		return false, nil
	}
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&partner.Supplier{}).
		Where("tenant_id = ? AND email = ?", tenantID, strings.ToLower(email)).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// applyFilter applies filter options to the query
func (r *GormSupplierRepository) applyFilter(query *gorm.DB, filter shared.Filter) *gorm.DB {
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
func (r *GormSupplierRepository) applyFilterWithoutPagination(query *gorm.DB, filter shared.Filter) *gorm.DB {
	// Apply search
	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		query = query.Where("name ILIKE ? OR code ILIKE ? OR phone ILIKE ? OR email ILIKE ?",
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
		case "has_balance":
			if value == true {
				query = query.Where("balance > 0")
			} else {
				query = query.Where("balance = 0")
			}
		case "has_credit_limit":
			if value == true {
				query = query.Where("credit_limit > 0")
			} else {
				query = query.Where("credit_limit = 0")
			}
		case "over_credit_limit":
			if value == true {
				query = query.Where("credit_limit > 0 AND balance > credit_limit")
			}
		case "min_rating":
			query = query.Where("rating >= ?", value)
		case "max_rating":
			query = query.Where("rating <= ?", value)
		}
	}

	return query
}

// Ensure GormSupplierRepository implements SupplierRepository
var _ partner.SupplierRepository = (*GormSupplierRepository)(nil)
