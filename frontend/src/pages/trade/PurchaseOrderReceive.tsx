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
} from '@douyinfe/semi-ui-19'
import { IconArrowLeft, IconSave, IconRefresh } from '@douyinfe/semi-icons'
import { useNavigate, useParams } from 'react-router-dom'
import { Container } from '@/components/common/layout'
import { getPurchaseOrders } from '@/api/purchase-orders/purchase-orders'
import { listWarehouses } from '@/api/warehouses/warehouses'
import type {
  HandlerPurchaseOrderResponse,
  HandlerPurchaseOrderItemResponse,
  HandlerWarehouseResponse,
  HandlerReceiveItemInput,
} from '@/api/models'
import { useI18n } from '@/hooks/useI18n'
import { useFormatters } from '@/hooks/useFormatters'
import { safeToFixed } from '@/utils'
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
  const { t } = useI18n({ ns: 'trade' })
  const { formatCurrency, formatDate } = useFormatters()

  const purchaseOrderApi = useMemo(() => getPurchaseOrders(), [])

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
      const orderResponse = await purchaseOrderApi.getPurchaseOrderById(id)
      if (orderResponse.success && orderResponse.data) {
        setOrder(orderResponse.data)

        // Set default warehouse if order has one
        if (orderResponse.data.warehouse_id) {
          setSelectedWarehouseId(orderResponse.data.warehouse_id)
        }
      } else {
        Toast.error(t('receive.messages.fetchDetailError'))
        return
      }

      // Fetch receivable items
      const itemsResponse = await purchaseOrderApi.getPurchaseOrderReceivableItems(id)
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
      Toast.error(t('receive.messages.fetchError'))
    } finally {
      setLoading(false)
    }
  }, [id, purchaseOrderApi])

  // Fetch warehouses
  const fetchWarehouses = useCallback(async () => {
    try {
      const response = await listWarehouses({
        status: 'enabled',
        page_size: 100,
      })
      if (response.status === 200 && response.data.success && response.data.data) {
        setWarehouses(response.data.data as HandlerWarehouseResponse[])

        // Set default warehouse if not already set
        if (!selectedWarehouseId) {
          const defaultWarehouse = response.data.data.find(
            (w: HandlerWarehouseResponse) => w.is_default
          )
          if (defaultWarehouse?.id) {
            setSelectedWarehouseId(defaultWarehouse.id)
          } else if (response.data.data.length > 0 && response.data.data[0].id) {
            setSelectedWarehouseId(response.data.data[0].id)
          }
        }
      }
    } catch {
      Toast.error(t('receive.messages.fetchWarehousesError'))
    }
  }, [selectedWarehouseId, t])

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
      Toast.warning(t('receive.validation.warehouseRequired'))
      return
    }

    // Filter items with receive_quantity > 0
    const itemsToReceive = receivableItems.filter((item) => item.receive_quantity > 0)

    if (itemsToReceive.length === 0) {
      Toast.warning(t('receive.validation.itemsRequired'))
      return
    }

    // Validate quantities don't exceed remaining
    for (const item of itemsToReceive) {
      if (item.receive_quantity > item.remaining_quantity) {
        Toast.error(t('receive.validation.exceedRemaining', { productName: item.product_name }))
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

      const response = await purchaseOrderApi.receivePurchaseOrder(id, {
        warehouse_id: selectedWarehouseId,
        items: requestItems,
      })

      if (response.success) {
        const isFullyReceived = response.data?.is_fully_received
        if (isFullyReceived) {
          Toast.success(t('receive.messages.receiveFullSuccess'))
        } else {
          Toast.success(t('receive.messages.receivePartialSuccess'))
        }
        navigate('/trade/purchase')
      } else {
        Toast.error(t('receive.messages.receiveError'))
      }
    } catch {
      Toast.error(t('receive.messages.receiveError'))
    } finally {
      setSubmitting(false)
    }
  }, [id, selectedWarehouseId, receivableItems, purchaseOrderApi, navigate, t])

  // Order summary data for descriptions
  const orderSummary = useMemo(() => {
    if (!order) return []
    const statusKey = order.status === 'partial_received' ? 'partialReceived' : order.status
    return [
      { key: t('receive.orderNumber'), value: order.order_number || '-' },
      { key: t('receive.supplier'), value: order.supplier_name || '-' },
      {
        key: t('receive.status'),
        value: (
          <Tag color={STATUS_TAG_COLORS[order.status || '']}>
            {t(`purchaseOrder.status.${statusKey || 'draft'}`)}
          </Tag>
        ),
      },
      {
        key: t('receive.orderAmount'),
        value: order.total_amount !== undefined ? formatCurrency(order.total_amount) : '-',
      },
      {
        key: t('receive.payableAmount'),
        value: order.payable_amount !== undefined ? formatCurrency(order.payable_amount) : '-',
      },
      {
        key: t('receive.confirmedAt'),
        value: order.confirmed_at ? formatDate(order.confirmed_at) : '-',
      },
    ]
  }, [order, t, formatCurrency, formatDate])

  // Warehouse options
  const warehouseOptions = useMemo(
    () =>
      warehouses.map((w) => ({
        value: w.id,
        label: w.is_default ? `${w.name} (${t('common.defaultWarehouse')})` : w.name,
      })),
    [warehouses, t]
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
        title: t('receive.columns.productCode'),
        dataIndex: 'product_code',
        width: 120,
      },
      {
        title: t('receive.columns.productName'),
        dataIndex: 'product_name',
        width: 180,
        ellipsis: true,
      },
      {
        title: t('receive.columns.unit'),
        dataIndex: 'unit',
        width: 60,
        align: 'center' as const,
      },
      {
        title: t('receive.columns.orderedQuantity'),
        dataIndex: 'ordered_quantity',
        width: 100,
        align: 'right' as const,
        render: (value: number) => safeToFixed(value),
      },
      {
        title: t('receive.columns.receivedQuantity'),
        dataIndex: 'received_quantity',
        width: 100,
        align: 'right' as const,
        render: (value: number) => safeToFixed(value),
      },
      {
        title: t('receive.columns.remainingQuantity'),
        dataIndex: 'remaining_quantity',
        width: 100,
        align: 'right' as const,
        render: (value: number) => (
          <Text strong type="warning">
            {safeToFixed(value)}
          </Text>
        ),
      },
      {
        title: t('receive.columns.unitCost'),
        dataIndex: 'unit_cost',
        width: 100,
        align: 'right' as const,
        render: (value: number) => formatCurrency(value),
      },
      {
        title: t('receive.columns.receiveQuantity'),
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
        title: t('receive.columns.batchNumber'),
        dataIndex: 'batch_number',
        width: 140,
        render: (_: unknown, record: ReceivableItem) => (
          <Input
            value={record.batch_number}
            placeholder={t('receive.columns.batchPlaceholder')}
            onChange={(value) => updateBatchNumber(record.id, value)}
          />
        ),
      },
      {
        title: t('receive.columns.expiryDate'),
        dataIndex: 'expiry_date',
        width: 160,
        render: (_: unknown, record: ReceivableItem) => (
          <DatePicker
            value={record.expiry_date || undefined}
            placeholder={t('receive.columns.expiryPlaceholder')}
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
    [updateReceiveQuantity, updateBatchNumber, updateExpiryDate, t, formatCurrency]
  )

  // Check if order is receivable
  const canReceive = order?.status === 'confirmed' || order?.status === 'partial_received'

  if (loading) {
    return (
      <Container size="full" className="purchase-order-receive-page">
        <Card className="loading-card">
          <Spin size="large" tip={t('receive.loading')} />
        </Card>
      </Container>
    )
  }

  if (!order) {
    return (
      <Container size="full" className="purchase-order-receive-page">
        <Card className="empty-card">
          <Empty description={t('receive.messages.notExist')} />
          <Button onClick={() => navigate('/trade/purchase')} style={{ marginTop: 16 }}>
            {t('orderDetail.back')}
          </Button>
        </Card>
      </Container>
    )
  }

  if (!canReceive) {
    const statusKey = order.status === 'partial_received' ? 'partialReceived' : order.status
    return (
      <Container size="full" className="purchase-order-receive-page">
        <Card className="empty-card">
          <Empty
            description={t('receive.cannotReceive', {
              status: t(`purchaseOrder.status.${statusKey || 'draft'}`),
            })}
          />
          <Button onClick={() => navigate('/trade/purchase')} style={{ marginTop: 16 }}>
            {t('orderDetail.back')}
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
            {t('receive.title')}
          </Title>
        </div>
        <div className="header-right">
          <Button icon={<IconRefresh />} onClick={fetchOrderData}>
            {t('receive.refresh')}
          </Button>
        </div>
      </div>

      {/* Order Summary */}
      <Card className="order-summary-card">
        <Title heading={5} className="card-title">
          {t('receive.orderInfo')}
        </Title>
        <Descriptions data={orderSummary} row />
      </Card>

      {/* Warehouse Selection */}
      <Card className="warehouse-selection-card">
        <div className="warehouse-selection">
          <Text strong>
            {t('receive.warehouse')} <Text type="danger">*</Text>
          </Text>
          <Select
            value={selectedWarehouseId}
            onChange={(value) => setSelectedWarehouseId(value as string)}
            optionList={warehouseOptions}
            placeholder={t('receive.warehousePlaceholder')}
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
            {t('receive.detail.title')}
          </Title>
          <Space>
            <Button size="small" onClick={receiveAll}>
              {t('receive.detail.receiveAll')}
            </Button>
            <Button size="small" onClick={clearAll}>
              {t('receive.detail.clearAll')}
            </Button>
          </Space>
        </div>

        <Table
          dataSource={receivableItems}
          columns={columns}
          rowKey="id"
          pagination={false}
          scroll={{ x: 1400 }}
          empty={<Empty description={t('receive.emptyItems')} />}
        />

        {/* Receiving Summary */}
        {receivingStats.totalItems > 0 && (
          <div className="receiving-summary">
            <Text type="secondary">
              {t('receive.summary', {
                items: receivingStats.totalItems,
                quantity: safeToFixed(receivingStats.totalQuantity),
                amount: formatCurrency(receivingStats.totalAmount),
              })}
            </Text>
          </div>
        )}
      </Card>

      {/* Actions */}
      <Card className="actions-card">
        <div className="actions-bar">
          <Button onClick={() => navigate('/trade/purchase')}>{t('receive.actions.cancel')}</Button>
          <Button
            type="primary"
            theme="solid"
            icon={<IconSave />}
            loading={submitting}
            disabled={receivingStats.totalItems === 0}
            onClick={handleSubmit}
          >
            {t('receive.actions.confirm')}
          </Button>
        </div>
      </Card>
    </Container>
  )
}
