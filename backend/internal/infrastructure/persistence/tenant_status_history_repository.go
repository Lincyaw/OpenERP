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

// GormTenantStatusHistoryRepository implements TenantStatusHistoryRepository using GORM
type GormTenantStatusHistoryRepository struct {
	db *gorm.DB
}

// NewGormTenantStatusHistoryRepository creates a new GormTenantStatusHistoryRepository
func NewGormTenantStatusHistoryRepository(db *gorm.DB) *GormTenantStatusHistoryRepository {
	return &GormTenantStatusHistoryRepository{db: db}
}

// Create creates a new status history record
func (r *GormTenantStatusHistoryRepository) Create(ctx context.Context, history *identity.TenantStatusHistory) error {
	model := models.TenantStatusHistoryModelFromDomain(history)
	return r.db.WithContext(ctx).Create(model).Error
}

// FindByTenantID retrieves status history for a tenant with pagination
func (r *GormTenantStatusHistoryRepository) FindByTenantID(ctx context.Context, tenantID uuid.UUID, filter identity.TenantStatusHistoryFilter) ([]identity.TenantStatusHistory, int64, error) {
	var historyModels []models.TenantStatusHistoryModel
	var total int64

	query := r.db.WithContext(ctx).Model(&models.TenantStatusHistoryModel{}).
		Where("tenant_id = ?", tenantID)

	// Apply filters
	if filter.ChangeType != nil {
		query = query.Where("change_type = ?", *filter.ChangeType)
	}
	if filter.StartDate != nil {
		query = query.Where("created_at >= ?", *filter.StartDate)
	}
	if filter.EndDate != nil {
		query = query.Where("created_at <= ?", *filter.EndDate)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination and ordering
	query = query.Order("created_at DESC").
		Offset(filter.Offset()).
		Limit(filter.Limit())

	if err := query.Find(&historyModels).Error; err != nil {
		return nil, 0, err
	}

	// Convert to domain entities
	histories := make([]identity.TenantStatusHistory, len(historyModels))
	for i, model := range historyModels {
		histories[i] = *model.ToDomain()
	}

	return histories, total, nil
}

// FindLatestByTenantID retrieves the most recent status change for a tenant
func (r *GormTenantStatusHistoryRepository) FindLatestByTenantID(ctx context.Context, tenantID uuid.UUID) (*identity.TenantStatusHistory, error) {
	var model models.TenantStatusHistoryModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("created_at DESC").
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindPendingReactivations finds all tenants with scheduled reactivation before the given time
func (r *GormTenantStatusHistoryRepository) FindPendingReactivations(ctx context.Context, before time.Time) ([]identity.TenantStatusHistory, error) {
	var historyModels []models.TenantStatusHistoryModel

	// Find the latest suspension record for each tenant that has a scheduled reactivation
	// and the reactivation time is before the given time
	subQuery := r.db.WithContext(ctx).Model(&models.TenantStatusHistoryModel{}).
		Select("MAX(id) as id").
		Where("change_type = ?", identity.TenantStatusChangeTypeSuspend).
		Where("scheduled_reactivate_at IS NOT NULL").
		Where("scheduled_reactivate_at <= ?", before).
		Group("tenant_id")

	if err := r.db.WithContext(ctx).
		Where("id IN (?)", subQuery).
		Find(&historyModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	histories := make([]identity.TenantStatusHistory, len(historyModels))
	for i, model := range historyModels {
		histories[i] = *model.ToDomain()
	}

	return histories, nil
}
