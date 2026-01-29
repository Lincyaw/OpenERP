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
} from '@douyinfe/semi-ui-19'
import { IconRefresh } from '@douyinfe/semi-icons'
import { DataTable, TableToolbar, useTableState, type DataTableColumn } from '@/components/common'
import { Container } from '@/components/common/layout'
import { getExpensCashFlow } from '@/api/expenses/expenses'
import type { HandlerCashFlowItemResponse, HandlerCashFlowSummaryResponse } from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import './CashFlow.css'

const { Title, Text } = Typography

// CashFlow direction type
type CashFlowDirection = 'INFLOW' | 'OUTFLOW'

// CashFlow item type
type CashFlowItemType = 'EXPENSE' | 'INCOME' | 'RECEIPT' | 'PAYMENT'

// CashFlow item type with index signature for DataTable compatibility
type CashFlowRow = HandlerCashFlowItemResponse & Record<string, unknown>

// Direction tag colors
const DIRECTION_TAG_COLORS: Record<CashFlowDirection, 'green' | 'red'> = {
  INFLOW: 'green',
  OUTFLOW: 'red',
}

// Type tag colors
const TYPE_TAG_COLORS: Record<CashFlowItemType, 'orange' | 'cyan' | 'green' | 'red'> = {
  EXPENSE: 'red',
  INCOME: 'green',
  RECEIPT: 'cyan',
  PAYMENT: 'orange',
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
 * Get default date range (current month)
 */
function getDefaultDateRange(): [Date, Date] {
  const now = new Date()
  const startOfMonth = new Date(now.getFullYear(), now.getMonth(), 1)
  const endOfMonth = new Date(now.getFullYear(), now.getMonth() + 1, 0)
  return [startOfMonth, endOfMonth]
}

/**
 * Cash Flow Page
 *
 * Features:
 * - Cash flow summary showing total inflow, outflow, and net cash flow
 * - Cash flow item listing with pagination
 * - Filter by direction (inflow/outflow), type, and date range
 * - Summary cards showing key metrics
 */
export default function CashFlowPage() {
  const { t } = useTranslation('finance')

  // Direction options for filter
  const DIRECTION_OPTIONS = useMemo(
    () => [
      { label: t('cashFlow.filter.allDirection'), value: '' },
      { label: t('cashFlow.direction.INFLOW'), value: 'INFLOW' },
      { label: t('cashFlow.direction.OUTFLOW'), value: 'OUTFLOW' },
    ],
    [t]
  )

  // Type options for filter
  const TYPE_OPTIONS = useMemo(
    () => [
      { label: t('cashFlow.filter.allType'), value: '' },
      { label: t('cashFlow.type.EXPENSE'), value: 'EXPENSE' },
      { label: t('cashFlow.type.INCOME'), value: 'INCOME' },
      { label: t('cashFlow.type.RECEIPT'), value: 'RECEIPT' },
      { label: t('cashFlow.type.PAYMENT'), value: 'PAYMENT' },
    ],
    [t]
  )

  // State for data
  const [cashFlowItems, setCashFlowItems] = useState<CashFlowRow[]>([])
  const [paginationMeta, setPaginationMeta] = useState<PaginationMeta | undefined>(undefined)
  const [loading, setLoading] = useState(false)
  const [summary, setSummary] = useState<HandlerCashFlowSummaryResponse | null>(null)
  const [summaryLoading, setSummaryLoading] = useState(false)

  // Filter state
  const [searchKeyword, setSearchKeyword] = useState('')
  const [directionFilter, setDirectionFilter] = useState<string>('')
  const [typeFilter, setTypeFilter] = useState<string>('')
  const [dateRange, setDateRange] = useState<[Date, Date] | null>(getDefaultDateRange())

  // Table state hook
  const { state, handleStateChange, setFilter } = useTableState({
    defaultPageSize: 20,
    defaultSortField: 'date',
    defaultSortOrder: 'desc',
  })

  // Fetch cash flow data
  const fetchCashFlow = useCallback(async () => {
    setLoading(true)
    setSummaryLoading(true)
    try {
      const params: {
        from_date?: string
        to_date?: string
        include_items?: boolean
      } = {
        include_items: true,
      }

      if (dateRange?.[0]) {
        params.from_date = dateRange[0].toISOString().split('T')[0]
      }
      if (dateRange?.[1]) {
        params.to_date = dateRange[1].toISOString().split('T')[0]
      }

      const response = await getExpensCashFlow(params)

      if (response.status === 200 && response.data.success && response.data.data) {
        setSummary(response.data.data)

        // Filter items client-side based on direction and type filters
        let items = ((response.data.data as { items?: CashFlowRow[] }).items || []) as CashFlowRow[]

        // Apply direction filter
        if (directionFilter) {
          items = items.filter((item) => item.direction === directionFilter)
        }

        // Apply type filter
        if (typeFilter) {
          items = items.filter((item) => item.type === typeFilter)
        }

        // Apply search filter
        if (searchKeyword) {
          const keyword = searchKeyword.toLowerCase()
          items = items.filter(
            (item) =>
              (item.number as string | undefined)?.toLowerCase().includes(keyword) ||
              (item.description as string | undefined)?.toLowerCase().includes(keyword) ||
              (item.category as string | undefined)?.toLowerCase().includes(keyword)
          )
        }

        // Sort by date descending by default
        items = items.sort((a, b) => {
          const dateA = new Date(a.date || 0).getTime()
          const dateB = new Date(b.date || 0).getTime()
          return state.sort.order === 'asc' ? dateA - dateB : dateB - dateA
        })

        // Apply pagination
        const pageSize = state.pagination.pageSize
        const page = state.pagination.page
        const startIndex = (page - 1) * pageSize
        const paginatedItems = items.slice(startIndex, startIndex + pageSize)

        setCashFlowItems(paginatedItems)
        setPaginationMeta({
          page,
          page_size: pageSize,
          total: items.length,
          total_pages: Math.ceil(items.length / pageSize),
        })
      }
    } catch {
      Toast.error(t('cashFlow.messages.fetchError'))
    } finally {
      setLoading(false)
      setSummaryLoading(false)
    }
  }, [
    dateRange,
    directionFilter,
    typeFilter,
    searchKeyword,
    state.pagination.page,
    state.pagination.pageSize,
    state.sort.order,
    t,
  ])

  // Fetch on mount and when filters change
  useEffect(() => {
    fetchCashFlow()
  }, [fetchCashFlow])

  // Handle search
  const handleSearch = useCallback(
    (value: string) => {
      setSearchKeyword(value)
      handleStateChange({ pagination: { page: 1, pageSize: state.pagination.pageSize } })
    },
    [handleStateChange, state.pagination.pageSize]
  )

  // Handle direction filter change
  const handleDirectionChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const directionValue = typeof value === 'string' ? value : ''
      setDirectionFilter(directionValue)
      setFilter('direction', directionValue || null)
      handleStateChange({ pagination: { page: 1, pageSize: state.pagination.pageSize } })
    },
    [setFilter, handleStateChange, state.pagination.pageSize]
  )

  // Handle type filter change
  const handleTypeChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const typeValue = typeof value === 'string' ? value : ''
      setTypeFilter(typeValue)
      setFilter('type', typeValue || null)
      handleStateChange({ pagination: { page: 1, pageSize: state.pagination.pageSize } })
    },
    [setFilter, handleStateChange, state.pagination.pageSize]
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

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchCashFlow()
  }, [fetchCashFlow])

  // Table columns
  const tableColumns: DataTableColumn<CashFlowRow>[] = useMemo(
    () => [
      {
        title: t('cashFlow.columns.transactionDate'),
        dataIndex: 'date',
        width: 110,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined),
      },
      {
        title: t('cashFlow.columns.referenceNo'),
        dataIndex: 'number',
        width: 140,
        render: (number: unknown) => (
          <span className="cashflow-number">{(number as string) || '-'}</span>
        ),
      },
      {
        title: t('cashFlow.columns.type'),
        dataIndex: 'type',
        width: 100,
        align: 'center',
        render: (type: unknown) => {
          const typeValue = type as CashFlowItemType | undefined
          if (!typeValue) return '-'
          return <Tag color={TYPE_TAG_COLORS[typeValue]}>{t(`cashFlow.type.${typeValue}`)}</Tag>
        },
      },
      {
        title: t('cashFlow.columns.description'),
        dataIndex: 'description',
        ellipsis: true,
        render: (desc: unknown) => <span>{(desc as string) || '-'}</span>,
      },
      {
        title: t('cashFlow.columns.direction'),
        dataIndex: 'direction',
        width: 80,
        align: 'center',
        render: (direction: unknown) => {
          const directionValue = direction as CashFlowDirection | undefined
          if (!directionValue) return '-'
          return (
            <Tag color={DIRECTION_TAG_COLORS[directionValue]}>
              {t(`cashFlow.direction.${directionValue}`)}
            </Tag>
          )
        },
      },
      {
        title: t('cashFlow.columns.amount'),
        dataIndex: 'amount',
        width: 120,
        align: 'right',
        render: (amount: unknown, record: CashFlowRow) => {
          const isInflow = record.direction === 'INFLOW'
          return (
            <span className={`amount-cell ${isInflow ? 'amount-inflow' : 'amount-outflow'}`}>
              {isInflow ? '+' : '-'}
              {formatCurrency(amount as number)}
            </span>
          )
        },
      },
    ],
    [t]
  )

  return (
    <Container size="full" className="cashflow-page">
      {/* Summary Cards */}
      <div className="cashflow-summary">
        <Spin spinning={summaryLoading}>
          <Descriptions row className="summary-descriptions">
            <Descriptions.Item itemKey="total_inflow">
              <div className="summary-item">
                <Text type="secondary" className="summary-label">
                  {t('cashFlow.summary.totalInflow')}
                </Text>
                <Text className="summary-value inflow">
                  {formatCurrency(summary?.total_inflow)}
                </Text>
              </div>
            </Descriptions.Item>
            <Descriptions.Item itemKey="total_outflow">
              <div className="summary-item">
                <Text type="secondary" className="summary-label">
                  {t('cashFlow.summary.totalOutflow')}
                </Text>
                <Text className="summary-value outflow">
                  {formatCurrency(summary?.total_outflow)}
                </Text>
              </div>
            </Descriptions.Item>
            <Descriptions.Item itemKey="net_cash_flow">
              <div className="summary-item">
                <Text type="secondary" className="summary-label">
                  {t('cashFlow.summary.netCashFlow')}
                </Text>
                <Text
                  className={`summary-value ${(summary?.net_cash_flow ?? 0) >= 0 ? 'inflow' : 'outflow'}`}
                >
                  {formatCurrency(summary?.net_cash_flow)}
                </Text>
              </div>
            </Descriptions.Item>
            <Descriptions.Item itemKey="expense_total">
              <div className="summary-item">
                <Text type="secondary" className="summary-label">
                  {t('cashFlow.type.EXPENSE')}
                </Text>
                <Text className="summary-value">
                  {formatCurrency((summary as { expense_total?: number })?.expense_total || 0)}
                </Text>
              </div>
            </Descriptions.Item>
            <Descriptions.Item itemKey="income_total">
              <div className="summary-item">
                <Text type="secondary" className="summary-label">
                  {t('cashFlow.type.INCOME')}
                </Text>
                <Text className="summary-value">
                  {formatCurrency((summary as { income_total?: number })?.income_total || 0)}
                </Text>
              </div>
            </Descriptions.Item>
          </Descriptions>
        </Spin>
      </div>

      {/* Main Content Card */}
      <Card className="cashflow-card">
        <div className="cashflow-header">
          <Title heading={4} style={{ margin: 0 }}>
            {t('cashFlow.title')}
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder={t('cashFlow.searchPlaceholder')}
          secondaryActions={[
            {
              key: 'refresh',
              label: t('cashFlow.refresh'),
              icon: <IconRefresh />,
              onClick: handleRefresh,
            },
          ]}
          filters={
            <Space className="cashflow-filter-container">
              <Select
                placeholder={t('cashFlow.filter.directionPlaceholder')}
                value={directionFilter}
                onChange={handleDirectionChange}
                optionList={DIRECTION_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder={t('cashFlow.filter.typePlaceholder')}
                value={typeFilter}
                onChange={handleTypeChange}
                optionList={TYPE_OPTIONS}
                style={{ width: 120 }}
              />
              <DatePicker
                type="dateRange"
                placeholder={[t('cashFlow.filter.startDate'), t('cashFlow.filter.endDate')]}
                value={dateRange as [Date, Date] | undefined}
                onChange={handleDateRangeChange}
                style={{ width: 240 }}
              />
            </Space>
          }
        />

        <Spin spinning={loading}>
          <DataTable<CashFlowRow>
            data={cashFlowItems}
            columns={tableColumns}
            rowKey="id"
            loading={loading}
            pagination={paginationMeta}
            onStateChange={handleStateChange}
            sortState={state.sort}
            scroll={{ x: 900 }}
          />
        </Spin>
      </Card>
    </Container>
  )
}
