package event

import (
	"context"
	"sync/atomic"

	"github.com/erp/backend/internal/domain/shared"
	"go.uber.org/zap"
)

// IdempotencyMetrics tracks idempotency-related statistics
type IdempotencyMetrics struct {
	// EventsProcessed is the total number of events processed (first time)
	EventsProcessed atomic.Int64

	// EventsDuplicate is the total number of duplicate events detected
	EventsDuplicate atomic.Int64

	// EventsFailed is the total number of events that failed to process
	EventsFailed atomic.Int64
}

// Stats returns a snapshot of the current metrics
func (m *IdempotencyMetrics) Stats() IdempotencyStats {
	return IdempotencyStats{
		EventsProcessed: m.EventsProcessed.Load(),
		EventsDuplicate: m.EventsDuplicate.Load(),
		EventsFailed:    m.EventsFailed.Load(),
	}
}

// IdempotencyStats is a snapshot of idempotency metrics
type IdempotencyStats struct {
	EventsProcessed int64 `json:"events_processed"`
	EventsDuplicate int64 `json:"events_duplicate"`
	EventsFailed    int64 `json:"events_failed"`
}

// IdempotentHandler wraps an EventHandler with idempotency checking
// It ensures each event is only processed once, even if delivered multiple times
type IdempotentHandler struct {
	handler shared.EventHandler
	store   shared.IdempotencyStore
	config  shared.IdempotencyConfig
	logger  *zap.Logger
	metrics *IdempotencyMetrics
}

// IdempotentHandlerOption is a functional option for IdempotentHandler
type IdempotentHandlerOption func(*IdempotentHandler)

// WithIdempotencyConfig sets the idempotency configuration
func WithIdempotencyConfig(config shared.IdempotencyConfig) IdempotentHandlerOption {
	return func(h *IdempotentHandler) {
		h.config = config
	}
}

// WithIdempotencyMetrics sets the metrics collector
func WithIdempotencyMetrics(metrics *IdempotencyMetrics) IdempotentHandlerOption {
	return func(h *IdempotentHandler) {
		h.metrics = metrics
	}
}

// NewIdempotentHandler creates a new idempotent handler wrapper
func NewIdempotentHandler(
	handler shared.EventHandler,
	store shared.IdempotencyStore,
	logger *zap.Logger,
	opts ...IdempotentHandlerOption,
) *IdempotentHandler {
	h := &IdempotentHandler{
		handler: handler,
		store:   store,
		config:  shared.DefaultIdempotencyConfig(),
		logger:  logger,
		metrics: &IdempotencyMetrics{},
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// EventTypes returns the event types this handler is interested in
func (h *IdempotentHandler) EventTypes() []string {
	return h.handler.EventTypes()
}

// Handle processes the event with idempotency checking
func (h *IdempotentHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	// If idempotency is disabled, process directly
	if !h.config.Enabled {
		return h.handler.Handle(ctx, event)
	}

	eventID := event.EventID().String()

	// Try to mark as processed (atomic check-and-set)
	isNew, err := h.store.MarkProcessed(ctx, eventID, h.config.TTL)
	if err != nil {
		// Log warning but continue processing
		// Better to risk duplicate processing than to drop events
		h.logger.Warn("failed to check idempotency, processing anyway",
			zap.String("event_id", eventID),
			zap.String("event_type", event.EventType()),
			zap.Error(err),
		)
		// Fall through to process the event
	} else if !isNew {
		// Event was already processed
		h.metrics.EventsDuplicate.Add(1)
		h.logger.Debug("duplicate event detected, skipping",
			zap.String("event_id", eventID),
			zap.String("event_type", event.EventType()),
		)
		return nil // Success - idempotent response
	}

	// Process the event
	if err := h.handler.Handle(ctx, event); err != nil {
		h.metrics.EventsFailed.Add(1)
		h.logger.Error("event handler failed",
			zap.String("event_id", eventID),
			zap.String("event_type", event.EventType()),
			zap.Error(err),
		)
		// Note: We don't remove the idempotency key on failure
		// This prevents rapid retries. The key will expire after TTL
		// allowing retry after a cooldown period
		return err
	}

	h.metrics.EventsProcessed.Add(1)
	h.logger.Debug("event processed successfully",
		zap.String("event_id", eventID),
		zap.String("event_type", event.EventType()),
	)

	return nil
}

// GetMetrics returns the metrics for this handler
func (h *IdempotentHandler) GetMetrics() *IdempotencyMetrics {
	return h.metrics
}

// GetWrappedHandler returns the underlying handler (useful for testing)
func (h *IdempotentHandler) GetWrappedHandler() shared.EventHandler {
	return h.handler
}

// Ensure IdempotentHandler implements EventHandler
var _ shared.EventHandler = (*IdempotentHandler)(nil)

// WrapHandlersWithIdempotency wraps multiple handlers with idempotency checking
// This is a convenience function for wrapping all handlers at once
func WrapHandlersWithIdempotency(
	handlers []shared.EventHandler,
	store shared.IdempotencyStore,
	logger *zap.Logger,
	opts ...IdempotentHandlerOption,
) []shared.EventHandler {
	wrapped := make([]shared.EventHandler, len(handlers))
	for i, h := range handlers {
		wrapped[i] = NewIdempotentHandler(h, store, logger, opts...)
	}
	return wrapped
}

// GlobalIdempotencyMetrics provides a shared metrics instance for aggregating
// statistics across all idempotent handlers in the application.
//
// Usage: Pass this to handlers that should share metrics using WithIdempotencyMetrics():
//
//	handler := NewIdempotentHandler(h, store, logger,
//	    WithIdempotencyMetrics(GlobalIdempotencyMetrics),
//	)
//
// NOTE: This is intentionally global for convenience in single-application deployments.
// For multi-tenant or more complex scenarios, inject a custom metrics instance instead.
var GlobalIdempotencyMetrics = &IdempotencyMetrics{}
