import { useCallback } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import {
  Layout,
  Nav,
  Avatar,
  Dropdown,
  Typography,
  Button,
  Space,
  Breadcrumb,
} from '@douyinfe/semi-ui-19'
import {
  IconMoon,
  IconSun,
  IconUser,
  IconExit,
  IconSetting,
  IconBell,
  IconHome,
  IconMenu,
  IconCreditCard,
  IconHistogram,
} from '@douyinfe/semi-icons'

import { useAppStore, useAuthStore, useUser } from '@/store'
import { getBreadcrumbs } from '@/router/routes'
import { useI18n } from '@/hooks'
import { LanguageSwitcher } from '@/components/common/LanguageSwitcher'
import { TenantSwitcher } from '@/components/common/TenantSwitcher'

import './Header.css'

const { Header: SemiHeader } = Layout
const { Text } = Typography

/**
 * Application header component
 *
 * Features:
 * - Breadcrumb navigation
 * - Theme toggle (light/dark)
 * - Notification bell
 * - User avatar with dropdown menu
 * - Logout functionality
 */
export function Header() {
  const navigate = useNavigate()
  const location = useLocation()
  const user = useUser()
  const logout = useAuthStore((state) => state.logout)
  const theme = useAppStore((state) => state.theme)
  const toggleTheme = useAppStore((state) => state.toggleTheme)
  const sidebarCollapsed = useAppStore((state) => state.sidebarCollapsed)
  const toggleMobileSidebar = useAppStore((state) => state.toggleMobileSidebar)
  const { t } = useI18n()

  // Generate breadcrumbs from current path
  const breadcrumbItems = getBreadcrumbs(location.pathname)

  // Handle logout
  const handleLogout = useCallback(() => {
    logout()
    navigate('/login')
  }, [logout, navigate])

  // Handle profile navigation
  const handleProfile = useCallback(() => {
    navigate('/profile')
  }, [navigate])

  // Handle settings navigation
  const handleSettings = useCallback(() => {
    navigate('/settings')
  }, [navigate])

  // Handle subscription navigation
  const handleSubscription = useCallback(() => {
    navigate('/subscription')
  }, [navigate])

  // Handle billing navigation
  const handleBilling = useCallback(() => {
    navigate('/billing')
  }, [navigate])

  // User dropdown menu items
  const userMenuItems = [
    {
      node: 'item',
      key: 'profile',
      name: t('nav.profile', 'Profile'),
      icon: <IconUser />,
      onClick: handleProfile,
    },
    {
      node: 'item',
      key: 'settings',
      name: t('nav.settings'),
      icon: <IconSetting />,
      onClick: handleSettings,
    },
    {
      node: 'item',
      key: 'subscription',
      name: t('nav.subscription', 'Subscription'),
      icon: <IconCreditCard />,
      onClick: handleSubscription,
    },
    {
      node: 'item',
      key: 'billing',
      name: t('nav.billing', 'Billing History'),
      icon: <IconHistogram />,
      onClick: handleBilling,
    },
    {
      node: 'divider',
      key: 'divider',
    },
    {
      node: 'item',
      key: 'logout',
      name: t('actions.logout', 'Logout'),
      icon: <IconExit />,
      onClick: handleLogout,
      type: 'danger' as const,
    },
  ]

  return (
    <SemiHeader className={`header ${sidebarCollapsed ? 'header--collapsed' : ''}`}>
      {/* Left side: Mobile menu button + Breadcrumb */}
      <div className="header__left">
        {/* Mobile menu toggle button */}
        <Button
          className="header__mobile-menu-btn"
          theme="borderless"
          icon={<IconMenu />}
          onClick={toggleMobileSidebar}
          aria-label={t('actions.toggleMenu', 'Toggle menu')}
        />

        {/* Breadcrumb navigation */}
        <div className="header__breadcrumb">
          <Breadcrumb>
            <Breadcrumb.Item
              href="/"
              icon={<IconHome size="small" />}
              onClick={(e) => {
                e.preventDefault()
                navigate('/')
              }}
            />
            {breadcrumbItems.map((item, index) => (
              <Breadcrumb.Item
                key={item.path}
                href={item.path}
                onClick={(e) => {
                  e.preventDefault()
                  if (index < breadcrumbItems.length - 1) {
                    navigate(item.path)
                  }
                }}
              >
                {item.title}
              </Breadcrumb.Item>
            ))}
          </Breadcrumb>
        </div>
      </div>

      {/* Right side actions */}
      <div className="header__right">
        <Nav mode="horizontal" className="header__nav">
          <Space spacing={8}>
            {/* Language switcher */}
            <LanguageSwitcher />

            {/* Tenant switcher */}
            <TenantSwitcher />

            {/* Theme toggle */}
            <Button
              theme="borderless"
              icon={theme === 'light' ? <IconMoon /> : <IconSun />}
              onClick={toggleTheme}
              aria-label={
                theme === 'light'
                  ? t('actions.switchToDark', 'Switch to dark mode')
                  : t('actions.switchToLight', 'Switch to light mode')
              }
            />

            {/* Notifications */}
            <Button
              theme="borderless"
              icon={<IconBell />}
              aria-label={t('nav.notifications', 'Notifications')}
            />

            {/* User menu */}
            <Dropdown
              trigger="click"
              position="bottomRight"
              getPopupContainer={() => document.body}
              render={
                <Dropdown.Menu>
                  <div className="header__user-info">
                    <Text strong>{user?.displayName || user?.username || 'User'}</Text>
                    {user?.email && (
                      <Text type="tertiary" size="small">
                        {user.email}
                      </Text>
                    )}
                  </div>
                  <Dropdown.Divider />
                  {userMenuItems.map((item) => {
                    if (item.node === 'divider') {
                      return <Dropdown.Divider key={item.key} />
                    }
                    return (
                      <Dropdown.Item
                        key={item.key}
                        icon={item.icon}
                        onClick={item.onClick}
                        type={item.type}
                      >
                        {item.name}
                      </Dropdown.Item>
                    )
                  })}
                </Dropdown.Menu>
              }
            >
              <div className="header__avatar">
                <Avatar
                  size="small"
                  src={user?.avatar}
                  color="light-blue"
                  alt={user?.displayName || user?.username}
                >
                  {(user?.displayName || user?.username || 'U').charAt(0).toUpperCase()}
                </Avatar>
              </div>
            </Dropdown>
          </Space>
        </Nav>
      </div>
    </SemiHeader>
  )
}
