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
} from '@douyinfe/semi-ui'
import { IconArrowLeft, IconEdit, IconTick, IconClose, IconSend } from '@douyinfe/semi-icons'
import { useParams, useNavigate } from 'react-router-dom'
import { Container } from '@/components/common/layout'
import { getSalesOrders } from '@/api/sales-orders/sales-orders'
import type { HandlerSalesOrderResponse, HandlerSalesOrderItemResponse } from '@/api/models'
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

// Status labels
const STATUS_LABELS: Record<string, string> = {
  draft: '草稿',
  confirmed: '已确认',
  shipped: '已发货',
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
 * Format datetime for display
 */
function formatDateTime(dateStr?: string): string {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
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
  const salesOrderApi = useMemo(() => getSalesOrders(), [])

  const [order, setOrder] = useState<HandlerSalesOrderResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState(false)

  // Fetch order details
  const fetchOrder = useCallback(async () => {
    if (!id) return

    setLoading(true)
    try {
      const response = await salesOrderApi.getTradeSalesOrdersId(id)
      if (response.success && response.data) {
        setOrder(response.data)
      } else {
        Toast.error('订单不存在')
        navigate('/trade/sales')
      }
    } catch {
      Toast.error('获取订单详情失败')
      navigate('/trade/sales')
    } finally {
      setLoading(false)
    }
  }, [id, salesOrderApi, navigate])

  useEffect(() => {
    fetchOrder()
  }, [fetchOrder])

  // Handle confirm order
  const handleConfirm = useCallback(async () => {
    if (!order?.id) return
    Modal.confirm({
      title: '确认订单',
      content: `确定要确认订单 "${order.order_number}" 吗？确认后将锁定库存。`,
      okText: '确认',
      cancelText: '取消',
      onOk: async () => {
        setActionLoading(true)
        try {
          await salesOrderApi.postTradeSalesOrdersIdConfirm(order.id!, {})
          Toast.success('订单已确认')
          fetchOrder()
        } catch {
          Toast.error('确认订单失败')
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [order, salesOrderApi, fetchOrder])

  // Handle ship order
  const handleShip = useCallback(async () => {
    if (!order?.id) return
    Modal.confirm({
      title: '发货',
      content: `确定要为订单 "${order.order_number}" 发货吗？发货后将扣减库存。`,
      okText: '确认发货',
      cancelText: '取消',
      onOk: async () => {
        setActionLoading(true)
        try {
          await salesOrderApi.postTradeSalesOrdersIdShip(order.id!, {
            warehouse_id: order.warehouse_id,
          })
          Toast.success('订单已发货')
          fetchOrder()
        } catch {
          Toast.error('发货失败')
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [order, salesOrderApi, fetchOrder])

  // Handle complete order
  const handleComplete = useCallback(async () => {
    if (!order?.id) return
    setActionLoading(true)
    try {
      await salesOrderApi.postTradeSalesOrdersIdComplete(order.id)
      Toast.success('订单已完成')
      fetchOrder()
    } catch {
      Toast.error('完成订单失败')
    } finally {
      setActionLoading(false)
    }
  }, [order, salesOrderApi, fetchOrder])

  // Handle cancel order
  const handleCancel = useCallback(async () => {
    if (!order?.id) return
    Modal.confirm({
      title: '取消订单',
      content: `确定要取消订单 "${order.order_number}" 吗？`,
      okText: '确认取消',
      cancelText: '返回',
      okButtonProps: { type: 'danger' },
      onOk: async () => {
        setActionLoading(true)
        try {
          await salesOrderApi.postTradeSalesOrdersIdCancel(order.id!, { reason: '用户取消' })
          Toast.success('订单已取消')
          fetchOrder()
        } catch {
          Toast.error('取消订单失败')
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [order, salesOrderApi, fetchOrder])

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
        title: '序号',
        dataIndex: 'index',
        width: 60,
        render: (_: unknown, __: unknown, index: number) => index + 1,
      },
      {
        title: '商品编码',
        dataIndex: 'product_code',
        width: 120,
        render: (code: string) => <Text className="product-code">{code || '-'}</Text>,
      },
      {
        title: '商品名称',
        dataIndex: 'product_name',
        width: 200,
        ellipsis: true,
      },
      {
        title: '单位',
        dataIndex: 'unit',
        width: 80,
        align: 'center' as const,
        render: (unit: string) => unit || '-',
      },
      {
        title: '数量',
        dataIndex: 'quantity',
        width: 100,
        align: 'right' as const,
        render: (qty: number) => qty?.toFixed(2) || '-',
      },
      {
        title: '单价',
        dataIndex: 'unit_price',
        width: 120,
        align: 'right' as const,
        render: (price: number) => formatPrice(price),
      },
      {
        title: '金额',
        dataIndex: 'amount',
        width: 120,
        align: 'right' as const,
        render: (amount: number) => <Text className="item-amount">{formatPrice(amount)}</Text>,
      },
      {
        title: '备注',
        dataIndex: 'remark',
        ellipsis: true,
        render: (remark: string) => remark || '-',
      },
    ],
    []
  )

  // Build timeline items based on order status
  const timelineItems = useMemo(() => {
    if (!order) return []

    const items = []

    // Created
    if (order.created_at) {
      items.push({
        time: formatDateTime(order.created_at),
        content: '订单创建',
        type: 'default' as const,
      })
    }

    // Confirmed
    if (order.confirmed_at) {
      items.push({
        time: formatDateTime(order.confirmed_at),
        content: '订单确认',
        type: 'success' as const,
      })
    }

    // Shipped
    if (order.shipped_at) {
      items.push({
        time: formatDateTime(order.shipped_at),
        content: '订单发货',
        type: 'success' as const,
      })
    }

    // Completed
    if (order.completed_at) {
      items.push({
        time: formatDateTime(order.completed_at),
        content: '订单完成',
        type: 'success' as const,
      })
    }

    // Cancelled
    if (order.cancelled_at) {
      items.push({
        time: formatDateTime(order.cancelled_at),
        content: `订单取消${order.cancel_reason ? `: ${order.cancel_reason}` : ''}`,
        type: 'error' as const,
      })
    }

    return items
  }, [order])

  // Render order basic info
  const renderBasicInfo = () => {
    if (!order) return null

    const data = [
      { key: '订单编号', value: order.order_number },
      { key: '客户名称', value: order.customer_name || '-' },
      {
        key: '订单状态',
        value: (
          <Tag color={STATUS_TAG_COLORS[order.status || 'draft']}>
            {STATUS_LABELS[order.status || 'draft']}
          </Tag>
        ),
      },
      { key: '商品数量', value: `${order.item_count || 0} 件` },
      { key: '总数量', value: order.total_quantity?.toFixed(2) || '0.00' },
      { key: '创建时间', value: formatDateTime(order.created_at) },
      { key: '更新时间', value: formatDateTime(order.updated_at) },
      { key: '备注', value: order.remark || '-' },
    ]

    return <Descriptions data={data} row className="order-basic-info" />
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
          <Text type="secondary">商品金额</Text>
          <Text>{formatPrice((order.total_amount || 0) + (order.discount_amount || 0))}</Text>
        </div>
        <div className="amount-row">
          <Text type="secondary">优惠金额 ({discountPercent}%)</Text>
          <Text className="discount-amount">-{formatPrice(order.discount_amount)}</Text>
        </div>
        <div className="amount-row total-row">
          <Text strong>应付金额</Text>
          <Text className="payable-amount" strong>
            {formatPrice(order.payable_amount)}
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
              编辑
            </Button>
            <Button
              type="primary"
              icon={<IconTick />}
              onClick={handleConfirm}
              loading={actionLoading}
            >
              确认订单
            </Button>
            <Button
              type="danger"
              icon={<IconClose />}
              onClick={handleCancel}
              loading={actionLoading}
            >
              取消
            </Button>
          </>
        )}
        {status === 'confirmed' && (
          <>
            <Button type="primary" icon={<IconSend />} onClick={handleShip} loading={actionLoading}>
              发货
            </Button>
            <Button
              type="warning"
              icon={<IconClose />}
              onClick={handleCancel}
              loading={actionLoading}
            >
              取消
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
            完成
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
        <Empty title="订单不存在" description="您访问的订单不存在或已被删除" />
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
            返回列表
          </Button>
          <Title heading={4} className="page-title">
            订单详情
          </Title>
          <Tag color={STATUS_TAG_COLORS[order.status || 'draft']} size="large">
            {STATUS_LABELS[order.status || 'draft']}
          </Tag>
        </div>
        <div className="header-right">{renderActions()}</div>
      </div>

      {/* Order Info Card */}
      <Card className="info-card" title="基本信息">
        {renderBasicInfo()}
      </Card>

      {/* Order Items Card */}
      <Card className="items-card" title="商品明细">
        <Table
          columns={itemColumns}
          dataSource={
            (order.items || []) as (HandlerSalesOrderItemResponse & Record<string, unknown>)[]
          }
          rowKey="id"
          pagination={false}
          size="small"
          empty={<Empty description="暂无商品" />}
        />
        {renderAmountSummary()}
      </Card>

      {/* Timeline Card */}
      <Card className="timeline-card" title="状态变更">
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
          <Empty description="暂无状态记录" />
        )}
      </Card>
    </Container>
  )
}
