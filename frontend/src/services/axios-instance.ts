import axios from 'axios'
import type { AxiosError, AxiosRequestConfig, AxiosResponse } from 'axios'

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

// Request interceptor - add auth token and tenant ID if available
axiosInstance.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('access_token')
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
    config.headers['X-Request-ID'] = config.headers['X-Request-ID'] || crypto.randomUUID()

    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor - handle common error cases
axiosInstance.interceptors.response.use(
  (response: AxiosResponse) => {
    return response
  },
  (error: AxiosError) => {
    if (error.response) {
      const status = error.response.status

      switch (status) {
        case 401:
          // Unauthorized - clear token and redirect to login
          localStorage.removeItem('access_token')
          localStorage.removeItem('refresh_token')
          localStorage.removeItem('user')
          // Optionally redirect to login page
          // window.location.href = '/login'
          break
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
