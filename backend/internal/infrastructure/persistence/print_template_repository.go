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

// PrintTemplateSortFields defines allowed sort fields for print templates
var PrintTemplateSortFields = map[string]bool{
	"created_at":    true,
	"updated_at":    true,
	"name":          true,
	"document_type": true,
	"status":        true,
	"is_default":    true,
}

// GormPrintTemplateRepository implements PrintTemplateRepository using GORM
type GormPrintTemplateRepository struct {
	db *gorm.DB
}

// NewGormPrintTemplateRepository creates a new GormPrintTemplateRepository
func NewGormPrintTemplateRepository(db *gorm.DB) *GormPrintTemplateRepository {
	return &GormPrintTemplateRepository{db: db}
}

// FindByID finds a template by ID
func (r *GormPrintTemplateRepository) FindByID(ctx context.Context, id uuid.UUID) (*printing.PrintTemplate, error) {
	var model models.PrintTemplateModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByIDForTenant finds a template by ID within a specific tenant
func (r *GormPrintTemplateRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*printing.PrintTemplate, error) {
	var model models.PrintTemplateModel
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

// FindAll finds all templates with optional filtering
func (r *GormPrintTemplateRepository) FindAll(ctx context.Context, filter shared.Filter) ([]printing.PrintTemplate, error) {
	var templateModels []models.PrintTemplateModel
	query := r.applyFilter(r.db.WithContext(ctx).Model(&models.PrintTemplateModel{}), filter)

	if err := query.Find(&templateModels).Error; err != nil {
		return nil, err
	}

	templates := make([]printing.PrintTemplate, len(templateModels))
	for i, model := range templateModels {
		templates[i] = *model.ToDomain()
	}
	return templates, nil
}

// FindAllForTenant finds all templates for a specific tenant
func (r *GormPrintTemplateRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]printing.PrintTemplate, error) {
	var templateModels []models.PrintTemplateModel
	query := r.applyFilter(r.db.WithContext(ctx).Model(&models.PrintTemplateModel{}).Where("tenant_id = ?", tenantID), filter)

	if err := query.Find(&templateModels).Error; err != nil {
		return nil, err
	}

	templates := make([]printing.PrintTemplate, len(templateModels))
	for i, model := range templateModels {
		templates[i] = *model.ToDomain()
	}
	return templates, nil
}

// FindByDocType finds all templates for a specific document type
func (r *GormPrintTemplateRepository) FindByDocType(ctx context.Context, tenantID uuid.UUID, docType printing.DocType) ([]printing.PrintTemplate, error) {
	var templateModels []models.PrintTemplateModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND document_type = ?", tenantID, string(docType)).
		Order("is_default DESC, name ASC").
		Find(&templateModels).Error; err != nil {
		return nil, err
	}

	templates := make([]printing.PrintTemplate, len(templateModels))
	for i, model := range templateModels {
		templates[i] = *model.ToDomain()
	}
	return templates, nil
}

// FindDefault finds the default template for a document type within a tenant
func (r *GormPrintTemplateRepository) FindDefault(ctx context.Context, tenantID uuid.UUID, docType printing.DocType) (*printing.PrintTemplate, error) {
	var model models.PrintTemplateModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND document_type = ? AND is_default = ?", tenantID, string(docType), true).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No default template, return nil without error
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindActiveByDocType finds all active templates for a document type
func (r *GormPrintTemplateRepository) FindActiveByDocType(ctx context.Context, tenantID uuid.UUID, docType printing.DocType) ([]printing.PrintTemplate, error) {
	var templateModels []models.PrintTemplateModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND document_type = ? AND status = ?", tenantID, string(docType), "ACTIVE").
		Order("is_default DESC, name ASC").
		Find(&templateModels).Error; err != nil {
		return nil, err
	}

	templates := make([]printing.PrintTemplate, len(templateModels))
	for i, model := range templateModels {
		templates[i] = *model.ToDomain()
	}
	return templates, nil
}

// Save saves a template (insert or update)
func (r *GormPrintTemplateRepository) Save(ctx context.Context, template *printing.PrintTemplate) error {
	model := models.PrintTemplateModelFromDomain(template)
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete deletes a template by ID
func (r *GormPrintTemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.PrintTemplateModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// Count returns the total count of templates matching the filter
func (r *GormPrintTemplateRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.PrintTemplateModel{})
	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountForTenant returns the total count of templates for a tenant
func (r *GormPrintTemplateRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.PrintTemplateModel{}).Where("tenant_id = ?", tenantID)
	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ExistsByDocTypeAndName checks if a template with the given doc type and name exists
func (r *GormPrintTemplateRepository) ExistsByDocTypeAndName(ctx context.Context, tenantID uuid.UUID, docType printing.DocType, name string, excludeID *uuid.UUID) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).
		Model(&models.PrintTemplateModel{}).
		Where("tenant_id = ? AND document_type = ? AND name = ?", tenantID, string(docType), name)

	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ClearDefaultForDocType clears the default flag for all templates of a document type
func (r *GormPrintTemplateRepository) ClearDefaultForDocType(ctx context.Context, tenantID uuid.UUID, docType printing.DocType) error {
	return r.db.WithContext(ctx).
		Model(&models.PrintTemplateModel{}).
		Where("tenant_id = ? AND document_type = ? AND is_default = ?", tenantID, string(docType), true).
		Update("is_default", false).Error
}

// applyFilter applies filter options to the query
func (r *GormPrintTemplateRepository) applyFilter(query *gorm.DB, filter shared.Filter) *gorm.DB {
	query = r.applyFilterWithoutPagination(query, filter)

	// Apply pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// Apply ordering
	if filter.OrderBy != "" {
		sortField := ValidateSortField(filter.OrderBy, PrintTemplateSortFields, "")
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
func (r *GormPrintTemplateRepository) applyFilterWithoutPagination(query *gorm.DB, filter shared.Filter) *gorm.DB {
	// Apply search
	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		query = query.Where("name ILIKE ? OR description ILIKE ?", searchPattern, searchPattern)
	}

	// Apply additional filters
	for key, value := range filter.Filters {
		switch key {
		case "document_type", "doc_type":
			query = query.Where("document_type = ?", value)
		case "status":
			query = query.Where("status = ?", value)
		case "is_default":
			query = query.Where("is_default = ?", value)
		case "paper_size":
			query = query.Where("paper_size = ?", value)
		}
	}

	return query
}

// Ensure GormPrintTemplateRepository implements PrintTemplateRepository
var _ printing.PrintTemplateRepository = (*GormPrintTemplateRepository)(nil)
