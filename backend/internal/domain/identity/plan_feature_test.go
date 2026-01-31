package identity

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPlanFeature(t *testing.T) {
	t.Run("creates plan feature successfully", func(t *testing.T) {
		pf := NewPlanFeature(TenantPlanBasic, FeatureMultiWarehouse, true, "Multiple warehouse management")

		require.NotNil(t, pf)
		assert.NotEqual(t, uuid.Nil, pf.ID)
		assert.Equal(t, TenantPlanBasic, pf.PlanID)
		assert.Equal(t, FeatureMultiWarehouse, pf.FeatureKey)
		assert.True(t, pf.Enabled)
		assert.Nil(t, pf.Limit)
		assert.Equal(t, "Multiple warehouse management", pf.Description)
		assert.False(t, pf.CreatedAt.IsZero())
		assert.False(t, pf.UpdatedAt.IsZero())
	})

	t.Run("creates disabled plan feature", func(t *testing.T) {
		pf := NewPlanFeature(TenantPlanFree, FeatureAPIAccess, false, "API access for integrations")

		require.NotNil(t, pf)
		assert.Equal(t, TenantPlanFree, pf.PlanID)
		assert.Equal(t, FeatureAPIAccess, pf.FeatureKey)
		assert.False(t, pf.Enabled)
	})
}

func TestNewPlanFeatureWithLimit(t *testing.T) {
	t.Run("creates plan feature with limit", func(t *testing.T) {
		pf := NewPlanFeatureWithLimit(TenantPlanBasic, FeatureDataImport, true, 1000, "Import data from CSV")

		require.NotNil(t, pf)
		assert.Equal(t, TenantPlanBasic, pf.PlanID)
		assert.Equal(t, FeatureDataImport, pf.FeatureKey)
		assert.True(t, pf.Enabled)
		require.NotNil(t, pf.Limit)
		assert.Equal(t, 1000, *pf.Limit)
		assert.Equal(t, "Import data from CSV", pf.Description)
	})

	t.Run("creates plan feature with zero limit", func(t *testing.T) {
		pf := NewPlanFeatureWithLimit(TenantPlanFree, FeatureDataImport, true, 0, "Import disabled")

		require.NotNil(t, pf)
		require.NotNil(t, pf.Limit)
		assert.Equal(t, 0, *pf.Limit)
	})
}

func TestNewValidatedPlanFeature(t *testing.T) {
	t.Run("creates validated plan feature successfully", func(t *testing.T) {
		pf, err := NewValidatedPlanFeature(TenantPlanBasic, FeatureMultiWarehouse, true, "Multiple warehouse management")

		require.NoError(t, err)
		require.NotNil(t, pf)
		assert.Equal(t, TenantPlanBasic, pf.PlanID)
		assert.Equal(t, FeatureMultiWarehouse, pf.FeatureKey)
		assert.True(t, pf.Enabled)
	})

	t.Run("rejects invalid plan ID", func(t *testing.T) {
		pf, err := NewValidatedPlanFeature(TenantPlan("invalid"), FeatureMultiWarehouse, true, "Test")

		assert.Error(t, err)
		assert.Nil(t, pf)
		assert.Contains(t, err.Error(), "Invalid tenant plan")
	})

	t.Run("rejects invalid feature key", func(t *testing.T) {
		pf, err := NewValidatedPlanFeature(TenantPlanBasic, FeatureKey("invalid_feature"), true, "Test")

		assert.Error(t, err)
		assert.Nil(t, pf)
		assert.Contains(t, err.Error(), "Invalid feature key")
	})
}

func TestNewValidatedPlanFeatureWithLimit(t *testing.T) {
	t.Run("creates validated plan feature with limit", func(t *testing.T) {
		pf, err := NewValidatedPlanFeatureWithLimit(TenantPlanBasic, FeatureDataImport, true, 1000, "Import data")

		require.NoError(t, err)
		require.NotNil(t, pf)
		assert.Equal(t, TenantPlanBasic, pf.PlanID)
		assert.Equal(t, FeatureDataImport, pf.FeatureKey)
		require.NotNil(t, pf.Limit)
		assert.Equal(t, 1000, *pf.Limit)
	})

	t.Run("rejects negative limit", func(t *testing.T) {
		pf, err := NewValidatedPlanFeatureWithLimit(TenantPlanBasic, FeatureDataImport, true, -1, "Import data")

		assert.Error(t, err)
		assert.Nil(t, pf)
		assert.Contains(t, err.Error(), "cannot be negative")
	})

	t.Run("allows zero limit", func(t *testing.T) {
		pf, err := NewValidatedPlanFeatureWithLimit(TenantPlanFree, FeatureDataImport, true, 0, "Import disabled")

		require.NoError(t, err)
		require.NotNil(t, pf)
		require.NotNil(t, pf.Limit)
		assert.Equal(t, 0, *pf.Limit)
	})

	t.Run("rejects invalid plan ID", func(t *testing.T) {
		pf, err := NewValidatedPlanFeatureWithLimit(TenantPlan("invalid"), FeatureDataImport, true, 100, "Test")

		assert.Error(t, err)
		assert.Nil(t, pf)
	})

	t.Run("rejects invalid feature key", func(t *testing.T) {
		pf, err := NewValidatedPlanFeatureWithLimit(TenantPlanBasic, FeatureKey("invalid"), true, 100, "Test")

		assert.Error(t, err)
		assert.Nil(t, pf)
	})
}

func TestPlanFeature_SetLimit(t *testing.T) {
	t.Run("sets limit on unlimited feature", func(t *testing.T) {
		pf := NewPlanFeature(TenantPlanPro, FeatureDataImport, true, "Import data")
		assert.Nil(t, pf.Limit)
		initialUpdatedAt := pf.UpdatedAt

		err := pf.SetLimit(5000)

		require.NoError(t, err)
		require.NotNil(t, pf.Limit)
		assert.Equal(t, 5000, *pf.Limit)
		assert.True(t, pf.UpdatedAt.After(initialUpdatedAt) || pf.UpdatedAt.Equal(initialUpdatedAt))
	})

	t.Run("updates existing limit", func(t *testing.T) {
		pf := NewPlanFeatureWithLimit(TenantPlanBasic, FeatureDataImport, true, 1000, "Import data")

		err := pf.SetLimit(2000)

		require.NoError(t, err)
		require.NotNil(t, pf.Limit)
		assert.Equal(t, 2000, *pf.Limit)
	})

	t.Run("rejects negative limit", func(t *testing.T) {
		pf := NewPlanFeature(TenantPlanPro, FeatureDataImport, true, "Import data")

		err := pf.SetLimit(-1)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be negative")
		assert.Nil(t, pf.Limit) // Limit should not be set
	})

	t.Run("allows zero limit", func(t *testing.T) {
		pf := NewPlanFeature(TenantPlanFree, FeatureDataImport, true, "Import data")

		err := pf.SetLimit(0)

		require.NoError(t, err)
		require.NotNil(t, pf.Limit)
		assert.Equal(t, 0, *pf.Limit)
	})
}

func TestPlanFeature_ClearLimit(t *testing.T) {
	t.Run("clears existing limit", func(t *testing.T) {
		pf := NewPlanFeatureWithLimit(TenantPlanBasic, FeatureDataImport, true, 1000, "Import data")
		require.NotNil(t, pf.Limit)

		pf.ClearLimit()

		assert.Nil(t, pf.Limit)
	})

	t.Run("clearing already unlimited feature is safe", func(t *testing.T) {
		pf := NewPlanFeature(TenantPlanEnterprise, FeatureDataImport, true, "Import data")
		assert.Nil(t, pf.Limit)

		pf.ClearLimit()

		assert.Nil(t, pf.Limit)
	})
}

func TestPlanFeature_Enable(t *testing.T) {
	t.Run("enables disabled feature", func(t *testing.T) {
		pf := NewPlanFeature(TenantPlanFree, FeatureAPIAccess, false, "API access")
		assert.False(t, pf.Enabled)

		pf.Enable()

		assert.True(t, pf.Enabled)
	})

	t.Run("enabling already enabled feature is safe", func(t *testing.T) {
		pf := NewPlanFeature(TenantPlanPro, FeatureAPIAccess, true, "API access")
		assert.True(t, pf.Enabled)

		pf.Enable()

		assert.True(t, pf.Enabled)
	})
}

func TestPlanFeature_Disable(t *testing.T) {
	t.Run("disables enabled feature", func(t *testing.T) {
		pf := NewPlanFeature(TenantPlanPro, FeatureAPIAccess, true, "API access")
		assert.True(t, pf.Enabled)

		pf.Disable()

		assert.False(t, pf.Enabled)
	})

	t.Run("disabling already disabled feature is safe", func(t *testing.T) {
		pf := NewPlanFeature(TenantPlanFree, FeatureAPIAccess, false, "API access")
		assert.False(t, pf.Enabled)

		pf.Disable()

		assert.False(t, pf.Enabled)
	})
}

func TestPlanFeature_IsUnlimited(t *testing.T) {
	t.Run("returns true for unlimited feature", func(t *testing.T) {
		pf := NewPlanFeature(TenantPlanEnterprise, FeatureDataImport, true, "Import data")

		assert.True(t, pf.IsUnlimited())
	})

	t.Run("returns false for limited feature", func(t *testing.T) {
		pf := NewPlanFeatureWithLimit(TenantPlanBasic, FeatureDataImport, true, 1000, "Import data")

		assert.False(t, pf.IsUnlimited())
	})

	t.Run("returns false for zero limit", func(t *testing.T) {
		pf := NewPlanFeatureWithLimit(TenantPlanFree, FeatureDataImport, true, 0, "Import data")

		assert.False(t, pf.IsUnlimited())
	})
}

func TestPlanFeature_GetLimit(t *testing.T) {
	t.Run("returns -1 for unlimited feature", func(t *testing.T) {
		pf := NewPlanFeature(TenantPlanEnterprise, FeatureDataImport, true, "Import data")

		assert.Equal(t, -1, pf.GetLimit())
	})

	t.Run("returns actual limit for limited feature", func(t *testing.T) {
		pf := NewPlanFeatureWithLimit(TenantPlanBasic, FeatureDataImport, true, 1000, "Import data")

		assert.Equal(t, 1000, pf.GetLimit())
	})

	t.Run("returns zero for zero limit", func(t *testing.T) {
		pf := NewPlanFeatureWithLimit(TenantPlanFree, FeatureDataImport, true, 0, "Import data")

		assert.Equal(t, 0, pf.GetLimit())
	})
}

func TestDefaultPlanFeatures(t *testing.T) {
	t.Run("free plan has correct features", func(t *testing.T) {
		features := DefaultPlanFeatures(TenantPlanFree)

		require.NotEmpty(t, features)
		// Check some specific features
		featureMap := makeFeatureMap(features)

		// Free plan should have basic trade features enabled
		assert.True(t, featureMap[FeatureSalesOrders].Enabled)
		assert.True(t, featureMap[FeaturePurchaseOrders].Enabled)
		assert.True(t, featureMap[FeatureDataExport].Enabled)

		// Free plan should have advanced features disabled
		assert.False(t, featureMap[FeatureMultiWarehouse].Enabled)
		assert.False(t, featureMap[FeatureAPIAccess].Enabled)
		assert.False(t, featureMap[FeatureAdvancedReporting].Enabled)

		// Free plan data import should have limit of 100
		assert.True(t, featureMap[FeatureDataImport].Enabled)
		require.NotNil(t, featureMap[FeatureDataImport].Limit)
		assert.Equal(t, 100, *featureMap[FeatureDataImport].Limit)
	})

	t.Run("basic plan has more features than free", func(t *testing.T) {
		features := DefaultPlanFeatures(TenantPlanBasic)
		featureMap := makeFeatureMap(features)

		// Basic plan should have multi-warehouse enabled
		assert.True(t, featureMap[FeatureMultiWarehouse].Enabled)
		assert.True(t, featureMap[FeatureBatchManagement].Enabled)
		assert.True(t, featureMap[FeatureAuditLog].Enabled)

		// Basic plan data import should have limit of 1000
		require.NotNil(t, featureMap[FeatureDataImport].Limit)
		assert.Equal(t, 1000, *featureMap[FeatureDataImport].Limit)

		// Basic plan should still have some features disabled
		assert.False(t, featureMap[FeatureAPIAccess].Enabled)
		assert.False(t, featureMap[FeatureAdvancedReporting].Enabled)
	})

	t.Run("pro plan has most features enabled", func(t *testing.T) {
		features := DefaultPlanFeatures(TenantPlanPro)
		featureMap := makeFeatureMap(features)

		// Pro plan should have advanced features enabled
		assert.True(t, featureMap[FeatureAPIAccess].Enabled)
		assert.True(t, featureMap[FeatureAdvancedReporting].Enabled)
		assert.True(t, featureMap[FeatureMultiCurrency].Enabled)
		assert.True(t, featureMap[FeatureSerialTracking].Enabled)
		assert.True(t, featureMap[FeatureWorkflowApproval].Enabled)

		// Pro plan data import should have limit of 10000
		require.NotNil(t, featureMap[FeatureDataImport].Limit)
		assert.Equal(t, 10000, *featureMap[FeatureDataImport].Limit)

		// Pro plan should still have enterprise-only features disabled
		assert.False(t, featureMap[FeatureWhiteLabeling].Enabled)
		assert.False(t, featureMap[FeatureDedicatedSupport].Enabled)
	})

	t.Run("enterprise plan has all features enabled", func(t *testing.T) {
		features := DefaultPlanFeatures(TenantPlanEnterprise)
		featureMap := makeFeatureMap(features)

		// Enterprise plan should have all features enabled
		assert.True(t, featureMap[FeatureWhiteLabeling].Enabled)
		assert.True(t, featureMap[FeatureDedicatedSupport].Enabled)
		assert.True(t, featureMap[FeatureSLA].Enabled)
		assert.True(t, featureMap[FeatureAPIAccess].Enabled)

		// Enterprise plan data import should be unlimited
		assert.Nil(t, featureMap[FeatureDataImport].Limit)
	})

	t.Run("invalid plan returns free plan features", func(t *testing.T) {
		features := DefaultPlanFeatures(TenantPlan("invalid"))
		freeFeatures := DefaultPlanFeatures(TenantPlanFree)

		assert.Equal(t, len(freeFeatures), len(features))
	})

	t.Run("all plans have same number of features", func(t *testing.T) {
		freeFeatures := DefaultPlanFeatures(TenantPlanFree)
		basicFeatures := DefaultPlanFeatures(TenantPlanBasic)
		proFeatures := DefaultPlanFeatures(TenantPlanPro)
		enterpriseFeatures := DefaultPlanFeatures(TenantPlanEnterprise)

		assert.Equal(t, len(freeFeatures), len(basicFeatures))
		assert.Equal(t, len(basicFeatures), len(proFeatures))
		assert.Equal(t, len(proFeatures), len(enterpriseFeatures))
	})
}

func TestGetAllFeatureKeys(t *testing.T) {
	t.Run("returns all feature keys", func(t *testing.T) {
		keys := GetAllFeatureKeys()

		require.NotEmpty(t, keys)
		// Check that we have the expected number of features
		assert.GreaterOrEqual(t, len(keys), 25) // We have at least 25 features defined

		// Check some specific keys exist
		assert.Contains(t, keys, FeatureMultiWarehouse)
		assert.Contains(t, keys, FeatureAPIAccess)
		assert.Contains(t, keys, FeatureSalesOrders)
		assert.Contains(t, keys, FeatureReceivables)
		assert.Contains(t, keys, FeatureWhiteLabeling)
	})

	t.Run("all keys are unique", func(t *testing.T) {
		keys := GetAllFeatureKeys()
		seen := make(map[FeatureKey]bool)

		for _, key := range keys {
			assert.False(t, seen[key], "Duplicate feature key: %s", key)
			seen[key] = true
		}
	})
}

func TestIsValidFeatureKey(t *testing.T) {
	t.Run("returns true for valid feature keys", func(t *testing.T) {
		assert.True(t, IsValidFeatureKey(FeatureMultiWarehouse))
		assert.True(t, IsValidFeatureKey(FeatureAPIAccess))
		assert.True(t, IsValidFeatureKey(FeatureSalesOrders))
		assert.True(t, IsValidFeatureKey(FeatureWhiteLabeling))
		assert.True(t, IsValidFeatureKey(FeatureSLA))
	})

	t.Run("returns false for invalid feature keys", func(t *testing.T) {
		assert.False(t, IsValidFeatureKey(FeatureKey("invalid_feature")))
		assert.False(t, IsValidFeatureKey(FeatureKey("")))
		assert.False(t, IsValidFeatureKey(FeatureKey("unknown")))
	})
}

func TestPlanHasFeature(t *testing.T) {
	t.Run("free plan feature checks", func(t *testing.T) {
		// Free plan has basic trade features
		assert.True(t, PlanHasFeature(TenantPlanFree, FeatureSalesOrders))
		assert.True(t, PlanHasFeature(TenantPlanFree, FeaturePurchaseOrders))
		assert.True(t, PlanHasFeature(TenantPlanFree, FeatureDataExport))

		// Free plan does not have advanced features
		assert.False(t, PlanHasFeature(TenantPlanFree, FeatureMultiWarehouse))
		assert.False(t, PlanHasFeature(TenantPlanFree, FeatureAPIAccess))
	})

	t.Run("basic plan feature checks", func(t *testing.T) {
		assert.True(t, PlanHasFeature(TenantPlanBasic, FeatureMultiWarehouse))
		assert.True(t, PlanHasFeature(TenantPlanBasic, FeatureBatchManagement))
		assert.False(t, PlanHasFeature(TenantPlanBasic, FeatureAPIAccess))
	})

	t.Run("pro plan feature checks", func(t *testing.T) {
		assert.True(t, PlanHasFeature(TenantPlanPro, FeatureAPIAccess))
		assert.True(t, PlanHasFeature(TenantPlanPro, FeatureAdvancedReporting))
		assert.False(t, PlanHasFeature(TenantPlanPro, FeatureWhiteLabeling))
	})

	t.Run("enterprise plan has all features", func(t *testing.T) {
		assert.True(t, PlanHasFeature(TenantPlanEnterprise, FeatureWhiteLabeling))
		assert.True(t, PlanHasFeature(TenantPlanEnterprise, FeatureDedicatedSupport))
		assert.True(t, PlanHasFeature(TenantPlanEnterprise, FeatureSLA))
	})

	t.Run("returns false for unknown feature", func(t *testing.T) {
		assert.False(t, PlanHasFeature(TenantPlanEnterprise, FeatureKey("unknown_feature")))
	})
}

func TestGetPlanFeatureLimit(t *testing.T) {
	t.Run("free plan data import limit", func(t *testing.T) {
		limit := GetPlanFeatureLimit(TenantPlanFree, FeatureDataImport)

		require.NotNil(t, limit)
		assert.Equal(t, 100, *limit)
	})

	t.Run("basic plan data import limit", func(t *testing.T) {
		limit := GetPlanFeatureLimit(TenantPlanBasic, FeatureDataImport)

		require.NotNil(t, limit)
		assert.Equal(t, 1000, *limit)
	})

	t.Run("pro plan data import limit", func(t *testing.T) {
		limit := GetPlanFeatureLimit(TenantPlanPro, FeatureDataImport)

		require.NotNil(t, limit)
		assert.Equal(t, 10000, *limit)
	})

	t.Run("enterprise plan data import is unlimited", func(t *testing.T) {
		limit := GetPlanFeatureLimit(TenantPlanEnterprise, FeatureDataImport)

		assert.Nil(t, limit)
	})

	t.Run("unlimited features return nil", func(t *testing.T) {
		// Most features don't have limits
		limit := GetPlanFeatureLimit(TenantPlanPro, FeatureAPIAccess)
		assert.Nil(t, limit)

		limit = GetPlanFeatureLimit(TenantPlanBasic, FeatureMultiWarehouse)
		assert.Nil(t, limit)
	})

	t.Run("unknown feature returns nil", func(t *testing.T) {
		limit := GetPlanFeatureLimit(TenantPlanEnterprise, FeatureKey("unknown"))
		assert.Nil(t, limit)
	})
}

func TestFeatureKeyConstants(t *testing.T) {
	t.Run("feature keys have expected values", func(t *testing.T) {
		// Core features
		assert.Equal(t, FeatureKey("multi_warehouse"), FeatureMultiWarehouse)
		assert.Equal(t, FeatureKey("batch_management"), FeatureBatchManagement)
		assert.Equal(t, FeatureKey("serial_tracking"), FeatureSerialTracking)
		assert.Equal(t, FeatureKey("api_access"), FeatureAPIAccess)

		// Trade features
		assert.Equal(t, FeatureKey("sales_orders"), FeatureSalesOrders)
		assert.Equal(t, FeatureKey("purchase_orders"), FeaturePurchaseOrders)

		// Finance features
		assert.Equal(t, FeatureKey("receivables"), FeatureReceivables)
		assert.Equal(t, FeatureKey("payables"), FeaturePayables)

		// Advanced features
		assert.Equal(t, FeatureKey("white_labeling"), FeatureWhiteLabeling)
		assert.Equal(t, FeatureKey("sla"), FeatureSLA)
	})
}

func TestPlanFeatureProgression(t *testing.T) {
	t.Run("higher plans have more enabled features", func(t *testing.T) {
		freeEnabled := countEnabledFeatures(DefaultPlanFeatures(TenantPlanFree))
		basicEnabled := countEnabledFeatures(DefaultPlanFeatures(TenantPlanBasic))
		proEnabled := countEnabledFeatures(DefaultPlanFeatures(TenantPlanPro))
		enterpriseEnabled := countEnabledFeatures(DefaultPlanFeatures(TenantPlanEnterprise))

		assert.Less(t, freeEnabled, basicEnabled, "Basic should have more features than Free")
		assert.Less(t, basicEnabled, proEnabled, "Pro should have more features than Basic")
		assert.LessOrEqual(t, proEnabled, enterpriseEnabled, "Enterprise should have at least as many features as Pro")
	})

	t.Run("enterprise plan has all features enabled", func(t *testing.T) {
		features := DefaultPlanFeatures(TenantPlanEnterprise)
		for _, f := range features {
			assert.True(t, f.Enabled, "Enterprise feature %s should be enabled", f.FeatureKey)
		}
	})
}

// Helper functions for tests

func makeFeatureMap(features []PlanFeature) map[FeatureKey]PlanFeature {
	m := make(map[FeatureKey]PlanFeature)
	for _, f := range features {
		m[f.FeatureKey] = f
	}
	return m
}

func countEnabledFeatures(features []PlanFeature) int {
	count := 0
	for _, f := range features {
		if f.Enabled {
			count++
		}
	}
	return count
}
