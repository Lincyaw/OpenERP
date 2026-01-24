import { useState, useEffect, useCallback, useMemo } from 'react'
import {
  Card,
  Typography,
  Table,
  Button,
  Toast,
  Spin,
  Empty,
  Descriptions,
  Tag,
  InputNumber,
  Input,
  DatePicker,
  Space,
  Select,
} from '@douyinfe/semi-ui'
import { IconArrowLeft, IconSave, IconRefresh } from '@douyinfe/semi-icons'
import { useNavigate, useParams } from 'react-router-dom'
import { Container } from '@/components/common/layout'
import { getPurchaseOrders } from '@/api/purchase-orders/purchase-orders'
import { getWarehouses } from '@/api/warehouses/warehouses'
import type {
  HandlerPurchaseOrderResponse,
  HandlerPurchaseOrderItemResponse,
  HandlerWarehouseResponse,
  HandlerReceiveItemInput,
} from '@/api/models'
import './PurchaseOrderReceive.css'

const { Title, Text } = Typography

// Status tag color mapping
const STATUS_TAG_COLORS: Record<string, 'blue' | 'cyan' | 'green' | 'orange' | 'grey' | 'red'> = {
  draft: 'blue',
  confirmed: 'cyan',
  partial_received: 'orange',
  completed: 'green',
  cancelled: 'grey',
}

// Status labels
const STATUS_LABELS: Record<string, string> = {
  draft: '草稿',
  confirmed: '已确认',
  partial_received: '部分收货',
  completed: '已完成',
  cancelled: '已取消',
}

/**
 * Format price for display
 */
function formatPrice(price?: number): string {
  if (price === undefined || price === null) return '-'
  return `¥${price.toFixed(2)}`
}

/**
 * Format date for display
 */
function formatDate(dateStr?: string): string {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  })
}

// Receivable item with receive form fields
interface ReceivableItem {
  id: string
  product_id: string
  product_name: string
  product_code: string
  unit: string
  ordered_quantity: number
  received_quantity: number
  remaining_quantity: number
  unit_cost: number
  // Form fields for this receiving session
  receive_quantity: number
  batch_number: string
  expiry_date: string | null
}

/**
 * Purchase Order Receive Page
 *
 * Features:
 * - Display order summary information
 * - Show receivable items with remaining quantities
 * - Input receiving quantity per item
 * - Batch number and expiry date for batch tracking
 * - Warehouse selection for receiving
 * - Partial receiving support
 */
export default function PurchaseOrderReceivePage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()

  const purchaseOrderApi = useMemo(() => getPurchaseOrders(), [])
  const warehouseApi = useMemo(() => getWarehouses(), [])

  // State
  const [order, setOrder] = useState<HandlerPurchaseOrderResponse | null>(null)
  const [receivableItems, setReceivableItems] = useState<ReceivableItem[]>([])
  const [warehouses, setWarehouses] = useState<HandlerWarehouseResponse[]>([])
  const [selectedWarehouseId, setSelectedWarehouseId] = useState<string | undefined>(undefined)

  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)

  // Fetch order details and receivable items
  const fetchOrderData = useCallback(async () => {
    if (!id) return

    setLoading(true)
    try {
      // Fetch order details
      const orderResponse = await purchaseOrderApi.getTradePurchaseOrdersId(id)
      if (orderResponse.success && orderResponse.data) {
        setOrder(orderResponse.data)

        // Set default warehouse if order has one
        if (orderResponse.data.warehouse_id) {
          setSelectedWarehouseId(orderResponse.data.warehouse_id)
        }
      } else {
        Toast.error('获取订单详情失败')
        return
      }

      // Fetch receivable items
      const itemsResponse = await purchaseOrderApi.getTradePurchaseOrdersIdReceivableItems(id)
      if (itemsResponse.success && itemsResponse.data) {
        const items: ReceivableItem[] = (
          itemsResponse.data as HandlerPurchaseOrderItemResponse[]
        ).map((item) => ({
          id: item.id || '',
          product_id: item.product_id || '',
          product_name: item.product_name || '',
          product_code: item.product_code || '',
          unit: item.unit || 'pcs',
          ordered_quantity: item.ordered_quantity || 0,
          received_quantity: item.received_quantity || 0,
          remaining_quantity: item.remaining_quantity || 0,
          unit_cost: item.unit_cost || 0,
          // Initialize form fields
          receive_quantity: item.remaining_quantity || 0, // Default to receive all remaining
          batch_number: '',
          expiry_date: null,
        }))
        setReceivableItems(items)
      }
    } catch {
      Toast.error('获取订单数据失败')
    } finally {
      setLoading(false)
    }
  }, [id, purchaseOrderApi])

  // Fetch warehouses
  const fetchWarehouses = useCallback(async () => {
    try {
      const response = await warehouseApi.getPartnerWarehouses({
        status: 'active',
        page_size: 100,
      })
      if (response.success && response.data) {
        setWarehouses(response.data)

        // Set default warehouse if not already set
        if (!selectedWarehouseId) {
          const defaultWarehouse = response.data.find((w) => w.is_default)
          if (defaultWarehouse?.id) {
            setSelectedWarehouseId(defaultWarehouse.id)
          } else if (response.data.length > 0 && response.data[0].id) {
            setSelectedWarehouseId(response.data[0].id)
          }
        }
      }
    } catch {
      Toast.error('获取仓库列表失败')
    }
  }, [warehouseApi, selectedWarehouseId])

  // Load data on mount
  useEffect(() => {
    fetchOrderData()
    fetchWarehouses()
  }, [fetchOrderData, fetchWarehouses])

  // Update receive quantity for an item
  const updateReceiveQuantity = useCallback((itemId: string, quantity: number) => {
    setReceivableItems((prev) =>
      prev.map((item) => (item.id === itemId ? { ...item, receive_quantity: quantity } : item))
    )
  }, [])

  // Update batch number for an item
  const updateBatchNumber = useCallback((itemId: string, batchNumber: string) => {
    setReceivableItems((prev) =>
      prev.map((item) => (item.id === itemId ? { ...item, batch_number: batchNumber } : item))
    )
  }, [])

  // Update expiry date for an item
  const updateExpiryDate = useCallback((itemId: string, expiryDate: string | null) => {
    setReceivableItems((prev) =>
      prev.map((item) => (item.id === itemId ? { ...item, expiry_date: expiryDate } : item))
    )
  }, [])

  // Set all items to receive max quantity
  const receiveAll = useCallback(() => {
    setReceivableItems((prev) =>
      prev.map((item) => ({
        ...item,
        receive_quantity: item.remaining_quantity,
      }))
    )
  }, [])

  // Clear all receive quantities
  const clearAll = useCallback(() => {
    setReceivableItems((prev) =>
      prev.map((item) => ({
        ...item,
        receive_quantity: 0,
      }))
    )
  }, [])

  // Submit receiving
  const handleSubmit = useCallback(async () => {
    if (!id) return

    // Validate warehouse selection
    if (!selectedWarehouseId) {
      Toast.warning('请选择收货仓库')
      return
    }

    // Filter items with receive_quantity > 0
    const itemsToReceive = receivableItems.filter((item) => item.receive_quantity > 0)

    if (itemsToReceive.length === 0) {
      Toast.warning('请至少输入一个商品的收货数量')
      return
    }

    // Validate quantities don't exceed remaining
    for (const item of itemsToReceive) {
      if (item.receive_quantity > item.remaining_quantity) {
        Toast.error(`${item.product_name} 收货数量不能超过待收数量`)
        return
      }
    }

    setSubmitting(true)
    try {
      const requestItems: HandlerReceiveItemInput[] = itemsToReceive.map((item) => ({
        product_id: item.product_id,
        quantity: item.receive_quantity,
        batch_number: item.batch_number || undefined,
        expiry_date: item.expiry_date || undefined,
      }))

      const response = await purchaseOrderApi.postTradePurchaseOrdersIdReceive(id, {
        warehouse_id: selectedWarehouseId,
        items: requestItems,
      })

      if (response.success) {
        const isFullyReceived = response.data?.is_fully_received
        if (isFullyReceived) {
          Toast.success('收货完成，订单已全部入库')
        } else {
          Toast.success('收货成功，部分商品已入库')
        }
        navigate('/trade/purchase')
      } else {
        Toast.error('收货失败')
      }
    } catch {
      Toast.error('收货失败')
    } finally {
      setSubmitting(false)
    }
  }, [id, selectedWarehouseId, receivableItems, purchaseOrderApi, navigate])

  // Order summary data for descriptions
  const orderSummary = useMemo(() => {
    if (!order) return []
    return [
      { key: '订单编号', value: order.order_number || '-' },
      { key: '供应商', value: order.supplier_name || '-' },
      {
        key: '状态',
        value: (
          <Tag color={STATUS_TAG_COLORS[order.status || '']}>
            {STATUS_LABELS[order.status || ''] || order.status}
          </Tag>
        ),
      },
      { key: '订单金额', value: formatPrice(order.total_amount) },
      { key: '应付金额', value: formatPrice(order.payable_amount) },
      { key: '确认时间', value: formatDate(order.confirmed_at) },
    ]
  }, [order])

  // Warehouse options
  const warehouseOptions = useMemo(
    () =>
      warehouses.map((w) => ({
        value: w.id,
        label: w.is_default ? `${w.name} (默认)` : w.name,
      })),
    [warehouses]
  )

  // Calculate total receiving stats
  const receivingStats = useMemo(() => {
    const totalItems = receivableItems.filter((item) => item.receive_quantity > 0).length
    const totalQuantity = receivableItems.reduce((sum, item) => sum + item.receive_quantity, 0)
    const totalAmount = receivableItems.reduce(
      (sum, item) => sum + item.receive_quantity * item.unit_cost,
      0
    )
    return { totalItems, totalQuantity, totalAmount }
  }, [receivableItems])

  // Table columns
  const columns = useMemo(
    () => [
      {
        title: '商品编码',
        dataIndex: 'product_code',
        width: 120,
      },
      {
        title: '商品名称',
        dataIndex: 'product_name',
        width: 180,
        ellipsis: true,
      },
      {
        title: '单位',
        dataIndex: 'unit',
        width: 60,
        align: 'center' as const,
      },
      {
        title: '订购数量',
        dataIndex: 'ordered_quantity',
        width: 100,
        align: 'right' as const,
        render: (value: number) => value.toFixed(2),
      },
      {
        title: '已收数量',
        dataIndex: 'received_quantity',
        width: 100,
        align: 'right' as const,
        render: (value: number) => value.toFixed(2),
      },
      {
        title: '待收数量',
        dataIndex: 'remaining_quantity',
        width: 100,
        align: 'right' as const,
        render: (value: number) => (
          <Text strong type="warning">
            {value.toFixed(2)}
          </Text>
        ),
      },
      {
        title: '单价',
        dataIndex: 'unit_cost',
        width: 100,
        align: 'right' as const,
        render: (value: number) => formatPrice(value),
      },
      {
        title: '本次收货数量',
        dataIndex: 'receive_quantity',
        width: 130,
        render: (_: unknown, record: ReceivableItem) => (
          <InputNumber
            value={record.receive_quantity}
            min={0}
            max={record.remaining_quantity}
            precision={2}
            step={1}
            style={{ width: '100%' }}
            onChange={(value) => updateReceiveQuantity(record.id, (value as number) || 0)}
          />
        ),
      },
      {
        title: '批次号',
        dataIndex: 'batch_number',
        width: 140,
        render: (_: unknown, record: ReceivableItem) => (
          <Input
            value={record.batch_number}
            placeholder="可选"
            onChange={(value) => updateBatchNumber(record.id, value)}
          />
        ),
      },
      {
        title: '有效期',
        dataIndex: 'expiry_date',
        width: 160,
        render: (_: unknown, record: ReceivableItem) => (
          <DatePicker
            value={record.expiry_date || undefined}
            placeholder="可选"
            style={{ width: '100%' }}
            onChange={(date) => {
              let dateStr: string | null = null
              if (date) {
                if (typeof date === 'string') {
                  dateStr = date
                } else if (date instanceof Date) {
                  dateStr = date.toISOString()
                } else if (Array.isArray(date) && date[0]) {
                  const firstDate = date[0]
                  dateStr = typeof firstDate === 'string' ? firstDate : firstDate.toISOString()
                }
              }
              updateExpiryDate(record.id, dateStr)
            }}
          />
        ),
      },
    ],
    [updateReceiveQuantity, updateBatchNumber, updateExpiryDate]
  )

  // Check if order is receivable
  const canReceive = order?.status === 'confirmed' || order?.status === 'partial_received'

  if (loading) {
    return (
      <Container size="full" className="purchase-order-receive-page">
        <Card className="loading-card">
          <Spin size="large" tip="加载中..." />
        </Card>
      </Container>
    )
  }

  if (!order) {
    return (
      <Container size="full" className="purchase-order-receive-page">
        <Card className="empty-card">
          <Empty description="订单不存在" />
          <Button onClick={() => navigate('/trade/purchase')} style={{ marginTop: 16 }}>
            返回列表
          </Button>
        </Card>
      </Container>
    )
  }

  if (!canReceive) {
    return (
      <Container size="full" className="purchase-order-receive-page">
        <Card className="empty-card">
          <Empty
            description={`订单状态为"${STATUS_LABELS[order.status || ''] || order.status}"，无法收货`}
          />
          <Button onClick={() => navigate('/trade/purchase')} style={{ marginTop: 16 }}>
            返回列表
          </Button>
        </Card>
      </Container>
    )
  }

  return (
    <Container size="full" className="purchase-order-receive-page">
      {/* Header */}
      <div className="page-header">
        <div className="header-left">
          <Button
            icon={<IconArrowLeft />}
            theme="borderless"
            onClick={() => navigate('/trade/purchase')}
          />
          <Title heading={4} style={{ margin: 0 }}>
            采购收货
          </Title>
        </div>
        <div className="header-right">
          <Button icon={<IconRefresh />} onClick={fetchOrderData}>
            刷新
          </Button>
        </div>
      </div>

      {/* Order Summary */}
      <Card className="order-summary-card">
        <Title heading={5} className="card-title">
          订单信息
        </Title>
        <Descriptions data={orderSummary} row />
      </Card>

      {/* Warehouse Selection */}
      <Card className="warehouse-selection-card">
        <div className="warehouse-selection">
          <Text strong>
            收货仓库 <Text type="danger">*</Text>
          </Text>
          <Select
            value={selectedWarehouseId}
            onChange={(value) => setSelectedWarehouseId(value as string)}
            optionList={warehouseOptions}
            placeholder="请选择收货仓库"
            style={{ width: 240 }}
            filter
            showClear={false}
          />
        </div>
      </Card>

      {/* Receivable Items Table */}
      <Card className="items-card">
        <div className="items-header">
          <Title heading={5} className="card-title" style={{ margin: 0 }}>
            收货明细
          </Title>
          <Space>
            <Button size="small" onClick={receiveAll}>
              全部收货
            </Button>
            <Button size="small" onClick={clearAll}>
              清空数量
            </Button>
          </Space>
        </div>

        <Table
          dataSource={receivableItems}
          columns={columns}
          rowKey="id"
          pagination={false}
          scroll={{ x: 1400 }}
          empty={<Empty description="暂无待收货商品" />}
        />

        {/* Receiving Summary */}
        {receivingStats.totalItems > 0 && (
          <div className="receiving-summary">
            <Text type="secondary">
              本次收货: {receivingStats.totalItems} 种商品, 共{' '}
              {receivingStats.totalQuantity.toFixed(2)} 件, 金额{' '}
              {formatPrice(receivingStats.totalAmount)}
            </Text>
          </div>
        )}
      </Card>

      {/* Actions */}
      <Card className="actions-card">
        <div className="actions-bar">
          <Button onClick={() => navigate('/trade/purchase')}>取消</Button>
          <Button
            type="primary"
            theme="solid"
            icon={<IconSave />}
            loading={submitting}
            disabled={receivingStats.totalItems === 0}
            onClick={handleSubmit}
          >
            确认收货
          </Button>
        </div>
      </Card>
    </Container>
  )
}
