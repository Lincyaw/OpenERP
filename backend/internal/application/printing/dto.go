package printing

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Template DTOs
// =============================================================================

// CreateTemplateRequest represents a request to create a new print template
type CreateTemplateRequest struct {
	DocumentType string      `json:"document_type" binding:"required"`
	Name         string      `json:"name" binding:"required,min=1,max=100"`
	Description  string      `json:"description" binding:"max=500"`
	Content      string      `json:"content" binding:"required"`
	PaperSize    string      `json:"paper_size" binding:"required"`
	Orientation  string      `json:"orientation"`
	Margins      *MarginsDTO `json:"margins"`
}

// UpdateTemplateRequest represents a request to update a print template
type UpdateTemplateRequest struct {
	Name        *string     `json:"name" binding:"omitempty,min=1,max=100"`
	Description *string     `json:"description" binding:"omitempty,max=500"`
	Content     *string     `json:"content"`
	PaperSize   *string     `json:"paper_size"`
	Orientation *string     `json:"orientation"`
	Margins     *MarginsDTO `json:"margins"`
}

// ListTemplatesRequest represents a request to list templates
type ListTemplatesRequest struct {
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=100"`
	OrderBy  string `form:"order_by"`
	OrderDir string `form:"order_dir" binding:"omitempty,oneof=asc desc"`
	Search   string `form:"search"`
	DocType  string `form:"doc_type"`
	Status   string `form:"status"`
}

// TemplateResponse represents a print template response
type TemplateResponse struct {
	ID           string     `json:"id"`
	TenantID     string     `json:"tenant_id"`
	DocumentType string     `json:"document_type"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	Content      string     `json:"content,omitempty"` // Template HTML content
	PaperSize    string     `json:"paper_size"`
	Orientation  string     `json:"orientation"`
	Margins      MarginsDTO `json:"margins"`
	IsDefault    bool       `json:"is_default"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// ListTemplatesResponse represents a paginated list of templates
type ListTemplatesResponse struct {
	Items []TemplateResponse `json:"items"`
	Total int64              `json:"total"`
	Page  int                `json:"page"`
	Size  int                `json:"size"`
}

// MarginsDTO represents page margins
type MarginsDTO struct {
	Top    int `json:"top"`
	Right  int `json:"right"`
	Bottom int `json:"bottom"`
	Left   int `json:"left"`
}

// =============================================================================
// Print Preview and PDF Generation DTOs
// =============================================================================

// PreviewRequest represents a request to preview a document
type PreviewRequest struct {
	DocumentType string     `json:"document_type" binding:"required"`
	DocumentID   uuid.UUID  `json:"document_id" binding:"required"`
	TemplateID   *uuid.UUID `json:"template_id"`
	Data         any        `json:"data"` // Document data for template rendering
}

// PreviewResponse represents the preview result
type PreviewResponse struct {
	HTML        string     `json:"html"`
	TemplateID  string     `json:"template_id"`
	PaperSize   string     `json:"paper_size"`
	Orientation string     `json:"orientation"`
	Margins     MarginsDTO `json:"margins"`
}

// GeneratePDFRequest represents a request to generate a PDF
type GeneratePDFRequest struct {
	DocumentType   string     `json:"document_type" binding:"required"`
	DocumentID     uuid.UUID  `json:"document_id" binding:"required"`
	DocumentNumber string     `json:"document_number" binding:"required"`
	TemplateID     *uuid.UUID `json:"template_id"`
	Copies         *int       `json:"copies" binding:"omitempty,min=1,max=100"`
	Data           any        `json:"data"` // Document data for template rendering
}

// =============================================================================
// Print Job DTOs
// =============================================================================

// ListJobsRequest represents a request to list print jobs
type ListJobsRequest struct {
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=100"`
	OrderBy  string `form:"order_by"`
	OrderDir string `form:"order_dir" binding:"omitempty,oneof=asc desc"`
	DocType  string `form:"doc_type"`
	Status   string `form:"status"`
}

// PrintJobResponse represents a print job response
type PrintJobResponse struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	TemplateID     string     `json:"template_id"`
	DocumentType   string     `json:"document_type"`
	DocumentID     string     `json:"document_id"`
	DocumentNumber string     `json:"document_number"`
	Status         string     `json:"status"`
	Copies         int        `json:"copies"`
	PdfURL         string     `json:"pdf_url,omitempty"`
	ErrorMessage   string     `json:"error_message,omitempty"`
	PrintedAt      *time.Time `json:"printed_at,omitempty"`
	PrintedBy      string     `json:"printed_by,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ListJobsResponse represents a paginated list of print jobs
type ListJobsResponse struct {
	Items []PrintJobResponse `json:"items"`
	Total int64              `json:"total"`
	Page  int                `json:"page"`
	Size  int                `json:"size"`
}

// =============================================================================
// Reference Data DTOs
// =============================================================================

// DocumentTypeResponse represents a document type
type DocumentTypeResponse struct {
	Code        string `json:"code"`
	DisplayName string `json:"display_name"`
}

// PaperSizeResponse represents a paper size
type PaperSizeResponse struct {
	Code   string `json:"code"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}
