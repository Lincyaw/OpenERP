package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GormStockBatchRepository implements StockBatchRepository using GORM
type GormStockBatchRepository struct {
	db *gorm.DB
}

// NewGormStockBatchRepository creates a new GormStockBatchRepository
func NewGormStockBatchRepository(db *gorm.DB) *GormStockBatchRepository {
	return &GormStockBatchRepository{db: db}
}

// FindByID finds a stock batch by its ID
func (r *GormStockBatchRepository) FindByID(ctx context.Context, id uuid.UUID) (*inventory.StockBatch, error) {
	var batch inventory.StockBatch
	if err := r.db.WithContext(ctx).First(&batch, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &batch, nil
}

// FindByInventoryItem finds all batches for an inventory item
func (r *GormStockBatchRepository) FindByInventoryItem(ctx context.Context, inventoryItemID uuid.UUID, filter shared.Filter) ([]inventory.StockBatch, error) {
	var batches []inventory.StockBatch
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&inventory.StockBatch{}).
			Where("inventory_item_id = ?", inventoryItemID),
		filter,
	)

	if err := query.Find(&batches).Error; err != nil {
		return nil, err
	}
	return batches, nil
}

// FindAvailable finds available (non-consumed, non-expired) batches
func (r *GormStockBatchRepository) FindAvailable(ctx context.Context, inventoryItemID uuid.UUID) ([]inventory.StockBatch, error) {
	var batches []inventory.StockBatch
	now := time.Now()

	if err := r.db.WithContext(ctx).
		Where("inventory_item_id = ? AND consumed = FALSE AND quantity > 0", inventoryItemID).
		Where("expiry_date IS NULL OR expiry_date > ?", now).
		Order("COALESCE(expiry_date, '9999-12-31') ASC, created_at ASC"). // FEFO (First Expired, First Out)
		Find(&batches).Error; err != nil {
		return nil, err
	}
	return batches, nil
}

// FindExpiringSoon finds batches expiring within a duration
func (r *GormStockBatchRepository) FindExpiringSoon(ctx context.Context, tenantID uuid.UUID, withinDays int, filter shared.Filter) ([]inventory.StockBatch, error) {
	var batches []inventory.StockBatch
	now := time.Now()
	expiryThreshold := now.AddDate(0, 0, withinDays)

	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&inventory.StockBatch{}).
			Joins("JOIN inventory_items ON inventory_items.id = stock_batches.inventory_item_id").
			Where("inventory_items.tenant_id = ?", tenantID).
			Where("stock_batches.consumed = FALSE AND stock_batches.quantity > 0").
			Where("stock_batches.expiry_date IS NOT NULL").
			Where("stock_batches.expiry_date > ? AND stock_batches.expiry_date <= ?", now, expiryThreshold),
		filter,
	)

	if err := query.Find(&batches).Error; err != nil {
		return nil, err
	}
	return batches, nil
}

// FindExpired finds expired batches that still have stock
func (r *GormStockBatchRepository) FindExpired(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]inventory.StockBatch, error) {
	var batches []inventory.StockBatch
	now := time.Now()

	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&inventory.StockBatch{}).
			Joins("JOIN inventory_items ON inventory_items.id = stock_batches.inventory_item_id").
			Where("inventory_items.tenant_id = ?", tenantID).
			Where("stock_batches.consumed = FALSE AND stock_batches.quantity > 0").
			Where("stock_batches.expiry_date IS NOT NULL AND stock_batches.expiry_date <= ?", now),
		filter,
	)

	if err := query.Find(&batches).Error; err != nil {
		return nil, err
	}
	return batches, nil
}

// FindByBatchNumber finds batches by batch number
func (r *GormStockBatchRepository) FindByBatchNumber(ctx context.Context, inventoryItemID uuid.UUID, batchNumber string) (*inventory.StockBatch, error) {
	var batch inventory.StockBatch
	if err := r.db.WithContext(ctx).
		Where("inventory_item_id = ? AND batch_number = ?", inventoryItemID, batchNumber).
		First(&batch).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &batch, nil
}

// Save creates or updates a stock batch
func (r *GormStockBatchRepository) Save(ctx context.Context, batch *inventory.StockBatch) error {
	return r.db.WithContext(ctx).Save(batch).Error
}

// SaveBatch creates or updates multiple stock batches
func (r *GormStockBatchRepository) SaveBatch(ctx context.Context, batches []inventory.StockBatch) error {
	if len(batches) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Save(&batches).Error
}

// Delete deletes a stock batch
func (r *GormStockBatchRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&inventory.StockBatch{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// CountByInventoryItem counts batches for an inventory item
func (r *GormStockBatchRepository) CountByInventoryItem(ctx context.Context, inventoryItemID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&inventory.StockBatch{}).
		Where("inventory_item_id = ?", inventoryItemID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// applyFilter applies filter options to the query
func (r *GormStockBatchRepository) applyFilter(query *gorm.DB, filter shared.Filter) *gorm.DB {
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
		query = query.Order("COALESCE(expiry_date, '9999-12-31') ASC, created_at ASC")
	}

	// Apply additional filters
	for key, value := range filter.Filters {
		switch key {
		case "consumed":
			query = query.Where("consumed = ?", value)
		case "has_stock":
			if value == true {
				query = query.Where("quantity > 0")
			}
		case "expired":
			if value == true {
				query = query.Where("expiry_date IS NOT NULL AND expiry_date <= ?", time.Now())
			}
		}
	}

	return query
}

// Ensure GormStockBatchRepository implements StockBatchRepository
var _ inventory.StockBatchRepository = (*GormStockBatchRepository)(nil)
