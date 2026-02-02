/**
 * usePrintServiceStatus Hook
 *
 * Custom hook for monitoring print service status including:
 * - Health check with periodic polling (every 10 seconds)
 * - Printer list discovery
 * - Connection state management
 * - Status change notifications via Toast
 *
 * @example
 * const { status, version, printers, error, refresh } = usePrintServiceStatus()
 *
 * if (status === 'connected') {
 *   console.log(`Print service v${version} is running`)
 *   console.log(`Found ${printers.length} printers`)
 * }
 */

import { useCallback, useEffect, useRef, useState } from 'react'
import { Toast } from '@douyinfe/semi-ui-19'

/** Print service connection status */
export type PrintServiceStatus = 'connected' | 'disconnected' | 'checking'

/** Printer information from the print service */
export interface PrinterInfo {
  /** Unique printer identifier */
  name: string
  /** Display name of the printer */
  displayName?: string
  /** Whether this is the default printer */
  isDefault?: boolean
  /** Printer status (online, offline, etc.) */
  status?: string
}

/** Health check response from the print service */
export interface PrintServiceHealthResponse {
  /** Service status */
  status: 'ok' | 'error'
  /** Service version (e.g., "1.0.0") */
  version?: string
  /** Optional error message */
  message?: string
}

/** Printer list response from the print service */
export interface PrintersResponse {
  /** List of available printers */
  printers: PrinterInfo[]
}

/** Configuration options for the hook */
export interface UsePrintServiceStatusOptions {
  /** Enable automatic polling (default: true) */
  enabled?: boolean
  /** Polling interval in milliseconds (default: 10000ms = 10 seconds) */
  pollingInterval?: number
  /** Print service base URL (default: http://localhost:9999) */
  baseUrl?: string
  /** Request timeout in milliseconds (default: 5000ms) */
  timeout?: number
  /** Show Toast notifications on status change (default: true) */
  showNotifications?: boolean
}

/** Return value of usePrintServiceStatus hook */
export interface UsePrintServiceStatusReturn {
  /** Current connection status */
  status: PrintServiceStatus
  /** Print service version (if connected) */
  version: string | null
  /** List of available printers */
  printers: PrinterInfo[]
  /** Whether printers are being loaded */
  isLoadingPrinters: boolean
  /** Last error message */
  error: string | null
  /** Timestamp of last successful health check */
  lastChecked: Date | null
  /** Manually trigger a health check */
  checkHealth: () => Promise<void>
  /** Manually refresh the printer list */
  refreshPrinters: () => Promise<void>
  /** Refresh both health and printers */
  refresh: () => Promise<void>
}

/** Default configuration values */
const DEFAULT_OPTIONS: Required<UsePrintServiceStatusOptions> = {
  enabled: true,
  pollingInterval: 10000, // 10 seconds
  baseUrl: 'http://localhost:9999',
  timeout: 5000, // 5 seconds
  showNotifications: true,
}

/**
 * Error type enumeration for better error handling
 */
type PrintServiceErrorType = 'CORS' | 'NETWORK' | 'TIMEOUT' | 'UNKNOWN'

/**
 * Classify the error type for user-friendly messaging
 */
function classifyError(error: unknown): PrintServiceErrorType {
  if (error instanceof TypeError && error.message.includes('Failed to fetch')) {
    return 'CORS'
  }

  if (error instanceof DOMException && error.name === 'AbortError') {
    return 'TIMEOUT'
  }

  if (error instanceof Error) {
    const message = error.message.toLowerCase()
    if (message.includes('network') || message.includes('connection')) {
      return 'NETWORK'
    }
    if (message.includes('cors') || message.includes('cross-origin')) {
      return 'CORS'
    }
    if (message.includes('timeout') || message.includes('abort')) {
      return 'TIMEOUT'
    }
  }

  return 'UNKNOWN'
}

/**
 * Get user-friendly error message based on error type
 */
function getErrorMessage(errorType: PrintServiceErrorType): string {
  switch (errorType) {
    case 'CORS':
      return '打印服务连接被拒绝，请检查服务是否允许跨域访问'
    case 'NETWORK':
      return '网络连接失败，请检查打印服务是否正在运行'
    case 'TIMEOUT':
      return '连接打印服务超时，请检查服务响应'
    case 'UNKNOWN':
    default:
      return '无法连接到打印服务'
  }
}

/**
 * Hook for monitoring print service status
 */
export function usePrintServiceStatus(
  options: UsePrintServiceStatusOptions = {}
): UsePrintServiceStatusReturn {
  const opts = { ...DEFAULT_OPTIONS, ...options }

  const [status, setStatus] = useState<PrintServiceStatus>('checking')
  const [version, setVersion] = useState<string | null>(null)
  const [printers, setPrinters] = useState<PrinterInfo[]>([])
  const [isLoadingPrinters, setIsLoadingPrinters] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [lastChecked, setLastChecked] = useState<Date | null>(null)

  // Use refs for mutable values to avoid dependency cycles in callbacks
  const statusRef = useRef<PrintServiceStatus>('checking')
  const isFirstCheckRef = useRef(true)
  const isMountedRef = useRef(true)

  // Keep refs for options to avoid recreating callbacks
  const baseUrlRef = useRef(opts.baseUrl)
  const timeoutRef = useRef(opts.timeout)
  const showNotificationsRef = useRef(opts.showNotifications)

  baseUrlRef.current = opts.baseUrl
  timeoutRef.current = opts.timeout
  showNotificationsRef.current = opts.showNotifications

  /**
   * Perform health check against the print service
   */
  const checkHealth = useCallback(async (): Promise<void> => {
    if (!isMountedRef.current) return

    const previousStatus = statusRef.current

    try {
      const controller = new AbortController()
      const timeoutId = setTimeout(() => controller.abort(), timeoutRef.current)

      const response = await fetch(`${baseUrlRef.current}/health`, {
        method: 'GET',
        signal: controller.signal,
        headers: {
          Accept: 'application/json',
        },
      })

      clearTimeout(timeoutId)

      if (!isMountedRef.current) return

      if (response.ok) {
        const data: PrintServiceHealthResponse = await response.json()

        if (data.status === 'ok') {
          statusRef.current = 'connected'
          setStatus('connected')
          setVersion(data.version ?? null)
          setError(null)
          setLastChecked(new Date())

          // Show notification on status change (not on first check)
          if (
            showNotificationsRef.current &&
            !isFirstCheckRef.current &&
            previousStatus === 'disconnected'
          ) {
            Toast.success('打印服务已连接')
          }
        } else {
          throw new Error(data.message || '打印服务状态异常')
        }
      } else {
        throw new Error(`HTTP ${response.status}`)
      }
    } catch (err) {
      if (!isMountedRef.current) return

      const errorType = classifyError(err)
      const errorMessage = getErrorMessage(errorType)

      statusRef.current = 'disconnected'
      setStatus('disconnected')
      setVersion(null)
      setError(errorMessage)

      // Show notification on status change (not on first check)
      if (
        showNotificationsRef.current &&
        !isFirstCheckRef.current &&
        previousStatus === 'connected'
      ) {
        Toast.warning('打印服务连接已断开')
      }
    } finally {
      if (isMountedRef.current) {
        isFirstCheckRef.current = false
      }
    }
  }, [])

  /**
   * Fetch the list of available printers
   */
  const refreshPrinters = useCallback(async (): Promise<void> => {
    if (!isMountedRef.current) return

    setIsLoadingPrinters(true)

    try {
      const controller = new AbortController()
      const timeoutId = setTimeout(() => controller.abort(), timeoutRef.current)

      const response = await fetch(`${baseUrlRef.current}/printers`, {
        method: 'GET',
        signal: controller.signal,
        headers: {
          Accept: 'application/json',
        },
      })

      clearTimeout(timeoutId)

      if (!isMountedRef.current) return

      if (response.ok) {
        const data: PrintersResponse = await response.json()
        setPrinters(data.printers || [])
      } else {
        throw new Error(`HTTP ${response.status}`)
      }
    } catch (err) {
      if (!isMountedRef.current) return

      const errorType = classifyError(err)
      const errorMessage = getErrorMessage(errorType)

      // Don't overwrite connection error with printer error
      if (statusRef.current === 'connected') {
        setError(`获取打印机列表失败: ${errorMessage}`)
      }
      setPrinters([])
    } finally {
      if (isMountedRef.current) {
        setIsLoadingPrinters(false)
      }
    }
  }, [])

  /**
   * Refresh both health status and printer list
   */
  const refresh = useCallback(async (): Promise<void> => {
    await checkHealth()
    // Only fetch printers if health check succeeded
    if (isMountedRef.current && statusRef.current === 'connected') {
      await refreshPrinters()
    }
  }, [checkHealth, refreshPrinters])

  // Initial health check and start polling
  useEffect(() => {
    if (!opts.enabled) {
      return
    }

    isMountedRef.current = true
    isFirstCheckRef.current = true
    statusRef.current = 'checking'

    // Perform initial health check
    checkHealth()

    // Start polling interval
    const intervalId = setInterval(() => {
      checkHealth()
    }, opts.pollingInterval)

    // Cleanup on unmount
    return () => {
      isMountedRef.current = false
      clearInterval(intervalId)
    }
  }, [opts.enabled, opts.pollingInterval, checkHealth])

  // Fetch printers when connected
  useEffect(() => {
    if (status === 'connected' && opts.enabled) {
      refreshPrinters()
    }
  }, [status, opts.enabled, refreshPrinters])

  return {
    status,
    version,
    printers,
    isLoadingPrinters,
    error,
    lastChecked,
    checkHealth,
    refreshPrinters,
    refresh,
  }
}

/**
 * Export default polling interval for testing
 */
export const DEFAULT_POLLING_INTERVAL = DEFAULT_OPTIONS.pollingInterval

/**
 * Export default base URL for testing
 */
export const DEFAULT_BASE_URL = DEFAULT_OPTIONS.baseUrl
