import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '@/store'

interface SuperAdminGuardProps {
  children: ReactNode
}

/**
 * Route guard component for super admin routes
 *
 * Features:
 * - Redirects unauthenticated users to login
 * - Redirects non-super-admin users to 403 page
 * - Preserves intended destination for post-login redirect
 *
 * Super admin requirements:
 * 1. User must be authenticated
 * 2. User must belong to system tenant (tenant_id = '00000000-0000-0000-0000-000000000000')
 * 3. User must have super_admin role OR tenant:* permissions
 */
export function SuperAdminGuard({ children }: SuperAdminGuardProps) {
  const location = useLocation()
  const { isAuthenticated, isSuperAdmin, isLoading } = useAuthStore()

  // If still loading auth state, don't redirect yet
  if (isLoading) {
    return null // or a loading spinner
  }

  // If not authenticated, redirect to login
  if (!isAuthenticated) {
    sessionStorage.setItem('auth_redirect_path', location.pathname)
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  // If authenticated but not super admin, redirect to 403
  if (!isSuperAdmin) {
    return <Navigate to="/403" state={{ from: location }} replace />
  }

  return <>{children}</>
}
