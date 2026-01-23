package strategy

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// CostMethod represents the inventory costing method
type CostMethod string

const (
	CostMethodMovingAverage CostMethod = "moving_average"
	CostMethodFIFO          CostMethod = "fifo"
	CostMethodLIFO          CostMethod = "lifo"
	CostMethodSpecific      CostMethod = "specific"
)

// String returns the string representation of the cost method
func (m CostMethod) String() string {
	return string(m)
}

// StockEntry represents a stock entry for cost calculation
type StockEntry struct {
	ID            string
	ProductID     string
	WarehouseID   string
	Quantity      decimal.Decimal
	UnitCost      decimal.Decimal
	TotalCost     decimal.Decimal
	EntryDate     time.Time
	BatchNumber   string
	ReferenceID   string
	ReferenceType string
}

// CostContext provides context for cost calculation
type CostContext struct {
	TenantID    string
	ProductID   string
	WarehouseID string
	Quantity    decimal.Decimal
	Date        time.Time
}

// CostResult contains the result of cost calculation
type CostResult struct {
	UnitCost     decimal.Decimal
	TotalCost    decimal.Decimal
	Method       CostMethod
	EntriesUsed  []StockEntry
	RemainingQty decimal.Decimal
}

// CostCalculationStrategy defines the interface for inventory cost calculation
type CostCalculationStrategy interface {
	Strategy
	// Method returns the costing method used by this strategy
	Method() CostMethod
	// CalculateCost calculates the cost for a given quantity of product
	CalculateCost(ctx context.Context, costCtx CostContext, entries []StockEntry) (CostResult, error)
	// CalculateAverageCost calculates the average cost for all entries
	CalculateAverageCost(ctx context.Context, entries []StockEntry) (decimal.Decimal, error)
}
