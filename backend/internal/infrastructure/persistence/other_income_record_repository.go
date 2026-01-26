package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/finance"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// GormOtherIncomeRecordRepository implements OtherIncomeRecordRepository using GORM
type GormOtherIncomeRecordRepository struct {
	db *gorm.DB
}

// NewGormOtherIncomeRecordRepository creates a new GormOtherIncomeRecordRepository
func NewGormOtherIncomeRecordRepository(db *gorm.DB) *GormOtherIncomeRecordRepository {
	return &GormOtherIncomeRecordRepository{db: db}
}

// FindByID finds an income record by its ID
func (r *GormOtherIncomeRecordRepository) FindByID(ctx context.Context, id uuid.UUID) (*finance.OtherIncomeRecord, error) {
	var model models.OtherIncomeRecordModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByIDForTenant finds an income record by ID for a specific tenant
func (r *GormOtherIncomeRecordRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*finance.OtherIncomeRecord, error) {
	var model models.OtherIncomeRecordModel
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

// FindByIncomeNumber finds by income number for a tenant
func (r *GormOtherIncomeRecordRepository) FindByIncomeNumber(ctx context.Context, tenantID uuid.UUID, incomeNumber string) (*finance.OtherIncomeRecord, error) {
	var model models.OtherIncomeRecordModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND income_number = ?", tenantID, incomeNumber).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindAllForTenant finds all income records for a tenant with filtering
func (r *GormOtherIncomeRecordRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter finance.OtherIncomeRecordFilter) ([]finance.OtherIncomeRecord, error) {
	var incomeModels []models.OtherIncomeRecordModel
	query := r.db.WithContext(ctx).Model(&models.OtherIncomeRecordModel{}).
		Where("tenant_id = ?", tenantID)
	query = r.applyFilter(query, filter)

	if err := query.Find(&incomeModels).Error; err != nil {
		return nil, err
	}
	incomes := make([]finance.OtherIncomeRecord, len(incomeModels))
	for i, model := range incomeModels {
		incomes[i] = *model.ToDomain()
	}
	return incomes, nil
}

// FindByCategory finds income records by category for a tenant
func (r *GormOtherIncomeRecordRepository) FindByCategory(ctx context.Context, tenantID uuid.UUID, category finance.IncomeCategory, filter finance.OtherIncomeRecordFilter) ([]finance.OtherIncomeRecord, error) {
	var incomeModels []models.OtherIncomeRecordModel
	query := r.db.WithContext(ctx).Model(&models.OtherIncomeRecordModel{}).
		Where("tenant_id = ? AND category = ?", tenantID, category)
	query = r.applyFilter(query, filter)

	if err := query.Find(&incomeModels).Error; err != nil {
		return nil, err
	}
	incomes := make([]finance.OtherIncomeRecord, len(incomeModels))
	for i, model := range incomeModels {
		incomes[i] = *model.ToDomain()
	}
	return incomes, nil
}

// FindByStatus finds income records by status for a tenant
func (r *GormOtherIncomeRecordRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status finance.IncomeStatus, filter finance.OtherIncomeRecordFilter) ([]finance.OtherIncomeRecord, error) {
	var incomeModels []models.OtherIncomeRecordModel
	query := r.db.WithContext(ctx).Model(&models.OtherIncomeRecordModel{}).
		Where("tenant_id = ? AND status = ?", tenantID, status)
	query = r.applyFilter(query, filter)

	if err := query.Find(&incomeModels).Error; err != nil {
		return nil, err
	}
	incomes := make([]finance.OtherIncomeRecord, len(incomeModels))
	for i, model := range incomeModels {
		incomes[i] = *model.ToDomain()
	}
	return incomes, nil
}

// FindDraft finds all draft income records for a tenant
func (r *GormOtherIncomeRecordRepository) FindDraft(ctx context.Context, tenantID uuid.UUID) ([]finance.OtherIncomeRecord, error) {
	var incomeModels []models.OtherIncomeRecordModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND status = ?", tenantID, finance.IncomeStatusDraft).
		Order("created_at DESC").
		Find(&incomeModels).Error; err != nil {
		return nil, err
	}
	incomes := make([]finance.OtherIncomeRecord, len(incomeModels))
	for i, model := range incomeModels {
		incomes[i] = *model.ToDomain()
	}
	return incomes, nil
}

// Save creates or updates an income record
func (r *GormOtherIncomeRecordRepository) Save(ctx context.Context, income *finance.OtherIncomeRecord) error {
	model := models.OtherIncomeRecordModelFromDomain(income)
	return r.db.WithContext(ctx).Save(model).Error
}

// SaveWithLock saves the income record with optimistic locking
func (r *GormOtherIncomeRecordRepository) SaveWithLock(ctx context.Context, income *finance.OtherIncomeRecord) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get current version
		var current models.OtherIncomeRecordModel
		if err := tx.Select("version").Where("id = ?", income.GetID()).First(&current).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// New record, just save
				model := models.OtherIncomeRecordModelFromDomain(income)
				return tx.Create(model).Error
			}
			return err
		}

		// Check version matches (domain model already incremented version)
		expectedVersion := income.GetVersion() - 1
		if current.Version != expectedVersion {
			return shared.NewDomainError("VERSION_CONFLICT", "Income record has been modified by another user")
		}

		// Update with version check
		model := models.OtherIncomeRecordModelFromDomain(income)
		result := tx.Model(model).
			Where("id = ? AND version = ?", income.GetID(), expectedVersion).
			Save(model)

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return shared.NewDomainError("VERSION_CONFLICT", "Income record has been modified by another user")
		}
		return nil
	})
}

// GenerateIncomeNumber generates a new income number for the tenant
func (r *GormOtherIncomeRecordRepository) GenerateIncomeNumber(ctx context.Context, tenantID uuid.UUID) (string, error) {
	var count int64
	today := time.Now()
	yearMonth := today.Format("200601")

	// Count incomes this month
	if err := r.db.WithContext(ctx).Model(&models.OtherIncomeRecordModel{}).
		Where("tenant_id = ? AND income_number LIKE ?", tenantID, fmt.Sprintf("INC-%s-%%", yearMonth)).
		Count(&count).Error; err != nil {
		return "", err
	}

	return fmt.Sprintf("INC-%s-%05d", yearMonth, count+1), nil
}

// CountForTenant counts income records for a tenant with filtering
func (r *GormOtherIncomeRecordRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter finance.OtherIncomeRecordFilter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.OtherIncomeRecordModel{}).
		Where("tenant_id = ?", tenantID)
	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// applyFilter applies filter conditions to query
func (r *GormOtherIncomeRecordRepository) applyFilter(query *gorm.DB, filter finance.OtherIncomeRecordFilter) *gorm.DB {
	query = r.applyFilterWithoutPagination(query, filter)

	// Apply sorting with whitelist validation to prevent SQL injection
	sortField := ValidateSortField(filter.OrderBy, OtherIncomeRecordSortFields, "created_at")
	sortOrder := ValidateSortOrder(filter.OrderDir)
	query = query.Order(fmt.Sprintf("%s %s", sortField, sortOrder))

	// Apply pagination
	if filter.PageSize > 0 {
		query = query.Limit(filter.PageSize)
		offset := (filter.Page - 1) * filter.PageSize
		if offset > 0 {
			query = query.Offset(offset)
		}
	}

	return query
}

// applyFilterWithoutPagination applies filter conditions without pagination
func (r *GormOtherIncomeRecordRepository) applyFilterWithoutPagination(query *gorm.DB, filter finance.OtherIncomeRecordFilter) *gorm.DB {
	// Search in income number and description
	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		query = query.Where("(income_number ILIKE ? OR description ILIKE ?)", searchPattern, searchPattern)
	}

	// Category filter
	if filter.Category != nil {
		query = query.Where("category = ?", *filter.Category)
	}

	// Status filter
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	// Receipt status filter
	if filter.ReceiptStatus != nil {
		query = query.Where("receipt_status = ?", *filter.ReceiptStatus)
	}

	// Date range filter
	if filter.FromDate != nil {
		query = query.Where("received_at >= ?", filter.FromDate)
	}
	if filter.ToDate != nil {
		query = query.Where("received_at <= ?", filter.ToDate)
	}

	// Amount range filter
	if filter.MinAmount != nil {
		query = query.Where("amount >= ?", *filter.MinAmount)
	}
	if filter.MaxAmount != nil {
		query = query.Where("amount <= ?", *filter.MaxAmount)
	}

	return query
}

// Delete soft deletes an income record
func (r *GormOtherIncomeRecordRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.OtherIncomeRecordModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// DeleteForTenant soft deletes an income record for a tenant
func (r *GormOtherIncomeRecordRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.OtherIncomeRecordModel{}, "tenant_id = ? AND id = ?", tenantID, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// CountByStatus counts income records by status for a tenant
func (r *GormOtherIncomeRecordRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status finance.IncomeStatus) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.OtherIncomeRecordModel{}).
		Where("tenant_id = ? AND status = ?", tenantID, status).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByCategory counts income records by category for a tenant
func (r *GormOtherIncomeRecordRepository) CountByCategory(ctx context.Context, tenantID uuid.UUID, category finance.IncomeCategory) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.OtherIncomeRecordModel{}).
		Where("tenant_id = ? AND category = ?", tenantID, category).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// SumByCategory calculates total amount by category for a tenant within a date range
func (r *GormOtherIncomeRecordRepository) SumByCategory(ctx context.Context, tenantID uuid.UUID, category finance.IncomeCategory, from, to time.Time) (decimal.Decimal, error) {
	var result struct {
		Total decimal.Decimal
	}
	if err := r.db.WithContext(ctx).Model(&models.OtherIncomeRecordModel{}).
		Select("COALESCE(SUM(amount), 0) as total").
		Where("tenant_id = ? AND category = ? AND received_at >= ? AND received_at <= ?", tenantID, category, from, to).
		Scan(&result).Error; err != nil {
		return decimal.Zero, err
	}
	return result.Total, nil
}

// SumForTenant calculates total income amount for a tenant within a date range
func (r *GormOtherIncomeRecordRepository) SumForTenant(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (decimal.Decimal, error) {
	var result struct {
		Total decimal.Decimal
	}
	if err := r.db.WithContext(ctx).Model(&models.OtherIncomeRecordModel{}).
		Select("COALESCE(SUM(amount), 0) as total").
		Where("tenant_id = ? AND received_at >= ? AND received_at <= ?", tenantID, from, to).
		Scan(&result).Error; err != nil {
		return decimal.Zero, err
	}
	return result.Total, nil
}

// SumConfirmedForTenant calculates total confirmed income amount for a tenant within a date range
func (r *GormOtherIncomeRecordRepository) SumConfirmedForTenant(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (decimal.Decimal, error) {
	var result struct {
		Total decimal.Decimal
	}
	if err := r.db.WithContext(ctx).Model(&models.OtherIncomeRecordModel{}).
		Select("COALESCE(SUM(amount), 0) as total").
		Where("tenant_id = ? AND status = ? AND received_at >= ? AND received_at <= ?", tenantID, finance.IncomeStatusConfirmed, from, to).
		Scan(&result).Error; err != nil {
		return decimal.Zero, err
	}
	return result.Total, nil
}

// ExistsByIncomeNumber checks if an income number exists for a tenant
func (r *GormOtherIncomeRecordRepository) ExistsByIncomeNumber(ctx context.Context, tenantID uuid.UUID, incomeNumber string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.OtherIncomeRecordModel{}).
		Where("tenant_id = ? AND income_number = ?", tenantID, incomeNumber).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
