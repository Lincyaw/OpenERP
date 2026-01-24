import { useState, useEffect, useCallback, useMemo } from 'react'
import { Card, Typography, Space, Spin, Tag, Empty, Toast, Progress } from '@douyinfe/semi-ui'
import {
  IconGridView,
  IconUserGroup,
  IconList,
  IconSend,
  IconPriceTag,
  IconCreditCard,
  IconAlertTriangle,
  IconTick,
  IconClock,
} from '@douyinfe/semi-icons'
import { useNavigate } from 'react-router-dom'
import { Container, Row, Stack } from '@/components/common/layout'
import { getProducts } from '@/api/products/products'
import { getCustomers } from '@/api/customers/customers'
import { getInventory } from '@/api/inventory/inventory'
import { getSalesOrders } from '@/api/sales-orders/sales-orders'
import { getFinanceApi } from '@/api/finance'
import './Dashboard.css'

const { Title, Text, Paragraph } = Typography

// Types for dashboard data
interface MetricCard {
  key: string
  label: string
  value: string | number
  subLabel?: string
  subValue?: string | number
  icon: React.ReactNode
  color: string
  onClick?: () => void
}

interface PendingTask {
  id: string
  type: 'order' | 'stock' | 'receivable' | 'payable'
  title: string
  description: string
  priority: 'high' | 'medium' | 'low'
  link: string
}

interface RecentOrder {
  id: string
  orderNumber: string
  customerName: string
  totalAmount: number
  status: string
  orderDate: string
}

/**
 * Format currency for display
 */
function formatCurrency(amount?: number): string {
  if (amount === undefined || amount === null) return '¥0.00'
  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: 'CNY',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(amount)
}

/**
 * Format number with comma separators
 */
function formatNumber(num?: number): string {
  if (num === undefined || num === null) return '0'
  return new Intl.NumberFormat('zh-CN').format(num)
}

/**
 * Format date for display
 */
function formatDate(dateStr?: string): string {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
  })
}

/**
 * Dashboard/Home page
 *
 * Features (P5-FE-006):
 * - Key metrics cards (products, customers, orders, inventory, finance)
 * - Sales trend overview (recent orders summary)
 * - Pending tasks section (actionable items)
 */
export default function DashboardPage() {
  const navigate = useNavigate()

  // API instances
  const productsApi = useMemo(() => getProducts(), [])
  const customersApi = useMemo(() => getCustomers(), [])
  const inventoryApi = useMemo(() => getInventory(), [])
  const salesOrdersApi = useMemo(() => getSalesOrders(), [])
  const financeApi = useMemo(() => getFinanceApi(), [])

  // Loading state
  const [loading, setLoading] = useState(true)

  // Metrics data
  const [productCount, setProductCount] = useState({ total: 0, active: 0 })
  const [customerCount, setCustomerCount] = useState({ total: 0, active: 0 })
  const [orderSummary, setOrderSummary] = useState({
    total: 0,
    draft: 0,
    confirmed: 0,
    shipped: 0,
    completed: 0,
  })
  const [lowStockCount, setLowStockCount] = useState(0)
  const [receivableSummary, setReceivableSummary] = useState({
    totalAmount: 0,
    pendingCount: 0,
  })
  const [payableSummary, setPayableSummary] = useState({
    totalAmount: 0,
    pendingCount: 0,
  })

  // Recent orders
  const [recentOrders, setRecentOrders] = useState<RecentOrder[]>([])

  // Pending tasks
  const [pendingTasks, setPendingTasks] = useState<PendingTask[]>([])

  // Fetch all dashboard data
  const fetchDashboardData = useCallback(async () => {
    setLoading(true)
    try {
      // Fetch all data in parallel
      const [
        productStatsRes,
        customerStatsRes,
        orderSummaryRes,
        lowStockRes,
        receivablesRes,
        payablesRes,
        recentOrdersRes,
      ] = await Promise.allSettled([
        productsApi.getCatalogProductsStatsCount(),
        customersApi.getPartnerCustomersStatsCount(),
        salesOrdersApi.getTradeSalesOrdersStatsSummary(),
        inventoryApi.getInventoryItemsAlertsLowStock({ page_size: 100 }),
        financeApi.getFinanceReceivablesSummary(),
        financeApi.getFinancePayablesSummary(),
        salesOrdersApi.getTradeSalesOrders({
          page_size: 5,
          order_by: 'order_date',
          order_dir: 'desc',
        }),
      ])

      // Process product stats
      if (productStatsRes.status === 'fulfilled' && productStatsRes.value.data) {
        const stats = productStatsRes.value.data
        setProductCount({
          total: (stats.active || 0) + (stats.inactive || 0) + (stats.discontinued || 0),
          active: stats.active || 0,
        })
      }

      // Process customer stats
      if (customerStatsRes.status === 'fulfilled' && customerStatsRes.value.data) {
        const stats = customerStatsRes.value.data
        setCustomerCount({
          total: (stats.active || 0) + (stats.inactive || 0) + (stats.suspended || 0),
          active: stats.active || 0,
        })
      }

      // Process order summary
      if (orderSummaryRes.status === 'fulfilled' && orderSummaryRes.value.data) {
        const summary = orderSummaryRes.value.data
        setOrderSummary({
          total: summary.total || 0,
          draft: summary.draft || 0,
          confirmed: summary.confirmed || 0,
          shipped: summary.shipped || 0,
          completed: summary.completed || 0,
        })
      }

      // Process low stock count
      if (lowStockRes.status === 'fulfilled' && lowStockRes.value.meta) {
        setLowStockCount(lowStockRes.value.meta.total || 0)
      }

      // Process receivables summary
      if (receivablesRes.status === 'fulfilled' && receivablesRes.value.data) {
        const summary = receivablesRes.value.data
        setReceivableSummary({
          totalAmount: summary.total_outstanding || 0,
          pendingCount: summary.pending_count || 0,
        })
      }

      // Process payables summary
      if (payablesRes.status === 'fulfilled' && payablesRes.value.data) {
        const summary = payablesRes.value.data
        setPayableSummary({
          totalAmount: summary.total_outstanding || 0,
          pendingCount: summary.pending_count || 0,
        })
      }

      // Process recent orders
      if (recentOrdersRes.status === 'fulfilled' && recentOrdersRes.value.data) {
        const orders = recentOrdersRes.value.data.map(
          (order: {
            id?: string
            order_number?: string
            customer_name?: string
            total_amount?: number
            status?: string
            order_date?: string
          }) => ({
            id: order.id || '',
            orderNumber: order.order_number || '',
            customerName: order.customer_name || '未知客户',
            totalAmount: order.total_amount || 0,
            status: order.status || '',
            orderDate: order.order_date || '',
          })
        )
        setRecentOrders(orders)
      }

      // Build pending tasks
      const tasks: PendingTask[] = []

      // Add draft orders as pending tasks
      if (orderSummaryRes.status === 'fulfilled' && orderSummaryRes.value.data?.draft) {
        const draftCount = orderSummaryRes.value.data.draft
        if (draftCount > 0) {
          tasks.push({
            id: 'draft-orders',
            type: 'order',
            title: `${draftCount} 个草稿订单待确认`,
            description: '有销售订单处于草稿状态，需要确认后才能发货',
            priority: 'medium',
            link: '/trade/sales-orders?status=DRAFT',
          })
        }
      }

      // Add confirmed orders as pending tasks
      if (orderSummaryRes.status === 'fulfilled' && orderSummaryRes.value.data?.confirmed) {
        const confirmedCount = orderSummaryRes.value.data.confirmed
        if (confirmedCount > 0) {
          tasks.push({
            id: 'confirmed-orders',
            type: 'order',
            title: `${confirmedCount} 个订单待发货`,
            description: '已确认的订单等待发货处理',
            priority: 'high',
            link: '/trade/sales-orders?status=CONFIRMED',
          })
        }
      }

      // Add low stock alerts
      if (lowStockRes.status === 'fulfilled' && lowStockRes.value.meta?.total) {
        const lowStockTotal = lowStockRes.value.meta.total
        if (lowStockTotal > 0) {
          tasks.push({
            id: 'low-stock',
            type: 'stock',
            title: `${lowStockTotal} 个商品库存不足`,
            description: '部分商品库存低于安全库存线，建议及时补货',
            priority: 'high',
            link: '/inventory/stock',
          })
        }
      }

      // Add pending receivables
      if (receivablesRes.status === 'fulfilled' && receivablesRes.value.data?.pending_count) {
        const pendingCount = receivablesRes.value.data.pending_count
        if (pendingCount > 0) {
          tasks.push({
            id: 'pending-receivables',
            type: 'receivable',
            title: `${pendingCount} 笔应收款待收`,
            description: formatCurrency(receivablesRes.value.data.total_outstanding) + ' 待收款',
            priority: 'medium',
            link: '/finance/receivables',
          })
        }
      }

      // Add pending payables
      if (payablesRes.status === 'fulfilled' && payablesRes.value.data?.pending_count) {
        const pendingCount = payablesRes.value.data.pending_count
        if (pendingCount > 0) {
          tasks.push({
            id: 'pending-payables',
            type: 'payable',
            title: `${pendingCount} 笔应付款待付`,
            description: formatCurrency(payablesRes.value.data.total_outstanding) + ' 待付款',
            priority: 'low',
            link: '/finance/payables',
          })
        }
      }

      setPendingTasks(tasks)
    } catch {
      Toast.error('获取仪表盘数据失败')
    } finally {
      setLoading(false)
    }
  }, [productsApi, customersApi, inventoryApi, salesOrdersApi, financeApi])

  // Fetch data on mount
  useEffect(() => {
    fetchDashboardData()
  }, [fetchDashboardData])

  // Build metric cards
  const metricCards: MetricCard[] = useMemo(
    () => [
      {
        key: 'products',
        label: '商品总数',
        value: formatNumber(productCount.total),
        subLabel: '启用商品',
        subValue: formatNumber(productCount.active),
        icon: <IconGridView size="large" />,
        color: 'var(--semi-color-primary)',
        onClick: () => navigate('/catalog/products'),
      },
      {
        key: 'customers',
        label: '客户总数',
        value: formatNumber(customerCount.total),
        subLabel: '活跃客户',
        subValue: formatNumber(customerCount.active),
        icon: <IconUserGroup size="large" />,
        color: 'var(--semi-color-success)',
        onClick: () => navigate('/partner/customers'),
      },
      {
        key: 'orders',
        label: '销售订单',
        value: formatNumber(orderSummary.total),
        subLabel: '待发货',
        subValue: formatNumber(orderSummary.confirmed),
        icon: <IconSend size="large" />,
        color: 'var(--semi-color-info)',
        onClick: () => navigate('/trade/sales-orders'),
      },
      {
        key: 'lowStock',
        label: '库存预警',
        value: formatNumber(lowStockCount),
        subLabel: '需要补货',
        subValue: lowStockCount > 0 ? '!' : '-',
        icon: <IconAlertTriangle size="large" />,
        color: lowStockCount > 0 ? 'var(--semi-color-danger)' : 'var(--semi-color-tertiary)',
        onClick: () => navigate('/inventory/stock'),
      },
      {
        key: 'receivables',
        label: '应收账款',
        value: formatCurrency(receivableSummary.totalAmount),
        subLabel: '待收笔数',
        subValue: formatNumber(receivableSummary.pendingCount),
        icon: <IconPriceTag size="large" />,
        color: 'var(--semi-color-warning)',
        onClick: () => navigate('/finance/receivables'),
      },
      {
        key: 'payables',
        label: '应付账款',
        value: formatCurrency(payableSummary.totalAmount),
        subLabel: '待付笔数',
        subValue: formatNumber(payableSummary.pendingCount),
        icon: <IconCreditCard size="large" />,
        color: 'var(--semi-color-tertiary)',
        onClick: () => navigate('/finance/payables'),
      },
    ],
    [
      productCount,
      customerCount,
      orderSummary,
      lowStockCount,
      receivableSummary,
      payableSummary,
      navigate,
    ]
  )

  // Get priority tag color
  const getPriorityColor = (priority: string): 'red' | 'orange' | 'blue' => {
    switch (priority) {
      case 'high':
        return 'red'
      case 'medium':
        return 'orange'
      default:
        return 'blue'
    }
  }

  // Get priority label
  const getPriorityLabel = (priority: string): string => {
    switch (priority) {
      case 'high':
        return '紧急'
      case 'medium':
        return '重要'
      default:
        return '一般'
    }
  }

  // Get task type icon
  const getTaskIcon = (type: string): React.ReactNode => {
    switch (type) {
      case 'order':
        return <IconSend />
      case 'stock':
        return <IconList />
      case 'receivable':
        return <IconPriceTag />
      case 'payable':
        return <IconCreditCard />
      default:
        return <IconClock />
    }
  }

  // Get order status tag
  const getOrderStatusTag = (status: string): React.ReactNode => {
    const statusConfig: Record<string, { color: string; label: string }> = {
      DRAFT: { color: 'grey', label: '草稿' },
      CONFIRMED: { color: 'blue', label: '已确认' },
      SHIPPED: { color: 'cyan', label: '已发货' },
      COMPLETED: { color: 'green', label: '已完成' },
      CANCELLED: { color: 'red', label: '已取消' },
    }
    const config = statusConfig[status] || { color: 'grey', label: status }
    return (
      <Tag color={config.color as 'grey' | 'blue' | 'cyan' | 'green' | 'red'}>{config.label}</Tag>
    )
  }

  // Calculate order completion rate
  const orderCompletionRate = useMemo(() => {
    if (orderSummary.total === 0) return 0
    return Math.round((orderSummary.completed / orderSummary.total) * 100)
  }, [orderSummary])

  return (
    <Container size="full" className="dashboard-page">
      <Spin spinning={loading} size="large">
        {/* Page Header */}
        <div className="dashboard-header">
          <Title heading={3} style={{ margin: 0 }}>
            工作台
          </Title>
          <Text type="secondary">欢迎回来，这是您的业务概览</Text>
        </div>

        {/* Metric Cards */}
        <div className="dashboard-metrics">
          <Row gap="md" wrap="wrap">
            {metricCards.map((card) => (
              <div key={card.key} className="metric-card-wrapper">
                <Card
                  className="metric-card"
                  style={{ cursor: card.onClick ? 'pointer' : 'default' }}
                >
                  <div
                    className="metric-card-content"
                    onClick={card.onClick}
                    role={card.onClick ? 'button' : undefined}
                    tabIndex={card.onClick ? 0 : undefined}
                    onKeyDown={(e) => {
                      if (card.onClick && (e.key === 'Enter' || e.key === ' ')) {
                        card.onClick()
                      }
                    }}
                  >
                    <div
                      className="metric-icon"
                      style={{
                        backgroundColor: card.color + '15',
                        color: card.color,
                      }}
                    >
                      {card.icon}
                    </div>
                    <div className="metric-info">
                      <Text type="tertiary" className="metric-label">
                        {card.label}
                      </Text>
                      <Title heading={3} className="metric-value" style={{ margin: 0 }}>
                        {card.value}
                      </Title>
                      {card.subLabel && (
                        <Text type="tertiary" size="small" className="metric-sub">
                          {card.subLabel}: <Text strong>{card.subValue}</Text>
                        </Text>
                      )}
                    </div>
                  </div>
                </Card>
              </div>
            ))}
          </Row>
        </div>

        {/* Main Content Area */}
        <Row gap="md" wrap="wrap" className="dashboard-content">
          {/* Left Column - Recent Orders & Order Stats */}
          <div className="dashboard-col-left">
            <Stack gap="md">
              {/* Order Statistics */}
              <Card title="订单统计" className="stats-card">
                <div className="order-stats">
                  <div className="order-progress">
                    <Progress
                      percent={orderCompletionRate}
                      type="circle"
                      width={100}
                      format={() => (
                        <div className="progress-content">
                          <Text strong>{orderCompletionRate}%</Text>
                          <Text type="tertiary" size="small">
                            完成率
                          </Text>
                        </div>
                      )}
                    />
                  </div>
                  <div className="order-breakdown">
                    <div className="breakdown-item">
                      <span className="breakdown-dot draft"></span>
                      <Text>草稿</Text>
                      <Text strong>{orderSummary.draft}</Text>
                    </div>
                    <div className="breakdown-item">
                      <span className="breakdown-dot confirmed"></span>
                      <Text>已确认</Text>
                      <Text strong>{orderSummary.confirmed}</Text>
                    </div>
                    <div className="breakdown-item">
                      <span className="breakdown-dot shipped"></span>
                      <Text>已发货</Text>
                      <Text strong>{orderSummary.shipped}</Text>
                    </div>
                    <div className="breakdown-item">
                      <span className="breakdown-dot completed"></span>
                      <Text>已完成</Text>
                      <Text strong>{orderSummary.completed}</Text>
                    </div>
                  </div>
                </div>
              </Card>

              {/* Recent Orders */}
              <Card
                title="最近订单"
                className="recent-orders-card"
                headerExtraContent={
                  <Text
                    link
                    onClick={() => navigate('/trade/sales-orders')}
                    style={{ cursor: 'pointer' }}
                  >
                    查看全部
                  </Text>
                }
              >
                {recentOrders.length === 0 ? (
                  <Empty description="暂无订单" />
                ) : (
                  <div className="recent-orders-list">
                    {recentOrders.map((order) => (
                      <div
                        key={order.id}
                        className="recent-order-item"
                        onClick={() => navigate(`/trade/sales-orders/${order.id}`)}
                      >
                        <div className="order-main">
                          <Text strong className="order-number">
                            {order.orderNumber}
                          </Text>
                          <Text type="tertiary" size="small">
                            {order.customerName}
                          </Text>
                        </div>
                        <div className="order-info">
                          <Text className="order-amount">{formatCurrency(order.totalAmount)}</Text>
                          <Space>
                            {getOrderStatusTag(order.status)}
                            <Text type="tertiary" size="small">
                              {formatDate(order.orderDate)}
                            </Text>
                          </Space>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </Card>
            </Stack>
          </div>

          {/* Right Column - Pending Tasks */}
          <div className="dashboard-col-right">
            <Card title="待办事项" className="pending-tasks-card">
              {pendingTasks.length === 0 ? (
                <div className="no-tasks">
                  <IconTick size="extra-large" style={{ color: 'var(--semi-color-success)' }} />
                  <Paragraph type="secondary" style={{ marginTop: 16 }}>
                    太棒了！暂无待办事项
                  </Paragraph>
                </div>
              ) : (
                <div className="pending-tasks-list">
                  {pendingTasks.map((task) => (
                    <div
                      key={task.id}
                      className="pending-task-item"
                      onClick={() => navigate(task.link)}
                    >
                      <div className="task-icon" style={{ color: getPriorityColor(task.priority) }}>
                        {getTaskIcon(task.type)}
                      </div>
                      <div className="task-content">
                        <div className="task-header">
                          <Text strong>{task.title}</Text>
                          <Tag size="small" color={getPriorityColor(task.priority)}>
                            {getPriorityLabel(task.priority)}
                          </Tag>
                        </div>
                        <Text type="tertiary" size="small">
                          {task.description}
                        </Text>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </Card>
          </div>
        </Row>
      </Spin>
    </Container>
  )
}
