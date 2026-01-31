/**
 * UsageChart Component
 *
 * A line chart component for displaying usage trends over time.
 * Supports daily, weekly, and monthly time periods.
 */
import { useState, useMemo, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { Card, Typography, Select, Spin, Empty } from '@douyinfe/semi-ui-19'
import { IconCalendar } from '@douyinfe/semi-icons'
import ReactEChartsCore from 'echarts-for-react/lib/core'
import * as echarts from 'echarts/core'
import { LineChart } from 'echarts/charts'
import {
  GridComponent,
  TooltipComponent,
  LegendComponent,
  DataZoomComponent,
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import type { EChartsOption } from 'echarts'

import { useGetUsageHistory, type GetUsageHistoryParams } from '@/api/usage'

import './UsageChart.css'

// Register ECharts components
echarts.use([
  LineChart,
  GridComponent,
  TooltipComponent,
  LegendComponent,
  DataZoomComponent,
  CanvasRenderer,
])

const { Title } = Typography

export type TimePeriod = 'daily' | 'weekly' | 'monthly'

export interface UsageChartProps {
  /** Chart title */
  title?: string
  /** Initial time period */
  defaultPeriod?: TimePeriod
  /** Chart height */
  height?: number | string
  /** Show period selector */
  showPeriodSelector?: boolean
  /** Custom class name */
  className?: string
}

/**
 * Get date range based on period
 */
function getDateRange(period: TimePeriod): { start_date: string; end_date: string } {
  const end = new Date()
  const start = new Date()

  switch (period) {
    case 'weekly':
      start.setDate(start.getDate() - 12 * 7) // 12 weeks
      break
    case 'monthly':
      start.setMonth(start.getMonth() - 12) // 12 months
      break
    default: // daily
      start.setDate(start.getDate() - 30) // 30 days
  }

  return {
    start_date: start.toISOString().split('T')[0],
    end_date: end.toISOString().split('T')[0],
  }
}

/**
 * Format date for display based on period
 */
function formatDate(dateStr: string, period: TimePeriod): string {
  const date = new Date(dateStr)
  switch (period) {
    case 'monthly':
      return date.toLocaleDateString('zh-CN', { year: 'numeric', month: 'short' })
    case 'weekly':
      return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
    default:
      return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
  }
}

/**
 * UsageChart displays historical usage trends as a line chart
 * with support for different time periods.
 */
export function UsageChart({
  title,
  defaultPeriod = 'daily',
  height = 300,
  showPeriodSelector = true,
  className = '',
}: UsageChartProps) {
  const { t } = useTranslation('system')
  const [period, setPeriod] = useState<TimePeriod>(defaultPeriod)

  const dateRange = useMemo(() => getDateRange(period), [period])

  const queryParams: GetUsageHistoryParams = useMemo(
    () => ({
      period,
      ...dateRange,
    }),
    [period, dateRange]
  )

  const { data: response, isLoading, isError } = useGetUsageHistory(queryParams)

  const historyData = response?.status === 200 ? response.data.data : null

  const periodOptions = useMemo(
    () => [
      { value: 'daily', label: t('usage.period.daily') },
      { value: 'weekly', label: t('usage.period.weekly') },
      { value: 'monthly', label: t('usage.period.monthly') },
    ],
    [t]
  )

  const handlePeriodChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      if (typeof value === 'string') {
        setPeriod(value as TimePeriod)
      }
    },
    []
  )

  const chartOption: EChartsOption = useMemo(() => {
    if (!historyData?.data_points?.length) {
      return {}
    }

    const dates = historyData.data_points.map((point) => formatDate(point.date, period))
    const usersData = historyData.data_points.map((point) => point.users)
    const productsData = historyData.data_points.map((point) => point.products)
    const warehousesData = historyData.data_points.map((point) => point.warehouses)

    return {
      tooltip: {
        trigger: 'axis',
        axisPointer: {
          type: 'cross',
          label: {
            backgroundColor: 'var(--semi-color-bg-3)',
          },
        },
      },
      legend: {
        data: [
          t('usage.metrics.users'),
          t('usage.metrics.products'),
          t('usage.metrics.warehouses'),
        ],
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
      yAxis: {
        type: 'value',
        minInterval: 1,
      },
      dataZoom:
        dates.length > 20
          ? [
              {
                type: 'inside',
                start: 0,
                end: 100,
              },
              {
                start: 0,
                end: 100,
              },
            ]
          : undefined,
      series: [
        {
          name: t('usage.metrics.users'),
          type: 'line',
          smooth: true,
          symbol: 'circle',
          symbolSize: 6,
          lineStyle: {
            width: 2,
          },
          itemStyle: {
            color: 'var(--semi-color-primary)',
          },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(var(--semi-blue-5), 0.3)' },
              { offset: 1, color: 'rgba(var(--semi-blue-5), 0.05)' },
            ]),
          },
          data: usersData,
        },
        {
          name: t('usage.metrics.products'),
          type: 'line',
          smooth: true,
          symbol: 'circle',
          symbolSize: 6,
          lineStyle: {
            width: 2,
          },
          itemStyle: {
            color: 'var(--semi-color-success)',
          },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(var(--semi-green-5), 0.3)' },
              { offset: 1, color: 'rgba(var(--semi-green-5), 0.05)' },
            ]),
          },
          data: productsData,
        },
        {
          name: t('usage.metrics.warehouses'),
          type: 'line',
          smooth: true,
          symbol: 'circle',
          symbolSize: 6,
          lineStyle: {
            width: 2,
          },
          itemStyle: {
            color: 'var(--semi-color-warning)',
          },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(var(--semi-orange-5), 0.3)' },
              { offset: 1, color: 'rgba(var(--semi-orange-5), 0.05)' },
            ]),
          },
          data: warehousesData,
        },
      ],
    }
  }, [historyData, period, t])

  const chartTitle = title || t('usage.chart.title')

  return (
    <Card className={`usage-chart ${className}`}>
      <div className="usage-chart__header">
        <Title heading={5} className="usage-chart__title">
          {chartTitle}
        </Title>
        {showPeriodSelector && (
          <Select
            value={period}
            onChange={handlePeriodChange}
            optionList={periodOptions}
            prefix={<IconCalendar />}
            size="small"
            className="usage-chart__period-select"
          />
        )}
      </div>

      <div className="usage-chart__content" style={{ height }}>
        {isLoading && (
          <div className="usage-chart__loading">
            <Spin size="large" />
          </div>
        )}

        {isError && (
          <div className="usage-chart__error">
            <Empty description={t('usage.chart.loadError')} />
          </div>
        )}

        {!isLoading && !isError && historyData?.data_points?.length === 0 && (
          <div className="usage-chart__empty">
            <Empty description={t('usage.chart.noData')} />
          </div>
        )}

        {!isLoading && !isError && (historyData?.data_points?.length ?? 0) > 0 && (
          <ReactEChartsCore
            echarts={echarts}
            option={chartOption}
            style={{ height: '100%', width: '100%' }}
            notMerge
            lazyUpdate
          />
        )}
      </div>
    </Card>
  )
}

export default UsageChart
