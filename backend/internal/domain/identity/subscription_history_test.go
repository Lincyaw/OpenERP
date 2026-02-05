package identity

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionChangeType_IsValid(t *testing.T) {
	tests := []struct {
		name       string
		changeType SubscriptionChangeType
		want       bool
	}{
		{"plan_upgrade is valid", SubscriptionChangeTypePlanUpgrade, true},
		{"plan_downgrade is valid", SubscriptionChangeTypePlanDowngrade, true},
		{"quota_update is valid", SubscriptionChangeTypeQuotaUpdate, true},
		{"empty string is invalid", SubscriptionChangeType(""), false},
		{"unknown type is invalid", SubscriptionChangeType("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.changeType.IsValid())
		})
	}
}

func TestSubscriptionChangeType_String(t *testing.T) {
	assert.Equal(t, "plan_upgrade", SubscriptionChangeTypePlanUpgrade.String())
	assert.Equal(t, "plan_downgrade", SubscriptionChangeTypePlanDowngrade.String())
	assert.Equal(t, "quota_update", SubscriptionChangeTypeQuotaUpdate.String())
}

func TestNewTenantQuota(t *testing.T) {
	config := TenantConfig{
		MaxUsers:      10,
		MaxWarehouses: 5,
		MaxProducts:   1000,
	}

	quota := NewTenantQuota(config)

	assert.Equal(t, 10, quota.MaxUsers)
	assert.Equal(t, 5, quota.MaxWarehouses)
	assert.Equal(t, 1000, quota.MaxProducts)
}

func TestTenantQuota_Equals(t *testing.T) {
	t.Run("equal quotas", func(t *testing.T) {
		q1 := TenantQuota{MaxUsers: 10, MaxWarehouses: 5, MaxProducts: 1000}
		q2 := TenantQuota{MaxUsers: 10, MaxWarehouses: 5, MaxProducts: 1000}

		assert.True(t, q1.Equals(q2))
	})

	t.Run("different max users", func(t *testing.T) {
		q1 := TenantQuota{MaxUsers: 10, MaxWarehouses: 5, MaxProducts: 1000}
		q2 := TenantQuota{MaxUsers: 20, MaxWarehouses: 5, MaxProducts: 1000}

		assert.False(t, q1.Equals(q2))
	})

	t.Run("different max warehouses", func(t *testing.T) {
		q1 := TenantQuota{MaxUsers: 10, MaxWarehouses: 5, MaxProducts: 1000}
		q2 := TenantQuota{MaxUsers: 10, MaxWarehouses: 10, MaxProducts: 1000}

		assert.False(t, q1.Equals(q2))
	})

	t.Run("different max products", func(t *testing.T) {
		q1 := TenantQuota{MaxUsers: 10, MaxWarehouses: 5, MaxProducts: 1000}
		q2 := TenantQuota{MaxUsers: 10, MaxWarehouses: 5, MaxProducts: 2000}

		assert.False(t, q1.Equals(q2))
	})
}

func TestNewSubscriptionHistory_PlanUpgrade(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	history, err := NewSubscriptionHistory(tenantID, userID, SubscriptionChangeTypePlanUpgrade).
		WithPlanChange(TenantPlanBasic, TenantPlanPro).
		WithReason("Customer requested upgrade").
		Build()

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, history.ID)
	assert.Equal(t, tenantID, history.TenantID)
	assert.Equal(t, userID, history.ChangedByUserID)
	assert.Equal(t, SubscriptionChangeTypePlanUpgrade, history.ChangeType)
	assert.Equal(t, TenantPlanBasic, history.OldPlan)
	assert.Equal(t, TenantPlanPro, history.NewPlan)
	assert.Equal(t, "Customer requested upgrade", history.Reason)
	assert.Nil(t, history.OldQuota)
	assert.Nil(t, history.NewQuota)
	assert.Nil(t, history.ScheduledAt)
}

func TestNewSubscriptionHistory_PlanDowngrade(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	effectiveAt := time.Now().AddDate(0, 1, 0) // 1 month from now
	scheduledAt := time.Now()

	history, err := NewSubscriptionHistory(tenantID, userID, SubscriptionChangeTypePlanDowngrade).
		WithPlanChange(TenantPlanPro, TenantPlanBasic).
		WithEffectiveAt(effectiveAt).
		WithScheduledAt(scheduledAt).
		WithReason("Cost reduction").
		Build()

	require.NoError(t, err)
	assert.Equal(t, SubscriptionChangeTypePlanDowngrade, history.ChangeType)
	assert.Equal(t, TenantPlanPro, history.OldPlan)
	assert.Equal(t, TenantPlanBasic, history.NewPlan)
	assert.Equal(t, effectiveAt.Unix(), history.EffectiveAt.Unix())
	assert.NotNil(t, history.ScheduledAt)
	assert.Equal(t, scheduledAt.Unix(), history.ScheduledAt.Unix())
}

func TestNewSubscriptionHistory_QuotaUpdate(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	oldQuota := TenantQuota{MaxUsers: 10, MaxWarehouses: 5, MaxProducts: 1000}
	newQuota := TenantQuota{MaxUsers: 20, MaxWarehouses: 10, MaxProducts: 5000}

	history, err := NewSubscriptionHistory(tenantID, userID, SubscriptionChangeTypeQuotaUpdate).
		WithQuotaChange(oldQuota, newQuota).
		WithReason("Increased capacity").
		Build()

	require.NoError(t, err)
	assert.Equal(t, SubscriptionChangeTypeQuotaUpdate, history.ChangeType)
	assert.NotNil(t, history.OldQuota)
	assert.NotNil(t, history.NewQuota)
	assert.Equal(t, oldQuota, *history.OldQuota)
	assert.Equal(t, newQuota, *history.NewQuota)
}

func TestNewSubscriptionHistory_ValidationErrors(t *testing.T) {
	validTenantID := uuid.New()
	validUserID := uuid.New()

	t.Run("empty tenant ID", func(t *testing.T) {
		_, err := NewSubscriptionHistory(uuid.Nil, validUserID, SubscriptionChangeTypePlanUpgrade).
			WithPlanChange(TenantPlanBasic, TenantPlanPro).
			Build()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Tenant ID cannot be empty")
	})

	t.Run("empty changed by user ID", func(t *testing.T) {
		_, err := NewSubscriptionHistory(validTenantID, uuid.Nil, SubscriptionChangeTypePlanUpgrade).
			WithPlanChange(TenantPlanBasic, TenantPlanPro).
			Build()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Changed by user ID cannot be empty")
	})

	t.Run("invalid change type", func(t *testing.T) {
		_, err := NewSubscriptionHistory(validTenantID, validUserID, SubscriptionChangeType("invalid")).
			Build()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid subscription change type")
	})

	t.Run("plan upgrade without plan change", func(t *testing.T) {
		_, err := NewSubscriptionHistory(validTenantID, validUserID, SubscriptionChangeTypePlanUpgrade).
			Build()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Plan change requires both old and new plan")
	})

	t.Run("plan change with same plans", func(t *testing.T) {
		_, err := NewSubscriptionHistory(validTenantID, validUserID, SubscriptionChangeTypePlanUpgrade).
			WithPlanChange(TenantPlanBasic, TenantPlanBasic).
			Build()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Old and new plan cannot be the same")
	})

	t.Run("quota update without quota change", func(t *testing.T) {
		_, err := NewSubscriptionHistory(validTenantID, validUserID, SubscriptionChangeTypeQuotaUpdate).
			Build()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Quota change requires both old and new quota")
	})
}

func TestSubscriptionHistory_IsScheduled(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("scheduled change in future", func(t *testing.T) {
		futureTime := time.Now().Add(24 * time.Hour)
		scheduledAt := time.Now()

		history, _ := NewSubscriptionHistory(tenantID, userID, SubscriptionChangeTypePlanDowngrade).
			WithPlanChange(TenantPlanPro, TenantPlanBasic).
			WithEffectiveAt(futureTime).
			WithScheduledAt(scheduledAt).
			Build()

		assert.True(t, history.IsScheduled())
	})

	t.Run("scheduled change in past", func(t *testing.T) {
		pastTime := time.Now().Add(-24 * time.Hour)
		scheduledAt := time.Now().Add(-48 * time.Hour)

		history, _ := NewSubscriptionHistory(tenantID, userID, SubscriptionChangeTypePlanDowngrade).
			WithPlanChange(TenantPlanPro, TenantPlanBasic).
			WithEffectiveAt(pastTime).
			WithScheduledAt(scheduledAt).
			Build()

		assert.False(t, history.IsScheduled())
	})

	t.Run("immediate change without scheduled at", func(t *testing.T) {
		history, _ := NewSubscriptionHistory(tenantID, userID, SubscriptionChangeTypePlanUpgrade).
			WithPlanChange(TenantPlanBasic, TenantPlanPro).
			Build()

		assert.False(t, history.IsScheduled())
	})
}

func TestSubscriptionHistory_IsPlanChange(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("plan upgrade is plan change", func(t *testing.T) {
		history, _ := NewSubscriptionHistory(tenantID, userID, SubscriptionChangeTypePlanUpgrade).
			WithPlanChange(TenantPlanBasic, TenantPlanPro).
			Build()

		assert.True(t, history.IsPlanChange())
	})

	t.Run("plan downgrade is plan change", func(t *testing.T) {
		history, _ := NewSubscriptionHistory(tenantID, userID, SubscriptionChangeTypePlanDowngrade).
			WithPlanChange(TenantPlanPro, TenantPlanBasic).
			Build()

		assert.True(t, history.IsPlanChange())
	})

	t.Run("quota update is not plan change", func(t *testing.T) {
		oldQuota := TenantQuota{MaxUsers: 10, MaxWarehouses: 5, MaxProducts: 1000}
		newQuota := TenantQuota{MaxUsers: 20, MaxWarehouses: 10, MaxProducts: 5000}

		history, _ := NewSubscriptionHistory(tenantID, userID, SubscriptionChangeTypeQuotaUpdate).
			WithQuotaChange(oldQuota, newQuota).
			Build()

		assert.False(t, history.IsPlanChange())
	})
}

func TestSubscriptionHistory_IsQuotaChange(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("quota update is quota change", func(t *testing.T) {
		oldQuota := TenantQuota{MaxUsers: 10, MaxWarehouses: 5, MaxProducts: 1000}
		newQuota := TenantQuota{MaxUsers: 20, MaxWarehouses: 10, MaxProducts: 5000}

		history, _ := NewSubscriptionHistory(tenantID, userID, SubscriptionChangeTypeQuotaUpdate).
			WithQuotaChange(oldQuota, newQuota).
			Build()

		assert.True(t, history.IsQuotaChange())
	})

	t.Run("plan upgrade is not quota change", func(t *testing.T) {
		history, _ := NewSubscriptionHistory(tenantID, userID, SubscriptionChangeTypePlanUpgrade).
			WithPlanChange(TenantPlanBasic, TenantPlanPro).
			Build()

		assert.False(t, history.IsQuotaChange())
	})
}
