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

// GormExpenseRecordRepository implements ExpenseRecordRepository using GORM
type GormExpenseRecordRepository struct {
	db *gorm.DB
}

// NewGormExpenseRecordRepository creates a new GormExpenseRecordRepository
func NewGormExpenseRecordRepository(db *gorm.DB) *GormExpenseRecordRepository {
	return &GormExpenseRecordRepository{db: db}
}

// FindByID finds an expense record by its ID
func (r *GormExpenseRecordRepository) FindByID(ctx context.Context, id uuid.UUID) (*finance.ExpenseRecord, error) {
	var model models.ExpenseRecordModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByIDForTenant finds an expense record by ID for a specific tenant
func (r *GormExpenseRecordRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*finance.ExpenseRecord, error) {
	var model models.ExpenseRecordModel
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

// FindByExpenseNumber finds by expense number for a tenant
func (r *GormExpenseRecordRepository) FindByExpenseNumber(ctx context.Context, tenantID uuid.UUID, expenseNumber string) (*finance.ExpenseRecord, error) {
	var model models.ExpenseRecordModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND expense_number = ?", tenantID, expenseNumber).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindAllForTenant finds all expense records for a tenant with filtering
func (r *GormExpenseRecordRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter finance.ExpenseRecordFilter) ([]finance.ExpenseRecord, error) {
	var expenseModels []models.ExpenseRecordModel
	query := r.db.WithContext(ctx).Model(&models.ExpenseRecordModel{}).
		Where("tenant_id = ?", tenantID)
	query = r.applyFilter(query, filter)

	if err := query.Find(&expenseModels).Error; err != nil {
		return nil, err
	}
	expenses := make([]finance.ExpenseRecord, len(expenseModels))
	for i, model := range expenseModels {
		expenses[i] = *model.ToDomain()
	}
	return expenses, nil
}

// FindByCategory finds expense records by category for a tenant
func (r *GormExpenseRecordRepository) FindByCategory(ctx context.Context, tenantID uuid.UUID, category finance.ExpenseCategory, filter finance.ExpenseRecordFilter) ([]finance.ExpenseRecord, error) {
	var expenseModels []models.ExpenseRecordModel
	query := r.db.WithContext(ctx).Model(&models.ExpenseRecordModel{}).
		Where("tenant_id = ? AND category = ?", tenantID, category)
	query = r.applyFilter(query, filter)

	if err := query.Find(&expenseModels).Error; err != nil {
		return nil, err
	}
	expenses := make([]finance.ExpenseRecord, len(expenseModels))
	for i, model := range expenseModels {
		expenses[i] = *model.ToDomain()
	}
	return expenses, nil
}

// FindByStatus finds expense records by status for a tenant
func (r *GormExpenseRecordRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status finance.ExpenseStatus, filter finance.ExpenseRecordFilter) ([]finance.ExpenseRecord, error) {
	var expenseModels []models.ExpenseRecordModel
	query := r.db.WithContext(ctx).Model(&models.ExpenseRecordModel{}).
		Where("tenant_id = ? AND status = ?", tenantID, status)
	query = r.applyFilter(query, filter)

	if err := query.Find(&expenseModels).Error; err != nil {
		return nil, err
	}
	expenses := make([]finance.ExpenseRecord, len(expenseModels))
	for i, model := range expenseModels {
		expenses[i] = *model.ToDomain()
	}
	return expenses, nil
}

// FindPendingApproval finds all pending approval expenses for a tenant
func (r *GormExpenseRecordRepository) FindPendingApproval(ctx context.Context, tenantID uuid.UUID) ([]finance.ExpenseRecord, error) {
	var expenseModels []models.ExpenseRecordModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND status = ?", tenantID, finance.ExpenseStatusPending).
		Order("submitted_at ASC").
		Find(&expenseModels).Error; err != nil {
		return nil, err
	}
	expenses := make([]finance.ExpenseRecord, len(expenseModels))
	for i, model := range expenseModels {
		expenses[i] = *model.ToDomain()
	}
	return expenses, nil
}

// Save creates or updates an expense record
func (r *GormExpenseRecordRepository) Save(ctx context.Context, expense *finance.ExpenseRecord) error {
	model := models.ExpenseRecordModelFromDomain(expense)
	return r.db.WithContext(ctx).Save(model).Error
}

// SaveWithLock saves the expense record with optimistic locking
func (r *GormExpenseRecordRepository) SaveWithLock(ctx context.Context, expense *finance.ExpenseRecord) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get current version
		var current models.ExpenseRecordModel
		if err := tx.Select("version").Where("id = ?", expense.GetID()).First(&current).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// New record, just save
				model := models.ExpenseRecordModelFromDomain(expense)
				return tx.Create(model).Error
			}
			return err
		}

		// Check version matches (domain model already incremented version)
		expectedVersion := expense.GetVersion() - 1
		if current.Version != expectedVersion {
			return shared.NewDomainError("VERSION_CONFLICT", "Expense record has been modified by another user")
		}

		// Update with version check
		model := models.ExpenseRecordModelFromDomain(expense)
		result := tx.Model(model).
			Where("id = ? AND version = ?", expense.GetID(), expectedVersion).
			Save(model)

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return shared.NewDomainError("VERSION_CONFLICT", "Expense record has been modified by another user")
		}
		return nil
	})
}

// GenerateExpenseNumber generates a new expense number for the tenant
func (r *GormExpenseRecordRepository) GenerateExpenseNumber(ctx context.Context, tenantID uuid.UUID) (string, error) {
	var count int64
	today := time.Now()
	yearMonth := today.Format("200601")

	// Count expenses this month
	if err := r.db.WithContext(ctx).Model(&models.ExpenseRecordModel{}).
		Where("tenant_id = ? AND expense_number LIKE ?", tenantID, fmt.Sprintf("EXP-%s-%%", yearMonth)).
		Count(&count).Error; err != nil {
		return "", err
	}

	return fmt.Sprintf("EXP-%s-%05d", yearMonth, count+1), nil
}

// CountForTenant counts expense records for a tenant with filtering
func (r *GormExpenseRecordRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter finance.ExpenseRecordFilter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.ExpenseRecordModel{}).
		Where("tenant_id = ?", tenantID)
	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// applyFilter applies filter conditions to query
func (r *GormExpenseRecordRepository) applyFilter(query *gorm.DB, filter finance.ExpenseRecordFilter) *gorm.DB {
	query = r.applyFilterWithoutPagination(query, filter)

	// Apply sorting with whitelist validation to prevent SQL injection
	sortField := ValidateSortField(filter.OrderBy, ExpenseRecordSortFields, "created_at")
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
func (r *GormExpenseRecordRepository) applyFilterWithoutPagination(query *gorm.DB, filter finance.ExpenseRecordFilter) *gorm.DB {
	// Search in expense number and description
	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		query = query.Where("(expense_number ILIKE ? OR description ILIKE ?)", searchPattern, searchPattern)
	}

	// Category filter
	if filter.Category != nil {
		query = query.Where("category = ?", *filter.Category)
	}

	// Status filter
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	// Payment status filter
	if filter.PaymentStatus != nil {
		query = query.Where("payment_status = ?", *filter.PaymentStatus)
	}

	// Date range filter
	if filter.FromDate != nil {
		query = query.Where("incurred_at >= ?", filter.FromDate)
	}
	if filter.ToDate != nil {
		query = query.Where("incurred_at <= ?", filter.ToDate)
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

// Delete soft deletes an expense record
func (r *GormExpenseRecordRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.ExpenseRecordModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// DeleteForTenant soft deletes an expense record for a tenant
func (r *GormExpenseRecordRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.ExpenseRecordModel{}, "tenant_id = ? AND id = ?", tenantID, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// CountByStatus counts expense records by status for a tenant
func (r *GormExpenseRecordRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status finance.ExpenseStatus) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.ExpenseRecordModel{}).
		Where("tenant_id = ? AND status = ?", tenantID, status).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByCategory counts expense records by category for a tenant
func (r *GormExpenseRecordRepository) CountByCategory(ctx context.Context, tenantID uuid.UUID, category finance.ExpenseCategory) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.ExpenseRecordModel{}).
		Where("tenant_id = ? AND category = ?", tenantID, category).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// SumByCategory calculates total amount by category for a tenant within a date range
func (r *GormExpenseRecordRepository) SumByCategory(ctx context.Context, tenantID uuid.UUID, category finance.ExpenseCategory, from, to time.Time) (decimal.Decimal, error) {
	var result struct {
		Total decimal.Decimal
	}
	if err := r.db.WithContext(ctx).Model(&models.ExpenseRecordModel{}).
		Select("COALESCE(SUM(amount), 0) as total").
		Where("tenant_id = ? AND category = ? AND incurred_at >= ? AND incurred_at <= ?", tenantID, category, from, to).
		Scan(&result).Error; err != nil {
		return decimal.Zero, err
	}
	return result.Total, nil
}

// SumForTenant calculates total expense amount for a tenant within a date range
func (r *GormExpenseRecordRepository) SumForTenant(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (decimal.Decimal, error) {
	var result struct {
		Total decimal.Decimal
	}
	if err := r.db.WithContext(ctx).Model(&models.ExpenseRecordModel{}).
		Select("COALESCE(SUM(amount), 0) as total").
		Where("tenant_id = ? AND incurred_at >= ? AND incurred_at <= ?", tenantID, from, to).
		Scan(&result).Error; err != nil {
		return decimal.Zero, err
	}
	return result.Total, nil
}

// SumApprovedForTenant calculates total approved expense amount for a tenant within a date range
func (r *GormExpenseRecordRepository) SumApprovedForTenant(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (decimal.Decimal, error) {
	var result struct {
		Total decimal.Decimal
	}
	if err := r.db.WithContext(ctx).Model(&models.ExpenseRecordModel{}).
		Select("COALESCE(SUM(amount), 0) as total").
		Where("tenant_id = ? AND status = ? AND incurred_at >= ? AND incurred_at <= ?", tenantID, finance.ExpenseStatusApproved, from, to).
		Scan(&result).Error; err != nil {
		return decimal.Zero, err
	}
	return result.Total, nil
}

// ExistsByExpenseNumber checks if an expense number exists for a tenant
func (r *GormExpenseRecordRepository) ExistsByExpenseNumber(ctx context.Context, tenantID uuid.UUID, expenseNumber string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.ExpenseRecordModel{}).
		Where("tenant_id = ? AND expense_number = ?", tenantID, expenseNumber).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
