import { Outlet } from 'react-router-dom'
import { Layout } from '@douyinfe/semi-ui'

import { useAppStore } from '@/store'
import { Header } from './Header'
import { Sidebar } from './Sidebar'

import './MainLayout.css'

const { Content } = Layout

/**
 * Main application layout with Header, Sidebar, and Content areas
 *
 * Features:
 * - Responsive sidebar that can be collapsed
 * - Fixed header with user menu and settings
 * - Scrollable content area
 * - Theme-aware styling
 */
export function MainLayout() {
  const sidebarCollapsed = useAppStore((state) => state.sidebarCollapsed)

  return (
    <Layout className="main-layout">
      <Sidebar />
      <Layout className="main-layout__right">
        <Header />
        <Content
          className={`main-layout__content ${sidebarCollapsed ? 'main-layout__content--collapsed' : ''}`}
        >
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  )
}
