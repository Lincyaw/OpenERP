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

// SalesOrderCancelledHandler handles SalesOrderCancelledEvent
// and releases inventory stock locks when a confirmed sales order is cancelled
type SalesOrderCancelledHandler struct {
	inventoryService *inventoryapp.InventoryService
	logger           *zap.Logger
}

// NewSalesOrderCancelledHandler creates a new handler for sales order cancelled events
func NewSalesOrderCancelledHandler(
	inventoryService *inventoryapp.InventoryService,
	logger *zap.Logger,
) *SalesOrderCancelledHandler {
	return &SalesOrderCancelledHandler{
		inventoryService: inventoryService,
		logger:           logger,
	}
}

// EventTypes returns the event types this handler is interested in
func (h *SalesOrderCancelledHandler) EventTypes() []string {
	return []string{trade.EventTypeSalesOrderCancelled}
}

// Handle processes a SalesOrderCancelledEvent by releasing stock locks if the order was confirmed
func (h *SalesOrderCancelledHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	// Type assert to SalesOrderCancelledEvent
	cancelledEvent, ok := event.(*trade.SalesOrderCancelledEvent)
	if !ok {
		h.logger.Error("unexpected event type",
			zap.String("expected", trade.EventTypeSalesOrderCancelled),
			zap.String("actual", event.EventType()),
		)
		return fmt.Errorf("unexpected event type: expected %s, got %s",
			trade.EventTypeSalesOrderCancelled, event.EventType())
	}

	h.logger.Info("processing sales order cancelled event",
		zap.String("order_id", cancelledEvent.OrderID.String()),
		zap.String("order_number", cancelledEvent.OrderNumber),
		zap.String("cancel_reason", cancelledEvent.CancelReason),
		zap.Bool("was_confirmed", cancelledEvent.WasConfirmed),
		zap.Int("items_count", len(cancelledEvent.Items)),
	)

	// Only release locks if the order was previously confirmed
	// Draft orders have no stock locks to release
	if !cancelledEvent.WasConfirmed {
		h.logger.Info("order was not confirmed, no stock locks to release",
			zap.String("order_id", cancelledEvent.OrderID.String()),
		)
		return nil
	}

	// Check if warehouse is set
	if cancelledEvent.WarehouseID == nil || *cancelledEvent.WarehouseID == uuid.Nil {
		h.logger.Warn("cancelled order has no warehouse ID, attempting to unlock by source anyway",
			zap.String("order_id", cancelledEvent.OrderID.String()),
		)
	}

	// Release all locks for this sales order
	sourceType := "SALES_ORDER"
	sourceID := cancelledEvent.OrderID.String()

	releasedCount, err := h.inventoryService.UnlockBySource(ctx, event.TenantID(), sourceType, sourceID)
	if err != nil {
		h.logger.Error("failed to release stock locks for cancelled order",
			zap.String("order_id", cancelledEvent.OrderID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to release stock locks: %w", err)
	}

	h.logger.Info("sales order stock locks released",
		zap.String("order_id", cancelledEvent.OrderID.String()),
		zap.String("order_number", cancelledEvent.OrderNumber),
		zap.Int("total_items", len(cancelledEvent.Items)),
		zap.Int("locks_released", releasedCount),
		zap.String("cancel_reason", cancelledEvent.CancelReason),
	)

	// Note: If releasedCount is 0 and there were items, the locks may have already:
	// - Expired and been released automatically
	// - Been consumed (order was shipped before cancel - shouldn't happen)
	// This is not an error condition, just informational
	if releasedCount == 0 && len(cancelledEvent.Items) > 0 {
		h.logger.Warn("no active locks found for cancelled order",
			zap.String("order_id", cancelledEvent.OrderID.String()),
			zap.String("order_number", cancelledEvent.OrderNumber),
			zap.Int("expected_items", len(cancelledEvent.Items)),
		)
	}

	return nil
}

// Ensure SalesOrderCancelledHandler implements shared.EventHandler
var _ shared.EventHandler = (*SalesOrderCancelledHandler)(nil)
