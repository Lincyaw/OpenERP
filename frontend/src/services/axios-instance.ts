import axios from 'axios'
import type {
  AxiosError,
  AxiosRequestConfig,
  AxiosResponse,
  InternalAxiosRequestConfig,
} from 'axios'
import {
  isTokenExpired,
  refreshAccessToken,
  isCurrentlyRefreshing,
  waitForRefresh,
  redirectToLogin,
} from './token-refresh'
import { handleError, ErrorType, parseError } from './error-handler'
import { useAuthStore } from '@/store'
import i18n from '@/i18n'

// API base URL - use environment variable or default to localhost
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1'

/**
 * Create axios instance with default configuration
 *
 * Security improvements (SEC-004):
 * - withCredentials: true - enables sending httpOnly cookies for auth
 * - Access token read from memory (Zustand store) instead of localStorage
 * - Refresh token is now an httpOnly cookie (not accessible via JS)
 */
export const axiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
  // CRITICAL: Enable credentials to send httpOnly cookies with requests
  withCredentials: true,
})

// URLs that don't require authentication
const PUBLIC_URLS = ['/auth/login', '/auth/refresh']

// Check if URL is public (doesn't require auth)
function isPublicUrl(url?: string): boolean {
  if (!url) return false
  return PUBLIC_URLS.some((publicUrl) => url.includes(publicUrl))
}

/**
 * Get current access token from memory (Zustand store)
 * Tokens are no longer stored in localStorage for security
 */
function getAccessToken(): string | null {
  return useAuthStore.getState().accessToken
}

// Request interceptor - add auth token and handle token refresh
axiosInstance.interceptors.request.use(
  async (config: InternalAxiosRequestConfig) => {
    // Skip auth for public URLs
    if (isPublicUrl(config.url)) {
      return config
    }

    // Get access token from memory (Zustand store)
    let token = getAccessToken()

    // Check if token needs refresh before making request
    if (token && isTokenExpired(token)) {
      // If already refreshing, wait for it to complete
      if (isCurrentlyRefreshing()) {
        const newToken = await waitForRefresh()
        if (newToken) {
          token = newToken
        }
      } else {
        // Start refresh - this will use the httpOnly cookie automatically
        const newToken = await refreshAccessToken()
        if (newToken) {
          token = newToken
        } else {
          // Refresh failed, token will be null
          token = null
        }
      }
    }

    // If no token in memory but user exists, try to refresh
    // This handles page refresh scenario where access token is lost
    if (!token && !isPublicUrl(config.url)) {
      const userStr = localStorage.getItem('user')
      if (userStr) {
        // User exists, try to refresh token using httpOnly cookie
        if (!isCurrentlyRefreshing()) {
          const newToken = await refreshAccessToken()
          if (newToken) {
            token = newToken
          }
        } else {
          token = await waitForRefresh()
        }
      }
    }

    // Add auth header if we have a token
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }

    // Add tenant ID from user data
    const userStr = localStorage.getItem('user')
    if (userStr) {
      try {
        const user = JSON.parse(userStr)
        if (user.tenantId) {
          config.headers['X-Tenant-ID'] = user.tenantId
        }
      } catch {
        // Invalid user data, skip
      }
    }

    // Add request ID for tracing
    config.headers['X-Request-ID'] =
      config.headers['X-Request-ID'] ||
      (typeof crypto !== 'undefined' && crypto.randomUUID
        ? crypto.randomUUID()
        : `${Date.now()}-${Math.random().toString(36).substring(2, 11)}`)

    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Track failed requests for retry after refresh
interface FailedRequest {
  resolve: (value: AxiosResponse) => void
  reject: (error: AxiosError) => void
  config: InternalAxiosRequestConfig
}

let failedQueue: FailedRequest[] = []
let isRefreshingFromResponse = false

// Process failed queue after refresh
function processQueue(token: string | null): void {
  failedQueue.forEach((request) => {
    if (token) {
      // Retry with new token
      request.config.headers.Authorization = `Bearer ${token}`
      axiosInstance(request.config)
        .then((response) => request.resolve(response))
        .catch((error) => request.reject(error))
    } else {
      // Reject if refresh failed
      request.reject(new axios.AxiosError('Token refresh failed'))
    }
  })
  failedQueue = []
}

// Response interceptor - handle 401 and other errors
axiosInstance.interceptors.response.use(
  (response: AxiosResponse) => {
    return response
  },
  async (error: AxiosError) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean }

    // Handle 401 Unauthorized
    if (error.response?.status === 401 && !originalRequest._retry) {
      // Don't retry for auth endpoints
      if (isPublicUrl(originalRequest.url)) {
        return Promise.reject(error)
      }

      // Mark as retry to prevent infinite loop
      originalRequest._retry = true

      // If already refreshing, queue this request
      if (isRefreshingFromResponse) {
        return new Promise<AxiosResponse>((resolve, reject) => {
          failedQueue.push({
            resolve,
            reject,
            config: originalRequest,
          })
        })
      }

      isRefreshingFromResponse = true

      try {
        // Refresh uses httpOnly cookie automatically (withCredentials: true)
        const newToken = await refreshAccessToken()

        if (newToken) {
          // Process queued requests
          processQueue(newToken)

          // Retry original request with new token
          originalRequest.headers.Authorization = `Bearer ${newToken}`
          return axiosInstance(originalRequest)
        }

        // Refresh failed, process queue with null
        processQueue(null)

        // Clear auth state (logout cleans up user data from localStorage)
        useAuthStore.getState().logout()
        redirectToLogin(i18n.t('auth:token.expired'))

        return Promise.reject(error)
      } finally {
        isRefreshingFromResponse = false
      }
    }

    // Handle other errors with unified error handler
    // Don't show toast for 401 errors as they're handled above (redirect to login)
    const errorDetails = parseError(error)

    if (error.response) {
      const status = error.response.status

      switch (status) {
        case 403:
          // Forbidden - show error message but don't redirect
          handleError(error, { showToast: true, context: 'Permission denied' })
          break
        case 429:
          // Rate limited - show error message
          handleError(error, { showToast: true, context: 'Rate limited' })
          break
        case 500:
        case 502:
        case 503:
        case 504:
          // Server errors - show error message
          handleError(error, { showToast: true, context: 'Server error' })
          break
        default:
          // Other HTTP errors (400, 404, 409, etc.)
          // Only show toast if it's not a validation error (those should be handled by forms)
          if (errorDetails.type !== ErrorType.VALIDATION) {
            handleError(error, { showToast: true })
          }
      }
    } else if (error.request) {
      // Network error - no response received
      handleError(error, { showToast: true, context: 'Network error' })
    }

    return Promise.reject(error)
  }
)

/**
 * Custom request function for orval to use
 *
 * Note: withCredentials is already set on the axiosInstance,
 * so all requests will include httpOnly cookies automatically.
 */
export const customInstance = <T>(
  config: AxiosRequestConfig,
  options?: AxiosRequestConfig
): Promise<T> => {
  // Use AbortController signal if provided, otherwise use CancelToken for backwards compatibility
  if (options?.signal) {
    const promise = axiosInstance({
      ...config,
      ...options,
    }).then(({ data }) => data)

    return promise
  }

  // Legacy CancelToken support for backwards compatibility
  const source = axios.CancelToken.source()

  const promise = axiosInstance({
    ...config,
    ...options,
    cancelToken: source.token,
  }).then(({ data }) => data)

  // @ts-expect-error - Adding cancel method to promise
  promise.cancel = () => {
    source.cancel('Query was cancelled')
  }

  return promise
}

export default axiosInstance
