package featureflag

import (
	"context"
	"testing"

	"github.com/erp/backend/internal/application/featureflag/dto"
	"github.com/erp/backend/internal/domain/featureflag"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockFlagOverrideRepository is a mock implementation of FlagOverrideRepository
type MockFlagOverrideRepository struct {
	mock.Mock
}

func (m *MockFlagOverrideRepository) Create(ctx context.Context, override *featureflag.FlagOverride) error {
	args := m.Called(ctx, override)
	return args.Error(0)
}

func (m *MockFlagOverrideRepository) Update(ctx context.Context, override *featureflag.FlagOverride) error {
	args := m.Called(ctx, override)
	return args.Error(0)
}

func (m *MockFlagOverrideRepository) FindByID(ctx context.Context, id uuid.UUID) (*featureflag.FlagOverride, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*featureflag.FlagOverride), args.Error(1)
}

func (m *MockFlagOverrideRepository) FindByFlagKey(ctx context.Context, flagKey string, filter shared.Filter) ([]featureflag.FlagOverride, error) {
	args := m.Called(ctx, flagKey, filter)
	return args.Get(0).([]featureflag.FlagOverride), args.Error(1)
}

func (m *MockFlagOverrideRepository) FindByTarget(ctx context.Context, targetType featureflag.OverrideTargetType, targetID uuid.UUID, filter shared.Filter) ([]featureflag.FlagOverride, error) {
	args := m.Called(ctx, targetType, targetID, filter)
	return args.Get(0).([]featureflag.FlagOverride), args.Error(1)
}

func (m *MockFlagOverrideRepository) FindForEvaluation(ctx context.Context, flagKey string, tenantID, userID *uuid.UUID) (*featureflag.FlagOverride, error) {
	args := m.Called(ctx, flagKey, tenantID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*featureflag.FlagOverride), args.Error(1)
}

func (m *MockFlagOverrideRepository) FindByFlagKeyAndTarget(ctx context.Context, flagKey string, targetType featureflag.OverrideTargetType, targetID uuid.UUID) (*featureflag.FlagOverride, error) {
	args := m.Called(ctx, flagKey, targetType, targetID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*featureflag.FlagOverride), args.Error(1)
}

func (m *MockFlagOverrideRepository) FindExpired(ctx context.Context, filter shared.Filter) ([]featureflag.FlagOverride, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]featureflag.FlagOverride), args.Error(1)
}

func (m *MockFlagOverrideRepository) FindActive(ctx context.Context, filter shared.Filter) ([]featureflag.FlagOverride, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]featureflag.FlagOverride), args.Error(1)
}

func (m *MockFlagOverrideRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFlagOverrideRepository) DeleteByFlagKey(ctx context.Context, flagKey string) (int64, error) {
	args := m.Called(ctx, flagKey)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockFlagOverrideRepository) DeleteExpired(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockFlagOverrideRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockFlagOverrideRepository) CountByFlagKey(ctx context.Context, flagKey string) (int64, error) {
	args := m.Called(ctx, flagKey)
	return args.Get(0).(int64), args.Error(1)
}

// Evaluation Service Tests

func TestEvaluationService_Evaluate_Success(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockOverrideRepo := new(MockFlagOverrideRepository)
	logger := newTestLogger()

	service := NewEvaluationService(mockFlagRepo, mockOverrideRepo, logger)

	ctx := context.Background()
	flag := createTestFlag("test-flag", "Test Flag")
	// Enable the flag
	_ = flag.Enable(nil)

	evalCtx := dto.EvaluationContextDTO{
		UserID:   "user-123",
		TenantID: "tenant-456",
	}

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(flag, nil)
	mockOverrideRepo.On("FindByFlagKeyAndTarget", ctx, "test-flag", featureflag.OverrideTargetTypeUser, mock.AnythingOfType("uuid.UUID")).Return(nil, shared.ErrNotFound)
	mockOverrideRepo.On("FindByFlagKeyAndTarget", ctx, "test-flag", featureflag.OverrideTargetTypeTenant, mock.AnythingOfType("uuid.UUID")).Return(nil, shared.ErrNotFound)

	result, err := service.Evaluate(ctx, "test-flag", evalCtx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-flag", result.Key)
	assert.False(t, result.Enabled) // Default value is false
}

func TestEvaluationService_Evaluate_FlagNotFound(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockOverrideRepo := new(MockFlagOverrideRepository)
	logger := newTestLogger()

	service := NewEvaluationService(mockFlagRepo, mockOverrideRepo, logger)

	ctx := context.Background()
	evalCtx := dto.EvaluationContextDTO{}

	// Create a domain error for NOT_FOUND
	notFoundErr := shared.NewDomainError("NOT_FOUND", "Flag not found")
	mockFlagRepo.On("FindByKey", ctx, "nonexistent").Return(nil, notFoundErr)

	result, err := service.Evaluate(ctx, "nonexistent", evalCtx)

	assert.Error(t, err)
	assert.Nil(t, result)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "FLAG_NOT_FOUND", domainErr.Code)
}

func TestEvaluationService_BatchEvaluate_Success(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockOverrideRepo := new(MockFlagOverrideRepository)
	logger := newTestLogger()

	service := NewEvaluationService(mockFlagRepo, mockOverrideRepo, logger)

	ctx := context.Background()
	flag1 := createTestFlag("flag-1", "Flag 1")
	_ = flag1.Enable(nil)
	flag2 := createTestFlag("flag-2", "Flag 2")
	_ = flag2.Enable(nil)

	evalCtx := dto.EvaluationContextDTO{
		UserID: "user-123",
	}

	mockFlagRepo.On("FindByKey", ctx, "flag-1").Return(flag1, nil)
	mockFlagRepo.On("FindByKey", ctx, "flag-2").Return(flag2, nil)
	mockOverrideRepo.On("FindByFlagKeyAndTarget", ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil, shared.ErrNotFound)

	result, err := service.BatchEvaluate(ctx, []string{"flag-1", "flag-2"}, evalCtx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, len(result.Results))
	assert.Contains(t, result.Results, "flag-1")
	assert.Contains(t, result.Results, "flag-2")
}

func TestEvaluationService_BatchEvaluate_EmptyKeys(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockOverrideRepo := new(MockFlagOverrideRepository)
	logger := newTestLogger()

	service := NewEvaluationService(mockFlagRepo, mockOverrideRepo, logger)

	ctx := context.Background()
	evalCtx := dto.EvaluationContextDTO{}

	result, err := service.BatchEvaluate(ctx, []string{}, evalCtx)

	assert.Error(t, err)
	assert.Nil(t, result)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "INVALID_REQUEST", domainErr.Code)
}

func TestEvaluationService_BatchEvaluate_TooManyKeys(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockOverrideRepo := new(MockFlagOverrideRepository)
	logger := newTestLogger()

	service := NewEvaluationService(mockFlagRepo, mockOverrideRepo, logger)

	ctx := context.Background()
	evalCtx := dto.EvaluationContextDTO{}

	// Create 101 keys
	keys := make([]string, 101)
	for i := range keys {
		keys[i] = "flag-" + string(rune(i))
	}

	result, err := service.BatchEvaluate(ctx, keys, evalCtx)

	assert.Error(t, err)
	assert.Nil(t, result)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "INVALID_REQUEST", domainErr.Code)
}

func TestEvaluationService_GetClientConfig_Success(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockOverrideRepo := new(MockFlagOverrideRepository)
	logger := newTestLogger()

	service := NewEvaluationService(mockFlagRepo, mockOverrideRepo, logger)

	ctx := context.Background()
	flag1 := createTestFlag("flag-1", "Flag 1")
	_ = flag1.Enable(nil)
	flag2 := createTestFlag("flag-2", "Flag 2")
	_ = flag2.Enable(nil)

	flags := []featureflag.FeatureFlag{*flag1, *flag2}

	evalCtx := dto.EvaluationContextDTO{
		UserID: "user-123",
	}

	mockFlagRepo.On("FindEnabled", ctx, mock.AnythingOfType("shared.Filter")).Return(flags, nil)
	mockOverrideRepo.On("FindByFlagKeyAndTarget", ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil, shared.ErrNotFound)

	result, err := service.GetClientConfig(ctx, evalCtx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, len(result.Flags))
	assert.Contains(t, result.Flags, "flag-1")
	assert.Contains(t, result.Flags, "flag-2")
}

func TestEvaluationService_IsEnabled_Success(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockOverrideRepo := new(MockFlagOverrideRepository)
	logger := newTestLogger()

	service := NewEvaluationService(mockFlagRepo, mockOverrideRepo, logger)

	ctx := context.Background()
	flag := createTestFlag("test-flag", "Test Flag")
	// Enable the flag and set default value to enabled
	_ = flag.Enable(nil)
	_ = flag.SetDefault(featureflag.NewBooleanFlagValue(true), nil)

	evalCtx := dto.EvaluationContextDTO{
		UserID: "user-123",
	}

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(flag, nil)
	mockOverrideRepo.On("FindByFlagKeyAndTarget", ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil, shared.ErrNotFound)

	result, err := service.IsEnabled(ctx, "test-flag", evalCtx)

	assert.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluationService_GetVariant_Success(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockOverrideRepo := new(MockFlagOverrideRepository)
	logger := newTestLogger()

	service := NewEvaluationService(mockFlagRepo, mockOverrideRepo, logger)

	ctx := context.Background()
	flag, _ := featureflag.NewVariantFlag("variant-flag", "Variant Flag", "control", nil)
	_ = flag.Enable(nil)

	evalCtx := dto.EvaluationContextDTO{
		UserID: "user-123",
	}

	mockFlagRepo.On("FindByKey", ctx, "variant-flag").Return(flag, nil)
	mockOverrideRepo.On("FindByFlagKeyAndTarget", ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil, shared.ErrNotFound)

	result, err := service.GetVariant(ctx, "variant-flag", evalCtx)

	assert.NoError(t, err)
	assert.Equal(t, "control", result)
}

func TestEvaluationService_Evaluate_DisabledFlag(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockOverrideRepo := new(MockFlagOverrideRepository)
	logger := newTestLogger()

	service := NewEvaluationService(mockFlagRepo, mockOverrideRepo, logger)

	ctx := context.Background()
	flag := createTestFlag("disabled-flag", "Disabled Flag")
	// Flag is disabled by default

	evalCtx := dto.EvaluationContextDTO{
		UserID: "user-123",
	}

	mockFlagRepo.On("FindByKey", ctx, "disabled-flag").Return(flag, nil)
	mockOverrideRepo.On("FindByFlagKeyAndTarget", ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil, shared.ErrNotFound)

	result, err := service.Evaluate(ctx, "disabled-flag", evalCtx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Enabled)
	assert.Equal(t, "disabled", result.Reason)
}
