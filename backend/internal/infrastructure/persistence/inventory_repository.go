package persistence

import (
	"context"
	"errors"
	"strings"

	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/persistence/datascope"
	"github.com/erp/backend/internal/infrastructure/persistence/models"
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

// WithTx returns a new repository instance with the given transaction
func (r *GormInventoryItemRepository) WithTx(tx *gorm.DB) *GormInventoryItemRepository {
	return &GormInventoryItemRepository{db: tx}
}

// FindByID finds an inventory item by its ID
func (r *GormInventoryItemRepository) FindByID(ctx context.Context, id uuid.UUID) (*inventory.InventoryItem, error) {
	var model models.InventoryItemModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByIDForTenant finds an inventory item by ID within a tenant
func (r *GormInventoryItemRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*inventory.InventoryItem, error) {
	var model models.InventoryItemModel
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

// FindByWarehouseAndProduct finds inventory by warehouse-product combination
func (r *GormInventoryItemRepository) FindByWarehouseAndProduct(ctx context.Context, tenantID, warehouseID, productID uuid.UUID) (*inventory.InventoryItem, error) {
	var model models.InventoryItemModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND warehouse_id = ? AND product_id = ?", tenantID, warehouseID, productID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByWarehouse finds all inventory items in a warehouse with data scope filtering
func (r *GormInventoryItemRepository) FindByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID, filter shared.Filter) ([]inventory.InventoryItem, error) {
	var itemModels []models.InventoryItemModel

	query := r.db.WithContext(ctx).Model(&models.InventoryItemModel{}).
		Where("tenant_id = ? AND warehouse_id = ?", tenantID, warehouseID)

	// Apply data scope filtering
	dsFilter := datascope.NewFilterFromContext(ctx)
	query = dsFilter.Apply(query, "inventory")

	query = r.applyFilter(query, filter)

	if err := query.Find(&itemModels).Error; err != nil {
		return nil, err
	}
	items := make([]inventory.InventoryItem, len(itemModels))
	for i, model := range itemModels {
		items[i] = *model.ToDomain()
	}
	return items, nil
}

// FindByProduct finds all inventory items for a product (across warehouses) with data scope filtering
func (r *GormInventoryItemRepository) FindByProduct(ctx context.Context, tenantID, productID uuid.UUID, filter shared.Filter) ([]inventory.InventoryItem, error) {
	var itemModels []models.InventoryItemModel

	query := r.db.WithContext(ctx).Model(&models.InventoryItemModel{}).
		Where("tenant_id = ? AND product_id = ?", tenantID, productID)

	// Apply data scope filtering - warehouse users only see their warehouse's inventory
	dsFilter := datascope.NewFilterFromContext(ctx)
	query = dsFilter.Apply(query, "inventory")

	query = r.applyFilter(query, filter)

	if err := query.Find(&itemModels).Error; err != nil {
		return nil, err
	}
	items := make([]inventory.InventoryItem, len(itemModels))
	for i, model := range itemModels {
		items[i] = *model.ToDomain()
	}
	return items, nil
}

// FindAllForTenant finds all inventory items for a tenant with data scope filtering
func (r *GormInventoryItemRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]inventory.InventoryItem, error) {
	var itemModels []models.InventoryItemModel

	query := r.db.WithContext(ctx).Model(&models.InventoryItemModel{}).
		Where("tenant_id = ?", tenantID)

	// Apply data scope filtering - warehouse users only see their warehouse's inventory
	dsFilter := datascope.NewFilterFromContext(ctx)
	query = dsFilter.Apply(query, "inventory")

	query = r.applyFilter(query, filter)

	if err := query.Find(&itemModels).Error; err != nil {
		return nil, err
	}
	items := make([]inventory.InventoryItem, len(itemModels))
	for i, model := range itemModels {
		items[i] = *model.ToDomain()
	}
	return items, nil
}

// FindBelowMinimum finds items below their minimum threshold with data scope filtering
func (r *GormInventoryItemRepository) FindBelowMinimum(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]inventory.InventoryItem, error) {
	var itemModels []models.InventoryItemModel

	query := r.db.WithContext(ctx).Model(&models.InventoryItemModel{}).
		Where("tenant_id = ? AND min_quantity > 0 AND (available_quantity + locked_quantity) < min_quantity", tenantID)

	// Apply data scope filtering
	dsFilter := datascope.NewFilterFromContext(ctx)
	query = dsFilter.Apply(query, "inventory")

	query = r.applyFilter(query, filter)

	if err := query.Find(&itemModels).Error; err != nil {
		return nil, err
	}
	items := make([]inventory.InventoryItem, len(itemModels))
	for i, model := range itemModels {
		items[i] = *model.ToDomain()
	}
	return items, nil
}

// FindWithAvailableStock finds items that have available stock with data scope filtering
func (r *GormInventoryItemRepository) FindWithAvailableStock(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]inventory.InventoryItem, error) {
	var itemModels []models.InventoryItemModel

	query := r.db.WithContext(ctx).Model(&models.InventoryItemModel{}).
		Where("tenant_id = ? AND available_quantity > 0", tenantID)

	// Apply data scope filtering
	dsFilter := datascope.NewFilterFromContext(ctx)
	query = dsFilter.Apply(query, "inventory")

	query = r.applyFilter(query, filter)

	if err := query.Find(&itemModels).Error; err != nil {
		return nil, err
	}
	items := make([]inventory.InventoryItem, len(itemModels))
	for i, model := range itemModels {
		items[i] = *model.ToDomain()
	}
	return items, nil
}

// FindByIDs finds multiple inventory items by their IDs
func (r *GormInventoryItemRepository) FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]inventory.InventoryItem, error) {
	if len(ids) == 0 {
		return []inventory.InventoryItem{}, nil
	}

	var itemModels []models.InventoryItemModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id IN ?", tenantID, ids).
		Find(&itemModels).Error; err != nil {
		return nil, err
	}
	items := make([]inventory.InventoryItem, len(itemModels))
	for i, model := range itemModels {
		items[i] = *model.ToDomain()
	}
	return items, nil
}

// Save creates or updates an inventory item
func (r *GormInventoryItemRepository) Save(ctx context.Context, item *inventory.InventoryItem) error {
	model := models.InventoryItemModelFromDomain(item)
	return r.db.WithContext(ctx).Save(model).Error
}

// SaveWithLock saves with optimistic locking (checks version)
func (r *GormInventoryItemRepository) SaveWithLock(ctx context.Context, item *inventory.InventoryItem) error {
	model := models.InventoryItemModelFromDomain(item)
	result := r.db.WithContext(ctx).
		Model(model).
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
	result := r.db.WithContext(ctx).Delete(&models.InventoryItemModel{}, "id = ?", id)
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
	result := r.db.WithContext(ctx).Delete(&models.InventoryItemModel{}, "tenant_id = ? AND id = ?", tenantID, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// CountForTenant counts inventory items for a tenant with data scope filtering
func (r *GormInventoryItemRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.InventoryItemModel{}).Where("tenant_id = ?", tenantID)

	// Apply data scope filtering
	dsFilter := datascope.NewFilterFromContext(ctx)
	query = dsFilter.Apply(query, "inventory")

	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByWarehouse counts inventory items in a warehouse with data scope
func (r *GormInventoryItemRepository) CountByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.InventoryItemModel{}).
		Where("tenant_id = ? AND warehouse_id = ?", tenantID, warehouseID)

	// Apply data scope filtering
	dsFilter := datascope.NewFilterFromContext(ctx)
	query = dsFilter.Apply(query, "inventory")

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByProduct counts inventory items for a product with data scope
func (r *GormInventoryItemRepository) CountByProduct(ctx context.Context, tenantID, productID uuid.UUID) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.InventoryItemModel{}).
		Where("tenant_id = ? AND product_id = ?", tenantID, productID)

	// Apply data scope filtering
	dsFilter := datascope.NewFilterFromContext(ctx)
	query = dsFilter.Apply(query, "inventory")

	if err := query.Count(&count).Error; err != nil {
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
		Model(&models.InventoryItemModel{}).
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
		Model(&models.InventoryItemModel{}).
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
		Model(&models.InventoryItemModel{}).
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
	model := models.InventoryItemModelFromDomain(item)
	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "warehouse_id"}, {Name: "product_id"}},
			DoNothing: true,
		}).
		Create(model).Error; err != nil {
		return nil, err
	}

	// If the row wasn't created (conflict), fetch the existing one
	if model.ID == uuid.Nil {
		return r.FindByWarehouseAndProduct(ctx, tenantID, warehouseID, productID)
	}

	return model.ToDomain(), nil
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
