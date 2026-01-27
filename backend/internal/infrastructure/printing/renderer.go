package printing

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/printing"
)

// RenderRequest contains the parameters for rendering HTML to PDF
type RenderRequest struct {
	// HTML content to render
	HTML string
	// PaperSize defines the output paper dimensions
	PaperSize printing.PaperSize
	// Orientation defines portrait or landscape
	Orientation printing.Orientation
	// Margins in millimeters
	Margins printing.Margins
	// Title for the PDF document metadata
	Title string
	// Header HTML content (optional)
	HeaderHTML string
	// Footer HTML content (optional)
	FooterHTML string
	// EnableLocalFileAccess allows loading local images (use with caution)
	EnableLocalFileAccess bool
	// Timeout overrides the default rendering timeout
	Timeout time.Duration
}

// RenderResult contains the output from PDF rendering
type RenderResult struct {
	// PDFData is the raw PDF file content
	PDFData []byte
	// PageCount is the number of pages in the PDF
	PageCount int
	// RenderDuration is how long the rendering took
	RenderDuration time.Duration
}

// PDFRenderer defines the interface for rendering HTML to PDF
type PDFRenderer interface {
	// Render converts HTML content to a PDF document
	Render(ctx context.Context, req *RenderRequest) (*RenderResult, error)
	// Close releases any resources held by the renderer
	Close() error
}

// RenderError represents an error during PDF rendering
type RenderError struct {
	Code    string
	Message string
	Cause   error
}

func (e *RenderError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *RenderError) Unwrap() error {
	return e.Cause
}

// Error codes for rendering failures
const (
	ErrCodeRenderTimeout    = "RENDER_TIMEOUT"
	ErrCodeRenderFailed     = "RENDER_FAILED"
	ErrCodeInvalidHTML      = "INVALID_HTML"
	ErrCodeBinaryNotFound   = "BINARY_NOT_FOUND"
	ErrCodeInvalidPaperSize = "INVALID_PAPER_SIZE"
	ErrCodeStorageFailed    = "STORAGE_FAILED"
)

// NewRenderError creates a new RenderError
func NewRenderError(code, message string, cause error) *RenderError {
	return &RenderError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}
