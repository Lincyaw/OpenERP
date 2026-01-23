package strategy

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// Batch represents a product batch for batch management
type Batch struct {
	ID              string
	TenantID        string
	ProductID       string
	WarehouseID     string
	BatchNumber     string
	Quantity        decimal.Decimal
	AvailableQty    decimal.Decimal
	UnitCost        decimal.Decimal
	ManufactureDate time.Time
	ExpiryDate      time.Time
	ReceivedDate    time.Time
	Status          string
}

// BatchSelection represents a selection of batch for consumption
type BatchSelection struct {
	BatchID     string
	BatchNumber string
	Quantity    decimal.Decimal
	ExpiryDate  time.Time
	UnitCost    decimal.Decimal
}

// BatchSelectionContext provides context for batch selection
type BatchSelectionContext struct {
	TenantID    string
	ProductID   string
	WarehouseID string
	Quantity    decimal.Decimal
	Date        time.Time
	PreferBatch string // Optional: preferred batch number
}

// BatchSelectionResult contains the result of batch selection
type BatchSelectionResult struct {
	Selections   []BatchSelection
	TotalQty     decimal.Decimal
	ShortfallQty decimal.Decimal
}

// BatchManagementStrategy defines the interface for batch selection and management
type BatchManagementStrategy interface {
	Strategy
	// SelectBatches selects batches for consumption based on strategy rules
	SelectBatches(ctx context.Context, selCtx BatchSelectionContext, batches []Batch) (BatchSelectionResult, error)
	// ConsidersExpiry returns true if the strategy considers expiry dates
	ConsidersExpiry() bool
	// SupportsFEFO returns true if the strategy supports First Expired First Out
	SupportsFEFO() bool
}
