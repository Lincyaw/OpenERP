import { useState, useEffect, useCallback, useMemo } from 'react'
import { Card, Typography, Table, Toast, Modal, Empty, TextArea } from '@douyinfe/semi-ui-19'
import { IconEdit, IconTick, IconClose, IconSend, IconBox, IconPrint } from '@douyinfe/semi-icons'
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
  getPurchaseReturnById,
  submitPurchaseReturn,
  approvePurchaseReturn,
  rejectPurchaseReturn,
  completePurchaseReturn,
  shipPurchaseReturn,
  cancelPurchaseReturn,
} from '@/api/purchase-returns/purchase-returns'
import type { HandlerPurchaseReturnResponse, HandlerPurchaseReturnItemResponse } from '@/api/models'
import { useI18n } from '@/hooks/useI18n'
import { useFormatters } from '@/hooks/useFormatters'
import { safeToFixed } from '@/utils'
import './PurchaseReturnDetail.css'

const { Text } = Typography

// Status variant mapping for DetailPageHeader
const STATUS_VARIANTS: Record<
  string,
  'default' | 'primary' | 'success' | 'warning' | 'danger' | 'info'
> = {
  DRAFT: 'default',
  PENDING: 'warning',
  APPROVED: 'info',
  REJECTED: 'danger',
  SHIPPED: 'primary',
  COMPLETED: 'success',
  CANCELLED: 'danger',
}

// Status key mapping for i18n
const STATUS_KEYS: Record<string, string> = {
  DRAFT: 'draft',
  PENDING: 'pendingApproval',
  APPROVED: 'approved',
  REJECTED: 'rejected',
  SHIPPED: 'shipped',
  COMPLETED: 'completed',
  CANCELLED: 'cancelled',
}

/**
 * Purchase Return Detail Page
 *
 * Features:
 * - Display complete return information using DetailPageHeader
 * - Display return status flow using StatusFlow component
 * - Display return line items
 * - Status action buttons (submit, approve, reject, ship, complete, cancel)
 */
export default function PurchaseReturnDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { t } = useI18n({ ns: 'trade' })
  const { formatCurrency, formatDateTime } = useFormatters()

  const [returnData, setReturnData] = useState<HandlerPurchaseReturnResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState(false)

  // Modal states
  const [approveModalVisible, setApproveModalVisible] = useState(false)
  const [rejectModalVisible, setRejectModalVisible] = useState(false)
  const [cancelModalVisible, setCancelModalVisible] = useState(false)
  const [shipModalVisible, setShipModalVisible] = useState(false)
  const [approvalNote, setApprovalNote] = useState('')
  const [rejectReason, setRejectReason] = useState('')
  const [cancelReason, setCancelReason] = useState('')
  const [shippingNote, setShippingNote] = useState('')

  // Fetch return details
  const fetchReturn = useCallback(async () => {
    if (!id) return

    setLoading(true)
    try {
      const response = await getPurchaseReturnById(id)
      if (response.status === 200 && response.data.success && response.data.data) {
        setReturnData(response.data.data)
      } else {
        Toast.error(t('purchaseReturnDetail.messages.notExist'))
        navigate('/trade/purchase-returns')
      }
    } catch {
      Toast.error(t('purchaseReturnDetail.messages.fetchDetailError'))
      navigate('/trade/purchase-returns')
    } finally {
      setLoading(false)
    }
  }, [id, navigate, t])

  useEffect(() => {
    fetchReturn()
  }, [fetchReturn])

  // Handle submit for approval
  const handleSubmit = useCallback(async () => {
    if (!returnData?.id) return
    Modal.confirm({
      title: t('purchaseReturn.modal.submitTitle'),
      content: t('purchaseReturn.modal.submitContent', { returnNumber: returnData.return_number }),
      okText: t('salesOrder.modal.confirmOk'),
      cancelText: t('salesOrder.modal.cancelBtn'),
      onOk: async () => {
        setActionLoading(true)
        try {
          await submitPurchaseReturn(returnData.id!, {})
          Toast.success(t('purchaseReturn.messages.submitSuccess'))
          fetchReturn()
        } catch {
          Toast.error(t('purchaseReturn.messages.submitError'))
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [returnData, fetchReturn, t])

  // Handle approve return
  const handleApprove = useCallback(async () => {
    if (!returnData?.id) return
    setActionLoading(true)
    try {
      await approvePurchaseReturn(returnData.id, { note: approvalNote })
      Toast.success(t('purchaseReturn.messages.approveSuccess'))
      setApproveModalVisible(false)
      setApprovalNote('')
      fetchReturn()
    } catch {
      Toast.error(t('purchaseReturn.messages.approveError'))
    } finally {
      setActionLoading(false)
    }
  }, [returnData, fetchReturn, approvalNote, t])

  // Handle reject return
  const handleReject = useCallback(async () => {
    if (!returnData?.id) return
    if (!rejectReason.trim()) {
      Toast.warning(t('purchaseReturnDetail.messages.rejectReasonRequired'))
      return
    }
    setActionLoading(true)
    try {
      await rejectPurchaseReturn(returnData.id, { reason: rejectReason })
      Toast.success(t('purchaseReturn.messages.rejectSuccess'))
      setRejectModalVisible(false)
      setRejectReason('')
      fetchReturn()
    } catch {
      Toast.error(t('purchaseReturn.messages.rejectError'))
    } finally {
      setActionLoading(false)
    }
  }, [returnData, fetchReturn, rejectReason, t])

  // Handle ship return (send goods back to supplier)
  const handleShip = useCallback(async () => {
    if (!returnData?.id) return
    setActionLoading(true)
    try {
      await shipPurchaseReturn(returnData.id, { note: shippingNote })
      Toast.success(t('purchaseReturn.messages.shipSuccess'))
      setShipModalVisible(false)
      setShippingNote('')
      fetchReturn()
    } catch {
      Toast.error(t('purchaseReturn.messages.shipError'))
    } finally {
      setActionLoading(false)
    }
  }, [returnData, fetchReturn, shippingNote, t])

  // Handle complete return
  const handleComplete = useCallback(async () => {
    if (!returnData?.id) return
    Modal.confirm({
      title: t('purchaseReturn.modal.completeTitle'),
      content: t('purchaseReturn.modal.completeContent', {
        returnNumber: returnData.return_number,
      }),
      okText: t('salesOrder.modal.confirmOk'),
      cancelText: t('salesOrder.modal.cancelBtn'),
      onOk: async () => {
        setActionLoading(true)
        try {
          await completePurchaseReturn(returnData.id!, {})
          Toast.success(t('purchaseReturn.messages.completeSuccess'))
          fetchReturn()
        } catch {
          Toast.error(t('purchaseReturn.messages.completeError'))
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [returnData, fetchReturn, t])

  // Handle cancel return
  const handleCancel = useCallback(async () => {
    if (!returnData?.id) return
    setActionLoading(true)
    try {
      await cancelPurchaseReturn(returnData.id, {
        reason: cancelReason || t('common.userCancel'),
      })
      Toast.success(t('purchaseReturn.messages.cancelSuccess'))
      setCancelModalVisible(false)
      setCancelReason('')
      fetchReturn()
    } catch {
      Toast.error(t('purchaseReturn.messages.cancelError'))
    } finally {
      setActionLoading(false)
    }
  }, [returnData, fetchReturn, cancelReason, t])

  // Handle edit return
  const handleEdit = useCallback(() => {
    if (returnData?.id) {
      navigate(`/trade/purchase-returns/${returnData.id}/edit`)
    }
  }, [returnData, navigate])

  // Build status for DetailPageHeader
  const headerStatus = useMemo((): DetailPageHeaderStatus | undefined => {
    if (!returnData) return undefined
    const status = returnData.status || 'DRAFT'
    const statusKey = STATUS_KEYS[status] || 'draft'
    return {
      label: t(`purchaseReturn.status.${statusKey}`),
      variant: STATUS_VARIANTS[status] || 'default',
    }
  }, [returnData, t])

  // Build metrics for DetailPageHeader
  const headerMetrics = useMemo((): DetailPageHeaderMetric[] => {
    if (!returnData) return []
    return [
      {
        label: t('purchaseReturnDetail.basicInfo.supplierName'),
        value: returnData.supplier_name || '-',
      },
      {
        label: t('purchaseReturnDetail.basicInfo.orderNumber'),
        value: returnData.purchase_order_number || '-',
      },
      {
        label: t('purchaseReturnDetail.basicInfo.itemCount'),
        value: `${returnData.item_count || 0} ${t('salesOrder.unit')}`,
      },
      {
        label: t('purchaseReturnDetail.amount.totalRefund'),
        value: formatCurrency(returnData.total_refund || 0),
        variant: 'primary',
      },
    ]
  }, [returnData, t, formatCurrency])

  // Build primary action for DetailPageHeader
  const primaryAction = useMemo((): DetailPageHeaderAction | undefined => {
    if (!returnData) return undefined
    const status = returnData.status || 'DRAFT'

    if (status === 'DRAFT') {
      return {
        key: 'submit',
        label: t('purchaseReturn.actions.submit'),
        icon: <IconSend />,
        type: 'primary',
        onClick: handleSubmit,
        loading: actionLoading,
      }
    }
    if (status === 'PENDING') {
      return {
        key: 'approve',
        label: t('purchaseReturn.actions.approve'),
        icon: <IconTick />,
        type: 'primary',
        onClick: () => setApproveModalVisible(true),
        loading: actionLoading,
      }
    }
    if (status === 'APPROVED') {
      return {
        key: 'ship',
        label: t('purchaseReturn.actions.ship'),
        icon: <IconBox />,
        type: 'primary',
        onClick: () => setShipModalVisible(true),
        loading: actionLoading,
      }
    }
    if (status === 'SHIPPED') {
      return {
        key: 'complete',
        label: t('purchaseReturn.actions.complete'),
        icon: <IconTick />,
        type: 'primary',
        onClick: handleComplete,
        loading: actionLoading,
      }
    }
    return undefined
  }, [returnData, t, handleSubmit, handleComplete, actionLoading])

  // Build secondary actions for DetailPageHeader
  const secondaryActions = useMemo((): DetailPageHeaderAction[] => {
    if (!returnData) return []
    const status = returnData.status || 'DRAFT'
    const actions: DetailPageHeaderAction[] = []

    // Print button is always available
    actions.push({
      key: 'print',
      label: t('purchaseReturn.actions.print'),
      icon: <IconPrint />,
      onClick: () => {
        const printBtn = document.querySelector('.detail-page-print-btn') as HTMLButtonElement
        printBtn?.click()
      },
    })

    if (status === 'DRAFT') {
      actions.push({
        key: 'edit',
        label: t('purchaseReturn.actions.edit'),
        icon: <IconEdit />,
        onClick: handleEdit,
        disabled: actionLoading,
      })
      actions.push({
        key: 'cancel',
        label: t('purchaseReturn.actions.cancel'),
        icon: <IconClose />,
        type: 'danger',
        onClick: () => setCancelModalVisible(true),
        loading: actionLoading,
      })
    }

    if (status === 'PENDING') {
      actions.push({
        key: 'reject',
        label: t('purchaseReturn.actions.reject'),
        icon: <IconClose />,
        type: 'warning',
        onClick: () => setRejectModalVisible(true),
        loading: actionLoading,
      })
      actions.push({
        key: 'cancel',
        label: t('purchaseReturn.actions.cancel'),
        icon: <IconClose />,
        type: 'danger',
        onClick: () => setCancelModalVisible(true),
        loading: actionLoading,
      })
    }

    if (status === 'APPROVED') {
      actions.push({
        key: 'cancel',
        label: t('purchaseReturn.actions.cancel'),
        icon: <IconClose />,
        type: 'danger',
        onClick: () => setCancelModalVisible(true),
        loading: actionLoading,
      })
    }

    return actions
  }, [returnData, t, handleEdit, actionLoading])

  // Build status flow steps
  const statusFlowSteps = useMemo((): StatusFlowStep[] => {
    if (!returnData) return []
    const status = returnData.status || 'DRAFT'
    const isCancelled = status === 'CANCELLED'
    const isRejected = status === 'REJECTED'

    // Define the normal flow
    const steps: StatusFlowStep[] = [
      {
        key: 'draft',
        label: t('purchaseReturn.status.draft'),
        state: 'completed',
        timestamp: returnData.created_at ? formatDateTime(returnData.created_at) : undefined,
      },
      {
        key: 'pending',
        label: t('purchaseReturn.status.pendingApproval'),
        state: returnData.submitted_at
          ? 'completed'
          : status === 'PENDING'
            ? 'current'
            : isCancelled || isRejected
              ? 'cancelled'
              : 'pending',
        timestamp: returnData.submitted_at ? formatDateTime(returnData.submitted_at) : undefined,
      },
      {
        key: 'approved',
        label: t('purchaseReturn.status.approved'),
        state: returnData.approved_at
          ? 'completed'
          : status === 'APPROVED'
            ? 'current'
            : isCancelled || isRejected
              ? 'cancelled'
              : 'pending',
        timestamp: returnData.approved_at ? formatDateTime(returnData.approved_at) : undefined,
      },
      {
        key: 'shipped',
        label: t('purchaseReturn.status.shipped'),
        state: returnData.shipped_at
          ? 'completed'
          : status === 'SHIPPED'
            ? 'current'
            : isCancelled || isRejected
              ? 'cancelled'
              : 'pending',
        timestamp: returnData.shipped_at ? formatDateTime(returnData.shipped_at) : undefined,
      },
      {
        key: 'completed',
        label: t('purchaseReturn.status.completed'),
        state: returnData.completed_at
          ? 'completed'
          : status === 'COMPLETED'
            ? 'current'
            : isCancelled || isRejected
              ? 'cancelled'
              : 'pending',
        timestamp: returnData.completed_at ? formatDateTime(returnData.completed_at) : undefined,
      },
    ]

    // Update current step based on actual status
    if (status === 'DRAFT') {
      steps[0].state = 'current'
      steps[1].state = 'pending'
    }

    // If rejected, add rejected step
    if (isRejected) {
      steps.push({
        key: 'rejected',
        label: t('purchaseReturn.status.rejected'),
        state: 'rejected',
        timestamp: returnData.rejected_at ? formatDateTime(returnData.rejected_at) : undefined,
        description: returnData.rejection_reason || undefined,
      })
    }

    // If cancelled, add cancelled step at the end
    if (isCancelled) {
      steps.push({
        key: 'cancelled',
        label: t('purchaseReturn.status.cancelled'),
        state: 'rejected',
        timestamp: returnData.cancelled_at ? formatDateTime(returnData.cancelled_at) : undefined,
        description: returnData.cancel_reason || undefined,
      })
    }

    return steps
  }, [returnData, t, formatDateTime])

  // Return items table columns
  const itemColumns = useMemo(
    () => [
      {
        title: t('purchaseReturnDetail.items.columns.index'),
        dataIndex: 'index',
        width: 60,
        render: (_: unknown, __: unknown, index: number) => index + 1,
      },
      {
        title: t('purchaseReturnDetail.items.columns.productCode'),
        dataIndex: 'product_code',
        width: 120,
        render: (code: string) => <Text className="product-code">{code || '-'}</Text>,
      },
      {
        title: t('purchaseReturnDetail.items.columns.productName'),
        dataIndex: 'product_name',
        width: 200,
        ellipsis: true,
      },
      {
        title: t('purchaseReturnDetail.items.columns.unit'),
        dataIndex: 'unit',
        width: 80,
        align: 'center' as const,
        render: (unit: string) => unit || '-',
      },
      {
        title: t('purchaseReturnDetail.items.columns.originalQuantity'),
        dataIndex: 'original_quantity',
        width: 100,
        align: 'right' as const,
        render: (qty: number) => safeToFixed(qty, 2, '-'),
      },
      {
        title: t('purchaseReturnDetail.items.columns.returnQuantity'),
        dataIndex: 'return_quantity',
        width: 100,
        align: 'right' as const,
        render: (qty: number) => (
          <Text className="return-quantity">{safeToFixed(qty, 2, '-')}</Text>
        ),
      },
      {
        title: t('purchaseReturnDetail.items.columns.unitCost'),
        dataIndex: 'unit_cost',
        width: 120,
        align: 'right' as const,
        render: (cost: number) => formatCurrency(cost),
      },
      {
        title: t('purchaseReturnDetail.items.columns.refundAmount'),
        dataIndex: 'refund_amount',
        width: 120,
        align: 'right' as const,
        render: (amount: number) => <Text className="refund-amount">{formatCurrency(amount)}</Text>,
      },
      {
        title: t('purchaseReturnDetail.items.columns.reason'),
        dataIndex: 'reason',
        ellipsis: true,
        render: (reason: string) => reason || '-',
      },
    ],
    [t, formatCurrency]
  )

  // Render return basic info
  const renderBasicInfo = () => {
    if (!returnData) return null

    return (
      <div className="order-info-grid">
        <div className="order-info-item">
          <Text type="secondary" className="order-info-label">
            {t('purchaseReturnDetail.basicInfo.returnNumber')}
          </Text>
          <Text strong className="order-info-value order-number-value">
            {returnData.return_number}
          </Text>
        </div>
        <div className="order-info-item">
          <Text type="secondary" className="order-info-label">
            {t('purchaseReturnDetail.basicInfo.orderNumber')}
          </Text>
          <Text className="order-info-value order-number-value">
            {returnData.purchase_order_number || '-'}
          </Text>
        </div>
        <div className="order-info-item">
          <Text type="secondary" className="order-info-label">
            {t('purchaseReturnDetail.basicInfo.supplierName')}
          </Text>
          <Text className="order-info-value">{returnData.supplier_name || '-'}</Text>
        </div>
        <div className="order-info-item">
          <Text type="secondary" className="order-info-label">
            {t('purchaseReturnDetail.basicInfo.totalQuantity')}
          </Text>
          <Text className="order-info-value">
            {safeToFixed(returnData.total_quantity, 2, '0.00')}
          </Text>
        </div>
        <div className="order-info-item">
          <Text type="secondary" className="order-info-label">
            {t('purchaseReturnDetail.basicInfo.createdAt')}
          </Text>
          <Text className="order-info-value">
            {returnData.created_at ? formatDateTime(returnData.created_at) : '-'}
          </Text>
        </div>
        <div className="order-info-item">
          <Text type="secondary" className="order-info-label">
            {t('purchaseReturnDetail.basicInfo.reason')}
          </Text>
          <Text className="order-info-value">{returnData.reason || '-'}</Text>
        </div>
        {returnData.tracking_number && (
          <div className="order-info-item">
            <Text type="secondary" className="order-info-label">
              {t('purchaseReturnDetail.basicInfo.trackingNumber')}
            </Text>
            <Text className="order-info-value order-number-value">
              {returnData.tracking_number}
            </Text>
          </div>
        )}
        <div className="order-info-item order-info-item--full">
          <Text type="secondary" className="order-info-label">
            {t('purchaseReturnDetail.basicInfo.remark')}
          </Text>
          <Text className="order-info-value">{returnData.remark || '-'}</Text>
        </div>
      </div>
    )
  }

  // Render amount summary
  const renderAmountSummary = () => {
    if (!returnData) return null

    return (
      <div className="amount-summary">
        <div className="amount-row total-row">
          <Text strong>{t('purchaseReturnDetail.amount.totalRefund')}</Text>
          <Text className="refund-total" strong>
            {formatCurrency(returnData.total_refund || 0)}
          </Text>
        </div>
      </div>
    )
  }

  if (loading) {
    return (
      <Container size="lg" className="purchase-return-detail-page">
        <DetailPageHeader
          title={t('purchaseReturnDetail.title')}
          loading={true}
          showBack={true}
          onBack={() => navigate('/trade/purchase-returns')}
          backLabel={t('purchaseReturnDetail.back')}
        />
      </Container>
    )
  }

  if (!returnData) {
    return (
      <Container size="lg" className="purchase-return-detail-page">
        <Empty
          title={t('purchaseReturnDetail.notExist')}
          description={t('purchaseReturnDetail.notExistDesc')}
        />
      </Container>
    )
  }

  return (
    <Container size="lg" className="purchase-return-detail-page">
      {/* Unified Header */}
      <DetailPageHeader
        title={t('purchaseReturnDetail.title')}
        documentNumber={returnData.return_number}
        status={headerStatus}
        metrics={headerMetrics}
        primaryAction={primaryAction}
        secondaryActions={secondaryActions}
        onBack={() => navigate('/trade/purchase-returns')}
        backLabel={t('purchaseReturnDetail.back')}
      />

      {/* Hidden Print Button for secondary action click */}
      <div style={{ display: 'none' }}>
        <PrintButton
          documentType="PURCHASE_RETURN"
          documentId={returnData.id || ''}
          documentNumber={returnData.return_number || ''}
          label={t('purchaseReturn.actions.print')}
          enableShortcut={true}
          className="detail-page-print-btn"
        />
      </div>

      {/* Status Flow */}
      <Card className="status-flow-card" title={t('purchaseReturnDetail.timeline.title')}>
        <StatusFlow
          steps={statusFlowSteps}
          showTimestamp
          ariaLabel={t('purchaseReturnDetail.timeline.title')}
        />
      </Card>

      {/* Return Info Card */}
      <Card className="info-card" title={t('purchaseReturnDetail.basicInfo.title')}>
        {renderBasicInfo()}
      </Card>

      {/* Return Items Card */}
      <Card className="items-card" title={t('purchaseReturnDetail.items.title')}>
        <Table
          columns={itemColumns}
          dataSource={
            (returnData.items || []) as (HandlerPurchaseReturnItemResponse &
              Record<string, unknown>)[]
          }
          rowKey="id"
          pagination={false}
          size="small"
          empty={<Empty description={t('purchaseReturnDetail.items.empty')} />}
        />
        {renderAmountSummary()}
      </Card>

      {/* Approve Modal */}
      <Modal
        title={t('purchaseReturn.modal.approveTitle')}
        visible={approveModalVisible}
        onOk={handleApprove}
        onCancel={() => {
          setApproveModalVisible(false)
          setApprovalNote('')
        }}
        okText={t('purchaseReturn.actions.approve')}
        cancelText={t('salesOrder.modal.cancelBtn')}
        confirmLoading={actionLoading}
      >
        <div className="modal-content">
          <Text>
            {t('purchaseReturn.modal.approveContent', { returnNumber: returnData.return_number })}
          </Text>
          <TextArea
            placeholder={t('purchaseReturnDetail.modal.approvalNotePlaceholder')}
            value={approvalNote}
            onChange={setApprovalNote}
            rows={3}
            style={{ marginTop: 16 }}
          />
        </div>
      </Modal>

      {/* Reject Modal */}
      <Modal
        title={t('purchaseReturn.modal.rejectTitle')}
        visible={rejectModalVisible}
        onOk={handleReject}
        onCancel={() => {
          setRejectModalVisible(false)
          setRejectReason('')
        }}
        okText={t('purchaseReturn.actions.reject')}
        cancelText={t('salesOrder.modal.cancelBtn')}
        okButtonProps={{ type: 'danger' }}
        confirmLoading={actionLoading}
      >
        <div className="modal-content">
          <Text>
            {t('purchaseReturn.modal.rejectContent', { returnNumber: returnData.return_number })}
          </Text>
          <TextArea
            placeholder={t('purchaseReturnDetail.modal.rejectReasonPlaceholder')}
            value={rejectReason}
            onChange={setRejectReason}
            rows={3}
            style={{ marginTop: 16 }}
          />
        </div>
      </Modal>

      {/* Ship Modal */}
      <Modal
        title={t('purchaseReturn.modal.shipTitle')}
        visible={shipModalVisible}
        onOk={handleShip}
        onCancel={() => {
          setShipModalVisible(false)
          setShippingNote('')
        }}
        okText={t('purchaseReturn.actions.ship')}
        cancelText={t('salesOrder.modal.cancelBtn')}
        confirmLoading={actionLoading}
      >
        <div className="modal-content">
          <Text>
            {t('purchaseReturn.modal.shipContent', { returnNumber: returnData.return_number })}
          </Text>
          <TextArea
            placeholder={t('purchaseReturnDetail.modal.shippingNotePlaceholder')}
            value={shippingNote}
            onChange={setShippingNote}
            rows={3}
            style={{ marginTop: 16 }}
          />
        </div>
      </Modal>

      {/* Cancel Modal */}
      <Modal
        title={t('purchaseReturn.modal.cancelTitle')}
        visible={cancelModalVisible}
        onOk={handleCancel}
        onCancel={() => {
          setCancelModalVisible(false)
          setCancelReason('')
        }}
        okText={t('salesOrder.modal.cancelOk')}
        cancelText={t('salesOrder.modal.backBtn')}
        okButtonProps={{ type: 'danger' }}
        confirmLoading={actionLoading}
      >
        <div className="modal-content">
          <Text>
            {t('purchaseReturn.modal.cancelContent', { returnNumber: returnData.return_number })}
          </Text>
          <TextArea
            placeholder={t('purchaseReturnDetail.modal.cancelReasonPlaceholder')}
            value={cancelReason}
            onChange={setCancelReason}
            rows={3}
            style={{ marginTop: 16 }}
          />
        </div>
      </Modal>
    </Container>
  )
}
