package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	featureflagapp "github.com/erp/backend/internal/application/featureflag"
	"github.com/erp/backend/internal/application/featureflag/dto"
	"github.com/erp/backend/internal/domain/featureflag"
	"github.com/erp/backend/internal/domain/shared"
	httpdto "github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ============================================================================
// Mock Repositories
// ============================================================================

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
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
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
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
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
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
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
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
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

// ============================================================================
// Test Helpers
// ============================================================================

func newTestLogger() *zap.Logger {
	return zap.NewNop()
}

func newTestUserID() *uuid.UUID {
	id := uuid.New()
	return &id
}

func createTestFeatureFlag(key, name string) *featureflag.FeatureFlag {
	flag, _ := featureflag.NewBooleanFlag(key, name, false, nil)
	return flag
}

func setupFeatureFlagRouter(handler *FeatureFlagHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Feature flag routes
	ffGroup := r.Group("/api/v1/feature-flags")
	{
		ffGroup.GET("", handler.ListFlags)
		ffGroup.POST("", handler.CreateFlag)
		ffGroup.GET("/:key", handler.GetFlag)
		ffGroup.PUT("/:key", handler.UpdateFlag)
		ffGroup.DELETE("/:key", handler.ArchiveFlag)
		ffGroup.POST("/:key/enable", handler.EnableFlag)
		ffGroup.POST("/:key/disable", handler.DisableFlag)
		ffGroup.POST("/:key/evaluate", handler.EvaluateFlag)
		ffGroup.GET("/:key/overrides", handler.ListOverrides)
		ffGroup.POST("/:key/overrides", handler.CreateOverride)
		ffGroup.DELETE("/:key/overrides/:id", handler.DeleteOverride)
		ffGroup.GET("/:key/audit-logs", handler.GetAuditLogs)
	}
	ffGroup.POST("/evaluate-batch", handler.BatchEvaluate)
	ffGroup.POST("/client-config", handler.GetClientConfig)

	return r
}

func createFeatureFlagHandler(
	flagRepo *MockFeatureFlagRepository,
	overrideRepo *MockFlagOverrideRepository,
	auditRepo *MockFlagAuditLogRepository,
	outboxRepo *MockOutboxRepository,
) *FeatureFlagHandler {
	logger := newTestLogger()

	flagService := featureflagapp.NewFlagService(flagRepo, auditRepo, outboxRepo, logger)
	evaluationService := featureflagapp.NewEvaluationService(flagRepo, overrideRepo, logger)
	overrideService := featureflagapp.NewOverrideService(flagRepo, overrideRepo, auditRepo, outboxRepo, logger)

	return NewFeatureFlagHandler(flagService, evaluationService, overrideService)
}

// ============================================================================
// Flag CRUD Tests
// ============================================================================

func TestFeatureFlagHandler_CreateFlag_Success(t *testing.T) {
	flagRepo := new(MockFeatureFlagRepository)
	overrideRepo := new(MockFlagOverrideRepository)
	auditRepo := new(MockFlagAuditLogRepository)
	outboxRepo := new(MockOutboxRepository)

	handler := createFeatureFlagHandler(flagRepo, overrideRepo, auditRepo, outboxRepo)
	router := setupFeatureFlagRouter(handler)

	flagRepo.On("ExistsByKey", mock.Anything, "new-feature").Return(false, nil)
	flagRepo.On("Create", mock.Anything, mock.AnythingOfType("*featureflag.FeatureFlag")).Return(nil)
	auditRepo.On("Create", mock.Anything, mock.AnythingOfType("*featureflag.FlagAuditLog")).Return(nil)
	outboxRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

	reqBody := CreateFlagHTTPRequest{
		Key:         "new-feature",
		Name:        "New Feature",
		Description: "A new feature flag",
		Type:        "boolean",
		DefaultValue: dto.FlagValueDTO{
			Enabled: false,
		},
		Tags: []string{"test", "feature"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/feature-flags", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp httpdto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	flagRepo.AssertExpectations(t)
}

func TestFeatureFlagHandler_CreateFlag_DuplicateKey(t *testing.T) {
	flagRepo := new(MockFeatureFlagRepository)
	overrideRepo := new(MockFlagOverrideRepository)
	auditRepo := new(MockFlagAuditLogRepository)
	outboxRepo := new(MockOutboxRepository)

	handler := createFeatureFlagHandler(flagRepo, overrideRepo, auditRepo, outboxRepo)
	router := setupFeatureFlagRouter(handler)

	flagRepo.On("ExistsByKey", mock.Anything, "existing-flag").Return(true, nil)

	reqBody := CreateFlagHTTPRequest{
		Key:  "existing-flag",
		Name: "Existing Flag",
		Type: "boolean",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/feature-flags", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var resp httpdto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, httpdto.ErrCodeAlreadyExists, resp.Error.Code)

	flagRepo.AssertExpectations(t)
}

func TestFeatureFlagHandler_ListFlags_Success(t *testing.T) {
	flagRepo := new(MockFeatureFlagRepository)
	overrideRepo := new(MockFlagOverrideRepository)
	auditRepo := new(MockFlagAuditLogRepository)
	outboxRepo := new(MockOutboxRepository)

	handler := createFeatureFlagHandler(flagRepo, overrideRepo, auditRepo, outboxRepo)
	router := setupFeatureFlagRouter(handler)

	flag1 := createTestFeatureFlag("flag-1", "Flag 1")
	flag2 := createTestFeatureFlag("flag-2", "Flag 2")
	flags := []featureflag.FeatureFlag{*flag1, *flag2}

	flagRepo.On("FindAll", mock.Anything, mock.AnythingOfType("shared.Filter")).Return(flags, nil)
	flagRepo.On("Count", mock.Anything, mock.AnythingOfType("shared.Filter")).Return(int64(2), nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/feature-flags?page=1&page_size=20", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp httpdto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	flagRepo.AssertExpectations(t)
}

func TestFeatureFlagHandler_GetFlag_Success(t *testing.T) {
	flagRepo := new(MockFeatureFlagRepository)
	overrideRepo := new(MockFlagOverrideRepository)
	auditRepo := new(MockFlagAuditLogRepository)
	outboxRepo := new(MockOutboxRepository)

	handler := createFeatureFlagHandler(flagRepo, overrideRepo, auditRepo, outboxRepo)
	router := setupFeatureFlagRouter(handler)

	flag := createTestFeatureFlag("test-flag", "Test Flag")
	flagRepo.On("FindByKey", mock.Anything, "test-flag").Return(flag, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/feature-flags/test-flag", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp httpdto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	flagRepo.AssertExpectations(t)
}

func TestFeatureFlagHandler_GetFlag_NotFound(t *testing.T) {
	flagRepo := new(MockFeatureFlagRepository)
	overrideRepo := new(MockFlagOverrideRepository)
	auditRepo := new(MockFlagAuditLogRepository)
	outboxRepo := new(MockOutboxRepository)

	handler := createFeatureFlagHandler(flagRepo, overrideRepo, auditRepo, outboxRepo)
	router := setupFeatureFlagRouter(handler)

	flagRepo.On("FindByKey", mock.Anything, "nonexistent").Return(nil, shared.ErrNotFound)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/feature-flags/nonexistent", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp httpdto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, httpdto.ErrCodeNotFound, resp.Error.Code)

	flagRepo.AssertExpectations(t)
}

func TestFeatureFlagHandler_UpdateFlag_Success(t *testing.T) {
	flagRepo := new(MockFeatureFlagRepository)
	overrideRepo := new(MockFlagOverrideRepository)
	auditRepo := new(MockFlagAuditLogRepository)
	outboxRepo := new(MockOutboxRepository)

	handler := createFeatureFlagHandler(flagRepo, overrideRepo, auditRepo, outboxRepo)
	router := setupFeatureFlagRouter(handler)

	flag := createTestFeatureFlag("test-flag", "Test Flag")
	flagRepo.On("FindByKey", mock.Anything, "test-flag").Return(flag, nil)
	flagRepo.On("Update", mock.Anything, mock.AnythingOfType("*featureflag.FeatureFlag")).Return(nil)
	auditRepo.On("Create", mock.Anything, mock.AnythingOfType("*featureflag.FlagAuditLog")).Return(nil)
	outboxRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

	newName := "Updated Name"
	reqBody := UpdateFlagHTTPRequest{
		Name: &newName,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/feature-flags/test-flag", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp httpdto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	flagRepo.AssertExpectations(t)
}

func TestFeatureFlagHandler_UpdateFlag_VersionConflict(t *testing.T) {
	flagRepo := new(MockFeatureFlagRepository)
	overrideRepo := new(MockFlagOverrideRepository)
	auditRepo := new(MockFlagAuditLogRepository)
	outboxRepo := new(MockOutboxRepository)

	handler := createFeatureFlagHandler(flagRepo, overrideRepo, auditRepo, outboxRepo)
	router := setupFeatureFlagRouter(handler)

	flag := createTestFeatureFlag("test-flag", "Test Flag")
	// Flag has version 1 by default
	flagRepo.On("FindByKey", mock.Anything, "test-flag").Return(flag, nil)

	newName := "Updated Name"
	wrongVersion := 99 // Wrong version to trigger conflict
	reqBody := map[string]interface{}{
		"name":    newName,
		"version": wrongVersion,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/feature-flags/test-flag", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var resp httpdto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, httpdto.ErrCodeConcurrencyConflict, resp.Error.Code)

	flagRepo.AssertExpectations(t)
}

func TestFeatureFlagHandler_ArchiveFlag_Success(t *testing.T) {
	flagRepo := new(MockFeatureFlagRepository)
	overrideRepo := new(MockFlagOverrideRepository)
	auditRepo := new(MockFlagAuditLogRepository)
	outboxRepo := new(MockOutboxRepository)

	handler := createFeatureFlagHandler(flagRepo, overrideRepo, auditRepo, outboxRepo)
	router := setupFeatureFlagRouter(handler)

	flag := createTestFeatureFlag("test-flag", "Test Flag")
	flagRepo.On("FindByKey", mock.Anything, "test-flag").Return(flag, nil)
	flagRepo.On("Update", mock.Anything, mock.AnythingOfType("*featureflag.FeatureFlag")).Return(nil)
	auditRepo.On("Create", mock.Anything, mock.AnythingOfType("*featureflag.FlagAuditLog")).Return(nil)
	outboxRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/feature-flags/test-flag", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	flagRepo.AssertExpectations(t)
}

// ============================================================================
// Flag State Operation Tests
// ============================================================================

func TestFeatureFlagHandler_EnableFlag_Success(t *testing.T) {
	flagRepo := new(MockFeatureFlagRepository)
	overrideRepo := new(MockFlagOverrideRepository)
	auditRepo := new(MockFlagAuditLogRepository)
	outboxRepo := new(MockOutboxRepository)

	handler := createFeatureFlagHandler(flagRepo, overrideRepo, auditRepo, outboxRepo)
	router := setupFeatureFlagRouter(handler)

	flag := createTestFeatureFlag("test-flag", "Test Flag")
	flagRepo.On("FindByKey", mock.Anything, "test-flag").Return(flag, nil)
	flagRepo.On("Update", mock.Anything, mock.AnythingOfType("*featureflag.FeatureFlag")).Return(nil)
	auditRepo.On("Create", mock.Anything, mock.AnythingOfType("*featureflag.FlagAuditLog")).Return(nil)
	outboxRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/feature-flags/test-flag/enable", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp httpdto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	flagRepo.AssertExpectations(t)
}

func TestFeatureFlagHandler_DisableFlag_Success(t *testing.T) {
	flagRepo := new(MockFeatureFlagRepository)
	overrideRepo := new(MockFlagOverrideRepository)
	auditRepo := new(MockFlagAuditLogRepository)
	outboxRepo := new(MockOutboxRepository)

	handler := createFeatureFlagHandler(flagRepo, overrideRepo, auditRepo, outboxRepo)
	router := setupFeatureFlagRouter(handler)

	flag := createTestFeatureFlag("test-flag", "Test Flag")
	_ = flag.Enable(nil) // Enable first so we can disable
	flagRepo.On("FindByKey", mock.Anything, "test-flag").Return(flag, nil)
	flagRepo.On("Update", mock.Anything, mock.AnythingOfType("*featureflag.FeatureFlag")).Return(nil)
	auditRepo.On("Create", mock.Anything, mock.AnythingOfType("*featureflag.FlagAuditLog")).Return(nil)
	outboxRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/feature-flags/test-flag/disable", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp httpdto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	flagRepo.AssertExpectations(t)
}

func TestFeatureFlagHandler_EnableArchivedFlag_Fails(t *testing.T) {
	flagRepo := new(MockFeatureFlagRepository)
	overrideRepo := new(MockFlagOverrideRepository)
	auditRepo := new(MockFlagAuditLogRepository)
	outboxRepo := new(MockOutboxRepository)

	handler := createFeatureFlagHandler(flagRepo, overrideRepo, auditRepo, outboxRepo)
	router := setupFeatureFlagRouter(handler)

	flag := createTestFeatureFlag("archived-flag", "Archived Flag")
	_ = flag.Archive(nil) // Archive the flag
	flagRepo.On("FindByKey", mock.Anything, "archived-flag").Return(flag, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/feature-flags/archived-flag/enable", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	var resp httpdto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)

	flagRepo.AssertExpectations(t)
}

// ============================================================================
// Evaluation API Tests
// ============================================================================

func TestFeatureFlagHandler_EvaluateFlag_Success(t *testing.T) {
	flagRepo := new(MockFeatureFlagRepository)
	overrideRepo := new(MockFlagOverrideRepository)
	auditRepo := new(MockFlagAuditLogRepository)
	outboxRepo := new(MockOutboxRepository)

	handler := createFeatureFlagHandler(flagRepo, overrideRepo, auditRepo, outboxRepo)
	router := setupFeatureFlagRouter(handler)

	flag := createTestFeatureFlag("test-flag", "Test Flag")
	_ = flag.Enable(nil)
	flagRepo.On("FindByKey", mock.Anything, "test-flag").Return(flag, nil)
	// The evaluator will try to find user and tenant overrides
	overrideRepo.On("FindByFlagKeyAndTarget", mock.Anything, "test-flag", mock.Anything, mock.Anything).Return(nil, shared.ErrNotFound)

	reqBody := EvaluateFlagHTTPRequest{
		Context: dto.EvaluationContextDTO{
			UserID:   "550e8400-e29b-41d4-a716-446655440000",
			TenantID: "550e8400-e29b-41d4-a716-446655440001",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/feature-flags/test-flag/evaluate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp httpdto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	flagRepo.AssertExpectations(t)
}

func TestFeatureFlagHandler_BatchEvaluate_Success(t *testing.T) {
	flagRepo := new(MockFeatureFlagRepository)
	overrideRepo := new(MockFlagOverrideRepository)
	auditRepo := new(MockFlagAuditLogRepository)
	outboxRepo := new(MockOutboxRepository)

	handler := createFeatureFlagHandler(flagRepo, overrideRepo, auditRepo, outboxRepo)
	router := setupFeatureFlagRouter(handler)

	flag1 := createTestFeatureFlag("flag-1", "Flag 1")
	_ = flag1.Enable(nil)
	flag2 := createTestFeatureFlag("flag-2", "Flag 2")
	_ = flag2.Enable(nil)

	flagRepo.On("FindByKey", mock.Anything, "flag-1").Return(flag1, nil)
	flagRepo.On("FindByKey", mock.Anything, "flag-2").Return(flag2, nil)
	// The evaluator will try to find user and tenant overrides for each flag
	overrideRepo.On("FindByFlagKeyAndTarget", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, shared.ErrNotFound)

	reqBody := BatchEvaluateHTTPRequest{
		Keys: []string{"flag-1", "flag-2"},
		Context: dto.EvaluationContextDTO{
			UserID: "550e8400-e29b-41d4-a716-446655440000",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/feature-flags/evaluate-batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp httpdto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	flagRepo.AssertExpectations(t)
}

func TestFeatureFlagHandler_GetClientConfig_Success(t *testing.T) {
	flagRepo := new(MockFeatureFlagRepository)
	overrideRepo := new(MockFlagOverrideRepository)
	auditRepo := new(MockFlagAuditLogRepository)
	outboxRepo := new(MockOutboxRepository)

	handler := createFeatureFlagHandler(flagRepo, overrideRepo, auditRepo, outboxRepo)
	router := setupFeatureFlagRouter(handler)

	flag1 := createTestFeatureFlag("enabled-flag", "Enabled Flag")
	_ = flag1.Enable(nil)
	flags := []featureflag.FeatureFlag{*flag1}

	flagRepo.On("FindEnabled", mock.Anything, mock.AnythingOfType("shared.Filter")).Return(flags, nil)
	overrideRepo.On("FindForEvaluation", mock.Anything, "enabled-flag", mock.Anything, mock.Anything).Return(nil, nil)

	reqBody := ClientConfigHTTPRequest{
		Context: dto.EvaluationContextDTO{
			UserID: "user-123",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/feature-flags/client-config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp httpdto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	flagRepo.AssertExpectations(t)
}

// ============================================================================
// Override API Tests
// ============================================================================

func TestFeatureFlagHandler_CreateOverride_Success(t *testing.T) {
	flagRepo := new(MockFeatureFlagRepository)
	overrideRepo := new(MockFlagOverrideRepository)
	auditRepo := new(MockFlagAuditLogRepository)
	outboxRepo := new(MockOutboxRepository)

	handler := createFeatureFlagHandler(flagRepo, overrideRepo, auditRepo, outboxRepo)
	router := setupFeatureFlagRouter(handler)

	flag := createTestFeatureFlag("test-flag", "Test Flag")
	_ = flag.Enable(nil)
	targetID := uuid.New()

	flagRepo.On("FindByKey", mock.Anything, "test-flag").Return(flag, nil)
	overrideRepo.On("FindByFlagKeyAndTarget", mock.Anything, "test-flag", featureflag.OverrideTargetTypeUser, targetID).Return(nil, shared.ErrNotFound)
	overrideRepo.On("Create", mock.Anything, mock.AnythingOfType("*featureflag.FlagOverride")).Return(nil)
	auditRepo.On("Create", mock.Anything, mock.AnythingOfType("*featureflag.FlagAuditLog")).Return(nil)
	outboxRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

	reqBody := CreateOverrideHTTPRequest{
		TargetType: "user",
		TargetID:   targetID.String(),
		Value: dto.FlagValueDTO{
			Enabled: true,
		},
		Reason: "Testing override",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/feature-flags/test-flag/overrides", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp httpdto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	flagRepo.AssertExpectations(t)
	overrideRepo.AssertExpectations(t)
}

func TestFeatureFlagHandler_ListOverrides_Success(t *testing.T) {
	flagRepo := new(MockFeatureFlagRepository)
	overrideRepo := new(MockFlagOverrideRepository)
	auditRepo := new(MockFlagAuditLogRepository)
	outboxRepo := new(MockOutboxRepository)

	handler := createFeatureFlagHandler(flagRepo, overrideRepo, auditRepo, outboxRepo)
	router := setupFeatureFlagRouter(handler)

	flag := createTestFeatureFlag("test-flag", "Test Flag")
	override1, _ := featureflag.NewFlagOverride(
		"test-flag",
		featureflag.OverrideTargetTypeUser,
		uuid.New(),
		featureflag.NewBooleanFlagValue(true),
		"reason1",
		nil,
		nil,
	)
	overrides := []featureflag.FlagOverride{*override1}

	flagRepo.On("FindByKey", mock.Anything, "test-flag").Return(flag, nil)
	overrideRepo.On("FindByFlagKey", mock.Anything, "test-flag", mock.AnythingOfType("shared.Filter")).Return(overrides, nil)
	overrideRepo.On("CountByFlagKey", mock.Anything, "test-flag").Return(int64(1), nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/feature-flags/test-flag/overrides", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp httpdto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	flagRepo.AssertExpectations(t)
	overrideRepo.AssertExpectations(t)
}

func TestFeatureFlagHandler_DeleteOverride_Success(t *testing.T) {
	flagRepo := new(MockFeatureFlagRepository)
	overrideRepo := new(MockFlagOverrideRepository)
	auditRepo := new(MockFlagAuditLogRepository)
	outboxRepo := new(MockOutboxRepository)

	handler := createFeatureFlagHandler(flagRepo, overrideRepo, auditRepo, outboxRepo)
	router := setupFeatureFlagRouter(handler)

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
	flag := createTestFeatureFlag("test-flag", "Test Flag")

	overrideRepo.On("FindByID", mock.Anything, overrideID).Return(override, nil)
	flagRepo.On("FindByKey", mock.Anything, "test-flag").Return(flag, nil)
	overrideRepo.On("Delete", mock.Anything, overrideID).Return(nil)
	auditRepo.On("Create", mock.Anything, mock.AnythingOfType("*featureflag.FlagAuditLog")).Return(nil)
	outboxRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/feature-flags/test-flag/overrides/"+overrideID.String(), nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	overrideRepo.AssertExpectations(t)
}

// ============================================================================
// Audit Log Tests
// ============================================================================

func TestFeatureFlagHandler_GetAuditLogs_Success(t *testing.T) {
	flagRepo := new(MockFeatureFlagRepository)
	overrideRepo := new(MockFlagOverrideRepository)
	auditRepo := new(MockFlagAuditLogRepository)
	outboxRepo := new(MockOutboxRepository)

	handler := createFeatureFlagHandler(flagRepo, overrideRepo, auditRepo, outboxRepo)
	router := setupFeatureFlagRouter(handler)

	flag := createTestFeatureFlag("test-flag", "Test Flag")
	auditLog, _ := featureflag.NewFlagAuditLog(
		"test-flag",
		featureflag.AuditActionCreated,
		nil,
		nil,
		newTestUserID(),
		"127.0.0.1",
		"test-agent",
	)
	auditLogs := []featureflag.FlagAuditLog{*auditLog}

	flagRepo.On("FindByKey", mock.Anything, "test-flag").Return(flag, nil)
	auditRepo.On("FindByFlagKey", mock.Anything, "test-flag", mock.AnythingOfType("shared.Filter")).Return(auditLogs, nil)
	auditRepo.On("CountByFlagKey", mock.Anything, "test-flag").Return(int64(1), nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/feature-flags/test-flag/audit-logs", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp httpdto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	flagRepo.AssertExpectations(t)
	auditRepo.AssertExpectations(t)
}

// ============================================================================
// Invalid Request Tests
// ============================================================================

func TestFeatureFlagHandler_CreateFlag_InvalidRequestBody(t *testing.T) {
	flagRepo := new(MockFeatureFlagRepository)
	overrideRepo := new(MockFlagOverrideRepository)
	auditRepo := new(MockFlagAuditLogRepository)
	outboxRepo := new(MockOutboxRepository)

	handler := createFeatureFlagHandler(flagRepo, overrideRepo, auditRepo, outboxRepo)
	router := setupFeatureFlagRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/feature-flags", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFeatureFlagHandler_CreateFlag_MissingRequiredFields(t *testing.T) {
	flagRepo := new(MockFeatureFlagRepository)
	overrideRepo := new(MockFlagOverrideRepository)
	auditRepo := new(MockFlagAuditLogRepository)
	outboxRepo := new(MockOutboxRepository)

	handler := createFeatureFlagHandler(flagRepo, overrideRepo, auditRepo, outboxRepo)
	router := setupFeatureFlagRouter(handler)

	// Missing key and name
	reqBody := CreateFlagHTTPRequest{
		Type: "boolean",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/feature-flags", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFeatureFlagHandler_CreateFlag_InvalidFlagType(t *testing.T) {
	flagRepo := new(MockFeatureFlagRepository)
	overrideRepo := new(MockFlagOverrideRepository)
	auditRepo := new(MockFlagAuditLogRepository)
	outboxRepo := new(MockOutboxRepository)

	handler := createFeatureFlagHandler(flagRepo, overrideRepo, auditRepo, outboxRepo)
	router := setupFeatureFlagRouter(handler)

	reqBody := CreateFlagHTTPRequest{
		Key:  "test-flag",
		Name: "Test Flag",
		Type: "invalid_type",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/feature-flags", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFeatureFlagHandler_CreateOverride_InvalidTargetID(t *testing.T) {
	flagRepo := new(MockFeatureFlagRepository)
	overrideRepo := new(MockFlagOverrideRepository)
	auditRepo := new(MockFlagAuditLogRepository)
	outboxRepo := new(MockOutboxRepository)

	handler := createFeatureFlagHandler(flagRepo, overrideRepo, auditRepo, outboxRepo)
	router := setupFeatureFlagRouter(handler)

	reqBody := CreateOverrideHTTPRequest{
		TargetType: "user",
		TargetID:   "not-a-valid-uuid",
		Value: dto.FlagValueDTO{
			Enabled: true,
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/feature-flags/test-flag/overrides", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFeatureFlagHandler_DeleteOverride_InvalidOverrideID(t *testing.T) {
	flagRepo := new(MockFeatureFlagRepository)
	overrideRepo := new(MockFlagOverrideRepository)
	auditRepo := new(MockFlagAuditLogRepository)
	outboxRepo := new(MockOutboxRepository)

	handler := createFeatureFlagHandler(flagRepo, overrideRepo, auditRepo, outboxRepo)
	router := setupFeatureFlagRouter(handler)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/feature-flags/test-flag/overrides/not-a-valid-uuid", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// Percentage and Variant Flag Tests
// ============================================================================

func TestFeatureFlagHandler_CreatePercentageFlag_Success(t *testing.T) {
	flagRepo := new(MockFeatureFlagRepository)
	overrideRepo := new(MockFlagOverrideRepository)
	auditRepo := new(MockFlagAuditLogRepository)
	outboxRepo := new(MockOutboxRepository)

	handler := createFeatureFlagHandler(flagRepo, overrideRepo, auditRepo, outboxRepo)
	router := setupFeatureFlagRouter(handler)

	flagRepo.On("ExistsByKey", mock.Anything, "percentage-flag").Return(false, nil)
	flagRepo.On("Create", mock.Anything, mock.AnythingOfType("*featureflag.FeatureFlag")).Return(nil)
	auditRepo.On("Create", mock.Anything, mock.AnythingOfType("*featureflag.FlagAuditLog")).Return(nil)
	outboxRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

	reqBody := CreateFlagHTTPRequest{
		Key:         "percentage-flag",
		Name:        "Percentage Flag",
		Description: "A percentage rollout flag",
		Type:        "percentage",
		DefaultValue: dto.FlagValueDTO{
			Enabled: false,
			Metadata: map[string]any{
				"percentage": 50,
			},
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/feature-flags", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	flagRepo.AssertExpectations(t)
}

func TestFeatureFlagHandler_CreateVariantFlag_Success(t *testing.T) {
	flagRepo := new(MockFeatureFlagRepository)
	overrideRepo := new(MockFlagOverrideRepository)
	auditRepo := new(MockFlagAuditLogRepository)
	outboxRepo := new(MockOutboxRepository)

	handler := createFeatureFlagHandler(flagRepo, overrideRepo, auditRepo, outboxRepo)
	router := setupFeatureFlagRouter(handler)

	flagRepo.On("ExistsByKey", mock.Anything, "variant-flag").Return(false, nil)
	flagRepo.On("Create", mock.Anything, mock.AnythingOfType("*featureflag.FeatureFlag")).Return(nil)
	auditRepo.On("Create", mock.Anything, mock.AnythingOfType("*featureflag.FlagAuditLog")).Return(nil)
	outboxRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

	reqBody := CreateFlagHTTPRequest{
		Key:         "variant-flag",
		Name:        "Variant Flag",
		Description: "A variant A/B test flag",
		Type:        "variant",
		DefaultValue: dto.FlagValueDTO{
			Variant: "control",
			Metadata: map[string]any{
				"variants": []map[string]any{
					{"key": "control", "value": "original", "weight": 50},
					{"key": "treatment", "value": "new", "weight": 50},
				},
			},
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/feature-flags", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	flagRepo.AssertExpectations(t)
}
