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

// GormSalesReturnRepository implements SalesReturnRepository using GORM
type GormSalesReturnRepository struct {
	db *gorm.DB
}

// NewGormSalesReturnRepository creates a new GormSalesReturnRepository
func NewGormSalesReturnRepository(db *gorm.DB) *GormSalesReturnRepository {
	return &GormSalesReturnRepository{db: db}
}

// FindByID finds a sales return by its ID
func (r *GormSalesReturnRepository) FindByID(ctx context.Context, id uuid.UUID) (*trade.SalesReturn, error) {
	var sr trade.SalesReturn
	if err := r.db.WithContext(ctx).
		Preload("Items").
		First(&sr, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &sr, nil
}

// FindByIDForTenant finds a sales return by ID within a tenant
func (r *GormSalesReturnRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*trade.SalesReturn, error) {
	var sr trade.SalesReturn
	if err := r.db.WithContext(ctx).
		Preload("Items").
		Where("tenant_id = ? AND id = ?", tenantID, id).
		First(&sr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &sr, nil
}

// FindByReturnNumber finds a sales return by return number for a tenant
func (r *GormSalesReturnRepository) FindByReturnNumber(ctx context.Context, tenantID uuid.UUID, returnNumber string) (*trade.SalesReturn, error) {
	var sr trade.SalesReturn
	if err := r.db.WithContext(ctx).
		Preload("Items").
		Where("tenant_id = ? AND return_number = ?", tenantID, returnNumber).
		First(&sr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &sr, nil
}

// FindAllForTenant finds all sales returns for a tenant with filtering
func (r *GormSalesReturnRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]trade.SalesReturn, error) {
	var returns []trade.SalesReturn
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&trade.SalesReturn{}).Where("tenant_id = ?", tenantID),
		filter,
	)

	if err := query.Find(&returns).Error; err != nil {
		return nil, err
	}
	return returns, nil
}

// FindByCustomer finds sales returns for a customer
func (r *GormSalesReturnRepository) FindByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, filter shared.Filter) ([]trade.SalesReturn, error) {
	var returns []trade.SalesReturn
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&trade.SalesReturn{}).
			Where("tenant_id = ? AND customer_id = ?", tenantID, customerID),
		filter,
	)

	if err := query.Find(&returns).Error; err != nil {
		return nil, err
	}
	return returns, nil
}

// FindBySalesOrder finds sales returns for a sales order
func (r *GormSalesReturnRepository) FindBySalesOrder(ctx context.Context, tenantID, salesOrderID uuid.UUID) ([]trade.SalesReturn, error) {
	var returns []trade.SalesReturn
	if err := r.db.WithContext(ctx).
		Preload("Items").
		Where("tenant_id = ? AND sales_order_id = ?", tenantID, salesOrderID).
		Order("created_at DESC").
		Find(&returns).Error; err != nil {
		return nil, err
	}
	return returns, nil
}

// FindByStatus finds sales returns by status for a tenant
func (r *GormSalesReturnRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status trade.ReturnStatus, filter shared.Filter) ([]trade.SalesReturn, error) {
	var returns []trade.SalesReturn
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&trade.SalesReturn{}).
			Where("tenant_id = ? AND status = ?", tenantID, status),
		filter,
	)

	if err := query.Find(&returns).Error; err != nil {
		return nil, err
	}
	return returns, nil
}

// FindPendingApproval finds sales returns pending approval
func (r *GormSalesReturnRepository) FindPendingApproval(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]trade.SalesReturn, error) {
	return r.FindByStatus(ctx, tenantID, trade.ReturnStatusPending, filter)
}

// Save creates or updates a sales return
func (r *GormSalesReturnRepository) Save(ctx context.Context, sr *trade.SalesReturn) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Save the return without auto-saving associations
		if err := tx.Omit("Items").Save(sr).Error; err != nil {
			return err
		}

		// Handle items: delete removed items and save/update existing ones
		if sr.ID != uuid.Nil {
			// Get existing item IDs
			currentItemIDs := make([]uuid.UUID, len(sr.Items))
			for i, item := range sr.Items {
				currentItemIDs[i] = item.ID
			}

			// Delete items not in the current list
			if len(currentItemIDs) > 0 {
				if err := tx.Where("return_id = ? AND id NOT IN ?", sr.ID, currentItemIDs).
					Delete(&trade.SalesReturnItem{}).Error; err != nil {
					return err
				}
			} else {
				// Delete all items if no items remain
				if err := tx.Where("return_id = ?", sr.ID).
					Delete(&trade.SalesReturnItem{}).Error; err != nil {
					return err
				}
			}

			// Save/update remaining items
			for i := range sr.Items {
				sr.Items[i].ReturnID = sr.ID
				if err := tx.Save(&sr.Items[i]).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// SaveWithLock saves with optimistic locking (version check)
func (r *GormSalesReturnRepository) SaveWithLock(ctx context.Context, sr *trade.SalesReturn) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get current version from database
		var currentVersion int
		if err := tx.Model(&trade.SalesReturn{}).
			Where("id = ?", sr.ID).
			Select("version").
			Scan(&currentVersion).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return shared.ErrNotFound
			}
			return err
		}

		// Check version matches
		if currentVersion != sr.Version {
			return shared.NewDomainError("CONCURRENT_MODIFICATION", "The return has been modified by another user")
		}

		// Increment version
		sr.Version++
		sr.UpdatedAt = time.Now()

		// Update return with version check
		result := tx.Model(&trade.SalesReturn{}).
			Where("id = ? AND version = ?", sr.ID, currentVersion).
			Updates(map[string]any{
				"sales_order_id":     sr.SalesOrderID,
				"sales_order_number": sr.SalesOrderNumber,
				"customer_id":        sr.CustomerID,
				"customer_name":      sr.CustomerName,
				"warehouse_id":       sr.WarehouseID,
				"total_refund":       sr.TotalRefund,
				"status":             sr.Status,
				"reason":             sr.Reason,
				"remark":             sr.Remark,
				"submitted_at":       sr.SubmittedAt,
				"approved_at":        sr.ApprovedAt,
				"approved_by":        sr.ApprovedBy,
				"approval_note":      sr.ApprovalNote,
				"rejected_at":        sr.RejectedAt,
				"rejected_by":        sr.RejectedBy,
				"rejection_reason":   sr.RejectionReason,
				"completed_at":       sr.CompletedAt,
				"cancelled_at":       sr.CancelledAt,
				"cancel_reason":      sr.CancelReason,
				"version":            sr.Version,
				"updated_at":         sr.UpdatedAt,
			})

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return shared.NewDomainError("CONCURRENT_MODIFICATION", "The return has been modified by another user")
		}

		// Handle items
		currentItemIDs := make([]uuid.UUID, len(sr.Items))
		for i, item := range sr.Items {
			currentItemIDs[i] = item.ID
		}

		// Delete items not in the current list
		if len(currentItemIDs) > 0 {
			if err := tx.Where("return_id = ? AND id NOT IN ?", sr.ID, currentItemIDs).
				Delete(&trade.SalesReturnItem{}).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Where("return_id = ?", sr.ID).
				Delete(&trade.SalesReturnItem{}).Error; err != nil {
				return err
			}
		}

		// Save/update remaining items
		for i := range sr.Items {
			sr.Items[i].ReturnID = sr.ID
			if err := tx.Save(&sr.Items[i]).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// Delete deletes a sales return (soft delete)
func (r *GormSalesReturnRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete items first
		if err := tx.Where("return_id = ?", id).Delete(&trade.SalesReturnItem{}).Error; err != nil {
			return err
		}

		// Delete return
		result := tx.Delete(&trade.SalesReturn{}, "id = ?", id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return shared.ErrNotFound
		}
		return nil
	})
}

// DeleteForTenant deletes a sales return for a tenant
func (r *GormSalesReturnRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Find the return first
		var sr trade.SalesReturn
		if err := tx.Where("tenant_id = ? AND id = ?", tenantID, id).First(&sr).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return shared.ErrNotFound
			}
			return err
		}

		// Delete items
		if err := tx.Where("return_id = ?", id).Delete(&trade.SalesReturnItem{}).Error; err != nil {
			return err
		}

		// Delete return
		result := tx.Delete(&trade.SalesReturn{}, "tenant_id = ? AND id = ?", tenantID, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return shared.ErrNotFound
		}
		return nil
	})
}

// CountForTenant counts sales returns for a tenant with optional filters
func (r *GormSalesReturnRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&trade.SalesReturn{}).Where("tenant_id = ?", tenantID)
	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByStatus counts sales returns by status for a tenant
func (r *GormSalesReturnRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status trade.ReturnStatus) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&trade.SalesReturn{}).
		Where("tenant_id = ? AND status = ?", tenantID, status).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByCustomer counts sales returns for a customer
func (r *GormSalesReturnRepository) CountByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&trade.SalesReturn{}).
		Where("tenant_id = ? AND customer_id = ?", tenantID, customerID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountBySalesOrder counts sales returns for a sales order
func (r *GormSalesReturnRepository) CountBySalesOrder(ctx context.Context, tenantID, salesOrderID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&trade.SalesReturn{}).
		Where("tenant_id = ? AND sales_order_id = ?", tenantID, salesOrderID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountPendingApproval counts returns pending approval for a tenant
func (r *GormSalesReturnRepository) CountPendingApproval(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	return r.CountByStatus(ctx, tenantID, trade.ReturnStatusPending)
}

// ExistsByReturnNumber checks if a return number exists for a tenant
func (r *GormSalesReturnRepository) ExistsByReturnNumber(ctx context.Context, tenantID uuid.UUID, returnNumber string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&trade.SalesReturn{}).
		Where("tenant_id = ? AND return_number = ?", tenantID, returnNumber).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// GenerateReturnNumber generates a unique return number for a tenant
// Format: SR-YYYY-NNNNN (e.g., SR-2026-00001)
func (r *GormSalesReturnRepository) GenerateReturnNumber(ctx context.Context, tenantID uuid.UUID) (string, error) {
	year := time.Now().Year()
	prefix := fmt.Sprintf("SR-%d-", year)

	// Get the highest return number for this year
	var lastReturn trade.SalesReturn
	err := r.db.WithContext(ctx).
		Model(&trade.SalesReturn{}).
		Where("tenant_id = ? AND return_number LIKE ?", tenantID, prefix+"%").
		Order("return_number DESC").
		First(&lastReturn).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	var nextNum int64 = 1
	if err == nil && lastReturn.ReturnNumber != "" {
		// Parse the number from the last return number
		parts := strings.Split(lastReturn.ReturnNumber, "-")
		if len(parts) == 3 {
			var num int64
			_, parseErr := fmt.Sscanf(parts[2], "%d", &num)
			if parseErr == nil {
				nextNum = num + 1
			}
		}
	}

	// Generate new return number
	returnNumber := fmt.Sprintf("%s%05d", prefix, nextNum)

	// Verify uniqueness
	exists, err := r.ExistsByReturnNumber(ctx, tenantID, returnNumber)
	if err != nil {
		return "", err
	}
	if exists {
		// If exists, try incrementing until we find a unique one
		for range 100 {
			nextNum++
			returnNumber = fmt.Sprintf("%s%05d", prefix, nextNum)
			exists, err = r.ExistsByReturnNumber(ctx, tenantID, returnNumber)
			if err != nil {
				return "", err
			}
			if !exists {
				break
			}
		}
	}

	return returnNumber, nil
}

// applyFilter applies filter options to the query
func (r *GormSalesReturnRepository) applyFilter(query *gorm.DB, filter shared.Filter) *gorm.DB {
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
func (r *GormSalesReturnRepository) applyFilterWithoutPagination(query *gorm.DB, filter shared.Filter) *gorm.DB {
	// Apply search
	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		query = query.Where("return_number ILIKE ? OR customer_name ILIKE ? OR sales_order_number ILIKE ?",
			searchPattern, searchPattern, searchPattern)
	}

	// Apply additional filters
	for key, value := range filter.Filters {
		switch key {
		case "customer_id":
			query = query.Where("customer_id = ?", value)
		case "sales_order_id":
			query = query.Where("sales_order_id = ?", value)
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
				query = query.Where("total_refund >= ?", d)
			}
		case "max_amount":
			if d, ok := value.(decimal.Decimal); ok {
				query = query.Where("total_refund <= ?", d)
			}
		}
	}

	return query
}

// Ensure GormSalesReturnRepository implements SalesReturnRepository
var _ trade.SalesReturnRepository = (*GormSalesReturnRepository)(nil)
