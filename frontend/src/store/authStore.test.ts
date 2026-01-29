/**
 * Auth Store Tests
 *
 * Tests for the Zustand auth store
 *
 * Note: As of SEC-004, refresh tokens are now stored as httpOnly cookies
 * managed by the backend. The frontend no longer stores or has access to
 * refresh tokens. Only access tokens are kept in memory (not localStorage).
 */

import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { act } from '@testing-library/react'
import { useAuthStore } from '@store/authStore'
import type { User } from '@store/types'

// Mock localStorage
const localStorageMock = {
  store: {} as Record<string, string>,
  getItem: vi.fn((key: string) => localStorageMock.store[key] || null),
  setItem: vi.fn((key: string, value: string) => {
    localStorageMock.store[key] = value
  }),
  removeItem: vi.fn((key: string) => {
    delete localStorageMock.store[key]
  }),
  clear: vi.fn(() => {
    localStorageMock.store = {}
  }),
}

Object.defineProperty(window, 'localStorage', { value: localStorageMock })

describe('useAuthStore', () => {
  const mockUser: User = {
    id: '1',
    username: 'testuser',
    displayName: 'Test User',
    permissions: ['products:read', 'products:create', 'customers:read'],
    roles: ['user'],
  }

  beforeEach(() => {
    // Reset store state before each test
    const store = useAuthStore.getState()
    act(() => {
      store.logout()
    })
    // Clear localStorage mock
    localStorageMock.clear()
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.resetAllMocks()
  })

  describe('login', () => {
    it('should set user and access token on login (refresh token stored in httpOnly cookie)', () => {
      const store = useAuthStore.getState()

      act(() => {
        store.login(mockUser, 'test-access-token', 'test-refresh-token')
      })

      const state = useAuthStore.getState()
      expect(state.user).toEqual(mockUser)
      expect(state.accessToken).toBe('test-access-token')
      // refreshToken should always be null - it's stored in httpOnly cookie by backend
      expect(state.refreshToken).toBeNull()
      expect(state.isAuthenticated).toBe(true)
      expect(state.isLoading).toBe(false)
    })

    it('should store user in localStorage but NOT tokens (security: SEC-004)', () => {
      const store = useAuthStore.getState()

      act(() => {
        store.login(mockUser, 'test-access-token', 'test-refresh-token')
      })

      // User is stored for display purposes
      expect(localStorageMock.setItem).toHaveBeenCalledWith('user', JSON.stringify(mockUser))
      // Tokens should NOT be stored in localStorage (XSS protection)
      expect(localStorageMock.setItem).not.toHaveBeenCalledWith('access_token', expect.any(String))
      expect(localStorageMock.setItem).not.toHaveBeenCalledWith('refresh_token', expect.any(String))
    })
  })

  describe('logout', () => {
    it('should clear user and tokens on logout', () => {
      const store = useAuthStore.getState()

      // First login
      act(() => {
        store.login(mockUser, 'test-access-token')
      })

      // Then logout
      act(() => {
        store.logout()
      })

      const state = useAuthStore.getState()
      expect(state.user).toBeNull()
      expect(state.accessToken).toBeNull()
      expect(state.refreshToken).toBeNull()
      expect(state.isAuthenticated).toBe(false)
    })

    it('should clear localStorage and legacy token storage on logout', () => {
      const store = useAuthStore.getState()

      act(() => {
        store.login(mockUser, 'test-access-token')
      })
      act(() => {
        store.logout()
      })

      // Should clean up legacy token storage and user data
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('access_token')
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('refresh_token')
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('user')
      // Note: httpOnly cookie is cleared by backend, not frontend
    })
  })

  describe('permissions', () => {
    it('should check if user has a specific permission', () => {
      const store = useAuthStore.getState()

      act(() => {
        store.login(mockUser, 'token')
      })

      const state = useAuthStore.getState()
      expect(state.hasPermission('products:read')).toBe(true)
      expect(state.hasPermission('products:delete')).toBe(false)
    })

    it('should check if user has any of the permissions', () => {
      const store = useAuthStore.getState()

      act(() => {
        store.login(mockUser, 'token')
      })

      const state = useAuthStore.getState()
      expect(state.hasAnyPermission(['products:read', 'admin:all'])).toBe(true)
      expect(state.hasAnyPermission(['products:delete', 'admin:all'])).toBe(false)
    })

    it('should check if user has all permissions', () => {
      const store = useAuthStore.getState()

      act(() => {
        store.login(mockUser, 'token')
      })

      const state = useAuthStore.getState()
      expect(state.hasAllPermissions(['products:read', 'products:create'])).toBe(true)
      expect(state.hasAllPermissions(['products:read', 'products:delete'])).toBe(false)
    })

    it('should return false for permissions when user is not logged in', () => {
      const state = useAuthStore.getState()
      expect(state.hasPermission('products:read')).toBe(false)
      expect(state.hasAnyPermission(['products:read'])).toBe(false)
      expect(state.hasAllPermissions(['products:read'])).toBe(false)
    })
  })

  describe('updateUser', () => {
    it('should update user properties', () => {
      const store = useAuthStore.getState()

      act(() => {
        store.login(mockUser, 'token')
      })
      act(() => {
        store.updateUser({ displayName: 'Updated Name' })
      })

      const state = useAuthStore.getState()
      expect(state.user?.displayName).toBe('Updated Name')
      expect(state.user?.username).toBe('testuser') // Other props unchanged
    })
  })

  describe('selector hooks', () => {
    it('should provide user selector', () => {
      const store = useAuthStore.getState()

      act(() => {
        store.login(mockUser, 'token')
      })

      // Test the selector function
      const selectUser = (state: ReturnType<typeof useAuthStore.getState>) => state.user
      const user = selectUser(useAuthStore.getState())
      expect(user).toEqual(mockUser)
    })
  })
})
