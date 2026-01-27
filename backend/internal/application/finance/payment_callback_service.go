package finance

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/erp/backend/internal/domain/finance"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/telemetry"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	// ErrCallbackGatewayNotRegistered is returned when no gateway is registered for the gateway type
	ErrCallbackGatewayNotRegistered = errors.New("payment callback: gateway not registered")
	// ErrCallbackInvalidPayload is returned when the callback payload is invalid
	ErrCallbackInvalidPayload = errors.New("payment callback: invalid payload")
	// ErrCallbackVerificationFailed is returned when callback verification fails
	ErrCallbackVerificationFailed = errors.New("payment callback: signature verification failed")
	// ErrCallbackAlreadyProcessed is returned when a callback has already been processed
	ErrCallbackAlreadyProcessed = errors.New("payment callback: already processed")
	// ErrCallbackOrderNotFound is returned when the order for the callback is not found
	ErrCallbackOrderNotFound = errors.New("payment callback: order not found")
)

// PaymentCallbackService handles payment gateway callbacks
// It implements the PaymentCallbackHandler interface defined in the domain layer
type PaymentCallbackService struct {
	gateways           map[finance.PaymentGatewayType]finance.PaymentGateway
	receiptVoucherRepo finance.ReceiptVoucherRepository
	receivableRepo     finance.AccountReceivableRepository
	refundRecordRepo   finance.RefundRecordRepository
	eventPublisher     shared.EventPublisher
	reconciliationSvc  *finance.ReconciliationService
	logger             *zap.Logger
	processedCallbacks sync.Map // For idempotency checking
}

// PaymentCallbackServiceConfig holds configuration for the callback service
type PaymentCallbackServiceConfig struct {
	Gateways           []finance.PaymentGateway
	ReceiptVoucherRepo finance.ReceiptVoucherRepository
	ReceivableRepo     finance.AccountReceivableRepository
	RefundRecordRepo   finance.RefundRecordRepository
	EventPublisher     shared.EventPublisher
	Logger             *zap.Logger
}

// NewPaymentCallbackService creates a new PaymentCallbackService
func NewPaymentCallbackService(config PaymentCallbackServiceConfig) *PaymentCallbackService {
	gateways := make(map[finance.PaymentGatewayType]finance.PaymentGateway)
	for _, gw := range config.Gateways {
		gateways[gw.GatewayType()] = gw
	}

	logger := config.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &PaymentCallbackService{
		gateways:           gateways,
		receiptVoucherRepo: config.ReceiptVoucherRepo,
		receivableRepo:     config.ReceivableRepo,
		refundRecordRepo:   config.RefundRecordRepo,
		eventPublisher:     config.EventPublisher,
		reconciliationSvc:  finance.NewReconciliationService(),
		logger:             logger,
	}
}

// RegisterGateway registers a payment gateway for callback processing
func (s *PaymentCallbackService) RegisterGateway(gateway finance.PaymentGateway) {
	s.gateways[gateway.GatewayType()] = gateway
}

// GetGateway returns the gateway for a given type
func (s *PaymentCallbackService) GetGateway(gatewayType finance.PaymentGatewayType) (finance.PaymentGateway, error) {
	gw, ok := s.gateways[gatewayType]
	if !ok {
		return nil, ErrCallbackGatewayNotRegistered
	}
	return gw, nil
}

// ProcessPaymentCallback processes a raw payment callback from a gateway
func (s *PaymentCallbackService) ProcessPaymentCallback(
	ctx context.Context,
	gatewayType finance.PaymentGatewayType,
	payload []byte,
	signature string,
) (*PaymentCallbackResult, error) {
	// Start tracing span for payment callback processing
	ctx, span := telemetry.StartServiceSpan(ctx, "payment", "process_callback")
	defer span.End()

	telemetry.SetAttribute(span, telemetry.SpanAttrPaymentGateway, string(gatewayType))

	// Get the gateway
	gateway, err := s.GetGateway(gatewayType)
	if err != nil {
		telemetry.RecordError(span, err)
		s.logger.Error("Gateway not registered",
			zap.String("gateway_type", string(gatewayType)),
			zap.Error(err))
		return nil, err
	}

	// Verify and parse the callback
	callback, err := gateway.VerifyCallback(ctx, payload, signature)
	if err != nil {
		telemetry.RecordError(span, err)
		s.logger.Warn("Callback verification failed",
			zap.String("gateway_type", string(gatewayType)),
			zap.Error(err))
		return nil, fmt.Errorf("%w: %v", ErrCallbackVerificationFailed, err)
	}

	if callback == nil {
		telemetry.RecordError(span, ErrCallbackInvalidPayload)
		return nil, ErrCallbackInvalidPayload
	}

	// Add callback details to span
	telemetry.SetAttributes(span,
		"gateway_order_id", callback.GatewayOrderID,
		telemetry.SpanAttrOrderNumber, callback.OrderNumber,
		"payment_status", string(callback.Status),
		telemetry.SpanAttrAmount, callback.Amount.String(),
	)

	s.logger.Info("Payment callback received",
		zap.String("gateway_type", string(gatewayType)),
		zap.String("gateway_order_id", callback.GatewayOrderID),
		zap.String("order_number", callback.OrderNumber),
		zap.String("status", string(callback.Status)),
		zap.String("amount", callback.Amount.String()))

	// Check for idempotency using gateway transaction ID
	idempotencyKey := fmt.Sprintf("payment:%s:%s", gatewayType, callback.GatewayTransactionID)
	if _, loaded := s.processedCallbacks.LoadOrStore(idempotencyKey, time.Now()); loaded {
		s.logger.Info("Callback already processed (idempotency check)",
			zap.String("idempotency_key", idempotencyKey))
		telemetry.AddEvent(span, "callback_already_processed")
		return &PaymentCallbackResult{
			Success:          true,
			AlreadyProcessed: true,
			GatewayResponse:  gateway.GenerateCallbackResponse(true, ""),
		}, nil
	}

	// Handle the payment callback
	if err := s.HandlePaymentCallback(ctx, callback); err != nil {
		// Remove from processed on error to allow retry
		s.processedCallbacks.Delete(idempotencyKey)

		telemetry.RecordError(span, err)
		s.logger.Error("Failed to handle payment callback",
			zap.String("gateway_order_id", callback.GatewayOrderID),
			zap.Error(err))

		return &PaymentCallbackResult{
			Success:         false,
			Error:           err,
			GatewayResponse: gateway.GenerateCallbackResponse(false, err.Error()),
		}, err
	}

	telemetry.AddEvent(span, "payment_callback_processed",
		"gateway_transaction_id", callback.GatewayTransactionID,
	)

	return &PaymentCallbackResult{
		Success:         true,
		Callback:        callback,
		GatewayResponse: gateway.GenerateCallbackResponse(true, ""),
	}, nil
}

// HandlePaymentCallback processes a verified payment callback
// This implements the PaymentCallbackHandler interface
func (s *PaymentCallbackService) HandlePaymentCallback(ctx context.Context, callback *finance.PaymentCallback) error {
	// Start tracing span for payment callback handling
	ctx, span := telemetry.StartServiceSpan(ctx, "payment", "handle_callback")
	defer span.End()

	telemetry.SetAttributes(span,
		telemetry.SpanAttrOrderNumber, callback.OrderNumber,
		"payment_status", string(callback.Status),
		telemetry.SpanAttrAmount, callback.Amount.String(),
	)

	// Only process successful payments
	if !callback.Status.IsSuccess() {
		s.logger.Info("Skipping non-successful payment callback",
			zap.String("order_number", callback.OrderNumber),
			zap.String("status", string(callback.Status)))
		telemetry.AddEvent(span, "payment_not_successful", "status", string(callback.Status))
		return nil
	}

	// Find the receipt voucher by payment reference (order number)
	// Note: The order number from the callback should match the receipt voucher's payment reference
	voucher, err := s.receiptVoucherRepo.FindByPaymentReference(ctx, callback.OrderNumber)
	if err != nil {
		telemetry.RecordError(span, err)
		return fmt.Errorf("failed to find receipt voucher: %w", err)
	}
	if voucher == nil {
		s.logger.Warn("Receipt voucher not found for callback",
			zap.String("order_number", callback.OrderNumber))
		telemetry.RecordError(span, ErrCallbackOrderNotFound)
		return ErrCallbackOrderNotFound
	}

	telemetry.SetAttribute(span, "voucher_id", voucher.ID.String())

	// Check if already processed
	if voucher.Status == finance.VoucherStatusConfirmed {
		s.logger.Info("Receipt voucher already confirmed",
			zap.String("voucher_id", voucher.ID.String()),
			zap.String("order_number", callback.OrderNumber))
		telemetry.AddEvent(span, "voucher_already_confirmed")
		return nil
	}

	// Update the voucher with gateway transaction info
	if err := voucher.SetPaymentReference(callback.GatewayTransactionID); err != nil {
		s.logger.Warn("Failed to set payment reference", zap.Error(err))
	}

	// Auto-confirm the receipt voucher on successful payment
	systemUserID := uuid.Nil // System action
	if err := voucher.Confirm(systemUserID); err != nil {
		telemetry.RecordError(span, err)
		return fmt.Errorf("failed to confirm receipt voucher: %w", err)
	}

	// Save the updated voucher
	if err := s.receiptVoucherRepo.SaveWithLock(ctx, voucher); err != nil {
		telemetry.RecordError(span, err)
		return fmt.Errorf("failed to save receipt voucher: %w", err)
	}

	telemetry.AddEvent(span, "voucher_confirmed",
		"voucher_id", voucher.ID.String(),
		"voucher_number", voucher.VoucherNumber,
	)

	s.logger.Info("Receipt voucher confirmed via gateway callback",
		zap.String("voucher_id", voucher.ID.String()),
		zap.String("voucher_number", voucher.VoucherNumber),
		zap.String("gateway_transaction_id", callback.GatewayTransactionID))

	// Auto-reconcile with outstanding receivables if configured
	if s.receivableRepo != nil && s.reconciliationSvc != nil {
		if err := s.autoReconcile(ctx, voucher); err != nil {
			s.logger.Warn("Auto-reconciliation failed",
				zap.String("voucher_id", voucher.ID.String()),
				zap.Error(err))
			// Don't fail the callback for reconciliation errors
		}
	}

	// Publish payment completed event
	if s.eventPublisher != nil {
		event := finance.NewGatewayPaymentCompletedEvent(voucher.TenantID, callback)
		if err := s.eventPublisher.Publish(ctx, event); err != nil {
			s.logger.Warn("Failed to publish payment completed event",
				zap.String("voucher_id", voucher.ID.String()),
				zap.Error(err))
			// Don't fail the callback for event publishing errors
		}
	}

	return nil
}

// ProcessRefundCallback processes a raw refund callback from a gateway
func (s *PaymentCallbackService) ProcessRefundCallback(
	ctx context.Context,
	gatewayType finance.PaymentGatewayType,
	payload []byte,
	signature string,
) (*RefundCallbackResult, error) {
	// Get the gateway
	gateway, err := s.GetGateway(gatewayType)
	if err != nil {
		s.logger.Error("Gateway not registered",
			zap.String("gateway_type", string(gatewayType)),
			zap.Error(err))
		return nil, err
	}

	// Verify and parse the callback
	callback, err := gateway.VerifyRefundCallback(ctx, payload, signature)
	if err != nil {
		s.logger.Warn("Refund callback verification failed",
			zap.String("gateway_type", string(gatewayType)),
			zap.Error(err))
		return nil, fmt.Errorf("%w: %v", ErrCallbackVerificationFailed, err)
	}

	if callback == nil {
		return nil, ErrCallbackInvalidPayload
	}

	s.logger.Info("Refund callback received",
		zap.String("gateway_type", string(gatewayType)),
		zap.String("gateway_refund_id", callback.GatewayRefundID),
		zap.String("refund_number", callback.RefundNumber),
		zap.String("status", string(callback.Status)),
		zap.String("amount", callback.RefundAmount.String()))

	// Check for idempotency
	idempotencyKey := fmt.Sprintf("refund:%s:%s", gatewayType, callback.GatewayRefundID)
	if _, loaded := s.processedCallbacks.LoadOrStore(idempotencyKey, time.Now()); loaded {
		s.logger.Info("Refund callback already processed (idempotency check)",
			zap.String("idempotency_key", idempotencyKey))
		return &RefundCallbackResult{
			Success:          true,
			AlreadyProcessed: true,
			GatewayResponse:  gateway.GenerateCallbackResponse(true, ""),
		}, nil
	}

	// Handle the refund callback
	if err := s.HandleRefundCallback(ctx, callback); err != nil {
		// Remove from processed on error to allow retry
		s.processedCallbacks.Delete(idempotencyKey)

		s.logger.Error("Failed to handle refund callback",
			zap.String("gateway_refund_id", callback.GatewayRefundID),
			zap.Error(err))

		return &RefundCallbackResult{
			Success:         false,
			Error:           err,
			GatewayResponse: gateway.GenerateCallbackResponse(false, err.Error()),
		}, err
	}

	return &RefundCallbackResult{
		Success:         true,
		Callback:        callback,
		GatewayResponse: gateway.GenerateCallbackResponse(true, ""),
	}, nil
}

// HandleRefundCallback processes a verified refund callback
// This implements the PaymentCallbackHandler interface
func (s *PaymentCallbackService) HandleRefundCallback(ctx context.Context, callback *finance.RefundCallback) error {
	// Only process successful refunds
	if callback.Status != finance.RefundStatusSuccess {
		s.logger.Info("Skipping non-successful refund callback",
			zap.String("refund_number", callback.RefundNumber),
			zap.String("status", string(callback.Status)))
		return nil
	}

	s.logger.Info("Processing successful refund callback",
		zap.String("gateway_refund_id", callback.GatewayRefundID),
		zap.String("refund_number", callback.RefundNumber),
		zap.String("amount", callback.RefundAmount.String()))

	// Update or create refund record if repository is available
	if s.refundRecordRepo != nil {
		if err := s.updateOrCreateRefundRecord(ctx, callback); err != nil {
			s.logger.Warn("Failed to update refund record",
				zap.String("refund_number", callback.RefundNumber),
				zap.Error(err))
			// Don't fail the callback for record update errors
		}
	}

	// Publish refund completed event
	if s.eventPublisher != nil {
		// Use Nil UUID as tenant ID for now - in production this should be looked up from the refund record
		tenantID := uuid.Nil
		if s.refundRecordRepo != nil {
			// Try to get tenant ID from existing refund record
			if record, err := s.refundRecordRepo.FindByGatewayRefundID(ctx, callback.GatewayType, callback.GatewayRefundID); err == nil && record != nil {
				tenantID = record.TenantID
			}
		}
		event := finance.NewGatewayRefundCompletedEvent(tenantID, callback)
		if err := s.eventPublisher.Publish(ctx, event); err != nil {
			s.logger.Warn("Failed to publish refund completed event",
				zap.String("refund_number", callback.RefundNumber),
				zap.Error(err))
			// Don't fail the callback for event publishing errors
		}
	}

	return nil
}

// updateOrCreateRefundRecord updates an existing refund record or creates a new one from the callback
func (s *PaymentCallbackService) updateOrCreateRefundRecord(ctx context.Context, callback *finance.RefundCallback) error {
	// Try to find existing refund record by gateway refund ID
	existingRecord, err := s.refundRecordRepo.FindByGatewayRefundID(ctx, callback.GatewayType, callback.GatewayRefundID)
	if err != nil {
		return fmt.Errorf("failed to find existing refund record: %w", err)
	}

	if existingRecord != nil {
		// Update existing record with callback information
		if err := existingRecord.CompleteFromCallback(callback); err != nil {
			return fmt.Errorf("failed to complete refund record: %w", err)
		}

		if err := s.refundRecordRepo.SaveWithLock(ctx, existingRecord); err != nil {
			return fmt.Errorf("failed to save refund record: %w", err)
		}

		s.logger.Info("Refund record updated from callback",
			zap.String("refund_id", existingRecord.ID.String()),
			zap.String("refund_number", existingRecord.RefundNumber),
			zap.String("status", string(existingRecord.Status)))

		return nil
	}

	// Try to find by refund number if provided
	if callback.RefundNumber != "" {
		// We need tenant ID to search by refund number, but we don't have it from the callback
		// This is a limitation - we'll create a new record instead
		s.logger.Info("No existing refund record found by gateway refund ID, will create new record",
			zap.String("gateway_refund_id", callback.GatewayRefundID))
	}

	// Create a new refund record from the callback
	// Note: This creates a minimal record since we don't have full context from the callback
	// In production, refund records should be created when initiating the refund, not from callbacks

	// Generate a new refund number for the orphan callback record
	// Use a tenant ID of Nil for orphan records - these should be matched to the correct tenant later
	refundNumber := fmt.Sprintf("RF-CB-%s", callback.GatewayRefundID[:min(8, len(callback.GatewayRefundID))])

	newRecord, err := finance.NewRefundRecordFromCallback(uuid.Nil, refundNumber, callback)
	if err != nil {
		return fmt.Errorf("failed to create refund record from callback: %w", err)
	}

	if err := s.refundRecordRepo.Save(ctx, newRecord); err != nil {
		return fmt.Errorf("failed to save new refund record: %w", err)
	}

	s.logger.Info("New refund record created from callback",
		zap.String("refund_id", newRecord.ID.String()),
		zap.String("refund_number", newRecord.RefundNumber),
		zap.String("gateway_refund_id", callback.GatewayRefundID),
		zap.String("status", string(newRecord.Status)))

	return nil
}

// autoReconcile automatically reconciles the receipt voucher with outstanding receivables
func (s *PaymentCallbackService) autoReconcile(ctx context.Context, voucher *finance.ReceiptVoucher) error {
	// Get outstanding receivables for the customer
	receivables, err := s.receivableRepo.FindOutstanding(ctx, voucher.TenantID, voucher.CustomerID)
	if err != nil {
		return fmt.Errorf("failed to find outstanding receivables: %w", err)
	}

	if len(receivables) == 0 {
		s.logger.Info("No outstanding receivables for auto-reconciliation",
			zap.String("voucher_id", voucher.ID.String()),
			zap.String("customer_id", voucher.CustomerID.String()))
		return nil
	}

	// Use FIFO strategy for auto-reconciliation
	result, err := s.reconciliationSvc.ReconcileReceipt(ctx, finance.ReconcileReceiptRequest{
		ReceiptVoucher: voucher,
		Receivables:    receivables,
		StrategyType:   finance.ReconciliationStrategyTypeFIFO,
	})
	if err != nil {
		return fmt.Errorf("reconciliation failed: %w", err)
	}

	// Save updated voucher
	if err := s.receiptVoucherRepo.SaveWithLock(ctx, result.ReceiptVoucher); err != nil {
		return fmt.Errorf("failed to save voucher after reconciliation: %w", err)
	}

	// Save updated receivables
	for i := range result.UpdatedReceivables {
		if err := s.receivableRepo.SaveWithLock(ctx, &result.UpdatedReceivables[i]); err != nil {
			return fmt.Errorf("failed to save receivable after reconciliation: %w", err)
		}
	}

	s.logger.Info("Auto-reconciliation completed",
		zap.String("voucher_id", voucher.ID.String()),
		zap.String("total_reconciled", result.TotalReconciled.String()),
		zap.Bool("fully_reconciled", result.FullyReconciled))

	return nil
}

// PaymentCallbackResult represents the result of processing a payment callback
type PaymentCallbackResult struct {
	Success          bool                     `json:"success"`
	AlreadyProcessed bool                     `json:"already_processed,omitempty"`
	Callback         *finance.PaymentCallback `json:"callback,omitempty"`
	Error            error                    `json:"-"`
	GatewayResponse  []byte                   `json:"-"`
}

// RefundCallbackResult represents the result of processing a refund callback
type RefundCallbackResult struct {
	Success          bool                    `json:"success"`
	AlreadyProcessed bool                    `json:"already_processed,omitempty"`
	Callback         *finance.RefundCallback `json:"callback,omitempty"`
	Error            error                   `json:"-"`
	GatewayResponse  []byte                  `json:"-"`
}

// Ensure PaymentCallbackService implements the domain interface
var _ finance.PaymentCallbackHandler = (*PaymentCallbackService)(nil)
