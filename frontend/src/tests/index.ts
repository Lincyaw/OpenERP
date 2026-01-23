/**
 * Test Utilities Module
 *
 * Re-exports all test utilities for convenient importing:
 *
 * @example
 * ```tsx
 * import { renderWithProviders, screen, waitFor, userEvent } from '@/tests'
 *
 * test('example', async () => {
 *   const { user } = renderWithProviders(<MyComponent />)
 *   await user.click(screen.getByRole('button'))
 *   await waitFor(() => {
 *     expect(screen.getByText('Success')).toBeInTheDocument()
 *   })
 * })
 * ```
 */

export {
  // Custom render functions
  renderWithProviders,
  renderWithUser,
  // Utilities
  waitForLoadingToFinish,
  createMockApiResponse,
  createMockApiError,
  createMockPaginatedResponse,
  // Re-exports from @testing-library/react
  screen,
  waitFor,
  waitForElementToBeRemoved,
  within,
  fireEvent,
  act,
  cleanup,
  // User event
  userEvent,
} from './utils'
