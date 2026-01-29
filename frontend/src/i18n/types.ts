/**
 * i18n Type Definitions
 *
 * This file provides TypeScript type declarations for type-safe translation keys.
 */

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
  'table.export.csv': string
  'table.export.excel': string
  'table.export.selected': string
  'table.export.all': string
  'table.bulk.confirm': string
  'table.bulk.delete': string
  'table.bulk.export': string
  'table.bulk.confirmTitle': string
  'table.bulk.confirmContent': string
  'table.bulk.deleteTitle': string
  'table.bulk.deleteContent': string
  'table.bulk.success': string
  'table.bulk.error': string
  'table.bulk.partialSuccess': string
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
  // Error summary
  'errorSummary.title': string
  'errorSummary.single': string
  'errorSummary.multiple': string
  'errorSummary.screenReaderAnnounce': string
  // Field labels
  'fieldLabels.name': string
  'fieldLabels.code': string
  'fieldLabels.email': string
  'fieldLabels.phone': string
  'fieldLabels.password': string
  'fieldLabels.confirmPassword': string
  'fieldLabels.address': string
  'fieldLabels.description': string
  'fieldLabels.remark': string
  'fieldLabels.price': string
  'fieldLabels.quantity': string
  'fieldLabels.amount': string
  'fieldLabels.date': string
  'fieldLabels.startDate': string
  'fieldLabels.endDate': string
  'fieldLabels.category': string
  'fieldLabels.status': string
  'fieldLabels.type': string
  'fieldLabels.sku': string
  'fieldLabels.barcode': string
  'fieldLabels.unit': string
  'fieldLabels.warehouse': string
  'fieldLabels.customer': string
  'fieldLabels.supplier': string
  'fieldLabels.contact': string
  'fieldLabels.contactPhone': string
  // Index signature for additional keys
  [key: string]: string
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
 * Catalog namespace translation keys
 */
export interface CatalogTranslations {
  // Products
  'products.title': string
  'products.searchPlaceholder': string
  'products.addProduct': string
  'products.editProduct': string
  'products.createProduct': string
  'products.statusFilter': string
  'products.allStatus': string
  'products.status.active': string
  'products.status.inactive': string
  'products.status.discontinued': string
  'products.columns.code': string
  'products.columns.name': string
  'products.columns.unit': string
  'products.columns.purchasePrice': string
  'products.columns.sellingPrice': string
  'products.columns.status': string
  'products.columns.createdAt': string
  'products.actions.view': string
  'products.actions.edit': string
  'products.actions.activate': string
  'products.actions.deactivate': string
  'products.actions.discontinue': string
  'products.actions.delete': string
  'products.actions.batchActivate': string
  'products.actions.batchDeactivate': string
  'products.messages.fetchError': string
  'products.messages.activateSuccess': string
  'products.messages.activateError': string
  'products.messages.deactivateSuccess': string
  'products.messages.deactivateError': string
  'products.messages.discontinueSuccess': string
  'products.messages.discontinueError': string
  'products.messages.deleteSuccess': string
  'products.messages.deleteError': string
  'products.messages.batchActivateSuccess': string
  'products.messages.batchActivateError': string
  'products.messages.batchDeactivateSuccess': string
  'products.messages.batchDeactivateError': string
  'products.messages.invalidId': string
  'products.messages.loadError': string
  'products.messages.createSuccess': string
  'products.messages.updateSuccess': string
  'products.messages.createError': string
  'products.messages.updateError': string
  'products.confirm.discontinueTitle': string
  'products.confirm.discontinueContent': string
  'products.confirm.discontinueOk': string
  'products.confirm.deleteTitle': string
  'products.confirm.deleteContent': string
  'products.confirm.deleteOk': string
  'products.form.basicInfo': string
  'products.form.basicInfoDesc': string
  'products.form.priceInfo': string
  'products.form.priceInfoDesc': string
  'products.form.stockSettings': string
  'products.form.stockSettingsDesc': string
  'products.form.category': string
  'products.form.categoryPlaceholder': string
  'products.form.categoryRequired': string
  'products.form.code': string
  'products.form.codePlaceholder': string
  'products.form.codeHelperCreate': string
  'products.form.codeHelperEdit': string
  'products.form.codeRegexError': string
  'products.form.name': string
  'products.form.namePlaceholder': string
  'products.form.unit': string
  'products.form.unitPlaceholder': string
  'products.form.unitHelperCreate': string
  'products.form.unitHelperEdit': string
  'products.form.barcode': string
  'products.form.barcodePlaceholder': string
  'products.form.description': string
  'products.form.descriptionPlaceholder': string
  'products.form.purchasePrice': string
  'products.form.purchasePricePlaceholder': string
  'products.form.sellingPrice': string
  'products.form.sellingPricePlaceholder': string
  'products.form.minStock': string
  'products.form.minStockPlaceholder': string
  'products.form.minStockHelper': string
  'products.form.sortOrder': string
  'products.form.sortOrderPlaceholder': string
  'products.form.sortOrderHelper': string
  // Categories
  'categories.title': string
  'categories.searchPlaceholder': string
  'categories.addRootCategory': string
  'categories.addChildCategory': string
  'categories.editCategory': string
  'categories.createChildTitle': string
  'categories.createRootTitle': string
  'categories.expandAll': string
  'categories.collapseAll': string
  'categories.deactivated': string
  'categories.rootCategory': string
  'categories.status.active': string
  'categories.status.inactive': string
  'categories.actions.viewDetail': string
  'categories.actions.addChild': string
  'categories.actions.edit': string
  'categories.actions.activate': string
  'categories.actions.deactivate': string
  'categories.actions.delete': string
  'categories.messages.fetchError': string
  'categories.messages.createSuccess': string
  'categories.messages.createError': string
  'categories.messages.updateSuccess': string
  'categories.messages.updateError': string
  'categories.messages.deleteSuccess': string
  'categories.messages.deleteError': string
  'categories.messages.activateSuccess': string
  'categories.messages.activateError': string
  'categories.messages.deactivateSuccess': string
  'categories.messages.deactivateError': string
  'categories.messages.moveSuccess': string
  'categories.messages.moveError': string
  'categories.messages.hasChildren': string
  'categories.confirm.deleteTitle': string
  'categories.confirm.deleteContent': string
  'categories.confirm.deleteOk': string
  'categories.confirm.deactivateTitle': string
  'categories.confirm.deactivateContent': string
  'categories.confirm.deactivateOk': string
  'categories.form.code': string
  'categories.form.codePlaceholder': string
  'categories.form.codeRequired': string
  'categories.form.codeMinLength': string
  'categories.form.codeMaxLength': string
  'categories.form.name': string
  'categories.form.namePlaceholder': string
  'categories.form.nameRequired': string
  'categories.form.nameMinLength': string
  'categories.form.nameMaxLength': string
  'categories.form.description': string
  'categories.form.descriptionPlaceholder': string
  'categories.form.sortOrder': string
  'categories.form.sortOrderPlaceholder': string
  'categories.form.parentCategory': string
  'categories.detail.title': string
  'categories.detail.code': string
  'categories.detail.name': string
  'categories.detail.description': string
  'categories.detail.status': string
  'categories.detail.level': string
  'categories.detail.sortOrder': string
  'categories.detail.childCount': string
  'categories.detail.parentCategory': string
  'categories.detail.childCategories': string
  'categories.empty.title': string
  'categories.empty.titleSearch': string
  'categories.empty.description': string
  'categories.empty.descriptionSearch': string
  // Product Detail
  'productDetail.title': string
  'productDetail.back': string
  'productDetail.notExist': string
  'productDetail.notExistDesc': string
  'productDetail.timestamps': string
  'productDetail.fields.code': string
  'productDetail.fields.name': string
  'productDetail.fields.unit': string
  'productDetail.fields.barcode': string
  'productDetail.fields.status': string
  'productDetail.fields.description': string
  'productDetail.fields.purchasePrice': string
  'productDetail.fields.sellingPrice': string
  'productDetail.fields.profitMargin': string
  'productDetail.fields.minStock': string
  'productDetail.fields.sortOrder': string
  'productDetail.fields.createdAt': string
  'productDetail.fields.updatedAt': string
}

/**
 * Translation keys type mapping by namespace
 */
export interface TranslationKeys {
  common: CommonTranslations
  validation: ValidationTranslations
  auth: AuthTranslations
  catalog: CatalogTranslations
  // Other namespaces will be added as they are implemented
  partner: Record<string, string>
  trade: Record<string, string>
  inventory: Record<string, string>
  finance: Record<string, string>
  system: Record<string, string>
  integration: Record<string, string>
  admin: Record<string, string>
}

/**
 * Type-safe translation key for a specific namespace
 */
export type TranslationKey<N extends keyof TranslationKeys> = keyof TranslationKeys[N]

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
