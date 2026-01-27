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

// Override Service Tests

func TestOverrideService_CreateOverride_Success(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockOverrideRepo := new(MockFlagOverrideRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewOverrideService(mockFlagRepo, mockOverrideRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	flag := createTestFlag("test-flag", "Test Flag")
	_ = flag.Enable(nil) // Ensure flag is not archived
	targetID := uuid.New()

	req := dto.CreateOverrideRequest{
		TargetType: "user",
		TargetID:   targetID,
		Value: dto.FlagValueDTO{
			Enabled: true,
		},
		Reason: "Testing override",
	}
	auditCtx := AuditContext{UserID: newTestUserID()}

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(flag, nil)
	mockOverrideRepo.On("FindByFlagKeyAndTarget", ctx, "test-flag", featureflag.OverrideTargetTypeUser, targetID).Return(nil, shared.ErrNotFound)
	mockOverrideRepo.On("Create", ctx, mock.AnythingOfType("*featureflag.FlagOverride")).Return(nil)
	mockAuditRepo.On("Create", ctx, mock.AnythingOfType("*featureflag.FlagAuditLog")).Return(nil)
	mockOutboxRepo.On("Save", ctx, mock.Anything).Return(nil)

	result, err := service.CreateOverride(ctx, "test-flag", req, auditCtx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-flag", result.FlagKey)
	assert.Equal(t, "user", result.TargetType)
	assert.Equal(t, targetID, result.TargetID)
	assert.True(t, result.Value.Enabled)
	mockFlagRepo.AssertExpectations(t)
	mockOverrideRepo.AssertExpectations(t)
}

func TestOverrideService_CreateOverride_FlagNotFound(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockOverrideRepo := new(MockFlagOverrideRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewOverrideService(mockFlagRepo, mockOverrideRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	targetID := uuid.New()

	req := dto.CreateOverrideRequest{
		TargetType: "user",
		TargetID:   targetID,
		Value:      dto.FlagValueDTO{Enabled: true},
	}
	auditCtx := AuditContext{}

	mockFlagRepo.On("FindByKey", ctx, "nonexistent").Return(nil, shared.ErrNotFound)

	result, err := service.CreateOverride(ctx, "nonexistent", req, auditCtx)

	assert.Error(t, err)
	assert.Nil(t, result)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "FLAG_NOT_FOUND", domainErr.Code)
}

func TestOverrideService_CreateOverride_FlagArchived(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockOverrideRepo := new(MockFlagOverrideRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewOverrideService(mockFlagRepo, mockOverrideRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	flag := createTestFlag("archived-flag", "Archived Flag")
	_ = flag.Archive(nil) // Archive the flag
	targetID := uuid.New()

	req := dto.CreateOverrideRequest{
		TargetType: "user",
		TargetID:   targetID,
		Value:      dto.FlagValueDTO{Enabled: true},
	}
	auditCtx := AuditContext{}

	mockFlagRepo.On("FindByKey", ctx, "archived-flag").Return(flag, nil)

	result, err := service.CreateOverride(ctx, "archived-flag", req, auditCtx)

	assert.Error(t, err)
	assert.Nil(t, result)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "FLAG_ARCHIVED", domainErr.Code)
}

func TestOverrideService_CreateOverride_DuplicateOverride(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockOverrideRepo := new(MockFlagOverrideRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewOverrideService(mockFlagRepo, mockOverrideRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	flag := createTestFlag("test-flag", "Test Flag")
	_ = flag.Enable(nil)
	targetID := uuid.New()

	existingOverride, _ := featureflag.NewFlagOverride(
		"test-flag",
		featureflag.OverrideTargetTypeUser,
		targetID,
		featureflag.NewBooleanFlagValue(true),
		"existing",
		nil,
		nil,
	)

	req := dto.CreateOverrideRequest{
		TargetType: "user",
		TargetID:   targetID,
		Value:      dto.FlagValueDTO{Enabled: true},
	}
	auditCtx := AuditContext{}

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(flag, nil)
	mockOverrideRepo.On("FindByFlagKeyAndTarget", ctx, "test-flag", featureflag.OverrideTargetTypeUser, targetID).Return(existingOverride, nil)

	result, err := service.CreateOverride(ctx, "test-flag", req, auditCtx)

	assert.Error(t, err)
	assert.Nil(t, result)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "OVERRIDE_EXISTS", domainErr.Code)
}

func TestOverrideService_ListOverrides_Success(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockOverrideRepo := new(MockFlagOverrideRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewOverrideService(mockFlagRepo, mockOverrideRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	flag := createTestFlag("test-flag", "Test Flag")

	override1, _ := featureflag.NewFlagOverride(
		"test-flag",
		featureflag.OverrideTargetTypeUser,
		uuid.New(),
		featureflag.NewBooleanFlagValue(true),
		"reason1",
		nil,
		nil,
	)
	override2, _ := featureflag.NewFlagOverride(
		"test-flag",
		featureflag.OverrideTargetTypeTenant,
		uuid.New(),
		featureflag.NewBooleanFlagValue(false),
		"reason2",
		nil,
		nil,
	)
	overrides := []featureflag.FlagOverride{*override1, *override2}

	filter := dto.OverrideListFilter{
		Page:     1,
		PageSize: 20,
	}

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(flag, nil)
	mockOverrideRepo.On("FindByFlagKey", ctx, "test-flag", mock.AnythingOfType("shared.Filter")).Return(overrides, nil)
	mockOverrideRepo.On("CountByFlagKey", ctx, "test-flag").Return(int64(2), nil)

	result, err := service.ListOverrides(ctx, "test-flag", filter)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, len(result.Overrides))
	assert.Equal(t, int64(2), result.Total)
	mockOverrideRepo.AssertExpectations(t)
}

func TestOverrideService_GetOverride_Success(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockOverrideRepo := new(MockFlagOverrideRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewOverrideService(mockFlagRepo, mockOverrideRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	overrideID := uuid.New()
	override, _ := featureflag.NewFlagOverride(
		"test-flag",
		featureflag.OverrideTargetTypeUser,
		uuid.New(),
		featureflag.NewBooleanFlagValue(true),
		"test reason",
		nil,
		nil,
	)

	mockOverrideRepo.On("FindByID", ctx, overrideID).Return(override, nil)

	result, err := service.GetOverride(ctx, overrideID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-flag", result.FlagKey)
	mockOverrideRepo.AssertExpectations(t)
}

func TestOverrideService_GetOverride_NotFound(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockOverrideRepo := new(MockFlagOverrideRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewOverrideService(mockFlagRepo, mockOverrideRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	overrideID := uuid.New()

	mockOverrideRepo.On("FindByID", ctx, overrideID).Return(nil, shared.ErrNotFound)

	result, err := service.GetOverride(ctx, overrideID)

	assert.Error(t, err)
	assert.Nil(t, result)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "OVERRIDE_NOT_FOUND", domainErr.Code)
}

func TestOverrideService_DeleteOverride_Success(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockOverrideRepo := new(MockFlagOverrideRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewOverrideService(mockFlagRepo, mockOverrideRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	overrideID := uuid.New()
	override, _ := featureflag.NewFlagOverride(
		"test-flag",
		featureflag.OverrideTargetTypeUser,
		uuid.New(),
		featureflag.NewBooleanFlagValue(true),
		"test reason",
		nil,
		nil,
	)
	flag := createTestFlag("test-flag", "Test Flag")
	auditCtx := AuditContext{UserID: newTestUserID()}

	mockOverrideRepo.On("FindByID", ctx, overrideID).Return(override, nil)
	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(flag, nil)
	mockOverrideRepo.On("Delete", ctx, overrideID).Return(nil)
	mockAuditRepo.On("Create", ctx, mock.AnythingOfType("*featureflag.FlagAuditLog")).Return(nil)
	mockOutboxRepo.On("Save", ctx, mock.Anything).Return(nil)

	err := service.DeleteOverride(ctx, overrideID, auditCtx)

	assert.NoError(t, err)
	mockOverrideRepo.AssertExpectations(t)
}

func TestOverrideService_DeleteOverride_NotFound(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockOverrideRepo := new(MockFlagOverrideRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewOverrideService(mockFlagRepo, mockOverrideRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	overrideID := uuid.New()
	auditCtx := AuditContext{}

	mockOverrideRepo.On("FindByID", ctx, overrideID).Return(nil, shared.ErrNotFound)

	err := service.DeleteOverride(ctx, overrideID, auditCtx)

	assert.Error(t, err)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "OVERRIDE_NOT_FOUND", domainErr.Code)
}

func TestOverrideService_CleanupExpiredOverrides_Success(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockOverrideRepo := new(MockFlagOverrideRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewOverrideService(mockFlagRepo, mockOverrideRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()

	mockOverrideRepo.On("DeleteExpired", ctx).Return(int64(5), nil)

	count, err := service.CleanupExpiredOverrides(ctx)

	assert.NoError(t, err)
	assert.Equal(t, int64(5), count)
	mockOverrideRepo.AssertExpectations(t)
}

func TestOverrideService_CreateOverride_TenantTarget(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockOverrideRepo := new(MockFlagOverrideRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewOverrideService(mockFlagRepo, mockOverrideRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	flag := createTestFlag("test-flag", "Test Flag")
	_ = flag.Enable(nil)
	targetID := uuid.New()

	req := dto.CreateOverrideRequest{
		TargetType: "tenant",
		TargetID:   targetID,
		Value: dto.FlagValueDTO{
			Enabled: false,
		},
		Reason: "Tenant-specific override",
	}
	auditCtx := AuditContext{UserID: newTestUserID()}

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(flag, nil)
	mockOverrideRepo.On("FindByFlagKeyAndTarget", ctx, "test-flag", featureflag.OverrideTargetTypeTenant, targetID).Return(nil, shared.ErrNotFound)
	mockOverrideRepo.On("Create", ctx, mock.AnythingOfType("*featureflag.FlagOverride")).Return(nil)
	mockAuditRepo.On("Create", ctx, mock.AnythingOfType("*featureflag.FlagAuditLog")).Return(nil)
	mockOutboxRepo.On("Save", ctx, mock.Anything).Return(nil)

	result, err := service.CreateOverride(ctx, "test-flag", req, auditCtx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "tenant", result.TargetType)
	assert.False(t, result.Value.Enabled)
	mockFlagRepo.AssertExpectations(t)
	mockOverrideRepo.AssertExpectations(t)
}
