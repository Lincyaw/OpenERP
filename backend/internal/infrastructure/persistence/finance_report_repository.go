package persistence

import (
	"time"

	"github.com/erp/backend/internal/domain/report"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// GormFinanceReportRepository implements FinanceReportRepository using GORM
type GormFinanceReportRepository struct {
	db *gorm.DB
}

// NewGormFinanceReportRepository creates a new GormFinanceReportRepository
func NewGormFinanceReportRepository(db *gorm.DB) *GormFinanceReportRepository {
	return &GormFinanceReportRepository{db: db}
}

// GetProfitLossStatement returns P&L statement for the period
func (r *GormFinanceReportRepository) GetProfitLossStatement(filter report.FinanceReportFilter) (*report.ProfitLossStatement, error) {
	// Get sales revenue from completed sales orders
	var salesRevenue decimal.Decimal
	if err := r.db.Table("sales_orders so").
		Select("COALESCE(SUM(so.total_amount), 0)").
		Where("so.tenant_id = ?", filter.TenantID).
		Where("so.order_date BETWEEN ? AND ?", filter.StartDate, filter.EndDate).
		Where("so.status IN ?", []string{"SHIPPED", "COMPLETED"}).
		Scan(&salesRevenue).Error; err != nil {
		return nil, err
	}

	// Get COGS (cost of goods sold)
	var cogs decimal.Decimal
	if err := r.db.Table("sales_order_items soi").
		Select("COALESCE(SUM(soi.quantity * soi.unit_cost), 0)").
		Joins("JOIN sales_orders so ON so.id = soi.order_id").
		Where("so.tenant_id = ?", filter.TenantID).
		Where("so.order_date BETWEEN ? AND ?", filter.StartDate, filter.EndDate).
		Where("so.status IN ?", []string{"SHIPPED", "COMPLETED"}).
		Scan(&cogs).Error; err != nil {
		return nil, err
	}

	// Get sales returns (placeholder - would need returns table)
	salesReturns := decimal.Zero

	// Get other income
	var otherIncome decimal.Decimal
	if err := r.db.Table("other_income_records").
		Select("COALESCE(SUM(amount), 0)").
		Where("tenant_id = ?", filter.TenantID).
		Where("income_date BETWEEN ? AND ?", filter.StartDate, filter.EndDate).
		Scan(&otherIncome).Error; err != nil {
		// Table might not exist, default to zero
		otherIncome = decimal.Zero
	}

	// Get expenses
	var expenses decimal.Decimal
	if err := r.db.Table("expense_records").
		Select("COALESCE(SUM(amount), 0)").
		Where("tenant_id = ?", filter.TenantID).
		Where("expense_date BETWEEN ? AND ?", filter.StartDate, filter.EndDate).
		Scan(&expenses).Error; err != nil {
		// Table might not exist, default to zero
		expenses = decimal.Zero
	}

	// Calculate derived values
	netSalesRevenue := salesRevenue.Sub(salesReturns)
	grossProfit := netSalesRevenue.Sub(cogs)
	totalIncome := grossProfit.Add(otherIncome)
	netProfit := totalIncome.Sub(expenses)

	var grossMargin, netMargin decimal.Decimal
	if !netSalesRevenue.IsZero() {
		grossMargin = grossProfit.Div(netSalesRevenue).Mul(decimal.NewFromInt(100))
		netMargin = netProfit.Div(netSalesRevenue).Mul(decimal.NewFromInt(100))
	}

	return &report.ProfitLossStatement{
		TenantID:        filter.TenantID,
		PeriodStart:     filter.StartDate,
		PeriodEnd:       filter.EndDate,
		SalesRevenue:    salesRevenue,
		SalesReturns:    salesReturns,
		NetSalesRevenue: netSalesRevenue,
		COGS:            cogs,
		GrossProfit:     grossProfit,
		GrossMargin:     grossMargin,
		OtherIncome:     otherIncome,
		TotalIncome:     totalIncome,
		Expenses:        expenses,
		NetProfit:       netProfit,
		NetMargin:       netMargin,
	}, nil
}

// GetProfitLossDetail returns detailed P&L breakdown
func (r *GormFinanceReportRepository) GetProfitLossDetail(filter report.FinanceReportFilter) (*report.ProfitLossDetail, error) {
	statement, err := r.GetProfitLossStatement(filter)
	if err != nil {
		return nil, err
	}

	// Revenue items
	revenueItems := []report.ProfitLossLineItem{
		{
			Category: "REVENUE",
			Name:     "Sales Revenue",
			Amount:   statement.SalesRevenue,
		},
		{
			Category: "REVENUE",
			Name:     "Sales Returns",
			Amount:   statement.SalesReturns.Neg(),
		},
	}

	// COGS items - aggregate by category
	type cogsByCategory struct {
		CategoryName string
		Amount       decimal.Decimal
	}
	var cogsItems []cogsByCategory

	r.db.Table("sales_order_items soi").
		Select(`
			COALESCE(c.name, 'Uncategorized') as category_name,
			COALESCE(SUM(soi.quantity * soi.unit_cost), 0) as amount
		`).
		Joins("JOIN sales_orders so ON so.id = soi.order_id").
		Joins("JOIN products p ON p.id = soi.product_id").
		Joins("LEFT JOIN categories c ON c.id = p.category_id").
		Where("so.tenant_id = ?", filter.TenantID).
		Where("so.order_date BETWEEN ? AND ?", filter.StartDate, filter.EndDate).
		Where("so.status IN ?", []string{"SHIPPED", "COMPLETED"}).
		Group("c.name").
		Order("amount DESC").
		Scan(&cogsItems)

	cogsLineItems := make([]report.ProfitLossLineItem, len(cogsItems))
	for i, item := range cogsItems {
		cogsLineItems[i] = report.ProfitLossLineItem{
			Category:    "COGS",
			SubCategory: item.CategoryName,
			Name:        "Cost of " + item.CategoryName,
			Amount:      item.Amount,
		}
	}

	// Expense items - aggregate by category
	type expenseByCategory struct {
		Category string
		Amount   decimal.Decimal
	}
	var expenseItems []expenseByCategory

	r.db.Table("expense_records").
		Select(`
			category,
			COALESCE(SUM(amount), 0) as amount
		`).
		Where("tenant_id = ?", filter.TenantID).
		Where("expense_date BETWEEN ? AND ?", filter.StartDate, filter.EndDate).
		Group("category").
		Order("amount DESC").
		Scan(&expenseItems)

	expenseLineItems := make([]report.ProfitLossLineItem, len(expenseItems))
	for i, item := range expenseItems {
		expenseLineItems[i] = report.ProfitLossLineItem{
			Category:    "EXPENSE",
			SubCategory: item.Category,
			Name:        item.Category,
			Amount:      item.Amount,
		}
	}

	// Income items - aggregate by category
	type incomeByCategory struct {
		Category string
		Amount   decimal.Decimal
	}
	var incomeItems []incomeByCategory

	r.db.Table("other_income_records").
		Select(`
			category,
			COALESCE(SUM(amount), 0) as amount
		`).
		Where("tenant_id = ?", filter.TenantID).
		Where("income_date BETWEEN ? AND ?", filter.StartDate, filter.EndDate).
		Group("category").
		Order("amount DESC").
		Scan(&incomeItems)

	incomeLineItems := make([]report.ProfitLossLineItem, len(incomeItems))
	for i, item := range incomeItems {
		incomeLineItems[i] = report.ProfitLossLineItem{
			Category:    "OTHER_INCOME",
			SubCategory: item.Category,
			Name:        item.Category,
			Amount:      item.Amount,
		}
	}

	return &report.ProfitLossDetail{
		Statement:    *statement,
		RevenueItems: revenueItems,
		COGSItems:    cogsLineItems,
		ExpenseItems: expenseLineItems,
		IncomeItems:  incomeLineItems,
	}, nil
}

// GetMonthlyProfitTrend returns monthly profit trend
func (r *GormFinanceReportRepository) GetMonthlyProfitTrend(filter report.FinanceReportFilter) ([]report.MonthlyProfitTrend, error) {
	type monthlyData struct {
		Year         int
		Month        int
		SalesRevenue decimal.Decimal
		COGS         decimal.Decimal
	}

	var salesData []monthlyData

	// Get monthly sales and COGS
	err := r.db.Table("sales_orders so").
		Select(`
			EXTRACT(YEAR FROM so.order_date)::int as year,
			EXTRACT(MONTH FROM so.order_date)::int as month,
			COALESCE(SUM(so.total_amount), 0) as sales_revenue,
			COALESCE(SUM(soi.quantity * soi.unit_cost), 0) as cogs
		`).
		Joins("LEFT JOIN sales_order_items soi ON soi.order_id = so.id").
		Where("so.tenant_id = ?", filter.TenantID).
		Where("so.order_date BETWEEN ? AND ?", filter.StartDate, filter.EndDate).
		Where("so.status IN ?", []string{"SHIPPED", "COMPLETED"}).
		Group("EXTRACT(YEAR FROM so.order_date), EXTRACT(MONTH FROM so.order_date)").
		Order("year ASC, month ASC").
		Scan(&salesData).Error

	if err != nil {
		return nil, err
	}

	// Get monthly expenses
	type monthlyExpense struct {
		Year   int
		Month  int
		Amount decimal.Decimal
	}
	var expenseData []monthlyExpense

	r.db.Table("expense_records").
		Select(`
			EXTRACT(YEAR FROM expense_date)::int as year,
			EXTRACT(MONTH FROM expense_date)::int as month,
			COALESCE(SUM(amount), 0) as amount
		`).
		Where("tenant_id = ?", filter.TenantID).
		Where("expense_date BETWEEN ? AND ?", filter.StartDate, filter.EndDate).
		Group("EXTRACT(YEAR FROM expense_date), EXTRACT(MONTH FROM expense_date)").
		Scan(&expenseData)

	// Get monthly other income
	var incomeData []monthlyExpense
	r.db.Table("other_income_records").
		Select(`
			EXTRACT(YEAR FROM income_date)::int as year,
			EXTRACT(MONTH FROM income_date)::int as month,
			COALESCE(SUM(amount), 0) as amount
		`).
		Where("tenant_id = ?", filter.TenantID).
		Where("income_date BETWEEN ? AND ?", filter.StartDate, filter.EndDate).
		Group("EXTRACT(YEAR FROM income_date), EXTRACT(MONTH FROM income_date)").
		Scan(&incomeData)

	// Build lookup maps
	expenseMap := make(map[string]decimal.Decimal)
	for _, e := range expenseData {
		key := monthKey(e.Year, e.Month)
		expenseMap[key] = e.Amount
	}

	incomeMap := make(map[string]decimal.Decimal)
	for _, i := range incomeData {
		key := monthKey(i.Year, i.Month)
		incomeMap[key] = i.Amount
	}

	// Combine data
	trends := make([]report.MonthlyProfitTrend, len(salesData))
	for i, s := range salesData {
		key := monthKey(s.Year, s.Month)
		expenses := expenseMap[key]
		otherIncome := incomeMap[key]

		grossProfit := s.SalesRevenue.Sub(s.COGS)
		netProfit := grossProfit.Add(otherIncome).Sub(expenses)

		var grossMargin, netMargin decimal.Decimal
		if !s.SalesRevenue.IsZero() {
			grossMargin = grossProfit.Div(s.SalesRevenue).Mul(decimal.NewFromInt(100))
			netMargin = netProfit.Div(s.SalesRevenue).Mul(decimal.NewFromInt(100))
		}

		trends[i] = report.MonthlyProfitTrend{
			Year:         s.Year,
			Month:        s.Month,
			SalesRevenue: s.SalesRevenue,
			GrossProfit:  grossProfit,
			NetProfit:    netProfit,
			GrossMargin:  grossMargin,
			NetMargin:    netMargin,
		}
	}

	return trends, nil
}

// GetProfitByProduct returns profit analysis by product
func (r *GormFinanceReportRepository) GetProfitByProduct(filter report.FinanceReportFilter) ([]report.ProfitByProduct, error) {
	type productProfit struct {
		ProductID    uuid.UUID
		ProductSKU   string
		ProductName  string
		CategoryName string
		SalesRevenue decimal.Decimal
		COGS         decimal.Decimal
	}

	var results []productProfit

	topN := filter.TopN
	if topN <= 0 {
		topN = 10
	}

	query := r.db.Table("sales_order_items soi").
		Select(`
			soi.product_id,
			p.sku as product_sku,
			p.name as product_name,
			COALESCE(c.name, '') as category_name,
			COALESCE(SUM(soi.subtotal), 0) as sales_revenue,
			COALESCE(SUM(soi.quantity * soi.unit_cost), 0) as cogs
		`).
		Joins("JOIN sales_orders so ON so.id = soi.order_id").
		Joins("JOIN products p ON p.id = soi.product_id").
		Joins("LEFT JOIN categories c ON c.id = p.category_id").
		Where("so.tenant_id = ?", filter.TenantID).
		Where("so.order_date BETWEEN ? AND ?", filter.StartDate, filter.EndDate).
		Where("so.status IN ?", []string{"SHIPPED", "COMPLETED"}).
		Group("soi.product_id, p.sku, p.name, c.name").
		Order("(COALESCE(SUM(soi.subtotal), 0) - COALESCE(SUM(soi.quantity * soi.unit_cost), 0)) DESC").
		Limit(topN)

	if filter.CategoryID != nil {
		query = query.Where("p.category_id = ?", *filter.CategoryID)
	}

	if err := query.Scan(&results).Error; err != nil {
		return nil, err
	}

	// Calculate total profit for contribution percentage
	var totalProfit decimal.Decimal
	for _, r := range results {
		totalProfit = totalProfit.Add(r.SalesRevenue.Sub(r.COGS))
	}

	profits := make([]report.ProfitByProduct, len(results))
	for i, r := range results {
		grossProfit := r.SalesRevenue.Sub(r.COGS)

		var grossMargin, contribution decimal.Decimal
		if !r.SalesRevenue.IsZero() {
			grossMargin = grossProfit.Div(r.SalesRevenue).Mul(decimal.NewFromInt(100))
		}
		if !totalProfit.IsZero() {
			contribution = grossProfit.Div(totalProfit).Mul(decimal.NewFromInt(100))
		}

		profits[i] = report.ProfitByProduct{
			ProductID:    r.ProductID,
			ProductSKU:   r.ProductSKU,
			ProductName:  r.ProductName,
			CategoryName: r.CategoryName,
			SalesRevenue: r.SalesRevenue,
			COGS:         r.COGS,
			GrossProfit:  grossProfit,
			GrossMargin:  grossMargin,
			Contribution: contribution,
		}
	}

	return profits, nil
}

// GetProfitByCustomer returns profit analysis by customer
func (r *GormFinanceReportRepository) GetProfitByCustomer(filter report.FinanceReportFilter) ([]report.ProfitByCustomer, error) {
	type customerProfit struct {
		CustomerID   uuid.UUID
		CustomerName string
		SalesRevenue decimal.Decimal
		COGS         decimal.Decimal
		OrderCount   int64
	}

	var results []customerProfit

	topN := filter.TopN
	if topN <= 0 {
		topN = 10
	}

	err := r.db.Table("sales_orders so").
		Select(`
			so.customer_id,
			so.customer_name,
			COALESCE(SUM(so.total_amount), 0) as sales_revenue,
			COALESCE(SUM(soi.quantity * soi.unit_cost), 0) as cogs,
			COUNT(DISTINCT so.id) as order_count
		`).
		Joins("LEFT JOIN sales_order_items soi ON soi.order_id = so.id").
		Where("so.tenant_id = ?", filter.TenantID).
		Where("so.order_date BETWEEN ? AND ?", filter.StartDate, filter.EndDate).
		Where("so.status IN ?", []string{"SHIPPED", "COMPLETED"}).
		Group("so.customer_id, so.customer_name").
		Order("(COALESCE(SUM(so.total_amount), 0) - COALESCE(SUM(soi.quantity * soi.unit_cost), 0)) DESC").
		Limit(topN).
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	profits := make([]report.ProfitByCustomer, len(results))
	for i, r := range results {
		grossProfit := r.SalesRevenue.Sub(r.COGS)

		var grossMargin decimal.Decimal
		if !r.SalesRevenue.IsZero() {
			grossMargin = grossProfit.Div(r.SalesRevenue).Mul(decimal.NewFromInt(100))
		}

		profits[i] = report.ProfitByCustomer{
			CustomerID:   r.CustomerID,
			CustomerName: r.CustomerName,
			SalesRevenue: r.SalesRevenue,
			COGS:         r.COGS,
			GrossProfit:  grossProfit,
			GrossMargin:  grossMargin,
			OrderCount:   r.OrderCount,
		}
	}

	return profits, nil
}

// GetCashFlowStatement returns cash flow statement for the period
func (r *GormFinanceReportRepository) GetCashFlowStatement(filter report.FinanceReportFilter) (*report.CashFlowStatement, error) {
	// Get receipts from customers
	var receiptsFromCustomers decimal.Decimal
	if err := r.db.Table("receipt_vouchers").
		Select("COALESCE(SUM(amount), 0)").
		Where("tenant_id = ?", filter.TenantID).
		Where("receipt_date BETWEEN ? AND ?", filter.StartDate, filter.EndDate).
		Where("status = ?", "CONFIRMED").
		Scan(&receiptsFromCustomers).Error; err != nil {
		receiptsFromCustomers = decimal.Zero
	}

	// Get payments to suppliers
	var paymentsToSuppliers decimal.Decimal
	if err := r.db.Table("payment_vouchers").
		Select("COALESCE(SUM(amount), 0)").
		Where("tenant_id = ?", filter.TenantID).
		Where("payment_date BETWEEN ? AND ?", filter.StartDate, filter.EndDate).
		Where("status = ?", "CONFIRMED").
		Scan(&paymentsToSuppliers).Error; err != nil {
		paymentsToSuppliers = decimal.Zero
	}

	// Get other income
	var otherIncome decimal.Decimal
	if err := r.db.Table("other_income_records").
		Select("COALESCE(SUM(amount), 0)").
		Where("tenant_id = ?", filter.TenantID).
		Where("income_date BETWEEN ? AND ?", filter.StartDate, filter.EndDate).
		Scan(&otherIncome).Error; err != nil {
		otherIncome = decimal.Zero
	}

	// Get expense payments
	var expensePayments decimal.Decimal
	if err := r.db.Table("expense_records").
		Select("COALESCE(SUM(amount), 0)").
		Where("tenant_id = ?", filter.TenantID).
		Where("expense_date BETWEEN ? AND ?", filter.StartDate, filter.EndDate).
		Scan(&expensePayments).Error; err != nil {
		expensePayments = decimal.Zero
	}

	// Calculate net operating cash flow
	netOperatingCashFlow := receiptsFromCustomers.
		Add(otherIncome).
		Sub(paymentsToSuppliers).
		Sub(expensePayments)

	// Beginning and ending cash would need a separate cash account tracking
	// For now, use net cash flow as the summary
	netCashFlow := netOperatingCashFlow

	return &report.CashFlowStatement{
		TenantID:              filter.TenantID,
		PeriodStart:           filter.StartDate,
		PeriodEnd:             filter.EndDate,
		ReceiptsFromCustomers: receiptsFromCustomers,
		PaymentsToSuppliers:   paymentsToSuppliers,
		OtherIncome:           otherIncome,
		ExpensePayments:       expensePayments,
		NetOperatingCashFlow:  netOperatingCashFlow,
		BeginningCash:         decimal.Zero, // Would need cash account
		NetCashFlow:           netCashFlow,
		EndingCash:            decimal.Zero, // Would need cash account
	}, nil
}

// GetCashFlowItems returns detailed cash flow items
func (r *GormFinanceReportRepository) GetCashFlowItems(filter report.FinanceReportFilter) ([]report.CashFlowItem, error) {
	var items []report.CashFlowItem

	// Get receipt vouchers
	type receiptItem struct {
		Date        time.Time
		ReferenceNo string
		Description string
		Amount      decimal.Decimal
	}
	var receipts []receiptItem

	r.db.Table("receipt_vouchers").
		Select(`
			receipt_date as date,
			voucher_number as reference_no,
			CONCAT('Receipt from ', customer_name) as description,
			amount
		`).
		Where("tenant_id = ?", filter.TenantID).
		Where("receipt_date BETWEEN ? AND ?", filter.StartDate, filter.EndDate).
		Where("status = ?", "CONFIRMED").
		Scan(&receipts)

	for _, rec := range receipts {
		items = append(items, report.CashFlowItem{
			Date:        rec.Date,
			Type:        "RECEIPT",
			ReferenceNo: rec.ReferenceNo,
			Description: rec.Description,
			Amount:      rec.Amount,
		})
	}

	// Get payment vouchers
	var payments []receiptItem

	r.db.Table("payment_vouchers").
		Select(`
			payment_date as date,
			voucher_number as reference_no,
			CONCAT('Payment to ', supplier_name) as description,
			amount
		`).
		Where("tenant_id = ?", filter.TenantID).
		Where("payment_date BETWEEN ? AND ?", filter.StartDate, filter.EndDate).
		Where("status = ?", "CONFIRMED").
		Scan(&payments)

	for _, pay := range payments {
		items = append(items, report.CashFlowItem{
			Date:        pay.Date,
			Type:        "PAYMENT",
			ReferenceNo: pay.ReferenceNo,
			Description: pay.Description,
			Amount:      pay.Amount.Neg(), // Negative for outflows
		})
	}

	// Get expenses
	type expenseItem struct {
		Date        time.Time
		ReferenceNo string
		Description string
		Amount      decimal.Decimal
	}
	var expenses []expenseItem

	r.db.Table("expense_records").
		Select(`
			expense_date as date,
			expense_number as reference_no,
			description,
			amount
		`).
		Where("tenant_id = ?", filter.TenantID).
		Where("expense_date BETWEEN ? AND ?", filter.StartDate, filter.EndDate).
		Scan(&expenses)

	for _, exp := range expenses {
		items = append(items, report.CashFlowItem{
			Date:        exp.Date,
			Type:        "EXPENSE",
			ReferenceNo: exp.ReferenceNo,
			Description: exp.Description,
			Amount:      exp.Amount.Neg(), // Negative for outflows
		})
	}

	// Get other income
	var incomes []expenseItem

	r.db.Table("other_income_records").
		Select(`
			income_date as date,
			income_number as reference_no,
			description,
			amount
		`).
		Where("tenant_id = ?", filter.TenantID).
		Where("income_date BETWEEN ? AND ?", filter.StartDate, filter.EndDate).
		Scan(&incomes)

	for _, inc := range incomes {
		items = append(items, report.CashFlowItem{
			Date:        inc.Date,
			Type:        "OTHER_INCOME",
			ReferenceNo: inc.ReferenceNo,
			Description: inc.Description,
			Amount:      inc.Amount,
		})
	}

	// Sort by date
	sortCashFlowItems(items)

	// Calculate running balance
	runningBalance := decimal.Zero
	for i := range items {
		runningBalance = runningBalance.Add(items[i].Amount)
		items[i].RunningBalance = runningBalance
	}

	return items, nil
}

// Helper functions

func monthKey(year, month int) string {
	return time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
}

func sortCashFlowItems(items []report.CashFlowItem) {
	// Simple bubble sort by date (for small datasets)
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].Date.Before(items[i].Date) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}
