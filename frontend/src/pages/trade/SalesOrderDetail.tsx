import { useState, useEffect, useCallback, useMemo } from 'react'
import {
  Card,
  Typography,
  Descriptions,
  Table,
  Tag,
  Toast,
  Button,
  Space,
  Spin,
  Modal,
  Empty,
  Timeline,
} from '@douyinfe/semi-ui-19'
import { IconArrowLeft, IconEdit, IconTick, IconClose, IconSend } from '@douyinfe/semi-icons'
import { useParams, useNavigate } from 'react-router-dom'
import { Container } from '@/components/common/layout'
import { getSalesOrders } from '@/api/sales-orders/sales-orders'
import type { HandlerSalesOrderResponse, HandlerSalesOrderItemResponse } from '@/api/models'
import { ShipOrderModal } from './components'
import { useI18n } from '@/hooks/useI18n'
import { useFormatters } from '@/hooks/useFormatters'
import { safeToFixed, toNumber } from '@/utils'
import './SalesOrderDetail.css'

const { Title, Text } = Typography

// Status tag color mapping
const STATUS_TAG_COLORS: Record<string, 'blue' | 'cyan' | 'green' | 'grey' | 'red'> = {
  draft: 'blue',
  confirmed: 'cyan',
  shipped: 'green',
  completed: 'grey',
  cancelled: 'red',
}

/**
 * Sales Order Detail Page
 *
 * Features:
 * - Display complete order information
 * - Display order line items
 * - Display status change timeline
 * - Status action buttons (confirm, ship, complete, cancel)
 */
export default function SalesOrderDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { t } = useI18n({ ns: 'trade' })
  const { formatCurrency, formatDateTime } = useFormatters()
  const salesOrderApi = useMemo(() => getSalesOrders(), [])

  const [order, setOrder] = useState<HandlerSalesOrderResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState(false)
  const [shipModalVisible, setShipModalVisible] = useState(false)

  // Fetch order details
  const fetchOrder = useCallback(async () => {
    if (!id) return

    setLoading(true)
    try {
      const response = await salesOrderApi.getTradeSalesOrdersId(id)
      if (response.success && response.data) {
        setOrder(response.data)
      } else {
        Toast.error(t('salesOrder.messages.notExist'))
        navigate('/trade/sales')
      }
    } catch {
      Toast.error(t('salesOrder.messages.fetchDetailError'))
      navigate('/trade/sales')
    } finally {
      setLoading(false)
    }
  }, [id, salesOrderApi, navigate, t])

  useEffect(() => {
    fetchOrder()
  }, [fetchOrder])

  // Handle confirm order
  const handleConfirm = useCallback(async () => {
    if (!order?.id) return
    Modal.confirm({
      title: t('salesOrder.modal.confirmTitle'),
      content: t('salesOrder.modal.confirmContent', { orderNumber: order.order_number }),
      okText: t('salesOrder.modal.confirmOk'),
      cancelText: t('salesOrder.modal.cancelBtn'),
      onOk: async () => {
        setActionLoading(true)
        try {
          await salesOrderApi.postTradeSalesOrdersIdConfirm(order.id!, {})
          Toast.success(t('orderDetail.messages.confirmSuccess'))
          fetchOrder()
        } catch {
          Toast.error(t('orderDetail.messages.confirmError'))
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [order, salesOrderApi, fetchOrder, t])

  // Handle ship order - open modal
  const handleShip = useCallback(() => {
    setShipModalVisible(true)
  }, [])

  // Handle ship confirm from modal
  const handleShipConfirm = useCallback(
    async (warehouseId: string) => {
      if (!order?.id) return

      try {
        await salesOrderApi.postTradeSalesOrdersIdShip(order.id, {
          warehouse_id: warehouseId,
        })
        Toast.success(t('orderDetail.messages.shipSuccess'))
        setShipModalVisible(false)
        fetchOrder()
      } catch {
        Toast.error(t('orderDetail.messages.shipError'))
        throw new Error(t('orderDetail.messages.shipError')) // Re-throw to keep modal open
      }
    },
    [order, salesOrderApi, fetchOrder, t]
  )

  // Handle complete order
  const handleComplete = useCallback(async () => {
    if (!order?.id) return
    setActionLoading(true)
    try {
      await salesOrderApi.postTradeSalesOrdersIdComplete(order.id)
      Toast.success(t('orderDetail.messages.completeSuccess'))
      fetchOrder()
    } catch {
      Toast.error(t('orderDetail.messages.completeError'))
    } finally {
      setActionLoading(false)
    }
  }, [order, salesOrderApi, fetchOrder, t])

  // Handle cancel order
  const handleCancel = useCallback(async () => {
    if (!order?.id) return
    Modal.confirm({
      title: t('salesOrder.modal.cancelTitle'),
      content: t('salesOrder.modal.cancelContent', { orderNumber: order.order_number }),
      okText: t('salesOrder.modal.cancelOk'),
      cancelText: t('salesOrder.modal.backBtn'),
      okButtonProps: { type: 'danger' },
      onOk: async () => {
        setActionLoading(true)
        try {
          await salesOrderApi.postTradeSalesOrdersIdCancel(order.id!, {
            reason: t('common.userCancel'),
          })
          Toast.success(t('orderDetail.messages.cancelSuccess'))
          fetchOrder()
        } catch {
          Toast.error(t('orderDetail.messages.cancelError'))
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [order, salesOrderApi, fetchOrder, t])

  // Handle edit order
  const handleEdit = useCallback(() => {
    if (order?.id) {
      navigate(`/trade/sales/${order.id}/edit`)
    }
  }, [order, navigate])

  // Order items table columns
  const itemColumns = useMemo(
    () => [
      {
        title: t('orderDetail.items.columns.index'),
        dataIndex: 'index',
        width: 60,
        render: (_: unknown, __: unknown, index: number) => index + 1,
      },
      {
        title: t('orderDetail.items.columns.productCode'),
        dataIndex: 'product_code',
        width: 120,
        render: (code: string) => <Text className="product-code">{code || '-'}</Text>,
      },
      {
        title: t('orderDetail.items.columns.productName'),
        dataIndex: 'product_name',
        width: 200,
        ellipsis: true,
      },
      {
        title: t('orderDetail.items.columns.unit'),
        dataIndex: 'unit',
        width: 80,
        align: 'center' as const,
        render: (unit: string) => unit || '-',
      },
      {
        title: t('orderDetail.items.columns.quantity'),
        dataIndex: 'quantity',
        width: 100,
        align: 'right' as const,
        render: (qty: number) => safeToFixed(qty, 2, '-'),
      },
      {
        title: t('orderDetail.items.columns.unitPrice'),
        dataIndex: 'unit_price',
        width: 120,
        align: 'right' as const,
        render: (price: number) => formatCurrency(price),
      },
      {
        title: t('orderDetail.items.columns.amount'),
        dataIndex: 'amount',
        width: 120,
        align: 'right' as const,
        render: (amount: number) => <Text className="item-amount">{formatCurrency(amount)}</Text>,
      },
      {
        title: t('orderDetail.items.columns.remark'),
        dataIndex: 'remark',
        ellipsis: true,
        render: (remark: string) => remark || '-',
      },
    ],
    [t, formatCurrency]
  )

  // Build timeline items based on order status
  const timelineItems = useMemo(() => {
    if (!order) return []

    const items = []

    // Created
    if (order.created_at) {
      items.push({
        time: formatDateTime(order.created_at),
        content: t('orderDetail.timeline.created'),
        type: 'default' as const,
      })
    }

    // Confirmed
    if (order.confirmed_at) {
      items.push({
        time: formatDateTime(order.confirmed_at),
        content: t('orderDetail.timeline.confirmed'),
        type: 'success' as const,
      })
    }

    // Shipped
    if (order.shipped_at) {
      items.push({
        time: formatDateTime(order.shipped_at),
        content: t('orderDetail.timeline.shipped'),
        type: 'success' as const,
      })
    }

    // Completed
    if (order.completed_at) {
      items.push({
        time: formatDateTime(order.completed_at),
        content: t('orderDetail.timeline.completed'),
        type: 'success' as const,
      })
    }

    // Cancelled
    if (order.cancelled_at) {
      items.push({
        time: formatDateTime(order.cancelled_at),
        content: `${t('orderDetail.timeline.cancelled')}${order.cancel_reason ? `: ${order.cancel_reason}` : ''}`,
        type: 'error' as const,
      })
    }

    return items
  }, [order, t, formatDateTime])

  // Render order basic info
  const renderBasicInfo = () => {
    if (!order) return null

    const data = [
      { key: t('orderDetail.basicInfo.orderNumber'), value: order.order_number },
      { key: t('orderDetail.basicInfo.customerName'), value: order.customer_name || '-' },
      {
        key: t('orderDetail.basicInfo.orderStatus'),
        value: (
          <Tag color={STATUS_TAG_COLORS[order.status || 'draft']}>
            {t(`salesOrder.status.${order.status || 'draft'}`)}
          </Tag>
        ),
      },
      {
        key: t('orderDetail.basicInfo.itemCount'),
        value: `${order.item_count || 0} ${t('salesOrder.unit')}`,
      },
      {
        key: t('orderDetail.basicInfo.totalQuantity'),
        value: safeToFixed(order.total_quantity, 2, '0.00'),
      },
      {
        key: t('orderDetail.basicInfo.createdAt'),
        value: order.created_at ? formatDateTime(order.created_at) : '-',
      },
      {
        key: t('orderDetail.basicInfo.updatedAt'),
        value: order.updated_at ? formatDateTime(order.updated_at) : '-',
      },
      { key: t('orderDetail.basicInfo.remark'), value: order.remark || '-' },
    ]

    return <Descriptions data={data} row className="order-basic-info" />
  }

  // Render amount summary
  const renderAmountSummary = () => {
    if (!order) return null

    const discountPercent =
      order.discount_amount && order.total_amount
        ? safeToFixed((toNumber(order.discount_amount) / (toNumber(order.total_amount) + toNumber(order.discount_amount))) * 100, 1, '0')
        : '0'

    return (
      <div className="amount-summary">
        <div className="amount-row">
          <Text type="secondary">{t('orderDetail.amount.productAmount')}</Text>
          <Text>{formatCurrency((order.total_amount || 0) + (order.discount_amount || 0))}</Text>
        </div>
        <div className="amount-row">
          <Text type="secondary">
            {t('orderDetail.amount.discountAmount')} ({discountPercent}%)
          </Text>
          <Text className="discount-amount">-{formatCurrency(order.discount_amount || 0)}</Text>
        </div>
        <div className="amount-row total-row">
          <Text strong>{t('orderDetail.amount.payableAmount')}</Text>
          <Text className="payable-amount" strong>
            {formatCurrency(order.payable_amount || 0)}
          </Text>
        </div>
      </div>
    )
  }

  // Render action buttons based on status
  const renderActions = () => {
    if (!order) return null

    const status = order.status || 'draft'

    return (
      <Space>
        {status === 'draft' && (
          <>
            <Button icon={<IconEdit />} onClick={handleEdit} disabled={actionLoading}>
              {t('orderDetail.actions.edit')}
            </Button>
            <Button
              type="primary"
              icon={<IconTick />}
              onClick={handleConfirm}
              loading={actionLoading}
            >
              {t('orderDetail.actions.confirmOrder')}
            </Button>
            <Button
              type="danger"
              icon={<IconClose />}
              onClick={handleCancel}
              loading={actionLoading}
            >
              {t('orderDetail.actions.cancel')}
            </Button>
          </>
        )}
        {status === 'confirmed' && (
          <>
            <Button type="primary" icon={<IconSend />} onClick={handleShip} loading={actionLoading}>
              {t('orderDetail.actions.ship')}
            </Button>
            <Button
              type="warning"
              icon={<IconClose />}
              onClick={handleCancel}
              loading={actionLoading}
            >
              {t('orderDetail.actions.cancel')}
            </Button>
          </>
        )}
        {status === 'shipped' && (
          <Button
            type="primary"
            icon={<IconTick />}
            onClick={handleComplete}
            loading={actionLoading}
          >
            {t('orderDetail.actions.complete')}
          </Button>
        )}
      </Space>
    )
  }

  if (loading) {
    return (
      <Container size="lg" className="sales-order-detail-page">
        <div className="loading-container">
          <Spin size="large" />
        </div>
      </Container>
    )
  }

  if (!order) {
    return (
      <Container size="lg" className="sales-order-detail-page">
        <Empty title={t('orderDetail.notExist')} description={t('orderDetail.notExistDesc')} />
      </Container>
    )
  }

  return (
    <Container size="lg" className="sales-order-detail-page">
      {/* Header */}
      <div className="page-header">
        <div className="header-left">
          <Button
            icon={<IconArrowLeft />}
            theme="borderless"
            onClick={() => navigate('/trade/sales')}
          >
            {t('orderDetail.back')}
          </Button>
          <Title heading={4} className="page-title">
            {t('orderDetail.title')}
          </Title>
          <Tag color={STATUS_TAG_COLORS[order.status || 'draft']} size="large">
            {t(`salesOrder.status.${order.status || 'draft'}`)}
          </Tag>
        </div>
        <div className="header-right">{renderActions()}</div>
      </div>

      {/* Order Info Card */}
      <Card className="info-card" title={t('orderDetail.basicInfo.title')}>
        {renderBasicInfo()}
      </Card>

      {/* Order Items Card */}
      <Card className="items-card" title={t('orderDetail.items.title')}>
        <Table
          columns={itemColumns}
          dataSource={
            (order.items || []) as (HandlerSalesOrderItemResponse & Record<string, unknown>)[]
          }
          rowKey="id"
          pagination={false}
          size="small"
          empty={<Empty description={t('orderDetail.items.empty')} />}
        />
        {renderAmountSummary()}
      </Card>

      {/* Timeline Card */}
      <Card className="timeline-card" title={t('orderDetail.timeline.title')}>
        {timelineItems.length > 0 ? (
          <Timeline mode="left" className="status-timeline">
            {timelineItems.map((item, index) => (
              <Timeline.Item
                key={index}
                time={item.time}
                type={item.type as 'default' | 'success' | 'warning' | 'error'}
              >
                {item.content}
              </Timeline.Item>
            ))}
          </Timeline>
        ) : (
          <Empty description={t('orderDetail.timeline.empty')} />
        )}
      </Card>

      {/* Ship Order Modal */}
      <ShipOrderModal
        visible={shipModalVisible}
        order={
          order
            ? {
                id: order.id!,
                order_number: order.order_number!,
                customer_name: order.customer_name,
                warehouse_id: order.warehouse_id,
                item_count: order.item_count,
                total_quantity: order.total_quantity,
                payable_amount: order.payable_amount,
              }
            : null
        }
        onConfirm={handleShipConfirm}
        onCancel={() => setShipModalVisible(false)}
      />
    </Container>
  )
}
