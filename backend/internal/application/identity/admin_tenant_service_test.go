package identity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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

// ============================================================================
// Mock Implementations for Service Testing
// ============================================================================

// MockAdminTenantRepository is a mock implementation of AdminTenantRepository
type MockAdminTenantRepository struct {
	mock.Mock
}

func (m *MockAdminTenantRepository) FindAll(ctx context.Context, filter identity.AdminTenantFilter) ([]identity.Tenant, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *MockAdminTenantRepository) FindByID(ctx context.Context, id uuid.UUID) (*identity.Tenant, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.Tenant), args.Error(1)
}

func (m *MockAdminTenantRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]identity.Tenant, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *MockAdminTenantRepository) Count(ctx context.Context, filter identity.AdminTenantFilter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAdminTenantRepository) Search(ctx context.Context, query string, filter identity.AdminTenantFilter) ([]identity.Tenant, error) {
	args := m.Called(ctx, query, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *MockAdminTenantRepository) CountByStatus(ctx context.Context) (map[identity.TenantStatus]int64, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[identity.TenantStatus]int64), args.Error(1)
}

func (m *MockAdminTenantRepository) CountByPlan(ctx context.Context) (map[identity.TenantPlan]int64, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[identity.TenantPlan]int64), args.Error(1)
}

func (m *MockAdminTenantRepository) FindTrialExpiring(ctx context.Context, withinDays int) ([]identity.Tenant, error) {
	args := m.Called(ctx, withinDays)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *MockAdminTenantRepository) FindSubscriptionExpiring(ctx context.Context, withinDays int) ([]identity.Tenant, error) {
	args := m.Called(ctx, withinDays)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *MockAdminTenantRepository) FindExpired(ctx context.Context) ([]identity.Tenant, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *MockAdminTenantRepository) GetStatistics(ctx context.Context) (*identity.TenantStatistics, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.TenantStatistics), args.Error(1)
}

// MockTenantRepository is a mock implementation of TenantRepository
type MockTenantRepository struct {
	mock.Mock
}

func (m *MockTenantRepository) Save(ctx context.Context, tenant *identity.Tenant) error {
	args := m.Called(ctx, tenant)
	return args.Error(0)
}

func (m *MockTenantRepository) ExistsByCode(ctx context.Context, code string) (bool, error) {
	args := m.Called(ctx, code)
	return args.Bool(0), args.Error(1)
}

func (m *MockTenantRepository) FindByID(ctx context.Context, id uuid.UUID) (*identity.Tenant, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.Tenant), args.Error(1)
}

func (m *MockTenantRepository) FindByCode(ctx context.Context, code string) (*identity.Tenant, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.Tenant), args.Error(1)
}

func (m *MockTenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTenantRepository) FindByDomain(ctx context.Context, domain string) (*identity.Tenant, error) {
	args := m.Called(ctx, domain)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.Tenant), args.Error(1)
}

func (m *MockTenantRepository) FindAll(ctx context.Context, filter shared.Filter) ([]identity.Tenant, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *MockTenantRepository) FindByStatus(ctx context.Context, status identity.TenantStatus, filter shared.Filter) ([]identity.Tenant, error) {
	args := m.Called(ctx, status, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *MockTenantRepository) FindByPlan(ctx context.Context, plan identity.TenantPlan, filter shared.Filter) ([]identity.Tenant, error) {
	args := m.Called(ctx, plan, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *MockTenantRepository) FindActive(ctx context.Context, filter shared.Filter) ([]identity.Tenant, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *MockTenantRepository) FindTrialExpiring(ctx context.Context, withinDays int) ([]identity.Tenant, error) {
	args := m.Called(ctx, withinDays)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *MockTenantRepository) FindSubscriptionExpiring(ctx context.Context, withinDays int) ([]identity.Tenant, error) {
	args := m.Called(ctx, withinDays)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *MockTenantRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]identity.Tenant, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *MockTenantRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTenantRepository) CountByStatus(ctx context.Context, status identity.TenantStatus) (int64, error) {
	args := m.Called(ctx, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTenantRepository) CountByPlan(ctx context.Context, plan identity.TenantPlan) (int64, error) {
	args := m.Called(ctx, plan)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTenantRepository) ExistsByDomain(ctx context.Context, domain string) (bool, error) {
	args := m.Called(ctx, domain)
	return args.Bool(0), args.Error(1)
}

func (m *MockTenantRepository) FindByStripeCustomerID(ctx context.Context, customerID string) (*identity.Tenant, error) {
	args := m.Called(ctx, customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.Tenant), args.Error(1)
}

func (m *MockTenantRepository) FindByStripeSubscriptionID(ctx context.Context, subscriptionID string) (*identity.Tenant, error) {
	args := m.Called(ctx, subscriptionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.Tenant), args.Error(1)
}

// MockSubscriptionHistoryRepository is a mock implementation
type MockSubscriptionHistoryRepository struct {
	mock.Mock
}

func (m *MockSubscriptionHistoryRepository) Create(ctx context.Context, history *identity.SubscriptionHistory) error {
	args := m.Called(ctx, history)
	return args.Error(0)
}

func (m *MockSubscriptionHistoryRepository) FindByTenantID(ctx context.Context, tenantID uuid.UUID, filter identity.SubscriptionHistoryFilter) ([]identity.SubscriptionHistory, int64, error) {
	args := m.Called(ctx, tenantID, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]identity.SubscriptionHistory), args.Get(1).(int64), args.Error(2)
}

func (m *MockSubscriptionHistoryRepository) FindByID(ctx context.Context, id uuid.UUID) (*identity.SubscriptionHistory, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.SubscriptionHistory), args.Error(1)
}

func (m *MockSubscriptionHistoryRepository) FindPendingDowngrades(ctx context.Context, beforeTime time.Time) ([]identity.SubscriptionHistory, error) {
	args := m.Called(ctx, beforeTime)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]identity.SubscriptionHistory), args.Error(1)
}

func (m *MockSubscriptionHistoryRepository) FindLatestByTenantID(ctx context.Context, tenantID uuid.UUID) (*identity.SubscriptionHistory, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.SubscriptionHistory), args.Error(1)
}

// MockTenantStatusHistoryRepository is a mock implementation
type MockTenantStatusHistoryRepository struct {
	mock.Mock
}

func (m *MockTenantStatusHistoryRepository) Create(ctx context.Context, history *identity.TenantStatusHistory) error {
	args := m.Called(ctx, history)
	return args.Error(0)
}

func (m *MockTenantStatusHistoryRepository) FindByTenantID(ctx context.Context, tenantID uuid.UUID, filter identity.TenantStatusHistoryFilter) ([]identity.TenantStatusHistory, int64, error) {
	args := m.Called(ctx, tenantID, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]identity.TenantStatusHistory), args.Get(1).(int64), args.Error(2)
}

func (m *MockTenantStatusHistoryRepository) FindLatestByTenantID(ctx context.Context, tenantID uuid.UUID) (*identity.TenantStatusHistory, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.TenantStatusHistory), args.Error(1)
}

func (m *MockTenantStatusHistoryRepository) FindPendingReactivations(ctx context.Context, before time.Time) ([]identity.TenantStatusHistory, error) {
	args := m.Called(ctx, before)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]identity.TenantStatusHistory), args.Error(1)
}

// MockAuditLogRepository is a mock implementation
type MockAuditLogRepository struct {
	mock.Mock
}

func (m *MockAuditLogRepository) Create(ctx context.Context, log *identity.AuditLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockAuditLogRepository) FindByID(ctx context.Context, id uuid.UUID) (*identity.AuditLog, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.AuditLog), args.Error(1)
}

func (m *MockAuditLogRepository) FindAll(ctx context.Context, filter identity.AuditLogFilter) ([]*identity.AuditLog, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*identity.AuditLog), args.Get(1).(int64), args.Error(2)
}

func (m *MockAuditLogRepository) FindByAdminUserID(ctx context.Context, adminUserID uuid.UUID, filter identity.AuditLogFilter) ([]*identity.AuditLog, int64, error) {
	args := m.Called(ctx, adminUserID, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*identity.AuditLog), args.Get(1).(int64), args.Error(2)
}

func (m *MockAuditLogRepository) FindByTargetID(ctx context.Context, targetType identity.AuditTargetType, targetID uuid.UUID, filter identity.AuditLogFilter) ([]*identity.AuditLog, int64, error) {
	args := m.Called(ctx, targetType, targetID, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*identity.AuditLog), args.Get(1).(int64), args.Error(2)
}

func (m *MockAuditLogRepository) Count(ctx context.Context, filter identity.AuditLogFilter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

// newTestLogger creates a noop logger for testing
func newTestLogger() *zap.Logger {
	return zap.NewNop()
}

// newTestAdminTenantService creates an AdminTenantService with mocked dependencies
func newTestAdminTenantService() (*AdminTenantService, *MockAdminTenantRepository, *MockTenantRepository, *MockSubscriptionHistoryRepository, *MockTenantStatusHistoryRepository, *MockAuditLogRepository) {
	adminRepo := new(MockAdminTenantRepository)
	tenantRepo := new(MockTenantRepository)
	subHistoryRepo := new(MockSubscriptionHistoryRepository)
	statusHistoryRepo := new(MockTenantStatusHistoryRepository)
	auditLogRepo := new(MockAuditLogRepository)
	logger := newTestLogger()

	auditService := NewAuditService(auditLogRepo, logger)

	service := NewAdminTenantService(
		adminRepo,
		tenantRepo,
		subHistoryRepo,
		statusHistoryRepo,
		auditService,
		logger,
	)

	return service, adminRepo, tenantRepo, subHistoryRepo, statusHistoryRepo, auditLogRepo
}

// ============================================================================
// AdminTenantService Unit Tests with Mocks
// ============================================================================

func TestNewAdminTenantService(t *testing.T) {
	t.Run("creates service with all dependencies", func(t *testing.T) {
		service, _, _, _, _, _ := newTestAdminTenantService()
		assert.NotNil(t, service)
	})
}

func TestAdminTenantService_ListTenants(t *testing.T) {
	t.Run("returns paginated tenant list", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()

		tenant1 := createTestTenant("TENANT1", "Tenant One", identity.TenantPlanBasic)
		tenant2 := createTestTenant("TENANT2", "Tenant Two", identity.TenantPlanPro)

		adminRepo.On("FindAll", ctx, mock.AnythingOfType("identity.AdminTenantFilter")).
			Return([]identity.Tenant{*tenant1, *tenant2}, nil)
		adminRepo.On("Count", ctx, mock.AnythingOfType("identity.AdminTenantFilter")).
			Return(int64(2), nil)

		input := AdminTenantFilterInput{Page: 1, PageSize: 20}
		result, err := service.ListTenants(ctx, input)

		require.NoError(t, err)
		assert.Len(t, result.Tenants, 2)
		assert.Equal(t, int64(2), result.Total)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 20, result.PageSize)
		adminRepo.AssertExpectations(t)
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()

		adminRepo.On("FindAll", ctx, mock.AnythingOfType("identity.AdminTenantFilter")).
			Return(nil, errors.New("db error"))

		input := AdminTenantFilterInput{Page: 1, PageSize: 20}
		result, err := service.ListTenants(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, result)
		adminRepo.AssertExpectations(t)
	})

	t.Run("applies filters correctly", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()

		tenant1 := createTestTenant("ACTIVE1", "Active Tenant", identity.TenantPlanPro)

		adminRepo.On("FindAll", ctx, mock.MatchedBy(func(f identity.AdminTenantFilter) bool {
			return f.Status != nil && *f.Status == identity.TenantStatusActive
		})).Return([]identity.Tenant{*tenant1}, nil)
		adminRepo.On("Count", ctx, mock.AnythingOfType("identity.AdminTenantFilter")).
			Return(int64(1), nil)

		status := "active"
		input := AdminTenantFilterInput{Page: 1, PageSize: 20, Status: &status}
		result, err := service.ListTenants(ctx, input)

		require.NoError(t, err)
		assert.Len(t, result.Tenants, 1)
		adminRepo.AssertExpectations(t)
	})

	t.Run("limits page size to 100", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()

		adminRepo.On("FindAll", ctx, mock.MatchedBy(func(f identity.AdminTenantFilter) bool {
			return f.PageSize == 100
		})).Return([]identity.Tenant{}, nil)
		adminRepo.On("Count", ctx, mock.AnythingOfType("identity.AdminTenantFilter")).
			Return(int64(0), nil)

		input := AdminTenantFilterInput{Page: 1, PageSize: 500}
		_, err := service.ListTenants(ctx, input)

		require.NoError(t, err)
		adminRepo.AssertExpectations(t)
	})
}

func TestAdminTenantService_GetTenant(t *testing.T) {
	t.Run("returns tenant by ID", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()

		tenant := createTestTenant("TEST001", "Test Tenant", identity.TenantPlanPro)

		adminRepo.On("FindByID", ctx, tenant.ID).Return(tenant, nil)

		result, err := service.GetTenant(ctx, tenant.ID)

		require.NoError(t, err)
		assert.Equal(t, tenant.ID, result.ID)
		assert.Equal(t, "TEST001", result.Code)
		assert.Equal(t, "Test Tenant", result.Name)
		adminRepo.AssertExpectations(t)
	})

	t.Run("returns error when tenant not found", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()
		tenantID := uuid.New()

		adminRepo.On("FindByID", ctx, tenantID).Return(nil, shared.ErrNotFound)

		result, err := service.GetTenant(ctx, tenantID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "Tenant not found")
		adminRepo.AssertExpectations(t)
	})
}

func TestAdminTenantService_CreateTenant(t *testing.T) {
	t.Run("creates new tenant successfully", func(t *testing.T) {
		service, adminRepo, tenantRepo, _, _, auditLogRepo := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		tenantRepo.On("ExistsByCode", ctx, "NEWTENANT").Return(false, nil)
		adminRepo.On("FindAll", ctx, mock.AnythingOfType("identity.AdminTenantFilter")).
			Return([]identity.Tenant{}, nil)
		tenantRepo.On("Save", ctx, mock.AnythingOfType("*identity.Tenant")).Return(nil)
		auditLogRepo.On("Create", ctx, mock.AnythingOfType("*identity.AuditLog")).Return(nil)

		input := AdminCreateTenantInput{
			Code:         "NEWTENANT",
			Name:         "New Tenant",
			ContactName:  "John Doe",
			ContactEmail: "john@example.com",
			Plan:         "pro",
		}

		result, err := service.CreateTenant(ctx, input, auditCtx)

		require.NoError(t, err)
		assert.Equal(t, "NEWTENANT", result.Code)
		assert.Equal(t, "New Tenant", result.Name)
		assert.Equal(t, "pro", result.Plan)
		tenantRepo.AssertExpectations(t)
	})

	t.Run("fails when code already exists", func(t *testing.T) {
		service, _, tenantRepo, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		tenantRepo.On("ExistsByCode", ctx, "EXISTING").Return(true, nil)

		input := AdminCreateTenantInput{
			Code: "EXISTING",
			Name: "Existing Tenant",
		}

		result, err := service.CreateTenant(ctx, input, auditCtx)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "already exists")
		tenantRepo.AssertExpectations(t)
	})

	t.Run("fails with invalid email", func(t *testing.T) {
		service, _, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		input := AdminCreateTenantInput{
			Code:         "TEST001",
			Name:         "Test Tenant",
			ContactEmail: "invalid-email",
		}

		result, err := service.CreateTenant(ctx, input, auditCtx)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "email")
	})

	t.Run("fails with empty code", func(t *testing.T) {
		service, _, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		input := AdminCreateTenantInput{
			Code: "",
			Name: "Test Tenant",
		}

		result, err := service.CreateTenant(ctx, input, auditCtx)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "code is required")
	})

	t.Run("fails with empty name", func(t *testing.T) {
		service, _, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		input := AdminCreateTenantInput{
			Code: "TEST001",
			Name: "",
		}

		result, err := service.CreateTenant(ctx, input, auditCtx)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "name")
	})

	t.Run("creates trial tenant when trial_days specified", func(t *testing.T) {
		service, adminRepo, tenantRepo, _, _, auditLogRepo := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		tenantRepo.On("ExistsByCode", ctx, "TRIALTENANT").Return(false, nil)
		adminRepo.On("FindAll", ctx, mock.AnythingOfType("identity.AdminTenantFilter")).
			Return([]identity.Tenant{}, nil)
		tenantRepo.On("Save", ctx, mock.MatchedBy(func(t *identity.Tenant) bool {
			return t.Status == identity.TenantStatusTrial
		})).Return(nil)
		auditLogRepo.On("Create", ctx, mock.AnythingOfType("*identity.AuditLog")).Return(nil)

		input := AdminCreateTenantInput{
			Code:      "TRIALTENANT",
			Name:      "Trial Tenant",
			TrialDays: 14,
		}

		result, err := service.CreateTenant(ctx, input, auditCtx)

		require.NoError(t, err)
		assert.Equal(t, "trial", result.Status)
		tenantRepo.AssertExpectations(t)
	})
}

func TestAdminTenantService_UpdateTenant(t *testing.T) {
	t.Run("updates tenant successfully", func(t *testing.T) {
		service, adminRepo, tenantRepo, _, _, auditLogRepo := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		tenant := createTestTenant("TEST001", "Old Name", identity.TenantPlanBasic)

		adminRepo.On("FindByID", ctx, tenant.ID).Return(tenant, nil)
		adminRepo.On("FindAll", ctx, mock.AnythingOfType("identity.AdminTenantFilter")).
			Return([]identity.Tenant{}, nil)
		tenantRepo.On("Save", ctx, mock.AnythingOfType("*identity.Tenant")).Return(nil)
		auditLogRepo.On("Create", ctx, mock.AnythingOfType("*identity.AuditLog")).Return(nil)

		newName := "New Name"
		input := AdminUpdateTenantInput{
			Name: &newName,
		}

		result, err := service.UpdateTenant(ctx, tenant.ID, input, auditCtx)

		require.NoError(t, err)
		assert.Equal(t, "New Name", result.Name)
		adminRepo.AssertExpectations(t)
		tenantRepo.AssertExpectations(t)
	})

	t.Run("returns error when tenant not found", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()
		tenantID := uuid.New()

		adminRepo.On("FindByID", ctx, tenantID).Return(nil, shared.ErrNotFound)

		newName := "New Name"
		input := AdminUpdateTenantInput{Name: &newName}

		result, err := service.UpdateTenant(ctx, tenantID, input, auditCtx)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "Tenant not found")
		adminRepo.AssertExpectations(t)
	})

	t.Run("validates email when updated", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		tenant := createTestTenant("TEST001", "Test Tenant", identity.TenantPlanBasic)

		adminRepo.On("FindByID", ctx, tenant.ID).Return(tenant, nil)

		invalidEmail := "invalid-email"
		input := AdminUpdateTenantInput{ContactEmail: &invalidEmail}

		result, err := service.UpdateTenant(ctx, tenant.ID, input, auditCtx)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "email")
	})
}

func TestAdminTenantService_DeleteTenant(t *testing.T) {
	t.Run("soft deletes tenant successfully", func(t *testing.T) {
		service, adminRepo, tenantRepo, _, _, auditLogRepo := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		tenant := createTestTenant("DELETE001", "Delete Me", identity.TenantPlanBasic)

		adminRepo.On("FindByID", ctx, tenant.ID).Return(tenant, nil)
		tenantRepo.On("Save", ctx, mock.MatchedBy(func(t *identity.Tenant) bool {
			return t.Status == identity.TenantStatusInactive
		})).Return(nil)
		auditLogRepo.On("Create", ctx, mock.AnythingOfType("*identity.AuditLog")).Return(nil)

		err := service.DeleteTenant(ctx, tenant.ID, auditCtx)

		require.NoError(t, err)
		adminRepo.AssertExpectations(t)
		tenantRepo.AssertExpectations(t)
	})

	t.Run("prevents deletion of system tenant", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		systemTenant := createTestTenant(identity.SystemTenantCode, "System Tenant", identity.TenantPlanEnterprise)

		adminRepo.On("FindByID", ctx, systemTenant.ID).Return(systemTenant, nil)

		err := service.DeleteTenant(ctx, systemTenant.ID, auditCtx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot delete system tenant")
	})

	t.Run("returns error when tenant not found", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()
		tenantID := uuid.New()

		adminRepo.On("FindByID", ctx, tenantID).Return(nil, shared.ErrNotFound)

		err := service.DeleteTenant(ctx, tenantID, auditCtx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Tenant not found")
	})
}

func TestAdminTenantService_ActivateTenant(t *testing.T) {
	t.Run("activates suspended tenant", func(t *testing.T) {
		service, adminRepo, tenantRepo, _, statusHistoryRepo, auditLogRepo := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		tenant := createTestTenant("SUSPENDED001", "Suspended Tenant", identity.TenantPlanBasic)
		tenant.SuspendWithReason("Test suspension", nil)

		adminRepo.On("FindByID", ctx, tenant.ID).Return(tenant, nil)
		tenantRepo.On("Save", ctx, mock.MatchedBy(func(t *identity.Tenant) bool {
			return t.Status == identity.TenantStatusActive
		})).Return(nil)
		statusHistoryRepo.On("Create", ctx, mock.AnythingOfType("*identity.TenantStatusHistory")).Return(nil)
		auditLogRepo.On("Create", ctx, mock.AnythingOfType("*identity.AuditLog")).Return(nil)

		result, err := service.ActivateTenant(ctx, tenant.ID, auditCtx)

		require.NoError(t, err)
		assert.Equal(t, "active", result.Status)
		adminRepo.AssertExpectations(t)
		tenantRepo.AssertExpectations(t)
	})

	t.Run("returns error when tenant not found", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()
		tenantID := uuid.New()

		adminRepo.On("FindByID", ctx, tenantID).Return(nil, shared.ErrNotFound)

		result, err := service.ActivateTenant(ctx, tenantID, auditCtx)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "Tenant not found")
	})
}

func TestAdminTenantService_SuspendTenant(t *testing.T) {
	t.Run("suspends active tenant with reason", func(t *testing.T) {
		service, adminRepo, tenantRepo, _, statusHistoryRepo, auditLogRepo := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		tenant := createTestTenant("ACTIVE001", "Active Tenant", identity.TenantPlanPro)

		adminRepo.On("FindByID", ctx, tenant.ID).Return(tenant, nil)
		tenantRepo.On("Save", ctx, mock.MatchedBy(func(t *identity.Tenant) bool {
			return t.Status == identity.TenantStatusSuspended
		})).Return(nil)
		statusHistoryRepo.On("Create", ctx, mock.AnythingOfType("*identity.TenantStatusHistory")).Return(nil)
		auditLogRepo.On("Create", ctx, mock.AnythingOfType("*identity.AuditLog")).Return(nil)

		input := SuspendTenantInput{
			TenantID: tenant.ID,
			Reason:   "Payment overdue",
		}

		result, err := service.SuspendTenantWithReason(ctx, input, auditCtx)

		require.NoError(t, err)
		assert.Equal(t, "suspended", result.Status)
		assert.Equal(t, "Payment overdue", result.SuspensionReason)
		adminRepo.AssertExpectations(t)
		tenantRepo.AssertExpectations(t)
	})

	t.Run("suspends with scheduled reactivation", func(t *testing.T) {
		service, adminRepo, tenantRepo, _, statusHistoryRepo, auditLogRepo := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		tenant := createTestTenant("ACTIVE002", "Active Tenant", identity.TenantPlanPro)
		reactivateAt := time.Now().Add(24 * time.Hour)

		adminRepo.On("FindByID", ctx, tenant.ID).Return(tenant, nil)
		tenantRepo.On("Save", ctx, mock.AnythingOfType("*identity.Tenant")).Return(nil)
		statusHistoryRepo.On("Create", ctx, mock.AnythingOfType("*identity.TenantStatusHistory")).Return(nil)
		auditLogRepo.On("Create", ctx, mock.AnythingOfType("*identity.AuditLog")).Return(nil)

		input := SuspendTenantInput{
			TenantID:              tenant.ID,
			Reason:                "Temporary suspension",
			ScheduledReactivateAt: &reactivateAt,
		}

		result, err := service.SuspendTenantWithReason(ctx, input, auditCtx)

		require.NoError(t, err)
		assert.Equal(t, "suspended", result.Status)
		assert.NotNil(t, result.ScheduledReactivateAt)
		adminRepo.AssertExpectations(t)
	})

	t.Run("prevents suspension of system tenant", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		systemTenant := createTestTenant(identity.SystemTenantCode, "System Tenant", identity.TenantPlanEnterprise)

		adminRepo.On("FindByID", ctx, systemTenant.ID).Return(systemTenant, nil)

		input := SuspendTenantInput{
			TenantID: systemTenant.ID,
			Reason:   "Test",
		}

		result, err := service.SuspendTenantWithReason(ctx, input, auditCtx)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "Cannot suspend system tenant")
	})
}

func TestAdminTenantService_GetStatistics(t *testing.T) {
	t.Run("returns aggregated statistics", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()

		stats := &identity.TenantStatistics{
			TotalTenants:     100,
			ActiveTenants:    80,
			InactiveTenants:  10,
			SuspendedTenants: 5,
			TrialTenants:     5,
			ByPlan: map[string]int64{
				"free":       30,
				"basic":      40,
				"pro":        25,
				"enterprise": 5,
			},
		}

		adminRepo.On("GetStatistics", ctx).Return(stats, nil)

		result, err := service.GetStatistics(ctx)

		require.NoError(t, err)
		assert.Equal(t, int64(100), result.TotalTenants)
		assert.Equal(t, int64(80), result.ActiveTenants)
		adminRepo.AssertExpectations(t)
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()

		adminRepo.On("GetStatistics", ctx).Return(nil, errors.New("db error"))

		result, err := service.GetStatistics(ctx)

		assert.Error(t, err)
		assert.Nil(t, result)
		adminRepo.AssertExpectations(t)
	})
}

func TestAdminTenantService_ChangePlan(t *testing.T) {
	t.Run("upgrades plan immediately", func(t *testing.T) {
		service, adminRepo, tenantRepo, subHistoryRepo, _, auditLogRepo := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		tenant := createTestTenant("UPGRADE001", "Upgrade Tenant", identity.TenantPlanBasic)

		adminRepo.On("FindByID", ctx, tenant.ID).Return(tenant, nil)
		tenantRepo.On("Save", ctx, mock.AnythingOfType("*identity.Tenant")).Return(nil)
		subHistoryRepo.On("Create", ctx, mock.AnythingOfType("*identity.SubscriptionHistory")).Return(nil)
		auditLogRepo.On("Create", ctx, mock.AnythingOfType("*identity.AuditLog")).Return(nil)

		input := ChangePlanInput{
			TenantID: tenant.ID,
			NewPlan:  "pro",
			Reason:   "Customer upgrade",
		}

		result, err := service.ChangePlan(ctx, input, auditCtx)

		require.NoError(t, err)
		assert.Equal(t, "immediate", result.ChangeType)
		assert.Equal(t, "basic", result.PreviousPlan)
		assert.Equal(t, "pro", result.NewPlan)
		adminRepo.AssertExpectations(t)
	})

	t.Run("schedules downgrade", func(t *testing.T) {
		service, adminRepo, tenantRepo, subHistoryRepo, _, auditLogRepo := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		tenant := createTestTenant("DOWNGRADE001", "Downgrade Tenant", identity.TenantPlanPro)

		adminRepo.On("FindByID", ctx, tenant.ID).Return(tenant, nil)
		tenantRepo.On("Save", ctx, mock.AnythingOfType("*identity.Tenant")).Return(nil)
		subHistoryRepo.On("Create", ctx, mock.AnythingOfType("*identity.SubscriptionHistory")).Return(nil)
		auditLogRepo.On("Create", ctx, mock.AnythingOfType("*identity.AuditLog")).Return(nil)

		input := ChangePlanInput{
			TenantID: tenant.ID,
			NewPlan:  "basic",
			Reason:   "Cost reduction",
		}

		result, err := service.ChangePlan(ctx, input, auditCtx)

		require.NoError(t, err)
		assert.Equal(t, "scheduled", result.ChangeType)
		assert.Equal(t, "pro", result.PreviousPlan)
		assert.Equal(t, "basic", result.NewPlan)
		adminRepo.AssertExpectations(t)
	})

	t.Run("fails with same plan", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		tenant := createTestTenant("SAME001", "Same Plan Tenant", identity.TenantPlanPro)

		adminRepo.On("FindByID", ctx, tenant.ID).Return(tenant, nil)

		input := ChangePlanInput{
			TenantID: tenant.ID,
			NewPlan:  "pro",
		}

		result, err := service.ChangePlan(ctx, input, auditCtx)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "same as current")
	})

	t.Run("fails with invalid plan", func(t *testing.T) {
		service, _, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		input := ChangePlanInput{
			TenantID: uuid.New(),
			NewPlan:  "invalid",
		}

		result, err := service.ChangePlan(ctx, input, auditCtx)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "Invalid")
	})

	t.Run("prevents modification of system tenant", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		systemTenant := createTestTenant(identity.SystemTenantCode, "System Tenant", identity.TenantPlanEnterprise)

		adminRepo.On("FindByID", ctx, systemTenant.ID).Return(systemTenant, nil)

		input := ChangePlanInput{
			TenantID: systemTenant.ID,
			NewPlan:  "basic",
		}

		result, err := service.ChangePlan(ctx, input, auditCtx)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "Cannot modify system tenant")
	})
}

func TestAdminTenantService_UpdateQuota(t *testing.T) {
	t.Run("updates quota successfully", func(t *testing.T) {
		service, adminRepo, tenantRepo, subHistoryRepo, _, auditLogRepo := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		tenant := createTestTenant("QUOTA001", "Quota Tenant", identity.TenantPlanPro)

		adminRepo.On("FindByID", ctx, tenant.ID).Return(tenant, nil)
		tenantRepo.On("Save", ctx, mock.AnythingOfType("*identity.Tenant")).Return(nil)
		subHistoryRepo.On("Create", ctx, mock.AnythingOfType("*identity.SubscriptionHistory")).Return(nil)
		auditLogRepo.On("Create", ctx, mock.AnythingOfType("*identity.AuditLog")).Return(nil)

		maxUsers := 100
		maxProducts := 10000
		input := UpdateQuotaInput{
			TenantID:    tenant.ID,
			MaxUsers:    &maxUsers,
			MaxProducts: &maxProducts,
			Reason:      "Custom quota",
		}

		result, err := service.UpdateQuota(ctx, input, auditCtx)

		require.NoError(t, err)
		assert.Equal(t, 100, result.Config.MaxUsers)
		assert.Equal(t, 10000, result.Config.MaxProducts)
		adminRepo.AssertExpectations(t)
	})

	t.Run("fails with invalid max users", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		tenant := createTestTenant("QUOTA002", "Quota Tenant", identity.TenantPlanPro)

		adminRepo.On("FindByID", ctx, tenant.ID).Return(tenant, nil)

		invalidMaxUsers := 0
		input := UpdateQuotaInput{
			TenantID: tenant.ID,
			MaxUsers: &invalidMaxUsers,
		}

		result, err := service.UpdateQuota(ctx, input, auditCtx)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "at least 1")
	})
}

func TestAdminTenantService_GetSubscriptionHistory(t *testing.T) {
	t.Run("returns paginated history", func(t *testing.T) {
		service, adminRepo, _, subHistoryRepo, _, _ := newTestAdminTenantService()
		ctx := context.Background()

		tenant := createTestTenant("HISTORY001", "History Tenant", identity.TenantPlanPro)

		adminRepo.On("FindByID", ctx, tenant.ID).Return(tenant, nil)

		history1, _ := identity.NewSubscriptionHistory(
			tenant.ID,
			uuid.New(),
			identity.SubscriptionChangeTypePlanUpgrade,
		).WithPlanChange(identity.TenantPlanBasic, identity.TenantPlanPro).Build()

		subHistoryRepo.On("FindByTenantID", ctx, tenant.ID, mock.AnythingOfType("identity.SubscriptionHistoryFilter")).
			Return([]identity.SubscriptionHistory{*history1}, int64(1), nil)

		input := SubscriptionHistoryFilterInput{Page: 1, PageSize: 20}
		result, err := service.GetSubscriptionHistory(ctx, tenant.ID, input)

		require.NoError(t, err)
		assert.Len(t, result.History, 1)
		assert.Equal(t, int64(1), result.Total)
		adminRepo.AssertExpectations(t)
		subHistoryRepo.AssertExpectations(t)
	})

	t.Run("returns empty result when no history", func(t *testing.T) {
		service, adminRepo, _, subHistoryRepo, _, _ := newTestAdminTenantService()
		ctx := context.Background()

		tenant := createTestTenant("NOHISTORY001", "No History Tenant", identity.TenantPlanBasic)

		adminRepo.On("FindByID", ctx, tenant.ID).Return(tenant, nil)
		subHistoryRepo.On("FindByTenantID", ctx, tenant.ID, mock.AnythingOfType("identity.SubscriptionHistoryFilter")).
			Return([]identity.SubscriptionHistory{}, int64(0), nil)

		input := SubscriptionHistoryFilterInput{Page: 1, PageSize: 20}
		result, err := service.GetSubscriptionHistory(ctx, tenant.ID, input)

		require.NoError(t, err)
		assert.Empty(t, result.History)
		assert.Equal(t, int64(0), result.Total)
	})
}

func TestAdminTenantService_CancelScheduledPlanChange(t *testing.T) {
	t.Run("cancels scheduled downgrade", func(t *testing.T) {
		service, adminRepo, tenantRepo, _, _, auditLogRepo := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		tenant := createTestTenant("CANCEL001", "Cancel Tenant", identity.TenantPlanPro)
		effectiveAt := time.Now().AddDate(0, 1, 0)
		tenant.SchedulePlanDowngrade(identity.TenantPlanBasic, effectiveAt)

		adminRepo.On("FindByID", ctx, tenant.ID).Return(tenant, nil)
		tenantRepo.On("Save", ctx, mock.MatchedBy(func(t *identity.Tenant) bool {
			return t.ScheduledPlan == nil
		})).Return(nil)
		auditLogRepo.On("Create", ctx, mock.AnythingOfType("*identity.AuditLog")).Return(nil)

		result, err := service.CancelScheduledPlanChange(ctx, tenant.ID, auditCtx)

		require.NoError(t, err)
		assert.Nil(t, result.ScheduledPlan)
		adminRepo.AssertExpectations(t)
		tenantRepo.AssertExpectations(t)
	})

	t.Run("fails when no scheduled change exists", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		tenant := createTestTenant("NOCHANGE001", "No Change Tenant", identity.TenantPlanPro)

		adminRepo.On("FindByID", ctx, tenant.ID).Return(tenant, nil)

		result, err := service.CancelScheduledPlanChange(ctx, tenant.ID, auditCtx)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "No scheduled")
	})
}

// ============================================================================
// Validation Helper Tests
// ============================================================================

func TestAdminTenantService_ValidateName(t *testing.T) {
	service, _, _, _, _, _ := newTestAdminTenantService()

	t.Run("valid name", func(t *testing.T) {
		err := service.validateName("Valid Tenant Name")
		assert.NoError(t, err)
	})

	t.Run("empty name", func(t *testing.T) {
		err := service.validateName("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name")
	})

	t.Run("name too long", func(t *testing.T) {
		longName := string(make([]byte, 201))
		err := service.validateName(longName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name")
	})
}

func TestAdminTenantService_ValidateEmail(t *testing.T) {
	service, _, _, _, _, _ := newTestAdminTenantService()

	t.Run("valid email", func(t *testing.T) {
		err := service.validateEmail("test@example.com")
		assert.NoError(t, err)
	})

	t.Run("invalid email format", func(t *testing.T) {
		err := service.validateEmail("invalid-email")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "email")
	})

	t.Run("email too long", func(t *testing.T) {
		longEmail := string(make([]byte, 201)) + "@example.com"
		err := service.validateEmail(longEmail)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Email")
	})
}

func TestAdminTenantService_ValidatePlan(t *testing.T) {
	service, _, _, _, _, _ := newTestAdminTenantService()

	t.Run("valid plans", func(t *testing.T) {
		validPlans := []string{"free", "basic", "pro", "enterprise"}
		for _, plan := range validPlans {
			err := service.validatePlan(plan)
			assert.NoError(t, err, "Plan %s should be valid", plan)
		}
	})

	t.Run("invalid plan", func(t *testing.T) {
		err := service.validatePlan("invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid")
	})
}
