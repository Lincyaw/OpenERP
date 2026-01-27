// API services and HTTP client configuration
// Contains axios instance, API interceptors, and service definitions

export { axiosInstance, customInstance } from './axios-instance'

// Token refresh service for automatic token management
export {
  isTokenExpired,
  isTokenCompletelyExpired,
  getTokenExpiration,
  getTimeUntilExpiry,
  refreshAccessToken,
  setupAutoRefresh,
  redirectToLogin,
} from './token-refresh'

// Unified error handler for API errors (UX-FE-002)
export {
  ErrorType,
  handleError,
  parseError,
  createErrorHandler,
  isAuthError,
  isPermissionError,
  isValidationError,
  isNetworkError,
  canRetryError,
  detectErrorType,
  isErrorType,
} from './error-handler'
export type { ErrorDetails, ErrorHandlerOptions } from './error-handler'

// Re-export generated API clients
export * from '../api/system/system'

// Re-export API models
export * from '../api/models'
