package inventory

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/shared"
	"go.uber.org/zap"
)

// StockLockExpirationService handles automatic expiration of stock locks
type StockLockExpirationService struct {
	lockRepo      inventory.StockLockRepository
	inventoryRepo inventory.InventoryItemRepository
	eventBus      shared.EventBus
	logger        *zap.Logger
}

// NewStockLockExpirationService creates a new StockLockExpirationService
func NewStockLockExpirationService(
	lockRepo inventory.StockLockRepository,
	inventoryRepo inventory.InventoryItemRepository,
	eventBus shared.EventBus,
	logger *zap.Logger,
) *StockLockExpirationService {
	return &StockLockExpirationService{
		lockRepo:      lockRepo,
		inventoryRepo: inventoryRepo,
		eventBus:      eventBus,
		logger:        logger,
	}
}

// SetEventBus sets the event bus for publishing events
// This is useful when the event bus is not available at construction time
func (s *StockLockExpirationService) SetEventBus(eventBus shared.EventBus) {
	s.eventBus = eventBus
}

// ExpiredLockStats contains statistics about expired lock processing
type ExpiredLockStats struct {
	TotalExpired    int       `json:"total_expired"`
	SuccessReleased int       `json:"success_released"`
	FailedReleases  int       `json:"failed_releases"`
	ProcessedAt     time.Time `json:"processed_at"`
}

// ReleaseExpiredLocks finds and releases all expired locks, publishing events for each
func (s *StockLockExpirationService) ReleaseExpiredLocks(ctx context.Context) (*ExpiredLockStats, error) {
	stats := &ExpiredLockStats{
		ProcessedAt: time.Now(),
	}

	// Find all expired locks
	expiredLocks, err := s.lockRepo.FindExpired(ctx)
	if err != nil {
		s.logger.Error("Failed to find expired locks", zap.Error(err))
		return nil, err
	}

	stats.TotalExpired = len(expiredLocks)
	if stats.TotalExpired == 0 {
		s.logger.Debug("No expired stock locks found")
		return stats, nil
	}

	s.logger.Info("Found expired stock locks",
		zap.Int("count", stats.TotalExpired),
	)

	// Process each expired lock
	for _, lock := range expiredLocks {
		if err := s.releaseExpiredLock(ctx, &lock); err != nil {
			s.logger.Error("Failed to release expired lock",
				zap.String("lock_id", lock.ID.String()),
				zap.String("source_type", lock.SourceType),
				zap.String("source_id", lock.SourceID),
				zap.Error(err),
			)
			stats.FailedReleases++
			continue
		}
		stats.SuccessReleased++
	}

	s.logger.Info("Completed expired lock release",
		zap.Int("total", stats.TotalExpired),
		zap.Int("released", stats.SuccessReleased),
		zap.Int("failed", stats.FailedReleases),
	)

	return stats, nil
}

// releaseExpiredLock releases a single expired lock and publishes an event
func (s *StockLockExpirationService) releaseExpiredLock(ctx context.Context, lock *inventory.StockLock) error {
	// Get inventory item first to get info for event and update quantities
	item, err := s.inventoryRepo.FindByID(ctx, lock.InventoryItemID)
	if err != nil {
		s.logger.Warn("Could not find inventory item for expired lock",
			zap.String("lock_id", lock.ID.String()),
			zap.String("inventory_item_id", lock.InventoryItemID.String()),
			zap.Error(err),
		)
		// Still release the lock even if we can't find the inventory item
		lock.Release()
		return s.lockRepo.Save(ctx, lock)
	}

	// Store lock info before releasing (for event)
	lockQuantity := lock.Quantity
	sourceType := lock.SourceType
	sourceID := lock.SourceID
	lockID := lock.ID

	// Mark the lock as released
	lock.Release()

	// Save the lock
	if err := s.lockRepo.Save(ctx, lock); err != nil {
		return err
	}

	// Update inventory item quantities directly
	// Move quantity from locked back to available using type-safe Quantity operations
	qtyToRelease, err := inventory.NewInventoryQuantity(lockQuantity)
	if err != nil {
		return err
	}

	newLocked, err := item.LockedQuantity.Subtract(qtyToRelease)
	if err != nil {
		// If subtraction fails (would be negative), set to zero
		newLocked = inventory.ZeroInventoryQuantity()
	}
	item.LockedQuantity = newLocked

	newAvailable, err := item.AvailableQuantity.Add(qtyToRelease)
	if err != nil {
		return err
	}
	item.AvailableQuantity = newAvailable
	item.IncrementVersion()

	if err := s.inventoryRepo.Save(ctx, item); err != nil {
		s.logger.Warn("Failed to update inventory quantities after lock expiration",
			zap.String("inventory_item_id", item.ID.String()),
			zap.Error(err),
		)
		// Don't return error - the lock was already released
	}

	// Publish StockLockExpired event
	if s.eventBus != nil {
		event := inventory.NewStockLockExpiredEvent(
			item.TenantID,
			item.ID,
			item.WarehouseID,
			item.ProductID,
			lockID,
			lockQuantity,
			sourceType,
			sourceID,
		)

		if err := s.eventBus.Publish(ctx, event); err != nil {
			s.logger.Warn("Failed to publish StockLockExpired event",
				zap.String("lock_id", lockID.String()),
				zap.Error(err),
			)
			// Don't return error - the lock was already released
		}
	}

	s.logger.Debug("Released expired stock lock",
		zap.String("lock_id", lockID.String()),
		zap.String("inventory_item_id", lock.InventoryItemID.String()),
		zap.String("source_type", sourceType),
		zap.String("source_id", sourceID),
		zap.String("quantity", lockQuantity.String()),
	)

	return nil
}

// BulkReleaseExpiredLocks releases all expired locks in a single database operation
// This is more efficient but doesn't publish individual events or update inventory quantities
// Use this only for cleanup scenarios where events are not needed
func (s *StockLockExpirationService) BulkReleaseExpiredLocks(ctx context.Context) (int, error) {
	count, err := s.lockRepo.ReleaseExpired(ctx)
	if err != nil {
		s.logger.Error("Failed to bulk release expired locks", zap.Error(err))
		return 0, err
	}

	if count > 0 {
		s.logger.Info("Bulk released expired stock locks",
			zap.Int("count", count),
		)
	}

	return count, nil
}

// GetExpiredLockCount returns the count of currently expired but unreleased locks
func (s *StockLockExpirationService) GetExpiredLockCount(ctx context.Context) (int, error) {
	locks, err := s.lockRepo.FindExpired(ctx)
	if err != nil {
		return 0, err
	}
	return len(locks), nil
}
