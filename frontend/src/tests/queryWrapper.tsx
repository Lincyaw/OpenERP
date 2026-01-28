/**
 * React Query Test Utilities
 *
 * Provides wrapper components and utilities for testing components
 * that use React Query hooks.
 */

import type { ReactNode } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

/**
 * Create a QueryClient configured for testing
 * - Disables retries to make tests deterministic
 * - Sets gcTime to 0 to prevent caching between tests
 */
export function createTestQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
        staleTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
  })
}

/**
 * Wrapper component for testing React Query hooks
 * Use this in test files with @testing-library/react's render function:
 *
 * @example
 * ```tsx
 * import { render } from '@testing-library/react'
 * import { QueryWrapper } from '@/tests/queryWrapper'
 *
 * render(<MyComponent />, { wrapper: QueryWrapper })
 * ```
 */
export function QueryWrapper({ children }: { children: ReactNode }) {
  const queryClient = createTestQueryClient()
  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
}

/**
 * Create a wrapper with a specific QueryClient instance
 * Useful when you need to access the queryClient in your test
 *
 * @example
 * ```tsx
 * const queryClient = createTestQueryClient()
 * const wrapper = createQueryWrapper(queryClient)
 *
 * render(<MyComponent />, { wrapper })
 *
 * // Later in test
 * await queryClient.invalidateQueries({ queryKey: ['myQuery'] })
 * ```
 */
export function createQueryWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}
