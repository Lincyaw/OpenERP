// TypeScript type definitions
// Shared types, interfaces, and DTOs

export type {
  ApiResponse,
  ApiError,
  ValidationError,
  PaginationMeta,
  PaginationParams,
  BaseEntity,
  TenantEntity,
} from './api'

// Domain Value Objects and Status Enums
// These types align with backend domain models for type safety
export type { Currency, Money } from './domain'

export {
  // Constants
  DEFAULT_CURRENCY,
  // Money utilities
  zeroMoney,
  createMoney,
  parseMoney,
  formatMoney,
  moneyEquals,
  isZeroMoney,
  // Identity status enums
  TenantStatus,
  UserStatus,
  // Catalog status enums
  ProductStatus,
  CategoryStatus,
  // Partner status enums
  CustomerStatus,
  SupplierStatus,
  WarehouseStatus,
  // Trade status enums
  SalesOrderStatus,
  SalesReturnStatus,
  PurchaseOrderStatus,
  PurchaseReturnStatus,
  // Finance status enums
  ReceivableStatus,
  PayableStatus,
  VoucherStatus,
  ExpenseStatus,
  PaymentStatus,
  IncomeStatus,
  ReceiptStatus,
  CreditMemoStatus,
  DebitMemoStatus,
  GatewayPaymentStatus,
  RefundStatus,
  // Inventory status enums
  StockTakingStatus,
  // Shared status enums
  OutboxStatus,
  TrialBalanceStatus,
} from './domain'
