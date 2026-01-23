package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// GormSalesOrderRepository implements SalesOrderRepository using GORM
type GormSalesOrderRepository struct {
	db *gorm.DB
}

// NewGormSalesOrderRepository creates a new GormSalesOrderRepository
func NewGormSalesOrderRepository(db *gorm.DB) *GormSalesOrderRepository {
	return &GormSalesOrderRepository{db: db}
}

// FindByID finds a sales order by its ID
func (r *GormSalesOrderRepository) FindByID(ctx context.Context, id uuid.UUID) (*trade.SalesOrder, error) {
	var order trade.SalesOrder
	if err := r.db.WithContext(ctx).
		Preload("Items").
		First(&order, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &order, nil
}

// FindByIDForTenant finds a sales order by ID within a tenant
func (r *GormSalesOrderRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*trade.SalesOrder, error) {
	var order trade.SalesOrder
	if err := r.db.WithContext(ctx).
		Preload("Items").
		Where("tenant_id = ? AND id = ?", tenantID, id).
		First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &order, nil
}

// FindByOrderNumber finds a sales order by order number for a tenant
func (r *GormSalesOrderRepository) FindByOrderNumber(ctx context.Context, tenantID uuid.UUID, orderNumber string) (*trade.SalesOrder, error) {
	var order trade.SalesOrder
	if err := r.db.WithContext(ctx).
		Preload("Items").
		Where("tenant_id = ? AND order_number = ?", tenantID, orderNumber).
		First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &order, nil
}

// FindAllForTenant finds all sales orders for a tenant with filtering
func (r *GormSalesOrderRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]trade.SalesOrder, error) {
	var orders []trade.SalesOrder
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&trade.SalesOrder{}).Where("tenant_id = ?", tenantID),
		filter,
	)

	if err := query.Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

// FindByCustomer finds sales orders for a customer
func (r *GormSalesOrderRepository) FindByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, filter shared.Filter) ([]trade.SalesOrder, error) {
	var orders []trade.SalesOrder
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&trade.SalesOrder{}).
			Where("tenant_id = ? AND customer_id = ?", tenantID, customerID),
		filter,
	)

	if err := query.Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

// FindByStatus finds sales orders by status for a tenant
func (r *GormSalesOrderRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status trade.OrderStatus, filter shared.Filter) ([]trade.SalesOrder, error) {
	var orders []trade.SalesOrder
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&trade.SalesOrder{}).
			Where("tenant_id = ? AND status = ?", tenantID, status),
		filter,
	)

	if err := query.Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

// FindByWarehouse finds sales orders for a warehouse
func (r *GormSalesOrderRepository) FindByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID, filter shared.Filter) ([]trade.SalesOrder, error) {
	var orders []trade.SalesOrder
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&trade.SalesOrder{}).
			Where("tenant_id = ? AND warehouse_id = ?", tenantID, warehouseID),
		filter,
	)

	if err := query.Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

// Save creates or updates a sales order
func (r *GormSalesOrderRepository) Save(ctx context.Context, order *trade.SalesOrder) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Save the order
		if err := tx.Save(order).Error; err != nil {
			return err
		}

		// Handle items: delete removed items and save/update existing ones
		if order.ID != uuid.Nil {
			// Get existing item IDs
			currentItemIDs := make([]uuid.UUID, len(order.Items))
			for i, item := range order.Items {
				currentItemIDs[i] = item.ID
			}

			// Delete items not in the current list
			if len(currentItemIDs) > 0 {
				if err := tx.Where("order_id = ? AND id NOT IN ?", order.ID, currentItemIDs).
					Delete(&trade.SalesOrderItem{}).Error; err != nil {
					return err
				}
			} else {
				// Delete all items if no items remain
				if err := tx.Where("order_id = ?", order.ID).
					Delete(&trade.SalesOrderItem{}).Error; err != nil {
					return err
				}
			}

			// Save/update remaining items
			for i := range order.Items {
				order.Items[i].OrderID = order.ID
				if err := tx.Save(&order.Items[i]).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// SaveWithLock saves with optimistic locking (version check)
func (r *GormSalesOrderRepository) SaveWithLock(ctx context.Context, order *trade.SalesOrder) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get current version from database
		var currentVersion int
		if err := tx.Model(&trade.SalesOrder{}).
			Where("id = ?", order.ID).
			Select("version").
			Scan(&currentVersion).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return shared.ErrNotFound
			}
			return err
		}

		// Check version matches
		if currentVersion != order.Version {
			return shared.NewDomainError("CONCURRENT_MODIFICATION", "The order has been modified by another user")
		}

		// Increment version
		order.Version++
		order.UpdatedAt = time.Now()

		// Update order with version check
		result := tx.Model(&trade.SalesOrder{}).
			Where("id = ? AND version = ?", order.ID, currentVersion).
			Updates(map[string]interface{}{
				"customer_id":     order.CustomerID,
				"customer_name":   order.CustomerName,
				"warehouse_id":    order.WarehouseID,
				"total_amount":    order.TotalAmount,
				"discount_amount": order.DiscountAmount,
				"payable_amount":  order.PayableAmount,
				"status":          order.Status,
				"remark":          order.Remark,
				"confirmed_at":    order.ConfirmedAt,
				"shipped_at":      order.ShippedAt,
				"completed_at":    order.CompletedAt,
				"cancelled_at":    order.CancelledAt,
				"cancel_reason":   order.CancelReason,
				"version":         order.Version,
				"updated_at":      order.UpdatedAt,
			})

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return shared.NewDomainError("CONCURRENT_MODIFICATION", "The order has been modified by another user")
		}

		// Handle items
		currentItemIDs := make([]uuid.UUID, len(order.Items))
		for i, item := range order.Items {
			currentItemIDs[i] = item.ID
		}

		// Delete items not in the current list
		if len(currentItemIDs) > 0 {
			if err := tx.Where("order_id = ? AND id NOT IN ?", order.ID, currentItemIDs).
				Delete(&trade.SalesOrderItem{}).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Where("order_id = ?", order.ID).
				Delete(&trade.SalesOrderItem{}).Error; err != nil {
				return err
			}
		}

		// Save/update remaining items
		for i := range order.Items {
			order.Items[i].OrderID = order.ID
			if err := tx.Save(&order.Items[i]).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// Delete deletes a sales order (soft delete)
func (r *GormSalesOrderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete items first
		if err := tx.Where("order_id = ?", id).Delete(&trade.SalesOrderItem{}).Error; err != nil {
			return err
		}

		// Delete order
		result := tx.Delete(&trade.SalesOrder{}, "id = ?", id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return shared.ErrNotFound
		}
		return nil
	})
}

// DeleteForTenant deletes a sales order for a tenant
func (r *GormSalesOrderRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Find the order first
		var order trade.SalesOrder
		if err := tx.Where("tenant_id = ? AND id = ?", tenantID, id).First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return shared.ErrNotFound
			}
			return err
		}

		// Delete items
		if err := tx.Where("order_id = ?", id).Delete(&trade.SalesOrderItem{}).Error; err != nil {
			return err
		}

		// Delete order
		result := tx.Delete(&trade.SalesOrder{}, "tenant_id = ? AND id = ?", tenantID, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return shared.ErrNotFound
		}
		return nil
	})
}

// CountForTenant counts sales orders for a tenant with optional filters
func (r *GormSalesOrderRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&trade.SalesOrder{}).Where("tenant_id = ?", tenantID)
	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByStatus counts sales orders by status for a tenant
func (r *GormSalesOrderRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status trade.OrderStatus) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&trade.SalesOrder{}).
		Where("tenant_id = ? AND status = ?", tenantID, status).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByCustomer counts sales orders for a customer
func (r *GormSalesOrderRepository) CountByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&trade.SalesOrder{}).
		Where("tenant_id = ? AND customer_id = ?", tenantID, customerID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ExistsByOrderNumber checks if an order number exists for a tenant
func (r *GormSalesOrderRepository) ExistsByOrderNumber(ctx context.Context, tenantID uuid.UUID, orderNumber string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&trade.SalesOrder{}).
		Where("tenant_id = ? AND order_number = ?", tenantID, orderNumber).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// GenerateOrderNumber generates a unique order number for a tenant
// Format: SO-YYYY-NNNNN (e.g., SO-2026-00001)
func (r *GormSalesOrderRepository) GenerateOrderNumber(ctx context.Context, tenantID uuid.UUID) (string, error) {
	year := time.Now().Year()
	prefix := fmt.Sprintf("SO-%d-", year)

	// Get the highest order number for this year
	var lastOrder trade.SalesOrder
	err := r.db.WithContext(ctx).
		Model(&trade.SalesOrder{}).
		Where("tenant_id = ? AND order_number LIKE ?", tenantID, prefix+"%").
		Order("order_number DESC").
		First(&lastOrder).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	var nextNum int64 = 1
	if err == nil && lastOrder.OrderNumber != "" {
		// Parse the number from the last order number
		parts := strings.Split(lastOrder.OrderNumber, "-")
		if len(parts) == 3 {
			var num int64
			_, parseErr := fmt.Sscanf(parts[2], "%d", &num)
			if parseErr == nil {
				nextNum = num + 1
			}
		}
	}

	// Generate new order number
	orderNumber := fmt.Sprintf("%s%05d", prefix, nextNum)

	// Verify uniqueness
	exists, err := r.ExistsByOrderNumber(ctx, tenantID, orderNumber)
	if err != nil {
		return "", err
	}
	if exists {
		// If exists, try incrementing until we find a unique one
		for i := 0; i < 100; i++ {
			nextNum++
			orderNumber = fmt.Sprintf("%s%05d", prefix, nextNum)
			exists, err = r.ExistsByOrderNumber(ctx, tenantID, orderNumber)
			if err != nil {
				return "", err
			}
			if !exists {
				break
			}
		}
	}

	return orderNumber, nil
}

// applyFilter applies filter options to the query
func (r *GormSalesOrderRepository) applyFilter(query *gorm.DB, filter shared.Filter) *gorm.DB {
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
		// Default ordering
		query = query.Order("created_at DESC")
	}

	return query
}

// applyFilterWithoutPagination applies filter options without pagination
func (r *GormSalesOrderRepository) applyFilterWithoutPagination(query *gorm.DB, filter shared.Filter) *gorm.DB {
	// Apply search
	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		query = query.Where("order_number ILIKE ? OR customer_name ILIKE ?",
			searchPattern, searchPattern)
	}

	// Apply additional filters
	for key, value := range filter.Filters {
		switch key {
		case "customer_id":
			query = query.Where("customer_id = ?", value)
		case "warehouse_id":
			query = query.Where("warehouse_id = ?", value)
		case "status":
			query = query.Where("status = ?", value)
		case "statuses":
			if statuses, ok := value.([]string); ok && len(statuses) > 0 {
				query = query.Where("status IN ?", statuses)
			}
		case "start_date":
			if t, ok := value.(time.Time); ok {
				query = query.Where("created_at >= ?", t)
			}
		case "end_date":
			if t, ok := value.(time.Time); ok {
				query = query.Where("created_at <= ?", t)
			}
		case "min_amount":
			if d, ok := value.(decimal.Decimal); ok {
				query = query.Where("payable_amount >= ?", d)
			}
		case "max_amount":
			if d, ok := value.(decimal.Decimal); ok {
				query = query.Where("payable_amount <= ?", d)
			}
		}
	}

	return query
}

// Ensure GormSalesOrderRepository implements SalesOrderRepository
var _ trade.SalesOrderRepository = (*GormSalesOrderRepository)(nil)
