package handler

import (
	"io"
	"net/http"
	"regexp"
	"strings"

	printingapp "github.com/erp/backend/internal/application/printing"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PrintHandler handles print-related API endpoints
type PrintHandler struct {
	BaseHandler
	printService *printingapp.PrintService
}

// NewPrintHandler creates a new PrintHandler
func NewPrintHandler(printService *printingapp.PrintService) *PrintHandler {
	return &PrintHandler{
		printService: printService,
	}
}

// =============================================================================
// Template Endpoints
// =============================================================================

// CreateTemplateRequest represents a request to create a new print template
//
//	@Description	Request body for creating a new print template
type CreateTemplateRequest struct {
	DocumentType string          `json:"document_type" binding:"required" example:"SALES_ORDER"`
	Name         string          `json:"name" binding:"required,min=1,max=100" example:"Default Sales Order Template"`
	Description  string          `json:"description" binding:"max=500" example:"Standard template for sales orders"`
	Content      string          `json:"content" binding:"required" example:"<html>...template content...</html>"`
	PaperSize    string          `json:"paper_size" binding:"required" example:"A4"`
	Orientation  string          `json:"orientation" example:"PORTRAIT"`
	Margins      *MarginsRequest `json:"margins"`
}

// UpdateTemplateRequest represents a request to update a print template
//
//	@Description	Request body for updating a print template
type UpdateTemplateRequest struct {
	Name        *string         `json:"name" binding:"omitempty,min=1,max=100" example:"Updated Template Name"`
	Description *string         `json:"description" binding:"omitempty,max=500" example:"Updated description"`
	Content     *string         `json:"content"`
	PaperSize   *string         `json:"paper_size" example:"A5"`
	Orientation *string         `json:"orientation" example:"LANDSCAPE"`
	Margins     *MarginsRequest `json:"margins"`
}

// MarginsRequest represents page margins in the request
type MarginsRequest struct {
	Top    int `json:"top" example:"10"`
	Right  int `json:"right" example:"10"`
	Bottom int `json:"bottom" example:"10"`
	Left   int `json:"left" example:"10"`
}

// PreviewDocumentRequest represents a request to preview a document
//
//	@Description	Request body for previewing a document
type PreviewDocumentRequest struct {
	DocumentType string  `json:"document_type" binding:"required" example:"SALES_ORDER"`
	DocumentID   string  `json:"document_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	TemplateID   *string `json:"template_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Data         any     `json:"data"`
}

// GeneratePDFRequest represents a request to generate a PDF
//
//	@Description	Request body for generating a PDF
type GeneratePDFHTTPRequest struct {
	DocumentType   string  `json:"document_type" binding:"required" example:"SALES_ORDER"`
	DocumentID     string  `json:"document_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	DocumentNumber string  `json:"document_number" binding:"required" example:"SO-2024-001"`
	TemplateID     *string `json:"template_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Copies         *int    `json:"copies" binding:"omitempty,min=1,max=100" example:"1"`
	Data           any     `json:"data"`
}

// TemplateResponse represents a print template response
//
//	@Description	Print template response
type TemplateResponse struct {
	ID           string          `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantID     string          `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	DocumentType string          `json:"document_type" example:"SALES_ORDER"`
	Name         string          `json:"name" example:"Default Sales Order Template"`
	Description  string          `json:"description" example:"Standard template for sales orders"`
	PaperSize    string          `json:"paper_size" example:"A4"`
	Orientation  string          `json:"orientation" example:"PORTRAIT"`
	Margins      MarginsResponse `json:"margins"`
	IsDefault    bool            `json:"is_default" example:"true"`
	Status       string          `json:"status" example:"ACTIVE"`
	CreatedAt    string          `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt    string          `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

// MarginsResponse represents page margins in the response
type MarginsResponse struct {
	Top    int `json:"top" example:"10"`
	Right  int `json:"right" example:"10"`
	Bottom int `json:"bottom" example:"10"`
	Left   int `json:"left" example:"10"`
}

// PrintJobResponse represents a print job response
//
//	@Description	Print job response
type PrintJobResponse struct {
	ID             string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantID       string  `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	TemplateID     string  `json:"template_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	DocumentType   string  `json:"document_type" example:"SALES_ORDER"`
	DocumentID     string  `json:"document_id" example:"550e8400-e29b-41d4-a716-446655440003"`
	DocumentNumber string  `json:"document_number" example:"SO-2024-001"`
	Status         string  `json:"status" example:"COMPLETED"`
	Copies         int     `json:"copies" example:"1"`
	PdfURL         string  `json:"pdf_url,omitempty" example:"/api/v1/print/jobs/xxx/download"`
	ErrorMessage   string  `json:"error_message,omitempty"`
	PrintedAt      *string `json:"printed_at,omitempty" example:"2024-01-15T10:30:00Z"`
	PrintedBy      string  `json:"printed_by,omitempty" example:"550e8400-e29b-41d4-a716-446655440004"`
	CreatedAt      string  `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt      string  `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

// PreviewResponse represents the preview result
//
//	@Description	Document preview response
type PreviewHTTPResponse struct {
	HTML        string          `json:"html"`
	TemplateID  string          `json:"template_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	PaperSize   string          `json:"paper_size" example:"A4"`
	Orientation string          `json:"orientation" example:"PORTRAIT"`
	Margins     MarginsResponse `json:"margins"`
}

// DocumentTypeResponse represents a document type
//
//	@Description	Document type information
type DocumentTypeResponse struct {
	Code        string `json:"code" example:"SALES_ORDER"`
	DisplayName string `json:"display_name" example:"销售订单"`
}

// PaperSizeResponse represents a paper size
//
//	@Description	Paper size information
type PaperSizeResponse struct {
	Code   string `json:"code" example:"A4"`
	Width  int    `json:"width" example:"210"`
	Height int    `json:"height" example:"297"`
}

// CreateTemplate godoc
//
//	@Summary		Create a new print template
//	@Description	Create a new print template for a specific document type
//	@Tags			print-templates
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateTemplateRequest	true	"Template creation request"
//	@Success		201		{object}	dto.Response{data=TemplateResponse}
//	@Failure		400		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		409		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500		{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/print/templates [post]
func (h *PrintHandler) CreateTemplate(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	appReq := printingapp.CreateTemplateRequest{
		DocumentType: req.DocumentType,
		Name:         req.Name,
		Description:  req.Description,
		Content:      req.Content,
		PaperSize:    req.PaperSize,
		Orientation:  req.Orientation,
	}

	if req.Margins != nil {
		appReq.Margins = &printingapp.MarginsDTO{
			Top:    req.Margins.Top,
			Right:  req.Margins.Right,
			Bottom: req.Margins.Bottom,
			Left:   req.Margins.Left,
		}
	}

	result, err := h.printService.CreateTemplate(c.Request.Context(), tenantID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Created(c, result)
}

// GetTemplate godoc
//
//	@Summary		Get print template by ID
//	@Description	Retrieve a print template by its ID
//	@Tags			print-templates
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Template ID"	format(uuid)
//	@Success		200	{object}	dto.Response{data=TemplateResponse}
//	@Failure		400	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		404	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500	{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/print/templates/{id} [get]
func (h *PrintHandler) GetTemplate(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid template ID format")
		return
	}

	result, err := h.printService.GetTemplate(c.Request.Context(), tenantID, templateID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// ListTemplates godoc
//
//	@Summary		List print templates
//	@Description	Retrieve a paginated list of print templates
//	@Tags			print-templates
//	@Accept			json
//	@Produce		json
//	@Param			page		query		int		false	"Page number"		default(1)
//	@Param			page_size	query		int		false	"Page size"			default(20)
//	@Param			order_by	query		string	false	"Order by field"	default(created_at)
//	@Param			order_dir	query		string	false	"Order direction"	Enums(asc, desc)	default(desc)
//	@Param			search		query		string	false	"Search term"
//	@Param			doc_type	query		string	false	"Filter by document type"
//	@Param			status		query		string	false	"Filter by status"
//	@Success		200			{object}	dto.Response{data=[]TemplateResponse,meta=dto.Meta}
//	@Failure		400			{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401			{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500			{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/print/templates [get]
func (h *PrintHandler) ListTemplates(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	req := printingapp.ListTemplatesRequest{
		Page:     1,
		PageSize: 20,
		OrderBy:  "created_at",
		OrderDir: "desc",
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Apply defaults if not provided
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}

	result, err := h.printService.ListTemplates(c.Request.Context(), tenantID, req)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.SuccessWithMeta(c, result.Items, result.Total, result.Page, result.Size)
}

// UpdateTemplate godoc
//
//	@Summary		Update print template
//	@Description	Update an existing print template
//	@Tags			print-templates
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Template ID"	format(uuid)
//	@Param			request	body		UpdateTemplateRequest	true	"Template update request"
//	@Success		200		{object}	dto.Response{data=TemplateResponse}
//	@Failure		400		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		404		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		409		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500		{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/print/templates/{id} [put]
func (h *PrintHandler) UpdateTemplate(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid template ID format")
		return
	}

	var req UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	appReq := printingapp.UpdateTemplateRequest{
		Name:        req.Name,
		Description: req.Description,
		Content:     req.Content,
		PaperSize:   req.PaperSize,
		Orientation: req.Orientation,
	}

	if req.Margins != nil {
		appReq.Margins = &printingapp.MarginsDTO{
			Top:    req.Margins.Top,
			Right:  req.Margins.Right,
			Bottom: req.Margins.Bottom,
			Left:   req.Margins.Left,
		}
	}

	result, err := h.printService.UpdateTemplate(c.Request.Context(), tenantID, templateID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// DeleteTemplate godoc
//
//	@Summary		Delete print template
//	@Description	Delete an existing print template
//	@Tags			print-templates
//	@Accept			json
//	@Produce		json
//	@Param			id	path	string	true	"Template ID"	format(uuid)
//	@Success		204	"No Content"
//	@Failure		400	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		404	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		422	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500	{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/print/templates/{id} [delete]
func (h *PrintHandler) DeleteTemplate(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid template ID format")
		return
	}

	if err := h.printService.DeleteTemplate(c.Request.Context(), tenantID, templateID); err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.NoContent(c)
}

// SetDefaultTemplate godoc
//
//	@Summary		Set template as default
//	@Description	Set a template as the default for its document type
//	@Tags			print-templates
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Template ID"	format(uuid)
//	@Success		200	{object}	dto.Response{data=TemplateResponse}
//	@Failure		400	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		404	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		422	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500	{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/print/templates/{id}/set-default [post]
func (h *PrintHandler) SetDefaultTemplate(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid template ID format")
		return
	}

	result, err := h.printService.SetDefaultTemplate(c.Request.Context(), tenantID, templateID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// ActivateTemplate godoc
//
//	@Summary		Activate template
//	@Description	Activate an inactive print template
//	@Tags			print-templates
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Template ID"	format(uuid)
//	@Success		200	{object}	dto.Response{data=TemplateResponse}
//	@Failure		400	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		404	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		422	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500	{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/print/templates/{id}/activate [post]
func (h *PrintHandler) ActivateTemplate(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid template ID format")
		return
	}

	result, err := h.printService.ActivateTemplate(c.Request.Context(), tenantID, templateID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// DeactivateTemplate godoc
//
//	@Summary		Deactivate template
//	@Description	Deactivate an active print template
//	@Tags			print-templates
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Template ID"	format(uuid)
//	@Success		200	{object}	dto.Response{data=TemplateResponse}
//	@Failure		400	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		404	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		422	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500	{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/print/templates/{id}/deactivate [post]
func (h *PrintHandler) DeactivateTemplate(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid template ID format")
		return
	}

	result, err := h.printService.DeactivateTemplate(c.Request.Context(), tenantID, templateID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// GetTemplatesByDocType godoc
//
//	@Summary		Get templates by document type
//	@Description	Retrieve all active templates for a specific document type
//	@Tags			print-templates
//	@Accept			json
//	@Produce		json
//	@Param			doc_type	path		string	true	"Document type"
//	@Success		200			{object}	dto.Response{data=[]TemplateResponse}
//	@Failure		400			{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401			{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500			{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/print/templates/by-doc-type/{doc_type} [get]
func (h *PrintHandler) GetTemplatesByDocType(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	docType := c.Param("doc_type")
	if docType == "" {
		h.BadRequest(c, "Document type is required")
		return
	}

	result, err := h.printService.GetTemplatesByDocType(c.Request.Context(), tenantID, docType)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// =============================================================================
// Print Preview and PDF Generation Endpoints
// =============================================================================

// PreviewDocument godoc
//
//	@Summary		Preview document as HTML
//	@Description	Generate HTML preview for a document using a print template
//	@Tags			print-preview
//	@Accept			json
//	@Produce		json
//	@Param			request	body		PreviewDocumentRequest	true	"Preview request"
//	@Success		200		{object}	dto.Response{data=PreviewHTTPResponse}
//	@Failure		400		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		404		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		422		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500		{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/print/preview [post]
func (h *PrintHandler) PreviewDocument(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req PreviewDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	documentID, err := uuid.Parse(req.DocumentID)
	if err != nil {
		h.BadRequest(c, "Invalid document ID format")
		return
	}

	appReq := printingapp.PreviewRequest{
		DocumentType: req.DocumentType,
		DocumentID:   documentID,
		Data:         req.Data,
	}

	if req.TemplateID != nil {
		templateID, err := uuid.Parse(*req.TemplateID)
		if err != nil {
			h.BadRequest(c, "Invalid template ID format")
			return
		}
		appReq.TemplateID = &templateID
	}

	result, err := h.printService.PreviewDocument(c.Request.Context(), tenantID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// GeneratePDF godoc
//
//	@Summary		Generate PDF
//	@Description	Generate a PDF for a document and create a print job
//	@Tags			print-jobs
//	@Accept			json
//	@Produce		json
//	@Param			request	body		GeneratePDFHTTPRequest	true	"PDF generation request"
//	@Success		201		{object}	dto.Response{data=PrintJobResponse}
//	@Failure		400		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		404		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		422		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500		{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/print/generate [post]
func (h *PrintHandler) GeneratePDF(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	userID, err := getUserID(c)
	if err != nil {
		h.Unauthorized(c, "User ID not found")
		return
	}

	var req GeneratePDFHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	documentID, err := uuid.Parse(req.DocumentID)
	if err != nil {
		h.BadRequest(c, "Invalid document ID format")
		return
	}

	appReq := printingapp.GeneratePDFRequest{
		DocumentType:   req.DocumentType,
		DocumentID:     documentID,
		DocumentNumber: req.DocumentNumber,
		Copies:         req.Copies,
		Data:           req.Data,
	}

	if req.TemplateID != nil {
		templateID, err := uuid.Parse(*req.TemplateID)
		if err != nil {
			h.BadRequest(c, "Invalid template ID format")
			return
		}
		appReq.TemplateID = &templateID
	}

	result, err := h.printService.GeneratePDF(c.Request.Context(), tenantID, userID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Created(c, result)
}

// =============================================================================
// Print Job Endpoints
// =============================================================================

// GetJob godoc
//
//	@Summary		Get print job by ID
//	@Description	Retrieve a print job by its ID
//	@Tags			print-jobs
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Job ID"	format(uuid)
//	@Success		200	{object}	dto.Response{data=PrintJobResponse}
//	@Failure		400	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		404	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500	{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/print/jobs/{id} [get]
func (h *PrintHandler) GetJob(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid job ID format")
		return
	}

	result, err := h.printService.GetJob(c.Request.Context(), tenantID, jobID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// ListJobs godoc
//
//	@Summary		List print jobs
//	@Description	Retrieve a paginated list of print jobs
//	@Tags			print-jobs
//	@Accept			json
//	@Produce		json
//	@Param			page		query		int		false	"Page number"		default(1)
//	@Param			page_size	query		int		false	"Page size"			default(20)
//	@Param			order_by	query		string	false	"Order by field"	default(created_at)
//	@Param			order_dir	query		string	false	"Order direction"	Enums(asc, desc)	default(desc)
//	@Param			doc_type	query		string	false	"Filter by document type"
//	@Param			status		query		string	false	"Filter by status"
//	@Success		200			{object}	dto.Response{data=[]PrintJobResponse,meta=dto.Meta}
//	@Failure		400			{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401			{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500			{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/print/jobs [get]
func (h *PrintHandler) ListJobs(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	req := printingapp.ListJobsRequest{
		Page:     1,
		PageSize: 20,
		OrderBy:  "created_at",
		OrderDir: "desc",
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}

	result, err := h.printService.ListJobs(c.Request.Context(), tenantID, req)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.SuccessWithMeta(c, result.Items, result.Total, result.Page, result.Size)
}

// GetJobsByDocument godoc
//
//	@Summary		Get jobs by document
//	@Description	Retrieve print jobs for a specific document
//	@Tags			print-jobs
//	@Accept			json
//	@Produce		json
//	@Param			doc_type	path		string	true	"Document type"
//	@Param			document_id	path		string	true	"Document ID"	format(uuid)
//	@Success		200			{object}	dto.Response{data=[]PrintJobResponse}
//	@Failure		400			{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401			{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500			{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/print/jobs/by-document/{doc_type}/{document_id} [get]
func (h *PrintHandler) GetJobsByDocument(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	docType := c.Param("doc_type")
	if docType == "" {
		h.BadRequest(c, "Document type is required")
		return
	}

	documentID, err := uuid.Parse(c.Param("document_id"))
	if err != nil {
		h.BadRequest(c, "Invalid document ID format")
		return
	}

	result, err := h.printService.GetJobsByDocument(c.Request.Context(), tenantID, docType, documentID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// DownloadPDF godoc
//
//	@Summary		Download PDF
//	@Description	Download the PDF file for a completed print job
//	@Tags			print-jobs
//	@Produce		application/pdf
//	@Param			id	path		string	true	"Job ID"	format(uuid)
//	@Success		200	{file}		binary	"PDF file"
//	@Failure		400	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		404	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500	{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/print/jobs/{id}/download [get]
func (h *PrintHandler) DownloadPDF(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid job ID format")
		return
	}

	// Get the job to verify access and get PDF URL
	job, err := h.printService.GetJob(c.Request.Context(), tenantID, jobID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	if job.Status != "COMPLETED" {
		h.Error(c, http.StatusBadRequest, dto.ErrCodeInvalidState, "PDF not available. Job status: "+job.Status)
		return
	}

	if job.PdfURL == "" {
		h.Error(c, http.StatusNotFound, dto.ErrCodeNotFound, "PDF file not found")
		return
	}

	// Security: Validate URL is a relative path to prevent open redirect attacks
	// The PDF URL should be relative (e.g., /api/v1/prints/...)
	if !strings.HasPrefix(job.PdfURL, "/") {
		h.InternalError(c, "Invalid PDF URL configuration")
		return
	}

	// The PDF URL is a relative path, redirect to serve it
	// In production, this would use the storage service to stream the file
	c.Redirect(http.StatusTemporaryRedirect, job.PdfURL)
}

// =============================================================================
// Reference Data Endpoints
// =============================================================================

// GetDocumentTypes godoc
//
//	@Summary		Get available document types
//	@Description	Retrieve all available document types that can be printed
//	@Tags			print-reference
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	dto.Response{data=[]DocumentTypeResponse}
//	@Security		BearerAuth
//	@Router			/print/document-types [get]
func (h *PrintHandler) GetDocumentTypes(c *gin.Context) {
	result := h.printService.GetDocumentTypes()
	h.Success(c, result)
}

// GetPaperSizes godoc
//
//	@Summary		Get available paper sizes
//	@Description	Retrieve all available paper sizes for printing
//	@Tags			print-reference
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	dto.Response{data=[]PaperSizeResponse}
//	@Security		BearerAuth
//	@Router			/print/paper-sizes [get]
func (h *PrintHandler) GetPaperSizes(c *gin.Context) {
	result := h.printService.GetPaperSizes()
	h.Success(c, result)
}

// GetTemplateContent godoc
//
//	@Summary		Get template content
//	@Description	Retrieve the HTML content of a print template for editing
//	@Tags			print-templates
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Template ID"	format(uuid)
//	@Success		200	{object}	dto.Response{data=object{content=string}}
//	@Failure		400	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		404	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500	{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/print/templates/{id}/content [get]
func (h *PrintHandler) GetTemplateContent(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid template ID format")
		return
	}

	result, err := h.printService.GetTemplate(c.Request.Context(), tenantID, templateID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	// Return just the content for the template editor
	h.Success(c, map[string]string{
		"content": result.Content,
	})
}

// UpdateTemplateContent godoc
//
//	@Summary		Update template content
//	@Description	Update only the HTML content of a print template
//	@Tags			print-templates
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string								true	"Template ID"	format(uuid)
//	@Param			request	body		object{content=string}				true	"Template content"
//	@Success		200		{object}	dto.Response{data=TemplateResponse}
//	@Failure		400		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		404		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500		{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/print/templates/{id}/content [put]
func (h *PrintHandler) UpdateTemplateContent(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid template ID format")
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	appReq := printingapp.UpdateTemplateRequest{
		Content: &req.Content,
	}

	result, err := h.printService.UpdateTemplate(c.Request.Context(), tenantID, templateID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// ServePDF godoc
//
//	@Summary		Serve PDF file
//	@Description	Serve a PDF file from storage
//	@Tags			print-files
//	@Produce		application/pdf
//	@Param			tenant_id	path		string	true	"Tenant ID"
//	@Param			year		path		string	true	"Year"
//	@Param			month		path		string	true	"Month"
//	@Param			filename	path		string	true	"Filename"
//	@Success		200			{file}		binary	"PDF file"
//	@Failure		400			{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		404			{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500			{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/prints/{tenant_id}/{year}/{month}/{filename} [get]
func (h *PrintHandler) ServePDF(c *gin.Context, storage interface {
	Get(ctx interface{}, path string) (io.ReadCloser, error)
}) {
	// Validate tenant access
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	pathTenantID := c.Param("tenant_id")
	if pathTenantID != tenantID.String() {
		h.Forbidden(c, "Access denied to this file")
		return
	}

	// Get and validate path components to prevent path traversal attacks
	year := c.Param("year")
	month := c.Param("month")
	filename := c.Param("filename")

	// Validate year format (4 digits)
	yearPattern := regexp.MustCompile(`^\d{4}$`)
	if !yearPattern.MatchString(year) {
		h.BadRequest(c, "Invalid year format")
		return
	}

	// Validate month format (01-12)
	monthPattern := regexp.MustCompile(`^(0[1-9]|1[0-2])$`)
	if !monthPattern.MatchString(month) {
		h.BadRequest(c, "Invalid month format")
		return
	}

	// Validate filename - must be UUID.pdf format and not contain path traversal
	filenamePattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.pdf$`)
	if !filenamePattern.MatchString(filename) {
		h.BadRequest(c, "Invalid filename format")
		return
	}

	// Additional security: check for path traversal patterns
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		h.BadRequest(c, "Invalid filename")
		return
	}

	// Build safe path
	path := pathTenantID + "/" + year + "/" + month + "/" + filename

	// Get file from storage
	file, err := storage.Get(c.Request.Context(), path)
	if err != nil {
		h.NotFound(c, "PDF file not found")
		return
	}
	defer file.Close()

	// Set headers (use safe filename without path components)
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "inline; filename=\""+filename+"\"")

	// Stream file
	if _, err := io.Copy(c.Writer, file); err != nil {
		h.InternalError(c, "Failed to serve PDF file")
		return
	}
}
