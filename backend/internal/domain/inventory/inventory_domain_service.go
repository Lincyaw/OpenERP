package inventory

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/shopspring/decimal"
)

// CostStrategyResolver resolves cost strategies by name.
// This interface allows the domain service to resolve strategies without depending
// on infrastructure details. The application layer injects a resolver that knows
// how to look up strategies from a registry.
type CostStrategyResolver interface {
	// ResolveCostStrategy returns a cost strategy by name.
	// If name is empty or strategy not found, returns nil.
	ResolveCostStrategy(name string) strategy.CostCalculationStrategy
}

// DefaultCostStrategyName is the default cost calculation method when no strategy is specified.
const DefaultCostStrategyName = "moving_average"

// MapTenantConfigToStrategyName maps tenant configuration values to strategy names.
// This centralizes the mapping logic in the domain layer.
func MapTenantConfigToStrategyName(configValue string) string {
	switch configValue {
	case "weighted_average", "moving_average":
		return "moving_average"
	case "fifo":
		return "fifo"
	case "lifo":
		return "lifo"
	case "specific":
		return "specific"
	default:
		if configValue != "" {
			return configValue
		}
		return DefaultCostStrategyName
	}
}

// InventoryDomainService provides domain logic that requires external strategy injection.
// This service follows DDD principles by keeping business logic in the domain layer
// while allowing configurable behavior through strategy pattern.
//
// The service handles strategy selection and fallback internally:
// - When a strategy resolver is provided, it can resolve strategies by name
// - When no resolver is available or strategy not found, it falls back to built-in moving average
// - The application layer only needs to provide the strategy name (not the strategy itself)
type InventoryDomainService struct {
	costStrategy     strategy.CostCalculationStrategy
	batchStrategy    strategy.BatchManagementStrategy
	strategyResolver CostStrategyResolver
}

// NewInventoryDomainService creates a new InventoryDomainService with injected strategies.
// If nil is passed for any strategy, a default strategy will be used.
func NewInventoryDomainService(
	costStrategy strategy.CostCalculationStrategy,
	batchStrategy strategy.BatchManagementStrategy,
) *InventoryDomainService {
	return &InventoryDomainService{
		costStrategy:  costStrategy,
		batchStrategy: batchStrategy,
	}
}

// NewInventoryDomainServiceWithResolver creates a new InventoryDomainService with a strategy resolver.
// This is the preferred constructor for production use, as it allows the domain service
// to resolve strategies by name and handle fallback internally.
func NewInventoryDomainServiceWithResolver(
	strategyResolver CostStrategyResolver,
	batchStrategy strategy.BatchManagementStrategy,
) *InventoryDomainService {
	return &InventoryDomainService{
		strategyResolver: strategyResolver,
		batchStrategy:    batchStrategy,
	}
}

// SetStrategyResolver sets the strategy resolver for runtime configuration.
func (s *InventoryDomainService) SetStrategyResolver(resolver CostStrategyResolver) {
	s.strategyResolver = resolver
}

// StockInResult contains the result of a stock-in operation
type StockInResult struct {
	Item           *InventoryItem
	NewUnitCost    valueobject.Money
	OldUnitCost    valueobject.Money
	CostMethod     strategy.CostMethod
	CostCalculated decimal.Decimal // The calculated unit cost before rounding
}

// StockIn performs a stock increase operation using the injected cost calculation strategy.
// It calculates the new unit cost based on the configured strategy (moving average, FIFO, etc.)
// and applies it to the inventory item.
func (s *InventoryDomainService) StockIn(
	ctx context.Context,
	item *InventoryItem,
	quantity decimal.Decimal,
	unitCost valueobject.Money,
	batchInfo *BatchInfo,
) (*StockInResult, error) {
	if item == nil {
		return nil, shared.NewDomainError("INVALID_ITEM", "Inventory item cannot be nil")
	}
	if quantity.LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("INVALID_QUANTITY", "Quantity must be positive")
	}
	if unitCost.Amount().IsNegative() {
		return nil, shared.NewDomainError("INVALID_COST", "Unit cost cannot be negative")
	}

	oldCost := item.UnitCost

	// Calculate new cost using the injected strategy
	newCostAmount, costMethod, err := s.calculateStockInCost(ctx, item, quantity, unitCost)
	if err != nil {
		return nil, err
	}

	// Create the new cost as Money
	newCost := valueobject.NewMoneyCNY(newCostAmount)

	// Increase stock with the calculated cost
	err = item.IncreaseStockWithCost(quantity, newCostAmount, batchInfo)
	if err != nil {
		return nil, err
	}

	// Emit cost changed event if cost actually changed
	if !oldCost.Equal(newCostAmount) {
		item.AddDomainEvent(NewInventoryCostChangedEvent(
			item,
			valueobject.NewMoneyCNY(oldCost),
			newCost,
		))
	}

	return &StockInResult{
		Item:           item,
		NewUnitCost:    newCost,
		OldUnitCost:    valueobject.NewMoneyCNY(oldCost),
		CostMethod:     costMethod,
		CostCalculated: newCostAmount,
	}, nil
}

// calculateStockInCost calculates the new unit cost using the configured strategy.
// Returns the calculated unit cost, the cost method used, and any error.
func (s *InventoryDomainService) calculateStockInCost(
	ctx context.Context,
	item *InventoryItem,
	incomingQuantity decimal.Decimal,
	incomingCost valueobject.Money,
) (decimal.Decimal, strategy.CostMethod, error) {
	// If no cost strategy is set, use moving average as default
	if s.costStrategy == nil {
		return s.calculateMovingAverageCost(item, incomingQuantity, incomingCost.Amount()), strategy.CostMethodMovingAverage, nil
	}

	// Build existing stock entries from the inventory item
	currentStock := item.TotalQuantity()

	// Create stock entries for the strategy
	entries := make([]strategy.StockEntry, 0)

	// Add existing stock as a single entry
	if currentStock.GreaterThan(decimal.Zero) {
		entries = append(entries, strategy.StockEntry{
			ID:          item.ID.String(),
			ProductID:   item.ProductID.String(),
			WarehouseID: item.WarehouseID.String(),
			Quantity:    currentStock,
			UnitCost:    item.UnitCost,
			TotalCost:   currentStock.Mul(item.UnitCost),
			EntryDate:   item.CreatedAt, // Use creation date as baseline
		})
	}

	// Add incoming stock as a new entry
	entries = append(entries, strategy.StockEntry{
		ID:          "incoming",
		ProductID:   item.ProductID.String(),
		WarehouseID: item.WarehouseID.String(),
		Quantity:    incomingQuantity,
		UnitCost:    incomingCost.Amount(),
		TotalCost:   incomingQuantity.Mul(incomingCost.Amount()),
		EntryDate:   time.Now(),
	})

	// Calculate average cost using the strategy
	avgCost, err := s.costStrategy.CalculateAverageCost(ctx, entries)
	if err != nil {
		// Fallback to moving average calculation on error
		return s.calculateMovingAverageCost(item, incomingQuantity, incomingCost.Amount()), s.costStrategy.Method(), nil
	}

	return avgCost.Round(4), s.costStrategy.Method(), nil
}

// calculateMovingAverageCost is the default cost calculation method.
// New Cost = (Old Quantity * Old Cost + New Quantity * New Cost) / (Old Quantity + New Quantity)
func (s *InventoryDomainService) calculateMovingAverageCost(
	item *InventoryItem,
	incomingQuantity decimal.Decimal,
	incomingCost decimal.Decimal,
) decimal.Decimal {
	currentStock := item.TotalQuantity()
	currentCost := item.UnitCost

	if currentStock.IsZero() {
		return incomingCost.Round(4)
	}

	totalValue := currentStock.Mul(currentCost).Add(incomingQuantity.Mul(incomingCost))
	totalQuantity := currentStock.Add(incomingQuantity)

	if totalQuantity.IsZero() {
		return decimal.Zero
	}

	return totalValue.Div(totalQuantity).Round(4)
}

// resolveStrategy resolves a cost strategy by name using the strategy resolver.
// It handles the fallback logic internally:
// 1. If strategyName is empty, uses the default strategy name
// 2. If resolver is available, tries to resolve the strategy
// 3. Returns nil if resolution fails (caller should use built-in fallback)
func (s *InventoryDomainService) resolveStrategy(strategyName string) strategy.CostCalculationStrategy {
	// Use injected strategy if set directly (legacy mode)
	if s.costStrategy != nil {
		return s.costStrategy
	}

	// Use resolver to get strategy by name
	if s.strategyResolver == nil {
		return nil
	}

	// Apply default if no name provided
	if strategyName == "" {
		strategyName = DefaultCostStrategyName
	}

	return s.strategyResolver.ResolveCostStrategy(strategyName)
}

// StockInWithStrategyName performs a stock increase operation using a cost strategy resolved by name.
// This is the preferred method for application layer usage, as it delegates strategy selection
// to the domain service. The domain service handles:
// 1. Strategy resolution by name (using injected resolver)
// 2. Fallback to built-in moving average if strategy not found
// 3. Cost calculation and domain event emission
//
// Parameters:
//   - ctx: context for the operation
//   - item: the inventory item to increase stock for
//   - quantity: the quantity to add
//   - unitCost: the unit cost of the incoming stock
//   - batchInfo: optional batch information
//   - strategyName: the name of the cost strategy to use (e.g., "moving_average", "fifo")
//     If empty, defaults to "moving_average"
//
// The application layer should:
// 1. Look up the tenant's configured strategy name
// 2. Map the tenant config value to a strategy name using MapTenantConfigToStrategyName
// 3. Pass the strategy name to this method
func (s *InventoryDomainService) StockInWithStrategyName(
	ctx context.Context,
	item *InventoryItem,
	quantity decimal.Decimal,
	unitCost valueobject.Money,
	batchInfo *BatchInfo,
	strategyName string,
) (*StockInResult, error) {
	// Resolve strategy by name
	resolvedStrategy := s.resolveStrategy(strategyName)

	// Temporarily set the resolved strategy for this operation
	originalStrategy := s.costStrategy
	s.costStrategy = resolvedStrategy
	defer func() {
		s.costStrategy = originalStrategy
	}()

	// Delegate to StockIn which handles validation, calculation, and events
	return s.StockIn(ctx, item, quantity, unitCost, batchInfo)
}

// GetCostStrategy returns the current cost calculation strategy
func (s *InventoryDomainService) GetCostStrategy() strategy.CostCalculationStrategy {
	return s.costStrategy
}

// GetCostMethod returns the cost method used by the current strategy
func (s *InventoryDomainService) GetCostMethod() strategy.CostMethod {
	if s.costStrategy == nil {
		return strategy.CostMethodMovingAverage
	}
	return s.costStrategy.Method()
}

// GetBatchStrategy returns the current batch management strategy
func (s *InventoryDomainService) GetBatchStrategy() strategy.BatchManagementStrategy {
	return s.batchStrategy
}
