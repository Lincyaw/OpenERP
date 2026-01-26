package persistence

import (
	"context"
	"errors"
	"strings"

	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// GormCustomerRepository implements CustomerRepository using GORM
type GormCustomerRepository struct {
	db *gorm.DB
}

// NewGormCustomerRepository creates a new GormCustomerRepository
func NewGormCustomerRepository(db *gorm.DB) *GormCustomerRepository {
	return &GormCustomerRepository{db: db}
}

// FindByID finds a customer by its ID
func (r *GormCustomerRepository) FindByID(ctx context.Context, id uuid.UUID) (*partner.Customer, error) {
	var model models.CustomerModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByIDForTenant finds a customer by ID within a tenant
func (r *GormCustomerRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*partner.Customer, error) {
	var model models.CustomerModel
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

// FindByCode finds a customer by its code within a tenant
func (r *GormCustomerRepository) FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*partner.Customer, error) {
	var model models.CustomerModel
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

// FindByPhone finds a customer by phone number within a tenant
func (r *GormCustomerRepository) FindByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (*partner.Customer, error) {
	if phone == "" {
		return nil, shared.NewDomainError("INVALID_PHONE", "Phone cannot be empty")
	}
	var model models.CustomerModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND phone = ?", tenantID, phone).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByEmail finds a customer by email within a tenant
func (r *GormCustomerRepository) FindByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*partner.Customer, error) {
	if email == "" {
		return nil, shared.NewDomainError("INVALID_EMAIL", "Email cannot be empty")
	}
	var model models.CustomerModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND email = ?", tenantID, strings.ToLower(email)).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindAll finds all customers matching the filter
func (r *GormCustomerRepository) FindAll(ctx context.Context, filter shared.Filter) ([]partner.Customer, error) {
	var customerModels []models.CustomerModel
	query := r.applyFilter(r.db.WithContext(ctx).Model(&models.CustomerModel{}), filter)

	if err := query.Find(&customerModels).Error; err != nil {
		return nil, err
	}

	customers := make([]partner.Customer, len(customerModels))
	for i, model := range customerModels {
		customers[i] = *model.ToDomain()
	}
	return customers, nil
}

// FindAllForTenant finds all customers for a tenant
func (r *GormCustomerRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Customer, error) {
	var customerModels []models.CustomerModel
	query := r.applyFilter(r.db.WithContext(ctx).Model(&models.CustomerModel{}).Where("tenant_id = ?", tenantID), filter)

	if err := query.Find(&customerModels).Error; err != nil {
		return nil, err
	}

	customers := make([]partner.Customer, len(customerModels))
	for i, model := range customerModels {
		customers[i] = *model.ToDomain()
	}
	return customers, nil
}

// FindByType finds customers by type (individual/organization)
func (r *GormCustomerRepository) FindByType(ctx context.Context, tenantID uuid.UUID, customerType partner.CustomerType, filter shared.Filter) ([]partner.Customer, error) {
	var customerModels []models.CustomerModel
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&models.CustomerModel{}).
			Where("tenant_id = ? AND type = ?", tenantID, customerType),
		filter,
	)

	if err := query.Find(&customerModels).Error; err != nil {
		return nil, err
	}

	customers := make([]partner.Customer, len(customerModels))
	for i, model := range customerModels {
		customers[i] = *model.ToDomain()
	}
	return customers, nil
}

// FindByLevel finds customers by tier level
func (r *GormCustomerRepository) FindByLevel(ctx context.Context, tenantID uuid.UUID, level partner.CustomerLevel, filter shared.Filter) ([]partner.Customer, error) {
	var customerModels []models.CustomerModel
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&models.CustomerModel{}).
			Where("tenant_id = ? AND level = ?", tenantID, level),
		filter,
	)

	if err := query.Find(&customerModels).Error; err != nil {
		return nil, err
	}

	customers := make([]partner.Customer, len(customerModels))
	for i, model := range customerModels {
		customers[i] = *model.ToDomain()
	}
	return customers, nil
}

// FindByStatus finds customers by status for a tenant
func (r *GormCustomerRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status partner.CustomerStatus, filter shared.Filter) ([]partner.Customer, error) {
	var customerModels []models.CustomerModel
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&models.CustomerModel{}).
			Where("tenant_id = ? AND status = ?", tenantID, status),
		filter,
	)

	if err := query.Find(&customerModels).Error; err != nil {
		return nil, err
	}

	customers := make([]partner.Customer, len(customerModels))
	for i, model := range customerModels {
		customers[i] = *model.ToDomain()
	}
	return customers, nil
}

// FindActive finds all active customers for a tenant
func (r *GormCustomerRepository) FindActive(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Customer, error) {
	return r.FindByStatus(ctx, tenantID, partner.CustomerStatusActive, filter)
}

// FindByIDs finds multiple customers by their IDs
func (r *GormCustomerRepository) FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]partner.Customer, error) {
	if len(ids) == 0 {
		return []partner.Customer{}, nil
	}

	var customerModels []models.CustomerModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id IN ?", tenantID, ids).
		Find(&customerModels).Error; err != nil {
		return nil, err
	}

	customers := make([]partner.Customer, len(customerModels))
	for i, model := range customerModels {
		customers[i] = *model.ToDomain()
	}
	return customers, nil
}

// FindByCodes finds multiple customers by their codes
func (r *GormCustomerRepository) FindByCodes(ctx context.Context, tenantID uuid.UUID, codes []string) ([]partner.Customer, error) {
	if len(codes) == 0 {
		return []partner.Customer{}, nil
	}

	upperCodes := make([]string, len(codes))
	for i, code := range codes {
		upperCodes[i] = strings.ToUpper(code)
	}

	var customerModels []models.CustomerModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND code IN ?", tenantID, upperCodes).
		Find(&customerModels).Error; err != nil {
		return nil, err
	}

	customers := make([]partner.Customer, len(customerModels))
	for i, model := range customerModels {
		customers[i] = *model.ToDomain()
	}
	return customers, nil
}

// FindWithPositiveBalance finds customers with prepaid balance > 0
func (r *GormCustomerRepository) FindWithPositiveBalance(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Customer, error) {
	var customerModels []models.CustomerModel
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&models.CustomerModel{}).
			Where("tenant_id = ? AND balance > ?", tenantID, decimal.Zero),
		filter,
	)

	if err := query.Find(&customerModels).Error; err != nil {
		return nil, err
	}

	customers := make([]partner.Customer, len(customerModels))
	for i, model := range customerModels {
		customers[i] = *model.ToDomain()
	}
	return customers, nil
}

// Save creates or updates a customer
func (r *GormCustomerRepository) Save(ctx context.Context, customer *partner.Customer) error {
	model := models.CustomerModelFromDomain(customer)
	return r.db.WithContext(ctx).Save(model).Error
}

// SaveWithLock saves a customer with optimistic locking (version check)
// Returns error if the version has changed (concurrent modification)
func (r *GormCustomerRepository) SaveWithLock(ctx context.Context, customer *partner.Customer) error {
	model := models.CustomerModelFromDomain(customer)
	result := r.db.WithContext(ctx).
		Model(model).
		Where("id = ? AND version = ?", customer.ID, customer.Version-1).
		Updates(model)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.NewDomainError("OPTIMISTIC_LOCK_ERROR", "The customer record has been modified by another transaction")
	}
	return nil
}

// SaveBatch creates or updates multiple customers
func (r *GormCustomerRepository) SaveBatch(ctx context.Context, customers []*partner.Customer) error {
	if len(customers) == 0 {
		return nil
	}
	customerModels := make([]*models.CustomerModel, len(customers))
	for i, c := range customers {
		customerModels[i] = models.CustomerModelFromDomain(c)
	}
	return r.db.WithContext(ctx).Save(customerModels).Error
}

// Delete deletes a customer
func (r *GormCustomerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.CustomerModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// DeleteForTenant deletes a customer within a tenant
func (r *GormCustomerRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.CustomerModel{}, "tenant_id = ? AND id = ?", tenantID, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// Count counts customers matching the filter
func (r *GormCustomerRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.CustomerModel{})
	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountForTenant counts customers for a tenant
func (r *GormCustomerRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.CustomerModel{}).Where("tenant_id = ?", tenantID)
	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByType counts customers by type for a tenant
func (r *GormCustomerRepository) CountByType(ctx context.Context, tenantID uuid.UUID, customerType partner.CustomerType) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.CustomerModel{}).
		Where("tenant_id = ? AND type = ?", tenantID, customerType).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByLevel counts customers by level for a tenant
func (r *GormCustomerRepository) CountByLevel(ctx context.Context, tenantID uuid.UUID, level partner.CustomerLevel) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.CustomerModel{}).
		Where("tenant_id = ? AND level = ?", tenantID, level).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByStatus counts customers by status for a tenant
func (r *GormCustomerRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status partner.CustomerStatus) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.CustomerModel{}).
		Where("tenant_id = ? AND status = ?", tenantID, status).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ExistsByCode checks if a customer with the given code exists in the tenant
func (r *GormCustomerRepository) ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.CustomerModel{}).
		Where("tenant_id = ? AND code = ?", tenantID, strings.ToUpper(code)).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ExistsByPhone checks if a customer with the given phone exists in the tenant
func (r *GormCustomerRepository) ExistsByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (bool, error) {
	if phone == "" {
		return false, nil
	}
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.CustomerModel{}).
		Where("tenant_id = ? AND phone = ?", tenantID, phone).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ExistsByEmail checks if a customer with the given email exists in the tenant
func (r *GormCustomerRepository) ExistsByEmail(ctx context.Context, tenantID uuid.UUID, email string) (bool, error) {
	if email == "" {
		return false, nil
	}
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.CustomerModel{}).
		Where("tenant_id = ? AND email = ?", tenantID, strings.ToLower(email)).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// applyFilter applies filter options to the query
func (r *GormCustomerRepository) applyFilter(query *gorm.DB, filter shared.Filter) *gorm.DB {
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
func (r *GormCustomerRepository) applyFilterWithoutPagination(query *gorm.DB, filter shared.Filter) *gorm.DB {
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
		case "level":
			query = query.Where("level = ?", value)
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
		}
	}

	return query
}

// Ensure GormCustomerRepository implements CustomerRepository
var _ partner.CustomerRepository = (*GormCustomerRepository)(nil)
