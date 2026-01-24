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
  Select,
  Input,
  Progress,
} from '@douyinfe/semi-ui'
import type { ColumnProps } from '@douyinfe/semi-ui/lib/es/table'
import {
  IconRefresh,
  IconSearch,
  IconPieChartStroked,
  IconBox,
  IconAlertTriangle,
} from '@douyinfe/semi-icons'
import ReactEChartsCore from 'echarts-for-react/lib/core'
import * as echarts from 'echarts/core'
import { BarChart, PieChart } from 'echarts/charts'
import {
  GridComponent,
  TooltipComponent,
  LegendComponent,
  TitleComponent,
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { Container, Grid, Row } from '@/components/common/layout'
import { getReports } from '@/api/reports'
import type {
  InventorySummary,
  InventoryTurnover as InventoryTurnoverType,
  InventoryValueByCategory,
  InventoryValueByWarehouse,
} from '@/api/reports'
import './InventoryTurnover.css'

// Register ECharts components
echarts.use([
  BarChart,
  PieChart,
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
 * Format turnover rate for display
 */
function formatTurnoverRate(rate?: number): string {
  if (rate === undefined || rate === null) return '0.00'
  return rate.toFixed(2)
}

/**
 * Format days of inventory
 */
function formatDays(days?: number): string {
  if (days === undefined || days === null || !isFinite(days)) return '--'
  return `${Math.round(days)}天`
}

/**
 * Get default date range (last 90 days for turnover analysis)
 */
function getDefaultDateRange(): [Date, Date] {
  const end = new Date()
  const start = new Date()
  start.setDate(start.getDate() - 90)
  return [start, end]
}

/**
 * Format date to YYYY-MM-DD
 */
function formatDateParam(date: Date): string {
  return date.toISOString().split('T')[0]
}

/**
 * Get turnover rate status
 */
function getTurnoverStatus(rate: number): {
  status: 'healthy' | 'slow' | 'critical'
  color: 'green' | 'amber' | 'red'
  label: string
} {
  if (rate >= 4) {
    return { status: 'healthy', color: 'green', label: '良好' }
  } else if (rate >= 2) {
    return { status: 'slow', color: 'amber', label: '偏慢' }
  } else {
    return { status: 'critical', color: 'red', label: '滞销' }
  }
}

/**
 * MetricCard component for displaying KPI metrics
 */
function MetricCard({ title, value, subValue, subLabel, icon, color }: MetricCardProps) {
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
          <Title heading={3} className="metric-value" style={{ margin: 0 }}>
            {value}
          </Title>
          {subValue !== undefined && subLabel && (
            <Text type="tertiary" size="small" className="metric-sub">
              {subLabel}: {subValue}
            </Text>
          )}
        </div>
      </div>
    </Card>
  )
}

/**
 * Inventory Turnover Report Page
 *
 * Displays inventory turnover metrics, charts, and detailed product data
 * with filtering by date range, category, and warehouse.
 */
export default function InventoryTurnoverPage() {
  // State
  const [dateRange, setDateRange] = useState<[Date, Date]>(getDefaultDateRange)
  const [loading, setLoading] = useState(true)
  const [searchText, setSearchText] = useState('')
  const [selectedCategory, setSelectedCategory] = useState<string | undefined>()
  const [selectedWarehouse, setSelectedWarehouse] = useState<string | undefined>()

  // Data state
  const [summary, setSummary] = useState<InventorySummary | null>(null)
  const [turnovers, setTurnovers] = useState<InventoryTurnoverType[]>([])
  const [valueByCategory, setValueByCategory] = useState<InventoryValueByCategory[]>([])
  const [valueByWarehouse, setValueByWarehouse] = useState<InventoryValueByWarehouse[]>([])

  // Refs
  const turnoverChartRef = useRef<ReactEChartsCore>(null)
  const categoryChartRef = useRef<ReactEChartsCore>(null)

  // API instance
  const api = useMemo(() => getReports(), [])

  /**
   * Fetch all data
   */
  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const params = {
        start_date: formatDateParam(dateRange[0]),
        end_date: formatDateParam(dateRange[1]),
        category_id: selectedCategory,
        warehouse_id: selectedWarehouse,
      }

      const [summaryRes, turnoverRes, categoryRes, warehouseRes] = await Promise.allSettled([
        api.getReportsInventorySummary(params),
        api.getReportsInventoryTurnover(params),
        api.getReportsInventoryValueByCategory(params),
        api.getReportsInventoryValueByWarehouse(params),
      ])

      if (summaryRes.status === 'fulfilled' && summaryRes.value.data) {
        setSummary(summaryRes.value.data as unknown as InventorySummary)
      } else {
        setSummary(null)
      }

      if (turnoverRes.status === 'fulfilled' && turnoverRes.value.data) {
        setTurnovers(turnoverRes.value.data as unknown as InventoryTurnoverType[])
      } else {
        setTurnovers([])
      }

      if (categoryRes.status === 'fulfilled' && categoryRes.value.data) {
        setValueByCategory(categoryRes.value.data as unknown as InventoryValueByCategory[])
      } else {
        setValueByCategory([])
      }

      if (warehouseRes.status === 'fulfilled' && warehouseRes.value.data) {
        setValueByWarehouse(warehouseRes.value.data as unknown as InventoryValueByWarehouse[])
      } else {
        setValueByWarehouse([])
      }
    } catch {
      Toast.error('获取库存周转数据失败')
    } finally {
      setLoading(false)
    }
  }, [api, dateRange, selectedCategory, selectedWarehouse])

  // Fetch data on mount and when filters change
  useEffect(() => {
    fetchData()
  }, [fetchData])

  /**
   * Handle date range change
   */
  const handleDateChange = useCallback(
    (dates: Date | Date[] | string | string[] | undefined) => {
      if (Array.isArray(dates) && dates.length === 2) {
        const [start, end] = dates as [Date | string, Date | string]
        const startDate = typeof start === 'string' ? new Date(start) : start
        const endDate = typeof end === 'string' ? new Date(end) : end
        setDateRange([startDate, endDate])
      }
    },
    []
  )

  /**
   * Apply preset date range
   */
  const applyPreset = useCallback((days: number) => {
    const end = new Date()
    const start = new Date()
    start.setDate(start.getDate() - days)
    setDateRange([start, end])
  }, [])

  /**
   * Filter turnovers by search text
   */
  const filteredTurnovers = useMemo(() => {
    if (!searchText) return turnovers
    const lowerSearch = searchText.toLowerCase()
    return turnovers.filter(
      (item) =>
        item.product_name.toLowerCase().includes(lowerSearch) ||
        item.product_sku.toLowerCase().includes(lowerSearch) ||
        (item.category_name && item.category_name.toLowerCase().includes(lowerSearch))
    )
  }, [turnovers, searchText])

  /**
   * Turnover distribution chart options
   */
  const turnoverChartOptions = useMemo(() => {
    // Group by turnover rate ranges
    const ranges = [
      { name: '滞销 (<2)', min: 0, max: 2, count: 0, color: '#ef4444' },
      { name: '偏慢 (2-4)', min: 2, max: 4, count: 0, color: '#f59e0b' },
      { name: '正常 (4-8)', min: 4, max: 8, count: 0, color: '#22c55e' },
      { name: '良好 (8-12)', min: 8, max: 12, count: 0, color: '#3b82f6' },
      { name: '优秀 (>12)', min: 12, max: Infinity, count: 0, color: '#8b5cf6' },
    ]

    turnovers.forEach((item) => {
      const rate = item.turnover_rate
      for (const range of ranges) {
        if (rate >= range.min && rate < range.max) {
          range.count++
          break
        }
      }
    })

    return {
      tooltip: {
        trigger: 'item',
        formatter: '{b}: {c}个商品 ({d}%)',
      },
      legend: {
        orient: 'vertical',
        right: 10,
        top: 'center',
      },
      series: [
        {
          type: 'pie',
          radius: ['40%', '70%'],
          avoidLabelOverlap: false,
          itemStyle: {
            borderRadius: 10,
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
          data: ranges
            .filter((r) => r.count > 0)
            .map((r) => ({
              name: r.name,
              value: r.count,
              itemStyle: { color: r.color },
            })),
        },
      ],
    }
  }, [turnovers])

  /**
   * Category value chart options
   */
  const categoryChartOptions = useMemo(() => {
    const sortedData = [...valueByCategory]
      .sort((a, b) => b.total_value - a.total_value)
      .slice(0, 10)

    return {
      tooltip: {
        trigger: 'axis',
        axisPointer: {
          type: 'shadow',
        },
        formatter: (params: { name: string; value: number }[]) => {
          const data = params[0]
          return `${data.name}<br/>库存价值: ${formatCurrency(data.value)}`
        },
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '3%',
        containLabel: true,
      },
      xAxis: {
        type: 'value',
        axisLabel: {
          formatter: (value: number) =>
            value >= 10000 ? `${(value / 10000).toFixed(0)}万` : value.toString(),
        },
      },
      yAxis: {
        type: 'category',
        data: sortedData.map((item) => item.category_name),
        axisLabel: {
          width: 80,
          overflow: 'truncate',
        },
      },
      series: [
        {
          type: 'bar',
          data: sortedData.map((item) => item.total_value),
          itemStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
              { offset: 0, color: '#6366f1' },
              { offset: 1, color: '#8b5cf6' },
            ]),
            borderRadius: [0, 4, 4, 0],
          },
          label: {
            show: true,
            position: 'right',
            formatter: (params: { value: number }) => formatCurrency(params.value),
            fontSize: 10,
          },
        },
      ],
    }
  }, [valueByCategory])

  /**
   * Table columns
   */
  const columns: ColumnProps<InventoryTurnoverType>[] = useMemo(
    () => [
      {
        title: '商品信息',
        dataIndex: 'product_name',
        width: 240,
        render: (text: string, record?: InventoryTurnoverType) => (
          <div className="product-info-cell">
            <Text strong ellipsis={{ showTooltip: true }}>
              {text}
            </Text>
            <Text type="tertiary" size="small">
              {record?.product_sku}
            </Text>
          </div>
        ),
      },
      {
        title: '分类',
        dataIndex: 'category_name',
        width: 120,
        render: (text: string) => text || '--',
      },
      {
        title: '仓库',
        dataIndex: 'warehouse_name',
        width: 120,
        render: (text: string) => text || '--',
      },
      {
        title: '期初库存',
        dataIndex: 'beginning_stock',
        width: 100,
        align: 'right' as const,
        sorter: (a?: InventoryTurnoverType, b?: InventoryTurnoverType) =>
          (a?.beginning_stock || 0) - (b?.beginning_stock || 0),
        render: (val: number) => formatNumber(val),
      },
      {
        title: '期末库存',
        dataIndex: 'ending_stock',
        width: 100,
        align: 'right' as const,
        sorter: (a?: InventoryTurnoverType, b?: InventoryTurnoverType) =>
          (a?.ending_stock || 0) - (b?.ending_stock || 0),
        render: (val: number) => formatNumber(val),
      },
      {
        title: '平均库存',
        dataIndex: 'average_stock',
        width: 100,
        align: 'right' as const,
        sorter: (a?: InventoryTurnoverType, b?: InventoryTurnoverType) =>
          (a?.average_stock || 0) - (b?.average_stock || 0),
        render: (val: number) => formatNumber(val),
      },
      {
        title: '销售数量',
        dataIndex: 'sold_quantity',
        width: 100,
        align: 'right' as const,
        sorter: (a?: InventoryTurnoverType, b?: InventoryTurnoverType) =>
          (a?.sold_quantity || 0) - (b?.sold_quantity || 0),
        render: (val: number) => formatNumber(val),
      },
      {
        title: '周转率',
        dataIndex: 'turnover_rate',
        width: 120,
        align: 'center' as const,
        sorter: (a?: InventoryTurnoverType, b?: InventoryTurnoverType) =>
          (a?.turnover_rate || 0) - (b?.turnover_rate || 0),
        render: (val: number) => {
          const status = getTurnoverStatus(val)
          return (
            <div className="turnover-rate-cell">
              <Text strong>{formatTurnoverRate(val)}</Text>
              <Tag color={status.color} size="small">
                {status.label}
              </Tag>
            </div>
          )
        },
      },
      {
        title: '库存天数',
        dataIndex: 'days_of_inventory',
        width: 100,
        align: 'right' as const,
        sorter: (a?: InventoryTurnoverType, b?: InventoryTurnoverType) =>
          (a?.days_of_inventory || 0) - (b?.days_of_inventory || 0),
        render: (val: number) => formatDays(val),
      },
      {
        title: '库存价值',
        dataIndex: 'stock_value',
        width: 120,
        align: 'right' as const,
        sorter: (a?: InventoryTurnoverType, b?: InventoryTurnoverType) =>
          (a?.stock_value || 0) - (b?.stock_value || 0),
        render: (val: number) => formatCurrency(val),
      },
    ],
    []
  )

  /**
   * Category options for filter
   */
  const categoryOptions = useMemo(
    () =>
      valueByCategory.map((cat) => ({
        value: cat.category_id || '',
        label: cat.category_name,
      })),
    [valueByCategory]
  )

  /**
   * Warehouse options for filter
   */
  const warehouseOptions = useMemo(
    () =>
      valueByWarehouse.map((wh) => ({
        value: wh.warehouse_id,
        label: wh.warehouse_name,
      })),
    [valueByWarehouse]
  )

  return (
    <Container className="inventory-turnover-page">
      {/* Page Header */}
      <div className="report-header">
        <div className="report-title">
          <Title heading={2} style={{ margin: 0 }}>
            库存周转报表
          </Title>
          <Text type="tertiary">分析库存周转效率，优化库存管理</Text>
        </div>
        <div className="report-filters">
          <Select
            placeholder="全部分类"
            style={{ width: 160 }}
            value={selectedCategory}
            onChange={(val) => setSelectedCategory(val as string | undefined)}
            optionList={categoryOptions}
            showClear
          />
          <Select
            placeholder="全部仓库"
            style={{ width: 160 }}
            value={selectedWarehouse}
            onChange={(val) => setSelectedWarehouse(val as string | undefined)}
            optionList={warehouseOptions}
            showClear
          />
          <div className="date-presets">
            <Tag onClick={() => applyPreset(30)} className="preset-tag">
              近30天
            </Tag>
            <Tag onClick={() => applyPreset(90)} className="preset-tag">
              近90天
            </Tag>
            <Tag onClick={() => applyPreset(180)} className="preset-tag">
              近半年
            </Tag>
            <Tag onClick={() => applyPreset(365)} className="preset-tag">
              近一年
            </Tag>
          </div>
          <DatePicker
            type="dateRange"
            value={dateRange}
            onChange={handleDateChange}
            style={{ width: 260 }}
            density="compact"
          />
        </div>
      </div>

      {loading ? (
        <div className="loading-container">
          <Spin size="large" />
          <Text type="tertiary">加载库存周转数据...</Text>
        </div>
      ) : (
        <>
          {/* Summary Metrics */}
          <div className="report-metrics">
            <Grid cols={{ mobile: 1, tablet: 2, desktop: 4 }} gap="md">
              <MetricCard
                title="商品总数"
                value={formatNumber(summary?.total_products)}
                icon={<IconBox size="large" />}
                color="#6366f1"
              />
              <MetricCard
                title="库存总值"
                value={formatCurrency(summary?.total_value)}
                icon={<IconPieChartStroked size="large" />}
                color="#8b5cf6"
              />
              <MetricCard
                title="平均周转率"
                value={formatTurnoverRate(summary?.avg_turnover_rate)}
                subLabel="周转次数/期"
                icon={<IconRefresh size="large" />}
                color="#22c55e"
              />
              <MetricCard
                title="滞销商品"
                value={formatNumber(summary?.low_stock_count)}
                subValue={summary?.out_of_stock_count}
                subLabel="缺货商品"
                icon={<IconAlertTriangle size="large" />}
                color="#ef4444"
              />
            </Grid>
          </div>

          {/* Charts Section */}
          <div className="report-charts">
            <Row gap="md" wrap="wrap">
              <div className="chart-container chart-pie">
                <Card
                  title="周转率分布"
                  className="chart-card"
                  headerExtraContent={
                    <Text type="tertiary" size="small">
                      按周转率等级分组
                    </Text>
                  }
                >
                  {turnovers.length > 0 ? (
                    <ReactEChartsCore
                      ref={turnoverChartRef}
                      echarts={echarts}
                      option={turnoverChartOptions}
                      style={{ height: 300 }}
                      notMerge
                      lazyUpdate
                    />
                  ) : (
                    <Empty description="暂无数据" />
                  )}
                </Card>
              </div>
              <div className="chart-container chart-bar">
                <Card
                  title="分类库存价值"
                  className="chart-card"
                  headerExtraContent={
                    <Text type="tertiary" size="small">
                      Top 10 分类
                    </Text>
                  }
                >
                  {valueByCategory.length > 0 ? (
                    <ReactEChartsCore
                      ref={categoryChartRef}
                      echarts={echarts}
                      option={categoryChartOptions}
                      style={{ height: 300 }}
                      notMerge
                      lazyUpdate
                    />
                  ) : (
                    <Empty description="暂无数据" />
                  )}
                </Card>
              </div>
            </Row>
          </div>

          {/* Warehouse Summary */}
          {valueByWarehouse.length > 0 && (
            <Card title="各仓库库存分布" className="warehouse-card">
              <div className="warehouse-grid">
                {valueByWarehouse.map((wh) => (
                  <div key={wh.warehouse_id} className="warehouse-item">
                    <div className="warehouse-header">
                      <Text strong>{wh.warehouse_name}</Text>
                      <Text type="tertiary" size="small">
                        {wh.product_count}个商品
                      </Text>
                    </div>
                    <div className="warehouse-value">
                      <Text>{formatCurrency(wh.total_value)}</Text>
                    </div>
                    <Progress
                      percent={wh.percentage * 100}
                      size="small"
                      showInfo={false}
                      style={{ marginTop: 8 }}
                    />
                    <Text type="tertiary" size="small">
                      占比 {(wh.percentage * 100).toFixed(1)}%
                    </Text>
                  </div>
                ))}
              </div>
            </Card>
          )}

          {/* Detail Table */}
          <Card
            title="商品周转明细"
            className="detail-card"
            headerExtraContent={
              <Input
                prefix={<IconSearch />}
                placeholder="搜索商品名称、SKU或分类"
                style={{ width: 280 }}
                value={searchText}
                onChange={setSearchText}
                showClear
              />
            }
          >
            <Table
              columns={columns}
              dataSource={filteredTurnovers}
              rowKey="product_id"
              pagination={{
                pageSize: 20,
                showSizeChanger: true,
                pageSizeOpts: [10, 20, 50, 100],
                showTotal: true,
              }}
              scroll={{ x: 1200 }}
              empty={<Empty description="暂无库存周转数据" />}
            />
          </Card>
        </>
      )}
    </Container>
  )
}
