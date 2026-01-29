import { useState, useEffect, useCallback, useMemo } from 'react'
import { Card, Tag, Toast, Modal, Empty, Typography } from '@douyinfe/semi-ui-19'
import { IconEdit, IconPlay, IconStop, IconDelete, IconStar } from '@douyinfe/semi-icons'
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
  getWarehouseById,
  enableWarehouse,
  disableWarehouse,
  setDefaultWarehouse,
  deleteWarehouse,
} from '@/api/warehouses/warehouses'
import type {
  HandlerWarehouseResponse,
  HandlerWarehouseResponseStatus,
  HandlerWarehouseResponseType,
} from '@/api/models'
import './WarehouseDetail.css'

const { Text } = Typography

// Status variant mapping for DetailPageHeader
const STATUS_VARIANTS: Record<HandlerWarehouseResponseStatus, 'default' | 'success'> = {
  enabled: 'success',
  disabled: 'default',
}

// Type tag color mapping
const TYPE_TAG_COLORS: Record<HandlerWarehouseResponseType, 'blue' | 'cyan' | 'violet'> = {
  normal: 'blue',
  virtual: 'cyan',
  transit: 'violet',
}

/**
 * Warehouse Detail Page
 *
 * Features:
 * - Display complete warehouse information using DetailPageHeader
 * - Display location and contact info
 * - Status action buttons (enable, disable, set default)
 * - Navigate to edit page
 */
export default function WarehouseDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { t } = useTranslation(['partner', 'common'])
  const { formatDateTime } = useFormatters()

  const [warehouse, setWarehouse] = useState<HandlerWarehouseResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState(false)

  // Fetch warehouse details
  const fetchWarehouse = useCallback(async () => {
    if (!id) return

    setLoading(true)
    try {
      const response = await getWarehouseById(id)
      if (response.status === 200 && response.data.success && response.data.data) {
        setWarehouse(response.data.data)
      } else {
        Toast.error(t('warehouseDetail.messages.fetchError'))
        navigate('/partner/warehouses')
      }
    } catch {
      Toast.error(t('warehouseDetail.messages.fetchError'))
      navigate('/partner/warehouses')
    } finally {
      setLoading(false)
    }
  }, [id, navigate, t])

  useEffect(() => {
    fetchWarehouse()
  }, [fetchWarehouse])

  // Handle enable warehouse
  const handleEnable = useCallback(async () => {
    if (!warehouse?.id) return
    setActionLoading(true)
    try {
      await enableWarehouse(warehouse.id, {})
      Toast.success(t('warehouses.messages.enableSuccess', { name: warehouse.name }))
      fetchWarehouse()
    } catch {
      Toast.error(t('warehouses.messages.enableError'))
    } finally {
      setActionLoading(false)
    }
  }, [warehouse, fetchWarehouse, t])

  // Handle disable warehouse
  const handleDisable = useCallback(async () => {
    if (!warehouse?.id) return
    if (warehouse.is_default) {
      Toast.warning(t('warehouses.messages.disableDefaultWarning'))
      return
    }
    setActionLoading(true)
    try {
      await disableWarehouse(warehouse.id, {})
      Toast.success(t('warehouses.messages.disableSuccess', { name: warehouse.name }))
      fetchWarehouse()
    } catch {
      Toast.error(t('warehouses.messages.disableError'))
    } finally {
      setActionLoading(false)
    }
  }, [warehouse, fetchWarehouse, t])

  // Handle set default warehouse
  const handleSetDefault = useCallback(async () => {
    if (!warehouse?.id) return
    if (warehouse.is_default) {
      Toast.info(t('warehouses.messages.alreadyDefault'))
      return
    }
    setActionLoading(true)
    try {
      await setDefaultWarehouse(warehouse.id, {})
      Toast.success(t('warehouses.messages.setDefaultSuccess', { name: warehouse.name }))
      fetchWarehouse()
    } catch {
      Toast.error(t('warehouses.messages.setDefaultError'))
    } finally {
      setActionLoading(false)
    }
  }, [warehouse, fetchWarehouse, t])

  // Handle delete warehouse
  const handleDelete = useCallback(async () => {
    if (!warehouse?.id) return
    if (warehouse.is_default) {
      Toast.warning(t('warehouses.messages.deleteDefaultWarning'))
      return
    }
    Modal.confirm({
      title: t('warehouses.confirm.deleteTitle'),
      content: t('warehouses.confirm.deleteContent', { name: warehouse.name }),
      okText: t('warehouses.confirm.deleteOk'),
      cancelText: t('common:actions.cancel'),
      okButtonProps: { type: 'danger' },
      onOk: async () => {
        setActionLoading(true)
        try {
          await deleteWarehouse(warehouse.id!)
          Toast.success(t('warehouses.messages.deleteSuccess', { name: warehouse.name }))
          navigate('/partner/warehouses')
        } catch {
          Toast.error(t('warehouses.messages.deleteError'))
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [warehouse, navigate, t])

  // Handle edit warehouse
  const handleEdit = useCallback(() => {
    if (warehouse?.id) {
      navigate(`/partner/warehouses/${warehouse.id}/edit`)
    }
  }, [warehouse, navigate])

  // Build status for DetailPageHeader
  const headerStatus = useMemo((): DetailPageHeaderStatus | undefined => {
    if (!warehouse?.status) return undefined
    return {
      label: t(`warehouses.status.${warehouse.status}`),
      variant: STATUS_VARIANTS[warehouse.status] || 'default',
    }
  }, [warehouse, t])

  // Build metrics for DetailPageHeader
  const headerMetrics = useMemo((): DetailPageHeaderMetric[] => {
    if (!warehouse) return []
    return [
      {
        label: t('warehouseDetail.fields.type'),
        value: warehouse.type ? t(`warehouses.type.${warehouse.type}`) : '-',
      },
      {
        label: t('warehouseDetail.fields.managerName'),
        value: warehouse.manager_name || '-',
      },
      {
        label: t('warehouseDetail.fields.city'),
        value: warehouse.city || '-',
      },
      {
        label: t('warehouseDetail.fields.isDefault'),
        value: warehouse.is_default ? t('warehouses.defaultTag') : '-',
        variant: warehouse.is_default ? 'primary' : 'default',
      },
    ]
  }, [warehouse, t])

  // Build primary action for DetailPageHeader
  const primaryAction = useMemo((): DetailPageHeaderAction | undefined => {
    if (!warehouse) return undefined
    const status = warehouse.status

    if (status === 'disabled') {
      return {
        key: 'enable',
        label: t('warehouses.actions.enable'),
        icon: <IconPlay />,
        type: 'primary',
        onClick: handleEnable,
        loading: actionLoading,
      }
    }
    if (!warehouse.is_default) {
      return {
        key: 'setDefault',
        label: t('warehouses.actions.setDefault'),
        icon: <IconStar />,
        type: 'primary',
        onClick: handleSetDefault,
        loading: actionLoading,
      }
    }
    return undefined
  }, [warehouse, t, handleEnable, handleSetDefault, actionLoading])

  // Build secondary actions for DetailPageHeader
  const secondaryActions = useMemo((): DetailPageHeaderAction[] => {
    if (!warehouse) return []
    const status = warehouse.status
    const actions: DetailPageHeaderAction[] = []

    actions.push({
      key: 'edit',
      label: t('common:actions.edit'),
      icon: <IconEdit />,
      onClick: handleEdit,
      disabled: actionLoading,
    })

    if (status === 'enabled' && !warehouse.is_default) {
      actions.push({
        key: 'disable',
        label: t('warehouses.actions.disable'),
        icon: <IconStop />,
        type: 'warning',
        onClick: handleDisable,
        loading: actionLoading,
      })
    }

    if (!warehouse.is_default) {
      actions.push({
        key: 'delete',
        label: t('warehouses.actions.delete'),
        icon: <IconDelete />,
        type: 'danger',
        onClick: handleDelete,
        loading: actionLoading,
      })
    }

    return actions
  }, [warehouse, t, handleEdit, handleDisable, handleDelete, actionLoading])

  // Render basic info
  const renderBasicInfo = () => {
    if (!warehouse) return null

    return (
      <div className="info-grid">
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('warehouseDetail.fields.code')}
          </Text>
          <Text strong className="info-value code-value">
            {warehouse.code || '-'}
          </Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('warehouseDetail.fields.name')}
          </Text>
          <Text className="info-value">{warehouse.name || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('warehouseDetail.fields.shortName')}
          </Text>
          <Text className="info-value">{warehouse.short_name || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('warehouseDetail.fields.type')}
          </Text>
          <div className="info-value">
            {warehouse.type ? (
              <Tag color={TYPE_TAG_COLORS[warehouse.type]}>
                {t(`warehouses.type.${warehouse.type}`)}
              </Tag>
            ) : (
              '-'
            )}
          </div>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('warehouseDetail.fields.isDefault')}
          </Text>
          <div className="info-value">
            {warehouse.is_default ? (
              <Tag color="light-blue" size="small">
                <IconStar size="small" /> {t('warehouses.defaultTag')}
              </Tag>
            ) : (
              t('warehouseDetail.fields.notDefault')
            )}
          </div>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('warehouseDetail.fields.sortOrder')}
          </Text>
          <Text className="info-value">{warehouse.sort_order ?? '-'}</Text>
        </div>
      </div>
    )
  }

  // Render contact info
  const renderContactInfo = () => {
    if (!warehouse) return null

    return (
      <div className="info-grid">
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('warehouseDetail.fields.managerName')}
          </Text>
          <Text className="info-value">{warehouse.manager_name || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('warehouseDetail.fields.phone')}
          </Text>
          <Text className="info-value">{warehouse.phone || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('warehouseDetail.fields.email')}
          </Text>
          <Text className="info-value">{warehouse.email || '-'}</Text>
        </div>
      </div>
    )
  }

  // Render address info
  const renderAddressInfo = () => {
    if (!warehouse) return null

    return (
      <div className="info-grid">
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('warehouseDetail.fields.country')}
          </Text>
          <Text className="info-value">{warehouse.country || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('warehouseDetail.fields.province')}
          </Text>
          <Text className="info-value">{warehouse.province || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('warehouseDetail.fields.city')}
          </Text>
          <Text className="info-value">{warehouse.city || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('warehouseDetail.fields.address')}
          </Text>
          <Text className="info-value">{warehouse.address || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('warehouseDetail.fields.postalCode')}
          </Text>
          <Text className="info-value code-value">{warehouse.postal_code || '-'}</Text>
        </div>
        <div className="info-item info-item--full">
          <Text type="secondary" className="info-label">
            {t('warehouseDetail.fields.fullAddress')}
          </Text>
          <Text className="info-value">{warehouse.full_address || '-'}</Text>
        </div>
      </div>
    )
  }

  // Render timestamps
  const renderTimestamps = () => {
    if (!warehouse) return null

    return (
      <div className="info-grid">
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('warehouseDetail.fields.createdAt')}
          </Text>
          <Text className="info-value">
            {warehouse.created_at ? formatDateTime(warehouse.created_at) : '-'}
          </Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('warehouseDetail.fields.updatedAt')}
          </Text>
          <Text className="info-value">
            {warehouse.updated_at ? formatDateTime(warehouse.updated_at) : '-'}
          </Text>
        </div>
      </div>
    )
  }

  if (loading) {
    return (
      <Container size="lg" className="warehouse-detail-page">
        <DetailPageHeader
          title={t('warehouseDetail.title')}
          loading={true}
          showBack={true}
          onBack={() => navigate('/partner/warehouses')}
          backLabel={t('warehouseDetail.back')}
        />
      </Container>
    )
  }

  if (!warehouse) {
    return (
      <Container size="lg" className="warehouse-detail-page">
        <Empty
          title={t('warehouseDetail.notExist')}
          description={t('warehouseDetail.notExistDesc')}
        />
      </Container>
    )
  }

  return (
    <Container size="lg" className="warehouse-detail-page">
      {/* Unified Header */}
      <DetailPageHeader
        title={t('warehouseDetail.title')}
        documentNumber={warehouse.code}
        status={headerStatus}
        metrics={headerMetrics}
        primaryAction={primaryAction}
        secondaryActions={secondaryActions}
        onBack={() => navigate('/partner/warehouses')}
        backLabel={t('warehouseDetail.back')}
        titleSuffix={
          warehouse.is_default ? (
            <Tag color="light-blue" size="large">
              <IconStar size="small" /> {t('warehouses.defaultTag')}
            </Tag>
          ) : undefined
        }
      />

      {/* Basic Info Card */}
      <Card className="info-card" title={t('warehouses.form.basicInfo')}>
        {renderBasicInfo()}
      </Card>

      {/* Contact Info Card */}
      <Card className="info-card" title={t('warehouses.form.contactInfo')}>
        {renderContactInfo()}
      </Card>

      {/* Address Info Card */}
      <Card className="info-card" title={t('warehouses.form.addressInfo')}>
        {renderAddressInfo()}
      </Card>

      {/* Timestamps Card */}
      <Card className="info-card" title={t('warehouseDetail.timestamps')}>
        {renderTimestamps()}
      </Card>

      {/* Notes Card */}
      {warehouse.notes && (
        <Card className="info-card" title={t('warehouseDetail.notes')}>
          <Text>{warehouse.notes}</Text>
        </Card>
      )}
    </Container>
  )
}
