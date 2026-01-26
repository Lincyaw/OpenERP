package catalog

import (
	"context"
	"errors"
	"testing"

	"github.com/erp/backend/internal/domain/catalog"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProductRepository is a mock implementation of catalog.ProductRepository for testing
type mockProductRepository struct {
	products    map[string]*catalog.Product
	returnError error
}

func newMockProductRepository() *mockProductRepository {
	return &mockProductRepository{
		products: make(map[string]*catalog.Product),
	}
}

func (m *mockProductRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*catalog.Product, error) {
	if m.returnError != nil {
		return nil, m.returnError
	}
	key := tenantID.String() + ":" + id.String()
	product, ok := m.products[key]
	if !ok {
		return nil, errors.New("product not found")
	}
	return product, nil
}

func (m *mockProductRepository) FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]catalog.Product, error) {
	if m.returnError != nil {
		return nil, m.returnError
	}
	var products []catalog.Product
	for _, id := range ids {
		key := tenantID.String() + ":" + id.String()
		if product, ok := m.products[key]; ok {
			products = append(products, *product)
		}
	}
	return products, nil
}

func (m *mockProductRepository) addProduct(tenantID, productID uuid.UUID, product *catalog.Product) {
	key := tenantID.String() + ":" + productID.String()
	m.products[key] = product
}

// Implement remaining interface methods as no-ops for the mock
func (m *mockProductRepository) FindByID(ctx context.Context, id uuid.UUID) (*catalog.Product, error) {
	return nil, errors.New("not implemented")
}

func (m *mockProductRepository) FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*catalog.Product, error) {
	return nil, errors.New("not implemented")
}

func (m *mockProductRepository) FindByBarcode(ctx context.Context, tenantID uuid.UUID, barcode string) (*catalog.Product, error) {
	return nil, errors.New("not implemented")
}

func (m *mockProductRepository) FindAll(ctx context.Context, filter shared.Filter) ([]catalog.Product, error) {
	return nil, errors.New("not implemented")
}

func (m *mockProductRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]catalog.Product, error) {
	return nil, errors.New("not implemented")
}

func (m *mockProductRepository) FindByCategory(ctx context.Context, tenantID, categoryID uuid.UUID, filter shared.Filter) ([]catalog.Product, error) {
	return nil, errors.New("not implemented")
}

func (m *mockProductRepository) FindByCategories(ctx context.Context, tenantID uuid.UUID, categoryIDs []uuid.UUID, filter shared.Filter) ([]catalog.Product, error) {
	return nil, errors.New("not implemented")
}

func (m *mockProductRepository) FindActive(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]catalog.Product, error) {
	return nil, errors.New("not implemented")
}

func (m *mockProductRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status catalog.ProductStatus, filter shared.Filter) ([]catalog.Product, error) {
	return nil, errors.New("not implemented")
}

func (m *mockProductRepository) FindByCodes(ctx context.Context, tenantID uuid.UUID, codes []string) ([]catalog.Product, error) {
	return nil, errors.New("not implemented")
}

func (m *mockProductRepository) Save(ctx context.Context, product *catalog.Product) error {
	return errors.New("not implemented")
}

func (m *mockProductRepository) SaveBatch(ctx context.Context, products []*catalog.Product) error {
	return errors.New("not implemented")
}

func (m *mockProductRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return errors.New("not implemented")
}

func (m *mockProductRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	return errors.New("not implemented")
}

func (m *mockProductRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	return 0, errors.New("not implemented")
}

func (m *mockProductRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	return 0, errors.New("not implemented")
}

func (m *mockProductRepository) CountByCategory(ctx context.Context, tenantID, categoryID uuid.UUID) (int64, error) {
	return 0, errors.New("not implemented")
}

func (m *mockProductRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status catalog.ProductStatus) (int64, error) {
	return 0, errors.New("not implemented")
}

func (m *mockProductRepository) ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error) {
	return false, errors.New("not implemented")
}

func (m *mockProductRepository) ExistsByBarcode(ctx context.Context, tenantID uuid.UUID, barcode string) (bool, error) {
	return false, errors.New("not implemented")
}

func TestProductSaleValidator_CanBeSold(t *testing.T) {
	tenantID := uuid.New()
	ctx := context.Background()

	t.Run("returns true for active product", func(t *testing.T) {
		repo := newMockProductRepository()
		productID := uuid.New()
		product, _ := catalog.NewProduct(tenantID, "SKU-001", "Test Product", "pcs")
		product.ID = productID
		repo.addProduct(tenantID, productID, product)

		validator := NewProductSaleValidator(repo)
		canBeSold, err := validator.CanBeSold(ctx, tenantID, productID)

		require.NoError(t, err)
		assert.True(t, canBeSold)
	})

	t.Run("returns false for inactive product", func(t *testing.T) {
		repo := newMockProductRepository()
		productID := uuid.New()
		product, _ := catalog.NewProduct(tenantID, "SKU-002", "Test Product", "pcs")
		product.ID = productID
		product.Status = catalog.ProductStatusInactive
		repo.addProduct(tenantID, productID, product)

		validator := NewProductSaleValidator(repo)
		canBeSold, err := validator.CanBeSold(ctx, tenantID, productID)

		require.NoError(t, err)
		assert.False(t, canBeSold)
	})

	t.Run("returns false for discontinued product", func(t *testing.T) {
		repo := newMockProductRepository()
		productID := uuid.New()
		product, _ := catalog.NewProduct(tenantID, "SKU-003", "Test Product", "pcs")
		product.ID = productID
		product.Status = catalog.ProductStatusDiscontinued
		repo.addProduct(tenantID, productID, product)

		validator := NewProductSaleValidator(repo)
		canBeSold, err := validator.CanBeSold(ctx, tenantID, productID)

		require.NoError(t, err)
		assert.False(t, canBeSold)
	})

	t.Run("returns error when product not found", func(t *testing.T) {
		repo := newMockProductRepository()
		validator := NewProductSaleValidator(repo)

		productID := uuid.New()
		canBeSold, err := validator.CanBeSold(ctx, tenantID, productID)

		require.Error(t, err)
		assert.False(t, canBeSold)
	})

	t.Run("returns error on repository error", func(t *testing.T) {
		repo := newMockProductRepository()
		repo.returnError = errors.New("database error")

		validator := NewProductSaleValidator(repo)
		productID := uuid.New()
		canBeSold, err := validator.CanBeSold(ctx, tenantID, productID)

		require.Error(t, err)
		assert.False(t, canBeSold)
	})
}

func TestProductSaleValidator_CanBeSoldBatch(t *testing.T) {
	tenantID := uuid.New()
	ctx := context.Background()

	t.Run("returns correct status for multiple products", func(t *testing.T) {
		repo := newMockProductRepository()

		activeProductID := uuid.New()
		activeProduct, _ := catalog.NewProduct(tenantID, "SKU-001", "Active Product", "pcs")
		activeProduct.ID = activeProductID
		repo.addProduct(tenantID, activeProductID, activeProduct)

		inactiveProductID := uuid.New()
		inactiveProduct, _ := catalog.NewProduct(tenantID, "SKU-002", "Inactive Product", "pcs")
		inactiveProduct.ID = inactiveProductID
		inactiveProduct.Status = catalog.ProductStatusInactive
		repo.addProduct(tenantID, inactiveProductID, inactiveProduct)

		discontinuedProductID := uuid.New()
		discontinuedProduct, _ := catalog.NewProduct(tenantID, "SKU-003", "Discontinued Product", "pcs")
		discontinuedProduct.ID = discontinuedProductID
		discontinuedProduct.Status = catalog.ProductStatusDiscontinued
		repo.addProduct(tenantID, discontinuedProductID, discontinuedProduct)

		validator := NewProductSaleValidator(repo)
		result, err := validator.CanBeSoldBatch(ctx, tenantID, []uuid.UUID{
			activeProductID,
			inactiveProductID,
			discontinuedProductID,
		})

		require.NoError(t, err)
		assert.Len(t, result, 3)
		assert.True(t, result[activeProductID])
		assert.False(t, result[inactiveProductID])
		assert.False(t, result[discontinuedProductID])
	})

	t.Run("returns empty map for empty input", func(t *testing.T) {
		repo := newMockProductRepository()
		validator := NewProductSaleValidator(repo)

		result, err := validator.CanBeSoldBatch(ctx, tenantID, []uuid.UUID{})

		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("omits not found products from result", func(t *testing.T) {
		repo := newMockProductRepository()
		activeProductID := uuid.New()
		activeProduct, _ := catalog.NewProduct(tenantID, "SKU-001", "Active Product", "pcs")
		activeProduct.ID = activeProductID
		repo.addProduct(tenantID, activeProductID, activeProduct)

		notFoundID := uuid.New()

		validator := NewProductSaleValidator(repo)
		result, err := validator.CanBeSoldBatch(ctx, tenantID, []uuid.UUID{
			activeProductID,
			notFoundID,
		})

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.True(t, result[activeProductID])
		_, exists := result[notFoundID]
		assert.False(t, exists)
	})
}
