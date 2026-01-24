package report

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// InventoryTurnover is a read model for inventory turnover statistics
type InventoryTurnover struct {
	ProductID        uuid.UUID       `json:"product_id"`
	ProductSKU       string          `json:"product_sku"`
	ProductName      string          `json:"product_name"`
	CategoryID       *uuid.UUID      `json:"category_id,omitempty"`
	CategoryName     string          `json:"category_name,omitempty"`
	WarehouseID      *uuid.UUID      `json:"warehouse_id,omitempty"`
	WarehouseName    string          `json:"warehouse_name,omitempty"`
	BeginningStock   decimal.Decimal `json:"beginning_stock"`
	EndingStock      decimal.Decimal `json:"ending_stock"`
	AverageStock     decimal.Decimal `json:"average_stock"`
	SoldQuantity     decimal.Decimal `json:"sold_quantity"`
	TurnoverRate     decimal.Decimal `json:"turnover_rate"`     // SoldQuantity / AverageStock
	DaysOfInventory  decimal.Decimal `json:"days_of_inventory"` // Average days inventory is held
	StockValue       decimal.Decimal `json:"stock_value"`       // Current value
}

// InventorySummary provides aggregated inventory statistics
type InventorySummary struct {
	TotalProducts      int64           `json:"total_products"`
	TotalQuantity      decimal.Decimal `json:"total_quantity"`
	TotalValue         decimal.Decimal `json:"total_value"`
	AvgTurnoverRate    decimal.Decimal `json:"avg_turnover_rate"`
	LowStockCount      int64           `json:"low_stock_count"`
	OutOfStockCount    int64           `json:"out_of_stock_count"`
	OverstockCount     int64           `json:"overstock_count"`
}

// InventoryValueByCategory represents inventory value grouped by category
type InventoryValueByCategory struct {
	CategoryID    *uuid.UUID      `json:"category_id,omitempty"`
	CategoryName  string          `json:"category_name"`
	ProductCount  int64           `json:"product_count"`
	TotalQuantity decimal.Decimal `json:"total_quantity"`
	TotalValue    decimal.Decimal `json:"total_value"`
	Percentage    decimal.Decimal `json:"percentage"` // Percentage of total value
}

// InventoryValueByWarehouse represents inventory value grouped by warehouse
type InventoryValueByWarehouse struct {
	WarehouseID   uuid.UUID       `json:"warehouse_id"`
	WarehouseName string          `json:"warehouse_name"`
	ProductCount  int64           `json:"product_count"`
	TotalQuantity decimal.Decimal `json:"total_quantity"`
	TotalValue    decimal.Decimal `json:"total_value"`
	Percentage    decimal.Decimal `json:"percentage"` // Percentage of total value
}

// InventoryAgingReport represents inventory aging analysis
type InventoryAgingReport struct {
	ProductID       uuid.UUID       `json:"product_id"`
	ProductSKU      string          `json:"product_sku"`
	ProductName     string          `json:"product_name"`
	WarehouseID     uuid.UUID       `json:"warehouse_id"`
	WarehouseName   string          `json:"warehouse_name"`
	Quantity        decimal.Decimal `json:"quantity"`
	Value           decimal.Decimal `json:"value"`
	DaysInStock     int             `json:"days_in_stock"`
	AgingBucket     string          `json:"aging_bucket"` // "0-30", "31-60", "61-90", "90+"
}

// InventoryTurnoverFilter defines filtering options for inventory reports
type InventoryTurnoverFilter struct {
	TenantID    uuid.UUID  `json:"-"`
	StartDate   time.Time  `json:"start_date"`
	EndDate     time.Time  `json:"end_date"`
	ProductID   *uuid.UUID `json:"product_id,omitempty"`
	CategoryID  *uuid.UUID `json:"category_id,omitempty"`
	WarehouseID *uuid.UUID `json:"warehouse_id,omitempty"`
	TopN        int        `json:"top_n,omitempty"`
}

// InventoryReportRepository defines the interface for inventory report queries
type InventoryReportRepository interface {
	// GetInventorySummary returns aggregated inventory summary
	GetInventorySummary(filter InventoryTurnoverFilter) (*InventorySummary, error)

	// GetInventoryTurnover returns inventory turnover for products
	GetInventoryTurnover(filter InventoryTurnoverFilter) ([]InventoryTurnover, error)

	// GetInventoryValueByCategory returns inventory value grouped by category
	GetInventoryValueByCategory(filter InventoryTurnoverFilter) ([]InventoryValueByCategory, error)

	// GetInventoryValueByWarehouse returns inventory value grouped by warehouse
	GetInventoryValueByWarehouse(filter InventoryTurnoverFilter) ([]InventoryValueByWarehouse, error)

	// GetSlowMovingProducts returns products with low turnover
	GetSlowMovingProducts(filter InventoryTurnoverFilter) ([]InventoryTurnover, error)
}
