package catalog

import (
	"context"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// ProductAttachmentReader defines the interface for reading individual attachments by ID
type ProductAttachmentReader interface {
	// FindByID finds an attachment by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*ProductAttachment, error)

	// FindByIDForTenant finds an attachment by ID within a tenant
	FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*ProductAttachment, error)

	// FindByIDs finds multiple attachments by their IDs
	FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]ProductAttachment, error)
}

// ProductAttachmentFinder defines the interface for searching and filtering attachments
type ProductAttachmentFinder interface {
	// FindByProduct finds all attachments for a product
	FindByProduct(ctx context.Context, tenantID, productID uuid.UUID, filter shared.Filter) ([]ProductAttachment, error)

	// FindByProductAndStatus finds attachments by product and status
	FindByProductAndStatus(ctx context.Context, tenantID, productID uuid.UUID, status AttachmentStatus, filter shared.Filter) ([]ProductAttachment, error)

	// FindActiveByProduct finds all active attachments for a product
	FindActiveByProduct(ctx context.Context, tenantID, productID uuid.UUID) ([]ProductAttachment, error)

	// FindMainImage finds the main image for a product (if any)
	FindMainImage(ctx context.Context, tenantID, productID uuid.UUID) (*ProductAttachment, error)

	// FindByType finds attachments by type for a product
	FindByType(ctx context.Context, tenantID, productID uuid.UUID, attachmentType AttachmentType) ([]ProductAttachment, error)

	// CountByProduct counts attachments for a product
	CountByProduct(ctx context.Context, tenantID, productID uuid.UUID) (int64, error)

	// CountActiveByProduct counts active attachments for a product
	CountActiveByProduct(ctx context.Context, tenantID, productID uuid.UUID) (int64, error)

	// ExistsByStorageKey checks if an attachment with the given storage key exists
	ExistsByStorageKey(ctx context.Context, tenantID uuid.UUID, storageKey string) (bool, error)
}

// ProductAttachmentWriter defines the interface for attachment persistence (create, update, delete)
type ProductAttachmentWriter interface {
	// Save creates or updates an attachment
	Save(ctx context.Context, attachment *ProductAttachment) error

	// SaveBatch creates or updates multiple attachments
	SaveBatch(ctx context.Context, attachments []*ProductAttachment) error

	// Delete permanently deletes an attachment
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteForTenant permanently deletes an attachment within a tenant
	DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error

	// DeleteByProduct permanently deletes all attachments for a product
	DeleteByProduct(ctx context.Context, tenantID, productID uuid.UUID) error
}

// ProductAttachmentRepository defines the full interface for attachment persistence
// This composite interface combines all attachment repository capabilities.
// Prefer using the specific interfaces (ProductAttachmentReader, ProductAttachmentFinder, ProductAttachmentWriter)
// when possible to improve testability and express intent more clearly.
type ProductAttachmentRepository interface {
	ProductAttachmentReader
	ProductAttachmentFinder
	ProductAttachmentWriter
}
