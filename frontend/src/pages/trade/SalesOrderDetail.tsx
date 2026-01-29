import { useState, useEffect, useCallback, useMemo } from 'react'
import { Card, Typography, Table, Toast, Modal, Empty } from '@douyinfe/semi-ui-19'
import { IconEdit, IconTick, IconClose, IconSend, IconPrint } from '@douyinfe/semi-icons'
import { useParams, useNavigate } from 'react-router-dom'
import { Container } from '@/components/common/layout'
import {
  DetailPageHeader,
  StatusFlow,
  type DetailPageHeaderAction,
  type DetailPageHeaderStatus,
  type DetailPageHeaderMetric,
  type StatusFlowStep,
} from '@/components/common'
import {
  getSalesOrderById,
  confirmSalesOrder,
  shipSalesOrder,
  completeSalesOrder,
  cancelSalesOrder,
} from '@/api/sales-orders/sales-orders'
import type { HandlerSalesOrderResponse, HandlerSalesOrderItemResponse } from '@/api/models'
import { ShipOrderModal } from './components'
import { PrintButton } from '@/components/printing'
import { useI18n } from '@/hooks/useI18n'
import { useFormatters } from '@/hooks/useFormatters'
import { safeToFixed, toNumber } from '@/utils'
import './SalesOrderDetail.css'

const { Text } = Typography

// Status variant mapping for DetailPageHeader
const STATUS_VARIANTS: Record<
  string,
  'default' | 'primary' | 'success' | 'warning' | 'danger' | 'info'
> = {
  draft: 'default',
  confirmed: 'info',
  shipped: 'primary',
  completed: 'success',
  cancelled: 'danger',
}

/**
 * Sales Order Detail Page
 *
 * Features:
 * - Display complete order information using DetailPageHeader
 * - Display order status flow using StatusFlow component
 * - Display order line items
 * - Status action buttons (confirm, ship, complete, cancel)
 */
export default function SalesOrderDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { t } = useI18n({ ns: 'trade' })
  const { formatCurrency, formatDateTime } = useFormatters()

  const [order, setOrder] = useState<HandlerSalesOrderResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState(false)
  const [shipModalVisible, setShipModalVisible] = useState(false)

  // Fetch order details
  const fetchOrder = useCallback(async () => {
    if (!id) return

    setLoading(true)
    try {
      const response = await getSalesOrderById(id)
      if (response.status === 200 && response.data.success && response.data.data) {
        setOrder(response.data.data)
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
  }, [id, navigate, t])

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
          await confirmSalesOrder(order.id!, {})
          Toast.success(t('orderDetail.messages.confirmSuccess'))
          fetchOrder()
        } catch (error) {
          Toast.error(t('orderDetail.messages.confirmError'))
          throw error
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [order, fetchOrder, t])

  // Handle ship order - open modal
  const handleShip = useCallback(() => {
    setShipModalVisible(true)
  }, [])

  // Handle ship confirm from modal
  const handleShipConfirm = useCallback(
    async (warehouseId: string) => {
      if (!order?.id) return

      try {
        await shipSalesOrder(order.id, {
          warehouse_id: warehouseId,
        })
        Toast.success(t('orderDetail.messages.shipSuccess'))
        setShipModalVisible(false)
        fetchOrder()
      } catch {
        Toast.error(t('orderDetail.messages.shipError'))
        throw new Error(t('orderDetail.messages.shipError'))
      }
    },
    [order, fetchOrder, t]
  )

  // Handle complete order
  const handleComplete = useCallback(async () => {
    if (!order?.id) return
    setActionLoading(true)
    try {
      await completeSalesOrder(order.id, {})
      Toast.success(t('orderDetail.messages.completeSuccess'))
      fetchOrder()
    } catch {
      Toast.error(t('orderDetail.messages.completeError'))
    } finally {
      setActionLoading(false)
    }
  }, [order, fetchOrder, t])

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
          await cancelSalesOrder(order.id!, {
            reason: t('common.userCancel'),
          })
          Toast.success(t('orderDetail.messages.cancelSuccess'))
          fetchOrder()
        } catch (error) {
          Toast.error(t('orderDetail.messages.cancelError'))
          throw error
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [order, fetchOrder, t])

  // Handle edit order
  const handleEdit = useCallback(() => {
    if (order?.id) {
      navigate(`/trade/sales/${order.id}/edit`)
    }
  }, [order, navigate])

  // Build status for DetailPageHeader
  const headerStatus = useMemo((): DetailPageHeaderStatus | undefined => {
    if (!order) return undefined
    const status = order.status || 'draft'
    return {
      label: t(`salesOrder.status.${status}`),
      variant: STATUS_VARIANTS[status] || 'default',
    }
  }, [order, t])

  // Build metrics for DetailPageHeader
  const headerMetrics = useMemo((): DetailPageHeaderMetric[] => {
    if (!order) return []
    return [
      {
        label: t('orderDetail.basicInfo.customerName'),
        value: order.customer_name || '-',
      },
      {
        label: t('orderDetail.basicInfo.itemCount'),
        value: `${order.item_count || 0} ${t('salesOrder.unit')}`,
      },
      {
        label: t('orderDetail.amount.payableAmount'),
        value: formatCurrency(order.payable_amount || 0),
        variant: 'primary',
      },
    ]
  }, [order, t, formatCurrency])

  // Build primary action for DetailPageHeader
  const primaryAction = useMemo((): DetailPageHeaderAction | undefined => {
    if (!order) return undefined
    const status = order.status || 'draft'

    if (status === 'draft') {
      return {
        key: 'confirm',
        label: t('orderDetail.actions.confirmOrder'),
        icon: <IconTick />,
        type: 'primary',
        onClick: handleConfirm,
        loading: actionLoading,
      }
    }
    if (status === 'confirmed') {
      return {
        key: 'ship',
        label: t('orderDetail.actions.ship'),
        icon: <IconSend />,
        type: 'primary',
        onClick: handleShip,
        loading: actionLoading,
      }
    }
    if (status === 'shipped') {
      return {
        key: 'complete',
        label: t('orderDetail.actions.complete'),
        icon: <IconTick />,
        type: 'primary',
        onClick: handleComplete,
        loading: actionLoading,
      }
    }
    return undefined
  }, [order, t, handleConfirm, handleShip, handleComplete, actionLoading])

  // Build secondary actions for DetailPageHeader
  const secondaryActions = useMemo((): DetailPageHeaderAction[] => {
    if (!order) return []
    const status = order.status || 'draft'
    const actions: DetailPageHeaderAction[] = []

    // Print button is always available
    actions.push({
      key: 'print',
      label: t('orderDetail.actions.print'),
      icon: <IconPrint />,
      onClick: () => {
        // PrintButton handles this, but we need a placeholder
        const printBtn = document.querySelector('.detail-page-print-btn') as HTMLButtonElement
        printBtn?.click()
      },
    })

    if (status === 'draft') {
      actions.push({
        key: 'edit',
        label: t('orderDetail.actions.edit'),
        icon: <IconEdit />,
        onClick: handleEdit,
        disabled: actionLoading,
      })
      actions.push({
        key: 'cancel',
        label: t('orderDetail.actions.cancel'),
        icon: <IconClose />,
        type: 'danger',
        onClick: handleCancel,
        loading: actionLoading,
      })
    }

    if (status === 'confirmed') {
      actions.push({
        key: 'cancel',
        label: t('orderDetail.actions.cancel'),
        icon: <IconClose />,
        type: 'danger',
        onClick: handleCancel,
        loading: actionLoading,
      })
    }

    return actions
  }, [order, t, handleEdit, handleCancel, actionLoading])

  // Build status flow steps
  const statusFlowSteps = useMemo((): StatusFlowStep[] => {
    if (!order) return []
    const status = order.status || 'draft'
    const isCancelled = status === 'cancelled'

    // Define the normal flow
    const steps: StatusFlowStep[] = [
      {
        key: 'draft',
        label: t('salesOrder.status.draft'),
        state: 'completed',
        timestamp: order.created_at ? formatDateTime(order.created_at) : undefined,
      },
      {
        key: 'confirmed',
        label: t('salesOrder.status.confirmed'),
        state: order.confirmed_at
          ? 'completed'
          : status === 'confirmed'
            ? 'current'
            : isCancelled
              ? 'cancelled'
              : 'pending',
        timestamp: order.confirmed_at ? formatDateTime(order.confirmed_at) : undefined,
      },
      {
        key: 'shipped',
        label: t('salesOrder.status.shipped'),
        state: order.shipped_at
          ? 'completed'
          : status === 'shipped'
            ? 'current'
            : isCancelled
              ? 'cancelled'
              : 'pending',
        timestamp: order.shipped_at ? formatDateTime(order.shipped_at) : undefined,
      },
      {
        key: 'completed',
        label: t('salesOrder.status.completed'),
        state: order.completed_at
          ? 'completed'
          : status === 'completed'
            ? 'current'
            : isCancelled
              ? 'cancelled'
              : 'pending',
        timestamp: order.completed_at ? formatDateTime(order.completed_at) : undefined,
      },
    ]

    // Update current step based on actual status
    if (status === 'draft') {
      steps[0].state = 'current'
      steps[1].state = 'pending'
    }

    // If cancelled, add cancelled step at the end
    if (isCancelled) {
      steps.push({
        key: 'cancelled',
        label: t('salesOrder.status.cancelled'),
        state: 'rejected',
        timestamp: order.cancelled_at ? formatDateTime(order.cancelled_at) : undefined,
        description: order.cancel_reason || undefined,
      })
    }

    return steps
  }, [order, t, formatDateTime])

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

  // Render order basic info
  const renderBasicInfo = () => {
    if (!order) return null

    return (
      <div className="order-info-grid">
        <div className="order-info-item">
          <Text type="secondary" className="order-info-label">
            {t('orderDetail.basicInfo.orderNumber')}
          </Text>
          <Text strong className="order-info-value order-number-value">
            {order.order_number}
          </Text>
        </div>
        <div className="order-info-item">
          <Text type="secondary" className="order-info-label">
            {t('orderDetail.basicInfo.customerName')}
          </Text>
          <Text className="order-info-value">{order.customer_name || '-'}</Text>
        </div>
        <div className="order-info-item">
          <Text type="secondary" className="order-info-label">
            {t('orderDetail.basicInfo.totalQuantity')}
          </Text>
          <Text className="order-info-value">{safeToFixed(order.total_quantity, 2, '0.00')}</Text>
        </div>
        <div className="order-info-item">
          <Text type="secondary" className="order-info-label">
            {t('orderDetail.basicInfo.createdAt')}
          </Text>
          <Text className="order-info-value">
            {order.created_at ? formatDateTime(order.created_at) : '-'}
          </Text>
        </div>
        <div className="order-info-item">
          <Text type="secondary" className="order-info-label">
            {t('orderDetail.basicInfo.updatedAt')}
          </Text>
          <Text className="order-info-value">
            {order.updated_at ? formatDateTime(order.updated_at) : '-'}
          </Text>
        </div>
        <div className="order-info-item order-info-item--full">
          <Text type="secondary" className="order-info-label">
            {t('orderDetail.basicInfo.remark')}
          </Text>
          <Text className="order-info-value">{order.remark || '-'}</Text>
        </div>
      </div>
    )
  }

  // Render amount summary
  const renderAmountSummary = () => {
    if (!order) return null

    const discountPercent =
      order.discount_amount && order.total_amount
        ? safeToFixed(
            (toNumber(order.discount_amount) /
              (toNumber(order.total_amount) + toNumber(order.discount_amount))) *
              100,
            1,
            '0'
          )
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

  if (loading) {
    return (
      <Container size="lg" className="sales-order-detail-page">
        <DetailPageHeader
          title={t('orderDetail.title')}
          loading={true}
          showBack={true}
          onBack={() => navigate('/trade/sales')}
          backLabel={t('orderDetail.back')}
        />
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
      {/* Unified Header */}
      <DetailPageHeader
        title={t('orderDetail.title')}
        documentNumber={order.order_number}
        status={headerStatus}
        metrics={headerMetrics}
        primaryAction={primaryAction}
        secondaryActions={secondaryActions}
        onBack={() => navigate('/trade/sales')}
        backLabel={t('orderDetail.back')}
      />

      {/* Hidden Print Button for secondary action click */}
      <div style={{ display: 'none' }}>
        <PrintButton
          documentType="SALES_ORDER"
          documentId={order.id || ''}
          documentNumber={order.order_number || ''}
          label={t('orderDetail.actions.print')}
          enableShortcut={true}
          className="detail-page-print-btn"
        />
      </div>

      {/* Status Flow */}
      <Card className="status-flow-card" title={t('orderDetail.timeline.title')}>
        <StatusFlow
          steps={statusFlowSteps}
          showTimestamp
          ariaLabel={t('orderDetail.timeline.title')}
        />
      </Card>

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
