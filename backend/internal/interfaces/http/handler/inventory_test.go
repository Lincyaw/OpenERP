package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	inventoryapp "github.com/erp/backend/internal/application/inventory"
	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations for inventory repositories

type mockInventoryItemRepository struct {
	items      map[uuid.UUID]*inventory.InventoryItem
	returnErr  error
	findByWPID *inventory.InventoryItem
}

func newMockInventoryItemRepository() *mockInventoryItemRepository {
	return &mockInventoryItemRepository{
		items: make(map[uuid.UUID]*inventory.InventoryItem),
	}
}

func (m *mockInventoryItemRepository) FindByID(ctx context.Context, id uuid.UUID) (*inventory.InventoryItem, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	if item, ok := m.items[id]; ok {
		return item, nil
	}
	return nil, shared.ErrNotFound
}

func (m *mockInventoryItemRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*inventory.InventoryItem, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	if item, ok := m.items[id]; ok && item.TenantID == tenantID {
		return item, nil
	}
	return nil, shared.ErrNotFound
}

func (m *mockInventoryItemRepository) FindByWarehouseAndProduct(ctx context.Context, tenantID, warehouseID, productID uuid.UUID) (*inventory.InventoryItem, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	if m.findByWPID != nil {
		return m.findByWPID, nil
	}
	for _, item := range m.items {
		if item.TenantID == tenantID && item.WarehouseID == warehouseID && item.ProductID == productID {
			return item, nil
		}
	}
	return nil, shared.ErrNotFound
}

func (m *mockInventoryItemRepository) FindByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID, filter shared.Filter) ([]inventory.InventoryItem, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	var result []inventory.InventoryItem
	for _, item := range m.items {
		if item.TenantID == tenantID && item.WarehouseID == warehouseID {
			result = append(result, *item)
		}
	}
	return result, nil
}

func (m *mockInventoryItemRepository) FindByProduct(ctx context.Context, tenantID, productID uuid.UUID, filter shared.Filter) ([]inventory.InventoryItem, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	var result []inventory.InventoryItem
	for _, item := range m.items {
		if item.TenantID == tenantID && item.ProductID == productID {
			result = append(result, *item)
		}
	}
	return result, nil
}

func (m *mockInventoryItemRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]inventory.InventoryItem, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	var result []inventory.InventoryItem
	for _, item := range m.items {
		if item.TenantID == tenantID {
			result = append(result, *item)
		}
	}
	return result, nil
}

func (m *mockInventoryItemRepository) FindBelowMinimum(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]inventory.InventoryItem, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	var result []inventory.InventoryItem
	for _, item := range m.items {
		if item.TenantID == tenantID && item.IsBelowMinimum() {
			result = append(result, *item)
		}
	}
	return result, nil
}

func (m *mockInventoryItemRepository) FindWithAvailableStock(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]inventory.InventoryItem, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	var result []inventory.InventoryItem
	for _, item := range m.items {
		if item.TenantID == tenantID && item.AvailableQuantity.GreaterThan(decimal.Zero) {
			result = append(result, *item)
		}
	}
	return result, nil
}

func (m *mockInventoryItemRepository) FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]inventory.InventoryItem, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	var result []inventory.InventoryItem
	for _, id := range ids {
		if item, ok := m.items[id]; ok && item.TenantID == tenantID {
			result = append(result, *item)
		}
	}
	return result, nil
}

func (m *mockInventoryItemRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	if m.returnErr != nil {
		return 0, m.returnErr
	}
	var count int64
	for _, item := range m.items {
		if item.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

func (m *mockInventoryItemRepository) CountByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID) (int64, error) {
	if m.returnErr != nil {
		return 0, m.returnErr
	}
	var count int64
	for _, item := range m.items {
		if item.TenantID == tenantID && item.WarehouseID == warehouseID {
			count++
		}
	}
	return count, nil
}

func (m *mockInventoryItemRepository) CountByProduct(ctx context.Context, tenantID, productID uuid.UUID) (int64, error) {
	if m.returnErr != nil {
		return 0, m.returnErr
	}
	var count int64
	for _, item := range m.items {
		if item.TenantID == tenantID && item.ProductID == productID {
			count++
		}
	}
	return count, nil
}

func (m *mockInventoryItemRepository) GetOrCreate(ctx context.Context, tenantID, warehouseID, productID uuid.UUID) (*inventory.InventoryItem, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	for _, item := range m.items {
		if item.TenantID == tenantID && item.WarehouseID == warehouseID && item.ProductID == productID {
			return item, nil
		}
	}
	// Create new
	item, err := inventory.NewInventoryItem(tenantID, warehouseID, productID)
	if err != nil {
		return nil, err
	}
	m.items[item.ID] = item
	return item, nil
}

func (m *mockInventoryItemRepository) Save(ctx context.Context, item *inventory.InventoryItem) error {
	if m.returnErr != nil {
		return m.returnErr
	}
	m.items[item.ID] = item
	return nil
}

func (m *mockInventoryItemRepository) SaveWithLock(ctx context.Context, item *inventory.InventoryItem) error {
	return m.Save(ctx, item)
}

func (m *mockInventoryItemRepository) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.items, id)
	return nil
}

func (m *mockInventoryItemRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	if item, ok := m.items[id]; ok && item.TenantID == tenantID {
		delete(m.items, id)
	}
	return nil
}

func (m *mockInventoryItemRepository) SumQuantityByProduct(ctx context.Context, tenantID, productID uuid.UUID) (decimal.Decimal, error) {
	if m.returnErr != nil {
		return decimal.Zero, m.returnErr
	}
	var sum decimal.Decimal
	for _, item := range m.items {
		if item.TenantID == tenantID && item.ProductID == productID {
			sum = sum.Add(item.AvailableQuantity)
		}
	}
	return sum, nil
}

func (m *mockInventoryItemRepository) SumValueByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID) (decimal.Decimal, error) {
	if m.returnErr != nil {
		return decimal.Zero, m.returnErr
	}
	return decimal.NewFromInt(1000), nil
}

func (m *mockInventoryItemRepository) ExistsByWarehouseAndProduct(ctx context.Context, tenantID, warehouseID, productID uuid.UUID) (bool, error) {
	if m.returnErr != nil {
		return false, m.returnErr
	}
	for _, item := range m.items {
		if item.TenantID == tenantID && item.WarehouseID == warehouseID && item.ProductID == productID {
			return true, nil
		}
	}
	return false, nil
}

type mockStockBatchRepository struct {
	batches   map[uuid.UUID]*inventory.StockBatch
	returnErr error
}

func newMockStockBatchRepository() *mockStockBatchRepository {
	return &mockStockBatchRepository{
		batches: make(map[uuid.UUID]*inventory.StockBatch),
	}
}

func (m *mockStockBatchRepository) FindByID(ctx context.Context, id uuid.UUID) (*inventory.StockBatch, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	if batch, ok := m.batches[id]; ok {
		return batch, nil
	}
	return nil, shared.ErrNotFound
}

func (m *mockStockBatchRepository) FindByInventoryItem(ctx context.Context, inventoryItemID uuid.UUID, filter shared.Filter) ([]inventory.StockBatch, error) {
	return nil, nil
}

func (m *mockStockBatchRepository) FindAvailable(ctx context.Context, inventoryItemID uuid.UUID) ([]inventory.StockBatch, error) {
	return nil, nil
}

func (m *mockStockBatchRepository) FindExpiringSoon(ctx context.Context, tenantID uuid.UUID, withinDays int, filter shared.Filter) ([]inventory.StockBatch, error) {
	return nil, nil
}

func (m *mockStockBatchRepository) FindExpired(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]inventory.StockBatch, error) {
	return nil, nil
}

func (m *mockStockBatchRepository) FindByBatchNumber(ctx context.Context, inventoryItemID uuid.UUID, batchNumber string) (*inventory.StockBatch, error) {
	return nil, shared.ErrNotFound
}

func (m *mockStockBatchRepository) Save(ctx context.Context, batch *inventory.StockBatch) error {
	if m.returnErr != nil {
		return m.returnErr
	}
	m.batches[batch.ID] = batch
	return nil
}

func (m *mockStockBatchRepository) SaveBatch(ctx context.Context, batches []inventory.StockBatch) error {
	if m.returnErr != nil {
		return m.returnErr
	}
	for i := range batches {
		m.batches[batches[i].ID] = &batches[i]
	}
	return nil
}

func (m *mockStockBatchRepository) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.batches, id)
	return nil
}

func (m *mockStockBatchRepository) CountByInventoryItem(ctx context.Context, inventoryItemID uuid.UUID) (int64, error) {
	if m.returnErr != nil {
		return 0, m.returnErr
	}
	var count int64
	for _, batch := range m.batches {
		if batch.InventoryItemID == inventoryItemID {
			count++
		}
	}
	return count, nil
}

type mockStockLockRepository struct {
	locks     map[uuid.UUID]*inventory.StockLock
	returnErr error
}

func newMockStockLockRepository() *mockStockLockRepository {
	return &mockStockLockRepository{
		locks: make(map[uuid.UUID]*inventory.StockLock),
	}
}

func (m *mockStockLockRepository) FindByID(ctx context.Context, id uuid.UUID) (*inventory.StockLock, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	if lock, ok := m.locks[id]; ok {
		return lock, nil
	}
	return nil, shared.ErrNotFound
}

func (m *mockStockLockRepository) FindByInventoryItem(ctx context.Context, inventoryItemID uuid.UUID) ([]inventory.StockLock, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	var result []inventory.StockLock
	for _, lock := range m.locks {
		if lock.InventoryItemID == inventoryItemID {
			result = append(result, *lock)
		}
	}
	return result, nil
}

func (m *mockStockLockRepository) FindActive(ctx context.Context, inventoryItemID uuid.UUID) ([]inventory.StockLock, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	var result []inventory.StockLock
	for _, lock := range m.locks {
		if lock.InventoryItemID == inventoryItemID && lock.IsActive() {
			result = append(result, *lock)
		}
	}
	return result, nil
}

func (m *mockStockLockRepository) FindExpired(ctx context.Context) ([]inventory.StockLock, error) {
	return nil, nil
}

func (m *mockStockLockRepository) FindBySource(ctx context.Context, sourceType, sourceID string) ([]inventory.StockLock, error) {
	return nil, nil
}

func (m *mockStockLockRepository) Save(ctx context.Context, lock *inventory.StockLock) error {
	if m.returnErr != nil {
		return m.returnErr
	}
	m.locks[lock.ID] = lock
	return nil
}

func (m *mockStockLockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.locks, id)
	return nil
}

func (m *mockStockLockRepository) ReleaseExpired(ctx context.Context) (int, error) {
	return 0, nil
}

type mockInventoryTransactionRepository struct {
	txs       map[uuid.UUID]*inventory.InventoryTransaction
	returnErr error
}

func newMockInventoryTransactionRepository() *mockInventoryTransactionRepository {
	return &mockInventoryTransactionRepository{
		txs: make(map[uuid.UUID]*inventory.InventoryTransaction),
	}
}

func (m *mockInventoryTransactionRepository) FindByID(ctx context.Context, id uuid.UUID) (*inventory.InventoryTransaction, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	if tx, ok := m.txs[id]; ok {
		return tx, nil
	}
	return nil, shared.ErrNotFound
}

func (m *mockInventoryTransactionRepository) FindByInventoryItem(ctx context.Context, inventoryItemID uuid.UUID, filter shared.Filter) ([]inventory.InventoryTransaction, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	var result []inventory.InventoryTransaction
	for _, tx := range m.txs {
		if tx.InventoryItemID == inventoryItemID {
			result = append(result, *tx)
		}
	}
	return result, nil
}

func (m *mockInventoryTransactionRepository) FindByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID, filter shared.Filter) ([]inventory.InventoryTransaction, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	var result []inventory.InventoryTransaction
	for _, tx := range m.txs {
		if tx.TenantID == tenantID && tx.WarehouseID == warehouseID {
			result = append(result, *tx)
		}
	}
	return result, nil
}

func (m *mockInventoryTransactionRepository) FindByProduct(ctx context.Context, tenantID, productID uuid.UUID, filter shared.Filter) ([]inventory.InventoryTransaction, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	var result []inventory.InventoryTransaction
	for _, tx := range m.txs {
		if tx.TenantID == tenantID && tx.ProductID == productID {
			result = append(result, *tx)
		}
	}
	return result, nil
}

func (m *mockInventoryTransactionRepository) FindBySource(ctx context.Context, sourceType inventory.SourceType, sourceID string) ([]inventory.InventoryTransaction, error) {
	return nil, nil
}

func (m *mockInventoryTransactionRepository) FindByDateRange(ctx context.Context, tenantID uuid.UUID, start, end time.Time, filter shared.Filter) ([]inventory.InventoryTransaction, error) {
	return nil, nil
}

func (m *mockInventoryTransactionRepository) FindByType(ctx context.Context, tenantID uuid.UUID, txType inventory.TransactionType, filter shared.Filter) ([]inventory.InventoryTransaction, error) {
	return nil, nil
}

func (m *mockInventoryTransactionRepository) FindForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]inventory.InventoryTransaction, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	var result []inventory.InventoryTransaction
	for _, tx := range m.txs {
		if tx.TenantID == tenantID {
			result = append(result, *tx)
		}
	}
	return result, nil
}

func (m *mockInventoryTransactionRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	if m.returnErr != nil {
		return 0, m.returnErr
	}
	var count int64
	for _, tx := range m.txs {
		if tx.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

func (m *mockInventoryTransactionRepository) CountByInventoryItem(ctx context.Context, inventoryItemID uuid.UUID) (int64, error) {
	if m.returnErr != nil {
		return 0, m.returnErr
	}
	var count int64
	for _, tx := range m.txs {
		if tx.InventoryItemID == inventoryItemID {
			count++
		}
	}
	return count, nil
}

func (m *mockInventoryTransactionRepository) SumQuantityByTypeAndDateRange(ctx context.Context, tenantID uuid.UUID, txType inventory.TransactionType, start, end time.Time) (decimal.Decimal, error) {
	return decimal.Zero, nil
}

func (m *mockInventoryTransactionRepository) Create(ctx context.Context, tx *inventory.InventoryTransaction) error {
	if m.returnErr != nil {
		return m.returnErr
	}
	m.txs[tx.ID] = tx
	return nil
}

func (m *mockInventoryTransactionRepository) CreateBatch(ctx context.Context, txs []*inventory.InventoryTransaction) error {
	if m.returnErr != nil {
		return m.returnErr
	}
	for _, tx := range txs {
		m.txs[tx.ID] = tx
	}
	return nil
}

// Test helper functions

func setupInventoryTestHandler() (*InventoryHandler, *mockInventoryItemRepository, *mockStockLockRepository, *mockInventoryTransactionRepository) {
	gin.SetMode(gin.TestMode)

	invRepo := newMockInventoryItemRepository()
	batchRepo := newMockStockBatchRepository()
	lockRepo := newMockStockLockRepository()
	txRepo := newMockInventoryTransactionRepository()

	service := inventoryapp.NewInventoryService(invRepo, batchRepo, lockRepo, txRepo)
	handler := NewInventoryHandler(service)

	return handler, invRepo, lockRepo, txRepo
}

func createTestInventoryItem(tenantID, warehouseID, productID uuid.UUID) *inventory.InventoryItem {
	item, _ := inventory.NewInventoryItem(tenantID, warehouseID, productID)
	item.AvailableQuantity = decimal.NewFromInt(100)
	item.UnitCost = decimal.NewFromFloat(15.50)
	return item
}

// Tests

func TestNewInventoryHandler(t *testing.T) {
	handler, _, _, _ := setupInventoryTestHandler()
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.inventoryService)
}

func TestInventoryHandler_GetByID_Success(t *testing.T) {
	handler, invRepo, _, _ := setupInventoryTestHandler()

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	itemID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	item := createTestInventoryItem(tenantID, warehouseID, productID)
	item.ID = itemID
	invRepo.items[itemID] = item

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/inventory/items/"+itemID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: itemID.String()}}
	c.Request.Header.Set("X-Tenant-ID", tenantID.String())

	handler.GetByID(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestInventoryHandler_GetByID_NotFound(t *testing.T) {
	handler, _, _, _ := setupInventoryTestHandler()

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	itemID := uuid.New()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/inventory/items/"+itemID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: itemID.String()}}
	c.Request.Header.Set("X-Tenant-ID", tenantID.String())

	handler.GetByID(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestInventoryHandler_GetByID_InvalidID(t *testing.T) {
	handler, _, _, _ := setupInventoryTestHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/inventory/items/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	handler.GetByID(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInventoryHandler_List_Success(t *testing.T) {
	handler, invRepo, _, _ := setupInventoryTestHandler()

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Add some test items
	for i := 0; i < 3; i++ {
		item := createTestInventoryItem(tenantID, uuid.New(), uuid.New())
		invRepo.items[item.ID] = item
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/inventory/items?page=1&page_size=20", nil)
	c.Request.Header.Set("X-Tenant-ID", tenantID.String())

	handler.List(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Meta)
}

func TestInventoryHandler_CheckAvailability_Success(t *testing.T) {
	handler, invRepo, _, _ := setupInventoryTestHandler()

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	warehouseID := uuid.New()
	productID := uuid.New()

	item := createTestInventoryItem(tenantID, warehouseID, productID)
	invRepo.items[item.ID] = item
	invRepo.findByWPID = item

	reqBody := CheckAvailabilityRequest{
		WarehouseID: warehouseID.String(),
		ProductID:   productID.String(),
		Quantity:    50.0,
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/inventory/availability/check", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Tenant-ID", tenantID.String())

	handler.CheckAvailability(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	data := resp.Data.(map[string]interface{})
	assert.True(t, data["available"].(bool))
	assert.Equal(t, float64(100), data["available_quantity"])
}

func TestInventoryHandler_IncreaseStock_Success(t *testing.T) {
	handler, invRepo, _, _ := setupInventoryTestHandler()

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	warehouseID := uuid.New()
	productID := uuid.New()

	// Pre-create an item
	item := createTestInventoryItem(tenantID, warehouseID, productID)
	invRepo.items[item.ID] = item
	invRepo.findByWPID = item

	reqBody := IncreaseStockRequest{
		WarehouseID: warehouseID.String(),
		ProductID:   productID.String(),
		Quantity:    50.0,
		UnitCost:    15.50,
		SourceType:  "PURCHASE_ORDER",
		SourceID:    "PO-2024-001",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/inventory/stock/increase", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Tenant-ID", tenantID.String())

	handler.IncreaseStock(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestInventoryHandler_IncreaseStock_InvalidSourceType(t *testing.T) {
	handler, _, _, _ := setupInventoryTestHandler()

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	reqBody := IncreaseStockRequest{
		WarehouseID: uuid.New().String(),
		ProductID:   uuid.New().String(),
		Quantity:    50.0,
		UnitCost:    15.50,
		SourceType:  "INVALID_SOURCE",
		SourceID:    "PO-2024-001",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/inventory/stock/increase", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Tenant-ID", tenantID.String())

	handler.IncreaseStock(c)

	// Should fail with domain error
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestInventoryHandler_LockStock_Success(t *testing.T) {
	handler, invRepo, _, _ := setupInventoryTestHandler()

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	warehouseID := uuid.New()
	productID := uuid.New()

	item := createTestInventoryItem(tenantID, warehouseID, productID)
	invRepo.items[item.ID] = item
	invRepo.findByWPID = item

	reqBody := LockStockRequest{
		WarehouseID: warehouseID.String(),
		ProductID:   productID.String(),
		Quantity:    10.0,
		SourceType:  "sales_order",
		SourceID:    "SO-2024-001",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/inventory/stock/lock", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Tenant-ID", tenantID.String())

	handler.LockStock(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	data := resp.Data.(map[string]interface{})
	assert.NotEmpty(t, data["lock_id"])
}

func TestInventoryHandler_LockStock_InsufficientStock(t *testing.T) {
	handler, invRepo, _, _ := setupInventoryTestHandler()

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	warehouseID := uuid.New()
	productID := uuid.New()

	// Create item with only 10 units
	item := createTestInventoryItem(tenantID, warehouseID, productID)
	item.AvailableQuantity = decimal.NewFromInt(10)
	invRepo.items[item.ID] = item
	invRepo.findByWPID = item

	// Try to lock 100 units
	reqBody := LockStockRequest{
		WarehouseID: warehouseID.String(),
		ProductID:   productID.String(),
		Quantity:    100.0,
		SourceType:  "sales_order",
		SourceID:    "SO-2024-001",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/inventory/stock/lock", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Tenant-ID", tenantID.String())

	handler.LockStock(c)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestInventoryHandler_AdjustStock_Success(t *testing.T) {
	handler, invRepo, _, _ := setupInventoryTestHandler()

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	warehouseID := uuid.New()
	productID := uuid.New()

	item := createTestInventoryItem(tenantID, warehouseID, productID)
	invRepo.items[item.ID] = item

	reqBody := AdjustStockRequest{
		WarehouseID:    warehouseID.String(),
		ProductID:      productID.String(),
		ActualQuantity: 95.0,
		Reason:         "Stock count variance - 5 units damaged",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/inventory/stock/adjust", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Tenant-ID", tenantID.String())

	handler.AdjustStock(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestInventoryHandler_SetThresholds_Success(t *testing.T) {
	handler, invRepo, _, _ := setupInventoryTestHandler()

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	warehouseID := uuid.New()
	productID := uuid.New()

	item := createTestInventoryItem(tenantID, warehouseID, productID)
	invRepo.items[item.ID] = item

	minQty := 20.0
	maxQty := 500.0
	reqBody := SetThresholdsRequest{
		WarehouseID: warehouseID.String(),
		ProductID:   productID.String(),
		MinQuantity: &minQty,
		MaxQuantity: &maxQty,
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPut, "/inventory/thresholds", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Tenant-ID", tenantID.String())

	handler.SetThresholds(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInventoryHandler_ListTransactions_Success(t *testing.T) {
	handler, _, _, txRepo := setupInventoryTestHandler()

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	itemID := uuid.New()

	// Add a test transaction
	tx, _ := inventory.NewInventoryTransaction(
		tenantID,
		itemID,
		uuid.New(),
		uuid.New(),
		inventory.TransactionTypeInbound,
		decimal.NewFromInt(50),
		decimal.NewFromFloat(15.50),
		decimal.Zero,
		decimal.NewFromInt(50),
		inventory.SourceTypePurchaseOrder,
		"PO-2024-001",
	)
	txRepo.txs[tx.ID] = tx

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/inventory/transactions?page=1&page_size=20", nil)
	c.Request.Header.Set("X-Tenant-ID", tenantID.String())

	handler.ListTransactions(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestInventoryHandler_GetActiveLocks_Success(t *testing.T) {
	handler, invRepo, lockRepo, _ := setupInventoryTestHandler()

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	warehouseID := uuid.New()
	productID := uuid.New()

	item := createTestInventoryItem(tenantID, warehouseID, productID)
	invRepo.items[item.ID] = item
	invRepo.findByWPID = item

	// Add an active lock
	lock := inventory.NewStockLock(
		item.ID,
		decimal.NewFromInt(10),
		"sales_order",
		"SO-2024-001",
		time.Now().Add(30*time.Minute),
	)
	lockRepo.locks[lock.ID] = lock

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/inventory/locks?warehouse_id="+warehouseID.String()+"&product_id="+productID.String(), nil)
	c.Request.Header.Set("X-Tenant-ID", tenantID.String())

	handler.GetActiveLocks(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestInventoryHandler_GetActiveLocks_MissingParams(t *testing.T) {
	handler, _, _, _ := setupInventoryTestHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/inventory/locks", nil)

	handler.GetActiveLocks(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestParseDateTime(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "RFC3339 format",
			input:   "2024-01-15T10:30:00Z",
			wantErr: false,
		},
		{
			name:    "ISO date format",
			input:   "2024-01-15",
			wantErr: false,
		},
		{
			name:    "Datetime without timezone",
			input:   "2024-01-15 10:30:00",
			wantErr: false,
		},
		{
			name:    "Invalid format",
			input:   "15/01/2024",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseDateTime(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
