import { useState, useEffect, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
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
} from '@douyinfe/semi-ui-19'
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
import {
  listFinanceReceivableReceivables,
  getFinanceReceivableReceivableSummary,
} from '@/api/finance-receivables/finance-receivables'
import type {
  HandlerAccountReceivableResponse,
  ListFinanceReceivableReceivablesParams,
  ListFinanceReceivableReceivablesStatus,
  ListFinanceReceivableReceivablesSourceType,
  HandlerReceivableSummaryResponse,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import './Receivables.css'

const { Title, Text } = Typography

// Receivable status type
type AccountReceivableStatus = 'PENDING' | 'PARTIAL' | 'PAID' | 'REVERSED' | 'CANCELLED'

// Receivable type with index signature for DataTable compatibility
type ReceivableRow = HandlerAccountReceivableResponse & Record<string, unknown>

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
 * Check if a receivable is overdue
 */
function isOverdue(receivable: HandlerAccountReceivableResponse): boolean {
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
  const { t } = useTranslation('finance')
  const navigate = useNavigate()

  // Status options for filter
  const STATUS_OPTIONS = useMemo(
    () => [
      { label: t('receivables.filter.allStatus'), value: '' },
      { label: t('receivables.status.PENDING'), value: 'PENDING' },
      { label: t('receivables.status.PARTIAL'), value: 'PARTIAL' },
      { label: t('receivables.status.PAID'), value: 'PAID' },
      { label: t('receivables.status.REVERSED'), value: 'REVERSED' },
      { label: t('receivables.status.CANCELLED'), value: 'CANCELLED' },
    ],
    [t]
  )

  // Source type options for filter
  const SOURCE_TYPE_OPTIONS = useMemo(
    () => [
      { label: t('receivables.filter.allSource'), value: '' },
      { label: t('receivables.sourceType.SALES_ORDER'), value: 'SALES_ORDER' },
      { label: t('receivables.sourceType.SALES_RETURN'), value: 'SALES_RETURN' },
      { label: t('receivables.sourceType.MANUAL'), value: 'MANUAL' },
    ],
    [t]
  )

  // State for data
  const [receivableList, setReceivableList] = useState<ReceivableRow[]>([])
  const [paginationMeta, setPaginationMeta] = useState<PaginationMeta | undefined>(undefined)
  const [loading, setLoading] = useState(false)
  const [summary, setSummary] = useState<HandlerReceivableSummaryResponse | null>(null)
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
      const params: ListFinanceReceivableReceivablesParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        search: searchKeyword || undefined,
        status: (statusFilter || undefined) as ListFinanceReceivableReceivablesStatus | undefined,
        source_type: (sourceTypeFilter || undefined) as
          | ListFinanceReceivableReceivablesSourceType
          | undefined,
        from_date: dateRange?.[0]?.toISOString().split('T')[0],
        to_date: dateRange?.[1]?.toISOString().split('T')[0],
        overdue: overdueOnly || undefined,
      }

      const response = await listFinanceReceivableReceivables(params)

      if (response.status === 200 && response.data.success && response.data.data) {
        setReceivableList(response.data.data as ReceivableRow[])
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
      Toast.error(t('receivables.messages.fetchError'))
    } finally {
      setLoading(false)
    }
  }, [
    state.pagination.page,
    state.pagination.pageSize,
    searchKeyword,
    statusFilter,
    sourceTypeFilter,
    dateRange,
    overdueOnly,
    t,
  ])

  // Fetch summary
  const fetchSummary = useCallback(async () => {
    setSummaryLoading(true)
    try {
      const response = await getFinanceReceivableReceivableSummary()
      if (response.status === 200 && response.data.success && response.data.data) {
        setSummary(response.data.data)
      }
    } catch {
      // Silently fail for summary
    } finally {
      setSummaryLoading(false)
    }
  }, [])

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
        title: t('receivables.columns.receivableNumber'),
        dataIndex: 'receivable_number',
        width: 140,
        sortable: true,
        render: (number: unknown, record: ReceivableRow) => (
          <div className="receivable-number-cell">
            <span
              className="receivable-number table-cell-link"
              onClick={() => {
                if (record.id) navigate(`/finance/receivables/${record.id}`)
              }}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault()
                  if (record.id) navigate(`/finance/receivables/${record.id}`)
                }
              }}
              role="link"
              tabIndex={0}
            >
              {(number as string) || '-'}
            </span>
            {isOverdue(record) && (
              <Tooltip content={t('receivables.tooltip.overdue')}>
                <span style={{ display: 'inline-flex' }}>
                  <IconAlertCircle className="overdue-icon" />
                </span>
              </Tooltip>
            )}
          </div>
        ),
      },
      {
        title: t('receivables.columns.customerName'),
        dataIndex: 'customer_name',
        sortable: true,
        ellipsis: true,
        render: (name: unknown) => <span className="customer-name">{(name as string) || '-'}</span>,
      },
      {
        title: t('receivables.columns.sourceDocument'),
        dataIndex: 'source_number',
        width: 140,
        render: (sourceNumber: unknown, record: ReceivableRow) => (
          <div className="source-cell">
            <span className="source-number">{(sourceNumber as string) || '-'}</span>
            <span className="source-type">
              {record.source_type ? String(t(`receivables.sourceType.${record.source_type}`)) : '-'}
            </span>
          </div>
        ),
      },
      {
        title: t('receivables.columns.totalAmount'),
        dataIndex: 'total_amount',
        width: 120,
        align: 'right',
        sortable: true,
        render: (amount: unknown) => (
          <span className="amount-cell total-amount">{formatCurrency(amount as number)}</span>
        ),
      },
      {
        title: t('receivables.columns.paidAmount'),
        dataIndex: 'paid_amount',
        width: 120,
        align: 'right',
        render: (amount: unknown) => (
          <span className="amount-cell paid-amount">{formatCurrency(amount as number)}</span>
        ),
      },
      {
        title: t('receivables.columns.outstandingAmount'),
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
        title: t('receivables.columns.dueDate'),
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
        title: t('receivables.columns.status'),
        dataIndex: 'status',
        width: 100,
        align: 'center',
        render: (status: unknown) => {
          const statusValue = status as AccountReceivableStatus | undefined
          if (!statusValue) return '-'
          return (
            <Tag color={STATUS_TAG_COLORS[statusValue]}>
              {t(`receivables.status.${statusValue}`)}
            </Tag>
          )
        },
      },
      {
        title: t('receivables.columns.createdAt'),
        dataIndex: 'created_at',
        width: 110,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined),
      },
    ],
    [t, navigate]
  )

  // Table row actions
  const tableActions: TableAction<ReceivableRow>[] = useMemo(
    () => [
      {
        key: 'view',
        label: t('receivables.actions.view'),
        onClick: handleView,
      },
      {
        key: 'collect',
        label: t('receivables.actions.collect'),
        type: 'primary',
        onClick: handleCollect,
        hidden: (record) =>
          record.status === 'PAID' || record.status === 'CANCELLED' || record.status === 'REVERSED',
      },
    ],
    [handleView, handleCollect, t]
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
                  {t('receivables.summary.totalOutstanding')}
                </Text>
                <Text className="summary-value primary">
                  {formatCurrency(summary?.total_outstanding)}
                </Text>
              </div>
            </Descriptions.Item>
            <Descriptions.Item itemKey="total_overdue">
              <div className="summary-item">
                <Text type="secondary" className="summary-label">
                  {t('receivables.summary.totalOverdue')}
                </Text>
                <Text className="summary-value danger">
                  {formatCurrency(summary?.total_overdue)}
                </Text>
              </div>
            </Descriptions.Item>
            <Descriptions.Item itemKey="pending_count">
              <div className="summary-item">
                <Text type="secondary" className="summary-label">
                  {t('receivables.summary.pendingCount')}
                </Text>
                <Text className="summary-value">{summary?.pending_count ?? '-'}</Text>
              </div>
            </Descriptions.Item>
            <Descriptions.Item itemKey="partial_count">
              <div className="summary-item">
                <Text type="secondary" className="summary-label">
                  {t('receivables.summary.partialCount')}
                </Text>
                <Text className="summary-value">{summary?.partial_count ?? '-'}</Text>
              </div>
            </Descriptions.Item>
            <Descriptions.Item itemKey="overdue_count">
              <div className="summary-item">
                <Text type="secondary" className="summary-label">
                  {t('receivables.summary.overdueCount')}
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
            {t('receivables.title')}
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder={t('receivables.searchPlaceholder')}
          secondaryActions={[
            {
              key: 'refresh',
              label: t('receivables.refresh'),
              icon: <IconRefresh />,
              onClick: handleRefresh,
            },
          ]}
          filters={
            <Space className="receivables-filter-container">
              <Select
                placeholder={t('receivables.filter.statusPlaceholder')}
                value={statusFilter}
                onChange={handleStatusChange}
                optionList={STATUS_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder={t('receivables.filter.sourcePlaceholder')}
                value={sourceTypeFilter}
                onChange={handleSourceTypeChange}
                optionList={SOURCE_TYPE_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder={t('receivables.filter.overduePlaceholder')}
                value={overdueOnly ? 'true' : ''}
                onChange={handleOverdueChange}
                optionList={[
                  { label: t('receivables.filter.all'), value: '' },
                  { label: t('receivables.filter.overdueOnly'), value: 'true' },
                ]}
                style={{ width: 100 }}
              />
              <DatePicker
                type="dateRange"
                placeholder={[t('receivables.filter.startDate'), t('receivables.filter.endDate')]}
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
