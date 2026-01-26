package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GormStockLockRepository implements StockLockRepository using GORM
type GormStockLockRepository struct {
	db *gorm.DB
}

// NewGormStockLockRepository creates a new GormStockLockRepository
func NewGormStockLockRepository(db *gorm.DB) *GormStockLockRepository {
	return &GormStockLockRepository{db: db}
}

// WithTx returns a new repository instance with the given transaction
func (r *GormStockLockRepository) WithTx(tx *gorm.DB) *GormStockLockRepository {
	return &GormStockLockRepository{db: tx}
}

// FindByID finds a stock lock by its ID
func (r *GormStockLockRepository) FindByID(ctx context.Context, id uuid.UUID) (*inventory.StockLock, error) {
	var model models.StockLockModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByInventoryItem finds all locks for an inventory item
func (r *GormStockLockRepository) FindByInventoryItem(ctx context.Context, inventoryItemID uuid.UUID) ([]inventory.StockLock, error) {
	var lockModels []models.StockLockModel
	if err := r.db.WithContext(ctx).
		Where("inventory_item_id = ?", inventoryItemID).
		Order("created_at DESC").
		Find(&lockModels).Error; err != nil {
		return nil, err
	}
	locks := make([]inventory.StockLock, len(lockModels))
	for i, model := range lockModels {
		locks[i] = *model.ToDomain()
	}
	return locks, nil
}

// FindActive finds active (non-released, non-consumed) locks
func (r *GormStockLockRepository) FindActive(ctx context.Context, inventoryItemID uuid.UUID) ([]inventory.StockLock, error) {
	var lockModels []models.StockLockModel
	if err := r.db.WithContext(ctx).
		Where("inventory_item_id = ? AND released = FALSE AND consumed = FALSE", inventoryItemID).
		Order("created_at ASC").
		Find(&lockModels).Error; err != nil {
		return nil, err
	}
	locks := make([]inventory.StockLock, len(lockModels))
	for i, model := range lockModels {
		locks[i] = *model.ToDomain()
	}
	return locks, nil
}

// FindExpired finds expired but not released locks
func (r *GormStockLockRepository) FindExpired(ctx context.Context) ([]inventory.StockLock, error) {
	var lockModels []models.StockLockModel
	now := time.Now()
	if err := r.db.WithContext(ctx).
		Where("released = FALSE AND consumed = FALSE AND expire_at < ?", now).
		Order("expire_at ASC").
		Find(&lockModels).Error; err != nil {
		return nil, err
	}
	locks := make([]inventory.StockLock, len(lockModels))
	for i, model := range lockModels {
		locks[i] = *model.ToDomain()
	}
	return locks, nil
}

// FindBySource finds locks by source type and ID
func (r *GormStockLockRepository) FindBySource(ctx context.Context, sourceType, sourceID string) ([]inventory.StockLock, error) {
	var lockModels []models.StockLockModel
	if err := r.db.WithContext(ctx).
		Where("source_type = ? AND source_id = ?", sourceType, sourceID).
		Order("created_at ASC").
		Find(&lockModels).Error; err != nil {
		return nil, err
	}
	locks := make([]inventory.StockLock, len(lockModels))
	for i, model := range lockModels {
		locks[i] = *model.ToDomain()
	}
	return locks, nil
}

// Save creates or updates a stock lock
func (r *GormStockLockRepository) Save(ctx context.Context, lock *inventory.StockLock) error {
	model := models.StockLockModelFromDomain(lock)
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete deletes a stock lock
func (r *GormStockLockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.StockLockModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// ReleaseExpired releases all expired locks and returns count
func (r *GormStockLockRepository) ReleaseExpired(ctx context.Context) (int, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&models.StockLockModel{}).
		Where("released = FALSE AND consumed = FALSE AND expire_at < ?", now).
		Updates(map[string]any{
			"released":    true,
			"released_at": now,
			"updated_at":  now,
		})

	if result.Error != nil {
		return 0, result.Error
	}
	return int(result.RowsAffected), nil
}

// Ensure GormStockLockRepository implements StockLockRepository
var _ inventory.StockLockRepository = (*GormStockLockRepository)(nil)
