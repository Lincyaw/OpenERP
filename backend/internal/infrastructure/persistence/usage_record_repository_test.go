package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/billing"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// UsageRecordModelSQLite is a SQLite-compatible version of UsageRecordModel for testing
type UsageRecordModelSQLite struct {
	ID          string    `gorm:"primaryKey"`
	TenantID    string    `gorm:"index;not null"`
	UsageType   string    `gorm:"not null"`
	Quantity    int64     `gorm:"not null"`
	Unit        string    `gorm:"not null;default:'requests'"`
	RecordedAt  time.Time `gorm:"not null"`
	PeriodStart time.Time `gorm:"not null"`
	PeriodEnd   time.Time `gorm:"not null"`
	SourceType  string
	SourceID    string
	Metadata    string
	UserID      *string
	IPAddress   string
	UserAgent   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (UsageRecordModelSQLite) TableName() string {
	return "usage_records"
}

func setupUsageRecordTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate the SQLite-compatible model
	err = db.AutoMigrate(&UsageRecordModelSQLite{})
	require.NoError(t, err)

	return db
}

func TestUsageRecordRepository_Save(t *testing.T) {
	db := setupUsageRecordTestDB(t)
	repo := NewUsageRecordRepository(db)
	ctx := context.Background()

	t.Run("saves new usage record", func(t *testing.T) {
		tenantID := uuid.New()
		now := time.Now()
		periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

		record, err := billing.NewUsageRecord(tenantID, billing.UsageTypeAPICalls, 100, periodStart, periodEnd)
		require.NoError(t, err)

		record.WithSource("api_request", "/api/v1/products")
		record.WithMetadata("endpoint", "/api/v1/products")

		err = repo.Save(ctx, record)
		require.NoError(t, err)

		// Verify it was saved
		found, err := repo.FindByID(ctx, record.ID)
		require.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, record.ID, found.ID)
		assert.Equal(t, tenantID, found.TenantID)
		assert.Equal(t, billing.UsageTypeAPICalls, found.UsageType)
		assert.Equal(t, int64(100), found.Quantity)
		assert.Equal(t, "api_request", found.SourceType)
		assert.Equal(t, "/api/v1/products", found.SourceID)
	})
}

func TestUsageRecordRepository_SaveBatch(t *testing.T) {
	db := setupUsageRecordTestDB(t)
	repo := NewUsageRecordRepository(db)
	ctx := context.Background()

	t.Run("saves multiple records in batch", func(t *testing.T) {
		tenantID := uuid.New()
		now := time.Now()
		periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

		records := make([]*billing.UsageRecord, 5)
		for i := 0; i < 5; i++ {
			record, err := billing.NewUsageRecord(tenantID, billing.UsageTypeAPICalls, int64(i+1), periodStart, periodEnd)
			require.NoError(t, err)
			records[i] = record
		}

		err := repo.SaveBatch(ctx, records)
		require.NoError(t, err)

		// Verify all were saved
		filter := billing.DefaultUsageRecordFilter()
		found, err := repo.FindByTenant(ctx, tenantID, filter)
		require.NoError(t, err)
		assert.Len(t, found, 5)
	})

	t.Run("handles empty batch", func(t *testing.T) {
		err := repo.SaveBatch(ctx, []*billing.UsageRecord{})
		require.NoError(t, err)
	})
}

func TestUsageRecordRepository_FindByID(t *testing.T) {
	db := setupUsageRecordTestDB(t)
	repo := NewUsageRecordRepository(db)
	ctx := context.Background()

	t.Run("returns not found for non-existent ID", func(t *testing.T) {
		_, err := repo.FindByID(ctx, uuid.New())
		assert.Equal(t, shared.ErrNotFound, err)
	})
}

func TestUsageRecordRepository_FindByTenant(t *testing.T) {
	db := setupUsageRecordTestDB(t)
	repo := NewUsageRecordRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

	// Create multiple records
	for i := 0; i < 10; i++ {
		record, _ := billing.NewUsageRecord(tenantID, billing.UsageTypeAPICalls, int64(i+1), periodStart, periodEnd)
		record.WithRecordedAt(now.Add(time.Duration(i) * time.Hour))
		err := repo.Save(ctx, record)
		require.NoError(t, err)
	}

	t.Run("returns all records for tenant", func(t *testing.T) {
		filter := billing.DefaultUsageRecordFilter()
		records, err := repo.FindByTenant(ctx, tenantID, filter)

		require.NoError(t, err)
		assert.Len(t, records, 10)
	})

	t.Run("applies pagination", func(t *testing.T) {
		filter := billing.DefaultUsageRecordFilter().WithPagination(1, 3)
		records, err := repo.FindByTenant(ctx, tenantID, filter)

		require.NoError(t, err)
		assert.Len(t, records, 3)
	})

	t.Run("filters by time range", func(t *testing.T) {
		start := now.Add(2 * time.Hour)
		end := now.Add(5 * time.Hour)
		filter := billing.DefaultUsageRecordFilter().WithTimeRange(start, end)

		records, err := repo.FindByTenant(ctx, tenantID, filter)

		require.NoError(t, err)
		assert.True(t, len(records) > 0 && len(records) <= 4)
	})
}

func TestUsageRecordRepository_FindByTenantAndType(t *testing.T) {
	db := setupUsageRecordTestDB(t)
	repo := NewUsageRecordRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

	// Create records of different types
	types := []billing.UsageType{
		billing.UsageTypeAPICalls,
		billing.UsageTypeAPICalls,
		billing.UsageTypeOrdersCreated,
		billing.UsageTypeStorageBytes,
	}

	for i, usageType := range types {
		record, _ := billing.NewUsageRecord(tenantID, usageType, int64(i+1), periodStart, periodEnd)
		err := repo.Save(ctx, record)
		require.NoError(t, err)
	}

	t.Run("filters by usage type", func(t *testing.T) {
		filter := billing.DefaultUsageRecordFilter()
		records, err := repo.FindByTenantAndType(ctx, tenantID, billing.UsageTypeAPICalls, filter)

		require.NoError(t, err)
		assert.Len(t, records, 2)
	})
}

func TestUsageRecordRepository_CountByTenant(t *testing.T) {
	db := setupUsageRecordTestDB(t)
	repo := NewUsageRecordRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

	// Create 5 records
	for i := 0; i < 5; i++ {
		record, _ := billing.NewUsageRecord(tenantID, billing.UsageTypeAPICalls, int64(i+1), periodStart, periodEnd)
		err := repo.Save(ctx, record)
		require.NoError(t, err)
	}

	t.Run("counts all records", func(t *testing.T) {
		filter := billing.DefaultUsageRecordFilter()
		count, err := repo.CountByTenant(ctx, tenantID, filter)

		require.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})
}

func TestUsageRecordRepository_SumByTenantAndType(t *testing.T) {
	db := setupUsageRecordTestDB(t)
	repo := NewUsageRecordRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

	// Create records with known quantities
	quantities := []int64{10, 20, 30, 40, 50}
	for _, qty := range quantities {
		record, _ := billing.NewUsageRecord(tenantID, billing.UsageTypeAPICalls, qty, periodStart, periodEnd)
		err := repo.Save(ctx, record)
		require.NoError(t, err)
	}

	t.Run("sums quantities correctly", func(t *testing.T) {
		sum, err := repo.SumByTenantAndType(ctx, tenantID, billing.UsageTypeAPICalls, periodStart, periodEnd)

		require.NoError(t, err)
		assert.Equal(t, int64(150), sum) // 10+20+30+40+50
	})

	t.Run("returns 0 for no records", func(t *testing.T) {
		otherTenantID := uuid.New()
		sum, err := repo.SumByTenantAndType(ctx, otherTenantID, billing.UsageTypeAPICalls, periodStart, periodEnd)

		require.NoError(t, err)
		assert.Equal(t, int64(0), sum)
	})
}

func TestUsageRecordRepository_DeleteOlderThan(t *testing.T) {
	db := setupUsageRecordTestDB(t)
	repo := NewUsageRecordRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

	// Create records with different recorded times
	times := []time.Time{
		now.AddDate(0, -3, 0), // 3 months ago
		now.AddDate(0, -2, 0), // 2 months ago
		now.AddDate(0, -1, 0), // 1 month ago
		now,                   // now
	}

	for _, recordedAt := range times {
		record, _ := billing.NewUsageRecord(tenantID, billing.UsageTypeAPICalls, 1, periodStart, periodEnd)
		record.WithRecordedAt(recordedAt)
		err := repo.Save(ctx, record)
		require.NoError(t, err)
	}

	t.Run("deletes old records", func(t *testing.T) {
		cutoff := now.AddDate(0, -1, -15) // 1.5 months ago
		deleted, err := repo.DeleteOlderThan(ctx, cutoff)

		require.NoError(t, err)
		assert.Equal(t, int64(2), deleted) // 3 months and 2 months ago

		// Verify remaining
		filter := billing.DefaultUsageRecordFilter()
		remaining, err := repo.FindByTenant(ctx, tenantID, filter)
		require.NoError(t, err)
		assert.Len(t, remaining, 2)
	})
}

func TestUsageRecordModel_ToEntity(t *testing.T) {
	userID := uuid.New()
	model := &UsageRecordModel{
		ID:          uuid.New(),
		TenantID:    uuid.New(),
		UsageType:   "API_CALLS",
		Quantity:    100,
		Unit:        "requests",
		RecordedAt:  time.Now(),
		PeriodStart: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:   time.Date(2024, 6, 30, 23, 59, 59, 0, time.UTC),
		SourceType:  "api_request",
		SourceID:    "/api/v1/products",
		Metadata:    []byte(`{"endpoint": "/api/v1/products"}`),
		UserID:      &userID,
		IPAddress:   "192.168.1.1",
		UserAgent:   "Mozilla/5.0",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	entity := model.ToEntity()

	assert.Equal(t, model.ID, entity.ID)
	assert.Equal(t, model.TenantID, entity.TenantID)
	assert.Equal(t, billing.UsageTypeAPICalls, entity.UsageType)
	assert.Equal(t, int64(100), entity.Quantity)
	assert.Equal(t, billing.UsageUnitRequests, entity.Unit)
	assert.Equal(t, "api_request", entity.SourceType)
	assert.Equal(t, "/api/v1/products", entity.SourceID)
	assert.Equal(t, "/api/v1/products", entity.Metadata["endpoint"])
	assert.Equal(t, &userID, entity.UserID)
	assert.Equal(t, "192.168.1.1", entity.IPAddress)
	assert.Equal(t, "Mozilla/5.0", entity.UserAgent)
}

func TestUsageRecordModelFromEntity(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

	record, err := billing.NewUsageRecord(tenantID, billing.UsageTypeAPICalls, 100, periodStart, periodEnd)
	require.NoError(t, err)

	record.WithSource("api_request", "/api/v1/products")
	record.WithMetadata("endpoint", "/api/v1/products")

	model := UsageRecordModelFromEntity(record)

	assert.Equal(t, record.ID, model.ID)
	assert.Equal(t, record.TenantID, model.TenantID)
	assert.Equal(t, "API_CALLS", model.UsageType)
	assert.Equal(t, int64(100), model.Quantity)
	assert.Equal(t, "requests", model.Unit)
	assert.Equal(t, "api_request", model.SourceType)
	assert.Equal(t, "/api/v1/products", model.SourceID)
}
