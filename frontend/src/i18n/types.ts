/**
 * i18n Type Definitions
 *
 * This file provides TypeScript type declarations for type-safe translation keys.
 */

import type { Namespace } from './config'

/**
 * Common namespace translation keys
 */
export interface CommonTranslations {
  // Actions
  'actions.create': string
  'actions.edit': string
  'actions.delete': string
  'actions.save': string
  'actions.cancel': string
  'actions.confirm': string
  'actions.search': string
  'actions.reset': string
  'actions.refresh': string
  'actions.export': string
  'actions.import': string
  'actions.back': string
  'actions.submit': string
  'actions.enable': string
  'actions.disable': string
  'actions.view': string
  'actions.more': string
  'actions.close': string
  'actions.retry': string
  'actions.download': string
  'actions.upload': string
  'actions.filter': string
  'actions.clear': string
  'actions.selectAll': string
  'actions.batchDelete': string
  'actions.approve': string
  'actions.reject': string
  'actions.logout': string
  'actions.switchToDark': string
  'actions.switchToLight': string
  'actions.moreActions': string
  'actions.cancelSelection': string
  // Status
  'status.enabled': string
  'status.disabled': string
  'status.active': string
  'status.inactive': string
  'status.pending': string
  'status.completed': string
  'status.cancelled': string
  'status.draft': string
  'status.confirmed': string
  'status.processing': string
  'status.success': string
  'status.failed': string
  'status.loading': string
  // Labels
  'labels.name': string
  'labels.code': string
  'labels.description': string
  'labels.status': string
  'labels.createdAt': string
  'labels.updatedAt': string
  'labels.createdBy': string
  'labels.remark': string
  'labels.operation': string
  'labels.total': string
  'labels.amount': string
  'labels.quantity': string
  'labels.price': string
  'labels.unit': string
  'labels.type': string
  'labels.date': string
  'labels.time': string
  'labels.startDate': string
  'labels.endDate': string
  'labels.all': string
  'labels.none': string
  'labels.yes': string
  'labels.no': string
  'labels.required': string
  'labels.optional': string
  // Messages
  'messages.loading': string
  'messages.noData': string
  'messages.createSuccess': string
  'messages.updateSuccess': string
  'messages.deleteSuccess': string
  'messages.operationSuccess': string
  'messages.operationFailed': string
  'messages.confirmDelete': string
  'messages.confirmDeleteTitle': string
  'messages.networkError': string
  'messages.serverError': string
  'messages.unauthorized': string
  'messages.forbidden': string
  'messages.notFound': string
  'messages.validationError': string
  'messages.unsavedChanges': string
  'messages.sessionExpired': string
  // Pagination
  'pagination.total': string
  'pagination.page': string
  'pagination.pageSize': string
  'pagination.prev': string
  'pagination.next': string
  'pagination.goto': string
  // Table
  'table.noData': string
  'table.noDataDescription': string
  'table.selectPlaceholder': string
  'table.actions': string
  'table.selectedItems': string
  'table.totalRecords': string
  'table.searchPlaceholder': string
  // Dashboard
  'dashboard.title': string
  'dashboard.welcome': string
  'dashboard.fetchError': string
  'dashboard.metrics.products': string
  'dashboard.metrics.activeProducts': string
  'dashboard.metrics.customers': string
  'dashboard.metrics.activeCustomers': string
  'dashboard.metrics.salesOrders': string
  'dashboard.metrics.pendingShipment': string
  'dashboard.metrics.lowStockAlert': string
  'dashboard.metrics.needRestock': string
  'dashboard.metrics.receivables': string
  'dashboard.metrics.pendingReceipts': string
  'dashboard.metrics.payables': string
  'dashboard.metrics.pendingPayments': string
  'dashboard.orderStats.title': string
  'dashboard.orderStats.completionRate': string
  'dashboard.orderStats.draft': string
  'dashboard.orderStats.confirmed': string
  'dashboard.orderStats.shipped': string
  'dashboard.orderStats.completed': string
  'dashboard.orderStats.cancelled': string
  'dashboard.recentOrders.title': string
  'dashboard.recentOrders.viewAll': string
  'dashboard.recentOrders.noOrders': string
  'dashboard.recentOrders.unknownCustomer': string
  'dashboard.pendingTasks.title': string
  'dashboard.pendingTasks.noTasks': string
  'dashboard.pendingTasks.priority.high': string
  'dashboard.pendingTasks.priority.medium': string
  'dashboard.pendingTasks.priority.low': string
  'dashboard.pendingTasks.draftOrders': string
  'dashboard.pendingTasks.draftOrdersDesc': string
  'dashboard.pendingTasks.confirmedOrders': string
  'dashboard.pendingTasks.confirmedOrdersDesc': string
  'dashboard.pendingTasks.lowStock': string
  'dashboard.pendingTasks.lowStockDesc': string
  'dashboard.pendingTasks.pendingReceivables': string
  'dashboard.pendingTasks.pendingReceivablesDesc': string
  'dashboard.pendingTasks.pendingPayables': string
  'dashboard.pendingTasks.pendingPayablesDesc': string
  // Navigation
  'nav.dashboard': string
  'nav.catalog': string
  'nav.products': string
  'nav.categories': string
  'nav.partners': string
  'nav.customers': string
  'nav.suppliers': string
  'nav.warehouses': string
  'nav.inventory': string
  'nav.stock': string
  'nav.stockTaking': string
  'nav.trade': string
  'nav.salesOrders': string
  'nav.purchaseOrders': string
  'nav.salesReturns': string
  'nav.purchaseReturns': string
  'nav.finance': string
  'nav.receivables': string
  'nav.payables': string
  'nav.receipts': string
  'nav.payments': string
  'nav.expenses': string
  'nav.otherIncome': string
  'nav.cashFlow': string
  'nav.reports': string
  'nav.salesReport': string
  'nav.salesRanking': string
  'nav.inventoryTurnover': string
  'nav.profitLoss': string
  'nav.system': string
  'nav.users': string
  'nav.roles': string
  'nav.permissions': string
  'nav.settings': string
  'nav.profile': string
  'nav.notifications': string
}

/**
 * Validation namespace translation keys
 */
export interface ValidationTranslations {
  required: string
  email: string
  phone: string
  url: string
  minLength: string
  maxLength: string
  min: string
  max: string
  pattern: string
  numeric: string
  integer: string
  positive: string
  nonNegative: string
  date: string
  dateRange: string
  unique: string
  confirm: string
  passwordStrength: string
  invalidFormat: string
}

/**
 * Auth namespace translation keys
 */
export interface AuthTranslations {
  // Login
  'login.title': string
  'login.subtitle': string
  'login.username': string
  'login.usernamePlaceholder': string
  'login.password': string
  'login.passwordPlaceholder': string
  'login.rememberMe': string
  'login.forgotPassword': string
  'login.submit': string
  'login.success': string
  'login.failed': string
  'login.invalidCredentials': string
  'login.accountLocked': string
  'login.accountDisabled': string
  'login.userNotFound': string
  // Logout
  'logout.title': string
  'logout.confirm': string
  'logout.success': string
  // Token
  'token.expired': string
  'token.refreshFailed': string
  'token.invalid': string
  // Permission
  'permission.denied': string
  'permission.noAccess': string
  // Forbidden page
  'forbidden.title': string
  'forbidden.code': string
  'forbidden.description': string
  'forbidden.attemptedPath': string
  'forbidden.backToDashboard': string
  'forbidden.goBack': string
  // Validation
  'validation.usernameRequired': string
  'validation.usernameMinLength': string
  'validation.usernameMaxLength': string
  'validation.passwordRequired': string
  'validation.passwordMinLength': string
  'validation.passwordMaxLength': string
}

/**
 * Translation keys type mapping by namespace
 */
export interface TranslationKeys {
  common: CommonTranslations
  validation: ValidationTranslations
  auth: AuthTranslations
  // Other namespaces will be added as they are implemented
  catalog: Record<string, string>
  partner: Record<string, string>
  trade: Record<string, string>
  inventory: Record<string, string>
  finance: Record<string, string>
  system: Record<string, string>
}

/**
 * Type-safe translation key for a specific namespace
 */
export type TranslationKey<N extends Namespace> = keyof TranslationKeys[N]

/**
 * Interpolation parameters type
 */
export type InterpolationParams = Record<string, string | number | boolean>

/**
 * Augment i18next types for type-safe translations
 */
declare module 'i18next' {
  interface CustomTypeOptions {
    defaultNS: 'common'
    resources: TranslationKeys
  }
}
