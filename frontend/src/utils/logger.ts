/**
 * Logger Service
 *
 * Centralized logging utility that can be configured for different environments.
 * In production, logging can be disabled or redirected to external services.
 *
 * Features:
 * - Production-configurable: logs are disabled in production by default
 * - Structured logging: consistent log format with timestamps and context
 * - Log levels: debug, info, warn, error
 * - Extensible: can be connected to external logging services (Sentry, DataDog, etc.)
 */

type LogLevel = 'debug' | 'info' | 'warn' | 'error'

interface LogEntry {
  level: LogLevel
  message: string
  context?: string
  data?: unknown
  timestamp: string
}

interface LoggerConfig {
  /**
   * Enable/disable logging
   * Default: true in development, false in production
   */
  enabled: boolean

  /**
   * Minimum log level to output
   * Default: 'debug' in development, 'error' in production
   */
  minLevel: LogLevel

  /**
   * Include timestamp in log output
   * Default: true
   */
  includeTimestamp: boolean

  /**
   * External logging handler (e.g., for Sentry, DataDog)
   * Called for all logs regardless of console output settings
   */
  externalHandler?: (entry: LogEntry) => void
}

// Log level priority (higher = more severe)
const LOG_LEVEL_PRIORITY: Record<LogLevel, number> = {
  debug: 0,
  info: 1,
  warn: 2,
  error: 3,
}

// Default configuration based on environment
const isDevelopment = import.meta.env.DEV
const defaultConfig: LoggerConfig = {
  enabled: isDevelopment,
  minLevel: isDevelopment ? 'debug' : 'error',
  includeTimestamp: true,
}

// Current configuration (mutable for runtime changes)
let currentConfig: LoggerConfig = { ...defaultConfig }

/**
 * Configure the logger
 *
 * @example
 * ```ts
 * // Enable logging in production for debugging
 * configureLogger({ enabled: true, minLevel: 'warn' })
 *
 * // Connect to external logging service
 * configureLogger({
 *   externalHandler: (entry) => {
 *     if (entry.level === 'error') {
 *       Sentry.captureException(entry.data)
 *     }
 *   }
 * })
 * ```
 */
export function configureLogger(config: Partial<LoggerConfig>): void {
  currentConfig = { ...currentConfig, ...config }
}

/**
 * Reset logger to default configuration
 */
export function resetLoggerConfig(): void {
  currentConfig = { ...defaultConfig }
}

/**
 * Get current logger configuration
 */
export function getLoggerConfig(): Readonly<LoggerConfig> {
  return { ...currentConfig }
}

/**
 * Check if a log level should be output based on current config
 */
function shouldLog(level: LogLevel): boolean {
  if (!currentConfig.enabled) return false
  return LOG_LEVEL_PRIORITY[level] >= LOG_LEVEL_PRIORITY[currentConfig.minLevel]
}

/**
 * Format log message with optional context and timestamp
 */
function formatMessage(level: LogLevel, message: string, context?: string): string {
  const parts: string[] = []

  if (currentConfig.includeTimestamp) {
    parts.push(`[${new Date().toISOString()}]`)
  }

  parts.push(`[${level.toUpperCase()}]`)

  if (context) {
    parts.push(`[${context}]`)
  }

  parts.push(message)

  return parts.join(' ')
}

/**
 * Create a log entry and handle output
 */
function log(level: LogLevel, message: string, context?: string, data?: unknown): void {
  const entry: LogEntry = {
    level,
    message,
    context,
    data,
    timestamp: new Date().toISOString(),
  }

  // Always call external handler if configured (for error tracking services)
  if (currentConfig.externalHandler) {
    try {
      currentConfig.externalHandler(entry)
    } catch {
      // Silently ignore errors in external handler to prevent logging loops
    }
  }

  // Output to console if enabled and level is high enough
  if (shouldLog(level)) {
    const formattedMessage = formatMessage(level, message, context)

    switch (level) {
      case 'debug':
        console.debug(formattedMessage, data !== undefined ? data : '')
        break
      case 'info':
        console.info(formattedMessage, data !== undefined ? data : '')
        break
      case 'warn':
        console.warn(formattedMessage, data !== undefined ? data : '')
        break
      case 'error':
        console.error(formattedMessage, data !== undefined ? data : '')
        break
    }
  }
}

/**
 * Logger object with methods for each log level
 *
 * @example
 * ```ts
 * // Basic usage
 * logger.error('Failed to fetch data', 'UserService', error)
 *
 * // Without context
 * logger.info('Application started')
 *
 * // With structured data
 * logger.debug('API response', 'ProductAPI', { status: 200, data: response })
 * ```
 */
export const logger = {
  /**
   * Log debug message (development only by default)
   */
  debug: (message: string, context?: string, data?: unknown): void => {
    log('debug', message, context, data)
  },

  /**
   * Log info message
   */
  info: (message: string, context?: string, data?: unknown): void => {
    log('info', message, context, data)
  },

  /**
   * Log warning message
   */
  warn: (message: string, context?: string, data?: unknown): void => {
    log('warn', message, context, data)
  },

  /**
   * Log error message
   */
  error: (message: string, context?: string, data?: unknown): void => {
    log('error', message, context, data)
  },
}

/**
 * Create a scoped logger with a fixed context
 *
 * @example
 * ```ts
 * const log = createScopedLogger('SalesOrderForm')
 *
 * log.error('Failed to fetch customers', error)
 * log.debug('Form state updated', { formData })
 * ```
 */
export function createScopedLogger(context: string) {
  return {
    debug: (message: string, data?: unknown) => logger.debug(message, context, data),
    info: (message: string, data?: unknown) => logger.info(message, context, data),
    warn: (message: string, data?: unknown) => logger.warn(message, context, data),
    error: (message: string, data?: unknown) => logger.error(message, context, data),
  }
}

export default logger
