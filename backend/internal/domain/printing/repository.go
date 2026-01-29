package printing

import (
	"context"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// PrintJobRepository defines the interface for print job persistence
type PrintJobRepository interface {
	// FindByID finds a job by ID
	FindByID(ctx context.Context, id uuid.UUID) (*PrintJob, error)

	// FindByIDForTenant finds a job by ID within a specific tenant
	FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*PrintJob, error)

	// FindAll finds all jobs with optional filtering
	FindAll(ctx context.Context, filter shared.Filter) ([]PrintJob, error)

	// FindAllForTenant finds all jobs for a specific tenant
	FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]PrintJob, error)

	// FindByDocument finds all print jobs for a specific document
	FindByDocument(ctx context.Context, tenantID uuid.UUID, docType DocType, documentID uuid.UUID) ([]PrintJob, error)

	// FindRecent finds recent print jobs (within the last N days)
	FindRecent(ctx context.Context, tenantID uuid.UUID, days int, limit int) ([]PrintJob, error)

	// FindPending finds all pending jobs for processing
	FindPending(ctx context.Context, limit int) ([]PrintJob, error)

	// Save saves a job (insert or update)
	Save(ctx context.Context, job *PrintJob) error

	// Delete deletes a job by ID
	Delete(ctx context.Context, id uuid.UUID) error

	// Count returns the total count of jobs matching the filter
	Count(ctx context.Context, filter shared.Filter) (int64, error)

	// CountForTenant returns the total count of jobs for a tenant
	CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error)

	// CountByStatus counts jobs by status for a tenant
	CountByStatus(ctx context.Context, tenantID uuid.UUID, status JobStatus) (int64, error)

	// DeleteOlderThan deletes jobs older than the specified number of days
	// Used for job history cleanup
	DeleteOlderThan(ctx context.Context, days int) (int64, error)
}

// PrintJobFilter extends the standard filter with print job specific criteria
type PrintJobFilter struct {
	shared.Filter
	DocumentType *DocType   // Filter by document type
	DocumentID   *uuid.UUID // Filter by document ID
	Status       *JobStatus // Filter by status
	TemplateID   *uuid.UUID // Filter by template ID
	PrintedByID  *uuid.UUID // Filter by user who printed
	DateFrom     *string    // Filter by date range start (YYYY-MM-DD)
	DateTo       *string    // Filter by date range end (YYYY-MM-DD)
}
