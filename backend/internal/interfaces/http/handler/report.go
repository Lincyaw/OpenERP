package handler

import (
	"errors"
	"time"

	reportapp "github.com/erp/backend/internal/application/report"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ReportHandler handles report-related API endpoints
type ReportHandler struct {
	BaseHandler
	reportService *reportapp.ReportService
}

// NewReportHandler creates a new ReportHandler
func NewReportHandler(reportService *reportapp.ReportService) *ReportHandler {
	return &ReportHandler{
		reportService: reportService,
	}
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
// @Success      200 {object} dto.Response{data=reportapp.SalesSummaryResponse}
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
// @Success      200 {object} dto.Response{data=[]reportapp.DailySalesTrendResponse}
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
// @Success      200 {object} dto.Response{data=[]reportapp.ProductSalesRankingResponse}
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
// @Success      200 {object} dto.Response{data=[]reportapp.CustomerSalesRankingResponse}
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
// @Success      200 {object} dto.Response{data=reportapp.InventorySummaryResponse}
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
// @Success      200 {object} dto.Response{data=[]reportapp.InventoryTurnoverResponse}
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
// @Success      200 {object} dto.Response{data=[]reportapp.InventoryValueByCategoryResponse}
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
// @Success      200 {object} dto.Response{data=[]reportapp.InventoryValueByWarehouseResponse}
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
// @Success      200 {object} dto.Response{data=[]reportapp.InventoryTurnoverResponse}
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
// @Success      200 {object} dto.Response{data=reportapp.ProfitLossStatementResponse}
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
// @Success      200 {object} dto.Response{data=[]reportapp.MonthlyProfitTrendResponse}
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
// @Success      200 {object} dto.Response{data=[]reportapp.ProfitByProductResponse}
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
// @Success      200 {object} dto.Response{data=reportapp.CashFlowStatementResponse}
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
// @Success      200 {object} dto.Response{data=[]reportapp.CashFlowItemResponse}
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
