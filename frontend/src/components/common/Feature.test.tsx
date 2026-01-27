/**
 * Feature Component Tests
 *
 * Tests for conditional rendering based on feature flags.
 */

import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { act } from 'react'
import { useFeatureFlagStore, type FlagValue } from '@/store'
import { Feature } from './Feature'

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

describe('Feature component', () => {
  const mockFlags: Record<string, FlagValue> = {
    enable_new_checkout: { enabled: true, variant: null },
    disable_feature: { enabled: false, variant: null },
    checkout_variant: { enabled: true, variant: 'B' },
    complex_feature: { enabled: true, variant: 'premium', metadata: { tier: 2 } },
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
  // Basic Rendering Tests
  // ============================================================================

  describe('basic rendering', () => {
    it('should render children when flag is enabled', () => {
      render(
        <Feature flag="enable_new_checkout">
          <div data-testid="new-checkout">New Checkout</div>
        </Feature>
      )

      expect(screen.getByTestId('new-checkout')).toBeInTheDocument()
    })

    it('should not render children when flag is disabled', () => {
      render(
        <Feature flag="disable_feature">
          <div data-testid="disabled-content">Disabled Content</div>
        </Feature>
      )

      expect(screen.queryByTestId('disabled-content')).not.toBeInTheDocument()
    })

    it('should not render children when flag does not exist', () => {
      render(
        <Feature flag="unknown_flag">
          <div data-testid="unknown-content">Unknown Content</div>
        </Feature>
      )

      expect(screen.queryByTestId('unknown-content')).not.toBeInTheDocument()
    })

    it('should render multiple children when flag is enabled', () => {
      render(
        <Feature flag="enable_new_checkout">
          <div data-testid="child-1">Child 1</div>
          <div data-testid="child-2">Child 2</div>
        </Feature>
      )

      expect(screen.getByTestId('child-1')).toBeInTheDocument()
      expect(screen.getByTestId('child-2')).toBeInTheDocument()
    })
  })

  // ============================================================================
  // Fallback Tests
  // ============================================================================

  describe('fallback rendering', () => {
    it('should render fallback when flag is disabled', () => {
      render(
        <Feature
          flag="disable_feature"
          fallback={<div data-testid="fallback">Fallback Content</div>}
        >
          <div data-testid="main-content">Main Content</div>
        </Feature>
      )

      expect(screen.queryByTestId('main-content')).not.toBeInTheDocument()
      expect(screen.getByTestId('fallback')).toBeInTheDocument()
    })

    it('should render fallback when flag does not exist', () => {
      render(
        <Feature flag="unknown_flag" fallback={<div data-testid="fallback">Fallback Content</div>}>
          <div data-testid="main-content">Main Content</div>
        </Feature>
      )

      expect(screen.queryByTestId('main-content')).not.toBeInTheDocument()
      expect(screen.getByTestId('fallback')).toBeInTheDocument()
    })

    it('should NOT render fallback when flag is enabled', () => {
      render(
        <Feature
          flag="enable_new_checkout"
          fallback={<div data-testid="fallback">Fallback Content</div>}
        >
          <div data-testid="main-content">Main Content</div>
        </Feature>
      )

      expect(screen.getByTestId('main-content')).toBeInTheDocument()
      expect(screen.queryByTestId('fallback')).not.toBeInTheDocument()
    })

    it('should render null when flag is disabled and no fallback provided', () => {
      const { container } = render(
        <Feature flag="disable_feature">
          <div data-testid="main-content">Main Content</div>
        </Feature>
      )

      expect(container.firstChild).toBeNull()
    })
  })

  // ============================================================================
  // Variant Rendering Tests
  // ============================================================================

  describe('variant rendering', () => {
    it('should pass variant to render function', () => {
      render(
        <Feature flag="checkout_variant">
          {(variant) => <div data-testid="variant-content">Variant: {variant}</div>}
        </Feature>
      )

      expect(screen.getByTestId('variant-content')).toHaveTextContent('Variant: B')
    })

    it('should pass null variant when flag has no variant', () => {
      render(
        <Feature flag="enable_new_checkout">
          {(variant) => (
            <div data-testid="variant-content">
              Variant is null: {variant === null ? 'yes' : 'no'}
            </div>
          )}
        </Feature>
      )

      expect(screen.getByTestId('variant-content')).toHaveTextContent('Variant is null: yes')
    })

    it('should support switch statement for variants', () => {
      render(
        <Feature flag="checkout_variant">
          {(variant) => {
            switch (variant) {
              case 'A':
                return <div data-testid="variant-a">Checkout A</div>
              case 'B':
                return <div data-testid="variant-b">Checkout B</div>
              default:
                return <div data-testid="variant-default">Checkout Default</div>
            }
          }}
        </Feature>
      )

      expect(screen.getByTestId('variant-b')).toBeInTheDocument()
      expect(screen.queryByTestId('variant-a')).not.toBeInTheDocument()
      expect(screen.queryByTestId('variant-default')).not.toBeInTheDocument()
    })

    it('should not call render function when flag is disabled', () => {
      const renderFn = vi.fn((variant) => <div>Variant: {variant}</div>)

      render(<Feature flag="disable_feature">{renderFn}</Feature>)

      expect(renderFn).not.toHaveBeenCalled()
    })

    it('should render fallback instead of calling render function when flag is disabled', () => {
      const renderFn = vi.fn((variant) => <div data-testid="render-fn">Variant: {variant}</div>)

      render(
        <Feature flag="disable_feature" fallback={<div data-testid="fallback">Fallback</div>}>
          {renderFn}
        </Feature>
      )

      expect(renderFn).not.toHaveBeenCalled()
      expect(screen.getByTestId('fallback')).toBeInTheDocument()
    })
  })

  // ============================================================================
  // Loading State Tests
  // ============================================================================

  describe('loading state', () => {
    it('should render null when flags are loading and no loading prop', () => {
      act(() => {
        useFeatureFlagStore.setState({
          ...useFeatureFlagStore.getState(),
          isLoading: true,
          isReady: false,
        })
      })

      const { container } = render(
        <Feature flag="enable_new_checkout">
          <div data-testid="content">Content</div>
        </Feature>
      )

      expect(container.firstChild).toBeNull()
      expect(screen.queryByTestId('content')).not.toBeInTheDocument()
    })

    it('should render loading component when flags are loading and loading prop is provided', () => {
      act(() => {
        useFeatureFlagStore.setState({
          ...useFeatureFlagStore.getState(),
          isLoading: true,
          isReady: false,
        })
      })

      render(
        <Feature flag="enable_new_checkout" loading={<div data-testid="loading">Loading...</div>}>
          <div data-testid="content">Content</div>
        </Feature>
      )

      expect(screen.getByTestId('loading')).toBeInTheDocument()
      expect(screen.queryByTestId('content')).not.toBeInTheDocument()
    })

    it('should render content after flags are loaded', () => {
      act(() => {
        useFeatureFlagStore.setState({
          ...useFeatureFlagStore.getState(),
          isLoading: true,
          isReady: false,
        })
      })

      const { rerender } = render(
        <Feature flag="enable_new_checkout" loading={<div data-testid="loading">Loading...</div>}>
          <div data-testid="content">Content</div>
        </Feature>
      )

      expect(screen.getByTestId('loading')).toBeInTheDocument()

      // Simulate flags loaded
      act(() => {
        useFeatureFlagStore.setState({
          flags: mockFlags,
          isLoading: false,
          isReady: true,
          lastUpdated: new Date(),
          error: null,
        })
      })

      rerender(
        <Feature flag="enable_new_checkout" loading={<div data-testid="loading">Loading...</div>}>
          <div data-testid="content">Content</div>
        </Feature>
      )

      expect(screen.queryByTestId('loading')).not.toBeInTheDocument()
      expect(screen.getByTestId('content')).toBeInTheDocument()
    })

    it('should NOT show loading when flags are ready even if isLoading is true (refresh case)', () => {
      act(() => {
        useFeatureFlagStore.setState({
          flags: mockFlags,
          isLoading: true, // Refreshing
          isReady: true, // But already have data
          lastUpdated: new Date(),
          error: null,
        })
      })

      render(
        <Feature flag="enable_new_checkout" loading={<div data-testid="loading">Loading...</div>}>
          <div data-testid="content">Content</div>
        </Feature>
      )

      expect(screen.queryByTestId('loading')).not.toBeInTheDocument()
      expect(screen.getByTestId('content')).toBeInTheDocument()
    })
  })

  // ============================================================================
  // Edge Cases
  // ============================================================================

  describe('edge cases', () => {
    it('should handle flag becoming enabled', () => {
      act(() => {
        useFeatureFlagStore.setState({
          flags: { ...mockFlags, dynamic_flag: { enabled: false, variant: null } },
          isLoading: false,
          isReady: true,
          lastUpdated: new Date(),
          error: null,
        })
      })

      const { rerender } = render(
        <Feature flag="dynamic_flag">
          <div data-testid="content">Content</div>
        </Feature>
      )

      expect(screen.queryByTestId('content')).not.toBeInTheDocument()

      // Flag becomes enabled
      act(() => {
        useFeatureFlagStore.setState({
          flags: { ...mockFlags, dynamic_flag: { enabled: true, variant: null } },
          isLoading: false,
          isReady: true,
          lastUpdated: new Date(),
          error: null,
        })
      })

      rerender(
        <Feature flag="dynamic_flag">
          <div data-testid="content">Content</div>
        </Feature>
      )

      expect(screen.getByTestId('content')).toBeInTheDocument()
    })

    it('should handle flag becoming disabled', () => {
      const { rerender } = render(
        <Feature flag="enable_new_checkout">
          <div data-testid="content">Content</div>
        </Feature>
      )

      expect(screen.getByTestId('content')).toBeInTheDocument()

      // Flag becomes disabled
      act(() => {
        useFeatureFlagStore.setState({
          flags: { ...mockFlags, enable_new_checkout: { enabled: false, variant: null } },
          isLoading: false,
          isReady: true,
          lastUpdated: new Date(),
          error: null,
        })
      })

      rerender(
        <Feature flag="enable_new_checkout">
          <div data-testid="content">Content</div>
        </Feature>
      )

      expect(screen.queryByTestId('content')).not.toBeInTheDocument()
    })

    it('should handle variant change', () => {
      const { rerender } = render(
        <Feature flag="checkout_variant">
          {(variant) => <div data-testid="content">Variant: {variant}</div>}
        </Feature>
      )

      expect(screen.getByTestId('content')).toHaveTextContent('Variant: B')

      // Variant changes
      act(() => {
        useFeatureFlagStore.setState({
          flags: { ...mockFlags, checkout_variant: { enabled: true, variant: 'C' } },
          isLoading: false,
          isReady: true,
          lastUpdated: new Date(),
          error: null,
        })
      })

      rerender(
        <Feature flag="checkout_variant">
          {(variant) => <div data-testid="content">Variant: {variant}</div>}
        </Feature>
      )

      expect(screen.getByTestId('content')).toHaveTextContent('Variant: C')
    })

    it('should handle null children gracefully', () => {
      const { container } = render(<Feature flag="enable_new_checkout">{null}</Feature>)

      // Should render null without errors
      expect(container.firstChild).toBeNull()
    })

    it('should handle undefined children gracefully', () => {
      const { container } = render(<Feature flag="enable_new_checkout">{undefined}</Feature>)

      // Should render null without errors
      expect(container.firstChild).toBeNull()
    })
  })

  // ============================================================================
  // Accessibility Tests
  // ============================================================================

  describe('accessibility', () => {
    it('should not affect children accessibility', () => {
      render(
        <Feature flag="enable_new_checkout">
          <button aria-label="Submit order">Submit</button>
        </Feature>
      )

      expect(screen.getByRole('button', { name: 'Submit order' })).toBeInTheDocument()
    })

    it('should preserve aria attributes on fallback', () => {
      render(
        <Feature
          flag="disable_feature"
          fallback={<button aria-label="Use old checkout">Old Checkout</button>}
        >
          <button aria-label="Use new checkout">New Checkout</button>
        </Feature>
      )

      expect(screen.getByRole('button', { name: 'Use old checkout' })).toBeInTheDocument()
    })
  })

  // ============================================================================
  // Error State Tests
  // ============================================================================

  describe('error state', () => {
    it('should render children when ready even if there is an error', () => {
      act(() => {
        useFeatureFlagStore.setState({
          flags: mockFlags,
          isLoading: false,
          isReady: true,
          lastUpdated: new Date(),
          error: 'Failed to refresh flags',
        })
      })

      render(
        <Feature flag="enable_new_checkout">
          <div data-testid="content">Content</div>
        </Feature>
      )

      expect(screen.getByTestId('content')).toBeInTheDocument()
    })

    it('should render fallback when ready with error and flag is disabled', () => {
      act(() => {
        useFeatureFlagStore.setState({
          flags: mockFlags,
          isLoading: false,
          isReady: true,
          lastUpdated: new Date(),
          error: 'Failed to refresh flags',
        })
      })

      render(
        <Feature flag="disable_feature" fallback={<div data-testid="fallback">Fallback</div>}>
          <div data-testid="content">Content</div>
        </Feature>
      )

      expect(screen.queryByTestId('content')).not.toBeInTheDocument()
      expect(screen.getByTestId('fallback')).toBeInTheDocument()
    })
  })
})
