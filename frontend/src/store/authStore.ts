import { create } from 'zustand'
import { devtools, persist } from 'zustand/middleware'
import type { AuthState, AuthActions, User } from './types'

const STORAGE_KEY = 'erp-auth'
const TOKEN_KEY = 'access_token'
const REFRESH_TOKEN_KEY = 'refresh_token'
const USER_KEY = 'user'

/**
 * Initial auth state
 */
const initialState: AuthState = {
  user: null,
  accessToken: null,
  refreshToken: null,
  isLoading: true,
  isAuthenticated: false,
}

/**
 * Auth store for managing authentication state
 *
 * Features:
 * - Persistent storage with localStorage
 * - Permission checking utilities
 * - Devtools integration for debugging
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
          // Also store in localStorage for backward compatibility with guards
          localStorage.setItem(USER_KEY, JSON.stringify(user))
        },

        setTokens: (accessToken: string, refreshToken?: string) => {
          set(
            {
              accessToken,
              refreshToken: refreshToken ?? get().refreshToken,
              isAuthenticated: true,
            },
            false,
            'auth/setTokens'
          )
          // Also store in localStorage for backward compatibility with guards
          localStorage.setItem(TOKEN_KEY, accessToken)
          if (refreshToken) {
            localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken)
          }
        },

        login: (user: User, accessToken: string, refreshToken?: string) => {
          set(
            {
              user,
              accessToken,
              refreshToken: refreshToken ?? null,
              isAuthenticated: true,
              isLoading: false,
            },
            false,
            'auth/login'
          )
          // Store in localStorage for backward compatibility
          localStorage.setItem(TOKEN_KEY, accessToken)
          localStorage.setItem(USER_KEY, JSON.stringify(user))
          if (refreshToken) {
            localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken)
          }
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
          localStorage.removeItem(TOKEN_KEY)
          localStorage.removeItem(REFRESH_TOKEN_KEY)
          localStorage.removeItem(USER_KEY)
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
          // Try to restore auth state from localStorage
          const token = localStorage.getItem(TOKEN_KEY)
          const refreshToken = localStorage.getItem(REFRESH_TOKEN_KEY)
          const userStr = localStorage.getItem(USER_KEY)

          if (token && userStr) {
            try {
              const user = JSON.parse(userStr) as User
              set(
                {
                  user,
                  accessToken: token,
                  refreshToken,
                  isAuthenticated: true,
                  isLoading: false,
                },
                false,
                'auth/initialize'
              )
            } catch {
              // Invalid stored data, clear it
              localStorage.removeItem(TOKEN_KEY)
              localStorage.removeItem(REFRESH_TOKEN_KEY)
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
        // Only persist these fields
        partialize: (state) => ({
          user: state.user,
          accessToken: state.accessToken,
          refreshToken: state.refreshToken,
          isAuthenticated: state.isAuthenticated,
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
