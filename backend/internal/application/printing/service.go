package printing

import (
	"context"
	"errors"
	"fmt"

	"github.com/erp/backend/internal/domain/printing"
	"github.com/erp/backend/internal/domain/shared"
	infra "github.com/erp/backend/internal/infrastructure/printing"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PrintService handles printing-related business operations
type PrintService struct {
	templateRepo   printing.PrintTemplateRepository
	jobRepo        printing.PrintJobRepository
	templateEngine *infra.TemplateEngine
	pdfRenderer    infra.PDFRenderer
	pdfStorage     infra.PDFStorage
	logger         *zap.Logger
}

// NewPrintService creates a new PrintService
func NewPrintService(
	templateRepo printing.PrintTemplateRepository,
	jobRepo printing.PrintJobRepository,
	templateEngine *infra.TemplateEngine,
	pdfRenderer infra.PDFRenderer,
	pdfStorage infra.PDFStorage,
	logger *zap.Logger,
) *PrintService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &PrintService{
		templateRepo:   templateRepo,
		jobRepo:        jobRepo,
		templateEngine: templateEngine,
		pdfRenderer:    pdfRenderer,
		pdfStorage:     pdfStorage,
		logger:         logger,
	}
}

// =============================================================================
// Print Template Operations
// =============================================================================

// CreateTemplate creates a new print template
func (s *PrintService) CreateTemplate(ctx context.Context, tenantID uuid.UUID, req CreateTemplateRequest) (*TemplateResponse, error) {
	// Validate document type first
	docType := printing.DocType(req.DocumentType)
	if !docType.IsValid() {
		return nil, shared.NewDomainError("INVALID_INPUT", "Invalid document type")
	}

	// Check if template with same name and doc type already exists
	exists, err := s.templateRepo.ExistsByDocTypeAndName(ctx, tenantID, docType, req.Name, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check template existence: %w", err)
	}
	if exists {
		return nil, shared.NewDomainError("ALREADY_EXISTS", "Template with this name already exists for this document type")
	}

	// Parse and validate paper size
	paperSize := printing.PaperSize(req.PaperSize)
	if !paperSize.IsValid() {
		return nil, shared.NewDomainError("INVALID_INPUT", "Invalid paper size")
	}

	// Create the template
	template, err := printing.NewPrintTemplate(
		tenantID,
		docType,
		req.Name,
		req.Content,
		paperSize,
	)
	if err != nil {
		return nil, err
	}

	// Set optional fields
	if req.Description != "" {
		if err := template.Update(req.Name, req.Description); err != nil {
			return nil, err
		}
	}

	if req.Orientation != "" {
		orientation := printing.Orientation(req.Orientation)
		if err := template.SetOrientation(orientation); err != nil {
			return nil, err
		}
	}

	if req.Margins != nil {
		margins := printing.Margins{
			Top:    req.Margins.Top,
			Right:  req.Margins.Right,
			Bottom: req.Margins.Bottom,
			Left:   req.Margins.Left,
		}
		if err := template.SetMargins(margins); err != nil {
			return nil, err
		}
	}

	// Save the template
	if err := s.templateRepo.Save(ctx, template); err != nil {
		return nil, fmt.Errorf("failed to save template: %w", err)
	}

	s.logger.Info("print template created",
		zap.String("id", template.ID.String()),
		zap.String("name", template.Name),
		zap.String("docType", string(template.DocumentType)))

	return toTemplateResponse(template), nil
}

// GetTemplate retrieves a template by ID
func (s *PrintService) GetTemplate(ctx context.Context, tenantID, templateID uuid.UUID) (*TemplateResponse, error) {
	template, err := s.templateRepo.FindByIDForTenant(ctx, tenantID, templateID)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, shared.NewDomainError("NOT_FOUND", "Template not found")
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	return toTemplateResponse(template), nil
}

// ListTemplates retrieves a paginated list of templates
func (s *PrintService) ListTemplates(ctx context.Context, tenantID uuid.UUID, req ListTemplatesRequest) (*ListTemplatesResponse, error) {
	// Build filter
	filter := shared.Filter{
		Page:     req.Page,
		PageSize: req.PageSize,
		OrderBy:  req.OrderBy,
		OrderDir: req.OrderDir,
		Search:   req.Search,
	}

	templates, err := s.templateRepo.FindAllForTenant(ctx, tenantID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}

	total, err := s.templateRepo.CountForTenant(ctx, tenantID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count templates: %w", err)
	}

	items := make([]TemplateResponse, len(templates))
	for i, t := range templates {
		items[i] = *toTemplateResponse(&t)
	}

	return &ListTemplatesResponse{
		Items: items,
		Total: total,
		Page:  req.Page,
		Size:  req.PageSize,
	}, nil
}

// UpdateTemplate updates an existing template
func (s *PrintService) UpdateTemplate(ctx context.Context, tenantID, templateID uuid.UUID, req UpdateTemplateRequest) (*TemplateResponse, error) {
	template, err := s.templateRepo.FindByIDForTenant(ctx, tenantID, templateID)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, shared.NewDomainError("NOT_FOUND", "Template not found")
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Check for name conflicts if name is being changed
	if req.Name != nil && *req.Name != template.Name {
		exists, err := s.templateRepo.ExistsByDocTypeAndName(ctx, tenantID, template.DocumentType, *req.Name, &templateID)
		if err != nil {
			return nil, fmt.Errorf("failed to check template existence: %w", err)
		}
		if exists {
			return nil, shared.NewDomainError("ALREADY_EXISTS", "Template with this name already exists for this document type")
		}
	}

	// Update fields
	if req.Name != nil || req.Description != nil {
		name := template.Name
		if req.Name != nil {
			name = *req.Name
		}
		description := template.Description
		if req.Description != nil {
			description = *req.Description
		}
		if err := template.Update(name, description); err != nil {
			return nil, err
		}
	}

	if req.Content != nil {
		if err := template.UpdateContent(*req.Content); err != nil {
			return nil, err
		}
	}

	if req.PaperSize != nil {
		paperSize := printing.PaperSize(*req.PaperSize)
		if err := template.SetPaperSize(paperSize); err != nil {
			return nil, err
		}
	}

	if req.Orientation != nil {
		orientation := printing.Orientation(*req.Orientation)
		if err := template.SetOrientation(orientation); err != nil {
			return nil, err
		}
	}

	if req.Margins != nil {
		margins := printing.Margins{
			Top:    req.Margins.Top,
			Right:  req.Margins.Right,
			Bottom: req.Margins.Bottom,
			Left:   req.Margins.Left,
		}
		if err := template.SetMargins(margins); err != nil {
			return nil, err
		}
	}

	// Save updates
	if err := s.templateRepo.Save(ctx, template); err != nil {
		return nil, fmt.Errorf("failed to save template: %w", err)
	}

	s.logger.Info("print template updated",
		zap.String("id", template.ID.String()),
		zap.String("name", template.Name))

	return toTemplateResponse(template), nil
}

// DeleteTemplate deletes a template
func (s *PrintService) DeleteTemplate(ctx context.Context, tenantID, templateID uuid.UUID) error {
	template, err := s.templateRepo.FindByIDForTenant(ctx, tenantID, templateID)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return shared.NewDomainError("NOT_FOUND", "Template not found")
		}
		return fmt.Errorf("failed to get template: %w", err)
	}

	// Cannot delete default template
	if template.IsDefault {
		return shared.NewDomainError("INVALID_STATE", "Cannot delete default template. Set another template as default first.")
	}

	if err := s.templateRepo.Delete(ctx, templateID); err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	s.logger.Info("print template deleted",
		zap.String("id", templateID.String()))

	return nil
}

// SetDefaultTemplate sets a template as the default for its document type
func (s *PrintService) SetDefaultTemplate(ctx context.Context, tenantID, templateID uuid.UUID) (*TemplateResponse, error) {
	template, err := s.templateRepo.FindByIDForTenant(ctx, tenantID, templateID)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, shared.NewDomainError("NOT_FOUND", "Template not found")
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Clear existing default for this doc type
	if err := s.templateRepo.ClearDefaultForDocType(ctx, tenantID, template.DocumentType); err != nil {
		return nil, fmt.Errorf("failed to clear existing default: %w", err)
	}

	// Set this template as default
	if err := template.SetAsDefault(); err != nil {
		return nil, err
	}

	if err := s.templateRepo.Save(ctx, template); err != nil {
		return nil, fmt.Errorf("failed to save template: %w", err)
	}

	s.logger.Info("print template set as default",
		zap.String("id", template.ID.String()),
		zap.String("docType", string(template.DocumentType)))

	return toTemplateResponse(template), nil
}

// ActivateTemplate activates a template
func (s *PrintService) ActivateTemplate(ctx context.Context, tenantID, templateID uuid.UUID) (*TemplateResponse, error) {
	template, err := s.templateRepo.FindByIDForTenant(ctx, tenantID, templateID)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, shared.NewDomainError("NOT_FOUND", "Template not found")
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	if err := template.Activate(); err != nil {
		return nil, err
	}

	if err := s.templateRepo.Save(ctx, template); err != nil {
		return nil, fmt.Errorf("failed to save template: %w", err)
	}

	return toTemplateResponse(template), nil
}

// DeactivateTemplate deactivates a template
func (s *PrintService) DeactivateTemplate(ctx context.Context, tenantID, templateID uuid.UUID) (*TemplateResponse, error) {
	template, err := s.templateRepo.FindByIDForTenant(ctx, tenantID, templateID)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, shared.NewDomainError("NOT_FOUND", "Template not found")
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	if err := template.Deactivate(); err != nil {
		return nil, err
	}

	if err := s.templateRepo.Save(ctx, template); err != nil {
		return nil, fmt.Errorf("failed to save template: %w", err)
	}

	return toTemplateResponse(template), nil
}

// GetTemplatesByDocType retrieves all active templates for a document type
func (s *PrintService) GetTemplatesByDocType(ctx context.Context, tenantID uuid.UUID, docType string) ([]TemplateResponse, error) {
	dt := printing.DocType(docType)
	if !dt.IsValid() {
		return nil, shared.NewDomainError("INVALID_INPUT", "Invalid document type")
	}

	templates, err := s.templateRepo.FindActiveByDocType(ctx, tenantID, dt)
	if err != nil {
		return nil, fmt.Errorf("failed to find templates: %w", err)
	}

	result := make([]TemplateResponse, len(templates))
	for i, t := range templates {
		result[i] = *toTemplateResponse(&t)
	}

	return result, nil
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
	var template *printing.PrintTemplate
	var err error

	if req.TemplateID != nil {
		template, err = s.templateRepo.FindByIDForTenant(ctx, tenantID, *req.TemplateID)
	} else {
		// Use default template for the document type
		template, err = s.templateRepo.FindDefault(ctx, tenantID, docType)
		if err == nil && template == nil {
			return nil, shared.NewDomainError("NOT_FOUND", "No default template found for this document type")
		}
	}

	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, shared.NewDomainError("NOT_FOUND", "Template not found")
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	if !template.CanBeUsed() {
		return nil, shared.NewDomainError("INVALID_STATE", "Template is not available for use")
	}

	// Render template with provided data
	result, err := s.templateEngine.Render(ctx, &infra.RenderTemplateRequest{
		Template: template,
		Data:     req.Data,
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
		TemplateID:  template.ID.String(),
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
	var template *printing.PrintTemplate
	var err error

	if req.TemplateID != nil {
		template, err = s.templateRepo.FindByIDForTenant(ctx, tenantID, *req.TemplateID)
	} else {
		template, err = s.templateRepo.FindDefault(ctx, tenantID, docType)
		if err == nil && template == nil {
			return nil, shared.NewDomainError("NOT_FOUND", "No default template found for this document type")
		}
	}

	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, shared.NewDomainError("NOT_FOUND", "Template not found")
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	if !template.CanBeUsed() {
		return nil, shared.NewDomainError("INVALID_STATE", "Template is not available for use")
	}

	// Create print job
	job, err := printing.NewPrintJob(
		tenantID,
		template.ID,
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

	// Render HTML
	renderResult, err := s.templateEngine.Render(ctx, &infra.RenderTemplateRequest{
		Template: template,
		Data:     req.Data,
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

func toTemplateResponse(t *printing.PrintTemplate) *TemplateResponse {
	return &TemplateResponse{
		ID:           t.ID.String(),
		TenantID:     t.TenantID.String(),
		DocumentType: string(t.DocumentType),
		Name:         t.Name,
		Description:  t.Description,
		Content:      t.Content,
		PaperSize:    string(t.PaperSize),
		Orientation:  string(t.Orientation),
		Margins: MarginsDTO{
			Top:    t.Margins.Top,
			Right:  t.Margins.Right,
			Bottom: t.Margins.Bottom,
			Left:   t.Margins.Left,
		},
		IsDefault: t.IsDefault,
		Status:    string(t.Status),
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
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
