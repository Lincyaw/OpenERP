package printing

import (
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// PrintJob represents a print job for a business document.
// Each print job corresponds to one document being printed.
type PrintJob struct {
	shared.TenantAggregateRoot
	TemplateID     uuid.UUID  // Reference to the print template used
	DocumentType   DocType    // Type of document being printed
	DocumentID     uuid.UUID  // ID of the document being printed
	DocumentNumber string     // Document number (for display)
	Status         JobStatus  // Current job status
	Copies         int        // Number of copies to print
	PrinterName    string     // Target printer name (optional, for local printing)
	PdfURL         string     // URL to the generated PDF file
	ErrorMessage   string     // Error message if job failed
	PrintedAt      *time.Time // When the job was printed
	PrintedBy      *uuid.UUID // User who initiated the print
}

// NewPrintJob creates a new print job
func NewPrintJob(
	tenantID uuid.UUID,
	templateID uuid.UUID,
	docType DocType,
	documentID uuid.UUID,
	documentNumber string,
	printedBy uuid.UUID,
) (*PrintJob, error) {
	if templateID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_TEMPLATE", "Template ID cannot be empty")
	}
	if err := validateDocType(docType); err != nil {
		return nil, err
	}
	if documentID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_DOCUMENT", "Document ID cannot be empty")
	}
	if documentNumber == "" {
		return nil, shared.NewDomainError("INVALID_DOCUMENT_NUMBER", "Document number cannot be empty")
	}

	job := &PrintJob{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		TemplateID:          templateID,
		DocumentType:        docType,
		DocumentID:          documentID,
		DocumentNumber:      documentNumber,
		Status:              JobStatusPending,
		Copies:              1,
		PrintedBy:           &printedBy,
	}

	job.AddDomainEvent(NewPrintJobCreatedEvent(job))

	return job, nil
}

// SetCopies sets the number of copies to print
func (j *PrintJob) SetCopies(copies int) error {
	if copies < 1 {
		return shared.NewDomainError("INVALID_COPIES", "Number of copies must be at least 1")
	}
	if copies > 100 {
		return shared.NewDomainError("INVALID_COPIES", "Number of copies cannot exceed 100")
	}

	j.Copies = copies
	j.UpdatedAt = time.Now()

	return nil
}

// SetPrinterName sets the target printer name for local printing
func (j *PrintJob) SetPrinterName(printerName string) {
	j.PrinterName = printerName
	j.UpdatedAt = time.Now()
}

// StartRendering marks the job as rendering
func (j *PrintJob) StartRendering() error {
	if !j.Status.CanTransitionTo(JobStatusRendering) {
		return shared.NewDomainError("INVALID_STATE",
			"Cannot start rendering from status: "+j.Status.String())
	}

	j.Status = JobStatusRendering
	j.UpdatedAt = time.Now()
	j.IncrementVersion()

	j.AddDomainEvent(NewPrintJobStatusChangedEvent(j, JobStatusPending, JobStatusRendering))

	return nil
}

// Complete marks the job as completed with the PDF URL
func (j *PrintJob) Complete(pdfURL string) error {
	if !j.Status.CanTransitionTo(JobStatusCompleted) {
		return shared.NewDomainError("INVALID_STATE",
			"Cannot complete from status: "+j.Status.String())
	}
	if pdfURL == "" {
		return shared.NewDomainError("INVALID_PDF_URL", "PDF URL cannot be empty")
	}

	oldStatus := j.Status
	j.Status = JobStatusCompleted
	j.PdfURL = pdfURL
	now := time.Now()
	j.PrintedAt = &now
	j.UpdatedAt = now
	j.IncrementVersion()

	j.AddDomainEvent(NewPrintJobStatusChangedEvent(j, oldStatus, JobStatusCompleted))
	j.AddDomainEvent(NewPrintJobCompletedEvent(j))

	return nil
}

// Fail marks the job as failed with an error message
func (j *PrintJob) Fail(errorMessage string) error {
	if j.Status.IsTerminal() {
		return shared.NewDomainError("INVALID_STATE",
			"Cannot fail a job that is already in terminal status: "+j.Status.String())
	}

	oldStatus := j.Status
	j.Status = JobStatusFailed
	j.ErrorMessage = errorMessage
	j.UpdatedAt = time.Now()
	j.IncrementVersion()

	j.AddDomainEvent(NewPrintJobStatusChangedEvent(j, oldStatus, JobStatusFailed))
	j.AddDomainEvent(NewPrintJobFailedEvent(j))

	return nil
}

// IsPending returns true if the job is pending
func (j *PrintJob) IsPending() bool {
	return j.Status == JobStatusPending
}

// IsRendering returns true if the job is rendering
func (j *PrintJob) IsRendering() bool {
	return j.Status == JobStatusRendering
}

// IsCompleted returns true if the job is completed
func (j *PrintJob) IsCompleted() bool {
	return j.Status == JobStatusCompleted
}

// IsFailed returns true if the job failed
func (j *PrintJob) IsFailed() bool {
	return j.Status == JobStatusFailed
}

// IsTerminal returns true if the job is in a terminal state
func (j *PrintJob) IsTerminal() bool {
	return j.Status.IsTerminal()
}

// HasPDF returns true if a PDF has been generated
func (j *PrintJob) HasPDF() bool {
	return j.PdfURL != ""
}
