package catalog

import (
	"context"
	"fmt"

	"github.com/erp/backend/internal/domain/catalog"
	"github.com/erp/backend/internal/domain/shared"
	"go.uber.org/zap"
)

// ProductDisabledHandler handles ProductDisabledEvent
// and notifies other contexts when a product is disabled
type ProductDisabledHandler struct {
	logger   *zap.Logger
	notifier ProductDisabledNotifier
}

// ProductDisabledNotifier is the interface for notifying about disabled products
// Implementations can support different notification channels (in-app, webhook, etc.)
type ProductDisabledNotifier interface {
	// NotifyProductDisabled sends a notification when a product is disabled
	NotifyProductDisabled(ctx context.Context, notification ProductDisabledNotification) error
}

// ProductDisabledNotification represents a notification about a disabled product
type ProductDisabledNotification struct {
	TenantID   string `json:"tenant_id"`
	ProductID  string `json:"product_id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	CategoryID string `json:"category_id,omitempty"`
}

// NewProductDisabledHandler creates a new handler for product disabled events
func NewProductDisabledHandler(logger *zap.Logger) *ProductDisabledHandler {
	return &ProductDisabledHandler{
		logger: logger,
	}
}

// WithNotifier sets the notifier for sending notifications
func (h *ProductDisabledHandler) WithNotifier(notifier ProductDisabledNotifier) *ProductDisabledHandler {
	h.notifier = notifier
	return h
}

// EventTypes returns the event types this handler is interested in
func (h *ProductDisabledHandler) EventTypes() []string {
	return []string{catalog.EventTypeProductDisabled}
}

// Handle processes a ProductDisabledEvent
func (h *ProductDisabledHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	// Type assert to ProductDisabledEvent
	disabledEvent, ok := event.(*catalog.ProductDisabledEvent)
	if !ok {
		h.logger.Error("unexpected event type",
			zap.String("expected", catalog.EventTypeProductDisabled),
			zap.String("actual", event.EventType()),
		)
		return fmt.Errorf("unexpected event type: expected %s, got %s",
			catalog.EventTypeProductDisabled, event.EventType())
	}

	h.logger.Warn("product disabled",
		zap.String("tenant_id", event.TenantID().String()),
		zap.String("product_id", disabledEvent.ProductID.String()),
		zap.String("code", disabledEvent.Code),
		zap.String("name", disabledEvent.Name),
	)

	// Create notification
	categoryID := ""
	if disabledEvent.CategoryID != nil {
		categoryID = disabledEvent.CategoryID.String()
	}

	notification := ProductDisabledNotification{
		TenantID:   event.TenantID().String(),
		ProductID:  disabledEvent.ProductID.String(),
		Code:       disabledEvent.Code,
		Name:       disabledEvent.Name,
		CategoryID: categoryID,
	}

	// Send notification if notifier is configured
	if h.notifier != nil {
		if err := h.notifier.NotifyProductDisabled(ctx, notification); err != nil {
			h.logger.Error("failed to send product disabled notification",
				zap.String("product_id", notification.ProductID),
				zap.Error(err),
			)
			// Don't return error - notification failure shouldn't fail the event handling
		} else {
			h.logger.Info("product disabled notification sent",
				zap.String("product_id", notification.ProductID),
				zap.String("code", notification.Code),
			)
		}
	}

	return nil
}

// Ensure ProductDisabledHandler implements shared.EventHandler
var _ shared.EventHandler = (*ProductDisabledHandler)(nil)

// LoggingProductDisabledNotifier is a simple notifier that logs notifications
// This is useful for development and testing
type LoggingProductDisabledNotifier struct {
	logger *zap.Logger
}

// NewLoggingProductDisabledNotifier creates a new logging notifier
func NewLoggingProductDisabledNotifier(logger *zap.Logger) *LoggingProductDisabledNotifier {
	return &LoggingProductDisabledNotifier{
		logger: logger,
	}
}

// NotifyProductDisabled logs the product disabled notification
func (n *LoggingProductDisabledNotifier) NotifyProductDisabled(ctx context.Context, notification ProductDisabledNotification) error {
	n.logger.Warn("PRODUCT DISABLED",
		zap.String("product_id", notification.ProductID),
		zap.String("code", notification.Code),
		zap.String("name", notification.Name),
	)
	return nil
}

// Ensure LoggingProductDisabledNotifier implements ProductDisabledNotifier
var _ ProductDisabledNotifier = (*LoggingProductDisabledNotifier)(nil)
