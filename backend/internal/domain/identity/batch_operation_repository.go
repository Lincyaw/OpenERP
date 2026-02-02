package identity

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// BatchOperationFilter represents filter options for batch operation queries
type BatchOperationFilter struct {
	Page          int
	PageSize      int
	OperationType *BatchOperationType
	Status        *BatchOperationStatus
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
}

// DefaultBatchOperationFilter returns a filter with default values
func DefaultBatchOperationFilter() BatchOperationFilter {
	return BatchOperationFilter{
		Page:     1,
		PageSize: 20,
	}
}

// Limit returns the page size
func (f BatchOperationFilter) Limit() int {
	return f.PageSize
}

// Offset returns the offset for pagination
func (f BatchOperationFilter) Offset() int {
	return (f.Page - 1) * f.PageSize
}

// BatchOperationRepository defines the interface for batch operation persistence
type BatchOperationRepository interface {
	// Create creates a new batch operation
	Create(ctx context.Context, op *BatchOperation) error

	// FindByID finds a batch operation by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*BatchOperation, error)

	// Update updates an existing batch operation
	Update(ctx context.Context, op *BatchOperation) error

	// FindByCreatorID finds batch operations created by a specific user
	FindByCreatorID(ctx context.Context, userID uuid.UUID, filter BatchOperationFilter) ([]BatchOperation, int64, error)

	// FindPending finds batch operations that are pending confirmation
	FindPending(ctx context.Context, filter BatchOperationFilter) ([]BatchOperation, int64, error)

	// FindInProgress finds batch operations that are currently processing
	FindInProgress(ctx context.Context) ([]BatchOperation, error)

	// Delete deletes a batch operation (for cleanup of old operations)
	Delete(ctx context.Context, id uuid.UUID) error
}
