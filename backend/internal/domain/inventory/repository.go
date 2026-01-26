package inventory

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// InventoryItemRepository defines the interface for inventory item persistence
type InventoryItemRepository interface {
	// FindByID finds an inventory item by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*InventoryItem, error)

	// FindByIDForTenant finds an inventory item by ID within a tenant
	FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*InventoryItem, error)

	// FindByWarehouseAndProduct finds inventory by warehouse-product combination
	FindByWarehouseAndProduct(ctx context.Context, tenantID, warehouseID, productID uuid.UUID) (*InventoryItem, error)

	// FindByWarehouse finds all inventory items in a warehouse
	FindByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID, filter shared.Filter) ([]InventoryItem, error)

	// FindByProduct finds all inventory items for a product (across warehouses)
	FindByProduct(ctx context.Context, tenantID, productID uuid.UUID, filter shared.Filter) ([]InventoryItem, error)

	// FindAllForTenant finds all inventory items for a tenant
	FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]InventoryItem, error)

	// FindBelowMinimum finds items below their minimum threshold
	FindBelowMinimum(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]InventoryItem, error)

	// FindWithAvailableStock finds items that have available stock
	FindWithAvailableStock(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]InventoryItem, error)

	// FindByIDs finds multiple inventory items by their IDs
	FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]InventoryItem, error)

	// Save creates or updates an inventory item
	Save(ctx context.Context, item *InventoryItem) error

	// SaveWithLock saves with optimistic locking (checks version)
	SaveWithLock(ctx context.Context, item *InventoryItem) error

	// Delete deletes an inventory item
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteForTenant deletes an inventory item within a tenant
	DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error

	// Count counts inventory items matching the filter
	CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error)

	// CountByWarehouse counts inventory items in a warehouse
	CountByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID) (int64, error)

	// CountByProduct counts inventory items for a product
	CountByProduct(ctx context.Context, tenantID, productID uuid.UUID) (int64, error)

	// SumQuantityByProduct sums total quantity for a product across all warehouses
	SumQuantityByProduct(ctx context.Context, tenantID, productID uuid.UUID) (decimal.Decimal, error)

	// SumValueByWarehouse sums total inventory value in a warehouse
	SumValueByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID) (decimal.Decimal, error)

	// ExistsByWarehouseAndProduct checks if inventory exists for warehouse-product
	ExistsByWarehouseAndProduct(ctx context.Context, tenantID, warehouseID, productID uuid.UUID) (bool, error)

	// GetOrCreate gets existing inventory item or creates a new one
	GetOrCreate(ctx context.Context, tenantID, warehouseID, productID uuid.UUID) (*InventoryItem, error)
}

// StockBatchRepository defines the interface for stock batch persistence.
//
// DDD Aggregate Boundary Notes:
// StockBatch is a child entity within the InventoryItem aggregate. According to DDD principles,
// child entities should be accessed and modified through the aggregate root.
//
// IMPORTANT: This repository should ONLY be used for READ operations that span multiple
// aggregates (e.g., dashboard queries for expiring batches, reporting). All batch modifications
// (create, update, delete) MUST go through the InventoryItem aggregate root methods.
//
// The application layer (InventoryService) does NOT use this repository for modifications.
// Batches are automatically persisted when the InventoryItem aggregate is saved via GORM's
// association handling.
type StockBatchRepository interface {
	// FindByID finds a stock batch by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*StockBatch, error)

	// FindByInventoryItem finds all batches for an inventory item
	FindByInventoryItem(ctx context.Context, inventoryItemID uuid.UUID, filter shared.Filter) ([]StockBatch, error)

	// FindAvailable finds available (non-consumed, non-expired) batches
	FindAvailable(ctx context.Context, inventoryItemID uuid.UUID) ([]StockBatch, error)

	// FindExpiringSoon finds batches expiring within a duration
	FindExpiringSoon(ctx context.Context, tenantID uuid.UUID, withinDays int, filter shared.Filter) ([]StockBatch, error)

	// FindExpired finds expired batches that still have stock
	FindExpired(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]StockBatch, error)

	// FindByBatchNumber finds batches by batch number
	FindByBatchNumber(ctx context.Context, inventoryItemID uuid.UUID, batchNumber string) (*StockBatch, error)

	// Save creates or updates a stock batch
	// Deprecated: Use InventoryItemRepository.Save() instead. Batches should be modified
	// through the aggregate root.
	Save(ctx context.Context, batch *StockBatch) error

	// SaveBatch creates or updates multiple stock batches
	// Deprecated: Use InventoryItemRepository.Save() instead. Batches should be modified
	// through the aggregate root.
	SaveBatch(ctx context.Context, batches []StockBatch) error

	// Delete deletes a stock batch
	// Deprecated: Use InventoryItemRepository.Save() instead. Batches should be modified
	// through the aggregate root.
	Delete(ctx context.Context, id uuid.UUID) error

	// CountByInventoryItem counts batches for an inventory item
	CountByInventoryItem(ctx context.Context, inventoryItemID uuid.UUID) (int64, error)
}

// StockLockRepository defines the interface for stock lock persistence.
//
// DDD Aggregate Boundary Notes:
// StockLock is a child entity within the InventoryItem aggregate. According to DDD principles,
// child entities should be accessed and modified through the aggregate root.
//
// This repository is used for:
//  1. Cross-aggregate READ queries (FindExpired, FindBySource) - needed for business operations
//     that span multiple inventory items (e.g., releasing all locks for a cancelled order)
//  2. Persisting individual lock state changes - locks are stored in a separate table for
//     query performance, but state changes MUST originate from InventoryItem domain methods
//
// The application layer (InventoryService) uses this repository to:
// - Find locks by ID, source, or expiry status (READ operations)
// - Save lock state changes AFTER the aggregate root has processed them (persistence sync)
//
// All lock state transitions (create, release, consume) MUST go through InventoryItem methods:
// - LockStock() creates a lock
// - UnlockStock() releases a lock
// - DeductStock() consumes a lock
type StockLockRepository interface {
	// FindByID finds a stock lock by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*StockLock, error)

	// FindByInventoryItem finds all locks for an inventory item
	FindByInventoryItem(ctx context.Context, inventoryItemID uuid.UUID) ([]StockLock, error)

	// FindActive finds active (non-released, non-consumed) locks
	FindActive(ctx context.Context, inventoryItemID uuid.UUID) ([]StockLock, error)

	// FindExpired finds expired but not released locks
	FindExpired(ctx context.Context) ([]StockLock, error)

	// FindBySource finds locks by source type and ID
	FindBySource(ctx context.Context, sourceType, sourceID string) ([]StockLock, error)

	// Save creates or updates a stock lock
	Save(ctx context.Context, lock *StockLock) error

	// Delete deletes a stock lock
	Delete(ctx context.Context, id uuid.UUID) error

	// ReleaseExpired releases all expired locks and returns count
	ReleaseExpired(ctx context.Context) (int, error)
}

// InventoryTransactionRepository defines the interface for transaction persistence
type InventoryTransactionRepository interface {
	// FindByID finds a transaction by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*InventoryTransaction, error)

	// FindByInventoryItem finds transactions for an inventory item
	FindByInventoryItem(ctx context.Context, inventoryItemID uuid.UUID, filter shared.Filter) ([]InventoryTransaction, error)

	// FindByWarehouse finds transactions for a warehouse
	FindByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID, filter shared.Filter) ([]InventoryTransaction, error)

	// FindByProduct finds transactions for a product
	FindByProduct(ctx context.Context, tenantID, productID uuid.UUID, filter shared.Filter) ([]InventoryTransaction, error)

	// FindBySource finds transactions by source document
	FindBySource(ctx context.Context, sourceType SourceType, sourceID string) ([]InventoryTransaction, error)

	// FindByDateRange finds transactions within a date range
	FindByDateRange(ctx context.Context, tenantID uuid.UUID, start, end time.Time, filter shared.Filter) ([]InventoryTransaction, error)

	// FindByType finds transactions by type
	FindByType(ctx context.Context, tenantID uuid.UUID, txType TransactionType, filter shared.Filter) ([]InventoryTransaction, error)

	// FindForTenant finds all transactions for a tenant
	FindForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]InventoryTransaction, error)

	// Create creates a new transaction (append-only, no update allowed)
	Create(ctx context.Context, tx *InventoryTransaction) error

	// CreateBatch creates multiple transactions
	CreateBatch(ctx context.Context, txs []*InventoryTransaction) error

	// Count counts transactions matching filter
	CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error)

	// CountByInventoryItem counts transactions for an inventory item
	CountByInventoryItem(ctx context.Context, inventoryItemID uuid.UUID) (int64, error)

	// SumQuantityByTypeAndDateRange sums quantities for analysis
	SumQuantityByTypeAndDateRange(ctx context.Context, tenantID uuid.UUID, txType TransactionType, start, end time.Time) (decimal.Decimal, error)
}

// InventoryFilter extends shared.Filter with inventory-specific filters
type InventoryFilter struct {
	shared.Filter
	WarehouseID  *uuid.UUID
	ProductID    *uuid.UUID
	BelowMinimum bool
	AboveMaximum bool
	HasStock     bool
	NoStock      bool
	MinQuantity  *decimal.Decimal
	MaxQuantity  *decimal.Decimal
}

// TransactionFilter extends shared.Filter with transaction-specific filters
type TransactionFilter struct {
	shared.Filter
	WarehouseID     *uuid.UUID
	ProductID       *uuid.UUID
	TransactionType *TransactionType
	SourceType      *SourceType
	SourceID        string
	StartDate       *time.Time
	EndDate         *time.Time
	OperatorID      *uuid.UUID
}

// StockTakingRepository defines the interface for stock taking persistence
type StockTakingRepository interface {
	// FindByID finds a stock taking by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*StockTaking, error)

	// FindByIDForTenant finds a stock taking by ID within a tenant
	FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*StockTaking, error)

	// FindByTakingNumber finds a stock taking by its number
	FindByTakingNumber(ctx context.Context, tenantID uuid.UUID, takingNumber string) (*StockTaking, error)

	// FindByWarehouse finds all stock takings for a warehouse
	FindByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID, filter shared.Filter) ([]StockTaking, error)

	// FindByStatus finds all stock takings with a specific status
	FindByStatus(ctx context.Context, tenantID uuid.UUID, status StockTakingStatus, filter shared.Filter) ([]StockTaking, error)

	// FindAllForTenant finds all stock takings for a tenant
	FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]StockTaking, error)

	// FindByDateRange finds stock takings within a date range
	FindByDateRange(ctx context.Context, tenantID uuid.UUID, start, end time.Time, filter shared.Filter) ([]StockTaking, error)

	// FindPendingApproval finds stock takings pending approval
	FindPendingApproval(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]StockTaking, error)

	// Save creates or updates a stock taking
	Save(ctx context.Context, st *StockTaking) error

	// SaveWithItems saves a stock taking with its items
	SaveWithItems(ctx context.Context, st *StockTaking) error

	// Delete deletes a stock taking
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteForTenant deletes a stock taking within a tenant
	DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error

	// CountForTenant counts stock takings matching the filter
	CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error)

	// CountByStatus counts stock takings by status
	CountByStatus(ctx context.Context, tenantID uuid.UUID, status StockTakingStatus) (int64, error)

	// ExistsByTakingNumber checks if a stock taking number exists
	ExistsByTakingNumber(ctx context.Context, tenantID uuid.UUID, takingNumber string) (bool, error)

	// GenerateTakingNumber generates a new unique taking number
	GenerateTakingNumber(ctx context.Context, tenantID uuid.UUID) (string, error)
}

// StockTakingFilter extends shared.Filter with stock-taking-specific filters
type StockTakingFilter struct {
	shared.Filter
	WarehouseID *uuid.UUID
	Status      *StockTakingStatus
	StartDate   *time.Time
	EndDate     *time.Time
	CreatedByID *uuid.UUID
}
