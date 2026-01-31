package billing

import (
	"context"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/billing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockUsageHistoryRepo is a mock implementation of billing.UsageHistoryRepository
type mockUsageHistoryRepo struct {
	mock.Mock
}

func (m *mockUsageHistoryRepo) Save(ctx context.Context, history *billing.UsageHistory) error {
	args := m.Called(ctx, history)
	return args.Error(0)
}

func (m *mockUsageHistoryRepo) SaveBatch(ctx context.Context, histories []*billing.UsageHistory) error {
	args := m.Called(ctx, histories)
	return args.Error(0)
}

func (m *mockUsageHistoryRepo) Upsert(ctx context.Context, history *billing.UsageHistory) error {
	args := m.Called(ctx, history)
	return args.Error(0)
}

func (m *mockUsageHistoryRepo) FindByID(ctx context.Context, id uuid.UUID) (*billing.UsageHistory, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*billing.UsageHistory), args.Error(1)
}

func (m *mockUsageHistoryRepo) FindByTenantAndDate(ctx context.Context, tenantID uuid.UUID, date time.Time) (*billing.UsageHistory, error) {
	args := m.Called(ctx, tenantID, date)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*billing.UsageHistory), args.Error(1)
}

func (m *mockUsageHistoryRepo) FindByTenant(ctx context.Context, tenantID uuid.UUID, filter billing.UsageHistoryFilter) ([]*billing.UsageHistory, error) {
	args := m.Called(ctx, tenantID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*billing.UsageHistory), args.Error(1)
}

func (m *mockUsageHistoryRepo) FindLatestByTenant(ctx context.Context, tenantID uuid.UUID) (*billing.UsageHistory, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*billing.UsageHistory), args.Error(1)
}

func (m *mockUsageHistoryRepo) CountByTenant(ctx context.Context, tenantID uuid.UUID, filter billing.UsageHistoryFilter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockUsageHistoryRepo) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	args := m.Called(ctx, before)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockUsageHistoryRepo) DeleteByTenant(ctx context.Context, tenantID uuid.UUID) error {
	args := m.Called(ctx, tenantID)
	return args.Error(0)
}

func (m *mockUsageHistoryRepo) GetAllTenantIDs(ctx context.Context) ([]uuid.UUID, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

// mockResourceCounter is a mock implementation of ResourceCounter
type mockResourceCounter struct {
	mock.Mock
}

func (m *mockResourceCounter) CountUsers(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockResourceCounter) CountProducts(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockResourceCounter) CountWarehouses(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockResourceCounter) CountCustomers(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockResourceCounter) CountSuppliers(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockResourceCounter) CountOrders(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).(int64), args.Error(1)
}

func TestUsageSnapshotService_CreateSnapshotForTenant(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	t.Run("creates snapshot with all counts", func(t *testing.T) {
		historyRepo := new(mockUsageHistoryRepo)
		resourceCounter := new(mockResourceCounter)

		tenantID := uuid.New()
		snapshotDate := time.Now()

		// Setup mocks
		resourceCounter.On("CountUsers", ctx, tenantID).Return(int64(10), nil)
		resourceCounter.On("CountProducts", ctx, tenantID).Return(int64(100), nil)
		resourceCounter.On("CountWarehouses", ctx, tenantID).Return(int64(3), nil)
		resourceCounter.On("CountCustomers", ctx, tenantID).Return(int64(50), nil)
		resourceCounter.On("CountSuppliers", ctx, tenantID).Return(int64(20), nil)
		resourceCounter.On("CountOrders", ctx, tenantID).Return(int64(500), nil)
		historyRepo.On("Upsert", ctx, mock.AnythingOfType("*billing.UsageHistory")).Return(nil)

		service := NewUsageSnapshotService(
			historyRepo,
			nil, // tenantRepo not needed for this test
			resourceCounter,
			logger,
			DefaultUsageSnapshotServiceConfig(),
		)

		history, err := service.CreateSnapshotForTenant(ctx, tenantID, snapshotDate)

		require.NoError(t, err)
		assert.NotNil(t, history)
		assert.Equal(t, tenantID, history.TenantID)
		assert.Equal(t, int64(10), history.UsersCount)
		assert.Equal(t, int64(100), history.ProductsCount)
		assert.Equal(t, int64(3), history.WarehousesCount)
		assert.Equal(t, int64(50), history.CustomersCount)
		assert.Equal(t, int64(20), history.SuppliersCount)
		assert.Equal(t, int64(500), history.OrdersCount)

		historyRepo.AssertExpectations(t)
		resourceCounter.AssertExpectations(t)
	})

	t.Run("returns error for nil tenant ID", func(t *testing.T) {
		historyRepo := new(mockUsageHistoryRepo)

		service := NewUsageSnapshotService(
			historyRepo,
			nil,
			nil,
			logger,
			DefaultUsageSnapshotServiceConfig(),
		)

		history, err := service.CreateSnapshotForTenant(ctx, uuid.Nil, time.Now())

		assert.Error(t, err)
		assert.Nil(t, history)
	})
}

func TestUsageSnapshotService_CleanupOldSnapshots(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	t.Run("deletes old snapshots based on retention days", func(t *testing.T) {
		historyRepo := new(mockUsageHistoryRepo)

		config := UsageSnapshotServiceConfig{
			RetentionDays: 90,
		}

		historyRepo.On("DeleteOlderThan", ctx, mock.AnythingOfType("time.Time")).Return(int64(50), nil)

		service := NewUsageSnapshotService(
			historyRepo,
			nil,
			nil,
			logger,
			config,
		)

		deleted, err := service.CleanupOldSnapshots(ctx)

		require.NoError(t, err)
		assert.Equal(t, int64(50), deleted)
		historyRepo.AssertExpectations(t)
	})
}

func TestUsageSnapshotService_GetUsageHistory(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	t.Run("returns usage history for tenant", func(t *testing.T) {
		historyRepo := new(mockUsageHistoryRepo)

		tenantID := uuid.New()
		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

		expectedHistories := []*billing.UsageHistory{
			{ID: uuid.New(), TenantID: tenantID},
			{ID: uuid.New(), TenantID: tenantID},
		}

		historyRepo.On("FindByTenant", ctx, tenantID, mock.AnythingOfType("billing.UsageHistoryFilter")).
			Return(expectedHistories, nil)

		service := NewUsageSnapshotService(
			historyRepo,
			nil,
			nil,
			logger,
			DefaultUsageSnapshotServiceConfig(),
		)

		histories, err := service.GetUsageHistory(ctx, tenantID, startDate, endDate)

		require.NoError(t, err)
		assert.Len(t, histories, 2)
		historyRepo.AssertExpectations(t)
	})

	t.Run("returns error for nil tenant ID", func(t *testing.T) {
		historyRepo := new(mockUsageHistoryRepo)

		service := NewUsageSnapshotService(
			historyRepo,
			nil,
			nil,
			logger,
			DefaultUsageSnapshotServiceConfig(),
		)

		histories, err := service.GetUsageHistory(ctx, uuid.Nil, time.Now(), time.Now())

		assert.Error(t, err)
		assert.Nil(t, histories)
	})
}

func TestDefaultUsageSnapshotServiceConfig(t *testing.T) {
	config := DefaultUsageSnapshotServiceConfig()

	assert.Equal(t, 90, config.RetentionDays)
}
