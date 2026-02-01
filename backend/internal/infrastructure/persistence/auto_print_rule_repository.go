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

// AutoPrintRuleSortFields defines allowed sort fields for auto print rules
var AutoPrintRuleSortFields = map[string]bool{
	"created_at":    true,
	"updated_at":    true,
	"document_type": true,
	"trigger_event": true,
	"enabled":       true,
}

// GormAutoPrintRuleRepository implements AutoPrintRuleRepository using GORM
type GormAutoPrintRuleRepository struct {
	db *gorm.DB
}

// NewGormAutoPrintRuleRepository creates a new GormAutoPrintRuleRepository
func NewGormAutoPrintRuleRepository(db *gorm.DB) *GormAutoPrintRuleRepository {
	return &GormAutoPrintRuleRepository{db: db}
}

// FindByID finds a rule by ID
func (r *GormAutoPrintRuleRepository) FindByID(ctx context.Context, id uuid.UUID) (*printing.AutoPrintRule, error) {
	var model models.AutoPrintRuleModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByIDForTenant finds a rule by ID within a specific tenant
func (r *GormAutoPrintRuleRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*printing.AutoPrintRule, error) {
	var model models.AutoPrintRuleModel
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

// FindAll finds all rules with optional filtering
func (r *GormAutoPrintRuleRepository) FindAll(ctx context.Context, filter shared.Filter) ([]printing.AutoPrintRule, error) {
	var ruleModels []models.AutoPrintRuleModel
	query := r.applyFilter(r.db.WithContext(ctx).Model(&models.AutoPrintRuleModel{}), filter)

	if err := query.Find(&ruleModels).Error; err != nil {
		return nil, err
	}

	rules := make([]printing.AutoPrintRule, len(ruleModels))
	for i, model := range ruleModels {
		rules[i] = *model.ToDomain()
	}
	return rules, nil
}

// FindAllForTenant finds all rules for a specific tenant
func (r *GormAutoPrintRuleRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]printing.AutoPrintRule, error) {
	var ruleModels []models.AutoPrintRuleModel
	query := r.applyFilter(r.db.WithContext(ctx).Model(&models.AutoPrintRuleModel{}).Where("tenant_id = ?", tenantID), filter)

	if err := query.Find(&ruleModels).Error; err != nil {
		return nil, err
	}

	rules := make([]printing.AutoPrintRule, len(ruleModels))
	for i, model := range ruleModels {
		rules[i] = *model.ToDomain()
	}
	return rules, nil
}

// FindByDocTypeAndEvent finds a rule by document type and trigger event for a tenant
func (r *GormAutoPrintRuleRepository) FindByDocTypeAndEvent(ctx context.Context, tenantID uuid.UUID, docType printing.DocType, event printing.TriggerEvent) (*printing.AutoPrintRule, error) {
	var model models.AutoPrintRuleModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND document_type = ? AND trigger_event = ?", tenantID, string(docType), string(event)).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil, nil when not found (not an error)
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindEnabledByDocTypeAndEvent finds an enabled rule by document type and trigger event
func (r *GormAutoPrintRuleRepository) FindEnabledByDocTypeAndEvent(ctx context.Context, tenantID uuid.UUID, docType printing.DocType, event printing.TriggerEvent) (*printing.AutoPrintRule, error) {
	var model models.AutoPrintRuleModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND document_type = ? AND trigger_event = ? AND enabled = ?", tenantID, string(docType), string(event), true).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil, nil when not found (not an error)
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindEnabledForTenant finds all enabled rules for a tenant
func (r *GormAutoPrintRuleRepository) FindEnabledForTenant(ctx context.Context, tenantID uuid.UUID) ([]printing.AutoPrintRule, error) {
	var ruleModels []models.AutoPrintRuleModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND enabled = ?", tenantID, true).
		Order("document_type ASC, trigger_event ASC").
		Find(&ruleModels).Error; err != nil {
		return nil, err
	}

	rules := make([]printing.AutoPrintRule, len(ruleModels))
	for i, model := range ruleModels {
		rules[i] = *model.ToDomain()
	}
	return rules, nil
}

// ExistsByDocTypeAndEvent checks if a rule exists for the given combination
func (r *GormAutoPrintRuleRepository) ExistsByDocTypeAndEvent(ctx context.Context, tenantID uuid.UUID, docType printing.DocType, event printing.TriggerEvent) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.AutoPrintRuleModel{}).
		Where("tenant_id = ? AND document_type = ? AND trigger_event = ?", tenantID, string(docType), string(event)).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// Save saves a rule (insert or update)
func (r *GormAutoPrintRuleRepository) Save(ctx context.Context, rule *printing.AutoPrintRule) error {
	model := models.AutoPrintRuleModelFromDomain(rule)
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete deletes a rule by ID
func (r *GormAutoPrintRuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.AutoPrintRuleModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// DeleteForTenant deletes a rule by ID within a specific tenant
func (r *GormAutoPrintRuleRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		Delete(&models.AutoPrintRuleModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// Count returns the total count of rules matching the filter
func (r *GormAutoPrintRuleRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.AutoPrintRuleModel{})
	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountForTenant returns the total count of rules for a tenant
func (r *GormAutoPrintRuleRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.AutoPrintRuleModel{}).Where("tenant_id = ?", tenantID)
	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// applyFilter applies filter options to the query
func (r *GormAutoPrintRuleRepository) applyFilter(query *gorm.DB, filter shared.Filter) *gorm.DB {
	query = r.applyFilterWithoutPagination(query, filter)

	// Apply pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// Apply ordering
	if filter.OrderBy != "" {
		sortField := ValidateSortField(filter.OrderBy, AutoPrintRuleSortFields, "")
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
func (r *GormAutoPrintRuleRepository) applyFilterWithoutPagination(query *gorm.DB, filter shared.Filter) *gorm.DB {
	// Apply additional filters
	for key, value := range filter.Filters {
		switch key {
		case "document_type", "doc_type":
			query = query.Where("document_type = ?", value)
		case "trigger_event", "event":
			query = query.Where("trigger_event = ?", value)
		case "enabled":
			query = query.Where("enabled = ?", value)
		case "auto_print":
			query = query.Where("auto_print = ?", value)
		}
	}

	return query
}

// Ensure GormAutoPrintRuleRepository implements AutoPrintRuleRepository
var _ printing.AutoPrintRuleRepository = (*GormAutoPrintRuleRepository)(nil)
