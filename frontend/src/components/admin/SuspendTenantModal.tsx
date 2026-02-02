import { useCallback, useMemo, useState } from 'react'
import {
  Modal,
  Toast,
  Typography,
  Space,
  Spin,
  TextArea,
  DatePicker,
  Tag,
  Checkbox,
} from '@douyinfe/semi-ui-19'
import type { TagColor } from '@douyinfe/semi-ui-19/lib/es/tag'
import { IconAlertTriangle } from '@douyinfe/semi-icons'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import {
  useGetTenantById,
  useSuspendTenant,
  getListTenantsQueryKey,
  getGetTenantByIdQueryKey,
} from '@/api/tenants/tenants'

const { Text, Title } = Typography

/**
 * Get color for tenant status badge
 */
function getStatusColor(status: string | undefined): TagColor {
  switch (status) {
    case 'active':
      return 'green'
    case 'suspended':
      return 'red'
    case 'inactive':
      return 'grey'
    case 'trial':
      return 'orange'
    default:
      return 'grey'
  }
}

interface SuspendTenantModalProps {
  visible: boolean
  tenantId: string | null
  tenantName?: string
  onClose: () => void
  onSuccess?: () => void
}

/**
 * SuspendTenantModal - Dialog for suspending a tenant account
 *
 * Features:
 * - Requires suspension reason (mandatory)
 * - Optional suspension duration with auto-reactivation date
 * - Shows suspension impact explanation
 * - Displays tenant info
 * - API integration with React Query
 * - Loading state during operation
 * - Success/error toast notifications
 * - Keyboard support (Esc to cancel)
 * - Refreshes tenant data on success
 */
export function SuspendTenantModal({
  visible,
  tenantId,
  tenantName,
  onClose,
  onSuccess,
}: SuspendTenantModalProps) {
  const { t } = useTranslation('admin')
  const queryClient = useQueryClient()

  // Form state
  const [reason, setReason] = useState('')
  const [hasExpiration, setHasExpiration] = useState(false)
  const [expirationDate, setExpirationDate] = useState<Date | null>(null)

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

  // Suspend mutation
  const suspendMutation = useSuspendTenant({
    mutation: {
      onSuccess: () => {
        Toast.success(t('tenants.messages.suspendSuccess', 'Tenant suspended successfully'))
        queryClient.invalidateQueries({ queryKey: getListTenantsQueryKey() })
        if (tenantId) {
          queryClient.invalidateQueries({ queryKey: getGetTenantByIdQueryKey(tenantId) })
        }
        onClose()
        onSuccess?.()
      },
      onError: (error: Error | null) => {
        const errorMessage =
          error?.message || t('tenants.messages.suspendError', 'Failed to suspend tenant')
        Toast.error(errorMessage)
      },
    },
  })

  // Handle suspend
  const handleSuspend = useCallback(() => {
    if (!tenantId || !reason.trim()) return

    const data: Record<string, unknown> = {
      reason: reason.trim(),
    }

    if (hasExpiration && expirationDate) {
      data.suspend_until = expirationDate.toISOString()
    }

    suspendMutation.mutate({
      id: tenantId,
      data,
    })
  }, [tenantId, reason, hasExpiration, expirationDate, suspendMutation])

  // Validation
  const isValid = reason.trim().length >= 10
  const reasonError =
    reason.length > 0 && reason.trim().length < 10
      ? t('tenants.suspend.reasonMinLength', 'Reason must be at least 10 characters')
      : undefined

  // Impact items
  const impactItems = useMemo(
    () => [
      t('tenants.suspend.impactLogin', 'Users will not be able to log in'),
      t('tenants.suspend.impactApi', 'API access will be blocked'),
      t('tenants.suspend.impactData', 'Data will be preserved but inaccessible'),
      t('tenants.suspend.impactBilling', 'Billing may continue depending on plan'),
    ],
    [t]
  )

  // Minimum date for expiration (tomorrow)
  const minExpirationDate = useMemo(() => {
    const tomorrow = new Date()
    tomorrow.setDate(tomorrow.getDate() + 1)
    tomorrow.setHours(0, 0, 0, 0)
    return tomorrow
  }, [])

  return (
    <Modal
      title={
        <Space>
          <IconAlertTriangle style={{ color: 'var(--semi-color-danger)' }} />
          {t('tenants.confirm.suspendTitle', 'Confirm Suspend')}
        </Space>
      }
      visible={visible}
      onOk={handleSuspend}
      onCancel={onClose}
      okText={t('tenants.confirm.suspendOk', 'Suspend')}
      cancelText={t('common.cancel', 'Cancel')}
      okButtonProps={{
        type: 'danger',
        disabled: isFetching || !isValid,
      }}
      confirmLoading={suspendMutation.isPending}
      width={520}
      closeOnEsc={false}
      maskClosable={false}
    >
      {isFetching ? (
        <div style={{ display: 'flex', justifyContent: 'center', padding: 48 }}>
          <Spin size="large" />
        </div>
      ) : (
        <Space vertical align="start" spacing="medium" style={{ width: '100%' }}>
          {/* Warning banner */}
          <div
            style={{
              background: 'var(--semi-color-danger-light-default)',
              padding: 16,
              borderRadius: 8,
              width: '100%',
            }}
          >
            <Text>
              {t(
                'tenants.suspend.warningMessage',
                'You are about to suspend tenant "{{name}}". This will immediately block all access.',
                { name: displayName }
              )}
            </Text>
          </div>

          {/* Tenant info */}
          {tenant && (
            <div style={{ width: '100%' }}>
              <Title heading={6} style={{ marginBottom: 8 }}>
                {t('tenants.suspend.tenantInfo', 'Tenant Information')}
              </Title>
              <Space>
                <Text strong>{tenant.name}</Text>
                <Tag color={getStatusColor(tenant.status)} size="small">
                  {tenant.status}
                </Tag>
              </Space>
              {tenant.contact_email && (
                <div style={{ marginTop: 4 }}>
                  <Text type="secondary">{tenant.contact_email}</Text>
                </div>
              )}
            </div>
          )}

          {/* Reason input (required) */}
          <div style={{ width: '100%' }}>
            <Title heading={6} style={{ marginBottom: 8 }}>
              {t('tenants.suspend.reason', 'Suspension Reason')} *
            </Title>
            <TextArea
              value={reason}
              onChange={(value) => setReason(value)}
              placeholder={t(
                'tenants.suspend.reasonPlaceholder',
                'Enter the reason for suspension (required, min 10 characters)'
              )}
              rows={3}
              maxCount={500}
              showClear
              validateStatus={reasonError ? 'error' : undefined}
            />
            {reasonError && (
              <Text type="danger" size="small" style={{ marginTop: 4 }}>
                {reasonError}
              </Text>
            )}
          </div>

          {/* Optional expiration */}
          <div style={{ width: '100%' }}>
            <Checkbox
              checked={hasExpiration}
              onChange={(e) => {
                setHasExpiration(e.target.checked ?? false)
                if (!e.target.checked) {
                  setExpirationDate(null)
                }
              }}
            >
              {t('tenants.suspend.setExpiration', 'Set automatic reactivation date')}
            </Checkbox>

            {hasExpiration && (
              <div style={{ marginTop: 12 }}>
                <DatePicker
                  type="dateTime"
                  value={expirationDate || undefined}
                  onChange={(date) => setExpirationDate(date as Date | null)}
                  placeholder={t(
                    'tenants.suspend.expirationPlaceholder',
                    'Select reactivation date'
                  )}
                  disabledDate={(date) => !!(date && date < minExpirationDate)}
                  style={{ width: '100%' }}
                />
                <Text type="tertiary" size="small" style={{ marginTop: 4, display: 'block' }}>
                  {t(
                    'tenants.suspend.expirationHelp',
                    'Tenant will be automatically reactivated at this date and time'
                  )}
                </Text>
              </div>
            )}
          </div>

          {/* Impact explanation */}
          <div
            style={{
              width: '100%',
              background: 'var(--semi-color-warning-light-default)',
              padding: 16,
              borderRadius: 8,
            }}
          >
            <Title heading={6} style={{ marginBottom: 8 }}>
              {t('tenants.suspend.impactTitle', 'Suspension Impact')}
            </Title>
            <Space vertical align="start" spacing="tight">
              {impactItems.map((item, index) => (
                <Text key={index} type="warning" size="small">
                  • {item}
                </Text>
              ))}
            </Space>
          </div>
        </Space>
      )}
    </Modal>
  )
}

export default SuspendTenantModal
