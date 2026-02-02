package persistence

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GormTenantStatsRepository implements TenantStatsRepository using GORM
// This repository bypasses tenant isolation and is only accessible to super admins
type GormTenantStatsRepository struct {
	db *gorm.DB
}

// NewGormTenantStatsRepository creates a new GormTenantStatsRepository
func NewGormTenantStatsRepository(db *gorm.DB) *GormTenantStatsRepository {
	return &GormTenantStatsRepository{db: db}
}

// GetTenantUsageStats retrieves usage statistics for a specific tenant
func (r *GormTenantStatsRepository) GetTenantUsageStats(ctx context.Context, tenantID uuid.UUID) (*identity.TenantUsageStats, error) {
	stats := &identity.TenantUsageStats{
		TenantID:    tenantID,
		LastUpdated: time.Now(),
	}

	// Count users
	userCount, err := r.CountUsersByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	stats.UserCount = userCount

	// Count products
	productCount, err := r.CountProductsByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	stats.ProductCount = productCount

	// Count orders
	orderCount, err := r.CountOrdersByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	stats.OrderCount = orderCount

	// Count warehouses
	warehouseCount, err := r.CountWarehousesByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	stats.WarehouseCount = warehouseCount

	// Get storage usage (placeholder - would need file storage tracking)
	storageUsage, err := r.GetStorageUsageByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	stats.StorageUsage = storageUsage

	// Get API call count (placeholder - would need API metrics tracking)
	apiCallCount, err := r.GetAPICallCountByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	stats.APICallCount = apiCallCount

	return stats, nil
}

// GetPlatformStats retrieves platform-wide statistics
func (r *GormTenantStatsRepository) GetPlatformStats(ctx context.Context) (*identity.PlatformStats, error) {
	stats := &identity.PlatformStats{
		TenantsByPlan: make(map[string]int64),
		LastUpdated:   time.Now(),
	}

	// Count tenants by status
	var tenantCounts []struct {
		Status string
		Count  int64
	}
	if err := r.db.WithContext(ctx).
		Model(&models.TenantModel{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&tenantCounts).Error; err != nil {
		return nil, err
	}

	for _, tc := range tenantCounts {
		stats.TotalTenants += tc.Count
		switch identity.TenantStatus(tc.Status) {
		case identity.TenantStatusActive:
			stats.ActiveTenants = tc.Count
		case identity.TenantStatusTrial:
			stats.TrialTenants = tc.Count
		case identity.TenantStatusSuspended:
			stats.SuspendedTenants = tc.Count
		case identity.TenantStatusInactive:
			stats.InactiveTenants = tc.Count
		}
	}

	// Count tenants by plan
	var planCounts []struct {
		Plan  string
		Count int64
	}
	if err := r.db.WithContext(ctx).
		Model(&models.TenantModel{}).
		Select("plan, COUNT(*) as count").
		Group("plan").
		Scan(&planCounts).Error; err != nil {
		return nil, err
	}

	for _, pc := range planCounts {
		stats.TenantsByPlan[pc.Plan] = pc.Count
	}

	// Count total users
	totalUsers, err := r.CountTotalUsers(ctx)
	if err != nil {
		return nil, err
	}
	stats.TotalUsers = totalUsers

	// Count total products
	totalProducts, err := r.CountTotalProducts(ctx)
	if err != nil {
		return nil, err
	}
	stats.TotalProducts = totalProducts

	// Count total orders
	totalOrders, err := r.CountTotalOrders(ctx)
	if err != nil {
		return nil, err
	}
	stats.TotalOrders = totalOrders

	// Get total storage usage
	totalStorage, err := r.GetTotalStorageUsage(ctx)
	if err != nil {
		return nil, err
	}
	stats.TotalStorageUsage = totalStorage

	return stats, nil
}

// GetTenantGrowthTrend retrieves tenant growth trend for a period
func (r *GormTenantStatsRepository) GetTenantGrowthTrend(ctx context.Context, period string, days int) (*identity.TenantGrowthTrend, error) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	trend := &identity.TenantGrowthTrend{
		Period:    period,
		StartDate: startDate,
		EndDate:   endDate,
	}

	// Determine the interval based on period
	var interval string
	switch period {
	case "daily":
		interval = "day"
	case "weekly":
		interval = "week"
	case "monthly":
		interval = "month"
	default:
		interval = "day"
	}

	// Generate data points
	dataPoints := []identity.TenantGrowthPoint{}
	currentDate := startDate

	for currentDate.Before(endDate) || currentDate.Equal(endDate) {
		var nextDate time.Time
		switch interval {
		case "day":
			nextDate = currentDate.AddDate(0, 0, 1)
		case "week":
			nextDate = currentDate.AddDate(0, 0, 7)
		case "month":
			nextDate = currentDate.AddDate(0, 1, 0)
		}

		// Count new tenants in this period
		newTenants, err := r.CountNewTenantsByDateRange(ctx, currentDate, nextDate)
		if err != nil {
			return nil, err
		}

		// Count churned tenants in this period
		churnedCount, err := r.CountChurnedTenantsByDateRange(ctx, currentDate, nextDate)
		if err != nil {
			return nil, err
		}

		// Count total tenants as of this date
		totalTenants, err := r.CountTenantsByDateRange(ctx, nextDate)
		if err != nil {
			return nil, err
		}

		dataPoints = append(dataPoints, identity.TenantGrowthPoint{
			Date:         currentDate,
			TotalTenants: totalTenants,
			NewTenants:   newTenants,
			ChurnedCount: churnedCount,
		})

		currentDate = nextDate
		if currentDate.After(endDate) {
			break
		}
	}

	trend.DataPoints = dataPoints
	return trend, nil
}

// CountUsersByTenant counts users for a specific tenant
func (r *GormTenantStatsRepository) CountUsersByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.UserModel{}).
		Where("tenant_id = ?", tenantID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountProductsByTenant counts products for a specific tenant
func (r *GormTenantStatsRepository) CountProductsByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.ProductModel{}).
		Where("tenant_id = ?", tenantID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountOrdersByTenant counts orders (sales + purchase) for a specific tenant
func (r *GormTenantStatsRepository) CountOrdersByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var salesCount, purchaseCount int64

	// Count sales orders
	if err := r.db.WithContext(ctx).
		Model(&models.SalesOrderModel{}).
		Where("tenant_id = ?", tenantID).
		Count(&salesCount).Error; err != nil {
		return 0, err
	}

	// Count purchase orders
	if err := r.db.WithContext(ctx).
		Model(&models.PurchaseOrderModel{}).
		Where("tenant_id = ?", tenantID).
		Count(&purchaseCount).Error; err != nil {
		return 0, err
	}

	return salesCount + purchaseCount, nil
}

// CountWarehousesByTenant counts warehouses for a specific tenant
func (r *GormTenantStatsRepository) CountWarehousesByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.WarehouseModel{}).
		Where("tenant_id = ?", tenantID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// GetStorageUsageByTenant gets storage usage in bytes for a specific tenant
// Note: This is a placeholder implementation. In production, this would query
// a file storage tracking table or external storage service.
func (r *GormTenantStatsRepository) GetStorageUsageByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	// Placeholder: Return 0 as storage tracking is not yet implemented
	// In production, this would query product attachments, documents, etc.
	return 0, nil
}

// GetAPICallCountByTenant gets API call count for a specific tenant (last 30 days)
// Note: This is a placeholder implementation. In production, this would query
// an API metrics/analytics table or external monitoring service.
func (r *GormTenantStatsRepository) GetAPICallCountByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	// Placeholder: Return 0 as API call tracking is not yet implemented
	// In production, this would query an API metrics table
	return 0, nil
}

// CountTotalUsers counts all users across all tenants
func (r *GormTenantStatsRepository) CountTotalUsers(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.UserModel{}).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountTotalProducts counts all products across all tenants
func (r *GormTenantStatsRepository) CountTotalProducts(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.ProductModel{}).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountTotalOrders counts all orders across all tenants
func (r *GormTenantStatsRepository) CountTotalOrders(ctx context.Context) (int64, error) {
	var salesCount, purchaseCount int64

	if err := r.db.WithContext(ctx).
		Model(&models.SalesOrderModel{}).
		Count(&salesCount).Error; err != nil {
		return 0, err
	}

	if err := r.db.WithContext(ctx).
		Model(&models.PurchaseOrderModel{}).
		Count(&purchaseCount).Error; err != nil {
		return 0, err
	}

	return salesCount + purchaseCount, nil
}

// GetTotalStorageUsage gets total storage usage across all tenants
func (r *GormTenantStatsRepository) GetTotalStorageUsage(ctx context.Context) (int64, error) {
	// Placeholder: Return 0 as storage tracking is not yet implemented
	return 0, nil
}

// CountNewTenantsByDateRange counts new tenants created within a date range
func (r *GormTenantStatsRepository) CountNewTenantsByDateRange(ctx context.Context, start, end time.Time) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.TenantModel{}).
		Where("created_at >= ? AND created_at < ?", start, end).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountChurnedTenantsByDateRange counts tenants that became inactive within a date range
func (r *GormTenantStatsRepository) CountChurnedTenantsByDateRange(ctx context.Context, start, end time.Time) (int64, error) {
	var count int64
	// Count tenants that were updated to inactive status within the date range
	if err := r.db.WithContext(ctx).
		Model(&models.TenantModel{}).
		Where("status = ?", identity.TenantStatusInactive).
		Where("updated_at >= ? AND updated_at < ?", start, end).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountTenantsByDateRange counts total tenants as of a specific date
func (r *GormTenantStatsRepository) CountTenantsByDateRange(ctx context.Context, asOf time.Time) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.TenantModel{}).
		Where("created_at < ?", asOf).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// Ensure GormTenantStatsRepository implements TenantStatsRepository
var _ identity.TenantStatsRepository = (*GormTenantStatsRepository)(nil)
