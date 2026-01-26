package inventory

import (
	"context"
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// StockAllocationService is a domain service that coordinates stock allocation
// across multiple InventoryItem aggregates. It implements a Saga pattern with
// compensation for handling partial failures in multi-aggregate operations.
//
// Key responsibilities:
// 1. Allocate (lock) stock for multiple items atomically
// 2. Handle partial failures with compensation (rollback successful locks)
// 3. Emit appropriate domain events for monitoring and eventual consistency
// 4. Provide allocation preview without side effects
//
// The service ensures that either all items are successfully locked, or none are
// (with compensation). This provides consistency guarantees for cross-aggregate
// operations while maintaining each aggregate's autonomy.
//
// Usage scenarios:
// - Sales order confirmation: Lock stock for all order items
// - Transfer orders: Lock stock at source warehouse
// - Stock reservations: Reserve stock for future fulfillment
type StockAllocationService struct {
	defaultLockDuration time.Duration
}

// StockAllocationServiceOption is a functional option for configuring StockAllocationService
type StockAllocationServiceOption func(*StockAllocationService)

// WithDefaultLockDuration sets the default lock duration for stock allocations
func WithDefaultLockDuration(duration time.Duration) StockAllocationServiceOption {
	return func(s *StockAllocationService) {
		if duration > 0 {
			s.defaultLockDuration = duration
		}
	}
}

// NewStockAllocationService creates a new stock allocation service
func NewStockAllocationService(opts ...StockAllocationServiceOption) *StockAllocationService {
	s := &StockAllocationService{
		defaultLockDuration: 30 * time.Minute, // Default: 30 minutes
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// GetDefaultLockDuration returns the default lock duration
func (s *StockAllocationService) GetDefaultLockDuration() time.Duration {
	return s.defaultLockDuration
}

// AllocationItem represents a single item to be allocated
type AllocationItem struct {
	InventoryItem *InventoryItem  // The inventory item to allocate from
	Quantity      decimal.Decimal // Quantity to allocate
}

// AllocationRequest represents a request to allocate stock for multiple items
type AllocationRequest struct {
	Items        []AllocationItem // Items to allocate
	SourceType   string           // Source document type (e.g., "SALES_ORDER", "TRANSFER")
	SourceID     string           // Source document ID
	LockDuration time.Duration    // Optional: override default lock duration
}

// Validate validates the allocation request
func (r *AllocationRequest) Validate() error {
	if len(r.Items) == 0 {
		return shared.NewDomainError("INVALID_REQUEST", "At least one item is required for allocation")
	}
	if r.SourceType == "" {
		return shared.NewDomainError("INVALID_SOURCE_TYPE", "Source type is required")
	}
	if r.SourceID == "" {
		return shared.NewDomainError("INVALID_SOURCE_ID", "Source ID is required")
	}

	for i, item := range r.Items {
		if item.InventoryItem == nil {
			return shared.NewDomainError("INVALID_ITEM", fmt.Sprintf("Inventory item at index %d is nil", i))
		}
		if item.Quantity.LessThanOrEqual(decimal.Zero) {
			return shared.NewDomainError("INVALID_QUANTITY",
				fmt.Sprintf("Quantity at index %d must be positive", i))
		}
	}

	return nil
}

// AllocationItemResult represents the result of allocating a single item
type AllocationItemResult struct {
	InventoryItemID uuid.UUID       `json:"inventory_item_id"`
	WarehouseID     uuid.UUID       `json:"warehouse_id"`
	ProductID       uuid.UUID       `json:"product_id"`
	Quantity        decimal.Decimal `json:"quantity"`
	LockID          uuid.UUID       `json:"lock_id"`
	ExpireAt        time.Time       `json:"expire_at"`
	Success         bool            `json:"success"`
	Error           error           `json:"-"` // Error if allocation failed
	ErrorMessage    string          `json:"error_message,omitempty"`
}

// AllocationResult represents the result of a multi-item stock allocation
type AllocationResult struct {
	// CorrelationID is a unique ID for tracking this allocation operation
	CorrelationID uuid.UUID `json:"correlation_id"`

	// SourceType is the source document type (e.g., "SALES_ORDER")
	SourceType string `json:"source_type"`

	// SourceID is the source document ID
	SourceID string `json:"source_id"`

	// Items contains results for each allocation item
	Items []AllocationItemResult `json:"items"`

	// TotalRequested is the total quantity requested across all items
	TotalRequested decimal.Decimal `json:"total_requested"`

	// TotalAllocated is the total quantity successfully allocated
	TotalAllocated decimal.Decimal `json:"total_allocated"`

	// Success is true if all items were successfully allocated
	Success bool `json:"success"`

	// PartialSuccess is true if some (but not all) items were allocated
	PartialSuccess bool `json:"partial_success"`

	// Compensated is true if compensation was applied (rollback of successful locks)
	Compensated bool `json:"compensated"`

	// CompensationResults contains results of compensation operations
	CompensationResults []CompensationResult `json:"compensation_results,omitempty"`

	// FailedItems contains indices of items that failed to allocate
	FailedItems []int `json:"failed_items,omitempty"`

	// DomainEvents contains events generated during the operation
	DomainEvents []shared.DomainEvent `json:"-"`
}

// GetSuccessfulLocks returns only the successful lock results
func (r *AllocationResult) GetSuccessfulLocks() []AllocationItemResult {
	successful := make([]AllocationItemResult, 0)
	for _, item := range r.Items {
		if item.Success {
			successful = append(successful, item)
		}
	}
	return successful
}

// CompensationResult represents the result of a compensation (rollback) operation
type CompensationResult struct {
	InventoryItemID uuid.UUID `json:"inventory_item_id"`
	LockID          uuid.UUID `json:"lock_id"`
	Success         bool      `json:"success"`
	Error           error     `json:"-"`
	ErrorMessage    string    `json:"error_message,omitempty"`
}

// AllocateStock attempts to allocate (lock) stock for all items in the request.
// It implements the Saga pattern:
// 1. Try to lock each item sequentially
// 2. If any lock fails, compensate by unlocking all previously successful locks
// 3. Return a result indicating success, partial success, or failure
//
// The method modifies the InventoryItem aggregates in-place but does NOT persist them.
// The caller is responsible for:
// 1. Retrieving InventoryItems from the repository
// 2. Calling this service method
// 3. Persisting the modified InventoryItems (if successful)
// 4. Publishing the domain events
//
// This design keeps the domain service focused on business logic while allowing
// the application layer to control transaction boundaries.
func (s *StockAllocationService) AllocateStock(
	ctx context.Context,
	req AllocationRequest,
) (*AllocationResult, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Create correlation ID for tracking
	correlationID := uuid.New()

	// Determine lock duration
	lockDuration := s.defaultLockDuration
	if req.LockDuration > 0 {
		lockDuration = req.LockDuration
	}
	expireAt := time.Now().Add(lockDuration)

	// Initialize result
	result := &AllocationResult{
		CorrelationID:  correlationID,
		SourceType:     req.SourceType,
		SourceID:       req.SourceID,
		Items:          make([]AllocationItemResult, len(req.Items)),
		TotalRequested: decimal.Zero,
		TotalAllocated: decimal.Zero,
		DomainEvents:   make([]shared.DomainEvent, 0),
	}

	// Calculate total requested
	for _, item := range req.Items {
		result.TotalRequested = result.TotalRequested.Add(item.Quantity)
	}

	// Track successful locks for potential compensation
	successfulLocks := make([]struct {
		index int
		item  *InventoryItem
		lock  *StockLock
	}, 0)

	// Try to allocate each item
	for i, item := range req.Items {
		itemResult := AllocationItemResult{
			InventoryItemID: item.InventoryItem.ID,
			WarehouseID:     item.InventoryItem.WarehouseID,
			ProductID:       item.InventoryItem.ProductID,
			Quantity:        item.Quantity,
		}

		// Check if we have enough available stock
		if !item.InventoryItem.CanFulfill(item.Quantity) {
			itemResult.Success = false
			itemResult.Error = shared.NewDomainError("INSUFFICIENT_STOCK",
				fmt.Sprintf("Insufficient stock: available=%s, requested=%s",
					item.InventoryItem.AvailableQuantity.String(), item.Quantity.String()))
			itemResult.ErrorMessage = itemResult.Error.Error()
			result.Items[i] = itemResult
			result.FailedItems = append(result.FailedItems, i)
			continue
		}

		// Try to lock the stock
		lock, err := item.InventoryItem.LockStock(item.Quantity, req.SourceType, req.SourceID, expireAt)
		if err != nil {
			itemResult.Success = false
			itemResult.Error = err
			itemResult.ErrorMessage = err.Error()
			result.Items[i] = itemResult
			result.FailedItems = append(result.FailedItems, i)
			continue
		}

		// Success - record the lock
		itemResult.Success = true
		itemResult.LockID = lock.ID
		itemResult.ExpireAt = expireAt
		result.Items[i] = itemResult
		result.TotalAllocated = result.TotalAllocated.Add(item.Quantity)

		successfulLocks = append(successfulLocks, struct {
			index int
			item  *InventoryItem
			lock  *StockLock
		}{index: i, item: item.InventoryItem, lock: lock})
	}

	// Determine overall success status
	if len(result.FailedItems) == 0 {
		// All items allocated successfully
		result.Success = true
		result.PartialSuccess = false

		// Emit success event
		result.DomainEvents = append(result.DomainEvents,
			NewStockAllocationCompletedEvent(correlationID, req.SourceType, req.SourceID, result.Items))
	} else if len(successfulLocks) > 0 {
		// Partial allocation - need to compensate
		result.PartialSuccess = true

		// Emit partial allocation event
		result.DomainEvents = append(result.DomainEvents,
			NewStockAllocationPartialEvent(correlationID, req.SourceType, req.SourceID, result.Items, result.FailedItems))

		// Apply compensation (rollback successful locks)
		result.CompensationResults = s.compensate(successfulLocks)
		result.Compensated = true

		// Emit compensation event
		result.DomainEvents = append(result.DomainEvents,
			NewStockAllocationCompensatedEvent(correlationID, req.SourceType, req.SourceID, result.CompensationResults))
	} else {
		// Complete failure
		result.Success = false

		// Emit failure event
		result.DomainEvents = append(result.DomainEvents,
			NewStockAllocationFailedEvent(correlationID, req.SourceType, req.SourceID, result.Items, result.FailedItems))
	}

	return result, nil
}

// compensate rolls back successful locks when allocation fails
func (s *StockAllocationService) compensate(locks []struct {
	index int
	item  *InventoryItem
	lock  *StockLock
}) []CompensationResult {
	results := make([]CompensationResult, len(locks))

	for i, l := range locks {
		result := CompensationResult{
			InventoryItemID: l.item.ID,
			LockID:          l.lock.ID,
		}

		err := l.item.UnlockStock(l.lock.ID)
		if err != nil {
			result.Success = false
			result.Error = err
			result.ErrorMessage = err.Error()
		} else {
			result.Success = true
		}

		results[i] = result
	}

	return results
}

// PreviewAllocation calculates what allocations would be made without making any changes.
// This is useful for showing the user what would happen before confirming an order.
func (s *StockAllocationService) PreviewAllocation(
	_ context.Context,
	req AllocationRequest,
) (*AllocationPreview, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	preview := &AllocationPreview{
		SourceType:     req.SourceType,
		SourceID:       req.SourceID,
		Items:          make([]AllocationPreviewItem, len(req.Items)),
		TotalRequested: decimal.Zero,
		TotalAvailable: decimal.Zero,
		CanFulfillAll:  true,
	}

	for i, item := range req.Items {
		previewItem := AllocationPreviewItem{
			InventoryItemID:   item.InventoryItem.ID,
			WarehouseID:       item.InventoryItem.WarehouseID,
			ProductID:         item.InventoryItem.ProductID,
			RequestedQuantity: item.Quantity,
			AvailableQuantity: item.InventoryItem.AvailableQuantity.Amount(),
			CanFulfill:        item.InventoryItem.CanFulfill(item.Quantity),
		}

		if !previewItem.CanFulfill {
			previewItem.ShortageQuantity = item.Quantity.Sub(item.InventoryItem.AvailableQuantity.Amount())
			preview.CanFulfillAll = false
			preview.ShortageItems = append(preview.ShortageItems, i)
		}

		preview.Items[i] = previewItem
		preview.TotalRequested = preview.TotalRequested.Add(item.Quantity)
		preview.TotalAvailable = preview.TotalAvailable.Add(item.InventoryItem.AvailableQuantity.Amount())
	}

	return preview, nil
}

// AllocationPreview represents a preview of what allocation would look like
type AllocationPreview struct {
	SourceType     string                  `json:"source_type"`
	SourceID       string                  `json:"source_id"`
	Items          []AllocationPreviewItem `json:"items"`
	TotalRequested decimal.Decimal         `json:"total_requested"`
	TotalAvailable decimal.Decimal         `json:"total_available"`
	CanFulfillAll  bool                    `json:"can_fulfill_all"`
	ShortageItems  []int                   `json:"shortage_items,omitempty"`
}

// AllocationPreviewItem represents a preview for a single item
type AllocationPreviewItem struct {
	InventoryItemID   uuid.UUID       `json:"inventory_item_id"`
	WarehouseID       uuid.UUID       `json:"warehouse_id"`
	ProductID         uuid.UUID       `json:"product_id"`
	RequestedQuantity decimal.Decimal `json:"requested_quantity"`
	AvailableQuantity decimal.Decimal `json:"available_quantity"`
	CanFulfill        bool            `json:"can_fulfill"`
	ShortageQuantity  decimal.Decimal `json:"shortage_quantity,omitempty"`
}

// ReleaseAllocation releases (unlocks) previously allocated stock.
// This is used when an order is cancelled or when locks need to be manually released.
func (s *StockAllocationService) ReleaseAllocation(
	ctx context.Context,
	items []*InventoryItem,
	sourceType, sourceID string,
) (*ReleaseResult, error) {
	if len(items) == 0 {
		return nil, shared.NewDomainError("INVALID_REQUEST", "At least one item is required for release")
	}
	if sourceType == "" || sourceID == "" {
		return nil, shared.NewDomainError("INVALID_SOURCE", "Source type and ID are required")
	}

	correlationID := uuid.New()
	result := &ReleaseResult{
		CorrelationID: correlationID,
		SourceType:    sourceType,
		SourceID:      sourceID,
		Items:         make([]ReleaseItemResult, 0),
		DomainEvents:  make([]shared.DomainEvent, 0),
	}

	for _, item := range items {
		// Find active locks for this source
		for _, lock := range item.GetActiveLocks() {
			if lock.SourceType == sourceType && lock.SourceID == sourceID {
				itemResult := ReleaseItemResult{
					InventoryItemID: item.ID,
					LockID:          lock.ID,
					Quantity:        lock.Quantity,
				}

				err := item.UnlockStock(lock.ID)
				if err != nil {
					itemResult.Success = false
					itemResult.Error = err
					itemResult.ErrorMessage = err.Error()
				} else {
					itemResult.Success = true
					result.TotalReleased = result.TotalReleased.Add(lock.Quantity)
				}

				result.Items = append(result.Items, itemResult)
			}
		}
	}

	// Determine success
	result.Success = true
	for _, item := range result.Items {
		if !item.Success {
			result.Success = false
			break
		}
	}

	// Emit event
	result.DomainEvents = append(result.DomainEvents,
		NewStockAllocationReleasedEvent(correlationID, sourceType, sourceID, result.Items))

	return result, nil
}

// ReleaseResult represents the result of releasing allocated stock
type ReleaseResult struct {
	CorrelationID uuid.UUID            `json:"correlation_id"`
	SourceType    string               `json:"source_type"`
	SourceID      string               `json:"source_id"`
	Items         []ReleaseItemResult  `json:"items"`
	TotalReleased decimal.Decimal      `json:"total_released"`
	Success       bool                 `json:"success"`
	DomainEvents  []shared.DomainEvent `json:"-"`
}

// ReleaseItemResult represents the result of releasing a single lock
type ReleaseItemResult struct {
	InventoryItemID uuid.UUID       `json:"inventory_item_id"`
	LockID          uuid.UUID       `json:"lock_id"`
	Quantity        decimal.Decimal `json:"quantity"`
	Success         bool            `json:"success"`
	Error           error           `json:"-"`
	ErrorMessage    string          `json:"error_message,omitempty"`
}
