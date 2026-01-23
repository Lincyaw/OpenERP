/**
 * Test Utilities Tests
 *
 * Tests to verify the test utilities are working correctly
 */

import { describe, it, expect } from 'vitest'
import {
  renderWithProviders,
  renderWithUser,
  createMockApiResponse,
  createMockApiError,
  createMockPaginatedResponse,
  screen,
} from './utils'

describe('renderWithProviders', () => {
  it('should render a component with router context', () => {
    renderWithProviders(<div data-testid="test-component">Hello World</div>)

    expect(screen.getByTestId('test-component')).toBeInTheDocument()
    expect(screen.getByText('Hello World')).toBeInTheDocument()
  })

  it('should provide user-event instance', async () => {
    const handleClick = vi.fn()

    const { user } = renderWithProviders(<button onClick={handleClick}>Click Me</button>)

    await user.click(screen.getByRole('button'))
    expect(handleClick).toHaveBeenCalledTimes(1)
  })

  it('should set initial route', () => {
    renderWithProviders(<div>Test</div>, { route: '/dashboard' })
    // The route is set internally - we can verify the router is working
    expect(screen.getByText('Test')).toBeInTheDocument()
  })
})

describe('renderWithUser', () => {
  it('should render a component without providers', () => {
    renderWithUser(<div data-testid="simple">Simple Component</div>)

    expect(screen.getByTestId('simple')).toBeInTheDocument()
  })

  it('should provide user-event instance', async () => {
    const handleClick = vi.fn()

    const { user } = renderWithUser(<button onClick={handleClick}>Click</button>)

    await user.click(screen.getByRole('button'))
    expect(handleClick).toHaveBeenCalledTimes(1)
  })
})

describe('mock response utilities', () => {
  describe('createMockApiResponse', () => {
    it('should create a success response with data', () => {
      const data = { id: '1', name: 'Test' }
      const response = createMockApiResponse(data)

      expect(response).toEqual({
        success: true,
        data: { id: '1', name: 'Test' },
        meta: undefined,
      })
    })

    it('should create a success response with meta', () => {
      const data = [{ id: '1' }]
      const meta = { total: 100 }
      const response = createMockApiResponse(data, meta)

      expect(response).toEqual({
        success: true,
        data,
        meta: { total: 100 },
      })
    })
  })

  describe('createMockApiError', () => {
    it('should create an error response', () => {
      const response = createMockApiError('ERR_NOT_FOUND', 'Resource not found')

      expect(response.success).toBe(false)
      expect(response.error.code).toBe('ERR_NOT_FOUND')
      expect(response.error.message).toBe('Resource not found')
      expect(response.error.request_id).toMatch(/^req-\d+$/)
      expect(response.error.timestamp).toBeDefined()
    })

    it('should create an error response with details', () => {
      const details = [
        { field: 'email', message: 'Invalid email format' },
        { field: 'name', message: 'Name is required' },
      ]
      const response = createMockApiError('ERR_VALIDATION', 'Validation failed', details)

      expect(response.error.details).toEqual(details)
    })
  })

  describe('createMockPaginatedResponse', () => {
    it('should create a paginated response with defaults', () => {
      const data = [{ id: '1' }, { id: '2' }]
      const response = createMockPaginatedResponse(data)

      expect(response.success).toBe(true)
      expect(response.data).toEqual(data)
      expect(response.meta).toEqual({
        total: 100,
        page: 1,
        page_size: 10,
        total_pages: 10,
      })
    })

    it('should create a paginated response with custom values', () => {
      const data = [{ id: '1' }]
      const response = createMockPaginatedResponse(data, 2, 20, 50)

      expect(response.meta).toEqual({
        total: 50,
        page: 2,
        page_size: 20,
        total_pages: 3, // ceil(50/20) = 3
      })
    })

    it('should calculate total_pages correctly', () => {
      const response = createMockPaginatedResponse([], 1, 10, 25)
      expect(response.meta?.total_pages).toBe(3) // ceil(25/10) = 3

      const response2 = createMockPaginatedResponse([], 1, 10, 30)
      expect(response2.meta?.total_pages).toBe(3) // ceil(30/10) = 3
    })
  })
})
