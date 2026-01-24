package report

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// SalesReport is a read model for sales statistics
// This is a CQRS read model optimized for querying
type SalesReport struct {
	Date          time.Time       `json:"date"`
	ProductID     uuid.UUID       `json:"product_id"`
	ProductSKU    string          `json:"product_sku"`
	ProductName   string          `json:"product_name"`
	CategoryID    *uuid.UUID      `json:"category_id,omitempty"`
	CategoryName  string          `json:"category_name,omitempty"`
	SalesQuantity decimal.Decimal `json:"sales_quantity"`
	SalesAmount   decimal.Decimal `json:"sales_amount"`
	CostAmount    decimal.Decimal `json:"cost_amount"`
	GrossProfit   decimal.Decimal `json:"gross_profit"`
	ProfitMargin  decimal.Decimal `json:"profit_margin"` // Percentage
}

// SalesSummary provides aggregated sales statistics
type SalesSummary struct {
	PeriodStart      time.Time       `json:"period_start"`
	PeriodEnd        time.Time       `json:"period_end"`
	TotalOrders      int64           `json:"total_orders"`
	TotalQuantity    decimal.Decimal `json:"total_quantity"`
	TotalSalesAmount decimal.Decimal `json:"total_sales_amount"`
	TotalCostAmount  decimal.Decimal `json:"total_cost_amount"`
	TotalGrossProfit decimal.Decimal `json:"total_gross_profit"`
	AvgOrderValue    decimal.Decimal `json:"avg_order_value"`
	ProfitMargin     decimal.Decimal `json:"profit_margin"` // Percentage
}

// DailySalesTrend represents daily sales trend data
type DailySalesTrend struct {
	Date         time.Time       `json:"date"`
	OrderCount   int64           `json:"order_count"`
	TotalAmount  decimal.Decimal `json:"total_amount"`
	TotalProfit  decimal.Decimal `json:"total_profit"`
	ItemsSold    decimal.Decimal `json:"items_sold"`
}

// ProductSalesRanking represents product sales ranking
type ProductSalesRanking struct {
	Rank          int             `json:"rank"`
	ProductID     uuid.UUID       `json:"product_id"`
	ProductSKU    string          `json:"product_sku"`
	ProductName   string          `json:"product_name"`
	CategoryName  string          `json:"category_name,omitempty"`
	TotalQuantity decimal.Decimal `json:"total_quantity"`
	TotalAmount   decimal.Decimal `json:"total_amount"`
	TotalProfit   decimal.Decimal `json:"total_profit"`
	OrderCount    int64           `json:"order_count"`
}

// CustomerSalesRanking represents customer sales ranking
type CustomerSalesRanking struct {
	Rank          int             `json:"rank"`
	CustomerID    uuid.UUID       `json:"customer_id"`
	CustomerName  string          `json:"customer_name"`
	TotalOrders   int64           `json:"total_orders"`
	TotalQuantity decimal.Decimal `json:"total_quantity"`
	TotalAmount   decimal.Decimal `json:"total_amount"`
	TotalProfit   decimal.Decimal `json:"total_profit"`
}

// SalesReportFilter defines filtering options for sales reports
type SalesReportFilter struct {
	TenantID   uuid.UUID  `json:"-"`
	StartDate  time.Time  `json:"start_date"`
	EndDate    time.Time  `json:"end_date"`
	ProductID  *uuid.UUID `json:"product_id,omitempty"`
	CategoryID *uuid.UUID `json:"category_id,omitempty"`
	CustomerID *uuid.UUID `json:"customer_id,omitempty"`
	TopN       int        `json:"top_n,omitempty"` // For rankings
}

// SalesReportRepository defines the interface for sales report queries
type SalesReportRepository interface {
	// GetSalesSummary returns aggregated sales summary for the period
	GetSalesSummary(filter SalesReportFilter) (*SalesSummary, error)

	// GetDailySalesTrend returns daily sales trend data
	GetDailySalesTrend(filter SalesReportFilter) ([]DailySalesTrend, error)

	// GetProductSalesReport returns sales report grouped by product
	GetProductSalesReport(filter SalesReportFilter) ([]SalesReport, error)

	// GetProductSalesRanking returns top N products by sales
	GetProductSalesRanking(filter SalesReportFilter) ([]ProductSalesRanking, error)

	// GetCustomerSalesRanking returns top N customers by sales
	GetCustomerSalesRanking(filter SalesReportFilter) ([]CustomerSalesRanking, error)
}
