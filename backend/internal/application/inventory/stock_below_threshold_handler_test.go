package inventory

import (
	"context"
	"sync"
	"testing"

	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// MockStockAlertNotifier is a mock notifier for testing
type MockStockAlertNotifier struct {
	mu     sync.Mutex
	alerts []StockAlert
}

func NewMockStockAlertNotifier() *MockStockAlertNotifier {
	return &MockStockAlertNotifier{
		alerts: make([]StockAlert, 0),
	}
}

func (n *MockStockAlertNotifier) SendAlert(ctx context.Context, alert StockAlert) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.alerts = append(n.alerts, alert)
	return nil
}

func (n *MockStockAlertNotifier) GetAlerts() []StockAlert {
	n.mu.Lock()
	defer n.mu.Unlock()
	result := make([]StockAlert, len(n.alerts))
	copy(result, n.alerts)
	return result
}

func (n *MockStockAlertNotifier) Reset() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.alerts = make([]StockAlert, 0)
}

func TestStockBelowThresholdHandler_Handle(t *testing.T) {
	logger := zaptest.NewLogger(t)
	notifier := NewMockStockAlertNotifier()

	handler := NewStockBelowThresholdHandler(logger).
		WithNotifier(notifier)

	tenantID := uuid.New()
	inventoryItemID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	t.Run("handles low stock event", func(t *testing.T) {
		notifier.Reset()

		event := &inventory.StockBelowThresholdEvent{
			BaseDomainEvent:   shared.NewBaseDomainEvent(inventory.EventTypeStockBelowThreshold, inventory.AggregateTypeInventoryItem, inventoryItemID, tenantID),
			InventoryItemID:   inventoryItemID,
			WarehouseID:       warehouseID,
			ProductID:         productID,
			CurrentQuantity:   decimal.NewFromInt(5),
			MinimumQuantity:   decimal.NewFromInt(10),
			AvailableQuantity: decimal.NewFromInt(3),
			LockedQuantity:    decimal.NewFromInt(2),
		}

		err := handler.Handle(context.Background(), event)
		require.NoError(t, err)

		alerts := notifier.GetAlerts()
		require.Len(t, alerts, 1)
		assert.Equal(t, "low_stock", alerts[0].AlertType)
		assert.Equal(t, tenantID.String(), alerts[0].TenantID)
		assert.Equal(t, productID.String(), alerts[0].ProductID)
		assert.Equal(t, "5", alerts[0].CurrentQuantity)
		assert.Equal(t, "10", alerts[0].MinimumQuantity)
	})

	t.Run("handles out of stock event", func(t *testing.T) {
		notifier.Reset()

		event := &inventory.StockBelowThresholdEvent{
			BaseDomainEvent:   shared.NewBaseDomainEvent(inventory.EventTypeStockBelowThreshold, inventory.AggregateTypeInventoryItem, inventoryItemID, tenantID),
			InventoryItemID:   inventoryItemID,
			WarehouseID:       warehouseID,
			ProductID:         productID,
			CurrentQuantity:   decimal.Zero,
			MinimumQuantity:   decimal.NewFromInt(10),
			AvailableQuantity: decimal.Zero,
			LockedQuantity:    decimal.Zero,
		}

		err := handler.Handle(context.Background(), event)
		require.NoError(t, err)

		alerts := notifier.GetAlerts()
		require.Len(t, alerts, 1)
		assert.Equal(t, "out_of_stock", alerts[0].AlertType)
	})

	t.Run("returns error for wrong event type", func(t *testing.T) {
		wrongEvent := &inventory.StockIncreasedEvent{}

		err := handler.Handle(context.Background(), wrongEvent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected event type")
	})
}

func TestStockBelowThresholdHandler_EventTypes(t *testing.T) {
	handler := NewStockBelowThresholdHandler(zap.NewNop())

	eventTypes := handler.EventTypes()
	assert.Len(t, eventTypes, 1)
	assert.Equal(t, inventory.EventTypeStockBelowThreshold, eventTypes[0])
}

func TestLoggingStockAlertNotifier_SendAlert(t *testing.T) {
	logger := zaptest.NewLogger(t)
	notifier := NewLoggingStockAlertNotifier(logger)

	alert := StockAlert{
		TenantID:          uuid.New().String(),
		InventoryItemID:   uuid.New().String(),
		WarehouseID:       uuid.New().String(),
		ProductID:         uuid.New().String(),
		CurrentQuantity:   "5",
		MinimumQuantity:   "10",
		AvailableQuantity: "3",
		LockedQuantity:    "2",
		AlertType:         "low_stock",
		Channels:          []string{"in_app"},
	}

	err := notifier.SendAlert(context.Background(), alert)
	assert.NoError(t, err)
}
