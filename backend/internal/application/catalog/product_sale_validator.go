package catalog

import (
	"context"

	"github.com/erp/backend/internal/domain/catalog"
	"github.com/google/uuid"
)

// ProductSaleValidator validates whether products can be sold
// This service can be used by the trade context to check product eligibility
// without directly depending on the catalog domain
type ProductSaleValidator struct {
	productRepo catalog.ProductRepository
}

// NewProductSaleValidator creates a new ProductSaleValidator
func NewProductSaleValidator(productRepo catalog.ProductRepository) *ProductSaleValidator {
	return &ProductSaleValidator{
		productRepo: productRepo,
	}
}

// CanBeSold checks if a product can be sold (is active)
// Returns true if the product can be sold, false otherwise
// Returns an error if the product is not found
func (v *ProductSaleValidator) CanBeSold(ctx context.Context, tenantID, productID uuid.UUID) (bool, error) {
	product, err := v.productRepo.FindByIDForTenant(ctx, tenantID, productID)
	if err != nil {
		return false, err
	}

	return product.CanBeSold(), nil
}

// CanBeSoldBatch checks if multiple products can be sold
// Returns a map of productID -> canBeSold
// Products not found will be omitted from the result (not included as false)
func (v *ProductSaleValidator) CanBeSoldBatch(ctx context.Context, tenantID uuid.UUID, productIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	if len(productIDs) == 0 {
		return make(map[uuid.UUID]bool), nil
	}

	products, err := v.productRepo.FindByIDs(ctx, tenantID, productIDs)
	if err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID]bool, len(products))
	for i := range products {
		result[products[i].ID] = products[i].CanBeSold()
	}

	return result, nil
}
