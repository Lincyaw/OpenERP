package billing

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUsageRecord(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

	t.Run("creates valid usage record", func(t *testing.T) {
		record, err := NewUsageRecord(tenantID, UsageTypeAPICalls, 100, periodStart, periodEnd)

		require.NoError(t, err)
		assert.NotNil(t, record)
		assert.Equal(t, tenantID, record.TenantID)
		assert.Equal(t, UsageTypeAPICalls, record.UsageType)
		assert.Equal(t, int64(100), record.Quantity)
		assert.Equal(t, UsageUnitRequests, record.Unit)
		assert.Equal(t, periodStart, record.PeriodStart)
		assert.Equal(t, periodEnd, record.PeriodEnd)
		assert.NotEqual(t, uuid.Nil, record.ID)
		assert.NotZero(t, record.RecordedAt)
	})

	t.Run("fails with nil tenant ID", func(t *testing.T) {
		record, err := NewUsageRecord(uuid.Nil, UsageTypeAPICalls, 100, periodStart, periodEnd)

		assert.Error(t, err)
		assert.Nil(t, record)
		assert.Contains(t, err.Error(), "Tenant ID cannot be empty")
	})

	t.Run("fails with invalid usage type", func(t *testing.T) {
		record, err := NewUsageRecord(tenantID, UsageType("INVALID"), 100, periodStart, periodEnd)

		assert.Error(t, err)
		assert.Nil(t, record)
		assert.Contains(t, err.Error(), "Invalid usage type")
	})

	t.Run("fails with negative quantity", func(t *testing.T) {
		record, err := NewUsageRecord(tenantID, UsageTypeAPICalls, -1, periodStart, periodEnd)

		assert.Error(t, err)
		assert.Nil(t, record)
		assert.Contains(t, err.Error(), "Quantity cannot be negative")
	})

	t.Run("allows zero quantity", func(t *testing.T) {
		record, err := NewUsageRecord(tenantID, UsageTypeAPICalls, 0, periodStart, periodEnd)

		require.NoError(t, err)
		assert.NotNil(t, record)
		assert.Equal(t, int64(0), record.Quantity)
	})

	t.Run("fails with invalid period", func(t *testing.T) {
		record, err := NewUsageRecord(tenantID, UsageTypeAPICalls, 100, periodEnd, periodStart)

		assert.Error(t, err)
		assert.Nil(t, record)
		assert.Contains(t, err.Error(), "Period end cannot be before period start")
	})
}

func TestNewUsageRecordSimple(t *testing.T) {
	tenantID := uuid.New()

	t.Run("creates record for current month", func(t *testing.T) {
		record, err := NewUsageRecordSimple(tenantID, UsageTypeOrdersCreated, 5)

		require.NoError(t, err)
		assert.NotNil(t, record)
		assert.Equal(t, tenantID, record.TenantID)
		assert.Equal(t, UsageTypeOrdersCreated, record.UsageType)
		assert.Equal(t, int64(5), record.Quantity)

		// Verify period is current month
		now := time.Now()
		expectedStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		assert.Equal(t, expectedStart, record.PeriodStart)
		assert.True(t, record.PeriodEnd.After(record.PeriodStart))
	})
}

func TestUsageRecord_WithSource(t *testing.T) {
	tenantID := uuid.New()
	record, _ := NewUsageRecordSimple(tenantID, UsageTypeAPICalls, 1)

	result := record.WithSource("api_request", "/api/v1/products")

	assert.Equal(t, "api_request", result.SourceType)
	assert.Equal(t, "/api/v1/products", result.SourceID)
	assert.Same(t, record, result) // Returns same pointer
}

func TestUsageRecord_WithUser(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	record, _ := NewUsageRecordSimple(tenantID, UsageTypeAPICalls, 1)

	result := record.WithUser(userID)

	require.NotNil(t, result.UserID)
	assert.Equal(t, userID, *result.UserID)
}

func TestUsageRecord_WithRequestInfo(t *testing.T) {
	tenantID := uuid.New()
	record, _ := NewUsageRecordSimple(tenantID, UsageTypeAPICalls, 1)

	result := record.WithRequestInfo("192.168.1.1", "Mozilla/5.0")

	assert.Equal(t, "192.168.1.1", result.IPAddress)
	assert.Equal(t, "Mozilla/5.0", result.UserAgent)
}

func TestUsageRecord_WithMetadata(t *testing.T) {
	tenantID := uuid.New()
	record, _ := NewUsageRecordSimple(tenantID, UsageTypeAPICalls, 1)

	result := record.WithMetadata("endpoint", "/api/v1/products").
		WithMetadata("method", "GET")

	assert.Equal(t, "/api/v1/products", result.Metadata["endpoint"])
	assert.Equal(t, "GET", result.Metadata["method"])
}

func TestUsageRecord_WithRecordedAt(t *testing.T) {
	tenantID := uuid.New()
	record, _ := NewUsageRecordSimple(tenantID, UsageTypeAPICalls, 1)
	customTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	result := record.WithRecordedAt(customTime)

	assert.Equal(t, customTime, result.RecordedAt)
}

func TestUsageRecord_IsInPeriod(t *testing.T) {
	tenantID := uuid.New()
	periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2024, 1, 31, 23, 59, 59, 999999999, time.UTC)
	record, _ := NewUsageRecord(tenantID, UsageTypeAPICalls, 1, periodStart, periodEnd)

	tests := []struct {
		name     string
		time     time.Time
		expected bool
	}{
		{"time at period start", periodStart, true},
		{"time at period end", periodEnd, true},
		{"time in middle of period", time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC), true},
		{"time before period", time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC), false},
		{"time after period", time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, record.IsInPeriod(tt.time))
		})
	}
}

func TestUsageRecord_GetFormattedQuantity(t *testing.T) {
	tenantID := uuid.New()

	t.Run("formats API calls", func(t *testing.T) {
		record, _ := NewUsageRecordSimple(tenantID, UsageTypeAPICalls, 1000)
		assert.Equal(t, "1000 requests", record.GetFormattedQuantity())
	})

	t.Run("formats storage bytes", func(t *testing.T) {
		record, _ := NewUsageRecordSimple(tenantID, UsageTypeStorageBytes, 1073741824)
		assert.Equal(t, "1.00 GB", record.GetFormattedQuantity())
	})

	t.Run("formats count", func(t *testing.T) {
		record, _ := NewUsageRecordSimple(tenantID, UsageTypeActiveUsers, 10)
		assert.Equal(t, "10", record.GetFormattedQuantity())
	})
}

func TestUsageRecordBuilder(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("builds complete record", func(t *testing.T) {
		periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		periodEnd := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
		recordedAt := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

		record, err := NewUsageRecordBuilder(tenantID, UsageTypeAPICalls, 100).
			WithPeriod(periodStart, periodEnd).
			WithSource("api_request", "/api/v1/products").
			WithUser(userID).
			WithRequestInfo("192.168.1.1", "Mozilla/5.0").
			WithMetadata("method", "GET").
			WithRecordedAt(recordedAt).
			Build()

		require.NoError(t, err)
		assert.Equal(t, tenantID, record.TenantID)
		assert.Equal(t, UsageTypeAPICalls, record.UsageType)
		assert.Equal(t, int64(100), record.Quantity)
		assert.Equal(t, periodStart, record.PeriodStart)
		assert.Equal(t, periodEnd, record.PeriodEnd)
		assert.Equal(t, "api_request", record.SourceType)
		assert.Equal(t, "/api/v1/products", record.SourceID)
		assert.Equal(t, userID, *record.UserID)
		assert.Equal(t, "192.168.1.1", record.IPAddress)
		assert.Equal(t, "Mozilla/5.0", record.UserAgent)
		assert.Equal(t, "GET", record.Metadata["method"])
		assert.Equal(t, recordedAt, record.RecordedAt)
	})

	t.Run("fails with invalid period", func(t *testing.T) {
		periodStart := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
		periodEnd := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

		record, err := NewUsageRecordBuilder(tenantID, UsageTypeAPICalls, 100).
			WithPeriod(periodStart, periodEnd).
			Build()

		assert.Error(t, err)
		assert.Nil(t, record)
	})

	t.Run("propagates initial error", func(t *testing.T) {
		record, err := NewUsageRecordBuilder(uuid.Nil, UsageTypeAPICalls, 100).
			WithSource("test", "test").
			Build()

		assert.Error(t, err)
		assert.Nil(t, record)
	})
}

func TestCreateAPICallRecord(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("creates API call record with user", func(t *testing.T) {
		record, err := CreateAPICallRecord(tenantID, "/api/v1/products", &userID, "192.168.1.1", "Mozilla/5.0")

		require.NoError(t, err)
		assert.Equal(t, UsageTypeAPICalls, record.UsageType)
		assert.Equal(t, int64(1), record.Quantity)
		assert.Equal(t, "api_request", record.SourceType)
		assert.Equal(t, "/api/v1/products", record.SourceID)
		assert.Equal(t, userID, *record.UserID)
		assert.Equal(t, "192.168.1.1", record.IPAddress)
		assert.Equal(t, "Mozilla/5.0", record.UserAgent)
		assert.Equal(t, "/api/v1/products", record.Metadata["endpoint"])
	})

	t.Run("creates API call record without user", func(t *testing.T) {
		record, err := CreateAPICallRecord(tenantID, "/api/v1/products", nil, "192.168.1.1", "Mozilla/5.0")

		require.NoError(t, err)
		assert.Nil(t, record.UserID)
	})
}

func TestCreateStorageRecord(t *testing.T) {
	tenantID := uuid.New()

	record, err := CreateStorageRecord(tenantID, 1048576, "attachment", "file-123")

	require.NoError(t, err)
	assert.Equal(t, UsageTypeStorageBytes, record.UsageType)
	assert.Equal(t, int64(1048576), record.Quantity)
	assert.Equal(t, UsageUnitBytes, record.Unit)
	assert.Equal(t, "attachment", record.SourceType)
	assert.Equal(t, "file-123", record.SourceID)
}

func TestCreateOrderRecord(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	record, err := CreateOrderRecord(tenantID, "sales_order", "SO-001", userID)

	require.NoError(t, err)
	assert.Equal(t, UsageTypeOrdersCreated, record.UsageType)
	assert.Equal(t, int64(1), record.Quantity)
	assert.Equal(t, "sales_order", record.SourceType)
	assert.Equal(t, "SO-001", record.SourceID)
	assert.Equal(t, userID, *record.UserID)
	assert.Equal(t, "sales_order", record.Metadata["order_type"])
}
