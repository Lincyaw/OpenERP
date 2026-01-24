import { useState, useEffect, useCallback, useMemo } from 'react'
import { Modal, Select, Toast, Typography, Descriptions, Spin, Empty } from '@douyinfe/semi-ui'
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
      Toast.error('获取仓库列表失败')
    } finally {
      setWarehousesLoading(false)
    }
  }, [warehouseApi])

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
      Toast.warning('请选择发货仓库')
      return
    }

    setSubmitting(true)
    try {
      await onConfirm(selectedWarehouseId)
    } finally {
      setSubmitting(false)
    }
  }, [selectedWarehouseId, onConfirm])

  // Warehouse options for select
  const warehouseOptions = useMemo(
    () =>
      warehouses.map((w) => ({
        value: w.id,
        label: w.is_default ? `${w.name} (默认)` : w.name,
      })),
    [warehouses]
  )

  // Order summary data
  const orderSummary = useMemo(() => {
    if (!order) return []
    return [
      { key: '订单编号', value: order.order_number },
      { key: '客户名称', value: order.customer_name || '-' },
      { key: '商品数量', value: `${order.item_count || 0} 件` },
      { key: '总数量', value: order.total_quantity?.toFixed(2) || '0.00' },
      { key: '应付金额', value: formatPrice(order.payable_amount) },
    ]
  }, [order])

  if (!order) {
    return null
  }

  return (
    <Modal
      title="发货确认"
      visible={visible}
      onOk={handleConfirm}
      onCancel={onCancel}
      okText="确认发货"
      cancelText="取消"
      confirmLoading={submitting || loading}
      maskClosable={false}
      className="ship-order-modal"
      width={520}
    >
      <div className="ship-order-modal-content">
        {/* Order summary */}
        <div className="order-summary-section">
          <Text strong className="section-title">
            订单信息
          </Text>
          <Descriptions data={orderSummary} row className="order-summary" />
        </div>

        {/* Warehouse selection */}
        <div className="warehouse-selection-section">
          <Text strong className="section-title">
            发货仓库 <Text type="danger">*</Text>
          </Text>
          {warehousesLoading ? (
            <div className="loading-container">
              <Spin size="small" />
              <Text type="secondary">加载仓库列表...</Text>
            </div>
          ) : warehouses.length === 0 ? (
            <Empty
              className="empty-warehouses"
              description="暂无可用仓库"
              style={{ padding: 'var(--spacing-4)' }}
            />
          ) : (
            <Select
              value={selectedWarehouseId}
              onChange={(value) => setSelectedWarehouseId(value as string)}
              optionList={warehouseOptions}
              placeholder="请选择发货仓库"
              style={{ width: '100%' }}
              filter
              showClear={false}
            />
          )}
        </div>

        {/* Warning message */}
        <div className="warning-section">
          <Text type="warning" size="small">
            提示：发货后将扣减所选仓库的库存，请确认仓库选择正确。
          </Text>
        </div>
      </div>
    </Modal>
  )
}
