package inventory

import (
	"context"
	"fmt"

	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/shared"
	"go.uber.org/zap"
)

// StockBelowThresholdHandler handles StockBelowThreshold events
// and triggers notifications/alerts when stock falls below minimum threshold
type StockBelowThresholdHandler struct {
	logger       *zap.Logger
	notifier     StockAlertNotifier
	alertService StockAlertService
}

// StockAlertNotifier is the interface for sending stock alerts
// Implementations can support different channels (in-app, email, SMS, etc.)
type StockAlertNotifier interface {
	// SendAlert sends a stock alert notification
	SendAlert(ctx context.Context, alert StockAlert) error
}

// StockAlertService provides access to alert configuration and persistence
type StockAlertService interface {
	// GetAlertConfig returns the alert configuration for a specific inventory item
	GetAlertConfig(ctx context.Context, tenantID, inventoryItemID string) (*AlertConfig, error)
	// RecordAlert records that an alert was sent
	RecordAlert(ctx context.Context, alert StockAlert) error
}

// StockAlert represents a stock level alert
type StockAlert struct {
	TenantID          string   `json:"tenant_id"`
	InventoryItemID   string   `json:"inventory_item_id"`
	WarehouseID       string   `json:"warehouse_id"`
	ProductID         string   `json:"product_id"`
	CurrentQuantity   string   `json:"current_quantity"`
	MinimumQuantity   string   `json:"minimum_quantity"`
	AvailableQuantity string   `json:"available_quantity"`
	LockedQuantity    string   `json:"locked_quantity"`
	AlertType         string   `json:"alert_type"` // "low_stock", "out_of_stock"
	Channels          []string `json:"channels"`   // "in_app", "email", "sms"
}

// AlertConfig contains configuration for stock alerts
type AlertConfig struct {
	Enabled         bool     `json:"enabled"`
	Channels        []string `json:"channels"`             // Notification channels to use
	MinInterval     int      `json:"min_interval_seconds"` // Minimum seconds between alerts
	EmailRecipients []string `json:"email_recipients,omitempty"`
}

// NewStockBelowThresholdHandler creates a new handler for stock below threshold events
func NewStockBelowThresholdHandler(logger *zap.Logger) *StockBelowThresholdHandler {
	return &StockBelowThresholdHandler{
		logger: logger,
	}
}

// WithNotifier sets the notifier for sending alerts
func (h *StockBelowThresholdHandler) WithNotifier(notifier StockAlertNotifier) *StockBelowThresholdHandler {
	h.notifier = notifier
	return h
}

// WithAlertService sets the alert service for configuration and persistence
func (h *StockBelowThresholdHandler) WithAlertService(service StockAlertService) *StockBelowThresholdHandler {
	h.alertService = service
	return h
}

// EventTypes returns the event types this handler is interested in
func (h *StockBelowThresholdHandler) EventTypes() []string {
	return []string{inventory.EventTypeStockBelowThreshold}
}

// Handle processes a StockBelowThresholdEvent
func (h *StockBelowThresholdHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	// Type assert to StockBelowThresholdEvent
	thresholdEvent, ok := event.(*inventory.StockBelowThresholdEvent)
	if !ok {
		h.logger.Error("unexpected event type",
			zap.String("expected", inventory.EventTypeStockBelowThreshold),
			zap.String("actual", event.EventType()),
		)
		return fmt.Errorf("unexpected event type: expected %s, got %s",
			inventory.EventTypeStockBelowThreshold, event.EventType())
	}

	h.logger.Warn("stock below threshold detected",
		zap.String("tenant_id", event.TenantID().String()),
		zap.String("inventory_item_id", thresholdEvent.InventoryItemID.String()),
		zap.String("warehouse_id", thresholdEvent.WarehouseID.String()),
		zap.String("product_id", thresholdEvent.ProductID.String()),
		zap.String("current_quantity", thresholdEvent.CurrentQuantity.String()),
		zap.String("minimum_quantity", thresholdEvent.MinimumQuantity.String()),
		zap.String("available_quantity", thresholdEvent.AvailableQuantity.String()),
		zap.String("locked_quantity", thresholdEvent.LockedQuantity.String()),
	)

	// Determine alert type based on current quantity
	alertType := "low_stock"
	if thresholdEvent.CurrentQuantity.IsZero() {
		alertType = "out_of_stock"
	}

	// Create alert
	alert := StockAlert{
		TenantID:          event.TenantID().String(),
		InventoryItemID:   thresholdEvent.InventoryItemID.String(),
		WarehouseID:       thresholdEvent.WarehouseID.String(),
		ProductID:         thresholdEvent.ProductID.String(),
		CurrentQuantity:   thresholdEvent.CurrentQuantity.String(),
		MinimumQuantity:   thresholdEvent.MinimumQuantity.String(),
		AvailableQuantity: thresholdEvent.AvailableQuantity.String(),
		LockedQuantity:    thresholdEvent.LockedQuantity.String(),
		AlertType:         alertType,
		Channels:          []string{"in_app"}, // Default to in-app notifications
	}

	// Get alert configuration if service is available
	if h.alertService != nil {
		config, err := h.alertService.GetAlertConfig(ctx, alert.TenantID, alert.InventoryItemID)
		if err != nil {
			h.logger.Debug("failed to get alert config, using defaults",
				zap.Error(err),
			)
		} else if config != nil && !config.Enabled {
			h.logger.Debug("alerts disabled for this inventory item",
				zap.String("inventory_item_id", alert.InventoryItemID),
			)
			return nil
		} else if config != nil && len(config.Channels) > 0 {
			alert.Channels = config.Channels
		}
	}

	// Send notification if notifier is configured
	if h.notifier != nil {
		if err := h.notifier.SendAlert(ctx, alert); err != nil {
			h.logger.Error("failed to send stock alert notification",
				zap.String("inventory_item_id", alert.InventoryItemID),
				zap.Error(err),
			)
			// Don't return error - notification failure shouldn't fail the event handling
		} else {
			h.logger.Info("stock alert notification sent",
				zap.String("inventory_item_id", alert.InventoryItemID),
				zap.String("alert_type", alertType),
				zap.Strings("channels", alert.Channels),
			)
		}
	}

	// Record the alert if service is available
	if h.alertService != nil {
		if err := h.alertService.RecordAlert(ctx, alert); err != nil {
			h.logger.Error("failed to record stock alert",
				zap.Error(err),
			)
			// Don't return error - recording failure shouldn't fail the event handling
		}
	}

	return nil
}

// Ensure StockBelowThresholdHandler implements shared.EventHandler
var _ shared.EventHandler = (*StockBelowThresholdHandler)(nil)

// LoggingStockAlertNotifier is a simple notifier that logs alerts
// This is useful for development and testing
type LoggingStockAlertNotifier struct {
	logger *zap.Logger
}

// NewLoggingStockAlertNotifier creates a new logging notifier
func NewLoggingStockAlertNotifier(logger *zap.Logger) *LoggingStockAlertNotifier {
	return &LoggingStockAlertNotifier{
		logger: logger,
	}
}

// SendAlert logs the stock alert
func (n *LoggingStockAlertNotifier) SendAlert(ctx context.Context, alert StockAlert) error {
	n.logger.Warn("STOCK ALERT",
		zap.String("type", alert.AlertType),
		zap.String("product_id", alert.ProductID),
		zap.String("warehouse_id", alert.WarehouseID),
		zap.String("current_qty", alert.CurrentQuantity),
		zap.String("minimum_qty", alert.MinimumQuantity),
		zap.Strings("channels", alert.Channels),
	)
	return nil
}

// Ensure LoggingStockAlertNotifier implements StockAlertNotifier
var _ StockAlertNotifier = (*LoggingStockAlertNotifier)(nil)
