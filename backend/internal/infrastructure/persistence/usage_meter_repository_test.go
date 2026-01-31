package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/billing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupUsageMeterTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate the SQLite-compatible usage records model (needed for CalculateMeter)
	err = db.AutoMigrate(&UsageRecordModelSQLite{})
	require.NoError(t, err)

	return db
}

func TestUsageMeterRepository_CalculateMeter(t *testing.T) {
	db := setupUsageMeterTestDB(t)
	// Create repository without Redis (nil client)
	repo := NewUsageMeterRepository(db, nil)
	ctx := context.Background()

	tenantID := uuid.New()
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

	// Create usage records
	recordRepo := NewUsageRecordRepository(db)
	quantities := []int64{10, 20, 30, 40, 50}
	for _, qty := range quantities {
		record, _ := billing.NewUsageRecord(tenantID, billing.UsageTypeAPICalls, qty, periodStart, periodEnd)
		err := recordRepo.Save(ctx, record)
		require.NoError(t, err)
	}

	t.Run("calculates meter from usage records", func(t *testing.T) {
		meter, err := repo.CalculateMeter(ctx, tenantID, billing.UsageTypeAPICalls, periodStart, periodEnd)

		require.NoError(t, err)
		assert.NotNil(t, meter)
		assert.Equal(t, tenantID, meter.TenantID)
		assert.Equal(t, billing.UsageTypeAPICalls, meter.UsageType)
		assert.Equal(t, int64(150), meter.TotalUsage) // 10+20+30+40+50
		assert.Equal(t, int64(5), meter.RecordCount)
		assert.Equal(t, int64(50), meter.PeakUsage)
	})

	t.Run("returns zero for no records", func(t *testing.T) {
		otherTenantID := uuid.New()
		meter, err := repo.CalculateMeter(ctx, otherTenantID, billing.UsageTypeAPICalls, periodStart, periodEnd)

		require.NoError(t, err)
		assert.NotNil(t, meter)
		assert.Equal(t, int64(0), meter.TotalUsage)
		assert.Equal(t, int64(0), meter.RecordCount)
	})
}

func TestUsageMeterRepository_CalculateSummary(t *testing.T) {
	db := setupUsageMeterTestDB(t)
	repo := NewUsageMeterRepository(db, nil)
	ctx := context.Background()

	tenantID := uuid.New()
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

	// Create usage records for different types
	recordRepo := NewUsageRecordRepository(db)
	types := []billing.UsageType{
		billing.UsageTypeAPICalls,
		billing.UsageTypeAPICalls,
		billing.UsageTypeOrdersCreated,
	}

	for i, usageType := range types {
		record, _ := billing.NewUsageRecord(tenantID, usageType, int64((i+1)*10), periodStart, periodEnd)
		err := recordRepo.Save(ctx, record)
		require.NoError(t, err)
	}

	t.Run("calculates summary for all usage types", func(t *testing.T) {
		summary, err := repo.CalculateSummary(ctx, tenantID, periodStart, periodEnd)

		require.NoError(t, err)
		assert.NotNil(t, summary)
		assert.Equal(t, tenantID, summary.TenantID)

		// Check API calls meter
		apiMeter := summary.GetMeter(billing.UsageTypeAPICalls)
		assert.NotNil(t, apiMeter)
		assert.Equal(t, int64(30), apiMeter.TotalUsage) // 10+20

		// Check orders meter
		ordersMeter := summary.GetMeter(billing.UsageTypeOrdersCreated)
		assert.NotNil(t, ordersMeter)
		assert.Equal(t, int64(30), ordersMeter.TotalUsage)
	})
}

func TestUsageMeterRepository_CacheOperations_WithoutRedis(t *testing.T) {
	db := setupUsageMeterTestDB(t)
	repo := NewUsageMeterRepository(db, nil)
	ctx := context.Background()

	tenantID := uuid.New()
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

	t.Run("GetMeter returns error without Redis", func(t *testing.T) {
		_, err := repo.GetMeter(ctx, tenantID, billing.UsageTypeAPICalls, periodStart, periodEnd)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis client not configured")
	})

	t.Run("SetMeter returns error without Redis", func(t *testing.T) {
		meter := billing.NewUsageMeter(tenantID, billing.UsageTypeAPICalls, periodStart, periodEnd)
		err := repo.SetMeter(ctx, meter, time.Minute)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis client not configured")
	})

	t.Run("GetSummary returns error without Redis", func(t *testing.T) {
		_, err := repo.GetSummary(ctx, tenantID, periodStart, periodEnd)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis client not configured")
	})

	t.Run("SetSummary returns error without Redis", func(t *testing.T) {
		summary := billing.NewUsageSummary(tenantID, periodStart, periodEnd)
		err := repo.SetSummary(ctx, summary, time.Minute)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis client not configured")
	})

	t.Run("InvalidateMeter is no-op without Redis", func(t *testing.T) {
		err := repo.InvalidateMeter(ctx, tenantID, billing.UsageTypeAPICalls)
		assert.NoError(t, err)
	})

	t.Run("InvalidateAllMeters is no-op without Redis", func(t *testing.T) {
		err := repo.InvalidateAllMeters(ctx, tenantID)
		assert.NoError(t, err)
	})
}

func TestUsageMeterRepository_CacheKeyGeneration(t *testing.T) {
	db := setupUsageMeterTestDB(t)
	repo := NewUsageMeterRepository(db, nil)

	tenantID := uuid.New()
	periodStart := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2024, 6, 30, 23, 59, 59, 0, time.UTC)

	t.Run("generates correct meter cache key", func(t *testing.T) {
		key := repo.meterCacheKey(tenantID, billing.UsageTypeAPICalls, periodStart, periodEnd)

		assert.Contains(t, key, "usage_meter:")
		assert.Contains(t, key, "meter:")
		assert.Contains(t, key, tenantID.String())
		assert.Contains(t, key, "API_CALLS")
		assert.Contains(t, key, "2024-06-01")
		assert.Contains(t, key, "2024-06-30")
	})

	t.Run("generates correct summary cache key", func(t *testing.T) {
		key := repo.summaryCacheKey(tenantID, periodStart, periodEnd)

		assert.Contains(t, key, "usage_meter:")
		assert.Contains(t, key, "summary:")
		assert.Contains(t, key, tenantID.String())
		assert.Contains(t, key, "2024-06-01")
		assert.Contains(t, key, "2024-06-30")
	})
}

func TestUsageMeterRepository_CountableResources(t *testing.T) {
	// Skip this test as it requires the full database schema with users, products, etc.
	// This should be tested in integration tests against a real PostgreSQL database
	t.Skip("Countable resources test requires full database schema, skipping for unit tests")
}
