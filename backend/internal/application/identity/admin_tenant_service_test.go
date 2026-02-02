package identity

import (
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a test tenant
func createTestTenant(code, name string, plan identity.TenantPlan) *identity.Tenant {
	tenant, _ := identity.NewTenant(code, name)
	tenant.SetPlan(plan)
	tenant.ClearDomainEvents()
	return tenant
}

// Helper function to create test audit context
func createTestAuditContext() AuditContext {
	return AuditContext{
		AdminUserID: uuid.New(),
		IPAddress:   "127.0.0.1",
		UserAgent:   "test-agent",
	}
}

// Tests for ChangePlanInput validation
func TestChangePlanInput_Validation(t *testing.T) {
	t.Run("valid upgrade input", func(t *testing.T) {
		input := ChangePlanInput{
			NewPlan: "pro",
			Reason:  "Customer requested upgrade",
		}
		assert.NotEmpty(t, input.NewPlan)
	})

	t.Run("valid downgrade input", func(t *testing.T) {
		input := ChangePlanInput{
			NewPlan: "basic",
			Reason:  "Cost reduction",
		}
		assert.NotEmpty(t, input.NewPlan)
	})
}

// Tests for UpdateQuotaInput validation
func TestUpdateQuotaInput_Validation(t *testing.T) {
	t.Run("valid quota input with all fields", func(t *testing.T) {
		maxUsers := 100
		maxWarehouses := 50
		maxProducts := 10000

		input := UpdateQuotaInput{
			MaxUsers:      &maxUsers,
			MaxWarehouses: &maxWarehouses,
			MaxProducts:   &maxProducts,
			Reason:        "Custom quota for enterprise customer",
		}

		assert.NotNil(t, input.MaxUsers)
		assert.Equal(t, 100, *input.MaxUsers)
		assert.NotNil(t, input.MaxWarehouses)
		assert.Equal(t, 50, *input.MaxWarehouses)
		assert.NotNil(t, input.MaxProducts)
		assert.Equal(t, 10000, *input.MaxProducts)
	})

	t.Run("valid partial quota input", func(t *testing.T) {
		maxUsers := 50

		input := UpdateQuotaInput{
			MaxUsers: &maxUsers,
		}

		assert.NotNil(t, input.MaxUsers)
		assert.Nil(t, input.MaxWarehouses)
		assert.Nil(t, input.MaxProducts)
	})
}

// Tests for ChangePlanResult
func TestChangePlanResult(t *testing.T) {
	t.Run("immediate upgrade result", func(t *testing.T) {
		result := ChangePlanResult{
			ChangeType:   "immediate",
			EffectiveAt:  time.Now(),
			PreviousPlan: "basic",
			NewPlan:      "pro",
		}

		assert.Equal(t, "immediate", result.ChangeType)
		assert.Equal(t, "basic", result.PreviousPlan)
		assert.Equal(t, "pro", result.NewPlan)
	})

	t.Run("scheduled downgrade result", func(t *testing.T) {
		effectiveAt := time.Now().AddDate(0, 1, 0)
		result := ChangePlanResult{
			ChangeType:   "scheduled",
			EffectiveAt:  effectiveAt,
			PreviousPlan: "pro",
			NewPlan:      "basic",
		}

		assert.Equal(t, "scheduled", result.ChangeType)
		assert.Equal(t, "pro", result.PreviousPlan)
		assert.Equal(t, "basic", result.NewPlan)
		assert.True(t, result.EffectiveAt.After(time.Now()))
	})
}

// Tests for SubscriptionHistoryDTO conversion
func TestSubscriptionHistoryDTO(t *testing.T) {
	t.Run("plan upgrade history", func(t *testing.T) {
		dto := SubscriptionHistoryDTO{
			ChangeType: "plan_upgrade",
			OldPlan:    "basic",
			NewPlan:    "pro",
			Reason:     "Customer requested upgrade",
		}

		assert.Equal(t, "plan_upgrade", dto.ChangeType)
		assert.Equal(t, "basic", dto.OldPlan)
		assert.Equal(t, "pro", dto.NewPlan)
		assert.Nil(t, dto.OldQuota)
		assert.Nil(t, dto.NewQuota)
	})

	t.Run("quota update history", func(t *testing.T) {
		dto := SubscriptionHistoryDTO{
			ChangeType: "quota_update",
			OldQuota: &QuotaDTO{
				MaxUsers:      10,
				MaxWarehouses: 5,
				MaxProducts:   5000,
			},
			NewQuota: &QuotaDTO{
				MaxUsers:      100,
				MaxWarehouses: 50,
				MaxProducts:   50000,
			},
			Reason: "Custom quota for enterprise customer",
		}

		assert.Equal(t, "quota_update", dto.ChangeType)
		assert.NotNil(t, dto.OldQuota)
		assert.NotNil(t, dto.NewQuota)
		assert.Equal(t, 10, dto.OldQuota.MaxUsers)
		assert.Equal(t, 100, dto.NewQuota.MaxUsers)
	})
}

// Tests for toSubscriptionHistoryDTO conversion
func TestToSubscriptionHistoryDTO(t *testing.T) {
	t.Run("converts plan upgrade history", func(t *testing.T) {
		history, err := identity.NewSubscriptionHistory(
			testUUIDFromBytes(),
			testUUIDFromBytes(),
			identity.SubscriptionChangeTypePlanUpgrade,
		).
			WithPlanChange(identity.TenantPlanBasic, identity.TenantPlanPro).
			WithReason("Customer requested upgrade").
			Build()

		require.NoError(t, err)

		dto := toSubscriptionHistoryDTO(history)

		assert.Equal(t, "plan_upgrade", dto.ChangeType)
		assert.Equal(t, "basic", dto.OldPlan)
		assert.Equal(t, "pro", dto.NewPlan)
		assert.Equal(t, "Customer requested upgrade", dto.Reason)
		assert.Nil(t, dto.OldQuota)
		assert.Nil(t, dto.NewQuota)
	})

	t.Run("converts quota update history", func(t *testing.T) {
		oldQuota := identity.TenantQuota{MaxUsers: 10, MaxWarehouses: 5, MaxProducts: 5000}
		newQuota := identity.TenantQuota{MaxUsers: 100, MaxWarehouses: 50, MaxProducts: 50000}

		history, err := identity.NewSubscriptionHistory(
			testUUIDFromBytes(),
			testUUIDFromBytes(),
			identity.SubscriptionChangeTypeQuotaUpdate,
		).
			WithQuotaChange(oldQuota, newQuota).
			WithReason("Custom quota").
			Build()

		require.NoError(t, err)

		dto := toSubscriptionHistoryDTO(history)

		assert.Equal(t, "quota_update", dto.ChangeType)
		assert.NotNil(t, dto.OldQuota)
		assert.NotNil(t, dto.NewQuota)
		assert.Equal(t, 10, dto.OldQuota.MaxUsers)
		assert.Equal(t, 100, dto.NewQuota.MaxUsers)
		assert.Equal(t, 5, dto.OldQuota.MaxWarehouses)
		assert.Equal(t, 50, dto.NewQuota.MaxWarehouses)
		assert.Equal(t, 5000, dto.OldQuota.MaxProducts)
		assert.Equal(t, 50000, dto.NewQuota.MaxProducts)
	})

	t.Run("converts scheduled downgrade history", func(t *testing.T) {
		effectiveAt := time.Now().AddDate(0, 1, 0)
		scheduledAt := time.Now()

		history, err := identity.NewSubscriptionHistory(
			testUUIDFromBytes(),
			testUUIDFromBytes(),
			identity.SubscriptionChangeTypePlanDowngrade,
		).
			WithPlanChange(identity.TenantPlanPro, identity.TenantPlanBasic).
			WithEffectiveAt(effectiveAt).
			WithScheduledAt(scheduledAt).
			Build()

		require.NoError(t, err)

		dto := toSubscriptionHistoryDTO(history)

		assert.Equal(t, "plan_downgrade", dto.ChangeType)
		assert.Equal(t, "pro", dto.OldPlan)
		assert.Equal(t, "basic", dto.NewPlan)
		assert.NotNil(t, dto.ScheduledAt)
	})
}

// Tests for toAdminTenantDTO with scheduled plan
func TestToAdminTenantDTO_WithScheduledPlan(t *testing.T) {
	t.Run("includes scheduled plan info", func(t *testing.T) {
		tenant := createTestTenant("TENANT001", "Test Company", identity.TenantPlanPro)
		effectiveAt := time.Now().AddDate(0, 1, 0)
		tenant.SchedulePlanDowngrade(identity.TenantPlanBasic, effectiveAt)

		dto := toAdminTenantDTO(tenant, nil)

		assert.Equal(t, "pro", dto.Plan)
		assert.NotNil(t, dto.ScheduledPlan)
		assert.Equal(t, "basic", *dto.ScheduledPlan)
		assert.NotNil(t, dto.ScheduledPlanEffectiveAt)
	})

	t.Run("no scheduled plan when not set", func(t *testing.T) {
		tenant := createTestTenant("TENANT001", "Test Company", identity.TenantPlanPro)

		dto := toAdminTenantDTO(tenant, nil)

		assert.Equal(t, "pro", dto.Plan)
		assert.Nil(t, dto.ScheduledPlan)
		assert.Nil(t, dto.ScheduledPlanEffectiveAt)
	})
}

// Tests for SubscriptionHistoryFilterInput
func TestSubscriptionHistoryFilterInput(t *testing.T) {
	t.Run("default filter values", func(t *testing.T) {
		input := SubscriptionHistoryFilterInput{
			Page:     1,
			PageSize: 10,
		}

		assert.Equal(t, 1, input.Page)
		assert.Equal(t, 10, input.PageSize)
		assert.Nil(t, input.ChangeType)
		assert.Nil(t, input.StartDate)
		assert.Nil(t, input.EndDate)
	})

	t.Run("filter with change type", func(t *testing.T) {
		changeType := "plan_upgrade"
		input := SubscriptionHistoryFilterInput{
			Page:       1,
			PageSize:   10,
			ChangeType: &changeType,
		}

		assert.NotNil(t, input.ChangeType)
		assert.Equal(t, "plan_upgrade", *input.ChangeType)
	})

	t.Run("filter with date range", func(t *testing.T) {
		startDate := time.Now().AddDate(0, -1, 0)
		endDate := time.Now()
		input := SubscriptionHistoryFilterInput{
			Page:      1,
			PageSize:  10,
			StartDate: &startDate,
			EndDate:   &endDate,
		}

		assert.NotNil(t, input.StartDate)
		assert.NotNil(t, input.EndDate)
		assert.True(t, input.EndDate.After(*input.StartDate))
	})
}

// Tests for SubscriptionHistoryListResult
func TestSubscriptionHistoryListResult(t *testing.T) {
	t.Run("empty result", func(t *testing.T) {
		result := SubscriptionHistoryListResult{
			History:    []SubscriptionHistoryDTO{},
			Total:      0,
			Page:       1,
			PageSize:   10,
			TotalPages: 0,
		}

		assert.Empty(t, result.History)
		assert.Equal(t, int64(0), result.Total)
		assert.Equal(t, 0, result.TotalPages)
	})

	t.Run("result with pagination", func(t *testing.T) {
		result := SubscriptionHistoryListResult{
			History: []SubscriptionHistoryDTO{
				{ChangeType: "plan_upgrade"},
				{ChangeType: "quota_update"},
			},
			Total:      25,
			Page:       1,
			PageSize:   10,
			TotalPages: 3,
		}

		assert.Len(t, result.History, 2)
		assert.Equal(t, int64(25), result.Total)
		assert.Equal(t, 3, result.TotalPages)
	})
}

// Helper to create UUID from bytes for testing
func testUUIDFromBytes() uuid.UUID {
	return uuid.MustParse("01020304-0506-0708-090a-0b0c0d0e0f10")
}

// ============================================================================
// Tests for Tenant Status Management (P3-ADMIN-007)
// ============================================================================

// Tests for SuspendTenantInput validation
func TestSuspendTenantInput_Validation(t *testing.T) {
	t.Run("valid suspend input with reason", func(t *testing.T) {
		input := SuspendTenantInput{
			TenantID: uuid.New(),
			Reason:   "Payment overdue",
		}
		assert.NotEmpty(t, input.Reason)
		assert.NotEqual(t, uuid.Nil, input.TenantID)
	})

	t.Run("valid suspend input with scheduled reactivation", func(t *testing.T) {
		reactivateAt := time.Now().Add(24 * time.Hour)
		input := SuspendTenantInput{
			TenantID:              uuid.New(),
			Reason:                "Temporary suspension",
			ScheduledReactivateAt: &reactivateAt,
		}
		assert.NotNil(t, input.ScheduledReactivateAt)
		assert.True(t, input.ScheduledReactivateAt.After(time.Now()))
	})

	t.Run("valid suspend input without reason", func(t *testing.T) {
		input := SuspendTenantInput{
			TenantID: uuid.New(),
			Reason:   "",
		}
		assert.Empty(t, input.Reason)
	})
}

// Tests for TenantStatusDTO
func TestTenantStatusDTO(t *testing.T) {
	t.Run("active tenant status", func(t *testing.T) {
		dto := TenantStatusDTO{
			TenantID: uuid.New(),
			Status:   "active",
		}
		assert.Equal(t, "active", dto.Status)
		assert.Empty(t, dto.SuspensionReason)
		assert.Nil(t, dto.SuspendedAt)
		assert.Nil(t, dto.ScheduledReactivateAt)
	})

	t.Run("suspended tenant status with reason", func(t *testing.T) {
		suspendedAt := time.Now()
		dto := TenantStatusDTO{
			TenantID:         uuid.New(),
			Status:           "suspended",
			SuspensionReason: "Payment overdue",
			SuspendedAt:      &suspendedAt,
		}
		assert.Equal(t, "suspended", dto.Status)
		assert.Equal(t, "Payment overdue", dto.SuspensionReason)
		assert.NotNil(t, dto.SuspendedAt)
	})

	t.Run("suspended tenant with scheduled reactivation", func(t *testing.T) {
		suspendedAt := time.Now()
		reactivateAt := time.Now().Add(24 * time.Hour)
		dto := TenantStatusDTO{
			TenantID:              uuid.New(),
			Status:                "suspended",
			SuspensionReason:      "Temporary suspension",
			SuspendedAt:           &suspendedAt,
			ScheduledReactivateAt: &reactivateAt,
		}
		assert.Equal(t, "suspended", dto.Status)
		assert.NotNil(t, dto.ScheduledReactivateAt)
		assert.True(t, dto.ScheduledReactivateAt.After(*dto.SuspendedAt))
	})
}

// Tests for TenantStatusHistoryDTO
func TestTenantStatusHistoryDTO(t *testing.T) {
	t.Run("suspension history entry", func(t *testing.T) {
		dto := TenantStatusHistoryDTO{
			ID:              uuid.New(),
			ChangeType:      "suspend",
			OldStatus:       "active",
			NewStatus:       "suspended",
			Reason:          "Payment overdue",
			ChangedByUserID: uuid.New(),
			CreatedAt:       time.Now(),
		}
		assert.Equal(t, "suspend", dto.ChangeType)
		assert.Equal(t, "active", dto.OldStatus)
		assert.Equal(t, "suspended", dto.NewStatus)
		assert.Equal(t, "Payment overdue", dto.Reason)
	})

	t.Run("activation history entry", func(t *testing.T) {
		dto := TenantStatusHistoryDTO{
			ID:              uuid.New(),
			ChangeType:      "activate",
			OldStatus:       "suspended",
			NewStatus:       "active",
			ChangedByUserID: uuid.New(),
			CreatedAt:       time.Now(),
		}
		assert.Equal(t, "activate", dto.ChangeType)
		assert.Equal(t, "suspended", dto.OldStatus)
		assert.Equal(t, "active", dto.NewStatus)
	})

	t.Run("suspension with scheduled reactivation", func(t *testing.T) {
		reactivateAt := time.Now().Add(24 * time.Hour)
		dto := TenantStatusHistoryDTO{
			ID:                    uuid.New(),
			ChangeType:            "suspend",
			OldStatus:             "active",
			NewStatus:             "suspended",
			Reason:                "Temporary suspension",
			ScheduledReactivateAt: &reactivateAt,
			ChangedByUserID:       uuid.New(),
			CreatedAt:             time.Now(),
		}
		assert.NotNil(t, dto.ScheduledReactivateAt)
		assert.True(t, dto.ScheduledReactivateAt.After(time.Now()))
	})
}

// Tests for TenantStatusHistoryFilterInput
func TestTenantStatusHistoryFilterInput(t *testing.T) {
	t.Run("default filter values", func(t *testing.T) {
		input := TenantStatusHistoryFilterInput{
			Page:     1,
			PageSize: 10,
		}
		assert.Equal(t, 1, input.Page)
		assert.Equal(t, 10, input.PageSize)
		assert.Nil(t, input.ChangeType)
	})

	t.Run("filter with change type", func(t *testing.T) {
		changeType := "suspend"
		input := TenantStatusHistoryFilterInput{
			Page:       1,
			PageSize:   10,
			ChangeType: &changeType,
		}
		assert.NotNil(t, input.ChangeType)
		assert.Equal(t, "suspend", *input.ChangeType)
	})

	t.Run("filter with date range", func(t *testing.T) {
		startDate := time.Now().AddDate(0, -1, 0)
		endDate := time.Now()
		input := TenantStatusHistoryFilterInput{
			Page:      1,
			PageSize:  10,
			StartDate: &startDate,
			EndDate:   &endDate,
		}
		assert.NotNil(t, input.StartDate)
		assert.NotNil(t, input.EndDate)
		assert.True(t, input.EndDate.After(*input.StartDate))
	})
}

// Tests for TenantStatusHistoryListResult
func TestTenantStatusHistoryListResult(t *testing.T) {
	t.Run("empty result", func(t *testing.T) {
		result := TenantStatusHistoryListResult{
			History:    []TenantStatusHistoryDTO{},
			Total:      0,
			Page:       1,
			PageSize:   10,
			TotalPages: 0,
		}
		assert.Empty(t, result.History)
		assert.Equal(t, int64(0), result.Total)
		assert.Equal(t, 0, result.TotalPages)
	})

	t.Run("result with pagination", func(t *testing.T) {
		result := TenantStatusHistoryListResult{
			History: []TenantStatusHistoryDTO{
				{ChangeType: "suspend"},
				{ChangeType: "activate"},
			},
			Total:      25,
			Page:       1,
			PageSize:   10,
			TotalPages: 3,
		}
		assert.Len(t, result.History, 2)
		assert.Equal(t, int64(25), result.Total)
		assert.Equal(t, 3, result.TotalPages)
	})
}

// Tests for toTenantStatusHistoryDTO conversion
func TestToTenantStatusHistoryDTO(t *testing.T) {
	t.Run("converts suspension history", func(t *testing.T) {
		reactivateAt := time.Now().Add(24 * time.Hour)
		history, err := identity.NewTenantStatusHistory(
			testUUIDFromBytes(),
			testUUIDFromBytes(),
			identity.TenantStatusChangeTypeSuspend,
		).
			WithStatusChange(identity.TenantStatusActive, identity.TenantStatusSuspended).
			WithReason("Payment overdue").
			WithScheduledReactivateAt(&reactivateAt).
			WithIPAddress("127.0.0.1").
			WithUserAgent("test-agent").
			Build()

		require.NoError(t, err)

		dto := toTenantStatusHistoryDTO(history)

		assert.Equal(t, "suspend", dto.ChangeType)
		assert.Equal(t, "active", dto.OldStatus)
		assert.Equal(t, "suspended", dto.NewStatus)
		assert.Equal(t, "Payment overdue", dto.Reason)
		assert.NotNil(t, dto.ScheduledReactivateAt)
	})

	t.Run("converts activation history", func(t *testing.T) {
		history, err := identity.NewTenantStatusHistory(
			testUUIDFromBytes(),
			testUUIDFromBytes(),
			identity.TenantStatusChangeTypeActivate,
		).
			WithStatusChange(identity.TenantStatusSuspended, identity.TenantStatusActive).
			WithIPAddress("127.0.0.1").
			WithUserAgent("test-agent").
			Build()

		require.NoError(t, err)

		dto := toTenantStatusHistoryDTO(history)

		assert.Equal(t, "activate", dto.ChangeType)
		assert.Equal(t, "suspended", dto.OldStatus)
		assert.Equal(t, "active", dto.NewStatus)
		assert.Empty(t, dto.Reason)
		assert.Nil(t, dto.ScheduledReactivateAt)
	})
}

// Tests for toAdminTenantDTO with suspension fields
func TestToAdminTenantDTO_WithSuspensionFields(t *testing.T) {
	t.Run("includes suspension info for suspended tenant", func(t *testing.T) {
		tenant := createTestTenant("TENANT001", "Test Company", identity.TenantPlanPro)
		reactivateAt := time.Now().Add(24 * time.Hour)
		tenant.SuspendWithReason("Payment overdue", &reactivateAt)

		dto := toAdminTenantDTO(tenant, nil)

		assert.Equal(t, "suspended", dto.Status)
		assert.Equal(t, "Payment overdue", dto.SuspensionReason)
		assert.NotNil(t, dto.SuspendedAt)
		assert.NotNil(t, dto.ScheduledReactivateAt)
	})

	t.Run("no suspension info for active tenant", func(t *testing.T) {
		tenant := createTestTenant("TENANT001", "Test Company", identity.TenantPlanPro)

		dto := toAdminTenantDTO(tenant, nil)

		assert.Equal(t, "active", dto.Status)
		assert.Empty(t, dto.SuspensionReason)
		assert.Nil(t, dto.SuspendedAt)
		assert.Nil(t, dto.ScheduledReactivateAt)
	})

	t.Run("clears suspension info after activation", func(t *testing.T) {
		tenant := createTestTenant("TENANT001", "Test Company", identity.TenantPlanPro)
		reactivateAt := time.Now().Add(24 * time.Hour)
		tenant.SuspendWithReason("Payment overdue", &reactivateAt)
		tenant.Activate()

		dto := toAdminTenantDTO(tenant, nil)

		assert.Equal(t, "active", dto.Status)
		assert.Empty(t, dto.SuspensionReason)
		assert.Nil(t, dto.SuspendedAt)
		assert.Nil(t, dto.ScheduledReactivateAt)
	})
}
