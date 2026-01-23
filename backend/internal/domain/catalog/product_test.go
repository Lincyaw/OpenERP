package catalog

import (
	"testing"

	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProduct(t *testing.T) {
	tenantID := uuid.New()

	t.Run("creates product with valid inputs", func(t *testing.T) {
		product, err := NewProduct(tenantID, "SKU-001", "Test Product", "pcs")
		require.NoError(t, err)
		require.NotNil(t, product)

		assert.Equal(t, tenantID, product.TenantID)
		assert.Equal(t, "SKU-001", product.Code)
		assert.Equal(t, "Test Product", product.Name)
		assert.Equal(t, "pcs", product.Unit)
		assert.True(t, product.PurchasePrice.IsZero())
		assert.True(t, product.SellingPrice.IsZero())
		assert.True(t, product.MinStock.IsZero())
		assert.Equal(t, ProductStatusActive, product.Status)
		assert.Nil(t, product.CategoryID)
		assert.Empty(t, product.Barcode)
		assert.NotEmpty(t, product.ID)
		assert.Equal(t, 1, product.GetVersion())
	})

	t.Run("converts code to uppercase", func(t *testing.T) {
		product, err := NewProduct(tenantID, "sku-001", "Test Product", "pcs")
		require.NoError(t, err)
		assert.Equal(t, "SKU-001", product.Code)
	})

	t.Run("publishes ProductCreated event", func(t *testing.T) {
		product, err := NewProduct(tenantID, "SKU-002", "Test Product", "pcs")
		require.NoError(t, err)

		events := product.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeProductCreated, events[0].EventType())

		event, ok := events[0].(*ProductCreatedEvent)
		require.True(t, ok)
		assert.Equal(t, product.ID, event.ProductID)
		assert.Equal(t, product.Code, event.Code)
		assert.Equal(t, product.Name, event.Name)
		assert.Equal(t, product.Unit, event.Unit)
	})

	t.Run("fails with empty code", func(t *testing.T) {
		_, err := NewProduct(tenantID, "", "Test Product", "pcs")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "code cannot be empty")
	})

	t.Run("fails with code too long", func(t *testing.T) {
		longCode := "ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCDEFGHIJKLMNOP"
		_, err := NewProduct(tenantID, longCode, "Test Product", "pcs")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot exceed 50 characters")
	})

	t.Run("fails with invalid code characters", func(t *testing.T) {
		_, err := NewProduct(tenantID, "SKU@001", "Test Product", "pcs")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "can only contain letters")
	})

	t.Run("fails with empty name", func(t *testing.T) {
		_, err := NewProduct(tenantID, "SKU-001", "", "pcs")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name cannot be empty")
	})

	t.Run("fails with name too long", func(t *testing.T) {
		longName := string(make([]byte, 201))
		_, err := NewProduct(tenantID, "SKU-001", longName, "pcs")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot exceed 200 characters")
	})

	t.Run("fails with empty unit", func(t *testing.T) {
		_, err := NewProduct(tenantID, "SKU-001", "Test Product", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Unit cannot be empty")
	})

	t.Run("fails with unit too long", func(t *testing.T) {
		longUnit := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		_, err := NewProduct(tenantID, "SKU-001", "Test Product", longUnit)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot exceed 20 characters")
	})

	t.Run("accepts code with underscore and hyphen", func(t *testing.T) {
		product, err := NewProduct(tenantID, "SKU_TEST-001", "Test Product", "pcs")
		require.NoError(t, err)
		assert.Equal(t, "SKU_TEST-001", product.Code)
	})
}

func TestNewProductWithPrices(t *testing.T) {
	tenantID := uuid.New()

	t.Run("creates product with prices", func(t *testing.T) {
		purchasePrice := valueobject.NewMoneyCNYFromFloat(50.00)
		sellingPrice := valueobject.NewMoneyCNYFromFloat(100.00)

		product, err := NewProductWithPrices(tenantID, "SKU-001", "Test Product", "pcs", purchasePrice, sellingPrice)
		require.NoError(t, err)
		require.NotNil(t, product)

		assert.True(t, product.PurchasePrice.Equal(decimal.NewFromFloat(50.00)))
		assert.True(t, product.SellingPrice.Equal(decimal.NewFromFloat(100.00)))
	})

	t.Run("fails with negative purchase price", func(t *testing.T) {
		purchasePrice := valueobject.NewMoneyCNYFromFloat(-10.00)
		sellingPrice := valueobject.NewMoneyCNYFromFloat(100.00)

		_, err := NewProductWithPrices(tenantID, "SKU-001", "Test Product", "pcs", purchasePrice, sellingPrice)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be negative")
	})
}

func TestProductUpdate(t *testing.T) {
	tenantID := uuid.New()
	product, _ := NewProduct(tenantID, "SKU-001", "Original Name", "pcs")
	product.ClearDomainEvents()

	t.Run("updates name and description", func(t *testing.T) {
		originalVersion := product.GetVersion()
		err := product.Update("Updated Name", "New description")
		require.NoError(t, err)

		assert.Equal(t, "Updated Name", product.Name)
		assert.Equal(t, "New description", product.Description)
		assert.Equal(t, originalVersion+1, product.GetVersion())
	})

	t.Run("publishes ProductUpdated event", func(t *testing.T) {
		product.ClearDomainEvents()
		err := product.Update("Another Name", "Another description")
		require.NoError(t, err)

		events := product.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeProductUpdated, events[0].EventType())
	})

	t.Run("fails with empty name", func(t *testing.T) {
		err := product.Update("", "Description")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name cannot be empty")
	})
}

func TestProductUpdateCode(t *testing.T) {
	tenantID := uuid.New()
	product, _ := NewProduct(tenantID, "OLD-CODE", "Test", "pcs")
	product.ClearDomainEvents()

	t.Run("updates code", func(t *testing.T) {
		err := product.UpdateCode("NEW-CODE")
		require.NoError(t, err)
		assert.Equal(t, "NEW-CODE", product.Code)
	})

	t.Run("converts code to uppercase", func(t *testing.T) {
		err := product.UpdateCode("lowercase")
		require.NoError(t, err)
		assert.Equal(t, "LOWERCASE", product.Code)
	})

	t.Run("fails with invalid code", func(t *testing.T) {
		err := product.UpdateCode("INVALID@CODE")
		require.Error(t, err)
	})
}

func TestProductBarcode(t *testing.T) {
	tenantID := uuid.New()
	product, _ := NewProduct(tenantID, "SKU-001", "Test", "pcs")

	t.Run("sets barcode", func(t *testing.T) {
		err := product.SetBarcode("1234567890123")
		require.NoError(t, err)
		assert.Equal(t, "1234567890123", product.Barcode)
	})

	t.Run("allows empty barcode", func(t *testing.T) {
		err := product.SetBarcode("")
		require.NoError(t, err)
		assert.Empty(t, product.Barcode)
	})

	t.Run("fails with barcode too long", func(t *testing.T) {
		longBarcode := string(make([]byte, 51))
		err := product.SetBarcode(longBarcode)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot exceed 50 characters")
	})
}

func TestProductCategory(t *testing.T) {
	tenantID := uuid.New()
	product, _ := NewProduct(tenantID, "SKU-001", "Test", "pcs")
	product.ClearDomainEvents()

	t.Run("sets category", func(t *testing.T) {
		categoryID := uuid.New()
		product.SetCategory(&categoryID)

		assert.NotNil(t, product.CategoryID)
		assert.Equal(t, categoryID, *product.CategoryID)
		assert.True(t, product.HasCategory())
	})

	t.Run("clears category", func(t *testing.T) {
		product.SetCategory(nil)
		assert.Nil(t, product.CategoryID)
		assert.False(t, product.HasCategory())
	})

	t.Run("publishes ProductUpdated event", func(t *testing.T) {
		product.ClearDomainEvents()
		categoryID := uuid.New()
		product.SetCategory(&categoryID)

		events := product.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeProductUpdated, events[0].EventType())
	})
}

func TestProductPrices(t *testing.T) {
	tenantID := uuid.New()

	t.Run("sets both prices", func(t *testing.T) {
		product, _ := NewProduct(tenantID, "SKU-001", "Test", "pcs")
		product.ClearDomainEvents()

		purchasePrice := valueobject.NewMoneyCNYFromFloat(50.00)
		sellingPrice := valueobject.NewMoneyCNYFromFloat(100.00)

		err := product.SetPrices(purchasePrice, sellingPrice)
		require.NoError(t, err)

		assert.True(t, product.PurchasePrice.Equal(decimal.NewFromFloat(50.00)))
		assert.True(t, product.SellingPrice.Equal(decimal.NewFromFloat(100.00)))

		events := product.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeProductPriceChanged, events[0].EventType())
	})

	t.Run("fails with negative purchase price", func(t *testing.T) {
		product, _ := NewProduct(tenantID, "SKU-001", "Test", "pcs")
		purchasePrice := valueobject.NewMoneyCNYFromFloat(-10.00)
		sellingPrice := valueobject.NewMoneyCNYFromFloat(100.00)

		err := product.SetPrices(purchasePrice, sellingPrice)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Purchase price cannot be negative")
	})

	t.Run("fails with negative selling price", func(t *testing.T) {
		product, _ := NewProduct(tenantID, "SKU-001", "Test", "pcs")
		purchasePrice := valueobject.NewMoneyCNYFromFloat(50.00)
		sellingPrice := valueobject.NewMoneyCNYFromFloat(-10.00)

		err := product.SetPrices(purchasePrice, sellingPrice)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Selling price cannot be negative")
	})

	t.Run("updates purchase price only", func(t *testing.T) {
		product, _ := NewProduct(tenantID, "SKU-001", "Test", "pcs")
		product.ClearDomainEvents()

		price := valueobject.NewMoneyCNYFromFloat(75.00)
		err := product.UpdatePurchasePrice(price)
		require.NoError(t, err)

		assert.True(t, product.PurchasePrice.Equal(decimal.NewFromFloat(75.00)))
	})

	t.Run("updates selling price only", func(t *testing.T) {
		product, _ := NewProduct(tenantID, "SKU-001", "Test", "pcs")
		product.ClearDomainEvents()

		price := valueobject.NewMoneyCNYFromFloat(150.00)
		err := product.UpdateSellingPrice(price)
		require.NoError(t, err)

		assert.True(t, product.SellingPrice.Equal(decimal.NewFromFloat(150.00)))
	})
}

func TestProductMinStock(t *testing.T) {
	tenantID := uuid.New()
	product, _ := NewProduct(tenantID, "SKU-001", "Test", "pcs")

	t.Run("sets min stock", func(t *testing.T) {
		err := product.SetMinStock(decimal.NewFromInt(10))
		require.NoError(t, err)
		assert.True(t, product.MinStock.Equal(decimal.NewFromInt(10)))
	})

	t.Run("fails with negative min stock", func(t *testing.T) {
		err := product.SetMinStock(decimal.NewFromInt(-5))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be negative")
	})
}

func TestProductSortOrder(t *testing.T) {
	tenantID := uuid.New()
	product, _ := NewProduct(tenantID, "SKU-001", "Test", "pcs")
	originalVersion := product.GetVersion()

	product.SetSortOrder(10)
	assert.Equal(t, 10, product.SortOrder)
	assert.Equal(t, originalVersion+1, product.GetVersion())
}

func TestProductAttributes(t *testing.T) {
	tenantID := uuid.New()
	product, _ := NewProduct(tenantID, "SKU-001", "Test", "pcs")

	t.Run("sets valid JSON attributes", func(t *testing.T) {
		err := product.SetAttributes(`{"color": "red", "size": "large"}`)
		require.NoError(t, err)
		assert.Equal(t, `{"color": "red", "size": "large"}`, product.Attributes)
	})

	t.Run("handles empty attributes", func(t *testing.T) {
		err := product.SetAttributes("")
		require.NoError(t, err)
		assert.Equal(t, "{}", product.Attributes)
	})

	t.Run("fails with invalid JSON", func(t *testing.T) {
		err := product.SetAttributes("not valid json")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "valid JSON object")
	})
}

func TestProductStatus(t *testing.T) {
	tenantID := uuid.New()

	t.Run("activates inactive product", func(t *testing.T) {
		product, _ := NewProduct(tenantID, "SKU-001", "Test", "pcs")
		product.Status = ProductStatusInactive
		product.ClearDomainEvents()

		err := product.Activate()
		require.NoError(t, err)
		assert.Equal(t, ProductStatusActive, product.Status)
		assert.True(t, product.IsActive())

		events := product.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeProductStatusChanged, events[0].EventType())
	})

	t.Run("fails to activate already active product", func(t *testing.T) {
		product, _ := NewProduct(tenantID, "SKU-001", "Test", "pcs")
		err := product.Activate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already active")
	})

	t.Run("fails to activate discontinued product", func(t *testing.T) {
		product, _ := NewProduct(tenantID, "SKU-001", "Test", "pcs")
		product.Status = ProductStatusDiscontinued

		err := product.Activate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot activate a discontinued product")
	})

	t.Run("deactivates active product", func(t *testing.T) {
		product, _ := NewProduct(tenantID, "SKU-001", "Test", "pcs")
		product.ClearDomainEvents()

		err := product.Deactivate()
		require.NoError(t, err)
		assert.Equal(t, ProductStatusInactive, product.Status)
		assert.True(t, product.IsInactive())

		events := product.GetDomainEvents()
		require.Len(t, events, 1)
	})

	t.Run("fails to deactivate already inactive product", func(t *testing.T) {
		product, _ := NewProduct(tenantID, "SKU-001", "Test", "pcs")
		product.Status = ProductStatusInactive

		err := product.Deactivate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already inactive")
	})

	t.Run("fails to deactivate discontinued product", func(t *testing.T) {
		product, _ := NewProduct(tenantID, "SKU-001", "Test", "pcs")
		product.Status = ProductStatusDiscontinued

		err := product.Deactivate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot deactivate a discontinued product")
	})

	t.Run("discontinues active product", func(t *testing.T) {
		product, _ := NewProduct(tenantID, "SKU-001", "Test", "pcs")
		product.ClearDomainEvents()

		err := product.Discontinue()
		require.NoError(t, err)
		assert.Equal(t, ProductStatusDiscontinued, product.Status)
		assert.True(t, product.IsDiscontinued())

		events := product.GetDomainEvents()
		require.Len(t, events, 1)
	})

	t.Run("discontinues inactive product", func(t *testing.T) {
		product, _ := NewProduct(tenantID, "SKU-001", "Test", "pcs")
		product.Status = ProductStatusInactive

		err := product.Discontinue()
		require.NoError(t, err)
		assert.Equal(t, ProductStatusDiscontinued, product.Status)
	})

	t.Run("fails to discontinue already discontinued product", func(t *testing.T) {
		product, _ := NewProduct(tenantID, "SKU-001", "Test", "pcs")
		product.Status = ProductStatusDiscontinued

		err := product.Discontinue()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already discontinued")
	})
}

func TestProductProfitMargin(t *testing.T) {
	tenantID := uuid.New()

	t.Run("calculates profit margin", func(t *testing.T) {
		product, _ := NewProduct(tenantID, "SKU-001", "Test", "pcs")
		product.PurchasePrice = decimal.NewFromFloat(50.00)
		product.SellingPrice = decimal.NewFromFloat(100.00)

		margin := product.GetProfitMargin()
		// Expected: (100 - 50) / 50 * 100 = 100%
		assert.True(t, margin.Equal(decimal.NewFromInt(100)))
	})

	t.Run("returns zero when purchase price is zero", func(t *testing.T) {
		product, _ := NewProduct(tenantID, "SKU-001", "Test", "pcs")
		product.PurchasePrice = decimal.Zero
		product.SellingPrice = decimal.NewFromFloat(100.00)

		margin := product.GetProfitMargin()
		assert.True(t, margin.IsZero())
	})

	t.Run("handles negative margin", func(t *testing.T) {
		product, _ := NewProduct(tenantID, "SKU-001", "Test", "pcs")
		product.PurchasePrice = decimal.NewFromFloat(100.00)
		product.SellingPrice = decimal.NewFromFloat(80.00)

		margin := product.GetProfitMargin()
		// Expected: (80 - 100) / 100 * 100 = -20%
		assert.True(t, margin.Equal(decimal.NewFromInt(-20)))
	})
}

func TestProductPriceMoney(t *testing.T) {
	tenantID := uuid.New()
	product, _ := NewProduct(tenantID, "SKU-001", "Test", "pcs")
	product.PurchasePrice = decimal.NewFromFloat(50.00)
	product.SellingPrice = decimal.NewFromFloat(100.00)

	t.Run("returns purchase price as Money", func(t *testing.T) {
		money := product.GetPurchasePriceMoney()
		assert.True(t, money.Amount().Equal(decimal.NewFromFloat(50.00)))
		assert.Equal(t, valueobject.CNY, money.Currency())
	})

	t.Run("returns selling price as Money", func(t *testing.T) {
		money := product.GetSellingPriceMoney()
		assert.True(t, money.Amount().Equal(decimal.NewFromFloat(100.00)))
		assert.Equal(t, valueobject.CNY, money.Currency())
	})
}

func TestProductEvents(t *testing.T) {
	tenantID := uuid.New()
	product, _ := NewProduct(tenantID, "SKU-001", "Test Product", "pcs")

	t.Run("ProductCreatedEvent has correct fields", func(t *testing.T) {
		events := product.GetDomainEvents()
		require.Len(t, events, 1)

		event, ok := events[0].(*ProductCreatedEvent)
		require.True(t, ok)

		assert.Equal(t, product.ID, event.ProductID)
		assert.Equal(t, product.Code, event.Code)
		assert.Equal(t, product.Name, event.Name)
		assert.Equal(t, product.Unit, event.Unit)
		assert.Equal(t, product.CategoryID, event.CategoryID)
		assert.Equal(t, tenantID, event.TenantID())
		assert.Equal(t, EventTypeProductCreated, event.EventType())
		assert.Equal(t, AggregateTypeProduct, event.AggregateType())
	})

	t.Run("ProductDeletedEvent has correct fields", func(t *testing.T) {
		event := NewProductDeletedEvent(product)
		assert.Equal(t, product.ID, event.ProductID)
		assert.Equal(t, product.Code, event.Code)
		assert.Equal(t, product.CategoryID, event.CategoryID)
		assert.Equal(t, EventTypeProductDeleted, event.EventType())
	})

	t.Run("ProductPriceChangedEvent has correct fields", func(t *testing.T) {
		oldPurchase := decimal.NewFromFloat(50.00)
		oldSelling := decimal.NewFromFloat(100.00)
		product.PurchasePrice = decimal.NewFromFloat(60.00)
		product.SellingPrice = decimal.NewFromFloat(120.00)

		event := NewProductPriceChangedEvent(product, oldPurchase, oldSelling)
		assert.Equal(t, product.ID, event.ProductID)
		assert.Equal(t, product.Code, event.Code)
		assert.True(t, event.OldPurchasePrice.Equal(oldPurchase))
		assert.True(t, event.NewPurchasePrice.Equal(decimal.NewFromFloat(60.00)))
		assert.True(t, event.OldSellingPrice.Equal(oldSelling))
		assert.True(t, event.NewSellingPrice.Equal(decimal.NewFromFloat(120.00)))
		assert.Equal(t, EventTypeProductPriceChanged, event.EventType())
	})

	t.Run("ProductStatusChangedEvent has correct fields", func(t *testing.T) {
		event := NewProductStatusChangedEvent(product, ProductStatusActive, ProductStatusInactive)
		assert.Equal(t, product.ID, event.ProductID)
		assert.Equal(t, product.Code, event.Code)
		assert.Equal(t, ProductStatusActive, event.OldStatus)
		assert.Equal(t, ProductStatusInactive, event.NewStatus)
		assert.Equal(t, EventTypeProductStatusChanged, event.EventType())
	})
}

func TestValidateProductCode(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{"valid uppercase", "SKU001", false},
		{"valid lowercase", "sku001", false},
		{"valid with underscore", "SKU_001", false},
		{"valid with hyphen", "SKU-001", false},
		{"valid with numbers", "PRODUCT01", false},
		{"empty", "", true},
		{"too long", "ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCDEFGHIJKLMNOP", true},
		{"special character @", "SKU@001", true},
		{"special character space", "SKU 001", true},
		{"special character /", "SKU/001", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProductCode(tt.code)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateProductName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid name", "Test Product", false},
		{"valid with spaces", "Premium Widget XL", false},
		{"valid chinese", "测试产品", false},
		{"empty", "", true},
		{"too long", string(make([]byte, 201)), true},
		{"max length", string(make([]byte, 200)), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProductName(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateUnit(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid pcs", "pcs", false},
		{"valid kg", "kg", false},
		{"valid box", "box", false},
		{"valid with numbers", "ml50", false},
		{"empty", "", true},
		{"too long", "ABCDEFGHIJKLMNOPQRSTUVWXYZ", true},
		{"max length", "12345678901234567890", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUnit(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
