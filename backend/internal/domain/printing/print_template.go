package printing

import (
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// PrintTemplate represents a print template for business documents.
// Templates are now static and loaded from embedded files or external directory.
// This struct is used for rendering purposes only.
type PrintTemplate struct {
	shared.TenantAggregateRoot
	DocumentType DocType        // Type of document this template is for
	Name         string         // Template name
	Description  string         // Template description
	Content      string         // HTML template content
	PaperSize    PaperSize      // Paper size (A4, A5, receipt, etc.)
	Orientation  Orientation    // Page orientation (portrait/landscape)
	Margins      Margins        // Page margins
	IsDefault    bool           // Whether this is the default template for the doc type
	Status       TemplateStatus // Template status (always ACTIVE for static templates)
}

// NewStaticTemplate creates a new static template for rendering
func NewStaticTemplate(
	docType DocType,
	name string,
	description string,
	content string,
	paperSize PaperSize,
	orientation Orientation,
	margins Margins,
	isDefault bool,
) *PrintTemplate {
	return &PrintTemplate{
		TenantAggregateRoot: shared.TenantAggregateRoot{
			BaseAggregateRoot: shared.BaseAggregateRoot{
				BaseEntity: shared.BaseEntity{
					ID: uuid.New(), // Generate a new ID for each instance
				},
			},
		},
		DocumentType: docType,
		Name:         name,
		Description:  description,
		Content:      content,
		PaperSize:    paperSize,
		Orientation:  orientation,
		Margins:      margins,
		IsDefault:    isDefault,
		Status:       TemplateStatusActive,
	}
}
