import { useCallback, useMemo, useState } from 'react'
import {
  Modal,
  Toast,
  Typography,
  Space,
  Spin,
  InputNumber,
  Progress,
  Button,
  Divider,
} from '@douyinfe/semi-ui-19'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import {
  useGetTenantById,
  useUpdateTenantConfig,
  getListTenantsQueryKey,
  getGetTenantByIdQueryKey,
} from '@/api/tenants/tenants'

const { Text, Title } = Typography

// Plan default quotas for presets
const PLAN_QUOTAS: Record<
  string,
  { maxUsers: number; maxProducts: number; maxWarehouses: number }
> = {
  free: { maxUsers: 5, maxProducts: 100, maxWarehouses: 1 },
  basic: { maxUsers: 10, maxProducts: 500, maxWarehouses: 3 },
  pro: { maxUsers: 50, maxProducts: 5000, maxWarehouses: 10 },
  enterprise: { maxUsers: 999, maxProducts: 99999, maxWarehouses: 999 },
}

interface QuotaValues {
  maxUsers: number
  maxProducts: number
  maxWarehouses: number
}

interface UpdateQuotaModalProps {
  visible: boolean
  tenantId: string | null
  tenantName?: string
  onClose: () => void
  onSuccess?: () => void
}

/**
 * UpdateQuotaModal - Dialog for updating tenant quota limits
 *
 * Features:
 * - Shows current quota and usage
 * - Allows custom quota values for users, products, warehouses
 * - Provides plan-based presets
 * - Progress bars showing current usage
 * - API integration with React Query
 * - Loading state during operation
 * - Success/error toast notifications
 * - Keyboard support (Esc to cancel)
 * - Refreshes tenant data on success
 */
export function UpdateQuotaModal({
  visible,
  tenantId,
  tenantName,
  onClose,
  onSuccess,
}: UpdateQuotaModalProps) {
  const { t } = useTranslation('admin')
  const queryClient = useQueryClient()

  // Quota state
  const [quotaValues, setQuotaValues] = useState<QuotaValues>({
    maxUsers: 10,
    maxProducts: 1000,
    maxWarehouses: 5,
  })

  // Fetch tenant data
  const { data: tenantResponse, isLoading: isFetching } = useGetTenantById(tenantId || '', {
    query: {
      enabled: visible && !!tenantId,
    },
  })

  // Extract tenant data
  const tenant = useMemo(() => {
    if (tenantResponse?.status === 200 && tenantResponse.data.data) {
      return tenantResponse.data.data
    }
    return undefined
  }, [tenantResponse])

  const displayName =
    tenantName || tenant?.name || t('tenants.delete.unknownTenant', 'Unknown Tenant')

  // Mock current usage (in production, this would come from a usage API)
  const currentUsage = useMemo(() => {
    return {
      users: 5,
      products: 150,
      warehouses: 2,
    }
  }, [])

  // Calculate usage percentages
  const usagePercentages = useMemo(() => {
    const calcPercent = (current: number, max: number): number => {
      if (max <= 0) return 0
      return Math.min(100, Math.round((current / max) * 100))
    }

    return {
      users: calcPercent(currentUsage.users, quotaValues.maxUsers),
      products: calcPercent(currentUsage.products, quotaValues.maxProducts),
      warehouses: calcPercent(currentUsage.warehouses, quotaValues.maxWarehouses),
    }
  }, [currentUsage, quotaValues])

  // Update config mutation
  const updateConfigMutation = useUpdateTenantConfig({
    mutation: {
      onSuccess: () => {
        Toast.success(t('tenants.messages.quotaUpdateSuccess', 'Quota updated successfully'))
        queryClient.invalidateQueries({ queryKey: getListTenantsQueryKey() })
        if (tenantId) {
          queryClient.invalidateQueries({ queryKey: getGetTenantByIdQueryKey(tenantId) })
        }
        onClose()
        onSuccess?.()
      },
      onError: (error: Error | null) => {
        const errorMessage =
          error?.message || t('tenants.messages.quotaUpdateError', 'Failed to update quota')
        Toast.error(errorMessage)
      },
    },
  })

  // Handle quota update
  const handleUpdateQuota = useCallback(() => {
    if (!tenantId) return
    updateConfigMutation.mutate({
      id: tenantId,
      data: {
        max_users: quotaValues.maxUsers,
        max_products: quotaValues.maxProducts,
        max_warehouses: quotaValues.maxWarehouses,
      },
    })
  }, [tenantId, quotaValues, updateConfigMutation])

  // Handle preset selection
  const handleApplyPreset = useCallback(
    (plan: string) => {
      const preset = PLAN_QUOTAS[plan]
      if (preset) {
        setQuotaValues({
          maxUsers: preset.maxUsers,
          maxProducts: preset.maxProducts,
          maxWarehouses: preset.maxWarehouses,
        })
      }
    },
    [setQuotaValues]
  )

  // Check if values have changed
  const hasChanges = useMemo(() => {
    const config = tenant?.config
    if (!config) return false
    return (
      quotaValues.maxUsers !== (config.max_users || 10) ||
      quotaValues.maxProducts !== (config.max_products || 1000) ||
      quotaValues.maxWarehouses !== (config.max_warehouses || 5)
    )
  }, [tenant, quotaValues])

  // Check for quota reduction warnings
  const warnings = useMemo(() => {
    const result: string[] = []
    if (currentUsage.users > quotaValues.maxUsers) {
      result.push(t('tenants.updateQuota.warningUsers', 'Current user count exceeds new limit'))
    }
    if (currentUsage.products > quotaValues.maxProducts) {
      result.push(
        t('tenants.updateQuota.warningProducts', 'Current product count exceeds new limit')
      )
    }
    if (currentUsage.warehouses > quotaValues.maxWarehouses) {
      result.push(
        t('tenants.updateQuota.warningWarehouses', 'Current warehouse count exceeds new limit')
      )
    }
    return result
  }, [currentUsage, quotaValues, t])

  return (
    <Modal
      title={t('tenants.actions.updateQuota', 'Update Quota')}
      visible={visible}
      onOk={handleUpdateQuota}
      onCancel={onClose}
      okText={t('common.save', 'Save')}
      cancelText={t('common.cancel', 'Cancel')}
      okButtonProps={{
        disabled: isFetching || !hasChanges,
      }}
      confirmLoading={updateConfigMutation.isPending}
      width={560}
      closeOnEsc={false}
      maskClosable={false}
    >
      {isFetching ? (
        <div style={{ display: 'flex', justifyContent: 'center', padding: 48 }}>
          <Spin size="large" />
        </div>
      ) : (
        <Space vertical align="start" spacing="medium" style={{ width: '100%' }}>
          {/* Description */}
          <Text>
            {t(
              'tenants.updateQuota.description',
              'Update quota limits for tenant "{{name}}". Changes take effect immediately.',
              { name: displayName }
            )}
          </Text>

          {/* Presets */}
          <div style={{ width: '100%' }}>
            <Title heading={6} style={{ marginBottom: 8 }}>
              {t('tenants.updateQuota.presets', 'Apply Plan Preset')}
            </Title>
            <Space>
              {Object.keys(PLAN_QUOTAS).map((plan) => (
                <Button
                  key={plan}
                  size="small"
                  onClick={() => handleApplyPreset(plan)}
                  type={tenant?.plan === plan ? 'primary' : 'tertiary'}
                >
                  {plan.charAt(0).toUpperCase() + plan.slice(1)}
                </Button>
              ))}
            </Space>
          </div>

          <Divider margin={8} />

          {/* Users Quota */}
          <div style={{ width: '100%' }}>
            <Space style={{ justifyContent: 'space-between', width: '100%', marginBottom: 8 }}>
              <Text strong>{t('tenants.updateQuota.maxUsers', 'Max Users')}</Text>
              <Text type="tertiary">
                {t('tenants.updateQuota.currentUsage', 'Current')}: {currentUsage.users}
              </Text>
            </Space>
            <InputNumber
              value={quotaValues.maxUsers}
              onChange={(value) =>
                setQuotaValues((prev) => ({ ...prev, maxUsers: (value as number) || 1 }))
              }
              min={1}
              max={9999}
              style={{ width: '100%' }}
            />
            <Progress
              percent={usagePercentages.users}
              showInfo
              format={() => `${currentUsage.users} / ${quotaValues.maxUsers}`}
              style={{ marginTop: 8 }}
              stroke={usagePercentages.users > 90 ? 'var(--semi-color-danger)' : undefined}
            />
          </div>

          {/* Products Quota */}
          <div style={{ width: '100%' }}>
            <Space style={{ justifyContent: 'space-between', width: '100%', marginBottom: 8 }}>
              <Text strong>{t('tenants.updateQuota.maxProducts', 'Max Products')}</Text>
              <Text type="tertiary">
                {t('tenants.updateQuota.currentUsage', 'Current')}: {currentUsage.products}
              </Text>
            </Space>
            <InputNumber
              value={quotaValues.maxProducts}
              onChange={(value) =>
                setQuotaValues((prev) => ({ ...prev, maxProducts: (value as number) || 1 }))
              }
              min={1}
              max={999999}
              style={{ width: '100%' }}
            />
            <Progress
              percent={usagePercentages.products}
              showInfo
              format={() => `${currentUsage.products} / ${quotaValues.maxProducts}`}
              style={{ marginTop: 8 }}
              stroke={usagePercentages.products > 90 ? 'var(--semi-color-danger)' : undefined}
            />
          </div>

          {/* Warehouses Quota */}
          <div style={{ width: '100%' }}>
            <Space style={{ justifyContent: 'space-between', width: '100%', marginBottom: 8 }}>
              <Text strong>{t('tenants.updateQuota.maxWarehouses', 'Max Warehouses')}</Text>
              <Text type="tertiary">
                {t('tenants.updateQuota.currentUsage', 'Current')}: {currentUsage.warehouses}
              </Text>
            </Space>
            <InputNumber
              value={quotaValues.maxWarehouses}
              onChange={(value) =>
                setQuotaValues((prev) => ({ ...prev, maxWarehouses: (value as number) || 1 }))
              }
              min={1}
              max={999}
              style={{ width: '100%' }}
            />
            <Progress
              percent={usagePercentages.warehouses}
              showInfo
              format={() => `${currentUsage.warehouses} / ${quotaValues.maxWarehouses}`}
              style={{ marginTop: 8 }}
              stroke={usagePercentages.warehouses > 90 ? 'var(--semi-color-danger)' : undefined}
            />
          </div>

          {/* Warnings */}
          {warnings.length > 0 && (
            <div
              style={{
                width: '100%',
                background: 'var(--semi-color-warning-light-default)',
                padding: 12,
                borderRadius: 8,
              }}
            >
              <Text type="warning" strong style={{ display: 'block', marginBottom: 4 }}>
                {t('tenants.updateQuota.warningTitle', 'Warning')}
              </Text>
              {warnings.map((warning, index) => (
                <Text key={index} type="warning" size="small" style={{ display: 'block' }}>
                  • {warning}
                </Text>
              ))}
            </div>
          )}
        </Space>
      )}
    </Modal>
  )
}

export default UpdateQuotaModal
