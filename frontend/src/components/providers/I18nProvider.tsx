/**
 * I18nProvider Component
 *
 * Provides internationalization context to the application.
 * Integrates react-i18next with Semi Design's LocaleProvider.
 */

import { Suspense, useEffect, useMemo } from 'react'
import { I18nextProvider } from 'react-i18next'
import { LocaleProvider } from '@douyinfe/semi-ui-19'
import zh_CN from '@douyinfe/semi-ui-19/lib/es/locale/source/zh_CN'
import en_US from '@douyinfe/semi-ui-19/lib/es/locale/source/en_US'
import i18n from '@/i18n'
import type { SupportedLanguage } from '@/i18n/config'
import { useAppStore } from '@/store'

/**
 * Map of supported languages to Semi Design locale objects
 */
const semiLocales: Record<SupportedLanguage, typeof zh_CN> = {
  'zh-CN': zh_CN,
  'en-US': en_US,
}

/**
 * Loading fallback component
 */
function LoadingFallback() {
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        height: '100vh',
        width: '100vw',
      }}
    >
      <span>Loading...</span>
    </div>
  )
}

/**
 * Props for I18nProvider
 */
interface I18nProviderProps {
  children: React.ReactNode
}

/**
 * I18nProvider Component
 *
 * Wraps the application with:
 * 1. I18nextProvider for react-i18next translations
 * 2. Semi Design LocaleProvider for component translations
 * 3. Suspense for lazy loading translations
 *
 * @example
 * ```tsx
 * // In main.tsx
 * createRoot(document.getElementById('root')!).render(
 *   <StrictMode>
 *     <I18nProvider>
 *       <App />
 *     </I18nProvider>
 *   </StrictMode>
 * )
 * ```
 */
export function I18nProvider({ children }: I18nProviderProps) {
  const locale = useAppStore((state) => state.locale) as SupportedLanguage
  const setLocale = useAppStore((state) => state.setLocale)

  // Get Semi Design locale based on current language
  const semiLocale = useMemo(() => {
    return semiLocales[locale] || semiLocales['zh-CN']
  }, [locale])

  // Sync i18n language with app store locale
  useEffect(() => {
    const currentLang = i18n.language as SupportedLanguage

    // If store locale differs from i18n, update i18n
    if (currentLang !== locale) {
      i18n.changeLanguage(locale)
    }

    // Listen to i18n language changes and sync to store
    const handleLanguageChange = (lng: string) => {
      if (lng !== locale) {
        setLocale(lng)
      }
    }

    i18n.on('languageChanged', handleLanguageChange)

    return () => {
      i18n.off('languageChanged', handleLanguageChange)
    }
  }, [locale, setLocale])

  return (
    <I18nextProvider i18n={i18n}>
      <LocaleProvider locale={semiLocale}>
        <Suspense fallback={<LoadingFallback />}>{children}</Suspense>
      </LocaleProvider>
    </I18nextProvider>
  )
}

export default I18nProvider
