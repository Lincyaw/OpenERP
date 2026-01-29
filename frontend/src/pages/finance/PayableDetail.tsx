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
  Empty,
} from '@douyinfe/semi-ui-19'
import { IconArrowLeft, IconLink, IconRefresh } from '@douyinfe/semi-icons'
import { useParams, useNavigate } from 'react-router-dom'
import { Container } from '@/components/common/layout'
import { getFinancePayablePayableByID } from '@/api/finance-payables/finance-payables'
import type {
  HandlerAccountPayableResponse,
  HandlerPayablePaymentRecordResponse,
} from '@/api/models'
import { useTranslation } from 'react-i18next'
import './PayableDetail.css'

const { Title, Text } = Typography

// Status type
type AccountPayableStatus = 'PENDING' | 'PARTIAL' | 'PAID' | 'REVERSED' | 'CANCELLED'

// Status tag color mapping
const STATUS_TAG_COLORS: Record<
  AccountPayableStatus,
  'orange' | 'blue' | 'green' | 'red' | 'grey'
> = {
  PENDING: 'orange',
  PARTIAL: 'blue',
  PAID: 'green',
  REVERSED: 'red',
  CANCELLED: 'grey',
}

/**
 * Format currency for display
 */
function formatCurrency(amount?: number): string {
  if (amount === undefined || amount === null) return '-'
  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: 'CNY',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(amount)
}

/**
 * Format date for display
 */
function formatDate(dateStr?: string): string {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  })
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
  })
}

/**
 * Check if a payable is overdue
 */
function isOverdue(payable: HandlerAccountPayableResponse): boolean {
  if (!payable.due_date) return false
  if (payable.status === 'PAID' || payable.status === 'CANCELLED') return false
  return new Date(payable.due_date) < new Date()
}

/**
 * Payable Detail Page
 *
 * Features:
 * - Display complete payable information
 * - Display payment history records
 * - Navigate to source document
 * - Navigate to create payment voucher
 */
export default function PayableDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { t } = useTranslation('finance')

  const [payableData, setPayableData] = useState<HandlerAccountPayableResponse | null>(null)
  const [loading, setLoading] = useState(true)

  // Fetch payable details
  const fetchPayable = useCallback(async () => {
    if (!id) return

    setLoading(true)
    try {
      const response = await getFinancePayablePayableByID(id)
      if (response.status === 200 && response.data.success && response.data.data) {
        setPayableData(response.data.data)
      } else {
        Toast.error(t('payableDetail.messages.notExist'))
        navigate('/finance/payables')
      }
    } catch {
      Toast.error(t('payableDetail.messages.fetchError'))
      navigate('/finance/payables')
    } finally {
      setLoading(false)
    }
  }, [id, navigate, t])

  useEffect(() => {
    fetchPayable()
  }, [fetchPayable])

  // Handle navigate to source document
  const handleViewSource = useCallback(() => {
    if (!payableData?.source_id || !payableData?.source_type) return

    switch (payableData.source_type) {
      case 'PURCHASE_ORDER':
        navigate(`/trade/purchase/${payableData.source_id}`)
        break
      case 'PURCHASE_RETURN':
        navigate(`/trade/purchase-returns/${payableData.source_id}`)
        break
      default:
        Toast.info(t('payableDetail.messages.sourceNotNavigable'))
    }
  }, [payableData, navigate, t])

  // Handle create payment voucher
  const handlePay = useCallback(() => {
    if (payableData?.id && payableData?.supplier_id) {
      navigate(
        `/finance/payments/new?payable_id=${payableData.id}&supplier_id=${payableData.supplier_id}`
      )
    }
  }, [payableData, navigate])

  // Payment records table columns
  const paymentColumns = useMemo(
    () => [
      {
        title: t('payableDetail.paymentRecords.columns.index'),
        dataIndex: 'index',
        width: 60,
        render: (_: unknown, __: unknown, index: number) => index + 1,
      },
      {
        title: t('payableDetail.paymentRecords.columns.amount'),
        dataIndex: 'amount',
        width: 150,
        align: 'right' as const,
        render: (amount: number) => (
          <Text className="payment-amount">{formatCurrency(amount)}</Text>
        ),
      },
      {
        title: t('payableDetail.paymentRecords.columns.appliedAt'),
        dataIndex: 'applied_at',
        width: 180,
        render: (date: string) => formatDateTime(date),
      },
      {
        title: t('payableDetail.paymentRecords.columns.remark'),
        dataIndex: 'remark',
        ellipsis: true,
        render: (remark: string) => remark || '-',
      },
    ],
    [t]
  )

  // Render basic info
  const renderBasicInfo = () => {
    if (!payableData) return null

    const status = payableData.status as AccountPayableStatus | undefined
    const overdueFlag = isOverdue(payableData)

    const data = [
      {
        key: t('payableDetail.basicInfo.payableNumber'),
        value: payableData.payable_number || '-',
      },
      {
        key: t('payableDetail.basicInfo.supplierName'),
        value: payableData.supplier_name || '-',
      },
      {
        key: t('payableDetail.basicInfo.sourceDocument'),
        value: payableData.source_number ? (
          <Space>
            <Text>{payableData.source_number}</Text>
            {payableData.source_type && payableData.source_type !== 'MANUAL' && (
              <Button
                size="small"
                icon={<IconLink />}
                theme="borderless"
                onClick={handleViewSource}
              >
                {t('payableDetail.basicInfo.viewSource')}
              </Button>
            )}
          </Space>
        ) : (
          '-'
        ),
      },
      {
        key: t('payableDetail.basicInfo.sourceType'),
        value: payableData.source_type
          ? String(t(`payables.sourceType.${payableData.source_type}`))
          : '-',
      },
      {
        key: t('payableDetail.basicInfo.status'),
        value: status ? (
          <Tag color={STATUS_TAG_COLORS[status]}>{t(`payables.status.${status}`)}</Tag>
        ) : (
          '-'
        ),
      },
      {
        key: t('payableDetail.basicInfo.dueDate'),
        value: (
          <span className={overdueFlag ? 'overdue-date' : ''}>
            {formatDate(payableData.due_date)}
            {overdueFlag && (
              <Tag color="red" style={{ marginLeft: 8 }}>
                {t('payables.tooltip.overdue')}
              </Tag>
            )}
          </span>
        ),
      },
      {
        key: t('payableDetail.basicInfo.createdAt'),
        value: formatDateTime(payableData.created_at),
      },
      {
        key: t('payableDetail.basicInfo.remark'),
        value: payableData.remark || '-',
      },
    ]

    return <Descriptions data={data} row className="payable-basic-info" />
  }

  // Render amount summary
  const renderAmountSummary = () => {
    if (!payableData) return null

    return (
      <div className="amount-summary">
        <div className="amount-row">
          <Text type="tertiary">{t('payableDetail.amount.totalAmount')}</Text>
          <Text className="total-amount" strong>
            {formatCurrency(payableData.total_amount)}
          </Text>
        </div>
        <div className="amount-row">
          <Text type="tertiary">{t('payableDetail.amount.paidAmount')}</Text>
          <Text className="paid-amount">{formatCurrency(payableData.paid_amount)}</Text>
        </div>
        <div className="amount-row outstanding-row">
          <Text strong>{t('payableDetail.amount.outstandingAmount')}</Text>
          <Text className={`outstanding-amount ${isOverdue(payableData) ? 'overdue' : ''}`} strong>
            {formatCurrency(payableData.outstanding_amount)}
          </Text>
        </div>
      </div>
    )
  }

  // Render action buttons
  const renderActions = () => {
    if (!payableData) return null

    const status = payableData.status
    const canPay = status !== 'PAID' && status !== 'CANCELLED' && status !== 'REVERSED'

    return (
      <Space>
        <Button icon={<IconRefresh />} onClick={fetchPayable} disabled={loading}>
          {t('payableDetail.actions.refresh')}
        </Button>
        {canPay && (
          <Button type="primary" onClick={handlePay}>
            {t('payables.actions.pay')}
          </Button>
        )}
      </Space>
    )
  }

  if (loading) {
    return (
      <Container size="lg" className="payable-detail-page">
        <div className="loading-container">
          <Spin size="large" />
        </div>
      </Container>
    )
  }

  if (!payableData) {
    return (
      <Container size="lg" className="payable-detail-page">
        <Empty title={t('payableDetail.notExist')} description={t('payableDetail.notExistDesc')} />
      </Container>
    )
  }

  const status = payableData.status as AccountPayableStatus | undefined

  return (
    <Container size="lg" className="payable-detail-page">
      {/* Header */}
      <div className="page-header">
        <div className="header-left">
          <Button
            icon={<IconArrowLeft />}
            theme="borderless"
            onClick={() => navigate('/finance/payables')}
          >
            {t('payableDetail.back')}
          </Button>
          <Title heading={4} className="page-title">
            {t('payableDetail.title')}
          </Title>
          {status && (
            <Tag color={STATUS_TAG_COLORS[status]} size="large">
              {t(`payables.status.${status}`)}
            </Tag>
          )}
        </div>
        <div className="header-right">{renderActions()}</div>
      </div>

      {/* Basic Info Card */}
      <Card className="info-card" title={t('payableDetail.basicInfo.title')}>
        {renderBasicInfo()}
      </Card>

      {/* Amount Card */}
      <Card className="amount-card" title={t('payableDetail.amount.title')}>
        {renderAmountSummary()}
      </Card>

      {/* Payment Records Card */}
      <Card className="payment-records-card" title={t('payableDetail.paymentRecords.title')}>
        <Table
          columns={paymentColumns}
          dataSource={
            (payableData.payment_records || []) as (HandlerPayablePaymentRecordResponse &
              Record<string, unknown>)[]
          }
          rowKey="id"
          pagination={false}
          size="small"
          empty={<Empty description={t('payableDetail.paymentRecords.empty')} />}
        />
      </Card>
    </Container>
  )
}
