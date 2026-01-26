package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GormBalanceTransactionRepository implements BalanceTransactionRepository using GORM
type GormBalanceTransactionRepository struct {
	db *gorm.DB
}

// NewGormBalanceTransactionRepository creates a new GormBalanceTransactionRepository
func NewGormBalanceTransactionRepository(db *gorm.DB) *GormBalanceTransactionRepository {
	return &GormBalanceTransactionRepository{db: db}
}

// Create creates a new balance transaction
func (r *GormBalanceTransactionRepository) Create(ctx context.Context, transaction *partner.BalanceTransaction) error {
	model := models.BalanceTransactionModelFromDomain(transaction)
	return r.db.WithContext(ctx).Create(model).Error
}

// FindByID finds a balance transaction by ID within a tenant
func (r *GormBalanceTransactionRepository) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*partner.BalanceTransaction, error) {
	var model models.BalanceTransactionModel
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

// FindByCustomerID finds all balance transactions for a customer
func (r *GormBalanceTransactionRepository) FindByCustomerID(ctx context.Context, tenantID, customerID uuid.UUID, filter partner.BalanceTransactionFilter) ([]*partner.BalanceTransaction, int64, error) {
	var transactionModels []models.BalanceTransactionModel
	var total int64

	query := r.db.WithContext(ctx).Model(&models.BalanceTransactionModel{}).
		Where("tenant_id = ? AND customer_id = ?", tenantID, customerID)

	query = r.applyFilter(query, filter)

	// Count total
	countQuery := r.db.WithContext(ctx).Model(&models.BalanceTransactionModel{}).
		Where("tenant_id = ? AND customer_id = ?", tenantID, customerID)
	countQuery = r.applyFilterWithoutPagination(countQuery, filter)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// Order by transaction date descending (most recent first)
	query = query.Order("transaction_date DESC")

	if err := query.Find(&transactionModels).Error; err != nil {
		return nil, 0, err
	}

	transactions := make([]*partner.BalanceTransaction, len(transactionModels))
	for i, model := range transactionModels {
		transactions[i] = model.ToDomain()
	}
	return transactions, total, nil
}

// FindBySourceID finds balance transactions by source document ID
func (r *GormBalanceTransactionRepository) FindBySourceID(ctx context.Context, tenantID uuid.UUID, sourceType partner.BalanceTransactionSourceType, sourceID string) ([]*partner.BalanceTransaction, error) {
	var transactionModels []models.BalanceTransactionModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND source_type = ? AND source_id = ?", tenantID, sourceType, sourceID).
		Order("transaction_date DESC").
		Find(&transactionModels).Error; err != nil {
		return nil, err
	}
	transactions := make([]*partner.BalanceTransaction, len(transactionModels))
	for i, model := range transactionModels {
		transactions[i] = model.ToDomain()
	}
	return transactions, nil
}

// List lists balance transactions with filtering
func (r *GormBalanceTransactionRepository) List(ctx context.Context, tenantID uuid.UUID, filter partner.BalanceTransactionFilter) ([]*partner.BalanceTransaction, int64, error) {
	var transactionModels []models.BalanceTransactionModel
	var total int64

	query := r.db.WithContext(ctx).Model(&models.BalanceTransactionModel{}).
		Where("tenant_id = ?", tenantID)

	query = r.applyFilter(query, filter)

	// Count total
	countQuery := r.db.WithContext(ctx).Model(&models.BalanceTransactionModel{}).
		Where("tenant_id = ?", tenantID)
	countQuery = r.applyFilterWithoutPagination(countQuery, filter)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// Order by transaction date descending (most recent first)
	query = query.Order("transaction_date DESC")

	if err := query.Find(&transactionModels).Error; err != nil {
		return nil, 0, err
	}

	transactions := make([]*partner.BalanceTransaction, len(transactionModels))
	for i, model := range transactionModels {
		transactions[i] = model.ToDomain()
	}
	return transactions, total, nil
}

// GetLatestByCustomerID gets the latest balance transaction for a customer
func (r *GormBalanceTransactionRepository) GetLatestByCustomerID(ctx context.Context, tenantID, customerID uuid.UUID) (*partner.BalanceTransaction, error) {
	var model models.BalanceTransactionModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND customer_id = ?", tenantID, customerID).
		Order("transaction_date DESC").
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// SumByCustomerIDAndType sums the amount by customer ID and transaction type within a date range
func (r *GormBalanceTransactionRepository) SumByCustomerIDAndType(ctx context.Context, tenantID, customerID uuid.UUID, txType partner.BalanceTransactionType, from, to time.Time) (float64, error) {
	var result struct {
		Total float64
	}

	if err := r.db.WithContext(ctx).
		Model(&models.BalanceTransactionModel{}).
		Select("COALESCE(SUM(amount), 0) as total").
		Where("tenant_id = ? AND customer_id = ? AND transaction_type = ? AND transaction_date >= ? AND transaction_date <= ?",
			tenantID, customerID, txType, from, to).
		Scan(&result).Error; err != nil {
		return 0, err
	}

	return result.Total, nil
}

// applyFilter applies filter options to the query
func (r *GormBalanceTransactionRepository) applyFilter(query *gorm.DB, filter partner.BalanceTransactionFilter) *gorm.DB {
	return r.applyFilterWithoutPagination(query, filter)
}

// applyFilterWithoutPagination applies filter options without pagination
func (r *GormBalanceTransactionRepository) applyFilterWithoutPagination(query *gorm.DB, filter partner.BalanceTransactionFilter) *gorm.DB {
	if filter.CustomerID != nil {
		query = query.Where("customer_id = ?", *filter.CustomerID)
	}

	if filter.TransactionType != nil {
		query = query.Where("transaction_type = ?", strings.ToUpper(string(*filter.TransactionType)))
	}

	if filter.SourceType != nil {
		query = query.Where("source_type = ?", strings.ToUpper(string(*filter.SourceType)))
	}

	if filter.DateFrom != nil {
		query = query.Where("transaction_date >= ?", *filter.DateFrom)
	}

	if filter.DateTo != nil {
		query = query.Where("transaction_date <= ?", *filter.DateTo)
	}

	return query
}

// Ensure GormBalanceTransactionRepository implements BalanceTransactionRepository
var _ partner.BalanceTransactionRepository = (*GormBalanceTransactionRepository)(nil)
