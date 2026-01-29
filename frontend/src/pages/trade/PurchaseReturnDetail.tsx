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
import {
  IconArrowLeft,
  IconEdit,
  IconTick,
  IconClose,
  IconSend,
  IconBox,
} from '@douyinfe/semi-icons'
import { useParams, useNavigate } from 'react-router-dom'
import { Container } from '@/components/common/layout'
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

const { Title, Text } = Typography

// Status tag color mapping
const STATUS_TAG_COLORS: Record<
  string,
  'blue' | 'cyan' | 'green' | 'grey' | 'red' | 'orange' | 'amber' | 'violet'
> = {
  DRAFT: 'blue',
  PENDING: 'orange',
  APPROVED: 'cyan',
  REJECTED: 'red',
  SHIPPED: 'violet',
  COMPLETED: 'green',
  CANCELLED: 'grey',
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
 * - Display complete return information
 * - Display return line items
 * - Display status change timeline
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

  // Build timeline items based on return status
  const timelineItems = useMemo(() => {
    if (!returnData) return []

    const items = []

    // Created
    if (returnData.created_at) {
      items.push({
        time: formatDateTime(returnData.created_at),
        content: t('purchaseReturnDetail.timeline.created'),
        type: 'default' as const,
      })
    }

    // Submitted
    if (returnData.submitted_at) {
      items.push({
        time: formatDateTime(returnData.submitted_at),
        content: t('purchaseReturnDetail.timeline.submitted'),
        type: 'ongoing' as const,
      })
    }

    // Approved
    if (returnData.approved_at) {
      items.push({
        time: formatDateTime(returnData.approved_at),
        content: `${t('purchaseReturnDetail.timeline.approved')}${returnData.approval_note ? `: ${returnData.approval_note}` : ''}`,
        type: 'success' as const,
      })
    }

    // Rejected
    if (returnData.rejected_at) {
      items.push({
        time: formatDateTime(returnData.rejected_at),
        content: `${t('purchaseReturnDetail.timeline.rejected')}${returnData.rejection_reason ? `: ${returnData.rejection_reason}` : ''}`,
        type: 'error' as const,
      })
    }

    // Shipped
    if (returnData.shipped_at) {
      items.push({
        time: formatDateTime(returnData.shipped_at),
        content: `${t('purchaseReturnDetail.timeline.shipped')}${returnData.shipping_note ? `: ${returnData.shipping_note}` : ''}`,
        type: 'ongoing' as const,
      })
    }

    // Completed
    if (returnData.completed_at) {
      items.push({
        time: formatDateTime(returnData.completed_at),
        content: t('purchaseReturnDetail.timeline.completed'),
        type: 'success' as const,
      })
    }

    // Cancelled
    if (returnData.cancelled_at) {
      items.push({
        time: formatDateTime(returnData.cancelled_at),
        content: `${t('purchaseReturnDetail.timeline.cancelled')}${returnData.cancel_reason ? `: ${returnData.cancel_reason}` : ''}`,
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
      { key: t('purchaseReturnDetail.basicInfo.returnNumber'), value: returnData.return_number },
      {
        key: t('purchaseReturnDetail.basicInfo.orderNumber'),
        value: returnData.purchase_order_number || '-',
      },
      {
        key: t('purchaseReturnDetail.basicInfo.supplierName'),
        value: returnData.supplier_name || '-',
      },
      {
        key: t('purchaseReturnDetail.basicInfo.returnStatus'),
        value: (
          <Tag color={STATUS_TAG_COLORS[returnData.status || 'DRAFT']}>
            {t(`purchaseReturn.status.${statusKey}`)}
          </Tag>
        ),
      },
      {
        key: t('purchaseReturnDetail.basicInfo.itemCount'),
        value: `${returnData.item_count || 0} ${t('salesOrder.unit')}`,
      },
      {
        key: t('purchaseReturnDetail.basicInfo.totalQuantity'),
        value: safeToFixed(returnData.total_quantity, 2, '0.00'),
      },
      {
        key: t('purchaseReturnDetail.basicInfo.createdAt'),
        value: returnData.created_at ? formatDateTime(returnData.created_at) : '-',
      },
      { key: t('purchaseReturnDetail.basicInfo.reason'), value: returnData.reason || '-' },
      { key: t('purchaseReturnDetail.basicInfo.remark'), value: returnData.remark || '-' },
    ]

    // Add tracking number if shipped
    if (returnData.tracking_number) {
      data.push({
        key: t('purchaseReturnDetail.basicInfo.trackingNumber'),
        value: returnData.tracking_number,
      })
    }

    return <Descriptions data={data} row className="return-basic-info" />
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

  // Render action buttons based on status
  const renderActions = () => {
    if (!returnData) return null

    const status = returnData.status || 'DRAFT'

    return (
      <Space>
        {/* Print button - always available */}
        <PrintButton
          documentType="PURCHASE_RETURN"
          documentId={returnData.id || ''}
          documentNumber={returnData.return_number || ''}
          label={t('purchaseReturn.actions.print')}
          enableShortcut={true}
        />
        {status === 'DRAFT' && (
          <>
            <Button icon={<IconEdit />} onClick={handleEdit} disabled={actionLoading}>
              {t('purchaseReturn.actions.edit')}
            </Button>
            <Button
              type="primary"
              icon={<IconSend />}
              onClick={handleSubmit}
              loading={actionLoading}
            >
              {t('purchaseReturn.actions.submit')}
            </Button>
            <Button
              type="danger"
              icon={<IconClose />}
              onClick={() => setCancelModalVisible(true)}
              loading={actionLoading}
            >
              {t('purchaseReturn.actions.cancel')}
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
              {t('purchaseReturn.actions.approve')}
            </Button>
            <Button
              type="warning"
              icon={<IconClose />}
              onClick={() => setRejectModalVisible(true)}
              loading={actionLoading}
            >
              {t('purchaseReturn.actions.reject')}
            </Button>
            <Button
              type="danger"
              icon={<IconClose />}
              onClick={() => setCancelModalVisible(true)}
              loading={actionLoading}
            >
              {t('purchaseReturn.actions.cancel')}
            </Button>
          </>
        )}
        {status === 'APPROVED' && (
          <>
            <Button
              type="primary"
              icon={<IconBox />}
              onClick={() => setShipModalVisible(true)}
              loading={actionLoading}
            >
              {t('purchaseReturn.actions.ship')}
            </Button>
            <Button
              type="danger"
              icon={<IconClose />}
              onClick={() => setCancelModalVisible(true)}
              loading={actionLoading}
            >
              {t('purchaseReturn.actions.cancel')}
            </Button>
          </>
        )}
        {status === 'SHIPPED' && (
          <>
            <Button
              type="primary"
              icon={<IconTick />}
              onClick={handleComplete}
              loading={actionLoading}
            >
              {t('purchaseReturn.actions.complete')}
            </Button>
          </>
        )}
      </Space>
    )
  }

  if (loading) {
    return (
      <Container size="lg" className="purchase-return-detail-page">
        <div className="loading-container">
          <Spin size="large" />
        </div>
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

  const statusKey = STATUS_KEYS[returnData.status || 'DRAFT'] || 'draft'

  return (
    <Container size="lg" className="purchase-return-detail-page">
      {/* Header */}
      <div className="page-header">
        <div className="header-left">
          <Button
            icon={<IconArrowLeft />}
            theme="borderless"
            onClick={() => navigate('/trade/purchase-returns')}
          >
            {t('purchaseReturnDetail.back')}
          </Button>
          <Title heading={4} className="page-title">
            {t('purchaseReturnDetail.title')}
          </Title>
          <Tag color={STATUS_TAG_COLORS[returnData.status || 'DRAFT']} size="large">
            {t(`purchaseReturn.status.${statusKey}`)}
          </Tag>
        </div>
        <div className="header-right">{renderActions()}</div>
      </div>

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

      {/* Timeline Card */}
      <Card className="timeline-card" title={t('purchaseReturnDetail.timeline.title')}>
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
          <Empty description={t('purchaseReturnDetail.timeline.empty')} />
        )}
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
