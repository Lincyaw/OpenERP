package report

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/report"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ReportService provides application-level report operations
type ReportService struct {
	salesRepo     report.SalesReportRepository
	inventoryRepo report.InventoryReportRepository
	financeRepo   report.FinanceReportRepository
}

// NewReportService creates a new ReportService
func NewReportService(
	salesRepo report.SalesReportRepository,
	inventoryRepo report.InventoryReportRepository,
	financeRepo report.FinanceReportRepository,
) *ReportService {
	return &ReportService{
		salesRepo:     salesRepo,
		inventoryRepo: inventoryRepo,
		financeRepo:   financeRepo,
	}
}

// ===================== Sales Report Operations =====================

// SalesSummaryResponse represents the sales summary response
type SalesSummaryResponse struct {
	PeriodStart      time.Time `json:"period_start"`
	PeriodEnd        time.Time `json:"period_end"`
	TotalOrders      int64     `json:"total_orders"`
	TotalQuantity    float64   `json:"total_quantity"`
	TotalSalesAmount float64   `json:"total_sales_amount"`
	TotalCostAmount  float64   `json:"total_cost_amount"`
	TotalGrossProfit float64   `json:"total_gross_profit"`
	AvgOrderValue    float64   `json:"avg_order_value"`
	ProfitMargin     float64   `json:"profit_margin"`
}

// DailySalesTrendResponse represents daily sales trend data
type DailySalesTrendResponse struct {
	Date        time.Time `json:"date"`
	OrderCount  int64     `json:"order_count"`
	TotalAmount float64   `json:"total_amount"`
	TotalProfit float64   `json:"total_profit"`
	ItemsSold   float64   `json:"items_sold"`
}

// ProductSalesRankingResponse represents product sales ranking
type ProductSalesRankingResponse struct {
	Rank          int     `json:"rank"`
	ProductID     string  `json:"product_id"`
	ProductSKU    string  `json:"product_sku"`
	ProductName   string  `json:"product_name"`
	CategoryName  string  `json:"category_name,omitempty"`
	TotalQuantity float64 `json:"total_quantity"`
	TotalAmount   float64 `json:"total_amount"`
	TotalProfit   float64 `json:"total_profit"`
	OrderCount    int64   `json:"order_count"`
}

// CustomerSalesRankingResponse represents customer sales ranking
type CustomerSalesRankingResponse struct {
	Rank          int     `json:"rank"`
	CustomerID    string  `json:"customer_id"`
	CustomerName  string  `json:"customer_name"`
	TotalOrders   int64   `json:"total_orders"`
	TotalQuantity float64 `json:"total_quantity"`
	TotalAmount   float64 `json:"total_amount"`
	TotalProfit   float64 `json:"total_profit"`
}

// SalesReportFilter defines the request filter for sales reports
type SalesReportFilter struct {
	StartDate  time.Time  `form:"start_date" binding:"required"`
	EndDate    time.Time  `form:"end_date" binding:"required"`
	ProductID  *uuid.UUID `form:"product_id"`
	CategoryID *uuid.UUID `form:"category_id"`
	CustomerID *uuid.UUID `form:"customer_id"`
	TopN       int        `form:"top_n"`
}

// GetSalesSummary returns sales summary for the period
func (s *ReportService) GetSalesSummary(ctx context.Context, tenantID uuid.UUID, filter SalesReportFilter) (*SalesSummaryResponse, error) {
	domainFilter := report.SalesReportFilter{
		TenantID:   tenantID,
		StartDate:  filter.StartDate,
		EndDate:    filter.EndDate,
		ProductID:  filter.ProductID,
		CategoryID: filter.CategoryID,
		CustomerID: filter.CustomerID,
	}

	summary, err := s.salesRepo.GetSalesSummary(domainFilter)
	if err != nil {
		return nil, err
	}

	return &SalesSummaryResponse{
		PeriodStart:      summary.PeriodStart,
		PeriodEnd:        summary.PeriodEnd,
		TotalOrders:      summary.TotalOrders,
		TotalQuantity:    toFloat64(summary.TotalQuantity),
		TotalSalesAmount: toFloat64(summary.TotalSalesAmount),
		TotalCostAmount:  toFloat64(summary.TotalCostAmount),
		TotalGrossProfit: toFloat64(summary.TotalGrossProfit),
		AvgOrderValue:    toFloat64(summary.AvgOrderValue),
		ProfitMargin:     toFloat64(summary.ProfitMargin),
	}, nil
}

// GetDailySalesTrend returns daily sales trend
func (s *ReportService) GetDailySalesTrend(ctx context.Context, tenantID uuid.UUID, filter SalesReportFilter) ([]DailySalesTrendResponse, error) {
	domainFilter := report.SalesReportFilter{
		TenantID:  tenantID,
		StartDate: filter.StartDate,
		EndDate:   filter.EndDate,
	}

	trends, err := s.salesRepo.GetDailySalesTrend(domainFilter)
	if err != nil {
		return nil, err
	}

	responses := make([]DailySalesTrendResponse, len(trends))
	for i, t := range trends {
		responses[i] = DailySalesTrendResponse{
			Date:        t.Date,
			OrderCount:  t.OrderCount,
			TotalAmount: toFloat64(t.TotalAmount),
			TotalProfit: toFloat64(t.TotalProfit),
			ItemsSold:   toFloat64(t.ItemsSold),
		}
	}

	return responses, nil
}

// GetProductSalesRanking returns top products by sales
func (s *ReportService) GetProductSalesRanking(ctx context.Context, tenantID uuid.UUID, filter SalesReportFilter) ([]ProductSalesRankingResponse, error) {
	topN := filter.TopN
	if topN <= 0 {
		topN = 10
	}

	domainFilter := report.SalesReportFilter{
		TenantID:   tenantID,
		StartDate:  filter.StartDate,
		EndDate:    filter.EndDate,
		CategoryID: filter.CategoryID,
		TopN:       topN,
	}

	rankings, err := s.salesRepo.GetProductSalesRanking(domainFilter)
	if err != nil {
		return nil, err
	}

	responses := make([]ProductSalesRankingResponse, len(rankings))
	for i, r := range rankings {
		responses[i] = ProductSalesRankingResponse{
			Rank:          r.Rank,
			ProductID:     r.ProductID.String(),
			ProductSKU:    r.ProductSKU,
			ProductName:   r.ProductName,
			CategoryName:  r.CategoryName,
			TotalQuantity: toFloat64(r.TotalQuantity),
			TotalAmount:   toFloat64(r.TotalAmount),
			TotalProfit:   toFloat64(r.TotalProfit),
			OrderCount:    r.OrderCount,
		}
	}

	return responses, nil
}

// GetCustomerSalesRanking returns top customers by sales
func (s *ReportService) GetCustomerSalesRanking(ctx context.Context, tenantID uuid.UUID, filter SalesReportFilter) ([]CustomerSalesRankingResponse, error) {
	topN := filter.TopN
	if topN <= 0 {
		topN = 10
	}

	domainFilter := report.SalesReportFilter{
		TenantID:  tenantID,
		StartDate: filter.StartDate,
		EndDate:   filter.EndDate,
		TopN:      topN,
	}

	rankings, err := s.salesRepo.GetCustomerSalesRanking(domainFilter)
	if err != nil {
		return nil, err
	}

	responses := make([]CustomerSalesRankingResponse, len(rankings))
	for i, r := range rankings {
		responses[i] = CustomerSalesRankingResponse{
			Rank:          r.Rank,
			CustomerID:    r.CustomerID.String(),
			CustomerName:  r.CustomerName,
			TotalOrders:   r.TotalOrders,
			TotalQuantity: toFloat64(r.TotalQuantity),
			TotalAmount:   toFloat64(r.TotalAmount),
			TotalProfit:   toFloat64(r.TotalProfit),
		}
	}

	return responses, nil
}

// ===================== Inventory Report Operations =====================

// InventorySummaryResponse represents inventory summary
type InventorySummaryResponse struct {
	TotalProducts   int64   `json:"total_products"`
	TotalQuantity   float64 `json:"total_quantity"`
	TotalValue      float64 `json:"total_value"`
	AvgTurnoverRate float64 `json:"avg_turnover_rate"`
	LowStockCount   int64   `json:"low_stock_count"`
	OutOfStockCount int64   `json:"out_of_stock_count"`
	OverstockCount  int64   `json:"overstock_count"`
}

// InventoryTurnoverResponse represents inventory turnover data
type InventoryTurnoverResponse struct {
	ProductID       string  `json:"product_id"`
	ProductSKU      string  `json:"product_sku"`
	ProductName     string  `json:"product_name"`
	CategoryName    string  `json:"category_name,omitempty"`
	WarehouseName   string  `json:"warehouse_name,omitempty"`
	BeginningStock  float64 `json:"beginning_stock"`
	EndingStock     float64 `json:"ending_stock"`
	AverageStock    float64 `json:"average_stock"`
	SoldQuantity    float64 `json:"sold_quantity"`
	TurnoverRate    float64 `json:"turnover_rate"`
	DaysOfInventory float64 `json:"days_of_inventory"`
	StockValue      float64 `json:"stock_value"`
}

// InventoryValueByCategoryResponse represents inventory value by category
type InventoryValueByCategoryResponse struct {
	CategoryID    string  `json:"category_id,omitempty"`
	CategoryName  string  `json:"category_name"`
	ProductCount  int64   `json:"product_count"`
	TotalQuantity float64 `json:"total_quantity"`
	TotalValue    float64 `json:"total_value"`
	Percentage    float64 `json:"percentage"`
}

// InventoryValueByWarehouseResponse represents inventory value by warehouse
type InventoryValueByWarehouseResponse struct {
	WarehouseID   string  `json:"warehouse_id"`
	WarehouseName string  `json:"warehouse_name"`
	ProductCount  int64   `json:"product_count"`
	TotalQuantity float64 `json:"total_quantity"`
	TotalValue    float64 `json:"total_value"`
	Percentage    float64 `json:"percentage"`
}

// InventoryReportFilter defines the request filter for inventory reports
type InventoryReportFilter struct {
	StartDate   time.Time  `form:"start_date" binding:"required"`
	EndDate     time.Time  `form:"end_date" binding:"required"`
	ProductID   *uuid.UUID `form:"product_id"`
	CategoryID  *uuid.UUID `form:"category_id"`
	WarehouseID *uuid.UUID `form:"warehouse_id"`
	TopN        int        `form:"top_n"`
}

// GetInventorySummary returns inventory summary
func (s *ReportService) GetInventorySummary(ctx context.Context, tenantID uuid.UUID, filter InventoryReportFilter) (*InventorySummaryResponse, error) {
	domainFilter := report.InventoryTurnoverFilter{
		TenantID:    tenantID,
		StartDate:   filter.StartDate,
		EndDate:     filter.EndDate,
		ProductID:   filter.ProductID,
		CategoryID:  filter.CategoryID,
		WarehouseID: filter.WarehouseID,
	}

	summary, err := s.inventoryRepo.GetInventorySummary(domainFilter)
	if err != nil {
		return nil, err
	}

	return &InventorySummaryResponse{
		TotalProducts:   summary.TotalProducts,
		TotalQuantity:   toFloat64(summary.TotalQuantity),
		TotalValue:      toFloat64(summary.TotalValue),
		AvgTurnoverRate: toFloat64(summary.AvgTurnoverRate),
		LowStockCount:   summary.LowStockCount,
		OutOfStockCount: summary.OutOfStockCount,
		OverstockCount:  summary.OverstockCount,
	}, nil
}

// GetInventoryTurnover returns inventory turnover data
func (s *ReportService) GetInventoryTurnover(ctx context.Context, tenantID uuid.UUID, filter InventoryReportFilter) ([]InventoryTurnoverResponse, error) {
	domainFilter := report.InventoryTurnoverFilter{
		TenantID:    tenantID,
		StartDate:   filter.StartDate,
		EndDate:     filter.EndDate,
		ProductID:   filter.ProductID,
		CategoryID:  filter.CategoryID,
		WarehouseID: filter.WarehouseID,
	}

	turnovers, err := s.inventoryRepo.GetInventoryTurnover(domainFilter)
	if err != nil {
		return nil, err
	}

	responses := make([]InventoryTurnoverResponse, len(turnovers))
	for i, t := range turnovers {
		responses[i] = InventoryTurnoverResponse{
			ProductID:       t.ProductID.String(),
			ProductSKU:      t.ProductSKU,
			ProductName:     t.ProductName,
			CategoryName:    t.CategoryName,
			WarehouseName:   t.WarehouseName,
			BeginningStock:  toFloat64(t.BeginningStock),
			EndingStock:     toFloat64(t.EndingStock),
			AverageStock:    toFloat64(t.AverageStock),
			SoldQuantity:    toFloat64(t.SoldQuantity),
			TurnoverRate:    toFloat64(t.TurnoverRate),
			DaysOfInventory: toFloat64(t.DaysOfInventory),
			StockValue:      toFloat64(t.StockValue),
		}
	}

	return responses, nil
}

// GetInventoryValueByCategory returns inventory value by category
func (s *ReportService) GetInventoryValueByCategory(ctx context.Context, tenantID uuid.UUID, filter InventoryReportFilter) ([]InventoryValueByCategoryResponse, error) {
	domainFilter := report.InventoryTurnoverFilter{
		TenantID:  tenantID,
		StartDate: filter.StartDate,
		EndDate:   filter.EndDate,
	}

	values, err := s.inventoryRepo.GetInventoryValueByCategory(domainFilter)
	if err != nil {
		return nil, err
	}

	responses := make([]InventoryValueByCategoryResponse, len(values))
	for i, v := range values {
		categoryID := ""
		if v.CategoryID != nil {
			categoryID = v.CategoryID.String()
		}
		responses[i] = InventoryValueByCategoryResponse{
			CategoryID:    categoryID,
			CategoryName:  v.CategoryName,
			ProductCount:  v.ProductCount,
			TotalQuantity: toFloat64(v.TotalQuantity),
			TotalValue:    toFloat64(v.TotalValue),
			Percentage:    toFloat64(v.Percentage),
		}
	}

	return responses, nil
}

// GetInventoryValueByWarehouse returns inventory value by warehouse
func (s *ReportService) GetInventoryValueByWarehouse(ctx context.Context, tenantID uuid.UUID, filter InventoryReportFilter) ([]InventoryValueByWarehouseResponse, error) {
	domainFilter := report.InventoryTurnoverFilter{
		TenantID:  tenantID,
		StartDate: filter.StartDate,
		EndDate:   filter.EndDate,
	}

	values, err := s.inventoryRepo.GetInventoryValueByWarehouse(domainFilter)
	if err != nil {
		return nil, err
	}

	responses := make([]InventoryValueByWarehouseResponse, len(values))
	for i, v := range values {
		responses[i] = InventoryValueByWarehouseResponse{
			WarehouseID:   v.WarehouseID.String(),
			WarehouseName: v.WarehouseName,
			ProductCount:  v.ProductCount,
			TotalQuantity: toFloat64(v.TotalQuantity),
			TotalValue:    toFloat64(v.TotalValue),
			Percentage:    toFloat64(v.Percentage),
		}
	}

	return responses, nil
}

// GetSlowMovingProducts returns slow moving products
func (s *ReportService) GetSlowMovingProducts(ctx context.Context, tenantID uuid.UUID, filter InventoryReportFilter) ([]InventoryTurnoverResponse, error) {
	topN := filter.TopN
	if topN <= 0 {
		topN = 10
	}

	domainFilter := report.InventoryTurnoverFilter{
		TenantID:    tenantID,
		StartDate:   filter.StartDate,
		EndDate:     filter.EndDate,
		WarehouseID: filter.WarehouseID,
		TopN:        topN,
	}

	products, err := s.inventoryRepo.GetSlowMovingProducts(domainFilter)
	if err != nil {
		return nil, err
	}

	responses := make([]InventoryTurnoverResponse, len(products))
	for i, p := range products {
		responses[i] = InventoryTurnoverResponse{
			ProductID:       p.ProductID.String(),
			ProductSKU:      p.ProductSKU,
			ProductName:     p.ProductName,
			CategoryName:    p.CategoryName,
			WarehouseName:   p.WarehouseName,
			BeginningStock:  toFloat64(p.BeginningStock),
			EndingStock:     toFloat64(p.EndingStock),
			AverageStock:    toFloat64(p.AverageStock),
			SoldQuantity:    toFloat64(p.SoldQuantity),
			TurnoverRate:    toFloat64(p.TurnoverRate),
			DaysOfInventory: toFloat64(p.DaysOfInventory),
			StockValue:      toFloat64(p.StockValue),
		}
	}

	return responses, nil
}

// ===================== Finance Report Operations =====================

// ProfitLossStatementResponse represents P&L statement
type ProfitLossStatementResponse struct {
	PeriodStart     time.Time `json:"period_start"`
	PeriodEnd       time.Time `json:"period_end"`
	SalesRevenue    float64   `json:"sales_revenue"`
	SalesReturns    float64   `json:"sales_returns"`
	NetSalesRevenue float64   `json:"net_sales_revenue"`
	COGS            float64   `json:"cogs"`
	GrossProfit     float64   `json:"gross_profit"`
	GrossMargin     float64   `json:"gross_margin"`
	OtherIncome     float64   `json:"other_income"`
	TotalIncome     float64   `json:"total_income"`
	Expenses        float64   `json:"expenses"`
	NetProfit       float64   `json:"net_profit"`
	NetMargin       float64   `json:"net_margin"`
}

// MonthlyProfitTrendResponse represents monthly profit trend
type MonthlyProfitTrendResponse struct {
	Year         int     `json:"year"`
	Month        int     `json:"month"`
	SalesRevenue float64 `json:"sales_revenue"`
	GrossProfit  float64 `json:"gross_profit"`
	NetProfit    float64 `json:"net_profit"`
	GrossMargin  float64 `json:"gross_margin"`
	NetMargin    float64 `json:"net_margin"`
}

// ProfitByProductResponse represents profit by product
type ProfitByProductResponse struct {
	ProductID    string  `json:"product_id"`
	ProductSKU   string  `json:"product_sku"`
	ProductName  string  `json:"product_name"`
	CategoryName string  `json:"category_name,omitempty"`
	SalesRevenue float64 `json:"sales_revenue"`
	COGS         float64 `json:"cogs"`
	GrossProfit  float64 `json:"gross_profit"`
	GrossMargin  float64 `json:"gross_margin"`
	Contribution float64 `json:"contribution"`
}

// CashFlowStatementResponse represents cash flow statement
type CashFlowStatementResponse struct {
	PeriodStart              time.Time `json:"period_start"`
	PeriodEnd                time.Time `json:"period_end"`
	ReceiptsFromCustomers    float64   `json:"receipts_from_customers"`
	PaymentsToSuppliers      float64   `json:"payments_to_suppliers"`
	OtherIncome              float64   `json:"other_income"`
	ExpensePayments          float64   `json:"expense_payments"`
	NetOperatingCashFlow     float64   `json:"net_operating_cash_flow"`
	BeginningCash            float64   `json:"beginning_cash"`
	NetCashFlow              float64   `json:"net_cash_flow"`
	EndingCash               float64   `json:"ending_cash"`
}

// CashFlowItemResponse represents a cash flow item
type CashFlowItemResponse struct {
	Date           time.Time `json:"date"`
	Type           string    `json:"type"`
	ReferenceNo    string    `json:"reference_no"`
	Description    string    `json:"description"`
	Amount         float64   `json:"amount"`
	RunningBalance float64   `json:"running_balance,omitempty"`
}

// FinanceReportFilter defines the request filter for finance reports
type FinanceReportFilter struct {
	StartDate  time.Time  `form:"start_date" binding:"required"`
	EndDate    time.Time  `form:"end_date" binding:"required"`
	ProductID  *uuid.UUID `form:"product_id"`
	CustomerID *uuid.UUID `form:"customer_id"`
	CategoryID *uuid.UUID `form:"category_id"`
	TopN       int        `form:"top_n"`
}

// GetProfitLossStatement returns P&L statement
func (s *ReportService) GetProfitLossStatement(ctx context.Context, tenantID uuid.UUID, filter FinanceReportFilter) (*ProfitLossStatementResponse, error) {
	domainFilter := report.FinanceReportFilter{
		TenantID:  tenantID,
		StartDate: filter.StartDate,
		EndDate:   filter.EndDate,
	}

	statement, err := s.financeRepo.GetProfitLossStatement(domainFilter)
	if err != nil {
		return nil, err
	}

	return &ProfitLossStatementResponse{
		PeriodStart:     statement.PeriodStart,
		PeriodEnd:       statement.PeriodEnd,
		SalesRevenue:    toFloat64(statement.SalesRevenue),
		SalesReturns:    toFloat64(statement.SalesReturns),
		NetSalesRevenue: toFloat64(statement.NetSalesRevenue),
		COGS:            toFloat64(statement.COGS),
		GrossProfit:     toFloat64(statement.GrossProfit),
		GrossMargin:     toFloat64(statement.GrossMargin),
		OtherIncome:     toFloat64(statement.OtherIncome),
		TotalIncome:     toFloat64(statement.TotalIncome),
		Expenses:        toFloat64(statement.Expenses),
		NetProfit:       toFloat64(statement.NetProfit),
		NetMargin:       toFloat64(statement.NetMargin),
	}, nil
}

// GetMonthlyProfitTrend returns monthly profit trend
func (s *ReportService) GetMonthlyProfitTrend(ctx context.Context, tenantID uuid.UUID, filter FinanceReportFilter) ([]MonthlyProfitTrendResponse, error) {
	domainFilter := report.FinanceReportFilter{
		TenantID:  tenantID,
		StartDate: filter.StartDate,
		EndDate:   filter.EndDate,
	}

	trends, err := s.financeRepo.GetMonthlyProfitTrend(domainFilter)
	if err != nil {
		return nil, err
	}

	responses := make([]MonthlyProfitTrendResponse, len(trends))
	for i, t := range trends {
		responses[i] = MonthlyProfitTrendResponse{
			Year:         t.Year,
			Month:        t.Month,
			SalesRevenue: toFloat64(t.SalesRevenue),
			GrossProfit:  toFloat64(t.GrossProfit),
			NetProfit:    toFloat64(t.NetProfit),
			GrossMargin:  toFloat64(t.GrossMargin),
			NetMargin:    toFloat64(t.NetMargin),
		}
	}

	return responses, nil
}

// GetProfitByProduct returns profit by product
func (s *ReportService) GetProfitByProduct(ctx context.Context, tenantID uuid.UUID, filter FinanceReportFilter) ([]ProfitByProductResponse, error) {
	topN := filter.TopN
	if topN <= 0 {
		topN = 10
	}

	domainFilter := report.FinanceReportFilter{
		TenantID:   tenantID,
		StartDate:  filter.StartDate,
		EndDate:    filter.EndDate,
		CategoryID: filter.CategoryID,
		TopN:       topN,
	}

	profits, err := s.financeRepo.GetProfitByProduct(domainFilter)
	if err != nil {
		return nil, err
	}

	responses := make([]ProfitByProductResponse, len(profits))
	for i, p := range profits {
		responses[i] = ProfitByProductResponse{
			ProductID:    p.ProductID.String(),
			ProductSKU:   p.ProductSKU,
			ProductName:  p.ProductName,
			CategoryName: p.CategoryName,
			SalesRevenue: toFloat64(p.SalesRevenue),
			COGS:         toFloat64(p.COGS),
			GrossProfit:  toFloat64(p.GrossProfit),
			GrossMargin:  toFloat64(p.GrossMargin),
			Contribution: toFloat64(p.Contribution),
		}
	}

	return responses, nil
}

// GetCashFlowStatement returns cash flow statement
func (s *ReportService) GetCashFlowStatement(ctx context.Context, tenantID uuid.UUID, filter FinanceReportFilter) (*CashFlowStatementResponse, error) {
	domainFilter := report.FinanceReportFilter{
		TenantID:  tenantID,
		StartDate: filter.StartDate,
		EndDate:   filter.EndDate,
	}

	statement, err := s.financeRepo.GetCashFlowStatement(domainFilter)
	if err != nil {
		return nil, err
	}

	return &CashFlowStatementResponse{
		PeriodStart:           statement.PeriodStart,
		PeriodEnd:             statement.PeriodEnd,
		ReceiptsFromCustomers: toFloat64(statement.ReceiptsFromCustomers),
		PaymentsToSuppliers:   toFloat64(statement.PaymentsToSuppliers),
		OtherIncome:           toFloat64(statement.OtherIncome),
		ExpensePayments:       toFloat64(statement.ExpensePayments),
		NetOperatingCashFlow:  toFloat64(statement.NetOperatingCashFlow),
		BeginningCash:         toFloat64(statement.BeginningCash),
		NetCashFlow:           toFloat64(statement.NetCashFlow),
		EndingCash:            toFloat64(statement.EndingCash),
	}, nil
}

// GetCashFlowItems returns cash flow items
func (s *ReportService) GetCashFlowItems(ctx context.Context, tenantID uuid.UUID, filter FinanceReportFilter) ([]CashFlowItemResponse, error) {
	domainFilter := report.FinanceReportFilter{
		TenantID:  tenantID,
		StartDate: filter.StartDate,
		EndDate:   filter.EndDate,
	}

	items, err := s.financeRepo.GetCashFlowItems(domainFilter)
	if err != nil {
		return nil, err
	}

	responses := make([]CashFlowItemResponse, len(items))
	for i, item := range items {
		responses[i] = CashFlowItemResponse{
			Date:           item.Date,
			Type:           item.Type,
			ReferenceNo:    item.ReferenceNo,
			Description:    item.Description,
			Amount:         toFloat64(item.Amount),
			RunningBalance: toFloat64(item.RunningBalance),
		}
	}

	return responses, nil
}

// ===================== Helper Functions =====================

func toFloat64(d decimal.Decimal) float64 {
	f, _ := d.Float64()
	return f
}
