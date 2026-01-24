/**
 * Vitest Test Setup
 *
 * This file runs before each test file. It configures:
 * - @testing-library/jest-dom matchers for DOM assertions
 * - Global test utilities and cleanup
 * - Mock configurations for browser APIs
 */

import '@testing-library/jest-dom'
import { cleanup } from '@testing-library/react'
import { afterEach, vi } from 'vitest'

// Cleanup after each test case
afterEach(() => {
  cleanup()
})

// Mock window.matchMedia for components that use media queries
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(), // deprecated
    removeListener: vi.fn(), // deprecated
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
})

// Mock ResizeObserver for components that observe element sizes
class MockResizeObserver {
  observe = vi.fn()
  unobserve = vi.fn()
  disconnect = vi.fn()
}
window.ResizeObserver = MockResizeObserver

// Mock IntersectionObserver for lazy loading components
class MockIntersectionObserver {
  constructor(callback: IntersectionObserverCallback) {
    this.callback = callback
  }
  callback: IntersectionObserverCallback
  root = null
  rootMargin = ''
  thresholds = []
  observe = vi.fn()
  unobserve = vi.fn()
  disconnect = vi.fn()
  takeRecords = vi.fn(() => [])
}
window.IntersectionObserver = MockIntersectionObserver as unknown as typeof IntersectionObserver

// Mock scrollTo for smooth scrolling tests
window.scrollTo = vi.fn()

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
  length: 0,
  key: vi.fn(),
}
Object.defineProperty(window, 'localStorage', { value: localStorageMock })

// Mock sessionStorage
const sessionStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
  length: 0,
  key: vi.fn(),
}
Object.defineProperty(window, 'sessionStorage', { value: sessionStorageMock })

// Suppress console errors during tests (optional - comment out to see errors)
// This can be useful when testing error boundaries or expected errors
vi.spyOn(console, 'error').mockImplementation(() => {})
vi.spyOn(console, 'warn').mockImplementation(() => {})

// Mock HTMLCanvasElement.getContext for lottie-web (used by Semi UI)
HTMLCanvasElement.prototype.getContext = vi.fn(() => ({
  fillRect: vi.fn(),
  clearRect: vi.fn(),
  getImageData: vi.fn(() => ({ data: [] })),
  putImageData: vi.fn(),
  createImageData: vi.fn(() => ({})),
  setTransform: vi.fn(),
  drawImage: vi.fn(),
  save: vi.fn(),
  fillText: vi.fn(),
  restore: vi.fn(),
  beginPath: vi.fn(),
  moveTo: vi.fn(),
  lineTo: vi.fn(),
  closePath: vi.fn(),
  stroke: vi.fn(),
  translate: vi.fn(),
  scale: vi.fn(),
  rotate: vi.fn(),
  arc: vi.fn(),
  fill: vi.fn(),
  measureText: vi.fn(() => ({ width: 0 })),
  transform: vi.fn(),
  rect: vi.fn(),
  clip: vi.fn(),
  fillStyle: '',
  strokeStyle: '',
  globalAlpha: 1,
  lineWidth: 1,
  lineCap: 'butt',
  lineJoin: 'miter',
  miterLimit: 10,
  shadowBlur: 0,
  shadowColor: '',
  shadowOffsetX: 0,
  shadowOffsetY: 0,
  canvas: {
    width: 0,
    height: 0,
  },
})) as unknown as typeof HTMLCanvasElement.prototype.getContext

// Reset mocks before each test
beforeEach(() => {
  vi.clearAllMocks()
  localStorageMock.getItem.mockReturnValue(null)
  sessionStorageMock.getItem.mockReturnValue(null)
})

// Mock react-dom findDOMNode which is deprecated in React 19 but used by Semi UI
vi.mock('react-dom', async () => {
  const actual = await vi.importActual('react-dom')
  return {
    ...actual,
    findDOMNode: (component: unknown) => {
      // Return the component if it's already an element, otherwise return null
      if (component instanceof HTMLElement) {
        return component
      }
      return null
    },
  }
})
