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

// SalesReturnCancelledHandler handles SalesReturnCancelledEvent
// and reverses inventory stock when a sales return is cancelled after goods were received
type SalesReturnCancelledHandler struct {
	inventoryService *inventoryapp.InventoryService
	logger           *zap.Logger
}

// NewSalesReturnCancelledHandler creates a new handler for sales return cancelled events
func NewSalesReturnCancelledHandler(
	inventoryService *inventoryapp.InventoryService,
	logger *zap.Logger,
) *SalesReturnCancelledHandler {
	return &SalesReturnCancelledHandler{
		inventoryService: inventoryService,
		logger:           logger,
	}
}

// EventTypes returns the event types this handler is interested in
func (h *SalesReturnCancelledHandler) EventTypes() []string {
	return []string{trade.EventTypeSalesReturnCancelled}
}

// Handle processes a SalesReturnCancelledEvent by reversing stock if goods were received
// If WasApproved is true, the return was in APPROVED or RECEIVING status before cancellation
// In RECEIVING status, goods may have already been received and stock restored
// This handler reverses any inventory restoration that occurred
func (h *SalesReturnCancelledHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	// Type assert to SalesReturnCancelledEvent
	cancelledEvent, ok := event.(*trade.SalesReturnCancelledEvent)
	if !ok {
		h.logger.Error("unexpected event type",
			zap.String("expected", trade.EventTypeSalesReturnCancelled),
			zap.String("actual", event.EventType()),
		)
		return fmt.Errorf("unexpected event type: expected %s, got %s",
			trade.EventTypeSalesReturnCancelled, event.EventType())
	}

	h.logger.Info("processing sales return cancelled event",
		zap.String("return_id", cancelledEvent.ReturnID.String()),
		zap.String("return_number", cancelledEvent.ReturnNumber),
		zap.String("sales_order_id", cancelledEvent.SalesOrderID.String()),
		zap.String("cancel_reason", cancelledEvent.CancelReason),
		zap.Bool("was_approved", cancelledEvent.WasApproved),
		zap.Int("items_count", len(cancelledEvent.Items)),
	)

	// If the return was not approved, no inventory operations would have been performed
	if !cancelledEvent.WasApproved {
		h.logger.Info("return was not approved, no inventory to reverse",
			zap.String("return_id", cancelledEvent.ReturnID.String()),
		)
		return nil
	}

	// Check if warehouse is set
	if cancelledEvent.WarehouseID == nil || *cancelledEvent.WarehouseID == uuid.Nil {
		h.logger.Warn("cancelled return has no warehouse ID, cannot reverse inventory",
			zap.String("return_id", cancelledEvent.ReturnID.String()),
		)
		return nil
	}

	// Check if there are any inventory transactions to reverse by querying transactions
	// We look for inbound transactions with source type SALES_RETURN and source ID matching this return
	filter := inventoryapp.TransactionListFilter{
		SourceType: string(inventory.SourceTypeSalesReturn),
		SourceID:   cancelledEvent.ReturnID.String(),
		Page:       1,
		PageSize:   100, // Should be enough for any return
	}

	transactions, _, err := h.inventoryService.ListTransactions(ctx, event.TenantID(), filter)
	queryFailed := err != nil
	if queryFailed {
		h.logger.Warn("failed to query inventory transactions for return, will attempt full reversal",
			zap.String("return_id", cancelledEvent.ReturnID.String()),
			zap.Error(err),
		)
		// Continue with full reversal attempt - queryFailed flag will be used below
	}

	// If query succeeded and no transactions found, no inventory was restored yet
	if !queryFailed && len(transactions) == 0 {
		h.logger.Info("no inventory transactions found for return, nothing to reverse",
			zap.String("return_id", cancelledEvent.ReturnID.String()),
		)
		return nil
	}

	// Reverse inventory for each item that was received
	var lastErr error
	successCount := 0
	reference := fmt.Sprintf("SR-CANCEL:%s", cancelledEvent.ReturnNumber)

	for _, item := range cancelledEvent.Items {
		// Check if this specific item had inventory restored
		// Skip this check if query failed - we'll attempt full reversal
		if !queryFailed && len(transactions) > 0 {
			itemRestored := false
			for _, tx := range transactions {
				if tx.ProductID == item.ProductID && tx.TransactionType == "INBOUND" {
					itemRestored = true
					break
				}
			}

			// If item wasn't restored, skip it
			if !itemRestored {
				h.logger.Debug("item was not restored, skipping reversal",
					zap.String("product_id", item.ProductID.String()),
					zap.String("product_name", item.ProductName),
				)
				continue
			}
		}

		// Use BaseQuantity for inventory operations (the quantity in base units)
		// This ensures correct inventory quantities when using auxiliary units
		req := inventoryapp.DecreaseStockRequest{
			WarehouseID: *cancelledEvent.WarehouseID,
			ProductID:   item.ProductID,
			Quantity:    item.BaseQuantity,
			SourceType:  string(inventory.SourceTypeSalesReturn),
			SourceID:    cancelledEvent.ReturnID.String(),
			Reference:   reference,
			Reason:      fmt.Sprintf("Sales return cancelled: %s - %s", cancelledEvent.ReturnNumber, cancelledEvent.CancelReason),
		}

		err := h.inventoryService.DecreaseStock(ctx, event.TenantID(), req)
		if err != nil {
			h.logger.Error("failed to reverse stock for cancelled return item",
				zap.String("return_id", cancelledEvent.ReturnID.String()),
				zap.String("product_id", item.ProductID.String()),
				zap.String("product_name", item.ProductName),
				zap.String("quantity", item.BaseQuantity.String()),
				zap.Error(err),
			)
			lastErr = err
			// Continue processing other items even if one fails
			continue
		}

		successCount++
		h.logger.Debug("stock reversed for cancelled return item",
			zap.String("product_id", item.ProductID.String()),
			zap.String("product_code", item.ProductCode),
			zap.String("product_name", item.ProductName),
			zap.String("return_quantity", item.ReturnQuantity.String()),
			zap.String("unit", item.Unit),
			zap.String("base_quantity", item.BaseQuantity.String()),
			zap.String("base_unit", item.BaseUnit),
		)
	}

	h.logger.Info("sales return cancellation inventory reversal completed",
		zap.String("return_id", cancelledEvent.ReturnID.String()),
		zap.String("return_number", cancelledEvent.ReturnNumber),
		zap.Int("total_items", len(cancelledEvent.Items)),
		zap.Int("success_count", successCount),
		zap.Bool("has_errors", lastErr != nil),
	)

	// Return the last error if any item failed
	if lastErr != nil {
		return fmt.Errorf("some items failed to reverse stock: %w", lastErr)
	}

	return nil
}

// Ensure SalesReturnCancelledHandler implements shared.EventHandler
var _ shared.EventHandler = (*SalesReturnCancelledHandler)(nil)
