package billing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/billing"
	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Mock implementations

type mockUsageQuotaRepository struct {
	mock.Mock
}

func (m *mockUsageQuotaRepository) Save(ctx context.Context, quota *billing.UsageQuota) error {
	args := m.Called(ctx, quota)
	return args.Error(0)
}

func (m *mockUsageQuotaRepository) Update(ctx context.Context, quota *billing.UsageQuota) error {
	args := m.Called(ctx, quota)
	return args.Error(0)
}

func (m *mockUsageQuotaRepository) FindByID(ctx context.Context, id uuid.UUID) (*billing.UsageQuota, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*billing.UsageQuota), args.Error(1)
}

func (m *mockUsageQuotaRepository) FindByPlan(ctx context.Context, planID string) ([]*billing.UsageQuota, error) {
	args := m.Called(ctx, planID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*billing.UsageQuota), args.Error(1)
}

func (m *mockUsageQuotaRepository) FindByPlanAndType(ctx context.Context, planID string, usageType billing.UsageType) (*billing.UsageQuota, error) {
	args := m.Called(ctx, planID, usageType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*billing.UsageQuota), args.Error(1)
}

func (m *mockUsageQuotaRepository) FindByTenant(ctx context.Context, tenantID uuid.UUID) ([]*billing.UsageQuota, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*billing.UsageQuota), args.Error(1)
}

func (m *mockUsageQuotaRepository) FindByTenantAndType(ctx context.Context, tenantID uuid.UUID, usageType billing.UsageType) (*billing.UsageQuota, error) {
	args := m.Called(ctx, tenantID, usageType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*billing.UsageQuota), args.Error(1)
}

func (m *mockUsageQuotaRepository) FindEffectiveQuota(ctx context.Context, tenantID uuid.UUID, planID string, usageType billing.UsageType) (*billing.UsageQuota, error) {
	args := m.Called(ctx, tenantID, planID, usageType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*billing.UsageQuota), args.Error(1)
}

func (m *mockUsageQuotaRepository) FindAllEffectiveQuotas(ctx context.Context, tenantID uuid.UUID, planID string) ([]*billing.UsageQuota, error) {
	args := m.Called(ctx, tenantID, planID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*billing.UsageQuota), args.Error(1)
}

func (m *mockUsageQuotaRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockUsageQuotaRepository) DeleteByTenant(ctx context.Context, tenantID uuid.UUID) error {
	args := m.Called(ctx, tenantID)
	return args.Error(0)
}

type mockUsageRecordRepository struct {
	mock.Mock
}

func (m *mockUsageRecordRepository) Save(ctx context.Context, record *billing.UsageRecord) error {
	args := m.Called(ctx, record)
	return args.Error(0)
}

func (m *mockUsageRecordRepository) SaveBatch(ctx context.Context, records []*billing.UsageRecord) error {
	args := m.Called(ctx, records)
	return args.Error(0)
}

func (m *mockUsageRecordRepository) FindByID(ctx context.Context, id uuid.UUID) (*billing.UsageRecord, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*billing.UsageRecord), args.Error(1)
}

func (m *mockUsageRecordRepository) FindByTenant(ctx context.Context, tenantID uuid.UUID, filter billing.UsageRecordFilter) ([]*billing.UsageRecord, error) {
	args := m.Called(ctx, tenantID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*billing.UsageRecord), args.Error(1)
}

func (m *mockUsageRecordRepository) FindByTenantAndType(ctx context.Context, tenantID uuid.UUID, usageType billing.UsageType, filter billing.UsageRecordFilter) ([]*billing.UsageRecord, error) {
	args := m.Called(ctx, tenantID, usageType, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*billing.UsageRecord), args.Error(1)
}

func (m *mockUsageRecordRepository) CountByTenant(ctx context.Context, tenantID uuid.UUID, filter billing.UsageRecordFilter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockUsageRecordRepository) SumByTenantAndType(ctx context.Context, tenantID uuid.UUID, usageType billing.UsageType, start, end time.Time) (int64, error) {
	args := m.Called(ctx, tenantID, usageType, start, end)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockUsageRecordRepository) GetAggregatedUsage(ctx context.Context, tenantID uuid.UUID, usageType billing.UsageType, start, end time.Time, groupBy billing.AggregationPeriod) ([]billing.UsageAggregation, error) {
	args := m.Called(ctx, tenantID, usageType, start, end, groupBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]billing.UsageAggregation), args.Error(1)
}

func (m *mockUsageRecordRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	args := m.Called(ctx, before)
	return args.Get(0).(int64), args.Error(1)
}

type mockTenantRepository struct {
	mock.Mock
}

func (m *mockTenantRepository) FindByID(ctx context.Context, id uuid.UUID) (*identity.Tenant, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.Tenant), args.Error(1)
}

func (m *mockTenantRepository) FindByCode(ctx context.Context, code string) (*identity.Tenant, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.Tenant), args.Error(1)
}

func (m *mockTenantRepository) FindByDomain(ctx context.Context, domain string) (*identity.Tenant, error) {
	args := m.Called(ctx, domain)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.Tenant), args.Error(1)
}

func (m *mockTenantRepository) FindAll(ctx context.Context, filter shared.Filter) ([]identity.Tenant, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *mockTenantRepository) FindByStatus(ctx context.Context, status identity.TenantStatus, filter shared.Filter) ([]identity.Tenant, error) {
	args := m.Called(ctx, status, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *mockTenantRepository) FindByPlan(ctx context.Context, plan identity.TenantPlan, filter shared.Filter) ([]identity.Tenant, error) {
	args := m.Called(ctx, plan, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *mockTenantRepository) FindActive(ctx context.Context, filter shared.Filter) ([]identity.Tenant, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *mockTenantRepository) FindExpiringSoon(ctx context.Context, days int, filter shared.Filter) ([]identity.Tenant, error) {
	args := m.Called(ctx, days, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *mockTenantRepository) Save(ctx context.Context, tenant *identity.Tenant) error {
	args := m.Called(ctx, tenant)
	return args.Error(0)
}

func (m *mockTenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockTenantRepository) ExistsByCode(ctx context.Context, code string) (bool, error) {
	args := m.Called(ctx, code)
	return args.Bool(0), args.Error(1)
}

func (m *mockTenantRepository) ExistsByDomain(ctx context.Context, domain string) (bool, error) {
	args := m.Called(ctx, domain)
	return args.Bool(0), args.Error(1)
}

func (m *mockTenantRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockTenantRepository) CountByStatus(ctx context.Context, status identity.TenantStatus) (int64, error) {
	args := m.Called(ctx, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockTenantRepository) CountByPlan(ctx context.Context, plan identity.TenantPlan) (int64, error) {
	args := m.Called(ctx, plan)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockTenantRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]identity.Tenant, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *mockTenantRepository) FindTrialExpiring(ctx context.Context, withinDays int) ([]identity.Tenant, error) {
	args := m.Called(ctx, withinDays)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *mockTenantRepository) FindSubscriptionExpiring(ctx context.Context, withinDays int) ([]identity.Tenant, error) {
	args := m.Called(ctx, withinDays)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

type mockUsageEventPublisher struct {
	mock.Mock
}

func (m *mockUsageEventPublisher) PublishQuotaWarning(ctx context.Context, tenantID uuid.UUID, result billing.QuotaCheckResult) error {
	args := m.Called(ctx, tenantID, result)
	return args.Error(0)
}

func (m *mockUsageEventPublisher) PublishQuotaExceeded(ctx context.Context, tenantID uuid.UUID, result billing.QuotaCheckResult) error {
	args := m.Called(ctx, tenantID, result)
	return args.Error(0)
}

func (m *mockUsageEventPublisher) PublishUsageRecorded(ctx context.Context, record *billing.UsageRecord) error {
	args := m.Called(ctx, record)
	return args.Error(0)
}

// Helper functions

func createTestTenant(plan identity.TenantPlan) *identity.Tenant {
	tenant, _ := identity.NewTenant("TEST", "Test Tenant")
	tenant.Plan = plan
	return tenant
}

func createTestQuota(usageType billing.UsageType, limit int64, policy billing.OveragePolicy) *billing.UsageQuota {
	quota, _ := billing.NewUsageQuota("basic", usageType, limit, billing.ResetPeriodMonthly)
	quota.WithOveragePolicy(policy)
	return quota
}

func createTestQuotaWithSoftLimit(usageType billing.UsageType, limit, softLimit int64, policy billing.OveragePolicy) *billing.UsageQuota {
	quota := createTestQuota(usageType, limit, policy)
	quota.WithSoftLimit(softLimit)
	return quota
}

func newTestQuotaService(
	quotaRepo *mockUsageQuotaRepository,
	usageRepo *mockUsageRecordRepository,
	tenantRepo *mockTenantRepository,
	eventPublisher *mockUsageEventPublisher,
) *QuotaService {
	logger := zap.NewNop()
	return NewQuotaService(
		quotaRepo,
		usageRepo,
		nil, // meterRepo
		tenantRepo,
		eventPublisher,
		logger,
		DefaultQuotaServiceConfig(),
	)
}

// Tests for QuotaExceededError

func TestQuotaExceededError(t *testing.T) {
	t.Run("creates error with correct message", func(t *testing.T) {
		err := NewQuotaExceededError(billing.UsageTypeOrdersCreated, 150, 100)

		assert.NotNil(t, err)
		assert.Equal(t, billing.UsageTypeOrdersCreated, err.UsageType)
		assert.Equal(t, int64(150), err.CurrentUsage)
		assert.Equal(t, int64(100), err.Limit)
		assert.Contains(t, err.Error(), "Orders Created")
		assert.Contains(t, err.Error(), "150")
		assert.Contains(t, err.Error(), "100")
	})

	t.Run("returns 429 status code", func(t *testing.T) {
		err := NewQuotaExceededError(billing.UsageTypeActiveUsers, 10, 5)

		assert.Equal(t, 429, err.HTTPStatusCode())
	})
}

// Tests for CheckQuota

func TestCheckQuota_WithinLimit(t *testing.T) {
	t.Run("allows operation when within quota", func(t *testing.T) {
		quotaRepo := new(mockUsageQuotaRepository)
		usageRepo := new(mockUsageRecordRepository)
		tenantRepo := new(mockTenantRepository)
		eventPublisher := new(mockUsageEventPublisher)

		tenantID := uuid.New()
		tenant := createTestTenant(identity.TenantPlanBasic)
		tenant.ID = tenantID

		quota := createTestQuota(billing.UsageTypeOrdersCreated, 100, billing.OveragePolicyBlock)

		tenantRepo.On("FindByID", mock.Anything, tenantID).Return(tenant, nil)
		quotaRepo.On("FindEffectiveQuota", mock.Anything, tenantID, "basic", billing.UsageTypeOrdersCreated).Return(quota, nil)
		usageRepo.On("SumByTenantAndType", mock.Anything, tenantID, billing.UsageTypeOrdersCreated, mock.Anything, mock.Anything).Return(int64(50), nil)

		service := newTestQuotaService(quotaRepo, usageRepo, tenantRepo, eventPublisher)

		result, err := service.CheckQuota(context.Background(), QuotaCheckInput{
			TenantID:  tenantID,
			UsageType: billing.UsageTypeOrdersCreated,
			Amount:    1,
		})

		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, billing.QuotaStatusOK, result.Status)
		assert.Equal(t, int64(50), result.CurrentUsage)
		assert.Equal(t, int64(100), result.Limit)
		assert.Nil(t, result.Error)
		assert.Nil(t, result.Warning)
	})
}

func TestCheckQuota_ExceedsLimit(t *testing.T) {
	t.Run("blocks operation when quota exceeded with BLOCK policy", func(t *testing.T) {
		quotaRepo := new(mockUsageQuotaRepository)
		usageRepo := new(mockUsageRecordRepository)
		tenantRepo := new(mockTenantRepository)
		eventPublisher := new(mockUsageEventPublisher)

		tenantID := uuid.New()
		tenant := createTestTenant(identity.TenantPlanBasic)
		tenant.ID = tenantID

		quota := createTestQuota(billing.UsageTypeOrdersCreated, 100, billing.OveragePolicyBlock)

		tenantRepo.On("FindByID", mock.Anything, tenantID).Return(tenant, nil)
		quotaRepo.On("FindEffectiveQuota", mock.Anything, tenantID, "basic", billing.UsageTypeOrdersCreated).Return(quota, nil)
		usageRepo.On("SumByTenantAndType", mock.Anything, tenantID, billing.UsageTypeOrdersCreated, mock.Anything, mock.Anything).Return(int64(100), nil)
		// Both warning and exceeded events may be published for exceeded status
		eventPublisher.On("PublishQuotaWarning", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		eventPublisher.On("PublishQuotaExceeded", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

		service := newTestQuotaService(quotaRepo, usageRepo, tenantRepo, eventPublisher)

		result, err := service.CheckQuota(context.Background(), QuotaCheckInput{
			TenantID:  tenantID,
			UsageType: billing.UsageTypeOrdersCreated,
			Amount:    1,
		})

		require.NoError(t, err)
		assert.False(t, result.Allowed)
		assert.Equal(t, billing.QuotaStatusExceeded, result.Status)
		assert.NotNil(t, result.Error)
		assert.Equal(t, 429, result.Error.HTTPStatusCode())
	})

	t.Run("allows operation when quota exceeded with WARN policy", func(t *testing.T) {
		quotaRepo := new(mockUsageQuotaRepository)
		usageRepo := new(mockUsageRecordRepository)
		tenantRepo := new(mockTenantRepository)
		eventPublisher := new(mockUsageEventPublisher)

		tenantID := uuid.New()
		tenant := createTestTenant(identity.TenantPlanBasic)
		tenant.ID = tenantID

		quota := createTestQuota(billing.UsageTypeOrdersCreated, 100, billing.OveragePolicyWarn)

		tenantRepo.On("FindByID", mock.Anything, tenantID).Return(tenant, nil)
		quotaRepo.On("FindEffectiveQuota", mock.Anything, tenantID, "basic", billing.UsageTypeOrdersCreated).Return(quota, nil)
		usageRepo.On("SumByTenantAndType", mock.Anything, tenantID, billing.UsageTypeOrdersCreated, mock.Anything, mock.Anything).Return(int64(100), nil)
		eventPublisher.On("PublishQuotaWarning", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

		service := newTestQuotaService(quotaRepo, usageRepo, tenantRepo, eventPublisher)

		result, err := service.CheckQuota(context.Background(), QuotaCheckInput{
			TenantID:  tenantID,
			UsageType: billing.UsageTypeOrdersCreated,
			Amount:    1,
		})

		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, billing.QuotaStatusExceeded, result.Status)
		assert.NotNil(t, result.Warning)
	})
}

func TestCheckQuota_SoftLimit(t *testing.T) {
	t.Run("returns warning when soft limit reached", func(t *testing.T) {
		quotaRepo := new(mockUsageQuotaRepository)
		usageRepo := new(mockUsageRecordRepository)
		tenantRepo := new(mockTenantRepository)
		eventPublisher := new(mockUsageEventPublisher)

		tenantID := uuid.New()
		tenant := createTestTenant(identity.TenantPlanBasic)
		tenant.ID = tenantID

		quota := createTestQuotaWithSoftLimit(billing.UsageTypeOrdersCreated, 100, 80, billing.OveragePolicyBlock)

		tenantRepo.On("FindByID", mock.Anything, tenantID).Return(tenant, nil)
		quotaRepo.On("FindEffectiveQuota", mock.Anything, tenantID, "basic", billing.UsageTypeOrdersCreated).Return(quota, nil)
		usageRepo.On("SumByTenantAndType", mock.Anything, tenantID, billing.UsageTypeOrdersCreated, mock.Anything, mock.Anything).Return(int64(80), nil)
		eventPublisher.On("PublishQuotaWarning", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

		service := newTestQuotaService(quotaRepo, usageRepo, tenantRepo, eventPublisher)

		result, err := service.CheckQuota(context.Background(), QuotaCheckInput{
			TenantID:  tenantID,
			UsageType: billing.UsageTypeOrdersCreated,
			Amount:    1,
		})

		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, billing.QuotaStatusWarning, result.Status)
		assert.NotNil(t, result.Warning)
		assert.Equal(t, int64(80), result.Warning.SoftLimit)
	})
}

func TestCheckQuota_NoQuotaDefined(t *testing.T) {
	t.Run("allows unlimited usage when no quota defined", func(t *testing.T) {
		quotaRepo := new(mockUsageQuotaRepository)
		usageRepo := new(mockUsageRecordRepository)
		tenantRepo := new(mockTenantRepository)
		eventPublisher := new(mockUsageEventPublisher)

		tenantID := uuid.New()
		tenant := createTestTenant(identity.TenantPlanBasic)
		tenant.ID = tenantID

		tenantRepo.On("FindByID", mock.Anything, tenantID).Return(tenant, nil)
		quotaRepo.On("FindEffectiveQuota", mock.Anything, tenantID, "basic", billing.UsageTypeOrdersCreated).Return(nil, shared.ErrNotFound)

		service := newTestQuotaService(quotaRepo, usageRepo, tenantRepo, eventPublisher)

		result, err := service.CheckQuota(context.Background(), QuotaCheckInput{
			TenantID:  tenantID,
			UsageType: billing.UsageTypeOrdersCreated,
			Amount:    1,
		})

		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, int64(-1), result.Limit) // Unlimited
		assert.Equal(t, billing.QuotaStatusOK, result.Status)
	})
}

func TestCheckQuota_InvalidInput(t *testing.T) {
	t.Run("returns error for empty tenant ID", func(t *testing.T) {
		quotaRepo := new(mockUsageQuotaRepository)
		usageRepo := new(mockUsageRecordRepository)
		tenantRepo := new(mockTenantRepository)
		eventPublisher := new(mockUsageEventPublisher)

		service := newTestQuotaService(quotaRepo, usageRepo, tenantRepo, eventPublisher)

		result, err := service.CheckQuota(context.Background(), QuotaCheckInput{
			TenantID:  uuid.Nil,
			UsageType: billing.UsageTypeOrdersCreated,
			Amount:    1,
		})

		assert.Nil(t, result)
		assert.Error(t, err)
		var domainErr *shared.DomainError
		assert.True(t, errors.As(err, &domainErr))
		assert.Equal(t, "INVALID_TENANT", domainErr.Code)
	})

	t.Run("returns error for invalid usage type", func(t *testing.T) {
		quotaRepo := new(mockUsageQuotaRepository)
		usageRepo := new(mockUsageRecordRepository)
		tenantRepo := new(mockTenantRepository)
		eventPublisher := new(mockUsageEventPublisher)

		service := newTestQuotaService(quotaRepo, usageRepo, tenantRepo, eventPublisher)

		result, err := service.CheckQuota(context.Background(), QuotaCheckInput{
			TenantID:  uuid.New(),
			UsageType: billing.UsageType("INVALID"),
			Amount:    1,
		})

		assert.Nil(t, result)
		assert.Error(t, err)
		var domainErr *shared.DomainError
		assert.True(t, errors.As(err, &domainErr))
		assert.Equal(t, "INVALID_USAGE_TYPE", domainErr.Code)
	})

	t.Run("returns error when tenant not found", func(t *testing.T) {
		quotaRepo := new(mockUsageQuotaRepository)
		usageRepo := new(mockUsageRecordRepository)
		tenantRepo := new(mockTenantRepository)
		eventPublisher := new(mockUsageEventPublisher)

		tenantID := uuid.New()
		tenantRepo.On("FindByID", mock.Anything, tenantID).Return(nil, shared.ErrNotFound)

		service := newTestQuotaService(quotaRepo, usageRepo, tenantRepo, eventPublisher)

		result, err := service.CheckQuota(context.Background(), QuotaCheckInput{
			TenantID:  tenantID,
			UsageType: billing.UsageTypeOrdersCreated,
			Amount:    1,
		})

		assert.Nil(t, result)
		assert.Error(t, err)
		var domainErr *shared.DomainError
		assert.True(t, errors.As(err, &domainErr))
		assert.Equal(t, "TENANT_NOT_FOUND", domainErr.Code)
	})
}

// Tests for GetUsageSummary

func TestGetUsageSummary(t *testing.T) {
	t.Run("returns usage summary for tenant", func(t *testing.T) {
		quotaRepo := new(mockUsageQuotaRepository)
		usageRepo := new(mockUsageRecordRepository)
		tenantRepo := new(mockTenantRepository)
		eventPublisher := new(mockUsageEventPublisher)

		tenantID := uuid.New()
		tenant := createTestTenant(identity.TenantPlanBasic)
		tenant.ID = tenantID

		// Use accumulative types (not countable) for testing
		orderQuota := createTestQuotaWithSoftLimit(billing.UsageTypeOrdersCreated, 100, 80, billing.OveragePolicyBlock)
		reportQuota := createTestQuota(billing.UsageTypeReportsGenerated, 50, billing.OveragePolicyBlock)

		quotas := []*billing.UsageQuota{orderQuota, reportQuota}

		tenantRepo.On("FindByID", mock.Anything, tenantID).Return(tenant, nil)
		quotaRepo.On("FindAllEffectiveQuotas", mock.Anything, tenantID, "basic").Return(quotas, nil)
		usageRepo.On("SumByTenantAndType", mock.Anything, tenantID, billing.UsageTypeOrdersCreated, mock.Anything, mock.Anything).Return(int64(50), nil)
		usageRepo.On("SumByTenantAndType", mock.Anything, tenantID, billing.UsageTypeReportsGenerated, mock.Anything, mock.Anything).Return(int64(25), nil)

		service := newTestQuotaService(quotaRepo, usageRepo, tenantRepo, eventPublisher)

		summary, err := service.GetUsageSummary(context.Background(), tenantID, billing.ResetPeriodMonthly)

		require.NoError(t, err)
		assert.NotNil(t, summary)
		assert.Equal(t, tenantID, summary.TenantID)
		assert.Len(t, summary.Usages, 2)

		// Check order usage
		orderUsage, ok := summary.Usages[string(billing.UsageTypeOrdersCreated)]
		assert.True(t, ok)
		assert.Equal(t, int64(50), orderUsage.CurrentUsage)
		assert.Equal(t, int64(100), orderUsage.Limit)
		assert.Equal(t, "OK", orderUsage.Status)

		// Check report usage
		reportUsage, ok := summary.Usages[string(billing.UsageTypeReportsGenerated)]
		assert.True(t, ok)
		assert.Equal(t, int64(25), reportUsage.CurrentUsage)
		assert.Equal(t, int64(50), reportUsage.Limit)
	})

	t.Run("includes warnings for quotas near limit", func(t *testing.T) {
		quotaRepo := new(mockUsageQuotaRepository)
		usageRepo := new(mockUsageRecordRepository)
		tenantRepo := new(mockTenantRepository)
		eventPublisher := new(mockUsageEventPublisher)

		tenantID := uuid.New()
		tenant := createTestTenant(identity.TenantPlanBasic)
		tenant.ID = tenantID

		productQuota := createTestQuotaWithSoftLimit(billing.UsageTypeOrdersCreated, 100, 80, billing.OveragePolicyBlock)
		quotas := []*billing.UsageQuota{productQuota}

		tenantRepo.On("FindByID", mock.Anything, tenantID).Return(tenant, nil)
		quotaRepo.On("FindAllEffectiveQuotas", mock.Anything, tenantID, "basic").Return(quotas, nil)
		usageRepo.On("SumByTenantAndType", mock.Anything, tenantID, billing.UsageTypeOrdersCreated, mock.Anything, mock.Anything).Return(int64(85), nil)

		service := newTestQuotaService(quotaRepo, usageRepo, tenantRepo, eventPublisher)

		summary, err := service.GetUsageSummary(context.Background(), tenantID, billing.ResetPeriodMonthly)

		require.NoError(t, err)
		assert.NotNil(t, summary)
		assert.Len(t, summary.Warnings, 1)
		assert.Equal(t, billing.UsageTypeOrdersCreated, summary.Warnings[0].UsageType)
	})

	t.Run("includes exceeded quotas", func(t *testing.T) {
		quotaRepo := new(mockUsageQuotaRepository)
		usageRepo := new(mockUsageRecordRepository)
		tenantRepo := new(mockTenantRepository)
		eventPublisher := new(mockUsageEventPublisher)

		tenantID := uuid.New()
		tenant := createTestTenant(identity.TenantPlanBasic)
		tenant.ID = tenantID

		productQuota := createTestQuota(billing.UsageTypeOrdersCreated, 100, billing.OveragePolicyBlock)
		quotas := []*billing.UsageQuota{productQuota}

		tenantRepo.On("FindByID", mock.Anything, tenantID).Return(tenant, nil)
		quotaRepo.On("FindAllEffectiveQuotas", mock.Anything, tenantID, "basic").Return(quotas, nil)
		usageRepo.On("SumByTenantAndType", mock.Anything, tenantID, billing.UsageTypeOrdersCreated, mock.Anything, mock.Anything).Return(int64(150), nil)

		service := newTestQuotaService(quotaRepo, usageRepo, tenantRepo, eventPublisher)

		summary, err := service.GetUsageSummary(context.Background(), tenantID, billing.ResetPeriodMonthly)

		require.NoError(t, err)
		assert.NotNil(t, summary)
		assert.Len(t, summary.Exceeded, 1)
		assert.Equal(t, string(billing.UsageTypeOrdersCreated), summary.Exceeded[0])
	})

	t.Run("returns error for invalid tenant ID", func(t *testing.T) {
		quotaRepo := new(mockUsageQuotaRepository)
		usageRepo := new(mockUsageRecordRepository)
		tenantRepo := new(mockTenantRepository)
		eventPublisher := new(mockUsageEventPublisher)

		service := newTestQuotaService(quotaRepo, usageRepo, tenantRepo, eventPublisher)

		summary, err := service.GetUsageSummary(context.Background(), uuid.Nil, billing.ResetPeriodMonthly)

		assert.Nil(t, summary)
		assert.Error(t, err)
		var domainErr *shared.DomainError
		assert.True(t, errors.As(err, &domainErr))
		assert.Equal(t, "INVALID_TENANT", domainErr.Code)
	})
}

// Tests for convenience methods

func TestCheckOrderQuota(t *testing.T) {
	t.Run("returns nil when within quota", func(t *testing.T) {
		quotaRepo := new(mockUsageQuotaRepository)
		usageRepo := new(mockUsageRecordRepository)
		tenantRepo := new(mockTenantRepository)
		eventPublisher := new(mockUsageEventPublisher)

		tenantID := uuid.New()
		tenant := createTestTenant(identity.TenantPlanBasic)
		tenant.ID = tenantID

		quota := createTestQuota(billing.UsageTypeOrdersCreated, 100, billing.OveragePolicyBlock)

		tenantRepo.On("FindByID", mock.Anything, tenantID).Return(tenant, nil)
		quotaRepo.On("FindEffectiveQuota", mock.Anything, tenantID, "basic", billing.UsageTypeOrdersCreated).Return(quota, nil)
		usageRepo.On("SumByTenantAndType", mock.Anything, tenantID, billing.UsageTypeOrdersCreated, mock.Anything, mock.Anything).Return(int64(50), nil)

		service := newTestQuotaService(quotaRepo, usageRepo, tenantRepo, eventPublisher)

		err := service.CheckOrderQuota(context.Background(), tenantID)

		assert.NoError(t, err)
	})

	t.Run("returns QuotaExceededError when exceeded", func(t *testing.T) {
		quotaRepo := new(mockUsageQuotaRepository)
		usageRepo := new(mockUsageRecordRepository)
		tenantRepo := new(mockTenantRepository)
		eventPublisher := new(mockUsageEventPublisher)

		tenantID := uuid.New()
		tenant := createTestTenant(identity.TenantPlanBasic)
		tenant.ID = tenantID

		quota := createTestQuota(billing.UsageTypeOrdersCreated, 100, billing.OveragePolicyBlock)

		tenantRepo.On("FindByID", mock.Anything, tenantID).Return(tenant, nil)
		quotaRepo.On("FindEffectiveQuota", mock.Anything, tenantID, "basic", billing.UsageTypeOrdersCreated).Return(quota, nil)
		usageRepo.On("SumByTenantAndType", mock.Anything, tenantID, billing.UsageTypeOrdersCreated, mock.Anything, mock.Anything).Return(int64(100), nil)
		eventPublisher.On("PublishQuotaWarning", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		eventPublisher.On("PublishQuotaExceeded", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

		service := newTestQuotaService(quotaRepo, usageRepo, tenantRepo, eventPublisher)

		err := service.CheckOrderQuota(context.Background(), tenantID)

		assert.Error(t, err)
		var quotaErr *QuotaExceededError
		assert.True(t, errors.As(err, &quotaErr))
		assert.Equal(t, 429, quotaErr.HTTPStatusCode())
	})
}

func TestCheckReportQuota(t *testing.T) {
	t.Run("returns nil when within quota", func(t *testing.T) {
		quotaRepo := new(mockUsageQuotaRepository)
		usageRepo := new(mockUsageRecordRepository)
		tenantRepo := new(mockTenantRepository)
		eventPublisher := new(mockUsageEventPublisher)

		tenantID := uuid.New()
		tenant := createTestTenant(identity.TenantPlanBasic)
		tenant.ID = tenantID

		quota := createTestQuota(billing.UsageTypeReportsGenerated, 10, billing.OveragePolicyBlock)

		tenantRepo.On("FindByID", mock.Anything, tenantID).Return(tenant, nil)
		quotaRepo.On("FindEffectiveQuota", mock.Anything, tenantID, "basic", billing.UsageTypeReportsGenerated).Return(quota, nil)
		usageRepo.On("SumByTenantAndType", mock.Anything, tenantID, billing.UsageTypeReportsGenerated, mock.Anything, mock.Anything).Return(int64(5), nil)

		service := newTestQuotaService(quotaRepo, usageRepo, tenantRepo, eventPublisher)

		// Use CheckQuotaForResourceCreation directly since there's no specific method for reports
		err := service.CheckQuotaForResourceCreation(context.Background(), tenantID, billing.UsageTypeReportsGenerated)

		assert.NoError(t, err)
	})

	t.Run("returns QuotaExceededError when report limit exceeded", func(t *testing.T) {
		quotaRepo := new(mockUsageQuotaRepository)
		usageRepo := new(mockUsageRecordRepository)
		tenantRepo := new(mockTenantRepository)
		eventPublisher := new(mockUsageEventPublisher)

		tenantID := uuid.New()
		tenant := createTestTenant(identity.TenantPlanBasic)
		tenant.ID = tenantID

		quota := createTestQuota(billing.UsageTypeReportsGenerated, 10, billing.OveragePolicyBlock)

		tenantRepo.On("FindByID", mock.Anything, tenantID).Return(tenant, nil)
		quotaRepo.On("FindEffectiveQuota", mock.Anything, tenantID, "basic", billing.UsageTypeReportsGenerated).Return(quota, nil)
		usageRepo.On("SumByTenantAndType", mock.Anything, tenantID, billing.UsageTypeReportsGenerated, mock.Anything, mock.Anything).Return(int64(10), nil)
		eventPublisher.On("PublishQuotaWarning", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		eventPublisher.On("PublishQuotaExceeded", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

		service := newTestQuotaService(quotaRepo, usageRepo, tenantRepo, eventPublisher)

		err := service.CheckQuotaForResourceCreation(context.Background(), tenantID, billing.UsageTypeReportsGenerated)

		assert.Error(t, err)
		var quotaErr *QuotaExceededError
		assert.True(t, errors.As(err, &quotaErr))
	})
}

// Tests for period boundary calculations

func TestCalculatePeriodBoundaries(t *testing.T) {
	service := &QuotaService{}

	t.Run("calculates daily boundaries", func(t *testing.T) {
		start, end := service.calculatePeriodBoundaries(billing.ResetPeriodDaily)

		now := time.Now()
		expectedStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		assert.Equal(t, expectedStart, start)
		assert.True(t, end.After(start))
		assert.Equal(t, expectedStart.AddDate(0, 0, 1).Add(-time.Nanosecond), end)
	})

	t.Run("calculates monthly boundaries", func(t *testing.T) {
		start, end := service.calculatePeriodBoundaries(billing.ResetPeriodMonthly)

		now := time.Now()
		expectedStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

		assert.Equal(t, expectedStart, start)
		assert.True(t, end.After(start))
	})

	t.Run("calculates yearly boundaries", func(t *testing.T) {
		start, end := service.calculatePeriodBoundaries(billing.ResetPeriodYearly)

		now := time.Now()
		expectedStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())

		assert.Equal(t, expectedStart, start)
		assert.True(t, end.After(start))
	})

	t.Run("calculates never reset boundaries", func(t *testing.T) {
		start, end := service.calculatePeriodBoundaries(billing.ResetPeriodNever)

		assert.Equal(t, 2000, start.Year())
		assert.Equal(t, 2100, end.Year())
	})
}

// Tests for GetQuotaStatus

func TestGetQuotaStatus(t *testing.T) {
	t.Run("returns quota status for all types", func(t *testing.T) {
		quotaRepo := new(mockUsageQuotaRepository)
		usageRepo := new(mockUsageRecordRepository)
		tenantRepo := new(mockTenantRepository)
		eventPublisher := new(mockUsageEventPublisher)

		tenantID := uuid.New()
		tenant := createTestTenant(identity.TenantPlanBasic)
		tenant.ID = tenantID

		orderQuota := createTestQuota(billing.UsageTypeOrdersCreated, 100, billing.OveragePolicyBlock)
		reportQuota := createTestQuota(billing.UsageTypeReportsGenerated, 10, billing.OveragePolicyBlock)
		quotas := []*billing.UsageQuota{orderQuota, reportQuota}

		tenantRepo.On("FindByID", mock.Anything, tenantID).Return(tenant, nil)
		quotaRepo.On("FindAllEffectiveQuotas", mock.Anything, tenantID, "basic").Return(quotas, nil)
		usageRepo.On("SumByTenantAndType", mock.Anything, tenantID, billing.UsageTypeOrdersCreated, mock.Anything, mock.Anything).Return(int64(50), nil)
		usageRepo.On("SumByTenantAndType", mock.Anything, tenantID, billing.UsageTypeReportsGenerated, mock.Anything, mock.Anything).Return(int64(5), nil)

		service := newTestQuotaService(quotaRepo, usageRepo, tenantRepo, eventPublisher)

		status, err := service.GetQuotaStatus(context.Background(), tenantID)

		require.NoError(t, err)
		assert.Len(t, status, 2)

		orderStatus := status[billing.UsageTypeOrdersCreated]
		assert.True(t, orderStatus.Allowed)
		assert.Equal(t, int64(50), orderStatus.CurrentUsage)

		reportStatus := status[billing.UsageTypeReportsGenerated]
		assert.True(t, reportStatus.Allowed)
		assert.Equal(t, int64(5), reportStatus.CurrentUsage)
	})
}

// Tests for default amount handling

func TestCheckQuota_DefaultAmount(t *testing.T) {
	t.Run("defaults to amount 1 when not specified", func(t *testing.T) {
		quotaRepo := new(mockUsageQuotaRepository)
		usageRepo := new(mockUsageRecordRepository)
		tenantRepo := new(mockTenantRepository)
		eventPublisher := new(mockUsageEventPublisher)

		tenantID := uuid.New()
		tenant := createTestTenant(identity.TenantPlanBasic)
		tenant.ID = tenantID

		quota := createTestQuota(billing.UsageTypeOrdersCreated, 100, billing.OveragePolicyBlock)

		tenantRepo.On("FindByID", mock.Anything, tenantID).Return(tenant, nil)
		quotaRepo.On("FindEffectiveQuota", mock.Anything, tenantID, "basic", billing.UsageTypeOrdersCreated).Return(quota, nil)
		usageRepo.On("SumByTenantAndType", mock.Anything, tenantID, billing.UsageTypeOrdersCreated, mock.Anything, mock.Anything).Return(int64(50), nil)

		service := newTestQuotaService(quotaRepo, usageRepo, tenantRepo, eventPublisher)

		// Amount is 0, should default to 1
		result, err := service.CheckQuota(context.Background(), QuotaCheckInput{
			TenantID:  tenantID,
			UsageType: billing.UsageTypeOrdersCreated,
			Amount:    0,
		})

		require.NoError(t, err)
		assert.True(t, result.Allowed)
	})
}

// Tests for weekly period boundaries

func TestCalculatePeriodBoundaries_Weekly(t *testing.T) {
	service := &QuotaService{}

	t.Run("calculates weekly boundaries", func(t *testing.T) {
		start, end := service.calculatePeriodBoundaries(billing.ResetPeriodWeekly)

		assert.True(t, end.After(start))
		// Week should be 7 days
		diff := end.Sub(start)
		assert.True(t, diff >= 6*24*time.Hour && diff <= 7*24*time.Hour)
	})
}

// Tests for default period handling

func TestCalculatePeriodBoundaries_Default(t *testing.T) {
	service := &QuotaService{}

	t.Run("defaults to monthly for unknown period", func(t *testing.T) {
		start, end := service.calculatePeriodBoundaries(billing.ResetPeriod("UNKNOWN"))

		now := time.Now()
		expectedStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

		assert.Equal(t, expectedStart, start)
		assert.True(t, end.After(start))
	})
}

// Tests for error handling in GetUsageSummary

func TestGetUsageSummary_TenantNotFound(t *testing.T) {
	t.Run("returns error when tenant not found", func(t *testing.T) {
		quotaRepo := new(mockUsageQuotaRepository)
		usageRepo := new(mockUsageRecordRepository)
		tenantRepo := new(mockTenantRepository)
		eventPublisher := new(mockUsageEventPublisher)

		tenantID := uuid.New()
		tenantRepo.On("FindByID", mock.Anything, tenantID).Return(nil, shared.ErrNotFound)

		service := newTestQuotaService(quotaRepo, usageRepo, tenantRepo, eventPublisher)

		summary, err := service.GetUsageSummary(context.Background(), tenantID, billing.ResetPeriodMonthly)

		assert.Nil(t, summary)
		assert.Error(t, err)
		var domainErr *shared.DomainError
		assert.True(t, errors.As(err, &domainErr))
		assert.Equal(t, "TENANT_NOT_FOUND", domainErr.Code)
	})
}

// Tests for CheckQuota with repository errors

func TestCheckQuota_RepositoryErrors(t *testing.T) {
	t.Run("returns error when quota repo fails", func(t *testing.T) {
		quotaRepo := new(mockUsageQuotaRepository)
		usageRepo := new(mockUsageRecordRepository)
		tenantRepo := new(mockTenantRepository)
		eventPublisher := new(mockUsageEventPublisher)

		tenantID := uuid.New()
		tenant := createTestTenant(identity.TenantPlanBasic)
		tenant.ID = tenantID

		tenantRepo.On("FindByID", mock.Anything, tenantID).Return(tenant, nil)
		quotaRepo.On("FindEffectiveQuota", mock.Anything, tenantID, "basic", billing.UsageTypeOrdersCreated).Return(nil, errors.New("database error"))

		service := newTestQuotaService(quotaRepo, usageRepo, tenantRepo, eventPublisher)

		result, err := service.CheckQuota(context.Background(), QuotaCheckInput{
			TenantID:  tenantID,
			UsageType: billing.UsageTypeOrdersCreated,
			Amount:    1,
		})

		assert.Nil(t, result)
		assert.Error(t, err)
	})

	t.Run("returns error when usage repo fails", func(t *testing.T) {
		quotaRepo := new(mockUsageQuotaRepository)
		usageRepo := new(mockUsageRecordRepository)
		tenantRepo := new(mockTenantRepository)
		eventPublisher := new(mockUsageEventPublisher)

		tenantID := uuid.New()
		tenant := createTestTenant(identity.TenantPlanBasic)
		tenant.ID = tenantID

		quota := createTestQuota(billing.UsageTypeOrdersCreated, 100, billing.OveragePolicyBlock)

		tenantRepo.On("FindByID", mock.Anything, tenantID).Return(tenant, nil)
		quotaRepo.On("FindEffectiveQuota", mock.Anything, tenantID, "basic", billing.UsageTypeOrdersCreated).Return(quota, nil)
		usageRepo.On("SumByTenantAndType", mock.Anything, tenantID, billing.UsageTypeOrdersCreated, mock.Anything, mock.Anything).Return(int64(0), errors.New("database error"))

		service := newTestQuotaService(quotaRepo, usageRepo, tenantRepo, eventPublisher)

		result, err := service.CheckQuota(context.Background(), QuotaCheckInput{
			TenantID:  tenantID,
			UsageType: billing.UsageTypeOrdersCreated,
			Amount:    1,
		})

		assert.Nil(t, result)
		assert.Error(t, err)
	})

	t.Run("returns error when tenant repo fails with non-NotFound error", func(t *testing.T) {
		quotaRepo := new(mockUsageQuotaRepository)
		usageRepo := new(mockUsageRecordRepository)
		tenantRepo := new(mockTenantRepository)
		eventPublisher := new(mockUsageEventPublisher)

		tenantID := uuid.New()
		tenantRepo.On("FindByID", mock.Anything, tenantID).Return(nil, errors.New("database error"))

		service := newTestQuotaService(quotaRepo, usageRepo, tenantRepo, eventPublisher)

		result, err := service.CheckQuota(context.Background(), QuotaCheckInput{
			TenantID:  tenantID,
			UsageType: billing.UsageTypeOrdersCreated,
			Amount:    1,
		})

		assert.Nil(t, result)
		assert.Error(t, err)
	})
}

// Tests for GetUsageSummary with repository errors

func TestGetUsageSummary_RepositoryErrors(t *testing.T) {
	t.Run("returns error when quota repo fails", func(t *testing.T) {
		quotaRepo := new(mockUsageQuotaRepository)
		usageRepo := new(mockUsageRecordRepository)
		tenantRepo := new(mockTenantRepository)
		eventPublisher := new(mockUsageEventPublisher)

		tenantID := uuid.New()
		tenant := createTestTenant(identity.TenantPlanBasic)
		tenant.ID = tenantID

		tenantRepo.On("FindByID", mock.Anything, tenantID).Return(tenant, nil)
		quotaRepo.On("FindAllEffectiveQuotas", mock.Anything, tenantID, "basic").Return(nil, errors.New("database error"))

		service := newTestQuotaService(quotaRepo, usageRepo, tenantRepo, eventPublisher)

		summary, err := service.GetUsageSummary(context.Background(), tenantID, billing.ResetPeriodMonthly)

		assert.Nil(t, summary)
		assert.Error(t, err)
	})

	t.Run("returns error when tenant repo fails with non-NotFound error", func(t *testing.T) {
		quotaRepo := new(mockUsageQuotaRepository)
		usageRepo := new(mockUsageRecordRepository)
		tenantRepo := new(mockTenantRepository)
		eventPublisher := new(mockUsageEventPublisher)

		tenantID := uuid.New()
		tenantRepo.On("FindByID", mock.Anything, tenantID).Return(nil, errors.New("database error"))

		service := newTestQuotaService(quotaRepo, usageRepo, tenantRepo, eventPublisher)

		summary, err := service.GetUsageSummary(context.Background(), tenantID, billing.ResetPeriodMonthly)

		assert.Nil(t, summary)
		assert.Error(t, err)
	})
}

// Tests for unlimited quota handling

func TestCheckQuota_UnlimitedQuota(t *testing.T) {
	t.Run("allows operation when quota is unlimited", func(t *testing.T) {
		quotaRepo := new(mockUsageQuotaRepository)
		usageRepo := new(mockUsageRecordRepository)
		tenantRepo := new(mockTenantRepository)
		eventPublisher := new(mockUsageEventPublisher)

		tenantID := uuid.New()
		tenant := createTestTenant(identity.TenantPlanEnterprise)
		tenant.ID = tenantID

		// Create unlimited quota (limit = -1)
		quota, _ := billing.NewUsageQuota("enterprise", billing.UsageTypeOrdersCreated, -1, billing.ResetPeriodMonthly)

		tenantRepo.On("FindByID", mock.Anything, tenantID).Return(tenant, nil)
		quotaRepo.On("FindEffectiveQuota", mock.Anything, tenantID, "enterprise", billing.UsageTypeOrdersCreated).Return(quota, nil)
		usageRepo.On("SumByTenantAndType", mock.Anything, tenantID, billing.UsageTypeOrdersCreated, mock.Anything, mock.Anything).Return(int64(999999), nil)

		service := newTestQuotaService(quotaRepo, usageRepo, tenantRepo, eventPublisher)

		result, err := service.CheckQuota(context.Background(), QuotaCheckInput{
			TenantID:  tenantID,
			UsageType: billing.UsageTypeOrdersCreated,
			Amount:    1,
		})

		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, billing.QuotaStatusOK, result.Status)
		assert.Equal(t, int64(-1), result.Limit)
	})
}
