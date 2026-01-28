/**
 * FeatureFlagProvider Component
 *
 * Context provider that initializes feature flags on application startup.
 * Supports real-time updates via Server-Sent Events (SSE) with automatic
 * fallback to polling when SSE is unavailable.
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

import { useEffect, useRef, useCallback, type ReactNode } from 'react'
import { useFeatureFlagStore } from '@/store'
import {
  createFeatureFlagSSE,
  type FeatureFlagSSEClient,
  type SSEConnectionState,
  type FlagUpdatedEvent,
} from '@/services/featureFlagSSE'

// ============================================================================
// Constants
// ============================================================================

const DEFAULT_POLLING_INTERVAL = 30000 // 30 seconds
const DEFAULT_SSE_HEARTBEAT_TIMEOUT = 60000 // 60 seconds

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
  /** Whether to enable real-time updates. Default: true */
  enableRealtime?: boolean
  /** Whether to prefer SSE over polling. Default: true */
  preferSSE?: boolean
  /** Component to show while flags are loading initially */
  loadingComponent?: ReactNode
  /** SSE heartbeat timeout in ms. Default: 60000 (60 seconds) */
  sseHeartbeatTimeout?: number
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
 * - Real-time updates via SSE (Server-Sent Events)
 * - Automatic fallback to polling if SSE fails
 * - Graceful error handling - initialization failures don't block the app
 * - Cleans up resources on unmount
 *
 * Update Strategy (priority order):
 * 1. SSE (if preferSSE=true and browser supports it)
 * 2. Polling (fallback when SSE fails or is disabled)
 *
 * Error Handling:
 * - If initialization fails, the app continues with default values (all flags disabled)
 * - If cached flags exist (from sessionStorage), they are used as fallback
 * - SSE connection failures automatically trigger polling fallback
 * - Errors are logged to console for debugging
 *
 * @example
 * ```tsx
 * // Basic usage (SSE enabled by default)
 * <FeatureFlagProvider>
 *   <App />
 * </FeatureFlagProvider>
 *
 * // With custom polling interval (used as fallback when SSE unavailable)
 * <FeatureFlagProvider pollingInterval={60000}>
 *   <App />
 * </FeatureFlagProvider>
 *
 * // Disable SSE, use polling only
 * <FeatureFlagProvider preferSSE={false}>
 *   <App />
 * </FeatureFlagProvider>
 *
 * // Disable real-time updates entirely
 * <FeatureFlagProvider enableRealtime={false}>
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
  enableRealtime = true,
  preferSSE = true,
  loadingComponent,
  sseHeartbeatTimeout = DEFAULT_SSE_HEARTBEAT_TIMEOUT,
}: FeatureFlagProviderProps) {
  // Track if we've started initialization to prevent double-init in strict mode
  const hasInitialized = useRef(false)
  // Track if SSE has failed and we should use polling
  const sseFailedRef = useRef(false)
  // SSE client reference
  const sseClientRef = useRef<FeatureFlagSSEClient | null>(null)

  // Store selectors
  const initialize = useFeatureFlagStore((state) => state.initialize)
  const refresh = useFeatureFlagStore((state) => state.refresh)
  const startPolling = useFeatureFlagStore((state) => state.startPolling)
  const stopPolling = useFeatureFlagStore((state) => state.stopPolling)
  const isReady = useFeatureFlagStore((state) => state.isReady)
  const isLoading = useFeatureFlagStore((state) => state.isLoading)
  const error = useFeatureFlagStore((state) => state.error)

  // Handle SSE flag update
  const handleFlagUpdate = useCallback(
    (event: FlagUpdatedEvent) => {
      console.log('[FeatureFlagProvider] Received flag update via SSE:', event.key)
      // Refresh all flags when any flag is updated
      // The backend sends minimal event data, so we fetch the full state
      refresh()
    },
    [refresh]
  )

  // Handle SSE connection state change
  const handleConnectionChange = useCallback(
    (state: SSEConnectionState, sseError?: Error) => {
      console.log('[FeatureFlagProvider] SSE connection state:', state, sseError?.message)

      if (state === 'error') {
        // SSE failed, fall back to polling
        if (!sseFailedRef.current) {
          sseFailedRef.current = true
          console.warn('[FeatureFlagProvider] SSE failed, falling back to polling')
          startPolling(pollingInterval)
        }
      } else if (state === 'connected') {
        // SSE connected, stop polling if it was running
        sseFailedRef.current = false
        stopPolling()
      }
    },
    [pollingInterval, startPolling, stopPolling]
  )

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

  // Setup SSE or polling based on configuration and ready state
  useEffect(() => {
    if (!enableRealtime || !isReady) {
      return
    }

    // Check if SSE is preferred and supported
    const sseSupported = typeof EventSource !== 'undefined'

    if (preferSSE && sseSupported && !sseFailedRef.current) {
      // Try SSE first
      console.log('[FeatureFlagProvider] Attempting SSE connection...')

      const sseClient = createFeatureFlagSSE({
        onFlagUpdate: handleFlagUpdate,
        onConnectionChange: handleConnectionChange,
        heartbeatTimeout: sseHeartbeatTimeout,
      })

      sseClientRef.current = sseClient
      sseClient.connect()

      return () => {
        sseClient.disconnect()
        sseClientRef.current = null
      }
    } else {
      // Use polling
      if (pollingInterval > 0) {
        console.log('[FeatureFlagProvider] Using polling for flag updates')
        startPolling(pollingInterval)
      }

      return () => {
        stopPolling()
      }
    }
  }, [
    enableRealtime,
    isReady,
    preferSSE,
    pollingInterval,
    sseHeartbeatTimeout,
    handleFlagUpdate,
    handleConnectionChange,
    startPolling,
    stopPolling,
  ])

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
