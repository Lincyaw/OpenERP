import { useState, useEffect, useCallback, useMemo } from 'react'
import {
  Card,
  Typography,
  Descriptions,
  Table,
  Toast,
  Modal,
  Empty,
  TextArea,
} from '@douyinfe/semi-ui-19'
import { IconEdit, IconTick, IconClose, IconSend, IconPrinter } from '@douyinfe/semi-icons'
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
  getSalesReturnById,
  submitSalesReturn,
  approveSalesReturn,
  rejectSalesReturn,
  completeSalesReturn,
  receiveSalesReturn,
  cancelSalesReturn,
} from '@/api/sales-returns/sales-returns'
import type { HandlerSalesReturnResponse, HandlerSalesReturnItemResponse } from '@/api/models'
import { useI18n } from '@/hooks/useI18n'
import { useFormatters } from '@/hooks/useFormatters'
import { safeToFixed } from '@/utils'
import './SalesReturnDetail.css'

const { Text } = Typography

// Status variant mapping for DetailPageHeader
const STATUS_VARIANTS: Record<
  string,
  'default' | 'primary' | 'success' | 'warning' | 'danger' | 'info'
> = {
  DRAFT: 'default',
  PENDING: 'warning',
  APPROVED: 'info',
  RECEIVING: 'primary',
  REJECTED: 'danger',
  COMPLETED: 'success',
  CANCELLED: 'danger',
}

// Status key mapping for i18n
const STATUS_KEYS: Record<string, string> = {
  DRAFT: 'draft',
  PENDING: 'pendingApproval',
  APPROVED: 'approved',
  RECEIVING: 'receiving',
  REJECTED: 'rejected',
  COMPLETED: 'completed',
  CANCELLED: 'cancelled',
}

/**
 * Sales Return Detail Page
 *
 * Features:
 * - Display complete return information using DetailPageHeader
 * - Display return status flow using StatusFlow component
 * - Display return line items
 * - Status action buttons (submit, approve, reject, complete, cancel)
 */
export default function SalesReturnDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { t } = useI18n({ ns: 'trade' })
  const { formatCurrency, formatDateTime } = useFormatters()

  const [returnData, setReturnData] = useState<HandlerSalesReturnResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState(false)

  // Modal states
  const [approveModalVisible, setApproveModalVisible] = useState(false)
  const [rejectModalVisible, setRejectModalVisible] = useState(false)
  const [cancelModalVisible, setCancelModalVisible] = useState(false)
  const [approvalNote, setApprovalNote] = useState('')
  const [rejectReason, setRejectReason] = useState('')
  const [cancelReason, setCancelReason] = useState('')

  // Fetch return details
  const fetchReturn = useCallback(async () => {
    if (!id) return

    setLoading(true)
    try {
      const response = await getSalesReturnById(id)
      if (response.status === 200 && response.data.success && response.data.data) {
        setReturnData(response.data.data)
      } else {
        Toast.error(t('salesReturnDetail.messages.notExist'))
        navigate('/trade/sales-returns')
      }
    } catch {
      Toast.error(t('salesReturnDetail.messages.fetchDetailError'))
      navigate('/trade/sales-returns')
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
      title: t('salesReturn.modal.submitTitle'),
      content: t('salesReturn.modal.submitContent', { returnNumber: returnData.return_number }),
      okText: t('salesOrder.modal.confirmOk'),
      cancelText: t('salesOrder.modal.cancelBtn'),
      onOk: async () => {
        setActionLoading(true)
        try {
          await submitSalesReturn(returnData.id!, {})
          Toast.success(t('salesReturn.messages.submitSuccess'))
          fetchReturn()
        } catch {
          Toast.error(t('salesReturn.messages.submitError'))
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
      await approveSalesReturn(returnData.id, { note: approvalNote })
      Toast.success(t('salesReturn.messages.approveSuccess'))
      setApproveModalVisible(false)
      setApprovalNote('')
      fetchReturn()
    } catch {
      Toast.error(t('salesReturn.messages.approveError'))
    } finally {
      setActionLoading(false)
    }
  }, [returnData, fetchReturn, approvalNote, t])

  // Handle reject return
  const handleReject = useCallback(async () => {
    if (!returnData?.id) return
    if (!rejectReason.trim()) {
      Toast.warning(t('salesReturnDetail.messages.rejectReasonRequired'))
      return
    }
    setActionLoading(true)
    try {
      await rejectSalesReturn(returnData.id, { reason: rejectReason })
      Toast.success(t('salesReturn.messages.rejectSuccess'))
      setRejectModalVisible(false)
      setRejectReason('')
      fetchReturn()
    } catch {
      Toast.error(t('salesReturn.messages.rejectError'))
    } finally {
      setActionLoading(false)
    }
  }, [returnData, fetchReturn, rejectReason, t])

  // Handle complete return
  const handleComplete = useCallback(async () => {
    if (!returnData?.id) return
    Modal.confirm({
      title: t('salesReturn.modal.completeTitle'),
      content: t('salesReturn.modal.completeContent', { returnNumber: returnData.return_number }),
      okText: t('salesOrder.modal.confirmOk'),
      cancelText: t('salesOrder.modal.cancelBtn'),
      onOk: async () => {
        setActionLoading(true)
        try {
          await completeSalesReturn(returnData.id!, {})
          Toast.success(t('salesReturn.messages.completeSuccess'))
          fetchReturn()
        } catch {
          Toast.error(t('salesReturn.messages.completeError'))
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [returnData, fetchReturn, t])

  // Handle receive returned goods (start receiving)
  const handleReceive = useCallback(async () => {
    if (!returnData?.id) return
    setActionLoading(true)
    try {
      await receiveSalesReturn(returnData.id!, {})
      Toast.success(t('salesReturn.messages.receiveSuccess'))
      fetchReturn()
    } catch {
      Toast.error(t('salesReturn.messages.receiveError'))
    } finally {
      setActionLoading(false)
    }
  }, [returnData, fetchReturn, t])

  // Handle cancel return
  const handleCancel = useCallback(async () => {
    if (!returnData?.id) return
    setActionLoading(true)
    try {
      await cancelSalesReturn(returnData.id, {
        reason: cancelReason || t('common.userCancel'),
      })
      Toast.success(t('salesReturn.messages.cancelSuccess'))
      setCancelModalVisible(false)
      setCancelReason('')
      fetchReturn()
    } catch {
      Toast.error(t('salesReturn.messages.cancelError'))
    } finally {
      setActionLoading(false)
    }
  }, [returnData, fetchReturn, cancelReason, t])

  // Handle edit return
  const handleEdit = useCallback(() => {
    if (returnData?.id) {
      navigate(`/trade/sales-returns/${returnData.id}/edit`)
    }
  }, [returnData, navigate])

  // Build status for DetailPageHeader
  const headerStatus = useMemo((): DetailPageHeaderStatus | undefined => {
    if (!returnData) return undefined
    const status = returnData.status || 'DRAFT'
    const statusKey = STATUS_KEYS[status] || 'draft'
    return {
      label: t(`salesReturn.status.${statusKey}`),
      variant: STATUS_VARIANTS[status] || 'default',
    }
  }, [returnData, t])

  // Build metrics for DetailPageHeader
  const headerMetrics = useMemo((): DetailPageHeaderMetric[] => {
    if (!returnData) return []
    return [
      {
        label: t('salesReturnDetail.basicInfo.customerName'),
        value: returnData.customer_name || '-',
      },
      {
        label: t('salesReturnDetail.basicInfo.orderNumber'),
        value: returnData.sales_order_number || '-',
      },
      {
        label: t('salesReturnDetail.basicInfo.itemCount'),
        value: `${returnData.item_count || 0} ${t('salesOrder.unit')}`,
      },
      {
        label: t('salesReturnDetail.amount.totalRefund'),
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
        label: t('salesReturn.actions.submit'),
        icon: <IconSend />,
        type: 'primary',
        onClick: handleSubmit,
        loading: actionLoading,
      }
    }
    if (status === 'PENDING') {
      return {
        key: 'approve',
        label: t('salesReturn.actions.approve'),
        icon: <IconTick />,
        type: 'primary',
        onClick: () => setApproveModalVisible(true),
        loading: actionLoading,
      }
    }
    if (status === 'APPROVED') {
      return {
        key: 'receive',
        label: t('salesReturn.actions.receive'),
        icon: <IconTick />,
        type: 'primary',
        onClick: handleReceive,
        loading: actionLoading,
      }
    }
    if (status === 'RECEIVING') {
      return {
        key: 'complete',
        label: t('salesReturn.actions.complete'),
        icon: <IconTick />,
        type: 'primary',
        onClick: handleComplete,
        loading: actionLoading,
      }
    }
    return undefined
  }, [returnData, t, handleSubmit, handleReceive, handleComplete, actionLoading])

  // Build secondary actions for DetailPageHeader
  const secondaryActions = useMemo((): DetailPageHeaderAction[] => {
    if (!returnData) return []
    const status = returnData.status || 'DRAFT'
    const actions: DetailPageHeaderAction[] = []

    // Print button is always available
    actions.push({
      key: 'print',
      label: t('salesReturn.actions.print'),
      icon: <IconPrinter />,
      onClick: () => {
        const printBtn = document.querySelector('.detail-page-print-btn') as HTMLButtonElement
        printBtn?.click()
      },
    })

    if (status === 'DRAFT') {
      actions.push({
        key: 'edit',
        label: t('salesReturn.actions.edit'),
        icon: <IconEdit />,
        onClick: handleEdit,
        disabled: actionLoading,
      })
      actions.push({
        key: 'cancel',
        label: t('salesReturn.actions.cancel'),
        icon: <IconClose />,
        type: 'danger',
        onClick: () => setCancelModalVisible(true),
        loading: actionLoading,
      })
    }

    if (status === 'PENDING') {
      actions.push({
        key: 'reject',
        label: t('salesReturn.actions.reject'),
        icon: <IconClose />,
        type: 'warning',
        onClick: () => setRejectModalVisible(true),
        loading: actionLoading,
      })
      actions.push({
        key: 'cancel',
        label: t('salesReturn.actions.cancel'),
        icon: <IconClose />,
        type: 'danger',
        onClick: () => setCancelModalVisible(true),
        loading: actionLoading,
      })
    }

    if (status === 'APPROVED' || status === 'RECEIVING') {
      actions.push({
        key: 'cancel',
        label: t('salesReturn.actions.cancel'),
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
        label: t('salesReturn.status.draft'),
        state: 'completed',
        timestamp: returnData.created_at ? formatDateTime(returnData.created_at) : undefined,
      },
      {
        key: 'pending',
        label: t('salesReturn.status.pendingApproval'),
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
        label: t('salesReturn.status.approved'),
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
        key: 'receiving',
        label: t('salesReturn.status.receiving'),
        state: returnData.received_at
          ? 'completed'
          : status === 'RECEIVING'
            ? 'current'
            : isCancelled || isRejected
              ? 'cancelled'
              : 'pending',
        timestamp: returnData.received_at ? formatDateTime(returnData.received_at) : undefined,
      },
      {
        key: 'completed',
        label: t('salesReturn.status.completed'),
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
        label: t('salesReturn.status.rejected'),
        state: 'rejected',
        timestamp: returnData.rejected_at ? formatDateTime(returnData.rejected_at) : undefined,
        description: returnData.rejection_reason || undefined,
      })
    }

    // If cancelled, add cancelled step at the end
    if (isCancelled) {
      steps.push({
        key: 'cancelled',
        label: t('salesReturn.status.cancelled'),
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
        title: t('salesReturnDetail.items.columns.index'),
        dataIndex: 'index',
        width: 60,
        render: (_: unknown, __: unknown, index: number) => index + 1,
      },
      {
        title: t('salesReturnDetail.items.columns.productCode'),
        dataIndex: 'product_code',
        width: 120,
        render: (code: string) => <Text className="product-code">{code || '-'}</Text>,
      },
      {
        title: t('salesReturnDetail.items.columns.productName'),
        dataIndex: 'product_name',
        width: 200,
        ellipsis: true,
      },
      {
        title: t('salesReturnDetail.items.columns.unit'),
        dataIndex: 'unit',
        width: 80,
        align: 'center' as const,
        render: (unit: string) => unit || '-',
      },
      {
        title: t('salesReturnDetail.items.columns.originalQuantity'),
        dataIndex: 'original_quantity',
        width: 100,
        align: 'right' as const,
        render: (qty: number) => safeToFixed(qty, 2, '-'),
      },
      {
        title: t('salesReturnDetail.items.columns.returnQuantity'),
        dataIndex: 'return_quantity',
        width: 100,
        align: 'right' as const,
        render: (qty: number) => (
          <Text className="return-quantity">{safeToFixed(qty, 2, '-')}</Text>
        ),
      },
      {
        title: t('salesReturnDetail.items.columns.unitPrice'),
        dataIndex: 'unit_price',
        width: 120,
        align: 'right' as const,
        render: (price: number) => formatCurrency(price),
      },
      {
        title: t('salesReturnDetail.items.columns.refundAmount'),
        dataIndex: 'refund_amount',
        width: 120,
        align: 'right' as const,
        render: (amount: number) => <Text className="refund-amount">{formatCurrency(amount)}</Text>,
      },
      {
        title: t('salesReturnDetail.items.columns.reason'),
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

    const data = [
      { key: t('salesReturnDetail.basicInfo.returnNumber'), value: returnData.return_number },
      {
        key: t('salesReturnDetail.basicInfo.orderNumber'),
        value: returnData.sales_order_number || '-',
      },
      {
        key: t('salesReturnDetail.basicInfo.customerName'),
        value: returnData.customer_name || '-',
      },
      {
        key: t('salesReturnDetail.basicInfo.totalQuantity'),
        value: safeToFixed(returnData.total_quantity, 2, '0.00'),
      },
      {
        key: t('salesReturnDetail.basicInfo.createdAt'),
        value: returnData.created_at ? formatDateTime(returnData.created_at) : '-',
      },
      { key: t('salesReturnDetail.basicInfo.reason'), value: returnData.reason || '-' },
      { key: t('salesReturnDetail.basicInfo.remark'), value: returnData.remark || '-' },
    ]

    return <Descriptions data={data} row className="return-basic-info" />
  }

  // Render amount summary
  const renderAmountSummary = () => {
    if (!returnData) return null

    return (
      <div className="amount-summary">
        <div className="amount-row total-row">
          <Text strong>{t('salesReturnDetail.amount.totalRefund')}</Text>
          <Text className="refund-total" strong>
            {formatCurrency(returnData.total_refund || 0)}
          </Text>
        </div>
      </div>
    )
  }

  if (loading) {
    return (
      <Container size="lg" className="sales-return-detail-page">
        <DetailPageHeader
          title={t('salesReturnDetail.title')}
          loading={true}
          showBack={true}
          onBack={() => navigate('/trade/sales-returns')}
          backLabel={t('salesReturnDetail.back')}
        />
      </Container>
    )
  }

  if (!returnData) {
    return (
      <Container size="lg" className="sales-return-detail-page">
        <Empty
          title={t('salesReturnDetail.notExist')}
          description={t('salesReturnDetail.notExistDesc')}
        />
      </Container>
    )
  }

  return (
    <Container size="lg" className="sales-return-detail-page">
      {/* Unified Header */}
      <DetailPageHeader
        title={t('salesReturnDetail.title')}
        documentNumber={returnData.return_number}
        status={headerStatus}
        metrics={headerMetrics}
        primaryAction={primaryAction}
        secondaryActions={secondaryActions}
        onBack={() => navigate('/trade/sales-returns')}
        backLabel={t('salesReturnDetail.back')}
      />

      {/* Hidden Print Button for secondary action click */}
      <div style={{ display: 'none' }}>
        <PrintButton
          documentType="SALES_RETURN"
          documentId={returnData.id || ''}
          documentNumber={returnData.return_number || ''}
          label={t('salesReturn.actions.print')}
          enableShortcut={true}
          className="detail-page-print-btn"
        />
      </div>

      {/* Status Flow */}
      <Card className="status-flow-card" title={t('salesReturnDetail.timeline.title')}>
        <StatusFlow
          steps={statusFlowSteps}
          showTimestamp
          ariaLabel={t('salesReturnDetail.timeline.title')}
        />
      </Card>

      {/* Return Info Card */}
      <Card className="info-card" title={t('salesReturnDetail.basicInfo.title')}>
        {renderBasicInfo()}
      </Card>

      {/* Return Items Card */}
      <Card className="items-card" title={t('salesReturnDetail.items.title')}>
        <Table
          columns={itemColumns}
          dataSource={
            (returnData.items || []) as (HandlerSalesReturnItemResponse & Record<string, unknown>)[]
          }
          rowKey="id"
          pagination={false}
          size="small"
          empty={<Empty description={t('salesReturnDetail.items.empty')} />}
        />
        {renderAmountSummary()}
      </Card>

      {/* Approve Modal */}
      <Modal
        title={t('salesReturn.modal.approveTitle')}
        visible={approveModalVisible}
        onOk={handleApprove}
        onCancel={() => {
          setApproveModalVisible(false)
          setApprovalNote('')
        }}
        okText={t('salesReturn.actions.approve')}
        cancelText={t('salesOrder.modal.cancelBtn')}
        confirmLoading={actionLoading}
      >
        <div className="modal-content">
          <Text>
            {t('salesReturn.modal.approveContent', { returnNumber: returnData.return_number })}
          </Text>
          <TextArea
            placeholder={t('salesReturnDetail.modal.approvalNotePlaceholder')}
            value={approvalNote}
            onChange={setApprovalNote}
            rows={3}
            style={{ marginTop: 16 }}
          />
        </div>
      </Modal>

      {/* Reject Modal */}
      <Modal
        title={t('salesReturn.modal.rejectTitle')}
        visible={rejectModalVisible}
        onOk={handleReject}
        onCancel={() => {
          setRejectModalVisible(false)
          setRejectReason('')
        }}
        okText={t('salesReturn.actions.reject')}
        cancelText={t('salesOrder.modal.cancelBtn')}
        okButtonProps={{ type: 'danger' }}
        confirmLoading={actionLoading}
      >
        <div className="modal-content">
          <Text>
            {t('salesReturn.modal.rejectContent', { returnNumber: returnData.return_number })}
          </Text>
          <TextArea
            placeholder={t('salesReturnDetail.modal.rejectReasonPlaceholder')}
            value={rejectReason}
            onChange={setRejectReason}
            rows={3}
            style={{ marginTop: 16 }}
          />
        </div>
      </Modal>

      {/* Cancel Modal */}
      <Modal
        title={t('salesReturn.modal.cancelTitle')}
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
            {t('salesReturn.modal.cancelContent', { returnNumber: returnData.return_number })}
          </Text>
          <TextArea
            placeholder={t('salesReturnDetail.modal.cancelReasonPlaceholder')}
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
