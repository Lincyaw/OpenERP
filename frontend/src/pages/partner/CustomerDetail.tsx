import { useState, useEffect, useCallback, useMemo } from 'react'
import { Card, Tag, Toast, Modal, Empty, Typography } from '@douyinfe/semi-ui-19'
import { IconEdit, IconPlay, IconStop, IconDelete, IconCreditCard } from '@douyinfe/semi-icons'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Container } from '@/components/common/layout'
import {
  DetailPageHeader,
  type DetailPageHeaderAction,
  type DetailPageHeaderStatus,
  type DetailPageHeaderMetric,
} from '@/components/common'
import { useFormatters } from '@/hooks/useFormatters'
import {
  getCustomerById,
  activateCustomer,
  deactivateCustomer,
  deleteCustomer,
} from '@/api/customers/customers'
import type {
  HandlerCustomerResponse,
  HandlerCustomerResponseStatus,
  HandlerCustomerResponseLevel,
} from '@/api/models'
import './CustomerDetail.css'

const { Text } = Typography

// Status variant mapping for DetailPageHeader
const STATUS_VARIANTS: Record<HandlerCustomerResponseStatus, 'default' | 'success' | 'warning'> = {
  active: 'success',
  inactive: 'default',
  suspended: 'warning',
}

// Level tag color mapping
const LEVEL_TAG_COLORS: Record<
  HandlerCustomerResponseLevel,
  'white' | 'grey' | 'amber' | 'cyan' | 'violet'
> = {
  normal: 'white',
  silver: 'grey',
  gold: 'amber',
  platinum: 'cyan',
  vip: 'violet',
}

/**
 * Customer Detail Page
 *
 * Features:
 * - Display complete customer information using DetailPageHeader
 * - Display balance information
 * - Status action buttons (activate, deactivate, suspend)
 * - Navigate to edit page
 * - Navigate to balance page
 */
export default function CustomerDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { t } = useTranslation(['partner', 'common'])
  const { formatCurrency, formatDateTime } = useFormatters()

  const [customer, setCustomer] = useState<HandlerCustomerResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState(false)

  // Fetch customer details
  const fetchCustomer = useCallback(async () => {
    if (!id) return

    setLoading(true)
    try {
      const response = await getCustomerById(id)
      if (response.status === 200 && response.data.success && response.data.data) {
        setCustomer(response.data.data)
      } else {
        Toast.error(t('customers.messages.fetchCustomerError'))
        navigate('/partner/customers')
      }
    } catch {
      Toast.error(t('customers.messages.fetchCustomerError'))
      navigate('/partner/customers')
    } finally {
      setLoading(false)
    }
  }, [id, navigate, t])

  useEffect(() => {
    fetchCustomer()
  }, [fetchCustomer])

  // Handle activate customer
  const handleActivate = useCallback(async () => {
    if (!customer?.id) return
    setActionLoading(true)
    try {
      await activateCustomer(customer.id, {})
      Toast.success(t('customers.messages.activateSuccess', { name: customer.name }))
      fetchCustomer()
    } catch {
      Toast.error(t('customers.messages.activateError'))
    } finally {
      setActionLoading(false)
    }
  }, [customer, fetchCustomer, t])

  // Handle deactivate customer
  const handleDeactivate = useCallback(async () => {
    if (!customer?.id) return
    setActionLoading(true)
    try {
      await deactivateCustomer(customer.id, {})
      Toast.success(t('customers.messages.deactivateSuccess', { name: customer.name }))
      fetchCustomer()
    } catch {
      Toast.error(t('customers.messages.deactivateError'))
    } finally {
      setActionLoading(false)
    }
  }, [customer, fetchCustomer, t])

  // Handle suspend customer
  const handleSuspend = useCallback(async () => {
    if (!customer?.id) return
    Modal.confirm({
      title: t('customers.confirm.suspendTitle'),
      content: t('customers.confirm.suspendContent', { name: customer.name }),
      okText: t('customers.confirm.suspendOk'),
      cancelText: t('common:actions.cancel'),
      okButtonProps: { type: 'warning' },
      onOk: async () => {
        setActionLoading(true)
        try {
          await deactivateCustomer(customer.id!, {})
          Toast.success(t('customers.messages.suspendSuccess', { name: customer.name }))
          fetchCustomer()
        } catch {
          Toast.error(t('customers.messages.suspendError'))
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [customer, fetchCustomer, t])

  // Handle delete customer
  const handleDelete = useCallback(async () => {
    if (!customer?.id) return
    Modal.confirm({
      title: t('customers.confirm.deleteTitle'),
      content: t('customers.confirm.deleteContent', { name: customer.name }),
      okText: t('customers.confirm.deleteOk'),
      cancelText: t('common:actions.cancel'),
      okButtonProps: { type: 'danger' },
      onOk: async () => {
        setActionLoading(true)
        try {
          await deleteCustomer(customer.id!)
          Toast.success(t('customers.messages.deleteSuccess', { name: customer.name }))
          navigate('/partner/customers')
        } catch {
          Toast.error(t('customers.messages.deleteError'))
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [customer, navigate, t])

  // Handle edit customer
  const handleEdit = useCallback(() => {
    if (customer?.id) {
      navigate(`/partner/customers/${customer.id}/edit`)
    }
  }, [customer, navigate])

  // Handle view balance
  const handleViewBalance = useCallback(() => {
    if (customer?.id) {
      navigate(`/partner/customers/${customer.id}/balance`)
    }
  }, [customer, navigate])

  // Build status for DetailPageHeader
  const headerStatus = useMemo((): DetailPageHeaderStatus | undefined => {
    if (!customer?.status) return undefined
    return {
      label: t(`customers.status.${customer.status}`),
      variant: STATUS_VARIANTS[customer.status] || 'default',
    }
  }, [customer, t])

  // Build metrics for DetailPageHeader
  const headerMetrics = useMemo((): DetailPageHeaderMetric[] => {
    if (!customer) return []
    return [
      {
        label: t('customerDetail.fields.type'),
        value: customer.type ? t(`customers.type.${customer.type}`) : '-',
      },
      {
        label: t('customerDetail.fields.level'),
        value: customer.level ? t(`customers.level.${customer.level}`) : '-',
      },
      {
        label: t('customerDetail.fields.balance'),
        value: customer.balance !== undefined ? formatCurrency(customer.balance) : '-',
        variant: customer.balance && customer.balance > 0 ? 'success' : 'default',
      },
      {
        label: t('customerDetail.fields.creditLimit'),
        value: customer.credit_limit !== undefined ? formatCurrency(customer.credit_limit) : '-',
        variant: 'primary',
      },
    ]
  }, [customer, t, formatCurrency])

  // Build primary action for DetailPageHeader
  const primaryAction = useMemo((): DetailPageHeaderAction | undefined => {
    if (!customer) return undefined
    const status = customer.status

    if (status !== 'active') {
      return {
        key: 'activate',
        label: t('customers.actions.activate'),
        icon: <IconPlay />,
        type: 'primary',
        onClick: handleActivate,
        loading: actionLoading,
      }
    }
    return {
      key: 'balance',
      label: t('customers.actions.balance'),
      icon: <IconCreditCard />,
      type: 'primary',
      onClick: handleViewBalance,
    }
  }, [customer, t, handleActivate, handleViewBalance, actionLoading])

  // Build secondary actions for DetailPageHeader
  const secondaryActions = useMemo((): DetailPageHeaderAction[] => {
    if (!customer) return []
    const status = customer.status
    const actions: DetailPageHeaderAction[] = []

    actions.push({
      key: 'edit',
      label: t('common:actions.edit'),
      icon: <IconEdit />,
      onClick: handleEdit,
      disabled: actionLoading,
    })

    if (status === 'active') {
      actions.push({
        key: 'deactivate',
        label: t('customers.actions.deactivate'),
        icon: <IconStop />,
        type: 'warning',
        onClick: handleDeactivate,
        loading: actionLoading,
      })
      actions.push({
        key: 'suspend',
        label: t('customers.actions.suspend'),
        type: 'warning',
        onClick: handleSuspend,
        loading: actionLoading,
      })
    }

    actions.push({
      key: 'delete',
      label: t('customers.actions.delete'),
      icon: <IconDelete />,
      type: 'danger',
      onClick: handleDelete,
      loading: actionLoading,
    })

    return actions
  }, [customer, t, handleEdit, handleDeactivate, handleSuspend, handleDelete, actionLoading])

  // Render basic info
  const renderBasicInfo = () => {
    if (!customer) return null

    return (
      <div className="info-grid">
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('customerDetail.fields.code')}
          </Text>
          <Text strong className="info-value code-value">
            {customer.code || '-'}
          </Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('customerDetail.fields.name')}
          </Text>
          <Text className="info-value">{customer.name || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('customerDetail.fields.shortName')}
          </Text>
          <Text className="info-value">{customer.short_name || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('customerDetail.fields.type')}
          </Text>
          <div className="info-value">
            {customer.type ? (
              <Tag color={customer.type === 'organization' ? 'blue' : 'light-blue'}>
                {t(`customers.type.${customer.type}`)}
              </Tag>
            ) : (
              '-'
            )}
          </div>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('customerDetail.fields.level')}
          </Text>
          <div className="info-value">
            {customer.level ? (
              <Tag color={LEVEL_TAG_COLORS[customer.level]}>
                {t(`customers.level.${customer.level}`)}
              </Tag>
            ) : (
              '-'
            )}
          </div>
        </div>
      </div>
    )
  }

  // Render contact info
  const renderContactInfo = () => {
    if (!customer) return null

    return (
      <div className="info-grid">
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('customerDetail.fields.contactName')}
          </Text>
          <Text className="info-value">{customer.contact_name || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('customerDetail.fields.phone')}
          </Text>
          <Text className="info-value">{customer.phone || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('customerDetail.fields.email')}
          </Text>
          <Text className="info-value">{customer.email || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('customerDetail.fields.taxId')}
          </Text>
          <Text className="info-value code-value">{customer.tax_id || '-'}</Text>
        </div>
      </div>
    )
  }

  // Render address info
  const renderAddressInfo = () => {
    if (!customer) return null

    return (
      <div className="info-grid">
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('customerDetail.fields.country')}
          </Text>
          <Text className="info-value">{customer.country || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('customerDetail.fields.province')}
          </Text>
          <Text className="info-value">{customer.province || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('customerDetail.fields.city')}
          </Text>
          <Text className="info-value">{customer.city || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('customerDetail.fields.address')}
          </Text>
          <Text className="info-value">{customer.address || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('customerDetail.fields.postalCode')}
          </Text>
          <Text className="info-value code-value">{customer.postal_code || '-'}</Text>
        </div>
        <div className="info-item info-item--full">
          <Text type="secondary" className="info-label">
            {t('customerDetail.fields.fullAddress')}
          </Text>
          <Text className="info-value">{customer.full_address || '-'}</Text>
        </div>
      </div>
    )
  }

  // Render finance info
  const renderFinanceInfo = () => {
    if (!customer) return null

    const balanceClass =
      customer.balance !== undefined ? (customer.balance > 0 ? 'positive' : 'negative') : ''

    return (
      <div className="info-grid">
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('customerDetail.fields.balance')}
          </Text>
          <Text className={`info-value balance-value ${balanceClass}`}>
            {customer.balance !== undefined ? formatCurrency(customer.balance) : '-'}
          </Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('customerDetail.fields.creditLimit')}
          </Text>
          <Text className="info-value credit-value">
            {customer.credit_limit !== undefined ? formatCurrency(customer.credit_limit) : '-'}
          </Text>
        </div>
      </div>
    )
  }

  // Render timestamps
  const renderTimestamps = () => {
    if (!customer) return null

    return (
      <div className="info-grid">
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('customerDetail.fields.createdAt')}
          </Text>
          <Text className="info-value">
            {customer.created_at ? formatDateTime(customer.created_at) : '-'}
          </Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('customerDetail.fields.updatedAt')}
          </Text>
          <Text className="info-value">
            {customer.updated_at ? formatDateTime(customer.updated_at) : '-'}
          </Text>
        </div>
      </div>
    )
  }

  if (loading) {
    return (
      <Container size="lg" className="customer-detail-page">
        <DetailPageHeader
          title={t('customerDetail.title')}
          loading={true}
          showBack={true}
          onBack={() => navigate('/partner/customers')}
          backLabel={t('customerDetail.back')}
        />
      </Container>
    )
  }

  if (!customer) {
    return (
      <Container size="lg" className="customer-detail-page">
        <Empty
          title={t('customerDetail.notExist')}
          description={t('customerDetail.notExistDesc')}
        />
      </Container>
    )
  }

  return (
    <Container size="lg" className="customer-detail-page">
      {/* Unified Header */}
      <DetailPageHeader
        title={t('customerDetail.title')}
        documentNumber={customer.code}
        status={headerStatus}
        metrics={headerMetrics}
        primaryAction={primaryAction}
        secondaryActions={secondaryActions}
        onBack={() => navigate('/partner/customers')}
        backLabel={t('customerDetail.back')}
        titleSuffix={
          customer.short_name ? (
            <span className="customer-short-name-suffix">{customer.short_name}</span>
          ) : undefined
        }
      />

      {/* Basic Info Card */}
      <Card className="info-card" title={t('customers.form.basicInfo')}>
        {renderBasicInfo()}
      </Card>

      {/* Contact Info Card */}
      <Card className="info-card" title={t('customers.form.contactInfo')}>
        {renderContactInfo()}
      </Card>

      {/* Address Info Card */}
      <Card className="info-card" title={t('customers.form.addressInfo')}>
        {renderAddressInfo()}
      </Card>

      {/* Finance Info Card */}
      <Card className="info-card" title={t('customerDetail.financeInfo')}>
        {renderFinanceInfo()}
      </Card>

      {/* Timestamps Card */}
      <Card className="info-card" title={t('customerDetail.timestamps')}>
        {renderTimestamps()}
      </Card>

      {/* Notes Card */}
      {customer.notes && (
        <Card className="info-card" title={t('customerDetail.notes')}>
          <Text>{customer.notes}</Text>
        </Card>
      )}
    </Container>
  )
}
