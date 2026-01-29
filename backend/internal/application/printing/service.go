package printing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/printing"
	"github.com/erp/backend/internal/domain/shared"
	infra "github.com/erp/backend/internal/infrastructure/printing"
	"github.com/erp/backend/internal/infrastructure/printing/providers"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PrintService handles printing-related business operations
type PrintService struct {
	templateStore  *infra.TemplateStore
	jobRepo        printing.PrintJobRepository
	templateEngine *infra.TemplateEngine
	pdfRenderer    infra.PDFRenderer
	pdfStorage     infra.PDFStorage
	dataProviders  *providers.DataProviderRegistry
	logger         *zap.Logger
}

// NewPrintService creates a new PrintService
func NewPrintService(
	templateStore *infra.TemplateStore,
	jobRepo printing.PrintJobRepository,
	templateEngine *infra.TemplateEngine,
	pdfRenderer infra.PDFRenderer,
	pdfStorage infra.PDFStorage,
	dataProviders *providers.DataProviderRegistry,
	logger *zap.Logger,
) *PrintService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &PrintService{
		templateStore:  templateStore,
		jobRepo:        jobRepo,
		templateEngine: templateEngine,
		pdfRenderer:    pdfRenderer,
		pdfStorage:     pdfStorage,
		dataProviders:  dataProviders,
		logger:         logger,
	}
}

// =============================================================================
// Template Query Operations (Read-only, from static templates)
// =============================================================================

// GetTemplatesByDocType retrieves all templates for a document type
func (s *PrintService) GetTemplatesByDocType(ctx context.Context, docType string) ([]TemplateResponse, error) {
	dt := printing.DocType(docType)
	if !dt.IsValid() {
		return nil, shared.NewDomainError("INVALID_INPUT", "Invalid document type")
	}

	templates := s.templateStore.GetByDocType(dt)
	result := make([]TemplateResponse, len(templates))
	for i, t := range templates {
		result[i] = staticTemplateToResponse(&t)
	}

	return result, nil
}

// GetTemplate retrieves a template by ID
func (s *PrintService) GetTemplate(ctx context.Context, templateID string) (*TemplateResponse, error) {
	template := s.templateStore.GetByID(templateID)
	if template == nil {
		return nil, shared.NewDomainError("NOT_FOUND", "Template not found")
	}

	resp := staticTemplateToResponse(template)
	return &resp, nil
}

// =============================================================================
// Print Preview and PDF Generation
// =============================================================================

// PreviewDocument generates HTML preview for a document
func (s *PrintService) PreviewDocument(ctx context.Context, tenantID uuid.UUID, req PreviewRequest) (*PreviewResponse, error) {
	// Validate document type
	docType := printing.DocType(req.DocumentType)
	if !docType.IsValid() {
		return nil, shared.NewDomainError("INVALID_INPUT", "Invalid document type")
	}

	// Get template
	var template *infra.StaticTemplate
	if req.TemplateID != nil {
		template = s.templateStore.GetByID(req.TemplateID.String())
		if template == nil {
			return nil, shared.NewDomainError("NOT_FOUND", "Template not found")
		}
	} else {
		// Use default template for the document type
		template = s.templateStore.GetDefault(docType)
		if template == nil {
			return nil, shared.NewDomainError("NOT_FOUND", "No default template found for this document type")
		}
	}

	// Load data from provider if not provided
	data := req.Data
	if data == nil && s.dataProviders != nil && req.DocumentID != uuid.Nil {
		docData, err := s.dataProviders.LoadData(ctx, tenantID, docType, req.DocumentID)
		if err != nil {
			s.logger.Warn("failed to load document data from provider",
				zap.Error(err),
				zap.String("docType", string(docType)),
				zap.String("documentID", req.DocumentID.String()))
			return nil, fmt.Errorf("failed to load document data: %w", err)
		}
		data = docData
	}

	// Render template with data
	result, err := s.templateEngine.Render(ctx, &infra.RenderTemplateRequest{
		Template: template.ToPrintTemplate(),
		Data:     data,
	})
	if err != nil {
		var renderErr *infra.RenderError
		if errors.As(err, &renderErr) {
			return nil, shared.NewDomainError(renderErr.Code, renderErr.Message)
		}
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	return &PreviewResponse{
		HTML:        result.HTML,
		TemplateID:  template.ID,
		PaperSize:   string(template.PaperSize),
		Orientation: string(template.Orientation),
		Margins: MarginsDTO{
			Top:    template.Margins.Top,
			Right:  template.Margins.Right,
			Bottom: template.Margins.Bottom,
			Left:   template.Margins.Left,
		},
	}, nil
}

// GeneratePDF generates a PDF for a document and creates a print job
func (s *PrintService) GeneratePDF(ctx context.Context, tenantID, userID uuid.UUID, req GeneratePDFRequest) (*PrintJobResponse, error) {
	// Validate document type
	docType := printing.DocType(req.DocumentType)
	if !docType.IsValid() {
		return nil, shared.NewDomainError("INVALID_INPUT", "Invalid document type")
	}

	// Get template
	var template *infra.StaticTemplate
	if req.TemplateID != nil {
		template = s.templateStore.GetByID(req.TemplateID.String())
		if template == nil {
			return nil, shared.NewDomainError("NOT_FOUND", "Template not found")
		}
	} else {
		template = s.templateStore.GetDefault(docType)
		if template == nil {
			return nil, shared.NewDomainError("NOT_FOUND", "No default template found for this document type")
		}
	}

	// Create print job
	templateUUID := uuid.MustParse(template.ID)
	job, err := printing.NewPrintJob(
		tenantID,
		templateUUID,
		docType,
		req.DocumentID,
		req.DocumentNumber,
		userID,
	)
	if err != nil {
		return nil, err
	}

	if req.Copies != nil && *req.Copies > 1 {
		if err := job.SetCopies(*req.Copies); err != nil {
			return nil, err
		}
	}

	// Save job in pending state
	if err := s.jobRepo.Save(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to save print job: %w", err)
	}

	// Start rendering
	if err := job.StartRendering(); err != nil {
		return nil, err
	}
	if err := s.jobRepo.Save(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to update job status: %w", err)
	}

	// Load data from provider if not provided
	data := req.Data
	if data == nil && s.dataProviders != nil && req.DocumentID != uuid.Nil {
		docData, err := s.dataProviders.LoadData(ctx, tenantID, docType, req.DocumentID)
		if err != nil {
			s.logger.Warn("failed to load document data from provider",
				zap.Error(err),
				zap.String("docType", string(docType)),
				zap.String("documentID", req.DocumentID.String()))
			_ = job.Fail("Failed to load document data")
			_ = s.jobRepo.Save(ctx, job)
			return nil, fmt.Errorf("failed to load document data: %w", err)
		}
		data = docData
	}

	// Render HTML
	renderResult, err := s.templateEngine.Render(ctx, &infra.RenderTemplateRequest{
		Template: template.ToPrintTemplate(),
		Data:     data,
	})
	if err != nil {
		s.logger.Error("template rendering failed", zap.Error(err), zap.String("jobId", job.ID.String()))
		_ = job.Fail("Template rendering failed. Please check template syntax.")
		_ = s.jobRepo.Save(ctx, job)
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	// Render PDF
	pdfResult, err := s.pdfRenderer.Render(ctx, &infra.RenderRequest{
		HTML:        renderResult.HTML,
		PaperSize:   template.PaperSize,
		Orientation: template.Orientation,
		Margins:     template.Margins,
		Title:       fmt.Sprintf("%s - %s", docType.DisplayName(), req.DocumentNumber),
	})
	if err != nil {
		s.logger.Error("PDF rendering failed", zap.Error(err), zap.String("jobId", job.ID.String()))
		_ = job.Fail("PDF generation failed. Please try again later.")
		_ = s.jobRepo.Save(ctx, job)
		return nil, fmt.Errorf("failed to render PDF: %w", err)
	}

	// Store PDF
	storeResult, err := s.pdfStorage.Store(ctx, &infra.StoreRequest{
		TenantID: tenantID,
		JobID:    job.ID,
		PDFData:  pdfResult.PDFData,
	})
	if err != nil {
		s.logger.Error("PDF storage failed", zap.Error(err), zap.String("jobId", job.ID.String()))
		_ = job.Fail("Failed to save PDF file. Please try again later.")
		_ = s.jobRepo.Save(ctx, job)
		return nil, fmt.Errorf("failed to store PDF: %w", err)
	}

	// Complete the job
	if err := job.Complete(storeResult.URL); err != nil {
		return nil, err
	}
	if err := s.jobRepo.Save(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to update job status: %w", err)
	}

	s.logger.Info("PDF generated",
		zap.String("jobId", job.ID.String()),
		zap.String("docType", string(docType)),
		zap.String("url", storeResult.URL))

	return toJobResponse(job), nil
}

// =============================================================================
// Print Job Operations
// =============================================================================

// GetJob retrieves a print job by ID
func (s *PrintService) GetJob(ctx context.Context, tenantID, jobID uuid.UUID) (*PrintJobResponse, error) {
	job, err := s.jobRepo.FindByIDForTenant(ctx, tenantID, jobID)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, shared.NewDomainError("NOT_FOUND", "Print job not found")
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return toJobResponse(job), nil
}

// ListJobs retrieves a paginated list of print jobs
func (s *PrintService) ListJobs(ctx context.Context, tenantID uuid.UUID, req ListJobsRequest) (*ListJobsResponse, error) {
	filter := shared.Filter{
		Page:     req.Page,
		PageSize: req.PageSize,
		OrderBy:  req.OrderBy,
		OrderDir: req.OrderDir,
	}

	jobs, err := s.jobRepo.FindAllForTenant(ctx, tenantID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	total, err := s.jobRepo.CountForTenant(ctx, tenantID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count jobs: %w", err)
	}

	items := make([]PrintJobResponse, len(jobs))
	for i, j := range jobs {
		items[i] = *toJobResponse(&j)
	}

	return &ListJobsResponse{
		Items: items,
		Total: total,
		Page:  req.Page,
		Size:  req.PageSize,
	}, nil
}

// GetJobsByDocument retrieves print jobs for a specific document
func (s *PrintService) GetJobsByDocument(ctx context.Context, tenantID uuid.UUID, docType string, documentID uuid.UUID) ([]PrintJobResponse, error) {
	dt := printing.DocType(docType)
	if !dt.IsValid() {
		return nil, shared.NewDomainError("INVALID_INPUT", "Invalid document type")
	}

	jobs, err := s.jobRepo.FindByDocument(ctx, tenantID, dt, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to find jobs: %w", err)
	}

	result := make([]PrintJobResponse, len(jobs))
	for i, j := range jobs {
		result[i] = *toJobResponse(&j)
	}

	return result, nil
}

// =============================================================================
// Reference Data Operations
// =============================================================================

// GetDocumentTypes returns all available document types
func (s *PrintService) GetDocumentTypes() []DocumentTypeResponse {
	docTypes := printing.AllDocTypes()
	result := make([]DocumentTypeResponse, len(docTypes))
	for i, dt := range docTypes {
		result[i] = DocumentTypeResponse{
			Code:        string(dt),
			DisplayName: dt.DisplayName(),
		}
	}
	return result
}

// GetPaperSizes returns all available paper sizes
func (s *PrintService) GetPaperSizes() []PaperSizeResponse {
	paperSizes := printing.AllPaperSizes()
	result := make([]PaperSizeResponse, len(paperSizes))
	for i, ps := range paperSizes {
		w, h := ps.Dimensions()
		result[i] = PaperSizeResponse{
			Code:   string(ps),
			Width:  w,
			Height: h,
		}
	}
	return result
}

// =============================================================================
// Helper Functions
// =============================================================================

func staticTemplateToResponse(t *infra.StaticTemplate) TemplateResponse {
	return TemplateResponse{
		ID:           t.ID,
		TenantID:     "", // Static templates are not tenant-specific
		DocumentType: string(t.DocType),
		Name:         t.Name,
		Description:  t.Description,
		Content:      "", // Don't include content in list responses
		PaperSize:    string(t.PaperSize),
		Orientation:  string(t.Orientation),
		Margins: MarginsDTO{
			Top:    t.Margins.Top,
			Right:  t.Margins.Right,
			Bottom: t.Margins.Bottom,
			Left:   t.Margins.Left,
		},
		IsDefault: t.IsDefault,
		Status:    "ACTIVE",
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
	}
}

func toJobResponse(j *printing.PrintJob) *PrintJobResponse {
	resp := &PrintJobResponse{
		ID:             j.ID.String(),
		TenantID:       j.TenantID.String(),
		TemplateID:     j.TemplateID.String(),
		DocumentType:   string(j.DocumentType),
		DocumentID:     j.DocumentID.String(),
		DocumentNumber: j.DocumentNumber,
		Status:         string(j.Status),
		Copies:         j.Copies,
		PdfURL:         j.PdfURL,
		ErrorMessage:   j.ErrorMessage,
		CreatedAt:      j.CreatedAt,
		UpdatedAt:      j.UpdatedAt,
	}
	if j.PrintedAt != nil {
		resp.PrintedAt = j.PrintedAt
	}
	if j.PrintedBy != nil {
		resp.PrintedBy = j.PrintedBy.String()
	}
	return resp
}
