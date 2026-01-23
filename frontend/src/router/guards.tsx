import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import type { RouteMeta } from './types'

interface AuthGuardProps {
  children: ReactNode
  meta?: RouteMeta
}

/**
 * Check if user is authenticated
 * TODO: Replace with actual auth check from auth store
 */
function isAuthenticated(): boolean {
  // Check for token in localStorage
  const token = localStorage.getItem('access_token')
  return !!token
}

/**
 * Check if user has required permissions
 * TODO: Replace with actual permission check from auth store
 */
function hasPermissions(requiredPermissions?: string[]): boolean {
  if (!requiredPermissions || requiredPermissions.length === 0) {
    return true
  }

  // TODO: Get user permissions from auth store
  // For now, return true if authenticated
  return isAuthenticated()
}

/**
 * Route guard component for authentication and authorization
 *
 * Features:
 * - Redirects unauthenticated users to login
 * - Checks route-level permissions
 * - Preserves intended destination for post-login redirect
 */
export function AuthGuard({ children, meta }: AuthGuardProps) {
  const location = useLocation()

  // Check if route requires authentication (default: true)
  const requiresAuth = meta?.requiresAuth !== false

  // If authentication is required and user is not authenticated
  if (requiresAuth && !isAuthenticated()) {
    // Redirect to login with return URL
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  // Check permissions if specified
  if (meta?.permissions && !hasPermissions(meta.permissions)) {
    // Redirect to 403 forbidden page
    return <Navigate to="/403" replace />
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

  if (isAuthenticated()) {
    // Redirect to intended destination or dashboard
    const from = (location.state as { from?: Location })?.from?.pathname || '/'
    return <Navigate to={from} replace />
  }

  return <>{children}</>
}
