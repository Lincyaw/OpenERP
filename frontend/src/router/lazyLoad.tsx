import { lazy, Suspense, type ReactNode } from 'react'
import { Spin } from '@douyinfe/semi-ui'

/**
 * Loading fallback component for lazy-loaded routes
 */
function LazyLoadingFallback() {
  return (
    <div
      style={{
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        height: '100%',
        minHeight: '200px',
      }}
    >
      <Spin size="large" tip="Loading..." />
    </div>
  )
}

/**
 * Wrap a lazy-loaded component with Suspense
 * @param factory - Dynamic import function
 * @returns Wrapped component with loading fallback
 */
export function lazyLoad(factory: () => Promise<{ default: React.ComponentType }>): ReactNode {
  const LazyComponent = lazy(factory)
  return (
    <Suspense fallback={<LazyLoadingFallback />}>
      <LazyComponent />
    </Suspense>
  )
}

/**
 * Preload a lazy component (for prefetching)
 * @param factory - Dynamic import function
 */
export function preloadComponent(factory: () => Promise<{ default: React.ComponentType }>): void {
  factory()
}
