package catalog

import (
	"context"
	"testing"

	"github.com/erp/backend/internal/domain/catalog"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockProductDisabledNotifier is a mock implementation of ProductDisabledNotifier
type mockProductDisabledNotifier struct {
	notifications []ProductDisabledNotification
	returnError   error
}

func (m *mockProductDisabledNotifier) NotifyProductDisabled(ctx context.Context, notification ProductDisabledNotification) error {
	if m.returnError != nil {
		return m.returnError
	}
	m.notifications = append(m.notifications, notification)
	return nil
}

func TestProductDisabledHandler_EventTypes(t *testing.T) {
	logger := zap.NewNop()
	handler := NewProductDisabledHandler(logger)

	eventTypes := handler.EventTypes()
	require.Len(t, eventTypes, 1)
	assert.Equal(t, catalog.EventTypeProductDisabled, eventTypes[0])
}

func TestProductDisabledHandler_Handle(t *testing.T) {
	logger := zap.NewNop()

	t.Run("handles ProductDisabledEvent successfully", func(t *testing.T) {
		notifier := &mockProductDisabledNotifier{}
		handler := NewProductDisabledHandler(logger).WithNotifier(notifier)

		tenantID := uuid.New()
		productID := uuid.New()
		categoryID := uuid.New()

		event := &catalog.ProductDisabledEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				catalog.EventTypeProductDisabled,
				catalog.AggregateTypeProduct,
				productID,
				tenantID,
			),
			ProductID:  productID,
			Code:       "SKU-001",
			Name:       "Test Product",
			CategoryID: &categoryID,
		}

		err := handler.Handle(context.Background(), event)
		require.NoError(t, err)

		// Verify notification was sent
		require.Len(t, notifier.notifications, 1)
		notification := notifier.notifications[0]
		assert.Equal(t, tenantID.String(), notification.TenantID)
		assert.Equal(t, productID.String(), notification.ProductID)
		assert.Equal(t, "SKU-001", notification.Code)
		assert.Equal(t, "Test Product", notification.Name)
		assert.Equal(t, categoryID.String(), notification.CategoryID)
	})

	t.Run("handles event without category", func(t *testing.T) {
		notifier := &mockProductDisabledNotifier{}
		handler := NewProductDisabledHandler(logger).WithNotifier(notifier)

		tenantID := uuid.New()
		productID := uuid.New()

		event := &catalog.ProductDisabledEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				catalog.EventTypeProductDisabled,
				catalog.AggregateTypeProduct,
				productID,
				tenantID,
			),
			ProductID:  productID,
			Code:       "SKU-002",
			Name:       "Product Without Category",
			CategoryID: nil,
		}

		err := handler.Handle(context.Background(), event)
		require.NoError(t, err)

		require.Len(t, notifier.notifications, 1)
		assert.Equal(t, "", notifier.notifications[0].CategoryID)
	})

	t.Run("handles without notifier configured", func(t *testing.T) {
		handler := NewProductDisabledHandler(logger) // No notifier set

		tenantID := uuid.New()
		productID := uuid.New()

		event := &catalog.ProductDisabledEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				catalog.EventTypeProductDisabled,
				catalog.AggregateTypeProduct,
				productID,
				tenantID,
			),
			ProductID: productID,
			Code:      "SKU-003",
			Name:      "Product",
		}

		err := handler.Handle(context.Background(), event)
		require.NoError(t, err) // Should not fail even without notifier
	})

	t.Run("continues on notification error", func(t *testing.T) {
		notifier := &mockProductDisabledNotifier{
			returnError: assert.AnError,
		}
		handler := NewProductDisabledHandler(logger).WithNotifier(notifier)

		tenantID := uuid.New()
		productID := uuid.New()

		event := &catalog.ProductDisabledEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				catalog.EventTypeProductDisabled,
				catalog.AggregateTypeProduct,
				productID,
				tenantID,
			),
			ProductID: productID,
			Code:      "SKU-004",
			Name:      "Product",
		}

		// Should not return error - notification failure is logged but doesn't fail handling
		err := handler.Handle(context.Background(), event)
		require.NoError(t, err)
	})

	t.Run("returns error for wrong event type", func(t *testing.T) {
		handler := NewProductDisabledHandler(logger)

		// Create a different event type
		event := &catalog.ProductCreatedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				catalog.EventTypeProductCreated,
				catalog.AggregateTypeProduct,
				uuid.New(),
				uuid.New(),
			),
		}

		err := handler.Handle(context.Background(), event)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected event type")
	})
}

func TestLoggingProductDisabledNotifier(t *testing.T) {
	logger := zap.NewNop()
	notifier := NewLoggingProductDisabledNotifier(logger)

	notification := ProductDisabledNotification{
		TenantID:  uuid.New().String(),
		ProductID: uuid.New().String(),
		Code:      "SKU-001",
		Name:      "Test Product",
	}

	err := notifier.NotifyProductDisabled(context.Background(), notification)
	require.NoError(t, err)
}
