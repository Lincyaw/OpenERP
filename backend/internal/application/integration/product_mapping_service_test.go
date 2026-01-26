package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/integration"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockProductMappingRepository is a mock implementation of ProductMappingRepository
type MockProductMappingRepository struct {
	mock.Mock
}

func (m *MockProductMappingRepository) FindByID(ctx context.Context, id uuid.UUID) (*integration.ProductMapping, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*integration.ProductMapping), args.Error(1)
}

func (m *MockProductMappingRepository) FindByLocalProduct(ctx context.Context, tenantID uuid.UUID, localProductID uuid.UUID) ([]integration.ProductMapping, error) {
	args := m.Called(ctx, tenantID, localProductID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]integration.ProductMapping), args.Error(1)
}

func (m *MockProductMappingRepository) FindByLocalProductAndPlatform(ctx context.Context, tenantID uuid.UUID, localProductID uuid.UUID, platformCode integration.PlatformCode) (*integration.ProductMapping, error) {
	args := m.Called(ctx, tenantID, localProductID, platformCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*integration.ProductMapping), args.Error(1)
}

func (m *MockProductMappingRepository) FindByPlatformProduct(ctx context.Context, tenantID uuid.UUID, platformCode integration.PlatformCode, platformProductID string) (*integration.ProductMapping, error) {
	args := m.Called(ctx, tenantID, platformCode, platformProductID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*integration.ProductMapping), args.Error(1)
}

func (m *MockProductMappingRepository) FindByPlatformSku(ctx context.Context, tenantID uuid.UUID, platformCode integration.PlatformCode, platformSkuID string) (*integration.ProductMapping, error) {
	args := m.Called(ctx, tenantID, platformCode, platformSkuID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*integration.ProductMapping), args.Error(1)
}

func (m *MockProductMappingRepository) FindAll(ctx context.Context, tenantID uuid.UUID, filter integration.ProductMappingFilter) ([]integration.ProductMapping, error) {
	args := m.Called(ctx, tenantID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]integration.ProductMapping), args.Error(1)
}

func (m *MockProductMappingRepository) FindActiveForPlatform(ctx context.Context, tenantID uuid.UUID, platformCode integration.PlatformCode) ([]integration.ProductMapping, error) {
	args := m.Called(ctx, tenantID, platformCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]integration.ProductMapping), args.Error(1)
}

func (m *MockProductMappingRepository) FindSyncEnabled(ctx context.Context, tenantID uuid.UUID, platformCode integration.PlatformCode) ([]integration.ProductMapping, error) {
	args := m.Called(ctx, tenantID, platformCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]integration.ProductMapping), args.Error(1)
}

func (m *MockProductMappingRepository) Count(ctx context.Context, tenantID uuid.UUID, filter integration.ProductMappingFilter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockProductMappingRepository) ExistsByLocalProductAndPlatform(ctx context.Context, tenantID uuid.UUID, localProductID uuid.UUID, platformCode integration.PlatformCode) (bool, error) {
	args := m.Called(ctx, tenantID, localProductID, platformCode)
	return args.Bool(0), args.Error(1)
}

func (m *MockProductMappingRepository) ExistsByPlatformProduct(ctx context.Context, tenantID uuid.UUID, platformCode integration.PlatformCode, platformProductID string) (bool, error) {
	args := m.Called(ctx, tenantID, platformCode, platformProductID)
	return args.Bool(0), args.Error(1)
}

func (m *MockProductMappingRepository) Save(ctx context.Context, mapping *integration.ProductMapping) error {
	args := m.Called(ctx, mapping)
	return args.Error(0)
}

func (m *MockProductMappingRepository) SaveBatch(ctx context.Context, mappings []*integration.ProductMapping) error {
	args := m.Called(ctx, mappings)
	return args.Error(0)
}

func (m *MockProductMappingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockProductMappingRepository) DeleteByLocalProduct(ctx context.Context, tenantID uuid.UUID, localProductID uuid.UUID) error {
	args := m.Called(ctx, tenantID, localProductID)
	return args.Error(0)
}

func (m *MockProductMappingRepository) DeleteByLocalProductAndPlatform(ctx context.Context, tenantID uuid.UUID, localProductID uuid.UUID, platformCode integration.PlatformCode) error {
	args := m.Called(ctx, tenantID, localProductID, platformCode)
	return args.Error(0)
}

// Ensure mock implements interface
var _ integration.ProductMappingRepository = (*MockProductMappingRepository)(nil)

// Test fixtures
var (
	testTenantID       = uuid.New()
	testLocalProductID = uuid.New()
	testMappingID      = uuid.New()
	testLocalSKUID     = uuid.New()
)

func createTestMapping() *integration.ProductMapping {
	now := time.Now()
	return &integration.ProductMapping{
		ID:                testMappingID,
		TenantID:          testTenantID,
		LocalProductID:    testLocalProductID,
		PlatformCode:      integration.PlatformCodeTaobao,
		PlatformProductID: "taobao-123",
		SKUMappings:       []integration.SKUMapping{},
		IsActive:          true,
		SyncEnabled:       true,
		LastSyncStatus:    integration.SyncStatusPending,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// ---------------------------------------------------------------------------
// CreateMapping Tests
// ---------------------------------------------------------------------------

func TestCreateMapping_Success(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	// Setup expectations
	mockRepo.On("ExistsByLocalProductAndPlatform", ctx, testTenantID, testLocalProductID, integration.PlatformCodeTaobao).Return(false, nil)
	mockRepo.On("ExistsByPlatformProduct", ctx, testTenantID, integration.PlatformCodeTaobao, "taobao-123").Return(false, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*integration.ProductMapping")).Return(nil)

	// Execute
	mapping, err := service.CreateMapping(ctx, testTenantID, testLocalProductID, integration.PlatformCodeTaobao, "taobao-123")

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, mapping)
	assert.Equal(t, testTenantID, mapping.TenantID)
	assert.Equal(t, testLocalProductID, mapping.LocalProductID)
	assert.Equal(t, integration.PlatformCodeTaobao, mapping.PlatformCode)
	assert.Equal(t, "taobao-123", mapping.PlatformProductID)
	assert.True(t, mapping.IsActive)
	assert.True(t, mapping.SyncEnabled)
	mockRepo.AssertExpectations(t)
}

func TestCreateMapping_AlreadyExists(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	// Setup expectations
	mockRepo.On("ExistsByLocalProductAndPlatform", ctx, testTenantID, testLocalProductID, integration.PlatformCodeTaobao).Return(true, nil)

	// Execute
	mapping, err := service.CreateMapping(ctx, testTenantID, testLocalProductID, integration.PlatformCodeTaobao, "taobao-123")

	// Verify
	assert.Error(t, err)
	assert.Nil(t, mapping)
	assert.Equal(t, integration.ErrMappingAlreadyExists, err)
	mockRepo.AssertExpectations(t)
}

func TestCreateMapping_PlatformProductAlreadyMapped(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	// Setup expectations
	mockRepo.On("ExistsByLocalProductAndPlatform", ctx, testTenantID, testLocalProductID, integration.PlatformCodeTaobao).Return(false, nil)
	mockRepo.On("ExistsByPlatformProduct", ctx, testTenantID, integration.PlatformCodeTaobao, "taobao-123").Return(true, nil)

	// Execute
	mapping, err := service.CreateMapping(ctx, testTenantID, testLocalProductID, integration.PlatformCodeTaobao, "taobao-123")

	// Verify
	assert.Error(t, err)
	assert.Nil(t, mapping)
	mockRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// GetMapping Tests
// ---------------------------------------------------------------------------

func TestGetMapping_Success(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	expectedMapping := createTestMapping()
	mockRepo.On("FindByID", ctx, testMappingID).Return(expectedMapping, nil)

	// Execute
	mapping, err := service.GetMapping(ctx, testTenantID, testMappingID)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, mapping)
	assert.Equal(t, expectedMapping.ID, mapping.ID)
	mockRepo.AssertExpectations(t)
}

func TestGetMapping_NotFound(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	mockRepo.On("FindByID", ctx, testMappingID).Return(nil, integration.ErrMappingNotFound)

	// Execute
	mapping, err := service.GetMapping(ctx, testTenantID, testMappingID)

	// Verify
	assert.Error(t, err)
	assert.Nil(t, mapping)
	assert.Equal(t, integration.ErrMappingNotFound, err)
	mockRepo.AssertExpectations(t)
}

func TestGetMapping_WrongTenant(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	wrongTenantID := uuid.New()
	expectedMapping := createTestMapping()
	mockRepo.On("FindByID", ctx, testMappingID).Return(expectedMapping, nil)

	// Execute with wrong tenant
	mapping, err := service.GetMapping(ctx, wrongTenantID, testMappingID)

	// Verify - should return not found for security
	assert.Error(t, err)
	assert.Nil(t, mapping)
	assert.Equal(t, integration.ErrMappingNotFound, err)
	mockRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// DeleteMapping Tests
// ---------------------------------------------------------------------------

func TestDeleteMapping_Success(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	expectedMapping := createTestMapping()
	mockRepo.On("FindByID", ctx, testMappingID).Return(expectedMapping, nil)
	mockRepo.On("Delete", ctx, testMappingID).Return(nil)

	// Execute
	err := service.DeleteMapping(ctx, testTenantID, testMappingID)

	// Verify
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// AddSKUMapping Tests
// ---------------------------------------------------------------------------

func TestAddSKUMapping_Success(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	expectedMapping := createTestMapping()
	mockRepo.On("FindByID", ctx, testMappingID).Return(expectedMapping, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*integration.ProductMapping")).Return(nil)

	// Execute
	err := service.AddSKUMapping(ctx, testTenantID, testMappingID, testLocalSKUID, "platform-sku-123")

	// Verify
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAddSKUMapping_WrongTenant(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	wrongTenantID := uuid.New()
	expectedMapping := createTestMapping()
	mockRepo.On("FindByID", ctx, testMappingID).Return(expectedMapping, nil)

	// Execute
	err := service.AddSKUMapping(ctx, wrongTenantID, testMappingID, testLocalSKUID, "platform-sku-123")

	// Verify
	assert.Error(t, err)
	assert.Equal(t, integration.ErrMappingNotFound, err)
	mockRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// RemoveSKUMapping Tests
// ---------------------------------------------------------------------------

func TestRemoveSKUMapping_Success(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	expectedMapping := createTestMapping()
	expectedMapping.SKUMappings = []integration.SKUMapping{
		{LocalSKUID: testLocalSKUID, PlatformSkuID: "platform-sku-123", IsActive: true},
	}
	mockRepo.On("FindByID", ctx, testMappingID).Return(expectedMapping, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*integration.ProductMapping")).Return(nil)

	// Execute
	err := service.RemoveSKUMapping(ctx, testTenantID, testMappingID, "platform-sku-123")

	// Verify
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Lookup Tests
// ---------------------------------------------------------------------------

func TestGetLocalProductID_Success(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	expectedMapping := createTestMapping()
	mockRepo.On("FindByPlatformProduct", ctx, testTenantID, integration.PlatformCodeTaobao, "taobao-123").Return(expectedMapping, nil)

	// Execute
	localProductID, err := service.GetLocalProductID(ctx, testTenantID, integration.PlatformCodeTaobao, "taobao-123")

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, testLocalProductID, localProductID)
	mockRepo.AssertExpectations(t)
}

func TestGetLocalSKUID_Success(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	expectedMapping := createTestMapping()
	expectedMapping.SKUMappings = []integration.SKUMapping{
		{LocalSKUID: testLocalSKUID, PlatformSkuID: "platform-sku-123", IsActive: true},
	}
	mockRepo.On("FindByPlatformSku", ctx, testTenantID, integration.PlatformCodeTaobao, "platform-sku-123").Return(expectedMapping, nil)

	// Execute
	localSKUID, err := service.GetLocalSKUID(ctx, testTenantID, integration.PlatformCodeTaobao, "platform-sku-123")

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, testLocalSKUID, localSKUID)
	mockRepo.AssertExpectations(t)
}

func TestGetPlatformProductID_Success(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	expectedMapping := createTestMapping()
	mockRepo.On("FindByLocalProductAndPlatform", ctx, testTenantID, testLocalProductID, integration.PlatformCodeTaobao).Return(expectedMapping, nil)

	// Execute
	platformProductID, err := service.GetPlatformProductID(ctx, testTenantID, testLocalProductID, integration.PlatformCodeTaobao)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, "taobao-123", platformProductID)
	mockRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Sync Operations Tests
// ---------------------------------------------------------------------------

func TestEnableSync_Success(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	expectedMapping := createTestMapping()
	expectedMapping.SyncEnabled = false
	mockRepo.On("FindByID", ctx, testMappingID).Return(expectedMapping, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*integration.ProductMapping")).Return(nil)

	// Execute
	err := service.EnableSync(ctx, testTenantID, testMappingID)

	// Verify
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestDisableSync_Success(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	expectedMapping := createTestMapping()
	mockRepo.On("FindByID", ctx, testMappingID).Return(expectedMapping, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*integration.ProductMapping")).Return(nil)

	// Execute
	err := service.DisableSync(ctx, testTenantID, testMappingID)

	// Verify
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestGetMappingsForSync_Success(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	expectedMappings := []integration.ProductMapping{*createTestMapping()}
	mockRepo.On("FindSyncEnabled", ctx, testTenantID, integration.PlatformCodeTaobao).Return(expectedMappings, nil)

	// Execute
	mappings, err := service.GetMappingsForSync(ctx, testTenantID, integration.PlatformCodeTaobao)

	// Verify
	assert.NoError(t, err)
	assert.Len(t, mappings, 1)
	mockRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// ListMappings Tests
// ---------------------------------------------------------------------------

func TestListMappings_Success(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	expectedMappings := []integration.ProductMapping{*createTestMapping()}
	filter := integration.ProductMappingFilter{}

	mockRepo.On("FindAll", ctx, testTenantID, mock.AnythingOfType("integration.ProductMappingFilter")).Return(expectedMappings, nil)
	mockRepo.On("Count", ctx, testTenantID, mock.AnythingOfType("integration.ProductMappingFilter")).Return(int64(1), nil)

	// Execute
	mappings, count, err := service.ListMappings(ctx, testTenantID, filter)

	// Verify
	assert.NoError(t, err)
	assert.Len(t, mappings, 1)
	assert.Equal(t, int64(1), count)
	mockRepo.AssertExpectations(t)
}

func TestListMappings_WithDefaults(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	expectedMappings := []integration.ProductMapping{}
	filter := integration.ProductMappingFilter{Page: 0, PageSize: 0} // Test defaults

	mockRepo.On("FindAll", ctx, testTenantID, mock.MatchedBy(func(f integration.ProductMappingFilter) bool {
		return f.Page == 1 && f.PageSize == 20
	})).Return(expectedMappings, nil)
	mockRepo.On("Count", ctx, testTenantID, mock.AnythingOfType("integration.ProductMappingFilter")).Return(int64(0), nil)

	// Execute
	mappings, count, err := service.ListMappings(ctx, testTenantID, filter)

	// Verify
	assert.NoError(t, err)
	assert.Empty(t, mappings)
	assert.Equal(t, int64(0), count)
	mockRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Batch Operations Tests
// ---------------------------------------------------------------------------

func TestCreateBatchMappings_Success(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	requests := []CreateMappingRequest{
		{LocalProductID: uuid.New(), PlatformCode: integration.PlatformCodeTaobao, PlatformProductID: "tb-1"},
		{LocalProductID: uuid.New(), PlatformCode: integration.PlatformCodeJD, PlatformProductID: "jd-1"},
	}

	// Setup expectations for first mapping
	mockRepo.On("ExistsByLocalProductAndPlatform", ctx, testTenantID, requests[0].LocalProductID, integration.PlatformCodeTaobao).Return(false, nil)
	mockRepo.On("ExistsByPlatformProduct", ctx, testTenantID, integration.PlatformCodeTaobao, "tb-1").Return(false, nil)
	mockRepo.On("Save", ctx, mock.MatchedBy(func(m *integration.ProductMapping) bool {
		return m.PlatformProductID == "tb-1"
	})).Return(nil)

	// Setup expectations for second mapping
	mockRepo.On("ExistsByLocalProductAndPlatform", ctx, testTenantID, requests[1].LocalProductID, integration.PlatformCodeJD).Return(false, nil)
	mockRepo.On("ExistsByPlatformProduct", ctx, testTenantID, integration.PlatformCodeJD, "jd-1").Return(false, nil)
	mockRepo.On("Save", ctx, mock.MatchedBy(func(m *integration.ProductMapping) bool {
		return m.PlatformProductID == "jd-1"
	})).Return(nil)

	// Execute
	results, err := service.CreateBatchMappings(ctx, testTenantID, requests)

	// Verify
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.True(t, results[0].Success)
	assert.True(t, results[1].Success)
	mockRepo.AssertExpectations(t)
}

func TestCreateBatchMappings_PartialFailure(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	requests := []CreateMappingRequest{
		{LocalProductID: uuid.New(), PlatformCode: integration.PlatformCodeTaobao, PlatformProductID: "tb-1"},
		{LocalProductID: uuid.New(), PlatformCode: integration.PlatformCodeJD, PlatformProductID: "jd-1"},
	}

	// First mapping succeeds
	mockRepo.On("ExistsByLocalProductAndPlatform", ctx, testTenantID, requests[0].LocalProductID, integration.PlatformCodeTaobao).Return(false, nil)
	mockRepo.On("ExistsByPlatformProduct", ctx, testTenantID, integration.PlatformCodeTaobao, "tb-1").Return(false, nil)
	mockRepo.On("Save", ctx, mock.MatchedBy(func(m *integration.ProductMapping) bool {
		return m.PlatformProductID == "tb-1"
	})).Return(nil)

	// Second mapping already exists
	mockRepo.On("ExistsByLocalProductAndPlatform", ctx, testTenantID, requests[1].LocalProductID, integration.PlatformCodeJD).Return(true, nil)

	// Execute
	results, err := service.CreateBatchMappings(ctx, testTenantID, requests)

	// Verify
	assert.NoError(t, err) // Batch operation doesn't fail overall
	assert.Len(t, results, 2)
	assert.True(t, results[0].Success)
	assert.False(t, results[1].Success)
	assert.NotEmpty(t, results[1].Error)
	mockRepo.AssertExpectations(t)
}

func TestActivateMappings_Success(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	mapping1ID := uuid.New()
	mapping2ID := uuid.New()

	mapping1 := createTestMapping()
	mapping1.ID = mapping1ID
	mapping1.IsActive = false

	mapping2 := createTestMapping()
	mapping2.ID = mapping2ID
	mapping2.IsActive = false

	mockRepo.On("FindByID", ctx, mapping1ID).Return(mapping1, nil)
	mockRepo.On("Save", ctx, mock.MatchedBy(func(m *integration.ProductMapping) bool {
		return m.ID == mapping1ID && m.IsActive
	})).Return(nil)

	mockRepo.On("FindByID", ctx, mapping2ID).Return(mapping2, nil)
	mockRepo.On("Save", ctx, mock.MatchedBy(func(m *integration.ProductMapping) bool {
		return m.ID == mapping2ID && m.IsActive
	})).Return(nil)

	// Execute
	err := service.ActivateMappings(ctx, testTenantID, []uuid.UUID{mapping1ID, mapping2ID})

	// Verify
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestDeactivateMappings_Success(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	mapping1ID := uuid.New()

	mapping1 := createTestMapping()
	mapping1.ID = mapping1ID
	mapping1.IsActive = true

	mockRepo.On("FindByID", ctx, mapping1ID).Return(mapping1, nil)
	mockRepo.On("Save", ctx, mock.MatchedBy(func(m *integration.ProductMapping) bool {
		return m.ID == mapping1ID && !m.IsActive
	})).Return(nil)

	// Execute
	err := service.DeactivateMappings(ctx, testTenantID, []uuid.UUID{mapping1ID})

	// Verify
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Additional Error Path Tests
// ---------------------------------------------------------------------------

func TestDeleteMapping_WrongTenant(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	wrongTenantID := uuid.New()
	expectedMapping := createTestMapping()
	mockRepo.On("FindByID", ctx, testMappingID).Return(expectedMapping, nil)

	// Execute with wrong tenant
	err := service.DeleteMapping(ctx, wrongTenantID, testMappingID)

	// Verify - should return not found for security
	assert.Error(t, err)
	assert.Equal(t, integration.ErrMappingNotFound, err)
	mockRepo.AssertExpectations(t)
}

func TestGetLocalSKUID_SKUNotFound(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	// Mapping exists but has no SKU mappings
	expectedMapping := createTestMapping()
	expectedMapping.SKUMappings = []integration.SKUMapping{} // Empty
	mockRepo.On("FindByPlatformSku", ctx, testTenantID, integration.PlatformCodeTaobao, "non-existent-sku").Return(expectedMapping, nil)

	// Execute
	localSKUID, err := service.GetLocalSKUID(ctx, testTenantID, integration.PlatformCodeTaobao, "non-existent-sku")

	// Verify
	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, localSKUID)
	assert.Equal(t, integration.ErrMappingSkuMappingInvalid, err)
	mockRepo.AssertExpectations(t)
}

func TestListMappings_FindAllError(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	filter := integration.ProductMappingFilter{}
	expectedErr := errors.New("database error")

	mockRepo.On("FindAll", ctx, testTenantID, mock.AnythingOfType("integration.ProductMappingFilter")).Return(nil, expectedErr)

	// Execute
	mappings, count, err := service.ListMappings(ctx, testTenantID, filter)

	// Verify
	assert.Error(t, err)
	assert.Nil(t, mappings)
	assert.Equal(t, int64(0), count)
	mockRepo.AssertExpectations(t)
}

func TestEnableSync_WrongTenant(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	wrongTenantID := uuid.New()
	expectedMapping := createTestMapping()
	mockRepo.On("FindByID", ctx, testMappingID).Return(expectedMapping, nil)

	// Execute
	err := service.EnableSync(ctx, wrongTenantID, testMappingID)

	// Verify
	assert.Error(t, err)
	assert.Equal(t, integration.ErrMappingNotFound, err)
	mockRepo.AssertExpectations(t)
}

func TestActivateMappings_WrongTenant(t *testing.T) {
	mockRepo := new(MockProductMappingRepository)
	service := NewProductMappingService(mockRepo)
	ctx := context.Background()

	wrongTenantID := uuid.New()
	mapping1ID := uuid.New()

	mapping1 := createTestMapping()
	mapping1.ID = mapping1ID

	mockRepo.On("FindByID", ctx, mapping1ID).Return(mapping1, nil)

	// Execute
	err := service.ActivateMappings(ctx, wrongTenantID, []uuid.UUID{mapping1ID})

	// Verify
	assert.Error(t, err)
	assert.Equal(t, integration.ErrMappingNotFound, err)
	mockRepo.AssertExpectations(t)
}
