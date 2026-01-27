/**
 * Feature Flag Hooks Tests
 *
 * Tests for useFeatureFlag.ts React hooks
 */

import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useFeatureFlagStore, type FlagValue } from '@/store'
import {
  useFeatureFlag,
  useFeatureVariant,
  useFeatureFlags,
  useFeatureFlagReady,
  useFeatureFlagValue,
  useFeatureFlagLoading,
  useFeatureFlagError,
} from './useFeatureFlag'

// Mock axios instance
vi.mock('@/services/axios-instance', () => ({
  axiosInstance: {
    post: vi.fn(),
  },
}))

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

describe('useFeatureFlag hooks', () => {
  const mockFlags: Record<string, FlagValue> = {
    new_checkout_flow: { enabled: true, variant: null },
    button_color: { enabled: true, variant: 'blue' },
    disabled_feature: { enabled: false, variant: null },
    complex_feature: { enabled: true, variant: 'v2', metadata: { maxItems: 10 } },
  }

  beforeEach(() => {
    // Reset store state before each test
    const store = useFeatureFlagStore.getState()
    act(() => {
      store.stopPolling()
      useFeatureFlagStore.setState({
        flags: mockFlags,
        isLoading: false,
        isReady: true,
        lastUpdated: new Date(),
        error: null,
      })
    })
    // Clear sessionStorage mock
    sessionStorageMock.clear()
    vi.clearAllMocks()
  })

  afterEach(() => {
    useFeatureFlagStore.getState().stopPolling()
    vi.resetAllMocks()
  })

  // ============================================================================
  // useFeatureFlag tests
  // ============================================================================

  describe('useFeatureFlag', () => {
    it('should return true for enabled flags', () => {
      const { result } = renderHook(() => useFeatureFlag('new_checkout_flow'))
      expect(result.current).toBe(true)
    })

    it('should return false for disabled flags', () => {
      const { result } = renderHook(() => useFeatureFlag('disabled_feature'))
      expect(result.current).toBe(false)
    })

    it('should return false for non-existent flags', () => {
      const { result } = renderHook(() => useFeatureFlag('unknown_flag'))
      expect(result.current).toBe(false)
    })

    it('should return defaultValue for non-existent flags when provided', () => {
      const { result } = renderHook(() => useFeatureFlag('unknown_flag', true))
      expect(result.current).toBe(true)
    })

    it('should not use defaultValue when flag exists', () => {
      const { result } = renderHook(() => useFeatureFlag('disabled_feature', true))
      expect(result.current).toBe(false) // Flag is explicitly disabled
    })

    it('should update when flag state changes', () => {
      const { result, rerender } = renderHook(() => useFeatureFlag('new_checkout_flow'))

      expect(result.current).toBe(true)

      act(() => {
        useFeatureFlagStore.setState({
          ...useFeatureFlagStore.getState(),
          flags: {
            ...mockFlags,
            new_checkout_flow: { enabled: false, variant: null },
          },
        })
      })

      rerender()
      expect(result.current).toBe(false)
    })
  })

  // ============================================================================
  // useFeatureVariant tests
  // ============================================================================

  describe('useFeatureVariant', () => {
    it('should return variant value for flags with variants', () => {
      const { result } = renderHook(() => useFeatureVariant('button_color'))
      expect(result.current).toBe('blue')
    })

    it('should return null for flags without variants', () => {
      const { result } = renderHook(() => useFeatureVariant('new_checkout_flow'))
      expect(result.current).toBeNull()
    })

    it('should return null for non-existent flags', () => {
      const { result } = renderHook(() => useFeatureVariant('unknown_flag'))
      expect(result.current).toBeNull()
    })

    it('should update when variant changes', () => {
      const { result, rerender } = renderHook(() => useFeatureVariant('button_color'))

      expect(result.current).toBe('blue')

      act(() => {
        useFeatureFlagStore.setState({
          ...useFeatureFlagStore.getState(),
          flags: {
            ...mockFlags,
            button_color: { enabled: true, variant: 'green' },
          },
        })
      })

      rerender()
      expect(result.current).toBe('green')
    })
  })

  // ============================================================================
  // useFeatureFlags (batch) tests
  // ============================================================================

  describe('useFeatureFlags', () => {
    it('should return record of flag enabled states', () => {
      const { result } = renderHook(() =>
        useFeatureFlags(['new_checkout_flow', 'disabled_feature'])
      )

      expect(result.current).toEqual({
        new_checkout_flow: true,
        disabled_feature: false,
      })
    })

    it('should return false for non-existent flags', () => {
      const { result } = renderHook(() => useFeatureFlags(['new_checkout_flow', 'unknown_flag']))

      expect(result.current).toEqual({
        new_checkout_flow: true,
        unknown_flag: false,
      })
    })

    it('should be type-safe with generic parameter', () => {
      const keys = ['feature_a', 'feature_b'] as const
      const { result } = renderHook(() => useFeatureFlags([...keys]))

      // TypeScript ensures these keys exist
      expect(result.current.feature_a).toBe(false)
      expect(result.current.feature_b).toBe(false)
    })

    it('should handle empty array', () => {
      const { result } = renderHook(() => useFeatureFlags([]))
      expect(result.current).toEqual({})
    })

    it('should update when any flag changes', () => {
      const { result, rerender } = renderHook(() =>
        useFeatureFlags(['new_checkout_flow', 'button_color'])
      )

      expect(result.current.new_checkout_flow).toBe(true)
      expect(result.current.button_color).toBe(true)

      act(() => {
        useFeatureFlagStore.setState({
          ...useFeatureFlagStore.getState(),
          flags: {
            ...mockFlags,
            new_checkout_flow: { enabled: false, variant: null },
          },
        })
      })

      rerender()
      expect(result.current.new_checkout_flow).toBe(false)
      expect(result.current.button_color).toBe(true)
    })

    it('should update when keys array changes', () => {
      const { result, rerender } = renderHook(({ keys }) => useFeatureFlags(keys), {
        initialProps: { keys: ['new_checkout_flow'] },
      })

      expect(result.current).toEqual({ new_checkout_flow: true })

      rerender({ keys: ['new_checkout_flow', 'disabled_feature'] })

      expect(result.current).toEqual({
        new_checkout_flow: true,
        disabled_feature: false,
      })
    })
  })

  // ============================================================================
  // useFeatureFlagReady tests
  // ============================================================================

  describe('useFeatureFlagReady', () => {
    it('should return true when flags are ready', () => {
      act(() => {
        useFeatureFlagStore.setState({
          ...useFeatureFlagStore.getState(),
          isReady: true,
        })
      })

      const { result } = renderHook(() => useFeatureFlagReady())
      expect(result.current).toBe(true)
    })

    it('should return false when flags are not ready', () => {
      act(() => {
        useFeatureFlagStore.setState({
          ...useFeatureFlagStore.getState(),
          isReady: false,
        })
      })

      const { result } = renderHook(() => useFeatureFlagReady())
      expect(result.current).toBe(false)
    })

    it('should update when ready state changes', () => {
      act(() => {
        useFeatureFlagStore.setState({
          ...useFeatureFlagStore.getState(),
          isReady: false,
        })
      })

      const { result, rerender } = renderHook(() => useFeatureFlagReady())
      expect(result.current).toBe(false)

      act(() => {
        useFeatureFlagStore.setState({
          ...useFeatureFlagStore.getState(),
          isReady: true,
        })
      })

      rerender()
      expect(result.current).toBe(true)
    })
  })

  // ============================================================================
  // useFeatureFlagValue tests
  // ============================================================================

  describe('useFeatureFlagValue', () => {
    it('should return full flag value with all properties', () => {
      const { result } = renderHook(() => useFeatureFlagValue('complex_feature'))

      expect(result.current).toEqual({
        enabled: true,
        variant: 'v2',
        metadata: { maxItems: 10 },
      })
    })

    it('should return flag value without metadata', () => {
      const { result } = renderHook(() => useFeatureFlagValue('button_color'))

      expect(result.current).toEqual({
        enabled: true,
        variant: 'blue',
      })
    })

    it('should return null for non-existent flags', () => {
      const { result } = renderHook(() => useFeatureFlagValue('unknown_flag'))
      expect(result.current).toBeNull()
    })

    it('should update when flag value changes', () => {
      const { result, rerender } = renderHook(() => useFeatureFlagValue('button_color'))

      expect(result.current?.variant).toBe('blue')

      act(() => {
        useFeatureFlagStore.setState({
          ...useFeatureFlagStore.getState(),
          flags: {
            ...mockFlags,
            button_color: { enabled: true, variant: 'red' },
          },
        })
      })

      rerender()
      expect(result.current?.variant).toBe('red')
    })
  })

  // ============================================================================
  // useFeatureFlagLoading tests
  // ============================================================================

  describe('useFeatureFlagLoading', () => {
    it('should return true when loading', () => {
      act(() => {
        useFeatureFlagStore.setState({
          ...useFeatureFlagStore.getState(),
          isLoading: true,
        })
      })

      const { result } = renderHook(() => useFeatureFlagLoading())
      expect(result.current).toBe(true)
    })

    it('should return false when not loading', () => {
      act(() => {
        useFeatureFlagStore.setState({
          ...useFeatureFlagStore.getState(),
          isLoading: false,
        })
      })

      const { result } = renderHook(() => useFeatureFlagLoading())
      expect(result.current).toBe(false)
    })

    it('should update when loading state changes', () => {
      act(() => {
        useFeatureFlagStore.setState({
          ...useFeatureFlagStore.getState(),
          isLoading: true,
        })
      })

      const { result, rerender } = renderHook(() => useFeatureFlagLoading())
      expect(result.current).toBe(true)

      act(() => {
        useFeatureFlagStore.setState({
          ...useFeatureFlagStore.getState(),
          isLoading: false,
        })
      })

      rerender()
      expect(result.current).toBe(false)
    })
  })

  // ============================================================================
  // useFeatureFlagError tests
  // ============================================================================

  describe('useFeatureFlagError', () => {
    it('should return null when no error', () => {
      act(() => {
        useFeatureFlagStore.setState({
          ...useFeatureFlagStore.getState(),
          error: null,
        })
      })

      const { result } = renderHook(() => useFeatureFlagError())
      expect(result.current).toBeNull()
    })

    it('should return error message when error exists', () => {
      act(() => {
        useFeatureFlagStore.setState({
          ...useFeatureFlagStore.getState(),
          error: 'Network error',
        })
      })

      const { result } = renderHook(() => useFeatureFlagError())
      expect(result.current).toBe('Network error')
    })

    it('should update when error state changes', () => {
      act(() => {
        useFeatureFlagStore.setState({
          ...useFeatureFlagStore.getState(),
          error: 'Initial error',
        })
      })

      const { result, rerender } = renderHook(() => useFeatureFlagError())
      expect(result.current).toBe('Initial error')

      act(() => {
        useFeatureFlagStore.getState().clearError()
      })

      rerender()
      expect(result.current).toBeNull()
    })
  })

  // ============================================================================
  // Integration tests
  // ============================================================================

  describe('hook integration', () => {
    it('should all hooks respond to same store update', () => {
      const { result: enabledResult, rerender: rerenderEnabled } = renderHook(() =>
        useFeatureFlag('test_flag')
      )
      const { result: variantResult, rerender: rerenderVariant } = renderHook(() =>
        useFeatureVariant('test_flag')
      )
      const { result: valueResult, rerender: rerenderValue } = renderHook(() =>
        useFeatureFlagValue('test_flag')
      )

      // Initially not present
      expect(enabledResult.current).toBe(false)
      expect(variantResult.current).toBeNull()
      expect(valueResult.current).toBeNull()

      // Add the flag
      act(() => {
        useFeatureFlagStore.setState({
          ...useFeatureFlagStore.getState(),
          flags: {
            ...mockFlags,
            test_flag: { enabled: true, variant: 'test_variant' },
          },
        })
      })

      rerenderEnabled()
      rerenderVariant()
      rerenderValue()

      expect(enabledResult.current).toBe(true)
      expect(variantResult.current).toBe('test_variant')
      expect(valueResult.current).toEqual({ enabled: true, variant: 'test_variant' })
    })

    it('should handle store reset gracefully', () => {
      const { result, rerender } = renderHook(() => useFeatureFlag('new_checkout_flow'))

      expect(result.current).toBe(true)

      // Simulate store reset (like logout)
      act(() => {
        useFeatureFlagStore.setState({
          flags: {},
          isLoading: false,
          isReady: false,
          lastUpdated: null,
          error: null,
        })
      })

      rerender()
      expect(result.current).toBe(false) // Falls back to default
    })
  })
})
