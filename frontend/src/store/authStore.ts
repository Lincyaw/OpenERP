import { create } from 'zustand'
import { devtools, persist } from 'zustand/middleware'
import type { AuthState, AuthActions, User } from './types'

const STORAGE_KEY = 'erp-auth'
const TOKEN_KEY = 'access_token'
const USER_KEY = 'user'

/**
 * Initial auth state
 *
 * Note: refreshToken is no longer stored in frontend state.
 * It is now stored as an httpOnly cookie for security.
 * The frontend only keeps the access token in memory.
 */
const initialState: AuthState = {
  user: null,
  accessToken: null,
  refreshToken: null, // Kept for type compatibility but always null
  isLoading: true,
  isAuthenticated: false,
}

/**
 * Auth store for managing authentication state
 *
 * Features:
 * - Access token stored in memory only (not localStorage) for XSS protection
 * - Refresh token stored as httpOnly cookie (handled by browser, not accessible via JS)
 * - User data persisted for display purposes
 * - Permission checking utilities
 * - Devtools integration for debugging
 *
 * Security improvements (SEC-004):
 * - Access token is NOT persisted to localStorage
 * - Refresh token is stored as httpOnly cookie by the backend
 * - This prevents XSS attacks from stealing tokens
 *
 * @example
 * ```tsx
 * import { useAuthStore } from '@/store'
 *
 * function MyComponent() {
 *   const { user, isAuthenticated, login, logout } = useAuthStore()
 *
 *   if (!isAuthenticated) {
 *     return <LoginForm onLogin={login} />
 *   }
 *
 *   return <div>Welcome, {user?.displayName}</div>
 * }
 * ```
 */
export const useAuthStore = create<AuthState & AuthActions>()(
  devtools(
    persist(
      (set, get) => ({
        ...initialState,

        setUser: (user: User) => {
          set({ user, isAuthenticated: true }, false, 'auth/setUser')
          // Store user in localStorage for guards (user data is not sensitive)
          localStorage.setItem(USER_KEY, JSON.stringify(user))
        },

        setTokens: (accessToken: string, _refreshToken?: string) => {
          // Only store access token in memory (not localStorage)
          // refreshToken parameter is ignored - it's handled via httpOnly cookie
          set(
            {
              accessToken,
              refreshToken: null, // Always null - stored in httpOnly cookie
              isAuthenticated: true,
            },
            false,
            'auth/setTokens'
          )
          // SECURITY: Do NOT store access token in localStorage
          // It's kept in memory only to prevent XSS token theft
        },

        login: (user: User, accessToken: string, _refreshToken?: string) => {
          // refreshToken parameter is ignored - it's handled via httpOnly cookie
          set(
            {
              user,
              accessToken,
              refreshToken: null, // Always null - stored in httpOnly cookie
              isAuthenticated: true,
              isLoading: false,
            },
            false,
            'auth/login'
          )
          // Store user in localStorage (not sensitive)
          localStorage.setItem(USER_KEY, JSON.stringify(user))
          // SECURITY: Do NOT store tokens in localStorage
        },

        logout: () => {
          set(
            {
              user: null,
              accessToken: null,
              refreshToken: null,
              isAuthenticated: false,
              isLoading: false,
            },
            false,
            'auth/logout'
          )
          // Clear localStorage
          localStorage.removeItem(TOKEN_KEY) // Clean up legacy storage
          localStorage.removeItem('refresh_token') // Clean up legacy storage
          localStorage.removeItem(USER_KEY)
          // Note: httpOnly cookie is cleared by the backend on logout
        },

        updateUser: (updates: Partial<User>) => {
          const currentUser = get().user
          if (currentUser) {
            const updatedUser = { ...currentUser, ...updates }
            set({ user: updatedUser }, false, 'auth/updateUser')
            localStorage.setItem(USER_KEY, JSON.stringify(updatedUser))
          }
        },

        setLoading: (isLoading: boolean) => {
          set({ isLoading }, false, 'auth/setLoading')
        },

        hasPermission: (permission: string) => {
          const { user } = get()
          return user?.permissions?.includes(permission) ?? false
        },

        hasAnyPermission: (permissions: string[]) => {
          const { user } = get()
          if (!user?.permissions) return false
          return permissions.some((p) => user.permissions?.includes(p))
        },

        hasAllPermissions: (permissions: string[]) => {
          const { user } = get()
          if (!user?.permissions) return false
          return permissions.every((p) => user.permissions?.includes(p))
        },

        initialize: () => {
          // Try to restore user from localStorage
          // Note: Access token is NOT restored - it's only kept in memory
          // If page is refreshed, user will need to re-authenticate via refresh token cookie
          const userStr = localStorage.getItem(USER_KEY)

          if (userStr) {
            try {
              const user = JSON.parse(userStr) as User
              // User exists but no access token - will need to refresh
              // The axios interceptor will handle token refresh automatically
              set(
                {
                  user,
                  accessToken: null, // Will be refreshed via httpOnly cookie
                  refreshToken: null,
                  isAuthenticated: false, // Not authenticated until token refresh succeeds
                  isLoading: true, // Keep loading until refresh completes
                },
                false,
                'auth/initialize'
              )
              // Clean up any legacy token storage
              localStorage.removeItem(TOKEN_KEY)
              localStorage.removeItem('refresh_token')
            } catch {
              // Invalid stored data, clear it
              localStorage.removeItem(TOKEN_KEY)
              localStorage.removeItem('refresh_token')
              localStorage.removeItem(USER_KEY)
              set({ ...initialState, isLoading: false }, false, 'auth/initialize')
            }
          } else {
            set({ isLoading: false }, false, 'auth/initialize')
          }
        },
      }),
      {
        name: STORAGE_KEY,
        // Only persist user (not tokens - they're memory-only or httpOnly cookie)
        partialize: (state) => ({
          user: state.user,
          // Do NOT persist accessToken or refreshToken
        }),
      }
    ),
    { name: 'AuthStore' }
  )
)

/**
 * Selector hooks for common auth state access patterns
 */
export const useUser = () => useAuthStore((state) => state.user)
export const useIsAuthenticated = () => useAuthStore((state) => state.isAuthenticated)
export const useAuthLoading = () => useAuthStore((state) => state.isLoading)
