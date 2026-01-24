/**
 * Report API Service
 *
 * Manual API service for report module.
 * TODO: Replace with auto-generated orval client once Report API is integrated
 */
import { customInstance } from '../../services/axios-instance'
import type {
  SalesSummary,
  DailySalesTrend,
  ProductSalesRanking,
  CustomerSalesRanking,
  SalesReportParams,
  InventorySummary,
  InventoryTurnover,
  InventoryValueByCategory,
  InventoryValueByWarehouse,
  InventoryReportParams,
  ProfitLossStatement,
  MonthlyProfitTrend,
  ProfitByProduct,
  CashFlowStatement,
  CashFlowItem,
  FinanceReportParams,
  ApiResponse,
} from './types'

export const getReportsApi = () => {
  // ===================== Sales Report APIs =====================

  /**
   * Get sales summary for the specified period
   */
  const getReportsSalesSummary = (params: SalesReportParams) => {
    return customInstance<ApiResponse<SalesSummary>>({
      url: '/reports/sales/summary',
      method: 'GET',
      params,
    })
  }

  /**
   * Get daily sales trend data
   */
  const getReportsSalesDailyTrend = (params: SalesReportParams) => {
    return customInstance<ApiResponse<DailySalesTrend[]>>({
      url: '/reports/sales/daily-trend',
      method: 'GET',
      params,
    })
  }

  /**
   * Get product sales ranking
   */
  const getReportsSalesProductsRanking = (params: SalesReportParams) => {
    return customInstance<ApiResponse<ProductSalesRanking[]>>({
      url: '/reports/sales/products/ranking',
      method: 'GET',
      params,
    })
  }

  /**
   * Get customer sales ranking
   */
  const getReportsSalesCustomersRanking = (params: SalesReportParams) => {
    return customInstance<ApiResponse<CustomerSalesRanking[]>>({
      url: '/reports/sales/customers/ranking',
      method: 'GET',
      params,
    })
  }

  // ===================== Inventory Report APIs =====================

  /**
   * Get inventory summary
   */
  const getReportsInventorySummary = (params: InventoryReportParams) => {
    return customInstance<ApiResponse<InventorySummary>>({
      url: '/reports/inventory/summary',
      method: 'GET',
      params,
    })
  }

  /**
   * Get inventory turnover data
   */
  const getReportsInventoryTurnover = (params: InventoryReportParams) => {
    return customInstance<ApiResponse<InventoryTurnover[]>>({
      url: '/reports/inventory/turnover',
      method: 'GET',
      params,
    })
  }

  /**
   * Get inventory value by category
   */
  const getReportsInventoryValueByCategory = (params: InventoryReportParams) => {
    return customInstance<ApiResponse<InventoryValueByCategory[]>>({
      url: '/reports/inventory/value-by-category',
      method: 'GET',
      params,
    })
  }

  /**
   * Get inventory value by warehouse
   */
  const getReportsInventoryValueByWarehouse = (params: InventoryReportParams) => {
    return customInstance<ApiResponse<InventoryValueByWarehouse[]>>({
      url: '/reports/inventory/value-by-warehouse',
      method: 'GET',
      params,
    })
  }

  /**
   * Get slow moving products
   */
  const getReportsInventorySlowMoving = (params: InventoryReportParams) => {
    return customInstance<ApiResponse<InventoryTurnover[]>>({
      url: '/reports/inventory/slow-moving',
      method: 'GET',
      params,
    })
  }

  // ===================== Finance Report APIs =====================

  /**
   * Get profit and loss statement
   */
  const getReportsFinanceProfitLoss = (params: FinanceReportParams) => {
    return customInstance<ApiResponse<ProfitLossStatement>>({
      url: '/reports/finance/profit-loss',
      method: 'GET',
      params,
    })
  }

  /**
   * Get monthly profit trend
   */
  const getReportsFinanceMonthlyTrend = (params: FinanceReportParams) => {
    return customInstance<ApiResponse<MonthlyProfitTrend[]>>({
      url: '/reports/finance/monthly-trend',
      method: 'GET',
      params,
    })
  }

  /**
   * Get profit by product
   */
  const getReportsFinanceProfitByProduct = (params: FinanceReportParams) => {
    return customInstance<ApiResponse<ProfitByProduct[]>>({
      url: '/reports/finance/profit-by-product',
      method: 'GET',
      params,
    })
  }

  /**
   * Get cash flow statement
   */
  const getReportsFinanceCashFlow = (params: FinanceReportParams) => {
    return customInstance<ApiResponse<CashFlowStatement>>({
      url: '/reports/finance/cash-flow',
      method: 'GET',
      params,
    })
  }

  /**
   * Get cash flow items
   */
  const getReportsFinanceCashFlowItems = (params: FinanceReportParams) => {
    return customInstance<ApiResponse<CashFlowItem[]>>({
      url: '/reports/finance/cash-flow/items',
      method: 'GET',
      params,
    })
  }

  return {
    // Sales Reports
    getReportsSalesSummary,
    getReportsSalesDailyTrend,
    getReportsSalesProductsRanking,
    getReportsSalesCustomersRanking,
    // Inventory Reports
    getReportsInventorySummary,
    getReportsInventoryTurnover,
    getReportsInventoryValueByCategory,
    getReportsInventoryValueByWarehouse,
    getReportsInventorySlowMoving,
    // Finance Reports
    getReportsFinanceProfitLoss,
    getReportsFinanceMonthlyTrend,
    getReportsFinanceProfitByProduct,
    getReportsFinanceCashFlow,
    getReportsFinanceCashFlowItems,
  }
}

// Export types for external use
export * from './types'
