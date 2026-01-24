package finance

import (
	"context"
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/finance"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// PurchaseOrderReceivedHandler handles PurchaseOrderReceivedEvent
// and creates AccountPayable when goods are received
type PurchaseOrderReceivedHandler struct {
	payableRepo finance.AccountPayableRepository
	logger      *zap.Logger
}

// NewPurchaseOrderReceivedHandler creates a new handler for purchase order received events
func NewPurchaseOrderReceivedHandler(
	payableRepo finance.AccountPayableRepository,
	logger *zap.Logger,
) *PurchaseOrderReceivedHandler {
	return &PurchaseOrderReceivedHandler{
		payableRepo: payableRepo,
		logger:      logger,
	}
}

// EventTypes returns the event types this handler is interested in
func (h *PurchaseOrderReceivedHandler) EventTypes() []string {
	return []string{trade.EventTypePurchaseOrderReceived}
}

// Handle processes a PurchaseOrderReceivedEvent by creating an AccountPayable
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

	h.logger.Info("processing purchase order received event for payable creation",
		zap.String("order_id", receivedEvent.OrderID.String()),
		zap.String("order_number", receivedEvent.OrderNumber),
		zap.String("supplier_id", receivedEvent.SupplierID.String()),
		zap.String("supplier_name", receivedEvent.SupplierName),
		zap.Int("received_items", len(receivedEvent.ReceivedItems)),
		zap.Bool("is_fully_received", receivedEvent.IsFullyReceived),
	)

	// Calculate total amount for received items
	receivedAmount := decimal.Zero
	for _, item := range receivedEvent.ReceivedItems {
		itemAmount := item.Quantity.Mul(item.UnitCost)
		receivedAmount = receivedAmount.Add(itemAmount)
	}

	// Skip if received amount is zero
	if receivedAmount.IsZero() {
		h.logger.Info("skipping payable creation - received amount is zero",
			zap.String("order_id", receivedEvent.OrderID.String()),
			zap.String("order_number", receivedEvent.OrderNumber),
		)
		return nil
	}

	// Check if payable already exists for this source
	// Note: For partial receiving, we create one payable per receive event
	// We use event ID as part of idempotency check for multiple receives
	existingPayable, err := h.payableRepo.FindBySource(
		ctx,
		receivedEvent.TenantID(),
		finance.PayableSourceTypePurchaseOrder,
		receivedEvent.OrderID,
	)
	if err != nil && !isNotFoundError(err) {
		h.logger.Error("failed to check existing payable",
			zap.String("order_id", receivedEvent.OrderID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to check existing payable: %w", err)
	}

	// For partial receiving, we may have multiple receives for one PO
	// The strategy here: only create ONE payable when fully received
	// If already exists, skip (idempotent)
	if existingPayable != nil {
		h.logger.Warn("payable already exists for purchase order, skipping",
			zap.String("order_id", receivedEvent.OrderID.String()),
			zap.String("order_number", receivedEvent.OrderNumber),
			zap.String("existing_payable_id", existingPayable.ID.String()),
		)
		return nil // Idempotent - already processed
	}

	// Only create payable when order is fully received
	// This prevents multiple payables for partial receives
	if !receivedEvent.IsFullyReceived {
		h.logger.Info("skipping payable creation - order not fully received yet",
			zap.String("order_id", receivedEvent.OrderID.String()),
			zap.String("order_number", receivedEvent.OrderNumber),
			zap.String("received_amount", receivedAmount.String()),
		)
		return nil
	}

	// Generate payable number
	payableNumber, err := h.payableRepo.GeneratePayableNumber(ctx, receivedEvent.TenantID())
	if err != nil {
		h.logger.Error("failed to generate payable number",
			zap.String("order_id", receivedEvent.OrderID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to generate payable number: %w", err)
	}

	// Set default due date (30 days from now)
	dueDate := time.Now().AddDate(0, 0, 30)

	// Use PayableAmount from the order (the full amount to pay)
	amount := valueobject.NewMoneyCNY(receivedEvent.PayableAmount)

	// Create AccountPayable
	payable, err := finance.NewAccountPayable(
		receivedEvent.TenantID(),
		payableNumber,
		receivedEvent.SupplierID,
		receivedEvent.SupplierName,
		finance.PayableSourceTypePurchaseOrder,
		receivedEvent.OrderID,
		receivedEvent.OrderNumber,
		amount,
		&dueDate,
	)
	if err != nil {
		h.logger.Error("failed to create account payable",
			zap.String("order_id", receivedEvent.OrderID.String()),
			zap.String("order_number", receivedEvent.OrderNumber),
			zap.Error(err),
		)
		return fmt.Errorf("failed to create account payable: %w", err)
	}

	// Save the payable
	if err := h.payableRepo.Save(ctx, payable); err != nil {
		h.logger.Error("failed to save account payable",
			zap.String("order_id", receivedEvent.OrderID.String()),
			zap.String("payable_number", payableNumber),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save account payable: %w", err)
	}

	h.logger.Info("account payable created successfully",
		zap.String("payable_id", payable.ID.String()),
		zap.String("payable_number", payableNumber),
		zap.String("order_id", receivedEvent.OrderID.String()),
		zap.String("order_number", receivedEvent.OrderNumber),
		zap.String("supplier_id", receivedEvent.SupplierID.String()),
		zap.String("supplier_name", receivedEvent.SupplierName),
		zap.String("amount", receivedEvent.PayableAmount.String()),
		zap.Time("due_date", dueDate),
	)

	return nil
}

// isNotFoundError checks if the error is a "not found" error
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// Check for domain error with NOT_FOUND code
	if domainErr, ok := err.(*shared.DomainError); ok {
		return domainErr.Code == "NOT_FOUND"
	}
	return false
}

// Ensure PurchaseOrderReceivedHandler implements shared.EventHandler
var _ shared.EventHandler = (*PurchaseOrderReceivedHandler)(nil)
