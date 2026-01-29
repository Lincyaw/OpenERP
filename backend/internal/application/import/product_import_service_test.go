package importapp

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/catalog"
	"github.com/erp/backend/internal/domain/shared"
	csvimport "github.com/erp/backend/internal/infrastructure/import"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockProductRepository is a mock implementation of catalog.ProductRepository
type MockProductRepository struct {
	mock.Mock
}

func (m *MockProductRepository) FindByID(ctx context.Context, id uuid.UUID) (*catalog.Product, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*catalog.Product), args.Error(1)
}

func (m *MockProductRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*catalog.Product, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*catalog.Product), args.Error(1)
}

func (m *MockProductRepository) FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*catalog.Product, error) {
	args := m.Called(ctx, tenantID, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*catalog.Product), args.Error(1)
}

func (m *MockProductRepository) FindByBarcode(ctx context.Context, tenantID uuid.UUID, barcode string) (*catalog.Product, error) {
	args := m.Called(ctx, tenantID, barcode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*catalog.Product), args.Error(1)
}

func (m *MockProductRepository) FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]catalog.Product, error) {
	args := m.Called(ctx, tenantID, ids)
	return args.Get(0).([]catalog.Product), args.Error(1)
}

func (m *MockProductRepository) FindByCodes(ctx context.Context, tenantID uuid.UUID, codes []string) ([]catalog.Product, error) {
	args := m.Called(ctx, tenantID, codes)
	return args.Get(0).([]catalog.Product), args.Error(1)
}

func (m *MockProductRepository) FindAll(ctx context.Context, filter shared.Filter) ([]catalog.Product, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]catalog.Product), args.Error(1)
}

func (m *MockProductRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]catalog.Product, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]catalog.Product), args.Error(1)
}

func (m *MockProductRepository) FindByCategory(ctx context.Context, tenantID, categoryID uuid.UUID, filter shared.Filter) ([]catalog.Product, error) {
	args := m.Called(ctx, tenantID, categoryID, filter)
	return args.Get(0).([]catalog.Product), args.Error(1)
}

func (m *MockProductRepository) FindByCategories(ctx context.Context, tenantID uuid.UUID, categoryIDs []uuid.UUID, filter shared.Filter) ([]catalog.Product, error) {
	args := m.Called(ctx, tenantID, categoryIDs, filter)
	return args.Get(0).([]catalog.Product), args.Error(1)
}

func (m *MockProductRepository) FindActive(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]catalog.Product, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]catalog.Product), args.Error(1)
}

func (m *MockProductRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status catalog.ProductStatus, filter shared.Filter) ([]catalog.Product, error) {
	args := m.Called(ctx, tenantID, status, filter)
	return args.Get(0).([]catalog.Product), args.Error(1)
}

func (m *MockProductRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockProductRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockProductRepository) CountByCategory(ctx context.Context, tenantID, categoryID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID, categoryID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockProductRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status catalog.ProductStatus) (int64, error) {
	args := m.Called(ctx, tenantID, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockProductRepository) ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error) {
	args := m.Called(ctx, tenantID, code)
	return args.Bool(0), args.Error(1)
}

func (m *MockProductRepository) ExistsByBarcode(ctx context.Context, tenantID uuid.UUID, barcode string) (bool, error) {
	args := m.Called(ctx, tenantID, barcode)
	return args.Bool(0), args.Error(1)
}

func (m *MockProductRepository) Save(ctx context.Context, product *catalog.Product) error {
	args := m.Called(ctx, product)
	return args.Error(0)
}

func (m *MockProductRepository) SaveBatch(ctx context.Context, products []*catalog.Product) error {
	args := m.Called(ctx, products)
	return args.Error(0)
}

func (m *MockProductRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockProductRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

// MockCategoryRepository is a mock implementation of catalog.CategoryRepository
type MockCategoryRepository struct {
	mock.Mock
}

func (m *MockCategoryRepository) FindByID(ctx context.Context, id uuid.UUID) (*catalog.Category, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*catalog.Category), args.Error(1)
}

func (m *MockCategoryRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*catalog.Category, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*catalog.Category), args.Error(1)
}

func (m *MockCategoryRepository) FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*catalog.Category, error) {
	args := m.Called(ctx, tenantID, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*catalog.Category), args.Error(1)
}

func (m *MockCategoryRepository) FindAll(ctx context.Context, filter shared.Filter) ([]catalog.Category, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]catalog.Category), args.Error(1)
}

func (m *MockCategoryRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]catalog.Category, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]catalog.Category), args.Error(1)
}

func (m *MockCategoryRepository) FindChildren(ctx context.Context, tenantID, parentID uuid.UUID) ([]catalog.Category, error) {
	args := m.Called(ctx, tenantID, parentID)
	return args.Get(0).([]catalog.Category), args.Error(1)
}

func (m *MockCategoryRepository) FindRootCategories(ctx context.Context, tenantID uuid.UUID) ([]catalog.Category, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]catalog.Category), args.Error(1)
}

func (m *MockCategoryRepository) FindDescendants(ctx context.Context, tenantID, categoryID uuid.UUID) ([]catalog.Category, error) {
	args := m.Called(ctx, tenantID, categoryID)
	return args.Get(0).([]catalog.Category), args.Error(1)
}

func (m *MockCategoryRepository) FindByPath(ctx context.Context, tenantID uuid.UUID, path string) (*catalog.Category, error) {
	args := m.Called(ctx, tenantID, path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*catalog.Category), args.Error(1)
}

func (m *MockCategoryRepository) Save(ctx context.Context, category *catalog.Category) error {
	args := m.Called(ctx, category)
	return args.Error(0)
}

func (m *MockCategoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCategoryRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *MockCategoryRepository) HasChildren(ctx context.Context, tenantID, categoryID uuid.UUID) (bool, error) {
	args := m.Called(ctx, tenantID, categoryID)
	return args.Bool(0), args.Error(1)
}

func (m *MockCategoryRepository) HasProducts(ctx context.Context, tenantID, categoryID uuid.UUID) (bool, error) {
	args := m.Called(ctx, tenantID, categoryID)
	return args.Bool(0), args.Error(1)
}

func (m *MockCategoryRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCategoryRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCategoryRepository) ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error) {
	args := m.Called(ctx, tenantID, code)
	return args.Bool(0), args.Error(1)
}

func (m *MockCategoryRepository) UpdatePath(ctx context.Context, tenantID, categoryID uuid.UUID, newPath string, levelDelta int) error {
	args := m.Called(ctx, tenantID, categoryID, newPath, levelDelta)
	return args.Error(0)
}

// MockEventPublisher is a mock implementation of shared.EventPublisher
type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) Publish(ctx context.Context, events ...shared.DomainEvent) error {
	args := m.Called(ctx, events)
	return args.Error(0)
}

// Test helpers
func newTestTenantID() uuid.UUID {
	return uuid.MustParse("11111111-1111-1111-1111-111111111111")
}

func newTestUserID() uuid.UUID {
	return uuid.MustParse("22222222-2222-2222-2222-222222222222")
}

func newTestCategoryID() uuid.UUID {
	return uuid.MustParse("33333333-3333-3333-3333-333333333333")
}

func newValidatedSession(tenantID, userID uuid.UUID) *csvimport.ImportSession {
	session := csvimport.NewImportSession(tenantID, userID, csvimport.EntityProducts, "products.csv", 1024)
	session.UpdateState(csvimport.StateValidating)
	session.TotalRows = 2
	session.ValidRows = 2
	session.ErrorRows = 0
	session.UpdateState(csvimport.StateValidated)
	return session
}

func newTestRow(lineNum int, data map[string]string) *csvimport.Row {
	return &csvimport.Row{
		LineNumber: lineNum,
		Data:       data,
	}
}

// Tests for ConflictMode
func TestConflictMode_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		mode     ConflictMode
		expected bool
	}{
		{"skip is valid", ConflictModeSkip, true},
		{"update is valid", ConflictModeUpdate, true},
		{"fail is valid", ConflictModeFail, true},
		{"empty is invalid", ConflictMode(""), false},
		{"unknown is invalid", ConflictMode("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.mode.IsValid())
		})
	}
}

// Tests for validation rules
func TestProductImportService_GetValidationRules(t *testing.T) {
	productRepo := new(MockProductRepository)
	categoryRepo := new(MockCategoryRepository)
	service := NewProductImportService(productRepo, categoryRepo, nil)

	rules := service.GetValidationRules()

	// Verify required fields
	requiredFields := map[string]bool{
		"name":           false,
		"base_unit":      false,
		"purchase_price": false,
		"selling_price":  false,
	}

	for _, rule := range rules {
		if _, ok := requiredFields[rule.Column]; ok {
			requiredFields[rule.Column] = rule.Required
		}
	}

	for field, required := range requiredFields {
		assert.True(t, required, "field %s should be required", field)
	}

	// Verify unique fields
	uniqueFields := map[string]bool{
		"sku":     false,
		"barcode": false,
	}

	for _, rule := range rules {
		if _, ok := uniqueFields[rule.Column]; ok {
			uniqueFields[rule.Column] = rule.Unique
		}
	}

	for field, unique := range uniqueFields {
		assert.True(t, unique, "field %s should be unique", field)
	}
}

// Tests for validateProductStatus
func TestValidateProductStatus(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty is valid", "", false},
		{"active is valid", "active", false},
		{"inactive is valid", "inactive", false},
		{"unknown is invalid", "unknown", true},
		{"discontinued is invalid", "discontinued", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProductStatus(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Tests for validateJSONObject
func TestValidateJSONObject(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty is valid", "", false},
		{"valid empty object", "{}", false},
		{"valid object with data", `{"key": "value"}`, false},
		{"missing opening brace", "key: value}", true},
		{"missing closing brace", "{key: value", true},
		{"array not allowed", "[1,2,3]", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJSONObject(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Tests for LookupCategory
func TestProductImportService_LookupCategory(t *testing.T) {
	ctx := context.Background()
	tenantID := newTestTenantID()
	categoryID := newTestCategoryID()

	t.Run("empty code returns true", func(t *testing.T) {
		productRepo := new(MockProductRepository)
		categoryRepo := new(MockCategoryRepository)
		service := NewProductImportService(productRepo, categoryRepo, nil)

		exists, err := service.LookupCategory(ctx, tenantID, "")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("existing category returns true", func(t *testing.T) {
		productRepo := new(MockProductRepository)
		categoryRepo := new(MockCategoryRepository)
		service := NewProductImportService(productRepo, categoryRepo, nil)

		category, _ := catalog.NewCategory(tenantID, "CAT001", "Test Category")
		categoryRepo.On("FindByCode", ctx, tenantID, "CAT001").Return(category, nil)

		exists, err := service.LookupCategory(ctx, tenantID, "CAT001")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("non-existing category returns false", func(t *testing.T) {
		productRepo := new(MockProductRepository)
		categoryRepo := new(MockCategoryRepository)
		service := NewProductImportService(productRepo, categoryRepo, nil)

		categoryRepo.On("FindByCode", ctx, tenantID, "UNKNOWN").Return(nil, shared.ErrNotFound)

		exists, err := service.LookupCategory(ctx, tenantID, "UNKNOWN")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("database error returns error", func(t *testing.T) {
		productRepo := new(MockProductRepository)
		categoryRepo := new(MockCategoryRepository)
		service := NewProductImportService(productRepo, categoryRepo, nil)

		dbErr := errors.New("database connection failed")
		categoryRepo.On("FindByCode", ctx, tenantID, "CAT001").Return(nil, dbErr)

		_, err := service.LookupCategory(ctx, tenantID, "CAT001")
		assert.Error(t, err)
		assert.Equal(t, dbErr, err)
	})

	_ = categoryID // Suppress unused warning
}

// Tests for LookupUnique
func TestProductImportService_LookupUnique(t *testing.T) {
	ctx := context.Background()
	tenantID := newTestTenantID()

	t.Run("empty value returns false (not duplicate)", func(t *testing.T) {
		productRepo := new(MockProductRepository)
		categoryRepo := new(MockCategoryRepository)
		service := NewProductImportService(productRepo, categoryRepo, nil)

		exists, err := service.LookupUnique(ctx, tenantID, "sku", "")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("existing sku returns true", func(t *testing.T) {
		productRepo := new(MockProductRepository)
		categoryRepo := new(MockCategoryRepository)
		service := NewProductImportService(productRepo, categoryRepo, nil)

		productRepo.On("ExistsByCode", ctx, tenantID, "SKU001").Return(true, nil)

		exists, err := service.LookupUnique(ctx, tenantID, "sku", "SKU001")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("existing barcode returns true", func(t *testing.T) {
		productRepo := new(MockProductRepository)
		categoryRepo := new(MockCategoryRepository)
		service := NewProductImportService(productRepo, categoryRepo, nil)

		productRepo.On("ExistsByBarcode", ctx, tenantID, "123456789").Return(true, nil)

		exists, err := service.LookupUnique(ctx, tenantID, "barcode", "123456789")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("unknown field returns false", func(t *testing.T) {
		productRepo := new(MockProductRepository)
		categoryRepo := new(MockCategoryRepository)
		service := NewProductImportService(productRepo, categoryRepo, nil)

		exists, err := service.LookupUnique(ctx, tenantID, "unknown", "value")
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

// Tests for SKU generation
func TestProductImportService_GenerateSKU(t *testing.T) {
	productRepo := new(MockProductRepository)
	categoryRepo := new(MockCategoryRepository)
	service := NewProductImportService(productRepo, categoryRepo, nil)

	// Reset sequence for clean test
	service.ResetSKUSequence()

	today := time.Now().Format("20060102")

	// Generate first SKU
	sku1, err := service.generateSKU()
	require.NoError(t, err)
	// SKU should start with expected prefix
	assert.True(t, len(sku1) > 0)
	assert.Contains(t, sku1, "PRD-"+today+"-")

	// Generate second SKU
	sku2, err := service.generateSKU()
	require.NoError(t, err)
	assert.Contains(t, sku2, "PRD-"+today+"-")

	// Verify uniqueness
	assert.NotEqual(t, sku1, sku2)
}

// Tests for ValidateWithWarnings
func TestProductImportService_ValidateWithWarnings(t *testing.T) {
	productRepo := new(MockProductRepository)
	categoryRepo := new(MockCategoryRepository)
	service := NewProductImportService(productRepo, categoryRepo, nil)

	t.Run("no warnings for normal prices", func(t *testing.T) {
		row := newTestRow(2, map[string]string{
			"purchase_price": "10.00",
			"selling_price":  "15.00",
		})

		warnings := service.ValidateWithWarnings(row)
		assert.Empty(t, warnings)
	})

	t.Run("warning when selling price less than purchase price", func(t *testing.T) {
		row := newTestRow(2, map[string]string{
			"purchase_price": "20.00",
			"selling_price":  "15.00",
		})

		warnings := service.ValidateWithWarnings(row)
		assert.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "selling price is less than purchase price")
	})

	t.Run("no warning for empty prices", func(t *testing.T) {
		row := newTestRow(2, map[string]string{
			"purchase_price": "",
			"selling_price":  "",
		})

		warnings := service.ValidateWithWarnings(row)
		assert.Empty(t, warnings)
	})
}

// Tests for Import
func TestProductImportService_Import(t *testing.T) {
	ctx := context.Background()
	tenantID := newTestTenantID()
	userID := newTestUserID()

	t.Run("import fails if session not validated", func(t *testing.T) {
		productRepo := new(MockProductRepository)
		categoryRepo := new(MockCategoryRepository)
		service := NewProductImportService(productRepo, categoryRepo, nil)

		session := csvimport.NewImportSession(tenantID, userID, csvimport.EntityProducts, "test.csv", 1024)
		// Session is in "created" state, not validated

		_, err := service.Import(ctx, tenantID, userID, session, nil, ConflictModeSkip)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validated state")
	})

	t.Run("import fails if session has errors", func(t *testing.T) {
		productRepo := new(MockProductRepository)
		categoryRepo := new(MockCategoryRepository)
		service := NewProductImportService(productRepo, categoryRepo, nil)

		session := csvimport.NewImportSession(tenantID, userID, csvimport.EntityProducts, "test.csv", 1024)
		session.UpdateState(csvimport.StateValidating)
		session.ErrorRows = 1
		session.UpdateState(csvimport.StateValidated)

		_, err := service.Import(ctx, tenantID, userID, session, nil, ConflictModeSkip)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation errors")
	})

	t.Run("successful import of new product", func(t *testing.T) {
		productRepo := new(MockProductRepository)
		categoryRepo := new(MockCategoryRepository)
		eventBus := new(MockEventPublisher)
		service := NewProductImportService(productRepo, categoryRepo, eventBus)
		service.ResetSKUSequence()

		session := newValidatedSession(tenantID, userID)

		row := newTestRow(2, map[string]string{
			"name":           "Test Product",
			"sku":            "TEST-001",
			"base_unit":      "pcs",
			"purchase_price": "10.00",
			"selling_price":  "15.00",
			"description":    "A test product",
			"status":         "active",
		})

		productRepo.On("FindByCode", ctx, tenantID, "TEST-001").Return(nil, shared.ErrNotFound)
		productRepo.On("Save", ctx, mock.AnythingOfType("*catalog.Product")).Return(nil)
		eventBus.On("Publish", ctx, mock.Anything).Return(nil)

		result, err := service.Import(ctx, tenantID, userID, session, []*csvimport.Row{row}, ConflictModeSkip)

		require.NoError(t, err)
		assert.Equal(t, 1, result.TotalRows)
		assert.Equal(t, 1, result.ImportedRows)
		assert.Equal(t, 0, result.UpdatedRows)
		assert.Equal(t, 0, result.SkippedRows)
		assert.Equal(t, 0, result.ErrorRows)
	})

	t.Run("skip existing product in skip mode", func(t *testing.T) {
		productRepo := new(MockProductRepository)
		categoryRepo := new(MockCategoryRepository)
		service := NewProductImportService(productRepo, categoryRepo, nil)

		session := newValidatedSession(tenantID, userID)

		row := newTestRow(2, map[string]string{
			"name":           "Test Product",
			"sku":            "EXISTING-001",
			"base_unit":      "pcs",
			"purchase_price": "10.00",
			"selling_price":  "15.00",
		})

		existingProduct, _ := catalog.NewProduct(tenantID, "EXISTING-001", "Existing Product", "pcs")
		productRepo.On("FindByCode", ctx, tenantID, "EXISTING-001").Return(existingProduct, nil)

		result, err := service.Import(ctx, tenantID, userID, session, []*csvimport.Row{row}, ConflictModeSkip)

		require.NoError(t, err)
		assert.Equal(t, 1, result.TotalRows)
		assert.Equal(t, 0, result.ImportedRows)
		assert.Equal(t, 1, result.SkippedRows)
	})

	t.Run("error on existing product in fail mode", func(t *testing.T) {
		productRepo := new(MockProductRepository)
		categoryRepo := new(MockCategoryRepository)
		service := NewProductImportService(productRepo, categoryRepo, nil)

		session := newValidatedSession(tenantID, userID)

		row := newTestRow(2, map[string]string{
			"name":           "Test Product",
			"sku":            "EXISTING-001",
			"base_unit":      "pcs",
			"purchase_price": "10.00",
			"selling_price":  "15.00",
		})

		existingProduct, _ := catalog.NewProduct(tenantID, "EXISTING-001", "Existing Product", "pcs")
		productRepo.On("FindByCode", ctx, tenantID, "EXISTING-001").Return(existingProduct, nil)

		result, err := service.Import(ctx, tenantID, userID, session, []*csvimport.Row{row}, ConflictModeFail)

		require.NoError(t, err)
		assert.Equal(t, 1, result.TotalRows)
		assert.Equal(t, 0, result.ImportedRows)
		assert.Equal(t, 1, result.ErrorRows)
		assert.Len(t, result.Errors, 1)
		assert.Equal(t, csvimport.ErrCodeImportDuplicateInDB, result.Errors[0].Code)
	})

	t.Run("update existing product in update mode", func(t *testing.T) {
		productRepo := new(MockProductRepository)
		categoryRepo := new(MockCategoryRepository)
		eventBus := new(MockEventPublisher)
		service := NewProductImportService(productRepo, categoryRepo, eventBus)

		session := newValidatedSession(tenantID, userID)

		row := newTestRow(2, map[string]string{
			"name":           "Updated Product",
			"sku":            "EXISTING-001",
			"base_unit":      "pcs",
			"purchase_price": "12.00",
			"selling_price":  "18.00",
		})

		existingProduct, _ := catalog.NewProduct(tenantID, "EXISTING-001", "Existing Product", "pcs")
		productRepo.On("FindByCode", ctx, tenantID, "EXISTING-001").Return(existingProduct, nil)
		productRepo.On("Save", ctx, mock.AnythingOfType("*catalog.Product")).Return(nil)
		eventBus.On("Publish", ctx, mock.Anything).Return(nil)

		result, err := service.Import(ctx, tenantID, userID, session, []*csvimport.Row{row}, ConflictModeUpdate)

		require.NoError(t, err)
		assert.Equal(t, 1, result.TotalRows)
		assert.Equal(t, 0, result.ImportedRows)
		assert.Equal(t, 1, result.UpdatedRows)
	})

	t.Run("auto-generate SKU when not provided", func(t *testing.T) {
		productRepo := new(MockProductRepository)
		categoryRepo := new(MockCategoryRepository)
		eventBus := new(MockEventPublisher)
		service := NewProductImportService(productRepo, categoryRepo, eventBus)
		service.ResetSKUSequence()

		session := newValidatedSession(tenantID, userID)

		row := newTestRow(2, map[string]string{
			"name":           "Product Without SKU",
			"sku":            "", // No SKU provided
			"base_unit":      "pcs",
			"purchase_price": "10.00",
			"selling_price":  "15.00",
		})

		// Use mock.Anything for the SKU since it's auto-generated with timestamp-based sequence
		productRepo.On("FindByCode", ctx, tenantID, mock.AnythingOfType("string")).Return(nil, shared.ErrNotFound)
		productRepo.On("Save", ctx, mock.MatchedBy(func(p *catalog.Product) bool {
			// Verify SKU was auto-generated with expected prefix
			today := time.Now().Format("20060102")
			return strings.HasPrefix(p.Code, "PRD-"+today+"-")
		})).Return(nil)
		eventBus.On("Publish", ctx, mock.Anything).Return(nil)

		result, err := service.Import(ctx, tenantID, userID, session, []*csvimport.Row{row}, ConflictModeSkip)

		require.NoError(t, err)
		assert.Equal(t, 1, result.ImportedRows)
	})

	t.Run("import with category", func(t *testing.T) {
		productRepo := new(MockProductRepository)
		categoryRepo := new(MockCategoryRepository)
		eventBus := new(MockEventPublisher)
		service := NewProductImportService(productRepo, categoryRepo, eventBus)
		service.ResetSKUSequence()

		session := newValidatedSession(tenantID, userID)
		categoryID := newTestCategoryID()

		row := newTestRow(2, map[string]string{
			"name":           "Product With Category",
			"sku":            "CAT-PROD-001",
			"category_code":  "ELECTRONICS",
			"base_unit":      "pcs",
			"purchase_price": "100.00",
			"selling_price":  "150.00",
		})

		category, _ := catalog.NewCategory(tenantID, "ELECTRONICS", "Electronics")
		// Set the category ID via reflection or type assertion since NewCategory doesn't set ID
		category.ID = categoryID

		productRepo.On("FindByCode", ctx, tenantID, "CAT-PROD-001").Return(nil, shared.ErrNotFound)
		productRepo.On("ExistsByBarcode", ctx, tenantID, "").Return(false, nil)
		categoryRepo.On("FindByCode", ctx, tenantID, "ELECTRONICS").Return(category, nil)
		productRepo.On("Save", ctx, mock.MatchedBy(func(p *catalog.Product) bool {
			return p.CategoryID != nil && *p.CategoryID == categoryID
		})).Return(nil)
		eventBus.On("Publish", ctx, mock.Anything).Return(nil)

		result, err := service.Import(ctx, tenantID, userID, session, []*csvimport.Row{row}, ConflictModeSkip)

		require.NoError(t, err)
		assert.Equal(t, 1, result.ImportedRows)
	})

	t.Run("import with inactive status", func(t *testing.T) {
		productRepo := new(MockProductRepository)
		categoryRepo := new(MockCategoryRepository)
		eventBus := new(MockEventPublisher)
		service := NewProductImportService(productRepo, categoryRepo, eventBus)
		service.ResetSKUSequence()

		session := newValidatedSession(tenantID, userID)

		row := newTestRow(2, map[string]string{
			"name":           "Inactive Product",
			"sku":            "INACTIVE-001",
			"base_unit":      "pcs",
			"purchase_price": "10.00",
			"selling_price":  "15.00",
			"status":         "inactive",
		})

		productRepo.On("FindByCode", ctx, tenantID, "INACTIVE-001").Return(nil, shared.ErrNotFound)
		productRepo.On("Save", ctx, mock.MatchedBy(func(p *catalog.Product) bool {
			return !p.IsActive()
		})).Return(nil)
		eventBus.On("Publish", ctx, mock.Anything).Return(nil)

		result, err := service.Import(ctx, tenantID, userID, session, []*csvimport.Row{row}, ConflictModeSkip)

		require.NoError(t, err)
		assert.Equal(t, 1, result.ImportedRows)
	})
}

// Tests for context cancellation
func TestProductImportService_Import_ContextCancellation(t *testing.T) {
	tenantID := newTestTenantID()
	userID := newTestUserID()

	productRepo := new(MockProductRepository)
	categoryRepo := new(MockCategoryRepository)
	service := NewProductImportService(productRepo, categoryRepo, nil)

	session := newValidatedSession(tenantID, userID)

	row := newTestRow(2, map[string]string{
		"name":           "Test Product",
		"sku":            "TEST-001",
		"base_unit":      "pcs",
		"purchase_price": "10.00",
		"selling_price":  "15.00",
	})

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := service.Import(ctx, tenantID, userID, session, []*csvimport.Row{row}, ConflictModeSkip)

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.Equal(t, csvimport.StateCancelled, session.State)
}

// Tests for barcode uniqueness
func TestProductImportService_Import_BarcodeUniqueness(t *testing.T) {
	ctx := context.Background()
	tenantID := newTestTenantID()
	userID := newTestUserID()

	t.Run("error when barcode already exists", func(t *testing.T) {
		productRepo := new(MockProductRepository)
		categoryRepo := new(MockCategoryRepository)
		service := NewProductImportService(productRepo, categoryRepo, nil)

		session := newValidatedSession(tenantID, userID)

		row := newTestRow(2, map[string]string{
			"name":           "Test Product",
			"sku":            "NEW-SKU",
			"barcode":        "EXISTING-BARCODE",
			"base_unit":      "pcs",
			"purchase_price": "10.00",
			"selling_price":  "15.00",
		})

		productRepo.On("FindByCode", ctx, tenantID, "NEW-SKU").Return(nil, shared.ErrNotFound)
		productRepo.On("ExistsByBarcode", ctx, tenantID, "EXISTING-BARCODE").Return(true, nil)

		result, err := service.Import(ctx, tenantID, userID, session, []*csvimport.Row{row}, ConflictModeFail)

		require.NoError(t, err)
		assert.Equal(t, 1, result.ErrorRows)
		assert.Len(t, result.Errors, 1)
		assert.Equal(t, csvimport.ErrCodeImportDuplicateInDB, result.Errors[0].Code)
		assert.Equal(t, "barcode", result.Errors[0].Column)
	})
}

// Tests for invalid price format
func TestProductImportService_Import_InvalidPriceFormat(t *testing.T) {
	ctx := context.Background()
	tenantID := newTestTenantID()
	userID := newTestUserID()

	productRepo := new(MockProductRepository)
	categoryRepo := new(MockCategoryRepository)
	service := NewProductImportService(productRepo, categoryRepo, nil)

	session := newValidatedSession(tenantID, userID)

	row := newTestRow(2, map[string]string{
		"name":           "Test Product",
		"sku":            "TEST-001",
		"base_unit":      "pcs",
		"purchase_price": "not-a-number",
		"selling_price":  "15.00",
	})

	result, err := service.Import(ctx, tenantID, userID, session, []*csvimport.Row{row}, ConflictModeSkip)

	require.NoError(t, err)
	assert.Equal(t, 1, result.ErrorRows)
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, csvimport.ErrCodeImportInvalidType, result.Errors[0].Code)
}

// Tests for min stock level
func TestProductImportService_Import_MinStockLevel(t *testing.T) {
	ctx := context.Background()
	tenantID := newTestTenantID()
	userID := newTestUserID()

	productRepo := new(MockProductRepository)
	categoryRepo := new(MockCategoryRepository)
	eventBus := new(MockEventPublisher)
	service := NewProductImportService(productRepo, categoryRepo, eventBus)
	service.ResetSKUSequence()

	session := newValidatedSession(tenantID, userID)

	row := newTestRow(2, map[string]string{
		"name":            "Test Product",
		"sku":             "MIN-STOCK-001",
		"base_unit":       "pcs",
		"purchase_price":  "10.00",
		"selling_price":   "15.00",
		"min_stock_level": "25.5",
	})

	expectedMinStock := decimal.NewFromFloat(25.5)

	productRepo.On("FindByCode", ctx, tenantID, "MIN-STOCK-001").Return(nil, shared.ErrNotFound)
	productRepo.On("Save", ctx, mock.MatchedBy(func(p *catalog.Product) bool {
		return p.MinStock.Equal(expectedMinStock)
	})).Return(nil)
	eventBus.On("Publish", ctx, mock.Anything).Return(nil)

	result, err := service.Import(ctx, tenantID, userID, session, []*csvimport.Row{row}, ConflictModeSkip)

	require.NoError(t, err)
	assert.Equal(t, 1, result.ImportedRows)
}
