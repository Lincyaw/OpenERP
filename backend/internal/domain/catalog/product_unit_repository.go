package catalog

import (
	"context"

	"github.com/google/uuid"
)

// ProductUnitRepository defines the interface for product unit persistence operations
type ProductUnitRepository interface {
	// FindByID finds a product unit by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*ProductUnit, error)

	// FindByIDForTenant finds a product unit by ID within a tenant
	FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*ProductUnit, error)

	// FindByProductID finds all units for a product
	FindByProductID(ctx context.Context, tenantID, productID uuid.UUID) ([]ProductUnit, error)

	// FindByProductIDAndCode finds a specific unit for a product by code
	FindByProductIDAndCode(ctx context.Context, tenantID, productID uuid.UUID, unitCode string) (*ProductUnit, error)

	// FindDefaultPurchaseUnit finds the default purchase unit for a product
	FindDefaultPurchaseUnit(ctx context.Context, tenantID, productID uuid.UUID) (*ProductUnit, error)

	// FindDefaultSalesUnit finds the default sales unit for a product
	FindDefaultSalesUnit(ctx context.Context, tenantID, productID uuid.UUID) (*ProductUnit, error)

	// Save creates or updates a product unit
	Save(ctx context.Context, unit *ProductUnit) error

	// SaveBatch creates or updates multiple product units
	SaveBatch(ctx context.Context, units []*ProductUnit) error

	// Delete deletes a product unit
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteForTenant deletes a product unit within a tenant
	DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error

	// DeleteByProductID deletes all units for a product
	DeleteByProductID(ctx context.Context, tenantID, productID uuid.UUID) error

	// ClearDefaultPurchaseUnit clears the default purchase unit flag for all units of a product
	ClearDefaultPurchaseUnit(ctx context.Context, tenantID, productID uuid.UUID) error

	// ClearDefaultSalesUnit clears the default sales unit flag for all units of a product
	ClearDefaultSalesUnit(ctx context.Context, tenantID, productID uuid.UUID) error

	// CountByProductID counts units for a product
	CountByProductID(ctx context.Context, tenantID, productID uuid.UUID) (int64, error)

	// ExistsByProductIDAndCode checks if a unit with the given code exists for a product
	ExistsByProductIDAndCode(ctx context.Context, tenantID, productID uuid.UUID, unitCode string) (bool, error)
}
