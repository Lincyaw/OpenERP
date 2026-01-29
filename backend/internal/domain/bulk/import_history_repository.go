package bulk

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ImportHistoryFilter defines the filters for querying import histories
type ImportHistoryFilter struct {
	EntityType  *ImportEntityType // Filter by entity type
	Status      *ImportStatus     // Filter by status
	ImportedBy  *uuid.UUID        // Filter by user who imported
	StartedFrom *time.Time        // Filter by start time (from)
	StartedTo   *time.Time        // Filter by start time (to)
}

// ImportHistoryListResult represents a paginated list of import histories
type ImportHistoryListResult struct {
	Items      []*ImportHistory
	TotalCount int64
	Page       int
	PageSize   int
}

// ImportHistoryRepository defines the interface for import history persistence
type ImportHistoryRepository interface {
	// FindByID finds an import history by ID
	FindByID(ctx context.Context, tenantID, id uuid.UUID) (*ImportHistory, error)

	// FindAll returns all import histories for a tenant with pagination and filtering
	FindAll(ctx context.Context, tenantID uuid.UUID, filter ImportHistoryFilter, page, pageSize int) (*ImportHistoryListResult, error)

	// FindByStatus finds all import histories with a specific status
	FindByStatus(ctx context.Context, tenantID uuid.UUID, status ImportStatus) ([]*ImportHistory, error)

	// FindPending finds all pending import histories (for recovery after restart)
	FindPending(ctx context.Context, tenantID uuid.UUID) ([]*ImportHistory, error)

	// Save saves an import history (create or update)
	Save(ctx context.Context, history *ImportHistory) error

	// Delete deletes an import history by ID
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}
