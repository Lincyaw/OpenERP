package finance

import (
	"context"
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/finance"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/erp/backend/internal/domain/trade"
	"go.uber.org/zap"
)

// PurchaseReturnCompletedHandler handles PurchaseReturnCompletedEvent
// and creates a red-letter (negative) AccountPayable when a purchase return is completed.
// This effectively reduces the amount owed to the supplier.
type PurchaseReturnCompletedHandler struct {
	payableRepo finance.AccountPayableRepository
	logger      *zap.Logger
}

// NewPurchaseReturnCompletedHandler creates a new handler for purchase return completed events
func NewPurchaseReturnCompletedHandler(
	payableRepo finance.AccountPayableRepository,
	logger *zap.Logger,
) *PurchaseReturnCompletedHandler {
	return &PurchaseReturnCompletedHandler{
		payableRepo: payableRepo,
		logger:      logger,
	}
}

// EventTypes returns the event types this handler is interested in
func (h *PurchaseReturnCompletedHandler) EventTypes() []string {
	return []string{trade.EventTypePurchaseReturnCompleted}
}

// Handle processes a PurchaseReturnCompletedEvent by creating a red-letter AccountPayable
func (h *PurchaseReturnCompletedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	// Type assert to PurchaseReturnCompletedEvent
	completedEvent, ok := event.(*trade.PurchaseReturnCompletedEvent)
	if !ok {
		h.logger.Error("unexpected event type",
			zap.String("expected", trade.EventTypePurchaseReturnCompleted),
			zap.String("actual", event.EventType()),
		)
		return fmt.Errorf("unexpected event type: expected %s, got %s",
			trade.EventTypePurchaseReturnCompleted, event.EventType())
	}

	h.logger.Info("processing purchase return completed event for red-letter payable creation",
		zap.String("return_id", completedEvent.ReturnID.String()),
		zap.String("return_number", completedEvent.ReturnNumber),
		zap.String("purchase_order_id", completedEvent.PurchaseOrderID.String()),
		zap.String("purchase_order_number", completedEvent.PurchaseOrderNumber),
		zap.String("supplier_id", completedEvent.SupplierID.String()),
		zap.String("supplier_name", completedEvent.SupplierName),
		zap.String("total_refund", completedEvent.TotalRefund.String()),
	)

	// Idempotency check: verify payable doesn't already exist for this source
	exists, err := h.payableRepo.ExistsBySource(
		ctx,
		completedEvent.TenantID(),
		finance.PayableSourceTypePurchaseReturn,
		completedEvent.ReturnID,
	)
	if err != nil {
		h.logger.Error("failed to check existing payable",
			zap.String("return_id", completedEvent.ReturnID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to check existing payable: %w", err)
	}
	if exists {
		h.logger.Warn("red-letter payable already exists for purchase return, skipping",
			zap.String("return_id", completedEvent.ReturnID.String()),
			zap.String("return_number", completedEvent.ReturnNumber),
		)
		return nil // Idempotent - already processed
	}

	// Skip if refund amount is zero (nothing to debit)
	if completedEvent.TotalRefund.IsZero() {
		h.logger.Info("skipping red-letter payable creation - refund amount is zero",
			zap.String("return_id", completedEvent.ReturnID.String()),
			zap.String("return_number", completedEvent.ReturnNumber),
		)
		return nil
	}

	// Generate payable number
	payableNumber, err := h.payableRepo.GeneratePayableNumber(ctx, completedEvent.TenantID())
	if err != nil {
		h.logger.Error("failed to generate payable number",
			zap.String("return_id", completedEvent.ReturnID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to generate payable number: %w", err)
	}

	// For red-letter entries, the due date is immediate (same day)
	dueDate := time.Now()

	// Create the payable amount - use positive amount for the domain object
	// The PayableSourceTypePurchaseReturn identifies this as a debit/red-letter entry
	amount := valueobject.NewMoneyCNY(completedEvent.TotalRefund)

	// Create red-letter AccountPayable
	// This creates a payable that will reduce the amount owed to the supplier
	payable, err := finance.NewAccountPayable(
		completedEvent.TenantID(),
		payableNumber,
		completedEvent.SupplierID,
		completedEvent.SupplierName,
		finance.PayableSourceTypePurchaseReturn,
		completedEvent.ReturnID,
		completedEvent.ReturnNumber,
		amount,
		&dueDate,
	)
	if err != nil {
		h.logger.Error("failed to create red-letter account payable",
			zap.String("return_id", completedEvent.ReturnID.String()),
			zap.String("return_number", completedEvent.ReturnNumber),
			zap.Error(err),
		)
		return fmt.Errorf("failed to create red-letter account payable: %w", err)
	}

	// Add remark to indicate this is a red-letter/debit entry
	payable.Remark = fmt.Sprintf("Red-letter entry for purchase return %s (original order: %s)",
		completedEvent.ReturnNumber, completedEvent.PurchaseOrderNumber)

	// Save the payable
	if err := h.payableRepo.Save(ctx, payable); err != nil {
		h.logger.Error("failed to save red-letter account payable",
			zap.String("return_id", completedEvent.ReturnID.String()),
			zap.String("payable_number", payableNumber),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save red-letter account payable: %w", err)
	}

	h.logger.Info("red-letter account payable created successfully",
		zap.String("payable_id", payable.ID.String()),
		zap.String("payable_number", payableNumber),
		zap.String("return_id", completedEvent.ReturnID.String()),
		zap.String("return_number", completedEvent.ReturnNumber),
		zap.String("purchase_order_id", completedEvent.PurchaseOrderID.String()),
		zap.String("purchase_order_number", completedEvent.PurchaseOrderNumber),
		zap.String("supplier_id", completedEvent.SupplierID.String()),
		zap.String("supplier_name", completedEvent.SupplierName),
		zap.String("refund_amount", completedEvent.TotalRefund.String()),
	)

	return nil
}

// Ensure PurchaseReturnCompletedHandler implements shared.EventHandler
var _ shared.EventHandler = (*PurchaseReturnCompletedHandler)(nil)
