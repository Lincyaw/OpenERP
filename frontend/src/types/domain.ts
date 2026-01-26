/**
 * Domain Value Objects and Enums
 *
 * This file contains TypeScript types that align with backend domain models.
 * These types provide type safety for domain-specific values.
 *
 * @see backend/internal/domain/shared/valueobject/money.go
 */

// =============================================================================
// Money Value Object
// =============================================================================

/**
 * Currency codes following ISO 4217 standard.
 * Must align with backend/internal/domain/shared/valueobject/money.go
 */
export type Currency = 'CNY' | 'USD' | 'EUR' | 'GBP' | 'JPY' | 'HKD'

/**
 * Default currency for the system (Chinese Yuan).
 */
export const DEFAULT_CURRENCY: Currency = 'CNY'

/**
 * Money value object representing monetary amounts.
 * Immutable - all operations should return new Money instances.
 *
 * @see backend/internal/domain/shared/valueobject/money.go
 */
export interface Money {
  /** The monetary amount as a string (for precision) */
  amount: string
  /** The currency code */
  currency: Currency
}

/**
 * Create a zero Money value in the specified currency.
 */
export const zeroMoney = (currency: Currency = DEFAULT_CURRENCY): Money => ({
  amount: '0',
  currency,
})

/**
 * Create a Money value from a number.
 */
export const createMoney = (amount: number, currency: Currency = DEFAULT_CURRENCY): Money => ({
  amount: amount.toString(),
  currency,
})

/**
 * Parse a Money value from a string.
 */
export const parseMoney = (amount: string, currency: Currency = DEFAULT_CURRENCY): Money => ({
  amount,
  currency,
})

/**
 * Format Money for display.
 */
export const formatMoney = (money: Money, locale: string = 'zh-CN'): string => {
  const amount = parseFloat(money.amount)
  return new Intl.NumberFormat(locale, {
    style: 'currency',
    currency: money.currency,
  }).format(amount)
}

/**
 * Check if two Money values are equal.
 */
export const moneyEquals = (a: Money, b: Money): boolean =>
  a.amount === b.amount && a.currency === b.currency

/**
 * Check if Money is zero.
 */
export const isZeroMoney = (money: Money): boolean => parseFloat(money.amount) === 0

// =============================================================================
// Identity Module Status Enums
// =============================================================================

/**
 * Tenant status values.
 * @see backend/internal/domain/identity/tenant.go
 */
export const TenantStatus = {
  ACTIVE: 'active',
  INACTIVE: 'inactive',
  SUSPENDED: 'suspended',
  TRIAL: 'trial',
} as const

export type TenantStatus = (typeof TenantStatus)[keyof typeof TenantStatus]

/**
 * User status values.
 * @see backend/internal/domain/identity/user.go
 */
export const UserStatus = {
  PENDING: 'pending',
  ACTIVE: 'active',
  LOCKED: 'locked',
  DEACTIVATED: 'deactivated',
} as const

export type UserStatus = (typeof UserStatus)[keyof typeof UserStatus]

// =============================================================================
// Catalog Module Status Enums
// =============================================================================

/**
 * Product status values.
 * @see backend/internal/domain/catalog/product.go
 */
export const ProductStatus = {
  ACTIVE: 'active',
  INACTIVE: 'inactive',
  DISCONTINUED: 'discontinued',
} as const

export type ProductStatus = (typeof ProductStatus)[keyof typeof ProductStatus]

/**
 * Category status values.
 * @see backend/internal/domain/catalog/category.go
 */
export const CategoryStatus = {
  ACTIVE: 'active',
  INACTIVE: 'inactive',
} as const

export type CategoryStatus = (typeof CategoryStatus)[keyof typeof CategoryStatus]

// =============================================================================
// Partner Module Status Enums
// =============================================================================

/**
 * Customer status values.
 * @see backend/internal/domain/partner/customer.go
 */
export const CustomerStatus = {
  ACTIVE: 'active',
  INACTIVE: 'inactive',
  SUSPENDED: 'suspended',
} as const

export type CustomerStatus = (typeof CustomerStatus)[keyof typeof CustomerStatus]

/**
 * Supplier status values.
 * @see backend/internal/domain/partner/supplier.go
 */
export const SupplierStatus = {
  ACTIVE: 'active',
  INACTIVE: 'inactive',
  BLOCKED: 'blocked',
} as const

export type SupplierStatus = (typeof SupplierStatus)[keyof typeof SupplierStatus]

/**
 * Warehouse status values.
 * @see backend/internal/domain/partner/warehouse.go
 */
export const WarehouseStatus = {
  ACTIVE: 'active',
  INACTIVE: 'inactive',
} as const

export type WarehouseStatus = (typeof WarehouseStatus)[keyof typeof WarehouseStatus]

// =============================================================================
// Trade Module Status Enums
// =============================================================================

/**
 * Sales order status values.
 * @see backend/internal/domain/trade/sales_order.go
 */
export const SalesOrderStatus = {
  DRAFT: 'DRAFT',
  CONFIRMED: 'CONFIRMED',
  SHIPPED: 'SHIPPED',
  COMPLETED: 'COMPLETED',
  CANCELLED: 'CANCELLED',
} as const

export type SalesOrderStatus = (typeof SalesOrderStatus)[keyof typeof SalesOrderStatus]

/**
 * Sales return status values.
 * @see backend/internal/domain/trade/sales_return.go
 */
export const SalesReturnStatus = {
  DRAFT: 'DRAFT',
  PENDING: 'PENDING',
  APPROVED: 'APPROVED',
  RECEIVING: 'RECEIVING',
  REJECTED: 'REJECTED',
  COMPLETED: 'COMPLETED',
  CANCELLED: 'CANCELLED',
} as const

export type SalesReturnStatus = (typeof SalesReturnStatus)[keyof typeof SalesReturnStatus]

/**
 * Purchase order status values.
 * @see backend/internal/domain/trade/purchase_order.go
 */
export const PurchaseOrderStatus = {
  DRAFT: 'DRAFT',
  CONFIRMED: 'CONFIRMED',
  PARTIAL_RECEIVED: 'PARTIAL_RECEIVED',
  COMPLETED: 'COMPLETED',
  CANCELLED: 'CANCELLED',
} as const

export type PurchaseOrderStatus = (typeof PurchaseOrderStatus)[keyof typeof PurchaseOrderStatus]

/**
 * Purchase return status values.
 * @see backend/internal/domain/trade/purchase_return.go
 */
export const PurchaseReturnStatus = {
  DRAFT: 'DRAFT',
  PENDING: 'PENDING',
  APPROVED: 'APPROVED',
  REJECTED: 'REJECTED',
  SHIPPED: 'SHIPPED',
  COMPLETED: 'COMPLETED',
  CANCELLED: 'CANCELLED',
} as const

export type PurchaseReturnStatus = (typeof PurchaseReturnStatus)[keyof typeof PurchaseReturnStatus]

// =============================================================================
// Finance Module Status Enums
// =============================================================================

/**
 * Receivable status values.
 * @see backend/internal/domain/finance/account_receivable.go
 */
export const ReceivableStatus = {
  PENDING: 'PENDING',
  PARTIAL: 'PARTIAL',
  PAID: 'PAID',
  REVERSED: 'REVERSED',
  CANCELLED: 'CANCELLED',
} as const

export type ReceivableStatus = (typeof ReceivableStatus)[keyof typeof ReceivableStatus]

/**
 * Payable status values.
 * @see backend/internal/domain/finance/account_payable.go
 */
export const PayableStatus = {
  PENDING: 'PENDING',
  PARTIAL: 'PARTIAL',
  PAID: 'PAID',
  REVERSED: 'REVERSED',
  CANCELLED: 'CANCELLED',
} as const

export type PayableStatus = (typeof PayableStatus)[keyof typeof PayableStatus]

/**
 * Voucher status values (for receipt/payment vouchers).
 * @see backend/internal/domain/finance/receipt_voucher.go
 */
export const VoucherStatus = {
  DRAFT: 'DRAFT',
  CONFIRMED: 'CONFIRMED',
  ALLOCATED: 'ALLOCATED',
  CANCELLED: 'CANCELLED',
} as const

export type VoucherStatus = (typeof VoucherStatus)[keyof typeof VoucherStatus]

/**
 * Expense record status values.
 * @see backend/internal/domain/finance/expense_record.go
 */
export const ExpenseStatus = {
  DRAFT: 'DRAFT',
  PENDING: 'PENDING',
  APPROVED: 'APPROVED',
  REJECTED: 'REJECTED',
  CANCELLED: 'CANCELLED',
} as const

export type ExpenseStatus = (typeof ExpenseStatus)[keyof typeof ExpenseStatus]

/**
 * Payment status values (for expense records).
 * @see backend/internal/domain/finance/expense_record.go
 */
export const PaymentStatus = {
  UNPAID: 'UNPAID',
  PAID: 'PAID',
} as const

export type PaymentStatus = (typeof PaymentStatus)[keyof typeof PaymentStatus]

/**
 * Income record status values.
 * @see backend/internal/domain/finance/other_income_record.go
 */
export const IncomeStatus = {
  DRAFT: 'DRAFT',
  CONFIRMED: 'CONFIRMED',
  CANCELLED: 'CANCELLED',
} as const

export type IncomeStatus = (typeof IncomeStatus)[keyof typeof IncomeStatus]

/**
 * Receipt status values (for income records).
 * @see backend/internal/domain/finance/other_income_record.go
 */
export const ReceiptStatus = {
  PENDING: 'PENDING',
  RECEIVED: 'RECEIVED',
} as const

export type ReceiptStatus = (typeof ReceiptStatus)[keyof typeof ReceiptStatus]

/**
 * Credit memo status values.
 * @see backend/internal/domain/finance/credit_memo.go
 */
export const CreditMemoStatus = {
  PENDING: 'PENDING',
  APPLIED: 'APPLIED',
  PARTIAL: 'PARTIAL',
  VOIDED: 'VOIDED',
  REFUNDED: 'REFUNDED',
} as const

export type CreditMemoStatus = (typeof CreditMemoStatus)[keyof typeof CreditMemoStatus]

/**
 * Debit memo status values.
 * @see backend/internal/domain/finance/debit_memo.go
 */
export const DebitMemoStatus = {
  PENDING: 'PENDING',
  APPLIED: 'APPLIED',
  PARTIAL: 'PARTIAL',
  VOIDED: 'VOIDED',
  REFUNDED: 'REFUNDED',
} as const

export type DebitMemoStatus = (typeof DebitMemoStatus)[keyof typeof DebitMemoStatus]

/**
 * Gateway payment status values.
 * @see backend/internal/domain/finance/payment_gateway.go
 */
export const GatewayPaymentStatus = {
  PENDING: 'PENDING',
  PAID: 'PAID',
  FAILED: 'FAILED',
} as const

export type GatewayPaymentStatus = (typeof GatewayPaymentStatus)[keyof typeof GatewayPaymentStatus]

/**
 * Refund status values.
 * @see backend/internal/domain/finance/payment_gateway.go
 */
export const RefundStatus = {
  PENDING: 'PENDING',
  SUCCESS: 'SUCCESS',
  FAILED: 'FAILED',
} as const

export type RefundStatus = (typeof RefundStatus)[keyof typeof RefundStatus]

// =============================================================================
// Inventory Module Status Enums
// =============================================================================

/**
 * Stock taking status values.
 * @see backend/internal/domain/inventory/stock_taking.go
 */
export const StockTakingStatus = {
  DRAFT: 'DRAFT',
  COUNTING: 'COUNTING',
  PENDING_APPROVAL: 'PENDING_APPROVAL',
  APPROVED: 'APPROVED',
  REJECTED: 'REJECTED',
  CANCELLED: 'CANCELLED',
} as const

export type StockTakingStatus = (typeof StockTakingStatus)[keyof typeof StockTakingStatus]

// =============================================================================
// Shared Status Enums
// =============================================================================

/**
 * Outbox message status values.
 * @see backend/internal/domain/shared/outbox.go
 */
export const OutboxStatus = {
  PENDING: 'PENDING',
  PROCESSING: 'PROCESSING',
  SENT: 'SENT',
  FAILED: 'FAILED',
  DEAD: 'DEAD',
} as const

export type OutboxStatus = (typeof OutboxStatus)[keyof typeof OutboxStatus]

/**
 * Trial balance status values.
 * @see backend/internal/domain/finance/trial_balance.go
 */
export const TrialBalanceStatus = {
  BALANCED: 'BALANCED',
  UNBALANCED: 'UNBALANCED',
} as const

export type TrialBalanceStatus = (typeof TrialBalanceStatus)[keyof typeof TrialBalanceStatus]
