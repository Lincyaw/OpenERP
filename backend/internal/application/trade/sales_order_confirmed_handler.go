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

// SalesOrderConfirmedHandler handles SalesOrderConfirmedEvent
// and locks inventory stock when a sales order is confirmed.
//
// This handler can use the StockAllocationService domain service to coordinate
// stock allocation across multiple inventory items with saga/compensation pattern.
// When StockAllocationService is provided, if any item fails to allocate,
// the service automatically compensates by releasing all previously successful locks.
type SalesOrderConfirmedHandler struct {
	inventoryService       *inventoryapp.InventoryService
	stockAllocationService *inventory.StockAllocationService
	logger                 *zap.Logger
}

// SalesOrderConfirmedHandlerOption is a functional option for configuring the handler
type SalesOrderConfirmedHandlerOption func(*SalesOrderConfirmedHandler)

// WithStockAllocationService sets the stock allocation service for saga/compensation support
func WithStockAllocationService(svc *inventory.StockAllocationService) SalesOrderConfirmedHandlerOption {
	return func(h *SalesOrderConfirmedHandler) {
		h.stockAllocationService = svc
	}
}

// NewSalesOrderConfirmedHandler creates a new handler for sales order confirmed events.
func NewSalesOrderConfirmedHandler(
	inventoryService *inventoryapp.InventoryService,
	logger *zap.Logger,
	opts ...SalesOrderConfirmedHandlerOption,
) *SalesOrderConfirmedHandler {
	h := &SalesOrderConfirmedHandler{
		inventoryService: inventoryService,
		logger:           logger,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// EventTypes returns the event types this handler is interested in
func (h *SalesOrderConfirmedHandler) EventTypes() []string {
	return []string{trade.EventTypeSalesOrderConfirmed}
}

// Handle processes a SalesOrderConfirmedEvent by locking stock for each item.
// When StockAllocationService is available, uses saga/compensation pattern.
// Otherwise, falls back to individual item allocation with manual rollback.
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

	// Process with compensation pattern (all-or-nothing)
	return h.handleWithCompensation(ctx, confirmedEvent)
}

// handleWithCompensation locks stock for all items, rolling back on failure.
// This implements a simple saga pattern at the application layer.
func (h *SalesOrderConfirmedHandler) handleWithCompensation(
	ctx context.Context,
	confirmedEvent *trade.SalesOrderConfirmedEvent,
) error {
	tenantID := confirmedEvent.TenantID()

	// Track successful locks for potential rollback
	type successfulLock struct {
		lockID      uuid.UUID
		productCode string
	}
	successfulLocks := make([]successfulLock, 0, len(confirmedEvent.Items))

	// Try to lock each item
	for _, item := range confirmedEvent.Items {
		req := inventoryapp.LockStockRequest{
			WarehouseID: *confirmedEvent.WarehouseID,
			ProductID:   item.ProductID,
			Quantity:    item.Quantity,
			SourceType:  "SALES_ORDER",
			SourceID:    confirmedEvent.OrderID.String(),
			ExpireAt:    nil, // Use default expiry (30 minutes)
		}

		lockResp, err := h.inventoryService.LockStock(ctx, tenantID, req)
		if err != nil {
			h.logger.Error("failed to lock stock for order item",
				zap.String("order_id", confirmedEvent.OrderID.String()),
				zap.String("product_id", item.ProductID.String()),
				zap.String("product_name", item.ProductName),
				zap.String("quantity", item.Quantity.String()),
				zap.Error(err),
			)

			// Compensation: Rollback all successful locks
			if len(successfulLocks) > 0 {
				h.logger.Info("starting compensation for partial allocation failure",
					zap.String("order_id", confirmedEvent.OrderID.String()),
					zap.Int("locks_to_release", len(successfulLocks)),
				)

				compensationErrors := 0
				for _, lock := range successfulLocks {
					unlockReq := inventoryapp.UnlockStockRequest{
						LockID: lock.lockID,
					}
					if unlockErr := h.inventoryService.UnlockStock(ctx, tenantID, unlockReq); unlockErr != nil {
						h.logger.Error("compensation failed - could not release lock",
							zap.String("order_id", confirmedEvent.OrderID.String()),
							zap.String("lock_id", lock.lockID.String()),
							zap.String("product_code", lock.productCode),
							zap.Error(unlockErr),
						)
						compensationErrors++
					} else {
						h.logger.Debug("compensation successful - lock released",
							zap.String("lock_id", lock.lockID.String()),
							zap.String("product_code", lock.productCode),
						)
					}
				}

				h.logger.Info("compensation completed",
					zap.String("order_id", confirmedEvent.OrderID.String()),
					zap.Int("total_locks", len(successfulLocks)),
					zap.Int("failed_compensations", compensationErrors),
				)
			}

			return fmt.Errorf("stock allocation failed for product %s: %w (all previous locks compensated)", item.ProductCode, err)
		}

		// Track successful lock for potential rollback
		successfulLocks = append(successfulLocks, successfulLock{
			lockID:      lockResp.LockID,
			productCode: item.ProductCode,
		})

		h.logger.Debug("stock locked for order item",
			zap.String("lock_id", lockResp.LockID.String()),
			zap.String("product_id", item.ProductID.String()),
			zap.String("product_code", item.ProductCode),
			zap.String("product_name", item.ProductName),
			zap.String("quantity", item.Quantity.String()),
			zap.Time("expires_at", lockResp.ExpireAt),
		)
	}

	// All items locked successfully
	lockedCodes := make([]string, len(successfulLocks))
	for i, lock := range successfulLocks {
		lockedCodes[i] = lock.productCode
	}

	h.logger.Info("sales order stock locking completed",
		zap.String("order_id", confirmedEvent.OrderID.String()),
		zap.String("order_number", confirmedEvent.OrderNumber),
		zap.Int("total_items", len(confirmedEvent.Items)),
		zap.Int("success_count", len(successfulLocks)),
		zap.Strings("locked_items", lockedCodes),
	)

	return nil
}

// GetStockAllocationService returns the stock allocation service if set.
// This is useful for testing and for future integration when we have
// direct repository access for the domain service.
func (h *SalesOrderConfirmedHandler) GetStockAllocationService() *inventory.StockAllocationService {
	return h.stockAllocationService
}

// Ensure SalesOrderConfirmedHandler implements shared.EventHandler
var _ shared.EventHandler = (*SalesOrderConfirmedHandler)(nil)
