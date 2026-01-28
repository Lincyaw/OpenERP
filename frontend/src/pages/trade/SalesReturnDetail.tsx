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
  TextArea,
} from '@douyinfe/semi-ui-19'
import { IconArrowLeft, IconEdit, IconTick, IconClose, IconSend } from '@douyinfe/semi-icons'
import { useParams, useNavigate } from 'react-router-dom'
import { Container } from '@/components/common/layout'
import { PrintButton } from '@/components/printing'
import { getSalesReturns } from '@/api/sales-returns/sales-returns'
import type { HandlerSalesReturnResponse, HandlerSalesReturnItemResponse } from '@/api/models'
import { useI18n } from '@/hooks/useI18n'
import { useFormatters } from '@/hooks/useFormatters'
import { safeToFixed } from '@/utils'
import './SalesReturnDetail.css'

const { Title, Text } = Typography

// Status tag color mapping
const STATUS_TAG_COLORS: Record<
  string,
  'blue' | 'cyan' | 'green' | 'grey' | 'red' | 'orange' | 'amber' | 'light-blue'
> = {
  DRAFT: 'blue',
  PENDING: 'orange',
  APPROVED: 'cyan',
  RECEIVING: 'light-blue',
  REJECTED: 'red',
  COMPLETED: 'green',
  CANCELLED: 'grey',
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
 * - Display complete return information
 * - Display return line items
 * - Display status change timeline
 * - Status action buttons (submit, approve, reject, complete, cancel)
 */
export default function SalesReturnDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { t } = useI18n({ ns: 'trade' })
  const { formatCurrency, formatDateTime } = useFormatters()
  const salesReturnApi = useMemo(() => getSalesReturns(), [])

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
      const response = await salesReturnApi.getSalesReturnById(id)
      if (response.success && response.data) {
        setReturnData(response.data)
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
  }, [id, salesReturnApi, navigate, t])

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
          await salesReturnApi.submitSalesReturn(returnData.id!)
          Toast.success(t('salesReturn.messages.submitSuccess'))
          fetchReturn()
        } catch {
          Toast.error(t('salesReturn.messages.submitError'))
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [returnData, salesReturnApi, fetchReturn, t])

  // Handle approve return
  const handleApprove = useCallback(async () => {
    if (!returnData?.id) return
    setActionLoading(true)
    try {
      await salesReturnApi.approveSalesReturn(returnData.id, { note: approvalNote })
      Toast.success(t('salesReturn.messages.approveSuccess'))
      setApproveModalVisible(false)
      setApprovalNote('')
      fetchReturn()
    } catch {
      Toast.error(t('salesReturn.messages.approveError'))
    } finally {
      setActionLoading(false)
    }
  }, [returnData, salesReturnApi, fetchReturn, approvalNote, t])

  // Handle reject return
  const handleReject = useCallback(async () => {
    if (!returnData?.id) return
    if (!rejectReason.trim()) {
      Toast.warning(t('salesReturnDetail.messages.rejectReasonRequired'))
      return
    }
    setActionLoading(true)
    try {
      await salesReturnApi.rejectSalesReturn(returnData.id, { reason: rejectReason })
      Toast.success(t('salesReturn.messages.rejectSuccess'))
      setRejectModalVisible(false)
      setRejectReason('')
      fetchReturn()
    } catch {
      Toast.error(t('salesReturn.messages.rejectError'))
    } finally {
      setActionLoading(false)
    }
  }, [returnData, salesReturnApi, fetchReturn, rejectReason, t])

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
          await salesReturnApi.completeSalesReturn(returnData.id!, {})
          Toast.success(t('salesReturn.messages.completeSuccess'))
          fetchReturn()
        } catch {
          Toast.error(t('salesReturn.messages.completeError'))
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [returnData, salesReturnApi, fetchReturn, t])

  // Handle receive returned goods (start receiving)
  const handleReceive = useCallback(async () => {
    if (!returnData?.id) return
    setActionLoading(true)
    try {
      await salesReturnApi.receiveSalesReturn(returnData.id!, {})
      Toast.success(t('salesReturn.messages.receiveSuccess'))
      fetchReturn()
    } catch {
      Toast.error(t('salesReturn.messages.receiveError'))
    } finally {
      setActionLoading(false)
    }
  }, [returnData, salesReturnApi, fetchReturn, t])

  // Handle cancel return
  const handleCancel = useCallback(async () => {
    if (!returnData?.id) return
    setActionLoading(true)
    try {
      await salesReturnApi.cancelSalesReturn(returnData.id, {
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
  }, [returnData, salesReturnApi, fetchReturn, cancelReason, t])

  // Handle edit return
  const handleEdit = useCallback(() => {
    if (returnData?.id) {
      navigate(`/trade/sales-returns/${returnData.id}/edit`)
    }
  }, [returnData, navigate])

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

  // Build timeline items based on return status
  const timelineItems = useMemo(() => {
    if (!returnData) return []

    const items = []

    // Created
    if (returnData.created_at) {
      items.push({
        time: formatDateTime(returnData.created_at),
        content: t('salesReturnDetail.timeline.created'),
        type: 'default' as const,
      })
    }

    // Submitted
    if (returnData.submitted_at) {
      items.push({
        time: formatDateTime(returnData.submitted_at),
        content: t('salesReturnDetail.timeline.submitted'),
        type: 'ongoing' as const,
      })
    }

    // Approved
    if (returnData.approved_at) {
      items.push({
        time: formatDateTime(returnData.approved_at),
        content: `${t('salesReturnDetail.timeline.approved')}${returnData.approval_note ? `: ${returnData.approval_note}` : ''}`,
        type: 'success' as const,
      })
    }

    // Receiving
    if (returnData.received_at) {
      items.push({
        time: formatDateTime(returnData.received_at),
        content: t('salesReturnDetail.timeline.receiving'),
        type: 'ongoing' as const,
      })
    }

    // Rejected
    if (returnData.rejected_at) {
      items.push({
        time: formatDateTime(returnData.rejected_at),
        content: `${t('salesReturnDetail.timeline.rejected')}${returnData.rejection_reason ? `: ${returnData.rejection_reason}` : ''}`,
        type: 'error' as const,
      })
    }

    // Completed
    if (returnData.completed_at) {
      items.push({
        time: formatDateTime(returnData.completed_at),
        content: t('salesReturnDetail.timeline.completed'),
        type: 'success' as const,
      })
    }

    // Cancelled
    if (returnData.cancelled_at) {
      items.push({
        time: formatDateTime(returnData.cancelled_at),
        content: `${t('salesReturnDetail.timeline.cancelled')}${returnData.cancel_reason ? `: ${returnData.cancel_reason}` : ''}`,
        type: 'error' as const,
      })
    }

    return items
  }, [returnData, t, formatDateTime])

  // Render return basic info
  const renderBasicInfo = () => {
    if (!returnData) return null

    const statusKey = STATUS_KEYS[returnData.status || 'DRAFT'] || 'draft'

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
        key: t('salesReturnDetail.basicInfo.returnStatus'),
        value: (
          <Tag color={STATUS_TAG_COLORS[returnData.status || 'DRAFT']}>
            {t(`salesReturn.status.${statusKey}`)}
          </Tag>
        ),
      },
      {
        key: t('salesReturnDetail.basicInfo.itemCount'),
        value: `${returnData.item_count || 0} ${t('salesOrder.unit')}`,
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

  // Render action buttons based on status
  const renderActions = () => {
    if (!returnData) return null

    const status = returnData.status || 'DRAFT'

    return (
      <Space>
        {/* Print button - always available */}
        <PrintButton
          documentType="SALES_RETURN"
          documentId={returnData.id || ''}
          documentNumber={returnData.return_number || ''}
          label={t('salesReturn.actions.print')}
          enableShortcut={true}
        />
        {status === 'DRAFT' && (
          <>
            <Button icon={<IconEdit />} onClick={handleEdit} disabled={actionLoading}>
              {t('salesReturn.actions.edit')}
            </Button>
            <Button
              type="primary"
              icon={<IconSend />}
              onClick={handleSubmit}
              loading={actionLoading}
            >
              {t('salesReturn.actions.submit')}
            </Button>
            <Button
              type="danger"
              icon={<IconClose />}
              onClick={() => setCancelModalVisible(true)}
              loading={actionLoading}
            >
              {t('salesReturn.actions.cancel')}
            </Button>
          </>
        )}
        {status === 'PENDING' && (
          <>
            <Button
              type="primary"
              icon={<IconTick />}
              onClick={() => setApproveModalVisible(true)}
              loading={actionLoading}
            >
              {t('salesReturn.actions.approve')}
            </Button>
            <Button
              type="warning"
              icon={<IconClose />}
              onClick={() => setRejectModalVisible(true)}
              loading={actionLoading}
            >
              {t('salesReturn.actions.reject')}
            </Button>
            <Button
              type="danger"
              icon={<IconClose />}
              onClick={() => setCancelModalVisible(true)}
              loading={actionLoading}
            >
              {t('salesReturn.actions.cancel')}
            </Button>
          </>
        )}
        {status === 'APPROVED' && (
          <>
            <Button
              type="primary"
              icon={<IconTick />}
              onClick={handleReceive}
              loading={actionLoading}
            >
              {t('salesReturn.actions.receive')}
            </Button>
            <Button
              type="danger"
              icon={<IconClose />}
              onClick={() => setCancelModalVisible(true)}
              loading={actionLoading}
            >
              {t('salesReturn.actions.cancel')}
            </Button>
          </>
        )}
        {status === 'RECEIVING' && (
          <>
            <Button
              type="primary"
              icon={<IconTick />}
              onClick={handleComplete}
              loading={actionLoading}
            >
              {t('salesReturn.actions.complete')}
            </Button>
            <Button
              type="danger"
              icon={<IconClose />}
              onClick={() => setCancelModalVisible(true)}
              loading={actionLoading}
            >
              {t('salesReturn.actions.cancel')}
            </Button>
          </>
        )}
      </Space>
    )
  }

  if (loading) {
    return (
      <Container size="lg" className="sales-return-detail-page">
        <div className="loading-container">
          <Spin size="large" />
        </div>
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

  const statusKey = STATUS_KEYS[returnData.status || 'DRAFT'] || 'draft'

  return (
    <Container size="lg" className="sales-return-detail-page">
      {/* Header */}
      <div className="page-header">
        <div className="header-left">
          <Button
            icon={<IconArrowLeft />}
            theme="borderless"
            onClick={() => navigate('/trade/sales-returns')}
          >
            {t('salesReturnDetail.back')}
          </Button>
          <Title heading={4} className="page-title">
            {t('salesReturnDetail.title')}
          </Title>
          <Tag color={STATUS_TAG_COLORS[returnData.status || 'DRAFT']} size="large">
            {t(`salesReturn.status.${statusKey}`)}
          </Tag>
        </div>
        <div className="header-right">{renderActions()}</div>
      </div>

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

      {/* Timeline Card */}
      <Card className="timeline-card" title={t('salesReturnDetail.timeline.title')}>
        {timelineItems.length > 0 ? (
          <Timeline mode="left" className="status-timeline">
            {timelineItems.map((item, index) => (
              <Timeline.Item
                key={index}
                time={item.time}
                type={item.type as 'default' | 'success' | 'warning' | 'error' | 'ongoing'}
              >
                {item.content}
              </Timeline.Item>
            ))}
          </Timeline>
        ) : (
          <Empty description={t('salesReturnDetail.timeline.empty')} />
        )}
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
