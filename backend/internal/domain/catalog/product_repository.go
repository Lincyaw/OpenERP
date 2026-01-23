package catalog

import (
	"context"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// ProductRepository defines the interface for product persistence
type ProductRepository interface {
	// FindByID finds a product by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*Product, error)

	// FindByIDForTenant finds a product by ID within a tenant
	FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*Product, error)

	// FindByCode finds a product by its code within a tenant
	FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*Product, error)

	// FindByBarcode finds a product by its barcode within a tenant
	FindByBarcode(ctx context.Context, tenantID uuid.UUID, barcode string) (*Product, error)

	// FindAll finds all products matching the filter
	FindAll(ctx context.Context, filter shared.Filter) ([]Product, error)

	// FindAllForTenant finds all products for a tenant
	FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]Product, error)

	// FindByCategory finds all products in a specific category
	FindByCategory(ctx context.Context, tenantID, categoryID uuid.UUID, filter shared.Filter) ([]Product, error)

	// FindByCategories finds all products in multiple categories
	FindByCategories(ctx context.Context, tenantID uuid.UUID, categoryIDs []uuid.UUID, filter shared.Filter) ([]Product, error)

	// FindActive finds all active products for a tenant
	FindActive(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]Product, error)

	// FindByStatus finds products by status for a tenant
	FindByStatus(ctx context.Context, tenantID uuid.UUID, status ProductStatus, filter shared.Filter) ([]Product, error)

	// FindByIDs finds multiple products by their IDs
	FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]Product, error)

	// FindByCodes finds multiple products by their codes
	FindByCodes(ctx context.Context, tenantID uuid.UUID, codes []string) ([]Product, error)

	// Save creates or updates a product
	Save(ctx context.Context, product *Product) error

	// SaveBatch creates or updates multiple products
	SaveBatch(ctx context.Context, products []*Product) error

	// Delete deletes a product
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteForTenant deletes a product within a tenant
	DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error

	// Count counts products matching the filter
	Count(ctx context.Context, filter shared.Filter) (int64, error)

	// CountForTenant counts products for a tenant
	CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error)

	// CountByCategory counts products in a specific category
	CountByCategory(ctx context.Context, tenantID, categoryID uuid.UUID) (int64, error)

	// CountByStatus counts products by status for a tenant
	CountByStatus(ctx context.Context, tenantID uuid.UUID, status ProductStatus) (int64, error)

	// ExistsByCode checks if a product with the given code exists in the tenant
	ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error)

	// ExistsByBarcode checks if a product with the given barcode exists in the tenant
	ExistsByBarcode(ctx context.Context, tenantID uuid.UUID, barcode string) (bool, error)
}
