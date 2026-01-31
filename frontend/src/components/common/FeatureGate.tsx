/**
 * FeatureGate Component
 *
 * Declarative component for conditional rendering based on SaaS plan features.
 * Provides a clean API for showing/hiding UI elements and displaying upgrade prompts.
 *
 * This is different from the Feature component (for feature flags/A/B testing).
 * FeatureGate handles subscription-based feature gating with upgrade prompts.
 *
 * @module components/common/FeatureGate
 *
 * @example Basic usage
 * ```tsx
 * <FeatureGate feature="advanced_reporting">
 *   <AdvancedReports />
 * </FeatureGate>
 * ```
 *
 * @example With custom fallback
 * ```tsx
 * <FeatureGate
 *   feature="api_access"
 *   fallback={<BasicAPIInfo />}
 * >
 *   <APIConfiguration />
 * </FeatureGate>
 * ```
 *
 * @example With upgrade prompt
 * ```tsx
 * <FeatureGate
 *   feature="multi_warehouse"
 *   showUpgradePrompt
 * >
 *   <WarehouseManager />
 * </FeatureGate>
 * ```
 *
 * @example Inline mode (for buttons/links)
 * ```tsx
 * <FeatureGate feature="bulk_import" inline>
 *   <Button>Bulk Import</Button>
 * </FeatureGate>
 * ```
 */

import type { ReactNode, CSSProperties } from 'react'
import { useShallow } from 'zustand/react/shallow'
import { useFeatureStore, getPlanDisplayName, type FeatureKey } from '@/store'
import { Button, Typography, Card, Space } from '@douyinfe/semi-ui-19'
import { IconLock, IconArrowUp } from '@douyinfe/semi-icons'

// ============================================================================
// Types
// ============================================================================

/**
 * Props for the FeatureGate component
 */
export interface FeatureGateProps {
  /**
   * The feature key to check
   */
  feature: FeatureKey

  /**
   * Content to render when the feature is enabled
   */
  children: ReactNode

  /**
   * Content to render when the feature is disabled.
   * If not provided and showUpgradePrompt is false, renders null.
   * If showUpgradePrompt is true, shows the default upgrade prompt.
   */
  fallback?: ReactNode

  /**
   * Content to render while features are loading.
   * If not provided, renders null while loading.
   */
  loading?: ReactNode

  /**
   * Whether to show the default upgrade prompt when feature is disabled.
   * Default: false
   */
  showUpgradePrompt?: boolean

  /**
   * Whether to render inline (for buttons/links) vs block (for sections).
   * Inline mode shows a smaller, more subtle upgrade indicator.
   * Default: false
   */
  inline?: boolean

  /**
   * Custom upgrade URL. If not provided, uses default settings page.
   */
  upgradeUrl?: string

  /**
   * Callback when upgrade button is clicked
   */
  onUpgradeClick?: () => void

  /**
   * Custom styles for the wrapper
   */
  style?: CSSProperties

  /**
   * Custom class name for the wrapper
   */
  className?: string
}

/**
 * Props for the UpgradePrompt component
 */
export interface UpgradePromptProps {
  /**
   * The feature that requires upgrade
   */
  feature: FeatureKey

  /**
   * Feature description
   */
  description: string

  /**
   * Required plan to unlock the feature
   */
  requiredPlan: string

  /**
   * Whether to render inline (smaller) or block (larger)
   */
  inline?: boolean

  /**
   * Custom upgrade URL
   */
  upgradeUrl?: string

  /**
   * Callback when upgrade button is clicked
   */
  onUpgradeClick?: () => void
}

// ============================================================================
// Sub-components
// ============================================================================

const { Text, Title } = Typography

/**
 * Default upgrade prompt component
 */
export function UpgradePrompt({
  feature: _feature,
  description,
  requiredPlan,
  inline = false,
  upgradeUrl = '/settings/subscription',
  onUpgradeClick,
}: UpgradePromptProps) {
  const handleClick = () => {
    if (onUpgradeClick) {
      onUpgradeClick()
    } else {
      // Navigate to upgrade page
      window.location.href = upgradeUrl
    }
  }

  if (inline) {
    return (
      <Button
        icon={<IconLock />}
        theme="light"
        type="tertiary"
        size="small"
        onClick={handleClick}
        style={{ opacity: 0.7 }}
      >
        Upgrade to {requiredPlan}
      </Button>
    )
  }

  return (
    <Card
      style={{
        textAlign: 'center',
        padding: 'var(--spacing-6)',
        backgroundColor: 'var(--semi-color-fill-0)',
        border: '1px dashed var(--semi-color-border)',
      }}
      bodyStyle={{ padding: 'var(--spacing-4)' }}
    >
      <Space vertical align="center" spacing="tight">
        <div
          style={{
            width: 48,
            height: 48,
            borderRadius: '50%',
            backgroundColor: 'var(--semi-color-primary-light-default)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            marginBottom: 'var(--spacing-2)',
          }}
        >
          <IconLock size="large" style={{ color: 'var(--semi-color-primary)' }} />
        </div>

        <Title heading={5} style={{ margin: 0 }}>
          {description}
        </Title>

        <Text type="tertiary" style={{ marginBottom: 'var(--spacing-2)' }}>
          This feature requires the <strong>{requiredPlan}</strong> plan
        </Text>

        <Button icon={<IconArrowUp />} theme="solid" type="primary" onClick={handleClick}>
          Upgrade to {requiredPlan}
        </Button>
      </Space>
    </Card>
  )
}

// ============================================================================
// Main Component
// ============================================================================

/**
 * FeatureGate Component
 *
 * Conditionally renders content based on SaaS plan feature availability.
 * Supports:
 * - Simple on/off rendering
 * - Custom fallback content
 * - Built-in upgrade prompts
 * - Inline mode for buttons/links
 * - Loading state during initialization
 *
 * @param props - Component props
 * @returns Rendered content based on feature state
 */
export function FeatureGate({
  feature,
  children,
  fallback,
  loading,
  showUpgradePrompt = false,
  inline = false,
  upgradeUrl,
  onUpgradeClick,
  style,
  className,
}: FeatureGateProps): ReactNode {
  // Single optimized store subscription with shallow comparison
  const { isEnabled, description, requiredPlan, isReady } = useFeatureStore(
    useShallow((state) => ({
      isEnabled: state.features[feature]?.enabled ?? false,
      description: state.features[feature]?.description ?? '',
      requiredPlan: state.features[feature]?.requiredPlan ?? 'enterprise',
      isReady: state.isReady,
    }))
  )

  // Show loading state only during initial load
  if (!isReady) {
    return loading ?? null
  }

  // If feature is enabled, render children
  if (isEnabled) {
    if (style || className) {
      return (
        <div style={style} className={className}>
          {children}
        </div>
      )
    }
    return children ?? null
  }

  // Feature is disabled - determine what to show
  if (fallback) {
    return fallback
  }

  if (showUpgradePrompt) {
    const prompt = (
      <UpgradePrompt
        feature={feature}
        description={description}
        requiredPlan={getPlanDisplayName(requiredPlan)}
        inline={inline}
        upgradeUrl={upgradeUrl}
        onUpgradeClick={onUpgradeClick}
      />
    )

    if (style || className) {
      return (
        <div style={style} className={className}>
          {prompt}
        </div>
      )
    }

    return prompt
  }

  // No fallback and no upgrade prompt - render nothing
  return null
}

// Add displayName for better debugging in React DevTools
FeatureGate.displayName = 'FeatureGate'
UpgradePrompt.displayName = 'UpgradePrompt'

// Default export for convenience
export default FeatureGate
