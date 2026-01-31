package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/billing"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// UsageMeterRepository implements the billing.UsageMeterRepository interface
// It uses Redis for caching and GORM for calculating meters from usage records
type UsageMeterRepository struct {
	db          *gorm.DB
	redisClient *redis.Client
	keyPrefix   string
}

// NewUsageMeterRepository creates a new usage meter repository
func NewUsageMeterRepository(db *gorm.DB, redisClient *redis.Client) *UsageMeterRepository {
	return &UsageMeterRepository{
		db:          db,
		redisClient: redisClient,
		keyPrefix:   "usage_meter:",
	}
}

// meterCacheKey generates a cache key for a usage meter
func (r *UsageMeterRepository) meterCacheKey(tenantID uuid.UUID, usageType billing.UsageType, periodStart, periodEnd time.Time) string {
	return fmt.Sprintf("%smeter:%s:%s:%s:%s",
		r.keyPrefix,
		tenantID.String(),
		string(usageType),
		periodStart.Format("2006-01-02"),
		periodEnd.Format("2006-01-02"),
	)
}

// summaryCacheKey generates a cache key for a usage summary
func (r *UsageMeterRepository) summaryCacheKey(tenantID uuid.UUID, periodStart, periodEnd time.Time) string {
	return fmt.Sprintf("%ssummary:%s:%s:%s",
		r.keyPrefix,
		tenantID.String(),
		periodStart.Format("2006-01-02"),
		periodEnd.Format("2006-01-02"),
	)
}

// GetMeter retrieves a cached usage meter for a tenant and type
func (r *UsageMeterRepository) GetMeter(ctx context.Context, tenantID uuid.UUID, usageType billing.UsageType, periodStart, periodEnd time.Time) (*billing.UsageMeter, error) {
	if r.redisClient == nil {
		return nil, fmt.Errorf("redis client not configured")
	}

	key := r.meterCacheKey(tenantID, usageType, periodStart, periodEnd)
	data, err := r.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	var meter billing.UsageMeter
	if err := json.Unmarshal(data, &meter); err != nil {
		return nil, err
	}

	return &meter, nil
}

// SetMeter caches a usage meter
func (r *UsageMeterRepository) SetMeter(ctx context.Context, meter *billing.UsageMeter, ttl time.Duration) error {
	if r.redisClient == nil {
		return fmt.Errorf("redis client not configured")
	}

	key := r.meterCacheKey(meter.TenantID, meter.UsageType, meter.PeriodStart, meter.PeriodEnd)
	data, err := json.Marshal(meter)
	if err != nil {
		return err
	}

	return r.redisClient.Set(ctx, key, data, ttl).Err()
}

// InvalidateMeter removes a cached meter
func (r *UsageMeterRepository) InvalidateMeter(ctx context.Context, tenantID uuid.UUID, usageType billing.UsageType) error {
	if r.redisClient == nil {
		return nil // No-op if Redis not configured
	}

	// Use pattern matching to delete all meters for this tenant and type
	pattern := fmt.Sprintf("%smeter:%s:%s:*", r.keyPrefix, tenantID.String(), string(usageType))
	return r.deleteByPattern(ctx, pattern)
}

// InvalidateAllMeters removes all cached meters for a tenant
func (r *UsageMeterRepository) InvalidateAllMeters(ctx context.Context, tenantID uuid.UUID) error {
	if r.redisClient == nil {
		return nil // No-op if Redis not configured
	}

	// Delete all meters for this tenant
	meterPattern := fmt.Sprintf("%smeter:%s:*", r.keyPrefix, tenantID.String())
	if err := r.deleteByPattern(ctx, meterPattern); err != nil {
		return err
	}

	// Delete all summaries for this tenant
	summaryPattern := fmt.Sprintf("%ssummary:%s:*", r.keyPrefix, tenantID.String())
	return r.deleteByPattern(ctx, summaryPattern)
}

// GetSummary retrieves a cached usage summary for a tenant
func (r *UsageMeterRepository) GetSummary(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time) (*billing.UsageSummary, error) {
	if r.redisClient == nil {
		return nil, fmt.Errorf("redis client not configured")
	}

	key := r.summaryCacheKey(tenantID, periodStart, periodEnd)
	data, err := r.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	var summary billing.UsageSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return nil, err
	}

	return &summary, nil
}

// SetSummary caches a usage summary
func (r *UsageMeterRepository) SetSummary(ctx context.Context, summary *billing.UsageSummary, ttl time.Duration) error {
	if r.redisClient == nil {
		return fmt.Errorf("redis client not configured")
	}

	key := r.summaryCacheKey(summary.TenantID, summary.PeriodStart, summary.PeriodEnd)
	data, err := json.Marshal(summary)
	if err != nil {
		return err
	}

	return r.redisClient.Set(ctx, key, data, ttl).Err()
}

// CalculateMeter calculates a fresh usage meter from usage records
func (r *UsageMeterRepository) CalculateMeter(ctx context.Context, tenantID uuid.UUID, usageType billing.UsageType, periodStart, periodEnd time.Time) (*billing.UsageMeter, error) {
	meter := billing.NewUsageMeter(tenantID, usageType, periodStart, periodEnd)

	// For countable resources, we need to query the actual resource count
	if usageType.IsCountable() {
		count, err := r.getCountableResourceCount(ctx, tenantID, usageType)
		if err != nil {
			return nil, err
		}
		meter.WithTotalUsage(count).WithRecordCount(1)
		return meter, nil
	}

	// For accumulative resources, sum from usage records
	type aggregateResult struct {
		TotalUsage  int64
		RecordCount int64
		PeakUsage   int64
	}

	var result aggregateResult
	err := r.db.WithContext(ctx).
		Model(&UsageRecordModel{}).
		Select(`
			COALESCE(SUM(quantity), 0) as total_usage,
			COUNT(*) as record_count,
			COALESCE(MAX(quantity), 0) as peak_usage
		`).
		Where("tenant_id = ?", tenantID).
		Where("usage_type = ?", string(usageType)).
		Where("recorded_at >= ?", periodStart).
		Where("recorded_at <= ?", periodEnd).
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	meter.WithTotalUsage(result.TotalUsage).
		WithRecordCount(result.RecordCount).
		WithPeakUsage(result.PeakUsage).
		CalculateAverageRate()

	return meter, nil
}

// CalculateSummary calculates a fresh usage summary from usage records
func (r *UsageMeterRepository) CalculateSummary(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time) (*billing.UsageSummary, error) {
	summary := billing.NewUsageSummary(tenantID, periodStart, periodEnd)

	// Calculate meters for all usage types
	for _, usageType := range billing.AllUsageTypes() {
		meter, err := r.CalculateMeter(ctx, tenantID, usageType, periodStart, periodEnd)
		if err != nil {
			// Log error but continue with other types
			continue
		}
		summary.AddMeter(meter)
	}

	return summary, nil
}

// getCountableResourceCount gets the current count of countable resources
func (r *UsageMeterRepository) getCountableResourceCount(ctx context.Context, tenantID uuid.UUID, usageType billing.UsageType) (int64, error) {
	var count int64
	var err error

	switch usageType {
	case billing.UsageTypeActiveUsers:
		err = r.db.WithContext(ctx).
			Table("users").
			Where("tenant_id = ?", tenantID).
			Where("is_active = ?", true).
			Count(&count).Error

	case billing.UsageTypeProductsSKU:
		err = r.db.WithContext(ctx).
			Table("products").
			Where("tenant_id = ?", tenantID).
			Where("deleted_at IS NULL").
			Count(&count).Error

	case billing.UsageTypeWarehouses:
		err = r.db.WithContext(ctx).
			Table("warehouses").
			Where("tenant_id = ?", tenantID).
			Where("deleted_at IS NULL").
			Count(&count).Error

	case billing.UsageTypeCustomers:
		err = r.db.WithContext(ctx).
			Table("partners").
			Where("tenant_id = ?", tenantID).
			Where("partner_type = ?", "customer").
			Where("deleted_at IS NULL").
			Count(&count).Error

	case billing.UsageTypeSuppliers:
		err = r.db.WithContext(ctx).
			Table("partners").
			Where("tenant_id = ?", tenantID).
			Where("partner_type = ?", "supplier").
			Where("deleted_at IS NULL").
			Count(&count).Error

	default:
		// For unknown countable types, return 0
		return 0, nil
	}

	if err != nil {
		return 0, err
	}
	return count, nil
}

// deleteByPattern deletes all keys matching a pattern
func (r *UsageMeterRepository) deleteByPattern(ctx context.Context, pattern string) error {
	iter := r.redisClient.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		if err := r.redisClient.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

// Ensure UsageMeterRepository implements the interface
var _ billing.UsageMeterRepository = (*UsageMeterRepository)(nil)
