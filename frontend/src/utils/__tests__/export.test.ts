import { formatDateForExport } from '../export'

describe('formatDateForExport', () => {
  // Use a Date object to avoid timezone conversion issues in tests
  // Create date in local timezone: 2026-02-02 14:30:45
  const testDate = new Date(2026, 1, 2, 14, 30, 45) // Month is 0-indexed

  it('should format with seconds by default', () => {
    const result = formatDateForExport(testDate)
    expect(result).toBe('2026-02-02 14:30:45')
  })

  it('should format date only when specified', () => {
    const result = formatDateForExport(testDate, 'YYYY-MM-DD')
    expect(result).toBe('2026-02-02')
  })

  it('should format with minutes when specified', () => {
    const result = formatDateForExport(testDate, 'YYYY-MM-DD HH:mm')
    expect(result).toBe('2026-02-02 14:30')
  })

  it('should format with seconds when specified', () => {
    const result = formatDateForExport(testDate, 'YYYY-MM-DD HH:mm:ss')
    expect(result).toBe('2026-02-02 14:30:45')
  })

  it('should handle null/undefined values', () => {
    expect(formatDateForExport(null)).toBe('')
    expect(formatDateForExport(undefined)).toBe('')
  })

  it('should handle invalid dates', () => {
    expect(formatDateForExport('invalid')).toBe('')
  })
})
