/**
 * useI18n Hook
 *
 * Custom hook that wraps react-i18next's useTranslation
 * with additional type safety and convenience features.
 */

import { useTranslation } from 'react-i18next'
import { useCallback } from 'react'
import type { Namespace, SupportedLanguage } from '@/i18n/config'
import { SUPPORTED_LANGUAGES, LANGUAGE_NAMES, LANGUAGE_FLAGS } from '@/i18n/config'
import { useAppStore } from '@/store'

/**
 * Options for useI18n hook
 */
interface UseI18nOptions {
  /**
   * Namespace(s) to load translations from
   * @default 'common'
   */
  ns?: Namespace | Namespace[]
}

/**
 * Custom i18n hook with type-safe translation support
 *
 * @example
 * ```tsx
 * function MyComponent() {
 *   const { t, language, changeLanguage } = useI18n({ ns: 'common' })
 *
 *   return (
 *     <div>
 *       <p>{t('messages.loading')}</p>
 *       <button onClick={() => changeLanguage('en-US')}>
 *         Switch to English
 *       </button>
 *     </div>
 *   )
 * }
 * ```
 */
export function useI18n(options: UseI18nOptions = {}) {
  const { ns = 'common' } = options
  const { t, i18n, ready } = useTranslation(ns)
  const setLocale = useAppStore((state) => state.setLocale)

  /**
   * Current language
   */
  const language = i18n.language as SupportedLanguage

  /**
   * Change language
   */
  const changeLanguage = useCallback(
    async (lng: SupportedLanguage) => {
      await i18n.changeLanguage(lng)
      setLocale(lng)
    },
    [i18n, setLocale]
  )

  /**
   * Check if a language is supported
   */
  const isLanguageSupported = useCallback((lng: string): lng is SupportedLanguage => {
    return SUPPORTED_LANGUAGES.includes(lng as SupportedLanguage)
  }, [])

  /**
   * Get display name for a language
   */
  const getLanguageName = useCallback((lng: SupportedLanguage): string => {
    return LANGUAGE_NAMES[lng] || lng
  }, [])

  /**
   * Get flag/icon for a language
   */
  const getLanguageFlag = useCallback((lng: SupportedLanguage): string => {
    return LANGUAGE_FLAGS[lng] || ''
  }, [])

  /**
   * List of available languages with metadata
   */
  const languages = SUPPORTED_LANGUAGES.map((lng) => ({
    code: lng,
    name: LANGUAGE_NAMES[lng],
    flag: LANGUAGE_FLAGS[lng],
    isCurrent: lng === language,
  }))

  return {
    /**
     * Translation function
     */
    t,

    /**
     * i18next instance
     */
    i18n,

    /**
     * Whether translations are loaded and ready
     */
    ready,

    /**
     * Current language code
     */
    language,

    /**
     * Change the current language
     */
    changeLanguage,

    /**
     * Check if a language is supported
     */
    isLanguageSupported,

    /**
     * Get display name for a language
     */
    getLanguageName,

    /**
     * Get flag/icon for a language
     */
    getLanguageFlag,

    /**
     * List of available languages
     */
    languages,

    /**
     * Supported language codes
     */
    supportedLanguages: SUPPORTED_LANGUAGES,
  }
}

export default useI18n
