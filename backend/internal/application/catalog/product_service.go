package catalog

import (
	"context"
	"errors"

	"github.com/erp/backend/internal/domain/catalog"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ProductService handles product-related business operations
type ProductService struct {
	productRepo  catalog.ProductRepository
	categoryRepo catalog.CategoryRepository
}

// NewProductService creates a new ProductService
func NewProductService(
	productRepo catalog.ProductRepository,
	categoryRepo catalog.CategoryRepository,
) *ProductService {
	return &ProductService{
		productRepo:  productRepo,
		categoryRepo: categoryRepo,
	}
}

// Create creates a new product
func (s *ProductService) Create(ctx context.Context, tenantID uuid.UUID, req CreateProductRequest) (*ProductResponse, error) {
	// Check if code already exists
	exists, err := s.productRepo.ExistsByCode(ctx, tenantID, req.Code)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, shared.NewDomainError("ALREADY_EXISTS", "Product with this code already exists")
	}

	// Check if barcode already exists (if provided)
	if req.Barcode != "" {
		exists, err = s.productRepo.ExistsByBarcode(ctx, tenantID, req.Barcode)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, shared.NewDomainError("ALREADY_EXISTS", "Product with this barcode already exists")
		}
	}

	// Validate category exists (if provided)
	if req.CategoryID != nil {
		_, err = s.categoryRepo.FindByIDForTenant(ctx, tenantID, *req.CategoryID)
		if err != nil {
			if errors.Is(err, shared.ErrNotFound) {
				return nil, shared.NewDomainError("INVALID_CATEGORY", "Category not found")
			}
			return nil, err
		}
	}

	// Create the product
	product, err := catalog.NewProduct(tenantID, req.Code, req.Name, req.Unit)
	if err != nil {
		return nil, err
	}

	// Set optional fields
	if req.Description != "" {
		if err := product.Update(req.Name, req.Description); err != nil {
			return nil, err
		}
	}

	if req.Barcode != "" {
		if err := product.SetBarcode(req.Barcode); err != nil {
			return nil, err
		}
	}

	if req.CategoryID != nil {
		product.SetCategory(req.CategoryID)
	}

	// Set prices
	purchasePrice := decimal.Zero
	sellingPrice := decimal.Zero
	if req.PurchasePrice != nil {
		purchasePrice = *req.PurchasePrice
	}
	if req.SellingPrice != nil {
		sellingPrice = *req.SellingPrice
	}
	if !purchasePrice.IsZero() || !sellingPrice.IsZero() {
		purchaseMoney := valueobject.NewMoneyCNY(purchasePrice)
		sellingMoney := valueobject.NewMoneyCNY(sellingPrice)
		if err := product.SetPrices(purchaseMoney, sellingMoney); err != nil {
			return nil, err
		}
	}

	// Set min stock
	if req.MinStock != nil {
		if err := product.SetMinStock(*req.MinStock); err != nil {
			return nil, err
		}
	}

	// Set sort order
	if req.SortOrder != nil {
		product.SetSortOrder(*req.SortOrder)
	}

	// Set attributes
	if req.Attributes != "" {
		if err := product.SetAttributes(req.Attributes); err != nil {
			return nil, err
		}
	}

	// Save the product
	if err := s.productRepo.Save(ctx, product); err != nil {
		return nil, err
	}

	response := ToProductResponse(product)
	return &response, nil
}

// GetByID retrieves a product by ID
func (s *ProductService) GetByID(ctx context.Context, tenantID, productID uuid.UUID) (*ProductResponse, error) {
	product, err := s.productRepo.FindByIDForTenant(ctx, tenantID, productID)
	if err != nil {
		return nil, err
	}

	response := ToProductResponse(product)
	return &response, nil
}

// GetByCode retrieves a product by code
func (s *ProductService) GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*ProductResponse, error) {
	product, err := s.productRepo.FindByCode(ctx, tenantID, code)
	if err != nil {
		return nil, err
	}

	response := ToProductResponse(product)
	return &response, nil
}

// List retrieves a list of products with filtering and pagination
func (s *ProductService) List(ctx context.Context, tenantID uuid.UUID, filter ProductListFilter) ([]ProductListResponse, int64, error) {
	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.OrderBy == "" {
		filter.OrderBy = "sort_order"
	}
	if filter.OrderDir == "" {
		filter.OrderDir = "asc"
	}

	// Build domain filter
	domainFilter := shared.Filter{
		Page:     filter.Page,
		PageSize: filter.PageSize,
		OrderBy:  filter.OrderBy,
		OrderDir: filter.OrderDir,
		Search:   filter.Search,
		Filters:  make(map[string]interface{}),
	}

	// Add specific filters
	if filter.Status != "" {
		domainFilter.Filters["status"] = filter.Status
	}
	if filter.CategoryID != nil {
		domainFilter.Filters["category_id"] = *filter.CategoryID
	}
	if filter.Unit != "" {
		domainFilter.Filters["unit"] = filter.Unit
	}
	if filter.MinPrice != nil {
		domainFilter.Filters["min_price"] = *filter.MinPrice
	}
	if filter.MaxPrice != nil {
		domainFilter.Filters["max_price"] = *filter.MaxPrice
	}
	if filter.HasBarcode != nil {
		domainFilter.Filters["has_barcode"] = *filter.HasBarcode
	}

	// Get products
	products, err := s.productRepo.FindAllForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	// Get total count
	total, err := s.productRepo.CountForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	return ToProductListResponses(products), total, nil
}

// Update updates a product
func (s *ProductService) Update(ctx context.Context, tenantID, productID uuid.UUID, req UpdateProductRequest) (*ProductResponse, error) {
	// Get existing product
	product, err := s.productRepo.FindByIDForTenant(ctx, tenantID, productID)
	if err != nil {
		return nil, err
	}

	// Update name and description
	if req.Name != nil {
		description := product.Description
		if req.Description != nil {
			description = *req.Description
		}
		if err := product.Update(*req.Name, description); err != nil {
			return nil, err
		}
	} else if req.Description != nil {
		if err := product.Update(product.Name, *req.Description); err != nil {
			return nil, err
		}
	}

	// Update barcode
	if req.Barcode != nil {
		// Check for duplicate barcode
		if *req.Barcode != "" && *req.Barcode != product.Barcode {
			exists, err := s.productRepo.ExistsByBarcode(ctx, tenantID, *req.Barcode)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, shared.NewDomainError("ALREADY_EXISTS", "Product with this barcode already exists")
			}
		}
		if err := product.SetBarcode(*req.Barcode); err != nil {
			return nil, err
		}
	}

	// Update category
	if req.CategoryID != nil {
		// Validate category exists
		_, err = s.categoryRepo.FindByIDForTenant(ctx, tenantID, *req.CategoryID)
		if err != nil {
			if errors.Is(err, shared.ErrNotFound) {
				return nil, shared.NewDomainError("INVALID_CATEGORY", "Category not found")
			}
			return nil, err
		}
		product.SetCategory(req.CategoryID)
	}

	// Update prices
	if req.PurchasePrice != nil || req.SellingPrice != nil {
		purchasePrice := product.PurchasePrice
		sellingPrice := product.SellingPrice
		if req.PurchasePrice != nil {
			purchasePrice = *req.PurchasePrice
		}
		if req.SellingPrice != nil {
			sellingPrice = *req.SellingPrice
		}
		purchaseMoney := valueobject.NewMoneyCNY(purchasePrice)
		sellingMoney := valueobject.NewMoneyCNY(sellingPrice)
		if err := product.SetPrices(purchaseMoney, sellingMoney); err != nil {
			return nil, err
		}
	}

	// Update min stock
	if req.MinStock != nil {
		if err := product.SetMinStock(*req.MinStock); err != nil {
			return nil, err
		}
	}

	// Update sort order
	if req.SortOrder != nil {
		product.SetSortOrder(*req.SortOrder)
	}

	// Update attributes
	if req.Attributes != nil {
		if err := product.SetAttributes(*req.Attributes); err != nil {
			return nil, err
		}
	}

	// Save the product
	if err := s.productRepo.Save(ctx, product); err != nil {
		return nil, err
	}

	response := ToProductResponse(product)
	return &response, nil
}

// UpdateCode updates a product's code
func (s *ProductService) UpdateCode(ctx context.Context, tenantID, productID uuid.UUID, newCode string) (*ProductResponse, error) {
	// Get existing product
	product, err := s.productRepo.FindByIDForTenant(ctx, tenantID, productID)
	if err != nil {
		return nil, err
	}

	// Check if new code already exists (if different from current)
	if newCode != product.Code {
		exists, err := s.productRepo.ExistsByCode(ctx, tenantID, newCode)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, shared.NewDomainError("ALREADY_EXISTS", "Product with this code already exists")
		}
	}

	// Update the code
	if err := product.UpdateCode(newCode); err != nil {
		return nil, err
	}

	// Save the product
	if err := s.productRepo.Save(ctx, product); err != nil {
		return nil, err
	}

	response := ToProductResponse(product)
	return &response, nil
}

// Delete deletes a product
func (s *ProductService) Delete(ctx context.Context, tenantID, productID uuid.UUID) error {
	// Verify product exists
	_, err := s.productRepo.FindByIDForTenant(ctx, tenantID, productID)
	if err != nil {
		return err
	}

	// TODO: Check if product is used in any transactions (orders, inventory, etc.)
	// This should be implemented when those modules are available

	return s.productRepo.DeleteForTenant(ctx, tenantID, productID)
}

// Activate activates a product
func (s *ProductService) Activate(ctx context.Context, tenantID, productID uuid.UUID) (*ProductResponse, error) {
	product, err := s.productRepo.FindByIDForTenant(ctx, tenantID, productID)
	if err != nil {
		return nil, err
	}

	if err := product.Activate(); err != nil {
		return nil, err
	}

	if err := s.productRepo.Save(ctx, product); err != nil {
		return nil, err
	}

	response := ToProductResponse(product)
	return &response, nil
}

// Deactivate deactivates a product
func (s *ProductService) Deactivate(ctx context.Context, tenantID, productID uuid.UUID) (*ProductResponse, error) {
	product, err := s.productRepo.FindByIDForTenant(ctx, tenantID, productID)
	if err != nil {
		return nil, err
	}

	if err := product.Deactivate(); err != nil {
		return nil, err
	}

	if err := s.productRepo.Save(ctx, product); err != nil {
		return nil, err
	}

	response := ToProductResponse(product)
	return &response, nil
}

// Discontinue discontinues a product
func (s *ProductService) Discontinue(ctx context.Context, tenantID, productID uuid.UUID) (*ProductResponse, error) {
	product, err := s.productRepo.FindByIDForTenant(ctx, tenantID, productID)
	if err != nil {
		return nil, err
	}

	if err := product.Discontinue(); err != nil {
		return nil, err
	}

	if err := s.productRepo.Save(ctx, product); err != nil {
		return nil, err
	}

	response := ToProductResponse(product)
	return &response, nil
}

// GetByCategory retrieves products by category
func (s *ProductService) GetByCategory(ctx context.Context, tenantID, categoryID uuid.UUID, filter ProductListFilter) ([]ProductListResponse, int64, error) {
	// Verify category exists
	_, err := s.categoryRepo.FindByIDForTenant(ctx, tenantID, categoryID)
	if err != nil {
		return nil, 0, err
	}

	// Set the category filter and use the List method
	filter.CategoryID = &categoryID
	return s.List(ctx, tenantID, filter)
}

// CountByStatus returns product counts by status for a tenant
func (s *ProductService) CountByStatus(ctx context.Context, tenantID uuid.UUID) (map[string]int64, error) {
	counts := make(map[string]int64)

	activeCount, err := s.productRepo.CountByStatus(ctx, tenantID, catalog.ProductStatusActive)
	if err != nil {
		return nil, err
	}
	counts["active"] = activeCount

	inactiveCount, err := s.productRepo.CountByStatus(ctx, tenantID, catalog.ProductStatusInactive)
	if err != nil {
		return nil, err
	}
	counts["inactive"] = inactiveCount

	discontinuedCount, err := s.productRepo.CountByStatus(ctx, tenantID, catalog.ProductStatusDiscontinued)
	if err != nil {
		return nil, err
	}
	counts["discontinued"] = discontinuedCount

	counts["total"] = activeCount + inactiveCount + discontinuedCount

	return counts, nil
}
