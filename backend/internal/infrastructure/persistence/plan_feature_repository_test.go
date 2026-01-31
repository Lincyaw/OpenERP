package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupPlanFeatureTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Create plan_features table
	err = db.Exec(`
		CREATE TABLE plan_features (
			id TEXT PRIMARY KEY,
			plan_id TEXT NOT NULL,
			feature_key TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 0,
			feature_limit INTEGER,
			description TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			UNIQUE(plan_id, feature_key)
		)
	`).Error
	require.NoError(t, err)

	// Create plan_feature_change_logs table
	err = db.Exec(`
		CREATE TABLE plan_feature_change_logs (
			id TEXT PRIMARY KEY,
			plan_id TEXT NOT NULL,
			feature_key TEXT NOT NULL,
			change_type TEXT NOT NULL,
			old_enabled INTEGER,
			new_enabled INTEGER,
			old_limit INTEGER,
			new_limit INTEGER,
			changed_by TEXT,
			changed_at DATETIME NOT NULL
		)
	`).Error
	require.NoError(t, err)

	return db
}

func TestGormPlanFeatureRepository_Save(t *testing.T) {
	db := setupPlanFeatureTestDB(t)
	repo := NewGormPlanFeatureRepository(db)
	ctx := context.Background()

	// Create a plan feature
	feature := &identity.PlanFeature{
		ID:          uuid.New(),
		PlanID:      identity.TenantPlanBasic,
		FeatureKey:  identity.FeatureMultiWarehouse,
		Enabled:     true,
		Limit:       nil,
		Description: "Multiple warehouse management",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save the feature
	err := repo.Save(ctx, feature)
	require.NoError(t, err)

	// Retrieve and verify
	retrieved, err := repo.FindByID(ctx, feature.ID)
	require.NoError(t, err)
	assert.Equal(t, feature.ID, retrieved.ID)
	assert.Equal(t, feature.PlanID, retrieved.PlanID)
	assert.Equal(t, feature.FeatureKey, retrieved.FeatureKey)
	assert.Equal(t, feature.Enabled, retrieved.Enabled)
	assert.Nil(t, retrieved.Limit)
}

func TestGormPlanFeatureRepository_SaveWithLimit(t *testing.T) {
	db := setupPlanFeatureTestDB(t)
	repo := NewGormPlanFeatureRepository(db)
	ctx := context.Background()

	limit := 1000
	feature := &identity.PlanFeature{
		ID:          uuid.New(),
		PlanID:      identity.TenantPlanBasic,
		FeatureKey:  identity.FeatureDataImport,
		Enabled:     true,
		Limit:       &limit,
		Description: "Import data from CSV",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := repo.Save(ctx, feature)
	require.NoError(t, err)

	retrieved, err := repo.FindByID(ctx, feature.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved.Limit)
	assert.Equal(t, limit, *retrieved.Limit)
}

func TestGormPlanFeatureRepository_FindByPlan(t *testing.T) {
	db := setupPlanFeatureTestDB(t)
	repo := NewGormPlanFeatureRepository(db)
	ctx := context.Background()

	// Create multiple features for the same plan
	features := []identity.PlanFeature{
		{
			ID:          uuid.New(),
			PlanID:      identity.TenantPlanPro,
			FeatureKey:  identity.FeatureMultiWarehouse,
			Enabled:     true,
			Description: "Multiple warehouse management",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          uuid.New(),
			PlanID:      identity.TenantPlanPro,
			FeatureKey:  identity.FeatureBatchManagement,
			Enabled:     true,
			Description: "Batch management",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          uuid.New(),
			PlanID:      identity.TenantPlanBasic, // Different plan
			FeatureKey:  identity.FeatureMultiWarehouse,
			Enabled:     false,
			Description: "Multiple warehouse management",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	for _, f := range features {
		err := repo.Save(ctx, &f)
		require.NoError(t, err)
	}

	// Find features for Pro plan
	proFeatures, err := repo.FindByPlan(ctx, identity.TenantPlanPro)
	require.NoError(t, err)
	assert.Len(t, proFeatures, 2)

	// Find features for Basic plan
	basicFeatures, err := repo.FindByPlan(ctx, identity.TenantPlanBasic)
	require.NoError(t, err)
	assert.Len(t, basicFeatures, 1)
}

func TestGormPlanFeatureRepository_FindByPlanAndFeature(t *testing.T) {
	db := setupPlanFeatureTestDB(t)
	repo := NewGormPlanFeatureRepository(db)
	ctx := context.Background()

	feature := &identity.PlanFeature{
		ID:          uuid.New(),
		PlanID:      identity.TenantPlanEnterprise,
		FeatureKey:  identity.FeatureAPIAccess,
		Enabled:     true,
		Description: "API access",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := repo.Save(ctx, feature)
	require.NoError(t, err)

	// Find by plan and feature
	retrieved, err := repo.FindByPlanAndFeature(ctx, identity.TenantPlanEnterprise, identity.FeatureAPIAccess)
	require.NoError(t, err)
	assert.Equal(t, feature.ID, retrieved.ID)

	// Not found case
	_, err = repo.FindByPlanAndFeature(ctx, identity.TenantPlanFree, identity.FeatureAPIAccess)
	assert.Equal(t, shared.ErrNotFound, err)
}

func TestGormPlanFeatureRepository_FindEnabledByPlan(t *testing.T) {
	db := setupPlanFeatureTestDB(t)
	repo := NewGormPlanFeatureRepository(db)
	ctx := context.Background()

	features := []identity.PlanFeature{
		{
			ID:          uuid.New(),
			PlanID:      identity.TenantPlanBasic,
			FeatureKey:  identity.FeatureMultiWarehouse,
			Enabled:     true,
			Description: "Enabled feature",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          uuid.New(),
			PlanID:      identity.TenantPlanBasic,
			FeatureKey:  identity.FeatureAPIAccess,
			Enabled:     false,
			Description: "Disabled feature",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	for _, f := range features {
		err := repo.Save(ctx, &f)
		require.NoError(t, err)
	}

	enabledFeatures, err := repo.FindEnabledByPlan(ctx, identity.TenantPlanBasic)
	require.NoError(t, err)
	assert.Len(t, enabledFeatures, 1)
	assert.Equal(t, identity.FeatureMultiWarehouse, enabledFeatures[0].FeatureKey)
}

func TestGormPlanFeatureRepository_HasFeature(t *testing.T) {
	db := setupPlanFeatureTestDB(t)
	repo := NewGormPlanFeatureRepository(db)
	ctx := context.Background()

	feature := &identity.PlanFeature{
		ID:          uuid.New(),
		PlanID:      identity.TenantPlanPro,
		FeatureKey:  identity.FeatureMultiCurrency,
		Enabled:     true,
		Description: "Multi-currency support",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := repo.Save(ctx, feature)
	require.NoError(t, err)

	// Has enabled feature
	has, err := repo.HasFeature(ctx, identity.TenantPlanPro, identity.FeatureMultiCurrency)
	require.NoError(t, err)
	assert.True(t, has)

	// Does not have feature (different plan)
	has, err = repo.HasFeature(ctx, identity.TenantPlanFree, identity.FeatureMultiCurrency)
	require.NoError(t, err)
	assert.False(t, has)
}

func TestGormPlanFeatureRepository_GetFeatureLimit(t *testing.T) {
	db := setupPlanFeatureTestDB(t)
	repo := NewGormPlanFeatureRepository(db)
	ctx := context.Background()

	limit := 5000
	feature := &identity.PlanFeature{
		ID:          uuid.New(),
		PlanID:      identity.TenantPlanBasic,
		FeatureKey:  identity.FeatureDataImport,
		Enabled:     true,
		Limit:       &limit,
		Description: "Data import with limit",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := repo.Save(ctx, feature)
	require.NoError(t, err)

	// Get limit
	retrievedLimit, err := repo.GetFeatureLimit(ctx, identity.TenantPlanBasic, identity.FeatureDataImport)
	require.NoError(t, err)
	require.NotNil(t, retrievedLimit)
	assert.Equal(t, limit, *retrievedLimit)

	// No limit (feature not found)
	retrievedLimit, err = repo.GetFeatureLimit(ctx, identity.TenantPlanFree, identity.FeatureDataImport)
	require.NoError(t, err)
	assert.Nil(t, retrievedLimit)
}

func TestGormPlanFeatureRepository_SaveBatch(t *testing.T) {
	db := setupPlanFeatureTestDB(t)
	repo := NewGormPlanFeatureRepository(db)
	ctx := context.Background()

	features := []identity.PlanFeature{
		{
			ID:          uuid.New(),
			PlanID:      identity.TenantPlanEnterprise,
			FeatureKey:  identity.FeatureWhiteLabeling,
			Enabled:     true,
			Description: "White labeling",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          uuid.New(),
			PlanID:      identity.TenantPlanEnterprise,
			FeatureKey:  identity.FeatureDedicatedSupport,
			Enabled:     true,
			Description: "Dedicated support",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	err := repo.SaveBatch(ctx, features)
	require.NoError(t, err)

	// Verify all features were saved
	retrieved, err := repo.FindByPlan(ctx, identity.TenantPlanEnterprise)
	require.NoError(t, err)
	assert.Len(t, retrieved, 2)
}

func TestGormPlanFeatureRepository_Delete(t *testing.T) {
	db := setupPlanFeatureTestDB(t)
	repo := NewGormPlanFeatureRepository(db)
	ctx := context.Background()

	feature := &identity.PlanFeature{
		ID:          uuid.New(),
		PlanID:      identity.TenantPlanBasic,
		FeatureKey:  identity.FeatureAuditLog,
		Enabled:     true,
		Description: "Audit log",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := repo.Save(ctx, feature)
	require.NoError(t, err)

	// Delete
	err = repo.Delete(ctx, feature.ID)
	require.NoError(t, err)

	// Verify deleted
	_, err = repo.FindByID(ctx, feature.ID)
	assert.Equal(t, shared.ErrNotFound, err)

	// Delete non-existent
	err = repo.Delete(ctx, uuid.New())
	assert.Equal(t, shared.ErrNotFound, err)
}

func TestGormPlanFeatureRepository_DeleteByPlan(t *testing.T) {
	db := setupPlanFeatureTestDB(t)
	repo := NewGormPlanFeatureRepository(db)
	ctx := context.Background()

	features := []identity.PlanFeature{
		{
			ID:          uuid.New(),
			PlanID:      identity.TenantPlanFree,
			FeatureKey:  identity.FeatureSalesOrders,
			Enabled:     true,
			Description: "Sales orders",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          uuid.New(),
			PlanID:      identity.TenantPlanFree,
			FeatureKey:  identity.FeaturePurchaseOrders,
			Enabled:     true,
			Description: "Purchase orders",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	for _, f := range features {
		err := repo.Save(ctx, &f)
		require.NoError(t, err)
	}

	// Delete all features for Free plan
	err := repo.DeleteByPlan(ctx, identity.TenantPlanFree)
	require.NoError(t, err)

	// Verify all deleted
	retrieved, err := repo.FindByPlan(ctx, identity.TenantPlanFree)
	require.NoError(t, err)
	assert.Len(t, retrieved, 0)
}

func TestGormPlanFeatureRepository_SaveBatchWithAuditLog(t *testing.T) {
	db := setupPlanFeatureTestDB(t)
	repo := NewGormPlanFeatureRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	features := []identity.PlanFeature{
		{
			ID:          uuid.New(),
			PlanID:      identity.TenantPlanPro,
			FeatureKey:  identity.FeatureIntegrations,
			Enabled:     true,
			Description: "Third-party integrations",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	// Save with audit log
	err := repo.SaveBatchWithAuditLog(ctx, features, &userID)
	require.NoError(t, err)

	// Verify feature was saved
	retrieved, err := repo.FindByPlanAndFeature(ctx, identity.TenantPlanPro, identity.FeatureIntegrations)
	require.NoError(t, err)
	assert.True(t, retrieved.Enabled)

	// Verify audit log was created
	var count int64
	db.Table("plan_feature_change_logs").Count(&count)
	assert.Equal(t, int64(1), count)

	// Update the feature
	features[0].Enabled = false
	features[0].UpdatedAt = time.Now()
	err = repo.SaveBatchWithAuditLog(ctx, features, &userID)
	require.NoError(t, err)

	// Verify audit log count increased
	db.Table("plan_feature_change_logs").Count(&count)
	assert.Equal(t, int64(2), count)
}

func TestGormPlanFeatureRepository_Update(t *testing.T) {
	db := setupPlanFeatureTestDB(t)
	repo := NewGormPlanFeatureRepository(db)
	ctx := context.Background()

	feature := &identity.PlanFeature{
		ID:          uuid.New(),
		PlanID:      identity.TenantPlanBasic,
		FeatureKey:  identity.FeatureNotifications,
		Enabled:     false,
		Description: "Notifications",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save initial
	err := repo.Save(ctx, feature)
	require.NoError(t, err)

	// Update
	feature.Enabled = true
	limit := 100
	feature.Limit = &limit
	feature.UpdatedAt = time.Now()

	err = repo.Save(ctx, feature)
	require.NoError(t, err)

	// Verify update
	retrieved, err := repo.FindByID(ctx, feature.ID)
	require.NoError(t, err)
	assert.True(t, retrieved.Enabled)
	require.NotNil(t, retrieved.Limit)
	assert.Equal(t, 100, *retrieved.Limit)
}
