import { useCallback, useMemo } from 'react'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { Layout, Nav, Breadcrumb } from '@douyinfe/semi-ui-19'
import {
  IconHome,
  IconUserGroup,
  IconPieChartStroked,
  IconHistory,
  IconGridView,
  IconChevronLeft,
} from '@douyinfe/semi-icons'
import { useTranslation } from 'react-i18next'

import { useAppStore } from '@/store'

import './AdminLayout.css'

const { Sider, Content, Header } = Layout

/**
 * Navigation items for admin sidebar
 */
interface AdminNavItem {
  itemKey: string
  text: string
  icon?: React.ReactNode
  items?: AdminNavItem[]
}

/**
 * Breadcrumb configuration for admin routes
 */
const adminBreadcrumbs: Record<string, { title: string; parent?: string }> = {
  '/super-admin': { title: 'Admin' },
  '/super-admin/tenants': { title: 'Tenants', parent: '/super-admin' },
  '/super-admin/stats': { title: 'Statistics', parent: '/super-admin' },
  '/super-admin/audit-logs': { title: 'Audit Logs', parent: '/super-admin' },
}

/**
 * Admin layout component for super admin routes
 *
 * Features:
 * - Dedicated sidebar for admin navigation
 * - Breadcrumb navigation
 * - Collapsible sidebar
 * - Back to main app link
 */
export function AdminLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const { t } = useTranslation()
  const sidebarCollapsed = useAppStore((state) => state.sidebarCollapsed)
  const toggleSidebar = useAppStore((state) => state.toggleSidebar)

  // Generate navigation items
  const navItems: AdminNavItem[] = useMemo(
    () => [
      {
        itemKey: '/super-admin/tenants',
        text: t('admin.nav.tenants', 'Tenants'),
        icon: <IconUserGroup />,
      },
      {
        itemKey: '/super-admin/stats',
        text: t('admin.nav.stats', 'Statistics'),
        icon: <IconPieChartStroked />,
      },
      {
        itemKey: '/super-admin/audit-logs',
        text: t('admin.nav.auditLogs', 'Audit Logs'),
        icon: <IconHistory />,
      },
    ],
    [t]
  )

  // Determine selected keys based on current path
  const selectedKeys = useMemo(() => {
    const path = location.pathname
    // Handle tenant detail routes
    if (path.startsWith('/super-admin/tenants/')) {
      return ['/super-admin/tenants']
    }
    return [path]
  }, [location.pathname])

  // Handle navigation
  const handleSelect = useCallback(
    (data: { itemKey?: string | number }) => {
      const key = data.itemKey?.toString()
      if (key) {
        navigate(key)
      }
    },
    [navigate]
  )

  // Generate breadcrumb items
  const breadcrumbItems = useMemo(() => {
    const items: { title: string; path?: string }[] = []
    let currentPath = location.pathname

    // Handle dynamic routes (e.g., /super-admin/tenants/:id)
    const pathParts = currentPath.split('/')
    if (pathParts.length > 3 && pathParts[2] === 'tenants') {
      // Add tenant detail breadcrumb
      items.unshift({ title: 'Tenant Details' })
      currentPath = '/super-admin/tenants'
    }

    // Build breadcrumb chain
    while (currentPath && adminBreadcrumbs[currentPath]) {
      const config = adminBreadcrumbs[currentPath]
      items.unshift({
        title: t(`admin.breadcrumb.${config.title.toLowerCase()}`, config.title),
        path: config.parent ? currentPath : undefined,
      })
      currentPath = config.parent || ''
    }

    // Add home link
    items.unshift({ title: t('nav.dashboard', 'Dashboard'), path: '/' })

    return items
  }, [location.pathname, t])

  return (
    <Layout className="admin-layout">
      <Sider
        className={`admin-layout__sider ${sidebarCollapsed ? 'admin-layout__sider--collapsed' : ''}`}
        style={{ width: sidebarCollapsed ? 60 : 220 }}
      >
        {/* Logo area */}
        <div className="admin-layout__logo">
          <div className="admin-layout__logo-icon">
            <IconGridView size="large" />
          </div>
          {!sidebarCollapsed && (
            <span className="admin-layout__logo-text">{t('admin.title', 'Admin Panel')}</span>
          )}
        </div>

        {/* Back to main app */}
        <div
          className="admin-layout__back"
          onClick={() => navigate('/')}
          role="button"
          tabIndex={0}
          onKeyDown={(e) => e.key === 'Enter' && navigate('/')}
        >
          <IconChevronLeft />
          {!sidebarCollapsed && (
            <span className="admin-layout__back-text">{t('admin.backToApp', 'Back to App')}</span>
          )}
        </div>

        {/* Navigation menu */}
        <Nav
          className="admin-layout__nav"
          items={navItems}
          selectedKeys={selectedKeys}
          onSelect={handleSelect}
          isCollapsed={sidebarCollapsed}
          onCollapseChange={toggleSidebar}
          footer={{
            collapseButton: true,
          }}
        />
      </Sider>

      <Layout className="admin-layout__main">
        <Header className="admin-layout__header">
          <Breadcrumb>
            {breadcrumbItems.map((item, index) => (
              <Breadcrumb.Item
                key={index}
                href={item.path}
                onClick={(e) => {
                  if (item.path) {
                    e.preventDefault()
                    navigate(item.path)
                  }
                }}
              >
                {index === 0 && <IconHome style={{ marginRight: 4 }} />}
                {item.title}
              </Breadcrumb.Item>
            ))}
          </Breadcrumb>
        </Header>

        <Content className="admin-layout__content">
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  )
}
