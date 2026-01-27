/**
 * FeatureFlagProvider Component
 *
 * Context provider that initializes feature flags on application startup.
 * Manages polling for real-time updates and provides graceful error handling.
 *
 * @example
 * ```tsx
 * // In main.tsx
 * createRoot(document.getElementById('root')!).render(
 *   <StrictMode>
 *     <FeatureFlagProvider>
 *       <App />
 *     </FeatureFlagProvider>
 *   </StrictMode>
 * )
 * ```
 */

import { useEffect, useRef, type ReactNode } from 'react'
import { useFeatureFlagStore } from '@/store'

// ============================================================================
// Constants
// ============================================================================

const DEFAULT_POLLING_INTERVAL = 30000 // 30 seconds

// ============================================================================
// Types
// ============================================================================

/**
 * Props for FeatureFlagProvider
 */
export interface FeatureFlagProviderProps {
  /** Child components to render */
  children: ReactNode
  /** Polling interval in milliseconds. Default: 30000 (30 seconds) */
  pollingInterval?: number
  /** Whether to enable polling for flag updates. Default: true */
  enablePolling?: boolean
  /** Component to show while flags are loading initially */
  loadingComponent?: ReactNode
}

// ============================================================================
// Component
// ============================================================================

/**
 * FeatureFlagProvider
 *
 * Initializes the feature flag system when the application starts.
 *
 * Features:
 * - Automatically initializes flags from server on mount
 * - Optionally polls for flag updates at configurable intervals
 * - Graceful error handling - initialization failures don't block the app
 * - Cleans up polling on unmount
 *
 * Error Handling:
 * - If initialization fails, the app continues with default values (all flags disabled)
 * - If cached flags exist (from sessionStorage), they are used as fallback
 * - Errors are logged to console for debugging
 *
 * @example
 * ```tsx
 * // Basic usage
 * <FeatureFlagProvider>
 *   <App />
 * </FeatureFlagProvider>
 *
 * // With custom polling interval (60 seconds)
 * <FeatureFlagProvider pollingInterval={60000}>
 *   <App />
 * </FeatureFlagProvider>
 *
 * // Disable polling
 * <FeatureFlagProvider enablePolling={false}>
 *   <App />
 * </FeatureFlagProvider>
 *
 * // With loading component
 * <FeatureFlagProvider loadingComponent={<LoadingSpinner />}>
 *   <App />
 * </FeatureFlagProvider>
 * ```
 */
export function FeatureFlagProvider({
  children,
  pollingInterval = DEFAULT_POLLING_INTERVAL,
  enablePolling = true,
  loadingComponent,
}: FeatureFlagProviderProps) {
  // Track if we've started initialization to prevent double-init in strict mode
  const hasInitialized = useRef(false)

  // Store selectors
  const initialize = useFeatureFlagStore((state) => state.initialize)
  const startPolling = useFeatureFlagStore((state) => state.startPolling)
  const stopPolling = useFeatureFlagStore((state) => state.stopPolling)
  const isReady = useFeatureFlagStore((state) => state.isReady)
  const isLoading = useFeatureFlagStore((state) => state.isLoading)
  const error = useFeatureFlagStore((state) => state.error)

  // Initialize flags on mount
  useEffect(() => {
    // Prevent double initialization in React Strict Mode
    if (hasInitialized.current) {
      return
    }
    hasInitialized.current = true

    const initializeFlags = async () => {
      try {
        await initialize()
      } catch (err) {
        // Error is already captured in the store state
        // Log to console for debugging but don't rethrow
        // This ensures the app continues running with default values

        console.error('[FeatureFlagProvider] Failed to initialize feature flags:', err)
      }
    }

    initializeFlags()
  }, [initialize])

  // Start/stop polling based on props and ready state
  useEffect(() => {
    // Only start polling if:
    // 1. Polling is enabled
    // 2. Flags are ready (initialized)
    // 3. Polling interval is valid
    if (enablePolling && isReady && pollingInterval > 0) {
      startPolling(pollingInterval)
    }

    // Cleanup: stop polling on unmount or when polling is disabled
    return () => {
      stopPolling()
    }
  }, [enablePolling, isReady, pollingInterval, startPolling, stopPolling])

  // Log errors for debugging
  useEffect(() => {
    if (error) {
      console.warn('[FeatureFlagProvider] Feature flag error:', error)
    }
  }, [error])

  // Show loading component only during initial load (not ready and loading)
  // Don't show loading during refresh (isReady=true, isLoading=true)
  if (!isReady && isLoading && loadingComponent) {
    return <>{loadingComponent}</>
  }

  // Render children even if not ready (with default values)
  // This ensures the app is not blocked by feature flag failures
  return <>{children}</>
}

// Display name for React DevTools
FeatureFlagProvider.displayName = 'FeatureFlagProvider'

export default FeatureFlagProvider
