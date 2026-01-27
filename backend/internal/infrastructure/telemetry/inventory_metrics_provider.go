// Package telemetry provides OpenTelemetry integration for metrics collection.
package telemetry

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GormInventoryMetricsProvider implements InventoryMetricsProvider using GORM.
// It queries the inventory_items table directly for aggregated metrics.
type GormInventoryMetricsProvider struct {
	db *gorm.DB
}

// NewGormInventoryMetricsProvider creates a new GormInventoryMetricsProvider.
func NewGormInventoryMetricsProvider(db *gorm.DB) *GormInventoryMetricsProvider {
	return &GormInventoryMetricsProvider{db: db}
}

// GetLockedQuantityByWarehouse returns total locked quantity per warehouse for a tenant.
func (p *GormInventoryMetricsProvider) GetLockedQuantityByWarehouse(ctx context.Context, tenantID uuid.UUID) (map[uuid.UUID]int64, error) {
	type result struct {
		WarehouseID    uuid.UUID `gorm:"column:warehouse_id"`
		LockedQuantity int64     `gorm:"column:locked_quantity"`
	}

	var results []result
	err := p.db.WithContext(ctx).
		Table("inventory_items").
		Select("warehouse_id, COALESCE(SUM(locked_quantity), 0) as locked_quantity").
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Group("warehouse_id").
		Having("SUM(locked_quantity) > 0").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	m := make(map[uuid.UUID]int64, len(results))
	for _, r := range results {
		m[r.WarehouseID] = r.LockedQuantity
	}

	return m, nil
}

// GetLowStockCount returns count of products below minimum threshold for a tenant.
func (p *GormInventoryMetricsProvider) GetLowStockCount(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var count int64
	err := p.db.WithContext(ctx).
		Table("inventory_items").
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Where("min_quantity > 0 AND (available_quantity + locked_quantity) < min_quantity").
		Count(&count).Error

	return count, err
}

// GormTenantProvider implements TenantProvider using GORM.
type GormTenantProvider struct {
	db *gorm.DB
}

// NewGormTenantProvider creates a new GormTenantProvider.
func NewGormTenantProvider(db *gorm.DB) *GormTenantProvider {
	return &GormTenantProvider{db: db}
}

// GetActiveTenantIDs returns all active tenant IDs.
func (p *GormTenantProvider) GetActiveTenantIDs(ctx context.Context) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := p.db.WithContext(ctx).
		Table("tenants").
		Select("id").
		Where("deleted_at IS NULL AND status = ?", "active").
		Find(&ids).Error

	return ids, err
}
