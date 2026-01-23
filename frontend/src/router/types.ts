import type { ReactNode } from 'react'

/**
 * Route metadata for navigation and access control
 */
export interface RouteMeta {
  /** Display title in breadcrumb/menu */
  title: string
  /** Icon component name for menu */
  icon?: string
  /** Required permissions to access this route */
  permissions?: string[]
  /** Whether this route requires authentication (default: true) */
  requiresAuth?: boolean
  /** Hide from sidebar menu */
  hideInMenu?: boolean
  /** Hide from breadcrumb */
  hideInBreadcrumb?: boolean
  /** Keep alive (cache component) */
  keepAlive?: boolean
  /** External link URL (if route links externally) */
  externalLink?: string
  /** Sort order in menu */
  order?: number
}

/**
 * Application route configuration
 */
export interface AppRoute {
  /** Route path */
  path: string
  /** Route element/component */
  element?: ReactNode
  /** Child routes */
  children?: AppRoute[]
  /** Route metadata */
  meta?: RouteMeta
  /** Index route (default child) */
  index?: boolean
  /** Redirect to another path */
  redirect?: string
}

/**
 * Breadcrumb item
 */
export interface BreadcrumbItem {
  path: string
  title: string
}

/**
 * Navigation menu item (derived from routes)
 */
export interface MenuItem {
  key: string
  path: string
  title: string
  icon?: string
  children?: MenuItem[]
  order?: number
}
