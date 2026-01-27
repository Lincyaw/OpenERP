/**
 * Feature Flag React Hooks
 *
 * Type-safe React hooks for feature flag evaluation.
 * These hooks provide reactive updates when flags change and
 * integrate with the feature flag store for consistent state management.
 *
 * @module hooks/useFeatureFlag
 */

import { useShallow } from 'zustand/react/shallow'
import { useFeatureFlagStore, type FlagValue } from '@/store'

// ============================================================================
// Types
// ============================================================================

/**
 * Full feature flag value including enabled state and variant.
 * Re-exported from the store for convenience.
 */
export type FeatureFlagValue = FlagValue

// ============================================================================
// Hooks
// ============================================================================

/**
 * Hook to check if a feature flag is enabled
 *
 * Returns whether the specified feature flag is enabled, with an optional
 * default value if the flag doesn't exist. The hook automatically subscribes
 * to store updates and re-renders when the flag state changes.
 *
 * @param key - The unique identifier of the feature flag
 * @param defaultValue - Optional default value if the flag doesn't exist (defaults to false)
 * @returns Whether the feature flag is enabled
 *
 * @example
 * ```tsx
 * function NewFeatureButton() {
 *   const isEnabled = useFeatureFlag('new_checkout_flow')
 *
 *   if (!isEnabled) {
 *     return null
 *   }
 *
 *   return <button>Try New Checkout</button>
 * }
 * ```
 *
 * @example
 * ```tsx
 * // With default value
 * function OptionalFeature() {
 *   const isEnabled = useFeatureFlag('experimental_feature', true)
 *
 *   return isEnabled ? <NewVersion /> : <OldVersion />
 * }
 * ```
 */
export function useFeatureFlag(key: string, defaultValue: boolean = false): boolean {
  return useFeatureFlagStore((state) => state.flags[key]?.enabled ?? defaultValue)
}

/**
 * Hook to get a feature flag's variant value
 *
 * Returns the variant string for A/B testing or multivariate experiments.
 * Returns null if the flag doesn't exist or has no variant configured.
 *
 * @param key - The unique identifier of the feature flag
 * @returns The variant name or null if not set
 *
 * @example
 * ```tsx
 * function ABTestButton() {
 *   const variant = useFeatureVariant('button_color_test')
 *
 *   const buttonColor = variant === 'blue' ? 'primary' : 'secondary'
 *
 *   return <Button color={buttonColor}>Click Me</Button>
 * }
 * ```
 *
 * @example
 * ```tsx
 * // With switch statement for multiple variants
 * function PricingPage() {
 *   const variant = useFeatureVariant('pricing_layout')
 *
 *   switch (variant) {
 *     case 'grid':
 *       return <PricingGrid />
 *     case 'list':
 *       return <PricingList />
 *     case 'cards':
 *       return <PricingCards />
 *     default:
 *       return <PricingDefault />
 *   }
 * }
 * ```
 */
export function useFeatureVariant(key: string): string | null {
  return useFeatureFlagStore((state) => state.flags[key]?.variant ?? null)
}

/**
 * Hook to get multiple feature flags at once
 *
 * Efficiently retrieves multiple flag values in a single hook call.
 * Returns a type-safe record mapping each key to its enabled state.
 * Uses Zustand's shallow comparison to prevent unnecessary re-renders
 * when the requested flag values haven't changed.
 *
 * @typeParam K - String literal type for the flag keys
 * @param keys - Array of feature flag keys to retrieve.
 *               Empty arrays return empty object.
 *               Duplicate keys are deduplicated.
 * @returns Record mapping each key to its enabled state (boolean)
 *
 * @example
 * ```tsx
 * function Dashboard() {
 *   const flags = useFeatureFlags(['analytics', 'notifications', 'dark_mode'])
 *
 *   return (
 *     <div>
 *       {flags.analytics && <AnalyticsWidget />}
 *       {flags.notifications && <NotificationBell />}
 *       {flags.dark_mode && <DarkModeToggle />}
 *     </div>
 *   )
 * }
 * ```
 *
 * @example
 * ```tsx
 * // Type-safe usage with const assertion
 * const FLAG_KEYS = ['feature_a', 'feature_b', 'feature_c'] as const
 *
 * function MyComponent() {
 *   const flags = useFeatureFlags([...FLAG_KEYS])
 *   // flags is typed as Record<'feature_a' | 'feature_b' | 'feature_c', boolean>
 *
 *   if (flags.feature_a) {
 *     // TypeScript knows this is valid
 *   }
 * }
 * ```
 */
export function useFeatureFlags<K extends string>(keys: K[]): Record<K, boolean> {
  // Use Zustand's useShallow to only re-render when requested flag values change
  // The selector extracts only the flags we need, and useShallow performs shallow equality
  return useFeatureFlagStore(
    useShallow((state) => {
      const result = {} as Record<K, boolean>
      for (const key of keys) {
        result[key] = state.flags[key]?.enabled ?? false
      }
      return result
    })
  )
}

/**
 * Hook to check if feature flags have been loaded
 *
 * Returns whether the feature flag store has completed its initial load
 * (either from the server or from cache). Use this to conditionally render
 * loading states while waiting for flag data.
 *
 * @returns Whether flags are ready for use
 *
 * @example
 * ```tsx
 * function App() {
 *   const isReady = useFeatureFlagReady()
 *
 *   if (!isReady) {
 *     return <LoadingSpinner />
 *   }
 *
 *   return <MainApp />
 * }
 * ```
 *
 * @example
 * ```tsx
 * // Combined with Suspense-like pattern
 * function FeatureAwareComponent() {
 *   const isReady = useFeatureFlagReady()
 *   const isNewFeatureEnabled = useFeatureFlag('new_feature')
 *
 *   if (!isReady) {
 *     // Show skeleton while loading
 *     return <ComponentSkeleton />
 *   }
 *
 *   return isNewFeatureEnabled ? <NewFeature /> : <LegacyFeature />
 * }
 * ```
 */
export function useFeatureFlagReady(): boolean {
  return useFeatureFlagStore((state) => state.isReady)
}

// ============================================================================
// Additional Utility Hooks
// ============================================================================

/**
 * Hook to get the full feature flag value including metadata
 *
 * Returns the complete flag object including enabled state, variant, and
 * any custom metadata. Useful when you need access to all flag properties.
 *
 * @param key - The unique identifier of the feature flag
 * @returns Full flag value or null if not found
 *
 * @example
 * ```tsx
 * function FeatureWithMetadata() {
 *   const flag = useFeatureFlagValue('complex_feature')
 *
 *   if (!flag?.enabled) {
 *     return null
 *   }
 *
 *   const config = flag.metadata as { maxItems?: number }
 *
 *   return <FeatureComponent maxItems={config?.maxItems ?? 10} />
 * }
 * ```
 */
export function useFeatureFlagValue(key: string): FeatureFlagValue | null {
  const flag = useFeatureFlagStore((state) => state.flags[key])
  return flag ?? null
}

/**
 * Hook to get the feature flag loading state
 *
 * Returns whether the initial flag fetch is currently in progress.
 * Different from `useFeatureFlagReady` - this is only true during the
 * initial API call, not when refreshing.
 *
 * @returns Whether flags are currently being loaded
 *
 * @example
 * ```tsx
 * function FeatureToggle() {
 *   const isLoading = useFeatureFlagLoading()
 *   const isEnabled = useFeatureFlag('my_feature')
 *
 *   if (isLoading) {
 *     return <Skeleton />
 *   }
 *
 *   return <Toggle checked={isEnabled} />
 * }
 * ```
 */
export function useFeatureFlagLoading(): boolean {
  return useFeatureFlagStore((state) => state.isLoading)
}

/**
 * Hook to get any error from feature flag loading
 *
 * Returns the error message if the last flag fetch failed,
 * or null if there's no error. Useful for displaying error banners
 * or fallback UI when flag loading fails.
 *
 * @returns Error message or null
 *
 * @example
 * ```tsx
 * function FeatureFlagStatus() {
 *   const error = useFeatureFlagError()
 *
 *   if (error) {
 *     return (
 *       <Banner type="warning">
 *         Feature flags unavailable: {error}
 *       </Banner>
 *     )
 *   }
 *
 *   return null
 * }
 * ```
 */
export function useFeatureFlagError(): string | null {
  return useFeatureFlagStore((state) => state.error)
}
