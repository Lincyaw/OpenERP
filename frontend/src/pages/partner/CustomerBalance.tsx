import { useState, useEffect, useCallback, useMemo } from 'react'
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
} from '@douyinfe/semi-ui'
import { IconArrowLeft, IconPlus, IconRefresh, IconHistory } from '@douyinfe/semi-icons'
import { useNavigate, useParams } from 'react-router-dom'
import { DataTable, TableToolbar, useTableState, type DataTableColumn } from '@/components/common'
import { Container } from '@/components/common/layout'
import { getBalance } from '@/api/balance/balance'
import { getCustomers } from '@/api/customers/customers'
import type {
  HandlerBalanceTransactionResponse,
  HandlerBalanceSummaryResponse,
  GetPartnerCustomersCustomerIdBalanceTransactionsParams,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import RechargeModal from './RechargeModal'
import './CustomerBalance.css'

const { Title, Text } = Typography

// Transaction type with index signature for DataTable compatibility
type BalanceTransaction = HandlerBalanceTransactionResponse & Record<string, unknown>

// Transaction type options for filter
const TRANSACTION_TYPE_OPTIONS = [
  { label: '全部类型', value: '' },
  { label: '充值', value: 'RECHARGE' },
  { label: '消费', value: 'CONSUME' },
  { label: '退款', value: 'REFUND' },
  { label: '调整', value: 'ADJUSTMENT' },
  { label: '过期', value: 'EXPIRE' },
]

// Source type options for filter
const SOURCE_TYPE_OPTIONS = [
  { label: '全部来源', value: '' },
  { label: '手动操作', value: 'MANUAL' },
  { label: '销售订单', value: 'SALES_ORDER' },
  { label: '销售退货', value: 'SALES_RETURN' },
  { label: '收款单', value: 'RECEIPT_VOUCHER' },
  { label: '系统', value: 'SYSTEM' },
]

// Transaction type labels
const TRANSACTION_TYPE_LABELS: Record<string, string> = {
  RECHARGE: '充值',
  CONSUME: '消费',
  REFUND: '退款',
  ADJUSTMENT: '调整',
  EXPIRE: '过期',
}

// Transaction type colors
const TRANSACTION_TYPE_COLORS: Record<string, 'green' | 'red' | 'blue' | 'orange' | 'grey'> = {
  RECHARGE: 'green',
  CONSUME: 'red',
  REFUND: 'blue',
  ADJUSTMENT: 'orange',
  EXPIRE: 'grey',
}

// Source type labels
const SOURCE_TYPE_LABELS: Record<string, string> = {
  MANUAL: '手动操作',
  SALES_ORDER: '销售订单',
  SALES_RETURN: '销售退货',
  RECEIPT_VOUCHER: '收款单',
  SYSTEM: '系统',
}

/**
 * Format currency for display
 */
function formatCurrency(amount?: number): string {
  if (amount === undefined || amount === null) return '¥0.00'
  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: 'CNY',
  }).format(amount)
}

/**
 * Format date for display
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
 * Customer Balance Management Page
 *
 * Features:
 * - Display customer balance summary (current balance, total recharge, consume, refund)
 * - Transaction history with filtering
 * - Recharge entry point
 */
export default function CustomerBalancePage() {
  const navigate = useNavigate()
  const { id: customerId } = useParams<{ id: string }>()
  const balanceApi = useMemo(() => getBalance(), [])
  const customerApi = useMemo(() => getCustomers(), [])

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
      const response = await customerApi.getPartnerCustomersId(customerId)
      if (response.success && response.data) {
        setCustomerName(response.data.name || '')
        setCustomerCode(response.data.code || '')
      }
    } catch {
      Toast.error('获取客户信息失败')
    }
  }, [customerId, customerApi])

  // Fetch balance summary
  const fetchBalanceSummary = useCallback(async () => {
    if (!customerId) return
    setSummaryLoading(true)
    try {
      const response = await balanceApi.getPartnerCustomersCustomerIdBalanceSummary(customerId)
      if (response.success && response.data) {
        setBalanceSummary(response.data)
      }
    } catch {
      Toast.error('获取余额信息失败')
    } finally {
      setSummaryLoading(false)
    }
  }, [customerId, balanceApi])

  // Fetch transactions
  const fetchTransactions = useCallback(async () => {
    if (!customerId) return
    setTransactionsLoading(true)
    try {
      const params: GetPartnerCustomersCustomerIdBalanceTransactionsParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        transaction_type: transactionTypeFilter || undefined,
        source_type: sourceTypeFilter || undefined,
        date_from: dateRange?.[0]?.toISOString().split('T')[0],
        date_to: dateRange?.[1]?.toISOString().split('T')[0],
      }

      const response = await balanceApi.getPartnerCustomersCustomerIdBalanceTransactions(
        customerId,
        params
      )

      if (response.success && response.data) {
        setTransactions(response.data as BalanceTransaction[])
        if (response.meta) {
          setPaginationMeta({
            page: response.meta.page || 1,
            page_size: response.meta.page_size || 20,
            total: response.meta.total || 0,
            total_pages: response.meta.total_pages || 1,
          })
        }
      }
    } catch {
      Toast.error('获取交易记录失败')
    } finally {
      setTransactionsLoading(false)
    }
  }, [
    customerId,
    balanceApi,
    state.pagination.page,
    state.pagination.pageSize,
    transactionTypeFilter,
    sourceTypeFilter,
    dateRange,
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
    Toast.success('充值成功')
  }, [fetchBalanceSummary, fetchTransactions])

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
    (dates: [Date, Date] | Date | string | undefined) => {
      if (Array.isArray(dates) && dates.length === 2) {
        setDateRange(dates as [Date, Date])
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

  // Table columns
  const tableColumns: DataTableColumn<BalanceTransaction>[] = useMemo(
    () => [
      {
        title: '交易时间',
        dataIndex: 'transaction_date',
        width: 160,
        sortable: true,
        render: (date: unknown) => formatDateTime(date as string | undefined),
      },
      {
        title: '交易类型',
        dataIndex: 'transaction_type',
        width: 100,
        align: 'center',
        render: (type: unknown) => {
          const typeValue = type as string | undefined
          if (!typeValue) return '-'
          return (
            <Tag color={TRANSACTION_TYPE_COLORS[typeValue] || 'grey'}>
              {TRANSACTION_TYPE_LABELS[typeValue] || typeValue}
            </Tag>
          )
        },
      },
      {
        title: '交易金额',
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
              {formatCurrency(amountValue)}
            </span>
          )
        },
      },
      {
        title: '变动前余额',
        dataIndex: 'balance_before',
        width: 120,
        align: 'right',
        render: (balance: unknown) => formatCurrency(balance as number | undefined),
      },
      {
        title: '变动后余额',
        dataIndex: 'balance_after',
        width: 120,
        align: 'right',
        render: (balance: unknown) => (
          <span className="balance-after">{formatCurrency(balance as number | undefined)}</span>
        ),
      },
      {
        title: '来源',
        dataIndex: 'source_type',
        width: 100,
        render: (source: unknown) => {
          const sourceValue = source as string | undefined
          if (!sourceValue) return '-'
          return SOURCE_TYPE_LABELS[sourceValue] || sourceValue
        },
      },
      {
        title: '参考号',
        dataIndex: 'reference',
        width: 140,
        ellipsis: true,
        render: (ref: unknown) => (ref as string) || '-',
      },
      {
        title: '备注',
        dataIndex: 'remark',
        ellipsis: true,
        render: (remark: unknown) => (remark as string) || '-',
      },
    ],
    []
  )

  if (!customerId) {
    return (
      <Container size="full" className="customer-balance-page">
        <Empty description="未找到客户" />
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
          返回客户列表
        </Button>
        <Title heading={4} style={{ margin: 0 }}>
          客户余额管理
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
                <div className="balance-card-label">当前余额</div>
                <div className="balance-card-value">
                  {formatCurrency(balanceSummary?.current_balance)}
                </div>
              </div>

              <div className="balance-card total-recharge">
                <div className="balance-card-label">累计充值</div>
                <div className="balance-card-value">
                  {formatCurrency(balanceSummary?.total_recharge)}
                </div>
              </div>

              <div className="balance-card total-consume">
                <div className="balance-card-label">累计消费</div>
                <div className="balance-card-value">
                  {formatCurrency(balanceSummary?.total_consume)}
                </div>
              </div>

              <div className="balance-card total-refund">
                <div className="balance-card-label">累计退款</div>
                <div className="balance-card-value">
                  {formatCurrency(balanceSummary?.total_refund)}
                </div>
              </div>
            </div>

            <div className="balance-actions-section">
              <Button
                type="primary"
                icon={<IconPlus />}
                onClick={() => setRechargeModalVisible(true)}
              >
                充值
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
            交易流水
          </Title>
        </div>

        <TableToolbar
          searchPlaceholder=""
          showSearch={false}
          secondaryActions={[
            {
              key: 'refresh',
              label: '刷新',
              icon: <IconRefresh />,
              onClick: handleRefresh,
            },
          ]}
          filters={
            <Space className="transactions-filter-container">
              <Select
                placeholder="交易类型"
                value={transactionTypeFilter}
                onChange={handleTransactionTypeChange}
                optionList={TRANSACTION_TYPE_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder="来源类型"
                value={sourceTypeFilter}
                onChange={handleSourceTypeChange}
                optionList={SOURCE_TYPE_OPTIONS}
                style={{ width: 120 }}
              />
              <DatePicker
                type="dateRange"
                placeholder={['开始日期', '结束日期']}
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
            emptyText="暂无交易记录"
          />
        </Spin>
      </Card>

      {/* Recharge Modal */}
      <RechargeModal
        visible={rechargeModalVisible}
        customerId={customerId}
        customerName={customerName}
        currentBalance={balanceSummary?.current_balance || 0}
        onClose={() => setRechargeModalVisible(false)}
        onSuccess={handleRechargeSuccess}
      />
    </Container>
  )
}
