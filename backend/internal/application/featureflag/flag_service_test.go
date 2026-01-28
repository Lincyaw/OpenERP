package featureflag

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/erp/backend/internal/application/featureflag/dto"
	"github.com/erp/backend/internal/domain/featureflag"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockFeatureFlagRepository is a mock implementation of FeatureFlagRepository
type MockFeatureFlagRepository struct {
	mock.Mock
}

func (m *MockFeatureFlagRepository) Create(ctx context.Context, flag *featureflag.FeatureFlag) error {
	args := m.Called(ctx, flag)
	return args.Error(0)
}

func (m *MockFeatureFlagRepository) Update(ctx context.Context, flag *featureflag.FeatureFlag) error {
	args := m.Called(ctx, flag)
	return args.Error(0)
}

func (m *MockFeatureFlagRepository) FindByKey(ctx context.Context, key string) (*featureflag.FeatureFlag, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*featureflag.FeatureFlag), args.Error(1)
}

func (m *MockFeatureFlagRepository) FindByID(ctx context.Context, id uuid.UUID) (*featureflag.FeatureFlag, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*featureflag.FeatureFlag), args.Error(1)
}

func (m *MockFeatureFlagRepository) FindAll(ctx context.Context, filter shared.Filter) ([]featureflag.FeatureFlag, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]featureflag.FeatureFlag), args.Error(1)
}

func (m *MockFeatureFlagRepository) FindByStatus(ctx context.Context, status featureflag.FlagStatus, filter shared.Filter) ([]featureflag.FeatureFlag, error) {
	args := m.Called(ctx, status, filter)
	return args.Get(0).([]featureflag.FeatureFlag), args.Error(1)
}

func (m *MockFeatureFlagRepository) FindByTags(ctx context.Context, tags []string, filter shared.Filter) ([]featureflag.FeatureFlag, error) {
	args := m.Called(ctx, tags, filter)
	return args.Get(0).([]featureflag.FeatureFlag), args.Error(1)
}

func (m *MockFeatureFlagRepository) FindByType(ctx context.Context, flagType featureflag.FlagType, filter shared.Filter) ([]featureflag.FeatureFlag, error) {
	args := m.Called(ctx, flagType, filter)
	return args.Get(0).([]featureflag.FeatureFlag), args.Error(1)
}

func (m *MockFeatureFlagRepository) FindEnabled(ctx context.Context, filter shared.Filter) ([]featureflag.FeatureFlag, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]featureflag.FeatureFlag), args.Error(1)
}

func (m *MockFeatureFlagRepository) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockFeatureFlagRepository) ExistsByKey(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockFeatureFlagRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockFeatureFlagRepository) CountByStatus(ctx context.Context, status featureflag.FlagStatus) (int64, error) {
	args := m.Called(ctx, status)
	return args.Get(0).(int64), args.Error(1)
}

// MockFlagAuditLogRepository is a mock implementation of FlagAuditLogRepository
type MockFlagAuditLogRepository struct {
	mock.Mock
}

func (m *MockFlagAuditLogRepository) Create(ctx context.Context, log *featureflag.FlagAuditLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockFlagAuditLogRepository) CreateBatch(ctx context.Context, logs []*featureflag.FlagAuditLog) error {
	args := m.Called(ctx, logs)
	return args.Error(0)
}

func (m *MockFlagAuditLogRepository) FindByFlagKey(ctx context.Context, flagKey string, filter shared.Filter) ([]featureflag.FlagAuditLog, error) {
	args := m.Called(ctx, flagKey, filter)
	return args.Get(0).([]featureflag.FlagAuditLog), args.Error(1)
}

func (m *MockFlagAuditLogRepository) FindByUserID(ctx context.Context, userID uuid.UUID, filter shared.Filter) ([]featureflag.FlagAuditLog, error) {
	args := m.Called(ctx, userID, filter)
	return args.Get(0).([]featureflag.FlagAuditLog), args.Error(1)
}

func (m *MockFlagAuditLogRepository) FindByAction(ctx context.Context, action featureflag.AuditAction, filter shared.Filter) ([]featureflag.FlagAuditLog, error) {
	args := m.Called(ctx, action, filter)
	return args.Get(0).([]featureflag.FlagAuditLog), args.Error(1)
}

func (m *MockFlagAuditLogRepository) FindAll(ctx context.Context, filter shared.Filter) ([]featureflag.FlagAuditLog, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]featureflag.FlagAuditLog), args.Error(1)
}

func (m *MockFlagAuditLogRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockFlagAuditLogRepository) CountByFlagKey(ctx context.Context, flagKey string) (int64, error) {
	args := m.Called(ctx, flagKey)
	return args.Get(0).(int64), args.Error(1)
}

// MockOutboxRepository is a mock implementation of OutboxRepository
type MockOutboxRepository struct {
	mock.Mock
}

func (m *MockOutboxRepository) Save(ctx context.Context, entries ...*shared.OutboxEntry) error {
	args := m.Called(ctx, entries)
	return args.Error(0)
}

func (m *MockOutboxRepository) FindPending(ctx context.Context, limit int) ([]*shared.OutboxEntry, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]*shared.OutboxEntry), args.Error(1)
}

func (m *MockOutboxRepository) FindRetryable(ctx context.Context, before time.Time, limit int) ([]*shared.OutboxEntry, error) {
	args := m.Called(ctx, before, limit)
	return args.Get(0).([]*shared.OutboxEntry), args.Error(1)
}

func (m *MockOutboxRepository) FindDead(ctx context.Context, page, pageSize int) ([]*shared.OutboxEntry, int64, error) {
	args := m.Called(ctx, page, pageSize)
	return args.Get(0).([]*shared.OutboxEntry), args.Get(1).(int64), args.Error(2)
}

func (m *MockOutboxRepository) FindByID(ctx context.Context, id uuid.UUID) (*shared.OutboxEntry, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*shared.OutboxEntry), args.Error(1)
}

func (m *MockOutboxRepository) MarkProcessing(ctx context.Context, ids []uuid.UUID) ([]*shared.OutboxEntry, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).([]*shared.OutboxEntry), args.Error(1)
}

func (m *MockOutboxRepository) Update(ctx context.Context, entry *shared.OutboxEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockOutboxRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	args := m.Called(ctx, before)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockOutboxRepository) CountByStatus(ctx context.Context) (map[shared.OutboxStatus]int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(map[shared.OutboxStatus]int64), args.Error(1)
}

// Test helpers
func newTestLogger() *zap.Logger {
	return zap.NewNop()
}

func newTestUserID() *uuid.UUID {
	id := uuid.New()
	return &id
}

func createTestFlag(key, name string) *featureflag.FeatureFlag {
	flag, _ := featureflag.NewBooleanFlag(key, name, false, nil)
	return flag
}

// FlagService Tests
func TestFlagService_CreateFlag_Success(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	req := dto.CreateFlagRequest{
		Key:  "test-flag",
		Name: "Test Flag",
		Type: "boolean",
		DefaultValue: dto.FlagValueDTO{
			Enabled: false,
		},
		Tags: []string{"test"},
	}
	auditCtx := AuditContext{
		UserID:    newTestUserID(),
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	mockFlagRepo.On("ExistsByKey", ctx, "test-flag").Return(false, nil)
	mockFlagRepo.On("Create", ctx, mock.AnythingOfType("*featureflag.FeatureFlag")).Return(nil)
	mockAuditRepo.On("Create", ctx, mock.AnythingOfType("*featureflag.FlagAuditLog")).Return(nil)
	mockOutboxRepo.On("Save", ctx, mock.Anything).Return(nil)

	result, err := service.CreateFlag(ctx, req, auditCtx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-flag", result.Key)
	assert.Equal(t, "Test Flag", result.Name)
	assert.Equal(t, "boolean", result.Type)
	mockFlagRepo.AssertExpectations(t)
}

func TestFlagService_CreateFlag_DuplicateKey(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	req := dto.CreateFlagRequest{
		Key:  "existing-flag",
		Name: "Existing Flag",
		Type: "boolean",
	}
	auditCtx := AuditContext{}

	mockFlagRepo.On("ExistsByKey", ctx, "existing-flag").Return(true, nil)

	result, err := service.CreateFlag(ctx, req, auditCtx)

	assert.Error(t, err)
	assert.Nil(t, result)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "FLAG_EXISTS", domainErr.Code)
}

func TestFlagService_GetFlag_Success(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	flag := createTestFlag("test-flag", "Test Flag")

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(flag, nil)

	result, err := service.GetFlag(ctx, "test-flag")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-flag", result.Key)
	mockFlagRepo.AssertExpectations(t)
}

func TestFlagService_GetFlag_NotFound(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()

	mockFlagRepo.On("FindByKey", ctx, "nonexistent").Return(nil, shared.ErrNotFound)

	result, err := service.GetFlag(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, result)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "FLAG_NOT_FOUND", domainErr.Code)
}

func TestFlagService_EnableFlag_Success(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	flag := createTestFlag("test-flag", "Test Flag")
	auditCtx := AuditContext{UserID: newTestUserID()}

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(flag, nil)
	mockFlagRepo.On("Update", ctx, mock.AnythingOfType("*featureflag.FeatureFlag")).Return(nil)
	mockAuditRepo.On("Create", ctx, mock.AnythingOfType("*featureflag.FlagAuditLog")).Return(nil)
	mockOutboxRepo.On("Save", ctx, mock.Anything).Return(nil)

	err := service.EnableFlag(ctx, "test-flag", auditCtx)

	assert.NoError(t, err)
	mockFlagRepo.AssertExpectations(t)
}

func TestFlagService_DisableFlag_Success(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	flag := createTestFlag("test-flag", "Test Flag")
	// First enable the flag so we can disable it
	_ = flag.Enable(nil)
	auditCtx := AuditContext{UserID: newTestUserID()}

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(flag, nil)
	mockFlagRepo.On("Update", ctx, mock.AnythingOfType("*featureflag.FeatureFlag")).Return(nil)
	mockAuditRepo.On("Create", ctx, mock.AnythingOfType("*featureflag.FlagAuditLog")).Return(nil)
	mockOutboxRepo.On("Save", ctx, mock.Anything).Return(nil)

	err := service.DisableFlag(ctx, "test-flag", auditCtx)

	assert.NoError(t, err)
	mockFlagRepo.AssertExpectations(t)
}

func TestFlagService_ArchiveFlag_Success(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	flag := createTestFlag("test-flag", "Test Flag")
	auditCtx := AuditContext{UserID: newTestUserID()}

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(flag, nil)
	mockFlagRepo.On("Update", ctx, mock.AnythingOfType("*featureflag.FeatureFlag")).Return(nil)
	mockAuditRepo.On("Create", ctx, mock.AnythingOfType("*featureflag.FlagAuditLog")).Return(nil)
	mockOutboxRepo.On("Save", ctx, mock.Anything).Return(nil)

	err := service.ArchiveFlag(ctx, "test-flag", auditCtx)

	assert.NoError(t, err)
	mockFlagRepo.AssertExpectations(t)
}

func TestFlagService_ListFlags_Success(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	flag1 := createTestFlag("flag-1", "Flag 1")
	flag2 := createTestFlag("flag-2", "Flag 2")
	flags := []featureflag.FeatureFlag{*flag1, *flag2}

	filter := dto.FlagListFilter{
		Page:     1,
		PageSize: 20,
	}

	mockFlagRepo.On("FindAll", ctx, mock.AnythingOfType("shared.Filter")).Return(flags, nil)
	mockFlagRepo.On("Count", ctx, mock.AnythingOfType("shared.Filter")).Return(int64(2), nil)

	result, err := service.ListFlags(ctx, filter)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, len(result.Flags))
	assert.Equal(t, int64(2), result.Total)
	mockFlagRepo.AssertExpectations(t)
}

func TestFlagService_UpdateFlag_Success(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	flag := createTestFlag("test-flag", "Test Flag")
	newName := "Updated Flag"
	newDesc := "Updated description"
	auditCtx := AuditContext{UserID: newTestUserID()}

	req := dto.UpdateFlagRequest{
		Name:        &newName,
		Description: &newDesc,
	}

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(flag, nil)
	mockFlagRepo.On("Update", ctx, mock.AnythingOfType("*featureflag.FeatureFlag")).Return(nil)
	mockAuditRepo.On("Create", ctx, mock.AnythingOfType("*featureflag.FlagAuditLog")).Return(nil)
	mockOutboxRepo.On("Save", ctx, mock.Anything).Return(nil)

	result, err := service.UpdateFlag(ctx, "test-flag", req, auditCtx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Updated Flag", result.Name)
	mockFlagRepo.AssertExpectations(t)
}

func TestFlagService_CreateFlag_WithRules(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	req := dto.CreateFlagRequest{
		Key:  "flag-with-rules",
		Name: "Flag With Rules",
		Type: "boolean",
		DefaultValue: dto.FlagValueDTO{
			Enabled: false,
		},
		Rules: []dto.TargetingRuleDTO{
			{
				RuleID:   "rule-1",
				Priority: 1,
				Conditions: []dto.ConditionDTO{
					{
						Attribute: "user_role",
						Operator:  "equals",
						Values:    []string{"admin"},
					},
				},
				Value: dto.FlagValueDTO{
					Enabled: true,
				},
				Percentage: 100,
			},
		},
	}
	auditCtx := AuditContext{UserID: newTestUserID()}

	mockFlagRepo.On("ExistsByKey", ctx, "flag-with-rules").Return(false, nil)
	mockFlagRepo.On("Create", ctx, mock.AnythingOfType("*featureflag.FeatureFlag")).Return(nil)
	mockAuditRepo.On("Create", ctx, mock.AnythingOfType("*featureflag.FlagAuditLog")).Return(nil)
	mockOutboxRepo.On("Save", ctx, mock.Anything).Return(nil)

	result, err := service.CreateFlag(ctx, req, auditCtx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result.Rules))
	assert.Equal(t, "rule-1", result.Rules[0].RuleID)
	mockFlagRepo.AssertExpectations(t)
}

func TestFlagService_CreateFlag_RepositoryError(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	req := dto.CreateFlagRequest{
		Key:  "test-flag",
		Name: "Test Flag",
		Type: "boolean",
	}
	auditCtx := AuditContext{}

	mockFlagRepo.On("ExistsByKey", ctx, "test-flag").Return(false, errors.New("database error"))

	result, err := service.CreateFlag(ctx, req, auditCtx)

	assert.Error(t, err)
	assert.Nil(t, result)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "INTERNAL_ERROR", domainErr.Code)
}

func TestFlagService_CreateFlag_CreateRepositoryError(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	req := dto.CreateFlagRequest{
		Key:  "test-flag",
		Name: "Test Flag",
		Type: "boolean",
	}
	auditCtx := AuditContext{}

	mockFlagRepo.On("ExistsByKey", ctx, "test-flag").Return(false, nil)
	mockFlagRepo.On("Create", ctx, mock.AnythingOfType("*featureflag.FeatureFlag")).Return(errors.New("create failed"))

	result, err := service.CreateFlag(ctx, req, auditCtx)

	assert.Error(t, err)
	assert.Nil(t, result)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "INTERNAL_ERROR", domainErr.Code)
}

func TestFlagService_CreateFlag_InvalidType(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	req := dto.CreateFlagRequest{
		Key:  "test-flag",
		Name: "Test Flag",
		Type: "invalid-type",
	}
	auditCtx := AuditContext{}

	mockFlagRepo.On("ExistsByKey", ctx, "test-flag").Return(false, nil)

	result, err := service.CreateFlag(ctx, req, auditCtx)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestFlagService_UpdateFlag_NotFound(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	newName := "Updated Name"
	req := dto.UpdateFlagRequest{
		Name: &newName,
	}
	auditCtx := AuditContext{}

	mockFlagRepo.On("FindByKey", ctx, "nonexistent").Return(nil, shared.ErrNotFound)

	result, err := service.UpdateFlag(ctx, "nonexistent", req, auditCtx)

	assert.Error(t, err)
	assert.Nil(t, result)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "FLAG_NOT_FOUND", domainErr.Code)
}

func TestFlagService_UpdateFlag_InternalError(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	newName := "Updated Name"
	req := dto.UpdateFlagRequest{
		Name: &newName,
	}
	auditCtx := AuditContext{}

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(nil, errors.New("database error"))

	result, err := service.UpdateFlag(ctx, "test-flag", req, auditCtx)

	assert.Error(t, err)
	assert.Nil(t, result)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "INTERNAL_ERROR", domainErr.Code)
}

func TestFlagService_UpdateFlag_OptimisticLockFailed(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	flag := createTestFlag("test-flag", "Test Flag")
	flag.Version = 5

	wrongVersion := 3
	newName := "Updated Name"
	req := dto.UpdateFlagRequest{
		Name:    &newName,
		Version: &wrongVersion,
	}
	auditCtx := AuditContext{}

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(flag, nil)

	result, err := service.UpdateFlag(ctx, "test-flag", req, auditCtx)

	assert.Error(t, err)
	assert.Nil(t, result)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "OPTIMISTIC_LOCK_FAILED", domainErr.Code)
}

func TestFlagService_UpdateFlag_WithDefaultValue(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	flag := createTestFlag("test-flag", "Test Flag")
	newValue := dto.FlagValueDTO{Enabled: true}
	auditCtx := AuditContext{UserID: newTestUserID()}

	req := dto.UpdateFlagRequest{
		DefaultValue: &newValue,
	}

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(flag, nil)
	mockFlagRepo.On("Update", ctx, mock.AnythingOfType("*featureflag.FeatureFlag")).Return(nil)
	mockAuditRepo.On("Create", ctx, mock.AnythingOfType("*featureflag.FlagAuditLog")).Return(nil)
	mockOutboxRepo.On("Save", ctx, mock.Anything).Return(nil)

	result, err := service.UpdateFlag(ctx, "test-flag", req, auditCtx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.DefaultValue.Enabled)
	mockFlagRepo.AssertExpectations(t)
}

func TestFlagService_UpdateFlag_WithRules(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	flag := createTestFlag("test-flag", "Test Flag")
	auditCtx := AuditContext{UserID: newTestUserID()}

	rules := []dto.TargetingRuleDTO{
		{
			RuleID:   "rule-1",
			Priority: 1,
			Conditions: []dto.ConditionDTO{
				{
					Attribute: "user_role",
					Operator:  "equals",
					Values:    []string{"admin"},
				},
			},
			Value: dto.FlagValueDTO{
				Enabled: true,
			},
			Percentage: 100,
		},
	}
	req := dto.UpdateFlagRequest{
		Rules: &rules,
	}

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(flag, nil)
	mockFlagRepo.On("Update", ctx, mock.AnythingOfType("*featureflag.FeatureFlag")).Return(nil)
	mockAuditRepo.On("Create", ctx, mock.AnythingOfType("*featureflag.FlagAuditLog")).Return(nil)
	mockOutboxRepo.On("Save", ctx, mock.Anything).Return(nil)

	result, err := service.UpdateFlag(ctx, "test-flag", req, auditCtx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result.Rules))
	mockFlagRepo.AssertExpectations(t)
}

func TestFlagService_UpdateFlag_WithTags(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	flag := createTestFlag("test-flag", "Test Flag")
	auditCtx := AuditContext{UserID: newTestUserID()}

	tags := []string{"beta", "ui"}
	req := dto.UpdateFlagRequest{
		Tags: &tags,
	}

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(flag, nil)
	mockFlagRepo.On("Update", ctx, mock.AnythingOfType("*featureflag.FeatureFlag")).Return(nil)
	mockAuditRepo.On("Create", ctx, mock.AnythingOfType("*featureflag.FlagAuditLog")).Return(nil)
	mockOutboxRepo.On("Save", ctx, mock.Anything).Return(nil)

	result, err := service.UpdateFlag(ctx, "test-flag", req, auditCtx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.ElementsMatch(t, []string{"beta", "ui"}, result.Tags)
	mockFlagRepo.AssertExpectations(t)
}

func TestFlagService_UpdateFlag_RepositoryUpdateError(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	flag := createTestFlag("test-flag", "Test Flag")
	newName := "Updated Name"
	auditCtx := AuditContext{UserID: newTestUserID()}

	req := dto.UpdateFlagRequest{
		Name: &newName,
	}

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(flag, nil)
	mockFlagRepo.On("Update", ctx, mock.AnythingOfType("*featureflag.FeatureFlag")).Return(errors.New("update failed"))

	result, err := service.UpdateFlag(ctx, "test-flag", req, auditCtx)

	assert.Error(t, err)
	assert.Nil(t, result)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "INTERNAL_ERROR", domainErr.Code)
}

func TestFlagService_GetFlag_InternalError(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(nil, errors.New("database error"))

	result, err := service.GetFlag(ctx, "test-flag")

	assert.Error(t, err)
	assert.Nil(t, result)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "INTERNAL_ERROR", domainErr.Code)
}

func TestFlagService_ListFlags_WithStatusFilter(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	flag1 := createTestFlag("flag-1", "Flag 1")
	_ = flag1.Enable(nil)
	flags := []featureflag.FeatureFlag{*flag1}

	status := "enabled"
	filter := dto.FlagListFilter{
		Page:     1,
		PageSize: 20,
		Status:   &status,
	}

	mockFlagRepo.On("FindByStatus", ctx, featureflag.FlagStatusEnabled, mock.AnythingOfType("shared.Filter")).Return(flags, nil)
	mockFlagRepo.On("CountByStatus", ctx, featureflag.FlagStatusEnabled).Return(int64(1), nil)

	result, err := service.ListFlags(ctx, filter)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result.Flags))
	mockFlagRepo.AssertExpectations(t)
}

func TestFlagService_ListFlags_WithInvalidStatus(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()

	status := "invalid-status"
	filter := dto.FlagListFilter{
		Page:     1,
		PageSize: 20,
		Status:   &status,
	}

	result, err := service.ListFlags(ctx, filter)

	assert.Error(t, err)
	assert.Nil(t, result)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "INVALID_STATUS", domainErr.Code)
}

func TestFlagService_ListFlags_WithTypeFilter(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	flag1 := createTestFlag("flag-1", "Flag 1")
	flags := []featureflag.FeatureFlag{*flag1}

	flagType := "boolean"
	filter := dto.FlagListFilter{
		Page:     1,
		PageSize: 20,
		Type:     &flagType,
	}

	mockFlagRepo.On("FindByType", ctx, featureflag.FlagTypeBoolean, mock.AnythingOfType("shared.Filter")).Return(flags, nil)
	mockFlagRepo.On("Count", ctx, mock.AnythingOfType("shared.Filter")).Return(int64(1), nil)

	result, err := service.ListFlags(ctx, filter)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result.Flags))
	mockFlagRepo.AssertExpectations(t)
}

func TestFlagService_ListFlags_WithInvalidType(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()

	flagType := "invalid-type"
	filter := dto.FlagListFilter{
		Page:     1,
		PageSize: 20,
		Type:     &flagType,
	}

	result, err := service.ListFlags(ctx, filter)

	assert.Error(t, err)
	assert.Nil(t, result)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "INVALID_TYPE", domainErr.Code)
}

func TestFlagService_ListFlags_WithTagsFilter(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	flag1 := createTestFlag("flag-1", "Flag 1")
	flags := []featureflag.FeatureFlag{*flag1}

	filter := dto.FlagListFilter{
		Page:     1,
		PageSize: 20,
		Tags:     []string{"beta"},
	}

	mockFlagRepo.On("FindByTags", ctx, []string{"beta"}, mock.AnythingOfType("shared.Filter")).Return(flags, nil)
	mockFlagRepo.On("Count", ctx, mock.AnythingOfType("shared.Filter")).Return(int64(1), nil)

	result, err := service.ListFlags(ctx, filter)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result.Flags))
	mockFlagRepo.AssertExpectations(t)
}

func TestFlagService_ListFlags_RepositoryError(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()

	filter := dto.FlagListFilter{
		Page:     1,
		PageSize: 20,
	}

	mockFlagRepo.On("FindAll", ctx, mock.AnythingOfType("shared.Filter")).Return([]featureflag.FeatureFlag{}, errors.New("database error"))

	result, err := service.ListFlags(ctx, filter)

	assert.Error(t, err)
	assert.Nil(t, result)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "INTERNAL_ERROR", domainErr.Code)
}

func TestFlagService_EnableFlag_NotFound(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	auditCtx := AuditContext{}

	mockFlagRepo.On("FindByKey", ctx, "nonexistent").Return(nil, shared.ErrNotFound)

	err := service.EnableFlag(ctx, "nonexistent", auditCtx)

	assert.Error(t, err)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "FLAG_NOT_FOUND", domainErr.Code)
}

func TestFlagService_EnableFlag_InternalError(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	auditCtx := AuditContext{}

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(nil, errors.New("database error"))

	err := service.EnableFlag(ctx, "test-flag", auditCtx)

	assert.Error(t, err)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "INTERNAL_ERROR", domainErr.Code)
}

func TestFlagService_EnableFlag_UpdateError(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	flag := createTestFlag("test-flag", "Test Flag")
	auditCtx := AuditContext{UserID: newTestUserID()}

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(flag, nil)
	mockFlagRepo.On("Update", ctx, mock.AnythingOfType("*featureflag.FeatureFlag")).Return(errors.New("update error"))

	err := service.EnableFlag(ctx, "test-flag", auditCtx)

	assert.Error(t, err)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "INTERNAL_ERROR", domainErr.Code)
}

func TestFlagService_DisableFlag_NotFound(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	auditCtx := AuditContext{}

	mockFlagRepo.On("FindByKey", ctx, "nonexistent").Return(nil, shared.ErrNotFound)

	err := service.DisableFlag(ctx, "nonexistent", auditCtx)

	assert.Error(t, err)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "FLAG_NOT_FOUND", domainErr.Code)
}

func TestFlagService_DisableFlag_InternalError(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	auditCtx := AuditContext{}

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(nil, errors.New("database error"))

	err := service.DisableFlag(ctx, "test-flag", auditCtx)

	assert.Error(t, err)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "INTERNAL_ERROR", domainErr.Code)
}

func TestFlagService_DisableFlag_UpdateError(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	flag := createTestFlag("test-flag", "Test Flag")
	_ = flag.Enable(nil) // Enable so we can disable
	auditCtx := AuditContext{UserID: newTestUserID()}

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(flag, nil)
	mockFlagRepo.On("Update", ctx, mock.AnythingOfType("*featureflag.FeatureFlag")).Return(errors.New("update error"))

	err := service.DisableFlag(ctx, "test-flag", auditCtx)

	assert.Error(t, err)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "INTERNAL_ERROR", domainErr.Code)
}

func TestFlagService_ArchiveFlag_NotFound(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	auditCtx := AuditContext{}

	mockFlagRepo.On("FindByKey", ctx, "nonexistent").Return(nil, shared.ErrNotFound)

	err := service.ArchiveFlag(ctx, "nonexistent", auditCtx)

	assert.Error(t, err)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "FLAG_NOT_FOUND", domainErr.Code)
}

func TestFlagService_ArchiveFlag_InternalError(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	auditCtx := AuditContext{}

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(nil, errors.New("database error"))

	err := service.ArchiveFlag(ctx, "test-flag", auditCtx)

	assert.Error(t, err)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "INTERNAL_ERROR", domainErr.Code)
}

func TestFlagService_ArchiveFlag_UpdateError(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	flag := createTestFlag("test-flag", "Test Flag")
	auditCtx := AuditContext{UserID: newTestUserID()}

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(flag, nil)
	mockFlagRepo.On("Update", ctx, mock.AnythingOfType("*featureflag.FeatureFlag")).Return(errors.New("update error"))

	err := service.ArchiveFlag(ctx, "test-flag", auditCtx)

	assert.Error(t, err)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "INTERNAL_ERROR", domainErr.Code)
}

func TestFlagService_GetAuditLogs_Success(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	flag := createTestFlag("test-flag", "Test Flag")

	auditLog1, _ := featureflag.NewFlagAuditLog(
		"test-flag",
		featureflag.AuditActionCreated,
		nil,
		map[string]any{"key": "test-flag"},
		newTestUserID(),
		"127.0.0.1",
		"test-agent",
	)
	auditLogs := []featureflag.FlagAuditLog{*auditLog1}

	filter := dto.AuditLogListFilter{
		Page:     1,
		PageSize: 20,
	}

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(flag, nil)
	mockAuditRepo.On("FindByFlagKey", ctx, "test-flag", mock.AnythingOfType("shared.Filter")).Return(auditLogs, nil)
	mockAuditRepo.On("CountByFlagKey", ctx, "test-flag").Return(int64(1), nil)

	result, err := service.GetAuditLogs(ctx, "test-flag", filter)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result.AuditLogs))
	assert.Equal(t, int64(1), result.Total)
	mockFlagRepo.AssertExpectations(t)
	mockAuditRepo.AssertExpectations(t)
}

func TestFlagService_GetAuditLogs_FlagNotFound(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()

	filter := dto.AuditLogListFilter{
		Page:     1,
		PageSize: 20,
	}

	mockFlagRepo.On("FindByKey", ctx, "nonexistent").Return(nil, shared.ErrNotFound)

	result, err := service.GetAuditLogs(ctx, "nonexistent", filter)

	assert.Error(t, err)
	assert.Nil(t, result)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "FLAG_NOT_FOUND", domainErr.Code)
}

func TestFlagService_GetAuditLogs_FindFlagError(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()

	filter := dto.AuditLogListFilter{
		Page:     1,
		PageSize: 20,
	}

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(nil, errors.New("database error"))

	result, err := service.GetAuditLogs(ctx, "test-flag", filter)

	assert.Error(t, err)
	assert.Nil(t, result)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "INTERNAL_ERROR", domainErr.Code)
}

func TestFlagService_GetAuditLogs_FindAuditLogsError(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	flag := createTestFlag("test-flag", "Test Flag")

	filter := dto.AuditLogListFilter{
		Page:     1,
		PageSize: 20,
	}

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(flag, nil)
	mockAuditRepo.On("FindByFlagKey", ctx, "test-flag", mock.AnythingOfType("shared.Filter")).Return([]featureflag.FlagAuditLog{}, errors.New("audit error"))

	result, err := service.GetAuditLogs(ctx, "test-flag", filter)

	assert.Error(t, err)
	assert.Nil(t, result)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "INTERNAL_ERROR", domainErr.Code)
}

func TestFlagService_GetAuditLogs_CountError(t *testing.T) {
	mockFlagRepo := new(MockFeatureFlagRepository)
	mockAuditRepo := new(MockFlagAuditLogRepository)
	mockOutboxRepo := new(MockOutboxRepository)
	logger := newTestLogger()

	service := NewFlagService(mockFlagRepo, mockAuditRepo, mockOutboxRepo, logger)

	ctx := context.Background()
	flag := createTestFlag("test-flag", "Test Flag")

	filter := dto.AuditLogListFilter{
		Page:     1,
		PageSize: 20,
	}

	mockFlagRepo.On("FindByKey", ctx, "test-flag").Return(flag, nil)
	mockAuditRepo.On("FindByFlagKey", ctx, "test-flag", mock.AnythingOfType("shared.Filter")).Return([]featureflag.FlagAuditLog{}, nil)
	mockAuditRepo.On("CountByFlagKey", ctx, "test-flag").Return(int64(0), errors.New("count error"))

	result, err := service.GetAuditLogs(ctx, "test-flag", filter)

	assert.Error(t, err)
	assert.Nil(t, result)
	domainErr, ok := err.(*shared.DomainError)
	assert.True(t, ok)
	assert.Equal(t, "INTERNAL_ERROR", domainErr.Code)
}
