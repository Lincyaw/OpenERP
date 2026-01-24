import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '@/store'
import type { RouteMeta } from './types'
import { appRoutes, findRouteByPath } from './routes'

interface AuthGuardProps {
  children: ReactNode
  meta?: RouteMeta
}

/**
 * Find route permissions for a given path
 * Handles dynamic routes with parameters like /products/:id/edit
 */
function findRoutePermissions(pathname: string): string[] | undefined {
  // Try exact match first
  const exactMatch = findRouteByPath(pathname)
  if (exactMatch?.meta?.permissions) {
    return exactMatch.meta.permissions
  }

  // For dynamic routes, try to match the pattern
  // Convert pathname to check parent routes
  const segments = pathname.split('/').filter(Boolean)

  // Try progressively shorter paths to find parent route permissions
  for (let i = segments.length; i > 0; i--) {
    const partialPath = '/' + segments.slice(0, i).join('/')
    const route = findRouteByPath(partialPath)
    if (route?.meta?.permissions) {
      return route.meta.permissions
    }
  }

  // Check module-level permissions (first segment)
  if (segments.length > 0) {
    const modulePath = '/' + segments[0]
    const moduleRoute = appRoutes.find((r) => r.path === modulePath)
    if (moduleRoute?.meta?.permissions) {
      return moduleRoute.meta.permissions
    }
  }

  return undefined
}

/**
 * Route guard component for authentication and authorization
 *
 * Features:
 * - Redirects unauthenticated users to login
 * - Checks route-level permissions using auth store
 * - Preserves intended destination for post-login redirect
 * - Automatically determines required permissions from route configuration
 */
export function AuthGuard({ children, meta }: AuthGuardProps) {
  const location = useLocation()
  const { isAuthenticated, hasAnyPermission, isLoading } = useAuthStore()

  // Check if route requires authentication (default: true)
  const requiresAuth = meta?.requiresAuth !== false

  // If authentication is required and user is not authenticated
  if (requiresAuth && !isAuthenticated) {
    // If still loading auth state, don't redirect yet
    if (isLoading) {
      return null // or a loading spinner
    }
    // Store intended destination and redirect to login
    sessionStorage.setItem('auth_redirect_path', location.pathname)
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  // Get permissions from route meta or from route configuration
  const requiredPermissions = meta?.permissions ?? findRoutePermissions(location.pathname)

  // Check permissions if specified
  if (requiredPermissions && requiredPermissions.length > 0) {
    // Check if user has any of the required permissions
    if (!hasAnyPermission(requiredPermissions)) {
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
  const isLoading = useAuthStore((state) => state.isLoading)

  // Wait for auth state to be determined
  if (isLoading) {
    return null
  }

  if (isAuthenticated) {
    // Redirect to intended destination or dashboard
    const redirectPath = sessionStorage.getItem('auth_redirect_path')
    const from = redirectPath || (location.state as { from?: Location })?.from?.pathname || '/'

    // Clear stored redirect path
    if (redirectPath) {
      sessionStorage.removeItem('auth_redirect_path')
    }

    return <Navigate to={from} replace />
  }

  return <>{children}</>
}
