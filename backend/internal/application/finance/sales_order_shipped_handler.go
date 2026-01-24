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

// SalesOrderShippedHandler handles SalesOrderShippedEvent
// and creates AccountReceivable when a sales order is shipped
type SalesOrderShippedHandler struct {
	receivableRepo finance.AccountReceivableRepository
	logger         *zap.Logger
}

// NewSalesOrderShippedHandler creates a new handler for sales order shipped events
func NewSalesOrderShippedHandler(
	receivableRepo finance.AccountReceivableRepository,
	logger *zap.Logger,
) *SalesOrderShippedHandler {
	return &SalesOrderShippedHandler{
		receivableRepo: receivableRepo,
		logger:         logger,
	}
}

// EventTypes returns the event types this handler is interested in
func (h *SalesOrderShippedHandler) EventTypes() []string {
	return []string{trade.EventTypeSalesOrderShipped}
}

// Handle processes a SalesOrderShippedEvent by creating an AccountReceivable
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

	h.logger.Info("processing sales order shipped event for receivable creation",
		zap.String("order_id", shippedEvent.OrderID.String()),
		zap.String("order_number", shippedEvent.OrderNumber),
		zap.String("customer_id", shippedEvent.CustomerID.String()),
		zap.String("customer_name", shippedEvent.CustomerName),
		zap.String("payable_amount", shippedEvent.PayableAmount.String()),
	)

	// Idempotency check: verify receivable doesn't already exist for this source
	exists, err := h.receivableRepo.ExistsBySource(
		ctx,
		shippedEvent.TenantID(),
		finance.SourceTypeSalesOrder,
		shippedEvent.OrderID,
	)
	if err != nil {
		h.logger.Error("failed to check existing receivable",
			zap.String("order_id", shippedEvent.OrderID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to check existing receivable: %w", err)
	}
	if exists {
		h.logger.Warn("receivable already exists for sales order, skipping",
			zap.String("order_id", shippedEvent.OrderID.String()),
			zap.String("order_number", shippedEvent.OrderNumber),
		)
		return nil // Idempotent - already processed
	}

	// Skip if payable amount is zero (fully prepaid orders)
	if shippedEvent.PayableAmount.IsZero() {
		h.logger.Info("skipping receivable creation - order is fully prepaid",
			zap.String("order_id", shippedEvent.OrderID.String()),
			zap.String("order_number", shippedEvent.OrderNumber),
		)
		return nil
	}

	// Generate receivable number
	receivableNumber, err := h.receivableRepo.GenerateReceivableNumber(ctx, shippedEvent.TenantID())
	if err != nil {
		h.logger.Error("failed to generate receivable number",
			zap.String("order_id", shippedEvent.OrderID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to generate receivable number: %w", err)
	}

	// Set default due date (30 days from now)
	dueDate := time.Now().AddDate(0, 0, 30)

	// Create the receivable amount
	amount := valueobject.NewMoneyCNY(shippedEvent.PayableAmount)

	// Create AccountReceivable
	receivable, err := finance.NewAccountReceivable(
		shippedEvent.TenantID(),
		receivableNumber,
		shippedEvent.CustomerID,
		shippedEvent.CustomerName,
		finance.SourceTypeSalesOrder,
		shippedEvent.OrderID,
		shippedEvent.OrderNumber,
		amount,
		&dueDate,
	)
	if err != nil {
		h.logger.Error("failed to create account receivable",
			zap.String("order_id", shippedEvent.OrderID.String()),
			zap.String("order_number", shippedEvent.OrderNumber),
			zap.Error(err),
		)
		return fmt.Errorf("failed to create account receivable: %w", err)
	}

	// Save the receivable
	if err := h.receivableRepo.Save(ctx, receivable); err != nil {
		h.logger.Error("failed to save account receivable",
			zap.String("order_id", shippedEvent.OrderID.String()),
			zap.String("receivable_number", receivableNumber),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save account receivable: %w", err)
	}

	h.logger.Info("account receivable created successfully",
		zap.String("receivable_id", receivable.ID.String()),
		zap.String("receivable_number", receivableNumber),
		zap.String("order_id", shippedEvent.OrderID.String()),
		zap.String("order_number", shippedEvent.OrderNumber),
		zap.String("customer_id", shippedEvent.CustomerID.String()),
		zap.String("customer_name", shippedEvent.CustomerName),
		zap.String("amount", shippedEvent.PayableAmount.String()),
		zap.Time("due_date", dueDate),
	)

	return nil
}

// Ensure SalesOrderShippedHandler implements shared.EventHandler
var _ shared.EventHandler = (*SalesOrderShippedHandler)(nil)
