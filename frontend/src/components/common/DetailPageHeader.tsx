import type { CSSProperties, ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Button, Tag, Dropdown, Space, Typography, Spin } from '@douyinfe/semi-ui-19'
import { IconArrowLeft, IconChevronDown } from '@douyinfe/semi-icons'
import './DetailPageHeader.css'

const { Title, Text } = Typography

/**
 * Status variant for the status badge
 */
export type DetailPageHeaderStatusVariant =
  | 'default'
  | 'primary'
  | 'success'
  | 'warning'
  | 'danger'
  | 'info'

/**
 * Metric item displayed in the header
 */
export interface DetailPageHeaderMetric {
  /** Label for the metric */
  label: string
  /** Value to display */
  value: ReactNode
  /** Optional variant for styling */
  variant?: 'default' | 'primary' | 'success' | 'warning' | 'danger'
}

/**
 * Action button configuration
 */
export interface DetailPageHeaderAction {
  /** Unique key for the action */
  key: string
  /** Display label */
  label: string
  /** Icon to display */
  icon?: ReactNode
  /** Button type/variant */
  type?: 'primary' | 'secondary' | 'tertiary' | 'warning' | 'danger'
  /** Click handler */
  onClick: () => void
  /** Whether the action is disabled */
  disabled?: boolean
  /** Loading state */
  loading?: boolean
}

/**
 * Status badge configuration
 */
export interface DetailPageHeaderStatus {
  /** Display text */
  label: string
  /** Color variant */
  variant: DetailPageHeaderStatusVariant
}

/**
 * DetailPageHeader component props
 */
export interface DetailPageHeaderProps {
  /** Page title (e.g., "Sales Order Detail") */
  title: string
  /** Document number to display prominently */
  documentNumber?: string
  /** Status badge configuration */
  status?: DetailPageHeaderStatus
  /** Key metrics to display */
  metrics?: DetailPageHeaderMetric[]
  /** Primary action button */
  primaryAction?: DetailPageHeaderAction
  /** Secondary actions (shown as dropdown on mobile) */
  secondaryActions?: DetailPageHeaderAction[]
  /** Back button click handler */
  onBack?: () => void
  /** Back button label (default: "Back") */
  backLabel?: string
  /** Whether to show back button */
  showBack?: boolean
  /** Loading state */
  loading?: boolean
  /** Optional className for custom styling */
  className?: string
  /** Optional inline styles */
  style?: CSSProperties
  /** Optional custom content in the title area */
  titleSuffix?: ReactNode
}

// Status tag color mapping
const STATUS_TAG_COLORS: Record<
  DetailPageHeaderStatusVariant,
  'blue' | 'cyan' | 'green' | 'grey' | 'red' | 'orange'
> = {
  default: 'grey',
  primary: 'blue',
  success: 'green',
  warning: 'orange',
  danger: 'red',
  info: 'cyan',
}

// Metric variant to className mapping
const METRIC_VARIANT_CLASSES: Record<string, string> = {
  default: '',
  primary: 'detail-page-header__metric-value--primary',
  success: 'detail-page-header__metric-value--success',
  warning: 'detail-page-header__metric-value--warning',
  danger: 'detail-page-header__metric-value--danger',
}

/**
 * DetailPageHeader - A unified header component for detail pages
 *
 * Features:
 * - Document number prominently displayed
 * - Status badge with color variants
 * - Key metrics row with optional color coding
 * - Primary action button + secondary actions dropdown
 * - Responsive design (horizontal on desktop, stacked on mobile)
 * - Back button always visible
 * - Mobile-friendly action collapsing
 * - Accessible (keyboard navigation, ARIA attributes)
 *
 * @example
 * // Basic usage
 * <DetailPageHeader
 *   title="Sales Order Detail"
 *   documentNumber="SO-2024-0001"
 *   status={{ label: 'Confirmed', variant: 'success' }}
 *   onBack={() => navigate('/trade/sales')}
 * />
 *
 * @example
 * // With metrics and actions
 * <DetailPageHeader
 *   title="Sales Order Detail"
 *   documentNumber="SO-2024-0001"
 *   status={{ label: 'Draft', variant: 'default' }}
 *   metrics={[
 *     { label: 'Total Amount', value: '¥12,500.00', variant: 'primary' },
 *     { label: 'Items', value: '5' },
 *   ]}
 *   primaryAction={{
 *     key: 'confirm',
 *     label: 'Confirm Order',
 *     icon: <IconTick />,
 *     type: 'primary',
 *     onClick: handleConfirm,
 *   }}
 *   secondaryActions={[
 *     { key: 'edit', label: 'Edit', icon: <IconEdit />, onClick: handleEdit },
 *     { key: 'cancel', label: 'Cancel', type: 'danger', onClick: handleCancel },
 *   ]}
 *   onBack={() => navigate(-1)}
 * />
 */
export function DetailPageHeader({
  title,
  documentNumber,
  status,
  metrics,
  primaryAction,
  secondaryActions,
  onBack,
  backLabel,
  showBack = true,
  loading = false,
  className = '',
  style,
  titleSuffix,
}: DetailPageHeaderProps) {
  const { t } = useTranslation('common')

  // Use provided backLabel or fall back to i18n
  const effectiveBackLabel = backLabel ?? t('actions.back')
  /**
   * Get button type prop from action type
   */
  const getButtonType = (
    actionType?: string
  ): 'primary' | 'secondary' | 'tertiary' | 'warning' | 'danger' | undefined => {
    if (actionType === 'danger') return 'danger'
    if (actionType === 'warning') return 'warning'
    if (actionType === 'primary') return 'primary'
    if (actionType === 'secondary') return 'secondary'
    if (actionType === 'tertiary') return 'tertiary'
    return undefined
  }

  /**
   * Render secondary actions as buttons (desktop view)
   */
  const renderSecondaryActionsButtons = () => {
    if (!secondaryActions || secondaryActions.length === 0) return null

    return secondaryActions.map((action) => (
      <Button
        key={action.key}
        icon={action.icon}
        type={getButtonType(action.type)}
        onClick={action.onClick}
        disabled={action.disabled}
        loading={action.loading}
        className="detail-page-header__secondary-action"
      >
        {action.label}
      </Button>
    ))
  }

  /**
   * Render mobile actions dropdown (combines all actions)
   */
  const renderMobileActions = () => {
    const allActions = [...(primaryAction ? [primaryAction] : []), ...(secondaryActions || [])]

    if (allActions.length === 0) return null

    if (allActions.length === 1 && primaryAction) {
      // If only primary action, show as button
      return (
        <Button
          type={getButtonType(primaryAction.type)}
          icon={primaryAction.icon}
          onClick={primaryAction.onClick}
          disabled={primaryAction.disabled}
          loading={primaryAction.loading}
          className="detail-page-header__mobile-primary"
        >
          {primaryAction.label}
        </Button>
      )
    }

    // Multiple actions: show dropdown
    const dropdownMenu = (
      <Dropdown.Menu>
        {allActions.map((action) => (
          <Dropdown.Item
            key={action.key}
            icon={action.icon}
            onClick={action.onClick}
            disabled={action.disabled}
            type={action.type === 'danger' ? 'danger' : undefined}
          >
            {action.label}
          </Dropdown.Item>
        ))}
      </Dropdown.Menu>
    )

    return (
      <Dropdown trigger="click" position="bottomRight" render={dropdownMenu}>
        <span style={{ display: 'inline-flex' }}>
          <Button icon={<IconChevronDown />} aria-label={t('actions.moreActions')}>
            {t('actions.moreActions')}
          </Button>
        </span>
      </Dropdown>
    )
  }

  /**
   * Render metrics row
   */
  const renderMetrics = () => {
    if (!metrics || metrics.length === 0) return null

    return (
      <div className="detail-page-header__metrics" role="list" aria-label="Key metrics">
        {metrics.map((metric, index) => (
          <div
            key={`${metric.label}-${index}`}
            className="detail-page-header__metric"
            role="listitem"
          >
            <Text className="detail-page-header__metric-label" type="secondary" size="small">
              {metric.label}
            </Text>
            <span
              className={`detail-page-header__metric-value ${METRIC_VARIANT_CLASSES[metric.variant || 'default']}`}
            >
              {metric.value}
            </span>
          </div>
        ))}
      </div>
    )
  }

  if (loading) {
    return (
      <header
        className={`detail-page-header detail-page-header--loading ${className}`}
        style={style}
        aria-busy="true"
        aria-live="polite"
      >
        <Spin size="large" aria-label="Loading content" />
      </header>
    )
  }

  return (
    <header className={`detail-page-header ${className}`} style={style}>
      {/* Top Row: Back button, Title, Status, Actions */}
      <div className="detail-page-header__top-row">
        <div className="detail-page-header__left">
          {/* Back Button */}
          {showBack && onBack && (
            <Button
              icon={<IconArrowLeft />}
              theme="borderless"
              onClick={onBack}
              className="detail-page-header__back-btn"
              aria-label={effectiveBackLabel}
            >
              <span className="detail-page-header__back-label">{effectiveBackLabel}</span>
            </Button>
          )}

          {/* Title and Document Number */}
          <div className="detail-page-header__title-group">
            <div className="detail-page-header__title-row">
              <Title heading={4} className="detail-page-header__title">
                {title}
              </Title>
              {status && (
                <Tag
                  color={STATUS_TAG_COLORS[status.variant]}
                  size="large"
                  className="detail-page-header__status"
                >
                  {status.label}
                </Tag>
              )}
              {titleSuffix}
            </div>
            {documentNumber && (
              <Text className="detail-page-header__document-number" strong>
                {documentNumber}
              </Text>
            )}
          </div>
        </div>

        {/* Desktop Actions */}
        <div className="detail-page-header__actions detail-page-header__actions--desktop">
          <Space>
            {renderSecondaryActionsButtons()}
            {primaryAction && (
              <Button
                type={getButtonType(primaryAction.type)}
                icon={primaryAction.icon}
                onClick={primaryAction.onClick}
                disabled={primaryAction.disabled}
                loading={primaryAction.loading}
                className="detail-page-header__primary-action"
              >
                {primaryAction.label}
              </Button>
            )}
          </Space>
        </div>

        {/* Mobile Actions */}
        <div className="detail-page-header__actions detail-page-header__actions--mobile">
          {renderMobileActions()}
        </div>
      </div>

      {/* Metrics Row */}
      {renderMetrics()}
    </header>
  )
}

export default DetailPageHeader
