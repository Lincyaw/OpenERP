/**
 * TenantSwitcher Component
 *
 * A dropdown component for switching between tenants.
 * Fetches available tenants from API and allows users to switch.
 * Remembers last selection via auth store persistence.
 */

import { useState, useEffect, useCallback } from 'react'
import { Dropdown, Button, Spin, Toast } from '@douyinfe/semi-ui-19'
import { IconGridSquare } from '@douyinfe/semi-icons'
import { useI18n } from '@/hooks'
import { useAuthStore, useUser } from '@/store'
import { getTenants } from '@/api/tenants/tenants'
import type { HandlerTenantResponse } from '@/api/models'
import { createScopedLogger } from '@/utils'

const log = createScopedLogger('TenantSwitcher')

interface TenantSwitcherProps {
  /** Show label next to icon */
  showLabel?: boolean
  /** Button size */
  size?: 'small' | 'default' | 'large'
  /** Additional class name */
  className?: string
}

interface TenantOption {
  id: string
  name: string
  code: string
  shortName?: string
  logoUrl?: string
  isCurrent: boolean
}

const TENANT_STORAGE_KEY = 'erp-current-tenant'

/**
 * Tenant switcher dropdown component
 *
 * @example
 * ```tsx
 * // Icon only (default)
 * <TenantSwitcher />
 *
 * // With label
 * <TenantSwitcher showLabel />
 *
 * // Custom size
 * <TenantSwitcher size="small" />
 * ```
 */
export function TenantSwitcher({
  showLabel = false,
  size = 'default',
  className,
}: TenantSwitcherProps) {
  const { t } = useI18n({ ns: 'system' })
  const user = useUser()
  const updateUser = useAuthStore((state) => state.updateUser)

  const [tenants, setTenants] = useState<TenantOption[]>([])
  const [loading, setLoading] = useState(false)
  const [dropdownVisible, setDropdownVisible] = useState(false)

  // Current tenant ID from user or localStorage fallback
  const currentTenantId = user?.tenantId || localStorage.getItem(TENANT_STORAGE_KEY)

  // Fetch tenants when dropdown opens
  const fetchTenants = useCallback(async () => {
    if (tenants.length > 0) return // Already loaded

    setLoading(true)
    try {
      const api = getTenants()
      const response = await api.listTenants({ page_size: 100 })

      if (response.success && response.data?.tenants) {
        const tenantOptions: TenantOption[] = response.data.tenants
          .filter((tenant: HandlerTenantResponse) => tenant.status === 'active')
          .map((tenant: HandlerTenantResponse) => ({
            id: tenant.id || '',
            name: tenant.name || '',
            code: tenant.code || '',
            shortName: tenant.short_name,
            logoUrl: tenant.logo_url,
            isCurrent: tenant.id === currentTenantId,
          }))

        setTenants(tenantOptions)
      }
    } catch (error) {
      log.error('Failed to fetch tenants', error)
      Toast.error(t('tenantSwitcher.messages.fetchError', 'Failed to load tenants'))
    } finally {
      setLoading(false)
    }
  }, [currentTenantId, t, tenants.length])

  // Load tenants when dropdown opens
  useEffect(() => {
    if (dropdownVisible && tenants.length === 0) {
      fetchTenants()
    }
  }, [dropdownVisible, fetchTenants, tenants.length])

  // Handle tenant switch
  const handleTenantSwitch = useCallback(
    (tenantId: string) => {
      if (tenantId === currentTenantId) {
        setDropdownVisible(false)
        return
      }

      // Update user's tenantId in auth store
      updateUser({ tenantId })

      // Also store in localStorage for persistence
      localStorage.setItem(TENANT_STORAGE_KEY, tenantId)

      // Update local state to reflect new current tenant
      setTenants((prev) =>
        prev.map((tenant) => ({
          ...tenant,
          isCurrent: tenant.id === tenantId,
        }))
      )

      setDropdownVisible(false)

      // Show success message
      const selectedTenant = tenants.find((tenant) => tenant.id === tenantId)
      Toast.success(
        t('tenantSwitcher.messages.switchSuccess', {
          defaultValue: 'Switched to {{name}}',
          name: selectedTenant?.name || tenantId,
        })
      )

      // Reload the page to refresh all data with new tenant context
      // This ensures all API calls use the new tenant ID
      window.location.reload()
    },
    [currentTenantId, tenants, t, updateUser]
  )

  // Get current tenant info for display
  const currentTenant = tenants.find((tenant) => tenant.isCurrent)

  // Render dropdown menu content
  const renderMenu = () => {
    if (loading) {
      return (
        <Dropdown.Menu>
          <Dropdown.Item disabled>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <Spin size="small" />
              <span>{t('tenantSwitcher.loading', 'Loading...')}</span>
            </div>
          </Dropdown.Item>
        </Dropdown.Menu>
      )
    }

    if (tenants.length === 0) {
      return (
        <Dropdown.Menu>
          <Dropdown.Item disabled>
            {t('tenantSwitcher.noTenants', 'No tenants available')}
          </Dropdown.Item>
        </Dropdown.Menu>
      )
    }

    return (
      <Dropdown.Menu>
        {tenants.map((tenant) => (
          <Dropdown.Item
            key={tenant.id}
            active={tenant.isCurrent}
            onClick={() => handleTenantSwitch(tenant.id)}
          >
            {tenant.shortName || tenant.name}
          </Dropdown.Item>
        ))}
      </Dropdown.Menu>
    )
  }

  return (
    <Dropdown
      trigger="click"
      position="bottomRight"
      render={renderMenu()}
      visible={dropdownVisible}
      onVisibleChange={setDropdownVisible}
      getPopupContainer={() => document.body}
    >
      <span style={{ display: 'inline-flex' }}>
        <Button
          theme="borderless"
          icon={<IconGridSquare />}
          size={size}
          className={className}
          aria-label={t('tenantSwitcher.ariaLabel', 'Switch tenant')}
        >
          {showLabel && currentTenant && (
            <span style={{ marginLeft: '4px' }}>
              {currentTenant.shortName || currentTenant.name}
            </span>
          )}
        </Button>
      </span>
    </Dropdown>
  )
}

export default TenantSwitcher
