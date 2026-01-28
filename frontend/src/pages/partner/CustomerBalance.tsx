import { useState, useEffect, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Card,
  Typography,
  Tag,
  Toast,
  Select,
  Space,
  Button,
  Spin,
  DatePicker,
  Empty,
} from '@douyinfe/semi-ui-19'
import { IconArrowLeft, IconPlus, IconRefresh, IconHistory } from '@douyinfe/semi-icons'
import { useNavigate, useParams } from 'react-router-dom'
import { DataTable, TableToolbar, useTableState, type DataTableColumn } from '@/components/common'
import { Container } from '@/components/common/layout'
import { getBalanceBalanceSummary, listBalanceTransactions } from '@/api/balance/balance'
import { getCustomerById } from '@/api/customers/customers'
import { useFormatters } from '@/hooks/useFormatters'
import type {
  HandlerBalanceTransactionResponse,
  HandlerBalanceSummaryResponse,
  ListBalanceTransactionsParams,
  ListBalanceTransactionsTransactionType,
  ListBalanceTransactionsSourceType,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import RechargeModal from './RechargeModal'
import './CustomerBalance.css'

const { Title, Text } = Typography

// Transaction type with index signature for DataTable compatibility
type BalanceTransaction = HandlerBalanceTransactionResponse & Record<string, unknown>

// Transaction type colors
const TRANSACTION_TYPE_COLORS: Record<string, 'green' | 'red' | 'blue' | 'orange' | 'grey'> = {
  RECHARGE: 'green',
  CONSUME: 'red',
  REFUND: 'blue',
  ADJUSTMENT: 'orange',
  EXPIRE: 'grey',
}

/**
 * Customer Balance Management Page
 *
 * Features:
 * - Display customer balance summary (current balance, total recharge, consume, refund)
 * - Transaction history with filtering
 * - Recharge entry point
 */
export default function CustomerBalancePage() {
  const { t } = useTranslation(['partner', 'common'])
  const { formatDateTime, formatCurrency } = useFormatters()
  const navigate = useNavigate()
  const { id: customerId } = useParams<{ id: string }>()

  // Memoized transaction type options with translations
  const transactionTypeOptions = useMemo(
    () => [
      { label: t('partner:balance.transactionType.all'), value: '' },
      { label: t('partner:balance.transactionType.RECHARGE'), value: 'RECHARGE' },
      { label: t('partner:balance.transactionType.CONSUME'), value: 'CONSUME' },
      { label: t('partner:balance.transactionType.REFUND'), value: 'REFUND' },
      { label: t('partner:balance.transactionType.ADJUSTMENT'), value: 'ADJUSTMENT' },
      { label: t('partner:balance.transactionType.EXPIRE'), value: 'EXPIRE' },
    ],
    [t]
  )

  // Memoized source type options with translations
  const sourceTypeOptions = useMemo(
    () => [
      { label: t('partner:balance.sourceType.all'), value: '' },
      { label: t('partner:balance.sourceType.MANUAL'), value: 'MANUAL' },
      { label: t('partner:balance.sourceType.SALES_ORDER'), value: 'SALES_ORDER' },
      { label: t('partner:balance.sourceType.SALES_RETURN'), value: 'SALES_RETURN' },
      { label: t('partner:balance.sourceType.RECEIPT_VOUCHER'), value: 'RECEIPT_VOUCHER' },
      { label: t('partner:balance.sourceType.SYSTEM'), value: 'SYSTEM' },
    ],
    [t]
  )

  // Memoized transaction type labels for display
  const transactionTypeLabels = useMemo(
    () => ({
      RECHARGE: t('partner:balance.transactionType.RECHARGE'),
      CONSUME: t('partner:balance.transactionType.CONSUME'),
      REFUND: t('partner:balance.transactionType.REFUND'),
      ADJUSTMENT: t('partner:balance.transactionType.ADJUSTMENT'),
      EXPIRE: t('partner:balance.transactionType.EXPIRE'),
    }),
    [t]
  )

  // Memoized source type labels for display
  const sourceTypeLabels = useMemo(
    () => ({
      MANUAL: t('partner:balance.sourceType.MANUAL'),
      SALES_ORDER: t('partner:balance.sourceType.SALES_ORDER'),
      SALES_RETURN: t('partner:balance.sourceType.SALES_RETURN'),
      RECEIPT_VOUCHER: t('partner:balance.sourceType.RECEIPT_VOUCHER'),
      SYSTEM: t('partner:balance.sourceType.SYSTEM'),
    }),
    [t]
  )

  // State for customer info
  const [customerName, setCustomerName] = useState<string>('')
  const [customerCode, setCustomerCode] = useState<string>('')

  // State for balance summary
  const [balanceSummary, setBalanceSummary] = useState<HandlerBalanceSummaryResponse | null>(null)
  const [summaryLoading, setSummaryLoading] = useState(false)

  // State for transaction list
  const [transactions, setTransactions] = useState<BalanceTransaction[]>([])
  const [paginationMeta, setPaginationMeta] = useState<PaginationMeta | undefined>(undefined)
  const [transactionsLoading, setTransactionsLoading] = useState(false)

  // Filter state
  const [transactionTypeFilter, setTransactionTypeFilter] = useState<string>('')
  const [sourceTypeFilter, setSourceTypeFilter] = useState<string>('')
  const [dateRange, setDateRange] = useState<[Date, Date] | null>(null)

  // Modal state
  const [rechargeModalVisible, setRechargeModalVisible] = useState(false)

  // Table state hook
  const { state, handleStateChange } = useTableState({
    defaultPageSize: 20,
    defaultSortField: 'transaction_date',
    defaultSortOrder: 'desc',
  })

  // Fetch customer info
  const fetchCustomerInfo = useCallback(async () => {
    if (!customerId) return
    try {
      const response = await getCustomerById(customerId)
      if (response.status === 200 && response.data.success && response.data.data) {
        setCustomerName(response.data.data.name || '')
        setCustomerCode(response.data.data.code || '')
      }
    } catch {
      Toast.error(t('partner:customers.messages.fetchCustomerError'))
    }
  }, [customerId, t])

  // Fetch balance summary
  const fetchBalanceSummary = useCallback(async () => {
    if (!customerId) return
    setSummaryLoading(true)
    try {
      const response = await getBalanceBalanceSummary(customerId)
      if (response.status === 200 && response.data.success && response.data.data) {
        setBalanceSummary(response.data.data)
      }
    } catch {
      Toast.error(t('partner:balance.fetchBalanceError'))
    } finally {
      setSummaryLoading(false)
    }
  }, [customerId, t])

  // Fetch transactions
  const fetchTransactions = useCallback(async () => {
    if (!customerId) return
    setTransactionsLoading(true)
    try {
      const params: ListBalanceTransactionsParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        transaction_type: (transactionTypeFilter || undefined) as
          | ListBalanceTransactionsTransactionType
          | undefined,
        source_type: (sourceTypeFilter || undefined) as
          | ListBalanceTransactionsSourceType
          | undefined,
        date_from: dateRange?.[0]?.toISOString().split('T')[0],
        date_to: dateRange?.[1]?.toISOString().split('T')[0],
      }

      const response = await listBalanceTransactions(customerId, params)

      if (response.status === 200 && response.data.success && response.data.data) {
        setTransactions(response.data.data as BalanceTransaction[])
        if (response.data.meta) {
          setPaginationMeta({
            page: response.data.meta.page || 1,
            page_size: response.data.meta.page_size || 20,
            total: response.data.meta.total || 0,
            total_pages: response.data.meta.total_pages || 1,
          })
        }
      }
    } catch {
      Toast.error(t('partner:balance.fetchTransactionsError'))
    } finally {
      setTransactionsLoading(false)
    }
  }, [
    customerId,
    state.pagination.page,
    state.pagination.pageSize,
    transactionTypeFilter,
    sourceTypeFilter,
    dateRange,
    t,
  ])

  // Fetch data on mount
  useEffect(() => {
    fetchCustomerInfo()
    fetchBalanceSummary()
  }, [fetchCustomerInfo, fetchBalanceSummary])

  // Fetch transactions when filters change
  useEffect(() => {
    fetchTransactions()
  }, [fetchTransactions])

  // Handle recharge success
  const handleRechargeSuccess = useCallback(() => {
    setRechargeModalVisible(false)
    fetchBalanceSummary()
    fetchTransactions()
    Toast.success(t('partner:balance.rechargeModal.rechargeSuccess'))
  }, [fetchBalanceSummary, fetchTransactions, t])

  // Handle transaction type filter change
  const handleTransactionTypeChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      setTransactionTypeFilter(typeof value === 'string' ? value : '')
      handleStateChange({ pagination: { page: 1, pageSize: state.pagination.pageSize } })
    },
    [handleStateChange, state.pagination.pageSize]
  )

  // Handle source type filter change
  const handleSourceTypeChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      setSourceTypeFilter(typeof value === 'string' ? value : '')
      handleStateChange({ pagination: { page: 1, pageSize: state.pagination.pageSize } })
    },
    [handleStateChange, state.pagination.pageSize]
  )

  // Handle date range change
  const handleDateRangeChange = useCallback(
    (dates: Date | Date[] | string | string[] | undefined) => {
      if (
        Array.isArray(dates) &&
        dates.length === 2 &&
        dates[0] instanceof Date &&
        dates[1] instanceof Date
      ) {
        setDateRange([dates[0], dates[1]])
      } else {
        setDateRange(null)
      }
      handleStateChange({ pagination: { page: 1, pageSize: state.pagination.pageSize } })
    },
    [handleStateChange, state.pagination.pageSize]
  )

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchBalanceSummary()
    fetchTransactions()
  }, [fetchBalanceSummary, fetchTransactions])

  // Helper function to format currency with fallback
  // Note: API returns balance values as strings, so we need to parse them
  const formatCurrencyValue = useCallback(
    (amount?: number | string): string => {
      if (amount === undefined || amount === null) return formatCurrency(0)
      const numValue = typeof amount === 'string' ? parseFloat(amount) : amount
      if (isNaN(numValue)) return formatCurrency(0)
      return formatCurrency(numValue)
    },
    [formatCurrency]
  )

  // Helper function to format date with fallback
  const formatDateTimeValue = useCallback(
    (dateStr?: string): string => {
      if (!dateStr) return '-'
      return formatDateTime(dateStr)
    },
    [formatDateTime]
  )

  // Table columns
  const tableColumns: DataTableColumn<BalanceTransaction>[] = useMemo(
    () => [
      {
        title: t('partner:balance.columns.transactionDate'),
        dataIndex: 'transaction_date',
        width: 160,
        sortable: true,
        render: (date: unknown) => formatDateTimeValue(date as string | undefined),
      },
      {
        title: t('partner:balance.columns.transactionType'),
        dataIndex: 'transaction_type',
        width: 100,
        align: 'center',
        render: (type: unknown) => {
          const typeValue = type as string | undefined
          if (!typeValue) return '-'
          return (
            <Tag color={TRANSACTION_TYPE_COLORS[typeValue] || 'grey'}>
              {transactionTypeLabels[typeValue as keyof typeof transactionTypeLabels] || typeValue}
            </Tag>
          )
        },
      },
      {
        title: t('partner:balance.columns.amount'),
        dataIndex: 'amount',
        width: 120,
        align: 'right',
        render: (amount: unknown, record: BalanceTransaction) => {
          const amountValue = amount as number | undefined
          const balanceBefore = record.balance_before as number | undefined
          const balanceAfter = record.balance_after as number | undefined
          const isIncrease =
            balanceAfter !== undefined &&
            balanceBefore !== undefined &&
            balanceAfter > balanceBefore

          return (
            <span className={isIncrease ? 'amount-increase' : 'amount-decrease'}>
              {isIncrease ? '+' : '-'}
              {formatCurrencyValue(amountValue)}
            </span>
          )
        },
      },
      {
        title: t('partner:balance.columns.balanceBefore'),
        dataIndex: 'balance_before',
        width: 120,
        align: 'right',
        render: (balance: unknown) => formatCurrencyValue(balance as number | undefined),
      },
      {
        title: t('partner:balance.columns.balanceAfter'),
        dataIndex: 'balance_after',
        width: 120,
        align: 'right',
        render: (balance: unknown) => (
          <span className="balance-after">
            {formatCurrencyValue(balance as number | undefined)}
          </span>
        ),
      },
      {
        title: t('partner:balance.columns.source'),
        dataIndex: 'source_type',
        width: 100,
        render: (source: unknown) => {
          const sourceValue = source as string | undefined
          if (!sourceValue) return '-'
          return sourceTypeLabels[sourceValue as keyof typeof sourceTypeLabels] || sourceValue
        },
      },
      {
        title: t('partner:balance.columns.reference'),
        dataIndex: 'reference',
        width: 140,
        ellipsis: true,
        render: (ref: unknown) => (ref as string) || '-',
      },
      {
        title: t('partner:balance.columns.remark'),
        dataIndex: 'remark',
        ellipsis: true,
        render: (remark: unknown) => (remark as string) || '-',
      },
    ],
    [t, formatDateTimeValue, formatCurrencyValue, transactionTypeLabels, sourceTypeLabels]
  )

  if (!customerId) {
    return (
      <Container size="full" className="customer-balance-page">
        <Empty description={t('partner:balance.customerNotFound')} />
      </Container>
    )
  }

  return (
    <Container size="full" className="customer-balance-page">
      {/* Header */}
      <div className="balance-page-header">
        <Button
          icon={<IconArrowLeft />}
          theme="borderless"
          onClick={() => navigate('/partner/customers')}
        >
          {t('partner:balance.backToList')}
        </Button>
        <Title heading={4} style={{ margin: 0 }}>
          {t('partner:balance.pageTitle')}
        </Title>
      </div>

      {/* Customer Info & Balance Summary */}
      <Card className="balance-summary-card">
        <Spin spinning={summaryLoading}>
          <div className="balance-summary-content">
            <div className="customer-info-section">
              <div className="customer-info-header">
                <Text strong className="customer-name-text">
                  {customerName}
                </Text>
                {customerCode && (
                  <Tag color="blue" className="customer-code-tag">
                    {customerCode}
                  </Tag>
                )}
              </div>
            </div>

            <div className="balance-cards-section">
              <div className="balance-card current-balance">
                <div className="balance-card-label">{t('partner:balance.currentBalance')}</div>
                <div className="balance-card-value">
                  {formatCurrencyValue(balanceSummary?.current_balance)}
                </div>
              </div>

              <div className="balance-card total-recharge">
                <div className="balance-card-label">{t('partner:balance.totalRecharge')}</div>
                <div className="balance-card-value">
                  {formatCurrencyValue(balanceSummary?.total_recharge)}
                </div>
              </div>

              <div className="balance-card total-consume">
                <div className="balance-card-label">{t('partner:balance.totalConsume')}</div>
                <div className="balance-card-value">
                  {formatCurrencyValue(balanceSummary?.total_consume)}
                </div>
              </div>

              <div className="balance-card total-refund">
                <div className="balance-card-label">{t('partner:balance.totalRefund')}</div>
                <div className="balance-card-value">
                  {formatCurrencyValue(balanceSummary?.total_refund)}
                </div>
              </div>
            </div>

            <div className="balance-actions-section">
              <Button
                type="primary"
                icon={<IconPlus />}
                onClick={() => setRechargeModalVisible(true)}
              >
                {t('partner:balance.recharge')}
              </Button>
            </div>
          </div>
        </Spin>
      </Card>

      {/* Transaction History */}
      <Card className="transactions-card">
        <div className="transactions-header">
          <Title heading={5} style={{ margin: 0 }}>
            <IconHistory style={{ marginRight: 8 }} />
            {t('partner:balance.transactionHistory')}
          </Title>
        </div>

        <TableToolbar
          searchPlaceholder=""
          showSearch={false}
          secondaryActions={[
            {
              key: 'refresh',
              label: t('common:actions.refresh'),
              icon: <IconRefresh />,
              onClick: handleRefresh,
            },
          ]}
          filters={
            <Space className="transactions-filter-container">
              <Select
                placeholder={t('partner:balance.transactionType.all')}
                value={transactionTypeFilter}
                onChange={handleTransactionTypeChange}
                optionList={transactionTypeOptions}
                style={{ width: 120 }}
              />
              <Select
                placeholder={t('partner:balance.sourceType.all')}
                value={sourceTypeFilter}
                onChange={handleSourceTypeChange}
                optionList={sourceTypeOptions}
                style={{ width: 120 }}
              />
              <DatePicker
                type="dateRange"
                placeholder={[
                  t('partner:balance.dateRange.start'),
                  t('partner:balance.dateRange.end'),
                ]}
                value={dateRange || undefined}
                onChange={handleDateRangeChange}
                style={{ width: 240 }}
              />
            </Space>
          }
        />

        <Spin spinning={transactionsLoading}>
          <DataTable<BalanceTransaction>
            data={transactions}
            columns={tableColumns}
            rowKey="id"
            loading={transactionsLoading}
            pagination={paginationMeta}
            onStateChange={handleStateChange}
            sortState={state.sort}
            scroll={{ x: 1000 }}
            empty={t('partner:balance.noTransactions')}
          />
        </Spin>
      </Card>

      {/* Recharge Modal */}
      <RechargeModal
        visible={rechargeModalVisible}
        customerId={customerId}
        customerName={customerName}
        currentBalance={parseFloat(String(balanceSummary?.current_balance || '0')) || 0}
        onClose={() => setRechargeModalVisible(false)}
        onSuccess={handleRechargeSuccess}
      />
    </Container>
  )
}
