// Custom hooks
// Reusable React hooks for the application

// Theme and accessibility hooks
export {
  useThemeManager,
  useFontScale,
  useAccessibilityPreferences,
  applyTheme,
  applyFontScale,
  getSystemTheme,
  prefersReducedMotion,
  prefersHighContrast,
  type Theme,
  type FontScale,
  type ThemeConfig,
} from './useTheme'

// Internationalization hooks
export { useI18n } from './useI18n'
export {
  useFormatters,
  useDateFormatter,
  useNumberFormatter,
  type DateFormatStyle,
  type NumberFormatOptions,
} from './useFormatters'
