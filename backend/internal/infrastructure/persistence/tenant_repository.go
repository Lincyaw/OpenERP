package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GormTenantRepository implements TenantRepository using GORM
type GormTenantRepository struct {
	db *gorm.DB
}

// NewGormTenantRepository creates a new GormTenantRepository
func NewGormTenantRepository(db *gorm.DB) *GormTenantRepository {
	return &GormTenantRepository{db: db}
}

// FindByID finds a tenant by its ID
func (r *GormTenantRepository) FindByID(ctx context.Context, id uuid.UUID) (*identity.Tenant, error) {
	var model models.TenantModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByCode finds a tenant by its unique code
func (r *GormTenantRepository) FindByCode(ctx context.Context, code string) (*identity.Tenant, error) {
	var model models.TenantModel
	if err := r.db.WithContext(ctx).
		Where("UPPER(code) = ?", strings.ToUpper(code)).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByDomain finds a tenant by its custom domain
func (r *GormTenantRepository) FindByDomain(ctx context.Context, domain string) (*identity.Tenant, error) {
	if domain == "" {
		return nil, shared.ErrNotFound
	}
	var model models.TenantModel
	if err := r.db.WithContext(ctx).
		Where("LOWER(domain) = ?", strings.ToLower(domain)).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindAll finds all tenants matching the filter
func (r *GormTenantRepository) FindAll(ctx context.Context, filter shared.Filter) ([]identity.Tenant, error) {
	var tenantModels []models.TenantModel
	query := r.db.WithContext(ctx).Model(&models.TenantModel{})

	// Apply keyword search
	if filter.Search != "" {
		keyword := "%" + filter.Search + "%"
		query = query.Where("name ILIKE ? OR code ILIKE ? OR short_name ILIKE ?", keyword, keyword, keyword)
	}

	// Apply sorting with whitelist validation to prevent SQL injection
	sortField := ValidateSortField(filter.OrderBy, TenantSortFields, "created_at")
	sortOrder := ValidateSortOrder(filter.OrderDir)
	query = query.Order(sortField + " " + sortOrder)

	// Apply pagination
	offset := (filter.Page - 1) * filter.PageSize
	if offset < 0 {
		offset = 0
	}
	limit := filter.PageSize
	if limit <= 0 {
		limit = 20
	}
	query = query.Offset(offset).Limit(limit)

	if err := query.Find(&tenantModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	tenants := make([]identity.Tenant, len(tenantModels))
	for i, model := range tenantModels {
		tenants[i] = *model.ToDomain()
	}

	return tenants, nil
}

// FindByStatus finds tenants by status
func (r *GormTenantRepository) FindByStatus(ctx context.Context, status identity.TenantStatus, filter shared.Filter) ([]identity.Tenant, error) {
	var tenantModels []models.TenantModel
	query := r.db.WithContext(ctx).Model(&models.TenantModel{}).
		Where("status = ?", status)

	// Apply keyword search
	if filter.Search != "" {
		keyword := "%" + filter.Search + "%"
		query = query.Where("name ILIKE ? OR code ILIKE ?", keyword, keyword)
	}

	// Apply sorting with whitelist validation to prevent SQL injection
	sortField := ValidateSortField(filter.OrderBy, TenantSortFields, "created_at")
	sortOrder := ValidateSortOrder(filter.OrderDir)
	query = query.Order(sortField + " " + sortOrder)

	// Apply pagination
	offset := (filter.Page - 1) * filter.PageSize
	if offset < 0 {
		offset = 0
	}
	limit := filter.PageSize
	if limit <= 0 {
		limit = 20
	}
	query = query.Offset(offset).Limit(limit)

	if err := query.Find(&tenantModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	tenants := make([]identity.Tenant, len(tenantModels))
	for i, model := range tenantModels {
		tenants[i] = *model.ToDomain()
	}

	return tenants, nil
}

// FindByPlan finds tenants by subscription plan
func (r *GormTenantRepository) FindByPlan(ctx context.Context, plan identity.TenantPlan, filter shared.Filter) ([]identity.Tenant, error) {
	var tenantModels []models.TenantModel
	query := r.db.WithContext(ctx).Model(&models.TenantModel{}).
		Where("plan = ?", plan)

	// Apply pagination
	offset := (filter.Page - 1) * filter.PageSize
	if offset < 0 {
		offset = 0
	}
	limit := filter.PageSize
	if limit <= 0 {
		limit = 20
	}
	query = query.Offset(offset).Limit(limit)

	if err := query.Find(&tenantModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	tenants := make([]identity.Tenant, len(tenantModels))
	for i, model := range tenantModels {
		tenants[i] = *model.ToDomain()
	}

	return tenants, nil
}

// FindActive finds all active tenants
func (r *GormTenantRepository) FindActive(ctx context.Context, filter shared.Filter) ([]identity.Tenant, error) {
	return r.FindByStatus(ctx, identity.TenantStatusActive, filter)
}

// FindTrialExpiring finds tenants whose trial is expiring within the given days
func (r *GormTenantRepository) FindTrialExpiring(ctx context.Context, withinDays int) ([]identity.Tenant, error) {
	var tenantModels []models.TenantModel
	expiryDate := time.Now().AddDate(0, 0, withinDays)

	if err := r.db.WithContext(ctx).
		Where("status = ?", identity.TenantStatusTrial).
		Where("trial_ends_at IS NOT NULL").
		Where("trial_ends_at <= ?", expiryDate).
		Where("trial_ends_at > ?", time.Now()).
		Find(&tenantModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	tenants := make([]identity.Tenant, len(tenantModels))
	for i, model := range tenantModels {
		tenants[i] = *model.ToDomain()
	}

	return tenants, nil
}

// FindSubscriptionExpiring finds tenants whose subscription is expiring within the given days
func (r *GormTenantRepository) FindSubscriptionExpiring(ctx context.Context, withinDays int) ([]identity.Tenant, error) {
	var tenantModels []models.TenantModel
	expiryDate := time.Now().AddDate(0, 0, withinDays)

	if err := r.db.WithContext(ctx).
		Where("status = ?", identity.TenantStatusActive).
		Where("expires_at IS NOT NULL").
		Where("expires_at <= ?", expiryDate).
		Where("expires_at > ?", time.Now()).
		Find(&tenantModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	tenants := make([]identity.Tenant, len(tenantModels))
	for i, model := range tenantModels {
		tenants[i] = *model.ToDomain()
	}

	return tenants, nil
}

// FindByIDs finds multiple tenants by their IDs
func (r *GormTenantRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]identity.Tenant, error) {
	if len(ids) == 0 {
		return []identity.Tenant{}, nil
	}

	var tenantModels []models.TenantModel
	if err := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&tenantModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	tenants := make([]identity.Tenant, len(tenantModels))
	for i, model := range tenantModels {
		tenants[i] = *model.ToDomain()
	}

	return tenants, nil
}

// Save creates or updates a tenant
func (r *GormTenantRepository) Save(ctx context.Context, tenant *identity.Tenant) error {
	model := models.TenantModelFromDomain(tenant)
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete deletes a tenant
func (r *GormTenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.TenantModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// Count counts tenants matching the filter
func (r *GormTenantRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.TenantModel{})

	// Apply keyword search
	if filter.Search != "" {
		keyword := "%" + filter.Search + "%"
		query = query.Where("name ILIKE ? OR code ILIKE ?", keyword, keyword)
	}

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

// CountByStatus counts tenants by status
func (r *GormTenantRepository) CountByStatus(ctx context.Context, status identity.TenantStatus) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.TenantModel{}).
		Where("status = ?", status).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByPlan counts tenants by plan
func (r *GormTenantRepository) CountByPlan(ctx context.Context, plan identity.TenantPlan) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.TenantModel{}).
		Where("plan = ?", plan).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ExistsByCode checks if a tenant with the given code exists
func (r *GormTenantRepository) ExistsByCode(ctx context.Context, code string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.TenantModel{}).
		Where("UPPER(code) = ?", strings.ToUpper(code)).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ExistsByDomain checks if a tenant with the given domain exists
func (r *GormTenantRepository) ExistsByDomain(ctx context.Context, domain string) (bool, error) {
	if domain == "" {
		return false, nil
	}
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.TenantModel{}).
		Where("LOWER(domain) = ?", strings.ToLower(domain)).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// FindByStripeCustomerID finds a tenant by its Stripe customer ID
func (r *GormTenantRepository) FindByStripeCustomerID(ctx context.Context, customerID string) (*identity.Tenant, error) {
	if customerID == "" {
		return nil, shared.ErrNotFound
	}
	var model models.TenantModel
	if err := r.db.WithContext(ctx).
		Where("stripe_customer_id = ?", customerID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByStripeSubscriptionID finds a tenant by its Stripe subscription ID
func (r *GormTenantRepository) FindByStripeSubscriptionID(ctx context.Context, subscriptionID string) (*identity.Tenant, error) {
	if subscriptionID == "" {
		return nil, shared.ErrNotFound
	}
	var model models.TenantModel
	if err := r.db.WithContext(ctx).
		Where("stripe_subscription_id = ?", subscriptionID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}
