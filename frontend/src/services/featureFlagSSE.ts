/**
 * Feature Flag SSE Service
 *
 * Provides real-time feature flag updates via Server-Sent Events.
 * Falls back to polling if SSE is not available or connection fails.
 *
 * @example
 * ```tsx
 * import { createFeatureFlagSSE } from '@/services/featureFlagSSE'
 *
 * const sse = createFeatureFlagSSE({
 *   onFlagUpdate: (event) => {
 *     console.log('Flag updated:', event.key)
 *     store.refresh()
 *   },
 *   onConnectionChange: (connected) => {
 *     console.log('SSE connected:', connected)
 *   }
 * })
 *
 * sse.connect()
 * // Later...
 * sse.disconnect()
 * ```
 */

import { useAuthStore } from '@/store'

// ============================================================================
// Types
// ============================================================================

/**
 * Event data received when a flag is updated
 */
export interface FlagUpdatedEvent {
  key: string
  value: {
    enabled: boolean
    variant?: string | null
    metadata?: Record<string, unknown>
  }
}

/**
 * SSE connection state
 */
export type SSEConnectionState = 'disconnected' | 'connecting' | 'connected' | 'error'

/**
 * Options for the SSE client
 */
export interface FeatureFlagSSEOptions {
  /** Base URL for the API (default: '/api/v1') */
  baseUrl?: string
  /** Called when a flag update event is received */
  onFlagUpdate?: (event: FlagUpdatedEvent) => void
  /** Called when SSE connection state changes */
  onConnectionChange?: (state: SSEConnectionState, error?: Error) => void
  /** Called when a heartbeat is received */
  onHeartbeat?: (timestamp: number) => void
  /** Maximum reconnection attempts (default: 5) */
  maxReconnectAttempts?: number
  /** Initial reconnect delay in ms (default: 1000) */
  reconnectDelay?: number
  /** Max reconnect delay in ms (default: 30000) */
  maxReconnectDelay?: number
  /** Heartbeat timeout in ms - connection considered dead if no heartbeat (default: 60000) */
  heartbeatTimeout?: number
}

/**
 * SSE client instance
 */
export interface FeatureFlagSSEClient {
  /** Connect to SSE stream */
  connect: () => void
  /** Disconnect from SSE stream */
  disconnect: () => void
  /** Get current connection state */
  getState: () => SSEConnectionState
  /** Check if connected */
  isConnected: () => boolean
}

// ============================================================================
// Constants
// ============================================================================

const DEFAULT_BASE_URL = '/api/v1'
const DEFAULT_MAX_RECONNECT_ATTEMPTS = 5
const DEFAULT_RECONNECT_DELAY = 1000
const DEFAULT_MAX_RECONNECT_DELAY = 30000
const DEFAULT_HEARTBEAT_TIMEOUT = 60000

// ============================================================================
// SSE Client Factory
// ============================================================================

/**
 * Creates a new Feature Flag SSE client
 *
 * The client automatically handles:
 * - Token-based authentication
 * - Automatic reconnection with exponential backoff
 * - Heartbeat monitoring to detect stale connections
 * - Graceful degradation when SSE is not supported
 */
export function createFeatureFlagSSE(options: FeatureFlagSSEOptions = {}): FeatureFlagSSEClient {
  const {
    baseUrl = DEFAULT_BASE_URL,
    onFlagUpdate,
    onConnectionChange,
    onHeartbeat,
    maxReconnectAttempts = DEFAULT_MAX_RECONNECT_ATTEMPTS,
    reconnectDelay = DEFAULT_RECONNECT_DELAY,
    maxReconnectDelay = DEFAULT_MAX_RECONNECT_DELAY,
    heartbeatTimeout = DEFAULT_HEARTBEAT_TIMEOUT,
  } = options

  // State
  let eventSource: EventSource | null = null
  let state: SSEConnectionState = 'disconnected'
  let reconnectAttempts = 0
  let reconnectTimeoutId: ReturnType<typeof setTimeout> | null = null
  let heartbeatTimeoutId: ReturnType<typeof setTimeout> | null = null
  // Note: lastHeartbeat kept for potential future monitoring/debugging use
  let _lastHeartbeat: number | null = null

  // Helper to update and notify state change
  const setState = (newState: SSEConnectionState, error?: Error) => {
    if (state !== newState) {
      state = newState
      onConnectionChange?.(newState, error)
    }
  }

  // Reset heartbeat timeout
  const resetHeartbeatTimeout = () => {
    if (heartbeatTimeoutId) {
      clearTimeout(heartbeatTimeoutId)
    }
    heartbeatTimeoutId = setTimeout(() => {
      console.warn('[FeatureFlagSSE] Heartbeat timeout, reconnecting...')
      reconnect()
    }, heartbeatTimeout)
  }

  // Clear all timeouts
  const clearTimeouts = () => {
    if (reconnectTimeoutId) {
      clearTimeout(reconnectTimeoutId)
      reconnectTimeoutId = null
    }
    if (heartbeatTimeoutId) {
      clearTimeout(heartbeatTimeoutId)
      heartbeatTimeoutId = null
    }
  }

  // Calculate reconnect delay with exponential backoff and jitter
  const getReconnectDelay = () => {
    const baseDelay = Math.min(reconnectDelay * Math.pow(2, reconnectAttempts), maxReconnectDelay)
    // Add +/- 20% jitter to prevent thundering herd
    const jitter = baseDelay * 0.2 * (Math.random() * 2 - 1)
    return Math.floor(baseDelay + jitter)
  }

  // Get authentication token
  const getToken = (): string | null => {
    return useAuthStore.getState().accessToken
  }

  // Build SSE URL with auth token
  // SECURITY NOTE: EventSource API doesn't support custom headers,
  // so we pass the token as a query parameter. This is a known limitation.
  // To mitigate risks:
  // 1. The backend should use short-lived access tokens
  // 2. Server logs should sanitize URLs to remove tokens
  // 3. HTTPS must be used in production
  const buildUrl = (): string => {
    const token = getToken()
    const url = new URL(`${baseUrl}/feature-flags/stream`, window.location.origin)
    if (token) {
      url.searchParams.set('token', token)
    }
    return url.toString()
  }

  // Reconnect with backoff
  const reconnect = () => {
    disconnect()

    if (reconnectAttempts >= maxReconnectAttempts) {
      console.error('[FeatureFlagSSE] Max reconnection attempts reached')
      setState('error', new Error('Max reconnection attempts reached'))
      return
    }

    const delay = getReconnectDelay()
    console.log(
      `[FeatureFlagSSE] Reconnecting in ${delay}ms (attempt ${reconnectAttempts + 1}/${maxReconnectAttempts})`
    )

    reconnectTimeoutId = setTimeout(() => {
      reconnectAttempts++
      connect()
    }, delay)
  }

  // Connect to SSE stream
  const connect = () => {
    // Check browser support
    if (typeof EventSource === 'undefined') {
      console.warn('[FeatureFlagSSE] EventSource not supported in this browser')
      setState('error', new Error('EventSource not supported'))
      return
    }

    // Check if already connected
    if (eventSource && state === 'connected') {
      return
    }

    // Close existing connection if any
    if (eventSource) {
      eventSource.close()
    }

    setState('connecting')

    try {
      const url = buildUrl()
      eventSource = new EventSource(url, { withCredentials: true })

      // Connection opened
      eventSource.onopen = () => {
        console.log('[FeatureFlagSSE] Connected')
        setState('connected')
        reconnectAttempts = 0
        resetHeartbeatTimeout()
      }

      // Connection error
      eventSource.onerror = (event) => {
        console.error('[FeatureFlagSSE] Connection error', event)
        // EventSource automatically reconnects, but we handle it manually for more control
        if (eventSource?.readyState === EventSource.CLOSED) {
          setState('disconnected')
          reconnect()
        }
      }

      // Connected event from server
      eventSource.addEventListener('connected', (event: MessageEvent) => {
        try {
          const data = JSON.parse(event.data)
          console.log('[FeatureFlagSSE] Server confirmed connection:', data)
          resetHeartbeatTimeout()
        } catch (e) {
          console.error('[FeatureFlagSSE] Failed to parse connected event', e)
        }
      })

      // Heartbeat event
      eventSource.addEventListener('heartbeat', (event: MessageEvent) => {
        try {
          const data = JSON.parse(event.data)
          _lastHeartbeat = data.timestamp
          onHeartbeat?.(data.timestamp)
          resetHeartbeatTimeout()
        } catch (e) {
          console.error('[FeatureFlagSSE] Failed to parse heartbeat event', e)
        }
      })

      // Flag updated event
      eventSource.addEventListener('flag_updated', (event: MessageEvent) => {
        try {
          const data = JSON.parse(event.data) as FlagUpdatedEvent
          console.log('[FeatureFlagSSE] Flag updated:', data.key)
          onFlagUpdate?.(data)
          resetHeartbeatTimeout()
        } catch (e) {
          console.error('[FeatureFlagSSE] Failed to parse flag_updated event', e)
        }
      })
    } catch (error) {
      console.error('[FeatureFlagSSE] Failed to create EventSource', error)
      setState('error', error instanceof Error ? error : new Error('Failed to connect'))
      reconnect()
    }
  }

  // Disconnect from SSE stream
  const disconnect = () => {
    clearTimeouts()

    if (eventSource) {
      eventSource.close()
      eventSource = null
    }

    setState('disconnected')
    reconnectAttempts = 0
    _lastHeartbeat = null
  }

  // Get current state
  const getState = (): SSEConnectionState => state

  // Check if connected
  const isConnected = (): boolean => state === 'connected'

  return {
    connect,
    disconnect,
    getState,
    isConnected,
  }
}

// ============================================================================
// Singleton Instance
// ============================================================================

let sseInstance: FeatureFlagSSEClient | null = null

/**
 * Get or create the singleton SSE client instance
 * This is useful when you need to share the SSE connection across components
 */
export function getFeatureFlagSSE(options?: FeatureFlagSSEOptions): FeatureFlagSSEClient {
  if (!sseInstance) {
    sseInstance = createFeatureFlagSSE(options)
  }
  return sseInstance
}

/**
 * Reset the singleton instance (useful for testing)
 */
export function resetFeatureFlagSSE(): void {
  if (sseInstance) {
    sseInstance.disconnect()
    sseInstance = null
  }
}

export default createFeatureFlagSSE
