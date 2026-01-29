import { useState, useEffect, useCallback } from 'react'
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
  IconStar,
} from '@douyinfe/semi-icons'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Container } from '@/components/common/layout'
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

const { Title, Text } = Typography

// Status tag color mapping
const STATUS_TAG_COLORS: Record<HandlerWarehouseResponseStatus, 'green' | 'grey'> = {
  enabled: 'green',
  disabled: 'grey',
}

// Type tag color mapping
const TYPE_TAG_COLORS: Record<
  HandlerWarehouseResponseType,
  'blue' | 'cyan' | 'violet' | 'green' | 'orange'
> = {
  normal: 'blue',
  virtual: 'cyan',
  transit: 'violet',
}

/**
 * Warehouse Detail Page
 *
 * Features:
 * - Display complete warehouse information
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

  // Render basic info
  const renderBasicInfo = () => {
    if (!warehouse) return null

    const data = [
      { key: t('warehouseDetail.fields.code'), value: warehouse.code || '-' },
      { key: t('warehouseDetail.fields.name'), value: warehouse.name || '-' },
      { key: t('warehouseDetail.fields.shortName'), value: warehouse.short_name || '-' },
      {
        key: t('warehouseDetail.fields.type'),
        value: warehouse.type ? (
          <Tag color={TYPE_TAG_COLORS[warehouse.type]}>
            {t(`warehouses.type.${warehouse.type}`)}
          </Tag>
        ) : (
          '-'
        ),
      },
      {
        key: t('warehouseDetail.fields.status'),
        value: warehouse.status ? (
          <Tag color={STATUS_TAG_COLORS[warehouse.status]}>
            {t(`warehouses.status.${warehouse.status}`)}
          </Tag>
        ) : (
          '-'
        ),
      },
      {
        key: t('warehouseDetail.fields.isDefault'),
        value: warehouse.is_default ? (
          <Tag color="light-blue" size="small">
            <IconStar size="small" /> {t('warehouses.defaultTag')}
          </Tag>
        ) : (
          t('warehouseDetail.fields.notDefault')
        ),
      },
      {
        key: t('warehouseDetail.fields.sortOrder'),
        value: warehouse.sort_order ?? '-',
      },
    ]

    return <Descriptions data={data} row className="warehouse-basic-info" />
  }

  // Render contact info
  const renderContactInfo = () => {
    if (!warehouse) return null

    const data = [
      { key: t('warehouseDetail.fields.managerName'), value: warehouse.manager_name || '-' },
      { key: t('warehouseDetail.fields.phone'), value: warehouse.phone || '-' },
      { key: t('warehouseDetail.fields.email'), value: warehouse.email || '-' },
    ]

    return <Descriptions data={data} row className="warehouse-contact-info" />
  }

  // Render address info
  const renderAddressInfo = () => {
    if (!warehouse) return null

    const data = [
      { key: t('warehouseDetail.fields.country'), value: warehouse.country || '-' },
      { key: t('warehouseDetail.fields.province'), value: warehouse.province || '-' },
      { key: t('warehouseDetail.fields.city'), value: warehouse.city || '-' },
      { key: t('warehouseDetail.fields.address'), value: warehouse.address || '-' },
      { key: t('warehouseDetail.fields.postalCode'), value: warehouse.postal_code || '-' },
      { key: t('warehouseDetail.fields.fullAddress'), value: warehouse.full_address || '-' },
    ]

    return <Descriptions data={data} row className="warehouse-address-info" />
  }

  // Render timestamps
  const renderTimestamps = () => {
    if (!warehouse) return null

    const data = [
      {
        key: t('warehouseDetail.fields.createdAt'),
        value: warehouse.created_at ? formatDateTime(warehouse.created_at) : '-',
      },
      {
        key: t('warehouseDetail.fields.updatedAt'),
        value: warehouse.updated_at ? formatDateTime(warehouse.updated_at) : '-',
      },
    ]

    return <Descriptions data={data} row className="warehouse-timestamps" />
  }

  // Render action buttons based on status
  const renderActions = () => {
    if (!warehouse) return null

    const status = warehouse.status

    return (
      <Space>
        <Button icon={<IconEdit />} onClick={handleEdit} disabled={actionLoading}>
          {t('common:actions.edit')}
        </Button>
        {!warehouse.is_default && (
          <Button
            type="secondary"
            icon={<IconStar />}
            onClick={handleSetDefault}
            loading={actionLoading}
          >
            {t('warehouses.actions.setDefault')}
          </Button>
        )}
        {status === 'disabled' && (
          <Button type="primary" icon={<IconPlay />} onClick={handleEnable} loading={actionLoading}>
            {t('warehouses.actions.enable')}
          </Button>
        )}
        {status === 'enabled' && !warehouse.is_default && (
          <Button
            type="warning"
            icon={<IconStop />}
            onClick={handleDisable}
            loading={actionLoading}
          >
            {t('warehouses.actions.disable')}
          </Button>
        )}
        {!warehouse.is_default && (
          <Button
            type="danger"
            icon={<IconDelete />}
            onClick={handleDelete}
            loading={actionLoading}
          >
            {t('warehouses.actions.delete')}
          </Button>
        )}
      </Space>
    )
  }

  if (loading) {
    return (
      <Container size="lg" className="warehouse-detail-page">
        <div className="loading-container">
          <Spin size="large" />
        </div>
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
      {/* Header */}
      <div className="page-header">
        <div className="header-left">
          <Button
            icon={<IconArrowLeft />}
            theme="borderless"
            onClick={() => navigate('/partner/warehouses')}
          >
            {t('warehouseDetail.back')}
          </Button>
          <Title heading={4} className="page-title">
            {t('warehouseDetail.title')}
          </Title>
          {warehouse.status && (
            <Tag color={STATUS_TAG_COLORS[warehouse.status]} size="large">
              {t(`warehouses.status.${warehouse.status}`)}
            </Tag>
          )}
          {warehouse.is_default && (
            <Tag color="light-blue" size="large">
              <IconStar size="small" /> {t('warehouses.defaultTag')}
            </Tag>
          )}
        </div>
        <div className="header-right">{renderActions()}</div>
      </div>

      {/* Warehouse Name Display */}
      <Card className="warehouse-name-card">
        <div className="warehouse-name-display">
          <Text className="warehouse-code">{warehouse.code}</Text>
          <Title heading={3} className="warehouse-name">
            {warehouse.name}
          </Title>
          {warehouse.short_name && <Text type="secondary">{warehouse.short_name}</Text>}
        </div>
      </Card>

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
