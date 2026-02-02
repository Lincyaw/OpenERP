/**
 * Platform Statistics Page
 *
 * Displays platform-wide statistics and metrics for super admin:
 * - Key indicator cards (total tenants, active tenants, new this month, total revenue)
 * - Tenant growth trend chart (line chart)
 * - Plan distribution pie chart
 * - Status distribution pie chart
 * - Recently registered tenants (top 10)
 * - Recently suspended tenants (top 10)
 * - Time range selector (7d/30d/90d/all)
 */
import { useState, useMemo, useCallback } from 'react'
import {
  Typography,
  Card,
  Row,
  Col,
  Space,
  Select,
  Spin,
  Empty,
  Table,
  Tag,
} from '@douyinfe/semi-ui-19'
import type { TagColor } from '@douyinfe/semi-ui-19/lib/es/tag'
import { IconUserGroup, IconTick, IconPlus, IconCalendar, IconStop } from '@douyinfe/semi-icons'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import ReactEChartsCore from 'echarts-for-react/lib/core'
import * as echarts from 'echarts/core'
import { LineChart, PieChart } from 'echarts/charts'
import {
  GridComponent,
  TooltipComponent,
  LegendComponent,
  DataZoomComponent,
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import type { EChartsOption } from 'echarts'

import { Container } from '@/components/common/layout'
import { useAdminPlatformStats, useAdminGrowthTrend, useAdminTenants } from '@/store/adminStore'
import type { TenantGrowthParams } from '@/api/admin'
import type { HandlerTenantResponse } from '@/api/models'
import { useFormatters } from '@/hooks/useFormatters'

// Register ECharts components
echarts.use([
  LineChart,
  PieChart,
  GridComponent,
  TooltipComponent,
  LegendComponent,
  DataZoomComponent,
  CanvasRenderer,
])

const { Title, Text } = Typography

// ============================================================================
// Types
// ============================================================================

type TimeRange = '7d' | '30d' | '90d' | 'all'

interface TimeRangeOption {
  value: TimeRange
  label: string
  days: number
  period: 'daily' | 'weekly' | 'monthly'
}

// ============================================================================
// Constants
// ============================================================================

const STATUS_COLOR_MAP: Record<string, TagColor> = {
  active: 'green',
  trial: 'blue',
  suspended: 'red',
  inactive: 'grey',
}

const PLAN_COLOR_MAP: Record<string, string> = {
  free: '#95a5a6',
  basic: '#3498db',
  pro: '#2ecc71',
  enterprise: '#9b59b6',
}

// ============================================================================
// Stat Card Component
// ============================================================================

interface StatCardProps {
  title: string
  value: string | number
  icon: React.ReactNode
  color: string
  loading?: boolean
}

function StatCard({ title, value, icon, color, loading }: StatCardProps) {
  return (
    <Card bodyStyle={{ padding: '20px 24px' }}>
      {loading ? (
        <div style={{ display: 'flex', justifyContent: 'center', padding: 16 }}>
          <Spin />
        </div>
      ) : (
        <Space>
          <div
            style={{
              width: 48,
              height: 48,
              borderRadius: 8,
              background: `${color}20`,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color,
            }}
          >
            {icon}
          </div>
          <div>
            <Text type="secondary" size="small">
              {title}
            </Text>
            <Title heading={3} style={{ margin: 0 }}>
              {typeof value === 'number' ? value.toLocaleString() : value}
            </Title>
          </div>
        </Space>
      )}
    </Card>
  )
}

// ============================================================================
// Main Component
// ============================================================================

export default function StatsPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { formatDate } = useFormatters()

  // Time range state
  const [timeRange, setTimeRange] = useState<TimeRange>('30d')

  const timeRangeOptions: TimeRangeOption[] = useMemo(
    () => [
      { value: '7d', label: t('admin.stats.range7d', '7 Days'), days: 7, period: 'daily' as const },
      {
        value: '30d',
        label: t('admin.stats.range30d', '30 Days'),
        days: 30,
        period: 'daily' as const,
      },
      {
        value: '90d',
        label: t('admin.stats.range90d', '90 Days'),
        days: 90,
        period: 'weekly' as const,
      },
      {
        value: 'all',
        label: t('admin.stats.rangeAll', 'All'),
        days: 365,
        period: 'monthly' as const,
      },
    ],
    [t]
  )

  const currentRange = useMemo(
    () => timeRangeOptions.find((o) => o.value === timeRange) ?? timeRangeOptions[1],
    [timeRange, timeRangeOptions]
  )

  // Growth trend params derived from time range
  const growthParams: TenantGrowthParams = useMemo(
    () => ({
      period: currentRange.period,
      days: currentRange.days,
    }),
    [currentRange]
  )

  // ============================================================================
  // Data Fetching
  // ============================================================================

  const platformStatsQuery = useAdminPlatformStats()
  const growthTrendQuery = useAdminGrowthTrend(growthParams)

  // Fetch recent registered tenants (sorted by created_at desc)
  const recentTenantsQuery = useAdminTenants({
    page: 1,
    page_size: 10,
    sort_by: 'created_at' as never,
    sort_dir: 'desc' as never,
  })

  // Fetch recently suspended tenants
  const suspendedTenantsQuery = useAdminTenants({
    page: 1,
    page_size: 10,
    status: 'suspended' as never,
    sort_by: 'created_at' as never,
    sort_dir: 'desc' as never,
  })

  // Extract data
  const platformStats = useMemo(() => {
    const resp = platformStatsQuery.data as unknown as {
      data?: { data?: Record<string, unknown> }
    }
    return resp?.data?.data ?? null
  }, [platformStatsQuery.data])

  const growthData = useMemo(() => {
    const resp = growthTrendQuery.data as unknown as {
      data?: { data?: { data_points?: Array<Record<string, unknown>> } }
    }
    return resp?.data?.data?.data_points ?? []
  }, [growthTrendQuery.data])

  const recentTenants = useMemo(() => {
    const resp = recentTenantsQuery.data as unknown as {
      data?: { data?: { tenants?: HandlerTenantResponse[] } }
    }
    return resp?.data?.data?.tenants ?? []
  }, [recentTenantsQuery.data])

  const suspendedTenants = useMemo(() => {
    const resp = suspendedTenantsQuery.data as unknown as {
      data?: { data?: { tenants?: HandlerTenantResponse[] } }
    }
    return resp?.data?.data?.tenants ?? []
  }, [suspendedTenantsQuery.data])

  // ============================================================================
  // Stat Cards
  // ============================================================================

  const totalTenants = (platformStats?.total_tenants as number) ?? 0
  const activeTenants = (platformStats?.active_tenants as number) ?? 0
  const trialTenants = (platformStats?.trial_tenants as number) ?? 0
  const suspendedCount = (platformStats?.suspended_tenants as number) ?? 0

  const statCards = useMemo(
    () => [
      {
        title: t('admin.stats.totalTenants', 'Total Tenants'),
        value: totalTenants,
        icon: <IconUserGroup size="extra-large" />,
        color: 'var(--semi-color-primary)',
      },
      {
        title: t('admin.stats.activeTenants', 'Active Tenants'),
        value: activeTenants,
        icon: <IconTick size="extra-large" />,
        color: 'var(--semi-color-success)',
      },
      {
        title: t('admin.stats.trialTenants', 'Trial Tenants'),
        value: trialTenants,
        icon: <IconPlus size="extra-large" />,
        color: 'var(--semi-color-info)',
      },
      {
        title: t('admin.stats.suspendedTenants', 'Suspended Tenants'),
        value: suspendedCount,
        icon: <IconStop size="extra-large" />,
        color: 'var(--semi-color-danger)',
      },
    ],
    [t, totalTenants, activeTenants, trialTenants, suspendedCount]
  )

  // ============================================================================
  // Growth Trend Chart
  // ============================================================================

  const growthChartOption: EChartsOption = useMemo(() => {
    if (!growthData.length) return {}

    const dates = growthData.map((p) => {
      const d = p.date as string
      return d.slice(5) // MM-DD or MM
    })
    const totalSeries = growthData.map((p) => p.total_tenants as number)
    const newSeries = growthData.map((p) => p.new_tenants as number)

    return {
      tooltip: {
        trigger: 'axis',
        axisPointer: { type: 'cross' },
      },
      legend: {
        data: [t('admin.stats.totalTenantsLine', 'Total'), t('admin.stats.newTenantsLine', 'New')],
        bottom: 0,
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '15%',
        top: '10%',
        containLabel: true,
      },
      xAxis: {
        type: 'category',
        boundaryGap: false,
        data: dates,
        axisLabel: {
          rotate: dates.length > 15 ? 45 : 0,
          fontSize: 11,
        },
      },
      yAxis: [
        {
          type: 'value',
          name: t('admin.stats.totalTenantsLine', 'Total'),
          minInterval: 1,
        },
        {
          type: 'value',
          name: t('admin.stats.newTenantsLine', 'New'),
          minInterval: 1,
        },
      ],
      series: [
        {
          name: t('admin.stats.totalTenantsLine', 'Total'),
          type: 'line',
          smooth: true,
          symbol: 'circle',
          symbolSize: 6,
          yAxisIndex: 0,
          lineStyle: { width: 2 },
          itemStyle: { color: 'var(--semi-color-primary)' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(var(--semi-blue-5), 0.3)' },
              { offset: 1, color: 'rgba(var(--semi-blue-5), 0.05)' },
            ]),
          },
          data: totalSeries,
        },
        {
          name: t('admin.stats.newTenantsLine', 'New'),
          type: 'line',
          smooth: true,
          symbol: 'circle',
          symbolSize: 6,
          yAxisIndex: 1,
          lineStyle: { width: 2 },
          itemStyle: { color: 'var(--semi-color-success)' },
          data: newSeries,
        },
      ],
    }
  }, [growthData, t])

  // ============================================================================
  // Plan Distribution Pie Chart
  // ============================================================================

  const planDistributionOption: EChartsOption = useMemo(() => {
    const tenantsByPlan = (platformStats?.tenants_by_plan as Record<string, number>) ?? {}
    const data = Object.entries(tenantsByPlan).map(([name, value]) => ({
      name: name.charAt(0).toUpperCase() + name.slice(1),
      value,
      itemStyle: { color: PLAN_COLOR_MAP[name] ?? '#bdc3c7' },
    }))

    if (!data.length) return {}

    return {
      tooltip: {
        trigger: 'item',
        formatter: '{b}: {c} ({d}%)',
      },
      legend: {
        orient: 'vertical',
        right: '5%',
        top: 'center',
      },
      series: [
        {
          type: 'pie',
          radius: ['40%', '70%'],
          center: ['40%', '50%'],
          avoidLabelOverlap: false,
          label: {
            show: true,
            formatter: '{b}\n{d}%',
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
    }
  }, [platformStats])

  // ============================================================================
  // Status Distribution Pie Chart
  // ============================================================================

  const statusDistributionOption: EChartsOption = useMemo(() => {
    if (!platformStats) return {}

    const statusData = [
      {
        name: t('admin.stats.statusActive', 'Active'),
        value: activeTenants,
        itemStyle: { color: '#2ecc71' },
      },
      {
        name: t('admin.stats.statusTrial', 'Trial'),
        value: trialTenants,
        itemStyle: { color: '#3498db' },
      },
      {
        name: t('admin.stats.statusSuspended', 'Suspended'),
        value: suspendedCount,
        itemStyle: { color: '#e74c3c' },
      },
      {
        name: t('admin.stats.statusInactive', 'Inactive'),
        value: (platformStats.inactive_tenants as number) ?? 0,
        itemStyle: { color: '#95a5a6' },
      },
    ].filter((item) => item.value > 0)

    if (!statusData.length) return {}

    return {
      tooltip: {
        trigger: 'item',
        formatter: '{b}: {c} ({d}%)',
      },
      legend: {
        orient: 'vertical',
        right: '5%',
        top: 'center',
      },
      series: [
        {
          type: 'pie',
          radius: ['40%', '70%'],
          center: ['40%', '50%'],
          avoidLabelOverlap: false,
          label: {
            show: true,
            formatter: '{b}\n{d}%',
          },
          emphasis: {
            label: {
              show: true,
              fontSize: 14,
              fontWeight: 'bold',
            },
          },
          data: statusData,
        },
      ],
    }
  }, [platformStats, activeTenants, trialTenants, suspendedCount, t])

  // ============================================================================
  // Event Handlers
  // ============================================================================

  const handleTimeRangeChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      if (typeof value === 'string') {
        setTimeRange(value as TimeRange)
      }
    },
    []
  )

  const handleTenantClick = useCallback(
    (id: string) => {
      navigate(`/super-admin/tenants/${id}`)
    },
    [navigate]
  )

  // ============================================================================
  // Table Columns
  // ============================================================================

  const tenantColumns = useMemo(
    () => [
      {
        title: t('admin.stats.tenantName', 'Name'),
        dataIndex: 'name',
        key: 'name',
        render: (_: unknown, record: HandlerTenantResponse) => (
          <Text
            link
            onClick={() => handleTenantClick(record.id ?? '')}
            style={{ cursor: 'pointer' }}
          >
            {record.name}
          </Text>
        ),
      },
      {
        title: t('admin.stats.tenantCode', 'Code'),
        dataIndex: 'code',
        key: 'code',
      },
      {
        title: t('admin.stats.tenantPlan', 'Plan'),
        dataIndex: 'plan',
        key: 'plan',
        render: (plan: string) => (
          <Tag color={STATUS_COLOR_MAP[plan] ?? ('blue' as TagColor)} size="small">
            {plan}
          </Tag>
        ),
      },
      {
        title: t('admin.stats.tenantStatus', 'Status'),
        dataIndex: 'status',
        key: 'status',
        render: (status: string) => (
          <Tag color={STATUS_COLOR_MAP[status] ?? ('grey' as TagColor)} size="small">
            {status}
          </Tag>
        ),
      },
      {
        title: t('admin.stats.tenantCreatedAt', 'Created'),
        dataIndex: 'created_at',
        key: 'created_at',
        render: (val: string) => formatDate(val),
      },
    ],
    [t, handleTenantClick, formatDate]
  )

  // ============================================================================
  // Loading states
  // ============================================================================

  const isStatsLoading = platformStatsQuery.isLoading
  const isGrowthLoading = growthTrendQuery.isLoading

  // ============================================================================
  // Render
  // ============================================================================

  return (
    <Container>
      <Space vertical align="start" spacing="medium" style={{ width: '100%' }}>
        {/* Header with title and time range selector */}
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            width: '100%',
          }}
        >
          <Title heading={4} style={{ margin: 0 }}>
            {t('admin.stats.title', 'Platform Statistics')}
          </Title>
          <Select
            value={timeRange}
            onChange={handleTimeRangeChange}
            optionList={timeRangeOptions.map((o) => ({
              value: o.value,
              label: o.label,
            }))}
            prefix={<IconCalendar />}
            size="small"
            style={{ width: 140 }}
          />
        </div>

        {/* Key Indicator Cards */}
        <Row gutter={[16, 16]} style={{ width: '100%' }}>
          {statCards.map((stat, index) => (
            <Col key={index} xs={24} sm={12} lg={6}>
              <StatCard {...stat} loading={isStatsLoading} />
            </Col>
          ))}
        </Row>

        {/* Tenant Growth Trend Chart */}
        <Row gutter={[16, 16]} style={{ width: '100%' }}>
          <Col xs={24}>
            <Card
              title={t('admin.stats.tenantGrowth', 'Tenant Growth Trend')}
              headerExtraContent={
                <Text type="secondary" size="small">
                  {currentRange.label}
                </Text>
              }
            >
              <div style={{ height: 350 }}>
                {isGrowthLoading ? (
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      height: '100%',
                    }}
                  >
                    <Spin size="large" />
                  </div>
                ) : growthData.length === 0 ? (
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      height: '100%',
                    }}
                  >
                    <Empty description={t('admin.stats.noData', 'No data available')} />
                  </div>
                ) : (
                  <ReactEChartsCore
                    echarts={echarts}
                    option={growthChartOption}
                    style={{ height: '100%', width: '100%' }}
                    notMerge
                    lazyUpdate
                  />
                )}
              </div>
            </Card>
          </Col>
        </Row>

        {/* Pie Charts Row */}
        <Row gutter={[16, 16]} style={{ width: '100%' }}>
          <Col xs={24} lg={12}>
            <Card title={t('admin.stats.planDistribution', 'Plan Distribution')}>
              <div style={{ height: 300 }}>
                {isStatsLoading ? (
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      height: '100%',
                    }}
                  >
                    <Spin size="large" />
                  </div>
                ) : Object.keys((platformStats?.tenants_by_plan as Record<string, number>) ?? {})
                    .length === 0 ? (
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      height: '100%',
                    }}
                  >
                    <Empty description={t('admin.stats.noData', 'No data available')} />
                  </div>
                ) : (
                  <ReactEChartsCore
                    echarts={echarts}
                    option={planDistributionOption}
                    style={{ height: '100%', width: '100%' }}
                    notMerge
                    lazyUpdate
                  />
                )}
              </div>
            </Card>
          </Col>
          <Col xs={24} lg={12}>
            <Card title={t('admin.stats.statusDistribution', 'Status Distribution')}>
              <div style={{ height: 300 }}>
                {isStatsLoading ? (
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      height: '100%',
                    }}
                  >
                    <Spin size="large" />
                  </div>
                ) : !platformStats ? (
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      height: '100%',
                    }}
                  >
                    <Empty description={t('admin.stats.noData', 'No data available')} />
                  </div>
                ) : (
                  <ReactEChartsCore
                    echarts={echarts}
                    option={statusDistributionOption}
                    style={{ height: '100%', width: '100%' }}
                    notMerge
                    lazyUpdate
                  />
                )}
              </div>
            </Card>
          </Col>
        </Row>

        {/* Recent Tenants Tables */}
        <Row gutter={[16, 16]} style={{ width: '100%' }}>
          <Col xs={24} lg={12}>
            <Card title={t('admin.stats.recentRegistered', 'Recently Registered Tenants')}>
              <Table
                columns={tenantColumns}
                dataSource={recentTenants}
                loading={recentTenantsQuery.isLoading}
                pagination={false}
                size="small"
                rowKey="id"
                empty={<Empty description={t('admin.stats.noTenants', 'No tenants')} />}
              />
            </Card>
          </Col>
          <Col xs={24} lg={12}>
            <Card title={t('admin.stats.recentSuspended', 'Recently Suspended Tenants')}>
              <Table
                columns={tenantColumns}
                dataSource={suspendedTenants}
                loading={suspendedTenantsQuery.isLoading}
                pagination={false}
                size="small"
                rowKey="id"
                empty={
                  <Empty
                    description={t('admin.stats.noSuspendedTenants', 'No suspended tenants')}
                  />
                }
              />
            </Card>
          </Col>
        </Row>
      </Space>
    </Container>
  )
}
