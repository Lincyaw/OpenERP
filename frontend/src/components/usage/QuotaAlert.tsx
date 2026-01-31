/**
 * QuotaAlert Component
 *
 * A warning component that displays alerts when usage is approaching
 * or has exceeded quota limits. Supports different severity levels
 * and actionable upgrade prompts.
 */
import { useMemo, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Banner, Button, Typography } from '@douyinfe/semi-ui-19'
import { IconAlertTriangle, IconAlertCircle, IconArrowUp } from '@douyinfe/semi-icons'
import type { QuotaItem, UsageMetric } from '@/api/usage'

import './QuotaAlert.css'

const { Text } = Typography

export type AlertSeverity = 'warning' | 'critical' | 'exceeded'

export interface QuotaAlertProps {
  /** Quota item or usage metric to check */
  quota?: QuotaItem
  metric?: UsageMetric
  /** Warning threshold percentage (default: 70) */
  warningThreshold?: number
  /** Critical threshold percentage (default: 90) */
  criticalThreshold?: number
  /** Show upgrade button */
  showUpgradeButton?: boolean
  /** Upgrade button URL */
  upgradeUrl?: string
  /** Custom class name */
  className?: string
  /** Callback when alert is dismissed */
  onDismiss?: () => void
  /** Whether the alert can be dismissed */
  dismissible?: boolean
}

/**
 * Calculate percentage from quota or metric
 */
function calculatePercentage(quota?: QuotaItem, metric?: UsageMetric): number {
  if (metric) {
    return metric.percentage
  }
  if (quota) {
    if (quota.is_unlimited || quota.limit <= 0) return 0
    return (quota.used / quota.limit) * 100
  }
  return 0
}

/**
 * Get alert severity based on percentage
 */
function getSeverity(
  percentage: number,
  warningThreshold: number,
  criticalThreshold: number
): AlertSeverity | null {
  if (percentage >= 100) return 'exceeded'
  if (percentage >= criticalThreshold) return 'critical'
  if (percentage >= warningThreshold) return 'warning'
  return null
}

/**
 * Get display name from quota or metric
 */
function getDisplayName(quota?: QuotaItem, metric?: UsageMetric): string {
  if (metric) return metric.display_name
  if (quota) return quota.display_name
  return ''
}

/**
 * Get current/limit values
 */
function getUsageValues(
  quota?: QuotaItem,
  metric?: UsageMetric
): { current: number; limit: number } {
  if (metric) {
    return { current: metric.current, limit: metric.limit }
  }
  if (quota) {
    return { current: quota.used, limit: quota.limit }
  }
  return { current: 0, limit: 0 }
}

/**
 * QuotaAlert displays a warning banner when usage approaches quota limits.
 * It provides actionable upgrade prompts and supports different severity levels.
 */
export function QuotaAlert({
  quota,
  metric,
  warningThreshold = 70,
  criticalThreshold = 90,
  showUpgradeButton = true,
  upgradeUrl = '/upgrade',
  className = '',
  onDismiss,
  dismissible = true,
}: QuotaAlertProps) {
  const { t } = useTranslation('system')
  const navigate = useNavigate()

  const percentage = useMemo(() => calculatePercentage(quota, metric), [quota, metric])

  const severity = useMemo(
    () => getSeverity(percentage, warningThreshold, criticalThreshold),
    [percentage, warningThreshold, criticalThreshold]
  )

  const displayName = useMemo(() => getDisplayName(quota, metric), [quota, metric])
  const { current, limit } = useMemo(() => getUsageValues(quota, metric), [quota, metric])

  const handleUpgrade = useCallback(() => {
    navigate(upgradeUrl)
  }, [navigate, upgradeUrl])

  // Don't render if no alert needed or unlimited
  if (!severity) return null
  if (quota?.is_unlimited) return null
  if (metric && metric.limit <= 0) return null

  const bannerType = severity === 'warning' ? 'warning' : 'danger'
  const Icon = severity === 'warning' ? IconAlertTriangle : IconAlertCircle

  const getMessage = () => {
    switch (severity) {
      case 'exceeded':
        return t('usage.alert.exceeded', {
          resource: displayName,
          current: current.toLocaleString(),
          limit: limit.toLocaleString(),
        })
      case 'critical':
        return t('usage.alert.critical', {
          resource: displayName,
          percentage: percentage.toFixed(0),
          remaining: (limit - current).toLocaleString(),
        })
      case 'warning':
        return t('usage.alert.warning', {
          resource: displayName,
          percentage: percentage.toFixed(0),
        })
      default:
        return ''
    }
  }

  return (
    <Banner
      type={bannerType}
      icon={<Icon />}
      closeIcon={dismissible ? undefined : null}
      onClose={onDismiss}
      className={`quota-alert quota-alert--${severity} ${className}`}
      description={
        <div className="quota-alert__content">
          <Text className="quota-alert__message">{getMessage()}</Text>
          {showUpgradeButton && (
            <Button
              theme="solid"
              type={severity === 'warning' ? 'warning' : 'danger'}
              size="small"
              icon={<IconArrowUp />}
              onClick={handleUpgrade}
              className="quota-alert__upgrade-btn"
            >
              {t('usage.alert.upgradeNow')}
            </Button>
          )}
        </div>
      }
    />
  )
}

/**
 * QuotaAlertList Component
 *
 * Displays multiple quota alerts for all metrics that are approaching limits.
 */
export interface QuotaAlertListProps {
  /** List of quotas to check */
  quotas?: QuotaItem[]
  /** List of metrics to check */
  metrics?: UsageMetric[]
  /** Warning threshold percentage (default: 70) */
  warningThreshold?: number
  /** Critical threshold percentage (default: 90) */
  criticalThreshold?: number
  /** Show upgrade button */
  showUpgradeButton?: boolean
  /** Maximum number of alerts to show */
  maxAlerts?: number
  /** Custom class name */
  className?: string
}

/**
 * QuotaAlertList displays alerts for all quotas/metrics that are approaching limits.
 */
export function QuotaAlertList({
  quotas = [],
  metrics = [],
  warningThreshold = 70,
  criticalThreshold = 90,
  showUpgradeButton = true,
  maxAlerts = 3,
  className = '',
}: QuotaAlertListProps) {
  // Filter and sort alerts by severity
  const alertItems = useMemo(() => {
    const items: Array<{
      type: 'quota' | 'metric'
      item: QuotaItem | UsageMetric
      percentage: number
    }> = []

    // Add quotas
    quotas.forEach((quota) => {
      if (quota.is_unlimited || quota.limit <= 0) return
      const percentage = (quota.used / quota.limit) * 100
      if (percentage >= warningThreshold) {
        items.push({ type: 'quota', item: quota, percentage })
      }
    })

    // Add metrics
    metrics.forEach((metric) => {
      if (metric.limit <= 0) return
      if (metric.percentage >= warningThreshold) {
        items.push({ type: 'metric', item: metric, percentage: metric.percentage })
      }
    })

    // Sort by percentage (highest first)
    items.sort((a, b) => b.percentage - a.percentage)

    return items.slice(0, maxAlerts)
  }, [quotas, metrics, warningThreshold, maxAlerts])

  if (alertItems.length === 0) return null

  return (
    <div className={`quota-alert-list ${className}`}>
      {alertItems.map((alertItem, index) => (
        <QuotaAlert
          key={`${alertItem.type}-${index}`}
          quota={alertItem.type === 'quota' ? (alertItem.item as QuotaItem) : undefined}
          metric={alertItem.type === 'metric' ? (alertItem.item as UsageMetric) : undefined}
          warningThreshold={warningThreshold}
          criticalThreshold={criticalThreshold}
          showUpgradeButton={showUpgradeButton}
        />
      ))}
    </div>
  )
}

export default QuotaAlert
