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
	assert.Equal(t, quantity, item.AvailableQuantity.Amount())
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
	assert.Equal(t, decimal.NewFromInt(150), item.AvailableQuantity.Amount()) // 50 + 100
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

// MockCostStrategyResolver implements CostStrategyResolver for testing
type MockCostStrategyResolver struct {
	strategies map[string]strategy.CostCalculationStrategy
}

func (m *MockCostStrategyResolver) ResolveCostStrategy(name string) strategy.CostCalculationStrategy {
	if m.strategies == nil {
		return nil
	}
	return m.strategies[name]
}

func TestMapTenantConfigToStrategyName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "weighted_average maps to moving_average",
			input:    "weighted_average",
			expected: "moving_average",
		},
		{
			name:     "moving_average maps to moving_average",
			input:    "moving_average",
			expected: "moving_average",
		},
		{
			name:     "fifo maps to fifo",
			input:    "fifo",
			expected: "fifo",
		},
		{
			name:     "lifo maps to lifo",
			input:    "lifo",
			expected: "lifo",
		},
		{
			name:     "specific maps to specific",
			input:    "specific",
			expected: "specific",
		},
		{
			name:     "empty string uses default",
			input:    "",
			expected: DefaultCostStrategyName,
		},
		{
			name:     "unknown value is passed through",
			input:    "custom_strategy",
			expected: "custom_strategy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapTenantConfigToStrategyName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewInventoryDomainServiceWithResolver(t *testing.T) {
	resolver := &MockCostStrategyResolver{
		strategies: map[string]strategy.CostCalculationStrategy{
			"moving_average": &MockCostStrategy{method: strategy.CostMethodMovingAverage},
		},
	}

	service := NewInventoryDomainServiceWithResolver(resolver, nil)

	assert.NotNil(t, service)
	assert.NotNil(t, service.strategyResolver)
	assert.Nil(t, service.costStrategy) // Direct strategy not set
}

func TestInventoryDomainService_StockInWithStrategyName(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	// Create inventory item
	item, err := NewInventoryItem(tenantID, warehouseID, productID)
	require.NoError(t, err)

	// Set up mock strategy
	expectedCost := decimal.NewFromFloat(15.5)
	mockStrategy := &MockCostStrategy{
		method:            strategy.CostMethodMovingAverage,
		calculatedAverage: expectedCost,
	}

	// Create resolver with the strategy
	resolver := &MockCostStrategyResolver{
		strategies: map[string]strategy.CostCalculationStrategy{
			"moving_average": mockStrategy,
		},
	}

	service := NewInventoryDomainServiceWithResolver(resolver, nil)

	// Perform stock-in with strategy name
	quantity := decimal.NewFromInt(100)
	unitCost := valueobject.NewMoneyCNY(decimal.NewFromFloat(10.0))

	result, err := service.StockInWithStrategyName(ctx, item, quantity, unitCost, nil, "moving_average")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, item, result.Item)
	assert.Equal(t, strategy.CostMethodMovingAverage, result.CostMethod)
	assert.Equal(t, expectedCost.Round(4), result.CostCalculated)
}

func TestInventoryDomainService_StockInWithStrategyName_FallbackOnUnknownStrategy(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	// Create inventory item
	item, err := NewInventoryItem(tenantID, warehouseID, productID)
	require.NoError(t, err)

	// Create resolver that doesn't have the requested strategy
	resolver := &MockCostStrategyResolver{
		strategies: map[string]strategy.CostCalculationStrategy{
			"fifo": &MockCostStrategy{method: strategy.CostMethodFIFO},
		},
	}

	service := NewInventoryDomainServiceWithResolver(resolver, nil)

	// Perform stock-in with unknown strategy name
	quantity := decimal.NewFromInt(100)
	unitCost := valueobject.NewMoneyCNY(decimal.NewFromFloat(10.0))

	result, err := service.StockInWithStrategyName(ctx, item, quantity, unitCost, nil, "unknown_strategy")

	// Should succeed using built-in moving average fallback
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, strategy.CostMethodMovingAverage, result.CostMethod)
}

func TestInventoryDomainService_StockInWithStrategyName_EmptyStrategyNameUsesDefault(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	// Create inventory item
	item, err := NewInventoryItem(tenantID, warehouseID, productID)
	require.NoError(t, err)

	// Set up mock strategy for default name
	expectedCost := decimal.NewFromFloat(12.0)
	mockStrategy := &MockCostStrategy{
		method:            strategy.CostMethodMovingAverage,
		calculatedAverage: expectedCost,
	}

	// Create resolver with strategy mapped to default name
	resolver := &MockCostStrategyResolver{
		strategies: map[string]strategy.CostCalculationStrategy{
			DefaultCostStrategyName: mockStrategy, // "moving_average"
		},
	}

	service := NewInventoryDomainServiceWithResolver(resolver, nil)

	// Perform stock-in with empty strategy name
	quantity := decimal.NewFromInt(100)
	unitCost := valueobject.NewMoneyCNY(decimal.NewFromFloat(10.0))

	result, err := service.StockInWithStrategyName(ctx, item, quantity, unitCost, nil, "")

	require.NoError(t, err)
	assert.NotNil(t, result)
	// Should use the default strategy
	assert.Equal(t, expectedCost.Round(4), result.CostCalculated)
}

func TestInventoryDomainService_StockInWithStrategyName_NoResolverFallback(t *testing.T) {
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

	// Service without resolver should use built-in fallback
	service := NewInventoryDomainServiceWithResolver(nil, nil)

	// Perform stock-in
	quantity := decimal.NewFromInt(100)
	unitCost := valueobject.NewMoneyCNY(decimal.NewFromFloat(20.0))

	result, err := service.StockInWithStrategyName(ctx, item, quantity, unitCost, nil, "any_strategy")

	require.NoError(t, err)
	assert.NotNil(t, result)
	// Should use built-in moving average fallback
	assert.Equal(t, strategy.CostMethodMovingAverage, result.CostMethod)

	// Expected moving average: (100 * 10 + 100 * 20) / 200 = 15.00
	expectedCost := decimal.NewFromFloat(15.0)
	assert.True(t, expectedCost.Equal(result.CostCalculated), "Expected %s but got %s", expectedCost, result.CostCalculated)
}

func TestInventoryDomainService_SetStrategyResolver(t *testing.T) {
	service := NewInventoryDomainService(nil, nil)

	resolver := &MockCostStrategyResolver{
		strategies: map[string]strategy.CostCalculationStrategy{
			"moving_average": &MockCostStrategy{method: strategy.CostMethodMovingAverage},
		},
	}

	service.SetStrategyResolver(resolver)

	assert.Equal(t, resolver, service.strategyResolver)
}
