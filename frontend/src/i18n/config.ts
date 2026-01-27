/**
 * i18n Configuration
 *
 * This file defines the i18n namespaces and supported languages.
 * Each namespace maps to a separate JSON translation file.
 */

/**
 * Supported languages
 */
export const SUPPORTED_LANGUAGES = ['zh-CN', 'en-US'] as const
export type SupportedLanguage = (typeof SUPPORTED_LANGUAGES)[number]

/**
 * Default language
 */
export const DEFAULT_LANGUAGE: SupportedLanguage = 'zh-CN'

/**
 * Fallback language when translation is missing
 */
export const FALLBACK_LANGUAGE: SupportedLanguage = 'zh-CN'

/**
 * Translation namespaces
 *
 * Each namespace corresponds to a JSON file in the locales directory.
 * Example: 'common' -> locales/zh-CN/common.json
 */
export const NAMESPACES = [
  'common', // Common UI strings (actions, status, labels, messages)
  'validation', // Form validation messages
  'auth', // Authentication related strings
  'catalog', // Product and category management
  'partner', // Customer, supplier, warehouse management
  'trade', // Sales and purchase orders
  'inventory', // Stock management
  'finance', // Financial management
  'system', // System settings, users, roles
  'integration', // E-commerce platform integration
  'admin', // Admin management (feature flags, etc.)
] as const

export type Namespace = (typeof NAMESPACES)[number]

/**
 * Default namespace used when none is specified
 */
export const DEFAULT_NAMESPACE: Namespace = 'common'

/**
 * Language display names for UI
 */
export const LANGUAGE_NAMES: Record<SupportedLanguage, string> = {
  'zh-CN': '简体中文',
  'en-US': 'English',
}

/**
 * Language flags/icons for UI (emoji or icon names)
 */
export const LANGUAGE_FLAGS: Record<SupportedLanguage, string> = {
  'zh-CN': '🇨🇳',
  'en-US': '🇺🇸',
}
