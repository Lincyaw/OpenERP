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
import i18n from '@/i18n'

// API base URL - use environment variable or default to localhost
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1'

// Create axios instance with default configuration
export const axiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// URLs that don't require authentication
const PUBLIC_URLS = ['/auth/login', '/auth/refresh']

// Check if URL is public (doesn't require auth)
function isPublicUrl(url?: string): boolean {
  if (!url) return false
  return PUBLIC_URLS.some((publicUrl) => url.includes(publicUrl))
}

// Request interceptor - add auth token and handle token refresh
axiosInstance.interceptors.request.use(
  async (config: InternalAxiosRequestConfig) => {
    // Skip auth for public URLs
    if (isPublicUrl(config.url)) {
      return config
    }

    let token = localStorage.getItem('access_token')

    // Check if token needs refresh before making request
    if (token && isTokenExpired(token)) {
      // If already refreshing, wait for it to complete
      if (isCurrentlyRefreshing()) {
        const newToken = await waitForRefresh()
        if (newToken) {
          token = newToken
        }
      } else {
        // Start refresh
        const newToken = await refreshAccessToken()
        if (newToken) {
          token = newToken
        } else {
          // Refresh failed, token will be null
          token = null
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

        // Clear auth state and redirect to login
        localStorage.removeItem('access_token')
        localStorage.removeItem('refresh_token')
        localStorage.removeItem('user')
        redirectToLogin(i18n.t('auth:token.expired'))

        return Promise.reject(error)
      } finally {
        isRefreshingFromResponse = false
      }
    }

    // Handle other errors
    if (error.response) {
      const status = error.response.status

      switch (status) {
        case 403:
          // Forbidden - user doesn't have permission
          console.error('Access forbidden')
          break
        case 429:
          // Rate limited
          console.error('Rate limit exceeded')
          break
        case 500:
          // Internal server error
          console.error('Server error')
          break
      }
    } else if (error.request) {
      // Network error
      console.error('Network error - no response received')
    }

    return Promise.reject(error)
  }
)

// Custom request function for orval to use
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
