import { renderHook } from '@testing-library/react'
import { useFormatters } from '../useFormatters'

describe('useFormatters', () => {
  describe('formatDateTime', () => {
    it('should display seconds by default', () => {
      const { result } = renderHook(() => useFormatters())
      const formatted = result.current.formatDateTime('2026-02-02T14:30:45Z')
      // Should contain seconds (format: HH:MM:SS)
      expect(formatted).toMatch(/\d{2}:\d{2}:\d{2}/)
    })

    it('should hide seconds when showSeconds is false', () => {
      const { result } = renderHook(() => useFormatters())
      const formatted = result.current.formatDateTime('2026-02-02T14:30:45Z', {
        showSeconds: false,
      })
      // Should contain time without seconds (format: HH:MM)
      // Should NOT contain seconds pattern
      expect(formatted).toMatch(/\d{2}:\d{2}/)
      expect(formatted).not.toMatch(/\d{2}:\d{2}:\d{2}/)
    })

    it('should handle invalid dates gracefully', () => {
      const { result } = renderHook(() => useFormatters())
      const formatted = result.current.formatDateTime('invalid-date')
      expect(formatted).toBe('')
    })
  })

  describe('formatTime', () => {
    it('should display seconds by default', () => {
      const { result } = renderHook(() => useFormatters())
      const formatted = result.current.formatTime('2026-02-02T14:30:45Z')
      // Should contain seconds (format: HH:MM:SS)
      expect(formatted).toMatch(/\d{2}:\d{2}:\d{2}/)
    })

    it('should hide seconds when showSeconds is false', () => {
      const { result } = renderHook(() => useFormatters())
      const formatted = result.current.formatTime('2026-02-02T14:30:45Z', {
        showSeconds: false,
      })
      // Should contain time without seconds (format: HH:MM)
      // Should NOT contain seconds pattern
      expect(formatted).toMatch(/\d{2}:\d{2}/)
      expect(formatted).not.toMatch(/\d{2}:\d{2}:\d{2}/)
    })
  })

  describe('formatDate', () => {
    it('should format date without time', () => {
      const { result } = renderHook(() => useFormatters())
      const formatted = result.current.formatDate('2026-02-02T14:30:45Z', 'medium')
      // Should not contain time pattern (HH:MM)
      expect(formatted).not.toMatch(/\d{2}:\d{2}/)
    })
  })
})
