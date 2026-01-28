/**
 * Auth Provider Component
 *
 * Initializes authentication state and sets up automatic token refresh.
 * Should wrap the entire application or at least the router.
 *
 * Features:
 * - Initializes auth state from localStorage on mount
 * - Sets up automatic token refresh before expiration
 * - Displays session expired message from redirect
 * - Cleans up refresh timer on unmount
 */

import { useEffect, useRef, type ReactNode } from 'react'
import { Toast } from '@douyinfe/semi-ui-19'
import { useAuthStore } from '@/store'
import { setupAutoRefresh, refreshAccessToken } from '@/services/token-refresh'

interface AuthProviderProps {
  children: ReactNode
}

/**
 * Authentication provider component
 *
 * @example
 * ```tsx
 * // In main.tsx or App.tsx
 * import { AuthProvider } from '@/components/auth'
 *
 * <AuthProvider>
 *   <RouterProvider router={router} />
 * </AuthProvider>
 * ```
 */
export function AuthProvider({ children }: AuthProviderProps) {
  const initialize = useAuthStore((state) => state.initialize)
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)
  const isLoading = useAuthStore((state) => state.isLoading)
  const user = useAuthStore((state) => state.user)
  const accessToken = useAuthStore((state) => state.accessToken)
  const setLoading = useAuthStore((state) => state.setLoading)
  const logout = useAuthStore((state) => state.logout)

  // Track if we've already attempted a refresh to prevent infinite loops
  const hasAttemptedRefresh = useRef(false)

  // Initialize auth state from localStorage
  useEffect(() => {
    initialize()
  }, [initialize])

  // Attempt token refresh when user exists but no access token (e.g., after page refresh)
  useEffect(() => {
    // Only attempt refresh if:
    // - Still loading (initial state after initialize)
    // - User exists (was previously logged in)
    // - No access token (lost on page refresh since it's memory-only)
    // - Haven't already attempted a refresh this session
    if (isLoading && user && !accessToken && !hasAttemptedRefresh.current) {
      hasAttemptedRefresh.current = true

      const attemptRefresh = async () => {
        try {
          const newToken = await refreshAccessToken()
          if (!newToken) {
            // Refresh failed, clear loading state
            // Don't call logout() here - refreshAccessToken already does that
            // Just ensure loading is false so guards can redirect
            setLoading(false)
          }
          // If refresh succeeded, the store will be updated by refreshAccessToken
          // and isLoading will be set appropriately
        } catch {
          // Refresh failed, just clear loading state
          // refreshAccessToken already handles logout
          setLoading(false)
        }
      }
      attemptRefresh()
    }
  }, [isLoading, user, accessToken, setLoading, logout])

  // Set up automatic token refresh when authenticated
  useEffect(() => {
    if (!isAuthenticated) {
      return
    }

    // Setup auto-refresh and get cleanup function
    const cleanup = setupAutoRefresh()

    return cleanup
  }, [isAuthenticated])

  // Check for redirect message (e.g., from session expiration)
  useEffect(() => {
    const message = sessionStorage.getItem('auth_redirect_message')
    if (message) {
      // Show message after a brief delay to ensure UI is ready
      setTimeout(() => {
        Toast.warning({
          content: message,
          duration: 5,
        })
      }, 100)

      // Clear the message
      sessionStorage.removeItem('auth_redirect_message')
    }
  }, [])

  return <>{children}</>
}

/**
 * Hook to access redirect path after login
 * Returns the path user was trying to access before being redirected to login
 */
export function useAuthRedirectPath(): string | null {
  return sessionStorage.getItem('auth_redirect_path')
}

/**
 * Clear the stored redirect path
 * Call this after successful navigation to the intended destination
 */
export function clearAuthRedirectPath(): void {
  sessionStorage.removeItem('auth_redirect_path')
}
