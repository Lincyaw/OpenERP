/**
 * Report API Types
 *
 * Types for report module API responses
 */

// ===================== Common Types =====================

export interface ApiResponse<T> {
  success: boolean
  data?: T
  error?: {
    code: string
    message: string
  }
  meta?: {
    total: number
    page: number
    page_size: number
  }
}

export interface ReportDateRange {
  start_date: string // YYYY-MM-DD
  end_date: string // YYYY-MM-DD
}

// ===================== Sales Report Types =====================

export interface SalesSummary {
  period_start: string
  period_end: string
  total_orders: number
  total_quantity: number
  total_sales_amount: number
  total_cost_amount: number
  total_gross_profit: number
  avg_order_value: number
  profit_margin: number
}

export interface DailySalesTrend {
  date: string
  order_count: number
  total_amount: number
  total_profit: number
  items_sold: number
}

export interface ProductSalesRanking {
  rank: number
  product_id: string
  product_sku: string
  product_name: string
  category_name?: string
  total_quantity: number
  total_amount: number
  total_profit: number
  order_count: number
}

export interface CustomerSalesRanking {
  rank: number
  customer_id: string
  customer_name: string
  total_orders: number
  total_quantity: number
  total_amount: number
  total_profit: number
}

export interface SalesReportParams extends ReportDateRange {
  product_id?: string
  category_id?: string
  customer_id?: string
  top_n?: number
}

// ===================== Inventory Report Types =====================

export interface InventorySummary {
  total_products: number
  total_quantity: number
  total_value: number
  avg_turnover_rate: number
  low_stock_count: number
  out_of_stock_count: number
  overstock_count: number
}

export interface InventoryTurnover {
  product_id: string
  product_sku: string
  product_name: string
  category_name?: string
  warehouse_name?: string
  beginning_stock: number
  ending_stock: number
  average_stock: number
  sold_quantity: number
  turnover_rate: number
  days_of_inventory: number
  stock_value: number
}

export interface InventoryValueByCategory {
  category_id?: string
  category_name: string
  product_count: number
  total_quantity: number
  total_value: number
  percentage: number
}

export interface InventoryValueByWarehouse {
  warehouse_id: string
  warehouse_name: string
  product_count: number
  total_quantity: number
  total_value: number
  percentage: number
}

export interface InventoryReportParams extends ReportDateRange {
  product_id?: string
  category_id?: string
  warehouse_id?: string
  top_n?: number
}

// ===================== Finance Report Types =====================

export interface ProfitLossStatement {
  period_start: string
  period_end: string
  sales_revenue: number
  sales_returns: number
  net_sales_revenue: number
  cogs: number
  gross_profit: number
  gross_margin: number
  other_income: number
  total_income: number
  expenses: number
  net_profit: number
  net_margin: number
}

export interface MonthlyProfitTrend {
  year: number
  month: number
  sales_revenue: number
  gross_profit: number
  net_profit: number
  gross_margin: number
  net_margin: number
}

export interface ProfitByProduct {
  product_id: string
  product_sku: string
  product_name: string
  category_name?: string
  sales_revenue: number
  cogs: number
  gross_profit: number
  gross_margin: number
  contribution: number
}

export interface CashFlowStatement {
  period_start: string
  period_end: string
  receipts_from_customers: number
  payments_to_suppliers: number
  other_income: number
  expense_payments: number
  net_operating_cash_flow: number
  beginning_cash: number
  net_cash_flow: number
  ending_cash: number
}

export interface CashFlowItem {
  date: string
  type: string
  reference_no: string
  description: string
  amount: number
  running_balance?: number
}

export interface FinanceReportParams extends ReportDateRange {
  product_id?: string
  customer_id?: string
  category_id?: string
  top_n?: number
}
