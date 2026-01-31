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

// UsageQuotaModelSQLite is a SQLite-compatible version of UsageQuotaModel for testing
type UsageQuotaModelSQLite struct {
	ID            string  `gorm:"primaryKey"`
	PlanID        string  `gorm:"not null"`
	TenantID      *string `gorm:"index"`
	UsageType     string  `gorm:"not null"`
	QuotaLimit    int64   `gorm:"column:quota_limit;not null;default:-1"`
	Unit          string  `gorm:"not null;default:'requests'"`
	ResetPeriod   string  `gorm:"not null;default:'MONTHLY'"`
	SoftLimit     *int64  `gorm:"column:soft_limit"`
	OveragePolicy string  `gorm:"not null;default:'BLOCK'"`
	Description   string
	IsActive      bool `gorm:"not null;default:true"`
	Version       int  `gorm:"not null;default:1"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (UsageQuotaModelSQLite) TableName() string {
	return "usage_quotas"
}

func setupUsageQuotaTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate the SQLite-compatible model
	err = db.AutoMigrate(&UsageQuotaModelSQLite{})
	require.NoError(t, err)

	return db
}

func TestUsageQuotaRepository_Save(t *testing.T) {
	db := setupUsageQuotaTestDB(t)
	repo := NewUsageQuotaRepository(db)
	ctx := context.Background()

	t.Run("saves new plan quota", func(t *testing.T) {
		quota, err := billing.NewUsageQuota("basic", billing.UsageTypeAPICalls, 10000, billing.ResetPeriodMonthly)
		require.NoError(t, err)

		quota.WithSoftLimit(8000)
		quota.WithOveragePolicy(billing.OveragePolicyWarn)
		quota.WithDescription("API calls per month")

		err = repo.Save(ctx, quota)
		require.NoError(t, err)

		// Verify it was saved
		found, err := repo.FindByID(ctx, quota.ID)
		require.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, quota.ID, found.ID)
		assert.Equal(t, "basic", found.PlanID)
		assert.Equal(t, billing.UsageTypeAPICalls, found.UsageType)
		assert.Equal(t, int64(10000), found.Limit)
		assert.NotNil(t, found.SoftLimit)
		assert.Equal(t, int64(8000), *found.SoftLimit)
		assert.Equal(t, billing.OveragePolicyWarn, found.OveragePolicy)
	})

	t.Run("saves tenant-specific quota override", func(t *testing.T) {
		tenantID := uuid.New()
		quota, err := billing.NewTenantUsageQuota(tenantID, "basic", billing.UsageTypeAPICalls, 20000, billing.ResetPeriodMonthly)
		require.NoError(t, err)

		err = repo.Save(ctx, quota)
		require.NoError(t, err)

		// Verify it was saved
		found, err := repo.FindByID(ctx, quota.ID)
		require.NoError(t, err)
		assert.NotNil(t, found)
		assert.NotNil(t, found.TenantID)
		assert.Equal(t, tenantID, *found.TenantID)
		assert.Equal(t, int64(20000), found.Limit)
	})
}

func TestUsageQuotaRepository_Update(t *testing.T) {
	db := setupUsageQuotaTestDB(t)
	repo := NewUsageQuotaRepository(db)
	ctx := context.Background()

	t.Run("updates existing quota", func(t *testing.T) {
		quota, err := billing.NewUsageQuota("basic", billing.UsageTypeAPICalls, 10000, billing.ResetPeriodMonthly)
		require.NoError(t, err)

		err = repo.Save(ctx, quota)
		require.NoError(t, err)

		// Update the quota
		err = quota.SetLimit(15000)
		require.NoError(t, err)

		err = repo.Update(ctx, quota)
		require.NoError(t, err)

		// Verify update
		found, err := repo.FindByID(ctx, quota.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(15000), found.Limit)
	})
}

func TestUsageQuotaRepository_FindByID(t *testing.T) {
	db := setupUsageQuotaTestDB(t)
	repo := NewUsageQuotaRepository(db)
	ctx := context.Background()

	t.Run("returns not found for non-existent ID", func(t *testing.T) {
		_, err := repo.FindByID(ctx, uuid.New())
		assert.Equal(t, shared.ErrNotFound, err)
	})
}

func TestUsageQuotaRepository_FindByPlan(t *testing.T) {
	db := setupUsageQuotaTestDB(t)
	repo := NewUsageQuotaRepository(db)
	ctx := context.Background()

	// Create quotas for different plans
	plans := []string{"free", "basic", "pro"}
	usageTypes := []billing.UsageType{billing.UsageTypeAPICalls, billing.UsageTypeStorageBytes, billing.UsageTypeOrdersCreated}

	for _, plan := range plans {
		for _, usageType := range usageTypes {
			quota, _ := billing.NewUsageQuota(plan, usageType, 1000, billing.ResetPeriodMonthly)
			err := repo.Save(ctx, quota)
			require.NoError(t, err)
		}
	}

	t.Run("returns all quotas for plan", func(t *testing.T) {
		quotas, err := repo.FindByPlan(ctx, "basic")

		require.NoError(t, err)
		assert.Len(t, quotas, 3)
		for _, q := range quotas {
			assert.Equal(t, "basic", q.PlanID)
			assert.Nil(t, q.TenantID)
		}
	})

	t.Run("returns empty for non-existent plan", func(t *testing.T) {
		quotas, err := repo.FindByPlan(ctx, "enterprise")

		require.NoError(t, err)
		assert.Len(t, quotas, 0)
	})
}

func TestUsageQuotaRepository_FindByPlanAndType(t *testing.T) {
	db := setupUsageQuotaTestDB(t)
	repo := NewUsageQuotaRepository(db)
	ctx := context.Background()

	// Create a quota
	quota, _ := billing.NewUsageQuota("basic", billing.UsageTypeAPICalls, 10000, billing.ResetPeriodMonthly)
	err := repo.Save(ctx, quota)
	require.NoError(t, err)

	t.Run("finds quota by plan and type", func(t *testing.T) {
		found, err := repo.FindByPlanAndType(ctx, "basic", billing.UsageTypeAPICalls)

		require.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, "basic", found.PlanID)
		assert.Equal(t, billing.UsageTypeAPICalls, found.UsageType)
	})

	t.Run("returns not found for non-existent combination", func(t *testing.T) {
		_, err := repo.FindByPlanAndType(ctx, "basic", billing.UsageTypeStorageBytes)
		assert.Equal(t, shared.ErrNotFound, err)
	})
}

func TestUsageQuotaRepository_FindByTenant(t *testing.T) {
	db := setupUsageQuotaTestDB(t)
	repo := NewUsageQuotaRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()

	// Create tenant-specific overrides
	usageTypes := []billing.UsageType{billing.UsageTypeAPICalls, billing.UsageTypeStorageBytes}
	for _, usageType := range usageTypes {
		quota, _ := billing.NewTenantUsageQuota(tenantID, "basic", usageType, 20000, billing.ResetPeriodMonthly)
		err := repo.Save(ctx, quota)
		require.NoError(t, err)
	}

	t.Run("returns all tenant overrides", func(t *testing.T) {
		quotas, err := repo.FindByTenant(ctx, tenantID)

		require.NoError(t, err)
		assert.Len(t, quotas, 2)
		for _, q := range quotas {
			assert.NotNil(t, q.TenantID)
			assert.Equal(t, tenantID, *q.TenantID)
		}
	})

	t.Run("returns empty for tenant without overrides", func(t *testing.T) {
		otherTenantID := uuid.New()
		quotas, err := repo.FindByTenant(ctx, otherTenantID)

		require.NoError(t, err)
		assert.Len(t, quotas, 0)
	})
}

func TestUsageQuotaRepository_FindByTenantAndType(t *testing.T) {
	db := setupUsageQuotaTestDB(t)
	repo := NewUsageQuotaRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()

	// Create a tenant-specific override
	quota, _ := billing.NewTenantUsageQuota(tenantID, "basic", billing.UsageTypeAPICalls, 20000, billing.ResetPeriodMonthly)
	err := repo.Save(ctx, quota)
	require.NoError(t, err)

	t.Run("finds tenant override by type", func(t *testing.T) {
		found, err := repo.FindByTenantAndType(ctx, tenantID, billing.UsageTypeAPICalls)

		require.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, tenantID, *found.TenantID)
		assert.Equal(t, billing.UsageTypeAPICalls, found.UsageType)
	})

	t.Run("returns not found for non-existent override", func(t *testing.T) {
		_, err := repo.FindByTenantAndType(ctx, tenantID, billing.UsageTypeStorageBytes)
		assert.Equal(t, shared.ErrNotFound, err)
	})
}

func TestUsageQuotaRepository_FindEffectiveQuota(t *testing.T) {
	db := setupUsageQuotaTestDB(t)
	repo := NewUsageQuotaRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()

	// Create plan default
	planQuota, _ := billing.NewUsageQuota("basic", billing.UsageTypeAPICalls, 10000, billing.ResetPeriodMonthly)
	err := repo.Save(ctx, planQuota)
	require.NoError(t, err)

	// Create tenant override
	tenantQuota, _ := billing.NewTenantUsageQuota(tenantID, "basic", billing.UsageTypeAPICalls, 20000, billing.ResetPeriodMonthly)
	err = repo.Save(ctx, tenantQuota)
	require.NoError(t, err)

	t.Run("returns tenant override when exists", func(t *testing.T) {
		found, err := repo.FindEffectiveQuota(ctx, tenantID, "basic", billing.UsageTypeAPICalls)

		require.NoError(t, err)
		assert.NotNil(t, found)
		assert.NotNil(t, found.TenantID)
		assert.Equal(t, int64(20000), found.Limit)
	})

	t.Run("returns plan default when no tenant override", func(t *testing.T) {
		otherTenantID := uuid.New()
		found, err := repo.FindEffectiveQuota(ctx, otherTenantID, "basic", billing.UsageTypeAPICalls)

		require.NoError(t, err)
		assert.NotNil(t, found)
		assert.Nil(t, found.TenantID)
		assert.Equal(t, int64(10000), found.Limit)
	})

	t.Run("returns not found when no quota exists", func(t *testing.T) {
		_, err := repo.FindEffectiveQuota(ctx, tenantID, "basic", billing.UsageTypeStorageBytes)
		assert.Equal(t, shared.ErrNotFound, err)
	})
}

func TestUsageQuotaRepository_FindAllEffectiveQuotas(t *testing.T) {
	db := setupUsageQuotaTestDB(t)
	repo := NewUsageQuotaRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()

	// Create plan defaults
	planQuota1, _ := billing.NewUsageQuota("basic", billing.UsageTypeAPICalls, 10000, billing.ResetPeriodMonthly)
	planQuota2, _ := billing.NewUsageQuota("basic", billing.UsageTypeStorageBytes, 1000000, billing.ResetPeriodNever)
	planQuota3, _ := billing.NewUsageQuota("basic", billing.UsageTypeOrdersCreated, 500, billing.ResetPeriodMonthly)

	err := repo.Save(ctx, planQuota1)
	require.NoError(t, err)
	err = repo.Save(ctx, planQuota2)
	require.NoError(t, err)
	err = repo.Save(ctx, planQuota3)
	require.NoError(t, err)

	// Create tenant override for API calls only
	tenantQuota, _ := billing.NewTenantUsageQuota(tenantID, "basic", billing.UsageTypeAPICalls, 20000, billing.ResetPeriodMonthly)
	err = repo.Save(ctx, tenantQuota)
	require.NoError(t, err)

	t.Run("returns merged quotas with tenant overrides", func(t *testing.T) {
		quotas, err := repo.FindAllEffectiveQuotas(ctx, tenantID, "basic")

		require.NoError(t, err)
		assert.Len(t, quotas, 3)

		// Find the API calls quota - should be the tenant override
		var apiQuota *billing.UsageQuota
		for _, q := range quotas {
			if q.UsageType == billing.UsageTypeAPICalls {
				apiQuota = q
				break
			}
		}
		require.NotNil(t, apiQuota)
		assert.NotNil(t, apiQuota.TenantID)
		assert.Equal(t, int64(20000), apiQuota.Limit)
	})
}

func TestUsageQuotaRepository_Delete(t *testing.T) {
	db := setupUsageQuotaTestDB(t)
	repo := NewUsageQuotaRepository(db)
	ctx := context.Background()

	t.Run("deletes existing quota", func(t *testing.T) {
		quota, _ := billing.NewUsageQuota("basic", billing.UsageTypeAPICalls, 10000, billing.ResetPeriodMonthly)
		err := repo.Save(ctx, quota)
		require.NoError(t, err)

		err = repo.Delete(ctx, quota.ID)
		require.NoError(t, err)

		// Verify deletion
		_, err = repo.FindByID(ctx, quota.ID)
		assert.Equal(t, shared.ErrNotFound, err)
	})

	t.Run("returns not found for non-existent ID", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.New())
		assert.Equal(t, shared.ErrNotFound, err)
	})
}

func TestUsageQuotaRepository_DeleteByTenant(t *testing.T) {
	db := setupUsageQuotaTestDB(t)
	repo := NewUsageQuotaRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()

	// Create tenant overrides
	for _, usageType := range []billing.UsageType{billing.UsageTypeAPICalls, billing.UsageTypeStorageBytes} {
		quota, _ := billing.NewTenantUsageQuota(tenantID, "basic", usageType, 20000, billing.ResetPeriodMonthly)
		err := repo.Save(ctx, quota)
		require.NoError(t, err)
	}

	t.Run("deletes all tenant overrides", func(t *testing.T) {
		err := repo.DeleteByTenant(ctx, tenantID)
		require.NoError(t, err)

		// Verify deletion
		quotas, err := repo.FindByTenant(ctx, tenantID)
		require.NoError(t, err)
		assert.Len(t, quotas, 0)
	})
}

func TestUsageQuotaModel_ToEntity(t *testing.T) {
	tenantID := uuid.New()
	softLimit := int64(8000)
	model := &UsageQuotaModel{
		ID:            uuid.New(),
		PlanID:        "basic",
		TenantID:      &tenantID,
		UsageType:     "API_CALLS",
		QuotaLimit:    10000,
		Unit:          "requests",
		ResetPeriod:   "MONTHLY",
		SoftLimit:     &softLimit,
		OveragePolicy: "WARN",
		Description:   "API calls per month",
		IsActive:      true,
		Version:       1,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	entity := model.ToEntity()

	assert.Equal(t, model.ID, entity.ID)
	assert.Equal(t, "basic", entity.PlanID)
	assert.NotNil(t, entity.TenantID)
	assert.Equal(t, tenantID, *entity.TenantID)
	assert.Equal(t, billing.UsageTypeAPICalls, entity.UsageType)
	assert.Equal(t, int64(10000), entity.Limit)
	assert.Equal(t, billing.UsageUnitRequests, entity.Unit)
	assert.Equal(t, billing.ResetPeriodMonthly, entity.ResetPeriod)
	assert.NotNil(t, entity.SoftLimit)
	assert.Equal(t, int64(8000), *entity.SoftLimit)
	assert.Equal(t, billing.OveragePolicyWarn, entity.OveragePolicy)
	assert.Equal(t, "API calls per month", entity.Description)
	assert.True(t, entity.IsActive)
}

func TestUsageQuotaModelFromEntity(t *testing.T) {
	quota, err := billing.NewUsageQuota("basic", billing.UsageTypeAPICalls, 10000, billing.ResetPeriodMonthly)
	require.NoError(t, err)

	quota.WithSoftLimit(8000)
	quota.WithOveragePolicy(billing.OveragePolicyWarn)
	quota.WithDescription("API calls per month")

	model := UsageQuotaModelFromEntity(quota)

	assert.Equal(t, quota.ID, model.ID)
	assert.Equal(t, "basic", model.PlanID)
	assert.Nil(t, model.TenantID)
	assert.Equal(t, "API_CALLS", model.UsageType)
	assert.Equal(t, int64(10000), model.QuotaLimit)
	assert.Equal(t, "requests", model.Unit)
	assert.Equal(t, "MONTHLY", model.ResetPeriod)
	assert.NotNil(t, model.SoftLimit)
	assert.Equal(t, int64(8000), *model.SoftLimit)
	assert.Equal(t, "WARN", model.OveragePolicy)
	assert.Equal(t, "API calls per month", model.Description)
	assert.True(t, model.IsActive)
}
