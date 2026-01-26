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

// GormAccountPayableRepository implements AccountPayableRepository using GORM
type GormAccountPayableRepository struct {
	db *gorm.DB
}

// NewGormAccountPayableRepository creates a new GormAccountPayableRepository
func NewGormAccountPayableRepository(db *gorm.DB) *GormAccountPayableRepository {
	return &GormAccountPayableRepository{db: db}
}

// FindByID finds an account payable by its ID
func (r *GormAccountPayableRepository) FindByID(ctx context.Context, id uuid.UUID) (*finance.AccountPayable, error) {
	var model models.AccountPayableModel
	if err := r.db.WithContext(ctx).
		Preload("PaymentRecords").
		First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByIDForTenant finds an account payable by ID for a specific tenant
func (r *GormAccountPayableRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*finance.AccountPayable, error) {
	var model models.AccountPayableModel
	if err := r.db.WithContext(ctx).
		Preload("PaymentRecords").
		Where("tenant_id = ? AND id = ?", tenantID, id).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByPayableNumber finds by payable number for a tenant
func (r *GormAccountPayableRepository) FindByPayableNumber(ctx context.Context, tenantID uuid.UUID, payableNumber string) (*finance.AccountPayable, error) {
	var model models.AccountPayableModel
	if err := r.db.WithContext(ctx).
		Preload("PaymentRecords").
		Where("tenant_id = ? AND payable_number = ?", tenantID, payableNumber).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindBySource finds by source document
func (r *GormAccountPayableRepository) FindBySource(ctx context.Context, tenantID uuid.UUID, sourceType finance.PayableSourceType, sourceID uuid.UUID) (*finance.AccountPayable, error) {
	var model models.AccountPayableModel
	if err := r.db.WithContext(ctx).
		Preload("PaymentRecords").
		Where("tenant_id = ? AND source_type = ? AND source_id = ?", tenantID, sourceType, sourceID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindAllForTenant finds all account payables for a tenant with filtering
func (r *GormAccountPayableRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter finance.AccountPayableFilter) ([]finance.AccountPayable, error) {
	var payableModels []models.AccountPayableModel
	query := r.db.WithContext(ctx).Model(&models.AccountPayableModel{}).
		Preload("PaymentRecords").
		Where("tenant_id = ?", tenantID)
	query = r.applyPayableFilter(query, filter)

	if err := query.Find(&payableModels).Error; err != nil {
		return nil, err
	}
	payables := make([]finance.AccountPayable, len(payableModels))
	for i, model := range payableModels {
		payables[i] = *model.ToDomain()
	}
	return payables, nil
}

// FindBySupplier finds account payables for a supplier
func (r *GormAccountPayableRepository) FindBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID, filter finance.AccountPayableFilter) ([]finance.AccountPayable, error) {
	var payableModels []models.AccountPayableModel
	query := r.db.WithContext(ctx).Model(&models.AccountPayableModel{}).
		Preload("PaymentRecords").
		Where("tenant_id = ? AND supplier_id = ?", tenantID, supplierID)
	query = r.applyPayableFilter(query, filter)

	if err := query.Find(&payableModels).Error; err != nil {
		return nil, err
	}
	payables := make([]finance.AccountPayable, len(payableModels))
	for i, model := range payableModels {
		payables[i] = *model.ToDomain()
	}
	return payables, nil
}

// FindByStatus finds account payables by status for a tenant
func (r *GormAccountPayableRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status finance.PayableStatus, filter finance.AccountPayableFilter) ([]finance.AccountPayable, error) {
	var payableModels []models.AccountPayableModel
	query := r.db.WithContext(ctx).Model(&models.AccountPayableModel{}).
		Preload("PaymentRecords").
		Where("tenant_id = ? AND status = ?", tenantID, status)
	query = r.applyPayableFilter(query, filter)

	if err := query.Find(&payableModels).Error; err != nil {
		return nil, err
	}
	payables := make([]finance.AccountPayable, len(payableModels))
	for i, model := range payableModels {
		payables[i] = *model.ToDomain()
	}
	return payables, nil
}

// FindOutstanding finds all outstanding payables for a supplier
func (r *GormAccountPayableRepository) FindOutstanding(ctx context.Context, tenantID, supplierID uuid.UUID) ([]finance.AccountPayable, error) {
	var payableModels []models.AccountPayableModel
	if err := r.db.WithContext(ctx).
		Preload("PaymentRecords").
		Where("tenant_id = ? AND supplier_id = ? AND status IN ?", tenantID, supplierID,
			[]finance.PayableStatus{finance.PayableStatusPending, finance.PayableStatusPartial}).
		Order("created_at ASC").
		Find(&payableModels).Error; err != nil {
		return nil, err
	}
	payables := make([]finance.AccountPayable, len(payableModels))
	for i, model := range payableModels {
		payables[i] = *model.ToDomain()
	}
	return payables, nil
}

// FindOverdue finds all overdue payables for a tenant
func (r *GormAccountPayableRepository) FindOverdue(ctx context.Context, tenantID uuid.UUID, filter finance.AccountPayableFilter) ([]finance.AccountPayable, error) {
	var payableModels []models.AccountPayableModel
	query := r.db.WithContext(ctx).Model(&models.AccountPayableModel{}).
		Preload("PaymentRecords").
		Where("tenant_id = ? AND due_date < ? AND status IN ?", tenantID, time.Now(),
			[]finance.PayableStatus{finance.PayableStatusPending, finance.PayableStatusPartial})
	query = r.applyPayableFilter(query, filter)

	if err := query.Find(&payableModels).Error; err != nil {
		return nil, err
	}
	payables := make([]finance.AccountPayable, len(payableModels))
	for i, model := range payableModels {
		payables[i] = *model.ToDomain()
	}
	return payables, nil
}

// Save creates or updates an account payable
func (r *GormAccountPayableRepository) Save(ctx context.Context, payable *finance.AccountPayable) error {
	model := models.AccountPayableModelFromDomain(payable)
	return r.db.WithContext(ctx).Save(model).Error
}

// SaveWithLock saves with optimistic locking
func (r *GormAccountPayableRepository) SaveWithLock(ctx context.Context, payable *finance.AccountPayable) error {
	model := models.AccountPayableModelFromDomain(payable)
	result := r.db.WithContext(ctx).
		Model(model).
		Where("id = ? AND version = ?", payable.ID, payable.Version-1).
		Updates(model)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.NewDomainError("OPTIMISTIC_LOCK_ERROR", "The record has been modified by another transaction")
	}
	return nil
}

// Delete soft deletes an account payable
func (r *GormAccountPayableRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.AccountPayableModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// DeleteForTenant soft deletes an account payable for a tenant
func (r *GormAccountPayableRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.AccountPayableModel{}, "tenant_id = ? AND id = ?", tenantID, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// CountForTenant counts account payables for a tenant
func (r *GormAccountPayableRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter finance.AccountPayableFilter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.AccountPayableModel{}).
		Where("tenant_id = ?", tenantID)
	query = r.applyPayableFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByStatus counts account payables by status
func (r *GormAccountPayableRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status finance.PayableStatus) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.AccountPayableModel{}).
		Where("tenant_id = ? AND status = ?", tenantID, status).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountBySupplier counts account payables for a supplier
func (r *GormAccountPayableRepository) CountBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.AccountPayableModel{}).
		Where("tenant_id = ? AND supplier_id = ?", tenantID, supplierID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountOverdue counts overdue payables
func (r *GormAccountPayableRepository) CountOverdue(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.AccountPayableModel{}).
		Where("tenant_id = ? AND due_date < ? AND status IN ?", tenantID, time.Now(),
			[]finance.PayableStatus{finance.PayableStatusPending, finance.PayableStatusPartial}).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// SumOutstandingBySupplier calculates total outstanding for a supplier
func (r *GormAccountPayableRepository) SumOutstandingBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (decimal.Decimal, error) {
	var result struct {
		Total decimal.Decimal
	}
	if err := r.db.WithContext(ctx).
		Model(&models.AccountPayableModel{}).
		Select("COALESCE(SUM(outstanding_amount), 0) as total").
		Where("tenant_id = ? AND supplier_id = ? AND status IN ?", tenantID, supplierID,
			[]finance.PayableStatus{finance.PayableStatusPending, finance.PayableStatusPartial}).
		Scan(&result).Error; err != nil {
		return decimal.Zero, err
	}
	return result.Total, nil
}

// SumOutstandingForTenant calculates total outstanding for a tenant
func (r *GormAccountPayableRepository) SumOutstandingForTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	var result struct {
		Total decimal.Decimal
	}
	if err := r.db.WithContext(ctx).
		Model(&models.AccountPayableModel{}).
		Select("COALESCE(SUM(outstanding_amount), 0) as total").
		Where("tenant_id = ? AND status IN ?", tenantID,
			[]finance.PayableStatus{finance.PayableStatusPending, finance.PayableStatusPartial}).
		Scan(&result).Error; err != nil {
		return decimal.Zero, err
	}
	return result.Total, nil
}

// SumOverdueForTenant calculates total overdue amount for a tenant
func (r *GormAccountPayableRepository) SumOverdueForTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	var result struct {
		Total decimal.Decimal
	}
	if err := r.db.WithContext(ctx).
		Model(&models.AccountPayableModel{}).
		Select("COALESCE(SUM(outstanding_amount), 0) as total").
		Where("tenant_id = ? AND due_date < ? AND status IN ?", tenantID, time.Now(),
			[]finance.PayableStatus{finance.PayableStatusPending, finance.PayableStatusPartial}).
		Scan(&result).Error; err != nil {
		return decimal.Zero, err
	}
	return result.Total, nil
}

// ExistsByPayableNumber checks if a payable number exists
func (r *GormAccountPayableRepository) ExistsByPayableNumber(ctx context.Context, tenantID uuid.UUID, payableNumber string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.AccountPayableModel{}).
		Where("tenant_id = ? AND payable_number = ?", tenantID, payableNumber).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ExistsBySource checks if a payable exists for the given source
func (r *GormAccountPayableRepository) ExistsBySource(ctx context.Context, tenantID uuid.UUID, sourceType finance.PayableSourceType, sourceID uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.AccountPayableModel{}).
		Where("tenant_id = ? AND source_type = ? AND source_id = ?", tenantID, sourceType, sourceID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// GeneratePayableNumber generates a unique payable number
func (r *GormAccountPayableRepository) GeneratePayableNumber(ctx context.Context, tenantID uuid.UUID) (string, error) {
	// Format: AP-YYYYMMDD-XXXXX
	date := time.Now().Format("20060102")
	prefix := fmt.Sprintf("AP-%s-", date)

	// Find the highest number for today
	var maxNumber string
	if err := r.db.WithContext(ctx).
		Model(&models.AccountPayableModel{}).
		Select("payable_number").
		Where("tenant_id = ? AND payable_number LIKE ?", tenantID, prefix+"%").
		Order("payable_number DESC").
		Limit(1).
		Pluck("payable_number", &maxNumber).Error; err != nil {
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

// applyPayableFilter applies filter options to the query
func (r *GormAccountPayableRepository) applyPayableFilter(query *gorm.DB, filter finance.AccountPayableFilter) *gorm.DB {
	query = r.applyPayableFilterWithoutPagination(query, filter)

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

// applyPayableFilterWithoutPagination applies filter options without pagination
func (r *GormAccountPayableRepository) applyPayableFilterWithoutPagination(query *gorm.DB, filter finance.AccountPayableFilter) *gorm.DB {
	// Apply search
	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		query = query.Where("payable_number ILIKE ? OR supplier_name ILIKE ? OR source_number ILIKE ?",
			searchPattern, searchPattern, searchPattern)
	}

	// Apply specific filters
	if filter.SupplierID != nil {
		query = query.Where("supplier_id = ?", *filter.SupplierID)
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
			[]finance.PayableStatus{finance.PayableStatusPending, finance.PayableStatusPartial})
	}
	if filter.MinAmount != nil {
		query = query.Where("outstanding_amount >= ?", *filter.MinAmount)
	}
	if filter.MaxAmount != nil {
		query = query.Where("outstanding_amount <= ?", *filter.MaxAmount)
	}

	return query
}

// Ensure GormAccountPayableRepository implements AccountPayableRepository
var _ finance.AccountPayableRepository = (*GormAccountPayableRepository)(nil)
