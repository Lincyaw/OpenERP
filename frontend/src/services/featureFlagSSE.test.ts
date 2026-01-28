/**
 * Feature Flag SSE Service Tests
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

// Mock useAuthStore before imports
vi.mock('@/store', () => ({
  useAuthStore: {
    getState: vi.fn(() => ({
      accessToken: 'test-token-123',
    })),
  },
}))

// Store for capturing event listeners
let eventListeners: Map<string, ((event: MessageEvent) => void)[]>
let mockEventSourceInstance: {
  close: ReturnType<typeof vi.fn>
  onopen: ((event: Event) => void) | null
  onerror: ((event: Event) => void) | null
  readyState: number
  CONNECTING: number
  OPEN: number
  CLOSED: number
}

// Create mock EventSource class
class MockEventSource {
  static readonly CONNECTING = 0
  static readonly OPEN = 1
  static readonly CLOSED = 2

  onopen: ((event: Event) => void) | null = null
  onerror: ((event: Event) => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  readyState = MockEventSource.OPEN
  url: string
  withCredentials: boolean

  close = vi.fn()

  constructor(url: string, options?: { withCredentials?: boolean }) {
    this.url = url
    this.withCredentials = options?.withCredentials ?? false
    eventListeners = new Map()
    // Store reference to this instance for test assertions
    // eslint-disable-next-line @typescript-eslint/no-this-alias
    mockEventSourceInstance = this
  }

  addEventListener(type: string, listener: (event: MessageEvent) => void) {
    const listeners = eventListeners.get(type) || []
    listeners.push(listener)
    eventListeners.set(type, listeners)
  }

  removeEventListener(type: string, listener: (event: MessageEvent) => void) {
    const listeners = eventListeners.get(type) || []
    const index = listeners.indexOf(listener)
    if (index > -1) {
      listeners.splice(index, 1)
    }
    eventListeners.set(type, listeners)
  }

  // Helper to dispatch events in tests
  dispatchEvent(type: string, data?: string) {
    const listeners = eventListeners.get(type) || []
    const event = new MessageEvent(type, { data })
    listeners.forEach((listener) => listener(event))
  }
}

// Mock EventSource globally
vi.stubGlobal('EventSource', MockEventSource)

// Import after mocking
import { createFeatureFlagSSE, resetFeatureFlagSSE, getFeatureFlagSSE } from './featureFlagSSE'

describe('featureFlagSSE', () => {
  beforeEach(() => {
    // Reset singleton
    resetFeatureFlagSSE()
    eventListeners = new Map()
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  describe('createFeatureFlagSSE', () => {
    it('creates an SSE client with default options', () => {
      const client = createFeatureFlagSSE()

      expect(client).toBeDefined()
      expect(client.connect).toBeInstanceOf(Function)
      expect(client.disconnect).toBeInstanceOf(Function)
      expect(client.getState).toBeInstanceOf(Function)
      expect(client.isConnected).toBeInstanceOf(Function)
    })

    it('starts with disconnected state', () => {
      const client = createFeatureFlagSSE()

      expect(client.getState()).toBe('disconnected')
      expect(client.isConnected()).toBe(false)
    })
  })

  describe('connect', () => {
    it('creates EventSource with correct URL containing token', () => {
      const client = createFeatureFlagSSE()
      client.connect()

      expect(mockEventSourceInstance.url).toContain('/api/v1/feature-flags/stream')
      expect(mockEventSourceInstance.url).toContain('token=test-token-123')
      expect(mockEventSourceInstance.withCredentials).toBe(true)
    })

    it('transitions to connecting state', () => {
      const onConnectionChange = vi.fn()

      const client = createFeatureFlagSSE({
        onConnectionChange,
      })

      client.connect()

      expect(onConnectionChange).toHaveBeenCalledWith('connecting', undefined)
    })

    it('transitions to connected state on open', () => {
      const onConnectionChange = vi.fn()

      const client = createFeatureFlagSSE({
        onConnectionChange,
      })

      client.connect()

      // Simulate connection open
      mockEventSourceInstance.onopen?.(new Event('open'))

      expect(onConnectionChange).toHaveBeenCalledWith('connected', undefined)
      expect(client.getState()).toBe('connected')
      expect(client.isConnected()).toBe(true)
    })

    it('registers event listeners for SSE events', () => {
      const client = createFeatureFlagSSE()
      client.connect()

      expect(eventListeners.has('connected')).toBe(true)
      expect(eventListeners.has('heartbeat')).toBe(true)
      expect(eventListeners.has('flag_updated')).toBe(true)
    })
  })

  describe('disconnect', () => {
    it('closes EventSource connection', () => {
      const client = createFeatureFlagSSE()

      client.connect()
      client.disconnect()

      expect(mockEventSourceInstance.close).toHaveBeenCalled()
    })

    it('transitions to disconnected state', () => {
      const onConnectionChange = vi.fn()

      const client = createFeatureFlagSSE({
        onConnectionChange,
      })

      client.connect()
      mockEventSourceInstance.onopen?.(new Event('open'))

      onConnectionChange.mockClear()

      client.disconnect()

      expect(onConnectionChange).toHaveBeenCalledWith('disconnected', undefined)
      expect(client.getState()).toBe('disconnected')
    })
  })

  describe('event handling', () => {
    it('calls onFlagUpdate when flag_updated event received', () => {
      const onFlagUpdate = vi.fn()

      const client = createFeatureFlagSSE({
        onFlagUpdate,
      })

      client.connect()

      // Dispatch flag_updated event
      const listeners = eventListeners.get('flag_updated') || []
      expect(listeners.length).toBeGreaterThan(0)

      const event = new MessageEvent('flag_updated', {
        data: JSON.stringify({ key: 'test_flag', value: { enabled: true } }),
      })
      listeners[0](event)

      expect(onFlagUpdate).toHaveBeenCalledWith({
        key: 'test_flag',
        value: { enabled: true },
      })
    })

    it('calls onHeartbeat when heartbeat event received', () => {
      const onHeartbeat = vi.fn()

      const client = createFeatureFlagSSE({
        onHeartbeat,
      })

      client.connect()

      // Dispatch heartbeat event
      const listeners = eventListeners.get('heartbeat') || []
      expect(listeners.length).toBeGreaterThan(0)

      const event = new MessageEvent('heartbeat', {
        data: JSON.stringify({ timestamp: 1234567890 }),
      })
      listeners[0](event)

      expect(onHeartbeat).toHaveBeenCalledWith(1234567890)
    })

    it('handles connected event from server', () => {
      const onConnectionChange = vi.fn()

      const client = createFeatureFlagSSE({
        onConnectionChange,
      })

      client.connect()

      // Dispatch connected event
      const listeners = eventListeners.get('connected') || []
      expect(listeners.length).toBeGreaterThan(0)

      const event = new MessageEvent('connected', {
        data: JSON.stringify({ client_id: 'test-123', timestamp: 1234567890 }),
      })
      listeners[0](event)

      // Should not change state (already in connecting)
      expect(client.getState()).toBe('connecting')
    })
  })

  describe('reconnection on error', () => {
    it('transitions to disconnected on connection close', () => {
      const onConnectionChange = vi.fn()

      const client = createFeatureFlagSSE({
        onConnectionChange,
        maxReconnectAttempts: 0, // Disable reconnection for this test
      })

      client.connect()
      mockEventSourceInstance.onopen?.(new Event('open'))

      // Simulate connection error with closed state
      mockEventSourceInstance.readyState = MockEventSource.CLOSED
      mockEventSourceInstance.onerror?.(new Event('error'))

      expect(onConnectionChange).toHaveBeenCalledWith('disconnected', undefined)
    })
  })

  describe('singleton', () => {
    it('getFeatureFlagSSE returns same instance', () => {
      const client1 = getFeatureFlagSSE()
      const client2 = getFeatureFlagSSE()

      expect(client1).toBe(client2)
    })

    it('resetFeatureFlagSSE clears singleton', () => {
      const client1 = getFeatureFlagSSE()
      resetFeatureFlagSSE()
      const client2 = getFeatureFlagSSE()

      expect(client1).not.toBe(client2)
    })
  })

  describe('no EventSource support', () => {
    it('handles browsers without EventSource', () => {
      // Temporarily remove EventSource
      const originalEventSource = window.EventSource
      // @ts-expect-error - intentionally removing EventSource
      delete window.EventSource

      const onConnectionChange = vi.fn()

      const client = createFeatureFlagSSE({
        onConnectionChange,
      })

      client.connect()

      expect(onConnectionChange).toHaveBeenCalledWith('error', expect.any(Error))
      expect(client.getState()).toBe('error')

      // Restore
      window.EventSource = originalEventSource
    })
  })
})
