import { create } from 'zustand'
import { devtools, persist, createJSONStorage } from 'zustand/middleware'
import { axiosInstance } from '@/services/axios-instance'

// ============================================================================
// Types
// ============================================================================

/**
 * Value of a feature flag as returned by the API
 */
export interface FlagValue {
  enabled: boolean
  variant: string | null
  metadata?: Record<string, unknown>
}

/**
 * Feature flag state
 */
export interface FeatureFlagState {
  /** Map of flag keys to their values */
  flags: Record<string, FlagValue>
  /** Whether the initial load is in progress */
  isLoading: boolean
  /** Whether the store has been initialized with server data */
  isReady: boolean
  /** Last time flags were updated from server */
  lastUpdated: Date | null
  /** Error message if last fetch failed */
  error: string | null
}

/**
 * Feature flag actions
 */
export interface FeatureFlagActions {
  /** Initialize flags from server (call on app startup) */
  initialize: () => Promise<void>
  /** Refresh flags from server */
  refresh: () => Promise<void>
  /** Check if a flag is enabled */
  isEnabled: (key: string) => boolean
  /** Get variant value for a flag */
  getVariant: (key: string) => string | null
  /** Get full flag value */
  getFlagValue: (key: string) => FlagValue | null
  /** Manually set flags (for testing or SSR hydration) */
  setFlags: (flags: Record<string, FlagValue>) => void
  /** Clear error state */
  clearError: () => void
  /** Start polling for updates */
  startPolling: (intervalMs?: number) => void
  /** Stop polling for updates */
  stopPolling: () => void
}

// ============================================================================
// API Types (matching backend dto.GetClientConfigResponse)
// ============================================================================

interface ClientConfigFlag {
  enabled: boolean
  variant?: string
}

interface ClientConfigResponse {
  flags: Record<string, ClientConfigFlag>
  evaluated_at: string
}

interface ApiResponse<T> {
  success: boolean
  data?: T
  error?: string
}

// ============================================================================
// Constants
// ============================================================================

const STORAGE_KEY = 'erp-feature-flags'
const DEFAULT_POLL_INTERVAL = 30000 // 30 seconds
const API_ENDPOINT = '/feature-flags/client-config'

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Normalize server flag response to FlagValue format
 */
const normalizeFlags = (
  serverFlags: Record<string, ClientConfigFlag>
): Record<string, FlagValue> => {
  const normalized: Record<string, FlagValue> = {}
  for (const key of Object.keys(serverFlags)) {
    const value = serverFlags[key]
    normalized[key] = {
      enabled: value.enabled,
      variant: value.variant ?? null,
    }
  }
  return normalized
}

// ============================================================================
// Initial State
// ============================================================================

const initialState: FeatureFlagState = {
  flags: {},
  isLoading: false,
  isReady: false,
  lastUpdated: null,
  error: null,
}

// ============================================================================
// Polling State (module-level to avoid store state bloat)
// ============================================================================

let pollingIntervalId: ReturnType<typeof setInterval> | null = null

// ============================================================================
// Store Implementation
// ============================================================================

/**
 * Feature Flag Store
 *
 * Manages feature flag state with:
 * - Server initialization via /api/v1/feature-flags/client-config
 * - sessionStorage caching for fast page refresh recovery
 * - Optional polling for real-time updates
 * - Graceful error handling with cache fallback
 *
 * @example
 * ```tsx
 * import { useFeatureFlagStore } from '@/store'
 *
 * function MyComponent() {
 *   const isEnabled = useFeatureFlagStore((state) => state.isEnabled)
 *
 *   if (isEnabled('new_checkout_flow')) {
 *     return <NewCheckoutFlow />
 *   }
 *
 *   return <LegacyCheckoutFlow />
 * }
 * ```
 *
 * @example
 * ```tsx
 * // Initialize on app startup
 * import { useFeatureFlagStore } from '@/store'
 *
 * function App() {
 *   const initialize = useFeatureFlagStore((state) => state.initialize)
 *   const isReady = useFeatureFlagStore((state) => state.isReady)
 *
 *   useEffect(() => {
 *     initialize()
 *   }, [initialize])
 *
 *   if (!isReady) {
 *     return <LoadingSpinner />
 *   }
 *
 *   return <MainApp />
 * }
 * ```
 */
export const useFeatureFlagStore = create<FeatureFlagState & FeatureFlagActions>()(
  devtools(
    persist(
      (set, get) => ({
        ...initialState,

        initialize: async () => {
          const state = get()

          // If already loading, skip
          if (state.isLoading) {
            return
          }

          // If we have cached flags from sessionStorage, mark as ready immediately
          // (the persist middleware will have restored them)
          if (Object.keys(state.flags).length > 0 && !state.isReady) {
            set({ isReady: true }, false, 'featureFlags/cacheRestored')
          }

          // Fetch fresh flags from server
          set({ isLoading: true, error: null }, false, 'featureFlags/initializeStart')

          try {
            const response = await axiosInstance.post<ApiResponse<ClientConfigResponse>>(
              API_ENDPOINT,
              { context: {} } // Empty context - server will enrich from JWT
            )

            if (response.data.success && response.data.data) {
              const normalizedFlags = normalizeFlags(response.data.data.flags)

              set(
                {
                  flags: normalizedFlags,
                  isLoading: false,
                  isReady: true,
                  lastUpdated: new Date(),
                  error: null,
                },
                false,
                'featureFlags/initializeSuccess'
              )
            } else {
              throw new Error(response.data.error || 'Failed to fetch feature flags')
            }
          } catch (error) {
            const errorMessage = error instanceof Error ? error.message : 'Unknown error'

            // If we have cached flags, use them and mark as ready
            const hasCache = Object.keys(get().flags).length > 0
            set(
              {
                isLoading: false,
                isReady: hasCache, // Ready if we have cache fallback
                error: errorMessage,
              },
              false,
              'featureFlags/initializeError'
            )

            // Don't throw - allow app to continue with cached flags
            // Note: Error is captured in state.error for UI display
          }
        },

        refresh: async () => {
          const state = get()

          // Don't refresh if not initialized yet
          if (!state.isReady && Object.keys(state.flags).length === 0) {
            return get().initialize()
          }

          // Don't set isLoading for refresh to avoid UI flicker
          set({ error: null }, false, 'featureFlags/refreshStart')

          try {
            const response = await axiosInstance.post<ApiResponse<ClientConfigResponse>>(
              API_ENDPOINT,
              { context: {} }
            )

            if (response.data.success && response.data.data) {
              const normalizedFlags = normalizeFlags(response.data.data.flags)

              set(
                {
                  flags: normalizedFlags,
                  lastUpdated: new Date(),
                  error: null,
                },
                false,
                'featureFlags/refreshSuccess'
              )
            }
          } catch (error) {
            const errorMessage = error instanceof Error ? error.message : 'Unknown error'
            set({ error: errorMessage }, false, 'featureFlags/refreshError')
            // Don't throw - keep using existing flags
          }
        },

        isEnabled: (key: string) => {
          const flag = get().flags[key]
          // Default to false if flag doesn't exist
          return flag?.enabled ?? false
        },

        getVariant: (key: string) => {
          const flag = get().flags[key]
          return flag?.variant ?? null
        },

        getFlagValue: (key: string) => {
          return get().flags[key] ?? null
        },

        setFlags: (flags: Record<string, FlagValue>) => {
          set(
            {
              flags,
              isReady: true,
              lastUpdated: new Date(),
            },
            false,
            'featureFlags/setFlags'
          )
        },

        clearError: () => {
          set({ error: null }, false, 'featureFlags/clearError')
        },

        startPolling: (intervalMs = DEFAULT_POLL_INTERVAL) => {
          // Clear any existing polling
          if (pollingIntervalId) {
            clearInterval(pollingIntervalId)
          }

          // Start new polling interval
          pollingIntervalId = setInterval(() => {
            get().refresh()
          }, intervalMs)
        },

        stopPolling: () => {
          if (pollingIntervalId) {
            clearInterval(pollingIntervalId)
            pollingIntervalId = null
          }
        },
      }),
      {
        name: STORAGE_KEY,
        storage: createJSONStorage(() => sessionStorage),
        // Only persist flags and lastUpdated
        partialize: (state) => ({
          flags: state.flags,
          lastUpdated: state.lastUpdated,
        }),
        // After rehydration, update isReady if we have cached flags
        onRehydrateStorage: () => (state) => {
          if (state && Object.keys(state.flags).length > 0) {
            // This runs after the store is created
            // We can't set state here, but initialize() will check for cached flags
          }
        },
      }
    ),
    { name: 'FeatureFlagStore' }
  )
)

// ============================================================================
// Selector Hooks
// ============================================================================

/**
 * Hook to check if a specific flag is enabled
 *
 * @example
 * ```tsx
 * const isNewCheckoutEnabled = useIsFeatureEnabled('new_checkout_flow')
 * ```
 */
export const useIsFeatureEnabled = (key: string) =>
  useFeatureFlagStore((state) => state.flags[key]?.enabled ?? false)

/**
 * Hook to get a flag's variant value
 *
 * @example
 * ```tsx
 * const variant = useFeatureVariant('button_color')
 * // variant might be 'blue', 'green', or null
 * ```
 */
export const useFeatureVariant = (key: string) =>
  useFeatureFlagStore((state) => state.flags[key]?.variant ?? null)

/**
 * Hook to get the full flag value
 *
 * @example
 * ```tsx
 * const flag = useFeatureFlag('new_checkout_flow')
 * if (flag?.enabled) {
 *   // ...
 * }
 * ```
 */
export const useFeatureFlag = (key: string) => useFeatureFlagStore((state) => state.flags[key])

/**
 * Hook to get loading state
 *
 * @returns `true` during initial fetch
 *
 * @example
 * ```tsx
 * const isLoading = useFeatureFlagsLoading()
 * if (isLoading) return <Spinner />
 * ```
 */
export const useFeatureFlagsLoading = () => useFeatureFlagStore((state) => state.isLoading)

/**
 * Hook to get ready state
 *
 * @returns `true` when flags have been loaded (from server or cache)
 *
 * @example
 * ```tsx
 * const isReady = useFeatureFlagsReady()
 * if (!isReady) return <LoadingState />
 * ```
 */
export const useFeatureFlagsReady = () => useFeatureFlagStore((state) => state.isReady)

/**
 * Hook to get error state
 *
 * @returns Error message if last fetch failed, null otherwise
 *
 * @example
 * ```tsx
 * const error = useFeatureFlagsError()
 * if (error) return <ErrorBanner message={error} />
 * ```
 */
export const useFeatureFlagsError = () => useFeatureFlagStore((state) => state.error)
