package persistence

import (
	"context"
	"errors"

	"github.com/erp/backend/internal/domain/printing"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PrintJobSortFields defines allowed sort fields for print jobs
var PrintJobSortFields = map[string]bool{
	"created_at":      true,
	"updated_at":      true,
	"document_type":   true,
	"document_number": true,
	"status":          true,
	"printed_at":      true,
}

// GormPrintJobRepository implements PrintJobRepository using GORM
type GormPrintJobRepository struct {
	db *gorm.DB
}

// NewGormPrintJobRepository creates a new GormPrintJobRepository
func NewGormPrintJobRepository(db *gorm.DB) *GormPrintJobRepository {
	return &GormPrintJobRepository{db: db}
}

// FindByID finds a job by ID
func (r *GormPrintJobRepository) FindByID(ctx context.Context, id uuid.UUID) (*printing.PrintJob, error) {
	var model models.PrintJobModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByIDForTenant finds a job by ID within a specific tenant
func (r *GormPrintJobRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*printing.PrintJob, error) {
	var model models.PrintJobModel
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

// FindAll finds all jobs with optional filtering
func (r *GormPrintJobRepository) FindAll(ctx context.Context, filter shared.Filter) ([]printing.PrintJob, error) {
	var jobModels []models.PrintJobModel
	query := r.applyFilter(r.db.WithContext(ctx).Model(&models.PrintJobModel{}), filter)

	if err := query.Find(&jobModels).Error; err != nil {
		return nil, err
	}

	jobs := make([]printing.PrintJob, len(jobModels))
	for i, model := range jobModels {
		jobs[i] = *model.ToDomain()
	}
	return jobs, nil
}

// FindAllForTenant finds all jobs for a specific tenant
func (r *GormPrintJobRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]printing.PrintJob, error) {
	var jobModels []models.PrintJobModel
	query := r.applyFilter(r.db.WithContext(ctx).Model(&models.PrintJobModel{}).Where("tenant_id = ?", tenantID), filter)

	if err := query.Find(&jobModels).Error; err != nil {
		return nil, err
	}

	jobs := make([]printing.PrintJob, len(jobModels))
	for i, model := range jobModels {
		jobs[i] = *model.ToDomain()
	}
	return jobs, nil
}

// FindByDocument finds all print jobs for a specific document
func (r *GormPrintJobRepository) FindByDocument(ctx context.Context, tenantID uuid.UUID, docType printing.DocType, documentID uuid.UUID) ([]printing.PrintJob, error) {
	var jobModels []models.PrintJobModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND document_type = ? AND document_id = ?", tenantID, string(docType), documentID).
		Order("created_at DESC").
		Find(&jobModels).Error; err != nil {
		return nil, err
	}

	jobs := make([]printing.PrintJob, len(jobModels))
	for i, model := range jobModels {
		jobs[i] = *model.ToDomain()
	}
	return jobs, nil
}

// FindRecent finds recent print jobs (within the last N days)
func (r *GormPrintJobRepository) FindRecent(ctx context.Context, tenantID uuid.UUID, days int, limit int) ([]printing.PrintJob, error) {
	var jobModels []models.PrintJobModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND created_at >= NOW() - INTERVAL '? days'", tenantID, days).
		Order("created_at DESC").
		Limit(limit).
		Find(&jobModels).Error; err != nil {
		return nil, err
	}

	jobs := make([]printing.PrintJob, len(jobModels))
	for i, model := range jobModels {
		jobs[i] = *model.ToDomain()
	}
	return jobs, nil
}

// FindPending finds all pending jobs for processing
func (r *GormPrintJobRepository) FindPending(ctx context.Context, limit int) ([]printing.PrintJob, error) {
	var jobModels []models.PrintJobModel
	if err := r.db.WithContext(ctx).
		Where("status = ?", "PENDING").
		Order("created_at ASC").
		Limit(limit).
		Find(&jobModels).Error; err != nil {
		return nil, err
	}

	jobs := make([]printing.PrintJob, len(jobModels))
	for i, model := range jobModels {
		jobs[i] = *model.ToDomain()
	}
	return jobs, nil
}

// Save saves a job (insert or update)
func (r *GormPrintJobRepository) Save(ctx context.Context, job *printing.PrintJob) error {
	model := models.PrintJobModelFromDomain(job)
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete deletes a job by ID
func (r *GormPrintJobRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.PrintJobModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// Count returns the total count of jobs matching the filter
func (r *GormPrintJobRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.PrintJobModel{})
	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountForTenant returns the total count of jobs for a tenant
func (r *GormPrintJobRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.PrintJobModel{}).Where("tenant_id = ?", tenantID)
	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByStatus counts jobs by status for a tenant
func (r *GormPrintJobRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status printing.JobStatus) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.PrintJobModel{}).
		Where("tenant_id = ? AND status = ?", tenantID, string(status)).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// DeleteOlderThan deletes jobs older than the specified number of days
func (r *GormPrintJobRepository) DeleteOlderThan(ctx context.Context, days int) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("created_at < NOW() - INTERVAL '? days'", days).
		Delete(&models.PrintJobModel{})
	return result.RowsAffected, result.Error
}

// applyFilter applies filter options to the query
func (r *GormPrintJobRepository) applyFilter(query *gorm.DB, filter shared.Filter) *gorm.DB {
	query = r.applyFilterWithoutPagination(query, filter)

	// Apply pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// Apply ordering
	if filter.OrderBy != "" {
		sortField := ValidateSortField(filter.OrderBy, PrintJobSortFields, "")
		if sortField != "" {
			sortOrder := ValidateSortOrder(filter.OrderDir)
			query = query.Order(sortField + " " + sortOrder)
		} else {
			query = query.Order("created_at DESC")
		}
	} else {
		query = query.Order("created_at DESC")
	}

	return query
}

// applyFilterWithoutPagination applies filter options without pagination
func (r *GormPrintJobRepository) applyFilterWithoutPagination(query *gorm.DB, filter shared.Filter) *gorm.DB {
	// Apply additional filters
	for key, value := range filter.Filters {
		switch key {
		case "document_type", "doc_type":
			query = query.Where("document_type = ?", value)
		case "status":
			query = query.Where("status = ?", value)
		case "template_id":
			query = query.Where("template_id = ?", value)
		case "document_id":
			query = query.Where("document_id = ?", value)
		case "printed_by":
			query = query.Where("printed_by = ?", value)
		}
	}

	return query
}

// Ensure GormPrintJobRepository implements PrintJobRepository
var _ printing.PrintJobRepository = (*GormPrintJobRepository)(nil)
