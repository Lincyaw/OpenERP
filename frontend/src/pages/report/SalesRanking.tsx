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
  Tabs,
  TabPane,
  Select,
  Radio,
  RadioGroup,
} from '@douyinfe/semi-ui'
import {
  IconBarChartHStroked,
  IconShoppingBag,
  IconUserGroup,
  IconStar,
} from '@douyinfe/semi-icons'
import ReactEChartsCore from 'echarts-for-react/lib/core'
import * as echarts from 'echarts/core'
import { BarChart } from 'echarts/charts'
import {
  GridComponent,
  TooltipComponent,
  LegendComponent,
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { Container, Row, Grid } from '@/components/common/layout'
import { getReports } from '@/api/reports'
import type {
  ProductSalesRanking,
  CustomerSalesRanking,
} from '@/api/reports'
import type { TagColor } from '@douyinfe/semi-ui/lib/es/tag'
import './SalesRanking.css'

// Register ECharts components
echarts.use([
  BarChart,
  GridComponent,
  TooltipComponent,
  LegendComponent,
  CanvasRenderer,
])

const { Title, Text } = Typography

type RankingDimension = 'amount' | 'quantity' | 'profit'
type RankingTab = 'products' | 'customers'

interface RankingMetrics {
  totalProducts: number
  totalCustomers: number
  topProductAmount: number
  topCustomerAmount: number
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
 * Get dimension value from ranking item
 */
function getDimensionValue(
  item: ProductSalesRanking | CustomerSalesRanking,
  dimension: RankingDimension
): number {
  switch (dimension) {
    case 'amount':
      return item.total_amount
    case 'quantity':
      return item.total_quantity
    case 'profit':
      return item.total_profit
    default:
      return item.total_amount
  }
}

/**
 * Get dimension label
 */
function getDimensionLabel(dimension: RankingDimension): string {
  switch (dimension) {
    case 'amount':
      return '销售额'
    case 'quantity':
      return '销售数量'
    case 'profit':
      return '毛利'
    default:
      return '销售额'
  }
}

/**
 * Format dimension value for display
 */
function formatDimensionValue(value: number, dimension: RankingDimension): string {
  switch (dimension) {
    case 'amount':
    case 'profit':
      return formatCurrency(value)
    case 'quantity':
      return formatNumber(value)
    default:
      return formatCurrency(value)
  }
}

/**
 * MetricCard component for summary metrics
 */
interface MetricCardProps {
  title: string
  value: string | number
  icon: React.ReactNode
  color: string
}

function MetricCard({ title, value, icon, color }: MetricCardProps) {
  return (
    <Card className="ranking-metric-card">
      <div className="ranking-metric-content">
        <div className="ranking-metric-icon" style={{ backgroundColor: color + '15', color }}>
          {icon}
        </div>
        <div className="ranking-metric-info">
          <Text type="tertiary" className="ranking-metric-label">
            {title}
          </Text>
          <Title heading={4} className="ranking-metric-value" style={{ margin: 0 }}>
            {value}
          </Title>
        </div>
      </div>
    </Card>
  )
}

/**
 * Sales Ranking Page
 *
 * Features (P5-FE-002):
 * - Product sales ranking with dimension switching
 * - Customer sales ranking with dimension switching
 * - Horizontal bar charts for visual comparison
 * - Date range filter with presets
 */
export default function SalesRankingPage() {
  const reportsApi = useMemo(() => getReports(), [])

  // Date range state
  const [dateRange, setDateRange] = useState<[Date, Date]>(getDefaultDateRange)
  const [presetRange, setPresetRange] = useState<string>('30days')

  // Loading states
  const [loading, setLoading] = useState(true)

  // Data states
  const [productRankings, setProductRankings] = useState<ProductSalesRanking[]>([])
  const [customerRankings, setCustomerRankings] = useState<CustomerSalesRanking[]>([])

  // UI states
  const [activeTab, setActiveTab] = useState<RankingTab>('products')
  const [dimension, setDimension] = useState<RankingDimension>('amount')
  const [topN, setTopN] = useState<number>(20)

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

  // Top N options
  const topNOptions = [
    { value: 10, label: 'Top 10' },
    { value: 20, label: 'Top 20' },
    { value: 50, label: 'Top 50' },
    { value: 100, label: 'Top 100' },
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

  // Handle top N change
  const handleTopNChange = (
    value: string | number | (string | number)[] | Record<string, unknown> | undefined
  ) => {
    if (typeof value === 'number') {
      setTopN(value)
    }
  }

  // Fetch ranking data
  const fetchRankingData = useCallback(async () => {
    setLoading(true)

    const params = {
      start_date: formatDateParam(dateRange[0]),
      end_date: formatDateParam(dateRange[1]),
      top_n: topN,
    }

    try {
      const [productsRes, customersRes] = await Promise.allSettled([
        reportsApi.getReportsSalesProductsRanking(params),
        reportsApi.getReportsSalesCustomersRanking(params),
      ])

      if (productsRes.status === 'fulfilled' && productsRes.value.data) {
        setProductRankings(productsRes.value.data)
      } else {
        setProductRankings([])
      }

      if (customersRes.status === 'fulfilled' && customersRes.value.data) {
        setCustomerRankings(customersRes.value.data)
      } else {
        setCustomerRankings([])
      }
    } catch {
      Toast.error('获取排行榜数据失败')
    } finally {
      setLoading(false)
    }
  }, [reportsApi, dateRange, topN])

  // Fetch data on mount and when dependencies change
  useEffect(() => {
    fetchRankingData()
  }, [fetchRankingData])

  // Calculate summary metrics
  const metrics: RankingMetrics = useMemo(() => {
    return {
      totalProducts: productRankings.length,
      totalCustomers: customerRankings.length,
      topProductAmount: productRankings[0]?.total_amount || 0,
      topCustomerAmount: customerRankings[0]?.total_amount || 0,
    }
  }, [productRankings, customerRankings])

  // Sort rankings by selected dimension
  const sortedProductRankings = useMemo(() => {
    return [...productRankings].sort(
      (a, b) => getDimensionValue(b, dimension) - getDimensionValue(a, dimension)
    )
  }, [productRankings, dimension])

  const sortedCustomerRankings = useMemo(() => {
    return [...customerRankings].sort(
      (a, b) => getDimensionValue(b, dimension) - getDimensionValue(a, dimension)
    )
  }, [customerRankings, dimension])

  // Build product bar chart options
  const productChartOptions = useMemo(() => {
    if (sortedProductRankings.length === 0) return null

    const top10 = sortedProductRankings.slice(0, 10)
    const names = top10.map((p) => p.product_name.length > 12
      ? p.product_name.substring(0, 12) + '...'
      : p.product_name
    ).reverse()
    const values = top10.map((p) => getDimensionValue(p, dimension)).reverse()

    return {
      tooltip: {
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
        formatter: (params: Array<{ name: string; value: number }>) => {
          const param = params[0]
          const originalItem = top10.find((p) =>
            p.product_name.startsWith(param.name.replace('...', ''))
          )
          return `${originalItem?.product_name || param.name}<br/>${getDimensionLabel(dimension)}: ${formatDimensionValue(param.value, dimension)}`
        },
      },
      grid: {
        left: '3%',
        right: '4%',
        top: '3%',
        bottom: '3%',
        containLabel: true,
      },
      xAxis: {
        type: 'value',
        axisLine: { show: false },
        axisTick: { show: false },
        splitLine: { lineStyle: { color: '#E5E6EB', type: 'dashed' } },
        axisLabel: {
          color: '#86909C',
          formatter: (value: number) => {
            if (dimension === 'quantity') return formatNumber(value)
            if (value >= 10000) return `${(value / 10000).toFixed(0)}万`
            return value.toString()
          },
        },
      },
      yAxis: {
        type: 'category',
        data: names,
        axisLine: { lineStyle: { color: '#E5E6EB' } },
        axisTick: { show: false },
        axisLabel: { color: '#1D2129', fontSize: 12 },
      },
      series: [
        {
          type: 'bar',
          data: values,
          barWidth: 16,
          itemStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
              { offset: 0, color: '#0077FA' },
              { offset: 1, color: '#00B4FA' },
            ]),
            borderRadius: [0, 4, 4, 0],
          },
          emphasis: {
            itemStyle: {
              color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
                { offset: 0, color: '#0057BA' },
                { offset: 1, color: '#0097FA' },
              ]),
            },
          },
        },
      ],
    }
  }, [sortedProductRankings, dimension])

  // Build customer bar chart options
  const customerChartOptions = useMemo(() => {
    if (sortedCustomerRankings.length === 0) return null

    const top10 = sortedCustomerRankings.slice(0, 10)
    const names = top10.map((c) => c.customer_name.length > 12
      ? c.customer_name.substring(0, 12) + '...'
      : c.customer_name
    ).reverse()
    const values = top10.map((c) => getDimensionValue(c, dimension)).reverse()

    return {
      tooltip: {
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
        formatter: (params: Array<{ name: string; value: number }>) => {
          const param = params[0]
          const originalItem = top10.find((c) =>
            c.customer_name.startsWith(param.name.replace('...', ''))
          )
          return `${originalItem?.customer_name || param.name}<br/>${getDimensionLabel(dimension)}: ${formatDimensionValue(param.value, dimension)}`
        },
      },
      grid: {
        left: '3%',
        right: '4%',
        top: '3%',
        bottom: '3%',
        containLabel: true,
      },
      xAxis: {
        type: 'value',
        axisLine: { show: false },
        axisTick: { show: false },
        splitLine: { lineStyle: { color: '#E5E6EB', type: 'dashed' } },
        axisLabel: {
          color: '#86909C',
          formatter: (value: number) => {
            if (dimension === 'quantity') return formatNumber(value)
            if (value >= 10000) return `${(value / 10000).toFixed(0)}万`
            return value.toString()
          },
        },
      },
      yAxis: {
        type: 'category',
        data: names,
        axisLine: { lineStyle: { color: '#E5E6EB' } },
        axisTick: { show: false },
        axisLabel: { color: '#1D2129', fontSize: 12 },
      },
      series: [
        {
          type: 'bar',
          data: values,
          barWidth: 16,
          itemStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
              { offset: 0, color: '#00B42A' },
              { offset: 1, color: '#7BE188' },
            ]),
            borderRadius: [0, 4, 4, 0],
          },
          emphasis: {
            itemStyle: {
              color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
                { offset: 0, color: '#009920' },
                { offset: 1, color: '#5BD168' },
              ]),
            },
          },
        },
      ],
    }
  }, [sortedCustomerRankings, dimension])

  // Product ranking table columns
  const productColumns = [
    {
      title: '排名',
      dataIndex: 'rank',
      key: 'rank',
      width: 70,
      render: (_: unknown, __: unknown, index: number) => {
        const rank = index + 1
        let color: TagColor = 'grey'
        if (rank === 1) color = 'amber'
        else if (rank === 2) color = 'white'
        else if (rank === 3) color = 'orange'

        return (
          <Tag color={color} type={rank <= 3 ? 'solid' : 'light'}>
            {rank <= 3 ? <IconStar size="small" style={{ marginRight: 2 }} /> : null}
            {rank}
          </Tag>
        )
      },
    },
    {
      title: '商品',
      dataIndex: 'product_name',
      key: 'product_name',
      render: (name: string, record: ProductSalesRanking) => (
        <div>
          <Text strong ellipsis={{ showTooltip: true }} style={{ maxWidth: 200 }}>
            {name}
          </Text>
          <br />
          <Text type="tertiary" size="small">
            {record.product_sku}
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
      title: '销售额',
      dataIndex: 'total_amount',
      key: 'total_amount',
      align: 'right' as const,
      sorter: (a?: ProductSalesRanking, b?: ProductSalesRanking) => (a?.total_amount ?? 0) - (b?.total_amount ?? 0),
      render: (amount: number) => (
        <Text strong={dimension === 'amount'}>{formatCurrency(amount)}</Text>
      ),
    },
    {
      title: '销售数量',
      dataIndex: 'total_quantity',
      key: 'total_quantity',
      align: 'right' as const,
      sorter: (a?: ProductSalesRanking, b?: ProductSalesRanking) => (a?.total_quantity ?? 0) - (b?.total_quantity ?? 0),
      render: (qty: number) => (
        <Text strong={dimension === 'quantity'}>{formatNumber(qty)}</Text>
      ),
    },
    {
      title: '订单数',
      dataIndex: 'order_count',
      key: 'order_count',
      align: 'right' as const,
      render: (count: number) => formatNumber(count),
    },
    {
      title: '毛利',
      dataIndex: 'total_profit',
      key: 'total_profit',
      align: 'right' as const,
      sorter: (a?: ProductSalesRanking, b?: ProductSalesRanking) => (a?.total_profit ?? 0) - (b?.total_profit ?? 0),
      render: (profit: number) => (
        <Text
          strong={dimension === 'profit'}
          style={{ color: profit >= 0 ? '#00B42A' : '#F53F3F' }}
        >
          {formatCurrency(profit)}
        </Text>
      ),
    },
  ]

  // Customer ranking table columns
  const customerColumns = [
    {
      title: '排名',
      dataIndex: 'rank',
      key: 'rank',
      width: 70,
      render: (_: unknown, __: unknown, index: number) => {
        const rank = index + 1
        let color: TagColor = 'grey'
        if (rank === 1) color = 'amber'
        else if (rank === 2) color = 'white'
        else if (rank === 3) color = 'orange'

        return (
          <Tag color={color} type={rank <= 3 ? 'solid' : 'light'}>
            {rank <= 3 ? <IconStar size="small" style={{ marginRight: 2 }} /> : null}
            {rank}
          </Tag>
        )
      },
    },
    {
      title: '客户',
      dataIndex: 'customer_name',
      key: 'customer_name',
      render: (name: string) => (
        <Text strong ellipsis={{ showTooltip: true }} style={{ maxWidth: 200 }}>
          {name}
        </Text>
      ),
    },
    {
      title: '销售额',
      dataIndex: 'total_amount',
      key: 'total_amount',
      align: 'right' as const,
      sorter: (a?: CustomerSalesRanking, b?: CustomerSalesRanking) => (a?.total_amount ?? 0) - (b?.total_amount ?? 0),
      render: (amount: number) => (
        <Text strong={dimension === 'amount'}>{formatCurrency(amount)}</Text>
      ),
    },
    {
      title: '销售数量',
      dataIndex: 'total_quantity',
      key: 'total_quantity',
      align: 'right' as const,
      sorter: (a?: CustomerSalesRanking, b?: CustomerSalesRanking) => (a?.total_quantity ?? 0) - (b?.total_quantity ?? 0),
      render: (qty: number) => (
        <Text strong={dimension === 'quantity'}>{formatNumber(qty)}</Text>
      ),
    },
    {
      title: '订单数',
      dataIndex: 'total_orders',
      key: 'total_orders',
      align: 'right' as const,
      render: (count: number) => formatNumber(count),
    },
    {
      title: '毛利',
      dataIndex: 'total_profit',
      key: 'total_profit',
      align: 'right' as const,
      sorter: (a?: CustomerSalesRanking, b?: CustomerSalesRanking) => (a?.total_profit ?? 0) - (b?.total_profit ?? 0),
      render: (profit: number) => (
        <Text
          strong={dimension === 'profit'}
          style={{ color: profit >= 0 ? '#00B42A' : '#F53F3F' }}
        >
          {formatCurrency(profit)}
        </Text>
      ),
    },
  ]

  return (
    <Container size="full" className="sales-ranking-page">
      <Spin spinning={loading} size="large">
        {/* Page Header with Filters */}
        <div className="ranking-header">
          <div className="ranking-title">
            <Title heading={3} style={{ margin: 0 }}>
              <IconBarChartHStroked style={{ marginRight: 8 }} />
              销售排行榜
            </Title>
            <Text type="secondary">商品与客户销售表现对比分析</Text>
          </div>
          <div className="ranking-filters">
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
            <Select
              value={topN}
              onChange={handleTopNChange}
              style={{ width: 100 }}
              optionList={topNOptions}
            />
          </div>
        </div>

        {/* Summary Metrics */}
        <div className="ranking-metrics">
          <Grid cols={{ mobile: 2, tablet: 4 }} gap="md">
            <MetricCard
              title="参与商品数"
              value={formatNumber(metrics.totalProducts)}
              icon={<IconShoppingBag size="large" />}
              color="var(--semi-color-primary)"
            />
            <MetricCard
              title="参与客户数"
              value={formatNumber(metrics.totalCustomers)}
              icon={<IconUserGroup size="large" />}
              color="var(--semi-color-success)"
            />
            <MetricCard
              title="商品冠军销售额"
              value={formatCurrency(metrics.topProductAmount)}
              icon={<IconStar size="large" />}
              color="var(--semi-color-warning)"
            />
            <MetricCard
              title="客户冠军销售额"
              value={formatCurrency(metrics.topCustomerAmount)}
              icon={<IconStar size="large" />}
              color="var(--semi-color-danger)"
            />
          </Grid>
        </div>

        {/* Dimension Selector */}
        <Card className="dimension-card">
          <div className="dimension-selector">
            <Text strong style={{ marginRight: 16 }}>排序维度：</Text>
            <RadioGroup
              type="button"
              value={dimension}
              onChange={(e) => setDimension(e.target.value as RankingDimension)}
            >
              <Radio value="amount">销售额</Radio>
              <Radio value="quantity">销售数量</Radio>
              <Radio value="profit">毛利</Radio>
            </RadioGroup>
          </div>
        </Card>

        {/* Main Content with Tabs */}
        <Card className="ranking-content-card">
          <Tabs activeKey={activeTab} onChange={(key) => setActiveTab(key as RankingTab)}>
            <TabPane
              tab={
                <span>
                  <IconShoppingBag style={{ marginRight: 4 }} />
                  商品销售排行
                </span>
              }
              itemKey="products"
            >
              <Row gap="lg" wrap="wrap" className="ranking-content">
                {/* Chart */}
                <div className="ranking-chart-container">
                  <Card title={`商品${getDimensionLabel(dimension)}Top 10`} className="ranking-chart-card">
                    {productChartOptions ? (
                      <ReactEChartsCore
                        echarts={echarts}
                        option={productChartOptions}
                        style={{ height: 350 }}
                        notMerge={true}
                      />
                    ) : (
                      <Empty description="暂无商品排行数据" style={{ height: 350 }} />
                    )}
                  </Card>
                </div>

                {/* Table */}
                <div className="ranking-table-container">
                  <Table
                    columns={productColumns}
                    dataSource={sortedProductRankings}
                    rowKey="product_id"
                    pagination={{
                      pageSize: 10,
                      showTotal: true,
                      showSizeChanger: true,
                      pageSizeOpts: [10, 20, 50],
                    }}
                    loading={loading}
                    empty={<Empty description="暂无商品销售数据" />}
                  />
                </div>
              </Row>
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
              <Row gap="lg" wrap="wrap" className="ranking-content">
                {/* Chart */}
                <div className="ranking-chart-container">
                  <Card title={`客户${getDimensionLabel(dimension)}Top 10`} className="ranking-chart-card">
                    {customerChartOptions ? (
                      <ReactEChartsCore
                        echarts={echarts}
                        option={customerChartOptions}
                        style={{ height: 350 }}
                        notMerge={true}
                      />
                    ) : (
                      <Empty description="暂无客户排行数据" style={{ height: 350 }} />
                    )}
                  </Card>
                </div>

                {/* Table */}
                <div className="ranking-table-container">
                  <Table
                    columns={customerColumns}
                    dataSource={sortedCustomerRankings}
                    rowKey="customer_id"
                    pagination={{
                      pageSize: 10,
                      showTotal: true,
                      showSizeChanger: true,
                      pageSizeOpts: [10, 20, 50],
                    }}
                    loading={loading}
                    empty={<Empty description="暂无客户销售数据" />}
                  />
                </div>
              </Row>
            </TabPane>
          </Tabs>
        </Card>
      </Spin>
    </Container>
  )
}
