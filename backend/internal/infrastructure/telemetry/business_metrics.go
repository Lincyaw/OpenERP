// Package telemetry provides OpenTelemetry integration for metrics collection.
package telemetry

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

// BusinessMetrics provides business metrics for the ERP system.
// It tracks order creation, payment activity, and inventory health.
type BusinessMetrics struct {
	meter  metric.Meter
	logger *zap.Logger

	// Counter metrics (monotonically increasing)
	orderCreatedTotal *Counter
	orderAmountTotal  *Counter
	paymentTotal      *Counter

	// Gauge metrics (point-in-time values)
	inventoryLockedQuantity *Gauge
	inventoryLowStockCount  *Gauge

	// Periodic collector
	stopChan    chan struct{}
	stopOnce    sync.Once
	collectOnce sync.Once

	// Data providers for periodic collection
	inventoryProvider InventoryMetricsProvider
}

// InventoryMetricsProvider provides inventory data for periodic metrics collection.
// This interface allows the telemetry layer to query inventory state without
// depending on the inventory domain directly.
type InventoryMetricsProvider interface {
	// GetLockedQuantityByWarehouse returns total locked quantity per warehouse for a tenant
	GetLockedQuantityByWarehouse(ctx context.Context, tenantID uuid.UUID) (map[uuid.UUID]int64, error)

	// GetLowStockCount returns count of products below minimum threshold for a tenant
	GetLowStockCount(ctx context.Context, tenantID uuid.UUID) (int64, error)
}

// BusinessMetricsConfig holds configuration for business metrics.
type BusinessMetricsConfig struct {
	Meter             metric.Meter
	Logger            *zap.Logger
	CollectInterval   time.Duration // Default: 5 minutes
	InventoryProvider InventoryMetricsProvider
}

// NewBusinessMetrics creates a new BusinessMetrics instance.
func NewBusinessMetrics(cfg BusinessMetricsConfig) (*BusinessMetrics, error) {
	if cfg.Meter == nil {
		return nil, ErrMeterNil
	}

	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	bm := &BusinessMetrics{
		meter:             cfg.Meter,
		logger:            logger,
		stopChan:          make(chan struct{}),
		inventoryProvider: cfg.InventoryProvider,
	}

	// Initialize counter metrics
	var err error

	// Order metrics
	bm.orderCreatedTotal, err = NewCounter(
		cfg.Meter,
		"erp_order_created_total",
		"Total number of orders created",
		"{orders}",
	)
	if err != nil {
		return nil, err
	}

	bm.orderAmountTotal, err = NewCounter(
		cfg.Meter,
		"erp_order_amount_total",
		"Total order amount in cents (fen)",
		"{fen}",
	)
	if err != nil {
		return nil, err
	}

	// Payment metrics
	bm.paymentTotal, err = NewCounter(
		cfg.Meter,
		"erp_payment_total",
		"Total number of payment transactions",
		"{payments}",
	)
	if err != nil {
		return nil, err
	}

	// Inventory gauge metrics
	bm.inventoryLockedQuantity, err = NewGauge(
		cfg.Meter,
		"erp_inventory_locked_quantity",
		"Current locked inventory quantity",
		"{units}",
	)
	if err != nil {
		return nil, err
	}

	bm.inventoryLowStockCount, err = NewGauge(
		cfg.Meter,
		"erp_inventory_low_stock_count",
		"Number of products below minimum stock threshold",
		"{products}",
	)
	if err != nil {
		return nil, err
	}

	return bm, nil
}

// =============================================================================
// Order Metrics
// =============================================================================

// OrderType represents the type of order for metrics labeling.
type OrderType string

const (
	OrderTypeSales    OrderType = "sales"
	OrderTypePurchase OrderType = "purchase"
)

// RecordOrderCreated records an order creation event.
// This should be called from the application layer when an order is created.
func (bm *BusinessMetrics) RecordOrderCreated(ctx context.Context, tenantID uuid.UUID, orderType OrderType) {
	bm.orderCreatedTotal.Inc(ctx,
		AttrTenantID.String(tenantID.String()),
		AttrOrderType.String(string(orderType)),
	)
}

// RecordOrderAmount records the order amount.
// Amount should be in the smallest currency unit (cents/fen).
func (bm *BusinessMetrics) RecordOrderAmount(ctx context.Context, tenantID uuid.UUID, orderType OrderType, amountFen int64) {
	bm.orderAmountTotal.Add(ctx, amountFen,
		AttrTenantID.String(tenantID.String()),
		AttrOrderType.String(string(orderType)),
	)
}

// RecordOrderWithAmount is a convenience method that records both order count and amount.
func (bm *BusinessMetrics) RecordOrderWithAmount(ctx context.Context, tenantID uuid.UUID, orderType OrderType, amount decimal.Decimal) {
	bm.RecordOrderCreated(ctx, tenantID, orderType)

	// Convert to fen (multiply by 100)
	amountFen := amount.Mul(decimal.NewFromInt(100)).IntPart()
	bm.RecordOrderAmount(ctx, tenantID, orderType, amountFen)
}

// =============================================================================
// Payment Metrics
// =============================================================================

// PaymentStatus represents the outcome of a payment for metrics labeling.
type PaymentStatus string

const (
	PaymentStatusSuccess PaymentStatus = "success"
	PaymentStatusFailed  PaymentStatus = "failed"
)

// RecordPayment records a payment transaction.
// This should be called when a payment callback is processed.
func (bm *BusinessMetrics) RecordPayment(ctx context.Context, tenantID uuid.UUID, paymentMethod string, status PaymentStatus) {
	bm.paymentTotal.Inc(ctx,
		AttrTenantID.String(tenantID.String()),
		AttrPaymentMethod.String(paymentMethod),
		AttrPaymentStatus.String(string(status)),
	)
}

// =============================================================================
// Inventory Metrics
// =============================================================================

// RecordLockedQuantity records the current locked inventory quantity for a warehouse.
// This is a gauge metric that should be updated periodically.
func (bm *BusinessMetrics) RecordLockedQuantity(ctx context.Context, tenantID, warehouseID uuid.UUID, quantity int64) {
	bm.inventoryLockedQuantity.Record(ctx, quantity,
		AttrTenantID.String(tenantID.String()),
		AttrWarehouseID.String(warehouseID.String()),
	)
}

// RecordLowStockCount records the number of products below minimum threshold.
// This is a gauge metric that should be updated periodically.
func (bm *BusinessMetrics) RecordLowStockCount(ctx context.Context, tenantID uuid.UUID, count int64) {
	bm.inventoryLowStockCount.Record(ctx, count,
		AttrTenantID.String(tenantID.String()),
	)
}

// =============================================================================
// Periodic Collection
// =============================================================================

// TenantProvider provides tenant IDs for periodic metrics collection.
type TenantProvider interface {
	GetActiveTenantIDs(ctx context.Context) ([]uuid.UUID, error)
}

// StartPeriodicCollection starts periodic collection of gauge metrics.
// It collects inventory metrics every interval (default: 5 minutes).
// This is non-blocking - use Stop() to stop collection.
func (bm *BusinessMetrics) StartPeriodicCollection(ctx context.Context, tenantProvider TenantProvider, interval time.Duration) {
	bm.collectOnce.Do(func() {
		if interval <= 0 {
			interval = 5 * time.Minute
		}

		go bm.runPeriodicCollection(ctx, tenantProvider, interval)
	})
}

// runPeriodicCollection runs the periodic collection loop.
func (bm *BusinessMetrics) runPeriodicCollection(ctx context.Context, tenantProvider TenantProvider, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Collect immediately on start
	bm.collectInventoryMetrics(ctx, tenantProvider)

	for {
		select {
		case <-bm.stopChan:
			bm.logger.Info("Stopping periodic business metrics collection")
			return
		case <-ctx.Done():
			bm.logger.Info("Context cancelled, stopping periodic business metrics collection")
			return
		case <-ticker.C:
			bm.collectInventoryMetrics(ctx, tenantProvider)
		}
	}
}

// collectInventoryMetrics collects inventory gauge metrics for all tenants.
func (bm *BusinessMetrics) collectInventoryMetrics(ctx context.Context, tenantProvider TenantProvider) {
	if bm.inventoryProvider == nil {
		bm.logger.Debug("No inventory provider configured, skipping inventory metrics collection")
		return
	}

	tenantIDs, err := tenantProvider.GetActiveTenantIDs(ctx)
	if err != nil {
		bm.logger.Error("Failed to get tenant IDs for metrics collection", zap.Error(err))
		return
	}

	for _, tenantID := range tenantIDs {
		bm.collectTenantInventoryMetrics(ctx, tenantID)
	}
}

// collectTenantInventoryMetrics collects inventory metrics for a single tenant.
func (bm *BusinessMetrics) collectTenantInventoryMetrics(ctx context.Context, tenantID uuid.UUID) {
	// Collect locked quantity by warehouse
	lockedByWarehouse, err := bm.inventoryProvider.GetLockedQuantityByWarehouse(ctx, tenantID)
	if err != nil {
		bm.logger.Warn("Failed to get locked quantity for tenant",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
	} else {
		for warehouseID, quantity := range lockedByWarehouse {
			bm.RecordLockedQuantity(ctx, tenantID, warehouseID, quantity)
		}
	}

	// Collect low stock count
	lowStockCount, err := bm.inventoryProvider.GetLowStockCount(ctx, tenantID)
	if err != nil {
		bm.logger.Warn("Failed to get low stock count for tenant",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
	} else {
		bm.RecordLowStockCount(ctx, tenantID, lowStockCount)
	}
}

// Stop stops the periodic collection.
func (bm *BusinessMetrics) Stop() {
	bm.stopOnce.Do(func() {
		close(bm.stopChan)
	})
}

// =============================================================================
// Error Types
// =============================================================================

// ErrMeterNil is returned when meter is nil.
var ErrMeterNil = &MetricsError{Op: "NewBusinessMetrics", Err: "meter cannot be nil"}

// MetricsError represents a metrics-related error.
type MetricsError struct {
	Op  string
	Err string
}

func (e *MetricsError) Error() string {
	return e.Op + ": " + e.Err
}

// =============================================================================
// Attribute Key Constants
// =============================================================================

// Business metrics attribute keys not already defined in metrics.go
var (
	// Additional business attributes can be added here
	AttrOrderSource = attribute.Key("order_source")
)
