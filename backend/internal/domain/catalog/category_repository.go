package catalog

import (
	"context"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// CategoryRepository defines the interface for category persistence
type CategoryRepository interface {
	// FindByID finds a category by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*Category, error)

	// FindByIDForTenant finds a category by ID within a tenant
	FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*Category, error)

	// FindByCode finds a category by its code within a tenant
	FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*Category, error)

	// FindAll finds all categories matching the filter
	FindAll(ctx context.Context, filter shared.Filter) ([]Category, error)

	// FindAllForTenant finds all categories for a tenant
	FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]Category, error)

	// FindChildren finds all direct children of a category
	FindChildren(ctx context.Context, tenantID, parentID uuid.UUID) ([]Category, error)

	// FindRootCategories finds all root categories for a tenant
	FindRootCategories(ctx context.Context, tenantID uuid.UUID) ([]Category, error)

	// FindDescendants finds all descendants of a category (using materialized path)
	FindDescendants(ctx context.Context, tenantID, categoryID uuid.UUID) ([]Category, error)

	// FindByPath finds a category by its materialized path
	FindByPath(ctx context.Context, tenantID uuid.UUID, path string) (*Category, error)

	// Save creates or updates a category
	Save(ctx context.Context, category *Category) error

	// Delete deletes a category
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteForTenant deletes a category within a tenant
	DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error

	// HasChildren checks if a category has any children
	HasChildren(ctx context.Context, tenantID, categoryID uuid.UUID) (bool, error)

	// HasProducts checks if a category has any associated products
	// Note: This may be implemented later when Product repository is available
	HasProducts(ctx context.Context, tenantID, categoryID uuid.UUID) (bool, error)

	// Count counts categories matching the filter
	Count(ctx context.Context, filter shared.Filter) (int64, error)

	// CountForTenant counts categories for a tenant
	CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error)

	// ExistsByCode checks if a category with the given code exists in the tenant
	ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error)

	// UpdatePath updates the path for a category and its descendants
	// This is used when moving a category to a new parent
	UpdatePath(ctx context.Context, tenantID, categoryID uuid.UUID, newPath string, levelDelta int) error
}
