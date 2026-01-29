import type { CSSProperties, ReactNode } from 'react'
import { IconTick, IconClose, IconAlertTriangle } from '@douyinfe/semi-icons'
import './StatusFlow.css'

/**
 * Status step state
 */
export type StatusFlowStepState =
  | 'completed'
  | 'current'
  | 'pending'
  | 'cancelled'
  | 'rejected'
  | 'error'

/**
 * Layout direction for the status flow
 */
export type StatusFlowDirection = 'horizontal' | 'vertical'

/**
 * Individual step in the status flow
 */
export interface StatusFlowStep {
  /** Unique key for the step */
  key: string
  /** Display label for the step */
  label: string
  /** State of this step */
  state: StatusFlowStepState
  /** Optional timestamp when this step was completed */
  timestamp?: string
  /** Optional description or note */
  description?: string
  /** Custom icon (overrides default) */
  icon?: ReactNode
}

/**
 * StatusFlow component props
 */
export interface StatusFlowProps {
  /** Array of status steps to display */
  steps: StatusFlowStep[]
  /** Layout direction - horizontal (default) or vertical */
  direction?: StatusFlowDirection
  /** Optional className for custom styling */
  className?: string
  /** Optional inline styles */
  style?: CSSProperties
  /** Size variant */
  size?: 'small' | 'default'
  /** Whether to show timestamps */
  showTimestamp?: boolean
  /** ARIA label for the component */
  ariaLabel?: string
}

/**
 * Get step state class modifier
 */
const getStepStateClass = (state: StatusFlowStepState): string => {
  const stateClasses: Record<StatusFlowStepState, string> = {
    completed: 'status-flow__step--completed',
    current: 'status-flow__step--current',
    pending: 'status-flow__step--pending',
    cancelled: 'status-flow__step--cancelled',
    rejected: 'status-flow__step--rejected',
    error: 'status-flow__step--error',
  }
  return stateClasses[state] || stateClasses.pending
}

/**
 * Get default icon for step state
 */
const getDefaultIcon = (state: StatusFlowStepState, stepIndex: number): ReactNode => {
  switch (state) {
    case 'completed':
      return <IconTick className="status-flow__step-icon-svg" />
    case 'cancelled':
    case 'rejected':
      return <IconClose className="status-flow__step-icon-svg" />
    case 'error':
      return <IconAlertTriangle className="status-flow__step-icon-svg" />
    default:
      return <span className="status-flow__step-number">{stepIndex + 1}</span>
  }
}

/**
 * StatusFlow - A status progression component
 *
 * Displays a sequence of status steps with visual indicators for
 * completed, current, pending, cancelled, rejected, or error states.
 *
 * Features:
 * - Horizontal and vertical step-by-step visualization
 * - Visual state indicators (completed, current, pending, cancelled, rejected, error)
 * - Optional timestamps display
 * - Responsive design (auto-stacks on mobile if horizontal)
 * - Accessible (proper ARIA attributes)
 *
 * @example
 * // Sales Order Status Flow (horizontal)
 * <StatusFlow
 *   direction="horizontal"
 *   steps={[
 *     { key: 'draft', label: 'Draft', state: 'completed' },
 *     { key: 'confirmed', label: 'Confirmed', state: 'completed' },
 *     { key: 'shipped', label: 'Shipped', state: 'current' },
 *     { key: 'completed', label: 'Completed', state: 'pending' },
 *   ]}
 * />
 *
 * @example
 * // Purchase Order Status Flow (vertical)
 * <StatusFlow
 *   direction="vertical"
 *   showTimestamp
 *   steps={[
 *     { key: 'draft', label: 'Draft', state: 'completed', timestamp: '2024-01-01 10:00' },
 *     { key: 'confirmed', label: 'Confirmed', state: 'current', timestamp: '2024-01-02 14:30' },
 *     { key: 'received', label: 'Received', state: 'pending' },
 *     { key: 'completed', label: 'Completed', state: 'pending' },
 *   ]}
 * />
 *
 * @example
 * // Error state
 * <StatusFlow
 *   steps={[
 *     { key: 'draft', label: 'Draft', state: 'completed' },
 *     { key: 'processing', label: 'Processing', state: 'error', description: 'Payment failed' },
 *     { key: 'completed', label: 'Completed', state: 'pending' },
 *   ]}
 * />
 */
export function StatusFlow({
  steps,
  direction = 'horizontal',
  className = '',
  style,
  size = 'default',
  showTimestamp = false,
  ariaLabel = 'Status progression',
}: StatusFlowProps) {
  if (!steps || steps.length === 0) {
    return null
  }

  const sizeClass = size === 'small' ? 'status-flow--small' : ''
  const directionClass = direction === 'vertical' ? 'status-flow--vertical' : ''

  return (
    <div
      className={`status-flow ${sizeClass} ${directionClass} ${className}`.trim()}
      style={style}
      role="list"
      aria-label={ariaLabel}
    >
      {steps.map((step, index) => {
        const isLast = index === steps.length - 1
        const stateClass = getStepStateClass(step.state)
        const icon = step.icon ?? getDefaultIcon(step.state, index)

        return (
          <div
            key={step.key}
            className={`status-flow__step ${stateClass}`}
            role="listitem"
            aria-current={step.state === 'current' ? 'step' : undefined}
          >
            {/* Step content */}
            <div className="status-flow__step-content">
              {/* Icon */}
              <div className="status-flow__step-icon">{icon}</div>

              {/* Label and timestamp */}
              <div className="status-flow__step-info">
                <span className="status-flow__step-label">{step.label}</span>
                {showTimestamp && step.timestamp && (
                  <span className="status-flow__step-timestamp">{step.timestamp}</span>
                )}
                {step.description && (
                  <span className="status-flow__step-description">{step.description}</span>
                )}
              </div>
            </div>

            {/* Connector line (not shown for last item) */}
            {!isLast && <div className="status-flow__connector" aria-hidden="true" />}
          </div>
        )
      })}
    </div>
  )
}

// ============================================================================
// Preset Flow Configurations
// ============================================================================

/**
 * Order flow types
 */
export type OrderFlowType = 'sales_order' | 'purchase_order' | 'return_order'

/**
 * Preset step configurations for common order flows
 */
export interface OrderFlowPreset {
  /** Display name for the flow */
  name: string
  /** Step definitions (key, label pairs) */
  steps: Array<{ key: string; label: string }>
}

/**
 * Preset configurations for order status flows
 *
 * Use these presets with the generateOrderFlowSteps helper function
 * to quickly create status flow steps based on the current status.
 */
export const ORDER_FLOW_PRESETS: Record<OrderFlowType, OrderFlowPreset> = {
  /**
   * Sales Order Flow: 草稿 → 已确认 → 已发货 → 已完成
   */
  sales_order: {
    name: '销售订单流程',
    steps: [
      { key: 'draft', label: '草稿' },
      { key: 'confirmed', label: '已确认' },
      { key: 'shipped', label: '已发货' },
      { key: 'completed', label: '已完成' },
    ],
  },

  /**
   * Purchase Order Flow: 草稿 → 已确认 → 已收货 → 已完成
   */
  purchase_order: {
    name: '采购订单流程',
    steps: [
      { key: 'draft', label: '草稿' },
      { key: 'confirmed', label: '已确认' },
      { key: 'received', label: '已收货' },
      { key: 'completed', label: '已完成' },
    ],
  },

  /**
   * Return Order Flow: 待审核 → 已批准 → 已入库 → 已完成
   */
  return_order: {
    name: '退货单流程',
    steps: [
      { key: 'pending_review', label: '待审核' },
      { key: 'approved', label: '已批准' },
      { key: 'stored', label: '已入库' },
      { key: 'completed', label: '已完成' },
    ],
  },
}

/**
 * Generate StatusFlowStep array from a preset and current status key
 *
 * @param flowType - The type of order flow (sales_order, purchase_order, return_order)
 * @param currentStatusKey - The key of the current status step
 * @param timestamps - Optional map of step keys to timestamps
 * @returns Array of StatusFlowStep objects
 *
 * @example
 * // Sales order at 'shipped' status
 * const steps = generateOrderFlowSteps('sales_order', 'shipped')
 * // Result: draft=completed, confirmed=completed, shipped=current, completed=pending
 *
 * @example
 * // With timestamps
 * const steps = generateOrderFlowSteps('purchase_order', 'confirmed', {
 *   draft: '2024-01-01 10:00',
 *   confirmed: '2024-01-02 14:30',
 * })
 */
export function generateOrderFlowSteps(
  flowType: OrderFlowType,
  currentStatusKey: string,
  timestamps?: Record<string, string>
): StatusFlowStep[] {
  const preset = ORDER_FLOW_PRESETS[flowType]
  if (!preset) {
    return []
  }

  const currentIndex = preset.steps.findIndex((step) => step.key === currentStatusKey)

  return preset.steps.map((step, index) => {
    let state: StatusFlowStepState

    if (currentIndex === -1) {
      // If current status not found, mark all as pending
      state = 'pending'
    } else if (index < currentIndex) {
      state = 'completed'
    } else if (index === currentIndex) {
      state = 'current'
    } else {
      state = 'pending'
    }

    return {
      key: step.key,
      label: step.label,
      state,
      timestamp: timestamps?.[step.key],
    }
  })
}

export default StatusFlow
