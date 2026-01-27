package printing

import (
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// Aggregate type constants
const (
	AggregateTypePrintTemplate = "PrintTemplate"
	AggregateTypePrintJob      = "PrintJob"
)

// Event type constants for PrintTemplate
const (
	EventTypePrintTemplateCreated       = "PrintTemplateCreated"
	EventTypePrintTemplateUpdated       = "PrintTemplateUpdated"
	EventTypePrintTemplateStatusChanged = "PrintTemplateStatusChanged"
	EventTypePrintTemplateSetAsDefault  = "PrintTemplateSetAsDefault"
	EventTypePrintTemplateDeleted       = "PrintTemplateDeleted"
)

// Event type constants for PrintJob
const (
	EventTypePrintJobCreated       = "PrintJobCreated"
	EventTypePrintJobStatusChanged = "PrintJobStatusChanged"
	EventTypePrintJobCompleted     = "PrintJobCompleted"
	EventTypePrintJobFailed        = "PrintJobFailed"
)

// ============================================================================
// PrintTemplate Events
// ============================================================================

// PrintTemplateCreatedEvent is published when a new print template is created
type PrintTemplateCreatedEvent struct {
	shared.BaseDomainEvent
	TemplateID   uuid.UUID `json:"template_id"`
	DocumentType DocType   `json:"document_type"`
	Name         string    `json:"name"`
	PaperSize    PaperSize `json:"paper_size"`
}

// NewPrintTemplateCreatedEvent creates a new PrintTemplateCreatedEvent
func NewPrintTemplateCreatedEvent(template *PrintTemplate) *PrintTemplateCreatedEvent {
	return &PrintTemplateCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypePrintTemplateCreated,
			AggregateTypePrintTemplate,
			template.ID,
			template.TenantID,
		),
		TemplateID:   template.ID,
		DocumentType: template.DocumentType,
		Name:         template.Name,
		PaperSize:    template.PaperSize,
	}
}

// PrintTemplateUpdatedEvent is published when a print template is updated
type PrintTemplateUpdatedEvent struct {
	shared.BaseDomainEvent
	TemplateID   uuid.UUID `json:"template_id"`
	DocumentType DocType   `json:"document_type"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
}

// NewPrintTemplateUpdatedEvent creates a new PrintTemplateUpdatedEvent
func NewPrintTemplateUpdatedEvent(template *PrintTemplate) *PrintTemplateUpdatedEvent {
	return &PrintTemplateUpdatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypePrintTemplateUpdated,
			AggregateTypePrintTemplate,
			template.ID,
			template.TenantID,
		),
		TemplateID:   template.ID,
		DocumentType: template.DocumentType,
		Name:         template.Name,
		Description:  template.Description,
	}
}

// PrintTemplateStatusChangedEvent is published when a template's status changes
type PrintTemplateStatusChangedEvent struct {
	shared.BaseDomainEvent
	TemplateID   uuid.UUID      `json:"template_id"`
	DocumentType DocType        `json:"document_type"`
	Name         string         `json:"name"`
	OldStatus    TemplateStatus `json:"old_status"`
	NewStatus    TemplateStatus `json:"new_status"`
}

// NewPrintTemplateStatusChangedEvent creates a new PrintTemplateStatusChangedEvent
func NewPrintTemplateStatusChangedEvent(template *PrintTemplate, oldStatus, newStatus TemplateStatus) *PrintTemplateStatusChangedEvent {
	return &PrintTemplateStatusChangedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypePrintTemplateStatusChanged,
			AggregateTypePrintTemplate,
			template.ID,
			template.TenantID,
		),
		TemplateID:   template.ID,
		DocumentType: template.DocumentType,
		Name:         template.Name,
		OldStatus:    oldStatus,
		NewStatus:    newStatus,
	}
}

// PrintTemplateSetAsDefaultEvent is published when a template is set as default
type PrintTemplateSetAsDefaultEvent struct {
	shared.BaseDomainEvent
	TemplateID   uuid.UUID `json:"template_id"`
	DocumentType DocType   `json:"document_type"`
	Name         string    `json:"name"`
}

// NewPrintTemplateSetAsDefaultEvent creates a new PrintTemplateSetAsDefaultEvent
func NewPrintTemplateSetAsDefaultEvent(template *PrintTemplate) *PrintTemplateSetAsDefaultEvent {
	return &PrintTemplateSetAsDefaultEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypePrintTemplateSetAsDefault,
			AggregateTypePrintTemplate,
			template.ID,
			template.TenantID,
		),
		TemplateID:   template.ID,
		DocumentType: template.DocumentType,
		Name:         template.Name,
	}
}

// PrintTemplateDeletedEvent is published when a template is deleted
type PrintTemplateDeletedEvent struct {
	shared.BaseDomainEvent
	TemplateID   uuid.UUID `json:"template_id"`
	DocumentType DocType   `json:"document_type"`
	Name         string    `json:"name"`
}

// NewPrintTemplateDeletedEvent creates a new PrintTemplateDeletedEvent
func NewPrintTemplateDeletedEvent(template *PrintTemplate) *PrintTemplateDeletedEvent {
	return &PrintTemplateDeletedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypePrintTemplateDeleted,
			AggregateTypePrintTemplate,
			template.ID,
			template.TenantID,
		),
		TemplateID:   template.ID,
		DocumentType: template.DocumentType,
		Name:         template.Name,
	}
}

// ============================================================================
// PrintJob Events
// ============================================================================

// PrintJobCreatedEvent is published when a new print job is created
type PrintJobCreatedEvent struct {
	shared.BaseDomainEvent
	JobID          uuid.UUID `json:"job_id"`
	TemplateID     uuid.UUID `json:"template_id"`
	DocumentType   DocType   `json:"document_type"`
	DocumentID     uuid.UUID `json:"document_id"`
	DocumentNumber string    `json:"document_number"`
}

// NewPrintJobCreatedEvent creates a new PrintJobCreatedEvent
func NewPrintJobCreatedEvent(job *PrintJob) *PrintJobCreatedEvent {
	return &PrintJobCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypePrintJobCreated,
			AggregateTypePrintJob,
			job.ID,
			job.TenantID,
		),
		JobID:          job.ID,
		TemplateID:     job.TemplateID,
		DocumentType:   job.DocumentType,
		DocumentID:     job.DocumentID,
		DocumentNumber: job.DocumentNumber,
	}
}

// PrintJobStatusChangedEvent is published when a print job's status changes
type PrintJobStatusChangedEvent struct {
	shared.BaseDomainEvent
	JobID          uuid.UUID `json:"job_id"`
	DocumentType   DocType   `json:"document_type"`
	DocumentID     uuid.UUID `json:"document_id"`
	DocumentNumber string    `json:"document_number"`
	OldStatus      JobStatus `json:"old_status"`
	NewStatus      JobStatus `json:"new_status"`
}

// NewPrintJobStatusChangedEvent creates a new PrintJobStatusChangedEvent
func NewPrintJobStatusChangedEvent(job *PrintJob, oldStatus, newStatus JobStatus) *PrintJobStatusChangedEvent {
	return &PrintJobStatusChangedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypePrintJobStatusChanged,
			AggregateTypePrintJob,
			job.ID,
			job.TenantID,
		),
		JobID:          job.ID,
		DocumentType:   job.DocumentType,
		DocumentID:     job.DocumentID,
		DocumentNumber: job.DocumentNumber,
		OldStatus:      oldStatus,
		NewStatus:      newStatus,
	}
}

// PrintJobCompletedEvent is published when a print job is completed successfully
type PrintJobCompletedEvent struct {
	shared.BaseDomainEvent
	JobID          uuid.UUID `json:"job_id"`
	TemplateID     uuid.UUID `json:"template_id"`
	DocumentType   DocType   `json:"document_type"`
	DocumentID     uuid.UUID `json:"document_id"`
	DocumentNumber string    `json:"document_number"`
	PdfURL         string    `json:"pdf_url"`
	Copies         int       `json:"copies"`
}

// NewPrintJobCompletedEvent creates a new PrintJobCompletedEvent
func NewPrintJobCompletedEvent(job *PrintJob) *PrintJobCompletedEvent {
	return &PrintJobCompletedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypePrintJobCompleted,
			AggregateTypePrintJob,
			job.ID,
			job.TenantID,
		),
		JobID:          job.ID,
		TemplateID:     job.TemplateID,
		DocumentType:   job.DocumentType,
		DocumentID:     job.DocumentID,
		DocumentNumber: job.DocumentNumber,
		PdfURL:         job.PdfURL,
		Copies:         job.Copies,
	}
}

// PrintJobFailedEvent is published when a print job fails
type PrintJobFailedEvent struct {
	shared.BaseDomainEvent
	JobID          uuid.UUID `json:"job_id"`
	TemplateID     uuid.UUID `json:"template_id"`
	DocumentType   DocType   `json:"document_type"`
	DocumentID     uuid.UUID `json:"document_id"`
	DocumentNumber string    `json:"document_number"`
	ErrorMessage   string    `json:"error_message"`
}

// NewPrintJobFailedEvent creates a new PrintJobFailedEvent
func NewPrintJobFailedEvent(job *PrintJob) *PrintJobFailedEvent {
	return &PrintJobFailedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypePrintJobFailed,
			AggregateTypePrintJob,
			job.ID,
			job.TenantID,
		),
		JobID:          job.ID,
		TemplateID:     job.TemplateID,
		DocumentType:   job.DocumentType,
		DocumentID:     job.DocumentID,
		DocumentNumber: job.DocumentNumber,
		ErrorMessage:   job.ErrorMessage,
	}
}
