package trade

import (
	"context"
	"fmt"

	inventoryapp "github.com/erp/backend/internal/application/inventory"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// SalesOrderShippedHandler handles SalesOrderShippedEvent
// and deducts locked inventory stock when a sales order is shipped
type SalesOrderShippedHandler struct {
	inventoryService *inventoryapp.InventoryService
	logger           *zap.Logger
}

// NewSalesOrderShippedHandler creates a new handler for sales order shipped events
func NewSalesOrderShippedHandler(
	inventoryService *inventoryapp.InventoryService,
	logger *zap.Logger,
) *SalesOrderShippedHandler {
	return &SalesOrderShippedHandler{
		inventoryService: inventoryService,
		logger:           logger,
	}
}

// EventTypes returns the event types this handler is interested in
func (h *SalesOrderShippedHandler) EventTypes() []string {
	return []string{trade.EventTypeSalesOrderShipped}
}

// Handle processes a SalesOrderShippedEvent by deducting locked stock
func (h *SalesOrderShippedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	// Type assert to SalesOrderShippedEvent
	shippedEvent, ok := event.(*trade.SalesOrderShippedEvent)
	if !ok {
		h.logger.Error("unexpected event type",
			zap.String("expected", trade.EventTypeSalesOrderShipped),
			zap.String("actual", event.EventType()),
		)
		return fmt.Errorf("unexpected event type: expected %s, got %s",
			trade.EventTypeSalesOrderShipped, event.EventType())
	}

	h.logger.Info("processing sales order shipped event",
		zap.String("order_id", shippedEvent.OrderID.String()),
		zap.String("order_number", shippedEvent.OrderNumber),
		zap.String("warehouse_id", shippedEvent.WarehouseID.String()),
		zap.Int("items_count", len(shippedEvent.Items)),
	)

	// Validate warehouse ID
	if shippedEvent.WarehouseID == uuid.Nil {
		h.logger.Error("warehouse ID is required for stock deduction",
			zap.String("order_id", shippedEvent.OrderID.String()),
		)
		return fmt.Errorf("warehouse ID is required for stock deduction")
	}

	// Get all locks for this sales order
	sourceType := "SALES_ORDER"
	sourceID := shippedEvent.OrderID.String()
	locks, err := h.inventoryService.GetLocksBySource(ctx, sourceType, sourceID)
	if err != nil {
		h.logger.Error("failed to get locks for sales order",
			zap.String("order_id", shippedEvent.OrderID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to get locks: %w", err)
	}

	// Create a map of productID -> lockID for quick lookup
	lockByProduct := make(map[uuid.UUID]uuid.UUID)
	for _, lock := range locks {
		// Only consider active locks (not released, not consumed)
		if !lock.Released && !lock.Consumed {
			lockByProduct[lock.ProductID] = lock.ID
		}
	}

	// Process each item and deduct stock
	var lastErr error
	successCount := 0
	reference := fmt.Sprintf("SO:%s", shippedEvent.OrderNumber)

	for _, item := range shippedEvent.Items {
		lockID, exists := lockByProduct[item.ProductID]
		if !exists {
			h.logger.Warn("no active lock found for item, skipping deduction",
				zap.String("order_id", shippedEvent.OrderID.String()),
				zap.String("product_id", item.ProductID.String()),
				zap.String("product_name", item.ProductName),
			)
			// Not having a lock is not an error - the lock might have expired
			// and been released. The item can still be shipped (manual reconciliation needed).
			continue
		}

		req := inventoryapp.DeductStockRequest{
			LockID:     lockID,
			SourceType: sourceType,
			SourceID:   sourceID,
			Reference:  reference,
			OperatorID: nil,
		}

		if err := h.inventoryService.DeductStock(ctx, event.TenantID(), req); err != nil {
			h.logger.Error("failed to deduct stock for order item",
				zap.String("order_id", shippedEvent.OrderID.String()),
				zap.String("product_id", item.ProductID.String()),
				zap.String("product_name", item.ProductName),
				zap.String("lock_id", lockID.String()),
				zap.Error(err),
			)
			lastErr = err
			// Continue processing other items
			continue
		}

		successCount++
		h.logger.Debug("stock deducted for order item",
			zap.String("product_id", item.ProductID.String()),
			zap.String("product_code", item.ProductCode),
			zap.String("product_name", item.ProductName),
			zap.String("quantity", item.Quantity.String()),
			zap.String("lock_id", lockID.String()),
		)
	}

	h.logger.Info("sales order stock deduction completed",
		zap.String("order_id", shippedEvent.OrderID.String()),
		zap.String("order_number", shippedEvent.OrderNumber),
		zap.Int("total_items", len(shippedEvent.Items)),
		zap.Int("success_count", successCount),
		zap.Int("locks_found", len(locks)),
		zap.Bool("has_errors", lastErr != nil),
	)

	// Return the last error if any item failed
	if lastErr != nil {
		return fmt.Errorf("some items failed to deduct: %w", lastErr)
	}

	return nil
}

// Ensure SalesOrderShippedHandler implements shared.EventHandler
var _ shared.EventHandler = (*SalesOrderShippedHandler)(nil)
