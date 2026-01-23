import { useEffect, useCallback } from 'react'
import { useAppStore } from '@/store'

/**
 * Theme type definitions
 */
export type Theme = 'light' | 'dark' | 'elder'
export type FontScale = 'default' | 'medium' | 'large' | 'xlarge'

/**
 * Theme configuration
 */
export interface ThemeConfig {
  theme: Theme
  fontScale: FontScale
}

/**
 * Apply theme to document
 */
export function applyTheme(theme: Theme): void {
  document.body.setAttribute('theme-mode', theme)

  // Sync with Semi Design
  if (theme === 'dark') {
    document.body.setAttribute('theme-mode', 'dark')
  } else {
    document.body.removeAttribute('theme-mode')
    if (theme === 'elder') {
      document.body.setAttribute('theme-mode', 'elder')
    }
  }
}

/**
 * Apply font scale to document
 */
export function applyFontScale(scale: FontScale): void {
  if (scale === 'default') {
    document.documentElement.removeAttribute('data-font-scale')
  } else {
    document.documentElement.setAttribute('data-font-scale', scale)
  }
}

/**
 * Get system color scheme preference
 */
export function getSystemTheme(): 'light' | 'dark' {
  if (typeof window === 'undefined') return 'light'
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

/**
 * Check if user prefers reduced motion
 */
export function prefersReducedMotion(): boolean {
  if (typeof window === 'undefined') return false
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

/**
 * Check if user prefers high contrast
 */
export function prefersHighContrast(): boolean {
  if (typeof window === 'undefined') return false
  return window.matchMedia('(prefers-contrast: high)').matches
}

/**
 * Hook to manage theme
 *
 * @example
 * function ThemeToggle() {
 *   const { theme, setTheme, toggleTheme } = useThemeManager()
 *
 *   return (
 *     <button onClick={toggleTheme}>
 *       Current theme: {theme}
 *     </button>
 *   )
 * }
 */
export function useThemeManager() {
  const { theme, setTheme, toggleTheme } = useAppStore()

  // Apply theme on mount and changes
  useEffect(() => {
    applyTheme(theme)
  }, [theme])

  // Listen to system theme changes
  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')

    const handleChange = (e: MediaQueryListEvent) => {
      // Only auto-switch if user hasn't explicitly set a preference
      const savedTheme = localStorage.getItem('erp-app-settings')
      if (!savedTheme) {
        setTheme(e.matches ? 'dark' : 'light')
      }
    }

    mediaQuery.addEventListener('change', handleChange)
    return () => mediaQuery.removeEventListener('change', handleChange)
  }, [setTheme])

  const setThemeWithPersist = useCallback(
    (newTheme: Theme) => {
      setTheme(newTheme === 'elder' ? 'light' : newTheme)
      applyTheme(newTheme)
    },
    [setTheme]
  )

  return {
    theme,
    setTheme: setThemeWithPersist,
    toggleTheme,
  }
}

/**
 * Hook to manage font scaling
 *
 * @example
 * function FontScaleSelector() {
 *   const { fontScale, setFontScale } = useFontScale()
 *
 *   return (
 *     <select value={fontScale} onChange={e => setFontScale(e.target.value)}>
 *       <option value="default">Default</option>
 *       <option value="medium">Medium</option>
 *       <option value="large">Large</option>
 *       <option value="xlarge">Extra Large</option>
 *     </select>
 *   )
 * }
 */
export function useFontScale() {
  // Get from localStorage or default
  const getFontScale = useCallback((): FontScale => {
    if (typeof window === 'undefined') return 'default'
    const saved = localStorage.getItem('erp-font-scale')
    return (saved as FontScale) || 'default'
  }, [])

  const setFontScale = useCallback((scale: FontScale) => {
    localStorage.setItem('erp-font-scale', scale)
    applyFontScale(scale)
  }, [])

  // Apply on mount
  useEffect(() => {
    applyFontScale(getFontScale())
  }, [getFontScale])

  return {
    fontScale: getFontScale(),
    setFontScale,
  }
}

/**
 * Hook to detect user accessibility preferences
 *
 * @example
 * function AccessibilityAwareComponent() {
 *   const { reducedMotion, highContrast } = useAccessibilityPreferences()
 *
 *   return (
 *     <div className={reducedMotion ? 'no-animation' : ''}>
 *       Content
 *     </div>
 *   )
 * }
 */
export function useAccessibilityPreferences() {
  const reducedMotion = prefersReducedMotion()
  const highContrast = prefersHighContrast()

  return {
    reducedMotion,
    highContrast,
  }
}
