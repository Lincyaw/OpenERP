package persistence

import (
	"time"

	"github.com/erp/backend/internal/domain/report"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// GormSalesReportRepository implements SalesReportRepository using GORM
type GormSalesReportRepository struct {
	db *gorm.DB
}

// NewGormSalesReportRepository creates a new GormSalesReportRepository
func NewGormSalesReportRepository(db *gorm.DB) *GormSalesReportRepository {
	return &GormSalesReportRepository{db: db}
}

// GetSalesSummary returns aggregated sales summary for the period
func (r *GormSalesReportRepository) GetSalesSummary(filter report.SalesReportFilter) (*report.SalesSummary, error) {
	type summaryResult struct {
		TotalOrders      int64
		TotalQuantity    decimal.Decimal
		TotalSalesAmount decimal.Decimal
		TotalCostAmount  decimal.Decimal
	}

	var result summaryResult

	query := r.db.Table("sales_orders so").
		Select(`
			COUNT(DISTINCT so.id) as total_orders,
			COALESCE(SUM(soi.quantity), 0) as total_quantity,
			COALESCE(SUM(so.total_amount), 0) as total_sales_amount,
			COALESCE(SUM(soi.amount), 0) as total_cost_amount
		`).
		Joins("LEFT JOIN sales_order_items soi ON soi.order_id = so.id").
		Where("so.tenant_id = ?", filter.TenantID).
		Where("so.created_at BETWEEN ? AND ?", filter.StartDate, filter.EndDate).
		Where("so.status IN ?", []string{"CONFIRMED", "SHIPPED", "COMPLETED"})

	if filter.ProductID != nil {
		query = query.Where("soi.product_id = ?", *filter.ProductID)
	}
	if filter.CustomerID != nil {
		query = query.Where("so.customer_id = ?", *filter.CustomerID)
	}
	if filter.CategoryID != nil {
		query = query.Joins("LEFT JOIN products p ON p.id = soi.product_id").
			Where("p.category_id = ?", *filter.CategoryID)
	}

	if err := query.Scan(&result).Error; err != nil {
		return nil, err
	}

	grossProfit := result.TotalSalesAmount.Sub(result.TotalCostAmount)
	var avgOrderValue, profitMargin decimal.Decimal
	if result.TotalOrders > 0 {
		avgOrderValue = result.TotalSalesAmount.Div(decimal.NewFromInt(result.TotalOrders))
	}
	if !result.TotalSalesAmount.IsZero() {
		profitMargin = grossProfit.Div(result.TotalSalesAmount).Mul(decimal.NewFromInt(100))
	}

	return &report.SalesSummary{
		PeriodStart:      filter.StartDate,
		PeriodEnd:        filter.EndDate,
		TotalOrders:      result.TotalOrders,
		TotalQuantity:    result.TotalQuantity,
		TotalSalesAmount: result.TotalSalesAmount,
		TotalCostAmount:  result.TotalCostAmount,
		TotalGrossProfit: grossProfit,
		AvgOrderValue:    avgOrderValue,
		ProfitMargin:     profitMargin,
	}, nil
}

// GetDailySalesTrend returns daily sales trend data
func (r *GormSalesReportRepository) GetDailySalesTrend(filter report.SalesReportFilter) ([]report.DailySalesTrend, error) {
	type dailyResult struct {
		Date        time.Time
		OrderCount  int64
		TotalAmount decimal.Decimal
		TotalCost   decimal.Decimal
		ItemsSold   decimal.Decimal
	}

	var results []dailyResult

	err := r.db.Table("sales_orders so").
		Select(`
			DATE(so.created_at) as date,
			COUNT(DISTINCT so.id) as order_count,
			COALESCE(SUM(so.total_amount), 0) as total_amount,
			COALESCE(SUM(soi.amount), 0) as total_cost,
			COALESCE(SUM(soi.quantity), 0) as items_sold
		`).
		Joins("LEFT JOIN sales_order_items soi ON soi.order_id = so.id").
		Where("so.tenant_id = ?", filter.TenantID).
		Where("so.created_at BETWEEN ? AND ?", filter.StartDate, filter.EndDate).
		Where("so.status IN ?", []string{"CONFIRMED", "SHIPPED", "COMPLETED"}).
		Group("DATE(so.created_at)").
		Order("date ASC").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	trends := make([]report.DailySalesTrend, len(results))
	for i, r := range results {
		trends[i] = report.DailySalesTrend{
			Date:        r.Date,
			OrderCount:  r.OrderCount,
			TotalAmount: r.TotalAmount,
			TotalProfit: r.TotalAmount.Sub(r.TotalCost),
			ItemsSold:   r.ItemsSold,
		}
	}

	return trends, nil
}

// GetProductSalesReport returns sales report grouped by product
func (r *GormSalesReportRepository) GetProductSalesReport(filter report.SalesReportFilter) ([]report.SalesReport, error) {
	type productResult struct {
		ProductID     uuid.UUID
		ProductSKU    string
		ProductName   string
		CategoryID    *uuid.UUID
		CategoryName  string
		SalesQuantity decimal.Decimal
		SalesAmount   decimal.Decimal
		CostAmount    decimal.Decimal
	}

	var results []productResult

	query := r.db.Table("sales_order_items soi").
		Select(`
			soi.product_id,
			p.sku as product_sku,
			p.name as product_name,
			p.category_id,
			COALESCE(c.name, '') as category_name,
			COALESCE(SUM(soi.quantity), 0) as sales_quantity,
			COALESCE(SUM(soi.amount), 0) as sales_amount,
			COALESCE(SUM(soi.amount), 0) as cost_amount
		`).
		Joins("JOIN sales_orders so ON so.id = soi.order_id").
		Joins("JOIN products p ON p.id = soi.product_id").
		Joins("LEFT JOIN categories c ON c.id = p.category_id").
		Where("so.tenant_id = ?", filter.TenantID).
		Where("so.created_at BETWEEN ? AND ?", filter.StartDate, filter.EndDate).
		Where("so.status IN ?", []string{"CONFIRMED", "SHIPPED", "COMPLETED"}).
		Group("soi.product_id, p.sku, p.name, p.category_id, c.name").
		Order("sales_amount DESC")

	if filter.ProductID != nil {
		query = query.Where("soi.product_id = ?", *filter.ProductID)
	}
	if filter.CategoryID != nil {
		query = query.Where("p.category_id = ?", *filter.CategoryID)
	}

	if err := query.Scan(&results).Error; err != nil {
		return nil, err
	}

	reports := make([]report.SalesReport, len(results))
	for i, r := range results {
		grossProfit := r.SalesAmount.Sub(r.CostAmount)
		var profitMargin decimal.Decimal
		if !r.SalesAmount.IsZero() {
			profitMargin = grossProfit.Div(r.SalesAmount).Mul(decimal.NewFromInt(100))
		}

		reports[i] = report.SalesReport{
			Date:          filter.StartDate,
			ProductID:     r.ProductID,
			ProductSKU:    r.ProductSKU,
			ProductName:   r.ProductName,
			CategoryID:    r.CategoryID,
			CategoryName:  r.CategoryName,
			SalesQuantity: r.SalesQuantity,
			SalesAmount:   r.SalesAmount,
			CostAmount:    r.CostAmount,
			GrossProfit:   grossProfit,
			ProfitMargin:  profitMargin,
		}
	}

	return reports, nil
}

// GetProductSalesRanking returns top N products by sales
func (r *GormSalesReportRepository) GetProductSalesRanking(filter report.SalesReportFilter) ([]report.ProductSalesRanking, error) {
	type rankingResult struct {
		ProductID     uuid.UUID
		ProductSKU    string
		ProductName   string
		CategoryName  string
		TotalQuantity decimal.Decimal
		TotalAmount   decimal.Decimal
		TotalCost     decimal.Decimal
		OrderCount    int64
	}

	var results []rankingResult

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
			COALESCE(SUM(soi.quantity), 0) as total_quantity,
			COALESCE(SUM(soi.amount), 0) as total_amount,
			COALESCE(SUM(soi.amount), 0) as total_cost,
			COUNT(DISTINCT so.id) as order_count
		`).
		Joins("JOIN sales_orders so ON so.id = soi.order_id").
		Joins("JOIN products p ON p.id = soi.product_id").
		Joins("LEFT JOIN categories c ON c.id = p.category_id").
		Where("so.tenant_id = ?", filter.TenantID).
		Where("so.created_at BETWEEN ? AND ?", filter.StartDate, filter.EndDate).
		Where("so.status IN ?", []string{"CONFIRMED", "SHIPPED", "COMPLETED"}).
		Group("soi.product_id, p.sku, p.name, c.name").
		Order("total_amount DESC").
		Limit(topN)

	if filter.CategoryID != nil {
		query = query.Where("p.category_id = ?", *filter.CategoryID)
	}

	if err := query.Scan(&results).Error; err != nil {
		return nil, err
	}

	rankings := make([]report.ProductSalesRanking, len(results))
	for i, r := range results {
		rankings[i] = report.ProductSalesRanking{
			Rank:          i + 1,
			ProductID:     r.ProductID,
			ProductSKU:    r.ProductSKU,
			ProductName:   r.ProductName,
			CategoryName:  r.CategoryName,
			TotalQuantity: r.TotalQuantity,
			TotalAmount:   r.TotalAmount,
			TotalProfit:   r.TotalAmount.Sub(r.TotalCost),
			OrderCount:    r.OrderCount,
		}
	}

	return rankings, nil
}

// GetCustomerSalesRanking returns top N customers by sales
func (r *GormSalesReportRepository) GetCustomerSalesRanking(filter report.SalesReportFilter) ([]report.CustomerSalesRanking, error) {
	type rankingResult struct {
		CustomerID    uuid.UUID
		CustomerName  string
		TotalOrders   int64
		TotalQuantity decimal.Decimal
		TotalAmount   decimal.Decimal
		TotalCost     decimal.Decimal
	}

	var results []rankingResult

	topN := filter.TopN
	if topN <= 0 {
		topN = 10
	}

	err := r.db.Table("sales_orders so").
		Select(`
			so.customer_id,
			so.customer_name,
			COUNT(DISTINCT so.id) as total_orders,
			COALESCE(SUM(soi.quantity), 0) as total_quantity,
			COALESCE(SUM(so.total_amount), 0) as total_amount,
			COALESCE(SUM(soi.amount), 0) as total_cost
		`).
		Joins("LEFT JOIN sales_order_items soi ON soi.order_id = so.id").
		Where("so.tenant_id = ?", filter.TenantID).
		Where("so.created_at BETWEEN ? AND ?", filter.StartDate, filter.EndDate).
		Where("so.status IN ?", []string{"CONFIRMED", "SHIPPED", "COMPLETED"}).
		Group("so.customer_id, so.customer_name").
		Order("total_amount DESC").
		Limit(topN).
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	rankings := make([]report.CustomerSalesRanking, len(results))
	for i, r := range results {
		rankings[i] = report.CustomerSalesRanking{
			Rank:          i + 1,
			CustomerID:    r.CustomerID,
			CustomerName:  r.CustomerName,
			TotalOrders:   r.TotalOrders,
			TotalQuantity: r.TotalQuantity,
			TotalAmount:   r.TotalAmount,
			TotalProfit:   r.TotalAmount.Sub(r.TotalCost),
		}
	}

	return rankings, nil
}
