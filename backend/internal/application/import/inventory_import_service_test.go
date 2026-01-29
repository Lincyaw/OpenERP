package importapp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/catalog"
	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	csvimport "github.com/erp/backend/internal/infrastructure/import"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockInventoryItemRepository is a mock implementation for inventory import tests
type MockInventoryItemRepository struct {
	mock.Mock
}

func (m *MockInventoryItemRepository) FindByID(ctx context.Context, id uuid.UUID) (*inventory.InventoryItem, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*inventory.InventoryItem), args.Error(1)
}

func (m *MockInventoryItemRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*inventory.InventoryItem, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*inventory.InventoryItem), args.Error(1)
}

func (m *MockInventoryItemRepository) FindByWarehouseAndProduct(ctx context.Context, tenantID, warehouseID, productID uuid.UUID) (*inventory.InventoryItem, error) {
	args := m.Called(ctx, tenantID, warehouseID, productID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*inventory.InventoryItem), args.Error(1)
}

func (m *MockInventoryItemRepository) FindByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID, filter shared.Filter) ([]inventory.InventoryItem, error) {
	args := m.Called(ctx, tenantID, warehouseID, filter)
	return args.Get(0).([]inventory.InventoryItem), args.Error(1)
}

func (m *MockInventoryItemRepository) FindByProduct(ctx context.Context, tenantID, productID uuid.UUID, filter shared.Filter) ([]inventory.InventoryItem, error) {
	args := m.Called(ctx, tenantID, productID, filter)
	return args.Get(0).([]inventory.InventoryItem), args.Error(1)
}

func (m *MockInventoryItemRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]inventory.InventoryItem, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]inventory.InventoryItem), args.Error(1)
}

func (m *MockInventoryItemRepository) FindBelowMinimum(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]inventory.InventoryItem, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]inventory.InventoryItem), args.Error(1)
}

func (m *MockInventoryItemRepository) FindWithAvailableStock(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]inventory.InventoryItem, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]inventory.InventoryItem), args.Error(1)
}

func (m *MockInventoryItemRepository) FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]inventory.InventoryItem, error) {
	args := m.Called(ctx, tenantID, ids)
	return args.Get(0).([]inventory.InventoryItem), args.Error(1)
}

func (m *MockInventoryItemRepository) Save(ctx context.Context, item *inventory.InventoryItem) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *MockInventoryItemRepository) SaveWithLock(ctx context.Context, item *inventory.InventoryItem) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *MockInventoryItemRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockInventoryItemRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *MockInventoryItemRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockInventoryItemRepository) CountByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID, warehouseID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockInventoryItemRepository) CountByProduct(ctx context.Context, tenantID, productID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID, productID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockInventoryItemRepository) SumQuantityByProduct(ctx context.Context, tenantID, productID uuid.UUID) (decimal.Decimal, error) {
	args := m.Called(ctx, tenantID, productID)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockInventoryItemRepository) SumValueByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID) (decimal.Decimal, error) {
	args := m.Called(ctx, tenantID, warehouseID)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockInventoryItemRepository) ExistsByWarehouseAndProduct(ctx context.Context, tenantID, warehouseID, productID uuid.UUID) (bool, error) {
	args := m.Called(ctx, tenantID, warehouseID, productID)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockInventoryItemRepository) GetOrCreate(ctx context.Context, tenantID, warehouseID, productID uuid.UUID) (*inventory.InventoryItem, error) {
	args := m.Called(ctx, tenantID, warehouseID, productID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*inventory.InventoryItem), args.Error(1)
}

// MockInventoryTransactionRepository is a mock for inventory transaction repository
type MockInventoryTransactionRepository struct {
	mock.Mock
}

func (m *MockInventoryTransactionRepository) FindByID(ctx context.Context, id uuid.UUID) (*inventory.InventoryTransaction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*inventory.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) FindByInventoryItem(ctx context.Context, inventoryItemID uuid.UUID, filter shared.Filter) ([]inventory.InventoryTransaction, error) {
	args := m.Called(ctx, inventoryItemID, filter)
	return args.Get(0).([]inventory.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) FindByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID, filter shared.Filter) ([]inventory.InventoryTransaction, error) {
	args := m.Called(ctx, tenantID, warehouseID, filter)
	return args.Get(0).([]inventory.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) FindByProduct(ctx context.Context, tenantID, productID uuid.UUID, filter shared.Filter) ([]inventory.InventoryTransaction, error) {
	args := m.Called(ctx, tenantID, productID, filter)
	return args.Get(0).([]inventory.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) FindBySource(ctx context.Context, sourceType inventory.SourceType, sourceID string) ([]inventory.InventoryTransaction, error) {
	args := m.Called(ctx, sourceType, sourceID)
	return args.Get(0).([]inventory.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) FindByDateRange(ctx context.Context, tenantID uuid.UUID, start, end time.Time, filter shared.Filter) ([]inventory.InventoryTransaction, error) {
	args := m.Called(ctx, tenantID, start, end, filter)
	return args.Get(0).([]inventory.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) FindByType(ctx context.Context, tenantID uuid.UUID, txType inventory.TransactionType, filter shared.Filter) ([]inventory.InventoryTransaction, error) {
	args := m.Called(ctx, tenantID, txType, filter)
	return args.Get(0).([]inventory.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) FindForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]inventory.InventoryTransaction, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]inventory.InventoryTransaction), args.Error(1)
}

func (m *MockInventoryTransactionRepository) Create(ctx context.Context, tx *inventory.InventoryTransaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockInventoryTransactionRepository) CreateBatch(ctx context.Context, txs []*inventory.InventoryTransaction) error {
	args := m.Called(ctx, txs)
	return args.Error(0)
}

func (m *MockInventoryTransactionRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockInventoryTransactionRepository) CountByInventoryItem(ctx context.Context, inventoryItemID uuid.UUID) (int64, error) {
	args := m.Called(ctx, inventoryItemID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockInventoryTransactionRepository) SumQuantityByTypeAndDateRange(ctx context.Context, tenantID uuid.UUID, txType inventory.TransactionType, start, end time.Time) (decimal.Decimal, error) {
	args := m.Called(ctx, tenantID, txType, start, end)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

// MockWarehouseRepository is a mock for warehouse repository
type MockWarehouseRepository struct {
	mock.Mock
}

func (m *MockWarehouseRepository) FindByID(ctx context.Context, id uuid.UUID) (*partner.Warehouse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*partner.Warehouse, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*partner.Warehouse, error) {
	args := m.Called(ctx, tenantID, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) FindDefault(ctx context.Context, tenantID uuid.UUID) (*partner.Warehouse, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Warehouse, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]partner.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) FindByType(ctx context.Context, tenantID uuid.UUID, warehouseType partner.WarehouseType, filter shared.Filter) ([]partner.Warehouse, error) {
	args := m.Called(ctx, tenantID, warehouseType, filter)
	return args.Get(0).([]partner.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status partner.WarehouseStatus, filter shared.Filter) ([]partner.Warehouse, error) {
	args := m.Called(ctx, tenantID, status, filter)
	return args.Get(0).([]partner.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) FindActive(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Warehouse, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]partner.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) FindAll(ctx context.Context, filter shared.Filter) ([]partner.Warehouse, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]partner.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]partner.Warehouse, error) {
	args := m.Called(ctx, tenantID, ids)
	return args.Get(0).([]partner.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) FindByCodes(ctx context.Context, tenantID uuid.UUID, codes []string) ([]partner.Warehouse, error) {
	args := m.Called(ctx, tenantID, codes)
	return args.Get(0).([]partner.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) Save(ctx context.Context, warehouse *partner.Warehouse) error {
	args := m.Called(ctx, warehouse)
	return args.Error(0)
}

func (m *MockWarehouseRepository) SaveBatch(ctx context.Context, warehouses []*partner.Warehouse) error {
	args := m.Called(ctx, warehouses)
	return args.Error(0)
}

func (m *MockWarehouseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockWarehouseRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *MockWarehouseRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockWarehouseRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status partner.WarehouseStatus) (int64, error) {
	args := m.Called(ctx, tenantID, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockWarehouseRepository) CountByType(ctx context.Context, tenantID uuid.UUID, warehouseType partner.WarehouseType) (int64, error) {
	args := m.Called(ctx, tenantID, warehouseType)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockWarehouseRepository) ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error) {
	args := m.Called(ctx, tenantID, code)
	return args.Bool(0), args.Error(1)
}

func (m *MockWarehouseRepository) ClearDefault(ctx context.Context, tenantID uuid.UUID) error {
	args := m.Called(ctx, tenantID)
	return args.Error(0)
}

func (m *MockWarehouseRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

// MockInventoryEventPublisher is a mock for event publisher
type MockInventoryEventPublisher struct {
	mock.Mock
}

func (m *MockInventoryEventPublisher) Publish(ctx context.Context, events ...shared.DomainEvent) error {
	args := m.Called(ctx, events)
	return args.Error(0)
}

// MockInventoryProductRepository is a mock for product repository in inventory tests
type MockInventoryProductRepository struct {
	mock.Mock
}

func (m *MockInventoryProductRepository) FindByID(ctx context.Context, id uuid.UUID) (*catalog.Product, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*catalog.Product), args.Error(1)
}

func (m *MockInventoryProductRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*catalog.Product, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*catalog.Product), args.Error(1)
}

func (m *MockInventoryProductRepository) FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*catalog.Product, error) {
	args := m.Called(ctx, tenantID, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*catalog.Product), args.Error(1)
}

func (m *MockInventoryProductRepository) FindByBarcode(ctx context.Context, tenantID uuid.UUID, barcode string) (*catalog.Product, error) {
	args := m.Called(ctx, tenantID, barcode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*catalog.Product), args.Error(1)
}

func (m *MockInventoryProductRepository) FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]catalog.Product, error) {
	args := m.Called(ctx, tenantID, ids)
	return args.Get(0).([]catalog.Product), args.Error(1)
}

func (m *MockInventoryProductRepository) FindByCodes(ctx context.Context, tenantID uuid.UUID, codes []string) ([]catalog.Product, error) {
	args := m.Called(ctx, tenantID, codes)
	return args.Get(0).([]catalog.Product), args.Error(1)
}

func (m *MockInventoryProductRepository) FindAll(ctx context.Context, filter shared.Filter) ([]catalog.Product, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]catalog.Product), args.Error(1)
}

func (m *MockInventoryProductRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]catalog.Product, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]catalog.Product), args.Error(1)
}

func (m *MockInventoryProductRepository) FindByCategory(ctx context.Context, tenantID, categoryID uuid.UUID, filter shared.Filter) ([]catalog.Product, error) {
	args := m.Called(ctx, tenantID, categoryID, filter)
	return args.Get(0).([]catalog.Product), args.Error(1)
}

func (m *MockInventoryProductRepository) FindByCategories(ctx context.Context, tenantID uuid.UUID, categoryIDs []uuid.UUID, filter shared.Filter) ([]catalog.Product, error) {
	args := m.Called(ctx, tenantID, categoryIDs, filter)
	return args.Get(0).([]catalog.Product), args.Error(1)
}

func (m *MockInventoryProductRepository) FindActive(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]catalog.Product, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]catalog.Product), args.Error(1)
}

func (m *MockInventoryProductRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status catalog.ProductStatus, filter shared.Filter) ([]catalog.Product, error) {
	args := m.Called(ctx, tenantID, status, filter)
	return args.Get(0).([]catalog.Product), args.Error(1)
}

func (m *MockInventoryProductRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockInventoryProductRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockInventoryProductRepository) CountByCategory(ctx context.Context, tenantID, categoryID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID, categoryID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockInventoryProductRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status catalog.ProductStatus) (int64, error) {
	args := m.Called(ctx, tenantID, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockInventoryProductRepository) ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error) {
	args := m.Called(ctx, tenantID, code)
	return args.Bool(0), args.Error(1)
}

func (m *MockInventoryProductRepository) ExistsByBarcode(ctx context.Context, tenantID uuid.UUID, barcode string) (bool, error) {
	args := m.Called(ctx, tenantID, barcode)
	return args.Bool(0), args.Error(1)
}

func (m *MockInventoryProductRepository) Save(ctx context.Context, product *catalog.Product) error {
	args := m.Called(ctx, product)
	return args.Error(0)
}

func (m *MockInventoryProductRepository) SaveBatch(ctx context.Context, products []*catalog.Product) error {
	args := m.Called(ctx, products)
	return args.Error(0)
}

func (m *MockInventoryProductRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockInventoryProductRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

// Test helpers for inventory import
func newInventoryValidatedSession(tenantID, userID uuid.UUID) *csvimport.ImportSession {
	session := csvimport.NewImportSession(tenantID, userID, csvimport.EntityInventory, "inventory.csv", 1024)
	session.UpdateState(csvimport.StateValidating)
	session.TotalRows = 2
	session.ValidRows = 2
	session.ErrorRows = 0
	session.UpdateState(csvimport.StateValidated)
	return session
}

func createInventoryTestProduct(tenantID uuid.UUID, code string) *catalog.Product {
	product, _ := catalog.NewProduct(tenantID, code, "Test Product "+code, "PCS")
	return product
}

func createInventoryTestWarehouse(tenantID uuid.UUID, code string) *partner.Warehouse {
	warehouse, _ := partner.NewWarehouse(tenantID, code, "Test Warehouse "+code, partner.WarehouseTypePhysical)
	return warehouse
}

func createInventoryImportRow(lineNumber int, productSKU, warehouseCode, quantity, unitCost string) *csvimport.Row {
	return &csvimport.Row{
		LineNumber: lineNumber,
		Data: map[string]string{
			"product_sku":    productSKU,
			"warehouse_code": warehouseCode,
			"quantity":       quantity,
			"unit_cost":      unitCost,
		},
	}
}

func createInventoryImportRowWithBatch(lineNumber int, productSKU, warehouseCode, quantity, unitCost, batchNumber, productionDate, expiryDate string) *csvimport.Row {
	return &csvimport.Row{
		LineNumber: lineNumber,
		Data: map[string]string{
			"product_sku":     productSKU,
			"warehouse_code":  warehouseCode,
			"quantity":        quantity,
			"unit_cost":       unitCost,
			"batch_number":    batchNumber,
			"production_date": productionDate,
			"expiry_date":     expiryDate,
		},
	}
}

// Tests for GetValidationRules
func TestInventoryImportService_GetValidationRules(t *testing.T) {
	service := NewInventoryImportService(nil, nil, nil, nil, nil)
	rules := service.GetValidationRules()

	assert.NotEmpty(t, rules)

	// Find required fields
	requiredFields := []string{"product_sku", "warehouse_code", "quantity", "unit_cost"}
	for _, fieldName := range requiredFields {
		found := false
		for _, rule := range rules {
			if rule.Column == fieldName {
				found = true
				assert.True(t, rule.Required, "field %s should be required", fieldName)
				break
			}
		}
		assert.True(t, found, "field %s should be in rules", fieldName)
	}
}

// Tests for parseDate function
func TestParseDate(t *testing.T) {
	tests := []struct {
		name    string
		dateStr string
		wantErr bool
	}{
		{"ISO format", "2024-01-15", false},
		{"slash format", "2024/01/15", false},
		{"US format", "01/15/2024", false},
		{"EU format", "15-01-2024", false},
		{"invalid format", "15-Jan-2024", true},
		{"empty string", "", true},
		{"random string", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseDate(tt.dateStr)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Tests for Import - Session state validation
func TestInventoryImportService_Import_SessionNotValidated(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	service := NewInventoryImportService(nil, nil, nil, nil, nil)

	session := csvimport.NewImportSession(tenantID, userID, csvimport.EntityInventory, "inventory.csv", 1024)
	// Session is in "created" state, not "validated"

	_, err := service.Import(ctx, tenantID, userID, session, nil, ConflictModeSkip)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validated state")
}

func TestInventoryImportService_Import_SessionHasErrors(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	service := NewInventoryImportService(nil, nil, nil, nil, nil)

	session := csvimport.NewImportSession(tenantID, userID, csvimport.EntityInventory, "inventory.csv", 1024)
	session.UpdateState(csvimport.StateValidating)
	session.ErrorRows = 1 // Has errors
	session.UpdateState(csvimport.StateValidated)

	_, err := service.Import(ctx, tenantID, userID, session, nil, ConflictModeSkip)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation errors")
}

// Tests for Import - Context cancellation
func TestInventoryImportService_Import_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	tenantID := uuid.New()
	userID := uuid.New()

	service := NewInventoryImportService(nil, nil, nil, nil, nil)
	session := newInventoryValidatedSession(tenantID, userID)

	rows := []*csvimport.Row{
		createInventoryImportRow(2, "SKU001", "WH001", "100", "10.00"),
	}

	_, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

// Tests for Import - Product not found
func TestInventoryImportService_Import_ProductNotFound(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	productRepo := new(MockInventoryProductRepository)
	warehouseRepo := new(MockWarehouseRepository)
	inventoryRepo := new(MockInventoryItemRepository)
	transactionRepo := new(MockInventoryTransactionRepository)

	productRepo.On("FindByCode", ctx, tenantID, "SKU001").Return(nil, shared.ErrNotFound)

	service := NewInventoryImportService(inventoryRepo, transactionRepo, productRepo, warehouseRepo, nil)
	session := newInventoryValidatedSession(tenantID, userID)

	rows := []*csvimport.Row{
		createInventoryImportRow(2, "SKU001", "WH001", "100", "10.00"),
	}

	result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)
	require.NoError(t, err)
	assert.Equal(t, 1, result.ErrorRows)
	assert.Equal(t, 0, result.ImportedRows)
	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0].Message, "not found")

	productRepo.AssertExpectations(t)
}

// Tests for Import - Warehouse not found
func TestInventoryImportService_Import_WarehouseNotFound(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	productRepo := new(MockInventoryProductRepository)
	warehouseRepo := new(MockWarehouseRepository)
	inventoryRepo := new(MockInventoryItemRepository)
	transactionRepo := new(MockInventoryTransactionRepository)

	product := createInventoryTestProduct(tenantID, "SKU001")
	productRepo.On("FindByCode", ctx, tenantID, "SKU001").Return(product, nil)
	warehouseRepo.On("FindByCode", ctx, tenantID, "WH001").Return(nil, shared.ErrNotFound)

	service := NewInventoryImportService(inventoryRepo, transactionRepo, productRepo, warehouseRepo, nil)
	session := newInventoryValidatedSession(tenantID, userID)

	rows := []*csvimport.Row{
		createInventoryImportRow(2, "SKU001", "WH001", "100", "10.00"),
	}

	result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)
	require.NoError(t, err)
	assert.Equal(t, 1, result.ErrorRows)
	assert.Equal(t, 0, result.ImportedRows)
	assert.Contains(t, result.Errors[0].Message, "warehouse")

	productRepo.AssertExpectations(t)
	warehouseRepo.AssertExpectations(t)
}

// Tests for Import - Successful new inventory import
func TestInventoryImportService_Import_Success(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	productRepo := new(MockInventoryProductRepository)
	warehouseRepo := new(MockWarehouseRepository)
	inventoryRepo := new(MockInventoryItemRepository)
	transactionRepo := new(MockInventoryTransactionRepository)
	eventBus := new(MockInventoryEventPublisher)

	product := createInventoryTestProduct(tenantID, "SKU001")
	warehouse := createInventoryTestWarehouse(tenantID, "WH001")

	productRepo.On("FindByCode", ctx, tenantID, "SKU001").Return(product, nil)
	warehouseRepo.On("FindByCode", ctx, tenantID, "WH001").Return(warehouse, nil)
	inventoryRepo.On("FindByWarehouseAndProduct", ctx, tenantID, warehouse.ID, product.ID).Return(nil, shared.ErrNotFound)
	inventoryRepo.On("Save", ctx, mock.AnythingOfType("*inventory.InventoryItem")).Return(nil)
	transactionRepo.On("Create", ctx, mock.AnythingOfType("*inventory.InventoryTransaction")).Return(nil)
	eventBus.On("Publish", ctx, mock.Anything).Return(nil)

	service := NewInventoryImportService(inventoryRepo, transactionRepo, productRepo, warehouseRepo, eventBus)
	session := newInventoryValidatedSession(tenantID, userID)

	rows := []*csvimport.Row{
		createInventoryImportRow(2, "SKU001", "WH001", "100", "10.00"),
	}

	result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)
	require.NoError(t, err)
	assert.Equal(t, 0, result.ErrorRows)
	assert.Equal(t, 1, result.ImportedRows)

	productRepo.AssertExpectations(t)
	warehouseRepo.AssertExpectations(t)
	inventoryRepo.AssertExpectations(t)
	transactionRepo.AssertExpectations(t)
}

// Tests for Import - ConflictMode Skip
func TestInventoryImportService_Import_ConflictModeSkip(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	productRepo := new(MockInventoryProductRepository)
	warehouseRepo := new(MockWarehouseRepository)
	inventoryRepo := new(MockInventoryItemRepository)
	transactionRepo := new(MockInventoryTransactionRepository)

	product := createInventoryTestProduct(tenantID, "SKU001")
	warehouse := createInventoryTestWarehouse(tenantID, "WH001")

	// Existing inventory item
	existingItem, _ := inventory.NewInventoryItem(tenantID, warehouse.ID, product.ID)

	productRepo.On("FindByCode", ctx, tenantID, "SKU001").Return(product, nil)
	warehouseRepo.On("FindByCode", ctx, tenantID, "WH001").Return(warehouse, nil)
	inventoryRepo.On("FindByWarehouseAndProduct", ctx, tenantID, warehouse.ID, product.ID).Return(existingItem, nil)

	service := NewInventoryImportService(inventoryRepo, transactionRepo, productRepo, warehouseRepo, nil)
	session := newInventoryValidatedSession(tenantID, userID)

	rows := []*csvimport.Row{
		createInventoryImportRow(2, "SKU001", "WH001", "100", "10.00"),
	}

	result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)
	require.NoError(t, err)
	assert.Equal(t, 0, result.ErrorRows)
	assert.Equal(t, 0, result.ImportedRows)
	assert.Equal(t, 1, result.SkippedRows)

	productRepo.AssertExpectations(t)
	warehouseRepo.AssertExpectations(t)
	inventoryRepo.AssertExpectations(t)
}

// Tests for Import - ConflictMode Fail
func TestInventoryImportService_Import_ConflictModeFail(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	productRepo := new(MockInventoryProductRepository)
	warehouseRepo := new(MockWarehouseRepository)
	inventoryRepo := new(MockInventoryItemRepository)
	transactionRepo := new(MockInventoryTransactionRepository)

	product := createInventoryTestProduct(tenantID, "SKU001")
	warehouse := createInventoryTestWarehouse(tenantID, "WH001")

	// Existing inventory item
	existingItem, _ := inventory.NewInventoryItem(tenantID, warehouse.ID, product.ID)

	productRepo.On("FindByCode", ctx, tenantID, "SKU001").Return(product, nil)
	warehouseRepo.On("FindByCode", ctx, tenantID, "WH001").Return(warehouse, nil)
	inventoryRepo.On("FindByWarehouseAndProduct", ctx, tenantID, warehouse.ID, product.ID).Return(existingItem, nil)

	service := NewInventoryImportService(inventoryRepo, transactionRepo, productRepo, warehouseRepo, nil)
	session := newInventoryValidatedSession(tenantID, userID)

	rows := []*csvimport.Row{
		createInventoryImportRow(2, "SKU001", "WH001", "100", "10.00"),
	}

	result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeFail)
	require.NoError(t, err)
	assert.Equal(t, 1, result.ErrorRows)
	assert.Equal(t, 0, result.ImportedRows)
	assert.Contains(t, result.Errors[0].Message, "already exists")

	productRepo.AssertExpectations(t)
	warehouseRepo.AssertExpectations(t)
	inventoryRepo.AssertExpectations(t)
}

// Tests for Import - ConflictMode Update
func TestInventoryImportService_Import_ConflictModeUpdate(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	productRepo := new(MockInventoryProductRepository)
	warehouseRepo := new(MockWarehouseRepository)
	inventoryRepo := new(MockInventoryItemRepository)
	transactionRepo := new(MockInventoryTransactionRepository)
	eventBus := new(MockInventoryEventPublisher)

	product := createInventoryTestProduct(tenantID, "SKU001")
	warehouse := createInventoryTestWarehouse(tenantID, "WH001")

	// Existing inventory item
	existingItem, _ := inventory.NewInventoryItem(tenantID, warehouse.ID, product.ID)

	productRepo.On("FindByCode", ctx, tenantID, "SKU001").Return(product, nil)
	warehouseRepo.On("FindByCode", ctx, tenantID, "WH001").Return(warehouse, nil)
	inventoryRepo.On("FindByWarehouseAndProduct", ctx, tenantID, warehouse.ID, product.ID).Return(existingItem, nil)
	inventoryRepo.On("Save", ctx, mock.AnythingOfType("*inventory.InventoryItem")).Return(nil)
	transactionRepo.On("Create", ctx, mock.AnythingOfType("*inventory.InventoryTransaction")).Return(nil)
	eventBus.On("Publish", ctx, mock.Anything).Return(nil)

	service := NewInventoryImportService(inventoryRepo, transactionRepo, productRepo, warehouseRepo, eventBus)
	session := newInventoryValidatedSession(tenantID, userID)

	rows := []*csvimport.Row{
		createInventoryImportRow(2, "SKU001", "WH001", "100", "10.00"),
	}

	result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeUpdate)
	require.NoError(t, err)
	assert.Equal(t, 0, result.ErrorRows)
	assert.Equal(t, 1, result.UpdatedRows)

	productRepo.AssertExpectations(t)
	warehouseRepo.AssertExpectations(t)
	inventoryRepo.AssertExpectations(t)
	transactionRepo.AssertExpectations(t)
}

// Tests for Import - With Batch info
func TestInventoryImportService_Import_WithBatch(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	productRepo := new(MockInventoryProductRepository)
	warehouseRepo := new(MockWarehouseRepository)
	inventoryRepo := new(MockInventoryItemRepository)
	transactionRepo := new(MockInventoryTransactionRepository)
	eventBus := new(MockInventoryEventPublisher)

	product := createInventoryTestProduct(tenantID, "SKU001")
	warehouse := createInventoryTestWarehouse(tenantID, "WH001")

	productRepo.On("FindByCode", ctx, tenantID, "SKU001").Return(product, nil)
	warehouseRepo.On("FindByCode", ctx, tenantID, "WH001").Return(warehouse, nil)
	inventoryRepo.On("FindByWarehouseAndProduct", ctx, tenantID, warehouse.ID, product.ID).Return(nil, shared.ErrNotFound)
	inventoryRepo.On("Save", ctx, mock.MatchedBy(func(item *inventory.InventoryItem) bool {
		// Verify batch was created
		return len(item.Batches) == 1 && item.Batches[0].BatchNumber == "BATCH001"
	})).Return(nil)
	transactionRepo.On("Create", ctx, mock.AnythingOfType("*inventory.InventoryTransaction")).Return(nil)
	eventBus.On("Publish", ctx, mock.Anything).Return(nil)

	service := NewInventoryImportService(inventoryRepo, transactionRepo, productRepo, warehouseRepo, eventBus)
	session := newInventoryValidatedSession(tenantID, userID)

	rows := []*csvimport.Row{
		createInventoryImportRowWithBatch(2, "SKU001", "WH001", "100", "10.00", "BATCH001", "2024-01-01", "2025-12-31"),
	}

	result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)
	require.NoError(t, err)
	assert.Equal(t, 0, result.ErrorRows)
	assert.Equal(t, 1, result.ImportedRows)

	productRepo.AssertExpectations(t)
	warehouseRepo.AssertExpectations(t)
	inventoryRepo.AssertExpectations(t)
	transactionRepo.AssertExpectations(t)
}

// Tests for Import - Invalid expiry date (before production date)
func TestInventoryImportService_Import_InvalidExpiryDate(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	productRepo := new(MockInventoryProductRepository)
	warehouseRepo := new(MockWarehouseRepository)
	inventoryRepo := new(MockInventoryItemRepository)
	transactionRepo := new(MockInventoryTransactionRepository)

	product := createInventoryTestProduct(tenantID, "SKU001")
	warehouse := createInventoryTestWarehouse(tenantID, "WH001")

	productRepo.On("FindByCode", ctx, tenantID, "SKU001").Return(product, nil)
	warehouseRepo.On("FindByCode", ctx, tenantID, "WH001").Return(warehouse, nil)

	service := NewInventoryImportService(inventoryRepo, transactionRepo, productRepo, warehouseRepo, nil)
	session := newInventoryValidatedSession(tenantID, userID)

	// expiry_date before production_date
	rows := []*csvimport.Row{
		createInventoryImportRowWithBatch(2, "SKU001", "WH001", "100", "10.00", "BATCH001", "2025-01-01", "2024-01-01"),
	}

	result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)
	require.NoError(t, err)
	assert.Equal(t, 1, result.ErrorRows)
	assert.Contains(t, result.Errors[0].Message, "expiry_date must be after production_date")

	productRepo.AssertExpectations(t)
	warehouseRepo.AssertExpectations(t)
}

// Tests for ValidateWithWarnings
func TestInventoryImportService_ValidateWithWarnings(t *testing.T) {
	service := NewInventoryImportService(nil, nil, nil, nil, nil)

	t.Run("high quantity warning", func(t *testing.T) {
		row := &csvimport.Row{
			LineNumber: 2,
			Data: map[string]string{
				"quantity": "2000000",
			},
		}
		warnings := service.ValidateWithWarnings(row)
		assert.NotEmpty(t, warnings)
		assert.Contains(t, warnings[0], "unusually high")
	})

	t.Run("high unit cost warning", func(t *testing.T) {
		row := &csvimport.Row{
			LineNumber: 2,
			Data: map[string]string{
				"unit_cost": "5000000",
			},
		}
		warnings := service.ValidateWithWarnings(row)
		assert.NotEmpty(t, warnings)
		assert.Contains(t, warnings[0], "unusually high")
	})

	t.Run("past expiry date warning", func(t *testing.T) {
		row := &csvimport.Row{
			LineNumber: 2,
			Data: map[string]string{
				"expiry_date": "2020-01-01",
			},
		}
		warnings := service.ValidateWithWarnings(row)
		assert.NotEmpty(t, warnings)
		assert.Contains(t, warnings[0], "past")
	})

	t.Run("batch without expiry warning", func(t *testing.T) {
		row := &csvimport.Row{
			LineNumber: 2,
			Data: map[string]string{
				"batch_number": "BATCH001",
			},
		}
		warnings := service.ValidateWithWarnings(row)
		assert.NotEmpty(t, warnings)
		assert.Contains(t, warnings[0], "without expiry date")
	})
}

// Tests for LookupReference
func TestInventoryImportService_LookupReference(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	productRepo := new(MockInventoryProductRepository)
	warehouseRepo := new(MockWarehouseRepository)

	t.Run("product found", func(t *testing.T) {
		product := createInventoryTestProduct(tenantID, "SKU001")
		productRepo.On("FindByCode", ctx, tenantID, "SKU001").Return(product, nil).Once()

		service := NewInventoryImportService(nil, nil, productRepo, warehouseRepo, nil)

		found, err := service.LookupReference(ctx, tenantID, "products", "SKU001")
		require.NoError(t, err)
		assert.True(t, found)
		productRepo.AssertExpectations(t)
	})

	t.Run("product not found", func(t *testing.T) {
		productRepo.On("FindByCode", ctx, tenantID, "INVALID").Return(nil, shared.ErrNotFound).Once()

		service := NewInventoryImportService(nil, nil, productRepo, warehouseRepo, nil)

		found, err := service.LookupReference(ctx, tenantID, "products", "INVALID")
		require.NoError(t, err)
		assert.False(t, found)
		productRepo.AssertExpectations(t)
	})

	t.Run("warehouse found", func(t *testing.T) {
		warehouse := createInventoryTestWarehouse(tenantID, "WH001")
		warehouseRepo.On("FindByCode", ctx, tenantID, "WH001").Return(warehouse, nil).Once()

		service := NewInventoryImportService(nil, nil, productRepo, warehouseRepo, nil)

		found, err := service.LookupReference(ctx, tenantID, "warehouses", "WH001")
		require.NoError(t, err)
		assert.True(t, found)
		warehouseRepo.AssertExpectations(t)
	})

	t.Run("warehouse not found", func(t *testing.T) {
		warehouseRepo.On("FindByCode", ctx, tenantID, "INVALID").Return(nil, shared.ErrNotFound).Once()

		service := NewInventoryImportService(nil, nil, productRepo, warehouseRepo, nil)

		found, err := service.LookupReference(ctx, tenantID, "warehouses", "INVALID")
		require.NoError(t, err)
		assert.False(t, found)
		warehouseRepo.AssertExpectations(t)
	})

	t.Run("empty value returns false", func(t *testing.T) {
		service := NewInventoryImportService(nil, nil, productRepo, warehouseRepo, nil)

		found, err := service.LookupReference(ctx, tenantID, "products", "")
		require.NoError(t, err)
		assert.False(t, found)
	})

	t.Run("unknown ref type returns false", func(t *testing.T) {
		service := NewInventoryImportService(nil, nil, productRepo, warehouseRepo, nil)

		found, err := service.LookupReference(ctx, tenantID, "unknown", "value")
		require.NoError(t, err)
		assert.False(t, found)
	})
}

// Tests for Import - Invalid quantity format
func TestInventoryImportService_Import_InvalidQuantity(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	service := NewInventoryImportService(nil, nil, nil, nil, nil)
	session := newInventoryValidatedSession(tenantID, userID)

	rows := []*csvimport.Row{
		{
			LineNumber: 2,
			Data: map[string]string{
				"product_sku":    "SKU001",
				"warehouse_code": "WH001",
				"quantity":       "invalid",
				"unit_cost":      "10.00",
			},
		},
	}

	result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)
	require.NoError(t, err)
	assert.Equal(t, 1, result.ErrorRows)
	assert.Contains(t, result.Errors[0].Message, "invalid decimal")
}

// Tests for Import - Invalid unit cost format
func TestInventoryImportService_Import_InvalidUnitCost(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	service := NewInventoryImportService(nil, nil, nil, nil, nil)
	session := newInventoryValidatedSession(tenantID, userID)

	rows := []*csvimport.Row{
		{
			LineNumber: 2,
			Data: map[string]string{
				"product_sku":    "SKU001",
				"warehouse_code": "WH001",
				"quantity":       "100",
				"unit_cost":      "invalid",
			},
		},
	}

	result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)
	require.NoError(t, err)
	assert.Equal(t, 1, result.ErrorRows)
	assert.Contains(t, result.Errors[0].Message, "invalid decimal")
}

// Tests for Import - Repository error
func TestInventoryImportService_Import_RepositoryError(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	productRepo := new(MockInventoryProductRepository)
	warehouseRepo := new(MockWarehouseRepository)
	inventoryRepo := new(MockInventoryItemRepository)
	transactionRepo := new(MockInventoryTransactionRepository)

	// Simulate repository error
	productRepo.On("FindByCode", ctx, tenantID, "SKU001").Return(nil, errors.New("database error"))

	service := NewInventoryImportService(inventoryRepo, transactionRepo, productRepo, warehouseRepo, nil)
	session := newInventoryValidatedSession(tenantID, userID)

	rows := []*csvimport.Row{
		createInventoryImportRow(2, "SKU001", "WH001", "100", "10.00"),
	}

	_, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database error")

	productRepo.AssertExpectations(t)
}
