import { useCallback, useMemo } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { Layout, Nav } from '@douyinfe/semi-ui-19'
import { useTranslation } from 'react-i18next'
import {
  // Dashboard
  IconHome,
  // Catalog module
  IconApps,
  IconBox,
  IconTreeTriangleDown,
  // Partner module
  IconUserGroup,
  IconUser,
  IconBriefcase,
  IconInbox,
  // Inventory module
  IconArchive,
  IconList,
  IconCheckList,
  IconAlertTriangle,
  // Trade module
  IconCart,
  IconSend,
  IconDownload,
  IconUndo,
  IconRedo,
  // Finance module
  IconCoinMoneyStroked,
  IconMoneyExchangeStroked,
  IconCreditCard,
  IconMinusCircle,
  IconPlusCircle,
  IconHistory,
  // Reports module
  IconPieChartStroked,
  IconLineChartStroked,
  IconBarChartHStroked,
  IconRefresh,
  IconCandlestickChartStroked,
  // System module
  IconSetting,
  IconMember,
  IconKey,
  IconLink,
  IconSync,
  IconConnectionPoint1,
  // Logo
  IconGridView,
} from '@douyinfe/semi-icons'

import { useAppStore, useAuthStore } from '@/store'
import { appRoutes } from '@/router/routes'
import type { AppRoute } from '@/router/types'

import './Sidebar.css'

const { Sider } = Layout

/**
 * Icon mapping from string names to Semi Design icons
 * Each menu item should have a unique, meaningful icon
 */
const iconMap: Record<string, React.ReactNode> = {
  // Dashboard
  IconHome: <IconHome />,

  // Catalog module
  IconApps: <IconApps />,
  IconBox: <IconBox />,
  IconTreeTriangleDown: <IconTreeTriangleDown />,

  // Partner module
  IconUserGroup: <IconUserGroup />,
  IconUser: <IconUser />,
  IconBriefcase: <IconBriefcase />,
  IconInbox: <IconInbox />,

  // Inventory module
  IconArchive: <IconArchive />,
  IconList: <IconList />,
  IconCheckList: <IconCheckList />,
  IconAlertTriangle: <IconAlertTriangle />,

  // Trade module
  IconCart: <IconCart />,
  IconSend: <IconSend />,
  IconDownload: <IconDownload />,
  IconUndo: <IconUndo />,
  IconRedo: <IconRedo />,

  // Finance module
  IconCoinMoneyStroked: <IconCoinMoneyStroked />,
  IconMoneyExchangeStroked: <IconMoneyExchangeStroked />,
  IconCreditCard: <IconCreditCard />,
  IconMinusCircle: <IconMinusCircle />,
  IconPlusCircle: <IconPlusCircle />,
  IconHistory: <IconHistory />,

  // Reports module
  IconPieChartStroked: <IconPieChartStroked />,
  IconLineChartStroked: <IconLineChartStroked />,
  IconBarChartHStroked: <IconBarChartHStroked />,
  IconRefresh: <IconRefresh />,
  IconCandlestickChartStroked: <IconCandlestickChartStroked />,

  // System module
  IconSetting: <IconSetting />,
  IconMember: <IconMember />,
  IconKey: <IconKey />,
  IconLink: <IconLink />,
  IconSync: <IconSync />,
  IconConnectionPoint1: <IconConnectionPoint1 />,

  // Logo
  IconGridView: <IconGridView />,
}

/**
 * Map route titles to i18n keys
 */
const titleToI18nKey: Record<string, string> = {
  Dashboard: 'nav.dashboard',
  Catalog: 'nav.catalog',
  Products: 'nav.products',
  Categories: 'nav.categories',
  Partners: 'nav.partners',
  Customers: 'nav.customers',
  Suppliers: 'nav.suppliers',
  Warehouses: 'nav.warehouses',
  Inventory: 'nav.inventory',
  'Stock List': 'nav.stock',
  'Stock Taking': 'nav.stockTaking',
  'Stock Alerts': 'nav.stockAlerts',
  Trade: 'nav.trade',
  'Sales Orders': 'nav.salesOrders',
  'Purchase Orders': 'nav.purchaseOrders',
  'Sales Returns': 'nav.salesReturns',
  'Purchase Returns': 'nav.purchaseReturns',
  Finance: 'nav.finance',
  Receivables: 'nav.receivables',
  Payables: 'nav.payables',
  Expenses: 'nav.expenses',
  'Other Income': 'nav.otherIncome',
  'Cash Flow': 'nav.cashFlow',
  Reports: 'nav.reports',
  'Sales Report': 'nav.salesReport',
  'Sales Ranking': 'nav.salesRanking',
  'Inventory Turnover': 'nav.inventoryTurnover',
  'Profit & Loss': 'nav.profitLoss',
  System: 'nav.system',
  Users: 'nav.users',
  Roles: 'nav.roles',
  Permissions: 'nav.permissions',
  'Payment Settings': 'nav.paymentSettings',
  'Platform Config': 'nav.platformConfig',
  'Platform Sync Status': 'nav.platformSync',
  'Product Mappings': 'nav.productMappings',
  Settings: 'nav.settings',
}

/**
 * Convert AppRoute to Semi Nav items
 */
interface NavItem {
  itemKey: string
  text: string
  icon?: React.ReactNode
  items?: NavItem[]
}

/**
 * Check if user has permission to access a route
 * @param userPermissions - Array of user's permission codes
 * @param routePermissions - Array of required permissions (any match grants access)
 */
function hasRoutePermission(
  userPermissions: string[] | undefined,
  routePermissions: string[] | undefined
): boolean {
  // If no permissions required, grant access
  if (!routePermissions || routePermissions.length === 0) {
    return true
  }

  // If user has no permissions, deny access to permission-protected routes
  if (!userPermissions || userPermissions.length === 0) {
    return false
  }

  // Check if user has ANY of the required permissions
  return routePermissions.some((perm) => userPermissions.includes(perm))
}

/**
 * Convert route to nav item, filtering by user permissions
 */
function routeToNavItem(
  route: AppRoute,
  userPermissions: string[] | undefined,
  translate: (key: string) => string
): NavItem | null {
  // Skip routes that should be hidden from menu
  if (route.meta?.hideInMenu || !route.path || route.path === '*') {
    return null
  }

  // Check user permissions for this route
  if (!hasRoutePermission(userPermissions, route.meta?.permissions)) {
    return null
  }

  const icon = route.meta?.icon ? iconMap[route.meta.icon] : undefined
  const title = route.meta?.title || route.path
  // Get translated text using the title to i18n key mapping
  const i18nKey = titleToI18nKey[title]
  const text = i18nKey ? translate(i18nKey) : title

  // Handle routes with children
  if (route.children && route.children.length > 0) {
    const childItems = route.children
      .filter((child) => !child.redirect && !child.meta?.hideInMenu)
      .map((child) => routeToNavItem(child, userPermissions, translate))
      .filter((item): item is NavItem => item !== null)
      .sort((a, b) => {
        const aOrder = appRoutes.find((r) => r.path === a.itemKey)?.meta?.order ?? 999
        const bOrder = appRoutes.find((r) => r.path === b.itemKey)?.meta?.order ?? 999
        return aOrder - bOrder
      })

    // If no accessible children, hide the parent menu item
    if (childItems.length === 0) {
      return null
    }

    return {
      itemKey: route.path,
      text,
      icon,
      items: childItems,
    }
  }

  return {
    itemKey: route.path,
    text,
    icon,
  }
}

/**
 * Sidebar navigation component
 *
 * Features:
 * - Collapsible sidebar with toggle button
 * - Navigation menu from route configuration
 * - Active state highlighting
 * - Nested menu support for module grouping
 * - Permission-based menu filtering
 */
export function Sidebar() {
  const navigate = useNavigate()
  const location = useLocation()
  const { t } = useTranslation()
  const sidebarCollapsed = useAppStore((state) => state.sidebarCollapsed)
  const toggleSidebar = useAppStore((state) => state.toggleSidebar)
  const mobileSidebarOpen = useAppStore((state) => state.mobileSidebarOpen)
  const setMobileSidebarOpen = useAppStore((state) => state.setMobileSidebarOpen)
  const userPermissions = useAuthStore((state) => state.user?.permissions)

  // Generate navigation items from routes, filtered by user permissions
  const navItems = useMemo(() => {
    // Create a simple translate function that wraps the i18next t function
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const translate = (key: string) => (t as any)(key) as string
    return appRoutes
      .filter((route) => !route.meta?.hideInMenu && route.path !== '*')
      .map((route) => routeToNavItem(route, userPermissions, translate))
      .filter((item): item is NavItem => item !== null)
      .sort((a, b) => {
        const aRoute = appRoutes.find((r) => r.path === a.itemKey)
        const bRoute = appRoutes.find((r) => r.path === b.itemKey)
        return (aRoute?.meta?.order ?? 999) - (bRoute?.meta?.order ?? 999)
      })
  }, [userPermissions, t])

  // Determine selected keys based on current path
  const selectedKeys = useMemo(() => {
    const path = location.pathname
    const keys: string[] = []

    // Add exact match
    keys.push(path)

    // Add parent path for nested routes
    const segments = path.split('/').filter(Boolean)
    if (segments.length > 1) {
      keys.push(`/${segments[0]}`)
    }

    return keys
  }, [location.pathname])

  // Determine open keys for submenus
  const openKeys = useMemo(() => {
    const path = location.pathname
    const segments = path.split('/').filter(Boolean)
    if (segments.length > 0) {
      return [`/${segments[0]}`]
    }
    return []
  }, [location.pathname])

  // Handle navigation
  const handleSelect = useCallback(
    (data: { itemKey?: string | number; selectedKeys?: (string | number)[] }) => {
      const key = data.itemKey?.toString()
      if (key) {
        navigate(key)
        // Close mobile sidebar after navigation
        setMobileSidebarOpen(false)
      }
    },
    [navigate, setMobileSidebarOpen]
  )

  // Close mobile sidebar when clicking overlay
  const handleOverlayClick = useCallback(() => {
    setMobileSidebarOpen(false)
  }, [setMobileSidebarOpen])

  return (
    <>
      {/* Mobile overlay */}
      <div
        className={`sidebar-overlay ${mobileSidebarOpen ? 'sidebar-overlay--visible' : ''}`}
        onClick={handleOverlayClick}
        aria-hidden="true"
      />

      <Sider
        className={`sidebar ${sidebarCollapsed ? 'sidebar--collapsed' : ''} ${mobileSidebarOpen ? 'sidebar--mobile-open' : ''}`}
        style={{
          width: sidebarCollapsed ? 60 : 220,
        }}
      >
        {/* Logo area */}
        <div className="sidebar__logo">
          <div className="sidebar__logo-icon">
            <IconGridView size="large" />
          </div>
          {!sidebarCollapsed && <span className="sidebar__logo-text">ERP System</span>}
        </div>

        {/* Navigation menu */}
        <Nav
          className="sidebar__nav"
          items={navItems}
          selectedKeys={selectedKeys}
          defaultOpenKeys={openKeys}
          onSelect={handleSelect}
          isCollapsed={sidebarCollapsed}
          onCollapseChange={toggleSidebar}
          footer={{
            collapseButton: true,
          }}
        />
      </Sider>
    </>
  )
}
