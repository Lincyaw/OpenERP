import { useState, useEffect, useCallback, useMemo } from 'react'
import {
  Card,
  Typography,
  Descriptions,
  Tag,
  Toast,
  Button,
  Space,
  Spin,
  Empty,
  Modal,
} from '@douyinfe/semi-ui-19'
import {
  IconArrowLeft,
  IconEdit,
  IconPlay,
  IconStop,
  IconDelete,
  IconCreditCard,
} from '@douyinfe/semi-icons'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Container } from '@/components/common/layout'
import { useFormatters } from '@/hooks/useFormatters'
import { getCustomers } from '@/api/customers/customers'
import type {
  HandlerCustomerResponse,
  HandlerCustomerResponseStatus,
  HandlerCustomerResponseLevel,
} from '@/api/models'
import './CustomerDetail.css'

const { Title, Text } = Typography

// Status tag color mapping
const STATUS_TAG_COLORS: Record<HandlerCustomerResponseStatus, 'green' | 'grey' | 'orange'> = {
  active: 'green',
  inactive: 'grey',
  suspended: 'orange',
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
 * - Display complete customer information
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
  const customersApi = useMemo(() => getCustomers(), [])

  const [customer, setCustomer] = useState<HandlerCustomerResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState(false)

  // Fetch customer details
  const fetchCustomer = useCallback(async () => {
    if (!id) return

    setLoading(true)
    try {
      const response = await customersApi.getCustomerById(id)
      if (response.success && response.data) {
        setCustomer(response.data)
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
  }, [id, customersApi, navigate, t])

  useEffect(() => {
    fetchCustomer()
  }, [fetchCustomer])

  // Handle activate customer
  const handleActivate = useCallback(async () => {
    if (!customer?.id) return
    setActionLoading(true)
    try {
      await customersApi.activateCustomer(customer.id)
      Toast.success(t('customers.messages.activateSuccess', { name: customer.name }))
      fetchCustomer()
    } catch {
      Toast.error(t('customers.messages.activateError'))
    } finally {
      setActionLoading(false)
    }
  }, [customer, customersApi, fetchCustomer, t])

  // Handle deactivate customer
  const handleDeactivate = useCallback(async () => {
    if (!customer?.id) return
    setActionLoading(true)
    try {
      await customersApi.deactivateCustomer(customer.id)
      Toast.success(t('customers.messages.deactivateSuccess', { name: customer.name }))
      fetchCustomer()
    } catch {
      Toast.error(t('customers.messages.deactivateError'))
    } finally {
      setActionLoading(false)
    }
  }, [customer, customersApi, fetchCustomer, t])

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
          await customersApi.deactivateCustomer(customer.id!)
          Toast.success(t('customers.messages.suspendSuccess', { name: customer.name }))
          fetchCustomer()
        } catch {
          Toast.error(t('customers.messages.suspendError'))
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [customer, customersApi, fetchCustomer, t])

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
          await customersApi.deleteCustomer(customer.id!)
          Toast.success(t('customers.messages.deleteSuccess', { name: customer.name }))
          navigate('/partner/customers')
        } catch {
          Toast.error(t('customers.messages.deleteError'))
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [customer, customersApi, navigate, t])

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

  // Render basic info
  const renderBasicInfo = () => {
    if (!customer) return null

    const data = [
      { key: t('customerDetail.fields.code'), value: customer.code || '-' },
      { key: t('customerDetail.fields.name'), value: customer.name || '-' },
      { key: t('customerDetail.fields.shortName'), value: customer.short_name || '-' },
      {
        key: t('customerDetail.fields.type'),
        value: customer.type ? (
          <Tag color={customer.type === 'organization' ? 'blue' : 'light-blue'}>
            {t(`customers.type.${customer.type}`)}
          </Tag>
        ) : (
          '-'
        ),
      },
      {
        key: t('customerDetail.fields.level'),
        value: customer.level ? (
          <Tag color={LEVEL_TAG_COLORS[customer.level]}>
            {t(`customers.level.${customer.level}`)}
          </Tag>
        ) : (
          '-'
        ),
      },
      {
        key: t('customerDetail.fields.status'),
        value: customer.status ? (
          <Tag color={STATUS_TAG_COLORS[customer.status]}>
            {t(`customers.status.${customer.status}`)}
          </Tag>
        ) : (
          '-'
        ),
      },
    ]

    return <Descriptions data={data} row className="customer-basic-info" />
  }

  // Render contact info
  const renderContactInfo = () => {
    if (!customer) return null

    const data = [
      { key: t('customerDetail.fields.contactName'), value: customer.contact_name || '-' },
      { key: t('customerDetail.fields.phone'), value: customer.phone || '-' },
      { key: t('customerDetail.fields.email'), value: customer.email || '-' },
      { key: t('customerDetail.fields.taxId'), value: customer.tax_id || '-' },
    ]

    return <Descriptions data={data} row className="customer-contact-info" />
  }

  // Render address info
  const renderAddressInfo = () => {
    if (!customer) return null

    const data = [
      { key: t('customerDetail.fields.country'), value: customer.country || '-' },
      { key: t('customerDetail.fields.province'), value: customer.province || '-' },
      { key: t('customerDetail.fields.city'), value: customer.city || '-' },
      { key: t('customerDetail.fields.address'), value: customer.address || '-' },
      { key: t('customerDetail.fields.postalCode'), value: customer.postal_code || '-' },
      { key: t('customerDetail.fields.fullAddress'), value: customer.full_address || '-' },
    ]

    return <Descriptions data={data} row className="customer-address-info" />
  }

  // Render finance info
  const renderFinanceInfo = () => {
    if (!customer) return null

    const data = [
      {
        key: t('customerDetail.fields.balance'),
        value: customer.balance !== undefined ? formatCurrency(customer.balance) : '-',
      },
      {
        key: t('customerDetail.fields.creditLimit'),
        value: customer.credit_limit !== undefined ? formatCurrency(customer.credit_limit) : '-',
      },
    ]

    return <Descriptions data={data} row className="customer-finance-info" />
  }

  // Render timestamps
  const renderTimestamps = () => {
    if (!customer) return null

    const data = [
      {
        key: t('customerDetail.fields.createdAt'),
        value: customer.created_at ? formatDateTime(customer.created_at) : '-',
      },
      {
        key: t('customerDetail.fields.updatedAt'),
        value: customer.updated_at ? formatDateTime(customer.updated_at) : '-',
      },
    ]

    return <Descriptions data={data} row className="customer-timestamps" />
  }

  // Render action buttons based on status
  const renderActions = () => {
    if (!customer) return null

    const status = customer.status

    return (
      <Space>
        <Button icon={<IconEdit />} onClick={handleEdit} disabled={actionLoading}>
          {t('common:actions.edit')}
        </Button>
        <Button
          type="primary"
          icon={<IconCreditCard />}
          onClick={handleViewBalance}
          disabled={actionLoading}
        >
          {t('customers.actions.balance')}
        </Button>
        {status !== 'active' && (
          <Button
            type="primary"
            icon={<IconPlay />}
            onClick={handleActivate}
            loading={actionLoading}
          >
            {t('customers.actions.activate')}
          </Button>
        )}
        {status === 'active' && (
          <>
            <Button
              type="warning"
              icon={<IconStop />}
              onClick={handleDeactivate}
              loading={actionLoading}
            >
              {t('customers.actions.deactivate')}
            </Button>
            <Button type="warning" onClick={handleSuspend} loading={actionLoading}>
              {t('customers.actions.suspend')}
            </Button>
          </>
        )}
        <Button type="danger" icon={<IconDelete />} onClick={handleDelete} loading={actionLoading}>
          {t('customers.actions.delete')}
        </Button>
      </Space>
    )
  }

  if (loading) {
    return (
      <Container size="lg" className="customer-detail-page">
        <div className="loading-container">
          <Spin size="large" />
        </div>
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
      {/* Header */}
      <div className="page-header">
        <div className="header-left">
          <Button
            icon={<IconArrowLeft />}
            theme="borderless"
            onClick={() => navigate('/partner/customers')}
          >
            {t('customerDetail.back')}
          </Button>
          <Title heading={4} className="page-title">
            {t('customerDetail.title')}
          </Title>
          {customer.status && (
            <Tag color={STATUS_TAG_COLORS[customer.status]} size="large">
              {t(`customers.status.${customer.status}`)}
            </Tag>
          )}
        </div>
        <div className="header-right">{renderActions()}</div>
      </div>

      {/* Customer Name Display */}
      <Card className="customer-name-card">
        <div className="customer-name-display">
          <Text className="customer-code">{customer.code}</Text>
          <Title heading={3} className="customer-name">
            {customer.name}
          </Title>
          {customer.short_name && <Text type="secondary">{customer.short_name}</Text>}
        </div>
      </Card>

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
