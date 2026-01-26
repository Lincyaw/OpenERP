package catalog

import (
	"context"
	"errors"
	"testing"

	"github.com/erp/backend/internal/domain/catalog"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProductReader is a mock implementation of catalog.ProductReader for testing
// This demonstrates the improved testability from interface segregation - we only
// need to implement the 6 methods from ProductReader instead of the full 22-method
// ProductRepository interface.
type mockProductReader struct {
	products    map[string]*catalog.Product
	returnError error
}

func newMockProductReader() *mockProductReader {
	return &mockProductReader{
		products: make(map[string]*catalog.Product),
	}
}

func (m *mockProductReader) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*catalog.Product, error) {
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

func (m *mockProductReader) FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]catalog.Product, error) {
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

func (m *mockProductReader) addProduct(tenantID, productID uuid.UUID, product *catalog.Product) {
	key := tenantID.String() + ":" + productID.String()
	m.products[key] = product
}

// Implement remaining ProductReader interface methods
func (m *mockProductReader) FindByID(ctx context.Context, id uuid.UUID) (*catalog.Product, error) {
	return nil, errors.New("not implemented")
}

func (m *mockProductReader) FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*catalog.Product, error) {
	return nil, errors.New("not implemented")
}

func (m *mockProductReader) FindByBarcode(ctx context.Context, tenantID uuid.UUID, barcode string) (*catalog.Product, error) {
	return nil, errors.New("not implemented")
}

func (m *mockProductReader) FindByCodes(ctx context.Context, tenantID uuid.UUID, codes []string) ([]catalog.Product, error) {
	return nil, errors.New("not implemented")
}

// Compile-time interface check
var _ catalog.ProductReader = (*mockProductReader)(nil)

func TestProductSaleValidator_CanBeSold(t *testing.T) {
	tenantID := uuid.New()
	ctx := context.Background()

	t.Run("returns true for active product", func(t *testing.T) {
		reader := newMockProductReader()
		productID := uuid.New()
		product, _ := catalog.NewProduct(tenantID, "SKU-001", "Test Product", "pcs")
		product.ID = productID
		reader.addProduct(tenantID, productID, product)

		validator := NewProductSaleValidator(reader)
		canBeSold, err := validator.CanBeSold(ctx, tenantID, productID)

		require.NoError(t, err)
		assert.True(t, canBeSold)
	})

	t.Run("returns false for inactive product", func(t *testing.T) {
		reader := newMockProductReader()
		productID := uuid.New()
		product, _ := catalog.NewProduct(tenantID, "SKU-002", "Test Product", "pcs")
		product.ID = productID
		product.Status = catalog.ProductStatusInactive
		reader.addProduct(tenantID, productID, product)

		validator := NewProductSaleValidator(reader)
		canBeSold, err := validator.CanBeSold(ctx, tenantID, productID)

		require.NoError(t, err)
		assert.False(t, canBeSold)
	})

	t.Run("returns false for discontinued product", func(t *testing.T) {
		reader := newMockProductReader()
		productID := uuid.New()
		product, _ := catalog.NewProduct(tenantID, "SKU-003", "Test Product", "pcs")
		product.ID = productID
		product.Status = catalog.ProductStatusDiscontinued
		reader.addProduct(tenantID, productID, product)

		validator := NewProductSaleValidator(reader)
		canBeSold, err := validator.CanBeSold(ctx, tenantID, productID)

		require.NoError(t, err)
		assert.False(t, canBeSold)
	})

	t.Run("returns error when product not found", func(t *testing.T) {
		reader := newMockProductReader()
		validator := NewProductSaleValidator(reader)

		productID := uuid.New()
		canBeSold, err := validator.CanBeSold(ctx, tenantID, productID)

		require.Error(t, err)
		assert.False(t, canBeSold)
	})

	t.Run("returns error on repository error", func(t *testing.T) {
		reader := newMockProductReader()
		reader.returnError = errors.New("database error")

		validator := NewProductSaleValidator(reader)
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
		reader := newMockProductReader()

		activeProductID := uuid.New()
		activeProduct, _ := catalog.NewProduct(tenantID, "SKU-001", "Active Product", "pcs")
		activeProduct.ID = activeProductID
		reader.addProduct(tenantID, activeProductID, activeProduct)

		inactiveProductID := uuid.New()
		inactiveProduct, _ := catalog.NewProduct(tenantID, "SKU-002", "Inactive Product", "pcs")
		inactiveProduct.ID = inactiveProductID
		inactiveProduct.Status = catalog.ProductStatusInactive
		reader.addProduct(tenantID, inactiveProductID, inactiveProduct)

		discontinuedProductID := uuid.New()
		discontinuedProduct, _ := catalog.NewProduct(tenantID, "SKU-003", "Discontinued Product", "pcs")
		discontinuedProduct.ID = discontinuedProductID
		discontinuedProduct.Status = catalog.ProductStatusDiscontinued
		reader.addProduct(tenantID, discontinuedProductID, discontinuedProduct)

		validator := NewProductSaleValidator(reader)
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
		reader := newMockProductReader()
		validator := NewProductSaleValidator(reader)

		result, err := validator.CanBeSoldBatch(ctx, tenantID, []uuid.UUID{})

		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("omits not found products from result", func(t *testing.T) {
		reader := newMockProductReader()
		activeProductID := uuid.New()
		activeProduct, _ := catalog.NewProduct(tenantID, "SKU-001", "Active Product", "pcs")
		activeProduct.ID = activeProductID
		reader.addProduct(tenantID, activeProductID, activeProduct)

		notFoundID := uuid.New()

		validator := NewProductSaleValidator(reader)
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
