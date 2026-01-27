/**
 * Feature Component
 *
 * Declarative component for conditional rendering based on feature flags.
 * Provides a clean API for showing/hiding UI elements and supporting A/B testing variants.
 *
 * @module components/common/Feature
 *
 * @example Basic usage
 * ```tsx
 * <Feature flag="enable_new_checkout">
 *   <NewCheckout />
 * </Feature>
 * ```
 *
 * @example With fallback
 * ```tsx
 * <Feature flag="enable_new_checkout" fallback={<OldCheckout />}>
 *   <NewCheckout />
 * </Feature>
 * ```
 *
 * @example Variant rendering
 * ```tsx
 * <Feature flag="checkout_variant">
 *   {(variant) => {
 *     switch(variant) {
 *       case 'A': return <CheckoutA />;
 *       case 'B': return <CheckoutB />;
 *       default: return <CheckoutDefault />;
 *     }
 *   }}
 * </Feature>
 * ```
 *
 * @note For optimal performance with render functions, wrap the function with
 * useCallback in the parent component.
 */

import type { ReactNode } from 'react'
import { useShallow } from 'zustand/react/shallow'
import { useFeatureFlagStore } from '@/store'

// ============================================================================
// Types
// ============================================================================

/**
 * Render function for variant-based rendering
 * @param variant - The variant string or null if no variant is set
 * @returns ReactNode to render
 */
export type FeatureRenderFunction = (variant: string | null) => ReactNode

/**
 * Props for the Feature component
 */
export interface FeatureProps {
  /**
   * The feature flag key to check
   */
  flag: string

  /**
   * Content to render when the feature is disabled or flag doesn't exist.
   * If not provided, renders null when feature is off.
   */
  fallback?: ReactNode

  /**
   * Content to render while flags are loading.
   * Only shown during initial load (isReady = false), not during refreshes.
   * If not provided, renders null while loading.
   */
  loading?: ReactNode

  /**
   * Content to render when the feature is enabled.
   * Can be:
   * - ReactNode: Static content to render
   * - Function: Render function that receives the variant value for A/B testing
   */
  children: ReactNode | FeatureRenderFunction
}

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Type guard to check if children is a render function
 */
const isRenderFunction = (
  children: ReactNode | FeatureRenderFunction
): children is FeatureRenderFunction => {
  return typeof children === 'function'
}

// ============================================================================
// Component
// ============================================================================

/**
 * Feature Component
 *
 * Conditionally renders content based on feature flag state.
 * Supports:
 * - Simple on/off rendering
 * - Fallback content for disabled features
 * - Loading state during initial flag fetch
 * - Variant-based rendering for A/B testing
 *
 * The component subscribes to the feature flag store and automatically
 * re-renders when flag state changes. Uses a single optimized store subscription
 * with shallow comparison for better performance.
 *
 * @param props - Component props
 * @returns Rendered content based on flag state
 */
export function Feature({ flag, fallback, loading, children }: FeatureProps): ReactNode {
  // Single optimized store subscription with shallow comparison
  // This prevents unnecessary re-renders when unrelated store state changes
  const { isEnabled, variant, isReady } = useFeatureFlagStore(
    useShallow((state) => ({
      isEnabled: state.flags[flag]?.enabled ?? false,
      variant: state.flags[flag]?.variant ?? null,
      isReady: state.isReady,
    }))
  )

  // Show loading state only during initial fetch (not during refresh)
  // isReady becomes true once flags are loaded (from server or cache)
  if (!isReady) {
    return loading ?? null
  }

  // If flag is disabled or doesn't exist, show fallback
  if (!isEnabled) {
    return fallback ?? null
  }

  // If children is a render function, call it with the variant
  if (isRenderFunction(children)) {
    return children(variant)
  }

  // Otherwise, render children as-is
  return children ?? null
}

// Add displayName for better debugging in React DevTools
Feature.displayName = 'Feature'

// Default export for convenience
export default Feature
