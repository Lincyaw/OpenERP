/**
 * usePrintServiceStatus Hook Tests
 *
 * Tests for the print service status monitoring hook functionality.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act, waitFor } from '@testing-library/react'
import { Toast } from '@douyinfe/semi-ui-19'
import {
  usePrintServiceStatus,
  DEFAULT_POLLING_INTERVAL,
  DEFAULT_BASE_URL,
  type PrintServiceHealthResponse,
  type PrintersResponse,
} from './usePrintServiceStatus'

// Mock Toast
vi.mock('@douyinfe/semi-ui-19', () => ({
  Toast: {
    success: vi.fn(),
    warning: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
  },
}))

// Mock fetch globally
const mockFetch = vi.fn()
global.fetch = mockFetch

describe('usePrintServiceStatus', () => {
  const mockHealthResponse: PrintServiceHealthResponse = {
    status: 'ok',
    version: '1.2.3',
  }

  const mockPrintersResponse: PrintersResponse = {
    printers: [
      { name: 'printer-1', displayName: 'HP LaserJet', isDefault: true, status: 'online' },
      { name: 'printer-2', displayName: 'Epson Thermal', isDefault: false, status: 'online' },
    ],
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  // Helper to create successful fetch responses
  function mockSuccessfulFetch() {
    mockFetch.mockImplementation(async (url: string) => {
      if (url.includes('/health')) {
        return {
          ok: true,
          json: async () => mockHealthResponse,
        }
      }
      if (url.includes('/printers')) {
        return {
          ok: true,
          json: async () => mockPrintersResponse,
        }
      }
      return { ok: false, status: 404 }
    })
  }

  // Helper to mock failed fetch (service unavailable)
  function mockFailedFetch() {
    mockFetch.mockRejectedValue(new TypeError('Failed to fetch'))
  }

  // ============================================================================
  // Initialization tests
  // ============================================================================

  describe('initialization', () => {
    it('should initialize with checking status', () => {
      mockSuccessfulFetch()
      const { result } = renderHook(() => usePrintServiceStatus({ enabled: false }))

      expect(result.current.status).toBe('checking')
      expect(result.current.version).toBeNull()
      expect(result.current.printers).toEqual([])
      expect(result.current.error).toBeNull()
    })

    it('should export default constants', () => {
      expect(DEFAULT_POLLING_INTERVAL).toBe(10000)
      expect(DEFAULT_BASE_URL).toBe('http://localhost:9999')
    })

    it('should not start polling when disabled', async () => {
      mockSuccessfulFetch()
      renderHook(() => usePrintServiceStatus({ enabled: false }))

      // Wait a bit to ensure no polling happens
      await new Promise((resolve) => setTimeout(resolve, 100))

      expect(mockFetch).not.toHaveBeenCalled()
    })
  })

  // ============================================================================
  // Health check tests
  // ============================================================================

  describe('health check', () => {
    it('should transition to connected when health check succeeds', async () => {
      mockSuccessfulFetch()
      const { result } = renderHook(() => usePrintServiceStatus({ enabled: false }))

      await act(async () => {
        await result.current.checkHealth()
      })

      expect(result.current.status).toBe('connected')
      expect(result.current.version).toBe('1.2.3')
      expect(result.current.error).toBeNull()
      expect(result.current.lastChecked).toBeInstanceOf(Date)
    })

    it('should transition to disconnected when health check fails', async () => {
      mockFailedFetch()
      const { result } = renderHook(() => usePrintServiceStatus({ enabled: false }))

      await act(async () => {
        await result.current.checkHealth()
      })

      expect(result.current.status).toBe('disconnected')
      expect(result.current.version).toBeNull()
      expect(result.current.error).toBeTruthy()
    })

    it('should call health endpoint with correct URL', async () => {
      mockSuccessfulFetch()
      const { result } = renderHook(() =>
        usePrintServiceStatus({ enabled: false, baseUrl: 'http://custom:8888' })
      )

      await act(async () => {
        await result.current.checkHealth()
      })

      expect(mockFetch).toHaveBeenCalledWith(
        'http://custom:8888/health',
        expect.objectContaining({
          method: 'GET',
          headers: { Accept: 'application/json' },
        })
      )
    })

    it('should handle HTTP error responses', async () => {
      mockFetch.mockResolvedValue({ ok: false, status: 500 })
      const { result } = renderHook(() => usePrintServiceStatus({ enabled: false }))

      await act(async () => {
        await result.current.checkHealth()
      })

      expect(result.current.status).toBe('disconnected')
    })

    it('should handle service returning error status', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({
          status: 'error',
          message: 'Service initializing',
        }),
      })

      const { result } = renderHook(() => usePrintServiceStatus({ enabled: false }))

      await act(async () => {
        await result.current.checkHealth()
      })

      expect(result.current.status).toBe('disconnected')
    })
  })

  // ============================================================================
  // Polling tests (with real timers)
  // ============================================================================

  describe('polling', () => {
    it('should perform initial health check when enabled', async () => {
      mockSuccessfulFetch()
      const { result } = renderHook(() => usePrintServiceStatus({ pollingInterval: 60000 }))

      // Wait for initial check
      await waitFor(() => {
        expect(result.current.status).toBe('connected')
      })

      expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining('/health'), expect.any(Object))
    })

    it('should stop polling on unmount', async () => {
      mockSuccessfulFetch()
      const { result, unmount } = renderHook(() =>
        usePrintServiceStatus({ pollingInterval: 60000 })
      )

      // Wait for initial check
      await waitFor(() => {
        expect(result.current.status).toBe('connected')
      })

      const callCountBeforeUnmount = mockFetch.mock.calls.filter((call) =>
        call[0].includes('/health')
      ).length

      unmount()

      // Wait a bit after unmount
      await new Promise((resolve) => setTimeout(resolve, 100))

      const callCountAfterUnmount = mockFetch.mock.calls.filter((call) =>
        call[0].includes('/health')
      ).length

      expect(callCountAfterUnmount).toBe(callCountBeforeUnmount)
    })
  })

  // ============================================================================
  // Printer list tests
  // ============================================================================

  describe('printer list', () => {
    it('should fetch printers when connected', async () => {
      mockSuccessfulFetch()
      const { result } = renderHook(() => usePrintServiceStatus({ enabled: false }))

      await act(async () => {
        await result.current.refresh()
      })

      expect(result.current.printers).toHaveLength(2)
      expect(result.current.printers[0].name).toBe('printer-1')
      expect(result.current.printers[0].isDefault).toBe(true)
    })

    it('should not fetch printers when disconnected (manual refresh)', async () => {
      mockFailedFetch()
      const { result } = renderHook(() => usePrintServiceStatus({ enabled: false }))

      await act(async () => {
        await result.current.refresh()
      })

      // Printers endpoint should NOT be called after failed health check
      const printerCalls = mockFetch.mock.calls.filter((call) => call[0].includes('/printers'))
      expect(printerCalls).toHaveLength(0)
      expect(result.current.printers).toEqual([])
    })

    it('should call printers endpoint with correct URL', async () => {
      mockSuccessfulFetch()
      const { result } = renderHook(() =>
        usePrintServiceStatus({ enabled: false, baseUrl: 'http://custom:8888' })
      )

      await act(async () => {
        await result.current.refreshPrinters()
      })

      expect(mockFetch).toHaveBeenCalledWith(
        'http://custom:8888/printers',
        expect.objectContaining({
          method: 'GET',
          headers: { Accept: 'application/json' },
        })
      )
    })

    it('should handle empty printer list', async () => {
      mockFetch.mockImplementation(async (url: string) => {
        if (url.includes('/health')) {
          return { ok: true, json: async () => mockHealthResponse }
        }
        if (url.includes('/printers')) {
          return { ok: true, json: async () => ({ printers: [] }) }
        }
        return { ok: false }
      })

      const { result } = renderHook(() => usePrintServiceStatus({ enabled: false }))

      await act(async () => {
        await result.current.refresh()
      })

      expect(result.current.status).toBe('connected')
      expect(result.current.printers).toEqual([])
    })
  })

  // ============================================================================
  // Error handling tests
  // ============================================================================

  describe('error handling', () => {
    it('should handle CORS errors', async () => {
      mockFetch.mockRejectedValue(new TypeError('Failed to fetch'))
      const { result } = renderHook(() => usePrintServiceStatus({ enabled: false }))

      await act(async () => {
        await result.current.checkHealth()
      })

      expect(result.current.error).toContain('跨域')
    })

    it('should handle timeout errors', async () => {
      mockFetch.mockRejectedValue(new DOMException('Aborted', 'AbortError'))
      const { result } = renderHook(() => usePrintServiceStatus({ enabled: false }))

      await act(async () => {
        await result.current.checkHealth()
      })

      expect(result.current.error).toContain('超时')
    })

    it('should handle network errors', async () => {
      mockFetch.mockRejectedValue(new Error('Network error: connection refused'))
      const { result } = renderHook(() => usePrintServiceStatus({ enabled: false }))

      await act(async () => {
        await result.current.checkHealth()
      })

      expect(result.current.error).toContain('网络')
    })

    it('should clear error on successful reconnection', async () => {
      // First fail
      mockFailedFetch()
      const { result } = renderHook(() => usePrintServiceStatus({ enabled: false }))

      await act(async () => {
        await result.current.checkHealth()
      })

      expect(result.current.error).toBeTruthy()

      // Then succeed
      mockSuccessfulFetch()

      await act(async () => {
        await result.current.checkHealth()
      })

      expect(result.current.error).toBeNull()
    })
  })

  // ============================================================================
  // Toast notification tests
  // ============================================================================

  describe('notifications', () => {
    it('should show success toast when reconnected', async () => {
      // Start disconnected
      mockFailedFetch()
      const { result } = renderHook(() => usePrintServiceStatus({ enabled: false }))

      await act(async () => {
        await result.current.checkHealth()
      })

      expect(result.current.status).toBe('disconnected')

      // Now reconnect
      mockSuccessfulFetch()

      await act(async () => {
        await result.current.checkHealth()
      })

      expect(result.current.status).toBe('connected')
      expect(Toast.success).toHaveBeenCalledWith('打印服务已连接')
    })

    it('should show warning toast when disconnected', async () => {
      // Start connected
      mockSuccessfulFetch()
      const { result } = renderHook(() => usePrintServiceStatus({ enabled: false }))

      await act(async () => {
        await result.current.checkHealth()
      })

      expect(result.current.status).toBe('connected')

      // Now disconnect
      mockFailedFetch()

      await act(async () => {
        await result.current.checkHealth()
      })

      expect(result.current.status).toBe('disconnected')
      expect(Toast.warning).toHaveBeenCalledWith('打印服务连接已断开')
    })

    it('should not show toast on first check', async () => {
      mockFailedFetch()
      const { result } = renderHook(() => usePrintServiceStatus({ enabled: false }))

      await act(async () => {
        await result.current.checkHealth()
      })

      // Should NOT show toast on initial disconnected state
      expect(Toast.warning).not.toHaveBeenCalled()
      expect(Toast.success).not.toHaveBeenCalled()
    })

    it('should not show notifications when disabled', async () => {
      // Start disconnected
      mockFailedFetch()
      const { result } = renderHook(() =>
        usePrintServiceStatus({ enabled: false, showNotifications: false })
      )

      await act(async () => {
        await result.current.checkHealth()
      })

      // Reconnect
      mockSuccessfulFetch()

      await act(async () => {
        await result.current.checkHealth()
      })

      expect(result.current.status).toBe('connected')
      expect(Toast.success).not.toHaveBeenCalled()
      expect(Toast.warning).not.toHaveBeenCalled()
    })
  })

  // ============================================================================
  // Manual refresh tests
  // ============================================================================

  describe('manual refresh', () => {
    it('should allow manual health check', async () => {
      mockSuccessfulFetch()
      const { result } = renderHook(() => usePrintServiceStatus({ enabled: false }))

      await act(async () => {
        await result.current.checkHealth()
      })

      expect(result.current.status).toBe('connected')
    })

    it('should allow manual printer refresh', async () => {
      mockSuccessfulFetch()
      const { result } = renderHook(() => usePrintServiceStatus({ enabled: false }))

      await act(async () => {
        await result.current.refreshPrinters()
      })

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/printers'),
        expect.any(Object)
      )
    })

    it('should allow combined refresh', async () => {
      mockSuccessfulFetch()
      const { result } = renderHook(() => usePrintServiceStatus({ enabled: false }))

      await act(async () => {
        await result.current.refresh()
      })

      expect(result.current.status).toBe('connected')
      expect(result.current.printers).toHaveLength(2)
    })
  })

  // ============================================================================
  // Version info tests
  // ============================================================================

  describe('version info', () => {
    it('should expose service version when connected', async () => {
      mockSuccessfulFetch()
      const { result } = renderHook(() => usePrintServiceStatus({ enabled: false }))

      await act(async () => {
        await result.current.checkHealth()
      })

      expect(result.current.version).toBe('1.2.3')
    })

    it('should clear version when disconnected', async () => {
      mockSuccessfulFetch()
      const { result } = renderHook(() => usePrintServiceStatus({ enabled: false }))

      await act(async () => {
        await result.current.checkHealth()
      })

      expect(result.current.version).toBe('1.2.3')

      // Disconnect
      mockFailedFetch()

      await act(async () => {
        await result.current.checkHealth()
      })

      expect(result.current.version).toBeNull()
    })

    it('should handle missing version in response', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ status: 'ok' }),
      })

      const { result } = renderHook(() => usePrintServiceStatus({ enabled: false }))

      await act(async () => {
        await result.current.checkHealth()
      })

      expect(result.current.status).toBe('connected')
      expect(result.current.version).toBeNull()
    })
  })
})
