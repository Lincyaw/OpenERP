import { useState, useEffect, useCallback, useMemo } from 'react'
import { Card, Typography, Table, Tag, Toast, Button, Space, Empty } from '@douyinfe/semi-ui-19'
import { IconLink, IconRefresh } from '@douyinfe/semi-icons'
import { useParams, useNavigate } from 'react-router-dom'
import { Container } from '@/components/common/layout'
import {
  DetailPageHeader,
  type DetailPageHeaderAction,
  type DetailPageHeaderStatus,
  type DetailPageHeaderMetric,
} from '@/components/common'
import { getFinanceReceivableReceivableByID } from '@/api/finance-receivables/finance-receivables'
import type { HandlerAccountReceivableResponse, HandlerPaymentRecordResponse } from '@/api/models'
import { useTranslation } from 'react-i18next'
import './ReceivableDetail.css'

const { Text } = Typography

// Status type
type AccountReceivableStatus = 'PENDING' | 'PARTIAL' | 'PAID' | 'REVERSED' | 'CANCELLED'

// Status variant mapping for DetailPageHeader
const STATUS_VARIANTS: Record<
  AccountReceivableStatus,
  'default' | 'warning' | 'primary' | 'success' | 'danger'
> = {
  PENDING: 'warning',
  PARTIAL: 'primary',
  PAID: 'success',
  REVERSED: 'danger',
  CANCELLED: 'default',
}

// Status tag color mapping (for inline usage)
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
 * - Display complete receivable information using DetailPageHeader
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

  // Build status for DetailPageHeader
  const headerStatus = useMemo((): DetailPageHeaderStatus | undefined => {
    if (!receivableData?.status) return undefined
    const status = receivableData.status as AccountReceivableStatus
    const overdueFlag = isOverdue(receivableData)

    if (overdueFlag) {
      return {
        label: `${t(`receivables.status.${status}`)} - ${t('receivables.tooltip.overdue')}`,
        variant: 'danger',
      }
    }

    return {
      label: t(`receivables.status.${status}`),
      variant: STATUS_VARIANTS[status] || 'default',
    }
  }, [receivableData, t])

  // Build metrics for DetailPageHeader
  const headerMetrics = useMemo((): DetailPageHeaderMetric[] => {
    if (!receivableData) return []
    const overdueFlag = isOverdue(receivableData)
    return [
      {
        label: t('receivableDetail.amount.totalAmount'),
        value: formatCurrency(receivableData.total_amount),
      },
      {
        label: t('receivableDetail.amount.paidAmount'),
        value: formatCurrency(receivableData.paid_amount),
        variant: 'success',
      },
      {
        label: t('receivableDetail.amount.outstandingAmount'),
        value: formatCurrency(receivableData.outstanding_amount),
        variant: overdueFlag ? 'danger' : 'warning',
      },
      {
        label: t('receivableDetail.basicInfo.dueDate'),
        value: formatDate(receivableData.due_date),
        variant: overdueFlag ? 'danger' : 'default',
      },
    ]
  }, [receivableData, t])

  // Build primary action for DetailPageHeader
  const primaryAction = useMemo((): DetailPageHeaderAction | undefined => {
    if (!receivableData) return undefined
    const status = receivableData.status
    const canCollect = status !== 'PAID' && status !== 'CANCELLED' && status !== 'REVERSED'

    if (canCollect) {
      return {
        key: 'collect',
        label: t('receivables.actions.collect'),
        type: 'primary',
        onClick: handleCollect,
      }
    }
    return undefined
  }, [receivableData, t, handleCollect])

  // Build secondary actions for DetailPageHeader
  const secondaryActions = useMemo((): DetailPageHeaderAction[] => {
    return [
      {
        key: 'refresh',
        label: t('receivableDetail.actions.refresh'),
        icon: <IconRefresh />,
        onClick: fetchReceivable,
        disabled: loading,
      },
    ]
  }, [t, fetchReceivable, loading])

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

    return (
      <div className="info-grid">
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('receivableDetail.basicInfo.receivableNumber')}
          </Text>
          <Text strong className="info-value code-value">
            {receivableData.receivable_number || '-'}
          </Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('receivableDetail.basicInfo.customerName')}
          </Text>
          <Text className="info-value">{receivableData.customer_name || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('receivableDetail.basicInfo.sourceDocument')}
          </Text>
          <div className="info-value">
            {receivableData.source_number ? (
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
            )}
          </div>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('receivableDetail.basicInfo.sourceType')}
          </Text>
          <Text className="info-value">
            {receivableData.source_type
              ? String(t(`receivables.sourceType.${receivableData.source_type}`))
              : '-'}
          </Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('receivableDetail.basicInfo.status')}
          </Text>
          <div className="info-value">
            {status ? (
              <Tag color={STATUS_TAG_COLORS[status]}>{t(`receivables.status.${status}`)}</Tag>
            ) : (
              '-'
            )}
          </div>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('receivableDetail.basicInfo.dueDate')}
          </Text>
          <div className="info-value">
            <span className={overdueFlag ? 'overdue-date' : ''}>
              {formatDate(receivableData.due_date)}
              {overdueFlag && (
                <Tag color="red" style={{ marginLeft: 8 }}>
                  {t('receivables.tooltip.overdue')}
                </Tag>
              )}
            </span>
          </div>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('receivableDetail.basicInfo.createdAt')}
          </Text>
          <Text className="info-value">{formatDateTime(receivableData.created_at)}</Text>
        </div>
        <div className="info-item info-item--full">
          <Text type="secondary" className="info-label">
            {t('receivableDetail.basicInfo.remark')}
          </Text>
          <Text className="info-value">{receivableData.remark || '-'}</Text>
        </div>
      </div>
    )
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

  if (loading) {
    return (
      <Container size="lg" className="receivable-detail-page">
        <DetailPageHeader
          title={t('receivableDetail.title')}
          loading={true}
          showBack={true}
          onBack={() => navigate('/finance/receivables')}
          backLabel={t('receivableDetail.back')}
        />
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

  return (
    <Container size="lg" className="receivable-detail-page">
      {/* Unified Header */}
      <DetailPageHeader
        title={t('receivableDetail.title')}
        documentNumber={receivableData.receivable_number}
        status={headerStatus}
        metrics={headerMetrics}
        primaryAction={primaryAction}
        secondaryActions={secondaryActions}
        onBack={() => navigate('/finance/receivables')}
        backLabel={t('receivableDetail.back')}
      />

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
