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
  Descriptions,
  Divider,
  Select,
} from '@douyinfe/semi-ui-19'
import type { TagColor } from '@douyinfe/semi-ui-19/lib/es/tag'
import type { ColumnProps } from '@douyinfe/semi-ui-19/lib/es/table'
import {
  IconPriceTag,
  IconMinus,
  IconTick,
  IconArrowUp,
  IconArrowDown,
  IconDownload,
  IconHistory,
  IconSync,
} from '@douyinfe/semi-icons'
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
import { Container, Grid } from '@/components/common/layout'
import { getReports } from '@/api/reports'
import type { CashFlowStatement, CashFlowItem } from '@/api/reports'
import './CashFlowReport.css'
import { safeToFixed, toNumber } from '@/utils'

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

interface MetricCardProps {
  title: string
  value: string | number
  subValue?: string | number
  subLabel?: string
  icon: React.ReactNode
  color: string
  trend?: 'up' | 'down' | 'neutral'
  trendValue?: string
}

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
 * MetricCard component for displaying KPI metrics
 */
function MetricCard({
  title,
  value,
  subValue,
  subLabel,
  icon,
  color,
  trend,
  trendValue,
}: MetricCardProps) {
  return (
    <Card className="metric-card">
      <div className="metric-card-content">
        <div className="metric-icon" style={{ backgroundColor: color + '15', color }}>
          {icon}
        </div>
        <div className="metric-info">
          <Text type="tertiary" className="metric-label">
            {title}
          </Text>
          <div className="metric-value-row">
            <Title heading={3} className="metric-value" style={{ margin: 0 }}>
              {value}
            </Title>
            {trend && trendValue && (
              <Tag
                className="metric-trend"
                color={trend === 'up' ? 'green' : trend === 'down' ? 'red' : 'grey'}
              >
                {trend === 'up' ? <IconArrowUp size="small" /> : <IconArrowDown size="small" />}
                {trendValue}
              </Tag>
            )}
          </div>
          {subLabel && subValue !== undefined && (
            <Text type="tertiary" size="small" className="metric-sub">
              {subLabel}: <Text strong>{subValue}</Text>
            </Text>
          )}
        </div>
      </div>
    </Card>
  )
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
  const reportsApi = useMemo(() => getReports(), [])

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
        reportsApi.getReportsFinanceCashFlow(params),
        reportsApi.getReportsFinanceCashFlowItems(params),
      ])

      // Process statement
      if (statementRes.status === 'fulfilled' && statementRes.value.data) {
        setStatement(statementRes.value.data as unknown as CashFlowStatement)
      } else {
        setStatement(null)
      }

      // Process items
      if (itemsRes.status === 'fulfilled' && itemsRes.value.data) {
        setCashFlowItems(itemsRes.value.data as unknown as CashFlowItem[])
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
          const compRes = await reportsApi.getReportsFinanceCashFlow(comparisonParams)
          if (compRes.data) {
            setComparisonStatement(compRes.data as unknown as CashFlowStatement)
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
  }, [reportsApi, dateRange, comparisonType])

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
      statement.beginning_cash,
      statement.receipts_from_customers,
      -statement.payments_to_suppliers,
      statement.other_income,
      -statement.expense_payments,
      statement.ending_cash,
    ]

    // Calculate cumulative for waterfall effect
    let cumulative = statement.beginning_cash
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
          const value = values[index]
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

        {/* Summary Metrics Cards */}
        <div className="report-metrics">
          <Grid cols={{ mobile: 1, tablet: 2, desktop: 4 }} gap="md">
            <MetricCard
              title="期末现金余额"
              value={formatCurrency(statement?.ending_cash)}
              subLabel="期初余额"
              subValue={formatCurrency(statement?.beginning_cash)}
              icon={<IconPriceTag size="large" />}
              color="var(--semi-color-primary)"
              trend={netCashFlowChange?.trend}
              trendValue={netCashFlowChange?.value}
            />
            <MetricCard
              title="现金净增加"
              value={formatCurrency(statement?.net_cash_flow)}
              subLabel="对比期间"
              subValue={
                comparisonStatement ? formatCurrency(comparisonStatement.net_cash_flow) : '-'
              }
              icon={
                statement?.net_cash_flow !== undefined && statement.net_cash_flow >= 0 ? (
                  <IconArrowUp size="large" />
                ) : (
                  <IconArrowDown size="large" />
                )
              }
              color={
                statement?.net_cash_flow !== undefined && statement.net_cash_flow >= 0
                  ? 'var(--semi-color-success)'
                  : 'var(--semi-color-danger)'
              }
            />
            <MetricCard
              title="经营活动现金流"
              value={formatCurrency(statement?.net_operating_cash_flow)}
              subLabel="对比期间"
              subValue={
                comparisonStatement
                  ? formatCurrency(comparisonStatement.net_operating_cash_flow)
                  : '-'
              }
              icon={<IconTick size="large" />}
              color="var(--semi-color-success)"
              trend={operatingCashFlowChange?.trend}
              trendValue={operatingCashFlowChange?.value}
            />
            <MetricCard
              title="客户收款"
              value={formatCurrency(statement?.receipts_from_customers)}
              subLabel="供应商付款"
              subValue={formatCurrency(statement?.payments_to_suppliers)}
              icon={<IconMinus size="large" />}
              color="var(--semi-color-warning)"
            />
          </Grid>
        </div>

        {/* Cash Flow Statement Breakdown */}
        <Card title="现金流量明细" className="statement-card">
          {statement ? (
            <Descriptions
              data={[
                {
                  key: '期初余额',
                  value: (
                    <div className="statement-section">
                      <div className="statement-row total">
                        <span>期初现金余额</span>
                        <span className="amount">{formatCurrency(statement.beginning_cash)}</span>
                      </div>
                    </div>
                  ),
                },
                {
                  key: '经营活动',
                  value: (
                    <div className="statement-section">
                      <div className="statement-row positive">
                        <span>加: 客户收款</span>
                        <span className="amount">
                          {formatCurrency(statement.receipts_from_customers)}
                        </span>
                      </div>
                      <div className="statement-row negative">
                        <span>减: 供应商付款</span>
                        <span className="amount">
                          {formatCurrency(statement.payments_to_suppliers)}
                        </span>
                      </div>
                      <div className="statement-row positive">
                        <span>加: 其他收入</span>
                        <span className="amount">{formatCurrency(statement.other_income)}</span>
                      </div>
                      <div className="statement-row negative">
                        <span>减: 费用支出</span>
                        <span className="amount">{formatCurrency(statement.expense_payments)}</span>
                      </div>
                      <Divider margin="8px" />
                      <div
                        className={`statement-row subtotal ${statement.net_operating_cash_flow >= 0 ? 'positive' : 'negative'}`}
                      >
                        <span>经营活动现金流净额</span>
                        <span className="amount">
                          {formatCurrency(statement.net_operating_cash_flow)}
                        </span>
                      </div>
                    </div>
                  ),
                },
                {
                  key: '期末余额',
                  value: (
                    <div className="statement-section">
                      <div
                        className={`statement-row ${statement.net_cash_flow >= 0 ? 'positive' : 'negative'}`}
                      >
                        <span>现金净增加额</span>
                        <span className="amount">{formatCurrency(statement.net_cash_flow)}</span>
                      </div>
                      <Divider margin="8px" />
                      <div className="statement-row total">
                        <span>期末现金余额</span>
                        <span className="amount">{formatCurrency(statement.ending_cash)}</span>
                      </div>
                      {comparisonStatement && (
                        <div className="statement-row info">
                          <span>对比期间期末余额</span>
                          <Tag>{formatCurrency(comparisonStatement.ending_cash)}</Tag>
                        </div>
                      )}
                    </div>
                  ),
                },
              ]}
              row
              size="medium"
              className="statement-descriptions"
            />
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
