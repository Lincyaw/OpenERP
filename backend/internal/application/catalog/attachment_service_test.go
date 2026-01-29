package catalog

import (
	"context"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/catalog"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// Mocks
// ============================================================================

// MockProductAttachmentRepository is a mock implementation of ProductAttachmentRepository
type MockProductAttachmentRepository struct {
	mock.Mock
}

func (m *MockProductAttachmentRepository) FindByID(ctx context.Context, id uuid.UUID) (*catalog.ProductAttachment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*catalog.ProductAttachment), args.Error(1)
}

func (m *MockProductAttachmentRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*catalog.ProductAttachment, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*catalog.ProductAttachment), args.Error(1)
}

func (m *MockProductAttachmentRepository) FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]catalog.ProductAttachment, error) {
	args := m.Called(ctx, tenantID, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]catalog.ProductAttachment), args.Error(1)
}

func (m *MockProductAttachmentRepository) FindByProduct(ctx context.Context, tenantID, productID uuid.UUID, filter shared.Filter) ([]catalog.ProductAttachment, error) {
	args := m.Called(ctx, tenantID, productID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]catalog.ProductAttachment), args.Error(1)
}

func (m *MockProductAttachmentRepository) FindByProductAndStatus(ctx context.Context, tenantID, productID uuid.UUID, status catalog.AttachmentStatus, filter shared.Filter) ([]catalog.ProductAttachment, error) {
	args := m.Called(ctx, tenantID, productID, status, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]catalog.ProductAttachment), args.Error(1)
}

func (m *MockProductAttachmentRepository) FindActiveByProduct(ctx context.Context, tenantID, productID uuid.UUID) ([]catalog.ProductAttachment, error) {
	args := m.Called(ctx, tenantID, productID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]catalog.ProductAttachment), args.Error(1)
}

func (m *MockProductAttachmentRepository) FindMainImage(ctx context.Context, tenantID, productID uuid.UUID) (*catalog.ProductAttachment, error) {
	args := m.Called(ctx, tenantID, productID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*catalog.ProductAttachment), args.Error(1)
}

func (m *MockProductAttachmentRepository) FindByType(ctx context.Context, tenantID, productID uuid.UUID, attachmentType catalog.AttachmentType) ([]catalog.ProductAttachment, error) {
	args := m.Called(ctx, tenantID, productID, attachmentType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]catalog.ProductAttachment), args.Error(1)
}

func (m *MockProductAttachmentRepository) CountByProduct(ctx context.Context, tenantID, productID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID, productID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockProductAttachmentRepository) CountActiveByProduct(ctx context.Context, tenantID, productID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID, productID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockProductAttachmentRepository) ExistsByStorageKey(ctx context.Context, tenantID uuid.UUID, storageKey string) (bool, error) {
	args := m.Called(ctx, tenantID, storageKey)
	return args.Bool(0), args.Error(1)
}

func (m *MockProductAttachmentRepository) GetMaxSortOrder(ctx context.Context, tenantID, productID uuid.UUID) (int, error) {
	args := m.Called(ctx, tenantID, productID)
	return args.Int(0), args.Error(1)
}

func (m *MockProductAttachmentRepository) Save(ctx context.Context, attachment *catalog.ProductAttachment) error {
	args := m.Called(ctx, attachment)
	return args.Error(0)
}

func (m *MockProductAttachmentRepository) SaveBatch(ctx context.Context, attachments []*catalog.ProductAttachment) error {
	args := m.Called(ctx, attachments)
	return args.Error(0)
}

func (m *MockProductAttachmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockProductAttachmentRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *MockProductAttachmentRepository) DeleteByProduct(ctx context.Context, tenantID, productID uuid.UUID) error {
	args := m.Called(ctx, tenantID, productID)
	return args.Error(0)
}

var _ catalog.ProductAttachmentRepository = (*MockProductAttachmentRepository)(nil)

// MockObjectStorageService is a mock implementation of ObjectStorageService
type MockObjectStorageService struct {
	mock.Mock
}

func (m *MockObjectStorageService) GenerateUploadURL(ctx context.Context, storageKey, contentType string, expiresIn time.Duration) (string, time.Time, error) {
	args := m.Called(ctx, storageKey, contentType, expiresIn)
	return args.String(0), args.Get(1).(time.Time), args.Error(2)
}

func (m *MockObjectStorageService) GenerateDownloadURL(ctx context.Context, storageKey string, expiresIn time.Duration) (string, time.Time, error) {
	args := m.Called(ctx, storageKey, expiresIn)
	return args.String(0), args.Get(1).(time.Time), args.Error(2)
}

func (m *MockObjectStorageService) DeleteObject(ctx context.Context, storageKey string) error {
	args := m.Called(ctx, storageKey)
	return args.Error(0)
}

func (m *MockObjectStorageService) ObjectExists(ctx context.Context, storageKey string) (bool, error) {
	args := m.Called(ctx, storageKey)
	return args.Bool(0), args.Error(1)
}

var _ ObjectStorageService = (*MockObjectStorageService)(nil)

// ============================================================================
// Test Helpers
// ============================================================================

func newAttachmentTestTenantID() uuid.UUID {
	return uuid.MustParse("11111111-1111-1111-1111-111111111111")
}

func newAttachmentTestProductID() uuid.UUID {
	return uuid.MustParse("22222222-2222-2222-2222-222222222222")
}

func newAttachmentTestAttachmentID() uuid.UUID {
	return uuid.MustParse("33333333-3333-3333-3333-333333333333")
}

func createTestAttachment(tenantID, productID uuid.UUID) *catalog.ProductAttachment {
	userID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	attachment, _ := catalog.NewProductAttachment(
		tenantID,
		productID,
		catalog.AttachmentTypeGalleryImage,
		"test-image.jpg",
		1024,
		"image/jpeg",
		"tenants/test/products/test/attachments/test.jpg",
		&userID,
	)
	return attachment
}

func createActiveTestAttachment(tenantID, productID uuid.UUID) *catalog.ProductAttachment {
	attachment := createTestAttachment(tenantID, productID)
	attachment.Confirm()
	return attachment
}

func newTestAttachmentService() (*AttachmentService, *MockProductAttachmentRepository, *MockProductRepository, *MockObjectStorageService) {
	mockAttachmentRepo := new(MockProductAttachmentRepository)
	mockProductRepo := new(MockProductRepository)
	mockStorageService := new(MockObjectStorageService)
	service := NewAttachmentService(mockAttachmentRepo, mockProductRepo, mockStorageService)
	return service, mockAttachmentRepo, mockProductRepo, mockStorageService
}

// ============================================================================
// InitiateUpload Tests
// ============================================================================

func TestAttachmentService_InitiateUpload_Success(t *testing.T) {
	service, mockAttachmentRepo, mockProductRepo, mockStorageService := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()
	product := createTestProduct(tenantID)
	userID := uuid.New()
	expiresAt := time.Now().Add(15 * time.Minute)

	req := InitiateUploadRequest{
		ProductID:   productID,
		Type:        "gallery_image",
		FileName:    "product-photo.jpg",
		FileSize:    2048,
		ContentType: "image/jpeg",
	}

	mockProductRepo.On("FindByIDForTenant", ctx, tenantID, productID).Return(product, nil)
	mockAttachmentRepo.On("CountActiveByProduct", ctx, tenantID, productID).Return(int64(5), nil)
	mockAttachmentRepo.On("Save", ctx, mock.AnythingOfType("*catalog.ProductAttachment")).Return(nil)
	mockStorageService.On("GenerateUploadURL", ctx, mock.AnythingOfType("string"), "image/jpeg", mock.AnythingOfType("time.Duration")).Return("https://storage.example.com/upload?token=xyz", expiresAt, nil)

	result, err := service.InitiateUpload(ctx, tenantID, req, &userID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEqual(t, uuid.Nil, result.AttachmentID)
	assert.Equal(t, "https://storage.example.com/upload?token=xyz", result.UploadURL)
	// Note: StorageKey is intentionally not exposed in response for security reasons
	mockProductRepo.AssertExpectations(t)
	mockAttachmentRepo.AssertExpectations(t)
	mockStorageService.AssertExpectations(t)
}

func TestAttachmentService_InitiateUpload_ProductNotFound(t *testing.T) {
	service, _, mockProductRepo, _ := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()
	userID := uuid.New()

	req := InitiateUploadRequest{
		ProductID:   productID,
		Type:        "gallery_image",
		FileName:    "product-photo.jpg",
		FileSize:    2048,
		ContentType: "image/jpeg",
	}

	mockProductRepo.On("FindByIDForTenant", ctx, tenantID, productID).Return(nil, shared.ErrNotFound)

	result, err := service.InitiateUpload(ctx, tenantID, req, &userID)

	assert.Error(t, err)
	assert.Nil(t, result)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "PRODUCT_NOT_FOUND", domainErr.Code)
	mockProductRepo.AssertExpectations(t)
}

func TestAttachmentService_InitiateUpload_AttachmentLimitExceeded(t *testing.T) {
	service, mockAttachmentRepo, mockProductRepo, _ := newTestAttachmentService()
	service.SetConfig(AttachmentServiceConfig{
		UploadURLExpiry:          15 * time.Minute,
		DownloadURLExpiry:        1 * time.Hour,
		MaxAttachmentsPerProduct: 10,
	})

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()
	product := createTestProduct(tenantID)
	userID := uuid.New()

	req := InitiateUploadRequest{
		ProductID:   productID,
		Type:        "gallery_image",
		FileName:    "product-photo.jpg",
		FileSize:    2048,
		ContentType: "image/jpeg",
	}

	mockProductRepo.On("FindByIDForTenant", ctx, tenantID, productID).Return(product, nil)
	mockAttachmentRepo.On("CountActiveByProduct", ctx, tenantID, productID).Return(int64(10), nil)

	result, err := service.InitiateUpload(ctx, tenantID, req, &userID)

	assert.Error(t, err)
	assert.Nil(t, result)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "ATTACHMENT_LIMIT_EXCEEDED", domainErr.Code)
	mockProductRepo.AssertExpectations(t)
	mockAttachmentRepo.AssertExpectations(t)
}

func TestAttachmentService_InitiateUpload_InvalidContentTypeForImage(t *testing.T) {
	service, mockAttachmentRepo, mockProductRepo, _ := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()
	product := createTestProduct(tenantID)
	userID := uuid.New()

	req := InitiateUploadRequest{
		ProductID:   productID,
		Type:        "main_image",
		FileName:    "document.pdf",
		FileSize:    2048,
		ContentType: "application/pdf",
	}

	mockProductRepo.On("FindByIDForTenant", ctx, tenantID, productID).Return(product, nil)
	mockAttachmentRepo.On("CountActiveByProduct", ctx, tenantID, productID).Return(int64(5), nil)

	result, err := service.InitiateUpload(ctx, tenantID, req, &userID)

	assert.Error(t, err)
	assert.Nil(t, result)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "INVALID_CONTENT_TYPE", domainErr.Code)
	mockProductRepo.AssertExpectations(t)
	mockAttachmentRepo.AssertExpectations(t)
}

func TestAttachmentService_InitiateUpload_MainImageAlreadyExists(t *testing.T) {
	service, mockAttachmentRepo, mockProductRepo, _ := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()
	product := createTestProduct(tenantID)
	existingMainImage := createActiveTestAttachment(tenantID, productID)
	userID := uuid.New()

	req := InitiateUploadRequest{
		ProductID:   productID,
		Type:        "main_image",
		FileName:    "new-main.jpg",
		FileSize:    2048,
		ContentType: "image/jpeg",
	}

	mockProductRepo.On("FindByIDForTenant", ctx, tenantID, productID).Return(product, nil)
	mockAttachmentRepo.On("CountActiveByProduct", ctx, tenantID, productID).Return(int64(5), nil)
	mockAttachmentRepo.On("FindMainImage", ctx, tenantID, productID).Return(existingMainImage, nil)

	result, err := service.InitiateUpload(ctx, tenantID, req, &userID)

	assert.Error(t, err)
	assert.Nil(t, result)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "MAIN_IMAGE_EXISTS", domainErr.Code)
	mockProductRepo.AssertExpectations(t)
	mockAttachmentRepo.AssertExpectations(t)
}

func TestAttachmentService_InitiateUpload_DisallowedContentType(t *testing.T) {
	service, mockAttachmentRepo, mockProductRepo, _ := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()
	product := createTestProduct(tenantID)
	userID := uuid.New()

	// Try to upload an executable - should be blocked
	req := InitiateUploadRequest{
		ProductID:   productID,
		Type:        "document",
		FileName:    "malware.exe",
		FileSize:    2048,
		ContentType: "application/x-msdownload",
	}

	mockProductRepo.On("FindByIDForTenant", ctx, tenantID, productID).Return(product, nil)
	mockAttachmentRepo.On("CountActiveByProduct", ctx, tenantID, productID).Return(int64(5), nil)

	result, err := service.InitiateUpload(ctx, tenantID, req, &userID)

	assert.Error(t, err)
	assert.Nil(t, result)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "DISALLOWED_CONTENT_TYPE", domainErr.Code)
	mockProductRepo.AssertExpectations(t)
	mockAttachmentRepo.AssertExpectations(t)
}

func TestAttachmentService_InitiateUpload_DocumentType_Success(t *testing.T) {
	service, mockAttachmentRepo, mockProductRepo, mockStorageService := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()
	product := createTestProduct(tenantID)
	userID := uuid.New()
	expiresAt := time.Now().Add(15 * time.Minute)

	req := InitiateUploadRequest{
		ProductID:   productID,
		Type:        "document",
		FileName:    "manual.pdf",
		FileSize:    5000000,
		ContentType: "application/pdf",
	}

	mockProductRepo.On("FindByIDForTenant", ctx, tenantID, productID).Return(product, nil)
	mockAttachmentRepo.On("CountActiveByProduct", ctx, tenantID, productID).Return(int64(5), nil)
	mockAttachmentRepo.On("Save", ctx, mock.AnythingOfType("*catalog.ProductAttachment")).Return(nil)
	mockStorageService.On("GenerateUploadURL", ctx, mock.AnythingOfType("string"), "application/pdf", mock.AnythingOfType("time.Duration")).Return("https://storage.example.com/upload?token=xyz", expiresAt, nil)

	result, err := service.InitiateUpload(ctx, tenantID, req, &userID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Note: StorageKey is intentionally not exposed in response for security reasons
	mockProductRepo.AssertExpectations(t)
	mockAttachmentRepo.AssertExpectations(t)
	mockStorageService.AssertExpectations(t)
}

// ============================================================================
// ConfirmUpload Tests
// ============================================================================

func TestAttachmentService_ConfirmUpload_Success(t *testing.T) {
	service, mockAttachmentRepo, _, mockStorageService := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()
	attachmentID := newAttachmentTestAttachmentID()
	attachment := createTestAttachment(tenantID, productID)
	expiresAt := time.Now().Add(1 * time.Hour)

	mockAttachmentRepo.On("FindByIDForTenant", ctx, tenantID, attachmentID).Return(attachment, nil)
	mockStorageService.On("ObjectExists", ctx, attachment.StorageKey).Return(true, nil)
	mockAttachmentRepo.On("GetMaxSortOrder", ctx, tenantID, productID).Return(2, nil)
	mockAttachmentRepo.On("Save", ctx, mock.AnythingOfType("*catalog.ProductAttachment")).Return(nil)
	mockStorageService.On("GenerateDownloadURL", ctx, attachment.StorageKey, mock.AnythingOfType("time.Duration")).Return("https://storage.example.com/download?token=xyz", expiresAt, nil)

	result, err := service.ConfirmUpload(ctx, tenantID, attachmentID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "active", result.Status)
	assert.Equal(t, 3, result.SortOrder)
	assert.Equal(t, "https://storage.example.com/download?token=xyz", result.URL)
	mockAttachmentRepo.AssertExpectations(t)
	mockStorageService.AssertExpectations(t)
}

func TestAttachmentService_ConfirmUpload_NotFound(t *testing.T) {
	service, mockAttachmentRepo, _, _ := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	attachmentID := newAttachmentTestAttachmentID()

	mockAttachmentRepo.On("FindByIDForTenant", ctx, tenantID, attachmentID).Return(nil, shared.ErrNotFound)

	result, err := service.ConfirmUpload(ctx, tenantID, attachmentID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, shared.ErrNotFound)
	mockAttachmentRepo.AssertExpectations(t)
}

func TestAttachmentService_ConfirmUpload_FileNotInStorage(t *testing.T) {
	service, mockAttachmentRepo, _, mockStorageService := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()
	attachmentID := newAttachmentTestAttachmentID()
	attachment := createTestAttachment(tenantID, productID)

	mockAttachmentRepo.On("FindByIDForTenant", ctx, tenantID, attachmentID).Return(attachment, nil)
	mockStorageService.On("ObjectExists", ctx, attachment.StorageKey).Return(false, nil)

	result, err := service.ConfirmUpload(ctx, tenantID, attachmentID)

	assert.Error(t, err)
	assert.Nil(t, result)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "UPLOAD_NOT_FOUND", domainErr.Code)
	mockAttachmentRepo.AssertExpectations(t)
	mockStorageService.AssertExpectations(t)
}

func TestAttachmentService_ConfirmUpload_AlreadyConfirmed(t *testing.T) {
	service, mockAttachmentRepo, _, mockStorageService := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()
	attachmentID := newAttachmentTestAttachmentID()
	attachment := createActiveTestAttachment(tenantID, productID) // Already active

	mockAttachmentRepo.On("FindByIDForTenant", ctx, tenantID, attachmentID).Return(attachment, nil)
	mockStorageService.On("ObjectExists", ctx, attachment.StorageKey).Return(true, nil)

	result, err := service.ConfirmUpload(ctx, tenantID, attachmentID)

	assert.Error(t, err)
	assert.Nil(t, result)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "ALREADY_CONFIRMED", domainErr.Code)
	mockAttachmentRepo.AssertExpectations(t)
	mockStorageService.AssertExpectations(t)
}

// ============================================================================
// GetByID Tests
// ============================================================================

func TestAttachmentService_GetByID_Success(t *testing.T) {
	service, mockAttachmentRepo, _, mockStorageService := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()
	attachmentID := newAttachmentTestAttachmentID()
	attachment := createActiveTestAttachment(tenantID, productID)
	expiresAt := time.Now().Add(1 * time.Hour)

	mockAttachmentRepo.On("FindByIDForTenant", ctx, tenantID, attachmentID).Return(attachment, nil)
	mockStorageService.On("GenerateDownloadURL", ctx, attachment.StorageKey, mock.AnythingOfType("time.Duration")).Return("https://storage.example.com/download", expiresAt, nil)

	result, err := service.GetByID(ctx, tenantID, attachmentID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, attachment.ID, result.ID)
	assert.Equal(t, "https://storage.example.com/download", result.URL)
	mockAttachmentRepo.AssertExpectations(t)
	mockStorageService.AssertExpectations(t)
}

func TestAttachmentService_GetByID_NotFound(t *testing.T) {
	service, mockAttachmentRepo, _, _ := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	attachmentID := newAttachmentTestAttachmentID()

	mockAttachmentRepo.On("FindByIDForTenant", ctx, tenantID, attachmentID).Return(nil, shared.ErrNotFound)

	result, err := service.GetByID(ctx, tenantID, attachmentID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, shared.ErrNotFound)
	mockAttachmentRepo.AssertExpectations(t)
}

// ============================================================================
// GetByProduct Tests
// ============================================================================

func TestAttachmentService_GetByProduct_Success(t *testing.T) {
	service, mockAttachmentRepo, mockProductRepo, mockStorageService := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()
	product := createTestProduct(tenantID)
	attachment := createActiveTestAttachment(tenantID, productID)
	attachments := []catalog.ProductAttachment{*attachment}
	expiresAt := time.Now().Add(1 * time.Hour)

	filter := AttachmentListFilter{
		Page:     1,
		PageSize: 20,
	}

	mockProductRepo.On("FindByIDForTenant", ctx, tenantID, productID).Return(product, nil)
	mockAttachmentRepo.On("FindByProduct", ctx, tenantID, productID, mock.AnythingOfType("shared.Filter")).Return(attachments, nil)
	mockAttachmentRepo.On("CountActiveByProduct", ctx, tenantID, productID).Return(int64(1), nil)
	mockStorageService.On("GenerateDownloadURL", ctx, attachment.StorageKey, mock.AnythingOfType("time.Duration")).Return("https://storage.example.com/download", expiresAt, nil)

	results, total, err := service.GetByProduct(ctx, tenantID, productID, filter)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, "https://storage.example.com/download", results[0].URL)
	mockProductRepo.AssertExpectations(t)
	mockAttachmentRepo.AssertExpectations(t)
	mockStorageService.AssertExpectations(t)
}

func TestAttachmentService_GetByProduct_ProductNotFound(t *testing.T) {
	service, _, mockProductRepo, _ := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()

	filter := AttachmentListFilter{}

	mockProductRepo.On("FindByIDForTenant", ctx, tenantID, productID).Return(nil, shared.ErrNotFound)

	results, total, err := service.GetByProduct(ctx, tenantID, productID, filter)

	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Equal(t, int64(0), total)
	assert.ErrorIs(t, err, shared.ErrNotFound)
	mockProductRepo.AssertExpectations(t)
}

// ============================================================================
// GetActiveByProduct Tests
// ============================================================================

func TestAttachmentService_GetActiveByProduct_Success(t *testing.T) {
	service, mockAttachmentRepo, _, mockStorageService := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()
	attachment := createActiveTestAttachment(tenantID, productID)
	attachments := []catalog.ProductAttachment{*attachment}
	expiresAt := time.Now().Add(1 * time.Hour)

	mockAttachmentRepo.On("FindActiveByProduct", ctx, tenantID, productID).Return(attachments, nil)
	mockStorageService.On("GenerateDownloadURL", ctx, attachment.StorageKey, mock.AnythingOfType("time.Duration")).Return("https://storage.example.com/download", expiresAt, nil)

	results, err := service.GetActiveByProduct(ctx, tenantID, productID)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "https://storage.example.com/download", results[0].URL)
	mockAttachmentRepo.AssertExpectations(t)
	mockStorageService.AssertExpectations(t)
}

// ============================================================================
// GetMainImage Tests
// ============================================================================

func TestAttachmentService_GetMainImage_Success(t *testing.T) {
	service, mockAttachmentRepo, _, mockStorageService := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()
	attachment := createActiveTestAttachment(tenantID, productID)
	attachment.SetAsMainImage()
	expiresAt := time.Now().Add(1 * time.Hour)

	mockAttachmentRepo.On("FindMainImage", ctx, tenantID, productID).Return(attachment, nil)
	mockStorageService.On("GenerateDownloadURL", ctx, attachment.StorageKey, mock.AnythingOfType("time.Duration")).Return("https://storage.example.com/download", expiresAt, nil)

	result, err := service.GetMainImage(ctx, tenantID, productID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "main_image", result.Type)
	mockAttachmentRepo.AssertExpectations(t)
	mockStorageService.AssertExpectations(t)
}

func TestAttachmentService_GetMainImage_NotFound(t *testing.T) {
	service, mockAttachmentRepo, _, _ := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()

	mockAttachmentRepo.On("FindMainImage", ctx, tenantID, productID).Return(nil, shared.ErrNotFound)

	result, err := service.GetMainImage(ctx, tenantID, productID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, shared.ErrNotFound)
	mockAttachmentRepo.AssertExpectations(t)
}

// ============================================================================
// Delete Tests
// ============================================================================

func TestAttachmentService_Delete_Success(t *testing.T) {
	service, mockAttachmentRepo, _, _ := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()
	attachmentID := newAttachmentTestAttachmentID()
	attachment := createActiveTestAttachment(tenantID, productID)

	mockAttachmentRepo.On("FindByIDForTenant", ctx, tenantID, attachmentID).Return(attachment, nil)
	mockAttachmentRepo.On("Save", ctx, mock.AnythingOfType("*catalog.ProductAttachment")).Return(nil)

	err := service.Delete(ctx, tenantID, attachmentID)

	assert.NoError(t, err)
	mockAttachmentRepo.AssertExpectations(t)
}

func TestAttachmentService_Delete_NotFound(t *testing.T) {
	service, mockAttachmentRepo, _, _ := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	attachmentID := newAttachmentTestAttachmentID()

	mockAttachmentRepo.On("FindByIDForTenant", ctx, tenantID, attachmentID).Return(nil, shared.ErrNotFound)

	err := service.Delete(ctx, tenantID, attachmentID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, shared.ErrNotFound)
	mockAttachmentRepo.AssertExpectations(t)
}

func TestAttachmentService_Delete_AlreadyDeleted(t *testing.T) {
	service, mockAttachmentRepo, _, _ := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()
	attachmentID := newAttachmentTestAttachmentID()
	attachment := createActiveTestAttachment(tenantID, productID)
	attachment.Delete() // Already deleted

	mockAttachmentRepo.On("FindByIDForTenant", ctx, tenantID, attachmentID).Return(attachment, nil)

	err := service.Delete(ctx, tenantID, attachmentID)

	assert.Error(t, err)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "ALREADY_DELETED", domainErr.Code)
	mockAttachmentRepo.AssertExpectations(t)
}

// ============================================================================
// PermanentDelete Tests
// ============================================================================

func TestAttachmentService_PermanentDelete_Success(t *testing.T) {
	service, mockAttachmentRepo, _, mockStorageService := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()
	attachmentID := newAttachmentTestAttachmentID()
	attachment := createActiveTestAttachment(tenantID, productID)

	mockAttachmentRepo.On("FindByIDForTenant", ctx, tenantID, attachmentID).Return(attachment, nil)
	mockStorageService.On("DeleteObject", ctx, attachment.StorageKey).Return(nil)
	mockAttachmentRepo.On("DeleteForTenant", ctx, tenantID, attachmentID).Return(nil)

	err := service.PermanentDelete(ctx, tenantID, attachmentID)

	assert.NoError(t, err)
	mockAttachmentRepo.AssertExpectations(t)
	mockStorageService.AssertExpectations(t)
}

// ============================================================================
// SetAsMainImage Tests
// ============================================================================

func TestAttachmentService_SetAsMainImage_Success(t *testing.T) {
	service, mockAttachmentRepo, _, mockStorageService := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()
	attachmentID := newAttachmentTestAttachmentID()
	attachment := createActiveTestAttachment(tenantID, productID)
	expiresAt := time.Now().Add(1 * time.Hour)

	mockAttachmentRepo.On("FindByIDForTenant", ctx, tenantID, attachmentID).Return(attachment, nil)
	mockAttachmentRepo.On("FindMainImage", ctx, tenantID, productID).Return(nil, shared.ErrNotFound)
	mockAttachmentRepo.On("SaveBatch", ctx, mock.AnythingOfType("[]*catalog.ProductAttachment")).Return(nil)
	mockStorageService.On("GenerateDownloadURL", ctx, attachment.StorageKey, mock.AnythingOfType("time.Duration")).Return("https://storage.example.com/download", expiresAt, nil)

	result, err := service.SetAsMainImage(ctx, tenantID, attachmentID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "main_image", result.Type)
	mockAttachmentRepo.AssertExpectations(t)
	mockStorageService.AssertExpectations(t)
}

func TestAttachmentService_SetAsMainImage_DemoteExisting(t *testing.T) {
	service, mockAttachmentRepo, _, mockStorageService := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()
	attachmentID := newAttachmentTestAttachmentID()
	existingMainID := uuid.New()
	attachment := createActiveTestAttachment(tenantID, productID)
	existingMain := createActiveTestAttachment(tenantID, productID)
	existingMain.SetAsMainImage()
	existingMain.ID = existingMainID
	expiresAt := time.Now().Add(1 * time.Hour)

	mockAttachmentRepo.On("FindByIDForTenant", ctx, tenantID, attachmentID).Return(attachment, nil)
	mockAttachmentRepo.On("FindMainImage", ctx, tenantID, productID).Return(existingMain, nil)
	// SaveBatch is called once with both attachments (atomic operation)
	mockAttachmentRepo.On("SaveBatch", ctx, mock.AnythingOfType("[]*catalog.ProductAttachment")).Return(nil)
	mockStorageService.On("GenerateDownloadURL", ctx, attachment.StorageKey, mock.AnythingOfType("time.Duration")).Return("https://storage.example.com/download", expiresAt, nil)

	result, err := service.SetAsMainImage(ctx, tenantID, attachmentID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "main_image", result.Type)
	mockAttachmentRepo.AssertExpectations(t)
	mockStorageService.AssertExpectations(t)
}

func TestAttachmentService_SetAsMainImage_NotAnImage(t *testing.T) {
	service, mockAttachmentRepo, _, _ := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()
	attachmentID := newAttachmentTestAttachmentID()
	userID := uuid.New()
	attachment, _ := catalog.NewProductAttachment(
		tenantID,
		productID,
		catalog.AttachmentTypeDocument,
		"manual.pdf",
		5000,
		"application/pdf",
		"tenants/test/products/test/attachments/manual.pdf",
		&userID,
	)
	attachment.Confirm()

	mockAttachmentRepo.On("FindByIDForTenant", ctx, tenantID, attachmentID).Return(attachment, nil)

	result, err := service.SetAsMainImage(ctx, tenantID, attachmentID)

	assert.Error(t, err)
	assert.Nil(t, result)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "NOT_AN_IMAGE", domainErr.Code)
	mockAttachmentRepo.AssertExpectations(t)
}

// ============================================================================
// ReorderAttachments Tests
// ============================================================================

func TestAttachmentService_ReorderAttachments_Success(t *testing.T) {
	service, mockAttachmentRepo, mockProductRepo, mockStorageService := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()
	product := createTestProduct(tenantID)

	att1 := createActiveTestAttachment(tenantID, productID)
	att1.ID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	att2 := createActiveTestAttachment(tenantID, productID)
	att2.ID = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	att3 := createActiveTestAttachment(tenantID, productID)
	att3.ID = uuid.MustParse("33333333-3333-3333-3333-333333333333")

	attachments := []catalog.ProductAttachment{*att1, *att2, *att3}
	newOrder := []uuid.UUID{att3.ID, att1.ID, att2.ID}
	expiresAt := time.Now().Add(1 * time.Hour)

	mockProductRepo.On("FindByIDForTenant", ctx, tenantID, productID).Return(product, nil)
	mockAttachmentRepo.On("FindByIDs", ctx, tenantID, newOrder).Return(attachments, nil)
	mockAttachmentRepo.On("SaveBatch", ctx, mock.AnythingOfType("[]*catalog.ProductAttachment")).Return(nil)
	mockStorageService.On("GenerateDownloadURL", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return("https://storage.example.com/download", expiresAt, nil).Times(3)

	results, err := service.ReorderAttachments(ctx, tenantID, productID, newOrder)

	assert.NoError(t, err)
	assert.Len(t, results, 3)
	mockProductRepo.AssertExpectations(t)
	mockAttachmentRepo.AssertExpectations(t)
	mockStorageService.AssertExpectations(t)
}

func TestAttachmentService_ReorderAttachments_ProductNotFound(t *testing.T) {
	service, _, mockProductRepo, _ := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()
	newOrder := []uuid.UUID{uuid.New()}

	mockProductRepo.On("FindByIDForTenant", ctx, tenantID, productID).Return(nil, shared.ErrNotFound)

	results, err := service.ReorderAttachments(ctx, tenantID, productID, newOrder)

	assert.Error(t, err)
	assert.Nil(t, results)
	assert.ErrorIs(t, err, shared.ErrNotFound)
	mockProductRepo.AssertExpectations(t)
}

func TestAttachmentService_ReorderAttachments_AttachmentsNotFound(t *testing.T) {
	service, mockAttachmentRepo, mockProductRepo, _ := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()
	product := createTestProduct(tenantID)
	newOrder := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	// Only return 2 of 3 attachments
	attachments := []catalog.ProductAttachment{
		*createActiveTestAttachment(tenantID, productID),
		*createActiveTestAttachment(tenantID, productID),
	}

	mockProductRepo.On("FindByIDForTenant", ctx, tenantID, productID).Return(product, nil)
	mockAttachmentRepo.On("FindByIDs", ctx, tenantID, newOrder).Return(attachments, nil)

	results, err := service.ReorderAttachments(ctx, tenantID, productID, newOrder)

	assert.Error(t, err)
	assert.Nil(t, results)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "ATTACHMENTS_NOT_FOUND", domainErr.Code)
	mockProductRepo.AssertExpectations(t)
	mockAttachmentRepo.AssertExpectations(t)
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestIsImageContentType(t *testing.T) {
	tests := []struct {
		contentType string
		expected    bool
	}{
		{"image/jpeg", true},
		{"image/png", true},
		{"image/gif", true},
		{"image/webp", true},
		{"IMAGE/JPEG", true},
		{"application/pdf", false},
		{"text/plain", false},
		{"video/mp4", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			result := isImageContentType(tt.contentType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateStorageKey(t *testing.T) {
	service := &AttachmentService{}
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()

	key := service.generateStorageKey(tenantID, productID, "test-image.jpg")

	assert.Contains(t, key, "tenants/"+tenantID.String())
	assert.Contains(t, key, "products/"+productID.String())
	assert.Contains(t, key, "attachments/")
	assert.Contains(t, key, ".jpg")
}

// ============================================================================
// DTO Conversion Tests
// ============================================================================

func TestToAttachmentResponse(t *testing.T) {
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()
	attachment := createActiveTestAttachment(tenantID, productID)

	response := ToAttachmentResponse(attachment)

	assert.Equal(t, attachment.ID, response.ID)
	assert.Equal(t, attachment.TenantID, response.TenantID)
	assert.Equal(t, attachment.ProductID, response.ProductID)
	assert.Equal(t, string(attachment.Type), response.Type)
	assert.Equal(t, string(attachment.Status), response.Status)
	assert.Equal(t, attachment.FileName, response.FileName)
	assert.Equal(t, attachment.FileSize, response.FileSize)
	assert.Equal(t, attachment.ContentType, response.ContentType)
}

func TestToAttachmentListResponses(t *testing.T) {
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()
	attachments := []catalog.ProductAttachment{
		*createActiveTestAttachment(tenantID, productID),
		*createActiveTestAttachment(tenantID, productID),
	}

	responses := ToAttachmentListResponses(attachments)

	assert.Len(t, responses, 2)
	assert.Equal(t, attachments[0].FileName, responses[0].FileName)
	assert.Equal(t, attachments[1].FileName, responses[1].FileName)
}

func TestAttachmentResponse_EnrichWithURLs(t *testing.T) {
	response := AttachmentResponse{}
	response.EnrichWithURLs("https://download.url", "https://thumbnail.url")

	assert.Equal(t, "https://download.url", response.URL)
	assert.Equal(t, "https://thumbnail.url", response.ThumbnailURL)
}

func TestAttachmentListResponse_EnrichWithURLs(t *testing.T) {
	response := AttachmentListResponse{}
	response.EnrichWithURLs("https://download.url", "https://thumbnail.url")

	assert.Equal(t, "https://download.url", response.URL)
	assert.Equal(t, "https://thumbnail.url", response.ThumbnailURL)
}

func TestIsAllowedContentType(t *testing.T) {
	tests := []struct {
		contentType string
		expected    bool
	}{
		// Allowed image types
		{"image/jpeg", true},
		{"image/png", true},
		{"image/gif", true},
		{"image/webp", true},
		{"image/bmp", true},
		{"image/tiff", true},
		{"IMAGE/JPEG", true}, // Case insensitive
		// Allowed document types
		{"application/pdf", true},
		{"application/msword", true},
		{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", true},
		{"application/vnd.ms-excel", true},
		{"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", true},
		// Allowed text types
		{"text/plain", true},
		{"text/csv", true},
		// Disallowed dangerous types
		{"application/x-msdownload", false}, // Executables
		{"application/x-executable", false},
		{"application/javascript", false},
		{"text/html", false},
		{"application/x-shellscript", false},
		{"", false},
		// SECURITY: SVG is disallowed due to XSS risk
		{"image/svg+xml", false},
		{"IMAGE/SVG+XML", false}, // Case insensitive check
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			result := isAllowedContentType(tt.contentType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================================
// SVG XSS Security Tests
// ============================================================================

func TestAttachmentService_InitiateUpload_SVG_XSS_Blocked(t *testing.T) {
	service, mockAttachmentRepo, mockProductRepo, _ := newTestAttachmentService()

	ctx := context.Background()
	tenantID := newAttachmentTestTenantID()
	productID := newAttachmentTestProductID()
	product := createTestProduct(tenantID)
	userID := uuid.New()

	mockProductRepo.On("FindByIDForTenant", ctx, tenantID, productID).Return(product, nil)
	mockAttachmentRepo.On("CountActiveByProduct", ctx, tenantID, productID).Return(int64(5), nil)

	// Test various SVG content type variations that should all be blocked
	svgContentTypes := []string{
		"image/svg+xml",
		"IMAGE/SVG+XML",
		"image/SVG+xml",
	}

	for _, contentType := range svgContentTypes {
		t.Run("blocks_"+contentType, func(t *testing.T) {
			req := InitiateUploadRequest{
				ProductID:   productID,
				Type:        "gallery_image",
				FileName:    "malicious.svg",
				FileSize:    1024,
				ContentType: contentType,
			}

			result, err := service.InitiateUpload(ctx, tenantID, req, &userID)

			assert.Error(t, err)
			assert.Nil(t, result)
			var domainErr *shared.DomainError
			assert.ErrorAs(t, err, &domainErr)
			assert.Equal(t, "DISALLOWED_CONTENT_TYPE", domainErr.Code)
		})
	}
}

func TestAttachmentService_SVG_XSS_Payloads_Are_Blocked(t *testing.T) {
	// This test documents various SVG XSS payloads that are now blocked
	// by removing SVG from the allowed content types
	xssPayloads := []struct {
		name        string
		description string
	}{
		{"script_tag", "SVG with <script>alert('XSS')</script>"},
		{"onload", "SVG with onload event: <svg onload=alert('XSS')>"},
		{"onerror_image", "SVG with <image xlink:href=x onerror=alert('XSS')>"},
		{"animate_xlink", "SVG with <animate xlink:href=#xss attributeName=href values=javascript:alert(1)>"},
		{"use_xlink", "SVG with <use xlink:href=data:image/svg+xml,<svg onload=alert(1)>>"},
		{"foreignObject", "SVG with <foreignObject> containing HTML/scripts"},
		{"set_event", "SVG with <set attributeName=onmouseover to=alert(1)>"},
		{"handler_attribute", "SVG with handler attribute exploits"},
		{"xml_entity", "SVG with XML entity expansion attacks"},
	}

	for _, payload := range xssPayloads {
		t.Run("svg_xss_"+payload.name+"_is_blocked", func(t *testing.T) {
			// All these attack vectors are now blocked because SVG uploads are rejected
			result := isAllowedContentType("image/svg+xml")
			assert.False(t, result, "SVG should be blocked to prevent: %s", payload.description)
		})
	}
}
