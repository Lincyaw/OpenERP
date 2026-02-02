import { useCallback, useMemo, useState } from 'react'
import { Modal, Toast, Typography, Space, Spin, Tag, RadioGroup, Radio } from '@douyinfe/semi-ui-19'
import type { TagColor } from '@douyinfe/semi-ui-19/lib/es/tag'
import { IconArrowUp, IconArrowDown, IconMinus } from '@douyinfe/semi-icons'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import {
  useGetTenantById,
  useSetPlanTenant,
  getListTenantsQueryKey,
  getGetTenantByIdQueryKey,
} from '@/api/tenants/tenants'
import { HandlerSetTenantPlanRequestPlan } from '@/api/models'

const { Text, Title } = Typography

// Plan hierarchy for upgrade/downgrade detection
const PLAN_HIERARCHY: Record<string, number> = {
  free: 0,
  basic: 1,
  pro: 2,
  enterprise: 3,
}

// Plan details for display
const PLAN_DETAILS: Record<
  string,
  { label: string; color: TagColor; maxUsers: number; maxProducts: number; maxWarehouses: number }
> = {
  free: { label: 'Free', color: 'grey', maxUsers: 5, maxProducts: 100, maxWarehouses: 1 },
  basic: { label: 'Basic', color: 'blue', maxUsers: 10, maxProducts: 500, maxWarehouses: 3 },
  pro: { label: 'Pro', color: 'purple', maxUsers: 50, maxProducts: 5000, maxWarehouses: 10 },
  enterprise: {
    label: 'Enterprise',
    color: 'amber',
    maxUsers: 999,
    maxProducts: 99999,
    maxWarehouses: 999,
  },
}

interface ChangePlanModalProps {
  visible: boolean
  tenantId: string | null
  tenantName?: string
  currentPlan?: string
  onClose: () => void
  onSuccess?: () => void
}

/**
 * ChangePlanModal - Dialog for changing tenant subscription plan
 *
 * Features:
 * - Shows current plan and available plans
 * - Displays upgrade/downgrade indicator
 * - Shows plan comparison (quotas)
 * - API integration with React Query
 * - Loading state during operation
 * - Success/error toast notifications
 * - Keyboard support (Esc to cancel)
 * - Refreshes tenant data on success
 */
export function ChangePlanModal({
  visible,
  tenantId,
  tenantName,
  currentPlan: propCurrentPlan,
  onClose,
  onSuccess,
}: ChangePlanModalProps) {
  const { t } = useTranslation('admin')
  const queryClient = useQueryClient()

  // Fetch tenant data if not provided
  const { data: tenantResponse, isLoading: isFetching } = useGetTenantById(tenantId || '', {
    query: {
      enabled: visible && !!tenantId && !propCurrentPlan,
    },
  })

  // Extract tenant data
  const tenant = useMemo(() => {
    if (tenantResponse?.status === 200 && tenantResponse.data.data) {
      return tenantResponse.data.data
    }
    return undefined
  }, [tenantResponse])

  const currentPlan = propCurrentPlan || tenant?.plan || 'free'
  const displayName = tenantName || tenant?.name || t('tenants.delete.unknownTenant', 'Unknown')

  // Initialize selected plan - use currentPlan as initial value
  const [selectedPlan, setSelectedPlan] = useState<string>(currentPlan)

  // Set plan mutation
  const setPlanMutation = useSetPlanTenant({
    mutation: {
      onSuccess: () => {
        Toast.success(t('tenants.messages.planChangeSuccess', 'Plan changed successfully'))
        queryClient.invalidateQueries({ queryKey: getListTenantsQueryKey() })
        if (tenantId) {
          queryClient.invalidateQueries({ queryKey: getGetTenantByIdQueryKey(tenantId) })
        }
        onClose()
        onSuccess?.()
      },
      onError: (error: Error | null) => {
        const errorMessage =
          error?.message || t('tenants.messages.planChangeError', 'Failed to change plan')
        Toast.error(errorMessage)
      },
    },
  })

  // Handle plan change
  const handleChangePlan = useCallback(() => {
    if (!tenantId || !selectedPlan || selectedPlan === currentPlan) return
    setPlanMutation.mutate({
      id: tenantId,
      data: { plan: selectedPlan as HandlerSetTenantPlanRequestPlan },
    })
  }, [tenantId, selectedPlan, currentPlan, setPlanMutation])

  // Determine change type (upgrade/downgrade/same)
  const changeType = useMemo(() => {
    if (!selectedPlan || selectedPlan === currentPlan) return 'same'
    const currentLevel = PLAN_HIERARCHY[currentPlan] ?? 0
    const selectedLevel = PLAN_HIERARCHY[selectedPlan] ?? 0
    return selectedLevel > currentLevel ? 'upgrade' : 'downgrade'
  }, [selectedPlan, currentPlan])

  // Get change indicator icon and text
  const changeIndicator = useMemo(() => {
    switch (changeType) {
      case 'upgrade':
        return {
          icon: <IconArrowUp style={{ color: 'var(--semi-color-success)' }} />,
          text: t('tenants.changePlan.upgrade', 'Upgrade'),
          color: 'green' as TagColor,
        }
      case 'downgrade':
        return {
          icon: <IconArrowDown style={{ color: 'var(--semi-color-warning)' }} />,
          text: t('tenants.changePlan.downgrade', 'Downgrade'),
          color: 'orange' as TagColor,
        }
      default:
        return {
          icon: <IconMinus style={{ color: 'var(--semi-color-text-2)' }} />,
          text: t('tenants.changePlan.noChange', 'No Change'),
          color: 'grey' as TagColor,
        }
    }
  }, [changeType, t])

  // Plan options for radio group
  const planOptions = useMemo(() => {
    return Object.entries(PLAN_DETAILS).map(([key, details]) => ({
      value: key,
      label: details.label,
      color: details.color,
      isCurrent: key === currentPlan,
    }))
  }, [currentPlan])

  // Selected plan details
  const selectedPlanDetails = PLAN_DETAILS[selectedPlan] || PLAN_DETAILS.free
  const currentPlanDetails = PLAN_DETAILS[currentPlan] || PLAN_DETAILS.free

  return (
    <Modal
      title={t('tenants.modal.changePlanTitle', 'Change Plan')}
      visible={visible}
      onOk={handleChangePlan}
      onCancel={onClose}
      okText={t('common.save', 'Save')}
      cancelText={t('common.cancel', 'Cancel')}
      okButtonProps={{
        disabled: isFetching || selectedPlan === currentPlan,
      }}
      confirmLoading={setPlanMutation.isPending}
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
              'tenants.changePlan.description',
              'Select a new plan for tenant "{{name}}". Plan changes take effect immediately.',
              { name: displayName }
            )}
          </Text>

          {/* Current Plan */}
          <div style={{ width: '100%' }}>
            <Title heading={6} style={{ marginBottom: 8 }}>
              {t('tenants.changePlan.currentPlan', 'Current Plan')}
            </Title>
            <Tag color={currentPlanDetails.color} size="large">
              {currentPlanDetails.label}
            </Tag>
          </div>

          {/* Plan Selection */}
          <div style={{ width: '100%' }}>
            <Title heading={6} style={{ marginBottom: 8 }}>
              {t('tenants.changePlan.selectPlan', 'Select New Plan')}
            </Title>
            <RadioGroup
              value={selectedPlan}
              onChange={(e) => setSelectedPlan(e.target.value as string)}
              direction="vertical"
              style={{ width: '100%' }}
            >
              {planOptions.map((option) => (
                <Radio
                  key={option.value}
                  value={option.value}
                  style={{
                    padding: '12px 16px',
                    border: '1px solid var(--semi-color-border)',
                    borderRadius: 8,
                    marginBottom: 8,
                    width: '100%',
                    background:
                      selectedPlan === option.value
                        ? 'var(--semi-color-primary-light-default)'
                        : 'transparent',
                  }}
                >
                  <Space>
                    <Tag color={option.color} size="small">
                      {option.label}
                    </Tag>
                    {option.isCurrent && (
                      <Text type="tertiary" size="small">
                        ({t('tenants.changePlan.current', 'Current')})
                      </Text>
                    )}
                  </Space>
                </Radio>
              ))}
            </RadioGroup>
          </div>

          {/* Change Impact */}
          {selectedPlan !== currentPlan && (
            <div
              style={{
                width: '100%',
                background:
                  changeType === 'upgrade'
                    ? 'var(--semi-color-success-light-default)'
                    : 'var(--semi-color-warning-light-default)',
                padding: 16,
                borderRadius: 8,
              }}
            >
              <Space style={{ marginBottom: 12 }}>
                {changeIndicator.icon}
                <Tag color={changeIndicator.color}>{changeIndicator.text}</Tag>
              </Space>

              <Title heading={6} style={{ marginBottom: 8 }}>
                {t('tenants.changePlan.quotaChanges', 'Quota Changes')}
              </Title>

              <Space vertical align="start" spacing="tight" style={{ width: '100%' }}>
                <Text>
                  {t('tenants.changePlan.maxUsers', 'Max Users')}: {currentPlanDetails.maxUsers} →{' '}
                  {selectedPlanDetails.maxUsers}
                </Text>
                <Text>
                  {t('tenants.changePlan.maxProducts', 'Max Products')}:{' '}
                  {currentPlanDetails.maxProducts} → {selectedPlanDetails.maxProducts}
                </Text>
                <Text>
                  {t('tenants.changePlan.maxWarehouses', 'Max Warehouses')}:{' '}
                  {currentPlanDetails.maxWarehouses} → {selectedPlanDetails.maxWarehouses}
                </Text>
              </Space>

              {changeType === 'downgrade' && (
                <div
                  style={{
                    marginTop: 12,
                    padding: 8,
                    background: 'var(--semi-color-warning-light-hover)',
                    borderRadius: 4,
                  }}
                >
                  <Text type="warning" size="small">
                    {t(
                      'tenants.changePlan.downgradeWarning',
                      'Downgrading may affect current usage if it exceeds the new plan limits.'
                    )}
                  </Text>
                </div>
              )}
            </div>
          )}
        </Space>
      )}
    </Modal>
  )
}

export default ChangePlanModal
