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

// SalesOrderConfirmedHandler handles SalesOrderConfirmedEvent
// and locks inventory stock when a sales order is confirmed
type SalesOrderConfirmedHandler struct {
	inventoryService *inventoryapp.InventoryService
	logger           *zap.Logger
}

// NewSalesOrderConfirmedHandler creates a new handler for sales order confirmed events
func NewSalesOrderConfirmedHandler(
	inventoryService *inventoryapp.InventoryService,
	logger *zap.Logger,
) *SalesOrderConfirmedHandler {
	return &SalesOrderConfirmedHandler{
		inventoryService: inventoryService,
		logger:           logger,
	}
}

// EventTypes returns the event types this handler is interested in
func (h *SalesOrderConfirmedHandler) EventTypes() []string {
	return []string{trade.EventTypeSalesOrderConfirmed}
}

// Handle processes a SalesOrderConfirmedEvent by locking stock for each item
func (h *SalesOrderConfirmedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	// Type assert to SalesOrderConfirmedEvent
	confirmedEvent, ok := event.(*trade.SalesOrderConfirmedEvent)
	if !ok {
		h.logger.Error("unexpected event type",
			zap.String("expected", trade.EventTypeSalesOrderConfirmed),
			zap.String("actual", event.EventType()),
		)
		return fmt.Errorf("unexpected event type: expected %s, got %s",
			trade.EventTypeSalesOrderConfirmed, event.EventType())
	}

	h.logger.Info("processing sales order confirmed event",
		zap.String("order_id", confirmedEvent.OrderID.String()),
		zap.String("order_number", confirmedEvent.OrderNumber),
		zap.String("customer_id", confirmedEvent.CustomerID.String()),
		zap.Int("items_count", len(confirmedEvent.Items)),
	)

	// Validate warehouse ID is set
	if confirmedEvent.WarehouseID == nil || *confirmedEvent.WarehouseID == uuid.Nil {
		h.logger.Error("warehouse ID is required for stock locking",
			zap.String("order_id", confirmedEvent.OrderID.String()),
		)
		return fmt.Errorf("warehouse ID is required for stock locking")
	}

	// Process each item and lock stock
	var lastErr error
	successCount := 0
	lockedItems := make([]string, 0)

	for _, item := range confirmedEvent.Items {
		req := inventoryapp.LockStockRequest{
			WarehouseID: *confirmedEvent.WarehouseID,
			ProductID:   item.ProductID,
			Quantity:    item.Quantity,
			SourceType:  "SALES_ORDER",
			SourceID:    confirmedEvent.OrderID.String(),
			ExpireAt:    nil, // Use default expiry (30 minutes)
		}

		lockResp, err := h.inventoryService.LockStock(ctx, event.TenantID(), req)
		if err != nil {
			h.logger.Error("failed to lock stock for order item",
				zap.String("order_id", confirmedEvent.OrderID.String()),
				zap.String("product_id", item.ProductID.String()),
				zap.String("product_name", item.ProductName),
				zap.String("quantity", item.Quantity.String()),
				zap.Error(err),
			)
			lastErr = err
			// Continue processing other items even if one fails
			// This allows partial fulfillment visibility
			continue
		}

		successCount++
		lockedItems = append(lockedItems, item.ProductCode)
		h.logger.Debug("stock locked for order item",
			zap.String("lock_id", lockResp.LockID.String()),
			zap.String("product_id", item.ProductID.String()),
			zap.String("product_code", item.ProductCode),
			zap.String("product_name", item.ProductName),
			zap.String("quantity", item.Quantity.String()),
			zap.Time("expires_at", lockResp.ExpireAt),
		)
	}

	h.logger.Info("sales order stock locking completed",
		zap.String("order_id", confirmedEvent.OrderID.String()),
		zap.String("order_number", confirmedEvent.OrderNumber),
		zap.Int("total_items", len(confirmedEvent.Items)),
		zap.Int("success_count", successCount),
		zap.Strings("locked_items", lockedItems),
		zap.Bool("has_errors", lastErr != nil),
	)

	// Return the last error if any item failed
	if lastErr != nil {
		return fmt.Errorf("some items failed to lock: %w", lastErr)
	}

	return nil
}

// Ensure SalesOrderConfirmedHandler implements shared.EventHandler
var _ shared.EventHandler = (*SalesOrderConfirmedHandler)(nil)
