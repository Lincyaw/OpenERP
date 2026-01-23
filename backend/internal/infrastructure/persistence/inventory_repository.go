package persistence

import (
	"context"
	"errors"
	"strings"

	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GormInventoryItemRepository implements InventoryItemRepository using GORM
type GormInventoryItemRepository struct {
	db *gorm.DB
}

// NewGormInventoryItemRepository creates a new GormInventoryItemRepository
func NewGormInventoryItemRepository(db *gorm.DB) *GormInventoryItemRepository {
	return &GormInventoryItemRepository{db: db}
}

// FindByID finds an inventory item by its ID
func (r *GormInventoryItemRepository) FindByID(ctx context.Context, id uuid.UUID) (*inventory.InventoryItem, error) {
	var item inventory.InventoryItem
	if err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

// FindByIDForTenant finds an inventory item by ID within a tenant
func (r *GormInventoryItemRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*inventory.InventoryItem, error) {
	var item inventory.InventoryItem
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

// FindByWarehouseAndProduct finds inventory by warehouse-product combination
func (r *GormInventoryItemRepository) FindByWarehouseAndProduct(ctx context.Context, tenantID, warehouseID, productID uuid.UUID) (*inventory.InventoryItem, error) {
	var item inventory.InventoryItem
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND warehouse_id = ? AND product_id = ?", tenantID, warehouseID, productID).
		First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

// FindByWarehouse finds all inventory items in a warehouse
func (r *GormInventoryItemRepository) FindByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID, filter shared.Filter) ([]inventory.InventoryItem, error) {
	var items []inventory.InventoryItem
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&inventory.InventoryItem{}).
			Where("tenant_id = ? AND warehouse_id = ?", tenantID, warehouseID),
		filter,
	)

	if err := query.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// FindByProduct finds all inventory items for a product (across warehouses)
func (r *GormInventoryItemRepository) FindByProduct(ctx context.Context, tenantID, productID uuid.UUID, filter shared.Filter) ([]inventory.InventoryItem, error) {
	var items []inventory.InventoryItem
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&inventory.InventoryItem{}).
			Where("tenant_id = ? AND product_id = ?", tenantID, productID),
		filter,
	)

	if err := query.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// FindAllForTenant finds all inventory items for a tenant
func (r *GormInventoryItemRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]inventory.InventoryItem, error) {
	var items []inventory.InventoryItem
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&inventory.InventoryItem{}).
			Where("tenant_id = ?", tenantID),
		filter,
	)

	if err := query.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// FindBelowMinimum finds items below their minimum threshold
func (r *GormInventoryItemRepository) FindBelowMinimum(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]inventory.InventoryItem, error) {
	var items []inventory.InventoryItem
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&inventory.InventoryItem{}).
			Where("tenant_id = ? AND min_quantity > 0 AND (available_quantity + locked_quantity) < min_quantity", tenantID),
		filter,
	)

	if err := query.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// FindWithAvailableStock finds items that have available stock
func (r *GormInventoryItemRepository) FindWithAvailableStock(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]inventory.InventoryItem, error) {
	var items []inventory.InventoryItem
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&inventory.InventoryItem{}).
			Where("tenant_id = ? AND available_quantity > 0", tenantID),
		filter,
	)

	if err := query.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// FindByIDs finds multiple inventory items by their IDs
func (r *GormInventoryItemRepository) FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]inventory.InventoryItem, error) {
	if len(ids) == 0 {
		return []inventory.InventoryItem{}, nil
	}

	var items []inventory.InventoryItem
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id IN ?", tenantID, ids).
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// Save creates or updates an inventory item
func (r *GormInventoryItemRepository) Save(ctx context.Context, item *inventory.InventoryItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}

// SaveWithLock saves with optimistic locking (checks version)
func (r *GormInventoryItemRepository) SaveWithLock(ctx context.Context, item *inventory.InventoryItem) error {
	result := r.db.WithContext(ctx).
		Model(item).
		Where("id = ? AND version = ?", item.ID, item.Version-1).
		Updates(map[string]interface{}{
			"available_quantity": item.AvailableQuantity,
			"locked_quantity":    item.LockedQuantity,
			"unit_cost":          item.UnitCost,
			"min_quantity":       item.MinQuantity,
			"max_quantity":       item.MaxQuantity,
			"version":            item.Version,
			"updated_at":         item.UpdatedAt,
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.NewDomainError("OPTIMISTIC_LOCK_FAILED", "Inventory item was modified by another transaction")
	}
	return nil
}

// Delete deletes an inventory item
func (r *GormInventoryItemRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&inventory.InventoryItem{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// DeleteForTenant deletes an inventory item within a tenant
func (r *GormInventoryItemRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&inventory.InventoryItem{}, "tenant_id = ? AND id = ?", tenantID, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// CountForTenant counts inventory items for a tenant
func (r *GormInventoryItemRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&inventory.InventoryItem{}).Where("tenant_id = ?", tenantID)
	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByWarehouse counts inventory items in a warehouse
func (r *GormInventoryItemRepository) CountByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&inventory.InventoryItem{}).
		Where("tenant_id = ? AND warehouse_id = ?", tenantID, warehouseID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByProduct counts inventory items for a product
func (r *GormInventoryItemRepository) CountByProduct(ctx context.Context, tenantID, productID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&inventory.InventoryItem{}).
		Where("tenant_id = ? AND product_id = ?", tenantID, productID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// SumQuantityByProduct sums total quantity for a product across all warehouses
func (r *GormInventoryItemRepository) SumQuantityByProduct(ctx context.Context, tenantID, productID uuid.UUID) (decimal.Decimal, error) {
	var result struct {
		Total decimal.Decimal
	}
	if err := r.db.WithContext(ctx).
		Model(&inventory.InventoryItem{}).
		Select("COALESCE(SUM(available_quantity + locked_quantity), 0) as total").
		Where("tenant_id = ? AND product_id = ?", tenantID, productID).
		Scan(&result).Error; err != nil {
		return decimal.Zero, err
	}
	return result.Total, nil
}

// SumValueByWarehouse sums total inventory value in a warehouse
func (r *GormInventoryItemRepository) SumValueByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID) (decimal.Decimal, error) {
	var result struct {
		Total decimal.Decimal
	}
	if err := r.db.WithContext(ctx).
		Model(&inventory.InventoryItem{}).
		Select("COALESCE(SUM((available_quantity + locked_quantity) * unit_cost), 0) as total").
		Where("tenant_id = ? AND warehouse_id = ?", tenantID, warehouseID).
		Scan(&result).Error; err != nil {
		return decimal.Zero, err
	}
	return result.Total, nil
}

// ExistsByWarehouseAndProduct checks if inventory exists for warehouse-product
func (r *GormInventoryItemRepository) ExistsByWarehouseAndProduct(ctx context.Context, tenantID, warehouseID, productID uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&inventory.InventoryItem{}).
		Where("tenant_id = ? AND warehouse_id = ? AND product_id = ?", tenantID, warehouseID, productID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetOrCreate gets existing inventory item or creates a new one
func (r *GormInventoryItemRepository) GetOrCreate(ctx context.Context, tenantID, warehouseID, productID uuid.UUID) (*inventory.InventoryItem, error) {
	// Try to find existing
	item, err := r.FindByWarehouseAndProduct(ctx, tenantID, warehouseID, productID)
	if err == nil {
		return item, nil
	}
	if !errors.Is(err, shared.ErrNotFound) {
		return nil, err
	}

	// Create new inventory item
	item, err = inventory.NewInventoryItem(tenantID, warehouseID, productID)
	if err != nil {
		return nil, err
	}

	// Use ON CONFLICT to handle race conditions
	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "warehouse_id"}, {Name: "product_id"}},
			DoNothing: true,
		}).
		Create(item).Error; err != nil {
		return nil, err
	}

	// If the row wasn't created (conflict), fetch the existing one
	if item.ID == uuid.Nil {
		return r.FindByWarehouseAndProduct(ctx, tenantID, warehouseID, productID)
	}

	return item, nil
}

// applyFilter applies filter options to the query
func (r *GormInventoryItemRepository) applyFilter(query *gorm.DB, filter shared.Filter) *gorm.DB {
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
		query = query.Order("created_at DESC")
	}

	return query
}

// applyFilterWithoutPagination applies filter options without pagination
func (r *GormInventoryItemRepository) applyFilterWithoutPagination(query *gorm.DB, filter shared.Filter) *gorm.DB {
	for key, value := range filter.Filters {
		switch key {
		case "warehouse_id":
			query = query.Where("warehouse_id = ?", value)
		case "product_id":
			query = query.Where("product_id = ?", value)
		case "below_minimum":
			if value == true {
				query = query.Where("min_quantity > 0 AND (available_quantity + locked_quantity) < min_quantity")
			}
		case "above_maximum":
			if value == true {
				query = query.Where("max_quantity > 0 AND (available_quantity + locked_quantity) > max_quantity")
			}
		case "has_stock":
			if value == true {
				query = query.Where("available_quantity > 0")
			}
		case "no_stock":
			if value == true {
				query = query.Where("available_quantity = 0 AND locked_quantity = 0")
			}
		case "min_quantity":
			query = query.Where("(available_quantity + locked_quantity) >= ?", value)
		case "max_quantity":
			query = query.Where("(available_quantity + locked_quantity) <= ?", value)
		}
	}

	return query
}

// Ensure GormInventoryItemRepository implements InventoryItemRepository
var _ inventory.InventoryItemRepository = (*GormInventoryItemRepository)(nil)
