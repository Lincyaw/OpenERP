/**
 * Unified Error Handler Service (UX-FE-002)
 *
 * Provides user-friendly error messages for all API errors.
 * Converts technical error codes/messages to localized, actionable feedback.
 *
 * Key Features:
 * - Error type detection from HTTP status codes
 * - Localized error messages (zh-CN/en-US)
 * - Actionable suggestions for users
 * - Toast notifications with appropriate severity
 */

import { AxiosError } from 'axios'
import { Toast } from '@douyinfe/semi-ui-19'
import i18n from '@/i18n'

/**
 * Error types that can be detected from API responses
 */
export const ErrorType = {
  /** Network connectivity issues (no internet, DNS failure, CORS) */
  NETWORK: 'NETWORK',
  /** Authentication failed (401) - token expired or invalid */
  AUTH: 'AUTH',
  /** Permission denied (403) - user lacks required permissions */
  PERMISSION: 'PERMISSION',
  /** Validation error (400, 422) - invalid request data */
  VALIDATION: 'VALIDATION',
  /** Resource not found (404) */
  NOT_FOUND: 'NOT_FOUND',
  /** Conflict error (409) - duplicate data, concurrent modification */
  CONFLICT: 'CONFLICT',
  /** Rate limited (429) - too many requests */
  RATE_LIMIT: 'RATE_LIMIT',
  /** Server error (500+) - internal server issues */
  SERVER: 'SERVER',
  /** Unknown or unclassified error */
  UNKNOWN: 'UNKNOWN',
} as const

export type ErrorType = (typeof ErrorType)[keyof typeof ErrorType]

/**
 * Error details extracted from API response
 */
export interface ErrorDetails {
  /** Type of error detected */
  type: ErrorType
  /** HTTP status code (if available) */
  statusCode?: number
  /** User-friendly error message (localized) */
  message: string
  /** Technical error details (for logging) */
  technicalMessage?: string
  /** Suggestion for user action (localized) */
  suggestion?: string
  /** Whether to show retry button */
  canRetry: boolean
  /** Whether user should contact support */
  showContactSupport: boolean
  /** Field-level validation errors (for forms) */
  fieldErrors?: Record<string, string>
}

/**
 * Options for error handling behavior
 */
export interface ErrorHandlerOptions {
  /** Show toast notification (default: true) */
  showToast?: boolean
  /** Custom toast duration in seconds */
  toastDuration?: number
  /** Log error to console (default: true in dev) */
  logError?: boolean
  /** Callback after error is handled */
  onError?: (details: ErrorDetails) => void
  /** Context message for the operation that failed */
  context?: string
  /** Fallback message when error details are not available */
  fallbackMessage?: string
}

/**
 * Backend error response structure (if available)
 */
interface BackendErrorResponse {
  error?: {
    code?: string
    message?: string
    details?: Record<string, string>
  }
  message?: string
  status?: string
}

/**
 * Get localized message for backend error code
 *
 * @param code - Backend error code (e.g., "INVALID_STATUS")
 * @returns Localized message or undefined if not found
 */
function getLocalizedErrorCodeMessage(code?: string): string | undefined {
  if (!code) return undefined
  const key = `errors.codes.${code}`
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const translated = String((i18n as any).t(key, { ns: 'common' }))
  // i18n returns the key itself if translation is not found
  return translated === key ? undefined : translated
}

/**
 * Detect error type from HTTP status code
 *
 * @param statusCode - HTTP status code
 * @returns ErrorType enum value
 */
export function detectErrorType(statusCode?: number): ErrorType {
  if (!statusCode) {
    return ErrorType.NETWORK
  }

  switch (statusCode) {
    case 400:
    case 422:
      return ErrorType.VALIDATION
    case 401:
      return ErrorType.AUTH
    case 403:
      return ErrorType.PERMISSION
    case 404:
      return ErrorType.NOT_FOUND
    case 409:
      return ErrorType.CONFLICT
    case 429:
      return ErrorType.RATE_LIMIT
    default:
      if (statusCode >= 500) {
        return ErrorType.SERVER
      }
      return ErrorType.UNKNOWN
  }
}

/**
 * Get localized error message for error type
 *
 * @param type - Error type
 * @returns Localized user-friendly message
 */
function getErrorMessage(type: ErrorType): string {
  const key = `errors.${type.toLowerCase()}.message`
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return String((i18n as any).t(key, { ns: 'common' }))
}

/**
 * Get localized suggestion for error type
 *
 * @param type - Error type
 * @returns Localized suggestion text
 */
function getErrorSuggestion(type: ErrorType): string {
  const key = `errors.${type.toLowerCase()}.suggestion`
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return String((i18n as any).t(key, { ns: 'common' }))
}

/**
 * Determine if error is retryable
 *
 * @param type - Error type
 * @returns Whether the operation can be retried
 */
function isRetryable(type: ErrorType): boolean {
  const retryableTypes: ErrorType[] = [ErrorType.NETWORK, ErrorType.SERVER, ErrorType.RATE_LIMIT]
  return retryableTypes.includes(type)
}

/**
 * Determine if support contact should be shown
 *
 * @param type - Error type
 * @returns Whether to show support contact info
 */
function shouldShowContactSupport(type: ErrorType): boolean {
  const supportTypes: ErrorType[] = [ErrorType.SERVER, ErrorType.UNKNOWN]
  return supportTypes.includes(type)
}

/**
 * Extract field-level validation errors from backend response
 *
 * @param data - Backend error response data
 * @returns Field errors map or undefined
 */
function extractFieldErrors(
  data: BackendErrorResponse | undefined
): Record<string, string> | undefined {
  if (!data?.error?.details) {
    return undefined
  }
  return data.error.details
}

/**
 * Parse error details from AxiosError or unknown error
 *
 * @param error - Error object (AxiosError or unknown)
 * @returns Parsed ErrorDetails
 */
export function parseError(error: unknown): ErrorDetails {
  // Handle AxiosError
  if (error instanceof AxiosError) {
    const statusCode = error.response?.status
    const type = detectErrorType(statusCode)
    const backendData = error.response?.data as BackendErrorResponse | undefined

    // Try to get backend error code and message
    const backendCode = backendData?.error?.code
    const backendMessage = backendData?.error?.message || backendData?.message

    // Try to get localized message for the error code first
    const localizedCodeMessage = getLocalizedErrorCodeMessage(backendCode)

    // For validation/business errors, prefer localized code message > backend message > generic message
    const isBusinessError = type === ErrorType.VALIDATION || type === ErrorType.CONFLICT
    const displayMessage = isBusinessError
      ? localizedCodeMessage || backendMessage || getErrorMessage(type)
      : getErrorMessage(type)

    // Don't show generic suggestions for business errors with specific messages
    const hasSpecificMessage = isBusinessError && (localizedCodeMessage || backendMessage)

    return {
      type,
      statusCode,
      message: displayMessage,
      technicalMessage: backendMessage || error.message,
      suggestion: hasSpecificMessage ? undefined : getErrorSuggestion(type),
      canRetry: isRetryable(type),
      showContactSupport: hasSpecificMessage ? false : shouldShowContactSupport(type),
      fieldErrors: extractFieldErrors(backendData),
    }
  }

  // Handle network errors (no response)
  if (error instanceof Error) {
    // Check for network-related error messages
    const isNetworkError =
      error.message.includes('Network Error') ||
      error.message.includes('Failed to fetch') ||
      error.message.includes('ERR_NETWORK') ||
      error.message.includes('ECONNREFUSED') ||
      error.message.includes('ETIMEDOUT')

    if (isNetworkError) {
      return {
        type: ErrorType.NETWORK,
        message: getErrorMessage(ErrorType.NETWORK),
        technicalMessage: error.message,
        suggestion: getErrorSuggestion(ErrorType.NETWORK),
        canRetry: true,
        showContactSupport: false,
      }
    }

    // For custom thrown errors (e.g., business logic errors), use the error message directly
    // This preserves user-friendly messages that were explicitly thrown
    return {
      type: ErrorType.VALIDATION,
      message: error.message,
      technicalMessage: error.message,
      suggestion: undefined,
      canRetry: false,
      showContactSupport: false,
    }
  }

  // Fallback for unknown error types
  return {
    type: ErrorType.UNKNOWN,
    message: getErrorMessage(ErrorType.UNKNOWN),
    technicalMessage: String(error),
    suggestion: getErrorSuggestion(ErrorType.UNKNOWN),
    canRetry: false,
    showContactSupport: true,
  }
}

/**
 * Show toast notification for error
 *
 * @param details - Error details
 * @param duration - Toast duration in seconds
 */
function showErrorToast(details: ErrorDetails, duration: number = 5): void {
  const { type, message, suggestion, showContactSupport } = details

  // Build full message with suggestion
  let fullMessage = message
  if (suggestion) {
    fullMessage += `\n${suggestion}`
  }
  if (showContactSupport) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    fullMessage += `\n${String((i18n as any).t('errors.contactSupport', { ns: 'common' }))}`
  }

  // Use appropriate toast type based on error severity
  switch (type) {
    case ErrorType.AUTH:
    case ErrorType.PERMISSION:
      Toast.warning({
        content: fullMessage,
        duration,
        showClose: true,
      })
      break
    case ErrorType.VALIDATION:
      // Validation errors typically shown inline, but show toast as fallback
      Toast.warning({
        content: fullMessage,
        duration,
        showClose: true,
      })
      break
    case ErrorType.NOT_FOUND:
    case ErrorType.CONFLICT:
    case ErrorType.RATE_LIMIT:
    case ErrorType.SERVER:
    case ErrorType.NETWORK:
    case ErrorType.UNKNOWN:
    default:
      Toast.error({
        content: fullMessage,
        duration,
        showClose: true,
      })
  }
}

/**
 * Unified error handler for API errors
 *
 * Parses the error, logs it (if enabled), shows toast notification,
 * and returns structured error details for further handling.
 *
 * @param error - Error object (AxiosError or unknown)
 * @param options - Error handling options
 * @returns Parsed ErrorDetails
 *
 * @example
 * ```ts
 * try {
 *   await api.createOrder(data)
 * } catch (error) {
 *   const details = handleError(error, {
 *     context: 'Creating order',
 *     onError: (details) => setFormErrors(details.fieldErrors)
 *   })
 *   if (details.canRetry) {
 *     setShowRetryButton(true)
 *   }
 * }
 * ```
 */
export function handleError(error: unknown, options: ErrorHandlerOptions = {}): ErrorDetails {
  const {
    showToast = true,
    toastDuration = 5,
    logError = import.meta.env.DEV,
    onError,
    context,
    fallbackMessage,
  } = options

  const details = parseError(error)

  // Use fallback message if provided and no specific message was found
  if (fallbackMessage && details.type === ErrorType.UNKNOWN && !details.technicalMessage) {
    details.message = fallbackMessage
  }

  // Log error in development or when enabled
  // TODO: Integrate with error tracking service (e.g., Sentry) for production
  if (logError) {
    const logContext = context ? `[${context}]` : ''
    console.error(
      `${logContext} Error (${details.type}):`,
      details.technicalMessage,
      error instanceof AxiosError ? error.response?.data : error
    )
  }

  // Show toast notification
  if (showToast) {
    showErrorToast(details, toastDuration)
  }

  // Call custom error handler if provided
  if (onError) {
    onError(details)
  }

  return details
}

/**
 * Create a wrapped error handler for use in catch blocks
 *
 * @param context - Context description for the operation
 * @param options - Additional error handling options
 * @returns Error handler function
 *
 * @example
 * ```ts
 * const handleOrderError = createErrorHandler('Creating order')
 *
 * createOrder(data).catch(handleOrderError)
 * ```
 */
export function createErrorHandler(
  context: string,
  options: Omit<ErrorHandlerOptions, 'context'> = {}
) {
  return (error: unknown): ErrorDetails => {
    return handleError(error, { ...options, context })
  }
}

/**
 * Check if error is of a specific type
 *
 * @param error - Error object
 * @param type - Expected error type
 * @returns Whether error matches the type
 */
export function isErrorType(error: unknown, type: ErrorType): boolean {
  const details = parseError(error)
  return details.type === type
}

/**
 * Check if error is an authentication error (401)
 */
export function isAuthError(error: unknown): boolean {
  return isErrorType(error, ErrorType.AUTH)
}

/**
 * Check if error is a permission error (403)
 */
export function isPermissionError(error: unknown): boolean {
  return isErrorType(error, ErrorType.PERMISSION)
}

/**
 * Check if error is a validation error (400/422)
 */
export function isValidationError(error: unknown): boolean {
  return isErrorType(error, ErrorType.VALIDATION)
}

/**
 * Check if error is a network error
 */
export function isNetworkError(error: unknown): boolean {
  return isErrorType(error, ErrorType.NETWORK)
}

/**
 * Check if error is retryable
 */
export function canRetryError(error: unknown): boolean {
  const details = parseError(error)
  return details.canRetry
}
