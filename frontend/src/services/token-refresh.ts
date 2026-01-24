/**
 * Token Refresh Service
 *
 * Handles automatic token refresh for JWT authentication.
 * Uses a singleton pattern to prevent multiple simultaneous refresh requests.
 *
 * Features:
 * - Automatic token refresh when access token is expired
 * - Request queuing during refresh to prevent race conditions
 * - Token expiration detection and proactive refresh
 * - Secure logout on refresh failure
 */

import { getAuth } from '@/api/auth'
import { useAuthStore } from '@/store'

// Create auth API instance
const authApi = getAuth()

// Token expiration buffer (refresh 1 minute before expiry)
const TOKEN_EXPIRY_BUFFER_MS = 60 * 1000

// Track refresh state
let isRefreshing = false
let refreshPromise: Promise<string | null> | null = null

/**
 * Decode JWT token to get expiration time
 * @param token JWT token string
 * @returns Expiration timestamp in milliseconds, or null if invalid
 */
export function getTokenExpiration(token: string): number | null {
  try {
    // JWT structure: header.payload.signature
    const parts = token.split('.')
    if (parts.length !== 3) {
      return null
    }

    // Decode payload (base64url to JSON)
    const payload = parts[1]
    const decoded = atob(payload.replace(/-/g, '+').replace(/_/g, '/'))
    const data = JSON.parse(decoded)

    // exp is in seconds, convert to milliseconds
    if (typeof data.exp === 'number') {
      return data.exp * 1000
    }

    return null
  } catch {
    return null
  }
}

/**
 * Check if token is expired or about to expire
 * @param token JWT token string
 * @returns true if token is expired or will expire within buffer time
 */
export function isTokenExpired(token: string | null): boolean {
  if (!token) {
    return true
  }

  const expiration = getTokenExpiration(token)
  if (!expiration) {
    // Can't determine expiration, assume expired for safety
    return true
  }

  // Check if token will expire within buffer time
  return Date.now() >= expiration - TOKEN_EXPIRY_BUFFER_MS
}

/**
 * Check if token is completely expired (past expiration time)
 * @param token JWT token string
 * @returns true if token is past its expiration time
 */
export function isTokenCompletelyExpired(token: string | null): boolean {
  if (!token) {
    return true
  }

  const expiration = getTokenExpiration(token)
  if (!expiration) {
    return true
  }

  return Date.now() >= expiration
}

/**
 * Get time until token expires
 * @param token JWT token string
 * @returns Time in milliseconds until expiration, or 0 if expired
 */
export function getTimeUntilExpiry(token: string | null): number {
  if (!token) {
    return 0
  }

  const expiration = getTokenExpiration(token)
  if (!expiration) {
    return 0
  }

  const remaining = expiration - Date.now()
  return Math.max(0, remaining)
}

/**
 * Refresh the access token using the refresh token
 * Uses a singleton pattern to prevent multiple simultaneous refresh requests.
 * @returns New access token or null if refresh failed
 */
export async function refreshAccessToken(): Promise<string | null> {
  const { refreshToken, logout } = useAuthStore.getState()

  // No refresh token available
  if (!refreshToken) {
    logout()
    return null
  }

  // If already refreshing, wait for the existing promise
  if (isRefreshing && refreshPromise) {
    return refreshPromise
  }

  // Start refresh process
  isRefreshing = true
  refreshPromise = performRefresh(refreshToken)

  try {
    const newToken = await refreshPromise
    return newToken
  } finally {
    isRefreshing = false
    refreshPromise = null
  }
}

/**
 * Perform the actual token refresh API call
 * @param refreshToken Current refresh token
 * @returns New access token or null if refresh failed
 */
async function performRefresh(refreshToken: string): Promise<string | null> {
  const { setTokens, logout } = useAuthStore.getState()

  try {
    const response = await authApi.postAuthRefresh({
      refresh_token: refreshToken,
    })

    if (!response.success || !response.data) {
      // Refresh failed, logout user
      logout()
      redirectToLogin('Session expired. Please log in again.')
      return null
    }

    const { token } = response.data

    // Update tokens in store
    if (token?.access_token && token?.refresh_token) {
      setTokens(token.access_token, token.refresh_token)
      return token.access_token
    }

    logout()
    redirectToLogin('Session expired. Please log in again.')
    return null
  } catch (error) {
    // Refresh failed, logout user
    logout()
    redirectToLogin('Session expired. Please log in again.')
    return null
  }
}

/**
 * Redirect to login page with an optional message
 * @param message Optional message to display on login page
 */
export function redirectToLogin(message?: string): void {
  // Store message for login page to display
  if (message) {
    sessionStorage.setItem('auth_redirect_message', message)
  }

  // Get current path for redirect back after login
  const currentPath = window.location.pathname
  if (currentPath !== '/login') {
    sessionStorage.setItem('auth_redirect_path', currentPath)
  }

  // Redirect to login
  window.location.href = '/login'
}

/**
 * Setup automatic token refresh timer
 * Schedules a refresh before the token expires
 * @returns Cleanup function to clear the timer
 */
export function setupAutoRefresh(): () => void {
  let timerId: ReturnType<typeof setTimeout> | null = null

  const scheduleRefresh = () => {
    const { accessToken, isAuthenticated } = useAuthStore.getState()

    // Clear existing timer
    if (timerId) {
      clearTimeout(timerId)
      timerId = null
    }

    // Don't schedule if not authenticated
    if (!isAuthenticated || !accessToken) {
      return
    }

    // Calculate time until we should refresh
    const timeUntilRefresh = getTimeUntilExpiry(accessToken) - TOKEN_EXPIRY_BUFFER_MS

    if (timeUntilRefresh <= 0) {
      // Token already needs refresh
      refreshAccessToken().then(() => {
        // Schedule next refresh after getting new token
        scheduleRefresh()
      })
    } else {
      // Schedule refresh
      timerId = setTimeout(async () => {
        await refreshAccessToken()
        scheduleRefresh()
      }, timeUntilRefresh)
    }
  }

  // Initial schedule
  scheduleRefresh()

  // Subscribe to auth state changes
  const unsubscribe = useAuthStore.subscribe((state, prevState) => {
    // Re-schedule when tokens change or auth state changes
    if (
      state.accessToken !== prevState.accessToken ||
      state.isAuthenticated !== prevState.isAuthenticated
    ) {
      scheduleRefresh()
    }
  })

  // Return cleanup function
  return () => {
    if (timerId) {
      clearTimeout(timerId)
    }
    unsubscribe()
  }
}

/**
 * Check if we should attempt to refresh (not currently refreshing)
 */
export function canRefresh(): boolean {
  return !isRefreshing
}

/**
 * Check if we're currently in a refresh operation
 */
export function isCurrentlyRefreshing(): boolean {
  return isRefreshing
}

/**
 * Wait for any ongoing refresh to complete
 * @returns The result of the refresh, or null if no refresh is in progress
 */
export async function waitForRefresh(): Promise<string | null> {
  if (refreshPromise) {
    return refreshPromise
  }
  return null
}
