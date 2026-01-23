package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// GormInventoryTransactionRepository implements InventoryTransactionRepository using GORM
type GormInventoryTransactionRepository struct {
	db *gorm.DB
}

// NewGormInventoryTransactionRepository creates a new GormInventoryTransactionRepository
func NewGormInventoryTransactionRepository(db *gorm.DB) *GormInventoryTransactionRepository {
	return &GormInventoryTransactionRepository{db: db}
}

// FindByID finds a transaction by its ID
func (r *GormInventoryTransactionRepository) FindByID(ctx context.Context, id uuid.UUID) (*inventory.InventoryTransaction, error) {
	var tx inventory.InventoryTransaction
	if err := r.db.WithContext(ctx).First(&tx, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &tx, nil
}

// FindByInventoryItem finds transactions for an inventory item
func (r *GormInventoryTransactionRepository) FindByInventoryItem(ctx context.Context, inventoryItemID uuid.UUID, filter shared.Filter) ([]inventory.InventoryTransaction, error) {
	var txs []inventory.InventoryTransaction
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&inventory.InventoryTransaction{}).
			Where("inventory_item_id = ?", inventoryItemID),
		filter,
	)

	if err := query.Find(&txs).Error; err != nil {
		return nil, err
	}
	return txs, nil
}

// FindByWarehouse finds transactions for a warehouse
func (r *GormInventoryTransactionRepository) FindByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID, filter shared.Filter) ([]inventory.InventoryTransaction, error) {
	var txs []inventory.InventoryTransaction
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&inventory.InventoryTransaction{}).
			Where("tenant_id = ? AND warehouse_id = ?", tenantID, warehouseID),
		filter,
	)

	if err := query.Find(&txs).Error; err != nil {
		return nil, err
	}
	return txs, nil
}

// FindByProduct finds transactions for a product
func (r *GormInventoryTransactionRepository) FindByProduct(ctx context.Context, tenantID, productID uuid.UUID, filter shared.Filter) ([]inventory.InventoryTransaction, error) {
	var txs []inventory.InventoryTransaction
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&inventory.InventoryTransaction{}).
			Where("tenant_id = ? AND product_id = ?", tenantID, productID),
		filter,
	)

	if err := query.Find(&txs).Error; err != nil {
		return nil, err
	}
	return txs, nil
}

// FindBySource finds transactions by source document
func (r *GormInventoryTransactionRepository) FindBySource(ctx context.Context, sourceType inventory.SourceType, sourceID string) ([]inventory.InventoryTransaction, error) {
	var txs []inventory.InventoryTransaction
	if err := r.db.WithContext(ctx).
		Where("source_type = ? AND source_id = ?", sourceType, sourceID).
		Order("transaction_date ASC").
		Find(&txs).Error; err != nil {
		return nil, err
	}
	return txs, nil
}

// FindByDateRange finds transactions within a date range
func (r *GormInventoryTransactionRepository) FindByDateRange(ctx context.Context, tenantID uuid.UUID, start, end time.Time, filter shared.Filter) ([]inventory.InventoryTransaction, error) {
	var txs []inventory.InventoryTransaction
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&inventory.InventoryTransaction{}).
			Where("tenant_id = ? AND transaction_date >= ? AND transaction_date <= ?", tenantID, start, end),
		filter,
	)

	if err := query.Find(&txs).Error; err != nil {
		return nil, err
	}
	return txs, nil
}

// FindByType finds transactions by type
func (r *GormInventoryTransactionRepository) FindByType(ctx context.Context, tenantID uuid.UUID, txType inventory.TransactionType, filter shared.Filter) ([]inventory.InventoryTransaction, error) {
	var txs []inventory.InventoryTransaction
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&inventory.InventoryTransaction{}).
			Where("tenant_id = ? AND transaction_type = ?", tenantID, txType),
		filter,
	)

	if err := query.Find(&txs).Error; err != nil {
		return nil, err
	}
	return txs, nil
}

// FindForTenant finds all transactions for a tenant
func (r *GormInventoryTransactionRepository) FindForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]inventory.InventoryTransaction, error) {
	var txs []inventory.InventoryTransaction
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&inventory.InventoryTransaction{}).
			Where("tenant_id = ?", tenantID),
		filter,
	)

	if err := query.Find(&txs).Error; err != nil {
		return nil, err
	}
	return txs, nil
}

// Create creates a new transaction (append-only, no update allowed)
func (r *GormInventoryTransactionRepository) Create(ctx context.Context, tx *inventory.InventoryTransaction) error {
	return r.db.WithContext(ctx).Create(tx).Error
}

// CreateBatch creates multiple transactions
func (r *GormInventoryTransactionRepository) CreateBatch(ctx context.Context, txs []*inventory.InventoryTransaction) error {
	if len(txs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&txs).Error
}

// CountForTenant counts transactions for a tenant
func (r *GormInventoryTransactionRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&inventory.InventoryTransaction{}).Where("tenant_id = ?", tenantID)
	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByInventoryItem counts transactions for an inventory item
func (r *GormInventoryTransactionRepository) CountByInventoryItem(ctx context.Context, inventoryItemID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&inventory.InventoryTransaction{}).
		Where("inventory_item_id = ?", inventoryItemID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// SumQuantityByTypeAndDateRange sums quantities for analysis
func (r *GormInventoryTransactionRepository) SumQuantityByTypeAndDateRange(ctx context.Context, tenantID uuid.UUID, txType inventory.TransactionType, start, end time.Time) (decimal.Decimal, error) {
	var result struct {
		Total decimal.Decimal
	}
	if err := r.db.WithContext(ctx).
		Model(&inventory.InventoryTransaction{}).
		Select("COALESCE(SUM(quantity), 0) as total").
		Where("tenant_id = ? AND transaction_type = ? AND transaction_date >= ? AND transaction_date <= ?",
			tenantID, txType, start, end).
		Scan(&result).Error; err != nil {
		return decimal.Zero, err
	}
	return result.Total, nil
}

// applyFilter applies filter options to the query
func (r *GormInventoryTransactionRepository) applyFilter(query *gorm.DB, filter shared.Filter) *gorm.DB {
	query = r.applyFilterWithoutPagination(query, filter)

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
		query = query.Order("transaction_date DESC")
	}

	return query
}

// applyFilterWithoutPagination applies filter options without pagination
func (r *GormInventoryTransactionRepository) applyFilterWithoutPagination(query *gorm.DB, filter shared.Filter) *gorm.DB {
	for key, value := range filter.Filters {
		switch key {
		case "warehouse_id":
			query = query.Where("warehouse_id = ?", value)
		case "product_id":
			query = query.Where("product_id = ?", value)
		case "transaction_type":
			query = query.Where("transaction_type = ?", value)
		case "source_type":
			query = query.Where("source_type = ?", value)
		case "source_id":
			query = query.Where("source_id = ?", value)
		case "start_date":
			query = query.Where("transaction_date >= ?", value)
		case "end_date":
			query = query.Where("transaction_date <= ?", value)
		case "operator_id":
			query = query.Where("operator_id = ?", value)
		case "batch_id":
			query = query.Where("batch_id = ?", value)
		case "lock_id":
			query = query.Where("lock_id = ?", value)
		}
	}

	return query
}

// Ensure GormInventoryTransactionRepository implements InventoryTransactionRepository
var _ inventory.InventoryTransactionRepository = (*GormInventoryTransactionRepository)(nil)
