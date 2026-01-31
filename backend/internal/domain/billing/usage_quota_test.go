package billing

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUsageQuota(t *testing.T) {
	t.Run("creates valid quota", func(t *testing.T) {
		quota, err := NewUsageQuota("basic", UsageTypeAPICalls, 10000, ResetPeriodMonthly)

		require.NoError(t, err)
		assert.NotNil(t, quota)
		assert.Equal(t, "basic", quota.PlanID)
		assert.Equal(t, UsageTypeAPICalls, quota.UsageType)
		assert.Equal(t, int64(10000), quota.Limit)
		assert.Equal(t, UsageUnitRequests, quota.Unit)
		assert.Equal(t, ResetPeriodMonthly, quota.ResetPeriod)
		assert.Equal(t, OveragePolicyBlock, quota.OveragePolicy)
		assert.True(t, quota.IsActive)
		assert.Nil(t, quota.TenantID)
	})

	t.Run("creates unlimited quota", func(t *testing.T) {
		quota, err := NewUsageQuota("enterprise", UsageTypeAPICalls, -1, ResetPeriodMonthly)

		require.NoError(t, err)
		assert.Equal(t, int64(-1), quota.Limit)
		assert.True(t, quota.IsUnlimited())
	})

	t.Run("fails with empty plan ID", func(t *testing.T) {
		quota, err := NewUsageQuota("", UsageTypeAPICalls, 10000, ResetPeriodMonthly)

		assert.Error(t, err)
		assert.Nil(t, quota)
		assert.Contains(t, err.Error(), "Plan ID cannot be empty")
	})

	t.Run("fails with invalid usage type", func(t *testing.T) {
		quota, err := NewUsageQuota("basic", UsageType("INVALID"), 10000, ResetPeriodMonthly)

		assert.Error(t, err)
		assert.Nil(t, quota)
		assert.Contains(t, err.Error(), "Invalid usage type")
	})

	t.Run("fails with invalid limit", func(t *testing.T) {
		quota, err := NewUsageQuota("basic", UsageTypeAPICalls, -2, ResetPeriodMonthly)

		assert.Error(t, err)
		assert.Nil(t, quota)
		assert.Contains(t, err.Error(), "Limit must be -1 (unlimited) or non-negative")
	})

	t.Run("fails with invalid reset period", func(t *testing.T) {
		quota, err := NewUsageQuota("basic", UsageTypeAPICalls, 10000, ResetPeriod("INVALID"))

		assert.Error(t, err)
		assert.Nil(t, quota)
		assert.Contains(t, err.Error(), "Invalid reset period")
	})
}

func TestNewTenantUsageQuota(t *testing.T) {
	tenantID := uuid.New()

	t.Run("creates tenant-specific quota", func(t *testing.T) {
		quota, err := NewTenantUsageQuota(tenantID, "basic", UsageTypeAPICalls, 20000, ResetPeriodMonthly)

		require.NoError(t, err)
		assert.NotNil(t, quota.TenantID)
		assert.Equal(t, tenantID, *quota.TenantID)
		assert.True(t, quota.IsTenantOverride())
	})

	t.Run("fails with nil tenant ID", func(t *testing.T) {
		quota, err := NewTenantUsageQuota(uuid.Nil, "basic", UsageTypeAPICalls, 20000, ResetPeriodMonthly)

		assert.Error(t, err)
		assert.Nil(t, quota)
		assert.Contains(t, err.Error(), "Tenant ID cannot be empty")
	})
}

func TestUsageQuota_WithSoftLimit(t *testing.T) {
	quota, _ := NewUsageQuota("basic", UsageTypeAPICalls, 10000, ResetPeriodMonthly)

	t.Run("sets valid soft limit", func(t *testing.T) {
		result := quota.WithSoftLimit(8000)

		require.NotNil(t, result.SoftLimit)
		assert.Equal(t, int64(8000), *result.SoftLimit)
	})

	t.Run("ignores soft limit >= hard limit", func(t *testing.T) {
		quota2, _ := NewUsageQuota("basic", UsageTypeAPICalls, 10000, ResetPeriodMonthly)
		result := quota2.WithSoftLimit(10000)

		assert.Nil(t, result.SoftLimit)
	})

	t.Run("allows soft limit for unlimited quota", func(t *testing.T) {
		unlimitedQuota, _ := NewUsageQuota("enterprise", UsageTypeAPICalls, -1, ResetPeriodMonthly)
		result := unlimitedQuota.WithSoftLimit(100000)

		require.NotNil(t, result.SoftLimit)
		assert.Equal(t, int64(100000), *result.SoftLimit)
	})
}

func TestUsageQuota_WithOveragePolicy(t *testing.T) {
	t.Run("sets valid policy", func(t *testing.T) {
		quota, _ := NewUsageQuota("basic", UsageTypeAPICalls, 10000, ResetPeriodMonthly)
		result := quota.WithOveragePolicy(OveragePolicyCharge)

		assert.Equal(t, OveragePolicyCharge, result.OveragePolicy)
	})

	t.Run("ignores invalid policy", func(t *testing.T) {
		quota, _ := NewUsageQuota("basic", UsageTypeAPICalls, 10000, ResetPeriodMonthly)
		result := quota.WithOveragePolicy(OveragePolicy("INVALID"))

		assert.Equal(t, OveragePolicyBlock, result.OveragePolicy) // Unchanged from default
	})
}

func TestUsageQuota_SetLimit(t *testing.T) {
	quota, _ := NewUsageQuota("basic", UsageTypeAPICalls, 10000, ResetPeriodMonthly)

	t.Run("updates limit", func(t *testing.T) {
		err := quota.SetLimit(20000)

		require.NoError(t, err)
		assert.Equal(t, int64(20000), quota.Limit)
	})

	t.Run("allows unlimited", func(t *testing.T) {
		err := quota.SetLimit(-1)

		require.NoError(t, err)
		assert.Equal(t, int64(-1), quota.Limit)
	})

	t.Run("fails with invalid limit", func(t *testing.T) {
		err := quota.SetLimit(-2)

		assert.Error(t, err)
	})
}

func TestUsageQuota_SetSoftLimit(t *testing.T) {
	quota, _ := NewUsageQuota("basic", UsageTypeAPICalls, 10000, ResetPeriodMonthly)

	t.Run("sets soft limit", func(t *testing.T) {
		softLimit := int64(8000)
		err := quota.SetSoftLimit(&softLimit)

		require.NoError(t, err)
		require.NotNil(t, quota.SoftLimit)
		assert.Equal(t, int64(8000), *quota.SoftLimit)
	})

	t.Run("clears soft limit", func(t *testing.T) {
		err := quota.SetSoftLimit(nil)

		require.NoError(t, err)
		assert.Nil(t, quota.SoftLimit)
	})

	t.Run("fails with negative soft limit", func(t *testing.T) {
		softLimit := int64(-1)
		err := quota.SetSoftLimit(&softLimit)

		assert.Error(t, err)
	})

	t.Run("fails with soft limit >= hard limit", func(t *testing.T) {
		softLimit := int64(10000)
		err := quota.SetSoftLimit(&softLimit)

		assert.Error(t, err)
	})
}

func TestUsageQuota_ActivateDeactivate(t *testing.T) {
	quota, _ := NewUsageQuota("basic", UsageTypeAPICalls, 10000, ResetPeriodMonthly)

	assert.True(t, quota.IsActive)

	quota.Deactivate()
	assert.False(t, quota.IsActive)

	quota.Activate()
	assert.True(t, quota.IsActive)
}

func TestUsageQuota_CheckUsage(t *testing.T) {
	t.Run("OK status when under limit", func(t *testing.T) {
		quota, _ := NewUsageQuota("basic", UsageTypeAPICalls, 10000, ResetPeriodMonthly)

		result := quota.CheckUsage(5000)

		assert.Equal(t, QuotaStatusOK, result.Status)
		assert.Equal(t, int64(5000), result.Remaining)
		assert.Equal(t, float64(50), result.UsagePercent)
		assert.Equal(t, int64(0), result.Overage)
		assert.True(t, result.IsAllowed())
		assert.False(t, result.ShouldWarn())
	})

	t.Run("WARNING status when at soft limit", func(t *testing.T) {
		quota, _ := NewUsageQuota("basic", UsageTypeAPICalls, 10000, ResetPeriodMonthly)
		quota.WithSoftLimit(8000)

		result := quota.CheckUsage(8500)

		assert.Equal(t, QuotaStatusWarning, result.Status)
		assert.True(t, result.IsAllowed())
		assert.True(t, result.ShouldWarn())
	})

	t.Run("EXCEEDED status when over limit", func(t *testing.T) {
		quota, _ := NewUsageQuota("basic", UsageTypeAPICalls, 10000, ResetPeriodMonthly)

		result := quota.CheckUsage(12000)

		assert.Equal(t, QuotaStatusExceeded, result.Status)
		assert.Equal(t, int64(-2000), result.Remaining)
		assert.Equal(t, int64(2000), result.Overage)
		assert.Equal(t, float64(120), result.UsagePercent)
		assert.False(t, result.IsAllowed()) // BLOCK policy
		assert.True(t, result.ShouldWarn())
	})

	t.Run("EXCEEDED with WARN policy is allowed", func(t *testing.T) {
		quota, _ := NewUsageQuota("basic", UsageTypeAPICalls, 10000, ResetPeriodMonthly)
		quota.WithOveragePolicy(OveragePolicyWarn)

		result := quota.CheckUsage(12000)

		assert.Equal(t, QuotaStatusExceeded, result.Status)
		assert.True(t, result.IsAllowed())
	})

	t.Run("EXCEEDED with CHARGE policy should charge", func(t *testing.T) {
		quota, _ := NewUsageQuota("pro", UsageTypeAPICalls, 10000, ResetPeriodMonthly)
		quota.WithOveragePolicy(OveragePolicyCharge)

		result := quota.CheckUsage(12000)

		assert.Equal(t, QuotaStatusExceeded, result.Status)
		assert.True(t, result.IsAllowed())
		assert.True(t, result.ShouldCharge())
	})

	t.Run("OK status for unlimited quota", func(t *testing.T) {
		quota, _ := NewUsageQuota("enterprise", UsageTypeAPICalls, -1, ResetPeriodMonthly)

		result := quota.CheckUsage(1000000)

		assert.Equal(t, QuotaStatusOK, result.Status)
		assert.True(t, result.IsUnlimited)
		assert.True(t, result.IsAllowed())
	})

	t.Run("INACTIVE status for inactive quota", func(t *testing.T) {
		quota, _ := NewUsageQuota("basic", UsageTypeAPICalls, 10000, ResetPeriodMonthly)
		quota.Deactivate()

		result := quota.CheckUsage(15000)

		assert.Equal(t, QuotaStatusInactive, result.Status)
		assert.True(t, result.IsAllowed())
	})
}

func TestUsageQuota_CanConsume(t *testing.T) {
	quota, _ := NewUsageQuota("basic", UsageTypeAPICalls, 10000, ResetPeriodMonthly)

	t.Run("can consume when under limit", func(t *testing.T) {
		assert.True(t, quota.CanConsume(5000, 1000))
	})

	t.Run("can consume up to limit", func(t *testing.T) {
		assert.True(t, quota.CanConsume(9000, 1000))
	})

	t.Run("cannot consume over limit", func(t *testing.T) {
		assert.False(t, quota.CanConsume(9500, 1000))
	})

	t.Run("can always consume when unlimited", func(t *testing.T) {
		unlimitedQuota, _ := NewUsageQuota("enterprise", UsageTypeAPICalls, -1, ResetPeriodMonthly)
		assert.True(t, unlimitedQuota.CanConsume(1000000, 1000000))
	})

	t.Run("can consume when inactive", func(t *testing.T) {
		quota.Deactivate()
		assert.True(t, quota.CanConsume(15000, 1000))
	})
}

func TestUsageQuota_GetFormattedLimit(t *testing.T) {
	t.Run("formats limit", func(t *testing.T) {
		quota, _ := NewUsageQuota("basic", UsageTypeAPICalls, 10000, ResetPeriodMonthly)
		assert.Equal(t, "10000 requests", quota.GetFormattedLimit())
	})

	t.Run("unlimited", func(t *testing.T) {
		quota, _ := NewUsageQuota("enterprise", UsageTypeAPICalls, -1, ResetPeriodMonthly)
		assert.Equal(t, "Unlimited", quota.GetFormattedLimit())
	})

	t.Run("formats bytes", func(t *testing.T) {
		quota, _ := NewUsageQuota("basic", UsageTypeStorageBytes, 1073741824, ResetPeriodNever)
		assert.Equal(t, "1.00 GB", quota.GetFormattedLimit())
	})
}

func TestUsageQuota_GetFormattedSoftLimit(t *testing.T) {
	t.Run("formats soft limit", func(t *testing.T) {
		quota, _ := NewUsageQuota("basic", UsageTypeAPICalls, 10000, ResetPeriodMonthly)
		quota.WithSoftLimit(8000)
		assert.Equal(t, "8000 requests", quota.GetFormattedSoftLimit())
	})

	t.Run("no soft limit", func(t *testing.T) {
		quota, _ := NewUsageQuota("basic", UsageTypeAPICalls, 10000, ResetPeriodMonthly)
		assert.Equal(t, "N/A", quota.GetFormattedSoftLimit())
	})
}

func TestQuotaCheckResult_GetMessage(t *testing.T) {
	tests := []struct {
		status   QuotaStatus
		expected string
	}{
		{QuotaStatusOK, "Usage is within quota"},
		{QuotaStatusWarning, "Usage is approaching quota limit"},
		{QuotaStatusExceeded, "Usage has exceeded quota limit"},
		{QuotaStatusInactive, "Quota is not active"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := QuotaCheckResult{Status: tt.status}
			assert.Equal(t, tt.expected, result.GetMessage())
		})
	}

	t.Run("unlimited message", func(t *testing.T) {
		result := QuotaCheckResult{Status: QuotaStatusOK, IsUnlimited: true}
		assert.Equal(t, "Usage is unlimited", result.GetMessage())
	})
}

func TestOveragePolicy_IsValid(t *testing.T) {
	tests := []struct {
		policy   OveragePolicy
		expected bool
	}{
		{OveragePolicyBlock, true},
		{OveragePolicyWarn, true},
		{OveragePolicyCharge, true},
		{OveragePolicyThrottle, true},
		{OveragePolicy("INVALID"), false},
		{OveragePolicy(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.policy), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.policy.IsValid())
		})
	}
}

func TestQuotaSet(t *testing.T) {
	t.Run("creates and manages quotas", func(t *testing.T) {
		set := NewQuotaSet("basic")

		apiQuota, _ := NewUsageQuota("basic", UsageTypeAPICalls, 10000, ResetPeriodMonthly)
		storageQuota, _ := NewUsageQuota("basic", UsageTypeStorageBytes, 1073741824, ResetPeriodNever)

		set.AddQuota(apiQuota).AddQuota(storageQuota)

		assert.True(t, set.HasQuota(UsageTypeAPICalls))
		assert.True(t, set.HasQuota(UsageTypeStorageBytes))
		assert.False(t, set.HasQuota(UsageTypeActiveUsers))

		assert.Equal(t, apiQuota, set.GetQuota(UsageTypeAPICalls))
	})

	t.Run("checks all usage", func(t *testing.T) {
		set := NewQuotaSet("basic")

		apiQuota, _ := NewUsageQuota("basic", UsageTypeAPICalls, 10000, ResetPeriodMonthly)
		storageQuota, _ := NewUsageQuota("basic", UsageTypeStorageBytes, 1073741824, ResetPeriodNever)

		set.AddQuota(apiQuota).AddQuota(storageQuota)

		usage := map[UsageType]int64{
			UsageTypeAPICalls:     5000,
			UsageTypeStorageBytes: 500000000,
		}

		results := set.CheckAllUsage(usage)

		assert.Len(t, results, 2)
		assert.Equal(t, QuotaStatusOK, results[UsageTypeAPICalls].Status)
		assert.Equal(t, QuotaStatusOK, results[UsageTypeStorageBytes].Status)
	})

	t.Run("gets exceeded quotas", func(t *testing.T) {
		set := NewQuotaSet("basic")

		apiQuota, _ := NewUsageQuota("basic", UsageTypeAPICalls, 10000, ResetPeriodMonthly)
		storageQuota, _ := NewUsageQuota("basic", UsageTypeStorageBytes, 1073741824, ResetPeriodNever)

		set.AddQuota(apiQuota).AddQuota(storageQuota)

		usage := map[UsageType]int64{
			UsageTypeAPICalls:     15000, // Over quota
			UsageTypeStorageBytes: 500000000,
		}

		exceeded := set.GetExceededQuotas(usage)

		assert.Len(t, exceeded, 1)
		assert.Equal(t, UsageTypeAPICalls, exceeded[0].UsageType)
	})

	t.Run("tenant quota set", func(t *testing.T) {
		tenantID := uuid.New()
		set := NewTenantQuotaSet("basic", tenantID)

		assert.Equal(t, "basic", set.PlanID)
		require.NotNil(t, set.TenantID)
		assert.Equal(t, tenantID, *set.TenantID)
	})
}
