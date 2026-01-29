import { useState, useEffect, useCallback, useMemo } from 'react'
import { Card, Typography, Table, Toast, Modal, Empty, Progress } from '@douyinfe/semi-ui-19'
import { IconEdit, IconTick, IconClose, IconBox, IconPrint } from '@douyinfe/semi-icons'
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
import { PrintButton } from '@/components/printing'
import {
  getPurchaseOrderById,
  confirmPurchaseOrder,
  cancelPurchaseOrder,
} from '@/api/purchase-orders/purchase-orders'
import type { HandlerPurchaseOrderResponse, HandlerPurchaseOrderItemResponse } from '@/api/models'
import { useI18n } from '@/hooks/useI18n'
import { useFormatters } from '@/hooks/useFormatters'
import './PurchaseOrderDetail.css'

const { Text } = Typography

// Status variant mapping for DetailPageHeader
const STATUS_VARIANTS: Record<
  string,
  'default' | 'primary' | 'success' | 'warning' | 'danger' | 'info'
> = {
  draft: 'default',
  confirmed: 'info',
  partial_received: 'warning',
  completed: 'success',
  cancelled: 'danger',
}

/**
 * Purchase Order Detail Page
 *
 * Features:
 * - Display complete order information using DetailPageHeader
 * - Display order status flow using StatusFlow component
 * - Display order line items with receive progress
 * - Status action buttons (confirm, receive, cancel)
 */
export default function PurchaseOrderDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { t } = useI18n({ ns: 'trade' })
  const { formatCurrency, formatDateTime } = useFormatters()

  const [order, setOrder] = useState<HandlerPurchaseOrderResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState(false)

  // Fetch order details
  const fetchOrder = useCallback(async () => {
    if (!id) return

    setLoading(true)
    try {
      const response = await getPurchaseOrderById(id)
      if (response.status === 200 && response.data.success && response.data.data) {
        setOrder(response.data.data)
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
  }, [id, navigate, t])

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
          await confirmPurchaseOrder(order.id!, {})
          Toast.success(t('purchaseOrderDetail.messages.confirmSuccess'))
          fetchOrder()
        } catch {
          Toast.error(t('purchaseOrderDetail.messages.confirmError'))
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [order, fetchOrder, t])

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
          await cancelPurchaseOrder(order.id!, {
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
  }, [order, fetchOrder, t])

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

  // Build status for DetailPageHeader
  const headerStatus = useMemo((): DetailPageHeaderStatus | undefined => {
    if (!order) return undefined
    const status = order.status || 'draft'
    const statusKey = status === 'partial_received' ? 'partialReceived' : status
    return {
      label: t(`purchaseOrder.status.${statusKey}`),
      variant: STATUS_VARIANTS[status] || 'default',
    }
  }, [order, t])

  // Build metrics for DetailPageHeader
  const headerMetrics = useMemo((): DetailPageHeaderMetric[] => {
    if (!order) return []
    const progress = order.receive_progress || 0
    return [
      {
        label: t('orderDetail.basicInfo.supplierName'),
        value: order.supplier_name || '-',
      },
      {
        label: t('orderDetail.basicInfo.itemCount'),
        value: `${order.item_count || 0} ${t('purchaseOrder.unit', { defaultValue: '种' })}`,
      },
      {
        label: t('purchaseOrderDetail.receiveProgress.title'),
        value: `${Math.round(progress * 100)}%`,
        variant: progress >= 1 ? 'success' : progress > 0 ? 'warning' : 'default',
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
    if (status === 'confirmed' || status === 'partial_received') {
      return {
        key: 'receive',
        label: t('purchaseOrderDetail.actions.receive'),
        icon: <IconBox />,
        type: 'primary',
        onClick: handleReceive,
        loading: actionLoading,
      }
    }
    return undefined
  }, [order, t, handleConfirm, handleReceive, actionLoading])

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

    if (status === 'confirmed' || status === 'partial_received') {
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
        label: t('purchaseOrder.status.draft'),
        state: 'completed',
        timestamp: order.created_at ? formatDateTime(order.created_at) : undefined,
      },
      {
        key: 'confirmed',
        label: t('purchaseOrder.status.confirmed'),
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
        key: 'partial_received',
        label: t('purchaseOrder.status.partialReceived'),
        state:
          status === 'partial_received'
            ? 'current'
            : status === 'completed'
              ? 'completed'
              : isCancelled
                ? 'cancelled'
                : 'pending',
      },
      {
        key: 'completed',
        label: t('purchaseOrder.status.completed'),
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
        label: t('purchaseOrder.status.cancelled'),
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
            {t('orderDetail.basicInfo.supplierName')}
          </Text>
          <Text className="order-info-value">{order.supplier_name || '-'}</Text>
        </div>
        <div className="order-info-item">
          <Text type="secondary" className="order-info-label">
            {t('orderDetail.basicInfo.totalQuantity')}
          </Text>
          <Text className="order-info-value">{formatQuantity(order.total_quantity)}</Text>
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

  if (loading) {
    return (
      <Container size="lg" className="purchase-order-detail-page">
        <DetailPageHeader
          title={t('purchaseOrderDetail.title')}
          loading={true}
          showBack={true}
          onBack={() => navigate('/trade/purchase')}
          backLabel={t('orderDetail.back')}
        />
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
      {/* Unified Header */}
      <DetailPageHeader
        title={t('purchaseOrderDetail.title')}
        documentNumber={order.order_number}
        status={headerStatus}
        metrics={headerMetrics}
        primaryAction={primaryAction}
        secondaryActions={secondaryActions}
        onBack={() => navigate('/trade/purchase')}
        backLabel={t('orderDetail.back')}
      />

      {/* Hidden Print Button for secondary action click */}
      <div style={{ display: 'none' }}>
        <PrintButton
          documentType="PURCHASE_ORDER"
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
    </Container>
  )
}
