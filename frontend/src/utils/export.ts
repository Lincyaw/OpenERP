/**
 * Export utilities for exporting data to CSV and Excel formats
 *
 * @example
 * ```tsx
 * import { exportToCSV, exportToExcel, downloadFile } from '@/utils/export'
 *
 * // Export to CSV
 * const csv = exportToCSV(data, {
 *   headers: ['Order Number', 'Customer', 'Amount', 'Status'],
 *   fields: ['order_number', 'customer_name', 'amount', 'status'],
 * })
 * downloadFile(csv, 'orders.csv', 'text/csv')
 *
 * // Export to Excel (requires xlsx library)
 * const excelBlob = await exportToExcel(data, {
 *   sheetName: 'Sales Orders',
 *   headers: ['Order Number', 'Customer', 'Amount', 'Status'],
 *   fields: ['order_number', 'customer_name', 'amount', 'status'],
 * })
 * downloadFile(excelBlob, 'orders.xlsx', 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet')
 * ```
 */

/**
 * Configuration for export operations
 */
export interface ExportConfig<T = Record<string, unknown>> {
  /** Column headers to display in the exported file */
  headers: string[]
  /** Field names (keys) in the data to extract */
  fields: (keyof T | string)[]
  /** Optional transform functions for specific fields */
  transforms?: Partial<Record<keyof T | string, (value: unknown, row: T) => string | number>>
  /** Sheet name for Excel export (defaults to 'Sheet1') */
  sheetName?: string
  /** Date format for date fields (defaults to 'YYYY-MM-DD') */
  dateFormat?: string
}

/**
 * Options for download file
 */
export interface DownloadOptions {
  /** Whether to include BOM for UTF-8 CSV (for Excel compatibility) */
  includeBOM?: boolean
}

/**
 * Escape a value for CSV format
 * - Wraps in quotes if contains comma, quote, or newline
 * - Escapes double quotes by doubling them
 */
function escapeCSVValue(value: unknown): string {
  if (value === null || value === undefined) {
    return ''
  }

  const stringValue = String(value)

  // Check if value needs escaping
  if (
    stringValue.includes(',') ||
    stringValue.includes('"') ||
    stringValue.includes('\n') ||
    stringValue.includes('\r')
  ) {
    // Escape double quotes by doubling them
    const escaped = stringValue.replace(/"/g, '""')
    return `"${escaped}"`
  }

  return stringValue
}

/**
 * Get value from nested object path (e.g., 'customer.name')
 */
function getNestedValue<T>(obj: T, path: string): unknown {
  return path.split('.').reduce((current: unknown, key) => {
    if (current && typeof current === 'object' && key in current) {
      return (current as Record<string, unknown>)[key]
    }
    return undefined
  }, obj as unknown)
}

/**
 * Export data to CSV format
 *
 * @param data - Array of data objects to export
 * @param config - Export configuration with headers and fields
 * @returns CSV string content
 */
export function exportToCSV<T extends Record<string, unknown>>(
  data: T[],
  config: ExportConfig<T>
): string {
  const { headers, fields, transforms } = config

  // Create header row
  const headerRow = headers.map(escapeCSVValue).join(',')

  // Create data rows
  const dataRows = data.map((row) => {
    return fields
      .map((field) => {
        const rawValue = getNestedValue(row, String(field))

        // Apply transform if exists
        if (transforms && field in transforms) {
          const transform = transforms[field as keyof typeof transforms]
          if (transform) {
            const transformedValue = transform(rawValue, row)
            return escapeCSVValue(transformedValue)
          }
        }

        return escapeCSVValue(rawValue)
      })
      .join(',')
  })

  return [headerRow, ...dataRows].join('\n')
}

/**
 * Export data to Excel format (.xlsx)
 *
 * This function dynamically imports the xlsx library to minimize bundle size.
 * The xlsx library must be installed: `npm install xlsx`
 *
 * @param data - Array of data objects to export
 * @param config - Export configuration with headers, fields, and sheet name
 * @returns Promise<Blob> - Excel file as a Blob
 */
export async function exportToExcel<T extends Record<string, unknown>>(
  data: T[],
  config: ExportConfig<T>
): Promise<Blob> {
  const { headers, fields, transforms, sheetName = 'Sheet1' } = config

  // Dynamically import xlsx library
  const XLSX = await import('xlsx')

  // Transform data to array of arrays (including header row)
  const rows: (string | number)[][] = []

  // Add header row
  rows.push(headers)

  // Add data rows
  data.forEach((row) => {
    const dataRow: (string | number)[] = fields.map((field) => {
      const rawValue = getNestedValue(row, String(field))

      // Apply transform if exists
      if (transforms && field in transforms) {
        const transform = transforms[field as keyof typeof transforms]
        if (transform) {
          return transform(rawValue, row)
        }
      }

      // Handle different types
      if (rawValue === null || rawValue === undefined) {
        return ''
      }
      if (typeof rawValue === 'number') {
        return rawValue
      }
      return String(rawValue)
    })
    rows.push(dataRow)
  })

  // Create workbook and worksheet
  const workbook = XLSX.utils.book_new()
  const worksheet = XLSX.utils.aoa_to_sheet(rows)

  // Auto-size columns based on content
  const MAX_EXCEL_COLUMN_WIDTH = 50 // Maximum column width in characters
  const colWidths = headers.map((_, colIndex) => {
    let maxWidth = headers[colIndex].length
    rows.forEach((row) => {
      const cellValue = String(row[colIndex] || '')
      maxWidth = Math.max(maxWidth, cellValue.length)
    })
    return { wch: Math.min(maxWidth + 2, MAX_EXCEL_COLUMN_WIDTH) }
  })
  worksheet['!cols'] = colWidths

  // Add worksheet to workbook
  XLSX.utils.book_append_sheet(workbook, worksheet, sheetName)

  // Generate Excel file as array buffer
  const excelBuffer = XLSX.write(workbook, {
    bookType: 'xlsx',
    type: 'array',
  })

  // Convert to Blob
  return new Blob([excelBuffer], {
    type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
  })
}

/**
 * Download file to user's computer
 *
 * @param content - File content (string for CSV, Blob for Excel)
 * @param filename - Name of the file to download
 * @param mimeType - MIME type of the file
 * @param options - Additional download options
 */
export function downloadFile(
  content: string | Blob,
  filename: string,
  mimeType: string,
  options: DownloadOptions = {}
): void {
  let blob: Blob

  if (typeof content === 'string') {
    // For CSV, optionally add BOM for Excel UTF-8 compatibility
    const { includeBOM = true } = options
    const bom = includeBOM && mimeType.includes('csv') ? '\uFEFF' : ''
    blob = new Blob([bom + content], { type: `${mimeType};charset=utf-8` })
  } else {
    blob = content
  }

  // Create download link
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename

  // Trigger download
  document.body.appendChild(link)
  link.click()

  // Cleanup
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

/**
 * Generate timestamped filename with sanitized input
 *
 * @param baseName - Base name for the file (without extension)
 * @param extension - File extension (e.g., 'csv', 'xlsx')
 * @returns Timestamped filename with sanitized characters
 */
export function generateExportFilename(baseName: string, extension: string): string {
  // Sanitize baseName - only allow alphanumeric, underscore, hyphen
  const sanitizedName = baseName.replace(/[^a-zA-Z0-9_-]/g, '_')
  // Sanitize extension - only allow alphanumeric
  const sanitizedExt = extension.replace(/[^a-zA-Z0-9]/g, '')

  const now = new Date()
  const timestamp = now
    .toISOString()
    .slice(0, 19)
    .replace(/[T:]/g, '-')
    .replace(/-/g, '')
    .slice(0, 14) // YYYYMMDDHHMMSS

  return `${sanitizedName}_${timestamp}.${sanitizedExt}`
}

/**
 * Format date value for export
 *
 * @param value - Date value (string, Date, or undefined)
 * @param format - Date format (default: 'YYYY-MM-DD HH:mm')
 * @returns Formatted date string
 */
export function formatDateForExport(
  value: string | Date | undefined | null,
  format: 'YYYY-MM-DD' | 'YYYY-MM-DD HH:mm' | 'YYYY-MM-DD HH:mm:ss' = 'YYYY-MM-DD HH:mm'
): string {
  if (!value) return ''

  const date = typeof value === 'string' ? new Date(value) : value

  if (isNaN(date.getTime())) return ''

  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')

  switch (format) {
    case 'YYYY-MM-DD':
      return `${year}-${month}-${day}`
    case 'YYYY-MM-DD HH:mm:ss':
      return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
    case 'YYYY-MM-DD HH:mm':
    default:
      return `${year}-${month}-${day} ${hours}:${minutes}`
  }
}

/**
 * Format number value for export
 *
 * @param value - Number value
 * @param decimals - Number of decimal places (default: 2)
 * @returns Formatted number or empty string
 */
export function formatNumberForExport(
  value: number | undefined | null,
  decimals: number = 2
): string | number {
  if (value === null || value === undefined) return ''
  return Number(value.toFixed(decimals))
}
