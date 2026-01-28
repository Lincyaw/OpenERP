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
  Rating,
} from '@douyinfe/semi-ui-19'
import { IconArrowLeft, IconEdit, IconPlay, IconStop, IconDelete } from '@douyinfe/semi-icons'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Container } from '@/components/common/layout'
import { useFormatters } from '@/hooks/useFormatters'
import { getSuppliers } from '@/api/suppliers/suppliers'
import type { HandlerSupplierResponse, HandlerSupplierResponseStatus } from '@/api/models'
import './SupplierDetail.css'

const { Title, Text } = Typography

// Status tag color mapping
const STATUS_TAG_COLORS: Record<HandlerSupplierResponseStatus, 'green' | 'grey' | 'red'> = {
  active: 'green',
  inactive: 'grey',
  blocked: 'red',
}

/**
 * Supplier Detail Page
 *
 * Features:
 * - Display complete supplier information
 * - Display bank and payment info
 * - Status action buttons (activate, deactivate, block)
 * - Navigate to edit page
 */
export default function SupplierDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { t } = useTranslation(['partner', 'common'])
  const { formatCurrency, formatDateTime } = useFormatters()
  const suppliersApi = useMemo(() => getSuppliers(), [])

  const [supplier, setSupplier] = useState<HandlerSupplierResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState(false)

  // Fetch supplier details
  const fetchSupplier = useCallback(async () => {
    if (!id) return

    setLoading(true)
    try {
      const response = await suppliersApi.getSupplierById(id)
      if (response.success && response.data) {
        setSupplier(response.data)
      } else {
        Toast.error(t('supplierDetail.messages.fetchError'))
        navigate('/partner/suppliers')
      }
    } catch {
      Toast.error(t('supplierDetail.messages.fetchError'))
      navigate('/partner/suppliers')
    } finally {
      setLoading(false)
    }
  }, [id, suppliersApi, navigate, t])

  useEffect(() => {
    fetchSupplier()
  }, [fetchSupplier])

  // Handle activate supplier
  const handleActivate = useCallback(async () => {
    if (!supplier?.id) return
    setActionLoading(true)
    try {
      await suppliersApi.activateSupplier(supplier.id)
      Toast.success(t('suppliers.messages.activateSuccess', { name: supplier.name }))
      fetchSupplier()
    } catch {
      Toast.error(t('suppliers.messages.activateError'))
    } finally {
      setActionLoading(false)
    }
  }, [supplier, suppliersApi, fetchSupplier, t])

  // Handle deactivate supplier
  const handleDeactivate = useCallback(async () => {
    if (!supplier?.id) return
    setActionLoading(true)
    try {
      await suppliersApi.deactivateSupplier(supplier.id)
      Toast.success(t('suppliers.messages.deactivateSuccess', { name: supplier.name }))
      fetchSupplier()
    } catch {
      Toast.error(t('suppliers.messages.deactivateError'))
    } finally {
      setActionLoading(false)
    }
  }, [supplier, suppliersApi, fetchSupplier, t])

  // Handle block supplier
  const handleBlock = useCallback(async () => {
    if (!supplier?.id) return
    Modal.confirm({
      title: t('suppliers.confirm.blockTitle'),
      content: t('suppliers.confirm.blockContent', { name: supplier.name }),
      okText: t('suppliers.confirm.blockOk'),
      cancelText: t('common:actions.cancel'),
      okButtonProps: { type: 'danger' },
      onOk: async () => {
        setActionLoading(true)
        try {
          await suppliersApi.blockSupplier(supplier.id!)
          Toast.success(t('suppliers.messages.blockSuccess', { name: supplier.name }))
          fetchSupplier()
        } catch {
          Toast.error(t('suppliers.messages.blockError'))
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [supplier, suppliersApi, fetchSupplier, t])

  // Handle delete supplier
  const handleDelete = useCallback(async () => {
    if (!supplier?.id) return
    Modal.confirm({
      title: t('suppliers.confirm.deleteTitle'),
      content: t('suppliers.confirm.deleteContent', { name: supplier.name }),
      okText: t('suppliers.confirm.deleteOk'),
      cancelText: t('common:actions.cancel'),
      okButtonProps: { type: 'danger' },
      onOk: async () => {
        setActionLoading(true)
        try {
          await suppliersApi.deleteSupplier(supplier.id!)
          Toast.success(t('suppliers.messages.deleteSuccess', { name: supplier.name }))
          navigate('/partner/suppliers')
        } catch {
          Toast.error(t('suppliers.messages.deleteError'))
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [supplier, suppliersApi, navigate, t])

  // Handle edit supplier
  const handleEdit = useCallback(() => {
    if (supplier?.id) {
      navigate(`/partner/suppliers/${supplier.id}/edit`)
    }
  }, [supplier, navigate])

  // Render basic info
  const renderBasicInfo = () => {
    if (!supplier) return null

    const data = [
      { key: t('supplierDetail.fields.code'), value: supplier.code || '-' },
      { key: t('supplierDetail.fields.name'), value: supplier.name || '-' },
      { key: t('supplierDetail.fields.shortName'), value: supplier.short_name || '-' },
      {
        key: t('supplierDetail.fields.status'),
        value: supplier.status ? (
          <Tag color={STATUS_TAG_COLORS[supplier.status]}>
            {t(`suppliers.status.${supplier.status}`)}
          </Tag>
        ) : (
          '-'
        ),
      },
      {
        key: t('supplierDetail.fields.rating'),
        value:
          supplier.rating !== undefined ? (
            <div className="supplier-rating">
              <Rating value={supplier.rating} disabled allowHalf size="small" />
              <Text type="secondary" className="rating-text">
                {t('suppliers.form.ratingScore', { score: supplier.rating })}
              </Text>
            </div>
          ) : (
            t('suppliers.form.ratingNotRated')
          ),
      },
    ]

    return <Descriptions data={data} row className="supplier-basic-info" />
  }

  // Render contact info
  const renderContactInfo = () => {
    if (!supplier) return null

    const data = [
      { key: t('supplierDetail.fields.contactName'), value: supplier.contact_name || '-' },
      { key: t('supplierDetail.fields.phone'), value: supplier.phone || '-' },
      { key: t('supplierDetail.fields.email'), value: supplier.email || '-' },
      { key: t('supplierDetail.fields.taxId'), value: supplier.tax_id || '-' },
    ]

    return <Descriptions data={data} row className="supplier-contact-info" />
  }

  // Render address info
  const renderAddressInfo = () => {
    if (!supplier) return null

    const data = [
      { key: t('supplierDetail.fields.country'), value: supplier.country || '-' },
      { key: t('supplierDetail.fields.province'), value: supplier.province || '-' },
      { key: t('supplierDetail.fields.city'), value: supplier.city || '-' },
      { key: t('supplierDetail.fields.address'), value: supplier.address || '-' },
      { key: t('supplierDetail.fields.postalCode'), value: supplier.postal_code || '-' },
      { key: t('supplierDetail.fields.fullAddress'), value: supplier.full_address || '-' },
    ]

    return <Descriptions data={data} row className="supplier-address-info" />
  }

  // Render bank info
  const renderBankInfo = () => {
    if (!supplier) return null

    const data = [
      { key: t('supplierDetail.fields.bankName'), value: supplier.bank_name || '-' },
      { key: t('supplierDetail.fields.bankAccount'), value: supplier.bank_account || '-' },
      { key: t('supplierDetail.fields.bankAccountName'), value: supplier.bank_account_name || '-' },
    ]

    return <Descriptions data={data} row className="supplier-bank-info" />
  }

  // Render payment terms info
  const renderPaymentInfo = () => {
    if (!supplier) return null

    const data = [
      {
        key: t('supplierDetail.fields.creditLimit'),
        value: supplier.credit_limit !== undefined ? formatCurrency(supplier.credit_limit) : '-',
      },
      {
        key: t('supplierDetail.fields.paymentTermDays'),
        value:
          supplier.payment_term_days !== undefined
            ? `${supplier.payment_term_days} ${t('suppliers.form.creditDaysSuffix')}`
            : '-',
      },
    ]

    return <Descriptions data={data} row className="supplier-payment-info" />
  }

  // Render timestamps
  const renderTimestamps = () => {
    if (!supplier) return null

    const data = [
      {
        key: t('supplierDetail.fields.createdAt'),
        value: supplier.created_at ? formatDateTime(supplier.created_at) : '-',
      },
      {
        key: t('supplierDetail.fields.updatedAt'),
        value: supplier.updated_at ? formatDateTime(supplier.updated_at) : '-',
      },
    ]

    return <Descriptions data={data} row className="supplier-timestamps" />
  }

  // Render action buttons based on status
  const renderActions = () => {
    if (!supplier) return null

    const status = supplier.status

    return (
      <Space>
        <Button icon={<IconEdit />} onClick={handleEdit} disabled={actionLoading}>
          {t('common:actions.edit')}
        </Button>
        {status !== 'active' && status !== 'blocked' && (
          <Button
            type="primary"
            icon={<IconPlay />}
            onClick={handleActivate}
            loading={actionLoading}
          >
            {t('suppliers.actions.activate')}
          </Button>
        )}
        {status === 'blocked' && (
          <Button
            type="primary"
            icon={<IconPlay />}
            onClick={handleActivate}
            loading={actionLoading}
          >
            {t('suppliers.actions.activate')}
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
              {t('suppliers.actions.deactivate')}
            </Button>
            <Button type="danger" onClick={handleBlock} loading={actionLoading}>
              {t('suppliers.actions.block')}
            </Button>
          </>
        )}
        <Button type="danger" icon={<IconDelete />} onClick={handleDelete} loading={actionLoading}>
          {t('suppliers.actions.delete')}
        </Button>
      </Space>
    )
  }

  if (loading) {
    return (
      <Container size="lg" className="supplier-detail-page">
        <div className="loading-container">
          <Spin size="large" />
        </div>
      </Container>
    )
  }

  if (!supplier) {
    return (
      <Container size="lg" className="supplier-detail-page">
        <Empty
          title={t('supplierDetail.notExist')}
          description={t('supplierDetail.notExistDesc')}
        />
      </Container>
    )
  }

  return (
    <Container size="lg" className="supplier-detail-page">
      {/* Header */}
      <div className="page-header">
        <div className="header-left">
          <Button
            icon={<IconArrowLeft />}
            theme="borderless"
            onClick={() => navigate('/partner/suppliers')}
          >
            {t('supplierDetail.back')}
          </Button>
          <Title heading={4} className="page-title">
            {t('supplierDetail.title')}
          </Title>
          {supplier.status && (
            <Tag color={STATUS_TAG_COLORS[supplier.status]} size="large">
              {t(`suppliers.status.${supplier.status}`)}
            </Tag>
          )}
        </div>
        <div className="header-right">{renderActions()}</div>
      </div>

      {/* Supplier Name Display */}
      <Card className="supplier-name-card">
        <div className="supplier-name-display">
          <Text className="supplier-code">{supplier.code}</Text>
          <Title heading={3} className="supplier-name">
            {supplier.name}
          </Title>
          {supplier.short_name && <Text type="secondary">{supplier.short_name}</Text>}
        </div>
      </Card>

      {/* Basic Info Card */}
      <Card className="info-card" title={t('suppliers.form.basicInfo')}>
        {renderBasicInfo()}
      </Card>

      {/* Contact Info Card */}
      <Card className="info-card" title={t('suppliers.form.contactInfo')}>
        {renderContactInfo()}
      </Card>

      {/* Address Info Card */}
      <Card className="info-card" title={t('suppliers.form.addressInfo')}>
        {renderAddressInfo()}
      </Card>

      {/* Bank Info Card */}
      <Card className="info-card" title={t('suppliers.form.bankInfo')}>
        {renderBankInfo()}
      </Card>

      {/* Payment Terms Card */}
      <Card className="info-card" title={t('suppliers.form.purchaseSettings')}>
        {renderPaymentInfo()}
      </Card>

      {/* Timestamps Card */}
      <Card className="info-card" title={t('supplierDetail.timestamps')}>
        {renderTimestamps()}
      </Card>

      {/* Notes Card */}
      {supplier.notes && (
        <Card className="info-card" title={t('supplierDetail.notes')}>
          <Text>{supplier.notes}</Text>
        </Card>
      )}
    </Container>
  )
}
