package inventory

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/shopspring/decimal"
)

// InventoryDomainService provides domain logic that requires external strategy injection.
// This service follows DDD principles by keeping business logic in the domain layer
// while allowing configurable behavior through strategy pattern.
type InventoryDomainService struct {
	costStrategy  strategy.CostCalculationStrategy
	batchStrategy strategy.BatchManagementStrategy
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
