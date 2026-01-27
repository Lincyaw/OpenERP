/**
 * Error Handler Service Tests (UX-FE-002)
 *
 * Tests for the unified error handler service that provides
 * user-friendly error messages for API errors.
 */

import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { AxiosError, AxiosHeaders } from 'axios'
import type { InternalAxiosRequestConfig, AxiosResponse } from 'axios'

// Mock i18n - must be defined before imports that use it
vi.mock('@/i18n', () => ({
  default: {
    t: (key: string) => {
      const translations: Record<string, string> = {
        'errors.network.message': '网络连接失败',
        'errors.network.suggestion': '请检查您的网络连接后重试',
        'errors.auth.message': '登录已过期',
        'errors.auth.suggestion': '请重新登录以继续操作',
        'errors.permission.message': '您没有执行此操作的权限',
        'errors.permission.suggestion': '请联系管理员获取相应权限',
        'errors.validation.message': '提交的数据格式有误',
        'errors.validation.suggestion': '请检查您填写的内容后重试',
        'errors.not_found.message': '请求的数据不存在',
        'errors.not_found.suggestion': '数据可能已被删除或移动',
        'errors.conflict.message': '数据冲突',
        'errors.conflict.suggestion': '该数据可能已被其他用户修改，请刷新页面后重试',
        'errors.rate_limit.message': '操作过于频繁',
        'errors.rate_limit.suggestion': '请稍等片刻后再试',
        'errors.server.message': '系统繁忙',
        'errors.server.suggestion': '请稍后重试',
        'errors.unknown.message': '操作失败',
        'errors.unknown.suggestion': '请稍后重试',
        'errors.contactSupport': '如问题持续，请联系客服',
      }
      return translations[key] || key
    },
  },
}))

// Mock Semi UI Toast - use factory function
vi.mock('@douyinfe/semi-ui-19', () => ({
  Toast: {
    error: vi.fn(),
    warning: vi.fn(),
    success: vi.fn(),
    info: vi.fn(),
  },
}))

// Import module under test after mocks are defined
import {
  ErrorType,
  detectErrorType,
  parseError,
  handleError,
  createErrorHandler,
  isAuthError,
  isPermissionError,
  isValidationError,
  isNetworkError,
  canRetryError,
  isErrorType,
} from './error-handler'

// Import Toast after the mock to get the mocked version
import { Toast } from '@douyinfe/semi-ui-19'

// Helper to create AxiosError
function createAxiosError(
  status: number | undefined,
  message: string = 'Test error',
  responseData?: unknown
): AxiosError {
  const config: InternalAxiosRequestConfig = {
    url: '/api/test',
    headers: new AxiosHeaders(),
  }

  const response: AxiosResponse | undefined = status
    ? {
        data: responseData || {},
        status,
        statusText: 'Error',
        headers: {},
        config,
      }
    : undefined

  const error = new AxiosError(
    message,
    status ? 'ERR_BAD_REQUEST' : 'ERR_NETWORK',
    config,
    { url: '/api/test' },
    response
  )

  return error
}

describe('Error Handler Service', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.resetAllMocks()
  })

  describe('detectErrorType', () => {
    it('should return NETWORK for undefined status code', () => {
      expect(detectErrorType(undefined)).toBe(ErrorType.NETWORK)
    })

    it('should return VALIDATION for 400 status', () => {
      expect(detectErrorType(400)).toBe(ErrorType.VALIDATION)
    })

    it('should return VALIDATION for 422 status', () => {
      expect(detectErrorType(422)).toBe(ErrorType.VALIDATION)
    })

    it('should return AUTH for 401 status', () => {
      expect(detectErrorType(401)).toBe(ErrorType.AUTH)
    })

    it('should return PERMISSION for 403 status', () => {
      expect(detectErrorType(403)).toBe(ErrorType.PERMISSION)
    })

    it('should return NOT_FOUND for 404 status', () => {
      expect(detectErrorType(404)).toBe(ErrorType.NOT_FOUND)
    })

    it('should return CONFLICT for 409 status', () => {
      expect(detectErrorType(409)).toBe(ErrorType.CONFLICT)
    })

    it('should return RATE_LIMIT for 429 status', () => {
      expect(detectErrorType(429)).toBe(ErrorType.RATE_LIMIT)
    })

    it('should return SERVER for 500 status', () => {
      expect(detectErrorType(500)).toBe(ErrorType.SERVER)
    })

    it('should return SERVER for 502 status', () => {
      expect(detectErrorType(502)).toBe(ErrorType.SERVER)
    })

    it('should return SERVER for 503 status', () => {
      expect(detectErrorType(503)).toBe(ErrorType.SERVER)
    })

    it('should return UNKNOWN for unrecognized status codes', () => {
      expect(detectErrorType(418)).toBe(ErrorType.UNKNOWN)
    })
  })

  describe('parseError', () => {
    describe('with AxiosError', () => {
      it('should parse network error correctly', () => {
        const error = createAxiosError(undefined, 'Network Error')
        const details = parseError(error)

        expect(details.type).toBe(ErrorType.NETWORK)
        expect(details.statusCode).toBeUndefined()
        expect(details.message).toBe('网络连接失败')
        expect(details.suggestion).toBe('请检查您的网络连接后重试')
        expect(details.canRetry).toBe(true)
        expect(details.showContactSupport).toBe(false)
      })

      it('should parse 401 auth error correctly', () => {
        const error = createAxiosError(401, 'Unauthorized')
        const details = parseError(error)

        expect(details.type).toBe(ErrorType.AUTH)
        expect(details.statusCode).toBe(401)
        expect(details.message).toBe('登录已过期')
        expect(details.suggestion).toBe('请重新登录以继续操作')
        expect(details.canRetry).toBe(false)
        expect(details.showContactSupport).toBe(false)
      })

      it('should parse 403 permission error correctly', () => {
        const error = createAxiosError(403, 'Forbidden')
        const details = parseError(error)

        expect(details.type).toBe(ErrorType.PERMISSION)
        expect(details.statusCode).toBe(403)
        expect(details.message).toBe('您没有执行此操作的权限')
        expect(details.canRetry).toBe(false)
      })

      it('should parse 400 validation error correctly', () => {
        const responseData = {
          error: {
            message: 'Validation failed',
            details: {
              email: 'Invalid email format',
              name: 'Name is required',
            },
          },
        }
        const error = createAxiosError(400, 'Bad Request', responseData)
        const details = parseError(error)

        expect(details.type).toBe(ErrorType.VALIDATION)
        expect(details.statusCode).toBe(400)
        expect(details.fieldErrors).toEqual({
          email: 'Invalid email format',
          name: 'Name is required',
        })
      })

      it('should parse 404 not found error correctly', () => {
        const error = createAxiosError(404, 'Not Found')
        const details = parseError(error)

        expect(details.type).toBe(ErrorType.NOT_FOUND)
        expect(details.statusCode).toBe(404)
        expect(details.message).toBe('请求的数据不存在')
      })

      it('should parse 409 conflict error correctly', () => {
        const error = createAxiosError(409, 'Conflict')
        const details = parseError(error)

        expect(details.type).toBe(ErrorType.CONFLICT)
        expect(details.statusCode).toBe(409)
        expect(details.message).toBe('数据冲突')
      })

      it('should parse 429 rate limit error correctly', () => {
        const error = createAxiosError(429, 'Too Many Requests')
        const details = parseError(error)

        expect(details.type).toBe(ErrorType.RATE_LIMIT)
        expect(details.statusCode).toBe(429)
        expect(details.canRetry).toBe(true)
      })

      it('should parse 500 server error correctly', () => {
        const error = createAxiosError(500, 'Internal Server Error')
        const details = parseError(error)

        expect(details.type).toBe(ErrorType.SERVER)
        expect(details.statusCode).toBe(500)
        expect(details.message).toBe('系统繁忙')
        expect(details.canRetry).toBe(true)
        expect(details.showContactSupport).toBe(true)
      })

      it('should extract backend error message', () => {
        const responseData = {
          error: {
            message: 'Custom backend message',
          },
        }
        const error = createAxiosError(400, 'Bad Request', responseData)
        const details = parseError(error)

        expect(details.technicalMessage).toBe('Custom backend message')
      })
    })

    describe('with standard Error', () => {
      it('should detect network-related errors', () => {
        const error = new Error('Network Error')
        const details = parseError(error)

        expect(details.type).toBe(ErrorType.NETWORK)
        expect(details.canRetry).toBe(true)
      })

      it('should detect failed to fetch errors', () => {
        const error = new Error('Failed to fetch')
        const details = parseError(error)

        expect(details.type).toBe(ErrorType.NETWORK)
      })

      it('should return UNKNOWN for other errors', () => {
        const error = new Error('Some random error')
        const details = parseError(error)

        expect(details.type).toBe(ErrorType.UNKNOWN)
        expect(details.showContactSupport).toBe(true)
      })
    })

    describe('with unknown error types', () => {
      it('should handle string errors', () => {
        const details = parseError('string error')

        expect(details.type).toBe(ErrorType.UNKNOWN)
        expect(details.technicalMessage).toBe('string error')
      })

      it('should handle null/undefined', () => {
        const details = parseError(null)

        expect(details.type).toBe(ErrorType.UNKNOWN)
      })
    })
  })

  describe('handleError', () => {
    it('should show toast by default', () => {
      const error = createAxiosError(500, 'Server Error')
      handleError(error)

      expect(Toast.error).toHaveBeenCalled()
    })

    it('should not show toast when showToast is false', () => {
      const error = createAxiosError(500, 'Server Error')
      handleError(error, { showToast: false })

      expect(Toast.error).not.toHaveBeenCalled()
    })

    it('should call onError callback with error details', () => {
      const onError = vi.fn()
      const error = createAxiosError(400, 'Bad Request')

      handleError(error, { onError, showToast: false })

      expect(onError).toHaveBeenCalledWith(
        expect.objectContaining({
          type: ErrorType.VALIDATION,
          statusCode: 400,
        })
      )
    })

    it('should use warning toast for auth errors', () => {
      const error = createAxiosError(401, 'Unauthorized')
      handleError(error)

      expect(Toast.warning).toHaveBeenCalled()
    })

    it('should use warning toast for permission errors', () => {
      const error = createAxiosError(403, 'Forbidden')
      handleError(error)

      expect(Toast.warning).toHaveBeenCalled()
    })

    it('should use error toast for server errors', () => {
      const error = createAxiosError(500, 'Server Error')
      handleError(error)

      expect(Toast.error).toHaveBeenCalled()
    })

    it('should use error toast for network errors', () => {
      const error = createAxiosError(undefined, 'Network Error')
      handleError(error)

      expect(Toast.error).toHaveBeenCalled()
    })

    it('should return error details', () => {
      const error = createAxiosError(404, 'Not Found')
      const details = handleError(error, { showToast: false })

      expect(details.type).toBe(ErrorType.NOT_FOUND)
      expect(details.statusCode).toBe(404)
    })

    it('should log errors in development mode', () => {
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
      const error = createAxiosError(500, 'Server Error')

      handleError(error, { showToast: false, logError: true, context: 'Test Operation' })

      expect(consoleSpy).toHaveBeenCalled()
      consoleSpy.mockRestore()
    })
  })

  describe('createErrorHandler', () => {
    it('should create a reusable error handler', () => {
      const handler = createErrorHandler('Creating order', { showToast: false })
      const error = createAxiosError(400, 'Bad Request')

      const details = handler(error)

      expect(details.type).toBe(ErrorType.VALIDATION)
    })
  })

  describe('Error type checkers', () => {
    describe('isAuthError', () => {
      it('should return true for 401 errors', () => {
        const error = createAxiosError(401, 'Unauthorized')
        expect(isAuthError(error)).toBe(true)
      })

      it('should return false for other errors', () => {
        const error = createAxiosError(403, 'Forbidden')
        expect(isAuthError(error)).toBe(false)
      })
    })

    describe('isPermissionError', () => {
      it('should return true for 403 errors', () => {
        const error = createAxiosError(403, 'Forbidden')
        expect(isPermissionError(error)).toBe(true)
      })

      it('should return false for other errors', () => {
        const error = createAxiosError(401, 'Unauthorized')
        expect(isPermissionError(error)).toBe(false)
      })
    })

    describe('isValidationError', () => {
      it('should return true for 400 errors', () => {
        const error = createAxiosError(400, 'Bad Request')
        expect(isValidationError(error)).toBe(true)
      })

      it('should return true for 422 errors', () => {
        const error = createAxiosError(422, 'Unprocessable Entity')
        expect(isValidationError(error)).toBe(true)
      })

      it('should return false for other errors', () => {
        const error = createAxiosError(500, 'Server Error')
        expect(isValidationError(error)).toBe(false)
      })
    })

    describe('isNetworkError', () => {
      it('should return true for network errors', () => {
        const error = createAxiosError(undefined, 'Network Error')
        expect(isNetworkError(error)).toBe(true)
      })

      it('should return false for HTTP errors', () => {
        const error = createAxiosError(500, 'Server Error')
        expect(isNetworkError(error)).toBe(false)
      })
    })

    describe('canRetryError', () => {
      it('should return true for network errors', () => {
        const error = createAxiosError(undefined, 'Network Error')
        expect(canRetryError(error)).toBe(true)
      })

      it('should return true for server errors', () => {
        const error = createAxiosError(500, 'Server Error')
        expect(canRetryError(error)).toBe(true)
      })

      it('should return true for rate limit errors', () => {
        const error = createAxiosError(429, 'Too Many Requests')
        expect(canRetryError(error)).toBe(true)
      })

      it('should return false for validation errors', () => {
        const error = createAxiosError(400, 'Bad Request')
        expect(canRetryError(error)).toBe(false)
      })

      it('should return false for auth errors', () => {
        const error = createAxiosError(401, 'Unauthorized')
        expect(canRetryError(error)).toBe(false)
      })
    })

    describe('isErrorType', () => {
      it('should correctly identify error types', () => {
        const serverError = createAxiosError(500, 'Server Error')
        expect(isErrorType(serverError, ErrorType.SERVER)).toBe(true)
        expect(isErrorType(serverError, ErrorType.NETWORK)).toBe(false)
      })
    })
  })
})
