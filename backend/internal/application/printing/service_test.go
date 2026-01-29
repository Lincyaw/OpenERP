package printing_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/erp/backend/internal/application/printing"
	domain "github.com/erp/backend/internal/domain/printing"
	"github.com/erp/backend/internal/domain/shared"
	infra "github.com/erp/backend/internal/infrastructure/printing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// =============================================================================
// Mock Implementations
// =============================================================================

type MockTemplateRepository struct {
	mock.Mock
}

func (m *MockTemplateRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.PrintTemplate, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PrintTemplate), args.Error(1)
}

func (m *MockTemplateRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*domain.PrintTemplate, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PrintTemplate), args.Error(1)
}

func (m *MockTemplateRepository) FindAll(ctx context.Context, filter shared.Filter) ([]domain.PrintTemplate, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.PrintTemplate), args.Error(1)
}

func (m *MockTemplateRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]domain.PrintTemplate, error) {
	args := m.Called(ctx, tenantID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.PrintTemplate), args.Error(1)
}

func (m *MockTemplateRepository) FindByDocType(ctx context.Context, tenantID uuid.UUID, docType domain.DocType) ([]domain.PrintTemplate, error) {
	args := m.Called(ctx, tenantID, docType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.PrintTemplate), args.Error(1)
}

func (m *MockTemplateRepository) FindDefault(ctx context.Context, tenantID uuid.UUID, docType domain.DocType) (*domain.PrintTemplate, error) {
	args := m.Called(ctx, tenantID, docType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PrintTemplate), args.Error(1)
}

func (m *MockTemplateRepository) FindActiveByDocType(ctx context.Context, tenantID uuid.UUID, docType domain.DocType) ([]domain.PrintTemplate, error) {
	args := m.Called(ctx, tenantID, docType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.PrintTemplate), args.Error(1)
}

func (m *MockTemplateRepository) Save(ctx context.Context, template *domain.PrintTemplate) error {
	args := m.Called(ctx, template)
	return args.Error(0)
}

func (m *MockTemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTemplateRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTemplateRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTemplateRepository) ExistsByDocTypeAndName(ctx context.Context, tenantID uuid.UUID, docType domain.DocType, name string, excludeID *uuid.UUID) (bool, error) {
	args := m.Called(ctx, tenantID, docType, name, excludeID)
	return args.Bool(0), args.Error(1)
}

func (m *MockTemplateRepository) ClearDefaultForDocType(ctx context.Context, tenantID uuid.UUID, docType domain.DocType) error {
	args := m.Called(ctx, tenantID, docType)
	return args.Error(0)
}

type MockJobRepository struct {
	mock.Mock
}

func (m *MockJobRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.PrintJob, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PrintJob), args.Error(1)
}

func (m *MockJobRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*domain.PrintJob, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PrintJob), args.Error(1)
}

func (m *MockJobRepository) FindAll(ctx context.Context, filter shared.Filter) ([]domain.PrintJob, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.PrintJob), args.Error(1)
}

func (m *MockJobRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]domain.PrintJob, error) {
	args := m.Called(ctx, tenantID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.PrintJob), args.Error(1)
}

func (m *MockJobRepository) FindByDocument(ctx context.Context, tenantID uuid.UUID, docType domain.DocType, documentID uuid.UUID) ([]domain.PrintJob, error) {
	args := m.Called(ctx, tenantID, docType, documentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.PrintJob), args.Error(1)
}

func (m *MockJobRepository) FindRecent(ctx context.Context, tenantID uuid.UUID, days int, limit int) ([]domain.PrintJob, error) {
	args := m.Called(ctx, tenantID, days, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.PrintJob), args.Error(1)
}

func (m *MockJobRepository) FindPending(ctx context.Context, limit int) ([]domain.PrintJob, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.PrintJob), args.Error(1)
}

func (m *MockJobRepository) Save(ctx context.Context, job *domain.PrintJob) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

func (m *MockJobRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockJobRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockJobRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockJobRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status domain.JobStatus) (int64, error) {
	args := m.Called(ctx, tenantID, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockJobRepository) DeleteOlderThan(ctx context.Context, days int) (int64, error) {
	args := m.Called(ctx, days)
	return args.Get(0).(int64), args.Error(1)
}

type MockPDFRenderer struct {
	mock.Mock
}

func (m *MockPDFRenderer) Render(ctx context.Context, req *infra.RenderRequest) (*infra.RenderResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*infra.RenderResult), args.Error(1)
}

func (m *MockPDFRenderer) Close() error {
	args := m.Called()
	return args.Error(0)
}

type MockPDFStorage struct {
	mock.Mock
}

func (m *MockPDFStorage) Store(ctx context.Context, req *infra.StoreRequest) (*infra.StoreResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*infra.StoreResult), args.Error(1)
}

func (m *MockPDFStorage) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	args := m.Called(ctx, path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockPDFStorage) Delete(ctx context.Context, path string) error {
	args := m.Called(ctx, path)
	return args.Error(0)
}

func (m *MockPDFStorage) CleanupOlderThan(ctx context.Context, age time.Duration) (int, error) {
	args := m.Called(ctx, age)
	return args.Int(0), args.Error(1)
}

func (m *MockPDFStorage) GetURL(path string) string {
	args := m.Called(path)
	return args.String(0)
}

// =============================================================================
// Helper Functions
// =============================================================================

func newTestService(
	templateRepo *MockTemplateRepository,
	jobRepo *MockJobRepository,
	renderer *MockPDFRenderer,
	storage *MockPDFStorage,
) *printing.PrintService {
	templateEngine := infra.NewTemplateEngine()
	logger := zap.NewNop()
	return printing.NewPrintService(templateRepo, jobRepo, templateEngine, renderer, storage, nil, logger)
}

func createTestTemplate(tenantID uuid.UUID, docType domain.DocType, name string) *domain.PrintTemplate {
	template, _ := domain.NewPrintTemplate(tenantID, docType, name, "<html>{{.Name}}</html>", domain.PaperSizeA4)
	return template
}

// =============================================================================
// Template Tests
// =============================================================================

func TestCreateTemplate_Success(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	templateRepo := new(MockTemplateRepository)
	jobRepo := new(MockJobRepository)
	renderer := new(MockPDFRenderer)
	storage := new(MockPDFStorage)

	templateRepo.On("ExistsByDocTypeAndName", ctx, tenantID, domain.DocTypeSalesOrder, "Test Template", (*uuid.UUID)(nil)).Return(false, nil)
	templateRepo.On("Save", ctx, mock.AnythingOfType("*printing.PrintTemplate")).Return(nil)

	service := newTestService(templateRepo, jobRepo, renderer, storage)

	req := printing.CreateTemplateRequest{
		DocumentType: "SALES_ORDER",
		Name:         "Test Template",
		Content:      "<html>{{.Name}}</html>",
		PaperSize:    "A4",
	}

	result, err := service.CreateTemplate(ctx, tenantID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Test Template", result.Name)
	assert.Equal(t, "SALES_ORDER", result.DocumentType)
	templateRepo.AssertExpectations(t)
}

func TestCreateTemplate_DuplicateName(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	templateRepo := new(MockTemplateRepository)
	jobRepo := new(MockJobRepository)
	renderer := new(MockPDFRenderer)
	storage := new(MockPDFStorage)

	templateRepo.On("ExistsByDocTypeAndName", ctx, tenantID, domain.DocTypeSalesOrder, "Test Template", (*uuid.UUID)(nil)).Return(true, nil)

	service := newTestService(templateRepo, jobRepo, renderer, storage)

	req := printing.CreateTemplateRequest{
		DocumentType: "SALES_ORDER",
		Name:         "Test Template",
		Content:      "<html>{{.Name}}</html>",
		PaperSize:    "A4",
	}

	result, err := service.CreateTemplate(ctx, tenantID, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "already exists")
	templateRepo.AssertExpectations(t)
}

func TestGetTemplate_Success(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	templateID := uuid.New()

	templateRepo := new(MockTemplateRepository)
	jobRepo := new(MockJobRepository)
	renderer := new(MockPDFRenderer)
	storage := new(MockPDFStorage)

	template := createTestTemplate(tenantID, domain.DocTypeSalesOrder, "Test Template")
	template.ID = templateID

	templateRepo.On("FindByIDForTenant", ctx, tenantID, templateID).Return(template, nil)

	service := newTestService(templateRepo, jobRepo, renderer, storage)

	result, err := service.GetTemplate(ctx, tenantID, templateID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, templateID.String(), result.ID)
	templateRepo.AssertExpectations(t)
}

func TestGetTemplate_NotFound(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	templateID := uuid.New()

	templateRepo := new(MockTemplateRepository)
	jobRepo := new(MockJobRepository)
	renderer := new(MockPDFRenderer)
	storage := new(MockPDFStorage)

	templateRepo.On("FindByIDForTenant", ctx, tenantID, templateID).Return(nil, shared.ErrNotFound)

	service := newTestService(templateRepo, jobRepo, renderer, storage)

	result, err := service.GetTemplate(ctx, tenantID, templateID)

	assert.Error(t, err)
	assert.Nil(t, result)
	templateRepo.AssertExpectations(t)
}

func TestListTemplates_Success(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	templateRepo := new(MockTemplateRepository)
	jobRepo := new(MockJobRepository)
	renderer := new(MockPDFRenderer)
	storage := new(MockPDFStorage)

	templates := []domain.PrintTemplate{
		*createTestTemplate(tenantID, domain.DocTypeSalesOrder, "Template 1"),
		*createTestTemplate(tenantID, domain.DocTypeSalesOrder, "Template 2"),
	}

	templateRepo.On("FindAllForTenant", ctx, tenantID, mock.AnythingOfType("shared.Filter")).Return(templates, nil)
	templateRepo.On("CountForTenant", ctx, tenantID, mock.AnythingOfType("shared.Filter")).Return(int64(2), nil)

	service := newTestService(templateRepo, jobRepo, renderer, storage)

	req := printing.ListTemplatesRequest{
		Page:     1,
		PageSize: 20,
	}

	result, err := service.ListTemplates(ctx, tenantID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, int64(2), result.Total)
	templateRepo.AssertExpectations(t)
}

func TestDeleteTemplate_Success(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	templateID := uuid.New()

	templateRepo := new(MockTemplateRepository)
	jobRepo := new(MockJobRepository)
	renderer := new(MockPDFRenderer)
	storage := new(MockPDFStorage)

	template := createTestTemplate(tenantID, domain.DocTypeSalesOrder, "Test Template")
	template.ID = templateID

	templateRepo.On("FindByIDForTenant", ctx, tenantID, templateID).Return(template, nil)
	templateRepo.On("Delete", ctx, templateID).Return(nil)

	service := newTestService(templateRepo, jobRepo, renderer, storage)

	err := service.DeleteTemplate(ctx, tenantID, templateID)

	assert.NoError(t, err)
	templateRepo.AssertExpectations(t)
}

func TestDeleteTemplate_CannotDeleteDefault(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	templateID := uuid.New()

	templateRepo := new(MockTemplateRepository)
	jobRepo := new(MockJobRepository)
	renderer := new(MockPDFRenderer)
	storage := new(MockPDFStorage)

	template := createTestTemplate(tenantID, domain.DocTypeSalesOrder, "Test Template")
	template.ID = templateID
	_ = template.SetAsDefault()

	templateRepo.On("FindByIDForTenant", ctx, tenantID, templateID).Return(template, nil)

	service := newTestService(templateRepo, jobRepo, renderer, storage)

	err := service.DeleteTemplate(ctx, tenantID, templateID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "default")
	templateRepo.AssertExpectations(t)
}

func TestSetDefaultTemplate_Success(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	templateID := uuid.New()

	templateRepo := new(MockTemplateRepository)
	jobRepo := new(MockJobRepository)
	renderer := new(MockPDFRenderer)
	storage := new(MockPDFStorage)

	template := createTestTemplate(tenantID, domain.DocTypeSalesOrder, "Test Template")
	template.ID = templateID

	templateRepo.On("FindByIDForTenant", ctx, tenantID, templateID).Return(template, nil)
	templateRepo.On("ClearDefaultForDocType", ctx, tenantID, domain.DocTypeSalesOrder).Return(nil)
	templateRepo.On("Save", ctx, mock.AnythingOfType("*printing.PrintTemplate")).Return(nil)

	service := newTestService(templateRepo, jobRepo, renderer, storage)

	result, err := service.SetDefaultTemplate(ctx, tenantID, templateID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsDefault)
	templateRepo.AssertExpectations(t)
}

// =============================================================================
// Print Job Tests
// =============================================================================

func TestGetJob_Success(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	jobID := uuid.New()
	templateID := uuid.New()
	documentID := uuid.New()
	userID := uuid.New()

	templateRepo := new(MockTemplateRepository)
	jobRepo := new(MockJobRepository)
	renderer := new(MockPDFRenderer)
	storage := new(MockPDFStorage)

	job, _ := domain.NewPrintJob(tenantID, templateID, domain.DocTypeSalesOrder, documentID, "SO-001", userID)
	job.ID = jobID

	jobRepo.On("FindByIDForTenant", ctx, tenantID, jobID).Return(job, nil)

	service := newTestService(templateRepo, jobRepo, renderer, storage)

	result, err := service.GetJob(ctx, tenantID, jobID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, jobID.String(), result.ID)
	jobRepo.AssertExpectations(t)
}

func TestGetJob_NotFound(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	jobID := uuid.New()

	templateRepo := new(MockTemplateRepository)
	jobRepo := new(MockJobRepository)
	renderer := new(MockPDFRenderer)
	storage := new(MockPDFStorage)

	jobRepo.On("FindByIDForTenant", ctx, tenantID, jobID).Return(nil, shared.ErrNotFound)

	service := newTestService(templateRepo, jobRepo, renderer, storage)

	result, err := service.GetJob(ctx, tenantID, jobID)

	assert.Error(t, err)
	assert.Nil(t, result)
	jobRepo.AssertExpectations(t)
}

func TestListJobs_Success(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	templateID := uuid.New()
	documentID := uuid.New()
	userID := uuid.New()

	templateRepo := new(MockTemplateRepository)
	jobRepo := new(MockJobRepository)
	renderer := new(MockPDFRenderer)
	storage := new(MockPDFStorage)

	job1, _ := domain.NewPrintJob(tenantID, templateID, domain.DocTypeSalesOrder, documentID, "SO-001", userID)
	job2, _ := domain.NewPrintJob(tenantID, templateID, domain.DocTypeSalesOrder, uuid.New(), "SO-002", userID)

	jobs := []domain.PrintJob{*job1, *job2}

	jobRepo.On("FindAllForTenant", ctx, tenantID, mock.AnythingOfType("shared.Filter")).Return(jobs, nil)
	jobRepo.On("CountForTenant", ctx, tenantID, mock.AnythingOfType("shared.Filter")).Return(int64(2), nil)

	service := newTestService(templateRepo, jobRepo, renderer, storage)

	req := printing.ListJobsRequest{
		Page:     1,
		PageSize: 20,
	}

	result, err := service.ListJobs(ctx, tenantID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, int64(2), result.Total)
	jobRepo.AssertExpectations(t)
}

// =============================================================================
// Reference Data Tests
// =============================================================================

func TestGetDocumentTypes(t *testing.T) {
	templateRepo := new(MockTemplateRepository)
	jobRepo := new(MockJobRepository)
	renderer := new(MockPDFRenderer)
	storage := new(MockPDFStorage)

	service := newTestService(templateRepo, jobRepo, renderer, storage)

	result := service.GetDocumentTypes()

	assert.NotNil(t, result)
	assert.Greater(t, len(result), 0)

	// Check that SALES_ORDER is in the list
	found := false
	for _, dt := range result {
		if dt.Code == "SALES_ORDER" {
			found = true
			assert.Equal(t, "销售订单", dt.DisplayName)
			break
		}
	}
	assert.True(t, found, "SALES_ORDER should be in document types")
}

func TestGetPaperSizes(t *testing.T) {
	templateRepo := new(MockTemplateRepository)
	jobRepo := new(MockJobRepository)
	renderer := new(MockPDFRenderer)
	storage := new(MockPDFStorage)

	service := newTestService(templateRepo, jobRepo, renderer, storage)

	result := service.GetPaperSizes()

	assert.NotNil(t, result)
	assert.Greater(t, len(result), 0)

	// Check that A4 is in the list
	found := false
	for _, ps := range result {
		if ps.Code == "A4" {
			found = true
			assert.Equal(t, 210, ps.Width)
			assert.Equal(t, 297, ps.Height)
			break
		}
	}
	assert.True(t, found, "A4 should be in paper sizes")
}
