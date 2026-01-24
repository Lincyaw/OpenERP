package trade

import (
	"context"
	"fmt"

	inventoryapp "github.com/erp/backend/internal/application/inventory"
	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// SalesReturnCompletedHandler handles SalesReturnCompletedEvent
// and restores inventory stock when a sales return is completed
type SalesReturnCompletedHandler struct {
	inventoryService *inventoryapp.InventoryService
	logger           *zap.Logger
}

// NewSalesReturnCompletedHandler creates a new handler for sales return completed events
func NewSalesReturnCompletedHandler(
	inventoryService *inventoryapp.InventoryService,
	logger *zap.Logger,
) *SalesReturnCompletedHandler {
	return &SalesReturnCompletedHandler{
		inventoryService: inventoryService,
		logger:           logger,
	}
}

// EventTypes returns the event types this handler is interested in
func (h *SalesReturnCompletedHandler) EventTypes() []string {
	return []string{trade.EventTypeSalesReturnCompleted}
}

// Handle processes a SalesReturnCompletedEvent by restoring stock for each returned item
func (h *SalesReturnCompletedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	// Type assert to SalesReturnCompletedEvent
	completedEvent, ok := event.(*trade.SalesReturnCompletedEvent)
	if !ok {
		h.logger.Error("unexpected event type",
			zap.String("expected", trade.EventTypeSalesReturnCompleted),
			zap.String("actual", event.EventType()),
		)
		return fmt.Errorf("unexpected event type: expected %s, got %s",
			trade.EventTypeSalesReturnCompleted, event.EventType())
	}

	h.logger.Info("processing sales return completed event",
		zap.String("return_id", completedEvent.ReturnID.String()),
		zap.String("return_number", completedEvent.ReturnNumber),
		zap.String("sales_order_id", completedEvent.SalesOrderID.String()),
		zap.String("warehouse_id", completedEvent.WarehouseID.String()),
		zap.String("customer_id", completedEvent.CustomerID.String()),
		zap.Int("items_count", len(completedEvent.Items)),
	)

	// Validate warehouse ID
	if completedEvent.WarehouseID == uuid.Nil {
		h.logger.Error("warehouse ID is required for stock restoration",
			zap.String("return_id", completedEvent.ReturnID.String()),
		)
		return fmt.Errorf("warehouse ID is required for stock restoration")
	}

	// Process each item and restore stock
	var lastErr error
	successCount := 0
	reference := fmt.Sprintf("SR:%s", completedEvent.ReturnNumber)

	for _, item := range completedEvent.Items {
		// Get the current unit cost from inventory for this product
		// This will be used to maintain accurate inventory valuation
		unitCost, err := h.getUnitCostForProduct(ctx, event.TenantID(), completedEvent.WarehouseID, item.ProductID)
		if err != nil {
			h.logger.Warn("failed to get unit cost, using item unit price",
				zap.String("product_id", item.ProductID.String()),
				zap.Error(err),
			)
			// Fall back to the original unit price from the sales order
			unitCost = item.UnitPrice
		}

		req := inventoryapp.IncreaseStockRequest{
			WarehouseID: completedEvent.WarehouseID,
			ProductID:   item.ProductID,
			Quantity:    item.ReturnQuantity,
			UnitCost:    unitCost,
			SourceType:  string(inventory.SourceTypeSalesReturn),
			SourceID:    completedEvent.ReturnID.String(),
			Reference:   reference,
			Reason:      fmt.Sprintf("Sales return: %s", completedEvent.ReturnNumber),
		}

		_, err = h.inventoryService.IncreaseStock(ctx, event.TenantID(), req)
		if err != nil {
			h.logger.Error("failed to restore stock for return item",
				zap.String("return_id", completedEvent.ReturnID.String()),
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
		h.logger.Debug("stock restored for return item",
			zap.String("product_id", item.ProductID.String()),
			zap.String("product_code", item.ProductCode),
			zap.String("product_name", item.ProductName),
			zap.String("quantity", item.ReturnQuantity.String()),
		)
	}

	h.logger.Info("sales return stock restoration completed",
		zap.String("return_id", completedEvent.ReturnID.String()),
		zap.String("return_number", completedEvent.ReturnNumber),
		zap.Int("total_items", len(completedEvent.Items)),
		zap.Int("success_count", successCount),
		zap.Bool("has_errors", lastErr != nil),
	)

	// Return the last error if any item failed
	if lastErr != nil {
		return fmt.Errorf("some items failed to restore stock: %w", lastErr)
	}

	return nil
}

// getUnitCostForProduct retrieves the current unit cost for a product in a warehouse
// Returns the unit price from the return item if inventory doesn't exist
func (h *SalesReturnCompletedHandler) getUnitCostForProduct(
	ctx context.Context,
	tenantID, warehouseID, productID uuid.UUID,
) (decimal.Decimal, error) {
	item, err := h.inventoryService.GetByWarehouseAndProduct(ctx, tenantID, warehouseID, productID)
	if err != nil {
		// If no inventory exists, return an error so caller can fall back
		return decimal.Zero, err
	}
	return item.UnitCost, nil
}

// Ensure SalesReturnCompletedHandler implements shared.EventHandler
var _ shared.EventHandler = (*SalesReturnCompletedHandler)(nil)
