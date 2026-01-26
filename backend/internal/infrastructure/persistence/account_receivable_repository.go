package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/finance"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// GormAccountReceivableRepository implements AccountReceivableRepository using GORM
type GormAccountReceivableRepository struct {
	db *gorm.DB
}

// NewGormAccountReceivableRepository creates a new GormAccountReceivableRepository
func NewGormAccountReceivableRepository(db *gorm.DB) *GormAccountReceivableRepository {
	return &GormAccountReceivableRepository{db: db}
}

// FindByID finds an account receivable by its ID
func (r *GormAccountReceivableRepository) FindByID(ctx context.Context, id uuid.UUID) (*finance.AccountReceivable, error) {
	var model models.AccountReceivableModel
	if err := r.db.WithContext(ctx).
		First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByIDForTenant finds an account receivable by ID for a specific tenant
func (r *GormAccountReceivableRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*finance.AccountReceivable, error) {
	var model models.AccountReceivableModel
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

// FindByReceivableNumber finds by receivable number for a tenant
func (r *GormAccountReceivableRepository) FindByReceivableNumber(ctx context.Context, tenantID uuid.UUID, receivableNumber string) (*finance.AccountReceivable, error) {
	var model models.AccountReceivableModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND receivable_number = ?", tenantID, receivableNumber).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindBySource finds by source document
func (r *GormAccountReceivableRepository) FindBySource(ctx context.Context, tenantID uuid.UUID, sourceType finance.SourceType, sourceID uuid.UUID) (*finance.AccountReceivable, error) {
	var model models.AccountReceivableModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND source_type = ? AND source_id = ?", tenantID, sourceType, sourceID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindAllForTenant finds all account receivables for a tenant with filtering
func (r *GormAccountReceivableRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter finance.AccountReceivableFilter) ([]finance.AccountReceivable, error) {
	var receivableModels []models.AccountReceivableModel
	query := r.db.WithContext(ctx).Model(&models.AccountReceivableModel{}).
		Where("tenant_id = ?", tenantID)
	query = r.applyReceivableFilter(query, filter)

	if err := query.Find(&receivableModels).Error; err != nil {
		return nil, err
	}
	receivables := make([]finance.AccountReceivable, len(receivableModels))
	for i, model := range receivableModels {
		receivables[i] = *model.ToDomain()
	}
	return receivables, nil
}

// FindByCustomer finds account receivables for a customer
func (r *GormAccountReceivableRepository) FindByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, filter finance.AccountReceivableFilter) ([]finance.AccountReceivable, error) {
	var receivableModels []models.AccountReceivableModel
	query := r.db.WithContext(ctx).Model(&models.AccountReceivableModel{}).
		Where("tenant_id = ? AND customer_id = ?", tenantID, customerID)
	query = r.applyReceivableFilter(query, filter)

	if err := query.Find(&receivableModels).Error; err != nil {
		return nil, err
	}
	receivables := make([]finance.AccountReceivable, len(receivableModels))
	for i, model := range receivableModels {
		receivables[i] = *model.ToDomain()
	}
	return receivables, nil
}

// FindByStatus finds account receivables by status for a tenant
func (r *GormAccountReceivableRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status finance.ReceivableStatus, filter finance.AccountReceivableFilter) ([]finance.AccountReceivable, error) {
	var receivableModels []models.AccountReceivableModel
	query := r.db.WithContext(ctx).Model(&models.AccountReceivableModel{}).
		Where("tenant_id = ? AND status = ?", tenantID, status)
	query = r.applyReceivableFilter(query, filter)

	if err := query.Find(&receivableModels).Error; err != nil {
		return nil, err
	}
	receivables := make([]finance.AccountReceivable, len(receivableModels))
	for i, model := range receivableModels {
		receivables[i] = *model.ToDomain()
	}
	return receivables, nil
}

// FindOutstanding finds all outstanding receivables for a customer
func (r *GormAccountReceivableRepository) FindOutstanding(ctx context.Context, tenantID, customerID uuid.UUID) ([]finance.AccountReceivable, error) {
	var receivableModels []models.AccountReceivableModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND customer_id = ? AND status IN ?", tenantID, customerID,
			[]finance.ReceivableStatus{finance.ReceivableStatusPending, finance.ReceivableStatusPartial}).
		Order("created_at ASC").
		Find(&receivableModels).Error; err != nil {
		return nil, err
	}
	receivables := make([]finance.AccountReceivable, len(receivableModels))
	for i, model := range receivableModels {
		receivables[i] = *model.ToDomain()
	}
	return receivables, nil
}

// FindOverdue finds all overdue receivables for a tenant
func (r *GormAccountReceivableRepository) FindOverdue(ctx context.Context, tenantID uuid.UUID, filter finance.AccountReceivableFilter) ([]finance.AccountReceivable, error) {
	var receivableModels []models.AccountReceivableModel
	query := r.db.WithContext(ctx).Model(&models.AccountReceivableModel{}).
		Where("tenant_id = ? AND due_date < ? AND status IN ?", tenantID, time.Now(),
			[]finance.ReceivableStatus{finance.ReceivableStatusPending, finance.ReceivableStatusPartial})
	query = r.applyReceivableFilter(query, filter)

	if err := query.Find(&receivableModels).Error; err != nil {
		return nil, err
	}
	receivables := make([]finance.AccountReceivable, len(receivableModels))
	for i, model := range receivableModels {
		receivables[i] = *model.ToDomain()
	}
	return receivables, nil
}

// Save creates or updates an account receivable
func (r *GormAccountReceivableRepository) Save(ctx context.Context, receivable *finance.AccountReceivable) error {
	model := models.AccountReceivableModelFromDomain(receivable)
	return r.db.WithContext(ctx).Save(model).Error
}

// SaveWithLock saves with optimistic locking
func (r *GormAccountReceivableRepository) SaveWithLock(ctx context.Context, receivable *finance.AccountReceivable) error {
	model := models.AccountReceivableModelFromDomain(receivable)
	result := r.db.WithContext(ctx).
		Model(model).
		Where("id = ? AND version = ?", receivable.ID, receivable.Version-1).
		Updates(model)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.NewDomainError("OPTIMISTIC_LOCK_ERROR", "The record has been modified by another transaction")
	}
	return nil
}

// Delete soft deletes an account receivable
func (r *GormAccountReceivableRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.AccountReceivableModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// DeleteForTenant soft deletes an account receivable for a tenant
func (r *GormAccountReceivableRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.AccountReceivableModel{}, "tenant_id = ? AND id = ?", tenantID, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// CountForTenant counts account receivables for a tenant
func (r *GormAccountReceivableRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter finance.AccountReceivableFilter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.AccountReceivableModel{}).
		Where("tenant_id = ?", tenantID)
	query = r.applyReceivableFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByStatus counts account receivables by status
func (r *GormAccountReceivableRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status finance.ReceivableStatus) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.AccountReceivableModel{}).
		Where("tenant_id = ? AND status = ?", tenantID, status).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByCustomer counts account receivables for a customer
func (r *GormAccountReceivableRepository) CountByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.AccountReceivableModel{}).
		Where("tenant_id = ? AND customer_id = ?", tenantID, customerID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountOverdue counts overdue receivables
func (r *GormAccountReceivableRepository) CountOverdue(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.AccountReceivableModel{}).
		Where("tenant_id = ? AND due_date < ? AND status IN ?", tenantID, time.Now(),
			[]finance.ReceivableStatus{finance.ReceivableStatusPending, finance.ReceivableStatusPartial}).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// SumOutstandingByCustomer calculates total outstanding for a customer
func (r *GormAccountReceivableRepository) SumOutstandingByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (decimal.Decimal, error) {
	var result struct {
		Total decimal.Decimal
	}
	if err := r.db.WithContext(ctx).
		Model(&models.AccountReceivableModel{}).
		Select("COALESCE(SUM(outstanding_amount), 0) as total").
		Where("tenant_id = ? AND customer_id = ? AND status IN ?", tenantID, customerID,
			[]finance.ReceivableStatus{finance.ReceivableStatusPending, finance.ReceivableStatusPartial}).
		Scan(&result).Error; err != nil {
		return decimal.Zero, err
	}
	return result.Total, nil
}

// SumOutstandingForTenant calculates total outstanding for a tenant
func (r *GormAccountReceivableRepository) SumOutstandingForTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	var result struct {
		Total decimal.Decimal
	}
	if err := r.db.WithContext(ctx).
		Model(&models.AccountReceivableModel{}).
		Select("COALESCE(SUM(outstanding_amount), 0) as total").
		Where("tenant_id = ? AND status IN ?", tenantID,
			[]finance.ReceivableStatus{finance.ReceivableStatusPending, finance.ReceivableStatusPartial}).
		Scan(&result).Error; err != nil {
		return decimal.Zero, err
	}
	return result.Total, nil
}

// SumOverdueForTenant calculates total overdue amount for a tenant
func (r *GormAccountReceivableRepository) SumOverdueForTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	var result struct {
		Total decimal.Decimal
	}
	if err := r.db.WithContext(ctx).
		Model(&models.AccountReceivableModel{}).
		Select("COALESCE(SUM(outstanding_amount), 0) as total").
		Where("tenant_id = ? AND due_date < ? AND status IN ?", tenantID, time.Now(),
			[]finance.ReceivableStatus{finance.ReceivableStatusPending, finance.ReceivableStatusPartial}).
		Scan(&result).Error; err != nil {
		return decimal.Zero, err
	}
	return result.Total, nil
}

// ExistsByReceivableNumber checks if a receivable number exists
func (r *GormAccountReceivableRepository) ExistsByReceivableNumber(ctx context.Context, tenantID uuid.UUID, receivableNumber string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.AccountReceivableModel{}).
		Where("tenant_id = ? AND receivable_number = ?", tenantID, receivableNumber).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ExistsBySource checks if a receivable exists for the given source
func (r *GormAccountReceivableRepository) ExistsBySource(ctx context.Context, tenantID uuid.UUID, sourceType finance.SourceType, sourceID uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.AccountReceivableModel{}).
		Where("tenant_id = ? AND source_type = ? AND source_id = ?", tenantID, sourceType, sourceID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// GenerateReceivableNumber generates a unique receivable number
func (r *GormAccountReceivableRepository) GenerateReceivableNumber(ctx context.Context, tenantID uuid.UUID) (string, error) {
	// Format: AR-YYYYMMDD-XXXXX
	date := time.Now().Format("20060102")
	prefix := fmt.Sprintf("AR-%s-", date)

	// Find the highest number for today
	var maxNumber string
	if err := r.db.WithContext(ctx).
		Model(&models.AccountReceivableModel{}).
		Select("receivable_number").
		Where("tenant_id = ? AND receivable_number LIKE ?", tenantID, prefix+"%").
		Order("receivable_number DESC").
		Limit(1).
		Pluck("receivable_number", &maxNumber).Error; err != nil {
		return "", err
	}

	var nextNum int
	if maxNumber != "" {
		// Extract the number part
		parts := strings.Split(maxNumber, "-")
		if len(parts) == 3 {
			fmt.Sscanf(parts[2], "%d", &nextNum)
		}
	}
	nextNum++

	return fmt.Sprintf("%s%05d", prefix, nextNum), nil
}

// applyReceivableFilter applies filter options to the query
func (r *GormAccountReceivableRepository) applyReceivableFilter(query *gorm.DB, filter finance.AccountReceivableFilter) *gorm.DB {
	query = r.applyReceivableFilterWithoutPagination(query, filter)

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
		query = query.Order("created_at DESC")
	}

	return query
}

// applyReceivableFilterWithoutPagination applies filter options without pagination
func (r *GormAccountReceivableRepository) applyReceivableFilterWithoutPagination(query *gorm.DB, filter finance.AccountReceivableFilter) *gorm.DB {
	// Apply search
	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		query = query.Where("receivable_number ILIKE ? OR customer_name ILIKE ? OR source_number ILIKE ?",
			searchPattern, searchPattern, searchPattern)
	}

	// Apply specific filters
	if filter.CustomerID != nil {
		query = query.Where("customer_id = ?", *filter.CustomerID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.SourceType != nil {
		query = query.Where("source_type = ?", *filter.SourceType)
	}
	if filter.SourceID != nil {
		query = query.Where("source_id = ?", *filter.SourceID)
	}
	if filter.FromDate != nil {
		query = query.Where("created_at >= ?", *filter.FromDate)
	}
	if filter.ToDate != nil {
		query = query.Where("created_at <= ?", *filter.ToDate)
	}
	if filter.DueFrom != nil {
		query = query.Where("due_date >= ?", *filter.DueFrom)
	}
	if filter.DueTo != nil {
		query = query.Where("due_date <= ?", *filter.DueTo)
	}
	if filter.Overdue != nil && *filter.Overdue {
		query = query.Where("due_date < ? AND status IN ?", time.Now(),
			[]finance.ReceivableStatus{finance.ReceivableStatusPending, finance.ReceivableStatusPartial})
	}
	if filter.MinAmount != nil {
		query = query.Where("outstanding_amount >= ?", *filter.MinAmount)
	}
	if filter.MaxAmount != nil {
		query = query.Where("outstanding_amount <= ?", *filter.MaxAmount)
	}

	return query
}

// Ensure GormAccountReceivableRepository implements AccountReceivableRepository
var _ finance.AccountReceivableRepository = (*GormAccountReceivableRepository)(nil)
