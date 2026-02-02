package printing

import (
	"context"
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/finance"
	"github.com/erp/backend/internal/domain/printing"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	// autoPrintIdempotencyKeyPrefix is the prefix for idempotency keys
	autoPrintIdempotencyKeyPrefix = "print:"
	// autoPrintIdempotencyTTL is the TTL for idempotency keys (24 hours)
	autoPrintIdempotencyTTL = 24 * time.Hour
)

// Finance event type constants (defined locally since finance package doesn't export them)
const (
	eventTypeReceiptVoucherCreated   = "ReceiptVoucherCreated"
	eventTypeReceiptVoucherConfirmed = "ReceiptVoucherConfirmed"
	eventTypePaymentVoucherCreated   = "PaymentVoucherCreated"
	eventTypePaymentVoucherConfirmed = "PaymentVoucherConfirmed"
)

// SystemUserID is a well-known UUID used for system-initiated operations like auto-print.
// This allows auditing to distinguish between user-initiated and system-initiated print jobs.
var SystemUserID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// AutoPrintHandler handles domain events and triggers automatic printing based on configured rules.
// It subscribes to business events (order confirmed, shipped, etc.) and creates print jobs
// when matching auto-print rules are found.
type AutoPrintHandler struct {
	ruleRepo     printing.AutoPrintRuleRepository
	printService *PrintService
	redisClient  *redis.Client
	logger       *zap.Logger
}

// NewAutoPrintHandler creates a new AutoPrintHandler.
// ruleRepo and printService are required dependencies.
// redisClient is optional - if nil, idempotency checking is skipped.
// logger is optional - if nil, a no-op logger is used.
func NewAutoPrintHandler(
	ruleRepo printing.AutoPrintRuleRepository,
	printService *PrintService,
	redisClient *redis.Client,
	logger *zap.Logger,
) *AutoPrintHandler {
	if ruleRepo == nil {
		panic("AutoPrintHandler: ruleRepo is required")
	}
	if printService == nil {
		panic("AutoPrintHandler: printService is required")
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &AutoPrintHandler{
		ruleRepo:     ruleRepo,
		printService: printService,
		redisClient:  redisClient,
		logger:       logger,
	}
}

// EventTypes returns the event types this handler is interested in
func (h *AutoPrintHandler) EventTypes() []string {
	return []string{
		// Sales Order events
		trade.EventTypeSalesOrderCreated,
		trade.EventTypeSalesOrderConfirmed,
		trade.EventTypeSalesOrderShipped,
		trade.EventTypeSalesOrderCompleted,
		// Purchase Order events
		trade.EventTypePurchaseOrderCreated,
		trade.EventTypePurchaseOrderConfirmed,
		trade.EventTypePurchaseOrderReceived,
		trade.EventTypePurchaseOrderCompleted,
		// Sales Return events
		trade.EventTypeSalesReturnCreated,
		trade.EventTypeSalesReturnApproved, // "Approved" maps to "Confirmed" trigger
		trade.EventTypeSalesReturnCompleted,
		// Purchase Return events
		trade.EventTypePurchaseReturnCreated,
		trade.EventTypePurchaseReturnApproved, // "Approved" maps to "Confirmed" trigger
		trade.EventTypePurchaseReturnCompleted,
		// Receipt Voucher events
		eventTypeReceiptVoucherCreated,
		eventTypeReceiptVoucherConfirmed,
		// Payment Voucher events
		eventTypePaymentVoucherCreated,
		eventTypePaymentVoucherConfirmed,
	}
}

// Handle processes a domain event and triggers auto-printing if applicable
func (h *AutoPrintHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	// Map the event to document type and trigger event
	docType, triggerEvent, docInfo := h.mapEventToTrigger(event)
	if docType == "" || triggerEvent == "" {
		// Event not applicable for auto-printing
		h.logger.Debug("event not applicable for auto-printing",
			zap.String("event_type", event.EventType()),
		)
		return nil
	}

	tenantID := event.TenantID()

	h.logger.Info("processing auto-print trigger",
		zap.String("event_type", event.EventType()),
		zap.String("doc_type", string(docType)),
		zap.String("trigger_event", string(triggerEvent)),
		zap.String("document_id", docInfo.DocumentID.String()),
		zap.String("document_number", docInfo.DocumentNumber),
		zap.String("tenant_id", tenantID.String()),
	)

	// Look up auto-print rule for this document type and trigger event
	// We check the rule BEFORE idempotency to avoid polluting Redis with keys
	// for events that have no matching rules
	rule, err := h.ruleRepo.FindEnabledByDocTypeAndEvent(ctx, tenantID, docType, triggerEvent)
	if err != nil {
		if err == shared.ErrNotFound {
			h.logger.Debug("no auto-print rule found",
				zap.String("doc_type", string(docType)),
				zap.String("trigger_event", string(triggerEvent)),
			)
			return nil
		}
		h.logger.Error("failed to lookup auto-print rule",
			zap.String("doc_type", string(docType)),
			zap.String("trigger_event", string(triggerEvent)),
			zap.Error(err),
		)
		// Don't return error - auto-print failure should not block business flow
		return nil
	}

	if rule == nil {
		// Repository returned (nil, nil) which is unexpected but handled gracefully
		h.logger.Debug("no enabled auto-print rule found",
			zap.String("doc_type", string(docType)),
			zap.String("trigger_event", string(triggerEvent)),
		)
		return nil
	}

	// Check if auto-print is enabled for this rule
	if !rule.AutoPrint {
		h.logger.Debug("auto-print disabled for rule",
			zap.String("rule_id", rule.ID.String()),
			zap.String("doc_type", string(docType)),
			zap.String("trigger_event", string(triggerEvent)),
		)
		return nil
	}

	// Check idempotency - prevent duplicate print jobs for the same event
	// This is done AFTER rule lookup to avoid Redis key pollution for events without rules
	idempotencyKey := h.buildIdempotencyKey(docType, docInfo.DocumentID, triggerEvent)
	isNew, err := h.checkIdempotency(ctx, idempotencyKey)
	if err != nil {
		// Log warning but continue - better to risk duplicate than miss a print
		h.logger.Warn("idempotency check failed, continuing anyway",
			zap.String("key", idempotencyKey),
			zap.Error(err),
		)
	} else if !isNew {
		h.logger.Debug("duplicate auto-print trigger detected, skipping",
			zap.String("key", idempotencyKey),
		)
		return nil
	}

	// Create print job
	err = h.createPrintJob(ctx, tenantID, rule, docInfo)
	if err != nil {
		h.logger.Error("failed to create auto-print job",
			zap.String("rule_id", rule.ID.String()),
			zap.String("document_id", docInfo.DocumentID.String()),
			zap.Error(err),
		)
		// Don't return error - auto-print failure should not block business flow
		return nil
	}

	h.logger.Info("auto-print job created successfully",
		zap.String("rule_id", rule.ID.String()),
		zap.String("doc_type", string(docType)),
		zap.String("trigger_event", string(triggerEvent)),
		zap.String("document_id", docInfo.DocumentID.String()),
		zap.String("document_number", docInfo.DocumentNumber),
	)

	return nil
}

// documentInfo holds information extracted from domain events
type documentInfo struct {
	DocumentID     uuid.UUID
	DocumentNumber string
}

// mapEventToTrigger maps a domain event to document type and trigger event
func (h *AutoPrintHandler) mapEventToTrigger(event shared.DomainEvent) (printing.DocType, printing.TriggerEvent, documentInfo) {
	var docInfo documentInfo

	switch e := event.(type) {
	// Sales Order events
	case *trade.SalesOrderCreatedEvent:
		docInfo = documentInfo{DocumentID: e.OrderID, DocumentNumber: e.OrderNumber}
		return printing.DocTypeSalesOrder, printing.TriggerEventCreated, docInfo
	case *trade.SalesOrderConfirmedEvent:
		docInfo = documentInfo{DocumentID: e.OrderID, DocumentNumber: e.OrderNumber}
		return printing.DocTypeSalesOrder, printing.TriggerEventConfirmed, docInfo
	case *trade.SalesOrderShippedEvent:
		docInfo = documentInfo{DocumentID: e.OrderID, DocumentNumber: e.OrderNumber}
		return printing.DocTypeSalesOrder, printing.TriggerEventShipped, docInfo
	case *trade.SalesOrderCompletedEvent:
		docInfo = documentInfo{DocumentID: e.OrderID, DocumentNumber: e.OrderNumber}
		return printing.DocTypeSalesOrder, printing.TriggerEventCompleted, docInfo

	// Purchase Order events
	case *trade.PurchaseOrderCreatedEvent:
		docInfo = documentInfo{DocumentID: e.OrderID, DocumentNumber: e.OrderNumber}
		return printing.DocTypePurchaseOrder, printing.TriggerEventCreated, docInfo
	case *trade.PurchaseOrderConfirmedEvent:
		docInfo = documentInfo{DocumentID: e.OrderID, DocumentNumber: e.OrderNumber}
		return printing.DocTypePurchaseOrder, printing.TriggerEventConfirmed, docInfo
	case *trade.PurchaseOrderReceivedEvent:
		docInfo = documentInfo{DocumentID: e.OrderID, DocumentNumber: e.OrderNumber}
		return printing.DocTypePurchaseOrder, printing.TriggerEventReceived, docInfo
	case *trade.PurchaseOrderCompletedEvent:
		docInfo = documentInfo{DocumentID: e.OrderID, DocumentNumber: e.OrderNumber}
		return printing.DocTypePurchaseOrder, printing.TriggerEventCompleted, docInfo

	// Sales Return events
	case *trade.SalesReturnCreatedEvent:
		docInfo = documentInfo{DocumentID: e.ReturnID, DocumentNumber: e.ReturnNumber}
		return printing.DocTypeSalesReturn, printing.TriggerEventCreated, docInfo
	case *trade.SalesReturnApprovedEvent:
		// "Approved" maps to "Confirmed" trigger for returns
		docInfo = documentInfo{DocumentID: e.ReturnID, DocumentNumber: e.ReturnNumber}
		return printing.DocTypeSalesReturn, printing.TriggerEventConfirmed, docInfo
	case *trade.SalesReturnCompletedEvent:
		docInfo = documentInfo{DocumentID: e.ReturnID, DocumentNumber: e.ReturnNumber}
		return printing.DocTypeSalesReturn, printing.TriggerEventCompleted, docInfo

	// Purchase Return events
	case *trade.PurchaseReturnCreatedEvent:
		docInfo = documentInfo{DocumentID: e.ReturnID, DocumentNumber: e.ReturnNumber}
		return printing.DocTypePurchaseReturn, printing.TriggerEventCreated, docInfo
	case *trade.PurchaseReturnApprovedEvent:
		// "Approved" maps to "Confirmed" trigger for returns
		docInfo = documentInfo{DocumentID: e.ReturnID, DocumentNumber: e.ReturnNumber}
		return printing.DocTypePurchaseReturn, printing.TriggerEventConfirmed, docInfo
	case *trade.PurchaseReturnCompletedEvent:
		docInfo = documentInfo{DocumentID: e.ReturnID, DocumentNumber: e.ReturnNumber}
		return printing.DocTypePurchaseReturn, printing.TriggerEventCompleted, docInfo

	// Receipt Voucher events
	case *finance.ReceiptVoucherCreatedEvent:
		docInfo = documentInfo{DocumentID: e.VoucherID, DocumentNumber: e.VoucherNumber}
		return printing.DocTypeReceiptVoucher, printing.TriggerEventCreated, docInfo
	case *finance.ReceiptVoucherConfirmedEvent:
		docInfo = documentInfo{DocumentID: e.VoucherID, DocumentNumber: e.VoucherNumber}
		return printing.DocTypeReceiptVoucher, printing.TriggerEventConfirmed, docInfo

	// Payment Voucher events
	case *finance.PaymentVoucherCreatedEvent:
		docInfo = documentInfo{DocumentID: e.VoucherID, DocumentNumber: e.VoucherNumber}
		return printing.DocTypePaymentVoucher, printing.TriggerEventCreated, docInfo
	case *finance.PaymentVoucherConfirmedEvent:
		docInfo = documentInfo{DocumentID: e.VoucherID, DocumentNumber: e.VoucherNumber}
		return printing.DocTypePaymentVoucher, printing.TriggerEventConfirmed, docInfo

	default:
		return "", "", documentInfo{}
	}
}

// buildIdempotencyKey builds the idempotency key for auto-print
// Format: print:{docType}:{docID}:{trigger}
func (h *AutoPrintHandler) buildIdempotencyKey(docType printing.DocType, docID uuid.UUID, trigger printing.TriggerEvent) string {
	return fmt.Sprintf("%s%s:%s:%s", autoPrintIdempotencyKeyPrefix, docType, docID.String(), trigger)
}

// checkIdempotency checks if this auto-print trigger has already been processed
// Returns true if this is a new trigger, false if it's a duplicate
func (h *AutoPrintHandler) checkIdempotency(ctx context.Context, key string) (bool, error) {
	if h.redisClient == nil {
		// If Redis is not configured, skip idempotency check
		return true, nil
	}

	// Use SETNX with TTL for atomic check-and-set
	result, err := h.redisClient.SetNX(ctx, key, "1", autoPrintIdempotencyTTL).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check idempotency: %w", err)
	}

	return result, nil
}

// createPrintJob creates a print job based on the auto-print rule
func (h *AutoPrintHandler) createPrintJob(ctx context.Context, tenantID uuid.UUID, rule *printing.AutoPrintRule, docInfo documentInfo) error {
	// Determine template ID to use
	var templateID *uuid.UUID
	if rule.TemplateID != nil {
		templateID = rule.TemplateID
	}

	// Create the print job request
	req := GeneratePDFRequest{
		DocumentType:   string(rule.DocumentType),
		DocumentID:     docInfo.DocumentID,
		DocumentNumber: docInfo.DocumentNumber,
		TemplateID:     templateID,
		Copies:         &rule.Copies,
	}

	// Generate the PDF using the system user ID for auto-print jobs
	_, err := h.printService.GeneratePDF(ctx, tenantID, SystemUserID, req)
	if err != nil {
		return fmt.Errorf("failed to generate PDF: %w", err)
	}

	return nil
}

// RegisterWithEventBus registers the handler with an event bus
func (h *AutoPrintHandler) RegisterWithEventBus(bus shared.EventSubscriber) {
	bus.Subscribe(h)
	h.logger.Info("AutoPrintHandler registered with event bus",
		zap.Strings("event_types", h.EventTypes()),
	)
}

// Ensure AutoPrintHandler implements EventHandler
var _ shared.EventHandler = (*AutoPrintHandler)(nil)
