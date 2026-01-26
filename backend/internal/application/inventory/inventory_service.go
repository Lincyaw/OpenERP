package inventory

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	// DefaultLockExpiry is the default lock expiry duration (30 minutes)
	DefaultLockExpiry = 30 * time.Minute
)

// CostStrategyProvider provides cost strategies based on tenant configuration
type CostStrategyProvider interface {
	// GetCostStrategy returns the cost strategy for the given name
	GetCostStrategy(name string) (strategy.CostCalculationStrategy, error)
	// GetCostStrategyOrDefault returns the cost strategy for the given name, or default if not found
	GetCostStrategyOrDefault(name string) strategy.CostCalculationStrategy
}

// InventoryService handles inventory-related business operations
type InventoryService struct {
	inventoryRepo    inventory.InventoryItemRepository
	batchRepo        inventory.StockBatchRepository
	lockRepo         inventory.StockLockRepository
	transactionRepo  inventory.InventoryTransactionRepository
	tenantRepo       identity.TenantRepository
	strategyProvider CostStrategyProvider
	eventPublisher   shared.EventPublisher
}

// NewInventoryService creates a new InventoryService
func NewInventoryService(
	inventoryRepo inventory.InventoryItemRepository,
	batchRepo inventory.StockBatchRepository,
	lockRepo inventory.StockLockRepository,
	transactionRepo inventory.InventoryTransactionRepository,
) *InventoryService {
	return &InventoryService{
		inventoryRepo:   inventoryRepo,
		batchRepo:       batchRepo,
		lockRepo:        lockRepo,
		transactionRepo: transactionRepo,
	}
}

// NewInventoryServiceWithStrategies creates a new InventoryService with strategy support
func NewInventoryServiceWithStrategies(
	inventoryRepo inventory.InventoryItemRepository,
	batchRepo inventory.StockBatchRepository,
	lockRepo inventory.StockLockRepository,
	transactionRepo inventory.InventoryTransactionRepository,
	tenantRepo identity.TenantRepository,
	strategyProvider CostStrategyProvider,
) *InventoryService {
	return &InventoryService{
		inventoryRepo:    inventoryRepo,
		batchRepo:        batchRepo,
		lockRepo:         lockRepo,
		transactionRepo:  transactionRepo,
		tenantRepo:       tenantRepo,
		strategyProvider: strategyProvider,
	}
}

// SetEventPublisher sets the event publisher for publishing domain events
func (s *InventoryService) SetEventPublisher(publisher shared.EventPublisher) {
	s.eventPublisher = publisher
}

// SetTenantRepository sets the tenant repository (optional, for strategy lookup)
func (s *InventoryService) SetTenantRepository(repo identity.TenantRepository) {
	s.tenantRepo = repo
}

// SetStrategyProvider sets the strategy provider (optional, for cost calculation)
func (s *InventoryService) SetStrategyProvider(provider CostStrategyProvider) {
	s.strategyProvider = provider
}

// getCostStrategyForTenant returns the cost calculation strategy based on tenant configuration.
// It looks up the tenant's configured cost strategy (e.g., "fifo", "weighted_average")
// and returns the corresponding strategy from the registry.
// Returns nil if no strategy provider is configured or tenant config lookup fails.
func (s *InventoryService) getCostStrategyForTenant(ctx context.Context, tenantID uuid.UUID) strategy.CostCalculationStrategy {
	// If no strategy provider, return nil to use fallback
	if s.strategyProvider == nil {
		return nil
	}

	// Try to get tenant configuration
	strategyName := "moving_average" // default
	if s.tenantRepo != nil {
		tenant, err := s.tenantRepo.FindByID(ctx, tenantID)
		if err == nil && tenant != nil && tenant.Config.CostStrategy != "" {
			// Map tenant config values to strategy names
			// Tenant config may use "weighted_average" but strategy is "moving_average"
			switch tenant.Config.CostStrategy {
			case "weighted_average", "moving_average":
				strategyName = "moving_average"
			case "fifo":
				strategyName = "fifo"
			default:
				strategyName = tenant.Config.CostStrategy
			}
		}
	}

	return s.strategyProvider.GetCostStrategyOrDefault(strategyName)
}

// publishDomainEvents publishes all domain events from the inventory item
func (s *InventoryService) publishDomainEvents(ctx context.Context, item *inventory.InventoryItem) {
	if s.eventPublisher == nil {
		return
	}
	events := item.GetDomainEvents()
	if len(events) == 0 {
		return
	}
	// Publish events (errors are logged by the event bus, not propagated)
	_ = s.eventPublisher.Publish(ctx, events...)
	// Clear events after publishing
	item.ClearDomainEvents()
}

// GetByID retrieves an inventory item by ID
func (s *InventoryService) GetByID(ctx context.Context, tenantID, itemID uuid.UUID) (*InventoryItemResponse, error) {
	item, err := s.inventoryRepo.FindByIDForTenant(ctx, tenantID, itemID)
	if err != nil {
		return nil, err
	}
	response := ToInventoryItemResponse(item)
	return &response, nil
}

// GetByWarehouseAndProduct retrieves inventory for a specific warehouse-product combination
func (s *InventoryService) GetByWarehouseAndProduct(ctx context.Context, tenantID, warehouseID, productID uuid.UUID) (*InventoryItemResponse, error) {
	item, err := s.inventoryRepo.FindByWarehouseAndProduct(ctx, tenantID, warehouseID, productID)
	if err != nil {
		return nil, err
	}
	response := ToInventoryItemResponse(item)
	return &response, nil
}

// List retrieves a list of inventory items with filtering and pagination
func (s *InventoryService) List(ctx context.Context, tenantID uuid.UUID, filter InventoryListFilter) ([]InventoryListItemResponse, int64, error) {
	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.OrderBy == "" {
		filter.OrderBy = "updated_at"
	}
	if filter.OrderDir == "" {
		filter.OrderDir = "desc"
	}

	// Build domain filter
	domainFilter := shared.Filter{
		Page:     filter.Page,
		PageSize: filter.PageSize,
		OrderBy:  filter.OrderBy,
		OrderDir: filter.OrderDir,
		Search:   filter.Search,
		Filters:  make(map[string]interface{}),
	}

	// Add specific filters
	if filter.WarehouseID != nil {
		domainFilter.Filters["warehouse_id"] = *filter.WarehouseID
	}
	if filter.ProductID != nil {
		domainFilter.Filters["product_id"] = *filter.ProductID
	}
	if filter.BelowMinimum != nil && *filter.BelowMinimum {
		domainFilter.Filters["below_minimum"] = true
	}
	if filter.HasStock != nil {
		if *filter.HasStock {
			domainFilter.Filters["has_stock"] = true
		} else {
			domainFilter.Filters["no_stock"] = true
		}
	}
	if filter.MinQuantity != nil {
		domainFilter.Filters["min_quantity"] = *filter.MinQuantity
	}
	if filter.MaxQuantity != nil {
		domainFilter.Filters["max_quantity"] = *filter.MaxQuantity
	}

	// Get items
	items, err := s.inventoryRepo.FindAllForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	// Get total count
	total, err := s.inventoryRepo.CountForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	return ToInventoryListItemResponses(items), total, nil
}

// ListByWarehouse retrieves inventory items for a specific warehouse
func (s *InventoryService) ListByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID, filter InventoryListFilter) ([]InventoryListItemResponse, int64, error) {
	filter.WarehouseID = &warehouseID
	return s.List(ctx, tenantID, filter)
}

// ListByProduct retrieves inventory items for a specific product (across all warehouses)
func (s *InventoryService) ListByProduct(ctx context.Context, tenantID, productID uuid.UUID, filter InventoryListFilter) ([]InventoryListItemResponse, int64, error) {
	filter.ProductID = &productID
	return s.List(ctx, tenantID, filter)
}

// ListBelowMinimum retrieves inventory items below their minimum threshold
func (s *InventoryService) ListBelowMinimum(ctx context.Context, tenantID uuid.UUID, filter InventoryListFilter) ([]InventoryListItemResponse, int64, error) {
	belowMin := true
	filter.BelowMinimum = &belowMin
	return s.List(ctx, tenantID, filter)
}

// GetTotalQuantityByProduct returns total quantity for a product across all warehouses
func (s *InventoryService) GetTotalQuantityByProduct(ctx context.Context, tenantID, productID uuid.UUID) (decimal.Decimal, error) {
	return s.inventoryRepo.SumQuantityByProduct(ctx, tenantID, productID)
}

// GetTotalValueByWarehouse returns total inventory value for a warehouse
func (s *InventoryService) GetTotalValueByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID) (decimal.Decimal, error) {
	return s.inventoryRepo.SumValueByWarehouse(ctx, tenantID, warehouseID)
}

// IncreaseStock increases stock for a warehouse-product combination
func (s *InventoryService) IncreaseStock(ctx context.Context, tenantID uuid.UUID, req IncreaseStockRequest) (*InventoryItemResponse, error) {
	// Validate source type
	sourceType := inventory.SourceType(req.SourceType)
	if !sourceType.IsValid() {
		return nil, shared.NewDomainError("INVALID_SOURCE_TYPE", "Invalid source type")
	}

	// Get or create inventory item
	item, err := s.inventoryRepo.GetOrCreate(ctx, tenantID, req.WarehouseID, req.ProductID)
	if err != nil {
		return nil, err
	}

	// Record balance before
	balanceBefore := item.AvailableQuantity

	// Prepare batch info if provided
	var batchInfo *inventory.BatchInfo
	if req.BatchNumber != "" {
		batchInfo = inventory.NewBatchInfo(req.BatchNumber, nil, req.ExpiryDate)
	}

	// Get cost strategy from tenant configuration
	costStrategy := s.getCostStrategyForTenant(ctx, tenantID)
	costMethod := string(strategy.CostMethodMovingAverage) // default

	// Use domain service with injected strategy if available
	if costStrategy != nil {
		domainService := inventory.NewInventoryDomainService(costStrategy, nil)
		unitCostMoney := valueobject.NewMoneyCNY(req.UnitCost)

		result, err := domainService.StockIn(ctx, item, req.Quantity, unitCostMoney, batchInfo)
		if err != nil {
			return nil, err
		}
		costMethod = string(result.CostMethod)
	} else {
		// Fallback to legacy method (uses built-in moving average)
		unitCostMoney := valueobject.NewMoneyCNY(req.UnitCost)
		if err := item.IncreaseStock(req.Quantity, unitCostMoney, batchInfo); err != nil {
			return nil, err
		}
	}

	// Save with optimistic locking
	if err := s.inventoryRepo.SaveWithLock(ctx, item); err != nil {
		return nil, err
	}

	// Publish domain events (including StockBelowThreshold if applicable)
	s.publishDomainEvents(ctx, item)

	// Create transaction record
	tx, err := inventory.CreateInboundTransaction(
		tenantID,
		item.ID,
		req.WarehouseID,
		req.ProductID,
		req.Quantity,
		item.UnitCost, // Use the calculated unit cost
		balanceBefore,
		item.AvailableQuantity,
		sourceType,
		req.SourceID,
	)
	if err != nil {
		return nil, err
	}

	// Set optional fields
	if req.Reference != "" {
		tx.WithReference(req.Reference)
	}
	if req.Reason != "" {
		tx.WithReason(req.Reason)
	}
	if req.OperatorID != nil {
		tx.WithOperatorID(*req.OperatorID)
	}

	// Record the cost calculation method used
	tx.WithCostMethod(costMethod)

	if err := s.transactionRepo.Create(ctx, tx); err != nil {
		// Log error but don't fail the operation - transaction is for audit only
		// In production, this would be handled by the outbox pattern
		_ = err
	}

	response := ToInventoryItemResponse(item)
	return &response, nil
}

// LockStock locks stock for a pending order
func (s *InventoryService) LockStock(ctx context.Context, tenantID uuid.UUID, req LockStockRequest) (*LockStockResponse, error) {
	// Get inventory item
	item, err := s.inventoryRepo.FindByWarehouseAndProduct(ctx, tenantID, req.WarehouseID, req.ProductID)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, shared.NewDomainError("NO_INVENTORY", "No inventory found for this warehouse-product combination")
		}
		return nil, err
	}

	// Set expiry time
	expireAt := time.Now().Add(DefaultLockExpiry)
	if req.ExpireAt != nil {
		expireAt = *req.ExpireAt
	}

	// Record balance before
	balanceBefore := item.AvailableQuantity

	// Lock stock
	lock, err := item.LockStock(req.Quantity, req.SourceType, req.SourceID, expireAt)
	if err != nil {
		return nil, err
	}

	// Save with optimistic locking
	if err := s.inventoryRepo.SaveWithLock(ctx, item); err != nil {
		return nil, err
	}

	// Publish domain events
	s.publishDomainEvents(ctx, item)

	// Save the lock
	if err := s.lockRepo.Save(ctx, lock); err != nil {
		return nil, err
	}

	// Create transaction record for the lock
	tx, err := inventory.NewInventoryTransaction(
		tenantID,
		item.ID,
		req.WarehouseID,
		req.ProductID,
		inventory.TransactionTypeLock,
		req.Quantity,
		item.UnitCost,
		balanceBefore,
		item.AvailableQuantity,
		inventory.SourceType(req.SourceType),
		req.SourceID,
	)
	if err == nil {
		tx.WithLockID(lock.ID)
		_ = s.transactionRepo.Create(ctx, tx)
	}

	return &LockStockResponse{
		LockID:          lock.ID,
		InventoryItemID: item.ID,
		WarehouseID:     req.WarehouseID,
		ProductID:       req.ProductID,
		Quantity:        req.Quantity,
		ExpireAt:        expireAt,
		SourceType:      req.SourceType,
		SourceID:        req.SourceID,
	}, nil
}

// UnlockStock releases a previously locked quantity back to available
func (s *InventoryService) UnlockStock(ctx context.Context, tenantID uuid.UUID, req UnlockStockRequest) error {
	// Find the lock
	lock, err := s.lockRepo.FindByID(ctx, req.LockID)
	if err != nil {
		return err
	}

	// Get the inventory item
	item, err := s.inventoryRepo.FindByID(ctx, lock.InventoryItemID)
	if err != nil {
		return err
	}

	// Verify tenant
	if item.TenantID != tenantID {
		return shared.NewDomainError("FORBIDDEN", "Lock does not belong to this tenant")
	}

	// Add lock to item's Locks slice so domain method can find it
	// (Repository doesn't preload associations)
	item.Locks = append(item.Locks, *lock)

	// Record balance before
	balanceBefore := item.AvailableQuantity

	// Unlock stock
	if err := item.UnlockStock(req.LockID); err != nil {
		return err
	}

	// Save with optimistic locking
	if err := s.inventoryRepo.SaveWithLock(ctx, item); err != nil {
		return err
	}

	// Publish domain events
	s.publishDomainEvents(ctx, item)

	// Update the lock record (use the updated lock from item.Locks, not the original)
	// The domain method marks the lock as Released in item.Locks
	if err := s.lockRepo.Save(ctx, &item.Locks[0]); err != nil {
		return err
	}

	// Create transaction record for the unlock
	tx, err := inventory.NewInventoryTransaction(
		tenantID,
		item.ID,
		item.WarehouseID,
		item.ProductID,
		inventory.TransactionTypeUnlock,
		lock.Quantity,
		item.UnitCost,
		balanceBefore,
		item.AvailableQuantity,
		inventory.SourceType(lock.SourceType),
		lock.SourceID,
	)
	if err == nil {
		tx.WithLockID(lock.ID)
		_ = s.transactionRepo.Create(ctx, tx)
	}

	return nil
}

// DeductStock deducts locked stock (actual shipment/consumption)
func (s *InventoryService) DeductStock(ctx context.Context, tenantID uuid.UUID, req DeductStockRequest) error {
	// Validate source type
	sourceType := inventory.SourceType(req.SourceType)
	if !sourceType.IsValid() {
		return shared.NewDomainError("INVALID_SOURCE_TYPE", "Invalid source type")
	}

	// Find the lock
	lock, err := s.lockRepo.FindByID(ctx, req.LockID)
	if err != nil {
		return err
	}

	// Get the inventory item
	item, err := s.inventoryRepo.FindByID(ctx, lock.InventoryItemID)
	if err != nil {
		return err
	}

	// Verify tenant
	if item.TenantID != tenantID {
		return shared.NewDomainError("FORBIDDEN", "Lock does not belong to this tenant")
	}

	// Add lock to item's Locks slice so domain method can find it
	// (Repository doesn't preload associations)
	item.Locks = append(item.Locks, *lock)

	// Record locked quantity before (deduct affects locked, not available)
	lockedBefore := item.LockedQuantity

	// Deduct stock
	if err := item.DeductStock(req.LockID); err != nil {
		return err
	}

	// Save with optimistic locking
	if err := s.inventoryRepo.SaveWithLock(ctx, item); err != nil {
		return err
	}

	// Publish domain events (including StockBelowThreshold if applicable)
	s.publishDomainEvents(ctx, item)

	// Update the lock record (use the updated lock from item.Locks, not the original)
	// The domain method marks the lock as Consumed in item.Locks
	if err := s.lockRepo.Save(ctx, &item.Locks[0]); err != nil {
		return err
	}

	// Create transaction record for the deduction (outbound)
	tx, err := inventory.CreateOutboundTransaction(
		tenantID,
		item.ID,
		item.WarehouseID,
		item.ProductID,
		lock.Quantity,
		item.UnitCost,
		lockedBefore,
		item.LockedQuantity,
		sourceType,
		req.SourceID,
	)
	if err == nil {
		tx.WithLockID(lock.ID)
		if req.Reference != "" {
			tx.WithReference(req.Reference)
		}
		if req.OperatorID != nil {
			tx.WithOperatorID(*req.OperatorID)
		}
		_ = s.transactionRepo.Create(ctx, tx)
	}

	return nil
}

// DecreaseStock directly decreases available stock (without requiring a prior lock)
// This is used for operations like purchase returns where goods are shipped back to supplier
func (s *InventoryService) DecreaseStock(ctx context.Context, tenantID uuid.UUID, req DecreaseStockRequest) error {
	// Validate source type
	sourceType := inventory.SourceType(req.SourceType)
	if !sourceType.IsValid() {
		return shared.NewDomainError("INVALID_SOURCE_TYPE", "Invalid source type")
	}

	// Get inventory item
	item, err := s.inventoryRepo.FindByWarehouseAndProduct(ctx, tenantID, req.WarehouseID, req.ProductID)
	if err != nil {
		return err
	}

	// Record balance before
	balanceBefore := item.AvailableQuantity

	// Decrease stock
	if err := item.DecreaseStock(req.Quantity, req.SourceType, req.SourceID, req.Reason); err != nil {
		return err
	}

	// Save with optimistic locking
	if err := s.inventoryRepo.SaveWithLock(ctx, item); err != nil {
		return err
	}

	// Publish domain events (including StockBelowThreshold if applicable)
	s.publishDomainEvents(ctx, item)

	// Create transaction record for the decrease (outbound)
	tx, err := inventory.CreateOutboundTransaction(
		tenantID,
		item.ID,
		item.WarehouseID,
		item.ProductID,
		req.Quantity,
		item.UnitCost,
		balanceBefore,
		item.AvailableQuantity,
		sourceType,
		req.SourceID,
	)
	if err == nil {
		if req.Reference != "" {
			tx.WithReference(req.Reference)
		}
		if req.Reason != "" {
			tx.WithReason(req.Reason)
		}
		if req.OperatorID != nil {
			tx.WithOperatorID(*req.OperatorID)
		}
		_ = s.transactionRepo.Create(ctx, tx)
	}

	return nil
}

// AdjustStock adjusts the stock to match actual quantity
func (s *InventoryService) AdjustStock(ctx context.Context, tenantID uuid.UUID, req AdjustStockRequest) (*InventoryItemResponse, error) {
	// Get or create inventory item
	item, err := s.inventoryRepo.GetOrCreate(ctx, tenantID, req.WarehouseID, req.ProductID)
	if err != nil {
		return nil, err
	}

	// Record balance before
	balanceBefore := item.AvailableQuantity

	// Adjust stock
	if err := item.AdjustStock(req.ActualQuantity, req.Reason); err != nil {
		return nil, err
	}

	// Save with optimistic locking
	if err := s.inventoryRepo.SaveWithLock(ctx, item); err != nil {
		return nil, err
	}

	// Publish domain events (including StockBelowThreshold if applicable)
	s.publishDomainEvents(ctx, item)

	// Determine source type and ID
	sourceType := inventory.SourceTypeManualAdjustment
	if req.SourceType != "" {
		st := inventory.SourceType(req.SourceType)
		if st.IsValid() {
			sourceType = st
		}
	}
	sourceID := req.SourceID
	if sourceID == "" {
		sourceID = fmt.Sprintf("ADJ-%s", time.Now().Format("20060102150405"))
	}

	// Calculate adjustment quantity (absolute value)
	adjustmentQty := req.ActualQuantity.Sub(balanceBefore).Abs()
	if adjustmentQty.IsZero() {
		// No actual change
		response := ToInventoryItemResponse(item)
		return &response, nil
	}

	// Create transaction record
	tx, err := inventory.CreateAdjustmentTransaction(
		tenantID,
		item.ID,
		req.WarehouseID,
		req.ProductID,
		adjustmentQty,
		item.UnitCost,
		balanceBefore,
		item.AvailableQuantity,
		sourceType,
		sourceID,
		req.Reason,
	)
	if err == nil {
		if req.OperatorID != nil {
			tx.WithOperatorID(*req.OperatorID)
		}
		_ = s.transactionRepo.Create(ctx, tx)
	}

	response := ToInventoryItemResponse(item)
	return &response, nil
}

// SetThresholds sets min/max quantity thresholds for an inventory item
func (s *InventoryService) SetThresholds(ctx context.Context, tenantID uuid.UUID, req SetThresholdsRequest) (*InventoryItemResponse, error) {
	// Get or create inventory item
	item, err := s.inventoryRepo.GetOrCreate(ctx, tenantID, req.WarehouseID, req.ProductID)
	if err != nil {
		return nil, err
	}

	// Set thresholds
	if req.MinQuantity != nil {
		if err := item.SetMinQuantity(*req.MinQuantity); err != nil {
			return nil, err
		}
	}
	if req.MaxQuantity != nil {
		if err := item.SetMaxQuantity(*req.MaxQuantity); err != nil {
			return nil, err
		}
	}

	// Save
	if err := s.inventoryRepo.Save(ctx, item); err != nil {
		return nil, err
	}

	response := ToInventoryItemResponse(item)
	return &response, nil
}

// GetActiveLocks retrieves all active locks for an inventory item
func (s *InventoryService) GetActiveLocks(ctx context.Context, tenantID uuid.UUID, warehouseID, productID uuid.UUID) ([]StockLockResponse, error) {
	// Get inventory item
	item, err := s.inventoryRepo.FindByWarehouseAndProduct(ctx, tenantID, warehouseID, productID)
	if err != nil {
		return nil, err
	}

	// Get active locks
	locks, err := s.lockRepo.FindActive(ctx, item.ID)
	if err != nil {
		return nil, err
	}

	return ToStockLockResponses(locks), nil
}

// GetLockByID retrieves a specific lock
func (s *InventoryService) GetLockByID(ctx context.Context, tenantID uuid.UUID, lockID uuid.UUID) (*StockLockResponse, error) {
	lock, err := s.lockRepo.FindByID(ctx, lockID)
	if err != nil {
		return nil, err
	}

	// Verify tenant by checking the inventory item
	item, err := s.inventoryRepo.FindByID(ctx, lock.InventoryItemID)
	if err != nil {
		return nil, err
	}
	if item.TenantID != tenantID {
		return nil, shared.NewDomainError("FORBIDDEN", "Lock does not belong to this tenant")
	}

	response := ToStockLockResponse(lock)
	return &response, nil
}

// ReleaseExpiredLocks releases all expired locks
func (s *InventoryService) ReleaseExpiredLocks(ctx context.Context) (int, error) {
	// Find expired locks
	expiredLocks, err := s.lockRepo.FindExpired(ctx)
	if err != nil {
		return 0, err
	}

	count := 0
	for i := range expiredLocks {
		lock := &expiredLocks[i]

		// Get inventory item
		item, err := s.inventoryRepo.FindByID(ctx, lock.InventoryItemID)
		if err != nil {
			// Item may have been deleted; skip this lock
			continue
		}

		// Add lock to item.Locks so UnlockStock can find it
		// (FindByID does not preload locks association)
		item.Locks = append(item.Locks, *lock)

		// Unlock - this will find the lock in item.Locks and mark it as released
		if err := item.UnlockStock(lock.ID); err != nil {
			continue
		}

		// Save inventory item with updated quantities
		if err := s.inventoryRepo.SaveWithLock(ctx, item); err != nil {
			continue
		}

		// Publish domain events
		s.publishDomainEvents(ctx, item)

		// Find the updated lock by ID (safer than assuming position)
		var releasedLock *inventory.StockLock
		for idx := range item.Locks {
			if item.Locks[idx].ID == lock.ID {
				releasedLock = &item.Locks[idx]
				break
			}
		}
		if releasedLock == nil {
			continue
		}
		if err := s.lockRepo.Save(ctx, releasedLock); err != nil {
			continue
		}

		count++
	}

	return count, nil
}

// ListTransactions retrieves inventory transactions with filtering
func (s *InventoryService) ListTransactions(ctx context.Context, tenantID uuid.UUID, filter TransactionListFilter) ([]TransactionResponse, int64, error) {
	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.OrderBy == "" {
		filter.OrderBy = "transaction_date"
	}
	if filter.OrderDir == "" {
		filter.OrderDir = "desc"
	}

	// Build domain filter
	domainFilter := shared.Filter{
		Page:     filter.Page,
		PageSize: filter.PageSize,
		OrderBy:  filter.OrderBy,
		OrderDir: filter.OrderDir,
		Filters:  make(map[string]interface{}),
	}

	// Add specific filters
	if filter.WarehouseID != nil {
		domainFilter.Filters["warehouse_id"] = *filter.WarehouseID
	}
	if filter.ProductID != nil {
		domainFilter.Filters["product_id"] = *filter.ProductID
	}
	if filter.TransactionType != "" {
		domainFilter.Filters["transaction_type"] = filter.TransactionType
	}
	if filter.SourceType != "" {
		domainFilter.Filters["source_type"] = filter.SourceType
	}
	if filter.SourceID != "" {
		domainFilter.Filters["source_id"] = filter.SourceID
	}
	if filter.StartDate != nil {
		domainFilter.Filters["start_date"] = *filter.StartDate
	}
	if filter.EndDate != nil {
		domainFilter.Filters["end_date"] = *filter.EndDate
	}

	// Get transactions
	txs, err := s.transactionRepo.FindForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	// Get total count
	total, err := s.transactionRepo.CountForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	return ToTransactionResponses(txs), total, nil
}

// ListTransactionsByInventoryItem retrieves transactions for a specific inventory item
func (s *InventoryService) ListTransactionsByInventoryItem(ctx context.Context, tenantID, itemID uuid.UUID, filter TransactionListFilter) ([]TransactionResponse, int64, error) {
	// Verify item belongs to tenant
	item, err := s.inventoryRepo.FindByIDForTenant(ctx, tenantID, itemID)
	if err != nil {
		return nil, 0, err
	}

	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.OrderBy == "" {
		filter.OrderBy = "transaction_date"
	}
	if filter.OrderDir == "" {
		filter.OrderDir = "desc"
	}

	domainFilter := shared.Filter{
		Page:     filter.Page,
		PageSize: filter.PageSize,
		OrderBy:  filter.OrderBy,
		OrderDir: filter.OrderDir,
	}

	txs, err := s.transactionRepo.FindByInventoryItem(ctx, item.ID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.transactionRepo.CountByInventoryItem(ctx, item.ID)
	if err != nil {
		return nil, 0, err
	}

	return ToTransactionResponses(txs), total, nil
}

// GetTransactionByID retrieves a specific transaction
func (s *InventoryService) GetTransactionByID(ctx context.Context, tenantID, txID uuid.UUID) (*TransactionResponse, error) {
	tx, err := s.transactionRepo.FindByID(ctx, txID)
	if err != nil {
		return nil, err
	}

	// Verify tenant
	if tx.TenantID != tenantID {
		return nil, shared.NewDomainError("FORBIDDEN", "Transaction does not belong to this tenant")
	}

	response := ToTransactionResponse(tx)
	return &response, nil
}

// CheckAvailability checks if a quantity is available for a product in a warehouse
func (s *InventoryService) CheckAvailability(ctx context.Context, tenantID, warehouseID, productID uuid.UUID, quantity decimal.Decimal) (bool, decimal.Decimal, error) {
	item, err := s.inventoryRepo.FindByWarehouseAndProduct(ctx, tenantID, warehouseID, productID)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return false, decimal.Zero, nil
		}
		return false, decimal.Zero, err
	}

	available := item.CanFulfill(quantity)
	return available, item.AvailableQuantity, nil
}

// GetLocksBySource retrieves all locks for a specific source (e.g., sales order)
// This method enriches lock responses with ProductID and WarehouseID from inventory items
func (s *InventoryService) GetLocksBySource(ctx context.Context, sourceType, sourceID string) ([]StockLockResponse, error) {
	locks, err := s.lockRepo.FindBySource(ctx, sourceType, sourceID)
	if err != nil {
		return nil, err
	}

	// Build responses with enriched data
	responses := make([]StockLockResponse, 0, len(locks))
	for _, lock := range locks {
		// Get inventory item to obtain ProductID and WarehouseID
		item, err := s.inventoryRepo.FindByID(ctx, lock.InventoryItemID)
		if err != nil {
			// Log and continue - include lock with missing product info
			responses = append(responses, ToStockLockResponse(&lock))
			continue
		}

		resp := ToStockLockResponse(&lock)
		resp.ProductID = item.ProductID
		resp.WarehouseID = item.WarehouseID
		responses = append(responses, resp)
	}

	return responses, nil
}

// UnlockBySource releases all active locks for a specific source
func (s *InventoryService) UnlockBySource(ctx context.Context, tenantID uuid.UUID, sourceType, sourceID string) (int, error) {
	// Find all locks for this source
	locks, err := s.lockRepo.FindBySource(ctx, sourceType, sourceID)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, lock := range locks {
		// Skip already released or consumed locks
		if lock.Released || lock.Consumed {
			continue
		}

		// Get inventory item
		item, err := s.inventoryRepo.FindByID(ctx, lock.InventoryItemID)
		if err != nil {
			continue
		}

		// Verify tenant
		if item.TenantID != tenantID {
			continue
		}

		// Add lock to item's Locks slice so domain method can find it
		// (Repository doesn't preload associations)
		item.Locks = append(item.Locks, lock)

		// Unlock
		if err := item.UnlockStock(lock.ID); err != nil {
			continue
		}

		// Save inventory item
		if err := s.inventoryRepo.SaveWithLock(ctx, item); err != nil {
			continue
		}

		// Publish domain events
		s.publishDomainEvents(ctx, item)

		// Update lock status (use the updated lock from item.Locks, not the original)
		// The domain method marks the lock as Released in item.Locks
		if err := s.lockRepo.Save(ctx, &item.Locks[0]); err != nil {
			continue
		}

		count++
	}

	return count, nil
}
