import type { CSSProperties, ReactNode } from 'react'
import { IconTick, IconClose } from '@douyinfe/semi-icons'
import './StatusFlow.css'

/**
 * Status step state
 */
export type StatusFlowStepState = 'completed' | 'current' | 'pending' | 'cancelled' | 'rejected'

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
    default:
      return <span className="status-flow__step-number">{stepIndex + 1}</span>
  }
}

/**
 * StatusFlow - A horizontal status progression component
 *
 * Displays a sequence of status steps with visual indicators for
 * completed, current, pending, cancelled, or rejected states.
 *
 * Features:
 * - Horizontal step-by-step visualization
 * - Visual state indicators (completed, current, pending, cancelled, rejected)
 * - Optional timestamps display
 * - Responsive design (stacks on mobile)
 * - Accessible (proper ARIA attributes)
 *
 * @example
 * // Sales Order Status Flow
 * <StatusFlow
 *   steps={[
 *     { key: 'draft', label: 'Draft', state: 'completed' },
 *     { key: 'confirmed', label: 'Confirmed', state: 'completed' },
 *     { key: 'shipped', label: 'Shipped', state: 'current' },
 *     { key: 'completed', label: 'Completed', state: 'pending' },
 *   ]}
 * />
 *
 * @example
 * // With timestamps
 * <StatusFlow
 *   showTimestamp
 *   steps={[
 *     { key: 'draft', label: 'Draft', state: 'completed', timestamp: '2024-01-01 10:00' },
 *     { key: 'confirmed', label: 'Confirmed', state: 'current', timestamp: '2024-01-02 14:30' },
 *     { key: 'completed', label: 'Completed', state: 'pending' },
 *   ]}
 * />
 */
export function StatusFlow({
  steps,
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

  return (
    <div
      className={`status-flow ${sizeClass} ${className}`.trim()}
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

export default StatusFlow
