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
import { useResponsive } from '@/hooks/useResponsive'
import {
  listFinancePayablePayables,
  getFinancePayablePayableSummary,
} from '@/api/finance-payables/finance-payables'
import type {
  HandlerAccountPayableResponse,
  ListFinancePayablePayablesParams,
  ListFinancePayablePayablesStatus,
  ListFinancePayablePayablesSourceType,
  HandlerPayableSummaryResponse,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import './Payables.css'

const { Title, Text } = Typography

// Payable status type
type AccountPayableStatus = 'PENDING' | 'PARTIAL' | 'PAID' | 'REVERSED' | 'CANCELLED'

// Payable type with index signature for DataTable compatibility
type PayableRow = HandlerAccountPayableResponse & Record<string, unknown>

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
 * Check if a payable is overdue
 */
function isOverdue(payable: HandlerAccountPayableResponse): boolean {
  if (!payable.due_date) return false
  if (payable.status === 'PAID' || payable.status === 'CANCELLED') return false
  return new Date(payable.due_date) < new Date()
}

/**
 * Payables list page
 *
 * Features:
 * - Account payable listing with pagination
 * - Search by payable number, supplier name
 * - Filter by status, source type, date range, overdue
 * - Summary cards showing key metrics
 * - Navigate to payable detail for payment
 */
export default function PayablesPage() {
  const { t } = useTranslation('finance')
  const navigate = useNavigate()
  const { isMobile } = useResponsive()

  // Status options for filter
  const STATUS_OPTIONS = useMemo(
    () => [
      { label: t('payables.filter.allStatus'), value: '' },
      { label: t('payables.status.PENDING'), value: 'PENDING' },
      { label: t('payables.status.PARTIAL'), value: 'PARTIAL' },
      { label: t('payables.status.PAID'), value: 'PAID' },
      { label: t('payables.status.REVERSED'), value: 'REVERSED' },
      { label: t('payables.status.CANCELLED'), value: 'CANCELLED' },
    ],
    [t]
  )

  // Source type options for filter
  const SOURCE_TYPE_OPTIONS = useMemo(
    () => [
      { label: t('payables.filter.allSource'), value: '' },
      { label: t('payables.sourceType.PURCHASE_ORDER'), value: 'PURCHASE_ORDER' },
      { label: t('payables.sourceType.PURCHASE_RETURN'), value: 'PURCHASE_RETURN' },
      { label: t('payables.sourceType.MANUAL'), value: 'MANUAL' },
    ],
    [t]
  )

  // State for data
  const [payableList, setPayableList] = useState<PayableRow[]>([])
  const [paginationMeta, setPaginationMeta] = useState<PaginationMeta | undefined>(undefined)
  const [loading, setLoading] = useState(false)
  const [summary, setSummary] = useState<HandlerPayableSummaryResponse | null>(null)
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

  // Fetch payables
  const fetchPayables = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListFinancePayablePayablesParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        search: searchKeyword || undefined,
        status: (statusFilter || undefined) as ListFinancePayablePayablesStatus | undefined,
        source_type: (sourceTypeFilter || undefined) as
          | ListFinancePayablePayablesSourceType
          | undefined,
        from_date: dateRange?.[0]?.toISOString().split('T')[0],
        to_date: dateRange?.[1]?.toISOString().split('T')[0],
        overdue: overdueOnly || undefined,
      }

      const response = await listFinancePayablePayables(params)

      if (response.status === 200 && response.data.success && response.data.data) {
        setPayableList(response.data.data as PayableRow[])
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
      Toast.error(t('payables.messages.fetchError'))
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
      const response = await getFinancePayablePayableSummary()
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
    fetchPayables()
  }, [fetchPayables])

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

  // Handle view payable
  const handleView = useCallback(
    (payable: PayableRow) => {
      if (payable.id) {
        navigate(`/finance/payables/${payable.id}`)
      }
    },
    [navigate]
  )

  // Handle pay (create payment voucher)
  const handlePay = useCallback(
    (payable: PayableRow) => {
      if (payable.id && payable.status !== 'PAID' && payable.status !== 'CANCELLED') {
        navigate(
          `/finance/payments/new?payable_id=${payable.id}&supplier_id=${payable.supplier_id}`
        )
      }
    },
    [navigate]
  )

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchPayables()
    fetchSummary()
  }, [fetchPayables, fetchSummary])

  // Table columns
  const tableColumns: DataTableColumn<PayableRow>[] = useMemo(
    () => [
      {
        title: t('payables.columns.payableNumber'),
        dataIndex: 'payable_number',
        width: 140,
        sortable: true,
        render: (number: unknown, record: PayableRow) => (
          <div className="payable-number-cell">
            <span
              className="payable-number table-cell-link"
              onClick={() => {
                if (record.id) navigate(`/finance/payables/${record.id}`)
              }}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault()
                  if (record.id) navigate(`/finance/payables/${record.id}`)
                }
              }}
              role="link"
              tabIndex={0}
            >
              {(number as string) || '-'}
            </span>
            {isOverdue(record) && (
              <Tooltip content={t('payables.tooltip.overdue')}>
                <span style={{ display: 'inline-flex' }}>
                  <IconAlertCircle className="overdue-icon" />
                </span>
              </Tooltip>
            )}
          </div>
        ),
      },
      {
        title: t('payables.columns.supplierName'),
        dataIndex: 'supplier_name',
        sortable: true,
        ellipsis: true,
        render: (name: unknown) => <span className="supplier-name">{(name as string) || '-'}</span>,
      },
      {
        title: t('payables.columns.sourceDocument'),
        dataIndex: 'source_number',
        width: 140,
        render: (sourceNumber: unknown, record: PayableRow) => (
          <div className="source-cell">
            <span className="source-number">{(sourceNumber as string) || '-'}</span>
            <span className="source-type">
              {record.source_type ? String(t(`payables.sourceType.${record.source_type}`)) : '-'}
            </span>
          </div>
        ),
      },
      {
        title: t('payables.columns.totalAmount'),
        dataIndex: 'total_amount',
        width: 120,
        align: 'right',
        sortable: true,
        render: (amount: unknown) => (
          <span className="amount-cell total-amount">{formatCurrency(amount as number)}</span>
        ),
      },
      {
        title: t('payables.columns.paidAmount'),
        dataIndex: 'paid_amount',
        width: 120,
        align: 'right',
        render: (amount: unknown) => (
          <span className="amount-cell paid-amount">{formatCurrency(amount as number)}</span>
        ),
      },
      {
        title: t('payables.columns.outstandingAmount'),
        dataIndex: 'outstanding_amount',
        width: 120,
        align: 'right',
        sortable: true,
        render: (amount: unknown, record: PayableRow) => (
          <span className={`amount-cell outstanding-amount ${isOverdue(record) ? 'overdue' : ''}`}>
            {formatCurrency(amount as number)}
          </span>
        ),
      },
      {
        title: t('payables.columns.dueDate'),
        dataIndex: 'due_date',
        width: 110,
        sortable: true,
        render: (date: unknown, record: PayableRow) => (
          <span className={`date-cell ${isOverdue(record) ? 'overdue' : ''}`}>
            {formatDate(date as string)}
          </span>
        ),
      },
      {
        title: t('payables.columns.status'),
        dataIndex: 'status',
        width: 100,
        align: 'center',
        render: (status: unknown) => {
          const statusValue = status as AccountPayableStatus | undefined
          if (!statusValue) return '-'
          return (
            <Tag color={STATUS_TAG_COLORS[statusValue]}>{t(`payables.status.${statusValue}`)}</Tag>
          )
        },
      },
      {
        title: t('payables.columns.createdAt'),
        dataIndex: 'created_at',
        width: 110,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined),
      },
    ],
    [t, navigate]
  )

  // Table row actions
  const tableActions: TableAction<PayableRow>[] = useMemo(
    () => [
      {
        key: 'view',
        label: t('payables.actions.view'),
        onClick: handleView,
      },
      {
        key: 'pay',
        label: t('payables.actions.pay'),
        type: 'primary',
        onClick: handlePay,
        hidden: (record) =>
          record.status === 'PAID' || record.status === 'CANCELLED' || record.status === 'REVERSED',
      },
    ],
    [handleView, handlePay, t]
  )

  return (
    <Container size="full" className="payables-page">
      {/* Summary Cards */}
      <div className="payables-summary">
        <Spin spinning={summaryLoading}>
          {isMobile ? (
            <div className="summary-grid-mobile">
              <div className="summary-item-mobile primary">
                <Text type="secondary" size="small">
                  {t('payables.summary.totalOutstanding')}
                </Text>
                <Text strong className="summary-value-mobile">
                  {formatCurrency(summary?.total_outstanding)}
                </Text>
              </div>
              <div className="summary-item-mobile danger">
                <Text type="secondary" size="small">
                  {t('payables.summary.totalOverdue')}
                </Text>
                <Text strong className="summary-value-mobile">
                  {formatCurrency(summary?.total_overdue)}
                </Text>
              </div>
              <div className="summary-item-mobile">
                <Text type="secondary" size="small">
                  {t('payables.summary.pendingCount')}
                </Text>
                <Text strong className="summary-value-mobile">
                  {summary?.pending_count ?? '-'}
                </Text>
              </div>
              <div className="summary-item-mobile">
                <Text type="secondary" size="small">
                  {t('payables.summary.partialCount')}
                </Text>
                <Text strong className="summary-value-mobile">
                  {summary?.partial_count ?? '-'}
                </Text>
              </div>
              <div className="summary-item-mobile warning">
                <Text type="secondary" size="small">
                  {t('payables.summary.overdueCount')}
                </Text>
                <Text strong className="summary-value-mobile">
                  {summary?.overdue_count ?? '-'}
                </Text>
              </div>
            </div>
          ) : (
            <Descriptions row className="summary-descriptions">
              <Descriptions.Item itemKey="total_outstanding">
                <div className="summary-item">
                  <Text type="secondary" className="summary-label">
                    {t('payables.summary.totalOutstanding')}
                  </Text>
                  <Text className="summary-value primary">
                    {formatCurrency(summary?.total_outstanding)}
                  </Text>
                </div>
              </Descriptions.Item>
              <Descriptions.Item itemKey="total_overdue">
                <div className="summary-item">
                  <Text type="secondary" className="summary-label">
                    {t('payables.summary.totalOverdue')}
                  </Text>
                  <Text className="summary-value danger">
                    {formatCurrency(summary?.total_overdue)}
                  </Text>
                </div>
              </Descriptions.Item>
              <Descriptions.Item itemKey="pending_count">
                <div className="summary-item">
                  <Text type="secondary" className="summary-label">
                    {t('payables.summary.pendingCount')}
                  </Text>
                  <Text className="summary-value">{summary?.pending_count ?? '-'}</Text>
                </div>
              </Descriptions.Item>
              <Descriptions.Item itemKey="partial_count">
                <div className="summary-item">
                  <Text type="secondary" className="summary-label">
                    {t('payables.summary.partialCount')}
                  </Text>
                  <Text className="summary-value">{summary?.partial_count ?? '-'}</Text>
                </div>
              </Descriptions.Item>
              <Descriptions.Item itemKey="overdue_count">
                <div className="summary-item">
                  <Text type="secondary" className="summary-label">
                    {t('payables.summary.overdueCount')}
                  </Text>
                  <Text className="summary-value warning">{summary?.overdue_count ?? '-'}</Text>
                </div>
              </Descriptions.Item>
            </Descriptions>
          )}
        </Spin>
      </div>

      {/* Main Content Card */}
      <Card className="payables-card">
        <div className="payables-header">
          <Title heading={4} style={{ margin: 0 }}>
            {t('payables.title')}
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder={t('payables.searchPlaceholder')}
          secondaryActions={[
            {
              key: 'refresh',
              label: t('payables.refresh'),
              icon: <IconRefresh />,
              onClick: handleRefresh,
            },
          ]}
          filters={
            <Space className="payables-filter-container">
              <Select
                placeholder={t('payables.filter.statusPlaceholder')}
                value={statusFilter}
                onChange={handleStatusChange}
                optionList={STATUS_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder={t('payables.filter.sourcePlaceholder')}
                value={sourceTypeFilter}
                onChange={handleSourceTypeChange}
                optionList={SOURCE_TYPE_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder={t('payables.filter.overduePlaceholder')}
                value={overdueOnly ? 'true' : ''}
                onChange={handleOverdueChange}
                optionList={[
                  { label: t('payables.filter.all'), value: '' },
                  { label: t('payables.filter.overdueOnly'), value: 'true' },
                ]}
                style={{ width: 100 }}
              />
              <DatePicker
                type="dateRange"
                placeholder={[t('payables.filter.startDate'), t('payables.filter.endDate')]}
                value={dateRange as [Date, Date] | undefined}
                onChange={handleDateRangeChange}
                style={{ width: 240 }}
              />
            </Space>
          }
        />

        <Spin spinning={loading}>
          <DataTable<PayableRow>
            data={payableList}
            columns={tableColumns}
            rowKey="id"
            loading={loading}
            pagination={paginationMeta}
            actions={tableActions}
            onStateChange={handleStateChange}
            sortState={state.sort}
            scroll={{ x: 1200 }}
            mobileCardPrimaryColumns={['reference_number', 'supplier_name']}
          />
        </Spin>
      </Card>
    </Container>
  )
}
