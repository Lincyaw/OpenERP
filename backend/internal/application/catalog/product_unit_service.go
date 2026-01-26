package catalog

import (
	"context"
	"errors"

	"github.com/erp/backend/internal/domain/catalog"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ProductUnitService handles product unit operations
type ProductUnitService struct {
	productReader   catalog.ProductReader
	productUnitRepo catalog.ProductUnitRepository
}

// NewProductUnitService creates a new ProductUnitService
// Note: Uses narrower ProductReader interface as only read operations are needed.
// The full ProductRepository also satisfies this interface.
func NewProductUnitService(
	productReader catalog.ProductReader,
	productUnitRepo catalog.ProductUnitRepository,
) *ProductUnitService {
	return &ProductUnitService{
		productReader:   productReader,
		productUnitRepo: productUnitRepo,
	}
}

// Create creates a new product unit
func (s *ProductUnitService) Create(ctx context.Context, tenantID, productID uuid.UUID, req CreateProductUnitRequest) (*ProductUnitResponse, error) {
	// Verify product exists
	product, err := s.productReader.FindByIDForTenant(ctx, tenantID, productID)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, shared.NewDomainError("PRODUCT_NOT_FOUND", "Product not found")
		}
		return nil, err
	}

	// Check if unit code is same as base unit
	if req.UnitCode == product.Unit {
		return nil, shared.NewDomainError("DUPLICATE_UNIT_CODE", "Unit code cannot be the same as base unit")
	}

	// Check if unit code already exists for this product
	exists, err := s.productUnitRepo.ExistsByProductIDAndCode(ctx, tenantID, productID, req.UnitCode)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, shared.NewDomainError("DUPLICATE_UNIT_CODE", "Unit code already exists for this product")
	}

	// Create the product unit
	unit, err := catalog.NewProductUnit(tenantID, productID, req.UnitCode, req.UnitName, req.ConversionRate)
	if err != nil {
		return nil, err
	}

	// Set prices if provided
	purchasePrice := decimal.Zero
	sellingPrice := decimal.Zero
	if req.DefaultPurchasePrice != nil {
		purchasePrice = *req.DefaultPurchasePrice
	}
	if req.DefaultSellingPrice != nil {
		sellingPrice = *req.DefaultSellingPrice
	}
	if err := unit.SetPrices(purchasePrice, sellingPrice); err != nil {
		return nil, err
	}

	// Set sort order if provided
	if req.SortOrder != nil {
		unit.SetSortOrder(*req.SortOrder)
	}

	// Handle default unit flags
	if req.IsDefaultPurchaseUnit {
		if err := s.productUnitRepo.ClearDefaultPurchaseUnit(ctx, tenantID, productID); err != nil {
			return nil, err
		}
		unit.SetAsDefaultPurchaseUnit(true)
	}
	if req.IsDefaultSalesUnit {
		if err := s.productUnitRepo.ClearDefaultSalesUnit(ctx, tenantID, productID); err != nil {
			return nil, err
		}
		unit.SetAsDefaultSalesUnit(true)
	}

	// Save the unit
	if err := s.productUnitRepo.Save(ctx, unit); err != nil {
		return nil, err
	}

	response := ToProductUnitResponse(unit)
	return &response, nil
}

// GetByID gets a product unit by ID
func (s *ProductUnitService) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*ProductUnitResponse, error) {
	unit, err := s.productUnitRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, shared.NewDomainError("PRODUCT_UNIT_NOT_FOUND", "Product unit not found")
		}
		return nil, err
	}

	response := ToProductUnitResponse(unit)
	return &response, nil
}

// ListByProduct lists all units for a product
func (s *ProductUnitService) ListByProduct(ctx context.Context, tenantID, productID uuid.UUID) ([]ProductUnitResponse, error) {
	// Verify product exists
	_, err := s.productReader.FindByIDForTenant(ctx, tenantID, productID)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, shared.NewDomainError("PRODUCT_NOT_FOUND", "Product not found")
		}
		return nil, err
	}

	units, err := s.productUnitRepo.FindByProductID(ctx, tenantID, productID)
	if err != nil {
		return nil, err
	}

	return ToProductUnitResponses(units), nil
}

// Update updates a product unit
func (s *ProductUnitService) Update(ctx context.Context, tenantID, id uuid.UUID, req UpdateProductUnitRequest) (*ProductUnitResponse, error) {
	unit, err := s.productUnitRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, shared.NewDomainError("PRODUCT_UNIT_NOT_FOUND", "Product unit not found")
		}
		return nil, err
	}

	// Update name and conversion rate if provided
	if req.UnitName != nil || req.ConversionRate != nil {
		unitName := unit.UnitName
		if req.UnitName != nil {
			unitName = *req.UnitName
		}
		conversionRate := unit.ConversionRate
		if req.ConversionRate != nil {
			conversionRate = *req.ConversionRate
		}
		if err := unit.Update(unitName, conversionRate); err != nil {
			return nil, err
		}
	}

	// Update prices if provided
	if req.DefaultPurchasePrice != nil || req.DefaultSellingPrice != nil {
		purchasePrice := unit.DefaultPurchasePrice
		sellingPrice := unit.DefaultSellingPrice
		if req.DefaultPurchasePrice != nil {
			purchasePrice = *req.DefaultPurchasePrice
		}
		if req.DefaultSellingPrice != nil {
			sellingPrice = *req.DefaultSellingPrice
		}
		if err := unit.SetPrices(purchasePrice, sellingPrice); err != nil {
			return nil, err
		}
	}

	// Update sort order if provided
	if req.SortOrder != nil {
		unit.SetSortOrder(*req.SortOrder)
	}

	// Handle default unit flags
	if req.IsDefaultPurchaseUnit != nil && *req.IsDefaultPurchaseUnit {
		if err := s.productUnitRepo.ClearDefaultPurchaseUnit(ctx, tenantID, unit.ProductID); err != nil {
			return nil, err
		}
		unit.SetAsDefaultPurchaseUnit(true)
	} else if req.IsDefaultPurchaseUnit != nil && !*req.IsDefaultPurchaseUnit {
		unit.SetAsDefaultPurchaseUnit(false)
	}

	if req.IsDefaultSalesUnit != nil && *req.IsDefaultSalesUnit {
		if err := s.productUnitRepo.ClearDefaultSalesUnit(ctx, tenantID, unit.ProductID); err != nil {
			return nil, err
		}
		unit.SetAsDefaultSalesUnit(true)
	} else if req.IsDefaultSalesUnit != nil && !*req.IsDefaultSalesUnit {
		unit.SetAsDefaultSalesUnit(false)
	}

	// Save the unit
	if err := s.productUnitRepo.Save(ctx, unit); err != nil {
		return nil, err
	}

	response := ToProductUnitResponse(unit)
	return &response, nil
}

// Delete deletes a product unit
func (s *ProductUnitService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	// Verify unit exists
	_, err := s.productUnitRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return shared.NewDomainError("PRODUCT_UNIT_NOT_FOUND", "Product unit not found")
		}
		return err
	}

	return s.productUnitRepo.DeleteForTenant(ctx, tenantID, id)
}

// ConvertQuantity converts quantity between units
func (s *ProductUnitService) ConvertQuantity(ctx context.Context, tenantID, productID uuid.UUID, req ConvertUnitRequest) (*ConvertUnitResponse, error) {
	// Get product to check base unit
	product, err := s.productReader.FindByIDForTenant(ctx, tenantID, productID)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, shared.NewDomainError("PRODUCT_NOT_FOUND", "Product not found")
		}
		return nil, err
	}

	var fromRate, toRate decimal.Decimal

	// Get conversion rate for from unit
	if req.FromUnitCode == product.Unit {
		fromRate = decimal.NewFromInt(1)
	} else {
		fromUnit, err := s.productUnitRepo.FindByProductIDAndCode(ctx, tenantID, productID, req.FromUnitCode)
		if err != nil {
			if errors.Is(err, shared.ErrNotFound) {
				return nil, shared.NewDomainError("UNIT_NOT_FOUND", "From unit not found for this product")
			}
			return nil, err
		}
		fromRate = fromUnit.ConversionRate
	}

	// Get conversion rate for to unit
	if req.ToUnitCode == product.Unit {
		toRate = decimal.NewFromInt(1)
	} else {
		toUnit, err := s.productUnitRepo.FindByProductIDAndCode(ctx, tenantID, productID, req.ToUnitCode)
		if err != nil {
			if errors.Is(err, shared.ErrNotFound) {
				return nil, shared.NewDomainError("UNIT_NOT_FOUND", "To unit not found for this product")
			}
			return nil, err
		}
		toRate = toUnit.ConversionRate
	}

	// Convert: first to base unit, then to target unit
	// baseQty = fromQty * fromRate
	// toQty = baseQty / toRate
	baseQty := req.Quantity.Mul(fromRate)
	toQty := baseQty.Div(toRate).Round(4)

	return &ConvertUnitResponse{
		FromQuantity: req.Quantity,
		FromUnitCode: req.FromUnitCode,
		ToQuantity:   toQty,
		ToUnitCode:   req.ToUnitCode,
	}, nil
}

// GetDefaultPurchaseUnit gets the default purchase unit for a product
func (s *ProductUnitService) GetDefaultPurchaseUnit(ctx context.Context, tenantID, productID uuid.UUID) (*ProductUnitResponse, error) {
	unit, err := s.productUnitRepo.FindDefaultPurchaseUnit(ctx, tenantID, productID)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, nil // Return nil if no default unit is set
		}
		return nil, err
	}

	response := ToProductUnitResponse(unit)
	return &response, nil
}

// GetDefaultSalesUnit gets the default sales unit for a product
func (s *ProductUnitService) GetDefaultSalesUnit(ctx context.Context, tenantID, productID uuid.UUID) (*ProductUnitResponse, error) {
	unit, err := s.productUnitRepo.FindDefaultSalesUnit(ctx, tenantID, productID)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, nil // Return nil if no default unit is set
		}
		return nil, err
	}

	response := ToProductUnitResponse(unit)
	return &response, nil
}
