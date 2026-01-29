import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import {
  Card,
  Typography,
  DatePicker,
  Spin,
  Toast,
  Empty,
  Table,
  Tag,
  Tabs,
  TabPane,
  Select,
} from '@douyinfe/semi-ui-19'
import {
  IconLineChartStroked,
  IconPieChartStroked,
  IconPriceTag,
  IconShoppingBag,
  IconUserGroup,
  IconTick,
  IconArrowUp,
  IconArrowDown,
} from '@douyinfe/semi-icons'
import ReactEChartsCore from 'echarts-for-react/lib/core'
import * as echarts from 'echarts/core'
import { LineChart, PieChart, BarChart } from 'echarts/charts'
import {
  GridComponent,
  TooltipComponent,
  LegendComponent,
  TitleComponent,
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { Container, Row, Grid } from '@/components/common/layout'
import {
  getReportSalesSummary,
  getReportDailySalesTrend,
  getReportProductSalesRanking,
  getReportCustomerSalesRanking,
} from '@/api/reports/reports'
import type {
  HandlerSalesSummaryResponse,
  HandlerDailySalesTrendResponse,
  HandlerProductSalesRankingResponse,
  HandlerCustomerSalesRankingResponse,
} from '@/api/models'
import './SalesReport.css'
import { safeToFixed, toNumber } from '@/utils'

// Type aliases for cleaner code
type SalesSummary = HandlerSalesSummaryResponse
type DailySalesTrend = HandlerDailySalesTrendResponse
type ProductSalesRanking = HandlerProductSalesRankingResponse
type CustomerSalesRanking = HandlerCustomerSalesRankingResponse

// Register ECharts components
echarts.use([
  LineChart,
  PieChart,
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
 * Format number with comma separators
 */
function formatNumber(num?: number): string {
  if (num === undefined || num === null) return '0'
  return new Intl.NumberFormat('zh-CN').format(num)
}

/**
 * Format percentage
 */
function formatPercent(value?: number | string): string {
  if (value === undefined || value === null) return '0%'
  return `${safeToFixed(toNumber(value) * 100, 1, '0')}%`
}

/**
 * Get default date range (last 30 days)
 */
function getDefaultDateRange(): [Date, Date] {
  const end = new Date()
  const start = new Date()
  start.setDate(start.getDate() - 30)
  return [start, end]
}

/**
 * Format date to YYYY-MM-DD
 */
function formatDateParam(date: Date): string {
  return date.toISOString().split('T')[0]
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
 * Sales Report Page
 *
 * Features (P5-FE-001):
 * - Sales summary metrics cards
 * - Sales trend line chart (daily)
 * - Sales composition charts (by product, by customer)
 * - Date range filter with time range presets
 */
export default function SalesReportPage() {
  const chartRef = useRef<ReactEChartsCore>(null)

  // Date range state
  const [dateRange, setDateRange] = useState<[Date, Date]>(getDefaultDateRange)
  const [presetRange, setPresetRange] = useState<string>('30days')

  // Loading states
  const [loading, setLoading] = useState(true)
  const [chartLoading, setChartLoading] = useState(true)

  // Data states
  const [summary, setSummary] = useState<SalesSummary | null>(null)
  const [dailyTrends, setDailyTrends] = useState<DailySalesTrend[]>([])
  const [productRankings, setProductRankings] = useState<ProductSalesRanking[]>([])
  const [customerRankings, setCustomerRankings] = useState<CustomerSalesRanking[]>([])

  // Active tab for rankings
  const [rankingTab, setRankingTab] = useState<string>('products')

  // Preset range options
  const presetRangeOptions = [
    { value: '7days', label: '最近7天' },
    { value: '30days', label: '最近30天' },
    { value: '90days', label: '最近90天' },
    { value: 'thisMonth', label: '本月' },
    { value: 'lastMonth', label: '上月' },
    { value: 'thisYear', label: '本年' },
    { value: 'custom', label: '自定义' },
  ]

  // Handle preset range change
  const handlePresetChange = (
    value: string | number | (string | number)[] | Record<string, unknown> | undefined
  ) => {
    const presetValue = typeof value === 'string' ? value : '30days'
    setPresetRange(presetValue)
    const end = new Date()
    const start = new Date()

    switch (presetValue) {
      case '7days':
        start.setDate(start.getDate() - 7)
        break
      case '30days':
        start.setDate(start.getDate() - 30)
        break
      case '90days':
        start.setDate(start.getDate() - 90)
        break
      case 'thisMonth':
        start.setDate(1)
        break
      case 'lastMonth':
        start.setMonth(start.getMonth() - 1, 1)
        end.setDate(0) // Last day of previous month
        break
      case 'thisYear':
        start.setMonth(0, 1)
        break
      case 'custom':
        // Don't change dates for custom
        return
    }

    setDateRange([start, end])
  }

  // Handle date range change from picker
  const handleDateRangeChange = (dates: unknown) => {
    if (Array.isArray(dates) && dates.length === 2) {
      const [start, end] = dates
      if (start instanceof Date && end instanceof Date) {
        setDateRange([start, end])
        setPresetRange('custom')
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
      const [summaryRes, trendsRes, productsRes, customersRes] = await Promise.allSettled([
        getReportSalesSummary(params),
        getReportDailySalesTrend(params),
        getReportProductSalesRanking({ ...params, top_n: 10 }),
        getReportCustomerSalesRanking({ ...params, top_n: 10 }),
      ])

      // Process summary
      if (
        summaryRes.status === 'fulfilled' &&
        summaryRes.value.status === 200 &&
        summaryRes.value.data.success &&
        summaryRes.value.data.data
      ) {
        setSummary(summaryRes.value.data.data)
      } else {
        setSummary(null)
      }

      // Process trends
      if (
        trendsRes.status === 'fulfilled' &&
        trendsRes.value.status === 200 &&
        trendsRes.value.data.success &&
        trendsRes.value.data.data
      ) {
        setDailyTrends(trendsRes.value.data.data)
      } else {
        setDailyTrends([])
      }

      // Process product rankings
      if (
        productsRes.status === 'fulfilled' &&
        productsRes.value.status === 200 &&
        productsRes.value.data.success &&
        productsRes.value.data.data
      ) {
        setProductRankings(productsRes.value.data.data)
      } else {
        setProductRankings([])
      }

      // Process customer rankings
      if (
        customersRes.status === 'fulfilled' &&
        customersRes.value.status === 200 &&
        customersRes.value.data.success &&
        customersRes.value.data.data
      ) {
        setCustomerRankings(customersRes.value.data.data)
      } else {
        setCustomerRankings([])
      }
    } catch {
      Toast.error('获取销售报表数据失败')
    } finally {
      setLoading(false)
      setChartLoading(false)
    }
  }, [dateRange])

  // Fetch data on mount and when date range changes
  useEffect(() => {
    fetchReportData()
  }, [fetchReportData])

  // Build sales trend chart options
  const trendChartOptions = useMemo(() => {
    if (dailyTrends.length === 0) return null

    const dates = dailyTrends.map((d) => {
      const date = new Date(d.date || '')
      return `${date.getMonth() + 1}/${date.getDate()}`
    })
    const amounts = dailyTrends.map((d) => d.total_amount)
    const profits = dailyTrends.map((d) => d.total_profit)
    const orders = dailyTrends.map((d) => d.order_count)

    return {
      tooltip: {
        trigger: 'axis',
        axisPointer: {
          type: 'cross',
        },
        formatter: (params: Array<{ seriesName: string; value: number; axisValue: string }>) => {
          let result = `${params[0].axisValue}<br/>`
          params.forEach((param) => {
            const value =
              param.seriesName === '订单数'
                ? formatNumber(param.value)
                : formatCurrency(param.value)
            result += `${param.seriesName}: ${value}<br/>`
          })
          return result
        },
      },
      legend: {
        data: ['销售额', '毛利', '订单数'],
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
        data: dates,
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
              if (value >= 10000) return `${safeToFixed(value / 10000, 0)}万`
              return value.toString()
            },
          },
        },
        {
          type: 'value',
          name: '订单数',
          position: 'right',
          axisLine: { show: false },
          axisTick: { show: false },
          splitLine: { show: false },
          axisLabel: { color: '#86909C' },
        },
      ],
      series: [
        {
          name: '销售额',
          type: 'line',
          smooth: true,
          symbol: 'circle',
          symbolSize: 6,
          lineStyle: { width: 2, color: '#0077FA' },
          itemStyle: { color: '#0077FA' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(0, 119, 250, 0.3)' },
              { offset: 1, color: 'rgba(0, 119, 250, 0.05)' },
            ]),
          },
          data: amounts,
        },
        {
          name: '毛利',
          type: 'line',
          smooth: true,
          symbol: 'circle',
          symbolSize: 6,
          lineStyle: { width: 2, color: '#00B42A' },
          itemStyle: { color: '#00B42A' },
          data: profits,
        },
        {
          name: '订单数',
          type: 'bar',
          yAxisIndex: 1,
          barMaxWidth: 20,
          itemStyle: { color: 'rgba(22, 93, 255, 0.2)', borderRadius: [4, 4, 0, 0] },
          data: orders,
        },
      ],
    }
  }, [dailyTrends])

  // Build product composition chart options
  const productPieOptions = useMemo(() => {
    if (productRankings.length === 0) return null

    const data = productRankings.slice(0, 5).map((p) => ({
      name: p.product_name,
      value: p.total_amount,
    }))

    // Add "Others" if there are more than 5 products
    if (productRankings.length > 5) {
      const othersAmount = productRankings
        .slice(5)
        .reduce((sum, p) => sum + (p.total_amount ?? 0), 0)
      data.push({ name: '其他', value: othersAmount })
    }

    return {
      tooltip: {
        trigger: 'item',
        formatter: (params: { name: string; value: number; percent: number }) =>
          `${params.name}<br/>销售额: ${formatCurrency(params.value)}<br/>占比: ${safeToFixed(params.percent, 1)}%`,
      },
      legend: {
        orient: 'vertical',
        right: '5%',
        top: 'center',
        itemWidth: 12,
        itemHeight: 12,
      },
      series: [
        {
          type: 'pie',
          radius: ['40%', '70%'],
          center: ['35%', '50%'],
          avoidLabelOverlap: true,
          itemStyle: {
            borderRadius: 8,
            borderColor: '#fff',
            borderWidth: 2,
          },
          label: {
            show: false,
          },
          emphasis: {
            label: {
              show: true,
              fontSize: 14,
              fontWeight: 'bold',
            },
          },
          data,
        },
      ],
      color: ['#0077FA', '#00B42A', '#FF7D00', '#F53F3F', '#722ED1', '#86909C'],
    }
  }, [productRankings])

  // Product ranking table columns
  const productColumns = [
    {
      title: '排名',
      dataIndex: 'rank',
      key: 'rank',
      width: 60,
      render: (rank: number) => (
        <Tag color={rank <= 3 ? 'orange' : 'grey'} type="light">
          {rank}
        </Tag>
      ),
    },
    {
      title: '商品',
      dataIndex: 'product_name',
      key: 'product_name',
      render: (name: string, record: ProductSalesRanking) => (
        <div>
          <Text strong>{name}</Text>
          <br />
          <Text type="tertiary" size="small">
            {record.product_sku}
          </Text>
        </div>
      ),
    },
    {
      title: '销售额',
      dataIndex: 'total_amount',
      key: 'total_amount',
      align: 'right' as const,
      render: (amount: number) => formatCurrency(amount),
    },
    {
      title: '数量',
      dataIndex: 'total_quantity',
      key: 'total_quantity',
      align: 'right' as const,
      render: (qty: number) => formatNumber(qty),
    },
    {
      title: '毛利',
      dataIndex: 'total_profit',
      key: 'total_profit',
      align: 'right' as const,
      render: (profit: number) => (
        <Text style={{ color: profit >= 0 ? '#00B42A' : '#F53F3F' }}>{formatCurrency(profit)}</Text>
      ),
    },
  ]

  // Customer ranking table columns
  const customerColumns = [
    {
      title: '排名',
      dataIndex: 'rank',
      key: 'rank',
      width: 60,
      render: (rank: number) => (
        <Tag color={rank <= 3 ? 'orange' : 'grey'} type="light">
          {rank}
        </Tag>
      ),
    },
    {
      title: '客户',
      dataIndex: 'customer_name',
      key: 'customer_name',
    },
    {
      title: '销售额',
      dataIndex: 'total_amount',
      key: 'total_amount',
      align: 'right' as const,
      render: (amount: number) => formatCurrency(amount),
    },
    {
      title: '订单数',
      dataIndex: 'total_orders',
      key: 'total_orders',
      align: 'right' as const,
      render: (orders: number) => formatNumber(orders),
    },
    {
      title: '毛利',
      dataIndex: 'total_profit',
      key: 'total_profit',
      align: 'right' as const,
      render: (profit: number) => (
        <Text style={{ color: profit >= 0 ? '#00B42A' : '#F53F3F' }}>{formatCurrency(profit)}</Text>
      ),
    },
  ]

  return (
    <Container size="full" className="sales-report-page">
      <Spin spinning={loading} size="large">
        {/* Page Header with Filter */}
        <div className="report-header">
          <div className="report-title">
            <Title heading={3} style={{ margin: 0 }}>
              <IconLineChartStroked style={{ marginRight: 8 }} />
              销售报表
            </Title>
            <Text type="secondary">销售数据分析与趋势洞察</Text>
          </div>
          <div className="report-filters">
            <Select
              value={presetRange}
              onChange={handlePresetChange}
              style={{ width: 120 }}
              optionList={presetRangeOptions}
            />
            <DatePicker
              type="dateRange"
              value={dateRange}
              onChange={handleDateRangeChange}
              style={{ width: 260 }}
              format="yyyy-MM-dd"
            />
          </div>
        </div>

        {/* Summary Metrics Cards */}
        <div className="report-metrics">
          <Grid cols={{ mobile: 1, tablet: 2, desktop: 4 }} gap="md">
            <MetricCard
              title="销售总额"
              value={formatCurrency(summary?.total_sales_amount)}
              subLabel="平均订单"
              subValue={formatCurrency(summary?.avg_order_value)}
              icon={<IconPriceTag size="large" />}
              color="var(--semi-color-primary)"
            />
            <MetricCard
              title="订单数量"
              value={formatNumber(summary?.total_orders)}
              subLabel="销售数量"
              subValue={formatNumber(summary?.total_quantity)}
              icon={<IconShoppingBag size="large" />}
              color="var(--semi-color-success)"
            />
            <MetricCard
              title="毛利润"
              value={formatCurrency(summary?.total_gross_profit)}
              subLabel="毛利率"
              subValue={formatPercent(summary?.profit_margin)}
              icon={<IconTick size="large" />}
              color="var(--semi-color-warning)"
            />
            <MetricCard
              title="销售成本"
              value={formatCurrency(summary?.total_cost_amount)}
              subLabel="成本占比"
              subValue={
                summary?.total_sales_amount
                  ? formatPercent((summary?.total_cost_amount || 0) / summary.total_sales_amount)
                  : '0%'
              }
              icon={<IconPieChartStroked size="large" />}
              color="var(--semi-color-tertiary)"
            />
          </Grid>
        </div>

        {/* Charts Section */}
        <Row gap="md" wrap="wrap" className="report-charts">
          {/* Sales Trend Chart */}
          <div className="chart-container chart-trend">
            <Card title="销售趋势" className="chart-card">
              <Spin spinning={chartLoading}>
                {trendChartOptions ? (
                  <ReactEChartsCore
                    ref={chartRef}
                    echarts={echarts}
                    option={trendChartOptions}
                    style={{ height: 350 }}
                    notMerge={true}
                  />
                ) : (
                  <Empty description="暂无销售趋势数据" style={{ height: 350 }} />
                )}
              </Spin>
            </Card>
          </div>

          {/* Sales Composition Chart */}
          <div className="chart-container chart-composition">
            <Card title="商品销售构成" className="chart-card">
              <Spin spinning={chartLoading}>
                {productPieOptions ? (
                  <ReactEChartsCore
                    echarts={echarts}
                    option={productPieOptions}
                    style={{ height: 350 }}
                    notMerge={true}
                  />
                ) : (
                  <Empty description="暂无商品销售数据" style={{ height: 350 }} />
                )}
              </Spin>
            </Card>
          </div>
        </Row>

        {/* Rankings Section */}
        <Card className="rankings-card">
          <Tabs activeKey={rankingTab} onChange={setRankingTab}>
            <TabPane
              tab={
                <span>
                  <IconShoppingBag style={{ marginRight: 4 }} />
                  商品销售排行
                </span>
              }
              itemKey="products"
            >
              <Table
                columns={productColumns}
                dataSource={productRankings}
                rowKey={(record, index) => `${record.product_id}-${index}`}
                pagination={false}
                loading={loading}
                empty={<Empty description="暂无商品销售数据" />}
              />
            </TabPane>
            <TabPane
              tab={
                <span>
                  <IconUserGroup style={{ marginRight: 4 }} />
                  客户销售排行
                </span>
              }
              itemKey="customers"
            >
              <Table
                columns={customerColumns}
                dataSource={customerRankings}
                rowKey="customer_id"
                pagination={false}
                loading={loading}
                empty={<Empty description="暂无客户销售数据" />}
              />
            </TabPane>
          </Tabs>
        </Card>
      </Spin>
    </Container>
  )
}
