/**
 * UsageGauge Component
 *
 * A progress bar component for displaying usage metrics with visual indicators
 * for warning and critical thresholds.
 */
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Progress, Typography, Tooltip } from '@douyinfe/semi-ui-19'
import type { UsageMetric } from '@/api/usage'

import './UsageGauge.css'

const { Text } = Typography

export interface UsageGaugeProps {
  /** Usage metric data */
  metric: UsageMetric
  /** Show percentage text */
  showPercentage?: boolean
  /** Show usage text (current/limit) */
  showUsageText?: boolean
  /** Size variant */
  size?: 'small' | 'default' | 'large'
  /** Warning threshold percentage (default: 70) */
  warningThreshold?: number
  /** Critical threshold percentage (default: 90) */
  criticalThreshold?: number
  /** Custom class name */
  className?: string
}

/**
 * Get status color based on percentage and thresholds
 */
function getStatusColor(
  percentage: number,
  warningThreshold: number,
  criticalThreshold: number
): 'success' | 'warning' | 'danger' {
  if (percentage >= criticalThreshold) return 'danger'
  if (percentage >= warningThreshold) return 'warning'
  return 'success'
}

/**
 * Format limit display
 */
function formatLimit(limit: number, t: (key: string) => string): string {
  if (limit <= 0) return t('usage.unlimited')
  if (limit >= 1000000) return `${(limit / 1000000).toFixed(1)}M`
  if (limit >= 1000) return `${(limit / 1000).toFixed(1)}K`
  return limit.toLocaleString()
}

/**
 * UsageGauge displays a single usage metric as a progress bar
 * with color-coded status indicators.
 */
export function UsageGauge({
  metric,
  showPercentage = true,
  showUsageText = true,
  size = 'default',
  warningThreshold = 70,
  criticalThreshold = 90,
  className = '',
}: UsageGaugeProps) {
  const { t } = useTranslation('system')

  const isUnlimited = metric.limit <= 0
  const percentage = isUnlimited ? 0 : Math.min(metric.percentage, 100)
  const status = getStatusColor(percentage, warningThreshold, criticalThreshold)

  const strokeColor = useMemo(() => {
    switch (status) {
      case 'danger':
        return 'var(--semi-color-danger)'
      case 'warning':
        return 'var(--semi-color-warning)'
      default:
        return 'var(--semi-color-success)'
    }
  }, [status])

  const progressSize = useMemo(() => {
    switch (size) {
      case 'small':
        return 'small'
      case 'large':
        return 'large'
      default:
        return 'default'
    }
  }, [size])

  const tooltipContent = useMemo(() => {
    if (isUnlimited) {
      return t('usage.unlimitedTooltip', { current: metric.current.toLocaleString() })
    }
    const remaining = metric.limit - metric.current
    return t('usage.tooltip', {
      current: metric.current.toLocaleString(),
      limit: metric.limit.toLocaleString(),
      remaining: remaining.toLocaleString(),
      percentage: percentage.toFixed(1),
    })
  }, [isUnlimited, metric.current, metric.limit, percentage, t])

  return (
    <div className={`usage-gauge usage-gauge--${size} ${className}`}>
      <div className="usage-gauge__header">
        <Text strong className="usage-gauge__label">
          {metric.display_name}
        </Text>
        {showUsageText && (
          <Text type="tertiary" size="small" className="usage-gauge__usage-text">
            {metric.current.toLocaleString()} / {formatLimit(metric.limit, t)}
          </Text>
        )}
      </div>

      <Tooltip content={tooltipContent}>
        <div className="usage-gauge__progress-wrapper">
          <Progress
            percent={isUnlimited ? 0 : percentage}
            stroke={strokeColor}
            size={progressSize}
            showInfo={showPercentage && !isUnlimited}
            aria-label={`${metric.display_name}: ${percentage.toFixed(0)}%`}
          />
          {isUnlimited && (
            <Text type="tertiary" size="small" className="usage-gauge__unlimited-badge">
              {t('usage.unlimited')}
            </Text>
          )}
        </div>
      </Tooltip>

      {status === 'danger' && !isUnlimited && (
        <Text type="danger" size="small" className="usage-gauge__warning">
          {t('usage.criticalWarning')}
        </Text>
      )}
      {status === 'warning' && !isUnlimited && (
        <Text type="warning" size="small" className="usage-gauge__warning">
          {t('usage.warningMessage')}
        </Text>
      )}
    </div>
  )
}

export default UsageGauge
