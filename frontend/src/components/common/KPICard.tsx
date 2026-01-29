import type { CSSProperties, ReactNode, KeyboardEvent } from 'react'
import { Spin, Typography } from '@douyinfe/semi-ui-19'
import { IconArrowUp, IconArrowDown } from '@douyinfe/semi-icons'
import './KPICard.css'

const { Text } = Typography

/**
 * Trend direction for KPI cards
 */
export type TrendDirection = 'up' | 'down' | 'neutral'

/**
 * KPICard color variants
 */
export type KPICardVariant = 'default' | 'primary' | 'success' | 'warning' | 'danger'

/**
 * Trend data for showing changes
 */
export interface KPICardTrend {
  /** Trend direction */
  direction: TrendDirection
  /** Percentage change (e.g., 12.5 for 12.5%) */
  value: number
  /** Optional label for trend context (e.g., "vs last month") */
  label?: string
}

/**
 * KPICard component props
 */
export interface KPICardProps {
  /** Label describing the KPI metric */
  label: string
  /** The main value to display */
  value: ReactNode
  /** Color variant for the card */
  variant?: KPICardVariant
  /** Trend information to display */
  trend?: KPICardTrend
  /** Click handler for filtering/drilling down */
  onClick?: () => void
  /** Whether the card is in loading state */
  loading?: boolean
  /** Optional icon to display */
  icon?: ReactNode
  /** Optional subtitle or additional context */
  subtitle?: string
  /** Optional className for custom styling */
  className?: string
  /** Optional inline styles */
  style?: CSSProperties
}

/**
 * KPICard - A card component for displaying key performance indicators
 *
 * Features:
 * - Multiple color variants (default, primary, success, warning, danger)
 * - Trend display with up/down indicators and percentage
 * - Click support for filtering/navigation
 * - Loading state with spinner
 * - Responsive design with mobile-first approach
 * - Accessible (keyboard navigation, ARIA attributes)
 *
 * @example
 * // Basic usage
 * <KPICard label="Total Revenue" value="¥125,000" />
 *
 * @example
 * // With trend and click handler
 * <KPICard
 *   label="Orders"
 *   value={128}
 *   variant="primary"
 *   trend={{ direction: 'up', value: 12.5, label: 'vs last month' }}
 *   onClick={() => handleFilter('orders')}
 * />
 *
 * @example
 * // Danger variant for overdue items
 * <KPICard
 *   label="Overdue"
 *   value={5}
 *   variant="danger"
 *   trend={{ direction: 'up', value: 25 }}
 *   onClick={handleShowOverdue}
 * />
 */
export function KPICard({
  label,
  value,
  variant = 'default',
  trend,
  onClick,
  loading = false,
  icon,
  subtitle,
  className = '',
  style,
}: KPICardProps) {
  const isClickable = Boolean(onClick)

  const handleClick = () => {
    if (onClick) {
      onClick()
    }
  }

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (onClick && (event.key === 'Enter' || event.key === ' ')) {
      event.preventDefault()
      onClick()
    }
  }

  const getTrendIcon = () => {
    if (!trend) return null
    switch (trend.direction) {
      case 'up':
        return <IconArrowUp className="kpi-card__trend-icon kpi-card__trend-icon--up" />
      case 'down':
        return <IconArrowDown className="kpi-card__trend-icon kpi-card__trend-icon--down" />
      default:
        return null
    }
  }

  const getTrendClassName = () => {
    if (!trend) return ''
    switch (trend.direction) {
      case 'up':
        return 'kpi-card__trend--up'
      case 'down':
        return 'kpi-card__trend--down'
      default:
        return 'kpi-card__trend--neutral'
    }
  }

  return (
    <div
      className={`kpi-card kpi-card--${variant} ${isClickable ? 'kpi-card--clickable' : ''} ${className}`}
      style={style}
      onClick={handleClick}
      onKeyDown={handleKeyDown}
      role={isClickable ? 'button' : undefined}
      tabIndex={isClickable ? 0 : undefined}
      aria-label={isClickable ? `${label}: ${value}. Click to filter.` : undefined}
    >
      <Spin spinning={loading} size="small">
        <div className="kpi-card__content">
          {icon && <div className="kpi-card__icon">{icon}</div>}
          <div className="kpi-card__main">
            <Text className="kpi-card__label" type="secondary">
              {label}
            </Text>
            <div className="kpi-card__value-row">
              <span className="kpi-card__value">{value}</span>
              {trend && (
                <span className={`kpi-card__trend ${getTrendClassName()}`}>
                  {getTrendIcon()}
                  <span className="kpi-card__trend-value">{trend.value.toFixed(1)}%</span>
                </span>
              )}
            </div>
            {subtitle && (
              <Text className="kpi-card__subtitle" type="tertiary" size="small">
                {subtitle}
              </Text>
            )}
            {trend?.label && (
              <Text className="kpi-card__trend-label" type="tertiary" size="small">
                {trend.label}
              </Text>
            )}
          </div>
        </div>
      </Spin>
    </div>
  )
}

export default KPICard
