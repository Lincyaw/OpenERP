import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '@/store'
import type { RouteMeta } from './types'

interface AuthGuardProps {
  children: ReactNode
  meta?: RouteMeta
}

/**
 * Route guard component for authentication and authorization
 *
 * Features:
 * - Redirects unauthenticated users to login
 * - Checks route-level permissions using auth store
 * - Preserves intended destination for post-login redirect
 */
export function AuthGuard({ children, meta }: AuthGuardProps) {
  const location = useLocation()
  const { isAuthenticated, hasAnyPermission } = useAuthStore()

  // Check if route requires authentication (default: true)
  const requiresAuth = meta?.requiresAuth !== false

  // If authentication is required and user is not authenticated
  if (requiresAuth && !isAuthenticated) {
    // Redirect to login with return URL
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  // Check permissions if specified
  if (meta?.permissions && meta.permissions.length > 0) {
    // Check if user has any of the required permissions
    if (!hasAnyPermission(meta.permissions)) {
      // Redirect to 403 forbidden page
      return <Navigate to="/403" replace />
    }
  }

  return <>{children}</>
}

interface GuestGuardProps {
  children: ReactNode
}

/**
 * Guard for guest-only routes (e.g., login page)
 * Redirects authenticated users away from login
 */
export function GuestGuard({ children }: GuestGuardProps) {
  const location = useLocation()
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)

  if (isAuthenticated) {
    // Redirect to intended destination or dashboard
    const from = (location.state as { from?: Location })?.from?.pathname || '/'
    return <Navigate to={from} replace />
  }

  return <>{children}</>
}
