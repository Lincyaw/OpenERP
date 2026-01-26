package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/erp/backend/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// GormPurchaseReturnRepository implements PurchaseReturnRepository using GORM
type GormPurchaseReturnRepository struct {
	db *gorm.DB
}

// NewGormPurchaseReturnRepository creates a new GormPurchaseReturnRepository
func NewGormPurchaseReturnRepository(db *gorm.DB) *GormPurchaseReturnRepository {
	return &GormPurchaseReturnRepository{db: db}
}

// FindByID finds a purchase return by its ID
func (r *GormPurchaseReturnRepository) FindByID(ctx context.Context, id uuid.UUID) (*trade.PurchaseReturn, error) {
	var model models.PurchaseReturnModel
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

// FindByIDForTenant finds a purchase return by ID within a tenant
func (r *GormPurchaseReturnRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*trade.PurchaseReturn, error) {
	var model models.PurchaseReturnModel
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

// FindByReturnNumber finds a purchase return by return number for a tenant
func (r *GormPurchaseReturnRepository) FindByReturnNumber(ctx context.Context, tenantID uuid.UUID, returnNumber string) (*trade.PurchaseReturn, error) {
	var model models.PurchaseReturnModel
	if err := r.db.WithContext(ctx).
		Preload("Items").
		Where("tenant_id = ? AND return_number = ?", tenantID, returnNumber).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindAllForTenant finds all purchase returns for a tenant with filtering
func (r *GormPurchaseReturnRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]trade.PurchaseReturn, error) {
	var returnModels []models.PurchaseReturnModel
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&models.PurchaseReturnModel{}).Where("tenant_id = ?", tenantID),
		filter,
	)

	if err := query.Find(&returnModels).Error; err != nil {
		return nil, err
	}
	returns := make([]trade.PurchaseReturn, len(returnModels))
	for i, model := range returnModels {
		returns[i] = *model.ToDomain()
	}
	return returns, nil
}

// FindBySupplier finds purchase returns for a supplier
func (r *GormPurchaseReturnRepository) FindBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID, filter shared.Filter) ([]trade.PurchaseReturn, error) {
	var returnModels []models.PurchaseReturnModel
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&models.PurchaseReturnModel{}).
			Where("tenant_id = ? AND supplier_id = ?", tenantID, supplierID),
		filter,
	)

	if err := query.Find(&returnModels).Error; err != nil {
		return nil, err
	}
	returns := make([]trade.PurchaseReturn, len(returnModels))
	for i, model := range returnModels {
		returns[i] = *model.ToDomain()
	}
	return returns, nil
}

// FindByPurchaseOrder finds purchase returns for a purchase order
func (r *GormPurchaseReturnRepository) FindByPurchaseOrder(ctx context.Context, tenantID, purchaseOrderID uuid.UUID) ([]trade.PurchaseReturn, error) {
	var returnModels []models.PurchaseReturnModel
	if err := r.db.WithContext(ctx).
		Preload("Items").
		Where("tenant_id = ? AND purchase_order_id = ?", tenantID, purchaseOrderID).
		Order("created_at DESC").
		Find(&returnModels).Error; err != nil {
		return nil, err
	}
	returns := make([]trade.PurchaseReturn, len(returnModels))
	for i, model := range returnModels {
		returns[i] = *model.ToDomain()
	}
	return returns, nil
}

// FindByStatus finds purchase returns by status for a tenant
func (r *GormPurchaseReturnRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status trade.PurchaseReturnStatus, filter shared.Filter) ([]trade.PurchaseReturn, error) {
	var returnModels []models.PurchaseReturnModel
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&models.PurchaseReturnModel{}).
			Where("tenant_id = ? AND status = ?", tenantID, status),
		filter,
	)

	if err := query.Find(&returnModels).Error; err != nil {
		return nil, err
	}
	returns := make([]trade.PurchaseReturn, len(returnModels))
	for i, model := range returnModels {
		returns[i] = *model.ToDomain()
	}
	return returns, nil
}

// FindPendingApproval finds purchase returns pending approval
func (r *GormPurchaseReturnRepository) FindPendingApproval(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]trade.PurchaseReturn, error) {
	return r.FindByStatus(ctx, tenantID, trade.PurchaseReturnStatusPending, filter)
}

// Save creates or updates a purchase return
func (r *GormPurchaseReturnRepository) Save(ctx context.Context, pr *trade.PurchaseReturn) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Convert to persistence model
		model := models.PurchaseReturnModelFromDomain(pr)

		// Save the return without auto-saving associations
		if err := tx.Omit("Items").Save(model).Error; err != nil {
			return err
		}

		// Handle items: delete removed items and save/update existing ones
		if pr.ID != uuid.Nil {
			// Get existing item IDs
			currentItemIDs := make([]uuid.UUID, len(pr.Items))
			for i, item := range pr.Items {
				currentItemIDs[i] = item.ID
			}

			// Delete items not in the current list
			if len(currentItemIDs) > 0 {
				if err := tx.Where("return_id = ? AND id NOT IN ?", pr.ID, currentItemIDs).
					Delete(&models.PurchaseReturnItemModel{}).Error; err != nil {
					return err
				}
			} else {
				// Delete all items if no items remain
				if err := tx.Where("return_id = ?", pr.ID).
					Delete(&models.PurchaseReturnItemModel{}).Error; err != nil {
					return err
				}
			}

			// Save/update remaining items
			for i := range pr.Items {
				pr.Items[i].ReturnID = pr.ID
				itemModel := models.PurchaseReturnItemModelFromDomain(&pr.Items[i])
				if err := tx.Save(itemModel).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// SaveWithLock saves with optimistic locking (version check)
func (r *GormPurchaseReturnRepository) SaveWithLock(ctx context.Context, pr *trade.PurchaseReturn) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get current version from database
		var currentVersion int
		if err := tx.Model(&models.PurchaseReturnModel{}).
			Where("id = ?", pr.ID).
			Select("version").
			Scan(&currentVersion).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return shared.ErrNotFound
			}
			return err
		}

		// Check version matches
		if currentVersion != pr.Version {
			return shared.NewDomainError("CONCURRENT_MODIFICATION", "The return has been modified by another user")
		}

		// Increment version
		pr.Version++
		pr.UpdatedAt = time.Now()

		// Update return with version check
		result := tx.Model(&models.PurchaseReturnModel{}).
			Where("id = ? AND version = ?", pr.ID, currentVersion).
			Updates(map[string]any{
				"purchase_order_id":     pr.PurchaseOrderID,
				"purchase_order_number": pr.PurchaseOrderNumber,
				"supplier_id":           pr.SupplierID,
				"supplier_name":         pr.SupplierName,
				"warehouse_id":          pr.WarehouseID,
				"total_refund":          pr.TotalRefund,
				"status":                pr.Status,
				"reason":                pr.Reason,
				"remark":                pr.Remark,
				"submitted_at":          pr.SubmittedAt,
				"approved_at":           pr.ApprovedAt,
				"approved_by":           pr.ApprovedBy,
				"approval_note":         pr.ApprovalNote,
				"rejected_at":           pr.RejectedAt,
				"rejected_by":           pr.RejectedBy,
				"rejection_reason":      pr.RejectionReason,
				"shipped_at":            pr.ShippedAt,
				"shipped_by":            pr.ShippedBy,
				"shipping_note":         pr.ShippingNote,
				"tracking_number":       pr.TrackingNumber,
				"completed_at":          pr.CompletedAt,
				"cancelled_at":          pr.CancelledAt,
				"cancel_reason":         pr.CancelReason,
				"version":               pr.Version,
				"updated_at":            pr.UpdatedAt,
			})

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return shared.NewDomainError("CONCURRENT_MODIFICATION", "The return has been modified by another user")
		}

		// Handle items
		currentItemIDs := make([]uuid.UUID, len(pr.Items))
		for i, item := range pr.Items {
			currentItemIDs[i] = item.ID
		}

		// Delete items not in the current list
		if len(currentItemIDs) > 0 {
			if err := tx.Where("return_id = ? AND id NOT IN ?", pr.ID, currentItemIDs).
				Delete(&models.PurchaseReturnItemModel{}).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Where("return_id = ?", pr.ID).
				Delete(&models.PurchaseReturnItemModel{}).Error; err != nil {
				return err
			}
		}

		// Save/update remaining items
		for i := range pr.Items {
			pr.Items[i].ReturnID = pr.ID
			itemModel := models.PurchaseReturnItemModelFromDomain(&pr.Items[i])
			if err := tx.Save(itemModel).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// Delete deletes a purchase return (soft delete)
func (r *GormPurchaseReturnRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete items first
		if err := tx.Where("return_id = ?", id).Delete(&models.PurchaseReturnItemModel{}).Error; err != nil {
			return err
		}

		// Delete return
		result := tx.Delete(&models.PurchaseReturnModel{}, "id = ?", id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return shared.ErrNotFound
		}
		return nil
	})
}

// DeleteForTenant deletes a purchase return for a tenant
func (r *GormPurchaseReturnRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Find the return first
		var model models.PurchaseReturnModel
		if err := tx.Where("tenant_id = ? AND id = ?", tenantID, id).First(&model).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return shared.ErrNotFound
			}
			return err
		}

		// Delete items
		if err := tx.Where("return_id = ?", id).Delete(&models.PurchaseReturnItemModel{}).Error; err != nil {
			return err
		}

		// Delete return
		result := tx.Delete(&models.PurchaseReturnModel{}, "tenant_id = ? AND id = ?", tenantID, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return shared.ErrNotFound
		}
		return nil
	})
}

// CountForTenant counts purchase returns for a tenant with optional filters
func (r *GormPurchaseReturnRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.PurchaseReturnModel{}).Where("tenant_id = ?", tenantID)
	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByStatus counts purchase returns by status for a tenant
func (r *GormPurchaseReturnRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status trade.PurchaseReturnStatus) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.PurchaseReturnModel{}).
		Where("tenant_id = ? AND status = ?", tenantID, status).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountBySupplier counts purchase returns for a supplier
func (r *GormPurchaseReturnRepository) CountBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.PurchaseReturnModel{}).
		Where("tenant_id = ? AND supplier_id = ?", tenantID, supplierID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByPurchaseOrder counts purchase returns for a purchase order
func (r *GormPurchaseReturnRepository) CountByPurchaseOrder(ctx context.Context, tenantID, purchaseOrderID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.PurchaseReturnModel{}).
		Where("tenant_id = ? AND purchase_order_id = ?", tenantID, purchaseOrderID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountPendingApproval counts returns pending approval for a tenant
func (r *GormPurchaseReturnRepository) CountPendingApproval(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	return r.CountByStatus(ctx, tenantID, trade.PurchaseReturnStatusPending)
}

// ExistsByReturnNumber checks if a return number exists for a tenant
func (r *GormPurchaseReturnRepository) ExistsByReturnNumber(ctx context.Context, tenantID uuid.UUID, returnNumber string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.PurchaseReturnModel{}).
		Where("tenant_id = ? AND return_number = ?", tenantID, returnNumber).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// GenerateReturnNumber generates a unique return number for a tenant
// Format: PR-YYYY-NNNNN (e.g., PR-2026-00001)
func (r *GormPurchaseReturnRepository) GenerateReturnNumber(ctx context.Context, tenantID uuid.UUID) (string, error) {
	year := time.Now().Year()
	prefix := fmt.Sprintf("PR-%d-", year)

	// Get the highest return number for this year
	var lastReturn models.PurchaseReturnModel
	err := r.db.WithContext(ctx).
		Model(&models.PurchaseReturnModel{}).
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
func (r *GormPurchaseReturnRepository) applyFilter(query *gorm.DB, filter shared.Filter) *gorm.DB {
	query = r.applyFilterWithoutPagination(query, filter)

	// Apply pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// Apply ordering with whitelist validation to prevent SQL injection
	if filter.OrderBy != "" {
		sortField := ValidateSortField(filter.OrderBy, PurchaseReturnSortFields, "")
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
func (r *GormPurchaseReturnRepository) applyFilterWithoutPagination(query *gorm.DB, filter shared.Filter) *gorm.DB {
	// Apply search
	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		query = query.Where("return_number ILIKE ? OR supplier_name ILIKE ? OR purchase_order_number ILIKE ?",
			searchPattern, searchPattern, searchPattern)
	}

	// Apply additional filters
	for key, value := range filter.Filters {
		switch key {
		case "supplier_id":
			query = query.Where("supplier_id = ?", value)
		case "purchase_order_id":
			query = query.Where("purchase_order_id = ?", value)
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

// Ensure GormPurchaseReturnRepository implements PurchaseReturnRepository
var _ trade.PurchaseReturnRepository = (*GormPurchaseReturnRepository)(nil)
