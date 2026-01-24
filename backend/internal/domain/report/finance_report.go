package report

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ProfitLossStatement is a read model for profit and loss statement
type ProfitLossStatement struct {
	TenantID       uuid.UUID       `json:"tenant_id"`
	PeriodStart    time.Time       `json:"period_start"`
	PeriodEnd      time.Time       `json:"period_end"`
	SalesRevenue   decimal.Decimal `json:"sales_revenue"`     // Total sales revenue
	SalesReturns   decimal.Decimal `json:"sales_returns"`     // Sales returns/refunds
	NetSalesRevenue decimal.Decimal `json:"net_sales_revenue"` // SalesRevenue - SalesReturns
	COGS           decimal.Decimal `json:"cogs"`              // Cost of Goods Sold
	GrossProfit    decimal.Decimal `json:"gross_profit"`      // NetSalesRevenue - COGS
	GrossMargin    decimal.Decimal `json:"gross_margin"`      // GrossProfit / NetSalesRevenue * 100
	OtherIncome    decimal.Decimal `json:"other_income"`      // Other income
	TotalIncome    decimal.Decimal `json:"total_income"`      // GrossProfit + OtherIncome
	Expenses       decimal.Decimal `json:"expenses"`          // Operating expenses
	NetProfit      decimal.Decimal `json:"net_profit"`        // TotalIncome - Expenses
	NetMargin      decimal.Decimal `json:"net_margin"`        // NetProfit / NetSalesRevenue * 100
}

// ProfitLossLineItem represents a line item in the P&L statement
type ProfitLossLineItem struct {
	Category    string          `json:"category"`    // e.g., "REVENUE", "COGS", "EXPENSE", "OTHER_INCOME"
	SubCategory string          `json:"sub_category,omitempty"`
	Name        string          `json:"name"`
	Amount      decimal.Decimal `json:"amount"`
	Percentage  decimal.Decimal `json:"percentage"` // Percentage of total revenue
}

// ProfitLossDetail provides detailed breakdown of P&L
type ProfitLossDetail struct {
	Statement     ProfitLossStatement  `json:"statement"`
	RevenueItems  []ProfitLossLineItem `json:"revenue_items"`
	COGSItems     []ProfitLossLineItem `json:"cogs_items"`
	ExpenseItems  []ProfitLossLineItem `json:"expense_items"`
	IncomeItems   []ProfitLossLineItem `json:"income_items"`
}

// MonthlyProfitTrend represents monthly profit trend data
type MonthlyProfitTrend struct {
	Year         int             `json:"year"`
	Month        int             `json:"month"`
	SalesRevenue decimal.Decimal `json:"sales_revenue"`
	GrossProfit  decimal.Decimal `json:"gross_profit"`
	NetProfit    decimal.Decimal `json:"net_profit"`
	GrossMargin  decimal.Decimal `json:"gross_margin"`
	NetMargin    decimal.Decimal `json:"net_margin"`
}

// ProfitByProduct represents profit analysis by product
type ProfitByProduct struct {
	ProductID     uuid.UUID       `json:"product_id"`
	ProductSKU    string          `json:"product_sku"`
	ProductName   string          `json:"product_name"`
	CategoryName  string          `json:"category_name,omitempty"`
	SalesRevenue  decimal.Decimal `json:"sales_revenue"`
	COGS          decimal.Decimal `json:"cogs"`
	GrossProfit   decimal.Decimal `json:"gross_profit"`
	GrossMargin   decimal.Decimal `json:"gross_margin"`
	Contribution  decimal.Decimal `json:"contribution"` // Percentage of total profit
}

// ProfitByCustomer represents profit analysis by customer
type ProfitByCustomer struct {
	CustomerID   uuid.UUID       `json:"customer_id"`
	CustomerName string          `json:"customer_name"`
	SalesRevenue decimal.Decimal `json:"sales_revenue"`
	COGS         decimal.Decimal `json:"cogs"`
	GrossProfit  decimal.Decimal `json:"gross_profit"`
	GrossMargin  decimal.Decimal `json:"gross_margin"`
	OrderCount   int64           `json:"order_count"`
}

// CashFlowStatement represents cash flow statement
type CashFlowStatement struct {
	TenantID                 uuid.UUID       `json:"tenant_id"`
	PeriodStart              time.Time       `json:"period_start"`
	PeriodEnd                time.Time       `json:"period_end"`

	// Operating Activities
	ReceiptsFromCustomers    decimal.Decimal `json:"receipts_from_customers"`
	PaymentsToSuppliers      decimal.Decimal `json:"payments_to_suppliers"`
	OtherIncome              decimal.Decimal `json:"other_income"`
	ExpensePayments          decimal.Decimal `json:"expense_payments"`
	NetOperatingCashFlow     decimal.Decimal `json:"net_operating_cash_flow"`

	// Summary
	BeginningCash            decimal.Decimal `json:"beginning_cash"`
	NetCashFlow              decimal.Decimal `json:"net_cash_flow"`
	EndingCash               decimal.Decimal `json:"ending_cash"`
}

// CashFlowItem represents a single cash flow item
type CashFlowItem struct {
	Date          time.Time       `json:"date"`
	Type          string          `json:"type"` // RECEIPT, PAYMENT, EXPENSE, OTHER_INCOME
	ReferenceNo   string          `json:"reference_no"`
	Description   string          `json:"description"`
	Amount        decimal.Decimal `json:"amount"`
	RunningBalance decimal.Decimal `json:"running_balance,omitempty"`
}

// FinanceReportFilter defines filtering options for finance reports
type FinanceReportFilter struct {
	TenantID   uuid.UUID  `json:"-"`
	StartDate  time.Time  `json:"start_date"`
	EndDate    time.Time  `json:"end_date"`
	ProductID  *uuid.UUID `json:"product_id,omitempty"`
	CustomerID *uuid.UUID `json:"customer_id,omitempty"`
	CategoryID *uuid.UUID `json:"category_id,omitempty"`
	TopN       int        `json:"top_n,omitempty"`
}

// FinanceReportRepository defines the interface for finance report queries
type FinanceReportRepository interface {
	// GetProfitLossStatement returns P&L statement for the period
	GetProfitLossStatement(filter FinanceReportFilter) (*ProfitLossStatement, error)

	// GetProfitLossDetail returns detailed P&L breakdown
	GetProfitLossDetail(filter FinanceReportFilter) (*ProfitLossDetail, error)

	// GetMonthlyProfitTrend returns monthly profit trend
	GetMonthlyProfitTrend(filter FinanceReportFilter) ([]MonthlyProfitTrend, error)

	// GetProfitByProduct returns profit analysis by product
	GetProfitByProduct(filter FinanceReportFilter) ([]ProfitByProduct, error)

	// GetProfitByCustomer returns profit analysis by customer
	GetProfitByCustomer(filter FinanceReportFilter) ([]ProfitByCustomer, error)

	// GetCashFlowStatement returns cash flow statement for the period
	GetCashFlowStatement(filter FinanceReportFilter) (*CashFlowStatement, error)

	// GetCashFlowItems returns detailed cash flow items
	GetCashFlowItems(filter FinanceReportFilter) ([]CashFlowItem, error)
}
