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

// GormSubscriptionHistoryRepository implements SubscriptionHistoryRepository using GORM
type GormSubscriptionHistoryRepository struct {
	db *gorm.DB
}

// NewGormSubscriptionHistoryRepository creates a new GormSubscriptionHistoryRepository
func NewGormSubscriptionHistoryRepository(db *gorm.DB) *GormSubscriptionHistoryRepository {
	return &GormSubscriptionHistoryRepository{db: db}
}

// Create creates a new subscription history record
func (r *GormSubscriptionHistoryRepository) Create(ctx context.Context, history *identity.SubscriptionHistory) error {
	model, err := models.SubscriptionHistoryModelFromDomain(history)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(model).Error
}

// FindByID finds a subscription history record by ID
func (r *GormSubscriptionHistoryRepository) FindByID(ctx context.Context, id uuid.UUID) (*identity.SubscriptionHistory, error) {
	var model models.SubscriptionHistoryModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByTenantID finds subscription history for a tenant with filtering and pagination
func (r *GormSubscriptionHistoryRepository) FindByTenantID(ctx context.Context, tenantID uuid.UUID, filter identity.SubscriptionHistoryFilter) ([]identity.SubscriptionHistory, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.SubscriptionHistoryModel{}).
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

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	sortField := "created_at"
	if filter.SortBy == "effective_at" {
		sortField = "effective_at"
	}
	sortOrder := "DESC"
	if filter.SortOrder == "asc" {
		sortOrder = "ASC"
	}
	query = query.Order(sortField + " " + sortOrder)

	// Apply pagination
	offset := filter.Offset()
	limit := filter.Limit()
	query = query.Offset(offset).Limit(limit)

	// Execute query
	var historyModels []models.SubscriptionHistoryModel
	if err := query.Find(&historyModels).Error; err != nil {
		return nil, 0, err
	}

	// Convert to domain entities
	histories := make([]identity.SubscriptionHistory, len(historyModels))
	for i, model := range historyModels {
		histories[i] = *model.ToDomain()
	}

	return histories, total, nil
}

// FindPendingDowngrades finds all tenants with scheduled plan downgrades that are due
func (r *GormSubscriptionHistoryRepository) FindPendingDowngrades(ctx context.Context, beforeTime time.Time) ([]identity.SubscriptionHistory, error) {
	var historyModels []models.SubscriptionHistoryModel

	// Find scheduled downgrades that are due (effective_at <= beforeTime and scheduled_at is not null)
	if err := r.db.WithContext(ctx).
		Where("change_type = ?", identity.SubscriptionChangeTypePlanDowngrade).
		Where("scheduled_at IS NOT NULL").
		Where("effective_at <= ?", beforeTime).
		Find(&historyModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	histories := make([]identity.SubscriptionHistory, len(historyModels))
	for i, model := range historyModels {
		histories[i] = *model.ToDomain()
	}

	return histories, nil
}

// FindLatestByTenantID finds the most recent subscription history for a tenant
func (r *GormSubscriptionHistoryRepository) FindLatestByTenantID(ctx context.Context, tenantID uuid.UUID) (*identity.SubscriptionHistory, error) {
	var model models.SubscriptionHistoryModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("created_at DESC").
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No history found is not an error
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// Allowed sort fields for subscription history
var SubscriptionHistorySortFields = map[string]bool{
	"created_at":   true,
	"effective_at": true,
}
