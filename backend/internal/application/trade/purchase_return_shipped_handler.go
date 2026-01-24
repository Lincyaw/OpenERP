package trade

import (
	"context"
	"fmt"

	inventoryapp "github.com/erp/backend/internal/application/inventory"
	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PurchaseReturnShippedHandler handles PurchaseReturnShippedEvent
// and deducts inventory stock when goods are shipped back to supplier
type PurchaseReturnShippedHandler struct {
	inventoryService *inventoryapp.InventoryService
	logger           *zap.Logger
}

// NewPurchaseReturnShippedHandler creates a new handler for purchase return shipped events
func NewPurchaseReturnShippedHandler(
	inventoryService *inventoryapp.InventoryService,
	logger *zap.Logger,
) *PurchaseReturnShippedHandler {
	return &PurchaseReturnShippedHandler{
		inventoryService: inventoryService,
		logger:           logger,
	}
}

// EventTypes returns the event types this handler is interested in
func (h *PurchaseReturnShippedHandler) EventTypes() []string {
	return []string{trade.EventTypePurchaseReturnShipped}
}

// Handle processes a PurchaseReturnShippedEvent by deducting stock for each returned item
func (h *PurchaseReturnShippedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	// Type assert to PurchaseReturnShippedEvent
	shippedEvent, ok := event.(*trade.PurchaseReturnShippedEvent)
	if !ok {
		h.logger.Error("unexpected event type",
			zap.String("expected", trade.EventTypePurchaseReturnShipped),
			zap.String("actual", event.EventType()),
		)
		return fmt.Errorf("unexpected event type: expected %s, got %s",
			trade.EventTypePurchaseReturnShipped, event.EventType())
	}

	h.logger.Info("processing purchase return shipped event",
		zap.String("return_id", shippedEvent.ReturnID.String()),
		zap.String("return_number", shippedEvent.ReturnNumber),
		zap.String("purchase_order_id", shippedEvent.PurchaseOrderID.String()),
		zap.String("warehouse_id", shippedEvent.WarehouseID.String()),
		zap.String("supplier_id", shippedEvent.SupplierID.String()),
		zap.Int("items_count", len(shippedEvent.Items)),
	)

	// Validate warehouse ID
	if shippedEvent.WarehouseID == uuid.Nil {
		h.logger.Error("warehouse ID is required for stock deduction",
			zap.String("return_id", shippedEvent.ReturnID.String()),
		)
		return fmt.Errorf("warehouse ID is required for stock deduction")
	}

	// Process each item and deduct stock
	var lastErr error
	successCount := 0
	reference := fmt.Sprintf("PR:%s", shippedEvent.ReturnNumber)

	for _, item := range shippedEvent.Items {
		req := inventoryapp.DecreaseStockRequest{
			WarehouseID: shippedEvent.WarehouseID,
			ProductID:   item.ProductID,
			Quantity:    item.ReturnQuantity,
			SourceType:  string(inventory.SourceTypePurchaseReturn),
			SourceID:    shippedEvent.ReturnID.String(),
			Reference:   reference,
			Reason:      fmt.Sprintf("Purchase return shipped: %s", shippedEvent.ReturnNumber),
		}

		err := h.inventoryService.DecreaseStock(ctx, event.TenantID(), req)
		if err != nil {
			h.logger.Error("failed to deduct stock for return item",
				zap.String("return_id", shippedEvent.ReturnID.String()),
				zap.String("product_id", item.ProductID.String()),
				zap.String("product_name", item.ProductName),
				zap.String("quantity", item.ReturnQuantity.String()),
				zap.Error(err),
			)
			lastErr = err
			// Continue processing other items even if one fails
			continue
		}

		successCount++
		h.logger.Debug("stock deducted for return item",
			zap.String("product_id", item.ProductID.String()),
			zap.String("product_code", item.ProductCode),
			zap.String("product_name", item.ProductName),
			zap.String("quantity", item.ReturnQuantity.String()),
		)
	}

	h.logger.Info("purchase return stock deduction completed",
		zap.String("return_id", shippedEvent.ReturnID.String()),
		zap.String("return_number", shippedEvent.ReturnNumber),
		zap.Int("total_items", len(shippedEvent.Items)),
		zap.Int("success_count", successCount),
		zap.Bool("has_errors", lastErr != nil),
	)

	// Return the last error if any item failed
	if lastErr != nil {
		return fmt.Errorf("some items failed to deduct stock: %w", lastErr)
	}

	return nil
}

// Ensure PurchaseReturnShippedHandler implements shared.EventHandler
var _ shared.EventHandler = (*PurchaseReturnShippedHandler)(nil)
