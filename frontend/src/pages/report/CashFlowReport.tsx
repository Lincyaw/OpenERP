import { useState, useEffect, useCallback, useMemo } from 'react'
import {
  Card,
  Typography,
  DatePicker,
  Spin,
  Toast,
  Empty,
  Table,
  Tag,
  Button,
  Select,
} from '@douyinfe/semi-ui-19'
import type { TagColor } from '@douyinfe/semi-ui-19/lib/es/tag'
import type { ColumnProps } from '@douyinfe/semi-ui-19/lib/es/table'
import { IconDownload, IconHistory, IconSync } from '@douyinfe/semi-icons'
import ReactEChartsCore from 'echarts-for-react/lib/core'
import * as echarts from 'echarts/core'
import { LineChart, BarChart } from 'echarts/charts'
import {
  GridComponent,
  TooltipComponent,
  LegendComponent,
  TitleComponent,
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { Container } from '@/components/common/layout'
import { getReportCashFlowStatement, getReportCashFlowItems } from '@/api/reports/reports'
import type { HandlerCashFlowStatementResponse, HandlerCashFlowItemResponse } from '@/api/models'
import './CashFlowReport.css'
import { safeToFixed, toNumber } from '@/utils'

// Type aliases for cleaner code
type CashFlowStatement = HandlerCashFlowStatementResponse
type CashFlowItem = HandlerCashFlowItemResponse

// Register ECharts components
echarts.use([
  LineChart,
  BarChart,
  GridComponent,
  TooltipComponent,
  LegendComponent,
  TitleComponent,
  CanvasRenderer,
])

const { Title, Text } = Typography

/**
 * Format currency for display
 */
function formatCurrency(amount?: number): string {
  if (amount === undefined || amount === null) return '¥0.00'
  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: 'CNY',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(amount)
}

/**
 * Format date to YYYY-MM-DD
 */
function formatDateParam(date: Date): string {
  return date.toISOString().split('T')[0]
}

/**
 * Format date for display
 */
function formatDateDisplay(dateStr?: string): string {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN')
}

/**
 * Get default date range (current month)
 */
function getDefaultDateRange(): [Date, Date] {
  const end = new Date()
  const start = new Date()
  start.setDate(1) // First day of current month
  return [start, end]
}

/**
 * Get comparison period date range
 */
function getComparisonDateRange(current: [Date, Date], type: string): [Date, Date] {
  const [start, end] = current
  const daysDiff = Math.ceil((end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24))

  if (type === 'previous') {
    // Previous period of same length
    const newEnd = new Date(start)
    newEnd.setDate(newEnd.getDate() - 1)
    const newStart = new Date(newEnd)
    newStart.setDate(newStart.getDate() - daysDiff)
    return [newStart, newEnd]
  } else if (type === 'year') {
    // Same period last year
    const newStart = new Date(start)
    newStart.setFullYear(newStart.getFullYear() - 1)
    const newEnd = new Date(end)
    newEnd.setFullYear(newEnd.getFullYear() - 1)
    return [newStart, newEnd]
  }

  return current
}

/**
 * Get cash flow item type label in Chinese
 */
function getCashFlowTypeLabel(type: string): string {
  const typeMap: Record<string, string> = {
    RECEIPT: '客户收款',
    PAYMENT: '供应商付款',
    INCOME: '其他收入',
    EXPENSE: '费用支出',
  }
  return typeMap[type] || type
}

/**
 * Get cash flow item type color
 */
function getCashFlowTypeColor(type: string): TagColor {
  const colorMap: Record<string, TagColor> = {
    RECEIPT: 'green',
    PAYMENT: 'red',
    INCOME: 'cyan',
    EXPENSE: 'orange',
  }
  return colorMap[type] || 'grey'
}

/**
 * Cash Flow Report Page
 *
 * Features (P5-FE-005):
 * - Cash flow statement display
 * - Categorized activities breakdown (operating, investing, financing)
 * - Period comparison support
 * - Export support (CSV)
 */
export default function CashFlowReportPage() {
  // Date range state
  const [dateRange, setDateRange] = useState<[Date, Date]>(getDefaultDateRange)
  const [comparisonType, setComparisonType] = useState<string>('none')

  // Loading states
  const [loading, setLoading] = useState(true)
  const [itemsLoading, setItemsLoading] = useState(true)

  // Data states
  const [statement, setStatement] = useState<CashFlowStatement | null>(null)
  const [comparisonStatement, setComparisonStatement] = useState<CashFlowStatement | null>(null)
  const [cashFlowItems, setCashFlowItems] = useState<CashFlowItem[]>([])

  // Handle date range change from picker
  const handleDateRangeChange = (dates: unknown) => {
    if (Array.isArray(dates) && dates.length === 2) {
      const [start, end] = dates
      if (start instanceof Date && end instanceof Date) {
        setDateRange([start, end])
      }
    }
  }

  // Fetch all report data
  const fetchReportData = useCallback(async () => {
    setLoading(true)
    setItemsLoading(true)

    const params = {
      start_date: formatDateParam(dateRange[0]),
      end_date: formatDateParam(dateRange[1]),
    }

    try {
      // Fetch current period data
      const [statementRes, itemsRes] = await Promise.allSettled([
        getReportCashFlowStatement(params),
        getReportCashFlowItems(params),
      ])

      // Process statement
      if (
        statementRes.status === 'fulfilled' &&
        statementRes.value.status === 200 &&
        statementRes.value.data.success &&
        statementRes.value.data.data
      ) {
        setStatement(statementRes.value.data.data)
      } else {
        setStatement(null)
      }

      // Process items
      if (
        itemsRes.status === 'fulfilled' &&
        itemsRes.value.status === 200 &&
        itemsRes.value.data.success &&
        itemsRes.value.data.data
      ) {
        setCashFlowItems(itemsRes.value.data.data)
      } else {
        setCashFlowItems([])
      }

      // Fetch comparison period if selected
      if (comparisonType !== 'none') {
        const comparisonRange = getComparisonDateRange(dateRange, comparisonType)
        const comparisonParams = {
          start_date: formatDateParam(comparisonRange[0]),
          end_date: formatDateParam(comparisonRange[1]),
        }

        try {
          const compRes = await getReportCashFlowStatement(comparisonParams)
          if (compRes.status === 200 && compRes.data.success && compRes.data.data) {
            setComparisonStatement(compRes.data.data)
          } else {
            setComparisonStatement(null)
          }
        } catch {
          setComparisonStatement(null)
        }
      } else {
        setComparisonStatement(null)
      }
    } catch {
      Toast.error('获取现金流量报表数据失败')
    } finally {
      setLoading(false)
      setItemsLoading(false)
    }
  }, [dateRange, comparisonType])

  // Fetch data on mount and when date range changes
  useEffect(() => {
    fetchReportData()
  }, [fetchReportData])

  // Calculate change percentage
  const calculateChange = (
    current?: number,
    previous?: number
  ): { value: string; trend: 'up' | 'down' | 'neutral' } | null => {
    if (current === undefined || previous === undefined || previous === 0) {
      return null
    }
    const change = ((current - previous) / Math.abs(previous)) * 100
    return {
      value: `${safeToFixed(Math.abs(change), 1)}%`,
      trend: change > 0 ? 'up' : change < 0 ? 'down' : 'neutral',
    }
  }

  // Build cash flow waterfall chart options
  const chartOptions = useMemo(() => {
    if (!statement) return null

    const categories = ['期初余额', '客户收款', '供应商付款', '其他收入', '费用支出', '期末余额']
    const values = [
      statement.beginning_cash ?? 0,
      statement.receipts_from_customers ?? 0,
      -(statement.payments_to_suppliers ?? 0),
      statement.other_income ?? 0,
      -(statement.expense_payments ?? 0),
      statement.ending_cash ?? 0,
    ]

    // Calculate cumulative for waterfall effect
    let cumulative = statement.beginning_cash ?? 0
    const positiveData: (number | string)[] = []
    const negativeData: (number | string)[] = []
    const invisibleData: number[] = []

    values.forEach((val, index) => {
      if (index === 0 || index === values.length - 1) {
        // Starting and ending points
        positiveData.push(val)
        negativeData.push('-')
        invisibleData.push(0)
      } else {
        if (val >= 0) {
          positiveData.push(val)
          negativeData.push('-')
          invisibleData.push(cumulative)
        } else {
          positiveData.push('-')
          negativeData.push(Math.abs(val))
          invisibleData.push(cumulative + val)
        }
        cumulative += val
      }
    })

    return {
      tooltip: {
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
        formatter: (
          params: Array<{ seriesName: string; value: number | string; axisValue: string }>
        ) => {
          const index = categories.indexOf(params[0].axisValue)
          const value = values[index] ?? 0
          const formattedValue = formatCurrency(Math.abs(value))
          const sign = value >= 0 ? '+' : '-'
          return `${params[0].axisValue}<br/>金额: ${sign}${formattedValue}`
        },
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '3%',
        top: '3%',
        containLabel: true,
      },
      xAxis: {
        type: 'category',
        data: categories,
        axisLine: { lineStyle: { color: '#E5E6EB' } },
        axisLabel: { color: '#86909C' },
      },
      yAxis: {
        type: 'value',
        axisLine: { show: false },
        axisTick: { show: false },
        splitLine: { lineStyle: { color: '#E5E6EB', type: 'dashed' } },
        axisLabel: {
          color: '#86909C',
          formatter: (value: number) => {
            if (Math.abs(value) >= 10000) return `${safeToFixed(value / 10000, 0)}万`
            return value.toString()
          },
        },
      },
      series: [
        {
          name: '辅助',
          type: 'bar',
          stack: 'Total',
          silent: true,
          itemStyle: { color: 'transparent' },
          data: invisibleData,
        },
        {
          name: '收入',
          type: 'bar',
          stack: 'Total',
          barMaxWidth: 50,
          itemStyle: {
            color: '#00B42A',
            borderRadius: [4, 4, 0, 0],
          },
          label: {
            show: true,
            position: 'top',
            formatter: (params: { value: number | string }) => {
              if (params.value === '-') return ''
              return formatCurrency(params.value as number)
            },
            color: '#00B42A',
            fontSize: 11,
          },
          data: positiveData,
        },
        {
          name: '支出',
          type: 'bar',
          stack: 'Total',
          barMaxWidth: 50,
          itemStyle: {
            color: '#F53F3F',
            borderRadius: [4, 4, 0, 0],
          },
          label: {
            show: true,
            position: 'bottom',
            formatter: (params: { value: number | string }) => {
              if (params.value === '-') return ''
              return '-' + formatCurrency(params.value as number)
            },
            color: '#F53F3F',
            fontSize: 11,
          },
          data: negativeData,
        },
      ],
    }
  }, [statement])

  // Cash flow items table columns
  const itemColumns: ColumnProps<CashFlowItem>[] = [
    {
      title: '日期',
      dataIndex: 'date',
      key: 'date',
      width: 120,
      render: (date: string) => formatDateDisplay(date),
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 120,
      render: (type: string) => (
        <Tag color={getCashFlowTypeColor(type)}>{getCashFlowTypeLabel(type)}</Tag>
      ),
      filters: [
        { text: '客户收款', value: 'RECEIPT' },
        { text: '供应商付款', value: 'PAYMENT' },
        { text: '其他收入', value: 'INCOME' },
        { text: '费用支出', value: 'EXPENSE' },
      ],
      onFilter: (value: string, record?: CashFlowItem) => record?.type === value,
    },
    {
      title: '单据号',
      dataIndex: 'reference_no',
      key: 'reference_no',
      width: 160,
      render: (refNo?: string) => refNo || '-',
    },
    {
      title: '说明',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
      render: (desc?: string) => desc || '-',
    },
    {
      title: '金额',
      dataIndex: 'amount',
      key: 'amount',
      width: 140,
      align: 'right' as const,
      sorter: (a?: CashFlowItem, b?: CashFlowItem) => (a?.amount ?? 0) - (b?.amount ?? 0),
      render: (amount: number, record?: CashFlowItem) => {
        const isInflow = record?.type === 'RECEIPT' || record?.type === 'INCOME'
        return (
          <Text style={{ color: isInflow ? '#00B42A' : '#F53F3F' }}>
            {isInflow ? '+' : '-'}
            {formatCurrency(Math.abs(amount))}
          </Text>
        )
      },
    },
    {
      title: '余额',
      dataIndex: 'running_balance',
      key: 'running_balance',
      width: 140,
      align: 'right' as const,
      render: (balance?: number) => (balance !== undefined ? formatCurrency(balance) : '-'),
    },
  ]

  // Export CSV handler
  const handleExportCSV = () => {
    if (!statement) {
      Toast.warning('暂无数据可导出')
      return
    }

    // Build CSV content
    const headers = ['项目', '金额']
    const rows = [
      ['期初现金余额', safeToFixed(statement.beginning_cash)],
      ['经营活动现金流', ''],
      ['  客户收款', safeToFixed(statement.receipts_from_customers)],
      ['  供应商付款', safeToFixed(-toNumber(statement.payments_to_suppliers))],
      ['  其他收入', safeToFixed(statement.other_income)],
      ['  费用支出', safeToFixed(-toNumber(statement.expense_payments))],
      ['经营活动现金流净额', safeToFixed(statement.net_operating_cash_flow)],
      ['现金净增加额', safeToFixed(statement.net_cash_flow)],
      ['期末现金余额', safeToFixed(statement.ending_cash)],
    ]

    const csvContent = [headers.join(','), ...rows.map((row) => row.join(','))].join('\n')

    // Create and download file
    const blob = new Blob(['\ufeff' + csvContent], { type: 'text/csv;charset=utf-8;' })
    const link = document.createElement('a')
    const url = URL.createObjectURL(blob)
    link.setAttribute('href', url)
    link.setAttribute(
      'download',
      `现金流量报表_${formatDateParam(dateRange[0])}_${formatDateParam(dateRange[1])}.csv`
    )
    link.style.visibility = 'hidden'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)

    Toast.success('导出成功')
  }

  // Net cash flow change
  const netCashFlowChange = comparisonStatement
    ? calculateChange(statement?.net_cash_flow, comparisonStatement.net_cash_flow)
    : null

  // Operating cash flow change
  const operatingCashFlowChange = comparisonStatement
    ? calculateChange(
        statement?.net_operating_cash_flow,
        comparisonStatement.net_operating_cash_flow
      )
    : null

  return (
    <Container size="full" className="cash-flow-report-page">
      <Spin spinning={loading} size="large">
        {/* Page Header with Filter */}
        <div className="report-header">
          <div className="report-title">
            <Title heading={3} style={{ margin: 0 }}>
              <IconHistory style={{ marginRight: 8 }} />
              现金流量报表
            </Title>
            <Text type="secondary">企业现金流入流出分析</Text>
          </div>
          <div className="report-filters">
            <DatePicker
              type="dateRange"
              value={dateRange}
              onChange={handleDateRangeChange}
              style={{ width: 260 }}
              format="yyyy-MM-dd"
            />
            <Select
              value={comparisonType}
              onChange={(value) => setComparisonType(value as string)}
              style={{ width: 140 }}
              placeholder="对比期间"
            >
              <Select.Option value="none">不对比</Select.Option>
              <Select.Option value="previous">上一期间</Select.Option>
              <Select.Option value="year">去年同期</Select.Option>
            </Select>
            <Button icon={<IconSync />} onClick={fetchReportData}>
              刷新
            </Button>
            <Button icon={<IconDownload />} onClick={handleExportCSV}>
              导出
            </Button>
          </div>
        </div>

        {/* Summary Metrics - Compact Inline Display */}
        <Card className="metrics-bar">
          <div className="metrics-inline">
            <div className="metric-item">
              <Text type="tertiary">期末现金</Text>
              <Text strong className="metric-value-inline">
                {formatCurrency(statement?.ending_cash)}
              </Text>
              <Text type="tertiary" size="small">
                期初 {formatCurrency(statement?.beginning_cash)}
              </Text>
              {netCashFlowChange && (
                <Tag size="small" color={netCashFlowChange.trend === 'up' ? 'green' : 'red'}>
                  {netCashFlowChange.value}
                </Tag>
              )}
            </div>
            <div className="metric-divider" />
            <div className="metric-item">
              <Text type="tertiary">现金净增加</Text>
              <Text
                strong
                className="metric-value-inline"
                style={{
                  color:
                    (statement?.net_cash_flow ?? 0) >= 0
                      ? 'var(--semi-color-success)'
                      : 'var(--semi-color-danger)',
                }}
              >
                {formatCurrency(statement?.net_cash_flow)}
              </Text>
              {comparisonStatement && (
                <Text type="tertiary" size="small">
                  对比 {formatCurrency(comparisonStatement.net_cash_flow)}
                </Text>
              )}
            </div>
            <div className="metric-divider" />
            <div className="metric-item">
              <Text type="tertiary">经营活动现金流</Text>
              <Text
                strong
                className="metric-value-inline"
                style={{
                  color:
                    (statement?.net_operating_cash_flow ?? 0) >= 0
                      ? 'var(--semi-color-success)'
                      : 'var(--semi-color-danger)',
                }}
              >
                {formatCurrency(statement?.net_operating_cash_flow)}
              </Text>
              {operatingCashFlowChange && (
                <Tag size="small" color={operatingCashFlowChange.trend === 'up' ? 'green' : 'red'}>
                  {operatingCashFlowChange.value}
                </Tag>
              )}
            </div>
            <div className="metric-divider" />
            <div className="metric-item">
              <Text type="tertiary">客户收款</Text>
              <Text
                strong
                className="metric-value-inline"
                style={{ color: 'var(--semi-color-success)' }}
              >
                {formatCurrency(statement?.receipts_from_customers)}
              </Text>
              <Text type="tertiary" size="small">
                付款 {formatCurrency(statement?.payments_to_suppliers)}
              </Text>
            </div>
          </div>
        </Card>

        {/* Cash Flow Statement Breakdown - Compact Multi-Column Layout */}
        <Card title="现金流量明细" className="statement-card">
          {statement ? (
            <div className="statement-compact">
              <div className="statement-column">
                <div className="statement-group-title">期初余额</div>
                <div className="statement-line total">
                  <span>期初现金余额</span>
                  <span className="amount">{formatCurrency(statement.beginning_cash)}</span>
                </div>
              </div>

              <div className="statement-column">
                <div className="statement-group-title">经营活动现金流</div>
                <div className="statement-line indent">
                  <span>加: 客户收款</span>
                  <span className="amount positive">
                    {formatCurrency(statement.receipts_from_customers)}
                  </span>
                </div>
                <div className="statement-line indent">
                  <span>减: 供应商付款</span>
                  <span className="amount negative">
                    ({formatCurrency(statement.payments_to_suppliers)})
                  </span>
                </div>
                <div className="statement-line indent">
                  <span>加: 其他收入</span>
                  <span className="amount positive">{formatCurrency(statement.other_income)}</span>
                </div>
                <div className="statement-line indent">
                  <span>减: 费用支出</span>
                  <span className="amount negative">
                    ({formatCurrency(statement.expense_payments)})
                  </span>
                </div>
                <div className="statement-line subtotal">
                  <span>净额</span>
                  <span
                    className={`amount ${(statement.net_operating_cash_flow ?? 0) >= 0 ? 'positive' : 'negative'}`}
                  >
                    {formatCurrency(statement.net_operating_cash_flow)}
                  </span>
                </div>
              </div>

              <div className="statement-column highlight">
                <div className="statement-group-title">期末余额</div>
                <div className="statement-line">
                  <span>现金净增加额</span>
                  <span
                    className={`amount ${(statement.net_cash_flow ?? 0) >= 0 ? 'positive' : 'negative'}`}
                  >
                    {formatCurrency(statement.net_cash_flow)}
                  </span>
                </div>
                <div className="statement-line total">
                  <span>期末现金余额</span>
                  <span className="amount">{formatCurrency(statement.ending_cash)}</span>
                </div>
                {comparisonStatement && (
                  <div className="statement-line info">
                    <span>对比期间</span>
                    <Tag size="small">{formatCurrency(comparisonStatement.ending_cash)}</Tag>
                  </div>
                )}
              </div>
            </div>
          ) : (
            <Empty description="暂无现金流量数据" />
          )}
        </Card>

        {/* Cash Flow Waterfall Chart */}
        <Card title="现金流瀑布图" className="chart-card">
          {chartOptions ? (
            <ReactEChartsCore
              echarts={echarts}
              option={chartOptions}
              style={{ height: 350 }}
              notMerge={true}
            />
          ) : (
            <Empty description="暂无图表数据" style={{ height: 350 }} />
          )}
        </Card>

        {/* Cash Flow Items Table */}
        <Card title="现金流水明细" className="items-card">
          <Table
            columns={itemColumns}
            dataSource={cashFlowItems}
            rowKey={(record) => `${record?.date}-${record?.reference_no}-${record?.amount}`}
            pagination={{
              pageSize: 10,
              showSizeChanger: true,
              pageSizeOpts: [10, 20, 50],
            }}
            loading={itemsLoading}
            empty={<Empty description="暂无现金流水明细" />}
          />
        </Card>
      </Spin>
    </Container>
  )
}
