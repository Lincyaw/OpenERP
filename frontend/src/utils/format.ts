/**
 * Safe Formatting Utilities
 *
 * These functions handle type coercion and validation to prevent
 * runtime errors when formatting values that may come from API
 * responses as strings instead of numbers.
 */

/**
 * Safely converts a value to a number
 * Handles: string, number, null, undefined
 */
export function toNumber(value: unknown): number {
  if (value === null || value === undefined) {
    return 0
  }
  if (typeof value === 'number') {
    return isNaN(value) ? 0 : value
  }
  if (typeof value === 'string') {
    const parsed = parseFloat(value)
    return isNaN(parsed) ? 0 : parsed
  }
  return 0
}

/**
 * Safely format a number with fixed decimal places
 * Handles: string, number, null, undefined
 *
 * @param value - Value to format (can be string or number)
 * @param decimals - Number of decimal places (default: 2)
 * @param fallback - Value to return if conversion fails (default: '0.00')
 * @returns Formatted string with fixed decimals
 *
 * @example
 * ```ts
 * safeToFixed(123.456)        // '123.46'
 * safeToFixed('123.456')      // '123.46'
 * safeToFixed(null)           // '0.00'
 * safeToFixed(undefined)      // '0.00'
 * safeToFixed('invalid')      // '0.00'
 * safeToFixed(123.456, 3)     // '123.456'
 * safeToFixed(null, 2, '-')   // '-'
 * ```
 */
export function safeToFixed(value: unknown, decimals: number = 2, fallback?: string): string {
  if (value === null || value === undefined) {
    return fallback ?? (0).toFixed(decimals)
  }

  const num = toNumber(value)
  if (num === 0 && fallback !== undefined) {
    // Check if original value was actually 0 or just couldn't be converted
    const originalWasZero = value === 0 || value === '0' || value === '0.00' || value === '0.0'
    if (!originalWasZero) {
      return fallback
    }
  }

  return num.toFixed(decimals)
}

/**
 * Safely format a currency value
 * Handles: string, number, null, undefined
 *
 * @param value - Value to format (can be string or number)
 * @param prefix - Currency prefix (default: '¥')
 * @param decimals - Number of decimal places (default: 2)
 * @param fallback - Value to return if conversion fails (default: '-')
 * @returns Formatted currency string
 *
 * @example
 * ```ts
 * safeFormatCurrency(123.456)           // '¥123.46'
 * safeFormatCurrency('123.456')         // '¥123.46'
 * safeFormatCurrency(null)              // '-'
 * safeFormatCurrency(undefined)         // '-'
 * safeFormatCurrency(99.99, '$')        // '$99.99'
 * safeFormatCurrency(0)                 // '¥0.00'
 * ```
 */
export function safeFormatCurrency(
  value: unknown,
  prefix: string = '¥',
  decimals: number = 2,
  fallback: string = '-'
): string {
  if (value === null || value === undefined) {
    return fallback
  }

  const num = toNumber(value)
  return `${prefix}${num.toFixed(decimals)}`
}

/**
 * Safely format a quantity value
 * Handles: string, number, null, undefined
 *
 * @param value - Value to format (can be string or number)
 * @param decimals - Number of decimal places (default: 2)
 * @param fallback - Value to return if conversion fails (default: '-')
 * @returns Formatted quantity string
 *
 * @example
 * ```ts
 * safeFormatQuantity(123.456)        // '123.46'
 * safeFormatQuantity('123.456')      // '123.46'
 * safeFormatQuantity(null)           // '-'
 * safeFormatQuantity(undefined)      // '-'
 * ```
 */
export function safeFormatQuantity(
  value: unknown,
  decimals: number = 2,
  fallback: string = '-'
): string {
  if (value === null || value === undefined) {
    return fallback
  }

  const num = toNumber(value)
  return num.toFixed(decimals)
}

/**
 * Safely format a signed quantity (with + or - prefix)
 * Handles: string, number, null, undefined
 *
 * @param value - Value to format (can be string or number)
 * @param decimals - Number of decimal places (default: 2)
 * @param fallback - Value to return if conversion fails (default: '-')
 * @returns Formatted quantity string with sign prefix
 *
 * @example
 * ```ts
 * safeFormatSignedQuantity(123.456)   // '+123.46'
 * safeFormatSignedQuantity(-123.456)  // '-123.46'
 * safeFormatSignedQuantity(0)         // '0.00'
 * safeFormatSignedQuantity(null)      // '-'
 * ```
 */
export function safeFormatSignedQuantity(
  value: unknown,
  decimals: number = 2,
  fallback: string = '-'
): string {
  if (value === null || value === undefined) {
    return fallback
  }

  const num = toNumber(value)
  const formatted = num.toFixed(decimals)
  if (num > 0) {
    return `+${formatted}`
  }
  return formatted
}

/**
 * Safely format a percentage value
 * Handles: string, number, null, undefined
 *
 * @param value - Value to format (can be string or number, 0.1 = 10%)
 * @param decimals - Number of decimal places (default: 2)
 * @param fallback - Value to return if conversion fails (default: '-')
 * @returns Formatted percentage string
 *
 * @example
 * ```ts
 * safeFormatPercent(0.156)        // '15.60%'
 * safeFormatPercent('0.156')      // '15.60%'
 * safeFormatPercent(null)         // '-'
 * ```
 */
export function safeFormatPercent(
  value: unknown,
  decimals: number = 2,
  fallback: string = '-'
): string {
  if (value === null || value === undefined) {
    return fallback
  }

  const num = toNumber(value)
  return `${(num * 100).toFixed(decimals)}%`
}
