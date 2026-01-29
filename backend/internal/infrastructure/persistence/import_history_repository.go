package persistence

import (
	"context"
	"errors"

	"github.com/erp/backend/internal/domain/bulk"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ImportHistorySortFields contains allowed sort fields for import histories
var ImportHistorySortFields = map[string]bool{
	"id":           true,
	"created_at":   true,
	"updated_at":   true,
	"entity_type":  true,
	"file_name":    true,
	"file_size":    true,
	"total_rows":   true,
	"success_rows": true,
	"error_rows":   true,
	"status":       true,
	"started_at":   true,
	"completed_at": true,
}

// GormImportHistoryRepository implements ImportHistoryRepository using GORM
type GormImportHistoryRepository struct {
	db *gorm.DB
}

// NewGormImportHistoryRepository creates a new GormImportHistoryRepository
func NewGormImportHistoryRepository(db *gorm.DB) *GormImportHistoryRepository {
	return &GormImportHistoryRepository{db: db}
}

// FindByID finds an import history by ID
func (r *GormImportHistoryRepository) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*bulk.ImportHistory, error) {
	var model models.ImportHistoryModel
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

// FindAll returns all import histories for a tenant with pagination and filtering
func (r *GormImportHistoryRepository) FindAll(
	ctx context.Context,
	tenantID uuid.UUID,
	filter bulk.ImportHistoryFilter,
	page, pageSize int,
) (*bulk.ImportHistoryListResult, error) {
	query := r.db.WithContext(ctx).Model(&models.ImportHistoryModel{}).
		Where("tenant_id = ?", tenantID)

	// Apply filters
	query = r.applyFilters(query, filter)

	// Get total count
	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, err
	}

	// Apply pagination
	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		query = query.Offset(offset).Limit(pageSize)
	}

	// Default ordering: most recent first
	query = query.Order("started_at DESC NULLS LAST, created_at DESC")

	var historyModels []models.ImportHistoryModel
	if err := query.Find(&historyModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	histories := make([]*bulk.ImportHistory, len(historyModels))
	for i, model := range historyModels {
		histories[i] = model.ToDomain()
	}

	return &bulk.ImportHistoryListResult{
		Items:      histories,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
	}, nil
}

// FindByStatus finds all import histories with a specific status
func (r *GormImportHistoryRepository) FindByStatus(
	ctx context.Context,
	tenantID uuid.UUID,
	status bulk.ImportStatus,
) ([]*bulk.ImportHistory, error) {
	var historyModels []models.ImportHistoryModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND status = ?", tenantID, status).
		Order("created_at DESC").
		Find(&historyModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	histories := make([]*bulk.ImportHistory, len(historyModels))
	for i, model := range historyModels {
		histories[i] = model.ToDomain()
	}
	return histories, nil
}

// FindPending finds all pending import histories (for recovery after restart)
func (r *GormImportHistoryRepository) FindPending(ctx context.Context, tenantID uuid.UUID) ([]*bulk.ImportHistory, error) {
	var historyModels []models.ImportHistoryModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND status IN ?", tenantID,
			[]bulk.ImportStatus{bulk.ImportStatusPending, bulk.ImportStatusProcessing}).
		Order("created_at ASC").
		Find(&historyModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	histories := make([]*bulk.ImportHistory, len(historyModels))
	for i, model := range historyModels {
		histories[i] = model.ToDomain()
	}
	return histories, nil
}

// Save saves an import history (create or update)
func (r *GormImportHistoryRepository) Save(ctx context.Context, history *bulk.ImportHistory) error {
	model := models.ImportHistoryModelFromDomain(history)
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete deletes an import history by ID
func (r *GormImportHistoryRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Delete(&models.ImportHistoryModel{}, "tenant_id = ? AND id = ?", tenantID, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// applyFilters applies filter options to the query
func (r *GormImportHistoryRepository) applyFilters(query *gorm.DB, filter bulk.ImportHistoryFilter) *gorm.DB {
	if filter.EntityType != nil {
		query = query.Where("entity_type = ?", *filter.EntityType)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.ImportedBy != nil {
		query = query.Where("imported_by = ?", *filter.ImportedBy)
	}
	if filter.StartedFrom != nil {
		query = query.Where("started_at >= ?", *filter.StartedFrom)
	}
	if filter.StartedTo != nil {
		query = query.Where("started_at <= ?", *filter.StartedTo)
	}
	return query
}

// Compile-time interface compliance check
var _ bulk.ImportHistoryRepository = (*GormImportHistoryRepository)(nil)
