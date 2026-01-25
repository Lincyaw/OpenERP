/**
 * Formatting Hooks
 *
 * Custom hooks for formatting dates, numbers, and currencies
 * based on the current locale settings.
 */

import { useMemo, useCallback } from 'react'
import { useI18n } from './useI18n'

/**
 * Date format options
 */
export type DateFormatStyle = 'short' | 'medium' | 'long' | 'full'

/**
 * Number format options
 */
export interface NumberFormatOptions {
  style?: 'decimal' | 'currency' | 'percent'
  currency?: string
  minimumFractionDigits?: number
  maximumFractionDigits?: number
  useGrouping?: boolean
}

/**
 * Hook for formatting dates according to the current locale
 *
 * @example
 * ```tsx
 * function DateDisplay() {
 *   const { formatDate, formatTime, formatDateTime } = useDateFormatter()
 *
 *   return (
 *     <div>
 *       <p>Date: {formatDate(new Date())}</p>
 *       <p>Time: {formatTime(new Date())}</p>
 *       <p>DateTime: {formatDateTime(new Date())}</p>
 *     </div>
 *   )
 * }
 * ```
 */
export function useDateFormatter() {
  const { language } = useI18n()

  /**
   * Get date format options based on style
   */
  const getDateOptions = useCallback((style: DateFormatStyle): Intl.DateTimeFormatOptions => {
    const options: Record<DateFormatStyle, Intl.DateTimeFormatOptions> = {
      short: { year: '2-digit', month: 'numeric', day: 'numeric' },
      medium: { year: 'numeric', month: 'short', day: 'numeric' },
      long: { year: 'numeric', month: 'long', day: 'numeric' },
      full: { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' },
    }
    return options[style]
  }, [])

  /**
   * Format date only
   */
  const formatDate = useCallback(
    (date: Date | string | number, style: DateFormatStyle = 'medium'): string => {
      const d = date instanceof Date ? date : new Date(date)
      if (isNaN(d.getTime())) return ''
      return new Intl.DateTimeFormat(language, getDateOptions(style)).format(d)
    },
    [language, getDateOptions]
  )

  /**
   * Format time only
   */
  const formatTime = useCallback(
    (
      date: Date | string | number,
      options: { hour12?: boolean; showSeconds?: boolean } = {}
    ): string => {
      const d = date instanceof Date ? date : new Date(date)
      if (isNaN(d.getTime())) return ''

      const { hour12 = language === 'en-US', showSeconds = false } = options

      return new Intl.DateTimeFormat(language, {
        hour: '2-digit',
        minute: '2-digit',
        second: showSeconds ? '2-digit' : undefined,
        hour12,
      }).format(d)
    },
    [language]
  )

  /**
   * Format date and time
   */
  const formatDateTime = useCallback(
    (
      date: Date | string | number,
      options: { dateStyle?: DateFormatStyle; showSeconds?: boolean } = {}
    ): string => {
      const d = date instanceof Date ? date : new Date(date)
      if (isNaN(d.getTime())) return ''

      const { dateStyle = 'medium', showSeconds = false } = options
      const dateStr = formatDate(d, dateStyle)
      const timeStr = formatTime(d, { showSeconds })

      return `${dateStr} ${timeStr}`
    },
    [formatDate, formatTime]
  )

  /**
   * Format relative time (e.g., "2 hours ago")
   */
  const formatRelative = useCallback(
    (date: Date | string | number): string => {
      const d = date instanceof Date ? date : new Date(date)
      if (isNaN(d.getTime())) return ''

      const now = new Date()
      const diff = now.getTime() - d.getTime()
      const seconds = Math.floor(diff / 1000)
      const minutes = Math.floor(seconds / 60)
      const hours = Math.floor(minutes / 60)
      const days = Math.floor(hours / 24)

      const rtf = new Intl.RelativeTimeFormat(language, { numeric: 'auto' })

      if (days > 0) return rtf.format(-days, 'day')
      if (hours > 0) return rtf.format(-hours, 'hour')
      if (minutes > 0) return rtf.format(-minutes, 'minute')
      return rtf.format(-seconds, 'second')
    },
    [language]
  )

  return {
    formatDate,
    formatTime,
    formatDateTime,
    formatRelative,
  }
}

/**
 * Hook for formatting numbers according to the current locale
 *
 * @example
 * ```tsx
 * function PriceDisplay() {
 *   const { formatNumber, formatCurrency, formatPercent } = useNumberFormatter()
 *
 *   return (
 *     <div>
 *       <p>Number: {formatNumber(12345.67)}</p>
 *       <p>Price: {formatCurrency(99.99)}</p>
 *       <p>Rate: {formatPercent(0.156)}</p>
 *     </div>
 *   )
 * }
 * ```
 */
export function useNumberFormatter() {
  const { language } = useI18n()

  /**
   * Get default currency based on locale
   */
  const defaultCurrency = useMemo(() => {
    return language === 'zh-CN' ? 'CNY' : 'USD'
  }, [language])

  /**
   * Format a number with locale-specific formatting
   */
  const formatNumber = useCallback(
    (value: number, options: NumberFormatOptions = {}): string => {
      if (typeof value !== 'number' || isNaN(value)) return ''

      const {
        style = 'decimal',
        currency = defaultCurrency,
        minimumFractionDigits,
        maximumFractionDigits,
        useGrouping = true,
      } = options

      return new Intl.NumberFormat(language, {
        style,
        currency: style === 'currency' ? currency : undefined,
        minimumFractionDigits,
        maximumFractionDigits,
        useGrouping,
      }).format(value)
    },
    [language, defaultCurrency]
  )

  /**
   * Format as currency
   */
  const formatCurrency = useCallback(
    (value: number, currency?: string): string => {
      return formatNumber(value, {
        style: 'currency',
        currency: currency || defaultCurrency,
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
      })
    },
    [formatNumber, defaultCurrency]
  )

  /**
   * Format as percentage
   */
  const formatPercent = useCallback(
    (value: number, decimals: number = 2): string => {
      return formatNumber(value, {
        style: 'percent',
        minimumFractionDigits: decimals,
        maximumFractionDigits: decimals,
      })
    },
    [formatNumber]
  )

  /**
   * Format as compact number (e.g., 1.2K, 3.4M)
   */
  const formatCompact = useCallback(
    (value: number): string => {
      if (typeof value !== 'number' || isNaN(value)) return ''

      return new Intl.NumberFormat(language, {
        notation: 'compact',
        compactDisplay: 'short',
      }).format(value)
    },
    [language]
  )

  /**
   * Format as integer (no decimals)
   */
  const formatInteger = useCallback(
    (value: number): string => {
      return formatNumber(Math.round(value), {
        maximumFractionDigits: 0,
      })
    },
    [formatNumber]
  )

  return {
    formatNumber,
    formatCurrency,
    formatPercent,
    formatCompact,
    formatInteger,
    defaultCurrency,
  }
}

/**
 * Combined formatters hook for convenience
 *
 * @example
 * ```tsx
 * function DataDisplay() {
 *   const { formatDate, formatCurrency } = useFormatters()
 *
 *   return (
 *     <div>
 *       <p>Date: {formatDate(order.createdAt)}</p>
 *       <p>Total: {formatCurrency(order.total)}</p>
 *     </div>
 *   )
 * }
 * ```
 */
export function useFormatters() {
  const dateFormatter = useDateFormatter()
  const numberFormatter = useNumberFormatter()

  return {
    ...dateFormatter,
    ...numberFormatter,
  }
}

export default useFormatters
