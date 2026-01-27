package handler

import (
	"errors"
	"maps"
	"slices"
	"time"

	reportapp "github.com/erp/backend/internal/application/report"
	"github.com/erp/backend/internal/infrastructure/scheduler"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ReportHandler handles report-related API endpoints
type ReportHandler struct {
	BaseHandler
	reportService      *reportapp.ReportService
	aggregationService *reportapp.ReportAggregationService
	cronScheduler      *scheduler.ReportCronScheduler
}

// NewReportHandler creates a new ReportHandler
func NewReportHandler(reportService *reportapp.ReportService) *ReportHandler {
	return &ReportHandler{
		reportService: reportService,
	}
}

// SetAggregationService sets the aggregation service for manual refresh
func (h *ReportHandler) SetAggregationService(aggService *reportapp.ReportAggregationService) {
	h.aggregationService = aggService
}

// SetCronScheduler sets the cron scheduler for status reporting
func (h *ReportHandler) SetCronScheduler(cronScheduler *scheduler.ReportCronScheduler) {
	h.cronScheduler = cronScheduler
}

// ===================== Request DTOs =====================

// SalesReportFilterRequest defines the filter for sales reports
// @Description Filter for sales report queries
type SalesReportFilterRequest struct {
	StartDate  string `form:"start_date" binding:"required" example:"2026-01-01"`
	EndDate    string `form:"end_date" binding:"required" example:"2026-01-31"`
	ProductID  string `form:"product_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CategoryID string `form:"category_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CustomerID string `form:"customer_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TopN       int    `form:"top_n" example:"10"`
}

// InventoryReportFilterRequest defines the filter for inventory reports
// @Description Filter for inventory report queries
type InventoryReportFilterRequest struct {
	StartDate   string `form:"start_date" binding:"required" example:"2026-01-01"`
	EndDate     string `form:"end_date" binding:"required" example:"2026-01-31"`
	ProductID   string `form:"product_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CategoryID  string `form:"category_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	WarehouseID string `form:"warehouse_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TopN        int    `form:"top_n" example:"10"`
}

// FinanceReportFilterRequest defines the filter for finance reports
// @Description Filter for finance report queries
type FinanceReportFilterRequest struct {
	StartDate  string `form:"start_date" binding:"required" example:"2026-01-01"`
	EndDate    string `form:"end_date" binding:"required" example:"2026-01-31"`
	ProductID  string `form:"product_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CustomerID string `form:"customer_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CategoryID string `form:"category_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TopN       int    `form:"top_n" example:"10"`
}

// ===================== Response DTOs (for Swagger) =====================

// SalesSummaryResponse represents the sales summary response
// @Description Sales summary data for a period
type SalesSummaryResponse struct {
	PeriodStart      string  `json:"period_start" example:"2026-01-01T00:00:00Z"`
	PeriodEnd        string  `json:"period_end" example:"2026-01-31T23:59:59Z"`
	TotalOrders      int64   `json:"total_orders" example:"150"`
	TotalQuantity    float64 `json:"total_quantity" example:"1250.5"`
	TotalSalesAmount float64 `json:"total_sales_amount" example:"125000.00"`
	TotalCostAmount  float64 `json:"total_cost_amount" example:"87500.00"`
	TotalGrossProfit float64 `json:"total_gross_profit" example:"37500.00"`
	AvgOrderValue    float64 `json:"avg_order_value" example:"833.33"`
	ProfitMargin     float64 `json:"profit_margin" example:"30.0"`
}

// DailySalesTrendResponse represents daily sales trend data
// @Description Daily sales trend data point
type DailySalesTrendResponse struct {
	Date        string  `json:"date" example:"2026-01-15"`
	OrderCount  int64   `json:"order_count" example:"25"`
	TotalAmount float64 `json:"total_amount" example:"5000.00"`
	TotalProfit float64 `json:"total_profit" example:"1500.00"`
	ItemsSold   float64 `json:"items_sold" example:"100"`
}

// ProductSalesRankingResponse represents product sales ranking
// @Description Product sales ranking item
type ProductSalesRankingResponse struct {
	Rank          int     `json:"rank" example:"1"`
	ProductID     string  `json:"product_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ProductSKU    string  `json:"product_sku" example:"SKU-001"`
	ProductName   string  `json:"product_name" example:"Sample Product"`
	CategoryName  string  `json:"category_name,omitempty" example:"Electronics"`
	TotalQuantity float64 `json:"total_quantity" example:"500"`
	TotalAmount   float64 `json:"total_amount" example:"25000.00"`
	TotalProfit   float64 `json:"total_profit" example:"7500.00"`
	OrderCount    int64   `json:"order_count" example:"50"`
}

// CustomerSalesRankingResponse represents customer sales ranking
// @Description Customer sales ranking item
type CustomerSalesRankingResponse struct {
	Rank          int     `json:"rank" example:"1"`
	CustomerID    string  `json:"customer_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CustomerName  string  `json:"customer_name" example:"ABC Company"`
	TotalOrders   int64   `json:"total_orders" example:"30"`
	TotalQuantity float64 `json:"total_quantity" example:"200"`
	TotalAmount   float64 `json:"total_amount" example:"15000.00"`
	TotalProfit   float64 `json:"total_profit" example:"4500.00"`
}

// InventorySummaryResponse represents inventory summary
// @Description Inventory summary data
type InventorySummaryResponse struct {
	TotalProducts     int64   `json:"total_products" example:"500"`
	TotalWarehouses   int64   `json:"total_warehouses" example:"5"`
	TotalQuantity     float64 `json:"total_quantity" example:"10000"`
	TotalValue        float64 `json:"total_value" example:"500000.00"`
	AvgValue          float64 `json:"avg_value" example:"1000.00"`
	LowStockCount     int64   `json:"low_stock_count" example:"25"`
	OutOfStockCount   int64   `json:"out_of_stock_count" example:"10"`
	OverstockCount    int64   `json:"overstock_count" example:"15"`
	TurnoverRate      float64 `json:"turnover_rate" example:"4.5"`
	DaysOfStockOnHand float64 `json:"days_of_stock_on_hand" example:"30.5"`
}

// InventoryMovementResponse represents inventory movement data
// @Description Inventory movement summary
type InventoryMovementResponse struct {
	PeriodStart      string  `json:"period_start" example:"2026-01-01T00:00:00Z"`
	PeriodEnd        string  `json:"period_end" example:"2026-01-31T23:59:59Z"`
	TotalIn          float64 `json:"total_in" example:"5000"`
	TotalOut         float64 `json:"total_out" example:"4500"`
	NetChange        float64 `json:"net_change" example:"500"`
	InValue          float64 `json:"in_value" example:"250000.00"`
	OutValue         float64 `json:"out_value" example:"225000.00"`
	TransactionCount int64   `json:"transaction_count" example:"150"`
}

// InventoryTurnoverResponse represents inventory turnover data
// @Description Inventory turnover data
type InventoryTurnoverResponse struct {
	ProductID         string  `json:"product_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ProductName       string  `json:"product_name" example:"Sample Product"`
	CategoryName      string  `json:"category_name,omitempty" example:"Electronics"`
	AvgInventory      float64 `json:"avg_inventory" example:"100"`
	TotalSold         float64 `json:"total_sold" example:"450"`
	TurnoverRate      float64 `json:"turnover_rate" example:"4.5"`
	DaysOfStockOnHand float64 `json:"days_of_stock_on_hand" example:"81.1"`
}

// InventoryValueByCategoryResponse represents inventory value by category
// @Description Inventory value grouped by category
type InventoryValueByCategoryResponse struct {
	CategoryID     string  `json:"category_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CategoryName   string  `json:"category_name" example:"Electronics"`
	TotalQuantity  float64 `json:"total_quantity" example:"2000"`
	TotalValue     float64 `json:"total_value" example:"100000.00"`
	PercentOfTotal float64 `json:"percent_of_total" example:"20.0"`
	ProductCount   int64   `json:"product_count" example:"50"`
}

// ProfitLossResponse represents profit and loss data
// @Description Profit and loss statement data
type ProfitLossResponse struct {
	PeriodStart       string  `json:"period_start" example:"2026-01-01T00:00:00Z"`
	PeriodEnd         string  `json:"period_end" example:"2026-01-31T23:59:59Z"`
	Revenue           float64 `json:"revenue" example:"125000.00"`
	CostOfGoodsSold   float64 `json:"cost_of_goods_sold" example:"87500.00"`
	GrossProfit       float64 `json:"gross_profit" example:"37500.00"`
	GrossProfitMargin float64 `json:"gross_profit_margin" example:"30.0"`
	OperatingExpenses float64 `json:"operating_expenses" example:"15000.00"`
	OtherIncome       float64 `json:"other_income" example:"2000.00"`
	OtherExpenses     float64 `json:"other_expenses" example:"1000.00"`
	NetProfit         float64 `json:"net_profit" example:"23500.00"`
	NetProfitMargin   float64 `json:"net_profit_margin" example:"18.8"`
}

// CashFlowResponse represents cash flow data
// @Description Cash flow data
type CashFlowResponse struct {
	PeriodStart        string  `json:"period_start" example:"2026-01-01T00:00:00Z"`
	PeriodEnd          string  `json:"period_end" example:"2026-01-31T23:59:59Z"`
	CashFromSales      float64 `json:"cash_from_sales" example:"100000.00"`
	CashToPurchases    float64 `json:"cash_to_purchases" example:"70000.00"`
	CashToExpenses     float64 `json:"cash_to_expenses" example:"15000.00"`
	OtherCashInflow    float64 `json:"other_cash_inflow" example:"2000.00"`
	OtherCashOutflow   float64 `json:"other_cash_outflow" example:"1000.00"`
	NetCashFlow        float64 `json:"net_cash_flow" example:"16000.00"`
	OpeningCashBalance float64 `json:"opening_cash_balance" example:"50000.00"`
	ClosingCashBalance float64 `json:"closing_cash_balance" example:"66000.00"`
}

// ReceivableSummaryReportResponse represents receivable summary
// @Description Accounts receivable summary
type ReceivableSummaryReportResponse struct {
	TotalReceivable   float64 `json:"total_receivable" example:"50000.00"`
	CollectedAmount   float64 `json:"collected_amount" example:"35000.00"`
	OutstandingAmount float64 `json:"outstanding_amount" example:"15000.00"`
	OverdueAmount     float64 `json:"overdue_amount" example:"5000.00"`
	CustomerCount     int64   `json:"customer_count" example:"25"`
}

// PayableSummaryReportResponse represents payable summary
// @Description Accounts payable summary
type PayableSummaryReportResponse struct {
	TotalPayable      float64 `json:"total_payable" example:"40000.00"`
	PaidAmount        float64 `json:"paid_amount" example:"30000.00"`
	OutstandingAmount float64 `json:"outstanding_amount" example:"10000.00"`
	OverdueAmount     float64 `json:"overdue_amount" example:"3000.00"`
	SupplierCount     int64   `json:"supplier_count" example:"15"`
}

// CustomerReceivableResponse represents customer receivable data
// @Description Customer receivable data
type CustomerReceivableResponse struct {
	CustomerID        string  `json:"customer_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CustomerName      string  `json:"customer_name" example:"ABC Company"`
	TotalReceivable   float64 `json:"total_receivable" example:"5000.00"`
	CollectedAmount   float64 `json:"collected_amount" example:"3500.00"`
	OutstandingAmount float64 `json:"outstanding_amount" example:"1500.00"`
	OverdueAmount     float64 `json:"overdue_amount" example:"500.00"`
}

// InventoryValueByWarehouseResponse represents inventory value by warehouse
// @Description Inventory value grouped by warehouse
type InventoryValueByWarehouseResponse struct {
	WarehouseID    string  `json:"warehouse_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	WarehouseName  string  `json:"warehouse_name" example:"Main Warehouse"`
	TotalQuantity  float64 `json:"total_quantity" example:"5000"`
	TotalValue     float64 `json:"total_value" example:"250000.00"`
	PercentOfTotal float64 `json:"percent_of_total" example:"50.0"`
	ProductCount   int64   `json:"product_count" example:"100"`
}

// ProfitLossStatementResponse represents profit and loss statement
// @Description Detailed profit and loss statement
type ProfitLossStatementResponse struct {
	PeriodStart     string  `json:"period_start" example:"2026-01-01T00:00:00Z"`
	PeriodEnd       string  `json:"period_end" example:"2026-01-31T23:59:59Z"`
	SalesRevenue    float64 `json:"sales_revenue" example:"125000.00"`
	SalesReturns    float64 `json:"sales_returns" example:"5000.00"`
	NetSalesRevenue float64 `json:"net_sales_revenue" example:"120000.00"`
	COGS            float64 `json:"cogs" example:"84000.00"`
	GrossProfit     float64 `json:"gross_profit" example:"36000.00"`
	GrossMargin     float64 `json:"gross_margin" example:"0.30"`
	OtherIncome     float64 `json:"other_income" example:"2000.00"`
	TotalIncome     float64 `json:"total_income" example:"38000.00"`
	Expenses        float64 `json:"expenses" example:"15000.00"`
	NetProfit       float64 `json:"net_profit" example:"23000.00"`
	NetMargin       float64 `json:"net_margin" example:"0.19"`
}

// MonthlyProfitTrendResponse represents monthly profit trend
// @Description Monthly profit trend data point
type MonthlyProfitTrendResponse struct {
	Year         int     `json:"year" example:"2026"`
	Month        int     `json:"month" example:"1"`
	SalesRevenue float64 `json:"sales_revenue" example:"125000.00"`
	GrossProfit  float64 `json:"gross_profit" example:"37500.00"`
	NetProfit    float64 `json:"net_profit" example:"22500.00"`
	GrossMargin  float64 `json:"gross_margin" example:"0.30"`
	NetMargin    float64 `json:"net_margin" example:"0.18"`
}

// ProfitByProductResponse represents profit by product
// @Description Profit breakdown by product
type ProfitByProductResponse struct {
	ProductID    string  `json:"product_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ProductSKU   string  `json:"product_sku" example:"SKU-001"`
	ProductName  string  `json:"product_name" example:"Sample Product"`
	CategoryName string  `json:"category_name,omitempty" example:"Electronics"`
	SalesRevenue float64 `json:"sales_revenue" example:"25000.00"`
	COGS         float64 `json:"cogs" example:"17500.00"`
	GrossProfit  float64 `json:"gross_profit" example:"7500.00"`
	GrossMargin  float64 `json:"gross_margin" example:"0.30"`
	Contribution float64 `json:"contribution" example:"0.15"`
}

// CashFlowStatementResponse represents cash flow statement
// @Description Cash flow statement
type CashFlowStatementResponse struct {
	PeriodStart           string  `json:"period_start" example:"2026-01-01T00:00:00Z"`
	PeriodEnd             string  `json:"period_end" example:"2026-01-31T23:59:59Z"`
	ReceiptsFromCustomers float64 `json:"receipts_from_customers" example:"100000.00"`
	PaymentsToSuppliers   float64 `json:"payments_to_suppliers" example:"70000.00"`
	OtherIncome           float64 `json:"other_income" example:"2000.00"`
	ExpensePayments       float64 `json:"expense_payments" example:"15000.00"`
	NetOperatingCashFlow  float64 `json:"net_operating_cash_flow" example:"17000.00"`
	BeginningCash         float64 `json:"beginning_cash" example:"50000.00"`
	NetCashFlow           float64 `json:"net_cash_flow" example:"17000.00"`
	EndingCash            float64 `json:"ending_cash" example:"67000.00"`
}

// CashFlowItemResponse represents cash flow item
// @Description Individual cash flow item
type CashFlowItemResponse struct {
	Date           string  `json:"date" example:"2026-01-15T00:00:00Z"`
	Type           string  `json:"type" example:"receipt"`
	ReferenceNo    string  `json:"reference_no" example:"RCV-2026-001"`
	Description    string  `json:"description" example:"Sales collection"`
	Amount         float64 `json:"amount" example:"5000.00"`
	RunningBalance float64 `json:"running_balance,omitempty" example:"55000.00"`
}

// ===================== Sales Report Endpoints =====================

// GetSalesSummary godoc
// @Summary      Get sales summary
// @Description  Get aggregated sales summary for the specified period
// @Tags         reports
// @Accept       json
// @Produce      json
// @Param        start_date query string true "Start date (YYYY-MM-DD)"
// @Param        end_date query string true "End date (YYYY-MM-DD)"
// @Param        product_id query string false "Filter by product ID"
// @Param        category_id query string false "Filter by category ID"
// @Param        customer_id query string false "Filter by customer ID"
// @Success      200 {object} dto.Response{data=SalesSummaryResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Router       /reports/sales/summary [get]
func (h *ReportHandler) GetSalesSummary(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req SalesReportFilterRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	filter, err := h.parseSalesFilter(req)
	if err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	summary, err := h.reportService.GetSalesSummary(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, summary)
}

// GetDailySalesTrend godoc
// @Summary      Get daily sales trend
// @Description  Get daily sales trend data for the specified period
// @Tags         reports
// @Accept       json
// @Produce      json
// @Param        start_date query string true "Start date (YYYY-MM-DD)"
// @Param        end_date query string true "End date (YYYY-MM-DD)"
// @Success      200 {object} dto.Response{data=[]DailySalesTrendResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Router       /reports/sales/daily-trend [get]
func (h *ReportHandler) GetDailySalesTrend(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req SalesReportFilterRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	filter, err := h.parseSalesFilter(req)
	if err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	trends, err := h.reportService.GetDailySalesTrend(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, trends)
}

// GetProductSalesRanking godoc
// @Summary      Get product sales ranking
// @Description  Get top products by sales for the specified period
// @Tags         reports
// @Accept       json
// @Produce      json
// @Param        start_date query string true "Start date (YYYY-MM-DD)"
// @Param        end_date query string true "End date (YYYY-MM-DD)"
// @Param        category_id query string false "Filter by category ID"
// @Param        top_n query int false "Number of top products (default 10)"
// @Success      200 {object} dto.Response{data=[]ProductSalesRankingResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Router       /reports/sales/products/ranking [get]
func (h *ReportHandler) GetProductSalesRanking(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req SalesReportFilterRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	filter, err := h.parseSalesFilter(req)
	if err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	rankings, err := h.reportService.GetProductSalesRanking(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, rankings)
}

// GetCustomerSalesRanking godoc
// @Summary      Get customer sales ranking
// @Description  Get top customers by sales for the specified period
// @Tags         reports
// @Accept       json
// @Produce      json
// @Param        start_date query string true "Start date (YYYY-MM-DD)"
// @Param        end_date query string true "End date (YYYY-MM-DD)"
// @Param        top_n query int false "Number of top customers (default 10)"
// @Success      200 {object} dto.Response{data=[]CustomerSalesRankingResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Router       /reports/sales/customers/ranking [get]
func (h *ReportHandler) GetCustomerSalesRanking(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req SalesReportFilterRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	filter, err := h.parseSalesFilter(req)
	if err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	rankings, err := h.reportService.GetCustomerSalesRanking(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, rankings)
}

// ===================== Inventory Report Endpoints =====================

// GetInventorySummary godoc
// @Summary      Get inventory summary
// @Description  Get aggregated inventory summary
// @Tags         reports
// @Accept       json
// @Produce      json
// @Param        start_date query string true "Start date (YYYY-MM-DD)"
// @Param        end_date query string true "End date (YYYY-MM-DD)"
// @Param        warehouse_id query string false "Filter by warehouse ID"
// @Param        category_id query string false "Filter by category ID"
// @Success      200 {object} dto.Response{data=InventorySummaryResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Router       /reports/inventory/summary [get]
func (h *ReportHandler) GetInventorySummary(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req InventoryReportFilterRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	filter, err := h.parseInventoryFilter(req)
	if err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	summary, err := h.reportService.GetInventorySummary(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, summary)
}

// GetInventoryTurnover godoc
// @Summary      Get inventory turnover
// @Description  Get inventory turnover data for products
// @Tags         reports
// @Accept       json
// @Produce      json
// @Param        start_date query string true "Start date (YYYY-MM-DD)"
// @Param        end_date query string true "End date (YYYY-MM-DD)"
// @Param        warehouse_id query string false "Filter by warehouse ID"
// @Param        category_id query string false "Filter by category ID"
// @Param        product_id query string false "Filter by product ID"
// @Success      200 {object} dto.Response{data=[]InventoryTurnoverResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Router       /reports/inventory/turnover [get]
func (h *ReportHandler) GetInventoryTurnover(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req InventoryReportFilterRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	filter, err := h.parseInventoryFilter(req)
	if err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	turnovers, err := h.reportService.GetInventoryTurnover(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, turnovers)
}

// GetInventoryValueByCategory godoc
// @Summary      Get inventory value by category
// @Description  Get inventory value grouped by category
// @Tags         reports
// @Accept       json
// @Produce      json
// @Param        start_date query string true "Start date (YYYY-MM-DD)"
// @Param        end_date query string true "End date (YYYY-MM-DD)"
// @Success      200 {object} dto.Response{data=[]InventoryValueByCategoryResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Router       /reports/inventory/value-by-category [get]
func (h *ReportHandler) GetInventoryValueByCategory(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req InventoryReportFilterRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	filter, err := h.parseInventoryFilter(req)
	if err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	values, err := h.reportService.GetInventoryValueByCategory(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, values)
}

// GetInventoryValueByWarehouse godoc
// @Summary      Get inventory value by warehouse
// @Description  Get inventory value grouped by warehouse
// @Tags         reports
// @Accept       json
// @Produce      json
// @Param        start_date query string true "Start date (YYYY-MM-DD)"
// @Param        end_date query string true "End date (YYYY-MM-DD)"
// @Success      200 {object} dto.Response{data=[]InventoryValueByWarehouseResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Router       /reports/inventory/value-by-warehouse [get]
func (h *ReportHandler) GetInventoryValueByWarehouse(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req InventoryReportFilterRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	filter, err := h.parseInventoryFilter(req)
	if err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	values, err := h.reportService.GetInventoryValueByWarehouse(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, values)
}

// GetSlowMovingProducts godoc
// @Summary      Get slow moving products
// @Description  Get products with low turnover rate
// @Tags         reports
// @Accept       json
// @Produce      json
// @Param        start_date query string true "Start date (YYYY-MM-DD)"
// @Param        end_date query string true "End date (YYYY-MM-DD)"
// @Param        warehouse_id query string false "Filter by warehouse ID"
// @Param        top_n query int false "Number of products (default 10)"
// @Success      200 {object} dto.Response{data=[]InventoryTurnoverResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Router       /reports/inventory/slow-moving [get]
func (h *ReportHandler) GetSlowMovingProducts(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req InventoryReportFilterRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	filter, err := h.parseInventoryFilter(req)
	if err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	products, err := h.reportService.GetSlowMovingProducts(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, products)
}

// ===================== Finance Report Endpoints =====================

// GetProfitLossStatement godoc
// @Summary      Get profit and loss statement
// @Description  Get P&L statement for the specified period
// @Tags         reports
// @Accept       json
// @Produce      json
// @Param        start_date query string true "Start date (YYYY-MM-DD)"
// @Param        end_date query string true "End date (YYYY-MM-DD)"
// @Success      200 {object} dto.Response{data=ProfitLossStatementResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Router       /reports/finance/profit-loss [get]
func (h *ReportHandler) GetProfitLossStatement(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req FinanceReportFilterRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	filter, err := h.parseFinanceFilter(req)
	if err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	statement, err := h.reportService.GetProfitLossStatement(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, statement)
}

// GetMonthlyProfitTrend godoc
// @Summary      Get monthly profit trend
// @Description  Get monthly profit trend for the specified period
// @Tags         reports
// @Accept       json
// @Produce      json
// @Param        start_date query string true "Start date (YYYY-MM-DD)"
// @Param        end_date query string true "End date (YYYY-MM-DD)"
// @Success      200 {object} dto.Response{data=[]MonthlyProfitTrendResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Router       /reports/finance/monthly-trend [get]
func (h *ReportHandler) GetMonthlyProfitTrend(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req FinanceReportFilterRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	filter, err := h.parseFinanceFilter(req)
	if err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	trends, err := h.reportService.GetMonthlyProfitTrend(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, trends)
}

// GetProfitByProduct godoc
// @Summary      Get profit by product
// @Description  Get profit analysis by product
// @Tags         reports
// @Accept       json
// @Produce      json
// @Param        start_date query string true "Start date (YYYY-MM-DD)"
// @Param        end_date query string true "End date (YYYY-MM-DD)"
// @Param        category_id query string false "Filter by category ID"
// @Param        top_n query int false "Number of products (default 10)"
// @Success      200 {object} dto.Response{data=[]ProfitByProductResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Router       /reports/finance/profit-by-product [get]
func (h *ReportHandler) GetProfitByProduct(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req FinanceReportFilterRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	filter, err := h.parseFinanceFilter(req)
	if err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	profits, err := h.reportService.GetProfitByProduct(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, profits)
}

// GetCashFlowStatement godoc
// @Summary      Get cash flow statement
// @Description  Get cash flow statement for the specified period
// @Tags         reports
// @Accept       json
// @Produce      json
// @Param        start_date query string true "Start date (YYYY-MM-DD)"
// @Param        end_date query string true "End date (YYYY-MM-DD)"
// @Success      200 {object} dto.Response{data=CashFlowStatementResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Router       /reports/finance/cash-flow [get]
func (h *ReportHandler) GetCashFlowStatement(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req FinanceReportFilterRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	filter, err := h.parseFinanceFilter(req)
	if err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	statement, err := h.reportService.GetCashFlowStatement(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, statement)
}

// GetCashFlowItems godoc
// @Summary      Get cash flow items
// @Description  Get detailed cash flow items for the specified period
// @Tags         reports
// @Accept       json
// @Produce      json
// @Param        start_date query string true "Start date (YYYY-MM-DD)"
// @Param        end_date query string true "End date (YYYY-MM-DD)"
// @Success      200 {object} dto.Response{data=[]CashFlowItemResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Router       /reports/finance/cash-flow/items [get]
func (h *ReportHandler) GetCashFlowItems(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req FinanceReportFilterRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	filter, err := h.parseFinanceFilter(req)
	if err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	items, err := h.reportService.GetCashFlowItems(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, items)
}

// ===================== Helper Functions =====================

func (h *ReportHandler) parseSalesFilter(req SalesReportFilterRequest) (reportapp.SalesReportFilter, error) {
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return reportapp.SalesReportFilter{}, errors.New("start_date: Invalid date format, expected YYYY-MM-DD")
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return reportapp.SalesReportFilter{}, errors.New("end_date: Invalid date format, expected YYYY-MM-DD")
	}

	// Set end date to end of day
	endDate = endDate.Add(24*time.Hour - time.Second)

	filter := reportapp.SalesReportFilter{
		StartDate: startDate,
		EndDate:   endDate,
		TopN:      req.TopN,
	}

	if req.ProductID != "" {
		productID, err := uuid.Parse(req.ProductID)
		if err != nil {
			return filter, errors.New("product_id: Invalid UUID format")
		}
		filter.ProductID = &productID
	}

	if req.CategoryID != "" {
		categoryID, err := uuid.Parse(req.CategoryID)
		if err != nil {
			return filter, errors.New("category_id: Invalid UUID format")
		}
		filter.CategoryID = &categoryID
	}

	if req.CustomerID != "" {
		customerID, err := uuid.Parse(req.CustomerID)
		if err != nil {
			return filter, errors.New("customer_id: Invalid UUID format")
		}
		filter.CustomerID = &customerID
	}

	return filter, nil
}

func (h *ReportHandler) parseInventoryFilter(req InventoryReportFilterRequest) (reportapp.InventoryReportFilter, error) {
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return reportapp.InventoryReportFilter{}, errors.New("start_date: Invalid date format, expected YYYY-MM-DD")
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return reportapp.InventoryReportFilter{}, errors.New("end_date: Invalid date format, expected YYYY-MM-DD")
	}

	// Set end date to end of day
	endDate = endDate.Add(24*time.Hour - time.Second)

	filter := reportapp.InventoryReportFilter{
		StartDate: startDate,
		EndDate:   endDate,
		TopN:      req.TopN,
	}

	if req.ProductID != "" {
		productID, err := uuid.Parse(req.ProductID)
		if err != nil {
			return filter, errors.New("product_id: Invalid UUID format")
		}
		filter.ProductID = &productID
	}

	if req.CategoryID != "" {
		categoryID, err := uuid.Parse(req.CategoryID)
		if err != nil {
			return filter, errors.New("category_id: Invalid UUID format")
		}
		filter.CategoryID = &categoryID
	}

	if req.WarehouseID != "" {
		warehouseID, err := uuid.Parse(req.WarehouseID)
		if err != nil {
			return filter, errors.New("warehouse_id: Invalid UUID format")
		}
		filter.WarehouseID = &warehouseID
	}

	return filter, nil
}

func (h *ReportHandler) parseFinanceFilter(req FinanceReportFilterRequest) (reportapp.FinanceReportFilter, error) {
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return reportapp.FinanceReportFilter{}, errors.New("start_date: Invalid date format, expected YYYY-MM-DD")
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return reportapp.FinanceReportFilter{}, errors.New("end_date: Invalid date format, expected YYYY-MM-DD")
	}

	// Set end date to end of day
	endDate = endDate.Add(24*time.Hour - time.Second)

	filter := reportapp.FinanceReportFilter{
		StartDate: startDate,
		EndDate:   endDate,
		TopN:      req.TopN,
	}

	if req.ProductID != "" {
		productID, err := uuid.Parse(req.ProductID)
		if err != nil {
			return filter, errors.New("product_id: Invalid UUID format")
		}
		filter.ProductID = &productID
	}

	if req.CustomerID != "" {
		customerID, err := uuid.Parse(req.CustomerID)
		if err != nil {
			return filter, errors.New("customer_id: Invalid UUID format")
		}
		filter.CustomerID = &customerID
	}

	if req.CategoryID != "" {
		categoryID, err := uuid.Parse(req.CategoryID)
		if err != nil {
			return filter, errors.New("category_id: Invalid UUID format")
		}
		filter.CategoryID = &categoryID
	}

	return filter, nil
}

// ===================== Manual Refresh Endpoints =====================

// RefreshReportRequest defines the request for manual report refresh
// @Description Request for manual report cache refresh
type RefreshReportRequest struct {
	ReportType string `json:"report_type" binding:"required" example:"SALES_SUMMARY"`
	StartDate  string `json:"start_date" binding:"required" example:"2026-01-01"`
	EndDate    string `json:"end_date" binding:"required" example:"2026-01-31"`
}

// RefreshAllReportsRequest defines the request for refreshing all reports
// @Description Request for refreshing all report caches
type RefreshAllReportsRequest struct {
	StartDate string `json:"start_date" binding:"required" example:"2026-01-01"`
	EndDate   string `json:"end_date" binding:"required" example:"2026-01-31"`
}

// RefreshReportResponse represents the response for report refresh
// @Description Response for report refresh operation
type RefreshReportResponse struct {
	Message    string `json:"message" example:"Report refresh scheduled"`
	ReportType string `json:"report_type,omitempty" example:"SALES_SUMMARY"`
}

// RefreshReport godoc
// @Summary      Manually refresh a report cache
// @Description  Triggers manual refresh of a specific report type cache
// @Tags         reports
// @Accept       json
// @Produce      json
// @Param        request body RefreshReportRequest true "Report refresh request"
// @Success      200 {object} dto.Response{data=RefreshReportResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /reports/refresh [post]
func (h *ReportHandler) RefreshReport(c *gin.Context) {
	if h.aggregationService == nil {
		h.InternalError(c, "Report aggregation service not configured")
		return
	}

	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req RefreshReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		h.BadRequest(c, "start_date: Invalid date format, expected YYYY-MM-DD")
		return
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		h.BadRequest(c, "end_date: Invalid date format, expected YYYY-MM-DD")
		return
	}
	endDate = endDate.Add(24*time.Hour - time.Second) // End of day

	// Validate report type
	reportType := scheduler.ReportType(req.ReportType)
	validTypes := scheduler.AllReportTypes()
	if !slices.Contains(validTypes, reportType) {
		h.BadRequest(c, "Invalid report_type. Valid types: SALES_SUMMARY, SALES_DAILY_TREND, INVENTORY_SUMMARY, PNL_MONTHLY, PRODUCT_RANKING, CUSTOMER_RANKING")
		return
	}

	// Execute refresh
	if err := h.aggregationService.RefreshReport(c.Request.Context(), tenantID, reportType, startDate, endDate); err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, RefreshReportResponse{
		Message:    "Report cache refreshed successfully",
		ReportType: req.ReportType,
	})
}

// RefreshAllReports godoc
// @Summary      Refresh all report caches
// @Description  Triggers manual refresh of all report type caches
// @Tags         reports
// @Accept       json
// @Produce      json
// @Param        request body RefreshAllReportsRequest true "Report refresh request"
// @Success      200 {object} dto.Response{data=RefreshReportResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /reports/refresh/all [post]
func (h *ReportHandler) RefreshAllReports(c *gin.Context) {
	if h.aggregationService == nil {
		h.InternalError(c, "Report aggregation service not configured")
		return
	}

	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req RefreshAllReportsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		h.BadRequest(c, "start_date: Invalid date format, expected YYYY-MM-DD")
		return
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		h.BadRequest(c, "end_date: Invalid date format, expected YYYY-MM-DD")
		return
	}
	endDate = endDate.Add(24*time.Hour - time.Second) // End of day

	// Execute refresh for all report types
	if err := h.aggregationService.RefreshAllReports(c.Request.Context(), tenantID, startDate, endDate); err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, RefreshReportResponse{
		Message: "All report caches refreshed successfully",
	})
}

// GetSchedulerStatus godoc
// @Summary      Get report scheduler status
// @Description  Returns the current status of the report scheduler
// @Tags         reports
// @Accept       json
// @Produce      json
// @Success      200 {object} dto.Response{data=map[string]interface{}}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /reports/scheduler/status [get]
func (h *ReportHandler) GetSchedulerStatus(c *gin.Context) {
	status := map[string]any{
		"enabled":            h.aggregationService != nil,
		"available_types":    scheduler.AllReportTypes(),
		"supported_schedule": "Daily",
	}

	// Include cron scheduler details if available
	if h.cronScheduler != nil {
		cronStatus := h.cronScheduler.GetStatus()
		maps.Copy(status, cronStatus)
	}

	h.Success(c, status)
}

// TriggerDailyAggregationRequest defines the request for triggering daily aggregation
// @Description Request for triggering daily report aggregation
type TriggerDailyAggregationRequest struct {
	StartDate string `json:"start_date" example:"2026-01-01"`
	EndDate   string `json:"end_date" example:"2026-01-31"`
}

// TriggerDailyAggregation godoc
// @Summary      Trigger daily report aggregation
// @Description  Manually triggers the daily report aggregation for a specific date range
// @Tags         reports
// @Accept       json
// @Produce      json
// @Param        request body TriggerDailyAggregationRequest false "Date range (optional, defaults to yesterday)"
// @Success      200 {object} dto.Response{data=RefreshReportResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /reports/scheduler/trigger [post]
func (h *ReportHandler) TriggerDailyAggregation(c *gin.Context) {
	if h.cronScheduler == nil {
		h.InternalError(c, "Report cron scheduler not configured")
		return
	}

	var req TriggerDailyAggregationRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		h.BadRequest(c, err.Error())
		return
	}

	// If dates provided, trigger for specific tenant with date range
	if req.StartDate != "" && req.EndDate != "" {
		tenantID, err := getTenantID(c)
		if err != nil {
			h.BadRequest(c, "Invalid tenant ID")
			return
		}

		startDate, err := time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			h.BadRequest(c, "start_date: Invalid date format, expected YYYY-MM-DD")
			return
		}

		endDate, err := time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			h.BadRequest(c, "end_date: Invalid date format, expected YYYY-MM-DD")
			return
		}
		endDate = endDate.Add(24*time.Hour - time.Second)

		if err := h.cronScheduler.TriggerTenantAggregation(c.Request.Context(), tenantID, startDate, endDate); err != nil {
			h.HandleError(c, err)
			return
		}

		h.Success(c, RefreshReportResponse{
			Message: "Report aggregation triggered for specified date range",
		})
		return
	}

	// Otherwise trigger full daily aggregation for all tenants
	if err := h.cronScheduler.TriggerManualRun(c.Request.Context()); err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, RefreshReportResponse{
		Message: "Daily report aggregation triggered for all tenants",
	})
}
