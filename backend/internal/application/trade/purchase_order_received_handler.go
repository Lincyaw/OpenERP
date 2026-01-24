package trade

import (
	"context"
	"fmt"

	inventoryapp "github.com/erp/backend/internal/application/inventory"
	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/trade"
	"go.uber.org/zap"
)

// PurchaseOrderReceivedHandler handles PurchaseOrderReceivedEvent
// and updates inventory when goods are received
type PurchaseOrderReceivedHandler struct {
	inventoryService *inventoryapp.InventoryService
	logger           *zap.Logger
}

// NewPurchaseOrderReceivedHandler creates a new handler for purchase order received events
func NewPurchaseOrderReceivedHandler(
	inventoryService *inventoryapp.InventoryService,
	logger *zap.Logger,
) *PurchaseOrderReceivedHandler {
	return &PurchaseOrderReceivedHandler{
		inventoryService: inventoryService,
		logger:           logger,
	}
}

// EventTypes returns the event types this handler is interested in
func (h *PurchaseOrderReceivedHandler) EventTypes() []string {
	return []string{trade.EventTypePurchaseOrderReceived}
}

// Handle processes a PurchaseOrderReceivedEvent
func (h *PurchaseOrderReceivedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	// Type assert to PurchaseOrderReceivedEvent
	receivedEvent, ok := event.(*trade.PurchaseOrderReceivedEvent)
	if !ok {
		h.logger.Error("unexpected event type",
			zap.String("expected", trade.EventTypePurchaseOrderReceived),
			zap.String("actual", event.EventType()),
		)
		return fmt.Errorf("unexpected event type: expected %s, got %s",
			trade.EventTypePurchaseOrderReceived, event.EventType())
	}

	h.logger.Info("processing purchase order received event",
		zap.String("order_id", receivedEvent.OrderID.String()),
		zap.String("order_number", receivedEvent.OrderNumber),
		zap.Int("items_count", len(receivedEvent.ReceivedItems)),
		zap.Bool("is_fully_received", receivedEvent.IsFullyReceived),
	)

	// Validate warehouse ID
	if receivedEvent.WarehouseID.String() == "00000000-0000-0000-0000-000000000000" {
		h.logger.Error("warehouse ID is required for inventory update",
			zap.String("order_id", receivedEvent.OrderID.String()),
		)
		return fmt.Errorf("warehouse ID is required for inventory update")
	}

	// Process each received item
	var lastErr error
	successCount := 0
	for _, item := range receivedEvent.ReceivedItems {
		req := inventoryapp.IncreaseStockRequest{
			WarehouseID: receivedEvent.WarehouseID,
			ProductID:   item.ProductID,
			Quantity:    item.Quantity,
			UnitCost:    item.UnitCost,
			SourceType:  string(inventory.SourceTypePurchaseOrder),
			SourceID:    receivedEvent.OrderID.String(),
			BatchNumber: item.BatchNumber,
			ExpiryDate:  item.ExpiryDate,
			Reference:   fmt.Sprintf("PO:%s", receivedEvent.OrderNumber),
			Reason:      "Purchase order receiving",
		}

		if _, err := h.inventoryService.IncreaseStock(ctx, event.TenantID(), req); err != nil {
			h.logger.Error("failed to increase stock for received item",
				zap.String("order_id", receivedEvent.OrderID.String()),
				zap.String("product_id", item.ProductID.String()),
				zap.String("product_name", item.ProductName),
				zap.String("quantity", item.Quantity.String()),
				zap.Error(err),
			)
			lastErr = err
			// Continue processing other items even if one fails
			continue
		}

		successCount++
		h.logger.Debug("stock increased for received item",
			zap.String("product_id", item.ProductID.String()),
			zap.String("product_name", item.ProductName),
			zap.String("quantity", item.Quantity.String()),
			zap.String("unit_cost", item.UnitCost.String()),
			zap.String("batch_number", item.BatchNumber),
		)
	}

	h.logger.Info("purchase order receiving completed",
		zap.String("order_id", receivedEvent.OrderID.String()),
		zap.Int("total_items", len(receivedEvent.ReceivedItems)),
		zap.Int("success_count", successCount),
		zap.Bool("has_errors", lastErr != nil),
	)

	// Return the last error if any item failed
	// This allows for partial success visibility while still indicating failure
	if lastErr != nil {
		return fmt.Errorf("some items failed to process: %w", lastErr)
	}

	return nil
}

// Ensure PurchaseOrderReceivedHandler implements shared.EventHandler
var _ shared.EventHandler = (*PurchaseOrderReceivedHandler)(nil)
