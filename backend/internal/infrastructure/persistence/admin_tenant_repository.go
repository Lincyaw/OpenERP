package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AdminTenantSortFields contains allowed sort fields for admin tenant queries
var AdminTenantSortFields = map[string]bool{
	"id":            true,
	"created_at":    true,
	"updated_at":    true,
	"code":          true,
	"name":          true,
	"short_name":    true,
	"status":        true,
	"plan":          true,
	"expires_at":    true,
	"trial_ends_at": true,
	"contact_name":  true,
	"contact_email": true,
}

// GormAdminTenantRepository implements AdminTenantRepository using GORM
// This repository bypasses tenant isolation and is only accessible to super admins
type GormAdminTenantRepository struct {
	db *gorm.DB
}

// NewGormAdminTenantRepository creates a new GormAdminTenantRepository
func NewGormAdminTenantRepository(db *gorm.DB) *GormAdminTenantRepository {
	return &GormAdminTenantRepository{db: db}
}

// FindAll finds all tenants matching the filter with pagination
func (r *GormAdminTenantRepository) FindAll(ctx context.Context, filter identity.AdminTenantFilter) ([]identity.Tenant, error) {
	var tenantModels []models.TenantModel
	query := r.buildQuery(ctx, filter)

	// Apply sorting with whitelist validation to prevent SQL injection
	sortField := ValidateSortField(filter.OrderBy, AdminTenantSortFields, "created_at")
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

// FindByID finds a tenant by its ID
func (r *GormAdminTenantRepository) FindByID(ctx context.Context, id uuid.UUID) (*identity.Tenant, error) {
	var model models.TenantModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByIDs finds multiple tenants by their IDs
func (r *GormAdminTenantRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]identity.Tenant, error) {
	if len(ids) == 0 {
		return []identity.Tenant{}, nil
	}

	var tenantModels []models.TenantModel
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&tenantModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	tenants := make([]identity.Tenant, len(tenantModels))
	for i, model := range tenantModels {
		tenants[i] = *model.ToDomain()
	}

	return tenants, nil
}

// Count counts tenants matching the filter
func (r *GormAdminTenantRepository) Count(ctx context.Context, filter identity.AdminTenantFilter) (int64, error) {
	var count int64
	query := r.buildQuery(ctx, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

// Search searches tenants by name, code, or contact email with fuzzy matching
func (r *GormAdminTenantRepository) Search(ctx context.Context, query string, filter identity.AdminTenantFilter) ([]identity.Tenant, error) {
	// Set the search query in the filter
	filter.Search = query
	return r.FindAll(ctx, filter)
}

// CountByStatus returns the count of tenants grouped by status
func (r *GormAdminTenantRepository) CountByStatus(ctx context.Context) (map[identity.TenantStatus]int64, error) {
	type statusCount struct {
		Status identity.TenantStatus
		Count  int64
	}

	var results []statusCount
	if err := r.db.WithContext(ctx).
		Model(&models.TenantModel{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&results).Error; err != nil {
		return nil, err
	}

	counts := make(map[identity.TenantStatus]int64)
	for _, result := range results {
		counts[result.Status] = result.Count
	}

	return counts, nil
}

// CountByPlan returns the count of tenants grouped by plan
func (r *GormAdminTenantRepository) CountByPlan(ctx context.Context) (map[identity.TenantPlan]int64, error) {
	type planCount struct {
		Plan  identity.TenantPlan
		Count int64
	}

	var results []planCount
	if err := r.db.WithContext(ctx).
		Model(&models.TenantModel{}).
		Select("plan, COUNT(*) as count").
		Group("plan").
		Scan(&results).Error; err != nil {
		return nil, err
	}

	counts := make(map[identity.TenantPlan]int64)
	for _, result := range results {
		counts[result.Plan] = result.Count
	}

	return counts, nil
}

// FindTrialExpiring finds tenants whose trial is expiring within the given days
func (r *GormAdminTenantRepository) FindTrialExpiring(ctx context.Context, withinDays int) ([]identity.Tenant, error) {
	var tenantModels []models.TenantModel
	expiryDate := time.Now().AddDate(0, 0, withinDays)

	if err := r.db.WithContext(ctx).
		Where("status = ?", identity.TenantStatusTrial).
		Where("trial_ends_at IS NOT NULL").
		Where("trial_ends_at <= ?", expiryDate).
		Where("trial_ends_at > ?", time.Now()).
		Order("trial_ends_at ASC").
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
func (r *GormAdminTenantRepository) FindSubscriptionExpiring(ctx context.Context, withinDays int) ([]identity.Tenant, error) {
	var tenantModels []models.TenantModel
	expiryDate := time.Now().AddDate(0, 0, withinDays)

	if err := r.db.WithContext(ctx).
		Where("status = ?", identity.TenantStatusActive).
		Where("expires_at IS NOT NULL").
		Where("expires_at <= ?", expiryDate).
		Where("expires_at > ?", time.Now()).
		Order("expires_at ASC").
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

// FindExpired finds tenants whose subscription has expired
func (r *GormAdminTenantRepository) FindExpired(ctx context.Context) ([]identity.Tenant, error) {
	var tenantModels []models.TenantModel
	now := time.Now()

	if err := r.db.WithContext(ctx).
		Where("(status = ? AND trial_ends_at IS NOT NULL AND trial_ends_at < ?) OR "+
			"(status = ? AND expires_at IS NOT NULL AND expires_at < ?)",
			identity.TenantStatusTrial, now,
			identity.TenantStatusActive, now).
		Order("created_at DESC").
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

// GetStatistics returns aggregated statistics for all tenants
func (r *GormAdminTenantRepository) GetStatistics(ctx context.Context) (*identity.TenantStatistics, error) {
	stats := &identity.TenantStatistics{
		ByPlan: make(map[string]int64),
	}

	// Get total count
	if err := r.db.WithContext(ctx).Model(&models.TenantModel{}).Count(&stats.TotalTenants).Error; err != nil {
		return nil, err
	}

	// Get counts by status
	statusCounts, err := r.CountByStatus(ctx)
	if err != nil {
		return nil, err
	}
	stats.ActiveTenants = statusCounts[identity.TenantStatusActive]
	stats.InactiveTenants = statusCounts[identity.TenantStatusInactive]
	stats.SuspendedTenants = statusCounts[identity.TenantStatusSuspended]
	stats.TrialTenants = statusCounts[identity.TenantStatusTrial]

	// Get counts by plan
	planCounts, err := r.CountByPlan(ctx)
	if err != nil {
		return nil, err
	}
	for plan, count := range planCounts {
		stats.ByPlan[string(plan)] = count
	}

	// Get trial expiring within 7 days
	trialExpiring, err := r.FindTrialExpiring(ctx, 7)
	if err != nil {
		return nil, err
	}
	stats.TrialExpiring7d = int64(len(trialExpiring))

	// Get subscriptions expiring within 30 days
	subExpiring, err := r.FindSubscriptionExpiring(ctx, 30)
	if err != nil {
		return nil, err
	}
	stats.SubExpiring30d = int64(len(subExpiring))

	// Get expired tenants
	expired, err := r.FindExpired(ctx)
	if err != nil {
		return nil, err
	}
	stats.ExpiredTenants = int64(len(expired))

	return stats, nil
}

// buildQuery builds the base query with filters applied
func (r *GormAdminTenantRepository) buildQuery(ctx context.Context, filter identity.AdminTenantFilter) *gorm.DB {
	query := r.db.WithContext(ctx).Model(&models.TenantModel{})

	// Apply search (fuzzy match on name, code, contact_email)
	if filter.Search != "" {
		keyword := "%" + filter.Search + "%"
		query = query.Where(
			"name ILIKE ? OR code ILIKE ? OR short_name ILIKE ? OR contact_email ILIKE ?",
			keyword, keyword, keyword, keyword,
		)
	}

	// Apply status filter
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	// Apply plan type filter
	if filter.PlanType != nil {
		query = query.Where("plan = ?", *filter.PlanType)
	}

	// Apply created date range filters
	if filter.CreatedAfter != nil {
		query = query.Where("created_at >= ?", *filter.CreatedAfter)
	}
	if filter.CreatedBefore != nil {
		query = query.Where("created_at <= ?", *filter.CreatedBefore)
	}

	// Apply expiration date range filters
	if filter.ExpiresAfter != nil {
		query = query.Where("expires_at >= ?", *filter.ExpiresAfter)
	}
	if filter.ExpiresBefore != nil {
		query = query.Where("expires_at <= ?", *filter.ExpiresBefore)
	}

	// Apply trial expiring filter
	if filter.IsTrialExpiring != nil && *filter.IsTrialExpiring {
		expiryDate := time.Now().AddDate(0, 0, 7)
		query = query.Where("status = ?", identity.TenantStatusTrial).
			Where("trial_ends_at IS NOT NULL").
			Where("trial_ends_at <= ?", expiryDate).
			Where("trial_ends_at > ?", time.Now())
	}

	// Apply expired filter
	if filter.IsExpired != nil && *filter.IsExpired {
		now := time.Now()
		query = query.Where(
			"(status = ? AND trial_ends_at IS NOT NULL AND trial_ends_at < ?) OR "+
				"(status = ? AND expires_at IS NOT NULL AND expires_at < ?)",
			identity.TenantStatusTrial, now,
			identity.TenantStatusActive, now,
		)
	}

	return query
}

// Ensure GormAdminTenantRepository implements AdminTenantRepository
var _ identity.AdminTenantRepository = (*GormAdminTenantRepository)(nil)
