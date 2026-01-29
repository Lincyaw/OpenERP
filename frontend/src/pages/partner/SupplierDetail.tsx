import { useState, useEffect, useCallback, useMemo } from 'react'
import { Card, Toast, Modal, Empty, Typography, Rating } from '@douyinfe/semi-ui-19'
import { IconEdit, IconPlay, IconStop, IconDelete } from '@douyinfe/semi-icons'
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
  getSupplierById,
  activateSupplier,
  deactivateSupplier,
  blockSupplier,
  deleteSupplier,
} from '@/api/suppliers/suppliers'
import type { HandlerSupplierResponse, HandlerSupplierResponseStatus } from '@/api/models'
import './SupplierDetail.css'

const { Text } = Typography

// Status variant mapping for DetailPageHeader
const STATUS_VARIANTS: Record<HandlerSupplierResponseStatus, 'default' | 'success' | 'danger'> = {
  active: 'success',
  inactive: 'default',
  blocked: 'danger',
}

/**
 * Supplier Detail Page
 *
 * Features:
 * - Display complete supplier information using DetailPageHeader
 * - Display bank and payment info
 * - Status action buttons (activate, deactivate, block)
 * - Navigate to edit page
 */
export default function SupplierDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { t } = useTranslation(['partner', 'common'])
  const { formatCurrency, formatDateTime } = useFormatters()

  const [supplier, setSupplier] = useState<HandlerSupplierResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState(false)

  // Fetch supplier details
  const fetchSupplier = useCallback(async () => {
    if (!id) return

    setLoading(true)
    try {
      const response = await getSupplierById(id)
      if (response.status === 200 && response.data.success && response.data.data) {
        setSupplier(response.data.data)
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
  }, [id, navigate, t])

  useEffect(() => {
    fetchSupplier()
  }, [fetchSupplier])

  // Handle activate supplier
  const handleActivate = useCallback(async () => {
    if (!supplier?.id) return
    setActionLoading(true)
    try {
      await activateSupplier(supplier.id, {})
      Toast.success(t('suppliers.messages.activateSuccess', { name: supplier.name }))
      fetchSupplier()
    } catch {
      Toast.error(t('suppliers.messages.activateError'))
    } finally {
      setActionLoading(false)
    }
  }, [supplier, fetchSupplier, t])

  // Handle deactivate supplier
  const handleDeactivate = useCallback(async () => {
    if (!supplier?.id) return
    setActionLoading(true)
    try {
      await deactivateSupplier(supplier.id, {})
      Toast.success(t('suppliers.messages.deactivateSuccess', { name: supplier.name }))
      fetchSupplier()
    } catch {
      Toast.error(t('suppliers.messages.deactivateError'))
    } finally {
      setActionLoading(false)
    }
  }, [supplier, fetchSupplier, t])

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
          await blockSupplier(supplier.id!, {})
          Toast.success(t('suppliers.messages.blockSuccess', { name: supplier.name }))
          fetchSupplier()
        } catch {
          Toast.error(t('suppliers.messages.blockError'))
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [supplier, fetchSupplier, t])

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
          await deleteSupplier(supplier.id!)
          Toast.success(t('suppliers.messages.deleteSuccess', { name: supplier.name }))
          navigate('/partner/suppliers')
        } catch {
          Toast.error(t('suppliers.messages.deleteError'))
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [supplier, navigate, t])

  // Handle edit supplier
  const handleEdit = useCallback(() => {
    if (supplier?.id) {
      navigate(`/partner/suppliers/${supplier.id}/edit`)
    }
  }, [supplier, navigate])

  // Build status for DetailPageHeader
  const headerStatus = useMemo((): DetailPageHeaderStatus | undefined => {
    if (!supplier?.status) return undefined
    return {
      label: t(`suppliers.status.${supplier.status}`),
      variant: STATUS_VARIANTS[supplier.status] || 'default',
    }
  }, [supplier, t])

  // Build metrics for DetailPageHeader
  const headerMetrics = useMemo((): DetailPageHeaderMetric[] => {
    if (!supplier) return []
    return [
      {
        label: t('supplierDetail.fields.rating'),
        value:
          supplier.rating !== undefined
            ? t('suppliers.form.ratingScore', { score: supplier.rating })
            : '-',
      },
      {
        label: t('supplierDetail.fields.creditLimit'),
        value: supplier.credit_limit !== undefined ? formatCurrency(supplier.credit_limit) : '-',
        variant: 'primary',
      },
      {
        label: t('supplierDetail.fields.paymentTermDays'),
        value:
          supplier.payment_term_days !== undefined
            ? `${supplier.payment_term_days} ${t('suppliers.form.creditDaysSuffix')}`
            : '-',
      },
    ]
  }, [supplier, t, formatCurrency])

  // Build primary action for DetailPageHeader
  const primaryAction = useMemo((): DetailPageHeaderAction | undefined => {
    if (!supplier) return undefined
    const status = supplier.status

    if (status !== 'active') {
      return {
        key: 'activate',
        label: t('suppliers.actions.activate'),
        icon: <IconPlay />,
        type: 'primary',
        onClick: handleActivate,
        loading: actionLoading,
      }
    }
    return {
      key: 'deactivate',
      label: t('suppliers.actions.deactivate'),
      icon: <IconStop />,
      type: 'warning',
      onClick: handleDeactivate,
      loading: actionLoading,
    }
  }, [supplier, t, handleActivate, handleDeactivate, actionLoading])

  // Build secondary actions for DetailPageHeader
  const secondaryActions = useMemo((): DetailPageHeaderAction[] => {
    if (!supplier) return []
    const status = supplier.status
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
        key: 'block',
        label: t('suppliers.actions.block'),
        type: 'danger',
        onClick: handleBlock,
        loading: actionLoading,
      })
    }

    actions.push({
      key: 'delete',
      label: t('suppliers.actions.delete'),
      icon: <IconDelete />,
      type: 'danger',
      onClick: handleDelete,
      loading: actionLoading,
    })

    return actions
  }, [supplier, t, handleEdit, handleBlock, handleDelete, actionLoading])

  // Render basic info
  const renderBasicInfo = () => {
    if (!supplier) return null

    return (
      <div className="info-grid">
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('supplierDetail.fields.code')}
          </Text>
          <Text strong className="info-value code-value">
            {supplier.code || '-'}
          </Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('supplierDetail.fields.name')}
          </Text>
          <Text className="info-value">{supplier.name || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('supplierDetail.fields.shortName')}
          </Text>
          <Text className="info-value">{supplier.short_name || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('supplierDetail.fields.rating')}
          </Text>
          <div className="info-value">
            {supplier.rating !== undefined ? (
              <div className="supplier-rating">
                <Rating value={supplier.rating} disabled allowHalf size="small" />
                <Text type="secondary" className="rating-text">
                  {t('suppliers.form.ratingScore', { score: supplier.rating })}
                </Text>
              </div>
            ) : (
              t('suppliers.form.ratingNotRated')
            )}
          </div>
        </div>
      </div>
    )
  }

  // Render contact info
  const renderContactInfo = () => {
    if (!supplier) return null

    return (
      <div className="info-grid">
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('supplierDetail.fields.contactName')}
          </Text>
          <Text className="info-value">{supplier.contact_name || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('supplierDetail.fields.phone')}
          </Text>
          <Text className="info-value">{supplier.phone || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('supplierDetail.fields.email')}
          </Text>
          <Text className="info-value">{supplier.email || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('supplierDetail.fields.taxId')}
          </Text>
          <Text className="info-value code-value">{supplier.tax_id || '-'}</Text>
        </div>
      </div>
    )
  }

  // Render address info
  const renderAddressInfo = () => {
    if (!supplier) return null

    return (
      <div className="info-grid">
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('supplierDetail.fields.country')}
          </Text>
          <Text className="info-value">{supplier.country || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('supplierDetail.fields.province')}
          </Text>
          <Text className="info-value">{supplier.province || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('supplierDetail.fields.city')}
          </Text>
          <Text className="info-value">{supplier.city || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('supplierDetail.fields.address')}
          </Text>
          <Text className="info-value">{supplier.address || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('supplierDetail.fields.postalCode')}
          </Text>
          <Text className="info-value code-value">{supplier.postal_code || '-'}</Text>
        </div>
        <div className="info-item info-item--full">
          <Text type="secondary" className="info-label">
            {t('supplierDetail.fields.fullAddress')}
          </Text>
          <Text className="info-value">{supplier.full_address || '-'}</Text>
        </div>
      </div>
    )
  }

  // Render bank info
  const renderBankInfo = () => {
    if (!supplier) return null

    return (
      <div className="info-grid">
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('supplierDetail.fields.bankName')}
          </Text>
          <Text className="info-value">{supplier.bank_name || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('supplierDetail.fields.bankAccount')}
          </Text>
          <Text className="info-value code-value">{supplier.bank_account || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('supplierDetail.fields.bankAccountName')}
          </Text>
          <Text className="info-value">{supplier.bank_account_name || '-'}</Text>
        </div>
      </div>
    )
  }

  // Render payment terms info
  const renderPaymentInfo = () => {
    if (!supplier) return null

    return (
      <div className="info-grid">
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('supplierDetail.fields.creditLimit')}
          </Text>
          <Text className="info-value credit-value">
            {supplier.credit_limit !== undefined ? formatCurrency(supplier.credit_limit) : '-'}
          </Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('supplierDetail.fields.paymentTermDays')}
          </Text>
          <Text className="info-value">
            {supplier.payment_term_days !== undefined
              ? `${supplier.payment_term_days} ${t('suppliers.form.creditDaysSuffix')}`
              : '-'}
          </Text>
        </div>
      </div>
    )
  }

  // Render timestamps
  const renderTimestamps = () => {
    if (!supplier) return null

    return (
      <div className="info-grid">
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('supplierDetail.fields.createdAt')}
          </Text>
          <Text className="info-value">
            {supplier.created_at ? formatDateTime(supplier.created_at) : '-'}
          </Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('supplierDetail.fields.updatedAt')}
          </Text>
          <Text className="info-value">
            {supplier.updated_at ? formatDateTime(supplier.updated_at) : '-'}
          </Text>
        </div>
      </div>
    )
  }

  if (loading) {
    return (
      <Container size="lg" className="supplier-detail-page">
        <DetailPageHeader
          title={t('supplierDetail.title')}
          loading={true}
          showBack={true}
          onBack={() => navigate('/partner/suppliers')}
          backLabel={t('supplierDetail.back')}
        />
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
      {/* Unified Header */}
      <DetailPageHeader
        title={t('supplierDetail.title')}
        documentNumber={supplier.code}
        status={headerStatus}
        metrics={headerMetrics}
        primaryAction={primaryAction}
        secondaryActions={secondaryActions}
        onBack={() => navigate('/partner/suppliers')}
        backLabel={t('supplierDetail.back')}
        titleSuffix={
          supplier.short_name ? (
            <span className="supplier-short-name-suffix">{supplier.short_name}</span>
          ) : undefined
        }
      />

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
