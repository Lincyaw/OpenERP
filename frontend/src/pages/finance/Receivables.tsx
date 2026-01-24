import { useState, useEffect, useCallback, useMemo } from 'react'
import {
  Card,
  Typography,
  Tag,
  Toast,
  Select,
  Space,
  Spin,
  DatePicker,
  Descriptions,
  Tooltip,
} from '@douyinfe/semi-ui'
import { IconRefresh, IconAlertCircle } from '@douyinfe/semi-icons'
import { useNavigate } from 'react-router-dom'
import {
  DataTable,
  TableToolbar,
  useTableState,
  type DataTableColumn,
  type TableAction,
} from '@/components/common'
import { Container } from '@/components/common/layout'
import { getFinanceApi } from '@/api/finance'
import type {
  AccountReceivable,
  AccountReceivableStatus,
  ReceivableSourceType,
  ReceivableSummary,
  GetReceivablesParams,
} from '@/api/finance'
import type { PaginationMeta } from '@/types/api'
import './Receivables.css'

const { Title, Text } = Typography

// Receivable type with index signature for DataTable compatibility
type ReceivableRow = AccountReceivable & Record<string, unknown>

// Status options for filter
const STATUS_OPTIONS = [
  { label: '全部状态', value: '' },
  { label: '待收款', value: 'PENDING' },
  { label: '部分收款', value: 'PARTIAL' },
  { label: '已收款', value: 'PAID' },
  { label: '已冲红', value: 'REVERSED' },
  { label: '已取消', value: 'CANCELLED' },
]

// Source type options for filter
const SOURCE_TYPE_OPTIONS = [
  { label: '全部来源', value: '' },
  { label: '销售订单', value: 'SALES_ORDER' },
  { label: '销售退货', value: 'SALES_RETURN' },
  { label: '手工录入', value: 'MANUAL' },
]

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

// Status labels
const STATUS_LABELS: Record<AccountReceivableStatus, string> = {
  PENDING: '待收款',
  PARTIAL: '部分收款',
  PAID: '已收款',
  REVERSED: '已冲红',
  CANCELLED: '已取消',
}

// Source type labels
const SOURCE_TYPE_LABELS: Record<ReceivableSourceType, string> = {
  SALES_ORDER: '销售订单',
  SALES_RETURN: '销售退货',
  MANUAL: '手工录入',
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
 * Check if a receivable is overdue
 */
function isOverdue(receivable: AccountReceivable): boolean {
  if (!receivable.due_date) return false
  if (receivable.status === 'PAID' || receivable.status === 'CANCELLED') return false
  return new Date(receivable.due_date) < new Date()
}

/**
 * Receivables list page
 *
 * Features:
 * - Account receivable listing with pagination
 * - Search by receivable number, customer name
 * - Filter by status, source type, date range, overdue
 * - Summary cards showing key metrics
 * - Navigate to receivable detail for collection
 */
export default function ReceivablesPage() {
  const navigate = useNavigate()
  const api = useMemo(() => getFinanceApi(), [])

  // State for data
  const [receivableList, setReceivableList] = useState<ReceivableRow[]>([])
  const [paginationMeta, setPaginationMeta] = useState<PaginationMeta | undefined>(undefined)
  const [loading, setLoading] = useState(false)
  const [summary, setSummary] = useState<ReceivableSummary | null>(null)
  const [summaryLoading, setSummaryLoading] = useState(false)

  // Filter state
  const [searchKeyword, setSearchKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [sourceTypeFilter, setSourceTypeFilter] = useState<string>('')
  const [dateRange, setDateRange] = useState<[Date, Date] | null>(null)
  const [overdueOnly, setOverdueOnly] = useState(false)

  // Table state hook
  const { state, handleStateChange, setFilter } = useTableState({
    defaultPageSize: 20,
    defaultSortField: 'created_at',
    defaultSortOrder: 'desc',
  })

  // Fetch receivables
  const fetchReceivables = useCallback(async () => {
    setLoading(true)
    try {
      const params: GetReceivablesParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        search: searchKeyword || undefined,
        status: (statusFilter || undefined) as AccountReceivableStatus | undefined,
        source_type: (sourceTypeFilter || undefined) as ReceivableSourceType | undefined,
        from_date: dateRange?.[0]?.toISOString().split('T')[0],
        to_date: dateRange?.[1]?.toISOString().split('T')[0],
        overdue: overdueOnly || undefined,
      }

      const response = await api.getFinanceReceivables(params)

      if (response.success && response.data) {
        setReceivableList(response.data as ReceivableRow[])
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
      Toast.error('获取应收账款列表失败')
    } finally {
      setLoading(false)
    }
  }, [
    api,
    state.pagination.page,
    state.pagination.pageSize,
    searchKeyword,
    statusFilter,
    sourceTypeFilter,
    dateRange,
    overdueOnly,
  ])

  // Fetch summary
  const fetchSummary = useCallback(async () => {
    setSummaryLoading(true)
    try {
      const response = await api.getFinanceReceivablesSummary()
      if (response.success && response.data) {
        setSummary(response.data)
      }
    } catch {
      // Silently fail for summary
    } finally {
      setSummaryLoading(false)
    }
  }, [api])

  // Fetch on mount and when state changes
  useEffect(() => {
    fetchReceivables()
  }, [fetchReceivables])

  // Fetch summary on mount
  useEffect(() => {
    fetchSummary()
  }, [fetchSummary])

  // Handle search
  const handleSearch = useCallback(
    (value: string) => {
      setSearchKeyword(value)
      handleStateChange({ pagination: { page: 1, pageSize: state.pagination.pageSize } })
    },
    [handleStateChange, state.pagination.pageSize]
  )

  // Handle status filter change
  const handleStatusChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const statusValue = typeof value === 'string' ? value : ''
      setStatusFilter(statusValue)
      setFilter('status', statusValue || null)
    },
    [setFilter]
  )

  // Handle source type filter change
  const handleSourceTypeChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const sourceValue = typeof value === 'string' ? value : ''
      setSourceTypeFilter(sourceValue)
      setFilter('source_type', sourceValue || null)
    },
    [setFilter]
  )

  // Handle date range change
  const handleDateRangeChange = useCallback(
    (dates: unknown) => {
      if (Array.isArray(dates) && dates.length === 2) {
        const [start, end] = dates
        if (start instanceof Date && end instanceof Date) {
          setDateRange([start, end])
        } else {
          setDateRange(null)
        }
      } else {
        setDateRange(null)
      }
      handleStateChange({ pagination: { page: 1, pageSize: state.pagination.pageSize } })
    },
    [handleStateChange, state.pagination.pageSize]
  )

  // Handle overdue filter
  const handleOverdueChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      setOverdueOnly(value === 'true')
    },
    []
  )

  // Handle view receivable
  const handleView = useCallback(
    (receivable: ReceivableRow) => {
      if (receivable.id) {
        navigate(`/finance/receivables/${receivable.id}`)
      }
    },
    [navigate]
  )

  // Handle collect (create receipt voucher)
  const handleCollect = useCallback(
    (receivable: ReceivableRow) => {
      if (receivable.id && receivable.status !== 'PAID' && receivable.status !== 'CANCELLED') {
        navigate(
          `/finance/receipts/new?receivable_id=${receivable.id}&customer_id=${receivable.customer_id}`
        )
      }
    },
    [navigate]
  )

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchReceivables()
    fetchSummary()
  }, [fetchReceivables, fetchSummary])

  // Table columns
  const tableColumns: DataTableColumn<ReceivableRow>[] = useMemo(
    () => [
      {
        title: '单据编号',
        dataIndex: 'receivable_number',
        width: 140,
        sortable: true,
        render: (number: unknown, record: ReceivableRow) => (
          <div className="receivable-number-cell">
            <span className="receivable-number">{(number as string) || '-'}</span>
            {isOverdue(record) && (
              <Tooltip content="已逾期">
                <IconAlertCircle className="overdue-icon" />
              </Tooltip>
            )}
          </div>
        ),
      },
      {
        title: '客户名称',
        dataIndex: 'customer_name',
        sortable: true,
        ellipsis: true,
        render: (name: unknown) => <span className="customer-name">{(name as string) || '-'}</span>,
      },
      {
        title: '来源单据',
        dataIndex: 'source_number',
        width: 140,
        render: (sourceNumber: unknown, record: ReceivableRow) => (
          <div className="source-cell">
            <span className="source-number">{(sourceNumber as string) || '-'}</span>
            <span className="source-type">
              {record.source_type ? SOURCE_TYPE_LABELS[record.source_type] : '-'}
            </span>
          </div>
        ),
      },
      {
        title: '应收金额',
        dataIndex: 'total_amount',
        width: 120,
        align: 'right',
        sortable: true,
        render: (amount: unknown) => (
          <span className="amount-cell total-amount">{formatCurrency(amount as number)}</span>
        ),
      },
      {
        title: '已收金额',
        dataIndex: 'paid_amount',
        width: 120,
        align: 'right',
        render: (amount: unknown) => (
          <span className="amount-cell paid-amount">{formatCurrency(amount as number)}</span>
        ),
      },
      {
        title: '待收金额',
        dataIndex: 'outstanding_amount',
        width: 120,
        align: 'right',
        sortable: true,
        render: (amount: unknown, record: ReceivableRow) => (
          <span className={`amount-cell outstanding-amount ${isOverdue(record) ? 'overdue' : ''}`}>
            {formatCurrency(amount as number)}
          </span>
        ),
      },
      {
        title: '到期日',
        dataIndex: 'due_date',
        width: 110,
        sortable: true,
        render: (date: unknown, record: ReceivableRow) => (
          <span className={`date-cell ${isOverdue(record) ? 'overdue' : ''}`}>
            {formatDate(date as string)}
          </span>
        ),
      },
      {
        title: '状态',
        dataIndex: 'status',
        width: 100,
        align: 'center',
        render: (status: unknown) => {
          const statusValue = status as AccountReceivableStatus | undefined
          if (!statusValue) return '-'
          return <Tag color={STATUS_TAG_COLORS[statusValue]}>{STATUS_LABELS[statusValue]}</Tag>
        },
      },
      {
        title: '创建时间',
        dataIndex: 'created_at',
        width: 110,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined),
      },
    ],
    []
  )

  // Table row actions
  const tableActions: TableAction<ReceivableRow>[] = useMemo(
    () => [
      {
        key: 'view',
        label: '查看',
        onClick: handleView,
      },
      {
        key: 'collect',
        label: '收款',
        type: 'primary',
        onClick: handleCollect,
        hidden: (record) =>
          record.status === 'PAID' || record.status === 'CANCELLED' || record.status === 'REVERSED',
      },
    ],
    [handleView, handleCollect]
  )

  return (
    <Container size="full" className="receivables-page">
      {/* Summary Cards */}
      <div className="receivables-summary">
        <Spin spinning={summaryLoading}>
          <Descriptions row className="summary-descriptions">
            <Descriptions.Item itemKey="total_outstanding">
              <div className="summary-item">
                <Text type="secondary" className="summary-label">
                  待收总额
                </Text>
                <Text className="summary-value primary">
                  {formatCurrency(summary?.total_outstanding)}
                </Text>
              </div>
            </Descriptions.Item>
            <Descriptions.Item itemKey="total_overdue">
              <div className="summary-item">
                <Text type="secondary" className="summary-label">
                  逾期总额
                </Text>
                <Text className="summary-value danger">
                  {formatCurrency(summary?.total_overdue)}
                </Text>
              </div>
            </Descriptions.Item>
            <Descriptions.Item itemKey="pending_count">
              <div className="summary-item">
                <Text type="secondary" className="summary-label">
                  待收款单
                </Text>
                <Text className="summary-value">{summary?.pending_count ?? '-'}</Text>
              </div>
            </Descriptions.Item>
            <Descriptions.Item itemKey="partial_count">
              <div className="summary-item">
                <Text type="secondary" className="summary-label">
                  部分收款
                </Text>
                <Text className="summary-value">{summary?.partial_count ?? '-'}</Text>
              </div>
            </Descriptions.Item>
            <Descriptions.Item itemKey="overdue_count">
              <div className="summary-item">
                <Text type="secondary" className="summary-label">
                  逾期单数
                </Text>
                <Text className="summary-value warning">{summary?.overdue_count ?? '-'}</Text>
              </div>
            </Descriptions.Item>
          </Descriptions>
        </Spin>
      </div>

      {/* Main Content Card */}
      <Card className="receivables-card">
        <div className="receivables-header">
          <Title heading={4} style={{ margin: 0 }}>
            应收账款
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder="搜索单据编号、客户名称..."
          secondaryActions={[
            {
              key: 'refresh',
              label: '刷新',
              icon: <IconRefresh />,
              onClick: handleRefresh,
            },
          ]}
          filters={
            <Space className="receivables-filter-container">
              <Select
                placeholder="状态筛选"
                value={statusFilter}
                onChange={handleStatusChange}
                optionList={STATUS_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder="来源筛选"
                value={sourceTypeFilter}
                onChange={handleSourceTypeChange}
                optionList={SOURCE_TYPE_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder="逾期筛选"
                value={overdueOnly ? 'true' : ''}
                onChange={handleOverdueChange}
                optionList={[
                  { label: '全部', value: '' },
                  { label: '仅逾期', value: 'true' },
                ]}
                style={{ width: 100 }}
              />
              <DatePicker
                type="dateRange"
                placeholder={['开始日期', '结束日期']}
                value={dateRange as [Date, Date] | undefined}
                onChange={handleDateRangeChange}
                style={{ width: 240 }}
              />
            </Space>
          }
        />

        <Spin spinning={loading}>
          <DataTable<ReceivableRow>
            data={receivableList}
            columns={tableColumns}
            rowKey="id"
            loading={loading}
            pagination={paginationMeta}
            actions={tableActions}
            onStateChange={handleStateChange}
            sortState={state.sort}
            scroll={{ x: 1200 }}
          />
        </Spin>
      </Card>
    </Container>
  )
}
