import { Navigate, type RouteObject } from 'react-router-dom'
import { lazyLoad } from './lazyLoad'
import { AuthGuard, GuestGuard } from './guards'
import type { AppRoute } from './types'

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
const CategoriesPage = () => lazyLoad(() => import('@/pages/catalog/Categories'))

// Partner module
const CustomersPage = () => lazyLoad(() => import('@/pages/partner/Customers'))
const SuppliersPage = () => lazyLoad(() => import('@/pages/partner/Suppliers'))
const WarehousesPage = () => lazyLoad(() => import('@/pages/partner/Warehouses'))

// Inventory module
const StockListPage = () => lazyLoad(() => import('@/pages/inventory/StockList'))

// Trade module
const SalesOrdersPage = () => lazyLoad(() => import('@/pages/trade/SalesOrders'))
const PurchaseOrdersPage = () => lazyLoad(() => import('@/pages/trade/PurchaseOrders'))

// Finance module
const ReceivablesPage = () => lazyLoad(() => import('@/pages/finance/Receivables'))
const PayablesPage = () => lazyLoad(() => import('@/pages/finance/Payables'))

/**
 * Application routes with metadata
 */
export const appRoutes: AppRoute[] = [
  // Public routes (no auth required)
  {
    path: '/login',
    element: <GuestGuard>{LoginPage()}</GuestGuard>,
    meta: {
      title: 'Login',
      requiresAuth: false,
      hideInMenu: true,
    },
  },

  // Protected routes (require auth)
  {
    path: '/',
    element: <AuthGuard>{DashboardPage()}</AuthGuard>,
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
    },
    children: [
      {
        path: '/catalog',
        redirect: '/catalog/products',
      },
      {
        path: '/catalog/products',
        element: <AuthGuard meta={{ title: 'Products' }}>{ProductsPage()}</AuthGuard>,
        meta: {
          title: 'Products',
          icon: 'IconGridView',
          order: 1,
        },
      },
      {
        path: '/catalog/categories',
        element: <AuthGuard meta={{ title: 'Categories' }}>{CategoriesPage()}</AuthGuard>,
        meta: {
          title: 'Categories',
          icon: 'IconTreeTriangleDown',
          order: 2,
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
    },
    children: [
      {
        path: '/partner',
        redirect: '/partner/customers',
      },
      {
        path: '/partner/customers',
        element: <AuthGuard meta={{ title: 'Customers' }}>{CustomersPage()}</AuthGuard>,
        meta: {
          title: 'Customers',
          icon: 'IconUserGroup',
          order: 1,
        },
      },
      {
        path: '/partner/suppliers',
        element: <AuthGuard meta={{ title: 'Suppliers' }}>{SuppliersPage()}</AuthGuard>,
        meta: {
          title: 'Suppliers',
          icon: 'IconUserCardVideo',
          order: 2,
        },
      },
      {
        path: '/partner/warehouses',
        element: <AuthGuard meta={{ title: 'Warehouses' }}>{WarehousesPage()}</AuthGuard>,
        meta: {
          title: 'Warehouses',
          icon: 'IconInbox',
          order: 3,
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
    },
    children: [
      {
        path: '/inventory',
        redirect: '/inventory/stock',
      },
      {
        path: '/inventory/stock',
        element: <AuthGuard meta={{ title: 'Stock List' }}>{StockListPage()}</AuthGuard>,
        meta: {
          title: 'Stock List',
          icon: 'IconList',
          order: 1,
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
    },
    children: [
      {
        path: '/trade',
        redirect: '/trade/sales',
      },
      {
        path: '/trade/sales',
        element: <AuthGuard meta={{ title: 'Sales Orders' }}>{SalesOrdersPage()}</AuthGuard>,
        meta: {
          title: 'Sales Orders',
          icon: 'IconSend',
          order: 1,
        },
      },
      {
        path: '/trade/purchases',
        element: <AuthGuard meta={{ title: 'Purchase Orders' }}>{PurchaseOrdersPage()}</AuthGuard>,
        meta: {
          title: 'Purchase Orders',
          icon: 'IconDownload',
          order: 2,
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
    },
    children: [
      {
        path: '/finance',
        redirect: '/finance/receivables',
      },
      {
        path: '/finance/receivables',
        element: <AuthGuard meta={{ title: 'Receivables' }}>{ReceivablesPage()}</AuthGuard>,
        meta: {
          title: 'Receivables',
          icon: 'IconPriceTag',
          order: 1,
        },
      },
      {
        path: '/finance/payables',
        element: <AuthGuard meta={{ title: 'Payables' }}>{PayablesPage()}</AuthGuard>,
        meta: {
          title: 'Payables',
          icon: 'IconCreditCard',
          order: 2,
        },
      },
    ],
  },

  // Error pages
  {
    path: '/403',
    element: ForbiddenPage(),
    meta: {
      title: 'Access Denied',
      requiresAuth: false,
      hideInMenu: true,
    },
  },
  {
    path: '/404',
    element: NotFoundPage(),
    meta: {
      title: 'Not Found',
      requiresAuth: false,
      hideInMenu: true,
    },
  },
  {
    path: '*',
    element: <Navigate to="/404" replace />,
  },
]

/**
 * Convert AppRoute to react-router RouteObject
 * Handles redirect and nested routes
 */
function convertToRouteObject(route: AppRoute): RouteObject {
  // Handle index routes separately due to React Router's discriminated union type
  if (route.index) {
    return {
      index: true,
      element: route.element,
    }
  }

  const routeObject: RouteObject = {
    path: route.path,
  }

  if (route.redirect) {
    routeObject.element = <Navigate to={route.redirect} replace />
  } else if (route.element) {
    routeObject.element = route.element
  }

  if (route.children) {
    routeObject.children = route.children.map(convertToRouteObject)
  }

  return routeObject
}

/**
 * Get routes in react-router format
 */
export function getRouteObjects(): RouteObject[] {
  return appRoutes.map(convertToRouteObject)
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
