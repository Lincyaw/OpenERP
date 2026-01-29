package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/erp/backend/internal/infrastructure/persistence/datascope"
	"github.com/erp/backend/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// GormPurchaseOrderRepository implements PurchaseOrderRepository using GORM
type GormPurchaseOrderRepository struct {
	db          *gorm.DB
	outboxSaver shared.OutboxEventSaver // optional, for transactional outbox pattern
}

// NewGormPurchaseOrderRepository creates a new GormPurchaseOrderRepository
func NewGormPurchaseOrderRepository(db *gorm.DB) *GormPurchaseOrderRepository {
	return &GormPurchaseOrderRepository{db: db}
}

// SetOutboxEventSaver sets the outbox event saver for transactional event publishing
func (r *GormPurchaseOrderRepository) SetOutboxEventSaver(saver shared.OutboxEventSaver) {
	r.outboxSaver = saver
}

// FindByID finds a purchase order by its ID
func (r *GormPurchaseOrderRepository) FindByID(ctx context.Context, id uuid.UUID) (*trade.PurchaseOrder, error) {
	var model models.PurchaseOrderModel
	if err := r.db.WithContext(ctx).
		Preload("Items").
		First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByIDForTenant finds a purchase order by ID within a tenant
func (r *GormPurchaseOrderRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*trade.PurchaseOrder, error) {
	var model models.PurchaseOrderModel
	if err := r.db.WithContext(ctx).
		Preload("Items").
		Where("tenant_id = ? AND id = ?", tenantID, id).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByOrderNumber finds a purchase order by order number for a tenant
func (r *GormPurchaseOrderRepository) FindByOrderNumber(ctx context.Context, tenantID uuid.UUID, orderNumber string) (*trade.PurchaseOrder, error) {
	var model models.PurchaseOrderModel
	if err := r.db.WithContext(ctx).
		Preload("Items").
		Where("tenant_id = ? AND order_number = ?", tenantID, orderNumber).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindAllForTenant finds all purchase orders for a tenant with filtering and data scope
func (r *GormPurchaseOrderRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]trade.PurchaseOrder, error) {
	var orderModels []models.PurchaseOrderModel

	// Start with tenant filter
	query := r.db.WithContext(ctx).Model(&models.PurchaseOrderModel{}).Where("tenant_id = ?", tenantID)

	// Apply data scope filtering from context
	dsFilter := datascope.NewFilterFromContext(ctx)
	query = dsFilter.Apply(query, "purchase_order")

	// Apply additional filters
	query = r.applyFilter(query, filter)

	// Preload Items to calculate item_count
	if err := query.Preload("Items").Find(&orderModels).Error; err != nil {
		return nil, err
	}
	orders := make([]trade.PurchaseOrder, len(orderModels))
	for i, model := range orderModels {
		orders[i] = *model.ToDomain()
	}
	return orders, nil
}

// FindBySupplier finds purchase orders for a supplier with data scope filtering
func (r *GormPurchaseOrderRepository) FindBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID, filter shared.Filter) ([]trade.PurchaseOrder, error) {
	var orderModels []models.PurchaseOrderModel

	query := r.db.WithContext(ctx).Model(&models.PurchaseOrderModel{}).
		Where("tenant_id = ? AND supplier_id = ?", tenantID, supplierID)

	// Apply data scope filtering
	dsFilter := datascope.NewFilterFromContext(ctx)
	query = dsFilter.Apply(query, "purchase_order")

	query = r.applyFilter(query, filter)

	if err := query.Find(&orderModels).Error; err != nil {
		return nil, err
	}
	orders := make([]trade.PurchaseOrder, len(orderModels))
	for i, model := range orderModels {
		orders[i] = *model.ToDomain()
	}
	return orders, nil
}

// FindByStatus finds purchase orders by status for a tenant with data scope filtering
func (r *GormPurchaseOrderRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status trade.PurchaseOrderStatus, filter shared.Filter) ([]trade.PurchaseOrder, error) {
	var orderModels []models.PurchaseOrderModel

	query := r.db.WithContext(ctx).Model(&models.PurchaseOrderModel{}).
		Where("tenant_id = ? AND status = ?", tenantID, status)

	// Apply data scope filtering
	dsFilter := datascope.NewFilterFromContext(ctx)
	query = dsFilter.Apply(query, "purchase_order")

	query = r.applyFilter(query, filter)

	if err := query.Find(&orderModels).Error; err != nil {
		return nil, err
	}
	orders := make([]trade.PurchaseOrder, len(orderModels))
	for i, model := range orderModels {
		orders[i] = *model.ToDomain()
	}
	return orders, nil
}

// FindByWarehouse finds purchase orders for a warehouse with data scope filtering
func (r *GormPurchaseOrderRepository) FindByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID, filter shared.Filter) ([]trade.PurchaseOrder, error) {
	var orderModels []models.PurchaseOrderModel

	query := r.db.WithContext(ctx).Model(&models.PurchaseOrderModel{}).
		Where("tenant_id = ? AND warehouse_id = ?", tenantID, warehouseID)

	// Apply data scope filtering
	dsFilter := datascope.NewFilterFromContext(ctx)
	query = dsFilter.Apply(query, "purchase_order")

	query = r.applyFilter(query, filter)

	if err := query.Find(&orderModels).Error; err != nil {
		return nil, err
	}
	orders := make([]trade.PurchaseOrder, len(orderModels))
	for i, model := range orderModels {
		orders[i] = *model.ToDomain()
	}
	return orders, nil
}

// FindPendingReceipt finds purchase orders pending receipt with data scope filtering
func (r *GormPurchaseOrderRepository) FindPendingReceipt(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]trade.PurchaseOrder, error) {
	var orderModels []models.PurchaseOrderModel

	query := r.db.WithContext(ctx).Model(&models.PurchaseOrderModel{}).
		Where("tenant_id = ? AND status IN ?", tenantID, []trade.PurchaseOrderStatus{
			trade.PurchaseOrderStatusConfirmed,
			trade.PurchaseOrderStatusPartialReceived,
		})

	// Apply data scope filtering
	dsFilter := datascope.NewFilterFromContext(ctx)
	query = dsFilter.Apply(query, "purchase_order")

	query = r.applyFilter(query, filter)

	if err := query.Find(&orderModels).Error; err != nil {
		return nil, err
	}
	orders := make([]trade.PurchaseOrder, len(orderModels))
	for i, model := range orderModels {
		orders[i] = *model.ToDomain()
	}
	return orders, nil
}

// Save creates or updates a purchase order
func (r *GormPurchaseOrderRepository) Save(ctx context.Context, order *trade.PurchaseOrder) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Convert to persistence model
		model := models.PurchaseOrderModelFromDomain(order)

		// Save the order without auto-saving associations
		if err := tx.Omit("Items").Save(model).Error; err != nil {
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
					Delete(&models.PurchaseOrderItemModel{}).Error; err != nil {
					return err
				}
			} else {
				// Delete all items if no items remain
				if err := tx.Where("order_id = ?", order.ID).
					Delete(&models.PurchaseOrderItemModel{}).Error; err != nil {
					return err
				}
			}

			// Save/update remaining items
			for i := range order.Items {
				order.Items[i].OrderID = order.ID
				itemModel := models.PurchaseOrderItemModelFromDomain(&order.Items[i])
				if err := tx.Save(itemModel).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// SaveWithLock saves with optimistic locking (version check)
func (r *GormPurchaseOrderRepository) SaveWithLock(ctx context.Context, order *trade.PurchaseOrder) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get current version from database
		var currentVersion int
		if err := tx.Model(&models.PurchaseOrderModel{}).
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
		result := tx.Model(&models.PurchaseOrderModel{}).
			Where("id = ? AND version = ?", order.ID, currentVersion).
			Updates(map[string]interface{}{
				"supplier_id":     order.SupplierID,
				"supplier_name":   order.SupplierName,
				"warehouse_id":    order.WarehouseID,
				"total_amount":    order.TotalAmount,
				"discount_amount": order.DiscountAmount,
				"payable_amount":  order.PayableAmount,
				"status":          order.Status,
				"remark":          order.Remark,
				"confirmed_at":    order.ConfirmedAt,
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
				Delete(&models.PurchaseOrderItemModel{}).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Where("order_id = ?", order.ID).
				Delete(&models.PurchaseOrderItemModel{}).Error; err != nil {
				return err
			}
		}

		// Save/update remaining items
		for i := range order.Items {
			order.Items[i].OrderID = order.ID
			itemModel := models.PurchaseOrderItemModelFromDomain(&order.Items[i])
			if err := tx.Save(itemModel).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// SaveWithLockAndEvents saves with optimistic locking and persists domain events atomically
// This implements the transactional outbox pattern - events are saved to the outbox table
// in the same transaction as the aggregate, ensuring guaranteed event delivery
func (r *GormPurchaseOrderRepository) SaveWithLockAndEvents(ctx context.Context, order *trade.PurchaseOrder, events []shared.DomainEvent) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get current version from database
		var currentVersion int
		if err := tx.Model(&models.PurchaseOrderModel{}).
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
		result := tx.Model(&models.PurchaseOrderModel{}).
			Where("id = ? AND version = ?", order.ID, currentVersion).
			Updates(map[string]interface{}{
				"supplier_id":     order.SupplierID,
				"supplier_name":   order.SupplierName,
				"warehouse_id":    order.WarehouseID,
				"total_amount":    order.TotalAmount,
				"discount_amount": order.DiscountAmount,
				"payable_amount":  order.PayableAmount,
				"status":          order.Status,
				"remark":          order.Remark,
				"confirmed_at":    order.ConfirmedAt,
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
				Delete(&models.PurchaseOrderItemModel{}).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Where("order_id = ?", order.ID).
				Delete(&models.PurchaseOrderItemModel{}).Error; err != nil {
				return err
			}
		}

		// Save/update remaining items
		for i := range order.Items {
			order.Items[i].OrderID = order.ID
			itemModel := models.PurchaseOrderItemModelFromDomain(&order.Items[i])
			if err := tx.Save(itemModel).Error; err != nil {
				return err
			}
		}

		// Save events to outbox within the same transaction
		if r.outboxSaver != nil && len(events) > 0 {
			if err := r.outboxSaver.SaveEvents(ctx, tx, events...); err != nil {
				return fmt.Errorf("failed to save events to outbox: %w", err)
			}
		}

		return nil
	})
}

// Delete deletes a purchase order (soft delete)
func (r *GormPurchaseOrderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete items first
		if err := tx.Where("order_id = ?", id).Delete(&models.PurchaseOrderItemModel{}).Error; err != nil {
			return err
		}

		// Delete order
		result := tx.Delete(&models.PurchaseOrderModel{}, "id = ?", id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return shared.ErrNotFound
		}
		return nil
	})
}

// DeleteForTenant deletes a purchase order for a tenant
func (r *GormPurchaseOrderRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Find the order first
		var model models.PurchaseOrderModel
		if err := tx.Where("tenant_id = ? AND id = ?", tenantID, id).First(&model).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return shared.ErrNotFound
			}
			return err
		}

		// Delete items
		if err := tx.Where("order_id = ?", id).Delete(&models.PurchaseOrderItemModel{}).Error; err != nil {
			return err
		}

		// Delete order
		result := tx.Delete(&models.PurchaseOrderModel{}, "tenant_id = ? AND id = ?", tenantID, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return shared.ErrNotFound
		}
		return nil
	})
}

// CountForTenant counts purchase orders for a tenant with optional filters and data scope
func (r *GormPurchaseOrderRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.PurchaseOrderModel{}).Where("tenant_id = ?", tenantID)

	// Apply data scope filtering
	dsFilter := datascope.NewFilterFromContext(ctx)
	query = dsFilter.Apply(query, "purchase_order")

	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByStatus counts purchase orders by status for a tenant with data scope
func (r *GormPurchaseOrderRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status trade.PurchaseOrderStatus) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.PurchaseOrderModel{}).
		Where("tenant_id = ? AND status = ?", tenantID, status)

	// Apply data scope filtering
	dsFilter := datascope.NewFilterFromContext(ctx)
	query = dsFilter.Apply(query, "purchase_order")

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountBySupplier counts purchase orders for a supplier with data scope
func (r *GormPurchaseOrderRepository) CountBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.PurchaseOrderModel{}).
		Where("tenant_id = ? AND supplier_id = ?", tenantID, supplierID)

	// Apply data scope filtering
	dsFilter := datascope.NewFilterFromContext(ctx)
	query = dsFilter.Apply(query, "purchase_order")

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountIncompleteBySupplier counts incomplete (not COMPLETED or CANCELLED) orders for a supplier
// Used for validation before supplier deletion
func (r *GormPurchaseOrderRepository) CountIncompleteBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.PurchaseOrderModel{}).
		Where("tenant_id = ? AND supplier_id = ? AND status NOT IN ?", tenantID, supplierID,
			[]trade.PurchaseOrderStatus{trade.PurchaseOrderStatusCompleted, trade.PurchaseOrderStatusCancelled}).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountPendingReceipt counts orders pending receipt for a tenant with data scope
func (r *GormPurchaseOrderRepository) CountPendingReceipt(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.PurchaseOrderModel{}).
		Where("tenant_id = ? AND status IN ?", tenantID, []trade.PurchaseOrderStatus{
			trade.PurchaseOrderStatusConfirmed,
			trade.PurchaseOrderStatusPartialReceived,
		})

	// Apply data scope filtering
	dsFilter := datascope.NewFilterFromContext(ctx)
	query = dsFilter.Apply(query, "purchase_order")

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ExistsByOrderNumber checks if an order number exists for a tenant
func (r *GormPurchaseOrderRepository) ExistsByOrderNumber(ctx context.Context, tenantID uuid.UUID, orderNumber string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.PurchaseOrderModel{}).
		Where("tenant_id = ? AND order_number = ?", tenantID, orderNumber).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// GenerateOrderNumber generates a unique order number for a tenant
// Format: PO-YYYY-NNNNN (e.g., PO-2026-00001)
func (r *GormPurchaseOrderRepository) GenerateOrderNumber(ctx context.Context, tenantID uuid.UUID) (string, error) {
	year := time.Now().Year()
	prefix := fmt.Sprintf("PO-%d-", year)

	// Get the highest order number for this year
	var lastOrder models.PurchaseOrderModel
	err := r.db.WithContext(ctx).
		Model(&models.PurchaseOrderModel{}).
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

// ExistsByProduct checks if any purchase order items exist for a product
// Used for validation before product deletion
func (r *GormPurchaseOrderRepository) ExistsByProduct(ctx context.Context, tenantID, productID uuid.UUID) (bool, error) {
	var count int64
	// Query purchase_order_items joined with purchase_orders to filter by tenant
	if err := r.db.WithContext(ctx).
		Table("purchase_order_items").
		Joins("JOIN purchase_orders ON purchase_orders.id = purchase_order_items.order_id").
		Where("purchase_orders.tenant_id = ? AND purchase_order_items.product_id = ?", tenantID, productID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// applyFilter applies filter options to the query
func (r *GormPurchaseOrderRepository) applyFilter(query *gorm.DB, filter shared.Filter) *gorm.DB {
	query = r.applyFilterWithoutPagination(query, filter)

	// Apply pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// Apply ordering with whitelist validation to prevent SQL injection
	if filter.OrderBy != "" {
		sortField := ValidateSortField(filter.OrderBy, PurchaseOrderSortFields, "")
		if sortField != "" {
			sortOrder := ValidateSortOrder(filter.OrderDir)
			query = query.Order(sortField + " " + sortOrder)
		} else {
			// Default ordering if invalid field
			query = query.Order("created_at DESC")
		}
	} else {
		// Default ordering
		query = query.Order("created_at DESC")
	}

	return query
}

// applyFilterWithoutPagination applies filter options without pagination
func (r *GormPurchaseOrderRepository) applyFilterWithoutPagination(query *gorm.DB, filter shared.Filter) *gorm.DB {
	// Apply search
	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		query = query.Where("order_number ILIKE ? OR supplier_name ILIKE ?",
			searchPattern, searchPattern)
	}

	// Apply additional filters
	for key, value := range filter.Filters {
		switch key {
		case "supplier_id":
			query = query.Where("supplier_id = ?", value)
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

// Ensure GormPurchaseOrderRepository implements PurchaseOrderRepository
var _ trade.PurchaseOrderRepository = (*GormPurchaseOrderRepository)(nil)
