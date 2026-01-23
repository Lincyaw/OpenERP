/**
 * Test Utility Functions
 *
 * This module provides reusable test utilities for testing React components
 * with providers (Router, Zustand stores, etc.).
 */

import type { ReactElement, ReactNode } from 'react'
import { render, type RenderOptions, type RenderResult } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BrowserRouter, MemoryRouter, type MemoryRouterProps } from 'react-router-dom'

/**
 * Extended render options including router configuration
 */
interface ExtendedRenderOptions extends Omit<RenderOptions, 'wrapper'> {
  /** Initial route for MemoryRouter */
  route?: string
  /** Additional routes for MemoryRouter */
  initialEntries?: MemoryRouterProps['initialEntries']
  /** Use BrowserRouter instead of MemoryRouter */
  useBrowserRouter?: boolean
}

/**
 * Extended render result including user-event instance
 */
interface ExtendedRenderResult extends RenderResult {
  /** Pre-configured user-event instance */
  user: ReturnType<typeof userEvent.setup>
}

/**
 * Creates a wrapper component with all providers needed for testing
 */
function createWrapper(options: ExtendedRenderOptions = {}): React.FC<{ children: ReactNode }> {
  const { route = '/', initialEntries, useBrowserRouter = false } = options

  return function Wrapper({ children }: { children: ReactNode }) {
    if (useBrowserRouter) {
      return <BrowserRouter>{children}</BrowserRouter>
    }

    return <MemoryRouter initialEntries={initialEntries || [route]}>{children}</MemoryRouter>
  }
}

/**
 * Custom render function that includes all providers and user-event setup
 *
 * @example
 * ```tsx
 * const { user, getByRole } = renderWithProviders(<MyComponent />)
 * await user.click(getByRole('button'))
 * ```
 */
export function renderWithProviders(
  ui: ReactElement,
  options: ExtendedRenderOptions = {}
): ExtendedRenderResult {
  const { route, initialEntries, useBrowserRouter, ...renderOptions } = options

  const user = userEvent.setup()

  const Wrapper = createWrapper({ route, initialEntries, useBrowserRouter })

  const result = render(ui, {
    wrapper: Wrapper,
    ...renderOptions,
  })

  return {
    ...result,
    user,
  }
}

/**
 * Render a component without providers (for isolated component tests)
 * Includes user-event setup for convenience
 */
export function renderWithUser(ui: ReactElement, options?: RenderOptions): ExtendedRenderResult {
  const user = userEvent.setup()
  const result = render(ui, options)

  return {
    ...result,
    user,
  }
}

/**
 * Wait for loading states to clear
 * Useful for async operations that show loading indicators
 */
export async function waitForLoadingToFinish() {
  // This is a simple implementation - in a real app you might want to
  // wait for specific loading indicators to disappear
  await new Promise((resolve) => setTimeout(resolve, 0))
}

/**
 * Create a mock API response
 */
export function createMockApiResponse<T>(data: T, meta?: Record<string, unknown>) {
  return {
    success: true,
    data,
    meta,
  }
}

/**
 * Create a mock API error response
 */
export function createMockApiError(
  code: string,
  message: string,
  details?: Array<{ field: string; message: string }>
) {
  return {
    success: false,
    error: {
      code,
      message,
      request_id: `req-${Date.now()}`,
      timestamp: new Date().toISOString(),
      details,
    },
  }
}

/**
 * Create a mock paginated API response
 */
export function createMockPaginatedResponse<T>(
  data: T[],
  page: number = 1,
  pageSize: number = 10,
  total: number = 100
) {
  return {
    success: true,
    data,
    meta: {
      total,
      page,
      page_size: pageSize,
      total_pages: Math.ceil(total / pageSize),
    },
  }
}

// Re-export everything from @testing-library/react for convenience
export * from '@testing-library/react'
export { userEvent }
