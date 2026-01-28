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
  Progress,
} from '@douyinfe/semi-ui-19'
import { IconArrowLeft, IconEdit, IconTick, IconClose, IconBox } from '@douyinfe/semi-icons'
import { useParams, useNavigate } from 'react-router-dom'
import { Container } from '@/components/common/layout'
import { PrintButton } from '@/components/printing'
import { getPurchaseOrders } from '@/api/purchase-orders/purchase-orders'
import type { HandlerPurchaseOrderResponse, HandlerPurchaseOrderItemResponse } from '@/api/models'
import { useI18n } from '@/hooks/useI18n'
import { useFormatters } from '@/hooks/useFormatters'
import './PurchaseOrderDetail.css'

const { Title, Text } = Typography

// Status tag color mapping
const STATUS_TAG_COLORS: Record<string, 'blue' | 'cyan' | 'green' | 'orange' | 'grey' | 'red'> = {
  draft: 'blue',
  confirmed: 'cyan',
  partial_received: 'orange',
  completed: 'green',
  cancelled: 'red',
}

/**
 * Purchase Order Detail Page
 *
 * Features:
 * - Display complete order information
 * - Display order line items with receive progress
 * - Display status change timeline
 * - Status action buttons (confirm, receive, cancel)
 */
export default function PurchaseOrderDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { t } = useI18n({ ns: 'trade' })
  const { formatCurrency, formatDateTime } = useFormatters()
  const purchaseOrderApi = useMemo(() => getPurchaseOrders(), [])

  const [order, setOrder] = useState<HandlerPurchaseOrderResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState(false)

  // Fetch order details
  const fetchOrder = useCallback(async () => {
    if (!id) return

    setLoading(true)
    try {
      const response = await purchaseOrderApi.getPurchaseOrderById(id)
      if (response.success && response.data) {
        setOrder(response.data)
      } else {
        Toast.error(t('purchaseOrderDetail.messages.notExist'))
        navigate('/trade/purchase')
      }
    } catch {
      Toast.error(t('purchaseOrderDetail.messages.fetchError'))
      navigate('/trade/purchase')
    } finally {
      setLoading(false)
    }
  }, [id, purchaseOrderApi, navigate, t])

  useEffect(() => {
    fetchOrder()
  }, [fetchOrder])

  // Handle confirm order
  const handleConfirm = useCallback(async () => {
    if (!order?.id) return
    Modal.confirm({
      title: t('purchaseOrder.modal.confirmTitle'),
      content: t('purchaseOrder.modal.confirmContent', { orderNumber: order.order_number }),
      okText: t('common:actions.confirm'),
      cancelText: t('common:actions.cancel'),
      onOk: async () => {
        setActionLoading(true)
        try {
          await purchaseOrderApi.confirmPurchaseOrder(order.id!, {})
          Toast.success(t('purchaseOrderDetail.messages.confirmSuccess'))
          fetchOrder()
        } catch {
          Toast.error(t('purchaseOrderDetail.messages.confirmError'))
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [order, purchaseOrderApi, fetchOrder, t])

  // Handle receive order - navigate to receive page
  const handleReceive = useCallback(() => {
    if (order?.id) {
      navigate(`/trade/purchase/${order.id}/receive`)
    }
  }, [order, navigate])

  // Handle cancel order
  const handleCancel = useCallback(async () => {
    if (!order?.id) return
    Modal.confirm({
      title: t('purchaseOrder.modal.cancelTitle'),
      content: t('purchaseOrder.modal.cancelContent', { orderNumber: order.order_number }),
      okText: t('common:actions.confirm'),
      cancelText: t('common:actions.cancel'),
      okButtonProps: { type: 'danger' },
      onOk: async () => {
        setActionLoading(true)
        try {
          await purchaseOrderApi.cancelPurchaseOrder(order.id!, {
            reason: t('common:userCancel', { defaultValue: '用户取消' }),
          })
          Toast.success(t('purchaseOrderDetail.messages.cancelSuccess'))
          fetchOrder()
        } catch {
          Toast.error(t('purchaseOrderDetail.messages.cancelError'))
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [order, purchaseOrderApi, fetchOrder, t])

  // Handle edit order
  const handleEdit = useCallback(() => {
    if (order?.id) {
      navigate(`/trade/purchase/${order.id}/edit`)
    }
  }, [order, navigate])

  // Format quantity safely
  const formatQuantity = useCallback((qty?: number): string => {
    if (qty === undefined || qty === null) return '-'
    const num = typeof qty === 'string' ? parseFloat(qty) : qty
    if (isNaN(num)) return '-'
    return num.toFixed(2)
  }, [])

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
        title: t('purchaseOrderDetail.items.orderedQuantity'),
        dataIndex: 'ordered_quantity',
        width: 100,
        align: 'right' as const,
        render: (qty: number) => formatQuantity(qty),
      },
      {
        title: t('purchaseOrderDetail.items.receivedQuantity'),
        dataIndex: 'received_quantity',
        width: 100,
        align: 'right' as const,
        render: (qty: number) => formatQuantity(qty),
      },
      {
        title: t('purchaseOrderDetail.items.remainingQuantity'),
        dataIndex: 'remaining_quantity',
        width: 100,
        align: 'right' as const,
        render: (qty: number) => (
          <Text type={qty && qty > 0 ? 'warning' : 'secondary'}>{formatQuantity(qty)}</Text>
        ),
      },
      {
        title: t('purchaseOrderDetail.items.unitCost'),
        dataIndex: 'unit_cost',
        width: 120,
        align: 'right' as const,
        render: (cost: number) => formatCurrency(cost),
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
    [t, formatCurrency, formatQuantity]
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
      { key: t('orderDetail.basicInfo.supplierName'), value: order.supplier_name || '-' },
      {
        key: t('orderDetail.basicInfo.orderStatus'),
        value: (
          <Tag color={STATUS_TAG_COLORS[order.status || 'draft']}>
            {t(
              `purchaseOrder.status.${order.status === 'partial_received' ? 'partialReceived' : order.status || 'draft'}`
            )}
          </Tag>
        ),
      },
      {
        key: t('orderDetail.basicInfo.itemCount'),
        value: `${order.item_count || 0} ${t('purchaseOrder.unit', { defaultValue: '种' })}`,
      },
      {
        key: t('orderDetail.basicInfo.totalQuantity'),
        value: formatQuantity(order.total_quantity),
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

  // Render receive progress
  const renderReceiveProgress = () => {
    if (!order) return null

    const progress = order.receive_progress || 0
    const receivedQty = order.received_quantity || 0
    const totalQty = order.total_quantity || 0

    return (
      <div className="receive-progress-section">
        <div className="progress-header">
          <Text strong>{t('purchaseOrderDetail.receiveProgress.title')}</Text>
          <Text type="secondary">
            {t('purchaseOrderDetail.receiveProgress.received')}: {formatQuantity(receivedQty)} /{' '}
            {formatQuantity(totalQty)}
          </Text>
        </div>
        <Progress
          percent={Math.round(progress * 100)}
          showInfo
          stroke={progress >= 1 ? 'var(--color-success)' : undefined}
        />
      </div>
    )
  }

  // Render amount summary
  const renderAmountSummary = () => {
    if (!order) return null

    const discountPercent =
      order.discount_amount && order.total_amount
        ? ((order.discount_amount / (order.total_amount + order.discount_amount)) * 100).toFixed(1)
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
        {/* Print button - always available */}
        <PrintButton
          documentType="PURCHASE_ORDER"
          documentId={order.id || ''}
          documentNumber={order.order_number || ''}
          label={t('orderDetail.actions.print')}
          enableShortcut={true}
        />
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
        {(status === 'confirmed' || status === 'partial_received') && (
          <>
            <Button
              type="primary"
              icon={<IconBox />}
              onClick={handleReceive}
              loading={actionLoading}
            >
              {t('purchaseOrderDetail.actions.receive')}
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
      </Space>
    )
  }

  if (loading) {
    return (
      <Container size="lg" className="purchase-order-detail-page">
        <div className="loading-container">
          <Spin size="large" />
        </div>
      </Container>
    )
  }

  if (!order) {
    return (
      <Container size="lg" className="purchase-order-detail-page">
        <Empty title={t('orderDetail.notExist')} description={t('orderDetail.notExistDesc')} />
      </Container>
    )
  }

  return (
    <Container size="lg" className="purchase-order-detail-page">
      {/* Header */}
      <div className="page-header">
        <div className="header-left">
          <Button
            icon={<IconArrowLeft />}
            theme="borderless"
            onClick={() => navigate('/trade/purchase')}
          >
            {t('orderDetail.back')}
          </Button>
          <Title heading={4} className="page-title">
            {t('purchaseOrderDetail.title')}
          </Title>
          <Tag color={STATUS_TAG_COLORS[order.status || 'draft']} size="large">
            {t(
              `purchaseOrder.status.${order.status === 'partial_received' ? 'partialReceived' : order.status || 'draft'}`
            )}
          </Tag>
        </div>
        <div className="header-right">{renderActions()}</div>
      </div>

      {/* Order Info Card */}
      <Card className="info-card" title={t('orderDetail.basicInfo.title')}>
        {renderBasicInfo()}
        {renderReceiveProgress()}
      </Card>

      {/* Order Items Card */}
      <Card className="items-card" title={t('orderDetail.items.title')}>
        <Table
          columns={itemColumns}
          dataSource={
            (order.items || []) as (HandlerPurchaseOrderItemResponse & Record<string, unknown>)[]
          }
          rowKey="id"
          pagination={false}
          size="small"
          scroll={{ x: 1200 }}
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
    </Container>
  )
}
