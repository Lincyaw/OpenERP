/**
 * FeatureFlagProvider Component Tests
 *
 * Tests for the feature flag provider component that initializes
 * feature flags on application startup.
 */

import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { render, screen, waitFor, act } from '@testing-library/react'
import { FeatureFlagProvider } from './FeatureFlagProvider'
import { useFeatureFlagStore, type FlagValue } from '@/store'

// Mock axios instance
vi.mock('@/services/axios-instance', () => ({
  axiosInstance: {
    post: vi.fn(),
  },
}))

// Mock SSE service (SSE is disabled by default in tests via preferSSE={false})
vi.mock('@/services/featureFlagSSE', () => ({
  createFeatureFlagSSE: vi.fn(() => ({
    connect: vi.fn(),
    disconnect: vi.fn(),
    getState: vi.fn(() => 'disconnected'),
    isConnected: vi.fn(() => false),
  })),
}))

import { axiosInstance } from '@/services/axios-instance'

// Mock sessionStorage
const sessionStorageMock = {
  store: {} as Record<string, string>,
  getItem: vi.fn((key: string) => sessionStorageMock.store[key] || null),
  setItem: vi.fn((key: string, value: string) => {
    sessionStorageMock.store[key] = value
  }),
  removeItem: vi.fn((key: string) => {
    delete sessionStorageMock.store[key]
  }),
  clear: vi.fn(() => {
    sessionStorageMock.store = {}
  }),
  get length() {
    return Object.keys(sessionStorageMock.store).length
  },
  key: vi.fn((index: number) => Object.keys(sessionStorageMock.store)[index] || null),
}

Object.defineProperty(window, 'sessionStorage', { value: sessionStorageMock })

// Capture console output
const originalConsoleError = console.error
const originalConsoleWarn = console.warn
let consoleErrorSpy: ReturnType<typeof vi.fn>
let consoleWarnSpy: ReturnType<typeof vi.fn>

describe('FeatureFlagProvider', () => {
  const mockFlags: Record<string, FlagValue> = {
    new_checkout_flow: { enabled: true, variant: null },
    button_color: { enabled: true, variant: 'blue' },
    disabled_feature: { enabled: false, variant: null },
  }

  const mockApiResponse = {
    data: {
      success: true,
      data: {
        flags: {
          new_checkout_flow: { enabled: true },
          button_color: { enabled: true, variant: 'blue' },
          disabled_feature: { enabled: false },
        },
        evaluated_at: '2024-01-01T00:00:00Z',
      },
    },
  }

  beforeEach(() => {
    consoleErrorSpy = vi.fn()
    consoleWarnSpy = vi.fn()
    console.error = consoleErrorSpy
    console.warn = consoleWarnSpy

    // Reset store state before each test
    const store = useFeatureFlagStore.getState()
    act(() => {
      store.stopPolling()
      useFeatureFlagStore.setState({
        flags: {},
        isLoading: false,
        isReady: false,
        lastUpdated: null,
        error: null,
      })
    })
    // Clear sessionStorage mock
    sessionStorageMock.clear()
    vi.clearAllMocks()
  })

  afterEach(() => {
    console.error = originalConsoleError
    console.warn = originalConsoleWarn
    // Stop any polling that might be running
    useFeatureFlagStore.getState().stopPolling()
    vi.resetAllMocks()
    vi.useRealTimers()
  })

  // ============================================================================
  // Initialization Tests
  // ============================================================================

  describe('initialization', () => {
    it('should initialize flags on mount', async () => {
      vi.mocked(axiosInstance.post).mockResolvedValueOnce(mockApiResponse)

      render(
        <FeatureFlagProvider>
          <div data-testid="child">Child Content</div>
        </FeatureFlagProvider>
      )

      // Should render children immediately
      expect(screen.getByTestId('child')).toBeInTheDocument()

      // Wait for initialization to complete
      await waitFor(() => {
        expect(useFeatureFlagStore.getState().isReady).toBe(true)
      })

      expect(axiosInstance.post).toHaveBeenCalledWith('/feature-flags/client-config', {
        context: {},
      })
    })

    it('should render children even when initialization fails', async () => {
      vi.mocked(axiosInstance.post).mockRejectedValueOnce(new Error('Network error'))

      render(
        <FeatureFlagProvider>
          <div data-testid="child">Child Content</div>
        </FeatureFlagProvider>
      )

      // Children should still be rendered
      expect(screen.getByTestId('child')).toBeInTheDocument()

      // Wait for initialization attempt to complete
      await waitFor(() => {
        expect(useFeatureFlagStore.getState().error).toBe('Network error')
      })
    })

    it('should handle initialization failures gracefully', async () => {
      vi.mocked(axiosInstance.post).mockRejectedValueOnce(new Error('Network error'))

      render(
        <FeatureFlagProvider>
          <div data-testid="child">Child</div>
        </FeatureFlagProvider>
      )

      await waitFor(() => {
        expect(useFeatureFlagStore.getState().error).toBe('Network error')
      })

      // Error should be logged (though we can't easily verify the exact call
      // due to React's error handling in tests)
      expect(screen.getByTestId('child')).toBeInTheDocument()
    })
  })

  // ============================================================================
  // Loading Component Tests
  // ============================================================================

  describe('loadingComponent', () => {
    it('should show loading component when isLoading is true and isReady is false', () => {
      // Set loading state manually before rendering
      act(() => {
        useFeatureFlagStore.setState({ isLoading: true, isReady: false })
      })

      // Mock the API to return a pending promise
      vi.mocked(axiosInstance.post).mockImplementation(
        () => new Promise(() => {}) // Never resolves
      )

      render(
        <FeatureFlagProvider loadingComponent={<div data-testid="loading">Loading...</div>}>
          <div data-testid="child">Child Content</div>
        </FeatureFlagProvider>
      )

      // Should show loading component
      expect(screen.getByTestId('loading')).toBeInTheDocument()
      expect(screen.queryByTestId('child')).not.toBeInTheDocument()
    })

    it('should not show loading component if not provided', () => {
      // Set loading state
      act(() => {
        useFeatureFlagStore.setState({ isLoading: true, isReady: false })
      })

      vi.mocked(axiosInstance.post).mockImplementation(() => new Promise(() => {}))

      render(
        <FeatureFlagProvider>
          <div data-testid="child">Child Content</div>
        </FeatureFlagProvider>
      )

      // Should show children immediately even during loading
      expect(screen.getByTestId('child')).toBeInTheDocument()
    })

    it('should not show loading component during refresh (when already ready)', () => {
      // Set state to ready but loading (refresh scenario)
      act(() => {
        useFeatureFlagStore.setState({ isLoading: true, isReady: true, flags: mockFlags })
      })

      vi.mocked(axiosInstance.post).mockImplementation(() => new Promise(() => {}))

      render(
        <FeatureFlagProvider loadingComponent={<div data-testid="loading">Loading...</div>}>
          <div data-testid="child">Child Content</div>
        </FeatureFlagProvider>
      )

      // Should show children, not loading (refresh doesn't show loading)
      expect(screen.queryByTestId('loading')).not.toBeInTheDocument()
      expect(screen.getByTestId('child')).toBeInTheDocument()
    })

    it('should show children after loading completes', async () => {
      vi.mocked(axiosInstance.post).mockResolvedValueOnce(mockApiResponse)

      render(
        <FeatureFlagProvider loadingComponent={<div data-testid="loading">Loading...</div>}>
          <div data-testid="child">Child Content</div>
        </FeatureFlagProvider>
      )

      // Wait for loading to complete
      await waitFor(() => {
        expect(useFeatureFlagStore.getState().isReady).toBe(true)
      })

      // Should now show children
      await waitFor(() => {
        expect(screen.getByTestId('child')).toBeInTheDocument()
      })
    })
  })

  // ============================================================================
  // Polling Tests
  // ============================================================================

  describe('polling', () => {
    it('should start polling after initialization when preferSSE is false', async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true })
      vi.mocked(axiosInstance.post).mockResolvedValue(mockApiResponse)

      render(
        <FeatureFlagProvider pollingInterval={1000} preferSSE={false}>
          <div>Child</div>
        </FeatureFlagProvider>
      )

      // Wait for initialization
      await vi.waitFor(() => {
        expect(useFeatureFlagStore.getState().isReady).toBe(true)
      })

      // Clear mocks to count only polling calls
      vi.clearAllMocks()

      // Advance timer and check polling
      await act(async () => {
        vi.advanceTimersByTime(1000)
      })

      expect(axiosInstance.post).toHaveBeenCalled()
    })

    it('should not start polling when enableRealtime is false', async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true })
      vi.mocked(axiosInstance.post).mockResolvedValue(mockApiResponse)

      render(
        <FeatureFlagProvider enableRealtime={false} pollingInterval={1000}>
          <div>Child</div>
        </FeatureFlagProvider>
      )

      // Wait for initialization
      await vi.waitFor(() => {
        expect(useFeatureFlagStore.getState().isReady).toBe(true)
      })

      // Clear mocks
      vi.clearAllMocks()

      // Advance timer
      await act(async () => {
        vi.advanceTimersByTime(5000)
      })

      // Should not have made any polling calls
      expect(axiosInstance.post).not.toHaveBeenCalled()
    })

    it('should use custom polling interval', async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true })
      vi.mocked(axiosInstance.post).mockResolvedValue(mockApiResponse)

      render(
        <FeatureFlagProvider pollingInterval={5000} preferSSE={false}>
          <div>Child</div>
        </FeatureFlagProvider>
      )

      // Wait for initialization
      await vi.waitFor(() => {
        expect(useFeatureFlagStore.getState().isReady).toBe(true)
      })

      vi.clearAllMocks()

      // Advance by 2000ms - should not trigger
      await act(async () => {
        vi.advanceTimersByTime(2000)
      })
      expect(axiosInstance.post).not.toHaveBeenCalled()

      // Advance to 5000ms - should trigger
      await act(async () => {
        vi.advanceTimersByTime(3000)
      })
      expect(axiosInstance.post).toHaveBeenCalled()
    })

    it('should stop polling on unmount', async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true })
      vi.mocked(axiosInstance.post).mockResolvedValue(mockApiResponse)

      const { unmount } = render(
        <FeatureFlagProvider pollingInterval={1000} preferSSE={false}>
          <div>Child</div>
        </FeatureFlagProvider>
      )

      // Wait for initialization
      await vi.waitFor(() => {
        expect(useFeatureFlagStore.getState().isReady).toBe(true)
      })

      // Unmount
      unmount()

      // Clear mocks
      vi.clearAllMocks()

      // Advance timer
      await act(async () => {
        vi.advanceTimersByTime(5000)
      })

      // Should not have made any calls after unmount
      expect(axiosInstance.post).not.toHaveBeenCalled()
    })

    it('should use default polling interval of 30 seconds (when SSE disabled)', async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true })
      vi.mocked(axiosInstance.post).mockResolvedValue(mockApiResponse)

      render(
        <FeatureFlagProvider preferSSE={false}>
          <div>Child</div>
        </FeatureFlagProvider>
      )

      // Wait for initialization
      await vi.waitFor(() => {
        expect(useFeatureFlagStore.getState().isReady).toBe(true)
      })

      vi.clearAllMocks()

      // Advance by 29 seconds - should not trigger
      await act(async () => {
        vi.advanceTimersByTime(29000)
      })
      expect(axiosInstance.post).not.toHaveBeenCalled()

      // Advance to 30 seconds - should trigger
      await act(async () => {
        vi.advanceTimersByTime(1000)
      })
      expect(axiosInstance.post).toHaveBeenCalled()
    })
  })

  // ============================================================================
  // Error Handling Tests
  // ============================================================================

  describe('error handling', () => {
    it('should continue to render children even with persistent errors', async () => {
      vi.mocked(axiosInstance.post).mockRejectedValue(new Error('Persistent error'))

      render(
        <FeatureFlagProvider>
          <div data-testid="child">Important Content</div>
        </FeatureFlagProvider>
      )

      // Children should be visible despite errors
      expect(screen.getByTestId('child')).toBeInTheDocument()

      await waitFor(() => {
        expect(useFeatureFlagStore.getState().error).toBe('Persistent error')
      })

      // Children should still be visible
      expect(screen.getByTestId('child')).toBeInTheDocument()
    })

    it('should warn when there is an error in the store', async () => {
      vi.mocked(axiosInstance.post).mockRejectedValueOnce(new Error('Server error'))

      render(
        <FeatureFlagProvider>
          <div>Child</div>
        </FeatureFlagProvider>
      )

      await waitFor(() => {
        expect(useFeatureFlagStore.getState().error).toBe('Server error')
      })

      // After error is set, the warning should be logged
      await waitFor(() => {
        expect(consoleWarnSpy).toHaveBeenCalledWith(
          '[FeatureFlagProvider] Feature flag error:',
          'Server error'
        )
      })
    })
  })

  // ============================================================================
  // Props Tests
  // ============================================================================

  describe('props', () => {
    it('should accept all documented props', async () => {
      vi.mocked(axiosInstance.post).mockResolvedValue(mockApiResponse)

      render(
        <FeatureFlagProvider
          pollingInterval={60000}
          enableRealtime={true}
          preferSSE={false}
          loadingComponent={<span>Loading</span>}
        >
          <div data-testid="child">Child</div>
        </FeatureFlagProvider>
      )

      // Wait for initialization
      await waitFor(() => {
        expect(useFeatureFlagStore.getState().isReady).toBe(true)
      })

      expect(screen.getByTestId('child')).toBeInTheDocument()
    })

    it('should work with minimal props', () => {
      vi.mocked(axiosInstance.post).mockResolvedValue(mockApiResponse)

      render(
        <FeatureFlagProvider>
          <div>Child</div>
        </FeatureFlagProvider>
      )

      expect(screen.getByText('Child')).toBeInTheDocument()
    })
  })

  // ============================================================================
  // Integration Tests
  // ============================================================================

  describe('integration', () => {
    it('should integrate with feature flag store correctly', async () => {
      vi.mocked(axiosInstance.post).mockResolvedValueOnce(mockApiResponse)

      render(
        <FeatureFlagProvider>
          <div>Child</div>
        </FeatureFlagProvider>
      )

      await waitFor(() => {
        const state = useFeatureFlagStore.getState()
        expect(state.isReady).toBe(true)
        expect(state.flags.new_checkout_flow).toEqual({ enabled: true, variant: null })
        expect(state.flags.button_color).toEqual({ enabled: true, variant: 'blue' })
      })
    })

    it('should work with cached flags from sessionStorage', async () => {
      // Pre-populate flags in store (simulating sessionStorage restoration)
      act(() => {
        useFeatureFlagStore.setState({ flags: mockFlags })
      })

      // API call fails
      vi.mocked(axiosInstance.post).mockRejectedValueOnce(new Error('Network error'))

      render(
        <FeatureFlagProvider>
          <div data-testid="child">Child</div>
        </FeatureFlagProvider>
      )

      expect(screen.getByTestId('child')).toBeInTheDocument()

      await waitFor(() => {
        const state = useFeatureFlagStore.getState()
        // Should still have cached flags
        expect(state.flags).toEqual(mockFlags)
        // Should be ready because we have cache fallback
        expect(state.isReady).toBe(true)
      })
    })
  })

  // ============================================================================
  // Edge Cases
  // ============================================================================

  describe('edge cases', () => {
    it('should handle zero polling interval gracefully', async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true })
      vi.mocked(axiosInstance.post).mockResolvedValue(mockApiResponse)

      render(
        <FeatureFlagProvider pollingInterval={0} preferSSE={false}>
          <div>Child</div>
        </FeatureFlagProvider>
      )

      await vi.waitFor(() => {
        expect(useFeatureFlagStore.getState().isReady).toBe(true)
      })

      vi.clearAllMocks()

      // Advance timer - should not trigger polling with interval 0
      await act(async () => {
        vi.advanceTimersByTime(100)
      })

      expect(axiosInstance.post).not.toHaveBeenCalled()
    })

    it('should handle negative polling interval gracefully', async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true })
      vi.mocked(axiosInstance.post).mockResolvedValue(mockApiResponse)

      render(
        <FeatureFlagProvider pollingInterval={-1000} preferSSE={false}>
          <div>Child</div>
        </FeatureFlagProvider>
      )

      await vi.waitFor(() => {
        expect(useFeatureFlagStore.getState().isReady).toBe(true)
      })

      vi.clearAllMocks()

      // Advance timer - should not trigger polling with negative interval
      await act(async () => {
        vi.advanceTimersByTime(5000)
      })

      expect(axiosInstance.post).not.toHaveBeenCalled()
    })

    it('should render multiple children', () => {
      vi.mocked(axiosInstance.post).mockResolvedValue(mockApiResponse)

      render(
        <FeatureFlagProvider>
          <div data-testid="child1">Child 1</div>
          <div data-testid="child2">Child 2</div>
          <span data-testid="child3">Child 3</span>
        </FeatureFlagProvider>
      )

      expect(screen.getByTestId('child1')).toBeInTheDocument()
      expect(screen.getByTestId('child2')).toBeInTheDocument()
      expect(screen.getByTestId('child3')).toBeInTheDocument()
    })

    it('should handle API response with empty flags', async () => {
      vi.mocked(axiosInstance.post).mockResolvedValueOnce({
        data: {
          success: true,
          data: {
            flags: {},
            evaluated_at: '2024-01-01T00:00:00Z',
          },
        },
      })

      render(
        <FeatureFlagProvider>
          <div data-testid="child">Child</div>
        </FeatureFlagProvider>
      )

      await waitFor(() => {
        const state = useFeatureFlagStore.getState()
        expect(state.isReady).toBe(true)
        expect(Object.keys(state.flags)).toHaveLength(0)
      })

      expect(screen.getByTestId('child')).toBeInTheDocument()
    })
  })

  // ============================================================================
  // Display Name
  // ============================================================================

  describe('displayName', () => {
    it('should have displayName for React DevTools', () => {
      expect(FeatureFlagProvider.displayName).toBe('FeatureFlagProvider')
    })
  })
})
