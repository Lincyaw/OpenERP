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
import { getFinanceReceivableReceivableByID } from '@/api/finance-receivables/finance-receivables'
import type { HandlerAccountReceivableResponse, HandlerPaymentRecordResponse } from '@/api/models'
import { useTranslation } from 'react-i18next'
import './ReceivableDetail.css'

const { Title, Text } = Typography

// Status type
type AccountReceivableStatus = 'PENDING' | 'PARTIAL' | 'PAID' | 'REVERSED' | 'CANCELLED'

// Status tag color mapping
const STATUS_TAG_COLORS: Record<
  AccountReceivableStatus,
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
 * Check if a receivable is overdue
 */
function isOverdue(receivable: HandlerAccountReceivableResponse): boolean {
  if (!receivable.due_date) return false
  if (receivable.status === 'PAID' || receivable.status === 'CANCELLED') return false
  return new Date(receivable.due_date) < new Date()
}

/**
 * Receivable Detail Page
 *
 * Features:
 * - Display complete receivable information
 * - Display payment history records
 * - Navigate to source document
 * - Navigate to create receipt voucher
 */
export default function ReceivableDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { t } = useTranslation('finance')

  const [receivableData, setReceivableData] = useState<HandlerAccountReceivableResponse | null>(
    null
  )
  const [loading, setLoading] = useState(true)

  // Fetch receivable details
  const fetchReceivable = useCallback(async () => {
    if (!id) return

    setLoading(true)
    try {
      const response = await getFinanceReceivableReceivableByID(id)
      if (response.status === 200 && response.data.success && response.data.data) {
        setReceivableData(response.data.data)
      } else {
        Toast.error(t('receivableDetail.messages.notExist'))
        navigate('/finance/receivables')
      }
    } catch {
      Toast.error(t('receivableDetail.messages.fetchError'))
      navigate('/finance/receivables')
    } finally {
      setLoading(false)
    }
  }, [id, navigate, t])

  useEffect(() => {
    fetchReceivable()
  }, [fetchReceivable])

  // Handle navigate to source document
  const handleViewSource = useCallback(() => {
    if (!receivableData?.source_id || !receivableData?.source_type) return

    switch (receivableData.source_type) {
      case 'SALES_ORDER':
        navigate(`/trade/sales/${receivableData.source_id}`)
        break
      case 'SALES_RETURN':
        navigate(`/trade/sales-returns/${receivableData.source_id}`)
        break
      default:
        Toast.info(t('receivableDetail.messages.sourceNotNavigable'))
    }
  }, [receivableData, navigate, t])

  // Handle create receipt voucher
  const handleCollect = useCallback(() => {
    if (receivableData?.id && receivableData?.customer_id) {
      navigate(
        `/finance/receipts/new?receivable_id=${receivableData.id}&customer_id=${receivableData.customer_id}`
      )
    }
  }, [receivableData, navigate])

  // Payment records table columns
  const paymentColumns = useMemo(
    () => [
      {
        title: t('receivableDetail.paymentRecords.columns.index'),
        dataIndex: 'index',
        width: 60,
        render: (_: unknown, __: unknown, index: number) => index + 1,
      },
      {
        title: t('receivableDetail.paymentRecords.columns.amount'),
        dataIndex: 'amount',
        width: 150,
        align: 'right' as const,
        render: (amount: number) => (
          <Text className="payment-amount">{formatCurrency(amount)}</Text>
        ),
      },
      {
        title: t('receivableDetail.paymentRecords.columns.appliedAt'),
        dataIndex: 'applied_at',
        width: 180,
        render: (date: string) => formatDateTime(date),
      },
      {
        title: t('receivableDetail.paymentRecords.columns.remark'),
        dataIndex: 'remark',
        ellipsis: true,
        render: (remark: string) => remark || '-',
      },
    ],
    [t]
  )

  // Render basic info
  const renderBasicInfo = () => {
    if (!receivableData) return null

    const status = receivableData.status as AccountReceivableStatus | undefined
    const overdueFlag = isOverdue(receivableData)

    const data = [
      {
        key: t('receivableDetail.basicInfo.receivableNumber'),
        value: receivableData.receivable_number || '-',
      },
      {
        key: t('receivableDetail.basicInfo.customerName'),
        value: receivableData.customer_name || '-',
      },
      {
        key: t('receivableDetail.basicInfo.sourceDocument'),
        value: receivableData.source_number ? (
          <Space>
            <Text>{receivableData.source_number}</Text>
            {receivableData.source_type && receivableData.source_type !== 'MANUAL' && (
              <Button
                size="small"
                icon={<IconLink />}
                theme="borderless"
                onClick={handleViewSource}
              >
                {t('receivableDetail.basicInfo.viewSource')}
              </Button>
            )}
          </Space>
        ) : (
          '-'
        ),
      },
      {
        key: t('receivableDetail.basicInfo.sourceType'),
        value: receivableData.source_type
          ? String(t(`receivables.sourceType.${receivableData.source_type}`))
          : '-',
      },
      {
        key: t('receivableDetail.basicInfo.status'),
        value: status ? (
          <Tag color={STATUS_TAG_COLORS[status]}>{t(`receivables.status.${status}`)}</Tag>
        ) : (
          '-'
        ),
      },
      {
        key: t('receivableDetail.basicInfo.dueDate'),
        value: (
          <span className={overdueFlag ? 'overdue-date' : ''}>
            {formatDate(receivableData.due_date)}
            {overdueFlag && (
              <Tag color="red" style={{ marginLeft: 8 }}>
                {t('receivables.tooltip.overdue')}
              </Tag>
            )}
          </span>
        ),
      },
      {
        key: t('receivableDetail.basicInfo.createdAt'),
        value: formatDateTime(receivableData.created_at),
      },
      {
        key: t('receivableDetail.basicInfo.remark'),
        value: receivableData.remark || '-',
      },
    ]

    return <Descriptions data={data} row className="receivable-basic-info" />
  }

  // Render amount summary
  const renderAmountSummary = () => {
    if (!receivableData) return null

    return (
      <div className="amount-summary">
        <div className="amount-row">
          <Text type="tertiary">{t('receivableDetail.amount.totalAmount')}</Text>
          <Text className="total-amount" strong>
            {formatCurrency(receivableData.total_amount)}
          </Text>
        </div>
        <div className="amount-row">
          <Text type="tertiary">{t('receivableDetail.amount.paidAmount')}</Text>
          <Text className="paid-amount">{formatCurrency(receivableData.paid_amount)}</Text>
        </div>
        <div className="amount-row outstanding-row">
          <Text strong>{t('receivableDetail.amount.outstandingAmount')}</Text>
          <Text
            className={`outstanding-amount ${isOverdue(receivableData) ? 'overdue' : ''}`}
            strong
          >
            {formatCurrency(receivableData.outstanding_amount)}
          </Text>
        </div>
      </div>
    )
  }

  // Render action buttons
  const renderActions = () => {
    if (!receivableData) return null

    const status = receivableData.status
    const canCollect = status !== 'PAID' && status !== 'CANCELLED' && status !== 'REVERSED'

    return (
      <Space>
        <Button icon={<IconRefresh />} onClick={fetchReceivable} disabled={loading}>
          {t('receivableDetail.actions.refresh')}
        </Button>
        {canCollect && (
          <Button type="primary" onClick={handleCollect}>
            {t('receivables.actions.collect')}
          </Button>
        )}
      </Space>
    )
  }

  if (loading) {
    return (
      <Container size="lg" className="receivable-detail-page">
        <div className="loading-container">
          <Spin size="large" />
        </div>
      </Container>
    )
  }

  if (!receivableData) {
    return (
      <Container size="lg" className="receivable-detail-page">
        <Empty
          title={t('receivableDetail.notExist')}
          description={t('receivableDetail.notExistDesc')}
        />
      </Container>
    )
  }

  const status = receivableData.status as AccountReceivableStatus | undefined

  return (
    <Container size="lg" className="receivable-detail-page">
      {/* Header */}
      <div className="page-header">
        <div className="header-left">
          <Button
            icon={<IconArrowLeft />}
            theme="borderless"
            onClick={() => navigate('/finance/receivables')}
          >
            {t('receivableDetail.back')}
          </Button>
          <Title heading={4} className="page-title">
            {t('receivableDetail.title')}
          </Title>
          {status && (
            <Tag color={STATUS_TAG_COLORS[status]} size="large">
              {t(`receivables.status.${status}`)}
            </Tag>
          )}
        </div>
        <div className="header-right">{renderActions()}</div>
      </div>

      {/* Basic Info Card */}
      <Card className="info-card" title={t('receivableDetail.basicInfo.title')}>
        {renderBasicInfo()}
      </Card>

      {/* Amount Card */}
      <Card className="amount-card" title={t('receivableDetail.amount.title')}>
        {renderAmountSummary()}
      </Card>

      {/* Payment Records Card */}
      <Card className="payment-records-card" title={t('receivableDetail.paymentRecords.title')}>
        <Table
          columns={paymentColumns}
          dataSource={
            (receivableData.payment_records || []) as (HandlerPaymentRecordResponse &
              Record<string, unknown>)[]
          }
          rowKey="id"
          pagination={false}
          size="small"
          empty={<Empty description={t('receivableDetail.paymentRecords.empty')} />}
        />
      </Card>
    </Container>
  )
}
