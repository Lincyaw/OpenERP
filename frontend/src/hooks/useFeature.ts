/**
 * SaaS Feature Permission Hooks
 *
 * React hooks for checking SaaS plan-based feature permissions.
 * These hooks provide reactive updates when the tenant's plan changes
 * and integrate with the feature store for consistent state management.
 *
 * This is different from feature flags (useFeatureFlag) which are for A/B testing.
 * These hooks handle subscription-based feature gating.
 *
 * @module hooks/useFeature
 *
 * @example Basic usage
 * ```tsx
 * function MyComponent() {
 *   const { enabled } = useFeature('advanced_reporting')
 *
 *   if (!enabled) {
 *     return <UpgradePrompt feature="advanced_reporting" />
 *   }
 *
 *   return <AdvancedReports />
 * }
 * ```
 *
 * @example With upgrade info
 * ```tsx
 * function FeatureButton() {
 *   const { enabled, requiredPlan, description } = useFeature('api_access')
 *
 *   if (!enabled) {
 *     return (
 *       <Tooltip content={`Upgrade to ${requiredPlan} to unlock: ${description}`}>
 *         <Button disabled>API Access (Pro)</Button>
 *       </Tooltip>
 *     )
 *   }
 *
 *   return <Button>Configure API</Button>
 * }
 * ```
 */

import { useMemo } from 'react'
import { useShallow } from 'zustand/react/shallow'
import {
  useFeatureStore,
  type FeatureKey,
  type TenantPlan,
  type FeatureDefinition,
  getPlanDisplayName,
  getNextPlan,
} from '@/store'

// ============================================================================
// Types
// ============================================================================

/**
 * Result of useFeature hook
 */
export interface UseFeatureResult {
  /** Whether the feature is enabled for the current plan */
  enabled: boolean
  /** Feature limit (null = unlimited, undefined = feature not found) */
  limit: number | null | undefined
  /** Human-readable description of the feature */
  description: string
  /** Minimum plan required to unlock this feature */
  requiredPlan: TenantPlan
  /** Human-readable name of the required plan */
  requiredPlanName: string
  /** Current tenant plan */
  currentPlan: TenantPlan
  /** Human-readable name of the current plan */
  currentPlanName: string
  /** Whether an upgrade is needed to access this feature */
  needsUpgrade: boolean
  /** The next plan to upgrade to (null if already on enterprise) */
  upgradePlan: TenantPlan | null
  /** Human-readable name of the upgrade plan */
  upgradePlanName: string | null
  /** Whether the feature store is ready */
  isReady: boolean
}

/**
 * Result of useFeatures hook (multiple features)
 */
export type UseFeaturesResult<K extends FeatureKey> = Record<K, boolean>

// ============================================================================
// Hooks
// ============================================================================

/**
 * Hook to check if a SaaS feature is available for the current tenant plan
 *
 * Returns comprehensive information about the feature including:
 * - Whether it's enabled
 * - Any limits
 * - Required plan for upgrade prompts
 * - Current plan info
 *
 * @param key - The feature key to check
 * @returns Feature availability information
 *
 * @example
 * ```tsx
 * function ImportButton() {
 *   const { enabled, limit, needsUpgrade, upgradePlanName } = useFeature('data_import')
 *
 *   if (needsUpgrade) {
 *     return <UpgradeButton plan={upgradePlanName} />
 *   }
 *
 *   return (
 *     <Button>
 *       Import Data {limit ? `(max ${limit} rows)` : '(unlimited)'}
 *     </Button>
 *   )
 * }
 * ```
 */
export function useFeature(key: FeatureKey): UseFeatureResult {
  const { feature, plan, isReady } = useFeatureStore(
    useShallow((state) => ({
      feature: state.features[key],
      plan: state.plan,
      isReady: state.isReady,
    }))
  )

  return useMemo(() => {
    const enabled = feature?.enabled ?? false
    const requiredPlan = feature?.requiredPlan ?? 'enterprise'
    const upgradePlan = getNextPlan(plan)

    return {
      enabled,
      limit: feature?.limit,
      description: feature?.description ?? '',
      requiredPlan,
      requiredPlanName: getPlanDisplayName(requiredPlan),
      currentPlan: plan,
      currentPlanName: getPlanDisplayName(plan),
      needsUpgrade: !enabled,
      upgradePlan,
      upgradePlanName: upgradePlan ? getPlanDisplayName(upgradePlan) : null,
      isReady,
    }
  }, [feature, plan, isReady])
}

/**
 * Hook to check multiple features at once
 *
 * Efficiently retrieves multiple feature states in a single hook call.
 * Returns a type-safe record mapping each key to its enabled state.
 *
 * @param keys - Array of feature keys to check
 * @returns Record mapping each key to its enabled state
 *
 * @example
 * ```tsx
 * function Dashboard() {
 *   const features = useFeatures(['advanced_reporting', 'api_access', 'integrations'])
 *
 *   return (
 *     <div>
 *       {features.advanced_reporting && <AdvancedReportsWidget />}
 *       {features.api_access && <APIStatusWidget />}
 *       {features.integrations && <IntegrationsWidget />}
 *     </div>
 *   )
 * }
 * ```
 */
export function useFeatures<K extends FeatureKey>(keys: K[]): UseFeaturesResult<K> {
  return useFeatureStore(
    useShallow((state) => {
      const result = {} as UseFeaturesResult<K>
      for (const key of keys) {
        result[key] = state.features[key]?.enabled ?? false
      }
      return result
    })
  )
}

/**
 * Hook to get all disabled features for upgrade prompts
 *
 * Returns an array of features that are not available on the current plan,
 * useful for showing what features would be unlocked by upgrading.
 *
 * @returns Array of disabled feature definitions
 *
 * @example
 * ```tsx
 * function UpgradeModal() {
 *   const disabledFeatures = useDisabledFeatures()
 *
 *   return (
 *     <Modal>
 *       <h2>Upgrade to unlock:</h2>
 *       <ul>
 *         {disabledFeatures.map((f) => (
 *           <li key={f.key}>
 *             {f.description} (requires {getPlanDisplayName(f.requiredPlan)})
 *           </li>
 *         ))}
 *       </ul>
 *     </Modal>
 *   )
 * }
 * ```
 */
export function useDisabledFeatures(): FeatureDefinition[] {
  return useFeatureStore(
    useShallow((state) => Object.values(state.features).filter((f) => !f.enabled))
  )
}

/**
 * Hook to get all enabled features
 *
 * Returns an array of features that are available on the current plan.
 *
 * @returns Array of enabled feature definitions
 *
 * @example
 * ```tsx
 * function PlanSummary() {
 *   const enabledFeatures = useEnabledFeatures()
 *
 *   return (
 *     <div>
 *       <h3>Your plan includes:</h3>
 *       <ul>
 *         {enabledFeatures.map((f) => (
 *           <li key={f.key}>{f.description}</li>
 *         ))}
 *       </ul>
 *     </div>
 *   )
 * }
 * ```
 */
export function useEnabledFeatures(): FeatureDefinition[] {
  return useFeatureStore(
    useShallow((state) => Object.values(state.features).filter((f) => f.enabled))
  )
}

/**
 * Hook to get the current tenant plan
 *
 * @returns Current plan and display name
 *
 * @example
 * ```tsx
 * function PlanBadge() {
 *   const { plan, planName } = usePlan()
 *
 *   return <Badge>{planName}</Badge>
 * }
 * ```
 */
export function usePlan(): { plan: TenantPlan; planName: string } {
  const plan = useFeatureStore((state) => state.plan)
  return useMemo(
    () => ({
      plan,
      planName: getPlanDisplayName(plan),
    }),
    [plan]
  )
}

/**
 * Hook to check if features are ready
 *
 * @returns Whether the feature store has been initialized
 *
 * @example
 * ```tsx
 * function App() {
 *   const isReady = useFeatureReady()
 *
 *   if (!isReady) {
 *     return <LoadingSpinner />
 *   }
 *
 *   return <MainApp />
 * }
 * ```
 */
export function useFeatureReady(): boolean {
  return useFeatureStore((state) => state.isReady)
}

// ============================================================================
// Re-exports for convenience
// ============================================================================

export type { FeatureKey, TenantPlan, FeatureDefinition } from '@/store'
export { getPlanDisplayName, getNextPlan, isPlanHigherOrEqual, getAllFeatureKeys } from '@/store'
