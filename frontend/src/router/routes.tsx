import { Navigate, type RouteObject } from 'react-router-dom'
import { lazyLoad } from './lazyLoad'
import { AuthGuard, GuestGuard } from './guards'
import { MainLayout } from '@/components/layout'
import type { AppRoute } from './types'
import { Permissions } from '@/config/permissions'

/**
 * Application route configuration
 *
 * Route structure mirrors the ERP domain modules:
 * - Dashboard (home)
 * - Catalog (products, categories)
 * - Partner (customers, suppliers, warehouses)
 * - Inventory (stock, adjustments, stocktaking)
 * - Trade (sales, purchases, returns)
 * - Finance (receivables, payables, expenses)
 */

// Lazy-loaded page components
const DashboardPage = () => lazyLoad(() => import('@/pages/Dashboard'))
const LoginPage = () => lazyLoad(() => import('@/pages/Login'))
const NotFoundPage = () => lazyLoad(() => import('@/pages/NotFound'))
const ForbiddenPage = () => lazyLoad(() => import('@/pages/Forbidden'))

// Catalog module
const ProductsPage = () => lazyLoad(() => import('@/pages/catalog/Products'))
const ProductNewPage = () => lazyLoad(() => import('@/pages/catalog/ProductNew'))
const ProductEditPage = () => lazyLoad(() => import('@/pages/catalog/ProductEdit'))
const CategoriesPage = () => lazyLoad(() => import('@/pages/catalog/Categories'))

// Partner module
const CustomersPage = () => lazyLoad(() => import('@/pages/partner/Customers'))
const CustomerNewPage = () => lazyLoad(() => import('@/pages/partner/CustomerNew'))
const CustomerEditPage = () => lazyLoad(() => import('@/pages/partner/CustomerEdit'))
const SuppliersPage = () => lazyLoad(() => import('@/pages/partner/Suppliers'))
const SupplierNewPage = () => lazyLoad(() => import('@/pages/partner/SupplierNew'))
const SupplierEditPage = () => lazyLoad(() => import('@/pages/partner/SupplierEdit'))
const WarehousesPage = () => lazyLoad(() => import('@/pages/partner/Warehouses'))
const WarehouseNewPage = () => lazyLoad(() => import('@/pages/partner/WarehouseNew'))
const WarehouseEditPage = () => lazyLoad(() => import('@/pages/partner/WarehouseEdit'))
const CustomerBalancePage = () => lazyLoad(() => import('@/pages/partner/CustomerBalance'))

// Inventory module
const StockListPage = () => lazyLoad(() => import('@/pages/inventory/StockList'))
const StockDetailPage = () => lazyLoad(() => import('@/pages/inventory/StockDetail'))
const StockTransactionsPage = () => lazyLoad(() => import('@/pages/inventory/StockTransactions'))
const StockAdjustPage = () => lazyLoad(() => import('@/pages/inventory/StockAdjust'))
const StockTakingListPage = () => lazyLoad(() => import('@/pages/inventory/StockTakingList'))
const StockTakingCreatePage = () => lazyLoad(() => import('@/pages/inventory/StockTakingCreate'))
const StockTakingExecutePage = () => lazyLoad(() => import('@/pages/inventory/StockTakingExecute'))

// Trade module
const SalesOrdersPage = () => lazyLoad(() => import('@/pages/trade/SalesOrders'))
const SalesOrderNewPage = () => lazyLoad(() => import('@/pages/trade/SalesOrderNew'))
const SalesOrderEditPage = () => lazyLoad(() => import('@/pages/trade/SalesOrderEdit'))
const SalesOrderDetailPage = () => lazyLoad(() => import('@/pages/trade/SalesOrderDetail'))
const PurchaseOrdersPage = () => lazyLoad(() => import('@/pages/trade/PurchaseOrders'))
const PurchaseOrderNewPage = () => lazyLoad(() => import('@/pages/trade/PurchaseOrderNew'))
const PurchaseOrderReceivePage = () => lazyLoad(() => import('@/pages/trade/PurchaseOrderReceive'))
const SalesReturnsPage = () => lazyLoad(() => import('@/pages/trade/SalesReturns'))
const SalesReturnNewPage = () => lazyLoad(() => import('@/pages/trade/SalesReturnNew'))
const SalesReturnApprovalPage = () => lazyLoad(() => import('@/pages/trade/SalesReturnApproval'))
const PurchaseReturnsPage = () => lazyLoad(() => import('@/pages/trade/PurchaseReturns'))
const PurchaseReturnNewPage = () => lazyLoad(() => import('@/pages/trade/PurchaseReturnNew'))

// Finance module
const ReceivablesPage = () => lazyLoad(() => import('@/pages/finance/Receivables'))
const PayablesPage = () => lazyLoad(() => import('@/pages/finance/Payables'))
const ReceiptVoucherNewPage = () => lazyLoad(() => import('@/pages/finance/ReceiptVoucherNew'))
const PaymentVoucherNewPage = () => lazyLoad(() => import('@/pages/finance/PaymentVoucherNew'))
const ReceiptReconcilePage = () => lazyLoad(() => import('@/pages/finance/ReceiptReconcile'))
const PaymentReconcilePage = () => lazyLoad(() => import('@/pages/finance/PaymentReconcile'))
const ExpensesPage = () => lazyLoad(() => import('@/pages/finance/Expenses'))
const ExpenseFormPage = () => lazyLoad(() => import('@/pages/finance/ExpenseForm'))
const OtherIncomesPage = () => lazyLoad(() => import('@/pages/finance/OtherIncomes'))
const OtherIncomeFormPage = () => lazyLoad(() => import('@/pages/finance/OtherIncomeForm'))
const CashFlowPage = () => lazyLoad(() => import('@/pages/finance/CashFlow'))

// Report module
const SalesReportPage = () => lazyLoad(() => import('@/pages/report/SalesReport'))
const SalesRankingPage = () => lazyLoad(() => import('@/pages/report/SalesRanking'))
const InventoryTurnoverPage = () => lazyLoad(() => import('@/pages/report/InventoryTurnover'))
const ProfitLossPage = () => lazyLoad(() => import('@/pages/report/ProfitLoss'))
const CashFlowReportPage = () => lazyLoad(() => import('@/pages/report/CashFlowReport'))

// System module
const UsersPage = () => lazyLoad(() => import('@/pages/system/Users'))
const RolesPage = () => lazyLoad(() => import('@/pages/system/Roles'))
const PermissionsPage = () => lazyLoad(() => import('@/pages/system/Permissions'))

/**
 * Application routes with metadata
 * Routes are organized into two groups:
 * 1. Public routes (login, error pages) - no layout
 * 2. Protected routes - wrapped in MainLayout
 *
 * Permission requirements:
 * - Routes without permissions array are accessible to all authenticated users
 * - Routes with permissions array require ANY of the listed permissions
 */
export const appRoutes: AppRoute[] = [
  // Dashboard (home) - accessible to all authenticated users
  {
    path: '/',
    meta: {
      title: 'Dashboard',
      icon: 'IconHome',
      order: 0,
    },
  },

  // Catalog module
  {
    path: '/catalog',
    meta: {
      title: 'Catalog',
      icon: 'IconGridView',
      order: 1,
      permissions: [Permissions.PRODUCT_READ, Permissions.CATEGORY_READ],
    },
    children: [
      {
        path: '/catalog',
        redirect: '/catalog/products',
      },
      {
        path: '/catalog/products',
        meta: {
          title: 'Products',
          icon: 'IconGridView',
          order: 1,
          permissions: [Permissions.PRODUCT_READ],
        },
      },
      {
        path: '/catalog/categories',
        meta: {
          title: 'Categories',
          icon: 'IconTreeTriangleDown',
          order: 2,
          permissions: [Permissions.CATEGORY_READ],
        },
      },
    ],
  },

  // Partner module
  {
    path: '/partner',
    meta: {
      title: 'Partners',
      icon: 'IconUserGroup',
      order: 2,
      permissions: [
        Permissions.CUSTOMER_READ,
        Permissions.SUPPLIER_READ,
        Permissions.WAREHOUSE_READ,
      ],
    },
    children: [
      {
        path: '/partner',
        redirect: '/partner/customers',
      },
      {
        path: '/partner/customers',
        meta: {
          title: 'Customers',
          icon: 'IconUserGroup',
          order: 1,
          permissions: [Permissions.CUSTOMER_READ],
        },
      },
      {
        path: '/partner/suppliers',
        meta: {
          title: 'Suppliers',
          icon: 'IconUserCardVideo',
          order: 2,
          permissions: [Permissions.SUPPLIER_READ],
        },
      },
      {
        path: '/partner/warehouses',
        meta: {
          title: 'Warehouses',
          icon: 'IconInbox',
          order: 3,
          permissions: [Permissions.WAREHOUSE_READ],
        },
      },
    ],
  },

  // Inventory module
  {
    path: '/inventory',
    meta: {
      title: 'Inventory',
      icon: 'IconList',
      order: 3,
      permissions: [Permissions.INVENTORY_READ],
    },
    children: [
      {
        path: '/inventory',
        redirect: '/inventory/stock',
      },
      {
        path: '/inventory/stock',
        meta: {
          title: 'Stock List',
          icon: 'IconList',
          order: 1,
          permissions: [Permissions.INVENTORY_READ],
        },
      },
      {
        path: '/inventory/stock-taking',
        meta: {
          title: 'Stock Taking',
          icon: 'IconCheckList',
          order: 2,
          permissions: [Permissions.INVENTORY_ADJUST],
        },
      },
    ],
  },

  // Trade module
  {
    path: '/trade',
    meta: {
      title: 'Trade',
      icon: 'IconSend',
      order: 4,
      permissions: [Permissions.SALES_ORDER_READ, Permissions.PURCHASE_ORDER_READ],
    },
    children: [
      {
        path: '/trade',
        redirect: '/trade/sales',
      },
      {
        path: '/trade/sales',
        meta: {
          title: 'Sales Orders',
          icon: 'IconSend',
          order: 1,
          permissions: [Permissions.SALES_ORDER_READ],
        },
      },
      {
        path: '/trade/purchase',
        meta: {
          title: 'Purchase Orders',
          icon: 'IconDownload',
          order: 2,
          permissions: [Permissions.PURCHASE_ORDER_READ],
        },
      },
      {
        path: '/trade/sales-returns',
        meta: {
          title: 'Sales Returns',
          icon: 'IconUndo',
          order: 3,
          permissions: [Permissions.SALES_ORDER_READ],
        },
      },
      {
        path: '/trade/purchase-returns',
        meta: {
          title: 'Purchase Returns',
          icon: 'IconRedo',
          order: 4,
          permissions: [Permissions.PURCHASE_ORDER_READ],
        },
      },
    ],
  },

  // Finance module
  {
    path: '/finance',
    meta: {
      title: 'Finance',
      icon: 'IconPriceTag',
      order: 5,
      permissions: [
        Permissions.ACCOUNT_RECEIVABLE_READ,
        Permissions.ACCOUNT_PAYABLE_READ,
        Permissions.EXPENSE_READ,
        Permissions.INCOME_READ,
      ],
    },
    children: [
      {
        path: '/finance',
        redirect: '/finance/receivables',
      },
      {
        path: '/finance/receivables',
        meta: {
          title: 'Receivables',
          icon: 'IconPriceTag',
          order: 1,
          permissions: [Permissions.ACCOUNT_RECEIVABLE_READ],
        },
      },
      {
        path: '/finance/payables',
        meta: {
          title: 'Payables',
          icon: 'IconCreditCard',
          order: 2,
          permissions: [Permissions.ACCOUNT_PAYABLE_READ],
        },
      },
      {
        path: '/finance/expenses',
        meta: {
          title: 'Expenses',
          icon: 'IconMinus',
          order: 3,
          permissions: [Permissions.EXPENSE_READ],
        },
      },
      {
        path: '/finance/incomes',
        meta: {
          title: 'Other Income',
          icon: 'IconPlus',
          order: 4,
          permissions: [Permissions.INCOME_READ],
        },
      },
      {
        path: '/finance/cashflow',
        meta: {
          title: 'Cash Flow',
          icon: 'IconHistory',
          order: 5,
          permissions: [Permissions.EXPENSE_READ, Permissions.INCOME_READ],
        },
      },
    ],
  },

  // Report module
  {
    path: '/report',
    meta: {
      title: 'Reports',
      icon: 'IconChartLine',
      order: 6,
      permissions: [Permissions.REPORT_READ],
    },
    children: [
      {
        path: '/report',
        redirect: '/report/sales',
      },
      {
        path: '/report/sales',
        meta: {
          title: 'Sales Report',
          icon: 'IconChartLine',
          order: 1,
          permissions: [Permissions.REPORT_READ],
        },
      },
      {
        path: '/report/ranking',
        meta: {
          title: 'Sales Ranking',
          icon: 'IconBarChartHStroked',
          order: 2,
          permissions: [Permissions.REPORT_READ],
        },
      },
      {
        path: '/report/inventory-turnover',
        meta: {
          title: 'Inventory Turnover',
          icon: 'IconRefresh',
          order: 3,
          permissions: [Permissions.REPORT_READ],
        },
      },
      {
        path: '/report/profit-loss',
        meta: {
          title: 'Profit & Loss',
          icon: 'IconPieChartStroked',
          order: 4,
          permissions: [Permissions.REPORT_READ],
        },
      },
      {
        path: '/report/cash-flow',
        meta: {
          title: 'Cash Flow',
          icon: 'IconHistory',
          order: 5,
          permissions: [Permissions.REPORT_READ],
        },
      },
    ],
  },

  // System module
  {
    path: '/system',
    meta: {
      title: 'System',
      icon: 'IconSetting',
      order: 7,
      permissions: [Permissions.USER_READ, Permissions.ROLE_READ],
    },
    children: [
      {
        path: '/system',
        redirect: '/system/users',
      },
      {
        path: '/system/users',
        meta: {
          title: 'Users',
          icon: 'IconUser',
          order: 1,
          permissions: [Permissions.USER_READ],
        },
      },
      {
        path: '/system/roles',
        meta: {
          title: 'Roles',
          icon: 'IconUserGroup',
          order: 2,
          permissions: [Permissions.ROLE_READ],
        },
      },
      {
        path: '/system/permissions',
        meta: {
          title: 'Permissions',
          icon: 'IconKey',
          order: 3,
          permissions: [Permissions.ROLE_READ],
        },
      },
    ],
  },
]

/**
 * Get route elements for protected routes (within MainLayout)
 */
function getProtectedRouteElement(path: string): React.ReactNode {
  switch (path) {
    case '/':
      return DashboardPage()
    case '/catalog/products':
      return ProductsPage()
    case '/catalog/products/new':
      return ProductNewPage()
    case '/catalog/products/:id/edit':
      return ProductEditPage()
    case '/catalog/categories':
      return CategoriesPage()
    case '/partner/customers':
      return CustomersPage()
    case '/partner/suppliers':
      return SuppliersPage()
    case '/partner/warehouses':
      return WarehousesPage()
    case '/inventory/stock':
      return StockListPage()
    case '/inventory/stock-taking':
      return StockTakingListPage()
    case '/trade/sales':
      return SalesOrdersPage()
    case '/trade/purchase':
      return PurchaseOrdersPage()
    case '/trade/sales-returns':
      return SalesReturnsPage()
    case '/trade/purchase-returns':
      return PurchaseReturnsPage()
    case '/finance/receivables':
      return ReceivablesPage()
    case '/finance/payables':
      return PayablesPage()
    case '/finance/expenses':
      return ExpensesPage()
    case '/finance/incomes':
      return OtherIncomesPage()
    case '/finance/cashflow':
      return CashFlowPage()
    case '/report/sales':
      return SalesReportPage()
    case '/report/ranking':
      return SalesRankingPage()
    case '/report/inventory-turnover':
      return InventoryTurnoverPage()
    case '/report/profit-loss':
      return ProfitLossPage()
    case '/report/cash-flow':
      return CashFlowReportPage()
    case '/system/users':
      return UsersPage()
    case '/system/roles':
      return RolesPage()
    case '/system/permissions':
      return PermissionsPage()
    default:
      return null
  }
}

/**
 * Convert AppRoute to react-router RouteObject for nested layout routes
 */
function convertToNestedRouteObject(route: AppRoute): RouteObject | null {
  if (route.redirect) {
    return {
      path: route.path,
      element: <Navigate to={route.redirect} replace />,
    }
  }

  const element = getProtectedRouteElement(route.path || '')

  // For parent routes without direct element, only handle children
  if (route.children) {
    const childRoutes = route.children
      .map(convertToNestedRouteObject)
      .filter((r): r is RouteObject => r !== null)

    if (element) {
      return {
        path: route.path,
        element,
        children: childRoutes,
      }
    }

    // Return children directly for grouping routes
    return {
      path: route.path,
      children: childRoutes,
    }
  }

  if (!element) {
    return null
  }

  return {
    path: route.path,
    element,
  }
}

/**
 * Get routes in react-router format
 * Uses a layout route pattern for protected routes
 */
export function getRouteObjects(): RouteObject[] {
  // Build nested routes for protected area
  const protectedChildRoutes: RouteObject[] = []

  for (const route of appRoutes) {
    if (route.path === '/') {
      // Dashboard as index route
      protectedChildRoutes.push({
        index: true,
        element: DashboardPage(),
      })
    } else if (route.children) {
      // Module with children
      const childRoutes = route.children
        .map(convertToNestedRouteObject)
        .filter((r): r is RouteObject => r !== null)

      // Add module-specific detail routes (not in menu)
      if (route.path === '/catalog') {
        childRoutes.push(
          { path: 'products/new', element: ProductNewPage() },
          { path: 'products/:id/edit', element: ProductEditPage() }
        )
      }

      // Add partner module detail routes (not in menu)
      if (route.path === '/partner') {
        childRoutes.push(
          { path: 'customers/new', element: CustomerNewPage() },
          { path: 'customers/:id/edit', element: CustomerEditPage() },
          { path: 'customers/:id/balance', element: CustomerBalancePage() },
          { path: 'suppliers/new', element: SupplierNewPage() },
          { path: 'suppliers/:id/edit', element: SupplierEditPage() },
          { path: 'warehouses/new', element: WarehouseNewPage() },
          { path: 'warehouses/:id/edit', element: WarehouseEditPage() }
        )
      }

      // Add inventory module detail routes (not in menu)
      if (route.path === '/inventory') {
        childRoutes.push(
          { path: 'stock/:id', element: StockDetailPage() },
          { path: 'stock/:id/transactions', element: StockTransactionsPage() },
          { path: 'adjust', element: StockAdjustPage() },
          { path: 'stock-taking', element: StockTakingListPage() },
          { path: 'stock-taking/new', element: StockTakingCreatePage() },
          { path: 'stock-taking/:id', element: StockTakingExecutePage() },
          { path: 'stock-taking/:id/execute', element: StockTakingExecutePage() }
        )
      }

      // Add trade module detail routes (not in menu)
      if (route.path === '/trade') {
        childRoutes.push(
          { path: 'sales/new', element: SalesOrderNewPage() },
          { path: 'sales/:id', element: SalesOrderDetailPage() },
          { path: 'sales/:id/edit', element: SalesOrderEditPage() },
          // Purchase order routes
          { path: 'purchase/new', element: PurchaseOrderNewPage() },
          { path: 'purchase/:id', element: PurchaseOrdersPage() }, // TODO: detail page
          { path: 'purchase/:id/edit', element: PurchaseOrdersPage() }, // TODO: edit page
          { path: 'purchase/:id/receive', element: PurchaseOrderReceivePage() },
          // Sales returns routes
          { path: 'sales-returns', element: SalesReturnsPage() },
          { path: 'sales-returns/new', element: SalesReturnNewPage() },
          { path: 'sales-returns/approval', element: SalesReturnApprovalPage() },
          // Purchase returns routes
          { path: 'purchase-returns', element: PurchaseReturnsPage() },
          { path: 'purchase-returns/new', element: PurchaseReturnNewPage() }
        )
      }

      // Add finance module detail routes (not in menu)
      if (route.path === '/finance') {
        childRoutes.push(
          { path: 'receipts/new', element: ReceiptVoucherNewPage() },
          { path: 'receipts/:id/reconcile', element: ReceiptReconcilePage() },
          { path: 'payments/new', element: PaymentVoucherNewPage() },
          { path: 'payments/:id/reconcile', element: PaymentReconcilePage() },
          { path: 'expenses/new', element: ExpenseFormPage() },
          { path: 'expenses/:id/edit', element: ExpenseFormPage() },
          { path: 'incomes/new', element: OtherIncomeFormPage() },
          { path: 'incomes/:id/edit', element: OtherIncomeFormPage() }
        )
      }

      protectedChildRoutes.push({
        path: route.path?.replace(/^\//, ''), // Remove leading slash for relative path
        children: childRoutes.map((child) => ({
          ...child,
          path: child.path?.replace(/^\/[^/]+\//, ''), // Make path relative
        })),
      })
    }
  }

  return [
    // Public routes (no layout)
    {
      path: '/login',
      element: <GuestGuard>{LoginPage()}</GuestGuard>,
    },
    {
      path: '/403',
      element: ForbiddenPage(),
    },
    {
      path: '/404',
      element: NotFoundPage(),
    },

    // Protected routes (with layout)
    {
      path: '/',
      element: (
        <AuthGuard>
          <MainLayout />
        </AuthGuard>
      ),
      children: protectedChildRoutes,
    },

    // Catch-all redirect
    {
      path: '*',
      element: <Navigate to="/404" replace />,
    },
  ]
}

/**
 * Flatten routes for menu generation
 */
export function flattenRoutes(routes: AppRoute[] = appRoutes): AppRoute[] {
  const result: AppRoute[] = []

  for (const route of routes) {
    result.push(route)
    if (route.children) {
      result.push(...flattenRoutes(route.children))
    }
  }

  return result
}

/**
 * Find route by path
 */
export function findRouteByPath(
  path: string,
  routes: AppRoute[] = appRoutes
): AppRoute | undefined {
  for (const route of routes) {
    if (route.path === path) {
      return route
    }
    if (route.children) {
      const found = findRouteByPath(path, route.children)
      if (found) {
        return found
      }
    }
  }
  return undefined
}

/**
 * Get breadcrumb items for a path
 */
export function getBreadcrumbs(path: string): { path: string; title: string }[] {
  const breadcrumbs: { path: string; title: string }[] = []
  const segments = path.split('/').filter(Boolean)
  let currentPath = ''

  for (const segment of segments) {
    currentPath += `/${segment}`
    const route = findRouteByPath(currentPath)
    if (route?.meta?.title && !route.meta.hideInBreadcrumb) {
      breadcrumbs.push({
        path: currentPath,
        title: route.meta.title,
      })
    }
  }

  return breadcrumbs
}
