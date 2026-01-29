import { useSyncExternalStore, useMemo } from 'react'

/**
 * Breakpoint values (must match CSS variables in breakpoints.css)
 */
const BREAKPOINTS = {
  mobile: 375,
  tablet: 768,
  desktop: 1024,
  wide: 1440,
} as const

type Breakpoint = keyof typeof BREAKPOINTS

interface ResponsiveState {
  /** Current viewport width */
  width: number
  /** Whether viewport is mobile (< 768px) */
  isMobile: boolean
  /** Whether viewport is tablet (>= 768px and < 1024px) */
  isTablet: boolean
  /** Whether viewport is desktop (>= 1024px and < 1440px) */
  isDesktop: boolean
  /** Whether viewport is wide (>= 1440px) */
  isWide: boolean
  /** Current breakpoint name */
  breakpoint: Breakpoint
}

/**
 * Get the current breakpoint based on viewport width
 */
function getBreakpoint(width: number): Breakpoint {
  if (width >= BREAKPOINTS.wide) return 'wide'
  if (width >= BREAKPOINTS.desktop) return 'desktop'
  if (width >= BREAKPOINTS.tablet) return 'tablet'
  return 'mobile'
}

/**
 * Get current window width safely
 */
function getWindowWidth(): number {
  return typeof window !== 'undefined' ? window.innerWidth : BREAKPOINTS.desktop
}

/**
 * Subscribe to window resize events
 */
function subscribeToResize(callback: () => void): () => void {
  window.addEventListener('resize', callback)
  return () => window.removeEventListener('resize', callback)
}

/**
 * Server snapshot for SSR
 */
function getServerSnapshot(): number {
  return BREAKPOINTS.desktop
}

/**
 * Hook to detect responsive breakpoints and viewport changes
 *
 * Uses useSyncExternalStore for proper integration with React's
 * concurrent rendering.
 *
 * @example
 * ```tsx
 * function MyComponent() {
 *   const { isMobile, isTablet, breakpoint } = useResponsive()
 *
 *   return isMobile ? <MobileView /> : <DesktopView />
 * }
 * ```
 */
export function useResponsive(): ResponsiveState {
  const width = useSyncExternalStore(subscribeToResize, getWindowWidth, getServerSnapshot)

  // Calculate state from width using useMemo for performance
  const state = useMemo<ResponsiveState>(
    () => ({
      width,
      isMobile: width < BREAKPOINTS.tablet,
      isTablet: width >= BREAKPOINTS.tablet && width < BREAKPOINTS.desktop,
      isDesktop: width >= BREAKPOINTS.desktop && width < BREAKPOINTS.wide,
      isWide: width >= BREAKPOINTS.wide,
      breakpoint: getBreakpoint(width),
    }),
    [width]
  )

  return state
}

/**
 * Hook to check if viewport matches a specific breakpoint or above
 *
 * @example
 * ```tsx
 * const isDesktopUp = useMediaQuery('desktop') // true if >= 1024px
 * ```
 */
export function useMediaQuery(breakpoint: Breakpoint): boolean {
  const { width } = useResponsive()
  return width >= BREAKPOINTS[breakpoint]
}

export { BREAKPOINTS }
export type { Breakpoint, ResponsiveState }
