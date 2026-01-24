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

// SalesReturnCompletedHandler handles SalesReturnCompletedEvent
// and creates a red-letter (negative) AccountReceivable when a sales return is completed.
// This effectively reduces the customer's outstanding balance.
type SalesReturnCompletedHandler struct {
	receivableRepo finance.AccountReceivableRepository
	logger         *zap.Logger
}

// NewSalesReturnCompletedHandler creates a new handler for sales return completed events
func NewSalesReturnCompletedHandler(
	receivableRepo finance.AccountReceivableRepository,
	logger *zap.Logger,
) *SalesReturnCompletedHandler {
	return &SalesReturnCompletedHandler{
		receivableRepo: receivableRepo,
		logger:         logger,
	}
}

// EventTypes returns the event types this handler is interested in
func (h *SalesReturnCompletedHandler) EventTypes() []string {
	return []string{trade.EventTypeSalesReturnCompleted}
}

// Handle processes a SalesReturnCompletedEvent by creating a red-letter AccountReceivable
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

	h.logger.Info("processing sales return completed event for red-letter receivable creation",
		zap.String("return_id", completedEvent.ReturnID.String()),
		zap.String("return_number", completedEvent.ReturnNumber),
		zap.String("sales_order_id", completedEvent.SalesOrderID.String()),
		zap.String("sales_order_number", completedEvent.SalesOrderNumber),
		zap.String("customer_id", completedEvent.CustomerID.String()),
		zap.String("customer_name", completedEvent.CustomerName),
		zap.String("total_refund", completedEvent.TotalRefund.String()),
	)

	// Idempotency check: verify receivable doesn't already exist for this source
	exists, err := h.receivableRepo.ExistsBySource(
		ctx,
		completedEvent.TenantID(),
		finance.SourceTypeSalesReturn,
		completedEvent.ReturnID,
	)
	if err != nil {
		h.logger.Error("failed to check existing receivable",
			zap.String("return_id", completedEvent.ReturnID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to check existing receivable: %w", err)
	}
	if exists {
		h.logger.Warn("red-letter receivable already exists for sales return, skipping",
			zap.String("return_id", completedEvent.ReturnID.String()),
			zap.String("return_number", completedEvent.ReturnNumber),
		)
		return nil // Idempotent - already processed
	}

	// Skip if refund amount is zero (nothing to credit)
	if completedEvent.TotalRefund.IsZero() {
		h.logger.Info("skipping red-letter receivable creation - refund amount is zero",
			zap.String("return_id", completedEvent.ReturnID.String()),
			zap.String("return_number", completedEvent.ReturnNumber),
		)
		return nil
	}

	// Generate receivable number
	receivableNumber, err := h.receivableRepo.GenerateReceivableNumber(ctx, completedEvent.TenantID())
	if err != nil {
		h.logger.Error("failed to generate receivable number",
			zap.String("return_id", completedEvent.ReturnID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to generate receivable number: %w", err)
	}

	// For red-letter entries, the due date is immediate (same day)
	dueDate := time.Now()

	// Create the receivable amount - use positive amount for the domain object
	// The SourceTypeSalesReturn identifies this as a credit/red-letter entry
	amount := valueobject.NewMoneyCNY(completedEvent.TotalRefund)

	// Create red-letter AccountReceivable
	// This creates a receivable that will reduce the customer's outstanding balance
	receivable, err := finance.NewAccountReceivable(
		completedEvent.TenantID(),
		receivableNumber,
		completedEvent.CustomerID,
		completedEvent.CustomerName,
		finance.SourceTypeSalesReturn,
		completedEvent.ReturnID,
		completedEvent.ReturnNumber,
		amount,
		&dueDate,
	)
	if err != nil {
		h.logger.Error("failed to create red-letter account receivable",
			zap.String("return_id", completedEvent.ReturnID.String()),
			zap.String("return_number", completedEvent.ReturnNumber),
			zap.Error(err),
		)
		return fmt.Errorf("failed to create red-letter account receivable: %w", err)
	}

	// Add remark to indicate this is a red-letter/credit entry
	receivable.Remark = fmt.Sprintf("Red-letter entry for sales return %s (original order: %s)",
		completedEvent.ReturnNumber, completedEvent.SalesOrderNumber)

	// Save the receivable
	if err := h.receivableRepo.Save(ctx, receivable); err != nil {
		h.logger.Error("failed to save red-letter account receivable",
			zap.String("return_id", completedEvent.ReturnID.String()),
			zap.String("receivable_number", receivableNumber),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save red-letter account receivable: %w", err)
	}

	h.logger.Info("red-letter account receivable created successfully",
		zap.String("receivable_id", receivable.ID.String()),
		zap.String("receivable_number", receivableNumber),
		zap.String("return_id", completedEvent.ReturnID.String()),
		zap.String("return_number", completedEvent.ReturnNumber),
		zap.String("sales_order_id", completedEvent.SalesOrderID.String()),
		zap.String("sales_order_number", completedEvent.SalesOrderNumber),
		zap.String("customer_id", completedEvent.CustomerID.String()),
		zap.String("customer_name", completedEvent.CustomerName),
		zap.String("refund_amount", completedEvent.TotalRefund.String()),
	)

	return nil
}

// Ensure SalesReturnCompletedHandler implements shared.EventHandler
var _ shared.EventHandler = (*SalesReturnCompletedHandler)(nil)
