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
} from '@douyinfe/semi-ui'
import type { ColumnProps } from '@douyinfe/semi-ui/lib/es/table'
import {
  IconPriceTag,
  IconMinus,
  IconTick,
  IconArrowUp,
  IconArrowDown,
  IconDownload,
  IconLineChartStroked,
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
import type { ProfitLossStatement, MonthlyProfitTrend, ProfitByProduct } from '@/api/reports'
import './ProfitLoss.css'

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
 * Format percentage
 */
function formatPercent(value?: number): string {
  if (value === undefined || value === null) return '0%'
  return `${(value * 100).toFixed(1)}%`
}

/**
 * Get default date range (last 12 months)
 */
function getDefaultDateRange(): [Date, Date] {
  const end = new Date()
  const start = new Date()
  start.setMonth(start.getMonth() - 12)
  start.setDate(1)
  return [start, end]
}

/**
 * Format date to YYYY-MM-DD
 */
function formatDateParam(date: Date): string {
  return date.toISOString().split('T')[0]
}

/**
 * Format month label
 */
function formatMonthLabel(year: number, month: number): string {
  return `${year}/${String(month).padStart(2, '0')}`
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
 * Profit/Loss Report Page
 *
 * Features (P5-FE-004):
 * - P&L statement display with breakdown
 * - Monthly profit trend chart
 * - Profit by product table
 * - Date range / period selection
 * - Export support (CSV)
 */
export default function ProfitLossPage() {
  const reportsApi = useMemo(() => getReports(), [])

  // Date range state
  const [dateRange, setDateRange] = useState<[Date, Date]>(getDefaultDateRange)

  // Loading states
  const [loading, setLoading] = useState(true)
  const [chartLoading, setChartLoading] = useState(true)

  // Data states
  const [statement, setStatement] = useState<ProfitLossStatement | null>(null)
  const [monthlyTrends, setMonthlyTrends] = useState<MonthlyProfitTrend[]>([])
  const [profitByProduct, setProfitByProduct] = useState<ProfitByProduct[]>([])

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
    setChartLoading(true)

    const params = {
      start_date: formatDateParam(dateRange[0]),
      end_date: formatDateParam(dateRange[1]),
    }

    try {
      // Fetch all data in parallel
      const [statementRes, trendsRes, productRes] = await Promise.allSettled([
        reportsApi.getReportsFinanceProfitLoss(params),
        reportsApi.getReportsFinanceMonthlyTrend(params),
        reportsApi.getReportsFinanceProfitByProduct({ ...params, top_n: 20 }),
      ])

      // Process statement
      if (statementRes.status === 'fulfilled' && statementRes.value.data) {
        setStatement(statementRes.value.data as unknown as ProfitLossStatement)
      } else {
        setStatement(null)
      }

      // Process trends
      if (trendsRes.status === 'fulfilled' && trendsRes.value.data) {
        setMonthlyTrends(trendsRes.value.data as unknown as MonthlyProfitTrend[])
      } else {
        setMonthlyTrends([])
      }

      // Process product profit
      if (productRes.status === 'fulfilled' && productRes.value.data) {
        setProfitByProduct(productRes.value.data as unknown as ProfitByProduct[])
      } else {
        setProfitByProduct([])
      }
    } catch {
      Toast.error('获取损益报表数据失败')
    } finally {
      setLoading(false)
      setChartLoading(false)
    }
  }, [reportsApi, dateRange])

  // Fetch data on mount and when date range changes
  useEffect(() => {
    fetchReportData()
  }, [fetchReportData])

  // Build monthly trend chart options
  const trendChartOptions = useMemo(() => {
    if (monthlyTrends.length === 0) return null

    const months = monthlyTrends.map((d) => formatMonthLabel(d.year, d.month))
    const revenue = monthlyTrends.map((d) => d.sales_revenue)
    const grossProfit = monthlyTrends.map((d) => d.gross_profit)
    const netProfit = monthlyTrends.map((d) => d.net_profit)
    const grossMargin = monthlyTrends.map((d) => d.gross_margin * 100)
    const netMargin = monthlyTrends.map((d) => d.net_margin * 100)

    return {
      tooltip: {
        trigger: 'axis',
        axisPointer: {
          type: 'cross',
        },
        formatter: (params: Array<{ seriesName: string; value: number; axisValue: string }>) => {
          let result = `${params[0].axisValue}<br/>`
          params.forEach((param) => {
            const isPercent = param.seriesName.includes('率')
            const value = isPercent ? `${param.value.toFixed(1)}%` : formatCurrency(param.value)
            result += `${param.seriesName}: ${value}<br/>`
          })
          return result
        },
      },
      legend: {
        data: ['销售收入', '毛利', '净利润', '毛利率', '净利率'],
        bottom: 0,
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '12%',
        top: '3%',
        containLabel: true,
      },
      xAxis: {
        type: 'category',
        boundaryGap: false,
        data: months,
        axisLine: { lineStyle: { color: '#E5E6EB' } },
        axisLabel: { color: '#86909C' },
      },
      yAxis: [
        {
          type: 'value',
          name: '金额',
          position: 'left',
          axisLine: { show: false },
          axisTick: { show: false },
          splitLine: { lineStyle: { color: '#E5E6EB', type: 'dashed' } },
          axisLabel: {
            color: '#86909C',
            formatter: (value: number) => {
              if (value >= 10000) return `${(value / 10000).toFixed(0)}万`
              return value.toString()
            },
          },
        },
        {
          type: 'value',
          name: '利润率(%)',
          position: 'right',
          min: 0,
          max: 100,
          axisLine: { show: false },
          axisTick: { show: false },
          splitLine: { show: false },
          axisLabel: {
            color: '#86909C',
            formatter: '{value}%',
          },
        },
      ],
      series: [
        {
          name: '销售收入',
          type: 'bar',
          barMaxWidth: 30,
          itemStyle: { color: 'rgba(0, 119, 250, 0.6)', borderRadius: [4, 4, 0, 0] },
          data: revenue,
        },
        {
          name: '毛利',
          type: 'bar',
          barMaxWidth: 30,
          itemStyle: { color: 'rgba(0, 180, 42, 0.6)', borderRadius: [4, 4, 0, 0] },
          data: grossProfit,
        },
        {
          name: '净利润',
          type: 'bar',
          barMaxWidth: 30,
          itemStyle: { color: 'rgba(255, 125, 0, 0.6)', borderRadius: [4, 4, 0, 0] },
          data: netProfit,
        },
        {
          name: '毛利率',
          type: 'line',
          yAxisIndex: 1,
          smooth: true,
          symbol: 'circle',
          symbolSize: 6,
          lineStyle: { width: 2, color: '#00B42A' },
          itemStyle: { color: '#00B42A' },
          data: grossMargin,
        },
        {
          name: '净利率',
          type: 'line',
          yAxisIndex: 1,
          smooth: true,
          symbol: 'circle',
          symbolSize: 6,
          lineStyle: { width: 2, color: '#FF7D00' },
          itemStyle: { color: '#FF7D00' },
          data: netMargin,
        },
      ],
    }
  }, [monthlyTrends])

  // Product profit table columns
  const productColumns: ColumnProps<ProfitByProduct>[] = [
    {
      title: '商品',
      dataIndex: 'product_name',
      key: 'product_name',
      render: (text: string, record?: ProfitByProduct) => (
        <div>
          <Text strong>{text}</Text>
          <br />
          <Text type="tertiary" size="small">
            {record?.product_sku}
          </Text>
        </div>
      ),
    },
    {
      title: '分类',
      dataIndex: 'category_name',
      key: 'category_name',
      render: (name?: string) => name || '-',
    },
    {
      title: '销售收入',
      dataIndex: 'sales_revenue',
      key: 'sales_revenue',
      align: 'right' as const,
      sorter: (a?: ProfitByProduct, b?: ProfitByProduct) =>
        (a?.sales_revenue ?? 0) - (b?.sales_revenue ?? 0),
      render: (amount: number) => formatCurrency(amount),
    },
    {
      title: '成本',
      dataIndex: 'cogs',
      key: 'cogs',
      align: 'right' as const,
      render: (amount: number) => formatCurrency(amount),
    },
    {
      title: '毛利',
      dataIndex: 'gross_profit',
      key: 'gross_profit',
      align: 'right' as const,
      sorter: (a?: ProfitByProduct, b?: ProfitByProduct) =>
        (a?.gross_profit ?? 0) - (b?.gross_profit ?? 0),
      render: (profit: number) => (
        <Text style={{ color: profit >= 0 ? '#00B42A' : '#F53F3F' }}>{formatCurrency(profit)}</Text>
      ),
    },
    {
      title: '毛利率',
      dataIndex: 'gross_margin',
      key: 'gross_margin',
      align: 'right' as const,
      sorter: (a?: ProfitByProduct, b?: ProfitByProduct) =>
        (a?.gross_margin ?? 0) - (b?.gross_margin ?? 0),
      render: (margin: number) => {
        const color = margin >= 0.3 ? 'green' : margin >= 0.1 ? 'orange' : 'red'
        return <Tag color={color}>{formatPercent(margin)}</Tag>
      },
    },
    {
      title: '利润贡献',
      dataIndex: 'contribution',
      key: 'contribution',
      align: 'right' as const,
      sorter: (a?: ProfitByProduct, b?: ProfitByProduct) =>
        (a?.contribution ?? 0) - (b?.contribution ?? 0),
      render: (contribution: number) => formatPercent(contribution),
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
      ['销售收入', statement.sales_revenue.toFixed(2)],
      ['销售退货', statement.sales_returns.toFixed(2)],
      ['净销售收入', statement.net_sales_revenue.toFixed(2)],
      ['销售成本', statement.cogs.toFixed(2)],
      ['毛利润', statement.gross_profit.toFixed(2)],
      ['毛利率', (statement.gross_margin * 100).toFixed(2) + '%'],
      ['其他收入', statement.other_income.toFixed(2)],
      ['总收入', statement.total_income.toFixed(2)],
      ['费用支出', statement.expenses.toFixed(2)],
      ['净利润', statement.net_profit.toFixed(2)],
      ['净利率', (statement.net_margin * 100).toFixed(2) + '%'],
    ]

    const csvContent = [headers.join(','), ...rows.map((row) => row.join(','))].join('\n')

    // Create and download file
    const blob = new Blob(['\ufeff' + csvContent], { type: 'text/csv;charset=utf-8;' })
    const link = document.createElement('a')
    const url = URL.createObjectURL(blob)
    link.setAttribute('href', url)
    link.setAttribute(
      'download',
      `损益报表_${formatDateParam(dateRange[0])}_${formatDateParam(dateRange[1])}.csv`
    )
    link.style.visibility = 'hidden'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)

    Toast.success('导出成功')
  }

  return (
    <Container size="full" className="profit-loss-page">
      <Spin spinning={loading} size="large">
        {/* Page Header with Filter */}
        <div className="report-header">
          <div className="report-title">
            <Title heading={3} style={{ margin: 0 }}>
              <IconLineChartStroked style={{ marginRight: 8 }} />
              损益报表
            </Title>
            <Text type="secondary">企业盈亏分析与财务状况</Text>
          </div>
          <div className="report-filters">
            <DatePicker
              type="dateRange"
              value={dateRange}
              onChange={handleDateRangeChange}
              style={{ width: 260 }}
              format="yyyy-MM-dd"
            />
            <Button icon={<IconDownload />} onClick={handleExportCSV}>
              导出
            </Button>
          </div>
        </div>

        {/* Summary Metrics Cards */}
        <div className="report-metrics">
          <Grid cols={{ mobile: 1, tablet: 2, desktop: 4 }} gap="md">
            <MetricCard
              title="净销售收入"
              value={formatCurrency(statement?.net_sales_revenue)}
              subLabel="总收入"
              subValue={formatCurrency(statement?.total_income)}
              icon={<IconPriceTag size="large" />}
              color="var(--semi-color-primary)"
            />
            <MetricCard
              title="销售成本"
              value={formatCurrency(statement?.cogs)}
              subLabel="退货金额"
              subValue={formatCurrency(statement?.sales_returns)}
              icon={<IconMinus size="large" />}
              color="var(--semi-color-danger)"
            />
            <MetricCard
              title="毛利润"
              value={formatCurrency(statement?.gross_profit)}
              subLabel="毛利率"
              subValue={formatPercent(statement?.gross_margin)}
              icon={<IconTick size="large" />}
              color="var(--semi-color-success)"
            />
            <MetricCard
              title="净利润"
              value={formatCurrency(statement?.net_profit)}
              subLabel="净利率"
              subValue={formatPercent(statement?.net_margin)}
              icon={<IconArrowUp size="large" />}
              color="var(--semi-color-warning)"
            />
          </Grid>
        </div>

        {/* P&L Statement Breakdown */}
        <Card title="损益明细" className="statement-card">
          {statement ? (
            <Descriptions
              data={[
                {
                  key: '收入',
                  value: (
                    <div className="statement-section">
                      <div className="statement-row">
                        <span>销售收入</span>
                        <span className="amount">{formatCurrency(statement.sales_revenue)}</span>
                      </div>
                      <div className="statement-row indent negative">
                        <span>减: 销售退货</span>
                        <span className="amount">{formatCurrency(statement.sales_returns)}</span>
                      </div>
                      <Divider margin="8px" />
                      <div className="statement-row subtotal">
                        <span>净销售收入</span>
                        <span className="amount">
                          {formatCurrency(statement.net_sales_revenue)}
                        </span>
                      </div>
                    </div>
                  ),
                },
                {
                  key: '成本与毛利',
                  value: (
                    <div className="statement-section">
                      <div className="statement-row negative">
                        <span>减: 销售成本 (COGS)</span>
                        <span className="amount">{formatCurrency(statement.cogs)}</span>
                      </div>
                      <Divider margin="8px" />
                      <div className="statement-row subtotal positive">
                        <span>毛利润</span>
                        <span className="amount">{formatCurrency(statement.gross_profit)}</span>
                      </div>
                      <div className="statement-row info">
                        <span>毛利率</span>
                        <Tag color="green">{formatPercent(statement.gross_margin)}</Tag>
                      </div>
                    </div>
                  ),
                },
                {
                  key: '其他收支',
                  value: (
                    <div className="statement-section">
                      <div className="statement-row">
                        <span>加: 其他收入</span>
                        <span className="amount">{formatCurrency(statement.other_income)}</span>
                      </div>
                      <Divider margin="8px" />
                      <div className="statement-row subtotal">
                        <span>总收入</span>
                        <span className="amount">{formatCurrency(statement.total_income)}</span>
                      </div>
                      <div className="statement-row negative">
                        <span>减: 费用支出</span>
                        <span className="amount">{formatCurrency(statement.expenses)}</span>
                      </div>
                    </div>
                  ),
                },
                {
                  key: '净利润',
                  value: (
                    <div className="statement-section">
                      <div
                        className={`statement-row total ${statement.net_profit >= 0 ? 'positive' : 'negative'}`}
                      >
                        <span>净利润</span>
                        <span className="amount">{formatCurrency(statement.net_profit)}</span>
                      </div>
                      <div className="statement-row info">
                        <span>净利率</span>
                        <Tag
                          color={
                            statement.net_margin >= 0.1
                              ? 'green'
                              : statement.net_margin >= 0
                                ? 'orange'
                                : 'red'
                          }
                        >
                          {formatPercent(statement.net_margin)}
                        </Tag>
                      </div>
                    </div>
                  ),
                },
              ]}
              row
              size="medium"
              className="statement-descriptions"
            />
          ) : (
            <Empty description="暂无损益数据" />
          )}
        </Card>

        {/* Monthly Trend Chart */}
        <Card title="月度趋势" className="chart-card">
          <Spin spinning={chartLoading}>
            {trendChartOptions ? (
              <ReactEChartsCore
                echarts={echarts}
                option={trendChartOptions}
                style={{ height: 400 }}
                notMerge={true}
              />
            ) : (
              <Empty description="暂无月度趋势数据" style={{ height: 400 }} />
            )}
          </Spin>
        </Card>

        {/* Profit by Product Table */}
        <Card title="商品利润分析" className="product-profit-card">
          <Table
            columns={productColumns}
            dataSource={profitByProduct}
            rowKey="product_id"
            pagination={{
              pageSize: 10,
              showSizeChanger: true,
              pageSizeOpts: [10, 20, 50],
            }}
            loading={loading}
            empty={<Empty description="暂无商品利润数据" />}
          />
        </Card>
      </Spin>
    </Container>
  )
}
