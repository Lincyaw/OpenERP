package printing

import (
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// PrintTemplate represents a print template for business documents.
// It is the aggregate root for template-related operations.
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
	Status       TemplateStatus // Template status (active/inactive)
}

// NewPrintTemplate creates a new print template
func NewPrintTemplate(
	tenantID uuid.UUID,
	docType DocType,
	name string,
	content string,
	paperSize PaperSize,
) (*PrintTemplate, error) {
	if err := validateDocType(docType); err != nil {
		return nil, err
	}
	if err := validateTemplateName(name); err != nil {
		return nil, err
	}
	if err := validateTemplateContent(content); err != nil {
		return nil, err
	}
	if err := validatePaperSize(paperSize); err != nil {
		return nil, err
	}

	template := &PrintTemplate{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		DocumentType:        docType,
		Name:                strings.TrimSpace(name),
		Content:             content,
		PaperSize:           paperSize,
		Orientation:         OrientationPortrait,
		Margins:             DefaultMargins(),
		IsDefault:           false,
		Status:              TemplateStatusActive,
	}

	// Use receipt margins for receipt paper sizes
	if paperSize.IsReceipt() {
		template.Margins = ReceiptMargins()
	}

	template.AddDomainEvent(NewPrintTemplateCreatedEvent(template))

	return template, nil
}

// Update updates the template's basic information
func (t *PrintTemplate) Update(name, description string) error {
	if err := validateTemplateName(name); err != nil {
		return err
	}

	t.Name = strings.TrimSpace(name)
	t.Description = strings.TrimSpace(description)
	t.UpdatedAt = time.Now()
	t.IncrementVersion()

	t.AddDomainEvent(NewPrintTemplateUpdatedEvent(t))

	return nil
}

// UpdateContent updates the template content
func (t *PrintTemplate) UpdateContent(content string) error {
	if err := validateTemplateContent(content); err != nil {
		return err
	}

	t.Content = content
	t.UpdatedAt = time.Now()
	t.IncrementVersion()

	t.AddDomainEvent(NewPrintTemplateUpdatedEvent(t))

	return nil
}

// SetPaperSize sets the paper size
func (t *PrintTemplate) SetPaperSize(paperSize PaperSize) error {
	if err := validatePaperSize(paperSize); err != nil {
		return err
	}

	t.PaperSize = paperSize
	// Auto-adjust margins for receipt paper
	if paperSize.IsReceipt() && !t.Margins.Equals(ReceiptMargins()) {
		t.Margins = ReceiptMargins()
	}
	t.UpdatedAt = time.Now()
	t.IncrementVersion()

	return nil
}

// SetOrientation sets the page orientation
func (t *PrintTemplate) SetOrientation(orientation Orientation) error {
	if !orientation.IsValid() {
		return shared.NewDomainError("INVALID_ORIENTATION", "Invalid orientation value")
	}

	t.Orientation = orientation
	t.UpdatedAt = time.Now()
	t.IncrementVersion()

	return nil
}

// SetMargins sets the page margins
func (t *PrintTemplate) SetMargins(margins Margins) error {
	t.Margins = margins
	t.UpdatedAt = time.Now()
	t.IncrementVersion()

	t.AddDomainEvent(NewPrintTemplateUpdatedEvent(t))

	return nil
}

// SetAsDefault marks this template as the default for its document type
// Note: The caller should ensure only one template is marked as default per document type
func (t *PrintTemplate) SetAsDefault() error {
	if t.Status != TemplateStatusActive {
		return shared.NewDomainError("INVALID_STATE", "Cannot set inactive template as default")
	}

	if t.IsDefault {
		return nil // Already default, no change needed
	}

	t.IsDefault = true
	t.UpdatedAt = time.Now()
	t.IncrementVersion()

	t.AddDomainEvent(NewPrintTemplateSetAsDefaultEvent(t))

	return nil
}

// UnsetDefault removes the default flag from this template
func (t *PrintTemplate) UnsetDefault() {
	if !t.IsDefault {
		return // Already not default
	}

	t.IsDefault = false
	t.UpdatedAt = time.Now()
	t.IncrementVersion()
}

// Activate activates the template
func (t *PrintTemplate) Activate() error {
	if t.Status == TemplateStatusActive {
		return shared.NewDomainError("ALREADY_ACTIVE", "Template is already active")
	}

	oldStatus := t.Status
	t.Status = TemplateStatusActive
	t.UpdatedAt = time.Now()
	t.IncrementVersion()

	t.AddDomainEvent(NewPrintTemplateStatusChangedEvent(t, oldStatus, TemplateStatusActive))

	return nil
}

// Deactivate deactivates the template
func (t *PrintTemplate) Deactivate() error {
	if t.Status == TemplateStatusInactive {
		return shared.NewDomainError("ALREADY_INACTIVE", "Template is already inactive")
	}

	// Cannot deactivate a default template
	if t.IsDefault {
		return shared.NewDomainError("INVALID_STATE", "Cannot deactivate a default template. Set another template as default first.")
	}

	oldStatus := t.Status
	t.Status = TemplateStatusInactive
	t.UpdatedAt = time.Now()
	t.IncrementVersion()

	t.AddDomainEvent(NewPrintTemplateStatusChangedEvent(t, oldStatus, TemplateStatusInactive))

	return nil
}

// IsActive returns true if the template is active
func (t *PrintTemplate) IsActive() bool {
	return t.Status == TemplateStatusActive
}

// CanBeUsed returns true if the template can be used for printing
func (t *PrintTemplate) CanBeUsed() bool {
	return t.Status == TemplateStatusActive && t.Content != ""
}

// Validation functions

func validateDocType(docType DocType) error {
	if !docType.IsValid() {
		return shared.NewDomainError("INVALID_DOC_TYPE", "Invalid document type")
	}
	return nil
}

func validateTemplateName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return shared.NewDomainError("INVALID_NAME", "Template name cannot be empty")
	}
	if len(trimmed) > 100 {
		return shared.NewDomainError("INVALID_NAME", "Template name cannot exceed 100 characters")
	}
	return nil
}

func validateTemplateContent(content string) error {
	if strings.TrimSpace(content) == "" {
		return shared.NewDomainError("INVALID_CONTENT", "Template content cannot be empty")
	}
	if len(content) > 1024*1024 { // 1MB limit
		return shared.NewDomainError("INVALID_CONTENT", "Template content cannot exceed 1MB")
	}
	return nil
}

func validatePaperSize(paperSize PaperSize) error {
	if !paperSize.IsValid() {
		return shared.NewDomainError("INVALID_PAPER_SIZE", "Invalid paper size")
	}
	return nil
}
