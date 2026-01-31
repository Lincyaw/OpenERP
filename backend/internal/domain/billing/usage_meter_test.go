package billing

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewUsageMeter(t *testing.T) {
	tenantID := uuid.New()
	periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	meter := NewUsageMeter(tenantID, UsageTypeAPICalls, periodStart, periodEnd)

	assert.Equal(t, tenantID, meter.TenantID)
	assert.Equal(t, UsageTypeAPICalls, meter.UsageType)
	assert.Equal(t, UsageUnitRequests, meter.Unit)
	assert.Equal(t, periodStart, meter.PeriodStart)
	assert.Equal(t, periodEnd, meter.PeriodEnd)
	assert.NotZero(t, meter.LastUpdated)
}

func TestNewUsageMeterForCurrentMonth(t *testing.T) {
	tenantID := uuid.New()

	meter := NewUsageMeterForCurrentMonth(tenantID, UsageTypeStorageBytes)

	assert.Equal(t, tenantID, meter.TenantID)
	assert.Equal(t, UsageTypeStorageBytes, meter.UsageType)
	assert.Equal(t, UsageUnitBytes, meter.Unit)

	// Verify period is current month
	now := time.Now()
	expectedStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	assert.Equal(t, expectedStart, meter.PeriodStart)
	assert.True(t, meter.PeriodEnd.After(meter.PeriodStart))
}

func TestUsageMeter_WithTotalUsage(t *testing.T) {
	tenantID := uuid.New()
	meter := NewUsageMeterForCurrentMonth(tenantID, UsageTypeAPICalls)

	result := meter.WithTotalUsage(5000)

	assert.Equal(t, int64(5000), result.TotalUsage)
	assert.Same(t, meter, result)
}

func TestUsageMeter_WithQuotaLimit(t *testing.T) {
	tenantID := uuid.New()
	meter := NewUsageMeterForCurrentMonth(tenantID, UsageTypeAPICalls)
	meter.WithTotalUsage(5000)

	result := meter.WithQuotaLimit(10000)

	assert.NotNil(t, result.QuotaLimit)
	assert.Equal(t, int64(10000), *result.QuotaLimit)
	assert.Equal(t, float64(50), result.QuotaUsed) // 5000/10000 * 100
}

func TestUsageMeter_IsOverQuota(t *testing.T) {
	tenantID := uuid.New()

	t.Run("not over quota", func(t *testing.T) {
		meter := NewUsageMeterForCurrentMonth(tenantID, UsageTypeAPICalls)
		meter.WithTotalUsage(5000).WithQuotaLimit(10000)

		assert.False(t, meter.IsOverQuota())
	})

	t.Run("over quota", func(t *testing.T) {
		meter := NewUsageMeterForCurrentMonth(tenantID, UsageTypeAPICalls)
		meter.WithTotalUsage(15000).WithQuotaLimit(10000)

		assert.True(t, meter.IsOverQuota())
	})

	t.Run("no quota limit", func(t *testing.T) {
		meter := NewUsageMeterForCurrentMonth(tenantID, UsageTypeAPICalls)
		meter.WithTotalUsage(15000)

		assert.False(t, meter.IsOverQuota())
	})
}

func TestUsageMeter_IsNearQuota(t *testing.T) {
	tenantID := uuid.New()
	meter := NewUsageMeterForCurrentMonth(tenantID, UsageTypeAPICalls)
	meter.WithTotalUsage(8000).WithQuotaLimit(10000) // 80% used

	assert.True(t, meter.IsNearQuota(80))
	assert.True(t, meter.IsNearQuota(75))
	assert.False(t, meter.IsNearQuota(85))
}

func TestUsageMeter_GetRemainingQuota(t *testing.T) {
	tenantID := uuid.New()

	t.Run("with remaining quota", func(t *testing.T) {
		meter := NewUsageMeterForCurrentMonth(tenantID, UsageTypeAPICalls)
		meter.WithTotalUsage(3000).WithQuotaLimit(10000)

		assert.Equal(t, int64(7000), meter.GetRemainingQuota())
	})

	t.Run("over quota returns 0", func(t *testing.T) {
		meter := NewUsageMeterForCurrentMonth(tenantID, UsageTypeAPICalls)
		meter.WithTotalUsage(15000).WithQuotaLimit(10000)

		assert.Equal(t, int64(0), meter.GetRemainingQuota())
	})

	t.Run("unlimited returns -1", func(t *testing.T) {
		meter := NewUsageMeterForCurrentMonth(tenantID, UsageTypeAPICalls)
		meter.WithTotalUsage(15000)

		assert.Equal(t, int64(-1), meter.GetRemainingQuota())
	})
}

func TestUsageMeter_GetFormattedTotalUsage(t *testing.T) {
	tenantID := uuid.New()

	t.Run("formats requests", func(t *testing.T) {
		meter := NewUsageMeterForCurrentMonth(tenantID, UsageTypeAPICalls)
		meter.WithTotalUsage(5000)

		assert.Equal(t, "5000 requests", meter.GetFormattedTotalUsage())
	})

	t.Run("formats bytes", func(t *testing.T) {
		meter := NewUsageMeterForCurrentMonth(tenantID, UsageTypeStorageBytes)
		meter.WithTotalUsage(1073741824)

		assert.Equal(t, "1.00 GB", meter.GetFormattedTotalUsage())
	})
}

func TestUsageMeter_GetFormattedQuotaLimit(t *testing.T) {
	tenantID := uuid.New()

	t.Run("formats limit", func(t *testing.T) {
		meter := NewUsageMeterForCurrentMonth(tenantID, UsageTypeAPICalls)
		meter.WithQuotaLimit(10000)

		assert.Equal(t, "10000 requests", meter.GetFormattedQuotaLimit())
	})

	t.Run("unlimited", func(t *testing.T) {
		meter := NewUsageMeterForCurrentMonth(tenantID, UsageTypeAPICalls)

		assert.Equal(t, "Unlimited", meter.GetFormattedQuotaLimit())
	})
}

func TestUsageMeter_CalculateAverageRate(t *testing.T) {
	tenantID := uuid.New()
	periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC) // 30 days

	meter := NewUsageMeter(tenantID, UsageTypeAPICalls, periodStart, periodEnd)
	meter.WithTotalUsage(3000)
	meter.CalculateAverageRate()

	assert.Equal(t, float64(100), meter.AverageRate) // 3000 / 30 days
}

func TestUsageMeter_ProjectedUsage(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

	meter := NewUsageMeter(tenantID, UsageTypeAPICalls, periodStart, periodEnd)
	meter.WithTotalUsage(1000)

	// Projected usage depends on days elapsed, so just verify it returns a value
	projected := meter.ProjectedUsage()
	assert.GreaterOrEqual(t, projected, int64(1000))
}

func TestUsageSummary(t *testing.T) {
	tenantID := uuid.New()
	periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	t.Run("creates and adds meters", func(t *testing.T) {
		summary := NewUsageSummary(tenantID, periodStart, periodEnd)

		apiMeter := NewUsageMeter(tenantID, UsageTypeAPICalls, periodStart, periodEnd)
		apiMeter.WithTotalUsage(5000).WithQuotaLimit(10000)

		storageMeter := NewUsageMeter(tenantID, UsageTypeStorageBytes, periodStart, periodEnd)
		storageMeter.WithTotalUsage(500000000).WithQuotaLimit(1073741824)

		summary.AddMeter(apiMeter).AddMeter(storageMeter)

		assert.Len(t, summary.Meters, 2)
		assert.Equal(t, apiMeter, summary.GetMeter(UsageTypeAPICalls))
		assert.Equal(t, storageMeter, summary.GetMeter(UsageTypeStorageBytes))
	})

	t.Run("gets over quota types", func(t *testing.T) {
		summary := NewUsageSummary(tenantID, periodStart, periodEnd)

		apiMeter := NewUsageMeter(tenantID, UsageTypeAPICalls, periodStart, periodEnd)
		apiMeter.WithTotalUsage(15000).WithQuotaLimit(10000) // Over quota

		storageMeter := NewUsageMeter(tenantID, UsageTypeStorageBytes, periodStart, periodEnd)
		storageMeter.WithTotalUsage(500000000).WithQuotaLimit(1073741824) // Under quota

		summary.AddMeter(apiMeter).AddMeter(storageMeter)

		overQuota := summary.GetOverQuotaTypes()
		assert.Len(t, overQuota, 1)
		assert.Contains(t, overQuota, UsageTypeAPICalls)
	})

	t.Run("gets near quota types", func(t *testing.T) {
		summary := NewUsageSummary(tenantID, periodStart, periodEnd)

		apiMeter := NewUsageMeter(tenantID, UsageTypeAPICalls, periodStart, periodEnd)
		apiMeter.WithTotalUsage(8500).WithQuotaLimit(10000) // 85% - near quota

		storageMeter := NewUsageMeter(tenantID, UsageTypeStorageBytes, periodStart, periodEnd)
		storageMeter.WithTotalUsage(500000000).WithQuotaLimit(1073741824) // ~47% - not near

		summary.AddMeter(apiMeter).AddMeter(storageMeter)

		nearQuota := summary.GetNearQuotaTypes(80)
		assert.Len(t, nearQuota, 1)
		assert.Contains(t, nearQuota, UsageTypeAPICalls)
	})

	t.Run("has any over quota", func(t *testing.T) {
		summary := NewUsageSummary(tenantID, periodStart, periodEnd)

		apiMeter := NewUsageMeter(tenantID, UsageTypeAPICalls, periodStart, periodEnd)
		apiMeter.WithTotalUsage(5000).WithQuotaLimit(10000)

		summary.AddMeter(apiMeter)
		assert.False(t, summary.HasAnyOverQuota())

		apiMeter.WithTotalUsage(15000)
		assert.True(t, summary.HasAnyOverQuota())
	})
}

func TestUsageTrend(t *testing.T) {
	tenantID := uuid.New()

	t.Run("creates and adds data points", func(t *testing.T) {
		trend := NewUsageTrend(tenantID, UsageTypeAPICalls)

		trend.AddDataPoint(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 100).
			AddDataPoint(time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), 150).
			AddDataPoint(time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), 200)

		assert.Len(t, trend.DataPoints, 3)
	})

	t.Run("gets latest value", func(t *testing.T) {
		trend := NewUsageTrend(tenantID, UsageTypeAPICalls)
		trend.AddDataPoint(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 100).
			AddDataPoint(time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), 200)

		assert.Equal(t, int64(200), trend.GetLatestValue())
	})

	t.Run("gets latest value empty", func(t *testing.T) {
		trend := NewUsageTrend(tenantID, UsageTypeAPICalls)

		assert.Equal(t, int64(0), trend.GetLatestValue())
	})

	t.Run("calculates growth rate", func(t *testing.T) {
		trend := NewUsageTrend(tenantID, UsageTypeAPICalls)
		trend.AddDataPoint(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 100).
			AddDataPoint(time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), 150)

		assert.Equal(t, float64(50), trend.GetGrowthRate()) // (150-100)/100 * 100
	})

	t.Run("growth rate with insufficient data", func(t *testing.T) {
		trend := NewUsageTrend(tenantID, UsageTypeAPICalls)
		trend.AddDataPoint(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 100)

		assert.Equal(t, float64(0), trend.GetGrowthRate())
	})

	t.Run("growth rate with zero first value", func(t *testing.T) {
		trend := NewUsageTrend(tenantID, UsageTypeAPICalls)
		trend.AddDataPoint(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 0).
			AddDataPoint(time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), 100)

		assert.Equal(t, float64(0), trend.GetGrowthRate())
	})
}
