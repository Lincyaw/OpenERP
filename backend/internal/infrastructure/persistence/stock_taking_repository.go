package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GormStockTakingRepository implements StockTakingRepository using GORM
type GormStockTakingRepository struct {
	db *gorm.DB
}

// NewGormStockTakingRepository creates a new GormStockTakingRepository
func NewGormStockTakingRepository(db *gorm.DB) *GormStockTakingRepository {
	return &GormStockTakingRepository{db: db}
}

// FindByID finds a stock taking by its ID
func (r *GormStockTakingRepository) FindByID(ctx context.Context, id uuid.UUID) (*inventory.StockTaking, error) {
	var model models.StockTakingModel
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

// FindByIDForTenant finds a stock taking by ID within a tenant
func (r *GormStockTakingRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*inventory.StockTaking, error) {
	var model models.StockTakingModel
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

// FindByTakingNumber finds a stock taking by its number
func (r *GormStockTakingRepository) FindByTakingNumber(ctx context.Context, tenantID uuid.UUID, takingNumber string) (*inventory.StockTaking, error) {
	var model models.StockTakingModel
	if err := r.db.WithContext(ctx).
		Preload("Items").
		Where("tenant_id = ? AND taking_number = ?", tenantID, takingNumber).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByWarehouse finds all stock takings for a warehouse
func (r *GormStockTakingRepository) FindByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID, filter shared.Filter) ([]inventory.StockTaking, error) {
	var stModels []models.StockTakingModel
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&models.StockTakingModel{}).
			Where("tenant_id = ? AND warehouse_id = ?", tenantID, warehouseID),
		filter,
	)

	if err := query.Find(&stModels).Error; err != nil {
		return nil, err
	}
	sts := make([]inventory.StockTaking, len(stModels))
	for i, model := range stModels {
		sts[i] = *model.ToDomain()
	}
	return sts, nil
}

// FindByStatus finds all stock takings with a specific status
func (r *GormStockTakingRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status inventory.StockTakingStatus, filter shared.Filter) ([]inventory.StockTaking, error) {
	var stModels []models.StockTakingModel
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&models.StockTakingModel{}).
			Where("tenant_id = ? AND status = ?", tenantID, status),
		filter,
	)

	if err := query.Find(&stModels).Error; err != nil {
		return nil, err
	}
	sts := make([]inventory.StockTaking, len(stModels))
	for i, model := range stModels {
		sts[i] = *model.ToDomain()
	}
	return sts, nil
}

// FindAllForTenant finds all stock takings for a tenant
func (r *GormStockTakingRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]inventory.StockTaking, error) {
	var stModels []models.StockTakingModel
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&models.StockTakingModel{}).
			Where("tenant_id = ?", tenantID),
		filter,
	)

	if err := query.Find(&stModels).Error; err != nil {
		return nil, err
	}
	sts := make([]inventory.StockTaking, len(stModels))
	for i, model := range stModels {
		sts[i] = *model.ToDomain()
	}
	return sts, nil
}

// FindByDateRange finds stock takings within a date range
func (r *GormStockTakingRepository) FindByDateRange(ctx context.Context, tenantID uuid.UUID, start, end time.Time, filter shared.Filter) ([]inventory.StockTaking, error) {
	var stModels []models.StockTakingModel
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&models.StockTakingModel{}).
			Where("tenant_id = ? AND taking_date >= ? AND taking_date <= ?", tenantID, start, end),
		filter,
	)

	if err := query.Find(&stModels).Error; err != nil {
		return nil, err
	}
	sts := make([]inventory.StockTaking, len(stModels))
	for i, model := range stModels {
		sts[i] = *model.ToDomain()
	}
	return sts, nil
}

// FindPendingApproval finds stock takings pending approval
func (r *GormStockTakingRepository) FindPendingApproval(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]inventory.StockTaking, error) {
	return r.FindByStatus(ctx, tenantID, inventory.StockTakingStatusPendingApproval, filter)
}

// Save creates or updates a stock taking (without items)
func (r *GormStockTakingRepository) Save(ctx context.Context, st *inventory.StockTaking) error {
	model := models.StockTakingModelFromDomain(st)
	return r.db.WithContext(ctx).Save(model).Error
}

// SaveWithItems saves a stock taking with its items in a transaction
func (r *GormStockTakingRepository) SaveWithItems(ctx context.Context, st *inventory.StockTaking) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Save the stock taking
		model := models.StockTakingModelFromDomain(st)
		if err := tx.Save(model).Error; err != nil {
			return err
		}

		// Delete existing items that are no longer in the list
		var existingItemIDs []uuid.UUID
		for _, item := range st.Items {
			existingItemIDs = append(existingItemIDs, item.ID)
		}

		if len(existingItemIDs) > 0 {
			if err := tx.Where("stock_taking_id = ? AND id NOT IN ?", st.ID, existingItemIDs).
				Delete(&models.StockTakingItemModel{}).Error; err != nil {
				return err
			}
		} else {
			// Delete all items if none remain
			if err := tx.Where("stock_taking_id = ?", st.ID).
				Delete(&models.StockTakingItemModel{}).Error; err != nil {
				return err
			}
		}

		// Save/update all items
		for i := range st.Items {
			st.Items[i].StockTakingID = st.ID
			itemModel := models.StockTakingItemModelFromDomain(&st.Items[i])
			if err := tx.Save(itemModel).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// Delete deletes a stock taking
func (r *GormStockTakingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete items first
		if err := tx.Where("stock_taking_id = ?", id).Delete(&models.StockTakingItemModel{}).Error; err != nil {
			return err
		}
		// Delete the stock taking
		result := tx.Delete(&models.StockTakingModel{}, "id = ?", id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return shared.ErrNotFound
		}
		return nil
	})
}

// DeleteForTenant deletes a stock taking within a tenant
func (r *GormStockTakingRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Verify the stock taking belongs to the tenant
		var model models.StockTakingModel
		if err := tx.Where("tenant_id = ? AND id = ?", tenantID, id).First(&model).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return shared.ErrNotFound
			}
			return err
		}

		// Delete items first
		if err := tx.Where("stock_taking_id = ?", id).Delete(&models.StockTakingItemModel{}).Error; err != nil {
			return err
		}

		// Delete the stock taking
		return tx.Delete(&model).Error
	})
}

// CountForTenant counts stock takings matching the filter
func (r *GormStockTakingRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.StockTakingModel{}).
		Where("tenant_id = ?", tenantID)

	// Apply search filter
	if filter.Search != "" {
		searchPattern := "%" + strings.ToLower(filter.Search) + "%"
		query = query.Where("LOWER(taking_number) LIKE ? OR LOWER(warehouse_name) LIKE ? OR LOWER(created_by_name) LIKE ?",
			searchPattern, searchPattern, searchPattern)
	}

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByStatus counts stock takings by status
func (r *GormStockTakingRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status inventory.StockTakingStatus) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.StockTakingModel{}).
		Where("tenant_id = ? AND status = ?", tenantID, status).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ExistsByTakingNumber checks if a stock taking number exists
func (r *GormStockTakingRepository) ExistsByTakingNumber(ctx context.Context, tenantID uuid.UUID, takingNumber string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.StockTakingModel{}).
		Where("tenant_id = ? AND taking_number = ?", tenantID, takingNumber).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// GenerateTakingNumber generates a new unique taking number
func (r *GormStockTakingRepository) GenerateTakingNumber(ctx context.Context, tenantID uuid.UUID) (string, error) {
	// Format: ST-YYYYMMDD-XXXX
	today := time.Now().Format("20060102")
	prefix := fmt.Sprintf("ST-%s-", today)

	// Find the max sequence number for today
	var maxNumber string
	err := r.db.WithContext(ctx).Model(&models.StockTakingModel{}).
		Select("taking_number").
		Where("tenant_id = ? AND taking_number LIKE ?", tenantID, prefix+"%").
		Order("taking_number DESC").
		Limit(1).
		Pluck("taking_number", &maxNumber).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	var seq int
	if maxNumber != "" {
		// Extract sequence number from the last part
		parts := strings.Split(maxNumber, "-")
		if len(parts) >= 3 {
			_, err := fmt.Sscanf(parts[len(parts)-1], "%04d", &seq)
			if err == nil {
				seq++
			}
		}
	}
	if seq == 0 {
		seq = 1
	}

	return fmt.Sprintf("%s%04d", prefix, seq), nil
}

// applyFilter applies common filter options to a query
func (r *GormStockTakingRepository) applyFilter(query *gorm.DB, filter shared.Filter) *gorm.DB {
	// Apply search filter
	if filter.Search != "" {
		searchPattern := "%" + strings.ToLower(filter.Search) + "%"
		query = query.Where("LOWER(taking_number) LIKE ? OR LOWER(warehouse_name) LIKE ? OR LOWER(created_by_name) LIKE ?",
			searchPattern, searchPattern, searchPattern)
	}

	// Apply pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// Apply ordering
	orderBy := "created_at"
	if filter.OrderBy != "" {
		// Validate order by field to prevent SQL injection
		validFields := map[string]bool{
			"taking_number": true,
			"taking_date":   true,
			"status":        true,
			"created_at":    true,
			"updated_at":    true,
			"total_items":   true,
		}
		if validFields[filter.OrderBy] {
			orderBy = filter.OrderBy
		}
	}

	orderDir := "DESC"
	if filter.OrderDir != "" && (filter.OrderDir == "asc" || filter.OrderDir == "ASC") {
		orderDir = "ASC"
	}

	query = query.Order(fmt.Sprintf("%s %s", orderBy, orderDir))

	return query
}

// Ensure GormStockTakingRepository implements StockTakingRepository
var _ inventory.StockTakingRepository = (*GormStockTakingRepository)(nil)
