import { useCallback, useMemo } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { Layout, Nav } from '@douyinfe/semi-ui'
import {
  IconHome,
  IconGridView,
  IconUserGroup,
  IconList,
  IconSend,
  IconDownload,
  IconPriceTag,
  IconCreditCard,
  IconInbox,
  IconTreeTriangleDown,
  IconUserCardVideo,
  IconMinus,
  IconPlus,
  IconHistory,
} from '@douyinfe/semi-icons'

import { useAppStore, useAuthStore } from '@/store'
import { appRoutes } from '@/router/routes'
import type { AppRoute } from '@/router/types'

import './Sidebar.css'

const { Sider } = Layout

/**
 * Icon mapping from string names to Semi Design icons
 */
const iconMap: Record<string, React.ReactNode> = {
  IconHome: <IconHome />,
  IconGridView: <IconGridView />,
  IconUserGroup: <IconUserGroup />,
  IconList: <IconList />,
  IconSend: <IconSend />,
  IconDownload: <IconDownload />,
  IconPriceTag: <IconPriceTag />,
  IconCreditCard: <IconCreditCard />,
  IconInbox: <IconInbox />,
  IconTreeTriangleDown: <IconTreeTriangleDown />,
  IconUserCardVideo: <IconUserCardVideo />,
  IconMinus: <IconMinus />,
  IconPlus: <IconPlus />,
  IconHistory: <IconHistory />,
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
function routeToNavItem(route: AppRoute, userPermissions: string[] | undefined): NavItem | null {
  // Skip routes that should be hidden from menu
  if (route.meta?.hideInMenu || !route.path || route.path === '*') {
    return null
  }

  // Check user permissions for this route
  if (!hasRoutePermission(userPermissions, route.meta?.permissions)) {
    return null
  }

  const icon = route.meta?.icon ? iconMap[route.meta.icon] : undefined
  const text = route.meta?.title || route.path

  // Handle routes with children
  if (route.children && route.children.length > 0) {
    const childItems = route.children
      .filter((child) => !child.redirect && !child.meta?.hideInMenu)
      .map((child) => routeToNavItem(child, userPermissions))
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
  const sidebarCollapsed = useAppStore((state) => state.sidebarCollapsed)
  const toggleSidebar = useAppStore((state) => state.toggleSidebar)
  const userPermissions = useAuthStore((state) => state.user?.permissions)

  // Generate navigation items from routes, filtered by user permissions
  const navItems = useMemo(() => {
    return appRoutes
      .filter((route) => !route.meta?.hideInMenu && route.path !== '*')
      .map((route) => routeToNavItem(route, userPermissions))
      .filter((item): item is NavItem => item !== null)
      .sort((a, b) => {
        const aRoute = appRoutes.find((r) => r.path === a.itemKey)
        const bRoute = appRoutes.find((r) => r.path === b.itemKey)
        return (aRoute?.meta?.order ?? 999) - (bRoute?.meta?.order ?? 999)
      })
  }, [userPermissions])

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
      }
    },
    [navigate]
  )

  return (
    <Sider
      className={`sidebar ${sidebarCollapsed ? 'sidebar--collapsed' : ''}`}
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
  )
}
