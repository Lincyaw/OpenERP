package telemetry_test

import (
	"context"
	"testing"
	"time"

	"github.com/erp/backend/internal/infrastructure/telemetry"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"
	"go.uber.org/zap"
)

func TestNewBusinessMetrics(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")

	bm, err := telemetry.NewBusinessMetrics(telemetry.BusinessMetricsConfig{
		Meter:  meter,
		Logger: zap.NewNop(),
	})

	require.NoError(t, err)
	require.NotNil(t, bm)
}

func TestNewBusinessMetrics_NilMeter(t *testing.T) {
	bm, err := telemetry.NewBusinessMetrics(telemetry.BusinessMetricsConfig{
		Meter:  nil,
		Logger: zap.NewNop(),
	})

	require.Error(t, err)
	assert.Nil(t, bm)
	assert.Equal(t, "NewBusinessMetrics: meter cannot be nil", err.Error())
}

func TestBusinessMetrics_RecordOrderCreated(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	bm, err := telemetry.NewBusinessMetrics(telemetry.BusinessMetricsConfig{
		Meter: meter,
	})
	require.NoError(t, err)

	ctx := context.Background()
	tenantID := uuid.New()

	// Should not panic
	bm.RecordOrderCreated(ctx, tenantID, telemetry.OrderTypeSales)
	bm.RecordOrderCreated(ctx, tenantID, telemetry.OrderTypePurchase)
}

func TestBusinessMetrics_RecordOrderAmount(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	bm, err := telemetry.NewBusinessMetrics(telemetry.BusinessMetricsConfig{
		Meter: meter,
	})
	require.NoError(t, err)

	ctx := context.Background()
	tenantID := uuid.New()

	// Should not panic
	bm.RecordOrderAmount(ctx, tenantID, telemetry.OrderTypeSales, 10000) // 100.00 CNY
	bm.RecordOrderAmount(ctx, tenantID, telemetry.OrderTypePurchase, 50000)
}

func TestBusinessMetrics_RecordOrderWithAmount(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	bm, err := telemetry.NewBusinessMetrics(telemetry.BusinessMetricsConfig{
		Meter: meter,
	})
	require.NoError(t, err)

	ctx := context.Background()
	tenantID := uuid.New()
	amount := decimal.NewFromFloat(199.99)

	// Should not panic and record both count and amount
	bm.RecordOrderWithAmount(ctx, tenantID, telemetry.OrderTypeSales, amount)
}

func TestBusinessMetrics_RecordPayment(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	bm, err := telemetry.NewBusinessMetrics(telemetry.BusinessMetricsConfig{
		Meter: meter,
	})
	require.NoError(t, err)

	ctx := context.Background()
	tenantID := uuid.New()

	// Should not panic
	bm.RecordPayment(ctx, tenantID, "wechat", telemetry.PaymentStatusSuccess)
	bm.RecordPayment(ctx, tenantID, "alipay", telemetry.PaymentStatusFailed)
}

func TestBusinessMetrics_RecordLockedQuantity(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	bm, err := telemetry.NewBusinessMetrics(telemetry.BusinessMetricsConfig{
		Meter: meter,
	})
	require.NoError(t, err)

	ctx := context.Background()
	tenantID := uuid.New()
	warehouseID := uuid.New()

	// Should not panic
	bm.RecordLockedQuantity(ctx, tenantID, warehouseID, 100)
	bm.RecordLockedQuantity(ctx, tenantID, warehouseID, 50)
}

func TestBusinessMetrics_RecordLowStockCount(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	bm, err := telemetry.NewBusinessMetrics(telemetry.BusinessMetricsConfig{
		Meter: meter,
	})
	require.NoError(t, err)

	ctx := context.Background()
	tenantID := uuid.New()

	// Should not panic
	bm.RecordLowStockCount(ctx, tenantID, 5)
	bm.RecordLowStockCount(ctx, tenantID, 10)
}

// Mock implementations for testing periodic collection

type mockTenantProvider struct {
	tenantIDs []uuid.UUID
	err       error
}

func (m *mockTenantProvider) GetActiveTenantIDs(ctx context.Context) ([]uuid.UUID, error) {
	return m.tenantIDs, m.err
}

type mockInventoryProvider struct {
	lockedQuantity map[uuid.UUID]int64
	lowStockCount  int64
	err            error
}

func (m *mockInventoryProvider) GetLockedQuantityByWarehouse(ctx context.Context, tenantID uuid.UUID) (map[uuid.UUID]int64, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.lockedQuantity, nil
}

func (m *mockInventoryProvider) GetLowStockCount(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.lowStockCount, nil
}

func TestBusinessMetrics_PeriodicCollection(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")

	tenantID := uuid.New()
	warehouseID := uuid.New()

	inventoryProvider := &mockInventoryProvider{
		lockedQuantity: map[uuid.UUID]int64{
			warehouseID: 100,
		},
		lowStockCount: 5,
	}

	bm, err := telemetry.NewBusinessMetrics(telemetry.BusinessMetricsConfig{
		Meter:             meter,
		Logger:            zap.NewNop(),
		InventoryProvider: inventoryProvider,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tenantProvider := &mockTenantProvider{
		tenantIDs: []uuid.UUID{tenantID},
	}

	// Start periodic collection with short interval for testing
	bm.StartPeriodicCollection(ctx, tenantProvider, 100*time.Millisecond)

	// Wait for at least one collection cycle
	time.Sleep(150 * time.Millisecond)

	// Stop collection
	bm.Stop()

	// Should complete without error
}

func TestBusinessMetrics_PeriodicCollection_NoProvider(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")

	bm, err := telemetry.NewBusinessMetrics(telemetry.BusinessMetricsConfig{
		Meter:  meter,
		Logger: zap.NewNop(),
		// No inventory provider
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tenantProvider := &mockTenantProvider{
		tenantIDs: []uuid.UUID{uuid.New()},
	}

	// Should not panic with no inventory provider
	bm.StartPeriodicCollection(ctx, tenantProvider, 50*time.Millisecond)

	time.Sleep(100 * time.Millisecond)
	bm.Stop()
}

func TestBusinessMetrics_Stop_Idempotent(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	bm, err := telemetry.NewBusinessMetrics(telemetry.BusinessMetricsConfig{
		Meter: meter,
	})
	require.NoError(t, err)

	// Calling Stop multiple times should not panic
	bm.Stop()
	bm.Stop()
	bm.Stop()
}

func TestBusinessMetrics_StartPeriodicCollection_OnlyOnce(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	bm, err := telemetry.NewBusinessMetrics(telemetry.BusinessMetricsConfig{
		Meter:  meter,
		Logger: zap.NewNop(),
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tenantProvider := &mockTenantProvider{
		tenantIDs: []uuid.UUID{},
	}

	// Calling StartPeriodicCollection multiple times should only start once
	bm.StartPeriodicCollection(ctx, tenantProvider, time.Hour)
	bm.StartPeriodicCollection(ctx, tenantProvider, time.Minute)
	bm.StartPeriodicCollection(ctx, tenantProvider, time.Second)

	bm.Stop()
}

func TestOrderType_Values(t *testing.T) {
	assert.Equal(t, telemetry.OrderType("sales"), telemetry.OrderTypeSales)
	assert.Equal(t, telemetry.OrderType("purchase"), telemetry.OrderTypePurchase)
}

func TestPaymentStatus_Values(t *testing.T) {
	assert.Equal(t, telemetry.PaymentStatus("success"), telemetry.PaymentStatusSuccess)
	assert.Equal(t, telemetry.PaymentStatus("failed"), telemetry.PaymentStatusFailed)
}

func TestMetricsError_Error(t *testing.T) {
	err := &telemetry.MetricsError{
		Op:  "TestOperation",
		Err: "test error message",
	}

	assert.Equal(t, "TestOperation: test error message", err.Error())
}
