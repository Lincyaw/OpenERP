import { useState, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Card,
  Typography,
  Button,
  Table,
  Tag,
  Skeleton,
  Banner,
  Empty,
  Toast,
  Popconfirm,
  Descriptions,
  Select,
  DatePicker,
} from '@douyinfe/semi-ui-19'
import {
  IconDownload,
  IconCreditCard,
  IconDelete,
  IconTick,
  IconCalendar,
  IconCoinMoneyStroked,
  IconRefresh,
} from '@douyinfe/semi-icons'
import type { TagColor } from '@douyinfe/semi-ui-19/lib/es/tag'
import type { ColumnProps } from '@douyinfe/semi-ui-19/lib/es/table'

import { Container } from '@/components/common/layout'
import { useUser } from '@/store'
import {
  useGetBillingHistory,
  useGetBillingSummary,
  useSetDefaultPaymentMethod,
  useDeletePaymentMethod,
  downloadInvoicePdf,
  type Invoice,
  type InvoiceStatus,
  type PaymentMethod,
  type GetBillingHistoryParams,
} from '@/api/billing'

import './BillingHistoryPage.css'

const { Title, Text } = Typography

/**
 * Status color mapping
 */
const STATUS_COLORS: Record<InvoiceStatus, TagColor> = {
  paid: 'green',
  pending: 'orange',
  overdue: 'red',
  cancelled: 'grey',
  refunded: 'blue',
}

/**
 * Payment method icon mapping
 */
const PAYMENT_METHOD_ICONS: Record<string, string> = {
  card: '💳',
  bank_transfer: '🏦',
  alipay: '🔵',
  wechat: '🟢',
}

/**
 * Billing History Page
 *
 * Displays billing history, invoices, payment methods,
 * and upcoming billing information.
 */
export default function BillingHistoryPage() {
  const { t } = useTranslation('system')
  const user = useUser()

  // Pagination and filter state
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [statusFilter, setStatusFilter] = useState<InvoiceStatus | undefined>(undefined)
  const [dateRange, setDateRange] = useState<[Date, Date] | undefined>(undefined)

  // Build query params
  const queryParams: GetBillingHistoryParams = useMemo(
    () => ({
      page,
      page_size: pageSize,
      status: statusFilter,
      start_date: dateRange?.[0]?.toISOString().split('T')[0],
      end_date: dateRange?.[1]?.toISOString().split('T')[0],
    }),
    [page, pageSize, statusFilter, dateRange]
  )

  // Fetch billing data
  const {
    data: historyResponse,
    isLoading: isHistoryLoading,
    isError: isHistoryError,
    refetch: refetchHistory,
  } = useGetBillingHistory(queryParams, {
    query: {
      enabled: !!user?.tenantId,
    },
  })

  const {
    data: summaryResponse,
    isLoading: isSummaryLoading,
    isError: isSummaryError,
    refetch: refetchSummary,
  } = useGetBillingSummary({
    query: {
      enabled: !!user?.tenantId,
    },
  })

  // Mutations
  const setDefaultMutation = useSetDefaultPaymentMethod()
  const deleteMutation = useDeletePaymentMethod()

  // Extract data
  const historyData = historyResponse?.status === 200 ? historyResponse.data.data : null
  const summaryData = summaryResponse?.status === 200 ? summaryResponse.data.data : null

  // Download invoice handler
  const handleDownloadInvoice = useCallback(
    async (invoice: Invoice) => {
      const result = await downloadInvoicePdf(invoice.id, invoice.invoice_number)
      if (result.success) {
        Toast.success(t('billing.messages.downloadSuccess'))
      } else {
        Toast.error(t('billing.messages.downloadError'))
      }
    },
    [t]
  )

  // Set default payment method handler
  const handleSetDefault = useCallback(
    async (paymentMethodId: string) => {
      try {
        await setDefaultMutation.mutateAsync({ payment_method_id: paymentMethodId })
        Toast.success(t('billing.messages.setDefaultSuccess'))
      } catch {
        Toast.error(t('billing.messages.setDefaultError'))
      }
    },
    [setDefaultMutation, t]
  )

  // Delete payment method handler
  const handleDeletePaymentMethod = useCallback(
    async (paymentMethodId: string) => {
      try {
        await deleteMutation.mutateAsync(paymentMethodId)
        Toast.success(t('billing.messages.deleteSuccess'))
      } catch {
        Toast.error(t('billing.messages.deleteError'))
      }
    },
    [deleteMutation, t]
  )

  // Refresh all data
  const handleRefresh = useCallback(() => {
    refetchHistory()
    refetchSummary()
  }, [refetchHistory, refetchSummary])

  // Table columns
  const columns: ColumnProps<Invoice>[] = useMemo(
    () => [
      {
        title: t('billing.columns.invoiceNumber'),
        dataIndex: 'invoice_number',
        key: 'invoice_number',
        width: 150,
      },
      {
        title: t('billing.columns.description'),
        dataIndex: 'description',
        key: 'description',
        width: 200,
        ellipsis: true,
      },
      {
        title: t('billing.columns.amount'),
        dataIndex: 'amount',
        key: 'amount',
        width: 120,
        render: (amount: number, record: Invoice) => (
          <Text strong>
            {record.currency} {amount.toFixed(2)}
          </Text>
        ),
      },
      {
        title: t('billing.columns.period'),
        key: 'period',
        width: 180,
        render: (_: unknown, record: Invoice) => (
          <Text type="tertiary">
            {new Date(record.period_start).toLocaleDateString()} -{' '}
            {new Date(record.period_end).toLocaleDateString()}
          </Text>
        ),
      },
      {
        title: t('billing.columns.status'),
        dataIndex: 'status',
        key: 'status',
        width: 100,
        render: (status: InvoiceStatus) => (
          <Tag color={STATUS_COLORS[status]}>{t(`billing.status.${status}`)}</Tag>
        ),
      },
      {
        title: t('billing.columns.dueDate'),
        dataIndex: 'due_date',
        key: 'due_date',
        width: 120,
        render: (date: string) => new Date(date).toLocaleDateString(),
      },
      {
        title: t('billing.columns.paidAt'),
        dataIndex: 'paid_at',
        key: 'paid_at',
        width: 120,
        render: (date: string | undefined) => (date ? new Date(date).toLocaleDateString() : '-'),
      },
      {
        title: t('billing.columns.actions'),
        key: 'actions',
        width: 100,
        fixed: 'right',
        render: (_: unknown, record: Invoice) => (
          <Button
            icon={<IconDownload />}
            theme="borderless"
            size="small"
            onClick={() => handleDownloadInvoice(record)}
            disabled={!record.pdf_url}
          >
            {t('billing.downloadPdf')}
          </Button>
        ),
      },
    ],
    [t, handleDownloadInvoice]
  )

  // Status filter options
  const statusOptions = useMemo(
    () => [
      { value: '', label: t('billing.filters.allStatus') },
      { value: 'paid', label: t('billing.status.paid') },
      { value: 'pending', label: t('billing.status.pending') },
      { value: 'overdue', label: t('billing.status.overdue') },
      { value: 'cancelled', label: t('billing.status.cancelled') },
      { value: 'refunded', label: t('billing.status.refunded') },
    ],
    [t]
  )

  // Render payment method card
  const renderPaymentMethod = useCallback(
    (method: PaymentMethod, isDefault: boolean) => (
      <div
        key={method.id}
        className={`payment-method-card ${isDefault ? 'payment-method-card--default' : ''}`}
      >
        <div className="payment-method-icon">{PAYMENT_METHOD_ICONS[method.type] || '💳'}</div>
        <div className="payment-method-info">
          <div className="payment-method-header">
            <Text strong>{t(`billing.paymentTypes.${method.type}`)}</Text>
            {isDefault && (
              <Tag color="green" size="small">
                {t('billing.defaultMethod')}
              </Tag>
            )}
          </div>
          {method.type === 'card' && (
            <Text type="tertiary">
              {method.brand} •••• {method.last_four}
              {method.exp_month && method.exp_year && (
                <span className="payment-method-expiry">
                  {' '}
                  ({method.exp_month}/{method.exp_year})
                </span>
              )}
            </Text>
          )}
          {method.type === 'bank_transfer' && (
            <Text type="tertiary">
              {method.bank_name} •••• {method.account_last_four}
            </Text>
          )}
          {(method.type === 'alipay' || method.type === 'wechat') && (
            <Text type="tertiary">{t(`billing.paymentTypes.${method.type}`)}</Text>
          )}
        </div>
        <div className="payment-method-actions">
          {!isDefault && (
            <Button
              icon={<IconTick />}
              theme="borderless"
              size="small"
              onClick={() => handleSetDefault(method.id)}
              loading={setDefaultMutation.isPending}
            >
              {t('billing.setAsDefault')}
            </Button>
          )}
          <Popconfirm
            title={t('billing.confirm.deleteTitle')}
            content={t('billing.confirm.deleteContent')}
            onConfirm={() => handleDeletePaymentMethod(method.id)}
          >
            <Button
              icon={<IconDelete />}
              theme="borderless"
              type="danger"
              size="small"
              loading={deleteMutation.isPending}
            >
              {t('common.delete')}
            </Button>
          </Popconfirm>
        </div>
      </div>
    ),
    [
      t,
      handleSetDefault,
      handleDeletePaymentMethod,
      setDefaultMutation.isPending,
      deleteMutation.isPending,
    ]
  )

  // Loading state
  if (isSummaryLoading && isHistoryLoading) {
    return (
      <Container size="lg" className="billing-history-page">
        <Skeleton.Title style={{ width: 200, marginBottom: 24 }} />
        <Skeleton.Paragraph rows={4} />
      </Container>
    )
  }

  return (
    <Container size="lg" className="billing-history-page">
      {/* Page Header */}
      <div className="billing-header">
        <div className="billing-header-content">
          <Title heading={3}>{t('billing.title')}</Title>
          <Text type="tertiary">{t('billing.subtitle')}</Text>
        </div>
        <Button icon={<IconRefresh />} onClick={handleRefresh}>
          {t('common.refresh')}
        </Button>
      </div>

      {/* Error Banners */}
      {(isHistoryError || isSummaryError) && (
        <Banner
          type="danger"
          description={t('billing.messages.loadError')}
          className="error-banner"
        />
      )}

      {/* Upcoming Billing Card */}
      <Card className="upcoming-billing-card">
        <Title heading={5} className="section-title">
          <IconCalendar className="section-icon" />
          {t('billing.upcomingBilling')}
        </Title>

        {isSummaryLoading ? (
          <Skeleton.Paragraph rows={2} />
        ) : summaryData?.upcoming ? (
          <div className="upcoming-billing-content">
            <Descriptions
              data={[
                {
                  key: t('billing.nextBillingDate'),
                  value: new Date(summaryData.upcoming.next_billing_date).toLocaleDateString(),
                },
                {
                  key: t('billing.amount'),
                  value: (
                    <Text strong className="upcoming-amount">
                      {summaryData.upcoming.currency} {summaryData.upcoming.amount.toFixed(2)}
                    </Text>
                  ),
                },
                {
                  key: t('billing.plan'),
                  value: summaryData.upcoming.plan_name,
                },
                {
                  key: t('billing.billingCycle'),
                  value: t(`billing.cycles.${summaryData.upcoming.billing_cycle}`),
                },
                {
                  key: t('billing.autoRenew'),
                  value: summaryData.upcoming.auto_renew ? (
                    <Tag color="green">{t('common.enabled')}</Tag>
                  ) : (
                    <Tag color="grey">{t('common.disabled')}</Tag>
                  ),
                },
              ]}
            />
          </div>
        ) : (
          <Empty
            image={
              <IconCoinMoneyStroked style={{ fontSize: 48, color: 'var(--semi-color-text-2)' }} />
            }
            description={t('billing.noUpcomingBilling')}
          />
        )}
      </Card>

      {/* Payment Methods Card */}
      <Card className="payment-methods-card">
        <div className="payment-methods-header">
          <Title heading={5} className="section-title">
            <IconCreditCard className="section-icon" />
            {t('billing.paymentMethods')}
          </Title>
          <Button theme="solid" type="primary" size="small">
            {t('billing.addPaymentMethod')}
          </Button>
        </div>

        {isSummaryLoading ? (
          <Skeleton.Paragraph rows={3} />
        ) : summaryData?.payment_methods && summaryData.payment_methods.length > 0 ? (
          <div className="payment-methods-list">
            {summaryData.payment_methods.map((method) =>
              renderPaymentMethod(method, method.id === summaryData.default_payment_method_id)
            )}
          </div>
        ) : (
          <Empty
            image={<IconCreditCard style={{ fontSize: 48, color: 'var(--semi-color-text-2)' }} />}
            description={t('billing.noPaymentMethods')}
          />
        )}
      </Card>

      {/* Billing History Card */}
      <Card className="billing-history-card">
        <div className="billing-history-header">
          <Title heading={5} className="section-title">
            {t('billing.invoiceHistory')}
          </Title>

          <div className="billing-filters">
            <Select
              placeholder={t('billing.filters.status')}
              optionList={statusOptions}
              value={statusFilter || ''}
              onChange={(value) =>
                setStatusFilter((value as InvoiceStatus | undefined) || undefined)
              }
              style={{ width: 150 }}
            />
            <DatePicker
              type="dateRange"
              placeholder={[t('billing.filters.startDate'), t('billing.filters.endDate')]}
              value={dateRange}
              onChange={(dates) => setDateRange(dates as [Date, Date] | undefined)}
              style={{ width: 260 }}
            />
          </div>
        </div>

        {isHistoryLoading ? (
          <Skeleton.Paragraph rows={5} />
        ) : historyData?.invoices && historyData.invoices.length > 0 ? (
          <Table
            columns={columns}
            dataSource={historyData.invoices}
            rowKey="id"
            pagination={{
              currentPage: page,
              pageSize,
              total: historyData.total,
              onPageChange: setPage,
              onPageSizeChange: setPageSize,
              showSizeChanger: true,
              pageSizeOpts: [10, 20, 50],
            }}
            scroll={{ x: 1100 }}
            className="billing-table"
          />
        ) : (
          <Empty
            image={<IconDownload style={{ fontSize: 48, color: 'var(--semi-color-text-2)' }} />}
            description={t('billing.noInvoices')}
          />
        )}
      </Card>
    </Container>
  )
}
