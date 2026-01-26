package inventory

import (
	"context"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCostStrategy implements CostCalculationStrategy for testing
type MockCostStrategy struct {
	method            strategy.CostMethod
	calculatedCost    decimal.Decimal
	calculatedAverage decimal.Decimal
	calculateErr      error
}

func (m *MockCostStrategy) Name() string {
	return string(m.method)
}

func (m *MockCostStrategy) Type() strategy.StrategyType {
	return strategy.StrategyTypeCost
}

func (m *MockCostStrategy) Description() string {
	return "Mock cost strategy for testing"
}

func (m *MockCostStrategy) Method() strategy.CostMethod {
	return m.method
}

func (m *MockCostStrategy) CalculateCost(
	ctx context.Context,
	costCtx strategy.CostContext,
	entries []strategy.StockEntry,
) (strategy.CostResult, error) {
	if m.calculateErr != nil {
		return strategy.CostResult{}, m.calculateErr
	}
	return strategy.CostResult{
		UnitCost:  m.calculatedCost,
		TotalCost: m.calculatedCost.Mul(costCtx.Quantity),
		Method:    m.method,
	}, nil
}

func (m *MockCostStrategy) CalculateAverageCost(
	ctx context.Context,
	entries []strategy.StockEntry,
) (decimal.Decimal, error) {
	if m.calculateErr != nil {
		return decimal.Zero, m.calculateErr
	}
	return m.calculatedAverage, nil
}

func TestNewInventoryDomainService(t *testing.T) {
	mockStrategy := &MockCostStrategy{method: strategy.CostMethodMovingAverage}

	service := NewInventoryDomainService(mockStrategy, nil)

	assert.NotNil(t, service)
	assert.Equal(t, mockStrategy, service.GetCostStrategy())
	assert.Equal(t, strategy.CostMethodMovingAverage, service.GetCostMethod())
}

func TestInventoryDomainService_GetCostMethod_DefaultsToMovingAverage(t *testing.T) {
	service := NewInventoryDomainService(nil, nil)

	assert.Equal(t, strategy.CostMethodMovingAverage, service.GetCostMethod())
}

func TestInventoryDomainService_StockIn_WithStrategy(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	// Create inventory item
	item, err := NewInventoryItem(tenantID, warehouseID, productID)
	require.NoError(t, err)

	// Set up mock strategy to return specific calculated cost
	expectedCost := decimal.NewFromFloat(15.5)
	mockStrategy := &MockCostStrategy{
		method:            strategy.CostMethodMovingAverage,
		calculatedAverage: expectedCost,
	}

	service := NewInventoryDomainService(mockStrategy, nil)

	// Perform stock-in
	quantity := decimal.NewFromInt(100)
	unitCost := valueobject.NewMoneyCNY(decimal.NewFromFloat(10.0))

	result, err := service.StockIn(ctx, item, quantity, unitCost, nil)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, item, result.Item)
	assert.Equal(t, strategy.CostMethodMovingAverage, result.CostMethod)
	assert.Equal(t, expectedCost.Round(4), result.CostCalculated)
	assert.Equal(t, quantity, item.AvailableQuantity)
}

func TestInventoryDomainService_StockIn_FIFOStrategy(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	// Create inventory item with existing stock
	item, err := NewInventoryItem(tenantID, warehouseID, productID)
	require.NoError(t, err)

	// Add some initial stock
	initialCost := valueobject.NewMoneyCNY(decimal.NewFromFloat(10.0))
	err = item.IncreaseStock(decimal.NewFromInt(50), initialCost, nil)
	require.NoError(t, err)

	// Set up FIFO strategy
	expectedCost := decimal.NewFromFloat(12.0)
	mockStrategy := &MockCostStrategy{
		method:            strategy.CostMethodFIFO,
		calculatedAverage: expectedCost,
	}

	service := NewInventoryDomainService(mockStrategy, nil)

	// Perform stock-in with new cost
	quantity := decimal.NewFromInt(100)
	unitCost := valueobject.NewMoneyCNY(decimal.NewFromFloat(15.0))

	result, err := service.StockIn(ctx, item, quantity, unitCost, nil)

	require.NoError(t, err)
	assert.Equal(t, strategy.CostMethodFIFO, result.CostMethod)
	assert.Equal(t, decimal.NewFromInt(150), item.AvailableQuantity) // 50 + 100
}

func TestInventoryDomainService_StockIn_InvalidQuantity(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	item, err := NewInventoryItem(tenantID, warehouseID, productID)
	require.NoError(t, err)

	service := NewInventoryDomainService(nil, nil)

	// Test zero quantity
	result, err := service.StockIn(ctx, item, decimal.Zero, valueobject.NewMoneyCNY(decimal.NewFromInt(10)), nil)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "Quantity must be positive")

	// Test negative quantity
	result, err = service.StockIn(ctx, item, decimal.NewFromInt(-10), valueobject.NewMoneyCNY(decimal.NewFromInt(10)), nil)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "Quantity must be positive")
}

func TestInventoryDomainService_StockIn_InvalidCost(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	item, err := NewInventoryItem(tenantID, warehouseID, productID)
	require.NoError(t, err)

	service := NewInventoryDomainService(nil, nil)

	// Test negative cost
	result, err := service.StockIn(ctx, item, decimal.NewFromInt(10), valueobject.NewMoneyCNY(decimal.NewFromInt(-10)), nil)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "Unit cost cannot be negative")
}

func TestInventoryDomainService_StockIn_NilItem(t *testing.T) {
	ctx := context.Background()

	service := NewInventoryDomainService(nil, nil)

	result, err := service.StockIn(ctx, nil, decimal.NewFromInt(10), valueobject.NewMoneyCNY(decimal.NewFromInt(10)), nil)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "Inventory item cannot be nil")
}

func TestInventoryDomainService_StockIn_WithBatch(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	item, err := NewInventoryItem(tenantID, warehouseID, productID)
	require.NoError(t, err)

	service := NewInventoryDomainService(nil, nil)

	// Create batch info
	expiryDate := time.Now().AddDate(1, 0, 0)
	batchInfo := NewBatchInfo("BATCH-001", nil, &expiryDate)

	quantity := decimal.NewFromInt(100)
	unitCost := valueobject.NewMoneyCNY(decimal.NewFromFloat(10.0))

	result, err := service.StockIn(ctx, item, quantity, unitCost, batchInfo)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(item.Batches))
	assert.Equal(t, "BATCH-001", item.Batches[0].BatchNumber)
}

func TestInventoryDomainService_StockIn_NilStrategy_FallbackToMovingAverage(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	// Create inventory item with existing stock
	item, err := NewInventoryItem(tenantID, warehouseID, productID)
	require.NoError(t, err)

	// Add initial stock: 100 units at $10.00
	initialCost := valueobject.NewMoneyCNY(decimal.NewFromFloat(10.0))
	err = item.IncreaseStock(decimal.NewFromInt(100), initialCost, nil)
	require.NoError(t, err)

	// Service with nil strategy should use fallback moving average
	service := NewInventoryDomainService(nil, nil)

	// Add new stock: 100 units at $20.00
	quantity := decimal.NewFromInt(100)
	unitCost := valueobject.NewMoneyCNY(decimal.NewFromFloat(20.0))

	result, err := service.StockIn(ctx, item, quantity, unitCost, nil)

	require.NoError(t, err)
	assert.Equal(t, strategy.CostMethodMovingAverage, result.CostMethod)

	// Expected moving average: (100 * 10 + 100 * 20) / 200 = 15.00
	expectedCost := decimal.NewFromFloat(15.0)
	assert.True(t, expectedCost.Equal(result.CostCalculated), "Expected %s but got %s", expectedCost, result.CostCalculated)
}

func TestInventoryDomainService_StockIn_EmitsEvents(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	item, err := NewInventoryItem(tenantID, warehouseID, productID)
	require.NoError(t, err)

	service := NewInventoryDomainService(nil, nil)

	quantity := decimal.NewFromInt(100)
	unitCost := valueobject.NewMoneyCNY(decimal.NewFromFloat(10.0))

	_, err = service.StockIn(ctx, item, quantity, unitCost, nil)

	require.NoError(t, err)

	// Check that domain events were emitted
	events := item.GetDomainEvents()
	assert.GreaterOrEqual(t, len(events), 1, "Should emit at least StockIncreased event")

	// First event should be StockIncreased
	foundStockIncreased := false
	for _, e := range events {
		if _, ok := e.(*StockIncreasedEvent); ok {
			foundStockIncreased = true
			break
		}
	}
	assert.True(t, foundStockIncreased, "Should emit StockIncreasedEvent")
}

func TestInventoryDomainService_StockIn_CostChanged_EmitsEvent(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	// Create inventory item with existing stock
	item, err := NewInventoryItem(tenantID, warehouseID, productID)
	require.NoError(t, err)

	// Add initial stock
	initialCost := valueobject.NewMoneyCNY(decimal.NewFromFloat(10.0))
	err = item.IncreaseStock(decimal.NewFromInt(100), initialCost, nil)
	require.NoError(t, err)
	item.ClearDomainEvents() // Clear initial events

	service := NewInventoryDomainService(nil, nil)

	// Add new stock with different cost
	quantity := decimal.NewFromInt(100)
	unitCost := valueobject.NewMoneyCNY(decimal.NewFromFloat(20.0))

	_, err = service.StockIn(ctx, item, quantity, unitCost, nil)

	require.NoError(t, err)

	// Check for InventoryCostChanged event
	events := item.GetDomainEvents()
	foundCostChanged := false
	for _, e := range events {
		if _, ok := e.(*InventoryCostChangedEvent); ok {
			foundCostChanged = true
			break
		}
	}
	assert.True(t, foundCostChanged, "Should emit InventoryCostChangedEvent when cost changes")
}
