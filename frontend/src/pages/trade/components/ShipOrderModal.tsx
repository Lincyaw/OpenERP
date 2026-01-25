import { useState, useEffect, useCallback, useMemo } from 'react'
import { Modal, Select, Toast, Typography, Descriptions, Spin, Empty } from '@douyinfe/semi-ui-19'
import { useTranslation } from 'react-i18next'
import { getWarehouses } from '@/api/warehouses/warehouses'
import type { HandlerWarehouseResponse } from '@/api/models'
import './ShipOrderModal.css'

const { Text } = Typography

interface OrderInfo {
  id: string
  order_number: string
  customer_name?: string
  warehouse_id?: string
  item_count?: number
  total_quantity?: number
  payable_amount?: number
}

interface ShipOrderModalProps {
  visible: boolean
  order: OrderInfo | null
  loading?: boolean
  onConfirm: (warehouseId: string) => Promise<void>
  onCancel: () => void
}

/**
 * Format price for display
 */
function formatPrice(price?: number): string {
  if (price === undefined || price === null) return '-'
  return `¥${price.toFixed(2)}`
}

/**
 * Ship Order Modal Component
 *
 * Features:
 * - Display order summary information
 * - Warehouse selection dropdown (only active warehouses)
 * - Default to order's warehouse or default warehouse
 * - Confirm/Cancel actions
 */
export default function ShipOrderModal({
  visible,
  order,
  loading = false,
  onConfirm,
  onCancel,
}: ShipOrderModalProps) {
  const { t } = useTranslation('trade')
  const warehouseApi = useMemo(() => getWarehouses(), [])

  const [warehouses, setWarehouses] = useState<HandlerWarehouseResponse[]>([])
  const [selectedWarehouseId, setSelectedWarehouseId] = useState<string | undefined>(undefined)
  const [warehousesLoading, setWarehousesLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  // Fetch active warehouses
  const fetchWarehouses = useCallback(async () => {
    setWarehousesLoading(true)
    try {
      const response = await warehouseApi.getPartnerWarehouses({
        status: 'active',
        page_size: 100, // Fetch all active warehouses
      })
      if (response.success && response.data) {
        setWarehouses(response.data)
      }
    } catch {
      Toast.error(t('shipModal.messages.fetchWarehousesError'))
    } finally {
      setWarehousesLoading(false)
    }
  }, [warehouseApi, t])

  // Load warehouses when modal opens
  useEffect(() => {
    if (visible) {
      fetchWarehouses()
    }
  }, [visible, fetchWarehouses])

  // Set default warehouse selection when order or warehouses change
  useEffect(() => {
    if (visible && warehouses.length > 0 && order) {
      // Priority: order's warehouse > default warehouse > first active warehouse
      if (order.warehouse_id) {
        const orderWarehouse = warehouses.find((w) => w.id === order.warehouse_id)
        if (orderWarehouse) {
          setSelectedWarehouseId(order.warehouse_id)
          return
        }
      }

      const defaultWarehouse = warehouses.find((w) => w.is_default)
      if (defaultWarehouse) {
        setSelectedWarehouseId(defaultWarehouse.id)
        return
      }

      // Fallback to first warehouse
      if (warehouses[0]?.id) {
        setSelectedWarehouseId(warehouses[0].id)
      }
    }
  }, [visible, warehouses, order])

  // Reset state when modal closes
  useEffect(() => {
    if (!visible) {
      setSelectedWarehouseId(undefined)
      setWarehouses([])
    }
  }, [visible])

  // Handle confirm
  const handleConfirm = useCallback(async () => {
    if (!selectedWarehouseId) {
      Toast.warning(t('shipModal.warehouseRequired'))
      return
    }

    setSubmitting(true)
    try {
      await onConfirm(selectedWarehouseId)
    } finally {
      setSubmitting(false)
    }
  }, [selectedWarehouseId, onConfirm, t])

  // Warehouse options for select
  const warehouseOptions = useMemo(
    () =>
      warehouses.map((w) => ({
        value: w.id,
        label: w.is_default ? t('shipModal.defaultWarehouseLabel', { name: w.name }) : w.name,
      })),
    [warehouses, t]
  )

  // Order summary data
  const orderSummary = useMemo(() => {
    if (!order) return []
    return [
      { key: t('shipModal.orderNumber'), value: order.order_number },
      { key: t('shipModal.customerName'), value: order.customer_name || '-' },
      {
        key: t('shipModal.itemCount'),
        value: `${order.item_count || 0} ${t('shipModal.itemsUnit')}`,
      },
      { key: t('shipModal.totalQuantity'), value: order.total_quantity?.toFixed(2) || '0.00' },
      { key: t('shipModal.payableAmount'), value: formatPrice(order.payable_amount) },
    ]
  }, [order, t])

  if (!order) {
    return null
  }

  return (
    <Modal
      title={t('shipModal.title')}
      visible={visible}
      onOk={handleConfirm}
      onCancel={onCancel}
      okText={t('shipModal.confirm')}
      cancelText={t('shipModal.cancel')}
      confirmLoading={submitting || loading}
      maskClosable={false}
      className="ship-order-modal"
      width={520}
    >
      <div className="ship-order-modal-content">
        {/* Order summary */}
        <div className="order-summary-section">
          <Text strong className="section-title">
            {t('shipModal.orderInfo')}
          </Text>
          <Descriptions data={orderSummary} row className="order-summary" />
        </div>

        {/* Warehouse selection */}
        <div className="warehouse-selection-section">
          <Text strong className="section-title">
            {t('shipModal.warehouse')} <Text type="danger">*</Text>
          </Text>
          {warehousesLoading ? (
            <div className="loading-container">
              <Spin size="small" />
              <Text type="secondary">{t('shipModal.loadingWarehouses')}</Text>
            </div>
          ) : warehouses.length === 0 ? (
            <Empty
              className="empty-warehouses"
              description={t('shipModal.noWarehouses')}
              style={{ padding: 'var(--spacing-4)' }}
            />
          ) : (
            <Select
              value={selectedWarehouseId}
              onChange={(value) => setSelectedWarehouseId(value as string)}
              optionList={warehouseOptions}
              placeholder={t('shipModal.warehousePlaceholder')}
              style={{ width: '100%' }}
              filter
              showClear={false}
            />
          )}
        </div>

        {/* Warning message */}
        <div className="warning-section">
          <Text type="warning" size="small">
            {t('shipModal.warning')}
          </Text>
        </div>
      </div>
    </Modal>
  )
}
