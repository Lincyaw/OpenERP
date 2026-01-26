/**
 * Permission constants for the ERP system
 *
 * Permissions follow the format: resource:action
 * These match the backend permission definitions in identity/role.go
 */

// Resources
export const Resources = {
  PRODUCT: 'product',
  CATEGORY: 'category',
  CUSTOMER: 'customer',
  SUPPLIER: 'supplier',
  WAREHOUSE: 'warehouse',
  INVENTORY: 'inventory',
  SALES_ORDER: 'sales_order',
  PURCHASE_ORDER: 'purchase_order',
  SALES_RETURN: 'sales_return',
  PURCHASE_RETURN: 'purchase_return',
  ACCOUNT_RECEIVABLE: 'account_receivable',
  ACCOUNT_PAYABLE: 'account_payable',
  RECEIPT: 'receipt',
  PAYMENT: 'payment',
  EXPENSE: 'expense',
  INCOME: 'income',
  REPORT: 'report',
  USER: 'user',
  ROLE: 'role',
  TENANT: 'tenant',
} as const

// Actions
export const Actions = {
  CREATE: 'create',
  READ: 'read',
  UPDATE: 'update',
  DELETE: 'delete',
  ENABLE: 'enable',
  DISABLE: 'disable',
  CONFIRM: 'confirm',
  CANCEL: 'cancel',
  SHIP: 'ship',
  RECEIVE: 'receive',
  APPROVE: 'approve',
  REJECT: 'reject',
  ADJUST: 'adjust',
  LOCK: 'lock',
  UNLOCK: 'unlock',
  RECONCILE: 'reconcile',
  EXPORT: 'export',
  IMPORT: 'import',
  ASSIGN_ROLE: 'assign_role',
  VIEW_ALL: 'view_all',
} as const

/**
 * Helper to create a permission code
 */
export function createPermission(
  resource: (typeof Resources)[keyof typeof Resources],
  action: (typeof Actions)[keyof typeof Actions]
): string {
  return `${resource}:${action}`
}

/**
 * Predefined permission codes for common operations
 */
export const Permissions = {
  // Product permissions
  PRODUCT_CREATE: createPermission(Resources.PRODUCT, Actions.CREATE),
  PRODUCT_READ: createPermission(Resources.PRODUCT, Actions.READ),
  PRODUCT_UPDATE: createPermission(Resources.PRODUCT, Actions.UPDATE),
  PRODUCT_DELETE: createPermission(Resources.PRODUCT, Actions.DELETE),
  PRODUCT_ENABLE: createPermission(Resources.PRODUCT, Actions.ENABLE),
  PRODUCT_DISABLE: createPermission(Resources.PRODUCT, Actions.DISABLE),

  // Category permissions
  CATEGORY_CREATE: createPermission(Resources.CATEGORY, Actions.CREATE),
  CATEGORY_READ: createPermission(Resources.CATEGORY, Actions.READ),
  CATEGORY_UPDATE: createPermission(Resources.CATEGORY, Actions.UPDATE),
  CATEGORY_DELETE: createPermission(Resources.CATEGORY, Actions.DELETE),

  // Customer permissions
  CUSTOMER_CREATE: createPermission(Resources.CUSTOMER, Actions.CREATE),
  CUSTOMER_READ: createPermission(Resources.CUSTOMER, Actions.READ),
  CUSTOMER_UPDATE: createPermission(Resources.CUSTOMER, Actions.UPDATE),
  CUSTOMER_DELETE: createPermission(Resources.CUSTOMER, Actions.DELETE),

  // Supplier permissions
  SUPPLIER_CREATE: createPermission(Resources.SUPPLIER, Actions.CREATE),
  SUPPLIER_READ: createPermission(Resources.SUPPLIER, Actions.READ),
  SUPPLIER_UPDATE: createPermission(Resources.SUPPLIER, Actions.UPDATE),
  SUPPLIER_DELETE: createPermission(Resources.SUPPLIER, Actions.DELETE),

  // Warehouse permissions
  WAREHOUSE_CREATE: createPermission(Resources.WAREHOUSE, Actions.CREATE),
  WAREHOUSE_READ: createPermission(Resources.WAREHOUSE, Actions.READ),
  WAREHOUSE_UPDATE: createPermission(Resources.WAREHOUSE, Actions.UPDATE),
  WAREHOUSE_DELETE: createPermission(Resources.WAREHOUSE, Actions.DELETE),
  WAREHOUSE_ENABLE: createPermission(Resources.WAREHOUSE, Actions.ENABLE),
  WAREHOUSE_DISABLE: createPermission(Resources.WAREHOUSE, Actions.DISABLE),

  // Inventory permissions
  INVENTORY_READ: createPermission(Resources.INVENTORY, Actions.READ),
  INVENTORY_ADJUST: createPermission(Resources.INVENTORY, Actions.ADJUST),
  INVENTORY_LOCK: createPermission(Resources.INVENTORY, Actions.LOCK),
  INVENTORY_UNLOCK: createPermission(Resources.INVENTORY, Actions.UNLOCK),

  // Sales order permissions
  SALES_ORDER_CREATE: createPermission(Resources.SALES_ORDER, Actions.CREATE),
  SALES_ORDER_READ: createPermission(Resources.SALES_ORDER, Actions.READ),
  SALES_ORDER_UPDATE: createPermission(Resources.SALES_ORDER, Actions.UPDATE),
  SALES_ORDER_DELETE: createPermission(Resources.SALES_ORDER, Actions.DELETE),
  SALES_ORDER_CONFIRM: createPermission(Resources.SALES_ORDER, Actions.CONFIRM),
  SALES_ORDER_CANCEL: createPermission(Resources.SALES_ORDER, Actions.CANCEL),
  SALES_ORDER_SHIP: createPermission(Resources.SALES_ORDER, Actions.SHIP),

  // Purchase order permissions
  PURCHASE_ORDER_CREATE: createPermission(Resources.PURCHASE_ORDER, Actions.CREATE),
  PURCHASE_ORDER_READ: createPermission(Resources.PURCHASE_ORDER, Actions.READ),
  PURCHASE_ORDER_UPDATE: createPermission(Resources.PURCHASE_ORDER, Actions.UPDATE),
  PURCHASE_ORDER_DELETE: createPermission(Resources.PURCHASE_ORDER, Actions.DELETE),
  PURCHASE_ORDER_CONFIRM: createPermission(Resources.PURCHASE_ORDER, Actions.CONFIRM),
  PURCHASE_ORDER_CANCEL: createPermission(Resources.PURCHASE_ORDER, Actions.CANCEL),
  PURCHASE_ORDER_RECEIVE: createPermission(Resources.PURCHASE_ORDER, Actions.RECEIVE),

  // Sales return permissions
  SALES_RETURN_CREATE: createPermission(Resources.SALES_RETURN, Actions.CREATE),
  SALES_RETURN_READ: createPermission(Resources.SALES_RETURN, Actions.READ),
  SALES_RETURN_APPROVE: createPermission(Resources.SALES_RETURN, Actions.APPROVE),
  SALES_RETURN_REJECT: createPermission(Resources.SALES_RETURN, Actions.REJECT),

  // Purchase return permissions
  PURCHASE_RETURN_CREATE: createPermission(Resources.PURCHASE_RETURN, Actions.CREATE),
  PURCHASE_RETURN_READ: createPermission(Resources.PURCHASE_RETURN, Actions.READ),
  PURCHASE_RETURN_SHIP: createPermission(Resources.PURCHASE_RETURN, Actions.SHIP),

  // Account receivable permissions
  ACCOUNT_RECEIVABLE_READ: createPermission(Resources.ACCOUNT_RECEIVABLE, Actions.READ),
  ACCOUNT_RECEIVABLE_RECONCILE: createPermission(Resources.ACCOUNT_RECEIVABLE, Actions.RECONCILE),

  // Account payable permissions
  ACCOUNT_PAYABLE_READ: createPermission(Resources.ACCOUNT_PAYABLE, Actions.READ),
  ACCOUNT_PAYABLE_RECONCILE: createPermission(Resources.ACCOUNT_PAYABLE, Actions.RECONCILE),

  // Receipt permissions
  RECEIPT_CREATE: createPermission(Resources.RECEIPT, Actions.CREATE),
  RECEIPT_READ: createPermission(Resources.RECEIPT, Actions.READ),

  // Payment permissions
  PAYMENT_CREATE: createPermission(Resources.PAYMENT, Actions.CREATE),
  PAYMENT_READ: createPermission(Resources.PAYMENT, Actions.READ),

  // Expense permissions
  EXPENSE_CREATE: createPermission(Resources.EXPENSE, Actions.CREATE),
  EXPENSE_READ: createPermission(Resources.EXPENSE, Actions.READ),
  EXPENSE_UPDATE: createPermission(Resources.EXPENSE, Actions.UPDATE),
  EXPENSE_DELETE: createPermission(Resources.EXPENSE, Actions.DELETE),

  // Income permissions
  INCOME_CREATE: createPermission(Resources.INCOME, Actions.CREATE),
  INCOME_READ: createPermission(Resources.INCOME, Actions.READ),
  INCOME_UPDATE: createPermission(Resources.INCOME, Actions.UPDATE),
  INCOME_DELETE: createPermission(Resources.INCOME, Actions.DELETE),

  // Report permissions
  REPORT_READ: createPermission(Resources.REPORT, Actions.READ),
  REPORT_EXPORT: createPermission(Resources.REPORT, Actions.EXPORT),

  // User management permissions
  USER_CREATE: createPermission(Resources.USER, Actions.CREATE),
  USER_READ: createPermission(Resources.USER, Actions.READ),
  USER_UPDATE: createPermission(Resources.USER, Actions.UPDATE),
  USER_DELETE: createPermission(Resources.USER, Actions.DELETE),
  USER_ASSIGN_ROLE: createPermission(Resources.USER, Actions.ASSIGN_ROLE),

  // Role management permissions
  ROLE_CREATE: createPermission(Resources.ROLE, Actions.CREATE),
  ROLE_READ: createPermission(Resources.ROLE, Actions.READ),
  ROLE_UPDATE: createPermission(Resources.ROLE, Actions.UPDATE),
  ROLE_DELETE: createPermission(Resources.ROLE, Actions.DELETE),

  // Tenant management permissions
  TENANT_CREATE: createPermission(Resources.TENANT, Actions.CREATE),
  TENANT_READ: createPermission(Resources.TENANT, Actions.READ),
  TENANT_UPDATE: createPermission(Resources.TENANT, Actions.UPDATE),
  TENANT_DELETE: createPermission(Resources.TENANT, Actions.DELETE),
} as const

/**
 * Route permission mappings for menu filtering
 * Maps route paths to required permissions (any of the listed permissions grants access)
 */
export const RoutePermissions: Record<string, string[]> = {
  // Dashboard - accessible to all authenticated users
  '/': [],

  // Catalog module
  '/catalog': [Permissions.PRODUCT_READ, Permissions.CATEGORY_READ],
  '/catalog/products': [Permissions.PRODUCT_READ],
  '/catalog/products/new': [Permissions.PRODUCT_CREATE],
  '/catalog/products/:id/edit': [Permissions.PRODUCT_UPDATE],
  '/catalog/categories': [Permissions.CATEGORY_READ],

  // Partner module
  '/partner': [Permissions.CUSTOMER_READ, Permissions.SUPPLIER_READ, Permissions.WAREHOUSE_READ],
  '/partner/customers': [Permissions.CUSTOMER_READ],
  '/partner/customers/new': [Permissions.CUSTOMER_CREATE],
  '/partner/customers/:id/edit': [Permissions.CUSTOMER_UPDATE],
  '/partner/suppliers': [Permissions.SUPPLIER_READ],
  '/partner/suppliers/new': [Permissions.SUPPLIER_CREATE],
  '/partner/suppliers/:id/edit': [Permissions.SUPPLIER_UPDATE],
  '/partner/warehouses': [Permissions.WAREHOUSE_READ],
  '/partner/warehouses/new': [Permissions.WAREHOUSE_CREATE],
  '/partner/warehouses/:id/edit': [Permissions.WAREHOUSE_UPDATE],

  // Inventory module
  '/inventory': [Permissions.INVENTORY_READ],
  '/inventory/stock': [Permissions.INVENTORY_READ],
  '/inventory/stock/:id': [Permissions.INVENTORY_READ],
  '/inventory/stock/:id/transactions': [Permissions.INVENTORY_READ],
  '/inventory/adjust': [Permissions.INVENTORY_ADJUST],
  '/inventory/stock-taking': [Permissions.INVENTORY_ADJUST],
  '/inventory/stock-taking/new': [Permissions.INVENTORY_ADJUST],
  '/inventory/stock-taking/:id': [Permissions.INVENTORY_READ],
  '/inventory/stock-taking/:id/execute': [Permissions.INVENTORY_ADJUST],

  // Trade module
  '/trade': [Permissions.SALES_ORDER_READ, Permissions.PURCHASE_ORDER_READ],
  '/trade/sales': [Permissions.SALES_ORDER_READ],
  '/trade/sales/new': [Permissions.SALES_ORDER_CREATE],
  '/trade/sales/:id': [Permissions.SALES_ORDER_READ],
  '/trade/sales/:id/edit': [Permissions.SALES_ORDER_UPDATE],
  '/trade/purchase': [Permissions.PURCHASE_ORDER_READ],
  '/trade/purchase/new': [Permissions.PURCHASE_ORDER_CREATE],
  '/trade/purchase/:id': [Permissions.PURCHASE_ORDER_READ],
  '/trade/purchase/:id/edit': [Permissions.PURCHASE_ORDER_UPDATE],
  '/trade/purchase/:id/receive': [Permissions.PURCHASE_ORDER_RECEIVE],

  // Finance module
  '/finance': [
    Permissions.ACCOUNT_RECEIVABLE_READ,
    Permissions.ACCOUNT_PAYABLE_READ,
    Permissions.EXPENSE_READ,
    Permissions.INCOME_READ,
  ],
  '/finance/receivables': [Permissions.ACCOUNT_RECEIVABLE_READ],
  '/finance/payables': [Permissions.ACCOUNT_PAYABLE_READ],
  '/finance/receipts/new': [Permissions.RECEIPT_CREATE],
  '/finance/receipts/:id/reconcile': [Permissions.ACCOUNT_RECEIVABLE_RECONCILE],
  '/finance/payments/new': [Permissions.PAYMENT_CREATE],
  '/finance/payments/:id/reconcile': [Permissions.ACCOUNT_PAYABLE_RECONCILE],
  '/finance/expenses': [Permissions.EXPENSE_READ],
  '/finance/expenses/new': [Permissions.EXPENSE_CREATE],
  '/finance/expenses/:id/edit': [Permissions.EXPENSE_UPDATE],
  '/finance/incomes': [Permissions.INCOME_READ],
  '/finance/incomes/new': [Permissions.INCOME_CREATE],
  '/finance/incomes/:id/edit': [Permissions.INCOME_UPDATE],
  '/finance/cashflow': [Permissions.EXPENSE_READ, Permissions.INCOME_READ],

  // Report module
  '/report': [Permissions.REPORT_READ],
  '/report/sales': [Permissions.REPORT_READ],
  '/report/ranking': [Permissions.REPORT_READ],
  '/report/inventory-turnover': [Permissions.REPORT_READ],
  '/report/profit-loss': [Permissions.REPORT_READ],
  '/report/cash-flow': [Permissions.REPORT_READ],

  // System module
  '/system': [Permissions.USER_READ, Permissions.ROLE_READ],
  '/system/users': [Permissions.USER_READ],
  '/system/roles': [Permissions.ROLE_READ],
  '/system/permissions': [Permissions.ROLE_READ],
  '/system/payment-settings': [Permissions.TENANT_UPDATE],
}

/**
 * Check if a user has access to a route based on permissions
 * @param userPermissions - Array of user's permission codes
 * @param routePath - The route path to check
 * @returns true if user has at least one required permission (or route has no requirements)
 */
export function hasRouteAccess(userPermissions: string[] | undefined, routePath: string): boolean {
  // If no permissions defined for route, allow access (authenticated users only)
  const requiredPermissions = RoutePermissions[routePath]
  if (!requiredPermissions || requiredPermissions.length === 0) {
    return true
  }

  // If user has no permissions, deny access to permission-protected routes
  if (!userPermissions || userPermissions.length === 0) {
    return false
  }

  // Check if user has ANY of the required permissions
  return requiredPermissions.some((perm) => userPermissions.includes(perm))
}

/**
 * Get all permissions required for a route (for debugging/display)
 */
export function getRoutePermissions(routePath: string): string[] {
  return RoutePermissions[routePath] || []
}
