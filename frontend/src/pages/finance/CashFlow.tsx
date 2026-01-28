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
} from '@douyinfe/semi-ui-19'
import { IconRefresh } from '@douyinfe/semi-icons'
import { DataTable, TableToolbar, useTableState, type DataTableColumn } from '@/components/common'
import { Container } from '@/components/common/layout'
import { getFinanceApi } from '@/api/finance'
import type {
  CashFlowItem,
  CashFlowSummary,
  CashFlowDirection,
  CashFlowItemType,
} from '@/api/finance'
import type { PaginationMeta } from '@/types/api'
import './CashFlow.css'

const { Title, Text } = Typography

// CashFlow item type with index signature for DataTable compatibility
type CashFlowRow = CashFlowItem & Record<string, unknown>

// Direction options for filter
const DIRECTION_OPTIONS = [
  { label: '全部方向', value: '' },
  { label: '收入', value: 'INFLOW' },
  { label: '支出', value: 'OUTFLOW' },
]

// Type options for filter
const TYPE_OPTIONS = [
  { label: '全部类型', value: '' },
  { label: '费用支出', value: 'EXPENSE' },
  { label: '其他收入', value: 'INCOME' },
  { label: '收款', value: 'RECEIPT' },
  { label: '付款', value: 'PAYMENT' },
]

// Direction tag colors
const DIRECTION_TAG_COLORS: Record<CashFlowDirection, 'green' | 'red'> = {
  INFLOW: 'green',
  OUTFLOW: 'red',
}

// Direction labels
const DIRECTION_LABELS: Record<CashFlowDirection, string> = {
  INFLOW: '收入',
  OUTFLOW: '支出',
}

// Type labels
const TYPE_LABELS: Record<CashFlowItemType, string> = {
  EXPENSE: '费用支出',
  INCOME: '其他收入',
  RECEIPT: '收款',
  PAYMENT: '付款',
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
  const api = useMemo(() => getFinanceApi(), [])

  // State for data
  const [cashFlowItems, setCashFlowItems] = useState<CashFlowRow[]>([])
  const [paginationMeta, setPaginationMeta] = useState<PaginationMeta | undefined>(undefined)
  const [loading, setLoading] = useState(false)
  const [summary, setSummary] = useState<CashFlowSummary | null>(null)
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

      const response = await api.getExpensCashFlow(params)

      if (response.success && response.data) {
        setSummary(response.data)

        // Filter items client-side based on direction and type filters
        let items = (response.data.items || []) as CashFlowRow[]

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
              item.number?.toLowerCase().includes(keyword) ||
              item.description?.toLowerCase().includes(keyword) ||
              item.category?.toLowerCase().includes(keyword)
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
      Toast.error('获取收支流水失败')
    } finally {
      setLoading(false)
      setSummaryLoading(false)
    }
  }, [
    api,
    dateRange,
    directionFilter,
    typeFilter,
    searchKeyword,
    state.pagination.page,
    state.pagination.pageSize,
    state.sort.order,
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
        title: '日期',
        dataIndex: 'date',
        width: 110,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined),
      },
      {
        title: '单据编号',
        dataIndex: 'number',
        width: 140,
        render: (number: unknown) => (
          <span className="cashflow-number">{(number as string) || '-'}</span>
        ),
      },
      {
        title: '类型',
        dataIndex: 'type',
        width: 100,
        align: 'center',
        render: (type: unknown) => {
          const typeValue = type as CashFlowItemType | undefined
          if (!typeValue) return '-'
          return <Tag color={TYPE_TAG_COLORS[typeValue]}>{TYPE_LABELS[typeValue]}</Tag>
        },
      },
      {
        title: '分类',
        dataIndex: 'category',
        width: 100,
        render: (category: unknown) => <span>{(category as string) || '-'}</span>,
      },
      {
        title: '描述',
        dataIndex: 'description',
        ellipsis: true,
        render: (desc: unknown) => <span>{(desc as string) || '-'}</span>,
      },
      {
        title: '方向',
        dataIndex: 'direction',
        width: 80,
        align: 'center',
        render: (direction: unknown) => {
          const directionValue = direction as CashFlowDirection | undefined
          if (!directionValue) return '-'
          return (
            <Tag color={DIRECTION_TAG_COLORS[directionValue]}>
              {DIRECTION_LABELS[directionValue]}
            </Tag>
          )
        },
      },
      {
        title: '金额',
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
    []
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
                  总收入
                </Text>
                <Text className="summary-value inflow">
                  {formatCurrency(summary?.total_inflow)}
                </Text>
              </div>
            </Descriptions.Item>
            <Descriptions.Item itemKey="total_outflow">
              <div className="summary-item">
                <Text type="secondary" className="summary-label">
                  总支出
                </Text>
                <Text className="summary-value outflow">
                  {formatCurrency(summary?.total_outflow)}
                </Text>
              </div>
            </Descriptions.Item>
            <Descriptions.Item itemKey="net_cash_flow">
              <div className="summary-item">
                <Text type="secondary" className="summary-label">
                  净现金流
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
                  费用支出
                </Text>
                <Text className="summary-value">{formatCurrency(summary?.expense_total)}</Text>
              </div>
            </Descriptions.Item>
            <Descriptions.Item itemKey="income_total">
              <div className="summary-item">
                <Text type="secondary" className="summary-label">
                  其他收入
                </Text>
                <Text className="summary-value">{formatCurrency(summary?.income_total)}</Text>
              </div>
            </Descriptions.Item>
          </Descriptions>
        </Spin>
      </div>

      {/* Main Content Card */}
      <Card className="cashflow-card">
        <div className="cashflow-header">
          <Title heading={4} style={{ margin: 0 }}>
            收支流水
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder="搜索单据编号、描述..."
          secondaryActions={[
            {
              key: 'refresh',
              label: '刷新',
              icon: <IconRefresh />,
              onClick: handleRefresh,
            },
          ]}
          filters={
            <Space className="cashflow-filter-container">
              <Select
                placeholder="收支方向"
                value={directionFilter}
                onChange={handleDirectionChange}
                optionList={DIRECTION_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder="流水类型"
                value={typeFilter}
                onChange={handleTypeChange}
                optionList={TYPE_OPTIONS}
                style={{ width: 120 }}
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
